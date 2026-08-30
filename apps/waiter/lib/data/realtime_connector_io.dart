import 'package:seatd_client/seatd_client.dart';
import 'package:web_socket_channel/io.dart';

Stream<RealtimeMessage> connectRealtime(
  Uri baseUrl,
  Map<String, String> headers,
) {
  final wsUrl = baseUrl.replace(
    scheme: baseUrl.scheme == 'https' ? 'wss' : 'ws',
    path: realtimeEndpoint,
  );
  final channel = IOWebSocketChannel.connect(wsUrl, headers: headers);
  return channel.stream.map(
    (event) => RealtimeMessage.fromJsonString(event as String),
  );
}
