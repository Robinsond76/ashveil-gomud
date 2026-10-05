"""Generate art set S0 (style foundation) in the pre-rendered style.

Usage: python3 scripts/sprites/s0.py   (requires Pillow and NumPy)
Writes _datafiles/html/public/static/sprites/style/ and sprites/contact/s0.png.

Figures are 3D models (figures.py) rendered to pixels by render3d.py; the
scenes and icons are in scenes2.py and icons2.py. Every pixel is checked
against the master palette, which is the union of all material and scene
ramps.
"""
import os
import numpy as np
from PIL import Image
import render3d as r3
import figures as fg
import scenes2
import icons2

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OUT = os.path.join(ROOT, "_datafiles", "html", "public", "static", "sprites")


def palette_entries():
    entries = []
    seen = set()
    for name, ramp in list(r3.RAMPS.items()) + list(scenes2.SCENE.items()):
        for i, c in enumerate(ramp):
            if c not in seen:
                seen.add(c)
                entries.append((c, f"{name} {i}"))
    return entries


PALETTE = palette_entries()
ALLOWED = {r3.rgb(c) for c, _ in PALETTE}


def write(arr, rel):
    bad = {tuple(int(v) for v in p[:3]) for p in arr.reshape(-1, 4) if p[3]} - ALLOWED
    assert not bad, f"{rel}: off-palette colours {sorted(bad)[:5]}"
    path = os.path.join(OUT, rel)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    Image.fromarray(arr, "RGBA").save(path, optimize=True)
    return arr


def palette():
    cols = 16
    rows = (len(PALETTE) + cols - 1) // cols
    arr = np.zeros((rows, cols, 4), np.uint8)
    for i, (c, _) in enumerate(PALETTE):
        arr[i // cols, i % cols, :3] = r3.rgb(c)
        arr[i // cols, i % cols, 3] = 255
    write(arr, "style/palette.png")
    lines = ["GIMP Palette", "Name: Ashveil Master", f"Columns: {cols}", "#"]
    for c, name in PALETTE:
        r, g, b = r3.rgb(c)
        lines.append(f"{r:3d} {g:3d} {b:3d}\t{name}")
    with open(os.path.join(OUT, "style", "palette.gpl"), "w") as f:
        f.write("\n".join(lines) + "\n")
    return arr


BATTLE = 80  # Battle M frame (owner revision: 80x80 instead of 64x64)


def battle_idle(name):
    return r3.render(fg.CLASSES[name](), BATTLE, BATTLE, r3.Camera(50, 16, bulk=1.25),
                     (BATTLE // 2, BATTLE - 2), ambient=0.3)


def contact(sheets):
    scaled = [Image.fromarray(a, "RGBA").resize((a.shape[1] * 2, a.shape[0] * 2), Image.NEAREST)
              for a in sheets]
    w = max(i.width for i in scaled) + 16
    h = sum(i.height for i in scaled) + 8 * (len(scaled) + 1)
    out = Image.new("RGBA", (w, h), (128, 128, 128, 255))
    y = 8
    for i in scaled:
        out.alpha_composite(i, (8, y))
        y += i.height + 8
    path = os.path.join(OUT, "contact", "s0.png")
    os.makedirs(os.path.dirname(path), exist_ok=True)
    out.save(path, optimize=True)


def main():
    pal = palette()
    pal_big = np.repeat(np.repeat(pal, 8, 0), 8, 1)
    smap = write(scenes2.style_map(), "style/style-map.png")
    sbat = write(scenes2.style_battle(), "style/style-battle.png")
    pm = write(np.concatenate([scenes2.map_unit(n) for n in fg.ORDER], 1), "style/proportions-map.png")
    pb = write(np.concatenate([battle_idle(n) for n in fg.ORDER], 1), "style/proportions-battle.png")
    ic = write(np.concatenate([fn() for _, fn in icons2.ORDER], 1), "style/icon-sample.png")
    # enemies from the battle frame, for review only (S3 delivers them)
    foes = np.zeros((128, 192, 4), np.uint8)
    scenes2.paste(foes, r3.render(fg.ogre(), 128, 128, r3.Camera(50, 16, bulk=1.1), (64, 126), ambient=0.3), 0, 0)
    scenes2.paste(foes, r3.render(fg.goblin(), 64, 64, r3.Camera(50, 16, bulk=1.25), (32, 62), ambient=0.3),
                  128, 64)
    contact([pal_big, smap, sbat, pm, pb, ic, write(foes, "contact/s0-foes.png")])
    print(f"palette: {len(PALETTE)} colours")


if __name__ == "__main__":
    main()
