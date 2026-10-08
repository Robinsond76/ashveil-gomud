# Phase 83: test hardening

Owner steer (Robinson, 2026-10-06, full autonomy): decide, record each
decision with its reason, no questions. Brief for this phase: find the real
cause of `TestBattleEventsThroughTheRealRound` failing about one run in a
hundred (its fight ended and a second one opened with bandits standing), fix
the cause and not the symptom (no retries, no skips, no loosened assertions),
and move the older tests that set `HealthMax.Value` directly onto the
phase 80 `hardTo` helper.

## The flake: a production bug, not a leaked test

**Cause (game code).** `engagedWith` (`internal/hooks/combat_engagement.go`)
decides whether the round's upkeep should keep a company and a battle's group
fighting as a whole. It counted only an aim at someone still alive. When the
last target of every fighter on both sides fell in the same round (the leader's
foe died, an archer's winding shot was still on a foe that fell, the last
bandit's mark was a companion who had just died), every aim was on a body at the
next upkeep. The upkeep read that as "nobody is fighting" and skipped the
re-aim it exists to do. `closeIdleBattles`, which runs right after it, then
found no aim on either side and broke the battle off with foes standing. That
ended the fight (a second `fight-end`, a second summary) and the next round
opened a new one, so the battle screen's round counter went back to 1. The
battle clock itself is not at fault: nothing in it decides who aims at whom.
Whether the case existed before the 82 overhaul was not checked; the test only
began to hit it after 82 changed how its fight plays out (tempo, one action
per beat).

**How it was found.** A non-race build of the test, run in four
processes, four at a time (about 0.5 s a run alone), reproduced the failure
alone: 5 failures in 2,886 runs (0.17%). The earlier "1000 of 1000 alone" was
too few runs to see a rate that low. A temporary dump at
`endBattle` printed the state at the break-off: one bandit standing, aimed at a
dead companion; the archer companion mid-shot on an already dead bandit; the
leader with no target; nobody aimed at anyone alive. (The reviewer's reading,
that this was `closeIdleBattles` breaking off a battle where nobody aims, was
right; the missing piece was why the upkeep did not re-aim first.)

**Fix.** `engagedWith` also counts a plain attack on a mob that has fallen or
been removed (`aimedAtTheFallen`), for the leader, each living companion and
each living member of the group. The existing upkeep (`keepCompanyEngaged`,
`keepPartyEngaged`) already re-aims a member whose target is dead, so the
company turns on the foes that stand and the battle goes on as one fight.
No other behavior changes: a battle where someone has no aim at all (a stood
down player, a group that lost interest) still closes as before.

**Regression test.** `TestABattleGoesOnWhenEveryAimFellInOneRound` puts every
fighter's aim on a fallen mob and runs one real round (`DoCombat`): before the
fix the battle ends with `broken-off` and the test fails; after, the same fight
goes on and no `fight-end` is sent. It is deterministic. The flaky test itself
is unchanged.

## A shared-state leak found on the way (tests)

The brief's first suspect was state leaking between tests, so the package was
audited for it. One real leak turned up, though it is not what made this test
fail (the failure above reproduces alone, with no other test before it):
`TestNarrationPreservesCombatOutcome` calls `rand.Seed(29)` to replay a golden
record. `rand.Seed` replaces the process-wide dice and nothing can put them
back, so every test that ran after it in the same process rolled the rest of one
fixed stream, from a position that depended on how many dice the tests between
had rolled. A failure seen under one shuffle then could not be repeated under
another. The test now seeds through `seedDice` (`isolation_test.go`), which
reseeds the dice from the clock when the test ends;
`TestSeededDiceDoNotOutliveTheirTest` fails without that. The seed is still
taken at the same point in the test, after the brawl is built, so the golden
does not move. `wiring_battlefield_test.go` also called `rand.Seed(30)`, which
Go 1.24 makes a no-op unless `GODEBUG=randseednop=0`, so it seeded nothing
(those tests have always rolled random dice and pass); the dead call and its
import are removed. Per-test state in `internal/hooks` (ability cooldowns,
fight-keyed order maps, cast aims) was also checked at each brawl's start in a
shuffled full run: it is keyed by instance ids, which never repeat in a
process, or is pruned each round, so none of it reaches the next test.

## Migrating `HealthMax.Value` to `hardTo`

Phase 80's `hardTo` sets `HealthMax.Training` as well as `Value`, because
`RecalculateStats` rebuilds `Value` from `Training + Mods` and any status that
ends recalculates. Every test in `modules/company` that set a fighter's
`HealthMax.Value` by hand now goes through `hardTo(character, hp)`, or through
the new `hardMaxTo(character, hp)` where the test sets only the maximum and
keeps the health it has (`hardTo` is `hardMaxTo` plus `Health = hp`). The
numbers are unchanged (1000, 10000, 100, 5000 and so on), a test that set a
different current health sets it after, and no assertion changed.

Left as they are on purpose: sites that only read `HealthMax.Value` (asserts,
balance printouts), the helper itself, and the tests in other packages
(`internal/combat`, `internal/characters`, `modules/gmcp`, and so on). `hardTo`
is a helper of the company package's tests, and those tests run a single swing
or a unit check, never a fight long enough for a recalculation, as phase 80
found.

## Found and not fixed

- `TestDreadWhisperMakesAFoeTakeAMoraleCheck` failed once in a shuffled subset
  run (see "Measured"). Not reproduced; a follow-up if it returns.
- Review note from the 79 reviewer: `make smoke` once failed at the restart
  step with no message and an empty server log, then passed on a verbose rerun.
  Not chased; the fixes above do not touch that step.

## Measured

The flaky test alone, non-race build, four processes at a time, same machine:

| | Runs | Failures |
| --- | --- | --- |
| Before the fix | 2,886 | 5 (0.17%) |
| After the fix | 2,961 | 0 |

At the before rate, 2,961 clean runs would happen by chance about 0.6% of the
time. The deterministic regression test
(`TestABattleGoesOnWhenEveryAimFellInOneRound`) fails before the fix and passes
after, every run.

Shuffled and loaded, as the brief asked: 100 shuffled passes of the 19 tests
that touch the battle clock, pace and morale (`-shuffle=on`, four at a time)
passed 1,900 of 1,900 with the fixes. A full-package shuffled run (race) takes
about 13 minutes on this container (the suite is about 1,100 tests), too slow
to repeat 50 times; the full race suite ran once on the final code instead (see
the status log). The failure rate is low enough (about 1 in 600) that the
alone-loop above is the stronger measurement.

Not fixed, seen once: `TestDreadWhisperMakesAFoeTakeAMoraleCheck` failed once in
the first shuffled subset run (its chant restarted with no morale check, so
neither "loses nerve and flees." nor "(dread)" appeared). It passed 600 of 600
alone and 1,900 of 1,900 in the shuffled passes above, so it was not
reproduced; recorded as a follow-up.

## Decisions

1. Fix the production upkeep, not the test. A fight that ends and reopens
   mid-battle is visible to a player (a summary mid-fight, the round counter
   back to 1), and the test was right to object.
2. Count aims on the fallen in `engagedWith` rather than loosening
   `closeIdleBattles`. The idle check is correct for "nobody is fighting"; the
   defect was that the upkeep, which should have re-aimed, was gated on the
   same wrong reading.
3. A deterministic regression test of the state (everyone aimed at a fallen
   mob), not a probabilistic loop, so the suite catches a regression every run.
4. `hardMaxTo` as a separate helper so sites that only raise the maximum keep
   their current health, which some tests rely on (the healer test needs Aria
   wounded).
5. Other packages' `HealthMax.Value` sites are not migrated (see above).
