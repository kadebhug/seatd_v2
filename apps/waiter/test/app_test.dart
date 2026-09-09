import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:seatd_waiter/data/waiter_session_store.dart';
import 'package:seatd_waiter/data/waiter_gateway.dart';
import 'package:seatd_waiter/main.dart';

void main() {
  testWidgets('renders floor operations from cached snapshot', (tester) async {
    await tester.pumpWidget(_app());
    await tester.pumpAndSettle();

    expect(find.text('Seatd Waiter'), findsOneWidget);
    expect(find.text('Ground Floor'), findsOneWidget);
    expect(find.text('Dining Room'), findsOneWidget);
    expect(find.text('T1'), findsOneWidget);
    expect(find.text('Available'), findsWidgets);
    expect(find.text('Occupied'), findsOneWidget);
  });

  testWidgets('optimistically occupies available table', (tester) async {
    await tester.pumpWidget(_app());
    await tester.pumpAndSettle();

    await tester.tap(find.text('T1'));
    await tester.pumpAndSettle();

    expect(find.text('Occupied'), findsNWidgets(2));
  });

  testWidgets('pairs an untrusted device before showing floor', (tester) async {
    await tester.pumpWidget(
      SeatdWaiterApp(
        gateway: DemoWaiterGateway(),
        sessionStore: MemoryWaiterSessionStore(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Pair device'), findsOneWidget);

    await tester.enterText(find.byType(EditableText), 'ABCD1234');
    await tester.tap(find.text('Pair device'));
    await tester.pumpAndSettle();

    expect(find.text('Ground Floor'), findsOneWidget);
  });

  testWidgets('opens command queue for a conflict', (tester) async {
    await tester.pumpWidget(_app());
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Command queue'));
    await tester.pumpAndSettle();

    expect(find.text('Command queue'), findsOneWidget);
    expect(find.text('No queued commands'), findsOneWidget);
  });
}

SeatdWaiterApp _app() => SeatdWaiterApp(
  gateway: DemoWaiterGateway(),
  sessionStore: MemoryWaiterSessionStore(session: demoWaiterSession),
);
