"""Check bilingual file parity and source route inventory coverage."""
from pathlib import Path
import re

root = Path(__file__).resolve().parents[1]
docs = root / "docs"
errors = []
for page in docs.rglob("*.md"):
    if page.name.endswith(".fa.md"):
        peer = page.with_name(page.name[:-6] + ".md")
    else:
        peer = page.with_name(page.stem + ".fa.md")
    if not peer.is_file():
        errors.append(f"Missing translation: {peer.relative_to(root)}")

app = root / "admin/src/app"
routes = set()
for page in app.rglob("page.tsx"):
    parts = [p for p in page.relative_to(app).parts[:-1] if not p.startswith("(")]
    routes.add("/" + "/".join(parts))
server = (root / "server/cmd/server/main.go").read_text()
api = set(re.findall(r'mux\.Handle(?:Func)?\("([^"]+)"', server))
for name, expected in [("admin/routes", routes), ("api/routes", api)]:
    for suffix in ("", ".fa"):
        path = docs / (name + suffix + ".md")
        actual = set(re.findall(r"^\| `(/[^\x60]*)` \|", path.read_text(), re.M))
        for route in sorted(expected - actual):
            errors.append(f"{path.name}: missing {route}")
        for route in sorted(actual - expected):
            errors.append(f"{path.name}: stale {route}")
# Ensure every bilingual page is reachable from the configured navigation.
nav = (root / "mkdocs.yml").read_text().split("\nnav:\n", 1)[1].split("\nplugins:", 1)[0]
nav_pages = set(re.findall(r":\s+([^\s]+\.md)\s*$", nav, re.M))
english_pages = {p.relative_to(docs).as_posix() for p in docs.rglob("*.md") if not p.name.endswith(".fa.md")}
for page in sorted(english_pages - nav_pages):
    errors.append(f"Page missing from navigation: {page}")
for page in sorted(nav_pages - english_pages):
    errors.append(f"Navigation target missing: {page}")
if errors:
    raise SystemExit("\n".join(errors))
print(f"Translation pairs and inventories OK: {len(routes)} app routes, {len(api)} API routes")
