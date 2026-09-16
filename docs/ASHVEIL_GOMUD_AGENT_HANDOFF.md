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

> **GoMud provides the MUD engine and infrastructure. Ashveil provides the game rules, expedition layer, survival systems, mercenary management, tactical formation combat, world content, and game identity.**

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
Ashveil gameplay layer
├── expedition
│   ├── travel
│   ├── routes
│   ├── terrain
│   └── encounters
├── survival
│   ├── hunger
│   ├── thirst
│   └── fatigue
├── party
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

# 24. GoMud Systems the Agent Must Inspect Before Planning Code

Before writing substantial Ashveil code, inspect and document at least:

```text
rooms
exits/directions
movement command
areas/zones
mapper/map editor
pathfinding

players/users/characters
mobs
mercenaries

parties
party commands

items
inventory
containers
equipment
item weight support

combat
combat rounds/ticks
mob AI

game time / day-night
server timers/ticks

events
hooks

commands/command registration
modules/module registration

persistence / save/load / data files

disconnect/reconnect behavior
server restart behavior
```

Also read repository guidance files such as:

```text
AGENTS.md
CLAUDE.md
README.md
contributor/development docs
```

before modifying source.

---

# 25. Phase 0 — Fork and Bootstrap

The first engineering task is **not gameplay**.

The first task is to establish an Ashveil GoMud fork that runs cleanly.

Recommended target repository:

```text
Robinsond76/ashveil-gomud
```

because `Robinsond76/ashveil-mud` already exists and must remain intact.

## Option A — GitHub CLI

If authenticated GitHub CLI is available:

```bash
gh repo fork GoMudEngine/GoMud \
  --fork-name ashveil-gomud \
  --clone
```

Verify remotes:

```bash
git remote -v
```

Desired conceptual setup:

```text
origin   -> Robinsond76/ashveil-gomud
upstream -> GoMudEngine/GoMud
```

If `upstream` is not configured automatically:

```bash
git remote add upstream https://github.com/GoMudEngine/GoMud.git
```

## Option B — existing GitHub fork UI

Fork:

```text
GoMudEngine/GoMud
```

into:

```text
Robinsond76/ashveil-gomud
```

then clone the fork and add upstream.

## Do not

- delete the Python prototype
- force-push over `ashveil-mud`
- remove GoMud history
- immediately rename all engine packages
- delete the default world before the engine has been understood
- rewrite GoMud core to "make it Ashveil" during bootstrap

## Bootstrap checks

Current GoMud documentation requires modern Go (currently Go 1.24+ in the reviewed README).

Run/read the repository's current instructions rather than trusting this document if they differ.

Typical commands currently documented include:

```bash
make reset-admin-pw
make build
make run
```

Also inspect:

```bash
make help
```

Establish a clean baseline:

- build succeeds
- tests succeed if test suite exists
- server starts
- web client loads
- admin panel loads
- character can log in
- character can move through rooms
- default combat works
- a party can be created if available
- hired mercenary behavior can be exercised if available

Commit a baseline before Ashveil changes.

Suggested commit:

```text
chore: establish Ashveil fork baseline
```

---

# 26. Phase 1 — Produce a GoMud Integration Map

Before coding new systems, create:

```text
docs/ASHVEIL_GOMUD_INTEGRATION.md
```

This should contain a concrete mapping from Ashveil concepts to exact GoMud packages/types/files.

Required mapping table:

| Ashveil concept | Existing GoMud type/package | Reuse / Extend / Replace | Notes |
|---|---|---|---|
| Party | TBD after inspection | Prefer extend | |
| Mercenary | TBD | Prefer extend | |
| Formation | none expected | Add | |
| Hunger | TBD | Add/extend | |
| Thirst | TBD | Add/extend | |
| Fatigue | TBD | Add | |
| Weather | TBD | Inspect first | |
| Camping | TBD | Add | |
| TravelSession | TBD | Add | |
| Route metadata | exits/rooms TBD | Extend | |
| Encumbrance | items TBD | Extend | |
| Mounts | TBD | Add | |

The agent should cite exact source files/functions in this internal document.

---

# 27. Phase 2 — Minimal Ashveil Party Slice

Goal:

- reuse existing GoMud party/mercenary mechanisms
- enforce Ashveil party cap
- persist the party
- expose useful party status

Target:

```text
1 player
0–4 mercenaries
max total = 5
```

Do not add formation combat yet.

Acceptance tests:

- player can hire/recruit a mercenary
- merc joins party
- party cannot exceed cap
- merc follows leader through ordinary rooms
- merc persists according to normal save behavior
- dismiss works
- no duplicate ownership/following framework was created unnecessarily

---

# 28. Phase 3 — 3x3 Formation State

Implement formation as persistent party metadata.

Acceptance:

```text
party has max five characters
each character can occupy one valid cell
two characters cannot occupy same cell
formation displays correctly
formation survives save/reload
```

No combat effects required yet.

---

# 29. Phase 4 — Survival State

Implement:

- hunger
- thirst
- fatigue

Requirements:

- persistent
- accessible for player + mercenaries
- threshold system
- commands/status display
- food/drink modifies needs
- clean service API so travel/camping can use it

Important:

Do not implement travel by setting giant countdown-specific survival hacks.

Provide operations conceptually like:

```text
ApplyTravelExertion(...)
ApplyRestRecovery(...)
ConsumeFood(...)
ConsumeWater(...)
```

---

# 30. Phase 5 — Terrain and Travel Profiles

Add data-driven terrain definitions.

Add travel metadata to the most natural GoMud world-link abstraction discovered in Phase 1.

Normal exits remain instant.

Travel-enabled exits start `TravelSession`.

Create only one or two test routes.

Example test world:

```text
Dunmar West Gate
    |
    | travel-enabled route
    |
Fork at the Black Oak
```

Acceptance:

- `north` or normal exit command detects travel profile
- party remains unavailable for ordinary movement while traveling
- connection remains responsive
- countdown/status works
- global world time is NOT modified
- party arrives when duration completes
- hunger/thirst/fatigue accrue based on progress

---

# 31. Phase 6 — Travel Interruptions

Implement one simple encounter type.

Example:

```text
bandit ambush
```

Acceptance:

- event can trigger mid-route
- travel progress is preserved
- travel pauses
- party resolves event/combat
- party resumes
- remaining duration/cost remains correct
- no duplicate completion timer fires

This phase needs strong concurrency/timer tests.

---

# 32. Phase 7 — Camping

Port Ashveil's campsite/campfire loop.

MVP:

- establish camp
- light fire
- rest
- break camp
- recover fatigue over real time
- no world fast-forward

Then integrate:

- weather
- encounters
- food
- water

---

# 33. Phase 8 — Weather

Integrate/port weather only after inspecting GoMud's actual current implementation.

Desired Ashveil behavior:

- shared environmental state
- preferably region/area scoped
- server driven
- affects travel duration
- affects fatigue/thirst
- affects camp quality
- affects descriptions

Weather transitions should not be based on a per-player clock.

---

# 34. Phase 9 — Encumbrance and Cargo

Add expedition weight calculations.

Inputs:

- personal equipment
- personal inventory
- shared cargo
- eventually mount capacity

Output:

- party total weight
- party total capacity
- load ratio
- travel modifier
- fatigue modifier

The implementation should reuse item weight if GoMud already supports it.

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
19. Use a deliberate agent-model split:
    - GPT-5.6 Terra at medium reasoning is the orchestrator. It owns architecture, planning, integration decisions, task scoping, and acceptance decisions.
    - GPT-5.6 Luna implements only narrowly scoped changes with explicit file boundaries, acceptance criteria, and test commands supplied by the orchestrator.
    - GPT-5.6 Terra at medium reasoning independently reviews every implementation diff and its test evidence before the change is accepted.
    - Elevate the implementation work to high reasoning, or assign it to Terra, when it involves concurrency, timers, persistent state recovery, disconnect/reconnect behavior, or other multiplayer invariants.
20. Do not dispatch an implementation task until the current phase design has explicit owner approval.
21. Keep the Python prototype read-only. Its local reference copy lives at `reference/ashveil-mud/`, is excluded through `.git/info/exclude`, and is a mechanics/design archive rather than a source tree to modify.

## Repository Layout and Remote Policy

Keep all project-owned material inside the Ashveil GoMud project:

```text
/Users/robinsondesouza/Documents/Codex/
└── ashveil-gomud/
    ├── docs/                # plans, handoff, integration map, verification evidence
    └── reference/
        └── ashveil-mud/     # read-only, untracked Python mechanics reference
```

`ashveil-gomud` may use two remotes:

```text
origin   -> Robinsond76/ashveil-gomud
upstream -> GoMudEngine/GoMud
```

`upstream` is strictly read-only: it is retained only to fetch/compare upstream changes and to preserve future merge options. Never push, force-push, or open a write workflow against `GoMudEngine/GoMud`. All Ashveil commits and any future pushes go only to `origin`.

---

# 51. First Agent Assignment

The first AI agent receiving this file should do the following.

## Task A — inspect both repositories

Inspect:

```text
https://github.com/Robinsond76/ashveil-mud
https://github.com/GoMudEngine/GoMud
```

Produce a concise repository audit.

For Ashveil:

- inventory actual current mechanics
- find their concrete files/classes/functions
- note anything this handoff missed
- identify what is prototype-only or incomplete

For GoMud:

- identify exact packages/files for rooms
- exits/directions
- areas
- maps/pathfinding
- parties
- hired mercenaries
- mobs
- players
- inventory/items/equipment
- combat
- game time
- events/hooks
- commands
- modules
- persistence
- server ticks/timers
- disconnect/reconnect

## Task B — fork GoMud

Create/use:

```text
Robinsond76/ashveil-gomud
```

unless the owner specifies a different name.

Preserve upstream remote.

Build/run/test vanilla GoMud.

Do not add gameplay until the baseline is confirmed.

## Task C — create integration document

Create:

```text
docs/ASHVEIL_GOMUD_INTEGRATION.md
```

Map exact Ashveil concepts to exact GoMud types/packages/functions.

## Task D — create implementation plan

Create a phased plan matching the dependency order in this handoff.

For each phase include:

- goal
- GoMud files/packages involved
- new files/packages
- data model changes
- commands
- persistence changes
- event hooks
- tests
- migration risks
- acceptance criteria

## Task E — begin only Phase 0/1

Do not jump straight into implementing all gameplay.

First complete:

- fork
- baseline
- integration audit
- plan

Then begin the smallest party-extension slice.

---

# 52. Reference Feature Mapping Summary

| Ashveil Feature | GoMud Foundation | Migration Direction |
|---|---|---|
| Multiplayer server | Existing | Reuse |
| Accounts/sessions | Existing | Reuse |
| Rooms | Existing | Reuse |
| Exits | Existing | Extend for travel metadata |
| N/S/E/W/U/D movement | Existing | Reuse |
| Diagonals | Not assumed | Investigate later |
| Maps/editor | Existing | Reuse/extend |
| Day/night | Existing | Reuse |
| Weather | Ashveil requirement | Inspect GoMud; add/extend |
| Hunger | Ashveil | Add |
| Thirst | Ashveil | Add |
| Fatigue | Ashveil | Add |
| Campfire | Ashveil | Port |
| Camping/rest | Ashveil | Port as real-time state |
| Parties | Existing | Extend |
| Hired mercs | Existing | Extend heavily |
| Party cap 5 | Ashveil rule | Add |
| 3x3 formation | Ashveil | Add |
| Formation combat | Ashveil | Add incrementally |
| Merc tactics | Ashveil | Integrate with combat AI |
| Inventory | Existing | Reuse |
| Equipment | Existing | Reuse |
| Shared cargo | Ashveil | Add atop item system |
| Weight/encumbrance | Ashveil requirement | Extend existing item model if possible |
| Food/water items | Existing item foundation | Add survival properties/effects |
| Mounts/horses | Ashveil | Add later |
| Route travel | Ashveil | Add |
| Real-time TravelSession | Ashveil | Add |
| Random travel events | Ashveil | Add |
| Travel encounters | Existing combat foundation | Integrate |
| Quests | Existing | Reuse |
| Shops | Existing | Reuse |
| NPC scripting | Existing | Reuse |
| Room scripting | Existing | Reuse |
| Admin tools | Existing | Reuse |
| Persistence | Existing framework | Extend |
| Web/Telnet clients | Existing | Reuse |

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

# 54. Source Links / Starting References

Ashveil prototype:

`https://github.com/Robinsond76/ashveil-mud`

GoMud:

`https://github.com/GoMudEngine/GoMud`

GoMud player navigation guide:

`https://github.com/GoMudEngine/GoMud/blob/master/_datafiles/guides/playing/README.md`

GoMud feature screenshots/documentation:

`https://github.com/GoMudEngine/GoMud/blob/master/feature-screenshots/README.md`

Important: repository code evolves. The agent should treat the current checked-out source as authoritative over this document where implementation details differ.

---

# 55. Short Context Prompt for a Fresh AI Agent

If a future agent needs a one-paragraph orientation, use this:

> We are migrating the Python multiplayer MUD Ashveil to a fork of GoMud. GoMud should remain the engine for networking, rooms, users, maps, combat, mobs, items, quests, shops, parties, mercenaries, persistence, and admin tooling wherever possible. Ashveil's identity is a maximum-five-member mercenary party, persistent 3x3 tactical formation, expedition logistics, hunger, thirst, long-term fatigue, carrying weight, food/water supplies, camping/campfires, weather, mounts, and dangerous long-distance journeys. Rooms should represent interesting locations rather than wilderness tiles. Long-distance links create asynchronous real-time TravelSessions: the player remains connected and can use allowed commands while a server-side timer/progress state runs; global multiplayer world time is never fast-forwarded. Travel costs accrue gradually, terrain/weather/load/mounts affect duration and fatigue, and random events can interrupt a journey before it resumes. Start by forking GoMud into a separate Ashveil Go repo, build vanilla GoMud, inspect exact integration points, document the mapping, and only then implement the smallest party/formation/travel vertical slices without duplicating mature GoMud systems.
