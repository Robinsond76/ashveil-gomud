# Phase 33b: Friendly Effects and Company Membership — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Helpful spells, buffs, cures, and utility effects use consistent membership and eligibility rules, whether cast manually outside combat or selected automatically.

## Current mechanics and prior art

`internal/usercommands/skill.cast.go` builds HelpMulti targets from the caster and other player-party members plus their charms, skipping the caster's own companions. `internal/hooks/combat_strategy.go` builds automatic healing targets from the company instead. Inspect `internal/mobcommands/cast.go`, `internal/scripting/spell.go`, spell scripts, and the existing healing/wound providers.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Define member, company, allied-companies, and area scopes. Scope is explicit spell/effect metadata, with a compatibility mapping for existing spell types.
- Company scope includes the leader and eligible attached companions present, even without a GoMud player party. Never include absent, dead, dismissed, or enemy-controlled members.
- Preserve the distinction between a downed-but-living player and a dead companion. Healing respects each patient's wound limit.
- Specify whether each effect allows self, downed actors, pets, temporary charms, or allied companies. Defaults must not silently broaden damaging or area effects.
- Manual and automatic forms of an effect share target resolution. Resolve at start, then revalidate at completion so movement, surrender, and ownership changes cannot create stale targets.
- Allied-company scope requires 33d's consent/membership policy; company-only correctness can ship independently. Area effects must not touch waiting groups simply because they share a room.

## State ownership, persistence, and recovery

Create one effect-target resolver below command adapters; do not store spell membership in UI or combatstream. Runtime target identities remain ephemeral and are checked at completion. Persist configured scope through existing spell/content files, with backward-compatible defaults and validation.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

33a supplies action legality; deliver company scope before broadening automatic abilities in 33e. Allied scopes depend on 33d.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Mapping each shipped helpful spell to scope; treatment of pets and temporary charms; cross-company support consent and costs. Audit existing content rather than assuming all HelpMulti spells should become alliance-wide.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Real manual and automatic casts reach the caster's own companions without a player party; allied scopes exclude non-consenting companies; room/death/charm-transfer changes are rechecked; wound caps hold; downed players remain eligible where appropriate; no duplicate target or double mana charge; harmful effects keep battle boundaries.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help cast`, `help spells`, healing/buff pages, and `help company`; document scope and eligible patients. Point to friendly scopes from the tutorial combat/support lesson.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
