# Phase 34 — Company UI and Container Logistics

Status: owner-approved design and delivery plan; gameplay implementation not started.
Baseline: `f4dd180c`. Owner endorsed the UI/logistics direction on 2026-10-01
and added automatic formation placement on recruitment. The owner approved
this concrete design and its proposed delivery defaults on 2026-10-01.
Defaults labelled proposed below are now approved starting values; tuning remains
subject to verification during implementation.

## Scope and delivery order

| Phase | Result | Dependency |
|---|---|---|
| 34a | Shared-inventory presentation, readable text, automatic formation placement | Shipped 33g management |
| 34b | Assigned packs, container-based capacity, cargo-only load, durable migration | 34a |
| 34c | Equipment editor with shared-cargo choices and stat comparisons | 34b |
| 34d | Company effects and current character skills/capabilities | 34c for delivery order; otherwise independent |

These follow up shipped systems. They do not renumber unfinished 33g catalog,
33h, 33i2, or 30g progression/tempo work, or claim those systems have shipped.
Deliver 34a–34d in sequence for this UI initiative.

## Current behavior and prior art

- `modules/gmcp/gmcp.CompanyInventory.go` exposes shared cargo, member equipment,
  horses, and load. Shared cargo uses the leader's item instances as its existing
  durable storage; this storage location does not mean personal ownership.
- `window-company.js` hides Carrying/No pack in shared mode but still renders
  Wearing per member. Seeing legacy labels warrants checking the deployed client
  and cargo migration marker before changing presentation.
- `modules/encumbrance/encumbrance.go` includes worn gear in company load and
  gives members base/Strength capacity plus the largest available pack bonuses.
  `sharedPackBonus` chooses packs automatically from pooled cargo.
- `internal/characters/load.go` already excludes shared cargo from personal
  combat burden, but includes equipped gear.
- `internal/company/formation.go` uses zero-based cells internally; player
  position 2,2 is internal row 1, column 1. Recruitment currently tells players
  to place recruits manually. Registry pruning alone does not place members.
- `modules/company/equipment.go` has exact-instance assignment, removal,
  durable asset recovery, and text comparison. Extend those operations.
- `window-gear.js` has separate Worn and Backpack/Cargo tabs. Character Skills
  is a flat trained-rank list; Company member cards lack a full effects list.
  Company GMCP already exposes automatic ability names.

## 34a — UI consistency and formation defaults

Company Inventory displays one shared Cargo list, treasury, supplies/loot
actions, and the existing capacity summary. Remove member Wearing/Carrying and
No pack blocks from shared mode. Keep equipment data available to Gear.
Container cards arrive in 34b; do not invent assigned containers in 34a.

Audit browser panel text, including inherited default colors and native form
controls. Use semantic theme foreground tokens and verify normal text at a
minimum 4.5:1 contrast in supported themes. Load state uses words and a meter,
not color alone. Keep keyboard focus, accessible names, and safe DOM rendering.

Formation is authoritative game state, shared by browser, text, and combat:

- A company with no companions always places its leader at player cell 2,2.
  Solo move/clear requests explain that the solo position is fixed.
- On successful enlistment, place an unplaced leader first, then the recruit
  in an empty cell. Never overwrite or move existing valid multi-member cells.
- Proposed deterministic vacancy order: 2,2; 1,2; 1,1; 1,3; 2,1; 2,3;
  3,2; 3,1; 3,3. This is a default placement, not class optimization.
- Inform the player of the new position and the existing formation move command.
  Manual rearrangement remains available for multi-member companies under
  existing battle/activity restrictions.
- Backfill unplaced members once for old records, retaining valid player
  arrangements. A deliberate later clear in a multi-member company is not
  silently undone on every refresh. Center the leader when the last companion
  is removed; define solo by roster, not temporary presence or unconsciousness.
- Apply at the shared enlistment seam, including fixed/generated/tutorial
  recruits. Persist placement with membership; failed enlistment/save leaves
  neither a charged recruit nor an extra formation cell. No UI-only fallback.

Acceptance: solo creation/login, existing solo saves, all recruit routes,
last-companion dismissal/loss, manual moves, failed saves, migration idempotency,
and text/GMCP/combat agreement. Diagnose stale deployment/shared-mode symptoms.

## 34b — Packs and capacity

One assigned Pack slot per character. New leaders start with a cloth knapsack
providing 10,000 grams of cargo capacity. New recruits bring an appropriate
starter pack; proposed default is the same 10 kg knapsack until content defines
another pack. Capacity values live in item data, not browser constants.

Cargo capacity equals eligible assigned-pack capacity plus eligible horse
capacity. Replace the old base/Strength cargo allowance and implicit selection
of the largest loose packs. Strength-based personal combat burden stays separate.
Only living, present carriers contribute pack capacity; preserve existing
mount eligibility rules. Fallen/away carriers expose unavailable capacity.

Cargo load is the weight of unequipped shared items. Equipped weapons/armor
are absent from Cargo and do not count toward cargo load. They still contribute
to personal combat burden. Assigned packs and their contents are dropped for
combat and do not contribute to personal combat burden. Spare unassigned packs
are cargo, contribute their item weight, and provide no capacity. Proposed
default: assigned pack tare does not consume its advertised usable capacity.

Inventory shows a container section with cards such as `Knapsack (Ralos) —
10 kg` and eligible horses, followed by one shared item list and total load.
Cards identify carrier, capacity, and availability. Do not invent per-container
item allocation, individual used-weight meters, or bag-to-bag transfers.
Removing a pack returns that exact instance to cargo; replacement cannot
duplicate the old or new item or capacity.

Capacity loss never deletes items: pack removal, carrier death/absence,
dismissal, and horse loss can leave cargo overloaded. Preserve existing overload
travel consequences, communicate overload immediately, and reject ordinary
additions that worsen it. Allow dropping/selling/consuming and capacity-restoring
pack assignment; replacement checks use the resulting load/capacity, not only
the starting overloaded state. Audit equipment removal because returning worn
gear now increases cargo load. Never strand the player without a recovery action.

Migration is durable and repeatable: retain all exact cargo/equipment instances,
assign existing suitable packs once to eligible members, and grant a starter
knapsack only to existing members lacking one. Proposed compatibility default:
one starter grant per such member, marked durably; excess items remain as
overloaded cargo. Never grant on every login or resurrection. Preview the
migration policy in release notes; no silent cargo deletion or forced sales.

Acceptance: no capacity from loose packs; no worn-weight double counting;
10 kg starter; recruits; pack swaps; cargo interactions and overload recovery;
death/resurrection/absence/dismissal/mount loss; save failure/restart/copyover;
old-save migration retry; text and GMCP totals; multiplayer ownership checks.

## 34c — Equipment editor

Character Gear becomes a slot-based equipment editor, including Pack. Select
a slot to see the equipped item and compatible exact instances in shared cargo.
Show empty slots, unavailable choices with reasons, and equip/remove actions.
Equipping returns displaced gear to shared cargo through the existing durable
transfer service and validates final cargo capacity. Preserve hand/slot
compatibility, class rules, activity restrictions, and instance-specific uses,
quality, and sharpening. No equipment presets or manual mid-battle actions.

Show current character stats and a before/after preview of damage, protection,
worn weight, burden/dodge, and other actual affected modifiers. Pack previews
show company capacity/load changes. Do not invent a combat-speed penalty:
current burden affects dodge; future tempo changes remain owned by 30g.
Extend the authoritative comparison read model for structured GMCP; browser
code does not reproduce gameplay formulas. Revalidate selection at execution
and handle stale items cleanly. Initially target the main character; companion
equipment retains its existing command access.

Acceptance: equipped/empty slots; compatible filtering; two-hand conflicts;
exact-instance selection; accurate preview versus applied stats/cargo totals;
stale selection; overloaded recovery; forbidden battle actions; failed save and
restart recovery; keyboard navigation and updates without lost focus.

## 34d — Effects and skills

Company Status adds each owned member's active effects and wounds, displaying
name, remaining duration where meaningful, and a concise mechanical explanation.
Keep persistent bonuses such as chemistry distinct. Read actual effect/wound
state; show unknown/away states honestly and never expose another company's
private state. Follow existing effect expiry and companion recovery rules.

Character Skills groups trained ranks, automatic combat abilities, field
capabilities, and camp capabilities. Reuse shipped 33e/33f unlocks, prerequisites,
strategy toggles, and specialist eligibility. Remove retired entries and stale
manual-action hints. A trained rank may support several capabilities; do not
present it as several independently trainable skills. Deferred progression and
class designs are not unlocks. Show meaningful unavailable reasons without
adding new skill mechanics or a speculative catalogue.

Acceptance: effects applied/expired/removed, wounds and persistent bonuses;
leader/companion/away/dead states; current class/skill unlocks and retirement
migration; automatic versus utility behavior; strategy disabled states;
reconnect and changed-only feed updates; readable keyboard-accessible views.

## Ownership, integration, and verification

Company owns formation/member assignment; existing character/company records
own equipment and assets; encumbrance owns cargo load/capacity; mount owns horse
eligibility; effect/skill owners supply presentation read models. GMCP and UI
are consumers. Extend existing durable asset journals for cross-record pack
transfers. No second cargo store, alternate item identity, or UI state owner.
All routes revalidate the requesting leader's authority. Preserve shared world
time, existing locks/game-loop ownership, and restart/copyover recovery.

Before each implementation phase read owning nested AGENTS.md and inspect the
current baseline. Each phase has its own worktree, integration tests at real
entry points, browser checks, independent review, and status update. Run focused
checks during work; after review fixes run `make generate`, `make validate`,
`go test -race ./...`, and applicable JS/Lua lint once for the final code.

Player help ships in every phase: 34a formation/company-inventory/webclient;
34b cargo/burden/equipment plus a pack topic; 34c equipment/equip/company-inventory;
34d company/conditions/skills and relevant specialist topics. Index new topics,
link hubs, correct tutorial hints, test help rendering, and run
`TestTutorialHelpPointersExist`. Update stale owning design descriptions when
load/capacity behavior changes, preserving their historical shipped records.

## Related records

- [Implementation plan](../plans/2026-10-01-phase-34-company-ui-logistics-plan.md)
- [Project Status](../PROJECT_STATUS.md)
- [Company roadmap](2026-10-01-company-gameplay-roadmap.md)
- [33g equipment and cargo](2026-10-01-phase-33g-company-equipment-loot-design.md)
- [30g burden and tempo](2026-09-30-phase-30g-tempo-defense-design.md)
