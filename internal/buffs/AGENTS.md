# Buffs Guide

## Scope

- `internal/buffs` defines buff specs, active buff instances, flag helpers, and admin save/load helpers.
- Treat this package as the core status-effect engine used by spells, items, races, and room effects.

## Files

- `buffspec.go`: immutable buff definitions, validation, duration math, and data-file loading.
- `buffs.go`: runtime buff state, collection indexing, trigger flow, pruning, and stat aggregation.
- `flags.go`: YAML-backed buff flag definitions (`FlagSpec`), accessors, `IsValidFlag`, and the `All` sentinel. Flags are plain strings everywhere; there is no `Flag` type. Flag data files live in `<DataFiles>/buffs-flags/<flag>.yaml`.
- `flags_admin.go`: CRUD helpers for flag specs. Locked flags reject edits/deletes; the flag identifier cannot be changed.
- `flags_plugin.go`: module flag data-file integration. `RegisterFlagFS(...)` registers plugin filesystems; `loadPluginFlags` merges embedded `buffs-flags/*.yaml` (disk wins on duplicate ids).
- `admin.go`: CRUD helpers that persist buff YAML and refresh in-memory state.
- `plugin.go`: module data-file integration. `RegisterFS(...)` registers plugin filesystems; `loadPluginBuffs` merges embedded `buffs/*.yaml` into the spec map inside `LoadDataFiles` (after disk load, before items load). `RegisterBuffScript(buffId, src)` registers embedded JS; `BuffSpec.GetScript()` checks it before the disk path. Disk buffs win on duplicate ids.

## Working Rules

- Keep spec fields and runtime behavior aligned. If you add a field to `BuffSpec`, update validation, load/save paths, and any admin helpers in the same change.
- Preserve the `buffIds` and `buffFlags` indexes in `Buffs`; changes to add/remove/expire behavior usually need `Validate()` or equivalent rebuild logic to stay correct.
- `CombatRounds` (Phase 30a): the buff's `TriggersLeft` counts combat rounds and is ticked by the combat loop (`internal/status`), never by game rounds (its `triggerrate` is set too long to fire); `GetDurations` converts to game rounds for display.
- `MaxStacks` / `Buff.Stacks` (Phase 30a): reapplying a live buff refreshes its count and adds a stack up to `MaxStacks`; a new buff starts at 1 stack. Only stacking statuses (bleeding) set it.
- `TriggerCount`, `TriggerRate`, and permanent buffs interact. Check both one-shot and repeating behavior before changing expiration logic.
- Flag changes are cross-cutting. Flags are plain strings (e.g. `"hidden"`, `"perma-gear"`); search for the literal flag string outside this package before renaming or changing semantics. Runtime flag checks are lenient: an unknown flag logs a warning and is treated as no-match. The `All` sentinel (`""`) matches any buff. New flag definitions must be added as `<DataFiles>/buffs-flags/<flag>.yaml`.

## Verification

- Prefer targeted Go tests for this package first: `go test ./internal/buffs`.
- If buff behavior changes affect combat, items, or scripting flows, run broader validation before claiming safety.
