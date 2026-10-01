# Phase 33e: Automatic Class Abilities and Combat Roles — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Existing skills and spells contribute to a member's automatic role, so recruiting a rogue, ranger, fighter, cleric, or wizard changes company behavior without returning to individual attack-command control.

## Current mechanics and prior art

`internal/strategy/decide.go` supports fighter/healer/caster/guardian decisions and four default automatic spells (heal, healall, mm, sparks). `internal/hooks/combat_strategy.go` starts actual casts. Manual backstab, tackle, disarm, cast, and related battle commands are restricted by the shipped automatic-combat direction. Inspect skill handlers, archetype unlocks, spell files, equipment eligibility, and 30g's action meter before selecting abilities.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Audit each inherited combat skill: retain as passive, adapt into an automatic action, keep as utility, or retire with migration/refund guidance. Owning a skill must have an explained benefit.
- Propose a small archetype repertoire: rogues exploit exposure; fighters choose suitable disruption; rangers use ranged opportunities; clerics protect/heal; wizards choose approved attack/support spells.
- Use a short priority list or archetype profile with visible conditions, not a general scripting language. Preserve fighter/healer/caster/guardian roles unless an explicit migration replaces them.
- Actions require unlocks, equipment, legal reach/targets, resources, and cooldowns. Interrupts, armor, defense, statuses, wounds, and immunity follow shared combat resolution.
- All actions spend the existing turn/action allowance; 30g5's meter cannot grant both an ability and a free ordinary attack. Pet assistance is separately audited.
- Coordinate healers against reserved/pending heals to avoid routine waste; support explicit mana conservation and weapon fallback.
- Broad commander orders beyond the existing focus order remain proposed future work, not an implicit approval to open manual battle commands.

## State ownership, persistence, and recovery

Pure choices belong to strategy with immutable situation inputs; execution remains in hooks/combat and ability-owning packages. Persist profiles/unlocks through strategy/archetype/member state, with stable defaults for old saves. Runtime cooldowns and pending reservations need documented restart behavior.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

33a and 33b establish legality and friendly scopes. Align with 30g4 progression and 30g5 turns; rerun balance in 30g6 or a subsequent tuning pass. Give enemies equivalent legal capabilities through 33i.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Initial ability roster, profile controls, cooldown/resource defaults, refunds for retired skills, and whether ranged behavior requires a distinct role. Do not promise every inherited skill in the first release.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Real player and companion rounds prove unlock/equipment gating, target/reach legality, action-meter charges, cooldowns, mana fallback, coordinated healing, interruptions, defense/status/wound interactions, and persistence. Test the adapted abilities through their integration paths rather than only strategy helpers.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Add `help abilities`; update `help strategy`, `help tactics`, archetype and affected skill/spell pages. Show selected priorities in member inspection and browser setup; tutorial combat explains automatic abilities.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.

## Final implementation decisions (2026-10-01)

The owner delegated open defaults to the lead and authorized merge/push after
the review gate. These decisions replace the open proposals above; the
"future design" wording at the top is historical.

**Roster (first release).** One automatic ability per weapon archetype; the
cleric and wizard keep their spell roles and gain coordination and a reserve.

| Archetype | Ability | Player unlock | Companion unlock | Used when | Effect | Cooldown |
|---|---|---|---|---|---|---|
| Warrior | Tackle | brawling ≥ 1 | warrior | no shooting weapon; its foe is on its feet (not knocked down or stunned) | chance Speed − foe Perception + 20, held to 20–80%: the foe is knocked down (30a status); a chanting foe's chant breaks (an enemy starts again), a wind-up breaks. Hit or miss, it is the warrior's turn: no swing that round | 4 combat rounds |
| Rogue | Opening Strike | skulduggery ≥ 1 | rogue | wielding a weapon, every wielded weapon a backstab kind (slashing, stabbing, cleaving, claws); its foe knocked down, stunned, staggered, or exposed | that round's swing: its first blow that lands is a critical hit. It still rolls to hit, meets block/parry/dodge and armor, and leaves its weapon's crit status and wound as any crit | 2 combat rounds |
| Ranger | Aimed Shot | track ≥ 1 | ranger | a shooting weapon; its foe not already exposed | that round's shot: its first blow that lands is a critical hit (a shooting crit leaves the foe exposed, opening it for a rogue) | 3 combat rounds |
| Cleric | coordinated healing | | | | healers skip allies a heal already in progress (or chosen this round by another healer of the same company) covers; a group heal in progress covers everyone | |
| Wizard (any caster) | mana reserve | | | | `strategy [who] reserve [percent]` (0–90, default 0): attack spells are cast only while that share of maximum mana would remain; heals ignore it | |

- **Ownership and scope.** Abilities are chosen per actor from the same
  side list as automatic spells: the player and their own living,
  present company companions (by `company.LeaderAndKeyForInstance`), never
  pets, temporary charms, another player's company, allies, the dead, the
  downed, or withdrawn/retreating members. The target is always the actor's
  current aim, a standing, visible foe of that player's own battle group,
  which the upkeep keeps legal for reach, so no ability reaches another
  battle or bystander. The formation gate applies as for a swing: an aim
  the enemy front row would intercept gets no ability (review fix), and a
  tackle needs the foe within hand-to-hand reach. No one in a company
  preparing to retreat uses an ability (review fix).
- **Turn and action budget.** An ability is the actor's whole turn this
  combat round (Tackle) or this round's swing itself (Opening Strike, Aimed
  Shot); it never adds an attack. A member that casts this round uses no
  ability. A member whose weapon is still waiting (`RoundsWaiting`), who is
  chanting, retreating, stood down, nerve-shaken, or loses the action to a
  status uses none. 30g5's meter will charge abilities like any action.
- **Resolution.** Shared rules apply: Opening Strike and Aimed Shot use the
  existing backstab crit (fixed so a crit never reports on a round with no
  landed blow), so hit, defense, armor, crit status and 30b wounds follow;
  Tackle applies the 30a knocked-down status and reuses the 30d1/30d2
  break paths. No ability uses, moves, or consumes an item (owner rule: no
  item may be used in a fight).
- **Controls.** `strategy [who] abilities on|off` (default on) and
  `strategy [who] reserve [percent]`. Both are durable fields of the
  member's strategy (`no_abilities`, `reserve`); old saves load as on/0.
  `default` clears them. Changes are refused in a battle like every
  strategy change. Abilities need no new role: ranged behavior follows the
  weapon, not a role.
- **Runtime state.** Cooldowns count combat rounds in a game-loop map keyed
  by actor and ability ("at most once every N rounds"; spent when tried,
  even if the foe falls before the blow); they are not saved, so a restart or copyover (which
  ends every battle anyway) makes every ability ready. Opening/aimed strike
  marks are cleared after the round's blows.
- **Skill audit.** brawling: Tackle automatic; disarm, throw, recover stay
  manual commands for fights outside company battles (player duels).
  skulduggery: backstab adapted as Opening Strike in battles (manual
  backstab unchanged outside them); sneak, pickpocket, bump, traps unchanged.
  track: utility, and unlocks Aimed Shot. dual-wield: passive. cast: the
  healer/caster roles. protection, tame, peep, enchant, portal: utility,
  unchanged. Nothing is retired, so no refund is owed. Pets get no
  abilities (pet assistance unchanged). Disarm is not automatic in this
  release (it moves an enemy's weapon; deferred to 33i/33g).
- **Presentation.** `strategy` lists each member's abilities and shows
  `abilities off` and a reserve; `strategy [who]` explains each ability's
  condition and cooldown. GMCP `Company` member strategies carry
  `abilities`, `abilities_off`, and `reserve`, and the web Combat setup
  shows them. Ability lines follow the 29c voice, and an `ability` event
  goes on the combat stream. `help abilities` is new; `help strategy`,
  `help tactics`, `help combat`, `help company`, `help archetype` (the
  archetype page; `help warrior`/`help ranger` are GoMud's job pages and
  stay as they are), `help brawling`, `help skulduggery`, and `help track`
  are updated; the practice-fight lesson points to `help abilities`.
- **Balance.** Not retuned here; 30g6 (or a later tuning pass) measures it.

Verification and independent review results are recorded in Project Status.
