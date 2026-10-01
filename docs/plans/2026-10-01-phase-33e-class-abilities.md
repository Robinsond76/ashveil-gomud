# Phase 33e: Automatic Class Abilities — Plan

Design: [33e design](../designs/2026-10-01-phase-33e-automatic-class-abilities-design.md),
"Final implementation decisions". Worktree `.worktrees/phase-33e-class-abilities`,
branch `phase-33e-class-abilities`.

## Task 1: domain (internal/strategy)

- `abilities.go`: `Ability` ids and specs (Tackle, Opening Strike, Aimed
  Shot), archetype/skill unlocks, cooldowns, `Known` (player by skill
  levels, companion by archetype), `AbilitySituation` and `DecideAbility`.
- `Strategy.NoAbilities` and `Strategy.Reserve` (durable, yaml omitempty);
  `IsZero`, `ParseReserve`.
- `Decide`: `Situation.MaxMana`/`Reserve` gate attack spells; `Ally.Pending`
  excludes covered allies from healing.
- Unit tests for each rule and gate.

## Task 2: execution (internal/hooks, internal/combat)

- `combat_abilities.go`: `abilityPass` after `strategyPass` (same battle
  and side lists), cooldown map by actor/ability in combat rounds, tackle
  resolution (roll seam, knocked-down status, chant/wind-up breaks, turn
  spent), opening/aimed strike as the round's backstab-type swing, cleanup
  after the blows, stream `ability` events, 29c lines.
- Players and companions skip their swing on a tackle turn.
- `strategyPass`: pending heals (in-progress chants and this round's
  choices) and the mana reserve.
- `combat.go`: the backstab crit is a local flag (no crit reported without
  a landed blow); drop the `*[BACKSTAB]*` prefix (29c voice).

## Task 3: controls and presentation

- `modules/strategy`: `strategy [who] abilities on|off`, `reserve [n]`,
  carried through role/rule changes, cleaned on load, shown in list and
  describe.
- GMCP `Company` strategy: `abilities`, `abilities_off`, `reserve`; web
  Combat setup shows them (js-lint).

## Task 4: help and tutorial

- `abilities.template` (new), keywords and aliases; links from `combat`,
  `company`, `strategy`, `tactics`; updates to warrior/rogue/ranger,
  brawling, skulduggery, track pages; practice-fight hint in
  `modules/tutorial/stages.go`; render test; `TestTutorialHelpPointersExist`.

## Task 5: integration tests (real entry points)

Through the brawl harness (`modules/company`): a warrior companion tackles
its foe in a real round (knocked down, no swing, cooldown), a tackle breaks
an enemy chant, a rogue's opening strike crits a knocked-down foe, a ranger
aims a shot, `abilities off` stops them, a shooting warrior never tackles,
two healers do not double-heal one ally, the reserve holds an attack spell,
the player uses their own ability, another player's companion/ally and a
non-company charm get none; strategy command round-trip and persistence
(reload with the new fields; old files load as defaults).

## Task 6: migration and recovery

New fields default to on/0 when absent; unknown values are dropped on load
with a warning; cooldowns are runtime only (documented). No data migration.

## Task 7: review and integration

Independent reviewer subagent on the full diff; fix findings with tests;
`make generate`, `make validate`, `go test -race ./...`, `make js-lint`;
Project Status; merge to master, push, remove the worktree.
