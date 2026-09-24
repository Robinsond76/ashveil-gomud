# Phase 25b: Companion Death and Resurrection — Implementation Plan

Design: [25b spec](../specs/2026-09-24-phase-25b-companion-death-design.md).
Each task is tests first.

- [ ] **Task 1: kept gear on `MobDeath`** (`internal/events/eventtypes.go`,
  `internal/mobcommands/suicide.go`, `suicide_kept_test.go`). Tests first:
  `TestMobDeathKeptWornDropChanceZero`, `TestMobDeathKeptWornDropChanceAll`,
  `TestMobDeathPermaGearKeepsEverything`. Then pre-rolled worn drops and the
  three fields.
- [ ] **Task 2: domain** (`internal/company/death.go`, `death_test.go`).
  Tests first: `TestPutKeepsLostOnlyRecord`, `TestGetClonesDeathAndLost`,
  `TestMarkDeadClearsCell`, `TestReviveRestoresFreeCell`,
  `TestReviveLeavesTakenCell`, `TestLoseArchivesAndCaps`,
  `TestResurrectionProviderNone`. Then `CompanionDeath`, `LostCompanion`,
  registry helpers, and the provider seam.
- [ ] **Task 3: survival `Dead` flag** (`internal/survival`,
  `modules/survival`). Tests first: `TestCompanyExertionSkipsDead`,
  `TestCompanyRestSkipsDead`, `TestProvisionRefusesDead`,
  `TestStatusMarksDead`. Then the field and the filters.
- [ ] **Task 4: other roster consumers** (`modules/walking`,
  `modules/exposure`, `modules/camping`). Tests first: a dead member isn't
  drained (walking, exposure), isn't granted a tier, isn't charged at the
  inn. Then the filters.
- [ ] **Task 5: company death and exclusions** (`modules/company/death.go`).
  Tests first: `TestCompanionDeathMarksDead` (kept gear, level, cell,
  allowance, save), `TestDeadNotRestoredOnSpawn`, `TestDeadExcludedFromDrift`,
  `TestRosterMarksDead`, `TestStatusShowsFallenAndLost`,
  `TestDeadCountsTowardCap`.
- [ ] **Task 6: allowance clock and expiry** (same file). Tests first:
  `TestAllowanceChargedOnlyOnline`, `TestAllowanceStepCapped`,
  `TestAllowanceChargedAtLogoutAndSaved`, `TestAllowanceWarnsOnce`,
  `TestExpiryArchivesAndFreesSlot`, `TestExpirySaveFailureRollsBack`,
  `TestLoginReminder`.
- [ ] **Task 7: resurrection** (`modules/company/resurrect.go`). Tests first:
  `TestResurrectCostsLevelAndSpawns`, `TestResurrectLevelOneFloor`,
  `TestResurrectRestoresCell`, `TestResurrectSaveFailureRollsBack`,
  `TestResurrectSpawnFailureAwaitsRestoration`,
  `TestResurrectExpiredIsLost`, `TestResurrectRefusesLivingAndUnknown`.
- [ ] **Task 8: the `resurrect` command** (`modules/death/resurrect.go`).
  Tests first: `TestParseKeeper`, `TestResurrectRefusedOutsideService`,
  `TestResurrectNeedsKeeper`, `TestResurrectNotWhileFighting`,
  `TestResurrectAtChurchAndShaman`, `TestResurrectListing`.
- [ ] **Task 9: content** (Fernhollow 2008/2009, mob 66, config keepers).
  Tests first: `TestShippedShaman` (tags, exits, keeper shape, the shipped
  config resolves the village and each keeper; not a checkpoint).
- [ ] **Task 10: wiring** (`modules/death/wiring_resurrect_test.go`).
  Through `plugins.Load`: real mob `suicide`, relog, refused outside,
  resurrected at the chapel and at the lodge, one expiry archived, clock
  unchanged.
- [ ] **Task 11: docs and verification.** Module `AGENTS.md` files, config
  comments, `go test -race ./...`, `make generate`, `make validate`,
  independent review, `docs/PROJECT_STATUS.md`.
