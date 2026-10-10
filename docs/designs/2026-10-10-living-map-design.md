# Living map (design, 2026-10-10)

The owner asked (2026-10-10) for a map that reflects the player's conditions
as much as possible:
- the map changes when you enter a cave and shows the cave's inside;
- when you camp, your figure becomes the camp, with a bonfire once it is
  lit;
- in the dark, the map dims and hides what you can't see until a light
  source is lit, which shows a bit more, perhaps with a torch on the
  figure.

This design covers those ideas, phased so each phase ships on its own.
It builds on the map client as of E2/E3 (`window-map.js`, `map-tiles.js`,
the density-4 art) and on the server's light model (Phase 14,
`internal/rooms/light.go`).

The current world is the stock GoMud world and will be replaced. Every rule
here works from data any world provides (biomes, zones, levels, exits,
`lit` tags); world-building notes are at the end.

## What exists today

- **Places.** The map shows one zone and one level at a time
  (`replayZone` on a zone or z change). A cave that is its own zone, or a
  level below, already gets its own map, and the `cave-mouth` landmark marks
  the entrance outside. Nothing marks the moment you cross over.
- **Camp.** Your camp draws on its tile (`drawCamps`):
  - a tent, or a rough camp of bedrolls;
  - an unlit fire ring, a lit fire with smoke, or embers;
  - a sleeping mark while resting, and an inn bed at an inn.

  Your figure still stands over it.
- **Light (server).** Each viewer has a sight level from 0 to 2
  (`Room.VisibilityForUser`). The ambient light comes from the sky (day,
  moon), the biome (dark, lit), indoor or outdoor, fog, mutators and fixtures
  (a `lit` tag, a lit campfire). It is raised by the viewer's own light
  (`lightsource`), an ally's `partylight` or `nightvision`.
  - 0: you can't see the room ("You can't see anything!").
  - 1: this room only.
  - 2: this room and its exits.

  Combat, `look`, `scout` and the status prompt use it. **The client never
  receives it.**
- **Day and night on the map.** It is shaded by the clock alone
  (`nightLevel`, `drawNightShade`). It ignores the moon, fog, dark biomes
  and any light the player carries.
- **Weather** reaches the client in `Gametime.weather` (name, cloud cover,
  fog penalty). Only the time panel uses it.

## Phases

### LM1: You are the camp

While your company is camped in the room you stand in (`campInfo.has_camp`
and `room_id === currentRoomId`, not an inn):
- your figure and your companions' figures are not drawn;
- the camp is your marker, drawn larger: the tent or rough camp at full
  tile size, with the fire in front of it at icon size;
- the here-ring stays under it, so you still see where you are, and the
  company badge sits on the tent.

The fire tells the camp's state:
- **unlit:** a ring of stones;
- **lit:** flames, sparks and smoke, flickering;
- **embers:** a low glow after rest;
- **resting:** the sleeping mark floats over the tent.

When you walk out of the camp room, your figure steps out of it and the
camp stays behind as an ordinary camp marker. When you break camp, the
camp is gone. Party camps keep today's drawing.

- **Server:** nothing; `Company` and its camp facts already reach the map.
- **Art:** nothing new needed (A4's tent, camp and fire pieces). An
  optional A13 item is a bonfire sized for the camp-as-you view (below).
- **Help:** `worldmap` Camps. **Tests:** `map-check` camp section: the figure
  is hidden while camped here and drawn when you leave, and the fire states.

### LM2: Light and sight

The map shows what your character can see.

**Server: send the sight.** `Gametime` gains `sight` for the viewer's room:
`{ level: 0-2, daylight: bool, source: "" | "lantern" | "torch" | "party" | "nightvision" | "fixture" }`.
- `level` is `VisibilityForUser`.
- `daylight` is true when the room's own ambient light is full without
  help: outdoors by day with no thick fog, or an indoor or lit place.
- `source` is what raised the level, if anything.
- It is sent with the rest of `Gametime`, and also when any of these
  change:
  - a light, party-light or night-vision buff starts or ends;
  - a campfire is lit, banked or doused;
  - the room changes;
  - the hour turns (dusk, dawn, the moon);
  - the zone's weather changes.

  It only reveals what `look` already tells the player.

**Client: draw the sight.**

| Sight | The map |
|---|---|
| `daylight` and level 2 | As today: everything you have seen is shown. |
| level 2, not daylight (a carried light at night, a lit room in a dark cave) | You see your room and the rooms joined to it by exits. Those are drawn lit by a soft glow centred on you, about one and a half tiles across. |
| level 1 | You see your own room only, in a small glow. |
| level 0 | You can't see your room. Your tile is drawn almost black, with your figure as a faint shape (you know where you stand). |

- Rooms outside your sight are drawn as **remembered** (owner decision 1):
  dark and desaturated, with no figures, landmarks, resource icons or
  animation. Fog tiles stay as they are.
- The glow is warm and flickering for a torch, cool and steady for a
  party light (a floating orb) and night vision (no glow; a cool
  monochrome tint over what you see).
- A lit campfire is a fixture. Its own tile glows for everyone, and a camp
  at night is a pool of firelight.
- The current day/night shade is replaced. The sky's shade still applies
  outdoors, and darkness comes from the server's sight rather than the
  clock alone, so a moonlit night, thick fog and a dark cave each look
  different.

**Markers.**
- A carried light shows a small lantern (or a torch, for a torch) beside
  your figure, flickering (owner decision 2).
- A party light shows a floating orb over the one carrying it, and its
  glow covers allies standing with them.
- Companions and party members don't carry their own light marker unless
  they are its source.

**Art:** order A13 (below): the torch and orb markers, and optional warm and
cool glow sprites (the glow can be drawn in code).

**Help:**
- `worldmap` gets a new "Light and dark" section;
- `help light` (if absent) or the existing light help page gets a pointer
  to the map;
- the tutorial hint where light is taught.

**Tests:**
- Go: `Gametime.sight` for day, night, a moonlit night, a dark biome, a
  carried torch, a party light, night vision and a lit campfire, and that a
  buff change sends it.
- Node: the pure sight rules (which rooms are in sight, the glow radius) in
  a `map-sight.js` module.
- Browser: `map-check` at each level, the torch marker, a remembered room
  drawn without its icons, and daylight unchanged.

### LM3: Crossing into a place

When the map switches zone or level, the old map fades to black. The new
one fades in with a title for about two seconds: the zone's name (from
`Room.Info.area`), and the level when it isn't the surface ("Hollowweb Deep
— level 2 below"). A cave entered by its mouth therefore reads as going
inside: the forest map gives way to the cave's own map, usually dark,
where LM2's sight takes over.

- **Client only.**
- **Help:** a line in `worldmap`.
- **Tests:** `map-check` (the title shows on a zone change and on a level
  change, and not on a step within a zone; reduced motion skips the fade).

### LM4 (proposed): Weather on the map

`Gametime.weather` already reaches the client. The map can show it over
outdoor tiles:
- light rain streaks;
- snowfall;
- mist or fog drifting (with fog also cutting sight through LM2).

It is drawn in code or from small A13 overlays, kept faint so the map stays
readable, and it follows the reduced-motion setting.

## Owner decisions (2026-10-10)

1. **Unseen rooms are remembered:** drawn dark and still, with no live
   details.
2. **A lantern, with real light gear.** The carried-light marker is a
   lantern, and light gets a gameplay phase (LG, below): torches are
   one-time consumables, lanterns burn oil, and tents wear out after about 50
   pitches.
3. **Companions disappear into the camp** with you.
4. **Order:** LM1, then LG and LM2, then LM3. LM4 when wanted.

## LG: Light gear and lasting tents (gameplay, before LM2)

The owner's rules (2026-10-10). The numbers are proposals to tune. A game
day is one real hour (900 rounds of 4 s), so a game hour is 2.5 real
minutes.

- **Torch** (new item, cheap). `light torch` lights it and uses it up. It
  burns for about **6 game hours (15 real minutes)**, giving `lightsource`
  to its bearer, then is gone. It can't be put out and relit.
- **Lantern** (the existing item 20036, off-hand). It now needs oil.
  - It holds up to **24 game hours** of oil (one real hour), tracked as the
    item's uses in hours.
  - `light lantern` and `douse lantern` turn it on and off. While lit and
    worn it gives `lightsource` and burns oil, one hour's worth per game
    hour. Doused, it burns nothing.
  - With no oil it goes out, and says so.
  - A **flask of lamp oil** (new item, consumable) refills it: `fill lantern`.
  - Today's lantern lights whenever worn; it will light only when lit and
    fuelled.
- **Tents** (items 46 and 300–302) gain `uses: 50`. Pitching one for a rest
  spends a use, the way camp bells already wear (`spendItem` at rest start).
  The last use leaves a "worn-out tent" message, and the tent is gone.
- **Shops:** general shops and outfitters sell torches and lamp oil; tents
  already sell. Prices are set with the economy's balance pass.
- **State:** a lantern's oil and lit state live on the item (uses, and a
  lit flag), so they persist and survive copyover. A lit lantern burns by
  the shared clock's rounds; a torch's burn is a buff with a duration.
  Nothing advances game time.
- **Help:** `help light` covers torches, lanterns, oil and how light affects
  sight, combat and the map. `help camp gear` covers tent wear. Each is
  indexed and points from the tutorial lesson that hands out a light, or
  from Departure.
- **Tests:**
  - burning, refilling and running dry;
  - a torch burning out;
  - a tent's last use;
  - the lit state persisting;
  - the `lightsource` flag on and off;
  - the GMCP `sight` source showing `lantern` or `torch`.

## Art order A13 (for LM1 and LM2)

- `map/markers/lantern.png`: 2 frames, 128×128 per frame. A hand lantern,
  its flame flickering behind glass, drawn to hang beside a figure's
  shoulder.
- `map/markers/torch.png`: 2 frames, 128×128 per frame. A hand torch,
  flame flickering.
- Item icons for the lantern, torch and lamp oil, if the inventory shows
  icons by then.
- `map/markers/light-orb.png`: 2 frames, 128×128 per frame. A small pale
  floating light, pulsing.
- *(optional)* `map/camp/bonfire.png`: 4 frames, 256×256 per frame. A larger
  camp fire for the camp-as-you view, from the same stone ring as
  `fire-lit`.
- *(optional, LM4)* rain, snow and mist overlays: 4 frames, 512×512, with
  transparency, seamless.

The A4 rules apply (4 px grain, one shared box per animated row).

## State, persistence, multiplayer

There's no new persistent state. `sight` is computed from the shared clock,
weather and the room's state, and never advances time. Each player gets only
their own sight, which `look` already reveals. The map never shows anything
outside what the game tells that player. Remembered rooms come only from
that player's own map data.

## Notes for building the new world

These are what the map relies on, so the new world gets them for free:
- **Caves and dungeons** are their own zone, or another level of a zone. An
  entrance room carries the `cave-mouth` or `dungeon-stair` landmark
  legend.
- **Dark places** use a dark biome (cave, dungeon), so sight and light
  apply. A room with a hearth or lanterns gets the `lit` tag.
- **Water:** either real `water` rooms (which may need item 20030 to enter)
  or a ring of `shore` rooms around an empty middle, which the map fills as
  a lake (E2).
- **Roads** are `road` rooms joined by exits; the map joins their pieces
  (E2).
- `docs/designs/tile-ready-conventions.md` gains these points when the
  world work starts.
