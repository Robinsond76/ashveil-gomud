"""Landmark overlays (art set S2): 32x32, transparent, outlined, drawn over a
terrain tile.  They replace the letter symbols on the map."""
import math

from kit import limb, poly
from pixels import Canvas, rect, line, ellipse, thick


def cv32():
    return Canvas(32, 32)


def hut(cv, x0, y0, w, h, wall, roof, door=True):
    cv.part(rect(x0, y0, x0 + w - 1, y0 + h - 1), wall)
    cv.part(poly([(x0 - 2, y0), (x0 + w + 1, y0), (x0 + w // 2, y0 - max(4, w // 3))]), roof)
    if door:
        cv.fill(rect(x0 + w // 2 - 1, y0 + h - 5, x0 + w // 2 + 1, y0 + h - 1), "leather.d")


def inn():
    cv = cv32()
    hut(cv, 5, 12, 22, 15, "wood", "oxblood")
    cv.fill(rect(9, 16, 12, 19), "ochre.l")
    cv.fill(rect(20, 16, 23, 19), "ochre.l")
    cv.fill(line(27, 13, 31, 13), "iron.m")  # sign bracket
    cv.part(rect(28, 14, 31, 18), "wood")
    cv.fill({(29, 16), (30, 16)}, "brass.m")  # an emblem, no lettering
    cv.outline()
    return cv


def bank():
    cv = cv32()
    cv.part(rect(5, 12, 26, 28), "stone")
    cv.part(rect(4, 9, 27, 12), "stone", flat="l")
    cv.fill(rect(10, 18, 21, 28), "iron.d")
    cv.part(ellipse(15.5, 16, 3.5, 3.5), "brass")  # coin emblem
    cv.fill({(15, 16), (16, 16)}, "brass.d")
    cv.fill(rect(7, 14, 8, 16), "charcoal.d")
    cv.outline()
    return cv


def shop():
    cv = cv32()
    cv.part(rect(6, 16, 25, 28), "wood", flat="m")
    for i, x in enumerate(range(5, 27, 4)):  # striped awning
        cv.part(rect(x, 10, x + 3, 16), "oxblood" if i % 2 else "wool")
    cv.part(rect(8, 22, 23, 24), "wood", flat="l")
    for x, c in ((10, "ochre.m"), (14, "moss.m"), (18, "plum.m"), (21, "bone.m")):
        cv.fill(rect(x, 20, x + 2, 21), c)
    cv.outline()
    return cv


def smithy():
    cv = cv32()
    hut(cv, 4, 14, 16, 14, "stone", "leather", door=False)
    cv.fill(rect(8, 20, 15, 27), "charcoal.d")
    cv.fill({(10, 25), (11, 25), (12, 25), (11, 24), (13, 26)}, "ember.m")
    cv.fill({(11, 23)}, "ember.l")
    cv.part(rect(22, 24, 29, 26), "iron")  # anvil
    cv.part(rect(24, 26, 27, 28), "iron", flat="d")
    cv.fill(rect(21, 23, 22, 24), "iron.l")
    cv.outline()
    return cv


def herbalist():
    cv = cv32()
    hut(cv, 6, 14, 18, 14, "wood", "moss")
    for x in (9, 13, 17, 21):  # drying herbs under the eave
        cv.fill(line(x, 14, x, 18), "moss.l")
        cv.fill({(x - 1, 18), (x + 1, 18)}, "moss.m")
    cv.part(rect(25, 22, 28, 28), "water", flat="m")  # bottle
    cv.fill(rect(26, 20, 27, 22), "bone.m")
    cv.outline()
    return cv


def trainer():
    cv = cv32()
    cv.part(rect(14, 16, 17, 28), "wood")  # post and a straw target
    cv.part(ellipse(15.5, 14, 6, 6), "ochre")
    cv.part(ellipse(15.5, 14, 3, 3), "oxblood", flat="m")
    cv.fill(line(4, 8, 14, 24), "steel.m")  # crossed practice swords
    cv.fill(line(27, 8, 17, 24), "steel.l")
    cv.fill({(3, 7), (4, 7), (28, 7), (27, 7)}, "wood.m")
    cv.outline()
    return cv


def temple():
    cv = cv32()
    cv.part(rect(8, 14, 23, 28), "wool", flat="m")
    cv.part(poly([(6, 14), (25, 14), (15, 6)]), "slate")
    cv.fill(rect(14, 2, 17, 6), "brass.m")  # bell turret
    cv.part(rect(13, 4, 18, 7), "brass")
    cv.fill({(15, 8), (16, 8)}, "iron.m")
    cv.fill(rect(14, 21, 17, 28), "leather.d")
    cv.fill(rect(10, 17, 11, 20), "slate.d")
    cv.fill(rect(20, 17, 21, 20), "slate.d")
    cv.outline()
    return cv


def shaman():
    cv = cv32()
    cv.part(poly([(4, 28), (22, 28), (13, 10)]), "leather")  # a hide tent
    cv.fill(line(13, 10, 13, 28), "leather.d")
    cv.fill(rect(11, 22, 14, 28), "charcoal.d")
    cv.part(rect(25, 12, 27, 28), "bone", flat="m")  # bone totem
    cv.part(ellipse(26, 10, 3, 3), "bone")
    cv.fill({(25, 10), (27, 10)}, "outline")
    cv.fill(rect(23, 17, 29, 17), "bone.d")
    cv.outline()
    return cv


def hermit():
    cv = cv32()
    cv.part(rect(7, 14, 21, 28), "wood", flat="d")
    cv.part(poly([(4, 15), (24, 15), (11, 6)]), "leather")
    cv.fill(rect(12, 21, 15, 28), "leather.d")
    cv.fill(line(24, 10, 24, 16), "iron.m")  # lantern on a post
    cv.part(rect(22, 16, 26, 21), "iron")
    cv.fill(rect(23, 17, 25, 20), "ember.m")
    cv.fill(rect(10, 17, 11, 18), "ochre.m")
    cv.outline()
    return cv


def gate():
    cv = cv32()
    cv.part(rect(2, 8, 9, 28), "stone")
    cv.part(rect(22, 8, 29, 28), "stone")
    cv.part(rect(2, 8, 29, 13), "stone")
    cv.fill(rect(10, 14, 21, 28), "charcoal.d")
    for x in range(11, 21, 3):  # open portcullis
        cv.fill(line(x, 14, x, 19), "iron.m")
    for x in range(2, 30, 6):
        cv.fill(rect(x, 5, x + 2, 8), "stone.m")
    cv.outline()
    return cv


def wall():
    cv = cv32()
    cv.part(rect(0, 14, 31, 28), "stone")
    for x in range(0, 32, 8):
        cv.part(rect(x, 9, x + 4, 14), "stone")
    for y in (18, 23):
        cv.fill(line(0, y, 31, y), "stone.d")
    cv.outline()
    return cv


def bridge():
    cv = cv32()
    for x in range(2, 30, 3):
        cv.part(rect(x, 8, x + 2, 24), "wood", flat="m" if (x // 3) % 2 else "l")
    cv.fill(line(2, 8, 29, 8), "wood.d")
    cv.fill(line(2, 24, 29, 24), "wood.d")
    for x in (3, 15, 27):  # rail posts
        cv.part(rect(x, 5, x + 1, 9), "wood", flat="d")
        cv.part(rect(x, 23, x + 1, 27), "wood", flat="d")
    cv.outline()
    return cv


def keep():
    cv = cv32()
    cv.part(rect(8, 10, 23, 28), "stone")
    for x in range(8, 24, 5):
        cv.part(rect(x, 6, x + 2, 10), "stone")
    cv.fill(rect(14, 20, 17, 28), "charcoal.d")
    cv.fill(rect(11, 14, 12, 17), "charcoal.d")
    cv.fill(rect(19, 14, 20, 17), "charcoal.d")
    cv.fill(line(15, 1, 15, 6), "wood.m")  # banner
    cv.part(rect(16, 1, 21, 4), "oxblood")
    cv.outline()
    return cv


def throne():
    cv = cv32()
    cv.part(rect(6, 6, 25, 9), "oxblood")  # canopy
    cv.fill({(x, 10) for x in range(6, 26, 2)}, "oxblood.d")
    cv.fill(rect(6, 6, 7, 27), "wood.m")
    cv.fill(rect(24, 6, 25, 27), "wood.m")
    cv.part(rect(12, 12, 19, 24), "wood")  # the throne
    cv.part(rect(10, 18, 21, 22), "brass", flat="m")
    cv.fill(rect(10, 25, 21, 27), "oxblood.m")
    cv.outline()
    return cv


def townsquare():
    cv = cv32()
    cv.part(ellipse(15.5, 24, 14, 5), "stone", flat="m")
    cv.part(ellipse(15.5, 22, 7, 3), "stone")
    cv.fill(ellipse(15.5, 22, 5, 2), "water.m")
    cv.part(rect(14, 12, 17, 21), "stone")
    cv.part(ellipse(15.5, 11, 4, 2), "stone")
    cv.fill({(15, 8), (16, 8), (15, 9), (16, 9), (13, 10), (18, 10)}, "water.l")
    cv.outline()
    return cv


def village():
    cv = cv32()
    for x, y, w, wall_, roof in ((2, 14, 12, "wood", "ochre"), (17, 11, 12, "stone", "leather"),
                                 (8, 21, 14, "wood", "oxblood")):
        hut(cv, x, y, w, 7 if y > 20 else 8, wall_, roof)
    cv.outline()
    return cv


def caravan():
    cv = cv32()
    cv.part(rect(5, 14, 24, 23), "wool")  # canvas cover over a wagon
    cv.part(ellipse(14.5, 14, 10, 6), "wool")
    cv.fill(line(8, 12, 8, 23), "wool.d")
    cv.fill(line(14, 8, 14, 23), "wool.d")
    cv.fill(line(20, 12, 20, 23), "wool.d")
    cv.part(rect(4, 23, 25, 25), "wood")
    for x in (8, 21):
        cv.part(ellipse(x, 25, 3, 3), "wood")
        cv.fill({(x, 25)}, "iron.m")
    cv.fill(line(25, 24, 30, 22), "wood.m")  # drawbar
    cv.outline()
    return cv


def lake_house():
    cv = cv32()
    cv.fill(rect(0, 24, 31, 31), "water.m")
    for x in range(1, 31, 5):
        cv.put(x, 27, "water.l")
    for x in (6, 12, 18, 24):  # stilts
        cv.part(rect(x, 18, x + 1, 27), "wood", flat="d")
    cv.part(rect(4, 17, 27, 18), "wood")
    hut(cv, 7, 9, 17, 8, "wood", "leather")
    cv.outline()
    return cv


def cave_mouth():
    cv = cv32()
    cv.part(poly([(1, 28), (6, 8), (14, 3), (24, 6), (30, 28)]), "stone")
    cv.fill(poly([(8, 28), (10, 14), (15, 10), (21, 14), (24, 28)]), "charcoal.d")
    cv.fill(poly([(12, 28), (13, 18), (16, 15), (19, 19), (20, 28)]), "outline")
    cv.fill({(7, 12), (22, 9), (26, 16)}, "stone.l")
    cv.outline()
    return cv


def dungeon_stair():
    cv = cv32()
    cv.part(rect(3, 4, 28, 28), "stone")
    cv.fill(rect(7, 8, 24, 24), "charcoal.d")
    for i, y in enumerate(range(9, 24, 3)):  # steps descending into darkness
        w = 16 - i * 2
        cv.fill(rect(16 - w // 2, y, 15 + w // 2, y + 1), ["stone.l", "stone.m", "stone.m", "stone.d", "stone.d"][min(i, 4)])
    cv.outline()
    return cv


def obelisk():
    cv = cv32()
    cv.part(poly([(12, 28), (20, 28), (18, 4), (14, 4)]), "stone")
    cv.part(poly([(14, 4), (18, 4), (16, 1)]), "stone", flat="l")
    cv.part(rect(9, 25, 22, 28), "stone", flat="d")
    for y in (8, 12, 16, 20):  # rune-carved marks (abstract glyphs)
        cv.fill({(15, y), (16, y), (15, y + 1), (14, y + 2)}, "ember.m" if y == 12 else "stone.d")
    cv.outline()
    return cv


def pond():
    cv = cv32()
    cv.part(ellipse(15.5, 20, 12, 7), "water", flat="m")
    cv.fill(ellipse(13, 19, 4, 2), "water.l")
    for x, h in ((3, 8), (6, 10), (26, 9), (28, 7), (24, 11)):  # reed fringe
        cv.fill(line(x, 22, x, 22 - h), "grass.m")
        cv.fill({(x, 22 - h)}, "ochre.m")
    cv.outline()
    return cv


def rocks():
    cv = cv32()
    cv.part(ellipse(12, 21, 8, 6), "stone")
    cv.part(ellipse(22, 22, 6, 5), "stone")
    cv.part(ellipse(16, 15, 6, 5), "stone")
    cv.fill({(9, 17), (10, 17), (14, 12), (13, 12)}, "stone.l")
    cv.outline()
    return cv


def desert_ruin():
    cv = cv32()
    cv.part(rect(11, 9, 18, 28), "bone")
    cv.fill(rect(9, 7, 20, 9), "bone.m")
    for y in (13, 17, 21):
        cv.fill(line(12, y, 17, y), "bone.d")
    cv.fill({(10, 6), (14, 6), (18, 5)}, "bone.d")
    cv.part(ellipse(24, 26, 4, 2), "bone")  # a fallen drum
    cv.fill(rect(2, 26, 8, 28), "ochre.l")  # drifted sand
    cv.outline()
    return cv


def alts():
    cv = cv32()
    cv.part(rect(14, 14, 17, 28), "wood")  # lectern
    cv.part(rect(9, 26, 22, 28), "wood", flat="d")
    cv.part(poly([(5, 14), (26, 14), (24, 9), (7, 9)]), "wood", flat="l")
    cv.part(rect(7, 8, 15, 14), "wool")  # an open book
    cv.part(rect(16, 8, 24, 14), "wool", flat="l")
    cv.fill(line(15, 8, 15, 14), "wool.d")
    for y in (10, 12):
        cv.fill(line(9, y, 13, y), "wool.d")
        cv.fill(line(18, y, 22, y), "wool.d")
    cv.outline()
    return cv


def landmark():
    cv = cv32()
    cv.part(poly([(10, 28), (22, 28), (20, 8), (13, 6)]), "stone")
    cv.part(rect(7, 25, 24, 28), "stone", flat="d")
    cv.fill(rect(14, 10, 17, 11), "stone.l")
    cv.outline()
    return cv


def boss_lair():
    cv = cv32()
    cv.part(rect(15, 8, 16, 28), "wood", flat="d")  # a stake
    cv.part(ellipse(15.5, 8, 6, 5.5), "bone")  # a cracked skull
    cv.fill({(13, 8), (14, 8), (17, 8), (18, 8), (13, 9), (18, 9)}, "outline")
    cv.fill(line(15, 3, 17, 8), "bone.d")
    cv.part(rect(12, 12, 19, 14), "bone", flat="d")
    cv.fill({(14, 13), (16, 13), (18, 13)}, "outline")
    cv.fill(rect(10, 26, 21, 28), "oxblood.d")
    cv.outline()
    return cv


LANDMARKS = {
    "inn": inn, "bank": bank, "shop": shop, "smithy": smithy, "herbalist": herbalist, "trainer": trainer,
    "temple": temple, "shaman": shaman, "hermit": hermit, "gate": gate, "wall": wall, "bridge": bridge,
    "keep": keep, "throne": throne, "townsquare": townsquare, "village": village, "caravan": caravan,
    "lake-house": lake_house, "cave-mouth": cave_mouth, "dungeon-stair": dungeon_stair,
    "obelisk": obelisk, "pond": pond, "rocks": rocks, "desert-ruin": desert_ruin, "alts": alts,
    "landmark": landmark, "boss-lair": boss_lair,
}
