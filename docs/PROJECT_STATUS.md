# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-09-22
- **Branch:** `phase-6-interruptions`
- **HEAD:** `92edb585` (Phase 6 code and coverage; this status record follows)
- **Upstream baseline:** `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`
- **Origin sync:** `master` is 68 commits ahead of `origin/master`; Phase 3–5 and the durable-reservation correction are local only. Nothing pushed.

## Current position

- **Completed:** Phase 0–1 (fork, baseline, integration map), Phase 2 (company
  companion slice), Phase 3 (company roster + 3×3 formation), Phase 4
  (survival state), Phase 5 (terrain and travel profiles), and Phase 6 (travel
  interruptions).
- **Next:** Phase 7 — camping.

## Phase progress

| Phase | Scope | Status |
|---|---|---|
| 0 | Fork and bootstrap GoMud baseline | Complete |
| 1 | GoMud integration map and handoff docs | Complete |
| 2 | Minimal company slice (one persistent companion) | Complete |
| 3 | Company roster (cap 5) + 3×3 formation state | Complete |
| 4 | Survival state (hunger/thirst/fatigue) | Complete |
| 5 | Terrain and travel profiles | Complete |
| 6 | Travel interruptions | Complete |
| 7 | Camping | Not started |
| 8 | Weather | Not started |
| 9 | Encumbrance and cargo | Not started |
| 10 | Mounts | Not started |
| 11 | Formation combat | Not started |
| 12 | Rich expedition encounters | Not started |

## Recent work log

### Phase 6 — Travel interruptions (complete, 2026-09-22)

- **What:** Added one durable, configured `fallen-tree` interruption to the
  expedition domain and module. A profile can interrupt once at checkpoint 1–9;
  the Oak Road proves it at checkpoint 5. Active-time arithmetic freezes route
  progress, remaining duration, and exertion while paused. The module schedules
  the next route boundary, finalizes Phase 5 checkpoint exertion before
  persisting an interruption, and retains malformed records for operator repair.
  `travel status`, `look`, and movement refusal identify a paused obstruction;
  `travel resume` banks paused UTC time and resumes exactly the remaining active
  duration, while `travel return` durably records `Cancelled` before cleanup
  without moving or refunding the company.
- **Why:** This is the first safe, deterministic interruption point for real-time
  multiplayer travel. It remains durable across restart/copyover, preserves the
  Phase 5 operation-ID exactly-once survival protocol, and never advances global
  game time or round count.
- **Step completed:** Handoff Phase 6 ("Travel Interruptions").
- **Key commits:** `e25d2ec6`, `2a45f473`, `17091a31`, `a8255e51`, `646d00ab`,
  `a9c2c018`, and lifecycle/acceptance coverage through `92edb585`.
- **Verification:** `go test -race ./...`, `make generate`, `make validate`,
  plus focused domain/module and cross-package race suites pass. Task-scoped
  reviews covered malformed-record retention, parser/configuration, timer and
  checkpoint ordering, and command/recovery lifecycle behavior.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic fake scheduler, injected clock, persistence, recovery,
  command, view, and race suites cover the proving route behavior.
- **Deferred:** Combat interruptions, random event tables, rewards, en-route
  rooms, camping, weather, cargo, mounts, and formation effects remain outside
  Phase 6.

### Phase 5 — Terrain and travel profiles (complete, 2026-09-21)

- **What:** Added an optional `travel_profile` field to `exit.RoomExit` and a
  data-driven expedition system. `internal/expedition` owns the GoMud-free
  travel domain: validated `TravelProfile`s, the `Traveling`/`Interrupted`/
  `Completed`/`Cancelled` session state machine, clamped real-UTC progress,
  ten-checkpoint proportional exertion math, and mutex-protected start/view/
  movement provider seams. `modules/expedition` owns configured profile loading, durable
  leader-keyed sessions (YAML `ReadBytes`/`WriteStruct` like `modules/survival`),
  real-time `time.AfterFunc` completion timers, load/copyover recovery,
  destination move, exactly-once arrival, and the `travel status` command.
  Native `go` runs the same lock/exit-message/destination/script admission and
  then starts travel before action-point deduction for marked exits; native
  `look` renders the travel view at entry. `internal/survival` gained a
  `CompanyService` seam (`ApplyCompanyExertion`, `CompanyNeeds`) implemented by
  `modules/survival`, so travel accrues hunger/thirst/fatigue per company member
  without importing another module. Added the Dunmar proving route (room 2001
  Dunmar West Gate to 2002 Fork at the Black Oak) with the `oak-road` profile.
- **Why:** Travel is Ashveil's core expedition mechanic; it must be durable
  across disconnect/restart/copyover, exactly-once on arrival, and must never
  advance GoMud's global clock or round count.
- **Step completed:** Handoff Phase 5 ("Terrain and Travel Profiles").
- **Key commits:** `af6aa611` (domain), `7d11113c` (exit field + go/look
  adapters), `a14b086c` (durable module, profiles, survival seam, views),
  `1ac90a93` (timers, checkpoint sync, recovery), `983d04bd` (oak-road route),
  `cceb47e5` (refuse ordinary movement while travelling), and `af34e504`
  (merged durability corrections: pending operation IDs, survival replay ledger,
  checkpoint recovery, and player-spawn reconciliation).
- **Behavior:** An unmarked exit stays instant. A marked exit starts a durable
  session; the leader and company stay at the origin. While travelling, ordinary
  movement is refused with progress/remaining time, `look` shows
  origin/destination/profile/progress/remaining time/company needs, and
  `travel status` shows the same and syncs any earned checkpoint. Exertion is
  charged only at the ten durable progress checkpoints and telescopes to exactly
  the profile total at completion. On completion the module applies the final
  checkpoint, persists `Completed`, moves the leader once via `rooms.MoveToRoom`,
  commands native charmed companions to follow, announces arrival once, and
  removes the record after verifying the destination. Recovery at the
  destination cleans up, at the origin retries the move, and anywhere else is
  retained for operator repair. Checkpoint costs are prepared as durable,
  deterministic operations before survival changes; the survival ledger makes a
  restart replay idempotent, then expedition finalizes the checkpoint. A
  returning leader synchronizes and reconciles travel from `PlayerSpawn`, and
  `look`/`travel status` retry an overdue completion after a transient timer
  failure.
- **Verification:** `go test -race ./...` (1603 tests / 69 packages),
  `make generate`, `make validate`, and `go build ./...` pass. Focused race
  suites cover `internal/expedition`, `internal/exit`, `internal/usercommands`,
  `internal/survival`, `modules/survival`, and `modules/expedition`.
- **Copyover note (deviation from plan):** The plan asked for a registered
  `copyover.Contributor`, but `internal/copyover/AGENTS.md` forbids plugins from
  implementing it. Continuity instead relies on the plugin `SetOnSave`/
  `SetOnLoad` path (which runs immediately before/after a copyover): `onLoad`
  re-derives progress from the durable UTC start, reschedules a remaining
  session, or completes an overdue one. No copyover contributor is registered.
- **Durability correction:** Survival and expedition remain separate plugin
  writes, but the persisted pending-operation protocol makes their recovery
  idempotent: a crash between writes retries the same operation ID without a
  second survival cost.
- **Deferred:** Phase 6 owns interruption/pause/resume; Phase 7 camping/rest;
  Phase 8 weather; Phase 9 cargo; Phase 10 mounts. Needs remain informational in
  Phase 5 and never change duration, arrival, movement, combat, or health.
- **Live acceptance:** Not run (no interactive Telnet prerequisites); unit,
  module, command, persistence, timer, recovery, and race coverage only.

### Phase 4 — Survival state (complete, 2026-09-17)

- **What:** Added durable, per-company-member hunger, thirst, and fatigue in
  `0..100`, with five threshold bands, a pure `internal/survival` domain, and a
  `modules/survival` plugin that persists state under the module, projects the
  company roster, resolves leader/companion provisioning selectors, and renders
  a read-only `survival` command. `items.ItemSpec` gained optional `nutrition`
  and `hydration` metadata; native `eat`/`drink` accept an optional trailing
  member selector and provision the target from the leader's backpack before
  consuming the item. `modules/company` synchronizes summon/dismiss through a
  one-way `internal/survival` lifecycle seam and supplies companion names
  through a roster seam.
- **Why:** Survival state is the prerequisite for Phase 5 travel exertion and
  Phase 7 camp/rest recovery; it must persist across login/logout/copyover and
  never advance global game time.
- **Step completed:** Handoff Phase 4 ("Survival State").
- **Corrections (2026-09-17):** A review of `e5625faa..5f07d4f0` found state
  loss and inheritance bugs. Companions now carry a persisted
  `next_companion_id` high-water mark, so a dismissed ID is never reassigned
  across dismiss, dismiss all, or save/load, and a failed summon that has not
  yet committed survival state restores the exact prior record. The lifecycle
  seam captures exact `MemberSnapshot`s, so a failed company save restores the
  dismissed companion's real needs instead of a fresh default and joins
  compensation errors. `modules/company` reconciles
  every loaded roster into survival once after its registry loads, pruning
  orphaned companions and initializing current ones; a reconcile failure blocks
  company mutations. Consumable parsing again uses native backpack matching
  (partial and `name#n` numbered) and treats a trailing token as a target only
  when the survival module confirms it names a current member. Item specs reject
  negative `nutrition`/`hydration` while zero stays valid for legacy items.
- **Key commits:** `5872dadd` (domain), `1271e45d` (item metadata), `1cc7caeb`
  + `2a62de1b` (module persistence and roster seam), `609d0e4f` (eat/drink),
  `9055034d` (company lifecycle sync), `8f72807b` (roster-default status);
  corrections `08d33467` (non-reusable companion IDs), `7ed9c926` (exact
  snapshots and roster reconciliation), `a758a0c0` (rollback compensation),
  `470a2415` (consumable matching), and the metadata validation in this
  record's commit.
- **Behavior:** Needs change only through explicit APIs (`ConsumeFood`,
  `ConsumeWater`, `ApplyRestRecovery`, `ApplyExertion`). `eat <item> [member]`
  and `drink <item> [member]` accept `leader`/`me`/`self`, `#<id>`/`<id>`, or an
  unambiguous companion name; items are consumed only after survival mutation
  and persistence succeed. Successful summon creates default companion state;
  single/all dismissal prunes exactly the removed companions.
- **Verification:** `go test -race ./...` (1538 tests / 67 packages),
  `make generate`, `make validate`, and `make build` pass. Focused race suites
  cover `internal/company`, `internal/survival`, `internal/items`,
  `internal/usercommands`, `modules/survival`, and `modules/company`.
- **Known limitation:** Company and survival persist to separate plugin files,
  so summon/dismiss is not cross-file atomic. Rollback compensates with exact
  snapshots and surfaces joined errors, but a crash between the two writes can
  still leave the files divergent.
- **Deferred:** Camp/rest/sleep recovery is Phase 7; cargo, capacity, and
  automatic provisioning are Phase 9. No idle/offline drain and no health,
  combat, movement, or travel penalties in Phase 4.
- **Live acceptance:** Not run (no interactive Telnet prerequisites); unit,
  module, command, and race coverage only.

### Phase 4 final corrections (2026-09-17)

- **What:** Failed summon cleanup now treats a companion ID as spent once
  survival has durably recorded it. The transient companion is removed but the
  advanced `next_companion_id` high-water mark is retained and persisted, and
  the primary failure, any failed survival removal, and any failed high-water
  write are returned together. Provisioning selectors are now authorized only
  through the authoritative company roster: a stale survival record can no
  longer target a dismissed companion, and a current companion with no stored
  record is initialized to full needs when provisioned.
- **Why:** A review of `5f07d4f0..8eeaf973` found that failed summon cleanup
  could restore a spent ID and let a retry inherit stale survival state, and
  that numeric selectors trusted persisted survival state for authorization.
- **Key commits:** `65c6460b` (retain spent IDs after failed summon cleanup),
  `ac32c100` (authorize numeric targets from roster).
- **Verification:** `make generate`, `make validate`, and
  `go test -race ./...` (1544 tests / 67 packages) pass.
- **Known limitation:** Unchanged; company and survival remain separate plugin
  writes, so summon/dismiss is not cross-file atomic. Cleanup errors are joined
  and surfaced, and a spent companion ID is never silently reused.

### Phase 4 durable identity reservation (merged, 2026-09-17)

- **What:** Survival persistence now stores a per-leader
  `reserved_next_companion_ids` lower bound alongside companion needs.
  `EnsureCompanyMember` advances that reservation before persisting the new
  companion state. Before every summon, the company registry adopts the
  reservation as its minimum next ID.
- **Why:** This closes the final restart-safety hole: if survival records a
  companion but spawning fails, survival cleanup fails, and the company
  rollback write also fails, a fresh company registry still starts at `#2`
  rather than reusing `#1` and inheriting stale survival state.
- **Key commits:** `1d465b8f` (durable survival reservation and regression
  coverage), merged to `master` by `e9432182`.
- **Verification:** `make validate` and `go test -race ./...` pass after the
  composed serialized failure/restart regression was added. An independent
  review found no critical or important issues.
- **Known limitation:** The company and survival files remain separate direct
  writes, so they are not a transaction and a partial low-level write is still
  unrecoverable. The durable survival reservation prevents reuse of any ID
  whose survival initialization was successfully persisted; load reconciliation
  continues to repair ordinary roster/needs divergence.

### Phase 3 invariant corrections (2026-09-17)

- **What:** Hard-capped companies at four companions plus their leader, and normalized duplicate persisted formation occupants by keeping the first row-major cell.
- **Why:** Restores Phase 3’s five-character cap and one-cell-per-member invariants even when module configuration or stored YAML is invalid.
- **Verification:** `make generate`, `make validate`, and `go test -race ./...` passed.
- **Merge:** `main-deepseek` fast-forwarded into `master` at `3d6addd4`; `go test -race ./...` re-run on `master` passed (1434 tests / 65 packages).

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

- Phase 5 was implemented and committed directly on `master`. Going forward,
  plan and phase work must run on an isolated worktree/feature branch and merge
  back only after verification; see the root `AGENTS.md` ("Branching &
  Worktrees") and `docs/superpowers/plans/README.md`.
- `make test` stalls in the `js-lint` stage because it shells out to `npx
  jshint`; the documented fallback `go test -race ./...` passes. Environmental,
  not a code failure.
- Plugin `WriteStruct`/`WriteBytes` persistence is a direct (non-atomic) file
  write. Command state rolls back on failure, but a partial low-level write
  cannot be recovered.
- Live server acceptance has not been run for Phase 3, Phase 4, or Phase 5;
  unit/race tests cover the roster, formation, migration, survival persistence,
  provisioning, travel start/view/completion/recovery, and command behavior.
- Company and survival use separate plugin writes, so summon/dismiss is not
  cross-file atomic. Compensation failures are surfaced alongside the primary
  error, and the survival-side durable reservation prevents reuse of an ID once
  its survival initialization has persisted. A partial low-level file write is
  still not transactionally recoverable.
- Expedition survival exertion and its own checkpoint remain separate plugin
  writes, but Phase 5 now persists a deterministic pending operation before
  charging survival. The survival ledger deduplicates recovery, so a crash or
  failed checkpoint write cannot double-charge the company.
- Phase 5 copyover continuity relies on plugin `SetOnSave`/`SetOnLoad` rather
  than a `copyover.Contributor`, because modules are forbidden from registering
  copyover contributors. `onLoad` runs after `copyover.Restore` and reschedules
  or completes sessions from the durable record.
- Company/formation/survival state is process-local with no mutex, matching the
  existing event-loop dispatch assumption; revisit if command dispatch moves off
  the main loop.
- Deferred by design: recruitment economics, companion custom names, equipment,
  injuries, AI orders, death/permadeath rules, formation combat effects, travel
  interruptions (Phase 6), and camp/weather/cargo/mount integration (Phases
  7–10).

## Key documents

- `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` — authoritative migration direction.
- `docs/ASHVEIL_GOMUD_INTEGRATION.md` — Ashveil concept → GoMud source map.
- `docs/BASELINE_VERIFICATION.md` — vanilla baseline evidence.
- `docs/superpowers/plans/` — phased implementation plans.
- `docs/superpowers/specs/` — feature designs.
