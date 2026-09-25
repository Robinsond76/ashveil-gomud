# Phase 27d: Tutorial Alignment Lesson and Browser Panel

The last slice of the [Ashveil tutorial spec](2026-09-23-ashveil-tutorial-design.md),
on 27a–27c (see the
[27c spec](2026-09-25-phase-27c-tutorial-practice-fight-design.md) for the
split). It adds stage 7 (Alignment), before Departure, and a browser panel
that shows the same stage, goal, and checklist as the terminal.

Open decisions were settled by applying this design's recommendations under
the owner's "carry on to 27d" instruction (2026-09-25).

## Prior-art check

- **Alignment (21a):**
  - Companions carry an alignment (shown as −100..100) and a loyalty.
  - They drift toward the rest of the company; the leader never drifts.
  - A companion whose loyalty runs out deserts.
  - `company summon` refuses a candidate more than `RecruitMaxGap` (60)
    from the company average.
  - `company alignment` shows the company average and each member.
  - `company inspect <candidate>` shows a recruitable candidate's alignment
    against the company and whether they'd be refused. It works on any
    recruiter's candidate, anywhere.
- **Standing (21b):** `standing` shows how a settlement regards the
  company, which sets market and inn markups or refusal.
- **Browser panel (26b):**
  - `modules/gmcp/gmcp.Company.go` builds a payload on
    `companyview.OnRefresh` and sends it only when it changes, only to that
    player. It resends on `PlayerSpawn`, on a `!!GMCP(Company)` request, and
    never to a telnet connection that hasn't accepted GMCP.
  - The web client's windows are `VirtualWindow`s fed from
    `Client.GMCPStructs`. The 26b Party window builds every string with
    `textContent`.
  - `scripts/browser/company-panel-check.mjs` drives the real window in
    Chromium.
- **Tutorial (27a–27c):**
  - A stage's `Inspections` are counted by registered command name through
    `usercommands.OnCommandDone`, which also carries the rest of the
    command line.
  - The course's text lives in `stages.go` and `view`.

## Decisions

1. **An Alignment stage before Departure**, in a new room 907 (the Oath
   Stone), appended to `TutorialRooms` (index 7). The order is Character,
   Company, Formation, Survival, Camp, Combat, Alignment, Departure.
2. **The lesson is inspections, with a harmless preview.** Nothing in it
   changes anyone's alignment.
   - **The goal:** run `company alignment`, `company inspect corvin`, and
     `standing`.
   - **Inspections can name a subcommand:** "company alignment" counts
     `company` with `alignment` as the first word of the rest. A key's
     command must be registered, as in 27b.
   - **The preview:** the Oath Stone's recruiter offers Corvin Blackthorn,
     an outlaw sellsword (a new mob 69, alignment −80, 150 gold). Inspecting
     him shows the gap to the company and the refusal (a gap over 60).
     Recruiting him is an ordinary paid recruit, and for a new company it is
     refused by the real gate anyway.
   - **The hints** explain drift, loyalty, desertion, the recruit gate, and
     settlement standing.
3. **A tutorial view for other surfaces.**
   - `internal/tutorial` gains a `View`: whether the course is active, the
     stage number and count, the title, the goal, the checklist (label and
     done), and the hints in plain text.
   - A provider that implements `Viewer` supplies it
     (`tutorial.ViewOf(userID)`). The terminal's `tutorial` output and the
     panel come from the same stage data and checks, so they can't disagree.
4. **A `Tutorial` GMCP package**, from `modules/gmcp`, built from
   `tutorial.ViewOf` on `companyview.OnRefresh` (every round and after
   every command, on the game loop).
   - It is sent on change, only to that player.
   - It is resent on `PlayerSpawn` and on `!!GMCP(Tutorial)`, and never to a
     telnet connection that hasn't accepted GMCP.
   - A player not in the course gets `{}` once.
5. **A Tutorial window in the web client** (`window-tutorial.js`).
   - It shows "Stage N of M: Title", the goal, a checklist, and the hints,
     all through `textContent`.
   - The checklist is a list whose items say "done" or "to do" in text, not
     only by mark or colour.
   - The window is docked on the right by default, as every window is
     opened at start-up, and shows "Not in the tutorial." outside the
     course. The player can close it like any other. It reads
     `Client.GMCPStructs.Tutorial`, so it is current when reopened.
6. **Help:** `help tutorial` lists Alignment, and `help gmcp-tutorial`
   documents the package.

## Constraints

- Never advances the world clock. The lesson changes no alignment,
  loyalty, or standing.
- Progress survives logout, restart, and copyover; the panel is resent on
  login and copyover.
- Nothing is granted, and `tutorial next` and `tutorial skip` still get a
  player out.

## Acceptance criteria

- **Pure:** the stage order (Alignment before Departure); subcommand
  inspections match and are registered by their command; the view for each
  stage matches the terminal checklist.
- **Module:** Alignment passes on the three inspections; the view's
  fields; inactive players have no view.
- **GMCP:** payload fields; sent on change only; `{}` for a player not in
  the course; resent on spawn and request; nothing for a telnet connection
  that hasn't accepted GMCP.
- **Wiring:**
  - Through `plugins.Load`: `company alignment`, `company inspect corvin`,
    and `standing` in the Oath Stone pass Alignment, and the inspect shows
    the refusal.
  - Through `plugins.Load` with the GMCP module: real `GMCPOut` events
    carry the stage as the player advances.
- **Browser check:** a Playwright script drives the real
  `window-tutorial.js` in Chromium. It checks:
  - the stage, goal, and checklist render;
  - markup in a string shows as text;
  - `{}` shows the empty state;
  - it fits at 360px;
  - it's reachable in the accessibility tree.
- **Shipped content:** eight rooms in stage order; Corvin offered at 907
  and refused by a new company's average; help.
- `go test -race ./...`, `make generate`, `make validate`, `make js-lint`
  pass.

## Implementation notes (2026-09-25)

- **`company inspect` weighs recruiter candidates.** 21a's inspect only
  knew summonable templates (`AllowedCompanionMobIDs`). It now also finds
  any recruiter's candidate by id or name, anywhere, so the Oath Stone's
  Corvin can be weighed. Corvin is mapped to the rogue archetype, as every
  candidate must be.
- **The panel moves at once.** The panel's refresh runs before the
  tutorial's gates on the same refresh, so a passed stage would show a round
  late. `internal/tutorial.OnChanged`, fired by the module whenever a
  player's place in the course changes (placement, a pass or waiver, a
  skip, leaving), makes the feed resend at once.
- **A course missing rooms is closed.** A deployment whose `TutorialRooms`
  lacks a stage's room can't run the course: `start` sends new characters
  to the start room, and a player mid-course is let out as skipped.
- **Readable muted text.** The panel's secondary text uses the theme's
  secondary text colour; the "dim" one was too faint on the dark
  background.
- **A steadier fight check.** 27c's wiring test matched "You hit straw
  footman", but attack messages vary ("You punch", "Your fists connect");
  it now matches any line of the player's own naming the foe.
