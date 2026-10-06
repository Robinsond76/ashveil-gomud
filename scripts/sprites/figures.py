"""Humanoid figures: the six base classes plus the fallback adventurer.

A figure is built from shaded parts (legs, torso, arms, head, gear) painted
in the class's materials, then given a 1 px outline.  The same rig draws the
32x32 map frames (down, up, side; idle and walk) and the 64x64 battle idle
frame (side, facing right), so a silhouette reads the same at both sizes.
Parts and colors are data; nothing here is a copy of any existing artwork.
"""
import math

from pixels import Canvas, rect, round_rect, line, thick, ellipse


class Size:
    def __init__(self, **kw):
        self.__dict__.update(kw)


# Map: about 5 heads tall (28 px).  Battle: about 7 heads tall (46 px).
MAP = Size(W=32, body_h=28, hh=6, T=10, L=12, hw=6, sw=10, lw=3, g=1, aw=2,
           boot=3, arm_len=8, hand=2, k=1.0)
BATTLE = Size(W=64, body_h=46, hh=7, T=17, L=22, hw=7, sw=14, lw=5, g=2, aw=3,
              boot=6, arm_len=13, hand=3, k=1.6)

IDLE_POSES = [dict(bob=0), dict(bob=1)]
# Walk: contact, pass, contact (other leg), pass.
WALK_POSES = [
    dict(bob=0, lift=(0, 1), stride=(-2, 2), swing=2),
    dict(bob=-1, lift=(0, 0), stride=(0, 0), swing=0),
    dict(bob=0, lift=(1, 0), stride=(2, -2), swing=-2),
    dict(bob=-1, lift=(0, 0), stride=(0, 0), swing=0),
]
STAND = dict(bob=0, lift=(0, 0), stride=(0, 0), swing=0)

# Per-class materials.  "wide" widens the torso; "robe" hides the legs.
CLASSES = {
    "warrior": dict(trousers="leather", boots="leather", torso="iron", sleeve="iron",
                    wide=2, hair="leather"),
    "rogue": dict(trousers="charcoal", boots="charcoal", torso="charcoal", sleeve="charcoal",
                  hood="charcoal"),
    "ranger": dict(trousers="leather", boots="leather", torso="leather", sleeve="moss",
                   hood="moss"),
    "cleric": dict(trousers="wool", boots="leather", torso="wool", sleeve="iron",
                   hair="leather"),
    "wizard": dict(robe="slate", boots="leather", torso="slate", sleeve="slate", hood="slate"),
    "witch": dict(robe="heather", boots="leather", torso="heather", sleeve="heather",
                  hood="heather"),
    "adventurer": dict(trousers="leather", boots="leather", torso="wool", sleeve="wool",
                       hood="leather"),
}


class Rig:
    """Geometry for one figure at one size, view and pose."""

    def __init__(self, cls, view, S, pose):
        self.spec = CLASSES[cls]
        self.skin = self.spec.get("skin", "skin")
        self.cls, self.view, self.S = cls, view, S
        self.pose = {**STAND, **pose}
        self.cv = Canvas(S.W, S.W)
        self.k = S.k
        self.cx = S.W // 2
        self.fy = S.W - 3
        self.bob = self.pose["bob"] if self.pose["bob"] > 0 else self.pose["bob"]
        self.top = self.fy - S.body_h + 1 + self.bob
        self.ysh = self.top + S.hh  # shoulder (torso top)
        self.yhip = self.ysh + S.T  # first leg row
        side = view == "side"
        self.sw = S.sw + self.spec.get("wide", 0) * (2 if S.k > 1 else 1)
        self.battle = side and S.k > 1
        self.tw = round(self.sw * (0.75 if self.battle else 0.6)) if side else self.sw
        if side and self.cls == "warrior" and not self.battle:
            self.tw += 1
        if self.battle and self.cls == "warrior":
            self.tw -= 2  # broad, but not a slab at 64 px
        self.tx0 = self.cx - self.tw // 2
        self.tx1 = self.tx0 + self.tw - 1
        self.hand_r = (self.tx1 + 2, self.ysh + S.arm_len - 3)  # refined by arms()

    def shade_down(self, pixels):
        """Push painted pixels one shade darker (far limbs, back layers)."""
        for x, y in pixels:
            c = self.cv.get(x, y)
            if c and c.endswith((".l", ".m")):
                self.cv.put(x, y, c[:-1] + ("m" if c.endswith(".l") else "d"))

    def px(self, v):
        return max(1, round(v * self.k))

    # -- body parts -----------------------------------------------------
    def legs(self, ramp=None, boot=None):
        S, p = self.S, self.pose
        ramp = ramp or self.spec["trousers"]
        boot = boot or self.spec["boots"]
        by = self.fy - S.boot + 1
        parts = []
        if self.view == "side":
            for i in (0, 1):  # back leg first, then front leg
                lift = p["lift"][i]
                off = p["stride"][i]
                x0 = self.cx - S.lw // 2 + off
                leg = rect(x0, self.yhip, x0 + S.lw - 1, self.fy - lift)
                parts.append((i, leg, x0, lift))
            for i, leg, x0, lift in parts:
                body = {q for q in leg if q[1] < by - lift}
                shoe = {q for q in leg if q[1] >= by - lift}
                shoe |= rect(x0 + S.lw, self.fy - lift - max(0, S.boot - 3),
                             x0 + S.lw + (1 if S.k == 1 else 2), self.fy - lift)
                self.cv.part(body, ramp)
                self.cv.part(shoe, boot)
                if self.battle and i == 0:
                    self.shade_down(body | shoe)
        else:
            for i, x0 in enumerate((self.cx - S.g // 2 - S.lw, self.cx + S.g - S.g // 2)):
                lift = p["lift"][i]
                leg = rect(x0, self.yhip, x0 + S.lw - 1, self.fy - lift)
                body = {q for q in leg if q[1] < by - lift}
                shoe = {q for q in leg if q[1] >= by - lift}
                self.cv.part(body, ramp)
                self.cv.part(shoe, boot)

    def torso(self, ramp, shape=None):
        t = shape or rect(self.tx0, self.ysh, self.tx1, self.yhip - 1)
        self.cv.part(t, ramp)
        return t

    def arms(self, sleeve, glove="skin", length=None, swing=None):
        S = self.S
        ln = length or S.arm_len
        sw = self.pose["swing"] if swing is None else swing
        if self.battle:
            hx, hy = self.tx1 + 2, self.ysh + ln - 2
            arm = thick(line(self.cx - 1, self.ysh + 2, hx - 1, hy - S.hand), S.aw)
            self.cv.part(arm, sleeve)
            self.cv.part(rect(hx - 1, hy - S.hand + 1, hx, hy), glove)
            self.hand_r = (hx, hy - 1)
            return
        if self.view == "side":
            x0 = self.cx - S.aw // 2 + sw
            y0 = self.ysh + 1
            arm = rect(x0, y0, x0 + S.aw - 1, y0 + ln - S.hand - 1)
            hand = rect(x0, y0 + ln - S.hand, x0 + S.aw - 1, y0 + ln - 1)
            self.cv.part(arm, sleeve)
            self.cv.part(hand, glove)
            self.hand_r = (x0 + S.aw, y0 + ln - 1)
            return
        for i, x0 in enumerate((self.tx0 - S.aw, self.tx1 + 1)):
            y0 = self.ysh
            off = sw if i == 0 else -sw
            arm = rect(x0, y0, x0 + S.aw - 1, y0 + ln - S.hand - 1 + (off > 0))
            hand = rect(x0, y0 + ln - S.hand + (off > 0), x0 + S.aw - 1, y0 + ln - 1 + (off > 0))
            self.cv.part(arm, sleeve)
            self.cv.part(hand, glove)
            if i == 0:
                self.hand_l = (x0, y0 + ln - 1)
            else:
                self.hand_r = (x0 + S.aw - 1, y0 + ln - 1)

    def head(self, hood=None, hair=None, mask=None, beard=None, helm=None):
        """Head, with optional hood / hair / helm / mask / beard."""
        S, v = self.S, self.view
        hw, hh = S.hw, S.hh
        if v == "side":
            x0 = self.cx - hw // 2
        else:
            x0 = self.cx - hw // 2
        y0 = self.top
        base = round_rect(x0, y0, x0 + hw - 1, y0 + hh - 1)
        eye_y = y0 + round(hh * 0.5)
        cover = hood or helm
        if v == "up":
            self.cv.part(base, cover or hair or "leather")
            if hood:
                self.cv.part({(x, y) for x, y in base if y > y0 + hh - 3}, hood, flat="d")
            return
        if v == "side":
            nose = {(x0 + hw, eye_y + 1)}
            self.cv.part(base | nose, self.skin)
            if cover:
                back = {(x, y) for x, y in base if x < x0 + hw - max(2, round(2 * self.k))}
                top = {(x, y) for x, y in base if y < y0 + max(2, round(2 * self.k))}
                ex = {(x, y) for x, y in (back | top)}
                ex |= {(x0 - 1, y) for y in range(y0 + 1, y0 + hh - 1)} if hood else set()
                self.cv.part(ex, cover)
                if helm:
                    brim = {(x, y0 + max(2, round(2 * self.k))) for x in range(x0, x0 + hw)}
                    self.cv.part(brim, cover)
            elif hair:
                hp = {(x, y) for x, y in base if y < y0 + 2 or x < x0 + 2}
                self.cv.part(hp, hair)
            self.cv.put(x0 + hw - 2, eye_y, "outline")
            if mask:
                mk = {(x, y) for x, y in base | nose if y > eye_y and x > x0 + 1}
                self.cv.part(mk, mask)
            if beard:
                bd = {(x, y) for x, y in rect(x0 + 2, eye_y + 1, x0 + hw, y0 + hh + self.px(2) - 1)}
                self.cv.part(bd, beard)
            return
        # front
        self.cv.part(base, self.skin)
        if cover:
            side_w = max(1, round(1.3 * self.k))
            frame = {(x, y) for x, y in base
                     if y < y0 + max(2, round(2 * self.k)) or x < x0 + side_w
                     or x >= x0 + hw - side_w}
            self.cv.part(frame, cover)
            if helm:  # nasal guard
                self.cv.part({(self.cx, y) for y in range(y0 + 1, eye_y + 2 + (self.k > 1))}, helm)
        elif hair:
            hp = {(x, y) for x, y in base if y < y0 + max(1, round(1.5 * self.k))}
            hp |= {(x, y) for x, y in base if (x < x0 + 1 or x > x0 + hw - 2) and y < eye_y}
            self.cv.part(hp, hair)
        ex = max(1, round(1.6 * self.k))
        self.cv.put(self.cx - ex, eye_y, "outline")
        self.cv.put(self.cx + ex - 1, eye_y, "outline")
        if mask:
            mk = {(x, y) for x, y in base if y > eye_y and x > x0 and x < x0 + hw - 1}
            self.cv.part(mk, mask)
        if beard:
            bd = rect(x0 + 1, eye_y + 2, x0 + hw - 2, y0 + hh + self.px(2) - 1)
            bd |= {(x0 + 2, eye_y + 1), (x0 + hw - 3, eye_y + 1)}
            self.cv.part(bd, beard)


# ---------------------------------------------------------------------------
# Class drawers.  Each paints back gear, the body, then front gear.
# ---------------------------------------------------------------------------

def _belt(r, ramp="leather", y=None, buckle=None):
    y = y if y is not None else r.yhip - r.px(2)
    r.cv.part({(x, y) for x in range(r.tx0, r.tx1 + 1)}, ramp, flat="m")
    if buckle and r.view == "down":
        r.cv.put(r.cx, y, buckle)


def draw_warrior(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    r.legs()
    r.torso("iron")
    # Faded oxblood surcoat over the mail, belted, hem to mid-thigh.
    hem = r.yhip + r.px(3)
    if v == "side":
        sur = rect(r.tx0, r.ysh + r.px(2), r.tx1, hem)
    else:
        sur = rect(r.tx0 + 1, r.ysh + r.px(2), r.tx1 - 1, hem)
    cv.part(sur, "oxblood")
    if r.battle:
        cv.part(line(r.cx, r.yhip, r.cx, hem), "iron", flat="d")
        r.shade_down({(x, y) for x, y in sur if x < r.tx0 + 2 and y > r.ysh + 3})
    _belt(r, "leather", r.yhip - r.px(1), "brass.m")
    r.arms("iron")
    if v != "up":
        r.head(helm="iron")
    else:
        r.head(helm="iron")
    # Round shield on the near arm; sword at the hip.
    sr = r.px(3.3) if v != "side" else r.px(3)
    if v == "down":
        sx, sy = r.tx0 - S.aw + 1 - (0 if k == 1 else 1), r.ysh + r.px(6)
        sh = ellipse(sx, sy, sr, sr)
        cv.part(sh, "wood")
        cv.put(sx, sy, "steel.m")
        if k > 1:
            cv.fill({(sx - 1, sy), (sx, sy - 1), (sx + 1, sy), (sx, sy + 1)}, "steel.m")
        sw = line(r.tx1 + S.aw + 1, r.yhip - r.px(4), r.tx1 + S.aw + 1, r.yhip + r.px(5))
        cv.part(thick(sw, 1), "steel")
        cv.part({(r.tx1 + S.aw + 1, r.yhip - r.px(5))}, "brass")
    elif v == "up":
        sx, sy = r.cx, r.ysh + r.px(5)
        cv.part(ellipse(sx, sy, sr + 1, sr + 1), "wood")
        cv.part({(sx, sy)}, "steel", flat="m")
    elif r.battle:
        # Braced: sword held low and forward by the far hand, the round
        # shield pushed forward on the near arm, seen edge-on.
        hx, hy = r.hand_r
        gx, gy = r.tx1 + 1, r.yhip + 1
        cv.part(thick(line(gx + 1, gy + 1, gx + 11, gy + 7), 1), "steel")
        cv.part(line(gx, gy - 2, gx + 2, gy + 2), "brass", flat="m")
        sx, sy = hx + 2, hy + 1
        cv.part(ellipse(sx, sy, 2.4, 8), "wood")
        cv.part({(sx, y) for y in range(sy - 8, sy + 9) if (sx, y) in ellipse(sx, sy, 2.4, 8)},
                "steel", flat="d")
        cv.part(rect(sx + 2, sy - 1, sx + 3, sy + 1), "steel", flat="m")
    else:
        sx, sy = r.cx + 1, r.ysh + r.px(5)
        sh = ellipse(sx, sy, sr - 0.4, sr + 0.4)
        cv.part(sh, "wood")
        cv.part({(sx, sy)}, "steel", flat="m")
        sw = line(r.tx0 - 1, r.yhip - r.px(2), r.tx0 - r.px(4), r.yhip + r.px(5))
        cv.part(thick(sw, 1), "steel")


def draw_rogue(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    # Battle: dark leather breeches under the charcoal jerkin, so the legs
    # read apart from the body.
    r.legs(ramp="leather" if r.battle else None)
    r.torso("charcoal")
    # Dull plum sash and cross-strap.
    cv.part({(x, r.yhip - r.px(2)) for x in range(r.tx0, r.tx1 + 1)}, "plum", flat="m")
    if v == "down":
        cv.fill(line(r.tx0 + 1, r.ysh + 1, r.tx1 - 1, r.yhip - r.px(2)), "leather.m")
    r.arms("charcoal", glove="leather")
    r.head(hood="charcoal", mask="plum")
    # Two short blades crossed low at the hips.
    if r.battle:
        # Crouched with both blades in a reverse grip, edges trailing back
        # along the forearms.
        hx, hy = r.hand_r
        cv.part(line(hx, hy - 1, hx - 7, hy - 6), "steel", flat="l")
        cv.part(rect(hx, hy - 2, hx + 1, hy - 1), "leather", flat="d")
        cv.part(line(r.tx0 + 1, r.yhip - 2, r.tx0 - 5, r.yhip + 4), "steel", flat="m")
        cv.part(line(r.tx0 + 2, r.yhip - 3, r.tx0 + 3, r.yhip - 2), "leather", flat="d")
    for sgn in (-1, 1):
        if v == "side" and (sgn == 1 or r.battle):
            continue
        bx = r.cx + sgn * (r.tw // 2 + (r.px(1) if v != "side" else 0))
        bl = line(bx, r.yhip - r.px(2), bx + sgn * r.px(1), r.yhip + r.px(4))
        cv.part(bl, "steel", flat="m")
        cv.put(bx, r.yhip - r.px(3), "leather.m")
    # Hood point trails behind the neck.
    if v == "side":
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx0, r.ysh + r.px(4)), "charcoal")


def draw_ranger(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    # Longbow and quiver behind the shoulder.
    if r.battle:
        qv = rect(r.tx0 - 1, r.ysh - 2, r.tx0 + 2, r.ysh + 10)
        cv.part(qv, "leather")
        for i in range(3):
            cv.put(r.tx0 + i - 1, r.ysh - 3 - i % 2, "bone.m")
    elif v == "side":
        bow = {(r.tx0 - r.px(1) + int(abs(y - (r.ysh + r.px(6))) * 0.18), y)
               for y in range(r.top + 1, r.yhip + r.px(4))}
        cv.part(thick(bow, 1), "wood", flat="m")
        qv = rect(r.tx0, r.ysh - 1, r.tx0 + r.px(2), r.ysh + r.px(6))
        cv.part(qv, "leather")
    else:
        bx = r.tx1 + S.aw + 1
        bow = {(bx + int(abs(y - (r.ysh + r.px(6))) * -0.1), y)
               for y in range(r.top + 2, r.yhip + r.px(4))}
        cv.part(thick(bow, 1), "wood", flat="m")
        qv = rect(r.tx0 + 1, r.top + r.px(3), r.tx0 + 1 + r.px(2), r.ysh + r.px(8))
        if v == "up":
            qv = rect(r.cx - 1, r.top + r.px(2), r.cx + r.px(2), r.ysh + r.px(7))
        cv.part(qv, "leather")
        cv.put(qv and min(qv)[0] + 1, r.top + r.px(2), "bone.m")
    r.legs()
    r.torso("leather")
    # Moss cloak hanging from the shoulders (back view shows it whole).
    if v == "up":
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx1 + 1, r.yhip + r.px(5)), "moss")
        cv.part(rect(r.cx, r.ysh + 1, r.cx, r.yhip + r.px(5)), "moss", flat="d")
    elif v == "side":
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx0 + 1, r.yhip + r.px(4)), "moss")
    else:
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx0, r.yhip + r.px(3)), "moss")
        cv.part(rect(r.tx1, r.ysh, r.tx1 + 1, r.yhip + r.px(3)), "moss")
    _belt(r, "leather", r.yhip - r.px(2), "brass.m")
    r.arms("moss", glove="leather")
    r.head(hood="moss")
    if r.battle:
        hx, hy = r.hand_r
        bow = {(hx + 3 - int(abs(y - hy) * 0.22), y) for y in range(hy - 14, hy + 13)}
        cv.part(bow, "wood", flat="m")
        cv.part(line(hx + 3 - 3, hy - 14, hx + 3 - 3, hy + 12), "bone", flat="d")
    # Long knife on the belt.
    if v != "up" and not r.battle:
        kx = r.tx1 - r.px(1) if v == "down" else r.tx0 + r.px(1)
        cv.part(line(kx, r.yhip - r.px(2), kx, r.yhip + r.px(3)), "steel", flat="m")


def draw_cleric(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    r.legs()
    r.torso("iron")
    # Undyed wool tabard over mail with brass trim.
    hem = r.yhip + r.px(4)
    tab = rect(r.tx0 + (1 if v != "side" else 0), r.ysh + 1, r.tx1 - (1 if v != "side" else 0), hem)
    cv.part(tab, "wool")
    if v != "up":
        cv.part({(x, hem) for x, _ in tab}, "brass", flat="m")
    _belt(r, "leather", r.yhip - r.px(1), None)
    r.arms("iron")
    if v == "down":  # iron holy symbol on a cord
        cv.put(r.cx, r.ysh + r.px(3), "steel.l")
        cv.put(r.cx, r.ysh + r.px(4), "steel.l")
        cv.put(r.cx - 1, r.ysh + r.px(4) - (1 if k == 1 else 0), "steel.l")
        cv.put(r.cx + 1, r.ysh + r.px(4) - (1 if k == 1 else 0), "steel.l")
    r.head(hair="leather")
    if r.battle:  # round shield on the near forearm, rim and boss in brass
        sx, sy = r.tx1, r.ysh + 11
        sh = ellipse(sx, sy, 2.6, 7)
        cv.part(sh, "wood")
        cv.part({(x, y) for x, y in sh if x == sx - 2}, "wood", flat="l")
        cv.part({(x, y) for x, y in sh if x >= sx + 2}, "brass", flat="m")
        cv.fill({(sx, sy), (sx, sy - 1), (sx, sy + 1)}, "brass.l")
    # Mace in the hand, head up.
    if v == "down":
        hx, hy = r.hand_r
        hx += 1
    elif v == "side":
        hx, hy = r.hand_r
    else:
        hx, hy = r.hand_r
        hx += 1
    cv.part(thick(line(hx, hy, hx, hy - r.px(9)), 1), "wood", flat="m")
    cv.part(rect(hx - 1, hy - r.px(12), hx + (1 if k == 1 else 2), hy - r.px(9)), "steel")


def _robe(r, ramp, trim, flare=0.22, hem_y=None):
    S, cv, v = r.S, r.cv, r.view
    hem_y = hem_y or r.fy - 1
    pixels = set()
    for y in range(r.ysh, hem_y + 1):
        grow = max(0, y - r.yhip + S.T // 2) * flare
        x0 = r.tx0 - round(grow * (0.5 if v == "side" else 1))
        x1 = r.tx1 + round(grow * (0.5 if v == "side" else 1))
        sway = r.pose["lift"][1] - r.pose["lift"][0] if y > r.yhip else 0
        pixels |= {(x + (sway if y > hem_y - 2 else 0), y) for x in range(x0, x1 + 1)}
    cv.part(pixels, ramp)
    hem = {(x, y) for x, y in pixels if y >= hem_y}
    cv.part(hem, trim, flat="m")
    # Feet peek below the hem.
    foot = r.fy
    if v == "side":
        cv.part(rect(r.cx - 1, foot, r.cx + r.px(2), foot), r.spec["boots"], flat="m")
    else:
        cv.part({(r.cx - 2 * r.px(1), foot), (r.cx - r.px(1), foot), (r.cx + r.px(1), foot),
                 (r.cx + 2 * r.px(1) - 1, foot)}, r.spec["boots"], flat="m")
    return pixels


def draw_wizard(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    # Tall gnarled staff, with a dim pale stone, held on the near side.
    if r.battle:
        sx = r.hand_r[0]
    elif v == "side":
        sx = r.cx + r.px(4)
    elif v == "down":
        sx = r.tx1 + S.aw + 1
    else:
        sx = r.tx0 - S.aw - 1
    stone_y = r.top if k == 1 else r.top - 3
    st = {(sx + (1 if (y // r.px(5)) % 2 else 0), y) for y in range(stone_y, r.fy)}
    cv.part(thick(st, 1 if k == 1 else 2), "wood", flat="m")
    stone = ellipse(sx + (0 if k == 1 else 1), stone_y, r.px(1.4), r.px(1.4))
    cv.part(stone, "bone", flat="m")
    cv.fill({(sx, stone_y)}, "wool.l")
    _robe(r, "slate", "slate.d")
    if v == "down":
        cv.part(rect(r.cx, r.ysh + 1, r.cx, r.fy - 2), "slate", flat="d")
    _belt(r, "leather", r.yhip - r.px(2))
    r.arms("slate", glove="skin")
    r.head(hood="slate", beard="bone")
    # Hood falls over the shoulders.
    cv.part(rect(r.tx0, r.ysh, r.tx1, r.ysh + r.px(1)), "slate", flat="d")


def draw_witch(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    _robe(r, "heather", "ashmoss", flare=0.26)
    # Layered shawl across the shoulders and a ragged ash-moss hem.
    shawl = rect(r.tx0 - 1, r.ysh, r.tx1 + 1, r.ysh + r.px(4))
    if v == "side":
        shawl = rect(r.tx0 - 1, r.ysh, r.tx1 + 1, r.ysh + r.px(4))
    cv.part(shawl, "ashmoss")
    if v != "up":
        cv.part({(x, r.ysh + r.px(4) + (x % 2)) for x in range(r.tx0, r.tx1 + 1)}, "ashmoss", flat="d")
    _belt(r, "leather", r.yhip - r.px(1))
    r.arms("heather", glove="skin")
    r.head(hood="heather")
    # Bone and herb charms on cords.
    if v == "down":
        for dx, ramp in ((-2, "bone.l"), (0, "moss.m"), (2, "bone.l")):
            cv.put(r.cx + dx * (1 if k == 1 else 2) // (2 if k == 1 else 1) - (0 if dx else 0),
                   r.yhip - r.px(3) + (abs(dx) // 2), ramp)
    elif v == "side":
        cv.put(r.cx + 1, r.yhip - r.px(3), "bone.l")
    # Crooked ash wand.
    if v == "side":
        hx, hy = r.hand_r
    elif v == "down":
        hx, hy = r.hand_r
        hx += 1
    else:
        hx, hy = r.hand_r
        hx += 1
    wand = line(hx, hy, hx + r.px(1), hy - r.px(4)) | line(hx + r.px(1), hy - r.px(4), hx, hy - r.px(8))
    cv.part(wand, "wood", flat="m")
    cv.put(hx, hy - r.px(8), "bone.l")


def draw_adventurer(r):
    S, cv, v, k = r.S, r.cv, r.view, r.k
    # Walking staff.
    if r.battle:
        sx = r.hand_r[0]
    elif v == "side":
        sx = r.cx + r.px(4)
    elif v == "down":
        sx = r.tx1 + S.aw + 1
    else:
        sx = r.tx1 + S.aw + 1
    cv.part(thick({(sx, y) for y in range(r.top + r.px(2), r.fy)}, 1 if k == 1 else 2), "wood", flat="m")
    # Pack on the back.
    if v == "side":
        cv.part(rect(r.tx0 - r.px(3), r.ysh + 1, r.tx0 - 1, r.ysh + r.px(7)), "leather")
    elif v == "up":
        cv.part(rect(r.tx0 + 1, r.ysh + 1, r.tx1 - 1, r.ysh + r.px(8)), "leather")
        cv.part(rect(r.tx0 + 1, r.ysh + r.px(2), r.tx1 - 1, r.ysh + r.px(2)), "leather", flat="d")
    r.legs()
    r.torso("wool")
    # Plain brown traveler's cloak.
    if v == "up":
        pass
    elif v == "side":
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx0 + 1, r.yhip + r.px(3)), "leather")
    else:
        cv.part(rect(r.tx0 - 1, r.ysh, r.tx0, r.yhip + r.px(3)), "leather")
        cv.part(rect(r.tx1, r.ysh, r.tx1 + 1, r.yhip + r.px(3)), "leather")
    _belt(r, "leather", r.yhip - r.px(2))
    r.arms("leather", glove="skin")
    r.head(hood="leather")


DRAWERS = {
    "warrior": draw_warrior, "rogue": draw_rogue, "ranger": draw_ranger,
    "cleric": draw_cleric, "wizard": draw_wizard, "witch": draw_witch,
    "adventurer": draw_adventurer,
}


def render(cls, view, S, pose=None):
    """One frame: a Canvas of S.W x S.W with the figure outlined."""
    r = Rig(cls, view, S, pose or {})
    DRAWERS[cls](r)
    r.cv.outline()
    return r.cv


def map_idle(cls):
    return [[render(cls, v, MAP, p) for p in IDLE_POSES] for v in ("down", "up", "side")]


def map_walk(cls):
    return [[render(cls, v, MAP, p) for p in WALK_POSES] for v in ("down", "up", "side")]


# Per-class battle stance tweaks over the shared braced stance.
BATTLE_STANCE = {"rogue": dict(bob=3, stride=(-4, 3))}
BREATH = [0, 1, 2, 1]  # four-frame idle: the body settles and rises


def _extras_ranger(r, t):
    """Arrow nocked low on the string, point toward the enemy."""
    hx, hy = r.hand_r
    cv = r.cv
    cv.part(line(hx - 1, hy + 2, hx + 9, hy + 6 + (t == 2)), "bone", flat="m")
    cv.fill({(hx + 10, hy + 6 + (t == 2)), (hx + 11, hy + 7 + (t == 2))}, "steel.l")
    cv.fill({(hx - 2, hy + 2), (hx - 2, hy + 3)}, "oxblood.m")


def _extras_wizard(r, t):
    """A dim pale glow that swells and fades at the staff stone."""
    cv = r.cv
    sx = r.hand_r[0] + 1
    sy = r.top - 3
    ring = [(-2, 0), (2, 0), (0, -2), (0, 2)]
    wide = [(-3, -1), (3, -1), (-1, -3), (1, -3), (-3, 1), (3, 1)]
    if t in (1, 3):
        for dx, dy in ring:
            if cv.get(sx + dx, sy + dy) is None:
                cv.put(sx + dx, sy + dy, "bone.d")
    if t == 2:
        for dx, dy in ring:
            if cv.get(sx + dx, sy + dy) is None:
                cv.put(sx + dx, sy + dy, "bone.m")
        for dx, dy in wide:
            if cv.get(sx + dx, sy + dy) is None:
                cv.put(sx + dx, sy + dy, "bone.d")
    cv.fill({(sx, sy), (sx + 1, sy)}, "wool.l" if t == 2 else "bone.l")


def _extras_witch(r, t):
    """A thin curl of grave-mist rising from the ground beside the hem."""
    cv = r.cv
    x0 = r.tx0 - 8
    base = r.fy - 1
    for i in range(14):
        wob = round(2.2 * math.sin((i + t * 2.5) / 2.2))
        c = "ashmoss.l" if i % 3 else "bone.d"
        if i > 9:
            c = "ashmoss.m"
        cv.put(x0 + wob, base - i, c)
        if i % 4 == 1:
            cv.put(x0 + wob + 1, base - i, "ashmoss.m")


def _extras_cleric(r, t):
    """The iron holy symbol glints on the tabard in the third frame."""
    if t == 2:
        r.cv.fill({(r.cx + 2, r.ysh + 5), (r.cx + 2, r.ysh + 4), (r.cx + 1, r.ysh + 5)}, "steel.l")


EXTRAS = {"ranger": _extras_ranger, "wizard": _extras_wizard, "witch": _extras_witch,
          "cleric": _extras_cleric}


def battle_idle(cls, t=0):
    """One frame (t = 0..3) of the 64x64 battle idle, facing right."""
    base = {**dict(bob=0, stride=(-3, 3)), **BATTLE_STANCE.get(cls, {})}
    pose = dict(base, bob=base["bob"] + BREATH[t % 4])
    r = Rig(cls, "side", BATTLE, pose)
    DRAWERS[cls](r)
    if cls in EXTRAS:
        EXTRAS[cls](r, t % 4)
    r.cv.outline()
    return r.cv


def battle_idle_frames(cls):
    return [battle_idle(cls, t) for t in range(4)]
