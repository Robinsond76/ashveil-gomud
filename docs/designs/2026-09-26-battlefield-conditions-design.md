# Potential Phase 30f: Battlefield Conditions

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26), from the "new combat
suggestions" list. The owner approved the
[detailed design](2026-10-02-phase-30f-battlefield-design.md) on 2026-10-02;
implementation is in [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12),
**awaiting review by Opus 5.5**, unmerged. This file preserves the original proposal.

## Scope

1. **Ambush and surprise.**
   - **Who spots it:** when a fight starts from a travel interruption
     (12c) or at camp (7), the company's best Perception is weighed against
     the enemy's stealth, light (14), and terrain (5).
   - **The outcome:** no one surprised; the enemy gets a free round; or the
     company does.
   - **Camp watch:** a posted watch at camp helps.

   ```
   Ysolde stops dead and lifts a hand. Something large is moving in the birches ahead.
       Ysolde spotted the ambush. No one is caught off guard.
   ```

2. **Area attacks and clusters.** Area spells and sweeping blows hit
   adjacent cells in a formation, so bunching up is a risk. `sparks` hits a
   cluster, not the whole room. Ogres sweep the front row.
3. **Leaping and flanking.**
   - **Leaping:** some enemies (skirmishers, beasts) can leap the front row
     to reach the middle or back, especially over a knocked-down front-liner.
   - **Flanking:** a broken or thin line exposes the back row to melee.
4. **Narrow ground.** Rooms tagged `narrow` (a bridge, a gorge bend) fit
   only two columns, so both formations fold to fit, and some members
   can't reach.
5. **Fatigue and cold in combat.**
   - **Fatigue:** high fatigue (4) lowers hit.
   - **Cold:** exposure (15) numbs hands, slowing casting and slings.
   - **Narration:** lines say so ("Ysolde's numb fingers fumble the
     stone.").

## Acceptance criteria

- **Ambush:** a seeded ambush with a failed spot gives the enemy one free
  round. A spotted one doesn't.
- **Area attacks:** a cluster spell hits only adjacent cells.
- **Leaping:** a leaper reaches the middle row over a knocked-down
  front-liner.
- **Narrow ground:** a `narrow` room folds both formations to two columns.
- **Fatigue and cold:** their penalties show in hit rolls and in the text.
