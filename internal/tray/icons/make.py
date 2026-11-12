# /// script
# requires-python = ">=3.10"
# dependencies = ["resvg-py"]
# ///
"""Draws chair.ico from chair.svg: the icon in the tray, and the one the exe
carries. Run from the repository root, then pack the icon for the exe:

    uv run internal/tray/icons/make.py
    go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico internal/tray/icons/chair.ico -o rsrc_windows_amd64.syso

chair.svg is the chair (U+1FA91) from Google's Noto Emoji, unchanged, under
the Apache License 2.0: see LICENSE-noto-emoji.
"""

import pathlib
import struct

import resvg_py

here = pathlib.Path(__file__).parent
chair = (here / "chair.svg").read_text()

# Every size is drawn from the vector, not scaled down from the biggest: at
# sixteen pixels that is the difference between a chair and a smudge. The
# images go in as PNG, which Windows has read in an .ico since Vista.
sizes = (16, 20, 24, 32, 40, 48, 64, 256)
images = [bytes(resvg_py.svg_to_bytes(svg_string=chair, width=n, height=n)) for n in sizes]

out = struct.pack("<HHH", 0, 1, len(sizes))
offset = 6 + 16 * len(sizes)
for n, data in zip(sizes, images):
    out += struct.pack("<BBBBHHII", n % 256, n % 256, 0, 0, 1, 32, len(data), offset)
    offset += len(data)
(here / "chair.ico").write_bytes(out + b"".join(images))
