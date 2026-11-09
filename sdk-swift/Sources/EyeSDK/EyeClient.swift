import Foundation

/// A Swift port of web-sdk's `EyeClient`: session persistence (via a
/// pluggable key-value store, `UserDefaults` by default on iOS), a bounded
/// ring buffer, and size/time-threshold flush batching against the eye
/// collector's `/sessions` and `/ingest` REST endpoints.
public struct EyeEvent: Codable {
    public let eventId: String
    public let sessionId: String
    public let eventName: String
    public let clientTs: Int64
    public let payload: [String: AnyCodable]
    public let dims: [String: AnyCodable]

    enum CodingKeys: String, CodingKey {
        case eventId = "event_id"
        case sessionId = "session_id"
        case eventName = "event_name"
        case clientTs = "client_ts"
        case payload, dims
    }
}

/// Minimal `Codable` wrapper so payload/dims can hold heterogeneous JSON
/// values, mirroring the TS SDK's `Record<string, any>`.
public struct AnyCodable: Codable {
    public let value: Any

    public init(_ value: Any) { self.value = value }

    public init(from decoder: Decoder) throws {
        let c = try decoder.singleValueContainer()
        if let v = try? c.decode(String.self) { value = v }
        else if let v = try? c.decode(Int.self) { value = v }
        else if let v = try? c.decode(Double.self) { value = v }
        else if let v = try? c.decode(Bool.self) { value = v }
        else { value = NSNull() }
    }

    public func encode(to encoder: Encoder) throws {
        var c = encoder.singleValueContainer()
        switch value {
        case let v as String: try c.encode(v)
        case let v as Int: try c.encode(v)
        case let v as Double: try c.encode(v)
        case let v as Bool: try c.encode(v)
        default: try c.encodeNil()
        }
    }
}

/// Key-value store abstraction so tests (and non-iOS targets) don't depend
/// on `UserDefaults`. `UserDefaults.standard` conforms to this out of the box.
public protocol KeyValueStore {
    func string(forKey key: String) -> String?
    func set(_ value: String, forKey key: String)
}

/// `UserDefaults.set(_:forKey:)` takes `Any?`, not `String`, so it needs an
/// explicit shim rather than a bare protocol conformance.
public final class UserDefaultsStore: KeyValueStore {
    private let defaults: UserDefaults
    public init(_ defaults: UserDefaults = .standard) { self.defaults = defaults }
    public func string(forKey key: String) -> String? { defaults.string(forKey: key) }
    public func set(_ value: String, forKey key: String) { defaults.set(value, forKey: key) }
}

public final class EyeClient {
    public struct Options {
        public var endpoint: String
        public var apiKey: String
        public var tenantId: String
        public var storageKey: String = "eye_session_id"
        public var maxBufferSize: Int = 1000
        public var batchSize: Int = 20
        public var flushIntervalSeconds: Double = 5.0

        public init(endpoint: String, apiKey: String, tenantId: String) {
            self.endpoint = endpoint
            self.apiKey = apiKey
            self.tenantId = tenantId
        }
    }

    private let opts: Options
    private let store: KeyValueStore
    private let session: URLSession
    private let queue = DispatchQueue(label: "eyesdk.client")

    private var buffer: [EyeEvent] = []
    private var sessionId: String?
    private var timer: Timer?

    public init(options: Options, store: KeyValueStore = UserDefaultsStore(), urlSession: URLSession = .shared) {
        self.opts = options
        self.store = store
        self.session = urlSession
        self.sessionId = store.string(forKey: options.storageKey)

        if options.flushIntervalSeconds > 0 {
            let t = Timer(timeInterval: options.flushIntervalSeconds, repeats: true) { [weak self] _ in
                self?.flush { _ in }
            }
            RunLoop.main.add(t, forMode: .common)
            self.timer = t
        }
    }

    deinit {
        timer?.invalidate()
    }

    /// Stops the background flush timer.
    public func destroy() {
        timer?.invalidate()
        timer = nil
    }

    /// Returns the current buffered event count (for tests/diagnostics).
    public var bufferCount: Int {
        queue.sync { buffer.count }
    }

    public enum EyeError: Error {
        case noSession
        case http(Int)
        case invalidResponse
    }

    /// Creates a session via POST /sessions if none is cached, persisting
    /// the id in the configured store (UserDefaults by default), matching
    /// web-sdk's localStorage persistence.
    public func ensureSession(completion: @escaping (Result<String, Error>) -> Void) {
        if let sid = queue.sync(execute: { sessionId }) {
            completion(.success(sid))
            return
        }
        guard let url = URL(string: opts.endpoint + "/sessions") else {
            completion(.failure(EyeError.invalidResponse))
            return
        }
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "content-type")
        req.setValue(opts.apiKey, forHTTPHeaderField: "x-api-key")
        req.httpBody = try? JSONEncoder().encode(["tenant_id": opts.tenantId])

        session.dataTask(with: req) { [weak self] data, resp, err in
            guard let self = self else { return }
            if let err = err { completion(.failure(err)); return }
            guard let http = resp as? HTTPURLResponse, http.statusCode < 300, let data = data else {
                completion(.failure(EyeError.invalidResponse))
                return
            }
            struct SessionResp: Decodable { let ID: String }
            guard let decoded = try? JSONDecoder().decode(SessionResp.self, from: data) else {
                completion(.failure(EyeError.invalidResponse))
                return
            }
            self.queue.sync { self.sessionId = decoded.ID }
            self.store.set(decoded.ID, forKey: self.opts.storageKey)
            completion(.success(decoded.ID))
        }.resume()
    }

    /// Lets a caller seed/restore a session id directly.
    public func setSessionId(_ id: String) {
        queue.sync { sessionId = id }
        store.set(id, forKey: opts.storageKey)
    }

    /// Buffers an event; evicts the oldest event once `maxBufferSize` is
    /// reached (ring buffer), and triggers a background flush once
    /// `batchSize` is reached (best-effort: errors are swallowed and the
    /// events stay buffered for the next attempt, matching web-sdk).
    @discardableResult
    public func track(name: String, payload: [String: AnyCodable] = [:], dims: [String: AnyCodable] = [:]) -> Bool {
        var shouldFlush = false
        var ok = true
        queue.sync {
            guard let sid = sessionId else { ok = false; return }
            let ev = EyeEvent(
                eventId: UUID().uuidString,
                sessionId: sid,
                eventName: name,
                clientTs: Int64(Date().timeIntervalSince1970 * 1000),
                payload: payload,
                dims: dims
            )
            if buffer.count >= opts.maxBufferSize {
                buffer.removeFirst()
            }
            buffer.append(ev)
            shouldFlush = opts.batchSize > 0 && buffer.count >= opts.batchSize
        }
        if !ok { return false }
        if shouldFlush {
            flush { _ in }
        }
        return true
    }

    /// POSTs all buffered events to /ingest and clears the buffer on
    /// success; re-queues them (trimmed to maxBufferSize) on failure.
    public func flush(completion: @escaping (Result<Void, Error>) -> Void) {
        let events: [EyeEvent] = queue.sync {
            let e = buffer
            buffer = []
            return e
        }
        guard !events.isEmpty else { completion(.success(())); return }
        guard let url = URL(string: opts.endpoint + "/ingest") else {
            requeue(events)
            completion(.failure(EyeError.invalidResponse))
            return
        }
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "content-type")
        req.setValue(opts.apiKey, forHTTPHeaderField: "x-api-key")
        guard let body = try? JSONEncoder().encode(["events": events]) else {
            requeue(events)
            completion(.failure(EyeError.invalidResponse))
            return
        }
        req.httpBody = body

        session.dataTask(with: req) { [weak self] _, resp, err in
            guard let self = self else { return }
            if let err = err {
                self.requeue(events)
                completion(.failure(err))
                return
            }
            guard let http = resp as? HTTPURLResponse, http.statusCode < 300 else {
                self.requeue(events)
                let code = (resp as? HTTPURLResponse)?.statusCode ?? -1
                completion(.failure(EyeError.http(code)))
                return
            }
            completion(.success(()))
        }.resume()
    }

    private func requeue(_ events: [EyeEvent]) {
        queue.sync {
            buffer = events + buffer
            if buffer.count > opts.maxBufferSize {
                buffer.removeFirst(buffer.count - opts.maxBufferSize)
            }
        }
    }
}
