# Phase 33 continuation handoff

Owner request: implement all 33a–33i, starting a; choose best recommended
answers without routine confirmation. After confidence, merge to master,
push origin, then start a new cloud session with context. If unavailable,
compact/save context and continue here. Never claim phases not completed.

33a is complete and verified; its plan/design and Project Status describe
behavior, integration tests, independent review findings/fixes and exact
checks. Next is 33b (friendly effects), then remaining 33c–33i. Read each
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
30g3–30g6 are not shipped; future burden/progression/action-budget changes
must coordinate without claiming those phases implemented.
