# Phase 33f: Company Specialists and Group Exploration — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Companion expertise matters outside combat: a ranger can track for the band, a medic can improve eligible care, and a skilled craftsperson can prepare equipment. Make group stealth and deliberate solo scouting explicit.

## Current mechanics and prior art

Search/track/train handlers and recipe selection generally consult the player's skills; inherited sneak buffs only the leader. Existing archetype utility automation and `heal wounds` already provide some company-aware services. Reuse those seams rather than award duplicate actions or invent a competing treatment system.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Propose scout, medic, and quartermaster assignments, with capability-based selection for other activities. Use the best eligible specialist present; assistance is bounded and never a sum of the entire roster.
- Route suitable search, tracking, crafting, preparation, and care through a shared specialist resolver. Attribute results to the member doing the work and explain unavailable expertise.
- Give members explicit utility capabilities/unlocks; do not assume a template's archetype grants every player skill. A leader can still perform their own personal action.
- Assignments use stable member identities. Dead, absent, separated, fled, or charmed-away specialists cannot contribute. Replacing a specialist does not reset cooldowns or manufacture supplies.
- Separate company stealth from solo reconnaissance. Group visibility/noise accounts for participating members and burden under proposed rules; hiding the leader alone must not hide visible followers.
- Solo scouting explicitly leaves or holds the company, with encounter/separation and return rules. No teleporting a held company to bypass terrain.
- Medicine modifies approved 30b care rules only; quartermaster benefits must use real supplies and capacity, not free food or cargo.

## State ownership, persistence, and recovery

Persist assignments on company-owned state; keep capability resolution pure and adapters in existing skill, treatment, and recipe owners. Stable IDs survive re-summon. The company roster provides present/eligible members; player commands remain responsible for initiation and permissions.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

Coordinate capability progression with 33h, action legality with 33a, burden with 30g3, solo separation with 33c, and intelligence with 33i. Start with a narrow tracking/search slice.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Specialist unlocks, training, bounded assistance, task costs, failure consequences, stealth/noise formulas, and whether solo scouting is supported in the first slice. Existing utility unlocks must be audited before migration.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Real search/track/craft/care actions use the selected eligible member; absence/death/charm transfer falls back correctly; assignment survives save/load; no skill stacking, duplicate automation, or free resources; group stealth reflects visible members; solo scouting and return respect exits, enemies, and persistence.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Add `help specialists` and group/solo scouting guidance; update `help search`, `help track`, `help sneak`, cooking/crafting/care pages, and company inspection. Tutorial Departure explains recruiting for expedition skills.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
