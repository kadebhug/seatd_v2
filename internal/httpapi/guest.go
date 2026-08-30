package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kadebhug/seatd_v2/internal/domain/operations"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const guestBodyLimit = 32 << 10

type guestActionDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type guestContextDTO struct {
	LocationName  string           `json:"locationName"`
	TableLabel    string           `json:"tableLabel"`
	Occupancy     occupancyDTO     `json:"occupancy"`
	Actions       []guestActionDTO `json:"actions"`
	ActiveRequest *assistDTO       `json:"activeRequest,omitempty"`
}

type guestCreateRequest struct {
	CommandID uuid.UUID `json:"commandId"`
	ActionKey string    `json:"actionKey"`
}

type guestCancelRequest struct {
	CommandID uuid.UUID `json:"commandId"`
}

type tableQRCapabilityDTO struct {
	ID           string  `json:"id"`
	TableID      string  `json:"tableId"`
	LookupPrefix string  `json:"lookupPrefix"`
	Label        *string `json:"label,omitempty"`
	IssuedAt     string  `json:"issuedAt"`
	ExpiresAt    *string `json:"expiresAt,omitempty"`
	RevokedAt    *string `json:"revokedAt,omitempty"`
	LastUsedAt   *string `json:"lastUsedAt,omitempty"`
	Version      int32   `json:"version"`
}

type qrExportDTO struct {
	tableQRCapabilityDTO
	Token     string `json:"token"`
	PublicURL string `json:"publicUrl"`
}

type exportQRRequest struct {
	Label     string `json:"label"`
	ExpiresAt string `json:"expiresAt"`
}

func (api *API) getGuestQR(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	ctx, err := api.ops.GuestContext(r.Context(), token)
	if err != nil {
		api.recordGuestAbuse(r, token, "", guestAbuseScope{}, "lookup_denied")
		api.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, guestContextFromDomain(ctx))
}

func (api *API) createGuestRequest(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	req, body, ok := readGuestJSON[guestCreateRequest](w, r)
	if !ok {
		return
	}
	actionKey := strings.TrimSpace(req.ActionKey)
	if req.CommandID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "commandId is required", nil)
		return
	}
	if actionKey == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "actionKey is required", nil)
		return
	}
	if !api.guestRL.allow(guestRateKey(r, token, actionKey)) {
		api.recordGuestAbuse(r, token, actionKey, guestAbuseScope{}, "rate_limited")
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many guest requests", nil)
		return
	}
	sum := sha256.Sum256(body)
	result, replay, err := api.ops.CreateGuestRequest(r.Context(), operations.GuestRequestParams{
		Token:          token,
		ActionKey:      actionKey,
		IdempotencyKey: req.CommandID,
		PayloadHash:    sum[:],
	}, func(assist db.AssistRequest) (int32, []byte, error) {
		return marshalStatus(http.StatusCreated, assistFromDB(assist))
	})
	if err != nil {
		if errors.Is(err, operations.ErrRateLimited) {
			api.recordGuestAbuse(r, token, actionKey, guestAbuseScope{}, "cooldown")
		}
		api.writeDomainError(w, err)
		return
	}
	api.writeCommandResult(w, result, replay, nil)
}

func (api *API) getGuestRequest(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	assistID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	assist, err := api.ops.GetGuestRequest(r.Context(), token, assistID)
	if err != nil {
		api.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, assistFromDB(assist))
}

func (api *API) cancelGuestRequest(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	assistID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	req, body, ok := readGuestJSON[guestCancelRequest](w, r)
	if !ok {
		return
	}
	if req.CommandID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "commandId is required", nil)
		return
	}
	sum := sha256.Sum256(body)
	result, replay, err := api.ops.CancelGuestRequest(r.Context(), operations.GuestCancelParams{
		Token:          token,
		AssistID:       assistID,
		IdempotencyKey: req.CommandID,
		PayloadHash:    sum[:],
	}, func(assist db.AssistRequest) (int32, []byte, error) {
		return marshalStatus(http.StatusOK, assistFromDB(assist))
	})
	api.writeCommandResult(w, result, replay, err)
}

func (api *API) listTableQRCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	tableID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var capabilities []db.TableQrCapability
	err := api.inTenantTx(r.Context(), ctx, func(q *db.Queries) error {
		var err error
		capabilities, err = q.ListTableQRCapabilitiesByTable(r.Context(), db.ListTableQRCapabilitiesByTableParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
			TableID:        tableID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]tableQRCapabilityDTO, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, tableQRCapabilityFromDB(capability))
	}
	writeJSON(w, http.StatusOK, map[string]any{"qrCapabilities": out})
}

func (api *API) exportTableQRCapability(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	tableID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req exportQRRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	expiresAt, ok := parseOptionalTime(w, req.ExpiresAt, "expiresAt")
	if !ok {
		return
	}
	export, err := api.ops.ExportTableQRCapability(r.Context(), ctx.OrganisationID, ctx.LocationID, tableID, req.Label, expiresAt)
	if err != nil {
		api.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, qrExportFromDomain(api.cfg.GuestWebOrigin, export))
}

func (api *API) rotateTableQRCapability(w http.ResponseWriter, r *http.Request) {
	ctx, tableID, capabilityID, ok := api.qrCapabilityInput(w, r)
	if !ok {
		return
	}
	export, err := api.ops.RotateTableQRCapability(r.Context(), ctx.OrganisationID, ctx.LocationID, tableID, capabilityID)
	if err != nil {
		api.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, qrExportFromDomain(api.cfg.GuestWebOrigin, export))
}

func (api *API) revokeTableQRCapability(w http.ResponseWriter, r *http.Request) {
	ctx, tableID, capabilityID, ok := api.qrCapabilityInput(w, r)
	if !ok {
		return
	}
	capability, err := api.ops.RevokeTableQRCapability(r.Context(), ctx.OrganisationID, ctx.LocationID, tableID, capabilityID)
	if err != nil {
		api.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tableQRCapabilityFromDB(capability))
}

func (api *API) qrCapabilityInput(w http.ResponseWriter, r *http.Request) (requestContext, uuid.UUID, uuid.UUID, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, uuid.Nil, uuid.Nil, false
	}
	tableID, ok := pathUUID(w, r, "id")
	if !ok {
		return requestContext{}, uuid.Nil, uuid.Nil, false
	}
	capabilityID, err := uuid.Parse(r.PathValue("capabilityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "capabilityId must be a UUID", nil)
		return requestContext{}, uuid.Nil, uuid.Nil, false
	}
	return ctx, tableID, capabilityID, true
}

func readGuestJSON[T any](w http.ResponseWriter, r *http.Request) (T, []byte, bool) {
	var zero T
	body := http.MaxBytesReader(w, r.Body, guestBodyLimit)
	defer body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(body); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body is invalid", nil)
		return zero, nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	decoder.DisallowUnknownFields()
	var req T
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be valid JSON", nil)
		return zero, nil, false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must contain one JSON object", nil)
		return zero, nil, false
	}
	canonical, err := canonicalJSON(buf.Bytes())
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be a JSON object", nil)
		return zero, nil, false
	}
	return req, canonical, true
}

func parseOptionalTime(w http.ResponseWriter, value string, field string) (pgtype.Timestamptz, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Timestamptz{}, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", field+" must be an RFC3339 timestamp", nil)
		return pgtype.Timestamptz{}, false
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, true
}

func guestContextFromDomain(ctx operations.GuestContext) guestContextDTO {
	actions := make([]guestActionDTO, 0, len(ctx.Actions))
	for _, action := range ctx.Actions {
		actions = append(actions, guestActionDTO{Key: action.Key, Label: action.Label})
	}
	var active *assistDTO
	if ctx.ActiveRequest != nil {
		dto := assistFromDB(*ctx.ActiveRequest)
		active = &dto
	}
	return guestContextDTO{
		LocationName:  ctx.LocationName,
		TableLabel:    ctx.TableLabel,
		Occupancy:     occupancyFromDB(ctx.Occupancy),
		Actions:       actions,
		ActiveRequest: active,
	}
}

func tableQRCapabilityFromDB(capability db.TableQrCapability) tableQRCapabilityDTO {
	return tableQRCapabilityDTO{
		ID:           capability.ID.String(),
		TableID:      capability.TableID.String(),
		LookupPrefix: capability.TokenLookupPrefix,
		Label:        textPtr(capability.Label),
		IssuedAt:     timeString(capability.IssuedAt),
		ExpiresAt:    timePtr(capability.ExpiresAt),
		RevokedAt:    timePtr(capability.RevokedAt),
		LastUsedAt:   timePtr(capability.LastUsedAt),
		Version:      capability.Version,
	}
}

func qrExportFromDomain(guestOrigin string, export operations.QRExport) qrExportDTO {
	return qrExportDTO{
		tableQRCapabilityDTO: tableQRCapabilityDTO{
			ID:           export.ID.String(),
			TableID:      export.TableID.String(),
			LookupPrefix: export.LookupPrefix,
			Label:        stringPtr(export.Label),
			IssuedAt:     timeString(export.IssuedAt),
			ExpiresAt:    timePtr(export.ExpiresAt),
			RevokedAt:    timePtr(export.RevokedAt),
			Version:      export.Version,
		},
		Token:     export.Token,
		PublicURL: guestPublicURL(guestOrigin, export.Token),
	}
}

func guestPublicURL(origin string, token string) string {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return "/qr/" + url.PathEscape(token)
	}
	return origin + "/qr/" + url.PathEscape(token)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

type guestAbuseScope struct {
	organisationID uuid.UUID
	locationID     uuid.UUID
	tableID        uuid.UUID
}

func (api *API) recordGuestAbuse(r *http.Request, token string, actionKey string, scope guestAbuseScope, decision string) {
	if api.pool == nil {
		return
	}
	prefix := tokenLookupPrefix(token)
	ipHash := hashClientIP(clientIP(r))
	err := api.queries.RecordGuestAbuseEvent(r.Context(), db.RecordGuestAbuseEventParams{
		OrganisationID:    nullUUID(scope.organisationID),
		LocationID:        nullUUID(scope.locationID),
		TableID:           nullUUID(scope.tableID),
		TokenLookupPrefix: prefix,
		ActionKey:         nullableGuestText(actionKey),
		IpHash:            ipHash,
		Decision:          decision,
	})
	if err != nil && api.logger != nil {
		api.logger.Warn("recording guest abuse event failed", "error", err)
	}
}

func tokenLookupPrefix(token string) string {
	prefix, _, err := operations.TokenLookupParts(token)
	if err != nil {
		sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
		return "invalid:" + hex.EncodeToString(sum[:8])
	}
	return prefix
}

func hashClientIP(ip string) []byte {
	sum := sha256.Sum256([]byte(ip))
	return sum[:]
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func nullUUID(value uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: value, Valid: value != uuid.Nil}
}

func nullableGuestText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	return pgtype.Text{String: value, Valid: value != ""}
}

func (api *API) withGuestCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/guest/") {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			allowedOrigin := strings.TrimSpace(api.cfg.GuestWebOrigin)
			if allowedOrigin == "" {
				allowedOrigin = "*"
			}
			if allowedOrigin == "*" || origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type guestRateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	hits   map[string]guestRateBucket
}

type guestRateBucket struct {
	reset time.Time
	count int
}

func newGuestRateLimiter(window time.Duration, limit int) *guestRateLimiter {
	return &guestRateLimiter{
		window: window,
		limit:  limit,
		hits:   map[string]guestRateBucket{},
	}
}

func (l *guestRateLimiter) allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.hits[key]
	if now.After(bucket.reset) {
		bucket = guestRateBucket{reset: now.Add(l.window)}
	}
	bucket.count++
	l.hits[key] = bucket
	return bucket.count <= l.limit
}

func guestRateKey(r *http.Request, token string, actionKey string) string {
	return fmt.Sprintf("%s:%s:%s", clientIP(r), tokenLookupPrefix(token), actionKey)
}
