# Phase 4 Survival Corrections Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate Phase 4 survival-state loss and inheritance bugs, preserve native consumable matching, and reject invalid survival item metadata.

**Architecture:** Make company companion identity durable and non-reusable with a persisted high-water mark. Extend the survival lifecycle seam with exact member snapshots and a post-company-load roster reconciliation operation, so cross-plugin rollback and migration preserve or prune real state rather than manufacture defaults. Retain GoMud item matching by resolving the item portion through `FindInBackpack`, and treat a trailing token as a provision target only when it resolves against the current company roster.

**Tech Stack:** Go 1.24, standard `testing`, `testify`, GoMud plugins/YAML persistence, `make validate`, `go test -race ./...`.

**Spec:** `docs/superpowers/specs/2026-09-17-survival-state-design.md`; review findings for `e5625faa..5f07d4f0`.

## Global Constraints

- A dismissed companion's exact hunger, thirst, and fatigue must survive any failed company command rollback.
- A previously issued companion ID must never be assigned again for the same leader, including after `dismiss all`, save/load, or an empty company record.
- Preserve `0..100` normalization and stable `leader`/`companion:<id>` survival identity. Never persist a mob instance ID.
- A current roster companion with no stored survival entry starts at `FullNeeds()` and can be provisioned immediately.
- Legacy item matching, including partial names and numbered matches, remains valid with and without a trailing companion selector.
- Negative `nutrition` or `hydration` data is invalid; zero remains valid metadata for legacy items.
- Do not add idle drain, combat penalties, travel, rest/sleep commands, cargo, or automatic provisioning.
- Keep the fix limited to Phase 4 correctness; do not alter native party behavior or global game time.

---

## File Structure

- `internal/company/company.go` — persists the non-reusable companion-ID high-water mark and normalizes legacy records.
- `internal/company/company_test.go` — proves IDs survive dismiss, dismiss-all, empty records, and YAML round-trip without reuse.
- `internal/survival/survival.go` — defines exact `MemberSnapshot` and roster-reconciliation lifecycle interfaces.
- `internal/survival/survival_test.go` — covers seam forwarding and snapshot semantics.
- `modules/survival/survival.go` — captures/restores exact needs and atomically reconciles persisted companion keys against the loaded company roster.
- `modules/survival/survival_test.go` — proves migration/default initialization, durable pruning, and no inheritance from stale records.
- `modules/company/company.go` — performs snapshot-aware dismiss rollback and calls reconciliation after its registry loads.
- `modules/company/company_test.go` — tests a non-default survival rollback through the real lifecycle contract.
- `internal/usercommands/eat.go` and `drink.go` — preserve legacy matching while accepting only valid trailing provision selectors.
- `internal/usercommands/eat_test.go` and `drink_test.go` — cover partial and numbered matching with valid and invalid candidate suffixes.
- `internal/items/itemspec.go` and `internal/items/items_test.go` — validate non-negative nutrition/hydration.
- `docs/PROJECT_STATUS.md` — replace the premature Phase 4 completion claim only after verification succeeds.

---

### Task 1: Persist non-reusable companion identities

**Files:**
- Modify: `internal/company/company.go`
- Modify: `internal/company/company_test.go`
- Modify: `modules/company/formation_test.go`

**Interfaces:**
- `Record.NextCompanionID int` persists as `next_companion_id`.
- `normalizeNextCompanionID(record Record) int` returns at least `1` and strictly greater than every current companion ID.
- `Registry.Summon` assigns `record.NextCompanionID` then increments it before `Put`.

- [ ] **Step 1: Add failing non-reuse and migration tests**

Add a domain regression showing that dismissing companion `#2` and summoning again returns `#3`, and that `DismissAll` followed by summon still returns the next unused number. Add a module decode test for legacy YAML with companions but no `next_companion_id`; after reload it must assign `max(existing IDs)+1`.

```go
func TestRegistryNeverReusesDismissedHighestCompanionID(t *testing.T) {
	r := company.NewRegistry()
	_, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.True(t, r.Dismiss(7, second.ID))

	replacement, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 3, replacement.ID)
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run:

```bash
go test ./internal/company ./modules/company -run 'TestRegistryNeverReuses|TestDecodeCompanies.*NextCompanion' -count=1
```

Expected: FAIL because the current implementation derives the next ID from only current companions.

- [ ] **Step 3: Add the high-water mark without breaking old YAML**

Add this field:

```go
type Record struct {
	LeaderUserID     int         `yaml:"leader_user_id"`
	Companions       []Companion `yaml:"companions"`
	Formation        Formation   `yaml:"formation"`
	NextCompanionID  int         `yaml:"next_companion_id,omitempty"`
}
```

In `Registry.Put`, normalize the field to at least one greater than the greatest current ID. Keep a record with no companions and no formation whenever `NextCompanionID > 1`; that empty record is the durable high-water mark after a dismissal. `Summon` must use the normalized field, increment it, and store it before returning. Ensure `Get` returns the scalar unchanged. Do not rely on an in-memory counter.

- [ ] **Step 4: Run focused race tests**

Run: `go test -race ./internal/company ./modules/company -count=1`

Expected: PASS.

- [ ] **Step 5: Commit durable identity protection**

```bash
git add internal/company/company.go internal/company/company_test.go modules/company/formation_test.go
git commit -m "fix(company): retain non-reusable companion ids"
```

### Task 2: Add exact survival snapshots and durable roster reconciliation

**Files:**
- Modify: `internal/survival/survival.go`
- Modify: `internal/survival/survival_test.go`
- Modify: `modules/survival/survival.go`
- Modify: `modules/survival/survival_test.go`

**Interfaces:**
- Produces `type MemberSnapshot struct { Exists bool; Needs Needs }`.
- Extends `Lifecycle` with `SnapshotCompanyMember(leaderUserID, companionID int) (MemberSnapshot, error)`, `RestoreCompanyMember(leaderUserID, companionID int, snapshot MemberSnapshot) error`, and `ReconcileCompanyRosters(map[int][]MemberRef) error`.
- `ReconcileCompanyRosters` preserves leaders, removes non-roster companion entries, and creates `FullNeeds()` for every current companion missing an entry in one persisted registry write.

- [ ] **Step 1: Add failing snapshot and migration tests**

Cover restoring `{Hunger: 12, Thirst: 34, Fatigue: 56}` exactly after a removal, restoring an absent snapshot by removing the key, and reconciling a stored orphan `companion:2` away while initializing roster member `companion:3` to full needs.

```go
func TestReconcileCompanyRostersPrunesOrphansAndInitializesCurrentMembers(t *testing.T) {
	m := newTestModule(domain.Registry{Leaders: map[int]map[domain.MemberKey]domain.Needs{
		7: {
			domain.CompanionMemberKey(2): {Hunger: 1, Thirst: 2, Fatigue: 3},
		},
	}})
	err := m.ReconcileCompanyRosters(map[int][]domain.MemberRef{7: {
		{Key: domain.LeaderMemberKey, Name: "Hero"},
		{Key: domain.CompanionMemberKey(3), Name: "Scout"},
	}})
	require.NoError(t, err)
	_, stale := m.registry.NeedsFor(7, domain.CompanionMemberKey(2))
	assert.False(t, stale)
	assert.Equal(t, domain.FullNeeds(), m.registry.MustNeedsFor(7, domain.CompanionMemberKey(3)))
}
```

- [ ] **Step 2: Run focused tests and verify failure**

Run: `go test ./internal/survival ./modules/survival -run 'Test.*(Snapshot|Reconcile)' -count=1`

Expected: FAIL because the lifecycle cannot capture/restore exact data or reconcile durable records.

- [ ] **Step 3: Implement exact, failure-aware lifecycle operations**

`SnapshotCompanyMember` returns the stored state without mutation. `RestoreCompanyMember` writes the captured needs when `Exists` is true; otherwise it removes the key. Both persist and restore the in-memory registry on save failure.

`ReconcileCompanyRosters` receives all loaded company rosters in a single map. Build a valid-key set per leader containing `leader` and current companion keys. For every persisted leader, keep the leader record but delete companion keys absent from its valid set; for every current companion absent from state, add `FullNeeds()`. Persist once only if the normalized registry differs. Return `ErrPersistenceUnavailable` without mutation if load failed.

The plugin loader runs callbacks in reverse registration order, so survival loads before company. Do not reconcile inside `SurvivalModule.load`; invoke it after `CompanyModule.load` has installed its authoritative registry.

- [ ] **Step 4: Run focused race tests**

Run: `go test -race ./internal/survival ./modules/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit exact recovery and reconciliation**

```bash
git add internal/survival/survival.go internal/survival/survival_test.go modules/survival/survival.go modules/survival/survival_test.go
git commit -m "fix(survival): restore exact state and reconcile rosters"
```

### Task 3: Make company rollback and load synchronization use the new lifecycle

**Files:**
- Modify: `modules/company/company.go`
- Modify: `modules/company/company_test.go`

**Interfaces:**
- `CompanyModule.load` supplies all current records to `survival.ReconcileCompanyRosters` after a successful company decode.
- `dismiss` captures `MemberSnapshot` before removal; a failed company save invokes `RestoreCompanyMember` with that exact snapshot.
- `dismissAll` captures a snapshot for every companion and restores each exact snapshot if the company save fails.

- [ ] **Step 1: Add failing end-to-end rollback tests**

Use a real `SurvivalModule` with an injectable store and seed companion `#1` with non-default needs. Force the company store save to fail during `dismiss` and `dismiss all`. Assert the company record and survival state both equal their pre-command snapshots.

```go
func TestDismissCompanySaveFailureRestoresExactSurvivalNeeds(t *testing.T) {
	module, survivalModule := linkedTestModules(t)
	seedCompanionNeeds(t, survivalModule, 7, 1, domain.Needs{Hunger: 12, Thirst: 34, Fatigue: 56})
	module.store = failingCompanySaveStore{}

	_, err := module.dismiss(7, "#1")
	require.Error(t, err)
	assert.Equal(t, domain.Needs{Hunger: 12, Thirst: 34, Fatigue: 56}, survivalModule.NeedsFor(7, 1))
	assertCompanyContains(t, module, 7, 1)
}
```

- [ ] **Step 2: Run the regression tests and verify failure**

Run: `go test ./modules/company -run 'Test.*(ExactSurvival|ReconcileCompany)' -count=1`

Expected: FAIL because the current compensation calls `EnsureCompanyMember`, which creates full needs.

- [ ] **Step 3: Replace defaulting compensation with exact restoration**

Before `RemoveCompanyMember`, call `SnapshotCompanyMember`. On a downstream company save failure, call `RestoreCompanyMember` with the saved snapshot, then restore the domain company record. If restoration itself fails, return an error that joins the primary persistence failure and compensation failure; never report successful dismissal.

For load synchronization, build the roster map directly from every `m.registry.Companies` record after `m.registry = *loaded`, including leaders whose roster is empty so stale companions are pruned. If reconciliation fails, set `m.loadErr` and leave company mutations unavailable rather than allowing divergent persistence.

- [ ] **Step 4: Run cross-module race tests**

Run: `go test -race ./modules/company ./modules/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit atomicity compensation**

```bash
git add modules/company/company.go modules/company/company_test.go
git commit -m "fix(company): restore exact survival state on rollback"
```

### Task 4: Restore native consumable matching with optional valid targets

**Files:**
- Modify: `internal/usercommands/eat.go`
- Modify: `internal/usercommands/drink.go`
- Modify: `internal/usercommands/eat_test.go`
- Modify: `internal/usercommands/drink_test.go`

**Interfaces:**
- `findConsumable(rest string, user *users.UserRecord)` returns the selected item and a selector only when the final token is a valid current target according to a new read-only `survival.IsMemberSelector(leaderUserID, selector string) bool` seam.
- When no valid target is present, `FindInBackpack(rest)` receives the complete original input and preserves partial/numbered behavior.

- [ ] **Step 1: Add failing command regressions**

Register a fake selector resolver that recognizes `#2` only. Assert `eat rati #2` and `drink water #2` forward `#2` after partial item matching, while `eat rati stranger` and an item name ending with a non-member word are resolved as ordinary full-input item matches or retain the legacy missing-item response. Add numbered item coverage such as `drink waterskin 2 #2`.

```go
func TestEatKeepsPartialItemMatchWithValidTarget(t *testing.T) {
	useFakeMemberSelector(t, "#2")
	user := userWithItem(t, 17, edibleSpec("ration pack", 20, 1))
	_, err := Eat("rati #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, "#2", fakeProvisionerFor(t).lastSelector)
}
```

- [ ] **Step 2: Run command regression tests and verify failure**

Run: `go test ./internal/usercommands -run 'Test(Eat|Drink).*(Partial|Numbered|Target)' -count=1`

Expected: FAIL because the current parser demands an exact item match before it recognizes a suffix.

- [ ] **Step 3: Implement target-first parsing without guessing**

Add `IsMemberSelector` to the provisioner seam and have `modules/survival` return true only for `leader`/`me`/`self`, an extant `#id`/`id`, or an unambiguous roster name. In `findConsumable`, split quoted tokens, test the final token with this seam, and only then call `FindInBackpack` on the preceding token sequence. If that path fails—or the final token is not a valid member—call `FindInBackpack(rest)` unchanged. Do not use `items.FindMatchIn` directly.

- [ ] **Step 4: Run command race tests**

Run: `go test -race ./internal/usercommands -count=1`

Expected: PASS.

- [ ] **Step 5: Commit parsing compatibility**

```bash
git add internal/usercommands/eat.go internal/usercommands/drink.go internal/usercommands/eat_test.go internal/usercommands/drink_test.go internal/survival/survival.go internal/survival/survival_test.go modules/survival/survival.go modules/survival/survival_test.go
git commit -m "fix(commands): preserve consumable matching with targets"
```

### Task 5: Validate survival item metadata and reverify Phase 4

**Files:**
- Modify: `internal/items/itemspec.go`
- Modify: `internal/items/items_test.go`
- Modify: `docs/PROJECT_STATUS.md`

**Interfaces:**
- `(*ItemSpec).Validate() error` returns an error when `Nutrition < 0` or `Hydration < 0`.
- Zero metadata remains accepted for legacy consumables.

- [ ] **Step 1: Add failing validation tests**

```go
func TestItemSpecRejectsNegativeSurvivalMetadata(t *testing.T) {
	for _, spec := range []items.ItemSpec{
		{Name: "bad food", Nutrition: -1},
		{Name: "bad drink", Hydration: -1},
		{Name: "legacy item", Nutrition: 0, Hydration: 0},
	} {
		err := spec.Validate()
		if spec.Name == "legacy item" {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}
```

- [ ] **Step 2: Run focused metadata tests and verify failure**

Run: `go test ./internal/items -run TestItemSpecRejectsNegativeSurvivalMetadata -count=1`

Expected: FAIL because validation currently accepts negative values.

- [ ] **Step 3: Reject invalid values before other derived calculations**

At the beginning of `ItemSpec.Validate`, after the required-name check, return precise errors:

```go
if i.Nutrition < 0 { return fmt.Errorf("item nutrition cannot be negative") }
if i.Hydration < 0 { return fmt.Errorf("item hydration cannot be negative") }
```

Do not clamp negative authoring mistakes to zero; invalid files must fail validation visibly.

- [ ] **Step 4: Run complete verification**

Run:

```bash
make generate
make validate
go test -race ./...
```

Expected: all commands exit 0. Run `make build` if the local toolchain is available. Do not mark Phase 4 accepted if any command fails.

- [ ] **Step 5: Correct the project status and commit**

Replace the existing Phase 4 verification/status entry with the actual correction commits and fresh command results. Keep the known limitation that plugin files are not cross-file transactional, but remove the resolved statement that rollback restores a fresh default.

```bash
git add internal/items/itemspec.go internal/items/items_test.go docs/PROJECT_STATUS.md
git commit -m "fix(items): reject invalid survival metadata"
```

## Plan Self-Review

- **Coverage:** Task 1 prevents ID reuse; Tasks 2–3 protect exact state through migration and rollback; Task 4 restores partial/numbered consumable matching; Task 5 rejects invalid metadata and runs broad verification.
- **Detail:** Every task defines files, interfaces, failing tests, concrete implementation behavior, verification commands, and a narrow commit.
- **Consistency:** `MemberSnapshot`, `ReconcileCompanyRosters`, and `IsMemberSelector` are defined in Task 2/Task 4 before their company and command consumers. `NextCompanionID` is introduced in Task 1 and used by all later lifecycle work.
