# Phase 38b: Promotions and talents

Sources: [branching design](../designs/2026-10-01-branching-class-progression-design.md),
[level impact](../designs/2026-10-05-level-impact-class-power-design.md) §1e and the
approved [faith routes design](../designs/2026-10-05-faith-routes-design.md).

## What shipped

- **Class model** (`internal/classes`): the archetype stays the lineage and the
  class is a separate layer, derived and never saved as a stat. A class has a
  parent, a tier, an alignment gate and ranks; `EffectsFor(class, level,
  talents)` applies ranks along the parent path (last wins), then adds the
  talents. Effects are read by key (`classes.Damage`, `Ward`, `HealPct`...), so
  later phases add routes as data. Neutral classes 39a-39h plug in as new
  lineages with `register`, `defineTalent` and `offer`; nothing else changes.
- **Promotion** at level 10 (advanced, three routes a lineage: good +30,
  neutral, evil -30) and level 30 (elite, only where a route has one: Hierarch,
  Demonologist, Elder Druid, Paladin, Dread Knight; the other lineages' elites
  are named but planned and keep the advanced ranks). `class`, `class paths`,
  `class promote [member] [class] [confirm]`. A promotion waits on alignment
  and a class is never lowered. Player state lives in the archetype registry;
  a companion's on its company record (`Class`, `Talents`), both saved at once
  and rolled back if the save fails.
- **Talents** at 5, 15, 25, 35, 45 and 55: five a lineage, most taken twice.
  `talent`, `talent list`, `talent pick [member] [talent] [confirm]`. A talent
  counts only for the first N picks, N the slots the level has earned.
- **Class effects in play**: damage, block, parry, armor, Attack and Evasion,
  spell cost and healing size, patching cost (`Healer.CostPct`), chant breaking,
  hex reach and landing, plus runtime battle state (`Character.RT`, shared by
  the by-value copies `calculateCombat` takes, cleared at fight end).
- **Spells** (`_datafiles/world/default/spells`, taught by rank): Ward,
  Arcane Ward, Greater Heal, Rejuvenation, Barkskin, Grove, Siphon, Entangle,
  Bless (clerics, level 8), Call the Host, Bind the Fiend; strategy uses
  (`big-heal`, `rejuv`, `grove`, `ward`, `bark`, `bless`, `siphon`, `summon`)
  in `modules/strategy` config.
- **Warrior routes**: Lay on Hands (takes the turn, no mana, uses per rest),
  Divine Shield, auras of Resolve and Dread, Smite, Blood Oath, Intimidation,
  Terror (staggering crits), Mercenary Tackle ranks.
- **Summons** (`internal/summons`, `internal/company/summons.go`,
  `internal/hooks/combat_summon.go`): the Angel and Demon are charmed mobs made
  at the caster's level, registered as members no formation holds, so combat
  finds them like companions. Called first in a battle by the `summon` use (3
  chant rounds, a tenth of max mana, once a battle); Mercy, Guard (`guardianFor`
  treats an Angel with guards as a guardian of the most hurt ally), Wings,
  Cleansing light, Dread, Hellfire, Soul feast and the broken binding are a
  per-round pass. They are dismissed at fight end, swept if orphaned, and a
  fallen one just fades (`mobcommands.Suicide`).
- **Level-up reports** (player and companion) name the next class milestone.
- **Help**: `help classes`, `promotion`, `talents`, `cleric-routes`,
  `warrior-routes`, `summoning` (with aliases for every class name), and the
  archetype, progression, strategy, combat, warrior and witch pages; tutorial
  hints in the Departure and combat lessons.

## Decisions and deviations

- Lay on Hands uses are in memory, reset by a rest (`RestClass` from
  `restoreVitals`) and refilled by a restart: a deliberate leniency, not saved.
- Rank text that could not be built faithfully was reworded: Terror (its critical
  hits that land stagger the target), Soul feast (a foe falls while the
  Demon fights, not "when the Demon kills"), Hellfire (every foe, a summon holds
  no formation cell).
- The faith routes design lists a help page per route; they are one page per
  lineage (`cleric-routes`, `warrior-routes`) plus `summoning`, with aliases so
  `help paladin` and the rest work.
- Entangle is a hex (Hobbled, 2 rounds, Vitality resist) so it uses the 38a
  machinery; row spells (Grove, Barkskin on a row) are `helpsingle` scripts that
  reach the row through `RowAllies`.
- A summoned creature has no company key a formation holds (`summon:<kind>`),
  and unplaced members reach every foe.

## Balance

No simulation was run in this phase (timeboxed out): every number above is the
design's first pass. The review thread or the next balance phase should run the
35b harness with a Paladin, Blackguard, Hierarch and Demonologist in place of a
warrior or cleric, and a Priest ward against the old Minor Heal line.

## Tests

`internal/classes`, `internal/combat/class_effects_test.go`,
`modules/archetype/class_test.go`, `modules/company/wiring_class_test.go`,
`wiring_class_spells_test.go` (every spell and the summons through the real
strategy pass and combat round), `internal/summons`, strategy decisions, level-up
line, and `TestClassHelpTopics`.
