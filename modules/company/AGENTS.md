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

- A recruiter is a room in the `Recruiters` config (`RoomId`, `Name`, `Candidates: [{Id, MobTemplateId, Price, Tutorial}]`). `company recruit` there lists the candidates (archetype, level, alignment, gear, price); `company recruit <candidate>` takes one on. Shipped: the Waymark Inn (2003) and Trappers' Post (2005), templates 61–64.
- Recruiting goes through `enlist`, the same path as `company summon` (survival identity, archetype, disposition, template gear, one company save, full rollback). The candidate list is its own allow list: never add a candidate template to `AllowedCompanionMobIDs`, or `summon` would skip the price and the claim (`TestShippedRecruitersResolve` pins this).
- A `Tutorial` candidate is free and claimable once per account (the record is keyed by user id, so alts share claims). The claim is the template id in `Record.Claimed`, written in the same save as the recruit and rolled back with it; it outlives dismissal (`Put` keeps a claims-only record).
- Gold is checked first and taken only after the company save succeeds, then the user is saved at once (`saveUser`). A failed user save leaves the deduction in memory for the next autosave.
- Candidate templates wear their gear, carry nothing, have no gold, and use `itemdropchance: 0`, so no recruit can be farmed for items or gold. Keep new candidates to that shape.
- `wiring_recruit_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard. Unit tests use template ids no fixture world defines, because the wiring tests load mob and item specs globally.

Phase 24 company chemistry (`chemistry.go`):

- Each record has `Bonds`, one per unordered pair of member keys (`a` < `b`), with the shared `rounds` and the `last_round` charged. `Registry.Put` prunes bonds of members no longer in the record, so dismissal and desertion end them in the same save; a companion's death only pauses its bonds, and the respawned companion (same ID) resumes them.
- `onNewRound` calls `accrueChemistry(evt.RoundNumber)` before the drift countdown, so a drift rollback snapshot includes it. A pair accrues when the leader is signed in and both members are alive (`Health >= 1`) and present (the leader online; a companion's tracked, live, attached mob) in the same room. A round is charged once (`last_round`); a counter more than 900 rounds behind a bond's last round counts as a reset. It only reads the round number.
- Bonds ride on every company save. A bond reaching a new tier saves at once; a failed save holds it one round short of the tier, so no tier is used or shown before it's on disk. The leader is told only after the save.
- The module implements `company.ChemistryProvider`; `internal/combat` adds `ChemistryBonusForUser`/`ChemistryBonusForInstance` to the hit modifier in all four `Attack*` functions. `ChemistryHitBonus` reads the registry map in place (no copying), since combat calls it on every strike. `company chemistry` and `status bonuses` show it; the browser Company panel is the information-surfaces phase's.
- The chemistry world seam (`chemistryWorld`) is separate from the alignment one, so the alignment test fakes don't change. Knobs are in the config overlay. They're parsed into `chemRules` on load and once a round (`refreshChemistryRules`), never per strike: `plug.Config.Get` flattens the whole modules config on every call.
- `wiring_chemistry_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard.
