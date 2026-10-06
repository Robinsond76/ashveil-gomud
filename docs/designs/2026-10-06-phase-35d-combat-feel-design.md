# Phase 35d: combat feel

Status: **design drafted 2026-10-06 at the owner's request**, after the
[combat rebalance second opinion](../plans/2026-10-06-combat-rebalance-second-opinion.md).
It needs the owner's approval before implementation (handoff rule 20).
Execution plan: [phase 35d plan](../plans/2026-10-06-phase-35d-combat-feel.md).
It follows [35b caster power](../plans/2026-10-05-phase-35b-caster-power.md)
(merged in PR #21) and comes **before** phase 37, whose encounter tuning
measures against the numbers this phase sets. It amends the
[35a2 design](2026-10-05-phase-35a2-skill-over-hit-points-design.md)
(acceptance rows 1 and 4) and the
[level impact design](2026-10-05-level-impact-class-power-design.md) §4
(the band table and the boss shape).

## Goal

Combat is watched, not played: the player sets the company up and the fight
runs on each character's strategy (the owner's Ogre Battle direction). In a
watched fight three things matter:

1. every line means something;
2. the setup decisions visibly pay off, or visibly fail;
3. the fight ends before the player's attention does.

After 35b, about 45% of swings produce no result, a cleric's heal breaks on
almost any touch, a company patches only to half health so its fourth fight
is a coin flip, a boss is thirty rounds of attrition, and the enemy grows
smarter with level while the company does not. The
[second opinion](../plans/2026-10-06-combat-rebalance-second-opinion.md)
traces each of 35b's remaining balance misses to one of those. This phase
changes the rules that cause them rather than retuning chances the data
shows are at their ceiling.

The owner (2026-10-06) is "ok with overall changes to give players better
gameplay".

## What stays

- **Skill over hit points** (35a2): a level makes a fighter better, not
  thicker. Attack and Evasion ratings, the skill edge and small HP growth
  are unchanged in shape.
- **Mana only from rest, inns and draughts**, healers patch afterwards, HP
  trickles only to 50% (35b, owner decisions).
- **No player input mid-battle**; strategies and formation decide.
- **Strategy rises with enemy level** (33i2 tiers).
- **Shields are the tank's defense**: block stays a full negate, so the
  warrior keeps taking about 0.70× a rogue's damage per swing (35b).
- **Crits stay Smarts** and are not touched by the skill edge.

## Decisions

### 1. A swing lands; skill decides how hard

Today a swing passes a 75% to-hit roll, then one active defense (block,
parry or dodge), then armor. The two rolls leave about 45% of swings with
no result, and 35a2's "five landed blows to kill" fixes fight length at
13 to 15 rounds for an even 5v5 (see the second opinion §1).

New rule: the to-hit roll stays but at `ToHitEven` **88** (bounds 10 to 95
as now), and a blow that lands is one of three **qualities**, chosen by a
roll the combined edge shifts:

| Quality | Damage | Even odds | Edge +1 | Edge −1 |
|---|---|---|---|---|
| glancing | ×0.5 | 25% | 5% | 50% |
| solid | ×1.0 | 55% | 45% | 45% |
| telling | ×1.4 | 20% | 50% | 5% |

The odds move linearly with the edge between the columns. Expected damage at
even is about 0.95× today's, so damage per landed blow is unchanged in the
mean; the skill edge now shows in the words and the numbers of every blow,
not in whether a line appears. A crit (Smarts) stacks on top of the quality
as it does today. Spells are untouched: their `SpellFactor` already scales
damage by the edge.

Dodge and parry stay as full negates at their 35b values (8 at even). They
matter for light fighters and they are rare enough not to pad fights.

Message templates gain the quality: the existing damage-percent message
tiers already distinguish weak and strong blows, so the quality maps onto
them (glancing picks from the low tier, telling from the high tier) and a
suffix names a telling blow. The battle view (34a) shows the quality word.

Expected effect, from the 35b tables: landed blows per fighter-round rise
from about 0.44 to about 0.56, so the mirror's median falls from 14 or 15
toward 11 or 12 and a band-middle fight from 9 or 10 toward 7 or 8, with no
change to the fifth-of-HP solid blow. The harness decides the final
numbers.

This amends 35a2 acceptance row 1: the 15 to 25% band now describes a
**solid** blow; glancing lands 8 to 13% and telling 21 to 35%.

### 2. A one-round heal resolves

Today any blow that does 1 damage breaks a chant with chance 40 + 2 per
percent of max HP taken, so a 55 HP cleric loses its heal about 80% of the
time it is touched. At level 10 the default company's cleric finishes 2.5 of
5 heals and heals 13% of the damage taken (35b measurements).

New rule: a **one-round heal chant** (`waitrounds: 1`, today Minor Heal) is
broken only by **heavy force** (a landed crit, a stagger, a knockdown, a
stun), never by an ordinary blow. Attack spells and two-round chants (Minor
Heal All) keep today's break chance. The rule is symmetric: enemy healers
get it too, which is why the Veteran tier's `BreakHeals` focus (switch aim
to a chanting healer) stays and now matters. `help interrupts` and
`help heal` change.

Why not a lower break chance: a 25% break is still a dice roll a whole class
is built around that the player cannot influence. Resolving the heal makes
"bring a cleric, and whom does it stand behind" a real setup decision.

### 3. Patch to full strength, fight on mana

The company heals in battle below the 50% threshold (30c1) and, since 35b,
patches to the **same** threshold after it. A company entering every fight
at half health loses its fourth fight more often than not (mana run: wins
100 / 89 / 65 / 45%), so the real loop is three fights, then camp. Rest is a
wall, not a decision.

New rule: tactics gain a second threshold, **patch**, defaulting to **80%**
of the wound limit (`company tactics patch <50-100>`). After-battle patching
and `company patch` heal to it; in-battle healing keeps the 50% default.
The mana target becomes **five to six** band-middle fights before the
healer is dry (the mana run asserts the cleric above its reserve after
fight 4 and under 25% by fight 6). Pools (`ManaBase`, `ManaPerLevel`) and
Minor Heal's cost (3) are tuned only if the run cannot reach five fights
with the heal landing as decision 2 makes it.

### 4. The company grows smarter with level

Enemy groups gain focus, guards and casters-first targeting as their level
rises (33i2). The company gains nothing to match, which is why members fall
far more often at bands 18+ than at 8 to 10 in level-flat cells. Mirror
the ladder on the company, as defaults a player can override:

| Leader level | Company default |
|---|---|
| any | a warrior with no ward guards the company's healer when there is one (the harness's "tactics" mode becomes the default) |
| 10 | shared aim: focus **weakest** when tactics are unset (today each member picks its own weakest) |
| 25 | **casters** joins the default focus choices as "casters first": casters while any stand, then weakest; guard slots already rise with level (`MaxGuardsFor`) |

A set tactic always wins over the default. `company tactics` shows which
defaults are in force. In Ogre Battle the army got smarter, not only
stronger; levels should show in the setup layer here too.

### 5. HP keeps pace after level 20

HP grows 0.2 a level after `HPFullLevels` (20) while Strength damage and
Speed tempo keep growing, so same-level fights at 30 are deadlier than at 10
(the mirror shortens from 15 to 11 rounds and members fall more often).
Raise `HPAfterFull` from 0.2 to **0.4** so a solid blow stays about a fifth
of HP at 30 and 60 (35a2 row 1 at those levels decides the exact value; 0.3
to 0.5 is the range). The alternative, holding the damage bonus after 20,
is kept as the fallback if HP alone overshoots the 1.6× level-60 cap.

### 6. Bosses are short and scary

A boss fight today is a full 5v5 at the band's top where one foe has 2.5× HP
and +2 levels and the whole group runs the band's coordination tier. It
lasts about 30 rounds and the win rate collapses where tier 2 and 3 focus
fire begins (38 to 46% at bands 18+ against 60 to 74% below). The owner's
encounter contract (2026-10-05) records "bosses of up to 5 with **no
strategy**".

New shape, in the harness now and in 37's boss rolls: the boss group runs
at **Rabble** (noise 30, no focus), the boss has **1.75× HP** and +2 levels,
and brings **two or three escorts at the band's low level**. Target 70 to
85% wins in 12 to 18 rounds. What makes a boss frightening is a thing it
does that the setup can answer, a telegraphed wind-up the guardian absorbs
or a stagger that punishes the front line; those are 38b's abilities and
this phase only leaves room for them.

### 7. Targets that describe what the player sees

- **Rounds become seconds and lines.** The harness reports median seconds
  (rounds × `RoundSeconds`) and lines per fight beside rounds. Targets: a
  band-middle fight 25 to 40 seconds; the equal 5v5 mirror is a stress
  check only, as the level impact design already says, reported and not
  asserted.
- **Four foes come from the band's low level**, not its top (the cell was a
  near-even 4v5). Target stays ≥90% wins and ≤1 fallen.
- **The under-levelled row** moves from 3 to **5** levels under the band's
  low end (edge 0.36), reported for 37, target 40 to 70% wins. Three levels
  is a nudge by the shipped skill curve and should stay one; danger for an
  under-levelled company comes from which zone it walks into. `SkillEdgeSpan`
  stays 14.
- **Tier probes:** a level-10 company with the default tactics (decision 4)
  wins ≥50% against tier 2 and ≥33% against tier 3; the "no tactics" cells
  stay as a report.

## Persistence and invariants

- The patch threshold and the default-tactics overrides save with the
  company's tactics (30c) and survive restart and copyover like the focus and
  healing threshold do.
- No world time changes. Patching, guards and heals are per-character.
- Durable state is unchanged in shape except the one new tactics field.
- Enemies and companions follow the same combat rules; nothing is
  player-only except the tactics defaults.

## Integration points

- `internal/combat/calculations.go` and `combat.go` (`calculateCombatPower`):
  the quality roll after `activeDefense`, the message tier and suffix;
  `expectedDPS` and `simulate.go` read the same helper so `consider` and the
  33i1 assessment stay honest.
- `internal/interrupt` and `internal/hooks/combat_interrupt.go`: the
  one-round heal exemption, keyed on the chant's `waitrounds` and the spell's
  heal use.
- `internal/strategy/tactics.go`, `modules/company/tactics.go` and
  `modules/company/wounds.go` (patch mode): the patch threshold.
- `internal/strategy/strategy.go` (`Default`, `Resolve`),
  `internal/hooks/combat_strategy.go` and `combat_coordination.go`: the
  level-keyed company defaults and the casters-first focus.
- `internal/configs/config.progression.go` and `_datafiles/config.yaml`:
  `HPAfterFull`.
- `modules/company/balance_test.go`: boss and 4-foe cells, the under row,
  seconds and lines columns; `balance_mana_test.go`: six fights.
- Phase 37's encounter design inherits the boss shape and the 4-foe level
  rule; the 40e combat event messages carry the quality word.

## Help and tutorial

- Update `help attack` (qualities), `help evasion`, `help combat`,
  `help interrupts`, `help heal`, `help tactics` (patch threshold, level
  defaults, casters first), `help patch`, `help guardian`, `help health`
  (late HP rate), `help combatpace` (seconds per fight).
- Tutorial: the combat lesson gains a hint on glancing and telling blows;
  Departure mentions `company tactics patch`.
- `keywords.yaml` aliases: `glancing`, `telling blow`, `patch threshold`,
  `casters first`.
- Render tests in the `help_combat_test.go` pattern;
  `TestTutorialHelpPointersExist` passes.

## Acceptance tests

With `ASHVEIL_BALANCE=1` at 100 fights a cell:

1. **Blow qualities:** between equal unarmored warriors at levels 1, 10, 30
   and 60, a solid blow lands within 15 to 25% of max HP on average, glancing
   8 to 13%, telling 21 to 35%; with a full edge the attacker's telling
   share is at least 45% and with a full edge against it at most 8%.
2. **Fewer dead lines:** at level 10, at most 30% of swings end with no
   damage (35b: about 45%).
3. **Fight length:** band-middle 2 to 3 foes median 6 to 9 rounds and 25 to
   40 seconds; the mirror reported only.
4. **Heals resolve:** a level-10 default company's cleric finishes at least
   75% of the heals it begins (35b: 50%), and heavy force still breaks one
   in a seeded test.
5. **Mana run:** six band-middle fights; wins ≥80% through fight 5; the
   cleric above its reserve after fight 4 and under 25% after fight 6.
6. **Company ladder:** a level-10 company with default tactics wins ≥50%
   against tier 2 and ≥33% against tier 3; the warrior guards the healer
   when no ward is set, and a set ward or focus overrides the default.
7. **Zone bands:** middle rows ≥97% wins and nobody fallen in ≥85% at every
   band including 18 to 20 and 28 to 30; 4 foes at band low ≥90% and ≤1
   fallen; boss 70 to 85% in 12 to 18 rounds; 5-under reported at 40 to
   70%.
8. **Kept rows:** skill wins (L20 vs 3×L10 ≥99%; L10 vs 2×L20 ≤30%),
   warriors the tanks (≤0.7× a rogue's damage per swing), level-60 HP at
   most 1.6× level 1, sides stay even, statuses land.
9. **Unit and wiring:** the quality roll through a real `AttackPlayerVsMob`
   pass with seeded dice; the simulator matching; the heal exemption through
   a real chant; the patch threshold through `company patch` and the
   battle-end hook; defaults by leader level through a real battle; save
   and reload of the new tactics field.
10. **Help:** every page above renders; the tutorial pointers resolve.

Every number is a starting value; the harness decides the final ones within
the timebox rule (owner, 2026-10-05: settle for the best achieved and record
it).

## Dependencies and ordering

- Depends on **35b** (merged). Nothing else.
- Must merge **before 37**, which tunes encounters, bosses and the 4-foe
  rule against these numbers, and before **38b**, whose talents and
  abilities build on the quality model and the boss shape.
- Runs in parallel with **36a** (loot model, no combat overlap) and **38a**
  (the Witch). 38a's hexes are attack spells and statuses: decision 2 leaves
  attack spells breakable and decision 1 leaves spells alone, so the overlap
  is merge-order in `internal/hooks` combat files and `PROJECT_STATUS.md`,
  not design. Whichever of 35d and 38a merges second merges master first.

## Open questions for the owner

1. Decision 1 widens 35a2's "15 to 25% per landed blow" to three bands. Is a
   glancing blow at about a tenth of HP acceptable as the common weak result?
2. Decision 2 makes a one-round heal unbreakable by ordinary blows on both
   sides. Should enemy healers instead keep today's break chance, making
   the rule player-only?
3. Decision 4 makes "guard the healer" and "focus weakest" defaults by
   level. Should they instead be unlocked options the player must set, so the
   setup layer stays a choice?
4. Decision 6 removes strategy from boss groups as the contract says. Keep
   that, or let bosses at bands 25+ run one tier below the band?
