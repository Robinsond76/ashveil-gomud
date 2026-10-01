# Ashveil Project Status

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-10-01
- **Latest completed phases:** 33c, company retreat; 33b, friendly effects;
  33a, command rules; and 30g3, personal load and agility, all 2026-10-01.
- **Upstream baseline:** `39e44013` (GoMud Telnet input-masking fix).

## Current position

The expedition/company loop, onboarding, combat presentation (29a–29f),
status effects, wounds, tactics, guardians, interrupts, wind-ups, morale
and mercy (30a–30e), active defense (30g2), and personal load (30g3) are shipped. The Phase 32
play-test improvements are implemented; 32b's status still carries an outstanding review note (see its retained plan).

**Current owner priority:** finish Phase 33a–33i in order, choosing the lead's
recommended defaults without further confirmation (2026-10-01). 33a–33c are
complete; **33d, allied companies, has started with its consent slice**. Phase 30g3 is also complete;
30g4 (progression) is the next combat-tempo slice, after the 33 series.

**Combat tempo queue:** Phase 30g, [combat tempo, personal load, and active
defense](designs/2026-09-30-phase-30g-tempo-defense-design.md), whose
decisions the owner settled on 2026-09-30. 30g1 (the balance harness and
baseline), 30g2, and 30g3 are done; each later slice is measured against 30g1:

1. **30g2, active defense and armor:** complete (work log below): one
   defense per strike: block with a shield (no dodge), else parry a
   melee strike or dodge; the shield's ×1.5 armor removed; the shield
   bash moves to a blocked melee strike (5–20% by Strength); armor's
   suffix reads `absorbed`; `help defense`.
2. **30g3, personal load:** complete (work log below): worn and carried
   weight against a Strength-based capacity (cargo and mounts never
   count); burden lowers dodge only; burden words in `status`, `look`,
   `scout`, and the web Overview; `help burden`.
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

**Future company gameplay:** the owner endorsed the 2026-10-01 gameplay review
and requested [Phase 33a–33i designs](designs/2026-10-01-company-gameplay-roadmap.md):
command consistency, friendly-effect scopes, retreat, allied companies,
automatic class abilities, specialists, equipment/loot, progression/recovery/
relocation, and group assessment/coordinated enemies. These are planning
records now authorized for implementation by the owner, with open defaults
delegated to the lead. 33a–33c are complete; 33d is in progress.

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
| 30g | Combat tempo, personal load, and active defense | In progress: [design](designs/2026-09-30-phase-30g-tempo-defense-design.md). 30g1 complete ([plan](plans/2026-09-30-phase-30g1-balance-harness.md)): the balance harness and baseline (no-focus 5v5: median 8 / 32 / 68 rounds at levels 1 / 5 / 10). 30g2 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g2-active-defense.md)): block, parry, or dodge, one per strike; the shield's ×1.5 removed; the bash on a blocked melee strike (5–20%); `absorbed`; `help defense`. 30g3 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g3-personal-load.md)): personal load and burden (dodge × (1 − 0.6 b)); `help burden`. 30g4 (progression) next |
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
| 33a | Company Command Rules and Legacy Action Routes | Complete: [design](designs/2026-10-01-phase-33a-company-command-rules-design.md), [plan](plans/2026-10-01-phase-33a-company-command-rules.md); safe ask aliases, queued ownership checks, battle/script/skill policy |
| 33b | Friendly Effects and Company Membership | Complete: [design](designs/2026-10-01-phase-33b-friendly-effect-scopes-design.md), [plan](plans/2026-10-01-phase-33b-friendly-effects.md); shared target snapshots, eligibility, ownership and callback revalidation |
| 33c | Company Retreat, Rout, and Separation | Complete: [design](designs/2026-10-01-phase-33c-company-retreat-design.md), [plan](plans/2026-10-01-phase-33c-company-retreat.md); ordered withdrawal, paid cover, legal escape and saved separation |
| 33d | Multiplayer Parties and Allied Companies | In progress: [design](designs/2026-10-01-phase-33d-allied-companies-design.md), [plan](plans/2026-10-01-phase-33d-allied-companies.md); initial membership/follow/support consent slice verified on feature branch; encounter rewards, loot and durable alliance recovery pending |
| 33e | Automatic Class Abilities and Combat Roles | Future design: [proposal](designs/2026-10-01-phase-33e-automatic-class-abilities-design.md); implementation not started |
| 33f | Company Specialists and Group Exploration | Future design: [proposal](designs/2026-10-01-phase-33f-company-specialists-design.md); implementation not started |
| 33g | Company Equipment, Loadouts, and Loot | Future design: [proposal](designs/2026-10-01-phase-33g-company-equipment-loot-design.md); implementation not started |
| 33h | Company Progression, Rewards, and Expedition Continuity | Future design: [proposal](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md); implementation not started |
| 33i | Company Encounter Assessment and Enemy Roles | Future design: [proposal](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md); implementation not started |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

### Phase 33d: initial alliance consent slice (2026-10-01)

- Pulled origin master `4b4db05e` and confirmed 33d was not implemented;
  isolated branch/worktree `phase-33d-allied-companies` retains this initial unit.
- Reused runtime parties and 33b's helpful-target resolver: accepted members
  opt into following and mutual allied/area support; joining grants neither.
  Company-only spells and companion command/formation/asset authority stay per
  owner. Autoattack consent rejects invitations; leaving/kicking/disbanding
  prune consent. Promotion resets following; logout reports the successor.
- Queued follows capture current leader, origin and a unique consent token,
  checked at actual execution after requeue. Off/on, leaving/rejoining,
  recreated parties, movement, promotion and active prompts invalidate them.
- Shipped indexed `help party`, aliases, company/combat/support links and a
  Departure hint. Tests exercise actual party commands, ordinary movement,
  production hook registration, cast completion and logout/recovery paths.
- Independent review accepted and fixed delayed-follow authority, production
  provider coverage, successor feedback and contradictory design status;
  re-review found solo logout notification, also fixed with regression coverage.
  No rejected findings or unresolved blockers for this initial unit.
- Verification passed: touched-package tests; `go test -race .
  ./internal/hooks -count=1`; `make generate`; `make validate`; final
  `go test -race ./...`; `git diff --check`. Initial race/test failures in the
  new fixture (redundant logger reset and capturing unrelated queued input)
  were corrected before the green final suite. No JS/Lua source changed.
- **33d is not complete and has not been merged/pushed.** Party state remains
  runtime-only. Encounter participation/XP, deterministic loot, durable alliance
  recovery and rank replacement remain the next tasks in the plan.

### 33a owner review follow-up (2026-10-01)

- **Owner review of 33a:** the owner kept two 33a choices. Players can't get, drop, give or put items during their own battle. Aid is refused during the aider's battle.
- **Help fix:** the 33a–33c sections of `help company` had been added after its "See also" line. That line is last again.
- **Script fix (`fix/33a-company-help`):** 33a treated a follower's own script replies (onAsk/onGive/item/room scripts) as orders from whoever's command triggered them.
  - Another player's follower therefore went silent and showed "You no longer command that member." The owner's follower could not whisper, emote or leave.
  - Script commands on a follower now count as its owner's (`MemberOrder.Scripted`). They are still refused, silently, if they would fight, or change gear during a battle, or if the charm or membership behind them has changed. They skip `ask`'s order list and the room checks.
  - Asking your own follower about a subject reaches its onAsk script again. An unanswered subject points to `help ask`.
  - `help ask` updated.
- **Independent review:** nothing blocking. All six findings accepted:
  - Another player's trigger could make a follower's script attack. Closed by attributing follower scripts to the owner.
  - Delayed replies were dropped after the follower moved. Room check removed for script commands.
  - Typed non-orders lost the pointer to `company`. Restored.
  - Missing tests for battle gear, emote shortcuts, hostile aliases, moving and typed attacks. Added in `modules/company/wiring_order_scripts_test.go`.
  - Inaccurate help wording. Fixed.
  - Status not yet recorded. Done here.
- **Checks:** the new tests fail on the old code and pass now. `make generate`, `make validate` and `go test -race ./...` passed. No JS or Lua files changed, so lint was not needed.

### Phase 33c: company retreat (2026-10-01)

- Ordered `retreat [exit]`: one preparation round, one escape attempt, slowest
  member's current speed/burden/lasting wounds against fastest active pursuer.
  An eligible guardian spends a guard and preparation attack for cover. Stable
  ownership and legal exits revalidate each round; blocked living members hold
  the company, while dead members retain resurrection and morale flights retain
  30e recovery. Only captured live members move; runtime orders never save.
- Emergency flee remains individual escape, with 30e's saved separation for
  blocked living companions. Failed separation save holds the leader; debt is
  paid once on rejoin. End battle only after successful movement; respect
  no-go, effective locks and room permissions. Departed casts/aims clear without
  ending another company's battle. No cloned gear, cargo or herd; no clock jump.
- Text, Company.Battle countdown and browser view; indexed retreat help,
  corrected legacy flee/attack/targeting/break help and tutorial.
- Independent full-diff review found no implementation blockers but six coverage
  gaps. Added real checks for waiting groups, another active company battle,
  late arrivals, departed spell targets, GMCP gather countdown and relocation
  rollback. Reviewer rechecked all six; no remaining blockers, no rejections.
  Stabilized the chant regression by making test blows miss before pruning.
- Verification: focused touched packages and new coverage repeated ten times;
  full browser dock harness with system Chromium/local HTTP passed. Final
  `make generate`, `make validate`, `go test -race ./...`, installed JSHint via
  `make js-lint`, `make lua-lint`, and `git diff --check` all passed.
- Next: 33d. Cloud session creation unavailable; saved handoff, continue here.

### Phase 33b: friendly effects (2026-10-01)

- Shared resolver for player, mob, automatic and script cast paths. Group help
  includes present attached company members; single help retains explicit
  patients. Runtime snapshots only prune, checking rooms, life, stable company
  identity and charm identity/expiry. Temporary followers stay outside group
  effects; allied expansion waits for 33d consent. Healing allows downed living
  players above -10, never revives fallen companions, and retains wound caps.
- Validated scope metadata and shipped spell mappings; costs and durations
  preserved. Indexed friendly-effects help, hub links, tutorial and render tests.
- Independent full-diff review accepted two P2 findings: HelpArea lacked actor
  arrays, and the void callback fix changed onCast's proceed default. Fixed both
  with actual JS/Lua cast regressions; other successful void handlers no longer
  report a missing handler. Reviewer rechecked against latest master (30g3),
  including actual round completion/pruned healing narration and stream; no
  remaining blockers. No findings rejected.
- Verification: focused spell/script/usercommand/company/tutorial tests;
  make generate, make validate, full go test -race ./..., installed JSHint via
  make js-lint, make lua-lint, and git diff --check. Full checks recorded after
  final review fixes. No save migration: cast snapshots and Aggro are runtime.
- Next: 33c. No cloud-session creation tool is available; continuation context
  is in the phase 33 session handoff and work continues in this session.

### Phase 33a: company command rules (2026-10-01)

- Safe `ask` management aliases for living present companions and temporary
  followers; all requested follower hostility is refused. Automatic battles,
  native NPC/lifecycle commands, and administrative input remain distinct.
- Queued orders carry requester, room, stable member key and runtime charm
  identity; execution rechecks battle, presence, ownership, dismissal, death,
  expiry and transfer-back. No persistent fields or migrations were added.
- The player dispatcher checks battle restrictions before scripts, including
  personal/global aliases; aid/tame cannot replace battle actions. Skill
  execution rechecks new battles and protected/owned tame targets.
- Indexed `help ask`, updated company/combat/equipment/consumable/skill pages,
  tutorial pointer and help rendering tests shipped with the implementation.
- Independent review: accepted and fixed both findings (multiword alias
  interception bypass and obsolete aggressive-taming help). Added requested
  aid/battle-owned target/temporary follower coverage. Reviewer rechecked the
  fixes and regressions: no remaining blockers; no findings rejected.
- Verification: focused dispatcher/queue/script/lifecycle tests; `make
  generate`, `make validate`, `go test -race ./...`, `make js-lint` using the
  installed JSHint binary, and `make lua-lint` all passed. `git diff --check`
  passed. Native follow regression caught during implementation was fixed
  by separating `CommandRequested` from autonomous `Command`.
- Next: 33b, company-only friendly-effect correctness first, then allied
  scopes once 33d defines consent. The current tooling has no cloud-session
  creation operation; continue here using the saved phase handoff.


### Company gameplay roadmap and future Phase 33 designs (2026-10-01)

- **What/why:** recorded all nine owner-endorsed review recommendations as
  [33a–33i future designs](designs/2026-10-01-company-gameplay-roadmap.md),
  with code evidence, dependencies, owner decisions, state/recovery rules,
  player help, and real integration acceptance requirements. The guides
  are Ogre Battle for composition/automatic roles and Mount & Blade II for
  readiness/specialists/company control. Updated the design index and combat
  roadmap; corrected its stale next-phase instruction to 30g2.
- **Baseline:** fetched origin master at `626df427`, incorporating shipped
  30e before planning retreat/return extensions. No gameplay changed and
  no future Phase 33 implementation is claimed.
- **Review:** independent default-agent reviewer checked the complete
  documentation diff and relevant code; no actionable findings, accepted
  or rejected findings, or unresolved review blockers.
- **Verification:** 74 local Markdown links and 29 explicit repository
  path references resolve; all nine designs contain scope, evidence,
  ownership/recovery, dependencies, decisions, verification, and help
  requirements. Staged whitespace checks pass. No gameplay tests run for
  this documentation-only change.

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

### Phase 30g3: personal load and agility (2026-10-01)

- **What:** the third slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-10-01-phase-30g3-personal-load.md)), on
  `claude/phase-30g3-load`. `Character.PersonalGrams` (worn and carried,
  moved from the peep panel) against an agility capacity of
  `AgilityBaseKg` 15 + `AgilityStrengthKg` 0.5 × Strength (new `Combat`
  keys, with `AgilityFreeLoad` 0.35, in `config.yaml` and the admin
  wizard) gives a burden 0–1; dodge becomes dodge × (1 − 0.6 b) in the
  strike loop and the weapon rankings' estimate. Parry and block are
  untouched; the parry-or-dodge choice compares parry with the burdened
  dodge. Cargo, packs' bonus, and mounts never count. Words (unburdened,
  lightly burdened, burdened, heavily burdened) in `status` (Vitals),
  `look` at any character, `scout`/`look` of a group, and the web
  Overview (`Char.Inventory.Backpack.Summary.burden`). Nothing saved.
  Help: new `help burden` (indexed under combat, aliases), `defense`,
  `perception`, `strength`, `cargo`, `encumbrance`, `scout`, `status`,
  `look`, and the combat hub updated; a Combat-lesson hint.
- **Why:** owner decisions 2 and 3.
- **Balance** (30g1 table, 50 fights a cell): no-focus median **8 / 35 /
  77** rounds at levels 1 / 5 / 10 (30g2: 9 / 38 / 71). Every harness
  fighter is unburdened (kits 0.4–4.6 kg, under the 5.25 kg+ free share),
  so 30g3 cannot move these fights: the differences are run-to-run noise.
- **Calibration, for the owner (kept as proposed; 30g6 tunes):** Strength
  stays small (0 at level 1, 2–3 at 5, 5–8 at 10, 10–16 at 20), so
  capacity is 15–23 kg and the free share 5.3–8 kg. A new character's
  starter kit (5.4–9 kg with food, water, and a satchel) starts lightly
  burdened (cleric, rogue, warrior, wizard barely) and the ranger's
  burdened (b ≈ 0.39, dodge −23%); real armor (a breastplate and an iron
  shield, 15 kg) leaves anyone heavily burdened. The design hoped "a light
  kit costs nothing". The company's travel `Load:` also uses the words
  Burdened/Unburdened; the help separates the two.
- **Review:** the independent default-agent reviewer found nothing
  blocking; it checked the math, every dodge caller, the character
  copies, game-loop safety, statistical bounds, hit-forcing helpers, help,
  the web client, and the admin pages. Each finding was verified.
  - **Fixed:** `cargo put`/`take` raised no `ItemOwnership`, so the web
    Overview's burden (and 32g's weight) went stale until another event
    (`TestCargoMovesRaiseItemOwnership`); an unvalidated test config with
    `AgilityFreeLoad` ≥ 1 gave NaN, and the defender silently never dodged
    (now no free share, and NaN burden is 0; regression cases in
    `TestBurdenForUnvalidatedFreeShare` and `TestBurdenedDodge`); help
    hard-coded "0.35 is about a third" beside the configured value, and
    the Strength page now reads the per-Strength key; the plan's alias
    list matched the shipped one.
  - **Rejected/deferred:** scout's Burdened line names duplicates by name
    only (the grid does the same; `look [group]` puts the word inline);
    `look` shows burden for shopkeepers and horses too (harmless, one
    rule for every character); no test looks at another player (same
    helper as a mob) or drives GMCP through a live event (the summary
    path and the event are tested apart); the armor rankings don't weigh
    burden (for 30g6).
  - **Found while reviewing:** `TestAClericCompanionHealsTheHurt`
    (`modules/company`) failed once in a full run and passed 40/40 alone
    on this branch and on master; every brawl member is unburdened, so it
    is an existing flake, not this phase's.
- **Verification** (2026-10-01, after the review fixes and merging
  `origin/master` with Phase 33a): `make generate` (no diff), `make
  validate`, `make js-lint`, and `go test -race ./...` pass (go test's own
  exit status, 92 packages ok, no races). While working, `-count=3` on
  `internal/combat`, `internal/characters`, the new brawl tests, and the
  mount/cargo and encumbrance tests; each new combat test was checked to
  fail with the burden change removed. The web Overview was not exercised
  in a browser (JSHint and the GMCP payload test only).

### Phase 30g2: active defense and armor (2026-10-01)

- **What:** the second slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-10-01-phase-30g2-active-defense.md)), on
  `claude/phase-30g2-defense`. A blow that would hit meets one active
  defense: a shield-bearer blocks any strike (no dodge fallback); with a
  weapon and no shield, a melee strike meets the higher of parry and
  dodge, rolled once; shots, claws, and bare hands meet a dodge; a stunned
  fighter gets none. Block: 15% + the shield's armor + the Strength share,
  held to 15–45% (wooden 20%, iron 25% at even Strength). Parry: Speed,
  5–30%, moved by the weapon (swords, reach weapons, and the staff's new
  item `parry` +5; daggers −5). The shield's ×1.5 is gone from
  `GetDefense` and the armor rankings. The bash follows a blocked melee
  strike, 5–20% by Strength (1d4, stun 25%, once a round, unchanged).
  Attack events carry `Defenses`; the battle summary has a `Defenses`
  line; the armor suffix reads `absorbed`; the 30g1 harness counts
  defenses. Player help: new `help defense`, `help armor` rewritten and
  indexed, nine pages updated, the Combat lesson's hints corrected.
- **Why:** owner decisions 4–6, 10, and 15–17.
- **A first attempt was not fit to merge.** It gave daggers +5 parry (the
  design says −5), left the ×1.5 in `armor_rank.go`, emitted no defense
  events, left help and the tutorial describing the old miss counter, and
  its status entry here called the phase complete while ten
  `modules/company` counter tests failed (a `go test ... | tail` exit code
  was read as the test result). The owner asked for a review of the work;
  it was rebuilt as the plan records.
- **Balance** (30g1 table, 50 fights a cell; the 30g1 baseline re-run on
  the same machine for comparison): the no-focus fight's median is now
  9 / 38 / 71 rounds at levels 1 / 5 / 10, against 9 / 32 / 67 before.
  Shield bashes fall from 4–14 a fight to under 0.3. Per fight at level
  10, each side makes about 3–6 blocks, 2–2.5 parries, and 2 dodges.
  The company's no-focus wins at level 1 fell from 58% to 48% (within the
  noise of 50 fights). Defenses lengthen fights slightly; 30g6 tunes
  against the 10–15 round target.
- **Review:** the independent default-agent reviewer found no blocking
  bug and confirmed the defense choice, numbers, strike loop, bash sites,
  stream, summary, and help. Each finding was verified.
  - **Fixed:** a parry range of 0 still let a sword parry 5% (the weapon's
    modifier came after the clamp), so hit-forcing tests could flake
    (`TestZeroParryRangeTurnsParryOff`); `TestShieldCounterOnlyOnBlock`
    and the stunned-bearer case now require the blow they test; the new
    keys added to `config.yaml` and the config wizard; admin armor pages,
    the items API page, the rankings help, and the empty world's armor
    page no longer describe the ×1.5; the Stunned buff and `help combat`
    name parry; stale comments; `defend` removed from the defense
    aliases (it is a strategy rule); a defended shot from another room
    now gets a line in the shooter's room.
  - **Rejected/deferred:** a reach weapon gets +5 one-handed or not (the
    design said two-handed polearms; none ship yet); a defended shot
    still reads "blow" (generic wording, kept).
- **Found while testing, recorded:** a bandit given a sling in the brawl
  world never attacks (the old bow-counter test passed without a blow);
  the rebuilt test uses the company ranger's shot. Not investigated
  further. `TestJoinsTheFight` (`internal/hooks`) fails under
  `-count=3` on the 30g1 base too: a test-isolation issue, not this
  phase's.
- **Verification** (2026-10-01, after the review fixes): `make generate` (no diff), `make validate`, `make js-lint`, and `go test -race ./...` pass (the go test exit status itself, 91 packages ok, no races). No Lua changed. While working: `go test -count=3` on `internal/combat` and the counter, status, wind-up, and pain tests of `modules/company`; the reviewer ran `internal/combat` 20 times. After merging `origin/master` (Phase 30e) into the branch: `make generate` (no diff), `make validate`, `make js-lint`, and `go test -race ./...` pass again (go test's own exit status).

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

- **Broken chants and counters (30d1, 30g2), for the owner:** the
  counter's numbers (5–20% of blocked melee blows by Strength since 30g2,
  1d4, a stun one time in four, once a round per bearer) are a
  recommendation to tune. A bash leaves no
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
