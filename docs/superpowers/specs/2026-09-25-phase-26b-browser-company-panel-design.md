# Phase 26b: Browser Company Panel

Implements the browser half of the
[player information surfaces spec](2026-09-23-player-information-surfaces-design.md),
after [Phase 26a](2026-09-24-phase-26a-company-summary-text-design.md).

The open decisions below were settled by applying this design's
recommendations, as in Phases 25a and 25b, under the owner's "continue with
next phase" instruction (2026-09-25).

## Prior-art check

- **GMCP** (`modules/gmcp`): payloads go out as `GMCPOut{UserId, Module,
  Payload}` events and are JSON-marshalled in `dispatchGMCP`, which sends
  nothing to a connection that hasn't accepted GMCP. The web client is
  always GMCP-enabled.
  - A client can ask for a namespace with `!!GMCP(<Name>)`. `HandleWebGMCP`
    runs on the connection goroutine and only queues an event.
  - `Party` and `Party.Vitals` carry GoMud's human party.
- **Web client** (`_datafiles/html/public/static/js`):
  - `webclient-core.js` stores each payload in `Client.GMCPStructs`, nesting
    on dots, and hands it to `VirtualWindows.handleGMCP`.
  - `window-party.js` renders the human party. It builds names into
    `innerHTML`, and a vitals-only payload with no members makes it show
    "Not in a party".
- **Read model** (26a): `companyview.For` builds the summary on the game
  loop. `Refresh` runs on every `NewRound` and after every command.

## Decisions

1. **Payloads.** `Company` is the full snapshot and `Company.Vitals` the
   lightweight update. Both are keyed by member key (`leader`,
   `companion:<id>`), never by name. The snapshot has:
   - **Leader:** name, level, archetype, HP, needs (value, label, warn),
     warmth, formation cell.
   - **Members:** each companion's ID, key, name, status
     (`present`/`awaiting`/`dead`), level, archetype, HP (present only),
     needs, cell, rescue seconds (dead only), and chemistry tier.
   - **Company:** alive, dead, load, activity, rest tier, checkpoint.

   An unknown value is omitted (`null`), never filled.

   `Company.Vitals` carries only each member's HP and needs. An empty
   `Company` means no company data at all; it is sent to a player whose
   company can't be read.
2. **Sent when it changes, from the refresh points 26a already has.**
   - `companyview` gains an `OnRefresh` hook, fired with each fresh
     summary.
   - The GMCP module builds both payloads from the summary, plus chemistry
     and the leader's cell from the company seams. It compares them with
     the last ones it sent that user (in memory).
   - It sends the snapshot when the structural part changed, `Company.Vitals`
     when only vitals did, and nothing otherwise.
   - This catches every change the summary can see (recruitment, dismissal,
     formation, death, resurrection, needs, rest) with no hook per
     mutation, and at most one round late.
3. **Resend in full on login, copyover, and request.**
   - `PlayerSpawn` (login and copyover reconnection) clears the user's
     last-sent record, so the next refresh sends the full snapshot.
   - `!!GMCP(Company)` queues a request event that does the same.
   - The Party window asks when it first mounts, so a reloaded browser tab
     catches up.
   - `PlayerDespawn` drops the record.
4. **Privacy.** A player's payload goes only to their own user ID; nothing
   is broadcast. The payload has no other player's data.
5. **The Party window gains a Company section**, above a Players section
   for GoMud's human party, with a heading on each so the two are never
   confused.
   - **Formation:** a 3×3 table, front row first, whose cells hold the
     member's name as text. It has a caption and needs no colour to read.
   - **Member cards:** name, level, and archetype; an HP bar with its
     numbers as text; needs as short words, marked when they warn;
     chemistry; and, for the dead, "Fallen: 1h 30m to raise".
   - **Layout:** flex, so cards wrap to one column at narrow widths.
   - **Safety:** every name and label goes in through `textContent` or
     `createElement`, never `innerHTML`. That applies to the Players
     section's names too, fixed while here.
   - **Robustness:** a `Company.Vitals` merges into the known roster and
     never replaces it. A partial or missing field is skipped, never
     guessed. A company member is never shown as a human party member.
   - "Not in a party" shows only in the Players section. With no company
     and no party, the window says "No company or party".
6. **No new locks.** The payloads are built on the game loop, inside the
   refresh. The last-sent map is touched only there and by `PlayerSpawn`,
   `PlayerDespawn`, and the request event, all on the game loop.

## Scope

**In scope:**

- `internal/companyview`: `OnRefresh`, and the leader's formation cell.
- `modules/gmcp`:
  - `gmcp.Company.go`: payload builders, change detection, spawn and
    despawn handling, and the request event;
  - the `Company` identifier in `HandleWebGMCP`;
  - the help template.
- `_datafiles/html/public/static/js/windows/window-party.js`: the Company
  section and safe DOM.
- Tests: Go unit and wiring tests, plus a browser check with Playwright.
  The browser check loads the real `window-party.js` with a stub client,
  at desktop and narrow widths, and checks keyboard focus and screen-reader
  text.

**Deferred:** clickable commands from the panel (e.g. `formation move`),
and the tutorial's use of the panel (Phase 27).

## Constraints

- The world clock is never advanced.
- Payloads are rebuilt from durable state after a restart or copyover, and
  nothing new is persisted.
- The `Party` and `Party.Vitals` payload shapes are unchanged; the new
  namespace is additive, per the GMCP guide.

## Acceptance criteria

- **Go:**
  - the snapshot's shape covers present, awaiting, and dead members;
    unknown values are `null`; keys are member keys;
  - an unchanged summary sends nothing, a change to vitals alone sends
    `Company.Vitals`, a change to the roster sends `Company`, and a
    `PlayerSpawn` or request resends the snapshot;
  - nothing goes to another user.
- **Wiring:** through `plugins.Load` with the real modules and GMCP
  captured from `GMCPOut` events:
  - login sends the snapshot, a recruit resends it, and damage sends only
    vitals;
  - a death shows the member dead with its time, and a resurrection shows
    it present;
  - the clock is unchanged.
- **Browser** (Playwright, real script, stub client):
  - the Company and Players sections are distinct, and a company shows
    with no party;
  - the formation table and cards render;
  - a name containing markup renders as text;
  - a vitals-only payload keeps the roster;
  - it works at 360px wide, can be reached by keyboard, and the
    formation's text is in the accessibility tree.
- `go test -race ./...`, `make generate`, `make validate`, and `jshint` on
  the changed script pass.
