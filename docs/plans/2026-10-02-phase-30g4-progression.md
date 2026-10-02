# Phase 30g4 — progression

Implements the [owner-approved 30g design](../designs/2026-09-30-phase-30g-tempo-defense-design.md), decisions 11–13 and 18. Branch: `phase-30g4-progression`.

## Implementation decisions

- Stat steps at levels 5, 10, 15, … match the design's acceptance tests and existing modulo-based point awards. The racial formula uses `1 + floor(L / StatStepLevels)` (level 1 starts at one; a one-level interval retains the old formula). This corrects the draft's conflicting `(L-1)` offset. Stat point awards default to every five levels; skill training remains every level. Existing player training and unspent points are preserved; companion training remains 33h1's derived allocation.
- Use the design's provisional HP rates: warrior 6, cleric/ranger 5, rogue 4, wizard 3 through level 20, then 1 per level. Vitality contributes 1 HP per point. HP gains are configured with archetypes, with a progression default of 5 for unchosen characters and ordinary enemies. Race and mob-template `hpperlevel` may override the enemy default. Derived HP has no additional racial stat growth; existing explicit HealthMax training and equipment/buff modifiers remain.
- Companion archetype is supplied from its durable record before restoring vitals, including resurrection and separation recovery. No new player or company saved state. Enemy HP overrides are authored content; their runtime copy is excluded from character saves.
- Preserve cumulative XP thresholds through level 60. Beyond the knee, grow the *incremental level cost* from the final pre-knee increment by 1.10 each level and add those costs to the knee threshold. Preserve racial TNL scaling, saturate at MaxInt, and introduce no level cap.
- The progression editor uses the same config helpers as live calculations, charts each configured archetype at representative Vitality, and reports absolute cumulative XP even when chart levels are downsampled. MaxLevel remains a display setting.
- Saved health/mana are clamped to the new maxima on restoration, never refilled by migration. No changes to world time, combat actions, or balance tuning owned by 30g5/6.

## Tasks and acceptance

- [x] Tests first: step boundaries, point rhythm/peak protection, archetype/race/template HP, HP knee, XP continuity/scaling/saturation.
- [x] Core config and formulas; player load, class choice, and companion spawn/restore wiring; retain earned state and 33h vitals.
- [x] Admin controls and preview parity, status/level-up next-step text, creation toughness, indexed progression help and tutorial pointers.
- [x] Record shipped starter/companion before/after values and re-run the 30g1 balance table (measure only; 30g6 owns its target).
- [x] Independent full-diff reviewer; verify and resolve findings with regressions.
- [x] Final generate, validate, JS/Lua lint where available, full race tests; status record and phase commit.

Measurements and verification: [30g4 record](2026-10-02-phase-30g4-verification.md).
