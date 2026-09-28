# Phase 32f: Company Logistics — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)). What a company can
carry, where it's kept, and feeding everyone at once.

## The owner's notes

1. "Company load is 200 kg with no one in the company."
2. "Load should come from horses, saddles, backpacks."
3. "I'd like to see every member's equipment together."
4. "A company inventory with cargo on the same screen."
5. "One command for everyone to eat and drink."

And the decision (2026-09-28, roadmap decision 2): **one weight limit
only**, no item-count limit. Horses, saddles, and backpacks raise it.

## Prior-art check

- **Capacity** is `CapacityKg: 200` in
  `modules/encumbrance/files/data-overlays/config.yaml`, plus
  `mount.CapacityBonus(leader)` (`modules/encumbrance/encumbrance.go`,
  `CurrentLoad`). Nothing about the members is read. `LoadBands` (75%,
  90%, 100%) slow route travel and raise walking strain; nothing blocks
  movement by weight.
- **The load** is `Load.TotalGrams()`: the leader's items and worn gear,
  `company.CompanionGearGrams` (Phase 28: living companions, the live mob
  when out, else its record or template; a fallen companion's gear stays
  with the body; one charmed away counts nothing), and the cargo. Every
  reader (departure, strain, `cargo`, `inventory`, `status`, GMCP) goes
  through it.
- **Mounts** (`modules/mount`, `internal/mount`): one per company, keyed
  by leader. `mount stable <type>` assigns one anywhere, free; the only
  type is `pack-horse` (+100 kg, travel 90%, fatigue 75%, 2 riders).
  There are no saddles.
- **Backpacks:** no such item. The worn slots are weapon, offhand, head,
  neck, body, belt, gloves, ring, legs, feet (`items.AllEquipSlots`); no
  back slot.
- **GoMud's count limit** is still live: `Character.CarryCapacity()` is
  `5 + Strength/3` items. Walking with more items than that costs 50
  action points instead of 10 ("You're too encumbered to move",
  `internal/usercommands/go.go`), and `inventory` and `peep` print
  `(n/max)`; GMCP and the prompt expose it too.
- **Seeing gear:** `company gear <member>` shows one companion
  (`modules/company/state.go`, `gearView`); `inventory` shows the
  player; `cargo` shows the load split and the cargo. Nothing shows them
  together.
- **Cargo** is stacks of `(ItemId, Count)` (`internal/encumbrance`).
  `cargo take` makes a fresh item with the spec's full `Uses`, so **a
  half-drunk waterskin put in cargo comes back full** (found while
  researching this; an existing bug).
- **Eating:** `eat`/`drink <item> [member]` take from the player's own
  pack only and provision one member through `survival.Provision`. Item
  buffs (Well Fed, Hydrated) go on the player even when a companion is
  fed. Cargo is never used.
- **Giving a companion something:** GoMud's `give <item> <mob>` works on
  any mob in the room (`internal/usercommands/give.go`); a companion's
  pack is recorded from the live mob (`refreshSnapshot`). Inferred, not
  yet tested: a given backpack would land in the companion's pack.

## Decisions

Recommendations below are **not yet confirmed by the owner**; each is
marked **(open)** until Robinson answers on the draft PR.

### A. Capacity comes from the members

The flat 200 kg goes. Capacity is the sum of what each member carries,
plus what the mounts carry:

```
capacity = Σ members (MemberBaseKg + StrengthKg × Strength + best pack)
         + Σ mounts   (mount type's base + its saddle)
```

- **Who counts** is the same set the load already weighs: the leader,
  and each living companion not charmed away. A fallen companion adds
  neither weight nor capacity; dismissing one takes both.
- **Starting numbers** (config, `modules/encumbrance`): `MemberBaseKg:
  20`, `StrengthKg: 0.5`. Strength is a few points at level 1 (a
  human's base is 1 per level, `races/1-human.yaml`), so a fresh player
  carrying a 5–8.5 kg starter kit has a little over 20 kg, about a third
  full; a company of five has about 100–110 kg before packs or horses.
  Strength matters more as members level. **(open: the numbers)**
- `shipped_weights_test.go`'s "a fresh company of five stays under 30%"
  becomes "a fresh member stays under 40% of their own share", since
  capacity is now per member.
- `CapacityKg` is retired. A saved config that still sets it is
  ignored with a warning.

### B. Backpacks

A backpack is a **carried** item with a new item field, `carrybonus`
(grams). Each member counts **one** pack, the largest they carry; a
second pack is just weight. It weighs what it weighs, like anything
else. **(open: carried, not worn)**

- The alternative is a new worn `back` slot. It reads more naturally
  ("wear backpack"), but touches GoMud's `Worn`, every slot list, the
  gear window, GMCP, and the save format, for no gameplay difference.
- Shipped content: three packs (a satchel +5 kg, a traveller's pack
  +10 kg, a frame pack +15 kg), sold in the market; the starter kit
  gets the satchel.
- A companion gets a pack with `give`. Its pack counts from the live mob
  when it's out, or its record, the same way its gear's weight does.

### C. Horses and saddles

- **More than one mount.** A company keeps up to `MaxMounts` (config,
  default 3) mounts. The mount record becomes a list per leader; each
  has an id, a type, and a saddle. Riders add up across mounts (each
  still carries the leader first, then companions by id).
  **(open: the cap)**
- **Mounts are bought, not conjured.** `mount stable <type>` works only
  in a room flagged as a stable, and costs the type's `Price` in gold.
  `mount release <mount>` sells it back for nothing. Today's free,
  anywhere assignment makes capacity free. **(open)**
- **Saddles** are items with a `saddle` field: its kind (`pack` or
  `riding`) and its bonus. `mount saddle <mount> <item>` fits one from
  the player's pack onto a mount (the old saddle goes into the pack);
  `mount unsaddle <mount>` takes it off. A saddle on a horse is part of
  the mount record, so it survives restart.
  - A **pack saddle** adds cargo capacity (+60 kg on a pack horse).
  - A **riding saddle** adds one rider and no capacity.
  - With no saddle a horse carries its base only.
- **Numbers:** a pack horse's base becomes 40 kg; with a pack saddle it
  carries 100 kg, today's figure. **(open)**
- **Existing saves:** a stored single mount becomes the first entry in
  the list, with a pack saddle fitted, so no one loses capacity on
  upgrade.

### D. One weight limit

GoMud's item-count limit goes, per the owner's decision:

- `go` no longer charges 50 action points for carrying more than
  `CarryCapacity()` items. Being heavy already costs strain and travel
  time through the load bands (`help strain`).
- `inventory` and `peep` show kilograms against the company's capacity
  instead of `(n/max)`. GMCP's inventory `Max` becomes the capacity in
  grams; the prompt token keeps working but reads the weight.
- `CarryCapacity()` itself stays for scripts that call it
  (`GetCarryCapacity`), marked deprecated.
- **Nothing blocks** picking up, buying, or walking by weight in this
  phase; being over capacity is the top load band (travel 150%, strain
  130%), as today. **(open: whether to add a hard stop, for example
  "can't move" past 150%)**

### E. `company inventory`

One screen for everything the company carries. `company inventory`
(also `company inv`) shows:

```
Company load: 61.2 kg / 175.0 kg (35%), of which mounts 100.0 kg.

Dain (you)            12.4 kg   pack: traveller's pack (+10 kg)
  Wearing: iron sword, leather jerkin, boots
  Carrying: waterskin (3 of 5), seared game meat x2, whetstone
#1 Tamsin Reed         9.1 kg   no pack
  Wearing: spear, padded coat
  Carrying: nothing
#2 Brother Oswin      ...

Mounts: pack horse (pack saddle, +100 kg)
Cargo (18.0 kg): rope, seared game meat x6, waterskin x2
```

- Each member's worn and carried items in one block, with their weight
  and pack. A fallen companion is listed as fallen, gear "with the
  body".
- Mounts with their saddles, then the cargo, from the same data
  `cargo` reads.
- `company gear <member>` stays for one member in full. `cargo` stays
  as it is, with the new capacity split added.
- The web client's view of this is 32g's Company tab; 32f adds the GMCP
  data it reads (members' packs, mounts, cargo with uses).

### F. Cargo keeps partly used items

Fix the half-drunk waterskin: a cargo stack gains `Uses`. Full items
still stack by id; a partly used item is its own stack of one with its
remaining uses, and `take` gives it back as it went in. Needed by G,
which draws single uses from cargo.

### G. `company eat` and `company drink`

One command feeds or waters everyone who needs it:

- **Who:** every living member present (leader and companions), most
  in need first. A member already at the top band ("Well fed",
  "Hydrated") is skipped.
- **From where:** the cargo first, then the leader's pack, then the
  member's own pack. **(open: the order)**
- **What:** one use of the edible (or drinkable) item that best covers
  that member's need without much waste; the smallest item that
  covers it, else the largest there is. Items with buffs other than Well
  Fed and Hydrated (potions, ale) are never used; those stay a choice
  made by hand.
- **Buffs:** Well Fed and Hydrated go on the member who ate, when that
  member is the player. A companion gets the provision only, as with
  `eat <item> <member>` today.
- **Output:** one line per member fed and what they ate, then anyone
  who still needs something because the food ran out:
  ```
  Tamsin Reed eats a seared game meat (cargo). Hunger: Well fed.
  You eat some of the cheese sandwich (your pack). Hunger: Sated.
  Brother Oswin is still hungry; there's nothing left to eat.
  ```
- The room sees one line: "Dain's company eats."
- `eat` and `drink` with an item keep working as they do.
- Each member is provisioned through `survival.Provision`, one call per
  member, so each save and rollback stays as it is; items are used only
  after that member's provision succeeds.

## Module

- `modules/encumbrance`: the member-based capacity; config
  (`MemberBaseKg`, `StrengthKg`); cargo uses; the capacity split in
  `cargo`.
- `internal/company` / `modules/company`: a provider of each counted
  member's Strength and best pack (read the way `CompanionGearGrams`
  reads gear); `company inventory`.
- `internal/items`: the `carrybonus` and `saddle` fields.
- `internal/mount` / `modules/mount`: the mount list, saddles, `MaxMounts`,
  prices, the stable room flag, the save migration.
- `internal/usercommands`: `go.go` (count limit), `inventory`, `peep`;
  `company eat`/`drink` share `eat.go`'s provisioning helpers.
- `modules/gmcp`: capacity in grams, packs, mounts, cargo uses.
- Content: packs and saddles in `items/`, a stable room and market
  stock, the starter kit's satchel.

## Invariants

- **The clock:** nothing here advances time. Eating is instant, as
  today.
- **Restart and copyover:** mounts, saddles, and cargo uses are saved
  in their modules; capacity is computed, never stored. The single-mount
  save migrates on load. A mid-command crash in `company eat` leaves
  some members fed and their items used, never an item used without
  its provision.
- **Locks:** capacity is read on the game loop, as the load is now. The
  company provider is read outside the encumbrance lock (as
  `companionGear` is today); `company eat` never holds the encumbrance
  and survival locks together.

## Acceptance criteria

- **Unit:** the capacity formula (members, Strength, one pack each,
  mounts, saddles, a fallen or charmed-away member); cargo stacks with
  uses (merge, split, round trip); the eat planner (who first, which
  item, which source, skipping the sated, running out); the mount list
  migration; saddle fit and swap.
- **Wiring** (shipped config, real commands):
  - a new player alone: `cargo` shows a little over 20 kg capacity, not
    200;
    recruiting a companion raises it; dismissing lowers it;
  - `give satchel tamsin` raises capacity by 5 kg; a second pack on the
    same member doesn't;
  - `mount stable pack-horse` outside a stable is refused; in one it
    costs its price; `mount saddle horse pack saddle` adds 60 kg;
    a second and third horse stack; a fourth is refused;
  - an old single-mount save loads as one saddled pack horse with the
    same capacity;
  - carrying 20 items no longer makes `go` cost 50 action points;
    `inventory` shows kilograms;
  - `cargo put` a waterskin with 3 uses, `cargo take` it: still 3;
  - `company inventory` lists every member, mounts, and cargo;
  - `company drink` with water only in cargo waters everyone thirsty,
    skips the hydrated, uses the cargo, and reports who went without
    when it runs out; a restart after it keeps what was used and what
    was provisioned.
- **Player help:** `help cargo` (member capacity, packs, horses,
  saddles, cargo keeps uses), `help mount` (several mounts, stables,
  prices, saddles), `help encumbrance` and GoMud's `help inventory`
  (no item count), new `help company inventory` and `help company eat`
  pages (aliases `company inv`, `company drink`), `help eat` and
  `help drink` pointing to `company eat`; the Survival lesson's hint
  mentions `company eat` and `company inventory`;
  `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- The web company dock, its Cargo tab and hover actions: 32g.
- Mount upkeep, feed, fatigue, and death; mounts in combat.
- Carts and wagons (a mount that needs a road).
- A hard weight stop, unless the owner picks it in D.
- Trading items between members from one screen (`company give`):
  `give` covers it for now.
