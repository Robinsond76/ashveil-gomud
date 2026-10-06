# Phase 39g: the Alchemist

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §8.
A flask-thrower with no alignment gate and no mana: it heals, cures, burns and
hardens from a satchel of flasks that a camp rest or `brew` refills.

## What shipped

- **Lineage and data**: the `alchemist` archetype (HP 3 and 0.65 a level,
  Attack 0.75, Evasion 0.85, light armor, no shield, daggers, staffs and
  slings, smarts 4 / perception 2 / vitality 2 / speed 2, the `cast` skill, no
  mana pool), school `alchemy`, reagents (item 95, sold in Dunmar, supply
  only, three in the starter kit), an Alchemist recruit (mob 190), creation
  text, code-drawn map and battle art, a battle-screen hue.
- **Flask spells** (`internal/spells`): `flask: 1` on a spell costs one flask
  instead of mana, with `waitrounds: 0` (no chant, so nothing to interrupt).
  Healing Draught (1), Antidote (3), Fire Flask (6), Bracing Tonic (8).
  Manual `cast` and the combat round both spend the flask and refuse an empty
  satchel.
- **Satchel** (`internal/flasks`): capacity 6 + level/3 + Deep satchel ranks;
  stored as `Character.FlasksSpent` (0 is full) so a new Alchemist starts full
  and restart or copyover keeps what was thrown. Companions persist it in
  `company.MemberState.FlasksSpent`.
- **Brewing**: `brew` (any time out of battle) and automatically at the end of
  a camp rest (`brewRestFlasks`), one reagent from the leader's pack a flask,
  the leader's own satchel first.
- **Strategy** (`internal/strategy`): an Alchemist is a Healer by default. New
  uses `cure` and `flame` and an `Afflicted` ally flag. Cure runs while no one
  is below half health; a Fire Flask is thrown only while more than 2 flasks
  remain (`FlaskKeep`), so heals are never left without one; a tonic goes to
  one ally a round before the blows.
- **Routes** (all open at any alignment; ranks 10/15/20/25): **Apothecary**
  (Potent draughts +30%, Deep satchel +4, Splash draught 30%, Clean draught),
  **Bombardier** (Hotter flasks 70% to 100% of Sparks, Wide throw +1 foe, Pitch
  and tar burning, Deep satchel), **Mutagenist** (Mutagen +10 armor for a tenth
  of the ally's health, Stable mutagen halves the cost, Strong mutagen +15
  armor and longer tonic, Pure mutagen free). Panacean, Grenadier and
  Transmuter are registered as planned elites (39i). Talents: Deep Satchel,
  Steady Brew, Hot Brew, Toughness, Footwork.
- **UI**: `status` shows "Flasks: N of M"; the level-up report names the
  satchel's size and each new rank; `strategy` lists flask spells "(a flask)"
  and hides locked ones; GMCP `Company.Vitals` carries `flasks`/`flasks_max`
  and `Char.Capabilities` the flask spells ("Satchel empty", "N flasks left");
  the battle screen, Combat tab and company card show the satchel. Help:
  `alchemist`, `alchemist-routes`, `brew`, plus the archetype, classes,
  promotion, talents, strategy, combat, camp, health, growth, armor, shields,
  evasion, progression and specialists pages, and a tutorial hint.

## Decisions

- **Flask spells are ordinary spells with a `flask` cost**, so the existing
  spell scripts, targeting and manual `cast` carry them; the satchel is a
  single count, not one per kind.
- **Brewing is automatic at camp-rest end plus a `brew` command**, mirroring
  the Doll Master's mending, so it costs no world time.
- **Mutagen is armor for an HP cost with no damage bonus**; Smoke Flask and
  Elixir from the design are deferred to the elites (39i).
- **Rank mapping differs from the design**: its rank-10 signatures moved so
  each route's first rank is felt at promotion.
- IDs: item 95, mob 190, school `alchemy`.
- The Alchemist's skill is `cast` with no mana pool and no camp utility.
- Doll Master art is missing on master (`TestEveryLineageHasBaseArt` fails for
  it alone); another thread draws it.

## Tests

`internal/flasks`, `internal/classes` (lineage, ranks, effects, talents,
level-up text), `internal/strategy` (cure, flame, keep, role), `modules/company`
`wiring_alchemist_test.go` (the real round: draught with no mana, empty
satchel, antidote before danger, tonic, Apothecary, Bombardier, Mutagenist,
restart round trip, `brew`, manual `cast`), `modules/camping` (rest brewing),
`modules/market` (reagents supply-only), `modules/gmcp` (vitals), help render
test, and the archetype, company and tutorial suites updated for the twelfth
archetype.

## Balance

100 fights a cell, five-foe group of the company's level, one member swapped
for the class (`balance_alchemist_test.go`, `ASHVEIL_BALANCE=1`). Win rate at
levels 5 / 10 / 20 / 25: Cleric 63 / 77 / 34 / 24, Ranger 79 / 87 / 70 / 71,
Alchemist 62 / 72 / 29 / 21. Within about 5 points of the Cleric, the other
healer. Routes (base / Apothecary / Bombardier / Mutagenist): level 15
43 / 43 / 39 / 44, level 25 24 / 16 / 29 / 20. All inside the noise except
Apothecary at 25, which trails the base class by 8; no tuning was done (its
Clean draught and satchel help sustain long fights the sim ends early). Elite
ranks are balanced in 39i.

## Follow-ups

Elites (Panacean raises a fallen ally, Grenadier, Transmuter), the design's
Smoke Flask and Elixir, a poison-flask enemy toolkit, and the art pass for
the route classes.
