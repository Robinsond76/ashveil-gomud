# Phase 30g3: Personal Load and Agility — Plan

Design: [phase-30g design](../designs/2026-09-30-phase-30g-tempo-defense-design.md),
slice 30g3 (owner decisions 2 and 3). Branch: `claude/phase-30g3-load`.

## Rules (from the design)

- **Personal load:** a character's worn and carried grams
  (`Character.PersonalGrams`), the same for players, companions' live
  mobs, and enemies. Company cargo, packs' carry bonus, and mounts are not
  in it and add nothing to capacity.
- **Agility capacity:** `AgilityBaseKg + AgilityStrengthKg × Strength`
  (adjusted Strength), new `GamePlay.Combat` keys, 15 kg and 0.5 kg.
- **Burden** `b = clamp((load / capacity − AgilityFreeLoad) / (1 −
  AgilityFreeLoad), 0, 1)`, `AgilityFreeLoad` 0.35.
- **Dodge** becomes `dodgeChance × (1 − 0.6 × b)`, rounded. Parry and
  block are untouched. The parry-or-dodge choice compares parry with the
  burdened dodge, so a burdened swordsman parries more often than he
  dodges (decision 15 still rolls the higher, once).
- **Words:** unburdened (b = 0), lightly burdened (b < 1/3), burdened
  (b < 2/3), heavily burdened. Never a ratio or kilograms beside them.
- Nothing saved; computed from the live items when needed.

## Decisions made while building

- **The 0.6 is a constant** (`burdenDodgeLoss`), not a key: the design
  names three new keys; 30g6 can promote it if tuning needs it.
- **`expectedDPS`** (weapon rankings) applies the defender's burden too,
  so the estimate matches the strike loop (its dummies carry nothing, so
  rankings don't move).
- **Where the words show:** the `status` Vitals panel (`Burden:`), the
  web Character Overview (from `Char.Inventory.Backpack.Summary.burden`,
  sent whenever items, equipment, or stats change), `look` at any
  character (a `Burden:` line beside the archetype line), and
  `scout`/`look` of an enemy group (members carrying weight are named
  with their word; the unburdened are left out to keep the grid narrow).

## Tasks

1. **Load and burden** (`internal/characters`, `internal/configs`):
   `PersonalGrams`, `AgilityCapacityGrams`, `Burden`, `BurdenWord`; the
   three keys with validation, `_datafiles/config.yaml`, and the admin
   config wizard and descriptions. The peep panel uses `PersonalGrams`.
   Unit tests: worn + carried (not cargo or packs' bonus), the free band,
   the clamp, Strength raising capacity, the word bands, validation.
2. **Dodge** (`internal/combat`): `activeDefense` and `expectedDPS` apply
   `burdenedDodge`. Tests through `calculateCombat`: a heavily burdened
   defender's certain dodge falls to 40%, an unburdened one keeps it,
   parry and block unchanged by burden, burden moving the parry-or-dodge
   choice. Through `DoCombat` (`modules/company` brawl): a burdened
   bandit dodges less over many strikes, and a horse or cargo leaves the
   leader's burden unchanged. Every hit-forcing helper still zeroes dodge,
   so they stay deterministic (`-count=3`).
3. **Surfaces:** `status`, `look`, `scout`, GMCP backpack summary, the
   web Overview. Tests: the status row, look of a burdened mob, scout's
   burden line, the GMCP field.
4. **Help and tutorial:** new `help burden` (aliases load, agility,
   weight, burdened, encumbered), indexed in `keywords.yaml` under
   combat and linked from `help combat`; `help defense`, `help
   perception`, `help status`, `help cargo`/encumbrance, `help scout`,
   and `help strength` updated where burden appears; a Combat-lesson hint
   in `modules/tutorial/stages.go`. Tests: `TestBurdenHelp` through
   `help`, `TestTutorialHelpPointersExist`.
5. **Measure, review, verify:** the 30g1 table re-run against 30g2's
   (no-focus median 9 / 38 / 71); an independent review; `make generate`,
   `make validate`, `make js-lint`, `go test -race ./...`;
   `docs/PROJECT_STATUS.md`.
