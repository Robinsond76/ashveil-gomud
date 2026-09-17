# Phase 4 Survival State Design

**Status:** Implemented and corrected through the durable identity reservation

**Goal:** Add durable, individual hunger, thirst, and long-term fatigue for every company leader and current companion, with manual food/water provisioning and no background time drain or gameplay penalties.

## Decisions

- Needs change only through explicit calls from gameplay systems. Phase 4 has no idle or logged-out drain.
- Hunger, thirst, and fatigue are each `0..100` integers. `100` means fully supplied or rested; `0` means depleted or exhausted.
- State is individual, not a party aggregate. It belongs to stable company member keys, never a runtime mob `InstanceId`.
- Food and drink can target the leader or a named companion, but the consumed item comes from the leader's existing backpack.
- Phase 4 provides warnings and status labels only. It does not apply health, combat, movement, or travel effects.
- Shared cargo, capacity/weight, automatic provisioning, weather, terrain, camping, and travel remain later work. Cargo is Phase 9 scope.
- Travel, rest, and survival must never advance the global GoMud world clock.

## Scope

### Included

- A pure `internal/survival` domain model and service API.
- Persistent state for leader and company companions.
- Threshold bands, status labels, and crossing results.
- Data-driven `nutrition` and `hydration` fields on GoMud item specifications.
- `eat` and `drink` integration for leader and named-companion provisioning from the leader's backpack.
- A `survival` status command that renders the current company roster.
- Company lifecycle integration so dismissed companions lose their survival record and a future companion cannot inherit it.
- Unit, module, command, persistence, and race coverage.

### Excluded

- Passive drain while idle or offline.
- Timers, `NewTurn` listeners, global clock mutations, and countdown-specific travel logic.
- Combat, health, action-point, room-movement, or travel-speed penalties.
- Camp/rest commands, sleeping, terrain, weather, injuries, mounts, cargo capacity, weight, and auto-provisioning.
- Companion inventories or a second inventory model.
- New food/drink content beyond the metadata needed to prove the integration path.

## Architecture

`internal/survival` owns validation, value mutation, band calculation, and the explicit service operations. It has no dependency on GoMud users, mobs, rooms, commands, plugins, clocks, or runtime instances.

`modules/survival` owns durable registry load/save, current-company roster projection, status rendering, and the adapter used by commands. It consults the authoritative company roster by stable leader and companion identity. The existing `modules/company` lifecycle notifies this boundary after successful summon/dismiss changes so survival records are initialized or removed deterministically.

The existing native `eat` and `drink` commands continue to perform item lookup, subtype validation, use-count handling, and ordinary buff behavior. Their Phase 4 adapter obtains the selected target, validates that it is the leader or a current companion, applies survival benefit before irreversible item consumption, then emits the normal consumption messaging plus any status-band crossing. The adapter only consumes from the leader's `Character.Items`.

## Domain Model

The domain package defines:

```go
type MemberKey string

const LeaderMemberKey MemberKey = "leader"

func CompanionMemberKey(id int) MemberKey

type Needs struct {
	Hunger  int `yaml:"hunger"`
	Thirst  int `yaml:"thirst"`
	Fatigue int `yaml:"fatigue"`
}

type Band uint8

type Change struct {
	Before Band
	After  Band
}

type Exertion struct {
	Hunger  int
	Thirst  int
	Fatigue int
}
```

`Needs` is always normalized to `0..100`. New or missing state defaults to `{Hunger: 100, Thirst: 100, Fatigue: 100}`. Food and water increase hunger/thirst, rest increases fatigue, and exertion reduces the relevant values. All operations return `Change` values for changed needs so callers can announce or react to threshold crossings without duplicating threshold logic.

The public service operations are:

```go
ApplyExertion(member MemberKey, cost Exertion) (hunger, thirst, fatigue Change, err error)
ApplyRestRecovery(member MemberKey, fatigue int) (Change, error)
ConsumeFood(member MemberKey, nutrition, hydration int) (hunger, thirst Change, err error)
ConsumeWater(member MemberKey, hydration int) (Change, error)
NeedsFor(member MemberKey) (Needs, bool)
BandFor(value int) Band
```

The registry/service boundary validates member identity before mutation; zero or negative benefits/costs are rejected rather than silently changing state in the opposite direction.

## Thresholds and Presentation

Every need uses these numeric bands:

| Range | Band | Hunger label | Thirst label | Fatigue label |
| --- | --- | --- | --- | --- |
| 76–100 | full/rested | Well fed | Hydrated | Rested |
| 51–75 | steady | Sated | Comfortable | Ready |
| 26–50 | low | Hungry | Thirsty | Tired |
| 1–25 | critical | Starving | Parched | Exhausted |
| 0 | depleted | Starving | Dehydrated | Collapsed |

The `survival` command renders the leader and every current company companion, including a companion whose native mob is temporarily unavailable. It shows each value and label. A command announces only a band crossing for the provisioned member; it does not broadcast to the room or create a global event.

## Consumables and Commands

Extend `items.ItemSpec` with optional fields:

```go
Nutrition int `yaml:"nutrition,omitempty"`
Hydration int `yaml:"hydration,omitempty"`
```

The fields are non-negative. Existing item files without them remain valid and retain their existing behavior. Phase 4 verifies the path using one edible item with `nutrition` and one drinkable item with `hydration`; adding a larger ration catalog is content work, not a requirement of this phase.

Command syntax:

```text
eat <item> [member]
drink <item> [member]
survival
```

With no member selector, `eat` and `drink` target the leader. A member selector accepts the existing company forms: `leader`, `me`, `self`, `#<companion-id>`, or an unambiguous companion name. The selected consumable is found only in the leader's backpack; companion inventories are not searched or modified. The target must be a current roster member. Existing food/drink subtype rules remain in force.

If applying survival state or persisting it fails, the item remains unconsumed and no normal consumption effect/buff is applied. Once state persistence succeeds, existing use-count, item-ownership, and buff behavior proceeds normally.

## Lifecycle and Persistence

The survival module persists a map keyed by leader user ID and stable member key. It loads through the plugin's byte/struct persistence path, preserves a read failure as unavailable state, and refuses mutations while persistence is unavailable. On decode, it normalizes values and discards malformed/unknown entries. It never writes live mob IDs.

The current company roster is authoritative. On a successful summon, default
survival state is created for the new companion and survival persists a
per-leader lower bound for the next companion ID in the same save. Before
assigning an ID, company adopts that lower bound. On dismiss or dismiss-all,
state for removed companion(s) is pruned while the reservation is retained.
Therefore, even if the separate company write or cleanup fails and the process
restarts, a survival-persisted companion identity is never reassigned. Roster
projection also prunes ordinary orphan state defensively on load and
status/read paths.

The leader needs state even with no companions and no formation record. Its first status or provisioning operation creates the default record. Normal login, logout, copyover, and companion re-spawn preserve the same durable state.

## Future Cargo Integration

Phase 7 will introduce camp, rest, and sleep commands. Those commands will use real elapsed server time and call `ApplyRestRecovery`; they must not instantly restore fatigue or advance the shared GoMud clock. Camp eligibility, safe locations, sleep duration, and interruptions are Phase 7 rules, not Phase 4 rules.

Phase 9 will introduce the company-owned shared cargo inventory, item capacity/weight rules, and provisioning policy. It may select an eligible ration or water item automatically when a member enters a configured low band. It must announce consumption, respect an opt-in policy, and use the same `ConsumeFood` or `ConsumeWater` service operation. No Phase 4 type assumes a backpack location, so cargo can become an alternate item source without changing survival semantics.

## Acceptance Criteria

- A leader and every current companion have independent, persistent `0..100` hunger, thirst, and fatigue values.
- New/missing state starts fully supplied/rested; decoded out-of-range values normalize safely.
- Explicit exertion and rest APIs mutate only the requested values and report band crossings.
- No idle/offline drain, timer, global-time mutation, or mechanical penalty exists.
- `eat` and `drink` retain GoMud's normal item validation and behavior while applying metadata-driven survival benefits to the selected roster member.
- Items are consumed only after survival mutation/persistence can succeed; an unavailable or failed persistence store leaves the item and existing buff state untouched.
- `survival` displays leader plus current roster with values and need-specific labels.
- Dismissed-companion state is removed, and stale state cannot appear for a newly summoned companion.
- Existing item specs without survival metadata and vanilla food/drink behavior remain valid.
- Focused tests, `make generate`, `make validate`, and `go test -race ./...` pass; document any environmental blocker rather than claiming success.

## Risks and Follow-up Boundaries

- Native `eat`/`drink` mutate item instances, buffs, and item-ownership events. The implementation must introduce a narrow, failure-aware survival adapter instead of reimplementing those command paths in a module.
- Company and survival persistence are separate direct plugin writes, not a
  cross-file transaction. Phase 4 compensates failed summon/dismiss operations
  and uses a survival-side durable companion-ID reservation to prevent identity
  reuse after the company write fails. A partial low-level file write remains
  unrecoverable.
- Phase 5 travel will be the first normal caller of `ApplyExertion`; it must accrue costs by real progress, not an upfront estimate or world-time mutation.
- Phase 7 camping/rest/sleep will call `ApplyRestRecovery`; it must use real elapsed time/server state, not local time skipping or instant restoration.
- Phase 9 cargo owns all automatic supply selection and policy. Phase 4 remains manual leader-backpack provisioning.
