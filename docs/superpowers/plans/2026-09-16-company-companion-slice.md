# Company Companion Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a persistent, one-companion `company` command slice that spawns an allow-listed mob, follows its leader through ordinary exits, and restores after login or copyover.

**Architecture:** `internal/company` owns the small, durable company model and its validation. `modules/company` owns GoMud integration: module-owned YAML persistence, commands, `PlayerSpawn` restoration, and a thin runtime adapter that applies the native charm/follow lifecycle. A durable template ID is saved; a transient mob instance ID is held only in memory.

**Tech Stack:** Go 1.24, Go standard `testing` plus the repository’s `testify` assertions, GoMud plugins/events/mobs/rooms/users, YAML module data.

**Spec:** `docs/superpowers/specs/2026-09-16-company-companion-slice-design.md`

## Global Constraints

- Use permanent game-domain names: `company`, `companion`, and `expedition`; do not add `ashveil*` package or type prefixes.
- Keep GoMud’s native `internal/parties` unchanged; it is not the durable company model.
- Store company data through module persistence under `_datafiles/plugin-data/`; do not add fields to `users.UserRecord` or persist mob instance IDs.
- Reuse GoMud charm tracking and ordinary-exit following. Do not add a second movement/follow framework.
- `PlayerSpawn` restores a missing live companion after normal login and copyover restoration.
- The first slice permits exactly one companion, uses an allow-list, and defers recruitment economics, formation, human members, equipment, injuries, and the five-member cap.
- Travel, rest, and this slice must never change global game time.
- Never modify or push to `upstream` (`GoMudEngine/GoMud`).

---

## File Structure

- `internal/company/company.go` — domain types, validation, and a leader-keyed registry; no engine globals or file I/O.
- `internal/company/company_test.go` — table-driven tests for durable company rules.
- `modules/company/company.go` — plugin registration, module persistence, command dispatch, and `PlayerSpawn` handling.
- `modules/company/runtime.go` — `Runtime` interface plus GoMud implementation for spawning, charm attachment, presence checks, and despawn.
- `modules/company/company_test.go` — fake-store/fake-runtime tests for lifecycle and command behavior.
- `modules/company/files/data-overlays/config.yaml` — default `AllowedCompanionMobIDs: [58]` development allow-list (the built-in training dummy).
- `modules/company/AGENTS.md` — local boundary and lifecycle guidance.
- `modules/all-modules.go` — generated blank import for `modules/company`; regenerate instead of hand-editing.

### Task 1: Add the durable company domain model

**Files:**
- Create: `internal/company/company.go`
- Test: `internal/company/company_test.go`

**Interfaces:**
- Produces `type Companion struct { MobTemplateID int \`yaml:"mob_template_id"\` }`.
- Produces `type Record struct { LeaderUserID int \`yaml:"leader_user_id"\`; Companion Companion \`yaml:"companion"\` }`.
- Produces `type Registry struct { Companies map[int]Record \`yaml:"companies"\` }`, `NewRegistry()`, `Get(leaderUserID int) (Record, bool)`, `Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}) error`, and `Dismiss(leaderUserID int) bool`.
- Produces sentinel errors `ErrInvalidLeader`, `ErrInvalidTemplate`, `ErrTemplateNotAllowed`, and `ErrCompanionAlreadyPresent`.

- [ ] **Step 1: Write the failing domain tests**

```go
func TestRegistrySummonStoresAllowedTemplate(t *testing.T) {
    registry := company.NewRegistry()
    err := registry.Summon(7, 58, map[int]struct{}{58: {}})
    require.NoError(t, err)
    got, ok := registry.Get(7)
    require.True(t, ok)
    assert.Equal(t, 58, got.Companion.MobTemplateID)
}

func TestRegistrySummonRejectsSecondCompanion(t *testing.T) {
    registry := company.NewRegistry()
    require.NoError(t, registry.Summon(7, 58, map[int]struct{}{58: {}}))
    assert.ErrorIs(t, registry.Summon(7, 58, map[int]struct{}{58: {}}), company.ErrCompanionAlreadyPresent)
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/company -run 'TestRegistry(Summon|Dismiss)' -count=1`

Expected: FAIL because `internal/company` and its registry API do not exist.

- [ ] **Step 3: Implement only the pure model and validation**

```go
func (r *Registry) Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}) error {
    if leaderUserID <= 0 { return ErrInvalidLeader }
    if mobTemplateID <= 0 { return ErrInvalidTemplate }
    if _, ok := allowed[mobTemplateID]; !ok { return ErrTemplateNotAllowed }
    if _, ok := r.Companies[leaderUserID]; ok { return ErrCompanionAlreadyPresent }
    r.Companies[leaderUserID] = Record{LeaderUserID: leaderUserID, Companion: Companion{MobTemplateID: mobTemplateID}}
    return nil
}
```

Initialize a nil `Companies` map before writing it. `Dismiss` deletes one leader’s record and returns whether it existed. Do not import `mobs`, `rooms`, `users`, or `plugins` into this package.

- [ ] **Step 4: Expand and pass the focused tests**

Add table cases for zero IDs, a disallowed template, a second summon preserving the original record, unknown/missing records, and idempotent dismissal. Run: `go test ./internal/company -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the domain deliverable**

```bash
git add internal/company/company.go internal/company/company_test.go
git commit -m "feat(company): add persistent companion model"
```

### Task 2: Add module persistence and lifecycle restoration

**Files:**
- Create: `modules/company/company.go`
- Create: `modules/company/runtime.go`
- Create: `modules/company/company_test.go`
- Create: `modules/company/files/data-overlays/config.yaml`
- Create: `modules/company/AGENTS.md`
- Modify (generated): `modules/all-modules.go`

**Interfaces:**
- Consumes `company.Registry`, `company.Record`, and its sentinel errors from Task 1.
- Defines `type Runtime interface { ResolveTemplate(string) (int, bool); Spawn(leaderUserID, roomID, mobTemplateID int) (int, error); IsLive(instanceID int) bool; Detach(leaderUserID, instanceID int) }`.
- Defines `type Store interface { Load(*company.Registry) error; Save(company.Registry) error }` and a private `pluginStore` adapter using `Plugin.ReadIntoStruct` and `Plugin.WriteStruct` with the identifier `companies`.
- Defines `CompanyModule` fields `plug *plugins.Plugin`, `store Store`, `registry company.Registry`, `liveByLeader map[int]int`, and `runtime Runtime`.
- Produces `restoreForLeader(leaderUserID, roomID int) error`, `save()`, and `load()`.

- [ ] **Step 1: Write failing lifecycle tests with a fake runtime and store**

```go
func TestRestoreForLeaderSpawnsFreshInstanceFromSavedTemplate(t *testing.T) {
    runtime := &fakeRuntime{nextInstanceID: 101}
    module := newTestModule(company.Registry{Companies: map[int]company.Record{
        7: {LeaderUserID: 7, Companion: company.Companion{MobTemplateID: 58}},
    }}, runtime)

    require.NoError(t, module.restoreForLeader(7, 12))
    assert.Equal(t, 58, runtime.spawnedTemplateID)
    assert.Equal(t, 101, module.liveByLeader[7])
}

func TestRestoreForLeaderKeepsRecordWhenTemplateCannotSpawn(t *testing.T) {
    runtime := &fakeRuntime{spawnErr: errors.New("unknown template")}
    module := newTestModule(company.Registry{Companies: map[int]company.Record{
        7: {LeaderUserID: 7, Companion: company.Companion{MobTemplateID: 58}},
    }}, runtime)
    assert.Error(t, module.restoreForLeader(7, 12))
    _, exists := module.registry.Get(7)
    assert.True(t, exists)
}
```

In the same test file, define `fakeRuntime` with `nextInstanceID`, `spawnedTemplateID`, `spawnErr`, `resolved map[string]int`, and `live map[int]bool`; its `Spawn` records the requested template and returns `nextInstanceID`. Define `fakeStore` with `saved company.Registry` and optional `loadErr`/`saveErr`. Define `newTestModule(registry company.Registry, runtime Runtime) *CompanyModule` to initialize the registry, `liveByLeader`, fake runtime, and fake store.

- [ ] **Step 2: Run the focused lifecycle tests to verify they fail**

Run: `go test ./modules/company -run 'TestRestoreForLeader' -count=1`

Expected: FAIL because the module, runtime interface, and restoration path do not exist.

- [ ] **Step 3: Implement the module and native runtime adapter**

Register `plugins.New("company", "1.0")`, attach the embedded filesystem, set `Callbacks.SetOnLoad(load)` and `Callbacks.SetOnSave(save)`, and listen for `events.PlayerSpawn` and `events.MobDeath`.

The GoMud `Runtime.Spawn` implementation must perform this sequence:

```go
leader := users.GetByUserId(leaderUserID)
if leader == nil { return 0, fmt.Errorf("company: leader %d is unavailable", leaderUserID) }
room := rooms.LoadRoom(roomID)
if room == nil { return 0, fmt.Errorf("company: room %d is unavailable", roomID) }
mob := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
if mob == nil { return 0, fmt.Errorf("company: mob template %d is unavailable", mobTemplateID) }
mob.Character.Charm(leaderUserID, -2, characters.CharmExpiredRevert)
leader.Character.TrackCharmed(mob.InstanceId, true)
room.AddMob(mob.InstanceId)
return mob.InstanceId, nil
```

`restoreForLeader` must return without spawning when there is no record or when `liveByLeader` points to a live instance. On a missing/invalid template, retain the registry record, clear any stale live mapping, and return an error for status reporting. The mob-death handler clears only the matching live mapping; it does not delete durable ownership. The `PlayerSpawn` handler obtains the current user room and invokes restoration. `pluginStore.Load`/`Save` use `plug.ReadIntoStruct("companies", &registry)` and `plug.WriteStruct("companies", registry)`; a missing data file initializes `company.NewRegistry()`. The module’s `load`/`save` delegate through `Store` so lifecycle tests use `fakeStore` without filesystem writes.

Add the default module overlay:

```yaml
AllowedCompanionMobIDs:
  - 58
```

Add `modules/company/AGENTS.md` stating that `MobTemplateID` is durable, `InstanceId` is runtime-only, `PlayerSpawn` handles normal login and copyover, and normal GoMud charm movement remains the only follower mechanism.

- [ ] **Step 4: Pass lifecycle tests and generate module wiring**

Add test cases for no saved record, no duplicate spawn when the tracked instance is live, stale-instance replacement, and mob death retaining the record. Run:

```bash
go test ./modules/company -run 'Test(RestoreForLeader|MobDeath)' -count=1
make generate
go test ./modules/company -count=1
```

Expected: PASS, and generated `modules/all-modules.go` contains `_ "github.com/GoMudEngine/GoMud/modules/company"`.

- [ ] **Step 5: Commit the lifecycle deliverable**

```bash
git add internal/company modules/company modules/all-modules.go
git commit -m "feat(company): restore persistent companions"
```

### Task 3: Add player commands and prove command behavior

**Files:**
- Modify: `modules/company/company.go`
- Modify: `modules/company/company_test.go`
- Modify: `modules/company/AGENTS.md`

**Interfaces:**
- Consumes `CompanyModule.restoreForLeader`, `CompanyModule.registry`, `CompanyModule.liveByLeader`, and `Runtime` from Task 2.
- Registers `company` as a non-admin user command through `plug.AddUserCommand("company", module.userCommand, false, false)`.
- Produces `summon(leaderUserID, roomID int, selector string) (string, error)`, `status(leaderUserID int) string`, `dismiss(leaderUserID int) (string, error)`, and the `company summon <mob-id-or-name>`, `company status`, and `company dismiss` command interface.

- [ ] **Step 1: Write failing command tests**

```go
func TestCompanySummonPersistsAndAttachesAllowedTemplate(t *testing.T) {
    module := newTestModule(company.NewRegistry(), &fakeRuntime{resolved: map[string]int{"training dummy": 58}})
    text, err := module.summon(7, 12, "training dummy")
    require.NoError(t, err)
    assert.Contains(t, text, "training dummy")
    record, saved := module.registry.Get(7)
    require.True(t, saved)
    assert.Equal(t, 58, record.Companion.MobTemplateID)
}

func TestCompanyDismissClearsSavedAndLiveState(t *testing.T) {
    module := newTestModule(company.Registry{Companies: map[int]company.Record{
        7: {LeaderUserID: 7, Companion: company.Companion{MobTemplateID: 58}},
    }}, &fakeRuntime{})
    module.liveByLeader[7] = 101
    _, err := module.dismiss(7)
    require.NoError(t, err)
    _, saved := module.registry.Get(7)
    assert.False(t, saved)
    assert.NotContains(t, module.liveByLeader, 7)
}
```

- [ ] **Step 2: Run command tests to verify they fail**

Run: `go test ./modules/company -run 'TestCompany(Summon|Status|Dismiss)' -count=1`

Expected: FAIL because the command service methods and parser do not exist.

- [ ] **Step 3: Implement command parsing, persistence, and clear output**

Parse with `util.SplitButRespectQuotes`. Resolve either a numeric template ID or a name through `Runtime.ResolveTemplate`; construct the allow-list with `allowedTemplateIDs(raw any) map[int]struct{}` that accepts both `[]int` and `[]interface{}` values from module configuration. Refuse unknown subcommands and missing arguments with usage text:

```go
func allowedTemplateIDs(raw any) map[int]struct{} {
    allowed := map[int]struct{}{}
    switch values := raw.(type) {
    case []int:
        for _, id := range values { allowed[id] = struct{}{} }
    case []interface{}:
        for _, value := range values {
            if id, ok := value.(int); ok { allowed[id] = struct{}{} }
        }
    }
    return allowed
}
```

```text
Usage: company summon <mob-id-or-name> | company status | company dismiss
```

On successful summon: validate through `registry.Summon`, spawn through `Runtime`, update `liveByLeader`, write module data immediately, and name the resolved mob in the confirmation. On a second summon, leave the existing record and live instance unchanged. `status` distinguishes `No companion`, `Companion: <name> (present)`, and `Companion: <name> (awaiting restoration)`. `dismiss` invokes `Runtime.Detach` when an instance is tracked, deletes the durable record even when no instance is live, writes module data immediately, and confirms dismissal.

The concrete `Detach` implementation must remove charm tracking from the leader, remove the instance from its current room, and call `mobs.DestroyInstance`; it must not call a player movement command or alter clock state.

- [ ] **Step 4: Pass command and module tests**

Add table cases for numeric/name resolution, unknown and disallowed templates, duplicate summon, status with stale state, and idempotent dismiss. Run:

```bash
go test ./modules/company -count=1
make validate
```

Expected: PASS.

- [ ] **Step 5: Commit the command deliverable**

```bash
git add modules/company
git commit -m "feat(company): add companion commands"
```

### Task 4: Perform end-to-end verification and update operator guidance

**Files:**
- Modify if verification exposes a gap: `modules/company/AGENTS.md`
- Modify if commands differ from the approved design: `docs/superpowers/specs/2026-09-16-company-companion-slice-design.md`

**Interfaces:**
- Consumes all preceding tasks and existing GoMud server commands.
- Produces evidence that the persisted template rehydrates as a fresh instance and native following occurs without time manipulation.

- [ ] **Step 1: Run the complete automated suite**

Run:

```bash
make generate
make validate
make test
```

Expected: PASS. If Docker Desktop is unavailable for Lua linting, record that exact blocker and still run `go test -race ./...`.

- [ ] **Step 2: Perform the local server acceptance sequence**

Run `make run`, log in with a disposable local user, and execute:

```text
company summon 58
company status
north
company status
company dismiss
company status
```

Expected: the training dummy appears, reaches the destination through the ordinary exit, disappears after dismiss, and status shows no companion. Restart or copyover with a saved companion and verify `company status` reports a new live instance for the same template. Confirm no command changes the displayed world time.

- [ ] **Step 3: Record verification outcome and commit only needed documentation corrections**

If behavior matches the spec, do not make a documentation-only churn commit. If operator steps or lifecycle behavior differ, update the closest relevant guide with the observed behavior, run `git diff --check`, and commit with `docs(company): clarify companion operation`.
