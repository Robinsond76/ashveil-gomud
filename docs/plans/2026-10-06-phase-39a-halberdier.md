# Phase 39a: The Halberdier

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §2.
The first neutral lineage: a polearm fighter that wins by crowding.

## What shipped

- **Archetype and recruit**: `halberdier` (HP 7 and 0.9 a level, Attack 1,
  Evasion 0.9, medium armor, no shield, glaives and spears, strength 4 /
  vitality 3 / speed 2 / perception 1, Polearm skill, kit), a recruitable
  Halberdier (mob 130) in the Dunmar templates, level-3 `ShippedFor`.
- **Abilities** (`internal/strategy`, `internal/hooks/combat_halberd.go`):
  - **Sweep** (cooldown 3): the whole turn. Strikes the foe and its nearest
    row neighbour at 90% damage, the whole row from level 8. Each foe is a
    real blow (`extraBlow`: hit, defense, armor, messages, events).
  - **Brace** (level 3, cooldown 1): holds the turn; the first foe to strike
    into its place is answered at 125%; lost if no foe strikes.
  - **Hook** (level 6): a glaive hit on a leaping foe knocks it down, 20%,
    40% from level 20.
- **Crowded rule**: new target rule `crowded` (aliases row, rows, sweep),
  the Halberdier's default: the weakest foe in the row with the most foes.
- **Routes** (`internal/classes/routes_halberdier.go`), all open at any
  alignment: Sweeper (Full sweep, Quick sweep, Practiced hands, Heavy edge),
  Vanguard (Hold the line, Parrying drill, Hardened guard, Unmoved), Valkyrie
  (Charged Sweep with lightning, Rising storm, Gathered storm, Deep reserves).
  The elites (Reaper, Linebreaker, Tempest Lancer) are `Planned`: 39i.
- **Talents**: the warrior's four plus Sweep Drill.
- **Help and tutorial**: `help halberdier`, `help halberdier-routes`, and
  every page the class touched (classes, promotion, archetype, abilities,
  strategy, talents, health, growth, armor, shields, evasion, progression,
  combat); a tutorial pointer in the creation lesson.

## Decisions

- Neutral from the start: all three routes use `GateAny`.
- Sweep strikes at 90% (80% in the first draft; tuned from the balance run).
- A Sweep that cannot find two foes in the row is not used; Brace needs a
  foe striking at the Halberdier's place, and otherwise the Halberdier swings.
- Hook affects only mobs flagged `Leap`; bosses are not exempt yet.
- The sprite is deferred to 40s5; the battle screen uses a grey and copper
  hue pair meanwhile.

## Balance (ASHVEIL_BALANCE=1, `balance_halberdier_test.go`)

A warrior or Halberdier in the third slot against groups of the company's
level. First draft (Sweep 80%, 15 fights a cell): the Halberdier wins as often
as the warrior against 2 and 3 foes, in fewer rounds and with less damage
taken at levels 5 and 10, and loses to five-foe groups (40% against 73% at
level 5, 66% against 100% at level 10). Tuned to 90%: five foes at level 5 and
10 win 70% and 93% (30 fights) against the warrior's 73% and 100% (15 fights);
level 20 five-foe groups win 66% against the warrior's 60%. Within noise of
the design's five-point target; timeboxed, settled.

## Tests

`internal/classes`, `internal/strategy`, `internal/combat` (BlowPct),
`modules/company/wiring_halberdier_test.go` (the real round: sweep, wide
sweep, fallen foes, brace, hook, Valkyrie lightning, Vanguard knockdown,
abilities off, a player Halberdier), help render tests, and the archetype and
company suites updated for the seventh archetype.
