# Company Module Guide

`MobTemplateID` and companion `ID` are durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores every companion for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

The company roster is one leader plus up to `MaxCompanions` (default 4) companions, for a five-member cap. Durable records are written immediately when commands change company state.

Player-facing management: `company summon <mob-id-or-name>`, `company status`, `company dismiss <member|all>`, and `formation` / `formation move <member> <row> <col>` / `formation swap <a> <b>` / `formation clear <member>`. Formation rows and columns are 1-based to players; row 1 is the front row.

Legacy single-`companion` records migrate to `companions[0]` with ID 1 on load. Formation cells for dismissed companions are pruned automatically. Travel, rest, and formation must never change global game time.

Phase 21a alignment: each companion record carries a `disposition` (engine alignment −100..100 and loyalty 0..100), seeded from the mob template (race default when 0) on summon and, for older records, on load at full loyalty. Players see alignment as 1–100 (`company status`, `company alignment`, `company inspect <mob>`). An `events.NewRound` listener counts the persisted `drift_in` down and, every `DriftEveryRounds`, drifts each online leader's companions toward the rest of the company, adjusts loyalty, and deserts companions at 0 loyalty through the same path as dismissal (never while the leader or that companion is fighting). The leader never drifts. `company summon` refuses candidates more than `RecruitMaxGap` from the company average. The live mob's `Character.Alignment` is set from the record on every spawn and after each drift. Knobs are in `files/data-overlays/config.yaml`, in engine points. `wiring_test.go` calls `plugins.Load`; it restores the plugin package state afterwards with `plugins.SnapshotLoadStateForTest`, so later tests can still call `plugins.New`.

Phase 21b: the module also implements `company.AlignmentProvider` (`CompanyAlignment`, the same company average the recruit gate uses) for `modules/standing`. It reports nothing while company data is unavailable.

Phase 22b durable level and gear (`state.go`):

- Each companion record has a `State` (level, experience, `characters.Worn`, carried items). It is the source of truth. `Runtime.Spawn` rebuilds the live mob from it at the saved level, replacing the template's minted gear with copies of the saved gear.
- **New recruits:** `company summon` snapshots the freshly spawned mob into the record before the summon's save.
- **Legacy companions** (`State == nil`): `ensureState` derives the state from the template spec and saves it before spawning. A failed save leaves the companion awaiting restoration and retries on the next spawn.
- **Snapshot seams**, all on the game loop:
  - `ItemOwnership` on a companion instance: refresh and save;
  - plugin `OnSave` (autosave, shutdown, copyover): refresh every live companion;
  - the leader's `PlayerDespawn`: refresh, save, then remove the live mobs so a reverted companion can't be looted for gear its record would restore;
  - the companion's `MobDeath`: gear cleared, level kept.
- Dismissal and desertion drop the state; the companion leaves with its gear. Don't add a path that drops a companion's gear into the world on dismissal, since summon-and-dismiss would then farm template gear.
- `company gear <member>` shows the recorded gear, refreshed from the live mob when it is out.
- `wiring_state_test.go` also calls `plugins.Load`, with the same `SnapshotLoadStateForTest` guard.
