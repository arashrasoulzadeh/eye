"""Fetch pinned upstream Vazirmatn assets for self-hosted documentation."""
from pathlib import Path
from urllib.request import urlopen

target = Path(__file__).resolve().parent / "fonts"
target.mkdir(exist_ok=True)
base = "https://raw.githubusercontent.com/rastikerdar/vazirmatn/v33.003/"
for remote, local in [
    ("fonts/webfonts/Vazirmatn%5Bwght%5D.woff2", "Vazirmatn[wght].woff2"),
    ("OFL.txt", "OFL.txt"),
]:
    with urlopen(base + remote, timeout=30) as response:
        data = response.read()
    if local.endswith(".woff2") and not data.startswith(b"wOF2"):
        raise ValueError("Invalid WOFF2 asset")
    (target / local).write_bytes(data)
