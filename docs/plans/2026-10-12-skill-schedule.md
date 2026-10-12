# Skill schedule: skills come as a character levels (2026-10-12)

Owner's ask (2026-10-12): Tackle should not be a skill a warrior knows at
level 1. Skills are learned as a character levels; each class's skills are
spaced out before its next tier, and keep spacing until the tier after.
Decided under the standing autonomy rule; this note records the schedule and
why.

## The rule

- **Level 1:** a plain blow. A caster knows one battle spell (and any spell
  used only outside a battle); a class whose plain action is something else
  keeps it (a Doll Master's Puppet Strike, an Alchemist's Healing Draught and
  Brew, a Beast Tamer's beast).
- **Levels 2-8 (base tier):** the lineage's skills and spells at 2, 4, 6 and
  8, in their old order. The talent stays at 5.
- **Levels 10-29 (advanced tier):** unchanged. Every route already brings a
  rank at 10, 15, 20 and 25; the witch's later hexes come at 10, 12, 15 and 18.
- **Levels 30-60 (elite tier):** unchanged, a rank every five levels.

Why 2/4/6/8: before this, the neutral lineages had their signature at 1 and
the next skill at 3, and the faith lineages had everything at 1. Moving only
the signature to 2 would have left 2 and 3 back to back. One skill every
other level reads as steady growth, keeps the old order, and moves nothing
later than 8, so no class reaches promotion weaker than before.

## Base tier, per class

| Class | 1 | 2 | 4 | 6 | 8 |
|---|---|---|---|---|---|
| Warrior | plain blow | Tackle (was 1) | | | |
| Rogue | plain blow | Opening Strike (was 1) | | | |
| Ranger | plain blow | Aimed Shot (was 1) | | | |
| Cleric | Minor Heal, Tend | | Minor Heal All (was 3) | Cure Poison | Bless |
| Wizard | Magic Missile, Floating Light | | Shower of Sparks (was 3) | | |
| Witch | Withering Hex | Slumber (was 1) | Earthbind (was 3) | Leaden Curse (was 5) | Miasma (was 7) |
| Shaman | Gust | Call Fog (was 1) | Chill Wind (was 3) | Rain | Lightning |
| Halberdier | plain blow | Sweep (was 1) | Brace (was 3) | Hook | Wide sweep |
| Samurai | plain blow | Iaijutsu (was 1) | Focus (was 3) | | Zanshin |
| Gryphon Rider | plain blow | Dive (was 1) | Talons (was 3) | | Power dive |
| Arbalist | plain shot | Piercing Bolt (was 1) | Steady Aim (was 3) | Crippling Bolt | Armor-breaker |
| Beast Tamer | its beast | Sic (was 1) | Rally (was 3) | | Pack Sense |
| Doll Master | Puppet Strike | | Guard String (was 5) | | Stronger strings |
| Alchemist | Healing Draught, Brew | | Antidote (was 3) | Fire Flask | Bracing Tonic |

The witch keeps Withering Hex at 1 (its weak damage curse, the one it falls
back on) and gains Slumber at 2; the shaman keeps Gust (its damage spell) and
gains Call Fog at 2. The Doll Master's Guard String moves from 5 to 4 to fit
the rhythm; nothing else moves earlier. Creatures (hound, stone golem) are
species, not careers: their level-1 ranks are their bodies and stay.

## How it is built

- `strategy.Abilities[].MinLevel` gates every automatic ability; Tackle,
  Opening Strike, Aimed Shot, Sweep, Dive, Piercing Bolt and Sic gain 2;
  Brace and Rally 4; Guard String 4.
- The warrior, rogue and ranger gain base ranks (`internal/classes/base_martial.go`)
  naming their signature at 2, so the level-up says "New rank: Tackle" and
  `class` says "Next: a rank (Tackle) at level 2". They set no effects.
- The neutral lineages' base ranks move with their abilities. A test
  (`TestAbilityLevelsMatchBaseRanks`) keeps every ability's level equal to its
  base rank's.
- Spells move in the archetype config's `GrantSpells`, `LevelSpells` and
  `CompanionSpells`.

## Existing characters

- A player keeps every spell already learned: owning the spell is the marker
  and nothing takes one away. A level-3 cleric made before this keeps Minor
  Heal All, a level-1 witch keeps Slumber.
- Abilities and ranks are derived from the level, never saved, so a character
  at or above a skill's new level keeps it, and one below waits for it (a
  level-1 warrior tackles again at 2). A companion's spells are derived from
  its level the same way.
- Enemies with an archetype follow the same levels, so level-1 enemy warriors
  don't tackle either.

## Level-1 fights

A timeboxed at-level sim (`TestBalanceAtLevel`, 20 fights a cell, bands 1-1
and 2-3, 5 and 3 members) before and after:

| Band | Company | Wins | HP lost a fight | Fights before a rest (median) |
|---|---|---|---|---|
| 1-1 | martial 5 | 100% → 100% | 3.8% → 3.8% | 8 → 9 |
| 1-1 | martial 3 | 100% → 100% | 6.2% → 8.3% | 5 → 6 |
| 1-1 | magic 5 | 100% → 100% | 5.2% → 5.4% | 4 → 4 |
| 1-1 | magic 3 | 100% → 100% | 21.7% → 19.8% | 2 → 3 |
| 2-3 | martial 5 | 100% → 100% | 3.7% → 4.5% | 10 → 10 |
| 2-3 | martial 3 | 100% → 100% | 6.4% → 5.5% | 8 → 8 |

Level-1 fights stay easy: every company wins, and the differences are within
the sample's noise. (Enemy damage is a separate thread.)

## Help

- New `help skill-levels` (aliases `unlocks`, `skill-schedule`, `learning`),
  indexed under character and linked from `help combat`, `help abilities`,
  `help progression`, `help archetype` and `help strategy`.
- Updated levels in `help abilities`, `halberdier`, `samurai`, `arbalist`,
  `gryphon-rider`, `beasttamer`, `dollmaster`, `alchemist`, `witch`, `hexes`,
  `shaman`, `spells`, `strategy`, `warrior`, `ranger` and `combat`;
  `help progression`'s stale milestones replaced.
- Tutorial: the progression and abilities hints point at `help skill-levels`.

## Left alone

- A samurai player has the brawling skill, so it tackles like a warrior
  (a samurai companion does not). Pre-existing; a follow-up if wanted.
- The web client's capability list shows an ability not yet reached as
  disabled with "Comes at level N", as it did for Brace.
