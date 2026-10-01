# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-10-01
- **Latest completed phase:** 30e, morale and mercy (2026-10-01).
- **Upstream baseline:** `39e44013` (GoMud Telnet input-masking fix).

## Current position

The expedition/company loop, onboarding, combat presentation (29a–29f),
status effects, wounds, tactics, guardians, interrupts, and wind-ups
(30a–30e) are shipped. The Phase 32 play-test improvements are implemented;
32b's status still carries an outstanding review note (see its retained plan).

**Next:** Phase 30g, [combat tempo, personal load, and active
defense](designs/2026-09-30-phase-30g-tempo-defense-design.md), whose
decisions the owner settled on 2026-09-30. 30g1 (the balance harness and
baseline) is done; each later slice is measured against it:

1. **30g2, active defense and armor:** one defense per strike: block
   with a shield (no dodge), else parry a melee strike or dodge; the
   shield's ×1.5 armor removed; the shield bash moves to a blocked melee
   strike (5–20% by Strength); armor's suffix reads `absorbed`;
   `help defense`.
2. **30g3, personal load:** worn and carried weight against a
   Strength-based capacity (cargo and mounts never count); burden lowers
   dodge; burden words in `status`.
3. **30g4, progression:** automatic stats grow in steps every 5 levels;
   HP by archetype, in small numbers; an XP knee at level 60.
4. **30g5, the action meter:** turns from raw Speed and burden, at most
   two a round, no banking.
5. **30g6, tuning:** HP, damage, and healing set so a no-focus even 5v5
   lasts 10–15 rounds at levels 1–60; the harness asserts it. It also
   takes 29f's cadence retune.

Phase 30e (morale and mercy) is complete. Phase 30f (battlefield
conditions; the owner removed mounted combat from it on 2026-09-30)
remains queued; the owner chooses where it falls among the 30g slices.
Other remaining limitations are listed below.

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
| 11d | Guard reactions, crit effects, wounds, AI personality | Superseded by shipped Phases 30a–30c2 |
| 12a | Encounter-kind abstraction | Complete: `InterruptionKind` widened to `fallen-tree`/`discovery`/`tracks`, data-driven text lookup |
| 12b | Weighted encounter tables | Complete: `InterruptionProfile.Kinds` weighted-roll form, resolved once at fire time; no shipped route uses it yet |
| 12c | Combat encounters | Complete: `Combat` interruption kind spawns its route's `CombatMobID` into the origin room and commands it to attack; resume/return gated on the mob still being alive and present |
| 13 | Sky and environment | Complete: moon phases, cloud cover, fog, indoor biomes/tags, indoor glimpse, richer `weather`; display-only |
| 14 | Visibility and light | Complete: ambient vs per-viewer light, personal/fixture/party light, darkness hit penalty, `light` command, `floatinglight` spell |
| 15 | Temperature, clothing, exposure | Complete: `internal/climate`, `modules/exposure`, item `warmth`, weather `TemperatureMod`, survival member drain + mutex, `temperature` command |
| 16 | Walking fatigue, inns, travel/rest multipliers | Complete: `internal/walking`, `modules/walking`, inn stays in `modules/camping`, multipliers locked at departure/rest start, Waymark Inn, Old Kings Road zone |
| 17 | Archetypes (17a) and utility skills (17b) | Complete: `internal/archetypes`, `modules/archetype`, training and spell gating, companion archetypes, `autoskill`, `trap`, auto-light. Cooking shipped in Phase 18b |
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
| 29c | Narration voice (weapons and spells) | Complete (branch `master-6csfy6`): [design](designs/2026-09-28-phase-29c-narration-voice-design.md). Every weapon line rewritten, `(N damage)` / `(critical hit, N damage)` / `, M blocked` at the end of each hit, only real crits draw the critical pool; no `***`, `!`, caps, or "prepares to fight"; "the" before common names; an opener per fight, "turns toward", death lines in order, a closing line after the last; the fallen notice indented and in words; spells chant with their rounds and land with `(N damage)` / `(N healed)`; `help narration` |
| 29d | Pronouns and ordinals | Complete and integrated: [spec](designs/2026-09-26-combat-pronouns-ordinals-design.md). NPC/race pronouns, neutral generated recruits, stable shared enemy labels, help/tutorial; independent review and full race suite passed. Player selection deferred |
| 29e | Pain reactions | Complete: [design](designs/2026-09-29-phase-29e-pain-reactions-design.md). A victim reacts after each damaging critical strike that leaves them standing; second person for the victim, third person for witnesses, distinct beast-race sets and NPC overrides, without changing combat mechanics |
| 29f | Paced combat output | Complete: [design](designs/2026-09-29-phase-29f-paced-combat-design.md). Combat resolves every `CombatEveryRounds` (2) game rounds, an 8-second combat round; everything a round causes is tagged (`events.WithCause`) and paced out per player (`internal/combatpace`), with `set combatpace fast|normal|slow|off` (off for screen readers); a pause before pain and death lines; typed output, tells, and says never wait; the prompt and the web client's vitals and battle view wait for the lines; flushes on move, quit, pace change, and copyover; `help combatpace` |
| 30a | Status effects and critical-hit effects (weapons only) | Complete (merged as `2ef931c`): [design](designs/2026-09-29-phase-30a-status-crit-effects-design.md). Nine statuses (bleeding, staggered, knocked down, armor broken, exposed, burning, overloaded, stunned, hobbled) as combat-round buffs (`internal/status`); crit effects by weapon subtype; ticked, and actions lost, in the combat round; cleared at fight end; Sparks overloads; `help statuses` |
| 30b | Wounds, treatment, and `heal wounds` | Complete: [design](designs/2026-09-30-phase-30b-wounds-design.md). Lasting wounds from damaging crits, light ones from crushing blows and finished bleeds; healing stops at the wound limit; `heal`/`heal wounds` (clerics' `tend` and heal, splints and bandages, the Waymark Inn physician); inn stay heals, camp rest one wound per splint or bandage (owner amendment); death clears; `status`, GMCP, web strip; `help wounds` |
| 30c | Company tactics | 30c1 complete: [design](designs/2026-09-29-phase-30c-company-tactics-design.md). `company tactics` (`tactics`): a durable company focus (none, leader, casters, nearest, weakest, strongest, wounded) and healing threshold (10–90%); in a battle only the focus changes, for that battle, one order a round, turning everyone at the next upkeep; web focus buttons; the `casters` rule; enemy personalities by race or template (`targeting`, `targetingnoise`); `help tactics`. 30c2 complete: [design](designs/2026-09-30-phase-30c2-guardian-design.md). A `guardian` role (`strategy <who> guard [other]`) steps in for its ward (set, else the most hurt in reach) within one column: 2 guards a battle, one back per 2 combat rounds, none while knocked down or stunned; `help guardian`. Rotate the wounded deferred |
| 30d | Wind-ups, telegraphs, and interrupts | 30d1 complete: [design](designs/2026-09-30-phase-30d1-chant-interrupts-design.md). A weapon blow that draws blood breaks a chant (owner: no pressure meter); 30d1b ([design](designs/2026-09-30-phase-30d1b-chant-break-chance-design.md)) made that a chance (40–90% by damage against max health; a crit, stagger, knockdown, or stun always); a company caster loses the spell with half its mana back, an enemy restarts from the first word; shield counters on a missed melee blow (50%, 1d4, stun 25%, once a round); the goblin hexer and Withering Hex in the Dark Forest; `help interrupts`. 30d2 complete: [design](designs/2026-09-30-phase-30d2-windups-design.md). Physical wind-ups (`internal/windup`, mob `windups`): the forest ogre (Dark Forest room 530) winds up Crushing Blow in plain view, then one swing of double damage that knocks down; only a crit that lands, a stagger, a knockdown, or a stun breaks it (owner); a broken or landed one is followed by 2 turns' cooldown; a shield bash is a counter strike only and breaks nothing (owner) |
| 30e | Morale and mercy | Complete: enemy temperaments and shared break checks; protected surrender; paced mercy decisions, rewards and reactions; companion hesitation, flight and saved return |
| 30f | Battlefield conditions | Proposed: [spec](designs/2026-09-26-battlefield-conditions-design.md). Ambush and surprise, area attacks on clusters, leaping and flanking, narrow ground, fatigue and cold in combat |
| 30g | Combat tempo, personal load, and active defense | In progress: [design](designs/2026-09-30-phase-30g-tempo-defense-design.md). Owner-agreed direction (2026-09-30): an action meter from Speed and personal load (capped turns a round); block with a shield (no dodge), parry or dodge without; the shield's ×1.5 armor removed; a 10–15 round 5v5 target. Shield bash on a block (5–20% by Strength); no level cap, an XP knee at ~60; small HP numbers by archetype; stats grow in steps every few levels, and tempo reads raw Speed; the 10–15 rounds is the no-focus fight. Slices: 30g1 balance harness, 30g2 defenses, 30g3 personal load, 30g4 progression (stat steps, archetype HP, XP knee), 30g5 meter, 30g6 tuning. All decisions settled (A, B, D by the lead's recommendation; stat steps every 5 levels). 30g1 complete ([plan](plans/2026-09-30-phase-30g1-balance-harness.md)): the balance harness and baseline (no-focus 5v5: median 8 / 32 / 68 rounds at levels 1 / 5 / 10); 30g2 (defenses) next |
| 32a | Company polish | Complete (PR from `claude/project-thread-1buera`): [spec](designs/2026-09-28-phase-32a-company-polish-design.md). No `♥friend` on companions; one arrival/departure line per company; no drink flourish; camp and fire in `look`; recruiters listed in the room; a readable formation grid |
| 32a2 | Per-player recruit rosters | Complete (PR #4 from `claude/project-thread-btmgj8`): [spec](designs/2026-09-28-phase-32a2-recruit-rosters-design.md). Generated candidates on each player's own notice, coming and going; companions get their own names |
| 32b | Tutorial replay | Complete, in review: [spec](designs/2026-09-28-phase-32b-tutorial-replay-design.md), [plan](plans/2026-09-28-phase-32b-tutorial-replay.md). `tutorial replay yes` hands the connection to a throwaway level-1 copy (id from 900,000,000, unindexed) that runs the course; any way out hands it back to the real character, exactly as it was; `UserPurged` drops the copy from every module and removes its file; a restart sweeps leftovers |
| 32c | Enemy groups and `scout` | Complete: [design](designs/2026-09-28-phase-32c-enemy-groups-design.md). Groups named as they form ("a band of ruffians") and shown on their own room line; `attack <group>` is the only way to start a fight; a battle plays out on its own (attack, cast, backstab, shoot, tackle, disarm refused in one; a bare `attack` after `break` rejoins); `look <group>` and a free `scout`; summaries name the group |
| 32e | Company experience | Complete: a kill that pays the leader pays every living companion still charmed by them in the kill room the same figure in full (no split), in both the solo and party branches; companions level up live (spending the level's points, so a respawn changes nothing); `experience` lists each companion; `company.MemberView`/`companyview.Member` carry live level and progress. **Verification (2026-09-28):** `go test -race ./...`, `make generate`, `make validate` green. **Review:** Opus reviewer found no blocking bugs. Fixed: a levelled companion kept unspent points until respawn (now `AutoTrain`); a companion charmed away was still paid (now requires `IsCharmed(leader)`); the design contradicted the code on rows for absent companions; dead `Leader` progress fields removed; party-branch and befriended-away regression tests added. Rejected/deferred: full seam test of logout/copyover (the snapshot/restore round-trip is tested, the seams are 22b's), a practice-mob companion test (mobs vanish before any XP), mid-fight level-up refill (accepted). A rare failure of the combat brawl test under `-count=4` was traced to a level-up resetting the test's forced health; the brawl now runs with kill XP off. |
| 32f | Company logistics | Complete: [design](designs/2026-09-28-phase-32f-company-logistics-design.md). Capacity from members (20 kg + Strength), one pack each, and horses; weight the only limit (a full company takes on nothing more; walking never blocked); a herd of riding and pack horses bought at stables, with saddles; cargo keeps uses; `company inventory`; `company eat`/`drink`/`meal` |
| 32h | Character deletion | Complete: [design](designs/2026-09-28-phase-32h-character-deletion-design.md). `delete character`, confirmed by the password (masked) and the name; every module's state purged with the login kept; back in creation on the same connection; a durable `Deleting` flag and a boot sweep; masked in-game password prompts (also `password`) |
| 32d | Automatic combat by strategy | Complete: [design](designs/2026-09-28-phase-32d-auto-combat-design.md). `strategy`: each character's role (fighter, healer, caster) and target rule (weakest, strongest, wounded, nearest, furthest, leader, assist, defend), durable; healers and casters cast real spells with mana; companions know spells by archetype and level and regain mana; wizards/clerics granted Magic Missile/Minor Heal; in a battle only `flee`; only hostile mobs group by tag |
| 32g | Web company dock | Complete: [design](designs/2026-09-29-phase-32g-company-dock-design.md). Left column the world (time, map, room, tutorial); right a tabbed dock: a vitals strip for every member above Character (Overview with worth, Gear with weights, Skills and jobs, Quests, Effects, Pet), Company (Status, Inventory with menus by exact item reference, Camp), Combat (setup: roles, targets, formation, Scout), Comm (unread count), and Who/Kills when enabled; `Company.Inventory`, `Company.Camp`, members' mana and strategies; `help webclient` |
| 32g2 | Live battle view | Complete: [design](designs/2026-09-29-phase-32g2-battle-view-design.md). During a battle the Combat tab shows the enemy group's formation above the company's (fronts to the middle), scout's health words (never numbers), reach, target lines both ways, outsiders struck, the fallen, the waiting groups, a text list, a live region, Flee, and Setup's member menu; a marker on the tab; `Company.Battle` for every player in a battle; dark rooms show nothing, as scout |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

### Phase 30e: morale and mercy (2026-10-01)

- Shipped original-group morale checks, race/template temperaments, protected
  surrender and fleeing, and stable shared ownership across player battles.
  Mercy questions wait for the paced summary, expire after 30 seconds, and
  release unresolved prisoners on departure. Spare/execution use exact ±5
  alignment, witnessed loyalty reactions, and execution's ordinary rewards.
- Shipped one-check companion nerve, one-action hesitation and chant refunds,
  durable flight/gear/return with one loyalty penalty, and zero-loyalty departure.
  Mercy recovery saves effect receipts without restoring prisoners or replaying
  rewards; offline retries load the latest saved character.
- Added GMCP/browser presentation, indexed morale/mercy help and tutorial hints.
  [Implementation record](plans/2026-09-30-phase-30e-implementation.md).
- **Review:** independent full-diff review and follow-up found failed-save
  prisoner abandonment, shared-owner transfer, same-round ownership ordering,
  zero-loyalty departure, and stale offline character saves. Fixed all with
  regression coverage; final reviewer found no remaining blocker. The pacing
  concern was withdrawn after checking queued delivery and its regression.
- **Integration:** preserved incoming Phase 30g1 balance-harness work when
  merging `origin/master`; independent integration review found no blocker.
  Generation, validation and the full race suite pass on the combined tree.
- **Verification:** `make generate`, `make validate`, `go test -race ./...`,
  focused mobcommands/company regressions, JS/Lua lint and Chromium dock checks
  all pass. JSHint used the installed executable instead of fetching with npx. Browser checks used the unchanged harness over local HTTP because this
  environment blocks file URLs. The first full race run exposed death-guard
  compatibility failures; the guard now uses a transient per-instance marker.

### Phase 30g1: balance harness and baseline (2026-09-30)

- **What:** the first slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-09-30-phase-30g1-balance-harness.md)),
  test-only (no player-visible change, so no help).
  - `modules/company/balance_test.go`: an even 5v5 through the real
    `DoCombat` with real dice and 30a's statuses: Aria and four
    companions against a mirror group with their kits, both levelled
    alike (`levelTo`). `TestBalance5v5` (gated by `ASHVEIL_BALANCE=1`,
    `ASHVEIL_BALANCE_FIGHTS`, default 50) logs a table over levels 1/5/10
    × company modes spread/default/focus × enemy modes spread/default.
  - Always on: `TestBalanceHarnessRunsAFight`, `TestBalanceSidesStayEven`,
    `TestBalanceStatusesLand`, `TestBalanceTally`,
    `TestBalancePercentile`.
  - The design gained the owner's decisions 10–18 (shield bash on a
    block, no level cap and an XP knee, small archetype HP, stats in
    steps of 5 levels, the spread-out fight as the baseline, parry,
    `absorbed`) and six slices; its 30g1 section is amended to the build.
- **Why:** decision 9: measure before changing combat, so each later 30g
  slice is judged against numbers.
- **Baseline** (the full table is in the plan): the no-focus fight
  (spread × spread) runs a median of **8 rounds at level 1, 32 at level
  5, 68 at level 10** (p90 13 / 46 / 92), against the 10–15 target.
  Fights lengthen about 4× by level 5 and 8.5× by level 10, because HP
  grows with level and weapon damage doesn't: the owner's concern,
  measured. Other readings:
  - The company wins 60% / 66% / 84% of even fights at levels 1 / 5 /
    10: its healer, first strike, and slightly more HP for the player
    (recorded asymmetries).
  - Fighters take a turn in about three rounds of four (0.72–0.85 turns
    per standing fighter per round: casting, lost actions, and falls).
    About 38% of turns land (dodges count as misses), and about 15% of
    those are crits.
  - Shipped targeting (everyone the weakest) is already focus fire; an
    enemy that focuses makes fights longer and closer (level 10: 80
    rounds and 54% against 68 and 84%), not shorter.
- **Verification:** the harness tests looped 3 times while fixing; each
  review fix's test was checked to fail without it
  (`TestBalanceSidesStayEven`, `TestBalanceStatusesLand`). Final
  (2026-09-30, after the review fixes): `make generate` (no diff),
  `make validate`, and `go test -race ./...` all passed, and passed again
  after merging `master` (30d2 and the docs reorganisation) into the
  branch. The gated table runs in about 2–3 minutes at 50 fights a cell.
- **Review:** the independent default-agent reviewer checked fairness,
  the tally, the leader clamp, the real paths, and the invariants (clock,
  leaks, concurrency: all fine); each finding was checked.
  - **Fixed, with regression tests:** template experience levelled
    Garrick and Ysolde on their first kill at level 1 (`levelTo` now
    sets experience and peak level; `TestBalanceSidesStayEven`, which
    failed before the fix, also pins the player HP quirk); shield bashes
    were counted as turns and hits (now counted apart; `TestBalanceTally`);
    status-tick damage wasn't counted (now for the other side). The lead
    found, following the review's tick finding, that no status ever
    landed in the brawl world (no buffs loaded, no `Buff` listener): the
    harness now loads 30a's statuses and the game's listener
    (`TestBalanceStatusesLand`, which fails without it).
  - **Amended in the docs:** the baseline cell is spread × spread (the
    shipped default already focuses); comments on enemy spread and
    regeneration; the plan's modes, names, and cells; the design's 30g1
    section (unseeded, 50 fights, stalls at 200, per-side counts).
  - **Recorded, not fixed (they are the game as it is; 30g6 weighs
    them):** no enemy healer; the players' pass strikes first; enemies
    can't be wounded; a player's `HealthMax` `Base: 1` gives a little
    more HP than a mob; both sides unplaced, so formation is inert here.

### Documentation maintenance and Phase 30e preparation (2026-09-30)

- Removed mounted combat from the 30f proposal at the owner's request.
- Removed completed implementation checklists, superseded initial combat
  proposals, the unused legacy workflow redirect, and stale bootstrap/history
  text. Retained current designs, unresolved 32b review material, known issues,
  and operating constraints. Prior files and detailed verification/review
  history are recoverable from commit `792455ea`.
- Removed the project's plugin enablement and mandatory skill workflow;
  retained designs now live in `docs/designs`, plans in `docs/plans`.
- Phase 30e remains in design, not implemented. The [next-task handoff](plans/2026-09-30-phase-30e-handoff.md)
  records context and the owner's requested 6.1 Sol model.
- **Review:** independent reviewer found a stale 11d deferred-work bullet;
  removed it because Phases 30a–30c2 shipped those mechanics. No other blocking
  findings. **Verification:** local documentation links resolve; staged diff
  whitespace checks pass; all eight changed Go files are comment-path edits
  only. No gameplay tests run for this documentation-only change.

### Latest gameplay verification (historical)

Phase 30d2's implementation, review fixes, and verification were recorded
before this cleanup; consult `git show 792455ea:docs/PROJECT_STATUS.md` for
those results. This documentation change does not rerun or supersede them.

## Known issues / deferred items

- **Broken chants and counters (30d1), for the owner:** the counter's
  numbers (50% of fumbled melee blows, 1d4, a stun one time in four, once
  a round per bearer) are a recommendation to tune. A bash leaves no
  wound; an enemy restarts its chant for ever while hit (no mana spent); a
  mob that runs out of mana wastes the turns it picks `cast` (GoMud's
  combat commands). No casting on the battle view's grid yet (32g2).
- **Chance to break a chant (30d1b), for the owner:** the numbers (40% +
  2 per percent of max health, held to 40–90%) are the lead's choice, to
  tune. A frail caster's floor is rarely 40% (a 30-health healer hit for
  3 is at 60%). A shield bash breaks no chant (owner, 2026-09-30, with
  30d2: a counter strike only). A weapon's own
  crit statuses still land on a crit the armor took (30a), so they make
  that round's other blows heavy.

- **Balance baseline (30g1), recorded for 30g6:** the harness's even
  5v5 keeps the game's asymmetries: no enemy healer, the players' pass
  strikes first, enemies take no wounds, a player's `HealthMax` starts
  at `Base: 1` (a mob's at 0, so a player gains a little more HP per
  level), and both sides fight unplaced (formation inert). The table
  runs only with `ASHVEIL_BALANCE=1`.

- **Wind-ups (30d2), for the owner:** the numbers (35% of the ogre's
  turns, x2 damage, 2 turns' cooldown) are the recommendation to tune.
  Company wind-ups, the proposal's sweep across the front line, and other
  abilities are deferred; a shooter can't wind up. Ordinary blows on a
  wind-up are silent (owner), so the summary counts only broken ones; a
  wind-up lost to a stun no blow dealt is credited to no one. Wind-ups
  aren't shown on the battle view's grid (32g2).

- **Company tactics (30c1), accepted:** an order given in the round its
  battle ends is dropped silently; a solo player has no web tactics row
  (the command works); enemy `wounded` weighs full health, not the wound
  limit; hunting the isolated (adjacency) and rotate-the-wounded are
  deferred. (The `casters` focus now has enemy casters: 30d1's hexer.)

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
  health aren't saved (a restart refills both); enemies aim by personality since 30c1; companions following a flee is untested. 32c's two owner items
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

- [Agent workflow](AGENT_IMPLEMENTATION_WORKFLOW.md) — implementation and review.
- [Handoff](ASHVEIL_GOMUD_AGENT_HANDOFF.md) — design direction and invariants.
- [Combat roadmap](designs/2026-09-26-combat-presentation-roadmap.md) — remaining phases and decisions.
- `docs/designs/` — active proposals and useful shipped design records.
- `docs/plans/` — execution guidance and work still requiring follow-up.
- Nested `AGENTS.md` files — package-specific constraints.
- Git history — retired plans/proposals and detailed completed work logs.
  Use `git show 792455ea:<old-path>` for the snapshot before this cleanup.
