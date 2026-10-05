"""Generate art set S0 (style foundation) from the sprite specification.

Usage: python3 scripts/sprites/s0.py   (requires Pillow and NumPy)
Writes _datafiles/html/public/static/sprites/style/ and sprites/contact/s0.png.
"""
import os
import numpy as np
from PIL import Image
import pixkit as pk
import map_units, battle_units, icons, scenes

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OUT = os.path.join(ROOT, "_datafiles", "html", "public", "static", "sprites")


def write(arr, rel):
    path = os.path.join(OUT, rel)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    bad = pk.check_palette(arr)
    assert not bad, f"{rel}: off-palette colours {bad}"
    pk.save(arr, path)
    return arr


def palette():
    arr = np.zeros((8, 8, 4), np.uint8)
    for i, (k, _, _) in enumerate(pk.PALETTE):
        arr[i // 8, i % 8, :3] = pk.RGB[k]
        arr[i // 8, i % 8, 3] = 255
    write(arr, "style/palette.png")
    lines = ["GIMP Palette", "Name: Ashveil Master", "Columns: 8", "#"]
    for k, _, name in pk.PALETTE:
        r, g, b = pk.RGB[k]
        lines.append(f"{r:3d} {g:3d} {b:3d}\t{k} {name}")
    with open(os.path.join(OUT, "style", "palette.gpl"), "w") as f:
        f.write("\n".join(lines) + "\n")
    return arr


def contact(sheets):
    """Every S0 image at 2x on mid-grey, stacked with 8 px gaps."""
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
    smap = write(scenes.style_map(), "style/style-map.png")
    sbat = write(scenes.style_battle(), "style/style-battle.png")
    pm = write(np.concatenate([map_units.build(n).render() for n in map_units.ORDER], 1),
               "style/proportions-map.png")
    pb = write(np.concatenate([battle_units.CLASSES[n]().render() for n in battle_units.ORDER], 1),
               "style/proportions-battle.png")
    ic = write(np.concatenate([fn().render() for _, fn in icons.ORDER], 1), "style/icon-sample.png")
    contact([pal_big, smap, sbat, pm, pb, ic])


if __name__ == "__main__":
    main()
