"""The creature companions (art pass): the Beast Tamer's war bear and drake
hatchling (battle idles) and the hound recruit's map sprite.  The stone golem's
map sprite is the humanoid rig in figures.py (draw_golem).  Same rules as the
rest of S3/S5: palette colors only, a 1 px outline, top-left light.
"""
from kit import BREATH, limb, poly
from pixels import Canvas, rect, line, ellipse
from beasts import speckle, leg


# -- war bear (battle, side-on, drawn on 96 and shrunk to the L frame) ------------------

def war_bear(t, W=96):
    cv = Canvas(W, W)
    fy = W - 4
    b = BREATH[t]
    bx, by = 44, fy - 33 + (1 if b == 2 else 0)
    # far legs (shaded), then the body, then near legs
    for fx in (bx + 17, bx - 18):
        leg(cv, (fx, by + 8), (fx + (1 if t == 1 else 0), fy), 8, "leather", far=True)
    body = ellipse(bx, by, 28, 17)
    cv.part(body, "leather")
    hump = ellipse(bx + 17, by - 9, 13, 11)  # the shoulder hump
    cv.part(hump, "leather")
    belly = {(x, y) for x, y in body if y >= by + 11 and x < bx + 20}
    cv.part(belly, "wood", flat="m")
    cv.part(ellipse(bx - 28, by - 2, 3, 3), "leather")  # stub tail
    for fx in (bx + 12, bx - 24):
        leg(cv, (fx, by + 9), (fx - (1 if t == 3 else 0), fy), 9, "leather")
        for i in range(3):  # claws on the near paws
            cv.put(fx + 2 + i * 3, fy, "bone.l")
    # head, low and forward, a short blunt muzzle, round ears
    hd = [0, 1, 1, 0][t]
    hx, hy = bx + 36, by - 7 + hd
    cv.part(limb((bx + 24, by - 12), (hx - 2, hy + 2), 9), "leather")  # neck
    cv.part(ellipse(hx + 2, hy + 3, 11, 9), "leather")
    cv.part(rect(hx + 7, hy + 4, hx + 18, hy + 11), "wood")  # muzzle
    cv.fill({(hx + 18, hy + 5), (hx + 18, hy + 6), (hx + 17, hy + 5), (hx + 17, hy + 6)}, "charcoal.d")  # nose
    cv.part(ellipse(hx - 4, hy - 6, 3, 3), "leather")
    cv.part(ellipse(hx + 3, hy - 7, 3, 3), "leather")
    cv.put(hx - 4, hy - 6, "wood.m")
    cv.put(hx + 3, hy - 7, "wood.m")
    cv.fill({(hx + 8, hy + 1), (hx + 9, hy + 1)}, "ember.m")  # eye
    if t in (1, 2):  # a snarl
        cv.fill({(x, hy + 12) for x in range(hx + 9, hx + 18, 2)}, "bone.l")
    # war harness: an iron-studded strap across the chest and a collar
    cv.fill(line(bx + 14, by - 12, bx + 23, by + 12), "leather.d")
    cv.fill(line(bx + 15, by - 12, bx + 24, by + 12), "leather.d")
    for sx, sy in ((bx + 16, by - 6), (bx + 19, by + 1), (bx + 22, by + 8)):
        cv.fill({(sx, sy), (sx + 1, sy)}, "brass.m")
    cv.fill(line(hx - 6, hy + 11, hx + 4, hy + 13), "iron.m")
    cv.fill(line(hx - 6, hy + 12, hx + 4, hy + 14), "iron.d")
    speckle(cv, body | hump, "leather", 21)
    cv.outline()
    return cv


# -- drake hatchling (battle) ------------------------------------------------------------

def drake_hatchling(t, W=64):
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    bx, by = 28, fy - 13 + (1 if b == 2 else 0)
    for fx in (bx + 7, bx - 9):
        leg(cv, (fx, by + 3), (fx + (1 if t == 1 else 0), fy), 3, "forest", far=True)
    # tail: long, curling up at the tip, ending in a spade
    sway = [0, 1, 0, -1][t]
    cv.part(limb((bx - 8, by - 1), (bx - 16, by + 4 + sway), 4), "forest")
    cv.part(limb((bx - 16, by + 4 + sway), (bx - 23, by + 1 + sway), 3), "forest")
    cv.part(limb((bx - 23, by + 1 + sway), (bx - 26, by - 5 + sway), 2), "forest")
    cv.part({(bx - 27, by - 8 + sway), (bx - 26, by - 8 + sway), (bx - 25, by - 7 + sway),
             (bx - 26, by - 7 + sway), (bx - 27, by - 7 + sway)}, "ember", flat="m")
    body = ellipse(bx, by, 12, 7)
    cv.part(body, "forest")
    cv.part({(x, y) for x, y in body if y >= by + 3}, "ochre", flat="m")
    for fx in (bx + 5, bx - 10):
        leg(cv, (fx, by + 4), (fx - (1 if t == 3 else 0), fy), 3, "forest")
    # folded wing on the back
    cv.part(poly([(bx - 6, by - 5), (bx + 2, by - 17 + BREATH[t]), (bx + 10, by - 5)]), "moss")
    cv.fill(line(bx - 5, by - 6, bx + 2, by - 16 + BREATH[t]), "forest.d")
    cv.fill(line(bx + 9, by - 6, bx + 2, by - 16 + BREATH[t]), "forest.d")
    # neck up and a wedge head
    hd = [0, 1, 1, 0][t]
    nx, ny = bx + 15, by - 12 + hd
    cv.part(limb((bx + 8, by - 3), (nx, ny + 1), 5), "forest")
    cv.part(ellipse(nx + 2, ny, 5, 4), "forest")
    cv.part(rect(nx + 5, ny, nx + 11, ny + 3), "forest", flat="m")  # snout
    cv.part(rect(nx + 5, ny + 3, nx + 10, ny + 4), "ochre", flat="m")  # lower jaw
    cv.put(nx + 11, ny + 1, "charcoal.d")
    cv.put(nx + 3, ny - 1, "ember.m")
    cv.fill(line(nx, ny - 4, nx - 2, ny - 8), "bone.m")  # horns
    cv.fill(line(nx + 3, ny - 4, nx + 2, ny - 8), "bone.m")
    # smoke curls from the nostril, a spark when it breathes out
    for i, (sx, sy) in enumerate([(nx + 12, ny - 1), (nx + 13, ny - 3), (nx + 12, ny - 5), (nx + 14, ny - 7)]):
        if i <= t:
            cv.put(sx, sy, "stone.l" if i % 2 else "stone.m")
    if t == 2:
        cv.put(nx + 12, ny + 1, "ember.l")
    for i in range(5):  # back spines
        cv.put(bx - 8 + i * 4, by - 8 + (1 if i in (0, 4) else 0), "bone.m")
    speckle(cv, body, "forest", 33)
    cv.outline()
    return cv


# -- hound recruit (map, 32 px, three views) ---------------------------------------------

FUR, BELLY = "leather", "ochre"
FY = 30  # feet row


def hound_frame(view, pose):
    cv = Canvas(32, 32)
    bob = pose.get("bob", 0)
    lift = pose.get("lift", (0, 0))
    stride = pose.get("stride", (0, 0))
    wag = 1 if pose.get("tail") else 0
    by = 21 + bob
    if view == "side":
        for i, lx in enumerate((8, 19)):  # far legs first
            x = lx + stride[i]
            cv.part(rect(x + 1, by + 2, x + 2, FY - lift[i] - 1), FUR, flat="d")
            cv.part(rect(x + 1, FY - lift[i], x + 3, FY - lift[i]), FUR, flat="d")
        cv.part(limb((6, by - 1), (3, by - 5 - wag), 2), FUR)
        body = ellipse(14, by, 8, 4)
        cv.part(body, FUR)
        cv.part({(x, y) for x, y in body if y >= by + 3}, BELLY, flat="m")
        for i, lx in enumerate((10, 17)):  # near legs
            x = lx - stride[i]
            cv.part(rect(x, by + 3, x + 1, FY - lift[1 - i] - 1), FUR)
            cv.part(rect(x, FY - lift[1 - i], x + 2, FY - lift[1 - i]), FUR, flat="m")
        cv.part(limb((19, by - 2), (22, by - 5), 3), FUR)
        cv.part(ellipse(24, by - 7, 3, 3), FUR)
        cv.part(rect(26, by - 7, 29, by - 5), BELLY, flat="m")
        cv.put(29, by - 7, "charcoal.d")
        cv.put(26, by - 9, "outline")
        cv.part({(22, by - 10), (23, by - 11), (23, by - 10)}, FUR, flat="d")  # ear
        cv.put(21, by - 4, "iron.l")  # collar
        cv.put(22, by - 4, "leather.d")
    elif view == "down":
        for i, x in enumerate((12, 18)):
            cv.part(rect(x, by + 2, x + 1, FY - lift[i] - 1), FUR)
            cv.part(rect(x, FY - lift[i], x + 2, FY - lift[i]), FUR, flat="m")
        cv.part(rect(12, by - 2, 19, by + 4), FUR)
        cv.part(rect(14, by + 1, 17, by + 4), BELLY, flat="m")
        cv.part(rect(13, by - 10, 18, by - 4), FUR)  # a narrow head ...
        cv.part(rect(14, by - 6, 17, by - 1), BELLY, flat="m")  # ... with a long muzzle
        cv.put(15, by - 1, "charcoal.d")
        cv.put(16, by - 1, "charcoal.d")
        cv.put(14, by - 8, "outline")
        cv.put(17, by - 8, "outline")
        cv.part({(12, by - 11), (12, by - 12), (13, by - 11), (13, by - 10), (12, by - 10)}, FUR, flat="d")  # upright ears
        cv.part({(19, by - 11), (19, by - 12), (18, by - 11), (18, by - 10), (19, by - 10)}, FUR, flat="d")
        cv.fill({(x, by - 3 + 2) for x in range(13, 19)}, "leather.d")
        cv.fill({(15, by - 1 + 1), (16, by - 1 + 1)}, "iron.l")
    else:  # up: seen from behind
        for i, x in enumerate((10, 20)):
            cv.part(rect(x, by + 2, x + 2, FY - lift[i] - 1), FUR)
            cv.part(rect(x, FY - lift[i], x + 2, FY - lift[i]), FUR, flat="m")
        cv.part(rect(10, by - 4, 21, by + 4), FUR)
        cv.part(limb((15, by - 3), (15, by - 6 - wag), 2), FUR, )
        cv.part(rect(13, by - 10, 18, by - 5), FUR)
        cv.part({(12, by - 11), (12, by - 12), (13, by - 11), (13, by - 10), (12, by - 10)}, FUR, flat="d")
        cv.part({(19, by - 11), (19, by - 12), (18, by - 11), (18, by - 10), (19, by - 10)}, FUR, flat="d")
    speckle(cv, {(x, y) for x in range(32) for y in range(32) if cv.get(x, y)}, FUR, 5, 0.05)
    cv.outline()
    return cv


IDLE = [dict(bob=0), dict(bob=1, tail=True)]
WALK = [
    dict(bob=0, lift=(0, 1), stride=(-2, 2)),
    dict(bob=-1, lift=(0, 0), stride=(0, 0), tail=True),
    dict(bob=0, lift=(1, 0), stride=(2, -2)),
    dict(bob=-1, lift=(0, 0), stride=(0, 0), tail=True),
]


def hound_idle():
    return [[hound_frame(v, p) for p in IDLE] for v in ("down", "up", "side")]


def hound_walk():
    return [[hound_frame(v, p) for p in WALK] for v in ("down", "up", "side")]
