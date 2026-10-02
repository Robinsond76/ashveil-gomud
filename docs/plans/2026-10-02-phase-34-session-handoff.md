# Phase 34 session handoff

Owner instruction: commit and push each completed Phase 34 slice to master,
then hand over context before starting the next slice so the owner can use a
new session. Do not continue into 34d after completing 34c without handing over.
The design and concrete defaults are already owner-approved; no new approval
is needed for the approved scope.

## Next session

Resume **34c — authoritative Character Gear editor**. 34a and 34b are complete,
reviewed, verified and pushed. 34d remains unstarted. Read root/nested guidance,
`docs/designs/2026-10-01-phase-34-company-ui-logistics-design.md`, and
`docs/plans/2026-10-01-phase-34-company-ui-logistics-plan.md`.

34c work was started before the owner asked for a session boundary. It is
uncommitted, unfinished, unreviewed work at
`/workspace/ashveil-gomud/.worktrees/phase-34c`, branch `phase-34c`, based on the
old local 34b merge `bad7cd0b`. Remote master subsequently incorporated 33h1
(companion growth/contracts), then the completed 34a/34b integration. Before
continuing, fetch master and preserve the work (`git stash push -u`), merge
origin/master into the feature branch, then restore the stash and resolve any
conflicts. Never discard this WIP or edit/commit directly on master. The old
phase worktrees are retained for reference; do not redo completed phases.

## Partial 34c implementation

Tracked edits:
- `internal/characters/character.go`: `Wear` accepts an optional target slot;
  explicit Weapon suppresses implicit offhand routing; explicit offhand weapon
  checks one-handed/dual-wield/disabled/bound/cursed constraints.
- `internal/combat/calculations.go`: exported `DodgeRetentionPct` uses the actual
  combat burden calculation, avoiding browser formulas.
- `modules/company/equipment.go`: extracted `equipmentProposal`, reused by
  commands and previews; exact reference plus optional fourth slot argument.
- `modules/gmcp/gmcp.Company.go`: added changed-only `Company.Equipment` extra.
- web `window-gear.js`: partial main-character slot editor, compatible cargo
  choices, current/after table, equip/remove commands, focus keys, stale-choice
  notice. Legacy Gear remains fallback. Shared mode hides old Cargo subtab.

New untracked files (include when preserving WIP):
- `internal/company/equipment_view.go`: provider/JSON data types.
- `modules/company/equipment_view.go`: owned leader proposal catalogue using
  existing clone/Wear rules and final load/capacity guard.
- `modules/gmcp/gmcp.CompanyEquipment.go`: feed adapter.

Focused `go test ./modules/company ./modules/gmcp ./internal/characters
./internal/combat` passed before pausing; JS lint passed before a trivial
`Client.SendInput` capitalization fix. These checks do not establish completion.
No new 34c integration tests, browser checks, help/tutorial updates, independent
review, or final full verification have been done.

Remaining 34c work:
1. Inspect the full WIP diff; verify read-only proposals never mutate live
   character/cargo/buffs, slot commands preserve legacy behavior, and previews
   match applied stats, hand conflicts, capacity and exact instances.
2. Check activity restrictions beyond battle, unavailable provider/save recovery,
   empty/cursed/disabled slots, stale selection, multiplayer ownership, two-hand
   displacement, overload recovery and journal failures/restart.
3. Review damage presentation (dice versus actual combat damage), burden/dodge
   label, current core stats and actual modifier changes. Reuse owner formulas.
4. Add real command and GMCP listener tests plus browser keyboard/focus/stale/
   narrow-layout checks. Verify `Company` snapshot replacement/reconnect resends
   the equipment extra and disabled reasons stay current.
5. Update equipment/equip/company-inventory/webclient help and tutorial pointers;
   add help rendering coverage.
6. Independent reviewer is required by repo workflow. Existing reviewer agent
   `/root/review_34a` performed 34a/34b reviews in this session; use an independent
   reviewer in the new session. Implement directly, no other delegation needed.
7. Final `make generate`, `make validate`, `go test -race ./...`, JS/Lua lint,
   browser checks; record review/results, commit, integrate latest master and
   push without force. Hand over before 34d.

## Environment

Repo `/workspace/ashveil-gomud`. Activate tools with:
`source /workspace/ashveil-env/activate.sh` (Go toolchain, readonly module cache,
GOMODCACHE/GOCACHE). Offline JS lint:
`make js-lint JSHINT=/workspace/ashveil-env/js/node_modules/.bin/jshint`.
`make lua-lint` works with local luacheck wrapper.

Playwright: `NODE_PATH=/opt/codex/runtimes/codex-primary-runtime/dependencies/node/node_modules`,
Chromium `/usr/bin/chromium`. Serve the correct worktree on localhost and set
`DOCK_HARNESS_URL=http://127.0.0.1:PORT/scripts/browser/dock-windows-harness.html`;
file URLs are blocked by browser policy. Local HTTP/browser and GitHub fetch/push
may require sandbox escalation; no force-push. Do not print credentials.

## Completed 34a/34b invariants

34a: solo leader fixed at 2,2; deterministic recruit vacancies; durable one-time
formation backfill preserving subsequent manual clears and existing placements;
failed saves roll back; shared Inventory hides worn/personal blocks; semantic
foreground and all shipped themes meet tested load/label contrast.

34b: one Pack slot; new kits/recruits receive cloth knapsack38 (10 kg); only
assigned packs of living present carriers plus eligible horses provide capacity;
unequipped cargo consumes it, worn gear does not; Pack has no combat weight,
stats/defense/worn buffs. Exact-instance old-pack migration uses existing asset
journal and durable once-only markers. Loss never deletes cargo; overload
recovery remains possible; ordinary weight-worsening operations reject final
excess. Inventory contains assigned pack availability cards and single cargo
list. Help and release notes document migration.

34a/34b original full gates and independent review passed. Integration with
33h1 was reviewed and checked again; preserve its growth state, retraining,
contract rewards, help and history. Review fixes include weapon-defense
retention (only Pack excluded), 10 kg creation kits, actual armor-removal guard
coverage and corrected inventory help. Formation combat test permits valid
retargeting after clearing an unreachable rear target.
