# Phase 38e: Creature recruits (Hound and Stone Golem pilot)

Source: the [expanded class reference](../designs/2026-10-01-class-reference-extractions.md)
("Additional creature families": creatures use their own authored upkeep,
equipment and recovery, and are not human class promotions) and the
[roadmap](2026-10-06-remaining-roadmap.md). The roadmap listed 38d and 39e as
prerequisites; neither is needed by the pilot (38d is catalogue bundles, 39e
the Beast Tamer, which will later reuse this phase's creature family rules), so
38e was built on master as the owner's full-autonomy rule allows. Every
decision below was taken under that rule; none waited on an answer.

## Decisions

1. **A creature is a species, not a career.** It has a base lineage (ranks by
   level, `internal/classes/routes_creature.go`) but no promotion route and no
   talents (`classes.MilestoneFor` and the `class` view skip both). Warhound
   and Runic Golem upgrade forms are deferred to a follow-up.
2. **Two kinds of upkeep** (`internal/creatures`, GoMud-free):
   - **Hound, biological.** Eats, drinks, tires, mends on the idle beat, in
     camp and at an inn, takes lasting wounds, and can drift or lose heart like
     any companion.
   - **Stone Golem, construct.** Needs no food, drink or rest (survival
     `MemberRef.Needless`, so provisioning, drain, recovery and the inn bed all
     skip it). Never drifts, loses heart or deserts (it is *bound*; shown as
     "bound" in `company status`). No idle regen, no camp or inn recovery.
     Mends only with **stone mortar** (item 290): `company repair [golem]`,
     each mortar restores 50% of its maximum health, spent from cargo, then
     member packs, then the leader's pack; refused in a fight or battle, when
     whole, when away or fallen. Healing spells still heal it. It takes no
     lasting wound (`wounds: none` on the mob).
3. **Creatures carry nothing.** No starter pack, no share of the company's
   load (`CompanionCarry`), no horse in the herd count.
4. **Equipment.** A creature wears only gear cut for its species (item
   `wornby`), and nobody else may wear creature gear
   (`creatures.CanWear`, enforced in `characters.CanWield`, so every equip
   route agrees). The Hound's spiked collar (20026, now `wornby: [hound]`) and a
   new hound harness (20500) are the only gear; the golem wears nothing.
   Race `disabledslots` back this up. A creature gear catalogue is a follow-up.
5. **Not a player's choice.** Creature archetypes (`hound`, `stone-golem`) are
   hidden from creation, `archetype list` and `archetype choose`, and
   `company archetype` refuses to change one. They claim one skill,
   `instinct`, that no player can train.
6. **New class effects**, all read where the blow, tempo and spell maths live:
   `Pounce` (+N% blow damage to a foe that is exposed, knocked down or
   hobbled), `Slow` (N% fewer turns, on top of the Tempo floor), `SpellWeak`
   (spells hit N% harder).

| Species | Health | Atk / Eva | Ranks |
|---|---|---|---|
| Hound (90 gold, Trappers' Post) | 4, 0.8/level | 1.0 / 1.1 | 1 Run down (Pounce 30), 5 Worry (+25% vs a foe at or below half health), 10 Fleet (+3 Evasion), 20 Savage pursuit (Pounce 50) |
| Stone Golem (180 gold, Waymark Inn) | 12, 1.3/level | 0.9 / 0.6 | 1 Stone body (+30 armor, Slow 25, SpellWeak 25), 1 Anchor (row takes 10% less), 10 Granite (armor 40 in all), 20 Bedrock (row 15% in all) |

Default target rules: Hound `wounded`, Stone Golem `nearest`. Mobs 260
(Brindle) and 261 (Cairn); recruiters in Dunmar; Dunmar's market sells mortar
(14 gold, supply only, so it can never be resold for profit).

## UI and player help

`company recruit` lists a "Creature:" line for a creature candidate;
`company inspect` and the `train`/`class` views name the family; `company
status` shows "Hound (creature)" and "Stone Golem (construct), bound"; the
survival status says "needs no food, drink or rest". The battle screen draws
the species (code-drawn `hound` and `stone-golem` units; the screen already
keys a companion's sprite by archetype). Help: `creatures`, `hound`,
`stone-golem` and `repair`, indexed under character with aliases, linked from
`classes`, `company` and `equipment`; a Departure-lesson tutorial hint.

## Acceptance

- Each rank fires in a real round (Pounce and Slow through `classBlowDamage` and
  `Tempo`; SpellWeak through `SpellFactor`; the live company mobs read the
  ranks).
- The golem takes no food, rest, bed, drift or morale, and repair works end to
  end through `company repair`; the hound keeps all of them.
- Creatures take no pack and add no carry; gear refuses both ways.
- Help pages render, are indexed and linked, and `TestTutorialHelpPointersExist`
  passes.

## Follow-ups (candidate phases)

- Warhound and Runic Golem forms (a creature's one upgrade), and a Hellhound or
  Iron Golem tier.
- A creature gear catalogue (barding, runed plates) and creature recruits for
  other biomes.
- Beast Tamer (39e) will reuse `internal/creatures`: tamed beasts are
  biological creatures that need their own rules for taming and bonding.
- Map sprites for the species (the battle unit art ships here; the map shows
  its fallback).
- Whether a healing spell should heal a construct at a reduced rate.
