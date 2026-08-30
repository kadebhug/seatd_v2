import 'package:flutter/material.dart';

import 'data/local_waiter_store.dart';
import 'data/waiter_gateway.dart';
import 'ui/core/waiter_theme.dart';
import 'ui/features/floor/view_models/floor_view_model.dart';
import 'ui/features/floor/views/floor_screen.dart';

void main() {
  const baseUrl = String.fromEnvironment('SEATD_API_BASE_URL');
  final gateway = baseUrl.isEmpty
      ? DemoWaiterGateway()
      : SeatdHttpWaiterGateway(baseUrl: Uri.parse(baseUrl));
  runApp(SeatdWaiterApp(gateway: gateway));
}

class SeatdWaiterApp extends StatelessWidget {
  const SeatdWaiterApp({super.key, required this.gateway});

  final WaiterGateway gateway;

  @override
  Widget build(BuildContext context) {
    final store = LocalWaiterStore();
    return MaterialApp(
      title: 'Seatd Waiter',
      theme: buildWaiterTheme(),
      home: FloorScreen(
        viewModel: FloorViewModel(
          store: store,
          gateway: gateway,
          commandProcessor: CommandProcessor(store: store, gateway: gateway),
        ),
      ),
    );
  }
}
