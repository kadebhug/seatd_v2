import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:seatd_client/seatd_client.dart';

const displayAppVersion = '0.1.0';
const demoOrganisationId = '11111111-1111-1111-1111-111111111111';
const demoLocationId = '22222222-2222-2222-2222-222222222222';

void main() {
  const baseUrl = String.fromEnvironment('SEATD_API_BASE_URL');
  final gateway = baseUrl.isEmpty
      ? DemoDisplayGateway()
      : SeatdHttpDisplayGateway(baseUrl: Uri.parse(baseUrl));
  runApp(SeatdDisplayApp(gateway: gateway, store: const SecureDisplayStore()));
}

class SeatdDisplayApp extends StatelessWidget {
  const SeatdDisplayApp({
    super.key,
    required this.gateway,
    required this.store,
  });

  final DisplayGateway gateway;
  final DisplayStore store;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Seatd Display',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff1f6f55)),
        scaffoldBackgroundColor: const Color(0xfff4f5f2),
      ),
      home: DisplayScreen(gateway: gateway, store: store),
    );
  }
}

class DisplayScreen extends StatefulWidget {
  const DisplayScreen({super.key, required this.gateway, required this.store});

  final DisplayGateway gateway;
  final DisplayStore store;

  @override
  State<DisplayScreen> createState() => _DisplayScreenState();
}

class _DisplayScreenState extends State<DisplayScreen> {
  final _codeController = TextEditingController();
  Timer? _timer;
  String? _credential;
  LocationSnapshotResponse? _snapshot;
  bool _isOnline = false;
  bool _isPairing = false;
  bool _isSyncing = false;
  String? _message;
  DateTime? _lastSync;

  @override
  void initState() {
    super.initState();
    unawaited(_restore());
  }

  @override
  void dispose() {
    _timer?.cancel();
    _codeController.dispose();
    super.dispose();
  }

  Future<void> _restore() async {
    final credential = await widget.store.readCredential();
    final snapshot = await widget.store.readSnapshot();
    if (!mounted) {
      return;
    }
    setState(() {
      _credential = credential;
      _snapshot = snapshot;
      _lastSync = snapshot == null ? null : DateTime.now();
    });
    if (credential != null) {
      await _sync();
      _timer = Timer.periodic(const Duration(seconds: 30), (_) {
        unawaited(_sync());
      });
    }
  }

  Future<void> _pair() async {
    setState(() {
      _isPairing = true;
      _message = null;
    });
    try {
      final result = await widget.gateway.pair(_codeController.text);
      await widget.store.writeCredential(result.credential);
      setState(() {
        _credential = result.credential;
        _isOnline = true;
      });
      await _sync();
      _timer ??= Timer.periodic(const Duration(seconds: 30), (_) {
        unawaited(_sync());
      });
    } catch (_) {
      setState(() {
        _message = 'Pairing failed';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isPairing = false;
        });
      }
    }
  }

  Future<void> _sync() async {
    final credential = _credential;
    if (credential == null || _isSyncing) {
      return;
    }
    setState(() {
      _isSyncing = true;
    });
    try {
      await widget.gateway.heartbeat(credential);
      final snapshot = await widget.gateway.loadSnapshot(credential);
      await widget.store.writeSnapshot(snapshot);
      setState(() {
        _snapshot = snapshot;
        _isOnline = true;
        _lastSync = DateTime.now();
        _message = null;
      });
    } catch (_) {
      setState(() {
        _isOnline = false;
        _message = 'Offline: showing last known data';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isSyncing = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_credential == null) {
      return PairingView(
        controller: _codeController,
        isPairing: _isPairing,
        message: _message,
        onPair: _pair,
      );
    }
    return LiveDisplayView(
      snapshot: _snapshot,
      isOnline: _isOnline,
      isSyncing: _isSyncing,
      lastSync: _lastSync,
      message: _message,
      onRefresh: _sync,
    );
  }
}

class PairingView extends StatelessWidget {
  const PairingView({
    super.key,
    required this.controller,
    required this.isPairing,
    required this.message,
    required this.onPair,
  });

  final TextEditingController controller;
  final bool isPairing;
  final String? message;
  final VoidCallback onPair;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 460),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Seatd Display',
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 24),
                TextField(
                  controller: controller,
                  autofocus: true,
                  decoration: const InputDecoration(labelText: 'Pairing code'),
                  textCapitalization: TextCapitalization.characters,
                ),
                const SizedBox(height: 14),
                FilledButton(
                  onPressed: isPairing ? null : onPair,
                  child: Text(isPairing ? 'Pairing' : 'Pair display'),
                ),
                if (message != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    message!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class LiveDisplayView extends StatelessWidget {
  const LiveDisplayView({
    super.key,
    required this.snapshot,
    required this.isOnline,
    required this.isSyncing,
    required this.lastSync,
    required this.message,
    required this.onRefresh,
  });

  final LocationSnapshotResponse? snapshot;
  final bool isOnline;
  final bool isSyncing;
  final DateTime? lastSync;
  final String? message;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final tables = snapshot?.tables ?? const <TableState>[];
    final occupied = tables
        .where((table) => table.occupancy.status == 'occupied')
        .length;
    final available = tables.length - occupied;
    return Scaffold(
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, _) {
            return Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  DisplayHeader(
                    isOnline: isOnline,
                    isSyncing: isSyncing,
                    lastSync: lastSync,
                    onRefresh: onRefresh,
                  ),
                  if (message != null) OfflineBanner(message: message!),
                  const SizedBox(height: 18),
                  AvailabilitySummary(available: available, occupied: occupied),
                  const SizedBox(height: 18),
                  Expanded(child: FloorPlan(tables: tables)),
                ],
              ),
            );
          },
        ),
      ),
    );
  }
}

class DisplayHeader extends StatelessWidget {
  const DisplayHeader({
    super.key,
    required this.isOnline,
    required this.isSyncing,
    required this.lastSync,
    required this.onRefresh,
  });

  final bool isOnline;
  final bool isSyncing;
  final DateTime? lastSync;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Seatd Display',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              Text(
                isOnline ? 'Live' : 'Offline',
                style: TextStyle(
                  color: isOnline
                      ? const Color(0xff1f6f55)
                      : const Color(0xff8f4a12),
                ),
              ),
            ],
          ),
        ),
        Text(
          lastSync == null
              ? 'No sync yet'
              : 'Updated ${lastSync!.toLocal().toIso8601String().substring(11, 16)}',
        ),
        const SizedBox(width: 12),
        IconButton(
          onPressed: isSyncing ? null : onRefresh,
          icon: const Icon(Icons.refresh),
        ),
      ],
    );
  }
}

class OfflineBanner extends StatelessWidget {
  const OfflineBanner({super.key, required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xfffff8d8),
        border: Border.all(color: const Color(0xffd7b640)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(message),
    );
  }
}

class AvailabilitySummary extends StatelessWidget {
  const AvailabilitySummary({
    super.key,
    required this.available,
    required this.occupied,
  });

  final int available;
  final int occupied;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: SummaryTile(
            label: 'Available',
            value: available,
            color: const Color(0xff1f6f55),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: SummaryTile(
            label: 'Occupied',
            value: occupied,
            color: const Color(0xffbf5b38),
          ),
        ),
      ],
    );
  }
}

class SummaryTile extends StatelessWidget {
  const SummaryTile({
    super.key,
    required this.label,
    required this.value,
    required this.color,
  });

  final String label;
  final int value;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: Colors.white,
        border: Border.all(color: const Color(0xffd9ded6)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: Theme.of(context).textTheme.titleMedium),
          Text(
            '$value',
            style: Theme.of(
              context,
            ).textTheme.displaySmall?.copyWith(color: color),
          ),
        ],
      ),
    );
  }
}

class FloorPlan extends StatelessWidget {
  const FloorPlan({super.key, required this.tables});

  final List<TableState> tables;

  @override
  Widget build(BuildContext context) {
    if (tables.isEmpty) {
      return const Center(child: Text('No tables available'));
    }
    return Container(
      decoration: BoxDecoration(
        color: Colors.white,
        border: Border.all(color: const Color(0xffd9ded6)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: GridView.builder(
        padding: const EdgeInsets.all(14),
        gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
          maxCrossAxisExtent: 180,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 1.35,
        ),
        itemCount: tables.length,
        itemBuilder: (context, index) => DisplayTableTile(state: tables[index]),
      ),
    );
  }
}

class DisplayTableTile extends StatelessWidget {
  const DisplayTableTile({super.key, required this.state});

  final TableState state;

  @override
  Widget build(BuildContext context) {
    final occupied = state.occupancy.status == 'occupied';
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: occupied ? const Color(0xfffff2ec) : const Color(0xfff5fff8),
        border: Border.all(
          color: occupied ? const Color(0xffbf5b38) : const Color(0xff287b5f),
          width: 2,
        ),
        borderRadius: BorderRadius.circular(
          state.table.shape == 'circle' ? 90 : 8,
        ),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            state.table.label,
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          Text('Seats ${state.table.capacityLabel}'),
          Text(occupied ? 'Occupied' : 'Available'),
        ],
      ),
    );
  }
}

abstract class DisplayGateway {
  Future<PairDeviceResponse> pair(String code);
  Future<Device> heartbeat(String credential);
  Future<LocationSnapshotResponse> loadSnapshot(String credential);
}

class SeatdHttpDisplayGateway implements DisplayGateway {
  SeatdHttpDisplayGateway({required this.baseUrl, http.Client? client})
    : _client = client ?? http.Client();

  final Uri baseUrl;
  final http.Client _client;

  @override
  Future<PairDeviceResponse> pair(String code) async {
    final response = await _client.post(
      baseUrl.resolve(pairDeviceEndpoint),
      headers: const {'Content-Type': 'application/json'},
      body: jsonEncode(
        PairDeviceRequest(
          code: code,
          platform: 'android',
          appVersion: displayAppVersion,
          name: 'Dining room display',
          capabilities: const {'rotation': true, 'offlineCache': true},
        ).toJson(),
      ),
    );
    return PairDeviceResponse.fromJson(_decodeSuccess(response));
  }

  @override
  Future<Device> heartbeat(String credential) async {
    final response = await _client.post(
      baseUrl.resolve(deviceHeartbeatEndpoint),
      headers: {
        'Authorization': 'Bearer $credential',
        'Content-Type': 'application/json',
      },
      body: jsonEncode(
        const DeviceHeartbeatRequest(
          appVersion: displayAppVersion,
          capabilities: {'rotation': true, 'offlineCache': true},
        ).toJson(),
      ),
    );
    return Device.fromJson(_decodeSuccess(response));
  }

  @override
  Future<LocationSnapshotResponse> loadSnapshot(String credential) async {
    final response = await _client.get(
      baseUrl.resolve(displaySnapshotEndpoint),
      headers: {'Authorization': 'Bearer $credential'},
    );
    return LocationSnapshotResponse.fromJson(_decodeSuccess(response));
  }

  Map<String, Object?> _decodeSuccess(http.Response response) {
    final body = jsonDecode(response.body) as Map<String, Object?>;
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return body;
    }
    throw StateError(SeatdErrorResponse.fromJson(body).error.message);
  }
}

class DemoDisplayGateway implements DisplayGateway {
  @override
  Future<PairDeviceResponse> pair(String code) async => PairDeviceResponse(
    credential: 'demo-credential',
    device: Device(
      id: '66666666-6666-6666-6666-666666666666',
      organisationId: demoOrganisationId,
      locationId: demoLocationId,
      name: 'Demo display',
      deviceType: 'display',
      platform: 'android',
      appVersion: displayAppVersion,
      trustState: 'trusted',
      registeredAt: DateTime.now().toUtc().toIso8601String(),
      heartbeatIntervalSeconds: 60,
    ),
  );

  @override
  Future<Device> heartbeat(String credential) async => Device(
    id: '66666666-6666-6666-6666-666666666666',
    organisationId: demoOrganisationId,
    locationId: demoLocationId,
    name: 'Demo display',
    deviceType: 'display',
    platform: 'android',
    appVersion: displayAppVersion,
    trustState: 'trusted',
    registeredAt: DateTime.now().toUtc().toIso8601String(),
    lastHeartbeatAt: DateTime.now().toUtc().toIso8601String(),
    heartbeatIntervalSeconds: 60,
  );

  @override
  Future<LocationSnapshotResponse> loadSnapshot(String credential) async {
    final now = DateTime.now().toUtc().toIso8601String();
    return LocationSnapshotResponse(
      organisationId: demoOrganisationId,
      locationId: demoLocationId,
      cursor: 'display-demo',
      tables: [
        _table(
          '55555555-5555-5555-5555-555555555551',
          'T1',
          '4',
          'available',
          now,
        ),
        _table(
          '55555555-5555-5555-5555-555555555552',
          'T2',
          '2',
          'occupied',
          now,
        ),
        _table(
          '55555555-5555-5555-5555-555555555553',
          'T3',
          '6',
          'available',
          now,
        ),
        _table(
          '55555555-5555-5555-5555-555555555554',
          'Bar 1',
          '2',
          'occupied',
          now,
        ),
      ],
      assists: const [],
    );
  }

  TableState _table(
    String id,
    String label,
    String capacity,
    String status,
    String updatedAt,
  ) => TableState(
    table: SeatdTable(
      id: id,
      floorId: '33333333-3333-3333-3333-333333333333',
      zoneId: '44444444-4444-4444-4444-444444444441',
      label: label,
      capacityLabel: capacity,
      shape: 'rectangle',
      geometry: const {},
      version: 1,
    ),
    occupancy: Occupancy(status: status, version: 1, updatedAt: updatedAt),
  );
}

abstract class DisplayStore {
  Future<String?> readCredential();
  Future<void> writeCredential(String credential);
  Future<LocationSnapshotResponse?> readSnapshot();
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot);
}

class SecureDisplayStore implements DisplayStore {
  const SecureDisplayStore();

  static const _storage = FlutterSecureStorage();
  static const _credentialKey = 'seatd.display.credential';
  static const _snapshotKey = 'seatd.display.snapshot';

  @override
  Future<String?> readCredential() => _storage.read(key: _credentialKey);

  @override
  Future<void> writeCredential(String credential) =>
      _storage.write(key: _credentialKey, value: credential);

  @override
  Future<LocationSnapshotResponse?> readSnapshot() async {
    final encoded = await _storage.read(key: _snapshotKey);
    if (encoded == null) {
      return null;
    }
    return LocationSnapshotResponse.fromJson(
      jsonDecode(encoded) as Map<String, Object?>,
    );
  }

  @override
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot) =>
      _storage.write(
        key: _snapshotKey,
        value: jsonEncode({
          'organisationId': snapshot.organisationId,
          'locationId': snapshot.locationId,
          'cursor': snapshot.cursor,
          'tables': snapshot.tables.map(_tableStateJson).toList(),
          'assists': snapshot.assists.map(_assistJson).toList(),
        }),
      );
}

Map<String, Object?> _tableStateJson(TableState state) => {
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

Map<String, Object?> _assistJson(Assist assist) => {
  'id': assist.id,
  'tableId': assist.tableId,
  if (assist.tableSessionId != null) 'tableSessionId': assist.tableSessionId,
  'status': assist.status,
  'requestedAt': assist.requestedAt,
  'version': assist.version,
  if (assist.note != null) 'note': assist.note,
};
