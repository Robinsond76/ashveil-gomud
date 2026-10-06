"""Render the real packed sprites at 1x/4x and on shipped terrain tiles."""
from pathlib import Path
import json
import math

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[2]
ART = ROOT / "_datafiles/html/public/static/sprites"
DEST = ROOT / "docs/verification"
CLASSES = ("warrior", "ranger", "witch")
BG = (96, 104, 100, 255)


def load(rel):
    with Image.open(ART / rel) as image:
        return image.convert("RGBA")


def unit(cls, animation, direction, frame):
    return load(f"map/units/{cls}/{animation}.png").crop(
        (32 * frame, 32 * direction, 32 * (frame + 1), 32 * (direction + 1)))


def scene(frame, fire_frame=0):
    image = Image.new("RGBA", (320, 128), BG)
    for y in range(4):
        for x in range(10):
            biome = "road" if y == 2 else "land"
            tile = load(f"map/terrain/{biome}.png")
            variant = (x + y) % 3
            image.alpha_composite(tile.crop((variant * 32, 0, variant * 32 + 32, 32)), (x * 32, y * 32))
    image.alpha_composite(load("map/camp/tent.png"), (40, 30))
    fire = load("map/camp/fire-lit.png").crop((fire_frame * 16, 0, fire_frame * 16 + 16, 16))
    image.alpha_composite(fire, (80, 50))
    for i, cls in enumerate(CLASSES):
        image.alpha_composite(unit(cls, "walk", 2, frame), (120 + i * 48, 60))
    return image


def main():
    DEST.mkdir(exist_ok=True)
    board = Image.new("RGBA", (960, 966), BG)
    draw = ImageDraw.Draw(board)
    ink = (237, 229, 208)
    draw.text((20, 15), "ASHVEIL / NATIVE ART PASS 01 - actual packed assets", fill=ink)
    for i, cls in enumerate(CLASSES):
        x = 100 + i * 280
        draw.text((x, 48), cls.upper(), fill=ink)
        for direction, label in enumerate(("DOWN", "UP", "RIGHT")):
            y = 70 + direction * 140
            if i == 0:
                draw.text((20, y + 50), label, fill=ink)
            sprite = unit(cls, "idle", direction, 0)
            board.alpha_composite(sprite.resize((128, 128), Image.Resampling.NEAREST), (x, y))
            board.alpha_composite(sprite, (x + 160, y + 96))
    draw.text((20, 502), "TENT / ALLIED TENT / FOUR FIRE FRAMES - 4x nearest neighbor", fill=ink)
    for i, name in enumerate(("tent", "tent-ally", "fire-lit")):
        im = load(f"map/camp/{name}.png")
        board.alpha_composite(im.resize((im.width * 4, im.height * 4), Image.Resampling.NEAREST), (100 + i * 210, 525))
    draw.text((20, 665), "SHIPPED TERRAIN / 2x nearest neighbor", fill=ink)
    board.alpha_composite(scene(0).resize((640, 256), Image.Resampling.NEAREST), (100, 690))
    board.convert("RGB").save(DEST / "sprite-art-pass-01.png")
    animation = []
    manifest = json.loads((ART / "manifest.json").read_text())["files"]
    walk_ms = manifest["map/units/warrior/walk.png"]["frame_ms"]
    fire_ms = manifest["map/camp/fire-lit.png"]["frame_ms"]
    tick = math.gcd(walk_ms, fire_ms)
    for now in range(0, math.lcm(walk_ms * 4, fire_ms * 4), tick):
        frame, fire_frame = (now // walk_ms) % 4, (now // fire_ms) % 4
        im = Image.new("RGBA", (640, 676), BG)
        for row in range(3):
            for col, cls in enumerate(CLASSES):
                sprite = unit(cls, "walk", row, frame)
                im.alpha_composite(sprite.resize((128, 128), Image.Resampling.NEAREST), (60 + col * 180, row * 140))
        im.alpha_composite(scene(frame, fire_frame).resize((640, 256), Image.Resampling.NEAREST), (0, 420))
        animation.append(im.convert("RGB"))
    animation[0].save(DEST / "sprite-art-pass-01-walk.gif", save_all=True,
                      append_images=animation[1:], duration=tick, loop=0)


if __name__ == "__main__":
    main()
