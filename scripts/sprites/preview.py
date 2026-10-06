"""Contact sheet: every generated sprite on one image, for review."""
import json
import os

from PIL import Image, ImageDraw

BG = (96, 104, 84)
INK = (235, 230, 215)


def _load(out, rel):
    return Image.open(os.path.join(out, rel)).convert("RGBA")


def contact_sheet(out, dest, zoom=3):
    doc = json.load(open(os.path.join(out, "manifest.json")))
    files = doc["files"]
    rows = []  # (label, [images])

    def add(label, rels):
        rows.append((label, [_load(out, r) for r in rels]))

    add("S0 style frames (1x)", ["style/style-map.png", "style/style-battle.png"])
    add("S0 proportions: map, battle", ["style/proportions-map.png", "style/proportions-battle.png"])
    add("S0 palette, icon sample", ["style/palette.png", "style/icon-sample.png"])
    for cls in ("warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer"):
        add(f"map unit {cls}: idle, walk (rows down, up, side)",
            [f"map/units/{cls}/idle.png", f"map/units/{cls}/walk.png"])
    add("resources", [r for r in files if r.startswith("map/resources/")])
    add("markers", [r for r in files if r.startswith("map/markers/")])
    add("camp", [r for r in files if r.startswith("map/camp/")])
    add("app icons (shown at 1x)", [r for r in files if r.startswith("app/")])

    margin, gap = 12, 10
    W = 1500
    tiles = []
    y = margin
    placed = []
    for label, imgs in rows:
        z = 1 if label.startswith(("app icons", "S0 style", "S0 palette")) else zoom
        if label.startswith("S0 palette"):
            z = 8 if False else zoom
        x = margin
        row_h = 0
        placed.append((label, x, y))
        y += 14
        for im in imgs:
            zz = 2 if im.width * z > W // 2 and z > 1 else z
            if label.startswith("S0 palette") and im.width == 8:
                zz = 8
            big = im.resize((im.width * zz, im.height * zz), Image.NEAREST)
            if x + big.width + margin > W:
                x = margin
                y += row_h + gap
                row_h = 0
            tiles.append((big, x, y))
            x += big.width + gap
            row_h = max(row_h, big.height)
        y += row_h + gap + 6
    sheet = Image.new("RGBA", (W, y + margin), BG + (255,))
    d = ImageDraw.Draw(sheet)
    for label, x, yy in placed:
        d.text((x, yy), label, fill=INK)
    for big, x, yy in tiles:
        sheet.alpha_composite(big, (x, yy))
    os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
    sheet.convert("RGB").save(dest, optimize=True)


def contact_sheet_s23(out, dest):
    """Sheet for sets S2 and S3: backgrounds, formations at 1x, units, UI icons, terrain, landmarks."""
    import backgrounds
    import roster
    import scenes
    doc = json.load(open(os.path.join(out, "manifest.json")))
    files = doc["files"]
    W, margin, gap = 1500, 12, 10
    items = []  # (label, image, zoom)
    items.append(("S3 backgrounds (1x, the ground band is y 100-176)",
                  [_load(out, f"battle/backgrounds/{b}.png") for b in backgrounds.BACKGROUNDS], 1))
    f = lambda uid: roster.BY_ID[uid].draw(0)
    forms = []
    for bg, comp, ene in (
        ("forest", ["warrior", "cleric", "wizard", "ranger", "rogue", "witch", "adventurer", "warrior", "cleric"],
         ["ogre-forest", "goblin", "goblin-hexer", "goblin", "wolf-timber", "goblin-loot", "imp-forest", "fungus", "faerie"]),
        ("catacombs", ["warrior", "warrior", "cleric", "wizard", "rogue", "ranger", "witch", "cleric", "wizard"],
         ["lich", "skeleton", "bone-warden", "bonecrafter", "skeleton", "grave-chanter", "acolyte-dark", "skeleton", "bats-echo"]),
        ("road", ["warrior", "rogue", "ranger", "cleric", "wizard", "witch"],
         ["brigand", "ruffian-enforcer", "poacher", "poacher-shieldman", "bonesetter", "ruffian"]),
        ("snowfield", ["warrior", "ranger", "cleric", "wizard", "witch"],
         ["wolf-snow", "ice-warrior", "ice-guardian", "snow-floof", "wolf-snow"]),
    ):
        cells = [(r, c) for r in (1, 2, 3) for c in (1, 2, 3)]
        company = {cells[i]: f(u) for i, u in enumerate(comp)}
        enemies = {cells[i]: f(u) for i, u in enumerate(ene)}
        forms.append(scenes.formation_scene(backgrounds.BACKGROUNDS[bg][0](), company, enemies).to_image())
    items.append(("S3 formations at 1x (3x3 company vs enemy, enemies mirrored)", forms, 1))
    classes = ["warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer"]
    for fam, ids in (
        ("S3 units: classes and fallbacks, 4 idle frames each (2x)", classes + ["unknown-humanoid", "unknown-beast"]),
        ("S3 units: beasts, spiders, fey (2x)", [u.id for u in roster.UNITS if u.size in ("S", "M") and u.family in ("rodent", "canine", "spider", "fey", "cave", "snow") and u.id not in ("spider-queen",)]),
        ("S3 units: people (2x)", [u.id for u in roster.UNITS if u.family in ("undead", "cultist", "bandit", "shadow guild", "goblin", "town guard", "training", "ice", "plant") and u.size == "M"]),
        ("S3 units: large and boss (1x)", [u.id for u in roster.UNITS if u.size in ("L", "XL")]),
    ):
        imgs = [_load(out, f"battle/units/{i}/idle.png") for i in ids]
        items.append((fam, imgs, 1 if "(1x)" in fam else 2))
    for grp in ("status", "roles", "morale", "conditions"):
        items.append((f"S3 {grp} icons (3x)", [_load(out, r) for r in files if r.startswith(f"ui/{grp}/")], 3))
    items.append(("S3 battle markers (3x)", [_load(out, r) for r in files if r.startswith("battle/ui/")], 3))
    items.append(("S2 terrain, 3 variants each (2x); anim overlays and fog (2x)",
                  [_load(out, r) for r in files if r.startswith("map/terrain/") and "-anim" not in r][:19]
                  , 2))
    items.append(("S2 anim overlays, fog, unknown, night mask (2x)",
                  [_load(out, r) for r in files if r.startswith("map/terrain/") and ("-anim" in r or r.split("/")[-1] in ("fog.png", "unknown.png", "night-mask.png"))], 2))
    items.append(("S2 landmarks (3x)", [_load(out, r) for r in files if r.startswith("map/landmarks/")], 3))

    placed, tiles = [], []
    y = margin
    for label, imgs, z in items:
        x = margin
        row_h = 0
        placed.append((label, x, y))
        y += 14
        for im in imgs:
            zz = z
            while im.width * zz > W - 2 * margin:
                zz -= 1
            big = im.resize((im.width * zz, im.height * zz), Image.NEAREST)
            if x + big.width + margin > W:
                x = margin
                y += row_h + gap
                row_h = 0
            tiles.append((big, x, y))
            x += big.width + gap
            row_h = max(row_h, big.height)
        y += row_h + gap + 6
    sheet = Image.new("RGBA", (W, y + margin), BG + (255,))
    d = ImageDraw.Draw(sheet)
    for label, x, yy in placed:
        d.text((x, yy), label, fill=INK)
    for big, x, yy in tiles:
        sheet.alpha_composite(big, (x, yy))
    os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
    sheet.convert("RGB").save(dest, optimize=True)


def contact_sheet_s5(out, dest):
    """Sheet for S5: every promoted class (battle idle frames at 2x, map views at 3x), the summons and the goblin shaman."""
    import promoted
    import roster
    W, margin, gap = 1500, 12, 10
    tiles, placed = [], []
    y = margin

    def put_row(label, imgs):
        nonlocal y
        x = margin
        row_h = 0
        placed.append((label, x, y))
        y += 14
        for im, z in imgs:
            big = im.resize((im.width * z, im.height * z), Image.NEAREST)
            if x + big.width + margin > W:
                x = margin
                y += row_h + gap
                row_h = 0
            tiles.append((big, x, y))
            x += big.width + gap
            row_h = max(row_h, big.height)
        y += row_h + gap + 6

    for lineage in ("warrior", "cleric", "rogue", "ranger", "wizard", "witch", "halberdier", "samurai", "shaman"):
        ids = [c for c in promoted.CLASS_IDS if promoted.LINEAGE[c] == lineage]
        put_row(f"{lineage} line: base, then " + ", ".join(ids) + " (battle idle 2x)",
                [(_load(out, f"battle/units/{i}/idle.png").crop((0, 0, 128, 64)), 2) for i in [lineage] + ids])
        put_row(f"{lineage} line: map units, down/up/side rows, idle (3x)",
                [(_load(out, f"map/units/{i}/idle.png").crop((0, 0, 32, 96)), 3) for i in [lineage] + ids])
    put_row("summons (1x), goblin shaman and its hexer (2x)",
            [(_load(out, f"battle/units/{i}/idle.png"), z) for i, z in
             (("angel", 1), ("demon", 1), ("goblin-hexer", 2), ("goblin-shaman", 2))])
    sheet = Image.new("RGBA", (W, y + margin), BG + (255,))
    d = ImageDraw.Draw(sheet)
    for label, x, yy in placed:
        d.text((x, yy), label, fill=INK)
    for big, x, yy in tiles:
        sheet.alpha_composite(big, (x, yy))
    os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
    sheet.convert("RGB").save(dest, optimize=True)
