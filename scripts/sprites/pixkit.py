"""Small pixel-art toolkit for Ashveil sprites.

Shapes are drawn as material regions. render() shades each region from the
top-left light, separates overlapping parts, and adds a 1 px outline in the
darkest palette color. Every pixel comes from the master palette.
"""
import numpy as np
from PIL import Image, ImageDraw

# Master palette: (id, hex, name). Ramps run dark to light.
PALETTE = [
    # Warm stone and ink
    ("W0", "14100e", "Ink"), ("W1", "241d1a", "Soot"), ("W2", "3a302b", "Peat"),
    ("W3", "554840", "Old Stone"), ("W4", "74665c", "Weathered Stone"),
    ("W5", "978a7d", "Lichen Grey"), ("W6", "beb3a3", "Dry Bone"),
    # Cool grey iron
    ("C0", "1a1d22", "Pitch"), ("C1", "2a2f36", "Gunmetal"), ("C2", "3d434b", "Dark Iron"),
    ("C3", "545b63", "Iron"), ("C4", "6f777e", "Worn Steel"), ("C5", "939ba0", "Steel"),
    ("C6", "c3c9cb", "Edge Light"),
    # Earth and leather
    ("E0", "2a1a12", "Loam"), ("E1", "3f2717", "Bark"), ("E2", "58371f", "Old Leather"),
    ("E3", "74492a", "Leather"), ("E4", "8f5f38", "Saddle"), ("E5", "ab7a4c", "Tan"),
    ("E6", "c69a68", "Straw"), ("E7", "dfc092", "Parchment"),
    # Forest greens
    ("G0", "101a12", "Deep Wood"), ("G1", "1a2a1a", "Pine Shadow"), ("G2", "263c23", "Pine"),
    ("G3", "344f2c", "Moss"), ("G4", "466434", "Fern"), ("G5", "5d7a3e", "Meadow"),
    ("G6", "7a944f", "Sunlit Grass"), ("G7", "9fb06c", "Pale Grass"),
    # River and slate blues
    ("B0", "121a26", "Night Water"), ("B1", "1d2b3b", "Deep River"), ("B2", "2b4054", "River"),
    ("B3", "3c566b", "Slate"), ("B4", "547086", "Shallows"), ("B5", "728d9f", "Mist"),
    ("B6", "9db4c2", "Pale Light"),
    # Blood, oxblood and ember
    ("R0", "260e0c", "Dried Blood"), ("R1", "3d1410", "Oxblood Shadow"), ("R2", "5a1d15", "Oxblood"),
    ("R3", "7a291b", "Rust"), ("R4", "9c3a22", "Brick"), ("R5", "c0582c", "Ember"),
    ("R6", "de8a44", "Flame"),
    # Ochre, brass and firelight
    ("O0", "2e2410", "Umber"), ("O1", "4a3a19", "Tarnish"), ("O2", "6b5424", "Old Brass"),
    ("O3", "8c6f2f", "Ochre"), ("O4", "ad8c3d", "Brass"), ("O5", "ccae5a", "Wheat"),
    ("O6", "ead68e", "Candlelight"),
    # Heather and plum
    ("V0", "1f1625", "Bruise"), ("V1", "30223a", "Dusk"), ("V2", "44314f", "Plum"),
    ("V3", "5a4466", "Heather"), ("V4", "765c80", "Thistle"), ("V5", "9a83a1", "Pale Heather"),
    # Skin, wool and bone
    ("S0", "6b4232", "Weathered Skin Shadow"), ("S1", "8c5a43", "Tanned Skin"),
    ("S2", "ad7657", "Skin"), ("S3", "c99677", "Fair Skin"), ("S4", "e0b898", "Skin Light"),
    ("S5", "e9dcc3", "Undyed Wool"), ("S6", "f4efe2", "Highlight"),
]
assert len(PALETTE) == 64
HEX = {k: v for k, v, _ in PALETTE}
RGB = {k: tuple(int(v[i:i + 2], 16) for i in (0, 2, 4)) for k, v in HEX.items()}
OUTLINE = "W0"

# Material ramps: deep, shadow, base, light, highlight.
RAMPS = {
    "iron": ["C1", "C2", "C3", "C4", "C5"],
    "steel": ["C2", "C3", "C4", "C5", "C6"],
    "mail": ["C0", "C2", "C3", "C4", "C5"],
    "oxblood": ["R0", "R1", "R2", "R3", "R4"],
    "leather": ["E0", "E1", "E2", "E3", "E4"],
    "tan": ["E1", "E2", "E3", "E4", "E5"],
    "wood": ["E0", "E1", "E2", "E3", "E4"],
    "paleWood": ["E1", "E3", "E4", "E5", "E6"],
    "wool": ["W3", "W4", "W5", "W6", "S5"],
    "brass": ["O0", "O1", "O2", "O4", "O5"],
    "skin": ["S0", "S0", "S1", "S2", "S3"],
    "slate": ["B0", "B1", "B2", "B3", "B4"],
    "pewter": ["C1", "C2", "C3", "C5", "C6"],
    "beard": ["W2", "W3", "W4", "W5", "W6"],
    "moss": ["G0", "G1", "G2", "G3", "G4"],
    "charcoal": ["W0", "W1", "W2", "W3", "W4"],
    "plum": ["V0", "V0", "V1", "V2", "V3"],
    "heather": ["V0", "V1", "V2", "V3", "V4"],
    "ashmoss": ["G0", "G1", "W3", "G3", "W4"],
    "bone": ["W3", "W4", "W5", "W6", "S5"],
    "hair": ["W0", "W1", "E1", "E2", "E3"],
    "goblin": ["G0", "G1", "G3", "G4", "W5"],
    "ogre": ["E0", "E1", "W3", "W4", "W5"],
    "hide": ["E0", "E1", "E2", "E3", "E4"],
    "ember": ["R4", "R5", "R6", "O6", "S6"],
    "glow": ["B4", "B5", "B6", "S6", "S6"],
    "mist": ["G3", "G4", "W5", "B5", "B6"],
    "wolfgrey": ["W1", "W2", "W3", "W4", "W5"],
}


class Canvas:
    """A material-region canvas rendered with directional shading."""

    def __init__(self, w, h, band=1):
        self.w, self.h, self.band = w, h, band
        self.mat = np.full((h, w), -1, int)
        self.mode = np.zeros((h, w), int)  # -1 auto, else forced tone 0..4
        self.z = np.zeros((h, w), int)
        self.mats = []
        self.zc = 0
        self._lit = {}

    # Materials -----------------------------------------------------------
    def m(self, ramp, tex=None, sep=True, outline=True):
        ramp = RAMPS[ramp] if isinstance(ramp, str) else ramp
        self.mats.append(dict(ramp=ramp, tex=tex, sep=sep, outline=outline))
        return len(self.mats) - 1

    def lit(self, color, outline=True):
        key = (color, outline)
        if key not in self._lit:
            self._lit[key] = self.m([color] * 5, sep=False, outline=outline)
        return self._lit[key]

    # Shapes ----------------------------------------------------------------
    def _apply(self, mask, mid, tone=-1):
        self.zc += 1
        self.mat[mask] = mid
        self.mode[mask] = tone
        self.z[mask] = self.zc

    def _mask(self, fn):
        img = Image.new("1", (self.w, self.h), 0)
        fn(ImageDraw.Draw(img))
        return np.array(img, dtype=bool)

    def poly(self, pts, mid, tone=-1):
        self._apply(self._mask(lambda d: d.polygon([tuple(p) for p in pts], fill=1, outline=1)), mid, tone)

    def rect(self, x0, y0, x1, y1, mid, tone=-1):
        self._apply(self._mask(lambda d: d.rectangle([x0, y0, x1, y1], fill=1)), mid, tone)

    def ellipse(self, x0, y0, x1, y1, mid, tone=-1):
        self._apply(self._mask(lambda d: d.ellipse([x0, y0, x1, y1], fill=1)), mid, tone)

    def line(self, pts, mid, width=1, tone=-1):
        def f(d):
            d.line([tuple(p) for p in pts], fill=1, width=width)
            if width > 2:
                r = width / 2 - 0.5
                for x, y in pts:
                    d.ellipse([x - r, y - r, x + r, y + r], fill=1)
        self._apply(self._mask(f), mid, tone)

    def px(self, x, y, mid, tone=-1):
        if 0 <= x < self.w and 0 <= y < self.h:
            self.zc += 1
            self.mat[y, x] = mid
            self.mode[y, x] = tone
            self.z[y, x] = self.zc

    def dot(self, x, y, color, outline=True):
        self.px(x, y, self.lit(color, outline))

    def erase(self, x, y):
        if 0 <= x < self.w and 0 <= y < self.h:
            self.mat[y, x] = -1

    def ascii(self, rows, legend, ox=0, oy=0):
        """Draw a character grid. legend maps char -> (mid, tone)."""
        for y, row in enumerate(rows):
            for x, ch in enumerate(row):
                if ch in legend:
                    mid, tone = legend[ch]
                    self.px(ox + x, oy + y, mid, tone)

    def blit(self, other, ox, oy, flip=False):
        """Copy another canvas's regions in, remapping its materials."""
        base = len(self.mats)
        self.mats.extend(other.mats)
        mat = other.mat[:, ::-1] if flip else other.mat
        mode = other.mode[:, ::-1] if flip else other.mode
        z = other.z[:, ::-1] if flip else other.z
        zoff = self.zc
        for y in range(other.h):
            for x in range(other.w):
                if mat[y, x] < 0:
                    continue
                X, Y = ox + x, oy + y
                if 0 <= X < self.w and 0 <= Y < self.h:
                    self.mat[Y, X] = mat[y, x] + base
                    self.mode[Y, X] = mode[y, x]
                    self.z[Y, X] = z[y, x] + zoff
        self.zc += int(other.z.max()) + 1

    # Rendering -------------------------------------------------------------
    def tones(self, flip_light=False):
        mat, h, w, k = self.mat, self.h, self.w, self.band
        same = lambda a, b: a == b
        dL = np.zeros((h, w), int); dR = np.zeros((h, w), int)
        dU = np.zeros((h, w), int); dD = np.zeros((h, w), int)
        for y in range(h):
            for x in range(1, w):
                if mat[y, x] >= 0 and mat[y, x - 1] == mat[y, x]:
                    dL[y, x] = dL[y, x - 1] + 1
            for x in range(w - 2, -1, -1):
                if mat[y, x] >= 0 and mat[y, x + 1] == mat[y, x]:
                    dR[y, x] = dR[y, x + 1] + 1
        for x in range(w):
            for y in range(1, h):
                if mat[y, x] >= 0 and mat[y - 1, x] == mat[y, x]:
                    dU[y, x] = dU[y - 1, x] + 1
            for y in range(h - 2, -1, -1):
                if mat[y, x] >= 0 and mat[y + 1, x] == mat[y, x]:
                    dD[y, x] = dD[y + 1, x] + 1
        if flip_light:
            dL, dR = dR, dL
        t = np.full((h, w), 2, int)
        a = np.minimum(dL, dU)
        b = np.minimum(dR, dD)
        t[(b < k)] = 1
        t[(a == 0) & (b > 0)] = 3
        t[(dU == 0) & (dL == 0) & (b > 0) & (dR > 1)] = 4
        t[(a == 0) & (b == 0)] = 2
        t[(dR == 0) & (dD == 0) & (a > 1)] = 0
        # textures
        for y in range(h):
            for x in range(w):
                mid = mat[y, x]
                if mid < 0:
                    continue
                tex = self.mats[mid]["tex"]
                if tex == "mail" and t[y, x] == 2:
                    if y % 2 == 0 and x % 2 == 0:
                        t[y, x] = 3
                    elif y % 2 == 1 and x % 2 == 1:
                        t[y, x] = 1
                elif tex == "wood" and t[y, x] == 2 and (x * 3 + y // 3) % 5 == 0:
                    t[y, x] = 1
                elif tex == "fur" and t[y, x] in (2, 3) and (x * 2 + y) % 4 == 0:
                    t[y, x] -= 1
        # separations between overlapping parts
        for y in range(h):
            for x in range(w):
                mid = mat[y, x]
                if mid < 0 or not self.mats[mid]["sep"]:
                    continue
                for dx, dy, tone in ((1, 0, 0), (0, 1, 0), (-1, 0, 1), (0, -1, 1)):
                    if flip_light and dx:
                        tone = 1 - tone
                    X, Y = x + dx, y + dy
                    if 0 <= X < w and 0 <= Y < h:
                        q = mat[Y, X]
                        if q >= 0 and q != mid and self.z[Y, X] < self.z[y, x] and self.mats[q]["sep"]:
                            t[Y, X] = min(t[Y, X], tone)
        forced = self.mode >= 0
        t[forced] = self.mode[forced]
        return t

    def render(self, outline=True, flip_light=False):
        t = self.tones(flip_light)
        out = np.zeros((self.h, self.w, 4), np.uint8)
        for y in range(self.h):
            for x in range(self.w):
                mid = self.mat[y, x]
                if mid < 0:
                    continue
                out[y, x, :3] = RGB[self.mats[mid]["ramp"][t[y, x]]]
                out[y, x, 3] = 255
        if outline:
            ol = np.zeros((self.h, self.w), bool)
            src = np.zeros((self.h, self.w), bool)
            for y in range(self.h):
                for x in range(self.w):
                    mid = self.mat[y, x]
                    src[y, x] = mid >= 0 and self.mats[mid]["outline"]
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                sh = np.zeros_like(src)
                ys = slice(max(dy, 0), self.h + min(dy, 0))
                yd = slice(max(-dy, 0), self.h + min(-dy, 0))
                xs = slice(max(dx, 0), self.w + min(dx, 0))
                xd = slice(max(-dx, 0), self.w + min(-dx, 0))
                sh[yd, xd] = src[ys, xs]
                ol |= sh
            ol &= out[:, :, 3] == 0
            out[ol, :3] = RGB[OUTLINE]
            out[ol, 3] = 255
        return out


def to_image(arr):
    return Image.fromarray(arr, "RGBA")


def paste(dst, src, ox, oy):
    """Alpha-paste (hard edges) src array onto dst array."""
    h, w = src.shape[:2]
    for y in range(h):
        for x in range(w):
            if src[y, x, 3] and 0 <= oy + y < dst.shape[0] and 0 <= ox + x < dst.shape[1]:
                dst[oy + y, ox + x] = src[y, x]


def flip(arr):
    return arr[:, ::-1].copy()


def save(arr, path):
    img = to_image(arr)
    img.save(path, optimize=True)  # Pillow writes no metadata chunks by default


def check_palette(arr):
    allowed = set(RGB.values())
    bad = set()
    for p in arr.reshape(-1, 4):
        if p[3] and tuple(int(c) for c in p[:3]) not in allowed:
            bad.add(tuple(int(c) for c in p[:3]))
    return bad
