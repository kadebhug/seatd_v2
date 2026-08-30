import 'package:flutter_test/flutter_test.dart';
import 'package:seatd_client/seatd_client.dart';
import 'package:seatd_waiter/data/local_waiter_store.dart';
import 'package:seatd_waiter/domain/waiter_models.dart';

void main() {
  test('snapshot merge stores tables, assists, cursor, and sync timestamp', () {
    final store = LocalWaiterStore();

    store.mergeSnapshot(_snapshot(tableVersion: 1, status: 'available'));

    expect(store.state.cursor, 'cursor-1');
    expect(store.state.tables.single.occupancy.status, 'available');
    expect(store.state.assists.single.status, 'pending');
    expect(store.state.lastSuccessfulSync, isNotNull);
  });

  test('realtime dedupes event ids and ignores stale table versions', () {
    final store = LocalWaiterStore();
    store.mergeSnapshot(_snapshot(tableVersion: 2, status: 'occupied'));

    final stale = RealtimeMessage(
      type: 'table.cleared',
      eventId: 'event-1',
      organisationId: demoOrganisationId,
      locationId: demoLocationId,
      entityType: 'table',
      entityId: _tableId,
      version: 1,
      occurredAt: '2026-08-30T10:00:00Z',
      data: const {'status': 'available'},
    );

    expect(store.applyRealtimeMessage(stale), isFalse);
    expect(store.applyRealtimeMessage(stale), isFalse);
    expect(store.state.tables.single.occupancy.status, 'occupied');
  });

  test('optimistic occupy queues command and keeps pending state', () {
    final store = LocalWaiterStore();
    store.mergeSnapshot(_snapshot(tableVersion: 1, status: 'available'));

    final command = store.occupyTable(store.state.tables.single);

    expect(command.type, CommandType.occupyTable);
    expect(store.state.tables.single.occupancy.status, 'occupied');
    expect(store.state.commands.single.state, CommandState.queued);
  });

  test('stale clear conflict is surfaced for waiter review', () {
    final store = LocalWaiterStore();
    store.mergeSnapshot(_snapshot(tableVersion: 2, status: 'occupied'));
    final command = store.clearTable(store.state.tables.single);

    store.classifyCommandFailure(
      command.id,
      const SeatdApiError(
        code: SeatdErrorCode.versionConflict,
        message: 'entity version conflict',
      ),
    );

    expect(store.state.commands.single.state, CommandState.conflict);
  });
}

const _tableId = '55555555-5555-5555-5555-555555555551';

LocationSnapshotResponse _snapshot({
  required int tableVersion,
  required String status,
}) {
  return LocationSnapshotResponse(
    organisationId: demoOrganisationId,
    locationId: demoLocationId,
    cursor: 'cursor-1',
    tables: [
      TableState(
        table: const SeatdTable(
          id: _tableId,
          floorId: '33333333-3333-3333-3333-333333333333',
          zoneId: '44444444-4444-4444-4444-444444444441',
          label: 'T1',
          capacityLabel: '4',
          shape: 'rectangle',
          geometry: {'x': 1, 'y': 1, 'width': 1, 'height': 1},
          version: 1,
        ),
        occupancy: Occupancy(
          status: status,
          currentSessionId: status == 'occupied' ? 'session-1' : null,
          version: tableVersion,
          updatedAt: '2026-08-30T09:00:00Z',
        ),
      ),
    ],
    assists: const [
      Assist(
        id: '77777777-7777-7777-7777-777777777777',
        tableId: _tableId,
        status: 'pending',
        requestedAt: '2026-08-30T09:30:00Z',
        version: 1,
      ),
    ],
  );
}
