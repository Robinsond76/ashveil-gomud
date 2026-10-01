# Company Gameplay — Future Phase Roadmap

Updated 2026-10-01, against origin master `626df427`. The owner endorsed
the company-gameplay review and requested future-phase documentation.
This roadmap records that direction; individual mechanics and balance
proposals still need decisions before implementation.

## Direction

Ashveil is a game about leading a persistent company. Recruitment,
composition, formation, preparation, and company orders should determine
what that company can do. Members execute their own legal actions.

Use Ogre Battle as a guide to composition, formation, and automatic roles,
and Mount & Blade II as a guide to specialists, readiness, company management,
and broad orders. Keep the existing small company and round-based multiplayer
world. Do not add an army-size expansion, continuous combat timeline,
mounted combat, or unrestricted individual battle commands through this roadmap.

The review used code and project records, not a live multiplayer playtest.
Origin advanced while planning: 30e is now shipped, including company nerve,
flight/return, and mercy. These phases extend that foundation.

## Phase register

Phase 31 was explicitly dropped. Phase 33 is a new planning family; it does
not reopen completed Phase 32 work. Every review recommendation maps here.

| Phase | Scope/design | Status |
|---|---|---|
| 33a | [Company Command Rules and Legacy Action Routes](2026-10-01-phase-33a-company-command-rules-design.md) | Complete (2026-10-01) |
| 33b | [Friendly Effects and Company Membership](2026-10-01-phase-33b-friendly-effect-scopes-design.md) | Complete (2026-10-01) |
| 33c | [Company Retreat, Rout, and Separation](2026-10-01-phase-33c-company-retreat-design.md) | Complete (2026-10-01) |
| 33d | [Multiplayer Parties and Allied Companies](2026-10-01-phase-33d-allied-companies-design.md) | Complete (2026-10-01) |
| 33e | [Automatic Class Abilities and Combat Roles](2026-10-01-phase-33e-automatic-class-abilities-design.md) | Complete (2026-10-01) |
| 33f | [Company Specialists and Expedition Skills](2026-10-01-phase-33f-company-specialists-design.md) | 33f1 skill and charm retirement and 33f2 expedition specialists complete (2026-10-01); 33f3 camp specialists next |
| 33g | [Company Equipment, Loadouts, and Loot](2026-10-01-phase-33g-company-equipment-loot-design.md) | Future design; implementation not started |
| 33h | [Company Progression, Rewards, and Expedition Continuity](2026-10-01-phase-33h-progression-recovery-continuity-design.md) | Future design; implementation not started |
| 33i | [Company Encounter Assessment and Enemy Roles](2026-10-01-phase-33i-company-assessment-enemy-roles-design.md) | Future design; implementation not started |

33h has three delivery slices: growth/rewards, readiness/recovery, and
relocation/separation. 33i has two: group assessment and coordinated enemies.
These are delivery boundaries inside their parent phases, not shipped work.

## Suggested delivery order

30g2 is shipped. The owner now requests all Phase 33 slices in order and
authorizes the lead to choose open defaults. 33a–33e are complete. Continue with 33f; 30g3 is shipped and 30g4–30g6
remain queued. Dependencies below guide integration choices:

1. Start 33a and company-only 33b as a compatibility pass after 30g2.
2. Settle 33d's multiplayer participation and ownership contract before
   implementing allied spell scopes, shared loot, or cooperative estimates.
3. Align 33c with 30g3's mobility/burden; 30e's flight/rejoining is already
   present. Deliver 33h readiness/relocation corrections alongside these
   where they remove existing lifecycle inconsistencies.
4. Align 33h growth with 30g4; build 33e around 30g5's action budget.
5. Add 33f specialists and 33g management once the ownership, capability,
   and inventory contracts are settled. Basic 33i group assessment can
   ship earlier using currently visible information.
6. Add 33i coordinated enemies and run representative balance checks.
   If these land after 30g6, schedule a new tuning pass; do not treat an
   older four-spell/no-enemy-healer baseline as final balance for new abilities.

Placement among 30g slices and 30f remains an owner decision. Compatibility
fixes may be narrow releases; new mechanics require their phase design and
execution plan. Phase 32b's outstanding review is separate.

## Shared gameplay and ownership rules

- One player owns their company; a player party coordinates allied companies.
  Party leadership never transfers another player's companions, assets, or
  strategy authority.
- Stable member and item identities authorize actions. Names, browser selections,
  generic charm, and stale battle IDs are insufficient.
- Automatic strategies remain in force. Focus is currently the limited
  mid-battle order; withdrawal proposals do not authorize manual ability spam
  or unrestricted formation changes.
- Friendly effects resolve eligible present allies consistently. Harmful
  effects respect enemy-group/battle boundaries, visibility, surrender, and
  ownership. Queued actions revalidate at execution.
- One enemy, corpse, item, reward, and mercy decision have one authoritative
  lifecycle even when several companies participate.
- Preserve 25b's companion death/resurrection, 30b's wound/treatment,
  30e's pending returns, and 30g's defense/progression/tempo ownership.
  Where new policies change these, explicitly amend the owning design.
- Pets and temporary charms need explicit participation, slot/action,
  friendly-effect, burden, and reward rules. Review their combat assistance
  against the company cap rather than assume they are free extra members.
  Do not remove existing pets or migrate ownership without a decision.
- Travel, rest, crafting, retreat, and care never advance global game time.
  Offline recovery, if adopted, needs an explicit bounded policy.
- Existing persistent records remain authoritative. Cross-file transfers
  need valid save seams or durable operation IDs/applied markers; UI and
  combat events do not become state owners.

## Decision register

| Decision | Owning phase |
|---|---|
| Legacy commands retained; allowed management and battle requests | 33a |
| Effect scope defaults; pets/charms; allied-support consent | 33b, 33d |
| Retreat versus flee; exit choice; rescue; covering costs; separation | 33c |
| Shared encounters, eligibility, party ranks/following, loot claims | 33d |
| Automatic skill repertoire, priorities, resource/action costs, refunds | 33e |
| Retired skills and charm; specialist capabilities; no group stealth or solo scouting (owner, 2026-10-01) | 33f |
| Treasury policy, exact item selection, presets and partial failures | 33g |
| Growth profiles, contracts, vitals migration, offline recovery, transports | 33h |
| Intelligence limits, enemy roles/wounds, difficulty and balance coverage | 33i |

Do not mistake the owner's endorsement of the review for approval of every
proposed formula, permanent loss, migration, or new control mode.

## Implementation and verification gate

Each design identifies current code, state owners, persistence/recovery,
integration points, dependencies, player help, and acceptance tests.
Before implementation, inspect current code/nested guidance, settle owner
decisions, and write a focused plan. Use an isolated feature branch/worktree.

Player-visible phases ship indexed help templates, hub links, tutorial
pointers, render tests, and matching text/browser behavior. Test real
command/round/save/load paths, then independently review the full phase
diff and run the required code checks. Record results in Project Status.
Document-only planning changes need link/reference and whitespace checks,
not claims that future gameplay acceptance tests have already passed.

## Related records

- [Project Status](../PROJECT_STATUS.md) is the authority for shipped work.
- [Combat roadmap](2026-09-26-combat-presentation-roadmap.md) tracks 30f/30g.
- [30g design](2026-09-30-phase-30g-tempo-defense-design.md) owns the balance sequence.
- [30e design](2026-09-30-phase-30e-morale-mercy-design.md) records shipped morale/mercy.
- [Agent workflow](../AGENT_IMPLEMENTATION_WORKFLOW.md) defines implementation gates.
