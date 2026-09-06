package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"github.com/kadebhug/seatd_v2/internal/domain/analytics"
	"github.com/kadebhug/seatd_v2/internal/domain/configuration"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/domain/integrations"
	"github.com/kadebhug/seatd_v2/internal/domain/operations"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	headerOrganisationID = "X-Seatd-Organisation-ID"
	headerLocationID     = "X-Seatd-Location-ID"
	headerActorRef       = "X-Seatd-Actor-Ref"
	headerDeviceID       = "X-Seatd-Device-ID"
	headerInternalSecret = "X-Seatd-Internal-Secret"
)

var errForbidden = errors.New("forbidden")

type tenantAuthzScope int

const (
	tenantAuthzOrganisation tenantAuthzScope = iota
	tenantAuthzLocation
)

type API struct {
	cfg       app.Config
	logger    *slog.Logger
	pool      *pgxpool.Pool
	queries   *db.Queries
	config    *configuration.Service
	ident     *identity.Service
	ops       *operations.Service
	analytics *analytics.Service
	ints      *integrations.Service
	guestRL   *guestRateLimiter
	metrics   *observability.ServiceMetrics
}

func NewHandler(cfg app.Config, logger *slog.Logger, pool *pgxpool.Pool, metrics ...*observability.ServiceMetrics) http.Handler {
	var serviceMetrics *observability.ServiceMetrics
	if len(metrics) > 0 {
		serviceMetrics = metrics[0]
	}
	opsService := operations.NewService(pool)
	api := &API{
		cfg:       cfg,
		logger:    logger,
		pool:      pool,
		queries:   db.New(pool),
		config:    configuration.NewService(pool),
		ident:     identity.NewService(pool),
		ops:       opsService,
		analytics: analytics.NewService(pool),
		ints:      integrations.NewService(pool, opsService),
		guestRL:   newGuestRateLimiter(time.Minute, 6),
		metrics:   serviceMetrics,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/guest/qr/{token}", api.getGuestQR)
	mux.HandleFunc("POST /v1/guest/qr/{token}/requests", api.createGuestRequest)
	mux.HandleFunc("GET /v1/guest/qr/{token}/requests/{id}", api.getGuestRequest)
	mux.HandleFunc("POST /v1/guest/qr/{token}/requests/{id}/cancel", api.cancelGuestRequest)
	mux.HandleFunc("POST /v1/auth/oidc/session", api.createOIDCWebSession)
	mux.HandleFunc("GET /v1/auth/session", api.getWebSession)
	mux.HandleFunc("POST /v1/auth/session/rotate", api.rotateWebSession)
	mux.HandleFunc("POST /v1/auth/session/logout", api.logoutWebSession)
	mux.HandleFunc("GET /v1/platform/tenants", api.searchPlatformTenants)
	mux.HandleFunc("GET /v1/platform/tenants/{id}", api.getPlatformTenant)
	mux.HandleFunc("GET /v1/platform/tenants/{id}/diagnostics", api.getPlatformTenantDiagnostics)
	mux.HandleFunc("GET /v1/organisations/{id}", api.getOrganisation)
	mux.HandleFunc("PUT /v1/organisations/{id}", api.updateOrganisation)
	mux.HandleFunc("GET /v1/owner/snapshot", api.getOwnerSnapshot)
	mux.HandleFunc("GET /v1/locations/{id}", api.getLocation)
	mux.HandleFunc("PUT /v1/locations/{id}", api.updateLocation)
	mux.HandleFunc("GET /v1/floors", api.listFloors)
	mux.HandleFunc("POST /v1/floors", api.createFloor)
	mux.HandleFunc("PUT /v1/floors/{id}", api.updateFloor)
	mux.HandleFunc("POST /v1/floors/{id}/archive", api.archiveFloor)
	mux.HandleFunc("POST /v1/floors/{id}/restore", api.restoreFloor)
	mux.HandleFunc("GET /v1/floors/{id}/zones", api.listZones)
	mux.HandleFunc("GET /v1/floors/{id}/tables", api.listTables)
	mux.HandleFunc("GET /v1/layout/editor-snapshot", api.getLayoutEditorSnapshot)
	mux.HandleFunc("POST /v1/zones", api.createZone)
	mux.HandleFunc("PUT /v1/zones/{id}", api.updateZone)
	mux.HandleFunc("POST /v1/zones/{id}/archive", api.archiveZone)
	mux.HandleFunc("GET /v1/tables/{id}", api.getTable)
	mux.HandleFunc("POST /v1/tables", api.createTable)
	mux.HandleFunc("GET /v1/tables/{id}/qr-capabilities", api.listTableQRCapabilities)
	mux.HandleFunc("POST /v1/tables/{id}/qr-capabilities", api.exportTableQRCapability)
	mux.HandleFunc("POST /v1/tables/{id}/qr-capabilities/{capabilityId}/rotate", api.rotateTableQRCapability)
	mux.HandleFunc("POST /v1/tables/{id}/qr-capabilities/{capabilityId}/revoke", api.revokeTableQRCapability)
	mux.HandleFunc("PUT /v1/tables/{id}", api.updateTable)
	mux.HandleFunc("POST /v1/tables/{id}/archive", api.archiveTable)
	mux.HandleFunc("POST /v1/tables/{id}/restore", api.restoreTable)
	mux.HandleFunc("GET /v1/location-state", api.getLocationState)
	mux.HandleFunc("GET /v1/assists", api.listAssists)
	mux.HandleFunc("GET /v1/audit/timeline", api.getAuditTimeline)
	mux.HandleFunc("GET /v1/analytics/summary", api.getAnalyticsSummary)
	mux.HandleFunc("GET /v1/analytics/timeseries", api.getAnalyticsTimeseries)
	mux.HandleFunc("GET /v1/analytics/comparison", api.getAnalyticsComparison)
	mux.HandleFunc("GET /v1/analytics/data-quality", api.getAnalyticsDataQuality)
	mux.HandleFunc("POST /v1/analytics/rebuild", api.rebuildAnalytics)
	mux.HandleFunc("GET /v1/service-periods", api.listServicePeriods)
	mux.HandleFunc("POST /v1/service-periods", api.createServicePeriod)
	mux.HandleFunc("PUT /v1/service-periods/{id}", api.updateServicePeriod)
	mux.HandleFunc("POST /v1/service-periods/{id}/archive", api.archiveServicePeriod)
	mux.HandleFunc("GET /v1/devices", api.listDevices)
	mux.HandleFunc("POST /v1/devices/pairing-codes", api.createDevicePairingCode)
	mux.HandleFunc("POST /v1/devices/pair", api.pairDevice)
	mux.HandleFunc("POST /v1/devices/heartbeat", api.deviceHeartbeat)
	mux.HandleFunc("GET /v1/devices/display-snapshot", api.getDisplaySnapshot)
	mux.HandleFunc("POST /v1/devices/{id}/revoke", api.revokeDevice)
	mux.HandleFunc("GET /v1/integrations", api.listIntegrations)
	mux.HandleFunc("GET /v1/integrations/{id}/health", api.getIntegrationHealth)
	mux.HandleFunc("GET /v1/integrations/{id}/mappings", api.listIntegrationMappings)
	mux.HandleFunc("PUT /v1/integrations/{id}/mappings/{mappingId}", api.updateIntegrationMapping)
	mux.HandleFunc("GET /v1/integrations/{id}/webhooks", api.listIntegrationWebhooks)
	mux.HandleFunc("POST /v1/integrations/{id}/webhooks/{webhookId}/replay", api.replayIntegrationWebhook)
	mux.HandleFunc("GET /v1/integrations/{id}/discrepancies", api.listIntegrationDiscrepancies)
	mux.HandleFunc("POST /v1/integrations/{id}/reconcile", api.reconcileIntegration)
	mux.HandleFunc("POST /v1/integrations/{vendor}/webhooks", api.receiveIntegrationWebhook)
	mux.HandleFunc("GET /v1/memberships", api.listMemberships)
	mux.HandleFunc("POST /v1/tables/{id}/occupy", api.occupyTable)
	mux.HandleFunc("POST /v1/tables/{id}/clear", api.clearTable)
	mux.HandleFunc("POST /v1/assists/{id}/acknowledge", api.acknowledgeAssist)
	mux.HandleFunc("POST /v1/assists/{id}/resolve", api.resolveAssist)
	mux.HandleFunc("POST /v1/assists/{id}/cancel", api.cancelAssist)
	return api.withGuestCORS(mux)
}

type requestContext struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
	DeviceID       uuid.NullUUID
	UserProfileID  uuid.NullUUID
	PrincipalType  principalType
	Roles          []string
	Permissions    []string
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type commandRequest struct {
	CommandID       uuid.UUID `json:"commandId"`
	ExpectedVersion int32     `json:"expectedVersion"`
	PartySize       *int32    `json:"partySize,omitempty"`
}

type organisationDTO struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type locationDTO struct {
	ID              string          `json:"id"`
	OrganisationID  string          `json:"organisationId"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Timezone        string          `json:"timezone"`
	Status          string          `json:"status"`
	OperatingConfig json.RawMessage `json:"operatingConfig"`
	FeatureFlags    json.RawMessage `json:"featureFlags"`
}

type floorDTO struct {
	ID                 string          `json:"id"`
	Slug               string          `json:"slug"`
	Name               string          `json:"name"`
	SortOrder          int32           `json:"sortOrder"`
	Canvas             json.RawMessage `json:"canvas"`
	BackgroundAssetRef *string         `json:"backgroundAssetRef,omitempty"`
	IsActive           bool            `json:"isActive"`
	Version            int32           `json:"version"`
}

type zoneDTO struct {
	ID        string `json:"id"`
	FloorID   string `json:"floorId"`
	Name      string `json:"name"`
	SortOrder int32  `json:"sortOrder"`
	IsActive  bool   `json:"isActive"`
	Version   int32  `json:"version"`
}

type tableDTO struct {
	ID            string          `json:"id"`
	FloorID       string          `json:"floorId"`
	ZoneID        string          `json:"zoneId"`
	Label         string          `json:"label"`
	CapacityLabel string          `json:"capacityLabel"`
	Shape         string          `json:"shape"`
	Geometry      json.RawMessage `json:"geometry"`
	IsActive      bool            `json:"isActive"`
	Version       int32           `json:"version"`
}

type occupancyDTO struct {
	Status           string  `json:"status"`
	CurrentSessionID *string `json:"currentSessionId,omitempty"`
	Version          int32   `json:"version"`
	UpdatedAt        string  `json:"updatedAt"`
}

type tableStateDTO struct {
	Table     tableDTO     `json:"table"`
	Occupancy occupancyDTO `json:"occupancy"`
}

type tableSessionDTO struct {
	ID        string  `json:"id"`
	TableID   string  `json:"tableId"`
	StartedAt string  `json:"startedAt"`
	EndedAt   *string `json:"endedAt,omitempty"`
	PartySize *int32  `json:"partySize,omitempty"`
	Source    string  `json:"source"`
}

type tableCommandResponse struct {
	Table     tableDTO        `json:"table"`
	Occupancy occupancyDTO    `json:"occupancy"`
	Session   tableSessionDTO `json:"session"`
}

type assistDTO struct {
	ID             string  `json:"id"`
	TableID        string  `json:"tableId"`
	TableSessionID *string `json:"tableSessionId,omitempty"`
	Status         string  `json:"status"`
	RequestedAt    string  `json:"requestedAt"`
	Version        int32   `json:"version"`
	Note           *string `json:"note,omitempty"`
	ActionKey      *string `json:"actionKey,omitempty"`
}

type deviceDTO struct {
	ID                       string          `json:"id"`
	OrganisationID           string          `json:"organisationId"`
	LocationID               *string         `json:"locationId,omitempty"`
	Name                     *string         `json:"name,omitempty"`
	DeviceType               string          `json:"deviceType"`
	Platform                 string          `json:"platform"`
	AppVersion               string          `json:"appVersion"`
	TrustState               string          `json:"trustState"`
	RegisteredAt             string          `json:"registeredAt"`
	LastSeenAt               *string         `json:"lastSeenAt,omitempty"`
	LastHeartbeatAt          *string         `json:"lastHeartbeatAt,omitempty"`
	AssignedAt               *string         `json:"assignedAt,omitempty"`
	RevokedAt                *string         `json:"revokedAt,omitempty"`
	HeartbeatIntervalSeconds int32           `json:"heartbeatIntervalSeconds"`
	Capabilities             json.RawMessage `json:"capabilities"`
	Configuration            json.RawMessage `json:"configuration"`
}

type membershipDTO struct {
	ID             string  `json:"id"`
	Scope          string  `json:"scope"`
	OrganisationID string  `json:"organisationId"`
	LocationID     *string `json:"locationId,omitempty"`
	MemberRef      string  `json:"memberRef"`
	Role           string  `json:"role"`
}

type timelineEventDTO struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	SchemaVersion int32           `json:"schemaVersion"`
	OccurredAt    string          `json:"occurredAt"`
	ActorRef      string          `json:"actorRef"`
	DeviceID      *string         `json:"deviceId,omitempty"`
	EntityType    string          `json:"entityType"`
	EntityID      string          `json:"entityId"`
	EntityVersion *int32          `json:"entityVersion,omitempty"`
	CommandID     *string         `json:"commandId,omitempty"`
	Data          json.RawMessage `json:"data"`
}

type ownerSnapshotDTO struct {
	Organisation organisationDTO `json:"organisation"`
	Locations    []locationDTO   `json:"locations"`
}

type analyticsMetricDTO struct {
	ActiveTableCount           int32   `json:"activeTableCount"`
	OccupancySeconds           float64 `json:"occupancySeconds"`
	UtilisationBasisSeconds    float64 `json:"utilisationBasisSeconds"`
	UtilisationRate            float64 `json:"utilisationRate"`
	CompletedSessionCount      int32   `json:"completedSessionCount"`
	TurnoverRate               float64 `json:"turnoverRate"`
	AvgSessionSeconds          float64 `json:"avgSessionSeconds"`
	P50SessionSeconds          float64 `json:"p50SessionSeconds"`
	P90SessionSeconds          float64 `json:"p90SessionSeconds"`
	AssistRequestCount         int32   `json:"assistRequestCount"`
	AvgAssistResponseSeconds   float64 `json:"avgAssistResponseSeconds"`
	AvgAssistResolutionSeconds float64 `json:"avgAssistResolutionSeconds"`
	AnomalyCount               int32   `json:"anomalyCount"`
}

type analyticsWindowMetricDTO struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name,omitempty"`
	WindowStart string             `json:"windowStart"`
	WindowEnd   string             `json:"windowEnd"`
	Metric      analyticsMetricDTO `json:"metric"`
}

type analyticsSummaryDTO struct {
	Location             locationDTO               `json:"location"`
	Date                 string                    `json:"date"`
	Today                analyticsWindowMetricDTO  `json:"today"`
	CurrentServicePeriod *analyticsWindowMetricDTO `json:"currentServicePeriod,omitempty"`
	Checkpoint           *analyticsCheckpointDTO   `json:"checkpoint,omitempty"`
	DataQuality          analyticsDataQualityDTO   `json:"dataQuality"`
}

type analyticsCheckpointDTO struct {
	ProjectorName    string  `json:"projectorName"`
	ProjectorVersion int32   `json:"projectorVersion"`
	CursorOccurredAt *string `json:"cursorOccurredAt,omitempty"`
	CursorEventID    *string `json:"cursorEventId,omitempty"`
	LagSeconds       float64 `json:"lagSeconds"`
	RebuildStatus    string  `json:"rebuildStatus"`
	UpdatedAt        string  `json:"updatedAt"`
}

type analyticsTimePointDTO struct {
	BucketStart string             `json:"bucketStart"`
	BucketEnd   string             `json:"bucketEnd"`
	Metric      analyticsMetricDTO `json:"metric"`
}

type analyticsComparisonPointDTO struct {
	GroupID     string             `json:"groupId"`
	GroupName   string             `json:"groupName"`
	MetricDate  string             `json:"metricDate"`
	WindowStart string             `json:"windowStart"`
	WindowEnd   string             `json:"windowEnd"`
	Metric      analyticsMetricDTO `json:"metric"`
}

type analyticsDataQualityDTO struct {
	OpenSessionCount       int32   `json:"openSessionCount"`
	LongOpenSessionCount   int32   `json:"longOpenSessionCount"`
	ImpossibleSessionCount int32   `json:"impossibleSessionCount"`
	MissingOccupancyCount  int32   `json:"missingOccupancyCount"`
	StaleDeviceCount       int32   `json:"staleDeviceCount"`
	ProjectorLagSeconds    float64 `json:"projectorLagSeconds"`
	RebuildStatus          string  `json:"rebuildStatus"`
}

type servicePeriodDTO struct {
	ID         string  `json:"id"`
	LocationID string  `json:"locationId"`
	Name       string  `json:"name"`
	DaysOfWeek []int16 `json:"daysOfWeek"`
	StartTime  string  `json:"startTime"`
	EndTime    string  `json:"endTime"`
	IsActive   bool    `json:"isActive"`
	Version    int32   `json:"version"`
}

type layoutEditorSnapshotDTO struct {
	Floors []floorDTO      `json:"floors"`
	Zones  []zoneDTO       `json:"zones"`
	Tables []tableStateDTO `json:"tables"`
}

type displaySnapshotDTO struct {
	OrganisationID string          `json:"organisationId"`
	LocationID     string          `json:"locationId"`
	Cursor         string          `json:"cursor"`
	Tables         []tableStateDTO `json:"tables"`
	Assists        []assistDTO     `json:"assists"`
}

type updateOrganisationRequest struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type updateLocationRequest struct {
	Name            string          `json:"name"`
	Timezone        string          `json:"timezone"`
	Status          string          `json:"status"`
	OperatingConfig json.RawMessage `json:"operatingConfig"`
	FeatureFlags    json.RawMessage `json:"featureFlags"`
}

type floorRequest struct {
	Slug               string          `json:"slug"`
	Name               string          `json:"name"`
	SortOrder          int32           `json:"sortOrder"`
	Canvas             json.RawMessage `json:"canvas"`
	BackgroundAssetRef string          `json:"backgroundAssetRef"`
	ExpectedVersion    int32           `json:"expectedVersion"`
}

type zoneRequest struct {
	FloorID         uuid.UUID `json:"floorId"`
	Name            string    `json:"name"`
	SortOrder       int32     `json:"sortOrder"`
	ExpectedVersion int32     `json:"expectedVersion"`
}

type tableRequest struct {
	FloorID         uuid.UUID       `json:"floorId"`
	ZoneID          uuid.UUID       `json:"zoneId"`
	Label           string          `json:"label"`
	CapacityLabel   string          `json:"capacityLabel"`
	Shape           string          `json:"shape"`
	Geometry        json.RawMessage `json:"geometry"`
	ExpectedVersion int32           `json:"expectedVersion"`
}

type archiveRequest struct {
	ExpectedVersion int32     `json:"expectedVersion"`
	FloorID         uuid.UUID `json:"floorId,omitempty"`
}

type createPairingCodeRequest struct {
	LocationID uuid.UUID `json:"locationId"`
	DeviceType string    `json:"deviceType"`
	TTLSeconds int32     `json:"ttlSeconds"`
}

type pairingCodeDTO struct {
	ID         string  `json:"id"`
	LocationID string  `json:"locationId"`
	DeviceType string  `json:"deviceType"`
	Code       string  `json:"code"`
	ExpiresAt  string  `json:"expiresAt"`
	ConsumedAt *string `json:"consumedAt,omitempty"`
}

type pairDeviceRequest struct {
	Code          string          `json:"code"`
	Platform      string          `json:"platform"`
	AppVersion    string          `json:"appVersion"`
	Name          string          `json:"name"`
	Capabilities  json.RawMessage `json:"capabilities"`
	Configuration json.RawMessage `json:"configuration"`
}

type pairDeviceResponse struct {
	Device     deviceDTO `json:"device"`
	Credential string    `json:"credential"`
}

type deviceHeartbeatRequest struct {
	AppVersion   string          `json:"appVersion"`
	Capabilities json.RawMessage `json:"capabilities"`
}

type servicePeriodRequest struct {
	Name            string  `json:"name"`
	DaysOfWeek      []int16 `json:"daysOfWeek"`
	StartTime       string  `json:"startTime"`
	EndTime         string  `json:"endTime"`
	ExpectedVersion int32   `json:"expectedVersion"`
}

type analyticsRebuildRequest struct {
	LocationID string `json:"locationId"`
	From       string `json:"from"`
	To         string `json:"to"`
}

func (api *API) getOrganisation(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, false)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if id != ctx.OrganisationID {
		writeError(w, http.StatusNotFound, "not_found", "organisation not found", nil)
		return
	}
	var org db.Organisation
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzOrganisation, identity.PermissionOrganisationManage, func(q *db.Queries) error {
		var err error
		org, err = q.GetOrganisation(r.Context(), id)
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organisationFromDB(org))
}

func (api *API) getOwnerSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, true)
	if !ok {
		return
	}
	snapshot, err := api.config.OwnerSnapshot(r.Context(), configActor(ctx))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	locations := make([]locationDTO, 0, len(snapshot.Locations))
	for _, location := range snapshot.Locations {
		locations = append(locations, locationFromDB(location))
	}
	writeJSON(w, http.StatusOK, ownerSnapshotDTO{
		Organisation: organisationFromDB(snapshot.Organisation),
		Locations:    locations,
	})
}

func (api *API) updateOrganisation(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req updateOrganisationRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	org, err := api.config.UpdateOrganisation(r.Context(), configuration.UpdateOrganisationParams{
		TenantActor: configActor(ctx),
		ID:          id,
		Slug:        req.Slug,
		Name:        req.Name,
		Status:      req.Status,
	})
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organisationFromDB(org))
}

func (api *API) getLocation(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if id != ctx.LocationID {
		writeError(w, http.StatusNotFound, "not_found", "location not found", nil)
		return
	}
	var location db.Location
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionLayoutRead, func(q *db.Queries) error {
		var err error
		location, err = q.GetLocation(r.Context(), db.GetLocationParams{ID: id, OrganisationID: ctx.OrganisationID})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locationFromDB(location))
}

func (api *API) updateLocation(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req updateLocationRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	location, err := api.config.UpdateLocation(r.Context(), configuration.UpdateLocationParams{
		TenantActor:     configActor(ctx),
		ID:              id,
		Name:            req.Name,
		Timezone:        req.Timezone,
		Status:          req.Status,
		OperatingConfig: req.OperatingConfig,
		FeatureFlags:    req.FeatureFlags,
	})
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locationFromDB(location))
}

func (api *API) listFloors(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	var floors []db.Floor
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionLayoutRead, func(q *db.Queries) error {
		var err error
		floors, err = q.ListFloorsByLocation(r.Context(), db.ListFloorsByLocationParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]floorDTO, 0, len(floors))
	for _, floor := range floors {
		out = append(out, floorFromDB(floor))
	}
	writeJSON(w, http.StatusOK, map[string]any{"floors": out})
}

func (api *API) getLayoutEditorSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	snapshot, err := api.config.LayoutSnapshot(r.Context(), configActor(ctx))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	floors := make([]floorDTO, 0, len(snapshot.Floors))
	for _, floor := range snapshot.Floors {
		floors = append(floors, floorFromDB(floor))
	}
	zones := make([]zoneDTO, 0, len(snapshot.Zones))
	for _, zone := range snapshot.Zones {
		zones = append(zones, zoneFromDB(zone))
	}
	tables := make([]tableStateDTO, 0, len(snapshot.Tables))
	for _, state := range snapshot.Tables {
		tables = append(tables, tableStateFromDB(state.Table, state.TableOccupancy))
	}
	writeJSON(w, http.StatusOK, layoutEditorSnapshotDTO{Floors: floors, Zones: zones, Tables: tables})
}

func (api *API) createFloor(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.floorInput(w, r)
	if !ok {
		return
	}
	floor, err := api.config.CreateFloor(r.Context(), floorParams(ctx, uuid.Nil, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, floorFromDB(floor))
}

func (api *API) updateFloor(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.floorInput(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	floor, err := api.config.UpdateFloor(r.Context(), floorParams(ctx, id, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, floorFromDB(floor))
}

func (api *API) archiveFloor(w http.ResponseWriter, r *http.Request) {
	api.changeFloorActive(w, r, api.config.ArchiveFloor)
}

func (api *API) restoreFloor(w http.ResponseWriter, r *http.Request) {
	api.changeFloorActive(w, r, api.config.RestoreFloor)
}

func (api *API) changeFloorActive(
	w http.ResponseWriter,
	r *http.Request,
	change func(context.Context, configuration.TenantActor, uuid.UUID, int32) (db.Floor, error),
) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req archiveRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	floor, err := change(r.Context(), configActor(ctx), id, req.ExpectedVersion)
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, floorFromDB(floor))
}

func (api *API) listZones(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	floorID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var zones []db.Zone
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionLayoutRead, func(q *db.Queries) error {
		var err error
		zones, err = q.ListZonesByFloor(r.Context(), db.ListZonesByFloorParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
			FloorID:        floorID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]zoneDTO, 0, len(zones))
	for _, zone := range zones {
		out = append(out, zoneFromDB(zone))
	}
	writeJSON(w, http.StatusOK, map[string]any{"zones": out})
}

func (api *API) createZone(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.zoneInput(w, r)
	if !ok {
		return
	}
	zone, err := api.config.CreateZone(r.Context(), zoneParams(ctx, uuid.Nil, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, zoneFromDB(zone))
}

func (api *API) updateZone(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.zoneInput(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	zone, err := api.config.UpdateZone(r.Context(), zoneParams(ctx, id, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zoneFromDB(zone))
}

func (api *API) archiveZone(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req archiveRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	zone, err := api.config.ArchiveZone(r.Context(), configActor(ctx), id, req.FloorID, req.ExpectedVersion)
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zoneFromDB(zone))
}

func (api *API) listTables(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	floorID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var tables []db.Table
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionLayoutRead, func(q *db.Queries) error {
		var err error
		tables, err = q.ListTablesByFloor(r.Context(), db.ListTablesByFloorParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
			FloorID:        floorID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]tableDTO, 0, len(tables))
	for _, table := range tables {
		out = append(out, tableFromDB(table))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": out})
}

func (api *API) createTable(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.tableInput(w, r)
	if !ok {
		return
	}
	table, err := api.config.CreateTable(r.Context(), tableParams(ctx, uuid.Nil, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tableFromDB(table))
}

func (api *API) updateTable(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.tableInput(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	table, err := api.config.UpdateTable(r.Context(), tableParams(ctx, id, req))
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tableFromDB(table))
}

func (api *API) archiveTable(w http.ResponseWriter, r *http.Request) {
	api.changeTableActive(w, r, api.config.ArchiveTable)
}

func (api *API) restoreTable(w http.ResponseWriter, r *http.Request) {
	api.changeTableActive(w, r, api.config.RestoreTable)
}

func (api *API) changeTableActive(
	w http.ResponseWriter,
	r *http.Request,
	change func(context.Context, configuration.TenantActor, uuid.UUID, int32) (db.Table, error),
) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req archiveRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	table, err := change(r.Context(), configActor(ctx), id, req.ExpectedVersion)
	if err != nil {
		api.writeConfigurationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tableFromDB(table))
}

func (api *API) getTable(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	tableID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	state, ok := api.loadTableState(w, r, ctx, tableID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, tableStateFromDB(state.Table, state.TableOccupancy))
}

func (api *API) getLocationState(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	var states []db.ListLocationTableStatesRow
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionOperationsRead, func(q *db.Queries) error {
		var err error
		states, err = q.ListLocationTableStates(r.Context(), db.ListLocationTableStatesParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]tableStateDTO, 0, len(states))
	for _, state := range states {
		out = append(out, tableStateFromDB(state.Table, state.TableOccupancy))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": out})
}

func (api *API) listAssists(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "active" {
		writeError(w, http.StatusBadRequest, "validation_failed", "status must be active", nil)
		return
	}
	var assists []db.AssistRequest
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionOperationsRead, func(q *db.Queries) error {
		var err error
		assists, err = q.ListActiveAssistsByLocation(r.Context(), db.ListActiveAssistsByLocationParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]assistDTO, 0, len(assists))
	for _, assist := range assists {
		out = append(out, assistFromDB(assist))
	}
	writeJSON(w, http.StatusOK, map[string]any{"assists": out})
}

func (api *API) getAuditTimeline(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, false)
	if !ok {
		return
	}
	entityType := strings.TrimSpace(r.URL.Query().Get("entityType"))
	if entityType == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "entityType is required", nil)
		return
	}
	entityID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("entityId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "entityId must be a UUID", nil)
		return
	}
	limit := int32(50)
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "validation_failed", "limit must be between 1 and 200", nil)
			return
		}
		limit = int32(parsed)
	}

	var events []timelineEventDTO
	err = api.inAuthorizedTenantDBTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionAuditRead, func(dbt db.DBTX, _ *db.Queries) error {
		rows, err := dbt.Query(r.Context(), `
SELECT id, event_type, schema_version, occurred_at, actor_ref, device_id, entity_type, entity_id, entity_version, command_id, event_data
FROM operational_events
WHERE organisation_id = $1
  AND location_id = $2
  AND entity_type = $3
  AND entity_id = $4
ORDER BY occurred_at, id
LIMIT $5
`, ctx.OrganisationID, ctx.LocationID, entityType, entityID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var event timelineEventDTO
			var id uuid.UUID
			var occurredAt pgtype.Timestamptz
			var deviceID uuid.NullUUID
			var entityID uuid.UUID
			var entityVersion pgtype.Int4
			var commandID uuid.NullUUID
			if err := rows.Scan(&id, &event.Type, &event.SchemaVersion, &occurredAt, &event.ActorRef, &deviceID, &event.EntityType, &entityID, &entityVersion, &commandID, &event.Data); err != nil {
				return err
			}
			event.ID = id.String()
			event.OccurredAt = timeString(occurredAt)
			event.DeviceID = uuidPtr(deviceID)
			event.EntityID = entityID.String()
			event.EntityVersion = int4Ptr(entityVersion)
			event.CommandID = uuidPtr(commandID)
			events = append(events, event)
		}
		return rows.Err()
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (api *API) getAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx, locationID, ok := api.analyticsInput(w, r)
	if !ok {
		return
	}
	date, ok := parseQueryDate(w, r, "date", time.Now())
	if !ok {
		return
	}
	summary, err := api.analytics.Summary(r.Context(), analyticsActor(ctx), locationID, date, time.Now().UTC())
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, analyticsSummaryFromDomain(summary))
}

func (api *API) getAnalyticsTimeseries(w http.ResponseWriter, r *http.Request) {
	ctx, locationID, ok := api.analyticsInput(w, r)
	if !ok {
		return
	}
	from, to, ok := parseDateRange(w, r)
	if !ok {
		return
	}
	grain := strings.TrimSpace(r.URL.Query().Get("grain"))
	if grain == "" {
		grain = analytics.GrainDay
	}
	actor := analyticsActor(ctx)
	actor.LocationID = locationID
	points, err := api.analytics.Timeseries(r.Context(), actor, from, to, grain)
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	out := make([]analyticsTimePointDTO, 0, len(points))
	for _, point := range points {
		out = append(out, analyticsTimePointFromDomain(point))
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": out})
}

func (api *API) getAnalyticsComparison(w http.ResponseWriter, r *http.Request) {
	ctx, locationID, ok := api.analyticsInput(w, r)
	if !ok {
		return
	}
	from, to, ok := parseDateRange(w, r)
	if !ok {
		return
	}
	groupBy := strings.TrimSpace(r.URL.Query().Get("groupBy"))
	actor := analyticsActor(ctx)
	actor.LocationID = locationID
	points, err := api.analytics.Comparison(r.Context(), actor, from, to, groupBy)
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	out := make([]analyticsComparisonPointDTO, 0, len(points))
	for _, point := range points {
		out = append(out, analyticsComparisonPointFromDomain(point))
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": out})
}

func (api *API) getAnalyticsDataQuality(w http.ResponseWriter, r *http.Request) {
	ctx, locationID, ok := api.analyticsInput(w, r)
	if !ok {
		return
	}
	actor := analyticsActor(ctx)
	actor.LocationID = locationID
	quality, err := api.analytics.DataQuality(r.Context(), actor)
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, analyticsDataQualityFromDomain(quality))
}

func (api *API) rebuildAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	var req analyticsRebuildRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	locationID, err := uuid.Parse(strings.TrimSpace(req.LocationID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "locationId must be a UUID", nil)
		return
	}
	from, err := time.Parse("2006-01-02", strings.TrimSpace(req.From))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "from must be YYYY-MM-DD", nil)
		return
	}
	to, err := time.Parse("2006-01-02", strings.TrimSpace(req.To))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "to must be YYYY-MM-DD", nil)
		return
	}
	err = api.analytics.Rebuild(r.Context(), analytics.RebuildParams{
		TenantActor: analyticsActor(ctx),
		LocationID:  locationID,
		From:        from,
		To:          to,
	})
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted"})
}

func (api *API) listServicePeriods(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	periods, err := api.analytics.ListServicePeriods(r.Context(), analyticsActor(ctx))
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	out := make([]servicePeriodDTO, 0, len(periods))
	for _, period := range periods {
		out = append(out, servicePeriodFromDB(period))
	}
	writeJSON(w, http.StatusOK, map[string]any{"servicePeriods": out})
}

func (api *API) createServicePeriod(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.servicePeriodInput(w, r)
	if !ok {
		return
	}
	period, err := api.analytics.CreateServicePeriod(r.Context(), analyticsActor(ctx), servicePeriodInput(req))
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, servicePeriodFromDB(period))
}

func (api *API) updateServicePeriod(w http.ResponseWriter, r *http.Request) {
	ctx, req, ok := api.servicePeriodInput(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	period, err := api.analytics.UpdateServicePeriod(r.Context(), analyticsActor(ctx), id, servicePeriodInput(req))
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, servicePeriodFromDB(period))
}

func (api *API) archiveServicePeriod(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req archiveRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	period, err := api.analytics.ArchiveServicePeriod(r.Context(), analyticsActor(ctx), id, req.ExpectedVersion)
	if err != nil {
		api.writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, servicePeriodFromDB(period))
}

func (api *API) listDevices(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, false)
	if !ok {
		return
	}
	devices, err := api.ident.ListDevices(r.Context(), identityActor(ctx))
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	out := make([]deviceDTO, 0, len(devices))
	for _, device := range devices {
		out = append(out, deviceFromDB(device))
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

func (api *API) createDevicePairingCode(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, true)
	if !ok {
		return
	}
	var req createPairingCodeRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	result, err := api.ident.CreatePairingCode(r.Context(), identity.CreatePairingCodeParams{
		TenantActor: identity.TenantActor{
			OrganisationID: ctx.OrganisationID,
			LocationID:     req.LocationID,
			ActorRef:       ctx.ActorRef,
		},
		DeviceType: req.DeviceType,
		TTL:        time.Duration(req.TTLSeconds) * time.Second,
	})
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"pairingCode": pairingCodeFromDB(result.PairingCode, result.Code),
	})
}

func (api *API) pairDevice(w http.ResponseWriter, r *http.Request) {
	var req pairDeviceRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	result, err := api.ident.PairDevice(r.Context(), identity.PairDeviceParams{
		Code:          req.Code,
		Platform:      req.Platform,
		AppVersion:    req.AppVersion,
		Name:          req.Name,
		Capabilities:  req.Capabilities,
		Configuration: req.Configuration,
	})
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pairDeviceResponse{
		Device:     deviceFromDB(result.Device),
		Credential: result.Secret,
	})
}

func (api *API) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	var req deviceHeartbeatRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	device, err := api.ident.RecordHeartbeat(r.Context(), identity.HeartbeatParams{
		Secret:       credential,
		AppVersion:   req.AppVersion,
		Capabilities: req.Capabilities,
	})
	if err != nil {
		if api.metrics != nil {
			api.metrics.ObserveDeviceHeartbeat("failure", "", "")
		}
		api.writeIdentityError(w, err)
		return
	}
	if api.metrics != nil {
		api.metrics.ObserveDeviceHeartbeat("success", device.DeviceType, device.Platform)
	}
	writeJSON(w, http.StatusOK, deviceFromDB(device))
}

func (api *API) getDisplaySnapshot(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	device, err := api.ident.AuthenticateDeviceCredential(r.Context(), credential)
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	if !device.LocationID.Valid {
		writeError(w, http.StatusForbidden, "forbidden", "device is not assigned to a location", nil)
		return
	}
	ctx := requestContext{
		OrganisationID: device.OrganisationID,
		LocationID:     device.LocationID.UUID,
		ActorRef:       "device:" + device.ID.String(),
		DeviceID:       uuid.NullUUID{UUID: device.ID, Valid: true},
		PrincipalType:  principalDevice,
		Permissions:    devicePermissions(device),
	}
	var states []db.ListLocationTableStatesRow
	var assists []db.AssistRequest
	err = api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionOperationsRead, func(q *db.Queries) error {
		var err error
		states, err = q.ListLocationTableStates(r.Context(), db.ListLocationTableStatesParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing display table states: %w", err)
		}
		assists, err = q.ListActiveAssistsByLocation(r.Context(), db.ListActiveAssistsByLocationParams{
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing display assists: %w", err)
		}
		return nil
	})
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	tables := make([]tableStateDTO, 0, len(states))
	for _, state := range states {
		tables = append(tables, tableStateFromDB(state.Table, state.TableOccupancy))
	}
	out := make([]assistDTO, 0, len(assists))
	for _, assist := range assists {
		out = append(out, assistFromDB(assist))
	}
	writeJSON(w, http.StatusOK, displaySnapshotDTO{
		OrganisationID: ctx.OrganisationID.String(),
		LocationID:     ctx.LocationID.String(),
		Cursor:         time.Now().UTC().Format(time.RFC3339Nano),
		Tables:         tables,
		Assists:        out,
	})
}

func (api *API) revokeDevice(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, true)
	if !ok {
		return
	}
	deviceID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	device, err := api.ident.RevokeDevice(r.Context(), identityActor(ctx), deviceID)
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deviceFromDB(device))
}

func (api *API) listMemberships(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, false, false)
	if !ok {
		return
	}
	orgMemberships, locMemberships, err := api.listMembershipsForContext(r.Context(), ctx, identity.PermissionOrganisationManage)
	if err != nil {
		api.writeDBError(w, err)
		return
	}
	out := make([]membershipDTO, 0, len(orgMemberships)+len(locMemberships))
	for _, membership := range orgMemberships {
		out = append(out, organisationMembershipFromDB(membership))
	}
	for _, membership := range locMemberships {
		out = append(out, locationMembershipFromDB(membership))
	}
	writeJSON(w, http.StatusOK, map[string]any{"memberships": out})
}

func (api *API) occupyTable(w http.ResponseWriter, r *http.Request) {
	ctx, req, tableID, body, ok := api.commandInput(w, r)
	if !ok {
		return
	}
	arg := operations.OccupyTableParams{
		OrganisationID:  ctx.OrganisationID,
		LocationID:      ctx.LocationID,
		TableID:         tableID,
		ExpectedVersion: req.ExpectedVersion,
		PartySize:       req.PartySize,
		ActorRef:        ctx.ActorRef,
		Source:          "api",
		Command:         commandMeta("table.occupy", req.CommandID, body, ctx.DeviceID),
	}
	result, replay, err := api.ops.OccupyTableCommand(r.Context(), arg, func(result operations.OccupyTableResult) (int32, []byte, error) {
		return marshalStatus(http.StatusOK, tableCommandResponseFromDB(result.Table, result.Occupancy, result.Session))
	})
	api.writeCommandResult(w, result, replay, err)
}

func (api *API) clearTable(w http.ResponseWriter, r *http.Request) {
	ctx, req, tableID, body, ok := api.commandInput(w, r)
	if !ok {
		return
	}
	arg := operations.ClearTableParams{
		OrganisationID:  ctx.OrganisationID,
		LocationID:      ctx.LocationID,
		TableID:         tableID,
		ExpectedVersion: req.ExpectedVersion,
		ActorRef:        ctx.ActorRef,
		Command:         commandMeta("table.clear", req.CommandID, body, ctx.DeviceID),
	}
	result, replay, err := api.ops.ClearTableCommand(r.Context(), arg, func(result operations.ClearTableResult) (int32, []byte, error) {
		return marshalStatus(http.StatusOK, tableCommandResponseFromDB(result.Table, result.Occupancy, result.Session))
	})
	api.writeCommandResult(w, result, replay, err)
}

func (api *API) acknowledgeAssist(w http.ResponseWriter, r *http.Request) {
	api.changeAssist(w, r, "assist.acknowledge", api.ops.AcknowledgeAssistCommand)
}

func (api *API) resolveAssist(w http.ResponseWriter, r *http.Request) {
	api.changeAssist(w, r, "assist.resolve", api.ops.ResolveAssistCommand)
}

func (api *API) cancelAssist(w http.ResponseWriter, r *http.Request) {
	api.changeAssist(w, r, "assist.cancel", api.ops.CancelAssistCommand)
}

func (api *API) changeAssist(
	w http.ResponseWriter,
	r *http.Request,
	commandType string,
	command func(context.Context, operations.ChangeAssistParams, func(db.AssistRequest) (int32, []byte, error)) (db.AssistRequest, operations.CommandReplay, error),
) {
	ctx, req, assistID, body, ok := api.commandInput(w, r)
	if !ok {
		return
	}
	arg := operations.ChangeAssistParams{
		OrganisationID:  ctx.OrganisationID,
		LocationID:      ctx.LocationID,
		AssistID:        assistID,
		ExpectedVersion: req.ExpectedVersion,
		ActorRef:        ctx.ActorRef,
		Command:         commandMeta(commandType, req.CommandID, body, ctx.DeviceID),
	}
	result, replay, err := command(r.Context(), arg, func(result db.AssistRequest) (int32, []byte, error) {
		return marshalStatus(http.StatusOK, assistFromDB(result))
	})
	api.writeCommandResult(w, result, replay, err)
}

func (api *API) floorInput(w http.ResponseWriter, r *http.Request) (requestContext, floorRequest, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, floorRequest{}, false
	}
	var req floorRequest
	if !decodeJSONBody(w, r, &req) {
		return requestContext{}, floorRequest{}, false
	}
	return ctx, req, true
}

func (api *API) zoneInput(w http.ResponseWriter, r *http.Request) (requestContext, zoneRequest, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, zoneRequest{}, false
	}
	var req zoneRequest
	if !decodeJSONBody(w, r, &req) {
		return requestContext{}, zoneRequest{}, false
	}
	return ctx, req, true
}

func (api *API) tableInput(w http.ResponseWriter, r *http.Request) (requestContext, tableRequest, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, tableRequest{}, false
	}
	var req tableRequest
	if !decodeJSONBody(w, r, &req) {
		return requestContext{}, tableRequest{}, false
	}
	return ctx, req, true
}

func (api *API) servicePeriodInput(w http.ResponseWriter, r *http.Request) (requestContext, servicePeriodRequest, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, servicePeriodRequest{}, false
	}
	var req servicePeriodRequest
	if !decodeJSONBody(w, r, &req) {
		return requestContext{}, servicePeriodRequest{}, false
	}
	return ctx, req, true
}

func (api *API) commandInput(w http.ResponseWriter, r *http.Request) (requestContext, commandRequest, uuid.UUID, []byte, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	resourceID, ok := pathUUID(w, r, "id")
	if !ok {
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(body); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body is invalid", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	var req commandRequest
	if err := json.Unmarshal(buf.Bytes(), &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be JSON", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	if req.CommandID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "commandId is required", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	if req.ExpectedVersion < 1 {
		writeError(w, http.StatusBadRequest, "validation_failed", "expectedVersion must be at least 1", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	if req.PartySize != nil && *req.PartySize < 1 {
		writeError(w, http.StatusBadRequest, "validation_failed", "partySize must be at least 1", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	canonical, err := canonicalJSON(buf.Bytes())
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be a JSON object", nil)
		return requestContext{}, commandRequest{}, uuid.Nil, nil, false
	}
	return ctx, req, resourceID, canonical, true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be valid JSON", nil)
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must contain one JSON object", nil)
		return false
	}
	return true
}

func (api *API) requestContext(w http.ResponseWriter, r *http.Request, requireLocation bool, requireActor bool) (requestContext, bool) {
	if !isDevelopmentEnvironment(api.cfg.Environment) {
		if api.cfg.TrustedIdentityHeaders && hasClientIdentityHeaders(r) {
			if !api.authorizeInternalRequest(w, r) {
				return requestContext{}, false
			}
		} else {
			if hasClientIdentityHeaders(r) {
				writeError(w, http.StatusBadRequest, "validation_failed", "client-supplied identity headers are not accepted", nil)
				return requestContext{}, false
			}
			return api.authenticatedRequestContext(w, r, requireLocation, requireActor)
		}
	}

	orgID, ok := parseHeaderUUID(w, r, headerOrganisationID, true)
	if !ok {
		return requestContext{}, false
	}
	locationID, ok := parseHeaderUUID(w, r, headerLocationID, requireLocation)
	if !ok {
		return requestContext{}, false
	}
	actorRef := strings.TrimSpace(r.Header.Get(headerActorRef))
	if requireActor && actorRef == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", headerActorRef+" is required", nil)
		return requestContext{}, false
	}
	deviceID, ok := parseHeaderUUID(w, r, headerDeviceID, false)
	if !ok {
		return requestContext{}, false
	}
	return requestContext{
		OrganisationID: orgID,
		LocationID:     locationID,
		ActorRef:       actorRef,
		DeviceID:       uuid.NullUUID{UUID: deviceID, Valid: deviceID != uuid.Nil},
		PrincipalType:  principalTrustedHeader,
	}, true
}

func hasClientIdentityHeaders(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get(headerActorRef)) != "" ||
		strings.TrimSpace(r.Header.Get(headerDeviceID)) != ""
}

func routeAuthPolicy(r *http.Request) authPolicy {
	policy := authPolicy{
		Permission:      routePermission(r),
		AllowUser:       true,
		AllowDevice:     routeAllowsDevice(r),
		RequireLocation: routeRequiresLocation(r),
		RequireActor:    routeRequiresActor(r),
	}
	if routeRequiresUser(r) {
		policy.AllowDevice = false
	}
	return policy
}

func routeRequiresUser(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/v1/platform/") ||
		strings.HasPrefix(path, "/v1/organisations/") ||
		strings.HasPrefix(path, "/v1/devices") ||
		strings.HasPrefix(path, "/v1/integrations") ||
		strings.HasPrefix(path, "/v1/analytics") ||
		strings.HasPrefix(path, "/v1/service-periods") ||
		strings.Contains(path, "/qr-capabilities") {
		return true
	}
	switch {
	case r.Method != http.MethodGet && strings.HasPrefix(path, "/v1/locations/"):
		return true
	case r.Method != http.MethodGet && strings.HasPrefix(path, "/v1/floors"):
		return true
	case r.Method != http.MethodGet && strings.HasPrefix(path, "/v1/zones"):
		return true
	case r.Method != http.MethodGet && strings.HasPrefix(path, "/v1/tables") &&
		!strings.HasSuffix(path, "/occupy") &&
		!strings.HasSuffix(path, "/clear"):
		return true
	default:
		return false
	}
}

func routeAllowsDevice(r *http.Request) bool {
	path := r.URL.Path
	if path == "/v1/location-state" ||
		path == "/v1/assists" ||
		strings.HasPrefix(path, "/v1/floors") ||
		strings.HasPrefix(path, "/v1/tables") ||
		strings.HasPrefix(path, "/v1/floors/") {
		return true
	}
	if strings.HasPrefix(path, "/v1/assists/") {
		return r.Method == http.MethodPost
	}
	if strings.HasPrefix(path, "/v1/locations/") {
		return r.Method == http.MethodGet
	}
	return false
}

func routePermission(r *http.Request) string {
	path := r.URL.Path
	if strings.HasPrefix(path, "/v1/platform/") {
		return identity.PermissionPlatformAdmin
	}
	if strings.HasPrefix(path, "/v1/devices") {
		return identity.PermissionDeviceManage
	}
	if strings.HasPrefix(path, "/v1/integrations") {
		if r.Method == http.MethodGet {
			return identity.PermissionIntegrationsRead
		}
		return identity.PermissionIntegrationsManage
	}
	if strings.HasPrefix(path, "/v1/analytics") ||
		strings.HasPrefix(path, "/v1/service-periods") {
		return identity.PermissionAnalyticsRead
	}
	if strings.HasPrefix(path, "/v1/organisations/") {
		if r.Method == http.MethodGet {
			return identity.PermissionOrganisationManage
		}
		return identity.PermissionOrganisationManage
	}
	if strings.HasPrefix(path, "/v1/locations/") {
		if r.Method == http.MethodGet {
			return identity.PermissionLayoutRead
		}
		return identity.PermissionLocationManage
	}
	if strings.HasPrefix(path, "/v1/audit/") {
		return identity.PermissionAuditRead
	}
	if strings.HasPrefix(path, "/v1/owner/snapshot") ||
		strings.HasPrefix(path, "/v1/layout/") ||
		strings.Contains(path, "/qr-capabilities") {
		if r.Method == http.MethodGet {
			return identity.PermissionLayoutRead
		}
		return identity.PermissionLayoutWrite
	}
	if strings.HasPrefix(path, "/v1/floors") ||
		strings.HasPrefix(path, "/v1/zones") {
		if r.Method == http.MethodGet {
			return identity.PermissionLayoutRead
		}
		return identity.PermissionLayoutWrite
	}
	if strings.HasPrefix(path, "/v1/tables") {
		if strings.HasSuffix(path, "/occupy") || strings.HasSuffix(path, "/clear") {
			return identity.PermissionOperationsWrite
		}
		if r.Method == http.MethodGet {
			return identity.PermissionLayoutRead
		}
		return identity.PermissionLayoutWrite
	}
	if path == "/v1/location-state" ||
		path == "/v1/assists" {
		return identity.PermissionOperationsRead
	}
	if strings.HasPrefix(path, "/v1/assists/") {
		return identity.PermissionOperationsWrite
	}
	if path == "/v1/memberships" {
		return identity.PermissionOrganisationManage
	}
	return ""
}

func routeRequiresLocation(r *http.Request) bool {
	path := r.URL.Path
	return path == "/v1/location-state" ||
		path == "/v1/assists" ||
		strings.HasPrefix(path, "/v1/locations/") ||
		strings.HasPrefix(path, "/v1/floors") ||
		strings.HasPrefix(path, "/v1/zones") ||
		strings.HasPrefix(path, "/v1/tables") ||
		strings.HasPrefix(path, "/v1/layout/") ||
		strings.HasPrefix(path, "/v1/audit/") ||
		strings.HasPrefix(path, "/v1/analytics") ||
		strings.HasPrefix(path, "/v1/service-periods") ||
		strings.HasPrefix(path, "/v1/integrations")
}

func routeRequiresActor(r *http.Request) bool {
	return r.Method != http.MethodGet ||
		strings.HasPrefix(r.URL.Path, "/v1/owner/snapshot") ||
		strings.HasPrefix(r.URL.Path, "/v1/layout/editor-snapshot") ||
		strings.HasPrefix(r.URL.Path, "/v1/service-periods")
}

func (api *API) inTenantTx(ctx context.Context, req requestContext, fn func(*db.Queries) error) error {
	tx, err := api.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	locationValue := ""
	if req.LocationID != uuid.Nil {
		locationValue = req.LocationID.String()
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, req.OrganisationID.String(), locationValue); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(db.New(tx)); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (api *API) inAuthorizedTenantTx(
	ctx context.Context,
	req requestContext,
	scope tenantAuthzScope,
	permission string,
	fn func(*db.Queries) error,
) error {
	return api.inAuthorizedTenantDBTx(ctx, req, scope, permission, func(_ db.DBTX, q *db.Queries) error {
		return fn(q)
	})
}

func (api *API) inAuthorizedTenantDBTx(
	ctx context.Context,
	req requestContext,
	scope tenantAuthzScope,
	permission string,
	fn func(db.DBTX, *db.Queries) error,
) error {
	return api.inTenantDBTx(ctx, req, func(dbt db.DBTX, q *db.Queries) error {
		if err := requireRequestPermission(ctx, q, req, scope, permission); err != nil {
			return err
		}
		return fn(dbt, q)
	})
}

func requireRequestPermission(
	ctx context.Context,
	q *db.Queries,
	req requestContext,
	scope tenantAuthzScope,
	permission string,
) error {
	if permission == "" || hasPermission(req.Permissions, permission) {
		return nil
	}
	if strings.TrimSpace(req.ActorRef) == "" {
		return errForbidden
	}

	var (
		allowed bool
		err     error
	)
	switch scope {
	case tenantAuthzOrganisation:
		allowed, err = q.ActorHasOrganisationPermission(ctx, db.ActorHasOrganisationPermissionParams{
			OrganisationID: req.OrganisationID,
			MemberRef:      req.ActorRef,
			PermissionName: permission,
		})
	case tenantAuthzLocation:
		if req.LocationID == uuid.Nil {
			return errForbidden
		}
		allowed, err = q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
			OrganisationID: req.OrganisationID,
			LocationID:     req.LocationID,
			MemberRef:      req.ActorRef,
			PermissionName: permission,
		})
	default:
		return errForbidden
	}
	if err != nil {
		return fmt.Errorf("checking actor permission: %w", err)
	}
	if !allowed {
		return errForbidden
	}
	return nil
}

func (api *API) inTenantDBTx(ctx context.Context, req requestContext, fn func(db.DBTX, *db.Queries) error) error {
	tx, err := api.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	locationValue := ""
	if req.LocationID != uuid.Nil {
		locationValue = req.LocationID.String()
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, req.OrganisationID.String(), locationValue); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(tx, db.New(tx)); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (api *API) loadTableState(w http.ResponseWriter, r *http.Request, ctx requestContext, tableID uuid.UUID) (db.GetTableDetailRow, bool) {
	var state db.GetTableDetailRow
	err := api.inAuthorizedTenantTx(r.Context(), ctx, tenantAuthzLocation, identity.PermissionLayoutRead, func(q *db.Queries) error {
		var err error
		state, err = q.GetTableDetail(r.Context(), db.GetTableDetailParams{
			ID:             tableID,
			OrganisationID: ctx.OrganisationID,
			LocationID:     ctx.LocationID,
		})
		return err
	})
	if err != nil {
		api.writeDBError(w, err)
		return db.GetTableDetailRow{}, false
	}
	return state, true
}

func (api *API) writeCommandResult(w http.ResponseWriter, _ any, replay operations.CommandReplay, err error) {
	if err != nil {
		api.writeDomainError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if replay.Replayed {
		w.Header().Set("X-Seatd-Idempotency-Replayed", "true")
	}
	w.WriteHeader(int(replay.Status))
	if _, err := w.Write(replay.Body); err != nil {
		api.logger.Error("writing command response failed", "error", err)
	}
}

func (api *API) writeDomainError(w http.ResponseWriter, err error) {
	var versionErr operations.VersionConflictError
	switch {
	case errors.Is(err, operations.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, "idempotency_conflict", "command id was already used with a different payload", nil)
	case errors.As(err, &versionErr):
		writeError(w, http.StatusConflict, "version_conflict", "entity version conflict", map[string]string{
			"entity":   versionErr.Entity,
			"id":       versionErr.ID,
			"expected": strconv.Itoa(int(versionErr.Expected)),
			"current":  strconv.Itoa(int(versionErr.Current)),
		})
	case errors.Is(err, operations.ErrVersionConflict):
		writeError(w, http.StatusConflict, "version_conflict", "entity version conflict", nil)
	case errors.Is(err, operations.ErrAlreadyOccupied):
		writeError(w, http.StatusConflict, "already_occupied", "table is already occupied", nil)
	case errors.Is(err, operations.ErrAlreadyAvailable):
		writeError(w, http.StatusConflict, "already_available", "table is already available", nil)
	case errors.Is(err, operations.ErrNoActiveSession):
		writeError(w, http.StatusConflict, "no_active_session", "table has no active session", nil)
	case errors.Is(err, operations.ErrAssistAlreadyResolved):
		writeError(w, http.StatusConflict, "assist_already_resolved", "assist is already resolved", nil)
	case errors.Is(err, operations.ErrInvalidAssistTransition):
		writeError(w, http.StatusConflict, "validation_failed", "assist status transition is not allowed", nil)
	case errors.Is(err, operations.ErrActionNotEnabled):
		writeError(w, http.StatusBadRequest, "action_not_enabled", "guest action is not enabled", nil)
	case errors.Is(err, operations.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many guest requests", nil)
	case errors.Is(err, operations.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("domain command failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func (api *API) writeConfigurationError(w http.ResponseWriter, err error) {
	var conflict configuration.ConflictError
	switch {
	case errors.Is(err, configuration.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, configuration.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "request validation failed", nil)
	case errors.As(err, &conflict):
		writeError(w, http.StatusConflict, "already_exists", conflict.Error(), nil)
	case errors.Is(err, configuration.ErrConflict):
		writeError(w, http.StatusConflict, "already_exists", "a resource with this unique value already exists", nil)
	case errors.Is(err, configuration.ErrVersionConflict):
		writeError(w, http.StatusConflict, "version_conflict", "entity version conflict", nil)
	case errors.Is(err, configuration.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("configuration request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func (api *API) writeIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "credential is invalid", nil)
	case errors.Is(err, identity.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, identity.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "request validation failed", nil)
	case errors.Is(err, identity.ErrAlreadyRevoked):
		writeError(w, http.StatusConflict, "already_revoked", "device is already revoked", nil)
	case errors.Is(err, identity.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("identity request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func (api *API) writeAnalyticsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, analytics.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, analytics.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "request validation failed", nil)
	case errors.Is(err, analytics.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("analytics request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func (api *API) writeDBError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("database request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message, Details: details}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func marshalStatus(status int, value any) (int32, []byte, error) {
	body, err := json.Marshal(value)
	return int32(status), body, err
}

func parseHeaderUUID(w http.ResponseWriter, r *http.Request, name string, required bool) (uuid.UUID, bool) {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		if required {
			writeError(w, http.StatusBadRequest, "validation_failed", name+" is required", nil)
			return uuid.Nil, false
		}
		return uuid.Nil, true
	}
	id, err := uuid.Parse(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", name+" must be a UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", name+" must be a UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func bearerCredential(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	credential, ok := strings.CutPrefix(value, "Bearer ")
	if !ok || strings.TrimSpace(credential) == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "bearer credential is required", nil)
		return "", false
	}
	return strings.TrimSpace(credential), true
}

func canonicalJSON(body []byte) ([]byte, error) {
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func commandMeta(commandType string, commandID uuid.UUID, canonicalBody []byte, deviceID uuid.NullUUID) operations.CommandMeta {
	sum := sha256.Sum256(canonicalBody)
	return operations.CommandMeta{
		ID:          commandID,
		Type:        commandType,
		PayloadHash: sum[:],
		DeviceID:    deviceID,
	}
}

func configActor(ctx requestContext) configuration.TenantActor {
	return configuration.TenantActor{
		OrganisationID: ctx.OrganisationID,
		LocationID:     ctx.LocationID,
		ActorRef:       ctx.ActorRef,
	}
}

func floorParams(ctx requestContext, id uuid.UUID, req floorRequest) configuration.UpsertFloorParams {
	return configuration.UpsertFloorParams{
		TenantActor:        configActor(ctx),
		ID:                 id,
		Slug:               req.Slug,
		Name:               req.Name,
		SortOrder:          req.SortOrder,
		Canvas:             req.Canvas,
		BackgroundAssetRef: req.BackgroundAssetRef,
		ExpectedVersion:    req.ExpectedVersion,
	}
}

func zoneParams(ctx requestContext, id uuid.UUID, req zoneRequest) configuration.UpsertZoneParams {
	return configuration.UpsertZoneParams{
		TenantActor:     configActor(ctx),
		ID:              id,
		FloorID:         req.FloorID,
		Name:            req.Name,
		SortOrder:       req.SortOrder,
		ExpectedVersion: req.ExpectedVersion,
	}
}

func tableParams(ctx requestContext, id uuid.UUID, req tableRequest) configuration.UpsertTableParams {
	return configuration.UpsertTableParams{
		TenantActor:     configActor(ctx),
		ID:              id,
		FloorID:         req.FloorID,
		ZoneID:          req.ZoneID,
		Label:           req.Label,
		CapacityLabel:   req.CapacityLabel,
		Shape:           req.Shape,
		Geometry:        req.Geometry,
		ExpectedVersion: req.ExpectedVersion,
	}
}

func analyticsActor(ctx requestContext) analytics.TenantActor {
	return analytics.TenantActor{
		OrganisationID: ctx.OrganisationID,
		LocationID:     ctx.LocationID,
		ActorRef:       ctx.ActorRef,
	}
}

func identityActor(ctx requestContext) identity.TenantActor {
	return identity.TenantActor{
		OrganisationID: ctx.OrganisationID,
		LocationID:     ctx.LocationID,
		ActorRef:       ctx.ActorRef,
	}
}

func servicePeriodInput(req servicePeriodRequest) analytics.ServicePeriodInput {
	return analytics.ServicePeriodInput{
		Name:            req.Name,
		DaysOfWeek:      req.DaysOfWeek,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		ExpectedVersion: req.ExpectedVersion,
	}
}

func (api *API) analyticsInput(w http.ResponseWriter, r *http.Request) (requestContext, uuid.UUID, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, uuid.Nil, false
	}
	locationID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("locationId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "locationId must be a UUID", nil)
		return requestContext{}, uuid.Nil, false
	}
	if locationID != ctx.LocationID {
		writeError(w, http.StatusNotFound, "not_found", "location not found", nil)
		return requestContext{}, uuid.Nil, false
	}
	return ctx, locationID, true
}

func parseQueryDate(w http.ResponseWriter, r *http.Request, key string, fallback time.Time) (time.Time, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", key+" must be YYYY-MM-DD", nil)
		return time.Time{}, false
	}
	return parsed, true
}

func parseDateRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	from, ok := parseQueryDate(w, r, "from", time.Time{})
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	to, ok := parseQueryDate(w, r, "to", time.Time{})
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		writeError(w, http.StatusBadRequest, "validation_failed", "from and to are required and from must be before to", nil)
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}

func (api *API) listMembershipsForContext(ctx context.Context, req requestContext, permission string) ([]db.OrganisationMembership, []db.LocationMembership, error) {
	var orgMemberships []db.OrganisationMembership
	var locMemberships []db.LocationMembership
	err := api.inAuthorizedTenantDBTx(ctx, req, tenantAuthzOrganisation, permission, func(dbt db.DBTX, _ *db.Queries) error {
		dbRows, err := dbt.Query(ctx, `
SELECT id, organisation_id, member_ref, role, created_at, updated_at, user_profile_id, disabled_at
FROM organisation_memberships
WHERE organisation_id = $1 AND disabled_at IS NULL
ORDER BY role, member_ref
`, req.OrganisationID)
		if err != nil {
			return err
		}
		defer dbRows.Close()
		for dbRows.Next() {
			var membership db.OrganisationMembership
			if err := dbRows.Scan(&membership.ID, &membership.OrganisationID, &membership.MemberRef, &membership.Role, &membership.CreatedAt, &membership.UpdatedAt, &membership.UserProfileID, &membership.DisabledAt); err != nil {
				return err
			}
			orgMemberships = append(orgMemberships, membership)
		}
		if err := dbRows.Err(); err != nil {
			return err
		}
		if req.LocationID == uuid.Nil {
			return nil
		}
		locationRows, err := dbt.Query(ctx, `
SELECT id, organisation_id, location_id, member_ref, role, created_at, updated_at, user_profile_id, disabled_at
FROM location_memberships
WHERE organisation_id = $1 AND location_id = $2 AND disabled_at IS NULL
ORDER BY role, member_ref
`, req.OrganisationID, req.LocationID)
		if err != nil {
			return err
		}
		defer locationRows.Close()
		for locationRows.Next() {
			var membership db.LocationMembership
			if err := locationRows.Scan(&membership.ID, &membership.OrganisationID, &membership.LocationID, &membership.MemberRef, &membership.Role, &membership.CreatedAt, &membership.UpdatedAt, &membership.UserProfileID, &membership.DisabledAt); err != nil {
				return err
			}
			locMemberships = append(locMemberships, membership)
		}
		return locationRows.Err()
	})
	if err != nil {
		return nil, nil, err
	}
	return orgMemberships, locMemberships, nil
}

func organisationFromDB(org db.Organisation) organisationDTO {
	return organisationDTO{ID: org.ID.String(), Slug: org.Slug, Name: org.Name, Status: org.Status}
}

func locationFromDB(location db.Location) locationDTO {
	return locationDTO{
		ID:              location.ID.String(),
		OrganisationID:  location.OrganisationID.String(),
		Slug:            location.Slug,
		Name:            location.Name,
		Timezone:        location.Timezone,
		Status:          location.Status,
		OperatingConfig: json.RawMessage(location.OperatingConfig),
		FeatureFlags:    json.RawMessage(location.FeatureFlags),
	}
}

func floorFromDB(floor db.Floor) floorDTO {
	return floorDTO{
		ID:                 floor.ID.String(),
		Slug:               floor.Slug,
		Name:               floor.Name,
		SortOrder:          floor.SortOrder,
		Canvas:             json.RawMessage(floor.Canvas),
		BackgroundAssetRef: textPtr(floor.BackgroundAssetRef),
		IsActive:           floor.IsActive,
		Version:            floor.Version,
	}
}

func zoneFromDB(zone db.Zone) zoneDTO {
	return zoneDTO{
		ID:        zone.ID.String(),
		FloorID:   zone.FloorID.String(),
		Name:      zone.Name,
		SortOrder: zone.SortOrder,
		IsActive:  zone.IsActive,
		Version:   zone.Version,
	}
}

func tableFromDB(table db.Table) tableDTO {
	return tableDTO{
		ID:            table.ID.String(),
		FloorID:       table.FloorID.String(),
		ZoneID:        table.ZoneID.String(),
		Label:         table.Label,
		CapacityLabel: table.CapacityLabel,
		Shape:         table.Shape,
		Geometry:      json.RawMessage(table.Geometry),
		IsActive:      table.IsActive && !table.DeletedAt.Valid,
		Version:       table.Version,
	}
}

func tableStateFromDB(table db.Table, occupancy db.TableOccupancy) tableStateDTO {
	return tableStateDTO{Table: tableFromDB(table), Occupancy: occupancyFromDB(occupancy)}
}

func tableCommandResponseFromDB(table db.Table, occupancy db.TableOccupancy, session db.TableSession) tableCommandResponse {
	return tableCommandResponse{Table: tableFromDB(table), Occupancy: occupancyFromDB(occupancy), Session: sessionFromDB(session)}
}

func occupancyFromDB(occupancy db.TableOccupancy) occupancyDTO {
	return occupancyDTO{Status: occupancy.Status, CurrentSessionID: uuidPtr(occupancy.CurrentSessionID), Version: occupancy.Version, UpdatedAt: timeString(occupancy.UpdatedAt)}
}

func sessionFromDB(session db.TableSession) tableSessionDTO {
	return tableSessionDTO{ID: session.ID.String(), TableID: session.TableID.String(), StartedAt: timeString(session.StartedAt), EndedAt: timePtr(session.EndedAt), PartySize: intPtr(session.PartySize), Source: session.Source}
}

func assistFromDB(assist db.AssistRequest) assistDTO {
	return assistDTO{ID: assist.ID.String(), TableID: assist.TableID.String(), TableSessionID: uuidPtr(assist.TableSessionID), Status: assist.Status, RequestedAt: timeString(assist.RequestedAt), Version: assist.Version, Note: textPtr(assist.Note), ActionKey: textPtr(assist.ActionKey)}
}

func deviceFromDB(device db.Device) deviceDTO {
	return deviceDTO{
		ID:                       device.ID.String(),
		OrganisationID:           device.OrganisationID.String(),
		LocationID:               uuidPtr(device.LocationID),
		Name:                     textPtr(device.Name),
		DeviceType:               device.DeviceType,
		Platform:                 device.Platform,
		AppVersion:               device.AppVersion,
		TrustState:               device.TrustState,
		RegisteredAt:             timeString(device.RegisteredAt),
		LastSeenAt:               timePtr(device.LastSeenAt),
		LastHeartbeatAt:          timePtr(device.LastHeartbeatAt),
		AssignedAt:               timePtr(device.AssignedAt),
		RevokedAt:                timePtr(device.RevokedAt),
		HeartbeatIntervalSeconds: device.HeartbeatIntervalSeconds,
		Capabilities:             json.RawMessage(device.Capabilities),
		Configuration:            json.RawMessage(device.Configuration),
	}
}

func pairingCodeFromDB(pairingCode db.DevicePairingCode, code string) pairingCodeDTO {
	return pairingCodeDTO{
		ID:         pairingCode.ID.String(),
		LocationID: pairingCode.LocationID.String(),
		DeviceType: pairingCode.DeviceType,
		Code:       code,
		ExpiresAt:  timeString(pairingCode.ExpiresAt),
		ConsumedAt: timePtr(pairingCode.ConsumedAt),
	}
}

func organisationMembershipFromDB(membership db.OrganisationMembership) membershipDTO {
	return membershipDTO{
		ID:             membership.ID.String(),
		Scope:          "organisation",
		OrganisationID: membership.OrganisationID.String(),
		MemberRef:      membership.MemberRef,
		Role:           membership.Role,
	}
}

func locationMembershipFromDB(membership db.LocationMembership) membershipDTO {
	locationID := membership.LocationID.String()
	return membershipDTO{
		ID:             membership.ID.String(),
		Scope:          "location",
		OrganisationID: membership.OrganisationID.String(),
		LocationID:     &locationID,
		MemberRef:      membership.MemberRef,
		Role:           membership.Role,
	}
}

func analyticsSummaryFromDomain(summary analytics.Summary) analyticsSummaryDTO {
	var current *analyticsWindowMetricDTO
	if summary.CurrentServicePeriod != nil {
		dto := analyticsWindowMetricFromDomain(*summary.CurrentServicePeriod)
		current = &dto
	}
	var checkpoint *analyticsCheckpointDTO
	if summary.Checkpoint != nil {
		dto := analyticsCheckpointFromDB(*summary.Checkpoint)
		checkpoint = &dto
	}
	return analyticsSummaryDTO{
		Location:             locationFromDB(summary.Location),
		Date:                 summary.Date.Format("2006-01-02"),
		Today:                analyticsWindowMetricFromDomain(summary.Today),
		CurrentServicePeriod: current,
		Checkpoint:           checkpoint,
		DataQuality:          analyticsDataQualityFromDomain(summary.DataQuality),
	}
}

func analyticsWindowMetricFromDomain(window analytics.WindowMetric) analyticsWindowMetricDTO {
	id := ""
	if window.ID != uuid.Nil {
		id = window.ID.String()
	}
	return analyticsWindowMetricDTO{
		ID:          id,
		Name:        window.Name,
		WindowStart: window.WindowStart.UTC().Format(time.RFC3339Nano),
		WindowEnd:   window.WindowEnd.UTC().Format(time.RFC3339Nano),
		Metric:      analyticsMetricFromDomain(window.Metric),
	}
}

func analyticsMetricFromDomain(metric analytics.Metric) analyticsMetricDTO {
	return analyticsMetricDTO{
		ActiveTableCount:           metric.ActiveTableCount,
		OccupancySeconds:           metric.OccupancySeconds,
		UtilisationBasisSeconds:    metric.UtilisationBasisSeconds,
		UtilisationRate:            metric.UtilisationRate,
		CompletedSessionCount:      metric.CompletedSessionCount,
		TurnoverRate:               metric.TurnoverRate,
		AvgSessionSeconds:          metric.AvgSessionSeconds,
		P50SessionSeconds:          metric.P50SessionSeconds,
		P90SessionSeconds:          metric.P90SessionSeconds,
		AssistRequestCount:         metric.AssistRequestCount,
		AvgAssistResponseSeconds:   metric.AvgAssistResponseSeconds,
		AvgAssistResolutionSeconds: metric.AvgAssistResolutionSeconds,
		AnomalyCount:               metric.AnomalyCount,
	}
}

func analyticsTimePointFromDomain(point analytics.TimePoint) analyticsTimePointDTO {
	return analyticsTimePointDTO{
		BucketStart: point.BucketStart.UTC().Format(time.RFC3339Nano),
		BucketEnd:   point.BucketEnd.UTC().Format(time.RFC3339Nano),
		Metric:      analyticsMetricFromDomain(point.Metric),
	}
}

func analyticsComparisonPointFromDomain(point analytics.ComparisonPoint) analyticsComparisonPointDTO {
	return analyticsComparisonPointDTO{
		GroupID:     point.GroupID.String(),
		GroupName:   point.GroupName,
		MetricDate:  point.MetricDate.Format("2006-01-02"),
		WindowStart: point.WindowStart.UTC().Format(time.RFC3339Nano),
		WindowEnd:   point.WindowEnd.UTC().Format(time.RFC3339Nano),
		Metric:      analyticsMetricFromDomain(point.Metric),
	}
}

func analyticsDataQualityFromDomain(quality analytics.DataQuality) analyticsDataQualityDTO {
	return analyticsDataQualityDTO{
		OpenSessionCount:       quality.OpenSessionCount,
		LongOpenSessionCount:   quality.LongOpenSessionCount,
		ImpossibleSessionCount: quality.ImpossibleSessionCount,
		MissingOccupancyCount:  quality.MissingOccupancyCount,
		StaleDeviceCount:       quality.StaleDeviceCount,
		ProjectorLagSeconds:    quality.ProjectorLagSeconds,
		RebuildStatus:          quality.RebuildStatus,
	}
}

func analyticsCheckpointFromDB(checkpoint db.AnalyticsProjectorCheckpoint) analyticsCheckpointDTO {
	return analyticsCheckpointDTO{
		ProjectorName:    checkpoint.ProjectorName,
		ProjectorVersion: checkpoint.ProjectorVersion,
		CursorOccurredAt: timePtr(checkpoint.CursorOccurredAt),
		CursorEventID:    uuidPtr(checkpoint.CursorEventID),
		LagSeconds:       checkpoint.LagSeconds,
		RebuildStatus:    checkpoint.RebuildStatus,
		UpdatedAt:        timeString(checkpoint.UpdatedAt),
	}
}

func servicePeriodFromDB(period db.ServicePeriod) servicePeriodDTO {
	return servicePeriodDTO{
		ID:         period.ID.String(),
		LocationID: period.LocationID.String(),
		Name:       period.Name,
		DaysOfWeek: period.DaysOfWeek,
		StartTime:  analytics.FormatClock(period.StartTime),
		EndTime:    analytics.FormatClock(period.EndTime),
		IsActive:   period.IsActive,
		Version:    period.Version,
	}
}

func uuidPtr(value uuid.NullUUID) *string {
	if !value.Valid {
		return nil
	}
	out := value.UUID.String()
	return &out
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func intPtr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func int4Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func timePtr(value pgtype.Timestamptz) *string {
	if !value.Valid {
		return nil
	}
	out := timeString(value)
	return &out
}

func timeString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339Nano)
}
