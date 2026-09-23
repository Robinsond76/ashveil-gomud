# Phase 13: Sky and Environment — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-23-phase-13-sky-environment-design.md`).
Open decisions resolved by recommendation: the user approved the roadmap and
asked to proceed.

## Tasks

- [x] **`internal/sky`: moon phases.**
  - Tests first (`internal/sky/sky_test.go`): `TestPhaseForDayCyclesAndWraps`,
    `TestPhaseForDayZeroCycleUsesDefault`, `TestMoonlight` (table: phase ×
    cloud cover), `TestAbsoluteDay`, `TestPhaseNamesDistinct`,
    `TestCloudCoverName`.
  - Then implement `sky.go`.
- [x] **`internal/weather`: cloud cover, fog, sky rendering.**
  - Tests first: extend `TestConditionValidate` with out-of-range
    `CloudCover`/`VisibilityMod`; `render_test.go`:
    `TestRenderSkyOutdoorDayShowsWeather`,
    `TestRenderSkyNightShowsMoon`, `TestRenderSkyOvercastHidesMoon`,
    `TestRenderSkyIndoorHidesWeather`, `TestRenderSkyIndoorGlimpse`,
    `TestRenderSkyNoMoonWhenMoonless`, `TestRenderSkyFullReportsFog`.
  - Then implement the fields, validation, `SkyView`, and `RenderSky`.
- [x] **`internal/rooms`: indoor flag and sky view.**
  - Tests first (`internal/rooms/sky_test.go`): `TestIsIndoorFromBiome`,
    `TestIsIndoorTagOverrides`, `TestOutdoorGlimpsePicksSortedNonSecretExit`
    (injected loader).
  - Then implement `BiomeInfo.Indoor`, `Room.IsIndoor`, `Room.SkyView`.
- [x] **`modules/weather` + `look`: config and commands.**
  - Tests first: parsing `CloudCover`/`VisibilityMod`, rejecting
    out-of-range values, `MoonCycleDays` parsing.
  - Then implement the parsing, set `sky` cycle days on load, the richer
    `weather` command, and `look` via `RenderSky`.
- [x] **Data:** `indoor: true` on cave/dungeon/house biomes (default and
      empty worlds); forest table gains `fog`/`thick-fog` and cloud cover.
- [x] `gofmt -l`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate`.
- [x] Update `docs/PROJECT_STATUS.md`.
