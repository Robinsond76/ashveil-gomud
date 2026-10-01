# Phase 33d execution plan

Base: origin/master `4b4db05e`; pulled and inspected on 2026-10-01. Phase 33c
is the latest completed phase; no 33d implementation was present.

## First unit: membership and consent

1. Add accepted-member follow/support consent and enforce autoattack membership.
2. Prune consent on leave/kick/disband, validate leadership transfer and reset
   following when leadership changes. Invited logout declines the invitation.
3. Wire ordinary movement and 33b allied-effect provider; keep company authority.
4. Ship indexed party help, update friendly-effects/company/combat and tutorial.
5. Cover command, movement and real cast completion integration, plus lifecycle.
6. Independent reviewer inspects the entire unit; fix findings, run generate,
   validate and full race checks, record actual results. Do not mark 33d complete.

## Remaining phase tasks

- Authoritative encounter participation: direct company contribution, late join,
  movement/death/disconnect/retreat eligibility and separate/shared groups.
- Reward policy: split one enemy XP pool among eligible contributors; no distant,
  invited or idle awards. Preserve single mercy settlement and test entry points.
- Deterministic shared loot rights/distribution, with one enemy life/claim/payout.
- Durable alliance schema using existing storage conventions, migration defaults,
  rollback on save failure, restart/copyover/leader recovery coverage.
- Replace or explicitly reconcile legacy targeting ranks and autojoin boundaries.
- Final player help/browser/text authority presentation, full integration tests,
  independent full-phase review, checks and status before integration.

Shared world time never advances for a local action. No implementation worker
is used; the required independent reviewer is read-only.

## First-unit result

Implemented and verified on the feature branch, without completing 33d or
integrating to master. Production follow orders carry current leader/origin and
unique consent tokens, revalidated at actual execution after queue delays.
Independent review findings were fixed and re-reviewed; no initial-unit blockers
remain. Passed touched-package tests, affected root/hooks race tests, generate,
validate, final full race suite and diff checks. Project Status records findings
and the initial fixture failures/fixes. No JavaScript/Lua source changes.

## Full-phase delivery

Completed participation and one-pool rewards; fixed deterministic corpse claims;
versioned durable membership with rollback/recovery; runtime-only consent;
legacy rank reconciliation; shared morale ownership; GMCP/browser/text authority;
updated party/protection help. Tests exercise real two-company combat entry,
independent formations/tactics/retreat, idle/nonparticipant exclusions, reward
and loot idempotence, persistence recovery/write failure and command gates.
Final independent review/checks and integration results live in Project Status.
