// Generated from packages/api-contract/openapi/seatd.v1.json. Do not edit by hand.

import 'dart:convert';

const healthEndpoint = '/healthz';
const statusEndpoint = '/status';
const versionEndpoint = '/version';
const organisationEndpoint = '/v1/organisations/{id}';
const locationEndpoint = '/v1/locations/{id}';
const floorsEndpoint = '/v1/floors';
const floorZonesEndpoint = '/v1/floors/{id}/zones';
const floorTablesEndpoint = '/v1/floors/{id}/tables';
const tableEndpoint = '/v1/tables/{id}';
const locationStateEndpoint = '/v1/location-state';
const assistsEndpoint = '/v1/assists';
const locationSnapshotEndpoint = '/v1/sync/location-snapshot';
const syncEventsEndpoint = '/v1/sync/events';
const realtimeEndpoint = '/v1/realtime';
const auditTimelineEndpoint = '/v1/audit/timeline';
const analyticsSummaryEndpoint = '/v1/analytics/summary';
const analyticsTimeseriesEndpoint = '/v1/analytics/timeseries';
const analyticsComparisonEndpoint = '/v1/analytics/comparison';
const analyticsDataQualityEndpoint = '/v1/analytics/data-quality';
const analyticsRebuildEndpoint = '/v1/analytics/rebuild';
const servicePeriodsEndpoint = '/v1/service-periods';
const servicePeriodEndpoint = '/v1/service-periods/{id}';
const servicePeriodArchiveEndpoint = '/v1/service-periods/{id}/archive';
const devicesEndpoint = '/v1/devices';
const devicePairingCodesEndpoint = '/v1/devices/pairing-codes';
const pairDeviceEndpoint = '/v1/devices/pair';
const deviceHeartbeatEndpoint = '/v1/devices/heartbeat';
const displaySnapshotEndpoint = '/v1/devices/display-snapshot';
const deviceRevokeEndpoint = '/v1/devices/{id}/revoke';
const integrationsEndpoint = '/v1/integrations';
const integrationHealthEndpoint = '/v1/integrations/{id}/health';
const integrationMappingsEndpoint = '/v1/integrations/{id}/mappings';
const integrationMappingEndpoint = '/v1/integrations/{id}/mappings/{mappingId}';
const integrationWebhooksEndpoint = '/v1/integrations/{id}/webhooks';
const integrationWebhookReplayEndpoint = '/v1/integrations/{id}/webhooks/{webhookId}/replay';
const integrationDiscrepanciesEndpoint = '/v1/integrations/{id}/discrepancies';
const integrationReconcileEndpoint = '/v1/integrations/{id}/reconcile';
const integrationWebhookEndpoint = '/v1/integrations/{vendor}/webhooks';
const membershipsEndpoint = '/v1/memberships';
const occupyTableEndpoint = '/v1/tables/{id}/occupy';
const clearTableEndpoint = '/v1/tables/{id}/clear';
const acknowledgeAssistEndpoint = '/v1/assists/{id}/acknowledge';
const resolveAssistEndpoint = '/v1/assists/{id}/resolve';
const cancelAssistEndpoint = '/v1/assists/{id}/cancel';


enum SeatdEnvironment { local, test, staging, production }
enum SeatdErrorCode {
  validationFailed,
  unauthorized,
  forbidden,
  tenantDisabled,
  notFound,
  versionConflict,
  alreadyOccupied,
  alreadyAvailable,
  noActiveSession,
  assistAlreadyResolved,
  idempotencyConflict,
  rateLimited,
  internalError,
}

SeatdErrorCode seatdErrorCodeFromWire(String value) => switch (value) {
      'validation_failed' => SeatdErrorCode.validationFailed,
      'unauthorized' => SeatdErrorCode.unauthorized,
      'forbidden' => SeatdErrorCode.forbidden,
      'tenant_disabled' => SeatdErrorCode.tenantDisabled,
      'not_found' => SeatdErrorCode.notFound,
      'version_conflict' => SeatdErrorCode.versionConflict,
      'already_occupied' => SeatdErrorCode.alreadyOccupied,
      'already_available' => SeatdErrorCode.alreadyAvailable,
      'no_active_session' => SeatdErrorCode.noActiveSession,
      'assist_already_resolved' => SeatdErrorCode.assistAlreadyResolved,
      'idempotency_conflict' => SeatdErrorCode.idempotencyConflict,
      'rate_limited' => SeatdErrorCode.rateLimited,
      _ => SeatdErrorCode.internalError,
    };

class BuildInfo {
  const BuildInfo({required this.version, required this.commit, required this.date});

  factory BuildInfo.fromJson(Map<String, Object?> json) => BuildInfo(
        version: json['version'] as String,
        commit: json['commit'] as String,
        date: json['date'] as String,
      );

  final String version;
  final String commit;
  final String date;
}

class ServiceStatus {
  const ServiceStatus({required this.service, required this.environment, required this.status, required this.version});

  factory ServiceStatus.fromJson(Map<String, Object?> json) => ServiceStatus(
        service: json['service'] as String,
        environment: SeatdEnvironment.values.byName(json['environment'] as String),
        status: json['status'] as String,
        version: BuildInfo.fromJson(json['version'] as Map<String, Object?>),
      );

  factory ServiceStatus.fromJsonString(String source) => ServiceStatus.fromJson(jsonDecode(source) as Map<String, Object?>);

  final String service;
  final SeatdEnvironment environment;
  final String status;
  final BuildInfo version;
}

class SeatdApiError {
  const SeatdApiError({required this.code, required this.message, this.details = const {}});

  factory SeatdApiError.fromJson(Map<String, Object?> json) => SeatdApiError(
        code: seatdErrorCodeFromWire(json['code'] as String),
        message: json['message'] as String,
        details: (json['details'] as Map<String, Object?>?)?.map((key, value) => MapEntry(key, value as String)) ?? const {},
      );

  final SeatdErrorCode code;
  final String message;
  final Map<String, String> details;
}

class SeatdErrorResponse {
  const SeatdErrorResponse({required this.error});

  factory SeatdErrorResponse.fromJson(Map<String, Object?> json) =>
      SeatdErrorResponse(error: SeatdApiError.fromJson(json['error'] as Map<String, Object?>));

  factory SeatdErrorResponse.fromJsonString(String source) => SeatdErrorResponse.fromJson(jsonDecode(source) as Map<String, Object?>);

  final SeatdApiError error;
}

class CommandRequest {
  const CommandRequest({required this.commandId, required this.expectedVersion});

  final String commandId;
  final int expectedVersion;

  Map<String, Object?> toJson() => {'commandId': commandId, 'expectedVersion': expectedVersion};
}

class TableCommandRequest extends CommandRequest {
  const TableCommandRequest({required super.commandId, required super.expectedVersion, this.partySize});

  final int? partySize;

  @override
  Map<String, Object?> toJson() => {...super.toJson(), if (partySize != null) 'partySize': partySize};
}

class Organisation {
  const Organisation({required this.id, required this.slug, required this.name, required this.status});

  factory Organisation.fromJson(Map<String, Object?> json) => Organisation(
        id: json['id'] as String,
        slug: json['slug'] as String,
        name: json['name'] as String,
        status: json['status'] as String,
      );

  final String id;
  final String slug;
  final String name;
  final String status;
}

class Location {
  const Location({required this.id, required this.organisationId, required this.slug, required this.name, required this.timezone, required this.status});

  factory Location.fromJson(Map<String, Object?> json) => Location(
        id: json['id'] as String,
        organisationId: json['organisationId'] as String,
        slug: json['slug'] as String,
        name: json['name'] as String,
        timezone: json['timezone'] as String,
        status: json['status'] as String,
      );

  final String id;
  final String organisationId;
  final String slug;
  final String name;
  final String timezone;
  final String status;
}

class Floor {
  const Floor({required this.id, required this.slug, required this.name, required this.sortOrder, required this.canvas, required this.version});

  factory Floor.fromJson(Map<String, Object?> json) => Floor(
        id: json['id'] as String,
        slug: json['slug'] as String,
        name: json['name'] as String,
        sortOrder: json['sortOrder'] as int,
        canvas: json['canvas'] as Map<String, Object?>,
        version: json['version'] as int,
      );

  final String id;
  final String slug;
  final String name;
  final int sortOrder;
  final Map<String, Object?> canvas;
  final int version;
}

class Zone {
  const Zone({required this.id, required this.floorId, required this.name, required this.sortOrder, required this.version});

  factory Zone.fromJson(Map<String, Object?> json) => Zone(
        id: json['id'] as String,
        floorId: json['floorId'] as String,
        name: json['name'] as String,
        sortOrder: json['sortOrder'] as int,
        version: json['version'] as int,
      );

  final String id;
  final String floorId;
  final String name;
  final int sortOrder;
  final int version;
}

class SeatdTable {
  const SeatdTable({required this.id, required this.floorId, required this.zoneId, required this.label, required this.capacityLabel, required this.shape, required this.geometry, required this.version});

  factory SeatdTable.fromJson(Map<String, Object?> json) => SeatdTable(
        id: json['id'] as String,
        floorId: json['floorId'] as String,
        zoneId: json['zoneId'] as String,
        label: json['label'] as String,
        capacityLabel: json['capacityLabel'] as String,
        shape: json['shape'] as String,
        geometry: json['geometry'] as Map<String, Object?>,
        version: json['version'] as int,
      );

  final String id;
  final String floorId;
  final String zoneId;
  final String label;
  final String capacityLabel;
  final String shape;
  final Map<String, Object?> geometry;
  final int version;
}

class Occupancy {
  const Occupancy({required this.status, this.currentSessionId, required this.version, required this.updatedAt});

  factory Occupancy.fromJson(Map<String, Object?> json) => Occupancy(
        status: json['status'] as String,
        currentSessionId: json['currentSessionId'] as String?,
        version: json['version'] as int,
        updatedAt: json['updatedAt'] as String,
      );

  final String status;
  final String? currentSessionId;
  final int version;
  final String updatedAt;
}

class TableState {
  const TableState({required this.table, required this.occupancy});

  factory TableState.fromJson(Map<String, Object?> json) => TableState(
        table: SeatdTable.fromJson(json['table'] as Map<String, Object?>),
        occupancy: Occupancy.fromJson(json['occupancy'] as Map<String, Object?>),
      );

  final SeatdTable table;
  final Occupancy occupancy;
}

class TableSession {
  const TableSession({required this.id, required this.tableId, required this.startedAt, this.endedAt, this.partySize, required this.source});

  factory TableSession.fromJson(Map<String, Object?> json) => TableSession(
        id: json['id'] as String,
        tableId: json['tableId'] as String,
        startedAt: json['startedAt'] as String,
        endedAt: json['endedAt'] as String?,
        partySize: json['partySize'] as int?,
        source: json['source'] as String,
      );

  final String id;
  final String tableId;
  final String startedAt;
  final String? endedAt;
  final int? partySize;
  final String source;
}

class TableCommandResponse extends TableState {
  const TableCommandResponse({required super.table, required super.occupancy, required this.session});

  factory TableCommandResponse.fromJson(Map<String, Object?> json) => TableCommandResponse(
        table: SeatdTable.fromJson(json['table'] as Map<String, Object?>),
        occupancy: Occupancy.fromJson(json['occupancy'] as Map<String, Object?>),
        session: TableSession.fromJson(json['session'] as Map<String, Object?>),
      );

  final TableSession session;
}

class Assist {
  const Assist({required this.id, required this.tableId, this.tableSessionId, required this.status, required this.requestedAt, required this.version, this.note});

  factory Assist.fromJson(Map<String, Object?> json) => Assist(
        id: json['id'] as String,
        tableId: json['tableId'] as String,
        tableSessionId: json['tableSessionId'] as String?,
        status: json['status'] as String,
        requestedAt: json['requestedAt'] as String,
        version: json['version'] as int,
        note: json['note'] as String?,
      );

  final String id;
  final String tableId;
  final String? tableSessionId;
  final String status;
  final String requestedAt;
  final int version;
  final String? note;
}

class Device {
  const Device({
    required this.id,
    required this.organisationId,
    this.locationId,
    this.name,
    required this.deviceType,
    required this.platform,
    required this.appVersion,
    required this.trustState,
    required this.registeredAt,
    this.lastSeenAt,
    this.lastHeartbeatAt,
    this.assignedAt,
    this.revokedAt,
    required this.heartbeatIntervalSeconds,
    this.capabilities = const {},
    this.configuration = const {},
  });

  factory Device.fromJson(Map<String, Object?> json) => Device(
        id: json['id'] as String,
        organisationId: json['organisationId'] as String,
        locationId: json['locationId'] as String?,
        name: json['name'] as String?,
        deviceType: json['deviceType'] as String,
        platform: json['platform'] as String,
        appVersion: json['appVersion'] as String,
        trustState: json['trustState'] as String,
        registeredAt: json['registeredAt'] as String,
        lastSeenAt: json['lastSeenAt'] as String?,
        lastHeartbeatAt: json['lastHeartbeatAt'] as String?,
        assignedAt: json['assignedAt'] as String?,
        revokedAt: json['revokedAt'] as String?,
        heartbeatIntervalSeconds: json['heartbeatIntervalSeconds'] as int,
        capabilities: json['capabilities'] as Map<String, Object?>? ?? const {},
        configuration: json['configuration'] as Map<String, Object?>? ?? const {},
      );

  final String id;
  final String organisationId;
  final String? locationId;
  final String? name;
  final String deviceType;
  final String platform;
  final String appVersion;
  final String trustState;
  final String registeredAt;
  final String? lastSeenAt;
  final String? lastHeartbeatAt;
  final String? assignedAt;
  final String? revokedAt;
  final int heartbeatIntervalSeconds;
  final Map<String, Object?> capabilities;
  final Map<String, Object?> configuration;
}

class CreatePairingCodeRequest {
  const CreatePairingCodeRequest({required this.locationId, this.deviceType = 'display', this.ttlSeconds});

  final String locationId;
  final String deviceType;
  final int? ttlSeconds;

  Map<String, Object?> toJson() => {
        'locationId': locationId,
        'deviceType': deviceType,
        if (ttlSeconds != null) 'ttlSeconds': ttlSeconds,
      };
}

class PairingCode {
  const PairingCode({required this.id, required this.locationId, required this.deviceType, required this.code, required this.expiresAt, this.consumedAt});

  factory PairingCode.fromJson(Map<String, Object?> json) => PairingCode(
        id: json['id'] as String,
        locationId: json['locationId'] as String,
        deviceType: json['deviceType'] as String,
        code: json['code'] as String,
        expiresAt: json['expiresAt'] as String,
        consumedAt: json['consumedAt'] as String?,
      );

  final String id;
  final String locationId;
  final String deviceType;
  final String code;
  final String expiresAt;
  final String? consumedAt;
}

class PairingCodeResponse {
  const PairingCodeResponse({required this.pairingCode});

  factory PairingCodeResponse.fromJson(Map<String, Object?> json) =>
      PairingCodeResponse(pairingCode: PairingCode.fromJson(json['pairingCode'] as Map<String, Object?>));

  final PairingCode pairingCode;
}

class PairDeviceRequest {
  const PairDeviceRequest({required this.code, required this.platform, required this.appVersion, this.name, this.capabilities = const {}, this.configuration = const {}});

  final String code;
  final String platform;
  final String appVersion;
  final String? name;
  final Map<String, Object?> capabilities;
  final Map<String, Object?> configuration;

  Map<String, Object?> toJson() => {
        'code': code,
        'platform': platform,
        'appVersion': appVersion,
        if (name != null) 'name': name,
        'capabilities': capabilities,
        'configuration': configuration,
      };
}

class PairDeviceResponse {
  const PairDeviceResponse({required this.device, required this.credential});

  factory PairDeviceResponse.fromJson(Map<String, Object?> json) => PairDeviceResponse(
        device: Device.fromJson(json['device'] as Map<String, Object?>),
        credential: json['credential'] as String,
      );

  final Device device;
  final String credential;
}

class DeviceHeartbeatRequest {
  const DeviceHeartbeatRequest({required this.appVersion, this.capabilities = const {}});

  final String appVersion;
  final Map<String, Object?> capabilities;

  Map<String, Object?> toJson() => {'appVersion': appVersion, 'capabilities': capabilities};
}

class Membership {
  const Membership({required this.id, required this.scope, this.locationId, required this.memberRef, required this.role});

  factory Membership.fromJson(Map<String, Object?> json) => Membership(
        id: json['id'] as String,
        scope: json['scope'] as String,
        locationId: json['locationId'] as String?,
        memberRef: json['memberRef'] as String,
        role: json['role'] as String,
      );

  final String id;
  final String scope;
  final String? locationId;
  final String memberRef;
  final String role;
}

class TimelineEvent {
  const TimelineEvent({required this.id, required this.type, required this.schemaVersion, required this.occurredAt, required this.actorRef, this.deviceId, required this.entityType, required this.entityId, this.entityVersion, this.commandId, required this.data});

  factory TimelineEvent.fromJson(Map<String, Object?> json) => TimelineEvent(
        id: json['id'] as String,
        type: json['type'] as String,
        schemaVersion: json['schemaVersion'] as int,
        occurredAt: json['occurredAt'] as String,
        actorRef: json['actorRef'] as String,
        deviceId: json['deviceId'] as String?,
        entityType: json['entityType'] as String,
        entityId: json['entityId'] as String,
        entityVersion: json['entityVersion'] as int?,
        commandId: json['commandId'] as String?,
        data: json['data'] as Map<String, Object?>,
      );

  final String id;
  final String type;
  final int schemaVersion;
  final String occurredAt;
  final String actorRef;
  final String? deviceId;
  final String entityType;
  final String entityId;
  final int? entityVersion;
  final String? commandId;
  final Map<String, Object?> data;
}

class RealtimeMessage {
  const RealtimeMessage({required this.type, required this.eventId, required this.organisationId, required this.locationId, required this.entityType, required this.entityId, this.version, required this.occurredAt, required this.data});

  factory RealtimeMessage.fromJson(Map<String, Object?> json) => RealtimeMessage(
        type: json['type'] as String,
        eventId: json['eventId'] as String,
        organisationId: json['organisationId'] as String,
        locationId: json['locationId'] as String,
        entityType: json['entityType'] as String,
        entityId: json['entityId'] as String,
        version: json['version'] as int?,
        occurredAt: json['occurredAt'] as String,
        data: json['data'] as Map<String, Object?>,
      );

  factory RealtimeMessage.fromJsonString(String source) => RealtimeMessage.fromJson(jsonDecode(source) as Map<String, Object?>);

  final String type;
  final String eventId;
  final String organisationId;
  final String locationId;
  final String entityType;
  final String entityId;
  final int? version;
  final String occurredAt;
  final Map<String, Object?> data;
}

class AuditTimelineResponse {
  const AuditTimelineResponse({required this.events});

  factory AuditTimelineResponse.fromJson(Map<String, Object?> json) => AuditTimelineResponse(
        events: (json['events'] as List<Object?>).cast<Map<String, Object?>>().map(TimelineEvent.fromJson).toList(),
      );

  final List<TimelineEvent> events;
}

class AnalyticsMetric {
  const AnalyticsMetric({
    required this.activeTableCount,
    required this.occupancySeconds,
    required this.utilisationBasisSeconds,
    required this.utilisationRate,
    required this.completedSessionCount,
    required this.turnoverRate,
    required this.avgSessionSeconds,
    required this.p50SessionSeconds,
    required this.p90SessionSeconds,
    required this.assistRequestCount,
    required this.avgAssistResponseSeconds,
    required this.avgAssistResolutionSeconds,
    required this.anomalyCount,
  });

  factory AnalyticsMetric.fromJson(Map<String, Object?> json) => AnalyticsMetric(
        activeTableCount: json['activeTableCount'] as int,
        occupancySeconds: (json['occupancySeconds'] as num).toDouble(),
        utilisationBasisSeconds: (json['utilisationBasisSeconds'] as num).toDouble(),
        utilisationRate: (json['utilisationRate'] as num).toDouble(),
        completedSessionCount: json['completedSessionCount'] as int,
        turnoverRate: (json['turnoverRate'] as num).toDouble(),
        avgSessionSeconds: (json['avgSessionSeconds'] as num).toDouble(),
        p50SessionSeconds: (json['p50SessionSeconds'] as num).toDouble(),
        p90SessionSeconds: (json['p90SessionSeconds'] as num).toDouble(),
        assistRequestCount: json['assistRequestCount'] as int,
        avgAssistResponseSeconds: (json['avgAssistResponseSeconds'] as num).toDouble(),
        avgAssistResolutionSeconds: (json['avgAssistResolutionSeconds'] as num).toDouble(),
        anomalyCount: json['anomalyCount'] as int,
      );

  final int activeTableCount;
  final double occupancySeconds;
  final double utilisationBasisSeconds;
  final double utilisationRate;
  final int completedSessionCount;
  final double turnoverRate;
  final double avgSessionSeconds;
  final double p50SessionSeconds;
  final double p90SessionSeconds;
  final int assistRequestCount;
  final double avgAssistResponseSeconds;
  final double avgAssistResolutionSeconds;
  final int anomalyCount;
}

class AnalyticsWindowMetric {
  const AnalyticsWindowMetric({this.id, this.name, required this.windowStart, required this.windowEnd, required this.metric});

  factory AnalyticsWindowMetric.fromJson(Map<String, Object?> json) => AnalyticsWindowMetric(
        id: json['id'] as String?,
        name: json['name'] as String?,
        windowStart: json['windowStart'] as String,
        windowEnd: json['windowEnd'] as String,
        metric: AnalyticsMetric.fromJson(json['metric'] as Map<String, Object?>),
      );

  final String? id;
  final String? name;
  final String windowStart;
  final String windowEnd;
  final AnalyticsMetric metric;
}

class AnalyticsDataQuality {
  const AnalyticsDataQuality({
    required this.openSessionCount,
    required this.longOpenSessionCount,
    required this.impossibleSessionCount,
    required this.missingOccupancyCount,
    required this.staleDeviceCount,
    required this.projectorLagSeconds,
    required this.rebuildStatus,
  });

  factory AnalyticsDataQuality.fromJson(Map<String, Object?> json) => AnalyticsDataQuality(
        openSessionCount: json['openSessionCount'] as int,
        longOpenSessionCount: json['longOpenSessionCount'] as int,
        impossibleSessionCount: json['impossibleSessionCount'] as int,
        missingOccupancyCount: json['missingOccupancyCount'] as int,
        staleDeviceCount: json['staleDeviceCount'] as int,
        projectorLagSeconds: (json['projectorLagSeconds'] as num).toDouble(),
        rebuildStatus: json['rebuildStatus'] as String,
      );

  final int openSessionCount;
  final int longOpenSessionCount;
  final int impossibleSessionCount;
  final int missingOccupancyCount;
  final int staleDeviceCount;
  final double projectorLagSeconds;
  final String rebuildStatus;
}

class AnalyticsSummary {
  const AnalyticsSummary({required this.location, required this.date, required this.today, this.currentServicePeriod, required this.dataQuality});

  factory AnalyticsSummary.fromJson(Map<String, Object?> json) => AnalyticsSummary(
        location: Location.fromJson(json['location'] as Map<String, Object?>),
        date: json['date'] as String,
        today: AnalyticsWindowMetric.fromJson(json['today'] as Map<String, Object?>),
        currentServicePeriod: json['currentServicePeriod'] == null
            ? null
            : AnalyticsWindowMetric.fromJson(json['currentServicePeriod'] as Map<String, Object?>),
        dataQuality: AnalyticsDataQuality.fromJson(json['dataQuality'] as Map<String, Object?>),
      );

  final Location location;
  final String date;
  final AnalyticsWindowMetric today;
  final AnalyticsWindowMetric? currentServicePeriod;
  final AnalyticsDataQuality dataQuality;
}

class AnalyticsTimePoint {
  const AnalyticsTimePoint({required this.bucketStart, required this.bucketEnd, required this.metric});

  factory AnalyticsTimePoint.fromJson(Map<String, Object?> json) => AnalyticsTimePoint(
        bucketStart: json['bucketStart'] as String,
        bucketEnd: json['bucketEnd'] as String,
        metric: AnalyticsMetric.fromJson(json['metric'] as Map<String, Object?>),
      );

  final String bucketStart;
  final String bucketEnd;
  final AnalyticsMetric metric;
}

class AnalyticsTimeseriesResponse {
  const AnalyticsTimeseriesResponse({required this.points});

  factory AnalyticsTimeseriesResponse.fromJson(Map<String, Object?> json) => AnalyticsTimeseriesResponse(
        points: (json['points'] as List<Object?>).cast<Map<String, Object?>>().map(AnalyticsTimePoint.fromJson).toList(),
      );

  final List<AnalyticsTimePoint> points;
}

class AnalyticsComparisonPoint {
  const AnalyticsComparisonPoint({required this.groupId, required this.groupName, required this.metricDate, required this.windowStart, required this.windowEnd, required this.metric});

  factory AnalyticsComparisonPoint.fromJson(Map<String, Object?> json) => AnalyticsComparisonPoint(
        groupId: json['groupId'] as String,
        groupName: json['groupName'] as String,
        metricDate: json['metricDate'] as String,
        windowStart: json['windowStart'] as String,
        windowEnd: json['windowEnd'] as String,
        metric: AnalyticsMetric.fromJson(json['metric'] as Map<String, Object?>),
      );

  final String groupId;
  final String groupName;
  final String metricDate;
  final String windowStart;
  final String windowEnd;
  final AnalyticsMetric metric;
}

class AnalyticsComparisonResponse {
  const AnalyticsComparisonResponse({required this.points});

  factory AnalyticsComparisonResponse.fromJson(Map<String, Object?> json) => AnalyticsComparisonResponse(
        points: (json['points'] as List<Object?>).cast<Map<String, Object?>>().map(AnalyticsComparisonPoint.fromJson).toList(),
      );

  final List<AnalyticsComparisonPoint> points;
}

class ServicePeriod {
  const ServicePeriod({required this.id, required this.locationId, required this.name, required this.daysOfWeek, required this.startTime, required this.endTime, required this.isActive, required this.version});

  factory ServicePeriod.fromJson(Map<String, Object?> json) => ServicePeriod(
        id: json['id'] as String,
        locationId: json['locationId'] as String,
        name: json['name'] as String,
        daysOfWeek: (json['daysOfWeek'] as List<Object?>).cast<int>(),
        startTime: json['startTime'] as String,
        endTime: json['endTime'] as String,
        isActive: json['isActive'] as bool,
        version: json['version'] as int,
      );

  final String id;
  final String locationId;
  final String name;
  final List<int> daysOfWeek;
  final String startTime;
  final String endTime;
  final bool isActive;
  final int version;
}

class ServicePeriodsResponse {
  const ServicePeriodsResponse({required this.servicePeriods});

  factory ServicePeriodsResponse.fromJson(Map<String, Object?> json) => ServicePeriodsResponse(
        servicePeriods: (json['servicePeriods'] as List<Object?>).cast<Map<String, Object?>>().map(ServicePeriod.fromJson).toList(),
      );

  final List<ServicePeriod> servicePeriods;
}

class LocationSnapshotResponse {
  const LocationSnapshotResponse({required this.organisationId, required this.locationId, required this.cursor, required this.tables, required this.assists});

  factory LocationSnapshotResponse.fromJson(Map<String, Object?> json) => LocationSnapshotResponse(
        organisationId: json['organisationId'] as String,
        locationId: json['locationId'] as String,
        cursor: json['cursor'] as String,
        tables: (json['tables'] as List<Object?>).cast<Map<String, Object?>>().map(TableState.fromJson).toList(),
        assists: (json['assists'] as List<Object?>).cast<Map<String, Object?>>().map(Assist.fromJson).toList(),
      );

  factory LocationSnapshotResponse.fromJsonString(String source) => LocationSnapshotResponse.fromJson(jsonDecode(source) as Map<String, Object?>);

  final String organisationId;
  final String locationId;
  final String cursor;
  final List<TableState> tables;
  final List<Assist> assists;
}

class SyncEventsResponse {
  const SyncEventsResponse({required this.events, required this.cursor});

  factory SyncEventsResponse.fromJson(Map<String, Object?> json) => SyncEventsResponse(
        events: (json['events'] as List<Object?>).cast<Map<String, Object?>>().map(RealtimeMessage.fromJson).toList(),
        cursor: json['cursor'] as String,
      );

  factory SyncEventsResponse.fromJsonString(String source) => SyncEventsResponse.fromJson(jsonDecode(source) as Map<String, Object?>);

  final List<RealtimeMessage> events;
  final String cursor;
}
