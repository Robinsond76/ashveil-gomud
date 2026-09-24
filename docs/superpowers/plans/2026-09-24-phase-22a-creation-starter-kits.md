# Phase 22a: Archetype Creation Step and Starter Kits — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-22a-creation-starter-kits-design.md`).

## Tasks

- [x] **`internal/archetypes`: kit field, `Choice`, `Creator` seam**
      (`archetypes.go`, `provider.go`).
  - Tests first: `TestValidateDropsNonPositiveKitIDs`
    (`archetypes_test.go`), `TestCreatorSeam` (`provider_test.go`: nil
    without a provider or a creator, nil once the user has an archetype,
    pass-through otherwise).
  - Then implement.
- [x] **`modules/archetype`: kit config and resolution** (`archetype.go`).
  - Tests first (`kit_test.go`): `TestParseKit`,
    `TestBuildTableDropsUnknownKitItems`, `TestShippedKitsResolveAndBalance`.
  - Then implement `Kit` parsing, the `itemName` seam, and resolution in
    `buildTable`.
- [x] **`modules/archetype`: owed kits, commit, grant** (`archetype.go`,
      `kit.go`).
  - Tests first (`kit_test.go`): `TestChooseOwesKitInSameSave`,
    `TestChooseSaveFailureOwesNothing`, `TestGrantKitExactlyOnce`,
    `TestGrantKitSkipsLegacyChoice`, `TestResetAndRechooseNoSecondKit`,
    `TestLostGrantRecoveredOnSpawn`, `TestPermadeathClearsOwedKit`,
    `TestKitMarkerSurvivesUserYAML`, `TestDecodeRegistryKits`,
    `TestListAndPreviewShowKit`.
  - Then implement `Registry.Kits` (clone, decode, clear), `commit`,
    `grantKit`, the `saveUser` seam, and the `Creator` methods.
- [x] **Content:** `10021-ash_quarterstaff.yaml`; `Kit` lists in the
      config overlay.
- [x] **`internal/usercommands/start.go`: the archetype step.**
  - Wiring tests first (`modules/archetype/wiring_creation_test.go`):
    `TestWiringStartArchetypeStepGrantsEachKit`,
    `TestWiringStartReconnectResumes`, `TestWiringStartConfirmNoAsksAgain`,
    `TestWiringStartCommitFailureContinues`,
    `TestWiringStartWithoutProviderUnchanged`,
    `TestWiringPlayerSpawnRecoversKit`.
  - Then implement.
- [x] `AGENTS.md` note for `modules/archetype` (none exists; add one) and the
      `internal/usercommands` note if it has one.
- [x] `go test -race ./...`, `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer; verify, fix,
      record.
- [ ] `docs/PROJECT_STATUS.md` Phase 22a entry with **Review:** line.
