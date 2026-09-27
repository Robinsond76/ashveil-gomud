# Combat Presentation and Tactics — Spec Roadmap

**Status:** Direction approved by the project owner on 2026-09-26, after a
simulated 5v5 fight and several rounds of mock transcripts. These are feature
specifications for potential phases, not implementation plans. Each needs its
own design pass (prior-art check, open decisions, plan) under `CLAUDE.md`
before code. The owner chooses when they run.

## Intended player experience

A fight reads like a story and unfolds on screen at a readable pace. Blows
are described in a dark, physical voice; every hit still says what it did.
The company fights on its own, by a strategy the player set beforehand.
Clerics heal as needed and guardians guard. Heavy blows and spells are
telegraphed and can be broken. Crits leave marks: bleeding, stagger, and
wounds that outlast the fight. Enemies have temperaments. Some break and
beg for mercy, and the player decides their fate when the fight is over.
Afterwards, one command tends the company's hurts, and a summary shows how
the fight went.

## How this was worked out

1. A 5v5 fight was run through the real combat round in a throwaway harness
   (see the [combat fixes spec](2026-09-26-combat-fixes-design.md)).
2. The owner reviewed the real output and asked for a different voice.
3. Mock transcripts were iterated until they read right. The final mocks
   are the reference text in each spec.
4. The external design reference
   ([2026-09-22](2026-09-22-combat-design-reference-external.md)) was
   reviewed with the owner. Its layered ideas are adopted here; its
   continuous timeline is declined (see below).

## Phases

### Phase 28 — Combat presentation

| Phase | Spec | Depends on |
|---|---|---|
| 28a Combat fixes | [combat fixes](2026-09-26-combat-fixes-design.md) | Phase 11 wiring |
| 28b Combat event stream and battle summary | [event stream](2026-09-26-combat-event-stream-design.md) | — |
| 28c Narration voice (weapons and spells) | [narration](2026-09-26-combat-narration-design.md) | 28b |
| 28d Pronouns and ordinals | [pronouns and ordinals](2026-09-26-combat-pronouns-ordinals-design.md) | 28c |
| 28e Pain reactions | [pain reactions](2026-09-26-combat-pain-reactions-design.md) | 28c, 28d |
| 28f Paced combat output | [pacing](2026-09-26-combat-pacing-design.md) | 28b |

### Phase 29 — Tactical combat

| Phase | Spec | Depends on |
|---|---|---|
| 29a Status effects and critical-hit effects (weapons only) | [status and crit effects](2026-09-26-status-crit-effects-design.md) | 28b; 28c for text |
| 29b Wounds, treatment, and `heal wounds` | [wounds](2026-09-26-wounds-treatment-design.md) | 29a |
| 29c Pre-fight tactics: roles, personalities, guards, companion casting | [tactics](2026-09-26-company-tactics-design.md) | 28b, 29a |
| 29d Wind-ups, telegraphs, and interrupts | [interrupts](2026-09-26-telegraphs-interrupts-design.md) | 29a, 29c |
| 29e Morale and mercy | [morale and mercy](2026-09-26-morale-mercy-design.md) | 28b, 29c |
| 29f Battlefield conditions | [battlefield](2026-09-26-battlefield-conditions-design.md) | 29c |

### Phase 30 — Browser battle panel

| Phase | Spec | Depends on |
|---|---|---|
| 30 Battle panel | [battle panel](2026-09-26-battle-panel-design.md) | 28b, 28d |

### Build order (decided 2026-09-27)

1. **28a** — independent, fixes behaviour rather than text; a quick,
   low-risk first slice.
2. **28b** — the event stream is the foundation every later phase reports
   through, so it comes before any text or tactical work.
3. **28c → 28d → 28e** — the narration voice, then pronouns/ordinals, then
   pain reactions, in that order since each extends the one before.
4. **28f** — pacing, once the text it's pacing out is settled.
5. **29a → 29b** — status/crit effects, then wounds, which is built on
   them.
6. **29c** — pre-fight tactics (roles, personalities, guards, companion
   casting). This is the largest slice in Phase 29 and unlocks 29d–29f.
7. **29d → 29e → 29f** — telegraphs/interrupts, morale/mercy, and
   battlefield conditions, each building on 29c's tactics layer. Order
   among these three is flexible; interrupts first gives 29e's "breaking
   under pressure" more to draw on.
8. **30** — the battle panel, last. It only strictly needs 28b and 28d,
   but showing the grid is far more interesting once 29c's roles,
   targeting, and guards give it something to display.

This is Phase 28 (presentation) end to end, then Phase 29 (tactics) end to
end, then Phase 30. A phase is never started before the phases it depends
on (per the tables above) are complete and reviewed.

## Decisions carried forward (owner, 2026-09-26)

### Voice and text

- **Tone:** dark and physical. No exclamation marks, no ALL-CAPS, no `***`.
- **Mechanics in parentheses** at the end of a line, lowercase:
  - `(5 damage)` on every hit;
  - `(critical hit, 9 damage, bleeding)`;
  - `(4 healed)`;
  - `(winding up: Crushing Blow, 1 round)`, `(chanting: Minor Heal, 2 rounds)`;
  - `(interrupt pressure 3 of 5)`, `(Withering Hex interrupted)`;
  - `(wound limit 10 of 16)`.
- **Critical hits:** a non-lethal one is followed by the victim's pain
  reaction; a lethal one by the death line.
- **Names:** companions carry no charmed tag. Pronouns for people, "it" for
  beasts. Same-named enemies get ordinals fixed for the fight ("the first
  cutthroat").
- **Engagement lines:** no "prepares to fight". One opener per fight,
  "turns toward" on a new target, and one closing line.
- **Kept as they are:** waiting and aiming lines, every miss, and the
  chanting lines each round.
- **Death notices:** companion death notices are indented.

### Timing and pacing

- **Rounds, not beats.** Combat stays round-based. Wind-ups and cast times
  are measured in whole combat rounds.
- **An 8-second combat round (decided 2026-09-27).** `DoCombat` resolves
  every second game round (`CombatEveryRounds: 2`); every other
  round-based system (survival, drift, chemistry, buffs, autosave) keeps
  the existing 4-second game round unchanged. See
  [28f](2026-09-26-combat-pacing-design.md).
- **Paced output:** each combat round's lines are paced out on screen over
  about 6 of its 8 seconds, with a fast, normal, or slow setting (and
  off, for screen readers or by preference).

### Command and healing

- **No commands mid-fight.** The company acts on its own by a strategy set
  before the fight (`company tactics`).
- **Healing in a fight:** clerics heal as needed on their own.
- **Healing after a fight:** one command, `heal wounds`, uses whoever and
  whatever can help:
  - a cleric;
  - bandages and splints;
  - an inn physician.

  A full camp rest also heals wounds.

### Morale and mercy

- Morale varies per enemy. The undead never yield, and some bosses never
  break.
- An enemy that yields leaves the fight at once and stands aside until the
  fight ends. The player is then asked, for each one, whether to spare it.
- Spared enemies never come back.

## Declined or tabled

- **The continuous Readiness/Wind-up timeline and sub-round "beats"** (the
  external reference's §§4–11, 16, 25): declined by the owner on
  2026-09-26. Combat stays round-based.
- **Manual orders mid-fight** (`company protect`, `company order … bash`, and
  the like): tabled. Tactics are set before the fight.
- **Spared enemies returning later:** a future idea, not planned.
- **New core stats** (DEX, AGI, a physical/magic defense split): not
  adopted. New values are derived from GoMud's existing stats.
- **Spell critical hits:** declined by the owner on 2026-09-27. Spells
  keep a single damage roll with no crit tier, no weapon-style secondary
  effect, and no 28e pain reaction. [29a](2026-09-26-status-crit-effects-design.md)
  reflects this: its crit effect table covers weapon subtypes only.

## Integration and review gate

Each slice follows `CLAUDE.md`'s testing and review gate:
- Wiring tests go through the real combat round, as in the 27c tutorial
  fight and the harness in the fixes spec.
- They assert on emitted events and rendered lines, not only on tables.
- They load the shipped config, so that progression values like `HPBase`
  apply.
- The owner-facing review compares the same seeded fight before and after.
- `docs/PROJECT_STATUS.md` records each slice.
