import 'package:seatd_client/seatd_client.dart';

const demoOrganisationId = '11111111-1111-1111-1111-111111111111';
const demoLocationId = '22222222-2222-2222-2222-222222222222';
const demoActorRef = 'user:waiter-demo';
const demoDeviceId = '66666666-6666-6666-6666-666666666666';

enum CommandType {
  occupyTable,
  clearTable,
  acknowledgeAssist,
  resolveAssist,
  cancelAssist,
}

CommandType commandTypeFromWire(String value) => switch (value) {
  'occupyTable' => CommandType.occupyTable,
  'clearTable' => CommandType.clearTable,
  'acknowledgeAssist' => CommandType.acknowledgeAssist,
  'resolveAssist' => CommandType.resolveAssist,
  'cancelAssist' => CommandType.cancelAssist,
  _ => throw FormatException('unknown command type: $value'),
};

String commandTypeToWire(CommandType value) => switch (value) {
  CommandType.occupyTable => 'occupyTable',
  CommandType.clearTable => 'clearTable',
  CommandType.acknowledgeAssist => 'acknowledgeAssist',
  CommandType.resolveAssist => 'resolveAssist',
  CommandType.cancelAssist => 'cancelAssist',
};

enum CommandState { queued, sending, complete, conflict, failed }

CommandState commandStateFromWire(String value) => switch (value) {
  'queued' => CommandState.queued,
  'sending' => CommandState.sending,
  'complete' => CommandState.complete,
  'conflict' => CommandState.conflict,
  'failed' => CommandState.failed,
  _ => throw FormatException('unknown command state: $value'),
};

String commandStateToWire(CommandState value) => switch (value) {
  CommandState.queued => 'queued',
  CommandState.sending => 'sending',
  CommandState.complete => 'complete',
  CommandState.conflict => 'conflict',
  CommandState.failed => 'failed',
};

class WaiterCommand {
  const WaiterCommand({
    required this.id,
    required this.type,
    required this.entityId,
    required this.payload,
    required this.expectedVersion,
    required this.createdAt,
    this.retryCount = 0,
    this.state = CommandState.queued,
    this.message,
  });

  final String id;
  final CommandType type;
  final String entityId;
  final Map<String, Object?> payload;
  final int expectedVersion;
  final DateTime createdAt;
  final int retryCount;
  final CommandState state;
  final String? message;

  WaiterCommand copyWith({
    int? expectedVersion,
    int? retryCount,
    CommandState? state,
    String? message,
    bool clearMessage = false,
  }) => WaiterCommand(
    id: id,
    type: type,
    entityId: entityId,
    payload: payload,
    expectedVersion: expectedVersion ?? this.expectedVersion,
    createdAt: createdAt,
    retryCount: retryCount ?? this.retryCount,
    state: state ?? this.state,
    message: clearMessage ? null : message ?? this.message,
  );

  Map<String, Object?> toJson() => {
    'id': id,
    'type': commandTypeToWire(type),
    'entityId': entityId,
    'payload': payload,
    'expectedVersion': expectedVersion,
    'createdAt': createdAt.toUtc().toIso8601String(),
    'retryCount': retryCount,
    'state': commandStateToWire(state),
    if (message != null) 'message': message,
  };

  factory WaiterCommand.fromJson(Map<String, Object?> json) => WaiterCommand(
    id: json['id'] as String,
    type: commandTypeFromWire(json['type'] as String),
    entityId: json['entityId'] as String,
    payload:
        (json['payload'] as Map<Object?, Object?>?)?.cast<String, Object?>() ??
        const {},
    expectedVersion: json['expectedVersion'] as int,
    createdAt: DateTime.parse(json['createdAt'] as String),
    retryCount: json['retryCount'] as int? ?? 0,
    state: commandStateFromWire(json['state'] as String? ?? 'queued'),
    message: json['message'] as String?,
  );
}

class FloorState {
  const FloorState({
    required this.floors,
    required this.zones,
    required this.tables,
    required this.assists,
    required this.commands,
    this.cursor = '',
    this.lastSuccessfulSync,
    this.isOnline = true,
    this.isSyncing = false,
  });

  const FloorState.empty()
    : floors = const [],
      zones = const [],
      tables = const [],
      assists = const [],
      commands = const [],
      cursor = '',
      lastSuccessfulSync = null,
      isOnline = true,
      isSyncing = false;

  final List<Floor> floors;
  final List<Zone> zones;
  final List<TableState> tables;
  final List<Assist> assists;
  final List<WaiterCommand> commands;
  final String cursor;
  final DateTime? lastSuccessfulSync;
  final bool isOnline;
  final bool isSyncing;

  FloorState copyWith({
    List<Floor>? floors,
    List<Zone>? zones,
    List<TableState>? tables,
    List<Assist>? assists,
    List<WaiterCommand>? commands,
    String? cursor,
    DateTime? lastSuccessfulSync,
    bool? isOnline,
    bool? isSyncing,
  }) => FloorState(
    floors: floors ?? this.floors,
    zones: zones ?? this.zones,
    tables: tables ?? this.tables,
    assists: assists ?? this.assists,
    commands: commands ?? this.commands,
    cursor: cursor ?? this.cursor,
    lastSuccessfulSync: lastSuccessfulSync ?? this.lastSuccessfulSync,
    isOnline: isOnline ?? this.isOnline,
    isSyncing: isSyncing ?? this.isSyncing,
  );
}

class ConflictDecision {
  const ConflictDecision.complete(this.message)
    : isRetryable = false,
      isConflict = false;
  const ConflictDecision.conflict(this.message)
    : isRetryable = false,
      isConflict = true;
  const ConflictDecision.retry(this.message)
    : isRetryable = true,
      isConflict = false;

  final String message;
  final bool isRetryable;
  final bool isConflict;
}

class WaiterSession {
  const WaiterSession({
    required this.credential,
    required this.deviceId,
    required this.organisationId,
    required this.locationId,
    required this.deviceName,
    required this.heartbeatIntervalSeconds,
  });

  factory WaiterSession.fromPairing({
    required String credential,
    required Device device,
  }) => WaiterSession(
    credential: credential,
    deviceId: device.id,
    organisationId: device.organisationId,
    locationId: device.locationId ?? '',
    deviceName: device.name ?? 'Waiter device',
    heartbeatIntervalSeconds: device.heartbeatIntervalSeconds,
  );

  factory WaiterSession.fromJson(Map<String, Object?> json) => WaiterSession(
    credential: json['credential'] as String,
    deviceId: json['deviceId'] as String,
    organisationId: json['organisationId'] as String,
    locationId: json['locationId'] as String,
    deviceName: json['deviceName'] as String? ?? 'Waiter device',
    heartbeatIntervalSeconds: json['heartbeatIntervalSeconds'] as int? ?? 60,
  );

  final String credential;
  final String deviceId;
  final String organisationId;
  final String locationId;
  final String deviceName;
  final int heartbeatIntervalSeconds;

  Map<String, Object?> toJson() => {
    'credential': credential,
    'deviceId': deviceId,
    'organisationId': organisationId,
    'locationId': locationId,
    'deviceName': deviceName,
    'heartbeatIntervalSeconds': heartbeatIntervalSeconds,
  };

  WaiterSession copyWith({Device? device}) {
    final next = device;
    if (next == null) {
      return this;
    }
    return WaiterSession(
      credential: credential,
      deviceId: next.id,
      organisationId: next.organisationId,
      locationId: next.locationId ?? locationId,
      deviceName: next.name ?? deviceName,
      heartbeatIntervalSeconds: next.heartbeatIntervalSeconds,
    );
  }
}
