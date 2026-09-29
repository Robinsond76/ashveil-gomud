# Phase 29c Narration Voice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Combat reads in a dark, physical voice with every mechanic in
lowercase parentheses at the end of a line.

**Architecture:** Text-only. The weapon message files are rewritten; the
combat code picks the pool (crit-aware) and appends `(N damage)` /
`(critical hit, N damage)`; names get articles through two `internal/util`
helpers; opener, closing, and death lines are rendered in `internal/hooks`
where the fight's stream events are produced; spells rewrite their own text.

**Tech Stack:** Go, YAML message pools, goja spell scripts, testify.

**Spec:** [2026-09-28-phase-29c-narration-voice-design.md](../specs/2026-09-28-phase-29c-narration-voice-design.md)
(and the feature spec it builds on, [combat narration](../specs/2026-09-26-combat-narration-design.md)).

## Global Constraints

- No `!`, no ALL-CAPS word, no `***` in any combat line this phase touches.
- Mechanics in lowercase parentheses at the line's end: `(5 damage)`,
  `(critical hit, 9 damage)`, `(5 damage, 2 blocked)`, `(4 healed)`,
  `(chanting: Magic Missile, 1 round)`.
- A name whose first visible letter is lowercase gets "the"; a capitalised
  name gets none.
- Only a real critical hit draws the `critical` pool; a non-crit caps at
  `heavy`.
- Text only: no state, no clock, nothing persisted.
- Existing tests that assert old text are updated deliberately to the new
  text, never loosened.

---

### Task 1: Article helpers

**Files:**
- Create: `internal/util/article.go`
- Test: `internal/util/article_test.go`

**Interfaces:**
- Produces: `util.Article(name string) string`,
  `util.CapitalizeFirst(line string) string`.

- [x] **Step 1: Write the failing test** — table cases:
  `Article("bandit captain") == "the bandit captain"`;
  `Article("Garrick Vane") == "Garrick Vane"`;
  `Article("<ansi fg=\"mobname\">rat</ansi>") == "the <ansi fg=\"mobname\">rat</ansi>"`;
  `Article("") == ""`; `Article("the rat") == "the rat"` (no double article).
  `CapitalizeFirst("the rat bites.") == "The rat bites."`;
  `CapitalizeFirst("<ansi fg=\"x\">the</ansi> rat") == "<ansi fg=\"x\">The</ansi> rat"`;
  `CapitalizeFirst("")`, `CapitalizeFirst("<ansi>")` unchanged.
- [x] **Step 2:** `go test ./internal/util -run 'TestArticle|TestCapitalizeFirst'` → FAIL (undefined).
- [x] **Step 3: Implement.** Both scan past `<...>` tags to the first
  visible rune. `Article` returns the name unchanged when that rune is
  upper-case or the visible text already starts with "the "; else prefixes
  `"the "`. `CapitalizeFirst` upper-cases that rune.
- [x] **Step 4:** tests PASS.
- [x] **Step 5: Commit** `feat(util): article and capitalisation helpers for narration (29c)`.

### Task 2: Pool choice, suffixes, and articles in the combat round

**Files:**
- Modify: `internal/items/attack_messages.go` (`GetAttackMessage`)
- Modify: `internal/combat/combat.go:174-215` (`buildCombatMessages`),
  `:345-440` (dodge lines, pool choice, `***`, suffixes, blocked)
- Test: `internal/items/attack_messages_test.go` (create),
  `internal/combat/narration_test.go` (create)

**Interfaces:**
- Consumes: Task 1 helpers.
- Produces: `items.GetAttackMessage(subType ItemSubType, pctDamage int, crit bool) AttackOptions`;
  `combat.damageSuffix(damage int, crit bool) string` →
  `" (5 damage)"` / `" (critical hit, 5 damage)"`.

- [x] **Step 1: Failing tests.**
  - `attack_messages_test.go`: with a test group registered for a subtype
    whose pools carry distinct marker lines, `GetAttackMessage(st, 150, false)`
    returns the `heavy` pool, `(st, 40, true)` the `critical` pool,
    `(st, 0, false)` `miss`.
  - `narration_test.go` (the `edge_test.go` fixtures: a 1d1 sword, lit
    room): over 300 `calculateCombat` rounds, every hit's source, target,
    and room lines end in `damageSuffix(res.DamageToTarget…, res.Crit)`
    (the defender's with `, N blocked)` when reduced), no line contains
    `***`, and every miss line has no `(`. A crit round is reached by
    giving the source the `accuracy` buff flag and looping.
- [x] **Step 2:** run both → FAIL.
- [x] **Step 3: Implement.**
  - `GetAttackMessage`: `crit` → `Critical`; else the tiers with
    `>= 75` → `Heavy` (the `>= 101` branch removed).
  - `buildCombatMessages`: `{source}`/`{target}` values through
    `util.Article`; each returned message through `util.CapitalizeFirst`.
  - Remove the `***` block. After building, when `attackTargetDamage > 0`
    append `damageSuffix` to attacker and room lines; the defender's line
    gets `(N damage, M blocked)` / `(critical hit, N damage, M blocked)`
    when `attackTargetReduction > 0` (replacing `[you blocked N]`).
  - Dodge lines: `The <target> twists aside from your blow.` /
    `You twist aside from the blow.` (articles via Task 1).
- [x] **Step 4:** `go test ./internal/items ./internal/combat` PASS.
- [x] **Step 5: Commit** `feat(combat): crit-only critical pool, damage suffixes, articles (29c)`.

### Task 3: Rewrite the eight weapon message files

**Files:**
- Modify: `_datafiles/world/default/combat-messages/{bludgeoning,claws,cleaving,generic,shooting,slashing,stabbing,whipping}.yaml`
- Test: `internal/items/attack_messages_shipped_test.go` (create)

- [x] **Step 1: Failing data test.** Load every shipped file (as
  `LoadDataFiles` does, from `_datafiles/world/default/combat-messages`),
  walk every line of every pool: none contains `!`, `{damage}`, or a word
  of two or more letters in ALL-CAPS outside `<ansi ...>` tags; every group
  still `Validate()`s; every pool variant that existed still has at least
  one line.
- [x] **Step 2:** run → FAIL on today's files.
- [x] **Step 3: Rewrite.** Keep the header comment (drop `{damage}` from
  it), `optionid`, every pool and variant, and `{source}`, `{target}`,
  `{itemname}`, `{sourcetype}`, `{targettype}`, `{exitname}`,
  `{entrancename}`. Three or four lines per variant. Names are written
  bare (`{source}`); articles and capitals come from code. Voice per the
  reference mock: `miss` "cuts only air", `weak` a nick or graze, `normal`
  a solid blow, `heavy` a strong, clean blow (never "critical"),
  `critical` decisive and bloody. `prepare`/`wait` are aiming, circling,
  drawing breath.
- [x] **Step 4:** data test and `go test ./internal/items ./internal/combat` PASS.
- [x] **Step 5: Commit** `content(combat): weapon text in the narration voice (29c)`.

### Task 4: Names and engagement lines

**Files:**
- ~~Modify: `modules/company/runtime.go:48` (clear `charmed` after `Charm`)~~ —
  already done by Phase 32a (`Character.CharmAsCompanion`, tested in
  `internal/characters/companion_name_test.go`), merged while 29c was open
- Modify: `internal/usercommands/attack.go:216,273,277`,
  `internal/mobcommands/attack.go:126,129,150` (remove "prepares to fight")
- Modify: `internal/hooks/combat_battle.go:195,205,365,464-478`,
  `internal/hooks/combat_engagement.go:286,390-392,547`,
  `internal/hooks/combat_formation.go:538,564` ("turns on" → "turns toward")
- Create: `internal/hooks/combat_narration.go`
- Test: `internal/hooks/combat_narration_test.go`,
  `modules/company/runtime_test.go` (or the nearest existing runtime test)

**Interfaces:**
- Produces: `hooks.turnsToward(who, target string) string` →
  `"The bandit captain turns toward Garrick Vane."` (names are coloured
  names; `who` may be `"You"` → `"You turn toward …"`).

- [x] **Step 1: Failing tests.** `turnsToward` cases (mob→player,
  player→mob, "You"); `groupTurnsOn` renamed `groupTurnsToward` and its test
  (`combat_stream_test.go:88`) updated to `"The ruffian turns toward"`;
  a summoned companion's formatted name has no `♥`.
- [x] **Step 2:** FAIL.
- [x] **Step 3: Implement.** Replace each site's `fmt.Sprintf` with
  `turnsToward`; delete the six "prepares to fight" sends (keep their aggro
  logic); `mob.Character.SetAdjective("charmed", false)` after `Charm`.
- [x] **Step 4:** `go test ./internal/hooks ./internal/usercommands ./internal/mobcommands ./modules/company` —
  update `wiring_combat_test.go:326,408,451,501-505` from "turns on" to
  "turns toward" and drop the `***` trim. PASS.
- [x] **Step 5: Commit** `feat(combat): no charmed tag on companions, no "prepares to fight", "turns toward" (29c)`.

### Task 5: Opener, closing, and death lines in order

**Files:**
- Modify: `internal/hooks/combat_narration.go` (pools and renderers)
- Modify: `internal/hooks/combat_battle.go` (`beginBattle`: opener)
- Modify: `internal/hooks/combat_stream.go` (`fightSides.end`: closing on victory)
- Modify: `internal/hooks/NewRound_DoCombat.go:1270-1290` (`handleAffected`: print the death/beaten line, queue `suicide quiet`)
- Modify: `internal/mobcommands/suicide.go` (`DeathLine`, `quiet`),
  `internal/usercommands/suicide.go:66` (player death line)
- Test: `internal/hooks/combat_narration_test.go`,
  `internal/mobcommands/suicide_test.go` (create if absent)

**As built:** the death pools live in `internal/combat/death_lines.go`
(`DeathLine`, `BeatenLine`, `PlayerDeathLine`, each taking a tagged name)
so `mobcommands`, `usercommands`, and `hooks` share them; the voice check
is `util.NarrationVoiceProblem`. No closing line when an enemy left the
room alive (the lines speak of the last one falling). A player's own death
line still comes from the queued `suicide` (its script hook may cancel the
death), so it can follow a closing line.

**Interfaces:**
- Produces: `mobcommands.DeathLine(name string) string` (a random line from
  a pool, article applied, capitalised); `hooks.fightOpener(groups []string) string`,
  `hooks.fightClosing(groups []string) string` — pool keyed by the first
  group with one, else generic (`""` key).

- [x] **Step 1: Failing tests.** Every pool line (openers, closings, death
  lines, player death lines) has no `!` and no ALL-CAPS word; the
  `bandits` key picks its own pool; an unknown group falls back to generic;
  `DeathLine("bandit captain")` starts with `The bandit captain`;
  `suicide quiet` sends no room line (a mob in a test room).
- [x] **Step 2:** FAIL.
- [x] **Step 3: Implement.**
  - Pools: generic, `bandits`, and the practice squad's group (straw
    soldiers: `The last straw soldier topples. The drill is done.`).
  - `beginBattle`: after `Open`, `room.SendText(fightOpener(firstEnemyGroups))`.
  - `fightSides.end`: on `OutcomeVictory`, send `fightClosing` to the fight's
    room before the summary.
  - `handleAffected`: for a mob without `revive-on-death`, send
    `DeathLine` (or, practice, the "is beaten" line) to the room at once and
    queue `suicide quiet`; with it, queue plain `suicide` as today.
  - `suicide`: `quiet` suppresses the death/beaten room line only; the rest
    of `quiet` behaves as the default path.
- [x] **Step 4:** focused tests PASS; `go test ./modules/tutorial` updated
  deliberately where it asserts "is beaten and yields the field" order.
- [x] **Step 5: Commit** `feat(combat): one opener, one closing line, death lines before them (29c)`.

### Task 6: Company fallen notice

**Files:**
- Modify: `modules/company/death.go:76-82,141-142`
- Test: `modules/company/death_test.go`, `modules/company/wiring_resurrect_test.go:181`

**Interfaces:**
- Produces: `allowanceWords(seconds int) string` → `3 hours`,
  `2 hours 30 minutes`, `1 hour 1 minute`, `45 seconds`.

- [x] **Step 1:** failing table test for `allowanceWords`; `death_test.go:103`
  asserts the notice starts with four spaces and reads `has fallen. You have 3 hours of your own time`.
- [x] **Step 2:** FAIL.
- [x] **Step 3:** implement; the notice becomes
  `    <name> has fallen. You have <words> of your own time to reach a church or a village shaman and <ansi fg="command">resurrect</ansi> them.`
- [x] **Step 4:** update `wiring_resurrect_test.go:181`; PASS.
- [x] **Step 5: Commit** `feat(company): indented fallen notice in words (29c)`.

### Task 7: Spells

**Files:**
- Modify: `internal/scripting/actor_func.go` (`GetCombatName`),
  `internal/scripting/objecttypes.go`, `internal/web/api_v1_scripting_dts.go`
- Modify: `_datafiles/world/default/spells/{mm,sparks,heal,healall}.js`;
  `!`/ALL-CAPS only in `{floatinglight,illum,poly,curepoison,aidskill,tameskill}.js`
- Test: `modules/company/wiring_spells_test.go` (create; the brawl harness)
  and a data test in `internal/spells` that no shipped spell script string
  literal contains `!`

**As built:** `GetCombatName(startOfLine)` (a capital when the name opens a
line) and `ChantRoundsLeft()` (rounds before release, counting this one,
so each chant line is accurate: onCast prints `waitrounds + 1`). Spells
apply their effect first and print what `AddHealth` actually changed. The
fizzle lines in `NewRound_DoCombat.go` lose `***` and `!` too. A player's
XP line ("You gained N experience points!") is GoMud's and out of scope.

**Interfaces:**
- Produces: JS `actor.GetCombatName()` → coloured name with its article
  (`the <ansi fg="mobname">bandit captain</ansi>`; a user's own name bare).

- [x] **Step 1: Failing tests.** In the brawl world: Aria casts `mm` at a
  bandit → her line ends `(N damage)` with N = health lost and she saw a
  `(chanting: Magic Missile, 1 round)` line; Oswin's `healall` (cast
  through the script entry the battles tests use) prints one cast line and
  one indented line of `Name (N healed)` entries joined by ` · `.
- [x] **Step 2:** FAIL.
- [x] **Step 3:** implement per design §7; the round counts are constants
  per script matching each spell's yaml `waitrounds`.
- [x] **Step 4:** PASS; `make js-lint` if it runs on this host.
- [x] **Step 5: Commit** `content(spells): chanting and release in the narration voice (29c)`.

### Task 8: Player help and tutorial

**Files:**
- Create: `_datafiles/world/default/templates/help/narration.template`
- Modify: `_datafiles/world/default/keywords.yaml` (topic under the combat
  help category; aliases `critical`, `crit`, `healed`, `chanting`,
  `combat-text`; `damage` is GoMud's own page, which now links here)
- Modify: `_datafiles/world/default/templates/help/combat.template` (link),
  and any help page that quotes `***`, "prepares to fight", or "turns on"
  (grep `templates/help`)
- Modify: `modules/tutorial/stages.go` (Practice Yard hint → `help narration`)
- Test: `internal/usercommands/help_combat_test.go` (add `narration`),
  `TestTutorialHelpPointersExist`

- [x] **Step 1:** add `narration` to the help render test → FAIL.
- [x] **Step 2:** write the page (what each parenthesis means, that every
  miss is shown, that critical hits are named, the opener/closing), index,
  link, hint.
- [x] **Step 3:** `go test ./internal/usercommands ./modules/tutorial` PASS.
- [x] **Step 4: Commit** `docs(help,tutorial): help narration (29c)`.

### Task 9: Wiring test through the real round

**Files:**
- Create: `modules/company/wiring_narration_test.go` (brawl harness,
  `newBrawl`, `fightToTheEnd`, `b.fight()`)

- [x] **Step 1: Write it** (should pass after Tasks 2–6; write it before
  Task 2 lands if executing strictly tests-first, and watch it fail):
  over the full 5v5 with the shipped config —
  - every line naming a hit ends in `(N damage)` or
    `(critical hit, N damage)` (a regexp over the transcript's hit lines,
    identified by the weapon pools' non-miss lines);
  - no `***`, no `prepares to fight`, no `♥`;
  - exactly one opener and, after the last bandit's death line, exactly
    one closing line;
  - at least one `turns toward` line.
  A second test forces a crit (captain given the `accuracy` flag, looped
  over seeds) and checks the `critical hit` suffix.
  **As built:** hits are matched to the stream's `Attack` events (each
  `(N damage)` / crit is narrated at least as often as the stream reports
  it); openers and closings are counted against `FightStart` and victorious
  `FightEnd` events; the crit test runs up to six fights until one lands
  (the unit test in `internal/combat` forces one directly). Mutation check:
  queuing plain `suicide` again makes the closing-after-death assertion fail.
- [x] **Step 2:** `go test ./modules/company -run Narration` PASS.
- [x] **Step 3: Commit** `test(company): narration through the real combat round (29c)`.

### Task 10: Verify, review, record, merge

- [x] Capture before/after transcripts of the same seeded brawl
  (`master` vs branch) into the scratchpad for the review.
- [x] Independent reviewer subagent over `git diff master..HEAD` with the
  design doc and invariants.
- [x] Verify each finding, fix with regression tests (focused package
  tests only).
- [x] Once, after the fixes: `go test -race ./...`, `make generate`,
  `make validate`.
- [x] `docs/PROJECT_STATUS.md`: 29c row Complete, work-log entry with
  **Review:** line, known issues folded; remove this phase's design/plan
  per the pruning convention only after merge.
- [ ] Merge to `master`, push, remove the worktree.
