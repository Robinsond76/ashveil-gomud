# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-09-17
- **Branch:** `main-deepseek` (Phase 3 feature merged; `master` unchanged)
- **HEAD:** `e674323a` (Phase 3 merged into `main-deepseek`)
- **Upstream baseline:** `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`
- **Origin sync:** `master` is 14 commits ahead of `origin/master`; `main-deepseek` adds the project-status doc and Phase 3. Nothing pushed.

## Current position

- **Completed:** Phase 0–1 (fork, baseline, integration map), Phase 2 (company
  companion slice), and Phase 3 (company roster + 3×3 formation).
- **Next:** Phase 4 — survival state (hunger/thirst/fatigue).

## Phase progress

| Phase | Scope | Status |
|---|---|---|
| 0 | Fork and bootstrap GoMud baseline | Complete |
| 1 | GoMud integration map and handoff docs | Complete |
| 2 | Minimal company slice (one persistent companion) | Complete |
| 3 | Company roster (cap 5) + 3×3 formation state | Complete |
| 4 | Survival state (hunger/thirst/fatigue) | Not started |
| 5 | Terrain and travel profiles | Not started |
| 6 | Travel interruptions | Not started |
| 7 | Camping | Not started |
| 8 | Weather | Not started |
| 9 | Encumbrance and cargo | Not started |
| 10 | Mounts | Not started |
| 11 | Formation combat | Not started |
| 12 | Rich expedition encounters | Not started |

## Recent work log

### Phase 3 — Company roster + 3×3 formation (complete, 2026-09-17)

- **What:** Expanded the company from one companion to a leader plus up to four
  companions (five-member cap) with stable companion IDs, and added a persistent,
  validated 3×3 tactical formation with `formation move|swap|clear` commands.
  `internal/company` owns the pure roster/formation model; `modules/company` owns
  multi-instance runtime tracking, legacy-record migration, persistence, and
  commands.
- **Why:** Formation is Ashveil's identity feature and needs a multi-member roster
  to be meaningful. Reused GoMud charm/follow, mobs, events, users, and module
  persistence; native `internal/parties` was left untouched.
- **Step completed:** Handoff Phase 3 ("3×3 Formation State") plus the deferred
  five-member company cap.
- **Key commits:** `8bf9d2ab` (formation grid), `159ec87b`+`e462c258` (roster and
  snapshot fix), `80cb28ba`+`3fb4eb43` (multi-instance runtime), `fc0e1987`+
  `3c963a1d` (legacy migration), `ed60995d`+`8d52b0c9`+`795a9222` (formation
  commands and rollback fixes), `39d993f2` (config + module docs).
- **Verification:** `go test ./internal/company ./modules/company` (72 tests),
  `make validate`, `make generate` (no wiring change), and `go test -race ./...`
  (1429 tests / 65 packages) pass. Legacy single-`companion` records migrate to
  `companions[0]` with ID 1; formation cells for dismissed companions are pruned.
- **Config:** `MaxCompanions: 4` in `modules/company/files/data-overlays/config.yaml`.

### Phase 2 — Company companion slice (complete, 2026-09-17)

- **What:** Added a durable, leader-keyed `company` model in `internal/company/`
  and a `modules/company/` plugin that persists the company record, spawns an
  allow-listed companion as a native GoMud mob, restores it after login/copyover,
  and exposes `company summon|status|dismiss`.
- **Why:** Prove a saved companion can be owned, followed, and restored without
  duplicating GoMud's party, mob, charm, movement, or persistence systems, and
  without an `ashveil*` layer.
- **Step completed:** Handoff Phase 2 ("Minimal Company Slice"). A durable
  `MobTemplateID` is stored; a runtime `InstanceId` is never persisted.
- **Key commits:** `8c1eca92` (model), `8be7c1f9` (restoration), `99f02261`
  (commands), `4dc1ea27` (attachment recovery + persistence-failure guards),
  `a4d71439`/`54305926` (wrap-up hygiene).
- **Verification:** `go test ./internal/company ./modules/company` (37 tests),
  `make validate`, and `go test -race ./...` (1394 tests / 65 packages) pass.
  Live Telnet acceptance passed (summon → native follow → dismiss → restart
  rehydration). `make test` is blocked locally by the `js-lint` stage (see
  Known issues).

### Phase 0–1 — Fork, baseline, integration map (complete, 2026-09-16)

- **What:** Forked GoMud into `Robinsond76/ashveil-gomud`, kept
  `GoMudEngine/GoMud` as read-only `upstream`, verified the vanilla baseline,
  and wrote the integration map, handoff, and phase plans.
- **Why:** Establish a clean, evidence-backed foundation before gameplay work.
- **Step completed:** Handoff Phases 0 and 1.
- **Key commits:** `1fba48a0` (BSD `awk` portability fix in `make help`),
  `6fd8c4b0` (baseline docs), `997d6db5`/`71c74f81`/`3cb4130c` (Phase 2 design
  and plan).
- **Verification:** Recorded in `docs/BASELINE_VERIFICATION.md`.

## Known issues / deferred items

- `make test` stalls in the `js-lint` stage because it shells out to `npx
  jshint`; the documented fallback `go test -race ./...` passes. Environmental,
  not a code failure.
- Plugin `WriteStruct`/`WriteBytes` persistence is a direct (non-atomic) file
  write. Command state rolls back on failure, but a partial low-level write
  cannot be recovered.
- Live server acceptance has not been run for Phase 3; unit/race tests cover the
  roster, formation, migration, and command behavior.
- Company/formation state is process-local with no mutex, matching the existing
  event-loop dispatch assumption; revisit if command dispatch moves off the main
  loop.
- Deferred by design: recruitment economics, companion custom names, equipment,
  injuries, AI orders, death/permadeath rules, formation combat effects, and
  travel/camp integration.

## Key documents

- `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` — authoritative migration direction.
- `docs/ASHVEIL_GOMUD_INTEGRATION.md` — Ashveil concept → GoMud source map.
- `docs/BASELINE_VERIFICATION.md` — vanilla baseline evidence.
- `docs/superpowers/plans/` — phased implementation plans.
- `docs/superpowers/specs/` — feature designs.
