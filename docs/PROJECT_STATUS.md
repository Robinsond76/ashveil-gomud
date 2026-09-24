# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-09-24
- **HEAD:** Phase 23a (rest tiers: camp Rested, inn Well Rested, exclusive and durable) is complete and reviewed, merged to `master`. Phase 23b (whetstones and Sharpened weapons, with the owner's 2026-09-24 amendment) is next.
- **Upstream baseline:** `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`

## Current position

- **Completed:** Phase 0–1 (fork, baseline, integration map), Phase 2 (company
  companion slice), Phase 3 (company roster + 3×3 formation), Phase 4
  (survival state), Phase 5 (terrain and travel profiles), Phase 6 (travel
  interruptions), Phase 7 (camping), Phase 8 (weather engine and
  descriptions; multiplier wiring deferred as documented Option A), Phase 9
  (encumbrance and cargo engine and commands; multiplier wiring into
  travel/rest deferred, same Option A shape), and Phase 10 (mounts, with a
  real wired cargo-capacity bonus; travel-speed wiring deferred, same
  Option A shape), Phase 11a (enemy parties: mobs sharing a `Groups` tag
  assemble into a party with an auto-assigned formation, shown to players
  as one grouped room listing), Phase 11b's domain layer
  (`internal/engagement.AssignTarget`: weakest/strongest/random target
  selection filtered by an injected legality predicate), and Phase 11c's
  domain layer (`internal/formationcombat`: column-occupancy reach,
  lateral range, front-row interception; a `Reach` trait on items/mobs; a
  read-only `formation reach <member>` query command), and the combat-loop
  wiring for all three player/mob attack directions plus 11b's
  reassignment-on-death (a `company.FormationProvider` seam, extended
  across four passes with `InstanceFor` and `LeaderAndKeyForInstance`;
  `internal/hooks/combat_formation.go` gates every direction through 11c's
  `Legal`/`InterceptFrontRow`, resolving the live enemy party fresh each
  round via 11a's `mobparty.Assemble`, and now also reassigns a company
  member's target via 11b's `engagement.AssignTarget` when it's lost),
  Phase 18a (category loot tables and a small shipped loot slice), and
  Phase 18b (skill-gated container recipes; Waymark Inn hearth cooking),
  and Phase 19 (zone stock ledgers with bounded round-driven price drift
  and a `market` command; Dunmar and Old Kings Road markets), and Phase
  19b (trading in tagged market rooms with separate buy and sell
  prices; Dunmar Market Square, Trappers' Post), and Phase 20 (trade
  rumours at inns from a stale, persisted market news snapshot), and
  Phase 21a (company alignment: companion alignment on a −100..100 display,
  drift toward the rest of the company, loyalty and desertion, and a
  recruit gate), and Phase 21b (settlement standing: market and inn
  markups or refusal by alignment gap, black markets for outlaws), and
  Phase 22a (an archetype step in `start` and one starter kit per
  archetype, granted exactly once), and Phase 22b (durable companion
  level, gear, and gold on the company record), and Phase 22c
  (settlement recruiters: free-once tutorial and paid candidates through
  `company recruit`), and Phase 23a (rest tiers: a camp rest grants
  Rested, an inn stay Well Rested, exclusive, and durable for companions).
- **Next:** Phase 23b, the whetstone half of the
  [rest and weapon preparation spec](superpowers/specs/2026-09-23-rest-weapon-preparation-design.md)
  (second on the [onboarding roadmap](superpowers/specs/2026-09-23-company-life-onboarding-roadmap.md)).
  The owner amended it on 2026-09-24: a whetstone is usable on demand at
  any time, has 10 uses, and spends one use per member sharpened.
  The Phase 19b inter-market profit question is resolved: the small
  standing trade-route profit stays (see the 19b spec). Phase 18 and 19 have design docs
  and implementation plans, confirmed with the user 2026-09-23 (all
  recommended options): 18a layers a new category-shared weighted loot
  table on top of the existing `ItemDropChance` roll (mirrors Phase 12b's
  weighted-table pattern); 18b is a minimal cooking slice over the
  existing room-container crafting mechanism; 19 is an engine-first,
  narrow zone-scoped market registry with bounded round-driven price
  drift and a read-only `market` command (mirrors `modules/weather`'s
  shape), with the buy/sell-integration question explicitly left open to
  the plan's first task. See
  [18 spec](superpowers/specs/2026-09-23-phase-18-loot-tables-design.md) /
  [18 plan](superpowers/plans/2026-09-23-phase-18-loot-tables.md) and
  [19 spec](superpowers/specs/2026-09-23-phase-19-commodities-markets-design.md) /
  [19 plan](superpowers/plans/2026-09-23-phase-19-commodities-markets.md).
  Phase 18, 19, and 19b are complete.
  Earlier notes: Phase 11's formation combat wiring is entirely done; Phase 11d
  (guard reactions, crit effects, wounds, AI personality) remains an
  unscheduled bucket. Phase 12's engine plumbing is now complete: 12a's
  encounter-kind abstraction, 12b's weighted tables, and 12c's combat
  encounters (see above) all ship. No shipped route uses any of it yet
  (`oak-road` stays plain `fallen-tree`) — deferred until there's a real
  reason to author new route content, same deferral every 12-series pass
  has made. Per explicit direction this session, combat is the only
  Phase 12 *encounter subsystem* being built; the rest of the overview
  doc's list — merchants, injured NPCs, route choices, camp
  opportunities, ruined sites, resources, social encounters — are kept
  as named future ideas (each needing its own subsystem: a shop flow,
  dialogue, branching choice UX, or `internal/camping` integration), not
  committed work.

## Phase progress

| Phase | Scope | Status |
|---|---|---|
| 0 | Fork and bootstrap GoMud baseline | Complete |
| 1 | GoMud integration map and handoff docs | Complete |
| 2 | Minimal company slice (one persistent companion) | Complete |
| 3 | Company roster (cap 5) + 3×3 formation state | Complete |
| 4 | Survival state (hunger/thirst/fatigue) | Complete |
| 5 | Terrain and travel profiles | Complete |
| 6 | Travel interruptions | Complete |
| 7 | Camping | Complete |
| 8 | Weather | Complete |
| 9 | Encumbrance and cargo | Complete |
| 10 | Mounts | Complete |
| 11a | Enemy parties | Complete |
| 11b | Unit-vs-unit engagement | Complete: target assignment, reassignment-on-death, and the proactive engagement trigger (pre-existing in `attack.go`, verified) all wired |
| 11c | Formation tactics | Complete: domain layer, schema, read-only query, all three player/mob attack directions wired, and the leader-as-interceptor gap closed |
| 11d | Guard reactions, crit effects, wounds, AI personality | Deferred, not scheduled |
| 12a | Encounter-kind abstraction | Complete: `InterruptionKind` widened to `fallen-tree`/`discovery`/`tracks`, data-driven text lookup |
| 12b | Weighted encounter tables | Complete: `InterruptionProfile.Kinds` weighted-roll form, resolved once at fire time; no shipped route uses it yet |
| 12c | Combat encounters | Complete: `Combat` interruption kind spawns its route's `CombatMobID` into the origin room and commands it to attack; resume/return gated on the mob still being alive and present |
| 13 | Sky and environment | Complete: moon phases, cloud cover, fog, indoor biomes/tags, indoor glimpse, richer `weather`; display-only |
| 14 | Visibility and light | Complete: ambient vs per-viewer light, personal/fixture/party light, darkness hit penalty, `light` command, `floatinglight` spell |
| 15 | Temperature, clothing, exposure | Complete: `internal/climate`, `modules/exposure`, item `warmth`, weather `TemperatureMod`, survival member drain + mutex, `temperature` command |
| 16 | Walking fatigue, inns, travel/rest multipliers | Complete: `internal/walking`, `modules/walking`, inn stays in `modules/camping`, multipliers locked at departure/rest start, Waymark Inn, Old Kings Road zone |
| 17 | Archetypes (17a) and utility skills (17b) | Complete: `internal/archetypes`, `modules/archetype`, training and spell gating, companion archetypes, `autoskill`, `trap`, auto-light. Cooking deferred to Phase 18b |
| 18a | Loot tables | Complete: category tables, boot/reload loading, corpse/floor drops, proving content |
| 18b | Cooking | Complete: `cooking` skill + `cook` profession, per-recipe skill requirements on room containers, deterministic recipe choice, Waymark Inn hearth with three meals |
| 19 | Commodities and markets | Complete: `internal/market`, `modules/market`, `market` command, Dunmar and Old Kings Road markets |
| 19b | Market trading | Complete: `market buy`/`market sell` in tagged market rooms, buy/sell spread, Dunmar Market Square, Trappers' Post; inter-market profit kept (resolved 2026-09-24) |
| 20 | Trade rumours | Complete: `rumors` at inns, fuzzy hints from a persisted market news snapshot refreshed every 150 rounds |
| 21a | Company alignment | Complete: durable companion alignment and loyalty, −100..100 display, drift toward the rest of the company, desertion, recruit gate, `company inspect`/`alignment` |
| 21b | Settlement standing | Complete: `internal/standing`, `modules/standing`, `standing` command, market/inn markups and refusals, black markets, Tanner's Back Alley |
| 22a | Creation step and starter kits | Complete: archetype step in `start`, per-archetype kits, owed-kit record and character claim marker, ash quarterstaff |
| 22b | Durable companion level and gear | Complete: `MemberState` on each companion (level, experience, worn and carried items, gold), restore from the record, snapshot seams, `company gear` |
| 22c | Recruiters and `company recruit` | Complete: recruiter rooms in config, free-once tutorial and paid candidates, claims on the company record, Waymark Inn and Trappers' Post, mobs 61–64 |
| 23a | Rest tiers | Complete: camp Rested (buff 1033, strain 75%), exclusive with inn Well Rested, durable companion grants, buff 16 renamed Refreshed |
| 23b | Whetstones | Next: 10-use whetstone, one use per member sharpened, usable any time; Sharpened weapons |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

### Phase 23a: rest tiers (2026-09-24)

- **What:** A completed camp rest now grants **Rested** (buff 1033,
  flag `rested`, 15 real minutes, walking strain 75%) to the leader and
  every live companion. An inn stay's **Well Rested** (1030, 30 minutes)
  replaces it, and a camp rest never downgrades Well Rested (it still
  restores fatigue). Grants happen on the game loop: the camp timer only
  marks `rested_pending` in the same save as its recovery. Every
  rostered companion's grant is kept durably (`owed`, real UTC expiry),
  so a companion absent at the grant, or respawned after a relog,
  restart, or copyover, gets the tier for the time left, never longer.
  Durations are camping config (`RestedDuration`, `WellRestedDuration`,
  `RestedBuffId`); `RestedPct` is walking config. Upstream buff 16 (a
  nap's reward, also named Well Rested) is renamed **Refreshed**, so the
  name means one tier. A `PlayerSpawn` listener removes Rested from a
  player also holding Well Rested. Pure rules are in
  `internal/camping/tiers.go`; grants are in `modules/camping/tiers.go`.
  Design and plan: [23a spec](superpowers/specs/2026-09-24-phase-23a-rest-tiers-design.md) /
  [23a plan](superpowers/plans/2026-09-24-phase-23a-rest-tiers.md).
  The owner amended the whetstone rules for 23b during this phase; the
  amendment is recorded in the parent spec.
- **Why:** Roadmap spec 2 (rest and weapon preparation), split like
  21a/21b: the rest tiers are 23a, whetstones 23b. Decisions were applied
  under the owner's "merge and begin the next phase" and are recorded in
  the spec.
- **Verification:** `go test -race ./...`, `make generate`,
  `make validate`. The wiring tests use the real `camp`/`inn` commands,
  timers, the `NewRound` listener, a `PlayerSpawn` through the event
  queue, and walking's `go` command.
- **Review:** The independent reviewer found:
  1. A mid-pass race: an inn timer's Well Rested marker could be cleared
     by a Rested grant. I had also found it, and it was fixed with
     `TestWellRestedMarkedMidPassIsNotLost`.
  2. Companion tiers were lost on relog, restart, or copyover. Fixed:
     durable grants for every companion, re-granted for the time left
     (`TestPresentCompanionRegrantedAfterRespawn`; reload and wiring
     tests extended).
  3. A failed save re-announced the grant every round. Fixed: a grant is
     announced only once saved.
  4. The grant was announced when nothing was granted. Fixed.
  5. A member holding both tiers kept Rested under Well Rested. Fixed:
     every lower tier held is removed.
  6. Restart tests were missing. Added for pending across a reload and
     for an overdue camp at load.
  7. Buff text nits. Fixed.

  Accepted and recorded in the spec: "present" means a live companion
  mob, and a persistently failing save re-grants the full duration until
  it succeeds.
- **Step completed:** Phase 23a. Phase 23b is next.

### Alignment display scale: −100..100 (2026-09-24)

- **What:** Players now see alignment on the engine's −100..100 scale
  (−100 most evil, 0 neutral, 100 most good) instead of 1–100, in
  `company status`, `company alignment`, `company inspect`, the
  `company summon` refusal, the `company recruit` list, and `standing`.
  `company.DisplayAlignment` now only clamps. Stored values, knobs, and
  behaviour are unchanged; knob points and displayed points are now the
  same, and the config comments say so. Tests updated to the new numbers,
  plus a check on the `company alignment` scale hint.
- **Why:** Owner preference. It also matches `help alignment`, which
  already listed the bands on −100..100. The 21a spec records the
  amendment.
- **Verification:** `go test -race ./...`, `make generate`,
  `make validate`.

### Phase 22c: settlement recruiters and `company recruit` (2026-09-24)

- **What:**
  - A recruiter is a room in the company module's `Recruiters` config.
    There, `company recruit` lists each candidate's archetype, level,
    alignment, gear, and price. `company recruit <candidate>` (id or name)
    takes one on.
  - Recruiting goes through `enlist`, the path `company summon` now also
    uses (survival identity, archetype, disposition, 22b template gear,
    one company save, full rollback).
  - Tutorial candidates are free and claimable once per account. The
    claim is the template id in `Record.Claimed`, saved with the recruit
    and rolled back with it. It outlives dismissal.
  - Paid candidates: gold is checked first and taken only after the
    company save, then the user is saved at once.
  - Content: the Waymark Inn (Tamsin Reed and Brother Oswin, free once;
    Garrick Vane, 120 gold) and Trappers' Post (Ysolde, 80 gold). These
    are new mobs 61–64, which wear their gear, carry nothing, and never
    drop anything.
  - Design and plan:
    [22c spec](superpowers/specs/2026-09-24-phase-22c-recruiters-design.md) /
    [22c plan](superpowers/plans/2026-09-24-phase-22c-recruiters.md).
- **Why:** This is the last part of the recruitment and creation spec.
  Players had no way to gain a companion other than the test-only
  `company summon`. Defaults were applied under the owner's "Continue
  with phase 22c" instruction and are recorded in the spec.
- **Step completed:** Phase 22c.
- **Verification:** `go test -race ./...`, `make generate` (no diff), and
  `make validate` passed. `modules/company` also passes with `-count=2`
  and `-shuffle=on`.
  - The wiring test uses `plugins.Load` with the shipped config, the real
    plugin store, and the shipped candidate mob files. Through
    `usercommands.TryCommand` it covers the listing, a name match, an
    ambiguous name, `company summon` refused for a candidate, a free and
    a paid recruit (the saved user file has the new gold), too little
    gold, `formation move`, `company status`, a real `plugins.Save()` and
    reload, dismissal, and a second claim refused. The clock is unchanged.
  - A shipped-data test pins the recruiter rooms, the templates and their
    items, archetypes, at least two tutorial candidates, and that no
    candidate can be summoned or farmed.
- **Review:** The independent reviewer found no blocking bugs and
  confirmed each invariant: no clock access, the claim and the recruit in
  one save, every rollback keeps claims, no summon bypass, and nothing to
  farm. Fixed:
  - Latent: a failed `Claim` in `enlist` restored the wrong record for a
    brand-new leader. It now uses the same restore as the survival
    failure. It can't be reached today, since `Claim` only fails on ids
    `Summon` has already accepted.
  - Name matching had no test coverage; the wiring test now recruits by
    name and refuses an ambiguous one.
  - Spawn or survival failure during a tutorial recruit had no test
    coverage; a new test checks that the claim is rolled back and
    persisted, and that no gold is taken.
  - The shipped test now also pins no loot category and no mob script,
    since a loot category drops items whatever the drop chance.
  - `company summon` of a candidate is now also refused through
    `usercommands.TryCommand` with the shipped config.

  Kept and documented: claims are per account, not per character, since
  alts share the leader's user id. That is stricter than the spec's
  wording. The accepted crash window between the company and user saves
  favours the player and can't repeat for a free candidate.

### Phase 22b: durable companion level and equipment (2026-09-24)

- **What:**
  - Each companion record now carries a `state`: level, experience, worn
    equipment (by slot), carried items, and gold. Restoration rebuilds the
    live mob from it: the mob is spawned at the saved level, and the
    template's minted gear and gold are replaced with copies of the saved
    ones. Gear given to a companion now survives logout, restart, and
    copyover, and template gear is minted only once.
  - `company summon` records the new recruit's template gear in the
    summon's own save. A companion saved before this phase has its state
    derived from the template and saved before it is spawned. If that save
    fails, it waits and is retried on the next spawn.
  - **Snapshot seams:**
    - a companion's `ItemOwnership` (in memory);
    - plugin `OnSave` (autosave, shutdown, copyover);
    - the leader's `PlayerDespawn`: the companion is recorded, then removed
      from the world so it can't linger to be looted;
    - the companion's `MobDeath`: gear and gold are cleared, the level is
      kept.
  - A companion befriended away by another player is lost: it keeps its
    gear and its record's gear is cleared.
  - Companions never roll elite.
  - `company status` shows levels; `company gear <member>` lists gear.
  - Design and plan:
    [22b spec](superpowers/specs/2026-09-24-phase-22b-durable-companion-state-design.md) /
    [22b plan](superpowers/plans/2026-09-24-phase-22b-durable-companion-state.md).
- **Why:** This is the durable model from the recruitment and creation
  spec, which 22c recruiters, sharpening, and resurrection build on.
  Before this phase every login re-minted template gear and dropped
  anything given to a companion. Defaults were applied under the owner's
  "Merge, then start Phase 22b" instruction and are recorded in the spec.
- **Step completed:** Phase 22b. Phase 22a was merged to `master` at the
  start of this step.
- **Verification:** `go test -race ./...`, `make generate` (no diff), and
  `make validate` passed. `modules/company` also passes with `-count=2`
  and `-shuffle=on`.
  - The wiring test uses `plugins.Load`, the real plugin store, and a
    fixture world with template gear and a 100% elite chance. It covers:
    - `company summon` and `give` through `usercommands.TryCommand`;
    - a real `plugins.Save()`;
    - `PlayerDespawn` (the mob is gone);
    - a simulated restart (reload from the real store, instances cleared)
      and `PlayerSpawn`, which restores exactly the recorded gear at the
      saved level, not elite;
    - a death through the real `mobcommands.Suicide`, after which the
      companion comes back without gear.

    The clock is unchanged throughout.
  - Engine regression tests in `internal/mobs` cover the template-items
    copy (the test fails without the fix) and the no-elite spawn.
- **Review:** Independent reviewer found no duplication in the normal
  logout, restart, copyover, death, and dismiss flows. It confirmed:
  - listener order (company before `HandleLeave`);
  - that link-dead leaders fire no `PlayerDespawn` until expiry;
  - `vanish`/`despawn` keep the record's gear with no duplicate;
  - the summon and legacy-upgrade rollbacks;
  - deep copies, and that there is no clock access.

  Fixed with regression tests:
  - Major: saving the company file on every gear change put it out of
    step with the user and room files, so a crash after a give duplicated
    the item. Gear changes are now recorded in memory and written by the
    same saves as the user and room files.
  - Shutdown's final plugin save and the SIGUSR1 copyover ran off the game
    loop while `OnSave` now writes game state. Both now hold
    `util.LockMud()`.
  - Elite rolls on restore inflated stats and changed a durable level. New
    `mobs.NewMobByIdNoElite`.
  - Gold wasn't durable: given gold was lost at logout, and template gold
    was re-minted and could be farmed through deaths. Added to the state
    and cleared on death.
  - Engine bug, pre-existing: `NewMobById` shared the template's `Items`
    array, so removing an item from a live mob rewrote the template.
    Instances now copy it. This also gives each spawn fresh item UUIDs.
  - A companion befriended by another player was snapshotted into the
    leader's record and then destroyed at logout. It is now treated as
    lost.
  - Item spec overrides were shared between the record and the mob; they
    are now copied.
  - The wiring test now uses the real `Suicide` death path and a real
    `plugins.Save()`.

  Documented, not changed:
  - item UUIDs aren't durable (engine `yaml:"-"`);
  - the window before `MobDeath` is processed is within one locked turn;
  - `company gear` refreshes the in-memory record from the live mob.

### Phase 22a: creation step and starter kits (2026-09-24)

- **What:**
  - `start` now asks "Which archetype will you follow?" between the name
    step and `CharacterCreated`. It lists each archetype's description,
    skills, and starter kit, and takes a number or name, then a yes/no
    confirmation. It goes through a new `internal/archetypes.Creator`
    seam, so the engine still never imports the module.
  - Each archetype has a `Kit` of item ids in `modules/archetype` config.
    Shipped kits total 246–266 in value; a new ash quarterstaff (10021)
    gives the wizard a staff.
  - A committed choice records the kit it owes (`Registry.Kits`) in the
    same save. The grant wears kit gear into empty slots, packs the rest,
    sets the character's `MiscData["archetype-kit"]` marker, and saves
    the user. Items and marker live in the same user file.
  - The grant runs after a choice and again on every `PlayerSpawn`, and
    gives only when a kit is owed and the marker is absent. That makes it
    exactly once through reconnect, restart, copyover, and
    reset-and-rechoose.
  - Characters that chose before this phase keep their gear and get no
    kit.
  - `archetype` and the `choose` preview show kits.
  - Design and plan:
    [22a spec](superpowers/specs/2026-09-24-phase-22a-creation-starter-kits-design.md) /
    [22a plan](superpowers/plans/2026-09-24-phase-22a-creation-starter-kits.md).
- **Why:** The recruitment and creation spec is first on the onboarding
  roadmap. It is split like 19/19b and 21a/21b: 22a covers creation and
  kits, 22b durable companion gear, and 22c recruiters. Defaults were
  applied under the session's "Begin next phase" instruction and are
  recorded in the spec.
- **Step completed:** Phase 22a.
- **Verification:** `go test -race ./...`, `make generate` (no diff), and
  `make validate` passed. `modules/archetype` also passes with `-count=2`
  and `-shuffle=on`.
  - The wiring tests drive the real `start` command through its prompt,
    with the real module as provider. They cover every shipped archetype
    (kit owned, weapon in hand, not encumbered, tutorial question reached)
    and:
    - reconnect before and after choosing;
    - a "no" or empty confirmation;
    - an unknown answer;
    - a failed commit;
    - persistence unavailable;
    - no provider;
    - an admin reset mid-creation;
    - a stored but unconfigured archetype.
  - `PlayerSpawn` and a permanent `PlayerDeath` go through
    `events.ProcessEvents`.
  - The shipped-data tests pin kit ids and value balance (within 1.25×,
    with buffs loaded as on the server). They also check that no
    selectable race starts over its carry capacity.
- **Review:** Independent reviewer found no way to duplicate a kit through
  the new code. It confirmed:
  - lock discipline (no engine call under `m.mu`);
  - no clock access, and no engine import of the module;
  - `PlayerSpawn` fires on login, link-dead reconnect, and copyover;
  - the marker survives the YAML user save.

  Fixed with regression tests:
  - Major: four of five kits left a new character over its carry
    capacity (5 at creation), so every step cost 5× action points. Kit
    gear is now worn into empty slots only (never displacing gear), and
    the rest is packed.
  - A module that failed to load still offered the step, then leaked an
    internal error. The step is now skipped.
  - A kit item missing after a data reload set the marker on a partial
    kit. The grant is now all or nothing and leaves the kit owed.
  - An admin reset mid-creation replayed the prompt's cached answers into
    a silent re-commit. The step now runs once per prompt.
  - A stored but unconfigured archetype hid the step, although `archetype
    choose` allows re-choosing. The two now agree.
  - Added a real `PlayerDeath` event test and an empty-confirmation test.
  - Also found while fixing: the engine's `HandsRequired` panics without a
    known race, so kit gear is packed rather than worn in that case.

  Documented, not changed:
  - the upstream give-then-crash duplicate window after a failed user
    save;
  - the Phase 17 permadeath-clear save failure, which carries the old
    choice (and now its kit) over;
  - the upstream tutorial's extra newbie kit;
  - that `help` isn't supported at the archetype question.

### Phase 21b: settlement standing (2026-09-24)

- **What:** A settlement is a configured zone with an alignment (Dunmar
  40, Old Kings Road 0, Frostfang 30). A company's standing there comes
  from the gap between its Phase 21a average and the settlement's:
  welcome (≤ 40), tolerated (≤ 80), distrusted (≤ 130), or shunned.
  Distrusted companies pay 20% more and are paid 20% less at the market
  and pay 50% more for an inn room. Shunned companies are refused by
  both. Rooms tagged `blackmarket` trade the zone ledger at normal prices
  with distrusted and shunned companies only; Tanner's Back Alley (2006),
  south of Dunmar Market Square, is the first. `standing` shows the
  current settlement's view of your company and your standing elsewhere.
  Standing is derived every time and never stored. Pure rules and the
  seam are in `internal/standing`, config and the command in
  `modules/standing`, and `internal/company` gained an
  `AlignmentProvider` seam served by `modules/company`. Design and plan:
  [21b spec](superpowers/specs/2026-09-24-phase-21b-settlement-standing-design.md) /
  [21b plan](superpowers/plans/2026-09-24-phase-21b-settlement-standing.md).
- **Why:** Roadmap decision 2(4), settlement standing (prices, inn
  access, black-market access), split out of Phase 21. Defaults were
  applied under the owner's go-ahead and are recorded in the spec.
  Effects are penalties only, so Phase 19b's no-profitable-round-trip
  invariant holds with no re-capping.
- **Also resolved:** the Phase 19b inter-market profit question, on the
  owner's instruction to settle open questions with the recommendation.
  The small standing trade-route profit and starting stocks stay.
- **Step completed:** Phase 21b, and with it Phase 21 and the
  2026-09-23 roadmap.
- **Verification:** `go test -race ./...`, `make generate` (adds
  `modules/standing` to `all-modules.go`), and `make validate` passed.
  The standing, market, and camping packages also pass with `-count=2`
  and `-shuffle=on`. The wiring test in `modules/standing` loads the real
  company, market, camping, survival, and standing modules through
  `plugins.Load` with the shipped world's items and Dunmar rooms. It runs
  `standing`, `market`, and `inn` through `usercommands.TryCommand` and
  checks four cases:
  - a distrusted company's marked-up listing and purchase;
  - that company served at the alley;
  - a shunned company refused at the square and the inn;
  - a welcome company turned away from the alley.

  It also checks the shipped room's tag and exits.
- **Review:** Independent reviewer found no critical or major bugs. It
  confirmed:
  - no clock access and no new durable state;
  - `standing.For` is only called on the game loop, outside the market
    and camping locks, and never from camping's timers;
  - the no-profit invariant holds under every tier, the sell floor, and
    the black market;
  - every refusal path moves no gold, items, or stock.

  Fixed with regression tests:
  - A room with both the market and black-market tags refused welcome
    companies. It is now an ordinary market for anyone the black market
    doesn't serve.
  - The `standing` text promised a market, inn, or black market that a
    settlement may not have. It is now worded conditionally.
  - The inn's per-member figure didn't match the marked-up total. It now
    shows "+50%".
  - A duplicated settlement warned once per entry instead of once per
    zone.
  - Added tests for a black market outside any settlement and for a sell
    floor of 1 at the maximum markup.

  Documented, not changed:
  - Standing fails open when company data is unavailable.
  - The gap is symmetric, so a saintly company is distrusted at the
    neutral Trappers' Post. Only Dunmar can shun anyone with the shipped
    alignments.
  - An empty `BlackMarketRoomTag` falls back to `blackmarket`.

### Phase 21a: company alignment (2026-09-24)

- **What:** Each companion now has a durable `disposition`: an alignment
  on the engine's −100..100 scale and a loyalty from 0 to 100. A new
  recruit takes its mob template's alignment (or the race default) and
  `StartLoyalty` 70. Companions saved before this phase are seeded the
  same way on load, at full loyalty. Players see alignment as 1–100 in
  `company status`, `company alignment`, and `company inspect <mob>`.
  Every `DriftEveryRounds` (75) rounds, each online leader's companions
  move up to `DriftStep` (2) toward the rest of the company (at most half
  the gap). A companion more than `LoyaltyToleranceGap` (60) from the
  rest loses 5 loyalty; otherwise it gains 2. The leader is warned below
  25, and at 0 the companion deserts through the dismissal path, but
  never while the leader or that companion is fighting. The leader never
  drifts. `company summon` refuses a candidate more than `RecruitMaxGap`
  (60) from the company average. The live mob's alignment follows the
  record. The drift countdown is saved with the company store. Pure rules
  are in `internal/company/alignment.go`; the seeding, drift listener, and
  commands are in `modules/company`. Design and plan:
  [21a spec](superpowers/specs/2026-09-24-phase-21a-company-alignment-design.md) /
  [21a plan](superpowers/plans/2026-09-24-phase-21a-company-alignment.md).
- **Why:** Roadmap decision 2 (loyalty/desertion and recruit gates) and
  Phase 21's companion alignment, 1–100 display, and drift. Phase 21 was
  split like 19/19b: settlement standing crosses into the market and inn
  modules and is Phase 21b. Defaults (scale, leader not drifting, knob
  values) were applied under the session's "Implement next phase"
  instruction and are recorded in the spec.
- **Step completed:** Phase 21a. Phase 21b is next.
- **Verification:** `go test -race ./...`, `make generate` (no diff),
  and `make validate` passed; `modules/company` also passes with
  `-count=2` and `-shuffle=on`. The wiring test runs `plugins.Load` in a
  disposable world, then `company inspect`/`summon`/`alignment`/`status`
  through `usercommands.TryCommand` with real mob specs and live mobs. It
  covers a refusal and the exact gap limit, and 75 real `NewRound`
  events through `events.ProcessEvents` drift the live mob's
  `Character.Alignment` and write the real store. The round and turn
  counts are unchanged.
- **Review:** Independent reviewer found no critical bugs. It confirmed
  there's no clock access, the countdown and dispositions persist (legacy
  wire decoder included), copies are deep, a failed drift save rolls
  back, and desertion reuses dismissal's rollback. Fixed with regression
  tests:
  - `plugins.Load` in the wiring test closed plugin registration for the
    test binary, so `-count`/`-shuffle` runs failed. A new
    `plugins.SnapshotLoadStateForTest` restores it.
  - The recruit gap (80) was looser than the loyalty tolerance (60), so
    accepted recruits started out losing loyalty. It now defaults to 60,
    and `inspect` warns when a looser config would admit an uneasy
    recruit.
  - A full company was refused for alignment instead of capacity.
  - `inspect` judged against an unloaded company.
  - Desertion only waited for the leader's fight, not the companion's.
  - Symmetric pairs swapped values every tick. Moves are now capped at
    half the gap.
  - A content companion at 0 loyalty with `LoyaltyGain` 0 would desert.
    Only uneasy ticks desert now.
  - Nested parentheses in the refusal message.

  Clarified rather than changed: a desertion postponed by combat is
  re-evaluated on the next tick, so a companion that has become content
  stays. Config combinations (`StartLoyalty` below `LoyaltyWarnBelow`)
  are documented in the overlay, not enforced. Also fixed:
  `lifecycle_test`'s data-dir override couldn't be restored, because
  `AddOverlayOverrides` never overwrites a set key.

### Phase 20: trade rumours (2026-09-24)

- **What:** In any room tagged `inn` (`RumorRoomTag`), `rumors` (also
  `rumours`, `rumor`, `rumour`) gives up to `RumorsPerAsk` (3) hints
  drawn at random from the market news: which market is short of or
  drowning in a good, and, for goods sold in two or more markets, where
  it comes cheapest and who pays best. No prices are quoted, and places
  are named by their market room. The news is a snapshot of every
  market's stock, refreshed every `RumorRefreshRounds` (150) rounds and
  saved with the ledger, so it is stale by design. Pure rules in
  `internal/market/rumor.go`; the snapshot, countdown, and command in
  `modules/market`. Design and plan:
  [20 spec](superpowers/specs/2026-09-24-phase-20-trade-rumours-design.md) /
  [20 plan](superpowers/plans/2026-09-24-phase-20-trade-rumours.md).
- **Why:** The roadmap's "trade rumours hint at where goods are cheap or
  wanted". Defaults (module owner, inn tag, snapshot staleness, four
  rumour kinds, free to ask) were applied under the session's "continue
  with next phase" instruction and are recorded in the spec.
- **Step completed:** Phase 20. Phase 21 (alignment) is next.
- **Verification:** `go test -race ./...`, `make generate` (no diff),
  and `make validate` passed. The wiring test runs all four command
  names through `usercommands.TryCommand` in the shipped Waymark Inn,
  checks refusals at the West Gate and Market Square, refreshes the news
  with a real `NewRound` and proves it moved from the stale start stock
  to live stock in the real store, and loads a truncated `news` section
  from the real file (left untouched, markets disabled).
- **Review:** Independent reviewer found no correctness bugs and
  confirmed the invariants (no clock access, leaf lock, atomic
  persistence, clean upgrade of pre-Phase-20 stores). Fixed with
  regression tests: the refresh countdown lived only in memory, so
  restarts more often than the interval meant the news never refreshed
  (now saved with the ledger and resumed, clamped to the interval); a
  newly configured market stayed out of the news until a refresh (now
  added at load); with no markets, every load and refresh rewrote the
  store (no longer); added the singular `rumor`/`rumour`; wiring test
  now covers a truncated `news` section through the real file and
  asserts live stock had moved before the refresh. Not changed: a
  non-string `RumorRoomTag` falls back to `inn` silently (same as
  `RoomTag`); a market can be both "drowning in" and "best buyer" for a
  good when zones use different curves (accurate under the design); the
  only open market counts as cheapest when the other is sold out (by
  design). The 19b inter-market profit decision is still open; the
  cheapest/best-buyer rumours point players at that route.

### Phase 19b: market trading (2026-09-23)

- **What:** The market now lives in rooms tagged `market` inside a
  market zone (configurable `RoomTag`). There, `market` lists a "You
  buy" and a "You sell" price per good, and `market buy`/`market sell`
  trade one unit against the zone ledger. The sell price is the stock
  price less `SpreadPct` (20), capped below the next unit's buy price,
  so no round trip within a market profits. Elsewhere in a market zone,
  `market` names the market room. New rooms: Dunmar Market Square (2004,
  east of the West Gate) and Trappers' Post (2005, east of the Fork at
  the Black Oak). Design and plan:
  [19b spec](superpowers/specs/2026-09-23-phase-19b-market-trading-design.md) /
  [19b plan](superpowers/plans/2026-09-23-phase-19b-market-trading.md).
- **Why:** The owner clarified that market prices belong to the local
  commodity market, which is its own trader; shopkeepers are untouched.
  This replaces Phase 19's recorded plan to route shopkeeper trades
  through market prices, which is no longer wanted.
- **Step completed:** Phase 19b. Phase 20 is next.
- **Verification:** `go test -race ./...`, `make generate` (no diff),
  and `make validate` passed in a clean worktree checkout. The wiring
  test loads the shipped market rooms zone-indexed and trades through
  `usercommands.TryCommand` in both market rooms, with the real
  `zoneExists` and market-room lookup, a real `NewRound`, and reloads
  of the real store; it also checks the ledgers stay separate and a
  downed player is refused.
- **Review:** Independent reviewer confirmed no profitable round trip
  within a market (including segment boundaries, flat curves, and the
  stock limits), no overflow, the leaf lock, and no clock access. Fixed
  with regression tests: a missing item spec could charge gold and stock
  for nothing (the item is now made first); `market sell hide` could pick
  hide armor or a special hide over a plain one (it now prefers an
  ordinary traded item); short names bought the first matching good
  (now asks which); buys didn't queue `events.Purchase` (now do); the
  real market-room lookup and zone check were never tested (a new
  `rooms.SetTestZoneRoom` helper lets the wiring test use both); Old
  Kings Road trading, ledger separation, the downed refusal, special
  items, and trade events were untested. Also fixed: messages now say
  "the" instead of "a", market room titles are cached per zone, the
  Fork's description no longer blurs its east exit with the old eastern
  track, and a wrong test comment. Documented rather than fixed: after a
  hard crash the ledger can keep a trade the character lost (same as
  shopkeepers); enchanted goods sell as plain ones (same as `sell`).
  **Open, owner decision:** trading between the two markets profits
  about 3-6 gold per trip at equilibrium and repeats indefinitely, and
  the shipped starting stocks give the first trader about 150 gold once.

### Phase 19: commodities and markets (2026-09-23)

- **What:** Added `internal/market` (validated goods; a two-segment,
  overflow-safe stock-to-price curve; bounded target-seeking stock drift;
  coarse stock descriptors) and `modules/market` (a durable zone-keyed
  stock ledger seeded from the module's config overlay, a
  `events.NewRound` drift listener with one batched save per tick, and a
  read-only `market` command). A settlement is a zone; only configured
  zones have a market. Shipped markets in Dunmar (hides, meat, thyme;
  starts short) and Old Kings Road (hides glutted, meat). Plugin data
  writes (`internal/plugins` `WriteBytes`) are now atomic for every
  module.
- **Why:** Phase 19's confirmed narrow slice: an engine-first market
  registry mirroring `modules/weather`, never touching the world clock
  and surviving restart/copyover.
- **Buy/sell decision (plan task 1):** vendor trading does **not** use
  market prices this phase. *(Superseded by Phase 19b: the market is its
  own trader and shopkeepers stay as they are.)* Price reaches a transaction through
  `buy.go`, `list.go`, `sell.go`/`offer.go` via `GetSellPrice`
  (quantity-scaled), player shops, and `shophooks.go`, and vendor
  quantity restocks independently; making them agree is Phase 19b,
  scheduled before Phase 20.
- **Step completed:** Phase 19.
- **Verification:** `go test -race ./...`, `make generate` (no diff),
  and `make validate` passed in a clean worktree checkout. The wiring
  test drives the real entry points: `plugins.Load` (command
  registration, shipped config overlay, `OnLoad` seeding into a temp
  plugin-data dir), `usercommands.TryCommand("market")` in market and
  non-market rooms, a real `NewRound` through `events.ProcessEvents`,
  `plugins.Save()`, and reloads of the real store file (including
  corrupt, empty, and truncated files).
- **Review:** Independent reviewer confirmed the invariants (no clock
  access, leaf mutex, restore-not-reseed, overflow-safe math) and found
  no critical bugs. Fixed with regression tests: an empty or truncated
  store file (possible because plugin writes truncated then rewrote)
  decoded as an empty ledger and was re-seeded over (plugin writes are
  now temp-file-plus-rename, and the ledger rejects empty data and
  records missing stock); a save before the first load could overwrite
  the store (persistence is now off until a successful load); a
  repeated zone kept its first, possibly broken, entry (now no market,
  matching duplicate items); id-less goods tripped the duplicate-item
  check; fractional ids were truncated; non-map entries were skipped
  silently; the shipped "glutted" comment was wrong (Old Kings Road
  hides now start at the ceiling); and coverage for `plugins.Save()`,
  real corrupt files, top-level mixed-case keys, and pre-round
  out-of-range quotes. Partly addressed: the real `zoneExists` still
  isn't run against loaded rooms (loading shipped rooms in a test
  writes a `NextRoomId` config override into the data dir); the test
  now checks market zones against the shipped rooms' `zone` fields, the
  key `GetAllZoneNames` uses. Recorded as a known limitation, not fixed:
  stored stock records are never pruned (see the spec's constraints).
- **Known issue found:** some existing test that loads the shipped
  world's rooms (likely `modules/weather`'s shipped-world test) writes
  a gitignored `_datafiles/world/default/config-overrides.yaml`. If
  that file ever contains `FilePaths.DataFiles`, `modules/walking`'s
  wiring tests fail on a relative data path. Delete a stray copy before
  running the suite.

### Phase 18b: cooking (2026-09-23)

- **What:** Room containers gained optional `reciperequirements` (skill
  and minimum level per recipe output). `use <container>` now picks the
  lowest ready recipe the player can make and refuses, consuming nothing,
  when only gated recipes are ready; `look` lists recipes in order with
  their requirement. Shipped a `cooking` skill, a `cook` profession, three
  meals from 18a meat and thyme, and a Waymark Inn hearth that trains
  cooking 1-4. Camp campfires are light fixtures, not containers, so camp
  cooking is deferred.
- **Why:** Close Phase 18 with the minimal cooking slice the spec
  confirmed, reusing the existing container crafting path.
- **Step completed:** Phase 18b (Phase 18 complete). Phase 19 is next.
- **Verification:** `go test -race ./...`, focused `go vet`, `go build ./...`,
  `make generate`, and `make validate` passed. Wiring tests drive the real
  `use` and `look` commands and a shipped-content test cooks at the real
  Waymark Inn hearth.
- **Review:** Independent reviewer found three important issues, all
  fixed with regression tests: the in-game room editor could save a
  requirement for a deleted recipe that would stop the next boot (now
  pruned before save); the web admin room PATCH dropped every requirement
  (now carried over for surviving recipes); and instance saves froze a
  container's recipes, hiding later template edits (recipes are now
  re-applied from the template on load; the test fails without the fix).
  Also fixed: missing output assertions for the refusal message and `look`
  listing, and the one-skill `cook` profession (now cooking, track,
  search). Deferred: load-time warnings for unknown requirement skills or
  over-max levels, since rooms load before skills; the shipped hearth is
  covered by the startup content test. The web PATCH fix is tested at the
  helper level, not through the HTTP handler.

### Company life and onboarding specifications (2026-09-23)

- **What:** Wrote a six-spec planning packet for recruitment/creation,
  camp Rested and company-wide whetstones, chemistry, player death and
  companion resurrection, command/browser surfaces, and the Ashveil
  tutorial. See the [roadmap](superpowers/specs/2026-09-23-company-life-onboarding-roadmap.md).
- **Why:** Record the owner's approved gameplay rules and the durable
  companion/settlement prerequisites before implementation planning.
- **Step:** Design documents drafted for owner review; no gameplay phase
  completed and Phase 18b remains next.

### Phase 18a: loot tables (2026-09-23)

- **What:** Added weighted category loot tables, boot/reload loading, a
  `LootCategory` mob tag, and a second death-drop roll beside existing gear
  drops. Timber wolves and ruffians use shared beast/humanoid tables; new
  hide, meat, and herb items support the later cooking slice.
- **Why:** Deliver the next planned loot mechanism without changing drops for
  uncategorized mobs. Missing loot directories and unknown item IDs leave
  existing worlds bootable and skip only the optional drops.
- **Step completed:** Phase 18a. Phase 18b remains next, per user direction.
- **Verification:** `go test -race ./...`, focused `go vet`, `go build ./...`,
  `make generate`, and `make validate` passed. The command test covers worn,
  carried, locked, corpse, floor, missing-spec, and event behavior.
- **Review:** Independent reviewer found four important gaps: absent loot
  directory handling, invalid nonpositive item IDs, incomplete worn-item
  coverage, and a reload test that could pass on stale data. All four were
  fixed and covered with regression tests; the worn-item and reload tests
  were mutation-checked. No findings were rejected.

### Phase 18–19 planning review (2026-09-23)

- **What:** Tightened the Phase 18 and 19 design specs and implementation
  plans after a read-only review of the relevant engine paths. The plans
  now cover loot-table boot/reload wiring, invalid item IDs, corpse/floor
  event semantics, deterministic skill-gated cooking recipes, complete
  vendor-price reconnaissance, and a defined market price/drift model.
- **Why:** These gaps could otherwise produce silent no-op loot, broken
  items, inconsistent trade quotes, or ambiguous market behavior.
- **Next:** Phase 18a remains the next implementation step. Phase 19's
  vendor trading decision remains at its first reconnaissance task.

### Phase 17: archetypes and utility skills (2026-09-23)

- **What:**
  - **17a:**
    - `internal/archetypes` (pure): the claim table, gating decisions,
      and a seam that allows everything without a provider.
    - `modules/archetype`:
      - the archetype table (config): warrior, rogue, wizard, cleric, and
        ranger
      - a one-time `archetype choose <name> confirm` that is persisted and
        applies grants (re-applied on login)
      - admin `archetypereset`
    - **Claimed skills and spell schools are gated** at every place they
      are handed out:
      - `train`
      - script `TrainSkill`
      - script and party `LearnSpell`
      - quest skill rewards

      Existing skills and spells are grandfathered.
    - **Companions** get an archetype on the company record: set from
      config at summon (template 58 is a warrior), or once with
      `company archetype`.
    - `look` shows archetypes.
    - The `illlusion` spell-school typo is fixed.
  - **17b:**
    - Utility levels, with company best-member resolution.
    - `autoskill` toggles.
    - `trap sense` / `trap disarm`. A disarm is persisted by round, and
      `picklock` respects it.
    - A free sense before picking a lock.
    - Auto-sense on entering a room.
    - Wizard auto floating light, through the real cast path for players
      and a direct buff for companions.
    - These use a new `walking.AddStepListener` seam.
    - Content: a trapped toll box at Dunmar West Gate (2001).
- **Why:** This is the roadmap phase for user decision 3 (exclusive
  archetypes and utility skills). The user confirmed decisions 1–8 as
  recommended. Deviations are recorded in the spec's "Implementation
  notes" ([spec](superpowers/specs/2026-09-23-phase-17-archetypes-utility-skills-design.md),
  [plan](superpowers/plans/2026-09-23-phase-17-archetypes-utility-skills.md)).
- **Durable / clock-safe:**
  - Archetype choices, autoskill toggles, and disarm expiries (as round
    numbers) persist in the archetype plugin store. Companion archetypes
    persist on the company record.
  - Legacy records load neutral.
  - There are no timers, and nothing advances the clock.
  - The module lock is a leaf lock, and engine reads happen outside it.
- **Review:** each slice had an independent reviewer subagent (the most
  capable model tier).
  - **17a:** 3 medium-high, 3 medium, and 4 low findings, plus coverage
    gaps.
    - *Fixed, each with a regression test:*
      - Script `TrainSkill` (the obelisk teaching `portal`) and quest
        `SkillInfo` rewards bypassed the gate.
      - A spell-school typo on either side went unreported; it is now
        warned about at load.
      - Permadeath kept the old character's archetype; it is now cleared.
      - A removed archetype stranded its players; they can choose again.
      - A failed registry load misreported everyone as unchosen; claimed
        skills now fail closed with an explicit reason.
      - `loadErr` was read without the lock.
    - *Fixed without an automated test:* Elara's `illum` lesson locked out
      anyone who finished the quest before becoming a wizard. She now
      re-teaches it from both `onGive` and `onAsk`. The script is
      syntax-checked; there is no mob-script test harness.
    - *Accepted and documented:*
      - Grandfathered claimed skills can't be trained higher without the
        archetype.
      - Login re-grants undo an admin's removal of a granted skill.
      - `archetypereset` only works on online characters.
    - *Coverage added:*
      - the real plugin-config path, for archetype and for company
      - the `company archetype` command
      - spawn and permadeath through the event queue
  - **17b:** no critical findings; 3 medium and 8 low, plus coverage
    gaps.
    - *Fixed, each with a regression test:*
      - Step reactions printed before the move text; the hook now runs
        last in `go`.
      - A downed or fighting companion could act or mask an able player
        wizard.
      - The auto-light cooldown was spent before checking anyone could
        cast.
      - Companion backfire grammar.
      - A mistyped `trap sense` target spent the cooldown.
      - The free pre-pick sense was used up when nobody could sense.
      - Engine reads happened under the module lock.
      - Permadeath kept the autoskill toggles.
    - *Accepted and documented:*
      - Disarms last a fixed 900 rounds rather than the lock's relock
        interval.
      - An exit disarm covers only that room's side of the door.
      - A permadeath clear isn't retried if the registry is down.
    - *Coverage added:*
      - a following companion rogue
      - unable companions
      - both cooldowns
      - the shipped toll box end to end
- **Process note:** the pure `internal/archetypes` tests and the 17b
  module code were written in the same pass as, or just before, their
  implementation, not strictly tests-first. The wiring tests for the
  train, `LearnSpell`, look, and quest gates, and every review regression,
  were seen failing first. The picklock and quest-reward wiring tests were
  also checked by putting the bug back and watching them fail.
- **Verification:**
  - `go test -race ./...` passes (65 packages).
  - The archetype, archetypes, and walking packages pass
    `-race -count=20`.
  - `go vet ./...` and `make validate` pass.
  - `make generate` adds `modules/archetype`.
  - The server boots with 25 plugins and no archetype or school-mismatch
    warnings. The only warnings are pre-existing content and config ones.
  - Live telnet acceptance was not run.
- **Known limitations:**
  - Cooking is deferred to Phase 18b. Room containers already support
    crafting `recipes` to build on.
  - No new trainers: grants give level 1.
  - Upstream trapped locks exist nowhere else in the content.

### Phase 16: walking fatigue, inns, and travel/rest multipliers (2026-09-23)

- **What:**
  - **`internal/walking`** (pure): terrain cost (settlements free, a biome
    table, a `strain:<n>` room tag), a clamped multiplier product, a
    centi-fatigue carry, and the cold multiplier.
  - **`modules/walking`:** `usercommands.Go` reports each successful
    ordinary step. The module charges the leader and the spawned
    companions walking with them, using load, outdoor weather, cold
    exposure, Well Rested, and mount relief (the leader and the lowest
    companion ids, at most `Riders`). The carry flushes on save.
    Exhausted (1031) and Collapsed (1032) band buffs, the `strain`
    command, and the Well Rested buff (1030) with its `well-rested` flag.
  - **Travel:** `TravelSession` locks `DurationPct`, `ExertionPct`, and
    `FatiguePct` at departure (0 = neutral, so legacy sessions are
    unchanged). A company with a member at fatigue 0 can't set out.
    Ordinary movement is never refused.
  - **Camping:** camp rest locks the weather-scaled recovery. Inns add the
    `inn` tag and command and a durable `InnStay`. Gold is refunded if the
    save fails. A stay blocks movement. Well Rested is granted on the
    round tick, never from the timer.
  - **Seams:** `encumbrance.CurrentBand`, `mount.Relief` and
    `mount.TravelDurationPct`, and `climate.ExposureOf`.
  - **Content:** the Waymark Inn (Dunmar 2003), the `inn` tag on the
    Frostfire (61), and Fork at the Black Oak (2002) moved into a new
    forest zone, Old Kings Road, so the proving route and camp see weather.
- **Why:** the 2026-09-23 roadmap. It also switches on the travel/rest
  multipliers deferred as Option A in Phases 8–10. The user confirmed
  decisions 1–7 (decision 3 changed to two riders). Deviations from the
  design are recorded in its "Implementation notes".
- **Verification:** `go test -race ./...` passes, as do `make generate`
  (adds `modules/walking`) and `make validate`. Real-entry-point wiring
  tests go through `usercommands.Go` (forest/city/route), `Go` into a real
  `StartTravel`, the camp and inn user commands (real room 2003, through
  to Well Rested), and the real exposure→walking chain. The `-race` tests
  cover cross-module steps, ticks, and timers, and camping's real timers
  against the game loop (`-count=20`). The server boots with +3 buffs, +1
  flag, +1 plugin, +1 zone, and +1 room, and no new warnings or errors
  compared with the pre-phase boot.
- **Review:** an independent reviewer (Sonnet subagent) found no bugs
  against the invariants (clock, durability, lock order, idempotency,
  refunds). It raised three low-severity concerns:
  - Config read outside the lock. Camping's `innRest` read inn config
    before taking the lock that `load()` writes it under. **Fixed:** it
    now reads under the lock. The walking half was **rejected:** `load()`
    runs once at boot before the game loop, the same pattern accepted for
    Phase 15's exposure module.
  - The Well Rested buff id is read at grant time instead of being locked
    onto the stay. **Rejected:** it is static deploy config, not a
    per-booking value.
  - The travel check runs before the camping lock. **Rejected:** `inn rest`
    and `go` both run on the single game loop, and an active stay already
    blocks route starts.

  Test gaps it noted and left open: no concurrent-reload test, and no
  real-timer race test for expedition (a pre-existing gap, not introduced
  by this phase).
- **Known limitations:** mount relief doesn't apply to route travel yet.
  Upstream buff 16 is also named "Well Rested" (from the Frostfire room
  rental, left unchanged). Inn price and power are placeholders pending the
  economy pass.

### Phase 15: temperature, clothing, and exposure (2026-09-23)

- **What:**
  - **`internal/climate`** (pure): air temperature, clothing warmth
    (per-slot defaults, explicit `warmth`, negative = none, `warmed` +20),
    the comfort band and stress, a signed exposure step toward
    `stress × 4`, bands, and heat-source and temperature-provider seams.
  - **`modules/exposure`:** every 5 rounds it moves exposure for online
    leaders and their spawned companions. It keeps the band buff refreshed
    (cold 1010–1013, heat 1020–1023), deals damage only in the severe band
    (never below 1 HP) and the critical band (outpacing regen), and drains
    fatigue in cold and thirst in heat. It persists exposure, clears it on
    player death, and adds the `temperature` command.
  - **Items** gain `warmth`; 30 existing clothes have values.
  - **Weather** conditions gain `TemperatureMod`, and the `weather` command
    shows the temperature.
  - **Campfires** warm their room.
  - **Survival** gains a non-ledgered `ApplyMemberDrain` that marks the
    registry dirty and flushes on the next save, and a module mutex.
- **Why:** User decision 4. Exposure kills only at extremes, after
  escalating penalties; mild mismatch only penalises. A tested invariant
  ensures only stress ≥ 25 can reach the lethal band.
- **Durable / clock-safe:** exposure is persisted per leader and member.
  Ticks are `NewRound` listeners that only read the clock. Offline
  characters don't tick.
- **Review (independent reviewer subagent over `f1386664..25e3ac15`):**
  1. *Critical, fixed:* the severe and critical buffs cut vitality by 30,
     which with `HPPerVitality` 4 collapsed max HP, so a supposedly
     survivable band killed players. Vitality was removed. A test now loads
     the shipped buff files with real config and checks that max HP
     survives; it fails if vitality is put back.
  2. *High, fixed:* survival had no mutex, and travel and camping timer
     goroutines write to it while exposure drains from the game loop, which
     risked concurrent map writes. Survival now takes a leaf mutex on every
     exported entry point, with a concurrent test under `-race`.
  3. *High, fixed:* damage didn't account for regen, so low-level players
     were immune and severe damage could down them. Severe now never goes
     below 1 HP, and critical adds the player's regen per tick.
  4. *Medium, fixed:* drains rewrote the whole survival file per member per
     tick. They now mark it dirty and flush on the next save.
  5. *Medium, fixed:* permanent band buffs were stripped by the engine's
     permabuff reconciliation on equip, remove, and login. They are now
     short non-permanent buffs refreshed each tick, with a regression test.
  6. *Medium, fixed:* companions that weren't spawned yet (after a restart)
     lost their stored exposure. Pruning now uses the roster, and unspawned
     companions are left untouched.
  7. *Low, fixed:* death now clears the player's exposure. "Furnished" now
     matches the design (lit biome, or an `indoor`/`lit` tag). The weather
     command's line reads "Temperature here".
  8. *Low, deferred:*
     - Upstream Freezing Snow (buff 31) still stacks with exposure in snow
       rooms; it is content for a builder to retire.
     - Admins and players in combat are not exempt.
     - Companions don't regenerate out of combat, so severe cold wears them
       down to 1 HP.
     - Drain errors can repeat in the log while survival persistence is
       down.
- **Process note:** for `modules/exposure` the implementation was written
  just before its tests, contrary to the tests-first plan. The tests then
  caught two bugs: buff revival after removal, and a wrong expectation.
- **Known pre-existing risk (not this phase):** camping and expedition
  timer callbacks run off the game loop and also read company state, which
  has no lock. Survival is now safe; company isn't audited.
- **Verification:** `go test -race ./...` passes (1886 passing test
  results). `make generate` adds `modules/exposure`. `make validate`
  passes. The server boots with 52 buffs (8 new) and no new errors. The
  regression tests for findings 1 and 5 were checked by putting each bug
  back and watching them fail. Live telnet acceptance was not run.

### Review gate: Phases 13–14, retroactive (2026-09-23)

- **What:** The user asked whether features were tested and independently
  reviewed. They were unit-tested, but not reviewed, and wiring was
  under-tested. The workflow now has a testing and review gate (`CLAUDE.md`,
  `AGENTS.md`, plans README). Phases 13–14 went through it after the fact.
- **Wiring tests added:**
  - `AttackPlayerVsMob` in a dark versus a lit room, statistically
  - `look` per-viewer visibility, and `look <exit>` in daytime fog
  - personal, party, and companion light with real users, mobs, and
    parties
  - `LightConditions` on real rooms
  - the campfire fixture through the real `camp` flow, including a failed
    save
  - gametime `DayNumber` rollover
- **Review (independent reviewer subagent over `42bdcc4..0ac4b69a`):**
  1. *Critical, fixed:* `Hits` adds its modifier, and the darkness penalty
     was passed as a positive number, so darkness gave +40 to hit. The new
     combat wiring test caught it at the same time as the reviewer. It is now
     negated, like `dualWieldHitPenalty`.
  2. *Medium, fixed:* an `indoor`-tagged room in an outdoor biome (an inn in
     the woods) was pitch black at noon. It now counts as a lit interior
     unless its biome is dark.
  3. *Medium, fixed (docs):* `ScriptRoom.GetVisibility()` now means ambient
     light. The admin scripting docs and type description say so.
  4. *Low, fixed:* `look <exit>` let any lit biome see through exits even in
     fog. That bypass is removed, since per-viewer visibility already counts
     lit biomes.
  5. *Low, fixed:* the moon changed phase at midnight, mid-night. Gametime
     now exposes `DayNumber` (including the admin time offset), and
     `sky.NightOfDay` rolls the phase over at noon.
  6. *Low, partly fixed:* the campfire fixture query no longer takes the
     camping mutex. It reads a snapshot under its own RWMutex, refreshed
     after every camping lock section, so look and combat never wait on a
     camp save. *Deferred:* fires never burn out; fuel belongs with
     camping/inn work.
  7. *Low, deferred:* the cost of computing light per attack. It is fine at
     current scale; caching per room per round is a follow-up if fights grow.

  Also: `floatinglight` now ships in the empty world too, and the reserved
  room tags `indoor`/`outdoor`/`lit` are documented in
  `internal/rooms/AGENTS.md`.
- **Verification:** `go test -race ./...` passes (1846 passing test
  results). `make generate` and `make validate` pass. The server boots
  cleanly.

### Phase 14: visibility and light (2026-09-23)

- **What:** `internal/rooms/light.go` replaces upstream's room-wide
  `GetVisibility()` with an ambient model (`LightConditions` →
  `ambientLevel`):
  - by day: bright; fog dims but never blinds
  - at night: dim under any visible moon, pitch black when moonless or
    overcast; lit biomes +1; fog applies
  - indoors: lit biome = bright, otherwise dark
  - dark biome: dark
  - then mutators, then a fixture (+1)

  `VisibilityForUser`/`VisibilityForMob` add the viewer's own
  `lightsource`, or an ally's `partylight`, and treat `nightvision` as
  bright. Allies are the same leader (a user and their charmed companions)
  or the same GoMud party. `internal/combat` passes the attacker's
  `HitPenaltyForVisibility` into `calculateCombat` (default −40 dark,
  −10 dim; a lit target is at least dim). `look` uses per-viewer
  visibility. `modules/camping` registers lit campfires as light fixtures.
  The new `modules/light` owns the penalty config and the `light` command,
  and ships the `partylight` flag and the Floating Light buff (id 1000).
  A `floatinglight` spell (default world) casts it.
- **Why:** User decision 1 in the roadmap: carried light is personal, room
  fixtures light everyone, and a mage's light covers the party. The
  combat penalty makes darkness matter.
- **Durable / clock-safe:** No new persisted state. Ambient light is
  derived from the clock, persisted weather, data, and persisted camp
  records. No timers.
- **Deferred:** Wizard-only gating of `floatinglight` (Phase 17); race
  darkvision; stealth in darkness.
- **Verification:** `go test -race ./...` passes (1836 passing test
  results). `make generate` (adds `modules/light`) and `make validate`
  pass. Server boot loads the new flag, buff, and spell (24/44/10) without
  errors. Live telnet acceptance was not run.

### Phase 13: sky and environment (2026-09-23)

- **What:** New pure package `internal/sky` derives an 8-phase moon from
  the shared round counter (`AbsoluteDay` → `PhaseForDay`, cycle length
  `MoonCycleDays` in weather config, default 8). It also computes
  `Moonlight` (0–2, dimmed by broken cloud, hidden by overcast), which
  Phase 14 will use. `weather.Condition` gains validated `CloudCover` (0–3)
  and `VisibilityMod` (−2..0). The forest table gains `fog`/`thick-fog`,
  and its existing conditions get cloud cover values. `BiomeInfo.Indoor`
  marks cave/dungeon/house as indoors, and an `indoor`/`outdoor` room tag
  overrides it. `Room.SkyView()` gathers what a room can see: an indoor room
  sees the weather only through its first non-secret exit (in sorted order)
  to an outdoor room. `weather.RenderSky`/`SkyLines` render it purely.
  `look` shows weather or a glimpse, plus the moon at night. `weather` now
  reports time, cloud cover, weather, fog, and moon, or "You can't see the
  sky from in here."
- **Why:** The first slice of the user's environment request. It is
  display-only by design; mechanical effects belong to Phase 14
  (visibility) and Phase 15 (temperature).
- **Durable / clock-safe:** Nothing new is persisted. The moon is a pure
  function of the round count, and cloud/fog are config on the
  already-persisted condition name. Restart and copyover reproduce the same
  sky, and nothing advances the world clock.
- **Step completed:** Roadmap, Phase 13 design and plan, and all plan tasks.
- **Verification:** `go test -race ./...` passes (1824 passing test results
  across 81 packages, up from 1798 before Phase 13, a figure that may count
  differently). `make generate` and `make validate` pass. The server boots
  with the new biome and weather data without errors. Live telnet
  acceptance was not run.

### Phase 12c: combat encounters during travel (2026-09-22)

- **What:** A `TravelProfile`'s interruption can now be a real fight, not
  just flavor text. A new `Combat` `InterruptionKind` joins
  `fallen-tree`/`discovery`/`tracks`, usable as a profile's singular
  `Kind` or as one entry in a 12b weighted `Kinds` table. A new
  `InterruptionProfile.CombatMobID` names the mob template to spawn,
  required whenever `Combat` is reachable for that profile (`Validate`
  enforces it). On firing, `modules/expedition.interruptLocked` spawns
  that template into the travel session's origin room via a new
  `MobSpawner` seam (mirroring the existing `Mover`/`Survival`/
  `Scheduler` pattern; `nativeMobSpawner` uses `mobs.NewMobById` +
  `room.AddMob`, the same primitives `modules/company/runtime.go`'s
  `Spawn` already uses for a different purpose) and immediately commands
  it to attack the leader (`mob.Command("attack @<leader>")`, the same
  dispatch pattern the existing charmed-mob-assist loops already use) —
  deterministic, not dependent on any idle-tick AI timing. The spawned
  instance id is persisted onto `TravelInterruption.CombatMobInstanceId`.
  `travel resume` and `travel return` both now refuse
  ("You can't do that while you're still fighting!") while
  `MobSpawner.EncounterActive` reports that instance alive and still in
  the origin room, and both work normally once it's dead, gone, or was
  never tracked.
- **Why:** Two facts checked against the actual engine before designing
  this made it far simpler than it could have been: (1) a traveling
  company never physically leaves its origin room until arrival
  (`mover.MoveToRoom` is only called from `completeLocked`), so a combat
  encounter needs no new "encounter room" concept — the mob spawns where
  the party already is; (2) once `Aggro` is set on the spawned mob, the
  entire existing round-based combat loop — including every bit of Phase
  11's formation-combat gating, if the leader has a company — runs
  completely unchanged. No new combat logic was written; this pass is
  purely "spawn the right mob in the right place at the right time and
  gate two commands on whether it's still alive."
- **Scope, per this session's explicit direction ("just do combat
  encounters... keep the other ideas as future potential
  suggestions/improvements"):** only combat. Every other Phase 12
  overview encounter type (merchants, injured NPCs, route choices, camp
  opportunities, ruined sites, resources, social encounters) is
  recorded as a future idea in "Next" above, not planned work. Within
  combat itself: one mob, not a formation-shaped ambush party (11a's
  `mobparty.Groups` tag is a natural follow-up, not needed to satisfy
  the ask); no shipped route uses `Combat` yet (`oak-road` stays plain
  `fallen-tree`), matching 12a/12b's own deferral of new route content
  until there's a real reason to author it.
- **Fails open / stays durable:** a spawn failure (bad template id, room
  unavailable) still fires the interruption as a plain pause rather than
  aborting it — `CombatMobInstanceId` just stays `0`, and resume/return
  work immediately since nothing is tracked as blocking. Dynamically
  spawned mob instances (no `SpawnInfo` entry) do not themselves survive
  a server restart in this engine, so after a restart
  `EncounterActive` correctly reports false and a player is never stuck
  waiting on a fight that no longer exists.
- **Step completed:** Full design doc
  (`docs/superpowers/specs/2026-09-22-phase-12c-combat-encounters-design.md`),
  implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-12c-combat-encounters.md`),
  both code tasks, and this status update.
- **Key commits:** `4fd57422` (design + plan), `79b9231c` (domain: kind,
  `CombatMobID`/`CombatMobInstanceId`), `c43f8df0` (module: `MobSpawner`
  seam, spawn-on-fire, resume/return gate), landing on
  `phase-12c-combat-encounters`.
- **Verification:** `go test -race ./...` (1798 tests / 80 packages, up
  from 1786 — 7 new `internal/expedition` tests covering `Combat`
  validity/text/`CombatMobID` validation, plus 5 new `modules/expedition`
  tests covering spawn-on-fire, spawn-failure fail-open, and the
  resume/return gate in both active and resolved states, via a new
  `fakeMobSpawner` test double mirroring the module's existing fake-seam
  style), `make generate` (no wiring change), `make validate` pass.
- **Live acceptance:** Not run: this host has no interactive Telnet
  client.

### Phase 12b: weighted encounter tables (2026-09-22)

- **What:** A `TravelProfile`'s interruption can now be configured as a
  weighted table instead of exactly one kind. `InterruptionProfile` gains
  an alternate `Kinds []WeightedInterruptionKind{Kind, Weight}` form
  (`Kind` and `Kinds` are mutually exclusive — `Validate` rejects both set
  or both empty). A new pure `ResolveKind(roll uint64)` selects
  proportionally to weight (`roll % totalWeight`, cumulative-band lookup),
  or returns the plain `Kind` unchanged (ignoring `roll` entirely) for a
  singular-form profile — today's exact behavior, unchanged. The existing
  `TravelSession.Interrupt(now, p)` state-transition function is
  untouched: it still reads `p.Interruption.Kind` directly, so all 14 of
  its existing tests kept passing unmodified. Instead,
  `modules/expedition.interruptLocked` resolves the roll *before* calling
  `Interrupt` — when `profile.Interruption.Kinds` is set, it calls
  `ResolveKind(m.rollUint64())` and builds a resolved, singular-form
  copy of the profile to hand to the unchanged `Interrupt`. `rollUint64`
  is a new seam on `ExpeditionModule` (`func() uint64`, mirroring the
  existing `clock func() time.Time` seam), defaulting to `rand.Uint64` and
  overridable in tests.
- **Why:** 12a's design doc named this as its own next slice, deferred
  because it needed a random source and weight schema 12a didn't need to
  decide yet. Keeping the roll at the engine edge (same place `clock()`
  already lives) rather than threading it through `Interrupt` avoided
  rewriting 15 existing call sites (14 tests plus the one production call)
  for a change only weighted-table profiles need.
- **Scope:** Only `internal/expedition`'s `InterruptionProfile`/
  `TravelInterruption` (`TravelInterruption.Validate` could no longer
  delegate via a struct conversion to `InterruptionProfile` once the
  latter gained a third field, so it now checks `Kind`/`Checkpoint`
  directly — a fired interruption always carries the one already-resolved
  `Kind`, never a `Kinds` table) and `modules/expedition`'s
  `interruptLocked`. No shipped config changes: `oak-road` stays on plain
  `Kind: fallen-tree`. Adding real route content that uses a weighted
  table is deferred until there's a reason to author a new route (new
  zone/exit work) — proving the mechanism through tests is enough for now.
- **Fails closed / stays durable:** the roll happens exactly once, at fire
  time; only the resolved singular `Kind` is ever persisted into
  `TravelInterruption`, so a restart or copyover mid-interruption
  re-renders the same already-resolved choice exactly as before — it
  never re-rolls.
- **Step completed:** Full design doc
  (`docs/superpowers/specs/2026-09-22-phase-12b-weighted-encounters-design.md`),
  implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-12b-weighted-encounters.md`),
  both code tasks, and this status update.
- **Key commits:** `32ec8367` (design + plan), `3f5b624d`
  (implementation), landing on `phase-12b-weighted-encounters`.
- **Verification:** `go test -race ./...` (1786 tests / 80 packages, up
  from 1777 — 8 new `internal/expedition` tests covering `Validate`'s new
  branches and `ResolveKind`'s singular/weighted/invalid-profile cases,
  plus 1 new `modules/expedition` test proving `interruptLocked` fires
  the `rollUint64`-resolved kind), `make generate` (no wiring change),
  `make validate` pass.
- **Live acceptance:** Not run: this host has no interactive Telnet
  client.

### Phase 12a: encounter-kind abstraction (2026-09-22)

- **What:** Phase 12 ("Rich Expedition Encounters") begins. The handoff
  doc scopes it only as a bullet list of example encounter types and "This
  becomes a content system rather than engine plumbing" — genuinely
  underspecified, spanning several different subsystems (combat,
  merchants, dialogue, branching choices, camping). Rather than scope all
  of it at once (Phase 11's five-branch combat-wiring arc showed what
  happens when a phase isn't sliced narrowly), 12a takes the first,
  narrowest step: reusing Phase 6's existing travel-interruption
  mechanism (`internal/expedition.InterruptionKind`/`InterruptionProfile`,
  fire-once-at-a-checkpoint, resolved via `travel resume`/`travel
  return`) but widening its single hardcoded kind (`fallen-tree`) into a
  small, real set (`fallen-tree`, `discovery`, `tracks`), each with its
  own player-facing text via a new `InterruptionText(kind, profileName)`
  lookup in `internal/expedition` (moved out of
  `modules/expedition`'s single hardcoded sentence). No new session
  states, commands, or persistence shape — `InterruptionProfile` still
  configures exactly one `Kind` per route, unchanged.
- **Why:** Proves the "kind" is real, extensible content — not just an
  enum with one member wearing an interruption-shaped mechanism — before
  building anything that needs a new subsystem. `oak-road`'s shipped
  config is left on `fallen-tree`; adding new route content is better
  sequenced after 12b's weighted-table work so a route can be authored
  once against the final shape.
- **Scope:** Only `internal/expedition.InterruptionKind`/`Valid`/the new
  `InterruptionText`, and `modules/expedition`'s
  `interruptionTextLocked` (now a one-line call-through). Fully backward
  compatible: existing persisted `fallen-tree` sessions and the
  `oak-road` config decode identically.
- **Deferred (see the design doc's "Scope" section for full reasoning):**
  weighted/random encounter tables (a route rolling among several kinds
  instead of naming exactly one at config time — 12b); combat, merchant,
  injured-NPC, route-choice, camp-opportunity, ruined-site, resource, and
  social encounters (each needs its own subsystem: mob spawning + Phase
  11 combat, a shop flow, dialogue, branching choice UX, or
  `internal/camping` integration); non-travel (room/camp-based)
  encounters.
- **Step completed:** Full design doc
  (`docs/superpowers/specs/2026-09-22-phase-12a-encounter-kinds-design.md`),
  implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-12a-encounter-kinds.md`), both
  code tasks, and this status update.
- **Key commits:** `697dee6e` (design + plan), `a77180e4`
  (implementation), landing on `phase-12a-encounter-kinds`.
- **Verification:** `go test -race ./...` (1777 tests / 80 packages, up
  from 1773 — 4 new tests: `Valid()` accepts all three kinds, `Valid()`
  still rejects unknown kinds, `InterruptionText` returns distinct
  non-empty text per kind, `InterruptionText` falls back safely for an
  unknown kind), `make generate` (no wiring change), `make validate`
  pass. `modules/expedition`'s `interruptionTextLocked` has no dedicated
  test: it has no `sendToLeader`-capturing test harness (grepped and
  confirmed none exists), so `internal/expedition`'s own
  `InterruptionText` tests are the binding coverage, per the plan's
  documented contingency.
- **Live acceptance:** Not run: this host has no interactive Telnet
  client.

### Formation combat-loop wiring: leader-as-interceptor (complete — Phase 11's formation combat wiring is now entirely done, 2026-09-22)

- **What:** Closed the one deliberate gap the mob-vs-mob pass left named:
  when a hostile mob attacks a company member and 11c's front-row
  interception would redirect that attack to the company's own leader
  (posted in the front row, same column), the redirect now actually
  happens. Previously `resolveAttackTargetCompanionOnly` refused this one
  specific redirect, because it crosses `combat.Attack*` functions
  mid-resolution (`AttackMobVsMob` to `AttackMobVsPlayer`) and only the
  opposite crossing existed (`gateMobVsPlayerAttack`/
  `resolveInterceptedMobAttack`, from the mob-vs-player pass). This pass
  builds the missing symmetric case: a new `resolveInterceptedAttackOnLeader`
  resolves the attack itself via `AttackMobVsPlayer` (mirroring
  `NewRound_DoCombat.go`'s existing mob-vs-player resolution block: buffs,
  messages, the charmed-mob-assist loop, offhand equipment-break, plus a
  `CharacterVitalsChanged` event so the leader's own client health bar
  updates). `gateMobVsMobAttack` and `gateEnemyAttacksCompanion` now
  return a three-state `(target, handled, ok)` contract mirroring
  `gateMobVsPlayerAttack`'s; `handled=true` tells the caller to skip its
  own `AttackMobVsMob` call and `continue` the round.
  `resolveAttackTargetCompanionOnly` is deleted — `gateEnemyAttacksCompanion`
  now reuses the same `resolveAttackTarget` every other direction already
  uses, checking after the fact whether the resolved key is
  `company.LeaderMemberKey`.
- **Why:** This was the one Phase 11 combat direction where formation
  tactics behaved asymmetrically: a front-row companion could already
  intercept for the leader, but a front-row leader couldn't return the
  favor for a companion. Closing it makes "post your leader (or your
  tankiest companion) in front" a formation choice that pays off
  consistently regardless of which side started the attack.
- **Scope:** Only `gateEnemyAttacksCompanion`'s redirect target changes.
  Every other direction (player-vs-mob, companion-vs-enemy,
  mob-vs-player, companion-vs-companion interception) is untouched;
  `mob.Character.Aggro` is still never touched by the redirect, matching
  `resolveInterceptedMobAttack`'s existing invariant — the interception
  is recomputed fresh every round.
- **Fails closed to today's exact behavior:** no company formation, no
  resolvable attacker column, or a legality check that already fails all
  fall through unchanged; only the new "interceptor resolves to the
  leader" branch produces new behavior.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11-leader-interceptor.md`),
  the gate/resolver rewrite, and this status update.
- **Key commits:** `eae9ac5f` (plan), `12c8a73d` (implementation), landing
  on `phase-11-leader-interceptor`.
- **Verification:** `go test -race ./...` (1773 tests / 80 packages — net
  -1 from the prior count: two obsolete
  `resolveAttackTargetCompanionOnly` tests removed, one new
  `resolveAttackTarget`-reaches-the-leader test added), `make generate`
  (no wiring change), and `make validate` pass. Focused test coverage:
  the pure `resolveAttackTarget` redirect-to-leader case; the two engine
  adapters (`gateEnemyAttacksCompanion`, `resolveInterceptedAttackOnLeader`)
  stay untested at that layer, same precedent as every prior
  combat-wiring pass.
- **Live acceptance:** Not run — same `internal/hooks` integration-harness
  limitation as every prior combat-wiring pass.

### Docs correction: proactive engagement trigger already existed (2026-09-22)

- **What:** The last four combat-wiring work-log entries and the phase
  table listed 11b's "proactive engagement trigger" (an idle company
  companion joining a fight because their leader started one, without
  being personally hit first) as deferred/unbuilt. It was not. Before
  scoping new work here, `internal/usercommands/attack.go`'s existing
  mob-target and player-target attack branches were re-read: right after
  `SetAggro`, both loop `room.GetMobs(rooms.FindCharmed)` and, for every
  idle (`Aggro == nil`) mob charmed by the attacking player, issue
  `m.Command("attack #<id>")` / `m.Command("attack @<id>")` — a
  same-round proactive join, not the reactive retaliate-once-hit pattern
  in `NewRound_DoCombat.go`. Company companions are spawned as
  permanently-charmed mobs (`modules/company/runtime.go`'s `Spawn`:
  `mob.Character.Charm(leaderUserID, -2, characters.CharmExpiredRevert)`),
  so `IsCharmed(leaderUserID)` is true for them and this pre-existing
  loop already covers them with no gap.
- **Why:** Formation legality still applies downstream regardless of how
  `Aggro` got set — a proactively-joining companion's subsequent attack
  still runs through `NewRound_DoCombat.go`'s normal per-round dispatch,
  so it hits `gateCompanionAttacksEnemy` exactly like a reactive one
  would. There was nothing left to wire; only the status doc was wrong.
- **Scope:** Documentation only. No production code changed. Verified by
  reading `attack.go` and `modules/company/runtime.go`, not by adding a
  new test — `attack.go` has no existing test file and is the same kind
  of engine-entangled adapter code this session has consistently left
  untested at that layer (see every prior combat-wiring pass's
  "Verification" note).
- **Step completed:** This status-doc correction, on its own worktree
  and branch per repo convention (docs-only changes still need one).
- **Key commit:** landing on `phase-11-docs-correction`.
- **Verification:** No code changed; `go test -race ./...` still passes
  at the same 1774 tests / 80 packages as the prior entry (re-run to
  confirm nothing regressed while investigating).

### Formation combat-loop wiring: 11b reassignment-on-death (complete — Phase 11's foundational combat wiring is now fully done, 2026-09-22)

- **What:** Wired 11b's last unwired acceptance criterion. When a company
  member's mob target dies (or otherwise becomes permanently invalid),
  they now pick a new legal target within the same hostile party via
  `engagement.AssignTarget`'s weakest-HP preference, instead of simply
  clearing `Aggro` and giving up. `reassignEnemyTarget` (new, in
  `internal/hooks/combat_formation.go`) reassembles whichever hostile
  party is currently in the room fresh (`mobparty.Assemble` is never
  cached, matching 11a's own design — once a target mob is gone there's
  no way to ask "which party was it in," so this reassembles from scratch
  and picks the first resulting party), builds `engagement.Combatant`
  candidates from its live members (`partyCombatants`, new), and applies
  11c's `Legal` predicate as the injected legality filter — the very
  `legal` parameter 11b's spec described as coming from 11c, now finally
  wired end to end. Two thin engine wrappers, `reassignPlayerTarget` and
  `reassignCompanionTarget` (the latter only firing when the attacker is a
  currently-attached company member — a hostile mob's own dead target is
  enemy AI, out of scope), are called from the four existing dead-target
  checks in `NewRound_DoCombat.go` (two in the player-vs-mob branch, two
  in the mob-vs-mob branch) right before each one's pre-existing "give up"
  path. On success, `Aggro` points at the new target and the round
  `continue`s — combat resumes normally next round, already gated by the
  three interception passes like any other round.
- **Why:** This is the piece all three interception passes explicitly
  deferred: it required modifying the existing target-validation block
  that clears `Aggro` and skips the round *before* any reassignment could
  run, a materially different (and until now, deliberately postponed)
  kind of change from adding a new gate purely before an attack call.
- **Scope: only the two "a company member attacks an enemy" directions.**
  11b's spec frames reassignment as a company member's own engagement, not
  sharpening enemy AI — so player-vs-player (explicit PvP, always out of
  scope) and mob-vs-player (the target there is the leader, not an enemy;
  nothing to reassign among) are untouched. This mirrors exactly how the
  three interception passes scoped themselves.
- **Fails closed to today's exact behavior:** no company formation, no
  hostile party currently in the room, or no living legal candidate within
  it, all mean the existing "target lost"/"rage subsides" message plus
  `Aggro = nil` plus `continue` runs completely unchanged — nothing about
  this pass can make a previously-working "give up" path behave
  differently when reassignment genuinely isn't possible.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11-reassignment-on-death.md`),
  both code tasks (the reassignment helpers, the four-site
  `NewRound_DoCombat.go` wiring), plus this status update.
- **Key commits:** `bb58571f` (plan), `83433972` (reassignment helpers and
  tests), `20d6893c` (the four `NewRound_DoCombat.go` call sites), landing
  on `phase-11-reassignment`.
- **Verification:** `go test -race ./...` (1774 tests / 80 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  test coverage: `partyCombatants`' formation-position/alive-HP contract
  (the one genuinely pure sub-piece — `reassignEnemyTarget` itself and the
  two engine wrappers are untested directly, same precedent every prior
  combat-wiring pass set for engine-entangled adapter glue, since
  `engagement.AssignTarget`'s own decision logic is already fully covered
  by its own package's tests from Phase 11b).
- **Live acceptance:** Not run: this host has no interactive Telnet
  client, and `internal/hooks` has no integration-test harness (same
  limitation as all three prior combat-wiring passes).
- **Deferred:** the leader-as-interceptor gap named here is now closed —
  see the "leader-as-interceptor" work-log entry above. What remains is
  Phase 11d (guard reactions, crit effects, wounds, full AI targeting
  personality — unscheduled from the start), and everything already
  deferred by 11a-11c themselves (guard-stance/chance-based interception,
  row/column AoE, formation buffs, flanking/exposure, movement-in-combat,
  `internal/parties`/PvP interaction). The "engagement trigger" item
  listed here in earlier work-log entries was a documentation error —
  see the docs-correction work-log entry above: `attack.go`'s existing
  charmed-mob-assist loop already makes idle company companions join the
  instant their leader attacks, since companions are permanently-charmed
  mobs. No code change was needed.

### Formation combat-loop wiring: mob-vs-mob (complete; formation combat's attack directions now fully wired, 2026-09-22)

- **What:** Completed the third of the three combat directions deferred by
  the original combat-wiring pass. When a companion and a hostile mob
  fight — either direction — 11c's legality/interception now gates and
  redirects the attack exactly as it already does for player-vs-mob and
  mob-vs-player. `gateMobVsMobAttack` first classifies both sides of the
  fight via a new seam query, `company.LeaderAndKeyForInstance(instanceId)
  (leaderUserID, key, found)` (wraps `modules/company`'s existing private
  `companionForInstance` reverse lookup, already used internally for
  `onMobDeath`, now exposed for combat), then dispatches: "my companion
  attacks the enemy" reuses `gateFormationAttack`'s exact core logic
  (extracted into a new shared `resolveEnemyAttack` — pure refactor, no
  behavior change, confirmed by the full existing test suite passing
  unchanged); "the enemy attacks my companion" generalizes
  `gateMobVsPlayerAttack`'s legality/interception half to any company
  member, not just the leader, via a new
  `resolveAttackTargetCompanionOnly` variant. Two hostile mobs, or two
  companions, fighting each other is left completely untouched — neither
  side has a formation concept to apply. Unlike the mob-vs-player pass,
  neither new gate needed a cross-`Attack*`-type redirect: a companion
  attacking a *different* enemy, or an enemy attacking a *different*
  companion, both stay `combat.AttackMobVsMob` throughout, so both gates
  reuse the simpler "substitute defender, or skip" shape instead of
  `gateMobVsPlayerAttack`'s "resolve the attack myself" shape.
- **Why:** Same reasoning as the previous two passes: connect
  already-shipped, already-tested pure domain packages into real combat,
  one well-scoped direction at a time.
- **One explicit, documented gap: a leader posted in front of their own
  companions can't intercept for them via this pass.** A company's
  `Formation` can place the leader anywhere, including the front row, so
  an attack aimed at a companion could in principle intercept to the
  leader specifically — which would need the exact same
  `AttackMobVsMob`-to-`AttackMobVsPlayer` cross-type redirect the
  mob-vs-player pass solved for a different direction, a third time.
  `resolveAttackTargetCompanionOnly` deliberately skips that one specific
  redirect (falling back to checking legality against the original
  companion target directly, exactly as if no interceptor existed — never
  breaking anything, just not claiming that one interception opportunity).
  This is a narrow, named boundary for an unusual formation arrangement,
  not a silently dropped case.
- **Fails open**, same contract as the previous two passes: no company
  record, attacker/defender not resolvable into a company or enemy party,
  or the leader-as-interceptor case above, all mean
  `combat.AttackMobVsMob(mob, defMob)` runs exactly as it did before this
  change.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11-mob-vs-mob.md`), all three
  code tasks (`LeaderAndKeyForInstance` seam, the two new gates plus the
  `gateFormationAttack` refactor, the one-line `NewRound_DoCombat.go`
  wiring), plus this status update.
- **Key commits:** `ca7d44f9` (plan), `27d43e30` (`LeaderAndKeyForInstance`
  seam), `67118692` (mob-vs-mob gates and refactor), `ac997bad` (the
  `NewRound_DoCombat.go` gate call), landing on `phase-11-mob-vs-mob`.
- **Verification:** `go test -race ./...` (1773 tests / 80 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  tests cover the seam's call-through/no-provider case for
  `LeaderAndKeyForInstance` (mirroring the existing `FormationFor`/
  `InstanceFor` tests) and `resolveAttackTargetCompanionOnly`'s two cases
  (skips redirecting to the leader; still intercepts normally between two
  companions). The `gateFormationAttack` refactor is verified by the full
  existing `internal/hooks` suite passing unchanged — it has no direct
  unit test of its own, same precedent as the previous two passes' engine
  adapters.
- **Live acceptance:** Not run: this host has no interactive Telnet
  client, and `internal/hooks` has no integration-test harness (same
  limitation as the previous two passes).
- **Deferred:** 11b's `engagement.AssignTarget` reassignment-on-death
  (needs modifying the existing target-validation block, a materially more
  invasive change than any interception pass made), the
  leader-as-interceptor edge case above, Phase 11d (unscheduled), and
  everything already deferred by 11a-11c themselves (guard-stance/
  chance-based interception, row/column AoE, formation buffs,
  flanking/exposure, movement-in-combat, `internal/parties`/PvP
  interaction).

### Formation combat-loop wiring: mob-vs-player interception (complete; two directions still deferred, 2026-09-22)

- **What:** Completed the second of the three combat directions deferred
  by the player-vs-mob pass. When a hostile mob attacks a player leader,
  11c's front-row interception now redirects the hit to a living, legal
  front-row companion — a real "my tank protects me" — and
  column-occupancy/lateral-range legality gates the attack exactly as it
  already does for player-vs-mob. A redirect here needs a genuinely
  different resolution path than the player-vs-mob case: the un-redirected
  attack is `combat.AttackMobVsPlayer(mob, defUser)`, but a redirect must
  instead call `combat.AttackMobVsMob(mob, interceptor)` — a different
  `Attack*` function with different message/equipment-break/scripting side
  effects (mirroring the existing mob-vs-mob call site's own post-attack
  handling, since a mob defender has no `SendText`). `gateMobVsPlayerAttack`
  therefore resolves an intercepted attack itself
  (`resolveInterceptedMobAttack`) and tells the caller "already handled,
  skip your own resolution" rather than handing back a substitute target
  the way `gateFormationAttack` does. `company.FormationProvider` gained a
  second query, `InstanceFor(leaderUserID, companionID) (instanceId int, ok bool)`
  (delegating to `modules/company`'s existing private `instance()` lookup,
  not duplicating it), and `internal/company` gained
  `CompanionIDFromMemberKey`, the reverse of the existing
  `CompanionMemberKey` — both needed to turn `InterceptFrontRow`'s returned
  `MemberKey` into a live `*mobs.Mob` to actually attack.
- **Why:** Same reasoning as the player-vs-mob pass: connect already-shipped,
  already-tested pure domain packages (11a/11c) into real combat, one
  well-scoped direction at a time rather than one large risky rewrite of
  the whole combat loop.
- **`Aggro` never retargets to the interceptor:** after an intercepted
  round, `mob.Character.Aggro` stays pointed at the leader's `UserId`,
  unchanged. Interception is recomputed fresh every round from current
  formation/alive state — not a persistent retarget — so if the
  interceptor later dies, the very next round's gate call finds no living
  front-row companion and the attack resolves directly against the leader
  again automatically, with zero `Aggro` bookkeeping needed by anyone.
  This is the same self-healing guarantee the player-vs-mob pass already
  provides, applied to the defending side instead of the attacking side.
- **The existing "leader is attacked → idle companions retaliate" loop
  also fires on an intercepted round:** `resolveInterceptedMobAttack`
  replicates the identical pre-existing loop
  (`NewRound_DoCombat.go:880-892`, unchanged) that sets every one of the
  leader's idle charmed mobs attacking back — a leader defended by an
  interceptor still rallies the rest of an idle company exactly as it
  would if the hit had landed on the leader directly.
- **Fails open**, same contract as the player-vs-mob pass: no company
  record for the defending leader, or the attacking mob can't be resolved
  into an assembled enemy party, means `combat.AttackMobVsPlayer(mob,
  defUser)` runs exactly as it did before this change.
- **`mob.Reach` (11c's innate-reach schema field, shipped but previously
  unread by any real call site) is exercised for the first time here** via
  `combat.ResolveReach(&mob.Character, mob.Reach)` — a hostile mob's own
  spec-level `Reach: true` now actually extends its melee reach in combat.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11-mob-vs-player-interception.md`),
  all three code tasks (`InstanceFor` seam + `CompanionIDFromMemberKey`,
  the gate and intercepted-attack resolver, the one-line
  `NewRound_DoCombat.go` wiring), plus this status update.
- **Key commits:** `7b53d3f9` (plan), `3462966f` (`InstanceFor` seam and
  `CompanionIDFromMemberKey`), `47705212`
  (`internal/hooks/combat_formation.go` additions), `718759fb` (the
  `NewRound_DoCombat.go` gate call), landing on
  `phase-11-mob-vs-player-interception`.
- **Verification:** `go test -race ./...` (1769 tests / 80 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  tests cover the seam's call-through/no-provider cases (`InstanceFor`,
  mirroring `FormationFor`'s existing tests) and
  `CompanionIDFromMemberKey`'s round trip and rejection of non-companion
  keys; the new decision logic in this pass reuses the already-tested
  `resolveAttackTarget` from the previous pass unchanged, and the new
  engine-facing adapter code (`gateMobVsPlayerAttack`,
  `resolveInterceptedMobAttack`) has no direct unit tests, matching the
  precedent `gateFormationAttack` already set — `internal/hooks` has no
  test harness for engine-entangled glue, only for pure decision logic.
- **Live acceptance:** Not run: this host has no interactive Telnet
  client, and `internal/hooks` has no integration-test harness (same
  limitation as the player-vs-mob pass).
- **Deferred:** mob-vs-mob (needs to disambiguate charmed/companion vs.
  hostile mobs on both sides of every call), 11b's
  `engagement.AssignTarget` reassignment-on-death (needs modifying the
  existing target-validation block, a materially more invasive change than
  either interception pass made), Phase 11d (unscheduled), and everything
  already deferred by 11a-11c themselves (guard-stance/chance-based
  interception, row/column AoE, formation buffs, flanking/exposure,
  movement-in-combat, `internal/parties`/PvP interaction).

### Formation combat-loop wiring: player-vs-mob (complete; three directions still deferred, 2026-09-22)

- **What:** Connected 11a/11b/11c into real combat for the single most
  central interaction: a player fighting a hostile enemy party. A new
  `company.FormationProvider` query seam (`internal/company/provider.go`,
  the same `Set<X>`/`<X>For` shape as `survival.CompanyService` and
  `weather.Provider`, registered from `modules/company`'s existing
  `init()`) lets `internal/hooks` read a leader's current `Formation`
  without importing `modules/company` (which it structurally can't —
  modules depend on `internal/`, never the reverse). `internal/hooks/combat_formation.go`
  adds `resolveAttackTarget` (pure: given a `Formation`, an `alive` map, an
  attacker's column, the original target key, and a `Reach`, decide the
  final legal target or that the attack should be skipped — fully unit
  tested with hand-built data, no engine state involved) and
  `gateFormationAttack` (the thin adapter: resolves the enemy party fresh
  each round via 11a's `mobparty.Assemble` over the room's hostile mobs,
  computes each one's `EHP` directly from its live `HealthMax`/`Defense`
  rather than calling `combat.RankMobs()` — which ranks every mob spec in
  the game and would be far too expensive to call every combat round —
  and calls `combat.ResolveReach` for the player's equipped weapon).
  `internal/hooks/NewRound_DoCombat.go` gets exactly one new gate call, at
  the existing player-vs-mob attack site (`:556`, right after the existing
  "can't see a hidden target" check and before the attack itself), which
  reassigns the local `defMob` variable to the actual (possibly
  intercepted) target or `continue`s the round if the attack is currently
  illegal — every later reference in that branch (the attack call itself,
  buff/message dispatch, hostility/aggro-acquisition logic) automatically
  picks up the final target with no further edits. `mobparty`'s previously
  internal `memberKey` helper is now exported as `MemberKeyFor`, plus a new
  `InstanceIdFromMemberKey` for the reverse direction, so `internal/hooks`
  can translate between a live mob instance ID and its formation key.
- **Why:** 11a, 11b, and 11c each shipped a fully tested pure domain layer
  with nothing actually reading it in a real fight. This is the join
  between them — not a numbered phase of its own, since it invents no new
  gameplay rules, only wires already-designed ones together.
- **Fails open by design:** if a player has no company record at all (a
  solo player, or one who's never summoned a companion),
  `company.FormationFor` returns `ok=false` and `gateFormationAttack`
  returns the original target unchanged — combat resolves exactly as it
  did before this change. This is deliberate: the feature is "company vs.
  party" tactics, and it should be zero-behavior-impact for every player
  who has never touched the company system, which is the overwhelming
  majority of existing playtesting to date.
- **Deliberately NOT done this pass (see
  `docs/superpowers/plans/2026-09-22-phase-11-combat-wiring.md`'s "Design
  decisions" section for the full reasoning on each):**
  - **Mob-vs-player interception** — redirecting an attack aimed at the
    leader to an intercepting companion means switching from
    `combat.AttackMobVsPlayer` to `combat.AttackMobVsMob`, a different
    `Attack*` function entirely, not just a different target. That's a
    materially different (and equally real) piece of work from anything in
    this pass, where every redirect has stayed mob-to-mob.
  - **Mob-vs-mob** (both "my companion attacks the enemy" and "the enemy
    attacks my companion") — needs to disambiguate charmed/companion vs.
    hostile mobs on both sides of the same call site, for every mob in the
    room, every round.
  - **11b's `engagement.AssignTarget` reassignment-on-death** — the
    existing target-validation block
    (`internal/hooks/NewRound_DoCombat.go:486-526`) already clears `Aggro`
    and `continue`s the round the instant a target is found dead or gone,
    *before* any reassignment attempt could run. Making reassignment work
    means modifying that existing, working, unmodified-since-launch logic
    — genuinely riskier than the purely additive pre-attack gate this pass
    added, which never touches Aggro-clearing at all.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11-combat-wiring.md`), all four
  code tasks (`FormationProvider` seam, exported `mobparty` key helpers,
  the pure/adapter gate, the one-line `NewRound_DoCombat.go` wiring), plus
  this status update.
- **Key commits:** `d4b9a4cd` (plan), `3632b4f7` (`FormationProvider`
  seam), `1b0054ce` (exported `mobparty` key helpers), `fa9c6931`
  (`internal/hooks/combat_formation.go` and its tests), `0504672d` (the
  `NewRound_DoCombat.go` gate call), landing on `phase-11-combat-wiring`.
- **Verification:** `go test -race ./...` (1765 tests / 80 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  tests cover: `FormationProvider`'s call-through and no-provider-registered
  cases, the `mobparty` key-helper round trip and rejection of non-mob
  keys, and `resolveAttackTarget`'s direct-hit/interception-redirect/
  blocked-with-no-rescue/missing-target/out-of-lateral-range cases plus
  `effectiveHP`'s formula (matching `combat.RankMobs`' own EHP math,
  including the 95%-mitigation clamp).
- **Live acceptance:** Not run: this host has no interactive Telnet
  client, and `internal/hooks` has no pre-existing integration-test harness
  to build on (confirmed zero test files in that package before this
  change) — `gateFormationAttack`'s engine-facing lookups
  (`mobs.GetInstance`, `rooms.LoadRoom`-backed `Room.GetMobs`,
  `company.FormationFor`) are exercised only indirectly, through the pure
  `resolveAttackTarget` core and the seam's own unit tests, not a live
  combat round.
- **Deferred:** the three directions above, Phase 11d (unscheduled),
  guard-stance/chance-based interception, row/column AoE, formation buffs,
  flanking/exposure, movement-in-combat, and any interaction with
  `internal/parties` or PvP (all already deferred by 11a-11c themselves).

### Phase 11c — Formation Tactics domain layer, schema, and read-only query (complete; combat-loop wiring deferred, 2026-09-22)

- **What:** Delivered the column-occupancy reach model as pure, tested
  logic, plus the `Reach` trait it needs and a first real (if narrow)
  player-facing use of it. `internal/formationcombat` (GoMud-free, matching
  11a/11b's split) implements: `FrontmostOccupant` (the nearest-to-front
  living occupant of a formation column), `InLateralRange` (±1 column),
  `InReachDepth`/`Reach{None,Extended,Any}` (plain melee = column
  frontmost only; polearm/innate Reach = frontmost-or-one-behind; ranged =
  any depth), `Legal` (combining all of the above into the single
  predicate Phase 11b's `AssignTarget` is designed to consume as its
  injected `legal` parameter), `InterceptFrontRow` (redirect a non-front-row
  attack to a living same-column front-row member, or leave the original
  target standing if none survive), and `LegalTargets` (the read-only
  adjacency-query list the design's acceptance criteria call for). Every
  function takes a `company.Formation` plus a caller-supplied `alive
  map[MemberKey]bool` — nothing here ever mutates a `Formation` or clears a
  target, so the design's "self-healing without clearing Aggro" behavior
  falls directly out of `Legal` simply being recomputed fresh from current
  `alive` state each time it's called. `items.ItemSpec` and `mobs.Mob` each
  gained one new `Reach bool` field (default `false`, zero-value-safe for
  every existing YAML file, following the exact `Weight`/`Hostile`
  precedent), and `internal/combat.ResolveReach(c *characters.Character,
  innateReach bool) formationcombat.Reach` adapts a live combatant's
  equipped weapon (or, for mobs, their innate flag) into the `Reach` value
  those pure functions need. `modules/company/formation.go` gained a
  `formation reach <member>` read-only subcommand — a real, always-available
  demo of `Legal`/`LegalTargets` against the company's own formation
  (there's no live enemy party to query outside combat, and this phase
  doesn't touch combat at all), listing which of a member's own
  companions/leader they could plain-melee-reach right now.
- **Why:** The Phase 11 design session's original five first-tactical-rules
  (handoff §36/§15 — interception, melee reach, polearm reach, ranged
  rear-line access, adjacency) needed a precisely-specified reach model
  before any of it could be implemented; the design session spent several
  rounds settling on column-occupancy (not row-distance) as the only model
  consistent with "formation is locked during combat, only enemy attrition
  changes what's reachable." This phase turns that resolved design directly
  into tested code, and — as importantly — unblocks Phase 11b, whose
  `AssignTarget` has been shipping since 11b's own phase entry with a test
  stub in place of a real `legal` predicate.
- **Deliberate scope boundary — `NewRound_DoCombat.go` wiring is NOT done
  this phase:** wiring `Legal`/`InterceptFrontRow` into the real per-round
  attack loop (and, with it, finally supplying 11b's `AssignTarget` a real
  `legal` function instead of a test stub) needs a lookup this codebase
  doesn't have: given a live mob instance ID, which leader's company does
  it belong to and what's its `MemberKey`. That mapping exists today only
  as `modules/company.CompanyModule`'s **unexported** `companionForInstance`
  method, and `internal/hooks` (home of `NewRound_DoCombat.go`) doesn't and
  structurally shouldn't import `modules/company` — the established pattern
  for this kind of cross-boundary query in this codebase is a query-seam
  interface defined in `internal/`, implemented by the module (Phase 5's
  `survival.CompanyService`, Phase 8's `weather.Provider`). Building that
  seam, plus threading it and a fresh-per-round `mobparty.Assemble` lookup
  for the enemy side through all four attack-direction call sites of a
  1150-line hot combat-loop file, is a materially different, higher-risk
  kind of change than the pure functions this phase delivers — a real,
  player-visible combat-behavior change touching a shared multiplayer
  invariant, exactly what `internal/combat/AGENTS.md` and
  `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` §50 flag for escalated care.
  Shipping it in the same pass would make both harder to review and roll
  back independently, so it's tracked as an explicit follow-up (see
  "Next" above), not silently dropped. This mirrors 11b's own precedent of
  shipping a tested domain layer and naming exactly what blocks the wiring.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11c-formation-tactics.md`),
  Tasks 1-3 (domain package, `Reach` schema + adapter, read-only query
  command), plus this status update. There is no task covering
  `NewRound_DoCombat.go` wiring in this plan — see the scope-boundary note
  above.
- **Key commits:** `765278dd` (plan), `541b85b1` (`internal/formationcombat`
  domain package and tests), `1eed92f6` (`Reach` schema fields and
  `ResolveReach` adapter), `4d3197c8` (`formation reach` command), landing
  on `phase-11c-formation-tactics`.
- **Verification:** `go test -race ./...` (1755 tests / 80 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  tests cover: the spec's own worked example (A/B/C) before and after A
  dies, reach-extended hitting the middle but not the back slot, ranged
  ignoring column depth entirely, interception redirect (and "no living
  front row, original stands"), all nine lateral-range column
  combinations, dead/missing-target rejection, `LegalTargets`'
  adjacency-query correctness, and `ResolveReach`'s four weapon/innate
  cases. `modules/company`'s existing 70 tests all still pass unchanged.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The `formation reach` command is exercised only by the deterministic
  `modules/company` unit suite and the underlying `formationcombat`/
  `combat` domain tests, not a live session.
- **Deferred:** the `NewRound_DoCombat.go` combat-loop wiring itself (see
  above — this is the big one), the equivalent wiring of 11b's
  `AssignTarget` (lands together with the above), Phase 11d (guard
  reactions, crit effects, wounds, full AI personality, unscheduled),
  guard-stance/chance-based interception, row/column AoE, formation buffs,
  flanking/exposure, movement-in-combat, and any interaction with
  `internal/parties` or PvP.

### Phase 11b — Unit-vs-Unit Engagement domain layer (complete; hooks integration deferred, 2026-09-22)

- **What:** Delivered the target-assignment building block for
  company-vs-party combat. `internal/engagement` is a GoMud-free domain
  package (matching `internal/mobparty`'s split): `Combatant{ID, HP, Row,
  Col}` is the minimal per-participant view, `Preference` (`Weakest`,
  `Strongest`, `Random`) selects among candidates, and
  `AssignTarget(attacker, candidates, pref, legal) (targetID int, ok bool)`
  filters candidates to those alive (`HP > 0`) and legal (per an injected
  `LegalFunc`), then applies the preference. `AssignTarget` is stateless —
  the same call that assigns an initial target also handles reassignment
  after a target's death, since a dead candidate is simply filtered out of
  the next call's `candidates` slice; no separate "reassign" path exists or
  is needed. `PartyAlive(candidates) bool` is the companion check for
  engagement-end: once every candidate's `HP <= 0`, the (future) hooks
  integration stops calling `AssignTarget` for that party. `Engagement{
  LeaderUserID, PartyID}` is a thin, unpersisted marker type — per the
  shared Phase 11 prior-art finding that `characters.Aggro` itself is not
  persisted and is not being rearchitected into a list; this phase adds a
  coordinated *initiation/reassignment* routine on top of individual
  `Aggro`, it doesn't replace it.
- **Why:** The Phase 11 design session identified that a companion today
  only retaliates reactively if it personally gets attacked, with no
  coordinated company-wide targeting. 11b's domain layer is the reusable
  "who should X attack" decision function that a future combat-loop
  integration calls once per under-targeted company member per round.
- **Deliberate scope boundary — hooks integration NOT done this phase:**
  the spec (`docs/superpowers/specs/2026-09-22-phase-11b-unit-engagement-design.md`,
  "Constraints and Deferrals") states explicitly that `AssignTarget`'s
  `legal` parameter must be satisfiable by Phase 11c's real
  lateral-range/reach predicate before the `internal/hooks/NewRound_DoCombat.go`
  wiring "can land for real," and that this wiring is "the last task, done
  together with or after 11c lands." 11c is designed but not yet planned or
  implemented. Wiring real combat behavior against a throwaway `legal` stub
  now would produce something that has to be rewired the moment 11c ships —
  worse than leaving it unwired. This phase therefore ships the tested,
  reusable `AssignTarget`/`PartyAlive` building block only; nothing in
  `internal/hooks` was touched, and no player-visible combat behavior
  changed. The follow-up wiring task is tracked here, not forgotten.
- **Spec deviation, deliberate:** the design doc's illustrative signature
  passes a `mobparty.Party` into `AssignTarget`; `Party` (11a) has no HP
  field, so weakest/strongest selection needs the caller to already have
  adapted each member's live HP into a `Combatant` first. `AssignTarget`
  therefore takes `[]Combatant` instead — `internal/engagement` doesn't
  import `internal/mobparty` at all. The eventual hooks integration is
  where `Party.Members` (instance IDs) plus live HP/formation position get
  adapted into `Combatant` values.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11b-unit-engagement.md`), Task
  1 (domain package and tests), plus this status update. There is no Task
  covering hooks wiring in this plan — see the scope-boundary note above.
- **Key commits:** `c7ede915` (plan), `746f28ef` (`internal/engagement`
  domain package and tests), landing on `phase-11b-unit-engagement`.
- **Verification:** `go test -race ./...` (1735 tests / 79 packages),
  `make generate` (no wiring change), and `make validate` pass. Focused
  tests cover: weakest/strongest/random selection, filtering by the
  injected legality predicate, skipping dead candidates (the reassignment
  case), no-legal-target and empty-candidate handling (returns `ok=false`,
  never panics), and `PartyAlive`'s true/false/empty cases.
- **Live acceptance:** Not run: this is a pure domain package with no
  engine wiring yet — there is no player-visible behavior to exercise.
- **Deferred:** the `internal/hooks/NewRound_DoCombat.go` integration
  itself (blocked on 11c's legality predicate, per the spec's own
  sequencing — see above), Phase 11c (formation tactics), Phase 11d (guard
  reactions, crit effects, wounds, full AI personality, all unscheduled),
  real per-member preference configuration (this phase ships one
  project-wide default rule, not player-configurable UI, matching the
  spec's own scoping), and any interaction with `internal/parties` or PvP.

### Phase 11a — Enemy Parties (complete, 2026-09-22)

- **What:** Delivered a `Party` concept for mobs, mirroring the player
  company. `internal/mobparty` is a GoMud-free domain package (matching
  `internal/expedition`/`internal/camping`/`internal/weather`'s split): a
  `MobSummary{InstanceId, Groups, EHP, DPS}` input and a pure
  `Assemble(mobs []MobSummary) []Party` that groups mobs by their first
  `Groups` tag (an untagged mob is always its own solo party — never merged
  with another untagged mob), splits any group over 5 members into
  multiple parties in input order, and auto-fills each party's
  `company.Formation` (the existing 3×3 grid type, reused directly, not
  reimplemented) front-to-back by descending `EHP`; a solo party occupies
  the front-row center cell. `internal/rooms/roomdetails.go`'s existing
  per-mob room-listing loop now buckets hostile mobs (friendly/charmed mobs
  are untouched) into `hostileMobDisplay` values and renders them through a
  new `internal/rooms/mobparty_display.go`: a party of one keeps its own
  line, a multi-member party collapses into one aggregate line (e.g. "a
  pack of 3 goblins"), and members with differing base names collapse to
  "a pack of N creatures".
- **Why:** The Phase 11 design session found the original single-doc scope
  (front-row interception, reach, adjacency) had a hidden prerequisite:
  none of it means anything against a mob, because mobs had no formation
  concept at all. 11a is the foundation sub-phase — it exists so 11b (real
  company-vs-party target assignment) and 11c (formation tactics) have
  something to act on.
- **A `Party` is never persisted**, matching `characters.Aggro`'s own
  non-persistence (see the shared prior-art check in the Phase 11 overview
  design doc) — it is assembled fresh, on demand, every time a room's mob
  listing is built, since there is no state to keep in sync and recomputing
  it is cheap.
- **Deliberate scope note (EHP/DPS left at zero in the room-display path):**
  `internal/combat` (the source of real `EHP`/`DPS` via `MobRank`) already
  imports `internal/rooms`, so `internal/rooms` cannot import
  `internal/combat` back without an import cycle. The room-display
  integration therefore calls `mobparty.Assemble` with `MobSummary.EHP`/
  `DPS` at their zero value — display grouping only needs *which* mobs
  share a party, never the front/mid/back `Formation` ordering, so this
  doesn't weaken the shipped feature. Wiring real `EHP`/`DPS` into a
  `Formation` that something actually reads for combat purposes is 11b's
  concern, from whatever integration point 11b turns out to need.
- **Step completed:** Full implementation plan
  (`docs/superpowers/plans/2026-09-22-phase-11a-enemy-parties.md`), both
  tasks (domain package, room-display integration), plus this status
  update.
- **Key commits:** `9a2c1da2` (plan), `04c571f3` (`internal/mobparty`
  domain package and tests), `0b6e274d` (`internal/rooms` party-grouped
  display and tests), landing on `phase-11a-enemy-parties`.
- **Verification:** `go test -race ./...` (1725 tests / 78 packages),
  `make generate` (no wiring change — no new module, no new config), and
  `make validate` pass. Focused tests cover: single-mob (solo) parties,
  multi-mob grouping by shared `Groups` tag, the EHP-descending formation
  ordering, the 5-member cap/split, and the room-display grouping/mixed-
  name/pluralization behavior.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic domain and room-display unit tests cover the grouping
  and rendering behavior described above.
- **Deferred:** Phase 11b (unit-vs-unit engagement — nothing yet reads a
  party's `Formation` for combat targeting), Phase 11c (formation tactics),
  Phase 11d (guard reactions, crit effects, wounds, AI personality, all
  unscheduled), real `EHP`/`DPS` wiring into a combat-facing `Formation`
  (see the scope note above), and `modules/gmcp/gmcp.Room.go`'s parallel
  GMCP mob-list builder (~line 366-384), which still lists mobs
  individually and was not touched this phase — a GMCP-aware client will
  not see grouped party listings yet, only the plain-text room description
  does.

### Phase 11 — Formation Combat design (decomposed, not yet implemented, 2026-09-22)

- **What:** A collaborative design session found the original single-doc
  Phase 11 scope (handoff §36: front-row interception, melee reach,
  polearm reach, ranged rear-line benefit, adjacency queries) had a hidden
  prerequisite — none of it can mean anything against a mob, because
  today's combat has no enemy formation, no coordinated company-vs-mob
  targeting, and single-target `Aggro` with zero positional awareness.
  The session decomposed Phase 11 into three sequential sub-phases:
  **11a Enemy Parties** (mobs get the same `company.Formation` 3×3 grid,
  auto-assigned via an `EHP`/`DPS` role heuristic from
  `internal/combat.MobRank`), **11b Unit-vs-Unit Engagement** (a fight
  becomes company-vs-party with real coordinated target assignment via a
  minimal weakest/strongest/random preference, not one ad hoc `Aggro`),
  and **11c Formation Tactics** (the original scope, with the reach model
  resolved through several rounds of back-and-forth to column-occupancy —
  a plain melee attack can only land on a column's current frontmost
  occupant; polearm/innate Reach extends to frontmost-or-one-behind;
  ranged ignores column depth — rather than a flat row-distance check,
  since formation is locked during combat and only enemy attrition
  changes what's reachable). A fourth bucket, **11d**, stays deferred and
  unscheduled: guard reactions, weapon-flavored crit effects, wounds, and
  full AI targeting personality (handoff items 6-9).
- **Why:** Building formation tactics directly on today's combat model
  would have produced inert code with nothing to act on — no enemy ever
  has a "row," so "front-row protection" and "reach" have no defenders to
  apply to. The sub-phases are ordered so each depends only on the
  previous one existing (11a before 11b before 11c).
- **Key decisions also locked in:** the v2 continuous
  Readiness/Wind-up/Cast/Recovery timing model stays parked — everything
  in 11a-11d is a "who is grouped with whom, who is a legal target"
  problem solvable within the existing round-based model, not a timing
  question. `internal/parties` (the separate native multiplayer grouping
  system) stays dormant and untouched; "Unit" is scoped to one player's
  own company for now, with a forward-looking note captured for later
  (a joining player's character becomes a formation member of the host's
  company, not a merge of two grids). PvP formation interaction stays out
  of scope per the handoff's own "future feature" framing.
- **Step completed:** Design only.
  `docs/superpowers/specs/2026-09-22-phase-11-formation-combat-design.md`
  (overview, shared prior-art, and locked decisions) plus
  `2026-09-22-phase-11a-enemy-parties-design.md`,
  `2026-09-22-phase-11b-unit-engagement-design.md`, and
  `2026-09-22-phase-11c-formation-tactics-design.md` are written,
  self-reviewed, and committed on `phase-11-formation-combat`, awaiting
  owner review before implementation plans are written.
- **Verification:** None yet — no code has been written. Design-only
  commit; `go test -race ./...` unaffected.
- **Deferred:** Implementation of 11a/11b/11c themselves (next), 11d
  (unscheduled), v2 timing model, multiplayer-party/company reconciliation,
  and PvP formation interaction.

### Phase 10 — Mounts (complete, 2026-09-22)

- **What:** Delivered a minimal, durable, leader-owned mount with one real,
  wired gameplay effect. `internal/mount` is a GoMud-free domain (matching
  `internal/expedition`/`internal/camping`/`internal/weather`/
  `internal/encumbrance`) — the simplest of the five: a `Mount{LeaderUserID,
  Type}` with no decaying state at all (no fatigue/health/feed; handoff §35
  defers those to "Later"). A `MountSpec` carries a wired
  `CargoCapacityBonusGrams` and a computed-but-unwired `TravelDurationPct`.
  `modules/mount` owns an embedded/overlaid mount-type table (one proving
  type, `pack-horse`), a durable leader-keyed YAML registry, `mount` /
  `mount stable <type>` / `mount release` commands, and registers itself as
  `internal/mount`'s `Provider`. The one integration change this phase:
  `modules/encumbrance`'s `CurrentLoad` now adds
  `mount.CapacityBonus(leaderUserID)` to its computed (never persisted)
  capacity — a real, observable increase in cargo capacity for a leader
  with a mount, touching no persisted schema on the already-shipped Phase 9
  module.
- **Why:** The handoff (§35) requires travel and load systems (Phases 5-9)
  before mounts, and states "MVP mount effects: increases travel speed
  and/or increases cargo capacity" — unlike weather/encumbrance, a
  display-only mount would have no value, so this phase wires one real
  effect. Cargo capacity was chosen over travel speed because it needed no
  schema change to an already-shipped phase (`modules/encumbrance`'s
  capacity is computed, not persisted), while travel speed still needs the
  same `TravelSession` schema risk flagged twice already in Phases 8 and 9.
- **Step completed:** Design doc
  (`docs/superpowers/specs/2026-09-22-phase-10-mounts-design.md`),
  confirmed and implemented directly per the owner's standing "continue
  with whatever you recommend" instruction.
- **Key commits:** design doc, plus `internal/mount`, `modules/mount`, and
  the `modules/encumbrance` capacity-bonus wiring landing on
  `phase-10-mounts`.
- **Verification:** `go test -race ./...` (1716 tests / 77 packages),
  `make generate`, and `make validate` pass. Focused tests cover mount
  spec/assignment validation, stable/release (including persistence-failure
  rollback for both), the capacity-bonus provider (including an unknown
  mount type contributing 0), the `mount` command, malformed mount-type
  config rejection, and — in `modules/encumbrance` — that a leader's
  computed capacity correctly includes the mount bonus, and is correctly
  unaffected without one.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic fake-store/injected-spec harness and domain/module/
  command test suites cover the engine behavior.
- **Deferred:** Mount fatigue/health/feed, terrain suitability, individual
  (per-member) assignment, an acquisition/cost economy, and — per the
  design doc's scope decision — actually wiring `TravelDurationPct` into
  `modules/expedition`.

### Phase 9 — Encumbrance and Cargo (complete, 2026-09-22)

- **What:** Delivered a party/expedition-level, weight-based encumbrance
  engine, deliberately separate from GoMud's native, unrelated, count-based
  `Character.CarryCapacity()` throttle (`internal/characters/character.go`),
  which this phase never touches. `items.ItemSpec` gained a `Weight int`
  field (grams; zero-value default, same soft-migration shape as Phase 4's
  `Nutrition`/`Hydration`). `internal/encumbrance` is a GoMud-free domain
  (matching `internal/expedition`/`internal/camping`/`internal/weather`): a
  durable `Cargo` container with pure, validated, copy-returning
  `Deposit`/`Withdraw`, and a computed (never persisted) `Load` with
  `Ratio()` and a `LoadBand` threshold table resolved by `ResolveBand`.
  `modules/encumbrance` owns a leader-keyed YAML cargo registry, a flat
  config-driven `CapacityKg` and `LoadBands` table (embedded default +
  on-disk overlay, malformed bands rejected and logged rather than
  guessed), a read-only `encumbrance.Provider` query seam
  (`CurrentLoad(leaderUserID)`), and a `cargo` / `cargo put <item>` /
  `cargo take <item>` command reusing `Character.FindInBackpack` and
  `items.FindMatchIn`'s established name-matching. Personal weight sums
  each carried/worn item's own resolved spec (`Item.GetSpec().Weight`,
  honoring any per-instance override); cargo weight sums each stack's
  configured item spec by ID.
- **Why:** Phase 9 is the prerequisite the handoff (§34, §18) requires
  before mounts (Phase 10: "Do not implement mounts before normal travel,
  encumbrance, and fatigue work"). The owner confirmed, per their standing
  "continue with whatever you recommend" instruction: Option A (engine +
  display only, no `TravelSession`/camp-rest schema change this phase —
  same fork Phase 8's weather design hit, same precedent applied), full
  `cargo put`/`cargo take` commands now (a container nobody can use is a
  hollow slice), a flat config-driven capacity (no per-company Strength
  aggregation), and zero-weight items with balance data authoring deferred.
  A design-time review found the "refuse cargo put/take that would exceed
  capacity" idea from the initial design doc was actually vacuous — moving
  an item between backpack and cargo never changes total party weight — so
  it was dropped rather than implemented as dead code; see the design doc's
  implementation-correction note.
- **Step completed:** Design doc
  (`docs/superpowers/specs/2026-09-22-phase-9-encumbrance-cargo-design.md`),
  confirmed and implemented directly.
- **Key commits:** design doc, plus the `items.ItemSpec.Weight` field,
  `internal/encumbrance` domain, and `modules/encumbrance` module landing on
  `phase-9-encumbrance`.
- **Verification:** `go test -race ./...` (1702 tests / 75 packages),
  `make generate`, and `make validate` pass. Focused domain/module tests
  cover cargo deposit/withdraw (including the merge-into-existing-stack and
  remove-when-empty cases), load-ratio/band resolution, personal-plus-cargo
  weight computation, put/take's weight-invariance under transfer,
  persistence-failure rollback for both put and take, the `cargo` status
  render, and malformed load-band config rejection.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic fake-store/injected-item-spec/injected-user harness and
  domain/module/command test suites cover the engine behavior.
- **Deferred:** Mount cargo capacity (Phase 10), stealth/noise load effects,
  per-companion carried gear (companions are native mobs with no modeled
  inventory in this codebase), cargo loss/theft/raiding, a `company
  status`-embedded load line (the standalone `cargo` command already shows
  it), and — per the Option A recommendation — actually wiring
  `TravelDurationPct`/`FatiguePct` into `modules/expedition`/
  `modules/camping`. Weight data for existing/Dunmar items was not authored
  this phase; every current item defaults to 0 (unweighted) until a
  follow-up data pass.

### Phase 8 — Weather (complete, 2026-09-22)

- **What:** Delivered a durable, round-driven, zone-scoped weather engine.
  `internal/weather` is a GoMud-free domain (matching `internal/expedition`
  and `internal/camping`'s split): a `Condition` (name, description, and
  informational `TravelDurationPct`/`ExertionPct`/`RestRecoveryPct`
  multipliers clamped to 25–300 at validation) and a `ZoneWeather` record
  whose `Due`/`Advance`/`Established` are pure, round-number-only, and
  copy-returning — the one durable-timing package in the codebase that is
  intentionally *not* real-UTC-based, since weather must track the shared
  round clock (handoff §33). `modules/weather` owns an embedded/overlaid
  biome condition-table config (forest only, four weighted conditions,
  40–120 round change interval), a durable zone-keyed YAML registry, a
  single `events.NewRound` listener that advances every already-tracked
  zone whose weather is due, and load/copyover recovery that establishes
  any newly trackable zone (biome now has a configured table) and rolls an
  overdue zone forward exactly once to the current round — never guessing a
  default and never replaying multiple missed transitions, since weather
  has no side effect to double-apply (simpler than expedition/camping
  recovery). A read-only `weather.Provider` query seam
  (`CurrentCondition(zone)`, `RenderLine(zone)`) mirrors
  `survival.CompanyService`/`camping.ViewProvider`. The `weather` command
  and `look` (purely additive — a line appended after the room description
  panel, never replacing it, unlike travel/camp views) consult it for a
  tracked zone; an untracked zone or one with no configured biome table
  shows neither.
- **Why:** Weather is the next expedition-adjacent atmosphere system and
  must never advance `gametime`, the round counter, or move any player. The
  design doc's Option A (read-only engine + descriptions only) was
  confirmed by the owner over Option B, so `modules/expedition`'s
  `TravelSession` and `modules/camping`'s rest recovery are untouched this
  phase; actual travel-duration/exertion/rest-recovery multiplier wiring is
  a deferred follow-up once the engine is proven, per the design doc's own
  recommendation.
- **Step completed:** Design doc
  (`docs/superpowers/specs/2026-09-22-phase-8-weather-design.md`) confirmed
  by the owner (Option A, forest-only table, 40–120 round cadence), then
  implemented directly given the timer/recovery/concurrency escalation rule.
- **Key commits:** `5d4d114c` (design), plus the `internal/weather` domain,
  `modules/weather` module, and `look` integration landing on
  `phase-8-weather`.
- **Verification:** `go test -race ./...` (1685 tests / 73 packages),
  `make generate`, and `make validate` pass. Focused domain/module tests
  cover condition/zone-weather validation, weighted rolls, load/copyover
  establishment and overdue-advance-exactly-once recovery, invalid-record
  and unknown-condition retention for operator repair, per-round due/not-due
  advancement, the provider seam, the `weather` command, and malformed
  biome-table config rejection.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic injected round-number/RNG harness and domain/module/
  command test suites cover the engine behavior.
- **Deferred:** Storms as discrete hazard encounters, weather-driven
  room/exit changes, player-visible forecasts, seasons/climate modeling,
  mount/encumbrance interaction (Phases 9–10), indoor/outdoor room-level
  overrides beyond "zone has a tracked biome", and (per the design doc's
  Option A recommendation) actually wiring the `TravelDurationPct`/
  `ExertionPct`/`RestRecoveryPct` multipliers into `modules/expedition` and
  `modules/camping` all remain outside Phase 8.

### Phase 7 — Camping (complete, 2026-09-22)

- **What:** Delivered a durable, real-time, fatigue-only campsite and rest
  loop. `internal/camping` is a GoMud-free domain: a leader/room-keyed `Camp`
  with `FireLit` and an optional `RestSession{StartedAtUTC,State}`; progress
  and due-ness derive from real UTC elapsed time. `internal/survival` gained
  an `ApplyCompanyRestRecovery(leaderUserID, operationID, fatigue)` seam with
  its own applied-operation ledger, so replay after a crash or restart cannot
  restore fatigue twice. `modules/camping` owns a leader-keyed YAML registry,
  `camp`/`camp status`/`camp fire`/`camp rest`/`camp break` commands, room-tag
  (`camping`) eligibility, a single 60-second completion timer per leader with
  generation-guarded stale-callback protection, and a persist-Completed-
  before-calling-survival protocol: a crash between finalizing the rest and
  applying recovery retries recovery alone (idempotently and silently) on the
  next status/look/load call, tracked by a durable per-leader
  `RecoveryApplied` marker in the same registry. Load/copyover recovery
  reschedules an active rest's remaining duration, completes an overdue rest
  once, retries pending recovery for an already-completed rest, and retains
  an invalid camp untouched for operator repair. `internal/camping` also
  exposes a `ViewProvider`/`MovementProvider` seam, mirroring
  `modules/expedition`'s pattern: native `go` refuses ordinary movement with
  remaining-rest progress while resting, and `look` renders the camp/rest
  view in place of the room only while resting (an idle or broken camp never
  blocks movement or replaces room rendering). Dunmar 2002 (Fork at the Black
  Oak) is tagged `camping` as the proving room.
- **Why:** Camping is the next expedition recovery vertical slice and must
  never advance the shared world clock or round count. Weather, supplies,
  encounters, temporary rooms, and multiplayer camp discovery remain
  deferred.
- **Step completed:** Handoff-style Phase 7 plan
  (`docs/superpowers/plans/2026-09-22-phase-7-camping.md`), all five tasks.
- **Key commits:** `904176f1` (design), `1675db0c` (plan), `472a7985`
  (validated `internal/camping` domain model), `955fc718` (idempotent company
  rest recovery), and the module/wiring commits establishing durable camps,
  completing real-time rest, and wiring movement/look/the proving room.
- **Verification:** `go test -race ./...` (1670 tests / 71 packages),
  `make generate`, and `make validate` pass. Focused race suites cover
  `internal/camping`, `internal/survival`, `modules/survival`, and
  `modules/camping`, including stale-timer, failed-recovery-retry, failed-
  finalization-save-retry, restart/copyover reschedule and overdue
  completion, invalid-record retention, and idle-camp-never-blocks-movement
  cases.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic fake store/scheduler/survival, injected clock, and
  command/view/movement/race suites cover the proving room behavior.
- **Deferred:** Weather, shelter, fire fuel/items, cooking, watches, camp
  encounters, temporary/discoverable camp rooms, multi-player camps, and
  sleep-until-dawn remain outside Phase 7.

### Phase 6 — Travel interruptions (complete, 2026-09-22)

- **What:** Added one durable, configured `fallen-tree` interruption to the
  expedition domain and module. A profile can interrupt once at checkpoint 1–9;
  the Oak Road proves it at checkpoint 5. Active-time arithmetic freezes route
  progress, remaining duration, and exertion while paused. The module schedules
  the next route boundary, finalizes Phase 5 checkpoint exertion before
  persisting an interruption, and retains malformed records for operator repair.
  `travel status`, `look`, and movement refusal identify a paused obstruction;
  `travel resume` banks paused UTC time and resumes exactly the remaining active
  duration, while `travel return` durably records `Cancelled` before cleanup
  without moving or refunding the company.
- **Why:** This is the first safe, deterministic interruption point for real-time
  multiplayer travel. It remains durable across restart/copyover, preserves the
  Phase 5 operation-ID exactly-once survival protocol, and never advances global
  game time or round count.
- **Step completed:** Handoff Phase 6 ("Travel Interruptions").
- **Key commits:** `e25d2ec6`, `2a45f473`, `17091a31`, `a8255e51`, `646d00ab`,
  `a9c2c018`, lifecycle/acceptance coverage through `92edb585`, and `56dc9ff6`
  (terminal, malformed-record, and stale-timer durability correction).
- **Verification:** `go test -race ./...`, `make generate`, and `make validate`
  pass after the final durability correction; focused domain/module and
  cross-package race suites also pass. Task-scoped and final whole-branch
  reviews covered malformed-record retention, parser/configuration, timer and
  checkpoint ordering, terminal recovery, and command/recovery lifecycle behavior.
- **Live acceptance:** Not run: this host has no interactive Telnet client.
  The deterministic fake scheduler, injected clock, persistence, recovery,
  command, view, and race suites cover the proving route behavior.
- **Deferred:** Combat interruptions, random event tables, rewards, en-route
  rooms, camping, weather, cargo, mounts, and formation effects remain outside
  Phase 6.

### Phase 5 — Terrain and travel profiles (complete, 2026-09-21)

- **What:** Added an optional `travel_profile` field to `exit.RoomExit` and a
  data-driven expedition system. `internal/expedition` owns the GoMud-free
  travel domain: validated `TravelProfile`s, the `Traveling`/`Interrupted`/
  `Completed`/`Cancelled` session state machine, clamped real-UTC progress,
  ten-checkpoint proportional exertion math, and mutex-protected start/view/
  movement provider seams. `modules/expedition` owns configured profile loading, durable
  leader-keyed sessions (YAML `ReadBytes`/`WriteStruct` like `modules/survival`),
  real-time `time.AfterFunc` completion timers, load/copyover recovery,
  destination move, exactly-once arrival, and the `travel status` command.
  Native `go` runs the same lock/exit-message/destination/script admission and
  then starts travel before action-point deduction for marked exits; native
  `look` renders the travel view at entry. `internal/survival` gained a
  `CompanyService` seam (`ApplyCompanyExertion`, `CompanyNeeds`) implemented by
  `modules/survival`, so travel accrues hunger/thirst/fatigue per company member
  without importing another module. Added the Dunmar proving route (room 2001
  Dunmar West Gate to 2002 Fork at the Black Oak) with the `oak-road` profile.
- **Why:** Travel is Ashveil's core expedition mechanic; it must be durable
  across disconnect/restart/copyover, exactly-once on arrival, and must never
  advance GoMud's global clock or round count.
- **Step completed:** Handoff Phase 5 ("Terrain and Travel Profiles").
- **Key commits:** `af6aa611` (domain), `7d11113c` (exit field + go/look
  adapters), `a14b086c` (durable module, profiles, survival seam, views),
  `1ac90a93` (timers, checkpoint sync, recovery), `983d04bd` (oak-road route),
  `cceb47e5` (refuse ordinary movement while travelling), and `af34e504`
  (merged durability corrections: pending operation IDs, survival replay ledger,
  checkpoint recovery, and player-spawn reconciliation).
- **Behavior:** An unmarked exit stays instant. A marked exit starts a durable
  session; the leader and company stay at the origin. While travelling, ordinary
  movement is refused with progress/remaining time, `look` shows
  origin/destination/profile/progress/remaining time/company needs, and
  `travel status` shows the same and syncs any earned checkpoint. Exertion is
  charged only at the ten durable progress checkpoints and telescopes to exactly
  the profile total at completion. On completion the module applies the final
  checkpoint, persists `Completed`, moves the leader once via `rooms.MoveToRoom`,
  commands native charmed companions to follow, announces arrival once, and
  removes the record after verifying the destination. Recovery at the
  destination cleans up, at the origin retries the move, and anywhere else is
  retained for operator repair. Checkpoint costs are prepared as durable,
  deterministic operations before survival changes; the survival ledger makes a
  restart replay idempotent, then expedition finalizes the checkpoint. A
  returning leader synchronizes and reconciles travel from `PlayerSpawn`, and
  `look`/`travel status` retry an overdue completion after a transient timer
  failure.
- **Verification:** `go test -race ./...` (1603 tests / 69 packages),
  `make generate`, `make validate`, and `go build ./...` pass. Focused race
  suites cover `internal/expedition`, `internal/exit`, `internal/usercommands`,
  `internal/survival`, `modules/survival`, and `modules/expedition`.
- **Copyover note (deviation from plan):** The plan asked for a registered
  `copyover.Contributor`, but `internal/copyover/AGENTS.md` forbids plugins from
  implementing it. Continuity instead relies on the plugin `SetOnSave`/
  `SetOnLoad` path (which runs immediately before/after a copyover): `onLoad`
  re-derives progress from the durable UTC start, reschedules a remaining
  session, or completes an overdue one. No copyover contributor is registered.
- **Durability correction:** Survival and expedition remain separate plugin
  writes, but the persisted pending-operation protocol makes their recovery
  idempotent: a crash between writes retries the same operation ID without a
  second survival cost.
- **Deferred:** Phase 6 owns interruption/pause/resume; Phase 7 camping/rest;
  Phase 8 weather; Phase 9 cargo; Phase 10 mounts. Needs remain informational in
  Phase 5 and never change duration, arrival, movement, combat, or health.
- **Live acceptance:** Not run (no interactive Telnet prerequisites); unit,
  module, command, persistence, timer, recovery, and race coverage only.

### Phase 4 — Survival state (complete, 2026-09-17)

- **What:** Added durable, per-company-member hunger, thirst, and fatigue in
  `0..100`, with five threshold bands, a pure `internal/survival` domain, and a
  `modules/survival` plugin that persists state under the module, projects the
  company roster, resolves leader/companion provisioning selectors, and renders
  a read-only `survival` command. `items.ItemSpec` gained optional `nutrition`
  and `hydration` metadata; native `eat`/`drink` accept an optional trailing
  member selector and provision the target from the leader's backpack before
  consuming the item. `modules/company` synchronizes summon/dismiss through a
  one-way `internal/survival` lifecycle seam and supplies companion names
  through a roster seam.
- **Why:** Survival state is the prerequisite for Phase 5 travel exertion and
  Phase 7 camp/rest recovery; it must persist across login/logout/copyover and
  never advance global game time.
- **Step completed:** Handoff Phase 4 ("Survival State").
- **Corrections (2026-09-17):** A review of `e5625faa..5f07d4f0` found state
  loss and inheritance bugs. Companions now carry a persisted
  `next_companion_id` high-water mark, so a dismissed ID is never reassigned
  across dismiss, dismiss all, or save/load, and a failed summon that has not
  yet committed survival state restores the exact prior record. The lifecycle
  seam captures exact `MemberSnapshot`s, so a failed company save restores the
  dismissed companion's real needs instead of a fresh default and joins
  compensation errors. `modules/company` reconciles
  every loaded roster into survival once after its registry loads, pruning
  orphaned companions and initializing current ones; a reconcile failure blocks
  company mutations. Consumable parsing again uses native backpack matching
  (partial and `name#n` numbered) and treats a trailing token as a target only
  when the survival module confirms it names a current member. Item specs reject
  negative `nutrition`/`hydration` while zero stays valid for legacy items.
- **Key commits:** `5872dadd` (domain), `1271e45d` (item metadata), `1cc7caeb`
  + `2a62de1b` (module persistence and roster seam), `609d0e4f` (eat/drink),
  `9055034d` (company lifecycle sync), `8f72807b` (roster-default status);
  corrections `08d33467` (non-reusable companion IDs), `7ed9c926` (exact
  snapshots and roster reconciliation), `a758a0c0` (rollback compensation),
  `470a2415` (consumable matching), and the metadata validation in this
  record's commit.
- **Behavior:** Needs change only through explicit APIs (`ConsumeFood`,
  `ConsumeWater`, `ApplyRestRecovery`, `ApplyExertion`). `eat <item> [member]`
  and `drink <item> [member]` accept `leader`/`me`/`self`, `#<id>`/`<id>`, or an
  unambiguous companion name; items are consumed only after survival mutation
  and persistence succeed. Successful summon creates default companion state;
  single/all dismissal prunes exactly the removed companions.
- **Verification:** `go test -race ./...` (1538 tests / 67 packages),
  `make generate`, `make validate`, and `make build` pass. Focused race suites
  cover `internal/company`, `internal/survival`, `internal/items`,
  `internal/usercommands`, `modules/survival`, and `modules/company`.
- **Known limitation:** Company and survival persist to separate plugin files,
  so summon/dismiss is not cross-file atomic. Rollback compensates with exact
  snapshots and surfaces joined errors, but a crash between the two writes can
  still leave the files divergent.
- **Deferred:** Camp/rest/sleep recovery is Phase 7; cargo, capacity, and
  automatic provisioning are Phase 9. No idle/offline drain and no health,
  combat, movement, or travel penalties in Phase 4.
- **Live acceptance:** Not run (no interactive Telnet prerequisites); unit,
  module, command, and race coverage only.

### Phase 4 final corrections (2026-09-17)

- **What:** Failed summon cleanup now treats a companion ID as spent once
  survival has durably recorded it. The transient companion is removed but the
  advanced `next_companion_id` high-water mark is retained and persisted, and
  the primary failure, any failed survival removal, and any failed high-water
  write are returned together. Provisioning selectors are now authorized only
  through the authoritative company roster: a stale survival record can no
  longer target a dismissed companion, and a current companion with no stored
  record is initialized to full needs when provisioned.
- **Why:** A review of `5f07d4f0..8eeaf973` found that failed summon cleanup
  could restore a spent ID and let a retry inherit stale survival state, and
  that numeric selectors trusted persisted survival state for authorization.
- **Key commits:** `65c6460b` (retain spent IDs after failed summon cleanup),
  `ac32c100` (authorize numeric targets from roster).
- **Verification:** `make generate`, `make validate`, and
  `go test -race ./...` (1544 tests / 67 packages) pass.
- **Known limitation:** Unchanged; company and survival remain separate plugin
  writes, so summon/dismiss is not cross-file atomic. Cleanup errors are joined
  and surfaced, and a spent companion ID is never silently reused.

### Phase 4 durable identity reservation (merged, 2026-09-17)

- **What:** Survival persistence now stores a per-leader
  `reserved_next_companion_ids` lower bound alongside companion needs.
  `EnsureCompanyMember` advances that reservation before persisting the new
  companion state. Before every summon, the company registry adopts the
  reservation as its minimum next ID.
- **Why:** This closes the final restart-safety hole: if survival records a
  companion but spawning fails, survival cleanup fails, and the company
  rollback write also fails, a fresh company registry still starts at `#2`
  rather than reusing `#1` and inheriting stale survival state.
- **Key commits:** `1d465b8f` (durable survival reservation and regression
  coverage), merged to `master` by `e9432182`.
- **Verification:** `make validate` and `go test -race ./...` pass after the
  composed serialized failure/restart regression was added. An independent
  review found no critical or important issues.
- **Known limitation:** The company and survival files remain separate direct
  writes, so they are not a transaction and a partial low-level write is still
  unrecoverable. The durable survival reservation prevents reuse of any ID
  whose survival initialization was successfully persisted; load reconciliation
  continues to repair ordinary roster/needs divergence.

### Phase 3 invariant corrections (2026-09-17)

- **What:** Hard-capped companies at four companions plus their leader, and normalized duplicate persisted formation occupants by keeping the first row-major cell.
- **Why:** Restores Phase 3’s five-character cap and one-cell-per-member invariants even when module configuration or stored YAML is invalid.
- **Verification:** `make generate`, `make validate`, and `go test -race ./...` passed.
- **Merge:** `main-deepseek` fast-forwarded into `master` at `3d6addd4`; `go test -race ./...` re-run on `master` passed (1434 tests / 65 packages).

### Phase 3 — Company roster + 3×3 formation (complete, 2026-09-17)

- **What:** Expanded the company from one companion to a leader plus up to four
  companions (five-member cap) with stable companion IDs, and added a persistent,
  validated 3×3 tactical formation with `formation move|swap|clear` commands.
  `internal/company` owns the pure roster/formation model; `modules/company` owns
  multi-instance runtime tracking, legacy-record migration, persistence, and
  commands.
- **Why:** Formation is Ashveil's identity feature and needs a multi-member roster
  to be meaningful. Reused GoMud charm/follow, mobs, events, users, and module
  persistence; native `internal/parties` was left untouched.
- **Step completed:** Handoff Phase 3 ("3×3 Formation State") plus the deferred
  five-member company cap.
- **Key commits:** `8bf9d2ab` (formation grid), `159ec87b`+`e462c258` (roster and
  snapshot fix), `80cb28ba`+`3fb4eb43` (multi-instance runtime), `fc0e1987`+
  `3c963a1d` (legacy migration), `ed60995d`+`8d52b0c9`+`795a9222` (formation
  commands and rollback fixes), `39d993f2` (config + module docs).
- **Verification:** `go test ./internal/company ./modules/company` (72 tests),
  `make validate`, `make generate` (no wiring change), and `go test -race ./...`
  (1429 tests / 65 packages) pass. Legacy single-`companion` records migrate to
  `companions[0]` with ID 1; formation cells for dismissed companions are pruned.
- **Config:** `MaxCompanions: 4` in `modules/company/files/data-overlays/config.yaml`.

### Phase 2 — Company companion slice (complete, 2026-09-17)

- **What:** Added a durable, leader-keyed `company` model in `internal/company/`
  and a `modules/company/` plugin that persists the company record, spawns an
  allow-listed companion as a native GoMud mob, restores it after login/copyover,
  and exposes `company summon|status|dismiss`.
- **Why:** Prove a saved companion can be owned, followed, and restored without
  duplicating GoMud's party, mob, charm, movement, or persistence systems, and
  without an `ashveil*` layer.
- **Step completed:** Handoff Phase 2 ("Minimal Company Slice"). A durable
  `MobTemplateID` is stored; a runtime `InstanceId` is never persisted.
- **Key commits:** `8c1eca92` (model), `8be7c1f9` (restoration), `99f02261`
  (commands), `4dc1ea27` (attachment recovery + persistence-failure guards),
  `a4d71439`/`54305926` (wrap-up hygiene).
- **Verification:** `go test ./internal/company ./modules/company` (37 tests),
  `make validate`, and `go test -race ./...` (1394 tests / 65 packages) pass.
  Live Telnet acceptance passed (summon → native follow → dismiss → restart
  rehydration). `make test` is blocked locally by the `js-lint` stage (see
  Known issues).

### Phase 0–1 — Fork, baseline, integration map (complete, 2026-09-16)

- **What:** Forked GoMud into `Robinsond76/ashveil-gomud`, kept
  `GoMudEngine/GoMud` as read-only `upstream`, verified the vanilla baseline,
  and wrote the integration map, handoff, and phase plans.
- **Why:** Establish a clean, evidence-backed foundation before gameplay work.
- **Step completed:** Handoff Phases 0 and 1.
- **Key commits:** `1fba48a0` (BSD `awk` portability fix in `make help`),
  `6fd8c4b0` (baseline docs), `997d6db5`/`71c74f81`/`3cb4130c` (Phase 2 design
  and plan).
- **Verification:** Recorded in `docs/BASELINE_VERIFICATION.md`.

## Known issues / deferred items

- Phase 5 was implemented and committed directly on `master`. Going forward,
  plan and phase work must run on an isolated worktree/feature branch and merge
  back only after verification; see the root `AGENTS.md` ("Branching &
  Worktrees") and `docs/superpowers/plans/README.md`.
- `make test` stalls in the `js-lint` stage because it shells out to `npx
  jshint`; the documented fallback `go test -race ./...` passes. Environmental,
  not a code failure.
- Plugin `WriteStruct`/`WriteBytes` persistence is a direct (non-atomic) file
  write. Command state rolls back on failure, but a partial low-level write
  cannot be recovered.
- Live server acceptance has not been run for Phase 3, Phase 4, or Phase 5;
  unit/race tests cover the roster, formation, migration, survival persistence,
  provisioning, travel start/view/completion/recovery, and command behavior.
- Company and survival use separate plugin writes, so summon/dismiss is not
  cross-file atomic. Compensation failures are surfaced alongside the primary
  error, and the survival-side durable reservation prevents reuse of an ID once
  its survival initialization has persisted. A partial low-level file write is
  still not transactionally recoverable.
- Expedition survival exertion and its own checkpoint remain separate plugin
  writes, but Phase 5 now persists a deterministic pending operation before
  charging survival. The survival ledger deduplicates recovery, so a crash or
  failed checkpoint write cannot double-charge the company.
- Phase 5 copyover continuity relies on plugin `SetOnSave`/`SetOnLoad` rather
  than a `copyover.Contributor`, because modules are forbidden from registering
  copyover contributors. `onLoad` runs after `copyover.Restore` and reschedules
  or completes sessions from the durable record.
- Company/formation/survival state is process-local with no mutex, matching the
  existing event-loop dispatch assumption; revisit if command dispatch moves off
  the main loop.
- Camping and survival rest recovery are separate plugin writes, like
  expedition/survival exertion. `modules/camping` persists `Completed` before
  calling survival, and a durable per-leader `RecoveryApplied` marker (plus
  survival's own applied-operation ledger) makes a crash between the two
  writes retry recovery alone rather than double-apply or silently drop it.
- Live server acceptance has not been run for Phase 7; unit/race coverage
  spans the camp/rest domain, module commands, eligibility, timers, recovery,
  and movement/view integration.
- Live server acceptance has not been run for Phase 8; unit/race coverage
  spans the weather domain, module recovery/round-advance/command behavior,
  and the `look` line integration.
- Phase 8 weather is read-only this phase (design doc Option A): its
  `TravelDurationPct`/`ExertionPct`/`RestRecoveryPct` multipliers are
  computed and validated but nothing yet applies them to
  `modules/expedition`'s `TravelSession` or `modules/camping`'s rest
  recovery. That wiring is an explicit deferred follow-up, not an oversight.
- Deferred by design: recruitment economics, companion custom names, equipment,
  injuries, AI orders, death/permadeath rules, formation combat effects, and
  cargo/mount integration (Phases 9–10). Camping itself excludes weather
  effects, shelter, fire fuel/items, cooking, watches, encounters,
  temporary/discoverable camp rooms, multi-player camps, and sleep-until-dawn.
  Weather itself excludes storms as hazard encounters, weather-driven
  room/exit changes, forecasts, seasons/climate modeling, and non-forest
  biome tables.
- Live server acceptance has not been run for Phase 9; unit/race coverage
  spans the encumbrance domain, cargo deposit/withdraw, module load
  computation, the `cargo` command, and config parsing.
- Phase 9 encumbrance is read-only this phase (design doc Option A, same
  shape as Phase 8): `TravelDurationPct`/`FatiguePct` are computed and
  validated but nothing yet applies them to `modules/expedition` or
  `modules/camping`. Every current item defaults to `Weight: 0`
  (unweighted); no item in the shipped data was authored with a real
  weight this phase, so the engine has nothing to compute against until a
  follow-up data pass. Encumbrance is a party/expedition-level weight
  system, kept deliberately separate from GoMud's native, unrelated,
  count-based `Character.CarryCapacity()` per-move throttle.
- Live server acceptance has not been run for Phase 10; unit/race coverage
  spans the mount domain, stable/release persistence-failure rollback, the
  capacity-bonus provider, the `mount` command, and the
  `modules/encumbrance` capacity-bonus integration.
- Phase 10 mounts wires only the cargo-capacity effect this phase;
  `TravelDurationPct` is computed and validated on `MountSpec` but nothing
  yet applies it to `modules/expedition`'s `TravelSession`, same Option A
  shape as weather's/encumbrance's own deferred multipliers. Mount
  fatigue/health/feed, terrain suitability, per-member assignment, and an
  acquisition economy are all deferred to a later pass, per handoff §35's
  own "Later" list.
- Live server acceptance has not been run for Phase 11a; unit coverage
  spans the `mobparty` domain package and the `internal/rooms` grouped-
  display rendering. `internal/rooms` cannot import `internal/combat`
  (would cycle), so the room-display integration passes zero-value
  `EHP`/`DPS` into `mobparty.Assemble` — display grouping doesn't need real
  values, but nothing has wired real `EHP`/`DPS` into a `Formation` that
  combat code actually reads yet; that's 11b's job. `modules/gmcp`'s mob
  list is still per-mob, not party-grouped.
- Phase 11b ships only the `internal/engagement` domain layer; nothing in
  `internal/hooks/NewRound_DoCombat.go` was touched. No player-visible
  combat behavior changed this phase — company members still only retaliate
  reactively (today's pre-11b behavior), because the coordinated-engagement
  wiring is intentionally deferred until Phase 11c's legality predicate
  exists to inject into `AssignTarget`'s `legal` parameter for real (see
  the Phase 11b work-log entry). Live server acceptance has not been run
  for Phase 11b; unit coverage spans `internal/engagement`'s selection,
  filtering, and engagement-end logic.
- Phase 11c ships `internal/formationcombat`, the `Reach` schema fields,
  and one read-only `formation reach` command; nothing in
  `internal/hooks/NewRound_DoCombat.go` was touched, so combat behavior is
  unchanged from pre-11c — no interception, no reach gating, no
  coordinated targeting actually happen in a fight yet. The
  `formation reach` command only queries a company's own formation (there
  is no live enemy party/formation to query outside combat), and only at
  `ReachNone` (plain melee) — it's a correctness demo of the predicate, not
  a combat feature. Both `internal/formationcombat` and 11b's
  `internal/engagement` are now fully tested and ready to be wired
  together; that wiring (and the new company-formation-by-mob-instance
  query seam it needs) is the next real piece of work — see the Phase 11c
  work-log entry and the "Next" line above for the full reasoning.
- The player-vs-mob combat-loop wiring above ships only that one
  direction. Mob-vs-player, mob-vs-mob, and 11b's `AssignTarget`
  reassignment-on-death remain exactly as described in the two bullets
  above (still unwired) — see the "Formation combat-loop wiring" work-log
  entry for why each is its own separable follow-up. `internal/hooks` has
  no live-server/integration test coverage for the new gate; only the pure
  `resolveAttackTarget` core and the `FormationProvider`/`mobparty`
  seam-level unit tests exercise it.
- Mob-vs-player interception now ships too (see the "mob-vs-player
  interception" work-log entry): the two bullets above's mob-vs-player
  item is resolved. Mob-vs-mob and 11b's reassignment-on-death remain
  unwired for the same reasons stated there. Same test-coverage caveat as
  the player-vs-mob pass: no `internal/hooks` integration harness, only
  pure/seam-level unit tests.
- Mob-vs-mob now ships too (see the "mob-vs-mob" work-log entry): all
  three bullets above's non-11b items are resolved. Only 11b's
  reassignment-on-death remains unwired for the reasons stated there, plus
  the newly-introduced leader-as-interceptor edge case (a leader placed in
  the front row of their own formation can't intercept for a companion —
  see that work-log entry). Same test-coverage caveat as the previous two
  passes: no `internal/hooks` integration harness, only pure/seam-level
  unit tests.
- 11b's reassignment-on-death now ships too (see the "reassignment-on-death"
  work-log entry): the item named in every bullet above is resolved. This
  closes out Phase 11's foundational combat wiring entirely — what's left
  (11d, the leader-as-interceptor gap) is genuinely new work, not
  follow-up wiring for 11a-11c. Same test-coverage caveat as every prior
  combat-wiring pass: no `internal/hooks` integration harness, only
  pure/seam-level unit tests.
- Docs correction (see that work-log entry): the "proactive
  engagement-trigger" item named above as unbuilt was a documentation
  error. It already existed pre-Ashveil in `attack.go`'s charmed-mob-
  assist loop, and company companions get it for free as permanently-
  charmed mobs. No code changed; only the status doc was wrong.

## Key documents

- `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` — authoritative migration direction.
- `docs/ASHVEIL_GOMUD_INTEGRATION.md` — Ashveil concept → GoMud source map.
- `docs/BASELINE_VERIFICATION.md` — vanilla baseline evidence.
- `docs/superpowers/plans/` — phased implementation plans.
- `docs/superpowers/specs/` — feature designs.
