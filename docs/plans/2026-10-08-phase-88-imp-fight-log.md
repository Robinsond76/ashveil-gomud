# Phase 88: imp fight log fixes

Robinson's log of an imp fight showed five problems. Decisions and findings:

1. **Stale corpse text in a battle** ("A skeleton corpse crumbles to dust" in an
   imp fight). `Room.UpdateCorpses` told the whole room. Players in a battle are
   now left out of the message; the corpse still decays and others still hear it.
   `TestACrumblingCorpseIsNotToldToThoseInABattle`.
2. **Surrender prompt twice** (`... Spare them? [yes/no]` then `.: ... [yes/no]
   [yes/no] yes`). The question text carried its own `[yes/no]`, and the prompt
   line adds it too. The kneeling is now one line ("The second imp kneels, hands
   raised.") and the prompt asks only "Spare them?". `TestMoraleRoundYieldProtectionAndSpare`.
3. **"goes for the forest imp"**. Phase 87 already names the battle label; a test
   now covers imps after a sibling surrendered (`TestGoesForNamesAnImpAfterASurrender`).
   A bare name can still appear for a foe outside the battle's captured labels
   (not reproduced; no log to match).
4. **A yielded foe counted as slain?** No: a spared foe leaves through `vanish`
   (no kill, death event, reward). Executing it is a real kill. The bestiary
   line in the log came from the first imp's death. Pinned by
   `TestASparedFoeIsNotSlain`.
5. **Loot line**. "Battle loot from X is claimed" was said for every claimed body,
   even empty ones. It is now said only when the body holds items or gold.
   `TestNoLootLineForAnEmptyBody`.

## More battle log notes (same day)

6. **Sunset art mid-battle.** Sunrise and sunset broadcasts are ambient
   (`Broadcast.HoldInBattle`): a player in a battle is told when it ends.
   `TestAmbientBroadcastWaitsForTheBattleToEnd`.
7. **Kill then "turns toward" read as an extra turn.** The round's death notices
   come at its end, but a fighter's retarget line came right after the kill
   blow. The fall is now told first (`fallenFirst`, idempotent with the round's
   own notice). `TestAFallenTargetIsToldBeforeTheTurn`.
8. **"Your rage subsides."** It was the old line for a foe that fell with no
   other target, with no rage ever announced. Removed.
9. **Unfinished heal.** Already handled by phase 87 ("The fight is over, but
   you finish your Minor Heal."); no change.
10. **"Brother Oswin's acolyte's mace".** Combat lines now name the kind of
    weapon ("Brother Oswin's mace"), the last word of its name
    (`combat.WeaponNoun`). `TestWeaponNounIsTheKindOfWeapon`, `TestFightLogWording`.

## Review (2026-10-08)

Played live at the Fork at the Black Oak with Brother Oswin and Tamsin Reed
against three forest imps (`set combatpace off`). The kill now comes before the
turn ("The first imp crumples and does not rise." then "Tamsin Reed turns
toward the second imp."), weapons read "Brother Oswin's mace", "Your
broadsword", "Tamsin Reed's cudgel", and no stray rage or corpse lines
appeared. The level-1 company lost that fight, so no foe surrendered live; the
surrender prompt is covered by `TestMoraleRoundYieldProtectionAndSpare`.

Fixed:

- **Rolled weapons named "(rare)".** The last-word rule read a rolled weapon's
  display name ("Dawnfang, a keen iron shortsword (rare)") and named it
  "(rare)"; an item adjective gave "(glowing)". Combat lines now take the kind
  from the item's base name (`combat.WeaponNounOf`); relics and other
  capitalised names keep their name ("Ashen Reaper"). Parry lines and "You
  draw your ..." use it too. `TestWeaponNounOfAnItem`.
- **"Battle loot from first imp".** The loot line now carries the article:
  "Battle loot from the first imp is claimed by Torvald."
  (`TestNoLootLineForAnEmptyBody`).

Checked, no change: nothing in classes, skills or buffs grants rage, so "Your
rage subsides" hid no effect. Held sunrise and sunset lines go out at
`endBattle`, which every battle end passes through (including a player who
logs off), and they queue behind any paced battle lines still being shown.
