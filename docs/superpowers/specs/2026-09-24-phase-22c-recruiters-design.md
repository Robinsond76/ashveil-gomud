# Phase 22c: Settlement Recruiters and `company recruit`

## Prior-art check

- **Spec:** player flow item 3 of
  [recruitment and creation](2026-09-23-recruitment-character-creation-design.md),
  split out as described in the
  [22a design](2026-09-24-phase-22a-creation-starter-kits-design.md). It
  builds on [22b](2026-09-24-phase-22b-durable-companion-state-design.md),
  so a recruit's gear is durable from the moment it joins.
- **`company summon`** (`modules/company`) is the only way to add a
  companion today. It checks capacity, then the Phase 21a alignment gate
  (`recruitRefusal`), reserves the survival identity, adds the record,
  seeds archetype (`CompanionArchetypes`) and disposition, spawns the mob,
  snapshots its template gear (22b), and saves. Every failure after the
  survival identity is spent rolls back through `rollbackSummon`. It only
  accepts `AllowedCompanionMobIDs` (the training dummy, 58), so it is a
  test and admin path, not a player one.
- **Settlements** are zones (Dunmar, Old Kings Road). Room features are
  already configured by room tag or room id in module config (markets by
  tag, inns by tag, black markets by tag).
- **Gold** is `Character.Gold`, deducted in memory with an
  `EquipmentChange` event, as `market buy` does. The user file is written
  by the engine's saves (autosave, logout, copyover); 22a also calls
  `users.SaveUser` right after a grant.
- **Farming vectors already closed by 22b:** dismissal destroys the mob
  with its gear, there is no way to take gear back from a companion, and a
  dead companion's gear is cleared from its record. A death still drops
  carried items and any worn item that passes `ItemDropChance`.
- **Content:** mob template ids stop at 60. Dunmar has no mobs of its own.
  The start room is still upstream's Frostfang (room 1); there is no
  Ashveil tutorial yet.

## Decisions

Applied under the owner's instruction "Continue with phase 22c", using
this design's recommendations, as in 22a and 22b. Each one is recorded
here.

1. **A recruiter is a configured room.** `modules/company` config gains
   `Recruiters`, each with a `RoomId`, a display `Name` (the hiring board
   or person), and its `Candidates`. A candidate has an `Id` (the word a
   player types), a `MobTemplateId`, a `Price` in gold (0 = free), and
   `Tutorial` (free, claimable once per character). A tutorial candidate
   always costs nothing, whatever `Price` says. Entries with no room, a
   blank or repeated id, a non-positive template, or a negative price are
   skipped with a warning. Templates are checked when used, not at load,
   so a missing template shows as unavailable rather than failing load.
2. **`company recruit`** with no argument lists the candidates here: name,
   archetype, level, alignment (1–100), equipment, and price, or "free,
   once" / "already claimed". `company recruit <candidate>` matches the
   candidate id, then the exact name, then a unique name substring.
   Outside a recruiter room it says no one is hiring here.
3. **Checks, in order, each a refusal that changes nothing:** persistence
   available; a recruiter here; a known candidate; its template exists;
   a tutorial candidate not already claimed; the company isn't full
   (`MaxCompanions`); the Phase 21a alignment gate; enough gold.
4. **Enlisting** reuses `company summon`'s path (extracted to `enlist`),
   with the candidate's template as the only allowed one. The candidate
   list is its own allow list; recruiter templates are deliberately not
   in `AllowedCompanionMobIDs`, so `company summon` can't bypass a price
   or a claim. A shipped-data test pins that.
5. **Tutorial claims are on the company record.** `Record.Claimed` lists
   the template ids of claimed tutorial candidates. It is written in the
   same save as the new companion, so the claim and the recruit are
   durable together or not at all. A failed save rolls both back. The
   claim outlives dismissal, desertion, and death: `Put` keeps a record
   that has claims even with no companions. Keying by template id means
   the same free candidate offered at two recruiters is still one claim.
6. **Payment is taken after the company save.** The gold check runs
   before anything changes, and the deduction happens on the same turn,
   under the world lock, after `enlist` has saved, so a failed save never
   costs gold. The user is then saved at once (`users.SaveUser`, a
   `saveUser` seam in tests) so the company and user files move together.
   If that user save fails, the deduction is still in memory and goes out
   with the next autosave, logout, or copyover; a crash inside that
   window leaves the player with the recruit and the gold. That is the
   player-favourable side, and it cannot repeat for a tutorial candidate
   (the claim is in the company save). Accepted and documented.
7. **No repeatable reward.** Candidate templates wear their gear (no
   carried items, no gold) and have `itemdropchance: 0`, so a death drops
   nothing. With 22b's dismissal (the mob leaves with its gear) and no
   way to take gear back, neither a free nor a paid recruit can be turned
   into items or gold. A shipped-data test pins the template shape.
8. **Settlement standing does not apply yet.** Prices are the configured
   ones; Phase 21b markups and refusals for recruiters are deferred. The
   alignment gate already keeps unlike companions out.
9. **Shipped content.**
   - Waymark Inn (2003, Dunmar), "the hiring slate by the hearth":
     - `tamsin`: Tamsin Reed (61), caravan guard, warrior, level 1,
       tutorial;
     - `oswin`: Brother Oswin (62), wandering cleric, level 1, tutorial;
     - `garrick`: Garrick Vane (63), sellsword, warrior, level 3,
       120 gold.
   - Trappers' Post (2005, Old Kings Road), "the notched stick":
     - `ysolde`: Ysolde (64), trapper-scout, ranger, level 2, 80 gold.
   - Room descriptions mention the hiring slate and the stick.
     `CompanionArchetypes` gains the four templates.

## Durable model

`companies` store, per leader (`omitempty`):

```yaml
claimed: [61, 62]
```

Decoded by the existing registry decoder (legacy records have none).
`Registry.Get` copies the slice. No clock access: nothing reads or
advances rounds.

## Module and seams

- `internal/company`: `Record.Claimed`, `Record.HasClaimed(templateID)`,
  `Registry.Claim(leader, templateID)`, deep copy in `Get`, and `Put`
  keeping a record with claims.
- `modules/company`:
  - `recruit.go`: config parsing (`parseRecruiters`), the candidate view,
    `recruit`, and the listing;
  - `enlist` (extracted from `summon`), taking the allowed template and an
    optional claim; `rollbackSummon` restores the pre-summon claims;
  - `wireRecord.Claimed`;
  - a `saveUser` seam (`users.SaveUser` by default);
  - `company recruit` in the command switch and usage text.
- Content: mobs 61–64 under `mobs/dunmar/` and `mobs/old_kings_road/`,
  the two room descriptions, and the company config overlay.
- The module stays on the game loop with no lock of its own.

## Constraints and deferrals

- Standing markups and refusals for recruiters (see decision 8).
- An authored Ashveil tutorial that walks a new player to the recruiter.
  The two free candidates are ready for it; the tutorial is later content.
- Recruiter NPCs with dialogue; the recruiter is a room fixture.
- Permadeath does not clear company records (true before this phase), so
  a replacement character on the same account keeps the old claims.

## Acceptance criteria

- **Domain:** `Claim` and `HasClaimed`; `Get` deep-copies claims; `Put`
  keeps a claims-only record; a YAML round trip keeps claims.
- **Module:**
  - config parsing: valid entries, each skipped malformed kind;
  - listing shows archetype, level, alignment, equipment, price, and the
    claimed state; outside a recruiter room, a clear refusal;
  - a free tutorial recruit joins, is claimed in the same save, and can't
    be claimed again after dismissal;
  - a priced recruit joins and costs exactly its price, saving the user;
  - refusals leave gold, record, and survival unchanged: not enough gold,
    full roster, unknown or unavailable candidate, already claimed,
    alignment gate;
  - a failed company save rolls back the companion and the claim and
    takes no gold;
  - `company summon` still refuses a recruiter template.
- **Wiring,** through `plugins.Load`, the real plugin store, a fixture
  world, and `usercommands.TryCommand`: list, recruit the free candidate
  and a priced one, place one in the formation, see both in
  `company status`, a real `plugins.Save()` and reload keeps the claim,
  and a second claim is refused. The clock never moves.
- **Shipped data:** every recruiter room and candidate template exists;
  at least two distinct tutorial candidates; no candidate template is in
  the summon allow list; templates have `itemdropchance: 0`, no carried
  items, no gold; each has a configured archetype.
- `go test -race ./...`, `make generate`, `make validate`.
