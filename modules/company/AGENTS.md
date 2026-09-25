# Company Module Guide

`MobTemplateID` and companion `ID` are durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores every companion for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

The company roster is one leader plus up to `MaxCompanions` (default 4) companions, for a five-member cap. Durable records are written immediately when commands change company state.

Player-facing management: `company recruit [candidate]` (Phase 22c), `company summon <mob-id-or-name>`, `company status`, `company dismiss <member|all>`, and `formation` / `formation move <member> <row> <col>` / `formation swap <a> <b>` / `formation clear <member>`. Formation rows and columns are 1-based to players; row 1 is the front row.

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

Phase 25a (`relocate.go`): the module implements `company.RelocationProvider`. When the leader dies, `modules/death` calls `RelocateCompany`, which moves every tracked companion whose mob is live, attached, and alive into the church and clears its aggro (`Runtime.Relocate`). It moves live mobs only: the record, formation, gear, alignment, and service are unchanged, and a crash needs no recovery because companions respawn with the leader on login. A companion that died has no live mob and stays behind.

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
