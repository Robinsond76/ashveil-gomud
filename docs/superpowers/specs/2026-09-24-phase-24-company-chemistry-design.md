# Phase 24: Company Chemistry

Implements the [company chemistry spec](2026-09-23-company-chemistry-design.md),
third on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

**Owner amendment (2026-09-24): company-wide chemistry.** The first
implementation kept a bond per pair of members, as the parent spec
described. The owner asked for chemistry across the whole company instead
("the longer a band stays together, they all get bonuses") and chose the
**dilute** variant: each member's service counts, a band's tier comes from
the average service of the members together, and a new recruit pulls that
average down but fights with the band. The pair model was never merged; it
was replaced on the same branch. This document describes the band model;
the parent spec carries the same amendment.

## Prior-art check

- **Parent spec (as amended):** durable per-member service under the
  leader's company record, with a last charged round. Tiers at 1, 3, and 7
  game-day equivalents (900, 2700, 6300 rounds), config. +2/+4/+6 points of
  hit chance, never stacking. No effect on experience, damage, targeting,
  reach, or interception. `company chemistry`, `status bonuses`, and a
  browser Company panel show it.
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

The owner chose the company-wide, dilute model; the remaining details are
this design's recommendations, applied as in Phases 19b–23b.

1. **Storage.** `Record.Service []Service` (`service,omitempty`), one per
   member: `{member, rounds, last_round}`. `Registry.Put` prunes entries
   of members no longer in the record (dismissal and desertion end that
   companion's service in the same save; the leader's own service stays
   through `dismiss all`) and normalizes the rest (one per member, rounds
   at least 0). `Get` and `Clone` copy the slice.
2. **Service.** The leader is signed in. A member is *present* when alive
   (`Health >= 1`) and there: the leader is an online user; a companion is
   a tracked, live mob still attached to the leader. A present member
   serves a round when at least one other present member is in its room.
   Companions together count while the leader is signed in but elsewhere.
   Offline time, absence, death, and being alone pause service. Travel
   and camping keep the company in one room, so they count.
3. **Charging.** Each `NewRound` adds 1 to each serving member whose
   `last_round` is below the event's round, and sets it, so a round is
   counted once and a round replayed after a restart isn't recounted. A
   counter more than 900 rounds behind resumes charging (a reset counter
   file). Credit is one per event, never the gap, so a clock jump awards
   nothing. The listener only reads the round number.
4. **The band and dilution.** A member's band is every present member in
   its room, itself included; fewer than two is no band. The band's
   average service (a member with none counts as 0, integer division)
   sets its tier, and **everyone in the band** gets that tier's bonus, a
   new recruit included. Four veterans at exactly Sworn (6300) and a fresh
   recruit average 5040, Trusted, until the recruit has served a while.
   Only one tier applies; nothing stacks.
5. **Durability.** Each entry also keeps `Saved`, in memory only: its
   rounds as of the last successful company save (or load). **Tiers are
   worked out from `Saved` only**, so no tier is used or shown before it is
   on disk, whoever walks in or out of the room. Service rides on every
   company save (autosave, logout, copyover, shutdown, commands, drift
   ticks); each successful save marks everything saved. When a round's
   service would lift a band's tier over its saved tier, the company is
   saved at once, and the leader is told only after the save succeeds. A
   failed save leaves the tier where it was and is retried the next round.
   A crash can lose the rounds since the last save.
6. **Tiers.** Familiar (900 rounds, +2), Trusted (2700, +4), Sworn (6300,
   +6). Knobs `ChemistryFamiliarRounds`, `ChemistryTrustedRounds`,
   `ChemistrySwornRounds` (1..1000000, strictly increasing) and
   `ChemistryFamiliarBonus`, `ChemistryTrustedBonus`,
   `ChemistrySwornBonus` (0..10, non-decreasing). A set out of order falls
   back to all defaults, warned about once. The knobs are parsed on load
   and once a round, never per strike.
7. **Combat.** An attacker who is a company member gets its band's bonus,
   added to the hit modifier (with darkness and dual-wield penalties)
   before `Hits` clamps, in all four `Attack*` functions: a player
   attacker is its own company's leader; a mob attacker is looked up with
   `LeaderAndKeyForInstance`, so a companion's auto-assigned target gets it
   too. Targeting, reach, interception, dodge, damage, crits, experience,
   and the pet's own strikes are untouched; the combat simulator passes no
   bonus. `ChemistryHitBonus` reads the registry in place and skips the
   presence check unless some member's saved service reaches a tier (an
   average never exceeds its largest member).
8. **Identifying chemistry in combat.** When a strike hits only because of
   the bonus, the attacker sees one line that round: *"Fighting beside a
   companion you know well, you find an opening."*
9. **Surfaces.** `company chemistry` shows the band with the leader (size,
   tier, bonus, progress as a percentage), any band of companions apart
   from the leader, and each member's saved service in game days.
   `status bonuses` has a **Company Chemistry** box ("Trusted band, 3
   together: +4% to hit"). Tier-ups are announced ("Your band grows
   closer: Trusted (+4% to hit fighting together)."; companions apart are
   named). The browser Company panel is deferred to the
   information-surfaces phase; `company.ChemistryStanding` is its read
   model.
10. **No new locks.** The module and combat both run on the game loop.

## Scope

**In scope:**

- `internal/company/chemistry.go` (pure): `Service`, `ChemistryRules`
  (`DefaultChemistryRules`, `Valid`, `Tier`, `Bonus`, `Progress`),
  `TierName`, `Record.ChargeService`, `FindService`, `MarkServiceSaved`,
  `BandAverage`, `BandTier`.
- `internal/company`: `Record.Service`, copy in `Get`, prune in `Put`;
  optional provider `ChemistryProvider` with `ChemistryBonusForUser`,
  `ChemistryBonusForInstance`, `ChemistryStanding`.
- `modules/company/chemistry.go`: the `chemistryWorld` seam, cached config,
  per-round accrual from `onNewRound`, tier-up save and announcement, the
  provider, `company chemistry`; `save` and `load` mark service saved.
- `internal/combat`: `hitRoll` and a `chemistryBonus` parameter on
  `calculateCombat`; the four `Attack*` functions look it up.
- `internal/usercommands`: the `status bonuses` box.
- Config knobs and comments, module `AGENTS.md`.

**Deferred:** the browser Company panel (information-surfaces phase);
chemistry effects beyond hit chance; service decay.

## Constraints

- Never advances or writes the world clock or round count.
- Survives restart and copyover: service is in the company file; loaded
  service counts as saved; a replayed round is not recounted.
- No locks added; everything runs on the game loop.
- Legality and automatic targeting are decided before the bonus.

## Acceptance criteria

- Pure: tier boundaries; progress; charging once per round, never on a
  replayed round, resuming after a large counter reset; dilution (four
  veterans and a recruit average down a tier); only saved rounds give a
  tier; a lone member is no band; prune and normalize; YAML round trip
  without `Saved`.
- Module: service accrues only for members present with another
  (separated, dead, detached, offline leader pause it); companions together
  while the leader is elsewhere; dismissal ends exactly that companion's
  service; a respawned companion resumes; a tier-up saves and announces;
  a failed save keeps the tier until saved and announces nothing; a
  recruit dilutes the band and still gets its bonus; a lone member gets
  nothing; service survives a reload and counts as saved; a failed drift
  save in the same round keeps the charge; views and standing.
- Combat: `hitRoll`; a bonus raises hits through all four `Attack*`
  functions; the chemistry line shows at most once a round.
- Wiring: through `plugins.Load`, `usercommands.TryCommand`, and real
  `NewRound` events, a leader and a live companion serve, reach a tier,
  `company chemistry` and `status bonuses` show it, service survives
  `plugins.Save`, logout, and reload, a real `AttackPlayerVsMob` sees the
  bonus, and a second recruit dilutes the band.
- `go test -race ./...`, `make generate`, `make validate` pass.
