# internal/interrupt

Phase 30d1's pure rules: when a blow breaks a chant (`Breaks`), what a
company caster gets back (`Refund`), and the shield counter (`CanCounter`,
`RollCounter`). No engine state and no imports: `internal/hooks`
(`combat_interrupt.go`) gathers the facts, rolls with `util.Rand`, and
applies the result. `BashChance` and `StunChance` are package values so
tests can make a counter certain; restore them after.

Design: `docs/superpowers/specs/2026-09-30-phase-30d1-chant-interrupts-design.md`.
