# Blessings Module Guide

Phase 77. Design and decisions: `docs/plans/2026-10-07-phase-77-hardcore-blessings.md`. What a blessing is and does lives in `internal/blessings` (read its `AGENTS.md`); this module owns the saved list of what each account has earned, the `blessings` command and the `Char.Blessings` GMCP message.

- **State is per account (user id)** in the plugin store `blessings`. A purge that keeps the account (`UserPurged{KeepAccount}`, a deleted character) keeps it; only a purge that removes the account drops it. Listed in `purge_coverage_test` as account-level, so no `userstate` contributor.
- **Earned from the chronicle:** `chronicle.OnRecord` and `PlayerSpawn` run `evaluate`, which grants each blessing whose milestone the signed-in character's chronicle has reached and the account lacks. A grant is saved at once; a failed save keeps it in memory for the next save; nothing is saved while a load error is outstanding.
- **Given at creation, not earned live:** `usercommands.Start` (`giveBlessings`) applies the account's earned blessings to the new character once. This module never touches a character's items.
- `mu` is a leaf lock: nothing reaches the world while holding it.
- UI: `window-character.js` reads `Char.Blessings`; keep `panel` in `view.go` in step.
