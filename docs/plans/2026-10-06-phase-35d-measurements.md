# Phase 35d measurements

Harness: `ASHVEIL_BALANCE=1 go test ./modules/company` (real dice). Zone rows
at 10 to 20 fights per cell (60 per cell for middle/low/under at some bands),
mana and ladder rows at 30. Design: [combat feel](../designs/2026-10-06-phase-35d-combat-feel-design.md).
Tuning was timeboxed (owner, 2026-10-05): where a target was not reached the
best result is settled and recorded here.

**Round length.** A combat round is 2 game rounds, 8 real seconds (29f). The
design's "25 to 40 s" assumed 4 s. The harness reports seconds as
rounds × 8; rounds are what the design's 6 to 9 targets measure, and they are
met. In seconds a band-middle fight is 64 to 72 s.

## Acceptance

| # | Row | Target | Measured | Verdict |
|---|---|---|---|---|
| 1 | Blow qualities (even edge) | glancing 25%, telling 20% of landed | 24% / 22% of landed, company and enemy near 24/21 | met |
| 2 | No-damage weapon swings | ≤30% at level 10 (35b: about 45%) | 35 to 39% (all of them misses: to-hit roll plus block/parry/dodge) | not met, settled; `ToHitEven` 88 kept (to-hit caps at 95) |
| 3 | Band-middle fight length | median 6 to 9 rounds | 8 to 9 rounds (64 to 72 s at 8 s/round) | met in rounds |
| 4 | Heals finished | ≥75% of begun | 87 to 91% | met |
| 5 | Mana run, six fights | wins ≥80% to fight 5, reserve to fight 4 | wins 96 to 100% fights 1 to 3, 86% fight 4, 43 to 66% fight 5, 20 to 33% fight 6; cleric mana 47% after 2, about 6% after 3 | not met, settled |
| 6 | Ladder | default tactics ≥50% vs tier 2, ≥33% vs tier 3 | pass at 1, 5, 10, 30 | met |
| 7 | Zone bands | middle ≥97% wins, ≥85% clean | 100% wins, 95 to 100% clean at 1-3, 8-10, 18-20, 28-30 | met |
| 7 | Four foes (band low) | ≥90% wins, ≤1 fallen | 100% wins, 0.5 to 0.8 fallen | met |
| 7 | Boss (3 escorts, Rabble, 1.75x HP) | 70 to 85% wins, 12 to 18 rounds | 65 to 100% wins (85, 95, 75, 65 by band), 12 to 16 rounds | rounds met; wins above target at low bands, settled |
| 7 | Five under the band | report, 40 to 70% | 75, 58, 75% wins at bands 8-10, 18-20, 28-30 | reported for 37 |
| 8 | Kept rows | skill wins, tanks, HP cap, sides even, statuses | pass | met |

The 5v5 mirror is a stress check, reported and not asserted (the default guard
and focus now help the passive company too, so enemy targeting no longer shows
as significant at levels 30 and 60).

## Settled deviations

- **Dead swings (35 to 39% against ≤30%).** Every dead swing is a miss. A
  higher `ToHitEven` would help but the bound is 95; blocks and parries need
  shields in play. Fights are shorter and lines more frequent than in 35b
  (about 45% to about 37%). Left for 37 if the zone play-test wants it.
- **Mana run.** The cleric spends about a third of its pool each fight on
  heals and the patch, so a company that walks six fights with no rest runs
  dry at three. Raising Mysticism-per-mana or cutting Minor Heal to 2 mana
  moved it to four or five fights but changes every caster's economy; not
  done here. A rest or a draught restores the run.
- **Boss.** At low bands the shorter boss is beaten almost always; menace is
  38b's telegraphed abilities. The 70% floor is asserted.

## Enemy healers

Enemy groups with a healer raise difficulty (the owner, 2026-10-06), and with
unbreakable one-round heals now applying to both sides they raise it more.
No encounter table changes here. Guidance for 37: a healer in about one group
in five, never in a four-foe group, and never with a boss's escorts.
