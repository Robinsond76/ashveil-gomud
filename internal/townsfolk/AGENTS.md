# Townsfolk Package Guide

Phase 68. GoMud-free: the `Line` shape and validation, `Catalog.Choose`, the `NPC`/`Context` inputs and the speaking seam (`Speak`, `Provider`). `modules/townsfolk` installs the provider, saves what each player was told and loads the line data; no other code imports the module. Decisions: `docs/plans/2026-10-07-phase-68-towns-that-remember.md`.

- A talker is a mob template with `townsfolk:` tags; `internal/hooks` `HandleIdleMobs` calls `Speak` on its idle turn. `Speak` is a no-op without a module, listeners or tags.
- `Choose` is pure: deeds newest first, once each (`Context.Heard`), a ref-specific line before a plain kind line, then state lines, then silence. Keep it free of the world so tests need no game.
- Add a condition: a field on `Line` with validation, a check in `conditionsMet` or `saidBy`, and a test. Add a placeholder in `knownPlaceholders` and `Fill`.
- A told deed is never repeated: the module saves it before the NPC speaks. Do not mark a deed told from a view.
- Real time only. Never use the world clock.
