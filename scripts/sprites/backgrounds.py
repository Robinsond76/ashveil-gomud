"""Battle backgrounds (art set S3): 320x180, opaque, painted from the palette.

Layout rules from the specification: the company stands on the left and the
enemy on the right.  The ground band (y 100-176, x 16-304) is kept clear and
quiet, so units and their shadows read against it; the horizon, scenery and
depth cues sit above y 100.  Backgrounds are painted day-lit; the client
darkens them for darkness and night.
"""
import math

from kit import dither, noise, limb, poly
from pixels import Canvas, rect, line, ellipse, thick

W, H = 320, 180
HORIZON = 100
BAND_TOP, BAND_BOTTOM = 100, 176


def new(fill="stone.m"):
    return Canvas(W, H, fill)


def grad(cv, y0, y1, c0, c1, x0=0, x1=W):
    """Dithered vertical blend from c0 (top) to c1 (bottom)."""
    for y in range(y0, y1):
        t = (y - y0) / max(1, y1 - y0 - 1)
        for x in range(x0, x1):
            cv.put(x, y, c1 if dither(x, y, t) else c0)


def ridge(cv, base, amp, f1, f2, seed, color, y_end, phase=0.0):
    for x in range(W):
        h = base - amp * (0.6 * math.sin(x / f1 + seed + phase) + 0.4 * math.sin(x / f2 + seed * 2))
        h += 3 * noise(x // 3, seed, 1)
        for y in range(int(h), y_end):
            cv.put(x, y, color)


def ground(cv, y0=HORIZON, y1=H, base="grass.m", dark="grass.d", light="grass.l", seed=7, density=0.05):
    """A quiet ground plane: one tone with sparse flecks, a darker lower edge."""
    for y in range(y0, y1):
        for x in range(W):
            n = noise(x, y, seed)
            c = base
            if n < density:
                c = dark
            elif n > 1 - density * 0.6:
                c = light
            cv.put(x, y, c)
    # a darker far edge where the ground meets the horizon
    for x in range(W):
        cv.put(x, y0, dark)
        if x % 2 == 0:
            cv.put(x, y0 + 1, dark)


def tuft_row(cv, y, color, seed, step=9):
    for x in range(2, W - 2, step):
        xx = x + int(noise(x, y, seed) * 5)
        cv.put(xx, y, color)
        cv.put(xx - 1, y - 1, color)


def pine(cv, x, base, h, c_dark, c_mid):
    top = base - h
    for y in range(top, base):
        wid = (y - top) // 7
        for xx in range(x - wid, x + wid + 1):
            cv.put(xx, y, c_mid if xx <= x else c_dark)


def trunk(cv, x, y0, y1, w, dark, mid):
    for y in range(y0, y1):
        for xx in range(x, x + w):
            cv.put(xx, y, mid if xx < x + w - 1 else dark)


def cloud_band(cv, y, c, seed, thick_=3):
    """Soft cloud banks: flat-bottomed puffs, lit on top, shaded underneath."""
    shade = {"wool.l": "wool.d", "slate.l": "slate.m", "steel.l": "steel.m", "stone.m": "stone.d"}.get(c, c)
    n = 4 + seed % 3
    for i in range(n):
        cx = int((i + 0.5) * W / n + (noise(i, seed, 5) - 0.5) * 40)
        for k in range(3):
            rx = 14 + int(noise(i, k, seed) * 18)
            ry = 3 + int(noise(k, i, seed) * 3) + thick_ // 2
            ox = cx + (k - 1) * 14
            oy = y + int(noise(k, seed, i) * 4)
            blob = ellipse(ox, oy, rx, ry)
            for px, py in blob:
                if py <= oy + ry - 2:
                    cv.put(px, py, c if py < oy else shade if dither(px, py, 0.3) else c)
                elif dither(px, py, 0.5):
                    cv.put(px, py, shade)


# ---------------------------------------------------------------------------

def forest():
    cv = new()
    grad(cv, 0, 60, "forest.d", "forest.m")
    grad(cv, 60, HORIZON, "forest.m", "moss.d")
    for layer, (c, top) in enumerate((("forest.m", 52), ("forest.d", 40))):
        for i in range(15):
            x = 6 + i * 22 + int(noise(i, layer, 3) * 10)
            pine(cv, x, HORIZON + 4, top + int(noise(i, 9, layer) * 24), "forest.d", c)
    # tall dark trunks and shafts of light
    for i, x in enumerate((20, 78, 150, 224, 292)):
        trunk(cv, x, 0, HORIZON + 6, 9 + (i % 2) * 3, "leather.d", "leather.d")
        cv.fill(rect(x + 1, 0, x + 2, HORIZON + 4), "leather.m")
    for k, x0 in enumerate((46, 120, 196, 262)):
        for y in range(0, HORIZON - 2):
            for dx in range(6):
                xx = x0 + y // 3 + dx
                if (xx + y) % 2 == 0 and noise(xx, y, k) > 0.3:
                    cv.put(xx, y, "wool.d" if y % 4 else "moss.l")
    ground(cv, base="moss.m", dark="moss.d", light="moss.m", seed=3, density=0.07)
    # moss and leaf litter, only dark flecks
    for x in range(0, W, 7):
        cv.put(x + int(noise(x, 1, 4) * 5), HORIZON + 3 + int(noise(x, 2, 4) * 6), "forest.d")
    return cv


def deep_web():
    cv = forest()
    # a pale green haze over the backdrop
    for y in range(0, HORIZON):
        for x in range(W):
            if dither(x, y, 0.35 + 0.35 * y / HORIZON) and cv.get(x, y) in ("forest.d", "forest.m", "moss.d"):
                cv.put(x, y, "moss.d" if y > 40 else "forest.m")
    # cocoons hang on silk
    for x, y0, h in ((48, 0, 52), (136, 0, 40), (212, 0, 60), (276, 0, 34)):
        cv.fill(line(x, y0, x, y0 + h), "bone.d")
        cv.part(ellipse(x, y0 + h + 7, 4, 8), "wool")
    # silk strands slung between the trunks
    for x0, x1, y0, y1 in ((24, 84, 20, 52), (84, 156, 52, 18), (156, 232, 22, 58), (232, 296, 58, 26),
                           (20, 150, 70, 78), (150, 296, 80, 70)):
        n = x1 - x0
        for i in range(n):
            x = x0 + i
            y = y0 + (y1 - y0) * i / n + 5 * math.sin(i / n * math.pi)
            if i % 2 == 0 or n < 80:
                cv.put(x, int(y), "bone.d")
    for x, y, r in ((60, 40, 10), (190, 36, 12), (270, 60, 9)):  # webs
        for a in range(0, 360, 45):
            cv.fill(line(x, y, x + int(r * math.cos(math.radians(a))), y + int(r * math.sin(math.radians(a)))), "wool.d")
        for rr in (r // 3, 2 * r // 3):
            for a in range(0, 360, 12):
                cv.put(x + int(rr * math.cos(math.radians(a))), y + int(rr * math.sin(math.radians(a))), "bone.d")
    ground(cv, base="forest.m", dark="forest.d", light="moss.d", seed=5, density=0.08)
    for x in range(10, W, 23):  # a few ground strands, dark
        cv.fill(line(x, HORIZON + 6, x + 12, HORIZON + 8), "forest.d")
    return cv


def plains():
    cv = new()
    grad(cv, 0, 54, "slate.l", "slate.l")
    grad(cv, 20, 70, "slate.l", "wool.d")
    grad(cv, 70, HORIZON, "wool.d", "wool.l")
    cloud_band(cv, 22, "wool.l", 1)
    cloud_band(cv, 38, "wool.l", 2, 2)
    ridge(cv, 82, 10, 28, 11, 1, "slate.m", HORIZON)
    ridge(cv, 92, 6, 19, 7, 2, "ashmoss.m", HORIZON)
    ridge(cv, 97, 3, 13, 5, 3, "grass.d", HORIZON + 1)
    ground(cv, base="grass.m", dark="grass.d", light="grass.m", seed=11, density=0.06)
    for x, y in ((30, 92), (66, 95), (250, 93), (300, 90)):  # scattered stones on the far ridge
        cv.part(ellipse(x, y, 3, 2), "stone")
    tuft_row(cv, HORIZON + 4, "grass.d", 2)
    return cv


def road():
    cv = plains()
    # distant trees and a milestone
    for x, h in ((20, 14), (52, 17), (96, 12), (240, 15), (270, 19), (300, 13)):
        cv.part(ellipse(x, 90 - h // 2, 7, h // 2 + 2), "forest")
        cv.fill(rect(x - 1, 89, x, 98), "leather.d")
    cv.part(rect(286, 84, 291, 98), "stone")
    cv.fill({(288, 87), (289, 87), (288, 90)}, "stone.d")
    # a rutted road through the field: ochre band with two wheel ruts, kept low contrast
    for y in range(HORIZON, H):
        t = (y - HORIZON) / (H - HORIZON)
        half = 54 + t * 120
        c = 160
        for x in range(int(c - half), int(c + half)):
            n = noise(x, y, 31)
            cv.put(x, y, "ochre.d" if n < 0.06 else "ochre.m")
        # verges
        cv.put(int(c - half) - 1, y, "grass.d")
        cv.put(int(c + half), y, "grass.d")
        for rut in (-0.35, 0.35):
            rx = int(c + rut * half)
            for dx in (0, 1):
                if (y // 2) % 5:
                    cv.put(rx + dx, y, "ochre.d")
    return cv


def _building(cv, x, w, top, base, wall, roof, trim=None, windows=True):
    cv.part(rect(x, top, x + w - 1, base), wall)
    cv.part(poly([(x - 3, top), (x + w + 2, top), (x + w // 2, top - w // 3)]), roof)
    if windows:
        for wx in range(x + 4, x + w - 5, 9):
            for wy in range(top + 6, base - 8, 14):
                cv.fill(rect(wx, wy, wx + 3, wy + 5), "charcoal.d")
                cv.fill({(wx, wy)}, "ochre.d")


def city():
    cv = new()
    grad(cv, 0, HORIZON, "slate.m", "slate.l")
    cloud_band(cv, 14, "slate.l", 4)
    # a street between timber and stone buildings
    for x, w, top, wall, roof in ((0, 40, 36, "wood", "oxblood"), (42, 36, 52, "stone", "slate"),
                                  (80, 34, 60, "wood", "leather"), (116, 28, 70, "stone", "oxblood"),
                                  (176, 30, 66, "stone", "slate"), (208, 36, 54, "wood", "oxblood"),
                                  (246, 34, 44, "stone", "leather"), (282, 40, 30, "wood", "slate")):
        _building(cv, x, w, top, HORIZON + 4, wall + ".d" if wall == "wood" else wall + ".m", roof + ".d")
    # distant keep wall in the gap
    cv.part(rect(146, 78, 172, HORIZON + 4), "stone")
    for x in range(146, 172, 6):
        cv.fill(rect(x, 74, x + 3, 78), "stone.d")
    ground(cv, base="stone.m", dark="stone.d", light="stone.m", seed=13, density=0.07)
    for y in range(HORIZON + 6, H, 9):  # cobble courses, dark and faint
        for x in range((y // 9 % 2) * 6, W, 12):
            cv.put(x, y, "stone.d")
            cv.put(x + 1, y, "stone.d")
    # barrels and crates tucked at the extreme edges only
    for x in (2, 304):
        cv.part(rect(x, 86, x + 9, 99), "wood")
        cv.fill(line(x, 90, x + 9, 90), "iron.m")
        cv.fill(line(x, 96, x + 9, 96), "iron.m")
    cv.part(rect(8, 90, 17, 99), "wood")
    return cv


def slums():
    cv = new()
    grad(cv, 0, HORIZON, "stone.d", "ashmoss.d")
    cloud_band(cv, 10, "stone.m", 6)
    # sagging shacks leaning in over a narrow alley
    for i, (x, w, top, lean) in enumerate(((0, 52, 40, 3), (46, 40, 58, -2), (84, 30, 72, 1),
                                           (206, 32, 64, -1), (236, 40, 50, 2), (270, 50, 38, -3))):
        for y in range(top, HORIZON + 4):
            sh = int(lean * (y - top) / 20)
            for xx in range(x + sh, x + w + sh):
                cv.put(xx, y, "wood.d" if (xx // 3) % 2 else "leather.d")
        cv.fill(poly([(x - 2, top + 2), (x + w + 2, top + 2), (x + w // 2 + lean * 2, top - 8)]), "leather.m")
        for wx in range(x + 6, x + w - 6, 12):
            cv.fill(rect(wx, top + 12, wx + 3, top + 17), "charcoal.d")
    # laundry lines with grey cloth
    for x0, x1, y in ((52, 210, 54), (86, 236, 70)):
        for x in range(x0, x1):
            cv.put(x, y + int(3 * math.sin((x - x0) / (x1 - x0) * math.pi)), "iron.d")
        for x in range(x0 + 14, x1 - 10, 28):
            yy = y + int(3 * math.sin((x - x0) / (x1 - x0) * math.pi))
            cv.part(rect(x, yy + 1, x + 7, yy + 9), "wool" if (x // 28) % 2 else "plum")
    ground(cv, base="leather.d", dark="charcoal.m", light="leather.m", seed=17, density=0.07)
    for x in range(14, W, 41):  # puddles, dark
        cv.fill(ellipse(x + int(noise(x, 3, 3) * 14), 118 + int(noise(x, 4, 3) * 40), 7, 1.5), "charcoal.m")
    return cv


def interior():
    cv = new()
    # plank wall and ceiling
    for y in range(0, HORIZON):
        for x in range(W):
            n = noise(x, y // 20, 5)
            cv.put(x, y, "wood.d" if (x // 11) % 2 else "leather.m")
            if x % 11 == 0:
                cv.put(x, y, "leather.d")
    for x in range(0, W, 64):  # ceiling beams and posts
        cv.fill(rect(x + 20, 0, x + 27, HORIZON + 2), "leather.d")
    cv.fill(rect(0, 18, W, 24), "leather.d")
    cv.fill(rect(0, 18, W, 19), "leather.m")
    # hearth in the middle, with a low fire
    cv.part(rect(118, 40, 202, HORIZON + 2), "stone")
    cv.fill(rect(130, 56, 190, HORIZON + 2), "charcoal.d")
    cv.fill(rect(120, 40, 200, 44), "stone.d")
    for x in range(138, 184, 5):
        cv.part(ellipse(x, 94, 3, 5 + int(noise(x, 2, 2) * 4)), "ember")
    cv.fill(rect(126, 46, 194, 47), "stone.l")
    # shelves and mugs at the edges
    for x0 in (8, 262):
        cv.fill(rect(x0, 50, x0 + 46, 53), "wood.m")
        cv.fill(rect(x0, 78, x0 + 46, 81), "wood.m")
        for k in range(5):
            cv.part(rect(x0 + 4 + k * 9, 42, x0 + 8 + k * 9, 49), "iron" if k % 2 else "leather")
    # tables and benches pushed aside, at the extreme edges
    for x in (0, 300):
        cv.part(rect(x, 86, x + 19, 90), "wood")
        cv.fill(rect(x + 2, 90, x + 3, 99), "wood.d")
        cv.fill(rect(x + 16, 90, x + 17, 99), "wood.d")
    ground(cv, base="wood.d", dark="leather.d", light="wood.m", seed=19, density=0.05)
    for y in range(HORIZON + 4, H, 10):  # floorboards, faint
        for x in range(0, W):
            if (x + y) % 3:
                cv.put(x, y, "leather.d")
    for x in range(24, W, 46):
        for y in range(HORIZON + 4, H, 10):
            cv.put(x + (y // 10 % 2) * 20, y + 1, "wood.m")
    return cv


def catacombs():
    cv = new()
    for y in range(0, HORIZON):
        for x in range(W):
            cv.put(x, y, "stone.d" if noise(x // 2, y // 2, 8) > 0.12 else "charcoal.m")
    for y in range(8, HORIZON, 14):  # masonry courses
        for x in range(W):
            if (x + y) % 2 == 0:
                cv.put(x, y, "charcoal.m")
    # arched burial niches with bone piles
    for x in range(6, W - 30, 40):
        for y in (14, 54):
            cv.fill(rect(x, y + 6, x + 25, y + 27), "charcoal.d")
            cv.fill(ellipse(x + 12, y + 7, 13, 7), "charcoal.d")
            for k in range(7):
                bx = x + 3 + int(noise(x, k, y) * 18)
                cv.fill(line(bx, y + 25, bx + 4, y + 24 - int(noise(k, y, x) * 3)), "bone.m")
            cv.part(ellipse(x + 10, y + 23, 3, 2.4), "bone")
    # candles in skull sconces: a faint warm light
    for x in (4, 82, 162, 244, 308):
        cv.part(ellipse(x, 80, 3, 3), "bone")
        cv.fill({(x - 1, 80), (x + 1, 80)}, "outline")
        cv.fill({(x, 76), (x, 75)}, "ember.m")
        cv.put(x, 74, "ember.l")
    ground(cv, base="stone.d", dark="charcoal.m", light="stone.m", seed=23, density=0.05)
    for y in range(HORIZON + 8, H, 14):  # flagstones, faint
        for x in range(W):
            if (x * 3 + y) % 7 == 0:
                cv.put(x, y, "charcoal.m")
    for x in range(0, W, 28):
        for y in range(HORIZON + 8, H - 14, 28):
            cv.fill(line(x + (y // 28 % 2) * 14, y, x + (y // 28 % 2) * 14, y + 14), "charcoal.m")
    return cv


def cave():
    cv = new()
    for y in range(0, HORIZON):
        for x in range(W):
            n = noise(x // 3, y // 3, 3)
            cv.put(x, y, "charcoal.m" if y < 40 else "stone.d" if n > 0.15 else "charcoal.m")
    grad(cv, 56, HORIZON, "stone.d", "stone.m")
    # stalactites
    for x in range(4, W, 15):
        h = 14 + int(noise(x, 1, 5) * 30)
        for y in range(0, h):
            wid = max(0, (h - y) // 6)
            for xx in range(x - wid, x + wid + 1):
                cv.put(xx, y, "stone.m" if xx <= x else "stone.d")
    # glowing crystal veins
    for x0, y0, x1, y1 in ((20, 70, 90, 46), (130, 58, 200, 80), (230, 48, 304, 70)):
        cv.fill(line(x0, y0, x1, y1), "heather.d")
        for k in range(0, x1 - x0, 6):
            x = x0 + k
            y = y0 + (y1 - y0) * k // (x1 - x0)
            cv.fill({(x, y - 1), (x + 1, y - 2)}, "heather.l")
            cv.put(x, y, "water.l")
    # a dark pool in the back
    cv.fill(ellipse(160, 94, 70, 5), "water.d")
    for x in range(110, 210, 7):
        cv.put(x, 93, "water.m")
    ground(cv, base="stone.d", dark="charcoal.m", light="stone.m", seed=29, density=0.06)
    return cv


def snowfield():
    cv = new()
    grad(cv, 0, HORIZON, "steel.m", "steel.l")
    cloud_band(cv, 24, "steel.l", 9)
    cloud_band(cv, 40, "bone.l", 10, 2)
    ridge(cv, 86, 8, 30, 9, 4, "slate.l", HORIZON)
    ridge(cv, 94, 5, 17, 6, 5, "steel.m", HORIZON)
    # frost-heavy pines
    for i in range(12):
        x = 10 + i * 27 + int(noise(i, 5, 5) * 12)
        h = 30 + int(noise(i, 6, 5) * 24)
        pine(cv, x, HORIZON + 2, h, "forest.d", "forest.m")
        for k in range(h // 5):  # snow on the boughs
            yy = HORIZON + 2 - h + k * 6 + 4
            cv.fill(line(x - k - 1, yy, x - 1, yy), "bone.l")
    ground(cv, base="steel.l", dark="steel.m", light="bone.l", seed=31, density=0.06)
    for x in range(0, W, 38):  # drifts: long low blue shadows
        yy = 112 + int(noise(x, 8, 8) * 55)
        cv.fill(line(x, yy, x + 22, yy), "steel.m")
    return cv


def ice_keep():
    cv = new()
    grad(cv, 0, HORIZON, "slate.d", "slate.m")
    # ice-sheathed stone arches and frozen pillars
    for x in range(-10, W, 64):
        cv.part(rect(x + 8, 0, x + 22, HORIZON + 4), "steel")
        cv.fill(rect(x + 9, 0, x + 11, HORIZON + 2), "bone.l")
        cv.fill(rect(x + 20, 0, x + 22, HORIZON + 2), "slate.m")
        arch_y = 28
        for k in range(0, 32):
            a = k / 32 * math.pi
            cv.put(x + 22 + k, arch_y - int(10 * math.sin(a)), "steel.l")
            cv.put(x + 22 + k, arch_y - int(10 * math.sin(a)) + 1, "steel.m")
    for x in range(6, W, 17):  # icicles
        h = 6 + int(noise(x, 3, 6) * 14)
        for y in range(0, h):
            cv.put(x, y, "steel.l")
            if y < h // 2:
                cv.put(x + 1, y, "steel.m")
    for x in range(30, W, 90):  # frosted banners
        cv.part(rect(x, 46, x + 9, 74), "slate")
        cv.fill(rect(x, 46, x + 9, 48), "steel.l")
    ground(cv, base="steel.m", dark="slate.m", light="steel.l", seed=37, density=0.06)
    for x in range(0, W, 21):  # cracks in the ice, faint
        yy = 108 + int(noise(x, 2, 9) * 66)
        cv.fill(line(x, yy, x + 9, yy + 2), "slate.m")
    return cv


def shore():
    cv = new()
    grad(cv, 0, 56, "slate.l", "wool.d")
    grad(cv, 56, 80, "wool.d", "wool.l")
    cloud_band(cv, 18, "wool.l", 14)
    ridge(cv, 70, 7, 35, 12, 7, "slate.m", 80)
    # the lake, with mist over it
    for y in range(78, HORIZON):
        t = (y - 78) / (HORIZON - 78)
        for x in range(W):
            n = noise(x // 4, y, 7)
            cv.put(x, y, "water.m" if t > 0.3 else "water.l")
            if n > 0.93:
                cv.put(x, y, "water.l")
    for y in range(74, 90):
        for x in range(W):
            if dither(x, y, 0.45 - (y - 74) * 0.03):
                cv.put(x, y, "wool.l")
    ground(cv, base="ochre.m", dark="ochre.d", light="ochre.m", seed=41, density=0.05)
    # a wet line of shingle, then reeds and an upturned boat at the edges
    for x in range(W):
        cv.put(x, HORIZON + int(2 * math.sin(x / 11)), "stone.m")
    for x in list(range(2, 40, 4)) + list(range(284, 316, 4)):
        h = 9 + int(noise(x, 3, 3) * 10)
        cv.fill(line(x, 99, x + int(noise(x, 4, 3) * 3) - 1, 99 - h), "grass.m")
    cv.part(poly([(14, 98), (50, 98), (44, 88), (20, 88)]), "wood")
    cv.fill(line(14, 98, 50, 98), "wood.d")
    return cv


def swamp():
    cv = new()
    grad(cv, 0, HORIZON, "forest.d", "ashmoss.d")
    for x in range(W):
        if noise(x // 3, 1, 3) > 0.4:
            for y in range(0, 30 + int(noise(x // 4, 2, 3) * 20)):
                if (x + y) % 3 == 0:
                    cv.put(x, y, "moss.d")
    # twisted roots and trunks, hanging moss
    for x0 in (30, 100, 180, 260):
        pts = [(x0, HORIZON + 2), (x0 - 4, 70), (x0 + 5, 40), (x0 - 2, 6)]
        for a, b in zip(pts, pts[1:]):
            cv.fill(limb(a, b, 5), "leather.d")
        for k in range(5):
            cv.fill(line(x0 + k * 3 - 6, 18 + k * 4, x0 + k * 3 - 6 + int(noise(k, x0, 2) * 3), 48 + k * 6), "moss.m")
    cv.fill(rect(0, 84, W, HORIZON), "water.d")
    for x in range(0, W, 7):
        cv.put(x, 88 + (x // 7) % 3, "water.m")
    ground(cv, base="forest.d", dark="charcoal.m", light="forest.m", seed=43, density=0.07)
    for x in range(0, W, 53):  # black water patches
        cv.fill(ellipse(x + 20, 120 + int(noise(x, 5, 4) * 50), 14, 3), "water.d")
    return cv


def desert():
    cv = new()
    grad(cv, 0, 60, "ochre.l", "wool.l")
    grad(cv, 60, HORIZON, "wool.l", "ochre.l")
    # heat shimmer: wavy dither bands near the horizon
    for y in range(84, HORIZON):
        for x in range(W):
            if int(2 * math.sin(x / 9 + y)) == 0 and dither(x, y, 0.4):
                cv.put(x, y, "wool.l")
    ridge(cv, 90, 9, 38, 13, 11, "ochre.m", HORIZON)
    ridge(cv, 96, 5, 21, 8, 12, "ochre.d", HORIZON)
    # a sun-bleached ruin: a broken column and a toppled drum
    cv.part(rect(40, 58, 49, 98), "bone")
    cv.fill(rect(38, 54, 51, 58), "bone.m")
    for y in range(60, 96, 6):
        cv.fill(line(41, y, 48, y), "bone.d")
    cv.fill({(44, 54), (46, 53), (48, 54)}, "outline")
    cv.part(ellipse(70, 96, 8, 3), "bone")
    cv.part(rect(250, 70, 257, 98), "bone")
    cv.fill(rect(248, 66, 259, 70), "bone.m")
    ground(cv, base="ochre.m", dark="ochre.d", light="ochre.m", seed=47, density=0.04)
    for y in range(108, H, 9):  # wind ripples, faint
        for x in range(W):
            if int(2.5 * math.sin(x / 13 + y / 4)) == 0 and (x % 3):
                cv.put(x, y, "ochre.d")
    return cv


def highlands():
    cv = new()
    grad(cv, 0, 50, "slate.l", "slate.l")
    grad(cv, 30, HORIZON, "slate.l", "wool.d")
    cloud_band(cv, 14, "wool.l", 15)
    # a cliff face behind and a rocky pass
    ridge(cv, 60, 18, 40, 12, 3, "stone.m", HORIZON)
    for x in range(0, W):
        h = 40 + 20 * math.sin(x / 40) + 6 * noise(x // 2, 6, 6)
        for y in range(int(h), HORIZON):
            band = (y + int(3 * math.sin(x / 23))) // 7
            if band % 3 == 0 and dither(x, y, 0.5):
                cv.put(x, y, "stone.d")
            elif noise(x // 2, y, 9) > 0.95:
                cv.put(x, y, "stone.l")
    ridge(cv, 90, 6, 17, 6, 6, "stone.d", HORIZON + 1)
    ground(cv, base="stone.m", dark="stone.d", light="stone.m", seed=53, density=0.06)
    for x in range(0, W, 5):  # scree
        yy = HORIZON + 4 + int(noise(x, 7, 7) * 70)
        cv.fill({(x, yy), (x + 1, yy), (x, yy - 1)}, "stone.d")
    return cv


def training_yard():
    cv = new()
    grad(cv, 0, HORIZON, "slate.l", "wool.d")
    cloud_band(cv, 18, "wool.l", 16)
    ridge(cv, 82, 6, 30, 10, 8, "ashmoss.m", HORIZON)
    # a fence around the yard, with straw targets and weapon racks at the edges
    cv.fill(rect(0, 80, W, 83), "wood.m")
    cv.fill(rect(0, 92, W, 94), "wood.m")
    for x in range(0, W, 16):
        cv.part(rect(x, 76, x + 3, HORIZON + 2), "wood")
    for x, ph in ((176, 0), (212, 1), (250, 2)):  # targets on posts
        cv.part(rect(x + 5, 60, x + 7, HORIZON), "wood")
        cv.part(ellipse(x + 6, 58, 10, 10), "ochre")
        cv.fill(ellipse(x + 6, 58, 6, 6), "oxblood.m")
        cv.fill(ellipse(x + 6, 58, 2, 2), "ochre.l")
    for x in (6, 40):  # weapon racks
        cv.fill(rect(x, 66, x + 28, 69), "wood.m")
        cv.fill(rect(x, 69, x + 2, 99), "wood.d")
        cv.fill(rect(x + 26, 69, x + 28, 99), "wood.d")
        for k in range(5):
            cv.fill(line(x + 4 + k * 5, 56 + k % 2 * 4, x + 4 + k * 5, 90), "steel.m" if k % 2 else "wood.m")
    ground(cv, base="ochre.d", dark="leather.d", light="ochre.m", seed=59, density=0.06)
    return cv


BACKGROUNDS = {
    "forest": (forest, ["forest"]),
    "deep-web": (deep_web, ["spiderweb"]),
    "plains": (plains, ["land", "farmland", "default"]),
    "road": (road, ["road"]),
    "city": (city, ["city", "fort"]),
    "slums": (slums, ["slums"]),
    "interior": (interior, ["house"]),
    "catacombs": (catacombs, ["dungeon"]),
    "cave": (cave, ["cave"]),
    "snowfield": (snowfield, ["snow"]),
    "ice-keep": (ice_keep, []),
    "shore": (shore, ["shore", "water"]),
    "swamp": (swamp, ["swamp"]),
    "desert": (desert, ["desert"]),
    "highlands": (highlands, ["mountains", "cliffs"]),
    "training-yard": (training_yard, []),
}
