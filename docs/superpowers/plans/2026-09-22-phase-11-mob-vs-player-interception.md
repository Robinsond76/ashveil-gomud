# Phase 11 Mob-vs-Player Interception Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Status: fully implemented and committed on `phase-11-mob-vs-player-interception` — every step below is checked off.**

**Goal:** Complete the second of the three combat directions deferred by
the player-vs-mob wiring pass: when a hostile mob attacks a player leader,
11c's front-row interception now redirects the hit to a living, legal
front-row companion instead — "my tank actually protects me" becomes real.
Column-occupancy/lateral-range legality also gates the attack (skip, don't
clear `Aggro`, if illegal) exactly as it already does for player-vs-mob.

**This is not a numbered phase** — same status as the player-vs-mob pass:
the join between already-designed pieces (11c's `Legal`/`InterceptFrontRow`,
already shipped and tested), not new gameplay rules. No separate spec doc;
this plan's "Design decisions" section covers the new decisions.

## Design decisions

**1. A redirect here means a different `combat.Attack*` function, not just
a different target.** Player-vs-mob redirection stays within
`combat.AttackPlayerVsMob` (only the mob argument changes). Here, the
un-redirected attack is `combat.AttackMobVsPlayer(mob, defUser)`; a
redirect to a companion must instead call `combat.AttackMobVsMob(mob,
interceptor)` — an entirely different resolution path with different
message/equipment-break/vitals side effects (a mob defender has no
`SendText`, uses room-broadcast messages and `onHurt` script events
instead, the same shape the existing "mob attacks mob" call site at
`NewRound_DoCombat.go:1020` already uses). This plan's gate therefore
**resolves the intercepted attack itself** (in
`resolveInterceptedMobAttack`, mirroring that existing mob-vs-mob
post-attack block) and tells the caller "already handled, skip your own
attack resolution and `continue`" rather than handing back a substitute
target the way the player-vs-mob gate does.

**2. `Aggro` never retargets to the interceptor.** After an intercepted
round, `mob.Character.Aggro` stays exactly as it was — pointed at
`defUser.UserId`. Interception is recomputed fresh every round from
current formation/alive state (11c's self-healing model); it is not a
persistent retarget. If the interceptor later dies, the very next round's
gate call finds no living front-row companion, `InterceptFrontRow` no
longer applies, and the attack resolves directly against the leader again
automatically — with zero Aggro bookkeeping needed on anyone's part. This
mirrors exactly what the player-vs-mob pass already does when a blocking
enemy dies.

**3. The existing "leader is attacked → idle companions retaliate" loop
also fires on an intercepted round.** `NewRound_DoCombat.go:880-892`
already loops every one of the leader's idle charmed mobs and sets them
attacking the mob whenever the leader takes a hit — this is today's
(pre-11) coarse version of 11b's "engagement trigger." `resolveInterceptedMobAttack`
replicates that identical loop, so a leader being defended by an
interceptor still rallies the rest of an idle company exactly as it would
if the hit had landed on the leader directly — no narrower or different
behavior just because the hit was absorbed.

**4. A new `company.InstanceFor` seam method, and one new exported
`company.CompanionIDFromMemberKey` parser.** `InterceptFrontRow` returns a
`company.MemberKey` (e.g. `"companion:3"`); resolving that into a live
`*mobs.Mob` to actually attack needs two things that don't exist outside
`modules/company` yet: parsing the companion ID out of the key format
(mirrors 11a/`mobparty`'s `InstanceIdFromMemberKey`, added last pass for
the exact same reason on the mob side), and looking up that companion's
current live mob instance ID (mirrors `modules/company`'s existing private
`instance()` method — reused via one new thin exported wrapper, not
duplicated). Both are added to the existing `FormationProvider`
seam/interface from the previous pass rather than inventing a second seam,
since both are "read live company/formation state" queries served by the
same implementer.

**5. `mob.Reach` (11c's innate-reach flag) is passed through for hostile
attackers.** The player-vs-mob gate always passed `innateReach=false` to
`combat.ResolveReach` (players have no innate reach source, only a
weapon). A hostile mob attacker can have `Reach: true` in its own spec
(11c's schema addition, shipped but previously unused by any real call
site) — this pass is the first to actually read it, via
`combat.ResolveReach(&mob.Character, mob.Reach)`.

## Global Constraints

- Never advances `gametime`, round count, or damage math.
- `Aggro` is never mutated by the new gate logic (see Design Decision 2) —
  a blocked round is skipped, an intercepted round resolves against a
  different mob but leaves `mob.Character.Aggro` untouched.
- Fails open exactly like the player-vs-mob pass: no company record for
  the defending leader, or the attacking mob can't be resolved into an
  assembled enemy party, means unchanged pre-existing behavior
  (`combat.AttackMobVsPlayer(mob, defUser)` runs exactly as it does today).
- Formation is read-only — nothing here mutates a `Formation` or a room's
  mob list.

---

## File Structure

- Modify: `internal/company/formation.go` — add
  `CompanionIDFromMemberKey`.
- Modify: `internal/company/formation_test.go` — round-trip test for the
  new parser.
- Modify: `internal/company/provider.go` — extend `FormationProvider` with
  `InstanceFor`; add the package-level `InstanceFor` call-through.
- Modify: `internal/company/provider_test.go` — call-through test for
  `InstanceFor`.
- Modify: `modules/company/company.go` — add one thin `InstanceFor` method
  delegating to the existing private `instance()`.
- Modify: `internal/hooks/combat_formation.go` — add
  `resolveHostileAttackerColumn`, `aliveMapForCompany`,
  `gateMobVsPlayerAttack`, `resolveInterceptedMobAttack`.
- Modify: `internal/hooks/NewRound_DoCombat.go` — insert the gate call at
  the mob-vs-player attack site.

## Task 1: `CompanionIDFromMemberKey` and the `InstanceFor` seam method

**Files:**
- Modify: `internal/company/formation.go`
- Modify: `internal/company/formation_test.go`
- Modify: `internal/company/provider.go`
- Modify: `internal/company/provider_test.go`
- Modify: `modules/company/company.go`

**Interfaces:**
- Produces: `func CompanionIDFromMemberKey(key MemberKey) (int, bool)`,
  extended `FormationProvider.InstanceFor(leaderUserID, companionID int) (int, bool)`,
  `func InstanceFor(leaderUserID, companionID int) (int, bool)` (package-level
  call-through), `func (m *CompanyModule) InstanceFor(leaderUserID, companionID int) (int, bool)`.

- [x] **Step 1: Write the failing tests**

Add to `internal/company/formation_test.go`:

```go
func TestCompanionIDFromMemberKeyRoundTrip(t *testing.T) {
	id, ok := company.CompanionIDFromMemberKey(company.CompanionMemberKey(7))
	require.True(t, ok)
	assert.Equal(t, 7, id)
}

func TestCompanionIDFromMemberKeyRejectsNonCompanionKeys(t *testing.T) {
	_, ok := company.CompanionIDFromMemberKey(company.LeaderMemberKey)
	assert.False(t, ok)

	_, ok = company.CompanionIDFromMemberKey(company.MemberKey("mob:9"))
	assert.False(t, ok)
}
```

Add to `internal/company/provider_test.go`:

```go
type fakeInstanceLookup struct {
	fakeFormationProvider
	instanceId int
	ok         bool
}

func (f fakeInstanceLookup) InstanceFor(leaderUserID, companionID int) (int, bool) {
	return f.instanceId, f.ok
}

func TestInstanceForReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, ok := company.InstanceFor(1, 2)
	assert.False(t, ok)
}

func TestInstanceForCallsThroughToRegisteredProvider(t *testing.T) {
	company.SetFormationProvider(fakeInstanceLookup{instanceId: 55, ok: true})
	defer company.SetFormationProvider(nil)

	got, ok := company.InstanceFor(1, 2)
	require.True(t, ok)
	assert.Equal(t, 55, got)
}
```

Note: `fakeFormationProvider` (from the previous pass) must now also
satisfy the extended `FormationProvider` interface for the file to compile
— give it a trivial `InstanceFor` method too (add directly to the existing
`fakeFormationProvider` type in `provider_test.go`):

```go
func (f fakeFormationProvider) InstanceFor(leaderUserID, companionID int) (int, bool) {
	return 0, false
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/company/... -run 'TestCompanionIDFromMemberKey|TestInstanceFor' -v`
Expected: FAIL — `undefined: company.CompanionIDFromMemberKey` /
`undefined: company.InstanceFor` / `fakeFormationProvider does not implement FormationProvider`.

- [x] **Step 3: Write the implementation**

In `internal/company/formation.go`, add `"strconv"` and `"strings"` to the
imports, and add near `CompanionMemberKey`:

```go
// CompanionIDFromMemberKey parses a companion's MemberKey (as produced by
// CompanionMemberKey) back into its companion ID. ok is false for any key
// not in that format, such as LeaderMemberKey or a mobparty mob key
// ("mob:<id>") — a caller resolving formation membership must not assume
// every non-leader key is a companion key.
func CompanionIDFromMemberKey(key MemberKey) (int, bool) {
	const prefix = "companion:"
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

In `internal/company/provider.go`, extend the interface and add the
call-through function:

```go
type FormationProvider interface {
	FormationFor(leaderUserID int) (Formation, bool)

	// InstanceFor returns the live mob instance ID currently attached to
	// leaderUserID's companionID, if the companion is currently spawned
	// and attached. ok is false otherwise (dismissed, never summoned, or
	// pending restoration).
	InstanceFor(leaderUserID, companionID int) (instanceId int, ok bool)
}

// InstanceFor calls through to the registered FormationProvider. See
// FormationFor for the no-provider-registered contract.
func InstanceFor(leaderUserID, companionID int) (int, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return 0, false
	}
	return p.InstanceFor(leaderUserID, companionID)
}
```

In `modules/company/company.go`, add near the existing `FormationFor`
method:

```go
// InstanceFor implements company.FormationProvider's second query: the
// live mob instance ID currently attached to a companion, delegating to
// the existing private lookup this module already maintains.
func (m *CompanyModule) InstanceFor(leaderUserID, companionID int) (int, bool) {
	return m.instance(leaderUserID, companionID)
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company/... ./modules/company/... -v`
Expected: PASS, all new tests plus every pre-existing test in both
packages.

- [x] **Step 5: `go vet`/`gofmt`, full build**

```bash
gofmt -l internal/company modules/company
go vet ./internal/company/... ./modules/company/...
go build ./...
```

- [x] **Step 6: Commit**

```bash
git add internal/company/formation.go internal/company/formation_test.go internal/company/provider.go internal/company/provider_test.go modules/company/company.go
git commit -m "feat(company): add InstanceFor seam and CompanionIDFromMemberKey"
```

## Task 2: The gate and intercepted-attack resolver

**Files:**
- Modify: `internal/hooks/combat_formation.go`

**Interfaces:**
- Consumes: `resolveAttackTarget` (already exists, previous pass),
  `resolveEnemyParty`/`hostileMobSummaries` (already exist), `mobparty.MemberKeyFor`
  (already exists), `company.CompanionIDFromMemberKey`/`InstanceFor` (Task
  1), `combat.ResolveReach` (11c), `combat.AttackMobVsMob` (existing).
- Produces: `func gateMobVsPlayerAttack(mob *mobs.Mob, defUser *users.UserRecord, mobRoom, defRoom *rooms.Room) (handled bool, ok bool)`,
  called from `NewRound_DoCombat.go` in Task 3.

- [x] **Step 1: Write the implementation**

This task adds only adapter code with real engine dependencies
(`mobs.GetInstance`, `rooms`, the `company`/`combat` seams) — it has no new
pure logic beyond what `resolveAttackTarget` (already tested, previous
pass) already covers, matching the precedent set by `gateFormationAttack`
itself (also untested directly, for the same reason — see that function's
own comment). No new test file for this task; Task 1's seam-level tests
and the existing `resolveAttackTarget` tests are what actually exercises
the decision logic this glues together.

Add to `internal/hooks/combat_formation.go`:

```go
// gateMobVsPlayerAttack decides whether mob's attack on defUser should be
// redirected to an intercepting companion (11c front-row interception) or
// skipped this round (11c legality), before the caller resolves the
// attack. Unlike gateFormationAttack (player-vs-mob), a redirect here
// means an entirely different combat.Attack* function — AttackMobVsMob
// against the interceptor, not AttackMobVsPlayer against defUser — so this
// function resolves the intercepted attack itself. handled=true tells the
// caller to skip its own attack resolution and continue the round.
// handled=false, ok=true means proceed exactly as before (attack defUser
// directly — no company, or no interception applies). ok=false means skip
// the round entirely; mob.Character.Aggro is left untouched in every case,
// so the engagement resumes automatically once whatever blocks it changes.
func gateMobVsPlayerAttack(mob *mobs.Mob, defUser *users.UserRecord, mobRoom, defRoom *rooms.Room) (handled bool, ok bool) {
	f, formationOk := company.FormationFor(defUser.UserId)
	if !formationOk {
		return false, true
	}

	attackerCol, attackerOk := resolveHostileAttackerColumn(mobRoom, mob.InstanceId)
	if !attackerOk {
		return false, true
	}

	alive := aliveMapForCompany(defUser, f)
	reach := combat.ResolveReach(&mob.Character, mob.Reach)

	finalKey, legalOk := resolveAttackTarget(attackerCol, f, company.LeaderMemberKey, alive, reach)
	if !legalOk {
		return false, false
	}
	if finalKey == company.LeaderMemberKey {
		return false, true
	}

	companionID, companionOk := company.CompanionIDFromMemberKey(finalKey)
	if !companionOk {
		return false, true
	}
	instanceId, instanceOk := company.InstanceFor(defUser.UserId, companionID)
	if !instanceOk {
		return false, true
	}
	interceptor := mobs.GetInstance(instanceId)
	if interceptor == nil {
		return false, true
	}

	resolveInterceptedMobAttack(mob, interceptor, mobRoom, defRoom, defUser.UserId)
	return true, true
}

// resolveHostileAttackerColumn returns a hostile mob's own column within
// its assembled enemy party (mirrors resolveEnemyParty, applied to the
// attacker instead of a target).
func resolveHostileAttackerColumn(room *rooms.Room, attackerInstanceId int) (int, bool) {
	party, ok := resolveEnemyParty(room, attackerInstanceId)
	if !ok {
		return 0, false
	}
	_, col, found := party.Formation.Find(mobparty.MemberKeyFor(attackerInstanceId))
	if !found {
		return 0, false
	}
	return col, true
}

// aliveMapForCompany reports, for the leader and every formation-placed
// companion, whether they're currently alive (the leader) or currently
// spawned, attached, and alive (a companion).
func aliveMapForCompany(leader *users.UserRecord, f company.Formation) map[company.MemberKey]bool {
	alive := map[company.MemberKey]bool{
		company.LeaderMemberKey: leader.Character.Health > 0,
	}
	for row := 0; row < company.FormationRows; row++ {
		for col := 0; col < company.FormationCols; col++ {
			key := f.At(row, col)
			if key == "" || key == company.LeaderMemberKey {
				continue
			}
			companionID, ok := company.CompanionIDFromMemberKey(key)
			if !ok {
				continue
			}
			instanceId, ok := company.InstanceFor(leader.UserId, companionID)
			if !ok {
				alive[key] = false
				continue
			}
			mob := mobs.GetInstance(instanceId)
			alive[key] = mob != nil && mob.Character.Health > 0
		}
	}
	return alive
}

// resolveInterceptedMobAttack resolves one round of an attack 11c's
// front-row interception redirected from the leader to interceptor. It
// deliberately never touches mob.Character.Aggro (see this task's plan,
// Design Decision 2) — interception is recomputed fresh every round, not
// a persistent retarget. It mirrors NewRound_DoCombat.go's existing
// mob-vs-mob attack resolution (AttackMobVsMob, room-broadcast messages,
// onHurt scripting, offhand equipment-break) and its existing
// "leader is attacked" idle-companion retaliation loop, so an intercepted
// round behaves identically to a direct hit in every way except who takes
// the damage.
func resolveInterceptedMobAttack(mob, interceptor *mobs.Mob, mobRoom, defRoom *rooms.Room, defenderUserId int) {
	roundResult := combat.AttackMobVsMob(mob, interceptor)

	for _, instanceId := range mobRoom.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defenderUserId) && charmedMob.Character.Aggro == nil {
				charmedMob.Character.Aggro = &characters.Aggro{Type: characters.DefaultAttack}
				charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
			}
		}
	}

	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		interceptor.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		mobRoom.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg)
	}

	if !roundResult.Hit {
		return
	}

	scripting.TryMobScriptEvent(`onHurt`, interceptor.InstanceId, mob.InstanceId, `mob`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})

	if interceptor.Character.Equipment.Offhand.ItemId == 0 {
		return
	}

	modifier := 0
	if roundResult.Crit {
		modifier = int(interceptor.Character.Equipment.Offhand.GetSpec().BreakChance)
	}
	if !interceptor.Character.Equipment.Offhand.BreakTest(modifier) {
		return
	}

	defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, interceptor.Character.Equipment.Offhand.NameSimple(), interceptor.Character.Name))
	events.AddToQueue(events.ItemOwnership{MobInstanceId: interceptor.InstanceId, Item: interceptor.Character.Equipment.Offhand, Gained: false})
	interceptor.Character.RemoveFromBody(interceptor.Character.Equipment.Offhand)
	itm := items.New(20)
	if !interceptor.Character.StoreItem(itm) {
		defRoom.AddItem(itm, false)
		events.AddToQueue(events.ItemOwnership{MobInstanceId: interceptor.InstanceId, Item: itm, Gained: true})
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/characters"`,
`"github.com/GoMudEngine/GoMud/internal/events"`,
`"github.com/GoMudEngine/GoMud/internal/items"`,
`"github.com/GoMudEngine/GoMud/internal/scripting"`, and `"fmt"` to this
file's imports (all already used elsewhere in `internal/hooks`, so no new
dependency risk).

- [x] **Step 2: Build**

Run: `go build ./...`
Expected: clean build (confirms every symbol resolves and there's no
import cycle).

- [x] **Step 3: `go vet`/`gofmt`**

Run: `gofmt -l internal/hooks && go vet ./internal/hooks/...`
Expected: no output.

- [x] **Step 4: Commit**

```bash
git add internal/hooks/combat_formation.go
git commit -m "feat(hooks): add mob-vs-player interception resolution"
```

## Task 3: Wire the gate into `NewRound_DoCombat.go`

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go`

- [x] **Step 1: Insert the gate call**

In the mob-vs-player block, immediately after the `RoundsWaiting` branch's
`continue` and before the existing `var roundResult combat.AttackResult` /
`combat.AttackMobVsPlayer(mob, defUser)` call:

```go
			if handled, gateOk := gateMobVsPlayerAttack(mob, defUser, mobRoom, defRoom); !gateOk {
				continue
			} else if handled {
				continue
			}

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsPlayer(mob, defUser)
```

This is the only change to this file for this task. When `gateOk` is false
(illegal, no rescue), the round is skipped exactly like the existing
hidden-target/dead-target `continue`s already in this block — no message
is sent (a hostile mob doesn't narrate "you can't reach your target" the
way a player command does; it simply doesn't land a hit this round, same
as a mob who rolls badly). When `handled` is true, the intercepted attack
already fully resolved inside the gate call itself, so the round is done.
Every other line in this block — the weapon-pickup heuristic, the
`RoundsWaiting` branch, and everything from `combat.AttackMobVsPlayer`
onward — is untouched.

- [x] **Step 2: Build and run the full test suite**

```bash
go build ./...
go test -race ./...
```
Expected: clean build, full suite green (record actual counts).

- [x] **Step 3: `make generate`/`make validate`**

```bash
make generate
make validate
```

- [x] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "feat(hooks): gate mob-vs-player attacks on formation interception"
```

## Task 4: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [x] **Step 1: Final full verification**

```bash
go test -race ./...
make generate
make validate
```

- [x] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a work-log entry ("Formation combat-loop wiring: mob-vs-player
interception (complete; two directions still deferred, <date>)") following
the established format. Update `## Current position`/`**HEAD:**`/"Next" to
reflect that two directions remain (mob-vs-mob, 11b's reassignment-on-death)
and that mob-vs-player is now live.

- [x] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record mob-vs-player interception combat wiring"
```

---

## Self-Review Notes

- **Spec coverage:** front-row interception now applies to attacks aimed
  at the leader, redirecting to a living front-row companion ✓. Legality
  gating (skip, no Aggro change) for un-intercepted attacks on the leader
  ✓ (reuses the already-tested `resolveAttackTarget`). Self-healing (no
  Aggro retarget on interception, interceptor death automatically restores
  direct legality) ✓ — Design Decision 2. `Reach`'s innate-mob-flag source
  is exercised for the first time (`mob.Reach`) ✓. Mob-vs-mob and 11b
  reassignment-on-death remain explicitly out of scope, tracked as before.
- **Placeholder scan:** no TBD/TODO, every step has real code.
- **Type consistency:** `gateMobVsPlayerAttack`, `resolveHostileAttackerColumn`,
  `aliveMapForCompany`, `resolveInterceptedMobAttack`,
  `CompanionIDFromMemberKey`, `InstanceFor` match between their Task
  1/2 definitions and Task 3's one call site.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11-mob-vs-player-interception.md`.**
