# Usercommands Package Guide

## Scope

- Use this file for player command registration and command-side behavior in `internal/usercommands`.
- This package is heavily user-facing and tightly coupled to many gameplay systems.

## Admin Command Permissions

Every command registered with `AdminOnly: true` in `userCommands` is gated by `HasRolePermission` in `TryCommand`. When a `mod` user runs a command they lack permission for, they receive:

```
You don't have permission to use <command>.
```

### Permission key conventions

Each admin command file has a `/* Role Permissions: ... */` comment block listing the permission keys it checks. The top-level command key matches the command name (e.g. `teleport`, `mob`, `room`). Sub-commands use dot-notation (e.g. `mob.create`, `room.edit.exits`).

The top-level check in `TryCommand` uses `simpleMatch=true`: granting `room` lets the user run the `room` command and all sub-commands. Individual sub-command handlers call `user.HasRolePermission("room.edit.exits")` (no simpleMatch) for finer control.

### Adding a new admin command

1. Register it in `userCommands` with `AdminOnly: true`.
2. Add a `/* Role Permissions: ... */` comment block listing all permission keys the command checks.
3. Inside the handler, call `user.HasRolePermission("cmd.subaction")` for each sub-action that warrants a separate permission.
4. Add the permission key(s) to `allPermissions` in `internal/web/api_v1_permissions.go` under `Category: "Commands"` with no `ReadKey`.

## Working Rules

- Keep command changes narrowly scoped. Avoid bundling broad gameplay refactors into a command fix.
- Preserve command names, argument expectations, and admin/user availability unless the task explicitly changes the player-facing contract.
- If a command interacts with another subsystem, review the subsystem rule as well instead of open-coding assumptions in the command.
- Prefer existing prompt, parsing, and helper patterns over inventing a command-local flow.
- Be careful with command registration or removal; that can affect help, aliases, autocomplete, and admin tooling.

## Verification

- Run targeted tests for the changed command path where available.
- Use at least one higher-level validation path that exercises the actual command behavior.
- If a command change is mostly prompt-driven or integration-heavy, note what was not exercised live.

## Ashveil panels (Phase 26a)

- `status`, `conditions`, `inventory`, and `experience` show Ashveil state from `internal/companyview` (the read model) through the `summaryFor` seam. The Ashveil parts live in `*.ashveil.go` files; keep the engine files' changes to calls into them, so upstream merges stay small.
- `status` fills optional panels only when the layout defines them (`templates.PanelLayout.HasPanel`): the default world's `panel-layouts/character/status.yaml` has `vitals` and `company`; the upstream `empty` world's doesn't and keeps the engine sheet.
- `TryCommand` defers `companyview.RefreshUser`, which refreshes the prompt cache after every command. Don't read game state from a prompt token: prompts are built off the game loop (see `internal/companyview/AGENTS.md`).
- Tests that point the data files at a world use `configs.RestoreOverrides`, not `AddOverlayOverrides`, which skips a key already set.
