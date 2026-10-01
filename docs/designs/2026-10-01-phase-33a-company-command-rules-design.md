# Phase 33a: Company Command Rules and Legacy Action Routes — Future Design

Status: complete, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. The current implementation request authorizes the phase and delegates open
decisions to the lead; the choices adopted below refine the original proposal.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Every way of ordering a member obeys the same ownership, battle, target, and action rules. Preserve automatic combat while closing inherited follower-command routes that bypass player restrictions.

## Current mechanics and prior art

`internal/usercommands/ask.go` queues attack, equip, remove, throw, eat, drink, and inventory commands on charmed mobs. Player equip/use routes reject battle actions; the mob equipment route does not apply the equivalent battle policy. Inspect `internal/usercommands/usercommands.go`, `skill.protection.aid.go`, `skill.tame.go`, `internal/mobcommands`, item/buff/pet scripts, and `internal/hooks/combat_battle.go`. These are static-code findings, not a claim that every route has a reproduced exploit.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Define a small shared action policy for member management, hostile actions, support, conversation, and observation. Validate actor ownership, presence, life state, battle membership, target legality, and the relevant cooldown/action allowance at execution as well as enqueue.
- Apply it to player commands, inherited `ask`, companion commands, skill-generated casts, and scripted player-requested actions. Keep administrative and autonomous world-NPC behavior explicitly distinct.
- Preserve today's automatic battles, one focus order per combat round, and locked equipment/formation/strategy. Do not add manual attacks through this phase.
- Use company management commands for company members; retain `ask` for NPC conversation and documented compatibility aliases where safe. Reject invalid requests with a useful explanation.
- Tame/aid and pet/item scripts must not overwrite a battle action or attack a waiting, allied, yielded, or protected target outside the governing policy.

## State ownership, persistence, and recovery

Policy belongs in a dependency-safe engine/domain seam, with adapters in usercommands, mobcommands, scripting, and hooks. Company ownership comes from stable member identity, not names or generic charm alone. Queued work must revalidate after death, dismissal, charm transfer, movement, or battle replacement. No new persistent action queue is required.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

Can begin after 30g2; coordinate its defended-hit rules. Supply the shared policy before 33c, 33e, and 33g introduce new commands.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Exact public company command names; which legacy aliases remain; the treatment of non-company charmed followers. Broader mid-battle orders remain a separate design decision.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Exercise the real dispatcher and queued mob commands: battle equip/remove/consumables denied through every route; allowed conversation/observation retained; ownership transfer and stale queued actions denied; aid/tame cannot replace a prohibited battle action; allied/waiting/yielded targets protected; legal out-of-battle management still works.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help company`, `help ask`, `help combat`, affected skill pages, and equipment/consumable pages. Add the management-policy explanation to the company/tutorial lesson.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.

## Implementation decisions (2026-10-01)

The owner authorized implementation and instructed the lead to choose defaults
for all questions. Keep safe `ask [member] [order]` compatibility aliases for
present owned companions and temporary charmed followers, alongside existing
company management commands. Conversation/observation are available in battle;
all manual follower hostility is refused. Use `attack [group]` for automatic
battles. No additional mid-battle orders or persistent queue are introduced.
See the [execution plan](../plans/2026-10-01-phase-33a-company-command-rules.md).
