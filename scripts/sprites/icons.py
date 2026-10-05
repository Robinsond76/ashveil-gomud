"""Sample 16x16 icons for the S0 icon sheet."""
from pixkit import Canvas, RAMPS

RAMPS.setdefault("water", ["B1", "B2", "B3", "B4", "B6"])
RAMPS.setdefault("berry", ["R0", "R1", "R3", "R4", "R6"])
RAMPS.setdefault("leaf", ["G1", "G2", "G4", "G5", "G6"])
RAMPS.setdefault("blood", ["R0", "R1", "R2", "R4", "R5"])
RAMPS.setdefault("rag", ["W3", "W4", "W6", "S5", "S6"])
RAMPS.setdefault("moon", ["W4", "W5", "W6", "S5", "S6"])
RAMPS.setdefault("star", ["O1", "O3", "O4", "O5", "O6"])
RAMPS.setdefault("heal", ["G3", "G4", "G6", "G7", "O6"])


def grid(rows, legend):
    c = Canvas(16, 16, band=1)
    leg, mats = {}, {}
    for ch, (ramp, tone) in legend.items():
        if ramp not in mats:
            mats[ramp] = c.m(ramp) if ramp in RAMPS else c.lit(ramp)
        leg[ch] = (mats[ramp], tone)
    c.ascii(rows, leg)
    return c


def water():
    return grid([
        "................",
        "................",
        ".......bb.......",
        "......bbbb......",
        "......bbbb......",
        ".....bbbbbb.....",
        ".....bhbbbb.....",
        "....bbhbbbbb....",
        "....bbbbbbbb....",
        "....bbbbbbbb....",
        ".....bbbbbb.....",
        "......bbbb......",
        "................",
        "..rrr......rrr..",
        "....rrrrrrrr....",
        "................",
    ], {"b": ("water", -1), "h": ("water", 4), "r": ("water", 3)})


def forage():
    return grid([
        "................",
        "...........ll...",
        "..........llll..",
        ".........lllll..",
        "...ll....Lllll..",
        "..llll..sLll....",
        "..lllll.s.......",
        "...lLLLss.......",
        "......s.........",
        ".....s.rr.......",
        "....s.rhrr.rr...",
        "...s..rrrrrhrr..",
        "......rrrr.rrr..",
        ".......rr.rr....",
        ".........rhrr...",
        "..........rr....",
    ], {"l": ("leaf", -1), "L": ("leaf", 1), "s": ("wood", 3), "r": ("berry", -1), "h": ("berry", 4)})


def bleeding():
    return grid([
        "................",
        ".......rr.......",
        ".......rr.......",
        "......rrrr......",
        "......rrrr....x.",
        ".....rrrrrr..x..",
        ".....rhrrrr.x...",
        "....rrhrrr.x....",
        "....rrrrr.x.r...",
        "....rrrr.x.rr...",
        "....rrr.x.rrr...",
        ".....r.x.rrr....",
        "......x.rrr.....",
        ".....x..........",
        "....x...........",
        "................",
    ], {"r": ("blood", -1), "h": ("blood", 4), "x": ("C6", -1)})


def stunned():
    c = Canvas(16, 16, band=1)
    ring = c.lit("W5", outline=False)
    for x, y in ((2, 8), (3, 7), (4, 6), (6, 5), (7, 5), (9, 5), (10, 5), (12, 6), (13, 7), (14, 8),
                 (13, 9), (12, 10), (10, 11), (9, 11), (7, 11), (6, 11), (4, 10), (3, 9)):
        c.px(x, y, ring)
    star = c.m("star")
    for cx, cy in ((4, 6), (12, 4), (8, 11)):
        c.px(cx, cy, star, 4)
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            c.px(cx + dx, cy + dy, star, 2)
        for dx, dy in ((2, 0), (-2, 0), (0, 2), (0, -2)):
            c.px(cx + dx, cy + dy, star, 1)
    return c


def fighter():
    return grid([
        "x..............x",
        ".xx..........xx.",
        ".xXx........xXx.",
        "..xXx......xXx..",
        "...xXx....xXx...",
        "....xXx..xXx....",
        ".....xXxxXx.....",
        "......xXXx......",
        "......xXXx......",
        ".....xXxxXx.....",
        "..gg.xx..xx.gg..",
        "...ggx....xgg...",
        "...gg......gg...",
        "..w.gg....gg.w..",
        ".w...........w..",
        "p..............p",
    ], {"x": ("steel", 3), "X": ("steel", 1), "g": ("brass", -1), "w": ("leather", 2), "p": ("brass", 3)})


def healer():
    c = grid([
        "................",
        "................",
        "................",
        "................",
        "................",
        "................",
        "....k.k.k.......",
        "....kkk.k.k.....",
        "....kkkkkkk.....",
        "...kkkkkkkk.....",
        "...kkkkkkkk.....",
        "....kkkkkkk.....",
        ".....kkkkk......",
        ".....kkkkk......",
        ".....sssss......",
        ".....sssss......",
    ], {"k": ("skin", -1), "s": ("wool", -1)})
    glow = c.m("heal", sep=False, outline=False)
    for y, row in enumerate(["....g.g.g.......", "...g.GGG.g......", "....GOOOG.......",
                             "...g.GGG.g......", "....g.g.g......."]):
        for x, ch in enumerate(row):
            if ch != ".":
                c.px(x, y + 1, glow, {"g": 1, "G": 2, "O": 4}[ch])
    return c


def yield_():
    return grid([
        "s...............",
        ".s..............",
        "..s.............",
        "...s............",
        "....s...........",
        ".....s..........",
        "......s.........",
        ".......sff......",
        "........sfff....",
        ".........sffff..",
        "........fsffFf..",
        "........ffsfFFf.",
        ".........ffsFFf.",
        "..........fFsFF.",
        "............FsF.",
        "..............s.",
    ], {"s": ("wood", 3), "f": ("rag", -1), "F": ("rag", 1)})


def dark():
    c = Canvas(16, 16, band=1)
    night = c.lit("B0")
    c.rect(1, 1, 14, 14, night)
    for x, y in ((3, 3), (12, 12), (2, 11)):
        c.dot(x, y, "B4")
    moon = c.m("moon")
    c.ellipse(4, 3, 12, 11, moon)
    c.ellipse(6, 2, 13, 9, night)
    c.dot(13, 5, "B3")
    return c


ORDER = [("water", water), ("forage", forage), ("bleeding", bleeding), ("stunned", stunned),
         ("fighter", fighter), ("healer", healer), ("yield", yield_), ("dark", dark)]
