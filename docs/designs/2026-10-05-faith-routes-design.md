# Faith routes: clerics, summoners and fighting healers

Status: **design draft; the owner's answers to its open questions are
recorded (2026-10-05)** (handoff rule 20). It
revises the cleric and warrior rows of the
[branching class progression design](2026-10-01-branching-class-progression-design.md)
and the cleric paths of the [expanded companion catalogue](2026-10-01-expanded-companion-classes-design.md).
It is delivered with class promotions (38b) and builds on the combat rules
of [35a2 skill over hit points](2026-10-05-phase-35a2-skill-over-hit-points-design.md).

## Owner direction (2026-10-05)

- **Clerics are priests:** healers who can't fight well and need
  protecting. No shields; maces, staffs or rods; a holy symbol.
- **Fighting healers are warriors,** with a route for **good** and one for
  **evil** characters.
- Clerics evolve differently on the **good** and **evil** routes, and a
  higher-level cleric can become a **summoner who commands an angel or a
  demon**.
- The owner is open to ideas for cleric skills as they level.

## Summary

| Lineage | Gate | Advanced (level 10) | Elite (level 30) | Role |
|---|---|---|---|---|
| Cleric | Good (+30) | **Priest** | **Hierarch** | Strongest direct healing and wards; from 30 calls an **Angel** that grows every 5 levels |
| Cleric | Any | **Druid** | **Elder Druid** | Healing over time and nature protection |
| Cleric | Evil (−30) | **Blood Priest** | **Demonologist** | Heals by draining foes; from 30 binds a **Demon** that grows every 5 levels |
| Warrior | Good (+30) | **Knight** | **Paladin** | Armored protector who heals by laying on hands |
| Warrior | Any | Mercenary | Warlord | Unchanged |
| Warrior | Evil (−30) | **Blackguard** | **Dread Knight** | Armored protector who heals by spilling blood |

Changes to the earlier designs:

- The cleric's **Chaplain → War Priest** route (frontline support) is
  removed: fighting healers are warriors now.
- The cleric's **Hexer → Hierophant of Ash** route is removed: the Witch
  owns hexes (level impact design §3).
- The catalogue's **Druid** becomes the cleric's unrestricted core route.
  Oracle, Shaman and Exorcist stay in the expanded catalogue.
- The warrior's evil route changes from **Reaver** (pure aggression) to
  **Blackguard**, a fighting healer. Pure aggression stays available
  through the catalogue's unrestricted **Berserker**. The elite keeps the
  name **Dread Knight**.
- The earlier "no persistent spirit summons" rule stands: summons last one
  battle and are never saved.

## 1. The base cleric, levels 1–9

The priest's job is to keep the company standing from the back row, and
the company's job is to protect the priest (35a2: cleric Attack and Evasion
as low as the wizard's, light armor, no shield).

| Level | Gains | Notes |
|---|---|---|
| 1 | Minor Heal, Tend (wounds), Vigil (camp utility) | As today |
| 3 | **Minor Heal All** | Moved down from 5 (milestone table) |
| 5 | **Talent** (pick 1 of 3) | See the list below |
| 6 | **Cure Poison** | Existing spell, now granted |
| 8 | **Bless** (new) | One ally: +5 Attack and +5 Evasion for 3 combat rounds; chant 1, cost 8; one Bless per ally at a time |
| 10 | **Promotion** | Priest, Druid or Blood Priest |

**Cleric talents** (levels 5, 15, 25, 35 …, one-time choices with caps):

- **Mending Hands:** +10% healing.
- **Steadfast Prayer:** a blow breaks the cleric's chant 25% less often.
- **Deep Well:** +10% maximum mana.
- **Sanctuary:** +3 Evasion while chanting.
- **Gentle Rest:** after-battle patching costs 15% less mana.

**Holy symbol** (new starting item): a light off-hand focus. It can't
block. It adds +5% to the cleric's healing and **+25% to a summon's
health** (sections 2 and 4). A cleric chooses between a two-handed staff
(better parry) and mace with symbol (better healing and summons).

## Ranks: something new every 5 levels to 60

Owner (2026-10-05): a route's benefits arrive one at a time, **one rank
every 5 levels**, so progression continues all the way to level 60 (level
impact design §1e). Advanced routes have ranks at 10, 15, 20 and 25; elite
routes at 30, 35, 40, 45, 50, 55 and 60. Talents still come at 5, 15, 25,
35, 45 and 55. The tables below are the cleric and warrior ranks; every
number is a starting value for the balance tests.

## 2. Good route: Priest → Hierarch

**Priest** (alignment +30 or higher). The best direct healer.

| Rank | Gains |
|---|---|
| 10 | **Ward** (signature): one ally's next blow is absorbed, up to one average hit of the Priest's level (15–20% of a warrior's HP). Chant 1, cost 10, one ward per ally |
| 15 | **Greater Heal:** about two average hits; chant 2, cost 14 |
| 20 | A ward absorbs two blows |
| 25 | **Prayer of Mending:** after-battle patching costs 20% less mana |

**Hierarch.** Calls an **Angel** that grows with every rank (summoning
rules below).

| Rank | Gains |
|---|---|
| 30 | **Call the Host** (signature): the Angel arrives with a warrior's HP of the Hierarch's level (+25% with a holy symbol), Attack 1.0 and Evasion 1.1 rates, a radiant blade (1d8), **Guard** once a battle (steps in for the most hurt ally, the 30c2 rule) and **Mercy** (every 3 rounds, half a Minor Heal to the most hurt ally). The Hierarch's own heals also remove one harmful status, once per patient per battle |
| 35 | **Armor of the Host:** the Angel gains armor like a warrior's heavy kit (defense 40), without heavy armor's slowness |
| 40 | **Mercy** becomes a full Minor Heal every 2 rounds |
| 45 | **Wings of the Host:** allies in the Angel's row gain +5 Evasion while it stands |
| 50 | **Sword of the Host:** Guard up to 3 times a battle; the blade becomes 1d10, +50% against undead and demons |
| 55 | **Cleansing light:** on arrival, the Angel removes one harmful status (bleeding, poisoned, knocked down) from every ally |
| 60 | **Swift Host:** the Angel arrives one chant round sooner, and Mercy heals the two most hurt allies |

## 3. Unrestricted route: Druid → Elder Druid

For any alignment; healing that works slowly. No summon (a later
spirit-beast companion belongs in the catalogue's Beastkeeper rules).

| Rank | Gains |
|---|---|
| 10 | **Rejuvenation** (signature): one ally heals over 3 rounds for 130% of a Minor Heal, at a Minor Heal's cost |
| 15 | **Barkskin:** one ally +10 armor for the battle |
| 20 | Rejuvenation lasts 4 rounds (160%) |
| 25 | **Thornhide:** a foe that strikes a Barkskinned ally takes 2 damage |
| 30 | **Grove** (elite signature): Rejuvenation on a whole formation row at 60% each |
| 35 | **Nature's patience:** after-battle patching costs 20% less mana |
| 40 | Grove at 80% |
| 45 | **Entangle:** one foe is hobbled for 2 rounds; chant 1, cost 10 |
| 50 | Barkskin covers a whole row |
| 55 | **Wild growth:** Rejuvenation also cures poison |
| 60 | Grove covers two rows |

## 4. Evil route: Blood Priest → Demonologist

**Blood Priest** (alignment −30 or lower). Heals by taking life. **Dark
healing is hungry:** its ordinary heals and after-battle patching cost 25%
more mana.

| Rank | Gains |
|---|---|
| 10 | **Siphon** (signature): damages one foe for a Magic Missile's worth (scaled by the 35a2 skill edge) and heals the most hurt ally for the damage dealt. Chant 1, cost 10; useless out of battle |
| 15 | Siphon costs 8 |
| 20 | Siphon also strikes a second foe and heals a second ally, at 60% |
| 25 | **Blood ward:** Siphon healing beyond an ally's full health becomes a ward of up to half a hit |

**Demonologist.** Binds a **Demon** that grows with every rank. The
**broken binding** is the evil route's price until it is mastered: if the
Demonologist falls, the Demon breaks free, its next turn attacks the
nearest creature, friend or foe, then it vanishes.

| Rank | Gains |
|---|---|
| 30 | **Bind the Fiend** (signature): the Demon arrives with 80% of a warrior's HP of the Demonologist's level (+25% with a holy symbol), Attack 1.1 and Evasion 0.9 rates, claws (2d6) and **Dread**: on arrival, one morale check against its target's group (30e) |
| 35 | **Infernal hide:** the Demon gains armor (defense 30) |
| 40 | **Terror:** Dread checks every enemy group, not only its target's |
| 45 | **Hellfire:** foes within the Demon's reach take 2 damage each round |
| 50 | **Rending claws:** 2d8, +50% against holy creatures |
| 55 | **Soul feast:** when the Demon kills, the most hurt ally heals half a Minor Heal and the Demonologist regains 5% of its mana |
| 60 | **Mastered binding:** the Demon arrives one chant round sooner, and a falling Demonologist's Demon simply vanishes |

## Summoning rules (Angel and Demon)

- **Casting:** a 3-round chant (a blow can break it, as with any chant),
  costing **10%** of the caster's maximum mana (owner). Battle only;
  **once per battle** from level 30 (owner). The cost is light, so a
  summoner calls in most battles; the once-per-battle limit and the
  protected chant are what hold it back.
- **Duration:** until the battle ends, the summon dies, or the summoner
  falls (the Angel departs; the Demon breaks free until the rank-60
  mastered binding).
- **Ranks:** a summon has every rank its summoner has reached, read when
  it arrives.
- **Not a company member:** it takes no company slot (the company stays
  leader + 4) and is **never saved**. A copyover during the battle
  banishes it.
- **Formation:** it stands in a free front-row cell, or in front of the
  summoner when the row is full (34a formation).
- **Control:** it follows the company's focus and its fixed role (Angel:
  guardian and healer; Demon: striker). It is not commanded directly.
- **Credit:** its kills count as the summoner's for experience and loot.
- **Companions** on these routes summon too, by strategy rule: when the
  battle has 3 or more foes or a boss, and their mana stays above the 33e
  reserve after the cost.
- **Enemies:** enemy priests may summon in later zone content, under the
  same rules.
- **Implementation:** a temporary mob instance from a new template
  (`angel of the host`, `bound fiend`), charmed to the summoner for the
  battle with the existing charm system, flagged as summoned, and despawned
  when the battle ends. This fills the empty `spells.Summon` stub.

## 5. Good fighting healer: Knight → Paladin

Warrior route, alignment +30 or higher. Heavy armor and any shield (35a2),
so the Paladin can stand in the front row and heal there. Paladins heal
**reliably**: no need to land a blow, but a fixed number of times per rest.

| Rank | Gains |
|---|---|
| 10 | **Lay on Hands** (signature): heals itself or an adjacent ally for half a Minor Heal of its level; takes its turn, no chant or mana; 2 uses per rest. Braces for its ward (guardian, already planned) |
| 15 | Lay on Hands: 3 uses per rest |
| 20 | **Shield of faith:** +5 block chance while guarding a ward |
| 25 | Lay on Hands also stops bleeding |
| 30 | **Paladin** (elite signature): Lay on Hands heals a full Minor Heal and removes one harmful status |
| 35 | **Aura of Resolve:** allies in the Paladin's row take 10% less damage (auras don't stack within a row) |
| 40 | Lay on Hands: 4 uses per rest |
| 45 | **Smite:** the Paladin's blows deal +50% against undead and demons |
| 50 | Aura of Resolve also gives the rest of the company 5% |
| 55 | Lay on Hands reaches any ally in the Paladin's reach, not only adjacent ones |
| 60 | **Divine shield:** once a battle, the Paladin ignores the next blow; Lay on Hands 5 uses per rest |

## 6. Evil fighting healer: Blackguard → Dread Knight

Warrior route, alignment −30 or lower. Heavy armor and any shield.
Blackguards heal **by hurting**: renewable every battle with no rest
needed, but only while they land blows. A missed or defended swing heals
no one.

| Rank | Gains |
|---|---|
| 10 | **Blood Oath** (signature): when it lands a melee blow, half the damage dealt heals the most hurt ally in the company (owner: any hurt ally), or itself if no one is hurt; up to 3 blows per battle, renewed every battle. **Intimidation:** a foe it wounded this round has −3 Attack against its allies |
| 15 | Blood Oath: 4 blows per battle |
| 20 | Intimidation: −5 Attack |
| 25 | Blood Oath: 5 blows per battle |
| 30 | **Dread Knight** (elite signature): Blood Oath heals 75% of the damage |
| 35 | **Aura of Dread:** foes within its reach have −5 Attack |
| 40 | Blood Oath also heals the next most hurt ally for half as much |
| 45 | Blood Oath: 6 blows per battle |
| 50 | **Terror:** its critical hits force a morale check on the target's group |
| 55 | Blood Oath: 7 blows per battle |
| 60 | **Unholy vigor:** Blood Oath heals 100% of the damage dealt |

## Balance and acceptance (with 38b)

With `ASHVEIL_BALANCE=1`:

1. **Summons:** a Hierarch or Demonologist company against a boss band wins
   10–20 points more often than the same company with a Priest or Blood
   Priest. Because summons are cheap (10% mana) and routine, the band
   tables for ordinary groups are also measured with a summon present,
   and stay within the level impact design's §4 targets (the summon makes
   fights easier, not trivial). The Angel and Demon companies stay within
   5 points of each other. A summon is never present after the battle,
   after a copyover or in a save.
2. **Broken binding:** a Demonologist that falls makes its Demon attack the
   nearest creature once, then vanish; tested through a real battle.
3. **Fighting healers:** a Paladin company and a Blackguard company each
   lose 15–30% less HP per fight than a Mercenary company at the same
   level, and neither heals as much as a Priest.
4. **Priests stay protected:** with no warrior in front, a cleric loses
   more fights than with one (the class needs protecting, as intended).
5. **Ranks:** each rank applies from its level, never earlier, for players
   and companions; a level lost to death removes the rank until it is
   regained (ranks are derived from level, never saved).
6. **Wiring:** promotion gates by alignment; each signature through its
   real cast or attack path; companion strategy rules for Ward, Siphon,
   Lay on Hands, Blood Oath and the summons; the holy symbol's bonuses.
7. **Help:** a page per route, each listing its rank table (`help priest`, `help hierarch`, `help
   druid`, `help blood priest`, `help demonologist`, `help knight`, `help
   paladin`, `help blackguard`, `help dread knight`), `help summoning`,
   updates to `help cleric` and `help warrior`, and tutorial pointers from
   the promotion lesson.

All numbers are starting values for the balance tests.

## Owner answers (2026-10-05)

1. **Summon frequency:** once per battle from level 30; mana limits it.
2. **Unrestricted cleric:** Druid.
3. **Blood Oath:** heals any hurt ally (the most hurt), not only a ward.

4. **Summon cost and the Angel** (later the same day): summons cost 10%
   of maximum mana, not 40%; the Angel is strengthened to be clearly
   useful (armor, more guards, a full Mercy heal every 2 rounds, an
   Evasion aura and a cleanse on arrival).

5. **Ranks every 5 levels** (later the same day): the Angel's benefits,
   and every route's, arrive one rank every 5 levels up to 60 instead of
   all at the elite promotion; the Demon follows the same ladder.

No questions are open. Delivery is planned with class promotions (38b).
