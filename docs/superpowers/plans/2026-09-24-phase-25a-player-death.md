# Phase 25a: Player Death and Church Return — Implementation Plan

Design: [25a spec](../specs/2026-09-24-phase-25a-player-death-design.md).
Each task is tests first.

- [ ] **Task 1: level loss** (`internal/characters/character.go`,
  `character_death_test.go`). Tests first: `TestLoseLevelDropsOneLevelToFloor`
  (level 10 → 9, experience `XPTL(8)`), `TestLoseLevelAtLevelTwo` (→ 1, 0
  experience), `TestLoseLevelAtLevelOneResetsProgress`,
  `TestLoseLevelRecalculatesAndClamps`, `TestLevelUpGrantsPointsOnlyAbovePeak`,
  `TestLevelUpLegacyPeakZero`. Then `PeakLevel`, `LoseLevel`, and the
  `LevelUp` rule.
- [ ] **Task 2: death domain** (`internal/death/death.go`, `death_test.go`).
  Tests first: `TestNewRegistryRejectsBadEntries`, `TestChurchForCityOnly`
  (a village is never a checkpoint), `TestIsChurch`,
  `TestDestinationPrefersValidCheckpoint`,
  `TestDestinationFallsBack`, `TestDestinationNone`,
  `TestProviderNoneRegistered`. Then the registry, `Destination`, the
  seam, and MiscData key constants.
- [ ] **Task 3: the `suicide` branch** (`internal/usercommands/suicide.go`,
  `suicide_test.go`). Tests first: `TestSuicideWithoutProviderUsesShadowRealm`,
  `TestSuicideWithProviderRespawnsOnce` (no XP penalty, not permanent,
  `Respawn` called once), `TestSuicidePendingGoesStraightToRespawn` (no
  corpse, no broadcast, no drop). Then the branch.
- [ ] **Task 4: travel abandon** (`internal/expedition`,
  `modules/expedition`). Tests first: `TestAbandonForDeathNoProvider`,
  `TestAbandonForDeathTraveling`, `TestAbandonForDeathInterrupted`,
  `TestAbandonForDeathCompletedRecord`, `TestAbandonForDeathSaveFailureRestores`,
  `TestAbandonForDeathTimerDoesNothingLater`, `TestAbandonForDeathNoSession`.
  Then the seam and the module method.
- [ ] **Task 5: camp abandon** (`internal/camping`, `modules/camping`).
  Tests first: `TestAbandonForDeathRestingCamp` (no recovery, no Rested),
  `TestAbandonForDeathIdleCamp`, `TestAbandonForDeathInnStay`,
  `TestAbandonForDeathCampSaveFailureRestores`, `TestAbandonForDeathNothing`.
  Then the seam and the module method.
- [ ] **Task 6: company relocation** (`internal/company/provider.go`,
  `modules/company/relocate.go`). Tests first:
  `TestRelocateCompanyNoProvider`, `TestRelocateCompanyMovesLiveAttached`
  (aggro cleared; record, formation, and service unchanged),
  `TestRelocateCompanySkipsDeadDetachedAndUnattached`. Then the provider
  and the method, behind a small runtime seam.
- [ ] **Task 7: death module** (`modules/death`). Tests first:
  `TestParseConfig` (defaults, bad entries skipped, percent clamped),
  `TestCheckpointSetOnEnteringCity`, `TestCheckpointIgnoresVillageAndChurchless`,
  `TestCheckpointToldOncePerChange`, `TestRespawnAtCheckpoint` (one level,
  vitals, marker cleared, sessions abandoned before the move, company
  relocated), `TestRespawnInvalidCheckpointUsesFallback`,
  `TestRespawnNoChurchStaysPending` (and a retry costs no second level),
  `TestRespawnAbandonFailureStaysPending`, `TestRespawnLevelOne`.
- [ ] **Task 8: content** (`_datafiles/world/default`). Tests first:
  `TestShippedChurches` (18 and 2007 tagged `church`, 2004 ↔ 2007 exits,
  mob 65 non-hostile with no drops or gold, the shipped config resolves
  both cities). Then room 18's tag, room 2007, the exit, mob 65, and the
  config overlay.
- [ ] **Task 9: wiring** (`modules/death/wiring_test.go`). Through
  `plugins.Load` with the shipped config and rooms, `usercommands.TryCommand`,
  and `events.ProcessEvents`: walk into Dunmar (checkpoint), start travel
  with a live companion, `suicide`, then check the chapel, the level, the
  companion, the travel session, `users.SaveUser`, and a reload. Separately,
  a death without a checkpoint wakes at room 18. The clock is unchanged.
- [ ] **Task 10: docs and verification.** Config comments, module
  `AGENTS.md` files, `go test -race ./...`, `make generate`,
  `make validate`, independent review, `docs/PROJECT_STATUS.md`.
