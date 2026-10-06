"""Contact sheet: every generated sprite on one image, for review."""
import json
import os

from PIL import Image, ImageDraw

BG = (96, 104, 84)
INK = (235, 230, 215)


def _load(out, rel):
    return Image.open(os.path.join(out, rel)).convert("RGBA")


def contact_sheet(out, dest, zoom=3):
    doc = json.load(open(os.path.join(out, "manifest.json")))
    files = doc["files"]
    rows = []  # (label, [images])

    def add(label, rels):
        rows.append((label, [_load(out, r) for r in rels]))

    add("S0 style frames (1x)", ["style/style-map.png", "style/style-battle.png"])
    add("S0 proportions: map, battle", ["style/proportions-map.png", "style/proportions-battle.png"])
    add("S0 palette, icon sample", ["style/palette.png", "style/icon-sample.png"])
    for cls in ("warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer"):
        add(f"map unit {cls}: idle, walk (rows down, up, side)",
            [f"map/units/{cls}/idle.png", f"map/units/{cls}/walk.png"])
    add("resources", [r for r in files if r.startswith("map/resources/")])
    add("markers", [r for r in files if r.startswith("map/markers/")])
    add("camp", [r for r in files if r.startswith("map/camp/")])
    add("app icons (shown at 1x)", [r for r in files if r.startswith("app/")])

    margin, gap = 12, 10
    W = 1500
    tiles = []
    y = margin
    placed = []
    for label, imgs in rows:
        z = 1 if label.startswith(("app icons", "S0 style", "S0 palette")) else zoom
        if label.startswith("S0 palette"):
            z = 8 if False else zoom
        x = margin
        row_h = 0
        placed.append((label, x, y))
        y += 14
        for im in imgs:
            zz = 2 if im.width * z > W // 2 and z > 1 else z
            if label.startswith("S0 palette") and im.width == 8:
                zz = 8
            big = im.resize((im.width * zz, im.height * zz), Image.NEAREST)
            if x + big.width + margin > W:
                x = margin
                y += row_h + gap
                row_h = 0
            tiles.append((big, x, y))
            x += big.width + gap
            row_h = max(row_h, big.height)
        y += row_h + gap + 6
    sheet = Image.new("RGBA", (W, y + margin), BG + (255,))
    d = ImageDraw.Draw(sheet)
    for label, x, yy in placed:
        d.text((x, yy), label, fill=INK)
    for big, x, yy in tiles:
        sheet.alpha_composite(big, (x, yy))
    os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
    sheet.convert("RGB").save(dest, optimize=True)
