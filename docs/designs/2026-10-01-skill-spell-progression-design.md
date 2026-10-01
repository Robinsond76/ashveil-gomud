# Skill, utility and spell progression review

Status: owner-requested review/design/plan, 2026-10-01. Documentation only.
New unlocks, balance, loadout choices and costs below are proposed defaults.
Do not override previously settled 33f retirement/specialist decisions.

Related: [class progression](2026-10-01-branching-class-progression-design.md),
[expanded classes](2026-10-01-expanded-companion-classes-design.md),
[33f specialists](2026-10-01-phase-33f-company-specialists-design.md),
[camp consumables](2026-10-01-camp-consumables-design.md).

## Current implementation review

Inspected local base: 54c11ac8 (33f1), plus planning documents. Remote freshness
is unverified after the earlier proxy failure. This is a source review, not
runtime verification or a claim that every inherited command works unchanged.

Skills are YAML-defined lowercase IDs with default maximum rank 4. Protection
is capped at 3. Player ranks live in Character.Skills; training uses points,
with the next rank costing its rank (1+2+3+4 = 10 total for rank 4). Training
requires a room's offered rank range and archetype permission. Level-up grants
points according to configured intervals; reaching a character level alone
currently does not grant every skill or spell.

Companion utility ranks derive from archetype and character thresholds
1/10/20/30. Player utility ranks read trained skill ranks. Player automatic
Tackle/Opening Strike/Aimed Shot read Brawling/Skulduggery/Track respectively;
companion unlocks compare archetype IDs. Cleric and Wizard companion spell
lists currently unlock heal/mm at level 1 and healall/sparks at level 5.
Player spells use class schools, grants and spellbook learning. Cast proficiency
is not a replacement for explicit spell ownership or action legality.

### All twelve shipped skill definitions

| Skill ID | Current source role | Review direction |
|---|---|---|
| brawling | Recover, throw, tackle and disarm description; automatic Tackle exists | Audit actual command restrictions; explain automatic combat and retain useful out-of-battle recovery only if it respects camp/rest rules |
| dual-wield | Equipment permission and hit penalties by rank | Retain passive; inspect both-hand/action calculations and clarify what ranks 2/3 actually add |
| skulduggery | Locks, traps and automatic Opening Strike | Retain; add approved Keen Eye/Haggle through 33f2 |
| track | Legacy individual tracking; automatic Aimed Shot | Retain ID; replace stock command with approved Read the Trail and Pathfinder in 33f2 |
| search | Legacy hidden-object search | Pending retirement in 33f2 with Keen Eye replacement; do not build a new Search tree |
| cast | Spell proficiency and learning | Retain; show school/unlock/resource rules separately |
| protection | Aid, capped rank 3 after Pray removal | Review aid versus persistent death/resurrection and automatic protection; never treat permanently dead companions as merely downed |
| cooking | Recipes at a hearth | Retain shared trade skill; camp cooking is the approved future 33f3 extension |
| enchant | Persistent object enchantment | Audit item/resource limits separately from proposed temporary Enchanter abilities |
| inspect | Item details | Retain utility; improve useful disclosure rather than hide essential costs or class eligibility |
| map | Mapping range/details and memory | Retain; supports expedition navigation without bypassing travel |
| trading | Commerce commands/appraisal | Retain shared trade skill; audit inherited global auction behavior and avoid confusing it with rogue Haggle |

The source descriptions of Brawling, Track and Search need reconciliation with
shipped automatic-combat restrictions and pending replacements. Do not claim
stale skill prose is the final gameplay contract. Review rank-by-rank handlers,
help, trainers and profession titles together; a paid rank must have an actual
visible benefit, not merely a higher number.

### All eleven shipped spell definitions

| Spell IDs | Current role | Review direction |
|---|---|---|
| heal, healall | Single-member/company restoration | Keep coordinated automatic healing, company scope and wound limits |
| tend | Wound treatment | Field/camp medical action, not ordinary battle healing |
| curepoison | Poison removal | Review automatic spell use; distinguish cure from future preventative draught |
| mm, sparks, hex | Offensive conjuration | Retain as available repertoire; audit grants/learning and automatic strategy coverage individually |
| floatinglight, illum | Illusion light utilities | Keep company lighting without stacking duplicate auras or granting permanent free light |
| poly | Random target transformation | Audit before any progression unlock: ownership, equipment, companion records and legality; do not reintroduce retired Change Form through an unsafe shortcut |
| aidskill | Skill-backed aid spell | Internal action implementation, not automatically a freely learnable Cleric spell |

Presence of a spell file does not prove automatic AI selects it. Inspect each
script, school, scope, chant, cast gate and strategy path before adding it to a
class unlock list. Charm/taming spells and Portal are not new progression options.

## Progression model

Keep three concepts visible and separate:

1. **Learned/trained skills:** existing player point investment and recipes.
2. **Level/class unlocks:** derived class repertoire and companion capability ranks.
3. **Prepared combat choices:** the unlocked options a member's automatic role uses.

Retain approved specialist rank thresholds 1/10/20/30 and best-member rules.
Do not replace player training with companion-style automatic ranks silently.
Propose class ability milestones at levels **1, 5, 10, 20, 30, 40 and 50**:

| Level | Proposed progression event |
|---|---|
| 1 | Base automatic role and class essentials; existing starter grants remain |
| 5 | A second basic combat option/spell, matching existing caster milestones |
| 10 | Advanced promotion signature if chosen; specialist rank 2 for companions |
| 20 | Branch improvement or alternative; specialist rank 3 |
| 30 | Elite signature if chosen; specialist rank 4 |
| 40 | Choose one modifier for the branch signature (efficiency, control or reliability) |
| 50 | Refine that modifier within explicit caps; no extra free actions |

Unpromoted characters keep base growth but cannot obtain a branch signature by
level alone. Old high-level characters see earned choices without automatic
moral promotions. Death below an unlock threshold temporarily disables that
level-gated option; promotion identity and chosen modifiers remain saved.
Regain never awards the same choice, spell grant or training points twice.

Level-40/50 choices are proposals, not a new skill-point currency. One alternative
is active per milestone; settling switching rules is required before launch.
No every-level spell flood: small repertoires with meaningful roles are easier
to understand and balance than dozens of nearly identical attacks.

## Field and camp capability catalogue

Preserve the owner's approved 33f capabilities and formulas. Specialist actions
use the best own living/present eligible member, with leader then member-ID tie
breaks, named attribution, no stacking and autoskill toggles where applicable.
They do not recruit allies/pets as free specialists, operate in battles, or
consume items in battle. Do not restore group stealth, solo scouting, medic or
quartermaster assignments, Portal, Tame, Change Form, Scribe, Pray or PvP theft.

| Capability | Source/rank progression | Use and status |
|---|---|---|
| Read the Trail | Ranger/Track; approved ranks 1–4 | Nearby public enemy information and travel ambush warning/avoidance; 33f2 planned |
| Pathfinder | Ranger/Track; approved ranks 1–4 | Less rough-terrain walking strain; 33f2 planned, no travel speed shortcut |
| Keen Eye | Rogue/Skulduggery | Secret exit discovery; 33f2 planned |
| Locks and Traps | Rogue/Skulduggery | Existing utility, with stronger validated checks at trained/derived ranks |
| Haggle | Rogue/Skulduggery | Approved bounded market prices; 33f2 planned, no resale arbitrage |
| Weather Sense | Wizard/Cast | Approved forecast detail by rank; 33f2 planned |
| Company Light | Wizard/Cast | Existing light utility; quality/coverage improvements require actual lighting tests |
| Camp Watch | Warrior rank | Approved camp raid detection; 33f3 planned |
| Field Smith | Warrior rank | Approved longer whetstone edge; 33f3 planned |
| Vigil | Cleric rank | Approved rest loyalty benefit; 33f3 planned |
| Forage | Ranger rank | Approved food/water gathering at eligible rest sites; 33f3 planned |
| Cooking | Trained trade skill | Approved camp hearth extension; recipes unlocked by rank, no assumed companion Cook implementation |
| Alchemy | Proposed trained trade skill | Future poison/draught/salve recipes; needs kits, ingredients, ownership and deterministic yields |
| Construct Repair | Enchanter/Artificer capability | Future creature repairs using parts; no free human/construct full heal |
| Beast Care | Beastkeeper/Dragonkeeper capability | Future creature readiness/compatible treatment using supplies; no extra roster member or guaranteed free revive |
| Cartography | Existing Map rank | Useful route/detail disclosure; map information is not instant travel |
| Appraisal | Existing Inspect/Trading | Transparent item value/properties; core requirements remain visible without a skill |

Cooking retains the existing trained-leader rule until explicitly extended.
Alchemy uses a single new data-driven skill only if approved; do not create
separate redundant brewing/poison/salve skills. Recipes, not hidden moral class
gates, distinguish special consumables. Specialist benefits inherited from base
lineage continue after promotion (Knight still has Warrior Field Smith access).

## Offensive abilities and spells by lineage

These are candidate milestones, not finalized damage/cooldown numbers. Existing
33e abilities remain baseline unless a reviewed migration explicitly changes them.

| Lineage | Base repertoire | Advanced/elite examples |
|---|---|---|
| Warrior | Tackle; validated ordinary weapon/passive choices | Knight Brace, Lancer Piercing Thrust, Berserker Reckless Strike, Monk Grapple; elites improve their own signature |
| Rogue | Opening Strike; legal blade exploitation | Duelist Riposte, Saboteur Armor Sabotage, Bard Rally, Assassin Venom Strike after poison delivery exists |
| Ranger | Aimed Shot; ordinary ranged fallback | Hunter Focused Shot, Warden Cover, Arbalist Guarded Shot, Storm Archer elemental shot |
| Cleric | Minor Heal; group heal when unlocked; Tend outside battle | Priest stronger efficient healing, Druid Regeneration, Exorcist Cleanse, Shaman Weakening Hex, Hexer Wither |
| Wizard | Magic Missile; Sparks when unlocked; field light | Sorcerer Arcane Lance (costly long chant), Witch Slow/Expose, Elementalist affinity spell, Illusionist Veil, Enchanter Imbue |

Names such as Weakening Hex must use distinct IDs/effects from the existing
`hex` damage spell; naming similarity must not silently rewrite old spells.
Cleanse targets eligible statuses and never erases all wounds or permanent death.
Bard songs, covers, ripostes and elemental shots have explicit action/reaction
costs; no bonus free turn. Healing follows existing friendly scopes and pending
heal reservations. No repeated stun/silence/invulnerability loops.

Every ability/spell specification must list: source class and minimum level,
player trained prerequisites if any, school, target/scope/reach, equipment,
action cost, mana/material cost, chant time, cooldown, exact effect/duration,
stacking/immunity, AI trigger/priority, fallback and narration. Set quantities
against the balance harness, not arbitrary exponential rank scaling. A specialist
skill does not improve every combat spell just because both use Cast rank.

## Controls and automatic combat

Proposed `abilities [member]` view groups Field, Camp and Combat, showing learned,
ready, locked and planned options with requirements/costs and next milestone.
Reuse `company specialists` rather than inventing another specialist assignment.
Link existing skills/train/spellbook views; keep training and learning explicit.

Combat options run through strategy automatically. Use the existing ability toggle
and role controls first. Propose a small preparation profile for casters:
conserve, balanced or aggressive, with exact mana reserve/priority rules specified
before implementation. Do not add manual per-turn skill commands, general AI
scripting or an unrestricted forty-slot loadout. Field spell/manual utility actions
are allowed only in their documented out-of-battle contexts. A battle spell that
cures poison remains a spell action, not permission to use consumable items.

Audit the earlier poison proposal's combat antidote item against the approved 33f
“no items in a fight” rule before implementing it: keep curative consumables in
field/camp and use eligible automatic cure spells in battle unless the owner
explicitly changes that rule. This review does not silently authorize the exception.

## State, dependencies and acceptance

Store learned/trained skills and owned spells in their existing durable records.
Derived unlocks come from effective lineage/class/current level; choice modifiers
belong to the corresponding player/companion progression owner. Use stable IDs,
old-save defaults, atomic choice/grant commits and exactly-once grant/refund markers.
Do not keep a second independent ability-level ledger that drifts from class state.
Restart/copyover must not refill mana, clear owed costs, duplicate choices or bypass
cooldowns; use existing runtime/durable action policies or explicitly migrate them.

Depend on the then-current 33e/33f implementation, class promotion graph,
30g4/30g5 progression/action budget, 33h continuity, camp crafting and creature
lifecycle. Core review/help repairs can ship before broader new classes, but no
consumer assumes planned dependencies already exist.

Before coding, audit all twelve skills and eleven spells through actual callers,
rank gates, trainers, grants, scopes, scripts and AI. Ship indexed `help abilities`,
`help skills`, `help spellbook`/learning guidance and affected class/field/camp
pages, hub links and tutorial pointers. Show useful rank benefits and upcoming
unlocks in text/browser, without publishing unavailable abilities as usable.

Acceptance tests drive training/learning and real player/companion turns, utility
commands and rest/crafting callbacks. Cover promotion inheritance, school legality,
actions/cooldowns/chant breaks, resources, best-member resolution, dead/absent
members, supplies, rank boundaries, death/regain, failed saves, old saves and
copyover. Verify no retired mechanic returns, no world-time advancement, no other
company's state changes and no duplicate rewards/grants. Measure damage/healing,
control uptime, mana sustainability, encounter frequency and preparation economy.
