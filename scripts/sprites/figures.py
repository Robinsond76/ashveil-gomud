"""3D figure models for the pre-rendered sprites.

Model space: +x is the facing direction, +y is up, +z is the figure's left
(the side turned toward the camera). Feet stand on y = 0; an adult is about
58 units (pixels) tall, roughly seven heads.
"""
import numpy as np
from render3d import Part, frame_y, norm

V = np.array


def leg(mat, hip, knee, ankle, r=(2.7, 2.2, 1.7), boot="leather", boot_len=3.4, shaft=8.0):
    hip, knee, ankle = V(hip, float), V(knee, float), V(ankle, float)
    parts = [Part("cap", mat, blend=0.6, a=hip, b=knee, r1=r[0], r2=r[1]),
             Part("cap", mat, blend=0.6, a=knee, b=ankle, r1=r[1], r2=r[2])]
    if boot:
        parts += [Part("cap", boot, blend=0.4, a=ankle + V([0, -1.0, 0]), b=ankle + V([0, shaft - 3.5, 0]),
                       r1=r[2] + 0.35, r2=r[2] + 0.45),
                  Part("ellip", boot, blend=0.6, c=(ankle[0] + 1.4, 1.9, ankle[2]), r=(boot_len, 2.2, 1.9),
                       clip=[((0, 1, 0), 0.0)])]
    return parts


def arm(mat, shoulder, elbow, hand, r=(2.0, 1.7, 1.4), skin="skin", hand_r=1.45):
    shoulder, elbow, hand = V(shoulder, float), V(elbow, float), V(hand, float)
    return [Part("ellip", mat, blend=0.8, c=shoulder, r=(2.6, 2.6, 2.6)),
            Part("cap", mat, blend=0.6, a=shoulder, b=elbow, r1=r[0], r2=r[1]),
            Part("cap", mat, blend=0.6, a=elbow, b=hand, r1=r[1], r2=r[2]),
            Part("ellip", skin, blend=0.4, c=hand, r=(hand_r, hand_r * 1.1, hand_r))]


def head(skin="skin", tilt=0.0, beard=None, beard_long=False, eyes="eye", c=(0.6, 54.2, 0)):
    cx, cy, cz = c
    p = [Part("cap", skin, blend=0.8, a=(cx - 0.6, cy - 6.2, 0), b=(cx - 0.2, cy - 3.0, 0), r1=1.9, r2=1.8),
         Part("ellip", skin, blend=0.6, c=c, r=(3.8, 4.4, 3.6)),
         Part("ellip", skin, blend=0.5, c=(cx + 3.5, cy - 0.6, cz), r=(0.9, 1.3, 0.7)),
         Part("ellip", skin, blend=0.4, c=(cx + 2.6, cy + 1.5, cz), r=(1.4, 0.8, 3.0)),  # brow
         Part("ellip", eyes, c=(cx + 3.15, cy + 0.5, cz + 1.4), r=(0.6, 0.55, 0.6)),
         Part("ellip", eyes, c=(cx + 3.15, cy + 0.5, cz - 1.4), r=(0.6, 0.55, 0.6)),
         Part("ellip", "hair", c=(cx + 3.4, cy - 2.2, cz), r=(0.5, 0.35, 1.2))]  # mouth line
    if beard:
        if beard_long:
            p.append(Part("cap", beard, blend=0.8, a=(cx + 1.8, cy - 2.6, cz), b=(cx + 2.4, cy - 10.5, cz),
                          r1=2.6, r2=1.0, sq=((0, 0, 1), 1.0)))
        else:
            p.append(Part("ellip", beard, blend=0.6, c=(cx + 1.8, cy - 3.0, cz), r=(2.3, 2.3, 2.8)))
    return p


def blade(hand, direction, length, width=1.0, mat="steel", guard="brass", pommel="brass"):
    hand, d = V(hand, float), norm(direction)
    R = frame_y(d)
    side = norm(np.cross(d, (0, 1, 0))) if abs(d[1]) < 0.95 else V([0, 0, 1.0])
    return [Part("box", mat, c=hand + d * (2.0 + length / 2), R=R, b=(width, length / 2, 0.22), rr=0.1),
            Part("box", guard, c=hand + d * 1.6, R=frame_y(side), b=(0.45, 2.6, 0.45), rr=0.15),
            Part("cap", "leather", a=hand - d * 1.2, b=hand + d * 1.2, r1=0.6),
            Part("ellip", pommel, c=hand - d * 2.0, r=(0.8, 0.8, 0.8))]


def staff(bottom, top, mat="wood", r=0.85):
    return [Part("cap", mat, a=bottom, b=top, r1=r, r2=r * 0.9)]


def body_core(mat, skirt=None, skirt_y0=17.0, skirt_r0=7.0, chest=(3.8, 7.4, 6.2)):
    p = [Part("ellip", mat, blend=1.2, c=(0, 41, 0), r=chest),
         Part("ellip", mat, blend=1.2, c=(0, 32.5, 0), r=(3.6, 4.2, 5.2))]
    if skirt:
        p.append(Part("cone", skirt, blend=0.8, c=(0, 0, 0), y0=skirt_y0, y1=34.5, r0=skirt_r0, r1=5.6, sx=0.68,
                      folds=(7, 0.45, 0, 0, skirt_y0, 30)))
    return p


# --- Classes --------------------------------------------------------------------
def warrior():
    p = []
    p += leg("hair", (0.6, 29, 3.0), (2.8, 16, 4.6), (3.0, 3.5, 5.6))
    p += leg("hair", (-0.6, 29, -3.0), (-0.8, 16, -4.4), (-1.4, 3.5, -5.4))
    p += body_core("mail", skirt="mail", chest=(4.2, 7.4, 6.6))
    # oxblood surcoat over the hauberk, open at the sides
    p.append(Part("cone", "oxblood", c=(0, 0, 0), y0=18.5, y1=47.5, r0=7.6, r1=6.7, sx=0.64,
                  folds=(9, 0.35, 0, 0, 18.5, 30), clip=[((0, 0, -1), -4.0), ((0, 0, 1), -4.0)]))
    p.append(Part("ellip", "leather", c=(0, 34.6, 0), r=(5.0, 0.9, 6.0)))
    p.append(Part("box", "brass", c=(5.0, 34.6, 0), R=None, b=(0.4, 0.8, 0.9), rr=0.1))
    p.append(Part("ellip", "mail", blend=0.8, c=(0.2, 49.6, 0), r=(3.6, 2.6, 4.6)))  # aventail
    p += head(beard="hair")
    p.append(Part("ellip", "iron", c=(0.4, 55.4, 0), r=(4.1, 4.5, 3.9), clip=[((0, 1, 0), 54.4)]))
    p.append(Part("ellip", "iron", c=(0.4, 54.8, 0), r=(4.4, 0.75, 4.2)))
    p.append(Part("box", "iron", c=(4.35, 53.4, 0), b=(0.35, 1.7, 0.45), rr=0.1))
    # sword arm (far), blade low and forward
    p += arm("mail", (0, 47, -6.3), (1.0, 39.5, -7.6), (5.0, 35.5, -6.8))
    p += blade((5.0, 35.5, -6.8), (1, -0.62, 0.15), 17, width=1.3)
    # shield arm (near), round shield held forward
    p += arm("mail", (0, 47, 6.3), (2.0, 40, 8.2), (5.0, 41, 9.2))
    p += round_shield(V([5.6, 40.0, 10.4]), (0.55, 0.06, 0.83), 7.6)
    return p


def round_shield(c, normal, rad, face="wood", rim="iron", boss="brass", band="oxblood"):
    n = norm(normal)
    R = frame_y(n)
    p = [Part("disc", face, c=c, R=R, rad=rad - 0.4, h=0.45, rr=0.35),
         Part("torus", rim, c=c, R=R, rad=rad - 0.3, r=0.75),
         Part("ellip", boss, c=c + n * 0.9, r=(1.7, 1.7, 1.7))]
    if band:
        up = R[:, 0]
        p.append(Part("box", band, c=c + n * 0.55, R=R, b=(rad - 1.2, 0.12, 1.3), rr=0.0))
    return p


def oval_shield(c, normal, w, h, face="iron", rim="brass", flat_top=True):
    n = norm(normal)
    right = norm(np.cross((0, 1, 0), n))
    up = np.cross(n, right)
    R = np.stack([right, up, n], 1)
    c = V(c, float)
    clip = [((0, -1, 0), -(c[1] + h * 0.62))] if flat_top else []
    return [Part("ellip", rim, c=c, R=R, r=(w, h, 0.75), clip=clip),
            Part("ellip", face, c=c + n * 0.25, R=R, r=(w - 0.8, h - 0.8, 0.7), clip=clip),
            Part("box", rim, c=c + n * 0.75, R=R, b=(0.45, h * 0.62, 0.2)),
            Part("box", rim, c=c + n * 0.75 + up * 1.2, R=R, b=(w * 0.75, 0.45, 0.2))]


def hood(mat, c=(-1.3, 55.0, 0), r=(4.7, 5.3, 4.5), peak=True):
    cx, cy, cz = c
    p = [Part("ellip", mat, blend=0.8, c=c, r=r)]
    if peak:
        p.append(Part("cap", mat, blend=1.0, a=(cx - 0.5, cy + 3.0, 0), b=(cx - 3.8, cy + 5.6, 0), r1=2.4, r2=0.6))
    p.append(Part("ellip", mat, blend=1.2, c=(cx + 0.2, cy - 5.5, 0), r=(4.4, 2.6, 6.4)))  # mantle
    return p


def cleric():
    p = []
    p += leg("hair", (0.6, 29, 3.0), (2.4, 16, 4.2), (2.6, 3.5, 5.0))
    p += leg("hair", (-0.6, 29, -3.0), (-0.8, 16, -4.2), (-1.4, 3.5, -5.0))
    p += body_core("mail", skirt="mail", chest=(4.0, 7.4, 6.4))
    p.append(Part("cone", "wool", c=(0, 0, 0), y0=17.0, y1=47.5, r0=7.8, r1=6.7, sx=0.64,
                  folds=(9, 0.4, 0, 0, 17.0, 30), clip=[((0, 0, -1), -4.3), ((0, 0, 1), -4.3)]))
    p.append(Part("ellip", "leather", c=(0, 34.6, 0), r=(5.1, 0.9, 6.2)))
    # iron holy symbol on a cord
    p.append(Part("box", "iron", c=(5.1, 41.5, 1.0), R=frame_y((0.6, 0, 0.8)), b=(0.6, 0.25, 1.7), rr=0.15))
    p.append(Part("box", "iron", c=(5.1, 42.0, 1.0), R=frame_y((0.6, 0, 0.8)), b=(1.3, 0.25, 0.5), rr=0.15))
    # mail coif around the face
    p += head()
    p.append(Part("ellip", "mail", c=(-0.4, 54.6, 0), r=(4.3, 5.4, 4.2), clip=[((-1, 0, 0), -2.2)]))
    p.append(Part("ellip", "mail", blend=1.0, c=(0.2, 49.4, 0), r=(3.8, 2.8, 5.0)))
    # mace raised in the far hand
    p += arm("mail", (0, 47, -6.3), (2.5, 41, -8.2), (5.0, 45.5, -7.6))
    p += staff(V([5.0, 41.5, -7.6]), V([5.6, 54.0, -7.4]), r=0.6)
    p.append(Part("ellip", "iron", c=(5.7, 55.5, -7.4), r=(1.9, 2.4, 1.9)))
    for ax in ((1, 0, 0), (0, 0, 1)):
        p.append(Part("box", "iron", c=(5.7, 55.5, -7.4), R=frame_y((0, 1, 0)),
                      b=(2.6 if ax[0] else 0.35, 1.8, 2.6 if ax[2] else 0.35), rr=0.1))
    # heater shield carried low on the near arm
    p += arm("mail", (0, 47, 6.3), (1.6, 40, 8.0), (4.5, 37.5, 8.8))
    p += oval_shield((5.2, 35.5, 10.0), (0.55, 0.05, 0.83), 4.6, 6.6)
    return p


def wizard():
    p = []
    p.append(Part("cone", "slate", c=(0, 0, 0), y0=0.5, y1=48.0, r0=9.5, r1=6.0, sx=0.82,
                  folds=(10, 0.6, 0, 0, 0.5, 40)))
    p.append(Part("ellip", "leather", c=(4.5, 1.6, 2.6), r=(3.0, 1.8, 1.8), clip=[((0, 1, 0), 0.0)]))
    p.append(Part("ellip", "slate", blend=1.0, c=(0, 42, 0), r=(4.2, 7.0, 6.6)))
    p.append(Part("ellip", "brass", c=(0, 33.5, 0), r=(5.6, 0.7, 7.0)))
    p += head(skin="skin_pale", beard="beard_grey", beard_long=True)
    p += hood("slate")
    # far sleeve
    p.append(Part("cap", "slate", blend=0.6, a=(0, 47, -6.5), b=(1.5, 38, -8.5), r1=2.4, r2=3.2))
    # staff planted ahead, gnarled crown and a dim stone
    top = V([9.0, 64.0, 10.4])
    p += staff(V([9.4, 0.0, 10.8]), top, r=0.95)
    p.append(Part("cap", "wood", a=top, b=top + V([-1.6, 3.0, 0.4]), r1=0.8, r2=0.4))
    p.append(Part("cap", "wood", a=top, b=top + V([1.6, 3.2, -0.3]), r1=0.8, r2=0.4))
    p.append(Part("cap", "wood", a=top, b=top + V([0.2, 3.6, 1.4]), r1=0.7, r2=0.3))
    p.append(Part("ellip", "glow", c=top + V([0.1, 2.6, 0.3]), r=(1.6, 1.6, 1.6)))
    # near sleeve reaching to the staff
    p.append(Part("cap", "slate", blend=0.6, a=(0, 47, 6.5), b=(4.5, 40, 8.5), r1=2.4, r2=3.0))
    p.append(Part("cap", "slate", blend=0.6, a=(4.5, 40, 8.5), b=(7.8, 43.5, 10.0), r1=2.6, r2=2.4))
    p.append(Part("ellip", "skin_pale", c=(9.0, 44.0, 10.4), r=(1.5, 1.6, 1.5)))
    return p


def rogue():
    p = []
    # crouched: hips low, knees bent, weight forward
    p += leg("charcoal", (0.5, 25, 3.0), (5.0, 15, 5.8), (2.6, 3.5, 6.6))
    p += leg("charcoal", (-0.5, 25, -3.0), (1.4, 14, -6.0), (-2.4, 3.5, -6.6))
    p.append(Part("ellip", "leather", blend=1.2, c=(2.2, 36.0, 0), r=(3.6, 6.6, 5.6)))
    p.append(Part("ellip", "leather", blend=1.2, c=(0.4, 28.0, 0), r=(3.4, 4.0, 4.8)))
    p.append(Part("ellip", "plum", c=(0.9, 30.6, 0), r=(4.0, 1.0, 5.2)))
    p.append(Part("cap", "plum", a=(-2.8, 30.6, 3.0), b=(-4.6, 25.0, 3.6), r1=0.8, r2=0.6))
    # short cloak over the back
    p.append(Part("ellip", "charcoal", c=(-1.4, 37.0, 0), r=(3.4, 9.5, 7.2), clip=[((-1, 0, 0), -0.2)],
                  folds=(7, 0.4, 0, 0, 28, 38)))
    p += head(c=(4.6, 48.6, 0))
    p += hood("charcoal", c=(3.0, 49.4, 0), r=(4.6, 5.2, 4.4), peak=False)
    p.append(Part("ellip", "plum", c=(6.6, 47.0, 0), r=(2.2, 2.0, 3.4)))  # mask cloth
    # far arm, blade reversed along the forearm
    p += arm("leather", (1.6, 41.5, -6.0), (4.2, 35, -7.6), (8.4, 34.0, -6.6))
    p += blade((8.4, 34.0, -6.6), (-0.45, -0.88, 0.1), 8, width=0.9)
    # near arm low and forward
    p += arm("charcoal", (1.6, 41.5, 6.0), (5.4, 34, 7.6), (10.2, 31.0, 7.0))
    p += blade((10.2, 31.0, 7.0), (-0.35, -0.92, 0.15), 8, width=0.9)
    return p


def ranger():
    p = []
    p += leg("leather", (0.6, 29, 3.0), (2.6, 16, 4.4), (2.8, 3.5, 5.4))
    p += leg("leather", (-0.6, 29, -3.0), (-0.8, 16, -4.4), (-1.4, 3.5, -5.2))
    p += body_core("tan", chest=(3.9, 7.4, 6.3))
    p.append(Part("cone", "tan", blend=0.6, c=(0, 0, 0), y0=24.0, y1=34.5, r0=6.6, r1=5.6, sx=0.68,
                  folds=(6, 0.3, 0, 0, 24, 30)))
    p.append(Part("ellip", "leather", c=(0, 34.6, 0), r=(5.0, 0.9, 6.1)))
    p.append(Part("cap", "steel", a=(3.6, 33.6, 4.4), b=(5.6, 27.4, 5.0), r1=0.5, r2=0.3))
    # quiver on the back
    p.append(Part("cap", "leather", a=(-4.6, 34.0, -1.5), b=(-6.0, 50.0, 1.0), r1=1.8, r2=2.0))
    for dz in (-0.6, 0.6, 1.6):
        p.append(Part("box", "wool", c=(-6.2, 53.0, dz + 0.6), R=frame_y((-0.25, 1, 0.1)), b=(0.6, 1.4, 0.2)))
    # hooded cloak hanging behind
    p.append(Part("cone", "moss", c=(-1.2, 0, 0), y0=16.0, y1=48.0, r0=9.0, r1=6.8, sx=0.7,
                  folds=(8, 0.55, -1.2, 0, 16, 40), clip=[((-1, 0, 0), -0.6)]))
    p += head()
    p += hood("moss")
    # far arm drawing the string
    p += arm("tan", (0, 47, -6.3), (2.8, 40, -6.8), (7.0, 37.0, -2.0))
    # bow held low in the near hand, arrow nocked
    p += arm("moss", (0, 47, 6.3), (3.4, 40, 8.2), (10.5, 37.0, 8.4))
    pts = [V([8.4, 56.0, 8.4]), V([10.6, 47.5, 8.4]), V([11.2, 37.0, 8.4]), V([10.6, 26.5, 8.4]),
           V([8.4, 18.0, 8.4])]
    for a, b in zip(pts, pts[1:]):
        p.append(Part("cap", "wood", a=a, b=b, r1=0.8, r2=0.8))
    p.append(Part("cap", "bone", a=pts[0], b=V([7.0, 37.0, 5.2]), r1=0.18))
    p.append(Part("cap", "bone", a=pts[-1], b=V([7.0, 37.0, 5.2]), r1=0.18))
    p.append(Part("cap", "paleWood", a=V([6.5, 37.0, 5.0]), b=V([17.5, 37.4, 8.6]), r1=0.3))
    p.append(Part("cap", "iron", a=V([17.5, 37.4, 8.6]), b=V([19.4, 37.5, 9.2]), r1=0.7, r2=0.1))
    return p


def witch():
    p = []
    p.append(Part("cone", "heather", c=(0, 0, 0), y0=0.0, y1=47.0, r0=9.0, r1=6.0, sx=0.85,
                  folds=(11, 0.9, 0, 0, 0.0, 30)))
    p.append(Part("ellip", "leather", c=(4.6, 1.4, 2.6), r=(2.8, 1.6, 1.7), clip=[((0, 1, 0), 0.0)]))
    p.append(Part("ellip", "heather", blend=1.0, c=(0, 41, 0), r=(4.0, 7.0, 6.4)))
    # layered ash-moss shawls with ragged edges
    p.append(Part("ellip", "ashmoss", c=(0.2, 43.0, 0), r=(5.0, 6.4, 7.6), clip=[((0, 1, 0), 34.0)],
                  folds=(13, 0.7, 0, 0, 34, 42)))
    p.append(Part("ellip", "heather", c=(0.4, 47.0, 0), r=(4.8, 3.6, 7.0), clip=[((0, 1, 0), 43.0)],
                  folds=(9, 0.5, 0, 0, 43, 47)))
    p.append(Part("ellip", "leather", c=(0, 32.0, 0), r=(5.6, 0.7, 7.0)))
    for x, z, ln in ((4.6, 2.8, 3.0), (3.0, 5.2, 4.0), (5.0, -2.0, 2.4)):
        p.append(Part("cap", "hair", a=(x, 32, z), b=(x + 0.4, 32 - ln, z + 0.3), r1=0.2))
        p.append(Part("ellip", "bone", c=(x + 0.4, 31.2 - ln, z + 0.3), r=(0.8, 1.1, 0.8)))
    p.append(Part("ellip", "moss", c=(4.9, 29.0, 4.4), r=(0.8, 1.2, 0.8)))
    p += head(c=(0.9, 53.6, 0))
    p += hood("heather", c=(-0.9, 54.8, 0), r=(4.8, 5.4, 4.6))
    # crooked wand held low, a thread of grave-mist
    p.append(Part("cap", "heather", blend=0.6, a=(0, 46, 6.5), b=(3.6, 38.5, 8.4), r1=2.3, r2=2.8))
    p.append(Part("cap", "heather", blend=0.6, a=(3.6, 38.5, 8.4), b=(7.6, 35.0, 8.6), r1=2.6, r2=2.0))
    p.append(Part("ellip", "skin", c=(8.6, 34.6, 8.6), r=(1.4, 1.5, 1.4)))
    w = [V([9.4, 34.6, 8.6]), V([11.4, 33.2, 9.0]), V([12.4, 30.8, 9.2]), V([14.6, 29.6, 9.4])]
    for a, b in zip(w, w[1:]):
        p.append(Part("cap", "wood", a=a, b=b, r1=0.45, r2=0.4))
    for i, (dx, dy) in enumerate(((1.4, 0.6), (2.6, 1.8), (2.4, 3.4), (3.8, 4.8), (5.2, 4.6))):
        p.append(Part("ellip", "gravemist", c=(14.6 + dx, 29.6 + dy, 9.6), r=(0.7 + i * 0.12,) * 3))
    p.append(Part("cap", "heather", blend=0.6, a=(0, 46, -6.5), b=(2.0, 38, -8.2), r1=2.3, r2=2.8))
    return p


def ogre():
    p = []
    sk = "ogre"
    p += leg(sk, (1, 38, 7.5), (4.5, 22, 8.5), (4.5, 6, 8.6), r=(7.2, 6.0, 5.0), boot=None)
    p += leg(sk, (-1, 38, -7.5), (-3.5, 22, -8.5), (-4.5, 6, -8.6), r=(7.2, 6.0, 5.0), boot=None)
    for z, x in ((8.6, 4.5), (-8.6, -4.5)):
        p.append(Part("ellip", sk, blend=1.5, c=(x + 3.2, 3.0, z), r=(6.4, 3.4, 4.8), clip=[((0, 1, 0), 0.0)]))
    p.append(Part("ellip", sk, blend=3.0, c=(3.0, 46, 0), r=(11.5, 14, 13.5)))
    p.append(Part("ellip", sk, blend=3.0, c=(0, 64, 0), r=(10.5, 12, 15.5)))
    p.append(Part("ellip", sk, blend=3.0, c=(-4.5, 72, 0), r=(9, 8, 12)))
    p.append(Part("cone", "hide", c=(1.5, 0, 0), y0=27.0, y1=41.0, r0=13.4, r1=13.0, sx=0.95,
                  folds=(9, 1.0, 1.5, 0, 27, 34)))
    p.append(Part("ellip", "leather", c=(2.0, 41.0, 0), r=(13.0, 1.2, 13.4)))
    # small head thrust forward, heavy brow, underbite and tusks
    p.append(Part("ellip", sk, blend=2.0, c=(11.0, 70, 0), r=(6.2, 6.6, 5.6)))
    p.append(Part("ellip", sk, blend=1.2, c=(13.5, 65.2, 0), r=(5.0, 3.6, 5.2)))
    p.append(Part("ellip", sk, blend=0.8, c=(15.2, 72.4, 0), r=(2.4, 1.5, 4.6)))
    p.append(Part("ellip", sk, blend=0.8, c=(17.2, 69.6, 0), r=(1.6, 2.0, 1.5)))
    for z in (1.8, -1.8):
        p.append(Part("ellip", "ember_eye", c=(16.4, 71.0, z), r=(0.55, 0.55, 0.55)))
        p.append(Part("cap", "bone", a=(17.0, 64.8, z * 1.3), b=(17.6, 69.0, z * 1.4), r1=0.9, r2=0.25))
    for z in (5.4, -5.4):
        p.append(Part("ellip", sk, blend=0.6, c=(9.0, 71, z), r=(1.4, 2.2, 1.2)))
    # far arm hanging heavy
    p += arm(sk, (1, 68, -14), (4, 50, -17.5), (6, 36, -16.5), r=(5.6, 4.8, 4.2), skin=sk, hand_r=4.6)
    # near arm with the tree-trunk club over the shoulder
    p += arm(sk, (1, 68, 14), (9, 52, 18), (12, 62, 15.5), r=(5.6, 4.8, 4.2), skin=sk, hand_r=4.6)
    a, b = V([13.5, 58.5, 16.0]), V([-9.0, 92.0, 8.0])
    p.append(Part("cap", "wood", a=a, b=b, r1=2.4, r2=6.4))
    p.append(Part("cap", "wood", a=b + V([2.0, -6.0, 1.0]), b=b + V([7.0, -4.0, 4.0]), r1=1.8, r2=0.8))
    for t, dz in ((0.55, 2.5), (0.8, -3.0)):
        q = a + (b - a) * t
        p.append(Part("ellip", "wood", c=q + V([0, 0, dz]), r=(2.0, 2.0, 2.0)))
    return p


def goblin():
    p = []
    sk = "goblin"
    p += leg(sk, (0, 17, 2.4), (3.4, 10, 2.8), (0.8, 2.5, 3.0), r=(1.9, 1.6, 1.2), boot=None)
    p += leg(sk, (0, 17, -2.4), (2.0, 9.5, -3.0), (-2.6, 2.5, -3.0), r=(1.9, 1.6, 1.2), boot=None)
    for z, x in ((3.0, 0.8), (-3.0, -2.6)):
        p.append(Part("ellip", sk, blend=0.6, c=(x + 1.4, 1.2, z), r=(2.4, 1.3, 1.3), clip=[((0, 1, 0), 0.0)]))
    p.append(Part("ellip", sk, blend=1.0, c=(2.4, 24.5, 0), r=(3.0, 5.6, 4.2)))
    p.append(Part("cone", "hide", c=(1.0, 0, 0), y0=13.5, y1=27.0, r0=5.0, r1=4.4, sx=0.8,
                  folds=(7, 0.7, 1.0, 0, 13.5, 17)))
    p.append(Part("ellip", "leather", c=(1.2, 21.0, 0), r=(4.0, 0.7, 4.8)))
    # head jutting forward, long nose, ears swept back
    hc = V([6.0, 31.5, 0])
    p.append(Part("ellip", sk, blend=0.8, c=hc, r=(3.3, 3.3, 3.0)))
    p.append(Part("cap", sk, blend=0.5, a=hc + V([2.4, -0.4, 0]), b=hc + V([5.0, -1.6, 0]), r1=1.0, r2=0.5))
    p.append(Part("ellip", sk, blend=0.6, c=hc + V([1.6, -2.4, 0]), r=(2.2, 1.4, 2.4)))
    for z in (1.2, -1.2):
        p.append(Part("ellip", "yellow_eye", c=hc + V([2.7, 0.6, z]), r=(0.45,) * 3))
    for z in (2.8, -2.8):
        p.append(Part("cap", sk, a=hc + V([-0.6, 0.8, z]), b=hc + V([-5.0, 3.2, z * 1.9]), r1=1.1, r2=0.15,
                      sq=((0, 0, 1), 0.45)))
    p.append(Part("cap", "bone", a=hc + V([2.8, -2.8, 0.8]), b=hc + V([3.0, -1.6, 0.9]), r1=0.3, r2=0.1))
    # crude spear held low and forward
    a, b = V([-9.0, 18.0, 3.4]), V([17.0, 27.5, 4.6])
    p.append(Part("cap", "wood", a=a, b=b, r1=0.55))
    d = (b - a) / np.linalg.norm(b - a)
    p.append(Part("cap", "iron", a=b, b=b + d * 4.5, r1=1.2, r2=0.1, sq=((0, 0, 1), 0.4)))
    p.append(Part("cap", "leather", a=b - d * 1.2, b=b + d * 0.2, r1=0.8))
    p += arm(sk, (2.0, 29.0, 3.8), (4.8, 23.0, 5.2), (9.0, 23.6, 4.2), r=(1.2, 1.0, 0.9), skin=sk, hand_r=1.0)
    p += arm(sk, (2.0, 29.0, -3.8), (1.0, 22.5, -4.2), (2.0, 20.6, 3.6), r=(1.2, 1.0, 0.9), skin=sk, hand_r=1.0)
    return p


CLASSES = {"warrior": warrior, "rogue": rogue, "ranger": ranger,
           "cleric": cleric, "wizard": wizard, "witch": witch}
ORDER = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"]
