# Phase 22a: Archetype Creation Step and Starter Kits

## Prior-art check

- **Roadmap:** the
  [company life and onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md)
  puts [recruitment and creation](2026-09-23-recruitment-character-creation-design.md)
  first. That spec has three parts with different owners and risks, so it
  is split like Phases 19/19b and 21a/21b:
  - **22a (this doc):** the archetype step during character creation, and
    one starter kit per archetype, granted exactly once (player flow items
    1–2).
  - **22b:** durable companion level, progression, and equipment on the
    company record (the "durable model" section).
  - **22c:** settlement recruiters, `company recruit`, priced and free
    tutorial candidates (player flow item 3). It needs 22b so a recruit's
    gear is durable.
- **Archetypes** (Phase 17, `modules/archetype`): a configured table of five
  archetypes. `archetype choose <name> confirm` records a one-time choice in
  the module's durable registry, then applies idempotent skill and spell
  grants. The grants are re-applied on `PlayerSpawn` for crash recovery.
  Permadeath clears the choice. `archetypereset` (admin) clears it too. The
  engine reads the module only through the `internal/archetypes` provider
  seam.
- **Character creation** is the upstream `start` command
  (`internal/usercommands/start.go`), run in the void (`RoomId == -1`). It
  is a `prompt.Prompt` flow: race, then name, then `CharacterCreated`,
  then "skip the tutorial?". Each answer re-enters `Start`, and completed
  steps are skipped because their result is already on the character, so a
  reconnect resumes at the first unanswered step.
- **Upstream newbie kit** (item 100) is a scripted consumable that grants a
  fixed item list when used. It isn't archetype-aware and nothing hands it
  out at creation. It stays as it is.
- **Items:** no staff exists. The sling (10014) is a two-handed `shooting`
  weapon, and the engine has no ammunition, so no ammo check is needed.
  Item values are auto-calculated at load when not set.

## Decisions

Applied under the owner's standing instruction to proceed with the
recommendation (session request "Begin next phase"). Each one is recorded
here.

1. **Kits are archetype config.** Each archetype gets a `Kit: [itemId, ...]`
   list in `modules/archetype`'s config overlay (repeat an id to give two).
   At load, an item id that doesn't resolve is dropped from that kit with a
   warning. The archetype still loads.
2. **Shipped kits.** Values are the engine's auto-calculated item values with
   buffs loaded, as on the server. The spread is within 9%:

   | Archetype | Kit | Value |
   |---|---|---|
   | Warrior | guardsman's broadsword, wooden shield, leather cap, cheese sandwich, waterskin | 266 |
   | Rogue | dagger, lockpick kit, leather pants, worn boots, rope, cheese sandwich, waterskin | 256 |
   | Wizard | ash quarterstaff (new item 10021), leather cap, cotton shirt, student's amulet, cheese sandwich, waterskin, 2× small blue potion | 257 |
   | Cleric | crude cudgel, wooden shield, cotton shirt, cheese sandwich, waterskin, 2× small red potion | 260 |
   | Ranger | sling, leather cap, fur cape, worn boots, hunter's stew, waterskin, rope | 246 |

   A shipped-data test pins every id and the balance (the largest kit
   total is at most 1.25× the smallest). Spell prerequisites are already
   covered: wizards and clerics get `cast` and their spells from the
   Phase 17 grants.
3. **Exactly once: an owed record plus a claim marker.**
   - The registry gains `Kits map[userID]archetypeID`. It is written in the
     same save as the choice, so a committed choice always records the kit
     it owes.
   - Only choices made from this phase on owe a kit. Existing characters
     with an archetype keep their choice and gear and get no kit. An
     unchosen existing character who chooses gets one.
   - The claim marker is the character's `MiscData["archetype-kit"]`, set
     to the archetype id. It is stored in the user record, the same file as
     the inventory. Items and marker are saved together, so neither can be
     saved without the other.
   - **Grant:** runs on the game loop right after a committed choice, and
     again on every `PlayerSpawn` as recovery. It gives the kit only when a
     kit is owed and the marker is absent. It stores the items in the
     backpack, sets the marker, and saves the user.
   - **If the save fails:** the grant stays in memory and goes out with the
     next autosave, logout, or copyover save.
   - **If the server crashes before any save:** items and marker are lost
     together. The next spawn sees a kit owed and no marker, and grants it
     once.
   - An admin `archetypereset` followed by a new choice owes a kit again,
     but the marker is still set, so no second kit is given.
   - Permadeath clears the owed record along with the choice. The engine
     replaces the character, so the new character has no marker and gets
     its own kit.
4. **Kit goes to the backpack; nothing is auto-equipped.** The message
   names every item and suggests `equip`. Auto-equipping would have to deal
   with race slot and hand rules, and gains little.
5. **Creation step.** A new optional `archetypes.Creator` interface is
   implemented by the module and exposed through package functions. `Start`
   uses only `internal/archetypes`, never a module. Flow:
   - After the name step and before `CharacterCreated`, `Start` asks "Which
     archetype will you follow?" with a numbered list: each archetype's
     description, skills, and kit.
   - The player answers by number or name, then confirms (`yes`/`no`).
     `no` asks again.
   - The step is skipped when there is no provider, when no archetypes are
     configured, or when the character already has one. Because of the last
     case, a reconnect after choosing resumes at the tutorial question, and
     a reconnect before choosing asks again.
   - If committing fails (for example, persistence is unavailable), the
     player sees the reason and a hint to use `archetype choose` later, and
     creation carries on. Creation is never blocked on the module.
6. **Surfaces:** `archetype` (list) and `archetype choose <name>` (preview)
   show each archetype's kit.

## Durable model

- `modules/archetype` registry: `Kits map[int]string` (`kits`,
  `omitempty`), decoded like `Players`: invalid user ids and blank values
  are dropped. Cloned and cleared with the rest of the character's data.
- Character `MiscData["archetype-kit"] = <archetype id>`, persisted by the
  engine's user save (autosave, logout, copyover).
- No world-clock access. Nothing here reads or advances rounds.

## Module and seams

- `internal/archetypes`:
  - `Archetype.Kit []int`. `Validate` drops non-positive ids.
  - `Choice{ID, Name, Description, Skills, Kit []string}` (kit item
    display names).
  - The `Creator` interface: `CreationChoices() []Choice` and
    `ChooseAtCreation(userID int, id string) (text string, ok bool)`.
  - Package functions `CreationChoices(userID)` (nil without a creator or
    when the user already has an archetype) and `ChooseAtCreation`.
- `modules/archetype`:
  - config parsing for `Kit`, and kit resolution in `buildTable` through an
    `itemName` seam;
  - `commit` (shared by `choose` and `ChooseAtCreation`) writes the choice
    and the owed kit in one save, with rollback;
  - `grantKit(user)`, called after a commit and from `onPlayerSpawn`, with a
    `saveUser` seam so tests never write the shipped data dir;
  - kit previews in `list` and `choose`.
- `internal/usercommands/start.go`: the archetype step.
- `_datafiles/world/default/items/weapons-10000/10021-ash_quarterstaff.yaml`.

## Constraints and deferrals

- Lock order: the grant reads the registry under `m.mu`, then releases it
  before touching the user or saving. The engine is never called while the
  lock is held, matching `applyGrants`.
- Tutorial content, recruiters, and durable companion gear are 22b, 22c,
  and later phases.
- Kit balance is a shipped-data test, not a runtime rule.

## Acceptance criteria

- Tests cover both config parsing and resolution: unknown items dropped,
  duplicates kept.
- `commit` writes the choice and the owed kit together. A failed save rolls
  both back and grants nothing.
- The grant gives the kit exactly once, covering each of these:
  - a repeated grant;
  - a repeated spawn;
  - reset and re-choose;
  - a choice made before this phase (no kit);
  - a lost unsaved grant recovered on spawn;
  - permadeath clears the owed record.
- The marker survives a YAML round trip of the user record, the same
  encoding used by user saves and copyover.
- Wiring tests:
  - `usercommands.Start` driven through its real prompt, where each
    shipped archetype's kit lands in the backpack. It covers reconnect
    (a fresh prompt) before and after the choice, a "no" at the
    confirmation, a failed commit that still reaches the tutorial
    question, and no provider (the upstream flow).
  - The `PlayerSpawn` event through `events.ProcessEvents` recovers an
    owed kit once.
- Every shipped kit id resolves, and kit totals are within 1.25×.
- `go test -race ./...`, `make generate`, `make validate`.
