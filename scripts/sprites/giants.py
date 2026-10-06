"""Large and strange battle creatures (art set S3): ogre, ent, lich, ice
guardian, fungus, fey and the small oddities.  Side-on, facing right."""
import math

from beasts import speckle
from kit import BREATH, SWAY, limb, stroke, shade_pixels, recolor, noise, poly
from pixels import Canvas, rect, round_rect, line, thick, ellipse


def ogre(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    hip = fy - 28
    # far leg and arm (behind, shaded)
    far_leg = limb((52, hip), (56, fy - 4), 11)
    cv.part(far_leg, "ashmoss")
    cv.part(rect(54, fy - 4, 69, fy), "leather")
    shade_pixels(cv, far_leg | rect(54, fy - 4, 69, fy))
    # near leg
    leg = limb((38, hip), (36, fy - 4), 12)
    cv.part(leg, "ashmoss")
    cv.part(rect(34, fy - 4, 52, fy), "leather")
    # torso: hunched, the back hump high behind, belly slung forward
    top = hip - 40 + b
    torso = ellipse(42, hip - 12 + b // 2, 21, 20) | ellipse(33, top + 14, 15, 14)
    cv.part(torso, "ashmoss")
    belly = ellipse(54, hip - 4 + b // 2, 12, 12)
    cv.part(belly, "ashmoss", flat="m")
    speckle(cv, torso | belly, "ashmoss", 31, 0.08)
    # loincloth with a rope belt
    cv.part(rect(36, hip - 1, 62, hip + 9), "leather")
    cv.part({(x, hip - 1) for x in range(36, 63)}, "wood", flat="l")
    for x in range(37, 62, 5):
        cv.put(x, hip + 10, "leather.d")
        cv.put(x, hip + 11, "leather.d")
    # club: a tree-trunk planted in front, gripped in the near hand
    hx, hy = 82, hip - 8 + b
    cv.part(limb((84, fy - 2), (80, top - 12 + SWAY[t]), 6), "wood")
    cv.part(ellipse(80, top - 14 + SWAY[t], 7, 10), "wood")
    for x, y in ((76, top - 18), (82, top - 20), (80, top - 8), (75, top - 11), (85, top - 14)):
        cv.put(x, y + SWAY[t], "iron.l")
    # near arm, thick, reaching forward to the grip
    arm = limb((50, top + 10), (hx - 2, hy), 9)
    cv.part(arm, "ashmoss")
    cv.part(ellipse(hx, hy + 1, 5, 5), "ashmoss")
    cv.part({(x, y) for x, y in arm if y < top + 14 + (x - 50) // 4}, "ashmoss", flat="l")
    # head: low and forward under a heavy brow
    hd = [0, 1, 1, 0][t]
    hx0, hy0 = 64, top + 6 + hd
    head = ellipse(hx0, hy0, 11, 9)
    cv.part(head, "ashmoss")
    cv.part({(x, hy0 - 3) for x in range(hx0 - 2, hx0 + 10)}, "ashmoss", flat="d")  # brow
    cv.fill({(hx0 + 5, hy0 - 1), (hx0 + 6, hy0 - 1)}, "outline")
    cv.put(hx0 + 6, hy0, "ember.m")
    cv.part(rect(hx0 + 4, hy0 + 3, hx0 + 10, hy0 + 5), "oxblood", flat="d")  # mouth
    cv.part({(hx0 + 4, hy0 + 2), (hx0 + 4, hy0 + 1), (hx0 + 10, hy0 + 2), (hx0 + 10, hy0 + 1)}, "bone")  # tusks
    cv.part(rect(hx0 - 4, hy0 - 7, hx0 - 2, hy0 - 5), "ashmoss", flat="d")
    cv.outline()
    return cv


def ent(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    sway = [0, 1, 0, -1][t]
    # root legs
    for x0, x1 in ((34, 28), (52, 60)):
        cv.part(limb((x0, fy - 34), (x1, fy - 2), 9), "leather")
        cv.part(rect(min(x0, x1) - 4, fy - 4, max(x0, x1) + 8, fy), "leather")
    # trunk
    trunk = poly([(30, fy - 30), (56, fy - 30), (60, fy - 62), (54, fy - 78 + b), (36, fy - 78 + b), (26, fy - 62)])
    cv.part(trunk, "leather")
    speckle(cv, trunk, "leather", 41, 0.3)
    for y in range(fy - 74, fy - 32, 4):
        cv.fill({(x, y) for x in range(34 + (y % 7), 52, 5)}, "leather.d")
    # far arm
    far = limb((54, fy - 64), (74, fy - 40 + sway), 7)
    cv.part(far, "leather")
    shade_pixels(cv, far)
    # near arm: a thick branch ending in twigs
    arm = limb((34, fy - 66 + b), (22, fy - 42 - sway), 8)
    cv.part(arm, "leather")
    for dx, dy in ((-3, 4), (-1, 6), (2, 5), (-6, 1)):
        cv.part(limb((22, fy - 42 - sway), (22 + dx, fy - 42 - sway + dy + 6), 1), "wood", flat="d")
    # face on the right side of the upper trunk, a mossy beard below
    fx, fy0 = 52, fy - 66 + b
    cv.fill({(fx, fy0), (fx + 1, fy0), (fx + 4, fy0), (fx + 5, fy0)}, "outline")
    cv.put(fx + 1, fy0 + 1, "ember.m")
    cv.put(fx + 5, fy0 + 1, "ember.m")
    cv.fill({(fx + 2, fy0 + 2), (fx + 3, fy0 + 3)}, "leather.d")  # nose ridge
    cv.fill(rect(fx, fy0 + 7, fx + 7, fy0 + 9), "outline")
    for k in range(5):
        cv.part(limb((fx + k * 2 - 2, fy0 + 9), (fx + k * 2 - 3 + (sway if k % 2 else 0), fy0 + 18), 1), "moss", flat="m")
    # leafy crown in clumps
    for cx, cy, r in ((38, fy - 88, 11), (54, fy - 86, 10), (28, fy - 80, 8), (62, fy - 78, 8), (46, fy - 92, 9)):
        cx += sway if (cx // 2) % 2 else 0
        clump = ellipse(cx, cy + b, r, r - 2)
        cv.part(clump, "forest")
        cv.fill({(cx - 3, cy - 3 + b), (cx - 2, cy - 3 + b), (cx - 3, cy - 2 + b)}, "moss.m")
    # moss patches on the trunk
    for x, y in ((38, fy - 56), (43, fy - 44), (36, fy - 40), (50, fy - 36)):
        cv.part({(x, y), (x + 1, y), (x, y + 1)}, "moss", flat="m")
    cv.outline()
    return cv


def lich(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    sway = [0, 1, 0, -1][t]
    # staff with soulfire
    sx = 70
    cv.part(thick({(sx + (1 if (y // 8) % 2 else 0), y) for y in range(fy - 74, fy)}, 2), "wood", flat="m")
    flame = [
        [(0, -2), (-1, -5), (1, -6), (0, -9)],
        [(0, -2), (1, -5), (-1, -7), (0, -10)],
        [(0, -2), (-1, -6), (1, -7), (1, -11)],
        [(0, -2), (1, -5), (0, -8), (-1, -9)],
    ][t]
    fy0 = fy - 76
    cv.part(ellipse(sx + 1, fy0, 3, 3), "moss", flat="l")
    for dx, dy in flame:
        cv.put(sx + 1 + dx, fy0 + dy, "ashmoss.l")
        cv.put(sx + 1 + dx, fy0 + dy + 1, "moss.l")
    cv.fill({(sx - 2, fy0), (sx + 4, fy0), (sx + 1, fy0 - 3 - (t == 2))}, "moss.m")
    # robe: tall, tattered at the hem
    robe = set()
    for y in range(fy - 62 + b, fy - 1):
        grow = (y - (fy - 62)) * 0.38
        for x in range(int(40 - 8 - grow), int(40 + 8 + grow) + 1):
            robe.add((x, y))
    ragged = {(x, y) for x, y in robe if y > fy - 8 and (x * 7 + y) % 5 == 0}
    robe -= ragged
    cv.part(robe, "charcoal")
    for x in range(26, 56, 6):
        cv.fill({(x, y) for y in range(fy - 40, fy - 4, 1) if (y + x) % 9 == 0}, "plum.d")
    cv.part({(x, y) for x, y in robe if y >= fy - 3}, "plum", flat="d")
    cv.part(rect(30, fy - 44, 50, fy - 41), "plum", flat="m")  # sash
    # shoulders / hood and skull
    cv.part(ellipse(40, fy - 62 + b, 14, 6), "charcoal")
    hood = ellipse(41, fy - 72 + b, 8, 9)
    cv.part(hood, "charcoal")
    skull = ellipse(44, fy - 71 + b, 5.5, 6)
    cv.part(skull, "bone")
    cv.part(rect(46, fy - 66 + b, 51, fy - 62 + b), "bone", flat="d")  # jaw
    cv.fill({(46, fy - 66 + b), (48, fy - 66 + b), (50, fy - 66 + b)}, "outline")
    glow = "ashmoss.l" if t != 2 else "moss.l"
    cv.fill({(46, fy - 72 + b), (47, fy - 72 + b)}, "outline")
    cv.put(47, fy - 72 + b, glow)
    cv.fill({(x, fy - 80 + b + (1 if x % 2 else 0)) for x in range(37, 49)}, "brass.m")  # crown
    for x in (38, 42, 46):
        cv.fill({(x, fy - 82 + b), (x, fy - 81 + b)}, "brass.l")
    # near arm raised to the staff, bony hand
    cv.part(limb((48, fy - 60 + b), (sx - 1, fy - 50), 4), "charcoal")
    cv.part(rect(sx - 3, fy - 52, sx + 1, fy - 48), "bone")
    cv.put(sx - 4, fy - 50, "bone.d")
    cv.outline()
    return cv


def ice_guardian(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    glint = t == 2
    # legs: two faceted pillars
    for x0 in (28, 52):
        leg = poly([(x0, fy - 36), (x0 + 16, fy - 36), (x0 + 18, fy - 2), (x0 - 2, fy - 2)])
        cv.part(leg, "slate")
        cv.part({(x, y) for x, y in leg if x < x0 + 5}, "steel", flat="m")
    shade_pixels(cv, poly([(28, fy - 36), (44, fy - 36), (46, fy - 2), (26, fy - 2)]))
    # torso: a great chunk of ice, broad shoulders
    torso = poly([(26, fy - 38), (64, fy - 38), (72, fy - 62 + b), (60, fy - 78 + b), (36, fy - 78 + b), (20, fy - 62 + b)])
    cv.part(torso, "water")
    cv.part({(x, y) for x, y in torso if x - 20 < (fy - y) * 0.15 and y < fy - 44}, "steel", flat="l")
    for (x0, y0, x1, y1) in ((30, fy - 74, 42, fy - 46), (46, fy - 72, 56, fy - 50)):  # facet cracks
        cv.fill(line(x0, y0 + b, x1, y1 + b), "slate.d")
    # shoulder spikes
    for dx, dy in ((-1, -1), (1, -1), (0, -2)):
        pass
    cv.part(poly([(18, fy - 60 + b), (26, fy - 70 + b), (30, fy - 56 + b)]), "steel")
    cv.part(poly([(62, fy - 68 + b), (70, fy - 82 + b), (72, fy - 64 + b)]), "steel")
    # arms: blocks of ice ending in fists
    cv.part(limb((64, fy - 62 + b), (74, fy - 34), 10), "water")
    cv.part(rect(70, fy - 36, 82, fy - 24), "steel")
    cv.part(limb((24, fy - 60 + b), (16, fy - 34), 9), "slate")
    cv.part(rect(8, fy - 36, 20, fy - 24), "steel")
    # head: a diamond with a cold core
    head = poly([(48, fy - 100 + b), (58, fy - 88 + b), (48, fy - 76 + b), (38, fy - 88 + b)])
    cv.part(head, "steel")
    cv.part({(x, y) for x, y in head if x > 50}, "water", flat="d")
    cv.fill({(47, fy - 89 + b), (48, fy - 89 + b)}, "bone.l" if glint else "steel.l")
    cv.fill({(52, fy - 87 + b)}, "bone.l")
    # core glow in the chest
    cv.fill({(46, fy - 56 + b), (47, fy - 56 + b), (46, fy - 55 + b)}, "bone.l" if glint else "steel.l")
    # frost mist at the feet
    for x in range(16, 82, 5):
        if noise(x, t, 3) > 0.5:
            cv.put(x + t, fy - 1, "steel.l")
    cv.outline()
    return cv


def unknown_large(t, W=96):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    for x0 in (34, 52):
        cv.part(limb((x0 + 5, fy - 30), (x0 + 5, fy - 3), 11), "charcoal")
        cv.part(rect(x0 - 1, fy - 5, x0 + 15, fy), "charcoal")
    shade_pixels(cv, limb((39, fy - 30), (39, fy - 3), 11))
    torso = ellipse(48, fy - 52 + b, 24, 24)
    cv.part(torso, "charcoal")
    cv.part(limb((64, fy - 66 + b), (76, fy - 34), 9), "charcoal")
    cv.part(ellipse(76, fy - 32, 5, 5), "charcoal")
    cv.part(limb((32, fy - 66 + b), (22, fy - 36), 9), "charcoal")
    cv.part(ellipse(22, fy - 34, 5, 5), "charcoal")
    cv.part(ellipse(60, fy - 78 + b + [0, 1, 1, 0][t], 9, 8), "charcoal")
    cv.put(65, fy - 79 + b, "ember.d")
    speckle(cv, torso, "charcoal", 77, 0.2)
    cv.outline()
    return cv


# -- smaller oddities ---------------------------------------------------------------------

def fungus(t, W=64):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    # three walking mushrooms, the big one in the middle
    for cx, ph, capw, caph, stalk, cap in ((18, 0, 9, 6, 16, "ochre"), (46, 2, 8, 5, 14, "heather"),
                                            (32, 1, 13, 9, 22, "oxblood")):
        bob = [0, 1, 0, 0][(t + ph) % 4]
        top = fy - stalk - caph * 2 + bob
        # stubby legs
        for dx, lift in ((-3, (t + ph) % 2), (3, (t + ph + 1) % 2)):
            cv.part(rect(cx + dx - 1, fy - 5 - lift, cx + dx + 1, fy - lift), "wool", flat="m")
        cv.part(rect(cx - 3, top + caph + 1, cx + 3, fy - 5), "wool")
        cv.part(round_rect(cx - capw, top, cx + capw, top + caph * 2 - 3), cap)
        cv.part({(x, y) for x in range(cx - capw + 1, cx + capw) for y in range(top + caph * 2 - 3, top + caph * 2 - 1)},
                "wool", flat="d")
        for k in range(3):
            cv.put(cx - capw + 3 + k * (capw * 2 - 6) // 2, top + 2 + (k % 2), "bone.l")
        cv.fill({(cx + 2, top + caph + 2), (cx + 4, top + caph + 2)}, "outline")  # sunken eyes
    # spore puffs drift up
    for i in range(6):
        px = 14 + i * 7 + (t * 2 if i % 2 else -t)
        py = 14 - (t * 2 + i * 3) % 14 + 8
        cv.put(px, py, "moss.l" if i % 2 else "ashmoss.l")
    cv.outline()
    return cv


def snow_floof(t, W=48):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    body = ellipse(22, fy - 11 + b // 2, 15, 11 - b // 2)
    for x, y in ((9, fy - 4), (15, fy - 2), (27, fy - 2), (33, fy - 4)):  # stubby legs
        cv.part(rect(x, y, x + 3, fy), "steel", flat="d")
    cv.part(body, "steel")
    cv.part({(x, y) for x, y in body if y < fy - 14}, "bone", flat="l")
    speckle(cv, body, "steel", 51, 0.35)
    for x, y in ((10, fy - 16), (18, fy - 21), (26, fy - 20), (31, fy - 14)):  # tufts
        cv.put(x, y, "steel.l")
        cv.put(x + 1, y + 1, "steel.d")
    cv.part(ellipse(8, fy - 10, 3, 3), "steel")  # tail stub
    # glaring eyes, a dark mouth with teeth that show when it opens
    cv.fill({(31, fy - 14), (32, fy - 14), (33, fy - 14)}, "outline")
    cv.put(32, fy - 13, "ember.m")
    open_ = t in (1, 2)
    cv.fill({(x, fy - 8) for x in range(30, 37 + open_)}, "oxblood.d")
    if open_:
        cv.fill({(x, fy - 7) for x in range(31, 37)}, "oxblood.d")
        cv.fill({(31, fy - 9), (33, fy - 9), (35, fy - 9), (32, fy - 6), (34, fy - 6)}, "bone.l")
    cv.outline()
    return cv


def faerie(t, W=48):
    cv = Canvas(W, W)
    fy = W - 3
    b = [0, 1, 0, -1][t]
    flap = [6, 3, -1, 3][t]
    # moth-like wings behind
    cv.part(poly([(20, fy - 22), (6, fy - 40 - flap), (3, fy - 26), (8, fy - 16), (19, fy - 16)]), "heather")
    cv.part(poly([(22, fy - 22), (34, fy - 38 - flap), (38, fy - 26), (32, fy - 16), (23, fy - 16)]), "plum")
    for x, y in ((8, fy - 28), (6, fy - 31), (35, fy - 28), (33, fy - 24)):
        cv.put(x, y - flap // 2, "bone.d")
    # thin long limbs
    cv.part(limb((20, fy - 14), (17, fy - 1), 1), "bone", flat="d")
    cv.part(limb((24, fy - 14), (27, fy - 1), 1), "bone", flat="m")
    torso = rect(19, fy - 28 + b, 24, fy - 14)
    cv.part(torso, "wool")
    cv.part({(x, fy - 14) for x in range(18, 26)}, "wool", flat="d")
    cv.part(limb((24, fy - 26 + b), (28, fy - 14 + b), 1), "bone", flat="m")
    cv.part(limb((19, fy - 26 + b), (15, fy - 16 + b), 1), "bone", flat="d")
    head = ellipse(22, fy - 31 + b, 3.5, 4.5)
    cv.part(head, "bone")
    cv.put(24, fy - 31 + b, "outline")
    cv.fill({(21, fy - 36 + b), (22, fy - 37 + b), (23, fy - 36 + b)}, "bone.d")
    # an uncanny cold glow
    for x, y in ((14, fy - 12), (30, fy - 8), (12, fy - 24), (34, fy - 34), (24, fy - 42)):
        if (x + y + t) % 3:
            cv.put(x, y, "bone.l")
    cv.outline()
    return cv


def imp_forest(t, W=48):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    cv.part(limb((16, fy - 12 + b), (13, fy - 2), 3), "wood", flat="d")
    cv.part(rect(11, fy - 2, 16, fy), "wood")
    cv.part(limb((22, fy - 12 + b), (24, fy - 2), 3), "wood")
    cv.part(rect(22, fy - 2, 28, fy), "wood")
    cv.part(limb((14, fy - 12), (8, fy - 8 + [0, 1, 0, -1][t]), 1), "leather")  # tail
    torso = ellipse(21, fy - 18 + b, 6, 7)
    cv.part(torso, "leather")
    speckle(cv, torso, "leather", 61, 0.3)
    cv.part(limb((26, fy - 22 + b), (31, fy - 12 + b), 2), "leather")  # long near arm
    cv.put(32, fy - 11 + b, "bone.l")
    cv.put(33, fy - 12 + b, "bone.l")
    cv.part(limb((17, fy - 22 + b), (13, fy - 10 + b), 2), "leather", flat="d")
    head = ellipse(24, fy - 28 + b, 5, 4.5)
    cv.part(head, "wood")
    cv.part(rect(27, fy - 27 + b, 30, fy - 25 + b), "wood", flat="m")  # snout
    cv.put(26, fy - 29 + b, "ember.m")
    cv.fill({(28, fy - 24 + b), (30, fy - 24 + b)}, "bone.l")
    for dx, dy in ((20, -34), (19, -37), (18, -38), (26, -34), (27, -37), (28, -38)):
        cv.put(dx, fy + dy + b, "wood.l")
    cv.put(21, fy - 35 + b, "moss.m")
    cv.outline()
    return cv
