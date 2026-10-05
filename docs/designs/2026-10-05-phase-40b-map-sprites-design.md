# Phase 40b: map sprites, company badge and camp marker

Status: **design draft, awaiting owner approval** (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 2. Art: sets S0 and S1 in the
[sprite specification](2026-10-05-sprite-specification.md) (map units,
markers, camp).

## Goal

The map shows **you** as your class sprite, walking from tile to tile and
facing the way you went. A badge shows how many members travel with you.
Your camp appears on its tile when it is set up, with its fire, smoke and
resting states. Allied companies' camps appear too; nobody else's do.

## Prior art (code as of 2026-10-05)

- `window-map.js` draws the current room as a red square
  (`CURRENT_ROOM_COLOR`) and eases the camera to it (`setCameraTarget`).
- **Party members already show** as hearts at their rooms, eased between
  rooms (`partyHeartEase`), from the `Party` GMCP feed. Allied companies are
  GoMud parties (`internal/parties`).
- **Class sources:**
  - `Char.Info.class` is GoMud's skill-based profession, not the Ashveil
    class;
  - the `Company` payload carries each member's `archetype`;
  - Phase 38b adds a lineage and a current class.
- **Camp:** `Company.Camp` (`gmcp.CompanyCamp.go`) sends:
  - `has_camp` and `here`;
  - `room`, the room's **name**, not its ID;
  - `fire_lit`, `resting`, `rested`, `rest_percent`, `can_camp` and `inn`.

  `look` already lists the camps pitched in the current room
  (`camping.CampLines`).
- **Movement deltas:** `Room.Info.exitsv2` carries `dx/dy/dz` per exit. The
  client also knows the previous and new grid positions.

## Owner decisions

- A sprite represents the player on the map, chosen by class (2026-10-05).
- A camp sprite appears on the map when a camp is set up (2026-10-05).
- **Other companies' camps are hidden** on the map. Revisit for PvP, where
  visibility should depend on lighting, terrain and concealment skills
  (2026-10-05, deferred).
- **Other players do not appear on the map for now**, apart from your
  own party members, who already show today. Circle back later, alongside
  the camp idea (2026-10-05, deferred).
- **No race or gender variants for now.** Revisit when the game's races
  are reviewed (2026-10-05).

## Proposed for approval

### Unit sprite

- **The sprite key is chosen in this order:** current class (after 38b),
  then lineage, then the `adventurer` fallback. The client maps keys to
  sprite folders. A missing image falls back down the same chain, and
  finally to today's red square.
- **Server:** `Char.Info` gains `lineage` and `classid` (stable lowercase
  IDs such as `warrior` and `knight`). Before 38b ships, `lineage` is the
  archetype and `classid` equals it. `class` stays as it is for GoMud
  compatibility.
- **Facing:** the client takes it from the grid delta between the old and
  new positions:
  - north is `up`, south is `down`, east and west are `side`, with west
    mirrored;
  - diagonals use `side`;
  - z changes keep the facing and fade the sprite out and in;
  - teleports and recalls jump without walking.
- **Walking:** the existing camera ease drives the sprite. It plays the walk
  animation for the ease's duration, then idles. A queue of moves (fast
  typing or 40d's travel) plays in order, but never more than 2 steps
  behind; beyond that it snaps.
- **Scale:** the unit is drawn at the tile size's nearest whole-number
  multiple of 32 px, never smoothed.

### Company badge and allies

- The badge shows the number of company members **with the leader** (not
  separated and not away), taken from the `Company` payload. It is hidden
  when the leader travels alone.
- **Party (allied) players:** their hearts become their class sprites at
  75% scale, plus an ally pennant, when `Party` reports their class. Add
  `lineage` and `classid` to Party members. Otherwise they stay hearts.
  This shows nothing that isn't already shown today.

### Camp marker

- **`Company.Camp` gains:**
  - `room_id`, the camp room's ID;
  - `allied_camps`, a list of `{room_id, leader, fire_lit, resting}` for
    camps of companies in the player's party.

  **No other company's camp is ever sent.**
- **Drawing:**
  - `tent` (or `tent-ally`) on the camp's tile;
  - `fire-lit` with `smoke` when the fire is lit, `fire-unlit` otherwise;
  - `resting` while a rest runs;
  - `inn-rest` on the inn's tile while resting at an inn.
- Making, breaking or abandoning a camp updates the marker through the
  existing payload changes.
- **Not changed:** `look` still lists any camp pitched in your own room,
  because that is in-room sight, not the map. PvP visibility is deferred.

### Settings

Map settings gain:
- `sprites` (on or off; off restores the classic square);
- `showCamp` (on or off).

Both are per-viewer browser settings.

## State and persistence

There is no new server state. All of it is derived from existing company,
camp and party state. After a copyover or reconnect, the client rebuilds
everything from the next payloads.

## Integration points

| Area | Change |
|---|---|
| `modules/gmcp/gmcp.Char.go` | `lineage` and `classid` |
| `modules/gmcp` Party payload | member `lineage` and `classid` |
| `modules/gmcp/gmcp.CompanyCamp.go` | `room_id`, `allied_camps` (party-only) |
| `window-map.js` | unit sprite, facing, walk queue, badge, camp layers, settings |
| A new client sprite loader | loads images once, with the fallback chain and an onload redraw. 40c and 40f reuse it |

## Player help and tutorial

- **New page:** `help worldmap`, with the aliases `map window`, `tile map`
  and `world map`. It covers what the sprite, badge, pennants and camp
  marker mean, and the settings. It links to `help resources` (40a) and
  `help camp`.
- **Updated pages:**
  - `help map` (GoMud's skill page) gets a see-also;
  - `help webclient` gets the map section;
  - `help camp` mentions the map marker and allied camps.
- **Tutorial:** the Camp lesson hints that the camp appears on the map
  window.

## Acceptance tests

1. `Char.Info` carries `lineage` and `classid` for a new character of each
   base class.
2. `Company.Camp`:
   - carries `room_id`;
   - lists an allied (party) company's camp;
   - **never lists a camp of a company outside the party**, including one
     in the same room.

   This is a regression test for the owner's rule.
3. `Party` members carry `lineage` and `classid`.
4. **Browser checks** (scripted with the pre-installed Chromium; screenshot
   kept with the PR):
   - walking north, east and west shows the correct facings;
   - making a camp shows the tent;
   - lighting the fire animates it;
   - breaking the camp removes the marker;
   - with `sprites` off, the classic square shows.
5. A missing sprite image falls back without errors.
6. `help worldmap` renders, and `TestTutorialHelpPointersExist` passes.
7. `make js-lint` passes.

## Open questions

None. Other players' visibility and race variants are deferred owner
decisions (above).
