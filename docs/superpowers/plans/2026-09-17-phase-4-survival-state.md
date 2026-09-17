# Phase 4 Survival State Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver persistent, per-company-member hunger, thirst, and fatigue, with manual leader-backpack provisioning and a stable API for later travel and rest.

**Architecture:** `internal/survival` owns normalized needs, threshold bands, and a narrow provider interface that lets native commands invoke the module-owned durable registry without importing modules. `modules/survival` owns registry persistence, roster projection, and status rendering; `modules/company` notifies the provider after committed roster changes. Native `eat`/`drink` preserve GoMud lookup, subtype, uses, ownership events, and buffs, but invoke the provider before consuming an item with survival metadata.

**Tech Stack:** Go 1.24, standard `testing`, `testify`, GoMud plugins/YAML persistence, `make generate`, `make validate`.

**Spec:** `docs/superpowers/specs/2026-09-17-survival-state-design.md`

## Global Constraints

- Hunger, thirst, and fatigue are individual integer values in `0..100`; `100` is fully supplied/rested.
- State changes only through explicit APIs. Do not add idle/offline drain, timers, `NewTurn` listeners, or any mutation of shared game time.
- No Phase 4 health, combat, action-point, movement, or travel-speed penalty.
- Use stable leader/companion keys; never persist or identify survival state by a mob `InstanceId`.
- Leader-backpack items may provision the leader or a current companion. Do not add companion inventory, cargo, auto-provisioning, weight, or capacity.
- Preserve ordinary GoMud food/drink compatibility for specs with zero survival metadata.
- Camp/rest/sleep commands and real-time recovery scheduling are Phase 7. Cargo and any automatic provisioning policy are Phase 9.
- Preserve the existing untracked `docs/superpowers/plans/2026-09-17-phase-3-invariant-fixes.md`; do not stage it in Phase 4 commits.

---

## File Structure

- `internal/survival/survival.go` — value types, normalization, bands, explicit mutation service, and provider registration contract.
- `internal/survival/survival_test.go` — domain normalization, operations, and threshold-crossing coverage.
- `internal/items/itemspec.go` — optional item-spec nutrition/hydration metadata.
- `internal/items/items_test.go` — item metadata loading/default compatibility.
- `internal/usercommands/eat.go` and `drink.go` — call survival provisioning before consuming a qualifying item.
- `internal/usercommands/eat_test.go` and `drink_test.go` — native command behavior, selector forwarding, and no-consume failure coverage.
- `modules/survival/survival.go` — plugin store, roster-bound registry, provider implementation, and `survival` command.
- `modules/survival/survival_test.go` — persistence, roster validation, provisioning, and rendering tests.
- `modules/survival/files/data-overlays/config.yaml` — enabled module metadata/configuration if required by the plugin convention.
- `modules/company/company.go` and `modules/company/company_test.go` — notify survival only after durable summon/dismiss succeeds; cover failure ordering.
- `modules/all-modules.go` — generated import wiring after `make generate`.
- `_datafiles/items/...` — one existing/default edible and one drinkable item updated with metadata for integration coverage, following the existing item-file layout found during implementation.
- `docs/PROJECT_STATUS.md` — record Phase 4 completion and verification when all tasks land.

---

### Task 1: Build the pure survival domain and command-provider seam

**Files:**
- Create: `internal/survival/survival.go`
- Create: `internal/survival/survival_test.go`

**Interfaces:**
- Produces `MemberKey`, `LeaderMemberKey`, `CompanionMemberKey(id int)`, `Needs`, `Band`, `Change`, and `Exertion`.
- Produces `Normalize(Needs) Needs`, `BandFor(value int) Band`, and a registry/service with `ApplyExertion`, `ApplyRestRecovery`, `ConsumeFood`, `ConsumeWater`, and `NeedsFor`.
- Produces a GoMud-free `Provisioner` interface for `usercommands`: `Provision(leaderUserID int, selector string, benefit Benefit) (ProvisionResult, error)`.

- [ ] **Step 1: Write failing domain tests**

Create tests for all five bands, default needs, decode normalization of negative and oversized values, food/water caps, fatigue-only rest recovery, exertion costs, zero/negative rejection, and exact boundary changes such as `26 -> 25`.

```go
func TestConsumeFoodCapsNeedAndReportsCrossing(t *testing.T) {
	r := survival.NewRegistry()
	r.Ensure(7, survival.LeaderMemberKey)
	r.PutNeeds(7, survival.LeaderMemberKey, survival.Needs{Hunger: 25, Thirst: 100, Fatigue: 100})

	change, _, err := r.ConsumeFood(7, survival.LeaderMemberKey, 90, 0)
	require.NoError(t, err)
	assert.Equal(t, survival.BandCritical, change.Before)
	assert.Equal(t, survival.BandFull, change.After)
	assert.Equal(t, 100, r.MustNeedsFor(7, survival.LeaderMemberKey).Hunger)
}
```

- [ ] **Step 2: Run the focused tests and verify failure**

Run: `go test ./internal/survival -count=1`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the model with no engine imports**

Use integer arithmetic only. Define the bands exactly as the spec: `0`, `1..25`, `26..50`, `51..75`, `76..100`. Make a leader-ID plus member-key registry map; `Ensure` initializes `{100,100,100}`; every public mutation validates IDs/keys and rejects non-positive costs/benefits. Return a `Change` for each potentially changed need.

```go
func clamp(value int) int { return max(0, min(100, value)) }

func BandFor(value int) Band {
	switch value = clamp(value); {
	case value == 0:
		return BandDepleted
	case value <= 25:
		return BandCritical
	case value <= 50:
		return BandLow
	case value <= 75:
		return BandSteady
	default:
		return BandFull
	}
}
```

Keep provider registration a tiny synchronized package-level adapter (`SetProvisioner`, `Provision`) that returns a typed unavailable error before the survival module is loaded. Do not put user, item, room, clock, or plugin types in this package.

- [ ] **Step 4: Run focused race tests**

Run: `go test -race ./internal/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the domain seam**

```bash
git add internal/survival/survival.go internal/survival/survival_test.go
git commit -m "feat(survival): add durable needs domain"
```

### Task 2: Add data-driven nutrition and hydration metadata

**Files:**
- Modify: `internal/items/itemspec.go`
- Modify: `internal/items/items_test.go`
- Modify: exact existing edible/drinkable `_datafiles/items/` YAML fixtures selected after inspecting the data tree

**Interfaces:**
- Produces `ItemSpec.Nutrition int` (`yaml:"nutrition,omitempty"`) and `ItemSpec.Hydration int` (`yaml:"hydration,omitempty"`).
- Existing zero-valued item specs retain ordinary use/buff behavior.

- [ ] **Step 1: Add failing metadata/default tests**

Test a YAML item spec with both fields, an ordinary legacy edible spec without fields, and a drink spec with hydration. Assert parsed values and zero defaults.

- [ ] **Step 2: Run targeted item tests and verify failure**

Run: `go test ./internal/items -run 'Test.*(Nutrition|Hydration)' -count=1`

Expected: FAIL because `ItemSpec` has no fields.

- [ ] **Step 3: Extend the spec and update two safe sample items**

Add only these fields beside the other item behavior metadata:

```go
Nutrition int `yaml:"nutrition,omitempty"`
Hydration int `yaml:"hydration,omitempty"`
```

Select existing default-world edible and drinkable data files rather than inventing a second item catalog. Give the edible positive `nutrition` and the drink positive `hydration`; do not change type, subtype, uses, buffs, name, or ordinary gameplay data.

- [ ] **Step 4: Verify data and package behavior**

Run:

```bash
go test ./internal/items -count=1
make validate
```

Expected: both commands exit 0.

- [ ] **Step 5: Commit item metadata**

```bash
git add internal/items/itemspec.go internal/items/items_test.go _datafiles/items
git commit -m "feat(items): add survival consumable metadata"
```

### Task 3: Implement module persistence, roster projection, and status rendering

**Files:**
- Create: `modules/survival/survival.go`
- Create: `modules/survival/survival_test.go`
- Create: `modules/survival/files/data-overlays/config.yaml`

**Interfaces:**
- Produces `EnsureCompanyMember(leaderUserID, companionID int) error`, `RemoveCompanyMember(leaderUserID, companionID int) error`, and `RemoveAllCompanyMembers(leaderUserID int) error` for company lifecycle calls.
- Implements `survival.Provisioner` and registers it during module initialization.
- Registers `survival` as a player command.
- Persists `map[int]map[survival.MemberKey]survival.Needs` under the module, never under a mob instance.

- [ ] **Step 1: Write failing module tests**

Cover: absent file creates an empty registry; malformed bytes make mutations unavailable and do not overwrite data; YAML values normalize; provisioning accepts `leader`, `me`, `self`, `#2`, and an unambiguous current companion; it rejects unknown/ambiguous/dismissed companions; and status shows leader plus current roster with correct labels.

```go
func TestProvisionRejectsDismissedCompanionWithoutChangingState(t *testing.T) {
	m := newTestModule(/* roster contains no #2 */)
	before := m.registry.Clone()
	_, err := m.Provision(7, "#2", survival.Benefit{Nutrition: 30})
	require.ErrorIs(t, err, survival.ErrUnknownMember)
	assert.Equal(t, before, m.registry)
}
```

- [ ] **Step 2: Run focused module tests and verify failure**

Run: `go test ./modules/survival -count=1`

Expected: FAIL because the module does not exist.

- [ ] **Step 3: Implement the plugin with failure-aware persistence**

Follow `modules/company`'s `Store`, `ReadBytes`, `WriteStruct`, load-error, and test-double patterns. Keep one module instance owning the registry. Persist before reporting a successful provisioning result; restore the in-memory snapshot if write fails. The leader is always valid; companions are valid only when supplied by the company lifecycle projection. `survival` status must be read-only and may defensively prune stale entries before rendering.

Render one line per member with name/selector, numeric values, and labels. Preserve a companion's durable needs through native re-spawn; remove its state only after successful company dismissal notification.

- [ ] **Step 4: Run focused module race tests**

Run: `go test -race ./modules/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the survival module**

```bash
git add modules/survival
git commit -m "feat(survival): persist company survival state"
```

### Task 4: Make native eat and drink provision safely

**Files:**
- Modify: `internal/usercommands/eat.go`
- Modify: `internal/usercommands/drink.go`
- Create or modify: focused `internal/usercommands/*_test.go` files following existing command-test conventions

**Interfaces:**
- `eat <item> [member]` forwards `Nutrition` and `Hydration` as food benefit.
- `drink <item> [member]` forwards `Hydration` as water benefit.
- A provisioning failure leaves the item, uses, ownership events, and buffs unchanged.

- [ ] **Step 1: Add failing command tests**

Use a test provisioner registered through the internal seam. Cover self-targeting, forwarding `#2`, unqualified zero-metadata items retaining legacy consumption, provider errors preserving the item, and success applying provider output before the existing use/buff path.

```go
func TestEatDoesNotConsumeWhenSurvivalProvisionFails(t *testing.T) {
	survival.SetProvisioner(failingProvisioner{err: survival.ErrPersistenceUnavailable})
	t.Cleanup(func() { survival.SetProvisioner(nil) })
	user := userWithEdible(t, 17, 1)

	_, err := Eat("ration #2", user, testRoom(), 0)
	require.ErrorIs(t, err, survival.ErrPersistenceUnavailable)
	assert.Len(t, user.Character.Items, 1)
}
```

- [ ] **Step 2: Run the new tests and verify failure**

Run: `go test ./internal/usercommands -run 'Test(Eat|Drink).*Survival' -count=1`

Expected: FAIL because the commands ignore survival metadata and selectors.

- [ ] **Step 3: Parse an optional final member selector without breaking item matching**

Do not use a naïve `strings.Fields` split because existing item names may contain spaces. Reuse `util.SplitButRespectQuotes`; resolve the longest item-name portion through `FindInBackpack`, then treat only a remaining final token as a member selector. If no matching item is found with a selector, retry the whole input as the legacy item name before producing the existing missing-item message.

For qualifying metadata, call `survival.Provision` after item/subtype validation and before `UseItem`, ownership events, or buffs. For zero metadata, keep the legacy path exactly. On provider success, append target and band-crossing text to the existing personal message; room message remains ordinary consumption text.

- [ ] **Step 4: Run command and regression tests**

Run:

```bash
go test -race ./internal/usercommands -count=1
go test -race ./internal/items ./internal/survival ./internal/usercommands -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit command integration**

```bash
git add internal/usercommands/eat.go internal/usercommands/drink.go internal/usercommands/*_test.go
git commit -m "feat(survival): provision company members with supplies"
```

### Task 5: Synchronize company lifecycle without leaking stale needs

**Files:**
- Modify: `modules/company/company.go`
- Modify: `modules/company/company_test.go`
- Modify: `modules/survival/survival_test.go`

**Interfaces:**
- Successful `summon` ensures default state for its assigned companion ID.
- Successful single/all dismissal removes state for exactly the removed companion IDs.
- A survival persistence failure causes the company command to return an error and restores the company registry to its pre-command snapshot; no native runtime companion is attached/detached as a successful command effect.

- [ ] **Step 1: Add lifecycle failure-order tests**

Test successful summon/dismiss synchronization and injected survival write failures. Assert that the company record and survival registry both retain their prior states on a failed cross-module update, and that a later newly assigned companion cannot display removed state.

- [ ] **Step 2: Run the new lifecycle tests and verify failure**

Run: `go test ./modules/company ./modules/survival -run 'Test.*Survival' -count=1`

Expected: FAIL because company currently does not notify survival.

- [ ] **Step 3: Add narrow lifecycle calls at the durable command boundaries**

Place the survival call after the company registry mutation has passed its own save precondition but before native spawn/detach side effects and success messaging. Retain existing company rollback behavior on either company or survival persistence error. For `dismiss all`, remove each actual companion ID captured from the pre-mutation record; never infer IDs from runtime instances.

Do not add a global event type, make `modules/survival` import `modules/company`, or persist state to a mob. Keep the dependency one-way through the `internal/survival` interface.

- [ ] **Step 4: Run focused cross-module race tests**

Run: `go test -race ./modules/company ./modules/survival -count=1`

Expected: PASS.

- [ ] **Step 5: Commit lifecycle consistency**

```bash
git add modules/company/company.go modules/company/company_test.go modules/survival/survival_test.go
git commit -m "feat(company): synchronize companion survival state"
```

### Task 6: Wire, verify, and document the completed phase

**Files:**
- Modify: generated `modules/all-modules.go` through `make generate`
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/superpowers/plans/2026-09-17-phase-4-survival-state.md` to mark executed steps only after their evidence exists

**Interfaces:**
- The generated module import list includes `modules/survival`.
- Project status marks Phase 4 complete only after all verification succeeds.

- [ ] **Step 1: Generate module wiring**

Run: `make generate`

Expected: `modules/all-modules.go` gains the sorted blank import for `modules/survival`; no hand edit.

- [ ] **Step 2: Run full validation**

Run:

```bash
make validate
go test -race ./...
```

Expected: both commands exit 0. If `make test` is attempted and its documented JavaScript-lint environment blocker recurs, record exact output and retain `go test -race ./...` as the broad verification evidence.

- [ ] **Step 3: Perform a focused live acceptance pass if local server prerequisites are available**

Start the server using the documented local procedure. As a player with one companion and test consumables: run `survival`; use `eat <item> #1`; use `drink <item> #1`; verify labels change; log out/in and verify durability; dismiss the companion and verify it disappears from `survival`. Stop the server cleanly. If prerequisites are unavailable, record that fact rather than claiming live validation.

- [ ] **Step 4: Update status and commit the final integration**

Update `docs/PROJECT_STATUS.md` with the completed Phase 4 scope, persistence/command behavior, explicit deferrals (rest/sleep Phase 7; cargo Phase 9), commits, and actual verification results.

```bash
git add modules/all-modules.go docs/PROJECT_STATUS.md docs/superpowers/plans/2026-09-17-phase-4-survival-state.md
git commit -m "docs: record phase 4 survival completion"
```

## Plan Self-Review

- **Spec coverage:** Tasks 1–5 cover pure state, all explicit API operations, thresholds, stable identity, persistence, command provisioning, status rendering, and dismissal cleanup. Task 6 covers generation, broad verification, live acceptance, and project status. Idle drain, penalties, travel, camp/sleep commands, cargo, weather, terrain, and automation are excluded explicitly.
- **Detail scan:** The plan provides concrete commands, interfaces, and test cases. It instructs the implementer to inspect the exact existing default item files before selecting two metadata updates, avoiding invented paths or content IDs.
- **Interface consistency:** `MemberKey`, `Needs`, `Band`, `Change`, and `Provisioner` originate in Task 1; item metadata in Task 2; module/provider behavior in Task 3; commands consume that provider in Task 4; company lifecycle synchronizes through that same seam in Task 5.
