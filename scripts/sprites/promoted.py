"""Promoted classes (art set S5): the advanced classes at level 10 and the elite
classes that exist, drawn on the base-class rig.

Each class is its lineage's base figure (figures.DRAWERS), whole ramps swapped to
the class's own materials (kit.recolor), plus a few accessories painted on top
or behind: a cape, horns, a hat, a halo.  The map (32 px, three views) and the
battle idle (64 px, four frames) both come from here, so a promoted figure keeps
the silhouette of its lineage and reads as a step up from it.  Evil routes go
darker with ember touches, good routes lighter with brass, neutral routes
earth-toned.  Palette and proportions are the S0 ones; no new colors.
"""
import figures
from figures import Rig, MAP, BATTLE, DRAWERS, STAND, BREATH, BATTLE_STANCE
from kit import recolor
from pixels import Canvas, rect, line, ellipse


# -- anchors ----------------------------------------------------------------------------

def head_box(r):
    S = r.S
    x0 = r.cx - S.hw // 2
    y0 = r.top
    return x0, y0, S.hw, S.hh, y0 + round(S.hh * 0.5)


def under(r, pixels, ramp, flat=None):
    """Paint behind the figure: only where nothing is drawn yet."""
    tmp = Canvas(r.cv.w, r.cv.h)
    tmp.part(set(pixels), ramp, flat=flat)
    for x, y in set(pixels):
        if r.cv.get(x, y) is None:
            r.cv.put(x, y, tmp.get(x, y))


def over(r, pixels, ramp, flat=None):
    r.cv.part(set(pixels), ramp, flat=flat)


# -- accessories: each is f(r, t) -> None ------------------------------------------------

def cape(ramp, trim=None, length=7):
    def f(r, t):
        v, k = r.view, r.k
        end = r.yhip + r.px(length)
        if v == "up":
            over(r, rect(r.tx0 - 1, r.ysh, r.tx1 + 1, end), ramp)
            over(r, {(r.cx, y) for y in range(r.ysh + 1, end)}, ramp, flat="d")
            if trim:
                over(r, {(x, end) for x in range(r.tx0 - 1, r.tx1 + 2)}, trim, flat="m")
        elif v == "side":
            sway = 1 if t == 2 else 0
            pix = rect(r.tx0 - r.px(3), r.ysh, r.tx0 - 1, end)
            pix |= {(r.tx0 - r.px(3) - sway, y) for y in range(end - 3, end + 1)}
            under(r, pix, ramp)
            if trim:
                under(r, {(x, end) for x, _ in pix}, trim, flat="m")
        else:
            under(r, rect(r.tx0 - 2, r.ysh, r.tx1 + 2, end), ramp)
            if trim:
                under(r, {(x, end) for x in range(r.tx0 - 2, r.tx1 + 3)}, trim, flat="m")
    return f


def hem_band(ramp, drop=3):
    def f(r, t):
        y = r.yhip + r.px(drop)
        if r.view == "up":
            return
        for x in range(r.tx0, r.tx1 + 1):
            if r.cv.get(x, y) not in (None, "outline"):
                r.cv.put(x, y, f"{ramp}.m")
    return f


def collar(ramp, h=2):
    def f(r, t):
        over(r, rect(r.tx0 - 1, r.ysh, r.tx1 + 1, r.ysh + r.px(h) - 1), ramp)
    return f


def pauldron(ramp):
    def f(r, t):
        if r.view == "side":
            xs = [r.cx - 1, r.cx]
            over(r, ellipse(r.cx, r.ysh + 1, 2.4 if r.k > 1 else 1.6, 1.8 if r.k > 1 else 1.2), ramp)
        else:
            for x in (r.tx0 - 1, r.tx1):
                over(r, ellipse(x + 0.5, r.ysh + 1, 2 if r.k > 1 else 1.4, 1.5), ramp)
    return f


def strap(ramp, buckle=None):
    def f(r, t):
        if r.view == "up":
            return
        a = (r.tx0 + 1, r.ysh + 1)
        b = (r.tx1, r.yhip - r.px(2))
        over(r, line(a[0], a[1], b[0], b[1]), ramp, flat="m")
        if buckle:
            r.cv.put((a[0] + b[0]) // 2, (a[1] + b[1]) // 2, buckle)
    return f


def plume(ramp, tall=4):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        sway = [0, 0, 1, 0][t]
        h = max(tall, r.px(tall * 0.7))
        if r.view == "side":
            pts = {(x0 + 1 - i // 2 - sway * (i > 1), y0 - 1 - i) for i in range(h)}
            pts |= {(x0 + 2 - i // 2 - sway * (i > 1), y0 - 1 - i) for i in range(h - 1)}
        else:
            pts = {(r.cx - 1 + (i // 3) - (i == h - 1), y0 - 1 - i) for i in range(h)}
            pts |= {(r.cx + (i // 3), y0 - 1 - i) for i in range(h - 1)}
        over(r, pts, ramp, flat="m")
    return f


def horns(ramp, size=2):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        s = max(size, r.px(size))
        if r.view == "side":
            pts = {(x0 + 2 + i // 2 + (i > 1) , y0 - i) for i in range(s + 1)}
            pts |= {(x0 + 3 + s // 2 + 1, y0 - s)}
        else:
            pts = {(x0 - i // 2, y0 - 1 - i) for i in range(s + 1)}
            pts |= {(x0 + hw - 1 + i // 2, y0 - 1 - i) for i in range(s + 1)}
        over(r, pts, ramp, flat="l")
    return f


def antlers(ramp):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        h = r.px(4)
        if r.view == "side":
            pts = line(x0 + 2, y0, x0 + 1, y0 - h) | line(x0 + 1, y0 - h + 2, x0 + 4, y0 - h) \
                | line(x0 + 3, y0, x0 + 4, y0 - h + 1)
        else:
            pts = line(x0, y0, x0 - 1, y0 - h) | line(x0 - 1, y0 - h + 2, x0 - 3, y0 - h + 1) \
                | line(x0 + hw - 1, y0, x0 + hw, y0 - h) | line(x0 + hw, y0 - h + 2, x0 + hw + 2, y0 - h + 1)
        over(r, pts, ramp, flat="m")
    return f


def halo(ramp="brass"):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        yy = y0 - (3 if r.k > 1 else 2)
        pts = {(x, yy) for x in range(x0, x0 + hw)}
        if r.k > 1:
            pts |= {(x0 - 1, yy + 1), (x0 + hw, yy + 1)}
        c = f"{ramp}.l" if t == 2 else f"{ramp}.m"
        for x, y in pts:
            r.cv.put(x, y, c)
    return f


def circlet(ramp):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        yy = y0 + max(1, round(1.4 * r.k))
        for x in range(x0, x0 + hw):
            if r.cv.get(x, yy) not in (None, "outline"):
                r.cv.put(x, yy, f"{ramp}.m")
        r.cv.put(r.cx, yy - 1, f"{ramp}.l")
    return f


def pointed_hat(ramp, trim=None, tall=6, bend=True):
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        h = r.px(tall)
        brim = {(x, y0 - 1) for x in range(x0 - 1, x0 + hw + 1)}
        cone = set()
        for i in range(h):
            half = max(0, (hw - 1) * (h - i) // (h * 2))
            c = (x0 + hw // 2) - (i * (1 if bend else 0)) // 4
            cone |= {(x, y0 - 2 - i) for x in range(c - half, c + half + 1)}
        over(r, cone | {(x, y0) for x in range(x0, x0 + hw)} - set(), ramp)
        over(r, brim, trim or ramp, flat="d")
    return f


def hood_cowl(ramp):
    """A deep hood over a bare head: top, back and side, face left open."""
    def f(r, t):
        x0, y0, hw, hh, eye_y = head_box(r)
        if r.view == "side":
            pix = {(x, y) for x in range(x0 - 1, x0 + hw - 1) for y in range(y0 - 1, y0 + hh)}
            pix |= {(x, y0 - 1) for x in range(x0, x0 + hw)}
        elif r.view == "up":
            pix = rect(x0 - 1, y0 - 1, x0 + hw, y0 + hh)
        else:
            pix = {(x, y) for x in range(x0 - 1, x0 + hw + 1) for y in range(y0 - 1, y0 + hh)
                   if y < y0 + 2 or x <= x0 or x >= x0 + hw - 1}
        over(r, pix, ramp)
    return f


def fur_collar(ramp="wool"):
    def f(r, t):
        y = r.ysh
        pix = {(x, y + dy) for x in range(r.tx0 - 1, r.tx1 + 2) for dy in range(max(1, r.px(1.5)))}
        over(r, pix, ramp)
        for x in range(r.tx0 - 1, r.tx1 + 2, 2):
            r.cv.put(x, y + max(1, r.px(1.5)), f"{ramp}.d")
    return f


def eyes(color="ember.m", flicker=False):
    def f(r, t):
        if r.view == "up":
            return
        x0, y0, hw, hh, eye_y = head_box(r)
        c = color
        if flicker and t == 2:
            c = "ember.l"
        if r.view == "side":
            r.cv.put(x0 + hw - 2, eye_y, c)
        else:
            ex = max(1, round(1.6 * r.k))
            r.cv.put(r.cx - ex, eye_y, c)
            r.cv.put(r.cx + ex - 1, eye_y, c)
    return f


def skin_tint(ramp):
    """Repaint exposed skin in another ramp (a hag's grey-green face)."""
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        for y in range(y0 - 1, y0 + hh + 2):
            for x in range(x0 - 1, x0 + hw + 2):
                c = r.cv.get(x, y)
                if c and c.startswith("skin."):
                    r.cv.put(x, y, f"{ramp}.{c[-1]}")
    return f


def hooked_nose(ramp="skin"):
    def f(r, t):
        if r.view != "side":
            return
        x0, y0, hw, hh, eye_y = head_box(r)
        r.cv.put(x0 + hw + 1, eye_y + 1, f"{ramp}.d")
        if r.k > 1:
            r.cv.put(x0 + hw + 1, eye_y + 2, f"{ramp}.d")
    return f


def grey_hair(ramp="bone"):
    def f(r, t):
        x0, y0, hw, hh, eye_y = head_box(r)
        if r.view == "side":
            pix = {(x0 - 1 - i // 2, y0 + 1 + i) for i in range(r.px(5))}
        else:
            pix = {(x0 - 1, y) for y in range(y0 + 1, y0 + r.px(5))} | {(x0 + hw, y) for y in range(y0 + 1, y0 + r.px(5))}
        under(r, pix, ramp, flat="m")
    return f


def beard(ramp="bone"):
    def f(r, t):
        if r.view == "up":
            return
        x0, y0, hw, hh, eye_y = head_box(r)
        if r.view == "side":
            pix = rect(x0 + 2, eye_y + 2, x0 + hw, y0 + hh + r.px(3))
        else:
            pix = rect(x0 + 1, eye_y + 2, x0 + hw - 2, y0 + hh + r.px(3))
        over(r, pix, ramp)
    return f


def sprig(ramp="moss", n=3):
    """A few leaves along the brow, a wreath of green."""
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        for i in range(n * 2):
            x = x0 + i * (hw + 1) // (n * 2)
            r.cv.put(x, y0 - (i % 2), f"{ramp}.l" if i % 2 else f"{ramp}.m")
    return f


def tint_spot(pts, ramp):
    """Fixed spots in ramp, relative to hand_r: (dx, dy)."""
    def f(r, t):
        hx, hy = r.hand_r
        for dx, dy in pts:
            r.cv.put(hx + dx, hy + dy, ramp)
    return f


def feather_cap(ramp="bone", cap="oxblood"):
    """A jaunty cap with a long plume trailing back."""
    def f(r, t):
        x0, y0, hw, hh, _ = head_box(r)
        sway = [0, 1, 1, 0][t]
        cap_px = {(x, y0 - 1) for x in range(x0 - 1, x0 + hw)} | {(x, y0 - 2) for x in range(x0 + 1, x0 + hw - 1)}
        over(r, cap_px, cap)
        if r.view == "side":
            tail = {(x0 - i + (1 if i > 3 else 0) + sway, y0 - 3 - (i // 2)) for i in range(r.px(5))}
            tail |= {(x0 - 2 - i + sway, y0 - 3 + (i // 2)) for i in range(2)}
        else:
            tail = {(x0 + hw - 1 + i, y0 - 3 - i // 2) for i in range(r.px(4))}
        over(r, tail, ramp, flat="m")
    return f


def rapier():
    def f(r, t):
        if r.view != "side":
            return
        hx, hy = r.hand_r
        n = 12 if r.k > 1 else 7
        r.cv.part(line(hx, hy, hx + n, hy + (n // 2) + [0, 0, 1, 0][t]), "steel", flat="l")
        r.cv.part(line(hx - 1, hy - 1, hx + 1, hy + 1), "brass", flat="m")
    return f


def satchel(ramp="leather", clasp="brass.m"):
    def f(r, t):
        if r.view == "up":
            return
        yy = r.yhip - r.px(1)
        pix = rect(r.tx0 - 2 if r.view == "side" else r.tx1, yy, r.tx0 if r.view == "side" else r.tx1 + 2, yy + r.px(3))
        over(r, pix, ramp)
        r.cv.put(min(pix)[0] + 1, yy, clasp)
    return f


def herb_bundle():
    def f(r, t):
        if r.view == "up":
            return
        x = r.tx1 if r.view != "side" else r.cx + 1
        y = r.yhip - r.px(1)
        for i in range(r.px(2) + 1):
            r.cv.put(x, y + i, "moss.m")
            r.cv.put(x + 1, y + i, "moss.l" if i % 2 else "ochre.l")
        r.cv.put(x, y - 1, "bone.l")
    return f


def basket():
    def f(r, t):
        hx, hy = r.hand_r
        if r.view == "up":
            return
        pix = rect(hx - 1, hy - 1, hx + (3 if r.k > 1 else 2), hy + (3 if r.k > 1 else 2))
        over(r, pix, "ochre")
        for x in range(hx - 1, hx + 4, 2):
            r.cv.put(x, hy + (1 if r.k > 1 else 0), "ochre.d")
        r.cv.put(hx + 1, hy - 2, "moss.l")
        r.cv.put(hx + 2, hy - 2, "moss.m")
    return f


def crescent_top():
    """A pale crescent on the top of the staff-wand."""
    def f(r, t):
        hx, hy = r.hand_r
        top = hy - r.px(9) - 1
        for dx, dy in ((0, 0), (-1, 1), (-1, 2), (0, 3), (1, 0), (2, 1)) if r.k > 1 else ((0, 0), (-1, 1), (0, 2)):
            r.cv.put(hx + dx, top + dy, "bone.l")
    return f


def spikes(ramp="iron.l"):
    def f(r, t):
        if r.view == "up":
            return
        pts = [(r.tx0 + 1, r.ysh - 1), (r.cx, r.ysh - 2)] if r.view == "side" else \
            [(r.tx0 - 1, r.ysh - 1), (r.tx0, r.ysh - 2), (r.tx1, r.ysh - 2), (r.tx1 + 1, r.ysh - 1)]
        for x, y in pts:
            r.cv.put(x, y, ramp)
            if r.k > 1:
                r.cv.put(x, y - 1, ramp)
    return f


def glow_stone(ramp="ember"):
    """Recolor the staff stone to a glowing coal (wizard-line staffs)."""
    def f(r, t):
        sx = r.hand_r[0] + (1 if r.battle else 0) if r.battle else None
        # find the stone: the topmost cluster of bone.* pixels above the head
        x0, y0, hw, hh, _ = head_box(r)
        for y in range(0, y0 + 2):
            for x in range(r.cv.w):
                c = r.cv.get(x, y)
                if c and c.startswith("bone."):
                    r.cv.put(x, y, f"{ramp}.{'l' if t == 2 else 'm'}" if c.endswith(("m", "l")) else f"{ramp}.d")
    return f


def skull_charm():
    def f(r, t):
        if r.view == "up":
            return
        x = r.cx + (1 if r.view == "side" else 0)
        y = r.yhip - r.px(3)
        r.cv.put(x, y, "bone.l")
        r.cv.put(x, y + 1, "bone.m")
        if r.k > 1:
            r.cv.put(x + 1, y, "bone.m")
            r.cv.put(x + 1, y + 1, "outline")
    return f


def blood_drip():
    def f(r, t):
        hx, hy = r.hand_r
        if r.view == "up":
            return
        r.cv.put(hx + 1, hy - r.px(9) + 2 + [0, 1, 2, 1][t], "ember.m")
    return f


def rune_glow():
    def f(r, t):
        hx, hy = r.hand_r
        if r.view == "up":
            return
        c = "ember.l" if t == 2 else "ember.m"
        r.cv.put(hx + 1, hy - 1, c)
        r.cv.put(hx + 2, hy, "ember.d")
        r.cv.put(hx, hy - 2, "ember.d")
    return f


def scars_markings():
    """Bone-white war paint stripes across the face (stalker)."""
    def f(r, t):
        if r.view != "side":
            return
        x0, y0, hw, hh, eye_y = head_box(r)
        r.cv.put(x0 + hw - 3, eye_y + 1, "bone.l")
        r.cv.put(x0 + hw - 2, eye_y + 2, "bone.l")
    return f


def fur_back():
    """A hide mantle over one shoulder trailing down the back (hunter)."""
    def f(r, t):
        pix = rect(r.tx0 - 1, r.ysh, r.tx1 + 1, r.ysh + r.px(5))
        over(r, pix, "wool") if r.view == "up" else under(r, pix, "wool")
        for x in range(r.tx0 - 1, r.tx1 + 2, 2):
            r.cv.put(x, r.ysh + r.px(5), "wool.d")
    return f


# -- the classes --------------------------------------------------------------------------
#
# base: the lineage figure.  mat: whole-ramp swaps.  acc: accessories in painting order.

CLASS_ART = {
    # Warrior lineage ---------------------------------------------------------------
    "knight": dict(base="warrior", mat={"oxblood": "slate"},
                   acc=[hem_band("brass"), plume("bone", 4)]),
    "paladin": dict(base="warrior", mat={"oxblood": "bone", "iron": "steel"},
                    acc=[cape("oxblood", "brass", 8), hem_band("brass"), pauldron("brass"),
                         circlet("brass"), halo("brass")]),
    "mercenary": dict(base="warrior", mat={"oxblood": "ochre", "iron": "leather"},
                      acc=[strap("leather", "brass.l"), satchel("leather")]),
    "blackguard": dict(base="warrior", mat={"oxblood": "charcoal", "iron": "iron"},
                       acc=[spikes("steel.l"), hem_band("oxblood")]),
    "dread-knight": dict(base="warrior", mat={"oxblood": "plum", "iron": "charcoal", "steel": "iron"},
                         acc=[cape("charcoal", "plum", 9), spikes("bone.l"), horns("bone", 3),
                              eyes("ember.m", True)]),
    # Cleric lineage ----------------------------------------------------------------
    "priest": dict(base="cleric", mat={"iron": "wool", "wool": "bone"},
                   acc=[collar("brass", 2), circlet("brass")]),
    "hierarch": dict(base="cleric", mat={"iron": "bone", "wool": "brass", "steel": "brass"},
                     acc=[cape("bone", "brass", 9), collar("oxblood", 3), halo("brass")]),
    "druid": dict(base="cleric", mat={"iron": "leather", "wool": "moss", "steel": "wood", "brass": "moss"},
                  acc=[sprig("moss"), herb_bundle()]),
    "elder-druid": dict(base="cleric", mat={"iron": "leather", "wool": "ashmoss", "steel": "wood", "brass": "moss"},
                        acc=[cape("moss", "ashmoss", 9), antlers("bone"), beard("bone"), herb_bundle()]),
    "blood-priest": dict(base="cleric", mat={"iron": "charcoal", "wool": "oxblood", "steel": "oxblood"},
                         acc=[collar("oxblood", 2), blood_drip(), eyes("ember.m")]),
    "demonologist": dict(base="cleric", mat={"iron": "plum", "wool": "charcoal", "steel": "bone"},
                         acc=[cape("plum", "ember", 9), horns("bone", 2), rune_glow(), eyes("ember.m", True)]),
    # Rogue lineage -----------------------------------------------------------------
    "scout": dict(base="rogue", mat={"charcoal": "moss", "plum": "leather"},
                  acc=[plume("bone", 3), satchel("leather")]),
    "duelist": dict(base="rogue", mat={"charcoal": "oxblood", "plum": "brass"},
                    acc=[feather_cap("bone", "oxblood"), rapier()]),
    "assassin": dict(base="rogue", mat={"charcoal": "charcoal", "plum": "iron", "steel": "moss"},
                     acc=[hood_cowl("charcoal"), eyes("ember.m")]),
    # Ranger lineage ----------------------------------------------------------------
    "warden": dict(base="ranger", mat={"moss": "forest", "leather": "moss"},
                   acc=[cape("forest", "moss", 8), sprig("moss", 2)]),
    "hunter": dict(base="ranger", mat={"moss": "leather", "leather": "ochre"},
                   acc=[fur_collar("wool"), satchel("wool", "bone.l")]),
    "stalker": dict(base="ranger", mat={"moss": "charcoal", "leather": "charcoal", "wood": "iron"},
                    acc=[hood_cowl("charcoal"), scars_markings(), eyes("ember.m")]),
    # Wizard lineage ----------------------------------------------------------------
    "theurgist": dict(base="wizard", mat={"slate": "wool", "bone": "brass"},
                      acc=[hem_band("brass", 5), collar("brass", 2), circlet("brass")]),
    "arcanist": dict(base="wizard", mat={"slate": "water"},
                     acc=[pointed_hat("water", "brass", 6), hem_band("brass", 5)]),
    "warlock": dict(base="wizard", mat={"slate": "charcoal"},
                    acc=[hem_band("oxblood", 5), skull_charm(), eyes("ember.m", True)]),
    # Witch lineage -----------------------------------------------------------------
    "hedge-witch": dict(base="witch", mat={"heather": "moss", "ashmoss": "wool"},
                        acc=[pointed_hat("moss", "wool", 4), basket()]),
    "coven-sage": dict(base="witch", mat={"heather": "slate", "ashmoss": "ochre"},
                       acc=[crescent_top(), circlet("bone")]),
    "hag": dict(base="witch", mat={"heather": "plum", "ashmoss": "charcoal", "skin": "ashmoss"},
                acc=[hooked_nose("ashmoss"), grey_hair("bone"), eyes("ember.m")]),
}

CLASS_IDS = list(CLASS_ART)
LINEAGE = {"knight": "warrior", "paladin": "warrior", "mercenary": "warrior", "blackguard": "warrior",
           "dread-knight": "warrior", "priest": "cleric", "hierarch": "cleric", "druid": "cleric",
           "elder-druid": "cleric", "blood-priest": "cleric", "demonologist": "cleric", "scout": "rogue",
           "duelist": "rogue", "assassin": "rogue", "warden": "ranger", "hunter": "ranger",
           "stalker": "ranger", "theurgist": "wizard", "arcanist": "wizard", "warlock": "wizard",
           "hedge-witch": "witch", "coven-sage": "witch", "hag": "witch"}
assert set(LINEAGE) == set(CLASS_ART)
NAMES = {cid: cid.replace("-", " ") for cid in CLASS_IDS}


def _paint(cid, view, S, pose, t):
    art = CLASS_ART[cid]
    r = Rig(art["base"], view, S, pose)
    DRAWERS[art["base"]](r)
    if art["base"] in figures.EXTRAS and S is BATTLE:
        figures.EXTRAS[art["base"]](r, t)
    r.cv = recolor(r.cv, art["mat"]) if art["mat"] else r.cv
    for acc in art["acc"]:
        acc(r, t)
    r.cv.outline()
    return r.cv


def render(cid, view, S, pose=None):
    return _paint(cid, view, S, pose or {}, 0)


def map_idle(cid):
    return [[render(cid, v, MAP, p) for p in figures.IDLE_POSES] for v in ("down", "up", "side")]


def map_walk(cid):
    return [[render(cid, v, MAP, p) for p in figures.WALK_POSES] for v in ("down", "up", "side")]


def battle_idle(cid, t=0):
    base = CLASS_ART[cid]["base"]
    st = {**dict(bob=0, stride=(-3, 3)), **BATTLE_STANCE.get(base, {})}
    pose = dict(st, bob=st["bob"] + BREATH[t % 4])
    return _paint(cid, "side", BATTLE, pose, t % 4)
