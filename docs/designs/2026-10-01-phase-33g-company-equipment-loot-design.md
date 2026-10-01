# Phase 33g: Company Equipment, Loadouts, and Loot — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Managing a band's equipment becomes a direct, reviewable company action rather than a chain of inherited give/ask commands. Tie recommendations to roles, defense, reach, and burden.

The owner approved the complementary [equipment catalog, tiers, and
Glaivewarden design](2026-10-01-equipment-tiers-glaivewarden-design.md) on
2026-10-01: six tiers, parallel armor paths, weapon families including a
two-handed reach glaive, and the Glaivewarden class with automatic Sweeping
Cut. Use that catalog for role-aware comparisons and future content; numerical
balance remains subject to verification. This approval does not settle the
treasury, transfer, or loot-policy decisions below.

## Current mechanics and prior art

`modules/company/inventory.go` and gear/member providers show durable companion equipment. `internal/usercommands/give.go`, `ask.go`, and `internal/mobcommands/equip.go` provide transfer/equip routes; `internal/usercommands/gearup.go` evaluates only the player. 22b's save seams protect gear ownership, and 32f already supplies cargo/capacity.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Propose direct equip/remove/transfer operations by stable member and exact item reference, plus comparisons showing role eligibility, reach, shield defense, armor, burden, and resulting capacity.
- Treat gold as a company-managed treasury through an explicit policy; retain ownership for trading and multiplayer transfer. Do not silently confiscate a companion's saved gold.
- Let loot enter shared cargo when capacity permits, then assign it deliberately. Respect corpse/claim ownership, quest items, and other players' rights.
- Save formation/equipment presets as desired arrangements, never duplicate item instances. Apply only outside battle and after validating members, ownership, required items, and capacity.
- Automatic upgrade recommendations show tradeoffs and require deliberate application; raw damage/armor ranking must not override a healer's or guardian's role.
- Preserve 33a's restrictions through every UI/command/legacy route. Use the same backend for browser and text.
- Missing preset items or failed transfers produce an accurate result; avoid partial application without an approved rollback/recovery rule.

## State ownership, persistence, and recovery

Actual gear remains on player/live-member records and 22b snapshots; cargo stays with encumbrance. Presets store references/preferences, not copies of assets. Cross-record transfers must use existing valid save seams or durable operation IDs/applied markers. Never save a company gear snapshot independently in a way that can restore a transferred item twice.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

Depends on 33a and 33d's multiplayer loot/claim contract. Uses 30g2 defense and 30g3 personal load; aligns role comparisons with 33e and growth with 33h.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Treasury migration, loot claim/distribution policy, preset failure semantics, exact item references, command names, and whether upgrade application is single-member or whole-company. No auto-loot ownership bypass.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Acceptance criteria and verification

Real transfers/equips across leader, companions, cargo, corpses, and players; exact duplicate-name items; full capacity; incompatible gear; missing presets; battle denial; stale browser requests; save failure/restart/copyover with neither duplication nor lost ownership; recommendation tradeoffs match actual combat/load calculations.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Add `help loadouts` and company loot guidance; update `help equipment`, `help company`, `help inventory`, `help cargo`, `help gearup`, and webclient help. Tutorial preparation points to equipment assignment.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
