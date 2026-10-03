# Phase 30g6 — candidate verification

Status: **in progress; acceptance gaps remain, not ready to merge**.
Base: master `3bdcb5fb`. Branch: `phase-30g6-tuning`.

## Scope and invariants

See the [approved design](../designs/2026-09-30-phase-30g-tempo-defense-design.md)
and [implementation plan](2026-10-03-phase-30g6-tuning.md). Small HP and bounded
Strength damage replace provisional progression numbers. Stat steps remain
five levels; saved training, vitals, class ordering, uncapped levels and
shared world time retain their existing rules. Natural regeneration already
skips battles; an explicit guard now also keeps combat-round buffs out of the
game-round trigger path. Ordinary timed buffs keep their real-time cadence.

Tests exercise actual round hooks for regeneration, buff cadence, Strength
damage, distinct passive opening aims, actual weakest focus opening, queued
successful tackles, and earned turns after another fighter's kill.

## Fixture fidelity

- Five existing starter kits at every level; no upgraded gear catalog.
- Mirrored level, trained stats, class HP rates and defenses are asserted.
- Balanced stat training is a controlled investment on both sides, rather than
  the shipped weighted companion growth distribution.
- Company automatic abilities and cleric spells are enabled. Passive and
  personality mirrors have no equivalent abilities or healer; equal stats
  therefore do not imply equal physical output. Enhanced healer/guardian
  coordination is measured separately.
- Passive opening aims are paired across ten distinct fighters. Afterwards
  normal sticky aims and strategy upkeep apply. Personalities start through
  upkeep and use rabble's minimum 30% targeting noise. Focus starts on the
  legal weakest foe through the actual formation/strategy selector.
- All outcomes are retained. Fight medians include losses; level 100 is
  reported without enforcing the 10–15-round target. Health removed is the
  starting minus final live health, excluding overkill and accounting for
  healing. Damage and healing are also reported separately.
- No global clock advancement or XP gain; real rolls, no stalls hidden.

The original calibration provider omitted an outer YAML tag and loaded no
class rates. All samples before the corrected provider are invalid for final
calibration; symmetric side-parity alone could not detect the empty provider.

## HP envelope

Actual balanced-investment maxima, before and after this candidate:

| Fighter | Old L10 / L60 | Candidate L10 / L60 | Ratio |
|---|---|---|---|
| Aria (default) | 56 / 157 | 22 / 63 | 2.86 |
| Tamsin (warrior) | 66 / 177 | 30 / 71 | 2.37 |
| Oswin (cleric) | 56 / 157 | 27 / 68 | 2.52 |
| Garrick (warrior, gear bonus) | 67 / 178 | 31 / 71 | 2.29 |
| Ysolde (ranger) | 56 / 157 | 27 / 68 | 2.52 |

`TestBalanceHPEnvelope` asserts each ratio is between 2 and 3. Values use the
same stat investment and item modifiers with the two progression formulas.
Fractional formula terms round down separately; fractional gains accumulate
across levels, so a level may leave max health unchanged.

## Measurements and checks

The first corrected full run completed 6,800 fights (100 per cell): spread
medians 11/14/14/13/12 at L1/5/10/30/60, focused medians
13/15/15/14/13. Focus improved win rates at all five levels but failed duration
at all five; L30 weakest-enemy net health removed was 194.45 versus passive
197.62; L15 versus L10 won 100/100 with no observed loss. L30 versus L10
won 100/100 with no member lost. Coordination assertions passed and no fights
stalled. These measurements precede the ranged readiness fix below and are
not the final post-fix result. The full table is being recaptured.

Post-fix run: **6,800 fights**, 100 per cell, all retained, no stalls. Full
[measurements and distributions](2026-10-03-phase-30g6-measurements.md) include
all 54 tactics/level cells, 12 independent coordination cells and both mismatches.

| Level | Spread median | Focus median | Spread wins | Focus wins |
|---|---|---|---|---|
| 1 | 12 | 12 | 27% | 37% |
| 5 | 13 | 15 | 26% | 31% |
| 10 | 14 | 15 | 51% | 58% |
| 30 | 13 | 13 | 44% | 59% |
| 60 | 11 | 14 | 35% | 50% |
| 100 (report only) | 14 | 16 | 33% | 61% |

Acceptance failures remain visible:

- Focus duration is equal or longer at **all five** asserted levels, although
  its observed win rate improves at each.
- At L30, caster-targeting enemies remove mean 187.50 company HP versus
  passive 187.97. This small difference does not establish a statistical
  conclusion, but it fails the required strict comparison.
- L15 versus L10 wins 100/100, with no observed loss; the favored-but-fallible
  outcome is not demonstrated. Median duration is 6 rounds.

L30 versus L10 wins 100/100 with **no company member lost in all 100**; median
5 rounds. Coordination assertions pass. Spread medians meet 10–15 at every
asserted level, and HP ratios meet 2–3. These successes do not waive the
remaining criteria. Real dice are stochastic; do not rerun or select cells
solely to discard unfavorable measurements.

Checks on the final code:

- `make generate` — passed.
- `make validate` — passed.
- `go test -race ./...` — passed, including company (216.653s).
- New caster/chanting and player/companion readiness/reload regressions passed;
  independent follow-up repeated the relevant tests three times.
- Corrected ordinal and bleed-only fixtures passed three focused runs and
  the full race suite.
- Indexed help renders and tutorial pointer checks pass in the full suite.
- Both admin inline scripts pass `node --check`; Go templates and the mocked
  DOM control/save checks pass. Browser rendering limitation is recorded below.
- Opt-in 100-fight balance command — **failed acceptance as listed above**,
  while the independent coordination subtest passed. This is distinct from
  the green ordinary regression/race suite.

Only documentation changed after the final generation/validation/race run.

## Next calibration

Resolve focused completion time jointly with healing and offensive cadence,
then establish enemy targeting and the L15 mismatch with adequate samples.
Keep the starter-kit, balanced-investment and asymmetric ability/healer
limitations explicit. Do not silently disable class abilities, redefine focus
as a different opening opponent, change sticky targeting, or relax assertions
in order to complete the phase.

## Independent review

Initial review found missing acceptance assertions, incorrect focus/passive
opening setup, Strength omitted from equipment ranking, missing admin control,
and the empty archetype HP provider. These were confirmed and addressed with
integration checks. Final complete-diff review confirmed two P2 integration defects in immediate
replacement: a ready ranged fighter acquired a fresh weapon wait, and caster
focus ignored caster/chanting metadata. Both were fixed narrowly: readiness
and cold-delay flags survive replacement before the fighter's blow, while
its own firing still starts normal reload; replacement candidates now carry
the same caster/chanting information as the ordinary selector. Actual-round
regressions cover player and companion shooting/reload and caster/chanting
priority. A P3 wording correction describes zero Strength growth accurately
without promising the previous formula for arbitrary nonzero minima.
Follow-up review confirmed both fixes, their player/companion reload and
caster-priority coverage, and the wording correction with no new findings.
A final fixture follow-up accepted the ordinal-event ordering correction and
the isolated bleed-only setup. Reviewer-made source edits: none.

Admin Go templates rendered, and both pages passed a JS DOM check against
mocked config reads. The wizard displayed 1.75 with min 0, max 100, step 0.05,
and accepting 2 submitted the exact DamagePerStrength patch. Chromium could
not finish rendering in this execution environment, including with a writable
cache and relaxed sandbox; no screenshot/layout verification is claimed.
