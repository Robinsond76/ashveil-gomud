"""Style frames (S0) and the app emblem.

The two style frames are mock scenes that show the art direction in one
picture: a rich-but-natural map and a battle with a company facing a forest
ogre.  Their terrain, ogre and goblin pieces are throwaway mocks; sets S2 and
S3 replace them with the real tiles and enemy sprites.
"""
import math

from pixels import Canvas, rect, round_rect, line, thick, ellipse
import figures
import icons


def noise(x, y, seed=0):
    n = (x * 374761393 + y * 668265263 + seed * 2147483647) & 0xFFFFFFFF
    n = ((n ^ (n >> 13)) * 1274126177) & 0xFFFFFFFF
    return ((n ^ (n >> 16)) & 0xFFFF) / 65535.0


# -- mock map tiles (32x32, no outline) ----------------------------------------

def tile_grass(seed=0):
    cv = Canvas(32, 32, "grass.m")
    for y in range(32):
        for x in range(32):
            n = noise(x, y, seed)
            if n > 0.93:
                cv.put(x, y, "grass.l")
            elif n < 0.07:
                cv.put(x, y, "grass.d")
    for i in range(4):
        x, y = int(noise(i, 1, seed) * 28) + 2, int(noise(i, 2, seed) * 28) + 2
        cv.put(x, y, "grass.d")
        cv.put(x, y - 1, "grass.l")
    if seed % 3 == 0:
        x, y = int(noise(5, 5, seed) * 26) + 3, int(noise(6, 6, seed) * 26) + 3
        cv.put(x, y, "bone.l")
    return cv


def _tree(cv, cx, cy, r):
    cv.fill(rect(cx - 1, cy + r - 1, cx, cy + r + 2), "leather.d")
    cv.part(ellipse(cx, cy, r, r - 1), "forest")
    cv.fill(ellipse(cx - 2, cy - 2, r * 0.4, r * 0.35), "moss.m")
    cv.fill({(cx - 3, cy - 3), (cx - 2, cy - 3)}, "moss.l")


def tile_forest(seed=0):
    cv = tile_grass(seed + 11)
    cv.fill(ellipse(16, 16, 15, 14), "forest.d")
    for dx, dy, r in ((-8, -6, 7), (8, -4, 7), (-5, 7, 7), (9, 8, 6), (0, 0, 6)):
        _tree(cv, 16 + dx, 16 + dy, r)
    return cv


def tile_road(seed=0):
    cv = tile_grass(seed + 3)
    for y in range(9, 24):
        for x in range(32):
            n = noise(x, y, seed)
            cv.put(x, y, "ochre.d" if n < 0.1 else ("ochre.l" if n > 0.9 else "ochre.m"))
    for x in range(32):
        cv.put(x, 8 + (noise(x, 0, seed) > 0.6), "grass.d")
        cv.put(x, 24 - (noise(x, 1, seed) > 0.6), "ochre.d")
    for x in range(0, 32):
        if x % 7 != 3:
            cv.put(x, 14, "ochre.d")
            cv.put(x, 19, "ochre.d")
    return cv


def tile_river(seed=0, bridge=False):
    cv = tile_grass(seed + 5)
    for y in range(32):
        left = 9 + int(2 * math.sin(y / 5 + seed))
        right = 23 + int(2 * math.sin(y / 5 + seed))
        for x in range(left, right + 1):
            cv.put(x, y, "water.m")
        cv.put(left - 1, y, "ochre.m")
        cv.put(right + 1, y, "ochre.d")
        cv.put(left, y, "water.d")
        for x in range(left + 1, right):
            if noise(x, y // 3, seed) > 0.86 and (x + y) % 2 == 0:
                cv.put(x, y, "water.l")
    if bridge:
        for y in range(7, 25):
            for x in range(6, 26):
                cv.put(x, y, "wood.m" if (x // 3) % 2 else "wood.l")
        for x in range(6, 26):
            cv.put(x, 7, "wood.d")
            cv.put(x, 24, "wood.d")
    return cv


def tile_village(seed=0):
    cv = tile_grass(seed + 1)
    for x0, y0, w in ((3, 4, 12), (16, 12, 13)):
        cv.part(rect(x0, y0 + 6, x0 + w - 1, y0 + 14), "stone")
        roof = {(x, y) for y in range(y0, y0 + 7)
                for x in range(x0 - 1 + (y0 + 6 - y) // 2, x0 + w - (y0 + 6 - y) // 2)}
        cv.part(roof, "ochre")
        cv.fill({(x0 + w // 2, y0 + 12), (x0 + w // 2 - 1, y0 + 12), (x0 + w // 2, y0 + 13),
                 (x0 + w // 2 - 1, y0 + 13), (x0 + w // 2, y0 + 14), (x0 + w // 2 - 1, y0 + 14)},
                "leather.d")
        cv.fill({(x0 + 2, y0 + 9)}, "brass.l")
        cv.fill({(x0 + w - 3, y0 + 9)}, "brass.l")
    cv.fill(line(2, 26, 12, 26), "leather.m")
    return cv


def mock_map_scene(W=320, H=180):
    layout = [
        "fffgggwgff",
        "ffgggvwggf",
        "ggggggwggg",
        "gggcggwggf",
        "rrrrrrBrrr",
        "ggfgggwgfg",
    ]
    cv = Canvas(W, H)
    for r, row in enumerate(layout):
        for c, ch in enumerate(row):
            seed = r * 10 + c
            t = {"g": tile_grass, "f": tile_forest, "r": tile_road, "w": tile_river,
                 "v": tile_village}.get(ch)
            if ch == "B":
                tl = tile_road(seed)
                br = tile_river(seed, bridge=True)
                tl.blit(br, 0, 0)
                tile = tl
            elif ch == "c":
                tile = tile_grass(seed)
                tile.blit(icons.tent(), 0, 0)
            else:
                tile = t(seed)
            cv.blit(tile, c * 32, r * 32)
    # Resource icons in tile corners.
    cv.blit(icons.res_water(), 6 * 32 + 14, 2 * 32 + 1)
    cv.blit(icons.res_forage(), 2 * 32 + 15, 0 * 32 + 15)
    cv.blit(icons.here_ring_frames()[1], 3 * 32 + 0, 4 * 32 + 0)
    # Units: the warrior on the road, the witch beside the camp.
    cv.blit(figures.render("warrior", "side", figures.MAP, figures.STAND), 3 * 32, 4 * 32 - 4)
    cv.blit(figures.render("witch", "down", figures.MAP, figures.STAND), 3 * 32 + 14, 3 * 32 - 4)
    cv.blit(icons.company_badge(), 3 * 32 + 20, 4 * 32 - 2)
    cv.blit(icons.walk_dot(), 5 * 32 - 6, 4 * 32 + 12)
    return cv


# -- mock battle pieces ----------------------------------------------------------

def mock_ogre():
    """A forest ogre, 96x96, facing left."""
    cv = Canvas(96, 96)
    # legs
    cv.part(rect(36, 62, 47, 91), "ashmoss")
    cv.part(rect(52, 62, 63, 91), "ashmoss")
    cv.part(rect(33, 88, 47, 92), "leather")
    cv.part(rect(52, 88, 66, 92), "leather")
    # hunched torso, belly and loincloth
    cv.part(ellipse(48, 46, 21, 21), "ashmoss")
    cv.part(ellipse(44, 54, 13, 12), "ashmoss", flat="m")
    cv.part(rect(34, 62, 62, 74), "leather")
    cv.fill({(x, 62) for x in range(34, 63)}, "leather.l")
    for x in range(35, 62, 4):
        cv.put(x, 75, "leather.d")
        cv.put(x, 76, "leather.d")
    # far arm and club, near arm with a stone-headed club held across the body
    cv.part(thick(line(66, 38, 80, 62), 8), "ashmoss")
    cv.part(ellipse(80, 64, 5, 5), "ashmoss")
    cv.part(thick(line(80, 66, 60, 16), 6), "wood")
    cv.part(ellipse(58, 18, 8, 10), "wood")
    for x, y in ((54, 14), (60, 12), (58, 22), (53, 20), (62, 18)):
        cv.put(x, y, "iron.l")
    # near arm
    cv.part(thick(line(30, 34, 22, 62), 9), "ashmoss")
    cv.part(ellipse(20, 64, 6, 6), "ashmoss")
    # head: low and forward, heavy brow, tusks
    cv.part(ellipse(30, 28, 12, 10), "ashmoss")
    cv.fill({(x, 25) for x in range(22, 31)}, "ashmoss.d")
    cv.put(24, 26, "outline")
    cv.put(25, 26, "outline")
    cv.put(24, 27, "ember.d")
    cv.part({(21, 34), (21, 35), (21, 36), (22, 33), (22, 34)}, "bone")
    cv.part({(28, 36), (28, 37), (28, 35)}, "bone")
    cv.fill({(x, 33) for x in range(22, 30)}, "oxblood.d")
    cv.outline()
    return cv


def mock_goblin():
    """A gaunt goblin, 48x48 frame, figure about 28 px, facing left."""
    cv = Canvas(48, 48)
    cv.part(rect(20, 32, 23, 42), "ashmoss")
    cv.part(rect(26, 32, 29, 42), "ashmoss")
    cv.part(rect(18, 41, 23, 43), "leather")
    cv.part(rect(26, 41, 31, 43), "leather")
    cv.part(rect(19, 22, 30, 33), "leather")  # ragged tunic
    cv.fill({(x, 33) for x in range(19, 31, 2)}, "leather.d")
    cv.part(rect(15, 24, 18, 31), "moss")
    cv.part(rect(31, 23, 34, 30), "moss")
    cv.fill({(14, 30), (15, 30), (31, 29), (32, 29)}, "moss.l")
    # hooked blade
    cv.part(thick(line(14, 29, 8, 22), 1), "steel", flat="m")
    cv.fill({(7, 21), (8, 21)}, "steel.l")
    # head, big ears, hooked nose
    cv.part(round_rect(18, 14, 27, 22), "moss")
    cv.part({(17, 16), (16, 15), (15, 14), (17, 15), (28, 16), (29, 15), (30, 14), (28, 15)}, "moss")
    cv.put(20, 17, "ember.m")
    cv.put(24, 17, "ember.m")
    cv.part({(17, 19), (16, 20), (17, 20)}, "moss", flat="d")
    cv.fill({(x, 21) for x in range(20, 26)}, "bone.m")
    cv.outline()
    return cv


# -- mock battle scene ------------------------------------------------------------

def _bg_forest(cv, W, H):
    horizon = 92
    for y in range(horizon):
        for x in range(W):
            t = y / horizon + 0.04 * noise(x // 2, y // 2, 1)
            cv.put(x, y, "slate.m" if t < 0.18 else "slate.l" if t < 0.50 else
                   ("wool.d" if (x + y) % 2 and t < 0.62 else "slate.l") if t < 0.62 else
                   ("ashmoss.l" if t < 0.80 else "wool.d"))
    # Far tree line, then nearer pines in two layers.
    for layer, (base, col, top) in enumerate((("ashmoss.m", "ashmoss.m", 60), ("moss.d", "moss.d", 48))):
        for x in range(W):
            h = top + 10 * noise(x // 4, layer, 4) + 6 * math.sin(x / 9 + layer)
            for y in range(int(h), horizon + 6):
                cv.put(x, y, col)
    for i in range(14):
        x = 8 + i * 24 + int(noise(i, 9, 2) * 10)
        top = 20 + int(noise(i, 8, 2) * 22)
        for y in range(top, horizon + 4):
            wid = (y - top) // 9
            for xx in range(x - wid, x + wid + 1):
                cv.put(xx, y, "forest.m" if xx <= x else "forest.d")
        cv.fill({(x, y) for y in range(horizon, horizon + 6)}, "leather.d")
    # Ground, a trodden dirt track and grass tufts.
    for y in range(horizon, H):
        for x in range(W):
            n = noise(x, y, 7)
            cv.put(x, y, "grass.d" if y < horizon + 6 else
                   ("grass.m" if n < 0.9 else "grass.l"))
    for y in range(120, H):
        reach = 20 + (y - 120) * 1.6
        for x in range(int(W / 2 - reach * 2.2), int(W / 2 + reach * 2.2)):
            n = noise(x, y, 3)
            cv.put(x, y, "leather.m" if n < 0.88 else "leather.l")
            if n < 0.06:
                cv.put(x, y, "leather.d")


def mock_battle_scene(W=320, H=180):
    cv = Canvas(W, H)
    _bg_forest(cv, W, H)

    def shadow(cx, cy, rx):
        cv.fill(ellipse(cx, cy, rx, max(2, rx // 4)), "grass.d")

    # Company on the left, in formation.
    placements = (("wizard", 54, 126), ("cleric", 90, 142), ("warrior", 128, 160))
    for cls, fx, fy in placements:
        shadow(fx, fy, 13)
        cv.blit(figures.battle_idle(cls), fx - 32, fy - 61)
    # Enemies on the right (mirrored to face left).
    ogre = mock_ogre()
    shadow(244, 158, 30)
    cv.blit(ogre, 244 - 48, 158 - 92)
    for fx, fy in ((190, 128), (302, 136)):
        g = mock_goblin()
        shadow(fx, fy, 11)
        cv.blit(g, fx - 24, fy - 44)
    return cv


# -- app emblem: an ember-lit campfire before a dark ridge -----------------------

def emblem(S, e=1.0, detail=True):
    """Pixel emblem on an S x S grid.  e < 1 shrinks the picture about the
    centre while the sky and ground bleed to the edges (maskable icons)."""
    cv = Canvas(S, S)
    for y in range(S):
        for x in range(S):
            u = ((x + 0.5) / S - 0.5) / e + 0.5
            v = ((y + 0.5) / S - 0.5) / e + 0.5
            cv.put(x, y, _emblem_px(u, v, x, y, S, detail))
    return cv


def _emblem_px(u, v, x, y, S, detail):
    checker = (x + y) % 2 == 0
    ridge = 0.50 + 0.07 * math.sin(u * 7.0) + 0.05 * math.sin(u * 17.0 + 1.0) \
        - 0.12 * math.exp(-((u - 0.78) ** 2) / 0.01)
    near = 0.64 + 0.04 * math.sin(u * 9.0 + 2.0)
    # fire geometry
    fu, fv = u - 0.5, v - 0.80
    flame_h = 0.20
    in_flame = -flame_h < fv < 0.0 and abs(fu) < 0.085 * (1 + fv / flame_h * 0.9 + 0.1) * (1.3 if fv > -0.06 else 1)
    core = -0.11 < fv < 0.0 and abs(fu) < 0.04 * (1 + fv / 0.11)
    tip = -0.20 < fv < -0.12 and abs(fu - 0.01) < 0.03 * (1 + (fv + 0.12) / 0.08)
    logs = 0.0 <= fv < 0.045 and abs(fu) < 0.17 and (abs(fu) + fv * 1.5) > 0.03
    stones = 0.03 < fv < 0.075 and 0.15 < abs(fu) < 0.24
    dist = math.hypot(fu * 1.0, (fv + 0.05) * 1.2)
    if logs:
        return "wood.m" if (int((fu + 1) * 40) % 2 == 0) else "wood.d"
    if stones:
        return "stone.m"
    if core:
        return "ember.l"
    if in_flame:
        return "ember.m"
    if tip:
        return "ember.m"
    if detail and v < 0.30 and (x * 7 + y * 13) % 29 == 0:
        return "bone.l"
    if v > near:  # foreground ground, lit near the fire
        if dist < 0.17:
            return "ochre.d"
        if dist < 0.24 and checker:
            return "ochre.d"
        return "forest.d"
    if v > ridge:  # the dark ridge
        return "charcoal.d"
    # sky: dusk bands, with a faint ember wash low down
    if v < 0.22:
        return "charcoal.m"
    if v < 0.34:
        return "plum.d"
    if v < ridge - 0.04:
        return "heather.d" if v < 0.44 else "plum.m"
    return "oxblood.d"
