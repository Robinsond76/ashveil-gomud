# Environment, Utility Skills, Loot, Trade, and Alignment — Roadmap (Phases 13–21)

Status: roadmap agreed with the user on 2026-09-23. Each phase below still
gets its own design doc and plan under `docs/superpowers/{specs,plans}/`
before any code, per `CLAUDE.md`.

## Source request (summarised)

- Weather, day/night, moon cycle, fog.
- Temperature with clothing needs: cold without warmth hurts; heavy clothing
  in heat drains water faster.
- Visibility from clouds, fog, night, indoors/caves; a light source is needed
  in the dark, otherwise combat penalties. Indoors you can't read the weather
  unless the room has an exit to the outside.
- Every character has their own light source; a mage can conjure a floating
  light for the whole party.
- Archetype utility skills (wizard light, rogue locks/traps, cook, ...),
  auto-triggered when needed and usable manually.
- Commands to inspect all of the above.
- Walking causes fatigue; rest at an inn (bonus) or a camp to recover.
- Rich loot: humanoids drop gear/food/ingredients/tools; beasts drop fur,
  horns, etc. as trade goods.
- Bannerlord-style commodities: settlements produce/consume goods, prices
  float with stock, players trade across the map, trade rumours hint at
  where goods are cheap or wanted, bandits drop commodities.
- A 1–100 alignment (evil to good), visible when inspecting a recruit, with
  company members influencing each other's alignment (Ogre Battle-style).

## Decisions (user, 2026-09-23)

1. **Light.** Carried light (torch, *lantern*) is **personal**, lighting only
   its bearer. A **room fixture** (campfire, hearth, street lanterns, a
   lit biome) lights the room for everyone. A mage's floating light lights
   the caster's whole company/party.
2. **Alignment affects:** (1) loyalty/morale and desertion when a member is far
   from the company average; (2) recruitment gates (a good paladin won't join
   an evil company); (4) settlement standing (prices, inn access, black-market
   access). Rejected for now: archetype gates, alignment-driven encounters.
3. **Archetypes are exclusive.** A player picks exactly one; a warrior can't
   learn another archetype's skills. Professions (cooking, skinning, mending)
   remain open to all.
4. **Temperature can kill**, but only under extreme exposure after a
   sustained stretch of escalating penalties (e.g. naked in a winter storm:
   debilitated fast, dead soon after). Mild mismatch only penalises.

## Engine facts this builds on

| Area | Existing | Where |
|---|---|---|
| Day/night | `gametime.IsNight()`, dusk, `moon_count` config (no phases) | `internal/gametime` |
| Weather | Phase 8 round-driven zone weather, biome tables; multipliers informational (Option A) | `internal/weather`, `modules/weather` |
| Visibility | `Room.GetVisibility()` 0–2: night, dark/lit biome, mutator `LightMod`, any `lightsource` buff lights whole room | `internal/rooms/rooms.go` |
| Survival | per-member hunger/thirst/fatigue 0–100, explicit exertion | `internal/survival`, `modules/survival` |
| Rest | Phase 7 camping sessions | `internal/camping`, `modules/camping` |
| Skills | trained skills + data-driven professions; `picklock`, `search`, `track`, `peep`, skulduggery; trapped locks | `internal/skills`, `internal/usercommands/skill.*` |
| Loot | per-mob item list + `ItemDropChance` | `internal/mobs` |
| Shops | fixed item value, stock, restock | `internal/characters/shop.go` |
| Mounts | real cargo-capacity bonus | `modules/mount` |
| Alignment | −100..100, named bands, kill shifts, decay, mob alignment hatred | `internal/characters/alignment.go` |

## Phases

| Phase | Scope | Notes |
|---|---|---|
| 13 | Sky and environment: moon phases, cloud cover, fog, indoor rooms, expanded `weather` command | Display + data only; no mechanical effects yet |
| 14 | Visibility and light: per-viewer visibility (sun/moon/cloud/fog/indoor), personal vs fixture vs party light, darkness combat penalty, `light` command | Implements decision 1 |
| 15 | Temperature and clothing: biome base temp + weather + night, item `Warmth`, exposure bands feeding survival (cold → fatigue, heat+clothing → thirst), escalating exposure to death at extremes; switch on deferred weather multipliers | Implements decision 4 |
| 16 | Walking fatigue and inns: per-step exertion (terrain, load, cold; mounts reduce), `inn` room tag + paid rest reusing camping, *Well Rested* buff | Real-time sessions; never advances world clock |
| 17 | Archetypes and utility skills: exclusive archetype per character (players and companions), skill gating, auto-trigger toggles (`autoskill`), company "best member" skill resolution, wizard light, rogue sense/disarm trap, cooking profession | Implements decision 3 |
| 18 | Loot tables: weighted, data-driven, by mob category (humanoid/bandit/beast); beast parts as goods | Reuses the 12b weighted-table pattern |
| 19 | Commodities and markets: commodity items, per-settlement production/consumption, stock-driven prices, round-driven drift, `market` command | Persisted, recovers on restart |
| 20 | Trade rumours: `rumors` at taverns/innkeepers, stale/fuzzy hints | Builds on 19 |
| 21 | Alignment: companion alignment, 1–100 display, Ogre Battle-style drift toward company average, loyalty/desertion, recruit gates, settlement standing | Implements decision 2 |

## Invariants for every phase

- Never advance or fast-forward the shared world clock/round count; derive
  moon/temperature/visibility from the clock, schedule anything periodic on
  `events.NewRound`, and keep travel/rest as real-time sessions.
- All new durable state (markets, exposure, alignment drift bookkeeping,
  archetype choice) must survive restart/copyover.
- Data-driven balance: new numbers live in module config overlays.
