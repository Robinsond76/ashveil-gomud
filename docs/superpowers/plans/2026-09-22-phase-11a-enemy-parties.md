# Phase 11a Enemy Parties Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Status: fully implemented and committed on `phase-11a-enemy-parties` — every step below is checked off.**

**Goal:** Give mobs a first-class "party" concept — mobs sharing a `Groups`
tag in a room assemble into a stable-for-this-listing group with an
auto-assigned `company.Formation` 3×3 grid — so a room can contain multiple
distinct hostile groups, each shown to players as one aggregate listing
instead of N individual mob lines.

**Architecture:** A new GoMud-free domain package, `internal/mobparty`,
exposes a pure `Assemble(mobs []MobSummary) []Party` function: it groups
input mobs by their first `Groups` tag (untagged mobs are solo parties of
one), splits any group over 5 members into multiple parties in input order,
and auto-fills each party's `company.Formation` front-to-back by descending
`EHP`. `internal/rooms/roomdetails.go`'s existing per-mob room-listing loop
is the only integration point: it now buckets its already-resolved hostile
mob display strings by party via `mobparty.Assemble` and renders a
multi-member party as one aggregate line (e.g. "a pack of 3 goblins")
instead of one line per mob. Friendly/charmed mobs are untouched — grouping
only applies to the hostile bucket, per the design's "hostile party" framing.

**Note on `EHP`/`DPS`:** `internal/combat.MobRank` (the source of `EHP`/`DPS`)
lives in a package that itself imports `internal/rooms`, so `internal/rooms`
cannot import `internal/combat` back (import cycle). The room-display
integration in this plan therefore calls `mobparty.Assemble` with
`EHP`/`DPS` left at their zero value — display grouping only needs *which*
mobs share a party, never the internal front/mid/back ordering, so this
doesn't weaken the display feature. Wiring real `EHP`/`DPS` into the
`Formation` that 11b/11c will actually read for combat purposes is 11b's
concern, from whichever integration point 11b turns out to need (11b is
unplanned; do not resolve this here).

**Tech Stack:** Go, `testify` (`assert`/`require`), no new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-22-phase-11a-enemy-parties-design.md`
(and the shared prior-art/decisions in
`docs/superpowers/specs/2026-09-22-phase-11-formation-combat-design.md`).

## Global Constraints

- Never advances `gametime`, the round counter, or moves any mob.
- Never mutates combat, health, or hostility state — `Groups`/`MakeHostile`
  are read-only inputs to this phase.
- A party is capped at 5 members (mirrors `company.MaxCompanions + 1`); a
  `Groups` tag shared by more than 5 room mobs splits into multiple parties
  in spawn/input order rather than overflowing one `Formation`.
- No persisted state: `Party`/`Formation` here are assembled fresh on every
  call, matching `characters.Aggro`'s own non-persistence.
- No new commands, no new config file, no new durable registry.
- `internal/mobparty` stays GoMud-free: it takes plain `MobSummary` structs,
  never `*mobs.Mob`/`*rooms.Room` directly.

---

## File Structure

- Create: `internal/mobparty/party.go` — `MobSummary`, `Party`, `Assemble`,
  and the grouping/formation-fill heuristic. Pure domain logic, no GoMud
  imports.
- Create: `internal/mobparty/party_test.go` — domain tests for grouping,
  EHP ordering, solo front-center placement, and the 5-member cap/split.
- Modify: `internal/rooms/roomdetails.go:272-305` — replace the flat
  per-mob-instance loop with one that still resolves display names exactly
  as today, but buckets hostile mobs into a local slice instead of appending
  straight to `details.VisibleMobs`, then hands that bucket to a new
  `groupedMobDisplay` helper.
- Create: `internal/rooms/mobparty_display.go` — `hostileMobDisplay` (the
  small local struct carrying instance ID, `Groups`, raw name, and rendered
  display string), `groupedMobDisplay`, `describeParty`, `pluralize`. This
  is the engine-specific glue: it imports `internal/mobparty` and renders
  party membership into player-facing text.
- Create: `internal/rooms/mobparty_display_test.go` — tests for
  `groupedMobDisplay`/`describeParty`/`pluralize` using hand-built
  `hostileMobDisplay` values (no `mobs.GetInstance`/user/engine setup
  needed — these helpers are pure functions of already-resolved strings).

## Task 1: `internal/mobparty` domain package

**Files:**
- Create: `internal/mobparty/party.go`
- Test: `internal/mobparty/party_test.go`

**Interfaces:**
- Consumes: `company.Formation`, `company.MemberKey`, `company.FormationCols`,
  `company.MaxCompanions` (all from the already-existing
  `github.com/GoMudEngine/GoMud/internal/company` package).
- Produces (for Task 2 and for any future 11b work):
  - `type MobSummary struct { InstanceId int; Groups []string; EHP float64; DPS float64 }`
  - `type Party struct { ID string; Members []int; Formation company.Formation }`
  - `func Assemble(mobs []MobSummary) []Party`
  - `const MaxPartySize = company.MaxCompanions + 1` (= 5)

- [x] **Step 1: Write the failing tests**

Create `internal/mobparty/party_test.go`:

```go
package mobparty_test

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleSoloMobsEachBecomeOwnParty(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1},
		{InstanceId: 2},
	})

	require.Len(t, parties, 2)
	for _, p := range parties {
		require.Len(t, p.Members, 1)
		row, col, ok := p.Formation.Find(company.MemberKey(memberKeyFor(p.Members[0])))
		require.True(t, ok)
		assert.Equal(t, 0, row, "solo party occupies the front row")
		assert.Equal(t, 1, col, "solo party occupies the center column")
	}
	assert.NotEqual(t, parties[0].ID, parties[1].ID)
}

func TestAssembleGroupsSharedTagIntoOneParty(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, Groups: []string{"goblin-raiders"}},
		{InstanceId: 2, Groups: []string{"goblin-raiders"}},
		{InstanceId: 3, Groups: []string{"goblin-raiders"}},
		{InstanceId: 4}, // untagged, solo
	})

	require.Len(t, parties, 2)

	var grouped, solo mobparty.Party
	for _, p := range parties {
		if len(p.Members) == 3 {
			grouped = p
		} else {
			solo = p
		}
	}

	assert.ElementsMatch(t, []int{1, 2, 3}, grouped.Members)
	assert.Equal(t, []int{4}, solo.Members)
}

func TestAssembleOrdersFormationByEHPDescending(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, Groups: []string{"pack"}, EHP: 10},
		{InstanceId: 2, Groups: []string{"pack"}, EHP: 50},
		{InstanceId: 3, Groups: []string{"pack"}, EHP: 30},
	})

	require.Len(t, parties, 1)
	f := parties[0].Formation

	// Highest EHP (instance 2) fills the front row first, left to right,
	// in descending EHP order: 2 (50), 3 (30), 1 (10).
	assert.Equal(t, company.MemberKey(memberKeyFor(2)), f.At(0, 0))
	assert.Equal(t, company.MemberKey(memberKeyFor(3)), f.At(0, 1))
	assert.Equal(t, company.MemberKey(memberKeyFor(1)), f.At(0, 2))
}

func TestAssembleCapsAtFiveAndSplits(t *testing.T) {
	members := make([]mobparty.MobSummary, 0, 6)
	for i := 1; i <= 6; i++ {
		members = append(members, mobparty.MobSummary{InstanceId: i, Groups: []string{"horde"}})
	}

	parties := mobparty.Assemble(members)

	require.Len(t, parties, 2)
	assert.Len(t, parties[0].Members, mobparty.MaxPartySize)
	assert.Len(t, parties[1].Members, 1)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5}, parties[0].Members)
	assert.Equal(t, []int{6}, parties[1].Members)
	assert.NotEqual(t, parties[0].ID, parties[1].ID)
}

func memberKeyFor(instanceId int) string {
	return fmt.Sprintf("mob:%d", instanceId)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mobparty/... -run TestAssemble -v`
Expected: FAIL — `package internal/mobparty is not a package` / no such file
(the package doesn't exist yet).

- [x] **Step 3: Write the implementation**

Create `internal/mobparty/party.go`:

```go
// Package mobparty groups hostile mobs present in a room into stable
// "parties" — mirroring the player's company — and auto-assigns each
// party's 3x3 company.Formation by a simple EHP/DPS role heuristic.
//
// A Party is never persisted: it is cheap to recompute and is assembled
// fresh every time a caller needs it, the same way characters.Aggro is
// never persisted today (see the Phase 11 design doc's prior-art check).
package mobparty

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/company"
)

// MaxPartySize mirrors the company cap (leader + MaxCompanions): a party's
// Formation has only 9 cells and the company side is capped at 5, so the
// enemy side uses the same cap rather than inventing a different one.
const MaxPartySize = company.MaxCompanions + 1

// MobSummary is the minimal, GoMud-free view of a mob this package needs.
// Callers (module/engine-layer code) adapt a live *mobs.Mob into this.
type MobSummary struct {
	InstanceId int
	Groups     []string
	EHP        float64
	DPS        float64
}

// Party is a stable-for-this-listing group of mobs with an auto-assigned
// formation. It is not persisted.
type Party struct {
	ID        string
	Members   []int
	Formation company.Formation
}

// Assemble groups mobs sharing a Groups tag (their first tag; a mob with no
// Groups tag is always its own solo party) into parties of up to
// MaxPartySize, then assigns each party's Formation front-to-back by
// descending EHP. Parties are returned in first-occurrence order of their
// grouping key, matching the input order of mobs.
func Assemble(mobs []MobSummary) []Party {
	type bucket struct {
		key     string
		members []MobSummary
	}

	var buckets []*bucket
	byKey := make(map[string]*bucket)

	for _, m := range mobs {
		key := groupKey(m)
		if key == "" {
			// Untagged: always a fresh solo party, never merged with
			// another untagged mob.
			buckets = append(buckets, &bucket{
				key:     fmt.Sprintf("solo:%d", m.InstanceId),
				members: []MobSummary{m},
			})
			continue
		}

		b, ok := byKey[key]
		if !ok {
			b = &bucket{key: key}
			byKey[key] = b
			buckets = append(buckets, b)
		}
		b.members = append(b.members, m)
	}

	parties := make([]Party, 0, len(buckets))
	for _, b := range buckets {
		for i, chunkMembers := range chunk(b.members, MaxPartySize) {
			id := b.key
			if len(b.members) > MaxPartySize {
				id = fmt.Sprintf("%s#%d", b.key, i+1)
			}
			parties = append(parties, buildParty(id, chunkMembers))
		}
	}

	return parties
}

// groupKey returns the grouping key for a mob: its first Groups tag,
// prefixed to avoid colliding with the "solo:<id>" key space, or "" if the
// mob has no Groups tag (always solo).
func groupKey(m MobSummary) string {
	if len(m.Groups) == 0 || m.Groups[0] == "" {
		return ""
	}
	return "group:" + m.Groups[0]
}

// chunk splits members into groups of at most size, preserving order —
// this is the "split by spawn order" rule for parties over the cap.
func chunk(members []MobSummary, size int) [][]MobSummary {
	var chunks [][]MobSummary
	for len(members) > 0 {
		n := size
		if n > len(members) {
			n = len(members)
		}
		chunks = append(chunks, members[:n])
		members = members[n:]
	}
	return chunks
}

// buildParty ranks members by descending EHP and fills the Formation
// front-row-first (up to 3), then mid, then back. A solo party occupies
// the front-row center cell rather than front-row-left, matching the
// design's explicit "even a single mob is a unit" placement.
func buildParty(id string, members []MobSummary) Party {
	ranked := make([]MobSummary, len(members))
	copy(ranked, members)
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].EHP > ranked[j].EHP
	})

	var f company.Formation
	memberIds := make([]int, len(ranked))
	for i, m := range ranked {
		memberIds[i] = m.InstanceId
		row, col := slotFor(i, len(ranked))
		// Every (row, col) here is in-bounds by construction (MaxPartySize
		// caps len(ranked) at 5, and slotFor never exceeds the 3x3 grid for
		// i < 5), and each memberKey is unique and unplaced, so Place
		// cannot fail; the error is intentionally discarded.
		_ = f.Place(memberKey(m.InstanceId), row, col)
	}

	return Party{ID: id, Members: memberIds, Formation: f}
}

// slotFor returns the formation cell for the i-th ranked (0-indexed, by
// descending EHP) member of a party of the given size.
func slotFor(i, size int) (row, col int) {
	if size == 1 {
		return 0, 1 // front-row center for a solo party
	}
	return i / company.FormationCols, i % company.FormationCols
}

func memberKey(instanceId int) company.MemberKey {
	return company.MemberKey(fmt.Sprintf("mob:%d", instanceId))
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mobparty/... -v`
Expected: PASS, all four tests.

- [x] **Step 5: Run `go vet` and `gofmt` on the new package**

Run: `gofmt -l internal/mobparty && go vet ./internal/mobparty/...`
Expected: `gofmt -l` prints nothing (no formatting issues); `go vet` prints
nothing.

- [x] **Step 6: Commit**

```bash
git add internal/mobparty/party.go internal/mobparty/party_test.go
git commit -m "feat(mobparty): add Phase 11a enemy party assembly"
```

## Task 2: Room display integration

**Files:**
- Modify: `internal/rooms/roomdetails.go:272-305` (the hostile-mob branch of
  the existing loop)
- Create: `internal/rooms/mobparty_display.go`
- Test: `internal/rooms/mobparty_display_test.go`

**Interfaces:**
- Consumes: `mobparty.MobSummary`, `mobparty.Party`, `mobparty.Assemble`
  (from Task 1).
- Produces: `hostileMobDisplay` (package-local struct, `internal/rooms`),
  `groupedMobDisplay(entries []hostileMobDisplay) []string` — this is what
  `roomdetails.go`'s loop calls to build the grouped `details.VisibleMobs`
  slice for hostile mobs.

- [x] **Step 1: Write the failing tests**

Create `internal/rooms/mobparty_display_test.go`:

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupedMobDisplaySoloMobRendersItsOwnLine(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, rawName: "goblin", display: "a rusty goblin"},
	})

	assert.Equal(t, []string{"a rusty goblin"}, lines)
}

func TestGroupedMobDisplayGroupsSharedTagIntoOneAggregateLine(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a rusty goblin"},
		{instanceId: 2, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a scarred goblin"},
		{instanceId: 3, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a young goblin"},
	})

	assert.Equal(t, []string{"a pack of 3 goblins"}, lines)
}

func TestGroupedMobDisplayMixedNamesUseGenericLabel(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, groups: []string{"mixed-pack"}, rawName: "goblin", display: "a rusty goblin"},
		{instanceId: 2, groups: []string{"mixed-pack"}, rawName: "wolf", display: "a gray wolf"},
	})

	assert.Equal(t, []string{"a pack of 2 creatures"}, lines)
}

func TestGroupedMobDisplayKeepsUngroupedMobsSeparate(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, rawName: "bear", display: "a hungry bear"},
		{instanceId: 2, rawName: "bear", display: "a sleepy bear"},
	})

	// No shared Groups tag: two solo parties, not one aggregate line.
	assert.ElementsMatch(t, []string{"a hungry bear", "a sleepy bear"}, lines)
}

func TestPluralize(t *testing.T) {
	assert.Equal(t, "goblins", pluralize("goblin"))
	assert.Equal(t, "foxes", pluralize("fox"))
	assert.Equal(t, "", pluralize(""))
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rooms/... -run 'TestGroupedMobDisplay|TestPluralize' -v`
Expected: FAIL — `undefined: hostileMobDisplay` / `undefined: groupedMobDisplay` / `undefined: pluralize`.

- [x] **Step 3: Write the implementation**

Create `internal/rooms/mobparty_display.go`:

```go
package rooms

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
)

// hostileMobDisplay is the already-resolved view of one hostile room mob
// that roomdetails.go's mob loop hands to groupedMobDisplay: everything
// needed to render either its own line or fold it into a party's aggregate
// line, without re-touching mobs.GetInstance or the engine.
type hostileMobDisplay struct {
	instanceId int
	groups     []string
	rawName    string // undecorated Character.Name, used for the mixed-party check
	display    string // fully rendered mob name (colors, adjectives, quest alert, etc.)
}

// groupedMobDisplay renders hostile room mobs grouped by mobparty.Assemble:
// a party of one keeps its own individually rendered line; a multi-member
// party collapses to one aggregate line (e.g. "a pack of 3 goblins").
//
// EHP/DPS are left at zero here deliberately: internal/rooms cannot import
// internal/combat (internal/combat already imports internal/rooms, so the
// reverse would cycle), and display grouping only needs party membership,
// never the front/mid/back Formation ordering — that ordering matters once
// something actually reads Formation for combat purposes, which is out of
// scope for Phase 11a's room-display integration.
func groupedMobDisplay(entries []hostileMobDisplay) []string {
	summaries := make([]mobparty.MobSummary, len(entries))
	displayByInstance := make(map[int]string, len(entries))
	rawNameByInstance := make(map[int]string, len(entries))

	for i, e := range entries {
		summaries[i] = mobparty.MobSummary{InstanceId: e.instanceId, Groups: e.groups}
		displayByInstance[e.instanceId] = e.display
		rawNameByInstance[e.instanceId] = e.rawName
	}

	lines := make([]string, 0, len(entries))
	for _, p := range mobparty.Assemble(summaries) {
		if len(p.Members) == 1 {
			lines = append(lines, displayByInstance[p.Members[0]])
			continue
		}
		lines = append(lines, describeParty(p, rawNameByInstance))
	}

	return lines
}

// describeParty builds one aggregate line for a multi-member party. Members
// that don't all share the same base name collapse to a generic label.
//
// ponytail: naive "+s"/"+es" pluralization and a flat "a pack of N X"
// phrasing — good enough for the first grouped listing; swap in real
// pluralization/party-noun-by-race if players start noticing "wolfs".
func describeParty(p mobparty.Party, rawNameByInstance map[int]string) string {
	name := ""
	mixed := false
	for _, id := range p.Members {
		n := rawNameByInstance[id]
		if name == "" {
			name = n
		} else if n != name {
			mixed = true
		}
	}

	if mixed || name == "" {
		return fmt.Sprintf("a pack of %d creatures", len(p.Members))
	}
	return fmt.Sprintf("a pack of %d %s", len(p.Members), pluralize(name))
}

func pluralize(name string) string {
	if name == "" {
		return name
	}
	switch name[len(name)-1] {
	case 's', 'x', 'z':
		return name + "es"
	default:
		return name + "s"
	}
}
```

Now wire it into the existing loop. In `internal/rooms/roomdetails.go`,
replace lines 272-305 (the block starting at `visibleFriendlyMobs := []string{}`
and ending at `details.VisibleMobs = append(details.VisibleMobs, visibleFriendlyMobs...)`)
with:

```go
	visibleFriendlyMobs := []string{}
	hostileMobs := []hostileMobDisplay{}

	for idx, mobInstanceId := range r.mobs {
		if mob := mobs.GetInstance(mobInstanceId); mob != nil {

			if mob.Character.HasBuffFlag("hidden") { // Don't show them if they are sneaking
				if !user.Character.Pet.Exists() || !user.Character.HasBuffFlag("see-hidden") {
					continue
				}
			}

			tmpNameFlags := nameFlags

			mobName := mob.Character.GetMobName(user.UserId, tmpNameFlags...)

			for _, qFlag := range mob.QuestFlags {
				if user.Character.HasQuest(qFlag) || (len(qFlag) >= 5 && qFlag[len(qFlag)-5:] == `start`) {
					mobName.QuestAlert = true
					break
				}
			}

			if mob.Character.IsCharmed() {
				visibleFriendlyMobs = append(visibleFriendlyMobs, mobName.String())
			} else {
				hostileMobs = append(hostileMobs, hostileMobDisplay{
					instanceId: mobInstanceId,
					groups:     mob.Groups,
					rawName:    mob.Character.Name,
					display:    mobName.String(),
				})
			}
		} else {
			r.mobs = append(r.mobs[:idx], r.mobs[idx+1:]...)
		}
	}

	details.VisibleMobs = groupedMobDisplay(hostileMobs)

	// Add the friendly mobs to the end
	details.VisibleMobs = append(details.VisibleMobs, visibleFriendlyMobs...)
```

This preserves every existing behavior byte-for-byte (hidden-mob filtering,
quest-flag alerts, charmed/friendly separation, and the stale-instance
pruning side effect on `r.mobs`) — the only change is that hostile mobs go
into `hostileMobs` first and get grouped before landing in
`details.VisibleMobs`, instead of being appended directly.

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rooms/... -v`
Expected: PASS, including the new `TestGroupedMobDisplay*`/`TestPluralize`
tests and every pre-existing `internal/rooms` test (no regressions).

- [x] **Step 5: Run `go vet` and `gofmt`, then the full test suite**

Run: `gofmt -l internal/rooms internal/mobparty && go vet ./internal/rooms/... ./internal/mobparty/...`
Expected: no output from either command.

Run: `go build ./...`
Expected: builds cleanly (confirms no import-cycle regression and no other
package broke from the `RoomTemplateDetails`-adjacent change).

Run: `go test -race ./...`
Expected: PASS, full suite, no regressions anywhere in the repo.

- [x] **Step 6: Commit**

```bash
git add internal/rooms/roomdetails.go internal/rooms/mobparty_display.go internal/rooms/mobparty_display_test.go
git commit -m "feat(rooms): group hostile mob room listings into parties"
```

## Task 3: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [x] **Step 1: Run full verification**

```bash
make generate
make validate
go test -race ./...
```

Expected: all three succeed (record the actual test/package counts from the
`go test -race ./...` output in the status update below — don't guess a
number).

- [x] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a "Phase 11a — Enemy Parties (complete, <today's date>)" work-log entry
following the existing format used by Phase 8/9/10's entries (What/Why/Step
completed/Key commits/Verification/Live acceptance/Deferred), and update:
- The phase progress table row `11a | Enemy parties | Designed, not implemented`
  to `Complete`.
- The `## Current position` "Completed"/"Next" summary.
- The file header's `**HEAD:**` line.

State plainly in "Deferred": 11b (unit-vs-unit engagement — nothing yet
reads a party's `Formation` for combat), 11c (formation tactics), real
`EHP`/`DPS` wiring into the room-display `Formation` (left at zero per the
Task 2 note — display doesn't need it, 11b will need it and will resolve
where it gets computed from), and GMCP's parallel mob-list building
(`modules/gmcp/gmcp.Room.go`, ~line 366-384) which still lists mobs
individually and was not touched this phase.

- [x] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record Phase 11a enemy parties completion"
```

---

## Self-Review Notes

- **Spec coverage:** "Party" domain type ✓ (Task 1). Auto-formation on
  assembly, never hand-authored/player-controlled ✓ (Task 1, `Assemble` is
  the only constructor). Solo mob = party of one, front-row center ✓ (Task
  1, `slotFor` size==1 case, tested). Room display: multiple hostile
  parties each as one entry, aggregate messaging ✓ (Task 2). Never
  persisted ✓ (no registry/YAML anywhere in this plan). 5-member cap with
  split ✓ (Task 1, `chunk`, tested). Never mutates `Groups`/`MakeHostile`
  or advances gametime/round count ✓ (both tasks are pure read-and-render,
  no writes to mob/character/round state anywhere).
- **Placeholder scan:** no TBD/TODO, every step has real code, no "similar
  to Task N" references.
- **Type consistency:** `MobSummary`, `Party`, `Assemble`, `MaxPartySize`
  from Task 1 are used with matching names/signatures in Task 2's
  `groupedMobDisplay`. `hostileMobDisplay`'s fields (`instanceId`, `groups`,
  `rawName`, `display`) match between its definition and every test/call
  site.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11a-enemy-parties.md`.**
