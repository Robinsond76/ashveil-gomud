# Combat Presentation and Feel — Spec Roadmap

**Status:** Direction approved by the project owner on 2026-09-26, after a
review of a simulated 5v5 fight and several rounds of mock transcripts. These
are feature specifications for potential phases, not implementation plans.
Each needs its own design pass (prior-art check, open decisions, plan) under
`CLAUDE.md` before code. The owner chooses when, and in what order, they run.

## Intended player experience

A fight reads like a story, not a log. Blows are described in a dark,
physical voice without exclamation marks; every hit still says how much it
hurt; critical hits are dramatic and marked, and a wounded foe reacts to
them. The company are companions, not charmed thralls. Enemies that share a
name can be told apart. Spellcasting has the same voice as steel. In the
browser, the two formations and who is fighting whom can be seen at a glance.

## How this was worked out

1. A 5v5 fight (a leader and four companions against a five-bandit party)
   was run through the real combat round in a throwaway test harness:
   `hooks.UserRoundTick`, `MobRoundTick`, `DoCombat`, `AutoHeal`,
   `IdleMobs`, and `HandleIdleMobs`, with the company, formation, and death
   modules loaded through `plugins.Load`. Its findings are in the
   [combat fixes spec](2026-09-26-combat-fixes-design.md).
2. The owner reviewed the real output and asked for a different voice.
3. Mock transcripts were iterated until they read right. The final mocks
   are the reference text in the
   [narration spec](2026-09-26-combat-narration-design.md).

## Specs and suggested order

| Order | Potential phase | Spec | Main dependency |
|---|---|---|---|
| 1 | 28a Combat fixes | [Combat fixes](2026-09-26-combat-fixes-design.md) | Phase 11 formation combat wiring |
| 2 | 28b Narration voice | [Combat narration](2026-09-26-combat-narration-design.md) | None; content plus small combat-loop changes |
| 3 | 28c Pronouns and ordinals | [Pronouns and ordinals](2026-09-26-combat-pronouns-ordinals-design.md) | 28b's message tokens |
| 4 | 28d Pain reactions | [Pain reactions](2026-09-26-combat-pain-reactions-design.md) | 28b's critical-hit path; 28c's pronouns |
| 5 | 28e Spell narration and combat casting | [Spells in combat](2026-09-26-spell-narration-combat-casting-design.md) | 28b voice; 28c pronouns |
| 6 | 29 Battle panel | [Battle panel](2026-09-26-battle-panel-design.md) | 26b Company GMCP; 28c ordinals for names |

28a is independent of the rest and fixes behaviour, not text; it can run
first or in parallel. 28b–28d change how every fight in the game reads
(players and mobs alike), so each needs a before/after transcript in its
review. 29 can start after 28c, since the grid labels should match the
narration's names.

## Decisions carried forward (owner, 2026-09-26)

- No exclamation marks in combat text. A darker, physical tone.
- Every hit shows its damage at the end of the line: `(5 damage)`. A
  critical hit ends `(critical hit, 9 damage)`. A heal ends `(4 healed)`.
- No `***` wrapping or ALL-CAPS on critical hits; the drama is in the words.
- A non-lethal critical hit is followed by a pain reaction from the victim.
  A lethal one is followed by the death line instead.
- Company members show no `(charmed)` / `(♥friend)` tag.
- No "prepares to fight" lines. One opener when a fight starts; after that,
  only a short "turns toward" line when a fighter picks a new target.
- Waiting/aiming lines and every miss stay.
- Companion death notices are indented under the action.
- Pronouns for people; beasts use "it".
- Same-named enemies are told apart by ordinals ("the first cutthroat",
  "the second cutthroat"), fixed for the whole fight.
- Pain reactions come in a set per beast race, with a generic fallback.
- Spell chanting lines stay each round while a spell builds.

## Integration and review gate

Each slice follows `CLAUDE.md`'s testing and review gate. For the text
phases, wiring tests go through the real combat round (the 27c tutorial
fight and the harness described above are the pattern) and assert on the
rendered lines, not only the message tables. The owner-facing review compares
the same fight before and after. `docs/PROJECT_STATUS.md` records each
slice.
