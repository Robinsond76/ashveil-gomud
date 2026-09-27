# Potential Phase 29b: Combat Event Stream and Battle Summary

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.

## Goal

Every combat happening becomes one structured event. Narration (29c–29e),
paced delivery (29f), the battle summary (below), the battle panel (31), and
balancing data all read the same stream. Combat code produces events;
presentation consumes them.

## Prior-art check (2026-09-26)

- **Where combat builds its text:** `internal/combat/combat.go` resolves an
  attack into an `AttackResult`, which carries text for the attacker, the
  defender, and the room, plus `Hit`, `Crit`, `DamageToTarget`, and buff
  ids. `hooks/NewRound_DoCombat.go` and `hooks/combat_formation.go` send
  those texts straight to users and rooms.
- **Spells:** their scripts send their own text
  (`_datafiles/world/default/spells/*.js`).
- **Death:** the core handles it. The company's fallen notice comes from
  `modules/company` (25b).
- **The events bus** (`internal/events`) is the project's pattern for
  cross-module signals, and modules subscribe with `RegisterListener`.

## Scope

1. **Event kinds.** One `CombatEvent` type with a kind, a round number, the
   source and target (user or mob id, and company member key if any), the
   enemy party id (11a), and kind-specific fields. The kinds:
   - fight start and fight end;
   - target change;
   - attack: hit, miss, or critical, with damage and weapon type;
   - heal, including the amount held back by a wound limit;
   - cast start, cast progress, and cast complete;
   - wind-up start and wind-up land;
   - interrupt, including pressure applied and whether it succeeded;
   - status applied and status expired;
   - wound change;
   - guard used and guard exhausted;
   - yield and flee;
   - death or incapacitation;
   - mercy decision.
2. **Producers.** The existing combat paths emit events at the point they
   now build text. In this phase they still send today's text too, so 29b
   changes no player-visible output except the summary.
3. **Fight identity.** A fight is the engagement between one company and
   one enemy party, from the first blow to the last enemy's fall, flight,
   or yield, or the company's defeat. It gets an id so events group.
4. **Battle summary.** At fight end, the leader (and each company player)
   gets a compact block built from the fight's events:
   - damage dealt, company and enemies;
   - healing, and how much a wound limit held back;
   - most damage by member;
   - the highest single hit;
   - interrupts dealt, failed, and taken;
   - guards used;
   - status effects applied;
   - kills;
   - how each enemy ended: slain, spared, fled, or yielded;
   - the company's health at the end.

   It is a setting (on by default).
5. **Per-fight totals** are kept in memory until the summary is sent. They
   are not persisted. A copyover mid-fight loses the summary, not the
   fight.
   - Open: whether that's acceptable, or the totals ride on the company
     record.

## Reference text

```
── The road falls quiet ──
Damage dealt   Company 96 · Enemies 38
Healing        Company 5 (1 held back by a wound)
Most damage    You 23 · Garrick Vane 21 · Ysolde 18
Highest hit    Ironhide 11 on Tamsin Reed (critical)
Interrupts     dealt 2 (Withering Hex, Crushing Blow) · failed 1 · taken 0
Guards         Tamsin Reed 1 of 2 used · lost when knocked down
Effects        bleeding 2 · staggered 2 · knocked down 2
Enemies        Ironhide slain · goblin hexer slain · first skirmisher spared · second skirmisher fled
Company        You 12/14 · Garrick 6/15 · Tamsin 7/16 (wounded, limit 10) · Oswin 9/12 · Ysolde 9/11
```

## Constraints

- Events are emitted on the game loop. Listeners must not block it.
- The stream never advances the clock and is never the source of truth for
  health or state. It only reports.

## Acceptance criteria

- A wiring test drives a 5v5 through the real round and asserts:
  - the event sequence (fight start, attacks, a death, fight end) with
    correct ids;
  - a summary whose totals equal the sum of the attack events.
- Existing combat text is unchanged by this phase (golden lines from the
  27c fight still pass).
