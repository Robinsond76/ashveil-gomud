# Mobs Package Guide

## Scope

- Use this file for mob lifecycle, mob instances, persistence/loading behavior, and mob-side interactions in `internal/mobs`.
- Mob changes often affect combat, rooms, scripting, and AI behavior.

## Working Rules

- Preserve the distinction between mob templates/specs and live mob instances.
- Be careful with spawn/despawn, room placement, and instance lookup changes. Those assumptions are reused across many systems.
- If a change affects charmed mobs, pets, or scripted mobs, inspect the connected package behavior rather than patching only one call site.
- Avoid folding AI policy into this package when it belongs in mob commands, hooks, or modules.

- Ashveil (Phase 22b): `NewMobById` gives each instance its own copy of the template's `Items` slice. Keep it that way: the struct copy of the template otherwise shares the backing array, and removing an item from a live mob would rewrite the template. `NewMobByIdNoElite` spawns without the elite roll, for company companions whose level is durable.

## Verification

- Run targeted mob-package tests when changing lifecycle or lookup behavior.
- Use a higher-level check for changes that affect combat, room movement, or scripting.
- Call out untested instance-lifecycle or persistence edges when behavior changed.

## Documentation

- Keep only durable mob-lifecycle rules here.

## Ashveil Phase 30c: enemy personalities

- `targeting`/`targetingnoise` on a mob template (overriding its race's, `internal/races`) set how it re-aims at a company (`Mob.Personality`). Only `internal/hooks`' enemy upkeep reads it, and only when the enemy re-aims (joins, loses, or can't reach its target). No personality keeps the old weakest pick unchanged.

## Ashveil Phase 30d2: wind-ups

- `windups` on a mob template maps a wind-up ability id (`internal/windup`) to the percent of its turns it starts one. Only `internal/hooks` (`combat_windup.go`) reads it, and only for an enemy (never a company companion or a charmed mob). The template map is shared by every instance; never write to it. A mob that shoots (a bow's `DefaultAttack` becomes `Shooting`) never winds up, so don't give one `windups` (`TestShippedWindUpsAreRegistered`).

Phase 30f templates may opt into `leap` and `sweep`; both replace normal enemy melee actions and have three-combat-round cooldowns. `stealth` is an optional non-negative ambush detection override. Runtime Ambush fields are never persisted or copied into template authority.

## Ashveil Phase 38a: bosses

`boss: true` on a mob template marks a boss: the Witch's hexes (`internal/hexes`) resist it 25 points more and hold it half as long. Nothing else reads it yet.

## Ashveil: idle chatter limits

Says, saytos, shouts and emotes a mob queues during its idle turn (`BeginIdle`/`EndIdle`, set by `internal/hooks` `HandleIdleMobs`) pass `idleChatterAllowed` (`chatter.go`): one decision per turn from the cooldown (`GamePlay.MobChatterCooldownRounds`) and each listener's memory of the line (`GamePlay.MobChatterMemoryRounds`). Commands queued outside an idle turn are never held back, so don't wrap replies, combat or conversation steps in an idle turn.

## Ashveil Phase 68: townsfolk

`townsfolk: [tags]` on a mob template makes it a talker (`Mob.Townsfolk`). Only `internal/hooks` `HandleIdleMobs` reads it, to ask `internal/townsfolk` what the NPC says about a listener's deeds; the line goes through the same idle-chatter limits as any say. The slice is shared by every instance of the template; never write to it.
