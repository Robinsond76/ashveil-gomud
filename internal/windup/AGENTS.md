# internal/windup

Phase 30d2's pure wind-up rules: the registered abilities (`Get`; one
today, `crushing-blow`), the start roll (`RollStart`), the cooldown, and
the lines (`Render`). No engine state and no imports beyond the standard
library. `internal/hooks` (`combat_windup.go`) keeps who is winding up
(runtime maps, never saved), rolls with an injectable `util.Rand`, and
tells the lines; `internal/combat` resolves the landing blow
(`combat.Power`); `internal/interrupt.BreaksWindUp` says what breaks one.

A mob template opts in with `windups: {<ability id>: <percent>}`; only an
enemy uses it. Add an ability here with its four lines in the 29c voice.

Design: `docs/superpowers/specs/2026-09-30-phase-30d2-windups-design.md`.
