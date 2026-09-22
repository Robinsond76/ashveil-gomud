# Phase 11 Reassignment-on-Death Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Status: fully implemented and committed on `phase-11-reassignment` — every step below is checked off.**

**Goal:** Complete 11b's last unwired acceptance criterion: when a company
member's mob target dies (or otherwise becomes permanently invalid), they
now pick a new legal target within the same enemy party via
`engagement.AssignTarget`, instead of simply clearing `Aggro` and giving
up. This is the piece all three interception passes explicitly deferred
because it requires modifying the existing target-validation block that
clears `Aggro` and `continue`s the round *before* any reassignment could
run.

**This is not a numbered phase** — same status as the three interception
passes: wiring an already-shipped, already-tested domain function
(`engagement.AssignTarget`, shipped since 11b, never actually called by
any real code path until this plan) into the combat loop. No separate spec
doc; this plan's "Design decisions" covers what's new.

## Design decisions

**1. Scope: the two "a company member attacks an enemy" directions only,
matching what `engagement.AssignTarget` is actually for.** 11b's spec
frames reassignment as *"a company member's target dies... they get a new
target."* This is about a company member's own engagement, not about
sharpening enemy AI. So this plan touches exactly the two places a company
member (the player leader, or a companion) loses a mob target: the
player-vs-mob branch's existing dead-target checks
(`NewRound_DoCombat.go:511`, `:521`) and the mob-vs-mob branch's dead-target
checks (`NewRound_DoCombat.go:985`, `:995`) — but only when the *attacker*
in that mob-vs-mob branch is a companion (`company.LeaderAndKeyForInstance`
resolves it), not when it's a hostile mob whose target died. Enemy AI
retargeting stays out of scope, exactly as the interception passes already
scoped it (11d territory).

**2. Reassignment happens fresh from whatever hostile party is currently
in the room, not "the exact same party the dead mob belonged to."** Once
a target mob is gone (`mobs.GetInstance` returns `nil`), there is no
longer any way to ask "which party was it in" — `mobparty.Assemble` is
never cached (11a's own design), so the identity of a since-despawned
mob's party can't be recovered. This plan reassembles the room's hostile
mobs (`mobparty.Assemble(hostileMobSummaries(room))`) and picks the first
resulting party if one exists. In the overwhelmingly common case (one
hostile party in the room) this is exactly "the same party." Multiple
simultaneous unrelated hostile parties in one room is an edge case 11a
itself didn't design targeting behavior for either; picking the first
assembled party (a stable, deterministic order) is a reasonable, simple
default — not full multi-party-aware AI, matching 11b's own "minimal
preference rule, not full personality" framing.

**3. On success, the caller sets the new target and skips attacking this
round — combat resumes normally next round.** Reassignment sets
`Aggro`/`SetAggro` to the new target and `continue`s, rather than trying to
also resolve an attack against the new target in the same pass (which
would mean re-running all of this round's earlier gating/hidden/RoundsWaiting
checks against a target that wasn't validated against them). The next
round's normal flow (already gated by the three interception passes)
attacks the new target like any other round.

**4. New shared helpers in `internal/hooks/combat_formation.go`**,
consistent with where every other piece of this wiring already lives:
`firstHostilePartyInRoom`, `partyCombatants` (adapts a `mobparty.Party`'s
members into `[]engagement.Combatant` using live HP and formation
position), `reassignEnemyTarget` (the actual `engagement.AssignTarget`
call, pure enough to unit test with a hand-built room-free party), and two
thin engine-facing wrappers, `reassignPlayerTarget`/`reassignCompanionTarget`,
called from `NewRound_DoCombat.go`.

## Global Constraints

- Never advances `gametime`, round count, or damage math.
- Reassignment never fabricates combat: it only picks a legal existing
  target and sets `Aggro`; it never deals damage, moves anyone, or skips
  the normal per-round attack gating for the new target on subsequent
  rounds.
- Fails closed to today's exact behavior: no company formation, no hostile
  party currently in the room, or no living legal candidate within it,
  all mean the existing "target lost" message + `Aggro = nil` + `continue`
  runs completely unchanged — nothing about this plan can make a
  previously-working "give up" path behave differently when reassignment
  isn't possible.

---

## File Structure

- Modify: `internal/hooks/combat_formation.go` — add
  `firstHostilePartyInRoom`, `partyCombatants`, `reassignEnemyTarget`,
  `reassignPlayerTarget`, `reassignCompanionTarget`.
- Modify: `internal/hooks/combat_formation_test.go` — tests for
  `reassignEnemyTarget` (the one function here with real decision logic,
  not just engine plumbing).
- Modify: `internal/hooks/NewRound_DoCombat.go` — insert
  `reassignPlayerTarget` at the player-vs-mob dead-target checks, and
  `reassignCompanionTarget` at the mob-vs-mob dead-target checks.

## Task 1: Reassignment helpers

**Files:**
- Modify: `internal/hooks/combat_formation.go`
- Modify: `internal/hooks/combat_formation_test.go`

**Interfaces:**
- Consumes: `engagement.AssignTarget`/`Combatant`/`Weakest` (11b, already
  exists, never previously called from real code), `mobparty.Assemble`/
  `MemberKeyFor` (11a), `formationcombat.Legal` (11c), `hostileMobSummaries`/
  `aliveMapForParty`/`resolvePlayerColumn` (already exist, previous
  passes), `company.LeaderAndKeyForInstance`/`FormationFor` (already
  exist).
- Produces: `func reassignEnemyTarget(attackerCol int, reach formationcombat.Reach, room *rooms.Room) (targetInstanceId int, ok bool)`
  (the core decision logic), `func reassignPlayerTarget(user *users.UserRecord, room *rooms.Room) bool`,
  `func reassignCompanionTarget(mob *mobs.Mob, room *rooms.Room) bool`
  (both called from `NewRound_DoCombat.go` in Task 2).

- [x] **Step 1: Write the failing test**

Add to `internal/hooks/combat_formation_test.go`. This test can't build a
real `*rooms.Room` cheaply, so it exercises `reassignEnemyTarget`'s
decision core indirectly isn't possible without engine state — instead,
test the one genuinely pure sub-piece, `partyCombatants`, which needs no
room/mob globals once given a `mobparty.Party` and an `alive` map (mob
lookups inside it are the only engine dependency, so this test documents
that boundary rather than pretending full coverage):

```go
func TestPartyCombatantsUsesFormationPositionAndAliveHP(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 1))
	require.NoError(t, f.Place(keyB, 2, 1))

	party := mobparty.Party{Members: nil, Formation: f}
	// partyCombatants resolves live HP via mobs.GetInstance, which returns
	// nil for instance IDs that were never spawned in this test process --
	// this documents the "gone mob reports HP 0" contract without needing
	// a real mob registry.
	party.Members = []int{1, 2}

	combatants := partyCombatants(party, map[company.MemberKey]bool{keyA: true, keyB: true})

	require.Len(t, combatants, 2)
	for _, c := range combatants {
		assert.Equal(t, 0, c.HP, "no mob instance 1/2 exists in this test process")
	}
}
```

Note: `keyA`/`keyB` here reuse the same package-level test constants
already declared in this file from the previous passes
(`company.MemberKey("a")`/`("b")`) — do not redeclare them.

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/hooks/... -run TestPartyCombatants -v`
Expected: FAIL — `undefined: partyCombatants`.

- [x] **Step 3: Write the implementation**

Add to `internal/hooks/combat_formation.go`:

```go
// firstHostilePartyInRoom returns the first assembled hostile party
// currently in room, if any. mobparty.Assemble is never cached (11a's own
// design), so this is always a fresh snapshot.
func firstHostilePartyInRoom(room *rooms.Room) (mobparty.Party, bool) {
	parties := mobparty.Assemble(hostileMobSummaries(room))
	if len(parties) == 0 {
		return mobparty.Party{}, false
	}
	return parties[0], true
}

// partyCombatants adapts an assembled party's members into
// engagement.Combatant values (live HP, formation row/col) for
// engagement.AssignTarget. A member with no live mob instance reports
// HP 0, which AssignTarget already treats as ineligible.
func partyCombatants(party mobparty.Party, alive map[company.MemberKey]bool) []engagement.Combatant {
	combatants := make([]engagement.Combatant, 0, len(party.Members))
	for _, id := range party.Members {
		hp := 0
		if mob := mobs.GetInstance(id); mob != nil {
			hp = mob.Character.Health
		}
		row, col, _ := party.Formation.Find(mobparty.MemberKeyFor(id))
		combatants = append(combatants, engagement.Combatant{ID: id, HP: hp, Row: row, Col: col})
	}
	return combatants
}

// reassignEnemyTarget picks a new legal target (11b's weakest-HP
// preference) for an attacker in attackerCol, from whichever hostile
// party is currently in room. ok=false means no hostile party is present,
// or none of its members are both alive and legal — the caller must fall
// back to its existing "target lost, give up" behavior unchanged.
func reassignEnemyTarget(attackerCol int, reach formationcombat.Reach, room *rooms.Room) (int, bool) {
	party, ok := firstHostilePartyInRoom(room)
	if !ok {
		return 0, false
	}

	alive := aliveMapForParty(party)
	candidates := partyCombatants(party, alive)

	legal := func(attacker, defender engagement.Combatant) bool {
		return formationcombat.Legal(attackerCol, party.Formation, mobparty.MemberKeyFor(defender.ID), alive, reach)
	}

	attacker := engagement.Combatant{Col: attackerCol}
	return engagement.AssignTarget(attacker, candidates, engagement.Weakest, legal)
}

// reassignPlayerTarget attempts 11b's reassignment-on-target-loss for a
// player whose current mob target just became invalid. On success it sets
// a new Aggro target and returns true — the caller skips its own "target
// lost" message/clear, and combat resumes normally next round against the
// new target. false means unchanged pre-existing behavior: no company
// formation, or no living legal replacement in any hostile party
// currently in the room.
func reassignPlayerTarget(user *users.UserRecord, room *rooms.Room) bool {
	col, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return false
	}
	reach := combat.ResolveReach(user.Character, false)
	newTargetId, ok := reassignEnemyTarget(col, reach, room)
	if !ok {
		return false
	}
	user.Character.SetAggro(0, newTargetId, characters.DefaultAttack)
	events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
	return true
}

// reassignCompanionTarget is reassignPlayerTarget's companion-mob
// counterpart. It only applies when mob is a currently-attached company
// member (hostile mobs whose own target died are not reassigned — that's
// enemy AI, out of scope; see this plan's Design Decision 1).
func reassignCompanionTarget(mob *mobs.Mob, room *rooms.Room) bool {
	leaderUserID, key, isCompanion := company.LeaderAndKeyForInstance(mob.InstanceId)
	if !isCompanion {
		return false
	}
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return false
	}
	_, col, found := f.Find(key)
	if !found {
		return false
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	newTargetId, ok := reassignEnemyTarget(col, reach, room)
	if !ok {
		return false
	}
	mob.Character.SetAggro(0, newTargetId, characters.DefaultAttack)
	events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
	return true
}
```

Add `"github.com/GoMudEngine/GoMud/internal/engagement"` to this file's
imports.

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hooks/... -v`
Expected: PASS, all existing tests plus the new `TestPartyCombatants*`
test.

- [x] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/hooks && go vet ./internal/hooks/... && go build ./...
```

- [x] **Step 6: Commit**

```bash
git add internal/hooks/combat_formation.go internal/hooks/combat_formation_test.go
git commit -m "feat(hooks): add 11b reassignment-on-death helpers"
```

## Task 2: Wire reassignment into the two dead-target checks

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go`

- [x] **Step 1: Wire the player-vs-mob dead-target checks**

Replace the player-vs-mob branch's two existing clear-Aggro spots:

```go
			if !targetFound {
				user.SendText("Your target can't be found.")
				user.Character.Aggro = nil
				continue
			}
```

becomes:

```go
			if !targetFound {
				if reassignPlayerTarget(user, uRoom) {
					continue
				}
				user.SendText("Your target can't be found.")
				user.Character.Aggro = nil
				continue
			}
```

and:

```go
			if defMob.Character.Health < 1 {
				user.SendText("Your rage subsides.")
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}
```

becomes:

```go
			if defMob.Character.Health < 1 {
				if reassignPlayerTarget(user, uRoom) {
					continue
				}
				user.SendText("Your rage subsides.")
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}
```

- [x] **Step 2: Wire the mob-vs-mob dead-target checks**

Replace the mob-vs-mob branch's two existing clear-Aggro spots:

```go
			if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}
```

becomes:

```go
			if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
				if reassignCompanionTarget(mob, mobRoom) {
					continue
				}
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}
```

and:

```go
			if defMob.Character.Health < 1 {
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}
```

becomes:

```go
			if defMob.Character.Health < 1 {
				if reassignCompanionTarget(mob, mobRoom) {
					continue
				}
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}
```

Note: `internal/hooks/NewRound_DoCombat.go` has two other structurally
identical `defMob == nil`/`Health < 1` clear-Aggro pairs — one in the
player-vs-player branch, one in the mob-vs-player branch. **Do not touch
either of those** — reassignment only applies where the attacker is a
company member targeting an enemy party (Design Decision 1); player-vs-
player is explicit PvP (always out of scope), and mob-vs-player's target
is the leader, not an enemy mob, so there is nothing to reassign among.

- [x] **Step 3: Build and run the full test suite**

```bash
go build ./...
go test -race ./...
```
Expected: clean build, full suite green (record actual counts).

- [x] **Step 4: `make generate`/`make validate`**

```bash
make generate
make validate
```

- [x] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "feat(hooks): wire 11b reassignment-on-death into combat loop"
```

## Task 3: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [x] **Step 1: Final full verification**

```bash
go test -race ./...
make generate
make validate
```

- [x] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a work-log entry ("Formation combat-loop wiring: 11b reassignment-on-death
(complete — Phase 11's foundational combat wiring is now fully done,
<date>)"). Update `## Current position`/`**HEAD:**`/"Next" to reflect that
all of 11a/11b/11c's shipped acceptance criteria are now live in combat,
and that further formation-combat work is genuinely new scope (11d and
beyond) rather than a follow-up to 11a-11c. Explicitly still-deferred:
the "engagement trigger" half of 11b (an idle companion proactively
joining because the leader started a fight with no prior hit landing on
that companion — today's reactive "companion retaliates once it's hit"
loops, replicated by the interception passes, remain the only trigger),
the leader-as-interceptor gap from the mob-vs-mob pass, and everything
11a-11c themselves already deferred (11d, guard-stance interception, AoE,
formation buffs, flanking, movement-in-combat, PvP/`internal/parties`).

- [x] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record 11b reassignment-on-death combat wiring"
```

---

## Self-Review Notes

- **Spec coverage:** "a company member whose target dies gets reassigned
  within that party, if one is still legal" ✓ (`reassignEnemyTarget` +
  the two engine wrappers, wired into both relevant dead-target checks).
  "No legal target: does nothing, doesn't panic or loop" ✓ — inherited
  directly from `engagement.AssignTarget`'s own already-tested contract;
  `reassignEnemyTarget` adds no new failure modes on top of it. The
  "engagement trigger" (idle companion proactively joining) is explicitly
  NOT this task — 11b's spec separates "target assignment"/"re-assignment
  on target loss" (this plan) from "engagement trigger" (already partly
  covered by the pre-existing charmed-retaliation loops the interception
  passes replicated, not by this plan) — both are real 11b acceptance
  criteria, this plan closes the reassignment one specifically, named as
  such in its title.
- **Placeholder scan:** no TBD/TODO, every step has real code.
- **Type consistency:** `reassignEnemyTarget`, `partyCombatants`,
  `firstHostilePartyInRoom`, `reassignPlayerTarget`,
  `reassignCompanionTarget` match between Task 1's definitions and Task
  2's four call sites.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11-reassignment-on-death.md`.**
