"""Terrain tiles, animated overlays and fog/state tiles (art set S2).

Tiles are 32x32 with three variants side by side.  They are deliberately
quiet: a flat mid tone with sparse detail in two shade steps and no outline,
so a unit sprite (outlined, higher contrast) never blends into its tile.
Detail stays inside a 2 px margin, which is kept flat so tiles sit beside
any neighbour without a visible seam.
"""
import math

from kit import dither, noise, limb, poly
from pixels import Canvas, rect, line, ellipse

T = 32
M = 2  # flat margin on every edge


class Tile:
    """A 32x32 canvas whose `dot` helpers respect the flat margin."""

    def __init__(self, base, seed=0):
        self.cv = Canvas(T, T, base)
        self.base, self.seed = base, seed

    def inner(self, x, y):
        return M <= x < T - M and M <= y < T - M

    def dot(self, x, y, c):
        if self.inner(x, y):
            self.cv.put(x, y, c)

    def fill(self, pixels, c):
        for x, y in pixels:
            self.dot(x, y, c)

    def scatter(self, c, density, salt=0, only=None):
        for y in range(M, T - M):
            for x in range(M, T - M):
                if noise(x, y, self.seed * 13 + salt) < density and (only is None or only(x, y)):
                    self.cv.put(x, y, c)

    def speck(self, n, c, salt=0, w=1):
        """n small flecks of width w, scattered by seed."""
        for i in range(n):
            x = M + int(noise(i, salt, self.seed + 1) * (T - 2 * M - w))
            y = M + int(noise(salt, i, self.seed + 2) * (T - 2 * M - 1))
            for k in range(w):
                self.dot(x + k, y, c)


def _row(tiles):
    out = Canvas(T * len(tiles), T)
    for i, t in enumerate(tiles):
        out.blit(t, i * T, 0)
    return out


def variants(fn):
    return _row([fn(s) for s in range(3)])


# -- the biomes --------------------------------------------------------------------------------

def cave(s):
    t = Tile("stone.d", s)
    t.scatter("stone.m", 0.07)
    t.scatter("charcoal.m", 0.04, 1)
    for i in range(3 + s):  # scattered rubble
        x = M + 3 + int(noise(i, 1, s) * 22)
        y = M + 3 + int(noise(i, 2, s) * 22)
        t.fill(ellipse(x, y, 2, 1.5), "stone.m")
        t.dot(x - 1, y - 1, "stone.l")
    t.speck(2 + s, "heather.l", 9)  # faint mineral glints
    t.speck(1 + s % 2, "water.l", 8)
    return t.cv


def city(s):
    t = Tile("stone.m", s)
    for row in range(M, T - M, 6):
        off = (row // 6 + s) % 2 * 4
        for x in range(M, T - M):
            t.dot(x, row, "stone.d")
        for x in range(M + off, T - M, 8):
            for y in range(row, min(row + 6, T - M)):
                t.dot(x, y, "stone.d")
    t.scatter("stone.l", 0.04, 2)
    if s == 2:  # a drain
        t.fill(rect(13, 13, 19, 19), "charcoal.m")
        for x in (14, 16, 18):
            t.fill(rect(x, 14, x, 18), "charcoal.d")
    return t.cv


def cliffs(s):
    t = Tile("stone.m", s)
    for y in range(M, 23):
        t.fill({(x, y) for x in range(M, T - M) if dither(x, y, 0.0)}, "stone.m")
    # a darker lower ledge and its drop shadow
    for y in range(24, T - M):
        for x in range(M, T - M):
            t.cv.put(x, y, "stone.d")
    t.fill({(x, 23) for x in range(M, T - M)}, "stone.l")
    for i in range(3):  # cracks
        x = 6 + i * 9 + int(noise(i, 5, s) * 5)
        t.fill(line(x, M + 2, x + int(noise(i, 6, s) * 6) - 3, 21), "stone.d")
    t.scatter("stone.l", 0.04, 3, only=lambda x, y: y < 23)
    for y in range(T - M, T):  # the margin takes the shadow's tone so tiles stack
        for x in range(T):
            t.cv.put(x, y, "stone.d")
    return t.cv


def default(s):
    t = Tile("ochre.d", s)
    t.scatter("leather.m", 0.08)
    t.scatter("ochre.m", 0.05, 1)
    t.speck(3, "stone.m", 4)
    return t.cv


def desert(s):
    t = Tile("ochre.m", s)
    for k in range(3):  # wind ripples: shallow arcs
        y0 = 7 + k * 9 + s * 2
        for x in range(M + 1, T - M - 1):
            y = y0 + int(1.6 * math.sin(x / 4.5 + k + s))
            t.dot(x, y, "ochre.l" if (x // 3) % 2 else "ochre.d")
    t.speck(3, "stone.m", 5)
    t.scatter("ochre.l", 0.03, 2)
    return t.cv


def dungeon(s):
    t = Tile("stone.m", s)
    for y in range(M, T - M):  # fitted flagstones with moss in the joints
        t.dot(M + 13 + (s % 2) * 2, y, "stone.d") if y < 17 else t.dot(M + 5 + s, y, "stone.d")
    for x in range(M, T - M):
        t.dot(x, 17, "stone.d")
        t.dot(x, 17 + (1 if noise(x, 1, s) > 0.8 else 0), "stone.d")
    for i in range(5):
        x = M + int(noise(i, 3, s) * 26)
        y = 16 + int(noise(i, 4, s) * 3)
        t.dot(x, y, "moss.d")
        t.dot(x + 1, y, "moss.m")
    t.scatter("stone.l", 0.03, 4)
    return t.cv


def farmland(s):
    t = Tile("leather.m", s)
    crop = ["grass.m", "grass.l", "ochre.l"][s]
    for r in range(M + 1, T - M, 5):
        t.fill({(x, r) for x in range(M, T - M)}, "leather.d")
        t.fill({(x, r + 1) for x in range(M, T - M)}, "leather.l" if s != 2 else "leather.m")
        for x in range(M + 2 + (r // 5) % 2, T - M, 4):
            h = [1, 2, 2][s]
            t.fill({(x, r - k) for k in range(1, h + 1)}, crop)
            if s == 2:
                t.dot(x, r - 3, "ochre.l")
    return t.cv


def forest(s):
    t = Tile("forest.d", s)
    for i, (cx, cy, r) in enumerate(((9, 9, 7), (22, 10, 6), (10, 22, 6), (23, 23, 7), (16, 16, 4))):
        cx += int(noise(i, 1, s) * 4) - 2
        cy += int(noise(i, 2, s) * 4) - 2
        blob = ellipse(cx, cy, r, r - 1)
        t.fill(blob, "forest.m")
        t.fill({(x, y) for x, y in blob if x < cx and y < cy}, "forest.m")
        t.fill({(cx - 2, cy - 3), (cx - 3, cy - 2)}, "moss.d")
        t.fill({(x, y) for x, y in blob if x > cx + r - 3 or y > cy + r - 4}, "forest.d")
    return t.cv


def fort(s):
    t = Tile("ochre.d", s)
    t.scatter("leather.m", 0.07)
    t.scatter("ochre.m", 0.05, 1)
    for x in range(T):  # a timber edge along the top
        for y in range(0, 5):
            t.cv.put(x, y, "wood.m" if y < 4 else "wood.d")
    for x in range(M + 2, T - M, 6):
        t.cv.put(x, 1, "wood.d")
        t.cv.put(x, 2, "wood.d")
    return t.cv


def house(s):
    t = Tile("wood.m", s)
    for x in range(M + 3, T - M, 6):
        for y in range(M, T - M):
            t.dot(x, y, "wood.d")
    for x in range(M, T - M):
        t.dot(x, M + 9 + (x // 6) * 2 % 5 + s, "wood.d")
    t.scatter("wood.l", 0.04, 3)
    if s == 1:  # a rug corner
        t.fill(rect(M, M, M + 10, M + 8), "oxblood.m")
        t.fill(rect(M, M + 8, M + 10, M + 8), "brass.d")
        t.fill(rect(M + 10, M, M + 10, M + 8), "brass.d")
        t.fill(rect(M + 2, M + 2, M + 8, M + 6), "oxblood.d")
    return t.cv


def land(s):
    t = Tile("grass.m", s)
    t.scatter("grass.d", 0.07)
    t.scatter("grass.l", 0.05, 1)
    for i in range(5 + s * 2):  # tufts
        x = M + 1 + int(noise(i, 1, s) * 26)
        y = M + 2 + int(noise(i, 2, s) * 26)
        t.dot(x, y, "grass.d")
        t.dot(x, y - 1, "grass.l")
    for i in range(3):  # small flowers
        x = M + int(noise(i, 7, s) * 27)
        y = M + int(noise(i, 8, s) * 27)
        t.dot(x, y, "bone.l" if (i + s) % 2 else "ochre.l")
    return t.cv


def mountains(s):
    t = Tile("stone.m", s)
    for k in range(4):  # rocky strata
        y0 = 8 + k * 7
        for x in range(M, T - M):
            t.dot(x, y0 + int(2 * math.sin(x / 6 + k + s)), "stone.d")
    t.scatter("stone.l", 0.05, 1)
    # a snowcap highlight on the upper-left slope
    for y in range(M, 11):
        for x in range(M, 20 - y):
            if dither(x, y, 0.65 - (y - M) * 0.05 + 0.1 * s):
                t.dot(x, y, "steel.l")
    return t.cv


def road(s):
    t = Tile("grass.m", s)
    t.scatter("grass.d", 0.05)
    lo, hi = 9 + s % 2, 22 + s % 2
    for y in range(lo, hi + 1):
        for x in range(T):
            t.cv.put(x, y, "ochre.m")
    for x in range(T):
        t.cv.put(x, lo - 1, "grass.d")
        t.cv.put(x, hi + 1, "ochre.d")
    for x in range(M, T - M):  # wheel ruts and a few pebbles
        if x % 7 != 3:
            t.cv.put(x, lo + 4, "ochre.d")
            t.cv.put(x, hi - 4, "ochre.d")
    t.speck(3, "stone.m", 3)
    t.scatter("ochre.l", 0.03, 2, only=lambda x, y: lo < y < hi)
    return t.cv


def shore(s):
    t = Tile("ochre.m", s)
    for y in range(T):
        for x in range(T):
            d = (x + y) - (T - 1) - [-3, 0, 3][s]
            if d > 2:
                t.cv.put(x, y, "water.m")
            elif d > 0:
                t.cv.put(x, y, "water.l")
            elif d > -2:
                t.cv.put(x, y, "ochre.d")
            else:
                t.cv.put(x, y, "ochre.l" if dither(x, y, 0.08) else "ochre.m")
    for y in range(T):  # deeper water, further from the shore
        for x in range(T):
            if (x + y) - (T - 1) - [-3, 0, 3][s] > 8 and dither(x, y, 0.4):
                t.cv.put(x, y, "water.d")
    return t.cv


def slums(s):
    t = Tile("leather.d", s)
    t.scatter("leather.m", 0.08)
    t.scatter("charcoal.m", 0.04, 1)
    for i in range(3):  # broken boards
        x = M + 2 + int(noise(i, 1, s) * 16)
        y = M + 3 + int(noise(i, 2, s) * 22)
        t.fill(rect(x, y, x + 8, y + 2), "wood.d")
        t.fill(rect(x, y, x + 8, y), "wood.m")
    t.speck(3, "bone.d", 6)  # refuse
    t.speck(2, "oxblood.d", 7)
    return t.cv


def snow(s):
    t = Tile("steel.l", s)
    t.scatter("bone.l", 0.07)
    for k in range(3):  # drifts with blue shadows on their lee sides
        y0 = 8 + k * 9 + s * 2
        for x in range(M + 2, T - M - 2):
            y = y0 + int(2 * math.sin(x / 5 + k))
            t.dot(x, y, "steel.m")
            t.dot(x, y + 1, "slate.l" if dither(x, y, 0.4) else "steel.m")
    return t.cv


def spiderweb(s):
    t = Tile("forest.d", s)
    t.scatter("forest.m", 0.05)
    cx, cy = 14 + s * 3, 15 + (s % 2) * 3
    for a in range(0, 360, 45):
        x1 = cx + int(11 * math.cos(math.radians(a)))
        y1 = cy + int(11 * math.sin(math.radians(a)))
        t.fill(line(cx, cy, x1, y1), "wool.d")
    for r in (4, 8):
        for a in range(0, 360, 10):
            t.dot(cx + int(r * math.cos(math.radians(a))), cy + int(r * math.sin(math.radians(a))), "bone.d")
    for i in range(2):  # cocoons
        x = M + 3 + int(noise(i, 1, s) * 22)
        y = M + 3 + int(noise(i, 2, s) * 22)
        t.fill(ellipse(x, y, 2, 3), "wool.d")
    return t.cv


def swamp(s):
    t = Tile("forest.d", s)
    t.scatter("moss.d", 0.07)
    for i in range(2 + s % 2):  # murky pools
        x = M + 6 + int(noise(i, 1, s) * 16)
        y = M + 6 + int(noise(i, 2, s) * 16)
        t.fill(ellipse(x, y, 6, 4), "water.d")
        t.fill(ellipse(x - 1, y - 1, 3, 2), "water.m")
    for i in range(4):  # reeds
        x = M + 2 + int(noise(i, 3, s) * 26)
        y = M + 6 + int(noise(i, 4, s) * 20)
        t.fill(line(x, y, x, y - 4), "moss.m")
        t.dot(x + 1, y - 4, "moss.l")
    for i in range(2):  # lily pads
        x = M + 5 + int(noise(i, 5, s) * 20)
        y = M + 5 + int(noise(i, 6, s) * 20)
        t.fill(ellipse(x, y, 2, 1), "grass.m")
    return t.cv


def water(s):
    t = Tile("water.m", s)
    t.scatter("water.d", 0.05)
    for k in range(3):
        y0 = 7 + k * 9 + s * 2
        for x in range(M + 1, T - M - 1):
            y = y0 + int(1.5 * math.sin(x / 3.5 + k * 2 + s))
            if (x // 3 + k) % 2 == 0:
                t.dot(x, y, "water.l")
    return t.cv


BIOMES = {
    "cave": cave, "city": city, "cliffs": cliffs, "default": default, "desert": desert,
    "dungeon": dungeon, "farmland": farmland, "forest": forest, "fort": fort, "house": house,
    "land": land, "mountains": mountains, "road": road, "shore": shore, "slums": slums, "snow": snow,
    "spiderweb": spiderweb, "swamp": swamp, "water": water,
}


# -- animated overlays: 4 frames of transparent, hard-edged details drawn over the static tile ---

def _overlay(frame_fn):
    frames = []
    for f in range(4):
        cv = Canvas(T, T)
        frame_fn(cv, f)
        frames.append(cv)
    out = Canvas(T * 4, T)
    for i, fr in enumerate(frames):
        out.blit(fr, i * T, 0)
    return out


def desert_anim(cv, f):
    """Sand drifts right along three streaks."""
    for k, y in enumerate((7, 16, 24)):
        x0 = (f * 3 + k * 9) % 28 + 2
        for dx in range(5):
            cv.put(x0 + dx, y, "ochre.l" if dx % 2 == 0 else "wool.d")


def shore_anim(cv, f):
    """The lapping edge advances and recedes along the diagonal."""
    off = [0, 1, 2, 1][f]
    for x in range(T):
        y = (T - 1) - x - 3 + off
        cv.put(x, y, "bone.l" if (x + f) % 3 else "water.l")
        cv.put(x, y + 1, "water.l")


def snow_anim(cv, f):
    for i in range(7):
        x = (int(noise(i, 1, 4) * 28) - f * 1) % 28 + 2
        y = (int(noise(i, 2, 4) * 28) + f * 7) % 28 + 2
        cv.put(x, y, "bone.l")
        if i % 2:
            cv.put(x + 1, y, "steel.l")


def swamp_anim(cv, f):
    """Bubbles rise from the pools, swell and pop."""
    for i, (bx, by) in enumerate(((9, 14), (21, 20), (15, 8))):
        k = (f + i) % 4
        y = by - k
        if k < 3:
            cv.put(bx, y, "bone.d" if k == 0 else "water.l")
            if k == 2:
                cv.put(bx - 1, y, "water.l")
                cv.put(bx + 1, y, "water.l")
                cv.put(bx, y - 1, "water.l")
        else:
            cv.put(bx - 2, by - 3, "water.l")
            cv.put(bx + 2, by - 3, "water.l")


def water_anim(cv, f):
    """Wave crests slide across the surface."""
    for k, y in enumerate((6, 14, 22, 28)):
        x0 = (f * 4 + k * 11) % 26 + 2
        for dx in range(4):
            cv.put(x0 + dx, y + (dx in (1, 2)) * -1, "water.l")


ANIMATED = {"desert": desert_anim, "shore": shore_anim, "snow": snow_anim, "swamp": swamp_anim,
            "water": water_anim}


# -- fog and state tiles ---------------------------------------------------------------------------

def fog():
    """Unexplored edge: soft dark mist, dithered, thickest toward the middle."""
    cv = Canvas(T, T)
    for y in range(T):
        for x in range(T):
            d = math.hypot((x - 15.5) / 15.5, (y - 15.5) / 15.5)
            level = max(0.0, 1.0 - d) * 1.5
            n = noise(x // 2, y // 2, 21) * 0.25
            if dither(x, y, level + n - 0.1):
                cv.put(x, y, "charcoal.m" if level > 0.9 else "charcoal.d")
            elif dither(x, y, (level + n) * 0.5):
                cv.put(x, y, "stone.d")
    return cv


def unknown():
    t = Tile("stone.d", 5)
    t.scatter("charcoal.m", 0.1)
    t.scatter("stone.m", 0.05, 1)
    return t.cv


def night_mask():
    """A vignette the client multiplies for dark rooms: dithered charcoal,
    clear at the middle, solid at the corners (hard alpha, palette colors)."""
    cv = Canvas(T, T)
    for y in range(T):
        for x in range(T):
            d = math.hypot((x - 15.5) / 15.5, (y - 15.5) / 15.5)
            if dither(x, y, (d - 0.35) * 1.2):
                cv.put(x, y, "charcoal.d")
    return cv
