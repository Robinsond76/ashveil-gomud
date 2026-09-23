# Phase 13: Sky and Environment (moon, cloud cover, fog, indoors)

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`.

## Prior-art check

- `internal/gametime` already owns the shared day/night clock
  (`GetDate()` → `GameDate{RoundNumber, RoundsPerDay, Night, MoonCount, ...}`).
  It has a `moon_count` setting (0–3) but no phase concept.
- Phase 8 (`internal/weather`, `modules/weather`) owns round-driven,
  zone-keyed weather with biome tables. `Condition` has name, description,
  and three informational percentages. Only the `forest` biome has a table.
- `internal/rooms.BiomeInfo` has `DarkArea`/`LitArea`, but nothing says
  whether a room is *indoors*. `internal/usercommands/look.go` appends
  `weather.RenderLine(room.Zone)` to every room in a tracked zone, including
  rooms that should be indoors.
- Rooms already have general-purpose `Tags`.

## Scope

1. **Moon phases.** A new pure package `internal/sky` derives an 8-phase
   moon from the absolute game day (`RoundNumber / RoundsPerDay`) and a
   configurable cycle length. Nothing is stored and nothing touches the
   clock. `moon_count: 0` means no moon. Moonlight strength (0–2) is
   computed and exposed now; Phase 14 will use it for visibility.
2. **Cloud cover and fog.** `weather.Condition` gains `CloudCover` (0–3:
   clear, scattered, broken, overcast) and `VisibilityMod` (−2..0, fog).
   Both default to 0, so existing configs stay valid. The forest table gains
   `fog` and `thick-fog` conditions, and the existing conditions get cloud
   cover values. Clouds hide the moon.
3. **Indoor rooms.** `BiomeInfo.Indoor` (yaml `indoor`) marks `cave`,
   `dungeon`, and `house` as indoors. A room tag `indoor` or `outdoor`
   overrides its biome, e.g. an inn room in a forest zone.
   `Room.IsIndoor()` resolves this.
4. **Indoor sky view.** An indoor room shows no weather, unless it has a
   non-secret exit to an outdoor room. Then it shows that room's weather as a
   glimpse ("Through the north exit: ..."). Exits are checked in sorted order
   so the choice is stable.
5. **Commands.** `look` shows the weather line (or glimpse), plus a moon line
   at night when outdoors and the moon is visible. `weather` now reports time
   of day, sky or cloud cover, fog, and moon phase, or a limited view
   indoors.

## Durable model

Nothing new is persisted. Moon phase is a pure function of the round counter.
Cloud cover and visibility are config on an already-persisted condition name,
and the indoor flag is data. Restart and copyover therefore reproduce the
same sky. The cycle length is module config (`MoonCycleDays`, default 8).

## Modules / packages

- `internal/sky` (new, pure): `MoonPhase`, `PhaseForDay`, `AbsoluteDay`,
  `Moonlight`, `CloudCoverName`, cycle-days setting.
- `internal/weather`: `Condition.CloudCover`, `Condition.VisibilityMod`
  (validated); `SkyView` + `RenderSky` (pure rendering of the lines).
- `internal/rooms`: `BiomeInfo.Indoor`, `Room.IsIndoor()`,
  `Room.SkyView()` (gathers indoor/glimpse/night/moon for a room).
- `modules/weather`: parse new fields, `MoonCycleDays`, richer command.
- `internal/usercommands/look.go`: use the room's sky view.
- Biome data in `_datafiles/world/{default,empty}/biomes`.

## Open decisions (resolved)

- Cycle length 8 game days, so players see the moon change within a play
  session. It is configurable.
- Multiple moons (`moon_count` 2–3) are shown as one moon for now.
- Visibility and combat effects are not in this phase; Phase 14 owns them.

Resolved by recommendation: the user approved the roadmap and asked to
proceed.

## Constraints / deferrals

- No gameplay effects in this phase (Phase 14 visibility, Phase 15
  temperature).
- Untracked zones still show the moon at night outdoors, with cloud cover
  treated as clear.

## Acceptance criteria

- `sky.PhaseForDay` cycles through all 8 phases over `cycleDays` days and
  wraps; `Moonlight` is 2 at full, 0 at new, and reduced by cloud cover.
- Conditions with out-of-range `CloudCover`/`VisibilityMod` are rejected.
- `Room.IsIndoor()` honours the biome flag and the `indoor`/`outdoor` tag
  overrides.
- `RenderSky` hides weather indoors, shows a glimpse through an outdoor
  exit, and shows the moon only at night when not fully clouded.
- `go test -race ./...`, `make generate`, `make validate` pass.
