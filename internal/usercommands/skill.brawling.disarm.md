# skill.brawling.disarm

## Skill Tag & Training

- Skill: `brawling` (`internal/skills/skills.go`)
- Trained at the Soldiers Training Yard (room 829), levels 1-4.
- `disarm` is the level 4 ability of the brawling skill tree.

## Overview

`disarm` is a combat-only move that attempts to knock the weapon out of the current aggro target's hand. On success the weapon is moved from the target's equipped weapon slot into their backpack inventory.

## Preconditions

- Brawling skill level >= 4.
- Player must be in combat (`Aggro != nil`).
- Cooldown: **1 real minute** per use (keyed `brawling:disarm`).
- Phase 32c: refused in a battle (`BattleUnderWay`).

## Execution Flow

1. Skill level check — returns `false` if level < 4.
2. Combat check — rejects with a message if not in combat.
3. Cooldown check (applied only if a valid target exists).
4. Fires a `SkillUsed` event (`skill: brawling`, `details: disarm`).
5. Validates the target:
   - Target must not have the `PermaGear` buff flag (prevents disarm entirely).
   - Target must have a weapon equipped (`Equipment.Weapon.ItemId != 0`).
   - Target's weapon must not be remove-locked (`IsRemoveLocked()`).
6. Rolls for success.
7. On success, calls `RemoveFromBody` then `StoreItem` on the target to move the weapon to their backpack.

## Success Roll

```
chanceIn100 = (attacker.Speed(adj) + attacker.Smarts(adj)) - (target.Strength(adj) + target.Perception(adj))
chanceIn100 = max(0, chanceIn100) + 5
roll = rand(100)
success = roll < chanceIn100
```

The base chance is 5%, increased by the attacker's Speed+Smarts advantage over the target's Strength+Perception.

## Outcome Details

On success the weapon is not dropped on the floor — it goes into the target's backpack. The target can re-equip it next round.

## Messaging

| Outcome | Attacker sees | Room sees | Target sees (PvP) |
|---|---|---|---|
| Success (mob) | "You disarm \<mob\>!" | "\<user\> disarms \<mob\>!" | — |
| Fail (mob) | "You try to disarm \<mob\> and fail!" | "\<user\> tries to disarm \<mob\> and fails!" | — |
| Success (player) | "You disarm \<player\>!" | "\<user\> disarms \<player\>!" | "\<user\> disarms you!" |
| Fail (player) | "You try to disarm \<player\> and miss!" | "\<user\> tries to disarm \<player\> and misses!" | "\<user\> tries to disarm you and misses!" |
| PermaGear | "Some force prevents you from disarming \<target\>!" | — | — |
| No weapon | "\<target\> has no weapon to disarm!" | — | — |
| Remove-locked | "\<target\>'s weapon is bound to them and cannot be disarmed!" | — | — |
