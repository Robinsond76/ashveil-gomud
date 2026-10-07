# Chronicle Module Guide

Phase 63. Design and decisions: `docs/plans/2026-10-07-pillars-phases.md` ("Phase 63 build decisions"). Entry shapes, kinds, prose and the query seam are in `internal/chronicle` (read its `AGENTS.md`); this module owns the saved logs, the `chronicle` command, the `Company.Chronicle` GMCP message and the leader's own `PlayerDeath` deed.

- **State is per company leader id** (`chronicle.Log`), saved in the plugin store `chronicle`, registered with `userstate` (the test area snapshots it), dropped on `UserPurged` (`purge_coverage_test` enforces both).
- **A deed is never refused:** `Record` saves at once, and a failed save leaves it in memory for the next save. Saving is blocked while a load error is outstanding so an unreadable file is not overwritten.
- **`mu` is a leaf lock:** nothing reaches the world while holding it; `push` runs after the unlock.
- **Other modules record, they do not import this module.** Each deed is written from its real source (company roster changes, the death path, `Suicide`, the relic roll, the mercy answer, `class promote`, a story event's last answer, a defeat scenario). Add a hook beside the state change, never in a view, and give it a test through the real entry point.
- UI: `window-company.js` reads `Company.Chronicle`; keep its payload in step with `panel` in `view.go`.
- Time is real time; nothing here touches the world clock.
