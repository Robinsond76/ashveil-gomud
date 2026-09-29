# GMCP Module Guide

## Scope

- Use this file for the `modules/gmcp` protocol module, including GMCP payload dispatch, connection capability tracking, and web-client text-prefix handling.
- This module bridges server events to telnet and web client protocol behavior.

## Working Rules

- Preserve existing namespace contracts unless the task explicitly changes a GMCP payload shape.
- Be careful with telnet negotiation versus WebSocket text-prefix behavior; those paths are related but not identical.
- If a change affects client capability tracking or Mudlet-specific behavior, verify the caller assumptions in web client or term code too.
- Prefer additive namespace changes over silently repurposing an existing payload.
- `GMCPCharModule_Payload_Inventory_Worn` is a `map[string]GMCPCharModule_Payload_Inventory_Item` keyed by slot name. It is populated by `buildWornPayload`, which iterates `items.AllEquipSlots()`. Adding or removing a slot in `internal/items/itemspec.go` is sufficient; no manual update to this module is needed.

## Verification

- Run targeted module/package tests for GMCP behavior changes.
- Verify the exact namespace or negotiation path changed, not just generic module load behavior.
- Call out client compatibility risk when changing payload shapes or negotiation rules.

## Documentation

- Keep this file about protocol compatibility and integration guardrails.

## Ashveil Company (Phase 26b)

- `gmcp.Company.go` builds `Company` (snapshot) and `Company.Vitals` from the `internal/companyview` summary, on `companyview.OnRefresh` (every round and after every command, on the game loop), and sends only on change: structure changed → `Company`, only health/needs → `Company.Vitals`. `PlayerSpawn` and a `!!GMCP(Company)` request resend the snapshot; `PlayerDespawn` forgets the user.
- Members are keyed by member key, never by name. Unknown values are `null`. Send a user only their own company.
- Anything that changes with time (health, needs, activity, rest and rescue countdowns) belongs in the live half (`companyLive`, sent as `Company.Vitals`), with countdowns in whole minutes. Putting a countdown in the structure resends the snapshot every round.
- Nothing is built for a telnet connection that hasn't accepted GMCP. `AcceptGMCPForTest` marks a test connection as accepted, as a web client's is.
- Phase 32g: the feed's *extras* (`companyExtra`) are more messages kept current the same way, each sent only when its JSON changes: `Company.Inventory` (`gmcp.CompanyInventory.go`, every member's gear, horses, cargo, the load split; items carry `ref`, the `!<id>:<uuid>` a command resolves to exactly that item, `name` for commands, and `label` without markup) and `Company.Camp` (`gmcp.CompanyCamp.go`). The client stores them under `Company`, which a new snapshot replaces, so sending `Company` forgets the extras and they follow it. `forget`, `prune`, and the purge drop them with the rest. `Company.Camp`'s rest countdown is in seconds, so it resends each round during a one-minute rest (accepted: the progress bar needs it). `Company.Vitals` carries each member's `mp`/`mp_max` when known, and `Company` each member's `strategy`.
- The web client's company dock renders all of this: `window-company.js` (Status, Inventory, Camp), `window-combat.js`, `window-vitals.js`'s rows, with `textContent` only, reading `Client.GMCPStructs` on each render and keeping keyboard focus across a rebuild. `scripts/browser/dock-windows-check.mjs` checks them in Chromium: `NODE_PATH=$(npm root -g) node scripts/browser/dock-windows-check.mjs`.
- Phase 32g2: `Company.Battle` (`gmcp.CompanyBattle.go`) is a third extra: the player's battle from `battle.Current` and the live room (never the combat event stream): the group's living, visible enemies with their 29d label, cell, `enemyparty.HealthWord` (never numbers), `reach` (only when the player is placed, as `scout`'s `*`), and target (a member key, or an `others` id); `fallen` (fallen or gone, in instance order); `company` (members' targets by key, only on listed enemies); `others` (outsiders an enemy strikes, by name); `waiting` (`battle.Waiting`). `{}` out of battle; in the dark only `dark: true`. A fallen enemy is named unless last seen hidden (`battleSeen`, runtime, per player and battle, pruned each round). `buildBattle` is pure over `battleFacts`; `gatherBattle` reads the game. The client carries the battle over a `Company` snapshot until it is re-sent. `window-combat.js` renders it; `dock-windows-check.mjs` checks it.
