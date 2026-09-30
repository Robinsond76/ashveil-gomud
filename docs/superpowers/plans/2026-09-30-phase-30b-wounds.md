# Phase 30b: Wounds, Treatment, and `heal wounds` — Plan

Design: [phase-30b design](../specs/2026-09-30-phase-30b-wounds-design.md).
Branch: `phase-30b-wounds`, worktree `.worktrees/phase-30b-wounds`.

Each task writes its tests first and runs only its own packages.

- [x] **1. `internal/wounds` (pure).**
  - Tests first:
    - `Limit` (the floor, and at least 1);
    - `FromCrit` points and kind by subtype;
    - `Crushing` threshold;
    - `Bled`;
    - `Close`/`CloseAll`/`CloseLight` (worst first, zero-point wounds
      dropped);
    - `Treat` by item and kind;
    - `Plan` (healer and patient order, tend before heal, mana spent,
      items after healers, bandage-on-damage rule, no negative stock).
  - Add the package and its `AGENTS.md`.
- [x] **2. Characters.**
  - Tests first:
    - `HealthLimit`;
    - `Heal`/`ApplyHealthChange` capped at the limit but never lowering
      health above it;
    - the level-up refill capped;
    - a YAML round trip of `Wounds`.
  - Add `Character.Wounds`, `HealthLimit`, `Wounded`, `AddWound`, and
    `CapHealing`, and wire the cap into those paths.
- [x] **3. Combat resolver (`internal/combat`).**
  - Tests first, through `AttackMobVsPlayer`/`AttackMobVsMob` with a
    forced crit:
    - a player and a companion get a lasting wound, and the line says
      `wounded`;
    - an enemy mob gets none;
    - a crushing non-crit blow leaves a light wound.
  - Add `AttackResult.WoundsToTarget` and apply it in the `Attack*`
    functions.
- [x] **4. Combat loop (`internal/hooks`).**
  - Wiring tests through `DoCombat`/`statusPass`:
    - a bleed that runs out leaves a light wound (a cleared one doesn't);
    - fight end closes light wounds and keeps lasting ones;
    - a stray light wound (no fight) closes;
    - the strategy healer skips an ally who is at their limit.
  - Wire all four.
- [x] **5. Scripting and spells.**
  - Tests first, through real casts:
    - `heal` held back reports `wound limit`;
    - `tend` closes points.
  - Add the `ScriptActor` `GetHealthLimit`/`WoundNote`/`TendWound` methods
    and the TS definitions, update `heal.js` and `healall.js`, and add
    `tend.yaml`/`tend.js`.
  - Add the cleric `GrantSpells`/`CompanionSpells` entries, with a test
    that the archetype config loads with `tend`.
- [x] **6. Companion durability (`modules/company`).**
  - Tests first:
    - a snapshot carries wounds;
    - `applyState` restores them and caps health;
    - a companion's death clears them;
    - a save/load round trip.
  - Add `MemberState.Wounds` and `HPLimit` on `MemberView`.
- [x] **7. `heal` / `heal wounds` (`modules/company/wounds.go`).**
  - Wiring tests through the real command (`plugins.Load` world, with the
    usual `SnapshotLoadStateForTest` guard):
    - refused in a fight;
    - a cleric companion tends and heals, spending mana;
    - splints and bandages are spent from the cargo and packs;
    - the physician prompt: `no` changes nothing; `yes` takes gold and
      closes the wounds; too little gold is refused;
    - `heal` lists without changing anything;
    - the round count is unchanged.
  - Add the `Physicians` config, the Waymark Inn entry, and items 36 and
    37 with their market goods.
- [ ] **8. Rest and death.**
  - Wiring tests:
    - a camp rest's grant closes the leader's and a live companion's
      wounds, with lines;
    - an inn stay's grant does too (`modules/camping`);
    - the church wake clears the wounds (`modules/death`).
- [ ] **9. Surfaces.**
  - Tests:
    - the `status` health line shows the limit;
    - the companyview `HPLimit`;
    - GMCP `hp_limit`.
  - Add the web dock's strip text (checked with a JS test or by
    Playwright if there is one for the strip).
- [ ] **10. Player help and tutorial.**
  - `help wounds`, and a rewrite of `help heal`.
  - Updates to `statuses`, `camp`, `inn`, `death`, `health`, and the
    `combat` hub link.
  - `keywords.yaml` entries and aliases.
  - A Camp lesson pointer to `help wounds`.
  - Render tests, and `TestTutorialHelpPointersExist`.
- [ ] **11. Review and verification.**
  - The independent reviewer goes over `git diff origin/master..HEAD`.
    Verify each finding, and fix the real ones with regression tests.
  - Then, once: `make generate`, `make validate`, `go test -race ./...`.
  - Add a `docs/PROJECT_STATUS.md` entry with its **Review:** line, and
    remove the superseded proposal spec.
