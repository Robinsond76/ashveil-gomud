# Errands Package Guide

Phase 70. Pure: `Kind`, `Length`, the saved `Errand`, `Pay`, `WoundRisk` and `Resolve`. `modules/company/errands.go` owns the saved state (`Companion.Errand`), the commands, the timed return and the world. Decisions: `docs/plans/2026-10-07-phase-70-errands.md`.

- Real time only (Unix seconds). Never read or advance the world clock.
- `Resolve` is a function of the saved errand and `hasLair`, nothing else, so a restart or a retried tick cannot reroll an outcome. Put anything an outcome needs into `Errand` when it is sent, never read it at the return.
- Pay is the budget for gold and for an item: the module may only pick an item worth no more than `Outcome.Gold`. Never raise pay or add a reward that can resell for profit (gathered and bought goods must not).
- A new job or length goes in `Kinds`/`Lengths` (and `KindByWord`/`LengthByWord`), the weights, `help errands`, and the web tab reads them through `company.ErrandOptions`.
