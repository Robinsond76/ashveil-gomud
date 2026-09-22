# Phase 11 Combat-Loop Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the three already-shipped, fully-tested pure domain
packages (11a `internal/mobparty`, 11b `internal/engagement`, 11c
`internal/formationcombat`) into real combat, for the single highest-value,
most central interaction: a player fighting a hostile enemy party. When a
player attacks (or is targeting) a member of an assembled enemy party,
front-row interception and column-occupancy/lateral-range legality now
actually gate and redirect the attack, instead of `Aggro` blindly resolving
against whatever mob instance it points to.

**This is not a numbered phase (11d) — it's the join between 11a/11b/11c**,
tracked in `docs/PROJECT_STATUS.md`'s "Next" line after Phase 11c. There is
no separate spec doc under `docs/superpowers/specs/`; the three sub-phase
specs already cover the design decisions this wiring consumes verbatim
(column-occupancy reach, lateral range, interception from 11c; targeting
preference from 11b — see the scope note below for why 11b's `AssignTarget`
specifically is *not* wired this pass). This plan folds the small amount of
additional design decision-making (the new seam, the fail-open contract,
the EHP computation) into its own "Design decisions" section below in place
of a separate spec file, given the scope is bounded to gluing existing
designs together rather than inventing new gameplay rules.

## Design decisions

**1. New seam: `company.FormationProvider`.** `internal/hooks` cannot
import `modules/company` (modules depend on `internal/`, not the reverse).
Mirroring the exact pattern already used by `internal/survival.CompanyService`
(Phase 5) and `internal/weather.Provider` (Phase 8) — a `Set<X>`/`<X>For`
package-level pair in the pure `internal/` package, registered from the
implementing module's own `init()`, no central plugin registry involved —
`internal/company` gets a new `FormationProvider` interface and
`SetFormationProvider`/`FormationFor` functions. `modules/company.CompanyModule`
implements it with one new method (`FormationFor`, reading straight from its
existing `registry.Get`) and registers it in `init()` next to its existing
`survival.SetRosterProvider(m)` call.

**2. Fail-open contract.** If a player has no company record at all
(`company.FormationFor` returns `ok=false` — a solo player, or one who's
never summoned a companion), or their own `LeaderMemberKey` somehow isn't
placed in their formation, formation gating is skipped entirely and combat
resolves exactly as it does today. This is deliberate, not a shortcut: the
feature is "company vs. party" tactics, and a player with no company has no
formation concept to apply. Failing open here means this change is
zero-behavior-impact for every player who has never touched the company
system, which is the overwhelming majority of existing playtesting to date.

**3. `EHP` is computed live, not via `combat.RankMobs()`.** 11a's own
work-log flagged `RankMobs()` (which ranks *every* mob spec in the game) as
too expensive to call per room-look; the same is true per combat round, more
so. For a *live* mob instance already in the fight, `EHP = HealthMax /
(1 - min(Defense/200, 0.95))` needs only that instance's own
`Character.HealthMax.Value`/`GetDefense()` — no simulation, no ranking pass.
This is the same formula `combat.RankMobs()` uses internally
(`internal/combat/mob_rank.go:164-171`), just applied directly to a live
character instead of a simulated one; it's reimplemented locally in
`internal/hooks` (three lines) rather than extracting a shared export from
`internal/combat`, since `internal/combat/AGENTS.md` asks for narrowly
scoped combat-package changes and this doesn't need a new public API there.
Using `HealthMax` (not current `Health`) also means the front/mid/back
ordering `mobparty.Assemble` computes stays stable round to round as mobs
take damage — matching 11c's "formation is locked once combat starts"
invariant — without needing to cache anything, exactly as 11a's own design
intended.

**4. Scope: player-vs-mob only, no reassignment-on-death this pass.**
Research before writing this plan found the existing dead-target handling
in `internal/hooks/NewRound_DoCombat.go` (`:486-526`) already clears
`Aggro` and `continue`s the round *before* reaching the attack call, for
both "mob instance no longer exists" and "mob found but `Health < 1"`. Real
reassignment-on-death (11b's stated acceptance criterion) means intervening
inside that existing, working, unmodified-since-launch validation block —
a materially more invasive change than adding a new gate immediately before
the attack call. This plan deliberately does the safer, purely additive
thing: a new pre-attack gate (interception + legality) that never touches
the existing target-validation/Aggro-clearing logic at all. 11b's
`engagement.AssignTarget` therefore stays unwired this pass — it has
nothing to attach to without also touching the validation block. The three
other attack-resolution call sites (mob-vs-player, mob-vs-mob,
player-vs-player) are also out of scope: player-vs-player is explicit PvP,
out of scope per the Phase 11 design doc; mob-vs-player interception would
need to redirect an attack from `combat.AttackMobVsPlayer` to
`combat.AttackMobVsMob` when a companion intercepts for the leader — a
different `Attack*` function, not just a different target, which is its own
piece of work; mob-vs-mob covers both "my companion attacks the enemy" and
"the enemy attacks my companion," which needs to disambiguate charmed vs.
hostile on both sides of every call. All three are real, valuable follow-ups,
each cleanly separable from this one and from each other — tracked in
`docs/PROJECT_STATUS.md`, not silently dropped.

**5. New pure/adapter split, matching the 11a-c pattern.** The actual
decision logic (given a `Formation`, an `alive` map, an attacker's column,
the original target key, and a `Reach`, decide the final legal target or
that the attack should be skipped) is one small pure function,
`resolveAttackTarget`, fully unit-testable with hand-built data — no
`mobs.GetInstance`/`rooms.LoadRoom`/global state involved. A thin adapter,
`gateFormationAttack`, does the actual engine lookups (room's hostile mobs,
live HP, the player's own formation column) and calls it. This keeps the
one part of the change that's genuinely hard to get wrong (the legality
math) covered by fast, deterministic tests, even though
`internal/hooks` itself has no existing test infrastructure to build
integration tests against.

## Global Constraints

- Never advances `gametime`, round count, or damage math — this change is
  entirely about *which* mob a player's existing attack lands on, never how
  hard it hits or whether it advances time.
- Formation is read-only here — nothing in this plan ever mutates a
  `Formation`, `Aggro`, or a room's mob list; it only *reads* current state
  to redirect or skip one round's attack.
- Fails open (see Design Decision 2): no formation record, or leader not
  placed in their own formation, means unchanged pre-existing behavior.
- A blocked round never clears `Aggro` — matches 11c's self-healing design;
  the same target becomes legal automatically once whatever blocks it dies,
  with no player action and no lost engagement.

---

## File Structure

- Create: `internal/company/provider.go` — `FormationProvider`,
  `SetFormationProvider`, `FormationFor`.
- Modify: `modules/company/company.go` — add `FormationFor` method, register
  in `init()`.
- Modify: `internal/mobparty/party.go` — export `memberKey` as
  `MemberKeyFor`; add `InstanceIdFromMemberKey`.
- Modify: `internal/mobparty/party_test.go` — update the private test
  helper's collision-free naming, add a round-trip test for the two newly
  exported functions.
- Create: `internal/hooks/combat_formation.go` — `resolveAttackTarget`
  (pure), `gateFormationAttack` (adapter) and its small helpers.
- Create: `internal/hooks/combat_formation_test.go` — tests for
  `resolveAttackTarget` and the `effectiveHP` helper.
- Modify: `internal/hooks/NewRound_DoCombat.go` — one gate call inserted at
  the player-vs-mob attack site (`:551-560`).

## Task 1: `company.FormationProvider` seam

**Files:**
- Create: `internal/company/provider.go`
- Modify: `modules/company/company.go`

**Interfaces:**
- Consumes: `company.Formation` (existing), `domain.Registry.Get` (existing,
  `modules/company/company.go`).
- Produces: `company.FormationProvider`, `company.SetFormationProvider(p FormationProvider)`,
  `company.FormationFor(leaderUserID int) (Formation, bool)`.

- [ ] **Step 1: Write the failing test**

Create `internal/company/provider_test.go`:

```go
package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFormationProvider struct {
	formation company.Formation
	ok        bool
}

func (f fakeFormationProvider) FormationFor(leaderUserID int) (company.Formation, bool) {
	return f.formation, f.ok
}

func TestFormationForReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, ok := company.FormationFor(1)
	assert.False(t, ok)
}

func TestFormationForCallsThroughToRegisteredProvider(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 1))

	company.SetFormationProvider(fakeFormationProvider{formation: f, ok: true})
	defer company.SetFormationProvider(nil)

	got, ok := company.FormationFor(42)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, got.At(0, 1))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/company/... -run TestFormationFor -v`
Expected: FAIL — `undefined: company.SetFormationProvider`.

- [ ] **Step 3: Write the implementation**

Create `internal/company/provider.go`:

```go
package company

import "sync"

// FormationProvider is implemented by modules/company. It is a read-only
// query seam — the same shape as survival.CompanyService and
// weather.Provider — so internal/ packages (in particular
// internal/hooks's combat loop) can read a leader's current Formation
// without importing modules/company.
type FormationProvider interface {
	// FormationFor returns leaderUserID's current company Formation. ok is
	// false when the leader has no company record at all (a solo player,
	// or one who has never summoned a companion) — callers should treat
	// that as "no formation concept applies," not as an error.
	FormationFor(leaderUserID int) (Formation, bool)
}

var (
	formationProviderMu sync.RWMutex
	formationProvider   FormationProvider
)

// SetFormationProvider registers the active formation provider. Passing
// nil clears it.
func SetFormationProvider(p FormationProvider) {
	formationProviderMu.Lock()
	defer formationProviderMu.Unlock()
	formationProvider = p
}

// FormationFor calls through to the registered FormationProvider. It
// returns ok=false if no provider is registered (e.g. a test binary that
// never loaded modules/company) or the leader has no company record.
func FormationFor(leaderUserID int) (Formation, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return Formation{}, false
	}
	return p.FormationFor(leaderUserID)
}
```

In `modules/company/company.go`, add this method (near the existing
`Roster` method that implements `survival.RosterProvider`):

```go
// FormationFor implements company.FormationProvider so internal/hooks can
// read a leader's current formation without importing modules/company.
func (m *CompanyModule) FormationFor(leaderUserID int) (domain.Formation, bool) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return domain.Formation{}, false
	}
	return record.Formation, true
}
```

And in `init()`, add one line next to the existing `survival.SetRosterProvider(m)`:

```go
	survival.SetRosterProvider(m)
	domain.SetFormationProvider(m)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company/... ./modules/company/... -v`
Expected: PASS, the two new tests plus every pre-existing test in both
packages (no regressions).

- [ ] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/company modules/company
go vet ./internal/company/... ./modules/company/...
go build ./...
```
Expected: no output, clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/company/provider.go internal/company/provider_test.go modules/company/company.go
git commit -m "feat(company): add FormationProvider query seam"
```

## Task 2: Export `mobparty`'s member-key helpers

**Files:**
- Modify: `internal/mobparty/party.go`
- Modify: `internal/mobparty/party_test.go`

**Interfaces:**
- Produces: `func MemberKeyFor(instanceId int) company.MemberKey` (renamed
  from the existing unexported `memberKey`), `func InstanceIdFromMemberKey(key company.MemberKey) (int, bool)`.

- [ ] **Step 1: Write the failing test**

Add to `internal/mobparty/party_test.go` (new test, existing tests
untouched — the existing `memberKeyFor` test helper stays as-is, it just
happens to produce strings in the same format the package now also exports
a real function for):

```go
func TestMemberKeyForAndInstanceIdFromMemberKeyRoundTrip(t *testing.T) {
	key := mobparty.MemberKeyFor(42)
	id, ok := mobparty.InstanceIdFromMemberKey(key)
	require.True(t, ok)
	assert.Equal(t, 42, id)
}

func TestInstanceIdFromMemberKeyRejectsNonMobKeys(t *testing.T) {
	_, ok := mobparty.InstanceIdFromMemberKey(company.LeaderMemberKey)
	assert.False(t, ok)

	_, ok = mobparty.InstanceIdFromMemberKey(company.CompanionMemberKey(3))
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mobparty/... -run 'TestMemberKeyFor|TestInstanceIdFromMemberKey' -v`
Expected: FAIL — `undefined: mobparty.MemberKeyFor`.

- [ ] **Step 3: Write the implementation**

In `internal/mobparty/party.go`, rename the existing unexported `memberKey`
function to `MemberKeyFor` (update its one call site inside `buildParty`
accordingly), and add:

```go
// MemberKeyFor returns the formation MemberKey a mob instance is placed
// under within an assembled Party's Formation.
func MemberKeyFor(instanceId int) company.MemberKey {
	return company.MemberKey(fmt.Sprintf("mob:%d", instanceId))
}

// InstanceIdFromMemberKey parses a mob's formation MemberKey (as produced
// by MemberKeyFor) back into its instance ID. ok is false for any key not
// in that format, such as a company.LeaderMemberKey or CompanionMemberKey
// (a caller resolving formation membership must not assume every key is a
// mob key just because it's non-empty).
func InstanceIdFromMemberKey(key company.MemberKey) (int, bool) {
	const prefix = "mob:"
	s := string(key)
	if !strings.HasPrefix(s, prefix) {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(s, prefix))
	if err != nil {
		return 0, false
	}
	return id, true
}
```

Add `"strconv"` and `"strings"` to the file's imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mobparty/... -v`
Expected: PASS, all existing tests plus the two new ones.

- [ ] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/mobparty && go vet ./internal/mobparty/... && go build ./...
```
Expected: no output, clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/mobparty/party.go internal/mobparty/party_test.go
git commit -m "feat(mobparty): export member-key helpers for combat wiring"
```

## Task 3: The gate — `internal/hooks/combat_formation.go`

**Files:**
- Create: `internal/hooks/combat_formation.go`
- Create: `internal/hooks/combat_formation_test.go`

**Interfaces:**
- Consumes: `formationcombat.Legal`/`InterceptFrontRow`/`Reach` (11c),
  `mobparty.Assemble`/`MemberKeyFor`/`MobSummary` (11a),
  `company.FormationFor` (Task 1), `combat.ResolveReach` (11c),
  `mobs.GetInstance`, `rooms.Room.GetMobs`.
- Produces: `func resolveAttackTarget(attackerCol int, f company.Formation, originalTarget company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) (finalTarget company.MemberKey, ok bool)`
  (pure), `func gateFormationAttack(user *users.UserRecord, defMob *mobs.Mob, room *rooms.Room) (*mobs.Mob, bool)`
  (adapter, called from `NewRound_DoCombat.go` in Task 4).

- [ ] **Step 1: Write the failing tests**

Create `internal/hooks/combat_formation_test.go`:

```go
package hooks

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

// workedExample mirrors formationcombat's own worked example: A at
// (front, col 1), B at (back, col 1), C at (back, col 2).
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

func TestResolveAttackTargetLegalDirectHitNeedsNoRedirect(t *testing.T) {
	f := workedExample(t)
	final, ok := resolveAttackTarget(1, f, keyA, allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	require.True(t, ok)
	assert.Equal(t, keyA, final)
}

func TestResolveAttackTargetInterceptsBackRowAttackToFrontRow(t *testing.T) {
	f := workedExample(t)
	final, ok := resolveAttackTarget(1, f, keyB, allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	require.True(t, ok, "A is alive and blocks column 1, so the attack redirects to A rather than being blocked")
	assert.Equal(t, keyA, final)
}

func TestResolveAttackTargetSkipsWhenFrontIsDeadAndALivingMiddleStillBlocks(t *testing.T) {
	// Interception only ever redirects to the front row. If the front slot
	// is dead (not empty — still occupied by a corpse), no interception
	// applies, and a living middle occupant still blocks plain melee from
	// reaching the back: there is no legal path this round.
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0)) // front, dead
	require.NoError(t, g.Place(keyD, 1, 0)) // middle, alive: blocks the column
	require.NoError(t, g.Place(keyB, 2, 0)) // back: the original target

	alive := map[company.MemberKey]bool{keyD: true, keyB: true} // A is dead

	_, ok := resolveAttackTarget(0, g, keyB, alive, formationcombat.ReachNone)
	assert.False(t, ok)
}

func TestResolveAttackTargetSkipsWhenTargetGoneAndNoFormationEntry(t *testing.T) {
	f := workedExample(t)
	_, ok := resolveAttackTarget(1, f, company.MemberKey("ghost"), allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	assert.False(t, ok)
}

func TestResolveAttackTargetOutOfLateralRangeSkipsEvenWithInterception(t *testing.T) {
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0))
	_, ok := resolveAttackTarget(2, g, keyA, allAlive(keyA), formationcombat.ReachAny)
	assert.False(t, ok, "column 2 attacker is out of lateral range of column 0, even with ReachAny")
}

func TestEffectiveHPMatchesRankMobsFormula(t *testing.T) {
	// 200 HP, 0 defense: no mitigation, EHP == HP.
	assert.InDelta(t, 200.0, effectiveHP(200, 0), 0.001)
	// 100 HP, 100 defense (50% mitigation): EHP == 200.
	assert.InDelta(t, 200.0, effectiveHP(100, 100), 0.001)
	// Defense clamps at 95% mitigation even for very high defense values.
	assert.InDelta(t, 100.0/0.05, effectiveHP(100, 10000), 0.001)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/hooks/... -run 'TestResolveAttackTarget|TestEffectiveHP' -v`
Expected: FAIL — `undefined: resolveAttackTarget` / `undefined: effectiveHP`.

- [ ] **Step 3: Write the implementation**

Create `internal/hooks/combat_formation.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveAttackTarget decides, given an enemy party's Formation and who is
// currently alive, which member an attack aimed at originalTarget should
// actually land on this round: front-row interception (11c) is applied
// first, then the result is checked for column-occupancy/lateral-range
// legality (11c). ok=false means the attack should be skipped entirely
// this round — the caller must leave Aggro untouched so the same target
// becomes legal again automatically once whatever blocks it dies (11c's
// self-healing model; see internal/formationcombat's own package doc).
func resolveAttackTarget(attackerCol int, f company.Formation, originalTarget company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) (finalTarget company.MemberKey, ok bool) {
	target := originalTarget
	if interceptor, intercepted := formationcombat.InterceptFrontRow(f, originalTarget, alive); intercepted {
		target = interceptor
	}
	if !formationcombat.Legal(attackerCol, f, target, alive, reach) {
		return "", false
	}
	return target, true
}

// gateFormationAttack is the engine-facing adapter around
// resolveAttackTarget for a player attacking a mob. It fails open (returns
// defMob, true — unchanged pre-existing behavior) whenever the player has
// no company formation to apply, or the target mob can't be resolved into
// an assembled enemy party at all; see this task's plan for why that's the
// deliberate contract, not a bug.
func gateFormationAttack(user *users.UserRecord, defMob *mobs.Mob, room *rooms.Room) (*mobs.Mob, bool) {
	attackerCol, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return defMob, true
	}

	party, ok := resolveEnemyParty(room, defMob.InstanceId)
	if !ok {
		return defMob, true
	}

	alive := aliveMapForParty(party)
	reach := combat.ResolveReach(&user.Character, false)
	targetKey := mobparty.MemberKeyFor(defMob.InstanceId)

	finalKey, ok := resolveAttackTarget(attackerCol, party.Formation, targetKey, alive, reach)
	if !ok {
		return nil, false
	}
	if finalKey == targetKey {
		return defMob, true
	}

	finalInstanceId, ok := mobparty.InstanceIdFromMemberKey(finalKey)
	if !ok {
		return defMob, true
	}
	finalMob := mobs.GetInstance(finalInstanceId)
	if finalMob == nil {
		return defMob, true
	}
	return finalMob, true
}

// resolvePlayerColumn returns the column of leaderUserID's own
// LeaderMemberKey within their company formation. ok is false when they
// have no company record, or (defensively) aren't placed in it.
func resolvePlayerColumn(leaderUserID int) (int, bool) {
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return 0, false
	}
	_, col, found := f.Find(company.LeaderMemberKey)
	if !found {
		return 0, false
	}
	return col, true
}

// resolveEnemyParty finds the assembled mobparty.Party (fresh, never
// cached — matching 11a's own design) that currently contains
// targetInstanceId among the room's hostile (non-charmed) mobs.
func resolveEnemyParty(room *rooms.Room, targetInstanceId int) (mobparty.Party, bool) {
	summaries := hostileMobSummaries(room)
	for _, p := range mobparty.Assemble(summaries) {
		for _, id := range p.Members {
			if id == targetInstanceId {
				return p, true
			}
		}
	}
	return mobparty.Party{}, false
}

func hostileMobSummaries(room *rooms.Room) []mobparty.MobSummary {
	var summaries []mobparty.MobSummary
	for _, instanceId := range room.GetMobs() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.IsCharmed() {
			continue
		}
		summaries = append(summaries, mobparty.MobSummary{
			InstanceId: instanceId,
			Groups:     mob.Groups,
			EHP:        effectiveHP(mob.Character.HealthMax.Value, mob.Character.GetDefense()),
		})
	}
	return summaries
}

// aliveMapForParty reports, for every member of an assembled party,
// whether its live mob instance still exists and has positive HP.
func aliveMapForParty(p mobparty.Party) map[company.MemberKey]bool {
	alive := make(map[company.MemberKey]bool, len(p.Members))
	for _, id := range p.Members {
		mob := mobs.GetInstance(id)
		alive[mobparty.MemberKeyFor(id)] = mob != nil && mob.Character.Health > 0
	}
	return alive
}

// effectiveHP mirrors combat.RankMobs' own EHP formula
// (internal/combat/mob_rank.go:164-171), applied directly to a live
// combatant's current HealthMax/Defense instead of a simulated spec —
// cheap enough to call once per hostile mob per round, unlike RankMobs
// itself which ranks every mob spec in the game.
func effectiveHP(hp, defense int) float64 {
	defFrac := float64(defense) / 200.0
	if defFrac > 0.95 {
		defFrac = 0.95
	}
	if defFrac < 0 {
		defFrac = 0
	}
	return float64(hp) / (1.0 - defFrac)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hooks/... -run 'TestResolveAttackTarget|TestEffectiveHP' -v`
Expected: PASS, all six tests.

- [ ] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/hooks && go vet ./internal/hooks/... && go build ./...
```
Expected: no output, clean build (confirms no import cycle: `internal/hooks`
already imports `internal/combat`/`internal/mobs`/`internal/rooms`/
`internal/users`; the two new imports, `internal/company` and
`internal/mobparty`/`internal/formationcombat`, have no reverse dependency
on `internal/hooks`).

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/combat_formation.go internal/hooks/combat_formation_test.go
git commit -m "feat(hooks): add formation-aware attack target resolution"
```

## Task 4: Wire the gate into `NewRound_DoCombat.go`

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go`

- [ ] **Step 1: Insert the gate call**

In `internal/hooks/NewRound_DoCombat.go`, immediately after the existing
"can't see them, can't fight them" hidden-buff check and before
`affectedPlayerIds` is appended (current lines `551-556`):

```go
			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag("hidden") {
				user.SendText("You can't seem to find your target.")
				continue
			}

			if gated, gateOk := gateFormationAttack(user, defMob, uRoom); !gateOk {
				user.SendText("You can't reach that target from here.")
				continue
			} else {
				defMob = gated
			}

			affectedPlayerIds = append(affectedPlayerIds, user.Character.Aggro.UserId)
```

This is the only change to this file. `defMob` is reassigned (not
redeclared) so every later reference in this branch — `combat.AttackPlayerVsMob(user, defMob)`,
buff/message dispatch, hostility/aggro-acquisition logic — automatically
uses the (possibly redirected) final target, with zero further edits
needed below this point.

- [ ] **Step 2: Build and run the full test suite**

```bash
go build ./...
go test -race ./...
```
Expected: clean build, full suite green (record the actual test/package
counts — don't guess).

- [ ] **Step 3: `make generate`/`make validate`**

```bash
make generate
make validate
```
Expected: both succeed, `make generate` produces no diff (no new module,
no new command, no config change).

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "feat(hooks): gate player-vs-mob attacks on formation legality"
```

## Task 5: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [ ] **Step 1: Final full verification**

```bash
go test -race ./...
make generate
make validate
```

- [ ] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a work-log entry ("Formation combat-loop wiring: player-vs-mob
(complete; three directions still deferred, <date>)") following the
established format. Update the `## Current position`/`**HEAD:**`/"Next"
lines to reflect that player-vs-mob is now live, and name the three still-
deferred directions (mob-vs-player interception needing cross-`Attack*`
dispatch, mob-vs-mob needing charmed/hostile disambiguation, and 11b's
`AssignTarget` reassignment-on-death needing the existing target-validation
block modified) as the next concrete follow-up items — no phase number
assigned yet, same as before this pass.

- [ ] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record formation-aware player-vs-mob combat wiring"
```

---

## Self-Review Notes

- **Spec coverage:** front-row interception (11c) ✓ wired and tested via
  `resolveAttackTarget`. Column-occupancy/lateral-range legality (11c) ✓
  same. Self-healing without clearing `Aggro` ✓ (`gateFormationAttack`
  never touches `user.Character.Aggro`, only decides which mob the attack
  call targets or whether to `continue`). "Attacking a member engages the
  attacker's whole company" (11b) and reassignment-on-death (11b) are
  explicitly NOT covered — see Design Decision 4; this is a stated scope
  boundary, not an oversight. Adjacency/`Reach` trait (11c) ✓ consumed via
  `combat.ResolveReach`.
- **Placeholder scan:** no TBD/TODO, every step has real code.
- **Type consistency:** `resolveAttackTarget`, `gateFormationAttack`,
  `resolvePlayerColumn`, `resolveEnemyParty`, `hostileMobSummaries`,
  `aliveMapForParty`, `effectiveHP` are used identically between Task 3's
  definitions and Task 4's one call site.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11-combat-wiring.md`.**
