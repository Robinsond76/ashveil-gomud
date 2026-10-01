# Phase 33g: Company Equipment and Loot

Status: management complete, 2026-10-01;
independently reviewed and verified. Catalog/class delivery remains separate. The owner requested implementation,
chose pooled gold, and removed equipment presets from this phase. Routine
33-series defaults are delegated to the lead. See the
[execution plan](../plans/2026-10-01-phase-33g-equipment.md) and
[company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md).

## Goal and scope

Manage equipment directly by durable member key and exact item reference,
using one company cargo and treasury. Compare actual defense, reach, burden,
and capacity before deliberate assignment. Equipment and formation presets
are excluded; no automatic whole-company upgrades are applied.

The complementary [equipment catalog, tiers, and Glaivewarden
design](2026-10-01-equipment-tiers-glaivewarden-design.md) is approved, but its
catalog migration, balance verification, and class ability are separate
remaining content/class slices. This management delivery does not ship them.

## Prior art and integration

Company inventory and 22b snapshots own companion gear; encumbrance owns
company capacity. Existing give/ask/equip routes used private backpacks and
`gearup` ranked prices. Direct company operations replace that preparation
workflow, while legacy leader equip/remove use the same backend. Ordinary
pickup, purchase, sale, gifts, consumption, cooking, and camp rewards continue
through existing entry points using the shared cargo source.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness and specialists. They are design
inspirations, not promises to reproduce their mechanics.

## Owner decisions and behavior

- All unworn possessions, including living companions' carried items, become
  company cargo. Preserve full instances: UUID, uses, enchantments, sharp
  edges, binding, and quest metadata. Worn gear remains on its member.
- Pool leader and living companions' carried gold into one treasury. The
  leader's bank balance remains the bank balance. Fallen members' retained
  body assets follow existing resurrection rules and are pooled on return.
- `company equipment` lists exact references; `company equip [member] [item]`,
  `company remove [member] [slot]`, and `company compare [member] [item]`
  resolve a durable companion key (`#N`) or leader. Missing, ambiguous, stale,
  foreign, dead, absent, or fighting targets are refused without moving gear.
- Transfers simulate normal wear/remove checks, including curses, bound items,
  eligibility, and two-handed displacement. Comparison shows role, weapon
  dice/hands/reach, protection and shields, personal burden, and the resulting
  Strength-based company capacity. It makes no automatic upgrade choice.
- `company treasury` shows pooled carried gold. Purchases and multiplayer
  gifts use it through existing gold semantics.
- Shared cargo contributes expedition load once. Personal dodge burden counts
  worn equipment (and existing pet burden), not cargo. One physical pack bonus
  may serve each living member; assign the largest available bonuses first.
- `loot` collects eligible corpse drops up to capacity. `loot own` collects
  only this company's claims; optional `autoloot` queues that command outside
  battle and starts off. Claimed spoils are private for two game hours, public
  for two more, then inaccessible even before cleanup. Existing allied
  rotation and unaffiliated damage-based claimant selection remain.
- No local action advances global time. Battle policy applies to direct,
  legacy, follower, and browser routes. No item use during battle.

## State ownership, persistence, and recovery

The leader's existing `Character.Items` and `Gold` fields back cargo and
treasury, keeping item commands compatible. Encumbrance converts old stacks
once: persist instances, applied operation IDs, and a migration marker in the
user file before removing the legacy entry. A failed cleanup cannot import
items twice. User writes use atomic replacement even when normal careful-save
configuration is disabled. Item UUIDs now persist across saves; newly spawned
mob gear receives fresh identity rather than copying template UUIDs.

Incoming gifts, pickups, shop proceeds, scripted rewards, and automatic cargo
mutations first recover the recipient. Failed recovery refuses the mutation
before source assets move; quest reward events retry. Successful shared writes
emit a leader inventory refresh alongside actual equipment/ownership events.

Equipment and pooling cross user/company files. First write the resulting
companion states and a durable operation containing resulting leader assets
into the company file; then apply the leader assets with the operation ID in
an atomic user save. Only then clear the journal. Recovery runs before user
commands and before/after login companion restoration. If the user already
acknowledged an operation, retry only its company acknowledgement; never
restore an old cargo snapshot over later changes. Failed writes retain a
recoverable journal or roll back uncommitted state.

GMCP and browser views report this state and invoke the same backend. They
hold no asset ownership. Runtime corpses retain the engine's existing restart
limitation: uncollected corpses disappear on restart.

## Acceptance criteria and verification

Cover real leader/companion assignment, duplicate exact items, two hands,
cursed/bound/incompatible gear, foreign/dead/absent members, battle refusals,
capacity and physical pack counting. Verify user save failure, company journal
failure, acknowledgement failure, restart/copyover recovery, and no replay of
an acknowledged operation. Exercise legacy migration, meal/cooking consumers,
camp deposits, pickups, gifts, and purchase entry points. Preserve item
metadata and pool assets exactly once.

Test loot private/public/expiry boundaries, same-name corpses, opt-in automatic
collection through real commands, and no automatic collection of foreign
public spoils. Browser controls must send durable member IDs and exact item
references. No travel/rest/preparation action may advance shared time.

An independent reviewer checks the full diff for bugs, design gaps, missing
coverage, and inaccurate help; fix confirmed findings with regressions. Run
`make generate`, `make validate`, `go test -race ./...`, JavaScript lint, and
the browser harness. Record actual outcomes in Project Status.

## Player help and tutorial

Ship indexed `help equipment`, `equip`, `gearup`, `treasury`, and `loot`.
Update cargo, inventory, company inventory, meals, burden, combat, and webclient
help with their new ownership and commands. Preparation and departure tutorial
hints point to equipment, treasury, and loot. Rendering and tutorial pointer
tests must pass. No loadout/preset help or controls are shipped.
