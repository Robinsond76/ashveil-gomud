# Stance Package Guide

Phase 69 weapon stances. GoMud-free leaf: `Stance`, the four `Defs`, `Fits` over a plain `Gear`, `Parse`, and `Effect`.

- One stance per weapon family (great weapon, shield, bow, dagger). A member picks one; it works only while the member holds what it needs, and does nothing otherwise.
- A stance is a sidegrade. Its numbers are in `Defs`; `internal/combat/stance_test.go` (`TestStancesAreSidegrades`) guards expected damage, and `modules/company/balance_stance_test.go` (`ASHVEIL_BALANCE=1`) runs the 5v5 cells. Criticals and extra turns are worth more in a real fight than their expected damage (they break chants and add procs), so retune with the cells, not by arithmetic alone.
- Keep this package free of engine imports. The durable choice lives in `modules/strategy`; combat reads it from `characters.ClassRT.Stance`, set each round by `internal/hooks` (`applyStance`).
- Adding a stance: a `Def`, `Fits`, `help stances`, the Combat tab menu (`window-combat.js`), and tests.
