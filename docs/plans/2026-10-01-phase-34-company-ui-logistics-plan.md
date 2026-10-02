# Phase 34 — Company UI and Logistics Delivery Plan

Status: owner-approved plan; 34a complete and verified; 34b–34d pending. Follow the
[design](../designs/2026-10-01-phase-34-company-ui-logistics-design.md).
Owner approved the concrete design and delivery defaults on 2026-10-01. No gameplay checks are claimed by this documentation change.

## Delivery procedure

Implement 34a, 34b, 34c, then 34d in separate feature worktrees. Recheck current
master and nested guidance before each phase. Complete each phase's acceptance
checks, independent review, help/tutorial work, and required final checks before
integration. Use current task model settings; implement directly, with an
independent reviewer at the required gate. Record real results in Project Status.

## 34a — Presentation and automatic placement

1. Trace a migrated save and Company.Inventory payload. Check old client assets,
   shared-mode marker, login migration errors, and reconnect/update behavior.
2. Remove personal/worn inventory blocks in shared mode while preserving the
   shared Cargo list, treasury/actions, and equipment data for Gear.
3. Audit theme foreground inheritance across company/character/gear and shared
   controls; fix contrast with semantic tokens and check supported themes.
4. Implement durable solo centering and vacant-cell enlistment at shared domain/
   lifecycle seams. Add versioned backfill; preserve later deliberate manual
   clears. Adjust solo move/clear responses and recruitment position messages.
5. Test fixed/generated/tutorial recruitment, failed saves, old-save backfill,
   final-member removal, and formation agreement across text, browser, combat.
6. Update formation/company-inventory/webclient help and tutorial hints. Run
   browser focus/contrast checks and help rendering/pointer tests; review and
   final checks, then record and integrate the phase.

## 34b — Pack equipment and capacity

1. Add Pack slot and item-driven capacity with data validation. Extend starter
   kits and fixed/generated recruits with once-only starter packs.
2. Integrate exact-instance pack assignment/removal with existing equipment
   journals and save rollback/replay. Exclude Pack from personal combat burden.
3. Replace cargo capacity with eligible assigned packs plus horses; cargo load
   counts only unequipped shared items. Audit every add/remove/equip/consume/
   trade/loot/crafting check against final load and capacity.
4. Implement durable old-save migration and overloaded recovery, testing grant
   markers and crash boundaries. Audit death/absence/dismissal/mount lifecycle.
5. Extend text/GMCP read models for assigned containers and availability;
   render capacity cards plus one shared item list in Company Inventory.
6. Update cargo/burden/equipment help, add indexed pack help and tutorial hints,
   amend superseded load/capacity design prose. Test real command and save/load
   paths, totals and migration; review, final checks, record and integrate.

## 34c — Equipment selection and comparison

1. Extract/extend a structured authoritative comparison view from existing
   company equipment operations, retaining exact instance identifiers.
2. Expose slot compatibility, current/candidate stats and load/capacity deltas
   through GMCP with changed-only updates and stale-state handling.
3. Build the main-character slot editor and compatible cargo chooser. Include
   Pack, equip/remove, empty states, reasons and before/after comparisons.
4. Verify applied changes match preview, final capacity guards, two-hand rules,
   activity restrictions, save failure/recovery, stale selections and ownership.
5. Update equipment/equip/company-inventory/webclient help and tutorial pointers.
   Verify keyboard/focus and browser interactions; review, final checks, record
   and integrate. Keep companion command management working.

## 34d — Effects and current capabilities

1. Inventory current effect/wound and 33e/33f capability read models; identify
   authoritative duration, eligibility, prerequisite and strategy state sources.
2. Add owned-member effects/wounds to Company feeds and Status cards, separating
   persistent bonuses and unavailable state; verify expiry/reconnect behavior.
3. Extend Character Skills payload/presentation with trained ranks, automatic
   abilities, field and camp capabilities using actual shipped unlocks.
4. Test class/rank changes, retired-skill migration, disabled strategy states,
   effect apply/remove/expire, absent/dead members and private-state boundaries.
5. Update company/conditions/skills and affected specialist help and tutorial
   pointers. Check readable responsive layout, keyboard/focus, help rendering
   and pointer tests; review, final checks, record and integrate.

## Boundaries

No class catalogue, new progression curve, combat tempo rewrite, container item
allocation, new skill unlocks, equipment presets, or companion gear editor is
required here. Those retain their owning phase or need a separate scope.
