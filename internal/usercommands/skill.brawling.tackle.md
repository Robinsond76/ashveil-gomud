# skill.brawling.tackle

## Skill Tag & Training

- Skill: `brawling` (`internal/skills/skills.go`)
- Trained at the Soldiers Training Yard (room 829), levels 1-4.
- `tackle` is the level 3 ability of the brawling skill tree.

## Overview

`tackle` is a combat-only move that attempts to knock the current aggro target to the ground, applying buff 12 ("Tackled") which prevents them from acting for 2 rounds.

## Preconditions

- Brawling skill level >= 3.
- Player must be in combat (`Aggro != nil`).
- Cooldown: **5 rounds** per use (keyed `brawling:tackle`).
- Phase 32c: refused in a battle (`BattleUnderWay`).

## Execution Flow

1. Skill level check — returns `false` if level < 3.
2. Combat check — rejects with a message if not in combat.
3. Cooldown check — rejects if too soon.
4. Fires a `SkillUsed` event (`skill: brawling`, `details: tackle`).
5. Reads the current aggro target (mob or player).
6. Rolls for success.
7. On success, queues a `Buff` event applying buff 12 to the target.

## Success Roll

```
chanceIn100 = Speed(adj) - target.Perception(adj) + 20
chanceIn100 = clamp(chanceIn100, 20, 80)
roll = rand(100)
success = roll < chanceIn100
```

The chance is always between 20% and 80%. A Speed advantage over the target's Perception pushes toward the ceiling; a disadvantage pushes toward the floor.

## Buff 12 — Tackled

| Field | Value |
|---|---|
| Name | Tackled |
| Trigger rate | 1 round |
| Trigger count | 2 (2 rounds duration) |
| Flags | `no-combat`, `no-flee`, `no-go` |

The target cannot attack, flee, or move for 2 rounds.

## Messaging

| Outcome | Attacker sees | Room sees | Target sees (PvP) |
|---|---|---|---|
| Success (mob) | "You lunge and tackle \<mob\>!" | "\<user\> lunges and tackles \<mob\>!" | — |
| Miss (mob) | "You try to tackle \<mob\> and miss!" | "\<user\> tries to tackle \<mob\> and misses!" | — |
| Success (player) | "You lunge and tackle \<player\>!" | "\<user\> lunges and tackles \<player\>!" | "\<user\> lunges and tackles you!" |
| Miss (player) | "You lunge to tackle \<player\> and miss!" | "\<user\> lunges to tackle \<player\> and misses!" | "\<user\> lunges to tackle you and misses!" |
