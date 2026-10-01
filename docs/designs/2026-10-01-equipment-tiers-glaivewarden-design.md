# Equipment Families, Tiers, and the Glaivewarden

Status: owner-approved design, 2026-10-01; no gameplay implemented. Owner request:
brainstorm weapons, armor, and tiers using fantasy RPGs as guides, and include
a glaive and a class for it. The owner approved the catalog, six tiers, armor
paths, Glaivewarden, and delivery direction after reviewing the detailed summary,
and requested committing, merging, and pushing this design. Numerical balance
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
| Glaives | militia glaive, war glaive | 2 | Slashing reach; Glaivewarden's signature weapon |
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
| Medium | padded jack → scale → brigandine → fitted brigandine → runescale → relic harness | Glaivewarden, mobile Warrior or Cleric | Protection between leather and heavy plate, moderate burden |
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

## New class: Glaivewarden

Proposed archetype ID: `glaivewarden`. Available to both player characters and
recruitable companions, alongside Warrior, Rogue, Ranger, Wizard, and Cleric.
Its role is a melee reach attacker, usually in the second row behind a shield
fighter. It can fight in front, but cannot block while wielding its glaive.
Medium armor is the recommended loadout, not an equipment prohibition.

Signature equipment: two-handed slashing glaive with existing `reach: true`.
Use the actual column/frontmost-or-one-behind reach rules; second-row placement
does not grant access to every enemy or bypass allied interception and targeting.
Other classes may wield glaives; the class earns its identity through its ability.

Proposed automatic ability **Sweeping Cut**: when two standing hostile members
of the active battle occupy adjacent columns in the same row and both are
reachable, replace the ordinary weapon turn with one sweep at them. Initial
balance candidate: each hit uses 60% of an ordinary strike's damage, separate
hit and defense checks, one strike per target, cooldown four combat rounds.
Misses consume the attempt. No eligible pair means an ordinary attack and no
cooldown spent. Prefer the current target plus a deterministic eligible neighbor.
It must respect ordinary armor, reactions, target ownership, and battle gates.
No extra guaranteed critical, forced movement, or knockdown is included.

This provides a different reason to recruit it than Warrior's single-target
Tackle. Reduced damage against a lone opponent and no shield are its costs.
The numbers remain provisional: integer rounding must not create zero-damage
starter attacks, and multi-hit crit/status behavior must be explicitly tested.
The sweep occupies the whole turn; it cannot multiply attacks or also cast.

Player ability eligibility needs a dedicated class skill/claim in the current
33e skill-based unlock seam; companions use their archetype. Inspect 33f1's
retirement rules before adding that claim. Do not reuse brawling and accidentally
grant Tackle or Sweeping Cut to the wrong class. Starter kit: Tier 1 glaive,
padded jack, simple boots, food, water, and satchel; equal-value budget to the
existing kits. Add a recruit template with a durable archetype and equipment.

## Integration, ownership, and delivery

Equipment remains normal item instances on the character/company save seams.
Use `internal/items/itemspec.go` for handedness, subtype, reach, weight, damage,
and protection; `_datafiles/world/default/items/` for content. Tier metadata,
if needed in comparisons, requires an optional field with a legacy default;
do not infer tier from item names or silently rewrite old saved instances.
Audit existing items before assigning tiers and reserve IDs without collisions.

Class config and kits live in `modules/archetype/files/data-overlays/config.yaml`;
automatic abilities in `internal/strategy/abilities.go` and their real combat
resolution in `internal/hooks`. Recheck all player/mob combat directions,
creation, training, strategy defaults, recruit persistence, and browser views.
Use 33g's comparisons to show damage, reach, shield loss, actual protection,
personal burden, and cost. Raw rank alone must not recommend plate to everyone.

No global time advancement, new inventory system, or changes to multiplayer
loot claims. Class and gear must survive save/restart/copyover. Cooldown state
must follow the existing battle lifecycle, including battle cleanup and recovery.

Recommended delivery: first audit/budget existing items and author a small
Tier 1–3 catalog (sword, axe, mace, spear, glaive, bow, staff, all armor paths,
and shields); then add Glaivewarden end to end; then expand Tier 4–6 rewards.
Do not introduce crafting, durability, ammunition, or additional polearm classes
as prerequisites. Gameplay implementation remains a subsequent unit of work
in the existing phase sequence; this change delivers the approved design only.

## Acceptance and player help

- Verify every new item loads with unique ID, valid subtype, correct hands,
  reach, weight, value, and existing equipment slots; two-handed glaives deny shields.
- Exercise real attacks from both sides, formation reach, a blocked neighbor,
  one surviving enemy, misses, crit/status effects, integer damage rounding,
  cooldown expiry, disabled actors, and no attacks on a waiting battle or ally.
- Test class creation, ability isolation from Warrior, companion recruitment,
  starter-kit value parity, equipment transfers, restart, and copyover.
- Run 30g balance scenarios for equal-tier companies, mixed gear, burden, and
  shield versus glaive loadouts; keep the established combat-duration target.
- Ship indexed `help glaivewarden` and `help equipmenttiers`; update equipment,
  armor, archetype, strategy, formation reach, and combat help as needed. Add a
  preparation/combat tutorial pointer and render/pointer tests. Publish these
  pages with implementation, not as claims that proposals are playable now.
- Obtain independent full implementation review and record actual verification
  in Project Status before integration. This draft requires documentation checks only.
