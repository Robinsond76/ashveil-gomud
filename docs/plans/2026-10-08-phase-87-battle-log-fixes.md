# Phase 87: battle log fixes and a slower pace (2026-10-08)

Robinson pasted a summary of one battle (14 rounds, Wizard, Cleric, Ranger and
a skeleton group) and asked for what could be done about it, plus: make today's
`slow` combat pace the normal one, and make `slow` slower. Full autonomy: each
decision below has its reason. Nothing here changes a damage or healing number.

## Pace

- Normal is what slow was: a beat of 1.5 seconds (follow 0.3, extra 0.7, tail
  0.8; by line: gap 1.0, dramatic 1.8, quick 0.3, window 7.5).
- Slow is 1.5 times that: a beat of 2.25 seconds (every figure x1.5).
- Fast (0.6 second beat) and off are unchanged.
- The web client is unchanged on purpose: it plays each blow's animation
  inside the beat (normal 1200 ms, slow 1800 ms against beats of 1.5 and 2.25
  seconds), the server sends the real setting, and the labels it shows are the
  pace names, not seconds.
- `help combatpace` and `help combat` name the new speeds. The pacer tests that
  pinned the old Normal numbers now use a local fixed spec, and a new test pins
  the three paces.
- Why: the fight reads faster than people read; slow was already the pace
  Robinson played at.

## 1. Ranger aimed shots

Finding: an Aimed Shot is readied in the ability pass at the start of the
round; the shot itself comes on the ranger's turn. When an earlier turn in the
same round killed the target, the real reassignment ran
(`reassignCompanionTarget`, `reassignPlayerTarget`) but rebuilt the aim as a
plain attack. The announcement ("takes careful aim") stood, the readied shot
and its bonus were dropped, and the cooldown stayed spent. That is the round-11
case.

Fixes:
- The retarget keeps a readied Opening Strike or Aimed Shot and its bonus
  (`retargetKeepingStrike`): the shot turns on another living foe it can reach.
- A readied Aimed Shot that is never loosed (no foe in reach, turn lost) costs
  no wait and says "Your aim finds no target; the shot is held back."
- The edge the text claims is unchanged and was already true: the first shot
  that lands is a critical hit with +2 +1 per 3 levels; it still has to hit
  (`help abilities` says so). The ranger's other misses are ordinary misses.
- "Aimed Shot 3" in the summary counts the ability's uses (announcements). With
  the two fixes above every use now ends in a shot or an explicit "held back"
  line, so the count means shots readied and loosed. Not changed further.

## 2. Timing text

- Chant countdown: the count was in "rounds" but runs on the caster's turns. A
  chant started in the strategy pass spends the round's own turn at once, and a
  quick caster takes two turns in a round, so "3 rounds" then "1 round" under
  one round header was the true count of turns. The word is now "turn(s)"
  everywhere a chant is counted (all spell scripts, restarted chants, help).
  Behaviour is unchanged.
- Knockdown: a tackle now says "(knocked down: its next action is lost)". The
  buff lands after the ability pass, so a foe that acts later the same round is
  not stopped; the next one is. The help page already said "next action".
- A heal that outlasts the fight: when a battle ends with a company member
  still chanting a helpful spell, the room is told "The fight is over, but X
  finishes Minor Heal." (harmful chants are wasted and say nothing new).

## 3. Narration

- A critical hit is never called glancing in its line; the damage still carries
  the quality's cut.
- Races marked `bloodless` (undead, golem, doll, orb, ghostly spirit, dummy) are
  struck with their own hit lines (`combat-messages/bloodless.yaml`, hits only,
  any weapon) and a crit never leaves them bleeding (also Hunt Down and talons).
  The world is temporary stock, so the lines are generic bone/cloth/dust.
- Training tip ("status train") never shows mid-battle.
- Numbered foes: target-change lines already use the battle's fixed labels. A
  foe that began alone keeps its plain name by the phase 29 label rule
  ("issued labels never change"), which is the likely source of the ambiguity;
  not reproduced without the log, and relabelling mid-battle would break that
  rule, so left as is. Follow-up if it recurs with a log.

## Wizard melee (55 of 95 damage)

A small expected-damage probe (`combat.ExpectedDamage` through the balance
mirror, each class with its shipped kit weapon, levels 5-30):

| level | wizard staff | witch staff | cleric mace | warrior |
|---|---|---|---|---|
| 5 | 4.3 | 4.3 | 4.3 | 5.4 |
| 10 | 3.4 | 3.4 | 3.4 | 5.4 |
| 20 | 1.7 | 1.7 | 1.7 | 6.5 |
| 30 | 0.7 | 0.7 | 0.7 | 8.6 |

The quarterstaff is a 1d6 weapon like the cleric's mace and the broadsword; every
caster's melee is identical and a fraction of a warrior's, falling with level.
One long fight where the Wizard ran out of mana and fought with the staff is not
a class problem. No change.

## Battle summary layout and exclamation marks (Robinson's second ask)

- The end-of-battle summary is grouped into three titled parts with a blank line
  between them: "The fight" (damage dealt, most damage, highest hit, kills,
  effects, moves, sigil), "What it cost" (healing, defenses, interrupts, guards,
  damage taken, never landed) and "How it ended" (spoils, enemies, company).
  A part with no lines is left out. The label column stays 15 characters wide
  so every value lines up. Lines keep their old text, so nothing that reads
  them changes. `help battle-summary` shows the new layout.
- Exclamation marks are gone from battle and reward text: experience gained
  (and its event-log twin), level reached (own log, level-up broadcast), skill
  level gained, quest gold and item rewards, "You attack the darkness", "is
  your friend", "is in your party". Alarms (bleeding out, kicked for
  inactivity) and non-reward flavour keep theirs.

## Help and tutorial

Pages updated: `combatpace`, `combat`, `narration` (turns left, bloodless),
`attack` (crit never glancing), `abilities` (Aimed Shot follows its target).
Indexed under the existing `narration` keyword (new aliases `bloodless`,
`skeleton`). The tutorial's `set combatpace` hint names no speeds and is still
right.

## Review (2026-10-08)

Independent review played three live fights (level 6 Warrior with a Ranger,
Wizard and Cleric against four skeletons) and read the full diff. Accepted and
fixed, each with a regression test:

- **A killing Aimed Shot read as never loosed.** The swing resolves on a copy of
  the fighter, so its live aim still read as readied after the kill; the new
  retarget carried the spent shot onto the next foe, and the turn's end then
  said "the shot finds no target" and refunded the wait (an Aimed Shot every
  round). `spendStrike` marks the strike loosed before the kill's retarget.
  `TestAKillingAimedShotIsSpent`.
- **Numbered foes.** The "goes for" line (a companion turning on a foe through
  the mob `attack` command) used the bare name: "Recruit Cleric goes for the
  skeleton". It now names the battle label ("goes for the third skeleton").
  `TestGoesForNamesANumberedFoe`.
- **Minor Heal's count skipped a number** (3 turns, then 1, then the heal): its
  script still said `WAIT_ROUNDS = 2` after heal.yaml went to 1. Fixed, and
  `TestShippedChantCountsMatchWaitRounds` pins every script to its spell file.
- **The summary broke apart at 360px.** Long values (most damage, company,
  enemies) were wrapped by the phone terminal mid-name and back under the
  labels. Rows now wrap at 48 columns between their parts, continuing under
  the value column (the phone shows 49 columns with "Smaller text" off).
  `TestSummaryWrapsLongValuesUnderTheirColumn`; `help battle-summary` updated.
- **Exclamation marks the build missed** in battle and reward text: ward of
  life, quest given/completed/progress, bleeding out, dropping to the ground
  (now capitalised), "You aren't in combat", "You are in combat".

Checked and accepted as built: Wizard melee (the quarterstaff is 1d6 like the
mace); bloodless skeleton hit lines; the tip after the summary; the "chant
finishing" line; knockdown wording.
