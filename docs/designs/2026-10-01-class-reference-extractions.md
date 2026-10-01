# Further class ideas from the owner's reference roster

Status: planning additions, 2026-10-01. Source: the owner's supplied English-name
class lists for Ogre Battle 64, Ogre Battle: The March of the Black Queen and
Unicorn Overlord. Treat these lists as supplied reference material, not newly
verified external research. Mechanics below are Ashveil proposals; a name in the
reference does not establish that game's exact abilities or promotion gates.

Related: [expanded companion catalogue](2026-10-01-expanded-companion-classes-design.md)
and [branching progression](2026-10-01-branching-class-progression-design.md).

## Strong additional humanoid roles

These candidates add distinct choices beyond the original thirty-five advanced
paths. Candidate gates default to unrestricted unless explicitly noted; ordinary
candidates use proposed levels 10/30. Final names, balance and lineage assignments
are settled before each delivery bundle. Do not count them as shipped classes.

| Candidate path | Lineage | Reference cue | Ashveil role and cost |
|---|---|---|---|
| Lancer → Dragoon | Warrior | Soldier/Sergeant, Phalanx, Dragoon | Offensive spear reach and thrusts against armored or large foes; differs from shield-focused Hoplite. No automatic mounted combat or jumping past formation |
| Grappler → Monk | Warrior | Grappler; Monk | Unarmed close-range disruption and self-discipline; weaker reach/armor pressure. Healing, if added, is a limited explicit ability, not full Cleric access |
| Vanguard → Battle Captain | Warrior | Fighter/Vanguard, Centurion, General | Frontline formation support: spends its turn preparing a bounded defense benefit for nearby allies; no passive extra turns for the whole company |
| Arbalist → Shieldshooter | Ranger | Arbalist/Shieldshooter | Crossbow-and-shield ranged guard; equipment/action-rate tradeoffs separate it from precision Hunter and magical Storm Archer |
| Dragonkeeper → Dragonmaster | Ranger | Dragon Tamer/Master, Dragoner/Master | Specialized support for actual recruited drakes/dragons; neither spawns free dragons nor duplicates Beastkeeper buffs |
| Enchanter → Artificer | Wizard | Doll Mage/Master, Doll Master/Enchanter | Weapon imbuement and construct maintenance/support; short enchantments consume actions/mana and cannot permanently mint item value |
| Valkyrie → Dawn Marshal | Warrior | Valkyrie/Freya, Valkyrie/Muse, Crusader/Valkyria | Positive-alignment spear-and-restoration hybrid; less reach damage than Lancer and less shielding/healing than Knight/Priest |

### Delivery constraints that make these real choices

Arbalist needs a real crossbow identity and validated shield compatibility.
If existing shooting weapons prohibit that equipment combination, implement an
explicit compatible weapon subtype/profile and off-hand rules first. Define
whether block is legal while firing and its turn/reload cost; do not merely
rename a bow. Help must explain the actual tradeoff.

Monk's innate/unarmed attacks go through normal equipment fallback, reach,
defense, crit, status and wound rules. Its weapon restrictions must leave it
functional without a hidden expensive item. No free counterattack chain.

Enchanter's combat imbuement is a transient modifier with explicit stacking
rules for poison, sharpening, oil and existing enchantments. It grants no saleable
permanent enchantment from a repeatable combat ability. Repair requires parts,
family-specific validation and exactly-once consumption; automatic role support
cannot heal every construct to full between fights.

Dragonkeeper is a candidate sibling path distinct from general Beastkeeper only
if dragons have meaningful upkeep/breath/recovery systems to support. Otherwise
ship dragon affinity as a Beastkeeper specialization rather than a redundant
class. A dragon without a handler still functions; handlers are useful choices,
not a compulsory tax on owning a creature.

## Commander and prestige ideas

Centurion, General, Lord, Prince and Princess suggest leadership identities,
but a title alone is not a new base lineage. Keep Battle Captain as the concrete
ordinary support-class proposal above. Consider a later **commander specialization**
for different current classes, with visible prerequisites and a single bounded
order benefit, rather than a mandatory “best leader class”. It must use existing
order/action limits and cannot enlarge the roster, grant free turns or silently
change other characters' classes. This overlay requires its own approval.

Dreadnought, Angel Knight/Seraph, Lich and Vampire are good rare prestige/recruit
candidates. They need authored quests/contracts, recovery and counterplay before
eligibility is offered. Prestige may add a graph edge or a unique recruit profile;
choose one explicit model per class. No player is converted to undead or an
angel automatically because alignment crosses a number. Do not reproduce gender
restrictions or character-exclusive access from the source rosters by default.

## Additional creature families and growth choices

Expand the future creature catalogue with the following optional candidates.
Creature families use their own authored evolution rules, equipment and recovery;
these are not automatic human class promotions.

| Family/path | Role | Required tradeoff or dependency |
|---|---|---|
| Hellhound → Cerberus | Fire/pursuit frontline pressure | Multiple heads become a bounded attack pattern within the action budget, not three free turns |
| Faerie → Sylph | Fragile magical support and cleansing | Limited healing/control and clear physical vulnerability |
| Imp → Demon | Offensive magic/control | Upkeep and explicit resistances; individual disposition, not mandatory evil alignment |
| Hawkfolk → Sky Warden / Night Raven | Flying humanoid physical or disruption role | Flight/reach rules and limited equipment; family-specific graph keeps the original five creation choices intact |
| Mermaid → Tidecaller | Water support and recovery | Land viability and water-route rules before recruitment; no inaccessible party trapped by travel |
| Lycanthrope → Werewolf or Werebear specialization | Pursuit or durable melee | Persistent controlled form with gear/recovery rules; avoid automatic day/night stat swings initially |
| Ghost → Wraith | Spectral interference | Bounded physical resistance, not universal immunity; specialist healing and death rules |
| Gorgon → Stone Oracle | Slow/control specialist | Limited petrifying status with saves/immunity/duration, never permanent free instant kills |
| Cockatrice → Basilisk | Fragile beast control | Same bounded status framework; distinct body/attack profile from Gorgon |
| Sphinx → Riddlekeeper | Defensive magical beast/support | Rare authored recruit and costly upkeep; enough differences from Gryphon to justify a family |
| Giant → Frost Giant / Fire Giant | Heavy physical specialist | Fit one slot/cell and balance there until larger footprints are separately designed |
| Young Dragon → affinity branch | Breath and resistance specialist | Family-specific growth, logistics, clear weakness and bounded area attacks |

For Hellhound, **Cerberus replaces the earlier provisional Infernal Hound name**
as the preferred proposed evolved form. Keep existing Stone/Runic and Iron/Ward
Golem paths; Baldr Golem and Rock Golem inform variant traits without requiring
another mechanically identical family.

Ghost/Wraith are creature identifiers; the Rogue Shadowdancer/Phantom branch
remains a living humanoid skill identity. Use stable distinct IDs and descriptive
UI labels so shared fantasy names do not accidentally share race/class behavior.

Lycanthropy is a family/form trait. Werefox/Wereowl/Werelion can be future Beastfolk
species/roles, not all mandatory evolution destinations for every lycanthrope.
Distinct form names require distinct authored traits and equipment compatibility.

### Dragon branches

Prefer a limited affinity tree rather than importing every named dragon as a
separate selectable class. Candidate Young Dragon branches:

- Ember Drake → Fire Dragon: burning pressure; explicit vulnerability/counterplay.
- Storm Drake → Thunder Dragon: magical disruption; bounded breath/cooldowns.
- Earth Drake → Earth Dragon: physical protection; slow turn/movement tradeoff.
- Frost Drake → Frost Dragon: slowing/control; immunity and duration caps.
- Umbral Drake → Shadow Dragon: affliction and drain; reduced defensive/support role.

Platinum/Gold Dragon, Hydra, Quetzalcoatl, Bahamut/Tiamat-style creatures and
Zombie Dragon are later rare species or prestige encounters, not ordinary
level-30 rewards. Choose original Ashveil names for setting-specific legendary
creatures when authoring content. A dragon's affinity is explicit species/class
data, never inferred from its color/name. Alignment gates on sentient creatures
are optional authored rules, not a default inferred from scales or ancestry.

## What the reference should refine, rather than duplicate

| Reference cluster | Existing Ashveil home |
|---|---|
| Fencer, Swordfighter, Sword Master/Swordmaster, Samurai | Duelist/Swordmaster; styles can vary without four identical branches |
| Phalanx, Cataphract, Hoplite/Legionnaire | Hoplite/Legionary or Armiger/Ironclad according to reach versus armor role |
| Wild Man, Evil One, Black/Dark/Doom Knight | Reaver/Dread Knight; distinguish Berserker's risk-based offense |
| Ninja/Ninja Master, Thief/Rogue | Assassin/Nightblade or Shadowdancer/Phantom, based on venom versus evasion |
| Archer, Diana, Hunter/Sniper | Hunter/Marksman; ranged identity should come from actual mechanics |
| Mage, Sorcerer/Sorceress, Wizard/Warlock | Keep Ashveil Sorcerer burst, Arcanist efficiency and Warlock affliction identities explicit |
| Cleric/Priest/Bishop, Shaman/Druid | Retain Ashveil Priest, Shaman and Druid as distinct proposed roles, not imported source promotion equivalence |
| Beast Tamer/Master, Beastman | Beastkeeper/Beastmaster for handler gameplay; species is separate |
| Gryphon/Wyvern Knight/Master | Later mounted/flight design, distinct from a recruited Gryphon or Drake companion |

Pumpkinhead/Gremlin and seasonal variants can become setting-specific rare
recruits or events if they have a compelling role; no need to inflate launch
scope. Octopus/Kraken wait for aquatic travel/combat viability. Coffin and daytime
forms belong to lifecycle/form records, not independently recruitable classes.
Boss and internal-only entries (Dark Marquess variants, Overlord, Death Templar,
Gatekeeper and similar) are useful enemy ability profiles. They need not all be
player promotions. Different weapons under one title can use different configured
profiles rather than duplicate IDs with no new class progression.

## Plan additions and acceptance

Prioritize Arbalist, Grappler/Monk, Lancer and Enchanter as distinct additions
after the core class foundation and Sorcerer. Gate Dragonkeeper, forms, flight,
aquatic and prestige classes on their respective dependencies. Keep a role and
capability matrix listing each branch's signature, equipment, costs, counters,
recruit source and planned/shipped status; merge or defer indistinguishable rows.

Future tests must cover crossbow/shield legality, unarmed fallback, reaction/order
budgets, temporary enchantment persistence/expiry, handler stacking, creature
family restrictions, form/equipment transitions, physiology/recovery, stable IDs
and no free unit/action/gear/reward duplication. Integrate class/creature help,
browser comparison and tutorial pointers with each delivered bundle. Current work
is documentation only and does not schedule or implement these proposals.
