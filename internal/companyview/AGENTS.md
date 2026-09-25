# Company View Guide

Phase 26a's read model: `For(user)` builds a `Summary` of a player and their company (needs, warmth, light, companions by member ID, load, activity, rest tier, checkpoint) from the providers that own each value. It stores nothing but the prompt cache. Design: `docs/superpowers/specs/2026-09-24-phase-26a-company-summary-text-design.md`.

- **Unknown is not healthy.** Every value has a `Known` flag (or an ok result); a missing provider leaves it false, and surfaces omit it. Never fill a default.
- **Game loop only.** `For` reads the company module (no mutex) and live mobs. Call it from commands, events, or `Refresh`.
- **The prompt reads a cache.** The engine builds prompts on connection goroutines too, outside the world lock. `Refresh`/`RefreshUser` (after every command, from `usercommands.TryCommand`) and the `NewRound` listener recompute each online player's token values on the game loop into a mutex-guarded map; `OnBuildPrompt` only reads it. Values lag by at most one round. Every token in `Tokens` renders "" when unknown, so the default prompt never shows a literal token.
- **Labels live here** (`labels.go`) so the text commands, the prompt, and the browser (Phase 26b) agree. Load bands have no configured names; the label follows the band's effect.
- **Buff groups.** Modules file their buffs for the grouped `conditions` panel with `RegisterBuffGroup` (camping: rest tiers, at load; exposure: band buffs, at init).
- Adding a value: add a seam in its owning `internal/` package (optional interface on an existing provider), a field with a `Known` flag, a label here, and tests with fake `sources`.
