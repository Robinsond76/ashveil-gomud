# Phase 81: tuning the classes earlier phases left (2026-10-07)

Earlier phases recorded several classes as "rejected or recorded, not tuned"
because each needed its own sim: the Bearward, Druid, Nightblade, Sorcerer,
the wizard advanced tier and the 38e Stone Golem, plus a Witch re-measure at
L10. This phase measures each against its tuned peers, changes what reads
weak and records the rest. Mirror sims are class-balance checks only; the
at-level encounter targets of PR #179 are untouched (no encounter, mob or
zone change).

## Method

- Existing opt-in harnesses (`ASHVEIL_BALANCE=1`): `TestPhase38bClassRoutes`
  (Druid, in its summon cells), `TestPhase38c2EliteRoutes` (Nightblade),
  `TestPhase39eBeastTamer` (Bearward), `TestPhase38eCreatures` (Golem),
  `TestPhase38aWitchAgainstWizard` (Witch). 100 fights a cell (150 to 200 for
  the noisiest), noise about +/-9 points.
- New `TestPhase81ClassTuning` (`balance_phase81_test.go`): swaps one member
  (Oswin the cleric, Garrick the wizard) for each cleric or wizard route, five
  foes of the company's level. `ASHVEIL_BALANCE_LINE=1` stands the company in
  the shipped line (warriors in front, casters behind). The harness default
  leaves the company unplaced, so foes reach the casters at will; that made
  every wizard cell read 6-30% whatever the class did (a +30% spell damage
  trial moved nothing). With the line placed the wizard classes separate,
  so the wizard numbers below use it.
- Signature abilities were already fired through real rounds by the existing
  wiring tests; no signature changed shape, only numbers.

## Baselines and results

Win% (HP lost where it matters). "Before" is master; "after" is this phase.

| Class | Before | After | Verdict |
| --- | --- | --- | --- |
| Bearward (Beast Tamer, L15, unplaced mirror) | base 80, Bearward 79 | unchanged | Measured level with the base Beast Tamer (the 9-point gap 39e recorded is gone). No change. |
| Druid (Oswin, L35 summon cells) | base 26, Priest 52, **Druid 29** | **Druid 51**, Priest 48 | Retuned (below). |
| Elder Druid (L35 / L45) | 37 | **60 / 56** | Leads the Druid by 9 / 14. The Hierarch (88 / 98) and Demonologist (75 / 75) still lead: they summon, the Elder Druid heals. |
| Nightblade (L50, 4 foes two levels up, 200 fights) | Assassin 45, Nightblade 40 | Assassin 33, **Nightblade 39**, HP lost 83 vs 78 | Lifted; the sim is noise-limited (below). |
| Sorcerer (wizard line, placed) L15/25/40/50 | L25 base 49, Sorcerer 42; L40 base 44, Sorcerer 28 | base 52/50/43/47, **Sorcerer 65/50/44/38** | No longer under the base wizard to L40. |
| High Sorcerer (L40 / L50) | 52 | **64 / 67** | Leads Sorcerer by 20 / 29, level with Archmage (63 / 70) and Necromancer (57 / 56). |
| Arcanist | 61/46 (L25/L40) | 58/54 | Small lift only; the placed line already separated it. |
| Theurgist, Warlock | 63/58, 52/54 | 66/56, 60/49 | Unchanged (noise only). |
| Stone Golem (L5 / L10 / L20, unplaced) | warrior 84/88/75, Golem 72/90/71 | warrior 81/90/77, **Golem 81/91/72** | Level with the warrior at L5 and L10; L20 is 5 under with more health lost (65% vs 55%, accepted: the golem soaks for the row). |
| Witch at L10 (re-measure) | wizard 49, witch 48 | same | Witch and wizard stay level at L10 (L5 41 vs 23, L20 22 vs 25 as before); warrior 90. No change. |

## What changed

- **Druid** (the ranks were weak, not the class idea). Barkskin was +10
  armor, which is 5% less damage; it is +25 (about 12%). Rejuvenation heals
  170% of a Minor Heal over 3 rounds (was 130%); Lasting growth 220% over 4
  (was 160%); Thornhide 4 damage (was 2). The Elder Druid's Grove builds on
  Rejuvenation's percent, so it grows with it (no change to its own ranks).
- **Sorcerer** (a 2-round chant that breaks and costs double was worse than
  just casting Magic Missile). Gathered power +60% Lance damage (was 25),
  Steady chant 50% fewer breaks (was 25). **High Sorcerer:** High Lance +75%
  (was 40), Unbroken chant 75% (was 50), Searing lance +90% (was 45). The
  Lance's cost and chant are unchanged.
- **Arcanist:** Arcane focus 15% and Sharper focus 25% more spell damage
  (were 10% and 15%).
- **Stone Golem:** Stone body slows it 10% (was 15%); its fist lands 3 harder
  at Granite (was 2) and 5 at Bedrock (was 4).
- **Nightblade:** Death Mark makes the marked foe take 60% more damage (was
  40), Envenom poisons a third of the time (was a quarter), Hunting the mark
  +25% critical chance (was +15).
- Help pages updated with the numbers: `cleric-routes`, `wizard-routes`,
  `high-sorcerer`, `stone-golem`, `nightblade`. Rank tests updated.

## Decisions and what was not tuned

- Tuned by value only, in rank effects; no new effects, spells or signatures.
- **Rejected:** Warlock and Theurgist buffs (tried; they were already at or
  above the base wizard and moved only inside noise).
- **Nightblade is still level with the Assassin in the sim.** The rogue is
  one front-line member of five and the mark lives for one foe at a time, so
  the sim's noise (Assassin read 45, then 33) hides a few points. The signature
  numbers went up about a third; no further tuning without a sim that makes
  the mark matter (4-5 foes and a company built around it).
- The wizard-lineage companies remain below martial ones in the unplaced
  mirror (6-30%); that is a harness property (casters take the aim), not a
  class defect, and it was left alone as the at-level tuning relies on it
  being so.
- Robinson ordered the combat overhaul to speed-ordered, one-at-a-time turns
  (phase 82, building now). Tuning here stopped at "no longer weak" and did not
  chase fine margins; every number in this plan is for the model on master
  today and must be re-measured after the overhaul.

## Verification

- `make generate`, `make validate`, `go test -race -timeout 30m ./...`,
  `make js-lint`, `make js-test`.

## Paused (2026-10-07 14:03)

Paused at Robinson's request so only the combat overhaul proceeds. Done:
measurements, tuning, help, tests and docs; `make generate`, `make validate`,
`make js-lint` and `make js-test` passed. Not done: the full
`go test -race -timeout 30m ./...` run (stopped part way) and the PR. Resume by
merging master, running the race suite and opening the PR (or re-measuring
after the overhaul first).
