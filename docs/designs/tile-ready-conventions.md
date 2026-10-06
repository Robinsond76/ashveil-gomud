# Tile-ready zone conventions

Phase 40d. In a tile-ready zone one room is one map tile, so the web client's
tiled map reads as a region and click-to-walk (`walkto`) works across it.
World building (phases 41 and 42) follows these rules for every new zone.
`ValidateTileReady` (`internal/rooms/tileready.go`) enforces them for each zone
that opts in; the shipped example is Alderbrook (`rooms/alderbrook`, ids
2101 to 2144, reached by the east exit of the Trappers' Post).

## Opting in

Set `tileready: true` in the zone's `zone-config.yaml`. A zone without it is
never checked, so the starting region and every older zone are untouched.
`TestShippedTileReadyZonesFollowTheConventions` runs the validator over every
shipped tile-ready zone.

## The conventions

1. **Hand-placed coordinates.** Every room sets `hascoordinates: true` with
   `mapx`, `mapy`, `mapz`, and no two rooms on one z-level share a coordinate.
2. **Exits lead to the adjacent coordinate.** Each compass exit goes to the
   room one tile away in its direction (north is y minus 1, east is x plus 1;
   diagonals allowed; `up` and `down` change z by one and keep x and y). An
   exit with another name, or a deliberate exception, declares `mapdirection`.
   Exits that leave the zone are not checked.
3. **A biome and a resource list.** Every room has a `biome`, or the zone sets
   `defaultbiome`, and every entry of `resources` is in the resource
   vocabulary (`water`, `forage`, `shelter`, `herbs`, `firewood`, `fishing`,
   `game`; `help resources`).
4. **Filler rooms are short.** A room with no `maplegend` has a description
   of at most two sentences, so walking stays quick to read. Landmark rooms
   may say more.
5. **Landmarks map to art.** A room's `maplegend` must be a legend in
   `sprites/map/landmarks.json` (or one of its intentional glyphs), so the
   room draws an S2 landmark. Add the legend and art together.

Not enforced, but the showcase shows the pattern: a campable clearing carries
the `camping` tag; terrain follows biome (a lake has a ring of shore rooms
around an open middle, since `water` needs an item to enter); stairs, caves
and lofts use `up` and `down`; a locked door is an exit `lock`.

## What walkto relies on

`walkto` plans over rooms the player has visited, so a tile-ready zone needs
no extra data. Exits with a `travel_profile` (journeys), an `exitmessage`, or
an unfound `secret` flag are never walked; a locked exit is walked only with
its key. See `help walkto` and the
[phase 40d design](2026-10-05-phase-40d-tile-region-travel-design.md).
