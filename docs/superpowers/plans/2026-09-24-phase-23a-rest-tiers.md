# Phase 23a: Rest Tiers — Implementation Plan

Design: [23a spec](../specs/2026-09-24-phase-23a-rest-tiers-design.md).
Each task is tests first.

- [ ] **Task 1: pure tier rules** (`internal/camping/tiers.go`,
  `tiers_test.go`). Tests first: `TestDecideNeverDowngrades`,
  `TestDecideRemovesLowerTier`, `TestMergeOwedKeepsHigherTier`,
  `TestMergeOwedReplacesExpired`, `TestRemainingRounds`. Then `Tier`,
  `Decide`, `OwedGrant`, `MergeOwed`, `RemainingRounds`.
- [ ] **Task 2: walking's Rested tier** (`modules/walking`: buff
  `1033-rested.yaml`, flag `rested.yaml`, `RestedPct`, precedence, view
  note). Tests first: `TestRestedCutsStrainByAQuarter`,
  `TestWellRestedWinsOverRested`, the config parse test extended, and the
  shipped buff/flag file test extended.
- [ ] **Task 3: camping config and seams** (`modules/camping/inn.go`).
  Tests first: `TestParseInnSettings` extended with `RestedBuffId`,
  `RestedDuration`, `WellRestedDuration`. Seams: `grantBuff(c, id, rounds)`,
  `removeBuff`, `hasBuff`, `companions(leader) (live map[int]*Character,
  roster []int)`, `roundsFor(duration)`.
- [ ] **Task 4: camp Rested grant and exclusivity** (`modules/camping`).
  Tests first: `TestCampTimerMarksRestedPendingWithoutBuffs`,
  `TestCampRestGrantsRestedOnce`, `TestCampRestDoesNotDowngradeWellRested`,
  `TestInnWellRestedRemovesRested`, `TestBothPendingGrantsWellRested`,
  `TestLegacyRegistryWithoutTierFieldsLoads`,
  `TestTierGrantSaveFailureRetries`.
- [ ] **Task 5: owed grants** (`modules/camping`). Tests first:
  `TestAbsentCompanionGetsOwedTierOnRestoration` (remaining rounds, once),
  `TestOwedTierExpires`, `TestOwedSurvivesReload`,
  `TestCampOwedNeverDowngradesOwedWellRested`.
- [ ] **Task 6: spawn normalization and buff 16** (`modules/camping`,
  `_datafiles/world/default/buffs/16-well_rested.*` → `16-refreshed.*`). Tests first:
  `TestPlayerSpawnDropsRestedUnderWellRested` (through the `PlayerSpawn`
  listener), `TestOnlyOneShippedBuffIsNamedWellRested` (reads the shipped
  default world and module buff files).
- [ ] **Task 7: wiring tests through real entry points**
  (`modules/camping/wiring_test.go`): `camp`, `camp fire`, `camp rest` user
  commands in the real Fork at the Black Oak with real buff specs, through
  the timer and the `NewRound` listener, ending with the Rested buff (flag
  `rested`, configured rounds) on the leader and a live companion mob, and
  an absent companion restored later. Walking wiring: a step with the
  Rested buff costs 75%.
- [ ] **Task 8: docs and verification.** Config comments, module
  `AGENTS.md` files, `go test -race ./...`, `make generate`,
  `make validate`, independent review, `docs/PROJECT_STATUS.md`.
