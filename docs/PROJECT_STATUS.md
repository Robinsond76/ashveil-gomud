**Phase 86 built, reviewed and merged (2026-10-08, PR #196, test-only): two flaky tests.** [Findings](plans/2026-10-08-phase-86-dreadwhisper-flake.md). `TestSparksOverloadsItsTargetsThroughARealCast` failed 5 of 240 race runs because five bandits broke nearly every recast chant (no game bug: a broken chant is simply lost); it now pins the chant-break dice, 0 of 240 after. `TestDreadWhisperMakesAFoeTakeAMoraleCheck` still could not be reproduced (about 2,200 runs); `witchBrawl` now pins the same dice, the one mechanism that matches the Phase 83 sighting. **Review:** no code change. The Phase 85 note that Aria "never casts Sparks again" after one interrupt was a misread: in three failing transcripts (3 of 80 race runs, unpinned) every one of the 31 attempts chanted again and was broken, one break per attempt, so the caster AI is sound; five bandits landing blows at the 40% floor break nearly every chant, as `interrupt.BreakChance` intends. Breaking each target on purpose fails its test: Sparks without the overloaded buff fails 3 of 3, Dread Whisper with every hex resisted fails 2 of 2. Pinned, Sparks passed 160 of 160 race runs. Noted: in 20 runs of every witch test the chant-break dice were never consulted (the bandits hold their blows), so the `witchBrawl` pin is harmless insurance, not a proven fix. Gates: generate, validate, `go test -race -timeout 30m ./...`, `make smoke`.

**Phase 85 built, reviewed and merged (2026-10-08, PR #195): the bestiary feeds the chronicle.** [Design, build notes and review](plans/2026-10-08-phase-85-bestiary-chronicle.md#review-2026-10-08). The kill that teaches a kind's habits (6th kill, 3rd for a boss) writes one `mastered` deed from `mobcommands.Suicide` ("The company learned the habits of the big rat at Old King's Road."); `chronicle.KindCaps` keeps at most 30 of them within the 300 deeds, so other deeds keep at least 270 (the tally keeps counting); `chronicle lore`/`mastered`/`beasts` filters; the web Chronicle tab shows a Beast lore button; one townsfolk test line; help and the Combat tutorial hint updated. No backfill, no effects. **Review:** one fix: `help chronicle` said the 30 lore lines were kept "apart from the 300" (read as 330 lines); it now says at most 30 of the 300 are lore. Two tests added: a full log of 300 deeds plus 100 lore lines keeps 270 others and 30 lore; lore and its cap survive a reload and the cap still drops only lore. Breaking the cap fails all three cap tests. Live: an admin's 6th rat kill wrote the line in the web Chronicle tab (desktop and 360px, screens `85-review-live-*.png`), and the 7th added none. Accepted: the web filter button appears only once a lore deed exists (every kind works this way; an empty button would show nothing). Accepted, not changed: a capitalised common name (stock "Angel", "Demon") reads without "the"; the world is temporary. Gates: generate, validate, `go test -race -timeout 30m ./...` (142 packages), the packages phase 84 touched again after merging it, `make smoke`. Seen, not this PR's: `TestSparksOverloadsItsTargetsThroughARealCast` (modules/company) fails about 1 run in 20 under `-race` on master before and after phase 84 (3 of 60 each); in the failing run Aria's chant is interrupted once and she never casts Sparks again in the later attempts, so it may be a real cast-state bug. Queued with the DreadWhisper flake.

**Phase 84 built, reviewed and merged (2026-10-08, PR #193): the Sorcerer's Lance and the level 20-22 rest rhythm.** [Decisions and measurements](plans/2026-10-07-phase-84-sorcerer-balance.md). (1) The advanced Sorcerer looses its Arcane Lance against three foes or fewer and showers four or more with sparks (class effect `lancefoes` 3, in `strategy.Decide` and the `strongest` order); 200-fight mirror cells read Sorcerer 48 / 48 / 47 against the base wizard 54 / 43 / 50 at L25 / L40 / L50 (L50 was 34 against 46). The High Sorcerer's High Lance rank lifts the limit: applying it there cost it 65 / 70 -> 49 / 48, so it keeps the Lance (63 / 66 measured). Help (`wizard-routes`, `high-sorcerer`) and rank texts updated. (2) Ordinary foes in bands from level 20 up spawn with 30% HP (was 40%; `encounters.HighBandHPPercent`); a five-member martial company at 20-22 now rests after a median 16 fights (was 12; target 15-20), four 12, three 9, five magic 10, solo wins 76%; 3-5 and 10-12 untouched. The five-member magic company at 3-5 stays at 5 (a level-5 wizard's mana). **Review:** no game code change. The limit is keyed on ranks (High Lance sets `lancefoes` 0), both deciders count standing foes, sparks use the wizard's own lines, `why` and help read true. Spot-checks: Sorcerer L50 46% against the base wizard's 46% (60 fights); higher bands, five members, 40 fights, 40% -> 30%: 25-28 martial 18 -> 21, 29-33 martial 15 -> 18, magic 11 and 11. Kept the 30% share for every band from 20 (errs easy as the owner asked, 29-33 sat on its floor at 40%, the stock world is being replaced); `ASHVEIL_BALANCE_BAND` now takes any band. Gates: `go test -race -timeout 30m ./...`, `make smoke`.

**Phase 85 designed (2026-10-08, docs only): the bestiary feeds the chronicle.** [Design and decisions](plans/2026-10-08-phase-85-bestiary-chronicle.md). Settles phase 66's open seam the other way round: the bestiary keeps reading the kill tally (a deed per kill would crowd the 300-line log, as the phase 76 review found), and instead the kill that teaches a kind's habits (6th kill, 3rd for a boss kind) writes one `mastered` deed ("The company learned the habits of the big rat at ..."), once per kind per company. Kept mastery deeds are capped at 30 of the 300 lines (oldest dropped, lifetime tally kept); `chronicle lore` filters them, the web filter gains Beast lore, towns can speak of them (one test line). No backfill, no elite or first-sighting deeds, no `why`, opinion, blessing or price effect. Next: a Sonnet build thread from the doc's acceptance list.

**Phase 83 built, reviewed and merged (2026-10-08, PR #192; one game-code fix plus tests): test hardening.** [Findings and decisions](plans/2026-10-08-phase-83-test-hardening.md). (1) `TestBattleEventsThroughTheRealRound` was a real game bug, not test noise: when the last target of every fighter on both sides fell in the same round, each aim was still on a body at the next upkeep, `engagedWith` read that as "nobody is fighting" and skipped the re-aim, and `closeIdleBattles` then broke the battle off with foes standing (a second `fight-end` and summary; the next round opened a new fight, so the battle screen's round counter went back to 1). `engagedWith` now also counts a plain attack on a mob that has fallen or been removed, so the upkeep turns the company onto the foes that stand. It reproduced alone (the earlier "1000 of 1000" was too few runs): 5 failures in 2,886 solo runs before, 0 in 2,961 after; new deterministic regression test `TestABattleGoesOnWhenEveryAimFellInOneRound` (every fighter aimed at a fallen mob, one real round; fails before, passes after). (2) A real cross-test leak found on the way: `TestNarrationPreservesCombatOutcome` called `rand.Seed(29)`, which replaces the process-wide dice for good, so every later test rolled the rest of one fixed stream; it now seeds through `seedDice`, which reseeds from the clock at the end (`TestSeededDiceDoNotOutliveTheirTest`); a dead `rand.Seed(30)` in the battlefield tests (a no-op under Go 1.24) is removed. (3) Every test in `modules/company` that set `HealthMax.Value` by hand now uses `hardTo`, or the new `hardMaxTo` where only the maximum is raised and the current health is kept; numbers and assertions unchanged (other packages' unit-level sites left alone). Seen once and not reproduced: `TestDreadWhisperMakesAFoeTakeAMoraleCheck` (600 of 600 alone, 1,900 of 1,900 shuffled). Gates run once on the final code: `make generate`, `make validate`, `make js-lint`, `make js-test` and `go test -race -timeout 30m ./...` (142 packages ok). No help or tutorial change (no player-facing command or number changed). **Review:** [no code change](plans/2026-10-08-phase-83-test-hardening.md#review-2026-10-08). The `engagedWith` change only gates the upkeep's re-aim for the group already in battle; a won fight still ends by `battleOutcome`, an idle one by `closeIdleBattles`, and retreat, `break` and a focus change are untouched. Breaking the fix fails the new battle test, and dropping `seedDice`'s cleanup fails the dice test. No skips, retries or removed assertions. Live: both road brigands zapped to 1 health fell in round 1 and the fight ended once, with no second "Round 1". Accepted, not changed: an aim on any corpse counts, not only one of the battle's group (harmless behind `inBattleWith`). Gates: generate, validate, `go test -race -timeout 30m ./...`, `make smoke`.
**Phase 81 built, reviewed and merged (2026-10-08, PR #190): class tuning, re-measured on the tempo combat.** [Plan, measurements and review](plans/2026-10-07-phase-81-class-tuning.md#review-2026-10-08). Rank values only: Druid (Barkskin +25 armor, Rejuvenation 170%/220%, Thornhide 4) 34 -> 44 against the Priest's 44-46, Elder Druid 30 -> 54, High Sorcerer 44/62 -> 60/76 at L40/L50 (Lance +60%/+75%/+90%, steadier chant), Arcanist 15%/25% spell damage, Stone Golem slows 10% and hits 3/5 harder (60/86/68 -> 66/94/76), Nightblade (Death Mark 60%, Envenom a third, +25% crit) 30 -> 38; Bearward, Warlock, Theurgist and Witch unchanged. New opt-in `TestPhase81ClassTuning` (`ASHVEIL_BALANCE_LINE=1` places the line). **Review:** no code change. Help pages and rank texts match the values. The build's reason for leaving the plain Sorcerer under the base wizard at L40-50 was wrong (the Lance chants the same two rounds as Magic Missile); spot-checks show the gap only at L50 (Sorcerer 22/34 vs base 40), and a trial that keeps the Lance for 1-2 foes and sparks larger groups closed it there (50/50) but read lower at L25, so the Lance-vs-Sparks choice is recorded as the Sorcerer follow-up instead of shipped. At-level rhythm re-measured on the tempo combat (30 fights a cell, median fights before rest, bands 3-5/10-12/20-22): 5 martial 18/16/12, 4 martial 13/14/12, 3 martial 10/9/8, 5 magic 5/7/10, solo wins 83/63/60%, every company of 3+ won every fight: the same as the pre-82 tuning, so no at-level foe change. Gates: `make generate`, `make validate`, `go test -race -timeout 30m ./...`, `make smoke`.
**Phase 79 built, reviewed and merged (2026-10-08, PR #191): a wider polish pass over the phase 60-77 features, their help pages and the web client.** [Findings, fixes and decisions](plans/2026-10-07-phase-79-polish.md). Found by playing scripted telnet sessions and live web screenshots at 1280 px and 390 px, then re-checked on the docked battle screen (82a) and turn-order strip (82d) after the combat overhaul merged. Fixed: the web client's login ate the creation window's early `!!GMCP(Char.Creation)` request as the username ("Invalid login"; the text-prefix handler now sits before the login prompts on websockets, as on telnet, and a smoke step logs in over `/ws` the way the browser does); `why` named rounds by the server's counter ("Round 1314418"; rolls now carry the fight's own round number, matching the dim "Round N" line, `TestRollLogNumbersRoundsWithinTheFight`); a fallen companion left a corpse that "crumbles to dust" mid-fight (a company member leaves no corpse on any of the three death paths; asserted in `TestCompanionDeathAndResurrectionThroughPluginsLoad`); the web Room panel tagged companions "charmed" (now `companion`, which the quick menu treats as not a foe, `TestRoomPanelTagsCompanionsNotCharmed`); a blessing perk read "will recruits cost 5% less"; mobs stood up "on their feet" and closed "their guard"; tab completion offered help-only topics as commands; `company` alone printed one 600-character line; the web Bonds tab said the same thing three times for two companions; and eleven help pages whose numbers or wording no longer matched the code (`rites`, `bonds`, `battlelog`, `relics`, `lifestory`, `bestiary`, `errands`, `bounties`, `chronicle`, and the `company` and `combat` hubs). No balance or economy changes. Checked and left: admin `teleport` during a paused journey, wide stock tables wrapping in the middle column, help-only topics staying topics. Gates run once on the final code: `make generate`, `make validate`, `make js-lint`, `make js-test`, the live smoke (`TestLiveSmoke`, with the new web login step) and `go test -race -timeout 30m ./...` all pass. **Review (Fable review thread, 2026-10-08):** two small fixes: the walking recovery line still said "finds their feet" after the pronoun pass (now "is up again, though still exhausted", as the knocked-down line), and the Bonds tab's comment said the per-member lines start at three pairs when the code starts them at two. Verified against the code: every help claim touched (`bonds.Affinity` and `BattleGain`, the bounty and errand pay, the chronicle kind `groups`, the relic opinions, the bestiary's unknown-name line, the rites prefix match), admin commands are still completed (they are registered), the `why` round and the dim "Round N" line both count from the battle's `StartRound`, and the websocket handler order now matches telnet. Checked on the real `webclient-pure` page against a live server in Chromium at 1280 px and 360 px: the login succeeds with the creation window's early GMCP request sent first (the fixed bug), `company` prints its grouped usage, the Room panel tags a companion `companion` and the quick menu's Attack list leaves it out. Firefox was not run; nothing here is browser-specific. Gates: generate, validate, js-lint, js-test, `go test -race -timeout 30m ./...`, `make smoke` (its first run failed at the restart step with no message and an empty server log; the verbose re-run passed; not reproduced). Merged master (80) before merging; only this file conflicted.
**Phase 80 built, reviewed and merged (2026-10-08, PR #185; tests only): tests that failed at random.** [Findings](plans/2026-10-07-phase-80-flaky-tests.md). Four causes, all in tests: (1) `toughen` and `hardenBandits` set only `HealthMax.Value`, which a stat recalculation (a knockdown ending) rebuilds from `Training + Mods`, so a "hard to kill" foe fell back to 48 health and died before the Warlord's Relentless test saw it stand (reproduced 5 in 600 under load; new `hardTo` helper plus `TestHardenedFightersKeepTheirHealthThroughARecalculation`; 240 of 240 clean after); (2) `alwaysLand` zeroes only base crit chance, so a Deadeye's crit (Eagle's eye, Hair trigger skips the winding) hid the winding its test counts (`noCrits` helper); (3) the Packlord level 44 foe sat 60% and could be worn under half before the bite (now 70%, no crits); (4) `make smoke`'s help step read topics in random map order and stopped at the example prompt in `keyring`/`lock`/`unlock`/`picklock` pages, so a later topic read empty (sorted, plus a drain). Not reproduced: PR #107's unnamed failure (CI log unreadable) and the unnamed 72a battle test. No game code changed. Gates: generate, validate, js-lint, js-test, full race suite and `make smoke` pass. **Review (Opus review thread):** no assertion weakened (mutation-checked the Packlord 70% case and the `hardTo` test); `noCrits` comment corrected, as it pins every d100 roll, not only crits (harmless beside `alwaysLand`). New flake found, not this PR's: `TestBattleEventsThroughTheRealRound` once saw its fight broken off and reopened mid-test (1000/1000 alone, about 1 in 100 behind other tests); likely also a candidate for #189's unnamed CI failure; recorded as a follow-up in the findings doc with the `HealthMax.Value` to `hardTo` migration. Gates: validate, race suite, `make smoke`, CI green.

**Phase 82d built, reviewed and merged (2026-10-08, PR #189, on top of 82c and 82a): turn order on screen and tempo shown. The combat overhaul (82a-82d) is complete.** [Decisions](plans/2026-10-07-speed-turn-combat.md#82d-as-built-2026-10-08-full-autonomy-amendments-and-checks). The battle screen shows the round's turn order as a strip of name tags under the picture, the one acting now bright and those done dimmed, following each blow as it plays; the Combat tab lists the order in words; each member's tempo shows on the Company panel, the Character tab and when hovering a figure (`Company` members carry `tempo`); and every player's text opens each round of a fight with a dim "Round N" line. Foes' tempos are never shown. Help: `help battlescreen` and `help tempo` updated, tutorial hint. Checked in the browser (battle, dock, phone and pane checks) and through the real round. Review (same thread as 82c and 82a; fixed in the review): reading a member's tempo for the panels went through `applyStance` on a copy and announced "You take the Shield wall stance" on every Company snapshot between battles (now `chosenStance` applies the stance to the number silently; regression test `TestReadingTempoNeverAnnouncesTheStance`); the strip's name tags read "the" and "a" for foes (now a member's first name and a foe's last word with its number, "wolf 2"); the battlescreen help's phone line described the pre-82a layout; and the battle clock waited the full 3 s minimum through a round in which nobody acted (after the opening turn every fighter slower than tempo 1 skips the next round, so round 2 of most fights was silent): a round with no turns and no held lines is now followed on the next turn (`roundHadTurns`, `TestAnEmptyRoundIsNotWaitedOut`). Checked and kept: the dim "Round N" line lands before each round's lines in a live telnet fight, one action per second with follow-ups a quarter second behind; the strip, Combat tab order and tempo numbers in Firefox at desktop and phone widths; foes' tempos never sent. Gates after the fixes: `make validate`, `make js-lint`, `make js-test`, the browser checks, `go test -race ./...` and `make smoke`.

**Phase 82a built, reviewed and merged (2026-10-07, PR #188; web only): the battle screen docks above the terminal.** [Decisions](plans/2026-10-07-speed-turn-combat.md#82a-as-built-2026-10-07-full-autonomy-amendments-and-checks). The battle screen opens at the top of the middle column with the terminal under it, as Robinson asked, instead of a floating overlay; the picture draws at the largest whole-number scale that leaves the terminal at least half the column, and the terminal font steps down a size while the pane is open ("Smaller text" checkbox on the screen's head, remembered; default on). On a phone the pane takes at most 55% of the Game view and scrolls, Retreat is also in the head, and a battle opening brings the Game view up. `help battlescreen` and `help webclient` updated, Combat tab tutorial hint corrected. Checked in Chromium and Firefox with a new `battle-pane-check.mjs` plus the battle, dock and phone checks; screenshots in `/mnt/project-files/screens/82a-*.png`. Built against master because every other task is paused (79 will rebase on it). **Review (Fable review thread, 2026-10-08):** no code change needed. Checked on the real `webclient-pure` page against a live server in Firefox (not only the harness): the pane opens at the top of the middle column with the terminal's own text under it, the font steps 16→13 with both docks (21 rows) and back on minimise, "Smaller text" off keeps 16, nothing scrolls sideways; at 1280 px wide with both docks the picture is 1× (320 px), 2× once the column is wider, which the whole-number rule guarantees. Merged master (82c) first; the only conflict was this file. Gates: battle-pane-check (Chromium and Firefox), battle, dock-windows and mobile checks, js-lint, js-test, help and tutorial tests, `go test -race -timeout 30m ./...`, `make smoke`.

**Phase 82c built, reviewed and merged (2026-10-07, PR #186): the battle clock.** [Amendments and measurement](plans/2026-10-07-speed-turn-combat.md#82c-as-built-2026-10-07-full-autonomy-amendments-and-measurement). While a player fights, combat rounds leave the 8 s cadence: `BattleClock` (a `NewTurn` listener) resolves the next round once the last one's lines have all gone out plus a short tail, never sooner than `MinRoundMs` (3 s) after it began; mob-against-mob fights keep the cadence while no player fights. A round still resolves at once under the game lock (sims and direct `DoCombat` tests untouched) but plays out one action per beat: each line carries its turn's slot (`events.Slot`), a turn's first line waits a beat (1 s at normal, 0.6 fast, 1.5 slow), its follow-ups a quarter of one, with the extra before a pain or death line, and nothing is squeezed into a window. Amended from the design with reasons recorded: one clock for the world rather than one per battle cluster; each player reads at their own pace and the round waits for the slowest; rounds numbered by a counter (the battle screen's round now counts 1, 2, 3). Buff audit: every in-fight status is already a combat-round buff; the stock Poisoned buff stays on game rounds (reason in the plan). `GetDurations` reports battle rounds; `conditions` says "N battle rounds left". Help: `help combatpace` rewritten, `help combat` and `help statuses` updated, tutorial hint. Measured: 7.0 s a round in the five-against-five brawl, so an at-level fight is about 25-30 s at normal. Tests: pacer beats, the real-round pace and battle-event tests rewritten for the clock (one beat a turn, the next round waits for playback and the minimum, a player's death in order, walking away flushes), pace off keeps the minimum round and a fresh clock resolves at once, the cadence leaves game rounds alone. Gates: `make generate`, `make validate`, `make js-lint`, `make js-test`, and the full race suite passed except `TestHealersDefaultSaysWhyTheLeaderTurns` (modules/company), which failed once under full-suite load (the leader's blow felled the marked healer, so the leader turned) and passed 5/5 alone on this branch; it calls `DoCombat` directly and the clock does not touch it, so it is a random-outcome flake, untouched here. **Review (Fable review thread, 2026-10-08):** fixed: a fight's last round, still playing out on its beats after the fight was over, was flushed in one burst when the fixed cadence came due (its `startPacedRound` flushes every held line; seen live over telnet: the last four blows, the closing line and the whole battle summary in one instant), so `CombatOnCadence` now waits while the pacer holds lines (`Pacer.Holding`), with a regression test; `conditions` says "combat rounds left" as help and GMCP do. Checked and kept: the counter-numbered rounds are consistent everywhere `combatRound` is compared, since a battle's `StartRound` comes from the same counter; retreat (`RoundsWaiting` counted by `retreatPass` at the top of each round) and company focus (applied at the upkeep) are untouched; `playerFightLive` runs on every 50 ms turn, a cheap scan when no player fights; a line requeued past the turn the clock computes its due time on joins the round late, which the pacer already tolerates; the web timeline's 8 s backlog collapse is harmless with beats. Live check (a new Warrior, the tutorial's straw squad, pace normal, over telnet): a turn a second, follow-ups a quarter second, a 1.5 s pause before a "beaten" line, rounds 3 s or more apart, and after the fix the fight's end and summary come a line at a time. Gates: generate, validate, js-lint, js-test, `go test -race -timeout 30m ./...`, `make smoke`.

**Phase 82b built, reviewed and merged (2026-10-07, PR #184 via the review PR): speed-ordered turns.** [Design, decisions and measurement](plans/2026-10-07-speed-turn-combat.md#82b-measurement-2026-10-07-before-phase-81-merged). A combat round no longer runs every player, then every mob: after the tempo fill, `buildTurnOrder` places each fighter's turns at turn number ÷ tempo (an opening bonus such as Iaijutsu or Scouted ground moves the first turn of the round the meter opened), ties go to Speed, then Perception, then a roll, and `runTurnOrder` plays the slots in that order across both sides, so a fast foe can strike before a slow companion and a fighter felled before its slot loses it. Turn counts, damage and class numbers are unchanged; retreats still resolve at the top of the round; held shots loose after the last first turn. Ability strikes, shadowsteps and battlefield powers now end per fighter after its first turn instead of in a pass. `Company.Battle.order` and the `slot` on battle events carry the order for 82d. The refactor into `actPlayer`/`actMob` is its own commit. Help: `help tempo` ("The turn order"), `help speed`, `help combat`, samurai, packlord and pathfinder pages, a tutorial hint. Tests: turn-order units, integration through the real round in `modules/company` (fast foe first, a kill denies the victim's blow, the order listed for the battle data), GMCP payload; five wiring tests that assumed leader-first order pin Speed with `actsFirst`; the phase 29d narration snapshot was recaptured (the order iterates sorted ids so the RNG draws are stable). Balance: `TestBalanceAtLevel` at 30 fights matches #179 within noise (five members 14-20, four 9-15, three 11-12, solo 67-80%, magic 5-9); the at-level knob stays at 40%. Measured before phase 81 merged. **Review (Opus review thread):** fixed: a mob's wasted wind-up line was closed only before the next mob's turn, so an interleaved player turn could fall between the blow and its line (`finishLanding` now runs before every slot); added the integration test the design asked for and `help tempo` claims, a faster foe's blow breaking a chant before its caster's release, with the reverse order as a control. Checked and kept: `tempoTurns` is filled by `beginTempoRound` before the order is built, so slot counts are this round's; per-fighter ending of ability strikes, shadowsteps and battlefield powers matches the old between-pass clear, since each is read only during its own fighter's turn (and `abilityDown` is written only before the turns); hoisting retreat is faithful, as `handleRetreat` sat ahead of the tempo gate; `roundOrder` is read by GMCP on the game loop (`companyview.OnRefresh`), as `hooks.PaceOf` already is; a fighter spawned mid-round gets no slot, and could not act before either. Accepted: three members at 11-12 fights before a rest is over Robinson's 5-7, unchanged from #179, and closing it needs party scaling, which the difficulty rule rules out. For 82d: a fighter whose meter gives it no turn still has a slot in `Company.Battle.order` (it shows losing the turn), so the strip should label it. UI: no client change needed for 82b; the timeline already plays events in arrival order, and the visible pace and order strip are 82c and 82d. Gates: generate, validate, js-lint, js-test, `go test -race -timeout 30m ./...` (142 ok), `make smoke`.

**Phase 82 designed (2026-10-07, docs only): speed turns and a docked battle screen.** [Design and plan](plans/2026-10-07-speed-turn-combat.md). Robinson found combat too fast to follow and asked for Ogre Battle turns by speed, the battle screen at the top of the middle column with the terminal below, and smaller battle text. Decided under full autonomy: speed is the existing tempo (no new stat; shown as "Tempo 1.2"); the action meter still sets how many turns; each turn's place in the round is turn number ÷ tempo, opening bonuses move round 1 earlier, ties by Speed then Perception then a seeded shuffle; each round still resolves at once under the game lock but plays back one action per beat (1.0 s at normal) and the next round waits for playback, per battle cluster, instead of squeezing every round into 6 s; in-battle effects count battle rounds; retreat and focus apply at the next round's upkeep. The battle pane docks above the terminal and the terminal font steps down a size while it is open (xterm draws one size). Build phases: 82a web layout and font (after 79 merges), 82b speed-ordered turns and balance re-measure (after 81 merges), 82c battle clock and beats, 82d turn-order strip and tempo display.

**Phase 78 built, reviewed and merged (2026-10-07, PR #180 via the review PR): polish for phases 60-72.** [Decisions](plans/2026-10-07-phase-78-polish.md). Seven follow-ups confirmed against master and fixed, two found not to be bugs. A story scene waiting through a restart now reopens its web modal (the modal message went out at login before GMCP was on; the client asks for it once connected, `event` still works). The Combat tab's Orders menu leaves out orders already set (`order_cmds`) and its Stance menu offers only stances the member's gear can use (`stances_fit`; a line replaces the button when none fit). The round heading no longer says "critical" when armor took the crit whole (the wire flag follows `CritLanded`). A background town line greets only the first member who has the background. `camp duties #id` and `Mira (#1)` labels tell same-named companions apart (and a bare name that fits two is refused). The Character panel hides the `nameless-N` placeholder. "Fall back a row" and forcing a class ability are not claimed by `help orders` and stay out. No balance change. Gates: generate, validate, js-lint, js-test, dock browser check, and the full race suite passed except `TestDeadeyeBoltThatFellsItsFoeNeedsNoWinding` (modules/company), which failed once under full-suite load and passed 8/8 alone on this branch and on master (a random-outcome flake, unrelated). **Review (Opus review thread):** no defects found; checked and kept as built: the scene request redraws the same page if both pushes arrive and sends nothing once a page is answered or left (`waiting` drops it), so it can't reopen a finished scene; the Stance menu offers only fitting stances while `stance` still takes any (with its "does nothing until" note), so a stance can be set before a gear swap; `camp duties #2` picks members as `strategy`/`orders` do, and refusing an ambiguous bare name is stricter than their first-match, on purpose. Added a 360 px check of the stance line and filtered menu to `dock-windows-check.mjs`. No balance change.

**Phase 77 built, reviewed and merged (2026-10-07, PR #177 plus review fixes): Hardcore (Iron) and account blessings.** [Decisions](plans/2026-10-07-phase-77-hardcore-blessings.md). Owner steer: no permanent death for now, so Hardcore is an option at character creation (the last question of `start`, confirmed, final) that makes a defeat cost two levels instead of one and never ends in a rescue, capture or robbery; foes are no stronger. Iron shows as a badge on `online`, `status`, `Char.Info` and the web Character window. Account blessings (`blessings.yaml`, nine, two Iron-only) are small perks unlocked by lifetime chronicle counts (a boss, relics, companions, mercy, scenes, a promotion): a starting camp supply or a recruit discount (5% each, 10% in total). They are saved per account in the new `modules/blessings` (kept through `delete character`, dropped with the account), checked on every deed and at login, and given once to each later character as creation ends; `blessings` and the Character window's Overview show carried, waiting and still-to-earn. `help hardcore`, `help blessings` and updated death, defeat, combat, delete, lifestory, company and webclient pages; a Departure hint. Follow-ups: permanent death, Iron companions, test-area blessings. Awaiting the independent review. Review: no permadeath and no combat effect for Iron confirmed; blessings can't be re-earned per account and the chronicle resets on delete; two UI fixes (Blessings moved below Worth on the Overview, the panel refreshes when a new character is given them); delete-and-recreate item farming accepted (about 100 gold of unsellable camp supplies per new character).

**Phase 76 merged (2026-10-07, PR #176 plus review fixes): bounty boards.** [Decisions](plans/2026-10-07-phase-76-bounty-boards.md). A room tagged `bounty-board` (Alderbrook green and the Frostfire Inn, for testing) lists up to five bounties from nearby zones' lairs and named groups, rotating every six real hours (derived from the board and the window, so restarts keep it). `bounty`, `bounty take|claim|drop [n]`: three held at once, 24 real hours each, proved by the chronicle (boss and group deeds in the target's zone after the take), paid in gold by the zone's band (20 per level of the band's middle for a lair, 6 per level for each of three groups), and written as a `bounty` deed. A bounty never scales a foe. The chronicle gained the `group` kind (written by the encounters module when an ordinary table group is wiped out; not for fled groups, boss groups or story-event groups), `bounty`, `Entry.Zone` (boss deeds now carry the zone) and `Filter.Zone`/`AfterSeq`; compositions take an optional `name`. Web: `Company.Bounties` and a Bounties tab (nine sub-tabs; the bar already scrolls). Help: `help bounties`, hubs, tutorial hint. Closes the phase 63 follow-up that non-boss named group kills were not recorded. Gates: `make generate`, `make validate`, `go test -race ./...`, `make js-lint`, `make js-test` and the dock browser check (see the PR). Review: accepted, a group deed was written for every won random fight, which would push boss kills, rites and choices out of the chronicle's 300 deeds and the towns' newest 80; now a group deed is written only while the leader holds a live bounty on that group in that zone (`bounty.Hunting`; tests in encounters and bounties). Fixed a merge conflict in the Frostfire Inn room tags. Checked and kept: proof cannot be reused (deeds must be numbered after the take, a claimed posting is settled, one bounty per mark), pay stays within band (at most five postings a board every six hours), the chronicle's readers ignore the new kinds, and the Bounties tab follows the other tabs' pattern.

**Phase 75 merged (2026-10-07, reviewed): inn room tiers.** [Decisions](plans/2026-10-07-phase-75-inn-tiers.md). An inn lets the common room (unchanged), private rooms (15 gold a member, Well Rested 1 hour) when tagged `inn-private`, and a suite (40 gold, 2 hours) when tagged `inn-suite`: `inn` lists them, `inn rest [private|suite]` buys one. A dearer room lengthens Well Rested and nothing else (same buff, vitals and wounds), so the at-level fight tuning is untouched; a short buff never shortens a longer one still running. `InnStay.Tier`, `Company.Camp.inn_rooms`, Camp tab room buttons, config keys in the camping overlay, `help inn` rewritten with aliases, a tutorial hint; Waymark Inn and Frostfire Inn have all three rooms in the default world, three more inns private rooms. Review: a cheaper room after a dearer one now says the longer Well Rested still runs; added a restart test and a Camp tab browser check (phone fit).

**Phase 74 built, reviewed and merged (2026-10-07, PR #171 plus review fixes): rites for the dead.** [Decisions](plans/2026-10-07-phase-74-rites.md). A companion lost for good, or one who leaves after long service (2,700 rounds, the Trusted tier), leaves a rite on the leader's record in the same save as the loss. The next camp rest or inn stay announces it (`company.OfferRites`, called from the real starts); `rites` lists, `rite hold|skip [member|#N|all]` answers at a camp or inn. Holding writes a chronicle line (`chronicle rites`), steadies the companions who trusted the one gone (+3, the rest of the band +1, up to 80) and draws the close ones together (new bond source `rite`, +2); letting it pass costs loyalty (-5, the rest -2, never below 30) and is a chronicle line too. A rite offered and still unanswered passes at the next camp or inn. No gold, supplies, experience or power; nothing in a fight; dismissal is not mourned. 73 (creeds) is not needed: the rite names no god and leaves a seam for creed words. Web: `Company.Rites` and a Rites tab in the Company window; the sub-tab bar now scrolls on a phone (eight tabs outgrew 360 px). Help: `help rites`, hubs, tutorial hint. **Folded-in check:** a companion brought down in a won fight does not take a lasting wound, because there is no knocked-out state: zero health is death, which clears wounds and costs a level to undo; recorded, not changed. **Bug found and fixed on the way:** phase 65's bonds were written to the company file but the decoder never read them back, so a restart dropped every bond (`TestRitesAndBondsSurviveARestart`).

**Phase 74 review (Opus review thread; `make smoke` passed).** Accepted and fixed: rites could not be held after an inn's 60-second stay ended (and paying again let them pass), so any inn room now counts; a rite passing unanswered no longer reads as the leader's choice; inn-safe wording; a guard test that every saved company field is read back on load; a mouse wheel scrolls the Company sub-tab bar. Rejected: a confirm on Let pass; tying camp rites to the camp's room. Details in the [decisions doc](plans/2026-10-07-phase-74-rites.md#review-2026-10-07-opus-review-thread).
**At-level fight tuning built, reviewed and merged (2026-10-07, PR #175 plus review fixes): a company in a zone for its level wins fast and cheaply.** [Decisions and measurements](plans/2026-10-07-at-level-fight-tuning.md). Ordinary random-group foes spawn with 40% of their level's HP and, at the band, aim at random among the members they can reach, fading to full HP and normal aim at five levels under the band (five under: 46% wins). Bosses, escorts and story groups are untouched. Opt-in `TestBalanceAtLevel` (placed company, warrior leader) measures fights before a rest: five martial members 12-17 (target 15-20), four 11-18, three 8-13 (over the 5-7 target, accepted), magic 8-9 at bands 10+ (5 at 3-5, wizard mana), solo wins 68-74% and rests every fight. **Review:** accepted: foes still focused the weakest member (the measure hid it with an unplaced company and a classless leader), so spread is now full at the band; help wording. Rejected: class-aware default formation (not needed). Softening by company level kept as zone-fixed (a step back to full strength under the band, never stronger for a stronger company).

**Phase 72 built, reviewed and merged (2026-10-07, PR #168 plus review fixes): backgrounds.** [Decisions](plans/2026-10-07-phase-72-backgrounds.md). A life story's picks become the leader's member tags (`homeland-<id>`, `upbringing-<id>`, `trade-<id>`) through the story-event tag-source seam, so story-event choices (`require: {tag: ...}`) and town lines (`member_tag:`) can react to background, homeland and upbringing. Leader-only (companions have no life story). A town line for a member's tag beats a plain line of the same kind until the listener has heard it lately. Test content: one background choice in each test scene and two town lines. `help lifestory` (alias `backgrounds`), `help events`, `help townsfolk` and the tutorial hints explain it. No stat or combat effect.

**Phase 72 review (Opus review thread, independent subagent plus lead checks; `make smoke` passed).** Accepted and fixed: (1) a background town line always beat plain lines, so a soldier heard the same drill line of every boss; it now wins only while not among the listener's last 12 tellings (`TestABackgroundLineIsNotSaidOfEveryDeed`, `TestABackgroundLineHeardLatelyJoinsThePlainOnes`). (2) An open background choice only named the leader, so nothing showed the past opened it; the note now reads `(Aldous, life story: a soldier)` in the terminal and the web modal (`because` field; `TestABackgroundChoiceSaysWhatOpenedIt`, JS test), leader only (`TestACompanionsTagIsNotCalledALifeStory`). (3) Closed hints read "needs you, once a soldier" to a player who never was; now "needs a leader who was a soldier", and a hintless tag reads `life story: a soldier` instead of the raw tag (`lifestory.TagName`, rune-safe). (4) A misspelt shipped tag would load and never open; `TestEveryShippedChoiceTagIsALifeStoryPick`. (5) Help and decision doc updated (help events no longer says a closed choice opens later; life stories are locked). Checked: pre-72a characters and partial life stories give no or partial tags and nothing breaks; tags are derived, never saved; no race in the townsfolk memory read. Rejected: `{who}` in a tag line fills every member the deed names (today's boss and relic deeds name only the leader; note for line authors); new choices inserted mid-list renumber a saved page's later choices (test content only, the world is temporary).

**Phase 71 built, reviewed and merged (2026-10-07, PR #167 via the review PR): trophy enchanting.** [Build decisions](plans/2026-10-07-phase-71-trophy-enchanting.md). Hunted creatures drop small trophies by race (about 20-25% an ordinary kill, doubled for an elite, a boss always; each company rolls its own; none in Training); an enchanter works one into a weapon or armor with `imbue [item] with [trophy]` for 25 gold a tier. The piece grants the trophy's effect while worn, through the same gear-effect path as relics (each trophy and all of a wearer's enchants together at most a sixth of an effect's cap since the review, relics trimmed to their cap with signature and awakenings), is permanent, one to a piece, and never sells for more than the plain piece (no spec override). The enchant is saved on the item, so it follows it into the pack and a companion's gear. Shown on `look`, GMCP and the web gear tooltip, tagged in lists, named by `why`, and listed by the bestiary's habits tier. New `items.TrophySpec`, `Item.Trophy`, `internal/loot/trophy.go`, `internal/usercommands/imbue.go`; five test-world trophies (40001-40005), enchanters in Frostfang and the test area; `help enchanting`, web quick-menu entry, tutorial hint. (Sonnet build thread.) **Review** (Opus review thread, independent reviewer subagent; `make smoke` and the full race suite passed): a new sim showed the build's limits (half the cap a wearer) let a fully enchanted company win every hard even fight (74% -> 100%, health lost 180 -> 15, worth two or three levels), so the limit became a sixth of the cap (+2 Attack, +2 Evasion, +1 damage, +1% reduction; a full set now worth about a level); also fixed `cargo put` wiping an enchant, the enchanter taking a trophy a capped relic could not use, `imbue` picking the trophy over a worn piece, and the web inventory missing the enchanted tag; `TestPacklordHoundHobblesFoesBelowThreeQuartersAtRankFortyFive` (level 44) failed once in the full race run and passed 20/20 alone and on a package re-run (the round's own blows can drop a 60% foe below half before the hobble check; untouched here); details in the [plan](plans/2026-10-07-phase-71-trophy-enchanting.md#review-opus-review-thread-independent-reviewer-subagent-plus-lead-checks).

**Phase 70 built, reviewed and merged (2026-10-07, PR #164 plus review fixes): errands.** [Plan and decisions](plans/2026-10-07-phase-70-errands.md). From an inn (a room tagged `inn`) a companion who is with the leader and not fighting can be sent on an escort, hunt or scouting job of half an hour, two hours or eight (`errand send [member] [job] [length]`, `errand recall`, `errands`; an Errands tab in the Company window). The errand is real-time and saved with the companion (`Companion.Errand`); it returns when due and the leader is online and free, with modest gold scaled by the zone's level band, a small gathered find counted at full worth with the rest of the pay in gold, word of a lair the company has not slain, or a lasting wound (risk grows for each level under the band). The outcome is fixed at the send from a saved seed, so restarts cannot reroll it. An away companion is absent everywhere a separated one is (`Companion.Away()`): no battles, camp, load, banter, bonds, survival or formation placement. Each return is a chronicle deed (`errand`). Help: `help errands` (hubs, keywords, tutorial Departure hint). Tests: pure rules, module unit tests with a fake world, a brawl-world wiring test through the real commands and round tick, `help_errands_test.go`, GMCP feed, and the dock browser check.

**Phase 70 review (Opus review thread, independent subagent plus lead checks; `make smoke` passed).** Accepted and fixed: (1) the send required "not in the formation", but an unplaced companion still fights (`combat_engagement.go` counts every companion in the room) and all four companions are auto-placed, so the gate only added a `formation clear` step and its help wrongly said unplaced members sit out; any present companion can now go, leaving its cell and taking it back on return or recall when still free (`Record.SendAway`/`BringBack`, `TestAPlacedCompanionLeavesItsCellAndTakesItBack`, `formation` lists it as away). (2) An item result replaced 44-264 gold with a find worth 3-20, so a hunt paid less than an escort; the find now counts at full worth and the rest is paid in gold (asserted in `TestItemOutcomeNeverWorthMoreThanTheGold`). (3) Loyalty drift and desertion still ran on away companions, contradicting the help; they skip a companion on an errand (`TestNoDriftOrDesertionWhileOnAnErrand`). (4) The Errands tab offered Send during a fight, journey or rest; the panel now says why not (`TestErrandPanelNotHereWhileBusy`). (5) "Escort: no danger" was wrong for an under-levelled escort; reworded. (6) A short hunt pays 44, not 45 (doc fixes; "12 times" corrected to 6). (7) Guarded the wound path against a missing snapshot. Economy check: a level-8 companion averages about 23 gold per short escort (about 46 an hour, needing a trip back to an inn each half hour) and about 120 per eight-hour long errand; four companions on long errands earn roughly 450-500 gold in eight hours of idle time, and every sender is one fighter fewer, against about 32-48 gold cache per won level-8 fight plus loot in active play, so errands stay well below active play. No loop yields extra rewards: the seed is fixed at the send and never shown, recall pays nothing, dismissal voids, the save comes before payment and a failed save retries the same outcome. Rejected: an errand item can push the leader over the carry limit (`StoreItem` does not check; items weigh almost nothing, same as other rewards); capping pay for a low-level companion sent from a high-band inn (wound risk climbs 8% a level to 60%, so expected pay barely moves, and reaching such an inn means crossing those zones). UI: Errands tab checked; notes updated for the formation change.

**Phase 69 built, reviewed and merged (2026-10-07, PR #163 via the review PR): weapon stances.** [Build decisions](plans/2026-10-07-phase-69-weapon-stances.md). Each company member can take one stance per weapon family, set between battles with `stance [who] [heavy|wall|quick|keen|off]`: heavy blows (two-handed, +30% damage, -15 to hit), shield wall (+12 block, -30% turns), quick draw (bow, +25% turns, -15% damage), keen edge (dagger, +10 critical points, -10% damage). A stance is kept with the member (saved in the strategy registry, pruned and purged with it) and does nothing unless the weapon is in hand. Applied each round in the tempo fill onto `ClassRT.Stance`; combat reads hit, damage, crit, tempo and block through `Character.StanceEffect()`; `ExpectedDamage` reads it too. The battle log names each stance in force and `why` names it on every blow it changed. New `internal/stance`, `internal/hooks/combat_stance.go`, `modules/strategy/stance_command.go`, `Company` GMCP `members[].stance`, Combat tab Stance button, battle caption, `help stances`, a tutorial hint. Balance (100 fights a variant, level 10 5v5 mirror): heavy 90 -> 86% wins, wall 92 -> 96% (15% less health lost, longer fights), quick 100 -> 100%, keen 70 -> 81%; expected damage at even stats 0.95 to 1.08 of the plain weapon. (Sonnet build thread.) **Review** (Opus review thread, independent reviewer subagent; `make smoke` and the full race suite passed): re-measured at 200 fights a variant (heavy 91 -> 93%, keen 82 -> 81%, wall 92 -> 92%), so the build's keen and wall gaps were noise and nothing was retuned; fixed the pre-battle estimate ignoring stances, a dual-wielder's second weapon taking a stance that doesn't fit it, the stance being re-read from the store every round (now once a battle), `stance heavy` with no one named, and help wording (incl. `help opinions` no longer calling story tags "stances"); rejected and deferred items are in the [plan](plans/2026-10-07-phase-69-weapon-stances.md#review-opus-review-thread-independent-reviewer-subagent-plus-lead-checks).

**Phase 68 built, reviewed and merged (2026-10-07, PR #160 via review PR #162): towns that remember.** [Plan and decisions](plans/2026-10-07-phase-68-towns-that-remember.md). A mob template with `townsfolk: [tag]` is a talker: on its idle turn it says what the company's chronicle holds that it has a line for, to the player by name, once per deed (kept through restart; a deed fades after 14 days unless a line sets 1-90), otherwise a weather or time-of-day line, otherwise nothing. Lines are YAML by deed kind (and optional ref, tags, zones), so the replacement world brings its own; a told line can leave a company flag (a mark) that later lines and story events read, and `member_tag` is the seam for phase 72 backgrounds. The idle-chatter limits still apply. No price effects (a fame discount would be a profit lever). `townsfolk`/`renown` command, `help townsfolk`, Departure tutorial hint, and a "What the towns say of you" block in the web Chronicle tab. Test content only: Gossip Corner (room 90015, mob 90301). Tests: rules, module with a fake world, the real idle turn, seams, help, a browser check. (Sonnet build thread.) **Review** (Opus review thread, independent reviewer subagent; `make smoke` and the full race suite passed): fixed company deeds (boss, relic, mercy) reading "The company" and never matching `member_tag` (now the leader's), a deed used up when the chatter limits held its line back (now recorded only once said), Gossip Corner's exit pointing away from the hub, case-sensitive member tags, name targeting (now `@id`), and a stale "not yet spoken of" view; rejected and recorded items, and the call that no price effects matches the spec and the economy rule, are in the [plan](plans/2026-10-07-phase-68-towns-that-remember.md#review-opus-review-thread-independent-reviewer-subagent-plus-lead-checks).

**Phase 65 built, reviewed and merged (2026-10-07, PR #159 via the review PR): bonds between companions.** [Build decisions](plans/2026-10-07-pillars-phases.md#phase-65-build-decisions-2026-10-07-full-autonomy). Each pair of companions has a saved bond (-100..100) moved by camp rests, won battles, banter, shared opinions (Phase 64's `Observe` seam), stepping in and refused guards, each with a per-pair real-time cooldown; time together stops at +50 and -25 (review). Friends (25+) step in for a friend at 40% health or less (once a battle, twice for kin, two a battle company-wide); a guardian set to guard a rival refuses; a rivalry at -85 warns and at -100 the less loyal of the pair leaves. New `internal/bonds`, `Record.Bonds`, `modules/company/bonds.go`, `internal/hooks/combat_bonds.go`, banter friend/rival lines, `bonds` command, Bonds tab (`Company.Bonds`, phone width checked), `help bonds`, a tutorial hint. Balance (80 fights, five foes, L5/L15): none 75/71, friends 81/82, kin 80/78, one friend pair 80/78, all rivals 57/51, one rival pair 55/53; fair fights unchanged. Gold, experience and power are untouched.

**Phase 65 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed, each with a regression test: (1) a rest that ended on its timer wrote bonds, and could send a rival away, on the camping timer's goroutine; camping's real scheduler now queues onto the game loop, as expedition's does; (2) a failed bond save kept the unsaved value, cooldown and desertion flag (the rollback shared slices); it now restores the whole registry; (3) time together lowered a clashing pair to "can't stand", so a guardian could refuse its ward with no act of the leader's; it now stops at -25, and only opinion splits and refusals make rivals; (4) a warned pair at -100 left on a camp that could not move the bond; now only a real clash; (5) help corrections (fourth clash pair, alignment rule for won battles, opinion cooldown, per-guardian steps). Added a mid-fight leave test. Decided: the guardian-ward rivalry cost (about 18-20 points) stays, with `bonds` and the Bonds tab now saying to set another ward. Rejected: a bond guard by a companion marked to leave; one banter line without `{other}`. [Review decisions](plans/2026-10-07-pillars-phases.md#phase-65-build-decisions-2026-10-07-full-autonomy).

**Phase 67 built, reviewed and merged (2026-10-07, PR #157 via the review PR): relic awakenings.** [Plan and decisions](plans/2026-10-07-phase-67-relic-awakenings.md). Every shipped relic (16) has two or three awakenings: slay a number of foes of a kind, defeat a lair's master, or cross into a zone, done while the relic is worn by a living member standing with the leader. Progress is saved on the item (`Item.Awaken`); a woken power joins the relic's gear effects at once (`items.GearEffects`, the gear cache is keyed by the awakened mask), so it works wherever a signature does. Sources: the kill credit in `mobcommands.Suicide`, the chronicle's boss deed through a new `chronicle.OnRecord` seam, and a `RoomChange` listener for zone crossings (each wearer credited by its own step). Each awakening adds at most a third of an effect's cap and signature plus awakenings never pass it (validated at load); sale value is unchanged; counts are finite, so there is no farmable loop. Waking tells the leader, refreshes the web gear window and writes an `awakened` chronicle deed. Progress shows in `look`, `inventory`, GMCP relic lines (gear tooltip gold/dim, Company window rows) and, for blow-raising powers, on the `why` strike line. `help awakenings`, relics/chronicle/inventory pages and the Departure tutorial hint updated. Not built: "carried while devout" (needs phase 73 creeds).

**Phase 67 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed, each with a regression test: (1) a companion's place awakening could never wake on foot: companions follow a moment after the leader, so the leader's crossing found them still in the old room, and their own `RoomChange` was ignored; each member is now credited by its own crossing, a companion while its leader is in that zone (`TestACompanionFollowingIntoAZoneWakesItsPlaceAwakening`, `TestACompanionIsCreditedByItsOwnZoneCrossing`). (2) The ogre and giant-spider slay awakenings named races that only their bosses have, so they asked for 15 to 30 boss kills; Ogrebane and the Ogre-hunter set now count ogres and goblins, Widow's Weave spiders and other insects (`TestShippedSlayAwakeningsNameFoesThatSpawn`). (3) A full Deepdelver set, fully awakened, gave healthpct 11 over its cap of 10 (set bonuses were not counted with set pieces' awakenings); the helm's lair awakening is now +2 (`TestAFullAwakenedSetStaysWithinTheCaps`). (4) Logging in inside a zone counted as crossing into it, and a hidden (sneaking) leader's crossing did not count; now a step from no room is not a crossing and sneaking counts. Also: the 36d/67 character gear tests now load races when run alone. Accepted, not changed: progress is stored by awakening position, so reordering a relic's awakenings in its file moves progress (world is temporary; keep order when editing); a companion's progress is saved with the next autosave, copyover or gear change, like its experience. Checked fine: one slay and one boss deed per company, executed bosses count once, Training kills excluded, leader and companion gear changed in place, gear cache keyed by awakened mask, sale value unchanged, `why` notes only powers in force. `make smoke` passed; gear tooltip checked at desktop and 390px (`scripts/browser/relic-check.mjs`, woken lines gold, sleeping dim).

**Phase 64 built, reviewed and merged (2026-10-07, PR #155 via the review PR): companion opinions.** [Build decisions](plans/2026-10-07-pillars-phases.md#phase-64-build-decisions-2026-10-07-full-autonomy). Each of the six personalities likes and dislikes a few of twelve kinds of choice (mercy, execution, an inn bed, a rough camp, sharing your own food, selling a relic, and six story stances an event choice can be tagged with). When a choice is made the companions who saw it say one line each and move ±2 loyalty through the real record: approval stops at 80, disapproval at 30 (above battle nerve's 25), a companion speaks on a kind again only after a real-time cooldown (10 min to 2 h), and a choice is counted once; one alignment already moved on a mercy choice stays quiet. New GoMud-free `internal/opinions` (kinds, leanings, lines, bounds, `Observe` seam for phase 65), the `company.Opinion` seam, and `modules/company/opinions.go`; sources wired at their real entry points (`hooks.resolveMercy`, `camping` camp rest and inn stay, `company meal` from the leader's own pack, `usercommands.Sell` of a relic, a story choice's `stance:`). `opinions [member]` / `company opinions` and one line in `company inspect` read them; deeds that name a companion come from the chronicle by key. Web: an Opinions tab in the Company window (`Company.Opinions`, phone width checked). Help: `help opinions` (indexed, linked from the pages it touches, `mercy`, `banter`, `camp`, `inn`, `relics` and `company-meal` updated), a tutorial hint. Tests: package units; module tests for reaction, once-per-choice, cooldown, bounds, witnesses, observers, the real mercy answer, shared food and the cargo exception, panel and command text; camping inn and camp rest; the relic sale command; story stance and its load check; GMCP feed; help render; Playwright checks. Nothing changes balance, gold or experience.

**Phase 64 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed, each with a regression test: (1) a mercy answer whose company save failed, retried after the leader walked away, indexed the emptied queue for the opinion's subject and panicked the server (`TestFailedMercyRetryAfterLeavingDoesNotPanic`). (2) `sell junk` sold a relic marked as junk with no reaction, a way round a devout companion's view; it now reports one `sell-relic` per command (`TestSellingJunkRelicsLetsTheCompanyReactOnce`). (3) `Company.Opinions` was resent about once a minute because each note's "ago" was part of its change key; the key now leaves the ages out (`TestOpinionsAreSentWhenTheyChange`). (4) The relic sale remembered `DisplayName()`, colour tags and all, which would show raw in the web tab; it stores the plain name. (5) For Phase 65, observers now hear the resolved witnesses (silent ones included) in `Choice.Witnesses` even when the source named none, and a companion already deserting (`MoraleDesert`) has no say (`TestObserversHearEveryWitnessAndDesertersStayQuiet`). (6) The story stance's live wiring (member keys to companion ids, leader left out) had only a fake-world test; `TestAStanceReachesTheRealCompanyByCompanionKey` drives the real company module. Help: `help opinions` says water counts as sharing and `sell junk` counts as a relic sale. Rejected: a story choice tagged with a stance that also has a `loyalty` outcome moves loyalty twice (the shrine's kneel); the outcome is the company's shared feeling and the stance each temperament's own view, both authored on purpose, and the scene is once per company. Returning scenes may carry a stance: the 10-minute cooldown and the 80 ceiling bound it, as they bound every kind. Checked fine: one report per choice from each real source, matching by companion id/key throughout, durable real-time cooldowns (restart does not reset them), the 30 floor above battle nerve and desertion, the web tab at desktop and 360px (Playwright). `make smoke` run in review.

**Phase 66 built, reviewed and merged (2026-10-07, PR #152 via the review PR): the bestiary.** [Plan and decisions](plans/2026-10-07-phase-66-bestiary.md). Beating a kind of creature enters it in the leader's bestiary: lore at the 1st kill, defences at the 3rd, habits and weaknesses at the 6th (a boss teaches at its 1st, 2nd and 3rd), every line generated from the creature's template (armor, evasion, poison, role, spells, wind-ups and what breaks them, who it aims at, gear it carries), nothing hand-written per creature. It reads the character's existing kill tally (`bestiary.KillsOf`, the single seam to switch to `internal/chronicle` later), so there is no new state and it survives restart. `bestiary`, `bestiary [name]`, `bestiary [zone]`; `consider` adds a Bestiary line for visible foes; a kill that takes a kind up a tier says so; the web client gets a Bestiary dock tab (`Char.Bestiary` GMCP) and the Battle view, Combat tab and battle-screen caption name the habits known of each foe. `help bestiary`, tutorial hint. Not built: drop chances (phase 71).

**Phase 66 review (Opus review thread, independent subagent plus lead checks).** No leaks found: entries, `consider`, `Char.Bestiary` and the Battle view read only kinds with a kill, tiers are gated on the server, hidden foes and dark rooms are left out. Accepted and fixed: (1) an encounter's boss is flagged on the live mob only, so its kill announced boss thresholds (1/2/3) while the entry used the template's (1/3/6); the tier-up line now reads the template (`bestiary.TierOf`, `TestBestiaryTierUpFollowsTheTemplate`, which also covers the defences line). (2) "Armor turns aside about N% of each blow" doubled the average (the roll is 0 to N-1); now "up to N%, about half that on average". (3) Help said any company kill counts; it counts when you or your companions hurt the foe and you are standing in the room (`eligibleContributors`). (4) Same-named kinds in two zones hid behind the better-known one, and a zone could never win over a name; all same-named entries now show, and a zone named in full lists that zone. (5) Lore and list showed the template's level, which room and encounter spawns override; removed (consider gives the real one). (6) "casts spells" was noted for any spell book, though a fighter only casts with a `cast` combat command; now checked. (7) The wind-up line now reads the ability's turns; the targeting line no longer quotes the template's random share (groups raise it). (8) The plan claimed clients that never open the tab get no refresh; the web client asks at load, so every web client gets a small payload per battle end; accepted, plan corrected. Added `TestBestiaryNamesNoHabitsOfAHiddenFoe`. Confirmed: the `TestEnemyLabelsOnSecondarySurfaces` change is right (the tier-up line names the kind on purpose; the death narration is still checked). Kept: the Bestiary dock tab on by default (one tab, a clear empty state). Chronicle seam: not switched; the chronicle records boss kills only, not every kill, so `KillsOf` stays on the kill tally (follow-up only if the chronicle gains per-kill counts). Not fixed: kills outside a battle (mercy execution) refresh the tab only at the next battle end. `make smoke` passed.

**Phase 63 built, reviewed and merged (2026-10-07, PR #150 via the review PR): company chronicle.** [Build decisions](plans/2026-10-07-pillars-phases.md#phase-63-build-decisions-2026-10-07-full-autonomy). A durable, capped (300 deeds) log per company with a lifetime tally, read in prose with `chronicle [kind] [page]` / `chronicle all` and in a Chronicle tab of the web Company window (filter by kind, phone width checked). New GoMud-free `internal/chronicle` (entry shapes, 13 kinds, prose, `Filter`, and the query seam `Record`/`Query`/`Count`/`Has`/`Total` that phases 64, 67, 68, 70, 73, 76 and 77 read) and `modules/chronicle` (saved logs, command, `Company.Chronicle` GMCP, the leader's own death written at the death itself in `usercommands/suicide.go`). Deeds are written from their real sources: recruits joining, dismissal, desertion, companion falls, raising and loss, defeat scenarios, boss kills, relics found, the mercy answer, class promotions, and a story event's last answer. Help: `help chronicle` (indexed under the road, linked from `adventure`, `company`, `events`, `webclient`), tutorial Departure hint. Tests: package units; module persistence, failed save, load error, purge, test-area snapshot, command paging and filters, panel payload; real-path tests for boss and relic kills (`Suicide`), the mercy answer, recruit, dismiss, desertion, fall, raise, loss and promotion, a defeat scenario, and a story event's end; help render; Playwright checks in `dock-windows-check.mjs`. Nothing changes balance.

**Phase 63 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed: (1) the `history` and `record` help aliases took over GoMud's own `help history`; removed (`TestChronicleHelp` checks `help history` is its own page). (2) Relic deeds stored `DisplayName()`, colour tags and "(legendary, unidentified)" included, which showed as raw tags in the web tab; now the relic's plain name (`TestBossKillAndRelicAreWrittenInTheChronicle`). (3) The Chronicle tab was blank after the Company window was closed and opened again; `update()` now redraws it (browser check). (4) A scene that ended on a move outcome was filed under the room the company was sent to; the deed is now written before the outcomes run (`TestASceneEndingOnAMoveIsWrittenBeforeTheMove`). (5) An executed boss wrote no `boss` deed, so bounty and lair checks (76, 67) matching `boss` + `mob:<id>` would miss it; an executed boss is now a boss kill as well as an execution. (6) The leader's fall came from the queued `PlayerDeath` listener, so it read after the defeat it caused; it is now written in `usercommands/suicide.go` at the death, with the killer, and not in the Training zone (`TestDefeatScenarioIsWrittenInTheChronicle` checks the order). (7) A defeat was placed at the room the company woke in, not where it fell; fixed (same test, a capture). (8) The last page's footer pointed at itself. (9) Relic prose said "took" for a roll onto the corpse; now "found". Lead additions: entries carry member `Keys` (`leader`, `companion:<id>`, ids never reused) and `Filter.Key`, so 64, 65, 73 and 74 can tell apart companions who share a name; companion fallbacks use `#id`; the help intro no longer promises that the world remembers; the web note says how many are shown only when some are not. `make smoke` passed, now with a `chronicle joined` step after the restart. Rejected: relic deeds in the Training zone (no relics drop there). Follow-up for 76: kills of named groups that are not bosses are not recorded; 76 adds that kind.

**Phase 62 built, reviewed and merged (PR #149 via the review PR): battle lines that explain themselves (2026-10-07).** [Plan and decisions](plans/2026-10-07-phase-62-explained-battle-lines.md). The combat engine now records each weapon strike's roll with its parts as it resolves it (chance and what moved it, the roll, the defence met and its chance, quality, dice, armor, ward and aura, named modifiers); `why`, `why list` and `why [number]` read the leader's latest fight back in plain words, and the Combat tab keeps a "Last rounds" list opening into the same breakdown (`Company.Battle.Event` `explain`). The battle summary adds Damage taken, Never landed (and why), Moves and Sigil. New: `combatstream.Strike`/`RollLog`, `battle-rounds.js`, `help battlelog` with hub links and a tutorial hint. Tests: engine strikes against the round's own damage, explain wording, summary extras, roll log through `emitAttack`, GMCP explain and privacy, real-round wiring (`why` after a brawl), JS test and a Chromium check at desktop and 280px.

**Phase 62 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed: (1) `why` (telnet) explained rounds against a foe the leader can't make out (dark room, hidden), giving away its armor and chances; the roll log now skips them and anything not the leader's own company's round (`TestRollLogSkipsUnseenFoesAndOthersRounds`). (2) "armor N took M" counted ward, aura and divine-shield takes as armor; armor's own share is now recorded apart (`ArmorTook`). (3) The damage note claimed weather effects that don't exist and folded leadroot in; class effects, leadroot, and hunger/thirst/meal now get their own lines. (4) Armor pierce (Piercing Bolt, Iaijutsu) lowered the shown rating silently; now noted. Iaijutsu no longer claims a surer hit (it adds crit chance). (5) A hit chance held by the to-hit limits now says so, so the parts add up. (6) A Marksman's crit through a shield read "it was not blocked"; a Perfect Parry read as a rolled 100%; both now say what happened. (7) Pet damage was in the round total with no strike; pet blows are now their own breakdown line and don't count as the member's swings. (8) Headings called a round critical when the critical strike was fully absorbed. (9) Web: a mixed round (one dodge, one hit) read "turned aside"; the last paced blows lost their names after the battle ended; the Combat tab redrew on every round message (now once a frame). (10) Help: roll scale (1-100, at or under lands), armor as a random share up to its rating, unseen foes left out, log lost on restart. Removed the unused `combatstream.Headline`. Smoke gained a `why` step after the tutorial fight. Not done: the web heading's crit flag still comes from the event (rare: a crit absorbed whole beside a landed blow).
**Phase 61 built, reviewed and merged (2026-10-07, PR #146 via the review PR): battle orders.** [Build decisions](plans/2026-10-07-pillars-phases.md#phase-61-build-decisions-2026-10-07-full-autonomy). Up to three "when this, do that" rules per company member, set before a battle with `orders [who] add [condition] then [action]` (also `remove`, `up`, `preset`, `clear`; refused in a battle), read each round before the member's role and target rule. Conditions: ally or self below a health share, a foe chanting, a boss, the first round, a foe of a kind (caster, healer). Actions: heal, break (turn on the chanter, boss or foe), guard, strongest, hold. The battle log says "as ordered" (text line and an `order-fired` event the web battle log narrates). New `internal/orders` (GoMud-free: parse, validate, evaluate, presets), storage in `modules/strategy`, the combat hook `internal/hooks/combat_orders.go` and a guard-order read in `guardianFor`, `members[].orders` in the GMCP Company payload, an Orders list and menu in the Combat tab. Help: `help orders` (aliases `order`, `battle-orders`), linked from `help combat`, `help strategy` and `help webclient`; a tutorial hint in the battle lesson. Tests: parse/evaluate/caster units; the module command, persistence, prune, purge and snapshot; real-round wiring through `orders` and `DoCombat` for heal, self-heal, break (melee and spell), guard, hold, strongest and the in-battle refusal; GMCP payload; help render; Playwright checks in `dock-windows-check.mjs`. Left out, with reasons: "fall back a row" (formation is durable) and forcing a fighter's class ability. No default orders, so nothing existing changed balance.

**Phase 61 review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed: (1) `hold` set the mana reserve to 100%, which let Lightning through (the reserve never covered it) and blocked summons, weather, wards and buffs that help says go on; a `Hold` flag on `strategy.Situation` now stops only attack spells, Lightning included (`TestDecideHoldKeepsAttackSpellsButNotASummon`). (2) A `guard` order fired with no guard left or the ward out of guarding ground, logging a guard that never happened and hiding the next order; it is now passed over (`TestAGuardOrderWithNoGuardLeftGivesWay`). (3) `first` compared the round with `StartRound`, so a battle that begins at a round's end (the next group stepping up) never had a first round; it is now the first round the battle's orders are read (`TestFirstOrdersRoundIsTheFirstRoundRead`). (4) A healer or caster with no attack spell fell back to a weapon swing on `break`, giving up its heal; now only fighters strike, others pass the order over (help says so, and that a break spell ignores the reserve). (5) A foe condition took only the first chanter/boss/kind; it now tries each in turn, so one out of reach gives way to one in reach (`TestEvaluateTriesEachFoeThatMeetsTheCondition`). (6) A hold or guard note could be swallowed by the previous battle's streak; streaks are now per fight. (7) `hold` fired for fighters (who cast nothing); no longer. (8) The same order could be added twice; refused (`TestOrdersAddListRemoveAndReorder`). (9) Test-area snapshot restore did not validate orders; it now cleans them like a saved file. (10) Web battle log verb forms for "You"; help wording on downed allies and `first` with several groups. `make smoke` passed, now with an orders step (set before the tutorial fight, still there after a restart). Coverage added: `boss` and `foe healer` in real rounds (`TestABreakOrderTurnsOnABossOrAHealer`); the Orders list and menu at phone width in `dock-windows-check.mjs`. Omissions kept as follow-ups: "fall back a row" (formation is durable; a battle-only column touches every reach gate) and forcing a fighter's class ability (`abilityPass` gates and cooldowns); both are larger than this phase. Not done: the Orders menu still offers an order already set (the command refuses it); Fire Flask is not an attack spell for `hold` (it costs flasks, not mana).

**Phase 60 built, reviewed and merged: story events (2026-10-07, PR #145 via the review PR).** [Plan and decisions](plans/2026-10-07-phase-60-story-events.md). Data-driven scenes (YAML pages with choices) open on a room, tag, journey-end or camp-rest trigger; choices can need a member's skill, class, personality, alignment, level or an item, and the best qualified member takes a gated choice and is named; risks show as words. Outcomes: wound, ailment, need, item, lose item, gold, loyalty, battle, move, flag. The answer is saved before outcomes run, so a crash never doubles one; a waiting page holds the company in place and survives restart; once-per-company or cooldown. New seams: `company.AdjustLoyaltyOnce`, `encounters.StartGroup`, `camping.RestEnded`, `MemberView.Personality`, `storyevents.MovementBlocked` and a tag-source hook for phase 72. Three test scenes (a climb, a stranger, a shrine) in new test-area rooms 90011-90014. Web modal over GMCP `Event`, `event` and `choose` commands, `help events` plus hub links and a Departure tutorial hint. Tests: rules, module (fake world) and wiring tests through real `go` steps, a JS test, help render. (Opus thread); accepted and rejected findings will be recorded here.

**Phase 60 review (Opus review thread, independent subagent plus lead checks; `make smoke` passed).** Accepted and fixed: (1) leaving a scene after an answer (death, teleport, being moved) dropped it without marking it done, so the gorge's pack could be looted again; a scene left past its first page is now done (`TestLeavingAfterAnAnswerCountsTheSceneDone`). (2) A page could be answered during a camp rest, moving the company or starting a fight mid-rest; answers now wait for the rest to end (`TestNoAnswerDuringARest`). (3) The web modal's number keys kept answering after the modal was closed; they stop once the scene is off screen. (4) A double click answered the next page too; the client sends `choose N <event>/<page>` and a turned page's answer is ignored (`TestAnAnswerForATurnedPageIsIgnored`, JS test). (5) Phase 68 had no way to read or set company flags; added `storyevents.CompanyFlags`/`HasCompanyFlag`/`SetCompanyFlag` (`TestCompanyFlagsSeam`); the tag-source seam was already usable. (6) Loyalty lines named companions whose loyalty did not move (`TestALoyaltyLineOnlyWhenLoyaltyMoved`). (7) Returning scenes could be farmed (the shrine paid gold and loyalty every 30 minutes); validation now forbids gold, item or loyalty gains on an event with a cooldown, and the shrine comes once (`TestAnEventThatReturnsCannotBeFarmed`). (8) Unknown class or personality names passed validation; now checked. (9) The failed gorge descent said "you land on the ledge" but left the company at the top and ended the gorge; it now moves to the ledge, and the ledge has an `up` exit so no data change can strand a player. (10) The camp trigger fired even when the camp reward's save failed; now only once the reward is settled. (11) The catalog was built while holding the module's leaf lock; now built outside it. Doc fixes: arrival triggers take a zone only; `require.item` reads the shared cargo. Rejected: login push of `Event` racing the GMCP handshake (it fires on PlayerSpawn, the same moment the Char and Company panels are sent, and `event` redraws it); wiring tests for real arrival and camp paths (the arrival and camp listeners are thin; covered by the fake-world trigger tests plus the camp module's `RestEnded` test).

**Camp music and inn gigs built, reviewed and merged (2026-10-07, PR #141 via the review PR).** [Spec and decisions](plans/2026-10-07-camp-music.md). A Music skill in four families, instruments in four tiers (masterworks looted, resellable), a camp song fixed at rest start whose effects the grant applies (strings, winds, drums, voice; ensemble upgrade that does not stack with the pavilion), raid and weather costs in the single roll, practice levels, persisted state, paid inn gigs (19:00-21:00, two families, per-inn-evening and 3-hour limits, company held until it ends), banter `onSong`, GMCP and web Camp tab. Help: `help music`, `instruments`, `gigs` plus camp, duties, inn, gear and recipes pages; tutorial camp hint.

**Camp music review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed: (1) Drumbeat sizes stacked across rests (five separate buffs; +15 speed possible before one battle); a grant now ends any earlier lift (`TestASecondRestReplacesTheDrumbeat`). (2) Older (Legacy) characters could craft every fine instrument without its page, and a page they read crumbled as "already known"; patterns now live in their own `patternbook` (`cookbook.KnowsPattern`/`LearnPattern`), so a page is always needed and reading one never ends a Legacy recipe book (`TestFineInstrumentNeedsItsPageEvenForAnOlderCharacter`, `TestUsingAnInstrumentPageLearnsThePattern`). (3) `help gigs` claimed the whole company must be at the inn; only those present play, and the page now says so; "half more again" corrected to "half more". (4) UI: the Camp tab and `camp music` never said the song draws raiders; both now show the added raid and thief chance, and the Camp tab names the teacher and fee. (5) Gig pay was a third of the spec's "an hour's hunting"; raised x3 (tuning note in the plan, `TestGigPayTuning`). (6) The voice cured the chill caught by sleeping cold the same night; the song now runs before the chill. Added tests: `FadeAilment` (`TestFadeAilmentShortensAndSaves`), death mid-gig pays nothing and spends no limit. Checked, kept: locking, persistence (song on the rest, pending grant rolled back), single raid roll, spoiled rest grants nothing, practice counted once, ensemble does not stack with the pavilion, gig pay saved as settled before credit, world clock only read, bought/crafted instruments refused by every sale path, masterworks only from flagged bosses. Not done: no test drives the song banter through `startSongText` (banter firing is covered by the banter package; low risk); `music craft` spends inputs one at a time after a pre-check, so a mid-spend failure is not rolled back (unlikely; same as cooking).

**World style bible added to the repo (2026-10-07), draft.** [docs/designs/world-style-bible.md](designs/world-style-bible.md) is a snapshot of the Claude Doc Robinson is co-designing in the Geas thread: tone pillars, the premise (a fallen kingdom whose frontier ward broke; magic feared; the old owners are their own non-human race), voice and room rules, NPC chatter rules, status-in-words bands, lore method, adult content lines and sample passages. Builders of the replacement world follow it. Still open: the world's name and the ash, faiths (tabled until races are settled), the old owners' design.

**NPC idle chatter limits (2026-10-07, Robinson's ask in the Geas thread).** Town NPCs repeated themselves because every 4-second round each idle mob rolled its activity level and fired a random say or emote from its `idlecommands` or `onIdle` script, with no memory (a 50% NPC spoke every ~8 s). Now, during a mob's idle turn (`HandleIdleMobs` wraps it in `BeginIdle`/`EndIdle`), its says, saytos, shouts and emotes go through one decision per turn (`internal/mobs/chatter.go`): allowed only when a player is in the room, after `GamePlay.MobChatterCooldownRounds` (60, about 4 minutes), and when the turn's first line is new to at least one player there (from that kind of mob, within `GamePlay.MobChatterMemoryRounds`, 900, about an hour); an allowed turn keeps every line so scripted speeches stay whole, a refused turn drops them all, other idle commands (wander, pathto, lookfortrouble) and anything queued outside an idle turn (combat, replies, conversations under way) are untouched, and mob-to-mob conversations start only off cooldown. Either setting at 0 turns that limit off. Decisions: the memory is cosmetic and in-memory only (a restart forgets it); emotes count as chatter (the style bible asks for silent work, not repeated flourishes); no help page, since players gain no command. Tests: `internal/mobs/chatter_test.go`, `TestIdleMobChatterIsRareThroughTheIdleHook` (150 lines in ten minutes before, 2 after). Review (independent subagent, accepted and fixed): (1) a veteran in the room silenced a line every newcomer had yet to hear, such as the hungry guard's and the king's quest hints that open with the same emote; the memory now refuses a line only when every listener has heard it (`TestAVeteranDoesNotSilenceALineANewcomerHasNotHeard`); (2) the entry said "that line" where the code judges the turn's first line (corrected); (3) a mob in an empty room spent its cooldown on nobody; no listeners now means no chatter and no cooldown (`TestNoChatterAndNoCooldownInAnEmptyRoom`); (4) players who left kept their memory until restart; the whole map is pruned once per memory window (`TestChatterMemoryForgetsPlayersWhoLeft`); (5) the `yell`/`scream`/`holler` and `.` aliases bypassed the gate; they now count (`TestChatterAliasesAreHeldBackToo`). Checked, no change: listeners are taken before onIdle runs, so a script that moves its mob and then speaks is judged by the old room (no shipped mob speaks after moving: the player guide uses `saytoonly`, which is not chatter); no test drives a real conversation file or JS onIdle script through the hook (scripts reach the gate through `CommandScripted`, covered directly). Style bible draft (Claude Doc): [Ashveil World Style Bible](https://claude.ai/code/artifact/fcde5707-6b43-41c1-8771-49ed38763a79).

**Company test speed-up (2026-10-07): `modules/company` under `go test -race` 794 s to 527 s** (same container, plain run vs the `-json` profiling run, so the real gain is a little smaller). Profile: 936 of 987 top-level tests run under 2 s but sum to about 540 s, almost all of it `newBrawl` fixtures; creating roughly 380 shipped item, spell and mob files for each test was a third of the CPU, YAML loads most of the rest. Changes, all in test files: (1) `copyShipped` reads and stages each shipped path once per process and hard-links it into each test's data dir (reason: it is the shared hot path of every wiring test and keeps every assertion); a staging-integrity check in `TestMain` fails the package if a test writes over a linked file in place (tests must remove, then write, as the friendly-effects test already does); (2) `TestAnApothecaryDraughtHealsMoreSplashesAndCleans` throws its eight draughts from one brawl per class instead of sixteen fixtures (about 25 s to about 3 s, same comparison); (3) `TestHotterFlasksBurnHarderThanAPlainOne` runs ten fights a class, not twenty (the 100% vs 70% Sparks gap is far wider than the dice spread). Not done: `t.Parallel` (brawls `Chdir`, swap global registries and providers, so it is unsafe); caching the YAML loads would touch engine packages. Stability: the two changed tests passed 30 of 30 runs under `-race`. The ASHVEIL_BALANCE opt-in sims and `ASHVEIL_BALANCE_ONLY` were already gated and are untouched. Merged from [PR #136](https://github.com/Robinsond76/ashveil-gomud/pull/136) after review. **Review (checked, kept):** no current test writes over a linked file (the suite's guard stays green; tests that write data files write brawl-only fixtures or remove first); a probe test that wrote over a linked knapsack file made the package FAIL through the `TestMain` guard; when `os.Link` fails (another filesystem, no hard-link support) `copyShipped` writes a private copy, so CI and Windows still work; both trimmed tests passed 20 of 20 under `-race -count=20`. Noted, not changed: a test that writes over a linked file would corrupt the shared copy for later tests until the end-of-run guard names it; making the staged files read-only was rejected because it does not stop root and blocks removal on Windows.

**Phase 72a review (Opus review thread, independent subagent plus lead checks).** Accepted and fixed: (1) the web panel could not answer the login offer: its buttons send option numbers and `skip`, which the opener's prompt options never matched; it now reads answers like every other question (`TestCreationOfferTakesNumbersAndSkip`). (2) `appearance edit` and `lifestory choose` could not be left without answering every question (in combat too); `cancel` or `skip` now ends them with nothing changed, and the panel shows Cancel (`TestAppearanceEditAndLifestoryChooseCanBeCancelled`, browser check). (3) A prompt replacing the creation prompt (mercy) left the web panel open, its buttons answering the new prompt; `StartPrompt`/`ClearPrompt` now close it (`TestAnotherPromptClosesTheCreationPanel`). (4) The free line was capped in characters but checked in bytes at commit, so a long accented line wiped all looks (`TestCheckCountsTheFreeLineInCharacters`). (5) Sprite tint threw on an unreadable (cross-origin CDN) canvas; it falls back to stock colours (JS test). (6) Help: `back` starts the part over (not "the last answer"); pronouns are description-only (help claimed battle text used them); a story gives at most +3. Telnet hints now show the skip/cancel answer (they never did). (7) UI, checked in Firefox desktop and 360px: answer buttons now fill their grid cells. (8) `make smoke` did not know the new steps; it now plays them, checks the summary, and logs in a pre-72a character (the shipped admin with a hashed password) to see the offer once, put it off, and not see it again after a restart; it passes. Rejected: `skip` typed as a new character's free line is stored as text (harmless, they can redo); a rejected free line clears the typed text in the panel (minor); `lifeStoryEffectsLine`/`StatMod` map per call (cheap).

**Phase 72a built, reviewed and merged (2026-10-07, PR #142 via the review PR): looks and life story at character creation.** [Spec and build decisions](plans/2026-10-07-phase-72a-character-creation.md). After the archetype step, `start` now asks looks (pronouns, age, height, build, face, skin, eyes, hair, voice, marks, an optional free line) and a three-stage life story (homeland, upbringing, trade), with a summary to redo from; both live in data files (`looks.yaml`, `lifestory.yaml`) so the replacement world renames them. Looks compose into the `look` description; the life story gives +1 to a chosen stat per stage (derived on read through `StatMod`, so it cannot apply twice; +2 per stat and +3 in all at most), a keepsake item (30400-30405) and sometimes a skill's first rung. The trade is the background id phase 72 reads (`lifestory.Background`). Commands: `appearance`, `appearance edit` (free, inns only), `lifestory`, `lifestory choose`, `lifestory [player]` and `creation`; existing characters are offered the steps once at login (`creation-offered`, the phase 56 pattern). Web client: `Char.Creation` GMCP and a creation panel (`window-creation.js`) answer with telnet's own input, so both clients share one state machine; skin and hair colour repaint the player's figure on the map and battle windows and on other members' via `sprite-tint.js` (the generator gains a hair ramp, 67 colours; `make sprites` stays reproducible). Help: `help appearance`, `help lifestory` (aliases `looks`, `description`, `backstory`, `background`...), linked from `help adventure` and the first tutorial lesson. Tests: telnet flow through the real `start`, `appearance` and `lifestory` entry points, persistence, apply-once, the login offer (`internal/hooks`), GMCP payloads, help render, 47 tint tests, and Playwright checks (`creation-panel-check.mjs` in Chromium and Firefox, plus tint checks in the map check). The browser check found and fixed a real bug: the panel read `window.Client`, which is undefined for the client's `const`, so no button sent anything, and a sprite load rebuilt the panel and dropped focus.

**Phase 72a spec agreed (2026-10-07), not built: looks and life story at character creation.** [Spec](plans/2026-10-07-phase-72a-character-creation.md), from Robinson's Bannerlord-style idea in the character creation thread. After the archetype step, `start` asks looks (pronouns, age, height, build, face, skin, eyes, hair, voice, marks) as bands that compose into the `look` description, then a three-stage life story (homeland, upbringing, trade) with small effects (+3 stats in all, a keepsake, a non-combat skill familiarity). The trade is phase 72's background, so phase 72 keeps only its event and town hooks. Web creation panel over GMCP; skin and hair colour on sprites, everything else in words. Robinson approved the defaults: three stages, chosen pronouns, looks changeable free at inns, existing characters asked once at login, age as words only.

**Camp music spec agreed (2026-10-07), not built.** [Spec](plans/2026-10-07-camp-music.md), from Robinson's idea in the camp music thread: a Music skill in four families (strings, winds, drums, voice) with crude, common, fine and looted masterwork instruments; a song before each camp rest gives one small buff per family, three strong families upgrade to Well Rested (no stacking with the pavilion tent), and music raises the raid chance. Inns post timed paid gigs, limited per evening and per three real hours. Robinson's rulings: musicians still sleep and take duties, camp only, gigs pay. Phase number assigned when slotted into the roadmap.

**Server-side cleanup: shared persistence, timers, live-companion roster and name matching (2026-10-07).** Four Go proposals from the [cleanup review](plans/2026-10-06-code-cleanup-review.md), carried out as behavior-neutral refactors. `internal/modstore` (`Load`, `Save`, `Available`) replaces the load/save/availability scaffolding in twelve modules; `internal/modtimer` (`Scheduler`, `Arm`, `Stop`, `Claim`) holds the one-shot timer seam and generation bookkeeping camping and expedition each copied; `internal/livecompanions` is the roster-to-mob walk exposure, walking, camping rest tiers, gathering and archetype utilities each wrote, each still passing its own liveness rule; `company.SplitNameMatches` is the exact-or-substring name scan under `resolveCompanion`, `ambiguousCompanion` and survival's matcher. Each package has unit tests; the modules' existing tests are the integration net. Not done, with reasons in the review doc: moving camping's timer onto the game loop (it changes when world work runs), making every module reject an empty store file (only market does), merging the company module's own "present and usable" predicates, and unifying what a player can type for `#N`, `me`, `leader` or an empty selector (needs help and test changes). `encounters`, `gathering` and `testarea` keep their own store code. Merged from [PR #132](https://github.com/Robinsond76/ashveil-gomud/pull/132) after review. **Review (checked, kept):** all twelve stores read and write the same file names, a missing file still loads the empty registry (exposure and walking keep their non-pointer `newRegistry`), any other read error or a decode error still fails the load and blocks saves with the same messages; camping timers still fire on the timer goroutine and expedition timers still go through the game loop, with the same bump-generation, stop-old, schedule-new order and the same stale-callback check; each live-companion caller keeps its own rule (exposure and walking drop the dead, rest tiers drop dead, separated and constructs, gathering and archetype utilities need a spawned, standing companion in the room); name matching gives the same result, including survival taking the first exact match. Nothing a player sees changed. No findings. Verification:, `make validate`, `go test -race -timeout 30m ./...`, `make js-lint`. Review re-ran `make generate`, `make validate`, `go test -race -timeout 30m ./...`, `make smoke` and `make smoke-world` after merging master.

**Phases 41 and 42 merged (2026-10-07): the test-only world, levels 2-33.** [Plan and decisions](plans/2026-10-07-phase-41-42-test-world.md). Nine new tile-ready zones chain east from Alderbrook (Brindle Downs 4-6, Marrowmere Fen 6-8, Greywatch Pass 9-11, Cinder Hollow 12-14, Thornreach Wood 15-17, Hollowweb Deep 18-20, Ashen Barrows 21-24, Glassvault Depths 25-28, Stormcrown Heights 29-33; rooms 3101-3925), each with a hearth inn, a camp and encounter tables of the existing mob templates; Alderbrook gains a 2-4 band. All five relic bosses (36d) lair in the chain (after review: ogre in Hollowweb Deep, lich in Glassvault Depths, ent, spider queen and abyssal creeper on the summit) and both recipe pages (56) spawn in Brindle Downs and Marrowmere Fen, so every merged system can be reached in order up to the level-30 elite promotion. New `help regions` (road category) lists the chain; `help encounters` and `help travel` link it and the Departure hint points to it. Per Robinson's ruling that the world is temporary, no new mobs, settlements or quests were built; prose is placeholder. Tests: `modules/encounters/world_test.go` (band chain, pacing, lairs for every relic boss, pages, compositions build, a real `Go` walk from Alderbrook to the summit lair), tile-ready conventions through the existing zone test, `help regions` render and alias test. Merged from [PR #128](https://github.com/Robinsond76/ashveil-gomud/pull/128) after review. **Review (fixed):** (1) relic bosses lairing 7-17 levels under their relics' wear level (36d: worn at the boss's intended level) moved to bands that fit; three lairs keep relic-less bosses (ice guardian, great wolf, bone warden), regression `TestTestWorldLairsMatchTheirRelicLevels`; (2) recipe pages were plain floor items, gone for every other company after one pickup, now `spawninfo` with a one-hour respawn (test spawns them through `Prepare`); (3) `help regions` aliases `zone`, `levels`, `world` hid the admin `zone` page and were too broad, removed with a regression check; the table now fits 80 columns. **Checked, kept:** encounter tables match their bands (levels come from the band; four-foe groups fight at the band's low end, healer groups at most a fifth); in a live web client the new zones draw as tiles with inns, lairs and fog on desktop and phone, the page spawns in the lean-to, and `help regions` renders; `make smoke-world` passes. Noted, not changed: the pilot lairs (Dark Forest ogre, Catacombs lich) still sit under their relics' wear levels, for the replacement world.

**Web client cleanup: GMCP dispatch and shared tooltip/tab helpers (2026-10-07).** Confirmed the dispatch bug from the [cleanup review](plans/2026-10-06-code-cleanup-review.md): handlers registered on a namespace and its parent ran twice (KillStats updated twice per `Char.Kills`, Online twice per `Game`), and `'*'` handlers ran once per namespace level. Now each window is called once per message for its namespace or any parent, then `'*'` once; the redundant map and Online registrations are gone and KillStats ignores `Char.Vitals`. `Client.tooltip` replaces four tooltip copies (Character, Gear, Map, Gametime) and `Client.tabs` three tab switchers (Gear, KillStats, Pet), with no visible change. New `scripts/browser/gmcp-dispatch-check.mjs` (five checks fail on the old dispatch). Not done, with reasons in the review doc: escaping tooltip HTML, one `.ui-tooltip` class, Character/Company tab switchers. Merged from [PR #129](https://github.com/Robinsond76/ashveil-gomud/pull/129) after review. **Review (checked, kept):** every window still receives the same namespaces as before (parent registrations cover the removed map `Party.Vitals` entry; Party and Company get their child messages; KillStats reads only `Char.Kills`; no window registers `'*'`); tab panels all carry the class the helper selects by; in Firefox 142 and Chromium, at desktop and small viewports, the Gear item and Character stat tooltips land at the same pixels, hide after the same 80 ms and cancel hiding on re-entry exactly as on master, and the gear tabs switch the same. The dispatch check now takes `BROWSER=firefox`. Noted, not changed: in Firefox `dock-windows-check` (focus ring, hover-media rules, theme hover contrast) and `mobile-check` (device emulation) fail identically on master; they are harness differences, not this change.

**Final art pass (2026-10-06): every built class, elite route and companion has art.** Drawn with the existing code-generated pipeline (`scripts/sprites/`, `make sprites`), no new colors. 22 classes that fell back to their lineage figure now have map and battle art: Sorcerer and High Sorcerer; the 39i neutral elites (Reaper, Linebreaker, Tempest Lancer, Sword Saint, Shogun, Kenshi, Tempest Lord, Veil Mother, Mountain Speaker, Grand Puppeteer, Golem Lord, String Sovereign); and the Alchemist (Apothecary, Bombardier, Mutagenist), Arbalist (Siegebreaker, Sharpshooter, Warden of the Wall) and Beast Tamer (Houndmaster, Bearward, Dragon Tamer) advanced routes. The still-Planned 39i2 elites are drawn ahead of their mechanics (Panacean, Grenadier, Transmuter, Siege Master, Deadeye, Bastion, Packlord, Beastlord, Dragon Lord, Gryphon Lord, Falcon Marshal, Wyvern Lord) so PR #122 lands with art. Companions: new battle units `war-bear` and `drake-hatchling` (the summon mobs and `BEAST_SPRITES` in `window-battle.js` now name them instead of silhouettes) and new map sprites for the `hound` (own quadruped rig, `scripts/sprites/companions.py`) and `stone-golem` (humanoid rig, `figures.draw_golem`), which had drawn as adventurers on the map. One new accessory, `goggles`. `TestEveryBuiltClassHasArt` is now strict (it no longer needs `ASHVEIL_ART_STRICT=1`; Planned classes are exempt), plus `TestCreatureArtExists`. Decisions (full autonomy): (1) elites are their advanced parent's figure made grander, as in 38c; (2) the wolf and warhound keep the wolf and dog sprites; (3) Direction D image-generated art stays parked (`sprite-art-review` not merged). If a class lands after this without art, the strict test fails: add its entry to `CLASS_ART`/`LINEAGE` in `promoted.py`. Review sheet: `/mnt/project-files/art/final-art-pass-new-art.png`, plus `docs/verification/40s5-contact-sheet.png`. Merged from [PR #125](https://github.com/Robinsond76/ashveil-gomud/pull/125) after review. **Review (checked, kept):** `make sprites` regenerates every committed sprite and contact sheet byte for byte, before and after merging master; with 39i2 merged (#126) no class is Planned and the strict `TestEveryBuiltClassHasArt` passes with the 12 pre-drawn elite keys matching the shipped class ids; in a browser check the 22 new classes draw their own keys on the battle screen, the war bear and drake hatchling replace the silhouettes, and the hound and golem walk as a dog and a stone figure on the map instead of adventurers. Noted, not changed: the golem's chest rune reads a little like a healer's cross at map size.

**Code cleanup pass (2026-10-06):** a survey of the day's Go modules and
the web client for duplication and drift, ranked in the
[cleanup review](plans/2026-10-06-code-cleanup-review.md). Carried out:
one `internal/modconfig` package replaces thirteen copies of the module
config coercion helpers (the copies had drifted on fractional floats and
duration formats); `survival.NeedsLine` replaces a four-way copy. Bugs
fixed with regression tests: making camp and the Camp tab compared room
tags exactly while the inn used `HasTag`; member keys were parsed four
ways and two accepted a bare number as a companion; inn and physician
payments never queued the gold event that refreshes the Worth panel;
purging a leader whose only camping state was a reward cooldown skipped
the save; `Char.Vitals` and `Char.Worth` dropped zero values so the client
showed "undefined / 50" and a dash for 0 gold; the company card had no
branch for a fled member; the gametime countdown could read "1h 60m".
Dead code removed: six Go helpers, two JS functions, WinBox-era CSS and
13 theme tokens read nowhere. Larger refactors (shared module
persistence, one live-companion roster helper, one member-selector
matcher, a shared scheduler, GMCP dispatch firing at every namespace
level, tooltip and tab-switcher consolidation) are proposals in the
review with the reason each waits. `modules/archetype` and loot/items
were left alone for the in-flight 39i2 and 36d builds. Verification:
`make generate`, `make validate`, `go test -race -timeout 30m ./...`,
`make js-lint`. **Review (2026-10-06, merged as PR #123):** every value in
the shipped module configs parses as before (the only fractional numbers,
`StrengthKg`, `MinRatio` and `HeatFactor`, are read as floats; every
duration is a unit string); the six Go helpers, two JS functions,
`.cw-tt-*` and `.vw-max/full/min` rules and 13 theme tokens have no
reader in Go, Lua, templates, JS or concatenated `--t-` names (the window
library only builds `.vw-close`). Accepted, fixed: `modules/encumbrance/AGENTS.md`
still pointed at the removed `company.AddedGrams`. Checked, kept: member
keys like a bare "5" no longer resolve to a companion's display name (no
code writes one). Browser: SP 0, zero gold, and the fled companion line
verified in Chromium at 1280px and 390px; `dock-windows-check.mjs` passes.

**Flaky test cleanup (2026-10-06).** `TestWiringKeenEyeWithoutARogueRarelySpots` failed about 1 run in 100: the archetype and encounters modules' `init()` step listeners stay registered in the test binary and rolled real dice alongside the test's fixed-roll module (with Keen Eye a 1% chance to spot). New `walking.SuspendListeners` sets them aside for tests that wire their own listener (`wired` in archetype, the encounters harness); 1500 repeats pass. `TestWarlordRelentlessQuickensItWhenItsFoeStandsUp` did not reproduce (200 repeats, six shuffled runs, two full company runs all green) and no cause was found, so it is unchanged; a full `go test -race ./...` is green. Not reproduced either: the unnamed company failure on PR #107 (its CI logs could not be read from here). **Review (accepted, fixed):** `SuspendListeners` returned a restore function any caller could forget, and nothing kept game code from calling it; it now takes the test's `testing.TB` and restores the listeners through `tb.Cleanup` (after the test's own listener is removed), with `TestSuspendListenersSilencesAndRestoresThemAfterTheTest`. **Review (checked, kept):** no archetype, encounters or walking test runs in parallel, so the global swap is safe; no other test binary links a module whose `init()` adds a walking listener next to a test listener (modules/walking's chill listener is the one its own test exercises). The PR #107 company log is still unreadable here (the tail holds only `FAIL`; the full log host is blocked).

**Phase 56 built, reviewed and merged from [PR #116](https://github.com/Robinsond76/ashveil-gomud/pull/116) (2026-10-06): recipe discovery.** [Plan and decisions](plans/2026-10-06-phase-56-recipe-discovery.md). `cook [ingredient]...` (and `camp cook [ingredient]...`) at a hearth or the leader's lit campfire tries exactly that mix from the pack and cargo: a match cooks the dish and, the first time, learns it into a per-character recipe book (`recipes`; `internal/cookbook`, kept in the character's MiscData); a miss makes a makeshift meal (item 30062, 25 Hunger, no buff, never bought back) and spends the ingredients; a dish above the cook's rank is refused with nothing spent or learned. Bare `camp cook`, `use hearth` and the cook duty make only learned dishes; Cooking 1 dishes (seared meat, grilled fish) are common knowledge. Recipe pages (items 30063-30064, `recipe:` on a usable item) teach a dish. `look hearth`, the manual cooking capability and the Camp tab (`Company.Camp` gains `recipes`) list what you know. Help: new `help recipes`; `cooking`, `camp`, `camp duties`, tutorial camp hint. Remedies (phase 55) share the book: thyme tea is common knowledge, gut-ache and fever are learned with `camp prepare remedy with [herb]...` (a wrong or unneeded mix spends its herbs). Follow-up: recipe pages need placing in the replacement world. **Review (accepted, fixed):** (1) characters saved before this phase would have lost every non-basic dish and two remedies; `characters.New` now writes an empty book, and a saved character with no book key is `cookbook.Legacy`, knowing every dish and remedy as before (no cutoff date, so online-through-deploy players are covered; `TestALegacyCharacterKeepsEveryDish`, `TestLegacyCharactersKnowEverything`). (2) Any recipe container counted as a hearth: `cook` at Frostfang's tattertail loom made a makeshift meal, and a gated non-Cooking recipe could never be learned; `rooms.Container.IsHearth` (name says hearth, or a Cooking gate) limits `cook`, `use` and `look` filtering to hearths (`TestCookAtALoomIsRefused`, `TestUseALoomNeedsNoRecipeBook`). (3) The right mix for an unknown dish above the cook's rank was refused with "needs cooking 3", confirming it for free; it is now a plain miss, and only a known dish is refused (`TestATooHardDishIsAMissUnlessKnown`). (4) A right remedy mix with two members ill charged two doses, or refused ("Nothing was used") and so confirmed the mix free; a mix is now one dose for the first member it helps, spending exactly the named herbs, and learning reads a cured count instead of matching text (`TestAMixIsOneDoseForOneMember`). (5) A single 3-copper herb made a 25-Hunger meal; herbs-only pots are refused before anything is spent (every dish has game or fish, so this reveals nothing; `TestHerbsAloneAreRefused`). (6) `recipes` and the Camp tab listed only camp dishes; hearth-only learned dishes now show. (7) Filler words ("meat and thyme") matched an arbitrary ingredient; they are skipped. (8) UI: the recipe list sat above the camp buttons and grows with the book; it is now a folded "Recipe book (N)" block under the needs table that stays open across refreshes (dock-windows browser check, desktop and 360px). **Review (checked, kept):** the iron cookpot also doubles a `cook` at a hearth (the player brings the pot; harmless). The Camp tab is rebuilt every refresh, so the book is never stale. The test area armory catalog lists every item spec, so recipe pages 30063-30064 are already offered there (`testarea catalog recipe`), and a trip's restore undoes anything learned. Economy: makeshift meals, cooked meals and pages are never bought back (`IsSpecialForSale`).

**Phase 55 built, reviewed and merged from [PR #111](https://github.com/Robinsond76/ashveil-gomud/pull/111) (2026-10-06): ailments and remedies.** Three sicknesses with a visible cause, one battle penalty and a camp remedy: **Chill** (walking out of doors, or finishing a camp rest, while Frozen, meaning frostbitten or worse; -10% damage dealt, 4 battles), **Gut-ache** (eating raw game meat, which is now edible for 10 Hunger and always gives it; +10% damage taken, 3 battles) and **Fever** (a lasting puncture wound open through 3 battles; -15% damage dealt, 5 battles). They are battle counts on survival `Needs` (like Phase 50's meal buff), add to the Phase 50 condition, fade on their own, and `camp prepare remedy [member|all]` ends them at once from gathered herbs (thyme tea 2 wild thyme; tisane 1 thyme + 1 mushroom; draught 1 glacial mint + 1 thyme), spent as made so nothing resells; `company eat` never serves raw meat. Shown in "Going in:", `status`, `conditions`, `survival`, Company panel, vitals-strip warnings, the Camp tab and `camp prepare status`. Help: new `help ailments`; updated `survival`, `wounds`, `campsupplies`, `cooking`, `conditions`, `combat`, `eat`; tutorial survival hint. Decisions and reasons in `docs/plans/2026-10-06-phase-55-ailments.md`. **Review (accepted, fixed):** (1) the spec says a chill comes from resting or *travelling* Frozen, but expedition travel skipped it; a journey that ends out of doors now checks on arrival, which runs on the event queue after the expedition lock is released (walking arrival listener, `TestAJourneyEndingOutdoorsWhileFrozenCatchesAChill`). (2) A bandaged puncture that stayed partly open kept its fever count, though help says only an *untreated* wound festers; any treatment or tending now starts the count over (`TestTreatingAPunctureStartsItsFeverCountOver`). (3) UI: the Camp tab's Ailing line gains a Make remedies button (`camp prepare remedy all`, hidden while resting; dock-windows browser check, desktop and 360px). (4) `status` Ailing row label was a column wider than the others. (5) `help combat` said an ailment lasts until cured; it also wears off. (6) `TestCriticalHitNarration` flaked because a crit can also be glancing or telling, named first ("(glancing, critical hit, ..."); the test now accepts that. **Review (checked, kept):** ailments fade on their own after 3-5 battles. The spec only asks for one remedy each; a penalty that never lifts would be a trap and hidden difficulty (difficulty comes from zone level), while the remedy stays the fast, certain cure. Economy: raw game meat's value (4) and buyback are unchanged by being edible, and remedies are never items, so there is no profit path. Help claims match code (camp rules, herbs, catch points, inn safety).


**Phase 52 built, reviewed and merged (2026-10-06): tents with trade-offs.** [Plan and decisions](plans/2026-10-06-phase-52-tents.md). Four tents, one pitched at a time, chosen with `camp tent [canvas|fur|camouflaged|large|clear]` (default: the first carried, canvas first, so existing players see no change). Canvas (46) is the plain shelter; the fur-lined tent (300) removes the weather's penalty to a rest instead of halving it; the camouflaged tent (301) halves the raider and thief chance but Rested lasts half as long; the large pavilion tent (302) raises the raider and thief chance by half and gives a camp rest's Well Rested buff (30 minutes) to everyone who slept (wounds and mana still follow a camp rest; duty members still miss it). The pitched tent is locked on the rest, scaled chances use the existing single roll at rest start, and the tent that sets the buff is saved with the pending Rested grant (`Registry.RestedTents`) so breaking camp or a restart before the grant neither drops nor swaps it. Tents are bought, never sold back (`SupplyOnly` at both provisioners), thieves leave them, and their weight (7/9/12/20 kg) counts against load. `camp status`, the rest-start gear line, `look`, the camp-gear view and the Camp tab (`Company.Camp` gains `tent_kind`, `tent_name`, `tent_note`, `tents`; a Tent picker appears under the camp buttons when two or more are carried) show the tent. Help: `help camp gear` Tents section, `camp`, `campwatch`, `inn`; aliases; camp-gear tutorial hint. Spec reading recorded: "no Well Rested" for the camouflaged tent is moot (only inns grant Well Rested to a camp rest), so its cost is a shorter Rested. Tests: model, each effect through a rest, break-camp and reload persistence, failed save, purge, command, GMCP, help, shipped items and shops, dock browser check. Review (accepted, fixed): (1) the build raised the encumbrance sanity guard for every mundane object from 10 to 20 kg; that blunts the typo guard for all objects, so the guard is back at 10 kg with a per-item allowance for the two heavy tents (`heavyObjects`: fur 12 kg, pavilion 20 kg). (2) The tent choice lived on the camp and was lost on `camp break`, so a company carrying two tents re-chose at every camp; it is now a per-leader preference that outlives the camp like auto-sharpen (`Registry.TentChoices`, saved, purged, test-area snapshotted) and can be set before camping (`TestTentChoiceOutlivesTheCamp`); help updated. (3) The Camp tab read "A oiled canvas tent is pitched"; it now picks the article (browser check). Review (checked, no change): no resale profit (tents are `SupplyOnly`, NPC buyback 25% of value is far below minimum prices); every help number matches code (Rested 15m halved to 7m30s, Well Rested 30m, x0.5/x1.5 capped chances); the pavilion's free Well Rested on raid-free roads is a stated, accepted trade (120 gold and 20 kg; no inn wound knitting). Camp tab picker checked in Chromium at desktop and 360px phone.
**Phase 51 reviewed and merged (2026-10-06, PR #103 via the review PR): rest duties at camp.** `camp duties [member|all] [sleep|watch|tend|forage|cook|brew]` sets what each member at the camp does during the next rest; everyone sleeps by default and a player who never sets a duty sees no change (the automatic specialist work is untouched). Duties persist on the camp (`Camp.Duties`), are locked on the rest for the members then at the camp (`RestSession.Duties`) and settled at rest end, so restarts never re-roll. Cost: any duty misses the Rested buff, and a watcher ends no better than Ready (fatigue ceiling 75 in survival's rest recovery). Watchers stack (`WatcherBasePct` 20, own level x 25, independent chances); tenders add a kit treatment or sharpen one member's blades; foragers share the 15 minute forage cooldown; cooks make one dish with their own Cooking; brew is Alchemist-only and served first on short reagents. A spoiled rest settles nothing. Camp tab has a per-member duty picker (`Company.Camp` GMCP `duties`); `camp status` shows each member's duty once any is set. Help: new `help camp duties`; updated `help camp`, `campwatch`, `vigil`, `forage`, `cooking`, `webclient`; tutorial camp hint. Decisions and reasons in `docs/plans/2026-10-06-phase-51-rest-duties.md`. Banter hook left as `CampingModule.onDuties`. **Review (Opus review thread, independent reviewer subagent):** accepted and fixed: (1) the rest's duties were read from the camp's current rest when the Rested grant ran on the next round, so `camp break` (or a new `camp rest`) in between gave duty members Rested and silently dropped tend and cook (or applied the new rest's duties to the old grant); the locked duties now go with the pending grant (`Registry.RestedDuties`, saved, purged and snapshotted with it; `TestBreakingCampBeforeTheGrantKeepsTheRestsDuties`); (2) the company's best forager on the forage duty foraged twice (watch already counted a specialist once); now once (`TestForageSpecialistOnForageDutyForagesOnce`), and help says so; (3) tests added for a watch specialist on watch duty counting once and duties left undone when the company is fighting; (4) help claimed duties persist "until you change them" (breaking camp clears them), didn't say tend and cook are dropped in a fight, and left forage finds out of what duties add; corrected; (5) `camp duties Clear` was case-sensitive; (6) the tend duty's kit treatment now says the kit wears; (7) `camp duties watch` with no member sets the leader's own duty (was the usage text). UI: the builder had not opened the picker; in Chromium the duty block sat above Rest/Break camp and its wrapped buttons ran under the member names on a phone; it now sits under the camp buttons with each name on its own line (dock-windows check gains four 51 checks: rows, Brew only for an Alchemist, the sent command, phone fit, locked while resting). **Economy, decided:** forage finds do sell (raw meat 1, thyme 1, mushroom 5 gold at a shopkeeper's ceil 25%, about 1.7 a find); the duty adds 1 find (more for a ranger) per member once per 15 real minutes and costs that member's Rested, the same gathered-food class the 40a review kept sellable (herbs about 3 a pick, hunting about 6), so kept; cooked dishes are never bought (phase 50). **Rejected or recorded:** duplicate companion names (the picker sends the name, and the watch specialist is matched by name since `archetypes.Specialist` has no companion id) and companions named "me"/"all" can't be told apart; rare, follow-up to send member keys. The "picked over" note can be skipped when an older forage reward is still owed (leader in battle across two rests); cosmetic. A sharpen refused because a member is still flagged aggressive reads oddly inside the tend line; cosmetic. The `tending` alias stays (no clash in the index).

**Phase 39i2 merged (2026-10-06, PR #122 via the review PR): elites for the Beast Tamer, Gryphon Rider, Alchemist and Arbalist.** Twelve elite classes open at level 30 (seven ranks each, 30-60, any alignment; elite talents at 35/45/55; `milestones.eliteShipped` lists the four lineages; `classes.SetPlannedForTest` lets the "planned elite" tests borrow one). Signatures, each fired through a real round in `modules/company/wiring_neutral_elite2_test.go`: **Packlord** (a bite on a hobbled foe leaves it exposed, hobbling reaches foes under 75% health, first strike: 50 meter points at 60), **Beastlord** (swipes also hit the foe beside the target at 50% then 75%, six guards, once a battle the bear stands up at half health), **Dragon Lord** (Breath leaves foes burning, reaches four foes, the first Breath comes in round one), **Gryphon Lord** (Dive cooldown 2, +75%/+100% Dive damage, Thunder landing knocks down the foes beside a downed target), **Falcon Marshal** (a Dive marks its foe for 2 rounds: every ally +5/+8/+12 Attack, through the Warlord mark), **Wyvern Lord** (Dive +25%/+40% against a poisoned foe, a tail lash that hits and poisons a second foe), **Panacean** (Elixir: a blow that would fell an ally leaves it with 25% then 40% health, once then twice a battle, after Ward of Life and Bargain), **Grenadier** (Fire Flask reaches four then five foes), **Transmuter** (a Mutagen also hardens an ally in the patient's row, then the whole row), **Siege Master** (a Piercing Bolt passes through to the foe behind in its column, 1/2/4 times a battle, 60% then 80%), **Deadeye** (no winding after a critical bolt, then after a bolt that fells its foe; bolt cooldown a round shorter), **Bastion** (a loaded crossbow answers a foe that strikes its column, 2/3 times a battle, unloading it). Help: twelve pages (`help packlord` ... `help bastion`), indexed with aliases, linked from `help elite`, the four routes pages, `classes`, `promotion`, `talents` and `progression` (stale "still planned" text removed); tutorial hint in the promotion lesson; `TestNeutralEliteHelpPages` covers them. UI: the battle caption names each signature (Elixir, Pack hunt, Swipe, Thunder landing, Lashing tail, Hair trigger after review; Covering shot, Ballista bolt, Marshal's mark). **Decisions under full autonomy:** Packlord's "second hound" became Pack hunt, the wider hobble and a first strike, because a second beast needs a rework of the single beast record; Panacean's "raise a fallen ally" became a save at the blow (the Ward of Life mechanism), because a mid-battle revive means reversing death processing; Grenadier reaches four then five foes rather than "a row" (Bombardier already reaches three); Siege Master's pass-through counts per battle; elite ranks keep the "values replace" rule of 39i. **Balance** (`TestPhase39i2Elites`, opt-in; even five-foe mirror, Garrick swapped, win%, L40/L60, 30 fights, +/-11 noise, hunting crossbow for the Arbalist): Beast Tamer is at the ceiling (all 96-100%), so only HP lost reads: Houndmaster 27.5/26.5 vs Packlord 21.9/28.8, Bearward 19.1/16.0 vs Beastlord 8.8/5.6, Dragon Tamer 17.0/22.2 vs Dragon Lord 19.8/16.5. Gryphon Knight 86/66 vs Gryphon Lord 76/86 (L40 re-run at 60 fights: 83 vs 86), Skyscout 73/70 vs Falcon Marshal 93/96, Wyvern Rider 76/90 vs Wyvern Lord 80/100. Alchemist sits at the caster floor (6-13%, as the Cleric-like mirror does in 39i): Apothecary 13/6 vs Panacean 33/70; Bombardier 6/6 vs Grenadier 3/13; Mutagenist 10/10 vs Transmuter 3/3 (the Mutagen has no damage to show here). Siegebreaker 66/60 vs Siege Master 66/80, Sharpshooter 83/80 vs Deadeye 73/90 (L40 re-run at 60 fights: 78 vs 81), Warden 56/56 vs Bastion 73/73. Not tuned (timeboxed): Gryphon Lord and Deadeye lead by only about 3 at L40 on the re-run, Grenadier and Transmuter cannot be ranked by a floor sim, Packlord L60 is level with the Houndmaster. **Follow-ups:** elite art for the twelve (art pass; `TestEveryBuiltClassHasArt` strict mode stays opt-in), a single-caster Alchemist measure, Deadeye and Gryphon Lord L40 numbers, a Bastion/Siege Master line on the battle screen beyond the caption. **Review (independent reviewer subagent; accepted, fixed):** (1) Siege Master's Ballista bolt and (2) the Wyvern Lord's Lashing tail struck for 60-80% and 50% of a *plain* blow while the ranks promise a share of the bolt's and the Dive's own damage (about a third of what was promised); both now take their share of the bolt or Dive (`TestSiegeMasterBoltPassesThroughToTheFoeBehind` and `TestWyvernLordTailLashesAndPoisonsTheFoeBeside` check the number through `hooks.RecordExtraBlowsForTest`). (3) Pack hunt counted any hobble (an Arbalist's Crippling Bolt, another company's); it now needs the hound's own (`ClassRT.HobbledBy`, an "another's hobble" case in the Packlord test). (4) The Elixir event had no source, so the web battle caption dropped it; it now names the saved ally. (5) Pack hunt, Swipe, Thunder landing, Lashing tail and Hair trigger emitted no ability event; each now names itself in the caption (tests assert Elixir, Thunder landing and Lashing tail events). (6) The Wyvern Lord's poisoned-foe bonus was not proven (the test passed with the branch deleted); it now compares the Dive's share on a clean and a poisoned foe (+25). (7) Rank and help text: totals that read as additions now say "in all" (Wyvern Lord Attack/Evasion, Bastion health, Deadeye crit and Attack); Covering shot "answers a foe that attacks" (it fires on any swing at its column, as the Halberdier's brace does); First strike "usually" strikes first. UI: the caption fixes above are the change; no new state needs the battle screen. **Balance sim recalibrated:** each swapped member now wields a weapon of its class (Garrick's broadsword is none of theirs): spear (Beast Tamer), war spear for the Gryphon Rider (a lance, so Lance charge and Thunder landing can fire; the short spear never let them), quarterstaff (Alchemist), hunting crossbow; and each lineage meets foes at its own lead so none sits at the ceiling or floor (Beast Tamer +4, Gryphon Rider +2, Alchemist -1, Arbalist 0; sweep -6..+5 recorded in the review thread). Results, 60-160 fights a cell, L40/L60: Houndmaster 66/53 vs Packlord 75/78; Bearward 16/31 vs Beastlord 46/93; Dragon Tamer 91/86 vs Dragon Lord 93/98 (HP lost 50/52 vs 44/36); Gryphon Knight 29/12 vs Gryphon Lord 40/72; Skyscout 24/18 vs Falcon Marshal 72/72; Wyvern Rider 40/23 vs Wyvern Lord 55/68; Apothecary 46/39 vs Panacean 74/90; Bombardier ~50/45 vs Grenadier ~51/45; Mutagenist ~47/47 vs Transmuter ~43/49; Siegebreaker 68/70 vs Siege Master 77/76; Sharpshooter ~77/68 vs Deadeye ~79/87; Warden ~62/59 vs Bastion ~60/60. So Gryphon Lord's thin L40 lead was the sim's short spear (it leads by 11 with a lance). **Rejected after measuring (timeboxed):** a trial buff of Grenadier (hotter flasks), Transmuter (+5 Mutagen armor), Bastion (three/four answers at 100/130%) and Deadeye (Quickened loading at 40) moved no cell beyond noise at 120 fights, so it was reverted: these four signatures act rarely with one member in a five-foe mirror, and they are recorded level with their advanced class, as follow-ups for a sim that exercises flasks, tonics and column attacks. Also recorded, not changed: Falcon Marshal's mark overwrites a stronger live mark (as the Warlord's does); Gryphon Lord's Thunder landing names a knockdown the foe may still resist; Transmuter's spread runs when the patient's own hardening failed; `classes.SetPlannedForTest` lives in a non-test file like the other `*ForTest` helpers.

**Phase 39i merged (2026-10-06, PR #95 via the review PR): elites for Halberdier, Samurai, Shaman and Doll Master, plus a balance pass.** Twelve elite classes open at level 30 (ranks 30-60, seven each; elite talents at 35/45/55 through `offerElite`; `milestones.eliteShipped` lists the four lineages). Signatures, each fired in a real round by `modules/company/wiring_neutral_elite_test.go`: **Reaper** (Sweep also strikes the row behind at 50%, then 75%; Harvest makes Sweep ready every other round), **Linebreaker** (its column takes 10% then 15% less damage; Twin brace answers two foes), **Tempest Lancer** (Charged Sweep bolts arc to the row behind; 1d12/1d14; Stormstruck 35% paralysis, boss 10%, never twice in 3 rounds), **Sword Saint** (Focus to +25%, Iaijutsu +20% crit and +100% damage, Twin draw keeps Iaijutsu for a second strike), **Shogun** (+15 opening meter for every ally, Bodyguard 4, row aura of resolve, Evasion and Attack), **Kenshi** (cannot be knocked down when last standing, +20/25% per fallen ally, Zanshin after every kill), **Tempest Lord** (Rain lasts the whole battle, chain at 100%, Lightning chains through the target's row), **Veil Mother** (weather seven rounds, Fog +10/+12 Evasion, Fog keeps extended reach from the back row), **Mountain Speaker** (Tremor knocks the front row down when Stoneskin lands, Stoneskin +25/+30, Stone cloak covers a row), **Grand Puppeteer** (sturdier dolls, a third doll), **Golem Lord** (golem blows knock down, 170% health, 35% armor, rises a second time), **String Sovereign** (a tangled foe is cut -15/-25 Attack on its next attack, Tangle pushes 70/75, snags three). New elite talents Heartwood and Master Carver (Doll Master). Help: one page per elite (`help reaper` ... `help string-sovereign`), indexed with aliases, linked from `help elite`, the four routes pages, `classes`, `promotion` and `talents`; tutorial hint in the promotion lesson; `TestNeutralEliteHelpPages`. UI: the battle screen header and Combat tab say "the whole battle" for endless weather (`Company.Battle` weather gains `endless`); Stormstruck and Hammer blow name themselves in the battle caption. **Decisions under full autonomy:** split in two (the other four lineages are 39i2 in the roadmap) to keep the PR reviewable; ranks are listed once and rank names do not repeat an earlier rank's; an elite's numbers were set to lead its advanced class by about 10 where the sim can see it. **Existing bug fixed on the way:** a Marionettist's Tangle snagged three foes instead of its two (`1 + TangleFoes`); it now snags `max(1, TangleFoes)` (`TestTangleSnagsAsManyFoesAsItsRouteSays`). **Balance** (`TestPhase39iNeutralElites`, opt-in; even five-foe mirror, two members of the lineage, win%, L40/L60, 100 fights Halberdier, 50 the others, +/-8 noise): Halberdier Sweeper 85/79, Reaper 91/95; Vanguard 77/70, Linebreaker 76/92; Valkyrie 78/68, Tempest Lancer 87/87. Samurai Kensai 88/76, Sword Saint 94/92; Hatamoto 80/74, Shogun 86/88; Ronin 98/94, Kenshi 100/100 (the Ronin is already near the ceiling, so Kenshi's lead shows only in HP lost, 33% against 45%). Doll Master Puppeteer 96/92, Grand Puppeteer 100/100; Golemancer 96/82, Golem Lord 94/100; Marionettist 84/72, String Sovereign 98/98. Retuned after the first 30-fight pass: Shogun and Linebreaker gained +3 Attack at 55 (both sat under their advanced class at L60). **Shaman:** at L40-60 a caster-lineage company loses 90-100% of mirror fights (wizard and witch bases win 10-13% at L40), and even a single swapped Shaman moves outcomes less than the noise (12-32%), so its elites cannot be ranked by this sim. Accepted fix: Gust and Lightning scale with level 1 per 7 and 1 per 6 levels (from 10 and 8), which lifts the late Shaman toward the Wizard without touching level 5-10 much (the old L10 +13 stays recorded, not re-measured), and Tempest Lord (+20% spell damage twice) and Veil Mother (+25% then +45%) carry more spell damage; Mountain Speaker leads Earthspeaker by 10-15 at L60. **Rejected or recorded, not tuned:** Beast Tamer Bearward 9 under base at L15, Druid vs Priest, Nightblade only level with Assassin, Sorcerer trailing siblings, wizard advanced tier at the floor, 38e Golem 17 under the Warrior at L5 (each needs its own sim and none is an elite item; the casters sit at the same mirror floor that hides the Shaman). Art for the twelve elites is pending the 40h pass. Follow-ups: 39i2 (Beast Tamer, Gryphon Rider, Alchemist, Arbalist elites), elite art, Mountain Speaker/Shaman sim with a single-caster measure. **Review (accepted):** a Tempest Lord's Storm wall replaced the chain instead of adding to it, so with its aim alone in a row its Lightning struck one foe and lost Full fork's second (`stormWall` now falls back to one other foe, `TestStormWallStillChainsWhenTheAimIsAloneInItsRow`); Deep tremor said "a boss 15%" where the code gives 35-15 = 20% (text corrected in the rank and `help mountain-speaker`); added `TestNeutralElitesPromoteThroughTheClassCommand` (the promotion line names `class promote <elite>` and it works for all four lineages at any alignment). **Review (checked, no change):** every rank's text matches its value (values replace, so "+4 Attack" over an advanced +2 is 6); each signature fires in the wiring tests; ability events name themselves in the caption. Shaman at L10, 2x60 fights: Shaman 51/63%, Wizard 55/55%, so it sits with the Wizard, not over it; the leveldiv lift changes nothing below level 12 (integer division), so L10 is unaffected by 39i. The elite sim gives Ysolde her poacher's sling (10014) whatever her class; elite and advanced cells share the kit, so the gaps hold, but absolute win% for a melee lineage is understated. Witch L10 read 36% on this branch against 44% on master (4x60 fights each, about 2 standard errors apart); no 39i code path is one a Witch company uses, so it is recorded as noise, with a follow-up to re-measure the casters at 100+ fights.

# Ashveil Project Status

**Web client: side panels keep their scroll; Firefox no longer offers saved logins on the arrow keys (2026-10-06, [PR #110](https://github.com/Robinsond76/ashveil-gomud/pull/110)).** Every panel rebuild (GMCP update) emptied its content and could snap scrolled boxes back to the top; `keepScroll` (`static/js/keep-scroll.js`, tested under `make js-test`) restores the scroll position of the panel and its ancestors after each rebuild in the Character, Company, Gear, Combat, Room, Pet, Online, Tutorial and Kill-stats windows. Firefox: the command box was switched to `type="password"` for the login prompt, and Firefox keeps treating any box that was ever a password field as a login field (it ignores `autocomplete="off"` there), so the arrow keys opened its saved-logins ("Manage passwords") dropdown all session. Unmasking now swaps in a fresh text box (`setTextMask` in `webclient-core.js`); the box also carries `autocomplete="off"` and password-manager ignore hints, and the quick menu takes focus off the box while open.

**PR #110 review (Opus review thread):** accepted and fixed: (1) the build's Firefox fix only blurred the box while the quick menu was open and leaned on `autocomplete="off"`, which Firefox ignores for a former password field, so the dropdown could still appear on walking arrows and history; the fresh-box swap fixes the cause (`scripts/browser/quickmenu-check.mjs`: after TEXTMASK true/false the box is a new empty text input that keeps focus, sends and opens the menu). (2) The Company Inventory tab still jumped to the top: the `Company` payload lands before `Company.Inventory`, so the tab briefly renders "Nothing yet", the restore is clamped, and the full rebuild then started from 0; `keepScroll` now remembers a clamped position until the player scrolls (`keep-scroll.test.mjs`). (3) The browser-check harnesses did not load `keep-scroll.js`, so `dock-windows-check` and the other harness checks threw "keepScroll is not defined"; all six harnesses load it now and every check passes. Verified on a live server with a recruited company in Chromium, forcing a layout after each panel is emptied (the reset Robinson sees): on master Status, Inventory, Skills and Gear all reset to 0; with the fix all hold. Not run in a real Firefox (no Firefox reachable from the container); the cause is from Firefox's password-field handling, so Robinson should confirm. Rejected: none. Noted, not fixed: after Alt+Up recalls history and the box is cleared by hand, the arrows keep walking history instead of the map until a command is sent (pre-existing).
**Click to move in the formation drawing (2026-10-06, Robinson's web client note).** In Company > Status the formation cells are buttons: click or tap a member, then an empty cell to move them there or another member to swap (Escape cancels; keyboard works with Tab and Enter). They send the existing `formation move` / `formation swap` commands, so the server's rules stand; in a battle, for a fallen member, or for a solo leader the panel refuses locally with a message and sends nothing, and members not yet placed appear under "Not placed". Help (`formation`, `webclient`) and the tutorial formation hint updated; checked in `scripts/browser/dock-windows-check.mjs`. Merged from [PR #107](https://github.com/Robinsond76/ashveil-gomud/pull/107) after review.

**Click to move review (Opus review thread):** checked end to end on a live server in Chromium at 1280 and 360 wide (a recruited company: a move and a swap both update the drawing from the server). Accepted and fixed: (1) picking an unplaced member and then a placed one (or the reverse) sent `formation swap`, which the server refuses for an unplaced member; the panel now asks for an empty cell, or picks up the unplaced member instead; (2) Escape only cancelled with focus inside the table, not from the Not placed row; (3) phone cells were 29px tall, now 44px targets; (4) the hint said "Esc cancels" on phones; it now says to pick the member again. Regression checks in `dock-windows-check.mjs`. Rejected: none.

**Phase 53 built, reviewed and merged from [PR #104](https://github.com/Robinsond76/ashveil-gomud/pull/104) (2026-10-06): defeat scenarios.** A fallen leader now wakes in a situation that fits what beat the company instead of at the church a level down: rescued (nearest settlement, every member Hungry and Exhausted), captured (by people: wakes bound in the Brigands' Tent with the pack and most of the gold in a guarded chest, `reclaim` once the two guards fall), left for dead (by beasts and the undead: foes gone, a lasting wound on the leader and each companion), robbed (by people: foes gone, a share of gold and loose goods gone for good). No level is lost in a scenario; the church and its level loss stay as the fallback when no row fits. Scenarios are YAML (`Scenarios` in the death module overlay, keyed by killer race or group and zone), claimed through a new `ScenarioProvider` seam before the engine's drops and corpse, and saved on the character with the death's pending mark so a restart resumes the same scenario and never re-rolls. Held goods ride the user file (`Character.Seized`), so no item is lost or doubled. New: buff 9301 (Bound), rooms 91001-91002 (Brigand Camp, linked north of the Fork at the Black Oak, test content), `help defeat`, Inventory tab "Held by your captors" notice with a Reclaim button. [Plan and decisions](plans/2026-10-06-phase-53-defeat-scenarios.md). Tests: `modules/death/wiring_defeat_test.go` (each kind through the real `suicide`, resume without rerolling, reclaim, no duplication), `internal/death/scenario_test.go`, `TestDefeatHelp`, `TestShippedDefeatScenarios`, gmcp held-goods payload. Follow-ups: a Hardcore option that routes to the church only; per-zone scenario rows for the replacement world.

**Phase 53 review (Opus review thread):** kept the build's decisions. Decision: robbery and capture gold comes from on-hand gold, which is the company treasury; banked gold is never touched (help says so). The spec's "treasury safe" can't hold beside "some gold gone", and the bank is the safe pool. Decision: guards are restored rather than saved with the room: the character counts undefeated guards and they stand again on return or reclaim. Accepted and fixed: (1) capture guards were lost to a restart or an unloaded room and the chest then opened without a fight (`TestDefeatScenarioGuardsReturnUntilDefeated`); (2) foes sent away were despawned without a cooldown and respawned the next round beside the wounded company; they now keep their respawn timer, and other hostiles in the room go too (`TestDefeatScenarioSentAwayFoesStayAwayForTheirRespawnTime`); (3) guards could flee or yield and so never count as beaten; they now never break; (4) protected deaths (ProtectionLevels 5, perma-gear) could be captured or robbed though they never lost goods before (`TestDefeatProtectedDeathIsNeverCapturedOrRobbed`); (5) reclaim in the round guards fell could post fresh guards or open early; reclaim now waits on the counted guards; (6) guard tracking checks the capture group and prunes vanished guards; (7) a stale claim is cleared when its row leaves the table; (8) the wake text says fallen companions can still be raised (`TestDefeatScenarioKeepsFallenCompanionsRaisable`, with a real companion death); (9) web client no longer shows "0 items and N gold". Help `defeat` updated (guards return until beaten, bank safe, level 5 and below never captured or robbed, hostiles cleared). UI: the Inventory tab notice and Reclaim button checked in Chromium at 1280 and 360 wide (`scripts/browser/defeat-check.mjs`; screenshots `screens/53-*-here.png` in project files); no other change needed. Rejected: none. Recorded as decisions instead: claim on the leader's death (that ends the fight here), and the held pack is the company cargo (one item list).

**Phase 50 built, reviewed and merged from [PR #100](https://github.com/Robinsond76/ashveil-gomud/pull/100) (2026-10-06): condition carries into battle; meal buffs.** [Plan and decisions](plans/2026-10-06-phase-50-battle-condition.md). As a battle begins, each member's needs set its battle condition for that battle: hunger cuts the damage it deals (Hungry -5%, Starving -10%), thirst raises the damage it takes (Thirsty +5%, Parched/Dehydrated +10%), fatigue keeps its existing hit cut (-5/-10/-20). Cooked meals (`meal:` on the item: seared game meat Strong +5% damage for 2 battles, thyme-roasted game Steady +5% damage and 5% less taken for 3, hunter's stew Hearty 10% less taken for 3, grilled fish Clear-headed 20% max mana back as each battle begins for 3) give a buff counted in battles, one at a time; it is saved on the member's survival needs (survives restart, test-area snapshots). A "Going in:" line opens the fight; `survival`, `status`, `conditions`, Company.Vitals `fare` (Company panel), Company.Battle `fare` (battle screen banner and caption, Combat tab) show it. A well-kept, unfed company is untouched. Help: `survival`, `cooking`, `conditions`, `eat`, `company meal`, `combat` hub; survival-lesson tutorial hint. Decision: thirst works on damage taken, not tempo (a 5% tempo cut made the whole company lose the same early turn: 50% to 12% wins). Balance (`TestPhase50BattleCondition`, opt-in; whole company in one condition, a Wizard in slot 3, five even foes, 40 fights a cell, about ±11 points of noise), wins at L5/L12/L20: fed (the unchanged baseline) 45/37/30; hungry 20/40/22, thirsty 25/45/20, tired 27/32/12; all three small 22/20/7; starving+parched 0/15/15; plus exhausted 2/5/5; meals Strong 47/52/32, Steady 42/65/30, Hearty 50/75/55, Clear-headed 42/55/17. Settled (timeboxed): penalties bite and stack, meals help modestly; Hearty is the strongest (the Cooking 3 dish) and is kept. Tests: `internal/survival`, `internal/combat` (strike loop), `modules/survival` (restart), `modules/company` `wiring_fare_test.go`, `modules/gmcp`, help and eat tests, root `TestShippedMealBuffs`, `scripts/browser/fare-check.mjs`. Follow-up: tents and rest duties (51, 52) can add to the condition; ailments (55) reuse the battle-start hook.

**Phase 50 review (Opus review thread):** kept the build's decisions. Decision: thirst raising damage taken (instead of the plan's tempo cut) is accepted; the tempo meter turns a small cut into a lost turn for the whole company, while damage taken scales smoothly and the help says so. Checked and fine: conditions are set per member from that member's needs and cleared with the class state at fight end, so nothing carries between battles; a meal buff counts down once per battle begun, a new meal replaces the old one (no stacking), eating is refused mid-battle, and the buff persists on the survival needs; Clear-headed's mana comes back once per battle begun and each costs a battle, so at most 60% per fish; heals never read `SpellFactor`; no game time advances. Balance re-run at 80 fights a cell (five even foes): fed 48/23% at L12/L20, Hearty 62/45%, Steady 72% at L12. Meals help by roughly one band in an even fight, which suits preparation for a harder zone (the difficulty rule is unchanged: a fed, unbuffed company fights exactly as before), so no tuning. Accepted and fixed: (1) economy: market meat and thyme at their lowest prices (2+2+2) cost less than a merchant paid for a hunter's stew (up to 8), so cooked meals are never bought back (`IsSpecialForSale`, `TestCookedMealsAreNeverBoughtBack`); (2) `company meal` picked food by nutrition alone, so it could swap a member's Hearty for a Strong slab or spend cooked meals as plain calories; a member on a buff now eats plain food when there is any (`TestPlanMealKeepsAMealBuff`); (3) help: `survival`, `status`, `conditions` and the Company panel show the next battle's condition while the battle screen and Combat tab show the current one (the meal count differs by one mid-fight); `help survival` now says so, and `help cooking` says a battle retreated from at once still counts; (4) coverage: `TestSpellFactorCarriesTheBattleCondition` checks the condition through the factor spell scripts read. UI: the "Going in:" line, Company panel, battle banner and caption, and Combat tab all render the same summary; no further change needed. Rejected: none.

**Keyboard play: arrow walking and the quick menu (2026-10-06, reviewed and merged from [PR #99](https://github.com/Robinsond76/ashveil-gomud/pull/99)).** With the web client's command box empty the arrow keys walk (Up north, Down south, Left west, Right east; Alt+Up/Down start the command history, which plain Up/Down used to). Enter on an empty box opens a quick menu over the game (`static/js/quickmenu.js`): Battle (Retreat, company focus, only in a battle), Look, Attack, Move (exits, map places, scout), Services (shops, trainer, bank, inn), Get, Gather (what the room offers and has not picked clean), Company (status, inventory, meal, patch, camp, tactics), Me, Help. Entries open submenus; every level ends with Back (Close at the first); Esc, Backspace and Left go back; 1-9 choose; click works; the phone touch bar's Menu button replaces its Status button. The menu is built from live GMCP (`Room.Info`, `Company.Camp`, `Company.Battle`, the map's places) and sends ordinary commands by id. Help `quickmenu` (aliases arrow keys, keyboard), linked from `webclient` and `mobile`, with a Departure tutorial hint. Tests: `scripts/js/quickmenu.test.mjs`, `scripts/browser/quickmenu-check.mjs` (and a phone case in `mobile-check.mjs`), `TestQuickMenuHelp`. **Decisions:** arrows walk only with an empty box (typing keeps them), Enter on an empty box no longer sends a blank line, no independent review per the speed rules (the review thread covers it).

**Quick menu review (Opus review thread):** accepted and fixed: (1) Attack listed every NPC, so your own companions (`charmed`), shopkeepers, and the downed or surrendered were offered as targets; it now lists foes only and hides itself when none stand. (2) During a battle the menu still offered Move, Services, Get, Gather and the company's meal, patch, camp and scout, which the server refuses mid-battle (`go` answers only-flee); a battle now keeps Battle (Retreat, focus), Look, Company (status, inventory), Me and Help, matching the rule that retreat and focus are the only battle inputs. (3) Enter on an empty box opened the menu even while a prompt question waited (76 `prompt.Ask` sites, many with a default a blank Enter takes, such as a yes/no); worse, choosing an entry would have answered the question with that command. The client now notes a pending question (the prompt line starting `.:`) and the login screen (no `Room.Info`), and in either case Enter sends a blank line and the arrows don't walk (`Client.Playing()`). Help `quickmenu` says so and names Shift+Enter for a blank line any time. Tests: `quickmenu.test.mjs` foe and battle cases, `quickmenu-check.mjs` question and shopkeeper cases. Judged and kept: Alt+Up/Down for history (plain Up/Down still continue once history shows, and with text typed); losing the blank Enter outside questions (it only redraws the prompt; Look is in the menu and Shift+Enter still sends one); the phone bar's Menu replacing Status (Status is under Menu > Me, and the bar fits 360 px). Rejected: none.

**Phase 38e complete, reviewed and merged from [PR #86](https://github.com/Robinsond76/ashveil-gomud/pull/86) (2026-10-06): creature recruits, a Hound and a Stone Golem.** [Plan and decisions](plans/2026-10-06-phase-38e-creature-recruits.md). A creature is a species with base ranks but no promotion or talents (`internal/creatures` family rules, `internal/classes/routes_creature.go`). **Hound** (biological, 90 gold at the Trappers' Post): eats, drinks, tires, mends and can lose heart like anyone; ranks Run down (+20% vs an exposed, downed or hobbled foe), Worry, Fleet, Savage pursuit. **Stone Golem** (construct, 180 gold at the Waymark Inn): needs no food, drink, rest or bed, never drifts or deserts ("bound"), no idle or camp recovery; `company repair [golem]` spends stone mortar (item 290, market supply only) at 50% of max health each; ranks Stone body (+30 armor, 15% fewer turns, spells hit it 25% harder), Anchor, Granite, Bedrock. Creatures carry nothing, wear only gear cut for their species (new item `wornby`; Hound collar 20026, harness 20500), can't be chosen or re-archetyped by a player. New effects `pounce`, `slow`, `spellweak`. Mobs 260 and 261, item 290 and 20500. Help: `creatures`, `hound`, `stone-golem`, `repair`, tutorial hint, code-drawn battle art. Build balance (100 fights a cell, third slot replaced, five-foe mirror, wins vs warrior 75/89/77% at L5/10/20): Hound 92/98/86%, Golem 81/93/63%; retuned in review (below). **Decisions taken under full autonomy:** constructs are bound rather than loyal; golems get no idle regen and are repaired with mortar; healing spells still heal them; creatures carry nothing; creature gear is item-level `wornby`; 38e built without waiting on 38d and 39e. **Follow-ups:** Warhound and Runic Golem forms, creature gear catalogue, map sprites for the species, Beast Tamer (39e) to reuse `internal/creatures`, a GMCP `family` field.

**Phase 38e review (Opus review thread):** the sim was checked first: each fight is its own subtest with a fresh company, so the swapped member really is a Hound or Golem with its own health, ranks and body (a Hound has no weapon, so there is no kit artifact like the 39h sling). Accepted and fixed: (1) *Balance.* The Hound's low health made it the "weakest" target the mirror's foes chase, and its high Evasion made them miss, so it tanked and bit; Attack rate was the steepest lever (0.65 to 0.8 moved L20 from 55% to 93%). Tuned: Hound HP head start 3 to 0, Attack 0.75 to 0.68, Run down 30 to 20%, Savage pursuit 50 to 35%, Worry 25 to 15%, Fleet +3 to +2 Evasion; Golem's fist gains +2 damage at Granite (10) and +4 at Bedrock (20), since its 2d5 fell behind a warrior's blade. Final (100 fights a cell, L5/10/20): Warrior 88/92/64, Hound 82/92/68, Golem 71/90/70; cells swing about 8 points between runs at this size, so all three now sit within that of the warrior (timeboxed after seven rounds). (2) `help beast` (the Beast Tamer's page, merged after this branch) lost its alias to `help creatures`; the creature aliases no longer claim "beast". (3) Admin test area: `testarea class hound` made the admin a hound, and `companion class` could turn a person into a hound (keeping its human body) or a hound into a warrior (keeping its bites). Both now refuse; `companion add hound|stone-golem` recruits one and the supplies kit carries stone mortar (`TestCreaturesInTheArea`, which also checks the return restores everything). (4) Web gear views offered a hound every slot, sword included, and offered creature gear to people (the server refused, but the tap menu and drag targets lied). `Company.Inventory` members now carry `closed` slots (the race's disabled ones) and `species`, items carry `worn_by`; the Company > Inventory slots, picks and drops follow them, and the Gear editor skips closed slots (`TestCreatureGearViewsOfferOnlyItsSlotsAndGear`, dock-windows browser check). (5) Camp and victory banter let a hound or golem speak; creatures no longer talk (`TestCreaturesDoNotTalk`). (6) The independent diff review found more: a recruited golem still took lasting wounds (the companion rule in `combat.MobWounds` ran before the template's `wounds: none`), so its health limit dropped and `company repair` called it "whole" below maximum; constructs now take no wounds (`TestStoneGolemTakesNoWounds`). A creature counted as a walker who needs a riding horse while the herd cap (from carrying members) gave it none, so a company with one could never ride at pace; creatures now need no horse (`TestCreaturesNeedNoRidingHorse`). The admin tool let a hound become a golem; species are now fixed. `help repair` gave the wrong mortar order. A tuning slip in this review that set the Alchemist's Attack rate to 0.65 was caught and reverted. Rejected or recorded: the spiked collar (20026) now worn only by hounds stays on anyone already wearing it (only weapons are swept on load; one old item, accepted); camp actions skip the golem when checking for a fight in progress (a lone golem fight is not reachable, since the company fights together); `help golem` stays with the Doll Master's golems. Checked and fine: mortar is supply-only at the market (9-28 gold) and a shopkeeper pays at most 25% of its value 10, so no resale profit; hiring and dismissing returns only gear the leader gave (the collar and harness are template gear, worth 0, and salvage yields far less than the 90 gold hire), so no farming loop; the companion-equipment equip and dismiss paths run through `CanWield`, which enforces `wornby`; the admin test area snapshot restores creature recruits. Rejected: none. Follow-ups: map sprites for both species (art pass), Warhound and Runic Golem forms, a creature gear catalogue, fall banter that suits a creature. Gates after merging master: generate, validate, js-lint, js-test, the dock-windows, dock, mobile and battle browser checks, and the full race suite.

**Phase 57 built (2026-10-06): the Character panel tidy-up.** Robinson's web client notes, each decided under full autonomy. (1) *Skills:* the capability text had no font size (it inherited the dock's full size) and trained ranks were centred native buttons. Skills now uses cards in Company's type scale (0.8em, `.cmp-block`'s border and fill): a skill shows its display name, `2/4` and pips, MAX, and what it does (`Char.Skills` gains `title` and `description`, additive, from the skill data); each combat, field and camp capability is a card with a Ready/Eligible or reason chip, its description, and a quiet line for skill, trigger and cooldown; companions' training is a card per member. (2) *Jobs removed from the panel.* Jobs are stock GoMud profession titles derived from skill points (`Cook`, `Warrior`, a percentage bar each); in Ashveil a class and its route name a character and the Skills tab already shows the ranks, so the bars were a second, unexplained list. The `jobs` command, its help page and `Char.Jobs` stay for text clients (the help page now says the web client does not show them). (3) *Gear:* the editor was unstyled native controls at full size; it now uses Company's controls (chips for members, slots and choices, a compact Current → After table with changed values in the accent), and hides the single-tab bar. (4) *Overview:* name on its own line with class, lineage and race beneath, level and alignment on one row, promotion status on its own line, readable stat and point labels, stat cells and race reachable by keyboard, Worth block flush with the rest. (5) *Hover:* two real causes. The shared hover highlight was white text on the theme's mid accent, which is 2.2:1 in the Aurora theme (and low in Grayscale), with quieter child text left unreadable; it is now the theme's own hover fill and text with an accent edge, checked at 3:1 in every theme. And hover rules applied on touch, so a tap on a control that opens a help screen left it highlighted: every hover highlight is now limited to hover-capable pointers, the menu (`uiMenu`) keeps one highlighted entry (the pointer moves focus; no inline state is left behind), and the help screen takes focus and the pointer from what opened it and returns focus on close. I could not reproduce the exact report on a desktop pointer (`make run` was not available in the build session), so the fixes cover every mechanism found: tell me if a case still sticks. Help: `webclient`, `skills` and `jobs` pages updated, tutorial web-client hint points at the clickable stats and skills. Tests: `dock-windows-check.mjs` (skills content, hover media gate, one highlighted menu entry, focus return, theme contrast, 360px fit), `TestCharacterPanelHelp`, the gmcp `Char.Skills` title and description. Screenshots: `screens/57-before-*` and `57-after-*` (desktop, 360 and 390 phone).

**Phase 57 review (Opus review thread, PR #91):** drove the real client against a live server (master and the branch side by side) with a desktop mouse in the Brooding (default), Aurora and Parchment themes, clicking the race, stats, skills, inventory rows, menu entries and map, and closing each help screen by Esc, ×, and a click outside. Accepted: (1) the help screen returned focus to the control that opened it even after a mouse click, so closing with Esc lit that control's focus ring, and the ring stayed (still on the race while the pointer hovered a stat) until the next click: the likeliest match for "the highlight doesn't go away". Focus now goes back to the control only when the keyboard opened the screen; after a mouse click it returns to the command line (touch screens skip that so no keyboard pops up). (2) A long capability status ("Missing recipe ingredients or trained ranks") squeezed the Camp Cooking card's name to one letter a line; the card head now wraps the badges below the name. Both have regressions in `dock-windows-check.mjs` that fail on the build's code. Checked and fine: `Char.Skills` `title` and `description` are `omitempty` and skip unknown skills, so other clients are unaffected; hover fills read in all three themes; the menu keeps one highlight. Not reproduced: a hover that "highlights everything" on a desktop pointer, on master or the branch; if it still happens, the place it happens is what's needed. Noted, not fixed (pre-existing, out of scope): the Character panel keeps a new character's placeholder name until the next `Char.Info`.

**Phase 39h built (2026-10-06): the Arbalist neutral lineage, the last of the eight.** An anti-armor crossbow shooter: Piercing Bolt at level 1 (the whole turn, 140% of a shot, ignores half the foe's armor, cooldown 2), then its next turn is spent winding the crossbow (`ClassRT.Reload`, the "winds the crossbow" line and a "Winding the crossbow" ability event), so it shoots every other turn. Ranks: Steady Aim 3 (+10 Attack on a bolt when no blow has landed on the Arbalist since its last), Crippling Bolt 6 (a landed bolt hobbles 2 rounds), Armor-breaker 8 (all the armor). Routes (open at any alignment): Siegebreaker (each landed bolt takes 10 armor off the foe for the battle up to 30, heavier bolts, 15 a bolt up to 45), Sharpshooter (+15% critical chance with a shooting weapon and crits can't be blocked, Steady Aim +20, the first bolt of a battle needs no winding, +4 Attack), Warden of the Wall (Pavise: allies in its row take 10% less damage, +4 armor, 15%, +8% health); elites planned (39i). New `armored` target rule (most armor reachable; the Arbalist's default; aliases `armor`, `tank`), Heavy Bolts talent, skill `arbalestry`, recruit mob 235, base map and battle art, `help arbalist` and `help arbalist-routes` plus the fourteen pages the class touches, creation-lesson tutorial hint, level-up report line for the bolt's size. Plan: [39h](plans/2026-10-06-phase-39h-arbalist.md). Decisions (autonomy): the bolt is the turn and reuses `extraBlow` (as Dive does); reload belongs to the bolt, so a crossbow in anyone else's hands stays a heavier bow (36b stands); Steady Aim is "no blow has landed on it"; armor shred lives on the foe's class state and floors at no armor; Pavise reuses the row aura instead of a ranged-only shield the engine has nothing to test against; no Utility (the shipped ones read skills the class lacks); fighter role with the `armored` rule. Balance at 100 fights a cell, levels 5/10/20: Warrior 74/90/75, bow Ranger 49/79/53 (a sword Ranger was 77/89/69 in 39f), Arbalist 89/90/80; tuned the bolt 160 to 140% and Keen sight 10 to 15%, Pavise 8 to 10%; the Warden reads level with base because the sim swaps one member (recorded, timeboxed). Follow-ups: elite ranks and Siege Master's pass-through bolt (39i), Arbalist art refinement, a bolt-reload indicator on the battle screen beyond the ability event.

**Admin test area built (2026-10-06): `testarea`.** An admin-only command that snapshots the whole character and company and moves them into a closed test zone (rooms 90001-90010: hub with help sign, combat yard, camp, thieves'-road camp, armory, stable, weather rooms, a dark cellar, a thicket and a narrow pass), then `testarea return` restores everything exactly and nothing earned there persists. The snapshot is the full user record plus every module's per-user state, collected through the new `internal/userstate` Contributor registry (archetype, camping, company incl. companions, horses and cargo, death, encounters, encumbrance, exposure, expedition, mount, strategy, survival, walking), is saved through the plugin store, and survives disconnect and restart. The area has no exits to the world; recall, walkto and any other exit are blocked. Tools (only on a trip): `class` (any class incl. elite and neutral), `level`, `companion add|class|level`, `fight [level] [size] [type]`, `clear`, `heal`, `gold`, `give`, `kit` (weapons, armor, camp gear, supplies, saddles), `items`, `weather`. Help: `help testarea` (admin). Tests: `modules/testarea` (a trip in and back leaves the saved user file byte-identical and every module's state equal, including a mid-camp-rest trip, a restart while away, a recruited companion and a horse bought in the area), `internal/userstate`, `TestEveryModuleWithUserStateIsSnapshotted`. Grant admin with `modify role [name] admin` or `role: admin` in the user file. Decisions (owner delegated): (1) one command with subcommands; (2) the thieves'-road "toggle" is two camp rooms, because camp theft and raids are keyed by zone; (3) no day/night control, since the shared clock is never touched: rooms carry the `lit` tag and only the cellar is dark; (4) fight groups are 2-6 foes and there is no allied-company option (not cheap); (5) entry is refused in battle, while resting at camp or in a party; return works from anywhere; (6) setting a class resets the spell book, a companion's class change keeps its gear; (7) permission key `testarea` for moderators; (8) no tutorial pointer (admin-only). Follow-up: tutorial and `modules` authors must register a Contributor for any new per-user state (enforced by the purge-coverage test).

**Admin test area review (Opus review thread, PR #83):** verified a trip in and back restores the user file byte for byte and every module's state, including companion gear, a dismissal, horses and cargo, a restart away, and a mid-rest camp. The "party" refusal only blocks GoMud player parties, never a solo company. Accepted and fixed: (1) a player typing `testarea`, or any admin command, was told "You don't have permission to use ...", and `help testarea` showed the page to everyone. Now an admin command a player may not use falls through as an unknown word. The player-facing `help/testarea` page is gone, and `help [admin command]` renders the command's admin page only for those allowed to run it, which also gives admins `help zap` and the like. (2) A death in the area sent the admin to a church in the world carrying test gear. `internal/death` gains a wake override, so the area's dead wake in the hub. (3) Leaving the area by any other way (an admin teleport, a script) left the test character loose in the world. Any move out of the area now ends the trip and restores at once. Rejected: the claim that recall and walkto are blocked needs no code, because the rooms have no exits. Tests: `TestCompanionGearDismissalAndCargoAreUndone`, `TestLeavingTheAreaEndsTheTrip`, `TestDeathInTheAreaWakesInTheHub`, plus the no-hint checks in `TestOnlyAdminsGoInAndToolsNeedATrip`. Companion equipment (PR #84, merged in) is covered generically because it lives in the company record the snapshot captures. The test equips, removes and re-equips an armory weapon on a companion, then dismisses it, which hands the gear to the leader, and the return undoes all of it. Gates after merging master: generate, validate, js-lint and the race suite pass, except `TestAnApothecaryDraughtHealsMoreSplashesAndCleans` in `modules/company`, a 39g test that fails about 1 run in 40 from heal rolls (it passed on rerun and does not touch this phase); after the companion-equipment merge, `TestWarlordRelentlessQuickensItWhenItsFoeStandsUp` failed once under the full company race run and passed 30 of 30 alone (this phase adds no combat code). Follow-ups: make that test deterministic; a test-area-only day/night override (skipped, since lighting is already testable through the lit rooms and the dark cellar); an allied company in the combat yard.

**Test area armory catalog (2026-10-06, Robinson's ask):** `testarea catalog [word]` replaces dumping a whole kit: the web client opens an armory screen (`window-armory.js`, inside the help modal) listing every item with a search box, type chips and a Take button with a count (sends `testarea give <id> <n>`); other clients get a text list (types with counts, or matches for a word). Server side the catalog goes out as the `Armory` GMCP message and gives nothing itself. The search box is `autocomplete=off` so Firefox does not offer saved passwords. Anything dropped in a test room vanishes at once (`ItemOwnership` hook), and each round sweeps floor items and gold from all ten rooms, so nothing can feed the economy; the area stays admin-only and a trip's return still restores the character. Help (`help testarea`) and the armory room text updated. Tests: `TestArmoryCatalogListsEverythingAndGivesNothing`, `TestDroppedItemsAndGoldVanishInTheArea`, `scripts/js/armory.test.mjs`, and an armory block in the dock-windows browser check. Review (Opus review thread, PR #109, checked live in Chromium at desktop and 360px): accepted and fixed: (1) on a phone the 27 type chips filled the screen and the item names wrapped one letter a line beside their nowrap details; the chips are now one scrolling row under 600px and each row stacks the name over its details; (2) opening the screen or tapping a chip focused the search box, which raised the phone keyboard over the list; touch devices now wait for a tap on the box; (3) from the armory itself the catalog was reachable only by typing `testarea catalog`; walking into the armory, or `testarea armory`, now opens it on the web client (`onEnterArmory`; covered in `TestArmoryCatalogListsEverythingAndGivesNothing`), and help and the room text say so. Checked and kept: the round sweep only loops 90001-90010 and the drop hook only acts for a user standing in one of them; battle spoils go straight to the company (`loot.TakeSpoils`), so sweeping floors does not stop loot tests; corpses are untouched; the catalog and give stay behind the trip, which only admins (or the `testarea` permission) can start.

**Phase 49 complete, reviewed from [PR #82](https://github.com/Robinsond76/ashveil-gomud/pull/82) (2026-10-06): companion banter.** Companions now talk like comrades. `internal/banter` is a pure engine over 374 authored lines (embedded YAML in `internal/banter/data/`: by personality, archetype or family, alignment bucket, call-and-response pairs, mentions of another member or the player, and untagged fillers). A line has required tags (`arch`, `pers`, `align`), optional `prefer` tags that weight it, a `ctx` (camp, rested, win, close, fall, flawless) and an optional `reply` naming the prompt it answers; `{other}`, `{fallen}` and `{leader}` fill in names. `Exchange` picks 2 or 3 members and builds 2-4 lines (an opener, an answer or a line of their own, a possible third voice, a possible last word); a line is not repeated for the same member within its last 12. Each companion rolls a personality (stoic, cheerful, grim, boastful, wry, devout) when it joins (`Companion.Personality`, saved once; older companions derive a stable one from their ID). Camp talk: always as a rest begins (`CtxCamp`), about half the time as it ends (`CtxRested`), both appended to the rest's own text; after a *won* battle about 1 in 3, never mid-battle, said after the patch-up and tied to the outcome (a companion newly dead since the last battle: fall; anyone at or under 30% health: close call; everyone at or over 90%: flawless; else a plain win; each falls back to win lines). Chances are config (`BanterCampStartPercent`, `BanterRestedPercent`, `BanterBattlePercent`). `set banter on|off` (bare toggles; listed by `set`) is a per-player option, on by default; the web client's Camp tab shows the latest exchange under "Around the fire" (`Company.Camp.banter`). Help: `help banter` (aliases camp-talk, chatter, companion-talk, comrades), linked from `help camp`, `help combat`, `help set` and `help webclient`; the tutorial's Camp lesson has a hint. Tests: engine unit tests (pool size, every tag and context has enough lines, replies follow prompts, no repeats in the window), company wiring (`wiring_banter_test.go`: personalities rolled, victory banter, off switch, only victories, 1-in-3 rate, fallen noticed once, purge), camping (`banter_test.go`: start and end of a rest), GMCP payload, `set banter`, help, and the dock browser check. Decisions (full autonomy, Robinson 2026-10-06): the leader never speaks (the player's own voice is theirs), only companions; the line pool is generic, so it survives the world being replaced; banter history and the latest exchange are in memory only (an exchange is flavor); lines are authored per tag rather than generated, so every line reads naturally. Review (accepted): (1) a fall was "any companion dead now and not at the last battle's end", so after a restart, or for a companion already awaiting resurrection, the next win mourned an old death; falls are now noted when a death is recorded in the fight and mourned once, unless raised first (`TestFallenCompanionIsNoticedOnce`, `TestCompanionKilledInTheFightDrawsFallLines`). (2) Fall lines buried the fallen ("Rest in peace", "We lost", "is gone") though a dead companion can be resurrected; they now grieve and push for a church or shaman (`TestFallLinesDoNotBuryTheFallen`). (3) Personality ignored alignment: an evil devout witch said the kind devout lines; kind lines are now good or neutral only, and 42 new lines (416 in all) blend personality with alignment (dark faith, gleeful cruelty, evil replies), cover Alchemist, Beast Tamer and Arbalist, and add falls (`blends.yaml`; `TestPersonalityAndAlignmentShapeThePool`, `TestEveryLineageHasAFamilyAndLines`). (4) Night talk (stars, moon, tonight) and "Good morning" lines fired at any hour; lines take `when: [day|night]` from the world clock and rest-end lines no longer assume a morning (`TestNightLinesWaitForNight`). (5) "Recruit Cleric" was called "Recruit", and two companions sharing a given name were indistinguishable; titles and clashes now use the whole name (`TestCallNameAvoidsTitlesAndClashes`). (6) A cheerful companion "laughed" every line, questions included; questions are asked and each personality has a second verb (`TestVerbsFitTheLine`). (7) `help banter` highlighted `personality` as a command, which leads to combat tactics; fixed, and it now explains falls and night talk. Checked and fine: nothing fires mid-battle (the BattleEnded listener drops talk if a new fight began), only the leader sees banter (no room spam), `{leader}` is the character's name, and rest-start talk is appended after the rest text, leaving room for phase 51 rest duties before it. Follow-ups: a companion's personality is shown nowhere (company status or the Company panel could show it); the Camp tab's "Around the fire" heading also shows after-battle talk. Gates after review: `make generate`, `make validate`, `make js-lint` and `go test -race ./...` pass; after the last master merges (38d, companion equipment) the banter, company, camping, gmcp, usercommands and tutorial packages were re-run.

**Companion equipment complete, reviewed and merged from [PR #84](https://github.com/Robinsond76/ashveil-gomud/pull/84) (2026-10-06): buff your companions and get their gear back.** Robinson asked to change what companions wear through the UI, with gear returning to the inventory on disband. What already existed (33g/34c): `company equip|remove|compare` for any member, an atomic cargo transfer (`commitAssets`, an `AssetOperation` journal written with the company file before the leader's file), and a Gear editor for the leader. What this adds: (1) **Gear return** (`modules/company/gearreturn.go`): dismissal (`company dismiss`, `dismiss all`), desertion and an expired rescue now return the gear the leader gave a companion to the leader's cargo in the **same company save** as the roster change (staged as an `AssetOperation`, installed on the leader's file after; a pending write finishes on the next command), with a report line; default gear stays behind (template items by item id, starter pack, each doll's cudgel) so recruit-dismiss-recruit never farms gear, and dolls' dressed gear returns with their Master. (2) **Web**: Company > Inventory (shared cargo) lists every member's slots, empty ones included, with tap-to-pick menus (phone-sized rows) and drag-and-drop (cargo onto a slot or member, worn gear onto the cargo); Character > Gear shows a row of member names and previews Current -> After for any member (`Company.Equipment` watch is now `open <slot> <member>`; `EquipmentViewForMember`). (3) **Text**: the equip reply names what moved ("Now: protection 3 -> 7, damage ..."), `company gear` adds the live effect line, `company status` has a gear line per member. (4) Help: new `help companion-gear` (indexed, linked from `help equipment` and `help company`, which are updated), tutorial Gear hint. Tests: dismissal, dismiss-all, expiry and pending-write recovery with no duplicated or lost item, default-gear and doll rules, member views and previews against applied stats, away members, GMCP watch and slots payload, help render, browser checks (dock windows with drag-and-drop and member tabs, phone Company view). Decisions (builder, owner delegation): (a) the shared cargo is the "inventory"; no separate transfer command was needed beyond `company equip|remove`; (b) cargo capacity never blocks the return (a company can be over capacity and still walk), so nothing drops in the room and a crash cannot lose an item; the report says when the company is now over capacity; (c) death: a fallen companion keeps its gear (existing 25b rule) and the gear returns only if the rescue expires; (d) default gear is identified by item id against the template, so a companion given a copy of its own default weapon returns nothing for that item; (e) an equipped drag-drop executes at once (the server validates and the change is reversible), tap-to-pick is the phone path; (f) screenshots: `/mnt/project-files/screens/companion-gear-*.png`. Review (independent, Opus): accepted: `help companion-gear` said a companion Doll Master's dolls could be dressed, but 39d keeps them at their cudgel and only the player's own dolls take gear; the page now says so (regression in `TestCompanionGearHelp`). Checked and kept: (1) "back to my inventory": on shared cargo the player's inventory *is* the company cargo (the same item list, with or without horses), so returning to cargo is returning to the inventory and nothing needs a pack-or-drop fallback; (2) item safety on equip, remove, dismiss, dismiss all, desertion, death (gear stays with the fallen), rescue expiry and a failed leader write (the staged `AssetOperation` finishes once on the next command); desertion and expiry only run for online leaders, so no return is skipped for an offline one; (3) recruit-remove-dismiss can still put a recruit's template gear in cargo (pre-existing since 33g), but shipped template gear has no sale value, so it is not a profit loop. Follow-ups: dressing a companion Doll Master's dolls (needs the same cargo-and-company-file operation as `company equip`); the admin test area (PR #83) must restore companion gear on return, checked by whichever review merges second.

**Phase 38d complete, merged via [PR #78](https://github.com/Robinsond76/ashveil-gomud/pull/78) (2026-10-06): Sorcerer and High Sorcerer, the first catalogue bundle.** The wizard gains a fourth advanced route, open to any alignment, with its elite. *Sorcerer* (10-25) learns **Arcane Lance** (`arcanelance`, a heavy bolt at one foe, 15+2d6 plus 1 per 7 levels and 1 per 8 Mysticism, about twice a Magic Missile; chant 2 rounds, cost 15, 12 at rank 25); ranks add Lance damage (+25%) and chant steadiness. *High Sorcerer* (30-60, `help high-sorcerer`): +40% Lance damage, a round off every other Lance's chant, chant breaks halved, Twin Lance (a second foe for a quarter), +45% Lance damage at 50, +20% mana, and an Instant Lance once a battle. Wiring: `UseBurst` in `internal/strategy` (caster casts the Lance ahead of Magic Missile while mana lasts, shipped config overlay), `startCast` trim/free, `SpellCost` override, `Lance*` effect keys, `harmmulti` spell so the twin bolt gets its target. Help: `wizard-routes`, `high-sorcerer`, `elite`, `classes`, `promotion`, `interrupts`, `strategy`, keywords, tutorial hint. Decisions (full autonomy): (1) 38d ships the Sorcerer bundle only; Elementalist, Illusionist and the other catalogue classes stay later bundles, and 38e no longer waits on 38d (each bundle is scoped and reviewed on its own). (2) Sorcerer is a burst class (costly heavy bolt), following the expanded-class design. (3) No new art: both fall back to the wizard sprite (art-pass follow-up).

**Phase 38d review (Opus review thread):** accepted and fixed: (1) Balance. The 3-round chant made the Sorcerer no better than a base wizard (a wizard in the mirror lives 3-6 rounds; mana was never short, 468 at L40), and Gathered chant trimming *every* Lance made the elite a one-round Lance each round once the chant was 2 rounds (L50 96%). Tuned: Lance chant 3 to 2 rounds, Gathered chant trims every other Lance (the 38c3 Quick casting pattern, shares `QuickCasts`), Lance level scaling 1 per 5 to 1 per 7 levels, Twin Lance 50% to 25%, Searing lance 60% to 45%. Result (100 fights a cell, even mirror, L40/L50): Sorcerer 5/4, Theurgist 8/15, Arcanist 5/9, Warlock 7/9; High Sorcerer 24/53, Archmage 25/29, Archon 31/37, Necromancer 13/19. The elite leads its advanced class by 19/49 and sits with the Archmage at L40; at L50 it leads Archon by 16 (accepted: the wizard advanced tier is at the floor, a 39i item). Regression `TestHighSorcererGatheredChantTrimsEveryOtherLance`. (2) UI: the chant's opening line gave the untrimmed count ("3 rounds") on a chant Gathered chant or Instant Lance shortens; it now names no count for a High Sorcerer (onWait gives the true one). (3) UI: the Instant Lance now emits a combat `Ability` event so the battle screen caption names it, as it does Overwatch and Aegis. Help, rank text, spell text and tutorial hint updated (no more "long chant"). Checked and fine: signature abilities fire through the real round (lance, twin, instant, trim, cost tests); promotion line and rank lines come from the shared 38c1 code; ranks listed once. Rejected: none. Follow-ups: Sorcerer and High Sorcerer sprites (art pass); the wizard advanced tier still wins under 15% in the mirror (39i); Archmage's Quick casting has the same untrimmed opening count (cosmetic). Gates: one `modules/company` race run failed once (output lost); three reruns and the final full race suite on the merged head passed.

**Phase 47 complete, reviewed from [PR #79](https://github.com/Robinsond76/ashveil-gomud/pull/79) (2026-10-06): polish follow-ups.** Six small items from the 46, 40i, 40h and 39d reviews, each decided under the full-autonomy rule. (1) *Overloaded flash:* in shared-cargo mode `CompanionCarry` counted only companions in the leader's room, so a step or camp break dropped a trailing companion's pack for a moment; it now uses a new `Runtime.Trailing` (same room, or one exit behind), so capacity holds while the company catches up (`TestCompanionCarryCountsTrailingCompanions`). Companions farther away still don't count. (2) *Gather after reconnect:* `VirtualWindows.setConnected(true)` fires `vwin:connected` and the Room window re-asks for `Room.Gather` on its next room (`room-check.mjs`). (3) *Phone views and docks:* mobile.js now claims panels by window id, not by dock (Map, Here: RoomInfo/Time/Tutorial; Company: everything else), marks panels `m-off` and docks `data-mshow`, and splits the screen when both docks hold panels of the front view; the old CSS assumed Map and Here in the left dock and Company in the right (`mobile-check.mjs` moves the Map panel to the right dock). (4) *Phone polish:* the map starts at zoom 1.5 (48 px tiles) on a phone unless a default zoom is set; the Camp Needs table gets a 34% name column and wrapping at phone width; Walk to... ends with "Find a visited room..." which opens a search sheet over every visited room, nearest first (`MapPlaces.search`; `uiMenu` items can now carry `fn`). (5) *Battle status legend:* a key under the picture names each coloured status dot on screen, in the dot's colour, and drops it when the status ends (`battle-check.mjs`; `help battlescreen`). Decision: the `ui/status` icons are not drawn on the battle screen; at 3 px dots in a 320x180 picture an icon would crowd the bars and the legend already names the colours. (6) *Doll Master:* a doll still standing when its Master falls now "goes limp as its Master falls" (room text and a battle-screen fall), then leaves the battle (`TestADollGoesLimpWhenItsMasterFalls`; `help dollmaster`). Skipped: nothing. Screenshots: `/mnt/project-files/screens/47-*.png`.

**Phase 47 review (Opus review thread):** accepted as built, with one help fix. Checked: (1) `Trailing` only widens `CompanionCarry` in shared-cargo mode, to a companion whose room has an exit into the leader's room; meals, morale, training, inventory availability and stray separation still use `WithLeader`, so the 32f presence rules hold, and a companion parked next door is separated as a stray after `strayRounds` as before. The `packs_test` change asserts the new intent (one move behind keeps capacity) and still asserts that a farther companion drops it. (6) The limp doll fires only when its Master is down and the doll still stands; the `Death`/incapacitated event lays it down on the battle screen, the Combat tab drops it with its next `Company.Battle` (live dolls only), and a battle that simply ended dismisses dolls without the line. UI at desktop and 360/390 px: legend, Needs table and search sheet read well. Fixed: `help mobile` and `help walkto` did not mention "Find a visited room..."; both now do. Rejected: drawing `ui/status` icons (agreed with the builder). Gates after merging master: generate, validate, js-lint, js-test, all nine browser checks, `go test -race ./...`.

**Phases 60–77 added to the roadmap (2026-10-07, owner request):** eighteen
phases drawn from a review of Pillars of Eternity and Deadfire, all of which
the owner liked. Story events, battle orders, explained battle lines and a
company chronicle come first; the chronicle feeds opinions, relic awakenings,
town memory, errands, creeds, bounties and blessings. A deep story dungeon is
recorded for the replacement world, not as a phase. Scope, dependencies and
build order are in the [phase plan](plans/2026-10-07-pillars-phases.md).

**Phases 50–56 added to the roadmap (2026-10-06, owner request):** seven
survival phases drawn from a review of Outward: condition into battle and meal
buffs (50), rest duties (51), tents (52), defeat scenarios (53), sigils (54),
ailments (55) and recipe discovery (56). Scope, overlap check and build order
are in the [phase plan](plans/2026-10-06-outward-survival-phases.md).

**Phase 54 complete, reviewed from [PR #87](https://github.com/Robinsond76/ashveil-gomud/pull/87) (2026-10-06): sigils.** `cast sigil of [fire|ward|stillness|mending]` lays a sigil before a fight: mana (12-15) plus one sigil chalk (item 30060, sold in both markets, never bought back), 15 real minutes by a saved Unix expiry (survives restart; never moves game time), one to a room and one at a time per company, refused in battle. Any battle the company begins in that room while it is lit takes it for that battle: fire spells (`element: fire`: Shower of Sparks, Fire Flask) +25% and leave the foe Burning, ward gives every standing member a small one-blow ward (4 + level/3), stillness windchills every foe for 3 rounds, mending makes heals +25%. Enemies never lay sigils. Shown by `look`, Room.Info `sigils` and a Room-panel badge, Company.Battle `sigil`, the battle screen header and the Combat tab. `help sigils`, links from cast/spells/spell/battlefield/combat, tutorial hint. Plan and decisions: [54](plans/2026-10-06-phase-54-sigils.md). Decisions: stillness reuses Windchilled; a battle keeps the sigil it began with; sigils live on the leader's character; chalk is market-only; fixed 15 minutes. Follow-ups: a gathered chalk source, a sigil drawn on the battle screen canvas, enemy sigils (out of scope here).

**Phase 54 review (Opus review thread):** checked no mid-battle laying (refused while in a battle or fighting), restart (Unix expiry on the leader's character, no game time), and other companies (a battle reads only its own leader's sigil; enemies never get one). Accepted and fixed: (1) a leader without the Cast skill could never lay a sigil, so Warrior- or Rogue-led companies had no use for it; now the first living caster companion in the room draws it from its own mana (`sigilDrawer`, tests). (2) The ward sigil warded the front row only, and foes going for the weakest mostly walked past it: the balance run showed it worthless (-5 to +7 wins). It now wards the whole company with one blow each. (3) A sigil's small ward blocked a healer's real ward and made healers count the member as warded; `ClassRT.WardSigil` lets a cast ward replace it and healers ignore it (regression test). (4) Fire looked strongest in the first run (+22 to +30 wins at L12/L20); re-runs at 15%, 20% and 25% put it at -17 to +13, so that was noise and the build's 25% stays. Balance (`TestPhase54Sigils`, opt-in; a Wizard in slot 3 against five even foes with a healing priest, 40 fights a cell, about ±11 points of noise): even level, wins vs no sigil at L5/L12/L20: ward +17/+5/+23, stillness -5/+5/-2, mending -13/-20/+8 (earlier run +5/+17/+10), fire at 25% (60-fight re-run) +13 at L12, 0 at L20. Every sigil left zones three levels above the company unwinnable (0-2% wins), so none breaks the difficulty rule. Timeboxed and settled: no sigil is a must-have; stillness and mending are modest. Rejected: per-room storage (leader storage is enough with one sigil per company). Follow-up: draw the sigil on the battle screen canvas; a gathered chalk source.

**Phase 40i complete, reviewed from [PR #68](https://github.com/Robinsond76/ashveil-gomud/pull/68) (2026-10-06): touch layout and installable web app.** On a viewport of 820 px or less the web client shows one view at a time from a bottom bar (**Game**, **Map**, **Here** for the room, clock and tutorial, **Company** for the dock), with the command box and a **touch bar** always on screen: N S E W U D, **Walk to...** (the named places the map knows, nearest first, sending `walkto [room]`; leads with "Stop walking" and shows a **Stop** button while a walk is under way), Look, Inventory, Status, Camp; it folds away and remembers that. The map takes touch: one finger pans, two pinch to zoom, a tap on a tile opens the 40d "Walk to [room]" menu (a drag or pinch never does), and the hover tooltip is off on phones; `uiMenu` rows are 44 px with a viewport-capped height. The battle screen is full screen on a phone, its picture fills the width at any scale, buttons are 44 px, an allied company's pennant and the figures have a larger touch hit area (tap a pennant to watch that company; own band to return), and the minimised badge clears the bottom bars. Mid-battle inputs are unchanged (retreat and company focus). The web app manifest (`site.webmanifest`, which was an empty stub) names the app Ashveil, starts at `/webclient-pure.html`, standalone, with the S1 app icons (192, 512, maskable); `webclient-pure.html` gains a viewport meta (device-width, `viewport-fit=cover`, `interactive-widget=resizes-content`), theme colour and the S1 favicon and touch icon. Code: `static/js/mobile.js`, `static/css/mobile.css` (every rule under `body.mobile`), docked panels carry `data-win`, `window.MapPlaces` (list, walking, viewport) from the map, touch handlers in the map, `fit()` in the battle screen. Help: new `help mobile` (configuration hub; aliases phone, touch, touchscreen, touch bar, install, install app, add to home screen, pwa, web app), linked from `help webclient`, `help walkto` and `help battlescreen`; the Character and Departure tutorial hints mention it. Tests: `scripts/browser/mobile-check.mjs` (the real `webclient-pure.html` at a 390 x 780 touch viewport: views and which panels each shows, 44 px targets, no sideways scroll, touch bar commands, tap-to-walk, pan and pinch, Walk to... list and Stop, Room Info gather strip, Camp tab, battle screen with an ally pennant tap, manifest and icons, return to the desktop layout), `TestMobileHelpRendersAndIsIndexed`, `TestWebClientShipsTheInstallableManifest`. Screenshots: `screens/40i-*.png`. Design: [phase 40i](designs/2026-10-06-phase-40i-mobile-design.md). Decisions (owner delegated): (1) a width breakpoint, not touch or user-agent detection, so a narrow desktop window is testable and a rotated phone switches live; (2) four views, with Room Info and the clock sharing Here so the map gets the whole screen; (3) the touch bar sends plain commands, so history, battle refusals and text parity are unchanged; (4) Walk to... lists only named places (landmark legends), other rooms are reached by tapping the map; (5) no service worker, because the game needs a live connection and current browsers install a manifest-only app; (6) floating (popped-out) windows are hidden and autohide docks ignored on phones; (7) a 13 px terminal font on phones. Follow-ups: the map starts at the desktop tile zoom (a larger phone default could help); the dock's Needs table is wide for 360 px screens (it fits at 390); a visited-room search for Walk to...; swipe between views.

**Weather on the Time and Date panel (owner ask, 2026-10-06):** the server already had per-zone weather (modules/weather, the `weather` command and room look line); only the web client did not show it. `Gametime` GMCP now carries a per-player `weather` object (name, description, cloud cover, visibility, `indoor` when seen through an exit) for the room the player stands in, omitted indoors with no view out or in a zone with no weather. The panel draws drifting cloud, rain, storm lightning and fog over the sky, dims the sun with cloud and hides the moon when overcast, and shows a name tag. No new gameplay effects. `help weather` and `help webclient` updated, the tutorial's weather hint points at the panel; browser check `scripts/browser/weather-check.mjs`. Follow-ups: snow and heat-haze conditions once zones have them; weather ambience sound.

**Weather panel review (Opus review thread, merged via [PR #97](https://github.com/Robinsond76/ashveil-gomud/pull/97)):** checked that the payload follows each player's own room (outdoors, glimpsed through an outdoor exit, none in a sealed room or untracked zone) and refreshes every round (4 s), so a zone change shows within one round; the animated sky keeps the weather between ticks. The panel reads well at a glance (screens/weather-panel.png). No defects found. Rejected: a weather line in `time` output, because players have no `time` command (only the admin `server time`); `weather` and the room look line already name it.

**Phase 40i review (Opus review thread):** judged at 360 and 390 px touch viewports plus the desktop browser checks (dock, dock windows, battle, map, room, class art, tutorial panel). Accepted and fixed: (1) the touch bar was one sideways-scrolling row, so at 360 px Walk to..., Look, Inventory, Status and Camp sat off screen (Playwright's tap scrolled them in, hiding it); it is now two rows (fold arrow and compass, then Walk to... and the commands), every button in sight at 360 px, and the fold leaves only its arrow (new checks). (2) Look, Inventory, Status and Camp answered in the hidden terminal when tapped from the Map, Here or Company view; they now bring the Game view forward (the compass leaves the map in front). (3) The battle screen's Help sent `help battlescreen` to the terminal behind the full-screen battle; on a phone it now minimises the screen and shows the Game view. (4) The focus buttons (none, leader, casters...) were named only by hover titles, which a finger can't see; the row now starts with a "Focus:" label (desktop too). (5) The map only re-measured through a ResizeObserver, and after the Map view was shown before map data arrived and a command was sent, the canvas stayed 1 x 1; the map now also re-measures on the window resize the view switch announces. (6) The minimised badge's offset was a fixed 150 px, under the now taller bars; `mobile.js` measures the bars into `--mobile-bars`. `help mobile` updated for (2) and (3). Gates after merging master (40h, 39d): generate, validate, js-lint, js-test, mobile check at both widths; the race suite failed in `modules/company` under full load (passes alone, the known 39d-noted flake) and in `scripts` `TestEveryLineageHasBaseArt`, which is red on master too (40h's strict lineage-art test landed after 39d added the `dollmaster` lineage without art; not this phase's). Follow-ups: Doll Master base art to turn master's art test green; the phone views assume the default dock sides (Map and Room Info left, the tab group right), so a desktop layout with the map moved to the right dock would leave the Map view empty: select panels by window id across both docks; plus the builder's (phone map zoom, Needs table at 360 px, visited-room search, swipe between views).

**Phase 46 complete, merged via [PR #61](https://github.com/Robinsond76/ashveil-gomud/pull/61) (2026-10-06): polish and caster balance.** Five loose ends from earlier reviews.
(1) *Caster balance:* re-measured at 100 fights a cell on current master, Wizard 36/44/33%, Witch 37/46/31%, Shaman 41/59/23% at levels 5/10/20. The Wizard is already within 5 points of the Witch, so no change: forcing Magic Missile over Shower of Sparks (never an area spell against groups of 4 or more) moved it only inside the noise (30/52/30% and 27/55/22% at 40 fights), so the spell choice was not the cause the 39c review suspected. Shaman at level 10 stays about 13 points high; trimming Lightning (base 9 to 8 and 7) did not move it (55%), so it comes from the Shaman's body and stays for the 39i balance pass; noise at 100 fights is about +/-5. (2) *City weather:* a `city` weather table (clear, overcast, rain, mist) so `weather`, look, temperature and the forecast work in Dunmar, Frostfang and other cities; every multiplier is neutral (a city never slows, tires or spoils a rest), and a journey whose origin weather changes nothing for travel now takes the destination's weather (`departureFactorsLocked`), so leaving Dunmar keeps the forest's weather as before (`TestNeutralOriginWeatherDefersToTheDestination`). Decision: neutral multipliers because cities shelter you; no fog (visibility) in cities. `help weather` updated. (3) *"Overloaded" at 12.9 of 30 kg after `camp break`:* not reproduced. Likely cause: company capacity only counts companions standing in the leader's room (`CompanionCarry` checks `WithLeader`), so a step that leaves a companion a room behind for a moment drops capacity below the load and the label reads Overloaded until it catches up. Deliberate (32f review), so not changed; follow-up if players see it again: count attached companions that are only one move behind. (4) *Halberdier:* Sweep, Brace, Hook and the wide sweep are base ranks (levels 1, 3, 6, 8, 20) so a level-up names them in the shared "New rank" line like the Samurai and Shaman; the power report no longer repeats them (`TestHalberdierLevelsAreNewRankLines`); `class` lists them. (5) *Reconnect mid-gather:* the Room window asks for `Room.Gather` on its first room after a load, and the server answers with the work in progress and how much is done (`elapsed`), so the bar resumes (`TestRoomGatherResumesWorkInProgress`, `scripts/browser/room-check.mjs`). Review (accepted): city rain and mist carry a TemperatureMod, so they do chill the air (and can feed cold exposure at night); `help weather` claimed city weather never tires you, and now says it never slows a walk, adds strain or spoils a rest but rain and mist chill the air a few degrees. Review (checked, no change): the Wizard and Shaman calls hold (Wizard within noise of the Witch; Shaman L10 left for 39i); the Overloaded explanation fits the code (the leader's prompt renders before following companions arrive, so `CompanionCarry` briefly drops their capacity), kept as a follow-up rather than reworking the 32f presence rule; Halberdier ranks carry no effects, so combat is unchanged and `class` now lists them; city firewood gathered in rain comes back damp like any outdoor rain, acceptable. Follow-up: the Room window asks for `Room.Gather` once per page load, so an in-page copyover reconnect does not re-ask.

**Phase 38c2 reviewed and merged via [PR #60](https://github.com/Robinsond76/ashveil-gomud/pull/60) (2026-10-06, Opus review).** Independent reviewer findings, each verified. Accepted and fixed (regression tests in `wiring_rogue_ranger_test.go` and `class_effects_elite_test.go`): (1) Second Nock was a second Aimed Shot (forced crit, strike bonus, Pinning Crit) because the live aggro was still a backstab; it is now a plain shot. (2) Watchful added nothing beside the Warden's own back-row aura; it now adds on top. (3) Coup de Grace lost its label on a crit (assignment instead of append) and read the boss flag only from battle state; it now reads the mob. (4) Perfect Shot was spent when chosen, even if never loosed; now spent when loosed. (5) Death Mark's foe state is re-read each round. (6) Help: Ravager 75% and +20% (stale), Sentinel and abilities wording, Marksman "needs a shooting weapon" only for Called Shot's chance, Vanish lasts "this round and the next" (what the code does). Rejected or deferred: Overwatch firing on the aimed member rather than the final one is intended ("goes for"); Quick Draw talent wasted at Marksman's cooldown floor, Pathfinder's Eye treating a foe whose blow Guardian Arrow stopped as not having acted, and missing round-loop tests for Perfect Parry, Shadowstep, Spree, Vanish, Envenom, Harrow, Guardian Arrow, Twin watch, Rend, Unblockable and Scouted ground are follow-ups.

Balance decisions (reason: elites must beat their advanced class): Overwatch now comes after a ready Aimed Shot, and a hold no foe answers looses its arrow at the Sentinel's own foe at the round's end (a quiet hold had cost a whole blow). Death Mark is 40% and leaves the marked foe exposed for 2 rounds. Final cells, 200 fights: Sentinel L50 93% wins / 35% HP lost vs Warden 81/45 (4 foes two levels above); Sentinel L35 92/36 vs Warden 91/35 (level: only Overwatch and Steady overwatch by then); Nightblade L35 70/51 vs Assassin 73/47 and L50 94/37 vs 94/38 (4 foes of the company's level). Accepted miss: the rogue front-liner's share of company output is too small for any rogue elite to move these cells beyond noise (base and advanced rogues sit within a few points of each other too); Pathfinder and Swordmaster show at L50 in the hard cell (62% wins each vs Scout 38%, Duelist 34%, 100 fights). Follow-up candidate: rogue front-liner survivability and a per-member damage measure in the harness. UI: the battle screen's caption now names abilities ("Ysolde: Overwatch on the bandit captain"), so elite signatures are visible there as well as in the text lines.

**Phase 38c2 built: rogue and ranger elites (2026-10-06):** Pathfinder, Swordmaster, Nightblade (rogue) and Sentinel, Marksman, Ravager (ranger), following 38c1's framework (promotion at 30, catch-up ranks, elite talents at 35/45/55, status/class/roster/GMCP/web display unchanged because they read `classes.Describe`). `eliteShipped` now covers rogue and ranger. Combat riders: `internal/hooks/combat_rogue.go` (Death Mark, Shadowstep, Vanish, Killing Spree, Opening-Strike riders, Envenom), `combat_ranger.go` (Overwatch, Second Nock, Hunt Down, Harrow, Apex), `internal/characters/class_elite.go` (`EliteRT`, runtime only), `combat` formulas (Coup de Grace, Perfect Shot, Perfect Parry, crit damage), `enemyparty.CompanyEffect` (Pathfinder's Eye, Ambush Master, Scouted ground, Trailwise), `loot.EquipmentWith`, and a reworked `riposteBlow`. Overwatch is class-granted through `strategy.WithClass` and shows in strategy, company summary and GMCP. Help: `help rogue-routes`, `ranger-routes`, `elite-pathfinder`, `swordmaster`, `nightblade`, `sentinel`, `marksman`, `ravager`; updated elite, classes, promotion, talents, abilities, combat and pathfinder pages; tutorial hint in the Combat lesson. Tests: rank and talent boundaries, wiring through real battles for each signature (Pathfinder opens, Nightblade mark, Sentinel hold and shot, twin ripostes, Ravager bleeding and Apex, Marksman Perfect Shot and Second Nock, ambush halving and flip), combat formulas, morale, loot, Trailwise gold, help render.

Task 0 decisions (design text adjusted, each with its reason): (1) roles in the design are descriptive, not new strategy enums; targeting uses the existing rules, and Death Mark marks the aim at battle start and passes to the most hurt foe in reach. (2) Pathfinder's Eye opens a foe that has not acted once a battle (twice at 50); halving and the Ambush Master flip happen in `engage`. (3) Trailwise: the shipped cache is guaranteed, so it adds 15% gold and 10 points to the equipment chance. (4) Swordmaster: no shipped per-battle riposte limit, so ripostes are limited per round; rank 55 "reach ripostes" became Counter-strike (target exposed a round); Parry is 9. (5) Sentinel: Warden has no Covering Shot, so +5 Attack applies to the Overwatch shot; Overwatch triggers when a foe is aimed at a middle or back-row ally (before the front row intercepts, since interception made a reach-only trigger almost never fire in the harness), and a hold with no strike is simply spent. (6) Marksman: Aimed Shot is already a sure crit, so Called Shot gives +15% crit on shooting blows and +50% crit damage. (7) Ravager: Hunt Down opens a bleed at or below 75% health (50% left Ravager level with Stalker), flee penalty never below 5 points of the band; Bloodscent +20%. (8) Elite talent "Keen Edge" renamed Razor's Edge (id exists as a base talent). (9) `help pathfinder` is the walking skill, so the elite page is `help elite-pathfinder`.

Balance (`TestPhase38c2EliteRoutes`, 40 fights a cell, 4 foes two levels above, wins / company HP lost): rangers L35 Hunter 97/26 vs Marksman 100/19, Stalker 95/28 vs Ravager 100/20, Warden 92/39 vs Sentinel 92/36; L50 Warden 80/43 vs Sentinel 90/40, Hunter 95/25 vs Marksman 100/21, Stalker 97/31 vs Ravager 97/27. Rogues L50 Scout 35/79 vs Pathfinder 55/67, Duelist 42/79 vs Swordmaster 55/76, Assassin 40/77 vs Nightblade 40/80. Misses: the rogue front-liner wins about 25% in every cell at L25 to L35 so elites cannot show there; Sentinel L35 and Nightblade are level with their advanced classes; Marksman/Hunter and Ravager/Stalker are within noise of each other at L50. Follow-up candidate: rogue-lineage tuning (Nightblade, Sentinel at 35) and elite art.

**Phase 39e complete, merged via [PR #70](https://github.com/Robinsond76/ashveil-gomud/pull/70) (2026-10-06): the Beast Tamer neutral lineage.** a handler who fights beside a living bonded beast. The beast is a record on the Tamer (name, damage, wounded) that stands as a charmed mob (mobs 200-203, race 26) for one battle, laid over the formation in front of its Tamer with no company slot, and **takes its own turn** at 70% tempo; a fall wounds it until the company rests, never kills it. Sic (rank 1, +10 Attack on the beast's strike, whip reaches like a polearm), Rally (3, twice a battle), Pack Sense (8, +5 Evasion); routes Houndmaster (warhound, hobbling bites), Bearward (war bear, guards), Dragon Tamer (drake, Breath every 3 rounds); archetype `beasttamer` (HP 5 and 0.8, Attack 0.8, Evasion 0.9, medium armor), recruit mob 204, `beast` command, rest heals the beast. Help: `help beasttamer`, `help beasttamer-routes`, `help beast` plus related pages; tutorial hint in the creation lesson; base map and battle art. Plan: [39e](plans/2026-10-06-phase-39e-beast-tamer.md). Balance (30 fights, five-foe groups, Tamer vs warrior and ranger third slot): 80/96/76% vs 76/86/90% (warrior) and 83/90/76% (ranger) at L5/10/20. Decisions (owner delegated): Sic costs the Tamer no blow; beast health 35% of a warrior's and Tamer Attack 0.8 instead of the design's 60% and 0.9 (design values won 100% at L10); beast kinds reuse existing unit art; feeding deferred. Follow-ups: feeding, beast sprites, morale panic, elites (39i), creature recruits (38e).

**Phase 39e review (Opus review thread):** kept the build's decisions (Sic costs no blow; rest is the cure; existing unit art for beasts) and judged feeding and morale panic fine as follow-ups (rest already gives the wounded beast a cost in time, and panic needs the company morale path to cover a mob with no record). Mob 204 and mobs 200-203 collide with nothing. Accepted and fixed: (1) a beast sent down the ordinary death path (`suicide`, a spell or status kill) died with a corpse and reward and left its record unwounded; it is now wounded and dismissed (`TestABeastOnTheDeathPathIsWoundedNotKilled`); (2) `strategy [member] abilities off` did not hold Sic or Rally back although the help says it does (`TestAbilitiesOffHoldsSicBack`); (3) the `beast` command's health ignored the beast race's vitality, one point off the live beast (`TestTheBeastCommandShowsTheHealthTheBeastFightsWith`); (4) UI: a wounded beast silently missed battles, so the battle now says once that it sits out (`TestAWoundedBeastSitsTheBattleOutAndSaysSo`); (5) help numbers were the design's, not the shipped ones (beast health, 70% tempo, Tamer Attack 0.8), and the Dragon Tamer's Scorching breath claimed Breath only while it also adds to bites. Balance at 100 fights a cell: the flat 35% beast won 94/97/73% at L5/10/20 against warrior 74/91/78 and ranger 79/91/74 (the build's 90% warrior at L20 was 30-fight noise), and tests showed the beast's value was its body, not its bite or Sic, so the beast now grows with its Tamer: health 8% of a warrior's to level 4, then 2% + 1.75% a level (19% at 10, 37% at 20), and a 1d3 bite (+1 every 3 levels). Final: Tamer 77/90/76, warrior 84/90/71, ranger 75/82/76; routes at L25 base 79, Houndmaster 81, Bearward 86, Dragon Tamer 89; at L15 the Bearward sits 9 under base (73 vs 82), accepted for 39i. Rejected: dismissing the beast mid-battle when its Tamer falls is kept (the help now says so). Follow-ups: feeding, morale panic, bear and drake sprites (they show as silhouettes), the beast's health in the level-up report and company display, and the Bearward's mid-level dip (39i). Not this PR's: `TestAlliedFinalEnemyPaysAfterCombatClosesBattle` failed 2 of 10 race runs under load after the last master merge (a companion corpse beside the captain's; no Beast Tamer in that test) and then passed 40 of 40 on this branch and on master, so it is a load-sensitive flake to watch.

**Phase 39d complete, reviewed from [PR #59](https://github.com/Robinsond76/ashveil-gomud/pull/59) (2026-10-06): the Doll Master neutral lineage.** a back-row Master fights through a wooden doll. Dolls are records on the Master (player character or companion state: name, damage, broken, gear) that stand as charmed mobs (mob 161) for one battle, laid over the formation in front of their Master, with no company slot, aim or turn of their own; Puppet Strike makes the Master's turn the doll's strike. Guard String (rank 5, 2 then 3 guards), Tangle (12), Emergency Splice (18); routes Puppeteer (two dolls), Golemancer (golem), Marionettist (tangle); archetype `dollmaster` (HP 1 and 0.55, Attack 0.7, Evasion 0.8, light armor), recruit mob 160, doll parts item 70 sold at Dunmar, `doll` command, camp-rest mending. Help: `help dollmaster`, `help dollmaster-routes`, `help doll` plus updates to the related pages and the formation page; tutorial hint in the creation lesson. Plan: [39d](plans/2026-10-06-phase-39d-dollmaster.md). Balance (60 fights, five-foe groups, Doll Master vs warrior third slot): 78/91/73% vs 81/91/65% at L5/10/20, within noise of the five-point target. Decisions (owner delegated): dolls are summon-like mobs plus a registry rather than company records, to avoid roster, food, morale and XP machinery; a Master with no able doll swings its own dagger; Tangle shifts the meter for the next round; `doll mend` works anywhere out of battle and camp rest mends automatically; companion Masters' dolls keep the starter cudgel; doll parts only bought back unused. Follow-ups: battle screen and GMCP presentation of dolls, dressing companion Masters' dolls, doll sprite, elites (39i).

**Phase 39d review (Opus review thread):** accepted and fixed: (1) UI: dolls were invisible in battle; the battle screen drew nothing in their cells and the Combat tab named no one when a foe struck a doll. `Company.Battle` now carries `dolls` [{key, name, master, hp, hp_max}] (the key is the id the doll's events use and its key in `positions`); the Combat tab lists each doll as a fighter ("Pip, 30 / 40 · your doll") and names it as a target, and the battle screen draws it as a painted-wood figure until doll art lands (`TestBattleFeedListsTheCompanysDolls`). (2) Marionettist's rank 15 gave +2 Attack to the Master, who does not swing while a doll stands; it now adds Attack to the doll's blows. (3) Puppeteer's rank text said Strike drives only the first doll; both standing dolls strike, as the code and help say. (4) `help dollmaster-routes` named ranks the code does not (Practiced hands, Deep core, Stone skin and others) and gave armor as flat points; it now uses the ranks' names and percent armor. Balance at 100 fights a cell (five-foe groups, Doll Master vs warrior third slot): 79/93/82% vs 80/92/75% at L5/10/20; L20 rechecked at 300 fights: 79% vs 74%, at the five-point target, so no tuning. Decisions: (a) companion Masters' dolls stay undressable this phase: moving an item from the leader's pack to a companion's saved doll crosses two stores and needs the company's asset commit path (`commitAssets`), so it is a follow-up rather than a quick fix; (b) doll parts are `SupplyOnly` (43b's flag, merged meanwhile): sold in Dunmar, never bought back, so the two starter parts can't be sold either; mending consumes parts and creates nothing resellable. Checked: dolls drop nothing and take no XP, wounds or food; mob 160/161 and item 70 collide with nothing on master or open branches; abilities stay hidden below their unlock level (`strategy.AtLevel`); route ranks use the shared "New rank" line. Gates after merging master (37c): generate, validate, js-lint and the race suite green; the first race run failed once in `modules/company` under full-suite load and passed in three reruns (the failing test wasn't captured; watch for it). Follow-ups: dressing companion Masters' dolls (asset commit), doll and Doll Master sprites (art pass), a "goes limp" line when a Master falls with its doll standing, elites (39i).

**Phase 43a reviewed and merged via [PR #58](https://github.com/Robinsond76/ashveil-gomud/pull/58) (2026-10-06, Opus review thread):** checked the queue/lock/spend paths (a refusal or short stock spends nothing; broth and incense are spent only when a rest starts and locked on the saved rest; a raid-spoiled rest gives no broth), the incense rule in `fireDueRaids` (only with a posted watch, capped at 90%, never lowering a sure watch), the exposure cut (direction-matched, rounded so a stress of 1 still counts), the empty waterskin in every pack the company drinks from (player, companion, cargo; the drinking order of 32f is unchanged and `company fill` refills all three), economy (all four supplies `SupplyOnly`; shopkeepers pay at most a quarter of value, below every market price; the empty skin has no value), item and buff ids (30040-30044 and buffs 70-75 are clear of 43b's items 280-283 and buffs 1120-1123), and the UI (`camp prepare status` names each member's benefit and its time left, conditions list the buffs, the Camp tab lists supplies and the queue). Accepted and fixed: (1) the broth was granted after the finished rest restored vitals, so the drinker woke at the old maximum and the raised limit was out of reach (road regeneration stops at half); it is now granted first, so the rest fills the drinker to the fortified limit, and `help camp supplies` says so (`TestBrothDrinkerWakesAtTheRaisedHealthLimit`). Decision (antidote consistency with 43b): no antidote ships now. 43b dropped the design's curative shop antidote as a combat action (poison ends with the fight; `curepoison` and cleansing cure it), and the camp antidote draught is preventive against poison delivered *to* the company, which waits until enemies coat their weapons; it stays in the camp consumables design with scent paste and weapon oil. Merged master (39c, 44b, 37c, 43b, 40d); 43b conflicts resolved by keeping both (the `camp` usage lists supplies and poisons, `help camp` links both, both supply-only market tests kept; ids do not collide). Follow-up: `TestSpawnLootRollsAnItemIntoTheRoom` still fails about half of plain `go test ./internal/usercommands` runs on master too (the roll picks the shipped scale hauberk over the test one).

**Phase 43a built (2026-10-06): camp supplies and the empty waterskin.** First slice of the [camp consumables design](designs/2026-10-01-camp-consumables-design.md): four supplies, prepared at an established camp with `camp supplies` and `camp prepare [supply] [member|all]` (also `status` and `clear`). Items 30040-30043 (consumables range, clear of camp gear 45-50): **fortifying broth** (+about 5% health limit, sized 2/5/10/20 by the member's maximum, for 15 real minutes after a finished rest; buffs 70-73), **warming draught** and **cooling salve** (a quarter less cold or heat exposure taken on for 15 real minutes; buffs 74-75, flags `warming-draught` / `cooling-salve`, read by `modules/exposure` in its tick), **watch incense** (+10 points to a posted watch's chance to spot raiders, never past 90%, never lowering a sure watch). Broth and incense are queued on the camp (`Camp.Prepared`, saved) and spent when the rest starts; funded broth and incense are locked on the rest (`RestSession.Broth`, `Incense`, saved), incense reaches `fireDueRaids`, broth is granted once, with the save that clears the rest's marker, and a rest spoiled by raiders gives none. One personal benefit per member (broth, draught or salve): never replaced silently, `camp prepare clear` drops it and refunds nothing; a refusal or too few doses uses nothing. Sold at Dunmar (10/12/12/8) and the road post (dearer), `SupplyOnly` so none buys back. Player surfaces: `help camp supplies` (indexed with aliases, linked from `help camp`, `help campwatch`, `help temperature`), rest-start report line, `camp status` queue line, web Camp tab "Supplies" and "Set by for the next rest" lines (GMCP `Company.Camp.supplies` / `prepared`; screenshot `/mnt/project-files/screens/43a-camp-supplies.png`), buffs listed under the conditions rest and survival groups, and a tutorial Camp hint. **Empty waterskin** (closes the 40a follow-up): item spec fields `emptyitemid` / `filleditemid`; spending a waterskin's last use (`Character.UseItem`, a companion's saved pack, the cargo via `Cargo.ConsumeUseLeaving`) leaves item 30044 (empty waterskin) instead of nothing, and `fill` and `company fill` turn an empty one back into a full 30015 in a pack, a companion's pack or the cargo; `help resources` and `help drink` updated. Tests: prepare/queue/spend/lock/grant/clear/refusal paths through the real `camp` command and rest start and completion, raid detection with incense (including the certain-watch and 90% rules), exposure tick with each supply, shipped data (items, buffs, Fortified sizes and expiry clamp, waterskin chain), market supply-only, GMCP payload, help render, browser check. Decisions (builder, owner delegation): (1) the first slice is broth, warming draught, cooling salve and incense, as the design recommends, because raids now exist; **scent paste** needs typed beast events in encounter data, **weapon oil** a weapon break-check hook, and the **antidote draught** weapon poisons (43b), so those three stay in the design for follow-ups; (2) personal benefits are real buffs of 15 real minutes (persisted with the character, shown in conditions) instead of a new UTC-expiry store, so they run while the member is online and a draught is not locked to a route segment; (3) supplies are spent when they take effect or when the rest starts, with no per-instance reservation (stock is re-counted at rest start and missing broth is dropped with a note), simpler than reserving item instances; (4) broth size follows four fixed buffs rather than a computed percentage because buff statmods are flat; (5) incense with no watch at rest start stays queued instead of being wasted; (6) the empty skin is its own item (30044) because an item at 0 uses is refilled to full by `Validate` and cargo stacks carry only an id and uses. Follow-ups: scent paste, weapon oil and the antidote draught (the last with 43b); crafting the supplies (design: later); merchants other than Dunmar and the road post sell none yet.

**Phase 40d complete, merged via [PR #56](https://github.com/Robinsond76/ashveil-gomud/pull/56) (2026-10-06): tile-ready showcase region and click-to-walk.**
a new zone **Alderbrook** (44 rooms, ids 2101 to 2144, `tileready: true`) lies east of the Trappers' Post (2005 gains an east exit; its tags and the camping fork are untouched). It shows every map feature: road, meadow, forest, farmland, hamlet, a stream and a lake with a dense eight-room shore ring, a rocky hillside with a cave mouth leading down, a farmhouse with a locked loft (`up`), a campable clearing, and landmarks (village, bridge, hermit, rocks, obelisk, cave, lake house), with every resource type. The tile-ready conventions are in [docs/designs/tile-ready-conventions.md](designs/tile-ready-conventions.md) and enforced by `rooms.ValidateTileReady` (coordinates unique per level, exits to the adjacent coordinate or a `mapdirection`, a biome and known resources, filler descriptions of at most two sentences, landmark legends that map to S2 art); `TestShippedTileReadyZonesFollowTheConventions` runs it on every shipped tile-ready zone and a fixture proves each rule fails. **`walkto [place]` / `walkto stop`** (module `modules/walkto`, planner `internal/walkto`, alias `autowalk`): the leader walks a route of at most 60 steps through visited rooms and usable exits (no unfound secret exit, journey exit or exit with a delay message; a locked door only with its key), one ordinary `go` per step every 1.5 s of real time (`StepMillis`), with strain, encounters, followers, hunger and weather as for a typed move, and never touching the game clock; it stops for a battle or aggro, a hostile in the room, a blocked or changed exit, a newly warning need or zero fatigue, any typed command other than look, map or walkto, and quit, death or restart (not persisted). GMCP `Walkto {target, path}` (`{}` when not walking) feeds the web map. **Web map:** click a room for `Walk to [room]` (one pick; `Stop walking` while walking; admins keep teleport), path dots and a destination flag from the S1 markers, and the two 40c review follow-ups: S1 resource icons replace the dots (dots stay on classic, on tiles under 24 px, and for an icon still loading) and outdoor tiles are shaded by the game's clock (dusk and dawn an hour each; `World.Map` biomes gain `indoor`/`dark` so houses and caves stay lit; Day/night setting). Help: new `help walkto` (road hub, aliases `autowalk`, `click-to-walk`, `click to walk`, `walk to`, `tile-ready`, `alderbrook`), `worldmap`, `webclient`, `resources` and `travel` updated; Departure tutorial hint points at it. Tests: planner (visited-only, lock and key, cap of 60, nearest match, no secret/journey/delay exits), module (start, status, stop, every stop reason through the real Go command and walking-strain hook, stale timers, the registered Input listener through `plugins.Load`, the game clock unchanged, the shipped Alderbrook rooms walked across a zone boundary, the lake loop, the locked loft), GMCP feed and biome flags, help, and `scripts/browser/map-check.mjs` (icons, path, click menu, day/night). Screenshots: `screens/40d-alderbrook-*.png`. Decisions (owner delegation): (1) the command is `walkto` because `travel` is the journeys command, and GMCP is `Walkto`; (2) planning is a new breadth-first search over visited rooms rather than the mapper's A* (the mapper walks every exit of a zone map; a walk must honour visited rooms, secrets and locks and cross zones); (3) landmark words search the current zone only, a room number works for any visited room; (4) exits with an `exitmessage` are not walked, as their delayed requeue would read as a typed command; (5) a warning already showing when the walk starts does not stop it, only a new crossing does, so a hungry company can still walk to the inn; (6) the showcase is a new zone, not a densified starting region, and its landmarks avoid shop, smithy and herbalist (41/42 add those); (7) resource icons draw at half size on a 32 px tile (up to three plus a plus), dots stay as the fallback; (8) day/night uses one multiply tint, not the S2 night mask, which is left for dark-room shading; (9) the lake's open middle is not a room because deep water needs an item to enter. Follow-ups: a `walkto` entry for the mobile touch layout (40i), shop/smithy/herbalist landmarks with 41/42, `night-mask` for dark rooms. Review (accepted): a step the Go command refused (room script, no-go buff, no action points, battle) was re-queued every 1.5 s forever with its refusal line each time; the walk now stops after one refused step (`TestARefusedStepStopsTheWalk`); the map tooltip stayed open under the walk menu and is now hidden. Review (checked, no change): walkto only plans and reveals visited rooms and found secret exits; a sprung encounter stops the walk on the next step; it calls only the existing journey/camp `MovementBlocked` checks, so it adds no locking beside the 44b fix; Alderbrook has no mobs, items or encounters (no ID clashes, difficulty and resale rules unaffected), so it is a peaceful zone until 41 populates it. Also fixed in review: `TestSpawnLootRollsAnItemIntoTheRoom` (the 37c flake) spawned "hauberk", which the shipped scale hauberk also answers; its test item is now a byrnie.

**Phase 44b complete (2026-10-06): world smoke playtest.** `make
smoke-world` (`live_smoke_world_test.go`) plays a Warrior that skipped the
tutorial through the world on a real server: recruiting at the Waymark Inn,
the Old Kings Road journey (the fallen tree and `travel resume`), a random
encounter fight and its loot, `gather firewood` and a camp, a restart, a
salvage at the Frostfang armorer, a sale at the Dunmar market and a night at
the Dunmar inn; see [Live smoke playtest](LIVE_SMOKE_PLAYTEST.md). It is its
own target (about four minutes). Three things are bent in the test's own
copy of the world only: new characters start at Dunmar's West Gate, the Old
Kings Road gets an always-springing encounter table (two unarmed brigands),
and the account is made an admin for the restart so it can teleport to
Frostfang, which no shipped road reaches. Live bugs it found, each fixed with
a regression test: (1) **a journey that ended in a room that springs random
encounters froze the whole server**: `moveAndFinishLocked` held the expedition
lock while the encounter roll asked the expedition module whether the leader
could move (the arrival is now queued and heard after the lock is released;
no shipped room was both a journey end and an encounter room, so only this
run met it); (2) **a new character skipping the tutorial woke at 11/59 HP**
(the engine seeds 10 health under the archetype's raised maximum; creation
now starts at full health); (3) `company status` kept a companion's old level
after it levelled in a fight (it showed the last save's snapshot); (4) "Your
company gathers 3 firewood bundle" (now plural); (5) the phase 44 open item,
"The battle is under way" lingering after a fight's summary: an aim at a foe
already slain counted as a battle until the next round cleared it, so
`loot`, `go`, `eat` and the rest were refused (it now does not). The fifth
fix is read-only (the two reverted attempts cleared the aim); the world smoke
loot step logs how long it waited, and the tutorial run keeps `doAfterBattle`
as a safety net. Not fixed, noted for later: Dunmar has a market but no
merchant or smith, and no road joins it to Frostfang, so gear a Dunmar
company loots can only be sold or salvaged after a trip no shipped route
makes; the only two camp rooms (the tutorial campground and the Fork) have free
deadfall, so gathered firewood bundles have no use at a camp in the shipped
world (a camp elsewhere would burn one) and markets never buy them; `weather`
in Dunmar and Frostfang (city biomes) says "You can't tell what the weather
is like here" because only the forest biome has a weather table (a content
gap, not a code bug); a camp rest was not played in the world run, because the
Fork's 15% camp-raid roll would make it depend on dice (the tutorial run covers
a rest). One observation not reproduced: the prompt briefly showed
"Overloaded" with 12.9 of 30 kg carried after `camp break` and a step.
Merging master (39b's Samurai) renumbered the creation menus and broke both
smoke runs; they now answer the race and archetype prompts by name.
Verification: `make generate`, `make validate`, `go test -race ./...`, `make
js-lint`, `make js-test`, `make smoke`, `make smoke-world` (all pass on the merged tree).

**Phase 44b reviewed and merged via [PR #55](https://github.com/Robinsond76/ashveil-gomud/pull/55) (2026-10-06, Opus review thread):**
checked the deadlock fix: the travel timer already completes on the event
loop, so queuing `journeyArrived` only moves the encounter roll past the
expedition lock; the encounter roll re-checks that the leader is still in the
arrival room, so a late arrival can't spring on someone who moved or logged
off. The smoke's world bending (start room, fixture encounter table, unarmed
brigands, admin role) only writes the test's temp copy and its overrides file.
`AimedAtMob` reads the mob map on the game loop like every other caller; the
`company status` refresh is the save path's own snapshot. No fixes needed.
Decisions: (1) the city-weather gap stays a follow-up, not a quick table,
because a city table would change journey weather (a Dunmar departure now
takes the forest's weather from its destination); (2) the firewood-at-camp
and Dunmar merchant/smith gaps go to world building 41, which places camps
and shops; (3) a camp rest stays out of the world run (the tutorial run covers
it). UI: the fixes are themselves the player-visible changes (HP, roster level,
plural, battle refusals); no help change needed.

**Phase 40h built: neutral and elite class art, status names (2026-10-06):** closes the art gap 40s5 deferred. `TestEveryBuiltClassHasArt` passes under `ASHVEIL_ART_STRICT=1` for the first time: ten built classes gain map (3 views, idle and walk) and battle idle art: Warlord (Mercenary line), Sweeper, Vanguard, Valkyrie (Halberdier), Kensai, Hatamoto, Ronin (Samurai) and Stormcaller, Mistweaver, Earthspeaker (Shaman). The three neutral lineages had no base figure at all (an unpromoted Halberdier, Samurai or Shaman, or a member whose class art was missing, drew a silhouette), so each gets one on the shared rig (`draw_halberdier`: kettle helm, breastplate, upright halberd; `draw_samurai`: banded lacquer cuirass, sode, kabuto with horns, katana; `draw_shaman`: hide mantle, bone mask, feathered totem staff), and `TestEveryLineageHasBaseArt` is strict, so a new lineage ships its base art. The roadmap's "18 advanced classes" are the 23 of 40s5 plus these ten, all code-drawn from the master palette; `make sprites` regenerates everything and the contact sheet gains the three lines. Follow-up folded in: battle-screen status marks were unlabeled hues; hovering or tapping a figure now names its statuses in the caption ("the first wolf, wounded, bleeding, stunned"; allies stay hidden, as their marks are), and `help battlescreen` says so and describes the neutral figures. Browser checks: `scripts/browser/class-art-check.mjs` (new: the sheets load over HTTP, neutral members draw their lineage key, class names show on hover) and a caption check in `battle-check.mjs`; screenshot `screens/40h-battle-neutral-classes.png`. Decisions (owner delegation): (1) the strict art test stays opt-in by default because 38c2/38c3 and the next neutral class are building in parallel and would otherwise fail on merge; the art pass runs it strict; (2) elites other than Warlord are still `Planned`, so they stay unpainted until built (they fall back to their lineage's art); (3) status marks stay as dots with names on hover rather than 16 px icons, which are larger than a figure's health bar at the screen's 320x180 scale. Follow-up: the S3 status icons (`ui/status/*.png`) still aren't drawn on the battle screen; a dot legend or icon strip in the caption area would help a first-time player.

**Phase 40h review (2026-10-06):** art judged consistent with the 40s5 lines (same rig, palette and outline; each of the ten classes reads distinct at map and battle scale). Captions checked: status names are words only, allies' statuses stay hidden, unseen foes never gain statuses, so no ally numbers leak. Accepted fix: the battle screen's idle hint now says hovering shows statuses too. No findings rejected. Built classes still without art after this merge (fall back to lineage art until the art pass): Doll Master (39d, PR #59), the six 38c2 and six 38c3 elites (PRs #60, #62) and Gryphon Rider (39f, PR #64) once they merge.

**Doll Master art completion (2026-10-06):** the 37d review drew the Doll Master base figure (kept; one registration). This adds the three built advanced classes Puppeteer, Golemancer and Marionettist (recolors with accessories on that base) and the `doll` battle unit (painted wooden puppet on swaying strings; the battle screen already used the `doll` sprite key and drew a silhouette). `scripts/browser/doll-art-check.mjs` checks the sheets load, lineage and doll keys, and hover captions; screenshot `screens/39d-doll-master-art.png`. `ASHVEIL_ART_STRICT=1` now passes for every Doll Master class; (superseded: the elite and route art pass drew these) it listed the 38c2 elites (Pathfinder, Swordmaster, Nightblade, Sentinel, Marksman, Ravager) and the 39f Gryphon Knight, Skyscout and Wyvern Rider for the art pass.

**Phase 39h review (Opus review thread, PR #80):** merged master (39g Alchemist, elite art) and kept both classes everywhere they collided; sprites regenerated with `make sprites`. Balance: the "bow Ranger" cell carried the Ranger's starting sling (1d4, shoots every other round), so its 49/79/53 was a sim artifact; with a shortbow at 100 fights a cell, Warrior 75/90/73, Ranger 83/89/77, Arbalist 87/80/78 at levels 5/10/20, inside the design's 5 points of the Ranger at 5 and 20 (-9 at 10, inside run-to-run noise), so no lever was pulled and Armor-breaker stays at level 8. Routes at 15/25: base 77/73, Siegebreaker 86/80, Sharpshooter 70/78, Warden 75/75; the Warden reading level with base is accepted (one swapped member, and its pavise covers one row). Checked: the winding comes only from Piercing Bolt, a crossbow in other hands shoots plainly, every route signature fires in the real round, Piercing Bolt is level 1 so nothing is hidden early, ranks use the shared "New rank" line, mob 235 collides with nothing. Fixed: a retarget after a kill (`chooseFromParty`) gave foes no armor, so the `armored` rule fell back to the weakest (`TestChooseFromPartyArmoredAimsAtTheMostArmored`); the Warden's role text said "the row behind it" where the pavise covers its own row; the battle feed's winding event read "Reload" and now reads "Winding the crossbow". UI checked: the battle feed names the bolt and the winding, the strategy list shows `armored` from data, and the level-up report sizes the bolt. Also: `modules/company` under `-race` now runs past Go's 10-minute default (620s after 38d merged), so `make test` passes `-timeout 30m`; run `go test -race -timeout 30m ./...` directly. Follow-ups: a winding marker on the battle unit; speed up the `modules/company` suite.

**Elite and route art pass (2026-10-06):** `ASHVEIL_ART_STRICT=1 TestEveryBuiltClassHasArt` listed fifteen classes on master: the 38c2 elites (Pathfinder, Swordmaster, Nightblade, Sentinel, Marksman, Ravager), the 38c3 elites (Archon, Archmage, Necromancer, Wise One, Coven Mother, Crone of Ash) and the 39f routes (Gryphon Knight, Skyscout, Wyvern Rider). Each now has map (3 views, idle and walk) and battle idle art in `scripts/sprites/promoted.py`: its advanced parent's recolor made grander (longer cape with trim, taller crest or hat, pauldrons, halo or glowing staff stone; evil routes darker with ember touches). The plain Gryphon Rider base gained a pale-tipped feather mantle. Strict art now passes for every built class. Decision: the test stays opt-in, because 39e Beast Tamer routes, 39g Alchemist, 39h and the 38d elites are still building in parallel and a default-strict test would fail their PRs on merge; each of those lands art in a later pass. Checks: new `scripts/browser/elite-art-check.mjs` (sheets load, own class key, hover names) plus the class, doll and battle checks; screenshots `screens/48-elite-art-battle-*.png`, contact sheet `art/48-elite-routes-contact-sheet.png`.

**Elite and route art review (2026-10-06, PR #75):** judged against the advanced-to-elite comparison sheet and battle screenshots. Each elite reads as its parent made grander (cape, taller crest or hat, halo or lit staff stone, evil routes darker with ember) and stays legible at battle and map scale on the 40h rig and palette. `make sprites` reproduces the committed sheets byte for byte. UI check: the hover caption, company card and battle caption already name the class, so no UI change is needed. No findings accepted or rejected. Follow-up: the 39e, 39g, 39h and 38d classes get art when they merge; make the strict art test default once nothing is building.

**Phase 39c complete, merged via [PR #54](https://github.com/Robinsond76/ashveil-gomud/pull/54) (2026-10-06): the Shaman neutral lineage.** A weather-caller: Call Fog and Gust at level 1, Chill Wind 3, Rain 6, Lightning 8. One battle-local weather at a time, 3 rounds, a new call replaces the old: Fog (Fogbound foes: ranged -10 to hit, spells -10%), Chill Wind (Windchilled: chants and sling shots one round slower), Rain (Lightning +50%). Routes Stormcaller (chain lightning at 50%), Mistweaver (+5 Evasion to allies in fog, longer weather), Earthspeaker (Stoneskin: +10/15/20 armor); elites planned (39i). Long Weather talent, recruit mob 150, `help shaman` and `help shaman-routes`, creation-lesson tutorial hint, strategy uses `weather` and `storm`. Plan: [39c](plans/2026-10-06-phase-39c-shaman.md). Decisions: Caster role, not a new support role; Fog and Chill are buffs so `conditions` shows them; weather marks only foes standing at the call; a spell with an apostrophe in its script text silently disabled the spell (fixed, `node --check` each spell script). Merged master (39a Halberdier) before the PR; powers are learned spells, so the strategy list, company summary and capability panel show them only once known, and route ranks reuse 39b's "New rank" level-up line. Follow-ups: Shaman battle sprites (art pass), elite ranks (39i).

**Phase 39g complete, merged via [PR #74](https://github.com/Robinsond76/ashveil-gomud/pull/74) (2026-10-06): the Alchemist neutral lineage.** a flask-thrower with no mana. Healing Draught 1, Antidote 3 (cures poison and bleeding), Fire Flask 6 (two foes, 70% of Sparks), Bracing Tonic 8; each costs one flask from a satchel (6 + level/3), refilled by `brew` or a camp rest from reagents (item 95, Dunmar). Routes Apothecary, Bombardier, Mutagenist; elites planned (39i). Decisions: flask is a spell cost, one satchel count (`FlasksSpent`), mutagen is armor for HP with no damage, Smoke Flask and Elixir deferred; mob 190, item 95. Balance (100 fights): within ~5 points of the Cleric, routes inside the noise except Apothecary at 25 (-8, untuned). Help, tutorial, GMCP, web panels and art shipped. Details: [plan](plans/2026-10-06-phase-39g-alchemist.md).

**Phase 39g review (Opus review thread):** kept the build's decisions. Economy: reagents (item 95) are supply-only in Dunmar like doll parts, and flasks are a count, not items, so nothing brewed or bought resells or salvages for profit (the three starter reagents fall under the accepted starter-kit exception). Antidote has a real use today: enemy slashing, clawing and stabbing crits leave bleeding on the company, which it cures; poison stays dormant until enemies poison (43a's camp antidote stays planned). Apothecary: re-measured at 200 fights a cell at level 25 (base 19-20%, Apothecary 23%, Bombardier 22%, Mutagenist 27%), so the -8 at 100 fights was noise and no lift was made. Neutral-class lessons hold: flasks are learned spells (hidden until their level), ranks use the shared "New rank" line, every route signature fires in the real round (`wiring_alchemist_test.go`), mob 190 and item 95 collide with nothing on master or open PRs. Fixed: `brew` and camp brewing left a satchel that shrank (a talent or rank change) short of full (`TestBrewFillsAShrunkSatchel`); Pitch and tar said 3 burning rounds where burning lasts 4 (`help statuses`); the tutorial's class sentence had a doubled "and". UI checked: status, company card, Combat tab, battle tap text, GMCP and capabilities all show the satchel. Follow-ups: a battle-log line when an Alchemist's satchel runs dry mid-fight, and a reagents count on the Camp tab.

**Phase 39f complete, merged via [PR #64](https://github.com/Robinsond76/ashveil-gomud/pull/64) (2026-10-06): the Gryphon Rider neutral lineage.** A sky skirmisher: Dive at level 1 (strikes a foe in lateral range past the front row, 3-round cooldown, costs the rider 10 Evasion for the round), Talons 3 (dive wounds bleed), Power dive 8 (+25% damage). Routes Gryphon Knight (Lance charge knockdown, barding armor, Falling stone +50%, Steady wings), Skyscout (Eagle eye and Far sight Perception that moves ambush detection, Quick stoop, Hawk's mark exposes), Wyvern Rider (Venom stoop poison, hide, tail, guile); elites planned (39i). Wing Drill talent, recruit mob 175, skill `skirmish`, `help gryphon-rider` and `help gryphon-rider-routes`, creation-lesson tutorial hint, strategy ability `dive`. Plan: [39f](plans/2026-10-06-phase-39f-gryphon-rider.md). Decisions (autonomy): no stabled-gryphon mount record yet (battles need only the rider's class and ground; help says the gryphon is the rider's own); Lance charge uses the two-handed war spear because no lance item family exists; fighter role with the `healers` default rule, not a new role; Dive at any foe in lateral range; Skyscout uses Perception for ambush detection instead of javelins and leaves "act first in an ambush" to the elite; Venom stoop is buff 13 and skips poison-immune creatures. Balance at 100 fights a cell: win rate at levels 5/10/20 Warrior 79/94/72, Ranger 77/89/69, Gryphon Rider 83/89/75; routes at 15 base 76, Knight 82, Skyscout 88, Wyvern 83; at 25 81/84/84/89; no tuning needed. Follow-ups: gryphon mount record (32f slot, flying, double feed), a lance item family, level-up report line for Dive, elite ranks and the Skyscout capstone (39i), sprite art (art pass).

**Phase 39f review (Opus review thread):** checked the build's decisions and kept them all: no gryphon mount record (battles need only the rider's class and ground; the help says the gryphon is the rider's own), the war spear as the lance, the fighter role with the `healers` rule, and Perception for the Skyscout (it moves ambush detection, javelins would only repeat the Ranger). Neutral-class lessons hold: Dive is a level 1 power and Talons/Power dive are ranks, so nothing shows before its level; ranks use the shared "New rank" line; every route signature fires in the real round (`TestRoutesShapeTheDive`: knockdown with a war spear only, exposed at 20, poison except on immune foes, no Evasion cost at 25); Dive shows on the battle screen through the ability event and its statuses through the usual marks; help claims match the code (route ranks replace earlier ones, so Falling stone and Far sight are totals, as the help says). Mob 175 collides with nothing on master or open branches. Accepted and fixed: the level-up report named no Dive growth; it now shows Dive's damage ("Dive 100% of a blow -> 125% of a blow" at Power dive and Falling stone, `TestPowerSnapshotNamesTheGryphonRidersPowerDive`). Rejected: capping Dive's cooldown at 2 (a Skyscout with Wing Drill dives every round at level 15, but it pays 10 Evasion each time and the route sat at 84-88%, inside the band). Merged master (39d Doll Master, 40h art, 38c2 elites): 40h's strict `TestEveryLineageHasBaseArt` needs base art for every lineage, and master was already red on the Doll Master, so the review drew a plain Gryphon Rider base figure (riding jack, winged helm, spear) in `scripts/sprites/figures.py` (`make sprites`) for the map and battle screen; routes use it until the art pass. The Doll Master's art is a separate thread's fix, so that test stays red for the Doll Master only until it lands. Not this PR's: `TestWarlordRelentlessQuickensItWhenItsFoeStandsUp` (38c1) is flaky on master too (1 to 4 failures in 20 runs). Follow-up for 39i: the planned Gryphon Lord's "Dive every other round" is already matched or beaten by Skyscout plus Wing Drill, so the elite needs a different signature.

**Phase 39c review (Opus review thread):** accepted and fixed: (1) Rain promised "fire damage halved on both sides" in its spell text, cast line and `help shaman`, but nothing in the game deals fire damage (Burning is never applied), so the claim is gone and the dead `FireDamage` helper removed; Rain now only feeds Lightning. (2) UI: the player could not see which weather was up or for how long; `Company.Battle` now carries `weather` {kind, name, rounds, effect}, the battle screen's header names it ("fog (3 rounds: foe ranged attacks and spells weaker)") and the Combat tab's Battle view adds a Weather note; `help battlescreen` and `help shaman` say so (`TestBattleFeedCarriesTheWeather`, `scripts/browser/battle-check.mjs`). (3) Balance was too strong at 100 fights a cell: Shaman 62/69/33% against Wizard 28/40/29% and Witch 44/41/22% at levels 5/10/20. Experiments showed the weather itself is worth little in the mirror (a Shaman that never calls weather still won 55% at level 5); the edge was its sturdier body and focused single-target Gust, and the Wizard underperforms because it chants Shower of Sparks at a group instead of Magic Missile. Tuned: health and Evasion to the Wizard's (no head start, 0.5 a level, Evasion 0.75), Wizard-like growth (mysticism 4, smarts 3, perception 2, speed 1), Gust base 6 to 5 (about 70% of Magic Missile), Lightning base 10 to 9, Chill Wind cost 8 to 12. Result at 150 fights: Shaman 44/52/32%, within 5 points of the Witch at level 5 and of the Wizard at 20; level 10 stays about 10 points above both (Rain's Lightning bonus at 30% instead of 50% made no measurable difference, so the design's 50% stays). Routes after tuning (80 fights a cell, base/Stormcaller/Mistweaver/Earthspeaker): level 15 38/32/27/36%, level 25 16/23/21/30%; at 15 the routes sit within noise of the base class (Mistweaver lowest), at 25 all beat it; left for the 39i balance pass with the elites. Rejected: the Mistweaver's fog Evasion covering all blows, not only melee as the design said (simpler and visible; kept). Mob 150 and buffs 1112/1113 collide with nothing on master or open branches at merge time. Follow-ups: the Wizard picks Shower of Sparks over Magic Missile against groups and trails the other casters (balance candidate); Earthspeaker's Stoneskin is not listed in `strategy` spell lists for casters (minor); Mistweaver trails at level 15 (39i balance pass); Shaman sprites (art pass) and elite ranks (39i).

**Phase 45 reviewed and merged via [PR #53](https://github.com/Robinsond76/ashveil-gomud/pull/53) (2026-10-06, Opus review thread):** checked that watching an ally full size draws only what the feed already carries for allies (health bands, chant mark; no numbers, statuses or role letters, since the server never sends them), that the tap only swaps the view (inputs stay retreat and company focus), and that the prompt's progress text rides the existing `{activity}` token (refreshed per round like travel, no extra lines). Accepted and fixed: (1) typing `status` cancelled the gather, though the help and tutorial pointed players at `status` to watch it; the bare sheet and its aliases now keep the work (`status train` still stops it), help and the start message updated (`TestTypedCommandsCancelTheWork`); (2) `Room.Gather` result lines carried terminal colour tags (`<ansi fg="itemname">`) that the Room Info panel would print raw; the GMCP handler strips them (`TestRoomGatherCarriesTheWorksProgress`); (3) companion rows on the status sheet did not line up with `Members:`; labels are padded. Merged master (38c1 elites): 38c1 added its own promoted-class display (a `Class` row on the sheet and `route`/`tier`/`rank` in `Char.Info`, drawn in the Character window), so 45's duplicate (Path row as "Knight (Warrior)", `Char.Info.class` renamed with `lineage_name` and a hover) was dropped in favour of 38c1's; the Path row is the lineage again and the companion roster rows stay. Follow-ups: a client that reconnects mid-gather gets no bar until the next start (no resend of `Room.Gather` on login).

**Phase 45 built: UI follow-ups (2026-10-06):** closes three gaps the 40s5, 40a2
and 40g2 reviews listed. (1) **Class everywhere.** `status` (the sheet `score`
aliases) names the leader's promoted class with its lineage ("Knight
(Warrior)") on the Path row and lists each companion on the Company panel
("Oswin: Priest (Cleric), Lv 6", fallen marked); `Member.RankName` in
`companyview` is the one formatter. `Char.Info.class` is now the promoted
class's name once promoted, with the lineage in `lineage_name`, which the web
Character window shows on hover. The company roster and Company window
already named classes (38b, 40s5). (2) **Gather progress.** The gathering
module queues `events.GatherProgress` on start, finish and stop; the GMCP
room module sends it as `Room.Gather` (phase, kind, label, seconds, lines)
and the Room Info window shows a strip: a bar filled over the work's length
with the seconds left, then the result lines (15 s) or "Work stopped".
In text, `gathering.ProgressOf` (a provider seam like camping's) feeds
`companyview.Activity` kind Gathering, so the prompt and the status
Doing row read "Gathering herbs 40%, 12s left". (3) **Battle screen.** Tap an
allied formation or its pennant to watch it full size (it swaps with the
player's company, which shrinks into the ally's place as "your band"); tap the
small band to return; the view returns itself when that company leaves.
View only: inputs are still retreat and company focus. An ally's blow on a
foe the player is not fighting (or by an undrawn third company) is dropped
from the picture and the last-blow line rather than landing on nothing.
Help: `gathering`, `status`, `battlescreen` updated; the gather and party
tutorial hints point at the new displays. Tests: `modules/gathering`
(announcements and provider), `companyview`, `usercommands` status sheet and
help, `modules/gmcp` (Char.Info, Room.Gather), `scripts/browser/room-check.mjs`
(Character and Room windows) and new sections of `battle-check.mjs`.
Screenshots `screens/45-*.png`. Decisions (delegated): (a) the gather strip
lives in the Room Info window, since the work is tied to the room and Room
Info is always docked, not in the Camp tab or map (both off limits here); (b)
the Doing row and prompt use a new activity kind rather than a new row, so
every surface shows it; (c) watching is a pure view swap with no new server
state; (d) dropping unseen-ally blows beats inventing a figure, because the
feed already hides ally numbers; (e) the Path row keeps its label and shows
the class with lineage in brackets rather than adding a Class row.

**Phase 39a complete, merged via [PR #44](https://github.com/Robinsond76/ashveil-gomud/pull/44) (2026-10-06): the Halberdier.** the first neutral class,
a polearm fighter that wins by crowding. Sweep (a whole-turn blow at 90% on
the foe and its row neighbour, the whole row from level 8), Brace (answers
the first strike at 125%), Hook (knocks a leaping foe down), a `crowded`
target rule, Sweeper, Vanguard and Valkyrie routes open at any alignment
(elites planned for 39i), a Sweep Drill talent, a recruitable Halberdier
(mob 130), `help halberdier` and `help halberdier-routes`. See the
[plan](plans/2026-10-06-phase-39a-halberdier.md). Decisions: Sweep 90% after
the balance run showed 80% lost five-foe groups; sprite deferred to 40s5.
Review (2026-10-06): accepted: the strategy, company and GMCP displays
listed Brace from level 1 though it comes at 3 (new `strategy.AtLevel`
filters displays and battle alike; GMCP shows it disabled, "Comes at level
3"); the level-up report now names Brace, Hook and the whole-row Sweep as
they arrive; help said Brace waits only on Sweep's cooldown (it also braces
when Sweep has no second foe); stale 80% comments. Checked and fine: mob
130 is clear of master (max 97) and 39b (139); brace state clears with the
fight's class state; no economy surface. Balance re-run (40 fights a cell):
the Halberdier is within 5 points of the Warrior everywhere but level 20
five-foe groups (90% vs 72%), its intended crowd strength; Sweep stays 90%.
After the 40a3/40b/40s5 master merge, one race run failed
`TestAimedShotGrowsWithLevel` (35b's ranger test, weapon dice: level 1 rolled
21); it passed 8 of 8 reruns and is left for the flaky-test phase (37c).
Merged after 39b: both neutral lineages share the help tables, recruit
lists and `DefaultRule` (Samurai strongest, Halberdier crowded).

**Phase 43b reviewed and merged via [PR #57](https://github.com/Robinsond76/ashveil-gomud/pull/57) (2026-10-06, Opus review thread):** builder decisions (1)-(6) kept: the shop-only launch needs nothing from 43a; dropping the antidote as a combat action matches the no-mid-battle-input rule (poison clears at fight end, and `curepoison` and cleansing exist); the camping registry is the same persistence path as `auto_sharpen`. Checked: the 10 real-minute coating uses an absolute Unix expiry, the same real-time model as the inn's Rested buffs, and never advances game time; poison is delivered automatically by the strike loop with no player input; combat-round durations match the existing convention (non-damage statuses carry rounds+1 ticks, as Blighted); vials are `SupplyOnly`, so nothing resells. Balance (level 8 company, 2-3 foes, 120 fights a cell, every member's blade coated): no poison 88% at 10-12 and 35% at 13-15; with poisons 90-92% and 29-39%, inside sampling noise, so the difficulty bands hold. Added: `TestCoatedBladePoisonsAFoeThroughTheRealRound` (DoCombat, Buff event, ApplyBuffs: the foe is poisoned, the blow names it, it ticks the next round), the only integration point without a real-round test. The new combat tests shift the package's dice order so `TestIaijutsuIsTheFirstStrikeOfABattleOnly` (the known Iaijutsu crit flake) failed every package run; ported 37c's fix unchanged (`util.UseRandForTest`, top-face rolls in that test), which no-ops when 37c merges. `TestSpawnLootRollsAnItemIntoTheRoom` failed once in the final race run and passed three package reruns; it is pre-existing on master and untouched by 37c, so it stays a follow-up. UI: coatings show in inventory, `conditions`, `coat status` and the gear window; a poisoned foe gets a status dot on the battle screen and the hit and tick lines name the poison; no change needed. Rejected: none. Follow-ups: the battle screen's status dots are unlabeled hues (true of every status, a candidate for a later UI pass).

**Phase 43b built (2026-10-06): weapon poisons, shop-only launch (review pending).** The roadmap lists 43b after 43a, but the poison design's launch is explicitly shop-only (crafting, `camp brew` and the antidote draught are the later slice), so the launch needs nothing from 43a and was built on master directly. Four poisons (`items.Poisons`): bitterleaf (1 damage a round, 3 rounds), leechbane (healing received -25% rounded down, min 1), leadroot (physical damage dealt -15% rounded down, min 1), mirethorn (dodge -10 points); each is a one-dose vial (items 280-283, markets sell all four in Dunmar and bitterleaf on the Old Kings Road, `SupplyOnly` so they never resell). `coat <poison> [self|member] [main|off]`, `coat status`, `coat clear` and `camp coat` put a dose on one bladed weapon outside fights, travel and rests; `camp poison [assign|unassign|preview|apply]` keeps a per-leader preparation list (persisted in the camping registry as `poison_plans`) and applies it all-or-nothing, listing every blocker. A coating is item state (`CoatKind`, absolute `CoatExpires`, `CoatContacts`): 10 real minutes or 8 wounding blows, expiry checked at every read, never refreshed or replaced. In the strike loop (`internal/combat/poison.go`) a blow whose final damage is positive spends one contact (`AttackResult.PoisonSpent`, charged to the real weapon beside the edge by all four attack entry points) and rolls 40% (resistant 20%, immune 0, mob templates `poison:`) to queue the poison's combat status through `BuffTarget`; a victim carries one at a time (a contact against an already-poisoned victim spends but skips the roll). The statuses are ordinary combat statuses (buffs 1120-1123, flags `poison`, `weapon-poison`), so ticking, expiry at fight end, `curepoison` (flag) and cleric cleansing (`status.CleanseOne`) all work with no new machinery. Marked immune: the dead, constructs and practice dummies (11 templates); resistant: the ent and sentient fungus. UI: the coating shows in inventory and equipment lines (`EdgeLabel` now includes it), `conditions` (Weapon edges group), the gear window stats rows (`weapon_coat`, `offhand_coat`), and the hit and tick lines name the poison. Help: new `help poisons` (aliases poison, coat, bitterleaf...), linked from `help combat`, `help camp` and `help statuses`; tutorial Camp lesson gains a hint. Decisions (builder, owner delegation): (1) built the shop-only launch without waiting for 43a; the antidote draught (prevention) and `camp brew` stay with 43a and a later crafting slice; (2) the design's shop antidote as a combat action is **dropped**: mid-battle input is not allowed (Ogre Battle vision), poison clears at fight end anyway, and the cure already exists as `curepoison` and cleric cleansing; (3) assignments are stored in the camping registry beside `auto_sharpen` rather than on the company record, for the same persistence path and purge; rows for a member who left the roster are ignored; (4) vials are drawn from company supplies (cargo first, then packs) through the existing `itemCount`/`spendItem` seams; (5) bitterleaf costs 6 (a bandage), the other three 12, on the road bitterleaf 8, all `SupplyOnly`; (6) the web battle screen needs no change (statuses already draw as dots); the gear window gains two rows. Gate notes: `TestIaijutsuIsTheFirstStrikeOfABattleOnly` and `TestSpawnLootRollsAnItemIntoTheRoom` each failed once in the full race run and pass on rerun (the first fails 5 in 40 solo runs on master, a crit roll; the second 0 in 15 solo, on master too); both are left for 37c. Full gates otherwise green (`make generate`, `make validate`, `go test -race ./...`, `make js-lint`). Screenshot: `/mnt/project-files/screens/43b-camp-poison.png` (the real `camp poison` and `coat status` text).

**Phase 40a4 reviewed and merged via [PR #50](https://github.com/Robinsond76/ashveil-gomud/pull/50) (2026-10-06, Opus review thread):** builder decisions (1)-(7) kept as reasoned below; (8) changed. Accepted and fixed: (a) breaking camp, or resting again, between a rest's end and the next round dodged its thieves (decision 8 only favoured a player who knew the trick), so `settleTheft` completes a due rest and resolves its theft first in `camp break` and `camp rest` (regression `TestBreakingCampOrRestingAgainDoesNotDodgeThieves`); (b) fairness/UI check: the only warning was help and a tutorial hint, so starting a rest on a road thieves work without bells now says so, and `Company.Camp.theft_risk` drives a Camp tab line "Thieves work this road" (the old "no gear" line, shown even in safe zones, no longer mentions thieves) (`TestThievesAreWarnedOfAtRestStartAndOnTheCampTab`, browser check); (c) `help camp gear` said "no warning" and "the rest report names what is missing"; reworded to match. Browser check `dock-windows-check.mjs` ran to the end (260 checks). Rejected: none. Follow-ups: a watch is only counted if the leader is still at the camp when the theft resolves (it resolves within a round of the rest, so left as is).

**Phase 40a4 built (2026-10-06): camp theft.** A camp rest without camp bells and trip lines may draw thieves. `RestSession.Theft` is rolled when the rest starts (never when bells are strung; a zone needs a `CampTheft` entry, shipped: Old Kings Road 20%) and saved with it. Once the rest is done and the leader is online and out of battle, `resolveCampTheft` (game loop, `modules/camping/theft.go`) marks it done and saves **before** taking anything, so a restart can never rob a rest twice, then `company.CampTheft` removes about 10% (`TheftSharePct`) of the loose goods, at least one item and at most 4 (`TheftMaxItems`), from the cargo and the companions' packs, and the leader reads "When you wake, the packs have been rifled. Missing: ..." with a pointer to `help camp gear`. Never taken: equipped gear, the leader's own pack, the treasury and gold, quest-token items, keys, and camp gear (items 45-50). A posted watch gets its raid-spot chance (25% a level) to catch them: "nothing is missing", nothing taken. Folded in from the 40a3 review: the web Camp tab now lists the camp gear the company carries (GMCP `Company.Camp.gear`, one label per piece, e.g. "Bedrolls 2/3", "Tent"; with none it hints that bells keep thieves out; `/mnt/project-files/screens/40a4-camp-gear.png`). `camp status` bells line says thieves keep out. Help: `help camp gear` gains a Thieves section (aliases thieves, thief, theft, camp theft, stolen) and its stale "nothing is stolen" line is gone; `help camp` and `help campwatch` updated; the tutorial's Camp gear hint mentions that bells keep thieves off. Tests: rolled and saved at rest start (and never with bells or in an unlisted zone), resolved once after the rest with the done flag saved first, reload does not rob twice, offline or in-battle leaders wait, watch catches, empty-handed thieves, settings parse, and the real company provider (cargo plus companion packs, spares quest tokens, keys, protected gear and the leader's pack; share and cap), GMCP gear payload, help render, browser check. Decisions (builder, owner delegation): (1) thieves roll once per rest at start, like raiders, and resolve after the rest, so the report is "noticed on waking"; (2) the leader's own pack is safe (the leader is the one asleep with it), while the cargo and companions' packs are "unattended"; (3) camp gear is never stolen, so a company never loses its bells or tent to the thing they guard against; (4) the chance is zone-based and only configured for Old Kings Road (tutorial and safe-zone camps are never robbed), 20% against raids' 15%; (5) a posted watch protects with its raid-spot chance; (6) stolen goods are gone, no tracking (economy rule: nothing gathered or bought can be reclaimed for profit); (7) a leader offline when the rest ends is robbed on return, so logging out does not dodge it (as raids); (8) a rest started before the theft resolves loses it, which only favours the player. Follow-ups: none needed; thief mobs or tracking stolen goods could be a later quest hook.

**Phase 38c3 reviewed and merged via [PR #62](https://github.com/Robinsond76/ashveil-gomud/pull/62) (2026-10-06, Opus review thread):** builder decisions kept except where changed below. Independent reviewer findings, each verified. Accepted and fixed: (1) the Crone's rank 40 "Dread everywhere" was never read by any code; replaced by **Quick curses** (every other hex chants a round less, `HexQuick`), a real effect that also carries the Crone's balance (`TestCroneQuickCursesChantShorter`); (2) widening "hexed" to Poisoned/KnockedDown/Hobbled/Exposed let a Warrior's Tackle, a weapon poison or Grave Chill trigger the Hag's and Crone's curse and Crone's Doom; those four now count only when a hex laid them this battle (`ClassRT.HexBuffs`, `TestSharedStatusesCountAsAHexOnlyWhenAHexLaidThem`); (3) a foe falling in another company's battle in the same room could be raised or harvested; `eliteFall` now checks `b.Has` (`TestNecromancerIgnoresAFoeOutsideItsBattle`); (4) Barrage's second target skipped the room and withdrawn checks; (5) a raise that failed after taking the fallen foe lost it; it is put back; (6) Soul Rot fired on foes no longer hexed; (7) a later witch's hex overwrote the Crone's claim on Crone's Doom; (8) dead code (`Countered`, empty block) and stale comments (Mend Charm, Hearthward, `raisefallen.yaml` "half its health"). **Ashen Curse now spreads** (`CurseDmg`): foes under the Crone's hexes take 10% more from every ally's blows (40%/55% from her own), since a casting witch rarely swings and her own-blow bonus did nothing (`TestAshenCurseRaisesEveryAllysBlows`). Rejected or deferred: Overchannel, Barrage, Quick casting skip a player's manual `cast` (manual casting is out-of-combat utility by the combat vision, as for 38b's chant trims); Dread does not count toward Twin Hex; Miasma's Lasting/Lingering timing is in poison game-round ticks; Ward of Life and Lich's Bargain stop blows, not spell or poison ticks (the ranks say "a blow"). Balance (100 fights a cell, even mirror, win%): chant speed dominates this sim (a full one-round trim doubles cast rate and was worth +30 points), so the Archmage's Quick casting and the Crone's new rank trim every other chant. Changes: Coven's Will loses its extra chant trim; Archon Counterspell cost 20 (15 from 50, replacing Triple ward), Mana Shield 10%; Archmage ranks reordered (35 Quick casting every other spell, 45 Barrage); Necromancer Deeper drain also sets spells to +35% damage; Crone Ashen Curse +5 Attack. Results L40: Archon 23 vs Theurgist 9; Archmage 21 vs Arcanist 6; Necromancer 18 vs Warlock 13 (2-13 across runs); Wise One 41-48 vs Hedge Witch 38-44; Coven Mother 34 vs Coven Sage 20; Crone 18 vs Hag 5. L50: Archon 37 vs 12, Archmage 30 vs 13, Necromancer 18-20 vs 10, Wise One 76 vs 50, Coven Mother 43 vs 27, Crone 28 vs 13. **Accepted misses:** Wise One trails the +10 target at L40 (every lever tried overshot by 20+, and it is +26 at L50); Archon and Wise One overshoot at L50. UI: help pages for the changed ranks updated; class, company, GMCP and web panels are generic and needed nothing. Follow-ups: the whole wizard advanced tier wins under 15% in this mirror, a candidate for 39i's balance pass.

**Phase 38c3 built: wizard and witch elites (2026-10-06):** Archon, Archmage, Necromancer (with its thrall), Wise One, Coven Mother and Crone of Ash, following the 38c1 pattern (promotion at 30, catch-up ranks, elite talents at 35/45/55, `help` pages, tutorial pointer). Every signature fires through real combat (`modules/company/wiring_elite_wizard_test.go`, `wiring_elite_witch_test.go`; `help archon|archmage|necromancer|thrall|wise-one|coven-mother|crone-of-ash|wizard-routes|witch-routes`). Decisions, each with its reason:

- **Counterspell is a held turn** (heldTurns/tempoBlocked), so an Archon that counters can't also act that round; this keeps it from out-acting its own tempo.
- **Life Drain (Siphon) moved to Warlock rank 10**, so the Necromancer inherits a spell its lineage already owns.
- **Hexed status uses `status.Live`; Poisoned (13) is a hex status**, so the Crone's Ashen Curse and Soul Rot see poison as a hex.
- **Thrall health is `SummonInfo.HealthPct`** applied in `RecalculateStats` (60%, 85% from 45) instead of a fixed half, so the Necromancer's raise scales with its ranks.
- **Overchannel and the Storm** are "swell a cast already coming" effects (never a separate action), skipped when mana would run short.
- **Tuning (timeboxed, two passes):** Overchannel every 4 (3 at 50), Thrifty casting 15%, Mana Shield 15%, Raise 60%/85%, Drain 150%, Hearthward wards three (four at 55, cap 200), Mend Charm half a Minor Heal, Coven's Will +5 (+5 more at 55), Ashen Curse 40% and +8 Attack (55% at 50).

Balance (L40, 40 fights, two lineage members per company, win%): Archon 12 vs Theurgist 10; Archmage 35 vs Arcanist 17; Necromancer 7 vs Warlock 2; Wise One 42 vs Hedge Witch 25; Coven Mother 57 vs Coven Sage 17; Crone 17 vs Hag 10. **Accepted misses:** every elite beats its twin, but Archon (+2), Necromancer and Crone are under the +10 target and Coven Mother overshoots (+40); samples are small and noisy and the targets are flexible. Follow-up: trim Coven Mother and lift Archon/Necromancer in a later balance pass. Elite art is deferred to the class-art pass.

UI check: class, company and level-up lines, GMCP and the web company panel are generic (`class`/`class_name`, `RankLines`), so they needed no change; promotion-ready lines name `class promote #2 <class>`.

**Phase 38c1 built: elite framework, UI and the warrior and cleric elites (2026-10-06):**
The promotion framework 38b shipped (level 30, gate wait, catch-up ranks) now
has its rules and UI for all eighteen elites; 38c2 and 38c3 only add routes.
Task 0 reconcile: Paladin, Dread Knight, Hierarch, Elder Druid and Demonologist
matched the faith routes design as shipped; the Warlord (Mercenary elite) was
the only elite left to build. Built:

- **Warlord** (30 Marked for Ruin, 35 Battle Cry, 40 Quicker tackle, 45 Sunder,
  50 Ruinous mark, 55 Relentless, 60 Warlord's Command) through real combat
  hooks (`internal/hooks/combat_warlord.go`): marks and cries feed
  `attackRating`, Relentless and the Command add action-meter points (never an
  extra turn); all runtime only.
- **Elite talents** at 35/45/55: `classes.EliteTalentsFor`, offered only to an
  elite class (warrior: Iron Hide, Second Wind, Veteran's Edge; cleric: Font of
  Grace, Radiant Healing, Unshaken). `CanPick` and `MenuFor` now take the
  level; the base lists are unchanged. 38c2/38c3 add rogue, ranger, wizard and
  witch lists with `offerElite`.
- **Rules/UI:** a clear refusal for base to elite ("take the Priest first, then
  the Hierarch in the same visit"); `classes.Describe`/`PromotionState`
  feed the company roster, the class preview (design format), `class`, level-up
  and companion level-up lines (`LevelNotes`), GMCP (`Company` members and
  `Char.Info` gain class, tier, rank, promotion), and the web company and
  character windows. The milestone schedule now lists the shipped elite ranks
  (`eliteShipped`; 38c2/38c3 flip theirs).
- **Help:** `help elite` and `help warlord`, updated promotion, classes,
  talents, warrior-routes, cleric-routes, alignment, warrior and combat pages,
  keywords and aliases, a tutorial hint (Departure lesson).

Decisions: (1) the Warlord's mark and Battle Cry add Attack (not damage) so
they stack with every role; (2) Relentless and the Command push meter points,
because a free turn would break tempo; (3) elite talents come from a second
list rather than replacing the lineage's five, so a level 35 pick is never a
trap; (4) a Grove is cast on three hurt allies, not two, and Grove is now 100%
and 130% (from 60% and 80%): the 3-round chant with the old numbers lost to
plain Rejuvenation in the boss mirror. Design text for those numbers is
superseded.

Balance (`TestPhase38bClassRoutes`, 30 fights a cell, 60 for the druid line;
wins / company HP lost): fighting healers vs Mercenary: L35 Warlord 80% / 49%
vs Mercenary 63% / 70%; L45 86% / 46% vs 66% / 58% (+17 and +20 wins);
Paladin 90/96% and Dread Knight 76/93% stay in the same band (the Paladin
and Dread Knight are within 5 points at L45). Summoners (boss mirror): Hierarch
100% vs Priest 53/46%; Demonologist 80/73% vs Blood Priest 43/46%. Missed:
the Elder Druid did not beat the Druid (27% vs 33% at L35 after tuning, 20% vs
33% at L45 on the old Grove). Its healing is not what a boss mirror rewards,
and Entangle is not used by the company strategy. Left as a follow-up
(candidate phase "Elder Druid tuning": let the strategy cast Entangle, then
re-measure; a healer elite's gain may belong in a heal-load cell rather than
a boss one).

Follow-ups for 38c2/38c3: elite talent lists for the other four lineages,
`eliteShipped` and GMCP `promotion` checks per route, route help pages.
Verification: `make generate`, `make validate`, `go test -race ./...`,
`make js-lint`; the Chromium dock-windows check passes (its pre-existing
battle-canvas hover timeout is unrelated). Independent review: Opus review
thread.

**Phase 38c1 reviewed (2026-10-06, PR #46, Opus review thread):** checked
the Warlord ranks, elite talent gating, catch-up ranks, runtime-only state
and the cleric elites (Hierarch, Elder Druid, Demonologist: promotion,
catch-up, Font of Grace through the command, help). Accepted and fixed:
(1) **the Elder Druid never cast Grove**: it waited for three allies below the
healing threshold, which a fight almost never shows, so it played as a Druid.
An idle Elder Druid now sows a Grove once two allies are below 85% without
Rejuvenation (`strategy.GroveBelow`), a hurt company needs two (`GroveHurt`),
and Grove chants 1 round (from 2) and blooms, healing half its share at once
(`TestElderDruidSowsAGroveEarly`; help and the rank text say so). Boss
mirror, 40 fights a cell: L35 Elder Druid 37% / 79% HP lost vs Druid 10% /
96%; L45 55% / 66% vs 40% / 83% (+27 and +15 wins). (2) **The level-up "promotion
ready" line named a command that failed** (`class promote` with no class);
it now names the class (`class promote #2 warlord`). (3) **The base-to-elite
refusal sent companions to the player's command**; the error no longer
carries a command and the refusal adds the subject's own (`class promote #1
mercenary`). (4) **`status` never showed the class**: the Character panel
gains Class (name and tier) and Promote (ready) rows (`TestStatusShowsTheClass`).
(5) A raw "that talent is for elite characters" error now reads like the
other refusals. (6) `TestGrantXPLevelReport/4` failed since 38c1 shipped the
level 5 talent (no "(coming)" left at level 4); the test now expects that, so
the build's green-suite claim above was wrong for that one test.
Rejected: the Warlord's Command counting members present rather than fallen
(a companion leaves a battle only by falling or with the whole company's
retreat, so a shrinking count is a fall); the always-"ready" promotion
preview (only reached after the check passes).
Master merge (40s5 class art, 40a3): GMCP `Company` members now follow
40s5's keys, `class` (id) and `class_name`, plus 38c1's `tier`, `rank` and
`promotion`; the company card shows the class name in its header and the
tier and rank beneath (dock-windows check passes in full). With 39b
(Samurai) merged, a level-up names its ranks once, through 39b's
`ClassRanks` ("New rank: ..."), and `ClassNotes`/`LevelNotes` carry only
the elite promotion line (`RankUpLines` removed); `help elite` lists the
Samurai elites as still to come.
Follow-ups: the Druid itself trails the Priest and even an unpromoted cleric
in the boss mirror (10-40% vs 37-47%), and Barkskin takes most idle turns;
a "Druid tuning" pass belongs with 39i or a small phase.

**Phase 37d reviewed and merged via [PR #67](https://github.com/Robinsond76/ashveil-gomud/pull/67) (2026-10-06, Opus review thread).** Review checked that each fix is real isolation, not masking: the event queue is cleared (listeners untouched), the world and data path are reset per test, and nothing was skipped or disabled. Agreed: the Warlord Relentless window (6 to 12 rounds) still asserts the Relentless line fires once the knocked-down foe stands; it only gives the foe more rounds to get up. Agreed: the stderr default logger is safe in production (`main.go` and `reset-admin-pw` still call `SetupLogger` first). Fixed in review: (1) a shuffled race run caught `SetupLogger` swapping the logger while `TestTypedInputNeverTakesACombatCause`'s input worker was still logging its stop line; the logger is now an `atomic.Pointer` (so a worker logging never races a setup), and that test waits for its worker to stop (regression `TestLoggingNeverRacesSetupLogger`); (2) `TestLeadrootCutsPhysicalDamageByFifteenPercent` (43b, arrived in the master merge) never loaded its own buff and weapon specs, so it failed alone and passed only after another poison test; it now calls `poisonSpecs`; (3) master was red on `TestEveryLineageHasBaseArt` (39d added the Doll Master lineage, 40h's test wants base art for every lineage), so review drew a base Doll Master figure (plum coat, cross-bar control, a wooden doll on strings; map views and battle idle) through `make sprites`; it stays a candidate for the later art pass. The Relentless test (also reported by the 39f review) passed 60 of 60 at the 12-round window. `modules/company` passed both full shuffled race runs in review, so no follow-up is open for it; `TestCombatEventStreamThroughTheRealRound` stays watched. UI check: test-only phase, nothing player-facing. Verification: master merged in, `make generate`, `make validate`, `make js-lint`, full `go test -race -shuffle=on ./...` runs (the last one after the final master merge), and the fixed packages again shuffled.

**Phase 37d built: test isolation (2026-10-06):** shuffled runs (`go test -shuffle=on`) of every package except `modules/company` now pass repeatedly; each failure was shared state, fixed at its cause, with no test skipped or disabled.
(1) Event queue leaks: the queue is global, so events an earlier test queued and never processed reached later tests' listeners (extra messages, deaths and arrivals). New `events.ClearQueueForTest`; tests that register a listener call a per-package `freshEvents(t)` first and at cleanup (`usercommands`, `mobcommands`, `expedition`, `camping`, `company`, `hooks`, `gathering`, `gmcp`, `death`, `archetype` and others). This was `TestSpawnLootRollsAnItemIntoTheRoom`, `TestAttackOnAWaitingGroupIsRefused`-style extras, `TestSuicidePendingGoesStraightToRespawn`, the camping raid and theft tests, and the practice and drop tests.
(2) Shared world state: tests used `AddOverlayOverrides` (never restored, and it ignores a key already set) and loaded the shipped world for good, so "room #100" titles and arrival text depended on order. New `configs.SetTestDataFiles` and `rooms.ResetForTest`; expedition and camping load the world per test (`useShippedWorld`, `loadShippedWorld`) and clear it after, and title tests call `noWorld`.
(3) Unset logger: `mudlog` had a nil logger until `SetupLogger` ran, so tests in a package without it panicked (`characters`, `companyview`, `hooks`, `rooms`...). It now logs to stderr by default (the one production change). `TestPanelLayout_PanelLookup_PanicsOnMissing` had only passed because of that panic; it now checks the documented no-op panel.
(4) Other order dependencies: `look` rooms were dark depending on the clock (rooms now tagged lit), `TestDefenseHelp` read whatever gameplay config an earlier pass left (now validated defaults), `useWorld` now loads the world's keywords (help tests panicked without them), drop tests inherited another test's noted spoils, and `TestWarlordRelentlessQuickensItWhenItsFoeStandsUp` gave a foe 6 rounds to stand (about 1 in 14 failed; now 12, 80 of 80).
Left watched: `modules/company` is slow (about 2 minutes) and one shuffled run failed `TestCombatEventStreamThroughTheRealRound` once ("the bandits fall" within 200 rounds; 40 of 40 solo); `TestAimedShotGrowsWithLevel` did not reproduce (40 of 40); the archetype flake and `TestAlliedFinalEnemyPaysAfterCombatClosesBattle` did not reproduce in the shuffled sweeps.

**Phase 37c reviewed and merged via [PR #52](https://github.com/Robinsond76/ashveil-gomud/pull/52) (2026-10-06, Opus review thread).** Review checked the company-level rating (`look`, `scout`, GMCP `Room.Info`, the web header resend on a level change), the conversation sweep fix, and each test fix. Accepted and fixed: (1) the rating line never said what level it rated, so a lone leader and a full company could read one zone differently with no clue why; `look` and `scout` now name it ("Risky at your company's level (7): expect losses."), and `help encounters` says the fallen count and `look` names the level; (2) `util.UseRandForTest` swapped a plain package variable that every goroutine's `Rand` reads; it is now an `atomic.Pointer`, so a test that pins it never races a goroutine still rolling. Agreed with the builder: the fallen count toward the company level (they come back; every member status is temporary). Verification: full `go test -race ./...` twice (once before and once after the master merge), plus shuffled repeat runs of `modules/company`, `modules/archetype` and `internal/conversations`. Still unreproduced and left watched: the unnamed archetype flake and `TestAlliedFinalEnemyPaysAfterCombatClosesBattle`; `TestAttackOnAWaitingGroupIsRefused` has a likely fix only (shared road corpses). Also fixed in review: `TestIaijutsuIsTheFirstStrikeOfABattleOnly` (about 1 run in 8: Iaijutsu's own +10% critical chance stands with the odds pinned to 0, so a crit doubled the strike; the test now pins `util.Rand` to its top face; 100 of 100). Follow-up (candidate phase 37d): `internal/usercommands` tests share package state, so shuffled runs and `-count>1` fail (`TestDefenseHelp`, `TestStartFallsBackWithoutTheTutorial`, `TestSuicidePendingGoesStraightToRespawn`, `TestLookNamesTheLootClaimant` and others; each passes alone and in the default order); `TestSpawnLootRollsAnItemIntoTheRoom` (failed once in a full race run, 200 of 200 solo) is likely the same family. After the phase 45 master merge `TestStartFallsBackWithoutTheTutorial` failed every run: its premise (no tutorial rooms) held only if no earlier test loaded the default world, and `TestDefenseHelp` does; it now clears `TutorialRooms` itself through the new `configs.SetTestSpecialRoomsConfig`.

**Phase 37c built: test stability and company-level zone rating (2026-10-06):**
(1) The zone band rating (easy, fair, risky, dangerous) in `look`, `scout`,
GMCP `Room.Info.levelband` and the web header now rates the **company's
level**: the rounded average of the leader and every companion, the fallen
included (`companyview.CompanyLevel`, the same "average level" the enemy
coordination tiers use; the leader alone when the company can't be read).
The web client's header refreshes when that level changes (a level-up, a
recruit, a dismissal): the GMCP room module resends `Room.Info` on a
company-level change inside a banded zone. Help (`encounters`, `scout`,
`look`), the web tooltip and the tutorial hint say "your company's level".
(2) Flaky tests, found by repeated and shuffled runs of `modules/company`
(`go test -race -count=N -shuffle=on`) and fixed at the cause:
`TestCompanyMovesAsOneThroughGo` (the moving, recruit and roster tests write
different Dunmar 2003/2001 room files into one process-wide room cache, so
whichever ran first left the others its rooms; now `freshDunmarRooms` evicts
them before and after each); `TestEncounterFightEndsInOneCacheAndTheSpoilsLine`
and likely `TestAttackOnAWaitingGroupIsRefused` (corpses and gold left on the
shared brawl road by earlier tests; `newBrawl` now clears them; the waiting
group test failed once in a shuffled run and never in 700 solo runs, so this
is a likely cause, not a proven one);
`TestBalanceMirrorClericIsACasterWhoCastsNothing` (about 1 run in 40: the
balance fixture installs the real random aim roll after the test pinned its
own, so the coordination tier's noise floor sent an enemy at a random
member; the pin now comes after the fixture, 250 of 250);
`TestAimedShotGrowsWithLevel` (dice and blow quality noise against a
tight margin; the test now pins `util.Rand` through the new
`util.UseRandForTest`); `TestSpellEventsThroughTheRealRound` (every
chant-breaking blow broke the cast, so about 1 try in 20 went off and 60
straight failures came up in about 1 run in 30; chants are now held against
ordinary blows, and a try succeeds about 28% of the time); and a nil map
panic in `TestRosterThroughPluginsLoad` (an event queued by an earlier test
fired before the test reset its message map); and `TestAttemptConversation_UsesPluginFile`, which was a real engine bug: the conversation sweep pruned a conversation that had not yet been stepped (its `LastRound` is 0) whenever the round count was past ten, so a fresh conversation could vanish (now aged from its start). Not reproduced: the
`modules/archetype` failure (25 shuffled runs clean; the test is unnamed) and
`TestAlliedFinalEnemyPaysAfterCombatClosesBattle` (failed once in one
shuffled run, clean on the same seed afterwards).

**Phase 40c reviewed and merged via [PR #49](https://github.com/Robinsond76/ashveil-gomud/pull/49) (2026-10-06, Opus review thread): terrain and landmark tiles.** Review: regrow watcher cost is bounded by rooms currently picked clean (entries dropped on full regrowth), `Ledger.Charges` is read-only, no game time touched; allied camp `embers`/`tent` add nothing beyond what party members already see. Accepted and fixed: (1) `World.Resources` went to every online player, telling them of rooms they had never visited; it now goes only to players who have visited the room (`TestWorldResourcesGoToVisitorsOnline`); (2) a regrow watch that found nothing clean stayed in the per-round scan forever; it is now dropped (`TestPickedCleanShowsOnLookAndQueuesARedraw`). Agreed with the builder: tiles ignore the size and spacing sliders (scaling 32 px art off-grid would smear it; zoom covers size). Built: the web map draws each room as its biome's S2 terrain tile (variant = room id mod 3), tiles touch, and a dark edge marks two touching rooms with no exit between them. Exits to unvisited rooms end in a fog tile, up and down exits show the S1 chevrons, and a room whose `maplegend` (or `mapsymbol`) maps in `sprites/map/landmarks.json` shows its landmark overlay; an unmapped symbol keeps its letter, outlined, and `Shore` is an intentional no-glyph legend. A biome with no art draws the `unknown` tile; art still loading or missing falls back to the classic colour square per room. Map settings gain `Style` (`tiles` default, `classic` unchanged; size, spacing and shape apply to classic only). Zoom moves on crisp steps (16 to 128 px tiles); animated biomes cycle at 250 ms and stop with reduced motion. Layer order: terrain, walls, fog, connections, landmark, resources, camps, units. Two follow-ups folded in: (1) regrowth now redraws the map: the gathering module watches rooms it picked clean and queues `RoomResourcesChanged` when a pool regrows (real time only), and the gmcp module sends every online player a small `World.Resources` update (room, shown, depleted) that the map patches into its room info; (2) companions are drawn beside you as their class at 75 percent (up to four, present only, class from the new `lineage`/`classid` fields on `Company` members: `lineage` plus the `class` key 40s5 added), with the badge still counting everyone. (3) camps use the 40a3 fields: `embers` draws the low-glowing embers sprite when the fire is not lit and `tent: false` draws the rough camp (bedrolls, no tent) instead of the tent, for your camp and, with the same two fields added to allied camps, your party's. Help: `help worldmap` (terrain, landmarks, companions, regrowth, Style) and `help webclient` updated, camp tutorial hint mentions the tiles. Tests: `TestMapLegendsHaveLandmarks` (every shipped `maplegend` maps or is an intentional glyph; landmark ids have art), gathering regrowth redraw, `World.Resources` payload, `Company` class ids, and the Chromium `scripts/browser/map-check.mjs` (tiles, walls, fog, landmarks, classic restore, missing-image fallback, reduced motion, crisp zoom, companions, regrowth redraw). Screenshots: `screens/40c-tiles.png`, `screens/40c-companions.png`. Decisions (owner delegation): tiles are the default (S2 covers every biome; classic kept); tiles ignore the room size and spacing sliders (32 px art, spacing equals size) rather than scaling art off-grid; the static-layer offscreen cache from the design is skipped (a few hundred `drawImage` calls per frame is cheap and the units already redraw continuously; revisit if a large zone measures slow); `Entrance` and `Exit` both use the cave mouth (the catacomb entrance shares it); regrow watching is in memory (a restart forgets it, and the client refreshes on the next World.Map). Follow-ups: S1 resource icons still draw as dots over tiles; the shared landmark table has no entry for shop, smithy or herbalist because no shipped room carries those legends yet (41/42 add rows).

**Phase 40g2 reviewed and merged via [PR #48](https://github.com/Robinsond76/ashveil-gomud/pull/48) (2026-10-06, Opus review thread):**
ally relay checked for consent (both players support the party), same room,
shared enemy and the receiver's own unseen masking; mid-battle inputs are
still only retreat and company focus; the nerve mark uses `morale.Losing` on
the fight's roster like the check does. Accepted and fixed: (1) an allied
player with no formation is named `u:<id>`, not `a:<leader>:<key>`, so what
befell them kept its numbers and statuses; `isAllyRef` now covers `u:` refs
(regression `TestAlliedHappeningsScrubAnAllyWithoutFormation`); (2) UI check:
allied members were drawn as their base class even when promoted, so
`allies[].members[].promoted` now carries the 40s5 class id and the screen
draws that art first; (3) `help battlescreen` said weak companions "may now"
falter while the mark shows, but the nerve test runs once per fight; reworded.
Rejected: the mark reads `fi.Company`, which grows if a companion joins
mid-fight, while the check uses the starting roster; reading `hooks`' fight
state from the GMCP goroutine would race, and the mismatch needs a mid-fight
join, so it stays (decision (d)). Merged master (40s5 class art): the caption
names a member's class and an ally's company. Follow-ups: tap an allied
formation to watch it full size; an ally's blow on a foe the receiver is not
fighting is relayed but has no figure to land on.

**Phase 40g2 built: battle screen follow-ups (2026-10-06):**
closes the 40g review's follow-ups. (1) **Allied formations.** Each allied
company fights its own fight (33d), so the 40e feed now also relays a
fight's happenings to the leader's consenting allies
(`parties.AlliedLeaders`) who are in a battle of their own against some of
the same mobs in the same room, with the ally's members as
`a:<leader>:<key>`. Only watchable kinds go (strikes, spells, heals, casts,
wind-ups, interrupts, falls, flight, abilities, target changes, statuses on
enemies), never a fight's start, end or focus, wound changes, guards or
mercy; what befell an ally has its numbers and status scrubbed and no status
event on an ally is sent. `Company.Battle` gains `allies` (leader id and
name; each member's id, name, class, cell, health in words, `down`), and the
screen draws up to two as half-scale formations behind and above the
player's, each with a pennant and the leader's name, a "+N more" mark for
the rest, and pennants with a count of those standing on a phone. Allied
units strike from where they stand (no lunge across the field). (2) **The
pace is sent:** every `Company.Battle.Event` message carries `pace` (fast,
normal, slow, off; `hooks.PaceOf`); the screen uses it and keeps the old
inference only as a fallback for a server that sends none. (3) **Morale is
drawn:** `Company.Battle.nerve` is "faltering" while the nerve rule
(`morale.Losing`: half the company down or a quarter of its health left)
holds; the header says "company faltering", each company figure shows a
drop of sweat, hover says "shaken", and a hesitating companion's lost action
reads "hesitates as the company falters" in the last-blow line. Enemy yield
and flight markers were already shown. **Role letters are crisp:** a 3x5
pixel font drawn in whole pixels (also used for "yields" and the allied
labels) replaces anti-aliased canvas text; a browser check samples the role
chip and finds two colours only. **The unseen presence** takes the first free
enemy cell (centre first) and steps aside when a foe comes into view.
Decisions (delegated): (a) allies are matched by party consent plus a shared
enemy and room rather than a new alliance store, so nothing is persisted and
33d's rules are untouched; (b) an ally's blows on the shared enemy keep
their damage digits (the enemy's numbers are already shown) while numbers on
ally members are hidden, the conservative reading of "bands only"; (c)
tap-to-swap to watch an ally full-size is not built (view-only, the design
marked it a recommendation), a candidate follow-up; (d) the nerve mark is
computed in `modules/gmcp` from the fight's roster with the same
`morale.Losing` rule rather than reading `hooks`' private fight state; (e)
only two allied formations are drawn (the design's cap). Tests:
`modules/gmcp/gmcp_battle_allies_test.go` (relay with consent, room, shared
enemy, unseen masking and scrubbing through the real stream, hooks pacer and
GMCP path; payload; ally selection; nerve; pace on released batches), a Node
planner test for allied refs, and the allied section of
`scripts/browser/battle-check.mjs`; help `battlescreen` gained Allies,
morale and pace text (tested) and the Departure tutorial lesson points to it;
screenshot `screens/40g2-battle.png`.

**Phase 39b complete (merged via PR #47, 2026-10-06): the Samurai neutral lineage.** Iaijutsu (first swing of a battle +50% damage, +10% crit, half a turn ahead; spent hit or miss), Focus (+3% crit per quiet round to +9%), Zanshin (+50 meter once a round when it fells a foe); routes Kensai (piercing draw), Hatamoto (Bodyguard: guards the leader twice a battle), Ronin (Vengeance per fallen ally). New base-rank mechanism (`classes/base.go`) for lineage ranks from level 1. Samurai archetype (6 HP, 0.85/level, medium armor, sword), recruit mob 139, default rule strongest, Camp Watch utility. Help: `help samurai`, `help samurai-routes` plus updates to related pages; tutorial hint in the creation lesson. Plan: [39b](plans/2026-10-06-phase-39b-samurai.md). Balance (80 fights, ±5): Samurai 83/91/77% vs Rogue 76/92/76% at L5/10/20. Decisions: duelist is a Fighter with default rule strongest; first strike spent on the first swing; Bodyguard precedes strategy guards; Focus 3/9 and Attack 1.0 tuned down from the design; elites stay planned (39i). Follow-ups: Samurai battle sprites (art pass), elite ranks (39i). Review (Opus review thread): accepted, level-up reports named only the next rank, so a player never saw what Focus or Zanshin (or any route rank) had just given; fixed with `classes.RanksGained`/`RankLines`, a "New rank: ..." line in the player and companion level-up reports, `help classes` updated, regression tests in classes, hooks and mobcommands; accepted, this entry described Zanshin as triggering on being struck (it fires on felling a foe), corrected. Checked and kept: Iaijutsu RT exists before the first blow (auraPass makes it for any character with class effects), Vengeance counts only standing members, Bodyguard skips the leader itself and spends its own count, effect keys are unique, recruit 139 does not collide with 39a's 130. UI: `class`, level-ups, help and the battle-screen hue cover the change; no web panel lists ranks, so none needed updating.

**Phase 40a3 complete (2026-10-06, PR #45): camp gear.** The fuel rule
and six durable camp items. A finished camp rest now burns the fire down to
**embers** (`Camp.Embers`): embers keep the room warm until the camp is broken
(a damp fire's embers stay cold) but give no light, and resting again needs
`camp fire` again, which spends another bundle (or the deadfall in a firewood
room); the camp then takes a fresh rest session (new operation id), once the
last rest's recovery is in. A camp saved before this phase burns down on load.
New items 45-50 (`bedroll` 2.5 kg, 8 gold; `oiled canvas tent` 9 kg, 40;
`fire steel and tinder` 0.2 kg, 5; `iron cookpot` 3 kg, 12; `camp bells and
trip lines` 1 kg, 6, 10 uses; `field surgeon's kit` 1.5 kg, 25, 5 uses), sold
at the Dunmar and Old Kings Road markets (road dearer) as `SupplyOnly`, so none
buys back. `modules/camping/gear.go` counts the gear through
`company.CompanyItemCount` (cargo, the leader's pack, present companions'
packs; separated and dead members' packs are skipped) before `m.mu`, and locks
it on the rest (`RestSession.Bedrolls/Bells/Kit`, `Camp.Tent`, all saved, so a
restart or copyover mid-rest keeps them). **Bedrolls:** one per member, leader
first then companions by number; `survival.ApplyCompanyRestRecoveryBonus` gives
those members +25% of the rest's fatigue on top (the ledger keeps the base
amount, so a replay is still a no-op). **Tent:** counts as shelter for the
weather (never stacks with a shelter room), makes the camp room a heat source
while resting (no rest-time cold), shows in `look`, `camp status` and GMCP
`Company.Camp.tent`. **Fire steel:** damp bundles light first time at full
warmth. **Cookpot:** `camp cook` makes two portions of a dish with two or more
inputs. **Bells:** a flat 20% spot chance with no watch, +10 points on a watch
capped at 90, one use worn when a rest starts; their own warning line.
**Surgeon's kit:** at the end of an unbroken camp rest a healer who knows Tend
Wounds and has the mana tends the worst lasting wound of the most wounded
member present (`company.FieldSurgery`), before bandages and splints; one use
worn only when a wound was treated. The rest start, `camp status` and the
rest-complete line report the gear in use. Web: the Camp tab shows embers
("Feed fire"), the tent, and offers Rest again once the fire is fed
(`/mnt/project-files/screens/40a3-camp-embers.png`); GMCP `Company.Camp` gains
`embers` and `tent` for 40b's map sprites. Help: new `help camp gear` (indexed
under `road`; aliases bedroll, tent, cookpot, fire steel, camp bells, trip
lines, surgeon's kit, embers) and updates to camp, gathering, campwatch,
cooking and wounds; tutorial: the Survival lesson hands out a bedroll and a
fire steel with its supplies and the Camp hints explain embers and gear.
Design: [40a3](designs/2026-10-05-phase-40a3-camp-gear-design.md).
Decisions (builder, owner delegation): (1) gear in the cargo counts without a
horse check: cargo has no per-horse presence concept and always travels with
the company, so the design's "cargo with its horse present" is met by the
existing cargo rules; (2) the bells wear one use at every rest start, raid or
not (the design's "wears 10 rests"); (3) no restring or restock service: a spent
set of bells or a kit is bought again (the 3 and 10 gold refill prices were
dropped; items with uses already model wear and a refill action would be a new
shop mechanic); (4) gear is sold `SupplyOnly` and is never loot, so nothing
gathered for free sells for more than a low-level fight pays; (5) the tutorial
gear is granted with the Survival supplies (same once-only flag) because stage
entry grants only there; (6) the cookpot is read when cooking, not locked on a
rest, since `camp cook` is its own command; (7) the tent is pitched when the
camp is made and refreshed at `camp fire` and `camp rest`. Not folded in: the
40a2 follow-ups (gather progress as a web panel, a redraw on regrowth) are
about gathering and the map, not camp gear; both stay open (the map redraw
belongs with 40b).
Review (2026-10-06): checked the fuel rule and repeat rests (a new operation
id per rest, the last rest's recovery settled first, the burn-down of old
saves), the bedroll ledger, raid spotting with bells, the kit's
before-refill mana gate, item ids 45-50 (no clash with master or any open
branch) and the economy (all six `SupplyOnly`; objects never salvage; a
generic merchant pays at most a quarter of value, so the tutorial's free
bedroll and fire steel fetch about 2 gold each, once per character:
accepted). Kept the builder's three decisions: cargo counts without a horse
check (cargo always travels with the company under 32f), bells wear at every
rest start (a count the player can predict: "10 rests"), and no restring
service (rebuying at 6 and 25 gold costs little and needs no new shop
mechanic). Fixed: `camp status` between rests now lists the gear at hand
and what it will do (`campGear.lines` was unused, so a player saw the gear
only once a rest began), with a regression test and the help updated; the
dock browser check timed out because the 40f battle screen opens over the
Combat tab, so it now sets the screen to manual (battle-check covers the
screen), and its focus-bar count was stale (eight buttons, not seven).
Follow-ups: the web Camp tab and GMCP carry only the tent and embers, not
bedrolls, bells or the kit (candidate: a gear line in `Company.Camp`);
the 40b map draws a burned-down camp as a cold fire (candidate: an embers
sprite from `Company.Camp.embers`).

**Phase 40s5 reviewed and merged (2026-10-06, Opus review thread):** the
class art, GMCP `class` key and battle-window fallback are sound; promoted
classes read apart from their base at 1x in the contact sheet and the battle
screen (UI check screenshot kept outside the repo as
`screens/40s5-battle.png` in the project files).
Decision (delegated): `TestEveryBuiltClassHasArt` no longer fails when a
built class has no art, because elite (38c) and neutral (39) classes are
built in parallel and the battle screen already falls back to the base
class (or a silhouette for a lineage with no art). It logs each pending
class and its fallback instead; the later art pass runs it with
`ASHVEIL_ART_STRICT=1`, which fails until every built class has art. It
still fails when `promotedClasses` names a class that does not exist.
UI check, fixed in review: the company list showed a promoted member only
by its base archetype ("Warrior" for a Paladin) and the battle caption only
by name, so Company GMCP members now also carry `class_name`; the company
card shows the class (its line on hover) and the battle caption reads
"Wren, Paladin, striking …"; `help battlescreen` says so (tests: GMCP
payload, `TestBattleScreenHelp`, and assertions in
`scripts/browser/battle-check.mjs` and `dock-windows-check.mjs`).
Follow-ups: `score` and the Character window still show no class (38b
follow-up); `dock-windows-check.mjs` stalls at its Combat-tab hover since
40f because the battle screen opens over the tab (pre-existing; the 40s5
assertions run before that point); three members in one row overlap
heavily on the battle screen. Merged master (44 smoke, 40a2 shaman
renumber: same mob 97, kept the shaman's own sprite) and 40g (battle
animation): a promoted member uses its own class's pose sheets once its
idle exists, never the base class's, so it does not flicker between looks.

**Phase 40s5 built: class art, art set S5 (2026-10-06):** map sprites
(down/up/side, idle and walk) and battle idles for the 23 promoted classes on
master (the 18 level-10 classes plus Paladin, Hierarch, Elder Druid, Blood
 Priest, Demonologist and Dread Knight), the Angel and Demon summons (large
units) and a goblin shaman of its own (was the hexer's art); `scripts/sprites/promoted.py`,
`summoned.py`, `make sprites` now also writes
[`docs/verification/40s5-contact-sheet.png`](verification/40s5-contact-sheet.png).
Forks, with reasons: (1) each promoted class is its lineage's base figure with
ramps swapped and a few accessories, not a new rig, so a promotion reads as a
step up and the S0 proportions and anchors hold for free; (2) good routes go
lighter with brass, neutral earth-toned, evil darker with ember touches, using
the existing 64-color palette only; (3) the Company GMCP member now carries
`class` (promoted class id, omitted before promotion) and the battle window
draws `battle/units/<class>/idle.png` when the manifest lists it, falling back
to the base class, so the art shows today; (4) mobs `95-angel`, `96-demon` and
the shaman use `sprite:` keys; (5) the goblin shaman is mob 97 (master made the same fix in 40a2, since 38b's Angel holds id 95). Elite (38c) and neutral (39) classes still
`Planned` are left to a later art pass: warlord, pathfinder, swordmaster,
nightblade, sentinel, marksman, ravager, archon, archmage, necromancer,
wise-one, coven-mother, crone-of-ash; `TestEveryBuiltClassHasArt` lists built classes
with no art (see the review below), and `TestMobSpriteKeysHaveArt` checks every mob
`sprite:` key. The 40b map window (merged alongside) draws a promoted
player's class map sprite first, so the S5 map art shows there too. No new player command, so no help page. Gates: `make generate`, `make validate`, `make js-lint`, `go test ./scripts` and `go test -race ./...` green; `TestAimedShotGrowsWithLevel` failed once in the full run (a random-roll comparison, 14 vs 15) and passed on three reruns, unrelated to this change.

**Phase 40b complete, merged via [PR #43](https://github.com/Robinsond76/ashveil-gomud/pull/43) (2026-10-06): map sprites in the web client.** The Map
window draws you as your class sprite (chain: current class, lineage,
`adventurer`, then the classic red square), a gold here-ring under you and the
company badge (members present with the leader, hidden alone), all drawn above
the terrain so you never blend in. Moves walk tile to tile facing the way you
went (up/down/side, west mirrored), queue at most 2 steps behind and then snap;
jumps (recall, teleport) do not walk; level or zone changes snap and fade in.
Your camp draws as a tent with fire, smoke and a resting mark, `inn-rest` while
resting at an inn; camps of your **party** draw as `tent-ally`; no other
company's camp is ever sent or drawn. Party members with a known class draw as
that class at 75% with an ally pennant, else stay hearts. Server: `Char.Info`
and `Party` vitals gain `lineage` and `classid`; `Company.Camp` gains
`room_id` and `allied_camps` (party members' camps only, regression-tested with
a same-room outsider). New `static/js/sprites.js` loader (manifest, status,
fallback, redraw) for 40c and later. Settings: Sprites and Camps. Help:
`help worldmap` (aliases `map window`, `tile map`, `world map`), indexed, linked
from `help webclient`, `help camp` and `help map`; Camp tutorial hint. Check:
`scripts/browser/map-check.mjs` (facings, queue, badge, camp, allies,
sprites-off, missing image); screenshot `screens/40b-map.png`.
Decisions (delegated, with reasons): sprite scale is the nearest whole multiple
of 32 px (half, 16 px, when zoomed out below a 16 px tile) so pixels stay
square; `inn-rest` shows when the camp tile is `here`, an inn, and resting
(`Company.Camp` has no separate inn-stay field, adding one is not worth a new
payload); the unit walks on its own queue at 200 ms a tile while the camera
keeps its existing ease; allied camps refresh with the company feed's changed-
payload sends (no new event); the sprite loader does not yet replace the
battle screen's own loader (follow-up, no behaviour change). Not changed: other
players do not appear on the map, no race variants (owner deferrals).
Review (Opus): decisions hold (whole-multiple scale keeps pixels square;
inn-rest rule matches the payload; allied camps reach the client within a
round because the company feed rebuilds every round). No leak: allied camps
come only from accepted party members (not invitees), party sprites ride the
existing party-only `Party.Vitals`. Accepted and fixed: a camp on your own
tile was hidden under your sprite, so it is now pitched behind your left
shoulder with the fire by your right foot (map-check regression); `help
worldmap` claimed the map never shows more than `look`, though it shows your
party's camps anywhere in the zone, reworded; after merging master (44's
`ClassTitle` in `Char.Info`), the sprite-keys test's fake archetype provider
gained `ArchetypeName`. Rejected: continuous redraw
while sprites show is needed (every map sprite animates, and closing the
window stops it). Follow-up: companions are not drawn on the map (only the
badge count).

**Phase 40g reviewed (2026-10-06, PR #40):** Checked the scheduler never
drops a state change (a collapse applies every unfired op in order) and
catches up after a round of backlog, that reduced motion drops lunges,
travel, tints and shake, and that an unseen foe stays unseen. Accepted and
fixed: (1) **the title showed the server's round counter** ("round 48213"),
since the feed's `round` is global; `Company.Battle.Event` now also sends
`fight_round` (from 1, from the fight's start round or, for a fight-end, its
summary) and the screen shows that (Go wiring test and gmcp unit test
assert it). (2) **The outcome and the victory pose came before the last
blow** while animation was behind: the fight's end now waits for every unit,
and the outcome (with its reason, read while the last snapshot stands) shows
when it plays; the empty snapshot that follows no longer shows "The battle is
over" first or closes the screen early. A fight-start in the same batch as
deaths no longer clears their hold. (3) **Spells were named by id** ("Ysolde's
mm strikes"): the feed carries `spell_name` and the last-blow line and chant
mark use it; an unseen caster's spell is neither named in the line nor
coloured in its glow. (4) The help page no longer promises the picture never
falls behind (it skips ahead after a round) and says the outcome follows the
last blow. Tests: Node planner tests for the end-wait and hidden spells,
browser checks for the outcome order, spell names and fight rounds (the
harness now sends a server-sized `round`). Rejected: none. UI check: at each
setting a player can follow who did what (last-blow line, digits, icons) and
why the fight ended; `off` keeps the 40f picture. **Follow-ups:** 40g2
allied formations (an allied relay in the 40e feed plus a third formation on
the canvas; not small, so not folded in); a `pace` field on `Company.Battle`
to replace pace inference; morale not drawn; role letters blurry; a `?`
presence can overlap a visible foe. Gates: one full race run failed once in
`modules/company` (its output was not kept); three package reruns and a
second full race run passed, so it is likely one of the known company
flakes, unidentified.

**Phase 40g built: battle animation and effects (2026-10-06, PR #40):**
the battle screen now plays each 40e event. A pure planner,
`static/js/battle-timeline.js`, turns an event batch into steps (lunge,
strike, shoot, chant, hurt, block, parry, dodge, windup, guard, tackle,
yield, flee, fall, victory) with hit effects, projectiles, glows, digits and
feedback icons, and a `Scheduler` queues them per unit (units overlap, one
unit's actions do not), fires state changes when their step starts or ends,
and when more than a round of work is queued collapses the older steps to
their end state. `window-battle.js` plays them (S4 sheets at
`battle/units/<key>/<pose>.png` and `battle/effects/...` when the manifest
lists them; otherwise the idle figure nudges and effects are drawn in code).
New on screen: the round in the title, a "last blow" line ("Wren hits the
second wolf (6)"), a named outcome reason ("Victory: no foe is left
standing"), a zone backdrop lookup (`battle/backgrounds/zone-<slug>.png` by
`Room.Info.area`), and an **Animation** menu (`full`, `reduced`, `off`;
`ashveil-battle-animations` in `localStorage`; default reduced when the
system asks for reduced motion). `off` is the 40f path unchanged. Tests:
`scripts/js/battle-timeline.test.mjs` (Node, new `make js-test`, run by
`make test` and CI) and the animated section of
`scripts/browser/battle-check.mjs`; screenshot `screens/40g-battle.png`.
`help battlescreen` gained the Animation section. Decisions (delegated):
(1) **Pace is inferred, not sent:** the feed carries no pace, so the client
reads it from how batches arrive (whole-round batches mean pacing off, else
the gap between batches: under 0.9 s fast, over 2.5 s slow); the budgets are
the design's 1.2 s action and 0.6 s reaction, half for fast, 1.5x for slow,
0.3 s for off. A `pace` field on `Company.Battle` would make this exact
(follow-up, small). (2) **S4 art is not drawn** (40s4 is not on master), so
every pose uses its fallback and effects are code-drawn; art lands with no
code change. (3) **Allied reserve formations stay deferred:** they still
need an allied relay in the 40e feed (a Go change to combat events, out of
scope for a client polish phase). (4) **Crit digits are large, not
"12!"** (the narration style has no exclamation marks) in animated modes;
`off` keeps the 40f text. (5) A fall or exit waits for its animation: a
`Company.Battle` snapshot arriving mid-fall does not lay the unit down or
drop it early (`holding`). (6) The outcome hold waits up to 6 s more for the
last animations. (7) Sound stays out of scope, per the roadmap. (8) **Review follow-ups folded in:** bars, role
letters and statuses now draw in a second pass over every figure, so a large
unit in front no longer hides the health of those behind it; a hit tints the
figure's own shape for art units (a box for code figures). The flaky
`TestSpellEventsThroughTheRealRound` ("mm never went off") is hardened:
Aria did not own Minor Heal or Magic Missile, so each try rolled a success
chance beside the bandits' interrupts and 60 misses in a row could happen;
she now learns both (an owned spell never fizzles in battle, 35b) and has
100 tries. Probable cause, not reproduced alone (25 and 15 clean runs
before and after). Still open: morale (nerve) is not drawn; role letters
are still blurry at the canvas font size; a `?` presence can overlap a
visible foe.

**Phase 44 complete: live smoke playtest (2026-10-06):** `make smoke` builds
the server, copies the shipped world and plays a new Warrior over telnet
through the whole tutorial (a real camp rest and a real battle), the `help`
index, copyover, a second player (a Witch) in the same room, and a restart
and relogin; see [Live smoke playtest](LIVE_SMOKE_PLAYTEST.md). It is
env-gated (a few minutes of real time) and meant to be re-run at the end of
each lane. First run found, and this phase fixed with regression tests:
the tutorial's straw soldiers hit for the flat damage floor (a 0d0 body with
no weapon now deals nothing; they had killed the whole company, against
"cannot hurt anyone"), `status`/`who`/GMCP called every warrior a "scrub
paladin" (now the archetype name), every login printed "inbox not
recognized" and "mudletmap not recognized", a won fight ended with "Your
target can't be found.", `help help` and `help bid|store|unstore` found nothing (the
index listed a command that does not exist). Not fixed, noted: a line typed
within one turn (50 ms) of the last is silently dropped, so a scripted
client must pace itself; the tutorial hand-off drops input typed during it;
the gate hand-off prints "looks a little confused (gate )" for each
companion; "The battle is under way" is still said for a moment after a
fight's summary; the `weather` command says "You can't tell what the weather
is like here" in the open Weather Yard. Not covered (proposed 44b): a fight
against a world mob with loot, travel on the Old Kings Road, and Dunmar's
inn (the gate lands in Frostfang; the one shipped travel route starts in
Dunmar, and no step of this run reached it), all needing 37's encounters.
Verification: `make generate`, `make validate`, `go test -race ./...`,
`make js-lint`, `make smoke`. One flake seen: `TestAttackOnAWaitingGroupIsRefused`
(modules/company) failed once in the full race run (its battle was already
over after one round) and passed on a rerun of the package; it is random,
not from this phase.
Review (2026-10-06, Opus review thread): accepted the five live fixes as
root-cause fixes (only races 19 dummy and 20 orb are 0d0, so no fighting
mob lost its blows; `ClassTitle` falls back to the profession title with no
archetype). Fixed in review, each with a regression test: a line typed
within a turn of the last now waits out the turn instead of being dropped
(telnet, websocket and restored connections; `waitForTurn`); companions no
longer print "looks a little confused (gate )" when the graduation has
already moved them beside their leader (`companionAlreadyWithLeader`; the
smoke run now fails on that line); `TestAttackOnAWaitingGroupIsRefused` is
deterministic (the bandits outlast the first round). Import grouping tidied.
Rejected or left as follow-ups: "The battle is under way" for a moment after
a summary (clearing the aim at victory did not change the brawl harness, so
the live cause is elsewhere; `doAfterBattle` covers the smoke run); `weather`
in the Weather Yard (only the forest biome has a weather table, so the
tutorial and Frostfang show none: a content gap, not a code bug); input
typed during the tutorial hand-off. A package run before the fixes showed
the known `TestBalanceMirrorClericIsACasterWhoCastsNothing` flake (37b).
Verification after review: `make generate`, `make validate`, `make
js-lint`, `go test -race ./...` (pass), `make smoke`. Merging master then
found 37b's goblin shaman and 38b's summoned Angel both used mob id 95
(every world load panicked); the shaman is now mob 97, its dark forest
encounter updated, since the Angel's id is a code constant (40a2's review made the same fix).

**Phase 40a2 complete, merged via [PR #41](https://github.com/Robinsond76/ashveil-gomud/pull/41) (2026-10-06): gathering.** Herbs,
firewood, fishing and game are real. New `gather [herbs|firewood]`, `fish`
and `hunt` commands (module `modules/gathering`, rules in `internal/gathering`)
run a real-time timed action (20s herbs and firewood, 30s fish and game) kept in
memory and resolved on the round tick with the real clock; it never touches the
world clock. A typed command (other than look and a few reads), a move, a fight
or death cancels it with nothing gained. Each room resource has a pool (herbs 3,
firewood 4, fishing 4, game 2) shared by every company, regrowing one charge per
20 minutes (game 40); only pools in recovery are saved (plugin `pools`, computed
on read, so a restart never refills early). Depleted resources show "(picked
clean)" in `look`, GMCP `Room.Info`/`World.Map` (`depleted`), the room panel
(dimmed badge) and the map (hollow dot with a slash). Herbs use the zone's
table; a company without a Forage specialist risks bitter weed; a knife, a
Forage specialist and a Scribe (rarer herb, 10%) add finds; darkness halves
them. Firewood doubles with an axe (+1 Field Smith) and half is damp in rain
or storm. Fishing needs a line (5% break) and costs half effort; game needs a
bow or sets snares (half chance), adds +10 to the room's encounter chance, and
hands a haul to cargo (leader's pack on `ErrNoCargo`). Effort goes through
`walking.Effort`; the encounter roll is `encounters.Attempt`; supplies through
`company.CompanyItemCount/SpendCompanyItem`. Camp fire rule: a fire needs a
firewood room (free) or one dry bundle; a damp bundle needs two tries and
lights a smoky fire with light but no warmth (`Camp.Damp`,
`RoomWarmedByFire`). New items 40-44 (firewood, damp firewood, fishing line,
raw fish, bitter weed) and grilled fish (30024, Dunmar hearth, cooking 1);
firewood and line sell at the Dunmar and Old Kings Road markets; 152 rooms
tagged (Dark Forest, Frost Lake, Fernhollow, Old Kings Road, tutorial).
Help: new `help gathering` (indexed under `road`; aliases gather, herbs,
firewood, fish, hunt, snares, deadfall, picked-clean) and updates to
resources, camp, forage, cooking, survival and webclient; tutorial hints in
the Survival and Camp lessons. Tests: rules, module, wiring through
`plugins.Load` and `TryCommand`, camp fuel, the three seams, help.
Design: [40a2](designs/2026-10-05-phase-40a2-gathering-design.md).
Decisions (builder, owner delegation): (1) room 2002 (the lightning-split oak)
and tutorial room 905 are firewood rooms, so the existing camp flows and the
tutorial keep working with no change to the starter kit; (2) ephemeral
(tutorial) rooms use a memory-only ledger and never deplete; (3) the encounter
roll happens after the haul, so an ambush never costs the work; (4) the damp
first try is in memory only (a restart gives another first try); (5) the
Forage specialist stands in for "Ranger" and a knife is any bladed weapon;
(6) raw fish has no spoilage (the game has no spoilage mechanic). Not folded
in: the 40a follow-up that a waterskin is destroyed by its last sip (belongs
with consumables, still open). Follow-ups: regrowth does not push a GMCP
redraw (the panel updates on the next room refresh); S1 `depleted` art
replaces the slashed dot. Browser check: `/mnt/project-files/screens/40a2-gathering-panel.png`.
Review (Opus review thread), exploit search against the 36c economy:
**Accepted:** (1) gather-and-sell: with an axe and a Field Smith a room gave
16-20 firewood bundles in 80 seconds, each sold back at about 3-5 gold to a
market whose stock drifts back every round, several times what a fight pays
at low level; lines and bundles also carried a haggled 1-gold margin from
Dunmar to the road post. Markets now take `SupplyOnly: true` (sold, never
bought back; `market` shows "never", `market sell` says so) for firewood and
fishing lines (`TestMarketNeverBuysBackASupplyOnlyGood`,
`TestShippedFirewoodAndLinesAreSupplyOnly`, `TestBidForStockClosedForSupplyOnly`).
Hunting (about 6 gold a hunt, 2 a room per 40 minutes) and herbs (about 3
gold a pick, 3 a room) stay sellable: fair. (2) Fishing lines were sold only
in Dunmar and at the road post, yet the help and refusal said
"provisioners"; the old fisherman at Frost Lake (where the fishing rooms
are) now sells them at 12, above every market. (3) UI: `camp` and
`camp status` now say what a fire would burn (free deadfall, N bundles, damp
only, or none and where to get some) before `camp fire` fails, and a damp
fire's status says it gives no warmth (`TestFuelLineSaysWhatAFireWouldBurn`,
`TestCampCommandsShowFuelUntilTheFireIsLit`); a bare `gather` says when a
picked-clean resource regrows. (4) The memory-only ledger for tutorial
copies grew with every gather and was never read; it is gone, and tutorial
copies record nothing. **Checked, no change:** `camp fire` stays free in the
tutorial camp (905) and at the starter road's oak (2002), where the 44 smoke
script and the tutorial flows camp; inns are untouched; everywhere else a
fire now needs a bundle, as designed. **Rejected:**
the tutorial's never-depleting rooms let a new character leave with one
load of herbs before playing (bounded by carry weight, once); the 40a
waterskin follow-up is not small (an item at 0 uses is refilled to full by
`Validate`, so an empty skin needs a new representation; left for 43a camp
consumables). Master fix carried with this merge: 37b's goblin shaman and
38b's summoned Angel both used mob id 95, so loading the world panicked
(`TestShippedEncounterContentIsValid` red on master); the shaman is now mob
97. Follow-ups: a gather in progress shows only in text (no
GMCP/panel progress); regrowth still does not push a redraw.

**Phase 40s2 + 40s3 reviewed (2026-10-06):** Art paths match what the 40f
battle screen loads (`battle/units/<key>/idle.png` with `frame`, `frames` and
`frame_ms` from the manifest, `battle/backgrounds/<id>.png` for all 11 scene
ids, and the `unknown-humanoid/-beast/-large` race fallbacks). Accepted and
fixed: (1) L and XL units covered three formation cells at 1x and hid whole
units behind them; they are now drawn at 96/128 and shrunk 3/4 to 72/96 with
the outline re-closed (`roster.shrink`), so they stand about 1.5x a person and
the test checks each size class's frame; (2) the `lit` status was a white star
read as `cold`, now a warm brass lantern; `staggered` is a plum spiral (was a
white one like `hidden`); `hobbled` gained a chain and iron ball so it reads
apart from `hamstrung`; (3) the forest backdrop's light shafts were bright
grey dither crossing the units, now sparse green. Rejected: redrawing large
units natively at 72 (timeboxed; the shrink keeps their silhouettes). UI
follow-ups for 40f/40g: when a large unit stands in front, the client should
still mark units it hides (draw their status and health pips above it, or
ghost it); the crocodile is 71 px long and fills its lane. Gates after
merging master (40a, 40f) pass; one full race run failed
`TestSpellEventsThroughTheRealRound` ("mm never went off", modules/company)
and it passed 5 of 5 reruns and the next full run, so it is a flaky 40e test
to harden. Battle screen with the real art: `/mnt/project-files/screens/40f-battle-art.png`
(the browser check needs HTTP, since `file://` skips the sprite manifest).

**Phase 40s2 + 40s3 built: art sets S2 (terrain) and S3 (battle) (2026-10-06):**
`make sprites` now also writes 179 files: S3 battle art (16 backgrounds
320x180, 56 battle units with 4-frame idles, 7 formation markers, 47 status,
role, morale and condition icons) and S2 (19 terrain biomes x 3 variants, 5
animated overlays, fog, unknown and night-mask tiles, 27 landmark overlays).
`battle/mapping.json` is the client key table (mob name -> unit id, biome ->
background, zone overrides, race fallbacks); the manifest lists each unit's
size class, family, names and baseline. 55 shipped mobs gained `sprite:` keys
(the field is read by 40f; unknown to master until #35 merges, which ignores
it). Review the art in [the contact sheet](verification/40s-s2-s3-contact-sheet.png).
Tests (`go test ./scripts`): spec coverage, unit anchors (feet on row
frame-2, 4 distinct frames), opaque backgrounds with a quiet ground band,
quiet terrain tiles with a flat margin and no outline color, mob `sprite:`
keys name real units. Class battle sprites now have 4-frame idles (ranger
arrow nocked low, cleric shield, wizard staff glow, witch grave-mist) and the
S0 style-battle frame uses the real ogre and goblins. Nothing is wired into the
client yet (40f, 40c, 40g do that), so no help or tutorial change is due.
Decisions (owner delegated): (1) the palette is already 64 colors, so snow and
ice use steel/slate/bone ramps and no color was added; (2) terrain animations
are 4-frame transparent overlays (`overlay: true`, `over:` names the tile) drawn
on the roomId-chosen variant, so variants survive; fog and the night mask use
hard-alpha dither, since the tests forbid partial alpha; (3) Winded has no sweat
drops (art direction bans them); (4) `shadow-master`, `guard-royal` and
`ruffian-dangerous` differ from their base in gear as well as color; (5) `bats-echo`
is a floating swarm with no ground baseline; (6) tile and unit feet are aligned in
`roster.py` so every grounded unit rests on row frame-2. Known soft spots for
review: L and XL units overlap neighbours in the 3x3 at 1x (the 40f client can
scale them); several status icons (hobbled, lit, staggered) are plain at 16 px.

**Phase 40f built: battle screen in the web client (2026-10-06):** a battle
opens as a picture (`window-battle.js`): the company left, the enemy right,
each in its 3x3 formation on a 320x180 canvas scaled by whole numbers, over
a biome backdrop (dimmed in the dark). It reads `Company.Battle`, `Company`
and `.Vitals`, and the 40e event feed (hit and heal flashes with numbers,
statuses, chant marks, falls, yields, the outcome held for 3 seconds).
Company health is exact; enemy health is five bands. Retreat and focus send
the dock's commands; Minimise leaves a badge; a setting (`Open
automatically`, kept in `localStorage`) and the Combat tab's "Battle screen"
button cover manual mode. Server: `Company.Battle.enemies[].sprite` (the mob
spec's new optional `sprite:` key, else `unknown-humanoid`, `-beast` or
`-large` by race). Help: `help battlescreen`, linked from `help combat` and
`help webclient`, and the Combat tutorial lesson. Browser check:
`scripts/browser/battle-check.mjs`; screenshot in the project files
(`screens/40f-battle.png`). 40e follow-ups done: the roster refreshes from
every `Company.Battle` snapshot (a fight that grows shows its newcomers);
every `?` is one unseen presence; a `?` caster's cast shows no spell name;
the chant mark lasts until `cast-complete`, so the lagging spell results
(which arrive with the line after the cast) flash when they come.
Decisions (delegated): (1) **Art:** S3 is not drawn yet, so figures are
drawn in code (class hues, a beast and a humanoid shape, unseen shadow) and
the screen loads `battle/units/<key>/idle.png` and
`battle/backgrounds/<id>.png` as soon as `manifest.json` lists them, with
no code change; (2) shipped mobs get no `sprite:` keys yet, since the S3 key
table is not drawn and the race silhouettes cover them; the keys come with
S3; (3) **allied reserve formations** are left as a follow-up: the 40e feed
has no allied relay (a fight has one leader), so the half-scale view would
have no events to animate; (4) per-member Company.Conditions icons are not
drawn (statuses come from events, which cover enemies too); (5) the
tutorial-zone and Stormwatchers Keep backdrop overrides wait for S3's
`training-yard` and `ice-keep` art; (6) the screen is a floating panel, not
a modal, so the terminal stays usable under it.

**Phase 40f reviewed and merged (2026-10-06, PR #35):** the review
confirmed the screen sends only `retreat` and `company tactics focus
[rule]` (both allowed mid-battle, as the Combat tab), and hides what scout
hides (enemy health in five bands, hidden foes skipped, every `?` one
shadow, no spell or status for it). Fixed: (1) enemy figures faced away
from the company (head, eyes and weapon mirrored the wrong way); (2)
members who fled or were separated still stood in the picture (now
filtered as the Combat tab does; dead `&& false` code removed); (3) a
`Company.Battle` snapshot arriving after `fight-end` cleared the outcome,
and one after the hold could reopen the finished battle and show "The
battle is over" again (an `ended` flag now holds until the battle clears or
a `fight-start`); (4) UI: a caption hint says to hover or tap a figure, a
Help button sends `help battlescreen`, and status ticks (bleeding) flash
their damage. Regression checks added to `battle-check.mjs`. Rejected: the
design's "unknown-* at 50% per enemy in the dark" was replaced by one
shadow on purpose (the 40e rule that `?` foes are one presence). UI
follow-ups for 40g: show the round and a short "last blow" line so who hit
whom reads without hovering (after the 35e merge the focus buttons also offer `healers`, as the Combat tab does); a named outcome reason (all fell, the company
withdrew); role letters are blurry at the canvas font size; a `?` presence
in a lit battle stands at front centre and can overlap a visible foe, and
stays until the battle ends; morale (nerve) is not drawn yet. Flaky test seen: `TestBalanceMirrorClericIsACasterWhoCastsNothing`
(`modules/company`) failed once in the full race run (1 of 5 foes aimed at
the cleric) and passed alone five times and in a package re-run; this PR
touches no company code, so it is left as a follow-up to make robust.

**Phase 40a complete, merged via [PR #32](https://github.com/Robinsond76/ashveil-gomud/pull/32): room resources (2026-10-06):** rooms carry a validated
`resources` list (water, forage, shelter; herbs, firewood, fishing and game
are accepted in data but hidden until 40a2). `look` prints a "Here:" line,
GMCP `Room.Info` and `World.Map` send `resources` (omitted when none), and
the web map draws a coloured corner dot per resource with a tooltip row and
an on/off setting (S1 sprites replace the dots later). Rules: `drink water`
or `drink source` (40 Thirst plus Hydrated, free, refused in battle), `fill`
and `company fill` (waterskin gains `refillable: water`; pack, companion
packs and cargo), `company drink` at a source waters everyone free, a camp
rest at a forage room gets +1 find, and a shelter room halves the weather
rest penalty. 291 default-world rooms are tagged (lakeshore, waterfall,
Fernhollow trough and the tutorial Weather Yard water; forest and island
forage; caves, keeps and lodges shelter). Help: new `help resources`
(indexed under `road`, aliases water, fill, spring, shelter), updates to
drink, survival, forage, camp, company meal and webclient, and a Survival
lesson hint. Design: [40a](designs/2026-10-05-phase-40a-room-resources-design.md).
Decisions (owner delegation): `drink water` yields to an item exactly
named "water" and keeps its old waterskin meaning away from a source (no
surprise for existing habits); a source drink also gives the Hydrated buff
(same as a waterskin glug); `company fill` refills part-used cargo by
withdraw-then-deposit through the existing cargo API (no new cargo
interface); the shelter bonus never beats a full rest. Review (2026-10-06, Opus):
the three builder decisions are kept. Accepted findings: (1) forage and
shelter were tagged on ~225 rooms where no camp can be made (only two
default rooms admit `camp`), so the markers promised a rule that could not
act; rooms now show them only where the camping module says a camp can be
made (`rooms.SetCampableCheck`), keeping the data for when 41/42 place
camps; (2) `company drink` at a source gave one glug each, so a parched
member stayed thirsty beside free water; members now drink until no longer
thirsty (at most three); (3) the Room Info panel did not show resources;
it now has a badge per resource that opens `help resources`. Help updated
(resources, webclient). Rejected: withdraw-then-deposit can lose part-used
cargo if the deposit's save fails right after a good withdraw (logged as
an error; same failure mode as other cargo moves). Browser check: dots,
tooltip, toggle and badge render and read clearly
(`/mnt/project-files/screens/40a-map.png`). Follow-ups: a waterskin is
destroyed by its last glug, so `fill` only tops up part-used skins (keep
an empty refillable container); the water dot on blue shore tiles is
low-contrast until the S1 icons land; the root package's
`world_party_follow_test.go` (already on master) leaves an ignored
`config-overrides.yaml` and two user files in the default world, which can
break a later `internal/usercommands` run that loads that world (the 40a
battle test no longer loads it).
Verification: `make generate`, `make validate`, `go test -race ./...`,
`make js-lint`.

**Phase 40s1 built: art sets S0 and S1 (2026-10-06):** `make sprites` runs
`scripts/sprites/generate.py` (Pillow) and writes 51 PNG/GPL files under
`_datafiles/html/public/static/sprites/` plus `manifest.json` (frame size,
frames, rows, anchor, timing per file). S0: 64-color palette (`palette.png`,
`palette.gpl`), the two style frames, proportions sheets and icon sample.
S1: 9 resource icons, markers, camp pieces, map units (idle and walk, down /
up / side) for the 6 base classes plus `adventurer`, and app icons. Review
the art in [the contact sheet](verification/40s1-contact-sheet.png). Tests:
`go test ./scripts` checks the spec layout, palette-only colors, hard edges,
the feet-baseline anchor and that committed PNGs equal generator output.
Nothing is wired into the client yet, so no player help or tutorial change
is due (40b, 40f and 40i wire it). Decisions (owner delegated): the
review thread approves S0 against the art direction (roadmap); `palette.png`
is an 8x8 image, one pixel per color; the here-ring has no outline (a 1 px
ring would double); the ogre and goblins in `style-battle` are throwaway mocks
for S3 to replace; battle idle shows one frame, since S3 owns full battle
sheets; app icons are scaled by whole numbers from a 64 / 48 / 32 grid (the
192 px version drops detail rather than just downscaling).
**Review (2026-10-06):** S0 approved against the art direction: muted
palette, adult 5- and 7-head proportions, grounded gear, natural map. Fixed:
battle warrior was a red slab with a raised sword (now narrower, split
surcoat, shield forward edge-on, sword low per the S3 pose); battle rogue was
one charcoal block (now crouched, leather breeches, reverse-grip blades); far
legs shade one step darker in battle; committed `__pycache__` removed and
ignored. Accepted: contact sheet lives in `docs/verification/` rather than
`sprites/contact/` (keeps review images out of shipped assets); manifest
carries what 40b (paths, rows, anchor, baseline, timing, `adventurer`
fallback), 40i (sizes, maskable safe zone) need; S3 extends it for 40f.
Follow-ups for S3/40s5: real goblin and ogre art drawn side-on (the mocks face
front; goblins must look feral, not comic); 4-frame battle idle; ranger
"arrow nocked low", cleric shield and wizard stone glow poses; a 1x
readability pass on the 3x3 formation with enemy mirroring; a darker
battle-ground band so unit feet and shadows read on dirt. UI check: the map
set gives a clear, readable class marker and camp state at 32 px; the gap is
that S2 terrain must keep tiles quieter than units, and 40b should draw the
here-ring and badge above terrain so the player's sprite never blends in.

**Phase 37b complete, merged via [PR #39](https://github.com/Robinsond76/ashveil-gomud/pull/39) (2026-10-06):** encounter follow-ups.
Enemy healers: a Dark Forest goblin shaman band; both pilot zones now hold
healer groups at one in six of their table (cap stays 20%). Boss respawn: a
company, and its party, finds a lair quiet for 30 real minutes after beating
the boss (saved; fleeing starts no wait), so lairs cannot be farmed. Zone
band: `look` and `scout` name it and rate it against your level (easy, fair,
risky, dangerous), GMCP `Room.Info.levelband` and the web client room header
("Lv 5-7", coloured) show it. The unused composition `kind` field is removed
(re-add with the scent-masking consumable). Tuning: `SkillEdgeSpan` 14 to 8
makes a zone above the company's level hard (five levels under: 87% to about
50% wins); bosses are 3 levels over and 2x HP. Measurements:
[37b measurements](plans/2026-10-06-phase-37b-measurements.md). Decisions
(delegated, with reasons): span 8 over 10 because it matched the owner's
8-vs-10-12 fair and 8-vs-13-15 hard gradient best without lengthening
at-band fights; boss +3/2x over +4/1.5x because those fights ran 3 minutes;
cooldown 30 minutes, party-wide, starts on the boss's fall, because a
fled fight should not lock the lair; `kind` removed because nothing reads it.
Deferred: tier-appropriate gear in the harness (needs item tiers in the
mirror), durable groups, bad-luck protection.

**37b review (2026-10-06):** accepted and fixed: (1) a member who beat the
boss could join a party led by a fresh character and farm the lair again,
because the quiet was checked for the leader only; it now holds while any
party member's quiet runs (regression test). (2) Only the owner heard that
the lair fell quiet; every member now does. (3) A player in a quiet lair had
no way to see why or for how long; `look` and `scout` now say how many
minutes remain, and `help encounters` says so. (4) The admin config hint
still said the span ships at 16. Verified, no change: the narration golden
differs only by the span (regenerated at span 14 it matches master byte for
byte); the span 8 gradient holds at higher bands (6-fight samples: band
25-27 and 35-37 at-band 100% wins, two under 79-83%, five under 41-45%), so
casters' lower rates saturating the edge at high levels does not break
at-level fights. Follow-ups: the rating uses the player's own level, not a
company average; the web header's rating refreshes only on the next room
update after a level-up; boss quiet is keyed by composition id, so two zones
sharing a lair id would share it; the new goblin shaman borrows the
goblin-hexer battle sprite until its own is drawn (art follow-up).

**Phase 37 complete, merged via [PR #29](https://github.com/Robinsond76/ashveil-gomud/pull/29) (2026-10-06):** Dark Forest (band 5-7) and the
Catacombs (band 10-12) spring 2-3 foe battles (sometimes 4) in opted-in
rooms, on steps and journey arrivals, with a boss lair (boss at band low+2,
1.75x HP, no strategy, 2-3 escorts, `boss: true` flagged). Levels follow the
zone band, never the player. Healer groups are about one in five and never
in four-foe or boss groups. Grace after a battle (2 eligible entries and
30 s) is saved per leader. Wins drop zone loot, a once-per-group cache and
boss rolls, personal per company, with a Spoils line in the battle summary.
`help encounters`, scout danger line, tutorial hint. Plan:
[37 plan](plans/2026-10-06-phase-37-random-encounters.md); balance:
[37 measurements](plans/2026-10-06-phase-37-measurements.md). Deferred:
bad-luck protection, smart loot, autoloot filters, durable (restart-proof)
groups, content migration to other zones, boss respawn clock.
Review (Opus review thread): accepted and fixed: (1) every eligible step
saved the grace registry to disk on the game loop, now only when the grace
changes; (2) spoils noted by a death outside any fight leaked into the next
fight's Spoils line, now cleared when a battle begins; (3) UI: players had
no way to see a zone's level band, so `scout` in a dangerous room now names
it (help updated). Each has a regression test. Rejected: a battle's end
removing every ownerless group at once, not only its own (they would vanish
within the 120 s timeout anyway). Follow-ups: a boss lair can be re-rolled
every few entries for a guaranteed Rare (boss respawn clock, already
deferred; 37b), the band shown in `look`/web client zone header, and the
composition `kind` field is unused by drops (goods come from each foe's
`lootcategory`).

**38c-d elite routes design (2026-10-06):** rank tables 30–60 for the
thirteen elites the faith routes design didn't cover (Warlord; Pathfinder,
Swordmaster, Nightblade; Sentinel, Marksman, Ravager; Archon, Archmage,
Necromancer; Wise One, Coven Mother, Crone of Ash), shared elite rules for
all eighteen (one elite per advanced route, the faith routes' gate wait for
every lineage, catch-up ranks for a late promoter, no cost, three elite
talents per lineage from 35) and how players see it (readiness and
gate-wait lines, preview, `class`, company markers, GMCP fields, battle
narration, `help elite` and a page per route). Every open question decided
under the owner's delegation, with reasons in the
[design](designs/2026-10-06-elite-routes-design.md). 38c splits into 38c1
(framework, UI, warrior and cleric elites), 38c2 (rogue, ranger) and 38c3
(wizard, witch): [plan](plans/2026-10-06-phase-38c-elite-routes.md). 38b's
plan wasn't published yet, so each slice reconciles with the advanced
signatures 38b ships. Documentation only. Verification: links and the
diff checked.

**Remaining roadmap planned (2026-10-06):** the owner handed over the rest
of the roadmap, delegating design approval and the visual direction. The
[remaining roadmap](plans/2026-10-06-remaining-roadmap.md) maps every
unmerged phase with its dependencies and adds 35e (focus the healer), 37b
(encounter tuning), 36d, 38c-d to 38e, 39i, 40a4 (camp theft), art sets
40s1–40s5, 41–42 (world building), 43a–43b (camp consumables, weapon
poisons) and 44 (live smoke playtest). Can start now beside 37 and 38b:
35e, 40e, 40a, 40s1, 44 and the 38c-d design. Visual choice: the battle
screen lane first, with code-generated pixel art. Open questions in the
40a–40g and loot designs are decided there, each with a reason.
Documentation only. Verification: links and the diff checked.

**Phase 35e complete: focus the healer (2026-10-06):** a `healers` target rule
(a chanting healer, else the weakest idle healer, else the casters order) and
focus. While the player has set no focus and the leader is level 5 or more,
the company goes for an enemy healer it can reach first (`strategy.HealersDefault`,
`enemyparty.RuleVs`, wired into `attack` aims, the round upkeep re-aim, and
target reassignment); an explicit focus, `none` included, or a focus called in
the battle wins. Player-facing: `company tactics` and its `default` reply say
the healers default is in force, the leader is told "Your company marks X as a
healer and goes for it first", GMCP carries `healers_first` (company tactics
and battle), the web client's focus bar gains a `healers` button and a note,
`help tactics` and `help strategy` document the rule, `focus the healer` and
`healers focus` are help aliases, and the tutorial's tactics hint mentions it.
Decisions: the override applies only when a healer is reachable, so a member
who can't reach it keeps the level's default instead of the casters order;
the default is dynamic (no healer standing means the usual default). Tests:
rule order, default ladder, live `attack` aims at levels 3-12, set-focus and
`none` overrides, the battle line, tactics text, GMCP payload, help.
Balance (16 fights a cell, `ASHVEIL_BALANCE=1 TestBalanceCoordinated`, default
company against tiers 1-3, each with a healer): level 1/5/10/30 wins were
69-81/62-81/50-100/38-69%, in line with the pre-35e rows; the one failed
assertion (level 30, tier 3, 37% against the 50% target) is within the noise
of 16 fights and was not re-measured.
Review (PR #33, Opus review thread): **accepted** the battle-start gap: the
"marks X as a healer" line came only on a later re-aim, never when `attack`
opened on the healer, so most fights showed no reason; `attack` now says it
once when the leader's or a companion's opening aim is an enemy healer
(`TestHealersDefaultSaysSoAsTheBattleOpens`). **Confirmed, no change:** the
mid-battle focus is the 30c owner decision 2 exception to the Ogre Battle
rule (company-wide, that battle only, once a round); 35e only makes such a
focus end the healers default. No half-applied work found in the diff.
**Balance re-measured** at 60 fights a cell, branch against master: level 30
tier 3 (answer in kind) 58% against master's 53%, tier 2 62% against 45%;
the 37% was noise and both runs pass every assertion. Level 5 and 10 tier 3
cells moved within the sample's noise (55-68% against 67-70%). Rejected: the
healers default also overriding a member's own `strategy` rule at levels 5-9
is intended (company-wide default; `company tactics focus none` opts out,
as `help tactics` says).

**Phase 36c complete: loot economy (2026-10-06):** merchants buy rolled gear
(priced by quality, unread Rare+ by rarity at a discount), `mark [item] junk`
and `sell junk`, `salvage` at smiths, identification fees at `appraise`
(60/150/400), all 24 trade goods in the Dunmar and Trappers' Post markets
with stock-driven saturation, and GMCP/web labels for rolled names. Plan:
[36c plan](plans/2026-10-06-phase-36c-loot-economy.md). Decisions and
deferrals (scrolls) are recorded there.
Review (PR #37, Opus review thread), exploit search: buy-and-sell of gear
can't pay (shops sell plain items; merchants pay at most 25%); no shop-sold
piece salvages into more than its price even at Dunmar's market price
(scanned every shipped item); appraise-then-sell is a gold sink, not a gain
(a Rare's fee exceeds what reading adds at tier 1). **Accepted:** Brynja
sold iron ore and tanned leather (18/16) under Dunmar's target-stock price
(23/21), a risk-free loop; her prices are now 24/22 and
`TestShopkeepersNeverUndercutMarketsOnTradedGoods` holds every shipped
shopkeeper at or above every market's target-stock price for a traded good.
**Accepted (UI):** `offer` and `sell` now say why a price is low (unread
gear, or a pile already on hand; `TestSaleNoteExplainsALowPrice`); GMCP
inventory details carry `junk`, so the web gear window shows the mark; help
`sell` no longer claims paid reading "usually pays", and `goods`/`market`
no longer say saturation recovers "as the days pass" (market stock drifts
every round, so a glut clears within a minute). **Confirmed fair:** the two
"hide" test updates (bear hide made bare "hide" ambiguous; the tests now
assert the question and use "wolf hide", same prices). Follow-ups: market
saturation is weak because stock drifts every 4-second round (37b or a
market pass could slow drift for the 36c goods); a `salvage` preview of
what an item would give; legacy weapons with no family (sharp stick, tree
trunk, sling) salvage as metal; rarity colours in web windows.
Verification: `make generate`, `make validate`, `go test -race ./...`,
`make js-lint`.

**Phase 36d built, reviewed and merged from [PR #120](https://github.com/Robinsond76/ashveil-gomud/pull/120) (2026-10-06): tier 4-6 gear, Legendary signatures and Set bonuses.** Tier 4 and 5 plain gear for every family (drops only), 16 named relics of tier 4 to 6 (items 50001-50033: 7 Legendaries with a signature and two rolled affixes, 9 Set pieces with none) across three 3-piece sets from five bosses, dropped by bosses at per-relic chances with bad-luck protection (forced on the 20th relic-less kill, saved on the character). Relic effects merge into `ClassEffects()` so combat reads them with no new plumbing; `help relics`, an inventory "Relics worn" panel, GMCP `relic`/`relic_lore` and the gear tooltip show them. Plan and decisions: [36d plan](plans/2026-10-06-phase-36d-relics.md). Economy (owner ruling: treasure pays): relics sell and salvage, valued 705-1750 gold; plain tier 4+ pieces trim their runestone when it would out-pay the piece. Follow-ups: smart loot bias, collection log, rarity colours in other web windows, Scribe rank 4 naming the boss, tier 6 plain bases, zone placement of relic bosses (Phase 42). **Review (accepted, fixed):** (1) a Divine Shield from a relic (Abyssal Bulwark) never held on a classless wearer struck before its first action, because `ShieldBlow` needed runtime state the class code had only made on acting; it now makes it (`TestRelicDivineShieldHoldsBeforeTheWearerActs`). (2) `ClassEffects` decided whether to redo the gear merge by the class map's pointer, which a recycled allocation could match; the merge is now dropped whenever the class's own effects are remade (`TestWornRelicsFollowTheClassAsItGrows`). (3) UI: the Company window's gear and cargo rows (where a relic is handed to a companion) said nothing about a relic; their tooltips now carry the signature or set bonuses (`relic` on `Company.Inventory` items). (4) `help relics` said "a few percent a kill" while bosses with several relics drop one in 4 kills; it now gives the real range (one in twelve to one in four), held by the shipped-data test. (5) Added `TestRelicHealthPctRaisesMaximumHealth`. **Checked, no change:** every gear-effect key has a live reader (skill Attack/Evasion, combat damage, armor, block, parry, Validate's health, heal and spell power, blow multipliers, Second Wind pass, row and company auras, Guard Fall), for classless wearers and companions alike; all five relic mobs are flagged bosses; no code scales foes by gear, so relics add power without becoming a difficulty lever (difficulty stays zone level). Relics sell and salvage for profit per the owner's 22:35 ruling. Chromium check (`scripts/browser/relic-check.mjs`): gear tooltip rarity colour, signature, set bonuses and lore at desktop and 390px, fitting the screen, and the Company row tooltips. Kept: phone users read relic text through Look (no hover); the gear window's row names are not rarity-coloured (the plan's deferred web-colour follow-up).

**Phase 40e complete, merged via [PR #30](https://github.com/Robinsond76/ashveil-gomud/pull/30): structured combat events (2026-10-06):** the web client
now receives `Company.Battle.Event`, one entry per combat happening of its
fight (attack, spell, heal, status, wind-up, guard, yield, flee, death, fight
start and end), released in step with the paced narration. No visible player
change, so no help page. Data entries ride the combat pacer's queue with the
text (flushed with it, sent at once with pacing off) and the module reuses
`Company.Battle`'s IDs. Decisions: the leader's own fight only (allied
companies each have their own fight, so no allied relay yet); fight-start
lists the roster, not cells, because `Company.Battle` is the one source of
cells; in the dark or for a hidden enemy every enemy ref is `?` and its
statuses are dropped, instead of tracking which enemies the narration
labelled; secret statuses are never sent. Design:
[40e](designs/2026-10-05-phase-40e-combat-event-messages-design.md).
Independent review (Opus, 2026-10-06): accepted, the leader's ref was
`me`, which matches nothing in `Company.Battle`; it is now their member key
`leader` (`me` only without a company), with a unit test and a real-round
check that the fight-start roster matches the formation's cells; an unused
viewer field removed. Rejected: data released one line after a spell's
narration (by design: an event rides the next line). Left for 40f: an
unseen enemy's cast still carries its spell ID (cast lines already hint at
the spell); 40f should not show it for a `?` caster.
Checked and sound: no health or secret status in the feed, dark/hidden
masking matches `Company.Battle`, flushes carry the data, pace off sends at
once. 40f follow-ups are in the design's Built section.

**Phase 36b complete: gear catalog (2026-10-06):** the first tier 1-3 catalog
(swords, axes, maces, short and war spears, glaives, staffs, bows,
crossbows, four armor paths, shields), 24 trade goods with value-per-kg
bands, a tier for every shipped weapon and armor piece, new `family` and
`goods` item fields shown on `look`, tier 1 stock at Frostfang Armory and
goods at the general trader, and `help equipmenttiers` and `help goods`.
Plan: [36b plan](plans/2026-10-06-phase-36b-gear-catalog.md). Independent
review accepted: a duplicate `padded shoes` name (catalog piece renamed,
uniqueness test), shield ladder inversion (shipped iron and tower shields
audited tier 3, ladder test), Ivar's 20-variety limit (stock cut to 12),
help claiming tanners and not saying goods do not drop yet. Rejected:
shipped uniques whose tier exceeds their armor number (tier counts stat
mods too; no mechanic reads tier). Deferred: harness cells in tiered gear
(Phase 37), goods in markets (36c). Merge review (second, independent)
accepted: `help equipmenttiers` said reach lets the second rank strike
(reach is target depth, not wielder rank; fixed with the iron war spear
text) and the hunting crossbow called itself slow before 39h's reload.
Upheld the rejection above: the tier 3-4 uniques carry large stat mods.
Glaive and crossbow data match the neutral classes design; note for 39a,
its kit's "padded jerkin" is the catalog padded jack (20163). 35b's flaky
`TestBattleEndPatchesTheCompany` was fixed in the 35d merge review.
Verification:
`make generate`, `make validate`, `go test -race ./...`, `make js-lint`.

**Phase 38b complete: promotions and talents (2026-10-06, PR #34):** class
promotion at level 10 and 30, talents at 5/15/25/35/45/55, the six lineages'
routes, and the faith routes (Priest and Hierarch with an Angel, Blood Priest
and Demonologist with a Demon, Knight and Paladin, Blackguard and Dread
Knight), with extension points for the neutral classes 39a-39h. See the
[plan](plans/2026-10-06-phase-38b-promotions-talents.md). Terror, Soul feast
and Hellfire rank texts were reworded to what is built.

Review (independent reviewer, fixed with regression tests): Bless never wore
off (now 3 rounds); Siphon cost 13 at every rank (now 10, then 8, free of the
hungry heal tax); Intimidation lasted the whole battle (now the round of the
wound and the next); a summon cast in peace spent the next battle's call (now
battle only); companions summoned against a single foe and ignored the mana
reserve (now 3+ foes or a boss, reserve kept); "10% less damage" auras gave
about 5% (now a true percent off the blow); the Angel stayed when its Hierarch
fell (now departs); two summons of a kind shared a member key (now per
instance); a player kept casting rank spells after a death cost the level
(now locked until regained). UI: the company roster names a companion's class
and flags "promotion ready" and talents to choose; `class paths` shows each
rank's text (15 advanced classes outside the cleric and warrior lines have no
help page of their own). Accepted as is: Lay on Hands uses live in memory,
reset by a camp or inn rest and refilled by a restart (rare, harmless
leniency). Left as follow-ups (below).

Balance (`TestPhase38bClassRoutes`, 100 fights a cell, even 5v5 mirror,
company HP lost per fight): fighting healers vs a Mercenary: L25 Knight 41%,
Blackguard 40% vs 56% (27-28% less, in the 15-30% target); L35 Paladin 38%,
Dread Knight 46% vs 60%; L45 Paladin 26%, Dread Knight 33% vs 54% (the
Mercenary has no elite yet, so the gap is wide there). Aura of Dread was the
outlier (-5 Attack took the Dread Knight to 15% HP lost); it is -1 now.
Summons against a boss mirror: Hierarch 99/98% wins vs Priest 72/67%,
Demonologist 85/93% vs Blood Priest 65/55% (+20 to +38, target +10-20). Halving
summon health or armor barely moved it: the gain comes from enemies turning to
the arriving summon (a decoy), because it holds no formation cell. Settled
(timeboxed) and left to the follow-up below.

Follow-ups: summons stand in a front-row cell as the design says (and aim
stickiness), then re-measure summon wins; Angel and Demon Attack/Evasion rates
(1.0/1.1, 1.1/0.9; both use warrior rates now); Hierarch cleansing on Minor
Heal, not only Greater Heal; the broken binding strikes allies only (design:
nearest creature, friend or foe); Hellfire and Thornhide damage gives no kill
credit and Soul feast fires on any mob death in the room; Quick chant shortens
every hex, and Swift Host and Mastered binding skip the manual cast path; help
pages for the 15 other advanced classes; class and talents in score and the
web client panels. The race suite surfaced the
`TestAnEnemysBleedLeavesALightWound` flake (3 in 40 on master: a crushing
blow's bruise); the test now counts only the bleed's wound.
Verification: `make generate`, `make validate`, `go test -race ./...`,
`make js-lint`.

**Phase 38a complete: the Witch (2026-10-06):** a sixth starting class that
takes enemy turns away. Eight hexes (Slumber to Blight) in the `hexcraft`
school, reach 1 to the whole group by level, three new statuses (Asleep,
Paralyzed, Blighted), a resist roll and a shared immunity so no foe is held
past half a fight, a `controller` strategy role that hexes on its own, a
recruitable Witch, `help witch`, `help hexes` and tutorial pointers. See the
[plan](plans/2026-10-06-phase-38a-witch.md). Balance: the Witch matches the
Wizard in the equal mirror and neither beats a warrior there (about half of
casters' chants break); timeboxed, no retune.
Independent review: accepted (fixed with tests) two stale company tests,
the missing `.d.ts` entries for `CastHex` and `HexTargets`, one shared
immunity for sleep, paralysis and knockdown (the half-of-a-fight rule held
only per status), and Dread Whisper no longer cast at unbreakable foes;
poison waking a sleeper was confirmed and now tested. Rejected: manual
Miasma row limiting (a hand `cast` is a utility outside battle; the help now
says so); "hex the largest group" (a battle has one group); design HP 1.75 a
level (the shipped HP scale is 0.55, noted in the plan).
Merge review (Opus): accepted and fixed with regression tests: blows
against a sleeper were 25 points *less* likely to hit (the bonus was added
to a penalty that is subtracted); bleed and burn tick damage now wake a
sleeper; Dread Whisper skips foes with no temperament (no morale check
happens); a hex finishing on a foe that fell mid-chant does nothing; hidden
foes no longer make the Witch pick a hex it then can't aim. Help corrected:
hexes ignore the mana reserve like heals; Miasma's poison outlives the
fight; the hand-cast sentence is gone, since a harmful `cast` is refused
both in and out of battle (so manual Miasma never reaches players).
Accepted as is: the boss resist waits on encounter data setting `Boss`;
the no-lock ledger covers hex holds only, not a tackle's knockdown.

**Phase 36a loot item model merged (2026-10-06):** gear can roll quality,
rarity, affixes and a level requirement (`items.Rolled`, `internal/loot`,
data in `lootaffixes/`); Rare and better arrive unidentified until a Scribe
reads them (new wizard/cleric skill, `scribe` command, camp-rest reading,
`autoskill scribe`); `Wear` enforces the level requirement; `spawn loot`
rolls items for testing. Players do not see drops yet. Plan:
[36a plan](plans/2026-10-06-phase-36a-loot-item-model.md), with deviations
and deferrals (merchant sale, GMCP names, signature and set effects).
Independent review accepted: identify clobbered later enchants (fixed),
`gearup` proposed refused gear and wore by item id (fixed, uuid wear),
multi-attack quality scaling (per hit), stale scribe rank wiped after
training (CanTrain settles first), inaccurate help (Legendary, Set, sale),
companion name lowercased at camp. Rejected: none outright; GMCP names and
merchant sale are deferred, not fixed. Second (merge) review accepted:
the weight affix never reached load (`Weight()` reads base data; now
applies identified `weightpct`), the warmth affix replaced the slot default
and could make gear colder (now a `WarmthBonus` added after the default),
the scribe refund said Scribe was gone, the level refusal said "you" to
companions, "a exquisite", rolled gear not answering to its shown words,
mob `gearup` dropping by item id, and enchanting losing a roll's quality
value; each has a regression test. Rejected: capitalised prefixed names
("Keen fine sword") are a deliberate style. Verification: `make generate`,
`make validate`, `go test -race ./...`, `make js-lint`.

**Neutral classes design approved (2026-10-05):** the owner asked for a
glaive class and classes with no good or evil path, inspired by Ogre Battle
and Unicorn Overlord. Added the [neutral classes design](designs/2026-10-05-neutral-classes-design.md):
eight new base lineages (Halberdier, Doll Master, Beast Tamer, Gryphon
Rider, Samurai, Shaman, Alchemist, Arbalist) with unrestricted routes,
phases 39a–39h after 38b. The owner approved it and answered all five
questions (doll and beast take a cell, not a slot; beasts never die; keep
Samurai and Kensai; flasks brewed at camp; Halberdier first).
Documentation only. Verification: links and the diff checked.

**Phase 40a3 design approved (2026-10-05):** the owner approved the camp
gear design as written: the fuel rule, the six items, and their weights,
effects and prices. The execution plan comes when the visual milestone
starts. Documentation only.

**Camp gear decisions (2026-10-05):** bedrolls are a bonus only. Camp
theft is deferred as a future feature: a silent camp event when a company
rests without bells and trip lines, noticed as missing loot on waking.
Recorded in the 40a3 design and the deferred items. Documentation only.
Verification: links and the diff checked.

**Camps need firewood; camp gear designed (2026-10-05):** the owner
approved the 40a2 rule that a camp fire needs firewood. They asked for camp
items that boost camping. Added the [40a3 camp gear](designs/2026-10-05-phase-40a3-camp-gear-design.md)
design: one bundle per rest, with embers between rests, and six durable,
weighted items. These complement the single-use camp consumables design.
Documentation only. Verification: relative Markdown links and the diff
checked; no Go tests required.

**Visual milestone decisions (2026-10-05):** the owner answered the
milestone's questions.
- **Sprite art direction:** mature, grounded high fantasy in the Lord of
  the Rings tradition, for a dangerous, unforgiving world. Not anime in
  any way. OB64 sprites are the reference. Recorded in the
  [sprite specification](designs/2026-10-05-sprite-specification.md).
- **Other players** are not on the map for now (deferred).
- **Every room resource is wanted.** The new [40a2 gathering](designs/2026-10-05-phase-40a2-gathering-design.md)
  design covers herbs, firewood, fishing and game, with room pools and
  firewood for the camp fire.
- **A showcase area** may be built for the new features (40d now
  recommends a new zone). World building moves after the milestone.
- **No race variants** for now.
- **An installable web app** is enough.
- **Travel and battle-screen auto-open** are approved.
- **Allied companies on the battle screen:** 40f recommends half-scale
  reserve formations with a read-only swap view, pending approval.

Documentation only. Verification: relative Markdown links and the diff
checked; no Go tests required.

**Phase 40a–40g designs drafted (2026-10-05):** at the owner's request,
added a design for each item of the visual roadmap. These are drafts
awaiting owner approval; no code has changed.
- 40a: room resources, with water wired into survival (`drink water`,
  `fill`), forage and shelter.
- 40b: class sprites, the company badge and camp markers (party camps
  only).
- 40c: terrain and landmark tiles.
- 40d: a tile-ready pilot region and `walkto` (click-to-walk). Merged 2026-10-06 (PR #56).
- 40e: structured combat-event messages released with paced narration.
- 40f: a static OB64-style battle screen.
- 40g: battle animations and effects.

The drafts are linked from the Phase 40 table below. Documentation only.
Verification: relative Markdown links and the diff checked; no Go tests
required.

**Visual client milestone planned (2026-10-05):** the owner chose to keep
the browser client and add a visual layer:
- a colorful tile map with class sprites, camps and room resource icons,
  inspired by MUME's mapper;
- an Ogre Battle 64–style battle screen.

Added the [milestone design](designs/2026-10-05-visual-client-milestone-design.md)
(phases 40a–40i) and a [sprite specification](designs/2026-10-05-sprite-specification.md)
listing every sprite in art sets S0–S7, for another agent to generate. The
milestone starts after the current phase sequence. Other companies' camps
stay hidden; PvP camp visibility is deferred. Documentation only.
Verification: relative Markdown links and the diff checked; no Go tests
required.

**Phase 35d complete: combat feel (2026-10-06, merged via PR #26):** every landed
blow is glancing (x0.5), solid or telling (x1.4) by a roll the skill edge
shifts; `ToHitEven` is 88; one-round heals (Minor Heal, Tend Wounds) break
only on heavy force, for both sides; the patch threshold is 80% and settable
(`company tactics patch`); level defaults (weakest from 10, casters first
from 25, first companion warrior guards the first healer) apply until the
player sets their own, and a stored focus of `none` stays a choice. Shipped
help (attack, evasion, interrupts, tactics, patch, heal, combat, guardian,
health), keyword aliases and tutorial hints. `HPAfterFull` is **0.3**, not
the design's 0.4: 0.4 breaks the 1.6x level-60 cap. Measurements and the
settled misses (dead swings 35 to 39%, six-fight mana run, boss wins above
target, rounds are 8 s not 4 s) are in the
[measurements](plans/2026-10-06-phase-35d-measurements.md). A pre-35d player
who stored only `focus none` now gets the level default (accepted).
Independent review accepted: GMCP/summary tactics ignored the level default
and patch (fixed, tests); an armor-absorbed blow reported a quality (fixed,
test); equipment role advice ignored the default guard (fixed). Rejected: a
battle-view quality field (the combat lines carry the word), partial
blow-quality config (zero shares are a documented "off" used by tests), and
glancing rounding on tiny dice (negligible).
Merge review (Opus): found the cause of the known battle-end patch flake,
a real bug: an archer still taking aim when the last foe fell kept a stale
aggro until the next round, so the battle-end patch (and `company patch`)
saw the company as still fighting and skipped. Aggro at a fallen or absent
foe no longer counts as fighting (`hasLiveFoe`, regression
`TestPatchIgnoresAimAtAFallenFoe`; the old test passed 150 of 150 runs).
Fixed a tutorial hint that promised the weakest focus below level 10. The
settled misses were judged sound and not blocking: the mana run now holds
four fights at 86% or better (35b held three), and the rest is left to 37's
encounter pacing and rest placement.

**35d combat feel designed (2026-10-06):** at the owner's request, a second
opinion on the 35b balance misses traced them to rules, not tuning: about 45%
of swings produce nothing, one-round heals break on any touch, patching stops
at 50% so the fourth fight is lost half the time, the enemy's coordination
ladder has no company counterpart, HP flattens after 20, and boss groups run
the band's tier against the "no strategy" contract. Recorded in the
[second opinion](plans/2026-10-06-combat-rebalance-second-opinion.md); the
[35d design](designs/2026-10-06-phase-35d-combat-feel-design.md) and
[plan](plans/2026-10-06-phase-35d-combat-feel.md) propose blow qualities,
resolving heals, an 80% patch threshold, level-keyed company defaults,
`HPAfterFull` 0.4, short bosses and seconds-and-lines targets. 35d depends on
35b only and must merge before 37 and 38b. Documentation only. The owner
approved it the same day, accepting the default for each open question, and
noted that enemy healers can be uncommon because unbreakable enemy heals
raise difficulty notably (frequency left to the implementer). Verification: relative Markdown links and the diff checked;
no Go tests required.

**Roadmap and designs (2026-10-05):** the owner reviewed the game's
progression, balance, loot and activities. Recorded the owner's decisions:
- the encounter contract (2–3 under-level foes, occasionally 4; bosses of up
  to 5 with no strategy);
- every level should matter;
- casters with less fizzling and more mana;
- stronger healing;
- a Witch class with hexes;
- a loot-heavy game;
- the first promotion at level 10;
- a new roadmap order.

Added the [level impact and class power](designs/2026-10-05-level-impact-class-power-design.md)
and [loot system](designs/2026-10-05-loot-system-design.md) designs.
Follow-up owner decisions:
- the Witch is a starting class;
- item level requirements;
- personal loot per allied company;
- unidentified Rare+ items with a caster Scribe;
- a stat point every 2 levels, kept in balance;
- mana refills only by rest or costly draughts, and healers patch the company
  up until they must camp;
- enemy levels set by zone bands, with strategy rising with enemy level;
- hex spell names, including a Miasma poison cloud;
- passive HP recovery trickles only up to 50% of max HP;
- Scribe reuses the retired `scribe` skill. Casters train it, including
  companions after joining, and some recruits already know it. It identifies
  automatically at camp and costs mana on command in the field;
- companions earn training points for optional skills; class specialist
  skills stay automatic by level. Owner
updates went into the branching class, random encounter and expanded class
designs. Documentation only. Verification: relative Markdown links and the
diff checked; no Go tests required.

**Equipment documentation updated (2026-10-02):** removed the proposed new
class and its ability, kit, recruitment, help, and delivery requirements at the
owner's request. Retained glaive weapons, equipment families, tiers, and armor
paths; renamed the equipment design and updated its links and roadmap scope.
Verification: documentation diff, residual-reference search, and relative
Markdown links checked; no gameplay changes or Go tests required.

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-10-06 (35b caster power, mana and recovery complete with retuned combat chances; 35a2 skill over hit points merged via PR #18; 35c companion training merged via PR #16; 35a merged via PR #15; 35a2 skill-over-HP design approved and planned, faith routes design approved (alignment wait at elite, broken binding and hungry dark healing kept, routes final); visual client milestone and sprite specification added; roadmap reprioritized)
- **Latest completed slices:** 35b, caster power; 35a2, skill over hit points; 35c, companion training; 35a, level impact; 30g5, action meter; 30g4, progression; 30f, battlefield conditions; Phase 34 review follow-up; 33i2, coordinated enemies; 34d, effects and current capabilities; 33h3,
  relocation and separation; 34c, equipment editor; 33h2, readiness and recovery; 34a,
  UI/formation; 34b, packs/capacity; 33h1, companion growth and contracts
  (2026-10-02);
  33g management (equipment catalog/class still pending),
  33i1, company encounter assessment, and
  33f3, camp specialists (built in parallel); 33f2, expedition specialists; 33f1, skill
  and charm retirement; 33e,
  automatic class abilities; 33d, allied
  companies; 33c, company retreat; 33b, friendly effects; 33a, command
  rules; and 30g3, personal load and agility, all 2026-10-01.
- **Upstream baseline:** `39e44013` (GoMud Telnet input-masking fix).

## Roadmap priorities (owner, 2026-10-05)

After a general game review, the owner reordered the next work. World
building waits until these ship. The level-impact design is owner-approved;
35a is complete and merged (PR #15). The
remaining slices are pending implementation or design review as listed below.

1. **Level impact and caster power.** [Design](designs/2026-10-05-level-impact-class-power-design.md)
   (slices 1–2):
   - every level counts: smooth stat growth, a stat point every 2 levels, and
     HP growth through level 20;
   - spells and abilities scale with level;
   - owned spells never fizzle in battle, and the roll-100 bug is fixed;
   - large caster mana pools that refill only through rest or costly mana
     draughts;
   - stronger healing, with healers patching the company up after battle
     from their mana until they must camp;
   - passive HP recovery outside rest only trickles up to 50% of max HP;
   - companions earn derived training points to learn optional skills
     (Scribe, Cooking, later Alchemy) with `company train`; class specialist
     skills stay automatic;
   - harness cells for the owner's encounter contract: groups of 2–3,
     occasionally 4, and bosses of up to 5, with levels set by the zone's
     band (not the player) and strategy growing with enemy level.
2. **Loot system and gear catalog.** [Design](designs/2026-10-05-loot-system-design.md):
   - tiers, quality, rarity, item level and affixes;
   - level requirements on Uncommon and better items, personal loot for each
     allied company, and unidentified Rare+ items read by **Scribe**, a
     trainable caster skill: automatic at camp, and mana-costing on command in
     the field;
   - legendaries and sets;
   - sellable goods for horse hauling, with salvage;
   - zone and boss drop tables.
   Absorbs the approved [equipment tiers](designs/2026-10-01-equipment-tiers-design.md)
   catalog slice. Moved up by the owner.
3. **Random room encounters.** [Design](designs/2026-10-01-random-room-encounters-design.md),
   updated with the encounter contract. Moved up by the owner. Its drop tables
   come from item 2.
4. **Class promotion at level 10 and class routes.** [Branching design](designs/2026-10-01-branching-class-progression-design.md):
   - level 10 is now an owner decision;
   - talents at levels 5, 15 and 25;
   - the **Witch** as a sixth base class (owner decision), with
     level-scaling hexes: Slumber, Earthbind, Leaden Curse, Miasma (a poison
     cloud), Binding Hex, Curse of Frailty, Dread Whisper and Blight;
   - its proposed Hedge Witch, Coven Sage and Hag routes, from the level
     impact design (slice 3).

Items 1 and 2 can be designed in parallel. Item 1 ships first because the
loot and encounter tuning depend on its numbers. Earlier items, including the
30g mirror target, remain as stress checks, not the tuning goal.

### Phase sequence (proposed 2026-10-05, extended 2026-10-06)

Each phase gets its execution plan in `docs/plans/` and the usual
review gate. Since 2026-10-06 the owner has delegated design approval
(handoff rule 20): a design or review thread decides open questions and
records each decision with its reason. The full map of remaining phases,
their dependencies and those decisions is the
[remaining roadmap](plans/2026-10-06-remaining-roadmap.md).

| Phase | Scope | Source | Depends on |
|---|---|---|---|
| 35a | Level impact: smooth stats, stat point every 2 levels, HP to level 20, level-up report, zone-band harness cells. [Plan](plans/2026-10-05-phase-35a-level-impact.md), complete (PR #15) | Level impact §1, §4.1 | — |
| 35a2 | Skill over hit points: derived Attack and Evasion ratings by level and class, one skill edge added to every opposed chance (block included), small HP growth with a 15–25% landed hit, a smaller Strength damage bonus, armor bulk with a significant untrained penalty (warriors the tanks), shields for warriors and rangers (bucklers) only, cleric staffs/rods/maces, spell and heal numbers sized to a weapon hit. [Design](designs/2026-10-05-phase-35a2-skill-over-hit-points-design.md), owner-approved 2026-10-05; [plan](plans/2026-10-05-phase-35a2-skill-over-hit-points.md); complete, merged via [PR #18](https://github.com/Robinsond76/ashveil-gomud/pull/18) ([measurements](plans/2026-10-05-phase-35a2-measurements.md)); three balance rows retuned in 35b | Owner direction 2026-10-05 | 35a |
| 35b | Caster power: no fizzle, roll-100 fix, scaling spells and abilities, caster mana pools, no passive mana, mana draughts, healing and after-battle patching, the 50% HP trickle, the easy-fight wound change. [Plan](plans/2026-10-05-phase-35b-caster-power.md), complete ([measurements](plans/2026-10-05-phase-35b-measurements.md)) | Level impact §2, §4 | 35a, 35a2 |
| 35c | Companion training: derived points, `company train`, trained optional skills (Cooking first). [Plan](plans/2026-10-05-phase-35c-companion-training.md), complete, merged via [PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16) | Level impact §5 | 35a |
| 36a | Loot item model and generator: layers, affixes, level requirements, display, persistence; Scribe and identification. [Plan](plans/2026-10-06-phase-36a-loot-item-model.md), complete, merged via [PR #23](https://github.com/Robinsond76/ashveil-gomud/pull/23) | Loot design slice 1 | 35b, 35c |
| 36b | Tier 1–3 gear catalog, goods and an audit of existing items. [Plan](plans/2026-10-06-phase-36b-gear-catalog.md), complete, merged via [PR #25](https://github.com/Robinsond76/ashveil-gomud/pull/25) | Loot slice 2; equipment tiers | 36a |
| 35d | Combat feel: every swing lands with a quality (glancing, solid, telling) the skill edge decides, one-round heals resolve, an 80% after-battle patch threshold, company tactics defaults that grow with the leader's level, HP keeping pace after level 20, short bosses with no strategy, seconds-and-lines targets. [Design](designs/2026-10-06-phase-35d-combat-feel-design.md), **approved 2026-10-06** (all open-question defaults accepted; enemy healers may be uncommon); [plan](plans/2026-10-06-phase-35d-combat-feel.md); from the [combat rebalance second opinion](plans/2026-10-06-combat-rebalance-second-opinion.md) | Owner direction 2026-10-06 | 35b |
| 37 | Random room encounters and zone level bands, with drop tables, caches, boss rolls and personal loot (loot slice 3). [Plan](plans/2026-10-06-phase-37-random-encounters.md), complete, merged via [PR #29](https://github.com/Robinsond76/ashveil-gomud/pull/29) | Encounter design; loot slice 3 | 35b, 35d, 36b |
| 38a | Witch base class: hexes, three new statuses, controller role. [Plan](plans/2026-10-06-phase-38a-witch.md), complete, merged via [PR #24](https://github.com/Robinsond76/ashveil-gomud/pull/24) | Level impact §3 | 35b |
| 38b | Complete, merged via [PR #34](https://github.com/Robinsond76/ashveil-gomud/pull/34). Class promotion at level 10, talents at 5/15/25, core routes for all six lineages; cleric and warrior routes per the approved [faith routes design](designs/2026-10-05-faith-routes-design.md) (summoned Angel and Demon, Paladin and Blackguard fighting healers) | Branching design; level impact §1e; faith routes | 35a, 35a2, 35d, 38a |
| 35e | Focus the healer: a `healers` focus rule, the company default whenever the enemy has a healer (from leader level 5). Complete, merged via [PR #33](https://github.com/Robinsond76/ashveil-gomud/pull/33) | Roadmap 2026-10-06 (owner's difficulty rule) | — |
| 44 | Live smoke playtest: a scripted run against a real server (tutorial, company, fight, copyover, two players). Complete, merged via [PR #38](https://github.com/Robinsond76/ashveil-gomud/pull/38) (`make smoke`) | Roadmap 2026-10-06 | — |
| 44b | World smoke playtest (`make smoke-world`): journey, encounter fight and loot, gathering, camp, restart, salvage, market and inn on a live server; fixed a journey-arrival deadlock and four other live bugs. Complete, in review | Roadmap 2026-10-06 | 44, 37, 36c, 40a2 |
| 37b | Encounter and pacing tuning: enemy healers to uncommon, boss respawn, zone band in look and web header, level-gap and boss tuning. Complete, merged via [PR #39](https://github.com/Robinsond76/ashveil-gomud/pull/39); harness gear deferred | Roadmap 2026-10-06 | 37, 35e |
| 37d | Test isolation: shuffled runs pass, shared event queue, world and logger state fixed at the cause. Complete, merged via [PR #67](https://github.com/Robinsond76/ashveil-gomud/pull/67) | Coordinator 2026-10-06 | 37c |
| 37c | Test stability (flaky tests found by shuffled and repeated runs) and the zone band rating by company level, refreshed on level change. Built, in review | Roadmap 2026-10-06 | 37b |
| 36c | Loot economy: goods in markets, saturation, salvage, `sell junk`, identification fees; merchants buy rolled gear and GMCP shows rolled names (36a deferrals). [Plan](plans/2026-10-06-phase-36c-loot-economy.md), complete, merged via [PR #37](https://github.com/Robinsond76/ashveil-gomud/pull/37) | Loot slice 4 | 37 |
| 38c-d | Elite routes design for the six lineages: [design](designs/2026-10-06-elite-routes-design.md) and [38c plan](plans/2026-10-06-phase-38c-elite-routes.md), complete (approved under delegation 2026-10-06) | Branching design | — |
| 38c1 | Built (PR open). Elite framework (promotion at 30, gates and waiting, catch-up ranks, elite talents), UI and `help elite`; warrior and cleric elites | Elite routes design; faith routes | 38b |
| 38c2 | Complete, merged via [PR #60](https://github.com/Robinsond76/ashveil-gomud/pull/60). Rogue and ranger elites (Pathfinder, Swordmaster, Nightblade, Sentinel, Marksman, Ravager) | Elite routes design | 38c1 |
| 38c3 | Complete, merged via [PR #62](https://github.com/Robinsond76/ashveil-gomud/pull/62). Wizard and witch elites (Archon, Archmage, Necromancer with its thrall, Wise One, Coven Mother, Crone of Ash) | Elite routes design | 38c1 |
| 38d | Sorcerer bundle merged (PR #78); further catalogue bundles (Elementalist, Illusionist, others) later | Expanded catalogue | 38c1 |
| 38e | Merged (PR #86). Creature recruits, Hound and Stone Golem pilot. [Plan](plans/2026-10-06-phase-38e-creature-recruits.md) | Expanded catalogue | 38d, 39e (not needed by the pilot) |
| 36d | Tier 4–6 gear, Legendary signatures and Set bonuses. [Plan](plans/2026-10-06-phase-36d-relics.md), merged (PR #120 via the review PR) | Loot slice 5 | 36c, 38c1–38c3 |
| 39a–39h | Neutral base classes, one per phase: Halberdier, Samurai, Shaman, Doll Master, Beast Tamer, Gryphon Rider, Alchemist, Arbalist; at most two building at once | [Neutral classes design](designs/2026-10-05-neutral-classes-design.md) | 38b (39e also 39d) |
| 39i | Elite ranks 30–60 for the eight neutral lineages | Neutral classes design §12 | 38c1, 39a–39h |
| 41 | World building, levels 1–15, tile-ready | Owner 2026-10-05 | 40d, 37b |
| 42 | Zones 15–30+, elite content, tier 4–6 placement | Roadmap 2026-10-06 | 41, 38c1–38c3, 36d |
| 43a | Camp consumables | [Design](designs/2026-10-01-camp-consumables-design.md) | 40a2 |
| 43b | Weapon poisons (shop-only launch merged 2026-10-06, PR #57; crafting stays later) | [Design](designs/2026-10-01-weapon-poisons-design.md) | 43a for crafting only |
| 50 | Condition carries into battle (needs bands set a pre-battle penalty); cooked meals give buffs for a few battles | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | 47 |
| 51 | Rest duties: each member sleeps, watches, tends, forages, cooks or brews during a camp rest | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | 47; coordinate with 49 |
| 52 | Tents with trade-offs (fur, camouflaged, large) | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | 51 |
| 53 | Defeat scenarios (rescued, captured, left for dead, robbed) in place of the church respawn | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | Companion equipment, 40a4 |
| 54 | Sigils laid in a room before a battle (complete, PR #87) | [Outward phases](plans/2026-10-06-outward-survival-phases.md), [plan](plans/2026-10-06-phase-54-sigils.md) | — |
| 55 | Ailments and remedies (Chill, Gut-ache, Fever) | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | 50 |
| 56 | Recipe discovery and a recipe book | [Outward phases](plans/2026-10-06-outward-survival-phases.md) | 50 |
| 57 | Character panel tidy-up (Skills, Jobs, Gear, Overview, hover highlight) built, PR open | — | — |
| 60–77 | Lessons from Pillars of Eternity: story events, battle orders, explained battle lines, company chronicle, companion opinions and bonds, bestiary, relic awakenings, towns that remember, weapon stances, errands, trophy enchanting, backgrounds, creeds, rites, inn tiers, bounties, Hardcore and blessings | [Pillars phases](plans/2026-10-07-pillars-phases.md) | See plan; 60–63 first |
| 78 | Polish for phases 60-72: scene modal after restart, orders and stance menus, round heading crit, background line once, same-name duty picker, nameless placeholder | [Plan](plans/2026-10-07-phase-78-polish.md) | — |
| 79 | Wider polish pass: web login after an early GMCP request, `why` rounds numbered within the fight, no corpse for a fallen companion, companion tag in the room panel, tab completion of real commands only, grouped `company` usage, Bonds tab, help audit of phases 60-77 | [Plan](plans/2026-10-07-phase-79-polish.md) | 78, 82a-82d |
| 83 | Test hardening: a battle no longer breaks off and reopens when every fighter's last target falls in one round (`engagedWith`), the process-wide dice are handed back after a seeded test, and `modules/company` tests use `hardTo`/`hardMaxTo` | [Plan](plans/2026-10-08-phase-83-test-hardening.md) | 80, 82a-82d |

World building (zones for levels 1–15) now waits until the visual client
milestone below is in place (owner, 2026-10-05). A small showcase area for
the new map features may be built in 40d.

### Next milestone: visual client (Phase 40, owner 2026-10-05)

Since 2026-10-06 it runs beside the sequence above: 40e, 40a and art set
40s1 can start now. The battle screen lane (40e, 40f, 40g) has priority,
and art is code-generated pixel art built by art threads one step ahead of
the code ([roadmap](plans/2026-10-06-remaining-roadmap.md#visual-direction-owner-asked-for-a-recommendation)).
The 40a–40g designs are approved under the owner's delegation. See the
[milestone design](designs/2026-10-05-visual-client-milestone-design.md)
and the [sprite specification](designs/2026-10-05-sprite-specification.md).

| Phase | Scope | Art set |
|---|---|---|
| [40a](designs/2026-10-05-phase-40a-room-resources-design.md) | Room resources: data, `look` line, GMCP, map icons, water in survival, forage, shelter | S1 |
| [40a2](designs/2026-10-05-phase-40a2-gathering-design.md) | Gathering: herbs, firewood, fishing, game; room pools; firewood for the camp fire | S1 |
| [40a3](designs/2026-10-05-phase-40a3-camp-gear-design.md) | Camp gear: a firewood bundle per rest, plus bedroll, tent, fire steel, cookpot, bells, surgeon's kit. **Done (PR #45)** | S1 |
| 40a4 | Camp theft without bells and trip lines, and a web gear line. **Built (review pending)** | — |
| 40s1–40s5 | Art sets S0+S1, S2, S3, S4, S5 as code-generated pixel art (S5 after 38b) | S0–S5 |
| [40b](designs/2026-10-05-phase-40b-map-sprites-design.md) | Class sprite on the map, company badge, own and allied camps | S0, S1 |
| [40c](designs/2026-10-05-phase-40c-terrain-tiles-design.md) | Terrain and landmark tiles, fog, classic toggle | S2 |
| [40d](designs/2026-10-05-phase-40d-tile-region-travel-design.md) | Tile-ready pilot region and click-to-walk | S2 |
| [40e](designs/2026-10-05-phase-40e-combat-event-messages-design.md) | Structured combat events (can run in parallel with 40a–40d) | — |
| [40f](designs/2026-10-05-phase-40f-battle-screen-design.md) | Static OB64-style battle screen | S3 |
| [40g](designs/2026-10-05-phase-40g-battle-animation-design.md) | Battle animation and effects | S4 |
| 40h | Neutral and elite class art, status names | S5 |
| 40i | Touch layout and installable web app | S1 |

The milestone's owner questions were answered on 2026-10-05; the phase
designs' remaining questions were decided on 2026-10-06 (roadmap,
"Decisions on open questions").

**Phase 35 delivery (2026-10-05):** 35a is complete and merged
([PR #15](https://github.com/Robinsond76/ashveil-gomud/pull/15)). 35c is
complete and merged ([PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16)). 35a2 and 35b are complete. Plans and approved scope are in
[the phase 35 handoff](plans/2026-10-05-phase-35-handoff.md).

## Current position

**Web page cleanup (2026-10-06):** the public site's nav links (Home, Who's
Online, Web Client, Configuration, Help) moved into a "Pages" tab of the
top-right settings menu; the header and footer are slimmer so the web client
fills more of the screen; the default `Server.MudName` is now "Ashveil"; the
footer is one tiny "Powered by GoMud" line linking to the GoMud GitHub.
Screenshots in `docs/screens/web-cleanup-*.png`. Review (2026-10-06):
one fix accepted, "Hide header and footer" (now in the settings modal) left
the modal open over the game; it now closes it. Checked in Chromium at
1280x800 and 390x844: the web client frame grows from 741 to 800 px tall on
desktop with the header hidden. A local `config-overrides.yaml` setting
`Server.MudName` would keep an old name; none ships in the repo.

**35b caster power, mana and recovery (2026-10-05), complete:** owned spells
never fizzle in a battle, and a 100% cast never fails anywhere. Spell and
ability power comes from one `spellpower` table per spell (YAML `power`
blocks, read by the scripts and `heal wounds`), growing with level and
Mysticism. The level-up report lists each scaling spell's new range, and
level spells (Shower of Sparks, Minor Heal All at 3) are taught on level-up.
Wizards and clerics have deep mana pools (40 + 10 a level, 36 + 9), and mana
never regenerates for players or companions. It comes back only from a
finished camp rest, an inn night, a mana draught (25/40/60%, sold by Moilyn,
refused in battle), half a broken chant, or death's return. HP trickles back
only to half. After a won battle, and on `company patch`, healers cast Minor
Heal to the healing threshold, stopping at their mana reserve. Chant breaks
add difficulty/5. Aimed Shot and guard counts grow with level; Tackle holds 1
round longer from 20. A crit from a foe 3+ levels below leaves no wound
above half health. New `help patch` and `help draughts`, 21 pages updated,
and tutorial hints.

**Balance:** [measurements](plans/2026-10-05-phase-35b-measurements.md).
Retuned: to-hit 75 (was 60), dodge and parry 8 (12), block 28 (20),
`SkillEdgeSpan` 14 (16), Minor Heal chant 1 round (2). Met: tank ratio
0.70×, skill wins (L10 beats 2×L20 20%), mismatches, no fizzles. Changed
under the owner's balance latitude: the 5v5 mirror runs 11–15 rounds
(asserted ≤16; hit size makes 8–12 unreachable). Coordinated tiers are beaten
with tactics (L10: tier 2 83%, tier 3 47–63%; untactical 48%/32%). The mana
run also tends wounds between fights. Zone bands meet win rate, HP lost and
≤12 rounds at band middle and low. **Settled per owner (2026-10-05, "settle for
the best you achieved"):** reported, not asserted, for phase 37's encounter
tuning: members fallen against 4 foes (1.4–2.3, target ≤1); band 18+
fights with nobody down (46–60%, target ≥85%); bosses (38–74%, target 70–85%,
about 30 rounds); a company 3 levels under band (93–98% wins, target 30–60%).

**Review:** an independent full-diff reviewer found 7 issues to fix and 8
nits. Accepted and fixed with regression tests: patching skipped a
downed leader (now healed back to their feet); the battle-end patch ignored
a companion still fighting (now waits); a level spell was missing from the
level-up report (granted before the report is measured); help patch,
wounds, friendly-effects, combat, mana and camp were wrong or stale;
`TestBalanceZoneBands` asserted nothing (it now asserts the rows that hold);
measurements and status missing (written); coverage for the report through
GrantXP, Tackle at 20, allied patching and enemy template mana (added); the
simulator ignored template mana (copied); a test wrote a user file into the
shipped world (temp folder). The race run caught `TestAimedShotGrowsWithLevel`
flaking (two brawls in one test shared listeners; each now runs in its own
subtest, 30/30 passes). The full harness also caught a skill-gap regression
(L10 beat 2×L20 48%), fixed by the span change. Rejected: updating the Go
config defaults to the shipped combat chances (they stay GoMud's engine
values; the shipped config sets every key and tests pin the defaults);
deriving the milestone list from the archetype overlay (two casters today;
revisit when phase 38 adds classes).

**35a2 skill over hit points (2026-10-05), built, in review as [PR #18](https://github.com/Robinsond76/ashveil-gomud/pull/18):**
Attack and Evasion are derived from level and class (warrior 1/1, rogue
0.9/1.1, ranger 1/0.9, cleric 0.7/0.75, wizard 0.7/0.75, enemies 1/1 plus a
template offset), never saved. Their gap over `SkillEdgeSpan` (tuned 20 → 16)
is added to every opposed chance's stat edge except crits; dodge and parry
are two-sided around 12%, block is 20 (tuned from 15) plus shield armor. HP is
48 + HPStart + the class rate through level 20, a quarter after; the damage
bonus is 6–12 from Strength. Armor bulk (medium −8% tempo, −20% dodge; heavy
−20%, −50%), doubled when untrained with −10 Attack and Evasion and +1 chant
round. Shields: warriors any, rangers bucklers, others none; clerics wield
staffs, rods and maces, with a holy symbol (+5% heals). `SettleClassGear`
puts away disallowed gear on spawn, class choice and copyover. Spells, heals
and Opening Strike scale with the gap and level. New `help evasion` and
`help shields`; armor, defense, stat-edge and the other pages it changes updated; tutorial
hints. Wolf pelt, snow wolf mane and spider exoskeleton retagged medium.

**Balance:** [measurements](plans/2026-10-05-phase-35a2-measurements.md).
Passing: skill wins (L20 beats 3×L10 100%, losing 5.3% HP; L10 beats 2×L20
12%), mirror parity, kit, targeting, sides even, statuses. **Owner
questions:** equal 5v5 fights run a median of 14–20 rounds (target 8–12;
30g6 was 12–14), because two-sided defenses and fifth-of-HP hits can't both
meet it; a warrior takes 0.84× a rogue's damage per swing (target 0.7×);
L15 now always beats L10 (the 30g6 row wants it to lose sometimes); enemy
coordination tiers 2–3 drop a level-10 company to 37% and 26% wins (33i2
wants ≥ 50%). **Owner decision (2026-10-05):** merge 35a2 as is; the
fight-length, tank and coordination targets move to the
[35b plan](plans/2026-10-05-phase-35b-caster-power.md)'s acceptance, and the
"L15 can lose" row is retired (its assertion removed).

**Review:** an independent full-diff reviewer found no blocking bug.
Accepted and fixed with regression tests: `help stat-edge` contradicted the
two-sided defenses (rewritten); a readied Opening Strike that never swung
kept its bonus for a later backstab (cleared); prediction helpers untested
against skill (`TestPredictionsFollowSkill`); measurements and status not
recorded (this entry); `help armor` said bulk adds to burden and gave wrong
examples (fixed); missing coverage of plain `equip`'s warning, `status` in
untrained armor, the companion level line and HPStart through a spawn
(added); cleric class text and listings lacked gear rules (each class's
listing now names its shields and weapons); the tempo hint and bash line
omitted bulk and skill (fixed); an out-of-range bulk share dropped to 0
(now held to 0–1); the equipment preview cache ignored the class (keyed).
Accepted as deviations: `equip` warns after an untrained piece goes on, not
before; the untrained −10 is 0.625 of an edge at span 16, not half.
Rejected: the damage bonus taking the skill edge (decision 4 keeps it
Strength-only so hits stay a steady share); missing `Bulk*`/`Untrained*`
keys defaulting to the shipped values (0 is a valid "off"; shipped config
sets them); caching the profile on the hot path (no measured cost); a
registry-reload test for companion gear (the move is idempotent on every
spawn and the next snapshot saves it; nothing can be lost).

**35c companion training (2026-10-05), complete, merged via [PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16):** companions
earn training points from level (`ProgressionConfig.TrainingPointsAt`, level 1
counts), never banked: points = earned minus the cost of trained ranks (a rank
costs its own number, so rank 4 costs 1+2+3+4). A lost level can leave a
companion owing points; views show 0 and the debt. `company train [member]
[skill]` previews and `... confirm` (idempotent when it carries the rank)
trains: ranks 1-2 at the leader's own camp or a trainer, 3-4 only at a room
whose `SkillTraining` covers the rank. It refuses in battle, while travelling
or resting, and for dead, separated, absent or fighting members. Optional
skills live in the archetype overlay (`OptionalSkills`; Cooking for everyone
first). Some recruits arrive already trained (20%, one in five of those at
rank 2, price +15% per rank). `camp cook` uses the best cook present among the
leader and living companions in the camp room (ties: leader, then lowest ID)
and names them. GMCP `Company` members carry `skills` and `training_points`;
the web client's Skills tab shows a Company training section. Help:
`company-train` (new) plus `company`, `growth`, `skills`, `cooking`,
`specialists`, `webclient`; Camp tutorial hint. No game time advances.
Deviations from the plan: trained ranks are stored on `company.Companion`
(`Skills`, `GrantedSkills`) rather than `MemberState`, so gear and level
snapshots can never drop them; confirm accepts an explicit rank; `company
inspect` matches your own companions by exact name before recruits and
templates. See the [plan](plans/2026-10-05-phase-35c-companion-training.md).

**Review:** independent full-diff reviewer found no P1. Accepted and fixed:
P2 no command-path best-cook test (added `TestCampCookCommandUsesTheBestCook`
through `camp cook`; presence still uses the test seam); P2 `company train
[member] confirm` without a skill said "no companion like that" (now asks for
a skill); P3 help overstated confirm idempotence (only the rank-bearing line
is); P3 confirm could spend points from a live level not yet saved (confirm
now writes the live level and experience into the record in the same save);
P3 unused `CookingView.Cook`/`CookIsLeader` removed; P3 cooking help example
corrected; P3 eligibility and max-rank refusals now come before battle,
travel, rest and presence. Rejected or noted: a mob template that defines
`cooking` would cook above its trained rank (no shipped recruit template does;
noted); `company inspect [name]` now prefers your own companion (intended; tested by
name and by `#N`); the client capitalises skill ids rather than
display names (fine for Cooking; revisit with multi-word skills). Also fixed
an order-dependent expectation in `TestCompanyMembersStates`.

**35a level impact (2026-10-05), complete, merged via [PR #15](https://github.com/Robinsond76/ashveil-gomud/pull/15):**
fractional racial growth preserves the fifth-level boundary values; keep
`StatStepLevels: 5` per the approved plan instead of the design draft's 1,
which would multiply the curve. Shared point awards now give one every two
levels, with an atomic, once-only peak-level migration on normal load and
copyover. HP grows at the full class rate through 20, then 60%. Player and
companion reports show final before/after values; experience/status announce
planned milestones as `(coming)`. Admin previews, save/reload, help and tutorial
follow the same rules. No milestone is shipped by 35a. 35b/35c remain pending.
See the [plan](plans/2026-10-05-phase-35a-level-impact.md),
[measurements and limitations](plans/2026-10-05-phase-35a-measurements.md), and
[admin screenshot](verification/phase-35a/progression-editor.png).

**Review:** independent full-diff reviewer found no P1 production blocker.
Accepted: P2 shipped human automatic adjusted stats change on only 8/59
level-ups (levels 5, 15, 25, 30, 35, 45, 55, 60), so the "most level-ups"
acceptance criterion is **unmet** despite faithfully implementing the approved
fractional formula. Recorded as a design limitation; no race or curve retuning
in 35a. Accepted and fixed: P2 missing real rendered stat-line, companion
post-retraining report, and experience milestone/no-future coverage. Also
isolated two pre-existing battle fixtures from automatic Tackle and captain
mortality. No findings rejected. The focused report/migration/fixture checks
passed ten repeated runs. Verification: 8,200 zone fights (100 per cell), zero
stalls; unchanged mismatch assertions pass (97% L15 vs L10, 100% L30 vs L10).
Chromium checked smoothing, cadence preview/save/reload, level-200 downsampling,
API reference and console errors. Final verification passed: `make generate`,
`make validate`, `make js-lint` with installed JSHint, local `make lua-lint`,
`go test -race ./...`, inline admin JavaScript syntax and relative Markdown
links. Gameplay implementation complete; merged via PR #15.

**35a PR review follow-up (2026-10-05):** a PR code review found the
once-only migration marked characters migrated even while the server still
ran the 5-level rhythm (for example through a saved
`StatPointsEveryNLevels: 5` override), so a later switch to 2 never paid them,
and assumed every character earned on 5. Accepted and fixed:
`Character.CatchUpStatPoints` tops a character up to the current rhythm's
total at its peak level, counting points it already holds (unspent plus stat
training), so pre-30g4 characters who earned a point every level, and players
given points, are not paid twice; nothing is taken away. It leaves
characters, new ones included, unmarked while the rhythm is still 5, and runs
on load (atomic save) or before a live level-up, so a mid-session change is
neither skipped nor counted in the level-up report. An independent review of
the fix found no P1. Accepted: P2 new characters were pre-marked (fixed, they
now start unmarked); P2 pre-30g4 characters (rhythm 1) would be overpaid
(fixed by the held-points top-up; the cost is that a stat coupon spent before
the catch-up reduces it). Rejected: P3 announce catch-up points (silent like
the original migration); P3 save the live catch-up at once (points and marker
share one record, so a crash reverts both and the next load repays); P3
display lookups of offline users can run the migration save (pre-existing,
once per character, main-loop only). Regression tests cover the load, live
level-up, held-points, lost-level and new-character paths. Full checks
passed (`make generate`, `make validate`, `go test -race ./...`), except one
run where `TestAClericCompanionHealsTheHurt` (company, untouched) failed once;
it passed 100 isolated runs (40 with `-race`) and two full `-race` package
runs, so it is noted as an unreproduced intermittent failure to investigate.

**30g6 stat edges and tuning (2026-10-04):** on top of 30g6a, per the
[amendment](designs/2026-10-04-phase-30g6-amendment.md) A, B, E and F. Every
opposed chance (hit, crit, dodge, parry, block, bash, crit damage, the
Strength edge in damage, tackle) reads a stat difference over
`StatEdgeSpan` (40) instead of a ratio of small stepped stats; damage adds
`DamagePerStrength` with a small `DamageEdgeMax`; HP keeps each archetype's
proportion after `HPFullLevels`; Minor Heal adds the caster's level (in battle
and in `heal wounds`) and Minor Heal All half of it. The whole opt-in suite
passes at 100 fights a cell: spread mirror medians 12–14 rounds at levels
1–60 (17 at 100), even mirrors, kit no handicap, enemy targeting
significantly worse for the company, level 15 beats level 10 90%, level 30
wins 100% and loses no member in most fights; HP L60/L10 2.09–2.20×. New
indexed `help stat-edge`; updated stat, defense, abilities, brawling, attack,
progression, health, heal and friendly-effects pages; Practice Yard hint. See
[verification](plans/2026-10-04-phase-30g6-verification.md) and
[measurements](plans/2026-10-04-phase-30g6-measurements.md). Supersedes the
first candidate (`phase-30g6-tuning`). **Review:** independent full-diff
review, no P1. Accepted and fixed: P2 `help strength` misdescribed the
Strength edge (now `+DamageEdgeMax` at a full span, per point a share); P2
`heal wounds` still healed a flat 2d3 (healers now add their level,
`TestPlanHealAddsTheHealersBonus` and the real-command check in
`TestHealWoundsALeaderClericHealsToTheLimit`); P3 `DamageEdgeMax` missing
from the admin pages and stale "Default" text there; P3 even-value inputs
allowed 0; P3 `hitChance` now rounds like the other chances; P3 help built
ordinals like "3th" from the span (now "1/40"). Rejected: P3 validation
fallbacks differ from shipped values (they guard invalid input, as the
existing tempo fallbacks do; shipped values live in `config.yaml`); P3 tackle
bounds as code constants (fixed odds shared by both tackle paths and their
help; not a tuned number); P3 Strength's ranking weight ignores the damage
cap (documented "below cap" approximation; rankings use no attacker stats);
P3 no test drives the manual `tackle` command (blocked in every battle since
32c; its chance is the tested shared helper); P3 flee and disarm formulas
(outside the stat edge's scope).

**30g6a combat fixes and balance harness (2026-10-04):** split from the first
30g6 candidate (`phase-30g6-tuning`, `5b36efd`) after a review found its
acceptance failures structural, not numeric: stepped stats broke the ratio
formulas, the "even" fight gave only the company abilities and a healer, and
"focus wins sooner" was measured over all fights. The owner approved a
[30g6 amendment](designs/2026-10-04-phase-30g6-amendment.md) and decided that
company focus is reported, not asserted (a strategy, not a guarantee). 30g6a
keeps master's numbers and ships: no repeated tackle within one ability pass,
caster data on replacement targets, an earned turn surviving an earlier kill
for company members *and* enemies, no game-round ticks for combat statuses,
and the harness (levels 1–100, mismatches, caster targeting, net health, a
true mirror plus a kit cell, Welch significance for enemy targeting). Help:
`tempo`, `abilities`, `combat`. On master's numbers the opt-in suite fails
as expected (level-10 fights about 73 rounds); see
[verification](plans/2026-10-04-phase-30g6a-verification.md).
**Review:** independent full-diff review, no P1. Accepted and fixed with
regressions: P2 a queued tackle opened a same-pass Opening Strike (now a
separate `FoeDownQueued`, `TestAQueuedTackleOpensNoStrikeThisPass`); P2 only
the company kept earned turns after an earlier kill (enemies now re-aim too,
`TestBalanceEnemyEarnedTurnSurvivesAnEarlierKill`); P2 the mirror had no
company caster, so the casters cell measured nearest-foe aims (the cleric
keeps the healer role with no mana, `TestBalanceMirrorClericIsACasterWhoCastsNothing`);
P3 routed or fled fighters counted as health removed; P3 tempo help wording.
Rejected: P3 player-side candidates mark casters by spellbook while enemy-side
ones use role (both are pre-existing selectors for different sides; the new
retarget matches its own side's ordinary selector); P3 the "rabble noise"
comment (tier 1's floor is 30%, as written). P3 a round-two cleanup check is
covered by the new queued-tackle test's second round.

**30g5 action meter complete (2026-10-03):** effective Speed and personal
burden now determine fractional turns, with one opening turn, at most two
physical turns per combat round, and no whole-turn banking. Shared enemies
fill once; fight replacement, separation, unmanaged restarts and death clear
stale progress. Chants, waits, wind-ups and upkeep retain their round cadence;
abilities consume one earned turn and reactions retain their existing budgets.
Target-relative unarmed/claw extra attacks are retired. Simulator and weapon
estimates use tempo. Indexed `help tempo`, related help, and the Practice Yard
hint explain the mechanic. No pet proposal remains in this slice.

Independent full-phase review found five issues (zero-turn waits, separated
companions, simulation ordering, ranking cap and help indexing). All were
verified, fixed and regression-tested; follow-up review found no blockers.
`make generate`, `make validate`, `go test -race ./...` and focused regressions
passed. Final measurements: 540 fights, no stalls, no-focus
medians 16/60/95 rounds at levels 1/5/10; narration cell means 7.3–9.6 lines,
peak 26. Pre-merge PR review follow-up: withdrawn fighters no longer bump the
combat generation, a redundant wait flag and single-pass DPS loop were
removed, and a regression confirms a foe felled in the company's pass makes no
blow (unchanged from master; the suspected behavior change was rejected).
Legacy `ExtraAttacks*` keys stay for old overrides. **30g6 tuning is next**, including the 10–15-round duration target.
See [amendment](designs/2026-10-02-phase-30g5-action-meter-amendment.md),
[plan](plans/2026-10-02-phase-30g5-action-meter.md) and
[verification](plans/2026-10-03-phase-30g5-verification.md).

**30g4 progression complete (2026-10-02):** automatic stats and stat-point
awards step at levels 5, 10, 15, …; skill training remains every level.
HP uses configured archetype rates through level 20, then one per level,
with race/template overrides for enemies. XP retains its thresholds through
level 60, then grows incremental costs by 1.10 without a level cap.
Existing player investment and saved vitals survive load/copyover; maxima
clamp vitals without refilling. Companion class HP uses its durable identity.
Admin controls/charts share live formulas; creation, status, level-up,
indexed help and tutorial explain the new progression.

Independent review accepted and fixed two P2 findings with regressions:
saturated HP/stat additions could overflow, and Mana previews omitted an
intrinsic racial term. Follow-up accepted both fixes with no remaining
blockers or rejected findings. Generate, validate, JS/Lua lint and editor
browser checks and the full `go test -race ./...` suite passed.
The 900-fight balance table had no stalls; no-focus medians at levels 1/5/10
are 14/44/87 rounds. These remain provisional; 30g5 adds the action meter,
and 30g6 owns the 10–15-round target. See the
[plan](plans/2026-10-02-phase-30g4-progression.md) and
[verification/upgrade notes](plans/2026-10-02-phase-30g4-verification.md).

Phase 30g4 PR integration with the Phase 34 review follow-up (`0a2facb7`)
preserved both status records. Independent integration review found a Gear
preview clone dropped class HP, and its cache could retain the old rate after
a class change. Both paths now preserve the resolved HP rate and key the cache
on it. Warrior/wizard player and companion clone regressions, plus cached
preview/class-change/applied HP parity, passed; follow-up accepted the fix with
no remaining blockers. Merged generation, validation, JS/Lua lint and the
full race suite passed. An existing retreat assertion failed on the first
post-fix run; three focused reruns passed, new test provider cleanup was
corrected, and the final full suite passed. See PR #13.

PR #13 pre-merge review (2026-10-02) found no blockers and accepted three
minor findings, fixed with regressions: player characters made by
`NewUserRecord`, `CreateUser`, deletion reset and permadeath reset now carry
their user id, so their HP uses the player's archetype path rather than the
enemy race/template override path; `ValidateActiveCharacters` skips a
missing user; and `archetype` and its choose preview show each live HP rate, which the
progression and stat-train help now point to instead of fixed numbers alone.
A follow-up review of those fixes found the replay character and
`ReplaceCharacter` also missed the id (fixed with regressions) and that
stat-train's "stat-step" wording conflated two settings (reworded). Rejected:
permadeath keeps the archetype registry entry, so the fresh character keeps
the old class rate; this predates 30g4 and Ashveil deaths are never
permanent. Lower early HP from Vitality 4 → 1 is noted as
balance for 30g6, not a defect.
Merged after 30f (#12); the only code conflict, `Mob.Validate`, keeps both
the `hpperlevel` and ambush-stealth checks. The merge also fixes a 30f flake in
`TestTutorialThroughPluginsLoad` (about 60% of runs on master): its own-line
matcher read the new dodge line "You sway aside, and the straw archer's blow"
as Aria aiming at the archer; it now skips lines naming the foe as attacker.
The merged suite also exposed two company test flakes: with 30g4's lower
enemy HP, `TestWaitingGroupsDontBlockFlight`'s battle foe could fall before
the retreat (a third of runs), letting a quick waiting group pursue; the
test now toughens it. `TestAnEnemysBleedLeavesALightWound` (flaky on master
too) now disables crits so only the bleed wounds the captain.

**30f battlefield conditions complete (2026-10-02):** ambush opening
rounds, formation clusters and sweeps, leaps/open flanks, narrow-ground
projections, fatigue hit penalties, and cold chant/sling delays, with indexed
player help, representative content, and Battle view updates; merged via
[PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12) after independent review. Accepted and fixed with regression tests:
(1) a leap was narrated and its 3-round cooldown spent on a strike ordinary
reach already allowed (front cell empty or fallen), leaving it unavailable
over a knocked-down protector; it is now spent only when needed; (2) the
fatigue suffix was appended to every strike; it now appears when a penalty
starts or changes; (3) a sweep could strike a front-row guardian twice (its
ward's blow, then its own); each member is now struck at most once, and the
help says so. Rejected: (4) silent melee turns in the narrow reserve, as
both sides are capped at five members and six cells are active, so no
reserve forms in play; (5) an opening recorded for a battle begun at
end-of-round settlement, as encounter groups begin at the top of a round
(travel and camp spawns never wait behind another battle).
After the fixes: make generate, make validate, JS lint, and go test -race
./... (96 packages) passed; Lua lint not run here (no Docker; no Lua changed).
[Approved design](designs/2026-10-02-phase-30f-battlefield-design.md),
[plan](plans/2026-10-02-phase-30f-battlefield-plan.md), and
[verification](verification/phase-30f/verification.md).

**On-demand Gear editor and refresh load benchmark (2026-10-02):** at the
owner's request, to keep the game loop light with many players. The server
builds `Company.Equipment` only while a client shows the Gear editor, and
previews only the selected slot's choices; the client announces open/closed
and the slot. A cache hit no longer marshals the full character, and a
rebuild reads the company load once. New `TestCompanyRefreshLoad`
(`ASHVEIL_LOAD_BENCH=1`) times one player's whole company refresh per round
(summary, snapshot and every extra) with companions, 36 cargo items and
ticking effects. Before → after, per player per round: Gear closed 0.78 →
0.37 ms; open and unchanged 2.77 → 1.09 ms; open while cargo changes every
round 15.1 → 2.3 ms. At 200 players with Gear closed that is about 74 ms of
each 4 s round. Combat, mob AI and movement are not in this measure. The
largest remaining parts are the inventory payload and the summary.
Independent review: no blockers or majors. Accepted and fixed with tests:
an open message refreshes only when the editor opens or changes slot, and
not within 200 ms of the last (a repeating client can't force rebuilds);
an unknown slot name falls back to the weapon; the web request parsing and
login clearing are now tested through `HandleWebGMCP` and `PlayerSpawn`; a
reset selection is announced at once; the benchmark removes the flags it
registers. Not fixed (cosmetic): a reopened editor shows its last view for
the moment before the fresh one arrives; commands revalidate regardless.

**Phase 34 review follow-up complete (2026-10-02):** an owner-requested
review of the finished Phase 34 found the Gear editor's read model rebuilt
every round per player at about 26 ms with 30 armour pieces in cargo (each
preview YAML-cloned the character). `Company.Equipment` now comes from a
cached view keyed on what its previews read: the leader's character less what
ticks each round (vitals, cooldowns, buff counters, play records), cargo, load
and availability; rebuilt at least every 15 rounds, pruned of offline leaders
on rebuild. Previews share one marshal without the cargo: about 0.9 ms per
unchanged round, 6 ms per rebuild. Also: `Registry.Put`'s unreachable
empty-record deletion removed (every leader keeps a record since 34a, which
holds the 34b pack grant); `Company.Conditions` no longer resends every round
for a ticking countdown (the client counts `seconds_left` down; the change key
uses the end round); effects are tagged Harmful or Helpful when known, from
stat modifiers and new `harmful`/`helpful` markers in `buffs-flags` data
(`BuffSpec.Effect()`), with a duration meter; no doubled full stops in effect
and capability text; clearer pack, burden, conditions and away-member wording.

Independent review: seven findings. Accepted and fixed with regression tests:
(1) effects with only a flag (Bleeding, Poisoned, Stunned) or secret were
labelled Helpful; now three-way from data, tested on the default world's
buffs; (2) the cache key included per-round ticks (buff counters, regen,
cooldowns), so ordinary play rebuilt every round; the key now omits them;
(3) refreshes before and after a round's buff tick keyed end rounds one apart,
resending Conditions after each command; ends within one round now match;
(4) no backstop cleared offline leaders' cached views; pruned on rebuild;
(5) the cached view is now documented as read-only to callers; (6) a
capability line with no description ended in a colon; (7) tests now go
through the provider (`EquipmentViewOf`) and simulate real `Buffs.Trigger`
ticks on both sides of the refresh. Accepted limits: a withdrawn companion
skips buff ticks, so its end round moves and Conditions resends every other
round while withdrawn; `seconds_left` sent from the round refresh can read
one round long. Verification: focused packages, all three browser suites
(effect-card contrast >= 4.5 in every theme), JS lint, make generate, make
validate and go test -race ./... (95 packages) passed. Lua lint not run (no Lua
changed; no Docker or luacheck here).

**33i2 coordinated enemies complete (2026-10-02):** enemy groups
fight by a coordination tier from their average level when the battle
begins (rabble 1–9, band 10–24, drilled company 25–44, veteran 45+; a
template's `coordination` overrides). Higher tiers follow their leader's
focus, go for casters, break heals, and say their focus aloud; a rabble
keeps its own aims with at least 30% noise. Templates may give `role:`
healer, caster, or guardian: enemy healers and casters run the companions'
`strategy.Decide`/`startCast` path (mana, chants, interrupts; single-target
heals only, as a group heal's scope reaches only a company), at 30/50/60/70%
thresholds, a rabble one heal a round; enemy guardians step in through the
attack gates, 0/1/2/2 guards a battle per group. Enemies take light wounds
only (`wounds: none` opts out: skeleton, fungus, bone warden) and recover
out of battle over 188 rounds (5 game hours), prorated by round. `scout`,
`consider`, and the Battle view name the tier and visible roles. Six new
templates (87–93) give tiers 1–3 a healer and a guardian (Frost Lake 310,
Slums 489, Catacombs 113). `help coordination`; assessment, scout, consider,
combat, wounds, guardian, tactics, webclient pages and the Combat lesson
hint updated. See the
[33i design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md)
("Final implementation decisions: 33i2") and
[plan](plans/2026-10-02-phase-33i2-coordinated-enemies.md).

Balance (30 fights a cell, default company vs default/roles groups, on the
final code): median rounds L1 11 → 10/11/12 (tiers 1–3), L5 42 → 39/37/46,
L10 95 → 92/98/94, all within the 125% bound; no clear winner flipped
(company wins L1 13% → 43/23/23%, L5 70% → 70/63/73%, L10 53% → 77/53/53%).
`TestBalanceCoordinated` (ASHVEIL_BALANCE=1) asserts the bounds.

Deviations from the plan: recovery applies to any mob outside a company and
not charmed (not only template-hostile ones); a band also avoids doubling
heals (harmless, as companions do); enemies with no personality keep the old
deterministic weakest pick apart from the tier's noise. The phase 29d
outcome golden was recaptured (an enemy crit's wound rolls its place) and
`breakChant` now asks `enemy()` instead of `!woundable()`.

Independent review: no blocking findings; seven raised. Accepted and fixed
with regression tests: (1) recovery rounded up per round healed small
enemies in ~40 rounds, now prorated so any max takes 188 rounds, and the help
no longer claims fled foes recover; (2) guards and a rabble's heal cap were
counted per player's battle, now per group across battles; (3) a hidden or
not-yet-fighting guardian could step in; (4) a busy leader dropped the
focus, now kept; (5) a hidden member's level could lift the assessed tier.
(7) Coverage added: an enemy caster returns to its player aim, enemy healers
against a solo player, shared guards across two battles. Rejected (6): the
focus line's "closes in" stays singular, as group names are "a band of …";
the band's no-double-heal is kept. Verification after the fixes: make
generate, make validate, make js-lint, and go test -race ./... (96 packages)
passed; the balance table passed its bounds. Lua lint can't run here (no Docker or
luacheck) and no Lua changed.

**34d complete:** Company Status now shows owned members' active effects and
wounds with authoritative duration and mechanical meaning, separate from
persistent bonuses. Away/live, recorded-away/separated, fallen and unavailable states
preserve privacy. Character Skills keeps trained ranks and adds automatic
abilities/spells plus actual field/camp eligibility, including strategy, mana,
autoskill and retirement rules. Camp Cooking is explicitly manual and uses
its owner's configured recipe selector. Character Effects uses safe text and
includes wounds. Existing persistence, readiness and global time are untouched.
Help/tutorial and responsive keyboard/focus checks updated. Independent review
accepted and resolved missing camp cooking with real command/GMCP regressions;
follow-up found no further blockers. A test fixture's stale room membership was
fixed and its paired regression passed three runs. All affected packages, all
three browser suites, generate/validate, JS/Lua lint and the full race suite
passed. [Verification](plans/2026-10-02-phase-34d-verification.md).
Integration with 33h3 (`f4e28dda`) preserved relocation/separation and adds
explicit recorded separated wounds. Independent integration review found no
blockers; real passage/reload/rejoin GMCP regression, all browser suites,
generate/validate, JS/Lua lint and the full race suite passed on the integrated
code. The design, delivery plan and handoff now mark all four slices complete.
Phase 34 is complete; stop here and hand over for the owner's next session.

**33h3 relocation and separation complete (2026-10-02):** every move that
isn't an exit (scripted passages, traps, ropes, portals, an inn room, jail,
a quest `roomid` reward, a journey's arrival and its crash recovery, death,
the tutorial, admin teleport) now brings the living companions standing
with the leader through one step, `company.RelocateCompany(leader, origin,
room)`. A living companion elsewhere, or one away from its leader for two
rounds, is saved and taken off the map as separated, and rejoins beside the
leader after `SeparationRounds` (15, about a minute) of online time once the
leader is out of any fight, journey, or camp rest; a relog never brings it
back early, and no world time advances. Player-requested passages refuse in
a battle (`ActorObject.InBattle()`). The dead, the fled, the herd, and cargo
are unaffected. `help separation`, Departure hint; company, travel, death,
morale, mount, cargo, readiness and retreat pages updated. Owner approved
the defaults (with a clarification on dead companions); see the
[33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h3") and
[plan](plans/2026-10-02-phase-33h3-relocation.md). 33h is complete.

Independent review: ten findings, all accepted and fixed with regression
tests. They were: the `portal` help alias broke 33f1's retired-skill check
(dropped); a failed rejoin spawn was announced and never retried (it now
stays separated and due); inn beds and camp rest tiers counted the separated
(skipped); a separated companion could be fed (`ErrAwayMember`); admin
teleport was untested and silent (test, line added); `help retreat` was
stale (updated); the unused `RejoinSeconds` was removed; JS indentation; and
the refusal line now reads "Not while you are fighting." Speculative findings:
S1 (the sweep firing on ordinary follow) was rejected, because a follower
trails by one turn and the sweep needs two round boundaries (about eight
seconds) away and out of a fight; S2 (separation saves gear early) was
accepted as the same window `BeginFlight` already has; S3 (the tutorial
gate) is as intended, and its wiring test now walks the companions instead
of relying on the old recall. Lua lint can't run in this container (no
Docker or luacheck) and no Lua changed. Verification after the fixes:
make generate, make validate, JS lint, and go test -race ./... passed.

**34c complete:** Character Gear now edits main-character equipment by slot,
including Pack, with exact-instance compatible cargo choices, unavailable reasons,
and authoritative current/after comparisons. Previews and commands share the same
hand, curse, capacity and asset-journal rules; stale actions revalidate server-side.
Comparisons include active edge bonuses/strikes for both hands, core modifiers,
worn burden/dodge and company capacity. Legacy Gear and companion commands remain
available. Help, tutorial pointers and keyboard/focus/narrow browser checks updated.
Independent review found missing sharpening information; fixed with regression
coverage, and follow-up review found no further issues. Additional regressions
cover permanent buffs retained by the other hand and legacy-feed availability.
Real command, save/recovery, private GMCP/reconnect, help and browser checks passed;
make generate/validate, JS/Lua lint and go test -race ./... passed.
Integration with 33h2 (63f79b32) passed independent review, browser checks,
generate/validate, JS/Lua lint and the full race suite; companion readiness
snapshots and recovery remain intact.
34d is complete. [Session handoff](plans/2026-10-02-phase-34-session-handoff.md).

**33h2 readiness and recovery complete (2026-10-02):** companions keep
their health and mana across logout, restart, copyover, and crash (as of
the last save) instead of refilling; they recover online out of battle as
players do, never offline; an inn stay restores the company, a live
level-up no longer refills a companion, and resurrection wakes it at half.
`help readiness`. See the [33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h2", owner-approved) and
[plan](plans/2026-10-02-phase-33h2-readiness-recovery.md). 33h3
(relocation and separation) followed (above).

**34b complete:** each member has one assigned Pack slot; new leaders and recruits
receive a 10 kg cloth knapsack. Shared cargo capacity now comes from packs on
living present carriers plus eligible horses, with no base/Strength allowance.
Only unequipped cargo consumes that capacity; Pack contributes no combat burden,
stats, defense or worn buffs. Migration preserves exact instances and journals
once-only grants; capacity loss preserves overloaded cargo and recovery actions.
Inventory shows assigned container availability and one shared cargo list.
Help, tutorial pointers and migration release notes updated. Independent review
resolved four findings (combat modifiers, 10 kg creation kits, armor-removal
coverage, stale help) and a follow-up weapon-defense regression. Final focused
checks, browser checks, make generate/validate, JS/Lua lint and go test -race ./...
passed. Existing creation, death and mount fixtures now exercise the new rules;
formation integration permits valid upkeep retargeting after clearing a rear foe.
Integration with 33h1 passed independent review, focused command/module checks,
browser checks, generate/validate, JS/Lua lint and the full race suite. Owner
requests a new session between slices; 34c is now complete.
[Next-session context](plans/2026-10-02-phase-34-session-handoff.md).

**34a complete:** shared browser Inventory shows cargo without member worn or
personal blocks; panel foreground inheritance and two light-theme secondary
colors corrected. Solo leaders stay at 2,2; recruitment uses deterministic
vacancies; old formations receive a durable one-time backfill. Last-member loss
recenters the leader and failed enlistment/migration restores placement.
[Phase 34 design](designs/2026-10-01-phase-34-company-ui-logistics-design.md)
and [plan](plans/2026-10-01-phase-34-company-ui-logistics-plan.md).
Independent review found one integration-coverage gap; resolved with real
PlayerSpawn, persisted reload, GMCP/text/combat checks, and repeat-login manual
clear preservation. Follow-up review found no further issues. Verification:
focused packages, help/tutorial pointers, all dock browser checks including
4.5:1 load/label contrast in every shipped theme, make generate, make validate,
JS/Lua lint, and go test -race ./... passed. Initial race run hit the existing
random-hit edge test; targeted rerun and final full run passed. Integration against 33h1 (ecd3255e)
passed independent review, focused checks, browser checks, generate/validate,
JS/Lua lint and the full race suite. 34d is complete.

**33h1 growth and contracts complete (2026-10-02):** the owner asked to
continue with 33h ahead of 33g's catalog slices. Companion
training is now derived from level by archetype growth weights plus an
optional focus (`company growth`), re-derived on every spawn and level
change so no level loop mints points; quests with `companyexperience` are
contracts paying every companion with the leader. `help growth`,
`help contracts`. See the [33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h1") and
[plan](plans/2026-10-02-phase-33h1-growth-rewards.md). 33h2 (readiness and
recovery) is next, then 33h3 (relocation).

**33g management complete:** pooled treasury (owner choice), shared
instance-preserving cargo, explicit equipment assignment/removal/comparison,
claimed/public loot windows and opt-in autoloot. Equipment presets removed at
the owner's request. Durable equipment journals and cargo migration protect
restart recovery. Browser controls, indexed help, and tutorial hints updated.
See [design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md) and
[plan](plans/2026-10-01-phase-33g-equipment.md). Independent review and
required checks passed;
equipment catalog and tiers are not shipped by this slice.

**Equipment design approved:** [weapon families, six tiers, and armor paths](designs/2026-10-01-equipment-tiers-design.md) records
the owner's approved equipment direction for 33g
and later progression/content work. Design documentation is complete; gameplay
is not implemented, and the phase order below is unchanged. Documentation
verification: whitespace and relative Markdown links checked; no Go changes.

The expedition/company loop, onboarding, combat presentation (29a–29f),
status effects, wounds, tactics, guardians, interrupts, wind-ups, morale
and mercy (30a–30e), active defense (30g2), and personal load (30g3) are shipped. The Phase 32
play-test improvements are implemented; 32b's status still carries an outstanding review note (see its retained plan).

**Current owner priority:** finish Phase 33a–33i in order, choosing the lead's
recommended defaults without further confirmation (2026-10-01). 33a–33d are
complete; 33e (automatic class abilities) and **33f1 (skill and charm
retirement)**, **33f2 (expedition specialists)**, and **33f3 (camp
specialists)** are complete, so 33f is done; **33i1 (group assessment)**
is complete too (33i2 remains); 33g management and its cargo prerequisite are
complete, reviewed and verified. Equipment
catalog delivery and 33h remain. Phase 30g3 is also complete;
30g4 (progression), 30g5 (the action meter), 30g6a (combat fixes and harness) and 30g6 (stat edges and tuning) are complete.

**Combat tempo queue:** Phase 30g, [combat tempo, personal load, and active
defense](designs/2026-09-30-phase-30g-tempo-defense-design.md), whose
decisions the owner settled on 2026-09-30. 30g1 (the balance harness and
baseline), 30g2, 30g3, 30g4, and 30g5 are done; each later slice is measured against 30g1:

1. **30g2, active defense and armor:** complete (work log below): one
   defense per strike: block with a shield (no dodge), else parry a
   melee strike or dodge; the shield's ×1.5 armor removed; the shield
   bash moves to a blocked melee strike (5–20% by Strength); armor's
   suffix reads `absorbed`; `help defense`.
2. **30g3, personal load:** complete (work log below): worn and carried
   weight against a Strength-based capacity (cargo and mounts never
   count); burden lowers dodge only; burden words in `status`, `look`,
   `scout`, and the web Overview; `help burden`.
3. **30g4, progression:** complete (verification above): automatic stats grow in steps every 5 levels;
   HP by archetype, in small numbers; an XP knee at level 60.
4. **30g5, the action meter:** complete: turns from effective Speed and burden,
   one opening turn, at most two a round, no whole-turn banking.
5. **30g6, tuning:** HP, damage, and healing set so a no-focus even 5v5
   lasts 10–15 rounds at levels 1–60; the harness asserts it. It also
   takes 29f's cadence retune.

Phase 30e (morale and mercy) is complete. Phase 30f (battlefield
conditions; mounted combat remains excluded) is implemented on
`phase-30f-battlefield` and merged via [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12).
Other remaining limitations are listed below.

**Future company gameplay:** the owner endorsed the 2026-10-01 gameplay review
and requested [Phase 33a–33i designs](designs/2026-10-01-company-gameplay-roadmap.md):
command consistency, friendly-effect scopes, retreat, allied companies,
automatic class abilities, specialists, equipment/loot, progression/recovery/
relocation, and group assessment/coordinated enemies. These are planning
records now authorized for implementation by the owner, with open defaults
delegated to the lead. 33a–33e are complete.

## Deferred preparation designs

- Owner endorsed company UI/logistics improvements on 2026-10-01, including
  automatic recruit formation placement. [Phase 34 design](designs/2026-10-01-phase-34-company-ui-logistics-design.md)
  and [delivery plan](plans/2026-10-01-phase-34-company-ui-logistics-plan.md)
  split delivery into 34a UI/formation, 34b assigned packs and cargo capacity,
  34c equipment editor, and 34d effects/current capabilities. Design and delivery defaults approved by the owner;
  34a–34d implemented and verified as recorded above.

- Owner requested a [weapon poison design](designs/2026-10-01-weapon-poisons-design.md)
  for future implementation on 2026-10-01: shop-bought temporary blade coatings,
  explicit per-member/per-blade camp assignments, and later camp recipes.
  Recorded only; no phase scheduled or gameplay implemented. Balance numbers
  remain proposed defaults.

- Owner requested the [camp consumables design](designs/2026-10-01-camp-consumables-design.md)
  on 2026-10-01: fortifying broth, warming draught, cooling salve, scent-masking
  paste, watch incense, weapon oil and antidote draught. Includes proposed
  preparation limits, rest/route lifecycle, acquisition and later crafting.
  Documentation only; no phase scheduled or gameplay implemented.

## Planned zone encounters

- Owner requested [zone room encounters and wandering parties](designs/2026-10-01-random-room-encounters-design.md)
  on 2026-10-01, with an [implementation plan](plans/2026-10-01-random-room-encounters-plan.md).
  Encounter-enabled rooms roll on entry using zone enemy tables; random battles
  may become the main ordinary enemy source while public wandering parties remain.
  Includes proposed grace, multiplayer ownership, durable recovery and zone
  migration. Planned only; phase scheduling and proposed defaults remain open.

## Planned branching class progression

- Owner requested a [class progression review and branching design](designs/2026-10-01-branching-class-progression-design.md)
  with an [implementation plan](plans/2026-10-01-branching-class-progression-plan.md)
  on 2026-10-01. All five base lineages gain proposed advanced/elite paths,
  individual alignment eligibility and explicit player promotion choices.
  Includes current-mechanics audit, old-save migration, ability inheritance,
  death retention and 30g4/33h dependencies. Planning only; levels, gates,
  branch signatures and implementation scheduling remain proposed.

- Owner expanded the class direction with an [extended companion catalogue](designs/2026-10-01-expanded-companion-classes-design.md):
  twenty additional advanced/elite paths including Sorcerer, plus eventual
  nonhuman humanoids and creature recruits such as Hellhounds and Golems.
  Species, class and creature growth remain separate; staged delivery includes
  complete upkeep/recovery/gear rules. Planning only; external reference pages
  were blocked by network policy, so the proposed roster is not a verified
  reproduction of Ogre Battle 64 or Unicorn Overlord.

- Added [further class-reference extractions](designs/2026-10-01-class-reference-extractions.md)
  from the owner's supplied Ogre Battle/Unicorn Overlord roster: seven new
  humanoid candidate paths, commander/prestige direction, Cerberus, additional
  creature families and dragon affinities. Includes consolidation of equivalent
  names and explicit equipment/flight/form/aquatic dependencies. Planning only.

## Planned skills and spell progression review

- Owner requested a [skill/utility/spell progression review](designs/2026-10-01-skill-spell-progression-design.md)
  and [implementation plan](plans/2026-10-01-skill-spell-progression-plan.md)
  on 2026-10-01. Inventories twelve shipped skill definitions and eleven spell
  definitions; separates trained ranks, class/level unlocks, field/camp specialists
  and automatic combat. Preserves approved retirements and flags stale prose,
  level/class inheritance and the no-battle-items conflict in earlier antidote
  proposals. Proposed new milestones/capabilities only; no gameplay implemented.

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
| 30f | Battlefield conditions | Complete: [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12), reviewed and merged 2026-10-02. [Approved design](designs/2026-10-02-phase-30f-battlefield-design.md), [verification](verification/phase-30f/verification.md). Mounted combat excluded |
| 30g | Combat tempo, personal load, and active defense | In progress: [design](designs/2026-09-30-phase-30g-tempo-defense-design.md). 30g1 complete ([plan](plans/2026-09-30-phase-30g1-balance-harness.md)): the balance harness and baseline (no-focus 5v5: median 8 / 32 / 68 rounds at levels 1 / 5 / 10). 30g2 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g2-active-defense.md)): block, parry, or dodge, one per strike; the shield's ×1.5 removed; the bash on a blocked melee strike (5–20%); `absorbed`; `help defense`. 30g3 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g3-personal-load.md)): personal load and burden (dodge × (1 − 0.6 b)); `help burden`. 30g4 progression and 30g5 action meter complete; 30g6a fixes and harness ([verification](plans/2026-10-04-phase-30g6a-verification.md)); 30g6 stat edges and tuning, acceptance passing ([amendment](designs/2026-10-04-phase-30g6-amendment.md), [verification](plans/2026-10-04-phase-30g6-verification.md)) |
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
| 33d | Multiplayer Parties and Allied Companies | Complete: [design](designs/2026-10-01-phase-33d-allied-companies-design.md), [plan](plans/2026-10-01-phase-33d-allied-companies.md); durable alliances, explicit consent, contribution-based XP, fixed shared loot claims and independent company authority |
| 33e | Automatic Class Abilities and Combat Roles | Complete: [design](designs/2026-10-01-phase-33e-automatic-class-abilities-design.md), [plan](plans/2026-10-01-phase-33e-class-abilities.md); automatic Tackle, Opening Strike and Aimed Shot, coordinated healing, mana reserve, `strategy [who] abilities on\|off` and `reserve`, `help abilities` |
| 33f | Company Specialists and Expedition Skills | [Design](designs/2026-10-01-phase-33f-company-specialists-design.md) rewritten with the owner (2026-10-01) in three slices. 33f1 complete ([plan](plans/2026-10-01-phase-33f1-skill-retirement.md)): retired peep, portal, tame, change form, scribe, sneak, bump, pickpocket, pray, and backstab; mercenary hiring, mob befriend, and the charm scripting API; one-time training-point refund; Protection capped at 3. 33f2 complete ([plan](plans/2026-10-01-phase-33f2-expedition-specialists.md)): Read the Trail, Keen Eye, Pathfinder, Weather Sense, Haggle, `company specialists`; `search`, stock `track`, and `trading` retired. 33f3 complete ([plan](plans/2026-10-01-phase-33f3-camp-specialists.md)): camp raids and Camp Watch, Field Smith, Vigil, Forage, `camp cook` |
| 33g | Company Equipment and Loot | Management/cargo/treasury complete; reviewed and verified. [Design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md); presets removed by owner. Catalog delivery remains a separate slice. |
| 33h | Company Progression, Rewards, and Expedition Continuity | [Design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md) in three slices. 33h1 complete ([plan](plans/2026-10-02-phase-33h1-growth-rewards.md)): derived archetype growth, `company growth` focus, contract quests; `help growth`, `help contracts`. 33h2 complete ([plan](plans/2026-10-02-phase-33h2-readiness-recovery.md)): durable companion vitals, online-only recovery, inn restore, half-vitals resurrection; `help readiness`. 33h3 complete ([plan](plans/2026-10-02-phase-33h3-relocation.md)): relocation and separation; `help separation` |
| 33i | Company Encounter Assessment and Enemy Roles | [Design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md) in two slices. 33i1 complete ([plan](plans/2026-10-01-phase-33i1-company-assessment.md)): the company's assessment of a visible enemy group in `scout [group]`, `consider [enemy]` (one-on-one odds retired), and the Battle view's outlook; `help assessment`. 33i2 complete 2026-10-02: coordination tiers, enemy roles, light enemy wounds and recovery; `help coordination` |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

### 38b faith routes design approved (2026-10-05)

- **Why:** the draft had four points the owner had not decided.
- **Decided:** a character below its elite gate at level 30 waits on its
  advanced route with its ranks and promotes once its alignment recovers;
  the Demon's broken binding stays; Blood Priest healing keeps its 25% mana
  surcharge; routes are final at launch, with retraining a later design.
  Recorded in the [faith routes design](designs/2026-10-05-faith-routes-design.md).
  Docs only; 38b still waits on 38a.

### Phase 33h2: company readiness and recovery (2026-10-02)

- **Why:** logout, restart, copyover, a crash, a level-up, and resurrection
  all refilled a companion for free, and companions never regained health
  on their own. The lead explained the plan and defaults in plain language
  first, as the owner asked; the owner approved them unchanged.
- **Delivered:** `MemberState.Vitals` on the 22b snapshot seams, resolved
  at every spawn by `Vitals.Resolve` (points clamped to the wound limit and
  mana maximum, never below 1; nil, an old record, spawns full once;
  `Percent` for resurrection's half). Online health regeneration for
  companions beside 32d's mana (`internal/hooks`), counted by exposure; a
  live level-up keeps a companion's vitals; an inn's Well Rested grant
  restores the leader and live companions. New `help readiness`; company,
  health, inn, camp, resurrect, wounds, heal, and quit pages updated; rest
  lesson and Departure hints.
- **Independent full-diff review:** no blockers and no remaining free
  refill path. Accepted and fixed: the `recover` alias collided with the
  brawling topic, so `help recover` resolved at random (removed; regression
  test plus a guard that no help alias names two topics); a level-up
  lowered health above a fresh wound's limit (now never lowers; regression
  test); help wording on when a lowered limit applies, the crash window
  (up to about a quarter of an hour), the inn sentence's placement, and an
  over-long line; the resurrection share's comment now says it is fixed
  apart from death's `RespawnVitalsPct`; coverage added for a morale-flight
  return and a snapshot taken at 0 health. Accepted as known risks, not
  changed: an inn restore reaches disk only at the next save seam, so a
  crash in between loses it (the same window as gear, wounds, and the
  leader's own file); and a Well Rested grant handled before a returning
  leader's companions respawn would miss their restore (plausible, not
  reproduced; the same ordering the 30b wound knitting already has).
  Not added: separate tests for the in-battle regen skip (shared with 32d's
  tested mana path) and the stale-mob replacement branch (unchanged 22b
  code; it now carries vitals through the same `Snapshot` call the logout
  and save tests exercise).
- **Verification (after review fixes):** `make generate` (no diff),
  `make validate`, and `go test -race ./...` (95 packages) passed with no
  failures. No JavaScript or Lua changed, so those lints were not
  applicable.
- **Integration:** owner approved merge and push. `origin/master` had moved
  to 34a/34b (`eb21ae3`); merged it into the branch (status log conflict
  only), then `make generate` (no diff), `make validate`, and
  `go test -race ./...` (95 packages) passed again before merging.

### Phase 33h1: companion growth and contracts (2026-10-02)

- **Why:** owner requested 33h next; open defaults chosen by the lead under
  the 2026-10-01 standing authority and recorded in the design.
- **Delivered:** `domain.Deal` (prefix-stable weighted rotation) over
  per-archetype `Growth` weights in the archetype config; training derived
  from `characters.StatPointsAtLevel` on spawn, live level-up
  (`company.RetrainCompanion` replaces `AutoTrain` for tracked companions),
  `company archetype`, and `company growth [member] [stat|balanced]`
  (+2 focus, in memory until the next save seam, refused in battle).
  `rewards.companyexperience` makes a quest a contract; Rodric's Rats and
  The King's Shadow ship as contracts. Also fixed the 32e award retraining
  every later companion once any companion levelled. No save migration:
  training is never saved.
- **Independent full-diff review:** no blocking findings. Accepted and
  fixed: `company growth` saved the company file outside the 22b seams
  (now in memory only, regression test); missing real-entry coverage for
  `company archetype` retraining and resurrection weights (tests added);
  contract message and help ignored the experience scale (now say "before
  scaling"); member names with spaces refused (stat is the last word);
  `company growth` missing from the company page's command list; "pays
  once" overstated (reworded). Rejected: none.
- **Verification (after review fixes):** `make generate`, `make validate`,
  and `go test -race ./...` passed with no failures. No JavaScript or Lua
  changed, so those lints were not applicable.

### Phase 33g: cargo, treasury, equipment, and loot management (2026-10-01)

- **Integration:** owner authorized merging the reviewed management slice
  (`cf8434f1`) to `master` and pushing to `origin`; no gameplay changes after
  the final verification below.
- **Owner choices:** pool carried gold into one treasury; exclude equipment
  presets. Completed the required shared cargo prerequisite and deliberate
  equipment management on `phase-33g-equipment`, based on origin/master
  `022100ca`. [Design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md)
  and [plan](plans/2026-10-01-phase-33g-equipment.md). Equipment tiers/catalog
  remain separate content delivery; 33g is not closed
  as a whole by this management slice.
- **Delivered:** instance-preserving shared cargo and pooled treasury, exact
  member/item equipment assignment, removal and comparison, recoverable
  company/user asset operations, idempotent legacy cargo migration, physical
  pack counting, `loot` and opt-in `autoloot`, two-hour private/two-hour public
  spoils, and battle/ownership checks. Updated real item/meal/cooking/camp
  consumers, text and browser inventories, help and tutorial pointers.
- **Independent full-diff review:** accepted and fixed pending-recovery incoming
  asset loss, missing leader GMCP inventory updates, trade-in pack-capacity
  loss, remaining name-based browser actions, and stale help/corpse claims.
  Added shared cooking/meal coverage and a real saved-user cargo reload check.
  Lead additionally fixed autoloot collecting foreign public spoils, stale
  exact remove references, migration rollback markers, and spawn UUID identity.
  Confirmed findings have regressions; no findings rejected. Follow-up reviews
  found no production blockers. Cooking branch tests use a provider double;
  atomic cargo replacement and the equipment/GMCP path use real implementations.
- **Verification:** `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint JSHINT=/workspace/ashveil-env/js/node_modules/.bin/jshint`, and
  the Chromium dock-window harness passed. Full-suite testing caught old mount
  assertions for private inventory; updated them while preserving burden,
  physical pack, saddle, meal, saved-cargo, reload and no-clock-advance checks.
  Relative Markdown links and whitespace checked. No Lua changed.
- **Retained limit:** uncollected runtime corpses disappear on restart, as the
  engine already did; equipment and collected cargo retain their exact instances.

### Phase 33i1: company encounter assessment (2026-10-01)

**Why:** the owner asked for the early 33i slice, in parallel with 33f3:
players couldn't judge whether their company could take a group; the old
`consider` compared the leader alone with one creature
([design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md),
"Final implementation decisions: 33i1"; [plan](plans/2026-10-01-phase-33i1-company-assessment.md)).

**What shipped:** `internal/assessment`, a read model: each counted
member's dice-free expected damage (`combat.ExpectedDamage`) against the
visible foes it can reach (formation reach, unplaced fails open), current
health as staying power; five words (an easy fight … hopeless) and "it
could go either way" near even or a band line; never numbers. Counted:
the leader and companions with them who can fight; fled, fallen,
awaiting, separated, down, and out-of-the-fight members named as not with
you; pets left out; party allies noted, not counted. `scout [group]` ends
with it; `consider [enemy]` gives it for the enemy's whole group (the
owner retired the one-on-one odds) and refuses players, own companions,
and harmless loners; `Company.Battle.outlook` and the web Battle view's
"Outlook" line. Visibility as scout: hidden foes left out, nothing in the
dark. No state, RNG, or world-time change; no durable state, so nothing to
migrate. Help `assessment` (new), `consider` (template replacing GoMud's
page), `scout`, `combat`, `webclient`; the Combat lesson's scout hint.

**Tests:** pure estimate tables (bands, closeness, reach, fail-open, no
one able to strike, no fighters); `ExpectedDamage` draws no dice (seeded
global source unchanged); real `scout`/`consider` in the company brawl
(counted and missing members, wounds, burden, a separated companion,
formation reach agreeing with scout's marks, hidden foes, darkness,
allies, refusals, no round/state change); the live `Company.Battle`
outlook and its absence in the dark; payload test; help render and
aliases; tutorial pointers; dock-windows browser check.

**Independent review** (default model, report-only). Accepted and fixed
with regression tests: (1, major) a company with nobody able to fight read
"a hard fight, could go either way": now hopeless; (2) members standing
here but down or withdrawn were labelled "away": now named with the
reason, the leader too; (3) `consider` by name could land on a hidden
namesake and miss the visible one: it matches visible group members only;
(5) `consider` assessed shopkeepers and other harmless loners: refused, as
scout doesn't list them; (6) an own companion now gets "travels with you";
(7) "Allied companies" reworded "Allies". Accepted as is: (4) reach uses
scout's alive map, so an unseen front-rank foe can put the one behind out
of reach (the design's rule, matching scout's marks); (9) the outlook is
recomputed on each Battle refresh (≤5×9 dice-free estimates on the game
loop; acceptable). Rejected: (8) remove `combat.CombatOdds`: kept as
engine API for upstream parity. Pre-existing, not changed: `scout`'s and
`attack`'s lone-mob lookup (`enemyparty.FindGroup`) can still land on a
hidden namesake.

**Checks:** after the review fixes, `make generate`, `make validate`,
`go test -race ./...` (all passing), and `make js-lint` once; the
dock-windows Chromium check (all passing, outlook included).

### Phase 33f3: camp specialists (2026-10-01)

**Why:** the owner asked for the camp skills from the 33f review and
approved the proposed raids and the simple camp cooking
([design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f3-camp-specialists.md)).

**What shipped:** camp raids (15% per Old Kings Road rest, rolled at rest
start and saved; a pair of new level-3 road brigands, mob 86, through the
shared `enemyparty.SpawnAmbush`); Camp Watch (warrior, 25%/level): spotted,
the rest still counts; unspotted or with the leader logged out, no Rested
and no rewards; raiders left standing when their leader falls or leaves
are sent off. Field Smith (warrior): 20 + 5/level strikes of edge per
whetstone use. Vigil (cleric): +level loyalty to present companions, cap
60. Forage (ranger): 1 + level/2 finds into cargo, within capacity.
Forage and Vigil come at most once per 15 minutes, are collected at the
camp after any fight, and are paid exactly once through operation IDs
saved with the cargo (`Cargo.Applied`) and the company record
(`Record.AppliedOps`). `camp cook`: the hearth's three dishes from pack and
cargo, into cargo. Help `campwatch`, `fieldsmith`, `vigil`, `forage`;
`camp`, `cooking`, `sharpen`, `specialists`, `autoskill`, `combat`
updated; a Camp lesson hint.

**Tests:** raids planned, durable, resolved once through the round pass
(unspotted, spotted, logged out, away, failed spawn), restart; raiders sent
off when the leader falls; rewards through rest completion and the round
pass (once on retry, after a fight, cooldown, forfeited away, capacity);
`camp cook` through the camp command (skill gates, battle refusal,
restore on a failed withdrawal, capacity); Field Smith through `sharpen`;
cargo and loyalty operations in their modules, including the real YAML
decoders; camping registry YAML round trip; shipped config; help render.

**Independent review** (default model, report-only; a first run was cut
off by an interrupted compaction and rerun). Accepted and fixed: (1,
blocking) the company decoder dropped `AppliedOps`, so a vigil could be
paid twice after a restart; (2, blocking) rewards owed while the leader
still fought a spotted raid were dropped: they now wait; (3) free rests
farmed rewards: once per 15 minutes; (4) raiders outlived their target and
could turn on other campers: sent off; (5) logging out dodged raids: it
now spoils the rest; (6) `camp cook` could lose ingredients on a failed
save and skipped capacity: cargo first with restore, capacity checked,
pack-full fallback; (7) rewards used the leader's room at payment: now
the camp's, forfeited elsewhere; (8) the watch was looked up under the
camping lock: moved out; (9) a failed spawn still spoiled the rest; (11)
"food and water" and the `help combat` link; (12) coverage gaps. Rejected:
(10) a retried forage message may name a re-rolled find (the deposit
itself is deduplicated; cosmetic, rare); goimports grouping (gofmt is the
repo's check). Known, unchanged: travel ambush raiders share finding 4's
leftover-mob behaviour (33c/33h scope).

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green). No JavaScript changed.

### Phase 33f2: expedition specialists (2026-10-01)

**Why:** the owner asked for the expedition skills proposed in the 33f
review ([design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f2-expedition-specialists.md)).

**What shipped:** for each capability the best present, living member of
the leader's own company acts (the 17b resolver, through
`archetypes.BestSpecialist`), switched by `autoskill`, resting in the
leader's battle. Read the Trail (ranger): reports hostile groups beyond
the exits on each step and on a bare `track`, more detail per level, two
rooms out at 4 (memory-loaded rooms only); warns of ambush routes at
departure and leads the company around an ambush 20%/level (the pause
becomes tracks). Keen Eye (rogue): spots secret exits on entry, remembered
in `KnownSecretExits` and shown in room text, Room GMCP, the world map, and
`map`. Pathfinder (ranger): 5%/level less strain on rough ground, never
below road. Weather Sense (wizard): zones foretell their next condition
(`ZoneWeather.Next`), so `weather` forecasts truly, more per level.
Haggle (rogue): 2%/level better market prices; every sale stays below the
cheapest possible buy-back. `company specialists`. The `search` command and
skill, the stock `track` command, and the dead `trading` skill (and the
Merchant profession) are retired with the 33f1 refund. Help pages
`specialists`, `trail`, `keeneye`, `pathfinder`, `forecast`, `haggle`;
updated company, weather, market, strain, autoskill, archetype, travel,
and job pages; a Departure hint.

**Tests:** real `go` steps (trail by a companion ranger and a level-1
player, no tracker, autoskill off; Keen Eye spotting, memory, room details;
no spotting without a rogue at a low roll); `track`, `specialists`,
`company specialists`; resolver ownership (separated, downed, other player,
switched off, in battle); walking strain and `strain`; market buys, sales,
listing, and a sweep of every shipped good, stock, standing, and haggle
pairing that no round trip profits; weather command at each level,
neighbour zones, foretold weather on advance and on recovery of old saves;
travel warning, evasion, and resume/return/recovery after it; Room GMCP;
`KnownSecretExits` save/reload; refunds; help render and index.

**Independent review** (default model, report-only). Accepted and fixed:
(1, blocking) a haggled buy-and-sell loop profited on steep goods (and by
toggling the haggler): the sale cap now always applies at the best
possible haggle; (2, blocking) an evaded ambush left a pause the route
validation rejected, stranding the journey: `CanFireAs` accepts it (and,
pre-existing, a weighted table's kinds); (3) specialists acted in the
leader's battle: gated in the resolver, `track`, and Keen Eye; (4) route
config never read `Kinds` or the ambush mob: parsed and tested (no shipped
route has an ambush yet: a content gap, not a code one); (5) coverage for
durability, Room GMCP, and refunds added; (6) the world map and `map`
still used the visited-only rule: fixed; (7) trail now reads only rooms
already in memory; (8) neighbour forecasts use weather zones; (9)
`autoskill` alignment; (10) the stale trading page (the skill is retired),
README, and unused trail colours; (11) trade notes name the gold actually
won. Not changed: Room GMCP needs no forced resend, since Keen Eye runs in
the step before the move's queued room update is built.

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green). No JavaScript changed.

### Phase 33f1: skill and charm retirement (2026-10-01)

**Why:** the owner reviewed GoMud's stock skills against a game about
leading a company on expeditions and asked to remove those built for a
lone player or that clash with Ashveil, and all player-facing charm
([33f design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f1-skill-retirement.md)).

**What shipped:**
- Player commands `peep`, `portal`, `tame`, `changeform`, `scribe`,
  `sneak`, `bump`, `pickpocket`, `pray`, and `backstab` are gone; their
  skills (peep, portal, tame, change form, scribe) are deleted; Protection
  stops at level 3 (level 4 granted only pray). Mob-side sneak, portal, and
  backstab stay.
- Charm: no taming (skill, spell, tame mastery, stat mod), no mercenaries
  for hire (shops skip `mobid` stock; the Bonecrafter's skeleton is gone),
  no mob `befriend`, and the scripting API's charm setters and tame calls
  are removed. Companions keep the internal charm link. Charms are
  runtime-only, so none survive the deploy; pets stay.
- A player still holding a retired skill (or Protection 4) gets the
  training points back once, at their next spawn, saved with the change.
- Archetypes (rogue, wizard, ranger), professions (Explorer and Monster
  Hunter removed), trainers (rooms 830, 160), the Whispering Wastes
  obelisk, the long whip, the shipped admin user, help pages, keywords,
  and the admin scripting reference follow. `search` and `track` stay
  until 33f2 replaces them.

**Tests:** refund through the real spawn listener, once and saved; no
archetype claims a retired skill; retired commands and help topics gone;
a mercenary shop neither lists nor sells (fails on the old code); a
world-data guard (trainers, professions, item stat mods, quests,
scripts); every indexed help topic and alias opens a page (pre-existing
GoMud gaps named). The 29d combat capture was regenerated: removing the
tame-learning roll on each kill shifts the seeded RNG (a dummy roll
reproduces the old capture exactly).

**Independent review** (default model, report-only). Accepted and fixed:
(1) backstab needed `sneak`'s hidden state, so it was retired too
(Opening Strike covers battles); (2) `hire` still indexed under shops;
(3) the skulduggery description named pockets; (4) Protection 4 granted
nothing: capped at 3 with refund (skulduggery's levels still drive traps
and utilities, so no refund); (5) stale admin scripting docs and
AGENTS notes; (6) dead mercenary listing code, `canCarryStolen`, and the
`backstab`/`pickpocket` autocomplete; (8) the admin user's tame mastery
block and a level-0 entry's empty refund notice. Rejected: (7) syncing
`world/empty` (unshipped upstream seed world, never loaded by Ashveil or
its tests; its retired pages are inert); the `MercHirePricePerLevel`
config key and its admin docs stay (engine config, now unused).

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green), `make js-lint`, and `scripts/browser/dock-windows-check.mjs` in Chromium over localhost HTTP (all dock window checks passed).

### Phase 33e: automatic class abilities (2026-10-01)

- **What:** warriors tackle (knocked down; breaks a foe's chant or
  wind-up; the whole turn), rogues strike an opening on a downed,
  stunned, staggered or exposed foe, and rangers aim a shot (both the
  round's own swing, its first landed blow a critical hit), each on its
  own by archetype (companions) or trained skill (players: brawling,
  skulduggery, track), with cooldowns of 4/2/3 combat rounds. Healers no
  longer double-heal one ally; `strategy [who] reserve [percent]` keeps
  mana back from attack spells; `strategy [who] abilities on|off`. Durable
  fields `no_abilities`/`reserve` (old saves load as on/0); cooldowns are
  runtime only. GMCP `Company` strategies and the web Combat setup show
  abilities. `help abilities` (new) and updated `strategy`, `combat`,
  `company`, `tactics`, `archetype`, `brawling`, `skulduggery`, `track`;
  a practice-fight tutorial hint. The backstab crit no longer reports a
  crit on a round with no landed blow, and its shouted prefix is gone.
  No ability uses an item (owner rule). Decisions: the design's "Final
  implementation decisions"; balance is left to 30g6.
- **Why:** owner priority to finish 33a–33i in order with the lead's
  recommended defaults.
- **Independent review** (report-only subagent) — accepted and fixed, each
  with a regression test: (1) abilities skipped the formation gate, so a
  reaching warrior could tackle a back-row caster the front row shields
  (now the swing's interception rules apply, and a tackle needs
  hand-to-hand reach); (2) companions still used abilities while the
  company prepared to retreat (now none do); (3) a tackle skipped the
  tackler's cancel-on-combat buff removal; (5) help said "rest 4 rounds"
  where the ability is ready every 4th round (now "at most once every 4
  combat rounds"); (6) the tutorial hint promised a "sure" crit; (8) added
  tests for interception, retreat, a casting member, and a readied strike
  set back, and removed an unused test seam. (4) a cooldown spent when the
  foe falls first is accepted and now documented. (7) `help warrior` and
  `help ranger` are GoMud job pages, not archetype pages: rejected as
  stated; `help archetype` carries the change and the design text was
  corrected. Ownership negatives (charms, allies, other companies) are
  structural (the side list is the leader's own companions) and were
  not given separate tests.
- **Verification:** after the review fixes, `make generate`, `make validate`
  and `go test -race ./...` passed (94 packages ok); `make js-lint` passed
  on the final `window-combat.js`; the Playwright dock check
  (`scripts/browser/dock-windows-check.mjs`, system Chromium, localhost)
  passed with the new ability checks. The new integration tests passed 20
  repeated runs. No Lua changed.

### 33d owner review follow-up (2026-10-01)

- **Owner review of 33d** found, and the owner approved fixing, with a
  regression test each (all fail on the old code):
  - A last enemy killed by a round-tick or immediate buff (poison) paid
    no XP: its queued death ran after the same round's battle pass ended
    the battle. Contributors are now frozen at those death sites too
    (not for revive-on-death mobs).
  - 33d dropped `get`'s CorpseItems gate, so in the default world anyone
    looted the gear a body keeps (failed drop rolls, perma-gear). Only a
    claimed corpse is a loot source there, never its worn gear; an
    unclaimed corpse name no longer shadows a floor item.
  - Claimed loot could be buried by anyone or any mob, and never decayed
    with corpses disabled. `bury` now refuses another player's claimed
    loot; claimed corpses decay regardless. `RemoveCorpse` no longer
    confuses look-alike corpses (one group, one round) and deleted a
    claimed twin. `get`, `look` and `bury` prefer the right same-named corpse.
  - Owner decision: outside one alliance the loot claim goes to the most
    damage (companions count; ties to the lowest id); allies keep the
    rotating claim.
  - `party leave` hands leadership to an online member when there is one;
    messages name players instead of `player #id`; `alliances.json` is
    gitignored. `help party` and `help bury` updated and render-tested.
  - The intermittent `TestBystanderCannotHelpAnotherPlayersBattle` was a
    test bug (its enemy patient could die in the opening round); fixed.
- **Independent review** of the fix diff: accepted a get shadowing bug,
  first-corpse-only `bury`/`look`, and an unrendered `bury` help test; all
  fixed with tests (it also exposed the `RemoveCorpse` twin bug). Rejected:
  `look` reading an offline claimant's saved name from disk, accepted as a
  rare, player-driven read during a claim's ~2½ minutes.
- **Owner decisions for a future cargo phase (before 33g):** no personal
  inventory besides worn equipment; everything carried, companions' carried
  items included, lives in company cargo, and spoils go there (`loot`,
  optional `autoloot`, reserved to the winners for 2 game hours, open for
  2 more, then gone). No item may be used during a fight.
- Verification: `make generate`, `make validate` and `go test -race ./...`
  passed after the review fixes; focused package runs passed throughout.
  No JavaScript or Lua changed, so their lint was not run.

### 33c owner review follow-up (2026-10-01)

- **Owner decision (`phase-33c-one-retreat`):** one way out of a fight.
  Review found emergency `flee` beat ordered retreat: it rolled only against
  foes striking the leader (a back-row leader always escaped), and a pinned
  companion was "separated" and rejoined the same round, because the flight
  itself ended the battle. The owner chose: `flee` is another name for
  `retreat [exit]` (wimpy and the web Battle view's button, now "Retreat",
  issue the same order); no separation on withdrawal (a pinned leader or
  member holds the company and is named); every active battle foe pursues;
  the preparation round stays. 30e morale flight remains the only separation.
  The emergency-flee path and `flee.md` are gone (`help flee` is `help
  retreat`); retreat also leaves a fight with another player.
- **Bug found while testing, fixed:** a retreat order carried the leader's old
  target, so that target dying in the preparation round (mob death clears
  every aim at it) silently cancelled the withdrawal; 33c had shipped with it.
  The aim to resume now lives in the runtime `RetreatInfo`.
- **Independent review:** no high findings. Accepted and fixed: wimpy
  re-ordered (and repeated refusals) on every hit, now once a round and not
  while withdrawing; missing wimpy and player-fight round tests (added); "held
  by magic" in `help retreat` (no such status) and missing "players pursue";
  `set wimpy` threshold wording ("below"); a dying leader told their legs were
  pinned; dead `holdsAgainstPlayer` check; a defender kept an empty aim after
  a failed attempt; stale "flee" in `help travel`. Kept: the leftover player
  `Flee` aggro branch, as a harmless guard (nothing sets it).
- **Help:** `retreat`, `combat`, `targeting`, `break`, `statuses`,
  `webclient`, `set-wimpy`, `travel`, `company`, `guardian`, `morale` (retreat
  notes above "See also"), aliases, and the tutorial hint.
- **Checks:** new tests (flee is the order and every battle foe pursues;
  pinned member holds the company with no loyalty loss or separation; wimpy
  once; player fight; target falling mid-order) fail on the old code and pass.
  `make generate`, `make validate`, `go test -race ./...`, `make js-lint` and
  the browser dock check passed. One intermittent failure of
  `TestSecondPlayerTakesTheNextGroup` (no retreat involved) appeared once in
  an early package run and did not recur in 39 later runs, isolated or whole
  package, on this branch or master; noted, not explained. After merging
  33d into the branch, the full race suite was rerun: everything passed except
  upstream `TestAttemptConversation_UsesPluginFile`, which fails about 2% of
  runs (4/200 measured): `getConversation`'s random 2% maintenance purges
  conversations older than 10 rounds, untouched by 33c or 33d; it passed on
  rerun. Left for an upstream-code fix.

### Phase 33d: allied companies completed (2026-10-01)

- Pulled master before implementation (`4b4db05e`); re-fetched and integrated
  latest `854fea80` (33a script authority and 33b battle-boundary fixes)
  before final checks.
- Durable version-1 alliance membership/leadership; atomic replacement and
  full registry rollback on failure; missing-file migration and corrupt startup
  refusal. Logout retains membership, resets consent, promotes online successors;
  character purge retries rather than proceeding after a failed alliance save.
- Independent company grids, commands, focus, assets and retreat. Accepted
  membership grants no autoattack/follow/support; queued orders revalidate tokens.
  Leadership changes revoke follow/autoattack. Recovery restores no consent.
- One enemy XP pool shared only by actual, eligible contributing companies;
  living attached companions receive their owner's share. Death-time battle
  snapshot survives final-enemy battle closure; payout rechecks presence/life.
  Shared ordinary loot/gold has one fixed corpse claimant until corpse decay.
  Legacy ranks are display only. One primary evaluates shared enemy morale and
  owns mercy; existing durable mercy recovery and runtime enemy-life guard remain.
- Full login GMCP refresh includes ownership/consent/offline members; browser
  distinguishes alliance leadership and company command. Indexed party/help
  aliases, protection correction, combat/company/support hubs and tutorial pointer.
- Independent full-phase reviewer found final-enemy payout loss, incomplete
  solo-leave rollback, login structural refresh gap, claimed-corpse inspection
  and missing integration coverage. All fixed with regressions and re-reviewed;
  no remaining blockers. Final-enemy regression narrowed to its own XP/claim.
- Tests cover two actual companies, shared battle/formation/focus/retreat and
  morale-owner departure, separate/idle/distant/fallen/withdrawn reward exclusion,
  duplicate death/loot pickup, delayed World autoattack in both queue branches,
  recovery/save rollback and purge retry, and GMCP recovery payloads.
- Ordinary battles/enemies/corpses remain runtime-only; no replay journal or
  atomic cross-character payout transaction is introduced. Existing character
  and company saves and durable mercy tokens retain their established seams.
- Verification: `make generate`, `make validate`, final integrated
  `go test -race ./...`, JavaScript and Lua lint, Chromium dock-window harness
  (including alliance authority/offline status), and `git diff --check` passed.
  Focused checks passed before the final suite. Earlier fixture checks exposed
  XP-disabled combat setup and an overly broad final-enemy assertion; corrected
  both before the final integrated run. No remaining failed checks.
- The final 33b merge requires mutual consent and overlapping live enemies
  to broaden battle participation for allied support; separate battles and idle
  allies remain blocked. Real cast-start/battle-entry/completion regression covers
  both ally owners and companions, including revoked consent. Manual battle casts
  remain prohibited; the fixture starts its chant before entering battle.
- Full-phase and both master-integration independent re-reviews found no remaining
  blockers. Owner explicitly authorized merge to master and push when confident.

### Phase 33d: initial alliance consent slice (historical) (2026-10-01)

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
- **At the end of this initial slice, 33d was incomplete and unmerged.** Party state remained
  runtime-only. Encounter participation/XP, deterministic loot, durable alliance
  recovery and rank replacement remain the next tasks in the plan.
### 33b owner review follow-up (2026-10-01)

- **Owner review of 33b:** the owner kept one 33b choice. Minor Heal All
  reaches only the caster's own company, not GoMud party members; 33d brings
  cross-company help back with consent.
- **Battle boundary (`phase-33b-review`):** a bystander could heal or aid
  either side of another player's battle (e.g. keep that player's bandits
  alive). `effecttargets.OtherBattle` now refuses a named patient fighting in a
  battle the caster isn't part of (its foes, its player, and the companions or
  charmed pets fighting for that player, as the battle counts them). `cast`
  and `aid` refuse at start; `Resolve` drops such a patient at completion.
  Group and area help stay narrowed to the caster's company, not refused.
- **Silent chants:** a helpful chant that lost every patient ended in silence.
  It now ends with "Your spell (or aid) finds no one left to help" and a
  `CastComplete` with the `wasted` outcome; the mana stays spent.
- **Help:** 33b paragraphs moved above the see-also lines of `cast`, `spells`
  and `heal`; `friendly-effects`, `protection` and the tutorial hint describe
  the boundary and the fade.
- **Independent review:** no high findings. Accepted and fixed: area help was
  refused beside another battle (pre-check now only for named patients);
  charmed pets ignored (now counted like the battle does); per-round battle
  cloning (cheap `battle.Involving`/`battle.RoomOf`); "spell" wording for
  first aid; refused aid counted as a skill use; weak tests; a ragged help
  line. Recorded, not changed: once 33d installs allied consent,
  `OtherBattle` would drop an ally fighting the same group in their own
  battle; 33d must count a caster sharing that enemy group as part of it.
- **Checks:** the new tests cover each entry point (`cast`, `aid`, chant
  completion, area help, pets); the probes that found the bugs ran against the
  old code. `make generate`, `make validate`, `go test -race ./...` and
  `git diff --check` passed. No JS or Lua changed, so no lint was needed.

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

- **Other players on the map (owner, 2026-10-05):** not shown for now,
  apart from your own party members. Revisit later, alongside camp
  visibility.
- **Race and gender sprite variants (owner, 2026-10-05):** not for now.
  Revisit with the races review.
- **Camp theft (owner, 2026-10-05), built in 40a4:** a camp event. A company
  resting **without camp bells and trip lines** may wake to find some loot
  and supplies missing, with no fight and no warning. Details are in the
  [40a3 camp gear](designs/2026-10-05-phase-40a3-camp-gear-design.md)
  design.
- **PvP camp visibility (owner, 2026-10-05):** the visual map shows only
  your own camp and allied camps. Revisit for PvP: visibility should depend
  on lighting, terrain, and the company's skill at concealing its camp.
  See the [visual client milestone](designs/2026-10-05-visual-client-milestone-design.md).
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

- **Plugin persistence uses atomic replacement** (`WriteStruct`/`WriteBytes`
  through `util.SafeSave`). Cross-file operations still need recovery markers.
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
- [Visual client milestone](designs/2026-10-05-visual-client-milestone-design.md) and [sprite specification](designs/2026-10-05-sprite-specification.md) — Phase 40 map and battle screen.
- `docs/designs/` — active proposals and useful shipped design records.
- `docs/plans/` — execution guidance and work still requiring follow-up.
- Nested `AGENTS.md` files — package-specific constraints.
- Git history — retired plans/proposals and detailed completed work logs.
  Use `git show 792455ea:<old-path>` for the snapshot before this cleanup.
