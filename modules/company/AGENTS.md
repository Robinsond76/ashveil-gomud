# Company Module Guide

`MobTemplateID` and companion `ID` are durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores every companion for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

The company roster is one leader plus up to `MaxCompanions` (default 4) companions, for a five-member cap. Durable records are written immediately when commands change company state.

Player-facing management: `company recruit [candidate]` (Phase 22c), `company summon <mob-id-or-name>`, `company status`, `company dismiss <member|all>`, and `formation` / `formation move <member> <row> <col>` / `formation swap <a> <b>` / `formation clear <member>`. Formation rows and columns are 1-based to players; row 1 is the front row. `formation reach <member>` (Phase 29a) answers against the enemy party the member is fighting, with its own reach (`internal/enemyparty`, `combat.ResolveReach`); out of a fight it is a labelled plain-melee demonstration within the company. `wiring_combat_test.go` is the 5v5 combat wiring world (shipped config, real round).

Legacy single-`companion` records migrate to `companions[0]` with ID 1 on load. Formation cells for dismissed companions are pruned automatically. Travel, rest, and formation must never change global game time.

Phase 21a alignment: each companion record carries a `disposition` (engine alignment −100..100 and loyalty 0..100), seeded from the mob template (race default when 0) on summon and, for older records, on load at full loyalty. Players see the same −100..100 scale (`company status`, `company alignment`, `company inspect <mob>`). An `events.NewRound` listener counts the persisted `drift_in` down and, every `DriftEveryRounds`, drifts each online leader's companions toward the rest of the company, adjusts loyalty, and deserts companions at 0 loyalty through the same path as dismissal (never while the leader or that companion is fighting). The leader never drifts. `company summon` refuses candidates more than `RecruitMaxGap` from the company average. The live mob's `Character.Alignment` is set from the record on every spawn and after each drift. Knobs are in `files/data-overlays/config.yaml`, in alignment points. `wiring_test.go` calls `plugins.Load`; it restores the plugin package state afterwards with `plugins.SnapshotLoadStateForTest`, so later tests can still call `plugins.New`.

Phase 21b: the module also implements `company.AlignmentProvider` (`CompanyAlignment`, the same company average the recruit gate uses) for `modules/standing`. It reports nothing while company data is unavailable.

Phase 22b durable level and gear (`state.go`):

- Each companion record has a `State` (level, experience, `characters.Worn`, carried items). It is the source of truth. `Runtime.Spawn` rebuilds the live mob from it at the saved level, replacing the template's minted gear with copies of the saved gear.
- **New recruits:** `company summon` snapshots the freshly spawned mob into the record before the summon's save.
- **Legacy companions** (`State == nil`): `ensureState` derives the state from the template spec and saves it before spawning. A failed save leaves the companion awaiting restoration and retries on the next spawn.
- **Snapshot seams**, all under the world lock (shutdown's final save and the SIGUSR1 copyover now take `util.LockMud()`):
  - `ItemOwnership` on a companion instance: refresh **in memory only**. Don't add a save here: the company file must change only with the user and room files (autosave, copyover, shutdown, logout), or a crash duplicates items;
  - plugin `OnSave` (autosave, shutdown, copyover): refresh every live companion;
  - the leader's `PlayerDespawn`: refresh, save, then remove the live mobs so a reverted companion can't be looted for gear its record would restore;
  - the companion's `MobDeath`: gear and gold cleared, level kept.
- A tracked mob now charmed by another player (`CharmedByOther`) is lost: it is untracked and never destroyed, and its record's gear is cleared. An uncharmed one is still the company's.
- Companions spawn with `mobs.NewMobByIdNoElite`, so they never roll elite.
- Dismissal and desertion drop the state; the companion leaves with its gear. Don't add a path that drops a companion's gear into the world on dismissal, since summon-and-dismiss would then farm template gear.
- `company gear <member>` shows the recorded gear, refreshed from the live mob when it is out.
- Phase 23b whetstone edges (`sharpbonus`/`sharpstrikes` on `items.Item`) live on the weapon instance, so they ride on these same snapshots with no extra code. `modules/camping` sharpens the live mob's weapons and combat spends them on the live mob; both are captured at the next snapshot seam, like any other gear change.
- `wiring_state_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard.

Phase 22c recruiters (`recruit.go`):

- A recruiter is a room in the `Recruiters` config (`RoomId`, `Name`, `Candidates: [{Id, MobTemplateId, Price, Tutorial}]`). `company recruit` there lists the candidates (archetype, level, alignment, gear, price); `company recruit <candidate>` takes one on. Shipped: the Waymark Inn (2003), Trappers' Post (2005), and the tutorial's Muster Yard (901, Phase 27a), templates 61–64. A recruiter room is matched by its template room (`rooms.GetOriginalRoom`), so each player's ephemeral copy of 901 offers the same candidates.
- Recruiting goes through `enlist`, the same path as `company summon` (survival identity, archetype, disposition, template gear, one company save, full rollback). The candidate list is its own allow list: never add a candidate template to `AllowedCompanionMobIDs`, or `summon` would skip the price and the claim (`TestShippedRecruitersResolve` pins this).
- A `Tutorial` candidate is free and claimable once per account (the record is keyed by user id, so alts share claims). The claim is the template id in `Record.Claimed`, written in the same save as the recruit and rolled back with it; it outlives dismissal (`Put` keeps a claims-only record).
- Gold is checked first and taken only after the company save succeeds, then the user is saved at once (`saveUser`). A failed user save leaves the deduction in memory for the next autosave.
- Candidate templates wear their gear, carry nothing, have no gold, and use `itemdropchance: 0`, so no recruit can be farmed for items or gold. Keep new candidates to that shape.
- `wiring_recruit_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard. Unit tests use template ids no fixture world defines, because the wiring tests load mob and item specs globally.

Phase 24 company chemistry (`chemistry.go`), company-wide with dilution (owner amendment; the pair-bond model was replaced before merge):

- Each record has `Service`, one per member: `rounds` served with the band and the `last_round` charged. `Registry.Put` prunes entries of members no longer in the record, so dismissal and desertion end them in the same save; a companion's death only pauses its service, and the respawned companion (same ID) resumes it.
- `onNewRound` calls `accrueChemistry(evt.RoundNumber)` before the drift countdown, so a drift rollback snapshot includes it. A member serves when the leader is signed in and it is alive (`Health >= 1`), present (the leader online; a companion's tracked, live, attached mob), and in one room with another present member. A round is charged once (`last_round`); a counter more than 900 rounds behind counts as a reset. It only reads the round number.
- A band is the present members in one room (two or more). Its tier comes from the average of their **saved** service (`Service.Saved`, in memory only, `yaml:"-"`), each member capped at the Sworn threshold so a veteran can't hide a recruit. `save()` and `load()` call `markServiceSaved`, so tiers never use rounds that aren't on disk. Chemistry never saves on its own (the company file also carries gear snapshots, which must reach disk only on the existing seams): accrual notes pending tier-ups, and `save()` announces them after it succeeds. Keep every company save going through `m.save()`, or tiers will lag behind what's on disk.
- The module implements `company.ChemistryProvider`; `internal/combat` adds `ChemistryBonusForUser`/`ChemistryBonusForInstance` to the hit modifier in all four `Attack*` functions. `ChemistryHitBonus` reads the registry map in place (no copying), since combat calls it on every strike. `company chemistry` (the band with the leader, bands apart, each member's saved service) and `status bonuses` show it; the browser Company panel is the information-surfaces phase's.
- The chemistry world seam (`chemistryWorld`) is separate from the alignment one, so the alignment test fakes don't change. Knobs are in the config overlay. They're parsed into `chemRules` on load and once a round (`refreshChemistryRules`), never per strike: `plug.Config.Get` flattens the whole modules config on every call.
- `wiring_chemistry_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard.

Phase 25a (`relocate.go`): the module implements `company.RelocationProvider`. `RelocateCompany(leader, origin, room)` (Phase 33h3 added `origin`) moves every tracked companion whose mob is live, attached, and alive and stands in `origin` or `room` into `room`, clearing its aggro (`Runtime.Relocate`). It moves live mobs only: the record, formation, gear, alignment, and service are unchanged, and a crash needs no recovery because companions respawn with the leader on login. A companion that died has no live mob and is untouched.

Phase 25b companion death (`death.go`, `resurrect.go`):

- A companion's death marks its record dead (`Companion.Death`: operation ID, allowance, remaining seconds, the formation cell it held) in one save. Its `State` keeps its level and only what the body kept: `events.MobDeath.KeptWorn`/`KeptItems`/`KeptGold`, which `mobcommands.Suicide` fills from the same drop rolls it uses. Recruit templates use `itemdropchance: 0`, so their worn gear comes back.
- **Dead means absent.** `restoreForLeader` never spawns the dead. `Roster` marks them `Dead`, and survival exertion and rest, walking and exposure drains, camp tiers, and the inn price skip them; provisioning refuses them. Drift, desertion, and the company average skip them. They still hold their roster slot, and `formation move/swap` refuses them (`ErrMemberDead`).
- **The allowance** (`ResurrectionAllowanceDays`, default 3 game days = RoundsPerDay × RoundSeconds each) is charged once a round from `onNewRound` (`chargeAllowances`), only for online leaders, from an in-memory anchor set at `PlayerSpawn` and dropped at `PlayerDespawn` (after a final charge and save). Each charge is capped at 60 seconds, so a stalled loop isn't spent. Remaining time reaches disk with every company save, so a crash refunds at most the time since the last save and can never expire a companion early. Don't persist the anchor: that would charge offline time.
- **Expiry** at zero goes through `dropCompanion` (the dismissal path with rollback) and adds the companion to `Record.Lost` (ten most recent) in the same save. `Put` keeps a record that has only `Lost`. Like desertion, this saves from `onNewRound`, so it also writes other leaders' in-memory gear snapshots; that is the same accepted risk desertion carries, not a new seam to copy.
- A leader's `PlayerDespawn` keeps a companion whose mob already died tracked, so a `MobDeath` queued behind the logout still records the death; `restoreForLeader` clears the stale entry at login.
- "Online" is `users.GetByUserId`, so a link-dead leader is charged until their despawn (at most `LinkDeadSeconds`).
- **Resurrection** is `company.ResurrectionProvider` (`DeadCompanions`, `ResurrectCompanion`), called by `modules/death`'s `resurrect` command. It charges first (so time that ran out is lost, not raised), takes one level (floor 1, experience 0), revives into the old cell if free, saves, and only then spawns; a failed save changes nothing; a failed spawn leaves the companion awaiting restoration. The dead are matched by name before the living.
- `wiring_resurrect_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard, and imports `modules/death` for the command.

Phase 26a (`members.go`): the module implements `company.MemberViewProvider` (`CompanyMembers`): each companion present (with live health via `Runtime.Vitals`), awaiting restoration, or dead (with rescue time), with its cell. `internal/companyview` reads it on the game loop for `status`, the prompt, and (26b) the browser.

## Phase 28: companions' gear in the company load

- `CompanionGearGrams` (the `company.GearProvider` seam) weighs every living companion's worn and carried gear. A companion out in the world is weighed in place (`runtime.GearGrams`, no copy); one charmed away by another player counts nothing. Otherwise it uses the record's `State`, or the template's gear before there is one. A fallen companion's gear stays with the body and is left out, and an unreadable company weighs nothing. Items are weighed with `Item.Weight()` (base data). `modules/encumbrance` adds it to the load as `CompanionGrams`. Call it on the game loop.

## Phase 32a2: per-player recruit rosters (`roster.go`, `internal/company/roster.go`)

- A recruiter with `Generated: true` (the Waymark Inn, the Trappers' Post) also posts each leader their own `RosterSize` generated candidates, stored on the leader's record (`Record.Rosters`, one per recruiter room, kept by `Put` even without companions). The tutorial's recruiters are authored only.
- **Lazy, read-only clock:** `rosterFor` refreshes a roster when it is read (the notice, `company recruit`, `company inspect`, `look [name]`) from `util.GetRoundCount()` through `m.round()`; nothing ticks and the round is never advanced. `domain.RefreshRoster` is pure; a newcomer arrives when its slot emptied, or part way through a stay if that stay would already be over, so a long absence is one step per slot.
- **Refreshes never save.** The company file also carries in-memory gear snapshots (22b), so a refresh stays in memory and reaches disk with the next company save. A crash only rerolls unsaved faces. A hire saves at once through `enlist` (`generatedHire`), with the shorter roster in the same save; `rollbackSummon` restores the pre-hire rosters and claims.
- Candidates are matched: roster exact (given name or full name), then the regulars (`matchCandidate`), then a unique partial roster name. Given names are unique across the leader's company, all their rosters, and the recruiter's regulars.
- **Companions' own names:** `Companion.Name`/`Description` (blank for authored companions). Always name a companion with `nameOf(c, fallback)`, never `templateName(c.MobTemplateID, …)`. `Runtime.Spawn` takes the `domain.Identity`, and the live mob wears it on every spawn, restore, and resurrection, so combat text and the room follow. Several companions may share a template (generated recruits use 80–84): look companions up by ID or name, never by template.
- The module's `math/rand` source (`m.rng`) is game-loop only, like the rest of the module.
- `wiring_roster_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard; it sets the real round with `util.SetRoundCount` and restores it.

## Phase 32f: company logistics (`members.go`, `inventory.go`, `provision.go`)

- `CompanionCarry` (`company.CarryProvider`) is each companion `CompanionGearGrams` weighs, with its Strength (live only) and its largest pack; `modules/encumbrance` turns it into capacity, and `company.CountedMembers` (1 + its length) caps the herd in `modules/mount`.
- `CompanionsWithLeader` (`company.PresenceProvider`, 32f review) is the living companions out, still the company's, and in the leader's room (`Runtime.WithLeader`). The riding pace (`company.WalkingMembers`) and meals count only these.
- `company inventory` (`inventory.go`) lists the load, every member's worn and carried items (live gear when out, as `company gear`), the herd, and the cargo. Name companions with `nameOf`.
- `company eat`/`drink`/`meal` (`provision.go`): `planMeal` is pure; `mealView` feeds the leader and the companions present, from the cargo, then the member's own pack, then the leader's. Each step spends the item first and only then provisions (so food that can't be spent feeds no one; a crash in between loses one use). Only meal buffs (17, 18, 34) qualify.
- `useCompanionItem` takes a use from a live companion (then refreshes its record in memory, as a gear change does) or from its record (saved at once, rolled back on a failed save; a companion not out has no other seam).

## Phase 30b: wounds (`wounds.go`, `runtime.go`, `death.go`)

- A companion's wounds live on its live mob (`Character.Wounds`) and in `MemberState.Wounds`, captured by `Snapshot` at the 22b seams like gear (in memory; written with the next save). `applyState` restores the lasting ones and spawns the mob at its wound limit, not max. A companion's death drops them (`keptState`).
- `MemberView.HPLimit` (via `Runtime.HealthLimit`) feeds `internal/companyview` and GMCP.
- `heal` lists the hurt; `heal wounds` (refused while the leader is in a battle or anyone present has aggro) plans with `wounds.Plan`: clerics first (a player needs `cast` and the spell; a companion its archetype's spells at its level), then items spent before they are applied (cargo, the patient's pack, the leader's, the other companions'), then a configured physician (`Physicians` in the config overlay, matched by template room) asked through the user prompt; `yes` re-checks the price, takes the gold, clears every present member's wounds, saves the company and the user. Instant; never touches the clock.
- `wiring_heal_wounds_test.go`, `wiring_wounds_test.go`, and `wiring_wound_spells_test.go` run in the brawl world.

## Phase 30c1: company tactics (`tactics.go`)

- `company tactics` (and the `tactics` shorthand) shows and sets the company focus and healing threshold, stored by `modules/strategy` through `strategy.SaveTactics` (never on the company record, so a solo player has them too). Out of a battle every change saves; in one only `focus` is open, as a battle-only order (`battle.SetFocus`/`ClearFocus`, one a round, "still turning" until the next upkeep applies it). The order's line names the focus's choice among every foe standing.
- `wiring_tactics_test.go` covers the focus, the threshold, the mid-battle order, and enemy personalities in the brawl world. `newBrawl` turns off personality noise (`hooks.UseAimRollForTest`).

## Phase 30c2: guardians (`formation.go`, `internal/hooks/combat_guard.go`)

- A guardian (a `strategy` role, `modules/strategy`) takes one blow aimed at its ward, decided in `internal/hooks` at the enemy-attack gates after 11c's interception; guard counts live on the battle (`internal/battle`). The `formation` view and every `move`/`swap`/`clear` warn (`guardWarnings`) when a guardian's set ward stands more than one column away, as `strategy` does.
- `wiring_guardian_test.go` runs guards in the brawl world: `guardBrawl` opens the battle with the bandits holding their first blows, so the guards start full, and `strike` aims one blow a round at a ward.

## Phase 30d1: broken chants and shield counters (`internal/hooks/combat_interrupt.go`)

- After every weapon blow's lines, `afterBlow` breaks the target's chant on a hit that did damage (a player or companion: the spell ends with half its mana back; an enemy: it restarts at its next turn) and lets a shield-bearer counter a missed melee blow. Rules live in `internal/interrupt`.
- `newBrawl` turns counters off (`hooks.UseCounterRollForTest`) so the brawls keep their seeded rolls; a test that wants a counter scripts its dice (`counterDice`). `forceBlows` makes every blow land or miss. `TestNarrationPreservesCombatOutcome` compares against a pre-30d1 golden and so turns 30d1 off (`hooks.DisableInterruptsForTest`).
- `wiring_interrupts_test.go` covers broken chants, restarts, guarded blows, counters, and the shipped goblin hexer.

## Phase 33h1: growth and contracts (`growth.go`, `runtime.go`)

- A companion's training is **derived, never saved**: the template's training plus `characters.StatPointsAtLevel(level)` dealt by `domain.Deal` over its archetype's `Growth` weights (`archetypes.CompanionGrowth`) and its `GrowthFocus` (+2). `Runtime.Spawn` takes the weights and retrains before vitals are set; `Runtime.Retrain` re-deals a live mob and only clamps health and mana. Every level change must re-derive (spawn, `RetrainCompanion` after a live level-up in `mobcommands.AwardCompanyXP`, `company archetype`, `company growth`); never call `AutoTrain` on a tracked companion, or the stats change at the next respawn.
- `company growth [member stat|balanced]` sets the focus in memory only (it reaches disk at the next 22b seam; never add a save here), then retrains; refused in a battle. The stat is the last word, so member names may have spaces.
- Contracts: `rewards.companyexperience` on a quest; the quest hook pays through `mobcommands.AwardCompanyXP` (the 32e presence rule). `wiring_growth_test.go` covers both in the brawl world.

## Phase 33h2: readiness and recovery (`runtime.go`, `internal/company/state.go`)

- `MemberState.Vitals` (health and mana) rides the 22b snapshot seams with gear and wounds (in memory; written with the next save, never on its own). `applyState` resolves it with `Vitals.Resolve` after the wounds and stats are set: saved points clamped to the wound limit and mana maximum, never rescaled, never below 1 health. Nil (a pre-33h2 record) spawns full once; `Percent` (a resurrection, `resurrectVitalsPct`) resolves at the spawn, so a crash first can't refill it.
- Death clears the vitals (`keptState`); resurrection sets the half share. Morale flight's `ReturnHP`/`ReturnMana` still override after its respawn.
- Companions recover online only: `internal/hooks` `regenCompanionVitals` (the players' every-third-round beat, out of a battle, leader online, health to the wound limit). No offline or elapsed-time recovery. `mobcommands.AwardCompanyXP` keeps a companion's vitals across a live level-up (GoMud's `LevelUp` refills them).
- An inn stay's Well Rested grant restores the leader and live companions (`modules/camping` `restoreVitals`). `wiring_readiness_test.go` covers logout, restart, crash, failed save, migration, clamps, level-up, regeneration, and resurrection with the real store.

## Phase 33h3: relocation and separation (`relocate.go`)

- **Every move that isn't an exit calls `company.RelocateCompany(leader, origin, room)` after the leader's move succeeds:** `ScriptActor.MoveRoom` (world scripts), the quest `roomid` reward, journey arrival and its recovery (`modules/expedition`), death (`modules/death`), the tutorial's `travel`/`leave`, and admin `teleport`. A new move path must do the same; never move a companion mob on its own (scripts' `MoveRoom` on a companion is a no-op). Ordinary exits keep native following.
- **Separation** (`separate`): a living companion out in the world but not in `origin`/`room`, or (`sweepStrays`, each round) away from its online leader and out of a fight for `strayRounds` (2) rounds, is snapshotted, marked `Companion.Separation` (reason, `RoundsLeft` = `SeparationRounds`, default 15), **saved**, then detached and untracked, as `BeginFlight` does. A failed save leaves it where it stands. The leader is told through `chemistryWorld().Tell`; messages name the companion without pronouns.
- **Return** (`tickSeparations` in `onNewRound`): counts down only for online leaders, in memory (it rides the next save; a crash only lengthens it). At zero it rejoins once `leaderFreeFor` (alive, no aggro, no battle, no journey, no camp rest) holds: one save clearing every due separation, then `restoreForLeader` spawns them beside the leader. `restoreForLeader` skips the separated, so a relog never brings one back early.
- While separated: `MemberSeparated` with `RejoinSeconds` in `CompanyMembers` (GMCP `separated`), `company status`, survival `MemberRef.Away` (spends and recovers nothing), and no load, carry, presence, meals, chemistry, or experience. The dead and the fled (`PendingReturn`) are never separated.
- `relocate_test.go` (fakes) and `wiring_relocation_test.go` (brawl world: a real room script, the quest hook, logout/restart/login, the stray sweep, the fallen) cover it.

Phase 49 banter (`banter.go`, pool in `internal/banter`): `Companion.Personality` is rolled in `enlist` (`rollPersonality`, set once; legacy companions derive one from their ID). `CampBanter`/`LastBanter` implement `company.BanterProvider` for `modules/camping` (rest start and end) and `modules/gmcp` (`Company.Camp.banter`); `onBattleEnded` works out `battleBanter` before the patch-up and says it after (only for a victory, never when a new battle has begun). Banter state (recent line ids, the latest exchange, known-dead set) is in memory only and dropped on purge. The leader never speaks. `set banter off` is the player option (`banter.OptionKey`). Chances come from the `Banter*Percent` config keys.
