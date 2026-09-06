package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"go.opentelemetry.io/otel/attribute"
)

const (
	headerOrganisationID = "X-Seatd-Organisation-ID"
	headerLocationID     = "X-Seatd-Location-ID"
	headerActorRef       = "X-Seatd-Actor-Ref"
	headerDeviceID       = "X-Seatd-Device-ID"
)

type Handler struct {
	cfg    app.Config
	logger *slog.Logger
	pool   *pgxpool.Pool
	hub    *Hub
	ws     WebSocketConfig
}

type WebSocketConfig struct {
	SendBuffer      int
	ReadLimit       int64
	Heartbeat       time.Duration
	ConnectionLimit int
}

type requestContext struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
	DeviceID       uuid.UUID
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

func NewHandler(cfg app.Config, logger *slog.Logger, pool *pgxpool.Pool, hub *Hub, ws WebSocketConfig) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if hub == nil {
		hub = NewHub(logger)
	}
	if ws.SendBuffer <= 0 {
		ws.SendBuffer = 64
	}
	if ws.ReadLimit <= 0 {
		ws.ReadLimit = 4096
	}
	if ws.Heartbeat <= 0 {
		ws.Heartbeat = 25 * time.Second
	}
	if ws.ConnectionLimit <= 0 {
		ws.ConnectionLimit = 5000
	}

	handler := &Handler{cfg: cfg, logger: logger, pool: pool, hub: hub, ws: ws}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/realtime", handler.serveWebSocket)
	mux.HandleFunc("GET /v1/sync/location-snapshot", handler.getLocationSnapshot)
	mux.HandleFunc("GET /v1/sync/events", handler.listEvents)
	mux.HandleFunc("GET /v1/realtime/metrics", handler.getMetrics)
	return mux
}

func (h *Handler) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "realtime.websocket")
	defer span.End()
	r = r.WithContext(ctx)
	req, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}
	span.SetAttributes(
		attribute.String("seatd.organisation_id", req.OrganisationID.String()),
		attribute.String("seatd.location_id", req.LocationID.String()),
		attribute.String("seatd.device_id", req.DeviceID.String()),
	)
	if h.hub.MetricsSnapshot().ActiveConnections >= h.ws.ConnectionLimit {
		writeError(w, http.StatusServiceUnavailable, "connection_limit", "realtime connection limit reached", nil)
		return
	}
	conn, err := Upgrade(w, r, h.ws.ReadLimit)
	if err != nil {
		recordSpanError(span, err)
		h.logger.WarnContext(r.Context(), "websocket upgrade failed", "error", err)
		return
	}

	client := h.hub.Subscribe(Scope{OrganisationID: req.OrganisationID, LocationID: req.LocationID}, req.ActorRef, req.DeviceID, h.ws.SendBuffer)
	h.logger.InfoContext(r.Context(), "realtime websocket connected",
		"organisation_id", req.OrganisationID,
		"location_id", req.LocationID,
		"actor_ref", req.ActorRef,
		"device_id", req.DeviceID,
	)
	defer h.hub.Unsubscribe(client)
	defer conn.Close()

	errc := make(chan error, 2)
	go func() {
		errc <- conn.ReadLoop(r.Context())
	}()
	go func() {
		errc <- conn.WriteLoop(r.Context(), client.send, h.ws.Heartbeat)
	}()

	select {
	case <-client.Done():
	case <-r.Context().Done():
	case err := <-errc:
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			h.logger.WarnContext(r.Context(), "realtime websocket closed with error", "error", err)
		}
	}
}

func (h *Handler) getLocationSnapshot(w http.ResponseWriter, r *http.Request) {
	req, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}
	var snapshot locationSnapshot
	err := h.inTenantTx(r.Context(), req, func(tx pgx.Tx) error {
		cursor, err := loadCurrentCursor(r.Context(), tx, req)
		if err != nil {
			return err
		}
		tables, err := loadTableStates(r.Context(), tx, req)
		if err != nil {
			return err
		}
		assists, err := loadActiveAssists(r.Context(), tx, req)
		if err != nil {
			return err
		}
		snapshot = locationSnapshot{
			OrganisationID: req.OrganisationID.String(),
			LocationID:     req.LocationID.String(),
			Cursor:         cursor.Encode(),
			Tables:         tables,
			Assists:        assists,
		}
		return nil
	})
	if err != nil {
		h.writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	req, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}
	limit := int32(200)
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			writeError(w, http.StatusBadRequest, "validation_failed", "limit must be between 1 and 500", nil)
			return
		}
		limit = int32(parsed)
	}
	after, err := DecodeCursor(strings.TrimSpace(r.URL.Query().Get("after")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "after cursor is invalid", nil)
		return
	}

	var response eventsResponse
	err = h.inTenantTx(r.Context(), req, func(tx pgx.Tx) error {
		events, cursor, err := loadEventsAfter(r.Context(), tx, req, after, limit)
		if err != nil {
			return err
		}
		response = eventsResponse{Events: events, Cursor: cursor.Encode()}
		return nil
	})
	if err != nil {
		h.writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.hub.MetricsSnapshot())
}

func (h *Handler) authorizeRequest(w http.ResponseWriter, r *http.Request) (requestContext, bool) {
	req, ok := parseRequestContext(w, r)
	if !ok {
		return requestContext{}, false
	}
	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "database is unavailable", nil)
		return requestContext{}, false
	}
	if err := h.authorize(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, errUnauthorized):
			writeError(w, http.StatusUnauthorized, "unauthorized", "realtime authorization failed", nil)
		case errors.Is(err, errForbidden):
			writeError(w, http.StatusForbidden, "forbidden", "realtime access is not allowed", nil)
		default:
			h.writeDBError(w, err)
		}
		return requestContext{}, false
	}
	return req, true
}

var (
	errUnauthorized = errors.New("unauthorized")
	errForbidden    = errors.New("forbidden")
)

func (h *Handler) authorize(ctx context.Context, req requestContext) error {
	ctx, span := startSpan(ctx, "realtime.authorize",
		attribute.String("seatd.organisation_id", req.OrganisationID.String()),
		attribute.String("seatd.location_id", req.LocationID.String()),
		attribute.String("seatd.device_id", req.DeviceID.String()),
	)
	defer span.End()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		recordSpanError(span, err)
		return fmt.Errorf("beginning authorization transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		recordSpanError(span, err)
		return rollback(tx, ctx, fmt.Errorf("setting authorization database context: %w", err))
	}
	var deviceOK bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM devices
    WHERE id = $1
      AND organisation_id = $2
      AND (location_id IS NULL OR location_id = $3)
      AND trust_state = 'trusted'
      AND revoked_at IS NULL
)`, req.DeviceID, req.OrganisationID, req.LocationID).Scan(&deviceOK)
	if err != nil {
		recordSpanError(span, err)
		return rollback(tx, ctx, fmt.Errorf("checking realtime device authorization: %w", err))
	}
	if !deviceOK {
		recordSpanError(span, errUnauthorized)
		return rollback(tx, ctx, errUnauthorized)
	}

	var permissionOK bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM location_memberships lm
    JOIN role_permissions rp ON rp.role_name = lm.role
    WHERE lm.organisation_id = $1
      AND lm.location_id = $2
      AND lm.member_ref = $3
      AND lm.disabled_at IS NULL
      AND rp.permission_name = 'operations.read'
    UNION ALL
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.organisation_id = $1
      AND om.member_ref = $3
      AND om.disabled_at IS NULL
      AND rp.permission_name = 'operations.read'
)`, req.OrganisationID, req.LocationID, req.ActorRef).Scan(&permissionOK)
	if err != nil {
		recordSpanError(span, err)
		return rollback(tx, ctx, fmt.Errorf("checking realtime staff authorization: %w", err))
	}
	if !permissionOK {
		recordSpanError(span, errForbidden)
		return rollback(tx, ctx, errForbidden)
	}
	if _, err := tx.Exec(ctx, `
UPDATE devices
SET last_seen_at = now(),
    updated_at = now()
WHERE id = $1 AND organisation_id = $2
`, req.DeviceID, req.OrganisationID); err != nil {
		recordSpanError(span, err)
		return rollback(tx, ctx, fmt.Errorf("marking realtime device seen: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		recordSpanError(span, err)
		return fmt.Errorf("committing authorization transaction: %w", err)
	}
	return nil
}

func (h *Handler) inTenantTx(ctx context.Context, req requestContext, fn func(pgx.Tx) error) error {
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, req.OrganisationID.String(), req.LocationID.String()); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(tx); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func parseRequestContext(w http.ResponseWriter, r *http.Request) (requestContext, bool) {
	orgID, ok := parseHeaderUUID(w, r, headerOrganisationID)
	if !ok {
		return requestContext{}, false
	}
	locationID, ok := parseHeaderUUID(w, r, headerLocationID)
	if !ok {
		return requestContext{}, false
	}
	deviceID, ok := parseHeaderUUID(w, r, headerDeviceID)
	if !ok {
		return requestContext{}, false
	}
	actorRef := strings.TrimSpace(r.Header.Get(headerActorRef))
	if actorRef == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", headerActorRef+" is required", nil)
		return requestContext{}, false
	}
	return requestContext{OrganisationID: orgID, LocationID: locationID, DeviceID: deviceID, ActorRef: actorRef}, true
}

func parseHeaderUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", name+" is required", nil)
		return uuid.Nil, false
	}
	id, err := uuid.Parse(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", name+" must be a UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message, Details: details}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) writeDBError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
		return
	}
	h.logger.Error("realtime database request failed", "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}

type locationSnapshot struct {
	OrganisationID string          `json:"organisationId"`
	LocationID     string          `json:"locationId"`
	Cursor         string          `json:"cursor"`
	Tables         []tableStateDTO `json:"tables"`
	Assists        []assistDTO     `json:"assists"`
}

type eventsResponse struct {
	Events []Message `json:"events"`
	Cursor string    `json:"cursor"`
}

type Cursor struct {
	OccurredAt time.Time `json:"occurredAt"`
	EventID    uuid.UUID `json:"eventId"`
}

func (c Cursor) Encode() string {
	if c.OccurredAt.IsZero() || c.EventID == uuid.Nil {
		return ""
	}
	body, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(body)
}

func DecodeCursor(value string) (Cursor, error) {
	if value == "" {
		return Cursor{}, nil
	}
	body, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, err
	}
	var cursor Cursor
	if err := json.Unmarshal(body, &cursor); err != nil {
		return Cursor{}, err
	}
	if cursor.OccurredAt.IsZero() || cursor.EventID == uuid.Nil {
		return Cursor{}, errors.New("cursor is incomplete")
	}
	return cursor, nil
}

func cursorFrom(occurredAt time.Time, eventID uuid.UUID) Cursor {
	if eventID == uuid.Nil {
		return Cursor{}
	}
	return Cursor{OccurredAt: occurredAt.UTC(), EventID: eventID}
}

func timeString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339Nano)
}
