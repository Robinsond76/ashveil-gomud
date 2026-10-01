# Encumbrance Module Guide

Phases 9, 16, 28, and 32f. See `internal/encumbrance` for the pure load and band
rules.

- **The company load** (`CurrentLoad`) has three parts:
  - `PersonalGrams`: the leader's carried and worn items, and their pet's
    pouch (32f review);
  - `CompanionGrams`: living companions' gear, from `company.CompanionGearGrams`
    (Phase 28);
  - `CargoGrams`: the shared cargo.

  Capacity (Phase 32f) is each member's share, `MemberCapacity(base,
  per-Strength, Strength, largest pack)`, for the leader and every companion
  `company.CompanionCarry` counts (one not out brings its pack and no
  Strength), plus the herd's (`mount.CapacityBonus`). `MemberBaseKg` and
  `StrengthKg` are in the config overlay; the old flat `CapacityKg` is
  retired and ignored with a warning. Every reader (expedition departure,
  walking strain, `cargo`, `inventory`, `status`, GMCP) uses
  `Load.TotalGrams()`, so add any new part there.
- **One weight limit** (32f): anything that adds weight from outside the
  company checks `encumbrance.WouldExceed`/`TooMuchToCarry` first, on the
  game loop, with `company.AddedGrams` (a pack counts the room it makes):
  `get`, `buy`, `market buy`, `give` from outside, and a
  companion mob's pickup. Moving things within the company (cargo
  put/take, `give` to your own companion or pet, saddles) and anything that
  lowers the load never checks. Walking is never blocked.
- **Weights** are item data (`weight`, grams). Every shipped item, in the
  default and `empty` worlds, weighs something except a service;
  `shipped_weights_test.go` pins the ranges per type, that each starter kit
  stays under 40% of a fresh member's own share, and that a fresh company of
  five stays under half its base shares. Give a new
  item a weight. Sum loads with `Item.Weight()`, which reads the base data:
  `GetSpec()` would return an item's own spec copy (from cargo, an
  enchantment, an old save) with a stale weight.
- GoMud's count-based `Character.CarryCapacity()` no longer limits anything
  (32f); it is deprecated and kept only for scripts (`GetCarryCapacity`).
  Don't use it.
