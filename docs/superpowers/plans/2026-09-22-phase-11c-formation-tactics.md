# Phase 11c Formation Tactics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Status: fully implemented and committed on `phase-11c-formation-tactics` — every step below is checked off.**

**Goal:** Implement the column-occupancy reach model (plain melee = column
frontmost only, polearm/innate Reach = frontmost-or-one-behind, ranged =
any depth), lateral column range (±1), and front-row interception as pure,
fully tested functions over `company.Formation`, plus the `Reach` trait
schema fields those functions need, plus a read-only adjacency-query
command — as independently reviewable pieces, deferring the
`NewRound_DoCombat.go` wiring itself to a follow-up task (see the scope
decision below).

**Architecture:** A new GoMud-free domain package, `internal/formationcombat`,
mirrors 11a (`internal/mobparty`) and 11b (`internal/engagement`): pure
functions over `company.Formation` plus a caller-supplied `alive
map[company.MemberKey]bool`, with no dependency on live `*mobs.Mob`/
`*characters.Character`. `internal/combat` (which already imports
`internal/items`/`internal/mobs`/`internal/characters`) gets one new
adapter function, `ResolveReach`, that inspects a live combatant's equipped
weapon (and, for mobs, an innate flag) and returns a
`formationcombat.Reach`. `items.ItemSpec` and `mobs.Mob` each gain one new
`Reach bool` field, following the exact precedent of `ItemSpec.Weight`/
`Mob.Hostile`. `modules/company/formation.go` gets one new read-only
subcommand, `formation reach <member>`, that lists which of the formation's
own members that member could legally target — the "adjacency queries
exposed read-only" acceptance criterion — built entirely from data
`modules/company` already owns (no new cross-module dependency).

**Scope decision — `NewRound_DoCombat.go` wiring is NOT in this plan:**
research before writing this plan found that wiring `Legal`/
`InterceptFrontRow` into the real per-round attack-resolution loop needs a
lookup this codebase does not have yet: given a live mob instance ID (an
attacking or defending companion), find which leader's company it belongs
to and its `MemberKey` within that company's `Formation`. Today that
mapping exists only as `modules/company.CompanyModule`'s *unexported*
`companionForInstance` method, backed by an unexported field — and
`internal/hooks` (where `NewRound_DoCombat.go` lives) does not and
structurally should not import `modules/company` (modules depend on
`internal/`, not the reverse — the established pattern for this kind of
cross-boundary query, per Phase 5's `survival.CompanyService` and Phase 8's
`weather.Provider`, is a **query-seam interface defined in `internal/`,
implemented by the module, registered at startup**). Building that seam,
plus the equivalent fresh-per-round `mobparty.Assemble` lookup for the
*enemy* side (11a's `Party`/`Formation` is never cached — see the 11a
work-log entry), plus threading both through all four attack-direction call
sites in a 1150-line hot combat-loop file, is a materially different kind
of change from the pure functions this plan delivers: it's a real,
player-visible combat-behavior change touching a shared multiplayer
invariant, exactly the kind of thing `internal/combat/AGENTS.md` says to
keep narrowly scoped and `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` §50 flags
for escalated care. Shipping it in the same pass as the domain layer would
make both harder to review and roll back independently. This plan
therefore delivers the tested, reusable `formationcombat` package, the
`Reach` schema, and the safe, purely additive read-only query command; the
`NewRound_DoCombat.go` wiring (and the new company-formation query seam it
needs) is tracked as an explicit follow-up in `docs/PROJECT_STATUS.md`,
not silently dropped. This mirrors 11b's own precedent of shipping a tested
domain layer and deferring hook wiring for a documented reason.

**Tech Stack:** Go, `testify` (`assert`/`require`), no new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-22-phase-11c-formation-tactics-design.md`
(and the shared prior-art/decisions in
`docs/superpowers/specs/2026-09-22-phase-11-formation-combat-design.md`).

## Global Constraints

- Never advances `gametime`, round count, or damage math.
- Formation is read-only during combat this phase — no rearrangement
  mid-fight (this plan doesn't touch combat at all, so this is trivially
  satisfied, but stays the standing invariant for the deferred wiring too).
- A member with no resolvable formation row fails closed: `FrontmostOccupant`/
  `Legal`/`InterceptFrontRow` return `ok=false`/`false` rather than
  guessing, for any key not found in the formation or not alive.
- `internal/formationcombat` stays GoMud-free: `company.Formation` and
  `company.MemberKey` (both already engine-agnostic types) are its only
  non-stdlib dependency; it never imports `internal/mobs`/
  `internal/characters`/`internal/items`.
- `Reach` (the new `ItemSpec`/`Mob` fields) defaults to `false`/no-reach —
  every existing item and mob YAML file is unaffected without a data pass,
  matching the `Weight`/`Nutrition`/`Hydration` zero-value precedent.

---

## File Structure

- Create: `internal/formationcombat/formationcombat.go` — `Reach`,
  `FrontmostOccupant`, `InLateralRange`, `InReachDepth`, `Legal`,
  `InterceptFrontRow`, `LegalTargets`. Pure domain logic, no GoMud imports.
- Create: `internal/formationcombat/formationcombat_test.go` — the worked
  example, reach-extension, ranged-ignores-depth, interception, and
  lateral-range-edge-column tests from the spec's acceptance criteria.
- Modify: `internal/items/itemspec.go` — add `Reach bool` to `ItemSpec`.
- Modify: `internal/mobs/mobs.go` — add `Reach bool` to `Mob`.
- Create: `internal/combat/reach.go` — `ResolveReach`.
- Create: `internal/combat/reach_test.go` — tests for `ResolveReach`.
- Modify: `modules/company/formation.go` — add the `formation reach
  <member>` subcommand.

## Task 1: `internal/formationcombat` domain package

**Files:**
- Create: `internal/formationcombat/formationcombat.go`
- Test: `internal/formationcombat/formationcombat_test.go`

**Interfaces:**
- Consumes: `company.Formation`, `company.MemberKey`, `company.FormationRows`,
  `company.FormationCols` (`internal/company`, already exists).
- Produces (for `internal/combat`'s `ResolveReach` adapter and the future
  hooks-integration follow-up):
  - `type Reach int` with `ReachNone`, `ReachExtended`, `ReachAny`
  - `func FrontmostOccupant(f company.Formation, col int, alive map[company.MemberKey]bool) (row int, ok bool)`
  - `func InLateralRange(attackerCol, defenderCol int) bool`
  - `func InReachDepth(frontmostRow, targetRow int, reach Reach) bool`
  - `func Legal(attackerCol int, f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool, reach Reach) bool`
  - `func InterceptFrontRow(f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool) (interceptor company.MemberKey, ok bool)`
  - `func LegalTargets(attackerCol int, f company.Formation, alive map[company.MemberKey]bool, reach Reach) []company.MemberKey`

- [x] **Step 1: Write the failing tests**

Create `internal/formationcombat/formationcombat_test.go`:

```go
package formationcombat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyA = company.MemberKey("a")
	keyB = company.MemberKey("b")
	keyC = company.MemberKey("c")
	keyD = company.MemberKey("d")
)

// workedExample builds the spec's own worked example: A at (front, col 1),
// B at (back, col 1), C at (back, col 2).
func workedExample(t *testing.T) company.Formation {
	t.Helper()
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 1))
	require.NoError(t, f.Place(keyB, 2, 1))
	require.NoError(t, f.Place(keyC, 2, 2))
	return f
}

func allAlive(keys ...company.MemberKey) map[company.MemberKey]bool {
	m := make(map[company.MemberKey]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func TestFrontmostOccupantSkipsDeadAndEmptyCells(t *testing.T) {
	f := workedExample(t)

	row, ok := formationcombat.FrontmostOccupant(f, 1, allAlive(keyA, keyB))
	require.True(t, ok)
	assert.Equal(t, 0, row, "A (front) blocks column 1")

	row, ok = formationcombat.FrontmostOccupant(f, 1, allAlive(keyB)) // A dead
	require.True(t, ok)
	assert.Equal(t, 2, row, "B becomes frontmost once A is gone")

	_, ok = formationcombat.FrontmostOccupant(f, 0, allAlive(keyA, keyB, keyC))
	assert.False(t, ok, "column 0 is empty in the worked example")
}

func TestInLateralRangeEdgeColumns(t *testing.T) {
	cases := []struct {
		attackerCol, defenderCol int
		want                     bool
	}{
		{0, 0, true}, {0, 1, true}, {0, 2, false},
		{1, 0, true}, {1, 1, true}, {1, 2, true},
		{2, 0, false}, {2, 1, true}, {2, 2, true},
	}
	for _, c := range cases {
		got := formationcombat.InLateralRange(c.attackerCol, c.defenderCol)
		assert.Equal(t, c.want, got, "attackerCol=%d defenderCol=%d", c.attackerCol, c.defenderCol)
	}
}

func TestInReachDepthNoneOnlyFrontmost(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachNone))
	assert.False(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachNone))
	assert.False(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachNone))
}

func TestInReachDepthExtendedFrontOrMiddleNotBack(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachExtended))
	assert.True(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachExtended))
	assert.False(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachExtended))
}

func TestInReachDepthAnyIgnoresDepth(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachAny))
	assert.True(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachAny))
	assert.True(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachAny))
}

func TestLegalWorkedExamplePlainMelee(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	// Attacker in column 1: legal against A and C, illegal against B
	// (blocked by A in the same column).
	assert.True(t, formationcombat.Legal(1, f, keyA, alive, formationcombat.ReachNone))
	assert.True(t, formationcombat.Legal(1, f, keyC, alive, formationcombat.ReachNone))
	assert.False(t, formationcombat.Legal(1, f, keyB, alive, formationcombat.ReachNone))
}

func TestLegalSelfHealsWhenBlockerDies(t *testing.T) {
	f := workedExample(t)
	aliveWithA := allAlive(keyA, keyB, keyC)
	aliveWithoutA := allAlive(keyB, keyC) // A died; formation itself is unchanged

	assert.False(t, formationcombat.Legal(1, f, keyB, aliveWithA, formationcombat.ReachNone),
		"B is blocked while A is alive")
	assert.True(t, formationcombat.Legal(1, f, keyB, aliveWithoutA, formationcombat.ReachNone),
		"B becomes legal automatically once A dies, with no formation change")
}

func TestLegalReachExtendedHitsMiddleNotBack(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 0)) // front
	require.NoError(t, f.Place(keyD, 1, 0)) // middle
	require.NoError(t, f.Place(keyB, 2, 0)) // back
	alive := allAlive(keyA, keyB, keyD)

	assert.True(t, formationcombat.Legal(0, f, keyA, alive, formationcombat.ReachExtended))
	assert.True(t, formationcombat.Legal(0, f, keyD, alive, formationcombat.ReachExtended), "extended reach hits the middle occupant")
	assert.False(t, formationcombat.Legal(0, f, keyB, alive, formationcombat.ReachExtended), "still blocked from the back slot")
}

func TestLegalRangedIgnoresColumnDepth(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 0)) // front
	require.NoError(t, f.Place(keyD, 1, 0)) // middle
	require.NoError(t, f.Place(keyB, 2, 0)) // back
	alive := allAlive(keyA, keyB, keyD)

	assert.True(t, formationcombat.Legal(0, f, keyA, alive, formationcombat.ReachAny))
	assert.True(t, formationcombat.Legal(0, f, keyD, alive, formationcombat.ReachAny))
	assert.True(t, formationcombat.Legal(0, f, keyB, alive, formationcombat.ReachAny), "ranged ignores blocking entirely")
}

func TestLegalOutOfLateralRangeIsIllegal(t *testing.T) {
	f := workedExample(t) // A/B/C all in columns 1-2
	alive := allAlive(keyA, keyB, keyC)

	// An attacker in column 2 is out of lateral range of column... wait,
	// use a target in column 0 to prove the edge case instead.
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0))
	assert.False(t, formationcombat.Legal(2, g, keyA, allAlive(keyA), formationcombat.ReachAny),
		"column 2 attacker is out of lateral range of column 0")
	_ = alive
}

func TestLegalDeadOrMissingTargetIsIllegal(t *testing.T) {
	f := workedExample(t)
	assert.False(t, formationcombat.Legal(1, f, keyA, allAlive(keyB, keyC), formationcombat.ReachAny), "A is dead")
	assert.False(t, formationcombat.Legal(1, f, company.MemberKey("ghost"), allAlive(keyA, keyB, keyC), formationcombat.ReachAny), "not in the formation at all")
}

func TestInterceptFrontRowRedirectsToSameColumnFrontRowMember(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	interceptor, ok := formationcombat.InterceptFrontRow(f, keyB, alive)
	require.True(t, ok)
	assert.Equal(t, keyA, interceptor)
}

func TestInterceptFrontRowNoInterceptionWhenTargetIsAlreadyFrontRow(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	_, ok := formationcombat.InterceptFrontRow(f, keyA, alive)
	assert.False(t, ok, "attacking the front row directly needs no redirect")
}

func TestInterceptFrontRowOriginalTargetStandsWithNoLivingFrontRow(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyB, keyC) // A dead: column 1's front slot is empty

	_, ok := formationcombat.InterceptFrontRow(f, keyB, alive)
	assert.False(t, ok, "no living front-row member in that column: attack proceeds against B")
}

func TestLegalTargetsListsExactlyTheLegalMembers(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	targets := formationcombat.LegalTargets(1, f, alive, formationcombat.ReachNone)

	assert.ElementsMatch(t, []company.MemberKey{keyA, keyC}, targets)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/formationcombat/... -v`
Expected: FAIL — the package doesn't exist yet.

- [x] **Step 3: Write the implementation**

Create `internal/formationcombat/formationcombat.go`:

```go
// Package formationcombat implements the column-occupancy reach model,
// lateral column range, and front-row interception over company.Formation
// (reused directly from internal/company for both the player company side
// and, via mobparty, the enemy party side).
//
// This package is deliberately GoMud-free: every function takes a
// company.Formation plus a caller-supplied "who's still alive" map, never
// a live *mobs.Mob/*characters.Character. Formation is locked once combat
// starts (no mid-fight rearrangement anywhere in this codebase), so the
// only thing that changes reach/legality round to round is which members
// are alive — the self-healing behavior described in the Phase 11c design
// doc falls directly out of that: nothing here ever mutates a Formation or
// clears a caller's stored target, it only answers "is this legal right
// now" fresh from current alive state.
package formationcombat

import "github.com/GoMudEngine/GoMud/internal/company"

// Reach determines how deep into a column an attacker can strike.
type Reach int

const (
	ReachNone     Reach = iota // plain melee: column's frontmost occupant only
	ReachExtended              // polearm or innate: frontmost, or one behind it
	ReachAny                   // ranged: any occupied depth in the column
)

// FrontmostOccupant returns the row of the nearest-to-front living
// occupant of a column: the cell nearest row 0 whose MemberKey is present
// in the formation and alive[key] is true. ok is false if the column has
// no living occupant (empty, or every occupant is dead).
func FrontmostOccupant(f company.Formation, col int, alive map[company.MemberKey]bool) (row int, ok bool) {
	for r := 0; r < company.FormationRows; r++ {
		key := f.At(r, col)
		if key == "" {
			continue
		}
		if alive != nil && !alive[key] {
			continue
		}
		return r, true
	}
	return 0, false
}

// InLateralRange reports whether defenderCol is within attackerCol's own
// column index plus or minus one.
func InLateralRange(attackerCol, defenderCol int) bool {
	diff := attackerCol - defenderCol
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1
}

// InReachDepth reports whether targetRow is reachable given the column's
// current frontmost occupied row and an attacker's Reach class.
func InReachDepth(frontmostRow, targetRow int, reach Reach) bool {
	switch reach {
	case ReachAny:
		return true
	case ReachExtended:
		return targetRow == frontmostRow || targetRow == frontmostRow+1
	default:
		return targetRow == frontmostRow
	}
}

// Legal reports whether an attacker in attackerCol may currently target
// targetKey within f, given who's alive and the attacker's Reach. This is
// the predicate Phase 11b's AssignTarget consumes as its injected
// LegalFunc (adapted to that package's Combatant shape by the caller).
func Legal(attackerCol int, f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool, reach Reach) bool {
	if targetKey == "" {
		return false
	}
	if alive != nil && !alive[targetKey] {
		return false
	}

	targetRow, targetCol, found := f.Find(targetKey)
	if !found {
		return false
	}
	if !InLateralRange(attackerCol, targetCol) {
		return false
	}

	frontmostRow, ok := FrontmostOccupant(f, targetCol, alive)
	if !ok {
		return false
	}
	return InReachDepth(frontmostRow, targetRow, reach)
}

// InterceptFrontRow reports the front-row member in the same column as an
// attack aimed at a non-front-row member of f, if one is alive. ok is
// false when no interception applies (targetKey is already front row, not
// found, or that column's front slot has no living occupant) — the caller
// should then proceed against the original target.
func InterceptFrontRow(f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool) (interceptor company.MemberKey, ok bool) {
	targetRow, targetCol, found := f.Find(targetKey)
	if !found || targetRow == 0 {
		return "", false
	}

	frontKey := f.At(0, targetCol)
	if frontKey == "" {
		return "", false
	}
	if alive != nil && !alive[frontKey] {
		return "", false
	}
	return frontKey, true
}

// LegalTargets returns every member of f that an attacker in attackerCol
// could currently legally target, given alive and reach — the read-only
// adjacency query the design doc's acceptance criteria call for.
func LegalTargets(attackerCol int, f company.Formation, alive map[company.MemberKey]bool, reach Reach) []company.MemberKey {
	var legal []company.MemberKey
	for r := 0; r < company.FormationRows; r++ {
		for c := 0; c < company.FormationCols; c++ {
			key := f.At(r, c)
			if key == "" {
				continue
			}
			if Legal(attackerCol, f, key, alive, reach) {
				legal = append(legal, key)
			}
		}
	}
	return legal
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/formationcombat/... -v`
Expected: PASS, all fourteen tests.

- [x] **Step 5: `go vet`/`gofmt`, full build, full race suite**

```bash
gofmt -l internal/formationcombat && go vet ./internal/formationcombat/...
go build ./...
go test -race ./...
```
Expected: no `gofmt`/`vet` output, clean build, full suite green with no
regressions (this package has zero engine dependencies).

- [x] **Step 6: Commit**

```bash
git add internal/formationcombat/formationcombat.go internal/formationcombat/formationcombat_test.go
git commit -m "feat(formationcombat): add Phase 11c column-occupancy reach/legality logic"
```

## Task 2: `Reach` trait schema and `ResolveReach` adapter

**Files:**
- Modify: `internal/items/itemspec.go` (add `Reach bool` to `ItemSpec`)
- Modify: `internal/mobs/mobs.go` (add `Reach bool` to `Mob`)
- Create: `internal/combat/reach.go`
- Test: `internal/combat/reach_test.go`

**Interfaces:**
- Consumes: `items.ItemSpec.Reach`, `items.ItemSpec.Subtype`,
  `items.Shooting`, `mobs.Mob.Reach` (schema additions this task makes),
  `characters.Character.Equipment.Weapon` (existing), `formationcombat.Reach`
  (Task 1).
- Produces: `func ResolveReach(c *characters.Character, innateReach bool) formationcombat.Reach`
  — the future hooks-integration follow-up calls this once per attacker
  per round, passing `mob.Reach` for a mob attacker or `false` for a
  player attacker (players have no innate reach, only weapon-granted).

- [x] **Step 1: Add the `Reach` field to `ItemSpec`**

In `internal/items/itemspec.go`, add to the `ItemSpec` struct (after the
existing `Weight` field, matching its comment style):

```go
	Reach           bool              `yaml:"reach,omitempty"`       // Polearm-class weapon: extends melee reach to a column's frontmost-or-one-behind occupant (see Phase 11c)
```

- [x] **Step 2: Add the `Reach` field to `Mob`**

In `internal/mobs/mobs.go`, add to the `Mob` struct next to `Hostile`:

```go
	Reach           bool     `yaml:"reach,omitempty"`          // Innate reach (e.g. a large/long-limbed monster), independent of any weapon
```

- [x] **Step 3: Write the failing test for `ResolveReach`**

Create `internal/combat/reach_test.go`:

```go
package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func TestResolveReachUnarmedNoInnateReachIsNone(t *testing.T) {
	var c characters.Character
	assert.Equal(t, formationcombat.ReachNone, combat.ResolveReach(&c, false))
}

func TestResolveReachInnateReachWithNoWeaponIsExtended(t *testing.T) {
	var c characters.Character
	assert.Equal(t, formationcombat.ReachExtended, combat.ResolveReach(&c, true))
}

func TestResolveReachShootingWeaponIsAny(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9001, Subtype: items.Shooting}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachAny, combat.ResolveReach(&c, false))
}

func TestResolveReachPolearmWeaponIsExtended(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9002, Subtype: items.Stabbing, Reach: true}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachExtended, combat.ResolveReach(&c, false))
}

func TestResolveReachOrdinaryWeaponIsNone(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9003, Subtype: items.Slashing}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachNone, combat.ResolveReach(&c, false))
}
```

Note: `items.Item` carries an optional per-instance `Spec *ItemSpec`
override field (`internal/items/items.go:43`) that `Item.GetSpec()`
(`internal/items/items.go:253-260`) prefers over the global registry when
set — exactly what these tests need, with no global-registry
setup/teardown (no `LoadDataFiles`, no on-disk writes, no naming collision
with real item IDs) required at all.

- [x] **Step 4: Run the test to verify it fails**

Run: `go test ./internal/combat/... -run TestResolveReach -v`
Expected: FAIL — `undefined: combat.ResolveReach`.

- [x] **Step 5: Write the implementation**

Create `internal/combat/reach.go`:

```go
package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ResolveReach determines a combatant's current formationcombat.Reach from
// their equipped weapon, falling back to innateReach (a mob's own Reach
// flag; always false for a player, who has no reach source but a weapon).
// It is computed at the point of use rather than stored, so it always
// reflects the currently equipped weapon with no second source of truth.
func ResolveReach(c *characters.Character, innateReach bool) formationcombat.Reach {
	if c != nil && c.Equipment.Weapon.ItemId > 0 {
		spec := c.Equipment.Weapon.GetSpec() // ItemSpec by value; zero value if unresolvable
		if spec.Subtype == items.Shooting {
			return formationcombat.ReachAny
		}
		if spec.Reach {
			return formationcombat.ReachExtended
		}
	}

	if innateReach {
		return formationcombat.ReachExtended
	}

	return formationcombat.ReachNone
}
```

- [x] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/combat/... -run TestResolveReach -v`
Expected: PASS, all five tests.

- [x] **Step 7: `go vet`/`gofmt`, full build, full race suite**

```bash
gofmt -l internal/items internal/mobs internal/combat internal/formationcombat
go vet ./internal/items/... ./internal/mobs/... ./internal/combat/... ./internal/formationcombat/...
go build ./...
go test -race ./...
```
Expected: no output from `gofmt`/`vet`, clean build, full suite green — the
two new struct fields are `omitempty`/zero-value-default, so no existing
YAML data or test fixture is affected.

- [x] **Step 8: Commit**

```bash
git add internal/items/itemspec.go internal/mobs/mobs.go internal/combat/reach.go internal/combat/reach_test.go
git commit -m "feat(combat): add Reach trait schema and ResolveReach adapter"
```

## Task 3: Read-only adjacency query command

**Files:**
- Modify: `modules/company/formation.go`

**Interfaces:**
- Consumes: `formationcombat.Legal`/`LegalTargets` (Task 1),
  `domain.Formation`/`domain.MemberKey` (existing), the existing
  `resolveMemberKey`/`memberName`/registry-lookup helpers already in this
  file.
- Produces: a `formation reach <member>` subcommand, purely additive to the
  existing `formationCommand` dispatcher.

- [x] **Step 1: Add the subcommand**

In `modules/company/formation.go`, inside `formationCommand`'s `switch
args[0]` (alongside the existing `move`/`swap`/`clear` cases), add:

```go
	case "reach":
		if len(args) < 2 {
			user.SendText("Usage: formation reach <member>")
			return true, nil
		}

		record, existed := m.registry.Get(user.UserId)
		if !existed {
			user.SendText("You don't have a company.")
			return true, nil
		}

		key, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			user.SendText(err.Error())
			return true, nil
		}

		_, col, placed := record.Formation.Find(key)
		if !placed {
			user.SendText(fmt.Sprintf("%s isn't placed in the formation.", m.memberName(user.UserId, key)))
			return true, nil
		}

		alive := make(map[domain.MemberKey]bool)
		alive[domain.LeaderMemberKey] = true
		for _, comp := range record.Companions {
			alive[domain.CompanionMemberKey(comp.ID)] = true
		}

		targets := formationcombat.LegalTargets(col, record.Formation, alive, formationcombat.ReachNone)
		if len(targets) == 0 {
			user.SendText(fmt.Sprintf("%s has no legal plain-melee targets within the company right now.", m.memberName(user.UserId, key)))
			return true, nil
		}

		names := make([]string, 0, len(targets))
		for _, t := range targets {
			if t == key {
				continue
			}
			names = append(names, m.memberName(user.UserId, t))
		}
		user.SendText(fmt.Sprintf("%s could plain-melee-reach: %s", m.memberName(user.UserId, key), strings.Join(names, ", ")))
		return true, nil
```

Note this queries legality **within the company's own formation** (a
harmless, always-available read-only demo of the predicate — there is no
live enemy party to query against outside combat, and this phase does not
wire combat at all per the Task 3/scope-decision note above). Add
`"github.com/GoMudEngine/GoMud/internal/formationcombat"` and `"strings"`
to this file's imports if not already present (`fmt` is already imported,
confirm before adding it again).

- [x] **Step 2: Update the usage string**

Update `formationUsage` (near the top of the file) to include the new
subcommand:

```go
const formationUsage = "Usage: formation | formation move <member> <row> <col> | formation swap <member-a> <member-b> | formation clear <member> | formation reach <member>"
```

- [x] **Step 3: Build and run existing module tests**

Run: `go build ./... && go test ./modules/company/... -v`
Expected: clean build, all existing `modules/company` tests still pass (no
test exists yet for the new subcommand — this module's existing tests are
integration-shaped and heavier to extend; the domain-level legality logic
is already fully covered by Task 1's tests, so this step is a smoke check,
not new coverage. If `modules/company` has no test file that exercises
`formationCommand` end-to-end today, skip writing one here — matching the
"don't invent test infrastructure a phase doesn't need" principle; Task 1
already gives this command's actual logic full coverage).

- [x] **Step 4: Commit**

```bash
git add modules/company/formation.go
git commit -m "feat(company): add read-only formation reach query command"
```

## Task 4: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [x] **Step 1: Run full verification**

```bash
make generate
make validate
go test -race ./...
```
Expected: all three succeed (record the actual test/package counts in the
status update — don't guess).

- [x] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a "Phase 11c — Formation Tactics (domain layer, schema, and read-only
query complete; combat-loop wiring deferred, <today's date>)" work-log
entry, following the established format. State plainly in "Deferred": the
`NewRound_DoCombat.go` wiring for both interception and legality-gated
targeting (blocked on a new company-formation-by-mob-instance-ID query seam
that doesn't exist yet — see this plan's scope-decision section for the
full reasoning), the equivalent wiring of 11b's `AssignTarget` using
`formationcombat.Legal` as its `legal` parameter (both land together, once
the seam exists), 11d, guard-stance/chance-based interception, row/column
AoE, formation buffs, flanking/exposure, and movement-in-combat. Update:
- The phase progress table row `11c | Formation tactics | Designed, not implemented`
  to `Domain layer, schema, and read-only query complete; combat-loop
  wiring deferred`.
- The `## Current position` "Completed"/"Next" summary — Phase 11
  (11a-11c) foundational work is now complete at the domain-layer level;
  the "Next" item becomes the combat-loop integration follow-up (wiring
  11b + 11c together), not a numbered phase of its own unless the owner
  wants one opened.
- The file header's `**HEAD:**` line.

- [x] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record Phase 11c domain-layer completion"
```

---

## Self-Review Notes

- **Spec coverage:** worked example (A/B/C, before/after A dies) ✓
  (`TestLegalWorkedExamplePlainMelee`, `TestLegalSelfHealsWhenBlockerDies`).
  Reach-extended hits middle not back ✓
  (`TestLegalReachExtendedHitsMiddleNotBack`). Ranged ignores depth ✓
  (`TestLegalRangedIgnoresColumnDepth`). Interception redirect + "no living
  front row, original stands" ✓ (three `TestInterceptFrontRow*` tests).
  Self-healing without clearing target — this plan doesn't touch `Aggro` at
  all (no hooks wiring), so there's nothing to clear; the underlying purity
  guarantee (`Legal` recomputes fresh from current `alive`, never mutates
  anything) is what the future wiring will rely on, and is itself tested by
  `TestLegalSelfHealsWhenBlockerDies`. Lateral-range edge columns ✓
  (`TestInLateralRangeEdgeColumns`, table-driven all 9 combinations).
  Adjacency query read-only correctness ✓ (`TestLegalTargetsListsExactlyTheLegalMembers`,
  plus the read-only `formation reach` command in Task 3). `Reach` trait
  derivable from either weapon or innate mob flag, uniformly ✓ (Task 2,
  `ResolveReach`). The acceptance criterion "an attack aimed at a
  non-front-row defender redirects to a living front-row member... with no
  living front-row member... the original target stands" is fully covered
  at the domain layer (`InterceptFrontRow`); actually redirecting a live
  attack is part of the deferred hooks wiring, called out explicitly in the
  scope decision, not silently missing.
- **Placeholder scan:** no TBD/TODO. Task 2 Step 3 has one explicit
  "verify the exact helper name" note because the real `internal/items`
  test-spec-registration helper name wasn't confirmed during planning
  research — this is a deliberate, named uncertainty for the implementer to
  resolve by checking existing `internal/items`/`internal/combat` test
  files before writing that one test, not a vague "add tests" placeholder;
  every other step has concrete, runnable code.
- **Type consistency:** `Reach`/`ReachNone`/`ReachExtended`/`ReachAny`,
  `Legal`, `InterceptFrontRow`, `LegalTargets` match between Task 1's
  definitions and Task 2/3's call sites.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11c-formation-tactics.md`.**
