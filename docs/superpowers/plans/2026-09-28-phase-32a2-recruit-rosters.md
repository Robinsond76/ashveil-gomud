# Phase 32a2: Per-Player Recruit Rosters — Plan

Design: [32a2 design](../specs/2026-09-28-phase-32a2-recruit-rosters-design.md).
Its decisions are the owner's (recommendations applied, 2026-09-28); this
plan builds on 32a (merged as #3).

Refinements to the design, decided here:

- **Refreshes live in memory; they reach disk with the next company
  save.** The company file also carries in-memory gear snapshots, which
  must reach disk only on the existing seams (22b; chemistry follows the
  same rule). So a `look` never writes the file. A hire saves at once,
  roster included, through `enlist`. A crash before the next save only
  rerolls the faces that changed since; it can't duplicate anything or
  hand out a hired candidate twice.
- **Staggered from the start:** a new roster's candidates arrive part way
  through their stays, so they leave one at a time. A candidate who left
  long ago is replaced by one part way through its own stay (not a chain
  of every face in between), so a week away costs one step per slot.
- **Recruit templates are 80–84** (warrior, rogue, wizard, cleric,
  ranger), leaving 70–79 for the other open phases.
- **Only settlement recruiters roll rosters** (`Generated: true` on the
  Waymark Inn and the Trappers' Post). The tutorial's recruiters stay
  authored only.
- **Config keys are flat** (`RosterSize`, `CandidateStayRoundsMin`, …),
  since the module config loader flattens maps; lists stay lists.
- **The notice keeps 32a's one-line form**, with the regulars first and
  then the viewer's generated candidates.

## Task 1: Companions carry their own names (`internal/company`, `modules/company`)

- [x] Tests first: `Companion.Name`/`Description` round-trip through the
  store and `Registry.Get` copies; `memberName` prefers `Name`;
  `resolveCompanion` finds two companions of one template by their own
  names and by `#id`.
- [x] `Companion.Name`, `Description` (`omitempty`), `Registry.SetIdentity`.
- [x] One helper, `memberName(c, fallback)`, replaces every
  `templateName(c.MobTemplateID, …)` on a companion (status, formation,
  gear, alignment view, dismissal, archetype, resurrection match,
  `Roster` for survival, `MemberView` for GMCP, lost companions).
- [x] `Runtime.Spawn` takes a `domain.Identity`; the native runtime puts
  a set name and description on the live mob, so combat text, `look`, and
  the room follow. `enlist`, `restoreForLeader`, and resurrection pass it.

## Task 2: Roster domain and generation (`internal/company/roster.go`)

- [x] Tests first (seeded RNG): generation fills `Size` slots with
  unique given names (against the company and the recruiter's regulars),
  level in leader −1..+1 and at least 1, price by level and archetype,
  archetype weights (a weight-only list yields only that archetype),
  alignment in range; stays are staggered; a lapsed candidate is
  replaced; a long absence refreshes in one pass; a hire opens a slot
  that refills only after its delay; refresh with nothing due reports no
  change.
- [x] `Candidate`, `Roster`, `RosterRules`, `RefreshRoster`,
  `HireFromRoster`; `Record.Rosters` (kept by `Put`, copied by `Get`,
  stored by the wire record).

## Task 3: Module wiring (`modules/company`)

- [x] Config parsing (`parseRosterRules`) with defaults, and recruiters'
  `Generated` and `ArchetypeWeights`.
- [x] `roster(user, room)`: lazily refreshes the viewer's roster from
  `util.GetRoundCount()` (read only; a `roundNow` seam for tests).
- [x] `company recruit` lists and hires generated candidates: alignment
  gate on the candidate's own alignment, price, capacity, then `enlist`
  with the candidate's identity, level, and alignment; the roster change
  rides in the same save and rolls back with it.
- [x] `company inspect [name]` and `look [name]` read the candidate
  (line of character, alignment verdict, price); the notice lists them.
- [x] Content: mob templates 80–84 (starter gear, `itemdropchance 0`, no
  gold, no loot); `CompanionArchetypes` entries; name, byname, and trait
  lists; `Generated` on the two settlement recruiters.
- [x] Wiring tests (`wiring_roster_test.go`, `plugins.Load`, shipped
  config and templates, real commands):
  - two players at the Waymark Inn see different rosters; one hiring
    leaves the other's unchanged;
  - a hired generated recruit joins under its generated name and keeps
    it in `company status`, `formation`, `look`, and `CompanyMembers`
    (GMCP), and after a despawn/respawn and a `plugins.Load` reload;
  - two generated warriors in one company are both addressable;
  - rounds past a stay: `look` shows a new face;
  - a far-aligned candidate is refused by `company recruit`;
  - the tutorial's recruiters still list Tamsin, Oswin, and Corvin, and
    nothing generated.

## Task 4: Player help and tutorial

- [x] `help company`: recruiting from a roster of your own that changes
  over time (faces come and go; what you hire doesn't touch anyone
  else's list). `help recruit` already aliases `company`.
- [x] Departure lesson hint: settlements post their own recruits, and
  the list changes.
- [x] Tests: the help page renders the new text; `TestTutorialHelpPointersExist`.

## Task 5: Verify, review, record

- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Independent review of the phase diff; verify each finding; fix with
  regression tests.
- [x] `docs/PROJECT_STATUS.md` work-log entry with its **Review:** line;
  `modules/company/AGENTS.md` notes.
