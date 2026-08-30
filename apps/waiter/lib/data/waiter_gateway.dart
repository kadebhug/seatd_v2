import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:seatd_client/seatd_client.dart';
import 'package:uuid/uuid.dart';

import '../domain/waiter_models.dart';
import 'realtime_connector.dart';

abstract class WaiterGateway {
  Future<List<Floor>> listFloors();
  Future<List<Zone>> listZones(String floorId);
  Future<LocationSnapshotResponse> getSnapshot();
  Future<Object?> submitCommand(WaiterCommand command);
  Stream<RealtimeMessage> realtimeMessages();
}

class SeatdHttpWaiterGateway implements WaiterGateway {
  SeatdHttpWaiterGateway({required this.baseUrl, http.Client? client})
    : _client = client ?? http.Client();

  final Uri baseUrl;
  final http.Client _client;

  Map<String, String> get _headers => const {
    'X-Seatd-Organisation-ID': demoOrganisationId,
    'X-Seatd-Location-ID': demoLocationId,
    'X-Seatd-Actor-Ref': demoActorRef,
    'X-Seatd-Device-ID': demoDeviceId,
    'Content-Type': 'application/json',
  };

  @override
  Future<List<Floor>> listFloors() async {
    final json = await _getJson(floorsEndpoint);
    return (json['floors'] as List<Object?>)
        .cast<Map<String, Object?>>()
        .map(Floor.fromJson)
        .toList();
  }

  @override
  Future<List<Zone>> listZones(String floorId) async {
    final json = await _getJson(floorZonesEndpoint.replaceAll('{id}', floorId));
    return (json['zones'] as List<Object?>)
        .cast<Map<String, Object?>>()
        .map(Zone.fromJson)
        .toList();
  }

  @override
  Future<LocationSnapshotResponse> getSnapshot() async {
    final json = await _getJson(locationSnapshotEndpoint);
    return LocationSnapshotResponse.fromJson(json);
  }

  @override
  Future<Object?> submitCommand(WaiterCommand command) async {
    final endpoint = switch (command.type) {
      CommandType.occupyTable => occupyTableEndpoint,
      CommandType.clearTable => clearTableEndpoint,
      CommandType.acknowledgeAssist => acknowledgeAssistEndpoint,
      CommandType.resolveAssist => resolveAssistEndpoint,
      CommandType.cancelAssist => cancelAssistEndpoint,
    }.replaceAll('{id}', command.entityId);
    final response = await _client.post(
      baseUrl.resolve(endpoint),
      headers: _headers,
      body: jsonEncode({
        'commandId': command.id,
        'expectedVersion': command.expectedVersion,
        ...command.payload,
      }),
    );
    final body = jsonDecode(response.body) as Map<String, Object?>;
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return body;
    }
    throw SeatdCommandException(SeatdErrorResponse.fromJson(body).error);
  }

  @override
  Stream<RealtimeMessage> realtimeMessages() =>
      connectRealtime(baseUrl, _headers);

  Future<Map<String, Object?>> _getJson(String endpoint) async {
    final response = await _client.get(
      baseUrl.resolve(endpoint),
      headers: _headers,
    );
    final body = jsonDecode(response.body) as Map<String, Object?>;
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return body;
    }
    throw SeatdCommandException(SeatdErrorResponse.fromJson(body).error);
  }
}

class DemoWaiterGateway implements WaiterGateway {
  final _uuid = const Uuid();

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
  Future<LocationSnapshotResponse> getSnapshot() async {
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
  Future<Object?> submitCommand(WaiterCommand command) async =>
      <String, Object?>{'commandId': command.id};

  @override
  Stream<RealtimeMessage> realtimeMessages() => const Stream.empty();

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
