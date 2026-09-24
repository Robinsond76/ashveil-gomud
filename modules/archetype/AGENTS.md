# Archetype Module Guide

Phase 17 archetypes and Phase 17b utility skills. The configured table (`files/data-overlays/config.yaml`) claims skills and spell schools; each player's one-time choice is in the durable registry. The engine reads the module only through `internal/archetypes` (`Provider`, plus the optional `TrapProvider`, `ClaimantNamer`, `TrapSenser`, and `Creator` interfaces). Engine code never imports this module.

Phase 22a starter kits and the creation step (`kit.go`):

- Each archetype's `Kit` lists item ids. Ids that don't resolve are dropped at load with a warning.
- A committed choice records `Registry.Kits[user]` in the same save as the choice, and rolls both back together if the save fails. Only choices made from Phase 22a on owe a kit.
- `grantKit` is all or nothing. It wears kit gear into empty slots only (never displacing gear), packs the rest, sets the character's `MiscData["archetype-kit"]` marker, then saves the user. The items and the marker live in the same user file, so they are always persisted together.
- The grant runs after a choice and again on every `PlayerSpawn`, and gives only when a kit is owed and the marker is absent. That is the exactly-once rule; don't add another grant path that skips the marker check.
- `start` (`internal/usercommands/start_archetype.go`) offers the archetypes through `archetypes.CreationChoices`/`ChooseAtCreation`. A failed commit never blocks creation.

Locking: `m.mu` guards the table, registry, and config. Never call the engine (users, items, skills) while holding it. Copy what you need, unlock, then act (`applyGrants`, `grantKit`, `list`, `CreationChoices`).

Tests: `testModule` loads the real skills, spells, buffs, and items from the shipped world. It stubs `saveUser` so tests never write user files into `_datafiles`. `wiring_creation_test.go` drives the real `start` prompt.
