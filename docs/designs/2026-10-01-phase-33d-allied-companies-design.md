# Phase 33d: Multiplayer Parties and Allied Companies — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

A player party coordinates allied companies while each player retains authority over their own companions. Battles, support, rewards, and movement must use explicit participation rather than unrelated party membership.

## Current mechanics and prior art

`internal/parties/parties.go` stores player membership, autoattack, and front/middle/back ranks. `internal/usercommands/go.go` queues party following. `internal/mobcommands/lookfortrouble.go` still weights initial player targeting by legacy rank. `internal/mobcommands/suicide.go` divides party XP across all members and grants it to online users without a battle-room check; companion XP has stricter presence rules. Battle ownership is currently player/company-oriented in `internal/hooks/combat_battle.go`.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Treat the party as an alliance of independently controlled companies. Invitations and participation are explicit; joining never transfers companion authority.
- Establish authoritative encounter participants and enemy ownership. An enemy shared by companies must have one life, surrender state, loot claim, and settlement, not duplicate payouts.
- Specify support eligibility, participation after movement/disconnect/death, and joining/leaving an ongoing fight. Reward only eligible participants under the approved contribution policy.
- Preserve focus/formation ownership per company. A party leader can request coordination but cannot rewrite another company's strategy, move its members, or spend its items.
- Audit legacy autoattack, following, and targeting ranks; replace or explain them alongside the 3x3 model. Do not silently require every company to occupy one shared 3x3 grid.
- Add explicit follow consent and separation feedback. Expedition/camp sharing requires its own durable lifecycle and is outside this first slice.
- Share battle loot through a declared claim/distribution policy; no first-command-wins reward race. Preserve 30e's single mercy owner unless a separately approved rule replaces it.

## State ownership, persistence, and recovery

Party/alliance membership belongs to parties; encounter participation and settlement to battle; roster and assets to each company owner. Runtime combat participation must not grant permanent ownership. Persist only approved alliance state; document existing runtime-only recovery. A single settlement identity must guard reward/mercy effects across player/company saves.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

Define this contract before allied scopes in 33b, loot distribution in 33g, and shared assessment/enemies in 33i. Coordinate 33c retreat without forcing allies to flee.

33b review constraint: `effecttargets.OtherBattle` treats a caster as part of
a battle only as its player, its foes, or a mob fighting for its player. When
the allied-leaders provider is installed, a consenting ally fighting the same
enemy group in their own battle must count as part of that battle, or allied
help will be dropped at cast start and completion.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

One shared encounter versus coordinated per-company battles; join-in-progress rules; XP eligibility/contribution and split; loot rights; party following defaults; party persistence; replacement for legacy ranks. No reward formula is approved by this document.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Two real users with companies fight the same and separate groups; distant/merely invited/idle members do not receive unintended XP; support and autojoin obey consent; one user's death/retreat does not command the other; disconnect/rejoin and leader promotion preserve authority; rewards, surrender, loot, and summaries settle once.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help party`, `help company`, `help combat`, reward/loot/follow pages, and allied support guidance. Tutorial Departure points to cooperative play; browser/text views distinguish party leadership from company ownership.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
