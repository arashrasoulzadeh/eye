import XCTest
@testable import EyeSDK

final class MockStore: KeyValueStore {
    var values: [String: String] = [:]
    func string(forKey key: String) -> String? { values[key] }
    func set(_ value: String, forKey key: String) { values[key] = value }
}

/// Routes URLSession requests to an in-process handler instead of the network,
/// so tests don't need a real server.
final class MockURLProtocol: URLProtocol {
    static var handler: ((URLRequest) -> (Int, Data))?

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        guard let handler = MockURLProtocol.handler else {
            client?.urlProtocol(self, didFailWithError: NSError(domain: "mock", code: -1))
            return
        }
        let (status, data) = handler(request)
        let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: nil, headerFields: nil)!
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {}
}

func mockSession() -> URLSession {
    let config = URLSessionConfiguration.ephemeral
    config.protocolClasses = [MockURLProtocol.self]
    return URLSession(configuration: config)
}

final class EyeClientTests: XCTestCase {
    func testRingBufferEvictsOldestAtCapacity() {
        MockURLProtocol.handler = { _ in (200, Data("{}".utf8)) }
        var opts = EyeClient.Options(endpoint: "https://example.invalid", apiKey: "k", tenantId: "t1")
        opts.maxBufferSize = 3
        opts.batchSize = 0
        opts.flushIntervalSeconds = 0
        let client = EyeClient(options: opts, store: MockStore(), urlSession: mockSession())
        client.setSessionId("sess-1")

        for i in 0..<5 {
            client.track(name: "ev", payload: ["i": AnyCodable(i)])
        }

        XCTAssertEqual(client.bufferCount, 3)
    }

    func testFlushTriggeredBySizeThreshold() {
        let expectation = expectation(description: "flush called")
        var receivedCount = 0
        MockURLProtocol.handler = { req in
            if let body = req.httpBody_mockable, let json = try? JSONSerialization.jsonObject(with: body) as? [String: Any],
               let events = json["events"] as? [[String: Any]] {
                receivedCount = events.count
                expectation.fulfill()
            }
            return (200, Data("{\"accepted\":2,\"rejected\":0}".utf8))
        }
        var opts = EyeClient.Options(endpoint: "https://example.invalid", apiKey: "k", tenantId: "t1")
        opts.batchSize = 2
        opts.flushIntervalSeconds = 0
        let client = EyeClient(options: opts, store: MockStore(), urlSession: mockSession())
        client.setSessionId("sess-1")

        client.track(name: "a")
        client.track(name: "b") // hits batchSize=2

        wait(for: [expectation], timeout: 2.0)
        XCTAssertEqual(receivedCount, 2)
    }

    func testFlushTriggeredByTimeThreshold() {
        let expectation = expectation(description: "timer flush called")
        MockURLProtocol.handler = { _ in
            expectation.fulfill()
            return (200, Data("{\"accepted\":1,\"rejected\":0}".utf8))
        }
        var opts = EyeClient.Options(endpoint: "https://example.invalid", apiKey: "k", tenantId: "t1")
        opts.batchSize = 0
        opts.flushIntervalSeconds = 0.05
        let client = EyeClient(options: opts, store: MockStore(), urlSession: mockSession())
        client.setSessionId("sess-1")
        client.track(name: "a")

        wait(for: [expectation], timeout: 2.0)
        client.destroy()
    }

    func testEnsureSessionCreatesAndPersists() {
        MockURLProtocol.handler = { req in
            (200, Data("{\"ID\":\"sess-abc\"}".utf8))
        }
        let store = MockStore()
        var opts = EyeClient.Options(endpoint: "https://example.invalid", apiKey: "k", tenantId: "t1")
        opts.flushIntervalSeconds = 0
        let client = EyeClient(options: opts, store: store, urlSession: mockSession())

        let exp = expectation(description: "ensureSession")
        client.ensureSession { result in
            if case .success(let id) = result {
                XCTAssertEqual(id, "sess-abc")
            } else {
                XCTFail("expected success")
            }
            exp.fulfill()
        }
        wait(for: [exp], timeout: 2.0)
        XCTAssertEqual(store.values["eye_session_id"], "sess-abc")
    }

    func testFlushClearsBufferOnSuccess() {
        MockURLProtocol.handler = { _ in (200, Data("{\"accepted\":1,\"rejected\":0}".utf8)) }
        var opts = EyeClient.Options(endpoint: "https://example.invalid", apiKey: "k", tenantId: "t1")
        opts.batchSize = 0
        opts.flushIntervalSeconds = 0
        let client = EyeClient(options: opts, store: MockStore(), urlSession: mockSession())
        client.setSessionId("sess-1")
        client.track(name: "a")

        let exp = expectation(description: "flush")
        client.flush { result in
            if case .failure(let e) = result { XCTFail("flush failed: \(e)") }
            exp.fulfill()
        }
        wait(for: [exp], timeout: 2.0)
        XCTAssertEqual(client.bufferCount, 0)
    }
}

private extension URLRequest {
    /// URLSession sometimes moves a POST body into `httpBodyStream` before a
    /// custom URLProtocol sees the request, even though EyeClient set
    /// `httpBody` directly — so fall back to draining the stream.
    var httpBody_mockable: Data? {
        if let body = httpBody { return body }
        guard let stream = httpBodyStream else { return nil }
        stream.open()
        defer { stream.close() }
        var data = Data()
        let bufferSize = 4096
        var buffer = [UInt8](repeating: 0, count: bufferSize)
        while stream.hasBytesAvailable {
            let read = stream.read(&buffer, maxLength: bufferSize)
            if read <= 0 { break }
            data.append(buffer, count: read)
        }
        return data
    }
}
