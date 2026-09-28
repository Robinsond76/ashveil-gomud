# Phase 32f: Company Logistics — Plan

Design: [32f design](../specs/2026-09-28-phase-32f-company-logistics-design.md).
Decisions were confirmed by the owner on 2026-09-28 and are recorded in the
design.

## Task 1: Item fields (`internal/items`)

- [x] Tests first (`internal/items`): `carrybonus` and `saddle` decode from
  YAML; `CarryBonusGrams()` reads the base spec, like `Weight()`.
- [x] `ItemSpec.CarryBonus` (grams, `carrybonus`) and `ItemSpec.Saddle`
  (`pack` | `riding`, `saddle`).

## Task 2: Member capacity (`internal/company`, `modules/company`, `modules/encumbrance`)

- [x] Tests first:
  - `internal/encumbrance`: `MemberCapacity(base, perStrength, strength,
    pack)`; `WouldExceed` (at, under, over, nothing added, untracked).
  - `modules/company`: `CompanionCarry` counts the same companions as
    `CompanionGearGrams` (living, not charmed away), with the live mob's
    Strength and its largest pack; a stored companion has its pack and no
    Strength.
  - `modules/encumbrance`: capacity is the leader's share plus each
    companion's plus the horses; `CapacityKg` set in config is ignored
    with a warning; the load split and capacity split in `cargo`.
- [x] `company.MemberCarry`, `CarryProvider`, `CompanionCarry`,
  `CountedMembers`.
- [x] `encumbrance.Load.MemberCapacityGrams`/`MountCapacityGrams`;
  `WouldExceed(leader, grams)`; config `MemberBaseKg`, `StrengthKg`.
- [x] `shipped_weights_test.go`: the per-member share replaces the 200 kg
  figure.

## Task 3: One weight limit (`internal/usercommands`, `internal/mobcommands`, `modules/market`, `modules/gmcp`, `internal/users`)

- [x] Tests first (through the real commands): at capacity, `get` from the
  floor, a container, and a corpse is refused; `buy` is refused with no
  gold taken; `market buy` likewise; `give` from another player is
  refused; `give` to your own companion works; a companion mob's `get` is
  held to its leader's capacity; carrying more than the old item count no
  longer raises `go`'s cost.
- [x] `canCarry(user, grams)` in `usercommands`; the refusals.
- [x] Remove the count throttle in `go.go`; `inventory` and `peep` show
  an item count only; the prompt's `{I}` shows capacity in kg; GMCP drops
  the backpack `Max`.

## Task 4: Horses and saddles (`internal/mount`, `modules/mount`)

- [x] Tests first:
  - `internal/mount`: a horse's capacity (bare, saddled); riders; the
    per-kind cap; `Herd` validation.
  - `modules/mount`: the old single-mount save migrates to one horse with
    the legacy pack saddle; `stable` refused outside a stable room and
    without gold, and past the cap; `saddle` fits a matching saddle from
    the pack and returns the old one; a wrong kind is refused; `unsaddle`;
    `release <horse>`; `Relief` counts saddled riding horses; the route
    pace needs every counted member mounted; restart keeps the herd.
- [x] `Horse`, `Herd`, mount spec `Kind`, `Price`, `BareCapacityKg`,
  `SaddledCapacityKg`; config `StableRoomTag`, `LegacySaddleItemId`.
- [x] The `mount` command: list, `stable`, `release`, `saddle`,
  `unsaddle`.

## Task 5: Cargo keeps uses (`internal/encumbrance`, `modules/encumbrance`)

- [x] Tests first: `DepositItem` stacks full items and keeps a partly used
  one apart; `Withdraw` prefers the partly used; a stored cargo without
  uses loads as full; `cargo put`/`take` round-trips a 3-use waterskin;
  `ConsumeUse` takes one use and saves.
- [x] `CargoStack.Uses`; the module's put/take; a `CargoProvider` seam
  (`Contents`, `ConsumeUse`).

## Task 6: `company inventory` (`modules/company`)

- [x] Tests first (real command): every member's weight, pack, worn and
  carried items; a fallen companion; horses; cargo with uses; the load
  line.
- [x] `company inventory` / `company inv`.

## Task 7: `company eat`, `drink`, `meal` (`modules/company`)

- [x] Tests first:
  - the planner (pure): most in need first; skips the top band; source
    order cargo, own pack, leader's pack; smallest item that covers the
    need, else the largest; never an item with another buff; food that
    also waters counts in `meal`.
  - wiring (real command, shipped items): `company drink` with water
    only in cargo; `company meal` drawing each source in order; a
    companion's own pack; a leader's buff; the room line; nothing left.
- [x] `provision.go`: the planner and the command, through
  `survival.CompanyNeeds` and `survival.Provision`.

## Task 8: Content

- [x] Items: satchel (+5 kg), traveller's pack (+10 kg), frame pack
  (+15 kg), pack saddle, riding saddle; weights in range.
- [x] Mount types: `pack-horse`, `riding-horse`; stable tags on Dunmar
  West Gate and the Trappers' Post; packs and saddles in both markets;
  a satchel in each starter kit.

## Task 9: Player help and tutorial

- [x] `help cargo`, `help mount`, `help encumbrance`, `help inventory`,
  `help get`, `help buy`, `help eat`, `help drink`, `help set-prompt`
  updated; new `help company-inventory` and `help company-meal` (aliases
  `company inventory`, `company inv`, `company eat`, `company drink`,
  `company meal`), listed in `keywords.yaml`, linked from `help company`.
- [x] The Survival lesson's hint mentions `company meal` and
  `company inventory`.
- [x] Render tests; `TestTutorialHelpPointersExist` passes.

## Task 10: Docs, verification, review

- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Independent review; verify and fix findings with regression tests.
- [x] `docs/PROJECT_STATUS.md`: the 32f row, a work-log entry with
  **Review:**, and the stale "capacity is flat" known issue removed.
