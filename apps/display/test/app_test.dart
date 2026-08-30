import 'package:flutter_test/flutter_test.dart';
import 'package:seatd_client/seatd_client.dart';
import 'package:seatd_display/main.dart';

const testOrganisationId = '11111111-1111-1111-1111-111111111111';
const testLocationId = '22222222-2222-2222-2222-222222222222';

void main() {
  testWidgets('renders pairing screen before credential exists', (
    tester,
  ) async {
    await tester.pumpWidget(
      SeatdDisplayApp(
        gateway: DemoDisplayGateway(),
        store: MemoryDisplayStore(),
      ),
    );
    await tester.pump();

    expect(find.text('Seatd Display'), findsOneWidget);
    expect(find.text('Pair display'), findsOneWidget);
  });

  testWidgets('renders cached display state when credential exists', (
    tester,
  ) async {
    final store = MemoryDisplayStore(
      credential: 'demo-credential',
      snapshot: LocationSnapshotResponse(
        organisationId: testOrganisationId,
        locationId: testLocationId,
        cursor: 'test',
        tables: [
          TableState(
            table: const SeatdTable(
              id: '55555555-5555-5555-5555-555555555551',
              floorId: '33333333-3333-3333-3333-333333333333',
              zoneId: '44444444-4444-4444-4444-444444444441',
              label: 'T1',
              capacityLabel: '4',
              shape: 'rectangle',
              geometry: {},
              version: 1,
            ),
            occupancy: Occupancy(
              status: 'available',
              version: 1,
              updatedAt: DateTime.now().toUtc().toIso8601String(),
            ),
          ),
        ],
        assists: const [],
      ),
    );

    await tester.pumpWidget(
      SeatdDisplayApp(gateway: DemoDisplayGateway(), store: store),
    );
    await tester.pumpAndSettle();

    expect(find.text('Available'), findsWidgets);
    expect(find.text('T1'), findsOneWidget);
  });
}

class MemoryDisplayStore implements DisplayStore {
  MemoryDisplayStore({String? credential, LocationSnapshotResponse? snapshot})
    : _credential = credential,
      _snapshot = snapshot;

  String? _credential;
  LocationSnapshotResponse? _snapshot;

  @override
  Future<String?> readCredential() async => _credential;

  @override
  Future<void> writeCredential(String credential) async {
    _credential = credential;
  }

  @override
  Future<LocationSnapshotResponse?> readSnapshot() async => _snapshot;

  @override
  Future<void> writeSnapshot(LocationSnapshotResponse snapshot) async {
    _snapshot = snapshot;
  }
}
