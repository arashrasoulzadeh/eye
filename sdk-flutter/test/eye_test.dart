import 'dart:io';
import 'package:eye_sdk/eye.dart';

class FakeTransport implements EyeTransport {
  int ingestCalls = 0;
  @override
  Future<EyeResponse> post(String url, Map<String, dynamic> body, Map<String, String> headers) async {
    if (url.endsWith('/sessions')) return EyeResponse(201, '{"ID":"s1"}');
    ingestCalls++;
    return EyeResponse(ingestCalls == 1 ? 503 : 202, '');
  }
}

Future<void> main() async {
  void check(bool value, String message) { if (!value) throw StateError(message); }
    final path = '${Directory.systemTemp.path}/eye-flutter-${DateTime.now().microsecondsSinceEpoch}.json';
    final transport = FakeTransport();
    final client = EyeClient(endpoint: 'https://eye.test', apiKey: 'key', tenantId: 'tenant', queuePath: path, transport: transport);
    check(await client.ensureSession() == 's1', 'session was not created');
    await client.track('purchase');
    var failed = false;
    try { await client.flush(); } on StateError { failed = true; }
    check(failed, 'failed flush did not fail');
    check((await client.diagnostics()).queuedEvents == 1, 'failed flush lost events');
    check(await client.flush() == 1, 'retry did not flush one event');
    check((await client.diagnostics()).queuedEvents == 0, 'queue was not cleared');
    final file = File(path); if (await file.exists()) await file.delete();
    print('Eye Flutter SDK contract test passed');
  }
