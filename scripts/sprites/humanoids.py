"""Humanoid enemies for battle (art set S3), drawn on the class rig.

Each enemy is data (materials, head, weapon, shield) over the same side-on
64x64 rig the player classes use, so the formation reads as one world.
Everything faces right; the client mirrors enemies.  Four idle frames: the
body settles and rises, weapons and cloth sway a pixel.
"""
import figures
from figures import Rig, BATTLE, Size, CLASSES, _robe
from kit import BREATH, SWAY, limb, recolor, shade_pixels
from pixels import Canvas, rect, round_rect, line, thick, ellipse


def size(**kw):
    return Size(**{**BATTLE.__dict__, **kw})


# -- weapons: drawn from the near hand, facing right ------------------------------

def w_sword(r, t, blade="steel", len_=13, grip="leather"):
    hx, hy = r.hand_r
    dy = SWAY[t]
    r.cv.part(thick(line(hx, hy, hx + len_, hy + 7 + dy), 1), blade, flat="m")
    r.cv.fill({(hx + len_, hy + 7 + dy)}, f"{blade}.l")
    r.cv.part(rect(hx - 2, hy - 1, hx - 1, hy), grip, flat="d")
    r.cv.fill({(hx, hy - 2), (hx + 1, hy - 1)}, "brass.m")


def w_notched(r, t):
    hx, hy = r.hand_r
    dy = SWAY[t]
    pts = line(hx, hy, hx + 13, hy + 7 + dy)
    r.cv.part(thick(pts, 1), "iron", flat="l")
    for i, (x, y) in enumerate(sorted(pts)):
        if i in (5, 8):
            r.cv.put(x, y, "outline")
    r.cv.fill({(hx, hy - 2), (hx + 1, hy - 1)}, "ochre.d")


def w_spear(r, t, head="steel", tall=24):
    hx, hy = r.hand_r
    dx = SWAY[t]
    shaft = line(hx - 3, hy + 9, hx + 9 + dx, hy - tall)
    r.cv.part(thick(shaft, 1), "wood", flat="m")
    tx, ty = hx + 9 + dx, hy - tall
    r.cv.part(thick(line(tx, ty + 6, tx + 1, ty - 3), 2), head, flat="l")
    r.cv.fill({(tx + 1, ty - 4)}, f"{head}.l")


def w_halberd(r, t):
    hx, hy = r.hand_r
    dx = SWAY[t]
    shaft = line(hx - 3, hy + 9, hx + 8 + dx, hy - 26)
    r.cv.part(thick(shaft, 1), "wood", flat="m")
    tx, ty = hx + 8 + dx, hy - 26
    r.cv.part(thick(line(tx, ty + 6, tx + 1, ty - 4), 2), "steel", flat="l")
    r.cv.part({(tx + 3, ty + 1), (tx + 4, ty + 1), (tx + 4, ty + 2), (tx + 5, ty + 2),
               (tx + 3, ty + 3), (tx + 4, ty + 3), (tx + 5, ty + 3), (tx + 4, ty + 4)},
              "steel", flat="m")


def w_club(r, t, ramp="wood", big=False):
    hx, hy = r.hand_r
    dy = SWAY[t]
    r.cv.part(thick(line(hx, hy, hx + 7, hy - 13 + dy), 2), ramp, flat="m")
    r.cv.part(ellipse(hx + 8, hy - 15 + dy, 3 if big else 2.4, 4 if big else 3.4), ramp)
    if big:
        r.cv.fill({(hx + 8, hy - 17 + dy), (hx + 10, hy - 14 + dy)}, "iron.l")


def w_axe(r, t):
    hx, hy = r.hand_r
    dy = SWAY[t]
    r.cv.part(thick(line(hx, hy, hx + 8, hy - 12 + dy), 1), "wood", flat="m")
    r.cv.part({(hx + 8, hy - 15 + dy), (hx + 9, hy - 15 + dy), (hx + 10, hy - 14 + dy),
               (hx + 8, hy - 14 + dy), (hx + 9, hy - 14 + dy), (hx + 10, hy - 13 + dy),
               (hx + 11, hy - 13 + dy), (hx + 9, hy - 13 + dy), (hx + 10, hy - 12 + dy),
               (hx + 11, hy - 12 + dy)}, "steel", flat="m")
    r.cv.fill({(hx + 11, hy - 13 + dy)}, "steel.l")


def w_dagger(r, t, color="steel"):
    hx, hy = r.hand_r
    dy = SWAY[t]
    r.cv.part(line(hx, hy - 1, hx + 6, hy + 3 + dy), color, flat="m")
    r.cv.fill({(hx + 6, hy + 3 + dy)}, f"{color}.l")
    r.cv.part(rect(hx - 1, hy - 2, hx, hy), "leather", flat="d")


def w_twin(r, t):
    """Reverse-grip twin blades, as the rogue holds them."""
    hx, hy = r.hand_r
    r.cv.part(line(hx, hy - 1, hx - 7, hy - 6), "steel", flat="l")
    r.cv.part(rect(hx, hy - 2, hx + 1, hy - 1), "leather", flat="d")
    r.cv.part(line(r.tx0 + 1, r.yhip - 2, r.tx0 - 5, r.yhip + 4 + SWAY[t]), "steel", flat="m")


def w_bow(r, t):
    hx, hy = r.hand_r
    bow = {(hx + 6 - int(abs(y - hy + 3) * 0.28), y) for y in range(hy - 15, hy + 10)}
    r.cv.part(bow, "wood", flat="m")
    r.cv.part(line(hx + 1, hy - 15, hx + 1, hy + 9), "bone", flat="d")
    r.cv.part(line(hx + 1, hy - 3, hx + 9, hy - 2 + SWAY[t]), "bone", flat="m")


def w_staff(r, t, top="bone"):
    hx, hy = r.hand_r
    sx = hx + 1
    r.cv.part(thick({(sx + (1 if (y // 6) % 2 else 0), y) for y in range(r.top - 2, r.fy)}, 2),
              "wood", flat="m")
    return sx, r.top - 2


def w_board(r, t, ramp="wood"):
    """Round board shield, seen edge-on at the near forearm."""
    hx, hy = r.hand_r
    sx, sy = hx + 1, hy
    sh = ellipse(sx, sy, 2.6, 8)
    r.cv.part(sh, ramp)
    r.cv.part({(sx - 1, y) for y in range(sy - 7, sy + 8)}, ramp, flat="l")
    r.cv.fill({(sx, sy - 1), (sx, sy), (sx, sy + 1)}, "iron.m")


def w_tower(r, t, ramp="iron"):
    hx, hy = r.hand_r
    sx = hx + 1
    sh = rect(sx - 1, hy - 14, sx + 3, hy + 10)
    r.cv.part(sh, ramp)
    r.cv.part({(sx - 1, y) for y in range(hy - 14, hy + 11)}, "steel", flat="m")
    r.cv.part({(x, hy - 14) for x in range(sx - 1, sx + 4)}, "steel", flat="m")
    for y in (hy - 8, hy + 2):
        r.cv.fill({(sx + 1, y), (sx + 2, y), (sx + 1, y + 1)}, "ochre.d")


def w_kite(r, t):
    hx, hy = r.hand_r
    sx = hx + 1
    sh = rect(sx - 1, hy - 8, sx + 2, hy + 7) | {(sx, hy + 8), (sx + 1, hy + 8), (sx, hy + 9)}
    r.cv.part(sh, "oxblood")
    r.cv.part({(sx - 1, y) for y in range(hy - 8, hy + 8)}, "brass", flat="m")
    r.cv.fill({(sx + 1, hy - 2), (sx + 1, hy - 1), (sx + 1, hy)}, "brass.l")


WEAPONS = {
    "sword": w_sword, "notched": w_notched, "spear": w_spear, "halberd": w_halberd,
    "club": w_club, "bigclub": lambda r, t: w_club(r, t, "wood", True), "axe": w_axe,
    "dagger": w_dagger, "twin": w_twin, "bow": w_bow,
    "icesword": lambda r, t: w_sword(r, t, "water", 17, "steel"),
    "bonetool": lambda r, t: w_club(r, t, "bone"),
    "toysword": lambda r, t: w_sword(r, t, "wood", 12, "ochre"),
    "bonestaff": w_staff, "staff": w_staff,
}
SHIELDS = {"board": w_board, "tower": w_tower, "kite": w_kite,
           "round": lambda r, t: w_board(r, t, "wood")}


# -- the generic enemy humanoid ----------------------------------------------------

def draw_enemy(r, c, t):
    cv = r.cv
    if "robe" in r.spec:
        _robe(r, r.spec["robe"], c.get("trim", r.spec["robe"] + ".d"), flare=c.get("flare", 0.2))
    else:
        r.legs()
        r.torso(r.spec["torso"])
    if c.get("tabard"):
        hem = r.yhip + r.px(3)
        tab = rect(r.tx0, r.ysh + r.px(2), r.tx1, hem)
        cv.part(tab, c["tabard"])
        if c.get("tabard_trim"):
            cv.part({(x, hem) for x, _ in tab}, c["tabard_trim"], flat="m")
    if c.get("ribs"):
        for y in range(r.ysh + 3, r.yhip - 2, 3):
            cv.part({(x, y) for x in range(r.tx0 + 1, r.tx1)}, "charcoal", flat="m")
        cv.part(rect(r.cx - 1, r.ysh + 1, r.cx - 1, r.yhip - 3), "bone", flat="d")
    if c.get("straps"):
        cv.fill(line(r.tx0, r.ysh + 1, r.tx1, r.yhip - 2), c["straps"] + ".m")
    if c.get("rust"):
        for dx, dy in c["rust"]:
            cv.put(r.tx0 + dx, r.ysh + dy, "ochre.d")
    if c.get("belt"):
        cv.part({(x, r.yhip - r.px(1)) for x in range(r.tx0, r.tx1 + 1)}, c["belt"], flat="m")
    r.arms(r.spec["sleeve"], glove=r.skin if not c.get("glove") else c["glove"])
    r.head(**c.get("head", {}))
    if c.get("rag"):
        cv.part(rect(r.tx0 - 1, r.yhip, r.tx0 + 2, r.yhip + r.px(5)), c["rag"])
    shield = c.get("shield")
    wp = c.get("weapon")
    if shield:
        SHIELDS[shield](r, t)
    out = WEAPONS[wp](r, t) if wp else None
    if c.get("extra"):
        c["extra"](r, t, out)


def humanoid(name, c, S=None):
    S = S or BATTLE
    CLASSES[name] = dict(trousers=c.get("trousers", "leather"), boots=c.get("boots", "leather"),
                         torso=c.get("torso", "leather"), sleeve=c.get("sleeve", c.get("torso", "leather")),
                         **({"robe": c["robe"]} if "robe" in c else {}),
                         **({"skin": c["skin"]} if "skin" in c else {}),
                         wide=c.get("wide", 0))

    def draw(t):
        pose = dict(bob=c.get("bob", 0) + BREATH[t], stride=c.get("stride", (-3, 3)))
        r = Rig(name, "side", S, pose)
        draw_enemy(r, c, t)
        r.cv.outline()
        return r.cv
    return draw


# -- extras ---------------------------------------------------------------------------

def x_fetish(r, t, _):
    """Bones and a brass charm hung at the belt."""
    cv = r.cv
    for i, (dx, c) in enumerate(((0, "bone.l"), (2, "brass.m"), (4, "bone.m"))):
        cv.part(line(r.tx0 + dx, r.yhip, r.tx0 + dx, r.yhip + 4 + (i % 2) + SWAY[t]), c, flat="m")


def x_censer(r, t, _):
    hx, hy = r.hand_r
    sx = hx + 3 + (1 if t in (1, 2) else 0)
    cv = r.cv
    cv.part(line(hx, hy, sx, hy + 9), "iron", flat="l")
    cv.part(ellipse(sx, hy + 11, 2.4, 2), "brass")
    for i in range(7):  # grave smoke curls up from the bowl
        wob = (1 if (i + t) % 3 else -1) * (i // 3)
        cv.put(sx + wob, hy + 8 - i, "ashmoss.l" if i % 2 else "bone.d")


def x_satchel(r, t, _):
    cv = r.cv
    bx = r.tx0 - 3
    cv.part(rect(bx, r.yhip - 2, bx + 5, r.yhip + 5), "leather")
    for i, x in enumerate((bx + 1, bx + 3, bx + 4)):
        cv.part(line(x, r.yhip - 3 - i % 2, x, r.yhip - 1), "bone", flat="l")
    cv.put(bx + 2, r.yhip + 1, "brass.m")


def x_bandage(r, t, _):
    hx, hy = r.hand_r
    r.cv.part(ellipse(hx + 1, hy - 2, 2, 2), "bone", flat="l")
    r.cv.fill({(hx + 1, hy - 2)}, "oxblood.m")


def x_plume(r, t, _):
    cv = r.cv
    px = r.cx - 4
    py = r.top - 2
    for i in range(5):
        cv.put(px - i, py - (1 if i in (1, 3) else 0) + SWAY[t] * (i > 2), "oxblood.m" if i % 2 else "oxblood.l")


def x_horns(r, t, _):
    r.cv.fill({(r.cx - 1, r.top - 1), (r.cx - 2, r.top - 2), (r.cx + 1, r.top - 1), (r.cx + 2, r.top - 2)},
              "wood.m")


def x_eyes_ember(r, t, _):
    r.cv.put(r.cx + 2, r.top + 3, "ember.l")


def x_straw(r, t, _):
    """Straw poking from the sleeves, collar and boots."""
    cv = r.cv
    for dx, dy in ((r.tx1 + 2, r.ysh + 11), (r.tx1 + 3, r.ysh + 12), (r.tx0 - 1, r.yhip),
                   (r.tx0, r.yhip + 1), (r.cx, r.top + 6), (r.cx + 2, r.top + 7)):
        cv.put(dx, dy, "ochre.l")


def x_ironbar(r, t, _):
    pass


H = {}  # id -> draw(t)

# Undead ------------------------------------------------------------------------------
H["skeleton"] = humanoid("e-skeleton", dict(
    skin="bone", trousers="bone", boots="bone", torso="bone", sleeve="bone", ribs=True,
    glove="bone", weapon="notched", rag="leather", bob=0,
    head=dict()), size(sw=10, lw=3, aw=2, hw=7, hh=7))
H["bone-warden"] = humanoid("e-bone-warden", dict(
    skin="bone", trousers="bone", boots="iron", torso="iron", sleeve="iron", glove="bone",
    rust=[(1, 5), (4, 8), (7, 4), (3, 12)], head=dict(helm="iron"), shield="tower",
    weapon="notched", belt="leather"), size(sw=12, lw=4, aw=3))
H["bonecrafter"] = humanoid("e-bonecrafter", dict(
    skin="bone", robe="leather", torso="leather", sleeve="leather", glove="bone", trim="leather.d",
    head=dict(hood="leather"), weapon="bonetool", extra=x_fetish, flare=0.12, bob=1),
    size(body_h=42, sw=10, hh=7))
H["acolyte-dark"] = humanoid("e-acolyte-dark", dict(
    robe="charcoal", trim="oxblood.d", torso="charcoal", sleeve="charcoal", glove="skin",
    head=dict(hood="charcoal", mask="oxblood"), weapon="dagger", flare=0.16))
H["grave-chanter"] = humanoid("e-grave-chanter", dict(
    robe="stone", trim="stone.d", torso="stone", sleeve="stone", head=dict(hood="stone"),
    extra=x_censer, flare=0.2))
# Bandits ----------------------------------------------------------------------------
H["brigand"] = humanoid("e-brigand", dict(
    torso="leather", sleeve="leather", trousers="wool", boots="leather", belt="iron",
    head=dict(hood="leather"), weapon="axe", straps="iron"))
H["ruffian"] = humanoid("e-ruffian", dict(
    torso="wool", sleeve="wool", trousers="leather", boots="leather", belt="leather",
    head=dict(hair="leather"), weapon="club"), size(sw=12))
H["ruffian-enforcer"] = humanoid("e-ruffian-enforcer", dict(
    torso="wool", sleeve="leather", trousers="leather", boots="leather", belt="iron", wide=2,
    straps="leather", head=dict(hair="charcoal"), weapon="bigclub", glove="skin"),
    size(body_h=49, sw=15, hw=8, hh=8, lw=6, aw=4))
H["poacher"] = humanoid("e-poacher", dict(
    torso="ochre", sleeve="ochre", trousers="leather", boots="leather", belt="leather",
    head=dict(hood="leather"), weapon="bow"))
H["poacher-shieldman"] = humanoid("e-poacher-shieldman", dict(
    torso="leather", sleeve="leather", trousers="wool", boots="leather", belt="iron",
    head=dict(helm="iron"), weapon="spear", shield="board"))
H["bonesetter"] = humanoid("e-bonesetter", dict(
    torso="wool", sleeve="wool", trousers="leather", boots="leather", belt="leather", tabard="bone",
    head=dict(hair="leather"), extra=lambda r, t, _: (x_satchel(r, t, _), x_bandage(r, t, _))))
# Shadow guild -----------------------------------------------------------------------
H["shadow-trainee"] = humanoid("e-shadow-trainee", dict(
    torso="stone", sleeve="stone", trousers="charcoal", boots="charcoal", belt="leather",
    head=dict(hood="stone", mask="charcoal"), weapon="dagger"))
# Town guard ---------------------------------------------------------------------------
H["guard"] = humanoid("e-guard", dict(
    torso="iron", sleeve="iron", trousers="leather", boots="leather", belt="leather",
    tabard="slate", tabard_trim="brass", head=dict(helm="iron"), weapon="spear", shield="round"))
H["guard-captain"] = humanoid("e-guard-captain", dict(
    torso="iron", sleeve="iron", trousers="leather", boots="leather", belt="brass",
    tabard="oxblood", tabard_trim="brass", head=dict(helm="steel"), weapon="sword", shield="kite",
    extra=x_plume), size(sw=14))
# Ice -------------------------------------------------------------------------------------
H["ice-warrior"] = humanoid("e-ice-warrior", dict(
    skin="steel", torso="steel", sleeve="water", trousers="water", boots="steel", glove="steel",
    head=dict(helm="steel"), weapon="icesword", extra=x_eyes_ember), size(sw=13, wide=0))
# Training -------------------------------------------------------------------------------------
H["straw-footman"] = humanoid("e-straw-footman", dict(
    skin="ochre", torso="ochre", sleeve="ochre", trousers="ochre", boots="leather", glove="ochre",
    belt="leather", head=dict(hood="wool"), weapon="toysword", extra=x_straw), size(sw=11))
H["straw-archer"] = humanoid("e-straw-archer", dict(
    skin="ochre", torso="ochre", sleeve="ochre", trousers="ochre", boots="leather", glove="ochre",
    belt="leather", head=dict(hood="wool"), weapon="bow", extra=x_straw), size(sw=11))


# -- goblins: small, sinewy, feral ---------------------------------------------------------

GOBLIN = size(body_h=36, hh=8, T=11, L=14, hw=8, sw=9, lw=3, boot=4, arm_len=11, aw=3)


def goblin_head(r):
    cv = r.cv
    r.head()
    x0, y0 = r.cx - r.S.hw // 2, r.top
    hw, hh = r.S.hw, r.S.hh
    eye_y = y0 + round(hh * 0.5)
    # long swept-back ear, hooked nose, sharp teeth, heavy brow
    cv.part({(x0 + 1, y0 + 2), (x0, y0 + 1), (x0 - 1, y0), (x0 - 2, y0 - 1), (x0 - 3, y0 - 2),
             (x0 - 2, y0), (x0 - 1, y0 + 1), (x0 - 2, y0 + 1), (x0 - 1, y0 + 2), (x0, y0 + 2),
             (x0, y0 + 3), (x0 + 1, y0 + 3), (x0 - 3, y0 - 1), (x0 - 4, y0 - 3)}, "ashmoss")
    cv.part({(x0 + hw, eye_y + 1), (x0 + hw + 1, eye_y + 1), (x0 + hw + 1, eye_y + 2),
             (x0 + hw, eye_y + 2)}, "ashmoss", flat="d")
    cv.fill({(x0 + hw - 4, eye_y - 1), (x0 + hw - 3, eye_y - 1), (x0 + hw - 2, eye_y - 1)}, "ashmoss.d")
    cv.put(x0 + hw - 2, eye_y, "ember.m")
    cv.fill({(x0 + hw - 4, y0 + hh - 2), (x0 + hw - 2, y0 + hh - 2)}, "bone.l")


def draw_goblin(r, c, t):
    cv = r.cv
    hunch = c.get("hunch", 0)
    r.legs()
    r.torso("leather")
    # ragged hem on the tunic
    for x in range(r.tx0, r.tx1 + 1, 2):
        cv.put(x, r.yhip, "leather.d")
        cv.put(x, r.yhip + 1, "leather.d")
    cv.part({(x, r.yhip - 2) for x in range(r.tx0, r.tx1 + 1)}, "wood", flat="d")
    r.arms("ashmoss", glove="ashmoss")
    goblin_head(r)
    c.get("gear", lambda r, t: None)(r, t)


def g_spear(r, t):
    w_spear(r, t, "iron", 20)


def g_hexer(r, t):
    hx, hy = r.hand_r
    cv = r.cv
    sx, ty = w_staff(r, t)
    cv.part({(sx - 1, ty), (sx, ty - 1), (sx + 1, ty - 1), (sx + 2, ty), (sx, ty), (sx + 1, ty)}, "bone", flat="l")
    cv.fill({(sx, ty + 1), (sx + 1, ty + 1)}, "outline")
    for i, dx in enumerate((-2, 0, 2)):  # fetishes on cords
        cv.fill(line(sx + dx, ty + 2, sx + dx, ty + 5 + (i + t) % 2), "bone.m")
    cv.put(sx + 1, ty - 2, "moss.l" if t % 2 else "ashmoss.l")  # a dim glimmer
    cv.fill({(r.tx0 + 1, r.ysh + 3), (r.tx0 + 3, r.ysh + 5)}, "bone.l")


def g_shaman(r, t):
    """A herb-healer: a staff crowned with bundled moss, a ragged moss mantle,
    bone charms and a strung herb bunch at the belt."""
    cv = r.cv
    sx, ty = w_staff(r, t)
    cv.part({(sx - 1, ty), (sx, ty - 1), (sx + 1, ty - 1), (sx + 2, ty), (sx - 1, ty + 1), (sx + 2, ty + 1),
             (sx, ty), (sx + 1, ty)}, "moss", flat="m")
    cv.fill({(sx, ty - 2 - (t == 2)), (sx + 1, ty - 2)}, "moss.l")
    cv.fill(line(sx - 1, ty + 2, sx + 2, ty + 2), "bone.m")  # the cord
    for dx in (-1, 2):  # dried herbs hanging from it
        cv.fill(line(sx + dx, ty + 3, sx + dx, ty + 6 + (t + dx) % 2), "ochre.m")
    cv.put(sx + 1, ty - 1, "ember.l" if t % 2 else "moss.l")  # a dim healing glimmer
    # moss mantle over the shoulders, ragged at the hem
    cv.part(rect(r.tx0 - 2, r.ysh, r.tx1 + 1, r.ysh + 4), "moss")
    for x in range(r.tx0 - 2, r.tx1 + 2, 2):
        cv.put(x, r.ysh + 5, "moss.d")
    cv.fill({(r.tx0 + 1, r.ysh + 7), (r.tx0 + 3, r.ysh + 8)}, "bone.l")  # bone charms
    cv.fill(line(r.tx0, r.yhip - 1, r.tx0, r.yhip + 3 + (t == 2)), "moss.m")  # herbs at the belt
    cv.put(r.tx0 + 1, r.yhip + 1, "ochre.l")


def g_loot(r, t):
    cv = r.cv
    # a heavy stolen sack hunched on the back
    sack = ellipse(r.tx0 - 2, r.ysh + 6, 7, 7)
    cv.part(sack, "leather")
    cv.part({(x, y) for x, y in sack if y < r.ysh + 1}, "leather", flat="l")
    cv.part(line(r.tx0 - 8, r.ysh + 1, r.tx0 + 3, r.ysh + 8), "wood", flat="d")
    cv.fill({(r.tx0 - 5, r.ysh + 1), (r.tx0 - 3, r.ysh - 1 - (t == 2))}, "brass.l")
    cv.fill({(r.tx0 - 4, r.ysh + 7)}, "brass.m")
    w_dagger(r, t, "iron")


def goblin(name, gear, **kw):
    CLASSES[name] = dict(trousers="ashmoss", boots="leather", torso="leather", sleeve="ashmoss",
                         skin="ashmoss", wide=0)
    c = dict(gear=gear, **kw)

    def draw(t):
        r = Rig(name, "side", GOBLIN, dict(bob=1 + BREATH[t], stride=(-3, 3)))
        draw_goblin(r, c, t)
        r.cv.outline()
        return r.cv
    return draw


H["goblin"] = goblin("e-goblin", g_spear)
H["goblin-hexer"] = goblin("e-goblin-hexer", g_hexer)
H["goblin-loot"] = goblin("e-goblin-loot", g_loot)
H["goblin-shaman"] = goblin("e-goblin-shaman", g_shaman)

# -- cave stalker and the generic foe --------------------------------------------------------

def x_claws(r, t, _):
    hx, hy = r.hand_r
    for i in range(3):
        r.cv.part(line(hx + 1, hy - 1 + i, hx + 5, hy + 1 + i + SWAY[t]), "bone", flat="l")
    r.cv.fill({(r.cx + 2, r.top + 3), (r.cx + 3, r.top + 3)}, "ember.l")


H["stalker-cave"] = humanoid("e-stalker-cave", dict(
    skin="stone", torso="stone", sleeve="stone", trousers="stone", boots="stone", glove="stone",
    ribs=True, extra=x_claws, head=dict()),
    size(body_h=47, sw=9, lw=3, aw=2, arm_len=17, hh=7, hw=7))
H["unknown-humanoid"] = humanoid("e-unknown-humanoid", dict(
    robe="stone", trim="stone.d", torso="stone", sleeve="stone", head=dict(hood="stone"),
    flare=0.17))

H["ruffian-dangerous"] = humanoid("e-ruffian-dangerous", dict(
    torso="charcoal", sleeve="charcoal", trousers="leather", boots="leather", belt="leather",
    head=dict(hair="charcoal", mask="skin"), weapon="dagger", straps="leather"), size(sw=12))
H["shadow-master"] = humanoid("e-shadow-master", dict(
    torso="charcoal", sleeve="charcoal", trousers="charcoal", boots="charcoal", belt="plum",
    head=dict(hood="charcoal", mask="plum"), weapon="twin", glove="leather"), size(sw=11))
H["guard-royal"] = humanoid("e-guard-royal", dict(
    torso="steel", sleeve="steel", trousers="steel", boots="steel", belt="brass", glove="steel",
    tabard="oxblood", tabard_trim="brass", head=dict(helm="steel"), weapon="halberd"), size(sw=14))
