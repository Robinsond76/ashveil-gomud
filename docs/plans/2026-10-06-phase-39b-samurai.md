# Phase 39b: the Samurai

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §6.
A seventh starting lineage with no alignment gate: a duelist that wins a fight
in its first moment.

## What shipped

- **Lineage and data**: the `samurai` archetype (HP 6 and 0.85 a level, Attack
  and Evasion 1.0, medium armor, no shield, swords only, speed 4, strength 3,
  perception 2, smarts 1, Camp Watch, an iron-shortsword kit), a Samurai recruit
  (mob 139) in the company templates and recruiter mix, creation text, a
  battle-screen hue.
- **Base ranks** (`internal/classes/base.go`): a lineage's own ranks, from
  level 1, applied before a route's: **Iaijutsu** (1), **Focus** (3),
  **Zanshin** (8). Effects are read by key like every class effect, derived
  from the level and never saved.
- **Routes** (all open at any alignment, level 10 and 30): **Kensai** (Piercing
  draw: the first strike ignores half the armor, Deep focus, Opening edge),
  **Hatamoto** (Bodyguard: steps in for the company leader twice a battle, three
  at 20; row Evasion and damage cuts), **Ronin** (Vengeance: +10% damage per
  fallen ally, 15% at 20). Sword Saint, Shogun and Kenshi are registered as
  planned elites (the elite pass, 39i). Five talents: Keen Edge, Footwork, Heavy
  Hands, Toughness, Sharp Eye (+3% critical chance).
- **Combat hooks**: the first strike (`internal/combat`, `IaiReady`): +50%
  damage and +10% critical chance on a battle's first swing, spent by that swing
  whether it lands or not (a wind-up is not one); `critsWith` adds class
  critical points (Sharp Eye, Focus) to every blow; the opening meter
  (`combat.Meter.Bonus`) starts half a turn up; Focus counts quiet rounds and a
  landed blow resets it, Zanshin tops the meter up when it fells a foe (once a
  round), Vengeance reads the side's peak against its standing members, and
  Bodyguard runs ahead of the strategy guards in `guardianFor` from its own
  per-battle count (`internal/hooks/combat_samurai.go`). All of it is runtime
  battle state on `ClassRT`, cleared by `EndFightRT`.
- **Strategy**: `strategy.DefaultRule("samurai")` is `strongest`.
- **UI**: `class` lists base ranks and level-ups name the next one
  (`classes.MilestoneFor`); `help samurai`, `help samurai-routes` (aliases for
  every class, rank and route name), updates to the archetype, classes,
  promotion, talents, strategy, tempo, guardian, combat, health, growth, armor,
  shields, evasion, progression, specialists and campwatch pages, and a
  creation-lesson hint in the tutorial.

## Decisions

- **Duelist role**: a fighter whose default rule is `strongest`, not a new role
  (the design names a `duelist` role; it would touch every strategy surface for
  a rule change).
- **First strike is spent by the swing**, hit or miss: it is "the first
  strike", and a miss is the price of speed.
- **Bodyguard** takes priority over a strategy guardian for blows on the leader
  and has its own count (it does not spend the strategy guards).
- **Vengeance** counts a side's fallen as its peak standing less its standing
  now, so no company roll-call is needed and a dead companion that despawns
  still counts; only a Ronin pays for the bookkeeping.
- **Camp Watch** (the Warrior's utility, from `brawling`) is the Samurai's
  utility, so the lineage has a camp role.
- **Elites** (ranks 30-60) stay planned with the other lineages' (39i).
- **Weapons**: the class rule is `sword`, which the three tiered swords carry;
  the guardsman's and captain's broadswords have no weapon class and are not
  Samurai weapons.
- Focus is 3% a quiet round to +9% (the design's 5% to +15% was too strong
  in the harness) and Attack rate 1.0 (the design's 1.1); see Balance.

## Balance

Harness: `TestPhase39bSamurai` (`ASHVEIL_BALANCE=1`), slot 3 swapped, 80 fights
per cell (about ±5 points of noise), final settings Focus 3%/+9%, Attack 1.0.

| Level | Warrior | Rogue | Samurai |
| --- | --- | --- | --- |
| 5 | 78% | 76% | 83% |
| 10 | 91% | 92% | 91% |
| 20 | 63% | 76% | 77% |

Samurai at L15 and L25: 87% and 81%. The design asks for a win rate within 5
points of the Rogue company: L10 and L20 meet it; L5 is 7 points above, which is
inside the noise at this sample size. The harness gives every companion the same
light gear, so it understates the Samurai's medium armor. The design's first
values (Focus 5%/+15%, Attack 1.1) ran 7-20 points above the Rogue and were
lowered. Route cells (Kensai, Hatamoto, Ronin) are in the test but were not
tuned further (timeboxed).
