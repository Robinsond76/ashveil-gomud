"""Master palette for the Ashveil sprite sets (docs/designs/2026-10-05-sprite-specification.md).

Sixty-seven colors: one outline color, twenty-one materials with three shade
steps each (dark, mid, light), and a three-step ember ramp.  Every sprite
uses only these colors.  The "skin" and "hair" ramps are the two a player's
chosen skin tone and hair colour repaint on their own figures (Phase 72a);
nothing else uses "hair", so the client can tell hair from any other brown.  Colors are addressed by name: "outline" or
"<material>.<d|m|l>".  Light comes from the top-left.
"""

# Muted, naturalistic colors: earth, iron, wool, leather, bone.
MATERIALS = {
    # Figures
    "skin": ("8a5a40", "b57d5a", "d4a37c"),
    "hair": ("2f2118", "4d3727", "715640"),
    "iron": ("2f3338", "55595f", "82868b"),
    "steel": ("5f666e", "959ba2", "cdd2d4"),
    "leather": ("3a2519", "5e3f27", "86603a"),
    "wood": ("54442f", "826b48", "b09870"),
    "wool": ("7f7660", "aea48a", "d6cdb2"),
    "brass": ("6b5420", "a1822f", "cfb255"),
    "bone": ("8a8473", "bdb8a6", "e8e4d4"),
    # Class accents (subdued)
    "oxblood": ("3c1518", "6a2224", "93383a"),
    "plum": ("2e1d30", "503450", "7a5578"),
    "charcoal": ("1d1d22", "34343c", "4f4f59"),
    "moss": ("263322", "44593a", "6d8553"),
    "slate": ("263040", "46566b", "72849a"),
    "heather": ("3b2f45", "655577", "8f80a0"),
    "ashmoss": ("3a4234", "5f6e55", "8a9a7c"),
    # Land
    "grass": ("34401c", "586a28", "889a3a"),
    "forest": ("16281e", "274330", "3d6044"),
    "water": ("1f3446", "35597a", "5e8aa6"),
    "ochre": ("5e4519", "946b2a", "c29a4a"),
    "stone": ("3b3935", "68645c", "98938a"),
    # Fire and ember glow
    "ember": ("6e1e10", "c4561c", "f0b04a"),
}
OUTLINE = "15120f"
TONES = ("d", "m", "l")


def _build():
    pal = {"outline": OUTLINE}
    for name, ramp in MATERIALS.items():
        for tone, hexv in zip(TONES, ramp):
            pal[f"{name}.{tone}"] = hexv
    return {k: tuple(int(v[i:i + 2], 16) for i in (0, 2, 4)) for k, v in pal.items()}


PAL = _build()
assert len(PAL) <= 67, len(PAL)
RGB_TO_NAME = {rgb: name for name, rgb in PAL.items()}
assert len(RGB_TO_NAME) == len(PAL), "palette colors must be unique"
