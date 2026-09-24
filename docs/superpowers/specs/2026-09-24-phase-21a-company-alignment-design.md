# Phase 21a: Company Alignment

## Prior-art check

- **Roadmap** (`2026-09-23-environment-skills-economy-roadmap.md`), decision
  2 and Phase 21: "companion alignment, 1–100 display, Ogre Battle-style
  drift toward company average, loyalty/desertion, recruit gates,
  settlement standing". Alignment affects (1) loyalty/morale and desertion
  when a member is far from the company average, (2) recruitment gates (a
  good paladin won't join an evil company), (4) settlement standing
  (prices, inn access, black-market access). Rejected: archetype gates,
  alignment-driven encounters.
- **Engine alignment** (`internal/characters/alignment.go`): every
  character has `Alignment int8` in −100..100 with named bands (unholy,
  evil, corrupt, misguided, neutral, lawful, virtuous, good, holy). Players
  shift on kills (`combat.AlignmentChange`) and decay toward neutral every
  `AlignmentDecayRounds`. Mobs take their template's `alignment`, or their
  race's `defaultalignment` when it is 0 (`mobs.NewMobById`). Aggressive
  mobs use the alignment gap (`AlignmentAggroThreshold`).
- **Companions** (`internal/company`, `modules/company`): a durable record
  per leader of `Companion{ID, MobTemplateID, Archetype}`; the live mob is
  respawned from the template on every login/copyover, so any per-companion
  state must live in the company record. `company summon` is today's only
  way to recruit (a recruiter flow is a separate, unscheduled spec that
  expects to show alignment "once Phase 21 exists").
- **Periodic work** hangs off `events.NewRound` through a module-owned,
  persisted countdown, never the round number (`modules/market` news,
  Phase 20 review finding: an in-memory countdown never fires if restarts
  are more frequent than the interval).

## Split

Phase 21 is split like 19/19b. **21a (this doc)** is everything inside the
company: companion alignment, the 1–100 display, drift, loyalty and
desertion, and the recruit gate. **21b** is settlement standing (market
prices, inn access, black market), which crosses into `modules/market` and
`modules/camping` and gets its own design doc.

## Decisions

The session instruction was "Implement next phase"; the defaults below are
this design's recommendations, applied without a separate confirmation
round, the same way Phases 19b and 20 applied theirs. Each is a config knob
or a small follow-up if the owner wants it changed.

1. **Scale.** Alignment is stored on the engine's −100..100 scale, so a
   live companion's `Character.Alignment` matches its record and existing
   engine code (aggro, kill shifts, scripts) keeps working. Players see
   **1–100** (1 most evil, 100 most good, 50 neutral):
   `display = (alignment + 100) * 99 / 200 + 1`, with the engine's band
   name beside it. Config knobs are in engine points (twice the display
   points), documented as such.
2. **Companion alignment is durable.** Each companion record gains a
   `disposition` (alignment and loyalty). A new recruit starts at its
   template's alignment (the race default when the template's is 0) and
   `StartLoyalty`. A companion saved before this phase is seeded the same
   way on load, at full loyalty (it has already been serving), and the
   upgraded store is saved at once.
3. **Drift.** Every `DriftEveryRounds` rounds, for each company whose
   leader is online, each companion's alignment moves `DriftStep` toward
   the average of the **rest** of the company (the leader and the other
   companions), never past it. All targets are taken from the values
   before the tick, so order doesn't matter. **The leader doesn't drift**:
   a player's alignment stays the product of their own deeds (kills,
   scripts, the engine's decay), and the module is not a second writer to
   the user record. The leader still pulls on every companion.
4. **Loyalty** (0..100, shown as the companion's morale). On each drift
   tick, a companion whose gap to the rest of the company (before the
   drift) is more than `LoyaltyToleranceGap` loses `LoyaltyLoss`; otherwise
   it gains `LoyaltyGain`, up to 100. Crossing `LoyaltyWarnBelow` downward
   warns the leader. At 0 the companion **deserts**: it leaves exactly as
   `company dismiss` would (survival state removed, formation cleared,
   live mob detached), and the leader is told. Desertion waits while the
   leader is in combat and happens on the first tick after.
5. **Recruit gate.** `company summon` refuses a candidate whose alignment
   is more than `RecruitMaxGap` from the company average (the leader and
   every current companion). The refusal names both values.
6. **Inspecting a recruit.** `company inspect <mob-id-or-name>` shows an
   allowed candidate's alignment, the company average, and whether they
   would join. `company status` shows each companion's alignment and
   loyalty and the company average; `company alignment` shows the leader
   and every companion with their gap to the rest of the company.
7. **Defaults** (engine points): `DriftEveryRounds` 75 (5 minutes at
   4-second rounds; valid 1..100000), `DriftStep` 2 (1..50),
   `LoyaltyToleranceGap` 60 (0..200), `LoyaltyLoss` 5 (0..100),
   `LoyaltyGain` 2 (0..100), `StartLoyalty` 70 (1..100),
   `LoyaltyWarnBelow` 25 (0..100), `RecruitMaxGap` 80 (0..200). Out of range
   or unparsable values fall back to the default.

With these defaults a companion 60 points (30 displayed) from the rest of
its company is content; one at 90 loses 5 loyalty every 5 minutes while
drifting 2 points closer, so from `StartLoyalty` 70 it deserts after about
70 minutes unless the gap closes or the leader changes course.

## Scope

**In scope:**

- `internal/company/alignment.go` (pure): `DisplayAlignment`,
  `AlignmentBand`, `AverageAlignment`, `DriftToward`, `AlignmentRules`
  (with `DefaultAlignmentRules`), `TickAlignment(leader, members, rules)`
  returning each member's new alignment and loyalty plus who warned and who
  deserts, and `CanRecruit(candidate, average, rules)`.
- `internal/company`: `Companion.Disposition *Disposition`
  (`disposition,omitempty`: `alignment`, `loyalty`), never mutated in
  place; `Registry.SetDisposition`; `Registry.DriftIn` (persisted countdown,
  `drift_in,omitempty`).
- `modules/company`:
  - an `alignmentWorld` seam (template alignment, online leader alignment
    and combat state, live-instance alignment, messaging) with a native
    implementation, so unit tests need no world;
  - seeding on summon and on load; applying the stored alignment to the
    live mob on every spawn (summon and restore) and after each drift;
  - an `events.NewRound` listener counting `DriftIn` down, running the tick
    for online leaders, saving once per tick, and deserting at 0 loyalty;
  - the recruit gate in `summon`; `company inspect`, `company alignment`,
    and the richer `company status`;
  - config knobs in the module's overlay.

**Explicitly deferred:** settlement standing (21b); companions' own kill
shifts (companions change only by drift in 21a); leader drift; morale
effects in combat; alignment in `look`/`status` for players (the
information-surfaces spec owns those); a recruiter NPC flow.

## Durable model

`Companion` gains `Disposition *Disposition` (`disposition,omitempty`)
with `Alignment int` and `Loyalty int`. A nil disposition is a companion
from before this phase and is seeded on load. The pointer is replaced,
never written through, so `Registry.Get`'s copies stay independent. Values
are clamped on seed and on every write (alignment −100..100, loyalty
0..100).

`Registry` gains `DriftIn int` (`drift_in,omitempty`): rounds until the
next drift tick. 0 or out of range means a full interval. It is saved with
every company save (commands, drift ticks, and the autosave/copyover/
shutdown `plugins.Save`), so a restart resumes from the last saved value.
The legacy wire decoder reads both fields.

## Constraints

- Never reads, drives, or advances the world clock or round counter; the
  countdown only counts `NewRound` events.
- The company module runs on the game loop (commands and event listeners),
  with no locks of its own; this phase adds none.
- Desertion reuses the dismiss path's ordering and rollback: a failed save
  restores the record and survival state and the companion stays.
- A failed drift save restores the pre-tick registry, so memory and disk
  agree; the tick is retried at the next interval.

## Acceptance criteria

- Pure tests: display mapping endpoints and neutral; band names; average
  rounding; drift never overshoots; simultaneous targets exclude the
  member itself; loyalty loss/gain/cap/warn/desert thresholds; recruit gap
  boundary.
- Module tests: summon seeds disposition from the template and refuses a
  far candidate without writing; load seeds a legacy companion and saves;
  the live mob gets the stored alignment on summon, restore, and drift;
  drift after exactly `DriftEveryRounds` rounds, only for online leaders,
  persisted; `DriftIn` resumes after reload; loyalty warning and
  desertion (survival removal, formation cleared, instance detached,
  leader told); desertion postponed in combat; a failed drift save rolls
  back; `inspect`/`alignment`/`status` output; config parsing and bounds.
- Wiring test: through `plugins.Load` and `usercommands.TryCommand`,
  `company summon`/`inspect`/`alignment`/`status` with real mob specs; a
  real `NewRound` through `events.ProcessEvents` drifts a real live
  companion's `Character.Alignment` and writes the real store.
- `go test -race ./...`, `make generate`, `make validate` pass.
