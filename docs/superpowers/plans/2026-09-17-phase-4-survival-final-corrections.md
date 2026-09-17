# Phase 4 Final Survival Corrections Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining failed-summon cleanup and stale numeric-selector authorization gaps in Phase 4 survival.

**Architecture:** Treat a companion ID as spent once survival initialization has durably succeeded; failed summon cleanup may remove the companion, but it must retain and persist the advanced company high-water mark before allowing another summon. Resolve every non-leader provision selector from the authoritative roster, never from survival persistence; provisioning initializes missing current-roster state only after that authorization check.

**Tech Stack:** Go 1.24, standard `testing`, `testify`, GoMud plugin/YAML persistence, `make validate`, `go test -race ./...`.

**Spec:** `docs/superpowers/specs/2026-09-17-survival-state-design.md`; corrective plan `docs/superpowers/plans/2026-09-17-phase-4-survival-corrections.md`; review of `5f07d4f0..8eeaf973`.

## Global Constraints

- After `EnsureCompanyMember` succeeds, the assigned companion ID must never be reused, even when spawn, company persistence, or survival cleanup subsequently fails.
- A failed command must return every primary and cleanup persistence error; it must never report a successful summon.
- A numerical selector (`#<id>` or `<id>`) is valid only for a companion in the authoritative current company roster.
- Survival persistence is not authorization: stale/missing records must neither authorize a dismissed companion nor block a current companion from receiving default state.
- Preserve exact dismissal rollback, `0..100` needs, native item matching, and the no-global-time invariant.
- Do not introduce a cross-file transaction, idle drain, cargo, travel, or combat changes.

---

## File Structure

- `modules/company/company.go` — preserves the advanced ID high-water mark and joins failed-summon cleanup errors.
- `modules/company/company_test.go` — proves retry after failed spawn/company save cannot reuse a survival-backed ID.
- `modules/survival/survival.go` — adds a shared roster-membership helper and makes both provisioning and `IsMemberSelector` use it.
- `modules/survival/survival_test.go` — proves stale numeric keys cannot be targeted and current members with missing state initialize correctly.
- `internal/usercommands/eat_test.go` and `drink_test.go` — prove command parsing does not treat stale numeric IDs as targets.
- `docs/PROJECT_STATUS.md` — records final correction verification after successful checks.

---

### Task 1: Retain spent IDs when failed summon cleanup cannot remove survival state

**Files:**
- Modify: `modules/company/company.go:255-294`
- Modify: `modules/company/company_test.go`

**Interfaces:**
- `summon` captures the newly advanced `NextCompanionID` after `Registry.Summon`.
- `rollbackSummon` removes the transient companion but retains that advanced high-water mark, then persists the company registry before returning a cleanup error.
- Failed spawn and failed company-save paths return `errors.Join(primaryErr, cleanupErr)` when `RemoveCompanyMember` or high-water persistence fails.

- [ ] **Step 1: Add failing failed-summon regressions**

Use a real survival lifecycle with an injectable survival store. Force native `Spawn` to fail after `EnsureCompanyMember` persists `companion:1`, then force `RemoveCompanyMember` to fail. Assert the failed command returns both errors, the company registry retains `NextCompanionID == 2` with no companion, and a retry assigns `#2` rather than inheriting `#1` state. Repeat for a company-save failure after native spawn succeeds.

```go
func TestSummonCleanupFailureRetainsSpentID(t *testing.T) {
	companyModule, survivalModule := linkedTestModules(t)
	companyModule.runtime = failingSpawnRuntime{}
	survivalModule.store.(*fakeStore).saveErr = errors.New("survival cleanup disk full")

	_, err := companyModule.summon(7, 100, "58")
	require.Error(t, err)
	assert.Equal(t, 2, nextID(t, companyModule, 7))
	assert.Equal(t, domain.FullNeeds(), survivalModule.MustNeedsFor(7, domain.CompanionMemberKey(1)))

	survivalModule.store.(*fakeStore).saveErr = nil
	companyModule.runtime = workingRuntime{}
	message, err := companyModule.summon(7, 100, "58")
	require.NoError(t, err)
	assert.Contains(t, message, "#2")
}
```

- [ ] **Step 2: Run the focused tests and verify failure**

Run: `go test ./modules/company -run 'TestSummon.*(CleanupFailure|SpentID)' -count=1`

Expected: FAIL because the current rollback restores the prior record and discards `RemoveCompanyMember` errors.

- [ ] **Step 3: Implement a durable high-water rollback path**

Replace the closure that restores the entire pre-summon record with a helper that removes only the transient companion from the post-summon record and preserves its already incremented `NextCompanionID`. Call `m.save()` on that high-water-only record before returning from a failed spawn or failed company save. Collect the primary failure, failed survival removal, and failed high-water persistence with `errors.Join`.

If survival initialization itself fails before it has persisted anything, restore the exact pre-summon company record as today. If cleanup succeeds, still retain the spent ID because survival initialization had already committed a durable identity. Do not call `Registry.Dismiss` followed by the old pre-summon restore, because that reopens ID reuse.

- [ ] **Step 4: Run focused cross-module race tests**

Run: `go test -race ./modules/company ./modules/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit failed-summon identity safety**

```bash
git add modules/company/company.go modules/company/company_test.go
git commit -m "fix(company): retain spent ids after failed summon cleanup"
```

### Task 2: Authorize numeric targets through the current roster

**Files:**
- Modify: `modules/survival/survival.go:350-410`
- Modify: `modules/survival/survival_test.go`
- Modify: `internal/usercommands/eat_test.go`
- Modify: `internal/usercommands/drink_test.go`

**Interfaces:**
- Produces `currentRosterMember(leaderUserID int, key domain.MemberKey) (domain.MemberRef, bool)`.
- `resolveMember` accepts a numeric companion key only when `currentRosterMember` returns it.
- `IsMemberSelector` uses the same helper.
- After roster authorization, `Provision` calls `registry.Ensure` so a current companion with no stored state receives `FullNeeds()`.

- [ ] **Step 1: Add failing roster-authorization tests**

Seed stale `companion:2` survival needs but provide a roster containing only the leader and `companion:3`. Assert both `Provision(7, "#2", ...)` and `IsMemberSelector(7, "#2")` reject it. Assert `Provision(7, "#3", ...)` succeeds without a pre-existing survival entry and persists default state plus the applied benefit.

```go
func TestNumericSelectorRequiresCurrentRosterMembership(t *testing.T) {
	m := newTestModule(registryWithNeeds(7, domain.CompanionMemberKey(2), domain.Needs{Hunger: 1, Thirst: 1, Fatigue: 1}))
	useRoster(t, roster(7, domain.LeaderMemberKey, domain.CompanionMemberKey(3)))

	assert.False(t, m.IsMemberSelector(7, "#2"))
	_, err := m.Provision(7, "#2", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrUnknownMember)

	result, err := m.Provision(7, "#3", domain.Benefit{Nutrition: 10})
	require.NoError(t, err)
	assert.Equal(t, 100, result.Needs.Hunger)
}
```

- [ ] **Step 2: Run focused module tests and verify failure**

Run: `go test ./modules/survival -run 'Test.*(NumericSelector|MissingState)' -count=1`

Expected: FAIL because current code trusts the existence of persisted needs for numeric selectors.

- [ ] **Step 3: Implement one shared authorization helper**

Implement `currentRosterMember` by scanning `domain.CurrentRoster(leaderUserID)` for an exact `MemberKey`. For a parsed numeric selector, call it in both `resolveMember` and `IsMemberSelector`; do not use `hasMember` as authorization. For named selectors, retain `matchCompanionName` over the same roster. In `Provision`, retain the existing `registry.Ensure` after resolution, which makes an authorized current companion with a missing record start at full needs.

Keep `hasMember` only where the implementation needs to inspect persisted state, not to decide who may be targeted. Add a command-level fake-provisioner test showing a stale `#2` is not stripped as a target suffix, so ordinary `FindInBackpack` receives the full input.

- [ ] **Step 4: Run module and command race tests**

Run:

```bash
go test -race ./modules/survival ./internal/usercommands -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit roster-authorized selectors**

```bash
git add modules/survival/survival.go modules/survival/survival_test.go internal/usercommands/eat_test.go internal/usercommands/drink_test.go
git commit -m "fix(survival): authorize numeric targets from roster"
```

### Task 3: Reverify and correct Phase 4 status

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/superpowers/plans/2026-09-17-phase-4-survival-final-corrections.md` only to check completed steps after evidence exists

**Interfaces:**
- Project status lists the new correction commits and actual verification results.

- [ ] **Step 1: Generate and run full verification**

Run:

```bash
make generate
make validate
go test -race ./...
```

Expected: each command exits 0. If `make validate` cannot access the Go cache in the sandbox, re-run it with the approved project validation permission and record that the command itself succeeded.

- [ ] **Step 2: Update the status log**

Add a concise Phase 4 final-corrections entry documenting that failed summon cleanup now retains spent IDs and that provisioning selectors are roster-authorized. Retain the non-atomic plugin-write caveat, but remove any statement that failed cleanup can silently leave a reusable identity.

- [ ] **Step 3: Commit verification documentation**

```bash
git add docs/PROJECT_STATUS.md docs/superpowers/plans/2026-09-17-phase-4-survival-final-corrections.md
git commit -m "docs: record final phase 4 corrections"
```

## Plan Self-Review

- **Coverage:** Task 1 eliminates stale-state inheritance caused by failed summon cleanup; Task 2 prevents stale records from authorizing numeric targets; Task 3 requires fresh full validation and accurate status documentation.
- **Detail:** Each task includes exact files, interfaces, regression tests, failure expectations, implementation behavior, verification, and commit boundaries.
- **Consistency:** Task 1 relies on the existing persistent `NextCompanionID`; Task 2 uses the existing authoritative `CurrentRoster` and `Provision` initialization path rather than introducing another membership store.
