# Potential Phase 30c: Pre-Fight Tactics — Roles, Personalities, Guards, Companion Casting

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.
It covers handoff §36 items 6 (guard reactions) and 9 (target-selection
personality), formerly in deferred Phase 11d.

## Owner decisions

- **No commands mid-fight.** The player doesn't pick skills or targets
  during a fight. The company acts automatically, by a strategy set
  beforehand. Manual orders mid-fight (`company protect`, `order … bash`)
  are tabled.
- **Clerics act as needed**, healing on their own during a fight.

## Prior-art check (2026-09-26)

- **Targeting:** `internal/engagement.AssignTarget` (11b) chooses a target
  by a pluggable rule (weakest, strongest, random) filtered by 11c's
  legality predicate. It is already called on reassignment.
- **Mob combat actions:** mob templates can list `combatcommands`, which
  `DoCombat` picks from at random during a fight (`internal/mobs`,
  `hooks/NewRound_DoCombat.go`). Companions have none today; a cleric
  companion only swings.
- **Archetypes** (17a) give companions warrior, rogue, wizard, cleric, or
  ranger, with spell schools.
- **Formation** (3, 11c): the 3×3 grid, reach, and front-row interception.

## Scope

1. **`company tactics`**, a durable company setting on the company record,
   that survives restart. It shows and sets:
   - **Focus:** leader first, casters and healers first, nearest,
     weakest, or strongest. Each rule falls back to the nearest legal
     target.
   - **Healing:** heal anyone below a threshold (default: half).
   - **Interrupts:** break heavy blows and spells when someone can (30d).
     On or off.
   - **Guard:** who each guardian guards.
   - **Rotate the wounded:** swap a badly hurt front-liner back when
     someone can take the place, at the cost of a round. On or off.
   - **Mercy:** always ask (the owner's default; see 30e).
2. **Roles and personalities.** Each member has a role from their
   archetype, which can be changed in tactics. The role decides whom they
   target and what they do:

   | Role | Behaviour |
   |---|---|
   | **Guardian** | guards an assigned ally; bashes heavy wind-ups (30d); otherwise fights threats to allies |
   | **Duelist** | fights whoever engages them; favours dangerous front-line foes |
   | **Hunter** | the focus, else the wounded, the exposed, and the lightly armoured |
   | **Cleric** | heals below the threshold (single or group heal by how many are hurt); otherwise fights |
   | **Wizard** | area spells when foes cluster (30f); bolts on the focus |
   | **Rogue** | goes for the exposed and the knocked down |

   - The **focus rule** overrides a role's own preference, and the text
     says so ("…but she keeps her sights on the goblin hexer, as
     ordered.").
   - The **player character** follows the same tactics, as an automatic
     fighter, by their own archetype's role.
   - Open: whether the player's own role is fully automatic, or only
     their targeting.
3. **Enemy personalities**, per mob template or race:

   | Enemy | Targets |
   |---|---|
   | wolf | hunts the wounded or isolated |
   | ogre | the nearest front-liner |
   | goblin hexer | casters and healers |
   | wild beast | some randomness |

   Each is an `engagement` rule plus a little noise. Enemies are competent
   but not perfect.
4. **Guard reactions.** A guardian has a small number of guards per fight
   (default 2), which refresh after the guardian completes a normal action
   cycle.
   - **What a guard does:** redirects a blow aimed at the guarded ally.
     It's an interception, and now narrated.
   - **Lost while knocked down or stunned** (30a).
5. **Companion casting.** Companions cast their archetype's spells by their
   role, with real mana and chant rounds (spells' `waitrounds`). The mob
   `combatcommands` path, or a new tactics driver, is chosen in the design
   pass.
6. **Events:** target change, guard used, guard exhausted, and cast
   start/complete (29b).

## Reference text

```
> company tactics
Company tactics
  Focus        casters and healers first, then the nearest foe
  Healing      heal anyone below half health; heal wounds after the fight
  Interrupts   break heavy blows and spells when someone can
  Mercy        ask me when a foe yields

  You            wizard      area spells on clusters, bolts on the focus
  Tamsin Reed    guardian    guards Brother Oswin; bashes heavy wind-ups
  Garrick Vane   duelist     fights whoever engages him; front-line threats
  Brother Oswin  cleric      heals below half; fights when no one needs him
  Ysolde         hunter      the focus, else the wounded and exposed

> company tactics tamsin guard ysolde
Tamsin Reed will guard Ysolde.
> company tactics focus leader
Your company will strike at the enemy's leader first, then the nearest foe.
```

In a fight:

```
The first cutthroat slips around the line and lunges at Brother Oswin. Tamsin Reed throws her shield across him, and the dagger skids off the rim. (guard, 1 left)
Ysolde's eyes flick to the bleeding skirmisher, but she keeps her sights on the goblin hexer, as ordered.
Brother Oswin kneels over Tamsin Reed and begins to pray, heedless of the knife at his back. (chanting: Minor Heal, 2 rounds)
Brother Oswin raises his arms, and a warm light washes over the company.
    Garrick Vane (3 healed) · Ysolde (2 healed) · you (4 healed)
```

## Acceptance criteria

- `company tactics` settings survive restart.
- Through the real round:
  - the focus rule picks the right target;
  - a guardian's guard redirects a blow and is spent, then refreshes;
  - a cleric companion casts a real heal, with mana and chant rounds, when
    an ally falls below the threshold;
  - a knocked-down guardian can't guard.
- Enemy personalities pick targets by their rule, with the noise seeded in
  tests.
