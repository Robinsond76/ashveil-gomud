# Phase 23b: Whetstones and *Sharpened* Weapons

## Prior-art check

- **Spec:** the "Whetstone behavior" section of
  [rest and weapon preparation](2026-09-23-rest-weapon-preparation-design.md),
  with the owner's 2026-09-24 amendment: a whetstone is used on demand at
  any time, has 10 uses, and spends one use per member sharpened. Phase
  23a shipped the rest-tier half.
- **Items** (`internal/items`): an `Item` instance is a value type carried
  in `Character.Items` and `Character.Equipment` (`characters.Worn`). Its
  `Uses` field is durable, and `Validate` refills `Uses` from the spec when
  it is 0, so a spent consumable has to be removed, not kept at 0. UUIDs
  are runtime only.
- **Weapon subtypes:** `slashing`, `cleaving`, `stabbing`, `bludgeoning`,
  `shooting`, `claws`, `whipping`, `generic`. Only an offhand item whose
  type is `weapon` attacks.
- **Combat** (`internal/combat`): `calculateCombat` takes both characters
  *by value*, resolves the attacker's weapons (main hand, then an offhand
  weapon; dual-wield can drop one at random), rolls each strike, and
  returns an `AttackResult`. The four `Attack*Vs*` functions, called from
  `NewRound_DoCombat` and the Phase 11 formation hooks, apply the result
  to the real characters. A companion's attacks are `AttackMobVsMob` /
  `AttackMobVsPlayer`.
- **Companion gear** (Phase 22b): a companion's equipment lives on its
  live mob and is snapshotted to the durable company record on plugin
  `OnSave` (autosave, shutdown, copyover) and on the leader's
  `PlayerDespawn`; `company gear` refreshes from the live mob first. The
  engine writes user files at the same save points. Any field on
  `items.Item` therefore persists for the leader and companions with no
  new store.
- **Camping** (`modules/camping`): camp rest completion is granted on the
  game loop by `grantPendingTiers` (Phase 23a), with `companions(leader)`
  giving live companion characters by companion ID.
- **Combat check precedent:** Phase 21a desertion waits while the leader
  or the companion has `Aggro` set.
- **Markets** (Phase 19/19b): `market buy` mints `items.New(id)`, so a
  bought stone has its spec's 10 uses; `market sell` pays the same price
  for any unit of a good.

## Scope

In: the whetstone item and its sale in shipped markets; the per-weapon
*Sharpened* edge on item instances; the `sharpen` command (and `camp
sharpen`), preview, and auto setting; combat applying and spending edges;
edge display in inventory, inspection, `company gear`, and `conditions`.

Out: company cargo as a stone source (the leader's pack only); a
sharpening skill; edges on non-bladed weapons.

## Decisions

The four open details are the owner's (2026-09-24). The rest are this
design's recommendations, applied under the owner's standing "proceed with
your recommendation" instruction, as in 22a–23a.

1. **Two blades on one member spend one use.** (Owner.)
2. **A member whose blades are all sharpened, or who has no bladed weapon,
   spends nothing.** (Owner.) A member with one sharpened and one dull
   blade has the dull one sharpened for one use; the sharpened one is
   untouched (no stacking, no reset).
3. **Running out partway:** members are sharpened in a fixed order,
   leader first, then companions by company number (ascending ID). When
   the stones run dry the pass stops, and the reply names who was left
   out. (Owner.) *Recommendation on top:* if the leader carries more than
   one whetstone the pass continues onto the next stone, most-used first,
   so partly used stones are finished before a fresh one is started; a
   member is only left out when every stone is empty.
4. **Not during combat.** Sharpening is refused while the leader or any
   live companion has an attack target (`Aggro`), the same check
   desertion uses. (Owner.) Nothing is spent.
5. **Bladed** means an equipped weapon whose subtype is `slashing`,
   `cleaving`, or `stabbing`, in the main hand or offhand. Claws,
   bows, whips, blunt weapons, `generic`, and unarmed attacks never
   qualify.
6. **Present** means the leader, and each companion with a live mob in the
   leader's room. A companion elsewhere (or not spawned) is not sharpened,
   costs nothing, and is named as not here. A whetstone is used by hand. (Review fix: a rostered
   companion with no live mob was first dropped silently; it is now named
   from the survival roster.)
7. **The edge is two durable fields on the item instance:**
   `sharpbonus` and `sharpstrikes` (`items.Item`, `omitempty`). The bonus
   is copied from config when sharpened, so combat needs no module config
   and an existing edge is unaffected by a config change. Value fields
   (not a pointer), because items are copied by value everywhere and a
   shared pointer would alias one edge across copies. Transferring the
   weapon transfers its edge; when strikes reach zero the bonus is
   cleared.
8. **Combat.** Each successful strike (it hit and was not dodged) by a
   weapon with an edge adds its bonus to that strike's damage, before
   armor reduction, and spends one strike. `calculateCombat` tracks
   strikes spent per slot (main hand, offhand) across the whole round and
   returns them on `AttackResult`; the four `Attack*` functions then
   spend them on the attacker's real equipment. The simulator does not
   spend them. Misses, dodges, pets, and shots never spend an edge.
9. **Atomic on the game loop, durable at the engine's save points.** A
   sharpen runs inside a user command or the `NewRound` listener, both on
   the game loop, so it changes the stones and every weapon in one step
   with nothing between. The parent spec's operation ID and per-weapon
   completion log were for a rest-timer-driven sharpen; on demand, the
   state rides on the user file and the company record, which the engine
   writes together at autosave, copyover, shutdown, and logout (the model
   22b adopted for every companion gear change). A crash between two
   saves usually loses both the edge and the use together. *(Corrected
   after review: not strictly together.* The company file is also written
   by company commands, a companion's death, and legacy upgrades, after
   `company gear` or an item event has refreshed the snapshot in memory.
   A crash in that window can keep a companion's edge and refund the
   stone use. That needs a server crash and is 22b's accepted window for
   every companion gear change.) A retry is also
   harmless by construction: sharpened blades cost nothing (decision 2),
   so a repeated pass can neither sharpen a blade twice nor spend a
   second use on it.
10. **Commands.** `sharpen` sharpens the company. `sharpen status` shows
    each present member's blades, which would be sharpened, the uses it
    would spend, and the stones' uses left. `sharpen auto on|off` sets
    the auto preference. `camp sharpen [status|auto on|off]` is the same
    command, as the parent spec named it, and `camp status` adds the auto
    setting and the preview.
11. **Auto mode** (off by default, durable per leader in the camping
    registry as `auto_sharpen`): when a camp rest's Rested grant is saved
    (Phase 23a's grant pass), a leader with auto on gets the same pass.
    It is silent when there is nothing to sharpen or no stone, and says
    so when it was skipped for a fight. A manual sharpen earlier needs no
    special casing: already sharpened blades cost nothing.
12. **The whetstone** is item 30 (`object`/`mundane`, `uses: 10`),
    carried in the leader's pack. The inventory shows its uses left, as
    it does for consumables. A spent stone is removed (and an
    `ItemOwnership` lost event queued), because `Validate` would refill a
    stone kept at 0 uses.
13. **Markets sell it**: Dunmar and Old Kings Road list the whetstone as
    a market good. So a stone can't be bought, mostly used, and sold back
    for full price, a market only buys a stone with all its uses (a used
    one is "not the ordinary article"). The engine's `IsSpecial` already
    counts an item with uses spent as special, and markets never buy
    special items, so this needs no new code (the review found the first
    draft's extra check redundant).
14. **Display.** Inventory (worn and carried), `look`/`inspect` of the
    item, and `company gear` show `(sharp: N)`; the item's description
    adds "Its edge is honed: +B damage for its next N strikes."
    `conditions` adds a *Sharpened* row for the leader's own edged
    weapons.

## Durable model

`items.Item` (user files, room items, company records):

```yaml
weapon: {itemid: 10002, sharpbonus: 1, sharpstrikes: 17}
```

`modules/camping` registry: `auto_sharpen: {7: true}` (`omitempty`, so
older files load unchanged).

Config (`modules/camping`): `WhetstoneItemId: 30`, `SharpenedBonus: 1`,
`SharpenedStrikes: 20`.

## Module and engine changes

- `internal/items`: `SharpBonus`/`SharpStrikes`, `Sharpened()`,
  `Sharpen(bonus, strikes) bool` (never stacks or resets),
  `SpendEdge(n)`, `EdgeLabel()`, the description line, and
  `NameComplex` showing the edge.
- `internal/camping/sharpen.go` (pure): `Bladed(subtype)`,
  `SharpenMember`, `PlanSharpen(members, uses)` → ordered outcomes and
  uses spent.
- `internal/combat`: slot-tracked weapon resolution, the per-strike bonus,
  `AttackResult.EdgeSpent`, and `spendEdges` in the four `Attack*`
  functions.
- `internal/usercommands`: the inventory shows uses for objects with uses
  and edges on carried weapons; `conditions` shows *Sharpened*.
- `modules/camping`: `sharpen.go` (command, preview, pass, stones, auto),
  config, registry field, the auto hook in `grantPendingTiers`, and `camp
  sharpen`/`camp status`.
- `modules/company`: `company gear` shows the edge.
- `modules/market`: whetstone goods and the full-uses sale rule.
- Content: `_datafiles/world/default/items/other-0/30-whetstone.yaml`.

## Constraints

- Characters are only touched on the game loop (commands, `NewRound`);
  `m.mu` guards only the registry and is never held while calling
  another module or touching a character.
- No global time or round advance; sharpening changes no rest recovery.

## Acceptance criteria

- One `sharpen` sharpens every present member's dull blades and spends one
  use per member sharpened; two blades on one member spend one; members
  with no dull blade spend none; non-bladed and already sharpened weapons
  are unchanged.
- With too few uses, members are sharpened leader first then by number,
  and the rest are named as left out.
- Refused, with nothing spent, while the leader or a companion fights.
- A spent stone leaves the pack; a second stone carries on.
- Combat adds the bonus and spends one strike per successful strike of
  that weapon only, for a player and for a companion mob, through the
  real `Attack*` functions; at zero the edge ends.
- The edge survives a save/load of the item (user file and company
  record), and moves with the item.
- Auto mode sharpens once at camp rest completion through the real
  `NewRound` listener and does nothing when off.
- Wiring tests go through `sharpen`/`camp` user commands, `NewRound`, the
  combat `Attack*` functions, the inventory and conditions panels, the
  `company gear` view, and `market sell`.
