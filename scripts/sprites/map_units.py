"""Map unit sprites (32x32, facing down) for the S0 proportion sheet."""
from pixkit import Canvas

# Each sprite: a character grid and a legend of char -> (ramp, tone, tex).
# Lowercase letters shade automatically; tones 0..4 force a ramp step.
SPRITES = {}

SPRITES["warrior"] = ([
    "......hhhh..........",
    ".....hhhhhh.........",
    ".....HHHHHH.........",
    ".....heHHeh.........",
    ".....kkhhkk.........",
    ".....mdddddm........",
    "....mmmdddmmm.......",
    "...mmmmmmmmmmm......",
    "..mmmsssssssmmm.....",
    "..mmmsssSsssmm.rrr..",
    "..mmmsssSsssmrrwwwr.",
    "..mmmsssSsssmrwwwwwr",
    "..mmmsssSsssmrwwbwwr",
    "..mmmllllllllrwwwwwr",
    "..kmmsssSsssmrwwwwwr",
    ".gggmsssSsssmmrwwwr.",
    "..x.mmssSssmmm.rrr..",
    "..x.mmsssSssmm......",
    "..x.mmmssSsmmm......",
    "..x..mmm.mmmm.......",
    "..x..ppp..ppp.......",
    "..x..ppp..ppp.......",
    ".....ppp..ppp.......",
    ".....ppp..ppp.......",
    ".....fff..fff.......",
    ".....fff..fff.......",
    "....ffff..ffff......",
], {
    "h": ("iron", -1), "H": ("iron", 1), "e": ("W0", None), "k": ("skin", -1),
    "d": ("hair", -1), "m": ("mail", -1, "mail"), "s": ("oxblood", -1), "S": ("oxblood", 1),
    "l": ("leather", -1), "r": ("iron", -1), "w": ("wood", -1, "wood"), "b": ("brass", 3),
    "g": ("brass", -1), "x": ("steel", 3), "p": ("hair", 3), "f": ("leather", -1),
})

SPRITES["rogue"] = ([
    "......cccc.........",
    ".....cccccc........",
    ".....cCCCCc........",
    ".....ceCCec........",
    ".....cqqqqc........",
    ".....cqqqqc........",
    "....ccqqqqcc.......",
    "...cccccccccc......",
    "...cjjjjjjjjc......",
    "..ccjjjJjjjjcc.....",
    "..cjjjjJjjjjjc.....",
    "..jjjjjJjjjjjj.....",
    "..jjjjjJjjjjjj.....",
    "..jjqqqqqqqqjj.....",
    "..kjjjjJjjjjjk.....",
    ".xk.jjjJjjjj.kx....",
    ".x..jjj..jjj..x....",
    ".x..ppp..ppp..x....",
    "....ppp..ppp.......",
    "....ppp..ppp.......",
    "....ppp..ppp.......",
    "....ppp..ppp.......",
    "....ppp..ppp.......",
    "....fff..fff.......",
    "....fff..fff.......",
    "....fff..fff.......",
    "...ffff..ffff......",
], {
    "c": ("charcoal", -1), "C": ("charcoal", 0), "e": ("C2", None), "q": ("plum", -1),
    "j": ("leather", -1), "J": ("leather", 1), "k": ("skin", -1), "x": ("steel", 3),
    "p": ("charcoal", 2), "f": ("leather", -1),
})

SPRITES["ranger"] = ([
    ".w....cccc.........",
    "..w..cccccc........",
    "..ww.cCCCCc........",
    "...w.ceCCec........",
    "...wckkkkkkc.......",
    "....ccdkkdcc.......",
    "...cccdddddccc.....",
    "..ccjjjjjjjjjcc....",
    "..ccjjjjjjjjjcc....",
    "..ccjjjJjjjjjcc....",
    "..ccjjjJjjjjjcc....",
    "..ccjjjJjjjjjcc....",
    "..ccbbbbbbbbbcc....",
    "..ckjjjJjjjjjkc....",
    "..ccjjjJjjjjjcc....",
    "..cc.jjjjjjj.cc....",
    "..cc.jjj.xjj.cc....",
    "..ccppp...pppcc....",
    "..cCppp...pppCc....",
    "...CCpp...ppCC.....",
    "....ppp...ppp......",
    "....ppp...ppp......",
    "....ppp...ppp......",
    "....fff...fff......",
    "....fff...fff......",
    "....fff...fff......",
    "...ffff...ffff.....",
], {
    "c": ("moss", -1, "cloth"), "C": ("moss", 1), "e": ("W0", None), "k": ("skin", -1),
    "d": ("hair", -1), "j": ("tan", -1), "J": ("tan", 1), "b": ("leather", 1),
    "w": ("wood", 3), "x": ("steel", 3), "p": ("leather", 2), "f": ("leather", -1),
})

SPRITES["cleric"] = ([
    "......mmmm..........",
    ".....mmmmmm.........",
    ".....mkkkkm.........",
    ".....mekkem.........",
    ".....mkkkkm.........",
    ".....mmkkmm.........",
    "....mmmmmmmm........",
    "...mmmmimmmmm.......",
    "..mmbuuuuuubmm......",
    "..mmbuuiuuubmm......",
    "..mmbuiiiuubmm......",
    "..mmbuuiuuubmm......",
    "..mmbuuuuuubmm......",
    "..mmlllllllllm......",
    ".rkmbuuuuuubmk......",
    "rrrmbuuuUuubm.......",
    ".rr.mbuuUuubm.......",
    "..w.mbuuUuubm.......",
    "..w.mmbuuuubmm......",
    "..w..mmm.mmmm.......",
    "..w..ppp..ppp.......",
    ".....ppp..ppp.......",
    ".....ppp..ppp.......",
    ".....ppp..ppp.......",
    ".....fff..fff.......",
    ".....fff..fff.......",
    "....ffff..ffff......",
], {
    "m": ("mail", -1, "mail"), "k": ("skin", -1), "e": ("W0", None), "u": ("wool", -1, "cloth"),
    "U": ("wool", 1), "b": ("brass", -1), "i": ("iron", 2), "l": ("leather", -1),
    "r": ("iron", -1), "w": ("wood", 2), "p": ("hair", 3), "f": ("leather", -1),
})

SPRITES["wizard"] = ([
    "...........g.......",
    "......sss.ggg......",
    ".....sssss.w.......",
    "....sSSSSSsw.......",
    "....sSeSeSsw.......",
    "....sSkkkSsw.......",
    "....ssdddssw.......",
    "...ssddddddsw......",
    "..sssdddddsskw.....",
    "..sssdddddsssw.....",
    "..sssSddddsssw.....",
    "..ssssSdddsss.w....",
    "..sssssSdssss.w....",
    "..ssllllllllss.w...",
    "..ksssSssssssk.w...",
    "...sssSsssssss.w...",
    "...sssSssssssss.w..",
    "...sssSssssssss.w..",
    "...sssSsssSssss.w..",
    "...ssssSsssSssss.w.",
    "...ssssSsssSssss.w.",
    "..sssssSsssSsssss..",
    "..sssssSsssSsssss..",
    "..sssssssssssssss..",
    "...fff.......fff...",
], {
    "s": ("slate", -1, "cloth"), "S": ("slate", 1), "e": ("B6", None), "k": ("skin", -1),
    "d": ("beard", -1), "w": ("wood", 2), "g": ("glow", 2), "l": ("pewter", -1),
    "f": ("leather", -1),
})

SPRITES["witch"] = ([
    "......vvvv..........",
    ".....vvvvvv.........",
    "....vvVVVVvv........",
    "....vVeVVeVv........",
    "....vVkkkkVv........",
    "....vvVkkVvv........",
    "...vvaaaaaavv.......",
    "..vvaaaoaoaaavv.....",
    "..vaaaaaaaaaaav.....",
    "..vaaAaaaaAaaav.....",
    "..vvaAaoaaAaavv.....",
    "..vvvaaaaaaavvv.....",
    "..vvvvAaaaAvvvv.....",
    "..kvvvvbbbvvvvk.....",
    ".w.vvvvvVvvvvv......",
    "..wvvvvvVvvvvvv.....",
    "..wvvvvVvvVvvvv.....",
    "..wvvvvVvvVvvvvv....",
    "...vvvvVvvvVvvvv....",
    "...vvvVvvvvVvvvv....",
    "..vvvvVvvvvvVvvvv...",
    "..vvvvVvvvvvVvvvv...",
    "..v.vvvvvvvvvvv.v...",
    "....v.vv.v.vv.v.....",
    ".....fff...fff......",
], {
    "v": ("heather", -1, "cloth"), "V": ("heather", 1), "a": ("ashmoss", -1, "cloth"),
    "A": ("ashmoss", 1), "e": ("W0", None), "k": ("skin", -1), "o": ("bone", 3),
    "b": ("bone", 2), "w": ("wood", 3), "f": ("leather", -1),
})


def build(name):
    rows, legend = SPRITES[name]
    c = Canvas(32, 32, band=1)
    mats = {}
    leg = {}
    for ch, spec in legend.items():
        ramp, tone = spec[0], spec[1]
        tex = spec[2] if len(spec) > 2 else None
        if tone is None:
            leg[ch] = (c.lit(ramp), -1)
            continue
        key = (ramp, tex)
        if key not in mats:
            mats[key] = c.m(ramp, tex=tex)
        leg[ch] = (mats[key], tone)
    width = max(len(r) for r in rows)
    ox = 16 - width // 2 + 2
    oy = 30 - len(rows) + 1  # feet baseline at frameHeight - 2
    c.ascii(rows, leg, ox, oy)
    return c


ORDER = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"]
