# Phase 29d: Pronouns and Stable Enemy Labels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. The owner now requests the current task’s default agent and reasoning settings for implementation; this supersedes the earlier Terra-medium selection.

**Goal:** Render correct NPC pronouns and enemy labels that stay fixed throughout a battle, including its fallen enemies.

**Architecture:** Character/template pronouns use a race fallback. The existing battle registry owns immutable enemy naming snapshots; combat, scripting, events, and summaries consume those snapshots without changing live names or combat identity.

**Tech Stack:** Go, existing YAML data, existing JS/Lua script actor bindings, Go testing/testify. No new dependencies.

**Spec:** [Phase 29d design](../specs/2026-09-26-combat-pronouns-ordinals-design.md).

**Status:** Owner approved; implementation in progress. Tasks 1–3 independently verified.

**Workspace:** `.worktrees/phase-29d-pronouns-ordinals`, branch `codex/phase-29d-pronouns-ordinals`, based on `b9729809`.

## Global constraints

- Never advance global game time. No new goroutines, timers, or database.
- Players remain they in third-person text and you in their own lines.
- Attack selection remains group-only under 32c.
- Do not change the live Character.Name.
- Never perform mob, room, company, or stream lookups while holding the battle mutex.
- Growth appends snapshots without changing any existing DisplayName.
- Naming snapshots are runtime only because battles, mob instances, and aggro themselves are runtime only.
- Use game-domain names; keep the Python prototype read-only.
- No player pronoun setting, creation step, battle dock, pain reactions, or tactical changes.
- Read root and owning nested AGENTS.md before edits. No baseline full suite; focused tests during work, full verification once after review fixes.
- Only the lead edits PROJECT_STATUS.md, commits, merges, pushes, or claims verification passed.

## Review focus

- Shared enemies with two player battles: both readers see identical labels; ending either battle preserves the other's labels (Task 2).
- A removed first enemy followed by growth: first stays reserved, old labels never change, and stream growth never restores base names (Tasks 2 and 4).
- Race change and generated identity: player polymorph stays they; named generated recruits never inherit an authored template's he/she (Tasks 1 and 5).
- Noun/case collisions and ordinals beyond ten: issued labels remain distinct and stable for new arrivals (Task 2).
- Less common output paths: separate-room shots, wait/aim, blocked flee, fizzle, interception, and quiet/fallback death notices agree with events (Tasks 3 and 4).

## Execution and review ownership

The owner approved the written design and plan, then requested on 2026-09-29
that implementation continue with the current task's default agent. This
supersedes the earlier Terra-medium selection. Execute the remaining tasks
directly using superpowers:executing-plans, retaining completed work and the
existing worktree. The lead owns review, verification, status, and commits.
A separate reviewer checks the full phase in Task 7 using inherited model
settings. Do not create a user-owned Codex task.

## Task 1: Pronoun model and authored defaults

**Files:**
- Create `internal/characters/pronouns.go`, `pronouns_test.go`.
- Modify `internal/characters/character.go`, `internal/races/races.go`.
- Create `internal/races/pronouns_test.go`, `internal/mobs/pronouns_test.go`.
- Modify the race YAML files enumerated in the spec and these NPC files:
  `_datafiles/world/default/mobs/dunmar/61-tamsin_reed.yaml`,
  `dunmar/62-brother_oswin.yaml`, `dunmar/63-garrick_vane.yaml`,
  `old_kings_road/64-ysolde.yaml`, `dunmar/65-sister_maren.yaml`,
  `fernhollow/66-old_wenna.yaml`, `tutorial/69-corvin_blackthorn.yaml`
  (abbreviated paths share the same mobs root).

**Interfaces:** `characters.PronounForms{Subject, Object, Possessive string}`;
`characters.PronounFormsFor(value string) PronounForms` (unknown returns they);
`(*Character).CombatPronouns() PronounForms`; `Character.Pronouns string`,
`Character.CombatNoun string`; `Race.DefaultPronouns string`.

- [x] Write table-driven tests `TestCombatPronouns` for all four sets,
  whitespace/case, empty/invalid values, unknown race, explicit override,
  effective race change, and no mutation during rendering. Pin she to
  `{Subject: "she", Object: "her", Possessive: "her"}` and it to
  `{Subject: "it", Object: "it", Possessive: "its"}`.
- [x] Add `TestPronounDataRoundTrip` using the existing YAML serializer:
  explicit pronouns/noun survive; old YAML with missing fields still loads
  and resolves through its race without requiring a save migration.
- [x] Add `TestRacePronounValidation` and `TestShippedPronounDefaults`:
  validate normalization/error behavior and load every specified shipped
  race/NPC through real loaders. `NewMobByIdNoElite` must retain the template
  pronouns. Reptilian (8) resolves they; reptile (21) resolves it.
- [x] Run `go test ./internal/characters ./internal/races ./internal/mobs -run 'Pronoun'`;
  confirm the new behavior fails before implementing it.
- [x] Add fields and resolver/default validation, then the exact shipped
  data from the spec. Do not guess genders for other authored mobs.
- [x] Run `go test ./internal/characters ./internal/races ./internal/mobs`;
  require exit 0. Lead inspects and commits as `feat(narration): add NPC pronouns and race defaults`.

## Task 2: Freeze enemy names inside battle ownership

**Files:** create `internal/battle/names.go`, `names_test.go`; modify
`internal/battle/battle.go`, `battle_test.go`.

**Interfaces:** `battle.EnemyName{InstanceId int; BaseName, Noun, DisplayName string}`;
`Battle.EnemyNames map[int]EnemyName`; `AssignEnemyNames(userId int, members []EnemyName)`;
`EnemyDisplayName(instanceId int, fallback string) string`.
Name inputs are collected outside the registry mutex. Keep battle's existing
Begin/Grow/End signatures. Name assignment contains no live game lookups.

- [x] Add `TestEnemyNamesFreezeAndGrow`: give IDs 42 and 41 the base name
  bandit cutthroat, in reversed input order. Assert ID 41 displays first
  cutthroat and 42 second cutthroat. Grow with only 42, then with 43:
  assert 42 remains second and 43 becomes third.
- [x] Add `TestEnemyNamesUniqueAndLateDuplicate`: a unique name remains
  bandit cutthroat; its later duplicate becomes second cutthroat without
  changing the original. Noun defaults strip articles and use the last
  word; an authored noun overrides that; blank names resolve creature.
- [x] Add `TestEnemyNamesCollisions`: case-fold duplicate cohorts; distinct
  bandit/goblin cutthroat cohorts use full base nouns; late noun collisions
  do not reuse existing labels; 11/12/13/21 render 11th/12th/13th/21st.
- [x] Add `TestEnemyNamesSharedBattles`: two users fight overlapping
  enemies in one room; both maps inherit fallen snapshots and receive
  growth additions. End/Forget one user and retain the other's labels.
  Different enemy IDs/groups stay independent.
- [x] Add `TestEnemyNamesLifecycleAndCopies`: mutate a Current result and
  prove the registry is unchanged; End/Forget/Retain/Reset release their
  state. Reset and begin a fresh battle with new IDs: old IDs resolve only
  their fallback and new IDs start fresh numbering.
- [x] Run `go test ./internal/battle -run 'EnemyNames'`; confirm failure.
- [x] Implement assignment, defensive cloning, overlap synchronization,
  growth collision handling, and lifecycle cleanup using the existing
  mutex. Retain original base names/nouns even if mobs disappear.
- [x] Run `go test -race ./internal/battle`; require exit 0. Lead reviews
  lock boundaries and commits as `feat(battle): freeze enemy narration labels`.

## Task 3: Weapon pronoun tokens and named character copies

**Files:** modify `internal/items/itemspec.go`, `internal/combat/combat.go`;
create `internal/combat/pronouns_test.go`, `enemy_names_test.go`; update
`internal/items/attack_messages_shipped_test.go`; modify the eight existing
world combat-message YAML files only where actor pronouns need tokens.

**Interfaces:** add `TokenSourceHe/Him/His`, `TokenTargetHe/Him/His` for
exact tokens in the spec. Add an internal
`mobCombatCharacter(m *mobs.Mob) characters.Character` helper in combat
that returns a local copy with Name replaced by EnemyDisplayName.
Use that copy in all mob sides of the attack wrappers; retain original
objects for health, edge spending, and damage attribution.

- [x] Add `TestCombatPronounTokens`: direct `buildCombatMessages` calls
  with fixed test templates exercise all six tokens in every together/
  separate recipient slot. Assert she/her/her, it/it/its, he/him/his,
  they/them/their; User actors always use they even with a beast form.
  Assert first-/second-person you/your and capitalization stay intact.
- [x] Add `TestAttackEntryPointsUseEnemyLabels`: execute player-to-mob,
  mob-to-player, and mob-to-mob entry points with tracked names. Assert
  rendered labels and unchanged original Character.Name/target IDs;
  include PvP/untracked fallbacks and damage/edge attribution assertions.
- [x] Add shipped pool coverage ensuring actor-owned possessives use
  tokens (including claws), new tokens resolve, voice rules hold, and
  literals such as “buries itself” are not rewritten as actor pronouns.
- [x] Run `go test ./internal/combat ./internal/items -run 'Pronoun|EnemyLabels'`;
  confirm failure.
- [x] Implement token resolution and local name copies, then replace
  actor-owned hard-coded pronouns in shipped pools. Preserve mechanics,
  seeds, pools, damage suffixes, tags, and named-subject verb grammar.
- [x] Run `go test ./internal/combat ./internal/items`; require exit 0.
  Lead commits as `feat(combat): render NPC pronouns and stable enemy names`.

## Task 4: Wire every combat surface and the event stream

**Files:** modify `internal/hooks/combat_battle.go`, `combat_stream.go`,
`combat_narration.go`, `combat_engagement.go`, `combat_formation.go`,
`NewRound_DoCombat.go`; modify `internal/scripting/actor_func.go`,
`internal/mobcommands/suicide.go`; sync the new script wrapper method in
`internal/scripting/objecttypes.go` and `internal/web/api_v1_scripting_dts.go`
(as required by scripting AGENTS.md); create focused pronoun/name tests in
hooks/scripting/mobcommands and `modules/company/wiring_pronouns_test.go`, `wiring_narration_outcome_test.go`,
  and its pre-phase JSON golden;
update `modules/company/wiring_narration_test.go`, `wiring_spells_test.go`
where exact old enemy names change. Modify combat spell JS files with
actor-owned hard-coded pronouns (particularly `spells/heal.js`).

**Interfaces:** before stream Open/Grow, hooks call AssignEnemyNames with
full plain mob inputs. Add `ScriptActor.GetCombatPronoun(form string) string`
(subject/object/possessive; unknown returns empty). GetCombatName looks up
mob labels; ordinary GetCharacterName stays unchanged. `mobRef` and gone
mob fallback refs use stored labels without changing Ref.Key.

- [x] Add `TestPronounsAndOrdinalsThroughRealRound` using `newBrawl` and
  real `hooks.DoCombat` with shipped config: two same-named enemies,
  ordered first/second names before their first hit, first's death, then
  second's later attack/target-change/death. Assert matching event refs,
  FightInfo enemy names and summary, and no live name mutation.
- [x] Add `TestEnemyLabelsOnSecondarySurfaces`: real wait/aim and a
  separate-room shot, blocked flee, fizzle, shield/interception, practice
  beaten, and suicide fallback. Each assertion goes through its actual
  entry point; assert one death/beaten line when quiet suicide follows.
- [x] Add `TestScriptCombatIdentity`: GetCombatName renders a label while
  GetCharacterName returns its base name. Execute script bindings in JS
  and Lua. Cast Minor Heal with authored she/he mobs; actual chanting and
  self-heal lines use her/his, while the user's lines use your/their.
- [x] Add `TestBattleNameGrowthReachesStream` for begin/growth ordering,
  shared player battles, and a removed enemy's retained ref name. No Grow
  call can overwrite labels with original names.
- [x] Add `TestNarrationPreservesCombatOutcome`: seeded before/after
  harness captured from base b9729809 and compared with the current code, resetting
  Go's existing math/rand after fixture setup (enable
  `GODEBUG=randseednop=0` for this test and call `rand.Seed(29)` after
  fixture setup). Isolate RNG draws from existing unordered mob turns with one
  attacking foe and a chanting twin; normalize independent per-round event order.
  Normalize fresh instance IDs to fixture roles; compare damage,
  crit flags, target IDs, mana, rewards, and outcome after stripping
  narration/Ref.Name. Keep kill XP controlled to avoid existing level-up
  refill behavior obscuring the comparison.
- [x] Run focused new tests in hooks, scripting, mobcommands, and company;
  confirm failures before wiring.
- [x] Implement begin/grow ordering and every listed rendering path.
  Wait calls pass local mob character copies with the stored label;
  fallback notices consult labels before battle teardown. Keep enemy name
  snapshots intact until death output and summary refs are captured.
- [x] Run `go test ./internal/hooks ./internal/scripting ./internal/mobcommands ./modules/company`;
  require exit 0; run `go test ./internal/web -run ObjectTypes` for editor
  method metadata synchronization; run `make js-lint` for changed JS if available and report
  any environmental block without claiming success. Lead reviews the
  full wiring diff and commits as `feat(narration): unify combat text and event identities`.

## Task 5: Generated recruit neutrality across restoration

**Files:** modify `modules/company/runtime.go`; update
`modules/company/wiring_roster_test.go`, `wiring_state_test.go`,
`resurrect_test.go`; create `modules/company/pronouns_test.go` if needed.

**Interfaces:** nativeRuntime.Spawn signature and durable Identity remain
unchanged. A nonempty identity.Name makes the spawned Character.Pronouns
explicitly they, regardless of its template's authored pronouns.

- [x] Add `TestGeneratedRecruitPronounsRestore`: hire a generated name
  from both an authored he and an authored she template; assert they
  live, serialize/reload the company, spawn again and assert they. Exercise
  the real restoration seam, not only a fake Runtime.
- [x] Add `TestAuthoredRecruitPronounsRestore`: empty saved identity keeps
  Tamsin's she/Brother Oswin's he before and after restore. Verify generated
  resurrection reapplies they and no old record needs a schema migration.
- [x] Run `go test ./modules/company -run 'RecruitPronouns|GeneratedRecruitPronouns'`;
  confirm failure.
- [x] Apply the explicit neutral override while applying saved generated
  identity; do not change company level/gear snapshot behavior or draw RNG.
- [x] Run `go test ./modules/company`; require exit 0. Lead commits as
  `fix(company): keep generated recruit pronouns independent of templates`.

## Task 6: Player help, aliases, and tutorial

**Files:** modify `_datafiles/world/default/templates/help/narration.template`,
`combat.template`, `targeting.template`, `battle-summary.template`,
`_datafiles/world/default/keywords.yaml`, `modules/tutorial/stages.go`,
`internal/usercommands/help_combat_test.go`, `modules/tutorial/shipped_test.go`, `wiring_test.go` (existing combat transcript assertions).

**Interfaces:** keep narration under combat; pronouns and ordinals are
help-aliases to narration. Combat tutorial hints link help narration.

- [ ] Extend real help rendering tests to assert labels stay fixed after
  a death, aliases match narration, and all four pages render. Assert help
  continues to require attacking groups and contains no set pronouns
  command. Pin the tutorial hint to first/second footman.
- [ ] Run `go test ./internal/usercommands ./modules/tutorial -run 'CombatHelp|TutorialHelpPointers|Narration'`;
  confirm the new content assertions fail.
- [ ] Write the player-facing explanation/examples from the spec; retain
  existing combat mechanics and add the fresh-battle restart explanation.
- [ ] Run `go test ./internal/usercommands ./modules/tutorial`; require
  exit 0. Lead commits as `docs(help): explain combat pronouns and enemy labels`.

## Task 7: Independent review, final verification, and integration

**Files:** lead-only plan checkboxes and `docs/PROJECT_STATUS.md`; any
confirmed review fixes and regression tests in the owning packages.

- [ ] Lead inspects the complete phase diff and cross-checks all ten spec
  acceptance criteria against actual tests. Check nested instructions,
  stable labels/lock order, missing output paths, and help scope.
- [ ] Dispatch an independent reviewer over `git diff b9729809..HEAD`
  with the spec/plan and phase invariants. Request findings only: bugs,
  design gaps, missing integration tests, and inaccurate/missing help.
  Use an independent reviewer with the task’s default model settings.
  The owner’s default-agent instruction supersedes earlier model overrides.
- [ ] Verify every finding, fix real issues with focused regression tests,
  and record rejected findings with reasons. Finish all code corrections
  before the single full verification run.
- [ ] Run `make generate`, then `make validate`, then `go test -race ./...`.
  Require exit 0 for each; inspect generated diffs. If generated code
  changes affect behavior, resolve that before claiming completion.
  Re-run full checks only if code changed after the green run.
- [ ] Record review outcome and exact verification in PROJECT_STATUS.md;
  mark implementation checkboxes only for work actually done. Include the
  intentional fresh-battle restart contract and deferred player selection.
- [ ] Lead commits the verified status/plan and any final fixes. Prepare
  a reviewable branch or PR for integration; do not merge/push unreviewed
  work or commit on master. Attach every created PR to the current task.

## Initial planning verification

The planning change is Markdown only. The lead checks diff whitespace,
relative links, acceptance-to-task coverage, and scope/approval consistency.
No Go checks are required until implementation changes shipped content or
code. All implementation verification boxes above remain unchecked.
