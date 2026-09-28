# skill.cast

## Skill Tag & Training

- Skill: `cast` (`internal/skills/skills.go`)
- Levels 1-4.

## Overview

`cast` is the primary spell-casting command. The skill level does not gate which spells are available — it affects the rate at which the player gains proficiency in each spell. The actual spell effects are defined in spell scripts.

## Skill Level Effects on Proficiency

| Level | Proficiency gain rate |
|---|---|
| 1 | Base rate |
| 2 | 125% |
| 3 | 175% |
| 4 | 250% |

Proficiency gain is handled by the spell resolution system, not in this command file.

## Preconditions

- Cast skill level >= 1.
- The player must know the spell (`user.Character.HasSpell(spellName)`).
- The player must have sufficient mana (`Character.Mana >= spellInfo.Cost`).
- Mana is deducted immediately when the cast is queued.
- Phase 32c: refused while the caster is in a battle (or aiming at a mob) with `BattleUnderWay`; a harmful spell at a mob that isn't the caster's charmed companion is refused out of a battle too, pointing at `attack <group>` (`NotAnOpener`). Cast is for spells outside fights.

## Syntax

```
cast <spellname> [on] [target]
```

The optional `on` particle between the spell name and target is stripped before target resolution.

## Execution Flow

1. Skill level check — rejects if level 0.
2. Argument parsing.
3. Spell lookup via `spells.GetSpell(spellName)`.
4. Spell known check and mana check.
5. Target resolution based on spell type (see below).
6. Calls `scripting.TrySpellScriptEvent("onCast", ...)` — scripts can intercept and cancel.
7. If not intercepted: fires `SkillUsed` event, deducts mana, fires `CharacterVitalsChanged`, and queues the cast via `user.Character.SetCast(waitRounds, spellAggro)`.

## Target Resolution by Spell Type

| Spell type | Target logic |
|---|---|
| `Neutral` | No target lookup; `SpellRest` is passed as the argument string |
| `HelpSingle` | Named target if provided; defaults to self if no argument |
| `HarmSingle` | Named target if provided; defaults to current aggro target, then any mob/player fighting the caster |
| `HelpMulti` | Always targets self + all party members + their charmed mobs |
| `HarmMulti` | Named mob target: all mobs in room. Named player target: all PvP-valid players. No target: all mobs aggro to caster, then aggro players. Falls back to all fighting mobs. |
| `HelpArea` | All players and all mobs in the room |
| `HarmArea` | All PvP-valid players + all mobs in the room |

For `HarmSingle` and `HarmArea`, PvP rules are checked before adding player targets.

## Spell Resolution

The cast is queued with a wait-round delay (`spellInfo.WaitRounds`). The actual spell effect fires after that delay through the combat/spell resolution system, driven by the `SpellAggroInfo` struct. This command file only handles the initiation of the cast.

## Notes

- If no valid targets are found after resolution, the cast is aborted with "Couldn't find a target for the spell."
- The `onCast` script event fires before mana is deducted — if the script cancels the cast, no mana is lost.
