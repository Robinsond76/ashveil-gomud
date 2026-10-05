# Loot System: Tiers, Quality, Rarity, Item Level, and Goods

Status: owner-requested design, 2026-10-05. Documentation only; no gameplay
implemented. The owner wants loot to be a major part of Ashveil. Hauling large
amounts with horses is part of that. The owner asked for tiers, quality,
rarity, item level and more, using loot-heavy RPGs as a guide, and moved the
gear catalog up the roadmap. All numbers are proposed defaults to verify in
play and with the balance harness.

Builds on the approved [equipment families and tiers](2026-10-01-equipment-tiers-design.md)
and the shipped [33g equipment management](2026-10-01-phase-33g-company-equipment-loot-design.md)
(cargo, treasury and loot claims). It also builds on
[32f logistics](2026-09-28-phase-32f-company-logistics-design.md), with packs
and horses, and on the commodity markets from Phase 19.
It feeds [random room encounters](2026-10-01-random-room-encounters-design.md).

## Current state (master `e8a1032`)

- 22 weapons, 45 armor pieces, 21 consumables; nearly all are stock GoMud,
  with no tier, rarity or item level.
- Two loot tables (`loot/beast.yaml` with 3 entries and
  `loot/humanoid.yaml` with 2) used by 9 mobs. Other mobs have an
  `itemdropchance` (often 5%) to drop what they wear.
- An item instance can already carry its own `overrides` spec
  (`items.Item.Spec`), `Adjectives` and `Enchantments`. Item UUIDs persist
  (33g). This is the hook for generated items: a rolled item saves its
  generated spec on the instance, and old items stay as they are.
- Stat bonuses on gear use `statmods` (`internal/statmods`), already read by
  combat.

## Prior art

| Game | What Ashveil takes |
|---|---|
| Diablo II / III | Rarity colours, random affixes, uniques and sets, and boss loot tables |
| Path of Exile | Item level decides which affix tiers can roll; affix tiers are visible |
| Grim Dawn / Last Epoch | Readable affix pools by slot, and loot filters |
| Mount & Blade | Quality modifiers on any item (Rusty, Fine, Masterwork), and trade goods with regional prices hauled by pack animals |
| World of Warcraft | Item level as one readable power number, and vendor goods called "grey" items |
| Borderlands | Weapon family plus manufacturer flavour, with the name telling you what rolled |

Ashveil keeps the earlier rule that **family is function, tier is the power
budget, and rarity is availability and special properties**. A rare iron sword
is not automatically better than a fine steel one.

## Item model

Every equippable item has five layers. All of them are shown on `look`,
`inspect` and `company compare`.

| Layer | What it is | Values |
|---|---|---|
| **Base type** | Family and form (arming sword, war glaive, cured leather jerkin) | From the approved catalog |
| **Tier** | Material and power budget | 1 Common, 2 Steel, 3 Tempered, 4 Masterwork, 5 Runeforged, 6 Relic |
| **Item level (ilvl)** | Where it came from; caps its affixes | 1–70, from the source's level |
| **Quality** | Workmanship; scales base stats | Crude, Worn, Standard, Fine, Superior, Exquisite |
| **Rarity** | Number and kind of special properties | Common, Uncommon, Rare, Epic, Legendary (unique), Set |

### Tier and item level

- **The tier comes from the base type.** A drop's base type is picked from the
  tiers its item level allows: the tier band plus or minus one, weighted to the
  band. The approved bands still apply: tier 1 at levels 1–9, tier 2 at 10–19,
  and so on.
- **Item level is the source's level.** That is the mob, the boss, the contract
  level, or the chest's zone level. It is held to the zone's configured ilvl
  range so a farmed zone can't produce runaway items.
- **Item level sets which affix tiers can roll** (Path of Exile style).
- **A level requirement to wear** an item is `ilvl − 5` for Rare and above,
  `ilvl − 10` for Uncommon, and none for Common (owner decision, 2026-10-05).
  This keeps rare drops from being handed down to new characters. It changes
  the equipment-tiers design, which had no equip gates. Company equip and
  compare (33g) refuse and explain an item the member is too low to wear.

### Quality

Quality multiplies the base type's damage dice bonus and its protection, and
changes the item's value.

| Quality | Weight | Base stat | Value | Name prefix |
|---|---|---|---|---|
| Crude | 10% | −20% | ×0.4 | "crude" |
| Worn | 20% | −10% | ×0.7 | "worn" |
| Standard | 45% | ±0 | ×1 | none |
| Fine | 17% | +10% | ×1.6 | "fine" |
| Superior | 7% | +20% | ×2.5 | "superior" |
| Exquisite | 1% | +30% | ×4 | "exquisite" |

Quality is independent of rarity, so an exquisite common sword is possible
(the Mount & Blade feel). Field Smith (33f3) can later raise quality one step
at a smith for gold and materials (an economy sink).

### Rarity

| Rarity | Colour | Affixes | Drop weight (ordinary foe) | Notes |
|---|---|---|---|---|
| Common | white | 0 | 70% | Most drops |
| Uncommon | green | 1 | 22% | One prefix or suffix |
| Rare | blue | 2–3 | 6.5% | Generated name ("Gloomfang") |
| Epic | purple | 3–4, one of them major | 1.4% | A major affix is a mechanic, not a number |
| Legendary | orange | Fixed unique effect + 2 rolled | 0.1% (bosses more) | Hand-authored named items |
| Set | teal | Fixed; set bonus at 2/3/4 worn | Bosses and contracts only | Small sets of 3–4 |

Bosses, elites and contract rewards use better weights (see **Drop
generation**). The rarity colours use the existing ANSI alias system so text
clients and the web client match.

### Affixes

Affixes are data in `_datafiles/world/default/loot/affixes/*.yaml`.

- **Fields.** Each affix has an ID, prefix or suffix, eligible slots and
  families, tiers by minimum item level, value ranges, and a weight. It also
  has a group key: one affix per group per item, so no item gets "+Strength"
  twice.
- **Mechanics.** Affixes map onto mechanics that already exist. New mechanics
  only arrive as Epic major affixes, each with its own specification.

| Group | Examples (tier 1 → tier 5 ranges) | Uses |
|---|---|---|
| Stats | +Strength, Speed, Smarts, Vitality, Mysticism, Perception (1–2 → 7–9) | `statmods` |
| Vitals | +max HP (5 → 40), +max mana (5 → 50), mana per round (1 → 3) | `statmods`, 2d sustain |
| Offense | +flat damage (1 → 6), +hit % (2 → 8), +crit % (1 → 5) | Combat formulas |
| Defense | +protection, +block/parry/dodge % (1 → 5) | 30g2 defense |
| Status on crit | Bleed, stagger, burning, armor-break chance on crit | 30a crit statuses |
| Resist | Resist a status (knocked down, stunned, bleeding, asleep) | 30a statuses, Witch |
| Spell | +spell power %, +healing %, chant-break resistance %, +1 hex target (Epic only) | Class power design |
| Logistics | −weight % (burden), +carry capacity (packs), +warmth | 30g3, 32f, 15 |
| Expedition | +Pathfinder / Keen Eye / Forage rank 1 (Epic only) | 33f specialists |

Names come from the affixes: *"Fine Steel War Glaive of the Wolf"* (quality,
tier material, base type, suffix). A Rare gets a generated two-part name with
its base type underneath.

**Epic major affixes** (examples; each needs a full specification):

- *Cleaving*: an extra 50%-damage blow to the next foe in the row on a kill.
- *Vampiric*: 10% of damage dealt returns as HP.
- *Warded*: the first status each fight is ignored.
- *Echoing*: 15% chance a spell repeats at half power.

### Legendaries and sets

- Legendaries are authored: a name, lore, a fixed signature effect within the
  approved tier budget, and 2 rolled affixes. The equipment design's tier 6
  relics (for example **Ashen Reaper**) become legendaries.
- Sets are small: 3–4 pieces. They give a bonus at 2 pieces worn, and a
  stronger one at 3 or 4. Set pieces are boss and contract rewards per region,
  so they give a reason to revisit content.
- Every legendary and set piece belongs to a drop source (a boss, contract or
  region), so a player can hunt for it. `inspect` names its source, and a
  later collection log tracks what the player has found.

### Identification

Rare and above drop **unidentified** (owner decision, 2026-10-05). They show
their base type, tier, quality and rarity, but not their affixes. Unidentified
items sell for their base value only.

The main way to identify them is a **Scribe** in the company (owner direction).
Scribe is a new company specialist capability in the 33f pattern:

- **Who can be a Scribe:** the Wizard, Witch and Cleric lineages and their
  promoted classes. It uses the best eligible living member present, with the
  leader and then member-ID tie-breaks, and names that member.
- **Ranks:** companions gain ranks at levels 1/10/20/30. Players take theirs
  from their Cast rank, like the other caster specialists.
- **What each rank identifies:** rank 1 Rare, rank 2 Epic, rank 3 Legendary
  and Set. Rank 4 also shows each affix's tier and range, and where the item
  comes from.
- **When it happens:** automatically, as an item enters the company's cargo
  or a member's hands (loot, pickup, contract reward). There's no command,
  and a company with a Scribe never sees an unidentified item it can read.
  `autoskill` can turn this off.
- **Not the retired skill:** GoMud's old `scribe` skill was retired in 33f1.
  The new capability uses a distinct ID (proposed `lore`), so old saved
  `scribe` ranks or refunds can't revive it.

Without a Scribe of the needed rank, items are identified:

- by a sage or merchant in a settlement, for a fee (an economy sink);
- by equipping the item through a camp rest;
- by an identification scroll (a rare drop, sold by sages).

## Sellable goods and hauling

Goods make the company's carrying capacity and horses matter. Every enemy can
drop goods, and goods have a value-to-weight ratio, so a full pack is a
choice.

| Category | Examples | Value density | Where sold |
|---|---|---|---|
| Trophies | wolf pelts, bear hides, antlers, fangs, troll tusks | Low–medium, heavy | Any market; best at tanners and trappers |
| Salvage | rusted mail, broken blades, scrap iron, bent arrows | Low, heavy | Smiths (or salvaged into materials) |
| Materials | iron ore, steel ingots, runestone shards, ashwood, silk | Medium | Smiths, crafters; the Phase 19 commodity markets |
| Valuables | coin purses, silver trinkets, gemstones, idols, art objects | High, light | Jewellers, black markets (21b standing) |
| Provisions | game meat, herbs, mushrooms | Low; spoils | Cooks, inns; or eaten (survival) |
| Draughts | minor, lesser and greater mana draughts; healing potions | Very high, light | Rare drops; alchemists sell them dear (level impact design, 2d) |
| Curios | sealed letters, maps, relic fragments | Varies | Contracts, sages; can start quests |

- **Goods tie into the Phase 19 commodity markets.** Pelts, ore, ingots and
  timber map to commodities with regional prices, so carrying a horse-load from
  the wilds to a better market is profitable. Haggle (33f2) and settlement
  standing (21b) adjust prices.
- **Demand saturation.** Selling many of one good in one market lowers its
  price, which recovers with world time. Markets already persist news
  snapshots; saturation joins that state.
- **Weight and value bands** are set per category so a riding horse plus a
  pack horse can carry about one zone's worth of trophies and salvage.
  Valuables stay light and lucrative. The target for a loot run (proposed): a
  full pack horse of tier-1 goods is worth about one tier-1 Fine weapon.
- **Salvage** at a smith turns unwanted gear into materials by tier, with a
  bonus for quality and rarity. Materials feed quality upgrades, sockets
  (later) and crafting (camp consumables design).

## Drop generation

Each kill makes one loot roll per enemy. Each encounter or boss also makes one
group roll, with these outcomes:

| Source | Gold | Goods rolls | Equipment chance | Rarity boost | Notes |
|---|---|---|---|---|---|
| Ordinary foe | per template | 1 (60% chance) | 12% | ×1 | Below-level foes from the encounter contract |
| Elite foe | ×2 | 2 | 30% | ×2 Rare+ | Existing runtime elite roll (`Mob.IsElite`) |
| Encounter group cache | — | 1–2 | 20% | ×1 | One per won random encounter; "the raiders' packs" |
| Boss | ×5 | 3 | 100%, 2 items | ×5 Rare+, guaranteed Rare+ | Legendary and set chance from its own table |
| Contract reward | fixed | — | choice of 1 of 3 | fixed rarity | Shown before accepting |
| Chests in dungeons | zone | 2 | 40% | ×2 | Keys, traps (Skulduggery) |

- **Tables are by category and zone.** The existing `loot/*.yaml` categories
  grow (beast, humanoid, undead, construct, spirit). Each zone config names its
  ilvl range, the goods table it adds, and its boss tables. A mob's table can
  override any of these.
- **Smart loot, partly.** Half of equipment drops are biased toward the
  families the company can use: the leader's and companions' classes and
  current weapons. The other half is unbiased, so there is always something to
  sell or give to a future recruit.
- **Bad-luck protection.** Each boss keeps a per-leader counter. A guaranteed
  legendary or set piece from that boss's table drops by the Nth kill
  (proposed N = 20). The counter persists with the company.
- **Gold.** Templates keep their gold; random encounter groups get gold by
  level band. Gold drops go into the 33g treasury.
- **Rolls happen on death, server-side, exactly once.** Rolled items join the
  corpse under the existing 33g claim rules (private for 2 game hours, then
  public). Allied companies each get **their own roll** for a shared kill
  (personal loot, as in Diablo III and Guild Wars 2) instead of splitting one
  roll (owner decision, 2026-10-05). It changes 33d's fixed shared claims,
  and each company keeps its own 33g claim on its own roll.

## Loot handling for players

- **`loot` rules.** `autoloot` (33g) gets filters: by rarity (`autoloot rare`),
  by goods value per kg (`autoloot goods 5`), and by type (`autoloot
  valuables`). Filters are saved per leader.
- **Value per kg** is shown for goods so players can decide what to leave
  behind when a pack is full.
- **`company compare`** shows the rolled affixes and their difference from what
  the member wears, using the same numbers combat uses.
- **Junk selling.** `sell junk` sells all common goods marked junk and all
  items the player marked (`mark [item] junk`) in one transaction at a market.
- **Rarity colours and a loot line in the battle summary:** "Spoils: 2 wolf
  pelts, a fine iron mace, **Gloomfang** (rare short sword)."

## State, persistence and integration points

- **Generated items** save their rolled result on the instance:
  - tier, ilvl, quality, rarity, identified, affix IDs and values, and the
    name parts;
  - the resulting combat numbers as the instance `Spec` overrides;
  - a generator version, so a later affix rebalance can migrate items
    explicitly instead of silently.
  Old items with no rolled data are "legacy" common items of their template's
  tier.
- Item templates gain optional fields: `tier`, `family`, `basevalue`,
  `goodscategory`, and an `affixslots` override. Defaults keep old content
  valid.
- **New pure package `internal/loot`:** affix data, rarity and quality tables,
  and the generator, driven by an injectable random source for deterministic
  tests. The existing loot tables (Phase 18a) move there or are wrapped.
- **Death and rewards:** the existing mob death and loot path, the 33g claims,
  and the 33d alliance rules. The roll is recorded on the death event so a
  copyover can't reroll it.
- **Markets:** `modules/market` for goods commodities and saturation; shops
  for equipment buy and sell; standing and Haggle modifiers.
- **Display:** `look`, `inspect`, `company equipment/compare`, the GMCP
  `Company.Inventory` and the web Gear editor (34c) show the layers and the
  rarity colour.
- **No change to world time,** and gear survives save, restart and copyover.
  Uncollected corpses still disappear on restart (accepted 33g limitation).

## Delivery (proposed slices)

1. **Item model and generator:** layers, quality, rarity, affix data,
   identification, display, and the persistence of rolled instances. Rolls only
   in tests and through an admin `spawn loot` command.
2. **Tier 1–3 catalog:** the approved equipment design's first catalog (sword,
   axe, mace, spear, glaive, bow, staff, all armor paths, and shields). Also
   goods by category, and an audit giving existing items a tier and family.
3. **Drop tables:** zone and category tables, elite and boss rolls, smart loot,
   bad-luck protection, gold, the battle-summary spoils line and autoloot
   filters. Ships with or just after random encounters.
4. **Economy:** goods in markets, saturation, salvage, `sell junk`, and
   identification fees.
5. **Tier 4–6, legendaries and sets,** placed with bosses and contracts as
   content arrives.

## Owner decisions

Settled 2026-10-05:
1. Level requirements: Rare and above at ilvl − 5, Uncommon at ilvl − 10.
2. Personal loot: each allied company gets its own roll.
3. Rare and above drop unidentified; a caster Scribe identifies them.

Still open (proposed defaults stand until the owner decides):
4. Bad-luck protection counter N = 20 per boss per leader.
5. Smart loot bias of 50% toward the company's families.
6. Demand saturation in markets.

## Player help and acceptance

Ship indexed help with its slice, with keywords and aliases, linked from
`help equipment`:

- `help loot` (update);
- `help rarity`, `help quality`, `help itemlevel`, `help affixes`;
- `help goods`, `help salvage`, `help identify`, `help scribe`,
  `help autoloot`;
- `help equipmenttiers`.

Add a Departure tutorial hint about loot rarity and goods, and test the help
render and tutorial pointer.

Acceptance:

- **Generation:** deterministic generator tests for each layer, affix group
  exclusivity, ilvl tier gates and value ranges.
- **Real kills:** real-kill integration tests for ordinary, elite, boss and
  allied kills, with one roll each, claims intact and no reroll after copyover.
- **Persistence:** rolled instances survive save/load/copyover with identical
  stats, and legacy items load unchanged.
- **Gear in play:** compare and equip use the rolled numbers in real combat,
  and level requirements refuse through real equip routes.
- **Identification:** Scribe rank boundaries identify on real pickup and loot
  with named attribution, and the retired `scribe` ID never revives. Fee,
  scroll and camp-rest paths and autoloot filters work through the real
  command,
  and selling, saturation, salvage and Haggle interact correctly.
- **Balance:** harness cells with a company in tier-appropriate Standard,
  Fine and Rare gear stay within the encounter contract's targets from the
  [level impact design](2026-10-05-level-impact-class-power-design.md).

An independent full-diff review runs before each merge.
