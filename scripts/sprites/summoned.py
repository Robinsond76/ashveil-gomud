"""Summoned battle units (art set S5): a Hierarch's Angel and a Demonologist's
Demon (docs/plans/2026-10-06-phase-38b-promotions-talents.md).  Large units:
drawn on a 96 px canvas and shrunk 3/4 by roster.py, side-on, facing right,
four idle frames.  Light and fire only use the master palette's bone, wool,
brass and ember ramps."""
from kit import BREATH, SWAY, limb, shade_pixels, noise, poly
from pixels import Canvas, rect, line, thick, ellipse


def _flames(cv, pixels, t, seed, tones, density=0.16):
    """Lick a few flame pixels onto the empty cells touching a figure's edge."""
    for x, y in sorted(pixels):
        for nx, ny in ((x, y - 1), (x - 1, y), (x + 1, y)):
            if cv.get(nx, ny) is None and noise(nx + t * 3, ny - t * 2, seed) < density:
                cv.put(nx, ny, tones[int(noise(nx, ny, seed + 9) * len(tones))])


def angel(t, W=96):
    """White fire in the shape of a warrior: folded wings, a blade of light."""
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    top = fy - 74 + b
    hip = fy - 36 + b // 2
    # wings, folded and swept back behind the shoulders (drawn first, shaded)
    wing_back = poly([(34, top + 14), (14, top + 30 + SWAY[t]), (10, fy - 22), (18, fy - 8 + SWAY[t]),
                      (28, fy - 24), (36, top + 40)])
    wing_near = poly([(36, top + 12), (22, top + 26), (20, fy - 30), (27, fy - 14 + SWAY[t]),
                      (34, fy - 34), (40, top + 36)])
    cv.part(wing_back, "bone")
    shade_pixels(cv, wing_back)
    cv.part(wing_near, "wool")
    for i in range(5):  # feather rows
        y = top + 24 + i * 9
        cv.fill(line(20 + i, y, 36, y - 4), "bone.d")
    # far leg, then near leg: greaves in pale steel, bare feet of light
    far = limb((40, hip + 2), (36, fy - 3), 6)
    cv.part(far, "steel")
    shade_pixels(cv, far)
    leg = limb((47, hip + 2), (50, fy - 3), 7)
    cv.part(leg, "steel")
    cv.part(rect(46, fy - 3, 57, fy), "bone")
    cv.part(rect(32, fy - 3, 43, fy), "bone", flat="d")
    # tassets and a brass-hemmed tabard of white fire
    skirt = poly([(36, hip - 4), (58, hip - 4), (62, hip + 22), (34, hip + 22)])
    cv.part(skirt, "wool", flat="l")
    cv.part({(x, hip + 22) for x in range(34, 63)}, "brass", flat="m")
    cv.part({(x, hip - 3) for x in range(37, 58)}, "brass", flat="m")
    # torso: broad cuirass with a bright core
    torso = poly([(35, top + 14), (60, top + 14), (58, hip - 2), (38, hip - 2)])
    cv.part(torso, "wool")
    cv.part(ellipse(48, top + 26, 4, 5), "bone", flat="l")
    cv.put(48, top + 26, "ember.l")
    # near arm raising the blade up and forward
    hx, hy = 66, top + 38
    arm = limb((52, top + 18), (hx - 2, hy), 6)
    cv.part(arm, "bone")
    cv.part({(x, y) for x, y in arm if y < top + 24}, "brass", flat="m")  # pauldron
    cv.part(ellipse(hx, hy, 3.2, 3.2), "wool")
    # blade of light, a hand's width, with a pale hot tip
    tip = (78 + SWAY[t], top - 8)
    blade = limb((hx, hy - 2), tip, 3)
    cv.part(blade, "wool", flat="l")
    cv.part(line(hx, hy - 2, tip[0], tip[1]), "bone", flat="l")
    cv.fill({(tip[0], tip[1] - 1), (tip[0] + 1, tip[1])}, "ember.l")
    cv.part(rect(hx - 5, hy - 2, hx + 5, hy - 1), "brass", flat="m")  # crossguard
    # head: bare, bright, a thin halo above
    hd = (50, top + 6)
    head = ellipse(hd[0], hd[1], 7, 8)
    cv.part(head, "wool")
    cv.part({(x, y) for x, y in head if y < hd[1] - 4}, "bone", flat="l")
    cv.fill({(hd[0] + 4, hd[1] - 1), (hd[0] + 5, hd[1] - 1)}, "ember.l")
    for x in range(hd[0] - 6, hd[0] + 7):
        y = top - 4 - (1 if abs(x - hd[0]) > 4 else 0)
        cv.put(x, y, "brass.l" if t == 2 else "brass.m")
    cv.outline()
    _flames(cv, {(x, y) for y in range(cv.h) for x in range(cv.w) if cv.get(x, y) == "outline"},
            t, 5, ["ember.l", "bone.l", "ember.l"], 0.3)
    return cv


def demon(t, W=96):
    """A hunched black fiend, a fire in its chest, chains of red light from its wrists."""
    cv = Canvas(W, W)
    fy = W - 3
    b = BREATH[t]
    hip = fy - 28
    # far leg and arm (behind, shaded)
    far_leg = limb((50, hip), (54, fy - 4), 9)
    cv.part(far_leg, "charcoal")
    cv.part(rect(52, fy - 4, 66, fy), "iron")
    shade_pixels(cv, far_leg | rect(52, fy - 4, 66, fy))
    leg = limb((38, hip), (34, fy - 4), 10)
    cv.part(leg, "charcoal")
    cv.part(rect(30, fy - 4, 48, fy), "iron")
    cv.part({(x, fy - 5) for x in range(31, 48, 4)}, "bone", flat="l")  # claws
    # back: hunched, the shoulders higher than the head
    top = hip - 42 + b
    back = ellipse(36, top + 20, 14, 18) | ellipse(40, hip - 10, 14, 15)
    belly = ellipse(52, hip - 6, 9, 11)
    cv.part(back | belly, "charcoal")
    cv.part({(x, y) for x, y in back if x < 30}, "charcoal", flat="l")
    # ribs and shoulder spines
    for i in range(4):
        cv.put(56 - i, top + 18 + i * 5, "iron.m")
        cv.put(55 - i, top + 18 + i * 5, "iron.m")
    for i, (x, y) in enumerate(((26, top + 8), (32, top + 2), (39, top - 1))):
        cv.part(line(x, y + 4, x - 2, y - 2), "bone", flat="m")
    # long near arm hanging forward, clawed; a shackle at the wrist
    wrist = (68 + SWAY[t], top + 46)
    arm = limb((46, top + 12), wrist, 8)
    cv.part(arm, "charcoal", flat="m")
    cv.part({(x, y) for x, y in arm if y < top + 20}, "charcoal", flat="l")
    cv.part(rect(wrist[0] - 3, wrist[1] - 3, wrist[0] + 3, wrist[1] + 2), "iron")
    for i in range(3):
        cv.part(line(wrist[0] + 3, wrist[1] + 1 + i, wrist[0] + 9, wrist[1] + 3 + i + SWAY[t]), "bone", flat="l")
    # the fire in its chest: a pulse of ember
    cx, cy = 53, top + 22
    glow = ellipse(cx, cy, 6, 7)
    cv.part(glow, "ember", flat="m")
    cv.part({(x, y) for x, y in glow if (x + y + t) % 3 == 0}, "ember", flat="l")
    cv.fill({(cx, cy - 2 - (t == 2)), (cx + 1, cy - 4 - (t == 2))}, "ember.l")
    # chains of red light trailing from the wrist to the ground
    for i in range(10):
        x = wrist[0] - 1 + (i // 4) * 2
        y = wrist[1] + 3 + i * 3
        if y >= fy - 1:
            break
        cv.put(x, y, "ember.m" if (i + t) % 2 else "ember.l")
        cv.put(x + 1, y + 1, "ember.d")
    # head: low and forward, horns swept back, ember eyes
    hd = [0, 1, 1, 0][t]
    hx0, hy0 = 60, top + 10 + hd
    head = ellipse(hx0, hy0, 9, 8)
    cv.part(head, "charcoal")
    cv.part({(x, hy0 - 3) for x in range(hx0 - 2, hx0 + 8)}, "charcoal", flat="d")  # brow
    for sgn_y, ln in ((hy0 - 7, 12), (hy0 - 4, 9)):
        cv.part(thick(line(hx0 - 2, sgn_y, hx0 - 2 - ln, sgn_y - 8), 2), "bone", flat="m")
    cv.fill({(hx0 + 4, hy0 - 1), (hx0 + 5, hy0 - 1)}, "ember.m" if t != 2 else "ember.l")
    cv.part(rect(hx0 + 3, hy0 + 3, hx0 + 9, hy0 + 4), "oxblood", flat="d")  # mouth
    cv.put(hx0 + 4, hy0 + 2, "bone.l")
    cv.put(hx0 + 8, hy0 + 2, "bone.l")
    cv.outline()
    return cv
