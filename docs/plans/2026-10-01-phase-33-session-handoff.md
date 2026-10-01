# Phase 33 continuation handoff

Owner request: implement all 33a–33i, starting a; choose best recommended
answers without routine confirmation. After confidence, merge to master,
push origin, then start a new cloud session with context. If unavailable,
compact/save context and continue here. Never claim phases not completed.

33a–33c are complete and verified; its plan/design and Project Status describe
behavior, integration tests, independent review findings/fixes and exact
checks. Next is 33d (allied companies), then remaining 33e–33i. Read each
design and choose scoped defaults before implementation. All designs are
in docs/designs/2026-10-01-phase-33*. Use isolated phase worktrees, ship
help/tutorial/integration coverage, obtain the independent reviewer required
by AGENTS.md, fix findings, run generate/validate/full race and applicable
lint once after fixes, update status, commit/merge/push. No implementation
worker delegation requested; implement directly. Review subagents required.

Environment: repo /workspace/ashveil-gomud. Activate tools with
`source /workspace/ashveil-env/activate.sh`. Installed JSHint is
/workspace/ashveil-env/js/node_modules/.bin/jshint; pass this as JSHINT to
make js-lint to avoid npx network lookup. Lua checker is in the activated
PATH. Preserve proxy/CA/credentials. Git remote access succeeds when
exec_command requests additional network permission; default sandbox fetch
failed with unreachable proxy. Only push origin Robinsond76/ashveil-gomud.

Tool discovery found environment_status but no cloud-session creation or
compaction operation. Continue in the current session with this handoff.
No automatic usage-reset continuation can be promised without a scheduling
or session capability.

33b research: manual HelpMulti in skill.cast.go omits caster companions;
automatic targets in hooks/combat_strategy.go include the company. Both mob
cast and scripting/spell.go need the same membership/life/room/ownership
resolver at start and completion, preserving wound caps. Company-only
correctness can precede 33d consent. Do not broaden harmful/area spells or
silently include pets/temporary charms; decide and document scope defaults.
30g3 is shipped (e58f20a1); 30g4–30g6 are not shipped; future burden/progression/action-budget changes
must coordinate without claiming those phases implemented.

33b uses internal/effecttargets shared target snapshots and an allied-leader
provider seam; 33d must wire consent. Helpful area callbacks receive arrays.
Void onCast proceeds, other successful void callbacks are recognized.
33c can use Character.Burden and AgilityCapacityGrams from shipped30g3.
Preserve flee as emergency escape; choose separate ordered retreat, legal exit
checks, bounded round sequence, guard cost, and existing30e pending-return
recovery. Read design, settle defaults and plan before coding.

33c adds characters.Retreat Aggro with runtime snapshot, internal/withdrawal
route/member/mobility rules, hooks/combat_retreat.go, selected company relocation,
paid guardian cover, GMCP/browser countdown, and real recovery/ownership tests.
The owner review removed emergency flee: `flee` is the retreat order, and a
pinned member holds the company (no withdrawal separation). Ordered retreat never sweeps
late/foreign/pet instances. Browser harness supports CHROMIUM_EXECUTABLE_PATH
and DOCK_HARNESS_URL; system Chromium plus localhost HTTP passed all checks.

33d research: parties.Get returns invited as well as accepted players; require
IsMember for authority/support/rewards. Leave/promote keeps pointer map but
needs careful leadership validation and consent pruning. go.go currently makes
all accepted party players follow without opt-in; suicide.go shares XP with all
online party members regardless of location/contribution. Prefer independent
company battles sharing one enemy life and existing30e single mercy settlement,
explicit follow/support/autoattack consent, eligible encounter participants and
single payouts. Party persistence needs a declared schema and safe save failure
behavior; existing membership is runtime-only. Read33d design before coding.
