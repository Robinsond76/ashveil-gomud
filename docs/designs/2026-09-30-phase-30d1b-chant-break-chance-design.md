# Phase 30d1b: A Blow Breaks a Chant Only by Chance — Design

An amendment to [Phase 30d1](2026-09-30-phase-30d1-chant-interrupts-design.md)
(broken chants, shield counters, enemy casters). Written 2026-09-30.

## Owner decisions (2026-09-30)

The owner asked, after 30d1 merged, that a damaging blow break a chant by
chance instead of always, and left the chance to us. The rule below was
set by the lead and given to this branch as decided; it was not re-asked.

1. **Heavy force always breaks:** a critical hit, a blow whose statuses
   include staggered, knocked down, or stunned (30a's `status.Staggered`,
   `KnockedDown`, `Stunned`). (A shield bash was on this list; amended by
   [30d2](2026-09-30-phase-30d2-windups-design.md), owner 2026-09-30: a
   bash is a counter strike only and breaks nothing.)
2. **Any other hit that does 1+ damage breaks with chance**
   `40 + 2 × (damage × 100 / the chanter's max health)`, clamped to
   40–90%: a nick is about 40%, a blow of a quarter of the chanter's
   health 90%.
3. **Unchanged from 30d1:** a miss, a dodge, 0 damage, status ticks, and
   spells never break a chant; a broken company chant is lost with half
   its mana back, an enemy's restarts from the first word; the counter's
   rules.
4. **A held chant is told** in the 29c voice: the room `Brother Oswin
   flinches, but the chant holds. (Minor Heal, chant held)`; the chanter
   (a player) `You flinch, but your chant holds. (Minor Heal, chant
   held)`; and an `Interrupt` event with `Outcome: failed`.
5. **30d2 (recorded, not built):** a wind-up for a physical attack does
   not break on ordinary blows, only on heavier force: a crit, a stagger,
   a knockdown, or a stun (the shield bash struck by 30d2, as above).

## Prior-art check (against `master` at `7306212`)

- `interrupt.Breaks(hit, damage, chanting)` is the whole 30d1 rule;
  `afterBlow` (six blow sites) and `counterBlow` (the bash) call it, then
  `breakChant`.
- The counter's dice are injectable (`counterRoll`,
  `UseCounterRollForTest`), and `newBrawl` pins them; the break roll
  follows the same shape.
- A blow's statuses are `AttackResult.BuffTarget` (30a: a crit's effect by
  weapon subtype, or a weapon's own crit buffs).
- The summary already counts company `Interrupt` events whose outcome
  isn't `succeeded` as "failed", shown only when non-zero; an enemy's
  failed interrupt (a company chant that held) is not counted.

## Scope

- **`internal/interrupt` (pure):** `Breaks` becomes `CanBreak` (the
  eligibility: a damaging hit on a chanter); new `BreakChance(damage,
  maxHP int, heavy bool) int` (100 when heavy, else the formula clamped
  40–90; 0 for no damage; a max health below 1 counts as 1) and
  `RollBreak(chance int, roll func(int) int) bool` (certain at 100 or
  more without rolling, never at 0 or less, else `roll(100) < chance`).
  Constants `BreakChanceMin = 40`, `BreakChanceMax = 90`.
- **`internal/hooks/combat_interrupt.go`:** `breakRoll` (defaults to
  `util.Rand`; `UseBreakRollForTest`); `heavyBlow(r)` (crit, or a
  staggered/knocked-down/stunned buff in `BuffTarget`); `afterBlow`
  rolls, then `breakChant` or the new `holdChant` (lines, event). The
  shield bash passed heavy, so it always broke (removed by 30d2: a bash
  breaks nothing).
- **Summary:** "failed N" now means company blows a foe's chant withstood;
  the comment is updated, the line itself is unchanged.
- **Tests' dice:** `newBrawl` pins the break roll to 0 (always breaks, so
  every existing brawl keeps 30d1's outcome and seeded rolls); the
  interrupt wiring tests set it explicitly.

## Durable model and invariants

- Nothing new is saved; `breakRoll` is a package function value. Aggro
  (and so every chant) stays unsaved; restart/copyover drops chants.
- No world-clock change; the roll happens inside the blow that causes it.
- Game loop only; no new locks.

## Constraints and deferrals

- 30d2 (wind-ups) is not built; decision 5 is recorded for it in the 30d1
  design and the status log.
- The held line is told for enemies and company alike; no line for a
  blow that drew no blood (as before).
- The shield bash's "always breaks" was kept in code for 30d2's wind-ups,
  but could never happen (a character who swings isn't chanting, and a
  foe winding up swings no blow to counter), so the help didn't list it
  (review finding). 30d2 removed it: a bash is a counter strike only.

## Acceptance criteria

- **Unit (`internal/interrupt`):** `BreakChance` (heavy is 100; a nick is
  40; a quarter of max health is 90; mid-range scales, e.g. 10% → 60;
  0 damage is 0; a max health of 0 does not divide by zero); `RollBreak`
  (certain, never, and the boundary); `CanBreak` (30d1's table).
- **Wiring through `DoCombat` (brawl world):**
  - a held chant: roll forced high, a light blow on Oswin's heal — the
    room line, an `Interrupt` `failed`, the chant continues;
  - a player's held chant gets `You flinch, but your chant holds.`;
  - a crit breaks even with the roll forced high (a stagger, knockdown,
    or stun comes only from a crit today, so the status cases are the
    `heavyBlow` unit test's; amended after review);
  - a crit the armor took entirely is no heavy blow (`CritLanded`;
    review finding);
  - the battle summary shows "failed 1" after a company blow a foe's
    chant withstood;
  - 30d1's interrupt tests pass with the roll forced to break.
- **Player help:** `help interrupts` gives the chance, the heavy blows,
  and the held line; `combat`, `cast`, `strategy`, `tactics`,
  `formation`, and `guardian` no longer say any blow that draws blood
  breaks a chant; the Practice Yard hint likewise;
  `internal/usercommands/help_interrupts_test.go` checks the new text;
  `TestTutorialHelpPointersExist` passes.
- `modules/company` looped (`-count=5`) with no new flake;
  `go test -race ./...`, `make generate`, `make validate` pass; the
  independent review is recorded in `docs/PROJECT_STATUS.md`.
