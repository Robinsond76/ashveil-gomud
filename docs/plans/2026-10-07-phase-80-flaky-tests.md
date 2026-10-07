# Phase 80: tests that fail at random

Owner steer (Robinson, 2026-10-06, full autonomy): use best judgment, record
each decision with its reason. Brief for this phase: fix root causes (shared
state, unseeded randomness, timing), never loosen what a test checks, never
skip or quarantine; no gameplay or balance change.

## Method

Reproduced under load: `go test -race -c`, then four copies of the test binary
in parallel (`-test.count=60-150` each) on a 4-core container, and two full
`modules/company` runs with `-shuffle=on`. A failing run's per-round output
was logged until the cause showed.

## Findings and fixes

| Flake | Root cause | Fix |
| --- | --- | --- |
| `TestWarlordRelentlessQuickensItWhenItsFoeStandsUp` (seen in PR #101 and 72a reviews) | Reproduced 5 in 600 runs. `toughen` and `hardenBandits` set `HealthMax.Value = 1000`, but `RecalculateStats` rebuilds it from `Training + Mods`, and a status ending (the knockdown) recalculates. The watched foe dropped to its natural 48 health, died in a round or two, and Relentless (which watches a *live* foe stand up) never fired. | New `hardTo` helper sets `Training` too; `toughen`, `hardenBandits` and the beast spawn hook use it. `TestHardenedFightersKeepTheirHealthThroughARecalculation` fails without the change. 240 of 240 clean under load afterwards. This also protects every other test that hardens a fighter through these helpers. |
| `TestDeadeyeBoltThatFellsItsFoeNeedsNoWinding/without_it` (seen in a full-suite run; 4 in 60 with the Packlord test beside it) | `alwaysLand` zeroes only the *base* crit chance. A Deadeye adds crit from Eagle's eye (+30%) and Hair trigger (rank 30) makes a *critical* bolt need no winding, so a lucky crit hid the winding the test counts. | New `noCrits` helper pins the percentile roll to a miss (other rolls stay random); the test uses it. 80 of 80 clean. |
| `TestPacklordHoundHobblesFoesBelowThreeQuartersAtRankFortyFive/level_44` | The company's blows land before the hound's bite and wear a foe set at 60% under half, where a level 44 hound legitimately hobbles. Crits made it likelier. | Foes start at 70% (still above half, below three quarters; 200 points of room instead of 100) and `noCrits`. What the test checks is unchanged. |
| `make smoke`: "1 of 1354 help topics render nothing: [keyring]" | The topic list came from map iteration (random order) and the step ended each read at the first `HP:../.. MP:../..]` prompt. `keyring`, `lock`, `unlock` and `picklock` print an example prompt, which ended the read early; the page's tail (`help keyring` is named in the lock pages) then satisfied the *next* topic's echo wait, so that topic read as empty. | Topics are sorted, and the step drains the rest of the page after the prompt. The cause is test-side; no page changed. |

## Not reproduced

- **PR #107's unnamed CI failure.** The GitHub Go Tests log is unreadable past
  the last 5000 lines (see the `ci-logs-unreadable` note), so the failing
  test was never named. The three company flakes above are all shared-helper
  or crit-roll problems that fail only under load, which fits a CI runner,
  so it is probably one of them. Recorded as unconfirmed.
- **A battle test that failed once in the 72a review** (name not recorded):
  the Warlord and Deadeye causes above both fit; no further reproduction in
  two shuffled full-suite runs and 240+ load runs.
- Two shuffled full `modules/company` runs (`-race -shuffle=on`) passed, so
  no further order dependence turned up there.

## Left alone

Many older tests still set `HealthMax.Value` directly (allied companies,
orders, battle view, balance). They sit in fights short enough to see no
recalculation; moving them to `hardTo` is a mechanical follow-up if one of
them flakes.

## Help and tutorial

No player-facing change, so no help page.
