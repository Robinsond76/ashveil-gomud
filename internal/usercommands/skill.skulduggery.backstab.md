# skill.skulduggery.backstab

## Skill Tag & Training

- Skill: `skulduggery` (`internal/skills/skills.go`)
- Trained at the Thieves Den (room 491), levels 1-4.
- `backstab` is the level 2 ability of the skulduggery skill tree.

## Overview

`backstab` initiates a special aggro attack against a target while the player is hidden. It sets the player's aggro state to `BackStab` type with a 2-round wait, meaning the backstab strike resolves through the normal combat system after a delay.

## Preconditions

- Skulduggery skill level >= 2.
- Player must currently have the `Hidden` buff flag (must be sneaking).
- The equipped weapon (main hand and off-hand if applicable) must be a backstab-compatible subtype — checked via `items.CanBackstab(subtype)`. Incompatible weapon types are rejected with a message.
- For player targets: PvP rules must allow the attack, and the target must not be in the player's party.
- For mob targets: the mob must not be charmed by the player.
- Phase 32c: refused in a battle (`BattleUnderWay`), and at a mob that isn't charmed out of one (`NotAnOpener`): a backstab doesn't start a fight.

## Target Resolution

- No argument: automatically targets the first mob or player that is currently attacking the user.
- With argument: uses `room.FindByName(rest)` to find a mob or player in the room.

## Execution Flow

1. Skill level check — returns `false` if level < 2.
2. Hidden check — rejects if not sneaking.
3. Weapon subtype check — rejects if any equipped weapon cannot backstab.
4. Target resolution.
5. Fires a `SkillUsed` event (`skill: skulduggery`, `details: backstab`).
6. Calls `user.Character.SetAggro(targetPlayerId, targetMobId, characters.BackStab, 2)`.

## Aggro Type

Setting aggro to `characters.BackStab` with a wait of `2` tells the combat system to execute a backstab strike after 2 rounds. The actual damage multiplier and mechanics are handled in the combat resolution layer, not in this file.

## Notes

- There is no cooldown on `backstab` itself.
- The player remains hidden until the backstab resolves or until the `Hidden` buff is cancelled.
- Attempting to backstab with no valid target sends "You attack the darkness!" and returns without setting aggro.
