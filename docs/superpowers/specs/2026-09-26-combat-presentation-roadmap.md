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

### Phase 29 — Combat presentation

| Phase | Spec | Depends on |
|---|---|---|
| 29a Combat fixes | [combat fixes](2026-09-26-combat-fixes-design.md) | Phase 11 wiring |
| 29b Combat event stream and battle summary | [event stream](2026-09-26-combat-event-stream-design.md) | — |
| 29c Narration voice (weapons and spells) | [narration](2026-09-26-combat-narration-design.md) | 29b |
| 29d Pronouns and ordinals | [pronouns and ordinals](2026-09-26-combat-pronouns-ordinals-design.md) | 29c |
| 29e Pain reactions | [pain reactions](2026-09-26-combat-pain-reactions-design.md) | 29c, 29d |
| 29f Paced combat output | [pacing](2026-09-26-combat-pacing-design.md) | 29b |

### Phase 30 — Tactical combat

| Phase | Spec | Depends on |
|---|---|---|
| 30a Status effects and critical-hit effects (weapons only) | [status and crit effects](2026-09-26-status-crit-effects-design.md) | 29b; 29c for text |
| 30b Wounds, treatment, and `heal wounds` | [wounds](2026-09-26-wounds-treatment-design.md) | 30a |
| 30c Pre-fight tactics: roles, personalities, guards, companion casting | [tactics](2026-09-26-company-tactics-design.md) | 29b, 30a |
| 30d Wind-ups, telegraphs, and interrupts | [interrupts](2026-09-26-telegraphs-interrupts-design.md) | 30a, 30c |
| 30e Morale and mercy | [morale and mercy](2026-09-26-morale-mercy-design.md) | 29b, 30c |
| 30f Battlefield conditions | [battlefield](2026-09-26-battlefield-conditions-design.md) | 30c |

### Phase 31 — Browser battle panel

| Phase | Spec | Depends on |
|---|---|---|
| 31 Battle panel | [battle panel](2026-09-26-battle-panel-design.md) | 30b, 30d |

### Build order (decided 2026-09-27)

1. **29a** — independent, fixes behaviour rather than text; a quick,
   low-risk first slice.
2. **29b** — the event stream is the foundation every later phase reports
   through, so it comes before any text or tactical work.
3. **29c → 29d → 29e** — the narration voice, then pronouns/ordinals, then
   pain reactions, in that order since each extends the one before.
4. **29f** — pacing, once the text it's pacing out is settled.
5. **30a → 30b** — status/crit effects, then wounds, which is built on
   them.
6. **30c** — pre-fight tactics (roles, personalities, guards, companion
   casting). This is the largest slice in Phase 30 and unlocks 30d–30f.
7. **30d → 30e → 30f** — telegraphs/interrupts, morale/mercy, and
   battlefield conditions, each building on 30c's tactics layer. Order
   among these three is flexible; interrupts first gives 30e's "breaking
   under pressure" more to draw on.
8. **31** — the battle panel, last. It only strictly needs 29b and 29d,
   but showing the grid is far more interesting once 30c's roles,
   targeting, and guards give it something to display.

This is Phase 29 (presentation) end to end, then Phase 30 (tactics) end to
end, then Phase 31. A phase is never started before the phases it depends
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
  [29f](2026-09-26-combat-pacing-design.md).
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
  effect, and no 29e pain reaction. [30a](2026-09-26-status-crit-effects-design.md)
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
