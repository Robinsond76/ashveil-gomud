# Wounds Package Guide

Phase 30b ([design](../../docs/designs/2026-09-30-phase-30b-wounds-design.md)).
Pure rules, no world state: callers pass characters' health and wounds in and apply
what comes back.

- A `Wound` holds back `Points` of health. `Limit(max, ws)` is how far healing may
  restore a fighter: `max − Σ points`, never below `Floor(max)` (a quarter, at least 1).
  Healing stops at the limit; nothing here ever lowers health.
- **Lasting** wounds come from damaging crits (`FromCrit`: half the damage, kind by
  weapon subtype via `KindFor`) and stay until treated or rested away. **Light** wounds
  (`Crushing`: a strike of `CrushingPct` of max health or more; `Bled`: a bleed that ran
  out) close at fight end (`CloseLight`). Treatment (`Close`, `Treat`) only ever touches
  lasting wounds, worst first.
- `Plan` is the `heal wounds` planner: patients most hurt first (`Order`), healers most
  mana first, tend each lasting wound then heal to the limit, then splints and bandages.
  It works on copies; `modules/company/wounds.go` spends items and applies the result.
  `DefaultRules` must match `tend.js`/`heal.js` dice (costs are read from the spell files).
- `internal/characters` holds `Character.Wounds` and `HealthLimit`/`CapHealing`; this
  package must not import `characters`.
