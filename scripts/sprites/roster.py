"""The S3 battle roster: every class and enemy of the default world with its
size class, family, in-game names and idle drawer
(docs/designs/2026-10-05-sprite-specification.md, S3 enemy table)."""
import beasts
import figures
import giants
import humanoids
import promoted
import summoned
from kit import BREATH, shift
from pixels import Canvas, rect, line, thick, ellipse

FRAME = {"S": 48, "M": 64, "L": 72, "XL": 96}
# Large and boss units are drawn on a bigger canvas and shrunk by 3/4, so they
# stand about 1.5x a person and cover two formation cells, not three, at 1x.
DRAW = {"L": 96, "XL": 128}

CLASS_IDS = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer",
             "halberdier", "samurai", "shaman", "dollmaster"]


class Unit:
    def __init__(self, uid, size, family, draw, names=(), variant_of=None, note="", floating=False):
        self.id, self.size, self.family, self.floating = uid, size, family, floating
        self.draw, self.names, self.variant_of, self.note = draw, list(names), variant_of, note
        self.frame = FRAME[size]

    def frames(self):
        """Four idle frames, shifted as one so the feet rest on row frame-2 (the anchor rule)."""
        out = [self.draw(t) for t in range(4)]
        if self.size in DRAW:
            out = [shrink(c, self.frame) for c in out]
        assert all(c.w == self.frame and c.h == self.frame for c in out), self.id
        if self.floating:
            return out
        low = max(max(y for y in range(c.h) if any(p is not None for p in c.g[y])) for c in out)
        d = self.frame - 2 - low
        return [shift(c, 0, d) for c in out] if d else out


def shrink(cv, n):
    """Nearest-neighbour shrink of a square canvas to n x n, then re-close the
    outline wherever a dropped row or column opened it."""
    out = Canvas(n, n)
    s = cv.w / n
    for y in range(n):
        for x in range(n):
            out.g[y][x] = cv.g[int((y + 0.5) * s)][int((x + 0.5) * s)]
    out.outline()
    return out


def dummy_training(t, W=64):
    """A straw dummy on a post: crossbar arms, a sack head, a rope belt."""
    cv = Canvas(W, W)
    fy = W - 3
    b = [0, 0, 1, 0][t]
    cv.part(rect(30, fy - 36, 33, fy), "wood")  # post
    cv.part(rect(24, fy, 40, fy), "wood", flat="d")  # footing
    cv.part(rect(14, fy - 31, 49, fy - 28), "wood")  # crossbar
    body = ellipse(32, fy - 22 + b, 8, 11)
    cv.part(body, "ochre")
    cv.part(ellipse(32, fy - 40 + b, 6, 7), "wool")  # sack head
    cv.fill({(34, fy - 41 + b), (35, fy - 41 + b)}, "outline")
    cv.fill(line(29, fy - 35 + b, 36, fy - 35 + b), "leather.d")  # tied neck
    cv.fill(line(24, fy - 14 + b, 40, fy - 14 + b), "leather.m")  # rope belt
    for x, y in ((14, fy - 29), (13, fy - 30), (49, fy - 29), (50, fy - 30), (27, fy - 11),
                 (38, fy - 10), (28, fy - 30), (38, fy - 31), (31, fy - 46)):
        cv.put(x, y + b, "ochre.l")
    for x, y in ((29, fy - 22), (35, fy - 26), (31, fy - 16)):  # stitched patches
        cv.fill({(x, y + b), (x + 1, y + b)}, "wool.d")
    cv.outline()
    return cv


UNITS = []


def add(uid, size, family, draw, names=(), variant_of=None, note="", floating=False):
    UNITS.append(Unit(uid, size, family, draw, names, variant_of, note, floating))


H = humanoids.H
for cls in CLASS_IDS:
    add(cls, "M", "class", (lambda c: lambda t: figures.battle_idle(c, t))(cls),
        [] if cls == "adventurer" else [f"recruit {cls}"])
# S5: the promoted classes (level 10 and the elite classes built so far), keyed by class id.
for cls in promoted.CLASS_IDS:
    add(cls, "M", "promoted class", (lambda c: lambda t: promoted.battle_idle(c, t))(cls),
        variant_of=promoted.LINEAGE[cls], note=f"{promoted.LINEAGE[cls]} line")
add("angel", "L", "summon", summoned.angel, ["angel"], note="a Hierarch's summon")
add("demon", "L", "summon", summoned.demon, ["demon"], note="a Demonologist's summon")
add("doll", "S", "construct", summoned.doll, ["doll"], note="a Doll Master's painted wooden puppet (39d)")
add("unknown-humanoid", "M", "fallback", H["unknown-humanoid"], note="fallback silhouette")
add("unknown-beast", "M", "fallback", beasts.unknown_beast, note="fallback silhouette")
add("unknown-large", "L", "fallback", giants.unknown_large, note="fallback silhouette")

add("rat", "S", "rodent", beasts.rat, ["rat"])
add("rat-big", "S", "rodent", beasts.rat_big, ["big rat"], "rat")
add("wolf-timber", "M", "canine", beasts.wolf, ["timber wolf"])
add("wolf-snow", "M", "canine", beasts.wolf_snow, ["snow wolf"], "wolf-timber")
add("dog-junkyard", "M", "canine", beasts.dog_junkyard, ["junkyard dog"])
add("spider-hatchling", "S", "spider", beasts.spider_hatchling, ["spider hatchling", "baby spider"])
add("spider-large", "M", "spider", beasts.spider_large, ["large spider"])
add("spider-warrior", "M", "spider", beasts.spider_warrior, ["spider warrior"], "spider-large")
add("spider-queen", "XL", "spider", beasts.spider_queen, ["spider queen"], note="boss")
add("skeleton", "M", "undead", H["skeleton"], ["skeleton"])
add("bone-warden", "M", "undead", H["bone-warden"], ["bone warden"])
add("bonecrafter", "M", "undead", H["bonecrafter"], ["bonecrafter"])
add("lich", "L", "undead", giants.lich, ["lich"], note="boss")
add("acolyte-dark", "M", "cultist", H["acolyte-dark"], ["dark acolyte", "dark acolyte trainer"])
add("grave-chanter", "M", "cultist", H["grave-chanter"], ["grave chanter"])
add("brigand", "M", "bandit", H["brigand"], ["road brigand"])
add("ruffian", "M", "bandit", H["ruffian"], ["ruffian"])
add("ruffian-dangerous", "M", "bandit", H["ruffian-dangerous"], ["dangerous ruffian"], "ruffian")
add("ruffian-enforcer", "M", "bandit", H["ruffian-enforcer"], ["ruffian enforcer"])
add("poacher", "M", "bandit", H["poacher"], ["lake poacher"])
add("poacher-shieldman", "M", "bandit", H["poacher-shieldman"], ["poacher shieldman"])
add("bonesetter", "M", "bandit", H["bonesetter"], ["poacher bonesetter", "back-alley bonesetter"])
add("shadow-trainee", "M", "shadow guild", H["shadow-trainee"], ["shadow trainee"])
add("shadow-master", "M", "shadow guild", H["shadow-master"], ["shadow master"], "shadow-trainee")
add("goblin", "M", "goblin", H["goblin"], ["goblin"], note="base for the variants and random encounters")
add("goblin-hexer", "M", "goblin", H["goblin-hexer"], ["goblin hexer"], "goblin")
add("goblin-loot", "M", "goblin", H["goblin-loot"], ["loot goblin"], "goblin")
add("goblin-shaman", "M", "goblin", H["goblin-shaman"], ["goblin shaman"], "goblin")
add("faerie", "S", "fey", giants.faerie, ["faerie folk"])
add("imp-forest", "S", "fey", giants.imp_forest, ["forest imp"])
add("fungus", "M", "plant", giants.fungus, ["sentient fungus"])
add("ent", "L", "plant", giants.ent, ["ent"])
add("ogre-forest", "L", "ogre", giants.ogre, ["forest ogre"])
add("crocodile", "L", "reptile", beasts.crocodile, ["crocodile"])
add("creeper-cave", "M", "cave", beasts.creeper_cave, ["cave creeper"])
add("creeper-abyssal", "M", "cave", beasts.creeper_abyssal, ["abyssal creeper"], "creeper-cave")
add("stalker-cave", "M", "cave", H["stalker-cave"], ["cave stalker"])
add("bats-echo", "S", "cave", beasts.bats, ["echo bats"], floating=True, note="a flying swarm; no ground baseline")
add("ice-warrior", "M", "ice", H["ice-warrior"], ["ice warrior"])
add("ice-guardian", "L", "ice", giants.ice_guardian, ["ice guardian"])
add("snow-floof", "S", "snow", giants.snow_floof, ["snow floof"])
add("dummy-training", "M", "training", dummy_training, ["training dummy"])
add("straw-footman", "M", "training", H["straw-footman"], ["straw footman"])
add("straw-archer", "M", "training", H["straw-archer"], ["straw archer"], "straw-footman")
add("guard", "M", "town guard", H["guard"], ["guard"])
add("guard-royal", "M", "town guard", H["guard-royal"], ["king's guard"], "guard")
add("guard-captain", "M", "town guard", H["guard-captain"], ["captain of the guard"], "guard")

BY_ID = {u.id: u for u in UNITS}
