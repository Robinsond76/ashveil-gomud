# Ashveil Combat Design — Prototype Notes

> **Intake note (2026-09-22):** Imported verbatim from an external design doc
> (`~/Downloads/ASHVEIL_COMBAT_DESIGN.md`) for reference. Not an approved
> Ashveil design doc and not owner-vetted section by section.
>
> - The **formation targeting, Guard Reactions, weapon-flavored crit effects,
>   Wounds, and AI target-selection-personality** ideas (§§12, 13, 15, 18, 19,
>   20) are low-risk additions that layer onto GoMud's existing round-based
>   `internal/combat` resolution loop and are folded into Phase 11 scope — see
>   handoff §36.
> - The **continuous Readiness/Wind-up/Cast/Recovery timeline model** (§§4–11,
>   14, 17, 25–26) is a different combat engine, not a layer on top of the
>   current one: GoMud resolves combat one attack per round per combatant
>   (`internal/combat/combat.go`), with no readiness meter, wind-up, or
>   timestamped event scheduling. Adopting it would replace that resolution
>   loop, which crosses the concurrency/timer-rewrite threshold this project
>   escalates past Luna and requires strong dedicated tests. **Deliberately
>   deferred** — revisit as its own future design doc/phase, only after
>   Phase 11's lighter formation-combat layer has shipped and the owner
>   decides whether Ashveil combat should move off round-based resolution at
>   all.

**Status:** Early combat prototype / design direction  
**Reference inspiration:** *Ogre Battle 64* — small party formations, automatic combat, character roles, and speed/initiative-driven action order. Ashveil should use these ideas as inspiration rather than copy its formulas.

---

## 1. Combat Vision

Ashveil combat should feel like a battle unfolding continuously rather than a rigid turn queue.

The core goals are:

- Parties consist of multiple characters with distinct classes, equipment, stats, roles, and AI behavior.
- Combat proceeds automatically once engaged, with characters acting according to their speed, abilities, current battlefield state, and behavioral priorities.
- **Speed matters**, but should not dominate every other stat.
- **Equipment weight affects speed**, creating meaningful tradeoffs between protection and activity rate.
- Abilities should have their own **wind-up / casting time** and **recovery time** so that character speed is not the only factor determining attack frequency.
- Heavy attacks and powerful magic should feel dangerous because of their payoff, not because their users simply have extremely low initiative.
- Actions can overlap. A wizard may be chanting while an ogre winds up a club swing, an archer fires, and a defender attempts an interception.
- Formations, reactions, interrupts, healing, wounds, status effects, and target-selection personalities should create emergent combat stories.
- The combat log should be flavorful and readable, with clear damage/healing numbers and a damage counter / battle summary.

---

## 2. Initial Five-Person Player Party

The first prototype party contained five archetypes.

| Character | Class | HP | STR | VIT | INT | AGI | DEX | Weight | Initial Effective Speed |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Kael Varyn | Swordsman | 185 | 76 | 58 | 22 | 72 | 70 | 24 kg | 61.0 |
| Lyra Thorn | Archer | 145 | 57 | 40 | 31 | 91 | 88 | 13 kg | 82.9 |
| Orin Vale | Wizard | 112 | 18 | 28 | 94 | 68 | 61 | 9 kg | 63.7 |
| Seraphine | Cleric | 152 | 32 | 52 | 79 | 60 | 58 | 18 kg | 52.9 |
| Brom Ironward | Shield Warrior | 245 | 64 | 91 | 18 | 48 | 48 | 47 kg | 35.5 |

### Intended class identities

**Swordsman — Kael Varyn**
- Flexible frontline damage dealer.
- Good balance of speed, damage, and defense.
- Can parry.
- Behavior should favor dangerous frontline enemies or opponents actively engaging him.

**Archer — Lyra Thorn**
- Fastest player combatant.
- Low-to-medium durability.
- Frequent ranged attacks.
- Prefers wounded, exposed, lightly armored, or priority targets.
- Good example of a lightweight build receiving more actions without necessarily doing the highest damage per hit.

**Wizard — Orin Vale**
- Low durability.
- Extremely high magical damage per completed spell.
- Should **not** attack at normal weapon cadence.
- Powerful spells require chanting / charging.
- Vulnerable to interruption while casting.
- Represents deliberate burst magic.

**Cleric — Seraphine**
- Support / healer with moderate defenses.
- Healing should have meaningful casting time rather than being instant and endlessly efficient.
- Can use holy attacks when healing is not necessary.
- Healing priority should respond dynamically to ally health and battlefield danger.

**Shield Warrior — Brom Ironward**
- Very high HP and VIT.
- Heavy armor and shield impose a noticeable speed penalty.
- Protects the party through blocks and intercepts.
- Defensive reactions should be limited by a reaction / guard resource rather than occurring infinitely between normal actions.

---

## 3. Initial Enemy Party

The first randomized enemy group was the **Blackwood Pack**.

| Monster | Type | HP | STR | VIT | INT | AGI | DEX | Weight | Initial Effective Speed |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Skitter | Dire Wolf | 128 | 51 | 37 | 8 | 103 | 91 | 8 kg | 97.2 |
| Mirefang | Bog Viper | 96 | 43 | 25 | 14 | 96 | 96 | 5 kg | 92.5 |
| Grusk | Hobgoblin Raider | 164 | 67 | 55 | 20 | 69 | 65 | 21 kg | 59.6 |
| Hexeye | Goblin Hexer | 108 | 20 | 29 | 82 | 74 | 67 | 7 kg | 70.3 |
| Ironhide | Ogre Brute | 315 | 96 | 84 | 9 | 39 | 31 | 61 kg | 26.8 |

### Intended monster identities

**Dire Wolf — Skitter**
- Very fast.
- Seeks vulnerable or wounded targets.
- Can pressure the back line.

**Bog Viper — Mirefang**
- Fast, low direct damage.
- Applies Venom / damage over time.

**Hobgoblin Raider — Grusk**
- Conventional frontline bruiser.
- Moderate speed and damage.
- May counterattack.

**Goblin Hexer — Hexeye**
- Ranged magical attacker / debuffer.
- Prefers fragile casters or otherwise vulnerable targets.

**Ogre Brute — Ironhide**
- Huge HP and physical damage.
- Heavy and slower than most combatants, but the first prototype made him **too slow to participate meaningfully**.
- Should act more often than in Prototype 1, while individual powerful attacks should carry longer wind-up and recovery times.

---

## 4. Prototype 1 Speed Formula

The initial test used:

> **Effective Speed = AGI × (100 / (100 + Weight × 0.75))**

Characters filled an action meter toward 100 according to Effective Speed. When the meter reached 100, they acted and the meter reset.

This allowed faster characters to **lap** slower characters rather than simply determining one fixed action order per round.

### What worked

- Lightweight characters genuinely felt faster.
- Equipment weight immediately mattered.
- Combat did not feel like a static round-robin turn system.
- Fast units could pressure slow units before they acted.
- The system naturally generated interesting differences between heavily armored defenders and agile attackers.

### What did not work

The speed curve was too punishing at the extremes.

For example:

- Skitter: 97.2 Speed
- Ironhide: 26.8 Speed

This allowed a fast character to receive roughly 3–4 actions during the slowest character's action window.

That is probably too large a disparity for ordinary combat.

**Target direction:** extreme fast-vs-slow differences should likely be closer to approximately **1.8–2.2 actions to 1**, not 4-to-1, except when a special buff, debuff, stun, haste, or unique creature mechanic intentionally creates a larger gap.

Weight should continue to matter, but the penalty should probably use a gentler curve / diminishing returns.

---

## 5. Key Lesson From Prototype 1: Action Economy Was Too Powerful

Orin the Wizard dominated the first simulation.

He kept his high spell damage but acted at roughly ordinary-character speed. Against Ironhide, this caused Orin to land approximately four major spells before the ogre's second attack could occur.

This created the wrong wizard fantasy.

The desired fantasy is:

> **A wizard should attack less often, but completed spells should be frightening.**

The answer is **not** necessarily to reduce wizard spell damage.

Instead, powerful magic should require **casting / chanting / charging time**.

---

## 6. Combat Prototype v2 — Core Timing Model

Prototype v2 separates **initiative** from **ability execution**.

The authoritative action lifecycle is:

> **Readiness fills → Ready → Wind-up / Cast → Effect → Recovery → Readiness fills again**

This timing sequence should be used consistently across weapons, spells, healing, monster attacks, and other active abilities.

Key rules:

- **Speed** determines how quickly a combatant becomes Ready.
- Speed does **not** directly determine how quickly an ability resolves.
- Once an action begins, Readiness stops filling.
- **Wind-up / Cast Time** determines how long the action takes to produce its effect.
- **Recovery** determines how long the combatant remains committed after the effect.
- Readiness begins filling again only after Recovery ends.
- Eligible defensive **Guard Reactions** may still occur while a combatant is in Wind-up, Cast, or Recovery if the specific reaction permits it.
- Multiple combatants may be in different phases simultaneously; combat is resolved as a timeline of timestamped events rather than discrete party-wide turns.

This separation keeps Speed, ability power, execution time, and commitment as independent balancing levers.

---

## 7. Initiative / Readiness

Combat runs continuously.

Each combatant has a **Readiness Meter** from 0–100.

**Speed** determines how quickly Readiness fills.

Possible Speed influences include:

- AGI
- Equipment weight
- Buffs / debuffs
- Fatigue
- Injuries
- Terrain

Equipment weight should reduce Speed using a **gentler diminishing-return curve** than Prototype 1.

For ordinary unmodified combatants, the desired natural extreme between very fast and very slow units is approximately **1.8–2.2 readiness cycles to 1**, rather than the roughly 4:1 disparity seen in Prototype 1.

Larger disparities may still occur deliberately through haste, slow, stun, severe wounds, exceptional creatures, or unique equipment and abilities.

When Readiness reaches 100, the combatant selects an action according to player orders or AI behavior and immediately enters that action's Wind-up / Cast phase.

---

## 8. Wind-Up and Cast Time

Once an action begins, its effect does not necessarily happen immediately.

Physical actions generally use **Wind-Up**. Magical and devotional actions generally use **Cast Time** or **Chant Time**.

Examples:

- Quick arrow shot: very short Wind-up
- Sword slash: short Wind-up
- Heavy overhead strike: moderate Wind-up
- Ogre Crushing Blow: long, telegraphed Wind-up
- Arcane Lance: long Wizard chant
- Greater healing spell: meaningful Cleric Cast Time
- Resurrection or major ritual magic: very long Cast Time

During Wind-up or Cast Time:

- The combatant is committed to the action.
- Other combatants continue acting.
- The action may be exposed to interruption if its rules allow it.
- A telegraphed action can create opportunities for blocks, interrupts, target changes, or other reactions.

Fast attacks should generally have shorter Wind-up and lower commitment. Heavy attacks should generally have longer Wind-up, stronger payoff, and greater counterplay.

---

## 9. Recovery

After an ability produces its effect, the combatant enters **Recovery**.

During Recovery:

- The combatant cannot begin another normal action.
- Readiness does not refill.
- Eligible defensive reactions may still occur if allowed by the reaction.
- The character remains vulnerable to enemy pressure.

When Recovery ends, the Readiness Meter resumes filling from 0.

Recovery is independent from both Speed and Wind-up.

Examples:

- Dagger strike: very short Recovery
- Quick Shot: short Recovery
- Sword attack: moderate Recovery
- Heavy hammer attack: long Recovery
- Ogre Crushing Blow: long Recovery
- Major Wizard spell: significant Recovery

This allows designs such as fast-to-start/long-to-recover, slow-to-execute/quick-to-recover, devastating slow actions, and rapid low-power attacks.

---

## 10. Example Ability Timing

These values remain balancing placeholders.

| Ability | Wind-up / Cast | Recovery | Intended Feel |
|---|---:|---:|---|
| Lyra — Quick Shot | 0.25 s | 0.8 s | Rapid sustained ranged pressure |
| Kael — Sword Slash | 0.4 s | 1.0 s | Responsive melee attack |
| Orin — Arcane Lance | 1.8 s | 1.4 s | Slow but devastating magic |
| Seraphine — Mend Flesh | 1.2 s | 1.1 s | Responsive but interruptible healing |
| Seraphine — Greater Mend | 1.8 s | 1.5 s | Large heal with meaningful commitment |
| Ironhide — Crushing Blow | 1.5 s | 1.7 s | Telegraphable, dangerous heavy attack |

Exact values may eventually be implemented as server ticks rather than literal seconds. The relative relationships are more important than the unit.

---

## 11. Wizard, Sorcerer, and Caster Identity

Prototype 1 showed that Orin's problem was primarily **action economy**, not necessarily excessive spell damage.

The desired Wizard fantasy is:

> **A Wizard attacks less often, but a completed major spell should be frightening.**

### Wizard

- Long chants / Cast Times
- Very high damage per completed major spell
- Significant Recovery
- Vulnerable to interruption
- Strong tactical target selection
- May develop Concentration or other anti-interrupt tools

Example: **Arcane Lance** — Cast 1.8 s, Recovery 1.4 s, very high magical damage, interruptible.

### Sorcerer

- Short casts or near-instant magic
- Lower damage per cast
- Shorter Recovery
- Greater mobility
- Higher sustained spell frequency

| Class | Casting Speed | Damage per Cast | Cadence | Identity |
|---|---|---|---|---|
| Wizard | Slow | Very High | Low | Huge deliberate spells |
| Sorcerer | Fast | Moderate | High | Sustained magical pressure |
| Cleric | Medium | Moderate | Medium | Healing, support, holy magic |
| Warlock | Medium | High over time | Medium | Curses, DoTs, debuffs |

INT should represent magical aptitude without forcing all INT-based classes into the same combat rhythm.

---

## 12. Interrupts, Stagger, and Casting Risk

Actions with Wind-up or Cast Time may be interruptible.

Potential interrupt sources include Stagger, Stun, Knockdown, Silence, forced movement, Shield Bash, heavy critical hits, and specialized anti-caster abilities.

Each interruptible action may define an **Interrupt Threshold**. Interrupt effects apply pressure against that threshold. When the threshold is exceeded, the action is disrupted.

Depending on the ability, interruption may:

- Cancel the action completely
- Remove partial Cast / Wind-up progress
- Delay the action rather than fully cancel it
- Consume part of the Mana / Stamina cost
- Trigger a shortened Recovery penalty

Characters or abilities may possess **Concentration / Stability**, increasing resistance to interruption.

Light incidental damage should not automatically cancel a Wizard spell. Interrupting a major action should normally require a meaningful interrupt effect or sufficient accumulated pressure.

---

## 13. Guard Reactions

Defensive characters use a limited **Guard Reaction** resource separate from normal initiative actions.

Possible Guard Reactions include Block, Intercept, Protect Ally, Shield Deflection, and Counterguard.

Example: **Brom Ironward — Maximum Guard Reactions: 3**.

> Brom blocks Skitter.  
> **Guard: 2 / 3**

> Brom intercepts Mirefang.  
> **Guard: 1 / 3**

> Brom blocks Grusk.  
> **Guard: 0 / 3 — GUARD EXHAUSTED**

Further attacks may bypass Brom until his Guard resource refreshes.

**Default Prototype v2 rule:** Guard Reactions refresh when the defender completes a normal action cycle — after that action's Recovery ends — unless an ability explicitly states otherwise.

This prevents unlimited interceptions while preserving the fantasy of a shield user defending allies between normal attacks.

Potential Guard-related influences include shield quality, shield weight, DEX, AGI, Guard skill, Stamina, formation / facing, and status effects.

---

## 14. Healing

Healing follows the same timing model as every other active ability:

> **Ready → Cast → Heal → Recovery**

Healing is therefore powerful but not instantaneous.

A healer can be interrupted while casting, pressured into defensive choices, forced to choose between healing/support/offense, and unable to immediately erase every incoming hit.

Example abilities:

**Mend Flesh** — Cast 1.2 s, Recovery 1.1 s, moderate healing.

**Greater Mend** — Cast 1.8 s, Recovery 1.5 s, large healing.

Healing numbers should be tuned alongside incoming damage, Cast Time, and Recovery rather than balanced only through raw healing magnitude.

---

## 15. Wounds and Healing Limits

Some incoming damage creates **Wounds**.

Wounds reduce the target's temporary maximum recoverable HP during combat.

Example:

- Kael Max HP: 185
- Current HP: 40
- Current Wound Cap: 150

Normal healing can restore Kael only to **150 / 185 HP** until his Wounds are treated or the encounter ends.

Wounds may be caused more frequently by critical hits, crushing attacks, bleeding, heavy weapons, and powerful monster abilities.

Normal healing restores HP but does not automatically remove Wounds. Dedicated abilities, medical treatment, consumables, rest, or post-combat systems may explicitly treat Wounds.

This preserves the value of healing while ensuring sustained enemy damage creates meaningful attrition.

---

## 16. Overlapping Actions

Actions occur simultaneously on the combat timeline.

Example:

> **0.0 s** — Ironhide begins Crushing Blow.  
> **0.2 s** — Orin begins chanting Arcane Lance.  
> **0.5 s** — Skitter lunges toward Orin.  
> **0.6 s** — Brom spends a Guard Reaction to intercept.  
> **0.8 s** — Lyra fires Quick Shot.  
> **1.1 s** — Skitter is Staggered.  
> **1.5 s** — Ironhide's Crushing Blow lands.  
> **2.0 s** — Orin completes Arcane Lance.

The combat simulator should resolve actions as timestamped events rather than discrete turns.

This overlapping structure is a central part of the intended Ashveil combat feel.

---

## 17. Ogre / Heavy Enemy Design

Prototype 1 made Ironhide too slow to participate meaningfully.

Heavy enemies should not necessarily have extremely poor initiative. Instead, much of their perceived slowness should exist in their **ability Wind-up and Recovery**.

For an ogre-like enemy:

- Initiative Speed should be substantially higher than Ironhide's Prototype 1 value of 26.8.
- A provisional target around **45–50 on the Prototype 1 scale** is reasonable for future testing.
- Basic actions can remain usable at a reasonable cadence.
- Signature attacks should have long, readable Wind-up and meaningful Recovery.

Example:

> **Ironhide raises his massive club.**  
> *Crushing Blow — winding up...*

During the Wind-up, Brom may brace, Kael may attempt an interrupt, Lyra may attack, and Orin may gamble on completing a spell first.

Then:

> **CRASH!**  
> Ironhide's club slams into Brom.  
> **74 DAMAGE — STAGGERED**

The desired fantasy is not "the ogre waits forever." It is "the ogre is visibly preparing something terrible."

---

## 18. Target Selection and Combat Personality

AI should be competent but **not mathematically perfect**.

If every combatant always chooses the globally optimal target, fights can feel like spreadsheets rather than characters making decisions.

Each class / monster should have behavioral tendencies.

### Example player behaviors

**Kael — Duelist**
- Favors enemies engaging him.
- Favors dangerous frontline opponents.

**Lyra — Hunter**
- Favors wounded targets.
- Favors exposed targets.
- Favors enemies with low armor.

**Orin — Tactician**
- Favors armored targets vulnerable to magic.
- Favors groups when using AoE.
- May choose high-threat enemies.

**Brom — Guardian**
- Prioritizes enemies threatening allies.
- Uses protection abilities before maximizing personal damage.

### Example monster behaviors

**Wolf**
- Hunts wounded / isolated targets.

**Ogre**
- Generally attacks nearest/frontline targets.
- Less sophisticated targeting.

**Goblin Hexer**
- Favors vulnerable casters and support units.

**Wild beasts**
- Some randomness / instinct rather than perfect tactical selection.

This produces controlled chaos and stronger combat storytelling.

---

## 19. Critical Hits Should Do More Than Multiply Damage

Crits should interact with weapons and attack types.

Possible examples:

- **Sword critical:** Bleeding
- **Mace critical:** Stagger
- **Hammer critical:** Armor break / knockdown
- **Arrow critical:** Piercing / exposed weak point
- **Dagger critical:** Deep wound / high bleed
- **Magic critical:** Overload / elemental status
- **Shield critical:** Knockback or stun

A critical hit can still deal bonus damage, but its secondary effect should reinforce weapon identity.

---

## 20. Formation

Formation should matter similarly in spirit to Ogre Battle, while fitting Ashveil's continuous combat.

Possible basic formation concepts:

- Frontline characters are easier to target with melee attacks.
- Backline characters receive protection from frontline allies.
- Certain weapons / abilities have range requirements.
- Some enemies can bypass or leap over the frontline.
- Tanks can intercept attacks aimed behind them.
- Area attacks can hit clusters.
- Flanking or broken formation can expose backline units.

Initial player formation:

**Front:**
- Kael — Swordsman
- Brom — Shield Warrior

**Back:**
- Lyra — Archer
- Seraphine — Cleric
- Orin — Wizard

Formation behavior should eventually become part of party strategy rather than merely visual placement.

---

## 21. Stats Under Consideration

Initial prototype stats:

- **HP** — Health
- **STR** — Physical power
- **VIT** — Physical durability
- **INT** — Magical aptitude
- **AGI** — Initiative / movement speed contributor
- **DEX** — Accuracy / finesse / possibly reactions
- **Weight** — Equipment / carried combat weight
- **Speed** — Derived readiness rate
- **Physical Defense**
- **Magic Defense**

Potential future derived values:

- Accuracy
- Evasion
- Block chance
- Parry chance
- Critical chance
- Critical severity
- Interrupt power
- Concentration
- Guard reactions
- Stamina
- Mana
- Threat
- Movement / positioning speed
- Armor penetration
- Magic penetration

The system should avoid turning every concept into an independent stat unless it adds meaningful build choices.

---

## 22. First Simulation Summary

The first 5v5 simulation resulted in a player-party victory.

Survivors:

- Orin
- Seraphine
- Brom

Defeated player characters:

- Kael
- Lyra

All five monsters were defeated.

### Major finding

Orin was clearly overpowered **because of action economy**, not necessarily because the spell damage itself was inherently too high.

His Arcane Lance retained huge damage while his Speed allowed him to act at a cadence similar to ordinary attackers.

Against Ironhide, Orin attacked approximately four times while Ironhide managed only one attack.

### Agreed interpretation

- Keep devastating Wizard spell damage.
- Slow Wizards down through **chant / cast times and recovery**, not simply by destroying their base Speed.
- Add a faster, lower-damage **Sorcerer** class to support rapid magic gameplay.
- Make ogres and heavy enemies somewhat faster in initiative while putting slowness into the attacks themselves.
- Soften extreme weight/speed scaling.

---

## 23. Combat Log Presentation

Combat messages should be cinematic but mechanically clear.

Example:

> **Ironhide — Crushing Blow**  
> The ogre heaves its club overhead as Brom braces beneath his shield.  
> **74 PHYSICAL DAMAGE — STAGGERED**  
> Brom: **171 / 245 HP**

For spellcasting:

> **Orin begins chanting Arcane Lance...**

Later:

> The final syllable leaves Orin's lips. Violet light condenses between his hands.  
> **ARCANE LANCE**  
> **91 MAGIC DAMAGE**

Important events should include:

- Damage
- Healing
- Blocks
- Parries
- Misses
- Criticals
- Status effects
- Interrupts
- Cast starts
- Cast completions
- Death / incapacitation
- Guard exhaustion
- Wound changes

---

## 24. Damage Counter / Battle Summary

At meaningful intervals and at the end of battle, Ashveil should show a compact counter.

Example:

**Damage dealt**  
Company: 186  
Enemies: 143

**Healing**  
Company: 52  
Enemies: 0

**Party Status**  
Kael 154/185  
Lyra 145/145  
Orin 81/112  
Seraphine 152/152  
Brom 228/245

The final summary can additionally include:

- Damage per character
- Healing per character
- Damage taken
- Blocks / parries
- Interrupts
- Kills
- Critical hits
- Status effects applied
- Highest single hit
- Number of actions completed
- Time alive

This data will be extremely useful for both player feedback and combat balancing.

---

## 25. Current Combat Prototype v2 Direction

The current working design is:

1. **Readiness fills continuously** according to Speed until it reaches 100.
2. **Weight reduces Speed** using a gentler / diminishing curve.
3. Speed determines when a character becomes **Ready**; it does not directly determine ability resolution speed.
4. The authoritative action lifecycle is **Readiness → Ready → Wind-up / Cast → Effect → Recovery → Readiness**.
5. Readiness does not refill during Wind-up, Cast Time, or Recovery.
6. Abilities define their own **Wind-up / Cast Time**, **Recovery**, power, and interrupt properties.
7. Actions **overlap on a shared combat timeline** and are processed as timestamped events.
8. Wizards retain very high spell damage but use long chants and meaningful Recovery.
9. Sorcerers use shorter casts and lower per-spell damage for higher magical cadence.
10. Heavy enemies receive reasonable initiative but use slow, telegraphed heavy attacks.
11. Casting and heavy attacks can be **interrupted / staggered** according to an Interrupt Threshold / resistance model.
12. Shield tanks use limited **Guard Reactions**, which by default refresh after completing a normal action cycle.
13. Healing uses the same **Ready → Cast → Effect → Recovery** timing rules as other abilities.
14. **Wounds** reduce temporary recoverable HP and are not automatically removed by ordinary healing.
15. AI target selection uses **class / personality priorities**, not perfect mathematical optimization.
16. Critical hits reinforce **weapon identity** through secondary effects.
17. Formation determines protection, target availability, interception, and potentially AoE / flanking behavior.
18. Combat messaging remains cinematic while exposing the underlying numbers.
19. Battle summaries track damage, healing, actions, defensive reactions, and other statistics for balancing.

---

## 26. Important Design Principle Going Forward

A central lesson from Prototype 1 is that **Speed, ability power, Wind-up / Cast Time, Recovery, interrupt resistance, Guard Reactions, and Wounds must remain separate balancing levers**.

A unit should be allowed to be:

- Fast but weak
- Slow but devastating
- Fast but fragile
- Slow but extremely defensive
- Slow to begin an action but quick to recover
- Quick to begin an action but committed to a long recovery
- Powerful but vulnerable while charging

This separation should provide enough space for classes, weapons, armor, monsters, and builds to feel genuinely different without relying on simple stat inflation.

---

## Next Prototype

The next useful step is to formalize **Combat Prototype v2** with actual formulas for:

- Weight → Speed
- Initiative gain
- Wind-up
- Recovery
- Physical damage
- Magical damage
- Armor / mitigation
- Accuracy / evasion
- Crits
- Stagger / interruption
- Guard reactions
- Healing / wounds

Then rerun the same **5v5 Broken Causeway encounter** using the revised system and compare the combat telemetry directly against Prototype 1.
