"""Battle idle sprites (frame 1), facing right, feet on frameHeight - 2."""
from pixkit import Canvas


def boot(c, mid, heel_x, y_bottom=62, length=7, height=5):
    c.poly([(heel_x, y_bottom - height), (heel_x + length - 3, y_bottom - height),
            (heel_x + length, y_bottom - 1), (heel_x + length, y_bottom), (heel_x, y_bottom)], mid)


def leg(c, mid, hip, knee, ankle, w1=5, w2=4):
    c.line([hip, knee], mid, width=w1)
    c.line([knee, ankle], mid, width=w2)


def warrior():
    c = Canvas(64, 64, band=2)
    mail, mail2 = c.m("mail", tex="mail"), c.m("mail", tex="mail")
    hose, hose2 = c.m("hair"), c.m("hair")
    bootb, bootf = c.m("leather"), c.m("leather")
    sur = c.m("oxblood", tex="cloth")
    belt = c.m("leather")
    skin = c.m("skin")
    helm = c.m("iron")
    beard = c.m("hair")
    steel = c.m("steel")
    grip = c.m("leather")
    brass = c.m("brass")
    rim = c.m("iron")
    wood = c.m("wood", tex="wood")
    armb = c.m("mail", tex="mail")
    # far leg and far arm (behind the body)
    leg(c, hose, (29, 44), (26, 52), (24, 58))
    boot(c, bootb, 21)
    c.line([(28, 29), (26, 37), (33, 44)], armb, width=4)
    # body: mail hauberk, then surcoat
    c.poly([(25, 26), (36, 26), (37, 37), (40, 50), (22, 50), (26, 37)], mail)
    leg(c, hose2, (33, 46), (37, 53), (38, 58))
    boot(c, bootf, 36)
    c.poly([(27, 27), (35, 27), (36, 37), (38, 49), (25, 49), (27, 37)], sur)
    c.line([(27, 49), (29, 37)], sur, tone=1)
    c.line([(35, 49), (33, 38)], sur, tone=1)
    c.rect(26, 36, 36, 37, belt)
    c.dot(31, 36, "O4")
    # sword low in the far hand, point forward and down
    c.line([(37, 45), (52, 53)], steel, width=2)
    c.line([(38, 44), (51, 51)], steel, tone=4)
    c.line([(35, 41), (37, 47)], brass, width=2)
    c.line([(33, 43), (35, 44)], grip, width=2)
    c.ellipse(31, 42, 33, 44, brass)
    c.ellipse(33, 41, 36, 45, c.m("skin"))
    # head: nasal helm, beard, mail aventail
    c.poly([(27, 23), (33, 23), (34, 28), (27, 28)], mail2)
    c.rect(31, 21, 35, 25, skin)
    c.dot(35, 22, "S1")  # nose
    c.poly([(31, 24), (35, 24), (35, 27), (32, 28), (31, 27)], beard)
    c.ellipse(27, 15, 35, 23, helm)
    c.rect(27, 20, 35, 21, helm, tone=1)
    c.line([(34, 20), (34, 23)], helm, tone=3)
    c.dot(33, 22, "W0")
    c.dot(29, 16, "C5")
    # round shield on the near arm, held forward
    c.ellipse(34, 26, 46, 43, rim)
    c.ellipse(35, 27, 45, 42, wood)
    c.line([(36, 34), (44, 34)], c.m("oxblood"), width=2)
    c.ellipse(38, 32, 41, 36, brass)
    c.dot(39, 33, "O6")
    return c


def cleric():
    c = Canvas(64, 64, band=2)
    mail, coif, armb = c.m("mail", tex="mail"), c.m("mail", tex="mail"), c.m("mail", tex="mail")
    hose, hose2 = c.m("hair"), c.m("hair")
    bootb, bootf = c.m("leather"), c.m("leather")
    tab = c.m("wool", tex="cloth")
    trim = c.m("brass")
    belt = c.m("leather")
    skin, hand = c.m("skin"), c.m("skin")
    haft = c.m("wood")
    mace = c.m("iron")
    shield, shrim = c.m("iron"), c.m("brass")
    holy = c.m("iron")
    leg(c, hose, (29, 44), (27, 52), (25, 58))
    boot(c, bootb, 22)
    # mace raised in the far hand, head above the shoulder
    c.line([(28, 29), (33, 36), (38, 33)], armb, width=4)
    c.line([(38, 38), (42, 22)], haft, width=2)
    c.poly([(40, 17), (45, 17), (46, 22), (44, 24), (41, 24), (39, 22)], mace)
    c.line([(42, 15), (43, 16)], mace)
    c.ellipse(36, 30, 40, 34, hand)
    c.poly([(25, 26), (36, 26), (37, 37), (39, 50), (23, 50), (26, 37)], mail)
    leg(c, hose2, (33, 46), (36, 53), (38, 58))
    boot(c, bootf, 36)
    c.poly([(27, 27), (35, 27), (35, 37), (37, 51), (26, 51), (27, 37)], tab)
    c.line([(27, 27), (27, 37), (26, 51)], trim)
    c.line([(35, 27), (35, 37), (37, 51)], trim)
    c.line([(31, 39), (31, 50)], tab, tone=1)
    c.rect(26, 36, 36, 37, belt)
    # iron holy symbol on a cord
    c.line([(30, 27), (32, 31)], c.m("leather"))
    c.poly([(32, 31), (33, 31), (33, 32), (34, 32), (34, 33), (33, 33), (33, 35), (32, 35), (32, 33), (31, 33), (31, 32), (32, 32)], holy)
    c.dot(32, 32, "C6")
    # head in a mail coif
    c.ellipse(26, 15, 35, 27, coif)
    c.poly([(31, 18), (35, 18), (36, 22), (35, 25), (31, 25)], skin)
    c.dot(36, 21, "S1")
    c.dot(34, 20, "W0")
    c.line([(32, 24), (34, 24)], skin, tone=0)
    # heater shield on the near arm, carried low
    c.poly([(33, 33), (43, 33), (43, 40), (38, 47), (33, 40)], shrim)
    c.poly([(34, 34), (42, 34), (42, 40), (38, 45), (34, 40)], shield)
    c.line([(38, 35), (38, 43)], shrim)
    c.line([(35, 38), (41, 38)], shrim)
    return c


def wizard():
    c = Canvas(64, 64, band=2)
    robe = c.m("slate", tex="cloth")
    robe2 = c.m("slate", tex="cloth")
    hood = c.m("slate")
    sleeve = c.m("slate")
    beard = c.m("beard")
    skin, hand = c.m("skin"), c.m("skin")
    staff = c.m("wood")
    stone = c.m("glow", outline=True)
    belt = c.m("pewter")
    bootf = c.m("leather")
    # staff planted ahead, gnarled at the top
    c.line([(43, 13), (42, 30), (43, 46), (42, 62)], staff, width=2)
    c.line([(43, 13), (40, 9), (41, 6)], staff, width=2)
    c.line([(43, 13), (46, 9), (45, 6)], staff, width=2)
    c.ellipse(41, 7, 45, 11, stone)
    c.dot(42, 8, "S6")
    # far sleeve behind
    c.line([(27, 29), (25, 38), (28, 44)], robe2, width=5)
    # long robe
    c.poly([(26, 26), (36, 26), (38, 40), (41, 61), (21, 61), (24, 40)], robe)
    for x0, x1 in ((28, 26), (33, 34), (37, 39)):
        c.line([(x0, 42), (x1, 60)], robe, tone=1)
    boot(c, bootf, 34, length=8, height=2)
    c.rect(26, 37, 37, 38, belt)
    c.dot(29, 38, "C6")
    # deep hood with the face in shadow and a long beard
    c.poly([(25, 20), (29, 14), (34, 14), (37, 19), (37, 27), (26, 29)], hood)
    c.poly([(31, 19), (36, 19), (37, 22), (36, 24), (31, 24)], skin, tone=1)
    c.dot(35, 20, "W0")
    c.dot(37, 21, "S0")
    c.poly([(31, 23), (37, 23), (37, 30), (35, 36), (33, 34), (31, 28)], beard)
    c.line([(34, 25), (35, 33)], beard, tone=1)
    # near arm, hand on the staff
    c.line([(32, 28), (35, 35), (40, 33)], sleeve, width=5)
    c.ellipse(40, 31, 43, 35, hand)
    return c


def rogue():
    c = Canvas(64, 64, band=2)
    cloak = c.m("charcoal", tex="cloth")
    jerk = c.m("leather")
    pants, pants2 = c.m("charcoal"), c.m("charcoal")
    bootb, bootf = c.m("leather"), c.m("leather")
    hood = c.m("charcoal")
    mask = c.m("plum")
    sash = c.m("plum")
    skin, hand1, hand2 = c.m("skin"), c.m("skin"), c.m("skin")
    blade1, blade2 = c.m("steel"), c.m("steel")
    armb, armf = c.m("leather"), c.m("charcoal")
    # crouched: knees bent, weight forward
    leg(c, pants, (28, 45), (22, 52), (23, 58), 5, 4)
    boot(c, bootb, 20)
    # far arm, blade reversed along the forearm
    c.line([(30, 31), (33, 38), (41, 39)], armb, width=4)
    c.line([(43, 37), (36, 45)], blade1, width=2)
    c.ellipse(40, 37, 43, 40, hand1)
    # cloak behind and lean torso
    c.poly([(26, 27), (32, 26), (31, 40), (27, 48), (21, 46), (22, 34)], cloak)
    c.poly([(28, 28), (36, 29), (37, 39), (34, 46), (27, 45), (27, 36)], jerk)
    c.rect(27, 41, 36, 42, sash)
    c.line([(26, 43), (24, 47)], sash)
    leg(c, pants2, (33, 46), (39, 50), (38, 58), 5, 4)
    boot(c, bootf, 36)
    # hood and mask
    c.poly([(28, 20), (32, 17), (37, 19), (39, 24), (37, 30), (29, 30), (27, 25)], hood)
    c.poly([(34, 22), (38, 22), (39, 25), (34, 25)], skin, tone=1)
    c.dot(37, 23, "W0")
    c.poly([(33, 25), (39, 25), (38, 28), (33, 28)], mask)
    # near arm low and forward, blade reversed
    c.line([(32, 30), (36, 38), (44, 42)], armf, width=4)
    c.line([(46, 40), (39, 49)], blade2, width=2)
    c.line([(45, 41), (40, 47)], blade2, tone=4)
    c.ellipse(43, 40, 46, 43, hand2)
    return c


def ranger():
    c = Canvas(64, 64, band=2)
    cloak = c.m("moss", tex="cloth")
    tunic = c.m("tan")
    pants, pants2 = c.m("leather"), c.m("leather")
    bootb, bootf = c.m("leather"), c.m("leather")
    hood = c.m("moss")
    skin, hand = c.m("skin"), c.m("skin")
    quiver = c.m("leather")
    fletch = c.m("wool")
    bow = c.m("wood")
    string = c.lit("W5", outline=False)
    arrow = c.m("paleWood")
    armb, armf = c.m("tan"), c.m("moss")
    belt = c.m("leather")
    knife = c.m("steel")
    # quiver on the back, fletchings over the shoulder
    c.line([(24, 22), (29, 40)], quiver, width=4)
    for x in (22, 24, 26):
        c.line([(x, 17), (x + 1, 21)], fletch, width=2)
    leg(c, pants, (29, 44), (26, 52), (24, 58))
    boot(c, bootb, 21)
    c.line([(28, 29), (31, 37), (36, 40)], armb, width=4)
    c.poly([(25, 26), (31, 25), (30, 40), (27, 52), (21, 51), (22, 36)], cloak)
    c.poly([(27, 27), (35, 27), (36, 38), (37, 46), (27, 46), (28, 37)], tunic)
    c.rect(27, 37, 36, 38, belt)
    c.line([(29, 39), (27, 45)], knife, width=1)
    leg(c, pants2, (33, 46), (37, 53), (38, 58))
    boot(c, bootf, 36)
    # hood up, face shadowed
    c.poly([(26, 21), (30, 15), (35, 15), (38, 20), (38, 27), (27, 28)], hood)
    c.poly([(32, 20), (37, 20), (38, 23), (37, 26), (32, 26)], skin, tone=2)
    c.dot(36, 22, "W0")
    c.dot(38, 23, "S1")
    c.line([(33, 26), (36, 26)], c.m("hair"))
    # bow held low in the near hand, arrow nocked
    c.line([(40, 19), (43, 25), (45, 37), (43, 49), (40, 55)], bow, width=2)
    c.line([(40, 19), (36, 38), (40, 55)], string)
    c.line([(36, 38), (51, 39)], arrow)
    c.poly([(51, 38), (53, 39), (51, 40)], c.m("iron"))
    c.line([(36, 37), (38, 37)], fletch)
    c.line([(32, 28), (37, 34), (43, 38)], armf, width=4)
    c.ellipse(42, 36, 45, 40, hand)
    return c


def witch():
    c = Canvas(64, 64, band=2)
    robe = c.m("heather", tex="cloth")
    shawl = c.m("ashmoss", tex="cloth")
    shawl2 = c.m("heather")
    hood = c.m("heather")
    skin, hand = c.m("skin"), c.m("skin")
    bone = c.m("bone")
    wand = c.m("wood")
    mist = c.m("mist", sep=False, outline=False)
    sleeve = c.m("heather")
    bootf = c.m("leather")
    # ragged robe to the ground
    c.poly([(26, 27), (35, 27), (38, 42), (41, 60), (21, 60), (23, 42)], robe)
    for x in range(21, 41, 3):
        c.poly([(x, 59), (x + 2, 59), (x + 1, 62)], robe)
    c.line([(29, 44), (27, 59)], robe, tone=1)
    c.line([(34, 44), (36, 59)], robe, tone=1)
    boot(c, bootf, 34, length=7, height=2)
    # layered shawls, tattered edges
    c.poly([(24, 27), (37, 27), (39, 37), (35, 44), (31, 39), (27, 45), (23, 38)], shawl)
    c.poly([(26, 25), (36, 25), (37, 31), (33, 35), (30, 31), (26, 33)], shawl2)
    # bone and herb charms on cords
    for x, y in ((27, 43), (33, 41), (37, 44)):
        c.line([(x, y - 3), (x, y)], c.lit("E2"))
        c.ellipse(x - 1, y, x + 1, y + 2, bone)
    c.dot(30, 42, "G4")
    c.dot(30, 43, "G3")
    # hood
    c.poly([(25, 22), (29, 15), (34, 15), (38, 21), (37, 27), (27, 28)], hood)
    c.poly([(32, 20), (37, 20), (38, 23), (36, 26), (32, 26)], skin, tone=1)
    c.dot(35, 21, "W0")
    c.dot(37, 23, "S0")
    c.line([(28, 18), (37, 20)], hood, tone=0)
    # near arm low, crooked wand, a curl of grave-mist
    c.line([(31, 29), (35, 37), (40, 41)], sleeve, width=4)
    c.ellipse(39, 39, 42, 42, hand)
    c.line([(42, 42), (44, 43), (45, 46), (48, 47)], wand)
    for x, y in ((48, 45), (49, 44), (50, 43), (51, 43), (52, 44), (52, 45), (51, 46),
                 (47, 42), (48, 41), (54, 41), (55, 40), (53, 38)):
        c.px(x, y, mist, tone=2)
    c.px(50, 42, mist, tone=3)
    return c


def ogre():
    import math
    c = Canvas(96, 96, band=2)
    body, belly = c.m("ogre"), c.m("ogre")
    arm1, arm2, fist = c.m("ogre"), c.m("ogre"), c.m("ogre")
    leg1, leg2 = c.m("ogre"), c.m("ogre")
    foot1, foot2 = c.m("ogre"), c.m("ogre")
    hide = c.m("hide", tex="fur")
    club = c.m("wood", tex="wood")
    head = c.m("ogre")
    tusk = c.m("bone")
    rope = c.m("leather")
    # far leg, far arm hanging to the knee
    leg(c, leg1, (47, 66), (42, 80), (42, 88), 12, 10)
    c.poly([(35, 87), (46, 87), (48, 94), (34, 94)], foot1)
    c.line([(42, 36), (34, 56), (36, 70)], arm1, width=11)
    c.ellipse(30, 66, 41, 78, arm1)
    c.line([(32, 72), (38, 72)], arm1, tone=1)
    # tree-trunk club over the shoulder, thick end up and back
    hx, hy, tx, ty = 70, 50, 38, 10
    L = math.hypot(tx - hx, ty - hy)
    ux, uy = (tx - hx) / L, (ty - hy) / L
    px_, py_ = -uy, ux
    pts = [(hx + px_ * 2.5, hy + py_ * 2.5), (tx + px_ * 8, ty + py_ * 8),
           (tx + ux * 4 + px_ * 4, ty + uy * 4 + py_ * 4), (tx + ux * 4 - px_ * 4, ty + uy * 4 - py_ * 4),
           (tx - px_ * 8, ty - py_ * 8), (hx - px_ * 2.5, hy - py_ * 2.5)]
    c.poly([(round(x), round(y)) for x, y in pts], club)
    c.ellipse(tx - 7, ty - 7, tx + 7, ty + 7, club)
    c.line([(46, 14), (40, 6)], club, width=3)  # broken branch stub
    for x, y in ((40, 18), (47, 20), (52, 30), (44, 11)):
        c.dot(x, y, "E0")
        c.dot(x + 1, y, "E3")
    # hunched body: back hump, heavy shoulders, sagging belly
    c.poly([(34, 42), (40, 28), (52, 21), (64, 22), (72, 32), (73, 52), (68, 66), (44, 68), (36, 58)], body)
    c.ellipse(48, 44, 73, 70, belly)
    c.line([(58, 40), (70, 44)], body, tone=1)
    c.line([(44, 30), (50, 36)], body, tone=1)
    c.poly([(42, 63), (70, 63), (71, 77), (62, 74), (55, 79), (49, 74), (41, 77)], hide)
    c.line([(42, 63), (70, 63)], rope, width=2)
    c.dot(56, 64, "W6")
    leg(c, leg2, (62, 70), (66, 80), (66, 88), 12, 10)
    c.poly([(60, 87), (71, 87), (75, 92), (75, 94), (60, 94)], foot2)
    c.dot(73, 93, "W2")
    c.dot(70, 93, "W2")
    # small head thrust forward, heavy brow and underbite
    c.poly([(60, 26), (67, 22), (74, 25), (77, 31), (77, 40), (72, 44), (63, 43), (59, 36)], head)
    c.line([(66, 29), (76, 30)], head, tone=0)
    c.line([(66, 28), (75, 28)], head, tone=3)
    c.dot(72, 31, "R5")
    c.dot(73, 31, "W0")
    c.dot(77, 33, "W3")
    c.line([(68, 39), (77, 39)], head, tone=0)
    c.poly([(73, 39), (75, 39), (74, 35)], tusk)
    c.poly([(69, 39), (71, 39), (70, 36)], tusk)
    c.poly([(60, 28), (63, 27), (62, 33)], head, tone=1)
    # near arm bent up, fist around the haft
    c.line([(58, 34), (64, 56)], arm2, width=11)
    c.line([(64, 56), (68, 50)], arm2, width=9)
    c.ellipse(64, 44, 75, 55, fist)
    c.line([(67, 47), (67, 52)], fist, tone=1)
    c.line([(70, 47), (70, 52)], fist, tone=1)
    for x, y in ((50, 30), (56, 52), (46, 58), (66, 60), (40, 48)):
        c.dot(x, y, "W2")
        c.dot(x + 1, y + 1, "E1")
    return c


def goblin():
    c = Canvas(64, 64, band=1)
    skin = c.m("goblin")
    arm1, arm2 = c.m("goblin"), c.m("goblin")
    leg1, leg2 = c.m("goblin"), c.m("goblin")
    hide = c.m("hide", tex="fur")
    head = c.m("goblin")
    ear = c.m("goblin")
    shaft = c.m("wood")
    tip = c.m("W4" and "iron")
    wrap = c.m("leather")
    # wiry, hunched, about 30 px tall
    leg(c, leg1, (28, 48), (24, 54), (25, 60), 3, 3)
    c.poly([(23, 59), (28, 59), (29, 62), (22, 62)], leg1)
    c.line([(29, 40), (27, 46), (34, 48)], arm1, width=3)
    c.poly([(27, 39), (34, 37), (37, 44), (35, 51), (27, 51), (25, 45)], skin)
    c.poly([(26, 43), (36, 43), (37, 52), (33, 50), (30, 53), (26, 51)], hide)
    leg(c, leg2, (32, 50), (36, 55), (35, 60), 3, 3)
    c.poly([(34, 59), (39, 59), (40, 62), (33, 62)], leg2)
    # spear held low and forward
    c.line([(16, 50), (50, 40)], shaft, width=2)
    c.poly([(50, 38), (56, 38), (52, 42), (49, 42)], tip)
    c.line([(47, 40), (49, 41)], wrap, width=2)
    # head jutting forward, long ears swept back
    c.poly([(32, 33), (37, 31), (41, 34), (42, 38), (38, 40), (33, 39)], head)
    c.poly([(30, 33), (33, 33), (33, 36), (26, 31)], ear)
    c.dot(38, 34, "O5")
    c.dot(39, 34, "W0")
    c.line([(38, 38), (41, 38)], head, tone=0)
    c.dot(40, 39, "W6")
    c.line([(33, 41), (38, 44), (42, 43)], arm2, width=3)
    c.ellipse(40, 41, 43, 44, arm2)
    return c


CLASSES = {"warrior": warrior, "rogue": rogue, "ranger": ranger,
           "cleric": cleric, "wizard": wizard, "witch": witch}
ORDER = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"]
