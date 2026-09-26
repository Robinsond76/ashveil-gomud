# Potential Phase 28e: Spell Narration and Casting in Combat

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction for the text (2026-09-26). The mechanics
items below are candidates, each needing its own decision.

## Prior-art check (2026-09-26)

- **Where spell text lives:** in each spell's script in
  `_datafiles/world/default/spells/*.js`, through `SendUserMessage` and
  `SendRoomMessage`.
- **The spells in the mock:**
  - wizard, conjuration: `mm` (Magic Missile, 1d6+2, 1 wait round) and
    `sparks` (Shower of Sparks, 1d3+1 to each enemy);
  - cleric, restoration: `heal` (Minor Heal, 2d3, 2 wait rounds) and
    `healall`.
- **Today's wording:** "begins to chant softly", "continues chanting...",
  "lets loose a magical projectile at X hurting them!".
- **Archetypes (17a):** the wizard has the conjuration and illusion
  schools and is granted `floatinglight`; the cleric has restoration and
  `protection`.
- **Companions don't cast.** A cleric companion (Brother Oswin) only
  swings the cudgel.
  - Mob templates can list `combatcommands` (`internal/mobs/mobs.go`),
    which `DoCombat` picks from during a fight (around line 773). That is a
    possible seam for companion casting.
- **Spells have no critical hits.**

## Scope — text (approved)

- **Rewrite the spells' messages** in the 28b voice:
  - chanting lines stay each round while a spell builds, voiced per school;
  - the release line is physical;
  - no `!`.
- **Damage:** ` (N damage)` at the end, as for weapons. A multi-target spell
  prints one line for the cast, then an indented line per target, each with
  its damage.
- **Heals:** end ` (N healed)`. A multi-target heal prints one line, then
  one indented line listing everyone healed:
  `Garrick Vane (3 healed) · Ysolde (2 healed) · you (4 healed)`.
- **Pronouns and names:** use 28c's pronouns and ordinals.

## Scope — mechanics (candidates, decide separately)

1. **Companion casting.** A cleric companion heals, and a wizard
   companion casts, during a fight.
   - It needs a simple rule, for example: heal a member below half health,
     otherwise fight.
   - It also needs mana, cooldown, and wait-round handling for mobs.
2. **Enemy casters.** New content, such as a bandit hedge-witch, using the
   same mob combat commands. Her spell reaches "you" lines.
3. **Spell critical hits.** This is a rules change, not text, and the
   owner hasn't decided it. If adopted, it would end
   `(critical hit, N damage)` and trigger 28d's pain reactions.
4. **Fight-end line** (from 28b) also closes fights won by spells.

## Reference text (approved mock, 2026-09-26)

```
> cast mm captain
You begin to murmur the words of binding. Frost creeps across your knuckles.
The words of binding coil tighter. Pale light gathers between your fingers, humming.
You loose a bolt of pale fire. It punches into the bandit captain's chest, and the reek of scorched cloth fills the air. (6 damage)

The hedge-witch begins to whisper in a tongue that makes your teeth ache.
The hedge-witch opens her hands and a knot of black fire lashes into you. It burns cold, and the skin of your arm blisters and splits. (5 damage)

> cast sparks
You begin to murmur again. Sparks spit and crawl across your palms.
The sparks in your palms knot together into a roaring, spitting ball.
You fling your hands wide, and a storm of sparks tears across the bandits.
    The sparks sear the bandit captain's face. (3 damage)
    The sparks catch in the first cutthroat's hair, and he beats at it, screaming. (2 damage)
    The sparks rake the second cutthroat's arm. (2 damage)
    The sparks scorch the hedge-witch's hands and snuff out her whisper. (4 damage)

Brother Oswin lowers his cudgel and begins to pray, his voice low and steady.
Brother Oswin's prayer swells. A dull gold light gathers in his hands.
Brother Oswin lays his glowing hands on Garrick Vane's hip, and the torn flesh knits closed beneath them. (4 healed)

Brother Oswin raises his arms, and a warm light washes over the company.
    Garrick Vane (3 healed) · Ysolde (2 healed) · you (4 healed)
```

## Acceptance criteria (text)

- Every shipped spell's messages match the voice. No `!`; damage and heal
  suffixes on every effect line.
- A wiring test casts `mm` and `heal` through the real command and round,
  and asserts on the rendered lines.
