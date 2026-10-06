#!/usr/bin/env python3
"""Generate the Ashveil sprite sets (S0 style, S1 map basics, S2 terrain, S3 battle, S5 promoted classes).

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
import backgrounds  # noqa: E402
import landmarks  # noqa: E402
import promoted  # noqa: E402
import roster  # noqa: E402
import terrain  # noqa: E402
import uiicons  # noqa: E402
from palette import PAL  # noqa: E402
from pixels import Canvas, sheet  # noqa: E402

DEFAULT_OUT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "..",
    "_datafiles", "html", "public", "static", "sprites")

BASE_CLASSES = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"]
NEUTRAL_LINEAGES = ["dollmaster", "halberdier", "samurai", "shaman"]
MAP_UNITS = BASE_CLASSES + ["adventurer"] + NEUTRAL_LINEAGES
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


def write_s2(w):
    """Terrain tiles (3 variants), animated overlays, fog and state tiles, landmarks."""
    D = "map/terrain/"
    for biome, fn in terrain.BIOMES.items():
        anim = biome in terrain.ANIMATED
        w.canvas(f"{D}{biome}.png", terrain.variants(fn), kind="terrain", set="S2", frame=[32, 32],
                 frames=3, variants=3, animated_overlay=f"{biome}-anim.png" if anim else None)
    for biome, fn in terrain.ANIMATED.items():
        w.canvas(f"{D}{biome}-anim.png", terrain._overlay(fn), kind="terrain-anim", set="S2",
                 frame=[32, 32], frames=4, frame_ms=250, overlay=True, over=f"{biome}.png")
    w.canvas(D + "fog.png", terrain.fog(), kind="terrain-state", set="S2", frame=[32, 32], frames=1,
             overlay=True)
    w.canvas(D + "unknown.png", terrain.unknown(), kind="terrain-state", set="S2", frame=[32, 32], frames=1)
    w.canvas(D + "night-mask.png", terrain.night_mask(), kind="terrain-state", set="S2", frame=[32, 32],
             frames=1, overlay=True, note="dithered vignette; hard alpha, palette colors")
    for name, fn in landmarks.LANDMARKS.items():
        w.canvas(f"map/landmarks/{name}.png", fn(), kind="landmark", set="S2", frame=[32, 32],
                 frames=1, anchor="center")


def write_s3(w):
    """Battle backgrounds, idle battle units, formation markers and the status/role/morale/condition icons."""
    for bid, (fn, biomes) in backgrounds.BACKGROUNDS.items():
        w.canvas(f"battle/backgrounds/{bid}.png", fn(), kind="battle-background", set="S3",
                 frame=[320, 180], frames=1, biomes=biomes, ground_band=[16, 100, 304, 176])
    for u in roster.UNITS:
        meta = dict(kind="battle-unit", set="S5" if u.family in ("promoted class", "summon") else "S3", frame=[u.frame, u.frame], frames=4, frame_ms=180,
                    anchor="bottom-center", feet_baseline=u.frame - 2, facing="right",
                    size_class=u.size, family=u.family)
        if u.names:
            meta["names"] = u.names
        if u.variant_of:
            meta["variant_of"] = u.variant_of
        if u.note:
            meta["note"] = u.note
        if u.floating:
            meta.update(floating=True, anchor="center")
            del meta["feet_baseline"]
        w.canvas(f"battle/units/{u.id}/idle.png", sheet([u.frames()]), **meta)
    B = "battle/ui/"
    one = dict(kind="battle-ui", set="S3", frames=1)
    w.canvas(B + "cell.png", uiicons.cell(), frame=[32, 16], anchor="center", **one)
    w.canvas(B + "cell-acting.png", sheet([uiicons.cell_acting_frames()]), kind="battle-ui", set="S3",
             frame=[32, 16], frames=4, anchor="center", frame_ms=180)
    w.canvas(B + "cell-targeted.png", sheet([uiicons.cell_targeted_frames()]), kind="battle-ui",
             set="S3", frame=[32, 16], frames=2, anchor="center", frame_ms=250)
    w.canvas(B + "acting-arrow.png", sheet([uiicons.acting_arrow_frames()]), kind="battle-ui", set="S3",
             frame=[8, 8], frames=2, anchor="center", frame_ms=250)
    w.canvas(B + "fallen.png", uiicons.fallen(), frame=[16, 16], anchor="center", **one)
    w.canvas(B + "surrendered.png", uiicons.surrendered(), frame=[16, 16], anchor="center", **one)
    w.canvas(B + "hp-frame.png", uiicons.hp_frame(), frame=[32, 6], anchor="top-left", **one)
    for folder, table in (("status", uiicons.STATUS), ("roles", uiicons.ROLES),
                          ("morale", uiicons.MORALE), ("conditions", uiicons.CONDITIONS)):
        for name, fn in table.items():
            w.canvas(f"ui/{folder}/{name}.png", fn(), kind="ui-icon", set="S3", frame=[16, 16],
                     frames=1, anchor="center", group=folder)


def write_s5(w):
    """Map sprites for the promoted classes (the battle idles come from the roster)."""
    for cls in promoted.CLASS_IDS:
        base = dict(kind="map-unit", set="S5", frame=[32, 32], rows=DIRECTIONS,
                    anchor="bottom-center", feet_baseline=30, variant_of=promoted.LINEAGE[cls])
        w.canvas(f"map/units/{cls}/idle.png", sheet(promoted.map_idle(cls)), frames=2,
                 frame_ms=500, **base)
        w.canvas(f"map/units/{cls}/walk.png", sheet(promoted.map_walk(cls)), frames=4,
                 frame_ms=120, **base)


def write_battle_mapping(w):
    """Key table for the client: mob name -> unit, biome -> background."""
    names = {}
    for u in roster.UNITS:
        for n in u.names:
            names[n.lower()] = u.id
    bg = {}
    for bid, (_, biomes) in backgrounds.BACKGROUNDS.items():
        for b in biomes:
            bg[b] = bid
    doc = {
        "version": 1,
        "notes": "Mob names (lowercase) map to battle unit ids; biomes map to background ids; "
                 "zone overrides win over biomes.  Units without art fall back to unknown-humanoid, "
                 "unknown-beast or unknown-large.",
        "units": dict(sorted(names.items())),
        "backgrounds": dict(sorted(bg.items())),
        "zone_overrides": {"Stormwatchers Keep": "ice-keep", "Tutorial": "training-yard"},
        "fallbacks": {"humanoid": "unknown-humanoid", "beast": "unknown-beast", "large": "unknown-large"},
    }
    path = os.path.join(w.out, "battle/mapping.json")
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        json.dump(doc, f, indent=1)
        f.write("\n")
    w.manifest["battle/mapping.json"] = {"kind": "mapping", "set": "S3"}


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
    ap.add_argument("--preview", help="also write the S0/S1 contact sheet PNG here")
    ap.add_argument("--preview-s5", help="also write the S5 (promoted classes, summons) contact sheet PNG here")
    ap.add_argument("--preview-s23", help="also write the S2/S3 contact sheet PNG here")
    args = ap.parse_args(argv)
    w = Writer(args.out)
    write_s0(w)
    write_s1(w)
    write_s2(w)
    write_s3(w)
    write_s5(w)
    write_battle_mapping(w)
    write_manifest(w)
    if args.preview:
        import preview
        preview.contact_sheet(w.out, args.preview)
    if args.preview_s23:
        import preview
        preview.contact_sheet_s23(w.out, args.preview_s23)
    if args.preview_s5:
        import preview
        preview.contact_sheet_s5(w.out, args.preview_s5)
    print(f"wrote {len(w.manifest)} files to {w.out}")


if __name__ == "__main__":
    main()
