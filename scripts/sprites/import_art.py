#!/usr/bin/env python3
"""Import the approved A2-A5 masters (terrain, landmarks, icons) as density-4 art.

    python3 scripts/sprites/import_art.py [--src art/source] [--out DIR]

Every master is drawn four times the runtime size (docs/art/00-standards.md,
section 3), so each is shrunk by 4 with a premultiplied 4x4 box average
into a density-4 sheet whose frames are the old 1x frames times 4.  App
icons are written at their exact sizes instead.  See
docs/designs/2026-10-09-e2-hires-terrain-icons-design.md.

The files go to `scripts/sprites/imported/` (or --out) and their manifest
entries into its `imported.json`, which `generate.py` copies over its own
drawn output.  Every expected master must be present, and a full import
refuses a PNG in a phase folder that no runtime file reads (`_`-prefixed
review sheets are ignored); any problem stops the import before anything is
written.
"""
import argparse
import io
import json
import os
import sys

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
IMPORTED = os.path.join(HERE, "imported")
DEFAULT_SRC = os.path.join(os.path.dirname(os.path.dirname(HERE)), "art", "source")

DENSITY = 4
SHRINK = 4            # master px per runtime px
GUTTER = 32           # optional gap between the frames of an animated row
SIDES = ["none", "n", "e", "s", "w", "ns", "ew", "ne", "es", "sw", "nw",
         "nes", "esw", "nsw", "new", "nesw"]
BIOMES = ["cave", "city", "cliffs", "default", "desert", "dungeon", "farmland", "forest",
          "fort", "house", "land", "mountains", "slums", "snow", "spiderweb", "swamp", "water"]
ANIMATED = ["water", "swamp", "snow", "desert"]
LANDMARKS = ["inn", "bank", "shop", "smithy", "herbalist", "trainer", "temple", "shaman",
             "hermit", "gate", "wall", "bridge", "keep", "throne", "townsquare", "village",
             "caravan", "lake-house", "cave-mouth", "dungeon-stair", "obelisk", "pond", "rocks",
             "desert-ruin", "alts", "landmark", "boss-lair"]
RESOURCES = ["water", "forage", "herbs", "firewood", "shelter", "fishing", "game", "unknown", "depleted"]
UI = {
    "status": ["bleeding", "staggered", "knocked-down", "stunned", "armor-broken", "exposed",
               "hobbled", "burning", "overloaded", "poisoned", "asleep", "paralyzed", "blighted",
               "weakened", "hamstrung", "winded", "tackled", "cold", "regenerating", "lit",
               "hidden", "chanting", "winding-up", "warded", "wounded-light", "wounded-lasting",
               "dread", "tracked"],
    "roles": ["fighter", "healer", "caster", "guardian", "controller"],
    "morale": ["hold", "yield", "flee", "nerve", "shaken"],
    "conditions": ["dark", "ambush", "narrow", "cold", "fatigue", "flanked", "cluster"],
}

# Budgets per frame (standards section 3): terrain-sized frames 60 KB,
# icon-sized frames 12 KB.
BIG, SMALL = 60 * 1024, 12 * 1024


class Job:
    """One runtime file: its masters, how to cut them and its manifest entry."""

    def __init__(self, rel, masters, frame, meta, frames=1, opaque=False, edge_check=False,
                 budget=SMALL, exact=None):
        self.rel = rel                # runtime path under sprites/
        self.masters = masters        # source paths under art/source/, in frame order
        self.frame = frame            # runtime frame [w, h] (1x frame * DENSITY)
        self.frames = frames          # frames per master (an animated row), else 1
        self.meta = meta              # manifest fields besides frame, size, density
        self.opaque = opaque
        self.edge_check = edge_check
        self.budget = budget          # bytes per frame
        self.exact = exact            # app icons: [w, h] written as is, no density


def jobs():
    out = []
    t = "A2/map/terrain/"
    for b in BIOMES:
        out.append(Job(f"map/terrain/{b}.png", [f"{t}{b}-{i}.png" for i in (1, 2, 3)], [128, 128],
                       {"kind": "terrain", "set": "S2", "frames": 3, "variants": 3,
                        "animated_overlay": f"{b}-anim.png" if b in ANIMATED else None},
                       opaque=True, edge_check=True, budget=BIG))
    for b in ANIMATED:
        out.append(Job(f"map/terrain/{b}-anim.png", [f"{t}{b}-anim-{i}.png" for i in (1, 2, 3, 4)],
                       [128, 128], {"kind": "terrain-anim", "set": "S2", "frames": 4, "frame_ms": 250,
                                    "replace": True, "over": f"{b}.png"},
                       opaque=True, budget=BIG))
    for kind in ("road", "shore"):
        for s in SIDES:
            out.append(Job(f"map/terrain/{kind}-{s}.png", [f"{t}{kind}-{s}.png"], [128, 128],
                           {"kind": "terrain-piece", "set": "S2", "frames": 1, "piece": kind, "sides": s},
                           opaque=True, budget=BIG))
    out.append(Job("map/terrain/fog.png", [f"{t}fog.png"], [128, 128],
                   {"kind": "terrain-state", "set": "S2", "frames": 1, "overlay": True}, budget=BIG))
    out.append(Job("map/terrain/unknown.png", [f"{t}unknown.png"], [128, 128],
                   {"kind": "terrain-state", "set": "S2", "frames": 1}, opaque=True, budget=BIG))

    for lm in LANDMARKS:
        out.append(Job(f"map/landmarks/{lm}.png", [f"A3/map/landmarks/{lm}.png"], [128, 128],
                       {"kind": "landmark", "set": "S2", "frames": 1, "anchor": "center"}, budget=BIG))
    for r in RESOURCES:
        out.append(Job(f"map/resources/{r}.png", [f"A4/map/resources/{r}.png"], [64, 64],
                       {"kind": "icon", "set": "S1", "frames": 1, "anchor": "center"}))
    markers = [("here-ring", 128, 4, 180), ("company-badge", 48, 1, 0), ("ally-banner", 64, 2, 250),
               ("walk-target", 64, 2, 250), ("walk-dot", 32, 1, 0), ("exit-up", 32, 1, 0),
               ("exit-down", 32, 1, 0)]
    for name, px, n, ms in markers:
        out.append(Job(f"map/markers/{name}.png", [f"A4/map/markers/{name}.png"], [px, px],
                       _anim({"kind": "marker", "set": "S1", "frames": n, "anchor": "center"}, ms),
                       frames=n, budget=BIG if px >= 128 else SMALL))
    camp = [("tent", 128, 1, 0), ("tent-ally", 128, 1, 0), ("camp-rough", 128, 1, 0),
            ("camp-rough-ally", 128, 1, 0), ("fire-unlit", 64, 1, 0), ("fire-lit", 64, 4, 150),
            ("embers", 64, 3, 150), ("smoke", 64, 4, 150), ("resting", 64, 3, 150), ("inn-rest", 64, 1, 0)]
    for name, px, n, ms in camp:
        out.append(Job(f"map/camp/{name}.png", [f"A4/map/camp/{name}.png"], [px, px],
                       _anim({"kind": "camp", "set": "S1", "frames": n, "anchor": "center"}, ms),
                       frames=n, budget=BIG if px >= 128 else SMALL))
    app = [("icon-512", 512, {}), ("icon-192", 192, {}),
           ("icon-maskable-512", 512, {"maskable": True, "safe_zone": 0.8}), ("favicon-32", 32, {})]
    for name, px, extra in app:
        out.append(Job(f"app/{name}.png", [f"A4/app/{name}.png"], None,
                       dict({"kind": "app-icon", "set": "S1"}, **extra),
                       opaque=name != "favicon-32", budget=BIG * 4, exact=[px, px]))

    for group, ids in UI.items():
        for i in ids:
            out.append(Job(f"ui/{group}/{i}.png", [f"A5/ui/{group}/{i}.png"], [64, 64],
                           {"kind": "ui-icon", "set": "S3", "frames": 1, "anchor": "center", "group": group}))
    battle = [("cell", [128, 64], 1, 0, "center"), ("cell-acting", [128, 64], 4, 180, "center"),
              ("cell-targeted", [128, 64], 2, 250, "center"), ("acting-arrow", [32, 32], 2, 250, "center"),
              ("fallen", [64, 64], 1, 0, "center"), ("surrendered", [64, 64], 1, 0, "center"),
              ("hp-frame", [128, 24], 1, 0, "top-left")]
    for name, fr, n, ms, anchor in battle:
        out.append(Job(f"battle/ui/{name}.png", [f"A5/battle/ui/{name}.png"], fr,
                       _anim({"kind": "battle-ui", "set": "S3", "frames": n, "anchor": anchor}, ms),
                       frames=n, budget=BIG if fr[0] >= 128 else SMALL))
    return out


def _anim(meta, ms):
    if ms:
        meta["frame_ms"] = ms
    return meta


def shrink(cell):
    """Shrink an RGBA cell by SHRINK: premultiplied box average, soft alpha."""
    h, w = cell.shape[0] // SHRINK, cell.shape[1] // SHRINK
    blocks = cell.reshape(h, SHRINK, w, SHRINK, 4).astype(np.float64)
    a = blocks[..., 3] / 255.0
    cover = a.sum(axis=(1, 3))
    rgb = (blocks[..., :3] * a[..., None]).sum(axis=(1, 3))
    out = np.zeros((h, w, 4), np.uint8)
    out[..., :3] = np.round(rgb / np.maximum(cover, 1e-9)[..., None]).clip(0, 255)
    out[..., 3] = np.round(cover / (SHRINK * SHRINK) * 255).clip(0, 255)
    out[cover == 0] = 0
    return out


def cut(src, n, cw, ch):
    """Split a 1-row master into n cells of cw x ch, with or without gutters."""
    h, w = src.shape[:2]
    if h != ch:
        return None
    for gap in (0, GUTTER):
        if w == n * cw + (n - 1) * gap:
            return [src[:, i * (cw + gap):i * (cw + gap) + cw] for i in range(n)]
    return None


def edge_dark(cell, band=4):
    """How much darker a tile's edges are than the whole tile (0 = not).

    A frame or vignette darkens all four edges, so this is the least dark
    of the four bands; a texture whose stripes or rows happen to end dark
    on two edges passes.
    """
    lum = cell[..., :3].astype(np.float64) @ [0.299, 0.587, 0.114]
    whole = max(lum.mean(), 1.0)
    edges = [lum[:band].mean(), lum[-band:].mean(), lum[:, :band].mean(), lum[:, -band:].mean()]
    return max(0.0, (whole - max(edges)) / whole)


def encode(img, budget):
    """PNG bytes for img: optimised, quantised to 256 colours when over budget."""
    buf = io.BytesIO()
    img.save(buf, "PNG", optimize=True)
    if buf.tell() <= budget:
        return buf.getvalue()
    q = img.quantize(256, method=Image.Quantize.FASTOCTREE, dither=Image.Dither.NONE)
    buf = io.BytesIO()
    q.save(buf, "PNG", optimize=True)
    return buf.getvalue()


EDGE_LIMIT = 0.25     # an edge band this much darker than the tile is a frame


def build(job, src_root):
    """(png bytes, manifest entry) for a job, or raise ValueError."""
    if job.exact:
        path = os.path.join(src_root, job.masters[0])
        im = Image.open(path).convert("RGBA")
        if im.width != im.height or im.width < job.exact[0]:
            raise ValueError(f"{job.masters[0]}: {im.size[0]}x{im.size[1]}, expected a square of at least {job.exact[0]} px")
        arr = np.array(im)
        if job.opaque and arr[..., 3].min() < 255:
            raise ValueError(f"{job.masters[0]}: must be opaque")
        out = im.resize(tuple(job.exact), Image.Resampling.LANCZOS)
        data = encode(out, job.budget)
        if len(data) > job.budget:
            raise ValueError(f"{job.rel}: {len(data)} bytes after quantising, over {job.budget}")
        return data, dict(job.meta, size=list(job.exact), source="imported")

    fw, fh = job.frame
    cw, ch = fw * SHRINK, fh * SHRINK
    cells = []
    for m in job.masters:
        path = os.path.join(src_root, m)
        if not os.path.exists(path):
            raise ValueError(f"{m}: missing")
        arr = np.array(Image.open(path).convert("RGBA"))
        parts = cut(arr, job.frames, cw, ch)
        if parts is None:
            raise ValueError(f"{m}: {arr.shape[1]}x{arr.shape[0]}, expected {job.frames} frame(s) of {cw}x{ch}")
        for p in parts:
            if job.opaque and p[..., 3].min() < 255:
                raise ValueError(f"{m}: must be opaque")
            if job.edge_check and edge_dark(p) > EDGE_LIMIT:
                raise ValueError(f"{m}: its edges are {edge_dark(p):.0%} darker than the tile (a frame or vignette)")
            cells.append(shrink(p))
    sheet = np.concatenate(cells, axis=1)
    img = Image.fromarray(sheet, "RGBA")
    data = encode(img, job.budget * len(cells))
    if len(data) > job.budget * len(cells):
        raise ValueError(f"{job.rel}: {len(data)} bytes after quantising, over {job.budget * len(cells)}")
    entry = dict(job.meta, frame=[fw, fh], size=[img.width, img.height], density=DENSITY, source="imported")
    return data, entry


def unlisted(src_root, todo):
    """Masters under the phase folders that no job reads (a misspelt name)."""
    wanted = {m for j in todo for m in j.masters}
    phases = {m.split("/")[0] for m in wanted}
    extra = []
    for phase in sorted(phases):
        for dirpath, _, files in os.walk(os.path.join(src_root, phase)):
            for f in files:
                rel = os.path.relpath(os.path.join(dirpath, f), src_root).replace(os.sep, "/")
                if f.endswith(".png") and not f.startswith("_") and rel not in wanted:
                    extra.append(f"{rel}: not an expected master")
    return sorted(extra)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--src", default=DEFAULT_SRC, help="the masters (default art/source)")
    ap.add_argument("--out", default=IMPORTED, help="output folder (default scripts/sprites/imported)")
    ap.add_argument("--only", nargs="+", default=[], help="import only these runtime paths (for tests)")
    ap.add_argument("--budget", type=int, default=0, help="override every per-frame budget in bytes (for tests)")
    args = ap.parse_args(argv)

    todo = [j for j in jobs() if not args.only or j.rel in args.only]
    if not todo:
        raise SystemExit("nothing to import: --only matched no runtime path")
    if args.budget:
        for j in todo:
            j.budget = args.budget
    built, problems = {}, []
    if not args.only:
        problems.extend(unlisted(args.src, todo))
    for job in todo:
        try:
            built[job.rel] = build(job, args.src)
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
