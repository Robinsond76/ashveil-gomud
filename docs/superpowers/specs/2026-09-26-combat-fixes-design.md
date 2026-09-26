# Potential Phase 28a: Combat Fixes from the 5v5 Simulation

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: findings recorded for a future phase; not scheduled.

## Source

A 5v5 fight run through the real combat round (see the roadmap) on
2026-09-26:
- **Company:** Aria (player, level 3), Tamsin Reed, Brother Oswin, Garrick
  Vane, and Ysolde, summoned through `company summon`.
- **Enemies:** a five-bandit party sharing one `groups` tag: two
  cutthroats, a bruiser, a slinger, and a captain.

It was run lit and unlit, with the bandits hostile and not hostile.

**Caveat:** the harness did not load `_datafiles/config.yaml`. Anything that
depends on configured values must be reproduced against the shipped config
before it is fixed. Each finding below says whether that applies.

## Findings

### F1. The leader can be stuck on an unreachable target (likely real)

- **What happened:** Aria stood in company column 3 and attacked the bandit
  captain, who stood in the enemy's column 1. Every round she got
  `You can't reach that target from here.` and never swung.
- **Why:** Companions are reassigned by 11b's `engagement.AssignTarget`
  when their target is lost, but the leader keeps an illegal target.
- **After the captain died:** her aggro cleared and she stopped fighting
  altogether, while her companions carried on.
- **Direction:** either reassign the leader to a legal target in the same
  enemy party (with a line saying so), or refuse the `attack` up front with
  who *can* be reached.
- **Needs a decision:** which of those two.

### F2. A fight can stall with enemies standing (reproduce)

- **What happened:** With non-hostile bandits, the group turned hostile
  when one of them was attacked, and the others joined through the idle
  loop's `lookfortrouble`. Group hostility wears off with time
  (`mobs.MakeHostile`, two minutes less the attacker's Perception). Mid-fight:
  - one companion went idle while a cutthroat was still fighting;
  - the last bandit slinger never engaged;
  - the fight ended silently, with the slinger standing next to the company.
- **Direction:** while any member of an enemy party is engaged with the
  company, keep the whole party engaged. Companions with no target take a
  legal one from the party still standing.
- **Reproduce first:** round timing and hostility duration come from config.

### F3. Level-1 companions with 1 HP (withdrawn)

- **What happened:** The harness showed Tamsin and Oswin (level 1) at 1/1 HP.
- **Why it's withdrawn:** The shipped config sets `HPBase: 5`, and the
  harness ran with 0. The code only falls back to 5 for a negative value.
  Live, they have about 6 HP or more.
- **Still worth doing:** have the phase's wiring test load the shipped
  progression values, so this can't recur unnoticed.

### F4. `formation reach` lists your own companions (UX)

- **What happened:** `formation reach me` answers `Aria could
  plain-melee-reach: Garrick Vane(#3)`.
- **Why:** This is by design. It demonstrates the reach rules against the
  company's own formation, the only formation live outside a fight. It still
  reads like a bug.
- **Direction:** in a fight, answer against the live enemy party. Outside
  one, say it's a demonstration.

### F5. Darkness penalty makes night fights mostly misses (intended; surface it)

- **What happened:** Unlit night fights were mostly misses. That is Phase
  14's darkness penalty working as intended.
- **Direction:** the narration opener (28b) can mention the dark when it
  applies, so the misses make sense.

### F6. Grouped enemies don't show in `look` in the harness (harness only)

- **What happened:** The harness rendered a markdown fallback instead of
  the `who` template.
- **Actual engine data:** the room details were correct:
  `Also here: a pack of 5 creatures, Tamsin Reed (♥friend), …`.
- **Status:** No game bug. It's noted so the next harness doesn't
  misreport it.

## Acceptance criteria (for when this is scheduled)

- A regression test through the real round for F1 and for F2 (after F2 is
  reproduced with shipped config).
- Wiring tests load the shipped progression config.
- `formation reach` in a fight names reachable enemies.
