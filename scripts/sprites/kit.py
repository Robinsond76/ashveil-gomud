"""Shared drawing helpers for the battle creature, background and terrain code."""
import math

from pixels import Canvas, rect, line, ellipse


def stroke(points, w=2):
    """Widen a pixel set by w pixels in both directions (round-ish limbs)."""
    out = set()
    for x, y in points:
        for dx in range(w):
            for dy in range(w):
                out.add((x + dx, y + dy))
    return out


def limb(a, b, w=2):
    return stroke(line(a[0], a[1], b[0], b[1]), w)


def poly(points):
    """Filled convex-ish polygon (scanline fill) as a pixel set."""
    ys = [p[1] for p in points]
    out = set()
    n = len(points)
    for y in range(min(ys), max(ys) + 1):
        xs = []
        for i in range(n):
            (x0, y0), (x1, y1) = points[i], points[(i + 1) % n]
            if y0 == y1:
                continue
            if min(y0, y1) <= y < max(y0, y1) or (y == max(y0, y1) == y):
                t = (y - y0) / (y1 - y0)
                if 0 <= t <= 1:
                    xs.append(x0 + t * (x1 - x0))
        xs.sort()
        for i in range(0, len(xs) - 1, 2):
            for x in range(int(round(xs[i])), int(round(xs[i + 1])) + 1):
                out.add((x, y))
    return out


def recolor(cv, mapping):
    """New canvas with whole ramps swapped: {"ashmoss": "stone"} maps every tone."""
    out = Canvas(cv.w, cv.h)
    for y in range(cv.h):
        for x in range(cv.w):
            c = cv.g[y][x]
            if c is None or c == "outline":
                out.g[y][x] = c
                continue
            ramp, _, tone = c.partition(".")
            if ramp in mapping:
                tgt = mapping[ramp]
                out.g[y][x] = tgt if "." in tgt else f"{tgt}.{tone}"
            else:
                out.g[y][x] = c
    return out


def shade_pixels(cv, pixels, steps=1):
    """Push painted pixels darker by steps (far limbs, undersides)."""
    order = {"l": "m", "m": "d", "d": "d"}
    for x, y in pixels:
        c = cv.get(x, y)
        if c and "." in c:
            ramp, tone = c.split(".")
            for _ in range(steps):
                tone = order[tone]
            cv.put(x, y, f"{ramp}.{tone}")


def shift(cv, dx=0, dy=0):
    out = Canvas(cv.w, cv.h)
    out.blit(cv, dx, dy)
    return out


def noise(x, y, seed=0):
    n = (x * 374761393 + y * 668265263 + seed * 2147483647) & 0xFFFFFFFF
    n = ((n ^ (n >> 13)) * 1274126177) & 0xFFFFFFFF
    return ((n ^ (n >> 16)) & 0xFFFF) / 65535.0


def dither(x, y, level):
    """Ordered 4x4 dither: True where `level` (0..1) is reached."""
    m = ((0, 8, 2, 10), (12, 4, 14, 6), (3, 11, 1, 9), (15, 7, 13, 5))
    return (m[y % 4][x % 4] + 0.5) / 16.0 < level


BREATH = [0, 1, 2, 1]
SWAY = [0, 0, 1, 0]
