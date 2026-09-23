# Phase 18: Loot Tables (18a) and Cooking (18b)

## Prior-art check

- **Loot today** (`internal/mobs.Mob.ItemDropChance`, `internal/mobcommands/suicide.go:329-390`):
  a single flat `itemdropchance` percent per mob spec. On death, every
  *carried* item always drops (no roll); every *worn* item (not
  remove-locked) rolls once against that one percent and drops or doesn't.
  Plus `Character.Gold`. There is no weighted table, no rarity, and no
  concept of a mob "category" driving loot — `Mob.Groups`/`Hates` are
  free-text AI/faction tags (`internal/mobs/mobs.go:54-55`), not loot
  categories.
- **The weighted-table pattern to reuse** (Phase 12b,
  `docs/superpowers/specs/2026-09-22-phase-12b-weighted-encounters-design.md`,
  `internal/expedition/expedition.go:81-157`): a pure `Resolve(roll uint64)`
  that sums entry weights, reduces the caller-supplied roll modulo the
  total, and walks a cumulative band to pick an entry. The RNG lives at the
  engine edge (a module seam), never inside the pure resolver. This is the
  exact shape Phase 18a mirrors for loot.
- **Items** (`internal/items/itemspec.go:196-224`): `ItemSpec` has `Value`
  (gold) and `Weight` (grams, Phase 9) but no generic tag field — each new
  concept (`Warmth`, Phase 15) has shipped as its own dedicated field or, for
  a genuinely new kind of item, a new `ItemType` enum value
  (`itemspec.go:97-125`: `Weapon`/`Food`/`Botanical`/`Junk`/... already
  exist; nothing for animal trade goods like fur or horn).
- **Cargo** (`internal/encumbrance/encumbrance.go:22-30`, Phase 9): `Cargo`
  stacks arbitrary `ItemId`s by count; any item with an `ItemSpec.Weight`
  already works as cargo with no new flag needed.
- **Professions** (`internal/skills/profession.go`): a `Profession` is a
  YAML-datafile-defined `{ProfessionId, Name, Skills []string}` grouping
  skill ids; a character's mastery drives its title. Cooking is not yet a
  profession or skill; Phase 17's status entry deferred it here explicitly:
  "Cooking is deferred to Phase 18b. Room containers already support
  crafting `recipes` to build on" (`docs/PROJECT_STATUS.md`, Phase 17
  entry). `internal/rooms/container.go` has an existing recipe/crafting
  mechanism for room containers (fireplaces, forges, etc.) that 18b reuses
  rather than inventing a second crafting system.
- **Roadmap** (`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`,
  Phase 18 row): "Loot tables: weighted, data-driven, by mob category
  (humanoid/bandit/beast); beast parts as goods. Reuses the 12b
  weighted-table pattern."

## Decisions confirmed (user, 2026-09-23, via AskUserQuestion)

1. **Loot tables are shared by mob category**, not authored per-mob. A new
   `LootCategory` tag goes on the mob spec; each category names a shared
   weighted loot table other mobs of the same category reuse.
2. **Layered, not replacing.** The new weighted table is a second, optional
   roll alongside today's `ItemDropChance` worn-item roll. Existing mobs
   with no category configured behave exactly as before — zero behavior
   change unless a mob spec opts in.
3. **18b (cooking) ships minimal**: data-driven recipes over the existing
   room-container crafting mechanism, gated by a new `cooking` skill/
   profession, producing a cooked `Food` item from raw ingredients. No new
   quality tiers or new buff types beyond what `Food`/`Nutrition` already do.

## Scope

**In scope (18a — loot tables):**
- A new pure package, `internal/loot`, holding the weighted-table type and
  its resolver (mirrors `internal/expedition`'s `WeightedInterruptionKind`/
  `ResolveKind` shape).
- A new `Mob.LootCategory string` field (mob spec).
- A new `Commodity` `ItemType` for beast trade goods (fur, horn, hide,
  tallow — things that are sellable/tradeable stock, not equipment, food,
  or an existing `Botanical`/`Junk`/`Object` fit). Plant/herb loot uses the
  existing `Botanical` type; ordinary gear/food loot uses existing types.
- Data-driven category tables loaded from `_datafiles/world/<world>/loot/`
  (one file per category, e.g. `humanoid.yaml`, `beast.yaml`, `bandit.yaml`),
  each a `[]WeightedLootEntry{ItemID int, Weight uint, MinCount, MaxCount int}`.
- Wiring: `internal/mobcommands/suicide.go`'s death-drop block gains one more
  roll, after the existing worn-item loop, that (if `mob.LootCategory != ""`
  and a table for it is loaded) resolves one table entry via a module-owned
  RNG seam and adds `MinCount..MaxCount` of that item to the same
  `corpseItems`/floor-drop branch the existing code already uses (so corpse
  vs. floor-drop config, and the existing `MobItemDrop` event, are reused
  unchanged).
- Content: tag a handful of existing shipped mobs with a category
  (at least one humanoid and one beast, e.g. whatever bandit/wolf-shaped
  mob already exists in the default world) and author their category
  tables with a small number of entries, enough to prove the mechanism
  end-to-end — not a full loot pass over every mob.

**In scope (18b — cooking):**
- A `cooking` skill (`internal/skills`) and a `cooking` profession entry.
- A small set of data-driven recipes (raw ingredient item(s) → cooked
  `Food` item) consumed through the existing room-container crafting
  mechanism in `internal/rooms/container.go` — reusing whatever recipe
  schema that mechanism already defines, not inventing a parallel one.
  If that mechanism's recipe schema can't express "requires the `cooking`
  skill at level N," add the smallest possible gate rather than widen the
  container system.
- A `cook` command (or reuse of whatever room-container crafting verb
  already exists, if the mechanism is already command-driven) usable at a
  cooking-tagged container (hearth/campfire/kitchen).
- Content: a couple of recipes using new `Commodity`/`Botanical` loot
  ingredients from 18a (meat/fur/herb → a cooked food item), so the two
  slices connect exactly as the roadmap intends ("beast parts as goods"
  feeding cooking).

**Explicitly deferred:**
- Rarity tiers, level-scaled loot, unique/named drops.
- Any loot-table content beyond a proving slice — most mobs stay
  uncategorized and keep today's exact behavior, same deferral pattern as
  every prior phase that proved a mechanism before authoring full content
  (12a/12b/16's route content, etc.).
- Quality-scaled cooking, new well-fed-style buffs distinct from ordinary
  `Food`/`Nutrition` (declined in the confirmed decisions above).
- Any market/sell-value tuning for `Commodity` items — Phase 19 owns
  pricing; 18a only needs a `Value` on each new item for existing
  vendor-sell flows to already work.

## Durable model

```go
// internal/loot/loot.go (new package, pure — no engine imports)

type WeightedLootEntry struct {
    ItemID   int  `yaml:"itemid"`
    Weight   uint `yaml:"weight"`
    MinCount int  `yaml:"mincount,omitempty"` // default 1 if zero
    MaxCount int  `yaml:"maxcount,omitempty"` // default MinCount if zero
}

type Table struct {
    Category string              `yaml:"category"` // matches Mob.LootCategory
    Entries  []WeightedLootEntry `yaml:"entries"`
}

func (t Table) Validate() error
// Resolve picks one entry proportional to Weight, exactly mirroring
// expedition.InterruptionProfile.ResolveKind's modulo-cumulative algorithm.
// Returns (entry, ok) — ok is false for an empty/invalid table so the
// caller can no-op cleanly (fails open, never panics).
func (t Table) Resolve(roll uint64) (WeightedLootEntry, bool)

// RollCount resolves how many of ItemID drop for a rolled entry, given a
// second caller-supplied roll (kept pure/injectable, same reason
// expedition never calls rand internally).
func (e WeightedLootEntry) RollCount(countRoll uint64) int
```

Loading: `internal/loot` also owns `LoadLootDataFiles()` /
`GetTable(category string) (Table, bool)`, following exactly the
`fileloader.LoadAllFlatFiles` + package-level map pattern
`internal/skills/profession.go` already uses (`allProfessions` →
`allLootTables`), keyed by `Category`. Cross-checking that every
`ItemID` referenced actually exists is a boot-time warning, not a load
failure — same policy `LoadProfessionDataFiles` already uses for unknown
skill refs (`profession.go:86-93`).

`Mob` gains:

```go
LootCategory string `yaml:"lootcategory,omitempty"` // internal/mobs/mobs.go
```

Empty string (the zero value, matching every existing mob spec on disk)
means "no category table roll" — pure additive change, no migration.

## Module / integration

No new module/plugin — `internal/loot` is a pure package like
`internal/expedition`'s domain layer, loaded at boot the same way
professions/skills are (`internal/skills` already has a
`LoadProfessionDataFiles()` boot call to mirror).

The RNG seam lives where `mobcommands.suicide.go`'s death handling already
runs (engine code, not pure) — it already calls `util.Rand(100)` directly
for the `ItemDropChance` roll (`suicide.go:359`), so the new category roll
reuses `util.Rand`/a `uint64` variant at the same call site, no new module
seam needed (unlike Phase 12b, which needed a seam because
`internal/expedition`'s resolution lived inside a module with its own test
harness; `suicide.go`'s existing precedent is to call `util.Rand` directly
and log the roll via `util.LogRoll`, e.g. `suicide.go:361`).

Wiring point (`internal/mobcommands/suicide.go`, right after the existing
worn-item loop, ~line 380, still inside the `!perma-gear` branch so it
respects the same corpse/no-corpse config):

```go
if mob.LootCategory != "" {
    if table, ok := loot.GetTable(mob.LootCategory); ok {
        if entry, ok := table.Resolve(util.Rand64()); ok {
            count := entry.RollCount(util.Rand64())
            for i := 0; i < count; i++ {
                item := items.New(entry.ItemID)
                // same corpseItems-vs-floor-drop branch as the worn-item
                // loop above, same MobItemDrop event
            }
        }
    }
}
```

(`util.Rand64()` may need adding alongside the existing `util.Rand(max
int)` — check `internal/util` first; if a `uint64` helper already exists,
reuse it instead.)

## 18b module / integration

`internal/skills` gains a `cooking` skill (data file,
`_datafiles/world/<world>/skills/cooking.yaml`) and profession entry
(`_datafiles/world/<world>/professions/cook.yaml`), following the existing
skill/profession datafile shape exactly (`internal/skills/AGENTS.md`:
skills keyed by lowercase id, filename is `SkillId + ".yaml"`; professions
use `ConvertForFilename`).

Recipes hang off whatever `internal/rooms/container.go`'s existing crafting
mechanism already defines — **read that mechanism's actual schema before
writing the plan's task list**; this design doc does not assume its shape
sight-unseen. If it already supports "produces item X from inputs Y, Z"
with no skill gate, the smallest addition is a skill-level check at the
crafting command's entry point, not a new field threaded through the
container system.

## Constraints and deferrals

- Never advances GoMud's global clock/round count — no timers here; loot
  rolls happen synchronously at mob death, cooking recipes resolve
  synchronously at the crafting command, same as every other engine action.
- Must survive restart/copyover: loot tables are boot-loaded static config
  (same as professions/skills), not persisted state — nothing to restore.
  A dropped item is persisted exactly like any other room/corpse item
  already is (unchanged mechanism). Recipe results are ordinary items;
  nothing new to persist for 18b either.
- Data-driven balance: new tables and recipes live in
  `_datafiles/world/<world>/loot/` and the existing skill/profession/
  container-recipe datafile locations — no hardcoded balance numbers in Go.
- Fails open: an uncategorized mob, an unloaded/missing category table, or
  a referenced `ItemID` that doesn't exist should all no-op the extra roll
  rather than error or drop a broken item — worn-item drops and gold are
  completely unaffected either way.

## Acceptance criteria

- `internal/loot.Table.Resolve` proportionally selects among entries for a
  fixed roll (deterministic, edge-of-band tested) and returns `ok=false`
  for an empty table.
- `internal/loot.Table.Validate` rejects a zero-weight entry, an entry with
  `MaxCount < MinCount`, and an empty entry list.
- A mob with `LootCategory` set to a loaded category rolls an additional
  drop through the real `suicide.go` death path (wiring test through the
  actual command, not just the pure resolver), landing in the same
  corpse-vs-floor branch the existing worn-item drops use.
- A mob with no `LootCategory` (i.e. every existing shipped mob) drops
  exactly as before — a regression test pinning today's exact behavior
  unchanged.
- 18b: a `cooking`-gated recipe run through the real container-crafting
  command produces the expected `Food` item and consumes its ingredients;
  running it below the required skill level is refused with the mechanism's
  existing failure message shape (no new UX invented if the container
  system already has one).
- `go test -race ./...`, `make generate`, `make validate` all pass.
