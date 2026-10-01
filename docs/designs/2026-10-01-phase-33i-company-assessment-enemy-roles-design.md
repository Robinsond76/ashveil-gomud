# Phase 33i: Company Encounter Assessment and Enemy Roles — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Encounter information describes company-level risks, and enemy groups offer coordinated roles that make formation, recruitment, and tactics matter. Estimates explain uncertainty instead of promising a result.

## Current mechanics and prior art

`internal/usercommands/consider.go` compares the leader against one character through CombatOdds. `scout.go` already gives visible group formation, health words, and the leader's reach. The 30g1 baseline records no enemy healer, first-strike asymmetry, no enemy wounds, and unplaced formations. Existing enemies have personalities, casters, guardians/interception foundations, wind-ups, and now 30e morale; extend these rather than invent a second combat engine.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Deliver 33i1 group assessment and 33i2 coordinated enemy roles separately.
- Make assessment describe visible counts, composition, formation/reach, exposed support, and the company's actual present readiness. Keep solo consideration explicitly personal or redirect it with an explanation.
- Use qualitative risks/confidence, not exact hidden enemy HP, mana, stats, or guaranteed win percentages. Specialist unlocks may improve intelligence only under an approved information policy.
- Distinguish visible information from inference: hidden enemies, unobserved spells, and reinforcements remain uncertain. Darkness and enemy labels follow scout/battle-view rules.
- Ship opt-in enemy groups with fighters, ranged support, healers, guardians, and approved automatic abilities. Respect resources, turn budgets, reach, defense, interrupts, and surrendered/flight states.
- Review the enemy-wound asymmetry as a separate explicit decision; do not silently apply persistent player wound rules to respawning monsters.
- Rebalance representative fights with healer-versus-healer and placed formations alongside the original no-focus baseline. Include focused play, terrain, burden, guardians, and retreat where available.
- Report casting, statuses, wounds, guards, morale, and relevant threat changes in text/GMCP/battle view as supported by the shipped mechanics.

## State ownership, persistence, and recovery

Assessment is a read model from authoritative company/enemy/battle state, never a second simulation that mutates combat or consumes the gameplay RNG. Enemy roles use template/config metadata plus existing strategy/combat execution; dynamic encounter state belongs to battle/mobs. UI observes the same visibility-filtered data as text.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

33i1 can improve visible assessment early; intelligence improvements depend on 33f. 33i2 depends on 33e/33b and coordinates with 30g6 tuning. Multiplayer estimates require 33d's participation contract; retreat/morale use 33c/30e.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Assessment vocabulary and confidence, specialist knowledge limits, initial enemy role kits, wound applicability, regeneration fairness, difficulty bands, and where enhanced-role balance runs after the original 30g tuning.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Assessment changes with present members, burden, gear, wounds, and formation without exposing hidden values; darkness and hidden foes stay hidden; no state/RNG mutation; real enemy healing/guarding/abilities respect costs and targets; surrender/retreat cancel actions; placed/no-focus and focused multi-level harnesses report outcomes without false guarantees.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help consider`, `help scout`, `help combat`, strategy/guardian/morale pages, and webclient battle-view guidance. Tutorial scout lesson explains group risk and uncertainty.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
