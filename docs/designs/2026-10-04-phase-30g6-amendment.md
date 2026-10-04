# Phase 30g6 amendment: stat edges, a true mirror, and measured tactics

Amends slice 30g6 of the [30g design](2026-09-30-phase-30g-tempo-defense-design.md).
Written 2026-10-04 from a review of the first 30g6 candidate (branch
`phase-30g6-tuning`, commit `5b36efd`); the owner approved every
recommendation ("Yes let's do everything you're suggesting", 2026-10-04).
Decisions 8 and 11–14 are unchanged; this changes how they are met and
measured.

## Why

The candidate met the spread-fight duration and the HP envelope but failed
focus duration at every level, caster targeting at level 30, and the level
15 against level 10 mismatch. A probe of the candidate found three causes
that tuning numbers cannot remove:

1. **Small stats break the opposed formulas.** Since 30g4's stat steps,
   Speed and Strength run about 0–6 from level 1 to 30. Hit, crit chance,
   crit multiplier and block use `a / (a + d)`, so Speed 3 against 2 is
   60% against 40% and one step decides a fight: level 15 hit 50% against
   level 10's 28%, and won 100 of 100. Dodge, parry, bash and the Strength
   advantage use `(a − d) / 100`, so they never leave their minimum. A
   tackle is `Speed − Perception + 20`, pinned at its 20% floor.
2. **The "even" fight was not even.** The company used class abilities and
   a healer against a mirror with neither. Failed tackles (about 2.5 lost
   turns per warrior per fight) and interrupted heals left the company
   winning only 26–51% of spread fights.
3. **The focus check measured the wrong thing.** When the company loses,
   the enemy's damage sets the fight's length; company focus kills enemies
   sooner, which slows their damage and lengthens a lost fight. A median
   over all fights cannot show "wins sooner". The caster-targeting check
   compared means 0.5 HP apart with a strict `>`, a coin flip.

The candidate's HP numbers also drifted from decision 12: class gains
stopped at level 5, then every class gained 0.7 a level, so a level-60
warrior had 71 HP to a default character's 63. Its default HP rate for
enemies was the wizard's, where the design gives enemies the middle
archetype's.

## Decisions

### A. One stat edge for every opposed chance (30g6)

A new `Combat.StatEdgeSpan` (proposed 10, tuned in 30g6) turns a stat
difference into an edge: `edge = clamp((a − d) / StatEdgeSpan, −1, 1)`,
and `advantage = max(0, edge)`. A full span of difference reaches a bound;
one point moves a tenth of the way.

- **Two-sided chances** start from their even value and move toward a
  bound: `even + edge × (max − even)` when ahead, `even + edge × (even −
  min)` when behind.
  - hit: new `ToHitEven` (50, today's equal-stat value) in
    `ToHitMin`–`ToHitMax`;
  - crit chance: new `CritChanceEven` (15, today's) in
    `CritChanceMin`–`CritChanceMax`, before accuracy, blink and exposed;
  - block: the minimum plus the shield's armor, moved by `edge × (max −
    min) / 2` as today's share does, held to the range.
- **One-sided chances** start at the minimum and grow with the advantage:
  `min + advantage × (max − min)` for dodge, parry (then the weapon's
  modifier), bash, the crit multiplier, and the Strength-advantage part of
  the damage bonus.
- **Tackle:** `40 + edge × 40` from Speed against Perception, held to
  20–80, through the same span (`strategy.TackleChance` and the manual
  command share it).
- Equal stats give the same values as today, so only gaps change.
- Every number is config (decision "config, not constants"); `help
  speed`, `smarts`, `perception`, `strength`, `defense` and `abilities`
  describe the edge in words and the bounds in numbers.

### B. Strength damage grows with Strength (30g6)

Keep the candidate's `DamagePerStrength`: the bonus is `DamageBonusMin +
Strength × DamagePerStrength + advantage × (DamageBonusMax −
DamageBonusMin)`, rounded down and capped at `DamageBonusMax`, so `e`
grows with level as decision 8 needs. The advantage term uses decision
A's edge.

### C. The asserted fights are a true mirror (30g6a)

- **Spread and focus** cells turn the company's class abilities off and
  make its cleric a fighter, so the company and the mirror have the same
  kits, stats, HP and options; the tactic is the only difference.
- A **kit** cell (spread, with the shipped abilities and healing) is
  measured against the same mirror; `default` keeps the shipped
  strategies as before.
- **New assertions:** the spread mirror is even (company wins 35–65%),
  and kit is not significantly worse than the mirror (abilities and
  healing are never a handicap).

### D. Tactics are measured as tactics (30g6a)

- **Focus wins sooner** compares mean rounds of **won fights** (focus
  against spread), with the focus win rate still higher over all fights.
- **Tactics claims need significance:** "focus wins sooner" and "enemy
  targeting takes more health" each need a one-sided Welch statistic of
  at least 1.645 (95%) over the cell's fights, not a bare `>` on two
  noisy means. Every fight stays in the sample; nothing is discarded.
- The table gains a won-fight mean-rounds column.

### E. HP keeps class differences (30g6)

- `HPFullLevels` at least 10, so the class rates carry the early levels;
  `HPAfterFull` small enough that a level 60 has 2–3× a level 10's HP.
- **New assertions:** at level 60 a warrior has at least 1.25× a
  wizard's HP with the same Vitality, and `DefaultHPPerLevel` equals the
  middle archetypes' rate (cleric and ranger).

## Slices

- **30g6a — combat fixes and harness** (from the candidate, on master's
  numbers): a successful tackle isn't repeated in the same pass; a
  replacement target respects caster focus; an earned turn survives an
  ally's kill; combat-round statuses never tick on game rounds; the
  harness covers levels 30/60/100, the mismatches, caster targeting, net
  health removed, distinct opening aims and the shipped class rates; C
  and D above. Help: `tempo`, `abilities`, `combat`. The opt-in suite is
  expected to fail acceptance on master's numbers; its table is recorded.
- **30g6 — stat edges and tuning:** A, B and E, then tune the configured
  numbers until the whole opt-in suite passes. Help for every changed
  formula and the tutorial pointer.

## Acceptance (replaces 30g6's assertion list)

With `ASHVEIL_BALANCE=1` and 100 fights a cell:

- spread mirror: median 10–15 rounds at levels 1, 5, 10, 30 and 60
  (reported at 100), and company wins 35–65%;
- kit not significantly worse than the spread mirror;
- focus against passive enemies: wins more often, and its won fights end
  significantly sooner;
- weakest and caster targeting against a passive company: significantly
  more company health removed;
- level 30 against level 10: wins at least 95%, no member lost in most;
- level 15 against level 10: wins more than half, and loses at least once;
- HP: level 60 is 2–3× level 10 for each fighter; warrior at least 1.25×
  wizard at level 60; default rate equals the middle archetypes'.

The always-on suite keeps every harness, wiring and help test green.
