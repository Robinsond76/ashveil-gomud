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
- Phase 30c2: a guardian's `strategy` in `Company` carries `ward` (a member key, dropped when it names no member) and `ward_reach: false` when both are placed more than one column apart (`markWards`). `Company.Battle` carries `guards: [{key, left, ward}]` for each guardian on the player's side (`gatherGuards`, from `battle.GuardsLeft` and the stored ward; blank ward: the most hurt), in the dark too. `window-combat.js` shows both; `dock-windows-check.mjs` checks them.

- Phase 34d: `Company.Conditions` is a changed-only extra by owned member key,
  distinguishing live/away-live, recorded away wounds, fallen and unavailable
  state. Temporary buffs, wound durations and persistent buffs are separate.
  `Char.Capabilities` is another changed-only extra, also included in full Char
  responses. It uses shipped ability specs, automatic spell settings and the
  archetype provider's current specialist eligibility and camping's manual
  cooking view (configured recipes, ranks, ingredients, camp and capacity); no progression catalogue.
  `Char.Skills` retains its array contract and adds each skill's `max_level`.
- Phase 34 review: an extra may set `buildKeyed` to compare a change key
  instead of its body (it receives the key last stored). `Company.Conditions`
  uses it: a timed effect carries `seconds_left`/`seconds_total`, the key holds
  the round it ends instead (an end within one round of the last keyed one is
  kept, since refreshes before and after the round's buff tick differ by one),
  and `company-data.js` counts it down from the message's arrival, so the
  message resends when an effect starts, is refreshed or ends, not every
  round. `harmful`/`helpful` come from `BuffSpec.Effect()` (stat modifiers and
  the `harmful`/`helpful` markers in `buffs-flags` data); neither is set when
  unknown or secret. `Company.Equipment` is built only for a client showing the
  Gear editor: `window-gear.js` sends `!!GMCP(Company.Equipment open <slot>)`
  or `... closed` when that changes (and again after a `Company` snapshot,
  since `PlayerSpawn` clears it; `prune` drops the offline), and only the
  named slot's choices are previewed (`EquipmentViewFocused`; other slots are
  `pending`). The view is cached in the company module, rebuilt only when the
  leader's character (less what ticks each round: vitals, cooldowns, buff
  counters), cargo, load, availability or the slot changes, or every 15
  rounds. `ASHVEIL_LOAD_BENCH=1 go test ./modules/company -run
  TestCompanyRefreshLoad -v` measures a player's whole company refresh.

- Phase 40e: `Company.Battle.Event` (`gmcp.CompanyBattleEvent.go`) is a
  stream, not state: a combat stream sink turns each event of the leader's
  fight into an entry and queues `events.CombatData`, which
  `hooks.CombatData_Hold` orders with the round's narration (held with it for
  a player who paces combat, released with the next text line or the round's
  last, flushed with it) and hands back through `hooks.SetCombatDataSender`
  as one message per released batch. Refs match `Company.Battle` (member key,
  `leader` for the player; `me` only without a company; `m:<instance>`, `u:<id>`; `?` for an enemy in the dark or hidden, whose
  statuses are also dropped). Never add enemy health, unshown numbers, or
  secret statuses. Sent only to the web client, or a client that listed the
  module in `Core.Supports.Set`. The web client does not store it:
  `Client.onBattleEvents`. `round` is the server's round counter;
  `fight_round` (40g review) counts the fight's own rounds from 1, for
  display, and `spell_name` is a spell's display name beside its id.
  Phase 40g2: each message carries `pace` (the receiver's combat pace, so the
  screen need not infer it), and a fight's happenings are also relayed to its
  leader's consenting allies (`parties.AlliedLeaders`) who fight some of the
  same mobs here in a battle of their own, with the ally's members as
  `a:<leader>:<key>` (the ids `Company.Battle.allies` lists). Only the kinds
  in `allyKinds`; never an ally's health numbers, statuses (their numbers and
  status fields are scrubbed), tactics, or fight start/end/focus. `Company.Battle`
  also carries `allies` (formation cells, class and health words) and `nerve`
  ("faltering" while `morale.Losing` holds for the company).

- Phase 57: `Char.Skills` entries also carry `title` (the skill's display name) and
  `description` (what it does), additive. The web client's Character > Skills renders
  both; it no longer renders `Char.Jobs`, which stays for other clients.
