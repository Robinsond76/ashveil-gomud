# Phase 28: Item Weights and the Company's Whole Load — Plan

Design: [28 spec](../specs/2026-09-26-phase-28-item-weights-design.md).

## Task 1: Weights for the shipped items

- [ ] Tests first (`modules/encumbrance/shipped_weights_test.go`): every
  shipped item except the service weighs something, within its type's
  range; the four authored weights are kept; each archetype's starter kit
  weighs 1–12 kg and sits below the first band.
- [ ] Set `weight` in every item YAML.

## Task 2: Companions' gear in the load

- [ ] Tests first:
  - `internal/encumbrance`: `TotalGrams` includes `CompanionGrams`;
  - `internal/company`: `CompanionGearGrams` without a provider and with
    one;
  - `modules/company`: live mob or record, living only;
  - `modules/encumbrance`: the load adds it, and the `cargo` status shows
    it.
- [ ] `Load.CompanionGrams`; `company.GearProvider`/`CompanionGearGrams`;
  `CompanyModule.CompanionGearGrams`; the encumbrance module reads it.

## Task 3: Wiring

- [ ] Through `plugins.Load` with the company and encumbrance modules:
  a recruit's gear raises the load; `cargo put` of heavy goods moves the
  band.

## Task 4: Docs, verification, review

- [ ] `modules/encumbrance`, `modules/company` guides; refresh the stale
  "Known issues" list in `docs/PROJECT_STATUS.md`.
- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review; verify findings; record with a **Review:** line.
