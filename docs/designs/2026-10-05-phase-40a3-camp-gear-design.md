# Phase 40a3: camp gear

Status: **approved by the owner on 2026-10-05**, including the item list,
weights, effects and prices (handoff rule 20). An execution plan comes
before implementation, which waits for the milestone to start. Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
It follows [40a2 gathering](2026-10-05-phase-40a2-gathering-design.md),
which makes camp fires need firewood. Art: S1 camp sprites (including
`camp-rough`) in the [sprite specification](2026-10-05-sprite-specification.md).
Item icons belong to the later S7 set.

## Goal

Give companies **durable camp gear** worth carrying. Each piece improves
camping in one clear way, and each one weighs something. So the choice of
what to haul, on your back or on a horse, becomes a real expedition
decision in a harsh world.

This complements, and does not repeat, the single-use
[camp consumables](2026-10-01-camp-consumables-design.md): fortifying
broth, draughts, watch incense and weapon oil. Gear is reusable;
consumables are spent.

## Prior art (code as of 2026-10-05)

- **Camp rest** (`modules/camping`):
  - restores 20 fatigue, scaled by the weather's `RestRecoveryPct`, which
    is locked when the rest starts;
  - a finished rest grants **Rested** (15 real minutes; cuts walking
    strain);
  - inns grant **Well Rested** (30 minutes).
- **Camp specialists** (33f3):
  - a warrior **watch** spots raiders, 25% per level;
  - a cleric **vigil** raises loyalty;
  - a ranger **forage** finds food;
  - raids in some zones (15% on the Old Kings Road) spoil an unspotted
    rest.
- **Camp cooking:** recipes from raw game meat and wild thyme (`camp
  cook`).
- The **whetstone** (`sharpen`) is the existing example of a carried camp
  tool.
- **A lit fire warms its room** (Phase 15 exposure).
- **Carrying:** personal load and agility (30g3), packs and capacity (34b)
  and horse cargo (32f) already make weight matter.
- **40a2:** `camp fire` needs firewood. A `shelter` room halves the
  weather's rest penalty (40a).

## Owner decisions

- **Camps need firewood** (2026-10-05). Approved from 40a2.
- **Camp items that boost camping effects** (2026-10-05). The design is
  left to the lead; this document is the proposal.
- **Bedrolls are a bonus only** (2026-10-05). Members without one recover
  normally.
- **Camp theft is a future feature** (2026-10-05, deferred). See
  "Deferred: camp theft" below.

## Approved design

### Fuel rule (refines 40a2)

- **One firewood bundle fuels one camp rest.** Lighting the fire spends a
  bundle, or uses the room's deadfall in a `firewood` room. The fire stays
  lit while the camp stands.
- **When a rest completes, the fire burns down to embers.** Resting again
  at the same camp needs it fed: `camp fire` again, spending another
  bundle or the deadfall. Warmth persists while embers glow, until the
  camp is broken.
- **Firewood rooms:** their deadfall can fuel any number of fires.
  Gathering bundles from them (40a2) still uses the room's pool.

### Gear catalogue

| Item | Weight | Covers | Effect | Notes |
|---|---|---|---|---|
| **Bedroll** | 2.5 kg | One member each | That member recovers **+25% fatigue** from a camp rest | Assigned leader first, then companions by number, from the leader's inventory, members' packs, then cargo |
| **Oiled canvas tent** | 9 kg | The company | Counts as **shelter** for the rest (halves the weather penalty, as in 40a). It also **blocks rest-time cold exposure** | Does not stack with a `shelter` room; the better one applies. Usually hauled by a horse |
| **Fire steel and tinder** | 0.2 kg | The company | **Damp firewood lights first time** and gives its full warmth | Never wears out |
| **Iron cookpot** | 3 kg | The company | Camp cooking makes **one extra portion** of any multi-ingredient recipe (hunter's stew, thyme-roasted game). Required for broths once camp consumables ship | Any member's pack or cargo |
| **Camp bells and trip lines** | 1 kg | The company | If no watch is posted, a **flat 20%** chance to spot raiders. With a watch, **+10 points**, capped at 90% (the incense cap) | Wears: 10 rests, then the lines are spent |
| **Field surgeon's kit** | 1.5 kg | The company | During a completed rest, a healer-role member with mana left treats **one lasting wound** on the most wounded member, before the usual rest effects | Wears: 5 uses. Pairs with 35b after-battle patching |

**Proposed prices** (provisioners): bedroll 8, tent 40, fire steel 5,
cookpot 12, bells 6 (and 3 to restring), surgeon's kit 25 (and 10 to
restock). These are balance defaults.

### Rules common to all gear

- **Gear counts if the company has it at camp:** in the leader's
  inventory, a present member's pack, or the cargo with its horse
  present. Separated or away members' gear doesn't count.
- **Gear effects are fixed when the rest starts**, as the weather is
  today. Losing gear mid-rest changes nothing until the next rest.
- **No stacking of the same item:** one tent, one cookpot and one set of
  bells count. Bedrolls count one per member.
- **Raids:** nothing is stolen in this phase. Camp theft is a deferred
  feature (below).
- **Text first:**
  - `camp` (the camp view) lists the gear in use and its effect in one
    line each;
  - `look` at a camp shows a tent when one is pitched;
  - the rest report states the bedroll and tent bonuses applied.

### Map and art

- **With a tent:** the map shows the `tent` camp sprite.
- **Without one:** the new **`camp-rough`** sprite, bedrolls around a fire
  ring, added to S1.
- Fire, smoke and resting overlays are unchanged (40b).

### Deferred: camp theft (owner, 2026-10-05)

A future camp event, not part of 40a3:

- **Only happens without bells and trip lines.** A company that has strung
  bells and trip lines is never robbed. Their pitch becomes: protect your
  sleep and your goods.
- **The theft is silent.** Thieves slip in during the rest, unseen. There
  is no fight and no warning.
- **The company notices on waking:** the rest report says what is missing,
  for example "When you wake, the cargo has been rifled. Missing: 2 raw
  game meat, a firewood bundle."
- **What can be taken:** some of the company's loot and supplies, from the
  cargo and unattended packs.

Details to decide when it is designed:
- the chance per rest, and the zones it happens in;
- what can be taken: never equipped items, and maybe never bound or quest
  items;
- caps on how much;
- whether a posted watch also protects;
- whether stolen goods can be tracked down;
- multiplayer fairness and the persisted rest record.

## State and persistence

- **Gear** is ordinary items. Inventory, packs and cargo already persist
  them. Wear uses the item's existing `uses`.
- **The rest's locked gear effects** are saved with the camp-rest record,
  as the locked weather percent is today. A copyover mid-rest keeps them.
- **The fire's ember state** is part of the camp record and is persisted
  with it.

## Integration points

| Area | Change |
|---|---|
| `modules/camping` | fuel per rest and embers; a gear resolver at rest start; bedroll fatigue; tent shelter and cold; cookpot portions; bells in the watch roll; surgeon's kit wound treatment; camp view and rest report lines; persisted locked effects |
| Exposure (Phase 15) | a tent blocks rest-time cold |
| `internal/wounds` | treat one lasting wound |
| World data | six items; provisioner stock; tutorial kit gains a bedroll and a fire steel |
| `modules/gmcp/gmcp.CompanyCamp.go` | `tent: bool` and `embers: bool`, so the map can pick its sprite and fire state |
| Sprite specification | `camp-rough` in S1 |

## Player help and tutorial

- **New page:** `help camp gear`, with the aliases `bedroll`, `tent`,
  `cookpot`, `fire steel`, `camp bells`, `trip lines` and `surgeon's kit`.
  It covers each item, its weight, effect, wear and price, and how gear is
  counted.
- **Updated pages:**
  - `help camp`: the fuel rule (one bundle a rest, embers) and gear;
  - `help gathering`: firewood;
  - `help campwatch`: bells;
  - `help cooking`: the cookpot;
  - `help wounds`: the surgeon's kit.
- **Tutorial:** the Camp lesson's kit includes a bedroll and a fire steel,
  and its hint mentions `help camp gear`.

## Acceptance tests

1. **Fuel:**
   - a rest needs a lit fire;
   - lighting spends one bundle, or the deadfall in a firewood room;
   - after a rest completes, the fire is embers and the next rest needs
     `camp fire` again;
   - embers keep their warmth until the camp is broken.
2. **Bedrolls:** each covers one member in order. Members without one
   recover normally. The bonus is locked at rest start.
3. **Tent:**
   - applies shelter;
   - does not stack with a `shelter` room;
   - blocks rest-time cold exposure;
   - shows the tent in `look` and on the map (GMCP `tent`).
4. **Fire steel:** damp bundles light first time and give full warmth.
5. **Cookpot:** one extra portion of multi-ingredient recipes only.
6. **Bells:** a 20% flat chance to spot without a watch, +10 with one,
   capped at 90%. They wear out after 10 rests.
7. **Surgeon's kit:**
   - treats one lasting wound per completed rest when a healer with mana is
     present;
   - wears 5 uses;
   - does nothing without a healer.
8. Gear on a separated member, or in cargo without its horse, doesn't
   count.
9. The locked gear effects and the ember state **survive a save and load,
   and a copyover fixture, mid-rest**.
10. The world clock never advances.
11. `help camp gear` renders, and `TestTutorialHelpPointersExist` passes.
    The tutorial Camp lesson still completes.

## Open questions

None. Resolved: the six items and their numbers were approved as
balance defaults; bedrolls are a bonus only. Camp theft is deferred as a future
feature.
