import 'package:flutter_test/flutter_test.dart';
import 'package:seatd_waiter/data/waiter_gateway.dart';
import 'package:seatd_waiter/main.dart';

void main() {
  testWidgets('renders floor operations from cached snapshot', (tester) async {
    await tester.pumpWidget(SeatdWaiterApp(gateway: DemoWaiterGateway()));
    await tester.pumpAndSettle();

    expect(find.text('Seatd Waiter'), findsOneWidget);
    expect(find.text('Ground Floor'), findsOneWidget);
    expect(find.text('Dining Room'), findsOneWidget);
    expect(find.text('T1'), findsOneWidget);
    expect(find.text('Available'), findsWidgets);
    expect(find.text('Occupied'), findsOneWidget);
  });

  testWidgets('optimistically occupies available table', (tester) async {
    await tester.pumpWidget(SeatdWaiterApp(gateway: DemoWaiterGateway()));
    await tester.pumpAndSettle();

    await tester.tap(find.text('T1'));
    await tester.pumpAndSettle();

    expect(find.text('Occupied'), findsNWidgets(2));
  });
}
