# Phase 11 Mob-vs-Mob Combat Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the third of the four attack-direction call sites: when
a hostile mob and a companion mob fight (either direction — a companion
attacking an enemy, or an enemy attacking a companion), 11c's
legality/interception now gates and redirects the attack exactly as it
already does for player-vs-mob and mob-vs-player. Two-hostile-mob and
two-companion combat (both already handled without any formation concept)
stay untouched.

**This is not a numbered phase** — same status as the previous two combat
wiring passes: gluing already-shipped, already-tested pieces together, no
new gameplay rules. No separate spec doc; this plan's "Design decisions"
covers what's new.

## Design decisions

**1. Disambiguating the two sides.** `mob-vs-mob`
(`NewRound_DoCombat.go:978+`) fires for *any* two mobs, regardless of
whether either is a companion. This pass adds one new seam query,
`company.LeaderAndKeyForInstance(instanceId int) (leaderUserID int, key company.MemberKey, found bool)`
(wraps `modules/company`'s existing private `companionForInstance` reverse
lookup — already used internally for `onMobDeath`, now exposed for combat)
to classify each side as "a currently-attached companion of some leader"
or not. Four combinations follow:

| Attacker | Defender | Handling |
|---|---|---|
| companion | hostile | **wired this pass**: gate as "my companion attacks the enemy party" |
| hostile | companion | **wired this pass**: gate as "the enemy attacks my company" |
| hostile | hostile | untouched — no formation concept applies to two independent hostile mobs |
| companion | companion | untouched (defensive fallback; this shouldn't normally arise) |

**2. Both new gates only ever need `combat.AttackMobVsMob` — no cross-type
redirect, with one documented exception.** Unlike the mob-vs-player pass
(which needed a redirect from `AttackMobVsPlayer` to `AttackMobVsMob`),
both directions here start and end as mob-vs-mob: a companion attacking an
enemy might get redirected to a *different* enemy party member (still a
mob); an enemy attacking a companion might get redirected to a *different*
companion (still a mob). Both gates therefore reuse the simpler
"return a substitute `*mobs.Mob`, or `ok=false` to skip" shape
`gateFormationAttack` already uses, not the "resolve the attack myself"
shape `gateMobVsPlayerAttack` needed. **The one exception:** a company's
own `Formation` can place the leader (a real player) anywhere, including
the front row — so an attack aimed at a companion could, in principle,
intercept to the *leader*, which genuinely would need the cross-type
redirect all over again. This pass explicitly does not implement that:
`resolveAttackTargetCompanionOnly` (a two-line variant of the existing
`resolveAttackTarget`) skips the redirect specifically when the
interceptor would be `company.LeaderMemberKey`, falling back to checking
legality against the original (companion) target directly instead — never
breaking anything, just not claiming an interception opportunity that
would require solving the same cross-type problem a third time. A
formation with the leader genuinely posted in front of their own
companions is an unusual (if valid) arrangement; this is a narrow,
explicitly documented gap, not silently dropped.

**3. `gateFormationAttack` gets refactored (not behaviorally changed) to
share its core with the new companion-attacks-enemy gate.**
`gateFormationAttack`'s body (resolve the enemy party, apply
`resolveAttackTarget`, translate the result back into a `*mobs.Mob`) is
extracted into `resolveEnemyAttack(attackerCol int, defenderInstanceId int, room *rooms.Room, reach formationcombat.Reach) (*mobs.Mob, bool)`,
identical logic, just parameterized so `gateCompanionAttacksEnemy` (this
pass) can call the same code instead of duplicating it. `gateFormationAttack`
itself becomes a two-line caller of the extracted function — this is a
pure refactor with no behavior change, verified by the full existing test
suite passing unchanged (there are no direct unit tests of
`gateFormationAttack` to update — it was never unit-tested directly, only
`resolveAttackTarget`, which this refactor doesn't touch at all).

## Global Constraints

- Never advances `gametime`, round count, or damage math.
- `Aggro` is never mutated by either new gate — a blocked round is
  skipped, a redirected round attacks a different mob, in both cases
  `mob.Character.Aggro` is left exactly as it was.
- Fails open: no company record, attacker/defender not resolvable into a
  company or enemy party, or (the one documented exception above) an
  interception that would require redirecting to the leader — all mean
  the original `combat.AttackMobVsMob(mob, defMob)` call proceeds
  unchanged.
- Formation is read-only — nothing here mutates a `Formation`, `Aggro`, or
  a room's mob list.

---

## File Structure

- Modify: `internal/company/provider.go` — extend `FormationProvider` with
  `LeaderAndKeyForInstance`.
- Modify: `internal/company/provider_test.go` — call-through test for it.
- Modify: `modules/company/company.go` — add the thin
  `LeaderAndKeyForInstance` method delegating to `companionForInstance`.
- Modify: `internal/hooks/combat_formation.go` — extract
  `resolveEnemyAttack` from `gateFormationAttack`; add
  `resolveAttackTargetCompanionOnly`, `gateMobVsMobAttack`,
  `gateCompanionAttacksEnemy`, `gateEnemyAttacksCompanion`.
- Modify: `internal/hooks/combat_formation_test.go` — tests for
  `resolveAttackTargetCompanionOnly`.
- Modify: `internal/hooks/NewRound_DoCombat.go` — insert the gate call at
  the mob-vs-mob attack site.

## Task 1: `LeaderAndKeyForInstance` seam method

**Files:**
- Modify: `internal/company/provider.go`
- Modify: `internal/company/provider_test.go`
- Modify: `modules/company/company.go`

**Interfaces:**
- Produces: extended `FormationProvider.LeaderAndKeyForInstance(instanceId int) (leaderUserID int, key MemberKey, found bool)`,
  `func LeaderAndKeyForInstance(instanceId int) (int, MemberKey, bool)`
  (package-level call-through), `func (m *CompanyModule) LeaderAndKeyForInstance(instanceId int) (int, domain.MemberKey, bool)`.

- [ ] **Step 1: Write the failing test**

Add to `internal/company/provider_test.go`:

```go
type fakeLeaderLookup struct {
	fakeFormationProvider
	leaderUserID int
	key          company.MemberKey
	found        bool
}

func (f fakeLeaderLookup) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	return f.leaderUserID, f.key, f.found
}

func TestLeaderAndKeyForInstanceReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, _, ok := company.LeaderAndKeyForInstance(9)
	assert.False(t, ok)
}

func TestLeaderAndKeyForInstanceCallsThroughToRegisteredProvider(t *testing.T) {
	company.SetFormationProvider(fakeLeaderLookup{leaderUserID: 7, key: company.CompanionMemberKey(3), found: true})
	defer company.SetFormationProvider(nil)

	leaderUserID, key, ok := company.LeaderAndKeyForInstance(9)
	require.True(t, ok)
	assert.Equal(t, 7, leaderUserID)
	assert.Equal(t, company.CompanionMemberKey(3), key)
}
```

Also add a trivial `LeaderAndKeyForInstance` method to the existing
`fakeFormationProvider` type (so it still satisfies the now three-method
interface):

```go
func (f fakeFormationProvider) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	return 0, "", false
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/company/... -run TestLeaderAndKeyForInstance -v`
Expected: FAIL — `undefined: company.LeaderAndKeyForInstance` /
interface-not-implemented compile error.

- [ ] **Step 3: Write the implementation**

In `internal/company/provider.go`, extend the interface and add the
call-through:

```go
	InstanceFor(leaderUserID, companionID int) (instanceId int, ok bool)

	// LeaderAndKeyForInstance returns the leader and formation MemberKey a
	// live mob instance is currently attached to as a companion. found is
	// false for anything that isn't a currently-attached companion of any
	// tracked company — a hostile mob, a detached/dismissed instance, or a
	// mob charmed outside the company system entirely.
	LeaderAndKeyForInstance(instanceId int) (leaderUserID int, key MemberKey, found bool)
}
```

```go
// LeaderAndKeyForInstance calls through to the registered
// FormationProvider. See FormationFor for the no-provider-registered
// contract.
func LeaderAndKeyForInstance(instanceId int) (int, MemberKey, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return 0, "", false
	}
	return p.LeaderAndKeyForInstance(instanceId)
}
```

In `modules/company/company.go`, add near the existing `InstanceFor`
method:

```go
// LeaderAndKeyForInstance implements company.FormationProvider's third
// query, delegating to the existing private reverse lookup this module
// already maintains for onMobDeath.
func (m *CompanyModule) LeaderAndKeyForInstance(instanceId int) (int, domain.MemberKey, bool) {
	leaderUserID, companionID, ok := m.companionForInstance(instanceId)
	if !ok {
		return 0, "", false
	}
	return leaderUserID, domain.CompanionMemberKey(companionID), true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company/... ./modules/company/... -v`
Expected: PASS, all new tests plus every pre-existing test.

- [ ] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/company modules/company
go vet ./internal/company/... ./modules/company/...
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/company/provider.go internal/company/provider_test.go modules/company/company.go
git commit -m "feat(company): add LeaderAndKeyForInstance seam query"
```

## Task 2: The two mob-vs-mob gates

**Files:**
- Modify: `internal/hooks/combat_formation.go`
- Modify: `internal/hooks/combat_formation_test.go`

**Interfaces:**
- Consumes: `resolveAttackTarget`, `resolveEnemyParty`, `aliveMapForParty`,
  `aliveMapForCompany`, `resolveHostileAttackerColumn` (all already exist),
  `company.LeaderAndKeyForInstance` (Task 1).
- Produces: `resolveEnemyAttack` (extracted), `resolveAttackTargetCompanionOnly`,
  `gateMobVsMobAttack(mob, defMob *mobs.Mob, mobRoom *rooms.Room) (*mobs.Mob, bool)`
  (called from `NewRound_DoCombat.go` in Task 3), `gateCompanionAttacksEnemy`,
  `gateEnemyAttacksCompanion`.

- [ ] **Step 1: Write the failing test**

Add to `internal/hooks/combat_formation_test.go`:

```go
func TestResolveAttackTargetCompanionOnlySkipsRedirectToLeader(t *testing.T) {
	// Leader posted front-row, a companion behind them in the same column.
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(keyD, 2, 0)) // "companion", back row
	alive := map[company.MemberKey]bool{company.LeaderMemberKey: true, keyD: true}

	// An attacker targeting the companion directly would normally be
	// intercepted by the leader (front row, same column) -- but this
	// variant must NOT redirect to the leader, only check legality
	// against the original target.
	final, ok := resolveAttackTargetCompanionOnly(0, f, keyD, alive, formationcombat.ReachNone)
	assert.False(t, ok, "blocked by a living leader in front, and no companion-side interception exists to rescue it")
	_ = final
}

func TestResolveAttackTargetCompanionOnlyStillInterceptsBetweenCompanions(t *testing.T) {
	// Two companions in the same column, leader elsewhere: companion-to-
	// companion interception still works normally.
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 1))
	require.NoError(t, f.Place(keyA, 0, 0)) // front companion
	require.NoError(t, f.Place(keyB, 2, 0)) // back companion
	alive := map[company.MemberKey]bool{company.LeaderMemberKey: true, keyA: true, keyB: true}

	final, ok := resolveAttackTargetCompanionOnly(1, f, keyB, alive, formationcombat.ReachNone)
	require.True(t, ok)
	assert.Equal(t, keyA, final)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/hooks/... -run TestResolveAttackTargetCompanionOnly -v`
Expected: FAIL — `undefined: resolveAttackTargetCompanionOnly`.

- [ ] **Step 3: Write the implementation**

In `internal/hooks/combat_formation.go`, replace the existing
`gateFormationAttack` function with the extracted-core version, and add
the new functions:

```go
// gateFormationAttack is the engine-facing adapter around
// resolveAttackTarget for a player attacking a mob. See resolveEnemyAttack
// for the fail-open contract shared with gateCompanionAttacksEnemy.
func gateFormationAttack(user *users.UserRecord, defMob *mobs.Mob, room *rooms.Room) (*mobs.Mob, bool) {
	attackerCol, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return defMob, true
	}
	reach := combat.ResolveReach(user.Character, false)
	return resolveEnemyAttack(attackerCol, defMob.InstanceId, room, reach)
}

// resolveEnemyAttack applies 11c legality/interception when a combatant in
// attackerCol attacks defenderInstanceId's assembled enemy party within
// room. It fails open (returns the original defender, true) whenever the
// defender can't be resolved into an assembled enemy party at all.
func resolveEnemyAttack(attackerCol int, defenderInstanceId int, room *rooms.Room, reach formationcombat.Reach) (*mobs.Mob, bool) {
	original := mobs.GetInstance(defenderInstanceId)

	party, ok := resolveEnemyParty(room, defenderInstanceId)
	if !ok {
		return original, true
	}

	alive := aliveMapForParty(party)
	targetKey := mobparty.MemberKeyFor(defenderInstanceId)

	finalKey, ok := resolveAttackTarget(attackerCol, party.Formation, targetKey, alive, reach)
	if !ok {
		return nil, false
	}
	if finalKey == targetKey {
		return original, true
	}

	finalInstanceId, ok := mobparty.InstanceIdFromMemberKey(finalKey)
	if !ok {
		return original, true
	}
	finalMob := mobs.GetInstance(finalInstanceId)
	if finalMob == nil {
		return original, true
	}
	return finalMob, true
}

// resolveAttackTargetCompanionOnly is resolveAttackTarget's counterpart
// for an attack aimed at a company (not enemy-party) member: it applies
// front-row interception exactly the same way, EXCEPT it never redirects
// to company.LeaderMemberKey — a leader-as-interceptor would need to
// switch combat.Attack* functions mid-resolution (AttackMobVsMob to
// AttackMobVsPlayer), which this pass doesn't implement (see this plan's
// Design Decision 2). When the would-be interceptor is the leader, it
// falls back to checking legality against the original target directly,
// as if no interceptor existed — never breaking anything, just not
// claiming that one specific (and unusual: leader posted in front of
// their own companions) interception opportunity.
func resolveAttackTargetCompanionOnly(attackerCol int, f company.Formation, originalTarget company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) (finalTarget company.MemberKey, ok bool) {
	target := originalTarget
	if interceptor, intercepted := formationcombat.InterceptFrontRow(f, originalTarget, alive); intercepted && interceptor != company.LeaderMemberKey {
		target = interceptor
	}
	if !formationcombat.Legal(attackerCol, f, target, alive, reach) {
		return "", false
	}
	return target, true
}

// gateMobVsMobAttack classifies both sides of a mob-vs-mob attack as a
// company member or not, and dispatches to the matching gate. Two hostile
// mobs, or two companions, fighting each other is left untouched — no
// formation concept applies to either.
func gateMobVsMobAttack(mob, defMob *mobs.Mob, mobRoom *rooms.Room) (*mobs.Mob, bool) {
	attackerLeaderId, attackerKey, attackerIsCompanion := company.LeaderAndKeyForInstance(mob.InstanceId)
	defenderLeaderId, defenderKey, defenderIsCompanion := company.LeaderAndKeyForInstance(defMob.InstanceId)

	switch {
	case attackerIsCompanion && !defenderIsCompanion:
		return gateCompanionAttacksEnemy(mob, attackerLeaderId, attackerKey, defMob, mobRoom)
	case !attackerIsCompanion && defenderIsCompanion:
		return gateEnemyAttacksCompanion(mob, defMob, mobRoom, defenderLeaderId, defenderKey)
	default:
		return defMob, true
	}
}

// gateCompanionAttacksEnemy gates "my companion attacks the enemy party" —
// the mob-vs-mob analog of gateFormationAttack, sharing its core via
// resolveEnemyAttack.
func gateCompanionAttacksEnemy(mob *mobs.Mob, leaderUserID int, attackerKey company.MemberKey, defMob *mobs.Mob, mobRoom *rooms.Room) (*mobs.Mob, bool) {
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return defMob, true
	}
	_, col, found := f.Find(attackerKey)
	if !found {
		return defMob, true
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	return resolveEnemyAttack(col, defMob.InstanceId, mobRoom, reach)
}

// gateEnemyAttacksCompanion gates "the enemy attacks my companion" — the
// mob-vs-mob analog of gateMobVsPlayerAttack's legality/interception half,
// generalized to any company member (not just the leader) and restricted
// to companion-to-companion interception (see resolveAttackTargetCompanionOnly).
func gateEnemyAttacksCompanion(mob, defMob *mobs.Mob, mobRoom *rooms.Room, leaderUserID int, defenderKey company.MemberKey) (*mobs.Mob, bool) {
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return defMob, true
	}
	attackerCol, ok := resolveHostileAttackerColumn(mobRoom, mob.InstanceId)
	if !ok {
		return defMob, true
	}
	leader := users.GetByUserId(leaderUserID)
	if leader == nil {
		return defMob, true
	}
	alive := aliveMapForCompany(leader, f)
	reach := combat.ResolveReach(&mob.Character, mob.Reach)

	finalKey, ok := resolveAttackTargetCompanionOnly(attackerCol, f, defenderKey, alive, reach)
	if !ok {
		return nil, false
	}
	if finalKey == defenderKey {
		return defMob, true
	}

	companionID, ok := company.CompanionIDFromMemberKey(finalKey)
	if !ok {
		return defMob, true
	}
	instanceId, ok := company.InstanceFor(leaderUserID, companionID)
	if !ok {
		return defMob, true
	}
	finalMob := mobs.GetInstance(instanceId)
	if finalMob == nil {
		return defMob, true
	}
	return finalMob, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hooks/... -v`
Expected: PASS, all existing tests (confirming the `gateFormationAttack`
refactor didn't break anything) plus the two new
`resolveAttackTargetCompanionOnly` tests.

- [ ] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/hooks && go vet ./internal/hooks/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/combat_formation.go internal/hooks/combat_formation_test.go
git commit -m "feat(hooks): add mob-vs-mob formation gating"
```

## Task 3: Wire the gate into `NewRound_DoCombat.go`

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go`

- [ ] **Step 1: Insert the gate call**

In the mob-vs-mob block, immediately after the "can't see them, hidden"
check and before `var roundResult combat.AttackResult` /
`combat.AttackMobVsMob(mob, defMob)`:

```go
			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag("hidden") {
				continue
			}

			if gated, gateOk := gateMobVsMobAttack(mob, defMob, mobRoom); !gateOk {
				continue
			} else {
				defMob = gated
			}

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsMob(mob, defMob)
```

`defMob` is reassigned, not redeclared, so every later reference in this
block (buff dispatch, messages, `onHurt` scripting, equipment-break, the
final `SetAggro(0, defMob.InstanceId, ...)`/`EndAggro` logic) automatically
uses the possibly-redirected final target. This is the only change to this
file for this task.

- [ ] **Step 2: Build and run the full test suite**

```bash
go build ./...
go test -race ./...
```
Expected: clean build, full suite green (record actual counts).

- [ ] **Step 3: `make generate`/`make validate`**

```bash
make generate
make validate
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "feat(hooks): gate mob-vs-mob attacks on formation legality"
```

## Task 4: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [ ] **Step 1: Final full verification**

```bash
go test -race ./...
make generate
make validate
```

- [ ] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a work-log entry ("Formation combat-loop wiring: mob-vs-mob (complete;
one direction still deferred, <date>)") following the established format.
Update `## Current position`/`**HEAD:**`/"Next" to reflect that all three
attack directions between a player's company and a hostile party are now
wired (player-vs-mob, mob-vs-player, mob-vs-mob), leaving only 11b's
reassignment-on-death and player-vs-player (explicit PvP, always
out-of-scope) unwired.

- [ ] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record mob-vs-mob combat wiring"
```

---

## Self-Review Notes

- **Spec coverage:** companion-attacks-enemy legality/interception ✓
  (`gateCompanionAttacksEnemy`, shares tested core with
  `gateFormationAttack` via `resolveEnemyAttack`). Enemy-attacks-companion
  legality/interception ✓ (`gateEnemyAttacksCompanion`, tested via
  `resolveAttackTargetCompanionOnly`'s two new tests). Two-hostile and
  two-companion combat explicitly untouched ✓ (`gateMobVsMobAttack`'s
  `default` case). The leader-as-interceptor edge case is explicitly
  scoped out with a real, documented reason (Design Decision 2), not
  silently missing.
- **Placeholder scan:** no TBD/TODO, every step has real code.
- **Type consistency:** `resolveEnemyAttack`, `resolveAttackTargetCompanionOnly`,
  `gateMobVsMobAttack`, `gateCompanionAttacksEnemy`, `gateEnemyAttacksCompanion`,
  `LeaderAndKeyForInstance` match between their Task 1/2 definitions and
  Task 3's one call site.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11-mob-vs-mob.md`.**
