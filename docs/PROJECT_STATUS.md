# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-09-28
- **HEAD:** Phase 32f (company logistics) merged 2026-09-28, after 32c (enemy groups). Before them, Phase 29b2 (one battle at a time; spawn groups) and player
  help for every Ashveil system are complete and merged to `master`
  (2026-09-28, from `claude/next-phase-wfav4w`). Docs cleanup (finished-phase
  plans/specs and the old work log moved to git history) on
  `claude/docs-cleanup-py1rfb`.
  Play-test feedback (2026-09-28): the
  [roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md)
  and the 32a, 32a2, and 32b designs, on `claude/hopeful-wozniak-piu2lb`.
- **Upstream baseline:** `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`

## Current position

- **Completed:** Phases 0–29c; see the table below. The survival and
  expedition loop (travel, camping, weather, load, mounts), formation
  combat, the environment/skills/economy roadmap (13–21), the company-life
  and onboarding roadmap (22–27, including the tutorial), item weights (28),
  the first combat slices (29a, 29b, 29b2, 29c), enemy groups (32c), and
  company logistics (32f) are all done.
- **Next:** 32d (automatic player and companion combat), then 32e, 32g,
  and 32h (32a, 32a2, 32b, 32c, and 32f are done; 32h is in progress in
  another session), from the owner's
  play-test notes, per the
  [play-test roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md#build-order-recommended-accepted-2026-09-28);
  the combat roadmap continues at 29d.
  Also open: the "Future ideas" row; see Known issues.

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
| 23b | Whetstones | Complete: 10-use whetstone (item 30) in both markets, `sharpen`/`camp sharpen` any time but not mid-fight, one use per member sharpened, durable per-weapon edge (+1 for 20 strikes) spent in combat, auto at camp rest end |
| 24 | Company chemistry | Complete: company-wide. Each member's durable service with the band; a band's tier from the average saved service of the members together (each capped at Sworn), so recruits dilute it; Familiar/Trusted/Sworn (900/2700/6300 rounds) give everyone in the band +2/+4/+6 hit in all four combat directions; `company chemistry`, `status bonuses` |
| 25a | Player death and church return | Complete: one level lost (no protection levels; peak level stops re-granted points), a durable pending mark so a death is charged once, wake at the last city's church (Dunmar's new Chapel of the Wayfarer, Frostfang's Sanctuary as fallback) with the living company; travel, camp, and inn stay abandoned first |
| 25b | Companion death and resurrection | Complete: a dead companion stays on the roster, keeping the gear its body kept; a 3-game-day rescue allowance spent only in the leader's online time; `resurrect` at a church or village shaman with its keeper costs a level; at zero it is lost and archived; Fernhollow village and Old Wenna |
| 26a | Company summary and text surfaces | Complete: `internal/companyview` read model; prompt tokens from a game-loop cache and a default prompt that warns only when needed; `status` as the Ashveil character sheet, grouped `conditions`, company load in `inventory`, last level lost in `experience` |
| 26b | Browser Company panel (GMCP) | Complete: `Company`/`Company.Vitals` GMCP from the 26a summary, sent on change and only to the leader; a Company section above Players in the web client's Party window, safe DOM, keyboard and screen-reader friendly, checked in Chromium |
| 27a | Tutorial framework and first lessons | Complete: `modules/tutorial` runs the course in per-player copies of rooms 900–903 (Waking Hall, Muster Yard, Drill Ground, Gate); progress in MiscData, resumed on login; gates check results; `tutorial`, `tutorial next`, `tutorial skip`; graduation cap once; old JS rooms removed |
| 27b | Tutorial: Survival and Camp lessons | Complete: Weather Yard (904) and Campground (905) before the Gate; Survival passes on a real meal and drink (`survival.OnProvision`) plus `weather`/`temperature`/`strain`/`cargo`, with food and water given once for what the pack lacks; Camp passes on Rested from a real camp rest; course camps struck (`camping.AbandonCamp`) on pass, skip, leave, logout, and placement |
| 27c | Tutorial: practice fight | Complete: Practice Yard (906) before the Gate; a squad of harmless straw soldiers (three footmen in front, an archer behind) per player; `practice` mobs beaten with no XP, drops, gold, kills, or `MobDeath` (`mobcommands.OnPracticeBeaten`); the gate is the squad beaten; death in the course decided (an ordinary death, ending the course as a skip) |
| 27d | Tutorial: Alignment lesson, browser panel | Complete: the Oath Stone (907) before the Gate; `company alignment`, `company inspect corvin` (an outlaw a new company is refused), and `standing`; `company inspect` weighs any recruiter's candidate; a `Tutorial` GMCP package and web client window from the same checklist as the terminal; a course missing rooms is closed but kept |
| 28 | Item weights and the company's whole load | Complete: every shipped item weighed (grams, per-type ranges), starter kits 4.8–8.5 kg; living companions' worn and carried gear in the company load (`company.CompanionGearGrams`), live mob when out, else record or template; `cargo` shows the split; GMCP `companion_g` |
| 29a | Combat fixes from the 5v5 simulation | Complete: a round-start engagement upkeep keeps an engaged company and enemy party fighting as a whole (the leader turns from an unreachable target, the killer and leader rejoin, the whole party joins, hostility can't lapse mid-fight); unplaced members can be struck; `break` holds; `formation reach` answers against the enemy in a fight; `internal/enemyparty` |
| 29b | Combat event stream and battle summary | Complete: `internal/combatstream` (one event per combat happening, fights of a company against the enemies it fights in a room, a summary folded from the events), producers at every attack, cast, target change, flee, and death; the summary at a fight's end (`set battlesummary`); interceptors fall in the round they're struck; player help for combat (`help combat` and seven pages), pointed to from the tutorial |
| 29b2 | One battle at a time; spawn groups | Complete: `internal/battle` (each player, with their company, fights one enemy group at a time; other groups set on them hold back, then begin the next battle in the order they turned; a waiting group turns on a free player); `attack`/`cast`/`backstab`/`shoot` refuse a waiting group; hostile spawns form groups of two to five from the room's list, a lone survivor or straggler regroups, `solitary` bosses stand alone; travel ambushes are a pair; solo players get battles and summaries; help for every Ashveil system |
| 29c | Narration voice (weapons and spells) | Complete (branch `master-6csfy6`): [design](superpowers/specs/2026-09-28-phase-29c-narration-voice-design.md), [plan](superpowers/plans/2026-09-28-phase-29c-narration-voice.md). Every weapon line rewritten, `(N damage)` / `(critical hit, N damage)` / `, M blocked` at the end of each hit, only real crits draw the critical pool; no `***`, `!`, caps, or "prepares to fight"; "the" before common names; an opener per fight, "turns toward", death lines in order, a closing line after the last; the fallen notice indented and in words; spells chant with their rounds and land with `(N damage)` / `(N healed)`; `help narration` |
| 29d | Pronouns and ordinals | Proposed: [spec](superpowers/specs/2026-09-26-combat-pronouns-ordinals-design.md). Mob pronouns (beasts "it"); "the first/second cutthroat" fixed for the fight |
| 29e | Pain reactions | Proposed: [spec](superpowers/specs/2026-09-26-combat-pain-reactions-design.md). A victim's reaction after a non-lethal critical hit; a set per beast race |
| 29f | Paced combat output | Proposed: [spec](superpowers/specs/2026-09-26-combat-pacing-design.md). Round lines released over time, with fast, normal (~6s), slow, and off settings; combat resolves every 2 game rounds (an 8-second combat round) |
| 30a | Status effects and critical-hit effects (weapons only) | Proposed: [spec](superpowers/specs/2026-09-26-status-crit-effects-design.md). Bleeding, stagger, knockdown, armor break, and others as buffs; crit effects by weapon type; spells do not critical hit |
| 30b | Wounds, treatment, and `heal wounds` | Proposed: [spec](superpowers/specs/2026-09-26-wounds-treatment-design.md). Wound limits on healing; durable critical-hit wounds; one after-fight command using clerics, splints and bandages, or an inn physician; camp rest heals |
| 30c | Pre-fight tactics | Proposed: [spec](superpowers/specs/2026-09-26-company-tactics-design.md). `company tactics` (focus, healing, interrupts, guards, rotation, mercy); roles and personalities; guard reactions; companions casting; no commands mid-fight |
| 30d | Wind-ups, telegraphs, and interrupts | Proposed: [spec](superpowers/specs/2026-09-26-telegraphs-interrupts-design.md). Wind-ups in whole rounds, interrupt thresholds with accumulated pressure, concentration, enemy casters |
| 30e | Morale and mercy | Proposed: [spec](superpowers/specs/2026-09-26-morale-mercy-design.md). Temperaments (the undead never yield); yielded foes leave the fight; a spare/kill prompt at fight end; alignment and loyalty reactions; company nerve |
| 30f | Battlefield conditions | Proposed: [spec](superpowers/specs/2026-09-26-battlefield-conditions-design.md). Ambush and surprise, area attacks on clusters, leaping and flanking, narrow ground, fatigue and cold in combat, mounted combat |
| 31 | Browser battle panel | Proposed: [spec](superpowers/specs/2026-09-26-battle-panel-design.md). Enemy and company grids with target lines, from the event stream |
| 32a | Company polish | Complete (PR from `claude/project-thread-1buera`): [spec](superpowers/specs/2026-09-28-phase-32a-company-polish-design.md), [plan](superpowers/plans/2026-09-28-phase-32a-company-polish.md). No `♥friend` on companions; one arrival/departure line per company; no drink flourish; camp and fire in `look`; recruiters listed in the room; a readable formation grid |
| 32a2 | Per-player recruit rosters | Complete (PR #4 from `claude/project-thread-btmgj8`): [spec](superpowers/specs/2026-09-28-phase-32a2-recruit-rosters-design.md), [plan](superpowers/plans/2026-09-28-phase-32a2-recruit-rosters.md). Generated candidates on each player's own notice, coming and going; companions get their own names |
| 32b | Tutorial replay | Complete, in review: [spec](superpowers/specs/2026-09-28-phase-32b-tutorial-replay-design.md), [plan](superpowers/plans/2026-09-28-phase-32b-tutorial-replay.md). `tutorial replay yes` hands the connection to a throwaway level-1 copy (id from 900,000,000, unindexed) that runs the course; any way out hands it back to the real character, exactly as it was; `UserPurged` drops the copy from every module and removes its file; a restart sweeps leftovers |
| 32c | Enemy groups and `scout` | Complete: [design](superpowers/specs/2026-09-28-phase-32c-enemy-groups-design.md), [plan](superpowers/plans/2026-09-28-phase-32c-enemy-groups.md). Groups named as they form ("a band of ruffians") and shown on their own room line; `attack <group>` is the only way to start a fight; a battle plays out on its own (attack, cast, backstab, shoot, tackle, disarm refused in one; a bare `attack` after `break` rejoins); `look <group>` and a free `scout`; summaries name the group |
| 32f | Company logistics | Complete: [design](superpowers/specs/2026-09-28-phase-32f-company-logistics-design.md), [plan](superpowers/plans/2026-09-28-phase-32f-company-logistics.md). Capacity from members (20 kg + Strength), one pack each, and horses; weight the only limit (a full company takes on nothing more; walking never blocked); a herd of riding and pack horses bought at stables, with saddles; cargo keeps uses; `company inventory`; `company eat`/`drink`/`meal` |
| 32d, 32e, 32g, 32h | Play-test follow-ups | Proposed ([roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md)): automatic player and companion combat (32d), company XP (32e), web company dock (32g, takes 32f's GMCP extras), character deletion (32h, [design](superpowers/specs/2026-09-28-phase-32h-character-deletion-design.md), in progress) |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

Keep only the latest phase's entry here (What / Why / Verification /
**Review:**). When a new phase lands, replace the previous entry with it and
fold anything still true into "Known issues". Older entries live in git
history: `git log -p -- docs/PROJECT_STATUS.md` (the full log through
Phase 29b2 is at commit `d5ace46`).

### Phase 32f: company logistics (2026-09-28)

- **What:** per the [32f design](superpowers/specs/2026-09-28-phase-32f-company-logistics-design.md)
  and [plan](superpowers/plans/2026-09-28-phase-32f-company-logistics.md).
  Capacity is each member's share (20 kg, half a kilogram per Strength, and
  their largest pack: satchel 5, traveller's 10, frame 15 kg) plus the
  herd's; the flat 200 kg is retired. Weight is the only limit: GoMud's item
  count no longer slows `go`; `get`, `buy`, `market buy`, `give` from outside
  the company, pickpocketing, and a companion's pickup are refused when they
  would overfill it (moves within the company always work; walking is never
  blocked, only slowed by the load bands). A herd of up to one riding and one
  pack horse per member, bought at stables (Dunmar West Gate, Trappers'
  Post), each needing its own saddle; riders strain less, and a route goes a
  tenth faster only when everyone walking rides; the old single mount
  migrates to a saddled pack horse. Cargo keeps a partly used item's uses.
  `company inventory`; `company eat`/`drink`/`meal` feed the members present
  from the cargo, their own packs, then the leader's. `{I}` reads capacity
  in kg; `peep` shows weight. Help: `cargo`, `mount`, `encumbrance`,
  `inventory`, `get`, `buy`, `give`, `market`, `eat`, `drink`, `set-prompt`,
  `company`, new `company-inventory` and `company-meal`; Survival lesson hints.
- **Why:** the owner's play-test notes ([roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md)):
  "company load is 200 kg with no one in the company"; one weight limit.
  Built on `claude/project-thread-rxps20` before 32a2, 32b, 29c, and 32c;
  merged onto them here (additive conflicts in the company provider and its
  test fake; 32b's mount purge moved to herds).
- **Verification:** `go test -race ./...` (83 packages), `make generate`,
  and `make validate` pass, once, after the review fixes. Wiring:
  `modules/mount/wiring_logistics_test.go` (plugins.Load, real commands,
  restart), plus usercommands, mobcommands, company, encumbrance, market,
  expedition, and companyview tests.
- **Review:** 1 major, 6 minor bugs, 4 design/doc gaps, 6 test gaps,
  7 nits; all reproduced before fixing. Fixed with tests: (1) the `{I}`
  prompt token read company state off the game loop (now from
  `companyview`'s cache); (2) `mount stable`/`saddle`/`unsaddle`/`release`
  now save the leader right after the herd; (3) `company inventory` names
  generated recruits by their own names; (4) pickpocketing respects the
  limit and a pet's pouch counts in its owner's load (a full company can't
  take through its pet); (5) the riding pace counts only members walking
  with the leader (`company.WalkingMembers`); (6) meals feed only the
  members present; (7) a meal spends the item before provisioning (a failed
  spend fed for free; the design's crash note updated to match), and a live
  companion's meal updates its record. Docs: (8) the design now says GMCP
  drops `Max` and its extras move to 32g; (9) encumbrance and company
  AGENTS guides, the stale "capacity is flat" known issue, the plan's
  boxes; (10) `CarryCapacity()` marked deprecated; (11) the design notes
  that pre-32f characters drop to ~20–25 kg with no satchel (intended: the
  200 kg was the bug). Test gaps closed: a companion's pickup, `give` to
  your own companion and cargo moves at capacity, walking over capacity,
  the route pace through the mount seam, the meal's room line, kits under
  40%. Nits fixed: a pack counts the room it makes; `get all` says once
  what it left; `peep` shows weight; `help give`/`help market` name the
  refusals; the design now matches the shipped `company meal` hint.
  Rejected: a horse type configured without `Kind` is refused with a
  warning and shown as unrecognized (the project's validate-don't-guess
  rule); the legacy migration's free pack saddle is intended (no one loses
  what their old mount carried). Two riders' strain is covered by the
  walking relief test plus the mount seam test, not one end-to-end test.

## Known issues / deferred items

- **Enemy groups (32c), for the owner:** non-hostile mobs sharing a tag
  form groups (scout lists them as enemies); `formation move`, `flee`,
  `break`, potions, and `use` stay allowed mid-battle until 32d. Test gaps
  (in git history, 32c's work-log entry): a regrouped survivor's new name,
  a shopkeeper by its own name, the weakest reachable first aim,
  `groupnoun` loading, an authored `groupname`, a hidden member in
  look/scout, `scout` in the tutorial wiring test.
- **Company logistics (32f):** GMCP drops the inventory `Max` (the web
  gear window shows "count / —") until 32g adds capacity, packs, mounts,
  and cargo uses; a crash mid-meal can spend one use without its
  provision; characters made before 32f start at ~20–25 kg with no
  satchel.

Refreshed 2026-09-26 (Phase 28). Earlier entries that later phases
resolved (the weather, load, and mount multipliers, and the 11a–11c
combat wiring) are removed; each phase's work-log entry (in git history)
keeps its own history.

- **Narration (29c):** a player's own death line still comes from the
  queued `suicide` (a user script may cancel the death), so it can follow
  a closing line; untested: no closing on defeat or broken-off, a count of
  "turns toward" lines, `sparks` and `heal` through a real cast, the PvP
  "goes for" line.
- **Tutorial replay (32b), untested paths:** death through the real death
  module, quit through its buff path, link-dead expiry, a real copyover,
  and the purge in every module at once (each module has its own purge
  test). Accepted: a real character's login elsewhere lands before the
  stale replay's purge.
- **Company polish (32a) and rosters (32a2), accepted:** the company move
  line doesn't recheck that every companion made it (a `no-go` buff, a
  locked far door); a pet isn't named in it; an offline leader's camp
  reads "A camp"; recruiter and roster config are parsed on each read, so
  a malformed entry warns on each look; rosters for recruiter rooms later
  removed from config stay on the record; `party`/GMCP "Company" status
  has no dedicated test. 29b2's accepted items (waiting-group order and
  `battle.Retain` unit-tested only) stand.

- **Plugin persistence is a direct, non-atomic file write**
  (`WriteStruct`/`WriteBytes`). Command state rolls back on a failed save,
  but a partial low-level write can't be recovered.
- **Separate plugin files, no cross-file transaction.** Company, survival,
  expedition, camping, and encumbrance each persist separately. Durable
  operation IDs and applied-markers (expedition exertion, camp recovery,
  the survival ledger) make a crash between writes retry, not double-apply.
  Summon and dismiss are still not atomic across the company and survival
  files.
- **Copyover continuity** relies on plugin `SetOnSave`/`SetOnLoad`, because
  modules may not register copyover contributors. `onLoad` runs after
  `copyover.Restore` and reschedules or completes sessions.
- **Game-loop state without mutexes** (company, formation, survival,
  tutorial, the company view) assumes command dispatch stays on the main
  loop.
- **Live server acceptance** hasn't been run for most phases. Unit, race,
  wiring (`plugins.Load`), and browser (Playwright) tests cover them. The
  tutorial has no test of a real copyover or of two players in the course
  at once.
- **Enemy parties in displays (11a):** the room listing groups party
  members but passes zero EHP/DPS into `mobparty.Assemble`
  (`internal/rooms` can't import `internal/combat`), so display order isn't
  combat order. The GMCP room mob list is still per mob, not grouped by
  party. Combat itself assembles parties with real values.
- **Combat upkeep (29a) limits:**
  - A blocked `flee` is followed by the leader turning back onto a foe
    the next round.
  - GoMud's own `lookfortrouble` can still draw a group-hostile
    shopkeeper into a fight; the upkeep never does.
  - The upkeep's cost is per online leader with a companion present, and
    skips rooms with no aggro.
  - `formation reach`'s "can't reach any of the enemy" branch has no
    test.
- **Deferred by design, not scheduled:**
  - 11d: guard reactions, crit effects, wounds, AI personality;
  - the "12+" encounter kinds;
  - mount fatigue, feed, and terrain;
  - companion custom names and AI orders;
  - multi-player camps;
  - forecasts and seasons.
- **Combat event stream (29b) limits:** the stream lives in memory, so a
  restart mid-fight loses that fight's summary. Each company player's own
  summary waits for companies with more than one player.
- **Battles and spawn groups (29b2) limits:**
  - Battles are runtime only: after a restart a fight resumes from
    `Aggro` as a new battle, and rooms respawn and regroup.
  - A lone survivor or straggler with no idle group to join lingers
    alone; roaming groups that wander together are deferred.
  - The shipped `solitary` list (lich, abyssal creeper, ent, spider
    queen) is for the owner to check; the dark acolyte stays grouped.
  - The only shipped route (oak-road) has no ambush, so the ambush pair
    is reached only by tests and future routes.
- **Found writing the help pages (behaviour, not text):** a camp rest
  says it "will be interrupted", but nothing interrupts it; stabling a
  mount is free; Hunger or Thirst at 0 has no further effect; the
  `company` and `archetype` usage strings leave out some subcommands.
- `modules/tutorial`'s wiring test can't run twice in one process
  (`go test -count=2` fails the second run, before 29b too), and running
  it alongside `modules/company` in one `go test` with a high `-count` can
  collide on the shared config-overrides file.
- `make test`'s `js-lint` stage can stall on some hosts (`npx jshint`); `make
  js-lint` on its own and `go test -race ./...` are the documented checks.
- A shipped-world test can leave a gitignored
  `_datafiles/world/default/config-overrides.yaml`; delete a stray copy if
  wiring tests fail on a relative data path.

## Key documents

- `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` — authoritative design direction and
  agent working rules (section 50).
- `docs/superpowers/specs/` — designs for proposed work (the combat roadmap,
  29c–31) and the design records that code cites.
- `docs/superpowers/plans/` — the plan workflow (`README.md`) and the current
  plan template.
- Nested `AGENTS.md` files — how each package works today.
- Git history — the designs, plans, and work-log entries of finished phases
  (all present at commit `d5ace46`).
