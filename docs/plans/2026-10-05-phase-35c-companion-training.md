# Phase 35c — companion training and optional skills

Implements section 5 of the owner-approved
[level impact and class power design](../designs/2026-10-05-level-impact-class-power-design.md).
Branch and worktree: `phase-35c-companion-training`, from master after 35a
merges. It can run before or after 35b; don't run both in one worktree.
Read the [phase 35 handoff](2026-10-05-phase-35-handoff.md) first.

## Goal

Companions earn training points as they level. The player spends them with
`company train` to teach a companion **optional** skills. Cooking is the first
one: the best cook in the company cooks at camp. Recruits can arrive already
trained. **Class specialist skills stay automatic** by level (33f). Scribe
joins the optional list in 36a.

## Code context (master after 35a)

- **Companion record:** `internal/company/company.go` `Companion` (ID,
  `Archetype`, `State *MemberState`, `Disposition`, `Death`). The saved part
  is `internal/company/state.go` `MemberState`: level, experience, gear, gold,
  wounds and vitals.
  - Live mobs get it through `applyState`, and back through `Snapshot`
    (`modules/company/runtime.go`).
  - Company file writes use the module's validated save, with the 33g journal
    for cross-file operations.
- **Growth (33h1):** a companion's trained stats are derived from its level's
  stat points and its archetype growth weights (`retrainMob`). No points are
  banked.
- **Specialists (33f):** class capabilities with companion ranks at levels
  1/10/20/30, resolved by best member with leader and ID tie-breaks
  (`company specialists`). Leave these unchanged.
- **Player training:**
  - `internal/usercommands/train.go` needs a room with `SkillTraining`
    ranges;
  - the cost of the next rank is that rank (1+2+3+4);
  - archetype `Skills` lists gate class skills.
  - Cooking is a shared trade skill (Phase 18b) any player can train.
- **Camp cooking:** `modules/camping/camp_specialists.go`. `cook` and
  `selectCampRecipe(user, …)` use only the **leader's**
  `GetSkillLevel("cooking")` against each recipe's `MinLevel`.
  `modules/camping/cooking_view.go` and the GMCP capabilities
  (`modules/gmcp/gmcp.CharCapabilities.go`) show the leader's rank.
- **Recruit rosters (32a2):** `internal/company/roster.go`:
  - `Candidate` (key, name, archetype, template, level, alignment, price,
    trait);
  - `generateCandidate(rules, ctx, taken, rng)`;
  - a hire copies the candidate into a `Companion`.
- **Command routing:** `company` subcommands are in
  `modules/company/company.go` (`case "specialists"`, `case "growth"`, …).

## Implementation decisions

1. **Optional skill registry.** Add `OptionalSkills` to the archetype overlay:

   ```yaml
   OptionalSkills:
     - { Skill: cooking, Archetypes: ["*"], MaxRank: 4 }
   ```

   - It is loaded and validated by `internal/archetypes`, with
     `archetypes.OptionalSkill(id)` and `CanLearn(archetype, skill)`.
   - Companions can train only skills on this list. Class skills and
     specialists stay derived.
   - 36a adds Scribe with `Archetypes: [wizard, cleric, witch]`.
2. **Saved state.** Add `Skills map[string]int` (trained ranks) and
   `GrantedSkills map[string]int` (ranks a recruit arrived with) to
   `MemberState`. Clone, snapshot and apply them. `applyState` writes the
   ranks into the live mob's `Character.Skills`, so ordinary
   `GetSkillLevel` works. Old saves have neither field; an absent map means no
   ranks.
3. **Derived points, never banked.**
   - `points available = earned(level) − spent`, where:
     - `earned(level) = configs.TrainingPointsAt(level)`, a helper beside
       35a's `StatPointsAt` using `TrainingPointsPerLevel` and
       `TrainingPointsEveryNLevels`;
     - `spent = Σ` over skills of the cost of ranks from `GrantedSkills[s]+1`
       to `Skills[s]`.
   - It can go negative after a level lost to death. Show 0 then, and train
     nothing until it is repaid; **ranks are never removed**.
   - Death, regain, re-summon, resurrection and copyover can't mint points,
     because nothing is stored but ranks.
4. **`company train` command.**
   - `company train` lists each companion's points, its optional skills and
     the next rank's cost.
   - `company train [member] [skill]` previews: the cost, the points left,
     where it can be trained, and the benefit text from the skill's help
     summary.
   - `company train [member] [skill] confirm` commits.
   - Member selectors follow 33g, resolving durable keys (`#N`) or names.
     Refuse a missing, ambiguous, dead, absent, separated or fighting member.
   - **Location:**
     - ranks 1–2 can be trained in the leader's own established camp (any
       camp, fire not required) or at a trainer;
     - ranks 3–4 need a room whose `SkillTraining` range for that skill
       includes the rank.
   - **Refusals** (each with its reason): in battle, travel or an active
     rest; not eligible (`CanLearn`); already at max rank; not enough points.
   - Commit through the company save. Confirm revalidates everything since
     the preview, and a repeated confirm is idempotent (rank already reached,
     so nothing to do). Announce only after the save succeeds.
5. **Best cook at camp.** Change `selectCampRecipe` and `cook` to take a
   cook's character and rank: the highest `cooking` rank among the leader and
   living companions present in the camp room, ties to the leader, then the
   lowest member ID.
   - The message names the cook: "Brannoc cooks a hunter's stew."
   - The blocked text names the best rank available.
   - The camping view and GMCP capability show the best cook and their rank.
   - The Waymark Inn hearth cooking (18b) stays leader-only. Say so in help.
6. **Recruits with skills.** `generateCandidate` gives about **20%** of
   candidates one optional skill they are eligible for, at rank 1 (80%) or
   rank 2 (20%), using the seeded `rng` so the tests stay deterministic.
   - The price rises by 15% per rank.
   - The candidate's card (`company inspect` of a candidate, and the
     recruiter list) shows the skill.
   - On hire, the ranks go into both `Skills` and `GrantedSkills`.
   - Old rosters on disk without the field load unchanged.
7. **Views.**
   - `company inspect [member]` shows trained optional skills and points
     available.
   - GMCP `Company` member data adds `skills` and `training_points`. The web
     Skills tab (32g) lists them read-only, with no browser train button in
     this phase.
   - `experience` companion rows add `(N training points)` when N > 0.

## Tasks

- [x] **Tests first:**
  - registry load and validation;
  - `CanLearn`;
  - the derived-points table across levels, including negative after a lost
    level;
  - granted ranks cost nothing;
  - `MemberState` clone, snapshot and apply round-trip, plus an old-save load;
  - the real `company train` command: preview, confirm, idempotent repeat,
    each refusal, camp versus trainer ranks, battle and travel refusal,
    save-failure rollback;
  - copyover keeps ranks with no extra points;
  - the best-cook choice and its tie-breaks through a real `camp cook`;
  - candidate generation (deterministic rng): skill, rank, price, and hire
    copying both maps;
  - GMCP payload fields.
- [x] **`internal/archetypes`** optional skills, `configs.TrainingPointsAt`,
  and the overlay data.
- [x] **`internal/company`** state fields and the derived-points helpers;
  `modules/company` `applyState` and `Snapshot`.
- [x] **The `company train` subcommand** with its save transaction (the
  existing company save path; no new cross-file journal is needed, since
  only the company file changes).
- [x] **The camping best cook** in `camp_specialists.go`, `cooking_view.go`
  and the GMCP capabilities.
- [x] **Roster generation and hire,** with the candidate display.
- [x] **Views:** inspect, GMCP, the web Skills tab and `experience`.
  Browser check with the Playwright harness.
- [x] **Help and tutorial:**
  - new `help company-train`, keywords `company train` and `train companion`,
    linked from `help company` and `help skills`;
  - update `help skills` (the class/optional split, companion points),
    `help cooking` (best cook at camp, inn hearth leader-only),
    `help growth` (stats automatic, optional skills trained),
    `help specialists` (still automatic) and `help company` (recruits may
    arrive trained);
  - a Campground lesson hint (`modules/tutorial/stages.go`) that companions
    can learn Cooking with `company train`;
  - render tests and `TestTutorialHelpPointersExist`.
- [x] **Independent full-diff reviewer.** Verify and fix findings with
  regressions; record them in Project Status.
- [x] **Final checks:** `make generate`, `make validate`, JS and Lua lint,
  `go test -race ./...`. Flip no milestone. Project Status entry, commit and
  merge.

## Acceptance

- A level-6 companion has 6 training points (shipped 1 per level). Training
  Cooking rank 1 and then rank 2 leaves 3.
- Points never increase through death, regain, re-summon, resurrection or
  copyover. A companion who loses a level keeps its ranks.
- Only eligible skills train, ranks 3–4 only at a trainer, and never in
  battle, travel or rest.
- The best cook present cooks at camp and is named.
- Some recruits arrive with a rank, cost more, and don't spend points for it.
- Help pages are accurate and indexed, and tutorial pointers resolve.
- No world-time change; company saves and allied ownership are intact.
