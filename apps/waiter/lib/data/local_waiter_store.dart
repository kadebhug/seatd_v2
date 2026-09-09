import 'dart:async';

import 'package:seatd_client/seatd_client.dart';
import 'package:uuid/uuid.dart';

import '../domain/waiter_models.dart';
import 'waiter_gateway.dart';

class LocalWaiterStore {
  LocalWaiterStore() : _state = const FloorState.empty();

  final _controller = StreamController<FloorState>.broadcast();
  final _processedEventIds = <String>{};
  FloorState _state;

  FloorState get state => _state;
  Stream<FloorState> get states => _controller.stream;

  void dispose() {
    _controller.close();
  }

  void setOnline(bool isOnline) {
    _emit(_state.copyWith(isOnline: isOnline));
  }

  void hydrate({
    LocationSnapshotResponse? snapshot,
    List<WaiterCommand> commands = const [],
  }) {
    var next = _state.copyWith(
      commands: commands
          .map(
            (command) => command.state == CommandState.sending
                ? command.copyWith(state: CommandState.queued)
                : command,
          )
          .toList(),
    );
    _state = next;
    if (snapshot != null) {
      mergeSnapshot(snapshot);
      next = _state.copyWith(commands: _state.commands);
    }
    _emit(next);
  }

  void beginSync() {
    _emit(_state.copyWith(isSyncing: true));
  }

  void mergeReferenceData(List<Floor> floors, List<Zone> zones) {
    _emit(
      _state.copyWith(
        floors: [...floors]..sort((a, b) => a.sortOrder.compareTo(b.sortOrder)),
        zones: [...zones]..sort((a, b) => a.sortOrder.compareTo(b.sortOrder)),
      ),
    );
  }

  void mergeSnapshot(LocationSnapshotResponse snapshot) {
    final activeAssistIds = snapshot.assists.map((assist) => assist.id).toSet();
    final retainedOptimistic = _state.assists.where((assist) {
      return assist.id.startsWith('optimistic-') &&
          !activeAssistIds.contains(assist.id);
    });

    _emit(
      _state.copyWith(
        tables: _mergeTables(snapshot.tables),
        assists: [...snapshot.assists, ...retainedOptimistic],
        cursor: snapshot.cursor,
        lastSuccessfulSync: DateTime.now(),
        isSyncing: false,
        isOnline: true,
      ),
    );
  }

  bool applyRealtimeMessage(RealtimeMessage message) {
    if (!_processedEventIds.add(message.eventId)) {
      return false;
    }
    if (message.entityType == 'table') {
      return _applyTableRealtime(message);
    }
    if (message.entityType == 'assist') {
      return _applyAssistRealtime(message);
    }
    return false;
  }

  void mergeSyncEvents(SyncEventsResponse response) {
    for (final event in response.events) {
      applyRealtimeMessage(event);
    }
    _emit(
      _state.copyWith(
        cursor: response.cursor,
        lastSuccessfulSync: DateTime.now(),
        isSyncing: false,
        isOnline: true,
      ),
    );
  }

  WaiterCommand occupyTable(TableState table, {int? partySize}) {
    final command = _command(
      CommandType.occupyTable,
      table.table.id,
      table.occupancy.version,
      {'partySize': ?partySize},
    );
    final updated = _replaceTable(
      table.table.id,
      TableState(
        table: table.table,
        occupancy: Occupancy(
          status: 'occupied',
          currentSessionId:
              table.occupancy.currentSessionId ?? 'optimistic-${command.id}',
          version: table.occupancy.version,
          updatedAt: DateTime.now().toUtc().toIso8601String(),
        ),
      ),
    );
    _emit(
      _state.copyWith(tables: updated, commands: [..._state.commands, command]),
    );
    return command;
  }

  WaiterCommand clearTable(TableState table) {
    final command = _command(
      CommandType.clearTable,
      table.table.id,
      table.occupancy.version,
      const {},
    );
    _emit(
      _state.copyWith(
        tables: _replaceTable(
          table.table.id,
          TableState(
            table: table.table,
            occupancy: Occupancy(
              status: 'available',
              currentSessionId: null,
              version: table.occupancy.version,
              updatedAt: DateTime.now().toUtc().toIso8601String(),
            ),
          ),
        ),
        commands: [..._state.commands, command],
      ),
    );
    return command;
  }

  WaiterCommand changeAssist(Assist assist, CommandType type) {
    final nextStatus = switch (type) {
      CommandType.acknowledgeAssist => 'acknowledged',
      CommandType.resolveAssist => 'resolved',
      CommandType.cancelAssist => 'cancelled',
      _ => assist.status,
    };
    final command = _command(type, assist.id, assist.version, const {});
    final assists = _state.assists
        .map(
          (current) => current.id == assist.id
              ? Assist(
                  id: assist.id,
                  tableId: assist.tableId,
                  tableSessionId: assist.tableSessionId,
                  status: nextStatus,
                  requestedAt: assist.requestedAt,
                  version: assist.version,
                  note: assist.note,
                )
              : current,
        )
        .where(
          (current) =>
              current.status == 'pending' || current.status == 'acknowledged',
        )
        .toList();
    _emit(
      _state.copyWith(
        assists: assists,
        commands: [..._state.commands, command],
      ),
    );
    return command;
  }

  void markCommandSending(String id) {
    _replaceCommand(
      id,
      (command) => command.copyWith(state: CommandState.sending),
    );
  }

  void markCommandComplete(String id, {Object? response}) {
    if (response is Map<String, Object?>) {
      if (response.containsKey('table') && response.containsKey('occupancy')) {
        final table = TableState.fromJson(response);
        _emit(_state.copyWith(tables: _replaceTable(table.table.id, table)));
      } else if (response.containsKey('tableId') &&
          response.containsKey('status')) {
        final assist = Assist.fromJson(response);
        _mergeAssist(assist);
      }
    }
    _replaceCommand(
      id,
      (command) => command.copyWith(state: CommandState.complete),
    );
  }

  void markCommandFailed(String id, String message) {
    _replaceCommand(
      id,
      (command) => command.copyWith(
        state: CommandState.failed,
        retryCount: command.retryCount + 1,
        message: message,
      ),
    );
  }

  void retryCommand(String id) {
    _replaceCommand(
      id,
      (command) =>
          command.copyWith(state: CommandState.queued, clearMessage: true),
    );
  }

  void acceptCurrentState(String id) {
    _replaceCommand(
      id,
      (command) => command.copyWith(state: CommandState.complete),
    );
  }

  void retryConflictWithLatestVersion(String id) {
    _replaceCommand(id, (command) {
      final table = _state.tables
          .where((item) => item.table.id == command.entityId)
          .firstOrNull;
      final assist = _state.assists
          .where((item) => item.id == command.entityId)
          .firstOrNull;
      final latestVersion = table?.occupancy.version ?? assist?.version;
      return command.copyWith(
        expectedVersion: latestVersion,
        state: CommandState.queued,
        clearMessage: true,
      );
    });
  }

  void clearCompletedCommands() {
    _emit(
      _state.copyWith(
        commands: [
          for (final command in _state.commands)
            if (command.state != CommandState.complete) command,
        ],
      ),
    );
  }

  void clearCommands() {
    _emit(_state.copyWith(commands: const []));
  }

  void classifyCommandFailure(String id, SeatdApiError error) {
    final command = _state.commands.firstWhere((item) => item.id == id);
    final decision = classifyConflict(command, error, _state);
    if (decision.isRetryable) {
      if (command.retryCount >= 2) {
        markCommandFailed(id, decision.message);
        return;
      }
      _replaceCommand(
        id,
        (command) => command.copyWith(
          state: CommandState.queued,
          retryCount: command.retryCount + 1,
          message: decision.message,
        ),
      );
    } else if (decision.isConflict) {
      _replaceCommand(
        id,
        (command) => command.copyWith(
          state: CommandState.conflict,
          message: decision.message,
        ),
      );
    } else {
      _replaceCommand(
        id,
        (command) => command.copyWith(
          state: CommandState.complete,
          message: decision.message,
        ),
      );
    }
  }

  static ConflictDecision classifyConflict(
    WaiterCommand command,
    SeatdApiError error,
    FloorState state,
  ) {
    final table = state.tables
        .where((item) => item.table.id == command.entityId)
        .firstOrNull;
    final assist = state.assists
        .where((item) => item.id == command.entityId)
        .firstOrNull;
    if (command.type == CommandType.occupyTable &&
        error.code == SeatdErrorCode.alreadyOccupied &&
        table?.occupancy.status == 'occupied') {
      return const ConflictDecision.complete('Table is already occupied.');
    }
    if (command.type == CommandType.clearTable &&
        error.code == SeatdErrorCode.alreadyAvailable &&
        table?.occupancy.status == 'available') {
      return const ConflictDecision.complete('Table is already clear.');
    }
    if (command.type == CommandType.clearTable &&
        error.code == SeatdErrorCode.versionConflict) {
      return const ConflictDecision.conflict(
        'Clear needs review because the table changed elsewhere.',
      );
    }
    if ((command.type == CommandType.resolveAssist ||
            command.type == CommandType.cancelAssist) &&
        error.code == SeatdErrorCode.assistAlreadyResolved &&
        (assist == null ||
            assist.status == 'resolved' ||
            assist.status == 'cancelled')) {
      return const ConflictDecision.complete('Assist was already closed.');
    }
    if (error.code == SeatdErrorCode.rateLimited ||
        error.code == SeatdErrorCode.internalError) {
      return ConflictDecision.retry(error.message);
    }
    return ConflictDecision.retry(error.message);
  }

  List<TableState> _mergeTables(List<TableState> incoming) {
    final byId = {for (final table in _state.tables) table.table.id: table};
    for (final table in incoming) {
      final pending = _state.commands.any(
        (command) =>
            command.entityId == table.table.id &&
            (command.type == CommandType.occupyTable ||
                command.type == CommandType.clearTable) &&
            command.state != CommandState.complete,
      );
      if (!pending ||
          (byId[table.table.id]?.occupancy.version ?? 0) <
              table.occupancy.version) {
        byId[table.table.id] = table;
      }
    }
    return byId.values.toList()
      ..sort((a, b) => a.table.label.compareTo(b.table.label));
  }

  bool _applyTableRealtime(RealtimeMessage message) {
    final current = _state.tables
        .where((item) => item.table.id == message.entityId)
        .firstOrNull;
    if (current == null) {
      return false;
    }
    final version = message.version;
    if (version != null && version <= current.occupancy.version) {
      return false;
    }
    final pending = _state.commands.any(
      (command) =>
          command.entityId == message.entityId &&
          command.state != CommandState.complete &&
          (command.type == CommandType.occupyTable ||
              command.type == CommandType.clearTable),
    );
    if (pending) {
      return false;
    }
    final status = message.data['status'] as String?;
    if (status == null) {
      return false;
    }
    _emit(
      _state.copyWith(
        tables: _replaceTable(
          message.entityId,
          TableState(
            table: current.table,
            occupancy: Occupancy(
              status: status,
              currentSessionId: message.data['sessionId'] as String?,
              version: version ?? current.occupancy.version + 1,
              updatedAt: message.occurredAt,
            ),
          ),
        ),
      ),
    );
    return true;
  }

  bool _applyAssistRealtime(RealtimeMessage message) {
    final status = message.data['status'] as String?;
    final tableId = message.data['tableId'] as String?;
    if (status == null || tableId == null) {
      return false;
    }
    final assist = Assist(
      id: message.entityId,
      tableId: tableId,
      tableSessionId: message.data['tableSessionId'] as String?,
      status: status,
      requestedAt: message.occurredAt,
      version: message.version ?? 1,
      note: message.data['note'] as String?,
    );
    _mergeAssist(assist);
    return true;
  }

  void _mergeAssist(Assist assist) {
    final byId = {for (final item in _state.assists) item.id: item};
    final current = byId[assist.id];
    if (current != null && current.version > assist.version) {
      return;
    }
    if (assist.status == 'pending' || assist.status == 'acknowledged') {
      byId[assist.id] = assist;
    } else {
      byId.remove(assist.id);
    }
    _emit(_state.copyWith(assists: byId.values.toList()));
  }

  WaiterCommand _command(
    CommandType type,
    String entityId,
    int expectedVersion,
    Map<String, Object?> payload,
  ) => WaiterCommand(
    id: const Uuid().v4(),
    type: type,
    entityId: entityId,
    payload: payload,
    expectedVersion: expectedVersion,
    createdAt: DateTime.now(),
  );

  List<TableState> _replaceTable(String tableId, TableState next) {
    final tables = [
      for (final table in _state.tables)
        if (table.table.id == tableId) next else table,
    ];
    return tables..sort((a, b) => a.table.label.compareTo(b.table.label));
  }

  void _replaceCommand(
    String id,
    WaiterCommand Function(WaiterCommand) update,
  ) {
    _emit(
      _state.copyWith(
        commands: [
          for (final command in _state.commands)
            if (command.id == id) update(command) else command,
        ],
      ),
    );
  }

  void _emit(FloorState state) {
    _state = state;
    _controller.add(state);
  }
}

extension _FirstOrNull<T> on Iterable<T> {
  T? get firstOrNull {
    final iterator = this.iterator;
    if (!iterator.moveNext()) {
      return null;
    }
    return iterator.current;
  }
}

class CommandProcessor {
  CommandProcessor({
    required LocalWaiterStore store,
    required WaiterGateway gateway,
  }) : _store = store,
       _gateway = gateway;

  final LocalWaiterStore _store;
  final WaiterGateway _gateway;

  Future<void> flush() async {
    final commands = _store.state.commands
        .where((command) => command.state == CommandState.queued)
        .toList();
    for (final command in commands) {
      _store.markCommandSending(command.id);
      try {
        final response = await _gateway.submitCommand(command);
        _store.markCommandComplete(command.id, response: response);
      } on SeatdCommandException catch (error) {
        _store.classifyCommandFailure(command.id, error.error);
      } catch (error) {
        if (command.retryCount >= 2) {
          _store.markCommandFailed(command.id, error.toString());
          continue;
        }
        _store.classifyCommandFailure(
          command.id,
          SeatdApiError(
            code: SeatdErrorCode.internalError,
            message: error.toString(),
          ),
        );
      }
    }
  }
}
