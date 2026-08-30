import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:seatd_client/seatd_client.dart';

import '../../../../data/local_waiter_store.dart';
import '../../../../data/waiter_gateway.dart';
import '../../../../domain/waiter_models.dart';

class FloorViewModel extends ChangeNotifier {
  FloorViewModel({
    required LocalWaiterStore store,
    required WaiterGateway gateway,
    required CommandProcessor commandProcessor,
  }) : _store = store,
       _gateway = gateway,
       _commandProcessor = commandProcessor {
    _subscription = _store.states.listen((state) {
      _state = state;
      _ensureSelection();
      notifyListeners();
    });
    _state = _store.state;
  }

  final LocalWaiterStore _store;
  final WaiterGateway _gateway;
  final CommandProcessor _commandProcessor;
  StreamSubscription<FloorState>? _subscription;
  StreamSubscription<RealtimeMessage>? _realtimeSubscription;

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
    _store.beginSync();
    try {
      final floors = await _gateway.listFloors();
      final zones = <Zone>[];
      for (final floor in floors) {
        zones.addAll(await _gateway.listZones(floor.id));
      }
      _store.mergeReferenceData(floors, zones);
      _store.mergeSnapshot(await _gateway.getSnapshot());
      await _realtimeSubscription?.cancel();
      _realtimeSubscription = _gateway.realtimeMessages().listen(
        _store.applyRealtimeMessage,
        onError: (_) => _store.setOnline(false),
      );
      await _commandProcessor.flush();
      _statusMessage = null;
    } catch (error) {
      _store.setOnline(false);
      _statusMessage = 'Offline mode: ${error.toString()}';
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
    await _commandProcessor.flush();
  }

  Future<void> acknowledgeAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.acknowledgeAssist);
    await _commandProcessor.flush();
  }

  Future<void> resolveAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.resolveAssist);
    await _commandProcessor.flush();
  }

  Future<void> cancelAssist(Assist assist) async {
    _store.changeAssist(assist, CommandType.cancelAssist);
    await _commandProcessor.flush();
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _realtimeSubscription?.cancel();
    _store.dispose();
    super.dispose();
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
