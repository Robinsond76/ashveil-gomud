# Phase 9 Encumbrance and Cargo Design

**Status:** Confirmed and implemented, 2026-09-22. Per the owner's standing
"continue implementation with whatever you recommend, no need to ask"
instruction, the recommended answers below were taken rather than asked
again: (1) Option A — engine/display only, no `TravelSession`/camp-rest
schema change; (2) full `cargo put`/`cargo take` commands this phase, since
cargo is named in the phase title and a container nobody can put anything
in is a hollow slice; (3) a flat, config-driven capacity (no company-wide
Strength aggregation across companions, matching weather's config-table
simplicity); (4) ship at the zero-weight default, defer balance data
authoring for Dunmar's existing items to a follow-up.

## Goal

Add party-level, weight-based encumbrance: personal equipment/inventory
weight plus a new shared company cargo container, rolled up into a party
total weight, capacity, and load ratio, with travel-duration and fatigue
modifiers derived from that ratio (handoff §34). Mount capacity is
explicitly out of scope until Phase 10 (handoff §18: "Do not implement
mounts before normal travel, encumbrance, and fatigue work").

## Prior-art check (handoff requirement: inspect before building)

GoMud/Ashveil already has two *different* things this must not collide with:

- **`items.ItemSpec` has no weight field at all.** Precedent for adding one:
  Phase 4 added `Nutrition`/`Hydration int` to `ItemSpec`
  (`internal/items/itemspec.go`), zero-value by default so existing items
  are unaffected until authored. A `Weight int` field follows the same
  pattern.
- **A native, *count*-based "encumbrance" already exists and is unrelated.**
  `Character.CarryCapacity() = 5 + Strength.ValueAdj/3`
  (`internal/characters/character.go:234`) counts *items*, not weight.
  `internal/usercommands/go.go:74` already refuses cheap movement
  (`actionCost = 50` instead of `10`) when `len(Character.Items) >
  CarryCapacity()`, with the message "You're too encumbered to move (help
  encumbrance)!". This is a per-character, per-move native throttle,
  already shipped, and already uses the word "encumbrance". Ashveil's
  Phase 9 must not rename, remove, or fight this — it needs to add a
  *separate*, weight-based, *party/expedition*-level system (matching
  `internal/survival`'s exertion model) that answers a different question:
  how does the company's total load affect a multi-checkpoint journey and
  camp recovery, not whether one player can afford their next single step.
- **No shared/party cargo concept exists.** `internal/company.Record` is
  leader + companions only (`internal/company/company.go`); no container
  field. This is new.
- **Companions are native GoMud mobs, not modeled with real inventories in
  this codebase.** `internal/survival`'s `MemberKey`/`CompanyNeeds` seam
  tracks a per-member need record keyed by a stable member identity
  regardless of whether the member is a native mob — that's the pattern to
  reuse for "who is in the company," but only the **leader** is a real
  player character with `Character.Items`/equipped slots. Companion combat
  loadout is out of scope for GoMud item weight (they have no `Items`
  field wired through this system).

**Conclusion:** no complete engine to integrate — build `internal/encumbrance`
+ `modules/encumbrance`, following the same domain/module split as
expedition/camping/weather, reusing `items.ItemSpec` once it gains `Weight`.

## Scope

Included:

- `Weight int` added to `items.ItemSpec` (grams; zero means unweighted,
  same soft-migration shape as `Nutrition`/`Hydration`).
- Leader's personal load: sum of `Character.Items` + equipped slot weights
  (native GoMud character, no new schema there).
- A new durable **company cargo container**: a flat, unslotted list of
  item stacks owned by the company (leader-keyed, like every other
  Ashveil registry), with deposit/withdraw commands.
- Party total weight = leader personal load + cargo weight (companions
  excluded — they carry nothing modeled in this system).
- Party capacity, load ratio, and the travel-duration/fatigue modifier
  outputs the handoff's §34/§17 examples describe.
- A `cargo` command (`cargo` / `cargo put <item>` / `cargo take <item>`)
  and an `encumbrance`-line addition to `company status`-style output
  showing load/capacity/ratio — explicitly not reusing the native
  `encumbrance` help topic or wording, to avoid confusion with the
  unrelated per-move throttle.

Excluded (deferred; see Constraints and Deferrals): mount capacity (Phase
10), stealth/noise modifiers from load (handoff lists this as "possibly"),
per-companion carried gear, cargo loss/theft, and any change to the native
`Character.CarryCapacity()`/`go.go` per-move throttle.

## Durable Model

`internal/encumbrance` is GoMud-free, matching `internal/expedition`,
`internal/camping`, and `internal/weather`.

```go
type CargoStack struct {
    ItemId int
    Count  int
}

type Cargo struct {
    LeaderUserID int
    Stacks       []CargoStack
}

// Deposit/Withdraw are pure, validated, copy-returning, like every other
// domain mutation in this codebase.
func (c Cargo) Deposit(itemId, count int) (Cargo, error)
func (c Cargo) Withdraw(itemId, count int) (Cargo, error)

type Load struct {
    PersonalKg float64
    CargoKg    float64
    CapacityKg float64
}

func (l Load) TotalKg() float64
func (l Load) Ratio() float64 // TotalKg / CapacityKg, 0 when CapacityKg <= 0
func (l Load) TravelDurationPct() int // derived from Ratio via a validated threshold table
func (l Load) FatiguePct() int        // derived from Ratio via a validated threshold table
```

- Weight is stored/computed in grams internally (matching typical item-spec
  integer conventions) and surfaced in kg for display, mirroring the
  handoff's kg examples.
- Threshold → modifier mapping is a small ordered table (e.g. ratio bands
  to `TravelDurationPct`/`FatiguePct`), validated and rejected rather than
  guessed if malformed — same shape as `weather`'s biome condition table,
  but simpler (one table, not per-biome).

## Module

`modules/encumbrance` owns:

- A durable, leader-keyed `Cargo` registry, YAML-persisted like every
  other module registry.
- The capacity formula and load-threshold table as module config (embedded
  default + on-disk overlay), not hardcoded.
- `cargo` / `cargo put` / `cargo take` commands operating on the leader's
  backpack and the company cargo container.
- A read-only query seam (`CurrentLoad(leaderUserID) (encumbrance.Load,
  bool)`), the same shape as `weather.Provider`/`camping.ViewProvider`,
  for `company status` (or a new `encumbrance status`) to render load/
  capacity/ratio without any other module importing this one directly.

## Integration with existing systems (open decision — see below)

Same fork the weather design hit: **Option A** ships the read-only load
engine (compute and display party weight/capacity/ratio/modifiers) without
touching `modules/expedition`'s `TravelSession` or `modules/camping`'s rest
recovery — those consult the query seam in a reviewed follow-up. **Option
B** wires `TravelDurationPct`/`FatiguePct` into travel/rest now, which
needs the same kind of schema addition Phase 8's Option B would have
needed (an effective-duration/exertion field resolved at journey start,
since `TravelSession` re-resolves the static profile by name today).

Recommendation: Option A first, consistent with Phase 5's and Phase 8's
precedent of landing the domain/module before touching an already-shipped,
tested phase's persisted schema.

## Constraints and Deferrals

- Never advances `gametime`, the round counter, or moves any player.
- Never mutates combat, health, or the native `Character.CarryCapacity()`/
  `go.go` per-move throttle.
- An item with no configured `Weight` contributes 0 — never guessed.
- A malformed capacity formula or threshold table entry is rejected and
  logged at config parse time, never guessed.
- Deferred: mount cargo capacity (Phase 10), stealth/noise load effects,
  per-companion carried gear, cargo loss/theft/raiding, and (per the
  recommendation above) actual travel-duration/fatigue multiplier wiring
  into `modules/expedition`/`modules/camping`.

## Acceptance Criteria (for this phase's slice)

- `items.ItemSpec.Weight` exists, defaults to 0, and is honored by the
  encumbrance calculation without changing any existing item's behavior.
- A company's total weight, capacity, and load ratio are computable and
  displayable, matching the handoff's example shape.
- `cargo put`/`cargo take` durably move item stacks between the leader's
  backpack and the company cargo container, refusing what would exceed
  capacity or what the leader doesn't have.
- Persists across restart/copyover like every other Ashveil registry.
- Focused domain/module tests, `make generate`, `make validate`, and
  `go test -race ./...` provide verification.

## Open questions for the owner

1. **Integration scope:** Option A (engine + display only, no
   `TravelSession`/camp-rest schema changes) for this phase, or Option B
   (wire the modifiers into travel/rest now)?
2. **Cargo command surface this phase:** full `cargo put`/`cargo take`
   commands moving real items now, or a smaller first slice — durable
   container + weight totals only, with item movement as a fast-follow?
3. **Capacity formula:** derive party capacity from company Strength
   stats (mirrors native `CarryCapacity()`'s Strength-scaling shape), or a
   flat/configurable base capacity (simpler, config-driven, like weather's
   biome tables)?
4. **Weight data for proving content:** should Dunmar's existing items get
   real `Weight` values authored this phase for a meaningful demo, or ship
   the engine with all current items at the zero-weight default and defer
   data authoring?
