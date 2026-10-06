#!/usr/bin/env python3
"""Import a commissioned map-unit sheet as high-density art.

    python3 scripts/sprites/import_sheet.py SHEET UNIT_ID [--density 4]

SHEET is a grid of figures on a plain light background: one row per
direction (down, up, side facing right) and the same number of frames in
each row.  The figures are cut out, scaled to one consistent size, given
their feet on the spec's baseline and packed into `idle.png` and
`walk.png` under `scripts/sprites/imported/map/units/UNIT_ID/`.  Their
manifest entries go to `scripts/sprites/imported/imported.json`, which
`generate.py` copies over its own output.

A density-N sheet has frames N times the spec's 32 px.  The client draws
it at the same on-map size as 1x art, so the extra pixels show as detail
when zoomed in or on high-resolution screens.
"""
import argparse
import json
import os
from collections import deque

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
IMPORTED = os.path.join(HERE, "imported")
DIRECTIONS = ["down", "up", "side"]

BASE_FRAME = 32        # the spec's map-unit frame
BASE_FEET = 30         # feet baseline: frameHeight - 2
BASE_FIGURE = 28       # spec: the figure fills about 14x28 px
BG_DISTANCE = 60       # how far from the background a pixel must be to count as figure
MIN_BLOB = 40          # smaller specks are compression noise


def figure_mask(cell, bg):
    """The figure's pixels: far enough from the background, specks removed."""
    if bg is None:
        mask = cell[..., 3] > 128
    else:
        mask = np.abs(cell[..., :3].astype(int) - bg).max(axis=2) > BG_DISTANCE
    seen = np.zeros_like(mask)
    keep = np.zeros_like(mask)
    h, w = mask.shape
    for y0, x0 in zip(*np.nonzero(mask)):
        if seen[y0, x0]:
            continue
        blob, q = [], deque([(y0, x0)])
        seen[y0, x0] = True
        while q:
            y, x = q.popleft()
            blob.append((y, x))
            for ny, nx in ((y + 1, x), (y - 1, x), (y, x + 1), (y, x - 1)):
                if 0 <= ny < h and 0 <= nx < w and mask[ny, nx] and not seen[ny, nx]:
                    seen[ny, nx] = True
                    q.append((ny, nx))
        if len(blob) >= MIN_BLOB:
            ys, xs = zip(*blob)
            keep[list(ys), list(xs)] = True
    return keep


def spans(occupied, count, min_gap):
    """The <count> widest runs of occupied lines, merging gaps under <min_gap>."""
    runs, start, gap = [], None, 0
    for i, on in enumerate(list(occupied) + [False] * min_gap):
        if on:
            if start is None:
                start = i
            gap, end = 0, i
        elif start is not None:
            gap += 1
            if gap >= min_gap:
                runs.append((start, end + 1))
                start = None
    if len(runs) < count:
        raise SystemExit(f"found {len(runs)} figures, expected {count}")
    return sorted(sorted(runs, key=lambda r: r[0] - r[1])[:count])


def cut_figures(path, rows, cols):
    src = np.array(Image.open(path).convert("RGBA"))
    # A sheet with a transparent background is cut by alpha; an opaque one
    # by distance from its border colour.
    bg = None
    if (src[..., 3] < 128).mean() < 0.2:
        bg = np.median(np.concatenate([src[0, :, :3], src[-1, :, :3]]), axis=0)
    full = figure_mask(src, bg)
    figs = []
    for top, bottom in spans(full.any(axis=1), rows, 6):
        band = full[top:bottom]
        row = []
        for left, right in spans(band.any(axis=0), cols, 12):
            mask = band[:, left:right]
            ys, xs = np.nonzero(mask)
            rgba = src[top:bottom, left:right].copy()
            rgba[..., 3] = np.where(mask, 255, 0)
            box = (xs.min(), ys.min(), xs.max() + 1, ys.max() + 1)
            # The horizontal anchor is the mask's centre of mass, so a cape
            # or staff swinging out does not shove the body sideways.
            cx = xs.mean() - box[0]
            row.append((Image.fromarray(rgba).crop(box), cx))
        figs.append(row)
    return figs


def place(fig, cx, scale, frame, feet):
    """Scale one cut-out figure and stand it bottom-centre in a frame."""
    w = max(1, round(fig.width * scale))
    h = max(1, round(fig.height * scale))
    # Premultiplied resampling keeps the edges free of a pale fringe.
    small = fig.convert("RGBa").resize((w, h), Image.LANCZOS).convert("RGBA")
    out = Image.new("RGBA", (frame, frame), (0, 0, 0, 0))
    x = round(frame / 2 - cx * scale)
    out.alpha_composite(small, (max(0, min(frame - w, x)), max(0, feet - h)))
    return out


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("sheet")
    ap.add_argument("unit")
    ap.add_argument("--density", type=int, default=4)
    ap.add_argument("--cols", type=int, default=6, help="frames per row in the source")
    ap.add_argument("--idle", default="0,0", help="source columns for the 2 idle frames")
    args = ap.parse_args(argv)

    d = args.density
    frame, feet = BASE_FRAME * d, BASE_FEET * d
    figs = cut_figures(args.sheet, len(DIRECTIONS), args.cols)
    tallest = max(f.height for row in figs for f, _ in row)
    scale = BASE_FIGURE * d / tallest

    def sheet(columns):
        img = Image.new("RGBA", (frame * len(columns), frame * len(DIRECTIONS)), (0, 0, 0, 0))
        for r, row in enumerate(figs):
            for i, c in enumerate(columns):
                f, cx = row[c]
                img.alpha_composite(place(f, cx, scale, frame, feet), (i * frame, r * frame))
        return img

    rel_dir = f"map/units/{args.unit}"
    os.makedirs(os.path.join(IMPORTED, rel_dir), exist_ok=True)
    meta = {"kind": "map-unit", "set": "S1", "frame": [frame, frame], "rows": DIRECTIONS,
            "anchor": "bottom-center", "feet_baseline": feet, "density": d,
            "source": "imported"}
    entries = {}
    for name, columns, ms in (("idle", [int(c) for c in args.idle.split(",")], 500),
                              ("walk", list(range(args.cols)), 120)):
        img = sheet(columns)
        rel = f"{rel_dir}/{name}.png"
        img.save(os.path.join(IMPORTED, rel), optimize=True)
        entries[rel] = dict(meta, frames=len(columns), frame_ms=ms, size=[img.width, img.height])

    index_path = os.path.join(IMPORTED, "imported.json")
    index = {}
    if os.path.exists(index_path):
        with open(index_path) as f:
            index = json.load(f)
    index.update(entries)
    with open(index_path, "w") as f:
        json.dump(dict(sorted(index.items())), f, indent=1, sort_keys=True)
        f.write("\n")
    for rel in entries:
        print("wrote", os.path.join(IMPORTED, rel))


if __name__ == "__main__":
    main()
