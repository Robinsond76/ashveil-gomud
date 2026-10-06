"""Beasts and crawlers for battle (art set S3): canines, rodents, spiders,
cave creatures, bats and the crocodile.  Side-on, facing right, four idle frames.
"""
import math

from kit import BREATH, SWAY, limb, stroke, shade_pixels, recolor, noise, poly
from pixels import Canvas, rect, round_rect, line, thick, ellipse


def speckle(cv, pixels, ramp, seed, density=0.12):
    """Break up flat fur with a few light and dark flecks (no new colors)."""
    for x, y in sorted(pixels):
        n = noise(x, y, seed)
        c = cv.get(x, y)
        if c is None or c == "outline" or not c.startswith(ramp + "."):
            continue
        if n < density / 2:
            cv.put(x, y, ramp + ".d")
        elif n > 1 - density / 2:
            cv.put(x, y, ramp + ".l")


def leg(cv, top, foot, w, ramp, boot=None, far=False):
    """A simple two-part leg from top (x, y) to foot (x, y); paw drawn as a block."""
    (x0, y0), (x1, y1) = top, foot
    px = limb((x0, y0), (x1, y1 - 2), w)
    cv.part(px, ramp)
    paw = rect(x1 - 1, y1 - 2, x1 + w + 1, y1)
    cv.part(paw, boot or ramp)
    if far:
        shade_pixels(cv, px | paw)


# ---------------------------------------------------------------------------
# Canines
# ---------------------------------------------------------------------------

def canine(t, W=64, fur="stone", belly=None, mastiff=False, seed=1, collar=False, mangy=False):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    scale = 1.25 if mastiff else 1.0
    ly = round(11 * scale)
    by = fy - ly - 5 + (1 if b == 2 else 0)
    bx = 26 if mastiff else 28
    rx, ry = round(15 * scale), round(8 * scale)
    lw = 4 if mastiff else 3
    # far legs (shaded), then tail, body, near legs
    for fx in (bx + rx - 8, bx - rx + 6):
        leg(cv, (fx, by + 2), (fx + (1 if t == 1 else 0), fy), lw, fur, far=True)
    tail_dy = [0, 1, 0, -1][t]
    if mastiff:
        cv.part(limb((bx - rx, by - 2), (bx - rx - 4, by + 3 + tail_dy), 3), fur)
    else:
        cv.part(limb((bx - rx + 2, by - 2), (bx - rx - 6, by + 8 + tail_dy), 3), fur)
        cv.part(limb((bx - rx - 6, by + 8 + tail_dy), (bx - rx - 8, by + 14 + tail_dy), 2), fur)
    body = ellipse(bx, by, rx, ry)
    cv.part(body, fur)
    # lighter belly line
    bel = {(x, y) for x, y in body if y >= by + ry - 2 and x < bx + rx - 3}
    cv.part(bel, belly or fur, flat="l" if belly else None)
    for fx in (bx + rx - 5, bx - rx + 9):
        leg(cv, (fx, by + 3), (fx - (1 if t == 3 else 0), fy), lw, fur)
    # shoulder hump / neck
    hd = [0, 1, 1, 0][t]
    hx, hy = bx + rx + 3, by - ry + 1 + hd
    cv.part(limb((bx + rx - 4, by - ry + 1), (hx, hy + 2), 5 if mastiff else 4), fur)
    # head
    sx = round(5 * scale)
    head = ellipse(hx + 2, hy + 2, sx, round(4 * scale))
    muzzle = rect(hx + 3, hy + 2, hx + 3 + round(8 * scale), hy + 5 + (1 if mastiff else 0))
    cv.part(head, fur)
    cv.part(muzzle, belly or fur, flat=None)
    cv.fill({(hx + 3 + round(8 * scale), hy + 2), (hx + 3 + round(8 * scale), hy + 3)}, "charcoal.d")
    # ears
    if mastiff:
        cv.part({(hx, hy - 2), (hx + 1, hy - 2), (hx + 2, hy - 2), (hx + 1, hy - 3)}, fur, flat="d")
    else:
        cv.part({(hx, hy - 2), (hx + 1, hy - 3), (hx + 1, hy - 4), (hx + 2, hy - 2), (hx + 2, hy - 3),
                 (hx + 4, hy - 2), (hx + 4, hy - 3)}, fur)
    # eye, teeth
    cv.put(hx + 4, hy + 1, "ember.m")
    jaw = hy + 6
    if t in (1, 2):
        cv.fill({(x, jaw) for x in range(hx + 5, hx + 4 + round(7 * scale), 2)}, "bone.l")
    if collar:
        for i, (cx, cy) in enumerate([(hx - 1, hy + 6), (hx + 1, hy + 7), (hx + 3, hy + 7)]):
            cv.put(cx, cy, "leather.d")
        cv.fill({(hx - 1, hy + 5), (hx + 1, hy + 6), (hx + 3, hy + 6)}, "iron.l")
    speckle(cv, body, fur, seed)
    if mangy:  # bald patches and old scars
        for x, y in ((bx - 4, by - 3), (bx + 3, by - 5), (bx - 8, by + 1)):
            cv.fill({(x, y), (x + 1, y)}, "skin.d")
    cv.outline()
    return cv


def wolf(t):
    return canine(t, fur="stone", belly="wool", seed=3)


def wolf_snow(t):
    return recolor(canine(t, fur="stone", belly="wool", seed=3),
                   {"stone": "steel", "wool": "bone"})


def dog_junkyard(t):
    return canine(t, fur="ochre", belly="leather", mastiff=True, collar=True, mangy=True, seed=5)


def unknown_beast(t):
    """A generic four-legged silhouette, dim and featureless."""
    cv = canine(t, fur="charcoal", belly=None, seed=9)
    return recolor(cv, {"ember": "charcoal"})


# ---------------------------------------------------------------------------
# Rodents, bats and small things (48x48)
# ---------------------------------------------------------------------------

def rat(t, W=48, fur="stone", seed=2):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    by = fy - 7 + (1 if b == 2 else 0)
    # tail: long and bare, curling behind
    tail = [(10, by + 1), (7, by + 3), (4, by + 2 + (t == 1) - (t == 3)), (2, by - 1 + (t == 1)),
            (1, by - 4 + (t == 2))]
    for a, c in zip(tail, tail[1:]):
        cv.part(line(a[0], a[1], c[0], c[1]), "skin", flat="d")
    for x in (14, 24):
        leg(cv, (x, by + 2), (x, fy), 2, fur, far=(x == 14))
    body = ellipse(20, by, 10, 5.5)
    cv.part(body, fur)
    hd = [0, 1, 1, 0][t]
    cv.part(ellipse(31, by + hd, 4.5, 3.5), fur)
    cv.part({(34, by + hd + 1), (35, by + hd + 1), (36, by + hd + 2), (35, by + hd + 2)}, fur)
    cv.fill({(36, by + hd + 2)}, "skin.d")
    cv.part({(29, by + hd - 4), (30, by + hd - 4), (29, by + hd - 3), (30, by + hd - 3)}, "skin", flat="m")
    cv.put(33, by + hd, "ember.m")
    if t in (1, 2):
        cv.fill({(35, by + hd + 3), (34, by + hd + 3)}, "bone.l")
    for x, y in ((37, by + hd + 1), (38, by + hd + 3)):
        cv.put(x, y, "bone.d")
    speckle(cv, body, fur, seed, 0.2)
    cv.outline()
    return cv


def rat_big(t):
    return recolor(rat(t, seed=6), {"stone": "charcoal"})


def bats(t, W=48):
    cv = Canvas(W, W)
    fy = W - 3
    flaps = [[-7, -3, 3, -3], [-3, 3, -3, -7], [3, -3, -7, -3], [-3, -7, -3, 3]]
    spots = [(12, 14), (30, 10), (22, 28), (38, 26)]
    for i, (bx, by) in enumerate(spots):
        up = flaps[i][t]
        by += [0, 1, 0, -1][(t + i) % 4]
        body = ellipse(bx, by, 2, 3)
        # wings: two membrane triangles, tips rise and fall
        near = poly([(bx, by - 1), (bx - 8, by + up), (bx - 5, by + 4), (bx - 1, by + 2)])
        far = poly([(bx + 1, by - 1), (bx + 8, by + up), (bx + 5, by + 4), (bx + 2, by + 2)])
        cv.part(far, "charcoal")
        cv.part(near, "plum")
        cv.part(body, "charcoal")
        cv.fill({(bx + 1, by - 2), (bx + 2, by - 3)}, "charcoal.d")
        cv.put(bx + 1, by - 1, "ember.m")
        cv.fill({(bx + 1, by + 1)}, "bone.l")
    cv.outline()
    return cv


# ---------------------------------------------------------------------------
# Spiders
# ---------------------------------------------------------------------------

def _spider_leg(cv, ax, ay, kx, ky, fx, fy, w, ramp, far=False):
    seg = limb((ax, ay), (kx, ky), w) | limb((kx, ky), (fx, fy), w)
    cv.part(seg, ramp)
    if far:
        shade_pixels(cv, seg)


def spider(t, W=64, s=1.0, body="leather", hair="wood", mark=None, plates=None, queen=False,
           pale=False, seed=4):
    cv = Canvas(W, W)
    fy = W - 3
    by = round(fy - 20 * s) + (1 if BREATH[t] == 2 else 0)
    bx = round(W * 0.5)
    ab_rx, ab_ry = (round(11 * s), round(8 * s)) if not queen else (round(15 * s), round(13 * s))
    ct_rx, ct_ry = round(7 * s), round(5.5 * s)
    w = 1 if s < 1.3 else 2
    rise = [0, 1, 0, -1]
    # legs: far side first.  Four per side radiate from the thorax, knees high.
    attach = [(-3, 0.95), (0, 0.8), (3, 0.8), (6, 0.9)]
    reach = [-1.25, -0.55, 0.45, 1.25]
    for side_far in (True, False):
        for i, (ox, dirn) in enumerate(zip([a[0] for a in attach], reach)):
            ph = rise[(t + i + (2 if side_far else 0)) % 4]
            ax = bx + round(ox * s) + (2 if side_far else 0)
            ay = by + round(2 * s)
            kx = ax + round(dirn * 9 * s)
            ky = by - round((13 if i in (0, 3) else 15) * s) + ph
            fx = ax + round(dirn * 19 * s)
            f_y = fy
            if i == 3 and not queen:  # the forelegs reach up and out
                f_y = by - round(1 * s) + ph
                fx = ax + round(dirn * 15 * s)
            _spider_leg(cv, ax, ay, kx, ky, fx, f_y, w, body, far=side_far)
    ab = ellipse(bx - round(ab_rx * 0.9), by + (1 if queen else 0), ab_rx, ab_ry)
    ct = ellipse(bx + 2, by + 1, ct_rx, ct_ry)
    cv.part(ab, body)
    cv.part(ct, body)
    # fuzz on the abdomen
    speckle(cv, ab, body, seed, 0.3)
    if pale:
        pass
    if mark:  # a pattern on the back
        mx, my = bx - round(ab_rx * 0.9), by - ab_ry + 3
        for i in range(3 if not queen else 5):
            cv.put(mx - 2 + i * round(2 * s), my + i % 2, mark)
            cv.put(mx - 2 + i * round(2 * s), my + 1 + i % 2, mark)
    if plates:
        for x, y in sorted(ct):
            if y < by and (x + y) % 3 == 0:
                cv.put(x, y, plates)
        cv.fill({(bx - round(ab_rx * 0.6) + k, by - ab_ry + 1 + abs(k) // 3) for k in range(-4, 5, 2)},
                plates)
    # head: eyes and fangs
    ex, ey = bx + 2 + round(ct_rx * 0.6), by - 1
    for dx, dy in ((0, 0), (2, -1), (-1, -2), (3, 1)):
        cv.put(ex + dx, ey + dy, "ember.m")
    fang = [0, 1, 1, 0][t]
    cv.fill({(ex + 3, ey + 3), (ex + 3, ey + 4 + fang)}, "bone.l")
    if queen:  # crown of spines
        for i, dx in enumerate(range(-4, 6, 3)):
            top = by - ct_ry - 5 - (i % 2) * 2
            cv.part(thick(line(bx + dx + 2, by - ct_ry + 1, bx + dx + 3, top), 1), "bone", flat="m")
    cv.outline()
    return cv


def spider_hatchling(t):
    return spider(t, W=48, s=0.6, body="bone", hair="wool", seed=8)


def spider_large(t):
    return spider(t, body="leather", hair="wood", seed=4)


def spider_warrior(t):
    return spider(t, body="charcoal", mark="oxblood.m", plates="iron.m", seed=7)


def spider_queen(t):
    cv = spider(t, W=128, s=1.7, body="charcoal", mark="oxblood.m", queen=True, seed=11)
    # egg sacs hang from the abdomen on silk
    fy = 128 - 3
    for i, (ex, ey) in enumerate(((14, fy - 36), (24, fy - 30), (8, fy - 24))):
        cv.fill(line(ex + 2, ey - 6, ex + 2, ey), "bone.d")
        cv.part(ellipse(ex + 2, ey + 3 + (1 if (t + i) % 4 == 1 else 0), 3, 4), "wool")
    cv.outline()
    return cv


# ---------------------------------------------------------------------------
# Cave crawlers
# ---------------------------------------------------------------------------

def creeper(t, W=64, body="bone", glow=None, seed=12):
    cv = Canvas(W, W)
    fy = W - 3
    by = fy - 11 + (1 if BREATH[t] == 2 else 0)
    segs = [(14, by + 1, 7, 6), (26, by, 8, 7), (39, by - 1, 8, 7)]
    for i, (sx, sy, rx, ry) in enumerate(segs):
        # many thin legs under each segment, the feet ticking
        for k in (-3, 3):
            x0 = sx + k
            ph = [0, 1, 0, -1][(t + i + (k > 0)) % 4]
            cv.part(limb((x0, sy + 2), (x0 - 3 + ph, fy), 1), body, flat="d")
            cv.part(limb((x0, sy + 2), (x0 + 3 - ph, fy), 1), body, flat="m")
    for i, (sx, sy, rx, ry) in enumerate(segs):
        seg = ellipse(sx, sy, rx, ry)
        cv.part(seg, body)
        speckle(cv, seg, body, seed + i, 0.25)
    # maw: a circle of teeth, no eyes
    hx, hy = 49, by
    head = ellipse(hx, hy + 1, 6, 6)
    cv.part(head, body)
    cv.fill({(hx + 5, hy - 1), (hx + 6, hy), (hx + 6, hy + 1), (hx + 6, hy + 2), (hx + 5, hy + 3)}, "oxblood.d")
    cv.fill({(hx + 5, hy - 2 + k * 2) for k in range(3)}, "bone.l")
    if t in (1, 2):
        cv.fill({(hx + 7, hy), (hx + 7, hy + 2)}, "bone.l")
    if glow:
        for x, y in ((12, by - 3), (24, by - 5), (30, by - 2), (38, by - 5), (45, by - 4), (20, by + 2)):
            if (x + t) % 3 or t == 2:
                cv.put(x, y, glow)
    cv.outline()
    return cv


def creeper_cave(t):
    return creeper(t, body="bone")


def creeper_abyssal(t):
    return creeper(t, body="plum", glow="water.l", seed=15)


# ---------------------------------------------------------------------------
# Crocodile (96x96, long and low)
# ---------------------------------------------------------------------------

def crocodile(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    by = fy - 11
    sway = [0, 1, 0, -1][t]
    # tail tapering to the left
    pts = [(24, by - 1), (16, by + 1), (10, by + 3 + sway), (5, by + 4 + 2 * sway), (2, by + 2 + 2 * sway)]
    for i, (a, c) in enumerate(zip(pts, pts[1:])):
        cv.part(limb(a, c, 6 - i * 1), "moss")
    # far legs
    for lx in (30, 62):
        cv.part(limb((lx + 2, by + 4), (lx - 3, fy), 4), "moss")
        shade_pixels(cv, limb((lx + 2, by + 4), (lx - 3, fy), 4))
    body = ellipse(46, by - b // 2, 28, 10)
    cv.part(body, "moss")
    cv.part({(x, y) for x, y in body if y > by + 3}, "ashmoss", flat="l")
    # near legs, splayed
    for lx in (28, 64):
        leg_px = limb((lx + 2, by + 3), (lx - 4, fy - 1), 5)
        cv.part(leg_px, "moss")
        cv.part(rect(lx - 8, fy - 2, lx - 1, fy), "moss")
        cv.fill({(lx - 8, fy), (lx - 6, fy), (lx - 4, fy)}, "bone.m")
    # osteoderms along the back
    for x in range(18, 74, 4):
        yy = by - 9 + int(abs(x - 46) * 0.04) - b // 2
        cv.part({(x, yy), (x + 1, yy), (x, yy - 1), (x + 1, yy - 1)}, "forest", flat="m")
    speckle(cv, body, "moss", 21, 0.25)
    # head: long jaws, upper and lower
    open_ = [0, 1, 3, 1][t]
    cv.part(rect(70, by - 9, 92, by - 4), "moss")
    cv.part(rect(70, by - 3 + open_, 91, by + 1 + open_), "moss")
    cv.part(ellipse(72, by - 6, 6, 5), "moss")
    cv.part({(75, by - 12), (76, by - 12), (75, by - 11), (76, by - 11)}, "moss")  # eye ridge
    cv.put(76, by - 11, "ember.m")
    cv.fill({(92, by - 8), (92, by - 7)}, "forest.d")
    for x in range(75, 91, 3):
        cv.put(x, by - 3, "bone.l")
        cv.put(x + 1, by - 2 + open_, "bone.l")
    cv.fill({(x, by - 2 + open_ // 2) for x in range(73, 90)}, "oxblood.d")
    cv.outline()
    return cv
