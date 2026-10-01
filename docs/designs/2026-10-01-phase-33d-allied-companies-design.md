# Phase 33d: Multiplayer Parties and Allied Companies — Design

Status: implementation started, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. The original planning request approved direction; the later owner priority
authorizes implementation and delegates defaults. The implementation decisions
below settle the final implementation.

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

## Final implementation decisions (2026-10-01)

The owner explicitly delegates defaults and authorizes merge/push after review.
These decisions replace the preliminary proposals above.

- Independent company battles share the actual enemy instances. Each owner keeps
  formation, focus, equipment and retreat authority; no shared formation grid.
- Accepted membership and leadership save atomically to version-1 alliances.json.
  Missing files migrate to empty membership; corrupt files fail startup closed.
  Failed writes restore the previous registry and suppress success notices.
  Logout preserves membership, clears consent and promotes an online successor.
  Purging a character removes membership before the purge can proceed.
- Follow, mutual allied support and mob autoattack default off. Invitations,
  ranks and consent do not save. Leadership changes revoke follow/autoattack.
  Queued movement/attacks revalidate owner, origin, target and consent generation.
  Existing company/single-target helpful scopes retain their contracts.
- Positive damage attributed to a company is required for that enemy's rewards.
  The owner must be alive, connected, present, not withdrawn and in its battle.
  Mercy can use the contribution snapshot after battle end. One original enemy
  XP pool (including variation/level scaling/elite bonus) is divided equally,
  with integer remainder in ascending user-ID order. Eligible living, attached,
  present companions receive their owner's share through the existing XP seam.
- Multiple qualifying companies force one shared corpse regardless of ordinary
  floor-loot settings. Sorted contributor IDs and enemy instance modulo count
  select one fixed claimant for all ordinary items/gold until corpse decay.
  The claimant distributes through existing give commands. No claim reassignment.
- One living shared-battle primary evaluates group morale and owns mercy;
  departure can transfer primary evaluation without commanding allies to retreat.
- Legacy party ranks are display only, with equal initial targeting weight.
  Browser/GMCP and text distinguish alliance leadership and company ownership.
- Enemy DeathProcessed guards ordinary settlement once per runtime enemy life.
  Runtime enemies/corpses/battles are not restored or replayed on restart, so
  there is no ordinary pending-payout replay ledger. Existing durable mercy
  settlement tokens and player/company save paths remain authoritative. This
  does not introduce atomic transactions across independent character saves.

Verification and independent review results are recorded in Project Status.
