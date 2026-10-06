"""Native authored sheets override procedural art without changing client contracts.

The checked-in PNGs are the source of truth. Validate before publishing so a
bad export cannot silently alter frame layout, transparency, or the palette.
"""
from pathlib import Path

from PIL import Image

from palette import PAL

ROOT = Path(__file__).with_name("authored")


def replacement(rel, generated, meta, root=ROOT):
    path = root / rel
    if not path.is_file():
        return generated
    with Image.open(path) as source:
        image = source.convert("RGBA")
    if image.size != generated.size:
        raise ValueError(f"{rel}: authored size {image.size}, expected {generated.size}")
    palette = set(PAL.values())
    for r, g, b, a in image.getdata():
        if a not in (0, 255) or (a and (r, g, b) not in palette):
            raise ValueError(f"{rel}: authored art must use hard alpha and master palette")
    if meta.get("kind") == "map-unit":
        fw, fh = meta["frame"]
        for y in range(0, image.height, fh):
            for x in range(0, image.width, fw):
                bounds = image.crop((x, y, x + fw, y + fh)).getbbox()
                if bounds is None or bounds[3] - 1 not in (fh - 3, fh - 2):
                    raise ValueError(f"{rel}: authored frame at {x},{y} has invalid feet baseline")
    return image
