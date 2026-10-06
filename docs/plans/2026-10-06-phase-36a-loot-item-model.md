# Phase 36a — loot item model, generator and Scribe

Implements slice 1 of the owner-approved
[loot system design](../designs/2026-10-05-loot-system-design.md): the item
layers, the generator, identification and the Scribe skill. Branch
`claude/36a-loot-item-model-ewed7l`. Players do not see drops yet: rolled
gear only appears through the admin `spawn loot` command and tests, until
the drop tables (Phase 37) wire the generator to kills.

## What ships

- **The roll, on the instance.** `items.Item.Loot` (`items.Rolled`) saves
  tier, item level, quality, rarity, identified, level requirement, the
  generated name, a generator version and the affixes (each with its
  mechanic, value and tier range). The resulting numbers live in the
  instance's `Spec` override, rebuilt from the base spec and the roll
  (`Item.ApplyRoll`, `Item.Identify`, `items.RolledSpec`). Items with no roll
  are "legacy" common items and load unchanged. `UnEnchant` rebuilds the
  roll's spec instead of dropping it.
- **Quality** scales a weapon's damage bonus and an armor's protection by
  −20% to +30% (always at least ±1), and value ×0.4 to ×4.
- **Rarity** sets the affix count (Uncommon 1, Rare 2–3, Epic 3–4 with one
  major, Legendary 3 with one major, Set 2). Rare and above start
  unidentified: base and quality numbers only, base value, affixes hidden
  and not applied until read. Legendary and Set never roll by chance; they
  are asked for explicitly until the authored catalog arrives.
- **Level requirement** (`ilvl − 5` Rare and above, `ilvl − 10` Uncommon),
  enforced in `Character.Wear`, the single route behind `equip`, `gearup`,
  `company equip` and `company compare`, with the reason shown.
- **Generator** (`internal/loot`): injectable random source, affix data in
  `_datafiles/world/default/lootaffixes/` (kept outside `loot/`, whose files
  are the Phase 18a category tables), group exclusivity, item-level tier
  gates, slot-class eligibility. `spawn loot [item] [ilvl] [rarity]
  [quality]` rolls one into the room (admin).
- **Display:** rarity colour aliases, quality and affix words in the name,
  `(rare, unidentified)` tag, layer lines in `look`, `show` and the
  `GetLongDescriptionFor` path, rank-4 detail (affix tier and range, source).
- **Scribe:** the retired `scribe` skill returns as a ranks 1–4 caster
  skill. Wizards and clerics claim it (trainable at the Frostfang Magic
  Academy), companions learn it as an optional skill (`company train`),
  and a one-time reset refunds any old rank (`Character.ScribeReset`).
  - **Camp, automatic:** when a camp rest completes, worn gear reveals
    itself (no Scribe) and the best Scribe present reads every unread item
    its rank covers, in the leader's and companions' packs (the company's
    cargo is the leader's pack), free, switched by `autoskill scribe`.
  - **Field:** `scribe`, `scribe [item]`, `scribe [member] [item]`: 8, 15 or
    25 mana from the reader, refused in battle.
- **Help and tutorial:** `rarity`, `quality`, `itemlevel`, `affixes`,
  `identify`, `scribe`; updates to `equipment`, `loot`, `skills`,
  `company-train`, `company`, `autoskill`, `camp`; a Departure/loot-lesson
  hint.

## Decisions and deviations (each a call this phase made)

1. **The roll type lives in `items`, the generator in `loot`.** `loot`
   already imports `items` (Phase 18a tables), so the saved type cannot
   live there.
2. **Affixes use mechanics that exist today:** stats, max health and mana,
   flat damage, protection, parry, healing percent, warmth, weight. Hit,
   crit, block and dodge chances, status-on-crit, resists, spell power,
   chant-break resistance and the Epic mechanics (cleaving, vampiric,
   warded, echoing) arrive with the systems that read them, each with its
   own specification. Epic's "major" affix is a large numeric one for now.
3. **Packs, consumables and keys never roll.** Pack validation forbids stat
   mods and protection.
4. **The tier word is not added to names.** Catalog names (36b) carry their
   material, so "fine steel war glaive" is the catalog's "steel war glaive"
   plus quality. Tier is shown on `look`.
5. **Unread affixes are not applied.** The design says unidentified items
   show but hide affixes and sell for base value; applying hidden stats
   would make reading pointless, so reading applies them.
6. **Rolled gear stays out of legacy cargo stacks.** A legacy cargo stack
   holds an item id only and would erase a roll, so `cargo put` refuses
   rolled gear. Shared cargo (Phase 33g), where the leader's pack is the
   cargo, keeps exact instances and is unaffected.
7. **Old scribe ranks are refunded, not just cleared.** The design says
   "cleared once"; a refund of the points they cost is the kinder reading,
   and 33f1 already refunded anyone who logged in since.
8. **Scribe is not a specialist utility.** Specialist levels come from
   archetype companion levels; Scribe's rank is the trained skill, so it
   has its own resolver (best rank, then leader, then lowest companion ID).
9. **Deferred from the design's acceptance** (they belong to later
   slices): identification by a sage's fee or scroll, affix-aware
   `company compare` numbers beyond what the spec shows, drop tables and
   `autoloot` rarity filters, goods, salvage, `sell junk`, and
   `help goods`/`salvage`/`autoloot`/`equipmenttiers`.
10. **Also deferred:** merchants refuse rolled gear (`IsSpecial`), so it
    cannot be sold until the goods slice; GMCP item lists still show base
    names; Legendary's signature effect and Set bonuses (a Legendary rolls
    three affixes with one major, a Set two). Phase 38a should add
    `scribe` to the Witch.

## Tasks

- [x] `items.Rolled`, `Item.Loot`, quality and rarity tables, `RolledSpec`,
  `ApplyRoll`, `Identify`, naming, display, `WearRefusal`.
- [x] `internal/loot` affix data, loader, generator, Scribe rules; affix
  and name data files; `LoadAffixDataFiles` at startup; `spawn loot`.
- [x] `Character.Wear` level requirement; `cargo put` guard; company clone.
- [x] Scribe: skill file, archetype claims, optional skill, one-time reset,
  camp hook (`archetypes.CampIdentify`), `scribe` command, `autoskill`.
- [x] Help pages, keywords, hub links, tutorial hint, render test.
- [x] Tests: item model (quality, naming, identify, UnEnchant, save/load,
  copy aliasing), generator (counts, group exclusivity, tier gates, slot
  eligibility, distributions, bad data), real `equip`, `spawn loot`, `look`,
  `company equip`/`compare`, camp-rest completion, `scribe` command,
  training gates, the one-time reset, `cargo put`.
- [x] Independent full-diff review; fix findings with regressions.
- [x] Final checks: `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint`. Project Status entry, PR.
