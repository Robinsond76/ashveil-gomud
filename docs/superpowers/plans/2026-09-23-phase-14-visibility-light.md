# Phase 14: Visibility and Light — implementation plan

See `docs/superpowers/specs/2026-09-23-phase-14-visibility-light-design.md`.
Open decisions resolved by recommendation (the user said "continue with the
next phase").

## Tasks

- [ ] **`internal/rooms/light.go`: ambient and per-viewer visibility.**
  - Tests first (`internal/rooms/light_test.go`): table test of
    `ambientLevel(LightConditions)`; `TestLightFixtureRegistry`;
    `TestViewerLevel` (personal light, party light, nightvision);
    `TestHitPenaltyForVisibility` (including the lit-target floor).
  - Then implement `LightConditions`, `ambientLevel`, `Describe()`,
    `Room.LightConditions()`, `Room.GetVisibility()` (now ambient),
    `VisibilityForUser`, `VisibilityForMob`, ally resolution, fixture
    registry, and the penalty settings.
- [ ] **`internal/combat`: darkness hit penalty.**
  - Test first: `TestCalculateCombatAppliesDarknessPenalty` style check via
    a pure helper.
  - Then pass the attacker's penalty into `calculateCombat` from the four
    `Attack*` entry points (simulation passes 0).
- [ ] **`internal/usercommands/look.go`:** use `VisibilityForUser`.
- [ ] **`modules/camping`:** register a fixture provider for rooms with a
      lit campfire. Test: `TestLitCampfireIsLightFixture`.
- [ ] **`modules/light` (new):** `light` command, config
      (`DarkHitPenalty`, `DimHitPenalty`), shipped `partylight` flag and
      Floating Light buff. Tests for config parsing and the command report.
      Run `make generate`.
- [ ] **Data:** `floatinglight` spell (default world).
- [ ] `gofmt`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate`.
- [ ] Update `docs/PROJECT_STATUS.md`.
