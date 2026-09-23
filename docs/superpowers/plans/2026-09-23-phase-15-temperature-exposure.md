# Phase 15: Temperature, Clothing, and Exposure — implementation plan

See `docs/superpowers/specs/2026-09-23-phase-15-temperature-exposure-design.md`.
Open decisions resolved by recommendation (the user said "continue").
Implemented directly, without delegation: it involves timers/ticks and
persisted state.

## Tasks

- [x] **`internal/climate` (pure).** Tests first: `TestAirTemperature`
      (table), `TestWarmth`, `TestComfortAndStress` (worked examples),
      `TestExposureStep` (growth, ceiling, recovery, sign switch, lethal only
      at stress ≥ 25), `TestBandFor`, `TestHeatSourceRegistry`.
- [x] **`internal/items`: `Warmth`.** Test that it loads from YAML.
- [x] **`internal/weather` + `modules/weather`: `TemperatureMod`.** Tests:
      validation range, parsing.
- [x] **`internal/survival` + `modules/survival`: `MemberDrainService`.**
      Tests: drain applies to one member, is persisted, and a failed save
      rolls back.
- [x] **`modules/exposure`.** Tests first:
  - config parsing and defaults
  - a tick grows exposure and applies the band buff
  - band change swaps buffs
  - lethal-band damage
  - recovery indoors or at a fire
  - heat thirst and cold fatigue drains
  - persistence and reload
  - the `temperature` command report

  Then implement, and ship the buffs.
- [x] **Wiring.** Camping registers its heat source; the `weather` command
      shows the temperature through the climate seam. Wiring tests with a
      real room, user, and equipment through the tick.
- [x] **Data:** forest `TemperatureMod`s; `warmth` on cloaks, robes, and
      fur items.
- [x] `gofmt`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate`.
- [ ] **Review gate:** an independent reviewer subagent over the phase diff;
      verify findings, fix with regression tests.
- [ ] Update `docs/PROJECT_STATUS.md` (including the **Review:** line),
      then merge and push.
