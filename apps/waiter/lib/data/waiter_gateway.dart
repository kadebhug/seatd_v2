import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:seatd_client/seatd_client.dart';
import 'package:uuid/uuid.dart';

import '../domain/waiter_models.dart';
import 'realtime_connector.dart';

abstract class WaiterGateway {
  Future<PairDeviceResponse> pairDevice(String code);
  Future<Device> heartbeat(WaiterSession session);
  Future<List<Floor>> listFloors();
  Future<List<Zone>> listZones(String floorId);
  Future<LocationSnapshotResponse> getSnapshot(WaiterSession session);
  Future<SyncEventsResponse> listEvents(WaiterSession session, String cursor);
  Future<Object?> submitCommand(WaiterCommand command);
  Stream<RealtimeMessage> realtimeMessages(WaiterSession session);
}

class SeatdHttpWaiterGateway implements WaiterGateway {
  SeatdHttpWaiterGateway({
    required this.apiBaseUrl,
    required this.realtimeBaseUrl,
    http.Client? client,
  }) : _client = client ?? http.Client();

  final Uri apiBaseUrl;
  final Uri realtimeBaseUrl;
  final http.Client _client;
  WaiterSession? session;

  Map<String, String> _headers(WaiterSession session) => {
    'Authorization': 'Bearer ${session.credential}',
    'X-Seatd-Organisation-ID': session.organisationId,
    'X-Seatd-Location-ID': session.locationId,
    'X-Seatd-Actor-Ref': 'device:${session.deviceId}',
    'X-Seatd-Device-ID': session.deviceId,
    'Content-Type': 'application/json',
  };

  @override
  Future<PairDeviceResponse> pairDevice(String code) async {
    final response = await _client.post(
      apiBaseUrl.resolve(pairDeviceEndpoint),
      headers: const {'Content-Type': 'application/json'},
      body: jsonEncode(
        const PairDeviceRequest(
          code: '',
          platform: 'flutter',
          appVersion: waiterAppVersion,
          name: 'Waiter mobile',
          capabilities: {
            'offlineQueue': true,
            'realtime': true,
            'conflictReview': true,
          },
        ).copyWithCode(code).toJson(),
      ),
    );
    return PairDeviceResponse.fromJson(_decodeSuccess(response));
  }

  @override
  Future<Device> heartbeat(WaiterSession session) async {
    final response = await _client.post(
      apiBaseUrl.resolve(deviceHeartbeatEndpoint),
      headers: _headers(session),
      body: jsonEncode(
        const DeviceHeartbeatRequest(
          appVersion: waiterAppVersion,
          capabilities: {
            'offlineQueue': true,
            'realtime': true,
            'conflictReview': true,
          },
        ).toJson(),
      ),
    );
    return Device.fromJson(_decodeSuccess(response));
  }

  @override
  Future<List<Floor>> listFloors() async {
    final json = await _getJson(apiBaseUrl, floorsEndpoint, _requiredSession());
    return (json['floors'] as List<Object?>)
        .cast<Map<String, Object?>>()
        .map(Floor.fromJson)
        .toList();
  }

  @override
  Future<List<Zone>> listZones(String floorId) async {
    final json = await _getJson(
      apiBaseUrl,
      floorZonesEndpoint.replaceAll('{id}', floorId),
      _requiredSession(),
    );
    return (json['zones'] as List<Object?>)
        .cast<Map<String, Object?>>()
        .map(Zone.fromJson)
        .toList();
  }

  @override
  Future<LocationSnapshotResponse> getSnapshot(WaiterSession session) async {
    final json = await _getJson(
      realtimeBaseUrl,
      locationSnapshotEndpoint,
      session,
    );
    return LocationSnapshotResponse.fromJson(json);
  }

  @override
  Future<SyncEventsResponse> listEvents(
    WaiterSession session,
    String cursor,
  ) async {
    final endpoint = Uri(
      path: syncEventsEndpoint,
      queryParameters: {if (cursor.isNotEmpty) 'after': cursor, 'limit': '500'},
    ).toString();
    final json = await _getJson(realtimeBaseUrl, endpoint, session);
    return SyncEventsResponse.fromJson(json);
  }

  @override
  Future<Object?> submitCommand(WaiterCommand command) async {
    final activeSession = _requiredSession();
    final endpoint = switch (command.type) {
      CommandType.occupyTable => occupyTableEndpoint,
      CommandType.clearTable => clearTableEndpoint,
      CommandType.acknowledgeAssist => acknowledgeAssistEndpoint,
      CommandType.resolveAssist => resolveAssistEndpoint,
      CommandType.cancelAssist => cancelAssistEndpoint,
    }.replaceAll('{id}', command.entityId);
    final response = await _client.post(
      apiBaseUrl.resolve(endpoint),
      headers: _headers(activeSession),
      body: jsonEncode({
        'commandId': command.id,
        'expectedVersion': command.expectedVersion,
        ...command.payload,
      }),
    );
    return _decodeSuccess(response);
  }

  @override
  Stream<RealtimeMessage> realtimeMessages(WaiterSession session) =>
      connectRealtime(realtimeBaseUrl, _headers(session));

  Future<Map<String, Object?>> _getJson(
    Uri baseUrl,
    String endpoint,
    WaiterSession session,
  ) async {
    final response = await _client.get(
      baseUrl.resolve(endpoint),
      headers: _headers(session),
    );
    return _decodeSuccess(response);
  }

  WaiterSession _requiredSession() {
    final active = session;
    if (active == null) {
      throw const SeatdCommandException(
        SeatdApiError(
          code: SeatdErrorCode.unauthorized,
          message: 'waiter device is not paired',
        ),
      );
    }
    return active;
  }
}

class DemoWaiterGateway implements WaiterGateway {
  final _uuid = const Uuid();

  @override
  Future<PairDeviceResponse> pairDevice(String code) async =>
      PairDeviceResponse(
        device: const Device(
          id: demoDeviceId,
          organisationId: demoOrganisationId,
          locationId: demoLocationId,
          name: 'Demo waiter',
          deviceType: 'waiter_mobile',
          platform: 'flutter',
          appVersion: waiterAppVersion,
          trustState: 'trusted',
          registeredAt: '2026-08-30T10:00:00Z',
          heartbeatIntervalSeconds: 60,
        ),
        credential: 'demo-waiter-credential',
      );

  @override
  Future<Device> heartbeat(WaiterSession session) async => const Device(
    id: demoDeviceId,
    organisationId: demoOrganisationId,
    locationId: demoLocationId,
    name: 'Demo waiter',
    deviceType: 'waiter_mobile',
    platform: 'flutter',
    appVersion: waiterAppVersion,
    trustState: 'trusted',
    registeredAt: '2026-08-30T10:00:00Z',
    heartbeatIntervalSeconds: 60,
  );

  @override
  Future<List<Floor>> listFloors() async => const [
    Floor(
      id: '33333333-3333-3333-3333-333333333333',
      slug: 'ground-floor',
      name: 'Ground Floor',
      sortOrder: 0,
      canvas: {'width': 1200, 'height': 800},
      version: 1,
    ),
    Floor(
      id: '33333333-3333-3333-3333-333333333334',
      slug: 'rooftop',
      name: 'Rooftop',
      sortOrder: 1,
      canvas: {'width': 1000, 'height': 650},
      version: 1,
    ),
  ];

  @override
  Future<List<Zone>> listZones(String floorId) async {
    if (floorId.endsWith('3334')) {
      return const [
        Zone(
          id: '44444444-4444-4444-4444-444444444443',
          floorId: '33333333-3333-3333-3333-333333333334',
          name: 'Rooftop Bar',
          sortOrder: 0,
          version: 1,
        ),
      ];
    }
    return const [
      Zone(
        id: '44444444-4444-4444-4444-444444444441',
        floorId: '33333333-3333-3333-3333-333333333333',
        name: 'Dining Room',
        sortOrder: 0,
        version: 1,
      ),
      Zone(
        id: '44444444-4444-4444-4444-444444444442',
        floorId: '33333333-3333-3333-3333-333333333333',
        name: 'Patio',
        sortOrder: 1,
        version: 1,
      ),
    ];
  }

  @override
  Future<LocationSnapshotResponse> getSnapshot(WaiterSession session) async {
    final now = DateTime.now().toUtc().toIso8601String();
    return LocationSnapshotResponse(
      organisationId: demoOrganisationId,
      locationId: demoLocationId,
      cursor: 'demo-cursor',
      tables: [
        _table(
          '55555555-5555-5555-5555-555555555551',
          'T1',
          '4',
          'rectangle',
          '44444444-4444-4444-4444-444444444441',
          'available',
          1,
          now,
          {'x': 120, 'y': 140, 'width': 120, 'height': 80},
        ),
        _table(
          '55555555-5555-5555-5555-555555555552',
          'T2',
          '2',
          'circle',
          '44444444-4444-4444-4444-444444444441',
          'occupied',
          2,
          now,
          {'x': 320, 'y': 140, 'radius': 46},
        ),
        _table(
          '55555555-5555-5555-5555-555555555553',
          'P1',
          '6',
          'rectangle',
          '44444444-4444-4444-4444-444444444442',
          'available',
          1,
          now,
          {'x': 140, 'y': 420, 'width': 160, 'height': 90},
        ),
        _table(
          '55555555-5555-5555-5555-555555555554',
          'R1',
          '4',
          'square',
          '44444444-4444-4444-4444-444444444443',
          'available',
          1,
          now,
          {'x': 180, 'y': 180, 'width': 90, 'height': 90},
        ),
      ],
      assists: const [
        Assist(
          id: '77777777-7777-7777-7777-777777777777',
          tableId: '55555555-5555-5555-5555-555555555552',
          status: 'pending',
          requestedAt: '2026-08-30T10:30:00Z',
          version: 1,
          note: 'Water',
        ),
      ],
    );
  }

  @override
  Future<SyncEventsResponse> listEvents(
    WaiterSession session,
    String cursor,
  ) async => const SyncEventsResponse(events: [], cursor: 'demo-cursor');

  @override
  Future<Object?> submitCommand(WaiterCommand command) async =>
      <String, Object?>{'commandId': command.id};

  @override
  Stream<RealtimeMessage> realtimeMessages(WaiterSession session) =>
      StreamController<RealtimeMessage>.broadcast().stream;

  TableState _table(
    String id,
    String label,
    String capacity,
    String shape,
    String zoneId,
    String status,
    int version,
    String updatedAt,
    Map<String, Object?> geometry,
  ) => TableState(
    table: SeatdTable(
      id: id,
      floorId: zoneId.endsWith('4443')
          ? '33333333-3333-3333-3333-333333333334'
          : '33333333-3333-3333-3333-333333333333',
      zoneId: zoneId,
      label: label,
      capacityLabel: capacity,
      shape: shape,
      geometry: geometry,
      version: 1,
    ),
    occupancy: Occupancy(
      status: status,
      currentSessionId: status == 'occupied' ? _uuid.v4() : null,
      version: version,
      updatedAt: updatedAt,
    ),
  );
}

class SeatdCommandException implements Exception {
  const SeatdCommandException(this.error);

  final SeatdApiError error;

  @override
  String toString() => error.message;
}

const waiterAppVersion = String.fromEnvironment(
  'SEATD_WAITER_APP_VERSION',
  defaultValue: '0.1.0',
);

Map<String, Object?> _decodeSuccess(http.Response response) {
  final body = response.body.isEmpty
      ? <String, Object?>{}
      : (jsonDecode(response.body) as Map<Object?, Object?>)
            .cast<String, Object?>();
  if (response.statusCode >= 200 && response.statusCode < 300) {
    return body;
  }
  throw SeatdCommandException(SeatdErrorResponse.fromJson(body).error);
}

extension _PairDeviceRequestCode on PairDeviceRequest {
  PairDeviceRequest copyWithCode(String code) => PairDeviceRequest(
    code: code,
    platform: platform,
    appVersion: appVersion,
    name: name,
    capabilities: capabilities,
    configuration: configuration,
  );
}
