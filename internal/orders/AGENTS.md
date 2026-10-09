# Orders Package Guide

Phase 61 battle orders. GoMud-free (it imports only `internal/strategy`): `Order` (a condition, an action, a health share or foe kind), `Parse`/`Validate`/`Describe`/`Command`, `Evaluate` over a plain `Snapshot`, `Caster` (which spell an order casts), `Preset`, and the `Provider` seam to the durable store in `modules/strategy`.

- At most `MaxOrders` (3) per member, read in list order each round before the member's role and target rule; the first whose condition holds and that `usable` accepts fires. `usable` is how `internal/hooks` says "this action can be carried out now", so an order never fires and does nothing.
- `Validate` is the one rule for which actions fit which conditions (heal needs ally/self, guard needs ally, break needs chanting/boss/foe); the store drops invalid orders on load.
- Adding a condition or action: extend the lists and aliases here, `Describe`/`Command`, `runOrders` in `internal/hooks/combat_orders.go`, `ORDER_ADDS` in `window-combat.js`, `help orders`, and the tests.
- Orders are keyed like strategies (`leader`, `companion:<id>`). Keep this package free of engine imports.
