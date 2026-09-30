# internal/interrupt

Phase 30d1's pure rules: whether a blow may break a chant (`CanBreak`)
and how likely it is (`BreakChance`, `RollBreak`; Phase 30d1b), what a
company caster gets back (`Refund`), whether a blow breaks a physical
wind-up (`BreaksWindUp`, Phase 30d2: heavy force only), and the shield
counter (`CanCounter`, `RollCounter`; a counter strike only, it breaks
nothing). No engine state and no imports: `internal/hooks`
(`combat_interrupt.go`) gathers the facts, rolls with `util.Rand` (injectable there for tests), and
applies the result. `BashChance` and `StunChance` are package values so
tests can make a counter certain; restore them after.

Design: `docs/superpowers/specs/2026-09-30-phase-30d1-chant-interrupts-design.md`,
its amendment `2026-09-30-phase-30d1b-chant-break-chance-design.md`, and
`2026-09-30-phase-30d2-windups-design.md`.
