# Company Companion Slice — Design

## Purpose

This Phase 2 slice proves that an Ashveil player can own one persistent companion without treating the game as a temporary layer over GoMud. GoMud remains the engine foundation; its existing mob, command, friendly/charm, room-movement, user persistence, event, and module mechanisms are reused.

The feature is intentionally a development recruitment stand-in. It does not introduce shops, payment, formation combat, human company members, travel, supplies, or the eventual five-member company limit.

## Vocabulary and boundaries

Use permanent game-domain terms:

- `company`: a leader-owned expedition organization.
- `companion`: an owned mercenary member represented in the world by a GoMud mob.
- `expedition`: later travel, survival, camp, and route systems.

Do not add `ashveil*` package/type prefixes. Do not alter GoMud’s native `parties` package: it remains the engine’s temporary player-party system. A new `internal/company` package owns durable rules and records; a `modules/company` module owns commands and GoMud lifecycle integration.

## Data model and persistence

The initial `Company` record is keyed by leader user ID and contains one `Companion` with its `MobTemplateID`. It must not persist a GoMud mob `InstanceId`, because instances are created at runtime and are invalid after restart.

Module-owned durable data stores the company record separately from generic user fields. A runtime mapping connects leader user ID and saved template to the current live mob instance. The module writes immediately after summon and dismiss, loads before restoration, and serializes the live mapping through a copyover contributor. Copyover restoration reattaches the mapped companion; ordinary restart restoration occurs when the leader reconnects.

## Commands and flow

`company summon <template>` validates a configured allow-list, refuses a leader who already owns a companion, spawns the selected template in the leader’s current room, applies GoMud’s permanent-friendly/charm behavior, tracks it as charmed, and saves the company record.

`company status` reports the saved template and whether its live instance is present and attached to the leader. `company dismiss` despawns the live instance, clears its ownership/charm tracking, removes the durable record, and confirms the action.

On login or reconnect, the module checks for a saved companion. If no live attachment exists, it respawns the template in the leader’s current room and re-applies the same GoMud behavior. Normal GoMud movement carries the charmed mob through ordinary exits; the company module does not create a second follower system.

## Failure rules

Invalid or disallowed templates never create or save a company. A second summon leaves the current companion unchanged. If a saved template is unavailable, the record is retained, no substitute is spawned, and `company status` reports the repairable condition. Dismiss is idempotent: it clears a valid saved record even if the live instance is already absent.

Mob death or destruction must clear the live mapping but preserve the saved record until later lifecycle rules are designed; automatic respawn is limited to login/reconnect in this slice. The feature never mutates global game time.

## Verification

Unit tests cover record validation, save/load, the one-companion rule, invalid-template rejection, idempotent dismissal, and the unavailable-template restore result. Module/integration tests verify spawn, persistent-friendly attachment, ordinary room following, and restore as a fresh instance rather than a persisted instance ID. Run focused Go tests, `make validate`, and `make test` before implementation completion.

## Deferred decisions

Later designs define recruitment economics, a company cap of five, custom names, equipment, injuries, AI orders, formation slots, death/permadeath, copyover continuity details, and travel/camp integration. Those additions extend this record deliberately rather than repurposing the native player-party schema.
