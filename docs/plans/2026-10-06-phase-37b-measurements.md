# Phase 37b measurements

Run: `ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=40 go test ./modules/company -run TestBalanceEncounterShapes`
(one full run, 447 s, after small-sample tuning of 8-10 fights per cell).
Five-member companies, real encounter shapes. Columns: band, shape (company
level), fights, wins, fights with no one down, fallen per fight, company HP
lost, median rounds, seconds. Compare [phase 37](2026-10-06-phase-37-measurements.md).

| Band | Shape | Fights | Wins | No one down | Fallen | HP lost | Rounds | Seconds |
|---|---|---|---|---|---|---|---|---|
| 5-7 | ordinary L5 | 160 | 100% | 89% | 0.13 | 10.7% | 9 | 72s |
| 5-7 | four L5 | 40 | 100% | 45% | 0.68 | 26.2% | 15 | 120s |
| 5-7 | healer L5 | 80 | 100% | 81% | 0.19 | 15.4% | 11 | 88s |
| 5-7 | ordinary L6 | 160 | 100% | 97% | 0.03 | 7.7% | 8 | 64s |
| 5-7 | four L6 | 40 | 100% | 68% | 0.40 | 17.8% | 12 | 96s |
| 5-7 | healer L6 | 80 | 100% | 94% | 0.06 | 10.7% | 10 | 80s |
| 5-7 | boss+2 escorts L7 | 40 | 100% | 100% | 0.00 | 7.4% | 15 | 120s |
| 5-7 | boss+2 escorts L7 tactics | 40 | 100% | 95% | 0.05 | 8.2% | 16 | 128s |
| 5-7 | boss+3 escorts L7 | 40 | 100% | 88% | 0.12 | 14.6% | 18 | 144s |
| 5-7 | boss+3 escorts L7 tactics | 40 | 100% | 75% | 0.28 | 14.6% | 19 | 152s |
| 5-7 | under 2 (L3) | 160 | 79% | 52% | 1.46 | 37.1% | 17 | 136s |
| 5-7 | under 3 (L2) | 160 | 60% | 41% | 2.26 | 52.6% | 19 | 152s |
| 10-12 | ordinary L10 | 160 | 100% | 94% | 0.06 | 7.5% | 10 | 80s |
| 10-12 | four L10 | 40 | 100% | 62% | 0.38 | 18.1% | 15 | 120s |
| 10-12 | healer L10 | 80 | 100% | 95% | 0.05 | 9.9% | 11 | 88s |
| 10-12 | ordinary L11 | 160 | 100% | 98% | 0.03 | 6.7% | 9 | 72s |
| 10-12 | four L11 | 40 | 100% | 92% | 0.07 | 10.7% | 13 | 104s |
| 10-12 | healer L11 | 80 | 100% | 98% | 0.03 | 7.9% | 11 | 88s |
| 10-12 | boss+2 escorts L12 | 40 | 100% | 98% | 0.03 | 10.2% | 16 | 128s |
| 10-12 | boss+2 escorts L12 tactics | 40 | 100% | 98% | 0.03 | 10.0% | 16 | 128s |
| 10-12 | boss+3 escorts L12 | 40 | 100% | 88% | 0.12 | 11.6% | 18 | 144s |
| 10-12 | boss+3 escorts L12 tactics | 40 | 100% | 75% | 0.25 | 16.8% | 20 | 160s |
| 10-12 | under 2 (L8) | 160 | 96% | 78% | 0.41 | 17.0% | 14 | 112s |
| 10-12 | under 3 (L7) | 160 | 85% | 58% | 1.15 | 32.4% | 18 | 144s |
| 10-12 | under 5 (L5) | 160 | 51% | 29% | 2.74 | 61.4% | 22 | 176s |

## What changed and why

- **Level gap (the difficulty rule):** `SkillEdgeSpan` 14 to 8. Under the old
  value a company five levels under a band still won 87% of fights. Trials at
  the 10-12 band, five levels under: span 14, 87%; span 10, 57-59%; span 8,
  46-51%. At the band's own levels nothing worsened (100% wins, 90%+ no one
  down), and two levels under stays fine (96% wins). The owner's gradient now
  reads: company 8 in a 7-9 zone easy, 10-12 fair (96%), 13-15 hard (about
  half the fights, 2.7 members down per fight, 61% of company HP lost).
  Three under: 85% wins and 1.15 down. It is a global edge change; the help
  pages and the shipped example number were updated.
- **Bosses:** 3 levels over their escorts (was 2) and twice the HP (was
  1.75x). Rounds at the band's top went 12-16 to 15-20 (two to three minutes),
  with a fallen member now and then (tactics: 0.25-0.28 fallen with three
  escorts). Wins stay 100% for a company at the band's top, so a boss is long
  and costly rather than lethal; a company under the band loses it by the
  level-gap change above. Trial values and rejected: +4 levels and 1.5x HP gave
  20-25 rounds (too slow at 8 s a round); +3 and 1.5x only slightly longer.
- **Healers:** Dark Forest gains a goblin shaman band (weight 20 of 120) and
  the Catacombs chanter pack drops to 16 of 96, so both tables are one group in
  six. Healer groups measured 100% wins, 81-98% no one down.
- **Boss respawn:** a lair stays quiet 30 real minutes for a company (and
  its party) after the boss falls. Not a balance number; tested.

## Rows that stay short of the original targets (recorded, not chased)

- Four-foe groups at the band's low still cost a member in about half the
  fights at the low level (45-62% no one down), at 100% wins. Fine, not easy.
- Band 5-7, two levels under (level 3): 79% wins, steeper than band 10-12
  because early levels are worth more; the harness asserts 75%.
- Cleric mana still runs dry around fight 3; pacing (rest points) covers it.
- Harness cells in tier-appropriate gear were not built (the harness mirror
  has no item tiers); deferred, see PROJECT_STATUS.

## Review check at higher bands (span 8, 6 fights per cell)

| Band | At band (ordinary) | Under 2 | Under 3 | Under 5 |
|---|---|---|---|---|
| 25-27 | 100% | 83% | 70% | 41% |
| 35-37 | 100% | 79% | 70% | 45% |

Span 14 at band 25-27 for comparison: under 2 100%, under 5 79%.
