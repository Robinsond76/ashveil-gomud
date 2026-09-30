# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-09-30
- **HEAD:** `master` 2026-09-30: the `modules/company` brawl test flakes
  fixed, merged from `fix-brawl-flakes` (test-only; see the work log).
  Before it, Phase 30b (wounds, treatment, and
  `heal wounds`), merged from `phase-30b-wounds`.
  Before it, on `master` 2026-09-29: Phase 30a (status and critical-hit
  effects), merged as `2ef931c` from `claude/next-phase-7ekckc`. Before it,
  Phase 29f (paced combat output),
  squash-merged as `fb28df3` from `claude/next-phase-planning-9u98ae`
  (branch since deleted). Before it, Phase 29e (pain reactions), integrated
  from `codex/phase-29e-pain-reactions`. Before that, Phase 32g2 (live battle view),
  merged from `phase-32g2-battle-view`, and Phase 32g (web company dock), merged from
  `phase-32g-company-dock`. Before that, Phase 29d (NPC pronouns and stable enemy
  labels), integrated from `codex/phase-29d-pronouns-ordinals`. Before that, Phase 32d (automatic combat by
  strategy), with 32c (enemy groups), 32f (company logistics), and 32h
  (character deletion), merged onto 32e (company experience, PR #10). Before them, Phase 29b2 (one battle at a time; spawn groups) and player
  help for every Ashveil system are complete and merged to `master`
  (2026-09-28, from `claude/next-phase-wfav4w`). Docs cleanup (finished-phase
  plans/specs and the old work log moved to git history) on
  `claude/docs-cleanup-py1rfb`.
  Play-test feedback (2026-09-28): the
  [roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md)
  and the 32a, 32a2, and 32b designs, on `claude/hopeful-wozniak-piu2lb`.
- **Upstream baseline:** `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`

## Current position

- **Completed:** Phases 0–29f, 30a, 30b, 32g,
  and 32g2; see the table below. The survival and
  expedition loop (travel, camping, weather, load, mounts), formation
  combat, the environment/skills/economy roadmap (13–21), the company-life
  and onboarding roadmap (22–27, including the tutorial), item weights (28),
  the first combat slices (29a, 29b, 29b2, 29c, 29d, 29e, 29f), enemy groups (32c),
  automatic combat by strategy (32d), company experience (32e), company
  logistics (32f), the web company dock (32g) and its live battle view
  (32g2), and character deletion
  (32h) are all done.
- **Latest completed phase:** 30b (wounds, treatment, and `heal wounds`),
  merged to `master` from `phase-30b-wounds`; see its work-log entry below. Before it, 30a
  (status and critical-hit effects), merged as `2ef931c`, and 29f
  (paced combat output), squash-merged to `master` as `fb28df3`, and before
  that 29e (pain reactions), from `codex/phase-29e-pain-reactions`, 32g2
  (live battle view), from `phase-32g2-battle-view`, and 32g (web company
  dock), from `phase-32g-company-dock`.
- **Before it:** 29d (pronouns and stable enemy labels), from
  `codex/phase-29d-pronouns-ordinals`; see the [plan](superpowers/plans/2026-09-28-phase-29d-pronouns-ordinals.md).
  NPC/race pronouns, generated recruit neutrality, frozen shared enemy labels,
  combat/script/event wiring, and player help/tutorial are verified. A seeded
  comparison against b9729809 preserves combat mechanics. Labels are runtime-only;
  restart/copyover starts fresh battles. Player pronoun selection remains deferred.
  **Review:** independent default-agent reviewer found fallback-label collisions and
  a Sparks possessive; both reproduced and were fixed with regression tests. A
  separate plain late-name collision claim was rejected because its regression passed
  before the fix. **Final verification:** `make generate`, `make validate`,
  `go test -race ./...`, `make js-lint`, and `git diff --check` all passed (2026-09-29);
  generation produced no diff. Workflow docs now use the current task's default
  agent/settings and direct implementation, retaining independent review and verification.
  Integration verification exposed a narration-test flake: a random dodge intentionally
  produces no room attack line. The fixture now disables dodging only within that test
  and restores settings afterward; 100 focused race-test repetitions passed.
  Independent follow-up review found no issues and passed 40 focused repetitions.
- **Next:** the play-test roadmap (32a–32h, 32g2) is done, per the
  [play-test roadmap](superpowers/specs/2026-09-28-playtest-feedback-roadmap.md#build-order-recommended-accepted-2026-09-28);
  the combat roadmap's 29 series is done with 29f, and 30a and 30b are done;
  next is 30c (company tactics). 30a's statuses count combat
  rounds; the other balance shifts 29f's cadence brought are still to
  retune (see Known issues).
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
| 29d | Pronouns and ordinals | Complete and integrated: [spec](superpowers/specs/2026-09-26-combat-pronouns-ordinals-design.md), [plan](superpowers/plans/2026-09-28-phase-29d-pronouns-ordinals.md). NPC/race pronouns, neutral generated recruits, stable shared enemy labels, help/tutorial; independent review and full race suite passed. Player selection deferred |
| 29e | Pain reactions | Complete: [design](superpowers/specs/2026-09-29-phase-29e-pain-reactions-design.md), [plan](superpowers/plans/2026-09-29-phase-29e-pain-reactions.md). A victim reacts after each damaging critical strike that leaves them standing; second person for the victim, third person for witnesses, distinct beast-race sets and NPC overrides, without changing combat mechanics |
| 29f | Paced combat output | Complete: [design](superpowers/specs/2026-09-29-phase-29f-paced-combat-design.md), [plan](superpowers/plans/2026-09-29-phase-29f-paced-combat.md). Combat resolves every `CombatEveryRounds` (2) game rounds, an 8-second combat round; everything a round causes is tagged (`events.WithCause`) and paced out per player (`internal/combatpace`), with `set combatpace fast|normal|slow|off` (off for screen readers); a pause before pain and death lines; typed output, tells, and says never wait; the prompt and the web client's vitals and battle view wait for the lines; flushes on move, quit, pace change, and copyover; `help combatpace` |
| 30a | Status effects and critical-hit effects (weapons only) | Complete (merged as `2ef931c`): [design](superpowers/specs/2026-09-29-phase-30a-status-crit-effects-design.md), [plan](superpowers/plans/2026-09-29-phase-30a-status-crit-effects.md). Nine statuses (bleeding, staggered, knocked down, armor broken, exposed, burning, overloaded, stunned, hobbled) as combat-round buffs (`internal/status`); crit effects by weapon subtype; ticked, and actions lost, in the combat round; cleared at fight end; Sparks overloads; `help statuses` |
| 30b | Wounds, treatment, and `heal wounds` | Complete: [design](superpowers/specs/2026-09-30-phase-30b-wounds-design.md), [plan](superpowers/plans/2026-09-30-phase-30b-wounds.md). Lasting wounds from damaging crits, light ones from crushing blows and finished bleeds; healing stops at the wound limit; `heal`/`heal wounds` (clerics' `tend` and heal, splints and bandages, the Waymark Inn physician); inn stay heals, camp rest one wound per splint or bandage (owner amendment); death clears; `status`, GMCP, web strip; `help wounds` |
| 30c | Company tactics | Proposed, rescoped 2026-09-29: [design](superpowers/specs/2026-09-29-phase-30c-company-tactics-design.md). 30c1: `company tactics` (company-wide focus, changeable mid-battle for that battle only, leader-only, one-round cooldown, with web Combat-tab buttons; healing threshold) and enemy personalities. 30c2: guardian role and guards. Rotate the wounded deferred |
| 30d | Wind-ups, telegraphs, and interrupts | Proposed: [spec](superpowers/specs/2026-09-26-telegraphs-interrupts-design.md). Wind-ups in whole rounds, interrupt thresholds with accumulated pressure, concentration, enemy casters |
| 30e | Morale and mercy | Proposed: [spec](superpowers/specs/2026-09-26-morale-mercy-design.md). Temperaments (the undead never yield); yielded foes leave the fight; a spare/kill prompt at fight end; alignment and loyalty reactions; company nerve |
| 30f | Battlefield conditions | Proposed: [spec](superpowers/specs/2026-09-26-battlefield-conditions-design.md). Ambush and surprise, area attacks on clusters, leaping and flanking, narrow ground, fatigue and cold in combat, mounted combat |
| 32a | Company polish | Complete (PR from `claude/project-thread-1buera`): [spec](superpowers/specs/2026-09-28-phase-32a-company-polish-design.md), [plan](superpowers/plans/2026-09-28-phase-32a-company-polish.md). No `♥friend` on companions; one arrival/departure line per company; no drink flourish; camp and fire in `look`; recruiters listed in the room; a readable formation grid |
| 32a2 | Per-player recruit rosters | Complete (PR #4 from `claude/project-thread-btmgj8`): [spec](superpowers/specs/2026-09-28-phase-32a2-recruit-rosters-design.md), [plan](superpowers/plans/2026-09-28-phase-32a2-recruit-rosters.md). Generated candidates on each player's own notice, coming and going; companions get their own names |
| 32b | Tutorial replay | Complete, in review: [spec](superpowers/specs/2026-09-28-phase-32b-tutorial-replay-design.md), [plan](superpowers/plans/2026-09-28-phase-32b-tutorial-replay.md). `tutorial replay yes` hands the connection to a throwaway level-1 copy (id from 900,000,000, unindexed) that runs the course; any way out hands it back to the real character, exactly as it was; `UserPurged` drops the copy from every module and removes its file; a restart sweeps leftovers |
| 32c | Enemy groups and `scout` | Complete: [design](superpowers/specs/2026-09-28-phase-32c-enemy-groups-design.md), [plan](superpowers/plans/2026-09-28-phase-32c-enemy-groups.md). Groups named as they form ("a band of ruffians") and shown on their own room line; `attack <group>` is the only way to start a fight; a battle plays out on its own (attack, cast, backstab, shoot, tackle, disarm refused in one; a bare `attack` after `break` rejoins); `look <group>` and a free `scout`; summaries name the group |
| 32e | Company experience | Complete: a kill that pays the leader pays every living companion still charmed by them in the kill room the same figure in full (no split), in both the solo and party branches; companions level up live (spending the level's points, so a respawn changes nothing); `experience` lists each companion; `company.MemberView`/`companyview.Member` carry live level and progress. **Verification (2026-09-28):** `go test -race ./...`, `make generate`, `make validate` green. **Review:** Opus reviewer found no blocking bugs. Fixed: a levelled companion kept unspent points until respawn (now `AutoTrain`); a companion charmed away was still paid (now requires `IsCharmed(leader)`); the design contradicted the code on rows for absent companions; dead `Leader` progress fields removed; party-branch and befriended-away regression tests added. Rejected/deferred: full seam test of logout/copyover (the snapshot/restore round-trip is tested, the seams are 22b's), a practice-mob companion test (mobs vanish before any XP), mid-fight level-up refill (accepted). A rare failure of the combat brawl test under `-count=4` was traced to a level-up resetting the test's forced health; the brawl now runs with kill XP off. |
| 32f | Company logistics | Complete: [design](superpowers/specs/2026-09-28-phase-32f-company-logistics-design.md), [plan](superpowers/plans/2026-09-28-phase-32f-company-logistics.md). Capacity from members (20 kg + Strength), one pack each, and horses; weight the only limit (a full company takes on nothing more; walking never blocked); a herd of riding and pack horses bought at stables, with saddles; cargo keeps uses; `company inventory`; `company eat`/`drink`/`meal` |
| 32h | Character deletion | Complete: [design](superpowers/specs/2026-09-28-phase-32h-character-deletion-design.md), [plan](superpowers/plans/2026-09-28-phase-32h-character-deletion.md). `delete character`, confirmed by the password (masked) and the name; every module's state purged with the login kept; back in creation on the same connection; a durable `Deleting` flag and a boot sweep; masked in-game password prompts (also `password`) |
| 32d | Automatic combat by strategy | Complete: [design](superpowers/specs/2026-09-28-phase-32d-auto-combat-design.md), [plan](superpowers/plans/2026-09-28-phase-32d-auto-combat.md). `strategy`: each character's role (fighter, healer, caster) and target rule (weakest, strongest, wounded, nearest, furthest, leader, assist, defend), durable; healers and casters cast real spells with mana; companions know spells by archetype and level and regain mana; wizards/clerics granted Magic Missile/Minor Heal; in a battle only `flee`; only hostile mobs group by tag |
| 32g | Web company dock | Complete: [design](superpowers/specs/2026-09-29-phase-32g-company-dock-design.md), [plan](superpowers/plans/2026-09-29-phase-32g-company-dock.md). Left column the world (time, map, room, tutorial); right a tabbed dock: a vitals strip for every member above Character (Overview with worth, Gear with weights, Skills and jobs, Quests, Effects, Pet), Company (Status, Inventory with menus by exact item reference, Camp), Combat (setup: roles, targets, formation, Scout), Comm (unread count), and Who/Kills when enabled; `Company.Inventory`, `Company.Camp`, members' mana and strategies; `help webclient` |
| 32g2 | Live battle view | Complete: [design](superpowers/specs/2026-09-29-phase-32g2-battle-view-design.md), [plan](superpowers/plans/2026-09-29-phase-32g2-battle-view.md). During a battle the Combat tab shows the enemy group's formation above the company's (fronts to the middle), scout's health words (never numbers), reach, target lines both ways, outsiders struck, the fallen, the waiting groups, a text list, a live region, Flee, and Setup's member menu; a marker on the tab; `Company.Battle` for every player in a battle; dark rooms show nothing, as scout |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

Keep only the latest phase's entry here (What / Why / Verification /
**Review:**). Older entries live in git history: `git log -p --
docs/PROJECT_STATUS.md`.

### Brawl test flakes fixed (2026-09-30, not a phase)

- **What:** the `modules/company` brawl wiring tests failed in about 1 of
  3 full package runs (7 of 30 on `master` here). Each failure was
  reproduced and root-caused; the fixes are test-only, and no game
  behaviour changed.
  - **Leaked grudges:** a fight makes the bandits' group hostile to Aria
    (user 7) for many rounds in `mobs` package state, so the next brawl's
    "non-hostile" bandits attacked at once (companions untracked or not
    healing in `TestMinorHealCombatPronouns`). New test helper
    `mobs.ResetHostility`; `newBrawl` resets before and after.
  - **Companions levelling:** a level-1 mob spawns holding level 2's
    experience (`NewMobById` sets `XPTL(0)`, which clamps to `XPTL(1)`),
    so with kill XP off its first kill still levels it and refills its
    health (`TestPacedCombatThroughTheRealRound`: the 1-HP companion
    never fell). `newBrawl` starts level-1 companions at 0 experience.
  - **Fizzles:** every cast has a 1% fizzle (a roll of 100 fails even a
    100% chance), so no skill can guarantee one. The Minor Heal test
    recasts after a fizzle; the Sparks test's loop now waits for
    Overloaded itself (it stopped on any status, such as a companion's
    stagger, after a fizzled cast).
  - **Kill speed:** `TestBattleViewPerPlayer` toughens the waiting bandits
    (a crit from Brom ended his battle in the round it began, before the
    test looked), and `TestPronounsAndOrdinalsThroughRealRound` keeps the
    second cutthroat standing until it has swung after the first's death.
  - **A last-round crit:** `TestLightWoundsCloseWithTheFight` turns crits
    off, so the final round can't add a lasting wound beside the fracture.
- **Verification:** before, 7 of 30 full `./modules/company` runs failed.
  After: each fixed test passed 200-300 times in a loop, and 30 of 30
  full package runs passed. `go test -race ./...`, `make generate` (no
  diff), and `make validate` passed (2026-09-30).
- **Review:** no phase review gate (test-only fix); the diff was read for
  weakened assertions (none: each still asserts the same text and events).

### Phase 30b: wounds, treatment, and `heal wounds` (2026-09-30)

- **What:**
  - `internal/wounds` (pure): wounds, the wound limit (max less the
    wounds, never under a quarter of max), wounds from crits, crushing
    blows, and finished bleeds, treatment, and the `heal wounds` planner.
  - `Character.Wounds`, persisted in the user file. `Heal`,
    `ApplyHealthChange`, `SetHealth`, and the level-up refill stop at the
    limit, and never lower health.
  - Combat: a damaging crit on a player or a companion leaves a lasting
    wound (`, wounded` in the hit line), and a crushing blow a light one.
    A bleed that runs out leaves a light wound. Light wounds close at
    fight end, in the stray pass, and at login. The strategy healer reads
    the limit.
  - Companions' wounds ride `MemberState`. They respawn at the limit, and
    death clears the wounds.
  - `heal` and `heal wounds` (`modules/company/wounds.go`): clerics tend
    and heal, then splints and bandages, then a physician (config
    `Physicians`, the Waymark Inn at 15 gold a wound). The physician asks
    through the prompt. Items 36 (bandage) and 37 (splint) are in both
    markets.
  - The new `tend` spell (clerics, companions at level 1). Heal spells
    note a held-back heal.
  - An inn stay knits every wound. A camp rest knits a broken bone per
    birch splint and a cut or puncture per linen bandage the company has,
    using them up; without them it heals none (owner amendments,
    2026-09-30, after merge, on branches `camp-wounds-bandages` and
    `camp-splints`). The church wake clears wounds.
  - `status` `(limit N)`, `companyview`/GMCP `hp_limit`, and the web
    vitals strip.
  - Help: `help wounds` (new, with aliases), and `help heal` rewritten
    from GoMud's stale skill page. `statuses`, `combat`, `camp`, `inn`,
    `death`, and `health` are updated. The Camp lesson has a pointer.
- **Why:** step 6 of the combat roadmap. Owner decisions (2026-09-30):
  - lasting and light wounds;
  - camp rest and inn stay both heal. Amended the same day: a camp rest
    heals only with bandages, one per wound, and a broken bone needs a
    splint;
  - `tend` is a castable spell;
  - death clears wounds.
- **Verification:**
  - Wiring tests through the real entry points:
    - crits through `Attack*`;
    - `DoCombat` for bleeds, fight end, strays, and the strategy healer;
    - real casts of heal and tend;
    - the `heal` command and its physician prompt;
    - the camping grant;
    - the death module;
    - the save/logout/restart/respawn seams (companion);
    - user save and load (player);
    - `HandleJoin`;
    - the Playwright vitals strip.
  - A mutation check confirmed the fight-end close is what the fight-end
    test catches.
  - Final (2026-09-30, after the review fixes): these all passed.
    - `make generate` (no diff);
    - `make validate`;
    - `make js-lint`;
    - `go test -race ./...`.
  - A first full run caught two things:
    - `internal/scripting`'s structured type list lacked the new actor
      methods (fixed);
    - two intermittent brawl tests.
      - `TestMinorHealCombatPronouns` fails on a random fizzle, 2 in 60
        runs on `master` too.
      - `TestBattleViewPerPlayer` failed 2 in 180 here and 0 in 210 on
        `master`. Nothing in 30b touches how waiting groups choose a
        player, so it's recorded as the brawl's dice, not fixed.
- **Review:** the independent default-agent reviewer found no blockers.
  Each of its findings was checked.
  - **Fixed, with regression tests:**
    - no vitals refresh after `heal wounds` or rest healing;
    - light wounds could carry into a fight begun in the first round after
      login (now closed in `HandleJoin`);
    - `tend` spent mana on a target with no wound;
    - the physician asked a leader who couldn't pay;
    - "first, the worst hurt" narration after self-treatment;
    - spent-healer and no-mana messages;
    - help wording (items go on "until it closes"; the physician only
      through `heal wounds`);
    - the physician is now in the Waymark Inn's room text.
  - **Tests added for the gaps it named:**
    - the physician's price re-check, a fight between the offer and the
      answer, and the companion record after paying;
    - companion-only aggro;
    - a player cleric healing to the limit;
    - an enemy's bleed leaving no wound;
    - a real user save and load.

    The fight-end test couldn't tell `fightSides.end` from the stray pass,
    and was rewritten.
  - **Accepted:**
    - bandages and splints keep going until a wound closes (documented);
    - healing from a rest or `heal wounds` reaches disk at the next save
      seam, as rest buffs already do (documented);
    - potion scripts are covered by the `ApplyHealthChange` unit tests,
      not a buff-script test.

## Known issues / deferred items

- **Wounds (30b), deferred:**
  - Enemies take no wounds, and there are no monster abilities that
    wound.
  - No wounds on the battle-view grid or in the battle summary, and no
    stream event for them.
  - Pets' strikes don't wound.
  - A companion not out when a rest completes keeps its wounds.
  - The physician's price is flat, with no standing markup.
  - Healing reaches disk at the next save seam.

- **Paced combat (29f), for the owner and Phase 30:**
  - Game-round systems now tick twice per combat round:
    - regeneration heals more between blows;
    - a fight costs about twice the survival;
    - round-timed buffs last half as many combat rounds;
    - bleeding out gives fewer combat rounds to save someone.

    The owner accepted this for now, to retune in Phase 30.
    `CombatEveryRounds: 1` reverts it.
  - A movement command's own first line can precede the flush its move
    triggers.
  - The Party GMCP payload isn't held.
  - Held lines aren't saved, so a crash loses at most one round's
    unshown text.

- **Status effects (30a), accepted:** a status struck in the last round of
  one battle lands if the next battle has already begun against the same
  player (who is then in that fight). No sling-specific stagger, shield stun,
  or fire spell for Burning yet (a weapon override can burn). Mobs are not
  held by `no-flee`, and natural (generic) attacks leave no status. (The
  brawl tests' intermittent failures are fixed; see the work log.)

- **Found fixing the brawl flakes, for the owner (behaviour, not fixed):**
  a level-1 mob, companions included, spawns with level 2's experience, so
  it levels on its first kill whatever the kill pays; and every cast
  fizzles on a roll of 100, even at a 100% chance.

- **Live battle view (32g2), deferred:** casting, statuses, wounds, and guards on the grid (Phase 30); the
  battle summary in the view; exact enemy numbers behind a skill. (29f
  now holds the view until a round's lines are out.) A hidden enemy that
  joins and falls within one round, before the view ever looks, is named
  under Fallen (accepted: the view names every foe it never saw hidden).

- **Automatic combat (32d), for the owner and later phases:** Magic
  Missile's difficulty 75 gives a new wizard about a one-in-three chance
  to cast (GoMud's odds; spell balance not retuned); companion mana and
  health aren't saved (a restart refills both); enemies still aim at the weakest they can reach (personalities are
  30c's); companions following a flee is untested. 32c's two owner items
  (peaceful tag groups; what's allowed mid-battle) are settled by 32d.
  32c's test gaps are in its work-log entry (git history, commit
  `ff1f663e`).
- **Company logistics (32f)** (work log at commit `bea052b`): a crash
  mid-meal can spend one use without its provision; characters made
  before 32f start at ~20–25 kg with no satchel. (The web gear window's
  "count / —" is fixed by 32g: it shows weights.)
- **Character deletion (32h), accepted** (work log at commit `f418e1d`): a replay's hand-back resets the
  wrong-password count; copyover with a flagged user online races the
  resumed connection; a failed reset waits for the boot sweep; the reused
  user id keeps references elsewhere (mob XP credit, party invites,
  charms outside the room); web macros can answer the password question.

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
  29c–30f) and the design records that code cites.
- `docs/superpowers/plans/` — the plan workflow (`README.md`) and the current
  plan template.
- Nested `AGENTS.md` files — how each package works today.
- Git history — the designs, plans, and work-log entries of finished phases
  (all present at commit `d5ace46`).
