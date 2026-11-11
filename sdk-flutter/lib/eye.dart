import 'dart:convert';
import 'dart:io';
import 'dart:math';

class EyeResponse {
  final int statusCode;
  final String body;
  EyeResponse(this.statusCode, this.body);
}

abstract class EyeTransport {
  Future<EyeResponse> post(String url, Map<String, dynamic> body, Map<String, String> headers);
}

class HttpEyeTransport implements EyeTransport {
  final HttpClient _client;
  HttpEyeTransport([HttpClient? client]) : _client = client ?? HttpClient();

  @override
  Future<EyeResponse> post(String url, Map<String, dynamic> body, Map<String, String> headers) async {
    final request = await _client.postUrl(Uri.parse(url));
    headers.forEach(request.headers.set);
    request.headers.contentType = ContentType.json;
    request.write(jsonEncode(body));
    final response = await request.close();
    return EyeResponse(response.statusCode, await response.transform(utf8.decoder).join());
  }
}

class EyeDiagnostics {
  final String tenantId;
  final String? sessionId;
  final int queuedEvents;
  final int droppedEvents;
  final int? lastFlushAt;
  final String? lastError;
  const EyeDiagnostics({required this.tenantId, required this.sessionId, required this.queuedEvents, required this.droppedEvents, required this.lastFlushAt, required this.lastError});
}

class EyeClient {
  final String endpoint;
  final String apiKey;
  final String tenantId;
  final String queuePath;
  final int maxQueueSize;
  final EyeTransport transport;
  Map<String, dynamic> _state = {'sessionId': null, 'events': <dynamic>[], 'droppedEvents': 0, 'lastFlushAt': null, 'lastError': null};
  bool _loaded = false;

  EyeClient({required this.endpoint, required this.apiKey, required this.tenantId, required this.queuePath, this.maxQueueSize = 10000, EyeTransport? transport}) : transport = transport ?? HttpEyeTransport();

  Future<void> _load() async { if (_loaded) return; final file = File(queuePath); if (await file.exists()) _state = jsonDecode(await file.readAsString()) as Map<String, dynamic>; _loaded = true; }
  Future<void> _save() async { final file = File(queuePath); await file.parent.create(recursive: true); final temp = File('${file.path}.${pid}.tmp'); await temp.writeAsString(jsonEncode(_state)); await temp.rename(file.path); }

  Future<String> ensureSession() async { await _load(); final cached = _state['sessionId'] as String?; if (cached != null && cached.isNotEmpty) return cached; final response = await transport.post('$endpoint/sessions', {'tenant_id': tenantId, 'device': {'runtime': 'flutter'}}, {'x-api-key': apiKey}); if (response.statusCode >= 300) throw StateError('session creation failed: HTTP ${response.statusCode}'); final data = jsonDecode(response.body) as Map<String, dynamic>; final id = data['session_id'] ?? data['ID'] ?? data['id']; if (id is! String || id.isEmpty) throw StateError('session creation returned no id'); _state['sessionId'] = id; await _save(); return id; }

  Future<void> track(String eventName, {Map<String, dynamic> payload = const {}, Map<String, dynamic> dims = const {}}) async { await _load(); if (eventName.trim().isEmpty) throw ArgumentError('event name is required'); final session = _state['sessionId']; if (session is! String || session.isEmpty) throw StateError('call ensureSession first'); final events = (_state['events'] as List<dynamic>); events.add({'event_id': '${DateTime.now().microsecondsSinceEpoch}-${Random().nextInt(1 << 32)}', 'session_id': session, 'event_name': eventName, 'client_ts': DateTime.now().millisecondsSinceEpoch, 'payload': payload, 'dims': dims}); if (events.length > maxQueueSize) { events.removeRange(0, events.length - maxQueueSize); _state['droppedEvents'] = (_state['droppedEvents'] as int) + 1; } await _save(); }

  Future<int> flush() async { await _load(); final events = List<dynamic>.from(_state['events'] as List<dynamic>); if (events.isEmpty) return 0; try { final response = await transport.post('$endpoint/ingest', {'events': events}, {'x-api-key': apiKey}); if (response.statusCode >= 300) throw StateError('ingest failed: HTTP ${response.statusCode}'); (_state['events'] as List<dynamic>).removeRange(0, events.length); _state['lastFlushAt'] = DateTime.now().millisecondsSinceEpoch; _state['lastError'] = null; await _save(); return events.length; } catch (error) { _state['lastError'] = error.toString(); await _save(); rethrow; } }

  Future<EyeDiagnostics> diagnostics() async { await _load(); return EyeDiagnostics(tenantId: tenantId, sessionId: _state['sessionId'] as String?, queuedEvents: (_state['events'] as List<dynamic>).length, droppedEvents: _state['droppedEvents'] as int, lastFlushAt: _state['lastFlushAt'] as int?, lastError: _state['lastError'] as String?); }
}
