# Strategy Package Guide

Phase 32d battle strategies. GoMud-free: `Pick` chooses a target by rule over plain `Foe` values, `Decide` chooses a role's action from a plain `Situation`, and `For`/`AutoSpells` read the durable store (`modules/strategy`) through the `Provider` seam, falling back to defaults when none is registered.

- A blank `Strategy` field means the character's default (`Default(archetype)`): only changes are stored.
- Rules choose among reachable foes; a choice out of reach falls back to the nearest reachable foe, and none in reach to the front-most. Spells pass `ignoreReach`.
- Only configured automatic spells (`AutoSpells`) are ever cast; the caller fills in their costs.
- Keep this package free of engine imports (users, mobs, rooms); the hooks adapt live state into its types.
- Phase 30c tactics (`tactics.go`): `Tactics` (a company-wide focus from `FocusRules`, `none` by default, and the healing threshold `Decide` reads as `Situation.HealBelow`, 50 by default) is stored by `modules/strategy` behind `TacticsProvider`; read it with `TacticsFor`, write with `SaveTactics`. A battle's own focus override lives in `internal/battle`, not here.
- `EnemyPick` is the enemy side (personalities): only reachable members, a noise roll from the `Roll` passed in (tests pass a fixed one), assist/defend/unknown as weakest. Keep noise to one roll per re-aim.
- Phase 30c2 guardian (`guard.go`): the `Guardian` role (words `guardian`, `guard`, `protector`; `guard` is no longer a `defend` alias) fights as a fighter and is never a default. `Strategy.Ward` is the member key it guards (blank: the most hurt), kept only for a guardian. `GuardWard` picks the ward over plain `Guarded` values; the guard itself (counts, the redirected blow) is `internal/battle`'s and `internal/hooks`'.
