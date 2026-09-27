# Potential Phase 29a: Status Effects and Critical-Hit Effects

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.
This is handoff §36 item 7, formerly in deferred Phase 11d.

## Goal

A critical hit does more than add damage. Its secondary effect matches the
weapon: swords make you bleed, maces stagger, hammers knock down. The
effects are ordinary timed statuses that other systems (interrupts,
wounds, morale) can read.

## Prior-art check (2026-09-26)

- **Buffs:** GoMud's buff system (`internal/buffs`, `_datafiles/.../buffs/`)
  already models timed effects, with flags, round triggers, and scripts.
  Poison is one.
- **Crits:** `internal/combat/combat.go` sets `isCrit` per strike. Its
  `critBuffs` come from the weapon's dice-roll spec (`BuffTarget`).
  - Some weapons can already apply buffs on a crit.
- **Weapon subtypes:** `items.ItemSubType` covers bludgeoning, slashing,
  stabbing, shooting, cleaving, claws, and whipping. These are the same
  keys as the combat message files.

## Scope

1. **New statuses**, as buffs with Ashveil flags:

   | Status | Effect |
   |---|---|
   | **Bleeding** | damage each combat round, stacking in severity |
   | **Staggered** | loses the next action; adds interrupt pressure (29d) |
   | **Knocked down** | loses the next action; can't guard (29c); is easier to hit and to leap over (29f) |
   | **Armor broken** | lowered defense for the fight |
   | **Exposed** | the next hit against it is more likely to crit |
   | **Burning** | fire damage over time (magic) |
   | **Overloaded** | a spell's elemental status |
   | **Stunned** | a shield's or a heavy blow's crit |

2. **A crit effect table by weapon subtype**, in data:

   | Weapon | Crit effect |
   |---|---|
   | slashing (sword) | bleeding |
   | stabbing (dagger) | deep bleeding and wound (29b) |
   | bludgeoning (mace, cudgel, club) | staggered |
   | cleaving and heavy (hammer, great club) | knocked down, or armor broken |
   | shooting (arrow, sling) | exposed; a sling staggers |
   | claws | bleeding |
   | shield | stunned |

   A weapon or mob can override its effect.
3. **Spells do not critical hit.** The owner declined this (2026-09-27).
   Spell damage stays a single roll with no crit tier, no secondary
   effect, and no pain reaction (28e only fires off a weapon crit). The
   `burning` and `overloaded` statuses above are for spells that
   deliberately apply them as part of their normal effect (e.g. a fire
   spell that always burns), not as a crit outcome. `magic` is dropped
   from the crit effect table.
4. **Durations** count combat rounds (see 28f's cadence). Statuses end when
   the fight ends, unless 29b turns them into a wound.
5. **Events and text:** applying, ticking, and expiring each emit an event
   (28b). The status is named in the hit's parentheses
   (`(critical hit, 6 damage, bleeding)`). A bleed tick reads
   `The first skirmisher bleeds. (1 damage, bleeding)`.

## Acceptance criteria

- A forced crit through the real round, for each weapon subtype, applies
  that subtype's status and emits its event and text.
- A bleed ticks each combat round and ends at fight end.
- A knocked-down combatant loses its next action.
