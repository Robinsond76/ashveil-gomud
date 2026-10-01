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
