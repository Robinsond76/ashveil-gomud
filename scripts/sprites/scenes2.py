"""Style frames in the pre-rendered look: a battle in a forest and a map."""
import math
import random
import numpy as np
import render3d as r3
import figures as fg
from render3d import Part, make_ramp, rgb, BAYER

# Scene ramps, built the same way as the material ramps.
SCENE = {
    "canopy": make_ramp("3c5a33", 10, 0.85, 0.45),
    "farwood": make_ramp("40564d", 8, 0.8, 0.35),
    "grass": make_ramp("55743a", 12, 0.8, 0.5),
    "earth": make_ramp("6b4c30", 8, 0.75, 0.45),
    "bark": make_ramp("54402e", 9, 0.85, 0.4),
    "sky": make_ramp("8fa9a6", 6, 0.4, 0.5),
    "water": make_ramp("33566c", 10, 0.85, 0.55),
    "road": make_ramp("8a6a45", 8, 0.7, 0.45),
    "thatch": make_ramp("9a7a3e", 8, 0.75, 0.45),
    "stone": make_ramp("6d6a62", 8, 0.75, 0.5),
    "night": make_ramp("1d2433", 4, 0.6, 0.3),
}


def paint(img, x, y, ramp, level, dither=1.0):
    """Paint a continuous ramp level with ordered dithering."""
    if not (0 <= y < img.shape[0] and 0 <= x < img.shape[1]):
        return
    rp = SCENE[ramp] if isinstance(ramp, str) else ramp
    lv = max(0.0, min(len(rp) - 1.0, level))
    b = math.floor(lv)
    idx = b + (1 if (lv - b) > 0.5 + (BAYER[y % 4, x % 4] - 0.5) * dither else 0)
    idx = min(idx, len(rp) - 1)
    img[y, x, :3] = rgb(rp[idx])
    img[y, x, 3] = 255


def noise2(x, y, seed=0):
    """Smooth value noise in about -1..1 (lattice 8 px)."""
    def h(i, j):
        v = math.sin(i * 127.1 + j * 311.7 + seed * 74.7) * 43758.5453
        return (v - math.floor(v)) * 2 - 1
    gx, gy = x / 8.0, y / 8.0
    i, j = math.floor(gx), math.floor(gy)
    fx, fy = gx - i, gy - j
    fx, fy = fx * fx * (3 - 2 * fx), fy * fy * (3 - 2 * fy)
    a = h(i, j) + (h(i + 1, j) - h(i, j)) * fx
    b = h(i, j + 1) + (h(i + 1, j + 1) - h(i, j + 1)) * fx
    return a + (b - a) * fy


def paste(dst, src, ox, oy):
    h, w = src.shape[:2]
    for y in range(h):
        Y = oy + y
        if not 0 <= Y < dst.shape[0]:
            continue
        for x in range(w):
            X = ox + x
            if src[y, x, 3] and 0 <= X < dst.shape[1]:
                dst[Y, X] = src[y, x]


def shadow(img, cx, cy, rx, ry, amount=0.8):
    """Darken an ellipse in place (unit shadows), staying on scene ramps."""
    lookup = {}
    for name, rp in SCENE.items():
        for i, c in enumerate(rp):
            lookup[rgb(c)] = (name, i)
    for y in range(int(cy - ry), int(cy + ry) + 1):
        for x in range(int(cx - rx), int(cx + rx) + 1):
            if 0 <= y < img.shape[0] and 0 <= x < img.shape[1]:
                d = ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2
                if d < 1:
                    key = lookup.get(tuple(int(v) for v in img[y, x, :3]))
                    if key:
                        name, i = key
                        drop = amount * (2.2 if d < 0.5 else 1.2)
                        paint(img, x, y, name, i - drop)


# --- Battle -----------------------------------------------------------------------
W, H = 384, 216
HORIZON = 112


def forest_battle_background(seed=7):
    rnd = random.Random(seed)
    img = np.zeros((H, W, 4), np.uint8)
    # pale sky glimpsed through the canopy, deepening toward the forest floor
    for y in range(H):
        for x in range(W):
            n = noise2(x, y, 1.0)
            if y < HORIZON:
                t = y / HORIZON
                paint(img, x, y, "farwood", 5.2 - t * 3.6 + n * 0.6)
    # misty far trunks, fading with distance
    for layer, (count, shade, wmin, wmax) in enumerate(((26, 4.2, 2, 4), (16, 2.6, 4, 7))):
        for _ in range(count):
            x0, w = rnd.randint(-10, W), rnd.randint(wmin, wmax)
            top = rnd.randint(0, 30)
            for y in range(top, HORIZON + 2):
                for dx in range(w):
                    lv = shade - 0.8 * (dx / max(w - 1, 1)) + (0.6 if dx == 0 else 0)
                    paint(img, x0 + dx, y, "farwood", lv - layer * 0.4)
    # canopy masses, lit from the top-left
    for _ in range(140):
        cx, cy, rad = rnd.randint(-20, W + 20), rnd.randint(-25, 52), rnd.randint(10, 26)
        for y in range(max(0, cy - rad), min(H, cy + rad)):
            for x in range(max(0, cx - rad), min(W, cx + rad)):
                dx, dy = x - cx, y - cy
                d = math.hypot(dx, dy) / rad
                if d < 1 - 0.18 * (noise2(x * 3, y * 3, cx) + 1) * 0.5:
                    lit = -(dx + dy) / (2 * rad) + 0.5
                    paint(img, x, y, "canopy", 1.0 + lit * 5.0 - d * 1.2 + (y / 52) * -0.6)
    # sky gaps
    for _ in range(26):
        x, y = rnd.randint(0, W), rnd.randint(0, 38)
        for i in range(rnd.randint(1, 4)):
            paint(img, x + i, y, "sky", 3.5 + rnd.random())
    # forest floor: perspective-scaled grass, darker far, brighter near
    for y in range(HORIZON - 4, H):
        t = (y - (HORIZON - 4)) / (H - HORIZON + 4)
        for x in range(W):
            n = noise2(x * 1.7, y * 2.2, 3.0) * 0.6 + noise2(x * 0.45, y * 0.6, 8.0) * 0.6
            lv = 2.0 + t * 5.4 + n
            paint(img, x, y, "grass", lv)
    # bare earth patches and a worn track across the clearing
    for _ in range(18):
        cx, cy = rnd.randint(30, W - 30), rnd.randint(HORIZON + 10, H - 10)
        t = (cy - HORIZON) / (H - HORIZON)
        rx, ry = rnd.randint(8, 22) * (0.5 + t), rnd.randint(2, 5) * (0.5 + t)
        for y in range(int(cy - ry), int(cy + ry) + 1):
            for x in range(int(cx - rx), int(cx + rx) + 1):
                d = ((x - cx) / rx) ** 2 + ((y - cy) / ry) ** 2
                if d < 1 - 0.3 * (noise2(x * 2, y * 4, cx) + 1) * 0.5:
                    paint(img, x, y, "earth", 2.4 + t * 3 + (1 - d) * 0.8 - (y - cy) / ry * 0.3)
    # grass tufts, scaled by depth
    for _ in range(420):
        x, y = rnd.randint(0, W), rnd.randint(HORIZON + 2, H - 1)
        t = (y - HORIZON) / (H - HORIZON)
        hgt = int(1 + t * 4)
        for i in range(hgt):
            paint(img, x + (i % 2), y - i, "grass", 3.5 + t * 6 - i * 0.2)
    # near trunks standing at the far edge of the clearing, rendered as cylinders
    trunk_parts = []
    for x, rad in ((50, 6.0), (122, 4.6), (190, 7.0), (262, 5.0), (330, 6.4)):
        x += rnd.randint(-6, 6)
        trunk_parts.append((x, rad))
    for x, rad in trunk_parts:
        parts = [Part("cap", "wood", a=(0, -2, 0), b=(0, 140, 0), r1=rad, r2=rad * 0.8),
                 Part("cap", "wood", blend=2.0, a=(0, 0, 0), b=(-rad * 1.6, -1, 1.5), r1=rad * 0.45, r2=0.6),
                 Part("cap", "wood", blend=2.0, a=(0, 0, 0), b=(rad * 1.7, -1, 0.5), r1=rad * 0.45, r2=0.6)]
        spr = r3.render(parts, int(rad * 6), HORIZON + 8, r3.Camera(0, 8), (int(rad * 3), HORIZON + 4),
                        ambient=0.26, dither=0.5)
        paste(img, spr, int(x - rad * 3), 0)
    # light shafts from the top-left, lifting what they cross
    for sx, wid in ((40, 10), (150, 16), (236, 9), (300, 12)):
        for y in range(0, H):
            cx = sx + y * 0.55
            for x in range(int(cx - wid), int(cx + wid)):
                if 0 <= x < W and BAYER[y % 4, x % 4] < 0.3 * (1 - abs(x - cx) / wid):
                    lift_pixel(img, x, y)
    # framing trunks and ferns at the edges
    for x0, rad, flip in ((-4, 13, False), (W - 20, 14, True)):
        parts = [Part("cap", "wood", a=(0, -10, 0), b=(0, 260, 0), r1=rad, r2=rad * 0.9)]
        spr = r3.render(parts, int(rad * 3), H, r3.Camera(0, 0), (int(rad * 1.5), H + 2), ambient=0.12,
                        light=(-0.6, 0.6, 0.5), dither=0.5)
        paste(img, spr, x0, 0)
    for _ in range(40):
        side = rnd.choice((0, 1))
        bx = rnd.randint(0, 34) if side == 0 else rnd.randint(W - 34, W)
        fern(img, bx, H - rnd.randint(0, 4), rnd.randint(10, 20), rnd)
    return img


_LIFT = None


def lift_pixel(img, x, y):
    global _LIFT
    if _LIFT is None:
        _LIFT = {}
        for name, rp in list(SCENE.items()) + [(k, v) for k, v in r3.RAMPS.items()]:
            for i, c in enumerate(rp):
                _LIFT[rgb(c)] = rgb(rp[min(i + 1, len(rp) - 1)])
    key = tuple(int(v) for v in img[y, x, :3])
    if key in _LIFT:
        img[y, x, :3] = _LIFT[key]


def fern(img, x, y, size, rnd):
    ang = -math.pi / 2 + rnd.uniform(-0.9, 0.9)
    for i in range(size):
        px = int(x + math.cos(ang) * i)
        py = int(y + math.sin(ang) * i + (i * i) / (size * 3))
        paint(img, px, py, "grass", 2 + i * 0.3)
        if i % 2 == 0 and i > 2:
            for s in (-1, 1):
                for k in range(1, max(2, (size - i) // 3)):
                    paint(img, px + s * k, py + k // 2, "grass", 1.5 + i * 0.25)


def battle_unit(parts, facing_left, size=80, anchor=None, bulk=1.25):
    yaw = 130 if facing_left else 50
    anchor = anchor or (size // 2, size - 2)
    return r3.render(parts, size, size, r3.Camera(yaw, 16, bulk=bulk), anchor, ambient=0.3)


def style_battle():
    img = forest_battle_background()
    units = [
        # (sprite, feet x, feet y, shadow rx)
        (battle_unit(fg.wizard(), False), 52, 150, 11),
        (battle_unit(fg.warrior(), False), 126, 170, 12),
        (battle_unit(fg.cleric(), False), 78, 196, 11),
        (battle_unit(fg.goblin(), True, 64), 318, 146, 8),
        (battle_unit(fg.ogre(), True, 128, (64, 125), bulk=1.1), 252, 186, 24),
        (battle_unit(fg.goblin(), True, 64), 336, 202, 8),
    ]
    for spr, fx, fy, rx in units:
        shadow(img, fx + 2, fy - 1, rx, rx * 0.28)
    for spr, fx, fy, rx in sorted(units, key=lambda u: u[2]):
        h, w = spr.shape[:2]
        paste(img, spr, fx - w // 2, fy - (h - 2))
    return img


# --- Map ----------------------------------------------------------------------------
MAP = [
    "FFFFLLWLLL",
    "FFFLLLWLVV",
    "FFLRRRBRVV",
    "FLLRLLWLLL",
    "LLCRLWWLFF",
    "LLLRLWLFFF",
]
MW, MH = 320, 180


def map_cell(r, c):
    return MAP[r][c] if 0 <= r < len(MAP) and 0 <= c < len(MAP[0]) else None


def terrain_at(x, y):
    return map_cell(y // 32, x // 32)


def style_map():
    rnd = random.Random(11)
    img = np.zeros((MH, MW, 4), np.uint8)
    # ground everywhere: meadow with soft variation, darker under woods
    forest = np.zeros((MH, MW))
    for y in range(MH):
        for x in range(MW):
            forest[y, x] = 1.0 if terrain_at(x, y) == "F" else 0.0
    k = 7
    pad = np.pad(forest, k, mode="edge")
    cs = pad.cumsum(0).cumsum(1)
    cs = np.pad(cs, ((1, 0), (1, 0)))
    n_ = 2 * k + 1
    soft = (cs[n_:, n_:] - cs[:-n_, n_:] - cs[n_:, :-n_] + cs[:-n_, :-n_]) / (n_ * n_)
    for y in range(MH):
        for x in range(MW):
            n = noise2(x * 1.5, y * 1.5, 5.0) * 0.8 + noise2(x * 4, y * 4, 2.0) * 0.35
            paint(img, x, y, "grass", 6.0 - soft[y, x] * 3.4 + n)
    # river with soft banks: distance field to water tiles
    water = np.zeros((MH, MW), bool)
    for y in range(MH):
        for x in range(MW):
            water[y, x] = terrain_at(x, y) in ("W", "B")
    # shrink the water a little and round its corners
    dist = np.full((MH, MW), 99.0)
    ys, xs = np.where(~water)
    land_pts = np.stack([xs, ys], 1)
    for y in range(MH):
        for x in range(MW):
            if water[y, x]:
                near = land_pts[(np.abs(land_pts[:, 0] - x) < 8) & (np.abs(land_pts[:, 1] - y) < 8)]
                dist[y, x] = np.min(np.hypot(near[:, 0] - x, near[:, 1] - y)) if len(near) else 8
    for y in range(MH):
        for x in range(MW):
            if water[y, x]:
                d = dist[y, x] + noise2(x * 4, y * 4, 2) * 0.8
                if d < 1.6:
                    paint(img, x, y, "earth", 3.6)
                elif d < 2.6:
                    paint(img, x, y, "earth", 2.4)
                else:
                    flow = math.sin(y * 0.45 + x * 0.2) * 0.6 + noise2(x * 2, y * 3, 9) * 0.5
                    paint(img, x, y, "water", 2.0 + min(d, 9) * 0.25 + flow)
    # reeds along the banks
    for _ in range(70):
        x, y = rnd.randint(0, MW - 1), rnd.randint(0, MH - 1)
        if water[y, x] and 1.5 < dist[y, x] < 3.2:
            for i in range(3):
                paint(img, x, y - i, "grass", 4 + i)
    # roads: a soft-edged track along road tiles, joined to neighbours
    def road_cells():
        for r_, row in enumerate(MAP):
            for c_, ch in enumerate(row):
                if ch in ("R", "B"):
                    yield r_, c_
    segs = []
    for r_, c_ in road_cells():
        cx, cy = c_ * 32 + 16, r_ * 32 + 16
        for dr, dc in ((0, 1), (1, 0), (0, -1), (-1, 0)):
            nb = map_cell(r_ + dr, c_ + dc)
            if nb in ("R", "B", "V", "C") or (nb is None and dr == 1):
                segs.append(((cx, cy), (cx + dc * 18, cy + dr * 18)))
    for y in range(MH):
        for x in range(MW):
            if water[y, x]:
                continue
            best = 99
            for (ax, ay), (bx, by) in segs:
                vx, vy = bx - ax, by - ay
                t = max(0, min(1, ((x - ax) * vx + (y - ay) * vy) / (vx * vx + vy * vy)))
                best = min(best, math.hypot(x - ax - vx * t, y - ay - vy * t))
            wob = noise2(x * 3, y * 3, 4) * 0.9
            if best < 4.6 + wob:
                rut = 1.2 if abs(best - 2.6) < 0.5 else 0
                paint(img, x, y, "road", 4.6 - rut - best * 0.15 + noise2(x * 5, y * 5, 1) * 0.6)
            elif best < 6.0 + wob:
                paint(img, x, y, "grass", 4.4)
    # bridge over the river
    br, bc = next((r_, c_) for r_, c_ in road_cells() if MAP[r_][c_] == "B")
    bx0, by0 = bc * 32, br * 32
    for y in range(by0 + 10, by0 + 22):
        for x in range(bx0 - 2, bx0 + 34):
            plank = (x - bx0) % 4 == 0
            edge = y in (by0 + 10, by0 + 21)
            paint(img, x, y, "bark", 2.0 if edge or plank else 5.0 - (y - by0 - 11) * 0.12)
    # 3D overlays: trees, cottages, camp
    objs = []
    for r_, row in enumerate(MAP):
        for c_, ch in enumerate(row):
            ox, oy = c_ * 32, r_ * 32
            if ch == "F":
                for (tx, ty) in ((8, 8), (24, 7), (16, 18), (6, 27), (26, 25)):
                    tx += rnd.randint(-3, 3)
                    ty += rnd.randint(-3, 3)
                    objs.append(("tree", ox + tx, oy + ty, rnd.uniform(6.0, 8.5)))
            elif ch == "V":
                spots = [(9, 10), (24, 26)] if (r_ + c_) % 2 else [(23, 9), (9, 27)]
                for tx, ty in spots:
                    objs.append(("cottage", ox + tx, oy + ty, rnd.choice((0, 1))))
            elif ch == "C":
                objs.append(("tent", ox + 11, oy + 19, 0))
                objs.append(("fire", ox + 23, oy + 24, 0))
    # units and resource icons
    objs.append(("unit", 4 * 32 + 16, 2 * 32 + 17, "warrior"))
    objs.append(("unit", 3 * 32 + 10, 4 * 32 + 22, "witch"))
    for kind, x, y, arg in sorted(objs, key=lambda o: o[2]):
        if kind == "tree":
            shadow_map(img, x + 3, y + 3, arg * 0.9, arg * 0.6)
            spr = tree_sprite(arg)
            paste(img, spr, x - spr.shape[1] // 2, y - spr.shape[0] + int(arg) + 2)
        elif kind == "cottage":
            shadow_map(img, x + 3, y + 1, 9, 4)
            spr = cottage_sprite(arg)
            paste(img, spr, x - spr.shape[1] // 2, y - spr.shape[0] + 4)
        elif kind == "tent":
            shadow_map(img, x + 2, y, 9, 3.5)
            spr = tent_sprite()
            paste(img, spr, x - spr.shape[1] // 2, y - spr.shape[0] + 4)
        elif kind == "fire":
            spr = fire_sprite()
            paste(img, spr, x - spr.shape[1] // 2, y - spr.shape[0] + 3)
        elif kind == "unit":
            shadow_map(img, x + 1, y - 1, 6, 2)
            spr = map_unit(arg)
            paste(img, spr, x - 16, y - 30)
    import icons2
    paste(img, icons2.water(), 7 * 32 + 2, 3 * 32 + 2)
    paste(img, icons2.forage(), 1 * 32 + 14, 3 * 32 + 3)
    return img


def shadow_map(img, cx, cy, rx, ry):
    shadow(img, cx, cy, rx, ry, amount=0.9)


MAP_CAM = dict(pitch=48)


def tree_sprite(rad):
    parts = [Part("cap", "wood", a=(0, 0, 0), b=(0, rad * 0.9, 0), r1=1.2, r2=1.0),
             Part("ellip", "moss", blend=2.0, c=(0, rad * 1.5, 0), r=(rad, rad * 0.9, rad)),
             Part("ellip", "moss", blend=2.0, c=(-rad * 0.35, rad * 1.9, rad * 0.2), r=(rad * 0.7,) * 3)]
    size = int(rad * 3) + 4
    return r3.render(parts, size, size, r3.Camera(20, MAP_CAM["pitch"]), (size // 2, size - int(rad) + 1),
                     ambient=0.35, dither=0.6)


def prism(mat, c, half_len, half_w, inner=None, yaw_axis="x"):
    """A triangular prism (gable roof or tent), ridge along x, sitting on c's y."""
    q = 0.70710678
    R45 = np.array([[1.0, 0, 0], [0, q, -q], [0, q, q]])  # 45 degrees about x; ridge stays on x
    cx, cy, cz = c
    kw = {}
    if inner is not None:
        kw["minus"] = Part("box", mat, c=(cx, cy - inner[1], cz), R=R45, b=(half_len + 2, inner[0], inner[0]))
    return Part("box", mat, c=c, R=R45, b=(half_len, half_w, half_w), rr=0.15, clip=[((0, 1, 0), cy)], **kw)


def cottage_sprite(variant):
    parts = [Part("box", "plaster", c=(0, 3.0, 0), b=(5.0, 3.0, 3.4), rr=0.15),
             prism("plaster", (0, 6.0, 0), 4.95, 2.6),
             # timber framing on the long wall
             Part("box", "wood", c=(0, 0.3, 3.45), b=(5.05, 0.3, 0.08)),
             Part("box", "wood", c=(0, 5.8, 3.45), b=(5.05, 0.3, 0.08)),
             Part("box", "wood", c=(-4.9, 3.0, 3.45), b=(0.3, 3.0, 0.08)),
             Part("box", "wood", c=(4.9, 3.0, 3.45), b=(0.3, 3.0, 0.08)),
             Part("box", "wood", c=(1.6, 1.7, 3.5), b=(0.9, 1.7, 0.1)),
             Part("box", "yellow_eye", c=(-2.4, 3.4, 3.5), b=(0.7, 0.7, 0.1)),
             Part("box", "wood", c=(5.05, 3.6, -0.2), b=(0.08, 0.8, 0.7)),
             prism("thatch_m", (0, 5.6, 0), 5.8, 3.6, inner=(3.0, 0.4)),
             Part("box", "stone_m", c=(-3.2, 9.8, -1.0), b=(0.9, 2.4, 0.9), rr=0.1)]
    return r3.render(parts, 28, 28, r3.Camera(35 + variant * 25, MAP_CAM["pitch"], bulk=1.15), (14, 22),
                     ambient=0.34, dither=0.4)


def tent_sprite():
    parts = [prism("canvas", (0, 0, 0), 6.2, 5.0, inner=(4.3, 0.6)),
             Part("box", "eye", c=(0, 0.2, 0), b=(5.8, 0.2, 2.8)),
             Part("cap", "oxblood", a=(7.6, 0.8, 3.6), b=(7.6, 0.8, -1.2), r1=1.1),
             Part("cap", "wood", a=(6.2, 0, 0), b=(6.2, 7.8, 0), r1=0.3),
             Part("cap", "wood", a=(-6.2, 0, 0), b=(-6.2, 7.8, 0), r1=0.3),
             Part("cap", "hair", a=(6.2, 7.6, 0), b=(9.4, 0, 0), r1=0.12)]
    return r3.render(parts, 30, 26, r3.Camera(60, MAP_CAM["pitch"]), (15, 19), ambient=0.34, dither=0.4)


def fire_sprite():
    parts = []
    for i in range(7):
        a = i / 7 * 2 * math.pi
        parts.append(Part("ellip", "iron", c=(math.cos(a) * 3.0, 0.5, math.sin(a) * 3.0), r=(1.1, 0.8, 1.1)))
    parts.append(Part("cap", "wood", a=(-2.2, 0.6, -1.0), b=(2.2, 0.6, 1.0), r1=0.6))
    parts.append(Part("ellip", "ember_eye", c=(0, 1.8, 0), r=(1.5, 2.2, 1.5)))
    parts.append(Part("ellip", "yellow_eye", c=(0.2, 2.6, 0.4), r=(0.8, 1.6, 0.8)))
    return r3.render(parts, 14, 14, r3.Camera(30, MAP_CAM["pitch"]), (7, 10), ambient=0.3, dither=0.3)


def map_unit(name, facing="down"):
    yaw = {"down": 110, "up": -70, "side": 20}[facing]
    return r3.render(fg.CLASSES[name](), 32, 32, r3.Camera(yaw, 30, scale=0.48, bulk=1.3), (16, 30),
                     ambient=0.34, dither=0.0)
