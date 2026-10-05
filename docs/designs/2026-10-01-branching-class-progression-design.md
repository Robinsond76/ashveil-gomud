# Branching class progression

Status: owner-requested review, design and plan, 2026-10-01. Inspired by Ogre
Battle's class paths and alignment, and Mount & Blade's visible upgrade choices.
No gameplay implemented or phase assigned. Branch names, levels, alignment gates,
abilities and switching rules below are proposed defaults for review.

**Owner update (2026-10-05):** the first promotion at **level 10** is now an
owner decision, and class promotions and routes move up the roadmap (see
Project Status, "Roadmap priorities"). The owner also asked for a **Witch**
with level-scaling hexes. As the [level impact and class power design](2026-10-05-level-impact-class-power-design.md)
proposed, the Witch is a sixth base lineage (owner decision), with proposed
Hedge Witch, Coven Sage and Hag routes. It also proposes talent choices at levels 5, 15 and 25 between
promotions. The rest of this document's proposals stand until reviewed with
that design.

**Owner update (2026-10-05, faith routes):** clerics are protected priests
and fighting healers are warriors, with good and evil routes. The draft
[faith routes design](2026-10-05-faith-routes-design.md) replaces this
table's cleric rows (Priest → Hierarch with an Angel, Druid → Elder Druid,
Blood Priest → Demonologist with a Demon) and the warrior's evil route
(Blackguard → Dread Knight instead of Reaver), and adds Lay on Hands to
Knight → Paladin.

## Review of current progression

The inspected local checkout is based on 54c11ac8 (33f1), plus recent planning
documents; remote freshness remains unverified after the earlier proxy failure.

- Five configured exclusive base archetypes exist: warrior, rogue, ranger,
  cleric and wizard (`modules/archetype/files/data-overlays/config.yaml`).
  The leader chooses once through a durable archetype registry; companions
  have an at-most-once Archetype field in `internal/company/company.go`.
  There is no branching promotion graph in the inspected implementation.
- `internal/strategy/abilities.go` provides automatic Tackle, Opening Strike
  and Aimed Shot. Player unlocks read trained skills, while companion unlocks
  currently compare exact archetype IDs. Cleric/wizard spells are supplied
  through class schools and grants. A promoted ID must not accidentally lose
  all base abilities, spells or utility access because of exact-string checks.
- 33f1 retired utility skills/charm mechanics. Older archetype config comments
  and design documents are historical context, not proof those mechanics remain.
- Alignment uses −100..100; companions have durable alignment and loyalty,
  drift toward company peers, recruitment limits and desertion rules. Promotion
  eligibility must use the promoted individual's alignment, not company average.
- The inspected progression config still contains general level/stat/HP/mana/XP
  formulas. 30g4 proposes stat steps, archetype HP and an XP knee; 33h proposes
  class-weighted companion training and continuity. Those plans are dependencies
  to reconcile with the actual implementation at delivery, not already shipped
  class promotion systems.
- Existing death level loss, highest-level/point protections, starter-kit
  exactly-once rules, companion gear and resurrection must remain intact.

## Core model

Each character has a permanent **base lineage** and one **current class**.
Warrior is a lineage; Knight is its chosen advanced class. Creation still offers
only the five base classes. Promotions follow configured parent-to-child edges,
never silently switch the base lineage or grant all branches at once.

Propose three tiers: base from level 1, advanced available at level 10, elite
available at level 30. A level-40 unpromoted warrior still chooses an advanced
class first; elite eligibility then requires that class's explicit parent path.
No random promotion, auto-choice, level reset, separate class XP or XP fee.
The player chooses for the leader and each companion independently. Delaying a
promotion is allowed; show readiness clearly without repeated notifications.

All promotion choices provide concrete roles, not just names and larger numbers.
An advanced class supplies one signature automatic ability/passive and a bounded
role/stat emphasis; elite develops that signature. Maintain the existing action
budget, range, mana, defense, statuses, cooldowns and tactics rules. Avoid an
extra free action just for being promoted.

## Expanded roster direction

The owner requested broader companion choice on 2026-10-01, inspired by Ogre
Battle 64 and Unicorn Overlord, including Sorcerers and eventual nonhuman units.
See the [expanded companion catalogue](2026-10-01-expanded-companion-classes-design.md)
for twenty additional advanced/elite paths, humanoid species and future creature
recruits including Hellhounds and Golems. The table below is the original core
roster, not the limit of the planned class graph. Ship the wider catalogue in
reviewed bundles once its distinct signatures and dependencies work.

## Proposed paths for all five lineages

Alignment gates: **positive +30..+100**, **negative −100..−30**, and
**unrestricted −100..+100**. The unrestricted route is a valid specialist choice
for any character, not a reward for keeping alignment exactly neutral.

| Base | Advanced (level 10) | Gate | Elite (level 30) | Role and proposed signature |
|---|---|---|---|---|
| Warrior | Knight | Positive | Paladin | Shield/guardian protection; Knight braces for its ward, Paladin adds limited restoration |
| Warrior | Mercenary | Unrestricted | Warlord | Physical disruption; improves a deliberate control strike, elite emphasizes pressure on the chosen target |
| Warrior | Reaver | Negative | Dread Knight | Aggressive melee and morale pressure, traded against protection |
| Rogue | Scout | Positive | Pathfinder | Reconnaissance and safe opening attacks; improves opportunities against exposed foes |
| Rogue | Duelist | Unrestricted | Swordmaster | Precise blade fighting and parry-based ripostes within the normal turn budget |
| Rogue | Assassin | Negative | Nightblade | Finishing attacks and poison delivery specialization, contingent on the separate poison system |
| Ranger | Warden | Positive | Sentinel | Ranged protection of vulnerable allies and responses to threats in legal reach |
| Ranger | Hunter | Unrestricted | Marksman | Accurate ranged focus and advanced Aimed Shot |
| Ranger | Stalker | Negative | Ravager | Pursuit and pressure on wounded targets, without blocking every flee |
| Cleric | Priest | Positive | High Priest | Efficient healing and protection of the company |
| Cleric | Chaplain | Unrestricted | War Priest | Mixed frontline support and restoration; sacrifices peak healing specialization |
| Cleric | Hexer | Negative | Hierophant of Ash | Offensive afflictions and weakened enemy recovery; limited restorative support |
| Wizard | Theurgist | Positive | Archon | Protective arcane support and disruption of hostile magic |
| Wizard | Arcanist | Unrestricted | Archmage | Reliable damage and mana-efficient casting |
| Wizard | Warlock | Negative | Necromancer | Debuffs and life-draining magic; no persistent summons in this initial design |

Names and signatures are provisional. Paladin does not grant mounted combat;
Knight needs no mount. Necromancer does not restore charm or permit unlimited
summons. New spell schools, poison delivery modifiers, morale actions and ally
reactions require explicit ability designs and tests; names do not make those
systems exist. Keep original restoration/conjuration/illusion access working
until the new capability resolver and spell roster are implemented.

The graph may later offer multiple elite choices per advanced class, convergence
or prestige branches. Start delivery with the core three advanced choices and one elite continuation
per branch for every base class, then add the expanded catalogue in bundles. Do not force cross-lineage multiclassing or
race/gender gates into the initial scope.

## Eligibility, alignment and player agency

Promotion checks current level, current parent class, the individual's current
alignment, ownership, presence, living/awake state and a safe preparation context.
Use level and alignment only initially; gear buffs or temporary stat boosts
cannot bypass eligibility. Stat requirements, loyalty requirements, quests and
rare promotion items are later choices requiring content design, not hidden
initial prerequisites.

Alignment gates are checked at promotion, not continuously afterward. A Knight
whose alignment later drifts negative remains a Knight with its abilities; it
may no longer qualify for Paladin when the next decision arrives. No automatic
class stripping or forced conversion. A Reaver who becomes positive can keep
that identity or later retrain when the retraining feature is approved.

The interface shows the exact signed requirement and each member's value.
Do not change alignment just by clicking a class or equipping class gear.
Preserve existing moral-action and companion-drift rules; audit how players can
actually move their alignment before launching alignment-gated classes. If the
world offers insufficient deliberate choices, add explicit quest decisions with
bounded gains and exactly-once reward checks in the content slice. Do not make
kill farming the only way to qualify. Warn about drift when describing a future
promotion requirement, without adding an unapproved new drift formula.

Death can lower level below a promotion threshold without demoting the class.
Gate later abilities by current level; the earned class identity remains. Regain
never repeats a promotion grant or mints training/stat points.

## Commands and promotion workflow

Proposed commands:

```text
class                         # self: lineage, current class, tier and next paths
class paths [self|member]      # full tree with requirements and roles
class promote [self|member] [class]
class promote [self|member] [class] confirm
```

Member selectors follow existing company rules. `company inspect` shows both
lineage and current class; browser member setup offers the same preview/confirm
backend. Existing `archetype` creation/selection remains valid and links to the
promotion view. Do not let `archetype choose` bypass lineage or change an earned
class.

Permit promotions at an established camp or safe settlement, outside combat,
travel and active rest. Camp allows company-wide preparation without requiring
a separate trainer per branch. Preview displays requirements, gained/replaced
abilities, role/stat changes, equipment compatibility and preserved resources.
Confirm revalidates all requirements against current state, including drift
since preview. Repeated confirmation is idempotent.

Promotion costs no gold or consumable initially. Equipment is never gifted,
deleted or silently replaced. A shield-dependent Knight can promote without a
shield, but its preview warns that its signature requires one. Preserve all owned
spells/training in saves; available actions follow the current effective class's
capabilities, not ownership alone. Retain inherited base capabilities by default;
branch-only capabilities do not leak to other branches.

Initial promotions are forward-only. Do not promise unrestricted class switching
or implement a costly retraining loop alongside initial delivery. Design later
settlement retraining as an explicit sibling-branch change with one active class,
preview, eligibility and no starter-kit/point farming. Permanent versus retrainable
branch commitment is an owner decision before player-facing launch.

## Growth, abilities and equipment

Keep lineage-derived baseline HP/mana growth initially; avoid recalculating all
past levels as a new branch's richer growth curve. Add small current-class
modifiers and class-weighted future companion training only once reconciled with
30g4/33h. A modifier is derived from the current class once, never accumulated
again on reconnect, death/regain or repeat promotion. Changing maximum resources
never refills current HP/mana; clamp safely if a maximum falls and respect wounds.
Already-spent player/companion points remain spent.

One effective capability resolver combines lineage inheritance, current branch,
level unlocks and trained player skills. Companion and player differences remain
explicit, but both recognize the same class identity/roles. Audit exact base-ID
comparisons in automatic abilities, spell grants, specialist eligibility, kits,
creation, recruit candidates, training, gear suggestions, HP calculations and
UI. Specialists inherit lineage eligibility unless a later specialist design
specifically narrows it; a Knight still counts as a warrior for Field Smith.

Every advanced signature must be specified before it ships: trigger, target,
legal reach, action cost, equipment, cooldown, mana cost, stacking/immunity,
AI priority and exact effect. Elite upgrades replace or improve their branch's
signature rather than repeatedly stacking duplicate passives. Balance matched
parties and mixed promotions in the existing combat harness; preserve the intended
fight cadence and keep every branch useful at its tier.

## Persistence and migration

Store base lineage, current class, promotion history/schema version and highest
earned tier with stable class IDs. The player's class record belongs in the
archetype registry; the companion's belongs in its durable company record.
Promotion does not change the immutable companion member ID, template, alignment,
loyalty, service, gear, XP, death record or company slot.

Use the respective owner's validated save transaction; roll back on failed save
and announce success only after commit. Derive abilities/modifiers from committed
class state instead of separately applying grants that can be duplicated. Where
new persistent grants are unavoidable, record grant markers with the same durable
owner or a recoverable journal. Do not mutate external engine state while holding
the archetype module lock.

Old saves initialize lineage/current class from their existing archetype and stay
unpromoted, even at high level. Never auto-pick a moral branch. Characters with no
archetype retain the existing selection flow. Unknown/removed class definitions
preserve the saved identity and history, expose an explicit recovery warning and
use a safe lineage capability fallback pending migration; do not silently delete
progress or grant a kit. Recruiters can later offer pre-promoted members only
through validated lineage/path records with clear price/role descriptions.

## Delivery and acceptance

Start with progression/state/UI and one Warrior pilot, then all advanced paths,
then elite signatures. Mark unavailable branches as planned, not selectable empty
classes. No branch is ready until its intended distinct gameplay works. Resolve
30g4/33h overlap and use the latest shipped code before implementation; do not
rewrite the existing progression roadmap implicitly.

Ship `help classes`, `help promotion`, base/class role pages, updates to
archetype/experience/strategy/company/alignment and affected spell/help pages,
keywords/aliases, browser inspection and a Departure tutorial pointer. Explain
thresholds, current-alignment gates, death retention and branch commitment.
Test rendered help and tutorial pointers.

Acceptance covers player and companion real promotion commands, alignment boundary
values, drift between preview/confirm, dead/absent members, battle/travel/rest
rejection, save failures, duplicate confirmations, old saves, copyover, unknown
class recovery, death/regain and no kit/point/resource duplication. Each signature
must pass real combat integration tests for legality, action costs, cooldowns,
resources and inherited capabilities. Include specialist/training/HP/gear/UI
consumers, not just graph unit tests. Review independently before merging gameplay.

## Decisions proposed for review

1. Promotions at levels 10 and 30; core three advanced paths per base class,
   followed by the expanded companion catalogue.
2. Positive/negative gates at +30/−30, with an unrestricted specialist path.
3. Individual alignment gates promotion only; drift and death never auto-demote.
4. Player chooses for each member at camp or safe settlement, without XP reset
   or initial promotion fee.
5. Forward-only initial branches; later retraining designed separately.
6. Branch roster/signatures, launch staging and relationship to 30g4/33h.
