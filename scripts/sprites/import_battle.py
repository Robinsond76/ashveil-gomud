#!/usr/bin/env python3
"""Import the approved battle masters (A6-A10) as exact high-density art.

    python3 scripts/sprites/import_battle.py [--src art/source] [--out DIR] [--only PATH ...]

Battle masters are drawn on a coarse pixel grain: each art pixel is a
square block of N master pixels (5 for a person in a 640 px cell, 6 for a
small creature, 8 for a beast or the boss, 12 for a large creature, 4 for a
backdrop).  The importer cuts each frame and shrinks it by its grain, so
every art pixel becomes exactly one runtime pixel: nothing is lost or
blurred.  The resulting density (runtime px per 1x px) is 2 for most sizes
(8/3 for small creatures).  See
docs/designs/2026-10-10-e3-hires-battle-design.md.

A unit sheet is 1 row of 4 frames, each a square cell, with equal gutters
(cell/16).  Its manifest entry keeps the roster's metadata (size class,
family, names, floating) and records the art's own feet row.  A frame
whose grain grid is offset from the cell's corner is shifted onto it (less
than one art pixel), and a sheet that still isn't on a clean grain is
refused.

The files go to `scripts/sprites/imported/` (or --out) with their entries in
its `imported.json`; `generate.py` copies them over its drawn output.
"""
import argparse
import glob
import json
import os
import sys

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import roster  # noqa: E402
from import_art import encode, shrink, IMPORTED, DEFAULT_SRC  # noqa: E402

PHASES = ["A6", "A7", "A8a", "A8b", "A9", "A10"]
FRAMES = 4
# master cell -> (grain, runtime frame) by size class.
CELLS = {
    "S": {768: 6},
    "M": {640: 5, 1024: 8},
    "L": {1728: 12},
    "XL": {1536: 8},
}
BACKDROP = (2560, 1440, 4)    # master size and grain: 640 x 360 runtime, density 2
UNIT_FRAME_BUDGET = 60 * 1024
BACKDROP_BUDGET = 400 * 1024
CLEAN = 0.002                 # the share of grain blocks allowed to be uneven (antialiased specks)


def uneven(cell, g, oy=0, ox=0):
    """The share of g x g blocks (from offset oy, ox) that aren't one colour."""
    h, w = (cell.shape[0] - oy) // g, (cell.shape[1] - ox) // g
    b = cell[oy:oy + h * g, ox:ox + w * g].astype(np.int16).reshape(h, g, w, g, 4)
    return float((b != b[:, :1, :, :1]).any(axis=(1, 3, 4)).mean())


def on_grain(cell, g):
    """The cell shifted so its grain grid starts at the corner, or None."""
    if uneven(cell, g) <= CLEAN:
        return cell
    best = min(((uneven(cell, g, oy, ox), oy, ox) for oy in range(g) for ox in range(g)))
    if best[0] > CLEAN:
        return None
    _, oy, ox = best
    out = np.zeros_like(cell)
    out[:cell.shape[0] - oy, :cell.shape[1] - ox] = cell[oy:, ox:]
    return out


def unit_job(uid, path):
    """(png bytes, manifest entry) for one unit's idle master, or raise ValueError."""
    units = {u.id: u for u in roster.UNITS}
    if uid not in units:
        raise ValueError(f"{path}: {uid} is not a battle unit in roster.py")
    u = units[uid]
    src = np.array(Image.open(path).convert("RGBA"))
    ch, width = src.shape[0], src.shape[1]
    grains = CELLS[u.size]
    if ch not in grains:
        raise ValueError(f"{path}: a {u.size} unit's cell is {sorted(grains)} px, not {ch}")
    g = grains[ch]
    gap = (width - FRAMES * ch) / (FRAMES - 1)
    if gap != int(gap) or gap < 0:
        raise ValueError(f"{path}: {width} px is not {FRAMES} cells of {ch} with equal gutters")
    gap = int(gap)
    cells = []
    for i in range(FRAMES):
        x = i * (ch + gap)
        if gap and src[:, x + ch:x + ch + gap, 3].any() and i < FRAMES - 1:
            raise ValueError(f"{path}: opaque pixels in the gutter after frame {i + 1}")
        cell = on_grain(src[:, x:x + ch], g)
        if cell is None:
            raise ValueError(f"{path}: frame {i + 1} is not drawn on a {g} px grain")
        cells.append(shrink(cell, g))
    frame = ch // g
    sheet = np.concatenate(cells, axis=1)
    data = encode(Image.fromarray(sheet, "RGBA"), UNIT_FRAME_BUDGET * FRAMES)
    if len(data) > UNIT_FRAME_BUDGET * FRAMES:
        raise ValueError(f"{path}: {len(data)} bytes, over {UNIT_FRAME_BUDGET * FRAMES}")
    meta = dict(kind="battle-unit", set="S5" if u.family in ("promoted class", "summon") else "S3",
                frame=[frame, frame], frames=FRAMES, frame_ms=180, anchor="bottom-center",
                facing="right", size_class=u.size, family=u.family, size=[sheet.shape[1], sheet.shape[0]],
                density=frame // u.frame if frame % u.frame == 0 else frame / u.frame, source="imported")
    if u.names:
        meta["names"] = u.names
    if u.variant_of:
        meta["variant_of"] = u.variant_of
    if u.note:
        meta["note"] = u.note
    if u.floating:
        meta.update(floating=True, anchor="center")
    else:
        rows = np.nonzero(sheet[..., 3].any(axis=1))[0]
        meta["feet_baseline"] = int(rows.max())
    return data, meta


def backdrop_job(path):
    src = np.array(Image.open(path).convert("RGBA"))
    w, h, g = BACKDROP
    if (src.shape[1], src.shape[0]) != (w, h):
        raise ValueError(f"{path}: {src.shape[1]}x{src.shape[0]}, expected {w}x{h}")
    if src[..., 3].min() < 255:
        raise ValueError(f"{path}: a backdrop must be opaque")
    img = Image.fromarray(shrink(src, g), "RGBA").convert("RGB")
    data = encode(img, BACKDROP_BUDGET)
    if len(data) > BACKDROP_BUDGET:
        raise ValueError(f"{path}: {len(data)} bytes, over {BACKDROP_BUDGET}")
    return data, None


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--src", default=DEFAULT_SRC, help="the masters (default art/source)")
    ap.add_argument("--out", default=IMPORTED, help="output folder (default scripts/sprites/imported)")
    ap.add_argument("--only", nargs="+", default=[], help="import only these runtime paths (for tests)")
    args = ap.parse_args(argv)

    # The generator's own backdrop entries carry biomes and the ground band;
    # an imported backdrop keeps them and adds its size and density.
    import backgrounds
    jobs = []
    for phase in PHASES:
        for path in sorted(glob.glob(os.path.join(args.src, phase, "battle", "units", "*", "idle.png"))):
            uid = os.path.basename(os.path.dirname(path))
            jobs.append((f"battle/units/{uid}/idle.png", "unit", uid, path))
        for path in sorted(glob.glob(os.path.join(args.src, phase, "battle", "backgrounds", "*.png"))):
            bid = os.path.splitext(os.path.basename(path))[0]
            jobs.append((f"battle/backgrounds/{bid}.png", "backdrop", bid, path))
    if args.only:
        jobs = [j for j in jobs if j[0] in args.only]
    if not jobs:
        raise SystemExit("nothing to import: no battle masters found" + (" for --only" if args.only else ""))
    rels = [j[0] for j in jobs]
    dupes = sorted({r for r in rels if rels.count(r) > 1})
    if dupes:
        raise SystemExit("the same runtime file has masters in two phases: " + ", ".join(dupes))

    built, problems = {}, []
    for rel, kind, key, path in jobs:
        try:
            if kind == "unit":
                built[rel] = unit_job(key, path)
            else:
                if key not in backgrounds.BACKGROUNDS:
                    raise ValueError(f"{path}: {key} is not a backdrop in backgrounds.py")
                data, _ = backdrop_job(path)
                _, biomes = backgrounds.BACKGROUNDS[key]
                built[rel] = (data, dict(kind="battle-background", set="S3", frame=[640, 360], frames=1,
                                         biomes=biomes, ground_band=[16, 100, 304, 176], size=[640, 360],
                                         density=2, source="imported"))
        except (ValueError, OSError) as e:
            problems.append(str(e))
    if problems:
        raise SystemExit("\n".join(problems))

    index_path = os.path.join(args.out, "imported.json")
    index = {}
    if os.path.exists(index_path):
        with open(index_path) as f:
            index = json.load(f)
    for rel, (data, entry) in built.items():
        dst = os.path.join(args.out, rel)
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        with open(dst, "wb") as f:
            f.write(data)
        index[rel] = entry
    os.makedirs(args.out, exist_ok=True)
    with open(index_path, "w") as f:
        json.dump(dict(sorted(index.items())), f, indent=1, sort_keys=True)
        f.write("\n")
    print(f"imported {len(built)} files", file=sys.stderr)


if __name__ == "__main__":
    main()
