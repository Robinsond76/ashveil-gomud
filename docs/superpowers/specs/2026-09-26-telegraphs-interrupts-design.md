# Potential Phase 29d: Wind-Ups, Telegraphs, and Interrupts

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.
Adapted from the external reference's §§8, 12, and 17 to round-based
combat. Sub-round beats and the continuous timeline are declined.

## Goal

- **Telegraphs:** heavy blows and big spells take visible time to prepare,
  measured in whole combat rounds, and they can be broken.
- **The ogre's fantasy:** the ogre isn't slow to act; it is visibly
  preparing something terrible.
- **The healer's risk:** a healer can be pressured, but light nicks don't
  break a prayer.

## Prior-art check (2026-09-26)

- **Spells already take time.** They have `waitrounds`: `heal` 2, `mm` 1,
  `sparks` 1. The chanting lines run during those rounds, so cast time
  exists already.
- **Mob special attacks** can be expressed as `combatcommands` or buff
  scripts. There is no telegraph concept today.

## Rules

1. **Wind-ups.** An ability can declare a wind-up of N combat rounds.
   - The attacker is committed: it takes no other action while winding up.
   - Others act normally.
   - The ability lands at the end of the wind-up.
   - A basic attack has none. An ogre's Crushing Blow has 1.
   - Spells keep their `waitrounds` as chant time.
2. **Interrupt threshold.** Each wind-up or chant has a threshold. Sources
   of pressure:
   - a shield bash, which a guardian does automatically when the tactic is
     on;
   - stagger, stun, or knockdown (29a);
   - a heavy crit;
   - specialised anti-caster abilities.
3. **Accumulation.** Pressure accumulates within the one wind-up. When it
   exceeds the threshold, the action is interrupted.
4. **Light hits don't interrupt.** Light incidental damage adds no pressure
   unless the caster's concentration fails. Concentration and stability,
   derived from existing stats (Vitality, Mysticism), resist.
5. **Outcomes, per ability:**
   - cancelled outright;
   - progress lost, restarting from the first word;
   - delayed a round;
   - part of the mana spent;
   - a short recovery penalty.
6. **Enemy casters.** New content, such as a goblin hexer or bandit
   hedge-witch, with chanted spells the company can interrupt.

## Events and text

Wind-up start and land, cast start/progress/complete, and interrupt
(pressure, success or failure) are all events (28b):

```
Ironhide plants his feet and drags the great club up over his shoulder. (winding up: Crushing Blow, 1 round)
Tamsin Reed rams her shield into Ironhide's gut. The ogre grunts and barely rocks. (2 damage, interrupt pressure 3 of 5)
Ironhide's Crushing Blow comes around in a flat sweep across the front line.

Tamsin Reed drives her shield into Ironhide's knee. (2 damage, interrupt pressure 4 of 5)
Brother Oswin's cudgel cracks down on the same knee. (critical hit, 4 damage, staggered)
Ironhide bellows, and the whole road seems to shake with it.
The club drops from its height as Ironhide staggers back a step. (Crushing Blow interrupted)

The goblin hexer's eyes slide past Tamsin Reed and settle on Brother Oswin. She begins to croak a hex. (chanting: Withering Hex, 2 rounds)
Ysolde's stone takes the goblin hexer square in the jaw with a crack like splitting kindling. (critical hit, 4 damage, staggered)
The goblin hexer reels, spitting teeth, and the hex dies in her throat. (Withering Hex interrupted)
The goblin hexer spits blood and starts the hex again from the first word. (chanting: Withering Hex, 2 rounds)

The first skirmisher slashes Brother Oswin's forearm. He flinches, but the prayer holds. (1 damage, concentration held)
```

## Acceptance criteria

- **Wind-up:** a 1-round wind-up lands one combat round later, and its
  attacker takes no other action.
- **Pressure:** from two sources in the same wind-up, it accumulates and
  interrupts at the threshold.
- **Light hits:** a light hit on a chanting cleric with concentration
  doesn't interrupt.
- **Save/load:** wind-up and chant state survives a save/load for a player
  mid-cast, or the design says why it needn't (a copyover mid-fight
  resets the action).
