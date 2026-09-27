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
- **Weights** are item data (`weight`, grams). Every shipped item, in the
  default and `empty` worlds, weighs something except a service;
  `shipped_weights_test.go` pins the ranges per type, that each starter kit
  stays light, and that a fresh company of five stays under 30%. Give a new
  item a weight. Sum loads with `Item.Weight()`, which reads the base data:
  `GetSpec()` would return an item's own spec copy (from cargo, an
  enchantment, an old save) with a stale weight.
- This is separate from GoMud's count-based `Character.CarryCapacity()`;
  don't mix them.
