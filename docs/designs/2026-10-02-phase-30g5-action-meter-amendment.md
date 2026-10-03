# Phase 30g5: Action meter — implementation amendment

Prepared 2026-10-03 against `b62f1987` (30f and 30g4 merged). The owner
requested implementation after reviewing the proposals, and explicitly
removed pets from this slice. This supplements the agreed
[30g design](2026-09-30-phase-30g-tempo-defense-design.md), slice 30g5.
Delivery plan: [30g5 plan](../plans/2026-10-02-phase-30g5-action-meter.md).

## Scope and sequencing

Implement 30g5 on the shipped 30g4 progression. No HP, stat-step, XP,
or final tuning changes belong in this slice; 30g6 remains the tuning pass.

## Shipped starting defaults and action rules

- Tempo reads `Stats.Speed.ValueAdj`: the character's own effective Speed,
  including training and equipment, without a target-relative or level term.
  Shipped `TempoSpeedRef: 10`, `TempoSpeedSpan: 40`, `TempoMin: 0.6`,
  `TempoMax: 1.5`, `MaxTurnsPerRound: 2`. Reference Speed is one turn per
  round without burden; Speed 30 reaches the upper cap. These are starting
  values for 30g6, not a claim that combat is balanced.
- Compute `(1 + (Speed - ref) / span + 0.1 * attacksMod) * (1 - 0.35 * burden)`,
  then clamp. This makes the existing `attacks` modifier a tempo bonus for
  every weapon type; personal burden affects the complete rate.
- Opening meter is `100 - 100 * tempo`, including negative values for fast
  actors, so everyone earns exactly one opening turn. Each subsequent round
  adds `100 * tempo`; spend complete hundreds up to the configured cap,
  retaining at most 99 points. Use a fractional accumulator with a tested
  boundary tolerance so decimal tempo does not lose turns to rounding.
- A turn contains both eligible hands and each weapon's own attacks dice.
  Remove the extra target-relative unarmed/claw attack formula. Keep dual
  wield, reach, guards, and damage rules inside the existing attack path.
- A tackle spends one available turn, successful or not. An opening strike
  or aimed shot replaces one weapon turn; any second turn is ordinary.
  A fighter with zero turns uses no automatic physical ability.
- Existing chants advance once per combat round, including a zero-turn
  round. A chanting actor takes no weapon turn, even if its chant ends or
  is broken during that round; discard available whole turns. Automatic
  spell selection retains its once-per-round cadence.
- Stagger, knockdown, stun, hesitation, withdrawal, and no-combat restrictions
  consume the round's available turns without banking. Recheck live statuses
  before a second turn; report a lost-action line once per actor per
  round. Newly applied statuses retain 30a's first-tick rule: action loss
  begins on the next combat round, not on a half-spent round. Weapon wait
  counters decrement once per combat round.
- Physical wind-ups retain a visible combat-round preparation window:
  preparation or release occupies the entire round, advances once per round,
  and cannot prepare and release in the same round. Wind-up cooldowns count
  combat rounds. Amend `help interrupts` to state that distinction explicitly.
- Guards and shield counters remain reactions with existing per-round limits.
- No pets exist in the game. Pet mechanics are outside this slice, as the
  owner directed; no new pet assistance proposal or pet acceptance work.


## Code integration and ownership

`internal/hooks/NewRound_DoCombat.go` owns the round. Today it runs upkeep,
statuses, automatic spells, and abilities once, then player and mob handlers
once. Add a meter allocation pass after engagement upkeep/status processing
and before ability selection. Keep round upkeep, chants, waiting counters,
retreat handling, and AI combat-command selection outside repeated weapon
turns. Do not repeat the whole `DoCombat` callback to obtain extra attacks.

Extract or narrowly loop physical turns through the current four attack
directions, rechecking health, aggro, location, visibility, battle eligibility,
reach/interception, and statuses for each turn. Apply each turn's buffs,
interrupts, narration, damage tracking, and affected-actor bookkeeping before
another turn. Stop if the attacker or target falls; no attack on a dead target.
Restore readied ability strikes after their first attempted weapon turn.

Meter state belongs to the game loop and is runtime-only. Key actors by
player ID or mob instance ID, not target ID. Associate membership with live
fight identity; a shared enemy in several players' battles fills only once
per combat round and resets only when its last engagement ends. PvP and
mob-vs-mob combat also need meters despite having no player battle record.
End-of-fight cleanup must reset immediately, including a battle replaced
between adjacent combat callbacks; pruning only on an idle round is not enough.
Tests reset the meter explicitly. Never save it on Character or company state.

`internal/combat/calculations.go`, `weapon_rank.go`, and `simulate.go` currently
depend on the old attack count. Change per-turn resolution to one complete
weapon turn and make per-round estimates/simulation use the new tempo model.
Avoid counting tempo twice. `internal/configs/config.gameplay.go` and
`_datafiles/config.yaml` hold defaults/validation. Keep obsolete extra-attack
config keys readable for legacy overrides, but document that they no longer
control attack frequency. No new scheduler, global-time changes, persistence
format, goroutine, or separate combat engine is needed.

## Player help and acceptance

Ship `help tempo` (aliases `actions`, `turns`), linked from combat,
encumbrance, burden, and speed. Correct speed's unarmed/claw-only claim;
update abilities, statuses, and interrupts where action wording
changes. Add the Practice Yard tutorial hint. Use config-rendered numbers.

Acceptance includes opening/slow/fast meter sequences, clamps, burden and
attacks modifiers, two-turn and carry caps, no formation/target dependency,
all four real attack directions, dead-target and attacker interruption,
status losses, one chant/wait/wind-up step per round, tackle/strike costs,
reaction limits, shared-enemy single fill,
fight-end/new-fight reset, and unchanged world time. Run the 30g1 harness
and record turns and output volume alongside fight-duration changes.

Before integration: independent full-phase review, focused help/tutorial
checks, `make generate`, `make validate`, applicable lint, and
`go test -race ./...`; record findings and actual checks in Project Status.

Delivery and review: [30g5 verification](../plans/2026-10-03-phase-30g5-verification.md).
