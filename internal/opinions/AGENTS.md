# Opinions Package Guide

Phase 64. GoMud-free: the kinds of choice, each banter personality's likes and dislikes, the authored lines, the loyalty nudge and its bounds, and the observer seam. `modules/company/opinions.go` applies them; sources report through `company.Opinion` and never import the module. Decisions: `docs/plans/2026-10-07-pillars-phases.md` ("Phase 64 build decisions").

- A choice is reported from its real source (the mercy answer, a started camp rest or inn stay, a meal from the leader's own pack, a relic sale, a story choice's `stance:`), never from a view.
- The nudge is `Nudge` (2), kept between `Floor` (30, above battle nerve's 25) and `Ceiling` (80); a kind has a cooldown per companion in `Kinds`. Do not raise these without re-checking that opinions cannot farm loyalty or make a companion hesitate.
- Add a kind: one entry in `Kinds`, a verdict for each personality that cares in `leanings`, a line for each in `lines` (the tests demand a liker for every kind and a line for every verdict), the kind in `help opinions`, and its source wired with a real-entry-point test.
- Phase 65 bonds should use `Observe`, not edit this package's verdict tables.
