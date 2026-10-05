"""16x16 sample icons, pre-rendered, with a dark outline for UI legibility."""
import math
import numpy as np
import render3d as r3
from render3d import Part, frame_y, norm, rgb

INK = rgb(r3.RAMPS["eye"][0])


def rotz(a):
    c, s_ = math.cos(a), math.sin(a)
    return np.array([[c, -s_, 0], [s_, c, 0], [0, 0, 1.0]])


def outline(arr):
    a = arr[:, :, 3] > 0
    o = np.zeros_like(a)
    o[1:, :] |= a[:-1, :]; o[:-1, :] |= a[1:, :]; o[:, 1:] |= a[:, :-1]; o[:, :-1] |= a[:, 1:]
    o &= ~a
    out = arr.copy()
    out[o, :3] = INK
    out[o, 3] = 255
    return out


def icon(parts, pitch=20, yaw=0, scale=1.0, anchor=(8, 8), light=(-0.6, 0.7, 0.6), amb=0.35):
    a = r3.render(parts, 16, 16, r3.Camera(yaw, pitch, scale=scale), anchor, ambient=amb, dither=0.3,
                  light=light)
    return outline(a)


def water():
    p = [Part("ellip", "water_m", c=(0, -1.0, 0), r=(3.6, 3.4, 3.6)),
         Part("cap", "water_m", blend=1.6, a=(0, -0.5, 0), b=(0, 5.0, 0), r1=3.0, r2=0.25),
         Part("torus", "water_m", c=(0, -5.2, 0), R=None, rad=5.6, r=0.45)]
    return icon(p, pitch=25, anchor=(8, 7.5))


def forage():
    p = [Part("cap", "wood", a=(-5, -5, 0), b=(1, 2, 0), r1=0.45),
         Part("cap", "wood", a=(1, 2, 0), b=(4, 5.5, 0), r1=0.4)]
    for c, rr in (((2.2, -2.0, 1.0), 1.9), ((-0.6, -3.4, 1.5), 1.8), ((0.8, -5.0, -0.6), 1.7),
                  ((3.8, -4.4, 0.4), 1.6)):
        p.append(Part("ellip", "berry", c=c, r=(rr,) * 3))
    for c, ang in (((-2.6, 3.0, -0.5), 0.7), ((4.2, 5.2, -0.5), -0.3)):
        p.append(Part("ellip", "leaf", c=c, R=rotz(ang), r=(3.6, 1.7, 0.6)))
    return icon(p, pitch=10, anchor=(8, 8))


def bleeding():
    p = [Part("ellip", "blood", c=(-0.8, -2.0, 0), r=(3.4, 3.4, 3.0)),
         Part("cap", "blood", blend=1.6, a=(-0.8, -1.4, 0), b=(-0.8, 5.2, 0), r1=2.8, r2=0.2),
         Part("box", "steel", c=(1.2, 0, 3.2), R=frame_y(norm((1, 1, 0))), b=(0.5, 7.4, 0.3))]
    return icon(p, anchor=(8, 8))


def stunned():
    R = frame_y(norm((0.0, 1.0, 0.25)))
    p = [Part("torus", "bone", c=(0, 0, 0), R=R, rad=6.0, r=0.35)]
    for a in (0.4, 2.5, 4.4):
        c = (math.cos(a) * 6.0, 0, math.sin(a) * 6.0)
        p.append(Part("ellip", "yellow_eye", c=c, r=(1.5, 1.5, 1.5)))
        for ax in ((2.4, 0.4, 0.4), (0.4, 2.4, 0.4)):
            p.append(Part("ellip", "yellow_eye", c=c, r=ax))
    return icon(p, pitch=30, anchor=(8, 9))


def fighter():
    p = []
    for s in (1, -1):
        d = norm((s, 1.0, 0))
        hand = np.array([-s * 4.4, -4.4, 0.0])
        p += [Part("box", "steel", c=hand + d * 7.0, R=frame_y(d), b=(0.8, 5.2, 0.25), rr=0.1),
              Part("box", "brass", c=hand + d * 1.4, R=frame_y(norm((-d[1], d[0], 0))), b=(0.5, 2.2, 0.5)),
              Part("cap", "leather", a=hand - d * 1.0, b=hand + d * 1.0, r1=0.6),
              Part("ellip", "brass", c=hand - d * 1.8, r=(0.8,) * 3)]
    return icon(p, pitch=0, anchor=(8, 7.5))


def healer():
    p = [Part("ellip", "skin", c=(0, -3.2, 0), r=(3.2, 2.6, 1.2)),
         Part("cap", "wool", a=(0, -7.8, 0), b=(0, -5.8, 0), r1=2.4)]
    for i, x in enumerate((-2.3, -0.8, 0.8, 2.3)):
        p.append(Part("cap", "skin", blend=0.4, a=(x, -1.6, 0), b=(x * 1.15, 1.6 - abs(i - 1.5) * 0.5, 0.8), r1=0.75))
    p.append(Part("cap", "skin", blend=0.4, a=(-2.8, -3.4, 0.3), b=(-5.0, -1.2, 1.2), r1=0.8))
    p.append(Part("ellip", "heal_glow", c=(0.0, 4.6, 1.0), r=(2.4, 2.4, 2.4)))
    return icon(p, pitch=0, anchor=(8, 8))


def yield_():
    p = [Part("cap", "wood", a=(-7, 7, 0), b=(6, -7, 0), r1=0.5)]
    p.append(Part("box", "cloth_white", c=(0.6, -1.2, 0.6), R=rotz(-0.55), b=(2.0, 3.6, 0.3), rr=0.3,
                  folds=(4, 0.3, 0.0, -3.0, -8, 4)))
    return icon(p, pitch=0, anchor=(8, 8))


def dark():
    out = np.zeros((16, 16, 4), np.uint8)
    from scenes2 import SCENE
    night = SCENE["night"]
    for y in range(1, 15):
        for x in range(1, 15):
            out[y, x, :3] = rgb(night[1 if (x + y) % 7 else 0])
            out[y, x, 3] = 255
    moon = r3.render([Part("ellip", "bone", c=(0, 0, 0), r=(4.4,) * 3)], 16, 16, r3.Camera(0, 0), (8, 8),
                     ambient=0.4, light=(-0.9, 0.3, 0.3), dither=0.3)
    for y in range(16):
        for x in range(16):
            if moon[y, x, 3] and math.hypot(x - 10.0, y - 6.4) > 4.2:
                out[y, x] = moon[y, x]
    for x, y in ((3, 3), (12, 12), (3, 12)):
        out[y, x, :3] = rgb(night[3])
    return outline(out) if False else out


ORDER = [("water", water), ("forage", forage), ("bleeding", bleeding), ("stunned", stunned),
         ("fighter", fighter), ("healer", healer), ("yield", yield_), ("dark", dark)]
