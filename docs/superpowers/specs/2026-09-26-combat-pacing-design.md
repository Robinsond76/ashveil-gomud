# Potential Phase 28f: Paced Combat Output

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); the cadence decision below is
open and must be settled first.

## Goal

A round's lines don't arrive as one block. They are released one by one so
the fight appears to happen in real time. Each player chooses fast, normal,
or slow. The owner's normal is about **6 seconds** per round's worth of
lines. The first mock's 3 seconds read too fast.

## Prior-art check (2026-09-26)

- **Timing:** `_datafiles/config.yaml` sets `TurnMs: 50` and
  `RoundSeconds: 4`, so 80 turns make a round. `hooks.DoCombat` runs once
  per `NewRound`.
- **Rounds are shared.** The round is the unit for many systems: survival
  drain, chemistry service (24), alignment drift (21a), buffs, hostility
  timers, and autosave.
- **Delivery:** combat text goes out as `events.Message` (user or room) and
  is delivered when the event queue is processed each turn.

## The cadence decision (open, must settle first)

Lines can't be paced over 6 seconds if a new combat round starts every 4:
- output would fall further behind each round;
- a line would describe a blow already several rounds old;
- a death could arrive after the next round's opener.

Options:

1. **A combat cadence of every second round (recommended).**
   - `DoCombat` resolves a fight every 2 game rounds, 8 seconds, via a new
     `CombatEveryRounds: 2` setting. Everything else keeps its 4-second
     round.
   - Normal pacing fills 6 of the 8 seconds, leaving a breath.
   - Fights take twice as long in wall time. Round-based combat numbers
     (wind-ups in rounds, status durations, cooldowns) count combat
     rounds, not game rounds.
2. **Lengthen `RoundSeconds` to 8.**
   - This slows every round-based system in the world: survival, drift,
     chemistry, and every "N rounds" rule.
   - It touches the travel/rest real-time invariant's tuning.
   - Not recommended.
3. **Keep 4-second rounds and cap pacing at about 3 seconds.**
   - The owner has said that reads too fast.

## Scope (assuming option 1)

- **A per-player paced queue.** Combat-event text for a player is queued,
  not sent at once. Each turn, due lines are released.
- **Pacing settings** (`set combatpace fast|normal|slow|off`):

  | Setting | Round's lines fill | Of an 8-second round |
  |---|---|---|
  | fast | ~3s | |
  | normal | ~6s | |
  | slow | ~7.5s | |
  | off | at once | |

  Off is also the default for screen readers.
- **Spacing:**
  - A critical hit and its pain or death line get a longer gap.
  - The indented lines under an area spell or a multi-heal come in quick
    succession.
  - A companion's death notice follows its death line.
- **Catching up:** if a round has so many lines that they can't fit, the
  gaps shrink. Output never runs past the start of the next combat
  round. Anything still queued then is flushed at once.
- **Non-combat text is never delayed:** says, tells, and your own command
  echoes pass straight through. Input is never blocked.
- **The battle panel (30)** is updated as each line is released, not at
  the end of the round.

## Reference text (normal pacing, times since the round began)

```
[0.0s] Ironhide plants his feet and drags the great club up over his shoulder. (winding up: Crushing Blow, 1 round)
[0.8s] The goblin hexer's eyes slide past Tamsin Reed and settle on Brother Oswin. She begins to croak a hex. (chanting: Withering Hex, 2 rounds)
[1.7s] You begin to murmur. Sparks spit and crawl across your palms. (chanting: Shower of Sparks, 1 round)
[2.5s] The first skirmisher darts wide around the line, hunting for the soft middle.
[3.3s] Garrick Vane's broadsword opens a shallow cut across the first skirmisher's ribs. (3 damage)
[4.2s] Tamsin Reed sees the club rise and sets her shield against Ironhide's knee. (interrupt pressure 3 of 5)
[5.1s] Ysolde sights on the goblin hexer.
[6.0s] Brother Oswin grips his cudgel and watches the club rise, lips moving.
```

## Acceptance criteria

- A wiring test with a fake clock:
  - releases a round's lines over the configured window, in order;
  - never lets a round's lines spill past the next combat round;
  - delivers non-combat messages without delay;
  - with `off`, delivers at once.
- With option 1, a test shows that survival drain, drift, and the world
  clock are unchanged by the combat cadence.
