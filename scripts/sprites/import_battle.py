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
import io
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
CLEAN = 0.001                 # the share of drawn grain blocks allowed to be uneven (a stray speck)


def clear_hidden(a):
    """A copy of an RGBA array with the colour of fully transparent pixels zeroed."""
    a = a.copy()
    a[a[..., 3] == 0] = 0
    return a


def uneven(cell, g, oy=0, ox=0):
    """The share of drawn g x g blocks (from offset oy, ox) that aren't one colour."""
    h, w = (cell.shape[0] - oy) // g, (cell.shape[1] - ox) // g
    b = cell[oy:oy + h * g, ox:ox + w * g].astype(np.int16).reshape(h, g, w, g, cell.shape[2])
    drawn = (b[..., -1] > 0).any(axis=(1, 3)) if cell.shape[2] == 4 else np.ones((h, w), bool)
    if not drawn.any():
        return 0.0
    bad = (b != b[:, :1, :, :1]).any(axis=(1, 3, 4))
    return float(bad[drawn].mean())


def shift(cell, oy, ox, g):
    """The cell moved so a grain grid starting at (oy, ox) starts at the corner,
    by the smaller way round (up or down, left or right): under half an art pixel."""
    dy = -oy if oy <= g // 2 else g - oy
    dx = -ox if ox <= g // 2 else g - ox
    out = np.zeros_like(cell)
    h, w = cell.shape[:2]
    ys, yd = (slice(-dy, h), slice(0, h + dy)) if dy <= 0 else (slice(0, h - dy), slice(dy, h))
    xs, xd = (slice(-dx, w), slice(0, w + dx)) if dx <= 0 else (slice(0, w - dx), slice(dx, w))
    out[yd, xd] = cell[ys, xs]
    return out


def grain_offset(cells, g):
    """The one (oy, ox) a sheet's grain grid starts at, or None when the art isn't on a g px grain."""
    if all(uneven(c, g) <= CLEAN for c in cells):
        return 0, 0
    best = min((sum(uneven(c, g, oy, ox) for c in cells), oy, ox) for oy in range(g) for ox in range(g))
    if any(uneven(c, g, best[1], best[2]) > CLEAN for c in cells):
        return None
    return best[1], best[2]


def exact_png(arr, budget):
    """PNG bytes for an RGB(A) array: an exact palette when it has 256 colours
    or fewer (lossless and small), else as encode() would."""
    flat = arr.reshape(-1, arr.shape[2])
    colours, index = np.unique(flat, axis=0, return_inverse=True)
    if len(colours) <= 256:
        img = Image.fromarray(index.reshape(arr.shape[:2]).astype(np.uint8), "P")
        img.putpalette(colours[:, :3].astype(np.uint8).flatten().tolist())
        if arr.shape[2] == 4:
            img.info["transparency"] = bytes(colours[:, 3].astype(np.uint8).tolist())
        buf = io.BytesIO()
        img.save(buf, "PNG", optimize=True, transparency=img.info.get("transparency"))
        return buf.getvalue()
    return encode(Image.fromarray(arr, "RGBA" if arr.shape[2] == 4 else "RGB"), budget)


def edges_touched(sheet, frame):
    """The sides of any frame its art touches (a clipped figure, or a whip or strings reaching out)."""
    sides = set()
    for i in range(sheet.shape[1] // frame):
        a = sheet[:, i * frame:(i + 1) * frame, 3] > 0
        for name, edge in (("top", a[0]), ("left", a[:, 0]), ("right", a[:, -1])):
            if edge.any():
                sides.add(name)
    return sorted(sides)


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
    src = clear_hidden(src)
    raw = []
    for i in range(FRAMES):
        x = i * (ch + gap)
        if gap and src[:, x + ch:x + ch + gap, 3].any() and i < FRAMES - 1:
            raise ValueError(f"{path}: opaque pixels in the gutter after frame {i + 1}")
        raw.append(src[:, x:x + ch])
    off = grain_offset(raw, g)
    if off is None:
        raise ValueError(f"{path}: the art is not drawn on a {g} px grain")
    if off != (0, 0):
        print(f"{path}: grain grid starts at {off}; shifted onto the cell", file=sys.stderr)
        raw = [shift(c, off[0], off[1], g) for c in raw]
    frame = ch // g
    sheet = np.concatenate([shrink(c, g) for c in raw], axis=1)
    touched = edges_touched(sheet, frame)
    if touched:
        print(f"{path}: art touches the frame's {', '.join(touched)} edge (check it isn't clipped)", file=sys.stderr)
    data = exact_png(sheet, UNIT_FRAME_BUDGET * FRAMES)
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
    if uneven(src[..., :3], g) > CLEAN:
        raise ValueError(f"{path}: the art is not drawn on a {g} px grain")
    data = exact_png(shrink(src, g)[..., :3].copy(), BACKDROP_BUDGET)
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
