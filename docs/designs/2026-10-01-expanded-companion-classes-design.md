# Expanded companion classes and future creature recruits

Status: owner-requested expansion, 2026-10-01. Companion class variety, including
Sorcerers, is requested; nonhuman companions such as Hellhounds and Golems are
an eventual direction. This is a proposed catalogue and delivery design, not
implemented classes or authorization to ship every proposed ability.

Companion to [branching progression](2026-10-01-branching-class-progression-design.md)
and its [implementation plan](../plans/2026-10-01-branching-class-progression-plan.md).

## Inspirations and research limit

Ogre Battle 64 inspires branching unit development, alignment-sensitive routes,
contrasting physical/magical roles and monster recruits. Unicorn Overlord
inspires sharply differentiated defenders, spear users, archers, healers,
affliction/support casters, reactions and company synergies. These are broad
design inspirations, not a promise to copy their rules or class restrictions.

Attempts to consult these class references were blocked by this environment's
network policy on 2026-10-01:

- https://ogrebattlesaga.fandom.com/wiki/Classes_(Ogre_Battle_64)
- https://unicornoverlord.fandom.com/wiki/Classes

The catalogue below is an Ashveil proposal informed by general knowledge, not
an exhaustively verified list from either game. Recheck accessible official or
reliable class references before finalizing names and signatures; do not present
invented Ashveil classes as original classes from those games.

## Supplied reference expansion

The owner subsequently supplied a detailed roster covering both Ogre Battle
games and Unicorn Overlord. See [further reference extractions](2026-10-01-class-reference-extractions.md)
for seven additional humanoid candidate paths, commander/prestige ideas, new
creature families, dragon affinities and explicit duplicate/dependency handling.
That analysis uses the supplied list without claiming external verification.
Counts below describe the initial catalogue; extractions are additional candidates.

## Roster model and scope

Keep the original five creation lineages. Preserve the earlier fifteen advanced
and fifteen elite class ideas; add the twenty advanced paths and twenty elite
continuations below. The combined human/humanoid design catalogue has **five
base classes, thirty-five advanced classes and thirty-five elite classes**.
This is breadth for staged content delivery, not seventy new classes at once.

The two promotion thresholds remain proposed levels 10 and 30. Each row is a
parent-to-child path within its lineage. Positive means individual alignment
+30..+100; negative −100..−30; unrestricted accepts any alignment. These gates
are proposals, including the moral interpretation of particular occupations.
Keep multiple unrestricted routes: a companion's build should not be determined
solely by a single morality number.

Each class needs a different combat decision, target or preparation purpose.
Names, armor preferences or a renamed +damage passive alone do not justify a
new selectable class. Prefer one signature plus inherited base capabilities,
with elite developing the same identity. Player and companion effective-class
resolution stays shared; this expansion prioritizes companion choice without
creating a second incompatible class engine. Humanoid paths can be available
to the leader too unless a specific future recruit-only class is declared.

## Additional Warrior paths

| Advanced | Gate | Elite | Distinct role and proposed signature |
|---|---|---|---|
| Hoplite | Unrestricted | Legionary | Spear-and-shield formation anchor; intercept a reachable column threat, trading mobility/attack for protection |
| Armiger | Unrestricted | Ironclad | Heavy shield defense; braces against physical focus but remains vulnerable to magic and disruption |
| Berserker | Unrestricted | Juggernaut | High-risk sustained melee; attack commitment trades active defense for damage, distinct from Reaver morale pressure |
| Spellblade | Unrestricted | Rune Knight | Weapon/magic hybrid; spends mana on a conditional elemental strike instead of receiving a free spell and attack |

Knight protects a ward; Hoplite controls a formation lane; Armiger absorbs
physical pressure. Keep those differences explicit. Spellblade's magic access
must be narrowly configured rather than opening every wizard school.

## Additional Rogue paths

| Advanced | Gate | Elite | Distinct role and proposed signature |
|---|---|---|---|
| Saboteur | Unrestricted | Demolitionist | Disrupts armor and wind-ups; sabotage action competes with a normal attack, with no implied explosive inventory system |
| Bard | Unrestricted | Troubadour | Company morale/support timing; a song spends its action and cannot stack endlessly across multiple singers |
| Shadowdancer | Unrestricted | Phantom | Evasive skirmisher; uses openings to disengage or counter within legal formation reach, without invulnerability |
| Cutthroat | Negative | Executioner | Pressure against heavily wounded targets; conditional finishing strike distinct from Assassin poison specialization |

Scout remains reconnaissance/opening utility, Duelist blade/parry precision,
and Assassin venom/setup. No branch reintroduces manual attack micromanagement,
charm recruitment or guaranteed kill-on-threshold effects.

## Additional Ranger paths

| Advanced | Gate | Elite | Distinct role and proposed signature |
|---|---|---|---|
| Beastkeeper | Unrestricted | Beastmaster | Supports recruited beast members; directs one coordinated opportunity without summoning a free extra unit |
| Skirmisher | Unrestricted | Outrider | Mobile ranged/melee flexibility; changes attack profile by legal reach rather than bypassing formation |
| Trapper | Unrestricted | Snaremaster | Camp/route preparation and limited battlefield slowing; supplies and action costs explicit, not an invisible minefield |
| Storm Archer | Unrestricted | Tempest Archer | Mana-using ranged magic pressure; elemental shot replaces a normal shot and has defined immunity handling |

Outrider is a foot-capable skirmisher, not a promise of mounted combat. Beastkeeper
ships only when creature members exist. Hunter stays single-target precision;
Warden ally protection; Stalker pursuit/wounded pressure. Scout (Rogue) and
Pathfinder remain reconnaissance identities, not aliases for Ranger damage paths.

## Additional Cleric paths

| Advanced | Gate | Elite | Distinct role and proposed signature |
|---|---|---|---|
| Oracle | Unrestricted | Seer | Anticipates threats; bounded protective response to a telegraphed strike, not guaranteed prediction or evasion |
| Shaman | Unrestricted | Spiritkeeper | Buff/debuff specialist; weakens enemy offense while contributing modest recovery |
| Exorcist | Positive | Inquisitor | Specialized anti-undead/curse cleansing; useful support beyond undead-only encounters |
| Druid | Unrestricted | Elder Druid | Regeneration and nature support; gradual healing instead of Priest burst restoration |

Priest is direct recovery, Chaplain martial support, Hexer offensive affliction.
Separate Oracle's defensive prediction from Theurgist's arcane protection.
No global time manipulation, free resurrection, persistent spirit summons or
unlimited cleanse is granted by these names.

## Additional Wizard paths

| Advanced | Gate | Elite | Distinct role and proposed signature |
|---|---|---|---|
| Sorcerer | Unrestricted | High Sorcerer | High-power, costly magical burst; longer chants/mana commitments create interrupt risk |
| Witch (2026-10-05 proposal: renamed Hexweaver, elite Malison; the Witch and Coven Sage names move to the new Witch lineage) | Unrestricted | Coven Sage | Status manipulation and affliction setup; trades direct damage for control, distinct from Warlock life drain |
| Elementalist | Unrestricted | Elemental Savant | Fire/frost/lightning specialization; picks one affinity with strengths and resisted matchups |
| Illusionist | Unrestricted | Mirage Weaver | Misdirection and defensive interference; finite effects, no unhittable illusion loop |

Arcanist remains dependable efficient damage; Sorcerer trades efficiency for burst.
Theurgist protects allies; Witch controls; Warlock drains/afflicts; Elementalist
specializes against resistances; Illusionist disrupts targeting under battle
ownership rules. Elemental affinity is one class-internal choice, not three more
near-identical class IDs. Switching affinity, new schools and area attacks need
explicit later rules. Sorcerer is an unrestricted path, not an automatic moral
label or an inherited skill-training loophole.

## Recruiting and choosing many classes

Keep `class paths [member]` manageable: list the current lineage's available
advanced routes first, with role, gate, requirements and elite continuation.
Allow role filters (defense, damage, control, healing, support) as presentation,
not new eligibility. Mark future branches unavailable until their signatures
and help ship. Show comparisons of gained/replaced abilities and equipment.

Companions promote independently through the existing explicit preview/confirm
flow. Recruit offers can eventually contain advanced members with a valid
lineage/path record and transparent level, role, alignment and price. Recruitment
does not auto-promote the rest of the company or change its cap. Do not solve a
large catalogue by adding every branch to every trainer's unstructured menu.

## Future nonhuman humanoids

Separate species/race from class lineage: an elf can be a Ranger or Wizard,
a dwarf a Warrior or Cleric, and another species can use the same graph where
its physiology and authored culture permit. Proposed future recruitment families:
Elves, Dwarves, Orcs and Beastfolk. Exact playable/recruitable species, starter
stats and cultural specialties remain a separate content decision.

Species can provide bounded physiology traits and equipment compatibility;
class provides the automatic role and promotion tree. Alignment remains individual:
no species is automatically evil, loyal, immune to desertion or locked to one
moral branch. Do not retroactively turn existing races into class IDs.

## Future creature members

Creatures are durable company companions recruited through authored contracts,
quests, handlers or construct makers, not transient combat pets or mass summons.
Their species/family owns innate attacks, allowed equipment, upkeep and recovery;
their creature-class path owns role growth. A creature cannot promote into Knight
or Sorcerer solely because it meets a level/alignment gate.

| Recruit family | Proposed evolved class/form | Combat role | Principal tradeoff |
|---|---|---|---|
| Hound | Warhound | Pursuit and exposed-target pressure | Light protection; needs ordinary food and rest |
| Hellhound | Cerberus | Fire/terror pressure | Fire resistance paired with a defined vulnerability; no blanket poison immunity |
| Stone Golem | Runic Golem | Physical formation anchor | Slow actions, poor magical defense and costly repairs |
| Iron Golem | Ward Golem | Shielding/protection against physical focus | Heavy transport load and maintenance; vulnerabilities defined in data |
| Gryphon | Storm Gryphon | Reach and back-line harassment | Fragile to ranged pressure; flight does not bypass every formation rule |
| Drake | Elder Drake | Breath and durable frontline pressure | Expensive upkeep; breath spends a turn and has limited reach/cooldown |
| Treant | Ancient Treant | Protection and gradual recovery | Fire vulnerability and slow actions |
| Wisp | Lantern Wisp | Light and magical support | Low physical durability; no free unlimited mana/light |
| Skeleton | Bone Sentinel | Attrition-focused guard | Restricted healing and explicit restoration costs |
| Revenant | Grave Warden | Life-draining melee | Specialist recovery and social/recruitment restrictions, not automatic company hostility |

The listed evolved forms are suggestions; creature thresholds and alignment gates
are designed separately, not inherited blindly from human level-10/30 promotions.
Hellhound is its own recruit family, not a normal dog automatically becoming
infernal. Golems can have role upgrades without changing physical material.
These units require authored recruit availability and budgeted stat/ability
profiles before release.

### Creature system requirements

- Keep leader plus four companion slots initially. A creature occupies one slot
  and one formation cell; balance its footprint to that limit. Large multi-cell
  units and Ogre Battle-style size budgets would be a separate formation redesign.
- Use existing race size/disabled equipment-slot support as foundations, but audit
  actual item validation. No dog wearing boots or golem wielding every weapon.
  Define allowed collars, harnesses, cores or innate-only slots without deleting
  saved gear or treating innate weapons as free tradable equipment.
- Describe innate attacks in ordinary combat resolution: reach, damage type,
  defense, statuses, crit rules, cooldown and turn cost. A fire breath is not an
  extra out-of-turn area attack. Flying reach changes need a separate approved
  rule; flight never automatically skips travel terrain or hostile encounters.
- Define physiology per family: food/water/fatigue, exposure, wounds, poison,
  morale, alignment/loyalty and rest. Constructs may use bond/stability in place
  of ordinary morale; they cannot disappear through accidental generic human
  desertion handling. Undead and constructs use explicit susceptibility data.
- Define recovery by family before recruitment: biological creatures can use
  compatible healing/treatment; golems require repair/parts; undead require
  configured restoration. Integrate resurrection, lost-member rescue allowance
  and death level loss deliberately rather than granting immortal free units.
- Count weight, supplies and cargo/transport implications; golems cannot bypass
  travel capacity by being called companions. A Hellhound is not simultaneously
  a mount, pet and roster member. No creature grants free cargo capacity.
- Persist family, class/form, individual identity, level, gear/innate loadout,
  upkeep, bond/disposition, vitals and recovery state through save/load/copyover.
  Recruiting, evolving and repairs follow exactly-once ownership rules.
- Explain settlement/recruit availability as explicit local rules. Avoid blanket
  punishments for choosing a creature; preview any access restriction before
  recruiting, without inventing a new faction crime system.

## Staged delivery and coverage

1. Keep the earlier Warrior pilot and effective-class architecture.
2. Implement the original all-lineage advanced paths, then expand by role bundles
   with Sorcerer prioritized among new caster classes.
3. Add distinct additional advanced signatures before elite upgrades. Beastkeeper,
   Trapper and poison specialties wait for their explicit dependencies.
4. Introduce nonhuman humanoid recruits using the proven shared graph.
5. Pilot Hound and Stone Golem with complete biological/construct lifecycle rules;
   then Hellhound and other creature families. Undead/flight arrive after their
   recovery/reach rules are designed and tested.

Keep each bundle scoped and independently reviewed. Balance total action economy,
formation synergy, control uptime, healing and upkeep, not individual damage only.
The catalogue is optional content breadth and does not reorder the owner's
current Phase 33/30g work without scheduling.

Acceptance adds real promotion and combat tests per branch, no leakage across
lineages/schools, affordable unrestricted choices, inherited specialists,
recruit previews and distinct role behavior. Creature acceptance includes actual
recruitment, roster limits, formation/reach, allowed equipment, innate damage,
supply/weight accounting, exposure/poison/wounds, morale/bond, healing/repair,
death/resurrection, dismissal and durable recovery. Verify both text and browser
surfaces, indexed class/creature help and tutorial pointers. Publish no help
claiming unavailable classes are selectable.
