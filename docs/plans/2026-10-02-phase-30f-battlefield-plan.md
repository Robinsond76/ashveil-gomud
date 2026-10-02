# Phase 30f delivery plan

Status: implementation, verification, and delivery complete.
[PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12) is **awaiting review by Opus 5.5**, unmerged.
The owner approved the
[design](../designs/2026-10-02-phase-30f-battlefield-design.md).
Branch: `phase-30f-battlefield`; base `f79995a5`.

1. Record owner decisions, inspect nested package instructions, finalize
   narrow placement/overflow rules, capabilities, and balance defaults.
2. Add pure battlefield geometry/detection helpers and focused boundary
   tests. Integrate effective formation into aim, reach, guard, and views.
3. Wire one-shot travel/camp encounter advantage into battle opening and
   all action paths; test actual game-loop suppression and multiplayer.
4. Restrict sparks to a live formation cluster; implement enemy sweeps and
   leaps with cooldowns, ordinary defenses, narration, and integration tests.
5. Wire authoritative fatigue hit penalties and one-time cold action delays
   across explicit/automatic cast and sling paths; test mechanic and text.
6. Add representative world content, reserved metadata documentation,
   indexed help pages, corrected existing help, and tutorial pointers.
7. Verify recovery/failure paths, independent companies, effective Battle
   presentation, and focused balance scenarios. Run required final checks.
8. Commit/push, open an unmerged GitHub PR with implementation and validation
   details. Add its URL to Project Status and roadmap, recording **awaiting
   review by Opus 5.5**. Push the status update to that same PR. No merge and
   no assertion that requested review has already happened.

## Delivery progress

Tasks 1–8 are complete: the verified implementation was committed as
`1dbd0114`, pushed, and opened as [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12).
Project Status, the combat roadmap, and the phase records link the PR and
record **awaiting review by Opus 5.5**. See the
[verification record](../verification/phase-30f/verification.md) for behavior,
coverage, checks, and the browser screenshot. Independent review remains
pending; the branch is unmerged.
