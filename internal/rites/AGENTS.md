# Rites Package Guide

Phase 74. GoMud-free: which losses are mourned (`Mourns`), and what holding or skipping does to loyalty (`Steady`, `Grieve`, with `Floor` and `Ceiling` matching the opinions package). `modules/company/rites.go` queues the occasion in the same save as the departure (`Record.Rites`), applies the outcome through the real loyalty record and the bonds source `bonds.Rite`, and records the chronicle deed; `modules/camping` offers pending rites when a camp or inn stay begins (`company.OfferRites`). Decisions: `docs/plans/2026-10-07-phase-74-rites.md`.

- A rite names no god or creed. Phase 73 adds words to a rite, never effects.
- Holding or skipping never gives gold, experience or power, and never touches a fight. A rite is queued only by a real departure (a companion lost, or a companion who left after long service), once per companion, so it cannot be farmed.
- Loyalty moves stay inside `Floor`..`Ceiling`, so a skipped rite can never be what makes someone desert, and a held one can never make loyalty unbreakable.
- Add a cause: a `Cause`, its `Mourns` rule, `Phrase` and `Verb`, the queue call at the departure, the help page, and a wiring test through that departure.
