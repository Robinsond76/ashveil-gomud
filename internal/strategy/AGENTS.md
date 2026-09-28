# Strategy Package Guide

Phase 32d battle strategies. GoMud-free: `Pick` chooses a target by rule over plain `Foe` values, `Decide` chooses a role's action from a plain `Situation`, and `For`/`AutoSpells` read the durable store (`modules/strategy`) through the `Provider` seam, falling back to defaults when none is registered.

- A blank `Strategy` field means the character's default (`Default(archetype)`): only changes are stored.
- Rules choose among reachable foes; a choice out of reach falls back to the nearest reachable foe, and none in reach to the front-most. Spells pass `ignoreReach`.
- Only configured automatic spells (`AutoSpells`) are ever cast; the caller fills in their costs.
- Keep this package free of engine imports (users, mobs, rooms); the hooks adapt live state into its types.
