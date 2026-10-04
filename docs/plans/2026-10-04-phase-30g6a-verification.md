# Phase 30g6a — combat fixes and harness: verification

Base: master `3bdcb5fb`. Branch: `phase-30g6a-fixes`. Design: the
[30g6 amendment](../designs/2026-10-04-phase-30g6-amendment.md), decisions C
and D and its "Slices" section. Split from the first 30g6 candidate
(`phase-30g6-tuning`, `5b36efd`); master's balance numbers are unchanged.

## What ships

- **Combat fixes** (from the candidate, with its real-round regressions):
  a successful tackle queued in the ability pass is not repeated by a
  second warrior; a replacement target carries caster and chanting data,
  so caster focus survives a retarget; a fighter whose foe fell earlier in
  the round picks a legal replacement and keeps its earned turn (a ready
  archer keeps its readiness, then reloads as usual); combat-round statuses
  never tick on game rounds.
- **Harness:** levels 1/5/10/30/60 asserted and 100 reported; level 15 and
  30 against level 10; weakest and caster enemy targeting; net health
  removed (no overkill, healing counted); distinct opening aims; shipped
  class HP rates; a true mirror (decision C: company abilities off and its
  cleric a fighter in the spread and focus cells) plus a kit cell; enemy
  targeting judged with a one-sided Welch statistic; company focus reported
  only (decision D, owner 2026-10-04); a won-fight mean-rounds column.
- **Help:** `tempo` (earned turn after an earlier kill), `abilities`
  (no repeated tackle), `combat` (no natural recovery in battle);
  `TestBalanceHelp`.

## Measurements on master's numbers

`ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=100 go test ./modules/company -run
'^TestBalance(5v5|Mismatches|Coordinated)$' -v` (100 fights a cell, every
fight kept), measured before the review follow-up below (enemy earned
turns, the mirror's cleric). As expected, master's numbers fail acceptance;
30g6 tunes them.

| level | company | wins | rounds p10/median/p90 | won-fight mean | stalls |
|---|---|---|---|---|---|
| 1 | spread (mirror) | 48% | 11/13/19 | 14.8 | 0 |
| 1 | kit | 56% | 12/16/23 | 17.8 | 0 |
| 1 | focus | 57% | 11/16/23 | 14.4 | 0 |
| 10 | spread (mirror) | 41% | 61/73/88 | 74.6 | 0 |
| 10 | kit | 97% | 65/75/94 | 77.1 | 0 |
| 30 | spread (mirror) | 58% | 133/147/178 | 148.7 | 1 |
| 60 | spread (mirror) | 48% | 149/164/193 | 165.0 | 6 |
| 100 | spread (mirror) | 39% | 168/185/200 | 179.5 | 26 |

- The mirror is even at every level (41–61%), as decision C intends.
- Fights from level 5 up run far past 10–15 rounds; level 15 beats level
  10 in 100 of 100. Weakest and caster targeting at level 1, and caster
  targeting at level 10, did not significantly hurt the company more than
  passive enemies. These are 30g6's to fix.
- Coordination (33i2's `TestBalanceCoordinated`): at level 10 the company
  won 64% against the default personality but 20% against tier-3 roles,
  flipping a clear winner, which that test forbids. Master's 100-round
  fights give coordination far longer to work; 30g6 rechecks it.

## Checks

After the review follow-up: `make generate` and `make validate` passed;
`go test -race ./...` passed. New regressions fail without their fixes
(checked by reverting each fix locally): the queued-tackle opening, the
enemy earned turn, and the mirror cleric as a caster target. The review
outcome is in `docs/PROJECT_STATUS.md`.
