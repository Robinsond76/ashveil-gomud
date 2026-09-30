# Potential Phase 30e: Morale and Mercy

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26). The [detailed design](2026-09-30-phase-30e-morale-mercy-design.md) is now proposed for review; its defaults are not yet approved.

## Owner decisions

- **Morale varies per enemy.** The undead never yield. Some bosses never
  break.
- **Yielding:** an enemy that yields (begs for mercy) leaves the fight at
  once and stands aside until the fight ends. No one attacks it, and it
  attacks no one.
- **The mercy prompt:** at the end of the fight, the player is asked, once
  for each enemy that yielded, whether to spare it.
- **No return:** spared enemies never come back. That is a future idea.

## Rules

1. **Enemy morale**, from the mob template or race:
   - **Breaking points:** their leader dies, their caster or healer dies,
     half the party has fallen, or they're badly hurt and alone.
   - **When one is hit:** each member rolls to hold, yield, or flee, by
     temperament.
   - **Never break:** the undead, constructs, some beasts (a mother
     defending young), and flagged bosses.
2. **Yield** means out of the fight, marked yielded, and waiting on the
   player's decision. **Flee** means the enemy leaves the room, and the
   fight counts it as fled.
3. **The mercy prompt**, at fight end (GoMud's `prompt` system, as used by
   picklock and the character creator):
   - The question: `The first skirmisher kneels in the mud, hands raised.
     Spare him? [yes/no]`.
   - **Spare:** the enemy leaves without its weapon, with no experience or
     loot.
   - **Kill:** the enemy is executed. Experience is given; loot drops.
4. **Consequences,** through the existing systems:
   - **Company alignment** (21a) moves: sparing nudges it up, and killing
     a yielded foe nudges it down.
   - **Companions react** by their own alignment, with a loyalty change
     and one spoken line. A cleric approves of mercy; a sellsword
     shrugs or scoffs.
5. **Unanswered prompts:** if the player logs out or the prompt is left,
   the enemy slips away (it is neither spared nor killed, so alignment
   doesn't move).
   - Open: the timeout length.
6. **Company nerve** (the company's own morale).
   - **Who hesitates:** a companion with low loyalty (21a) or weak chemistry
     (24) may hang back for a round, or flee a clearly losing fight.
   - **How much:** a small effect, narrated so the leader can see it
     coming.
   - **Durability:** a fled companion rejoins after the fight, with a
     loyalty loss. It is not a desertion unless loyalty reaches zero, as
     today.

## Reference text

```
With the hexer dead, the goblins falter. (morale breaking)
The first skirmisher throws down its knife and backs away, hands raised. (yields)
The second skirmisher bolts into the birches. (fled)
The skeleton's jaw hangs open, its sword arm hanging from a thread of sinew. It keeps coming. (the dead do not yield)
The ogre's eyes roll white with fury. (Ironhide will not break)

── The road falls quiet ──
...
The first skirmisher kneels in the mud, hands raised. Spare him? [yes/no]: yes
You lower your hand. The goblin stares at you, then flees for the trees without its knife.
    Brother Oswin nods slowly. (loyalty +2)
    Garrick Vane spits into the mud. "He'll be back with friends." (loyalty −1)
    Your company's alignment rises. (+1, now 42, virtuous)
```

## Acceptance criteria

- Through the real round, killing a party's leader:
  - makes members with a breaking temperament yield or flee;
  - leaves undead unaffected.
- A yielded enemy takes and deals no blows for the rest of the fight.
- At fight end, one prompt per yielded enemy. Spare and kill each apply
  their alignment, loyalty, experience, and loot effects.
- An unanswered prompt lets the enemy slip away with no alignment change.
- A low-loyalty companion's hesitation is narrated and bounded.
