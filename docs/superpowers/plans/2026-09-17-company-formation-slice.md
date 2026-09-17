# Company Roster + 3x3 Formation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand a company to one leader plus up to four persistent companions, and give it a persistent, validated 3x3 tactical formation with player commands.

**Architecture:** Extend the durable `internal/company` model with a roster (stable companion IDs, a five-member cap) and a 3x3 `Formation` grid. Extend the `modules/company` plugin with multi-instance runtime tracking, `formation` commands, legacy-record migration, and persistence. GoMud's native `internal/parties` package stays untouched; formation is company metadata, not native rank.

**Tech Stack:** Go 1.24, Go standard `testing` plus `testify` assertions, `gopkg.in/yaml.v2` module persistence, GoMud plugins/events/mobs/rooms/users.

**Spec:** `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` sections 15 and 28; `docs/ASHVEIL_GOMUD_INTEGRATION.md` Formation row; Python prototype `reference/ashveil-mud/server/engine/systems/campfire.py` (2x3, reference only) and `reference/ashveil-mud/server/engine/combat/grid.py`.

## Global Constraints

- Use permanent game-domain names: `company`, `companion`, `formation`, `expedition`; do not add `ashveil*` package or type prefixes.
- Keep GoMud's native `internal/parties` unchanged. Formation is company metadata.
- Store company data through module persistence under `_datafiles/plugin-data/`; never persist a GoMud mob `InstanceId`.
- Reuse GoMud charm tracking and ordinary-exit following. Do not add a second movement/follow framework.
- Formation rows/columns are 0-based internally and 1-based at the command layer; row 1 is the front row.
- Travel, rest, and formation must never change global game time.
- Never modify or push to `upstream` (`GoMudEngine/GoMud`).
- Run `make validate` and focused `go test` before each commit. Use `go test -race ./...` as the broad fallback because `make test` stalls in `js-lint` locally.

---

## File Structure

- `internal/company/company.go` — roster domain types, registry operations, sentinel errors; no engine imports.
- `internal/company/formation.go` — `MemberKey`, `Formation`, grid operations and their sentinel errors.
- `internal/company/company_test.go` — roster tests (updated).
- `internal/company/formation_test.go` — grid tests (new).
- `modules/company/company.go` — plugin registration, store + legacy migration, multi-companion lifecycle, command dispatch.
- `modules/company/formation.go` — formation rendering, member resolution, `formation` command handler (new).
- `modules/company/company_test.go` — lifecycle/command tests (updated).
- `modules/company/formation_test.go` — formation command tests (new).
- `modules/company/files/data-overlays/config.yaml` — `AllowedCompanionMobIDs` and `MaxCompanions`.
- `modules/company/AGENTS.md` — module boundary and lifecycle guidance (updated).

---

### Task 1: Add the 3x3 formation grid type

**Files:**
- Create: `internal/company/formation.go`
- Test: `internal/company/formation_test.go`

**Interfaces:**
- Produces `const FormationRows = 3`, `const FormationCols = 3`.
- Produces `type MemberKey string`, `const LeaderMemberKey MemberKey = "leader"`, `func CompanionMemberKey(id int) MemberKey`.
- Produces `type Formation [FormationRows][FormationCols]MemberKey`.
- Produces methods `At(row, col int) MemberKey`, `Find(key MemberKey) (int, int, bool)`, `Place(key MemberKey, row, col int) error`, `Swap(a, b MemberKey) error`, `Clear(key MemberKey)`, `Prune(valid map[MemberKey]bool)`, and `empty() bool`.
- Produces sentinel errors `ErrInvalidSlot`, `ErrSlotOccupied`, `ErrUnknownMember`.

- [ ] **Step 1: Write the failing grid tests**

Create `internal/company/formation_test.go`:

```go
package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormationPlaceAndFind(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 2, 1))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 1))
	row, col, ok := f.Find(company.LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, 2, row)
	assert.Equal(t, 1, col)
}

func TestFormationPlaceRejectsInvalidSlot(t *testing.T) {
	var f company.Formation
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, 3, 0), company.ErrInvalidSlot)
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, -1, 0), company.ErrInvalidSlot)
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, 0, 3), company.ErrInvalidSlot)
}

func TestFormationPlaceRejectsOccupiedSlot(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	err := f.Place(company.CompanionMemberKey(1), 0, 0)
	assert.ErrorIs(t, err, company.ErrSlotOccupied)
	// The rejected move must not have altered the grid.
	assert.Equal(t, company.LeaderMemberKey, f.At(0, 0))
	_, _, placed := f.Find(company.CompanionMemberKey(1))
	assert.False(t, placed)
}

func TestFormationPlaceMovesMemberClearingOldCell(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.LeaderMemberKey, 2, 2))
	assert.Equal(t, company.MemberKey(""), f.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 2))
}

func TestFormationSwapTwoMembers(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.CompanionMemberKey(1), 2, 2))
	require.NoError(t, f.Swap(company.LeaderMemberKey, company.CompanionMemberKey(1)))
	assert.Equal(t, company.CompanionMemberKey(1), f.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 2))
}

func TestFormationSwapRejectsUnknownMember(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	assert.ErrorIs(t, f.Swap(company.LeaderMemberKey, company.CompanionMemberKey(9)), company.ErrUnknownMember)
}

func TestFormationClearAndPrune(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.CompanionMemberKey(1), 1, 1))

	f.Clear(company.LeaderMemberKey)
	assert.Equal(t, company.MemberKey(""), f.At(0, 0))
	assert.Equal(t, company.CompanionMemberKey(1), f.At(1, 1))

	f.Prune(map[company.MemberKey]bool{company.LeaderMemberKey: true})
	assert.Equal(t, company.MemberKey(""), f.At(1, 1))
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/company -run 'TestFormation' -count=1`
Expected: FAIL because `Formation`, `MemberKey`, and the methods do not exist.

- [ ] **Step 3: Implement `internal/company/formation.go`**

```go
package company

import (
	"errors"
	"fmt"
)

const (
	FormationRows = 3
	FormationCols = 3
)

// MemberKey identifies a company member within a formation grid.
type MemberKey string

// LeaderMemberKey is the formation key for the company leader.
const LeaderMemberKey MemberKey = "leader"

// CompanionMemberKey returns the formation key for a companion ID.
func CompanionMemberKey(id int) MemberKey {
	return MemberKey(fmt.Sprintf("companion:%d", id))
}

// Formation is a 3x3 grid of member keys. The empty key means unoccupied.
type Formation [FormationRows][FormationCols]MemberKey

var (
	ErrInvalidSlot   = errors.New("formation slot is out of range")
	ErrSlotOccupied  = errors.New("formation slot is occupied")
	ErrUnknownMember = errors.New("member is not part of the formation")
)

// At returns the occupant of a cell, or "" when empty or out of range.
func (f *Formation) At(row, col int) MemberKey {
	if !validSlot(row, col) {
		return ""
	}
	return f[row][col]
}

// Find returns the cell occupied by key.
func (f *Formation) Find(key MemberKey) (int, int, bool) {
	if key == "" {
		return 0, 0, false
	}
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] == key {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// Place moves key to (row, col), clearing any previous cell.
func (f *Formation) Place(key MemberKey, row, col int) error {
	if key == "" {
		return ErrUnknownMember
	}
	if !validSlot(row, col) {
		return ErrInvalidSlot
	}
	if occupant := f[row][col]; occupant != "" && occupant != key {
		return ErrSlotOccupied
	}
	f.Clear(key)
	f[row][col] = key
	return nil
}

// Swap exchanges the cells of two placed members.
func (f *Formation) Swap(a, b MemberKey) error {
	if a == "" || b == "" {
		return ErrUnknownMember
	}
	if a == b {
		return nil
	}
	ar, ac, aok := f.Find(a)
	br, bc, bok := f.Find(b)
	if !aok || !bok {
		return ErrUnknownMember
	}
	f[ar][ac], f[br][bc] = f[br][bc], f[ar][ac]
	return nil
}

// Clear removes a member from every cell.
func (f *Formation) Clear(key MemberKey) {
	if key == "" {
		return
	}
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] == key {
				f[r][c] = ""
			}
		}
	}
}

// Prune removes every member not present in valid.
func (f *Formation) Prune(valid map[MemberKey]bool) {
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] != "" && !valid[f[r][c]] {
				f[r][c] = ""
			}
		}
	}
}

func (f *Formation) empty() bool {
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] != "" {
				return false
			}
		}
	}
	return true
}

func validSlot(row, col int) bool {
	return row >= 0 && row < FormationRows && col >= 0 && col < FormationCols
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./internal/company -run 'TestFormation' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/company/formation.go internal/company/formation_test.go
git commit -m "feat(company): add 3x3 formation grid"
```

---

### Task 2: Add the companion roster, cap, and dismiss semantics

**Files:**
- Modify: `internal/company/company.go`
- Test: `internal/company/company_test.go`

**Interfaces:**
- Changes `Companion` to `struct { ID int \`yaml:"id"\`; MobTemplateID int \`yaml:"mob_template_id"\` }`.
- Changes `Record` to `struct { LeaderUserID int \`yaml:"leader_user_id"\`; Companions []Companion \`yaml:"companions"\`; Formation Formation \`yaml:"formation"\` }`.
- Changes `Summon` to `Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}, maxCompanions int) (Companion, error)`.
- Produces `Dismiss(leaderUserID, companionID int) bool`, `DismissAll(leaderUserID int) int`, `Put(record Record)`, `PlaceMember(leaderUserID int, key MemberKey, row, col int) error`, `SwapMembers(leaderUserID int, a, b MemberKey) error`, `ClearMember(leaderUserID int, key MemberKey) error`.
- Produces sentinel `ErrCompanyFull`.
- Removes the single-companion `ErrCompanionAlreadyPresent` behavior (delete the sentinel and its test).

- [ ] **Step 1: Rewrite `internal/company/company_test.go` for the roster**

Replace the file body with:

```go
package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allowed58() map[int]struct{} { return map[int]struct{}{58: {}} }

func TestRegistrySummonAssignsIncrementingIDs(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 1, first.ID)
	assert.Equal(t, 2, second.ID)

	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, []company.Companion{first, second}, record.Companions)
}

func TestRegistrySummonEnforcesCap(t *testing.T) {
	registry := company.NewRegistry()
	for i := 0; i < 2; i++ {
		_, err := registry.Summon(7, 58, allowed58(), 2)
		require.NoError(t, err)
	}
	_, err := registry.Summon(7, 58, allowed58(), 2)
	assert.ErrorIs(t, err, company.ErrCompanyFull)
}

func TestRegistrySummonValidatesIDsAndAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		leaderID int
		template int
		allowed  map[int]struct{}
		wantErr  error
	}{
		{name: "zero leader", leaderID: 0, template: 58, allowed: allowed58(), wantErr: company.ErrInvalidLeader},
		{name: "zero template", leaderID: 7, template: 0, allowed: map[int]struct{}{0: {}}, wantErr: company.ErrInvalidTemplate},
		{name: "disallowed template", leaderID: 7, template: 58, allowed: map[int]struct{}{59: {}}, wantErr: company.ErrTemplateNotAllowed},
		{name: "nil allowlist", leaderID: 7, template: 58, allowed: nil, wantErr: company.ErrTemplateNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := company.NewRegistry().Summon(tt.leaderID, tt.template, tt.allowed, 4)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRegistryDismissRemovesOneAndPrunesItsFormationCells(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(second.ID), 1, 1))

	assert.True(t, registry.Dismiss(7, first.ID))
	record, ok := registry.Get(7)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, second.ID, record.Companions[0].ID)
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 0))
	assert.Equal(t, company.CompanionMemberKey(second.ID), record.Formation.At(1, 1))
	assert.False(t, registry.Dismiss(7, first.ID), "dismiss is idempotent")
}

func TestRegistryDismissLastCompanionKeepsLeaderPlacement(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 2, 1))

	require.True(t, registry.Dismiss(7, first.ID))
	record, ok := registry.Get(7)
	require.True(t, ok, "a leader placement keeps the record alive")
	assert.Empty(t, record.Companions)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(2, 1))
}

func TestRegistryDismissLastCompanionRemovesEmptyRecord(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.True(t, registry.Dismiss(7, first.ID))
	_, ok := registry.Get(7)
	assert.False(t, ok)
}

func TestRegistryDismissAllKeepsLeaderPlacement(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 1))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(second.ID), 0, 2))

	assert.Equal(t, 2, registry.DismissAll(7))
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Empty(t, record.Companions)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 1))
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 2))
}

func TestRegistryPlaceMemberValidatesMembershipAndSlots(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)

	assert.ErrorIs(t, registry.PlaceMember(7, company.CompanionMemberKey(99), 0, 0), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.PlaceMember(7, company.LeaderMemberKey, 3, 0), company.ErrInvalidSlot)

	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 0))
	assert.ErrorIs(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0), company.ErrSlotOccupied)
}

func TestRegistryLeaderFormationCreatesRecordWithoutCompanions(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 1, 1))
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(1, 1))
}

func TestRegistrySwapAndClearMembers(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 2, 2))

	require.NoError(t, registry.SwapMembers(7, company.LeaderMemberKey, company.CompanionMemberKey(first.ID)))
	record, _ := registry.Get(7)
	assert.Equal(t, company.CompanionMemberKey(first.ID), record.Formation.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(2, 2))

	require.NoError(t, registry.ClearMember(7, company.CompanionMemberKey(first.ID)))
	record, _ = registry.Get(7)
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 0))
	assert.ErrorIs(t, registry.ClearMember(7, company.CompanionMemberKey(first.ID)), company.ErrUnknownMember)
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/company -run 'TestRegistry' -count=1`
Expected: FAIL because `Summon` still returns only an error and `Record` has no `Companions`.

- [ ] **Step 3: Implement the roster in `internal/company/company.go`**

Replace the file body with:

```go
// Package company contains the durable company and companion domain model.
package company

import "errors"

var (
	ErrInvalidLeader      = errors.New("invalid leader user ID")
	ErrInvalidTemplate    = errors.New("invalid mob template ID")
	ErrTemplateNotAllowed = errors.New("mob template is not allowed")
	ErrCompanyFull        = errors.New("company is full")
)

type Companion struct {
	ID            int `yaml:"id"`
	MobTemplateID int `yaml:"mob_template_id"`
}

type Record struct {
	LeaderUserID int         `yaml:"leader_user_id"`
	Companions   []Companion `yaml:"companions"`
	Formation    Formation   `yaml:"formation"`
}

type Registry struct {
	Companies map[int]Record `yaml:"companies"`
}

func NewRegistry() *Registry {
	return &Registry{Companies: make(map[int]Record)}
}

func (r *Registry) Get(leaderUserID int) (Record, bool) {
	if r == nil {
		return Record{}, false
	}
	record, ok := r.Companies[leaderUserID]
	return record, ok
}

// Put stores a record after pruning stale formation cells. A record with no
// companions and an empty formation is removed entirely.
func (r *Registry) Put(record Record) {
	if r.Companies == nil {
		r.Companies = make(map[int]Record)
	}
	record.Formation.Prune(validMemberKeys(record))
	if len(record.Companions) == 0 && record.Formation.empty() {
		delete(r.Companies, record.LeaderUserID)
		return
	}
	r.Companies[record.LeaderUserID] = record
}

// Summon adds a companion and returns it with its assigned ID.
func (r *Registry) Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}, maxCompanions int) (Companion, error) {
	if leaderUserID <= 0 {
		return Companion{}, ErrInvalidLeader
	}
	if mobTemplateID <= 0 {
		return Companion{}, ErrInvalidTemplate
	}
	if _, ok := allowed[mobTemplateID]; !ok {
		return Companion{}, ErrTemplateNotAllowed
	}
	if maxCompanions < 1 {
		maxCompanions = 1
	}
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	if len(record.Companions) >= maxCompanions {
		return Companion{}, ErrCompanyFull
	}
	companion := Companion{ID: nextCompanionID(record), MobTemplateID: mobTemplateID}
	record.Companions = append(record.Companions, companion)
	r.Put(record)
	return companion, nil
}

// Dismiss removes one companion and its formation cells. It is idempotent.
func (r *Registry) Dismiss(leaderUserID, companionID int) bool {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return false
	}
	idx := -1
	for i, c := range record.Companions {
		if c.ID == companionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	record.Companions = append(record.Companions[:idx], record.Companions[idx+1:]...)
	record.Formation.Clear(CompanionMemberKey(companionID))
	r.Put(record)
	return true
}

// DismissAll removes every companion and returns how many were removed. The
// leader's own formation placement is preserved.
func (r *Registry) DismissAll(leaderUserID int) int {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return 0
	}
	count := len(record.Companions)
	for _, c := range record.Companions {
		record.Formation.Clear(CompanionMemberKey(c.ID))
	}
	record.Companions = nil
	r.Put(record)
	return count
}

// PlaceMember places a member in the formation, creating a leader-only record
// when the leader is placed before any companion exists.
func (r *Registry) PlaceMember(leaderUserID int, key MemberKey, row, col int) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		if key != LeaderMemberKey {
			return ErrUnknownMember
		}
		record = Record{LeaderUserID: leaderUserID}
	}
	if !validMemberKeys(record)[key] {
		return ErrUnknownMember
	}
	if err := record.Formation.Place(key, row, col); err != nil {
		return err
	}
	r.Put(record)
	return nil
}

// SwapMembers exchanges two members' formation cells.
func (r *Registry) SwapMembers(leaderUserID int, a, b MemberKey) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	valid := validMemberKeys(record)
	if !valid[a] || !valid[b] {
		return ErrUnknownMember
	}
	if err := record.Formation.Swap(a, b); err != nil {
		return err
	}
	r.Put(record)
	return nil
}

// ClearMember removes a member from the formation.
func (r *Registry) ClearMember(leaderUserID int, key MemberKey) error {
	record, ok := r.Get(leaderUserID)
	if !ok || !validMemberKeys(record)[key] {
		return ErrUnknownMember
	}
	record.Formation.Clear(key)
	r.Put(record)
	return nil
}

func nextCompanionID(record Record) int {
	maxID := 0
	for _, c := range record.Companions {
		if c.ID > maxID {
			maxID = c.ID
		}
	}
	return maxID + 1
}

func validMemberKeys(record Record) map[MemberKey]bool {
	valid := map[MemberKey]bool{LeaderMemberKey: true}
	for _, c := range record.Companions {
		valid[CompanionMemberKey(c.ID)] = true
	}
	return valid
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./internal/company -count=1`
Expected: PASS. Then run `make validate`.
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/company/company.go internal/company/company_test.go
git commit -m "feat(company): add companion roster and formation-aware dismiss"
```

---

### Task 3: Migrate legacy company records in the module store

**Files:**
- Modify: `modules/company/company.go`
- Test: `modules/company/formation_test.go` (store cases; create the file here, it grows in Task 5)

**Interfaces:**
- Consumes `domain.Registry.Put`, `domain.Companion`, `domain.Formation` from Tasks 1–2.
- Replaces `pluginStore.Load` decoding with a wire struct that accepts both `companions:` and legacy `companion:` keys.
- Preserves the existing failed-load guard and `os.ErrNotExist` empty-registry behavior.

- [ ] **Step 1: Write the failing migration tests**

Create `modules/company/formation_test.go` with:

```go
package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticStore struct {
	data    []byte
	missing bool
}

func (s staticStore) ReadBytes(string) ([]byte, error) {
	if s.missing {
		return nil, os.ErrNotExist
	}
	return s.data, nil
}

func TestPluginStoreMigratesLegacySingleCompanion(t *testing.T) {
	legacy := []byte("companies:\n  2:\n    leader_user_id: 2\n    companion:\n      mob_template_id: 58\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(legacy, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 1, record.Companions[0].ID)
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
}

func TestDecodeCompaniesReadsRosterAndFormation(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    companions:\n      - id: 1\n        mob_template_id: 58\n    formation:\n      - [\"leader\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
}

func TestDecodeCompaniesRejectsMalformedData(t *testing.T) {
	registry := domain.NewRegistry()
	assert.Error(t, decodeCompanies([]byte("companies: [this is: not valid"), registry))
	assert.Empty(t, registry.Companies)
}
```

Add `"os"` to the imports.

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./modules/company -run 'TestPluginStoreMigrates|TestDecodeCompanies' -count=1`
Expected: FAIL because `decodeCompanies` does not exist.

- [ ] **Step 3: Implement the wire decode and migration**

In `modules/company/company.go`, add above `pluginStore`:

```go
type wireRecord struct {
	LeaderUserID int                `yaml:"leader_user_id"`
	Companions   []domain.Companion `yaml:"companions"`
	Companion    *domain.Companion  `yaml:"companion"`
	Formation    domain.Formation   `yaml:"formation"`
}

type wireRegistry struct {
	Companies map[int]wireRecord `yaml:"companies"`
}

// decodeCompanies parses stored bytes, converting a legacy single-companion
// record into a roster entry with ID 1.
func decodeCompanies(data []byte, registry *domain.Registry) error {
	var wire wireRegistry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := domain.NewRegistry()
	for leaderID, wr := range wire.Companies {
		record := domain.Record{LeaderUserID: leaderID, Companions: wr.Companions, Formation: wr.Formation}
		if len(record.Companions) == 0 && wr.Companion != nil {
			legacy := *wr.Companion
			if legacy.ID == 0 {
				legacy.ID = 1
			}
			record.Companions = []domain.Companion{legacy}
		}
		loaded.Put(record)
	}
	*registry = *loaded
	return nil
}
```

Replace `pluginStore.Load` with:

```go
func (s pluginStore) Load(registry *domain.Registry) error {
	// ReadIntoStruct currently discards YAML decoding errors. Decode here so
	// unreadable company data cannot become an empty, writable registry.
	data, err := s.plug.ReadBytes("companies")
	if errors.Is(err, os.ErrNotExist) {
		*registry = *domain.NewRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeCompanies(data, registry)
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./modules/company -run 'TestPluginStoreMigrates|TestDecodeCompanies' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/company/company.go modules/company/formation_test.go
git commit -m "feat(company): migrate legacy records to companion roster"
```

---

### Task 4: Track multiple companion instances at runtime

**Files:**
- Modify: `modules/company/company.go`
- Modify: `modules/company/company_test.go`

**Interfaces:**
- Replaces `liveByLeader map[int]int` with `instances map[int]map[int]int` (leader user ID -> companion ID -> instance ID).
- Produces `instance(leader, companionID int) (int, bool)`, `setInstance(leader, companionID, instanceID int)`, `clearInstance(leader, companionID int)`, `companionForInstance(instanceID int) (int, int, bool)`.
- Produces `maxCompanions() int` reading `MaxCompanions` config (default 4) via `configInt(any) (int, bool)`.
- Updates `summon`, `status`, `dismiss`, `restoreForLeader`, `onMobDeath`, and `userCommand` for the roster.

- [ ] **Step 1: Update the test harness and lifecycle tests**

In `modules/company/company_test.go`:

- Add `"strconv"` to imports if not present; keep `maps`.
- Add a deep-clone helper and use it in `fakeStore`:

```go
func cloneRegistry(in domain.Registry) domain.Registry {
	out := domain.Registry{Companies: map[int]domain.Record{}}
	for leader, record := range in.Companies {
		clone := record
		clone.Companions = append([]domain.Companion(nil), record.Companions...)
		out.Companies[leader] = clone
	}
	return out
}
```

Replace `maps.Clone` calls in `fakeStore.Load`/`Save` with `cloneRegistry`. Remove the now-unused `maps` import.

- Replace `newTestModule` with:

```go
func newTestModule(registry domain.Registry, runtime Runtime) *CompanyModule {
	return &CompanyModule{registry: registry, instances: map[int]map[int]int{}, runtime: runtime, store: &fakeStore{}}
}
```

- Rewrite the record literals in every test from `Companion: domain.Companion{MobTemplateID: 58}` to `Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}`.
- Replace every `module.liveByLeader[7] = 99` with `module.setInstance(7, 1, 99)`, `module.liveByLeader[7]` with a `module.instance(7, 1)` lookup, and `assert.Empty(t, module.liveByLeader)` with `assert.Empty(t, module.instances)`.
- Change `module.summon(7, 12, "58")` assertions to accept two return values: `_, err := module.summon(7, 12, "58")`.
- Change `module.dismiss(7)` calls to `module.dismiss(7, "1")` (or `"all"` where the test means all). Add explicit new tests:

```go
func TestSummonRespectsMaxCompanions(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 100}
	module := newTestModule(*domain.NewRegistry(), runtime)
	for i := 0; i < 4; i++ {
		runtime.nextInstanceID++
		_, err := module.summon(7, 12, "58")
		require.NoError(t, err)
	}
	_, err := module.summon(7, 12, "58")
	assert.ErrorIs(t, err, domain.ErrCompanyFull)
}

func TestRestoreForLeaderSpawnsEveryCompanion(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 200}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 2, runtime.spawnCalls)
	_, ok := module.instance(7, 1)
	assert.True(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
}

func TestMobDeathClearsOnlyMatchingCompanionInstance(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	require.Equal(t, events.Continue, module.onMobDeath(events.MobDeath{InstanceId: 91}))
	_, ok := module.instance(7, 1)
	assert.False(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
	record, exists := module.registry.Get(7)
	assert.True(t, exists)
	assert.Len(t, record.Companions, 2)
}

func TestDismissOneCompanionDetachesOnlyItsInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	assert.Equal(t, 1, runtime.detachCalls)
	_, ok := module.instance(7, 1)
	assert.False(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
	record, _ := module.registry.Get(7)
	assert.Len(t, record.Companions, 1)
	assert.Equal(t, 2, record.Companions[0].ID)
}

func TestDismissAllCompanionsDetachesEveryInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	_, err := module.dismiss(7, "all")
	require.NoError(t, err)
	assert.Equal(t, 2, runtime.detachCalls)
	assert.Empty(t, module.instances)
}

func TestConfigIntAcceptsYAMLAndStringValues(t *testing.T) {
	for _, value := range []any{4, int64(4), float64(4), "4"} {
		got, ok := configInt(value)
		require.True(t, ok)
		assert.Equal(t, 4, got)
	}
	_, ok := configInt("not-a-number")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./modules/company -run 'TestSummon|TestRestore|TestMobDeath|TestDismiss|TestConfigInt' -count=1`
Expected: FAIL because `instances`, `configInt`, and the new `dismiss`/`summon` signatures do not exist.

- [ ] **Step 3: Implement multi-companion tracking in `modules/company/company.go`**

Replace the `CompanyModule` struct and add helpers:

```go
type CompanyModule struct {
	plug         *plugins.Plugin
	store        Store
	registry     domain.Registry
	instances    map[int]map[int]int
	runtime      Runtime
	loadErr      error
}

func (m *CompanyModule) instance(leaderUserID, companionID int) (int, bool) {
	if m.instances == nil {
		return 0, false
	}
	instanceID, ok := m.instances[leaderUserID][companionID]
	return instanceID, ok
}

func (m *CompanyModule) setInstance(leaderUserID, companionID, instanceID int) {
	if m.instances == nil {
		m.instances = map[int]map[int]int{}
	}
	if m.instances[leaderUserID] == nil {
		m.instances[leaderUserID] = map[int]int{}
	}
	m.instances[leaderUserID][companionID] = instanceID
}

func (m *CompanyModule) clearInstance(leaderUserID, companionID int) {
	if m.instances == nil {
		return
	}
	delete(m.instances[leaderUserID], companionID)
	if len(m.instances[leaderUserID]) == 0 {
		delete(m.instances, leaderUserID)
	}
}

func (m *CompanyModule) companionForInstance(instanceID int) (int, int, bool) {
	for leaderUserID, byCompanion := range m.instances {
		for companionID, tracked := range byCompanion {
			if tracked == instanceID {
				return leaderUserID, companionID, true
			}
		}
	}
	return 0, 0, false
}

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

func (m *CompanyModule) maxCompanions() int {
	if m.plug != nil {
		if n, ok := configInt(m.plug.Config.Get("MaxCompanions")); ok && n > 0 {
			return n
		}
	}
	return 4
}
```

Update `init()` to initialize `instances: map[int]map[int]int{}` and register the `formation` command:

```go
func init() {
	m := &CompanyModule{plug: plugins.New("company", "1.0"), instances: map[int]map[int]int{}}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.runtime = nativeRuntime{}
	m.plug.AddUserCommand("company", m.userCommand, false, false)
	m.plug.AddUserCommand("formation", m.formationCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("company: save", "error", err)
		}
	})
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.MobDeath{}, m.onMobDeath)
}
```

Replace `summon`:

```go
func (m *CompanyModule) summon(leaderUserID, roomID int, selector string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return companyUsage, fmt.Errorf("company: companion selector is required")
	}
	templateID, err := m.resolveTemplateID(selector)
	if err != nil {
		return "", err
	}
	companion, err := m.registry.Summon(leaderUserID, templateID, m.allowedTemplates(), m.maxCompanions())
	if err != nil {
		return "", err
	}
	instanceID, err := m.runtime.Spawn(leaderUserID, roomID, templateID)
	if err != nil {
		m.registry.Dismiss(leaderUserID, companion.ID)
		return "", err
	}
	if err := m.save(); err != nil {
		m.runtime.Detach(leaderUserID, instanceID)
		m.registry.Dismiss(leaderUserID, companion.ID)
		return "", err
	}
	m.setInstance(leaderUserID, companion.ID, instanceID)
	return fmt.Sprintf("Companion summoned: %s (#%d).", templateName(templateID, selector), companion.ID), nil
}

func (m *CompanyModule) resolveTemplateID(selector string) (int, error) {
	if id, err := strconv.Atoi(selector); err == nil {
		return id, nil
	}
	if id, ok := m.runtime.ResolveTemplate(strings.ToLower(selector)); ok {
		return id, nil
	}
	return 0, fmt.Errorf("company: unknown mob template %q", selector)
}
```

Replace `status`:

```go
func (m *CompanyModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions."
	}
	lines := []string{fmt.Sprintf("Company companions (%d/%d):", len(record.Companions), m.maxCompanions())}
	for _, c := range record.Companions {
		state := "awaiting restoration"
		if instanceID, tracked := m.instance(leaderUserID, c.ID); tracked {
			if m.runtime.IsAttached(leaderUserID, instanceID) {
				state = "present"
			} else if !m.runtime.IsLive(instanceID) {
				m.clearInstance(leaderUserID, c.ID)
			}
		}
		lines = append(lines, fmt.Sprintf("  #%d %s (%s)", c.ID, templateName(c.MobTemplateID, strconv.Itoa(c.MobTemplateID)), state))
	}
	return strings.Join(lines, "\n")
}
```

Replace `dismiss` with specific + all:

```go
func (m *CompanyModule) dismiss(leaderUserID int, selector string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	selector = strings.TrimSpace(strings.ToLower(selector))
	if selector == "all" {
		return m.dismissAll(leaderUserID)
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "No companions.", nil
	}
	companion, ok := resolveCompanion(record, selector)
	if !ok {
		return "", fmt.Errorf("company: no companion matches %q", selector)
	}
	if !m.registry.Dismiss(leaderUserID, companion.ID) {
		return "", fmt.Errorf("company: companion #%d is no longer in the company", companion.ID)
	}
	if err := m.save(); err != nil {
		m.registry.Put(record)
		return "", err
	}
	if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(leaderUserID, instanceID)
		}
		m.clearInstance(leaderUserID, companion.ID)
	}
	return fmt.Sprintf("Companion dismissed: %s (#%d).", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID), nil
}

func (m *CompanyModule) dismissAll(leaderUserID int) (string, error) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions.", nil
	}
	count := m.registry.DismissAll(leaderUserID)
	if err := m.save(); err != nil {
		m.registry.Put(record)
		return "", err
	}
	for _, companion := range record.Companions {
		if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
			if m.runtime.IsLive(instanceID) {
				m.runtime.Detach(leaderUserID, instanceID)
			}
			m.clearInstance(leaderUserID, companion.ID)
		}
	}
	return fmt.Sprintf("Dismissed %d companion(s).", count), nil
}
```

Replace `restoreForLeader`, `onMobDeath`, and the `userCommand` dispatch (`dismiss` now takes a selector):

```go
func (m *CompanyModule) restoreForLeader(leaderUserID, roomID int) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	var firstErr error
	for _, companion := range record.Companions {
		if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
			if m.runtime.IsAttached(leaderUserID, instanceID) {
				continue
			}
			if m.runtime.IsLive(instanceID) {
				m.runtime.Detach(leaderUserID, instanceID)
			}
			m.clearInstance(leaderUserID, companion.ID)
		}
		instanceID, err := m.runtime.Spawn(leaderUserID, roomID, companion.MobTemplateID)
		if err != nil {
			m.clearInstance(leaderUserID, companion.ID)
			if firstErr == nil {
				firstErr = fmt.Errorf("company: restore leader %d companion %d: %w", leaderUserID, companion.ID, err)
			}
			continue
		}
		m.setInstance(leaderUserID, companion.ID, instanceID)
	}
	return firstErr
}

func (m *CompanyModule) onMobDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Cancel
	}
	if leaderUserID, companionID, found := m.companionForInstance(evt.InstanceId); found {
		m.clearInstance(leaderUserID, companionID)
	}
	return events.Continue
}
```

In `userCommand`, replace the `dismiss` case:

```go
	case "dismiss":
		if len(args) < 2 {
			user.SendText(companyUsage)
			return true, nil
		}
		text, err := m.dismiss(user.UserId, strings.Join(args[1:], " "))
		if err != nil {
			return true, err
		}
		user.SendText(text)
```

Update `companyUsage` to `"Usage: company summon <mob-id-or-name> | company status | company dismiss <member|all>"`.

Add `resolveCompanion` (used by `dismiss` and by formation in Task 5):

```go
// resolveCompanion matches a companion by exact name, then substring, then #id.
func resolveCompanion(record domain.Record, selector string) (domain.Companion, bool) {
	if strings.HasPrefix(selector, "#") {
		id, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
		if err != nil {
			return domain.Companion{}, false
		}
		for _, c := range record.Companions {
			if c.ID == id {
				return c, true
			}
		}
		return domain.Companion{}, false
	}
	var exact, partial []domain.Companion
	for _, c := range record.Companions {
		name := strings.ToLower(templateName(c.MobTemplateID, ""))
		if name == selector {
			exact = append(exact, c)
		} else if strings.Contains(name, selector) {
			partial = append(partial, c)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(partial) == 1 {
		return partial[0], true
	}
	return domain.Companion{}, false
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./modules/company -count=1`
Expected: PASS. Then `make validate`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/company/company.go modules/company/company_test.go
git commit -m "feat(company): track multiple companion instances"
```

---

### Task 5: Add the `formation` command

**Files:**
- Create: `modules/company/formation.go`
- Test: `modules/company/formation_test.go`
- Modify: `modules/company/company_test.go` (helper only if needed)

**Interfaces:**
- Consumes `resolveCompanion`, `templateName`, `m.registry`, `m.save` from Task 4.
- Produces `renderFormation(leaderUserID int) string` and `formationCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)`.
- Produces `parseSlot(raw string) (int, error)` returning a 1-based slot.

- [ ] **Step 1: Write the failing formation command tests**

Append to `modules/company/formation_test.go`:

```go
func formationModule(t *testing.T) (*CompanyModule, *fakeStore) {
	t.Helper()
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	store := module.store.(*fakeStore)
	return module, store
}

func TestFormationCommandMovesLeaderAndPersists(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	handled, err := module.formationCommand("move leader 1 1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, domain.LeaderMemberKey, store.saved.Companies[7].Formation.At(0, 0))
}

func TestFormationCommandRejectsOccupiedSlotAndUnknownMember(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	_, err := module.formationCommand("move leader 1 1", user, nil, 0)
	require.NoError(t, err)
	_, err = module.formationCommand("move #1 1 1", user, nil, 0)
	assert.ErrorIs(t, err, domain.ErrSlotOccupied)
	_, err = module.formationCommand("move #9 2 2", user, nil, 0)
	assert.Error(t, err)
	assert.Equal(t, 1, store.saveCalls, "rejected moves must not persist")
}

func TestFormationCommandSwapAndClear(t *testing.T) {
	module, _ := formationModule(t)
	user := users.NewUserRecord(7, 1)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	require.NoError(t, module.registry.PlaceMember(7, domain.CompanionMemberKey(1), 2, 2))

	handled, err := module.formationCommand("swap leader #1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.CompanionMemberKey(1), record.Formation.At(0, 0))

	handled, err = module.formationCommand("clear #1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ = module.registry.Get(7)
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 0))
}

func TestFormationCommandSaveFailureRollsBack(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	store.saveErr = errors.New("disk full")
	_, err := module.formationCommand("move leader 1 1", user, nil, 0)
	assert.ErrorIs(t, err, store.saveErr)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 0), "failed save must not keep the move")
}

func TestFormationCommandRendersGridAndUnplaced(t *testing.T) {
	module, _ := formationModule(t)
	user := users.NewUserRecord(7, 1)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	text := module.renderFormation(7)
	assert.Contains(t, text, "front")
	assert.Contains(t, text, "leader")
	assert.Contains(t, text, "training dummy", "unplaced companions are listed")
}

func TestParseSlotRejectsOutOfRange(t *testing.T) {
	_, err := parseSlot("0")
	assert.Error(t, err)
	_, err = parseSlot("4")
	assert.Error(t, err)
	n, err := parseSlot("2")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
```

Add `"errors"` and `"users"` to the imports of `modules/company/formation_test.go`.

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./modules/company -run 'TestFormation|TestParseSlot' -count=1`
Expected: FAIL because `formationCommand`, `renderFormation`, and `parseSlot` do not exist.

- [ ] **Step 3: Implement `modules/company/formation.go`**

```go
package company

import (
	"fmt"
	"strconv"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const formationUsage = "Usage: formation | formation move <member> <row> <col> | formation swap <member-a> <member-b> | formation clear <member>"

var rowLabels = [domain.FormationRows]string{"front", "mid  ", "back "}

func (m *CompanyModule) renderFormation(leaderUserID int) string {
	record, _ := m.registry.Get(leaderUserID)
	lines := []string{"Company formation (3x3, row 1 = front):"}
	lines = append(lines, "         col 1        col 2        col 3")
	placed := map[domain.MemberKey]bool{}
	for r := 0; r < domain.FormationRows; r++ {
		cells := make([]string, 0, domain.FormationCols)
		for c := 0; c < domain.FormationCols; c++ {
			key := record.Formation.At(r, c)
			if key == "" {
				cells = append(cells, fmt.Sprintf("%-12s", "------"))
				continue
			}
			placed[key] = true
			cells = append(cells, fmt.Sprintf("%-12s", m.memberName(leaderUserID, key)))
		}
		lines = append(lines, fmt.Sprintf("%s [ %s ]", rowLabels[r], strings.Join(cells, " | ")))
	}

	unplaced := []string{}
	if !placed[domain.LeaderMemberKey] {
		unplaced = append(unplaced, "leader")
	}
	for _, companion := range record.Companions {
		if !placed[domain.CompanionMemberKey(companion.ID)] {
			unplaced = append(unplaced, fmt.Sprintf("%s (#%d)", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID))
		}
	}
	if len(unplaced) > 0 {
		lines = append(lines, "Unplaced: "+strings.Join(unplaced, ", "))
	}
	lines = append(lines, formationUsage)
	return strings.Join(lines, "\n")
}

func (m *CompanyModule) memberName(leaderUserID int, key domain.MemberKey) string {
	if key == domain.LeaderMemberKey {
		if user := users.GetByUserId(leaderUserID); user != nil {
			return user.Character.Name
		}
		return "leader"
	}
	record, _ := m.registry.Get(leaderUserID)
	for _, companion := range record.Companions {
		if domain.CompanionMemberKey(companion.ID) == key {
			return fmt.Sprintf("%s(#%d)", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID)
		}
	}
	return string(key)
}

func (m *CompanyModule) resolveMemberKey(leaderUserID int, selector string) (domain.MemberKey, error) {
	s := strings.ToLower(strings.TrimSpace(selector))
	if s == "" {
		return "", fmt.Errorf("company: a member is required")
	}
	if s == "leader" || s == "me" || s == "self" {
		return domain.LeaderMemberKey, nil
	}
	if strings.HasPrefix(s, "#") {
		id, err := strconv.Atoi(strings.TrimPrefix(s, "#"))
		if err != nil {
			return "", fmt.Errorf("company: invalid companion id %q", selector)
		}
		return domain.CompanionMemberKey(id), nil
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", fmt.Errorf("company: no company member matches %q", selector)
	}
	companion, ok := resolveCompanion(record, s)
	if !ok {
		return "", fmt.Errorf("company: no company member matches %q", selector)
	}
	return domain.CompanionMemberKey(companion.ID), nil
}

func parseSlot(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > domain.FormationCols {
		return 0, fmt.Errorf("company: slot %q must be a number 1-%d", raw, domain.FormationCols)
	}
	return n, nil
}

func (m *CompanyModule) formationCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	if err := m.persistenceAvailable(); err != nil {
		return true, err
	}
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) == 0 {
		user.SendText(m.renderFormation(user.UserId))
		return true, nil
	}
	switch args[0] {
	case "move":
		if len(args) != 4 {
			user.SendText(formationUsage)
			return true, nil
		}
		key, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		row, err := parseSlot(args[2])
		if err != nil {
			return true, err
		}
		col, err := parseSlot(args[3])
		if err != nil {
			return true, err
		}
		before, _ := m.registry.Get(user.UserId)
		if err := m.registry.PlaceMember(user.UserId, key, row-1, col-1); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			m.registry.Put(before)
			return true, err
		}
		user.SendText(fmt.Sprintf("Placed %s at row %d, column %d.", m.memberName(user.UserId, key), row, col))
	case "swap":
		if len(args) != 3 {
			user.SendText(formationUsage)
			return true, nil
		}
		a, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		b, err := m.resolveMemberKey(user.UserId, args[2])
		if err != nil {
			return true, err
		}
		before, _ := m.registry.Get(user.UserId)
		if err := m.registry.SwapMembers(user.UserId, a, b); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			m.registry.Put(before)
			return true, err
		}
		user.SendText("Formation positions swapped.")
	case "clear":
		if len(args) != 2 {
			user.SendText(formationUsage)
			return true, nil
		}
		key, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		before, _ := m.registry.Get(user.UserId)
		if err := m.registry.ClearMember(user.UserId, key); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			m.registry.Put(before)
			return true, err
		}
		user.SendText(fmt.Sprintf("Removed %s from the formation.", m.memberName(user.UserId, key)))
	default:
		user.SendText(formationUsage)
	}
	return true, nil
}
```

- [ ] **Step 4: Run the focused tests to verify they pass**

Run: `go test ./modules/company -count=1`
Expected: PASS. Then `make validate`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/company/formation.go modules/company/formation_test.go
git commit -m "feat(company): add formation commands"
```

---

### Task 6: Add `MaxCompanions` config and update module guidance

**Files:**
- Modify: `modules/company/files/data-overlays/config.yaml`
- Modify: `modules/company/AGENTS.md`

- [ ] **Step 1: Add the config default**

Replace `modules/company/files/data-overlays/config.yaml` with:

```yaml
AllowedCompanionMobIDs:
  - 58
MaxCompanions: 4
```

- [ ] **Step 2: Update `modules/company/AGENTS.md`**

Replace the file body with:

```markdown
# Company Module Guide

`MobTemplateID` and companion `ID` are durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores every companion for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

The company roster is one leader plus up to `MaxCompanions` (default 4) companions, for a five-member cap. Durable records are written immediately when commands change company state.

Player-facing management: `company summon <mob-id-or-name>`, `company status`, `company dismiss <member|all>`, and `formation` / `formation move <member> <row> <col>` / `formation swap <a> <b>` / `formation clear <member>`. Formation rows and columns are 1-based to players; row 1 is the front row.

Legacy single-`companion` records migrate to `companions[0]` with ID 1 on load. Formation cells for dismissed companions are pruned automatically. Travel, rest, and formation must never change global game time.
```

- [ ] **Step 3: Verify config parses and tests still pass**

Run: `go test ./modules/company -count=1`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add modules/company/files/data-overlays/config.yaml modules/company/AGENTS.md
git commit -m "docs(company): document roster and formation rules"
```

---

### Task 7: Full verification and project status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [ ] **Step 1: Run the complete focused and broad suites**

```bash
make generate
make validate
go test -race ./internal/company ./modules/company -count=1
go test -race ./...
```

Expected: PASS. `make generate` must leave `modules/all-modules.go` unchanged (no new module package). If `make test` is attempted, record the known `js-lint` stall and rely on the race suite.

- [ ] **Step 2: Perform the local acceptance sequence (optional but recommended)**

Run `make run`, log in with a disposable local user, and execute:

```text
company summon 58
company summon 58
company status
formation
formation move leader 1 1
formation move #1 1 2
formation swap leader #1
company dismiss #1
formation
company dismiss all
```

Expected: two companions appear with IDs 1 and 2; `formation` renders the 3x3 grid and lists unplaced members; moves and swaps update the grid; dismissing prunes that companion's cells; `dismiss all` clears companions but keeps the leader placement. Restart and confirm companions restore with the same formation. Confirm world time is unchanged.

- [ ] **Step 3: Update `docs/PROJECT_STATUS.md`**

Move Phase 3 to Complete, set the next phase to Phase 4 (Survival State), update the `HEAD` and origin-sync lines, and add a work-log entry recording what was done, why, and that handoff Phase 3 completed. Keep it short.

- [ ] **Step 4: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record phase 3 completion"
```

---

## Self-Review

- **Spec coverage:** roster cap five (Task 2), one member per cell and collision rejection (Tasks 1–2), leader placeable like any member (Task 2), formation display (Task 5), formation survives save/reload (Tasks 3, 7), companion dismissal prunes formation (Task 2), no global time mutation (no clock APIs used anywhere).
- **Placeholder scan:** no TBD/TODO; every code step contains complete code.
- **Type consistency:** `MemberKey`, `CompanionMemberKey`, `LeaderMemberKey`, `Formation`, `Record.Companions`, `instances`, `resolveCompanion`, `configInt`, `parseSlot`, and `renderFormation` are defined before later use; `Summon` returns `(Companion, error)` everywhere.
