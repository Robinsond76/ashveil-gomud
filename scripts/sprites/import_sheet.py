#!/usr/bin/env python3
"""Import approved map-unit masters as high-density art.

    python3 scripts/sprites/import_sheet.py SHEET UNIT_ID
    python3 scripts/sprites/import_sheet.py --dir FOLDER [FOLDER ...]

--out DIR writes to DIR instead of scripts/sprites/imported (for tests).

A master is an approved art-program sheet (docs/art/00-standards.md,
section 4): 3 rows (down, up, side facing right) of 8 cells, each 256 px
square with 32 px gutters, feet on y 239 and every figure drawn at one
fixed scale.  Columns 1-2 are the idle, 3-8 the walk.

Each cell is cut exactly from that grid (no per-figure fitting, so a
raised weapon never shrinks a body) and halved to a 128 px frame:
density 4, four times the spec's 32 px, feet baseline 120.  The soles'
lowest row is 119, so they rest exactly on the map's feet line; 1x art's
outline row hangs on that line (row 30 of 32), so a hi-res figure stands
one logical pixel higher than a 1x one.
The halving averages each 2x2 block with premultiplied alpha and keeps
alpha binary, which is exact for art drawn on a 4 px grain.

`idle.png` and `walk.png` go to `scripts/sprites/imported/map/units/ID/`
and their manifest entries to `scripts/sprites/imported/imported.json`,
which `generate.py` copies over its own drawn output.  With --dir every
`ID.png` in the folders is imported.
"""
import argparse
import glob
import json
import os
import sys

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
IMPORTED = os.path.join(HERE, "imported")
DIRECTIONS = ["down", "up", "side"]

CELL = 256            # master cell
GUTTER = 32           # transparent gap between cells
COLS = 8              # 2 idle + 6 walk
FEET = 239            # lowest figure row in a master cell
DENSITY = 4           # runtime frames are 32 * DENSITY px
SHRINK = CELL // (32 * DENSITY)
IDLE = (0, 1)
WALK = (2, 3, 4, 5, 6, 7)


def check_master(src, name):
    """Problems with a master's layout, as messages (empty when it fits)."""
    want = (len(DIRECTIONS) * CELL + (len(DIRECTIONS) - 1) * GUTTER,
            COLS * CELL + (COLS - 1) * GUTTER)
    if src.shape[:2] != want:
        return [f"{name}: sheet is {src.shape[1]}x{src.shape[0]}, expected {want[1]}x{want[0]}"]
    problems = []
    alpha = src[..., 3] > 0
    for r in range(len(DIRECTIONS)):
        for c in range(COLS):
            cell = alpha[r * (CELL + GUTTER):r * (CELL + GUTTER) + CELL,
                         c * (CELL + GUTTER):c * (CELL + GUTTER) + CELL]
            rows = np.nonzero(cell.any(axis=1))[0]
            if len(rows) == 0:
                problems.append(f"{name}: {DIRECTIONS[r]} cell {c + 1} is empty")
            elif rows.max() != FEET:
                problems.append(f"{name}: {DIRECTIONS[r]} cell {c + 1} has its feet on y {rows.max()}, not {FEET}")
    cells = np.zeros_like(alpha)
    for r in range(len(DIRECTIONS)):
        for c in range(COLS):
            cells[r * (CELL + GUTTER):r * (CELL + GUTTER) + CELL,
                  c * (CELL + GUTTER):c * (CELL + GUTTER) + CELL] = True
    if (alpha & ~cells).any():
        problems.append(f"{name}: opaque pixels in the gutters")
    return problems


def shrink(cell):
    """Halve an RGBA cell: premultiplied 2x2 average, binary alpha."""
    h, w = cell.shape[0] // SHRINK, cell.shape[1] // SHRINK
    blocks = cell.reshape(h, SHRINK, w, SHRINK, 4).astype(np.float64)
    a = blocks[..., 3] / 255.0
    cover = a.sum(axis=(1, 3))
    rgb = (blocks[..., :3] * a[..., None]).sum(axis=(1, 3))
    out = np.zeros((h, w, 4), np.uint8)
    keep = cover >= (SHRINK * SHRINK) / 2
    out[..., :3] = np.where(keep[..., None], np.round(rgb / np.maximum(cover, 1e-9)[..., None]), 0)
    out[..., 3] = np.where(keep, 255, 0)
    return out


def build(src, columns):
    frame = CELL // SHRINK
    out = np.zeros((frame * len(DIRECTIONS), frame * len(columns), 4), np.uint8)
    for r in range(len(DIRECTIONS)):
        for i, c in enumerate(columns):
            y, x = r * (CELL + GUTTER), c * (CELL + GUTTER)
            out[r * frame:(r + 1) * frame, i * frame:(i + 1) * frame] = shrink(src[y:y + CELL, x:x + CELL])
    return Image.fromarray(out, "RGBA")


def import_master(path, unit, out=IMPORTED):
    """Write one unit's idle and walk sheets; return their manifest entries."""
    src = np.array(Image.open(path).convert("RGBA"))
    problems = check_master(src, unit)
    if problems:
        raise SystemExit("\n".join(problems))
    frame = CELL // SHRINK
    rel_dir = f"map/units/{unit}"
    os.makedirs(os.path.join(out, rel_dir), exist_ok=True)
    meta = {"kind": "map-unit", "set": "S1", "frame": [frame, frame], "rows": DIRECTIONS,
            "anchor": "bottom-center", "feet_baseline": (FEET + 1) // SHRINK, "density": DENSITY,
            "source": "imported"}
    entries = {}
    for name, columns, ms in (("idle", IDLE, 500), ("walk", WALK, 120)):
        img = build(src, columns)
        rel = f"{rel_dir}/{name}.png"
        img.save(os.path.join(out, rel), optimize=True)
        entries[rel] = dict(meta, frames=len(columns), frame_ms=ms, size=[img.width, img.height])
    return entries


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("sheet", nargs="?")
    ap.add_argument("unit", nargs="?")
    ap.add_argument("--dir", nargs="+", default=[], help="folders of ID.png masters")
    ap.add_argument("--out", default=IMPORTED, help="output folder (default scripts/sprites/imported)")
    args = ap.parse_args(argv)

    jobs = []
    if args.sheet:
        if not args.unit:
            ap.error("give a UNIT_ID with a SHEET")
        jobs.append((args.sheet, args.unit))
    for folder in args.dir:
        for path in sorted(glob.glob(os.path.join(folder, "*.png"))):
            jobs.append((path, os.path.splitext(os.path.basename(path))[0]))
    if not jobs:
        ap.error("nothing to import")
    units = [u for _, u in jobs]
    if len(set(units)) != len(units):
        ap.error("the same unit appears twice")

    index_path = os.path.join(args.out, "imported.json")
    index = {}
    if os.path.exists(index_path):
        with open(index_path) as f:
            index = json.load(f)
    for path, unit in jobs:
        index.update(import_master(path, unit, args.out))
        print("imported", unit, file=sys.stderr)
    with open(index_path, "w") as f:
        json.dump(dict(sorted(index.items())), f, indent=1, sort_keys=True)
        f.write("\n")


if __name__ == "__main__":
    main()
