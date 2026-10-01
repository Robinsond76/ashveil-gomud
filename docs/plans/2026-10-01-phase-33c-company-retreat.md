# Phase 33c execution plan

The owner explicitly authorizes all phase 33 slices and delegates routine
choices. Preserve `flee` as emergency personal escape. Add `retreat [exit]`
as an ordered company withdrawal: one preparation round, then one escape
attempt. A repeated order cannot reset its clock. No automatic permanent
loss, resurrection, or shared-world time advancement.

Adopted defaults: present living attached companions withdraw together;
blocked living members hold up an ordered retreat rather than being silently
abandoned. Dead members retain existing resurrection rules; morale-fled
members retain 30e pending return. Recheck stable ownership, room, charm,
no-flee/no-go, route locks and scripts at resolution. Snapshot membership
at request; never collect late joiners or other companies. Minimum current
member mobility governs the escape roll: adjusted speed reduced by personal
burden (up to 60%) and lasting wound severity (up to 30%). Each active enemy
in the leader's battle exerts pressure; waiting groups do not. Chance is
30 + 70 * company mobility / (company mobility + fastest pursuer mobility),
clamped 30–95; one eligible guardian may spend one guard and its preparation
turn for +15 (cap95). On failure resume ordinary battle; no refunds or free
attacks. Route selection is an explicit exit, or the first legal visible
exit in alphabetical order, never a locked or private route.

1. Add runtime-only withdrawal request and real command. Wire two-round
   resolution on combat loop with injectable escape roll for verification.
2. Move validated company via ownership provider; clear departing casts,
   targets, guard state and windups only after successful leader movement.
   Ensure no unrelated company, mount or cargo is copied or relocated.
3. Make emergency escape respect legal routes and clear battle only after
   successful movement. Reuse saved flight/return for separated companions;
   existing morale flight remains the involuntary rout/separation path.
4. Expose withdrawal in text and Company.Battle/browser view; indexed retreat
   help, flee/morale/guardian/combat/company updates and tutorial/render tests.
5. Real-round tests: success/failure, locks/scripts/no exits, source/member
   restrictions, burden/wounds/guards, transfer/death/morale flight, waiting
   and unrelated companies, separation/rejoin/save failure, restart/copyover
   compatibility. Do not persist queued runtime identities.
6. Independent review; fix real findings with regressions; generate, validate,
   full race and applicable lint once after fixes; status, commit/merge/push.

Completed implementation, help/tutorial, browser and integration coverage.
Independent review accepted six coverage gaps, added and rechecked; no remaining
blockers. Runtime-only requests require no save schema migration; separation
uses30e saved debt and snapshot seams. Final full verification in Project Status.
