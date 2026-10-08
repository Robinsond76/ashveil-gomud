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

| `TestHealersDefaultSaysWhyTheLeaderTurns` (found 2026-10-08 in a shuffled loaded run after the speed-turn merge) | The healer foe kept its natural health, so the companions' blows (crits among them) could fell it before Aria's turn; she then turned toward another foe and the test's "no `You turn toward`" check failed. | `hardenBandits` before the round, so the healer outlasts it. 800 of 800 clean under load. |

## Speed-turn merge (82a-82d)

Merged master into the branch. The pace tests (`wiring_pace_test.go`, battle event, morale) replace the pace clock with `SetPaceClockForTest`, so they do not read wall-clock time; the sigil and poison tests use `time.Now()` only with hour-scale or minutes-scale expiries. The helper tests (`hardTo`, `noCrits`, Packlord, Deadeye, Warlord) stayed clean under load after the merge. The unnamed Go Tests failure on #189's head (d7b299b0) could not be read from CI; the healer test above is the only new failure two shuffled loaded full-package runs produced, and it passes locally in the 82d state, so it is the likeliest candidate but unconfirmed.

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

## Review (2026-10-08, Opus review thread)

Merged as built, with one comment fix. Checked:

- **No assertion was weakened or skipped.** The Packlord level 44 case still
  fails when foes start below half (mutation: 40% start fails "a foe at 70% is
  above half"), so the 70% start keeps the check live. The `hardTo` regression
  test fails when the `Training` line is removed.
- **`noCrits` pins every d100, not only the crit roll** (hit, dodge, poison and
  native ability rolls all read 99). Harmless in both callers because
  `alwaysLand` sets to-hit to 100, and the hobble and winding checks are
  health thresholds, not rolls, but the comment said "other rolls stay
  random". Comment corrected to say so and to pair it with `alwaysLand`.
- **Smoke drain**: the 30 ms quiet drain plus the sorted order is sound; a
  page arrives in one write, and only a page naming the next sorted topic as
  `help <topic>` could still mislead the echo wait.
- Gates: `make validate`, `go test -race -timeout 30m ./...` (141 ok, one
  failure below), `make smoke`, GitHub CI green on the reviewed head.

**New flake found, not this PR's:** `TestBattleEventsThroughTheRealRound`
failed once in the full race run: its fight ended and a second fight opened
mid-test (two `fight-end` events, the later payloads on fight 2 with
`fight_round` back at 1), while bandits still stood. It passed 1000 of 1000 runs
alone under four-way load and reproduced once in about 100 runs behind the
balance and banter tests, so it is a rare random sequence, most likely
`closeIdleBattles` breaking the battle off in a round where nobody aims at the
other side (the bandits are not hostile, and the companion set to 1 health
dies early), then the next round opening a new one. The test was last changed
by 82c (pace by action beat). The PR touches nothing it uses. Follow-up:
make the test hold one fight (keep Aria aimed each round, or assert per fight)
or confirm the break-off is intended under the battle clock.

**On #189's unnamed Go Tests failure:** partly agree. The healer test is a
fair candidate, but this battle-event test is the one actually seen failing
after the speed-turn merge, so it is at least as likely. Unconfirmed either way
(the CI log keeps only its last lines).

## Status

Reviewed and merged 2026-10-08 (PR #185). Follow-ups: migrate older tests that
set `HealthMax.Value` directly to `hardTo`; fix the battle-event fight restart
above.
