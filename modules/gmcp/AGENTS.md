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
- The web client's Party window (`window-party.js`) renders Company and Players as separate sections, with `textContent` only. It reads `Client.GMCPStructs` on each render, so a window closed while payloads arrived is current when reopened, and it keeps keyboard focus on the same card across a rebuild. `scripts/browser/company-panel-check.mjs` checks it in Chromium: `NODE_PATH=$(npm root -g) node scripts/browser/company-panel-check.mjs`.
