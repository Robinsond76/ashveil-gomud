# Encumbrance Module Guide

Phases 9, 16, and 28. See `internal/encumbrance` for the pure load and band
rules.

- **The company load** (`CurrentLoad`) has three parts:
  - `PersonalGrams`: the leader's carried and worn items;
  - `CompanionGrams`: living companions' gear, from `company.CompanionGearGrams`
    (Phase 28);
  - `CargoGrams`: the shared cargo.

  Capacity is `CapacityKg` plus a mount's bonus. Every reader (expedition
  departure, walking strain, `cargo`, `inventory`, `status`, GMCP) uses
  `Load.TotalGrams()`, so add any new part there.
- **Weights** are item data (`weight`, grams). Every shipped item except a
  service weighs something; `shipped_weights_test.go` pins the ranges per
  type and that each starter kit stays light. Give a new item a weight.
- This is separate from GoMud's count-based `Character.CarryCapacity()`;
  don't mix them.
