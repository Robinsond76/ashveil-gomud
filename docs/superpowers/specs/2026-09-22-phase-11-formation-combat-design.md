# Phase 11 Formation Combat — Overview

**Status:** Decomposed and confirmed via collaborative design session,
2026-09-22. This overview replaces the original single-doc draft (its
"enemies have no formation of their own this phase" assumption did not
survive the design discussion below and is no longer accurate). Each
sub-phase has its own detailed spec; this doc records the shared decisions
and the sequencing rationale.

## Why this became four sub-phases instead of one

The original Phase 11 scope (handoff §36: front-row interception, melee
reach, polearm reach, ranged rear-line benefit, adjacency queries) turned
out to have a hidden prerequisite: none of it can mean anything against a
mob, because today's combat has no concept of an enemy "formation," no
concept of a company auto-joining a fight as a unit, and single-target
`Aggro` with no positional awareness at all (see the Prior-art check below,
shared by all four sub-phases). Building formation tactics directly on that
foundation would produce inert code with almost nothing to act on. The
design session below arrived at a natural three-part foundation-then-payoff
split, plus a fourth bucket of already-identified but explicitly deferred
extras:

- **11a — Enemy Parties.** Give mobs a first-class party/formation concept,
  mirroring the player's company.
- **11b — Unit-vs-Unit Engagement.** Make a fight actually a company vs. a
  party, with real target assignment (a minimal targeting-preference rule)
  instead of one ad hoc `Aggro`.
- **11c — Formation Tactics.** The original scope — interception, reach,
  adjacency — now with something real underneath it.
- **11d — deferred, not scheduled.** Guard reactions, weapon-flavored crit
  effects, wounds, and full AI targeting personality (handoff items 6-9,
  already flagged as low-risk layer-on additions in
  `docs/superpowers/specs/2026-09-22-combat-design-reference-external.md`).
  Sequenced after 11a-11c ship so each phase stays independently reviewable.

## Prior-art check (shared across 11a/11b/11c)

A structured research pass over `internal/combat`, `internal/characters`,
`internal/company`, and `internal/parties` found:

- **Combat is entirely round-driven**, one exchange per combatant per
  round, via `events.NewRound` → `internal/hooks/NewRound_DoCombat.go`.
  There is no per-character timer and no shared "encounter" object.
- **Targeting is single-target.** `characters.Aggro` holds exactly one
  `UserId` or `MobInstanceId`. `combat.AttackPlayerVsMob`/etc.
  (`internal/combat/combat.go:28-96`) each take exactly one attacker and
  one defender. Multiple simultaneous fights in a room happen only because
  the round loop iterates every combatant independently.
- **No positional/reach concept exists in combat at all.** Same-room
  membership is the only spatial gate. `items.ItemSubType` only selects
  flavor-text and affects dual-wield counting, never range or legality.
- **`internal/company` has zero combat code.** The Phase 3 formation grid
  is a roster/UI concept only; combat has never read it.
- **Company members do not auto-join a fight.** A charmed companion only
  retaliates reactively when *it* is attacked
  (`NewRound_DoCombat.go:380-393`, `:873-885`). There is no "leader
  engages, so do I" rule and no shared targeting between leader and
  companions.
- **`internal/parties` is a separate, native, *multiplayer* grouping
  system** (real players only, in-memory/not persisted), already carrying
  a `front`/`middle`/`back` `Position` and a weighted `ChanceToBeTargetted`
  (front=2, middle=1, back=0). It has been left untouched since Phase 3 and
  stays untouched through 11a-11d (see "Unit scope" below).
- **`Aggro` is explicitly not persisted** (`yaml:"-"`,
  `internal/characters/character.go:78`) — combat state does not survive
  restart today. Parties/Engagements introduced by 11a/11b therefore don't
  need new durability/copyover machinery either.
- **PvP formation interaction is an explicit handoff "future feature"**
  (§6.8), not current scope. 11a-11d are PvE-only.
- **`internal/combat.MobRank`** (`internal/combat/mob_rank.go`) already
  computes per-mob-spec `EHP` (effective HP, a tankiness proxy) and `DPS`
  at the mob's effective level — the natural signal for 11a's automatic
  front/back assignment heuristic, no new stat needed.
- **`mobs.Mob.Groups []string`** is the existing hostility-grouping field
  (already used by `mobs.MakeHostile`) — the natural signal for which mobs
  in a room belong to the same enemy party.

## Decisions locked in by the design session

- **Interception:** strict redirect — while any front-row party member is
  alive, an attack aimed at a back-row member lands on the front-row member
  instead. No chance roll, no "guard" stance (that's a natural 11d-or-later
  extension, matching the handoff's "shields protecting adjacent units"
  future item).
- **Reach model (column-occupancy, not row-distance):** within a
  targetable column, a plain melee attack can only land on that column's
  current frontmost occupant — whoever is blocking the lane. **Reach**
  (granted by a polearm-class weapon, or as an innate characteristic on
  some mobs/monsters — not exclusively tied to wielding a polearm item)
  extends that to the frontmost occupant *or* the one directly behind them
  (front + middle of that column). Ranged ignores column depth entirely.
  Nobody can reach the back slot of a column while anyone occupies a slot
  in front of it. This is naturally self-healing: if the blocking front
  occupant dies, whoever was behind them becomes reachable next round with
  no player action needed — so a stored target is *never* cleared on a
  reach failure, only skipped that round.
- **Lateral range:** independent of weapon/reach, a combatant can only
  target enemy columns within its own column ±1 (a middle-column combatant
  can reach all three enemy columns; an edge-column combatant reaches two).
- **Targeting preference:** each combatant needs a minimal rule (favor
  weakest / strongest / random) to auto-pick among *legal* (in lateral
  range, in reach) candidates — the full personality richness from the
  external reference doc (wounded-seeking, threat-awareness) stays in 11d,
  but *some* selection rule has to exist in 11b, because without it
  auto-engagement has no way to assign initial targets at all.
- **Enemy formation:** mob parties get the same `company.Formation` 3×3
  grid type, auto-assigned at party-formation time via a role heuristic
  built on `MobRank`'s `EHP`/`DPS` (high EHP → front, high DPS/low EHP →
  back) — not hand-authored per spawn.
- **Formation is locked once combat starts.** No mid-fight rearrangement
  in 11a-11d; only enemy attrition changes what's reachable.
- **Unit scope:** a "Unit" is exactly one player's own company (leader +
  companions) for 11a-11d. `internal/parties` (multiplayer grouping) stays
  dormant and untouched — not reconciled, not removed. **Future note**
  (out of scope, captured so it doesn't need re-deriving later): the
  eventual multiplayer model is not "merge two companies' grids" — a
  joining player's character becomes a formation *member* within the
  host's existing company grid, so there's still exactly one authoritative
  3×3 per fight even once multiplayer joins.
- **v2 (continuous Readiness/Wind-up/Cast/Recovery timing model):**
  explicitly parked. Everything in 11a-11d is a "who is grouped with whom,
  who is a legal target" problem, solvable entirely within the existing
  one-attack-per-round-per-combatant model — it does not require knowing
  *when* in a round each combatant acts. Revisit only after 11a-11c ship,
  per the handoff's own deferral of this item.

## Sub-phase specs

- `docs/superpowers/specs/2026-09-22-phase-11a-enemy-parties-design.md`
- `docs/superpowers/specs/2026-09-22-phase-11b-unit-engagement-design.md`
- `docs/superpowers/specs/2026-09-22-phase-11c-formation-tactics-design.md`

Each is independently plannable and implementable once the shared
decisions above are settled, in that order (11a before 11b before 11c —
each depends on the previous one existing).
