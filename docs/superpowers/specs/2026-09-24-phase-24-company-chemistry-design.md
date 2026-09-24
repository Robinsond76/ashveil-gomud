# Phase 24: Company Chemistry

Implements the [company chemistry spec](2026-09-23-company-chemistry-design.md),
third on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Prior-art check

- **Parent spec:** one bond per unordered pair of stable member keys, under
  the leader's company record, with accumulated eligible shared rounds and
  a last charged round. Tiers at 1, 3, and 7 game-day equivalents (900,
  2700, 6300 rounds), config. +2/+4/+6 points of hit chance from the
  member's **highest** bond with a living, present partner, never
  stacking. No effect on experience, damage, targeting, reach, or
  interception. `company chemistry`, `status bonuses`, and a browser
  Company panel show it.
- **Stable IDs** (`internal/company`): `LeaderMemberKey` is `leader`,
  companions are `companion:<id>`, and IDs are never reused
  (`NextCompanionID`, survival's reservation). `Registry.Put` already
  prunes formation cells for members no longer in the record.
- **Presence:** the module tracks each companion's live mob
  (`m.instances`); `Runtime.IsAttached` says whether it still serves the
  leader. A companion that dies is untracked (`onMobDeath`) and respawned
  with the same ID on the leader's next spawn. Players are alive at
  `Health >= 1` (`NewRound_DoCombat`, `NewRound_AutoHeal`).
- **Periodic work:** Phase 21a's `onNewRound` listener in
  `modules/company`. `events.NewRound` carries `RoundNumber`; the counter
  is persisted every 10 seconds (`world.go`) and in copyover state, so after
  a crash it can come back a few rounds behind.
- **Hit chance:** `internal/combat.Hits(atkSpd, defSpd, modifier)` adds the
  modifier to `hitChance` and clamps to `[ToHitMin, ToHitMax]`. Phase 14's
  darkness penalty and dual-wield's offhand penalty already ride on the
  modifier. All four `Attack*` functions call `calculateCombat`; the
  combat simulator calls `calculateCombat` directly.
- **Cross-package seam:** `company.FormationProvider` and its optional
  extensions (`ArchetypeProvider`, `AlignmentProvider`) let engine packages
  read company state without importing `modules/company`.
- **`status bonuses`** (`internal/usercommands/status.panels.go`) lists
  equipment, buff, and pet bonuses in boxes.
- **No browser Company panel exists yet.** The
  [player information surfaces spec](2026-09-23-player-information-surfaces-design.md)
  owns it.

## Decisions

The session instruction was "Continue next phase"; the parent spec's
direction was approved by the owner on 2026-09-23, and the details below are
this design's recommendations, applied without a separate confirmation
round, as Phases 19b–23b did. Each is a config knob or a small follow-up.

1. **Storage.** `Record.Bonds []Bond` (`bonds,omitempty`), each
   `{a, b, rounds, last_round}` with `a < b` (keys sorted). A bond is
   created the first round its pair is eligible, at 0 rounds, which is the
   same as starting it at recruitment. `Registry.Put` prunes bonds whose
   member is no longer in the record, so dismissal and desertion end
   exactly that companion's bonds, in the same save. `Registry.Get` and
   `Clone` copy the slice.
2. **An eligible round.** The leader is signed in. A member is *present*
   when alive (`Health >= 1`) and there: the leader is an online user; a
   companion is a tracked, live mob still attached to the leader. A pair
   is eligible when both are present and in the same room. Companion
   pairs count while the leader is signed in even when the leader is in
   another room (the spec's rule). Offline time, absence, death, and
   separation pause accrual. Travel and camping keep the company in one
   room, so they count.
3. **Charging.** Each `NewRound` adds 1 to every eligible bond whose
   `last_round` is below the event's round, and sets `last_round`. So a
   round is counted once however many times it's seen, and a round
   replayed after a restart isn't counted again. If the round counter
   goes back more than 900 rounds (a reset counter file), charging
   resumes instead of waiting for the counter to catch up. Credit is one
   per event, never the gap between rounds, so a clock jump awards
   nothing. The listener only reads the round number; it never writes it.
4. **Durability.** Bonds ride on every company save: autosave, logout
   (`PlayerDespawn`), copyover, shutdown, commands, drift ticks. When a
   bond crosses into a new tier the module saves at once. If that save
   fails, the bond is held one round short of the tier and the error is
   logged, so the tier is never shown or used before it's on disk; it is
   retried on the next eligible round. The leader is told of a new tier
   only after it's saved. A crash can lose the rounds since the last
   save, which never crosses a tier.
5. **Tiers.** Familiar (900 rounds, +2), Trusted (2700, +4), Sworn (6300,
   +6). Knobs `ChemistryFamiliarRounds`, `ChemistryTrustedRounds`,
   `ChemistrySwornRounds` (1..1000000, strictly increasing) and
   `ChemistryFamiliarBonus`, `ChemistryTrustedBonus`,
   `ChemistrySwornBonus` (0..10, non-decreasing). A set that breaks
   either order falls back to all defaults.
6. **Combat.** An attacker who is a company member gets the bonus of its
   highest tier among bonds with a partner present in its room right now.
   It's added to the hit modifier (with darkness and dual-wield
   penalties) before `Hits` clamps. All four `Attack*` functions apply it:
   a player attacker is its own company's leader; a mob attacker is
   looked up with `LeaderAndKeyForInstance`, so a companion's
   auto-assigned target gets it too. Targeting, reach, interception,
   dodge, damage, crits, experience, and the pet's own strikes are
   untouched. The combat simulator passes no bonus. Per-strike cost is a
   map lookup and at most four bonds; no copying.
7. **Identifying chemistry in combat.** When a strike hits only because of
   the bonus (its roll fell between the chance without it and with it),
   the attacker sees one line per round: *"Fighting beside a trusted
   companion, you find an opening."* That's the only combat text; there's
   no per-strike tag.
8. **Surfaces.** `company chemistry` lists each present-or-not member:
   tier name, bonus, strongest partner, and whether that partner is at
   their side right now, plus progress to the next tier as a percentage
   (no calendar date). `status bonuses` gains a **Company Chemistry**
   box separate from buffs. The browser Company panel is deferred to the
   information-surfaces phase; this phase adds the read model
   (`company.ChemistryStanding`) it will use.
9. **No new locks.** The module and combat both run on the game loop; the
   provider reads the registry map directly, read-only.

## Scope

**In scope:**

- `internal/company/chemistry.go` (pure): `Bond`, `ChemistryRules`
  (`DefaultChemistryRules`, `Valid`, `Tier`, `Bonus`, `Progress`),
  `TierName`, `BondPair`, `Record.ChargeBond`, `BestBond`.
- `internal/company`: `Record.Bonds`, copy in `Get`, prune in `Put`;
  optional provider `ChemistryProvider` with `ChemistryBonusForUser`,
  `ChemistryBonusForInstance`, `ChemistryStanding`.
- `modules/company/chemistry.go`: a `chemistryWorld` seam (leader and
  instance presence), config parsing, the per-round accrual called from
  `onNewRound`, tier-crossing save and announcement, the provider
  methods, `company chemistry`.
- `internal/combat`: `hitRoll` returning whether the bonus decided the
  hit; a `hitBonus` parameter on `calculateCombat`; the four `Attack*`
  functions look it up.
- `internal/usercommands`: the `status bonuses` box.
- Config knobs and comments, module `AGENTS.md`.

**Deferred:** the browser Company panel (information-surfaces phase);
chemistry effects beyond hit chance; bond decay.

## Constraints

- Never advances or writes the world clock or round count.
- Survives restart and copyover: bonds are in the company file; a
  replayed round is not recounted.
- No locks added; everything runs on the game loop.
- Legality and automatic targeting are decided before the bonus and are
  unchanged.

## Acceptance criteria

- Pure: tier boundaries; bonus of the highest tier only; pair key order;
  charging once per round, never on a replayed round, resuming after a
  large counter reset; rules validation.
- Module: the leader and two companions gain independent bonds; only
  eligible pairs advance (separated, dead, detached, offline leader);
  dismissal ends exactly that companion's bonds; a companion killed and
  respawned keeps and resumes its bonds; a tier crossing saves and
  announces; a failed crossing save holds the bond short and announces
  nothing; bonds survive a store reload; the leader alone gets no bonus;
  several veteran partners never exceed the top bonus; an absent partner
  gives nothing.
- Combat: `hitRoll` with and without a bonus; a bonus raises hits through
  real `AttackPlayerVsMob` and `AttackMobVsMob`; the chemistry line shows.
- Wiring: through `plugins.Load`, `usercommands.TryCommand`, and real
  `NewRound` events, a leader and a live companion accrue, cross a tier
  (config lowered), `company chemistry` and `status bonuses` show it, the
  bond survives `plugins.Save` and a reload, and the real combat entry
  points see the bonus through the registered provider.
- `go test -race ./...`, `make generate`, `make validate` pass.
