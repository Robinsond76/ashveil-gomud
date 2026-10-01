# Phase 30e implementation plan

Design approved by the owner on 2026-09-30: [design](../designs/2026-09-30-phase-30e-morale-mercy-design.md).

## 30e1: enemy morale and mercy

1. Add pure temperament, break-trigger, and reaction rules with deterministic tests; validate race/template configuration and opt suitable shipped foes in.
2. Snapshot original groups at engagement, check resolved deaths and health before settlement, and share surrender ownership by instance across players.
3. Protect surrendered foes at targeting, health, scripts, commands, statuses, and formation seams; cancel hostile state and report yield/flee.
4. Add token-backed post-summary mercy prompts, real-time expiry, lifecycle abandonment, existing-prompt coexistence, and explicit execution rewards without ordinary alignment.
5. Persist mercy alignment and witnessed companion loyalty through owner save paths; cover failures and duplicate decisions.
6. Report yielded endings and surrendered actors in GMCP/browser without revealing dark-room information.

## 30e2: company nerve

1. Snapshot starting company members; evaluate a losing battle once using pure rules.
2. Bound hesitation to one action, cancel committed casts under existing refund rules, and remove fleeing companions from combat.
3. Save pending return with member gear and resolve return/loyalty once at settlement, logout, and copyover; preserve death state and retry failed saves.
4. Cover real round integration, return, desertion, restart, equipment, and failures.

## Delivery gate

Ship indexed morale/mercy help, update combat/alignment/chemistry, add tutorial pointers and render tests. Run focused tests while implementing. Obtain independent full-diff review, verify findings and fix confirmed bugs with regression tests. Run make generate, make validate, go test -race ./..., applicable JS/Lua lint and browser checks. Update PROJECT_STATUS with exact verification and review outcome; commit on phase-30e and integrate only after the gate passes.

## Implementation and review outcome

Both slices are implemented. Independent review findings were fixed with
regressions for abandonment after failed saves, shared ownership transfer and
same-round ordering, zero-loyalty desertion, and offline recovery preserving
later-session character changes. The final reviewer found no remaining blocker.
The pacing finding was withdrawn: cause-bearing messages are queued, and mercy
waits for the summary to drain.

Mercy persists only accepted alignment/reaction effects and receipt tokens.
Prisoner instances, questions, and rewards are transient; unresolved prisoners
leave on lifecycle changes and accepted effects can retry without reward replay.
Flight retains saved equipment and vitals, charges loyalty once, and preserves
existing death state. Full verification and integration are recorded in
PROJECT_STATUS.md.
