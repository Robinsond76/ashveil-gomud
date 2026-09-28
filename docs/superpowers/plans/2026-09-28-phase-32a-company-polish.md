# Phase 32a: Company Polish — Plan

Design: [32a design](../specs/2026-09-28-phase-32a-company-polish-design.md).
Decisions 1–6 are the owner's (recommendations applied, 2026-09-28).

One refinement to the design's "Module" list: the companion predicate for
`♥friend` is a `Companion` flag on the companion's `CharmInfo`, set by
`modules/company` when it spawns the companion. `characters` still never
imports `company`, the flag needs no lock, and a companion befriended away
by someone else gets a fresh `CharmInfo` and its `♥friend` back.

## Task 1: No `♥friend` on company members

- [ ] Tests first (`internal/characters`): a companion charm leaves
  `charmed` out of the formatted name; an ordinary charm keeps it;
  `RemoveCharm` clears both.
- [ ] `CharmInfo.Companion`, `Character.CharmAsCompanion`,
  `Character.IsCompanion`; `getFormattedName` drops `charmed` for a
  companion.
- [ ] `modules/company` `Spawn` charms as a companion.
- [ ] `party` and GMCP `Party` show `Company` for a companion's status,
  `♥friend` for other charmed mobs.
- [ ] Wiring (`modules/company`): `look` in a room with a live companion
  shows no `♥friend`; a charmed non-companion still does; `party`.

## Task 2: One line for a company on the move

- [ ] Tests first (pure, `internal/usercommands`): the leave/arrive
  wording with and without companions.
- [ ] `usercommands/go.go`: a leader with an attached companion in the
  room they leave sends "Dain leads their company west." to the origin
  and "Dain arrives from the east, their company behind." to the
  destination.
- [ ] `mobcommands/go.go`: a companion whose leader is already in the
  destination prints no leave/arrive/"moving around" line.
- [ ] Wiring (`modules/company`): a leader walks with two companions
  through the real `go`; the leader sees no companion line; a watcher in
  the destination sees exactly one arrival line and one in the origin sees
  exactly one departure line; a companion walking alone keeps its lines.

## Task 3: No drink flourish

- [ ] `buffs/34-hydrated.js`: remove the `onStart` message; it still
  cancels Thirsty.
- [ ] Wiring: `drink` water shows the status line and not the flourish.

## Task 4: The camp in the room

- [ ] Tests first (`internal/camping`): `CampLines` for lit, unlit, the
  viewer's own, someone else's, and none.
- [ ] `internal/camping`: a room-camps snapshot seam
  (`SetRoomCampsReader`, `RoomCamps`, `CampLines`).
- [ ] `modules/camping`: the lit-room snapshot becomes a room-camps
  snapshot under `litMu` (lock order unchanged: `mu`, then `litMu`);
  `RoomHasLitFire` reads it.
- [ ] `look` prints the camp lines after the sky lines.
- [ ] Wiring (`modules/camping`): `camp` then `look` shows the camp
  line; `camp fire` makes it lit; `camp break` removes it; a second player
  sees "Dain's camp".

## Task 5: Recruiters shown in the room

- [ ] Tests first (`modules/company`): the notice line for free, priced,
  claimed, in-company, refused, and none left.
- [ ] `internal/company`: an optional `RecruiterViewProvider`
  (`RecruiterLines`, `LookCandidate`) on the registered provider.
- [ ] `modules/company`: implements both over today's config.
- [ ] `look` prints the notice lines after the camp lines, and
  `look <candidate>` falls back to the candidate before "Look at what???".
- [ ] Wiring (shipped config): the Muster Yard lists Tamsin and Oswin;
  after `company recruit tamsin`, only Oswin; the Oath Stone lists Corvin
  as won't join; `look tamsin` shows the description.

## Task 6: A readable formation grid

- [ ] `window-party.js`: the grid uses the panel's body size and colour;
  cells wrap and carry a `title` with the full name; the leader keeps its
  accent.
- [ ] Browser check (Playwright, Chromium): the cell's font size equals the
  panel's, and the title carries the full name, at default and narrow
  widths.

## Task 7: Player help and tutorial

- [ ] `help company`: moving together, recruiters in the room.
- [ ] `help camp`: the camp in the room.
- [ ] `help drink`: the status line.
- [ ] The Company lesson's first hint says to look at the hiring post.
- [ ] Tests: the pages render through `help`;
  `TestTutorialHelpPointersExist` passes.

## Task 8: Verify, review, record

- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review over the phase diff; verify each finding.
- [ ] `docs/PROJECT_STATUS.md` work-log entry with the **Review:** line.
