# Phase 84: the Sorcerer's Lance and the level 20-22 rest rhythm (2026-10-08)

Two balance follow-ups from the phase 81 review ([plan](2026-10-07-phase-81-class-tuning.md#review-2026-10-08)).
Numbers and class logic only; no content (the stock world is temporary).

## 1. The advanced Sorcerer

**Problem.** At L50 the Sorcerer won 22-34% of mirror fights against the base
wizard's 40%; at L25 and L40 it was within noise. The cause (phase 81 review):
`strategy.Decide` and `orders.Caster.Attack` try the Arcane Lance (one foe, a
two-round chant, same chant as Magic Missile) before the group spell, so
against a full group the Sorcerer bolts one foe while the wizard's Shower of
Sparks hits them all.

**Rule (decided).** The Sorcerer looses the Lance against three foes or
fewer and showers the group with sparks against four or more, going back to the
Lance as the foes fall. The limit is a class effect (`lancefoes` = 3, set by
the Arcane Lance rank) so help, ranks and code agree; the High Sorcerer's
High Lance rank sets it to 0 (no limit), see "Decisions".
`strategy.Situation.LanceFoes` carries it into `Decide`; `orders.Caster.LanceFoes`
into the `strongest` order (an order aimed at one foe, `break`, keeps the
Lance). Foe casters have no limit (0). `strategy.LanceFits` is the one test.

**Why 3, a flat count and not a level-scaled one.** Candidates measured on the
Sorcerer (placed line, 200 fights a cell, noise about +/-7; base wizard
51 / 40 / 46 at L25 / L40 / L50 in the same run):

| Rule | L25 | L40 | L50 |
| --- | --- | --- | --- |
| As built (Lance always) | 50 | 37 | 34 |
| Lance only at 1 foe | 55 | 46 | 43 |
| Lance at 2 foes or fewer | 59 | 49 | 46 |
| **Lance at 3 foes or fewer** | 49 | 44 | 46 |

Every limit up to 3 closes the L50 gap and none hurts L25, so a level-scaled
limit buys nothing; 3 is the closest to the wizard at all three levels (2 reads
8 and 9 points above it at L25 and L40, inside noise but the less honest
"level"). An expected-damage rule was not built: the sim says the break-even
crowd is about 3, where the damage arithmetic (Lance about 5 Sparks hits)
says 5, because the Lance overkills a dying foe and its chant is lost to any
blow.

**Final measurement** (200 fights a cell, one run each; the base wizard from
the same runs, the earlier runs of the base in brackets):

| Cell | Base wizard | Sorcerer (this phase) | Sorcerer before |
| --- | --- | --- | --- |
| L25 | 54 (51) | **48** | 50 |
| L40 | 43 (40, 46) | **48** | 37 |
| L50 | 50 (46) | **47** | 34 |

Within noise of the wizard at all three. The runs took 22-260 seconds, so the
spec's 100-fight run was doubled.

**Decisions.**
- **The High Sorcerer keeps the Lance against any number of foes.** Applying
  the rule to the elite dropped it from 65 / 70 to 49 / 48 at L40 / L50 (it was
  tuned in phase 81 to lead the base wizard, level with the Archmage, and its
  Lance is +75% and +90% with a twin bolt). The High Lance rank (30) sets
  `lancefoes` to 0. Measured with the limit scoped this way: 63 / 66 (base 43 /
  50), unchanged within noise.
- A Sorcerer that has no Shower of Sparks known (a test fixture, never a
  played character: the base wizard teaches it) would cast Magic Missile at a
  crowd rather than the Lance; the existing fall-through order is kept.
- Help: `wizard-routes` (the Sorcerer's rank text and the closing paragraph),
  `high-sorcerer` (rank 30 and "How it plays") and the in-game rank texts say
  what the Sorcerer casts now. The tutorial hint ("a Sorcerer hurls a heavy,
  costly Arcane Lance") stays true.

**Tests.** `TestSorcererSparksACrowdAndLancesAFew` (strategy), the orders
equivalent, and `TestSorcererSparksACrowdAndLancesAFew` in `modules/company`
through the real strategy pass and combat round (four bandits: sparks; three:
the Lance; a High Sorcerer with five: the Lance). The older Sorcerer wiring
tests now thin the bandit group to three, which is what they meant (a Lance
at one foe).

## 2. Levels 20-22: five members rested after 12 fights

**Problem.** `TestBalanceAtLevel` read a median 12 fights before a rest for
five martial members at 20-22 against the 15-20 target (3-5 read 18, 10-12
read 16).

**Change.** The knob PR #179 used: the HP share ordinary at-level foes spawn
with. `encounters.OrdinaryHPPercent` stays 40; bands whose low end is
`HighBandLow` (20) or more use `HighBandHPPercent` = 30
(`encounters.BandHPPercent`; `HPPercent`'s fade to full HP under the band
starts from it). The 3-5 and 10-12 bands, bosses, escorts and story groups
are untouched. Applied to every band from 20 up, not the single 20-22 band: the
shipped bands 21-24, 25-28 and 29-33 share the curve that made 20-22 tire, and
a rule keyed on the exact band 20-22 would leave its neighbours at 12. Only
20-22 is measured (the harness's third band); the higher bands are a
projection, flagged as a follow-up to confirm when the world is rebuilt.

**Measurement** (`ASHVEIL_BALANCE_BAND=20-22`, 200 fights a cell, median
fights before a rest p25 / median / p75; 40% is the phase 81 review's 30-fight
table, 30% this phase):

| Company | Before (40%) | After (30%) |
| --- | --- | --- |
| 5 martial (target 15-20) | 10 / 12 / 15 | **13 / 16 / 19** |
| 4 martial (8-14) | 9 / 12 / 14 | 9 / 12 / 15 |
| 3 martial (5-7) | 6 / 8 / 9 | 7 / 9 / 11 |
| 5 magic (7-12) | 8 / 10 / 11 | 8 / 10 / 12 |
| 4 magic (5-8) | 6 / 9 / 11 | 7 / 10 / 12 |
| 3 magic (3-5) | 4 / 5 / 6 | 4 / 5 / 7 |
| solo wins | 60% | 76% |

Other settings tried (100 fights): 30% read 5 martial 15 (13 / 15 / 18) and 17
in a 60-fight run; 25% read 16 (12 / 16 / 19) and moved the 4- and 3-member
medians out to 15 and 11, so 30% stays. Three, four and magic companies are
unchanged within noise (they were already at or over their targets; the three
martial overshoot of 5-7 was accepted in #179). Solo still loses a quarter of
its fights and rests after every one. The full `TestBalanceAtLevel` (30 fights,
all bands) passes with its slack.

**Noted, left alone.** The magic-heavy five at levels 3-5 reads 5 against the
7-12 target: a level-5 wizard's mana limit, as before.

**Tests.** `TestHighBandsSoftenOrdinaryFoesFurther` (internal/encounters) and
`TestOrdinaryFoesSpawnSofterStillInAHighBand` (modules/encounters, a spawn
through the real step listener in a 20-22 zone).

## Verification

`make generate`, `make validate`, `make js-lint`, `make js-test`,
`go test -race -timeout 30m ./...`.

## Review (2026-10-08)

Independent review of PR #193. No game code changed.

**Checked.**
- The limit is keyed on the rank, not a name: the Arcane Lance rank sets
  `lancefoes` 3 and the High Sorcerer's High Lance rank sets it 0; ranks along
  a class path overwrite earlier values (`EffectsForLineage`), so a High
  Sorcerer reads 0 and a Sorcerer 3. `Decide` and the `strongest` order both
  count standing foes (`standingFoes`), so the Sorcerer goes back to the Lance
  as foes fall.
- Battle text: a crowd gets the wizard's own Shower of Sparks chant and lines,
  so nothing new to read. `why` explains rolls, not spell choice, so it needs
  no change. Help (`wizard-routes`, `high-sorcerer`, the rank texts) claims
  exactly the rule; `strategy` still reads true; no help page quotes the 40%
  foe share.
- Sorcerer L50 spot-check (placed line, 60 fights): Sorcerer 46%, base wizard
  46%, the same run. Matches the build's 47 / 50.

**Decision: the 30% share stays for every band from 20 up.** Spot-checked the
two higher shipped bands with five members (40 fights a cell; the harness's
`ASHVEIL_BALANCE_BAND` now takes any band, such as `25-28`):

| Band | 5 martial at 40% | 5 martial at 30% | 5 magic at 30% |
| --- | --- | --- | --- |
| 20-22 (build) | 12 | 16 | 10 |
| 25-28 | 18 | 21 | 11 |
| 29-33 | 15 | 18 | 11 |

At 40% the higher bands already sit inside 15-20, but 29-33 sits on its floor;
at 30% they read 21 and 18, one over the top at most, within the noise of a
40-fight cell. Kept the blanket rule rather than a per-band step because
(1) the owner asked that at-level fights err easy, so one over the top is the
safer miss than one under the floor; (2) a step per band would tune numbers
to a stock world that is being replaced; (3) one rule is easier to retune
when the real zones exist. Every company won every fight.

**Gates.** `go test -race -timeout 30m ./...` and `make smoke` on the final code.
