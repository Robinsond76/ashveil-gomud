# Phase 35a — level impact

Implements section 1 and section 4.1 of the owner-approved
[level impact and class power design](../designs/2026-10-05-level-impact-class-power-design.md)
(approved 2026-10-05). Branch and worktree: `phase-35a-level-impact`.
Read the [phase 35 handoff](2026-10-05-phase-35-handoff.md) first.

## Goal

Every level-up changes numbers the player can see:
- automatic stats rise a little each level instead of every fifth level;
- a stat point arrives every 2 levels;
- HP grows at the full class rate through level 20;
- the level-up message lists the real changes and the next milestone.

The balance harness gains zone-band cells, which measure only in this phase.

Out of scope here:
- spell, ability, mana and healing changes (35b);
- companion training (35c);
- talents and promotion (38b).

## Code context (master `0e1d11e`)

- `internal/configs/config.progression.go`:
  - `StatStep` returns `1 + level/StatStepLevels` (integer division);
  - `RacialForLevel(level, base)` builds racial stats from the step;
  - `HealthAtLevel` and `HealthAfterFull` implement HP, with
    `HealthAfterFull = HPAfterFull × perLevel / DefaultHPPerLevel`.
- `_datafiles/config.yaml` `GamePlay.Progression`:
  - `StatStepLevels: 5`, `StatPointsEveryNLevels: 5`;
  - `HPFullLevels: 10`, `HPAfterFull: 1.3`, `DefaultHPPerLevel: 2.5`.
- Archetype HP per level lives in
  `modules/archetype/files/data-overlays/config.yaml`: warrior 3, rogue 2,
  wizard 1.5, cleric 2.5, ranger 2.5.
- `Character.LevelUp()` (`internal/characters/character.go`):
  - awards stat points when `Level % StatPointsEveryNLevels == 0`, only above
    `PeakLevel`;
  - currently refills mana (`c.Mana = c.ManaMax.Value`). Leave that for 35b.
- `characters.StatPointsAtLevel(level)` derives stat points for mob spawns and
  33h1 companion growth. `internal/combat/simulate.go:56` uses the same
  modulo rule.
- The level-up message comes from `UserRecord.GrantXP` →
  `events.LevelUp` → `hooks.SendLevelNotifications` →
  `templates/character/levelup.template`. It shows stat deltas and
  `nextStatStep`.
- Companions level in `internal/mobcommands/companyxp.ashveil.go` (32e) with
  no report to the leader.
- Admin editor: `internal/web/api_v1_progression.go`, with charts sharing
  the config helpers.
- Harness: `modules/company/balance_test.go`:
  - opt-in (`ASHVEIL_BALANCE=1`);
  - `newBalanceFight(t, level, companyMode, enemyMode, enemyLevels...)`
    always fields the 4-member `balanceMirror` group and forces
    `mob.Coordination = 1`.
- Coordination tiers come from `internal/coordination.ForLevel`: tier 1
  below level 10, tier 2 at 10–24, tier 3 at 25–44, tier 4 at 45+.

## Implementation decisions

1. **Smooth stat growth keeps today's values at every fifth level.** Add
   `Progression.SmoothStatGrowth` (`ConfigBool`, shipped `true`). When it is
   on, `RacialForLevel` uses a fractional step `s = 1 + level / StatStepLevels`
   (float division) in the existing formula, truncating as today.
   - Levels 5, 10, 15, … give exactly the current values; the levels between
     interpolate. Levels 1–4 rise slightly above today's values (expected;
     measure it).
   - Keep `StatStepLevels: 5` as the curve's scale. Off, or an interval of 1,
     keeps today's behaviour.
   - Do **not** set `StatStepLevels: 1`, as the design's first draft said:
     `StatStep` returns the raw level for an interval of 1, which would
     roughly quintuple stats. Note this deviation in Project Status.
2. **`NextStatStep` retires from player text.** With smooth growth there's no
   step to wait for. The level-up template and `help progression` show the
   next milestone instead (decision 6).
3. **Stat points every 2 levels.** Set `StatPointsEveryNLevels: 2`. Every
   consumer of the rule must agree: `LevelUp`, `StatPointsAtLevel` and
   `simulate.go`. Factor one helper, `configs.ProgressionConfig.StatPointsAt(level)`,
   and use it everywhere. Enemies and companions derive from it, so they
   scale too.
4. **One-time player stat-point migration.** Characters made before 35a
   received points on the 5-level rhythm. On load, if
   `Character.StatPointRhythm` (new `int`, yaml `statpointrhythm,omitempty`)
   is below 2:
   - grant `StatPointsAt(PeakLevel)` under the new rule minus the same under
     the old 5-level rule (never negative);
   - set the field to 2;
   - save through the user's normal atomic save.
   This runs exactly once, also across copyover and replay copies (32b
   replays start at level 1 and owe nothing). Companions need no migration,
   since 33h1 derives their training.
5. **HP shape.** Set `HPFullLevels: 20` and `HPAfterFull: 1.5`, so
   `HealthAfterFull` gives 60% of each archetype's rate (1.5 / 2.5). Validate
   still defaults `HPFullLevels` to 20. Existing saved vitals clamp and never
   refill (30g4 rule).
6. **Level-up report.**
   - Extend `events.LevelUp` with `HealthMaxBefore/After`,
     `ManaMaxBefore/After`, `StatsBefore`/`StatsAfter` (`ValueAdj` for each of
     the six stats) and `NextMilestone` (level and text).
   - `GrantXP` captures the before-values ahead of the first `LevelUp` call.
   - Milestones come from a small table in a new pure package
     `internal/milestones`: level 3 "second class option", 5/15/25
     "talent", 10 "class promotion", 20 "advanced signature", 30 "elite
     promotion".
   - The texts show what is planned. Each says "(coming)" until its phase
     ships: a boolean per entry, flipped by the phase that delivers it.
   - Rewrite `levelup.template` per the design's example: Health, Mana and
     each changed stat as `old -> new`, training and stat points gained, and
     the next milestone. Keep `stat train` guidance. 35b appends spell lines.
7. **Companion level-up line.** When `companyxp.ashveil.go` levels a
   companion, send the leader one line, for example
   `Brannoc reaches level 7 (Health 70 -> 76, Strength 18 -> 19).`, with the
   same before/after capture. A multi-level gain is reported once.
8. **`experience`.** The experience template
   (`templates/character/experience.template`) adds the next milestone line.
   Companion rows are unchanged.
9. **Admin editor.** Expose `SmoothStatGrowth` and `StatPointsEveryNLevels` in
   the progression editor. Its preview and charts must use the same helpers
   (`RacialForLevel`, `StatPointsAt`), so they can't drift.
10. **Zone-band harness cells.** Refactor `newBalanceFight` to take an
    options struct; keep a thin wrapper so existing callers compile. Options:
    - `EnemyCount` 2–5 (the first N of an extended mirror roster, adding a
      fifth "brute" warrior template for 5);
    - `EnemyLevels []int`;
    - `Coordination` (0 means by level through `coordination.Of`, as live
      groups do);
    - `Boss` (the first enemy gets ×2.5 max HP and +2 levels).
    Add opt-in `TestBalanceZoneBands`. For bands 1–3, 8–10, 18–20 and 28–30,
    the company is 5 members at the band's low, middle and high level.
    Enemies follow the design's section 4 table:
    - groups of 2–3 at levels from band low −1 to band top −1;
    - a 4-group;
    - a boss +4 escorts at band top.
    In 35a the test **only reports** rows (`balanceRow` plus fallen and
    HP-lost means). Assertions arrive in 35b, which tunes for them.
    `TestBalanceMismatches` keeps its assertions; if smooth growth breaks
    them, record the new numbers and ask the owner rather than retuning here.

## Tasks

- [ ] **Tests first**, red before the code:
  - `RacialForLevel` equals today's values at levels 5, 10, 15, 20, 60,
    rises monotonically between them, and stays unchanged when the flag is
    off;
  - `StatPointsAt` boundaries 1, 2, 3, 4, 10, and its peak protection through
    a real `LevelUp`/`LoseLevel`/`LevelUp`;
  - HP at levels 10, 20, 21 and 60 per archetype (`HealthAfterFull` gives 60%
    of each rate);
  - migration grants the exact difference once, is idempotent on reload, and
    survives copyover;
  - `events.LevelUp` carries correct before/after values for one-level and
    multi-level gains.
- [ ] **Config and formulas:**
  - `config.progression.go` (field, Validate default, helpers);
  - `_datafiles/config.yaml` values and comments;
  - replace the modulo rule in `LevelUp`, `StatPointsAtLevel` and
    `simulate.go` with `StatPointsAt`.
- [ ] **Migration** on user load (`internal/users` load path, alongside the
  existing 30g4 vitals clamp). Test with a saved level-12 fixture: owed 6 − 2 = 4.
- [ ] **Level-up report:** event fields, `GrantXP` capture, `internal/milestones`,
  the template rewrite, the companion line and the experience template.
  Integration test through the real `GrantXP` → listener → rendered text.
- [ ] **Admin editor** fields, preview parity test and browser check
  (Playwright harness as 30g4 did).
- [ ] **Harness** options refactor, the fifth mirror template and
  `TestBalanceZoneBands` (report only). Run it at 100 fights a cell. Record
  the table in `docs/plans/2026-10-05-phase-35a-measurements.md` with
  before/after values for each starting class at levels 1, 5, 10, 20 and 30
  (stats, HP).
- [ ] **Help and tutorial:**
  - update `help progression`, `help stat-train`, `help stats`,
    `help experience` and `help health`. Remove the 5-level stat-step
    wording; explain smooth growth, a point every 2 levels, HP through 20,
    and the milestone table.
  - Fix any other page that says "level 5, 10, 15" for stats
    (`grep -rn "stat step\|every 5 levels\|every five" _datafiles`).
  - Update the creation-step and Practice Yard progression hints in
    `modules/tutorial/stages.go` if they quote the old rhythm.
  - Render tests in the `help_combat_test.go` pattern;
    `TestTutorialHelpPointersExist` passes.
- [ ] **Independent full-diff reviewer** (default model, reports only). Verify
  and fix findings with regressions, and record accepted and rejected
  findings in Project Status.
- [ ] **Final checks:** `make generate`, `make validate`, the JS lint (and Lua
  lint if available), `go test -race ./...`. Project Status entry;
  milestone entry flip (none flip in 35a); commit and merge.

## Acceptance

- At levels 5, 10, 15, … every race and class has today's automatic stats; in
  between, at least one stat rises on most level-ups.
- Stat points: 1 at level 2, 5 at level 10, 30 at level 60. Regained levels
  grant nothing; existing characters receive their difference exactly once.
- HP after level 20 grows at 60% of the class rate.
- A real level-up shows Health, Mana and stat changes and the next milestone.
  A companion's level-up tells the leader.
- The zone-band table runs without stalls and is recorded. Mismatch
  assertions hold or the owner has been asked.
- No world-time change; saves, copyover and the tutorial replay are intact.
