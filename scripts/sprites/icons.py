"""Icons, map markers and camp pieces (art set S1) drawn from small pixel maps.

Pixel maps use one letter per palette color (see LEGEND).  Maps are authored
inside the frame and outlined afterwards, so a 16x16 icon is drawn in 14x14.
"""
from pixels import Canvas, ascii_art, rect, line, thick, ellipse, round_rect

LEGEND = {
    "o": "outline",
    "b": "water.m", "B": "water.l", "c": "water.d",
    "g": "grass.m", "G": "grass.l", "h": "grass.d",
    "m": "moss.m", "M": "moss.l", "n": "moss.d",
    "r": "oxblood.m", "R": "oxblood.l", "x": "oxblood.d",
    "w": "wood.m", "W": "wood.l", "v": "wood.d",
    "l": "leather.m", "L": "leather.l", "k": "leather.d",
    "s": "stone.m", "S": "stone.l", "t": "stone.d",
    "e": "ember.m", "E": "ember.l", "f": "ember.d",
    "y": "ochre.m", "Y": "ochre.l", "z": "ochre.d",
    "p": "plum.m", "P": "plum.l", "q": "plum.d",
    "i": "iron.m", "I": "iron.l", "j": "iron.d",
    "u": "bone.m", "U": "bone.l", "a": "bone.d",
    "d": "charcoal.d", "D": "charcoal.m", "C": "charcoal.l",
    "T": "steel.m", "V": "steel.l", "A": "steel.d",
    "N": "wool.m", "O": "wool.l", "Q": "wool.d",
    "K": "brass.m", "J": "brass.l", "H": "brass.d",
    "Z": "slate.m", "X": "slate.l", "z2": "slate.d",
    "F": "skin.m", "1": "skin.l", "2": "skin.d",
    "3": "heather.m", "4": "heather.l", "5": "heather.d",
    "6": "ashmoss.m", "7": "ashmoss.l", "8": "ashmoss.d",
    "9": "forest.m", "0": "forest.d",
}


def icon(rows, size=16, outline=True, ox=1, oy=1):
    base = ascii_art(rows, LEGEND)
    cv = Canvas(size, size)
    cv.blit(base, ox, oy)
    if outline:
        cv.outline()
    return cv


# -- resource icons (16x16) ----------------------------------------------

def res_water():
    return icon([
        "......c.......",
        ".....cbc......",
        ".....cbbc.....",
        "....cbBbbc....",
        "...cbBbbbbc...",
        "...cbBbbbbc...",
        "...cbbbbbcc...",
        "....cbbbcc....",
        ".....cccc.....",
        "..............",
        ".b.......b....",
        "bBbb.bbBbbbb..",
        "..cbbbb.cccc..",
    ])


def res_forage():
    return icon([
        "......M.......",
        ".....MMm..mm..",
        "....MmmmmMMm..",
        ".....mmMmmn...",
        "......nm......",
        "...rr..n..rr..",
        "..rRrr.n.rRrr.",
        "..rrrx.n.rrrx.",
        "...xx..n..xx..",
        "......n.......",
    ])


def res_herbs():
    return icon([
        "....M..M..M...",
        "...MmM.MmM.m..",
        "..MmmMmmmMmM..",
        "...mnmmnmmnm..",
        "....mmnmnmm...",
        ".....mmnmm....",
        "....YyYyYyY...",
        "....zyzyzyz...",
        ".....mmnmm....",
        "....mm.n.mm...",
        "...m...n...m..",
    ])


def res_firewood():
    cv = Canvas(16, 16)
    cv.part(thick(line(2, 4, 12, 12), 2), "wood")
    cv.part(thick(line(2, 12, 12, 4), 2), "wood")
    for x, y in ((2, 4), (2, 12)):
        cv.fill({(x, y), (x + 1, y)}, "wood.l")
    cv.fill({(12, 4), (13, 4), (12, 12), (13, 12)}, "wood.l")
    cv.fill({(13, 5), (13, 13)}, "wood.d")
    cv.outline()
    return cv


def res_shelter():
    return icon([
        ".....ssss.....",
        "...sSSSSSss...",
        "..sSSssssSSs..",
        ".sSsstttssSSs.",
        ".sSsttttttsSs.",
        ".ssstttttttss.",
        ".sstttttttttss",
        "sstttttttttsst",
        "ssssttttttssst",
        "tssssssssssstt",
    ], ox=1, oy=3)


def res_fishing():
    return icon([
        "...........U..",
        "....VVVVV.UU..",
        "..VTTTTTTTUU..",
        ".VTVTTTTTAU...",
        ".TTTTTTTAA....",
        "..AATTAAA.....",
        "..............",
        "bB..bB.bB..bB.",
        "bbbbbbbbbbbbbb",
        "cc.cccc.cccccc",
    ], ox=1, oy=2)


def res_game():
    return icon([
        "..............",
        "..LL...LL.....",
        ".LkkL.LkkL....",
        ".LkkL.LkkL....",
        "..kk...kk.....",
        "...k....k.....",
        "........LL..LL",
        ".......LkkL.Lk",
        ".......LkkL.Lk",
        "........kk...k",
    ], ox=1, oy=2)


def res_unknown():
    cv = Canvas(16, 16)
    for p in (line(8, 2, 8, 13), line(2, 8, 13, 8)):
        cv.fill(p, "stone.m")
    for p in (line(5, 5, 11, 11), line(11, 5, 5, 11)):
        cv.fill(p, "stone.d")
    cv.fill({(8, 1), (8, 14), (1, 8), (14, 8)}, "stone.l")
    cv.fill({(8, 3), (7, 8), (9, 8), (8, 7), (8, 9)}, "stone.l")
    cv.outline()
    return cv


def res_depleted():
    return icon([
        "....t....t....",
        "...t......t...",
        "...tttttttt...",
        "..tStStStSt...",
        "..tsSsSsSsS...",
        "...tsStStst...",
        "....tsStst....",
        ".....tttt.....",
    ], ox=1, oy=3)


RESOURCES = {
    "water": res_water, "forage": res_forage, "herbs": res_herbs,
    "firewood": res_firewood, "shelter": res_shelter, "fishing": res_fishing,
    "game": res_game, "unknown": res_unknown, "depleted": res_depleted,
}


# -- status / role icons for the style sample (16x16) ----------------------

def st_bleeding():
    return icon([
        ".....r........",
        "....rRr.......",
        "...rRrrr......",
        "..rRrrrrx.....",
        "..rrrrrrx.....",
        "..rrrrrxx.....",
        "...rrrxx......",
        "....xxx...r...",
        ".........rRr..",
        "........rrrxx.",
        ".........xxx..",
    ])


def st_stunned():
    cv = Canvas(16, 16)
    for r, col in ((5, "brass.l"), (3, "brass.m")):
        ring = ellipse(8, 7, r + 1, r - 1) - ellipse(8, 7, r - 1, r - 3)
        cv.fill(ring, col)
    for x, y in ((3, 4), (12, 5), (7, 11), (11, 10)):
        cv.fill({(x, y), (x + 1, y), (x, y + 1)}, "bone.l")
    cv.outline()
    return cv


def role_fighter():
    return icon([
        "..........TV..",
        ".........TTV..",
        "........TTV...",
        ".......TTV....",
        "..K...TTV.....",
        "...K.TTV......",
        "....KTV.......",
        ".....Kl.......",
        "....lK.K......",
        "...lk...K.....",
    ])


def role_healer():
    return icon([
        "....rrrr......",
        "....rRRr......",
        "....rRRr......",
        "rrrrrRRrrrrr..",
        "rRRRRRRRRRRx..",
        "rRRRRRRRRRRx..",
        "rrrrrRRrrrxx..",
        "....rRRx......",
        "....rRRx......",
        "....xxxx......",
    ])


def st_yield():
    return icon([
        "OO............",
        "ONNO..........",
        "ONNNOOO.......",
        "ONNNNNNNOOO...",
        "ONNNNNNNNNNQ..",
        "ONNNNNNNNNQ...",
        "ONNNNNNNNQ....",
        "ONNNNNNQ......",
        "ONNNNQ........",
        "lkk...........",
        "lk............",
        "lk............",
    ])


def st_dark():
    return icon([
        "....dddd......",
        "...dDDDDd.....",
        "..dDCDDDDd....",
        "..dDDDDDDd....",
        "..dDDDDDdd....",
        "...dDDDddd....",
        "....dDDdd.....",
        ".....dddd.....",
    ], ox=1, oy=3)


# -- map markers ------------------------------------------------------------

def here_ring_frames():
    frames = []
    for rx in (8, 10, 12, 10):
        cv = Canvas(32, 32)
        ry = max(3, round(rx * 0.42))
        ring = ellipse(16, 22, rx, ry) - ellipse(16, 22, rx - 2, ry - 1.4)
        top = {(x, y) for x, y in ring if y <= 22 - ry // 2 or x < 16 - rx // 2}
        cv.fill(ring, "bone.d")
        cv.fill(top, "bone.m")
        cv.fill({(x, y) for x, y in top if y < 22 - ry + 1 and x < 15}, "bone.l")
        frames.append(cv)
    return frames


def company_badge():
    return ascii_art([
        "oooooooooooo",
        "oKKKKKKKKKKo",
        "oKrrrrrrrrHo",
        "oKrrrrrrrrHo",
        "oKrrrrrrrrHo",
        "oKrrrrrrrrHo",
        "oKrrrrrrrrHo",
        ".oKrrrrrrHo.",
        ".oKrrrrrHo..",
        "..oKrrrrHo..",
        "...oKrrHo...",
        "....oooo....",
    ], LEGEND, 12, 12)


def ally_banner_frames():
    a = icon([
        "..K...........",
        "..vmmmmmmm....",
        "..vMmmmmmmmn..",
        "..vmmmmmmmnn..",
        "..vmmmmmnn....",
        "..vnnnnn......",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
    ])
    b = icon([
        "..K...........",
        "..vmmmmm......",
        "..vMmmmmmmn...",
        "..vmmmmmmmmn..",
        "..vmmmmmmnn...",
        "..vnnnnnn.....",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
    ])
    return [a, b]


def walk_target_frames():
    a = icon([
        "..vrrrrrr.....",
        "..vRrrrrrrr...",
        "..vrrrrrrrx...",
        "..vrrrrrxx....",
        "..vxxxxx......",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        ".ttt..........",
    ])
    b = icon([
        "..vrrrr.......",
        "..vRrrrrrr....",
        "..vrrrrrrrrx..",
        "..vrrrrrrxx...",
        "..vxxxxxx.....",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        "..v...........",
        ".ttt..........",
    ])
    return [a, b]


def walk_dot():
    return ascii_art([
        "........",
        "..oooo..",
        ".oUUuo..",
        ".oUuuo..",
        ".oauuo..",
        "..oooo..",
        "........",
        "........",
    ], LEGEND, 8, 8)


def exit_chevron(up):
    rows = [
        "........",
        "...oo...",
        "..oUUo..",
        ".oUuuUo.",
        "oUu..uUo",
        "oo....oo",
        "........",
        "........",
    ]
    if not up:
        rows = list(reversed(rows))
        rows = [r for r in rows]
    return ascii_art(rows, LEGEND, 8, 8)


# -- camp -------------------------------------------------------------------

def _pennant(cv, x, y):
    cv.fill({(x, y + i) for i in range(5)}, "wood.d")
    cv.fill({(x + 1, y), (x + 2, y), (x + 3, y), (x + 1, y + 1), (x + 2, y + 1), (x + 1, y + 2)},
            "moss.m")
    cv.put(x + 1, y, "moss.l")


def tent(ally=False):
    cv = Canvas(32, 32)
    # Bedroll in front of the tent.
    cv.part(rect(7, 25, 14, 27), "moss")
    cv.part(rect(7, 25, 9, 27), "wool")
    # Canvas A-frame: lit left face, shaded right face, dark doorway, pegs.
    left = {(x, y) for y in range(8, 25) for x in range(16 - (y - 7) - 2, 17) if x >= 5}
    right = {(x, y) for y in range(8, 25) for x in range(17, 16 + (y - 7) + 3) if x <= 28}
    cv.part(left, "wool")
    cv.part(right, "wool", flat="m")
    cv.fill({(x, y) for x, y in right if x > 22}, "wool.d")
    door = {(x, y) for y in range(15, 25) for x in range(16 - (y - 14) // 2, 17 + (y - 14) // 2)}
    cv.fill(door, "charcoal.d")
    cv.fill({(16, y) for y in range(8, 15)}, "wool.d")  # ridge seam
    cv.fill(line(5, 24, 28, 24), "leather.d")
    for x in (4, 29):
        cv.fill({(x, 25), (x, 26)}, "wood.m")
    cv.fill({(16, 7), (16, 6)}, "wood.m")
    if ally:
        _pennant(cv, 17, 3)
    cv.outline()
    return cv


def camp_rough(ally=False):
    cv = Canvas(32, 32)
    # A log with a pack leaning on it, behind the fire.
    cv.part(rect(9, 11, 22, 13), "wood")
    cv.fill({(9, 11), (9, 12), (9, 13)}, "wood.l")
    cv.part(round_rect(15, 5, 21, 11), "leather")
    cv.fill({(17, 5), (18, 5)}, "bone.m")
    cv.fill(line(16, 8, 20, 8), "leather.d")
    # Stone fire ring with banked coals.
    ring = ellipse(16, 20, 5, 3)
    cv.fill(ring, "stone.m")
    cv.fill(ellipse(16, 20, 3, 1.6), "charcoal.d")
    cv.fill({(14, 20), (17, 19), (16, 21)}, "ember.m")
    cv.fill({(x, 17) for x in (12, 15, 18, 20)}, "stone.l")
    cv.fill({(x, 23) for x in (13, 16, 19)}, "stone.d")
    # Two bedrolls either side of the ring.
    cv.part(round_rect(3, 16, 9, 24), "moss")
    cv.part(rect(3, 16, 9, 18), "wool")
    cv.part(round_rect(23, 18, 29, 26), "leather")
    cv.part(rect(23, 18, 29, 20), "wool")
    if ally:
        _pennant(cv, 26, 8)
    cv.outline()
    return cv


def _stone_ring(cv, cx=8, cy=11):
    for dx, dy in ((-5, 0), (-4, 2), (-1, 3), (2, 3), (5, 2), (6, 0), (4, -2), (0, -2), (-3, -2)):
        cv.part({(cx + dx, cy + dy), (cx + dx + 1, cy + dy)}, "stone")


def embers_frames():
    out = []
    glow = [{(7, 9), (9, 10), (6, 11)}, {(8, 9), (10, 10), (7, 11), (9, 11)}, {(7, 10), (9, 9)}]
    for i, g in enumerate(glow):
        cv = Canvas(16, 16)
        _stone_ring(cv)
        cv.fill(rect(5, 9, 11, 11), "charcoal.d")
        cv.fill(g, "ember.m")
        cv.fill({(x, y) for x, y in g if (x + y + i) % 2 == 0}, "ember.l")
        cv.outline()
        out.append(cv)
    return out


def fire_unlit():
    cv = Canvas(16, 16)
    _stone_ring(cv)
    cv.part(thick(line(4, 9, 10, 6), 2), "wood")
    cv.part(thick(line(5, 6, 11, 9), 2), "wood")
    cv.outline()
    return cv


def fire_lit_frames():
    shapes = [
        [(8, 2), (7, 3), (8, 3), (7, 4), (8, 4), (9, 4), (6, 5), (7, 5), (8, 5), (9, 5), (10, 5)],
        [(9, 2), (8, 3), (9, 3), (8, 4), (9, 4), (7, 5), (8, 5), (9, 5), (10, 5), (6, 6)],
        [(7, 2), (7, 3), (8, 3), (7, 4), (8, 4), (9, 4), (6, 5), (7, 5), (8, 5), (9, 5), (10, 5)],
        [(8, 3), (9, 3), (8, 4), (9, 4), (7, 4), (7, 5), (8, 5), (9, 5), (10, 5), (11, 6)],
    ]
    out = []
    for sh in shapes:
        cv = Canvas(16, 16)
        _stone_ring(cv)
        cv.part(thick(line(4, 10, 10, 8), 2), "wood")
        flame = set(sh)
        for y in range(6, 10):
            flame |= {(x, y) for x in range(6 + (y - 6) // 2 * 0, 11 - (y - 6) // 3)}
        cv.fill(flame, "ember.d")
        inner = {(x, y) for x, y in flame if y >= 5 and 7 <= x <= 9}
        cv.fill(inner, "ember.m")
        cv.fill({(x, y) for x, y in inner if y >= 7 and x == 8}, "ember.l")
        cv.outline()
        out.append(cv)
    return out


def smoke_frames():
    out = []
    for i in range(4):
        cv = Canvas(16, 16)
        pts = []
        for y in range(14, 1, -1):
            wob = round(2 * __import__("math").sin((y + i * 3) / 2.2))
            pts.append((8 + wob, y))
        for n, (x, y) in enumerate(pts):
            col = "iron.l" if n < 4 else ("iron.m" if n < 9 else "iron.d")
            if (n + i) % 5 != 4:
                cv.put(x, y, col)
        out.append(cv)
    return out


def resting_frames():
    out = []
    for i in range(3):
        cv = Canvas(16, 16)
        moon = ellipse(7, 8, 4.4, 4.4) - ellipse(9, 7, 3.6, 3.6)
        cv.part(moon, "bone")
        for n, (x, y) in enumerate(((11, 11), (12, 6), (13, 2))):
            yy = y - (i + n) % 3
            cv.put(min(15, x + (i + n) % 2), max(0, yy), "bone.l" if n != 2 else "bone.m")
        cv.outline()
        out.append(cv)
    return out


def inn_rest():
    return icon([
        "..........K...",
        ".........KEK..",
        ".........KeK..",
        "..........H...",
        "..UUU.........",
        ".UUOO.rrrrrrr.",
        ".UOOO.rRRrrrx.",
        ".wwwwwwwwwwww.",
        ".wvvvvvvvvvvw.",
        ".w..........w.",
    ], ox=1, oy=3)


ALL_ICON_SAMPLES = (res_water, res_forage, st_bleeding, st_stunned, role_fighter,
                    role_healer, st_yield, st_dark)
