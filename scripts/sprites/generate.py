#!/usr/bin/env python3
"""Generate the Ashveil sprite sets (S0 style foundation, S1 map basics).

    python3 scripts/sprites/generate.py [--out DIR] [--preview FILE]

Writes PNG sheets in the layout of docs/designs/2026-10-05-sprite-specification.md
plus sprites/manifest.json, which later phases read to find frame sizes,
rows and timing.  Output is deterministic: rerunning changes no bytes.
Requires Pillow.  See scripts/sprites/README.md.
"""
import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from PIL import Image  # noqa: E402

import figures  # noqa: E402
import icons  # noqa: E402
import scenes  # noqa: E402
from palette import PAL  # noqa: E402
from pixels import Canvas, sheet  # noqa: E402

DEFAULT_OUT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "..",
    "_datafiles", "html", "public", "static", "sprites")

BASE_CLASSES = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"]
MAP_UNITS = BASE_CLASSES + ["adventurer"]
DIRECTIONS = ["down", "up", "side"]


class Writer:
    def __init__(self, out):
        self.out = os.path.abspath(out)
        self.manifest = {}

    def png(self, rel, img, **meta):
        path = os.path.join(self.out, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        img.save(path, optimize=True)
        meta.update(size=[img.width, img.height])
        self.manifest[rel] = meta

    def canvas(self, rel, cv, **meta):
        self.png(rel, cv.to_image(), **meta)


def write_palette(w):
    names = list(PAL)
    img = Image.new("RGBA", (8, 8), (0, 0, 0, 0))
    for i, n in enumerate(names):
        img.putpixel((i % 8, i // 8), PAL[n] + (255,))
    w.png("style/palette.png", img, kind="palette", set="S0", swatches=len(names))
    lines = ["GIMP Palette", "Name: Ashveil master palette", "Columns: 8", "#"]
    for n in names:
        r, g, b = PAL[n]
        lines.append(f"{r:3d} {g:3d} {b:3d}\t{n}")
    path = os.path.join(w.out, "style/palette.gpl")
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")
    w.manifest["style/palette.gpl"] = {"kind": "palette-gpl", "set": "S0", "swatches": len(names)}


def write_s0(w):
    write_palette(w)
    w.canvas("style/style-map.png", scenes.mock_map_scene(), kind="style-frame", set="S0")
    w.canvas("style/style-battle.png", scenes.mock_battle_scene(), kind="style-frame", set="S0")
    w.canvas("style/proportions-map.png",
             sheet([[figures.render(c, "down", figures.MAP, figures.STAND) for c in BASE_CLASSES]]),
             kind="proportions", set="S0", frame=[32, 32], frames=6, order=BASE_CLASSES)
    w.canvas("style/proportions-battle.png",
             sheet([[figures.battle_idle(c) for c in BASE_CLASSES]]),
             kind="proportions", set="S0", frame=[64, 64], frames=6, order=BASE_CLASSES)
    w.canvas("style/icon-sample.png", sheet([[f() for f in icons.ALL_ICON_SAMPLES]]),
             kind="icons", set="S0", frame=[16, 16], frames=8,
             order=["water", "forage", "bleeding", "stunned", "fighter", "healer", "yield", "dark"])


def write_s1(w):
    # Resource icons
    for name, fn in icons.RESOURCES.items():
        w.canvas(f"map/resources/{name}.png", fn(), kind="icon", set="S1", frame=[16, 16],
                 frames=1, anchor="center")
    # Markers
    M = "map/markers/"
    w.canvas(M + "here-ring.png", sheet([icons.here_ring_frames()]), kind="marker", set="S1",
             frame=[32, 32], frames=4, anchor="center", frame_ms=180)
    w.canvas(M + "company-badge.png", icons.company_badge(), kind="marker", set="S1",
             frame=[12, 12], frames=1, anchor="center")
    w.canvas(M + "ally-banner.png", sheet([icons.ally_banner_frames()]), kind="marker", set="S1",
             frame=[16, 16], frames=2, anchor="center", frame_ms=250)
    w.canvas(M + "walk-target.png", sheet([icons.walk_target_frames()]), kind="marker", set="S1",
             frame=[16, 16], frames=2, anchor="center", frame_ms=250)
    w.canvas(M + "walk-dot.png", icons.walk_dot(), kind="marker", set="S1", frame=[8, 8],
             frames=1, anchor="center")
    w.canvas(M + "exit-up.png", icons.exit_chevron(True), kind="marker", set="S1", frame=[8, 8],
             frames=1, anchor="center")
    w.canvas(M + "exit-down.png", icons.exit_chevron(False), kind="marker", set="S1",
             frame=[8, 8], frames=1, anchor="center")
    # Camp
    C = "map/camp/"
    one = dict(kind="camp", set="S1", frames=1, anchor="center")
    w.canvas(C + "tent.png", icons.tent(), frame=[32, 32], **one)
    w.canvas(C + "tent-ally.png", icons.tent(True), frame=[32, 32], **one)
    w.canvas(C + "camp-rough.png", icons.camp_rough(), frame=[32, 32], **one)
    w.canvas(C + "camp-rough-ally.png", icons.camp_rough(True), frame=[32, 32], **one)
    w.canvas(C + "embers.png", sheet([icons.embers_frames()]), kind="camp", set="S1",
             frame=[16, 16], frames=3, anchor="center", frame_ms=150)
    w.canvas(C + "fire-unlit.png", icons.fire_unlit(), frame=[16, 16], **one)
    w.canvas(C + "fire-lit.png", sheet([icons.fire_lit_frames()]), kind="camp", set="S1",
             frame=[16, 16], frames=4, anchor="center", frame_ms=150)
    w.canvas(C + "smoke.png", sheet([icons.smoke_frames()]), kind="camp", set="S1",
             frame=[16, 16], frames=4, anchor="center", frame_ms=150)
    w.canvas(C + "resting.png", sheet([icons.resting_frames()]), kind="camp", set="S1",
             frame=[16, 16], frames=3, anchor="center", frame_ms=150)
    w.canvas(C + "inn-rest.png", icons.inn_rest(), frame=[16, 16], **one)
    # Map unit sprites: one row per direction (down, up, side).
    for cls in MAP_UNITS:
        base = dict(kind="map-unit", set="S1", frame=[32, 32], rows=DIRECTIONS,
                    anchor="bottom-center", feet_baseline=30)
        w.canvas(f"map/units/{cls}/idle.png", sheet(figures.map_idle(cls)), frames=2,
                 frame_ms=500, **base)
        w.canvas(f"map/units/{cls}/walk.png", sheet(figures.map_walk(cls)), frames=4,
                 frame_ms=120, **base)
    # App icons (phase 40i consumes them).
    A = "app/"
    scale = lambda cv, n: cv.to_image().resize((cv.w * n, cv.h * n), Image.NEAREST)
    w.png(A + "icon-512.png", scale(scenes.emblem(64), 8), kind="app-icon", set="S1")
    w.png(A + "icon-192.png", scale(scenes.emblem(48, detail=False), 4), kind="app-icon", set="S1")
    w.png(A + "icon-maskable-512.png", scale(scenes.emblem(64, e=0.8), 8), kind="app-icon",
          set="S1", maskable=True, safe_zone=0.8)
    w.png(A + "favicon-32.png", scenes.emblem(32, detail=False).to_image(), kind="app-icon",
          set="S1")


def write_manifest(w):
    doc = {
        "version": 1,
        "palette": "style/palette.gpl",
        "notes": "Frames run left to right, rows top to bottom in `rows` order. "
                 "Units anchor bottom-center; icons and markers center. "
                 "Map units face down, up and side (side faces right; mirror for left).",
        "files": dict(sorted(w.manifest.items())),
    }
    with open(os.path.join(w.out, "manifest.json"), "w") as f:
        json.dump(doc, f, indent=1)
        f.write("\n")


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--out", default=DEFAULT_OUT, help="output directory")
    ap.add_argument("--preview", help="also write a contact sheet PNG here")
    args = ap.parse_args(argv)
    w = Writer(args.out)
    write_s0(w)
    write_s1(w)
    write_manifest(w)
    if args.preview:
        import preview
        preview.contact_sheet(w.out, args.preview)
    print(f"wrote {len(w.manifest)} files to {w.out}")


if __name__ == "__main__":
    main()
