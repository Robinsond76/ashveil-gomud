# Phase 34 session handoff

Owner instruction: commit and push each completed Phase 34 slice to master,
then hand over before starting the next slice so the owner can use a new
session. The design and concrete defaults are already approved; no new approval
is needed for the approved scope. Do not implement later phases in this session.

## Phase 34 complete — hand back to the owner

34a–34d are complete. Do not begin another slice in this session. The owner
chooses the next phase in a new session. Fetch master and read Project Status
and current nested guidance before any new work; retain existing worktrees.

34d adds Company.Conditions, an owned-member changed-only/reconnect feed for
live effects, recorded wounds, and persistent buffs with honest unavailable
state. Char.Capabilities adds current automatic class abilities and spells,
field/camp specialists and class-independent manual Camp Cooking. Char.Skills
keeps its array shape with real max_level, suppressing retired/orphaned ranks.
The views are read-only: no new persistence, readiness saves or world-time
changes. Player help, tutorial, safe text, narrow layout and keyboard/focus
checks ship with the slice.

Independent review resolved missing manual cooking and verified the follow-up.
Integration with shipped 33h3 (`f4e28dda`) passed independent review and all
final checks, including real passage/reload/rejoin conditions coverage.
Separated members show recorded lasting wounds, no saved temporary effects.
The cooking owner shares recipe selection with camp cook; the view consumes
nothing. A new away-member fixture was corrected to restore room bookkeeping
before teardown. See [34d verification](2026-10-02-phase-34d-verification.md)
for actual checks and integration results.

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
retraining and contract rewards, and 33h2 saved companion health/mana and
online recovery. Readiness vitals must ride existing snapshot seams; equipment
changes must not refill or independently save companion readiness.

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

34d: wounds remain separate from ordinary buffs. Light wounds last to fight
end; lasting wounds need treatment/rest. Combat effects retain combat-round
units; timed buffs use actual remaining seconds. Owned away live state is
labelled away; saved wounds are labelled recorded and no saved temporary buffs
are invented. Foreign charm state is unavailable. Capabilities use actual
skill/class configuration and strategy/autoskill switches; automatic spells
honour role, mana and attack reserve independently of NoAbilities. Cooking is
manual, class-independent, and derives recipes/ranks/ingredients/camp/capacity
from camping's existing selector.

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
