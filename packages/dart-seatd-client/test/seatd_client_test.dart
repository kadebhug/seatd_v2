import 'package:seatd_client/seatd_client.dart';
import 'package:test/test.dart';

void main() {
  test('parses service status', () {
    final status = ServiceStatus.fromJsonString('''
{
  "service": "api",
  "environment": "test",
  "status": "ok",
  "version": {
    "version": "dev",
    "commit": "abc123",
    "date": "2026-08-29"
  }
}
''');

    expect(status.service, 'api');
    expect(status.environment, SeatdEnvironment.test);
    expect(status.version.commit, 'abc123');
  });
}
