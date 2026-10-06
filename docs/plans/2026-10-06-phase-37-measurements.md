# Phase 37 measurements

Run: `ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=40 go test ./modules/company -run TestBalanceEncounterShapes`
(`modules/company/balance_encounter_test.go`, five-member companies, real
encounter shapes from `internal/encounters.Plan`; one full run, 346 s, after a
small-sample tuning pass). Columns: wins, fights with no one down, fallen per
fight, company HP lost, median rounds, seconds.

| Band | Shape (company level) | Fights | Wins | No one down | Fallen | HP lost | Rounds |
|---|---|---|---|---|---|---|---|
| 5-7 | ordinary (5) | 160 | 100% | 92% | 0.09 | 10% | 9 |
| 5-7 | four foes (5) | 40 | 100% | 68% | 0.40 | 22% | 14 |
| 5-7 | healer group (5) | 80 | 100% | 78% | 0.24 | 16% | 11 |
| 5-7 | ordinary (6) | 160 | 100% | 91% | 0.09 | 9% | 8 |
| 5-7 | four foes (6) | 40 | 100% | 78% | 0.28 | 18% | 12 |
| 5-7 | healer group (6) | 80 | 100% | 95% | 0.06 | 12% | 10 |
| 5-7 | boss + 2 escorts (7) | 40 | 100% | 100% | 0.00 | 9% | 12 |
| 5-7 | boss + 3 escorts (7) | 40 | 100% | 78% | 0.23 | 16% | 16 |
| 10-12 | ordinary (10) | 160 | 100% | 96% | 0.04 | 8% | 10 |
| 10-12 | four foes (10) | 40 | 100% | 55% | 0.53 | 21% | 15 |
| 10-12 | healer group (10) | 80 | 100% | 96% | 0.04 | 10% | 12 |
| 10-12 | ordinary (11) | 160 | 100% | 98% | 0.03 | 7% | 9 |
| 10-12 | four foes (11) | 40 | 100% | 75% | 0.28 | 16% | 13 |
| 10-12 | healer group (11) | 80 | 100% | 96% | 0.04 | 9% | 11 |
| 10-12 | boss + 2 escorts (12) | 40 | 100% | 98% | 0.03 | 9% | 13 |
| 10-12 | boss + 3 escorts (12) | 40 | 100% | 78% | 0.23 | 15% | 16 |
| 10-12 | under-levelled, 5 levels below the band's low | 160 | 87% | 61% | 1.01 | 29% | 16 |

The boss rows with company tactics on are within 10 points of the plain rows.

## Owner's difficulty rule (2026-10-06)

Difficulty should come only from a zone above the company's level; groups are
2-3 foes, four-foe groups carry lower-level foes, healers are rare.

- **Met:** a company at the band's levels wins ordinary, healer and boss
  fights (100% wins, 90%+ with no one down for 2-3 foe groups); the group
  size is 2-3 foes (four sometimes), four-foe groups are all at the band's
  low, boss escorts at the low, enemy levels never follow the player; healer
  groups are capped at 20% of a table's weight, never in four-foe or boss
  groups.
- **Partly met:** four-foe groups still cost a company at the band's low a
  life in about half the fights (55-68% no one down) at 100% wins. That is
  fine, not easy.
- **Not met:** "a zone two bands up is hard". A company five levels under the
  band still wins 87% (1 fallen per fight). Only two bands are shipped, so
  this measures the gradient, not a specific 13-15 zone. Low-level bosses are
  easy (no menace until 38b's abilities). Left to 37b (encounter and pacing
  tuning), which also raises healer frequency once the 35e "focus the healer"
  strategy lands.
- **Follow-up:** the "focus the healer" company strategy does not exist yet;
  it is phase 35e. Until then healers stay rare.

## Settled

- Asserted floors: ordinary and four-foe 85% wins, healer groups 60%. A
  6-fight quick pass failed the four-foe floor on noise; the 40-fight run
  passes.
- Cleric mana still runs dry around fight 3; mitigated by pacing: a rest
  point at least every ~20 eligible entries (about 8.7 entries per battle
  at 15%), see the plan.
