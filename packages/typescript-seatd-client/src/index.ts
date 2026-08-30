// Generated from packages/api-contract/openapi/seatd.v1.json. Do not edit by hand.

export const healthEndpoint = "/healthz";
export const statusEndpoint = "/status";
export const versionEndpoint = "/version";
export const organisationEndpoint = "/v1/organisations/{id}";
export const ownerSnapshotEndpoint = "/v1/owner/snapshot";
export const locationEndpoint = "/v1/locations/{id}";
export const floorsEndpoint = "/v1/floors";
export const floorZonesEndpoint = "/v1/floors/{id}/zones";
export const floorTablesEndpoint = "/v1/floors/{id}/tables";
export const layoutEditorSnapshotEndpoint = "/v1/layout/editor-snapshot";
export const zonesEndpoint = "/v1/zones";
export const zoneEndpoint = "/v1/zones/{id}";
export const tableEndpoint = "/v1/tables/{id}";
export const tableQRCapabilitiesEndpoint = "/v1/tables/{id}/qr-capabilities";
export const tableQRCapabilityRotateEndpoint = "/v1/tables/{id}/qr-capabilities/{capabilityId}/rotate";
export const tableQRCapabilityRevokeEndpoint = "/v1/tables/{id}/qr-capabilities/{capabilityId}/revoke";
export const locationStateEndpoint = "/v1/location-state";
export const assistsEndpoint = "/v1/assists";
export const guestQREndpoint = "/v1/guest/qr/{token}";
export const guestRequestsEndpoint = "/v1/guest/qr/{token}/requests";
export const guestRequestEndpoint = "/v1/guest/qr/{token}/requests/{id}";
export const guestRequestCancelEndpoint = "/v1/guest/qr/{token}/requests/{id}/cancel";
export const locationSnapshotEndpoint = "/v1/sync/location-snapshot";
export const syncEventsEndpoint = "/v1/sync/events";
export const realtimeEndpoint = "/v1/realtime";
export const auditTimelineEndpoint = "/v1/audit/timeline";
export const analyticsSummaryEndpoint = "/v1/analytics/summary";
export const analyticsTimeseriesEndpoint = "/v1/analytics/timeseries";
export const analyticsComparisonEndpoint = "/v1/analytics/comparison";
export const analyticsDataQualityEndpoint = "/v1/analytics/data-quality";
export const analyticsRebuildEndpoint = "/v1/analytics/rebuild";
export const servicePeriodsEndpoint = "/v1/service-periods";
export const servicePeriodEndpoint = "/v1/service-periods/{id}";
export const servicePeriodArchiveEndpoint = "/v1/service-periods/{id}/archive";
export const devicesEndpoint = "/v1/devices";
export const devicePairingCodesEndpoint = "/v1/devices/pairing-codes";
export const pairDeviceEndpoint = "/v1/devices/pair";
export const deviceHeartbeatEndpoint = "/v1/devices/heartbeat";
export const displaySnapshotEndpoint = "/v1/devices/display-snapshot";
export const deviceRevokeEndpoint = "/v1/devices/{id}/revoke";
export const membershipsEndpoint = "/v1/memberships";
export const occupyTableEndpoint = "/v1/tables/{id}/occupy";
export const clearTableEndpoint = "/v1/tables/{id}/clear";
export const acknowledgeAssistEndpoint = "/v1/assists/{id}/acknowledge";
export const resolveAssistEndpoint = "/v1/assists/{id}/resolve";
export const cancelAssistEndpoint = "/v1/assists/{id}/cancel";


export type Environment = "local" | "test" | "staging" | "production";
export type ErrorCode =
  | "validation_failed"
  | "unauthorized"
  | "forbidden"
  | "tenant_disabled"
  | "not_found"
  | "version_conflict"
  | "already_occupied"
  | "already_available"
  | "no_active_session"
  | "assist_already_resolved"
  | "idempotency_conflict"
  | "action_not_enabled"
  | "rate_limited"
  | "internal_error";

export interface BuildInfo {
  version: string;
  commit: string;
  date: string;
}

export interface ServiceStatus {
  service: string;
  environment: Environment;
  status: "ok";
  version: BuildInfo;
}

export interface SeatdErrorResponse {
  error: {
    code: ErrorCode;
    message: string;
    details?: Record<string, string>;
  };
}

export interface CommandRequest {
  commandId: string;
  expectedVersion: number;
}

export interface TableCommandRequest extends CommandRequest {
  partySize?: number;
}

export interface Organisation {
  id: string;
  slug: string;
  name: string;
  status: "active" | "disabled";
}

export interface Location {
  id: string;
  organisationId: string;
  slug: string;
  name: string;
  timezone: string;
  status: "active" | "disabled";
  operatingConfig: Record<string, unknown>;
  featureFlags: Record<string, unknown>;
}

export interface Floor {
  id: string;
  slug: string;
  name: string;
  sortOrder: number;
  canvas: Record<string, unknown>;
  backgroundAssetRef?: string;
  isActive: boolean;
  version: number;
}

export interface Zone {
  id: string;
  floorId: string;
  name: string;
  sortOrder: number;
  isActive: boolean;
  version: number;
}

export interface Table {
  id: string;
  floorId: string;
  zoneId: string;
  label: string;
  capacityLabel: string;
  shape: "rectangle" | "circle" | "square" | "custom";
  geometry: Record<string, unknown>;
  isActive: boolean;
  version: number;
}

export interface Occupancy {
  status: "available" | "occupied";
  currentSessionId?: string;
  version: number;
  updatedAt: string;
}

export interface TableState {
  table: Table;
  occupancy: Occupancy;
}

export interface TableSession {
  id: string;
  tableId: string;
  startedAt: string;
  endedAt?: string;
  partySize?: number;
  source: string;
}

export interface TableCommandResponse extends TableState {
  session: TableSession;
}

export interface Assist {
  id: string;
  tableId: string;
  tableSessionId?: string;
  status: "pending" | "acknowledged" | "resolved" | "cancelled";
  requestedAt: string;
  version: number;
  note?: string;
  actionKey?: string;
}

export interface GuestAction {
  key: string;
  label: string;
}

export interface GuestContext {
  locationName: string;
  tableLabel: string;
  occupancy: Occupancy;
  actions: GuestAction[];
  activeRequest?: Assist;
}

export interface GuestCreateRequest {
  commandId: string;
  actionKey: string;
}

export interface GuestCancelRequest {
  commandId: string;
}

export interface TableQRCapability {
  id: string;
  tableId: string;
  lookupPrefix: string;
  label?: string;
  issuedAt: string;
  expiresAt?: string;
  revokedAt?: string;
  lastUsedAt?: string;
  version: number;
}

export interface TableQRCapabilitiesResponse {
  qrCapabilities: TableQRCapability[];
}

export interface QRExport extends TableQRCapability {
  token: string;
  publicUrl: string;
}

export interface ExportQRRequest {
  label?: string;
  expiresAt?: string;
}

export interface Device {
  id: string;
  organisationId: string;
  locationId?: string;
  name?: string;
  deviceType: string;
  platform: string;
  appVersion: string;
  trustState: "pending" | "trusted" | "revoked";
  registeredAt: string;
  lastSeenAt?: string;
  lastHeartbeatAt?: string;
  assignedAt?: string;
  revokedAt?: string;
  heartbeatIntervalSeconds: number;
  capabilities: Record<string, unknown>;
  configuration: Record<string, unknown>;
}

export interface CreatePairingCodeRequest {
  locationId: string;
  deviceType?: string;
  ttlSeconds?: number;
}

export interface PairingCode {
  id: string;
  locationId: string;
  deviceType: string;
  code: string;
  expiresAt: string;
  consumedAt?: string;
}

export interface PairingCodeResponse {
  pairingCode: PairingCode;
}

export interface PairDeviceRequest {
  code: string;
  platform: string;
  appVersion: string;
  name?: string;
  capabilities?: Record<string, unknown>;
  configuration?: Record<string, unknown>;
}

export interface PairDeviceResponse {
  device: Device;
  credential: string;
}

export interface DeviceHeartbeatRequest {
  appVersion: string;
  capabilities?: Record<string, unknown>;
}

export interface Membership {
  id: string;
  scope: "organisation" | "location";
  locationId?: string;
  memberRef: string;
  role: string;
}

export interface TimelineEvent {
  id: string;
  type: string;
  schemaVersion: number;
  occurredAt: string;
  actorRef: string;
  deviceId?: string;
  entityType: string;
  entityId: string;
  entityVersion?: number;
  commandId?: string;
  data: Record<string, unknown>;
}

export interface RealtimeMessage {
  type: string;
  eventId: string;
  organisationId: string;
  locationId: string;
  entityType: string;
  entityId: string;
  version?: number;
  occurredAt: string;
  data: Record<string, unknown>;
}

export interface AuditTimelineResponse {
  events: TimelineEvent[];
}

export interface AnalyticsMetric {
  activeTableCount: number;
  occupancySeconds: number;
  utilisationBasisSeconds: number;
  utilisationRate: number;
  completedSessionCount: number;
  turnoverRate: number;
  avgSessionSeconds: number;
  p50SessionSeconds: number;
  p90SessionSeconds: number;
  assistRequestCount: number;
  avgAssistResponseSeconds: number;
  avgAssistResolutionSeconds: number;
  anomalyCount: number;
}

export interface AnalyticsWindowMetric {
  id?: string;
  name?: string;
  windowStart: string;
  windowEnd: string;
  metric: AnalyticsMetric;
}

export interface AnalyticsDataQuality {
  openSessionCount: number;
  longOpenSessionCount: number;
  impossibleSessionCount: number;
  missingOccupancyCount: number;
  staleDeviceCount: number;
  projectorLagSeconds: number;
  rebuildStatus: string;
}

export interface AnalyticsCheckpoint {
  projectorName: string;
  projectorVersion: number;
  cursorOccurredAt?: string;
  cursorEventId?: string;
  lagSeconds: number;
  rebuildStatus: string;
  updatedAt: string;
}

export interface AnalyticsSummary {
  location: Location;
  date: string;
  today: AnalyticsWindowMetric;
  currentServicePeriod?: AnalyticsWindowMetric;
  checkpoint?: AnalyticsCheckpoint;
  dataQuality: AnalyticsDataQuality;
}

export interface AnalyticsTimePoint {
  bucketStart: string;
  bucketEnd: string;
  metric: AnalyticsMetric;
}

export interface AnalyticsTimeseriesResponse {
  points: AnalyticsTimePoint[];
}

export interface AnalyticsComparisonPoint {
  groupId: string;
  groupName: string;
  metricDate: string;
  windowStart: string;
  windowEnd: string;
  metric: AnalyticsMetric;
}

export interface AnalyticsComparisonResponse {
  points: AnalyticsComparisonPoint[];
}

export interface ServicePeriod {
  id: string;
  locationId: string;
  name: string;
  daysOfWeek: number[];
  startTime: string;
  endTime: string;
  isActive: boolean;
  version: number;
}

export interface ServicePeriodsResponse {
  servicePeriods: ServicePeriod[];
}

export interface ServicePeriodRequest {
  name: string;
  daysOfWeek: number[];
  startTime: string;
  endTime: string;
  expectedVersion?: number;
}

export interface AnalyticsRebuildRequest {
  locationId: string;
  from: string;
  to: string;
}

export interface LocationSnapshotResponse {
  organisationId: string;
  locationId: string;
  cursor: string;
  tables: TableState[];
  assists: Assist[];
}

export interface OwnerSnapshotResponse {
  organisation: Organisation;
  locations: Location[];
}

export interface LayoutEditorSnapshotResponse {
  floors: Floor[];
  zones: Zone[];
  tables: TableState[];
}

export interface DevicesResponse {
  devices: Device[];
}

export interface MembershipsResponse {
  memberships: Membership[];
}

export interface UpdateOrganisationRequest {
  slug: string;
  name: string;
  status: "active" | "disabled";
}

export interface UpdateLocationRequest {
  name: string;
  timezone: string;
  status: "active" | "disabled";
  operatingConfig: Record<string, unknown>;
  featureFlags: Record<string, unknown>;
}

export interface FloorRequest {
  slug: string;
  name: string;
  sortOrder: number;
  canvas: Record<string, unknown>;
  backgroundAssetRef?: string;
  expectedVersion?: number;
}

export interface ZoneRequest {
  floorId: string;
  name: string;
  sortOrder: number;
  expectedVersion?: number;
}

export interface TableRequest {
  floorId: string;
  zoneId: string;
  label: string;
  capacityLabel: string;
  shape: Table["shape"];
  geometry: Record<string, unknown>;
  expectedVersion?: number;
}

export interface ArchiveRequest {
  expectedVersion: number;
  floorId?: string;
}

export interface SyncEventsResponse {
  events: RealtimeMessage[];
  cursor: string;
}

export async function getStatus(baseUrl: string, init?: RequestInit): Promise<ServiceStatus> {
  const response = await fetch(new URL(statusEndpoint, baseUrl), init);
  if (!response.ok) {
    throw new Error(`status request failed: ${response.status}`);
  }
  return response.json() as Promise<ServiceStatus>;
}
