# Phase 3 Invariant Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Guarantee the Phase 3 five-character company limit and repair duplicate formation occupants safely when persistent company data is loaded.

**Architecture:** Keep the company-size rule authoritative in `internal/company`, so no caller can create more than four companions. Clamp configuration at the module boundary as well, keeping status output consistent with the actual limit. Extend the existing `Formation.Prune` normalization path to retain only the first valid occurrence of each member in row-major order; `Registry.Put` already applies that path during decoding and every durable update.

**Tech Stack:** Go 1.24, Go standard `testing`, `testify`, YAML module persistence.

**Spec:** `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` §28, `docs/superpowers/plans/2026-09-17-company-formation-slice.md`, and the Phase 3 review findings.

## Global Constraints

- A company is one leader plus at most four companions, for a hard five-character maximum.
- Formation rows and columns remain 0-based internally and 1-based in commands; row 1 is the front row.
- Every formation member may occupy at most one cell. On persisted duplicate occupants, retain the first cell in row-major order and clear later copies.
- Keep company data under module persistence; never persist a mob `InstanceId`.
- Do not modify `internal/parties`, add a movement system, or mutate global game time.
- Run focused race tests and `make validate` before each commit. Use `go test -race ./...` as the broad Go verification fallback.

---

## File Structure

- `internal/company/company.go` — defines the hard maximum and enforces it in the registry, independently of module configuration.
- `internal/company/formation.go` — normalizes duplicate formation occupants while retaining the first valid cell.
- `internal/company/company_test.go` — proves oversized limits cannot create a sixth party member and duplicate saved cells normalize deterministically.
- `internal/company/formation_test.go` — proves `Prune` preserves the first valid cell and clears later duplicate cells.
- `modules/company/company.go` — clamps configured limits before they are displayed or supplied to the registry.
- `modules/company/company_test.go` — covers configured upper-limit clamping.
- `modules/company/formation_test.go` — proves YAML load routes duplicate occupants through domain normalization.
- `docs/PROJECT_STATUS.md` — records the corrective Phase 3 commit and its verification.

---

### Task 1: Make the domain model enforce both roster and formation invariants

**Files:**
- Modify: `internal/company/company.go:6-11,58-80`
- Modify: `internal/company/formation.go:103-112`
- Modify: `internal/company/company_test.go:157-174`
- Modify: `internal/company/formation_test.go`

**Interfaces:**
- Produces `const MaxCompanions = 4`.
- `Registry.Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}, maxCompanions int) (Companion, error)` continues to accept a requested limit, but clamps it to `1..MaxCompanions` before evaluating the roster.
- `Formation.Prune(valid map[MemberKey]bool)` removes unknown keys and duplicate valid keys, retaining the earliest row-major placement.

- [ ] **Step 1: Add failing upper-bound and duplicate-normalization tests**

Append to `internal/company/company_test.go`:

```go
func TestRegistrySummonClampsCapToPartyMaximum(t *testing.T) {
	registry := company.NewRegistry()
	for i := 0; i < company.MaxCompanions; i++ {
		_, err := registry.Summon(7, 58, allowed58(), company.MaxCompanions+10)
		require.NoError(t, err)
	}
	_, err := registry.Summon(7, 58, allowed58(), company.MaxCompanions+10)
	assert.ErrorIs(t, err, company.ErrCompanyFull)
}

func TestRegistryPutKeepsFirstDuplicateFormationOccupant(t *testing.T) {
	registry := company.NewRegistry()
	record := company.Record{LeaderUserID: 7}
	record.Formation[0][0] = company.LeaderMemberKey
	record.Formation[1][1] = company.LeaderMemberKey
	registry.Put(record)

	got, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, got.Formation.At(0, 0))
	assert.Equal(t, company.MemberKey(""), got.Formation.At(1, 1))
}
```

Append to `internal/company/formation_test.go`:

```go
func TestFormationPruneKeepsFirstDuplicateValidMember(t *testing.T) {
	var f company.Formation
	f[0][2] = company.LeaderMemberKey
	f[2][0] = company.LeaderMemberKey

	f.Prune(map[company.MemberKey]bool{company.LeaderMemberKey: true})

	assert.Equal(t, company.LeaderMemberKey, f.At(0, 2))
	assert.Equal(t, company.MemberKey(""), f.At(2, 0))
}
```

- [ ] **Step 2: Run the new tests and confirm the current branch fails them**

Run:

```bash
go test ./internal/company -run 'TestRegistrySummonClampsCapToPartyMaximum|TestRegistryPutKeepsFirstDuplicateFormationOccupant|TestFormationPruneKeepsFirstDuplicateValidMember' -count=1
```

Expected: FAIL because an upper limit greater than four is honored and `Prune` does not remove duplicate valid keys.

- [ ] **Step 3: Add the hard cap and deterministic formation normalization**

In `internal/company/company.go`, add the exported domain limit and use it before the existing company-full check:

```go
const MaxCompanions = 4

func clampCompanionLimit(limit int) int {
	if limit < 1 {
		return 1
	}
	if limit > MaxCompanions {
		return MaxCompanions
	}
	return limit
}
```

Replace the current lower-bound-only logic in `Registry.Summon` with:

```go
maxCompanions = clampCompanionLimit(maxCompanions)
```

In `internal/company/formation.go`, replace `Prune` with row-major normalization:

```go
// Prune removes unknown members and duplicate valid members. When a persisted
// formation repeats a member, the earliest row-major placement is retained.
func (f *Formation) Prune(valid map[MemberKey]bool) {
	seen := make(map[MemberKey]bool)
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			key := f[r][c]
			if key == "" {
				continue
			}
			if !valid[key] || seen[key] {
				f[r][c] = ""
				continue
			}
			seen[key] = true
		}
	}
}
```

- [ ] **Step 4: Run focused domain tests**

Run:

```bash
go test -race ./internal/company -count=1
```

Expected: PASS. This confirms the existing formation behavior plus lower and upper roster clamping.

- [ ] **Step 5: Commit the domain invariants**

```bash
git add internal/company/company.go internal/company/formation.go internal/company/company_test.go internal/company/formation_test.go
git commit -m "fix(company): enforce roster and formation invariants"
```

---

### Task 2: Clamp module configuration and verify decoded persistence behavior

**Files:**
- Modify: `modules/company/company.go:147-154`
- Modify: `modules/company/company_test.go:473-481`
- Modify: `modules/company/formation_test.go:25-36`

**Interfaces:**
- Produces `maxCompanionsFromConfig(raw any) int`, which returns `domain.MaxCompanions` for invalid/non-positive config, returns a valid value in `1..domain.MaxCompanions`, and clamps any oversized value to `domain.MaxCompanions`.
- `(*CompanyModule).maxCompanions() int` delegates to `maxCompanionsFromConfig` when a plugin config exists and otherwise returns `domain.MaxCompanions`.
- `decodeCompanies(data []byte, registry *domain.Registry) error` remains the load entry point; its `loaded.Put(record)` normalization clears duplicate occupants.

- [ ] **Step 1: Add failing module-boundary tests**

Append to `modules/company/company_test.go`:

```go
func TestMaxCompanionsFromConfigClampsToDomainLimit(t *testing.T) {
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig(domain.MaxCompanions+1))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig("999"))
	assert.Equal(t, 2, maxCompanionsFromConfig(2))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig(0))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig("invalid"))
}
```

Append to `modules/company/formation_test.go`:

```go
func TestDecodeCompaniesKeepsFirstDuplicateFormationOccupant(t *testing.T) {
	data := []byte("companies:\n  2:\n    formation:\n      - [\"leader\", \"leader\", \"\"]\n      - [\"\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 1))
}
```

- [ ] **Step 2: Run the new module tests and confirm they fail before implementation**

Run:

```bash
go test ./modules/company -run 'TestMaxCompanionsFromConfigClampsToDomainLimit|TestDecodeCompaniesKeepsFirstDuplicateFormationOccupant' -count=1
```

Expected: FAIL because `maxCompanionsFromConfig` does not yet exist; the decoding case will pass only after Task 1 is present.

- [ ] **Step 3: Centralize module configuration clamping**

In `modules/company/company.go`, replace `maxCompanions` with:

```go
func maxCompanionsFromConfig(raw any) int {
	n, ok := configInt(raw)
	if !ok || n < 1 {
		return domain.MaxCompanions
	}
	if n > domain.MaxCompanions {
		return domain.MaxCompanions
	}
	return n
}

func (m *CompanyModule) maxCompanions() int {
	if m.plug != nil {
		return maxCompanionsFromConfig(m.plug.Config.Get("MaxCompanions"))
	}
	return domain.MaxCompanions
}
```

Do not change `decodeCompanies`: it already calls `loaded.Put(record)`, and Task 1 makes that call normalize duplicate occupants before the loaded registry becomes active.

- [ ] **Step 4: Run all company package tests**

Run:

```bash
go test -race ./internal/company ./modules/company -count=1
make validate
```

Expected: both commands exit 0. Confirm `company status` will now show at most `4` as its configured companion limit, matching the registry’s hard boundary.

- [ ] **Step 5: Commit the module boundary and migration coverage**

```bash
git add modules/company/company.go modules/company/company_test.go modules/company/formation_test.go
git commit -m "fix(company): clamp config and normalize loaded formations"
```

---

### Task 3: Perform broad verification and record the corrective work

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

**Interfaces:**
- No production interface changes. The status log records the two restored Phase 3 invariants and the commands actually run.

- [ ] **Step 1: Run broad verification**

Run:

```bash
make generate
make validate
go test -race ./internal/company ./modules/company -count=1
go test -race ./...
```

Expected: every command exits 0. Confirm `make generate` leaves `modules/all-modules.go` unchanged.

- [ ] **Step 2: Update the Phase 3 work log**

Add a concise dated entry under `## Recent work log` in `docs/PROJECT_STATUS.md` stating:

```markdown
### Phase 3 invariant corrections (2026-09-17)

- **What:** Hard-capped companies at four companions plus their leader, and normalized duplicate persisted formation occupants by keeping the first row-major cell.
- **Why:** Restores Phase 3’s five-character cap and one-cell-per-member invariants even when module configuration or stored YAML is invalid.
- **Verification:** `make generate`, `make validate`, and `go test -race ./...` passed.
```

Update the `HEAD` line only after the final documentation commit exists, using its actual commit SHA and subject.

- [ ] **Step 3: Commit the status update**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record phase 3 invariant corrections"
```

---

## Self-Review

- **Spec coverage:** Task 1 enforces the five-character cap and one-cell-per-member invariant in the domain. Task 2 applies the same cap to configuration and proves YAML loading normalizes duplicate occupants. Task 3 validates the complete repository and records the correction.
- **Normalization policy:** Valid duplicates are handled deterministically: the first cell encountered from row 0/column 0 through row 2/column 2 remains; later copies are cleared. Unknown member keys continue to be cleared.
- **No scope expansion:** The plan changes no formation combat, movement, party engine, persisted runtime IDs, or global game time behavior.
