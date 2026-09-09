import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:seatd_client/seatd_client.dart';

import '../domain/waiter_models.dart';

abstract class WaiterSessionStore {
  Future<WaiterSession?> readSession();
  Future<void> writeSession(WaiterSession session);
  Future<LocationSnapshotResponse?> readSnapshot();
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot);
  Future<List<WaiterCommand>> readCommands();
  Future<void> writeCommands(List<WaiterCommand> commands);
  Future<void> clear();
}

class SecureWaiterSessionStore implements WaiterSessionStore {
  const SecureWaiterSessionStore({
    FlutterSecureStorage storage = const FlutterSecureStorage(),
  }) : _storage = storage;

  final FlutterSecureStorage _storage;

  static const _sessionKey = 'seatd.waiter.session';
  static const _snapshotKey = 'seatd.waiter.snapshot';
  static const _commandsKey = 'seatd.waiter.commands';

  @override
  Future<WaiterSession?> readSession() async {
    final value = await _storage.read(key: _sessionKey);
    if (value == null || value.isEmpty) {
      return null;
    }
    return WaiterSession.fromJson(
      (jsonDecode(value) as Map<Object?, Object?>).cast<String, Object?>(),
    );
  }

  @override
  Future<void> writeSession(WaiterSession session) =>
      _storage.write(key: _sessionKey, value: jsonEncode(session.toJson()));

  @override
  Future<LocationSnapshotResponse?> readSnapshot() async {
    final value = await _storage.read(key: _snapshotKey);
    if (value == null || value.isEmpty) {
      return null;
    }
    return LocationSnapshotResponse.fromJson(
      (jsonDecode(value) as Map<Object?, Object?>).cast<String, Object?>(),
    );
  }

  @override
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot) => _storage
      .write(key: _snapshotKey, value: jsonEncode(_snapshotToJson(snapshot)));

  @override
  Future<List<WaiterCommand>> readCommands() async {
    final value = await _storage.read(key: _commandsKey);
    if (value == null || value.isEmpty) {
      return const [];
    }
    return (jsonDecode(value) as List<Object?>)
        .map((item) => (item as Map<Object?, Object?>).cast<String, Object?>())
        .map(WaiterCommand.fromJson)
        .toList();
  }

  @override
  Future<void> writeCommands(List<WaiterCommand> commands) => _storage.write(
    key: _commandsKey,
    value: jsonEncode(commands.map((command) => command.toJson()).toList()),
  );

  @override
  Future<void> clear() async {
    await _storage.delete(key: _sessionKey);
    await _storage.delete(key: _snapshotKey);
    await _storage.delete(key: _commandsKey);
  }
}

class MemoryWaiterSessionStore implements WaiterSessionStore {
  MemoryWaiterSessionStore({
    this.session,
    this.snapshot,
    List<WaiterCommand>? commands,
  }) : commands = commands ?? <WaiterCommand>[];

  WaiterSession? session;
  LocationSnapshotResponse? snapshot;
  List<WaiterCommand> commands;

  @override
  Future<WaiterSession?> readSession() async => session;

  @override
  Future<void> writeSession(WaiterSession session) async {
    this.session = session;
  }

  @override
  Future<LocationSnapshotResponse?> readSnapshot() async => snapshot;

  @override
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot) async {
    this.snapshot = snapshot;
  }

  @override
  Future<List<WaiterCommand>> readCommands() async => commands;

  @override
  Future<void> writeCommands(List<WaiterCommand> commands) async {
    this.commands = commands;
  }

  @override
  Future<void> clear() async {
    session = null;
    snapshot = null;
    commands = <WaiterCommand>[];
  }
}

Map<String, Object?> _snapshotToJson(LocationSnapshotResponse snapshot) => {
  'organisationId': snapshot.organisationId,
  'locationId': snapshot.locationId,
  'cursor': snapshot.cursor,
  'tables': snapshot.tables.map(_tableStateToJson).toList(),
  'assists': snapshot.assists.map(_assistToJson).toList(),
};

Map<String, Object?> _tableStateToJson(TableState state) => {
  'table': {
    'id': state.table.id,
    'floorId': state.table.floorId,
    'zoneId': state.table.zoneId,
    'label': state.table.label,
    'capacityLabel': state.table.capacityLabel,
    'shape': state.table.shape,
    'geometry': state.table.geometry,
    'version': state.table.version,
  },
  'occupancy': {
    'status': state.occupancy.status,
    if (state.occupancy.currentSessionId != null)
      'currentSessionId': state.occupancy.currentSessionId,
    'version': state.occupancy.version,
    'updatedAt': state.occupancy.updatedAt,
  },
};

Map<String, Object?> _assistToJson(Assist assist) => {
  'id': assist.id,
  'tableId': assist.tableId,
  if (assist.tableSessionId != null) 'tableSessionId': assist.tableSessionId,
  'status': assist.status,
  'requestedAt': assist.requestedAt,
  'version': assist.version,
  if (assist.note != null) 'note': assist.note,
};
