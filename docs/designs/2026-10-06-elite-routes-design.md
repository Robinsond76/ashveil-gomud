# Elite routes: levels 30–60 for the six lineages

Status: **approved under the owner's delegation (2026-10-06)**. The owner
handed design approval to Claude on 2026-10-06
([remaining roadmap](../plans/2026-10-06-remaining-roadmap.md)); every open
question is decided below with a one-line reason. Every number is a starting
value for the balance harness. Built by phase 38c
([plan](../plans/2026-10-06-phase-38c-elite-routes.md)).

Builds on the [branching class progression](2026-10-01-branching-class-progression-design.md)
(route names, gates, commands), [level impact](2026-10-05-level-impact-class-power-design.md)
§1e (a rank every 5 levels) and §3 (the Witch), the approved
[faith routes](2026-10-05-faith-routes-design.md) (cleric and warrior
elites, summoning rules, the elite wait), and the
[neutral classes](2026-10-05-neutral-classes-design.md) (whose elites are
39i, not this design).

## Scope

1. **Shared elite rules** for all eighteen elite routes: promotion at 30,
   gates, catch-up ranks, elite talents and what a player sees.
2. **Rank tables 30–60** for the thirteen elites the faith routes design
   doesn't cover: Warlord; Pathfinder, Swordmaster, Nightblade; Sentinel,
   Marksman, Ravager; Archon, Archmage, Necromancer; Wise One, Coven Mother,
   Crone of Ash.
3. The five faith elites (Hierarch, Elder Druid, Demonologist, Paladin,
   Dread Knight) keep their approved tables unchanged; this design only adds
   the shared rules, talents and UI around them.

## Designing against 38b

38b (promotion at 10, talents, advanced ranks 10–25) is being built in
parallel and its plan was not yet published when this was written. This
design assumes what the approved designs say 38b ships:

- a class graph (lineage, current class, tier) with `class`, `class paths`
  and `class promote … confirm`, saved on the player's archetype record and
  the companion's company record;
- advanced ranks at 10, 15, 20 and 25, **derived from level**, never saved;
- talents at 5, 15 and 25 from a list per lineage;
- the faith routes' summon framework (a battle-only charmed mob instance).

Each elite table below names the **advanced signature it develops**, as the
branching design and level impact §3c describe it. If 38b shipped a
different signature for an advanced route, 38c keeps the elite's role and
rewrites the affected ranks to grow the shipped signature instead, and its
review thread records the change in Project Status. **Reason:** the elite
role is the design's promise; the exact advanced wording is 38b's.

## 1. Shared elite rules

### Promotion at 30

- **Who can promote:** a character at level 30 or higher whose current class
  is the elite's advanced parent, at a camp or safe settlement, out of
  combat, travel and active rest (the 38b promotion context).
- **One elite per advanced route.** The real choice is made at 10; each
  advanced route has one elite continuation. **Reason:** matches the
  branching design's launch scope; more elite choices arrive as catalogue
  bundles (38d).
- **Gates:** good routes need alignment +30, evil routes −30, checked at
  promotion. A character below its gate **waits** with its advanced ranks and
  promotes once its alignment meets the gate (faith routes rule, now for all
  six lineages). Unrestricted routes check level only.
- **A late promoter catches up.** An unpromoted base character at 30+ takes
  its advanced promotion first, then may confirm its elite in the same visit.
  A character that promotes to elite at level 42 gets ranks 30, 35 and 40 at
  once. **Reason:** ranks are derived from level; making a player wait twice
  adds nothing.
- **Inheritance:** an elite keeps every base ability, every advanced rank
  (10–25) and every talent. Elite ranks improve or replace their route's
  signature; they never stack a duplicate passive.
- **Death:** losing a level below a rank's level removes that rank until the
  level is regained; the elite class itself is never lost (branching rule).
- **Routes are final** (owner, faith routes). No command moves a promoted
  character to a sibling elite.
- **No cost.** Elite promotion costs no gold, item or experience.
  **Reason:** the level, the gate and the wait are the price, as at 10.

### Elite ranks

Ranks arrive at 30, 35, 40, 45, 50, 55 and 60, one benefit each (level
impact §1e). Rank 30 is the elite's **signature**; rank 60 is its
**capstone**.

Every rank keeps the existing battle rules:

- **No player input in battle** (Ogre Battle rule). Each elite names its
  strategy role and priority; every ability below is used automatically.
- **No free action.** A held turn, a riposte, an interrupt or a meter push
  spends a turn or a capped per-battle use. Meter boosts (as the Samurai's
  Zanshin) are allowed because they speed a turn rather than add one.
- **No lock loops.** The 38a caps hold: a foe is never asleep or paralyzed
  more than half a fight, and the 2-round re-hex immunity stays.
- **Bosses resist.** Control effects use the 38a boss rule (land chance −25,
  durations halved, at least 1 round). An "outright" kill becomes damage
  against a boss.
- **Formation is locked.** Nothing moves a member between cells.
- **Mana:** elite spells cost more than base ones (level impact §2d); the
  33e reserve stays.

### Elite talents (35, 45, 55)

Each lineage's 38b talent list gains **three elite talents**, offered only
from level 35 and only to an elite of that lineage. At 35, 45 and 55 the
choice shows every talent the character hasn't taken, elite talents
included. **Reason:** a 38b list of five plus three elite talents leaves a
real choice at every elite talent level; separate route-only talents would
add 18 more effects to balance for little gain.

| Lineage | Elite talents |
|---|---|
| Warrior | **Iron Hide:** +5 armor. **Second Wind:** once a battle, at the start of its turn below 25% HP, heals 10% of its maximum HP (the turn is still taken). **Veteran's Edge:** +3 Attack |
| Rogue | **Shadow Footing:** +5 Evasion. **Keen Edge:** +5% critical chance. **Quick Hands:** opening action meter +15 |
| Ranger | **Long Draw:** +10% ranged damage. **Eagle Eye:** +5 Attack with ranged weapons. **Quick Nock:** opening action meter +15 |
| Cleric | **Font of Grace:** +10% maximum mana. **Radiant Healing:** +10% healing. **Unshaken:** a blow breaks its chant 25% less often |
| Wizard | **Deep Reserves:** +10% maximum mana. **Focused Will:** a blow breaks its chant 25% less often. **Spell Edge:** +10% spell damage |
| Witch | **Deep Reserves:** +10% maximum mana. **Iron Will:** a blow breaks its chant 25% less often. **Hex Reach:** its hexes' land chance +5 (still capped at 90) |

Elite talents stack with a 38b talent of the same kind (for example Deep
Well and Font of Grace give +20% mana); each is still taken once.
Companions choose elite talents by the same rule 38b uses for their other
talents.

## 2. Warrior: Mercenary → Warlord

Unrestricted. **Develops:** the Mercenary's control strike (a stronger
Tackle or Shield Bash that knocks down or staggers). **Role:** `striker`,
focused on the company's focus target; the control strike goes to the foe
the company is focusing.

| Rank | Gains |
|---|---|
| 30 | **Marked for Ruin** (signature): a foe the Warlord's control strike lands on is marked for 2 rounds; every ally has +5 Attack against it |
| 35 | **Battle Cry:** at the start of each battle, every ally has +3 Attack for 2 rounds |
| 40 | The control strike's cooldown is 1 round shorter |
| 45 | **Sunder:** the control strike also breaks the target's armor (armor broken, 2 rounds) |
| 50 | Marked for Ruin gives +10 Attack |
| 55 | **Relentless:** when a foe it knocked down stands up, the Warlord's action meter gains 25 |
| 60 | **Warlord's Command** (capstone): once a battle, when an ally falls, every standing ally's action meter gains 25 |

## 3. Rogue

### Scout → Pathfinder (good, +30)

**Develops:** the Scout's reconnaissance (better ambush avoidance) and its
safe Opening Strike against exposed foes. **Role:** `skirmisher`, preferring
a foe that hasn't acted yet, then an exposed foe.

| Rank | Gains |
|---|---|
| 30 | **Pathfinder's Eye** (signature): enemy ambushes on the company happen half as often, and Opening Strike works on any foe that hasn't acted yet this battle |
| 35 | **Expose Weakness:** an Opening Strike that hits leaves the target exposed for 2 rounds |
| 40 | **Scouted ground:** every ally's opening action meter starts 10 higher |
| 45 | **Vanish:** once a battle, when struck below 30% HP, it has +20 Evasion until its next turn |
| 50 | Opening Strike can be used twice a battle |
| 55 | **Trailwise:** the company's chance to find a cache after a battle (37) rises by 10 points |
| 60 | **Ambush Master** (capstone): when the company would be ambushed, it ambushes instead |

### Duelist → Swordmaster (unrestricted)

**Develops:** the Duelist's riposte (a counter-blow after a successful parry,
a few times a battle). **Role:** `duelist`, preferring the strongest foe in
reach.

| Rank | Gains |
|---|---|
| 30 | **Blade Dance** (signature): ripostes deal +50% damage, and it has 2 more ripostes a battle |
| 35 | +5 parry chance |
| 40 | **Disarming Riposte:** a riposte that crits staggers the target (it loses its next action) |
| 45 | 2 more ripostes a battle |
| 50 | **Perfect Parry:** once a battle, it parries the next melee blow automatically, critical hits included |
| 55 | It can riposte foes striking with extended reach (polearms) |
| 60 | **Unbroken Guard** (capstone): no riposte limit while it is above half HP |

### Assassin → Nightblade (evil, −30)

**Develops:** the Assassin's finishing attack (bonus damage on a foe below
half HP, preferring the most hurt foe). **Role:** `finisher`, preferring the
marked foe, then the most hurt foe in reach.

| Rank | Gains |
|---|---|
| 30 | **Death Mark** (signature): its first target each battle is marked and takes +25% damage from it; when that foe falls, the mark passes to the most hurt foe in reach |
| 35 | **Envenom:** its hits poison the target 25% of the time (the existing poison buff). Once weapon poisons ship (43b), a coating lasts one extra hit |
| 40 | The finishing attack applies below 60% HP, not 50% |
| 45 | **Shadowstep:** once every 3 rounds, it may strike the marked foe in the middle row as if it had extended reach; guardians can still intercept |
| 50 | +15% critical chance against the marked foe |
| 55 | **Killing Spree:** when it fells the marked foe, its action meter gains 50, once a round |
| 60 | **Coup de Grâce** (capstone): a blow that lands on a foe below 20% HP fells it outright; against a boss it deals double damage instead |

## 4. Ranger

### Warden → Sentinel (good, +30)

**Develops:** the Warden's Covering Shot (a shot answering a foe that strikes
a back-row ally). **Role:** `protector`, holding a shot whenever a foe can
reach its middle or back row.

| Rank | Gains |
|---|---|
| 30 | **Overwatch** (signature): the Sentinel may hold its turn; the first foe that attacks one of its middle- or back-row allies is shot before the blow resolves, and on a hit that blow lands at half damage. It spends the Sentinel's turn |
| 35 | Covering Shot has +5 Attack |
| 40 | An Overwatch hit knocks down a leaping, diving or mounted foe |
| 45 | **Watchful:** back-row allies have +5 Evasion while it stands |
| 50 | Overwatch can also answer a foe beginning a chant; a hit breaks the chant |
| 55 | Overwatch can trigger twice a round (the second spends its next turn too) |
| 60 | **Guardian Arrow** (capstone): an Overwatch hit stops the blow entirely |

### Hunter → Marksman (unrestricted)

**Develops:** the Hunter's Aimed Shot. **Role:** `marksman`, preferring the
company's focus target, then the most valuable target in range (a caster or
healer). The Marksman is the critical-hit archer; the Arbalist (39h) owns
armor piercing.

| Rank | Gains |
|---|---|
| 30 | **Called Shot** (signature): Aimed Shot has +15% critical chance, and its critical hits deal +50% damage |
| 35 | Aimed Shot's cooldown is 1 round shorter |
| 40 | **Pinning Crit:** an Aimed Shot critical hit hobbles the target for 2 rounds |
| 45 | +10 Attack against back-row foes |
| 50 | Its critical hits can't be blocked |
| 55 | **Second Nock:** an Aimed Shot that fells its target shoots again at a new target, once a round |
| 60 | **Perfect Shot** (capstone): its first Aimed Shot each battle is always a critical hit |

### Stalker → Ravager (evil, −30)

**Develops:** the Stalker's pressure on wounded foes (bonus Attack against a
foe below half HP; its shots can cause bleeding). **Role:** `hunter`,
preferring the most wounded foe in range. It never stops every flight
(branching design): fleeing stays possible, only harder.

| Rank | Gains |
|---|---|
| 30 | **Hunt Down** (signature): its hits on a bleeding foe add another bleeding stack, and a foe it struck this round has 25 points less chance to flee (30e) |
| 35 | +10% damage against bleeding foes |
| 40 | **Harrow:** when a foe it wounded breaks morale, every ally has +5 Attack against that foe |
| 45 | Bleeding from its arrows lasts 1 round longer |
| 50 | **Rend:** its critical hits leave the target exposed for 2 rounds |
| 55 | Hunt Down's flee penalty is 40 points |
| 60 | **Apex** (capstone): each foe it fells forces a morale check on that foe's group |

## 5. Wizard

### Theurgist → Archon (good, +30)

**Develops:** the Theurgist's Spellward (an ally's next harmful spell is
absorbed). **Role:** `support`, holding a Counterspell when an enemy caster
stands, otherwise warding then casting damage.

| Rank | Gains |
|---|---|
| 30 | **Counterspell** (signature): the Archon may hold its turn; the first enemy to begin a chant is countered on a Mysticism edge roll (65% at equal stats, clamped 25–90, bosses −25), breaking the chant. Spends the turn; cost 12 |
| 35 | Spellward covers 2 allies |
| 40 | **Mana Shield:** allies in its row take 10% less spell damage |
| 45 | Counterspell's chance +10, and a countered caster loses 10% of its mana |
| 50 | Spellward also absorbs one melee blow |
| 55 | **Reflection:** once a battle, a Spellward that absorbs a spell turns it on its caster at half power |
| 60 | **Archon's Aegis** (capstone): once a battle, every ally is Spellwarded at once |

### Arcanist → Archmage (unrestricted)

**Develops:** the Arcanist's efficient casting (cheaper spells, slightly
stronger damage). **Role:** `blaster`, the existing wizard damage role.

| Rank | Gains |
|---|---|
| 30 | **Overchannel** (signature): once every 3 rounds, a damage spell deals +50% damage for +50% mana |
| 35 | Damage spells with a chant of 2 or more take one round less |
| 40 | Spells cost 10% less mana (on top of the Arcanist's saving) |
| 45 | **Arcane Barrage:** Magic Missile strikes a second foe at full damage |
| 50 | Overchannel every 2 rounds |
| 55 | **Steady Casting:** a blow breaks its chant 25% less often |
| 60 | **Archmage's Storm** (capstone): once a battle, Shower of Sparks strikes every foe in the enemy group |

### Warlock → Necromancer (evil, −30)

**Develops:** the Warlock's life drain (damage that heals itself) and its
withering debuff. **Role:** `controller`, raising a thrall when a foe falls,
otherwise draining the most dangerous foe in range.

| Rank | Gains |
|---|---|
| 30 | **Raise the Fallen** (signature): once a battle, when a foe falls, a 2-round chant (10% of maximum mana) raises it as a **thrall** with half its HP and its weapon attacks, no spells. It fights for the company until the battle ends or it is destroyed. Bosses can't be raised |
| 35 | Drain Life heals the most hurt ally instead when the Necromancer is above 75% HP |
| 40 | **Grave Chill:** Wither also hobbles the target for 1 round |
| 45 | The thrall rises with 75% of its HP |
| 50 | Raise the Fallen twice a battle |
| 55 | **Death's Harvest:** each foe that falls restores 3% of its mana |
| 60 | **Lich's Bargain** (capstone): once a battle, a blow that would fell the Necromancer leaves it at 1 HP, and its thrall crumbles instead |

A thrall follows the faith routes' summoning rules (no company slot, never
saved, banished by copyover, kills credited to the Necromancer, a free
front-row cell or in front of its master) and uses the same summon
framework. It is not charm magic for players, and it never persists: the
branching design's "no persistent summons" rule stands.

## 6. Witch

The Witch's hexes already reach the whole enemy group at level 30 (38a), so
Witch elites grow control quality, not target count.

### Hedge Witch → Wise One (good, +30)

**Develops:** the Hedge Witch's Warding Hex (a landed hex also wards the
most hurt ally) and longer Slumber. **Role:** `controller`, Slumber first
while an ally is hurt.

| Rank | Gains |
|---|---|
| 30 | **Hearthward** (signature): Warding Hex's ward absorbs a full hit and covers the two most hurt allies |
| 35 | **Deep Slumber:** its Slumber lasts 1 round longer (the 50% cap holds) |
| 40 | **Mend Charm:** when one of its wards breaks, the warded ally heals a quarter Minor Heal |
| 45 | A breaking ward also removes one harmful status |
| 50 | **Hearth's Peace:** allies holding its ward have +5 Evasion |
| 55 | Hearthward covers the three most hurt allies |
| 60 | **Ward of Life** (capstone): once a battle, a warded ally who would fall stays at 1 HP |

### Coven Sage → Coven Mother (unrestricted)

**Develops:** the Coven Sage's reach (an extra hex target before 30) and
shorter chants. **Role:** `controller`, unchanged priorities (38a).

| Rank | Gains |
|---|---|
| 30 | **Coven's Will** (signature): its hexes land 10 points more often (cap 90), and its 2-round chants take 1 round |
| 35 | Its hexes last 1 round longer (the 50% cap holds) |
| 40 | Its hexes cost 20% less mana |
| 45 | The boss resist against its hexes is halved (−12 instead of −25) |
| 50 | **Twin Hex:** every third hex it lands also leaves its targets exposed for 1 round |
| 55 | Its hexes land another 10 points more often (cap 90) |
| 60 | **Coven Circle** (capstone): once a battle, one hex can't be resisted (bosses still halve its duration) |

### Hag → Crone of Ash (evil, −30)

**Develops:** the Hag's curses (hexed foes take +15% damage; Dread Whisper
spreads to an adjacent group). **Role:** `controller`, preferring to hex the
company's focus target so the curse pays off.

| Rank | Gains |
|---|---|
| 30 | **Ashen Curse** (signature): hexed foes take +25% damage (replacing +15%), and allies have +5 Attack against them |
| 35 | **Rotting Miasma:** Miasma's poison deals double damage |
| 40 | Dread Whisper checks every enemy group |
| 45 | **Lingering Curse:** when a hex ends, the foe stays exposed for 1 round |
| 50 | Ashen Curse deals +35% |
| 55 | **Soul Rot:** a hexed foe that falls forces a morale check on its group |
| 60 | **Crone's Doom** (capstone): once a battle, a foe that stays hexed for 3 rounds in a row falls; a boss loses 10% of its maximum HP instead |

## 7. What the player sees

Players never act in battle, so they must understand an elite **before**
the fight, through its preview, its help page and the battle text.

**Readiness.**

- The level-up report at 30 says it once: `Elite promotion ready: Knight →
  Paladin. Visit a camp or town and type class promote.` Companions get the
  same line in the company level-up report.
- A character below its gate gets instead: `Paladin needs alignment +30
  (yours: +22). You keep your Knight ranks and can promote once it rises.`
- `experience` and the milestone line (`internal/milestones`) show the next
  rank or `elite promotion` as delivered (38c flips the level-30 milestone
  and adds 35–60 ranks).

**Promotion preview** (`class promote [self|member]`):

```text
Knight -> Paladin (elite, warrior lineage)
  Gate: alignment +30 or higher (yours: +41)  ready
  Role: armored protector who heals by laying on hands
  Now:  rank 30 Paladin: Lay on Hands heals a full Minor Heal and removes a harmful status
        rank 35 Aura of Resolve: allies in your row take 10% less damage
  Next: rank 40 (level 40): Lay on Hands 4 uses per rest
  Talents from 35 add: Iron Hide, Second Wind, Veteran's Edge
  Routes are final. Type: class promote self paladin confirm
```

The "Now" block lists every catch-up rank. Confirming announces the
promotion to the room and the company, then shows the same "Now" lines.

**Afterwards.**

- **`class`** shows lineage, class and tier, each reached rank on one line,
  and the next rank with its level.
- **`company` and `company inspect`** name each member's class with its tier
  (`Paladin (elite)`) and mark members ready to promote (`promote ready`) or
  waiting on a gate (`waiting: alignment`).
- **Rank-ups** at 35–60 add a line to the level-up report: `Rank 45
  Nightblade: Shadowstep. Once every 3 rounds you can strike the marked foe
  in the middle row.`
- **Battle text** names the effect when it fires, in the 29c narration
  voice: `Brother Aldric's Aura of Resolve blunts the blow.`, `Mira's
  Overwatch arrow spoils the troll's swing.`, `A fallen bandit rises as
  Vess's thrall.` Each signature and capstone has a line; passive number
  changes don't.
- **GMCP** (`modules/gmcp/gmcp.Company.go` and the character payload): each
  member gains `class`, `tier`, `rank` and `promotion` (`ready`,
  `waiting-gate` or empty). The web company window shows the class with an
  elite badge and a ready marker; the character window shows class, tier and
  rank. The battle view names the class. Sprites for elite classes stay
  with 40h.

**Help** (indexed in `keywords.yaml`, `.template` pages):

- `help elite`: the hub. Promotion at 30, gates and waiting, catch-up ranks,
  elite talents, a table of all eighteen elites with their lineage, gate and
  one-line role, and links to each route page.
- A page per elite in this design, each with its rank table and role:
  `help warlord`, `help pathfinder`, `help swordmaster`, `help nightblade`,
  `help sentinel`, `help marksman`, `help ravager`, `help archon`, `help
  archmage`, `help necromancer`, `help wise one`, `help coven mother`,
  `help crone of ash`. Aliases for two-word names (`wiseone`, `crone`).
- `help thrall` (Raise the Fallen and its rules), linked from `help
  summoning`.
- Updates: `help promotion` (elite step, catch-up, link to `help elite`),
  `help talents` (elite talents), `help classes`, each lineage's class page
  and advanced route page (its elite and gate), `help combat` (hub link),
  `help alignment` (gates at 30).
- Faith elite pages (`help hierarch` and the others) gain the shared
  elite-rule links if 38b didn't write them.

**Tutorial:** the lesson that covers promotion (38b's, or Departure) gains a
hint: `At level 30 your class can promote again: help elite.` No new lesson,
since a tutorial character never nears 30.

## 8. State and integration

- **Graph:** eighteen advanced → elite edges with gates, in 38b's class
  graph. No new saved fields beyond the current class 38b already saves.
- **Ranks:** derived from level and current class at use, for players and
  companions, never saved (faith routes rule).
- **Talents:** elite talents join 38b's talent store; an elite talent is
  refused below 35 or for a non-elite.
- **Runtime state:** marks (Marked for Ruin, Death Mark), held turns
  (Overwatch, Counterspell), per-battle uses, Overchannel cooldowns and
  thralls live on the battle and are never saved. A copyover mid-battle
  resolves as other battles do and banishes thralls.
- **Combat hooks:** held turns reuse the Halberdier's Brace pattern (spend
  the turn, trigger once); meter pushes reuse 30g5; marks are runtime
  battle tags like 33i2 coordination; ambush changes reuse the expedition
  ambush roll; morale checks reuse 30e; statuses reuse `internal/status`
  (exposed, hobbled, staggered, knocked down, armor broken, bleeding,
  poisoned).
- **No global time.** Nothing here advances world time.

## 9. Acceptance (38c)

With `ASHVEIL_BALANCE=1`, timeboxed per the balance rule:

1. **Elite power:** a company of elites at levels 35 and 50 stays within the
   level impact §4 band targets against its own band, and each elite wins
   within 5 points of its lineage siblings in the same company slot.
2. **Elites matter:** at level 40, the same company with elites beats its
   advanced-only twin in a hard band (3 levels above) by 10–20 points.
3. **Promotion:** real `class promote` commands for players and companions:
   gate boundaries (+29/+30, −29/−30), the wait and recovery, drift between
   preview and confirm, base → advanced → elite in one visit, catch-up ranks
   at 42, battle, travel and rest rejection, repeated confirms, copyover and
   restart.
4. **Ranks:** each rank applies from its level, never earlier, for players
   and companions; a level lost to death removes the rank until regained.
5. **Signatures through real paths:** each signature and capstone in a real
   battle (held turns spend the turn; marks pass on; thrall raised, never
   saved, banished by copyover; Coup de Grâce becomes damage on a boss;
   Ambush Master flips a real ambush; caps on sleep and paralysis hold).
6. **UI:** the level-up readiness line, preview text, `class`, `company`
   markers and GMCP fields, each covered by a test.
7. **Help:** every page above renders through `help`, is indexed, and
   `TestTutorialHelpPointersExist` passes.

## Decisions (2026-10-06, delegated)

| Question | Decision | Reason |
|---|---|---|
| How many elites per advanced route? | One | Branching launch scope; 38d adds choices |
| Can a late character promote twice in one visit? | Yes | Ranks are level-derived; a second trip adds nothing |
| Catch-up ranks on a late elite promotion? | All ranks up to its level at once | Waiting is already the cost |
| Elite promotion cost? | None | Level, gate and wait are the price, as at 10 |
| Elite talents | Three per lineage, offered from 35 | Keeps a real choice at 35, 45 and 55 with few new effects |
| Witch elites and target count | Grow control quality, not targets | 38a already reaches the whole group at 30 |
| Necromancer summons | One battle-only thrall from a fallen foe | Distinct from the Demon; keeps "no persistent summons" |
| Marksman vs Arbalist | Marksman owns crits, Arbalist armor piercing | Two archers that play differently |
| Nightblade and poisons | Uses the existing poison buff; coatings extend once 43b ships | No dependency on an unbuilt phase |
| Ravager and fleeing | Harder, never impossible | Branching design: never block every flee |
| Elite sprites | 40h | Visual lane owns art |
| One phase or several? | Three build slices, 38c1–38c3 (see the plan) | 126 rank effects is too much for one review |
| 38b signature differs from the assumed one | 38c rewrites the elite ranks to grow the shipped signature, records it | The elite role is the promise, not the wording |
