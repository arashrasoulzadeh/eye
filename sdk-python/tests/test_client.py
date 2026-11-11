import json
from pathlib import Path
from urllib.error import HTTPError

import tempfile
import unittest

from eye import EyeClient


class Response:
    status = 202

    def __init__(self, body):
        self.body = body

    def read(self):
        return self.body


class ClientTests(unittest.TestCase):
  def test_queue_survives_failed_flush(self):
    tmp_path = Path(tempfile.mkdtemp())
    calls = []

    def fake_urlopen(req):
        calls.append(req.full_url)
        if req.full_url.endswith("/sessions"):
            return Response(b'{"ID":"s1"}')
        if len([call for call in calls if call.endswith("/ingest")]) == 1:
            raise HTTPError(req.full_url, 503, "unavailable", {}, None)
        return Response(b"{}")

    import eye.client as module
    original = module.request.urlopen
    module.request.urlopen = fake_urlopen
    self.addCleanup(setattr, module.request, "urlopen", original)
    client = EyeClient("https://eye.test", "key", "tenant", str(tmp_path / "queue.sqlite"))
    assert client.ensure_session() == "s1"
    client.track("purchase", {"amount": 10})
    with self.assertRaises(HTTPError):
        client.flush()
    assert client.diagnostics().queued_events == 1
    assert client.flush() == 1
    assert client.diagnostics().queued_events == 0
    client.close()


  def test_queue_is_bounded_and_reports_drops(self):
    tmp_path = Path(tempfile.mkdtemp())
    import eye.client as module
    original = module.request.urlopen
    module.request.urlopen = lambda req: Response(b'{"ID":"s1"}')
    self.addCleanup(setattr, module.request, "urlopen", original)
    client = EyeClient("https://eye.test", "key", "tenant", str(tmp_path / "queue.sqlite"), max_queue_size=2)
    client.ensure_session()
    client.track("a"); client.track("b"); client.track("c")
    assert client.diagnostics().queued_events == 2
    assert client.diagnostics().dropped_events == 1
    client.close()


if __name__ == "__main__":
    unittest.main()
