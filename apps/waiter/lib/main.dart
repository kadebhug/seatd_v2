import 'package:flutter/material.dart';

import 'data/local_waiter_store.dart';
import 'data/waiter_session_store.dart';
import 'data/waiter_gateway.dart';
import 'domain/waiter_models.dart';
import 'ui/core/waiter_theme.dart';
import 'ui/features/floor/view_models/floor_view_model.dart';
import 'ui/features/floor/views/floor_screen.dart';

void main() {
  const apiBaseUrl = String.fromEnvironment('SEATD_API_BASE_URL');
  const realtimeBaseUrl = String.fromEnvironment('SEATD_REALTIME_BASE_URL');
  final demoMode = apiBaseUrl.isEmpty && realtimeBaseUrl.isEmpty;
  if (!demoMode && (apiBaseUrl.isEmpty || realtimeBaseUrl.isEmpty)) {
    runApp(const WaiterConfigErrorApp());
    return;
  }
  final gateway = demoMode
      ? DemoWaiterGateway()
      : SeatdHttpWaiterGateway(
          apiBaseUrl: Uri.parse(apiBaseUrl),
          realtimeBaseUrl: Uri.parse(realtimeBaseUrl),
        );
  runApp(
    SeatdWaiterApp(
      gateway: gateway,
      sessionStore: demoMode
          ? MemoryWaiterSessionStore(session: demoWaiterSession)
          : const SecureWaiterSessionStore(),
    ),
  );
}

class WaiterConfigErrorApp extends StatelessWidget {
  const WaiterConfigErrorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Seatd Waiter',
      theme: buildWaiterTheme(),
      home: const Scaffold(
        body: SafeArea(
          child: Center(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: Text(
                'SEATD_API_BASE_URL and SEATD_REALTIME_BASE_URL are required.',
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class SeatdWaiterApp extends StatelessWidget {
  const SeatdWaiterApp({
    super.key,
    required this.gateway,
    required this.sessionStore,
  });

  final WaiterGateway gateway;
  final WaiterSessionStore sessionStore;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Seatd Waiter',
      theme: buildWaiterTheme(),
      home: WaiterAppShell(gateway: gateway, sessionStore: sessionStore),
    );
  }
}

class WaiterAppShell extends StatefulWidget {
  const WaiterAppShell({
    super.key,
    required this.gateway,
    required this.sessionStore,
  });

  final WaiterGateway gateway;
  final WaiterSessionStore sessionStore;

  @override
  State<WaiterAppShell> createState() => _WaiterAppShellState();
}

class _WaiterAppShellState extends State<WaiterAppShell> {
  WaiterSession? _session;
  bool _isRestoring = true;
  bool _authExpired = false;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final session = await widget.sessionStore.readSession();
    if (!mounted) {
      return;
    }
    setState(() {
      _session = session;
      _isRestoring = false;
    });
  }

  Future<void> _pair(String code) async {
    final result = await widget.gateway.pairDevice(code);
    final session = WaiterSession.fromPairing(
      credential: result.credential,
      device: result.device,
    );
    await widget.sessionStore.writeSession(session);
    if (widget.gateway is SeatdHttpWaiterGateway) {
      (widget.gateway as SeatdHttpWaiterGateway).session = session;
    }
    if (!mounted) {
      return;
    }
    setState(() {
      _session = session;
      _authExpired = false;
    });
  }

  Future<void> _forgetDevice() async {
    await widget.sessionStore.clear();
    if (!mounted) {
      return;
    }
    setState(() {
      _session = null;
      _authExpired = false;
    });
  }

  void _markAuthExpired() {
    if (!mounted) {
      return;
    }
    setState(() {
      _session = null;
      _authExpired = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_isRestoring) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final session = _session;
    if (session == null) {
      return PairingScreen(authExpired: _authExpired, onPair: _pair);
    }
    final store = LocalWaiterStore();
    return FloorScreen(
      viewModel: FloorViewModel(
        store: store,
        gateway: widget.gateway,
        commandProcessor: CommandProcessor(
          store: store,
          gateway: widget.gateway,
        ),
        session: session,
        sessionStore: widget.sessionStore,
        onAuthExpired: _markAuthExpired,
      ),
      onForgetDevice: _forgetDevice,
    );
  }
}

class PairingScreen extends StatefulWidget {
  const PairingScreen({
    super.key,
    required this.authExpired,
    required this.onPair,
  });

  final bool authExpired;
  final Future<void> Function(String code) onPair;

  @override
  State<PairingScreen> createState() => _PairingScreenState();
}

class _PairingScreenState extends State<PairingScreen> {
  final _codeController = TextEditingController();
  bool _isPairing = false;
  String? _message;

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  Future<void> _pair() async {
    setState(() {
      _isPairing = true;
      _message = null;
    });
    try {
      await widget.onPair(_codeController.text);
    } catch (_) {
      if (!mounted) {
        return;
      }
      setState(() {
        _message = 'Pairing failed. Check the code and try again.';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isPairing = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 460),
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'Seatd Waiter',
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                  const SizedBox(height: 24),
                  TextField(
                    controller: _codeController,
                    autofocus: true,
                    decoration: const InputDecoration(
                      labelText: 'Pairing code',
                    ),
                    textCapitalization: TextCapitalization.characters,
                    onSubmitted: (_) => _isPairing ? null : _pair(),
                  ),
                  const SizedBox(height: 14),
                  FilledButton.icon(
                    onPressed: _isPairing ? null : _pair,
                    icon: _isPairing
                        ? const SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.phonelink_lock),
                    label: Text(_isPairing ? 'Pairing' : 'Pair device'),
                  ),
                  if (widget.authExpired || _message != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      _message ??
                          'Device access expired. Pair again to continue.',
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
      ),
    );
  }
}

const demoWaiterSession = WaiterSession(
  credential: 'demo-waiter-credential',
  deviceId: demoDeviceId,
  organisationId: demoOrganisationId,
  locationId: demoLocationId,
  deviceName: 'Demo waiter',
  heartbeatIntervalSeconds: 60,
);
