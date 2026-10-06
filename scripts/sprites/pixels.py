"""Pixel canvas helpers: palette-name grids, shaded parts, outlines, sheets."""
from PIL import Image

from palette import PAL


def rect(x0, y0, x1, y1):
    """Inclusive rectangle as a set of pixels."""
    return {(x, y) for x in range(x0, x1 + 1) for y in range(y0, y1 + 1)}


def round_rect(x0, y0, x1, y1):
    """Rectangle with its four corner pixels removed."""
    s = rect(x0, y0, x1, y1)
    return s - {(x0, y0), (x1, y0), (x0, y1), (x1, y1)}


def line(x0, y0, x1, y1):
    """Bresenham line as a set of pixels."""
    pts = set()
    dx, dy = abs(x1 - x0), -abs(y1 - y0)
    sx, sy = (1 if x0 < x1 else -1), (1 if y0 < y1 else -1)
    err = dx + dy
    while True:
        pts.add((x0, y0))
        if (x0, y0) == (x1, y1):
            break
        e2 = 2 * err
        if e2 >= dy:
            err += dy
            x0 += sx
        if e2 <= dx:
            err += dx
            y0 += sy
    return pts


def thick(points, w=2):
    """Widen a pixel set by w-1 pixels to the right."""
    out = set(points)
    for i in range(1, w):
        out |= {(x + i, y) for x, y in points}
    return out


def ellipse(cx, cy, rx, ry):
    return {
        (x, y)
        for x in range(int(cx - rx) - 1, int(cx + rx) + 2)
        for y in range(int(cy - ry) - 1, int(cy + ry) + 2)
        if ((x - cx) / (rx + 0.3)) ** 2 + ((y - cy) / (ry + 0.3)) ** 2 <= 1
    }


class Canvas:
    """A grid of palette-color names (None is transparent)."""

    def __init__(self, w, h, bg=None):
        self.w, self.h = w, h
        self.g = [[bg] * w for _ in range(h)]

    def put(self, x, y, c):
        if 0 <= x < self.w and 0 <= y < self.h:
            self.g[y][x] = c

    def get(self, x, y):
        if 0 <= x < self.w and 0 <= y < self.h:
            return self.g[y][x]
        return None

    def fill(self, pixels, c):
        for x, y in pixels:
            self.put(x, y, c)

    def part(self, pixels, ramp, flat=None):
        """Paint a pixel set in a material ramp with top-left light.

        Left and top edges take the light step, right and bottom edges the
        dark step, the interior the mid step.  flat="m" (or d/l) disables
        shading.
        """
        if "." in ramp:  # a fixed palette color, not a ramp
            self.fill(pixels, ramp)
            return
        for x, y in pixels:
            if flat:
                tone = flat
            else:
                dark = (x + 1, y) not in pixels or (x, y + 1) not in pixels
                light = (x - 1, y) not in pixels or (x, y - 1) not in pixels
                tone = "d" if dark else ("l" if light else "m")
                if dark and light and (x + 1, y) in pixels:
                    tone = "m"
            self.put(x, y, f"{ramp}.{tone}")

    def outline(self, color="outline"):
        """Add a 1 px outline around every opaque shape."""
        add = []
        for y in range(self.h):
            for x in range(self.w):
                if self.g[y][x] is None:
                    for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
                        if self.get(nx, ny) not in (None, color):
                            add.append((x, y))
                            break
        for x, y in add:
            self.g[y][x] = color

    def blit(self, other, ox=0, oy=0):
        for y in range(other.h):
            for x in range(other.w):
                c = other.g[y][x]
                if c is not None:
                    self.put(x + ox, y + oy, c)

    def mirror(self):
        m = Canvas(self.w, self.h)
        for y in range(self.h):
            m.g[y] = list(reversed(self.g[y]))
        return m

    def to_image(self):
        img = Image.new("RGBA", (self.w, self.h), (0, 0, 0, 0))
        px = img.load()
        for y in range(self.h):
            for x in range(self.w):
                c = self.g[y][x]
                if c is not None:
                    px[x, y] = PAL[c] + (255,)
        return img


def ascii_art(rows, legend, w=None, h=None):
    """Build a canvas from rows of characters; '.' and ' ' are transparent."""
    w = w or max(len(r) for r in rows)
    h = h or len(rows)
    cv = Canvas(w, h)
    for y, row in enumerate(rows):
        for x, ch in enumerate(row):
            if ch not in ". ":
                cv.put(x, y, legend[ch])
    return cv


def sheet(grid):
    """Join canvases (list of rows, each a list of frames) with no padding."""
    fw, fh = grid[0][0].w, grid[0][0].h
    out = Canvas(fw * len(grid[0]), fh * len(grid))
    for r, row in enumerate(grid):
        for c, frame in enumerate(row):
            assert (frame.w, frame.h) == (fw, fh)
            out.blit(frame, c * fw, r * fh)
    return out
