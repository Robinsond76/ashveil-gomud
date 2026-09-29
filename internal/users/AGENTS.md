# Users Package Guide

## Scope

- Use this file for account records, indexing, storage, search, authentication helpers, copyover support, and user persistence in `internal/users`.
- This package carries user data and login-critical behavior. Favor compatibility and safety over cleanup.

## Roles and Permissions

Three roles exist: `user` (no elevated access), `mod` (discrete permissions), `admin` (full access).

`UserRecord` carries:
- `Role string` — one of `RoleUser`, `RoleMod`, `RoleAdmin`, `RoleGuest`.
- `Permissions []string` — the list of permission keys granted to this user. Only meaningful for `mod` role; ignored for `admin` and `user`.

### Permission checking

```go
user.HasRolePermission("mobs.read")        // exact or prefix match, no simpleMatch
user.HasRolePermission("room", true)        // simpleMatch: "room" also matches granted "room.edit"
user.HasPermission("mobs.read")             // convenience alias, no simpleMatch
```

- `admin` always returns `true`.
- `user`/`guest` always returns `false`.
- `mod` checks `Permissions` with prefix semantics: a granted key of `room` satisfies `room.edit`, `room.edit.exits`, etc.
- `simpleMatch=true` additionally allows the inverse: requested `room` matches granted `room.edit`.

### Modifying permissions

Permissions are managed via the admin web UI (`/admin/users`) and the API:
- `GET /admin/api/v1/users/{userid}/permissions` — returns current permission list (open to any mod/admin).
- `PUT /admin/api/v1/users/{userid}/permissions` — replaces the full list. Requires `users.write`. Validates all keys against the catalog in `internal/web/api_v1_permissions.go`. Cannot target admin accounts.

Only `*.write` and command permission keys exist. There are no `*.read` keys — all read access is open to any authenticated mod/admin.

## Replay users (Ashveil 32b)

- `replay.go`: a tutorial replay is a throwaway `UserRecord` with
  `ReplayOf` set (durable, `replayof`), an id from `ReplayUserIdBase` up
  (never from the user index), and username `replay:<id>`. It is in
  neither index, so it can't be logged into; load it with `LoadUserFile`.
- `RemoveUserFile` refuses an online user and is idempotent; it is the
  final step of `events.UserPurged` (`internal/hooks/UserPurged_HandlePurge.go`).
- `OfflineReplayUserIds` lists replay files not online (the boot sweep).

## Working Rules

- Preserve on-disk compatibility for user records and user index behavior unless the task explicitly includes a migration.
- Be careful with password and login changes. Existing authentication, legacy-password migration, and reconnect/copyover flows are easy to break indirectly.
- Keep user search and indexing changes aligned with the existing storage/index flow rather than introducing a second lookup path.
- If a change affects item storage, online info, or account migration behavior, review the corresponding persistence code together.
- Avoid mixing unrelated gameplay refactors into user-data changes.

## Verification

- Run targeted `internal/users` tests for indexing, password, migration, or persistence changes.
- Run `go test ./internal/users/...` to cover `HasRolePermission` and `HasPermission` logic.
- If the change affects login or reconnect behavior, verify the exact code path changed and note any live flows not exercised.
- Use broader repo checks when the user-package change crosses into web auth, copyover, or item storage.
