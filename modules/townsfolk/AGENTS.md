# Townsfolk Module Guide

Phase 68. Design and decisions: `docs/plans/2026-10-07-phase-68-towns-that-remember.md`. Rules, line shapes and the speaking seam are in `internal/townsfolk` (read its `AGENTS.md`); this module owns what each player was told, the line catalog, the `townsfolk`/`renown` command and the `Company.Townsfolk` GMCP message.

- **State is per player** (`UserState{Heard, Told, Total}`; the company key is the leader's user id), saved in the plugin store `townsfolk`, registered with `userstate` (the test area snapshots it) and dropped on `UserPurged` (`purge_coverage_test` enforces both).
- **Save before speaking** (`told`, run by `Speech.Confirm` once the idle hook's line was let through): the deed's seq is stored, then the mark set and the view pushed. A new chronicle deed also pushes the view (`chronicle.OnRecord`). A failed save never silences the NPC, and a load error blocks saving so an unreadable file is not overwritten.
- **`mu` is a leaf lock:** chronicle queries, flags and the world are read outside it.
- **Lines** are `lines/*.yaml` (embedded, test content only; the world is temporary) plus `<DataFiles>/townsfolk/*.yaml`. A bad file or line is skipped with one warning.
- **World effects go through `world`** (`world.go`): company flags and member tags via `internal/storyevents`, weather via `internal/weather`, the clock via `internal/gametime` (read-only).
- UI: `window-company.js` reads `Company.Townsfolk` into the Chronicle tab; keep its payload in step with `panel` in `view.go`.
- Tests: `townsfolk_test.go` (fake world and `chronicle.Memory`, plus the real idle turn through `hooks.HandleIdleMobs`).
