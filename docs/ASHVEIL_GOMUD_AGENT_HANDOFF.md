# Ashveil on GoMud — AI Agent Handoff & Implementation Context

**Status:** Architecture / migration planning
**Primary goal:** Rebuild Ashveil as a multiplayer MUD on top of GoMud while preserving Ashveil's distinctive expedition, survival, mercenary-party, camping, mount, and 3x3 tactical-formation systems.
**Intended reader:** An AI coding agent that will inspect the repositories, fork GoMud, produce an implementation plan, and begin the migration carefully.

---

## 0. Read This First

This document is the source of truth for the intended direction of the project at the start of the GoMud migration.

Do **not** treat the current Python Ashveil implementation as disposable. It is the prototype and design reference for the mechanics that make Ashveil distinct.

Do **not** begin by rewriting GoMud systems wholesale.

The intended architecture is:

> **GoMud provides the engine foundation and infrastructure. This fork is becoming the Ashveil game: its game rules, expeditions, survival systems, mercenary companies, tactical formation combat, world content, and identity belong in its permanent codebase.**

Use durable game-domain names such as `company`, `companion`, and `expedition`. Do not treat gameplay code as a temporary `ashveil*` layer or prefix packages/types with the project name merely to distinguish them from GoMud.

The project should remain a **multiplayer MUD with a shared persistent world**.

A critical design correction from earlier planning:

> **Travel must never fast-forward or locally advance world time.**
>
> When a party begins a journey, it enters a server-side real-time `TravelSession` and waits for the configured travel duration to elapse. The shared GoMud world clock/day-night cycle continues normally for every player. During travel, the party may be interrupted by random events, encounters, discoveries, weather effects, or other route events.

---

# 1. Repositories

## Existing Ashveil prototype

Repository:

`https://github.com/Robinsond76/ashveil-mud`

Language:

- Python

Purpose during migration:

- gameplay prototype
- mechanic reference
- behavior reference
- design archive
- source for Ashveil-specific concepts that need to be ported

The existing repository should **not be overwritten by the GoMud fork**.

Recommended initial strategy:

- keep `ashveil-mud` intact
- create a separate GoMud fork, provisionally named `ashveil-gomud`
- only rename/reorganize repositories later after the Go implementation is established

## Engine to adopt

Repository:

`https://github.com/GoMudEngine/GoMud`

Language:

- Go

GoMud is a mature multiplayer MUD engine with an existing playable world and substantial infrastructure.

As of the migration-planning review, GoMud publicly documents or demonstrates features including:

- multiplayer MUD server
- Telnet access
- browser/websocket client
- web admin tools
- web map editor
- rooms and exits
- in-game maps
- mobs/NPCs
- inventory
- equipment
- shops
- quests
- combat
- day/night cycle
- room scripting
- mob scripting
- hired mercenaries
- parties containing players and NPCs
- loot/experience sharing
- alternate characters
- persistent world/game infrastructure
- optional compiled modules that can add gameplay, commands, events, and other features

Current player-facing GoMud movement documentation explicitly lists:

- north
- south
- east
- west
- up
- down

Do **not** assume northeast/northwest/southeast/southwest are natively supported everywhere until the code is inspected.

---

# 2. Product Vision

Ashveil should feel like a multiplayer fantasy MUD blended with elements inspired by:

- Mount & Blade: Bannerlord — expedition logistics and party management
- Ogre Battle — tactical formation positioning
- classic MUDs — persistent shared multiplayer world, detailed rooms, text commands, exploration, social play
- light survival RPGs — food, water, fatigue, camping, weather, carrying capacity, mounts

The core fantasy is not simply:

> Walk through rooms and kill mobs.

It is:

> Recruit and manage a small mercenary company, equip them, arrange them tactically, prepare supplies, undertake dangerous journeys through a persistent world, camp in the wilderness, manage hunger/thirst/fatigue/weight/mounts, survive random encounters, and arrive at meaningful destinations.

The party should remain intentionally small.

**Target maximum party size: 5 total characters**, normally:

- 1 player leader
- up to 4 mercenaries

The small party is important because every mercenary should feel individually meaningful.

---

# 3. Core Design Pillars

## 3.1 Multiplayer first

Ashveil is a real multiplayer MUD.

The world clock is global.

One player's travel cannot cause:

- the world to skip forward
- sunrise for every other player
- shops to jump schedules
- other players' buffs or timers to change
- global weather timers to skip
- NPC schedules to jump ahead

Any system that previously advanced simulation time locally must be redesigned for shared multiplayer.

## 3.2 Rooms represent interesting places

Primary world-design rule:

> **Rooms represent interesting places. Distance represents travel.**

Avoid creating thousands of filler wilderness rooms whose only purpose is to simulate miles.

A room should generally exist because something about that location is worth describing, exploring, interacting with, fighting in, camping at, discovering, or remembering.

Examples:

- Western Gate of Dunmar
- Fork at the Black Oak
- Ruined Tollhouse
- Saint Arel's Bridge
- Abandoned Shrine
- Blackwood Trailhead
- Greywatch South Gate

Not:

- Forest Tile 1827
- Forest Tile 1828
- Forest Tile 1829

## 3.3 Expedition logistics must matter

The following should influence whether a journey is safe or efficient:

- food
- water
- party fatigue
- individual fatigue
- party load / encumbrance
- equipped armor weight
- carried loot
- terrain
- route quality
- weather
- injuries
- mounts
- mount condition/fatigue
- possibly temperature and shelter later

The goal is not tedious micromanagement. The goal is meaningful preparation and tradeoffs.

## 3.4 Formation matters

The Ashveil 3x3 party grid should remain a major identity feature.

The grid represents **combat/tactical formation**, not geographic world position.

Example:

```text
               ENEMY

           [ ][T][ ]
           [ ][S][ ]
           [M][P][A]

               PARTY
```

Potential meanings:

- `T` tank
- `S` spear/polearm
- `M` mage
- `P` player
- `A` archer

A maximum of five party members occupy five of the nine cells.

Formation should eventually influence:

- melee reach
- interception
- shielding
- ranged safety
- polearm reach
- adjacency buffs
- row/column attacks
- area attacks
- flanking/exposure
- targeting priority
- movement skills

Do not implement all of these at once.

---

# 4. World Structure

Recommended hierarchy:

```text
World
└── Region
    └── Area
        └── Room
```

Example:

```text
WORLD: Ashveil

REGION: Western Marches

AREA: Dunmar
├── Western Gate
├── Central Market
├── Raven Tavern
├── Blacksmith
└── Castle Ward

AREA: Blackwood
├── Eastern Trailhead
├── Fork at the Black Oak
├── Abandoned Shrine
└── Saint Arel's Bridge

AREA: Greywatch
├── Southern Gate
├── Market
├── Inn
└── Keep
```

GoMud's existing concept of rooms/areas should be reused wherever possible rather than replaced.

The agent must inspect how GoMud currently represents:

- areas/zones
- rooms
- exits
- map coordinates
- room metadata
- room scripting
- pathfinding/mapping

before proposing new abstractions.

---

# 5. Local Movement vs Journey Travel

This distinction is fundamental.

## 5.1 Local movement

Examples:

- tavern -> market
- market -> town gate
- one dungeon chamber -> next chamber
- one street -> another street

This should remain normal GoMud room movement.

Typical characteristics:

- effectively immediate
- no meaningful food consumption
- little or no survival cost
- ordinary room exit
- standard MUD directional command

Example:

```text
north
```

moves the player normally.

## 5.2 Journey travel

Examples:

- Dunmar -> Greywatch
- town -> distant forest
- crossing a mountain pass
- traveling several miles down a road
- moving between major landmarks or areas

These transitions represent meaningful geographic distance.

They should initiate an asynchronous **TravelSession**.

Example:

```text
> north

You lead the party onto the Old King's Road.
Estimated travel time: 2m 40s.
Destination: Fork at the Black Oak.
Terrain: Road / forest edge.
```

The party is now traveling.

The player does **not** immediately appear at the destination.

The server does **not** advance global time.

The travel completes after the configured real-time duration unless interrupted.

---

# 6. Multiplayer Travel Model

This supersedes any earlier design based on "advance game time by N hours."

## 6.1 Core rule

Travel consumes **real server time** while the world's own shared clock continues normally.

Conceptual model:

```go
type TravelSession struct {
    ID              string
    PartyID         string

    OriginRoomID    int
    DestinationRoomID int

    Route           []RouteSegment
    SegmentIndex    int

    StartedAt       time.Time
    ExpectedEndAt   time.Time

    State           TravelState

    Progress        float64

    AccruedHunger   float64
    AccruedThirst   float64
    AccruedFatigue  float64

    PendingEvent    *TravelEvent
}
```

This is conceptual only. Use GoMud's established IDs/types/timing/event systems after inspecting them.

## 6.2 Travel state machine

Recommended states:

```text
Idle
  ↓
Preparing
  ↓
Traveling
  ↓
┌──────────────────┐
│ Event/Encounter? │
└───────┬──────────┘
        │ no
        ↓
   Traveling
        ↓
    Completed

If interrupted:
Traveling
  ↓
Interrupted
  ↓
Event / Combat / Decision
  ↓
Resume / Abort / Reroute
```

Possible enum:

```text
Traveling
Interrupted
Paused
Completed
Cancelled
```

Avoid making the state machine more complex than needed in the first implementation.

## 6.3 How travel duration should be calculated

Travel duration should derive from route difficulty, not arbitrary timers scattered through commands.

Conceptual equation:

```text
Base route duration
× terrain modifier
× weather modifier
× encumbrance modifier
× party-speed modifier
× injury modifier
× mount modifier
= real travel duration
```

The exact scale must be configuration-driven.

Example:

A route representing several in-world miles might intentionally take 90 real seconds, 3 real minutes, or 8 real minutes depending on server balance.

Do **not** interpret "8 in-world hours" as "8 real hours."

Do **not** fast-forward eight game hours either.

Ashveil can use a compressed game-time scale, but the global GoMud clock remains authoritative and simply keeps running at its normal rate.

## 6.4 Progress updates

During a journey, the player should be able to receive periodic status.

Example:

```text
You continue along the rain-soaked Blackwood Trail.

Progress: 43%
Estimated time remaining: 1m 22s

Party:
Food: 16
Water: 21
Load: 83%
Condition: Tiring
```

Potential command:

```text
travel
```

or:

```text
status
```

The exact command should follow GoMud command conventions.

## 6.5 Commands during travel

The first implementation should explicitly decide which commands remain usable.

Likely allowed:

- say/chat/social commands
- party
- inventory
- equipment
- stats
- travel/status
- look/travel surroundings
- cancel/turn back, subject to rules

Likely restricted:

- ordinary directional room movement
- interacting with objects at origin/destination
- entering shops
- starting unrelated room actions

The system should not freeze the connection or block the player's command loop while waiting.

## 6.6 Random events during travel

A journey should be interruptible.

Potential travel events:

- hostile ambush
- wild animal
- traveler/NPC encounter
- merchant caravan
- broken wagon
- injured stranger
- hidden trail
- resource discovery
- bad weather
- fallen tree / blocked route
- bridge damage
- mount injury
- party argument
- mercenary observation
- tracks
- ruins
- treasure clue

The event scheduler should use configurable route/terrain/region tables.

Conceptual probability inputs:

```text
region
terrain
route
time of day
weather
party size
party visibility/noise
skills
recent events
```

MVP should start simple.

Example:

```text
every N travel ticks:
    roll encounter
```

or use scheduled event checkpoints.

Do not roll every server frame.

## 6.7 Event interruption

When an event occurs:

1. travel timer/progress is paused or checkpointed
2. party enters an event context
3. event is resolved
4. travel resumes from remaining progress

Example:

```text
Garrick raises a hand.

"Hold."

Three figures step out from behind the trees.

Your journey has been interrupted.
```

If combat begins, the 3x3 party formation should eventually initialize from the saved party formation.

## 6.8 Multiplayer interaction during travel

Architect the system so future travel sessions can potentially become shared/interactable world objects.

Possible future features:

- two player parties meet on the same route
- PvP interception
- caravans can be followed
- grouped human players travel together
- players see another party arrive
- route congestion or events are shared

This does not need to be implemented in the first milestone.

Do not make travel so private/instanced that future multiplayer interaction becomes impossible without a rewrite.

## 6.9 Disconnect behavior

This requires an explicit design decision.

Do **not** silently invent permanent behavior.

Recommended architecture:

- persist enough travel state to survive disconnects/restarts
- record start time, progress, current state, and route
- design travel so the policy can later be changed

Possible policies:

1. travel continues offline
2. travel pauses on disconnect
3. travel completes but unresolved encounters wait for login
4. disconnect cancels at last safe waypoint

For the MVP, choose the simplest safe policy only after inspecting GoMud session/disconnect semantics and document the decision.

---

# 7. Travel Cost and Survival

Do not tie all needs to a single timer.

Separate:

- route progress
- hunger
- thirst
- fatigue

## 7.1 Hunger

Hunger should primarily represent food need accumulated during sustained activity / time.

For multiplayer Ashveil, do not use a hidden personal game-time skip.

Instead, hunger can accrue gradually while traveling.

Conceptually:

```text
Hunger += travel progress × activity hunger rate
```

Travel can have a higher rate than standing idle.

Whether characters also become hungry while merely online and idle is a separate balance decision.

## 7.2 Thirst

Thirst should respond more strongly to exertion and environment.

Conceptually:

```text
Thirst +=
    progress
    × exertion
    × temperature modifier
    × weather modifier
```

## 7.3 Fatigue

Fatigue is primarily expedition exertion.

Conceptually:

```text
Fatigue +=
    route distance
    × terrain difficulty
    × encumbrance
    × weather
    × injury
    × mount/use modifiers
```

In real-time implementation, distribute the expected total fatigue across travel progress/ticks so interruptions preserve partial cost.

## 7.4 Do not consume a full journey's resources at start

Do not immediately subtract all expected food/water/fatigue when travel begins.

Why:

- journeys can be interrupted
- journeys can be cancelled
- route conditions can change
- characters can consume supplies
- encounters may change party composition/weight
- mounts may be lost/injured

Accrue costs proportionally to progress or at route checkpoints.

---

# 8. Terrain

Terrain should be data-driven.

Candidate types:

```text
road
plains
grassland
forest
dense_forest
swamp
hills
mountain
desert
snow
river
urban
dungeon
```

Not all need to exist at launch.

Example configuration concept:

| Terrain | Travel Speed | Fatigue | Mount Suitability |
|---|---:|---:|---|
| Road | 1.00 | 1.00 | Excellent |
| Plains | 0.90 | 1.10 | Excellent |
| Forest | 0.75 | 1.25 | Good |
| Swamp | 0.45 | 1.75 | Poor |
| Hills | 0.65 | 1.50 | Moderate |
| Mountain | 0.40 | 2.00 | Poor |

Do not hard-code balancing constants throughout Go code.

Use configuration/data files in the style GoMud already uses.

---

# 9. Routes and Exit Metadata

Do not immediately replace GoMud's exit structure.

First inspect it.

The eventual goal is that some exits or inter-area links can carry Ashveil travel metadata.

Conceptual:

```go
type TravelProfile struct {
    Enabled          bool
    Distance         float64
    Terrain          TerrainType
    Elevation        ElevationType
    RoadQuality      RoadQuality
    BaseDuration     time.Duration
    Exposure         ExposureType
    EncounterTableID string
}
```

An ordinary town exit:

```text
travel.enabled = false
```

A wilderness transition:

```text
travel.enabled = true
terrain = forest
base_duration = 120 seconds
```

The movement command should delegate to normal GoMud movement for ordinary exits and to Ashveil's travel service for travel-enabled exits.

That is preferable to creating two unrelated movement systems.

---

# 10. Directions

Currently documented GoMud directions:

- north / n
- south / s
- east / e
- west / w
- up / u
- down / d

Ashveil originally discussed richer directional layouts including:

- northeast
- northwest
- southeast
- southwest

Do **not** add diagonal commands as an early migration task.

First inspect:

- GoMud direction parsing
- exit representation
- reverse exits
- map rendering
- TinyMap
- web map editor
- room editor
- pathfinding
- GMCP/client room data
- scripting APIs

If directions are represented by a flexible enum/string and all mapper layers can support additional directions cleanly, diagonals can be added later.

If directions are deeply assumed to be six-way, meaningful rooms can still achieve rich navigation without diagonals.

---

# 11. Ashveil Feature Inventory to Port

The list below is intentionally broad.

The coding agent should verify each item against the current Python repo before implementation and update this document/plan if the prototype differs.

## 11.1 World and environment

### Day/night cycle
Ashveil has a day/night concept.

GoMud already has a day/night cycle.

Action:

- reuse GoMud's global time/day-night system
- do not build a separate Ashveil clock
- integrate Ashveil mechanics with GoMud time events

Possible Ashveil uses:

- encounter changes
- visibility
- camp atmosphere
- NPC behavior
- travel danger
- weather presentation
- certain abilities/mobs

### Weather
Ashveil has weather.

Action:

- inspect current GoMud weather support before implementation
- if no suitable complete engine exists, port/rebuild weather as an Ashveil module/domain service
- weather should be scoped by region/area rather than one random weather state per room
- weather should affect travel and camping

Candidate properties:

```text
type
intensity
temperature
wind
started_at
ends_at / transition schedule
```

Candidate types:

```text
clear
cloudy
rain
heavy_rain
storm
fog
snow
wind
heat
```

Start smaller.

### Environmental descriptions
Ashveil's weather/day/night should alter descriptions and travel messaging.

Prefer GoMud room scripting/templates/hooks rather than duplicating a rendering engine.

---

## 11.2 Survival

### Hunger
Port.

Requirements:

- stored per character/mercenary
- persistent
- changes gradually from activity
- affects characters at thresholds
- food restores it
- travel contributes proportionally to journey progress

### Thirst
Port.

Requirements similar to hunger, but more sensitive to:

- weather
- heat
- exertion

### Energy/stamina
The Python prototype includes an energy/stamina concept.

Before porting, distinguish:

- combat stamina
- long-term expedition fatigue

Do not make one number perform both roles unless that is explicitly desired.

Recommended:

- retain GoMud combat resource mechanics if applicable
- add Ashveil `Fatigue` for long-term travel/rest state

### Fatigue / tiredness
Expand into a first-class expedition mechanic.

Potential effects:

- slower travel
- reduced combat effectiveness at high fatigue
- rest pressure
- morale later
- mount dependence

### Eating/drinking
Use GoMud item/inventory systems.

Ashveil should add survival effects/metadata to food and drink items rather than create a second inventory system.

Example item properties conceptually:

```text
nutrition
hydration
weight
spoilage? (later)
```

Do not implement spoilage in the first milestone unless already trivial in GoMud.

---

# 12. Camping

Ashveil already has a campfire/camping concept.

Port and expand it.

Candidate commands:

```text
camp
camp status
camp fire
rest
sleep
break camp
```

Exact names should follow GoMud command conventions.

Camping should eventually interact with:

- fatigue recovery
- food
- water
- weather
- shelter
- fire
- camp quality
- nighttime
- encounters
- watch/sentry system
- injuries
- cooking
- morale

## Multiplayer correction

Camping must **not** advance global world time.

Sleeping cannot instantly turn night into morning for the whole server.

Instead:

- rest takes real server time
- fatigue recovers over that time
- characters remain in a camp state
- random camp events may occur
- world time continues normally

Example:

```text
You settle into camp.
Rest duration: 60 seconds.
```

The balance scale can be compressed.

Later, "sleep until dawn" could mean entering a rest state that resolves when the shared world clock reaches dawn, but this is not required for MVP.

---

# 13. Party System

GoMud already supports parties and demonstrates NPC/player parties.

GoMud also demonstrates hired mercenaries.

Therefore:

> **Do not create an entirely parallel Ashveil party engine until the existing party/mercenary code has been inspected.**

The preferred architecture is an Ashveil extension around GoMud's existing party ownership/membership model.

Target rule:

```text
maximum total party size = 5
```

Normally:

```text
1 player + max 4 mercenaries
```

Potential future question:

Can multiple human players join one expedition party?

Do not block this possibility unnecessarily, but MVP focus is player + mercs.

Party systems needed:

- recruit/hire
- dismiss
- follow leader
- persistent membership
- member status
- individual equipment
- party formation
- travel together
- camp together
- party survival
- shared cargo/supplies
- combat coordination

---

# 14. Mercenaries

Ashveil's mercenaries are more important than ordinary disposable followers.

They should become persistent managed party members.

Needed concepts:

- identity/name
- combat stats
- equipment
- inventory/carry contribution
- hunger
- thirst
- fatigue
- health/injuries
- tactical position
- combat behavior/strategy
- possibly morale later
- mount assignment later

GoMud's existing hired mercenary feature should be extended rather than blindly replaced.

Before coding, locate:

- merc creation
- merc ownership
- commands
- persistence
- combat participation
- following/movement
- party integration
- equipment support

---

# 15. 3x3 Tactical Formation

This is an Ashveil-defining system and must be preserved.

The formation is attached to the party.

Conceptual representation:

```go
type Formation struct {
    Slots [3][3]*CharacterID
}
```

Use GoMud IDs/types after inspection.

Requirements:

- maximum one member per slot
- maximum five occupied slots due to party cap
- leader can be placed like any other party member
- formation persists
- formation is displayed clearly in text
- combat reads formation state
- changing formation outside combat should be straightforward
- changing formation in combat may eventually cost an action

Candidate commands:

```text
formation
formation move <member> <row> <column>
formation swap <member-a> <member-b>
```

Command UX should be improved after prototype.

## Initial combat mechanics

Do not implement every tactical idea at once.

First tactical slice:

1. front row can intercept/protect rear
2. melee weapons have limited reach
3. polearms can reach farther
4. ranged characters benefit from rear positioning
5. adjacency exists and can be queried

Then expand into:

- row attacks
- column attacks
- AoE shapes
- shields protecting adjacent units
- formation buffs
- flanking
- movement abilities
- large enemies
- enemy formations

---

# 16. Combat Strategy / Mercenary AI

The Python prototype has party/mercenary combat behavior concepts.

Port the design intent, not necessarily the implementation.

Potential modes:

```text
aggressive
defensive
protect
ranged
support
hold
```

First inspect GoMud mob/merc combat AI.

Prefer configuring or extending the existing combat decision system.

Formation and strategy should be separate concepts:

- formation = where member stands
- strategy = what member tends to do

---

# 17. Inventory, Supplies, and Weight

GoMud already has inventory/items.

Do not create a second basic item system.

Ashveil needs an expedition logistics layer built on top.

At minimum distinguish:

1. personal equipped gear weight
2. personal carried inventory
3. party/shared cargo/supplies
4. mount cargo later

Potential party inventory:

```text
food
water
camp supplies
medical supplies
loot
trade goods
mount feed
```

Actual implementation should use normal GoMud items/containers wherever practical.

## Encumbrance

Party load should influence:

- travel speed/duration
- fatigue
- possibly stealth/noise
- mount usefulness

Example:

```text
Party capacity: 200 kg
Current load: 181 kg
Load: 90.5%

Travel duration modifier: +14%
Fatigue modifier: +8%
```

Do not use these exact numbers without balance testing.

---

# 18. Mounts

The Python Ashveil prototype includes or anticipates horses/mounts.

Mounts should become part of the expedition system, not a completely separate movement game.

Possible model:

```go
type Mount struct {
    ID
    Owner/AssignedMember
    Type

    CarryCapacity
    TravelSpeedModifier

    Health
    Fatigue

    FeedRequirement
}
```

Potential mount categories:

- riding horse
- pack horse
- mule
- specialized mounts later

Mounts should eventually affect:

- speed
- cargo
- fatigue
- route suitability

A mount injury/loss should potentially cause an overloaded party.

Do **not** implement mounts before normal travel, encumbrance, and fatigue work.

---

# 19. Party Inventory vs Individual Inventory

The prototype has party-management concepts.

The Go implementation should clearly define ownership.

Recommended model:

- characters keep normal GoMud inventories/equipment
- party may also have a shared expedition container/cargo concept
- supplies can be consumed from party cargo according to rules
- loot may default to normal GoMud distribution rules unless explicitly stored as party cargo

Avoid duplicating the same physical item in two inventories.

---

# 20. Persistence Requirements

The following Ashveil state should survive normal saves/server restarts as appropriate:

- party membership
- mercenary ownership
- formation slots
- mercenary equipment
- mercenary survival state
- player survival state
- party cargo
- mounts
- active camp if persistence model supports it
- active travel session or enough travel state to recover safely
- weather state / schedule if needed

Before adding a database or serialization mechanism, inspect GoMud's persistence conventions.

Follow GoMud's established patterns.

---

# 21. Events and Hooks

Ashveil should prefer GoMud events/hooks/modules instead of scattering cross-system calls.

Potential Ashveil events:

```text
TravelStarted
TravelProgressed
TravelInterrupted
TravelCompleted
TravelCancelled

CampStarted
CampfireLit
RestStarted
RestCompleted
CampBroken

PartyMemberAdded
PartyMemberRemoved
FormationChanged

HungerThresholdCrossed
ThirstThresholdCrossed
FatigueThresholdCrossed

WeatherChanged
```

Do not create an event bus if GoMud already has one.

Map these concepts onto GoMud's existing event/hook infrastructure.

---

# 22. Architecture Boundary

Desired conceptual organization:

```text
GoMud engine
├── networking
├── sessions/accounts
├── rooms/exits
├── maps
├── mobs
├── items/inventory
├── equipment
├── combat
├── quests
├── shops
├── admin
├── persistence
├── scripting
└── base parties/mercs
        │
        ▼
Game-domain systems
├── expedition
│   ├── travel
│   ├── routes
│   ├── terrain
│   └── encounters
├── survival
│   ├── hunger
│   ├── thirst
│   └── fatigue
├── company
│   ├── merc extensions
│   ├── formation
│   └── cargo
├── camping
├── weather
├── mounts
└── tactical combat extensions
```

This is conceptual.

The coding agent must adapt naming/location to GoMud's actual module/package architecture.

---

# 23. What NOT to Port from the Python Engine

Do not port Ashveil infrastructure that GoMud already solves well.

Likely examples:

- network server layer
- Telnet/websocket plumbing
- account/authentication system
- base session management
- base room engine
- generic exit parser
- generic mob engine
- generic inventory engine
- generic equipment engine
- generic quest framework
- generic shop framework
- admin tools
- map editor
- base persistence framework
- generic scripting framework
- generic multiplayer connection management

The goal is **not** a line-by-line Python-to-Go rewrite.

The goal is to migrate the distinctive gameplay.

---

# 24. GoMud systems to inspect before planning (done in Phases 0–1; nested `AGENTS.md` files now document each package)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 25. Phase 0 — Fork and Bootstrap (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 26. Phase 1 — Produce a GoMud Integration Map (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 27. Phase 2 — Minimal Company Slice (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 28. Phase 3 — 3x3 Formation State (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 29. Phase 4 — Survival State (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 30. Phase 5 — Terrain and Travel Profiles (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 31. Phase 6 — Travel Interruptions (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 32. Phase 7 — Camping (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 33. Phase 8 — Weather (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 34. Phase 9 — Encumbrance and Cargo (shipped)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 35. Phase 10 — Mounts

Only after travel and load systems work.

MVP mount effects:

- increases travel speed and/or
- increases cargo capacity

Later:

- mount fatigue
- health
- feed
- terrain suitability
- individual assignment

---

# 36. Phase 11 — Formation Combat

Integrate formation into GoMud combat.

First tactical rules:

1. front-row protection/interception
2. melee reach
3. polearm extended reach
4. ranged rear-line behavior
5. adjacency queries

Do not replace all GoMud combat unless unavoidable.

Prefer adding a tactical targeting/reach layer around the existing combat lifecycle.

Also in scope, pulled from the external combat design reference
(`docs/superpowers/specs/2026-09-22-combat-design-reference-external.md`) as
low-risk additions that layer onto the existing round-based
`internal/combat` resolution loop:

6. Guard Reactions — a limited per-combatant intercept/block resource
   (e.g. a shield-warrior role) that refreshes after the defender completes
   its next normal action, rather than being unlimited.
7. Weapon-flavored critical-hit secondary effects (e.g. sword crit →
   bleed, mace crit → stagger, hammer crit → knockdown) layered onto the
   existing crit branch in `internal/combat/calculations.go`.
8. Wounds — a status effect that temporarily lowers a combatant's
   recoverable max HP, independent of the timing model.
9. AI target-selection personality — priority tendencies per class/monster
   (favor wounded targets, favor threats to allies, etc.) rather than
   always picking the mathematically optimal target.

**Deliberately out of scope for Phase 11:** the reference doc's continuous
Readiness/Wind-up/Cast/Recovery timeline model. That replaces GoMud's
round-based combat resolution loop rather than layering on it, crosses this
project's concurrency/timer-rewrite escalation threshold, and needs its own
future design doc and owner decision before any implementation. Revisit it
as a later phase, after Phase 11 ships — see the reference doc's intake note
for detail.

---

# 37. Phase 12 — Rich Expedition Encounters

Once travel is stable, expand encounter content.

Examples:

- combat
- merchants
- discoveries
- route choices
- injured NPC
- weather hazard
- camp opportunity
- ruined site
- tracks
- resources
- social encounters

This becomes a content system rather than engine plumbing.

---

# 38. First Playable Milestone

The first milestone that proves Ashveil's identity should allow:

```text
Create character
    ↓
Enter town
    ↓
Hire 2 mercenaries
    ↓
Equip them
    ↓
Arrange 3x3 formation
    ↓
Buy food + water
    ↓
Check party load
    ↓
Leave town
    ↓
Begin real-time journey
    ↓
Watch travel progress
    ↓
Accumulate hunger/thirst/fatigue
    ↓
Get interrupted by encounter
    ↓
Fight
    ↓
Resume travel
    ↓
Reach wilderness landmark
    ↓
Make camp
    ↓
Light fire
    ↓
Rest in real time
    ↓
Continue
    ↓
Reach next town
```

If that loop is fun, the architectural migration has succeeded.

---

# 39. Example Travel Experience

```text
> north

You lead your company beyond Dunmar's western gate and onto
the Old King's Road.

Destination: Fork at the Black Oak
Terrain: Old road through forest
Weather: Light rain
Party: 4/5
Load: 78%

Estimated travel: 2m 15s

You begin traveling north.
```

Later:

```text
The road narrows beneath ancient pines.

Progress: 37%
Time remaining: ~1m 25s

Garrick is becoming tired.
```

Event:

```text
Garrick suddenly raises a fist.

"Movement ahead."

Three men emerge from the trees, weapons drawn.

Travel interrupted.
```

After combat:

```text
The last bandit falls.

The Old King's Road grows quiet again.

> continue

Your company reforms and resumes the journey.
Progress: 41%
```

Completion:

```text
The trees open around an enormous black oak split by lightning.

You have reached the Fork at the Black Oak.
```

This is the desired feel.

---

# 40. Real-Time Travel Scaling

Travel should be long enough to create anticipation but short enough to remain playable.

Do not bake one ratio into code.

Configuration examples to experiment with:

```text
short route:       10–30 sec
local wilderness:  30–90 sec
regional route:    1–5 min
major expedition:  several minutes
```

These are starting points, not final balance.

Long journeys can consist of multiple meaningful route segments with event checkpoints.

Avoid twenty-minute forced inactivity unless gameplay during travel is sufficiently rich.

---

# 41. Travel UX Must Avoid "Waiting Simulator"

A real-time travel system risks becoming boring.

Therefore travel should support:

- chat/social interaction
- party management
- equipment inspection
- inventory management
- readable progress
- ambient descriptions
- events
- discoveries
- route decisions later
- possibly scouting and travel actions later

Potential future travel commands:

```text
scout
forage
talk <merc>
check supplies
set pace
set formation
```

Do not implement them all for MVP, but keep the architecture open to them.

---

# 42. Party Pace

Future mechanic.

Possible pace modes:

```text
cautious
normal
forced
```

Effects could trade:

- travel time
- fatigue
- encounter detection
- ambush risk
- food/water usage

Do not add until the basic travel loop is stable.

---

# 43. Shared Multiplayer World Considerations

Because Ashveil is multiplayer:

## Never

- fast-forward shared time for one player
- mutate global weather because one party sleeps
- block the server while one party travels
- use a sleep call in a command handler that prevents processing
- assume only one travel session exists
- store active travel only in a transient socket/session object
- use unsynchronized mutable globals for parties/travel/events

## Design for

- many parties traveling simultaneously
- timers/events firing concurrently
- server restart
- reconnect
- party composition changes
- combat interruption
- race-free completion/cancellation
- exactly-once arrival semantics

---

# 44. Concurrency / Timer Safety

This is a server game.

Travel implementation must be robust against:

- completion timer and interruption firing simultaneously
- cancel and complete happening simultaneously
- disconnect during event
- duplicate resume
- duplicate arrival
- mercenary removal during travel
- party disbanding during travel
- server shutdown

Preferred principles:

- server-authoritative state
- single owner/service for TravelSession transitions
- explicit state machine
- idempotent completion/cancel
- event scheduling through GoMud's own scheduler/tick model if possible
- tests using controllable/fake clock if architecture allows

Do not scatter raw goroutines/timers throughout commands before inspecting GoMud conventions.

---

# 45. Testing Expectations

Every major Ashveil system should have tests.

High-value tests:

## Party
- max size
- ownership
- persistence
- duplicate member prevention

## Formation
- slot validity
- slot collisions
- member removal
- persistence

## Travel
- duration calculation
- start
- progress
- completion
- cancel
- interruption
- resume
- no global time jump
- survival proportional to progress
- no double completion

## Survival
- hunger thresholds
- thirst thresholds
- fatigue accumulation/recovery
- food/drink effects

## Encumbrance
- weight totals
- modifiers
- mount capacity later

## Camping
- begin/end
- recovery
- interruption
- multiplayer clock safety

---

# 46. Data-Driven Balance

The following should be configuration/data, not scattered constants:

- terrain modifiers
- travel scale
- hunger rates
- thirst rates
- fatigue rates
- encumbrance thresholds
- weather modifiers
- encounter rates
- camp recovery
- mount capacity
- mount speed
- route metadata

Follow GoMud's existing data-file/config conventions.

---

# 47. Open Design Questions

The AI agent should not silently decide all of these.

Document recommendations and flag decisions.

## Travel
- What happens if leader disconnects?
- Can a leader cancel and return to origin?
- Can a party turn back after >50%?
- Are routes one segment or multiple checkpoints?
- Can players meet each other mid-route in MVP?
- Can players attack traveling parties later?

## Survival
- Do hunger/thirst increase while idle?
- Do logged-out characters consume resources?
- How punitive are zero hunger/water states?
- Is fatigue individual, party-level, or both?

Recommended: individual values, with party travel speed constrained by relevant slowest/aggregate factor.

## Party
- Can other human players join an Ashveil mercenary party?
- Does human grouping use the same formation?
- Who controls formation in a mixed human party?

## Camping
- Is camp represented as a temporary room/object?
- Can other players discover a camp?
- Can camps persist through logout?

## Death
- What happens to mercenaries?
- Can mercs permanently die?
- What happens to cargo/mounts?

Do not let these block the first vertical slice unless required.

---

# 48. Initial Recommendation on World Design

Use **detail-rich semantic rooms connected through meaningful routes**, not a giant homogeneous tile grid.

Reasons:

- matches classic MUD strengths
- matches GoMud architecture
- easier content authoring
- rooms remain memorable
- lower world-builder burden
- route distance can still create expedition scale
- real-time travel provides geographic weight without filler rooms

The 3x3 grid remains exclusively a tactical formation system.

---

# 49. Migration Philosophy

When deciding whether to port Python code directly or rewrite behavior using GoMud conventions:

Prefer:

> Preserve behavior and design intent.

Not:

> Preserve implementation shape.

For example:

Python Ashveil may have survival state embedded in a session or character class.

That does **not** imply the Go version should use the same ownership model.

Use the best native GoMud architecture after inspection.

---

# 50. Agent Working Rules

The AI coding agent should follow these rules.

1. Read GoMud's `AGENTS.md`, `CLAUDE.md`, README, and development docs first.
2. Inspect the actual code before naming integration points.
3. Keep the Python Ashveil repo available as a reference.
4. Do not overwrite or delete the Python repo.
5. Establish a clean GoMud fork baseline before feature work.
6. Prefer extending GoMud over duplicating systems.
7. Keep Ashveil-specific logic modular where practical.
8. Do not fast-forward shared multiplayer time.
9. Do not block command processing while traveling/resting.
10. Make travel server authoritative.
11. Make travel state explicit and persistent/recoverable.
12. Keep balance values data-driven.
13. Write tests for state-machine and concurrency-sensitive systems.
14. Commit in small phases.
15. At the end of each phase, document what was reused from GoMud and what was added for Ashveil.
16. Before large engine-core changes, explain why a module/hook/extension is insufficient.
17. Avoid premature systems such as morale, spoilage, temperature simulation, complex mounts, and advanced tactical AI until the vertical slice works.
18. Preserve compatibility with future upstream GoMud merges when reasonably possible.
19. Use the current task's default agent and reasoning settings for implementation:
    - The lead owns architecture, planning, task scoping, implementation, review, verification, and commits.
    - Do not require a named model unless the owner explicitly requests one; a later default-agent request supersedes an earlier model selection.
    - Execute directly by default. Any authorized delegation follows `docs/AGENT_IMPLEMENTATION_WORKFLOW.md` with inherited model settings and a narrow brief.
    - Preserve the independent full-phase review gate and all concurrency, persistence, and multiplayer checks.
20. Do not dispatch an implementation task until the current phase design has explicit owner approval.
21. Keep the Python prototype read-only. Its local reference copy lives at `reference/ashveil-mud/`, is excluded through `.git/info/exclude`, and is a mechanics/design archive rather than a source tree to modify.
22. Isolate every plan, phase, or feature on its own git worktree and feature branch. Never implement or commit plan work directly on `master`; `master` is an integration branch and must stay clean. Create the workspace with `git worktree add .worktrees/<branch-name> -b <branch-name>` (`.worktrees/` is gitignored and is the project convention), run the baseline checks there, commit the plan tasks on that branch, and merge locally or open a PR against `origin` only after the phase's checks pass. Remove the worktree when the branch is finished. If a worktree is unavailable, create and check out a feature branch before making any commit.
23. Ship player help with every player-facing change: a help page for each new command or mechanic (or an update to the page it makes stale), listed in `_datafiles/world/default/keywords.yaml` and linked from its hub page (`help combat` for battles), a pointer from the tutorial lesson that covers it, and tests that it renders and that the tutorial's pointers resolve. See the root `AGENTS.md` ("Testing Guidelines"). Adopted 2026-09-27 at the owner's request.

## Repository Layout and Remote Policy

Keep all project-owned material inside the Ashveil GoMud project:

```text
ashveil-gomud/
    ├── docs/                # handoff, status log, active specs and plans
    └── reference/
        └── ashveil-mud/     # read-only, untracked Python mechanics reference
```

`ashveil-gomud` may use two remotes:

```text
origin   -> Robinsond76/ashveil-gomud
upstream -> GoMudEngine/GoMud
```

`upstream` is strictly read-only: it is retained only to fetch/compare upstream changes and to preserve future merge options. Configure its push URL as `DISABLED` so a push fails locally. Never push, force-push, or open a write workflow against `GoMudEngine/GoMud`. All Ashveil commits and any future pushes go only to `origin`.

---

# 51. First agent assignment (Phase 0–1 bootstrap; done)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 52. Reference feature mapping summary (superseded by the shipped phases; see `docs/PROJECT_STATUS.md`)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 53. Definition of Success

The migration is successful when Ashveil no longer needs to spend most development effort becoming a generic MUD engine.

GoMud should provide the mature multiplayer foundation.

Ashveil development should focus increasingly on:

- worlds
- routes
- mercenaries
- formations
- expedition preparation
- travel
- survival
- camping
- weather
- tactical battles
- encounters
- content

The central design test is:

> Does preparing for and undertaking a journey with a small mercenary company create interesting decisions?

If yes, the project is heading in the intended direction.

---

# 54. Source links (the prototype is at `reference/ashveil-mud/`; `upstream` is GoMud)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---

# 55. Short context prompt for a fresh agent (superseded by `CLAUDE.md`/`AGENTS.md`)

Removed on 2026-09-28 once complete; see git history (commit `d5ace46`).

---
