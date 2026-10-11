#!/usr/bin/env python3
"""Import approved map-unit masters as high-density art.

    python3 scripts/sprites/import_sheet.py SHEET UNIT_ID
    python3 scripts/sprites/import_sheet.py --dir FOLDER [FOLDER ...]

--out DIR writes to DIR instead of scripts/sprites/imported (for tests).
--masks DIR reads each unit's A11 look masks, `DIR/ID.skin.png` and
`DIR/ID.hair.png` (E1b); it defaults to art/source/A11/map/units when that
folder exists, as the battle importer reads its masks from --src.

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

A unit whose masks mark any skin or hair also gets
`idle.look.png` and `walk.look.png`, cut and halved exactly like its sheets:
red where the skin is, green where the hair is, transparent elsewhere (a
three-colour PNG). Each sheet's entry names its mask (`look_mask`) so the
client can recolour it in the player's chosen look (sprite-tint.js). A mask
must match its master's size, mark only opaque art and not mark a pixel as
both skin and hair. Every master and mask is checked before anything is
written. A unit with neither mask file (none delivered yet, or no masks
folder) keeps the look masks it already has; one whose masks mark nothing
loses them.

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
DEFAULT_MASKS = os.path.join(os.path.dirname(os.path.dirname(HERE)), "art", "source", "A11", "map", "units")
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


SKIN_RGB, HAIR_RGB = (255, 0, 0), (0, 255, 0)


def look_mask(src, skin_path, hair_path, name):
    """The combined look mask (RGBA: red skin, green hair) for a master, or
    None when neither mask marks anything; raise ValueError on a bad mask."""
    marks = []
    for path in (skin_path, hair_path):
        if not os.path.exists(path):
            raise ValueError(f"{name}: {path} is missing (a unit needs both masks, empty or not)")
        m = np.array(Image.open(path).convert("RGBA"))
        if m.shape != src.shape:
            raise ValueError(f"{path}: {m.shape[1]}x{m.shape[0]}, but its master is {src.shape[1]}x{src.shape[0]}")
        marks.append(m[..., 3] >= 128)
    skin, hair = marks
    if (skin & hair).any():
        raise ValueError(f"{name}: {int((skin & hair).sum())} pixels are marked as both skin and hair")
    if ((skin | hair) & (src[..., 3] == 0)).any():
        raise ValueError(f"{name}: the masks mark pixels the master leaves transparent")
    if not (skin | hair).any():
        return None
    out = np.zeros(src.shape, np.uint8)
    out[skin] = SKIN_RGB + (255,)
    out[hair] = HAIR_RGB + (255,)
    return out


def mask_png(arr, sheet):
    """Save a cut look mask as a palette PNG, cleared wherever its sheet is transparent."""
    arr = arr.copy()
    arr[sheet[..., 3] == 0] = 0
    idx = np.zeros(arr.shape[:2], np.uint8)
    idx[(arr[..., 3] > 0) & (arr[..., 0] > 127)] = 1
    idx[(arr[..., 3] > 0) & (arr[..., 1] > 127)] = 2
    img = Image.fromarray(idx, "P")
    img.putpalette([0, 0, 0, *SKIN_RGB, *HAIR_RGB])
    img.info["transparency"] = 0      # index 0 is clear; saving keeps it
    return img


KEEP = "keep"   # import_master's mask: leave the unit's existing look masks as they are


def load_unit(path, unit, masks=None):
    """(master, mask) for a unit, checked: mask is its combined look mask,
    None when its masks mark nothing, or KEEP when it has no mask files.
    Raise ValueError with every problem found."""
    src = np.array(Image.open(path).convert("RGBA"))
    problems = check_master(src, unit)
    if problems:
        raise ValueError("\n".join(problems))
    skin = os.path.join(masks, f"{unit}.skin.png") if masks else ""
    hair = os.path.join(masks, f"{unit}.hair.png") if masks else ""
    if not masks or (not os.path.exists(skin) and not os.path.exists(hair)):
        return src, KEEP
    return src, look_mask(src, skin, hair, unit)


def import_master(path, unit, out=IMPORTED, masks=None, loaded=None, index=None):
    """Write one unit's idle and walk sheets (and their look masks, from the
    folder masks); return their manifest entries. loaded is load_unit's
    result when already checked; index the existing entries, for masks kept."""
    try:
        src, mask = loaded or load_unit(path, unit, masks)
    except ValueError as e:
        raise SystemExit(str(e))
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
        mrel = f"{rel_dir}/{name}.look.png"
        if mask is KEEP:
            old = (index or {}).get(mrel)
            if old and os.path.exists(os.path.join(out, mrel)):
                entries[rel]["look_mask"] = mrel
                entries[mrel] = old
        elif mask is not None:
            mask_png(np.array(build(mask, columns)), np.array(img)).save(os.path.join(out, mrel), optimize=True)
            entries[rel]["look_mask"] = mrel
            entries[mrel] = {"kind": "look-mask", "set": "S1", "frame": [frame, frame], "rows": DIRECTIONS,
                             "frames": len(columns), "density": DENSITY, "size": [img.width, img.height],
                             "of": rel, "source": "imported"}
        elif os.path.exists(os.path.join(out, mrel)):
            os.remove(os.path.join(out, mrel))
    return entries


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("sheet", nargs="?")
    ap.add_argument("unit", nargs="?")
    ap.add_argument("--dir", nargs="+", default=[], help="folders of ID.png masters")
    ap.add_argument("--out", default=IMPORTED, help="output folder (default scripts/sprites/imported)")
    ap.add_argument("--masks", default=DEFAULT_MASKS if os.path.isdir(DEFAULT_MASKS) else None,
                    help="the A11 map look masks, ID.skin.png and ID.hair.png (default art/source/A11/map/units)")
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
    # Check every master and mask before writing anything.
    loaded, problems = {}, []
    for path, unit in jobs:
        try:
            loaded[unit] = load_unit(path, unit, args.masks)
        except ValueError as e:
            problems.append(str(e))
    if problems:
        raise SystemExit("\n".join(problems))
    for path, unit in jobs:
        old = dict(index)
        for name in ("idle", "walk"):
            index.pop(f"map/units/{unit}/{name}.look.png", None)
        index.update(import_master(path, unit, args.out, args.masks, loaded[unit], old))
        print("imported", unit, file=sys.stderr)
    with open(index_path, "w") as f:
        json.dump(dict(sorted(index.items())), f, indent=1, sort_keys=True)
        f.write("\n")


if __name__ == "__main__":
    main()
