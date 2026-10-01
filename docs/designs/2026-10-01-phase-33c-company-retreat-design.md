# Phase 33c: Company Retreat, Rout, and Separation

Status: complete, 2026-10-01. The owner authorizes all
phase 33 slices and delegates open choices. Adopted defaults and verification
work are recorded in [the plan](../plans/2026-10-01-phase-33c-company-retreat.md).

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Withdrawal becomes an explicit company outcome with understandable risks, rather than an individual leader escape followed by best-effort follower movement.

## Current mechanics and prior art

`internal/hooks/NewRound_DoCombat.go` checks the leader's Speed against foes targeting the leader, selects a random exit, moves the player, and queues companion movement. Shipped 30e now provides nerve, companion flight, `PendingReturn`, and saved rejoining through `internal/hooks/combat_nerve.go`, `modules/company/morale.go`, and `internal/company/morale.go`. Extend those states; do not rebuild morale.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Distinguish ordered retreat from an involuntary rout and from a member's existing morale flight. Preserve a compatibility path for `flee` until its replacement is approved.
- An ordered retreat considers present members, mobility, burden, wounds, existing no-flee/no-go restrictions, enemy pressure, and a legal escape route. A proposed exit choice must still pass room/script permissions.
- Propose a small round-based withdrawal sequence in which eligible guards cover others. No manual strike spam or instantaneous free rearrangement.
- Explain who escapes, who remains, and why. Decide rescue/abandonment deliberately; do not invent automatic permanent losses.
- Clear departing actors' targets, casts, wind-ups, guards, and battle participation; remaining enemies and allied companies continue normally.
- Reconcile separation with 30e's saved return debt. Dead companions retain 25b's existing resurrection rules. Mount/cargo movement has one explicit outcome; no duplicated cargo or herd.
- Death, disconnected leaders, no legal exits, blocked companions, and successive waiting groups all require defined outcomes.

## State ownership, persistence, and recovery

Retreat resolution belongs to battle/hooks; durable separation and return state belong to company, reusing 30e's identity and pending-return model. Use stable operation IDs/applied markers if several persistent records participate. Snapshot only at valid save seams; do not persist stale battle IDs as recovery authority.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

Depends on 33a; mobility inputs depend on 30g3 and guardian behavior on 30c2. Coordinate allied participation with 33d and recovery with 33h. 30e is already shipped.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Whether flee becomes ordered retreat or emergency individual escape; exit choice; covering action cost; chances and duration; injured-member rescue rules; whether any rout can cause permanent loss. The execution plan records the adopted defaults.

The owner explicitly authorizes implementation and delegates choices.
Adopted formulas and behavior are in the execution plan and decisions below.

## Acceptance criteria and verification

Real rounds cover successful/blocked retreat, no exits, hobbled leader/member, different burdens, covering guards, separation/rejoin, death and morale flight during withdrawal, allied companies continuing, waiting groups, and restart/copyover recovery without cloned members, cargo, refunds, or skipped debt.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. The implementation exercises the real command, round, recovery and view paths;
actual verification and independent review are recorded in Project Status.

## Player help and tutorial acceptance

Add `help retreat`; update `help flee`, `help morale`, `help guardian`, `help combat`, and company return/death guidance. Expose withdrawal state in GMCP, text, and the battle view, and add a tutorial explanation.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.

## Adopted decisions

Preserve emergency flee; add ordered retreat with an explicit or deterministic
legal exit, one preparation round and one attempt. Blocked living members
hold the company, never automatic permanent losses. Use shipped30g3 personal
burden and current wounds for slowest-member mobility, one paid guardian
cover, and only active battle pursuers. Existing30e flight/return remains
durable separation authority; runtime requests never survive as save authority.
