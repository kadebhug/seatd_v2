import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:seatd_client/seatd_client.dart';

import '../../../../data/local_waiter_store.dart';
import '../../../../data/waiter_session_store.dart';
import '../../../../data/waiter_gateway.dart';
import '../../../../domain/waiter_models.dart';

class FloorViewModel extends ChangeNotifier {
  FloorViewModel({
    required LocalWaiterStore store,
    required WaiterGateway gateway,
    required CommandProcessor commandProcessor,
    required WaiterSession session,
    required WaiterSessionStore sessionStore,
    required VoidCallback onAuthExpired,
  }) : _store = store,
       _gateway = gateway,
       _commandProcessor = commandProcessor,
       _session = session,
       _sessionStore = sessionStore,
       _onAuthExpired = onAuthExpired {
    _subscription = _store.states.listen((state) {
      _state = state;
      _ensureSelection();
      unawaited(_sessionStore.writeCommands(_durableCommands(state)));
      notifyListeners();
    });
    _state = _store.state;
    final gateway = _gateway;
    if (gateway is SeatdHttpWaiterGateway) {
      gateway.session = session;
    }
  }

  final LocalWaiterStore _store;
  final WaiterGateway _gateway;
  final CommandProcessor _commandProcessor;
  final WaiterSessionStore _sessionStore;
  final VoidCallback _onAuthExpired;
  StreamSubscription<FloorState>? _subscription;
  StreamSubscription<RealtimeMessage>? _realtimeSubscription;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;
  WaiterSession _session;
  int _reconnectAttempt = 0;
  bool _disposed = false;

  FloorState _state = const FloorState.empty();
  String? _selectedFloorId;
  String? _selectedZoneId;
  String? _statusMessage;

  FloorState get state => _state;
  String? get selectedFloorId => _selectedFloorId;
  String? get selectedZoneId => _selectedZoneId;
  String? get statusMessage => _statusMessage;

  List<Zone> get visibleZones {
    final floorId = _selectedFloorId;
    return [
      for (final zone in _state.zones)
        if (floorId == null || zone.floorId == floorId) zone,
    ];
  }

  List<TableState> get visibleTables {
    final floorId = _selectedFloorId;
    final zoneId = _selectedZoneId;
    return [
      for (final table in _state.tables)
        if ((floorId == null || table.table.floorId == floorId) &&
            (zoneId == null || table.table.zoneId == zoneId))
          table,
    ];
  }

  List<Assist> assistsForTable(String tableId) =>
      _state.assists.where((assist) => assist.tableId == tableId).toList();

  WaiterCommand? commandForEntity(String entityId) {
    for (final command in _state.commands.reversed) {
      if (command.entityId == entityId &&
          command.state != CommandState.complete) {
        return command;
      }
    }
    return null;
  }

  Future<void> bootstrap() async {
    await _restoreCachedState();
    await resync(forceSnapshot: _state.cursor.isEmpty);
  }

  Future<void> resync({bool forceSnapshot = false}) async {
    _store.beginSync();
    try {
      final device = await _gateway.heartbeat(_session);
      _session = _session.copyWith(device: device);
      final gateway = _gateway;
      if (gateway is SeatdHttpWaiterGateway) {
        gateway.session = _session;
      }
      await _sessionStore.writeSession(_session);
      final floors = await _gateway.listFloors();
      final zones = <Zone>[];
      for (final floor in floors) {
        zones.addAll(await _gateway.listZones(floor.id));
      }
      _store.mergeReferenceData(floors, zones);
      if (!forceSnapshot && _state.cursor.isNotEmpty) {
        _store.mergeSyncEvents(
          await _gateway.listEvents(_session, _state.cursor),
        );
      } else {
        final snapshot = await _gateway.getSnapshot(_session);
        _store.mergeSnapshot(snapshot);
        await _sessionStore.writeSnapshot(snapshot);
      }
      await _realtimeSubscription?.cancel();
      _realtimeSubscription = _gateway
          .realtimeMessages(_session)
          .listen(
            _store.applyRealtimeMessage,
            onError: (_) => _scheduleReconnect(),
            onDone: _scheduleReconnect,
            cancelOnError: true,
          );
      await _commandProcessor.flush();
      _startHeartbeat();
      _reconnectAttempt = 0;
      _statusMessage = null;
    } catch (error) {
      if (_isAuthExpired(error)) {
        _statusMessage = 'Device access expired';
        notifyListeners();
        _onAuthExpired();
        return;
      }
      _store.setOnline(false);
      _statusMessage = 'Offline mode: ${error.toString()}';
      _scheduleReconnect();
      notifyListeners();
    }
  }

  void selectFloor(String floorId) {
    _selectedFloorId = floorId;
    _selectedZoneId = null;
    notifyListeners();
  }

  void selectZone(String? zoneId) {
    _selectedZoneId = zoneId;
    notifyListeners();
  }

  Future<void> toggleTable(TableState table) async {
    if (table.occupancy.status == 'occupied') {
      _store.clearTable(table);
    } else {
      _store.occupyTable(table);
    }
    await flushQueue();
  }

  Future<void> acknowledgeAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.acknowledgeAssist);
    await flushQueue();
  }

  Future<void> resolveAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.resolveAssist);
    await flushQueue();
  }

  Future<void> cancelAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.cancelAssist);
    await flushQueue();
  }

  Future<void> flushQueue() async {
    try {
      await _commandProcessor.flush();
    } catch (_) {
      _store.setOnline(false);
    }
  }

  Future<void> retryCommand(WaiterCommand command) async {
    _store.retryCommand(command.id);
    await flushQueue();
  }

  Future<void> retryConflictWithLatestVersion(WaiterCommand command) async {
    _store.retryConflictWithLatestVersion(command.id);
    await flushQueue();
  }

  void acceptCurrentState(WaiterCommand command) {
    _store.acceptCurrentState(command.id);
  }

  void clearCompletedCommands() {
    _store.clearCompletedCommands();
  }

  @override
  void dispose() {
    _disposed = true;
    _heartbeatTimer?.cancel();
    _reconnectTimer?.cancel();
    _subscription?.cancel();
    _realtimeSubscription?.cancel();
    _store.dispose();
    super.dispose();
  }

  Future<void> _restoreCachedState() async {
    final snapshot = await _sessionStore.readSnapshot();
    final commands = await _sessionStore.readCommands();
    _store.hydrate(snapshot: snapshot, commands: commands);
  }

  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    final seconds = _session.heartbeatIntervalSeconds <= 0
        ? 60
        : _session.heartbeatIntervalSeconds;
    _heartbeatTimer = Timer.periodic(Duration(seconds: seconds), (_) {
      unawaited(_heartbeat());
    });
  }

  Future<void> _heartbeat() async {
    try {
      final device = await _gateway.heartbeat(_session);
      _session = _session.copyWith(device: device);
      await _sessionStore.writeSession(_session);
      _store.setOnline(true);
    } catch (error) {
      if (_isAuthExpired(error)) {
        _onAuthExpired();
        return;
      }
      _store.setOnline(false);
    }
  }

  void _scheduleReconnect() {
    if (_disposed || _reconnectTimer?.isActive == true) {
      return;
    }
    _store.setOnline(false);
    _reconnectAttempt++;
    final seconds = switch (_reconnectAttempt) {
      1 => 2,
      2 => 5,
      3 => 10,
      _ => 20,
    };
    _statusMessage = 'Reconnecting...';
    notifyListeners();
    _reconnectTimer = Timer(Duration(seconds: seconds), () {
      _reconnectTimer = null;
      unawaited(resync());
    });
  }

  void _ensureSelection() {
    if (_selectedFloorId == null && _state.floors.isNotEmpty) {
      _selectedFloorId = _state.floors.first.id;
    }
    if (_selectedFloorId != null &&
        !_state.floors.any((floor) => floor.id == _selectedFloorId)) {
      _selectedFloorId = _state.floors.isEmpty ? null : _state.floors.first.id;
      _selectedZoneId = null;
    }
    if (_selectedZoneId != null &&
        !visibleZones.any((zone) => zone.id == _selectedZoneId)) {
      _selectedZoneId = null;
    }
  }
}

List<WaiterCommand> _durableCommands(FloorState state) => [
  for (final command in state.commands)
    if (command.state != CommandState.complete) command,
];

bool _isAuthExpired(Object error) =>
    error is SeatdCommandException &&
    (error.error.code == SeatdErrorCode.unauthorized ||
        error.error.code == SeatdErrorCode.forbidden);
