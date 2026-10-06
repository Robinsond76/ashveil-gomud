# Phase 39c: the Shaman

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md).
A starting lineage with no alignment gate that calls a battle-local weather.

## What shipped

- **Lineage and data**: the `shaman` archetype (HP 2 and 0.6 a level, Attack
  0.7, Evasion 0.8, light armor, no shield, mana 40 + 10 a level, the same
  growth kit as the Witch), recruit mob 150 in the company templates, creation
  text, a battle-screen hue. Spells: Call Fog and Gust at level 1, Chill Wind
  at 3, Rain at 6, Lightning at 8.
- **Battle weather** (`internal/stormcraft`, `internal/battle/weather.go`):
  one weather at a time per battle, three rounds (plus the Long Weather talent
  and the Mistweaver's longer calls), never `internal/climate`. A new call
  replaces the old one and removes its marks. Fog puts **Fogbound** on foes
  (ranged attacks -10 to hit, spells 10% weaker); Chill Wind puts
  **Windchilled** on them (chants and sling shots take one more round, as in
  the cold); Rain is battle state that Lightning reads (+50% damage).
  Weather reaches only the foes standing when it is called.
- **Routes** (open at any alignment, levels 10 and 30): **Stormcaller** (chain
  lightning to a second foe at 50%), **Mistweaver** (+5 Evasion to allies while
  its Fog lasts, longer weather), **Earthspeaker** (Stoneskin: +10 armor for
  the battle on an ally, 15 and 20 later). Tempest Lord, Veil Mother and
  Mountain Speaker are planned elites (39i). A Long Weather talent joins the
  Shaman's offers.
- **Strategy**: the Shaman is a `Caster`. New uses `weather` and `storm`:
  weather first when none is up (Rain only once it knows Lightning; fog and a
  chill only when a foe shoots or casts), then Stoneskin on an ally, Lightning,
  then Gust.
- **UI**: `help shaman`, `help shaman-routes` (aliases for every spell, route
  and status), updates to the archetype, health, growth, armor, shields,
  evasion, classes, promotion, talents, combat, progression, specialists,
  forecast, weather, mana, strategy and statuses pages, a creation-lesson
  hint in the tutorial, and the `New rank` lines for each route rank.

## Decisions

- **Role**: `Caster`, not the design's new `support` role (a role would touch
  every strategy surface; the weather use is a new Use inside Caster).
- **Statuses for Fog and Chill** are real buffs counted in combat rounds, so
  they show in `conditions` and the battle events with no new display code.
- **Fire halving in Rain** exists as `stormcraft.FireDamage` and in the help,
  but nothing deals fire damage with a spell yet, so it has no effect today.
- **Fog's -10 to hit** applies to ranged weapon attacks only; spells take the
  10% cut through the spell factor.
- **Mistweaver Veil of mist** gives +5 Evasion to every ally, not only
  melee ones, while fog is up.
- **Elites** (30+) stay planned for 39i; art deferred to the art pass.

## Balance

Even mirror (a five-foe group of the company's own level, 20 fights a cell,
±20 points): Shaman 50 / 75 / 35% at levels 5 / 10 / 20 against Wizard
30 / 40 / 35% and Witch 25 / 40 / 20%. Routes at 15 and 25 are 20-45% against
a base 35 and 15%. The Shaman sits at or above the other casters.
