"""Battle UI art (art set S3): status, role, morale and condition icons, plus the
formation cell, acting and target markers, the fallen and surrendered markers
and the health-bar frame.  Icons are 16x16 pixel maps (see icons.LEGEND).

Art direction note: the specification lists "sweat drops" for Winded; the
direction bans sweat-drops and other cartoon emotes, so Winded is a ragged
puff of breath only.
"""
import icons
from icons import icon
from pixels import Canvas, ellipse, line, rect

STATUS = {}
ROLES = {}
MORALE = {}
CONDITIONS = {}


def reg(table, name, rows, **kw):
    table[name] = lambda: icon(rows, **kw)


# -- status icons -------------------------------------------------------------------

STATUS["bleeding"] = icons.st_bleeding
STATUS["stunned"] = icons.st_stunned

reg(STATUS, "staggered", [
    "..............",
    "...PPPPP......",
    "..Pq...qP.....",
    "..P..PPP.P....",
    "..P.Pq..qP....",
    "..P.P.PP.P....",
    "..P.Pq.q.P....",
    "..Pq.PPP.P....",
    "...Pq...qP....",
    "....PPPPP.....",
    "..............",
    ".u.u.u.u.u.u..",
])
reg(STATUS, "knocked-down", [
    "..............",
    "..............",
    "........uU....",
    ".......uUUu...",
    "..N....uuuu...",
    "..NNN...Nu....",
    "...NNNNNNN....",
    "...QNNQ.NNN...",
    "..Qll..Q.lk...",
    "..lkk....kk...",
    "yyyyyyyyyyyy..",
])
reg(STATUS, "armor-broken", [
    "..iiiiiiiiii..",
    ".iTTTTiTTTTTi.",
    ".iTVTTjTTTTAi.",
    ".iTTTTjTTTTAi.",
    "..iTTTj.TTAi..",
    "..iTTjTTTTAi..",
    "...iTjjTTAi...",
    "...iTT.jAAi...",
    "....iTTTAi....",
    ".....iiii.....",
])
reg(STATUS, "exposed", [
    "......rr......",
    "....rrRRrr....",
    "..rrR....Rrr..",
    "..rR......Rr..",
    ".rR...rr...Rr.",
    "rrrrrrRRrrrrrr",
    ".rR...rr...Rr.",
    "..rR......Rr..",
    "..rrR....Rrr..",
    "....rrRRrr....",
    "......rr......",
])
reg(STATUS, "hobbled", [
    "....FF........",
    "....FF........",
    "....FF........",
    "....FF........",
    "...iIIi.......",
    "...ijjiji.....",
    "...iIIi.j.....",
    "...FFFFl.j....",
    "..FFFFFlk.j...",
    "..lllllkk..j..",
    "..........jIj.",
    "..........jjj.",
])
reg(STATUS, "burning", [
    ".......e......",
    "......ee......",
    ".....eEe.e....",
    "....eeEee.e...",
    "...eeEEEeee...",
    "...eEEyEEee...",
    "..eeEyyyEee...",
    "..eEyYYYyEe...",
    "..eEyYYYyEe...",
    "...eEyyyEe....",
    "....eeeee.....",
    "...ffffff.....",
])
reg(STATUS, "overloaded", [
    "...V.....V....",
    "..VBV...VBV...",
    "...VBV.VBV....",
    "....VBVBV.....",
    ".....VBV......",
    "....FFFFF.....",
    "...FFFFFFF....",
    "...FFFFFFF....",
    "...F1FFFFF....",
    "....FFFFF.....",
])
reg(STATUS, "poisoned", [
    ".....m........",
    "....mMm.......",
    "...mMmmm......",
    "..mMmmmmn.....",
    "..mmmmmmn.....",
    "..mmmmmnn.....",
    "...mmmnn......",
    "....nnn..M....",
    "........MmM...",
    ".....M..MmM...",
    "....MmM.......",
])
reg(STATUS, "asleep", [
    "..............",
    "..........OO..",
    "........OO....",
    ".......OO.....",
    "..ooooo.OO....",
    ".o.....o.OOOO.",
    "o.......o.....",
    ".o.o.o.o......",
    "..o.o.o.......",
])
reg(STATUS, "paralyzed", [
    "....FFF.......",
    "....FFF.......",
    "ppppppppppp...",
    "PPPPPPPPPPP...",
    "..NNNNNNN.....",
    "..NNNNNNN.....",
    "ppppppppppp...",
    "PPPPPPPPPPP...",
    "..NN...NN.....",
    "..NN...NN.....",
    "ppppppppppp...",
])
reg(STATUS, "blighted", [
    "......zzzz....",
    ".....zyyyyz...",
    "....zyYyyyz...",
    "....zyyyzz....",
    "..rrr.zz.rrr..",
    ".rRRRrrrrRRRr.",
    ".rRRrRrrrrRrx.",
    ".rrrRrxxrrrrx.",
    "..rrrrrxrrxx..",
    "...xrrrrrxx...",
    "....xrrrxx....",
    ".....xxxx.....",
])
reg(STATUS, "weakened", [
    "..iiii........",
    ".iTTTTi.......",
    ".iTTTTi.......",
    "..iTTi........",
    "...FF.........",
    "...FF.........",
    "....FF........",
    "....FFF.......",
    ".....FFl......",
    "......ll......",
    "....rrrrr.....",
    ".....rrr......",
    "......r.......",
])
reg(STATUS, "hamstrung", [
    "....FF........",
    "....FF........",
    "....FF........",
    "....FF.rR.....",
    "....FFrRr.....",
    "....FrRrF.....",
    "....rRrFF.....",
    "...lrrFFFl....",
    "..llllFFllk...",
    "..lllllllkk...",
])
reg(STATUS, "winded", [
    "..............",
    "...OOO........",
    "..ONNNOO......",
    ".ONNNNNNOO....",
    "OONNNQNNNNOO..",
    ".OQNNNNQNNQO..",
    "..OQQNNNQQQ...",
    "....OQQQ......",
])
reg(STATUS, "tackled", [
    "..uu......uu..",
    ".uUUu....uUUu.",
    ".uUUu....uUUu.",
    "..uu..ee..uu..",
    ".NNNN.EEe.NNNN",
    ".NNNNNyEeNNNN.",
    ".NNNN.EEe.NNN.",
    ".QNN...e..NNQ.",
    ".ll.......ll..",
    ".lk.......lk..",
])
reg(STATUS, "cold", [
    "......V.......",
    "..V...V...V...",
    "...V..V..V....",
    "....V.V.V.....",
    "VVVVVVVVVVVVV.",
    "....V.V.V.....",
    "...V..V..V....",
    "..V...V...V...",
    "......V.......",
])
reg(STATUS, "regenerating", [
    "......MMM.....",
    "....MMmmmM....",
    "...Mmm...mM...",
    "...mm..MM.m...",
    "...mM.Mmm.m...",
    "....mM.Mm.M...",
    ".....mMMm.....",
    "....M.........",
    "...Mm.........",
    "....M.........",
])
reg(STATUS, "lit", [
    "......HH......",
    ".....H..H.....",
    "....KKKKKK....",
    "..y.HYYYYH.y..",
    "...yHYEEYHy...",
    "....HEeeEH....",
    "yy..HEeeEH..yy",
    "....HYEEYH....",
    "...yHYYYYHy...",
    "..y.KKKKKK.y..",
])
reg(STATUS, "hidden", [
    "..............",
    "....OOOOOO....",
    "..OO......OO..",
    ".O...ooo....O.",
    "o.o.o.Oo.o.o.o",
    ".O..o.oo..o.O.",
    "..OO..oo..OO..",
    "....OOOOOO....",
])
reg(STATUS, "chanting", [
    "....O...O.....",
    "...OOO.O.O.O..",
    "....O...O.OOO.",
    "..........O...",
    "......O.......",
    ".....OOO......",
    "...FF.O.FF....",
    "..FFFF.FFFF...",
    "..FFFFFFFFF...",
    "...FFFFFFF....",
    "....FFFFF.....",
])
reg(STATUS, "winding-up", [
    "...........V..",
    "OO........V...",
    ".OO...ssss.V..",
    "OO...sSFFss...",
    ".OO..sFFFFss..",
    "OO...sFFFFss..",
    "......sFFss...",
    "......FFFF....",
    ".......FF.....",
])
reg(STATUS, "warded", [
    ".NN...ZZZZZ...",
    "NNNN.ZXZZZZZ..",
    "NNNN.ZXZZZZZ..",
    ".NN..ZXZZZZz..",
    ".NN..ZZZZZZz..",
    "NNNN.ZZZZZz...",
    "NNNN..ZZZz....",
    ".ll....Zz.....",
])
reg(STATUS, "wounded-light", [
    "..........OO..",
    ".........OOOO.",
    "........OOOOO.",
    ".......OOQQO..",
    "......OOQQO...",
    ".....OOQQO....",
    "....OOQQO.....",
    "...OOQQO......",
    "..OOOOO.......",
    "..OOOO........",
])
reg(STATUS, "wounded-lasting", [
    "..........OO..",
    ".........OOOO.",
    "........OrRrO.",
    ".......OrRrQO.",
    "......OOrRQO..",
    ".....OrrRQO...",
    "....OOrRQO....",
    "...OrrRQO.....",
    "..OOOrOO......",
    "..OOOOO..r....",
    "........rRr...",
])
reg(STATUS, "dread", [
    "...DDD........",
    "..DCCDD..D....",
    "..DCDDDDDDD...",
    "...DDDpDDD....",
    "....DDpD......",
    "...DDpp.......",
    "..N.NN.N......",
    "..NNNNNN......",
    "..NN.N.NN.....",
    "...N.N.N......",
    "..lk...kl.....",
])
reg(STATUS, "tracked", [
    "......ll......",
    ".....llll.....",
    "....l.ll.l....",
    "....ll..ll....",
    "..............",
    "..ll..........",
    ".llll..ll.....",
    "l.ll.l.llll...",
    "ll..ll.l.ll.l.",
    "...............",
])

# -- roles ------------------------------------------------------------------------------

ROLES["fighter"] = icons.role_fighter
ROLES["healer"] = icons.role_healer
reg(ROLES, "caster", [
    "........EEE...",
    ".......EEyEE..",
    "....E..EyYyE..",
    ".....E.EEyEE..",
    "......wEEEE...",
    ".....ww.E.....",
    "....ww........",
    "...ww.........",
    "..ww..........",
    ".ww...........",
])
reg(ROLES, "guardian", [
    ".TTTTTTTTTTT..",
    ".TVVTTTTTTTA..",
    ".TVTTTjjTTTA..",
    ".TVTTTjjTTTA..",
    ".TTTjjjjjTTA..",
    ".TTTTTjjTTTA..",
    "..TTTTjjTTA...",
    "..TTTTjjTTA...",
    "...TTTjjTA....",
    "....TTjjA.....",
    ".....TTA......",
])
reg(ROLES, "controller", [
    "..pp....pp....",
    ".pPPp..pPPp...",
    ".pP.pppp.Pp...",
    "..pp.PPP.p....",
    "...pPP..Pp....",
    "...pP....Pp...",
    "..pp.PPPP.pp..",
    ".pP.pppppp.Pp.",
    ".pPPp....pPPp.",
    "..pp......pp..",
])

# -- morale -------------------------------------------------------------------------------

reg(MORALE, "hold", [
    "..wrrrrrr.....",
    "..wrRRRRrr....",
    "..wrRrrrrrr...",
    "..wrrrrrrr....",
    "..wrrrxxx.....",
    "..wxx.........",
    "..w...........",
    "..w...........",
    "..w...........",
    "kkwkk.........",
])
MORALE["yield"] = icons.st_yield
reg(MORALE, "flee", [
    "...........uu.",
    "..........uUu.",
    ".....llll..u..",
    "....lLLLlk....",
    "...lLLLLlk....",
    "..lllllllk....",
    ".lkkkkkkkk....",
    "yy.yy.y.yy....",
    "..y.y...y.....",
])
reg(MORALE, "nerve", [
    ".......e......",
    "......eE......",
    ".....eEEe.....",
    "....eEyEe.....",
    "....eEyyEe....",
    "....eEYyEe....",
    "....eEYyEe....",
    ".....eyyEe....",
    ".....eEEe.....",
    "......ee......",
    "....ssssss....",
])
reg(MORALE, "shaken", [
    ".........f....",
    "........fe....",
    ".......fe.....",
    "......fe......",
    "......ee.f....",
    ".....feEe.....",
    ".....eEEe.....",
    ".....feEe.....",
    "......fe......",
    "....ssssss....",
])

# -- battlefield conditions ------------------------------------------------------------------

reg(CONDITIONS, "dark", [
    "..dddddddddd..",
    ".dDDDDDDDDDDd.",
    "dDDDUUUUDDDCDd",
    "dDDUUUDDDDDDDd",
    "dDDUUDDDDCDDDd",
    "dDDUUDDDDDDDDd",
    "dDDUUUDDDDDCDd",
    "dDDDUUUUDDDDDd",
    ".dDDDDDDDDDDd.",
    "..dddddddddd..",
])
reg(CONDITIONS, "ambush", [
    "..g...G...g...",
    ".gGg.gGg.gGg..",
    "gGgGgGgGgGgGg.",
    ".gggoooooggg..",
    "gGgo.EOO.ogGg.",
    ".ggo.EOOO.oggg",
    "gGgoo....oogGg",
    ".gggoooooggg..",
    "hGhhGhhGhhGhh.",
])
reg(CONDITIONS, "narrow", [
    "ssss......ssss",
    "sSSs......sSSs",
    "sSSs......sSSs",
    "ssSss....ssSss",
    ".ssSss..ssSss.",
    "..ssSs..sSss..",
    "...ssy..yss...",
    "....sy..ys....",
    "....sy..ys....",
    "....yy..yy....",
])
reg(CONDITIONS, "cold", [
    "......V.......",
    ".....VVV......",
    ".....VBV......",
    ".....VBV......",
    ".....VBV......",
    "....VVBVV.....",
    "....VBBBV.....",
    "....VBBBV.....",
    ".....VBV......",
    "......V.......",
    "..V.......V...",
    "...V.....V....",
])
reg(CONDITIONS, "fatigue", [
    "..............",
    "....FFF.......",
    "....FFF.......",
    "...FFFFF......",
    "...NNNNNN.....",
    "..NNNNNNNNN...",
    "...NNQNNNNN...",
    "....NNQ.NN....",
    "....NN..NN....",
    "....ll..ll....",
])
reg(CONDITIONS, "flanked", [
    "....yyyyyy....",
    "..yyYYYYYYyy..",
    ".yYy......Yyy.",
    ".yy.........yy",
    ".yy..ssss..yyy",
    "......sSSs..yy",
    "......ssss.yYy",
    ".........yyyy.",
])
reg(CONDITIONS, "cluster", [
    "..............",
    "....ee.ee.....",
    "...eEEeEEe.ee.",
    "...eEyyyEe.ee.",
    "....eyYye.....",
    "...eEyyyEe.ee.",
    "...eEEeEEe.ee.",
    "....ee.ee.....",
])


# -- formation and battle markers ------------------------------------------------------------

def _ellipse_ring(w, h, color, fill=None, dither=False):
    cv = Canvas(w, h)
    cx, cy = (w - 1) / 2, (h - 1) / 2
    for y in range(h):
        for x in range(w):
            d = ((x - cx) / (w / 2 - 0.5)) ** 2 + ((y - cy) / (h / 2 - 0.5)) ** 2
            if 0.62 <= d <= 1.0:
                if not dither or (x + y) % 2 == 0:
                    cv.put(x, y, color)
            elif fill and d < 0.62 and (x + y) % 2 == 0 and dither:
                cv.put(x, y, fill)
    return cv


def cell():
    return _ellipse_ring(32, 16, "stone.m", "stone.d", dither=True)


def cell_acting_frames():
    out = []
    for t in range(4):
        cv = Canvas(32, 16)
        ring = _ellipse_ring(32, 16, ["bone.d", "bone.m", "bone.l", "bone.m"][t])
        cv.blit(ring)
        if t in (1, 2):  # pulse swells: an inner glow ring
            inner = _ellipse_ring(22, 10, "bone.d")
            cv.blit(inner, 5, 3)
        out.append(cv)
    return out


def cell_targeted_frames():
    out = []
    for t in range(2):
        cv = Canvas(32, 16)
        cv.blit(_ellipse_ring(32, 16, ["oxblood.m", "oxblood.l"][t]))
        cv.blit(_ellipse_ring(24, 12, "oxblood.d", dither=True), 4, 2)
        out.append(cv)
    return out


def acting_arrow_frames():
    out = []
    for t in range(2):
        cv = Canvas(8, 8)
        y = t
        cv.fill({(3, y), (4, y), (3, y + 1), (4, y + 1), (3, y + 2), (4, y + 2), (1, y + 3), (2, y + 3),
                 (3, y + 3), (4, y + 3), (5, y + 3), (6, y + 3), (2, y + 4), (3, y + 4), (4, y + 4),
                 (5, y + 4), (3, y + 5), (4, y + 5)}, "bone.l")
        cv.outline()
        out.append(cv)
    return out


def fallen():
    return icon([
        "..............",
        "..............",
        "....T.........",
        "....TA........",
        "...iTTi.......",
        "....T.NNNN....",
        "....TNNNQNN...",
        "....wNNNQQ....",
        "....wlll......",
        "..yyywyyyyy...",
    ])


def surrendered():
    return icon([
        "..OOO.........",
        ".ONNNOO.......",
        ".ONNNNNOO.....",
        ".OQNNNQQ......",
        "..OQQQ........",
        "..w...........",
        "..w...........",
        "..w...........",
        "..w...........",
        "yywyy.........",
    ])


def hp_frame():
    cv = Canvas(32, 6)
    cv.fill(rect(0, 0, 31, 5), "outline")
    cv.fill(rect(1, 1, 30, 4), "iron.d")
    return cv


MARKERS = dict(cell=cell, fallen=fallen, surrendered=surrendered, hp_frame=hp_frame)
