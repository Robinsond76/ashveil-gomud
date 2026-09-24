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
