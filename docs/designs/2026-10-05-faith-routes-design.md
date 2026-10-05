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
| Cleric | Good (+30) | **Priest** | **Hierarch** | Strongest direct healing and wards; at 30 calls an **Angel** |
| Cleric | Any | **Druid** | **Elder Druid** | Healing over time and nature protection |
| Cleric | Evil (−30) | **Blood Priest** | **Demonologist** | Heals by draining foes; at 30 binds a **Demon** |
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

## 2. Good route: Priest → Hierarch

**Priest (level 10, alignment +30 or higher).** The best direct healer.

- **Signature: Ward.** Places a ward on one ally that absorbs the next
  blow's damage, up to about one average hit of the cleric's level (15–20%
  of a warrior's HP). Chant 1, cost 10, one ward per ally. At level 20 a
  ward absorbs two blows.
- **Greater Heal:** about two average hits; chant 2, cost 14.
- Priests heal at the standard after-battle patching cost.

**Hierarch (level 30).**

- **Signature: Call the Host.** Summons an **Angel** for the rest of the
  battle (summoning rules, section 4).
- Heals also remove one harmful status (bleeding, poisoned) from their
  target, once per patient per battle.
- **Level 40:** the Angel arrives one chant round sooner. **Level 50:**
  the Angel's Mercy heals every 2 rounds instead of 3.

**The Angel** (a guardian of the Host):

| | |
|---|---|
| Level | The Hierarch's level; Attack and Evasion rates 1.0 |
| HP | A warrior's of that level (+25% with a holy symbol) |
| Weapon | Radiant blade, 1d8; +50% against undead and demons |
| Guard | Steps in for the most hurt ally in reach (the 30c2 Guardian rule) |
| Mercy | Every 3 rounds, heals the most hurt ally for half a Minor Heal |

## 3. Unrestricted route: Druid → Elder Druid

For any alignment; healing that works slowly.

- **Druid (level 10). Signature: Rejuvenation.** Heals one ally over 3
  rounds for a total of 130% of a Minor Heal, at a Minor Heal's cost.
  **Barkskin:** one ally +10 armor for the battle.
- **Elder Druid (level 30). Signature: Grove.** Rejuvenation on a whole
  formation row at 60% strength each.
- No summon. A later spirit-beast companion for this route belongs in the
  expanded catalogue's Beastkeeper rules, not here.

## 4. Evil route: Blood Priest → Demonologist

**Blood Priest (level 10, alignment −30 or lower).** Heals by taking life.

- **Signature: Siphon.** Damages one foe for a Magic Missile's worth
  (scaled by the 35a2 skill edge) and heals the most hurt ally for the
  damage dealt. Chant 1, cost 10. A strong fighter's tool, useless out of
  battle.
- **Dark healing is hungry:** the Blood Priest's ordinary heals and
  after-battle patching cost 25% more mana.
- **Level 20:** Siphon also strikes a second foe and heals a second ally,
  at 60%.

**Demonologist (level 30).**

- **Signature: Bind the Fiend.** Summons a **Demon** for the rest of the
  battle (summoning rules below).
- **Level 40:** the Demon's arrival forces a morale check on every enemy
  group, not only its target's. **Level 50:** the Demon's claws hit
  harder (2d8).

**The Demon** (a bound fiend):

| | |
|---|---|
| Level | The Demonologist's level; Attack rate 1.1, Evasion rate 0.9 |
| HP | 80% of a warrior's of that level (+25% with a holy symbol) |
| Weapon | Claws, 2d6, which hits harder than the Angel |
| Dread | On arrival, one morale check against its target's group (30e) |
| Broken binding | If the Demonologist falls, the Demon breaks free: its next turn attacks the nearest creature, friend or foe, then it vanishes |

The broken binding is the evil route's price: a Demonologist the company
fails to protect becomes a danger to the company.

### Summoning rules (Angel and Demon)

- **Casting:** a 3-round chant (a blow can break it, as with any chant),
  costing 40% of the caster's maximum mana. Battle only; **once per
  battle** from level 30 (owner). The mana cost is what limits it: a full
  pool pays for about two summons before the company must rest.
- **Duration:** until the battle ends, the summon dies, or the summoner
  falls (the Angel departs; the Demon breaks free).
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
so the Paladin can stand in the front row and heal there.

**Knight (level 10).** Already planned to brace for its ward (guardian).

- **Signature: Lay on Hands.** A touch that heals itself or an adjacent
  ally for half a Minor Heal of the Knight's level. It takes the Knight's
  turn, has no chant and costs no mana: **2 uses per rest**, 3 at level 20.

**Paladin (level 30).**

- Lay on Hands heals a full Minor Heal and removes one harmful status;
  3 uses per rest, 4 at level 40.
- **Aura of Resolve:** allies in the Paladin's row take 10% less damage.
  Auras don't stack within a row.

Paladins heal **reliably**: no need to land a blow, but a fixed number of
times per rest.

## 6. Evil fighting healer: Blackguard → Dread Knight

Warrior route, alignment −30 or lower. Heavy armor and any shield.

**Blackguard (level 10).**

- **Signature: Blood Oath.** When the Blackguard lands a melee blow, half
  the damage dealt heals **the most hurt ally** in the company (owner: any
  hurt ally), or the Blackguard itself if no ally is hurt. It works on up
  to 3 blows per battle, 4 at level 20, and renews every battle.
- Keeps the Reaver's intimidation: a foe the Blackguard has wounded this
  round has −3 Attack against the Blackguard's allies.

**Dread Knight (level 30).**

- Blood Oath heals the most hurt ally for 75% of the damage, and the next
  most hurt for half that again; 5 blows per battle.
- **Aura of Dread:** foes in reach of the Dread Knight have −5 Attack.

Blackguards heal **by hurting**: renewable every battle with no rest
needed, but only while they land blows. A missed or defended swing heals
no one.

## Balance and acceptance (with 38b)

With `ASHVEIL_BALANCE=1`:

1. **Summons:** a Hierarch or Demonologist company against a boss band wins
   10–20 points more often than the same company with a Priest or Blood
   Priest, and spends most of its caster's mana doing it. A summon is
   never present after the battle, after a copyover or in a save.
2. **Broken binding:** a Demonologist that falls makes its Demon attack the
   nearest creature once, then vanish; tested through a real battle.
3. **Fighting healers:** a Paladin company and a Blackguard company each
   lose 15–30% less HP per fight than a Mercenary company at the same
   level, and neither heals as much as a Priest.
4. **Priests stay protected:** with no warrior in front, a cleric loses
   more fights than with one (the class needs protecting, as intended).
5. **Wiring:** promotion gates by alignment; each signature through its
   real cast or attack path; companion strategy rules for Ward, Siphon,
   Lay on Hands, Blood Oath and the summons; the holy symbol's bonuses.
6. **Help:** a page per route (`help priest`, `help hierarch`, `help
   druid`, `help blood priest`, `help demonologist`, `help knight`, `help
   paladin`, `help blackguard`, `help dread knight`), `help summoning`,
   updates to `help cleric` and `help warrior`, and tutorial pointers from
   the promotion lesson.

All numbers are starting values for the balance tests.

## Owner answers (2026-10-05)

1. **Summon frequency:** once per battle from level 30; mana limits it.
2. **Unrestricted cleric:** Druid.
3. **Blood Oath:** heals any hurt ally (the most hurt), not only a ward.

No questions are open. Delivery is planned with class promotions (38b).
