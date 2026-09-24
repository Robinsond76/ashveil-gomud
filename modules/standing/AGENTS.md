# Standing Module Guide

Phase 21b settlement standing. A settlement is a configured zone with an alignment (engine scale −100..100). A company's standing there comes from the gap between the company average (`company.CompanyAlignment`, served by `modules/company`) and the settlement's: welcome, tolerated, distrusted, or shunned. Standing is recomputed on every call and never stored, so there is nothing to persist or recover.

Pure rules and the provider seam live in `internal/standing`; `modules/market` (prices, refusals, black-market rooms tagged `BlackMarketRoomTag`) and `modules/camping` (inn price and refusal) call `standing.For` and never import this module. Effects are penalties only, so Phase 19b's no-profitable-round-trip invariant holds without re-capping prices. `standing.For` reads `modules/company` state, which lives on the game loop: call it from commands and event listeners only, and never while holding another module's lock (camping reads it before taking its own).

Config (`files/data-overlays/config.yaml`): `Settlements`, the three gaps (ordered, or all default), the two markups, and `BlackMarketRoomTag`. The `standing` command shows the current settlement and every other one.
