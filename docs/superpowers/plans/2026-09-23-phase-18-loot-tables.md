# Phase 18: Loot Tables (18a) and Cooking (18b) — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-23-phase-18-loot-tables-design.md`).
Open decisions (shared-by-category tables, layered on `ItemDropChance`
not replacing it, minimal cooking scope) confirmed by the user via
`AskUserQuestion` on 2026-09-23 — all three recommended options.

Work this as two independently-mergeable slices, 18a then 18b, each with
its own worktree/branch, its own testing-and-review gate, and its own
`docs/PROJECT_STATUS.md` entry — matching how 17a/17b shipped as two
passes.

## Slice 18a: loot tables

- [ ] **Read `internal/util` first** for any existing `uint64` random
      helper before assuming `util.Rand64()` needs adding; confirm the
      exact roll-logging convention (`util.LogRoll`) used at
      `suicide.go:361` so the new roll logs consistently.
- [ ] **`internal/loot` (new pure package): `WeightedLootEntry`, `Table`,
      `Validate`, `Resolve`, `RollCount`.**
  - Tests first, `internal/loot/loot_test.go`:
    - `TestTableValidateRejectsZeroWeightEntry`
    - `TestTableValidateRejectsMaxCountBelowMinCount`
    - `TestTableValidateRejectsEmptyEntries`
    - `TestTableResolveSelectsProportionally` (weights 3/1, assert each
      roll boundary via modulo, mirroring
      `TestResolveKindWeightedTableSelectsProportionally` from Phase 12b)
    - `TestTableResolveEmptyTableReturnsNotOk`
    - `TestRollCountDefaultsMinCountOneMaxCountEqualsMin` (zero-value
      fields behave as documented)
    - `TestRollCountRespectsBounds` (many rolls stay within
      `[MinCount, MaxCount]`)
  - Then implement.
- [ ] **`internal/loot`: data-file loading.**
  - Mirror `internal/skills/profession.go`'s
    `LoadProfessionDataFiles`/`GetProfessionSpec`/`GetAllProfessions`
    shape exactly: `LoadLootDataFiles()`, `GetTable(category string)
    (Table, bool)`, package-level `allLootTables` map, boot-time warning
    (not failure) for a referenced `ItemID` that doesn't exist in
    `internal/items`.
  - Test: `LoadLootDataFiles` against a temp fixture directory loads
    tables keyed by `Category`; an unknown `ItemID` warns and still loads
    (use whatever test-log-capture pattern `profession.go`'s own tests,
    if any, or `internal/skills/skills_test.go` already use).
- [ ] **`internal/mobs`: `Mob.LootCategory string` field.**
  - Zero value (`""`) on every existing mob spec — no migration, no
    behavior change. Add the yaml tag, update any mob-spec doc comment
    block near `ItemDropChance`/`Groups` for consistency.
- [ ] **`internal/items`: `Commodity` `ItemType`.**
  - Add to the `ItemType` const block (`itemspec.go:97-125`) alongside
    `Botanical`/`Junk`/etc. Check `internal/items`' own type-list/display
    helpers (anything that enumerates `ItemType` for admin UI or
    `AllEquipSlots`-style completeness) for a spot that needs the new
    value added, so it doesn't silently break an existing exhaustive
    switch — grep for `case Botanical` / `Botanical:` first.
- [ ] **Wiring: `internal/mobcommands/suicide.go`'s death-drop block.**
  - Test first (real command path, not just the pure package): a mob
    instance with `LootCategory` set to a loaded test category and killed
    via the real death/suicide code path drops the resolved item into the
    same corpse-vs-floor branch the existing worn-item drops use (assert
    via `room.Corpse`/`room.Items` per whatever `Death.CorpseItems`
    config the existing suicide tests already exercise, if any exist —
    otherwise via the existing `MobItemDrop` event queue, matching
    `suicide.go:341-346`'s existing pattern).
  - Regression test: a mob with `LootCategory == ""` (i.e. every existing
    shipped mob) drops identically to today — pin this explicitly so a
    later refactor can't silently start rolling for uncategorized mobs.
  - Then implement: after the existing worn-item loop
    (`suicide.go` ~line 380, still inside the `!perma-gear` branch),
    resolve `loot.GetTable(mob.LootCategory)` → `Table.Resolve` →
    `RollCount`, append `count` copies of `items.New(entry.ItemID)` to
    the same `corpseItems`/floor-drop path, reusing the existing
    `MobItemDrop` event emission for each dropped item.
- [ ] **Content: category tag + tables for a proving slice.**
  - Pick at least one existing shipped humanoid-ish mob and one
    beast-ish mob (grep `_datafiles/world/*/mobs/` for a bandit/wolf-
    shaped candidate) and set their `lootcategory`.
  - Author `_datafiles/world/<world>/loot/humanoid.yaml` and `beast.yaml`
    (or whatever categories the chosen mobs actually need), a handful of
    entries each, using the new `Commodity` type for at least one beast
    entry (e.g. a wolf hide/fur) and an existing type for the humanoid
    table (e.g. a `Junk`/`Object` trinket).
- [ ] `gofmt -l`, `go vet ./internal/loot/... ./internal/mobs/...
      ./internal/items/... ./internal/mobcommands/...`, `go build ./...`,
      `go test -race ./...` after each task.
- [ ] `make generate`, `make validate`.
- [ ] **Testing and review gate (18a):** independent reviewer subagent
      (most capable model tier) over the full 18a diff, briefed with this
      design doc, the non-negotiable invariants (clock, restart/copyover,
      lock order), and asked for bugs/design gaps/missing coverage.
      Verify each finding, fix real ones with a regression test, record
      rejected ones and why.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next,
      phase table row 18a, new work-log entry with a **Review:** line.

## Slice 18b: cooking

- [ ] **Read `internal/rooms/container.go`'s actual recipe/crafting
      mechanism before writing any code** — confirm its schema (inputs,
      output, any existing skill-gate hook) and its command entry point.
      This plan's remaining tasks assume that mechanism is reusable as-is;
      if it's materially different from what the design doc expects,
      stop and reconcile the design doc first rather than improvising.
- [ ] **`internal/skills`: `cooking` skill + `cook`/`chef`-style
      profession datafile.**
  - Follow `internal/skills/AGENTS.md`'s exact datafile conventions
    (skill filename = `SkillId + ".yaml"`, profession filename via
    `ConvertForFilename`).
  - Test: `SkillExists("cooking")` after `LoadDataFiles` against the real
    shipped datafile (mirror however existing skill-load tests check a
    shipped skill).
- [ ] **Recipes: 2-3 data-driven recipes using Phase 18a loot ingredients
      producing a `Food` item.**
  - Wire the skill gate into the container-crafting command's entry point
    (smallest change the real mechanism allows, per the task above).
  - Test first, through the real crafting command (not a bypassed
    helper): a character with `cooking` at the required level, standing
    at a cooking-tagged container, with the right raw ingredients,
    produces the expected `Food` item and the ingredients are consumed.
  - Test: a character below the required skill level is refused with the
    mechanism's existing failure-message shape.
  - Then implement.
- [ ] **Content:** tag at least one existing hearth/campfire-capable room
      container as cooking-capable (or confirm campfires already qualify
      per Phase 15's campfire-as-fixture work), and ship the 2-3 recipes.
- [ ] `gofmt -l`, `go vet ./internal/skills/... ./internal/rooms/...`,
      `go build ./...`, `go test -race ./...` after each task.
- [ ] `make generate`, `make validate`.
- [ ] **Testing and review gate (18b):** independent reviewer subagent
      over the full 18b diff, same invariants briefing as 18a. Verify
      findings, fix real ones with regression tests, record rejections.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next,
      phase table row 18b (Phase 18 now fully complete), new work-log
      entry with a **Review:** line.
