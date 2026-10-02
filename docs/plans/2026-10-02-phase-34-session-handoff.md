# Phase 34 session handoff

Owner instruction: commit and push each completed Phase 34 slice to master,
then hand over before starting the next slice so the owner can use a new
session. The design and concrete defaults are already approved; no new approval
is needed for the approved scope. Do not implement later phases in this session.

## Next session: 34d — Effects and current capabilities

34a, 34b and 34c are complete, independently reviewed, verified and integrated.
34d remains unstarted. Fetch current master and create a new isolated feature
worktree for 34d. Read root/nested guidance, `docs/PROJECT_STATUS.md`,
`docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`, the approved
[design](../designs/2026-10-01-phase-34-company-ui-logistics-design.md), and
[delivery plan](2026-10-01-phase-34-company-ui-logistics-plan.md).
Do not redo completed slices or discard retained worktrees.

34d delivers owned-member active effects and wounds in Company Status, showing
actual duration and mechanical meaning separately from persistent bonuses.
Represent away/dead/unavailable members honestly, without exposing another
company's private state. Extend Character Skills with trained ranks, automatic
combat abilities and field/camp capabilities derived from the shipped 33e/33f
eligibility, strategy toggles and retirement rules. Do not invent unlocks or
add a class catalogue, progression curve or companion equipment editor.

Useful starting points:
- `modules/gmcp/gmcp.Char.go`: existing Char.Affects and Char.Skills feeds;
  effect labels/descriptions and durations already have authoritative sources.
- `modules/gmcp/gmcp.Company.go`: changed-only extras and reconnect behavior;
  `modules/company/inventory_data.go`: owned-member presence/read-model pattern.
- `internal/company/state.go`, `internal/wounds/wounds.go` and
  `modules/company/wounds.go`: persisted wounds versus live character/mob buffs.
  Do not assume persisted wounds are represented in the ordinary buff list.
- `internal/strategy/abilities.go`: actual automatic ability specs and
  `NoAbilities` strategy toggle.
- `internal/archetypes/specialists.go`, `modules/archetype/specialists.go`,
  `modules/archetype/utility.go`, `modules/camping/camp_specialists.go`:
  actual field/camp eligibility and unlocks.
- `_datafiles/html/public/static/js/windows/window-company.js` Status and
  `window-character.js` Skills/Effects: existing browser presentation.

Test real entrypoints for effect apply/remove/expiry, wound duration, dead/away
members, private ownership and changed-only/reconnect updates. Cover class/rank
changes, retired skills, disabled automatic strategies and real field/camp
eligibility. Include responsive rendering, safe text, keyboard/focus, player
help rendering and tutorial pointer coverage. Obtain independent full-diff
review, resolve findings with meaningful regressions, then run required final
checks and record actual results before committing/integrating/pushing.
Hand over again when 34d completes.

## Completed invariants

34a: solo leader fixed at 2,2; deterministic recruitment vacancies; durable
one-time formation backfill preserves later manual clears and placements;
failed saves roll back. Shared Inventory shows cargo and semantic theme colors.

34b: one Pack per member; creation kits/recruits receive a 10 kg cloth knapsack.
Only packs on living present carriers plus eligible horses provide capacity.
Unequipped cargo consumes capacity, worn gear does not. Pack adds no combat
weight, stats, defense or worn buffs. Exact-instance migration uses durable
asset journals and once-only markers. Capacity loss preserves cargo and
recovery; ordinary changes cannot worsen excess. Inventory shows assigned
pack availability and one cargo list. Integration preserves 33h1 growth,
retraining and contract rewards.

34c: `Company.Equipment` is an owned-leader read-only authoritative proposal
catalogue. Character Gear uses exact-instance commands and explicit slots,
shows compatible choices/reasons and current/after stats, including both hands'
active sharpening bonus/strikes. Pack removal permits overload recovery;
ordinary changes enforce final capacity. Exact removal rejects replaced worn
instances. Existing journal/save/replay paths own mutation. Preview reads do
not mutate character, cargo or time. Legacy Gear remains available outside
company cargo mode; companion equipment remains command-managed. Focus and
stale choices survive updates safely. Independent review and follow-up passed,
as did real command/GMCP/recovery/help tests, all dock browser checks,
make generate/validate, JS/Lua lint and go test -race ./....

## Environment

Repo `/workspace/ashveil-gomud`. Activate tools with
`source /workspace/ashveil-env/activate.sh` (Go toolchain and readonly module
cache). JS lint: `make js-lint JSHINT=/workspace/ashveil-env/js/node_modules/.bin/jshint`.
`make lua-lint` uses the local luacheck wrapper.

Browser checks use `scripts/browser/dock-windows-check.mjs`,
`NODE_PATH=/opt/codex/runtimes/codex-primary-runtime/dependencies/node/node_modules`
and `CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium`. Serve the feature worktree on
localhost and set `DOCK_HARNESS_URL=http://127.0.0.1:PORT/scripts/browser/dock-windows-harness.html`;
file URLs are blocked. Local browser/network and GitHub fetch/push may require
sandbox escalation. Never force-push or print credentials.
