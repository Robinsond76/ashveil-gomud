"""Pre-rendered pixel sprites: signed-distance figures, lit and quantized.

Figures are built from rounded 3D primitives (ellipsoids, tapered capsules,
discs, boxes, tori). Each pixel is ray-marched orthographically, lit from
the top-left front, and snapped to its material's colour ramp. Edges are
darkened in the material's own colours rather than outlined in black.
"""
import colorsys
import numpy as np

# --- Colour ramps -------------------------------------------------------------
SHADOW_TINT = np.array([0x1b, 0x16, 0x2a]) / 255.0   # shadows lean cool violet
LIGHT_TINT = np.array([0xff, 0xf0, 0xd2]) / 255.0    # lights lean warm


def hexrgb(h):
    return np.array([int(h[i:i + 2], 16) for i in (0, 2, 4)]) / 255.0


def make_ramp(base, steps=8, dark=0.78, light=0.55):
    """Build a hue-shifted ramp around a base colour, dark to light."""
    b = hexrgb(base)
    out = []
    for i in range(steps):
        t = i / (steps - 1) * 2 - 1  # -1 .. 1, base sits a little above the middle
        if t < 0.15:
            k = (0.15 - t) / 1.15 * dark
            c = b * (1 - k) + SHADOW_TINT * k
        else:
            k = (t - 0.15) / 0.85 * light
            c = b * (1 - k) + LIGHT_TINT * k
        out.append("%02x%02x%02x" % tuple(int(round(v * 255)) for v in np.clip(c, 0, 1)))
    return out


MATERIALS = {
    # name: (base colour, steps, dark, light, pattern)
    "skin": ("b07a5c", 8, 0.75, 0.55, None),
    "skin_pale": ("bf9478", 8, 0.75, 0.55, None),
    "hair": ("4a3324", 7, 0.7, 0.45, None),
    "beard_grey": ("8d877c", 8, 0.72, 0.6, None),
    "iron": ("5e646b", 8, 0.75, 0.7, None),
    "steel": ("7d858c", 8, 0.7, 0.85, None),
    "mail": ("5b6168", 8, 0.75, 0.7, "mail"),
    "oxblood": ("6e2219", 8, 0.75, 0.45, "weave"),
    "leather": ("6a4228", 8, 0.75, 0.5, None),
    "tan": ("8d6440", 8, 0.72, 0.55, None),
    "wood": ("5f3f25", 8, 0.75, 0.5, "grain"),
    "wool": ("b7ab95", 8, 0.72, 0.55, "weave"),
    "brass": ("9a7a35", 8, 0.75, 0.8, None),
    "slate": ("3f566b", 8, 0.75, 0.5, "weave"),
    "charcoal": ("37302d", 8, 0.65, 0.45, None),
    "plum": ("4e2f4f", 8, 0.7, 0.45, None),
    "moss": ("3d5530", 8, 0.75, 0.45, "weave"),
    "heather": ("54405f", 8, 0.72, 0.5, "weave"),
    "ashmoss": ("4f5642", 8, 0.72, 0.45, "weave"),
    "bone": ("bdb197", 8, 0.7, 0.55, None),
    "ogre": ("5f5c45", 9, 0.8, 0.42, "mottle"),
    "goblin": ("5d6a46", 8, 0.75, 0.5, "mottle"),
    "hide": ("6b4a2e", 8, 0.75, 0.45, "fur"),
    "eye": ("1a1414", 2, 0.0, 0.0, None),
    "ember_eye": ("c2552a", 3, 0.3, 0.5, None),
    "glow": ("9fc0d4", 6, 0.5, 0.9, None),
    "canvas": ("a08f72", 8, 0.75, 0.5, None),
    "stone_m": ("6d6a62", 8, 0.75, 0.5, None),
    "heal_glow": ("8fb46a", 5, 0.4, 0.7, None),
    "plaster": ("c2b59a", 8, 0.75, 0.45, None),
    "gravemist": ("8a9c86", 4, 0.3, 0.4, None),
    "yellow_eye": ("c9a43c", 3, 0.3, 0.5, None),
    "paleWood": ("9c7a52", 7, 0.7, 0.5, "grain"),
    "thatch_m": ("9a7a3e", 8, 0.75, 0.5, "grain"),
    "water_m": ("4d7b96", 8, 0.75, 0.75, None),
    "blood": ("7e1d16", 8, 0.75, 0.55, None),
    "berry": ("8e2a2a", 8, 0.75, 0.6, None),
    "leaf": ("4e7034", 8, 0.75, 0.55, None),
    "cloth_white": ("b8b2a2", 8, 0.7, 0.6, None),
}
RAMPS = {k: make_ramp(v[0], v[1], v[2], v[3]) for k, v in MATERIALS.items()}
EMISSIVE = {"glow", "ember_eye", "gravemist", "yellow_eye", "heal_glow"}


def rgb(h):
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


BAYER = np.array([[0, 8, 2, 10], [12, 4, 14, 6], [3, 11, 1, 9], [15, 7, 13, 5]]) / 16.0


# --- Geometry helpers -----------------------------------------------------------
def norm(v):
    v = np.asarray(v, float)
    return v / np.linalg.norm(v)


def frame_y(up):
    """Rotation matrix whose local y axis points along `up`."""
    y = norm(up)
    a = np.array([1.0, 0, 0]) if abs(y[0]) < 0.9 else np.array([0, 0, 1.0])
    x = norm(np.cross(a, y))
    z = np.cross(x, y)
    return np.stack([x, y, z], 1)


def smin(a, b, k):
    h = np.clip(0.5 + 0.5 * (b - a) / k, 0, 1)
    return b * (1 - h) + a * h - k * h * (1 - h)


class Part:
    def __init__(self, kind, mat, blend=0.0, clip=(), folds=None, **kw):
        self.kind, self.mat, self.blend, self.clip, self.folds, self.kw = kind, mat, blend, clip, folds, kw

    def local(self, P):
        c = np.asarray(self.kw.get("c", (0, 0, 0)), float)
        R = self.kw.get("R")
        Q = P - c
        if R is not None:
            Q = Q @ R
        return Q

    def sdf(self, P):
        k, kw = self.kind, self.kw
        if k == "ellip":
            Q = self.local(P)
            r = np.asarray(kw["r"], float)
            k0 = np.linalg.norm(Q / r, axis=1)
            k1 = np.linalg.norm(Q / (r * r), axis=1)
            d = k0 * (k0 - 1) / np.maximum(k1, 1e-6)
        elif k == "cap":
            a, b = np.asarray(kw["a"], float), np.asarray(kw["b"], float)
            r1, r2 = kw["r1"], kw.get("r2", kw["r1"])
            ba = b - a
            pa = P - a
            h = np.clip((pa @ ba) / (ba @ ba), 0, 1)
            sq = kw.get("sq")  # optional squash of the cross-section along an axis
            diff = pa - np.outer(h, ba)
            if sq is not None:
                ax, f = np.asarray(sq[0], float), sq[1]
                comp = diff @ ax
                diff = diff + np.outer(comp * (1 / f - 1), ax)
            d = np.linalg.norm(diff, axis=1) - (r1 + (r2 - r1) * h)
        elif k == "box":
            Q = self.local(P)
            q = np.abs(Q) - np.asarray(kw["b"], float)
            d = np.linalg.norm(np.maximum(q, 0), axis=1) + np.minimum(q.max(1), 0) - kw.get("rr", 0)
        elif k == "disc":
            Q = self.local(P)
            dx = np.hypot(Q[:, 0], Q[:, 2]) - kw["rad"]
            dy = np.abs(Q[:, 1]) - kw["h"]
            rr = kw.get("rr", 0)
            d = np.hypot(np.maximum(dx, 0), np.maximum(dy, 0)) + np.minimum(np.maximum(dx, dy), 0) - rr
        elif k == "torus":
            Q = self.local(P)
            qx = np.hypot(Q[:, 0], Q[:, 2]) - kw["rad"]
            d = np.hypot(qx, Q[:, 1]) - kw["r"]
        elif k == "cone":
            # vertical frustum: centre c, from y0 (radius r0) up to y1 (radius r1), squashed in x
            c = np.asarray(kw["c"], float)
            Q = P - c
            y0, y1, r0, r1 = kw["y0"], kw["y1"], kw["r0"], kw["r1"]
            sx = kw.get("sx", 1.0)
            t = np.clip((Q[:, 1] - y0) / (y1 - y0), 0, 1)
            rad = r0 + (r1 - r0) * t
            rr = np.hypot(Q[:, 0] / sx, Q[:, 2])
            d_side = (rr - rad) * min(sx, 1.0) * 0.9
            d_cap = np.maximum(y0 - Q[:, 1], Q[:, 1] - y1)
            d = np.maximum(d_side, d_cap)
        else:
            raise ValueError(k)
        if self.folds is not None:
            n, amp, cx, cz, ya, yb = self.folds
            ang = np.arctan2(P[:, 2] - cz, P[:, 0] - cx)
            w = np.clip((yb - P[:, 1]) / max(yb - ya, 1e-3), 0, 1)
            d = d + amp * w * np.sin(n * ang + 0.7)
        for nrm, off in self.clip:
            d = np.maximum(d, off - P @ np.asarray(nrm, float))
        if "minus" in self.kw:
            d = np.maximum(d, -self.kw["minus"].sdf(P))
        return d


def pattern(name, P, n):
    """Ramp offset from a surface pattern at model point P."""
    x, y, z = P[:, 0], P[:, 1], P[:, 2]
    if name == "mail":
        return np.where(((np.floor(y * 1.0) + np.floor((x + z) * 1.0)) % 2) == 0, 0.45, -0.45)
    if name == "grain":
        return 0.5 * np.sin((x * 0.4 + z * 0.4 + y * 2.3)) * np.sin(y * 0.7 + x)
    if name == "weave":
        return 0.25 * np.sin(y * 1.7 + x * 0.3) * np.sin(z * 1.3)
    if name == "mottle":
        return 0.45 * np.sin(x * 0.9 + y * 0.53) * np.sin(z * 0.8 - y * 0.41)
    if name == "fur":
        return 0.55 * np.sin(x * 2.1 + y * 3.1 + z * 1.7)
    return np.zeros(len(P))


# --- Renderer -------------------------------------------------------------------
class Camera:
    def __init__(self, yaw=30, pitch=12, scale=1.0, bulk=1.0):
        self.bulk = bulk
        a, b = np.radians(yaw), np.radians(pitch)
        Ry = np.array([[np.cos(a), 0, -np.sin(a)], [0, 1, 0], [np.sin(a), 0, np.cos(a)]])
        Rx = np.array([[1, 0, 0], [0, np.cos(b), -np.sin(b)], [0, np.sin(b), np.cos(b)]])
        self.M = Rx @ Ry          # model -> camera
        self.Mi = self.M.T        # camera -> model
        self.scale = scale


def render(parts, w, h, cam, anchor, light=(-0.55, 0.7, 0.55), ambient=0.22, dither=0.35,
           edge=True, ground_y=0.0):
    """Render parts into an RGBA array. anchor = (px, py) where model origin lands."""
    ys, xs = np.mgrid[0:h, 0:w]
    sx = (xs.ravel() + 0.5 - anchor[0]) / cam.scale
    sy = (anchor[1] - (ys.ravel() + 0.5)) / cam.scale
    N = sx.size
    zfar = 120.0
    O = np.stack([sx, sy, np.full(N, zfar)], 1)
    D = np.array([0, 0, -1.0])

    def scene(Pc, want_mat=False):
        Pm = Pc @ cam.Mi.T
        if cam.bulk != 1.0:
            Pm = Pm.copy()
            Pm[:, 0] /= cam.bulk
            Pm[:, 2] /= cam.bulk
        ds = np.stack([p.sdf(Pm) for p in parts])
        d = ds[0].copy()
        for i in range(1, len(parts)):
            k = parts[i].blend
            d = smin(d, ds[i], k) if k > 0 else np.minimum(d, ds[i])
        if want_mat:
            return d, np.argmin(ds, 0), Pm
        return d

    t = np.zeros(N)
    alive = np.ones(N, bool)
    hit = np.zeros(N, bool)
    for _ in range(160):
        idx = np.where(alive)[0]
        if idx.size == 0:
            break
        P = O[idx] + np.outer(t[idx], D)
        d = scene(P)
        done = d < 0.02
        hit[idx[done]] = True
        alive[idx[done]] = False
        t[idx[~done]] += np.maximum(d[~done] * 0.7, 0.02)
        far = t[idx] > 2 * zfar
        alive[idx[far]] = False
    out = np.zeros((h, w, 4), np.uint8)
    depth = np.full(N, np.inf)
    level = np.full(N, -1.0)
    mats = np.full(N, -1)
    hi = np.where(hit)[0]
    if hi.size:
        P = O[hi] + np.outer(t[hi], D)
        depth[hi] = t[hi]
        e = 0.25
        nrm = np.stack([scene(P + np.array(v) * e) - scene(P - np.array(v) * e)
                        for v in ((1, 0, 0), (0, 1, 0), (0, 0, 1))], 1)
        nrm /= np.maximum(np.linalg.norm(nrm, axis=1, keepdims=True), 1e-6)
        _, mid, Pm = scene(P, True)
        L = norm(light)
        lam = np.clip(nrm @ L, 0, 1)
        # cheap ambient occlusion
        ao = np.ones(len(P))
        for i, s in enumerate((0.6, 1.4, 2.6, 4.0)):
            ao -= np.clip(s - scene(P + nrm * s), 0, None) / s * (0.5 ** i) * 0.35
        ao = np.clip(ao, 0.35, 1)
        rim = np.clip(1 - nrm[:, 2], 0, 1) ** 3 * np.clip(nrm[:, 0], 0, 1) * 0.25
        inten = (ambient + 0.78 * lam) * ao + rim
        for i, p in enumerate(parts):
            sel = mid == i
            if not sel.any():
                continue
            ramp = RAMPS[p.mat]
            n_ = len(ramp)
            lv = inten[sel] * (n_ - 1) * 0.92
            pat = MATERIALS[p.mat][4]
            if pat:
                lv = lv + pattern(pat, Pm[sel], nrm[sel])
            if p.mat in EMISSIVE:
                lv = (n_ - 1) * (0.55 + 0.45 * inten[sel])
            level[hi[sel]] = lv
            mats[hi[sel]] = i
    level = level.reshape(h, w)
    mats = mats.reshape(h, w)
    depth = depth.reshape(h, w)
    # colour-tinted edges: silhouette and depth steps darken in-material
    if edge:
        dark = np.zeros((h, w))
        for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
            nb = np.full((h, w), np.inf)
            ys_ = slice(max(dy, 0), h + min(dy, 0))
            yd = slice(max(-dy, 0), h + min(-dy, 0))
            xs_ = slice(max(dx, 0), w + min(dx, 0))
            xd = slice(max(-dx, 0), w + min(-dx, 0))
            nb[yd, xd] = depth[ys_, xs_]
            sil = (mats >= 0) & np.isinf(nb)
            with np.errstate(invalid='ignore'):
                step = (mats >= 0) & np.isfinite(nb) & (depth - nb > 2.2)
            dark = np.maximum(dark, np.where(sil, 1.6 if (dy, dx) in ((1, 0), (0, 1)) else 1.1, 0))
            dark = np.maximum(dark, np.where(step, 1.4, 0))
        level = level - dark
    for y in range(h):
        for x in range(w):
            m = mats[y, x]
            if m < 0:
                continue
            ramp = RAMPS[parts[m].mat]
            lv = level[y, x]
            base = np.floor(lv)
            frac = lv - base
            idx = int(base + (1 if frac > 0.5 + (BAYER[y % 4, x % 4] - 0.5) * dither else 0))
            idx = max(0, min(len(ramp) - 1, idx))
            out[y, x, :3] = rgb(ramp[idx])
            out[y, x, 3] = 255
    return out


def palette_colours():
    seen = []
    for k in MATERIALS:
        for c in RAMPS[k]:
            if c not in seen:
                seen.append(c)
    return seen
