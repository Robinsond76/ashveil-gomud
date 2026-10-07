# Story Events Module Guide

Phase 60. Design and decisions: `docs/plans/2026-10-07-phase-60-story-events.md`. Rules and data shapes live in `internal/storyevents`; this module owns saved state, triggers, commands, world effects and the `Event` GMCP message.

- **State is per company leader id** (`State{Pending, Done, Flags, Ops}`), saved in the plugin store `storyevents` and registered with `userstate` (the test area snapshots it; `purge_coverage_test` requires it).
- **Commit before apply** (`play.go`): the page advances or the event is marked done and saved, then outcomes run in `outcomeOrder`. Outcomes that can repeat after a crash use op ids `story:<event>:<ops>:<n>`. Never reorder to apply first.
- **World effects go through `world`** (`world.go`): `liveWorld` calls the owning modules' seams (wounds, survival, encumbrance cargo, company loyalty, `encounters.StartGroup`, `rooms.MoveToRoom`). Add a new outcome kind in `internal/storyevents` (validation and limits), then `play.go` and `world.go`, then a wiring test through a real trigger.
- **Triggers** (`trigger.go`): step and arrival listeners from `internal/walking`, and `camping.AddRestEndListener`. `Busy` holds a scene back during a fight, expedition or camp. A waiting page blocks `go` and `walkto` through `storyevents.MovementBlocked`.
- **Data**: embedded `events/*.yaml` plus the world's `events/` folder (disk wins by file name). A bad event is disabled with one warning. Keep shipped events test-only (the world is temporary); IDs of test rooms are 90011-90014.
- Time: real time only for cooldowns; never advance world time.
- UI: `window-event.js` reads GMCP `Event`; keep its payload in step with `view.go`'s `payload`.
- Tests: `storyevents_test.go` (fake `world`), `wiring_test.go` (real world through `go` steps). The live module tests run slowly under `-race`; keep fixtures small.
