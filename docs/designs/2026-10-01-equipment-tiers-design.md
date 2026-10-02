# Equipment Families and Tiers

Status: owner-approved equipment design, 2026-10-01; no gameplay implemented.
The owner approved the catalog, six tiers, armor paths, and delivery direction.
On 2026-10-02 the owner removed the proposed new class from documentation;
the equipment catalog, including glaives, remains approved. Numerical balance
and content-placement bands remain starting defaults to verify during implementation.

Local baseline: `54c11ac8` (33f1). Pulling origin failed because the environment
proxy could not be reached; recheck current master before implementation.
Complements [33g equipment and loot](2026-10-01-phase-33g-company-equipment-loot-design.md)
and later progression work without changing their delivery order.

## Design direction and prior art

Use RuneScape's readable material progression, Skyrim's parallel armor paths,
Ogre Battle's distinct formation roles, and Mount & Blade's practical weapon
and shield choices. Use original equipment and class names. Preparation should
offer choices among reach, protection, damage, and carrying supplies.

Keep three separate concepts: family defines function, tier defines the item's
power budget, and rarity describes availability and special properties. A rare
iron sword is not automatically better than every ordinary steel sword.
Within a family, tiers are upgrades; across families, equipment is a tradeoff.
Class preferences do not imply new hard equipment restrictions.

## Proposed weapon catalog

| Family | Examples | Hands | Tactical identity / tradeoff |
|---|---|---|---|
| Daggers | knife, dagger, stiletto | 1 | Light close weapon for Rogue openings; low ordinary damage |
| Swords | shortsword, arming sword, sabre | 1 | Reliable close damage with room for a shield |
| Greatswords | longsword wielded in two hands, greatsword | 2 | Strong close blows; gives up shield and polearm reach |
| Axes | hand axe, war axe | 1 | Cleaving pressure; alternative to swords |
| Heavy axes | battleaxe, greataxe | 2 | Heavy cleaving blows; greater personal load |
| Blunt weapons | club, mace, warhammer | 1 | Existing crushing/status identity; practical shield pairing |
| Heavy blunt weapons | maul, great hammer | 2 | Heavy crushing blows; no shield |
| Spears | short spear, war spear | 1 / 2 | Short spear is close-range with a shield; long war spear has reach and uses two hands |
| Glaives | militia glaive, war glaive | 2 | Two-handed slashing reach; gives up a shield |
| Other polearms | halberd, billhook, pike | 2 | Cleaving or stabbing reach; later specialist variants |
| Staves | quarterstaff, iron-shod staff, runestaff | 2 | Blunt melee option for casters; magical variants need a separate power budget |
| Bows | shortbow, longbow, composite bow | 2 | Ranged formation role; no shield |
| Crossbows | hunting crossbow, arbalest | 2 | Proposed slower, heavier ranged alternative; cadence must be measured |
| Slings | sling, reinforced sling | 1 proposed | Low-weight ranged option; verify offhand behavior before authoring |

Use existing slashing, stabbing, cleaving, bludgeoning, and shooting subtypes
where applicable. Reach and handedness are independent of damage subtype.
Do not promise axe armor penetration, dagger speed bonuses, ammunition, reload
rules, dual-mode weapons, or staff casting bonuses until their mechanics exist.
Improvised weapons and monster natural attacks remain outside this progression.

## Six proposed tiers

Level bands guide content placement, not equip gates. They must be revisited
against 30g4 progression and the 30g balance harness before setting numbers.

| Tier | Suggested band | Weapon materials / workmanship | Armor equivalents | Typical source |
|---|---|---|---|---|
| 1 — Common | 1–9 | Serviceable iron or bronze, plain wood | Padded cloth, hide, iron mail | Starter kits, village merchants, militia |
| 2 — Steel | 10–19 | Steel, seasoned wood, reinforced fittings | Quilted cloth, cured leather, steel mail / plate | Town smiths, guards, common expedition loot |
| 3 — Tempered | 20–29 | Tempered steel, laminated bows | Layered cloth, hardened leather, tempered steel | Specialist smiths, veteran soldiers |
| 4 — Masterwork | 30–39 | Precisely balanced steel, composite bows | Fine woven robes, supple reinforced leather, fitted plate | Regional masters, difficult contracts |
| 5 — Runeforged | 40–49 | Runesteel, rune-bound wood | Runewoven cloth, runehide, runesteel mail / plate | Rare materials, dangerous ruins |
| 6 — Relic | 50–60+ | Named weapons in ancient alloys or enchanted wood | Named robes, hides, mail, and plate | Major expedition rewards, bosses, quest chains |

Bronze and iron are alternatives within Tier 1, not mandatory consecutive
grinds. A masterwork steel weapon can outrank a poorly worked exotic material.
Shadowsteel already appears in shipped content; audit it before deciding its
tier. Relics have bounded special effects, not unrestricted stat inflation.

Every family has an equivalent at each tier, but launch content need not ship
every combination. Bows and cloth progress by construction, not metal names.
Higher tiers should improve effectiveness at broadly comparable family weight;
keep reducing burden as a possible premium benefit, not an automatic rule.

Proposed glaive progression: Militia Glaive → Steel Glaive → Tempered War
Glaive → Masterwork Glaive → Runesteel Glaive → named relic **Ashen Reaper**.
The final name does not imply a death spell or lifesteal mechanic.

## Armor paths and shields

| Path | Example progression | Recommended users | Tradeoff |
|---|---|---|---|
| Light cloth | padded tunic → quilted robe → layered robe → masterwoven robe → runewoven robe → relic vestments | Wizard, lightly equipped Cleric | Low protection, ample capacity for supplies; any magical bonus is separately budgeted |
| Light leather | hide → leather → hardened leather → masterwork leather → runehide → relic leather | Rogue, Ranger | Practical protection while keeping personal load low |
| Medium | padded jack → scale → brigandine → fitted brigandine → runescale → relic harness | Mobile Warrior or Cleric | Protection between leather and heavy plate, moderate burden |
| Heavy | iron mail → steel mail / plate → tempered plate → fitted plate → runesteel plate → relic plate | Frontline Warrior, armored Cleric | Highest protection and weight; often less dodge capacity |

Light/medium/heavy are catalog labels initially, not new engine armor skills.
The existing burden calculation determines the dodge cost from actual weight
and Strength. Cloth must not gain plate-level physical protection just by tier.
Avoid invented spell-failure or stealth penalties in the first slice.

Author coherent head/body/gloves/legs/feet sets using existing slots. Neck,
belt, and rings remain accessories with separately bounded bonuses. A cape
and coif compete for the existing neck slot unless a later slot redesign is
approved. No new slots or set bonuses are needed for initial content.

Shield choices: buckler (light), round/heater shield (balanced), kite shield
(heavier), tower shield (specialist, later). Each can use the same tier ladder.
Shield differences initially come from supported protection and weight fields;
do not imply new shield-size block formulas. Preserve 30g2's block rules and
the two-handed weapon's loss of an offhand.

## Integration, ownership, and delivery

Equipment remains normal item instances on the character/company save seams.
Use `internal/items/itemspec.go` for handedness, subtype, reach, weight, damage,
and protection; `_datafiles/world/default/items/` for content. Tier metadata,
if needed in comparisons, requires an optional field with a legacy default;
do not infer tier from item names or silently rewrite old saved instances.
Audit existing items before assigning tiers and reserve IDs without collisions.

Use 33g's comparisons to show damage, reach, shield loss, actual protection,
personal burden, and cost. Raw rank alone must not recommend plate to everyone.

No global time advancement, new inventory system, or changes to multiplayer
loot claims. Gear must survive save/restart/copyover.

Recommended delivery: first audit/budget existing items and author a small
Tier 1–3 catalog (sword, axe, mace, spear, glaive, bow, staff, all armor paths,
and shields); then expand Tier 4–6 rewards.
Do not introduce crafting, durability, ammunition, or new classes
as prerequisites. Gameplay implementation remains a subsequent unit of work
in the existing phase sequence; this change delivers the approved design only.

## Acceptance and player help

- Verify every new item loads with unique ID, valid subtype, correct hands,
  reach, weight, value, and existing equipment slots; two-handed glaives deny shields.
- Exercise real attacks from both sides, formation reach, ordinary hit/defense
  checks, and the loss of shields when equipping two-handed weapons.
- Test starter-kit value parity, equipment transfers, restart, and copyover.
- Run 30g balance scenarios for equal-tier companies, mixed gear, burden, and
  shield versus glaive loadouts; keep the established combat-duration target.
- Ship indexed `help equipmenttiers`; update equipment, armor, formation reach,
  and combat help as needed. Add a
  preparation/combat tutorial pointer and render/pointer tests. Publish these
  pages with implementation, not as claims that proposals are playable now.
- Obtain independent full implementation review and record actual verification
  in Project Status before integration. This draft requires documentation checks only.
