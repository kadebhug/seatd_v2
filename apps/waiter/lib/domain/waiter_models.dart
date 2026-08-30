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

enum CommandState { queued, sending, complete, conflict, failed }

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
    int? retryCount,
    CommandState? state,
    String? message,
  }) => WaiterCommand(
    id: id,
    type: type,
    entityId: entityId,
    payload: payload,
    expectedVersion: expectedVersion,
    createdAt: createdAt,
    retryCount: retryCount ?? this.retryCount,
    state: state ?? this.state,
    message: message ?? this.message,
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
