# Phase 17: Archetypes and Utility Skills — implementation plan

See `docs/superpowers/specs/2026-09-23-phase-17-archetypes-utility-skills-design.md`.
**Not started.** The spec's open decisions 1–8 must be confirmed, or
covered by the user's standing "proceed with your recommendation", and
recorded in the spec before Task 1.

It ships in two slices. **17a (Tasks 1–6)** is archetypes and gating.
**17b (Tasks 7–12)** is utility skills. Each slice is verified and
reviewed on its own, per the testing and review gate. Cooking is deferred
(decision 7).

**Delegation:** Tasks 1 and 7 are pure, single-package, and clearly
specified, so they may go to the cheap implementer tier with a narrow
brief. Everything else touches persisted state, engine command paths,
script entry points, or cross-module event wiring, so the lead implements
it directly (the `CLAUDE.md` escalation threshold).

Invariants for every task:

- Never advance the world clock or round count. There are no timers, and
  expiries are compared against the round count.
- Every new durable record survives restart and copyover. Grants are
  idempotent.
- With no provider registered, every seam allows everything (upstream
  behaviour).
- `modules/archetype` has a leaf mutex, and never calls another module
  or seam while holding it.
- Existing skills and spells are grandfathered: nothing already known is
  removed.

## 17a: Archetypes

- [ ] **0. Baseline.** On a fresh branch or worktree from `master`:
  - run `go test -race ./...`, `make generate`, and `make validate`
  - record decisions 1–8 in the spec
  - fix the `illlusion` school typo, and confirm by grep that no code
    depends on the misspelling
- [ ] **1. `internal/archetypes` (pure, delegable).** Files:
      `internal/archetypes/archetypes.go`, `archetypes_test.go`,
      `provider.go`.
  - Tests first:
    - `TestArchetypeValidate`: empty id or name; a grant of an unlisted
      skill; a grant of a spell outside the claimed schools;
      `companionlevels` not ascending or not length 4
    - `TestClaims`: claimed, shared (`cast`), trade skill, claimed school
      versus open school
    - `TestCanTrain` / `TestCanLearnSpell`: chosen, wrong archetype,
      unchosen with a claimed skill, unchosen with a trade skill, a
      grandfathered known spell, and no provider allows everything
    - `TestCompanionUtilityLevel`: the level table boundaries
  - Then implement: the `Archetype` type, `LoadDataFiles` with the
    skills-style cross-reference warnings, `SetTestData`, the claim index,
    and the `Provider` seam (`ArchetypeOf(userID)`) with `CanTrain`,
    `CanLearnSpell`, and `SetProvider`.
- [ ] **2. Data.** Add `_datafiles/world/default/archetypes/` with
      warrior, rogue, wizard, cleric, and ranger, per the spec table.
  - Test: `TestShippedArchetypesLoad` loads the real skill, spell, and
    archetype files, with no warnings and every claim resolved.
- [ ] **3. `modules/archetype`: registry, choice, and grants.** Files:
      `modules/archetype/archetype.go`, `archetype_test.go`,
      `files/data-overlays/config.yaml`.
  - Tests first, with a fake store and a fake user:
    - `archetype` lists the table
    - `choose` without confirm only previews
    - `choose ... confirm` persists and applies grants
    - a second choose is refused
    - a failed save leaves no choice and no grants
    - grants never lower a higher skill level
    - on reload with the grant missing on the user (a simulated crash),
      the grant is re-applied on `PlayerSpawn`
    - admin reset
    - toggles persist
  - Then implement, and register the provider. Run `make generate` and
    check that `modules/archetype` is the only addition.
- [ ] **4. Companion archetypes.** Files: `modules/archetype`,
      `modules/company` (the recruit hook and `company archetype`), and
      the company config.
  - Tests first:
    - a recruit of template 58 records warrior from `CompanionArchetypes`
    - `company archetype <member> <name>` sets an unset archetype once and
      refuses a second time
    - a member leaving the roster prunes the entry
    - legacy companions have none
  - Then implement, reusing the survival/exposure/walking reconciliation
    pattern.
- [ ] **5. Engine gating seams.** Files: `internal/usercommands/train.go`,
      `internal/scripting/actor_func.go`,
      `internal/scripting/party_func.go`.
  - Tests first, as wiring tests through real entry points:
    - `usercommands.Train` in a real room with `SkillTraining`:
      - an unchosen player is refused a claimed skill and allowed a trade
        skill
      - a wizard is allowed `cast`
      - a warrior is refused `cast`
      - the panel text marks locked skills
      - training points are untouched on refusal
    - `ScriptActor.LearnSpell` for a wizard is refused `heal`
      (restoration) and allowed `illum`
    - party `LearnSpell` gates per member
    - `admin spell` bypasses the gate
  - Then add the seam calls.
- [ ] **6. Display, verify, and review 17a.**
  - Show the archetype in `company` status and on inspecting a player or
    companion, with a wiring test.
  - Run `go test -race ./...`, `make generate`, and `make validate`, and
    boot the server (no new warnings).
  - **Review gate for 17a:** a reviewer subagent over the 17a diff. Verify
    each finding, fix with regression tests, and record the result in
    `docs/PROJECT_STATUS.md` with a **Review:** line.

## 17b: Utility skills

- [ ] **7. Utility math (pure, delegable).** Add to `internal/archetypes`:
      `utility.go` and `utility_test.go`.
  - Tests first:
    - `TestEffectiveLevel`: a player with the skill and the utility; a
      player without the utility gets 0; a companion by level; an
      unspawned or absent companion gets 0
    - `TestBestMember`: the highest level wins, ties go to the leader
      then the lowest companion id, and nobody qualifying returns none
    - `TestSenseRoll` / `TestDisarmRoll`: the success boundary, and the
      backfire margin
    - `TestDisarmExpiry`
  - Then implement.
- [ ] **8. `autoskill` and the `trap` command.** Files:
      `modules/archetype/utility.go`, `utility_test.go`, and the module
      config (`SensePerLevel`, `SenseDifficultyFactor`, `SenseCooldown`,
      `DisarmPerLevel`, `DisarmDifficultyFactor`,
      `DisarmBackfireMargin`, `DisarmRounds`, `AutoSensePenalty`,
      `AutoLightCooldown`, `CompanionLightManaCost`).
  - Tests first, with a fake roster, rooms, and a seeded roll:
    - `autoskill` listing and toggling
    - `trap sense` success naming the locks, failure, and cooldown
    - `trap disarm` success persisting the expiry, plain failure, and
      backfire applying the lock's trap buffs
    - no member with the utility is refused
    - an expired disarm is pruned on load and on use
    - reload keeps an unexpired disarm
  - Then implement, including `archetypes.TrapArmed(lockId)` on the
    provider.
- [ ] **9. Picklock integration and auto-sense on entry.** Files:
      `internal/usercommands/picklock.go` and a `RoomChange` listener in
      `modules/archetype`.
  - Tests first, as wiring tests:
    - `usercommands.Picklock` on a real trapped chest springs the trap
      when armed and not after a disarm
    - it warns before the first pin when autoskill `traps` is on and the
      sense succeeds
    - an ordinary `Go` into the trapped room sends one passive warning
      when the roll succeeds, nothing with autoskill off, and nothing for
      a teleport (the listener acts only on a walking step)
  - Then implement. Decide during the task whether to share Phase 16's
    walking step seam or filter `RoomChange` by origin exit, and record
    the choice in the spec's implementation notes.
- [ ] **10. Wizard auto-light.** Files: `modules/archetype/light.go` and
      `light_test.go`, reading Phase 14's light seam.
  - Tests first, as wiring tests with real rooms at night or in a dark
    biome:
    - a player wizard walking into darkness casts `floatinglight` through
      the real cast path (mana spent, buff 1000 on)
    - a companion wizard gets buff 1000 and the mob's mana is spent
    - nothing happens when the room is lit, when a party light is already
      up, when autoskill `light` is off, in combat, on cooldown, or
      without enough mana
    - there is at most one attempt per step
  - Then implement.
- [ ] **11. Content.**
  - A trapped, locked chest in a Dunmar room (a new container with
    `difficulty` and `trapbuffids` of an existing mild debuff).
  - A rogue trainer range for `skulduggery`, and a wizard source for
    `cast` reachable in the Ashveil start area, if none is reachable
    today. Check first; add only what is missing.
  - Test: the shipped room loads with the trapped lock.
- [ ] **12. Concurrency, verify, review 17b, and record.**
  - A `-race` test (`-count=20`) running `archetype choose`,
    `trap disarm`, auto-sense and auto-light listeners, and a registry
    save at the same time.
  - Run `go test -race ./...`, `make generate`, and `make validate`, and
    boot the server.
  - If an interactive Telnet client is available, run live acceptance:
    choose wizard, walk into the dark and watch the auto-light; choose
    rogue on a second character, sense, disarm, and pick the chest. If
    not, record that it wasn't run.
  - **Review gate for 17b:** a reviewer subagent over the 17b diff.
    Verify each finding, fix with regression tests, and record the result.
  - Update `docs/PROJECT_STATUS.md`:
    - the Phase 17 row
    - a work-log entry with a **Review:** line
    - cooking deferred to Phase 18b
    - Phase 18 as next

    Then merge and push.
