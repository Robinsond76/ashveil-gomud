"""S0 style frames: a mock map and a mock battle (320x180, opaque)."""
import random
import numpy as np
from pixkit import RGB, paste, flip, Canvas, RAMPS
import map_units, battle_units, icons

BAYER = np.array([[0, 8, 2, 10], [12, 4, 14, 6], [3, 11, 1, 9], [15, 7, 13, 5]]) / 16.0


def fill(img, x0, y0, x1, y1, color):
    img[y0:y1, x0:x1, :3] = RGB[color]
    img[y0:y1, x0:x1, 3] = 255


def dither(img, x, y, a, b, t):
    """Pick colour a or b by ordered dither; t is the share of b (0..1)."""
    if not (0 <= y < img.shape[0] and 0 <= x < img.shape[1]):
        return
    img[y, x, :3] = RGB[b] if t > BAYER[y % 4, x % 4] else RGB[a]
    img[y, x, 3] = 255


def setp(img, x, y, color):
    if 0 <= y < img.shape[0] and 0 <= x < img.shape[1]:
        img[y, x, :3] = RGB[color]
        img[y, x, 3] = 255


# --- Map tiles ---------------------------------------------------------------
def tile_land(img, ox, oy, rnd):
    for y in range(32):
        for x in range(32):
            n = rnd.random()
            dither(img, ox + x, oy + y, "G4", "G3" if n < 0.5 else "G5", 0.25)
    for _ in range(7):
        x, y = rnd.randint(3, 27), rnd.randint(3, 27)
        setp(img, ox + x, oy + y, "G6")
        setp(img, ox + x + 2, oy + y, "G6")
        setp(img, ox + x + 1, oy + y + 1, "G3")
    for _ in range(2):
        x, y = rnd.randint(4, 27), rnd.randint(4, 27)
        setp(img, ox + x, oy + y, rnd.choice(["O5", "S5", "V5"]))


def tree(img, cx, cy, r, rnd):
    for y in range(-r, r + 2):
        for x in range(-r - 1, r + 2):
            d = (x * x + y * y) ** 0.5
            if d <= r + 0.5:
                shade = (x + y) / (2 * r)
                c = "G5" if shade < -0.45 else "G4" if shade < 0.0 else "G3" if shade < 0.5 else "G2"
                if rnd.random() < 0.12:
                    c = {"G5": "G6", "G4": "G5", "G3": "G4", "G2": "G3"}[c]
                setp(img, cx + x, cy + y, c)
            elif d <= r + 1.5 and x + y > 0:
                setp(img, cx + x, cy + y, "G1")


def tile_forest(img, ox, oy, rnd):
    for y in range(32):
        for x in range(32):
            dither(img, ox + x, oy + y, "G1", "G0", 0.4)


def forest_crowns(img, ox, oy, rnd, open_sides):
    spots = [(8, 8), (23, 7), (15, 17), (6, 25), (25, 24), (16, 30)]
    for cx, cy in sorted(spots, key=lambda p: p[1]):
        cx += rnd.randint(-2, 2)
        cy += rnd.randint(-2, 2)
        # trees at an open edge pull back so the wood ends in a ragged line
        if "w" in open_sides and cx < 10: cx += 3
        if "e" in open_sides and cx > 22: cx -= 3
        if "n" in open_sides and cy < 10: cy += 3
        if "s" in open_sides and cy > 22: cy -= 4
        tree(img, ox + cx, oy + cy, rnd.randint(6, 8), rnd)


def tile_road(img, ox, oy, rnd, conn):
    tile_land(img, ox, oy, rnd)
    cx = cy = 16
    def dirt(x, y):
        n = rnd.random()
        setp(img, ox + x, oy + y, "E4" if n < 0.55 else "E3" if n < 0.9 else "E5")
    segs = []
    if "n" in conn: segs.append((range(11, 21), range(0, 21)))
    if "s" in conn: segs.append((range(11, 21), range(11, 32)))
    if "w" in conn: segs.append((range(0, 21), range(11, 21)))
    if "e" in conn: segs.append((range(11, 32), range(11, 21)))
    for xs, ys in segs:
        for y in ys:
            for x in xs:
                dirt(x, y)
    for xs, ys in segs:
        for y in ys:
            for x in xs:
                if (x in (13, 18) and len(xs) == 10) or (y in (13, 18) and len(ys) == 10):
                    if rnd.random() < 0.6:
                        setp(img, ox + x, oy + y, "E2")
    for _ in range(3):
        setp(img, ox + rnd.randint(12, 19), oy + rnd.randint(12, 19), "W5")


def tile_water(img, ox, oy, rnd, banks, t=0):
    for y in range(32):
        for x in range(32):
            dither(img, ox + x, oy + y, "B2", "B1", 0.35)
    for _ in range(6):
        x, y = rnd.randint(4, 24), rnd.randint(4, 27)
        for i in range(rnd.randint(3, 5)):
            setp(img, ox + x + i, oy + y, "B4" if i else "B3")
    for side in banks:
        for i in range(32):
            for d in range(4):
                x, y = {"w": (d, i), "e": (31 - d, i), "n": (i, d), "s": (i, 31 - d)}[side]
                c = "G4" if d == 0 else "G3" if d == 1 else "E3" if d == 2 else "B3"
                if d == 2 and rnd.random() < 0.3:
                    c = "G2"
                setp(img, ox + x, oy + y, c)
        for _ in range(4):
            i = rnd.randint(2, 29)
            for h in range(3):
                x, y = {"w": (2, i - h), "e": (29, i - h), "n": (i, 2 - h), "s": (i, 29 - h)}[side]
                setp(img, ox + x, oy + y, "G5" if h == 2 else "G3")


def tile_bridge(img, ox, oy, rnd):
    tile_water(img, ox, oy, rnd, "")
    for x in range(32):
        for y in range(10, 22):
            c = "E3" if (x % 4) else "E1"
            if y in (10, 21):
                c = "E1"
            elif y == 11:
                c = "E4" if x % 4 else "E1"
            setp(img, ox + x, oy + y, c)
    for x in (2, 13, 24):
        for y in (9, 22):
            setp(img, ox + x, oy + y, "E0")
            setp(img, ox + x + 1, oy + y, "E2")


def cottage(img, x, y, rnd):
    # walls
    for yy in range(y + 6, y + 13):
        for xx in range(x, x + 11):
            c = "S5" if xx < x + 9 else "W5"
            if xx in (x, x + 5, x + 10) or yy == y + 12:
                c = "E2"
            setp(img, xx, yy, c)
    setp(img, x + 2, y + 9, "O5"); setp(img, x + 3, y + 9, "O4")
    setp(img, x + 7, y + 10, "E1"); setp(img, x + 7, y + 11, "E1"); setp(img, x + 8, y + 10, "E1"); setp(img, x + 8, y + 11, "E1")
    # thatch roof, lit from the top-left
    for i in range(7):
        for xx in range(x - 1 + i, x + 12 - i):
            yy = y + 6 - i
            c = "O4" if xx < x + 5 else "O3" if xx < x + 9 else "O2"
            if rnd.random() < 0.15:
                c = "O2"
            setp(img, xx, yy, c)
    for xx in range(x - 1, x + 12):
        setp(img, xx, y + 6, "O1")
    for yy in range(y, y + 4):
        setp(img, x + 8, yy, "W4"); setp(img, x + 9, yy, "W3")
    setp(img, x + 9, y - 2, "W5"); setp(img, x + 10, y - 3, "W4")
    for xx in range(x - 1, x + 13):
        setp(img, xx, y + 13, "G2")


def tile_village(img, ox, oy, rnd, seed, path):
    tile_land(img, ox, oy, rnd)
    if path:
        for x in range(32):
            for y in range(13, 19):
                setp(img, ox + x, oy + y, "E4" if rnd.random() < 0.7 else "E3")
    spots = {0: [(4, 1), (17, 19)], 1: [(14, 0), (3, 19)]}[seed]
    for x, y in spots:
        cottage(img, ox + x, oy + y, rnd)
    # a kitchen garden in rows
    gx, gy = (21, 3) if seed == 0 else (3, 3)
    for j in range(0, 7, 2):
        for i in range(8):
            setp(img, ox + gx + i, oy + gy + j, "E3")
            setp(img, ox + gx + i, oy + gy + j + 1, "G6" if i % 2 else "G5")


def tile_camp(img, ox, oy, rnd):
    tile_land(img, ox, oy, rnd)
    # worn earth around the fire
    for y in range(14, 30):
        for x in range(4, 30):
            if ((x - 17) / 13) ** 2 + ((y - 22) / 8) ** 2 < 1 and rnd.random() < 0.7:
                setp(img, ox + x, oy + y, "E3" if rnd.random() < 0.6 else "G3")
    c = Canvas(32, 32, band=1)
    canvas = c.m(["W3", "W4", "W6", "S5", "S6"])
    canvas2 = c.m(["W3", "W3", "W4", "W5", "W6"])
    pole = c.m("wood")
    c.poly([(3, 18), (11, 5), (12, 5), (12, 18)], canvas)
    c.poly([(12, 5), (13, 5), (21, 18), (12, 18)], canvas2)
    c.poly([(10, 18), (12, 11), (14, 18)], c.lit("W1"))
    c.line([(11, 3), (12, 5)], pole)
    roll = c.m("oxblood")
    c.rect(4, 20, 11, 22, roll)
    c.dot(11, 21, "R3")
    stones = c.m("iron")
    for x, y in ((19, 25), (21, 24), (24, 24), (26, 25), (25, 27), (20, 27), (23, 28)):
        c.px(x, y, stones)
    logs = c.m("wood")
    c.line([(20, 26), (25, 25)], logs, width=1)
    fire = c.m("ember", outline=False)
    for x, y, t in ((22, 25, 2), (23, 25, 2), (21, 24, 1), (22, 23, 3), (23, 22, 2), (24, 24, 1),
                    (22, 24, 3), (23, 24, 3), (23, 23, 4), (22, 22, 2), (24, 23, 2)):
        c.px(x, y, fire, t)
    for x, y in ((24, 20), (25, 18), (24, 16), (25, 14)):
        c.dot(x, y, "W4", outline=False)
    paste(img, c.render(), ox, oy)


MAP = [
    "FFFFLLWLLL",
    "FFFLLLWLVV",
    "FFLRRRBRVV",
    "FLLRLLWLLL",
    "LLCRLWWLFF",
    "LLLRLWLFFF",
]


def style_map():
    img = np.zeros((180, 320, 4), np.uint8)
    for r, row in enumerate(MAP):
        for col, ch in enumerate(row):
            rnd = random.Random(r * 31 + col * 7 + 3)
            ox, oy = col * 32, r * 32
            at = lambda rr, cc: MAP[rr][cc] if 0 <= rr < len(MAP) and 0 <= cc < len(row) else ""
            if ch == "F":
                tile_forest(img, ox, oy, rnd)
            elif ch == "L":
                tile_land(img, ox, oy, rnd)
            elif ch == "R":
                conn = "".join(d for d, (dr, dc) in (("n", (-1, 0)), ("s", (1, 0)), ("w", (0, -1)), ("e", (0, 1)))
                               if at(r + dr, col + dc) in ("R", "B", "C", "V"))
                if r == 5:
                    conn += "s"
                tile_road(img, ox, oy, rnd, conn)
            elif ch == "W":
                banks = "".join(d for d, (dr, dc) in (("n", (-1, 0)), ("s", (1, 0)), ("w", (0, -1)), ("e", (0, 1)))
                                if at(r + dr, col + dc) not in ("W", "B", ""))
                tile_water(img, ox, oy, rnd, banks)
            elif ch == "B":
                tile_bridge(img, ox, oy, rnd)
            elif ch == "V":
                tile_village(img, ox, oy, rnd, (r + col) % 2, at(r, col - 1) == "R" or (at(r, col - 1) == "V" and at(r, col - 2) == "R"))
            elif ch == "C":
                tile_camp(img, ox, oy, rnd)
    for r, row in enumerate(MAP):
        for col, ch in enumerate(row):
            if ch == "F":
                at = lambda rr, cc: MAP[rr][cc] if 0 <= rr < len(MAP) and 0 <= cc < len(row) else "F"
                opens = "".join(d for d, (dr, dc) in (("n", (-1, 0)), ("s", (1, 0)), ("w", (0, -1)), ("e", (0, 1)))
                                if at(r + dr, col + dc) != "F")
                forest_crowns(img, col * 32, r * 32, random.Random(r * 13 + col), opens)
    # the company: the Warrior on the road, the Witch by the camp
    paste(img, map_units.build("warrior").render(), 4 * 32, 2 * 32 - 4)
    paste(img, map_units.build("witch").render(), 3 * 32 - 4, 4 * 32 - 6)
    # resource icons in tile corners
    paste(img, icons.water().render(), 7 * 32 + 1, 3 * 32 + 1)
    paste(img, icons.forage().render(), 1 * 32 + 15, 3 * 32 + 1)
    return img


# --- Battle background -------------------------------------------------------
def forest_background():
    rnd = random.Random(40)
    img = np.zeros((180, 320, 4), np.uint8)
    # canopy and sky gaps
    for y in range(180):
        for x in range(320):
            dither(img, x, y, "G1", "G0", 0.5 - y / 400)
    for _ in range(60):
        cx, cy, r = rnd.randint(0, 320), rnd.randint(-10, 40), rnd.randint(8, 20)
        for y in range(cy - r, cy + r):
            for x in range(cx - r, cx + r):
                if (x - cx) ** 2 + (y - cy) ** 2 < r * r and 0 <= y < 70:
                    t = ((x - cx) + (y - cy)) / (2 * r) + 0.5
                    if 0 <= x < 320:
                        dither(img, x, y, "G3", "G2", t)
    for _ in range(40):
        x, y = rnd.randint(20, 300), rnd.randint(2, 40)
        for dx in range(rnd.randint(1, 3)):
            setp(img, x + dx, y, "B4")
            setp(img, x + dx, y + 1, "B3")
    # far trunks, low contrast
    x = 4
    while x < 320:
        w = rnd.randint(2, 6)
        top = rnd.randint(20, 50)
        for y in range(top, 97):
            for dx in range(w):
                c = "G0" if dx == w - 1 else "W1" if dx == 0 else "C0"
                setp(img, x + dx, y, c)
        x += rnd.randint(12, 30)
    # ground: mossy clearing, lighter toward the front
    for y in range(92, 180):
        for x in range(320):
            t = (y - 92) / 88
            if t < 0.35:
                dither(img, x, y, "G2", "G3", t / 0.35)
            elif t < 0.75:
                dither(img, x, y, "G3", "G4", (t - 0.35) / 0.4)
            else:
                dither(img, x, y, "G4", "G3", (t - 0.75) / 0.6)
    # near trunks standing on the far edge of the clearing
    for x, w in ((40, 9), (98, 7), (150, 10), (214, 8), (268, 11)):
        x += rnd.randint(-3, 3)
        for y in range(10, 101):
            for dx in range(w):
                c = "E2" if dx < 2 else "E1" if dx < w - 2 else "E0"
                if (y + dx * 7) % 13 == 0:
                    c = "E0"
                setp(img, x + dx, y, c)
        for dx in range(-3, w + 3):
            setp(img, x + dx, 100, "E1")
            setp(img, x + dx, 101, "G1")
        for i in range(3):
            setp(img, x - 1 - i, 99 + i // 2, "E1")
            setp(img, x + w + i, 99 + i // 2, "E0")
    # dirt patches, stones and fallen leaves in the clearing
    for _ in range(14):
        cx, cy = rnd.randint(30, 290), rnd.randint(108, 170)
        rx, ry = rnd.randint(6, 16), rnd.randint(2, 4)
        for y in range(cy - ry, cy + ry + 1):
            for x in range(cx - rx, cx + rx + 1):
                if ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2 < 1 and rnd.random() < 0.8:
                    setp(img, x, y, "E2" if rnd.random() < 0.7 else "E3")
    for _ in range(40):
        x, y = rnd.randint(16, 304), rnd.randint(104, 175)
        setp(img, x, y, rnd.choice(["W4", "W3", "O2", "G5"]))
        if rnd.random() < 0.4:
            setp(img, x + 1, y, "W2")
    # shafts of light from the top-left
    for sx in (60, 170, 240):
        for y in range(0, 180):
            for x in range(sx + y // 2 - 6, sx + y // 2 + 6):
                if 0 <= x < 320 and BAYER[y % 4, x % 4] < 0.22:
                    cur = tuple(img[y, x, :3])
                    lift = {RGB["G0"]: "G2", RGB["G1"]: "G3", RGB["G2"]: "G4", RGB["G3"]: "G5",
                            RGB["G4"]: "G6", RGB["E2"]: "E3", RGB["E1"]: "E2", RGB["B1"]: "B2"}
                    if cur in lift:
                        setp(img, x, y, lift[cur])
    # foreground framing trees and ferns at the edges
    for x0, w in ((0, 12), (308, 12)):
        for y in range(180):
            for dx in range(w):
                c = "E1" if dx in (2, 3) and x0 == 0 else "E0"
                if x0 == 0 and dx > w - 3:
                    c = "W0"
                if x0 > 0 and dx < 2:
                    c = "E1"
                setp(img, x0 + dx, y, c)
    for x in range(0, 320, 5):
        if 16 < x < 304 and rnd.random() < 0.7:
            continue
        h = rnd.randint(6, 12)
        for i in range(h):
            setp(img, x + i // 3, 179 - i, "G2" if i < h - 2 else "G4")
            setp(img, x - i // 3, 179 - i, "G1")
    return img


def style_battle():
    img = forest_background()
    # formation: company on the left, foes on the right (mirrored to face left)
    w = battle_units.warrior().render()
    cl = battle_units.cleric().render()
    wz = battle_units.wizard().render()
    og = flip(battle_units.ogre().render(flip_light=True))
    gb = flip(battle_units.goblin().render(flip_light=True))
    units = [(wz, 18, 82), (w, 76, 96), (cl, 44, 112), (gb, 246, 76), (og, 160, 82), (gb, 236, 112)]
    for arr, x, y in units:
        foot = arr.shape[0] - 2
        xs = [i for i in range(arr.shape[1]) if arr[foot - 4:foot + 1, i, 3].any()]
        cx, half = (xs[0] + xs[-1]) // 2, (xs[-1] - xs[0]) // 2 + 3
        for yy in range(-2, 3):
            for xx in range(-half, half + 1):
                if (xx / half) ** 2 + (yy / 2.5) ** 2 < 1:
                    X, Y = x + cx + xx, y + foot + yy
                    cur = tuple(img[Y, X, :3])
                    darker = {RGB["G4"]: "G3", RGB["G3"]: "G2", RGB["G2"]: "G1", RGB["E2"]: "E1", RGB["E3"]: "E2",
                              RGB["G5"]: "G4", RGB["G6"]: "G5"}
                    if cur in darker:
                        setp(img, X, Y, darker[cur])
    for arr, x, y in sorted(units, key=lambda u: u[2] + u[0].shape[0]):
        paste(img, arr, x, y)
    return img
