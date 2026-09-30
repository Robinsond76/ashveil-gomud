# Phase 32g2: Live Battle View — Design

Split from [32g](2026-09-29-phase-32g-company-dock-design.md) (its
decision C), carrying the proposed Phase 31
battle panel (retired Phase 31 proposal; see git history) into the dock's Combat
tab, per the [play-test roadmap](2026-09-28-playtest-feedback-roadmap.md).
Status: **decisions A–G settled by the owner (2026-09-29): "go with
recommendations"; ready for the plan.**

## Goal

While the player's battle runs, the dock's Combat tab shows it: the enemy
group's formation facing the company's, how hurt each enemy looks, who
each fighter is striking, and who has fallen. Out of battle the tab is
32g's Setup view, unchanged.

## Prior-art check

Researched on `bcc7e6c` (2026-09-29).

- **Battles** (`internal/battle`, 29b2/32c/32d): runtime only, keyed by
  the player (`battle.Current(userId)`), holding the room, the group's
  party id, every enemy instance seen (`Enemies`), each enemy's frozen
  29d label (`EnemyNames[id].DisplayName`, "the second cutthroat"), and
  the fight id on the event stream. `battle.Players()` lists the players
  in a battle. A player in a GoMud human party has their own battle;
  overlapping battles in one room share enemy labels.
- **The enemy formation** is re-assembled whenever asked
  (`enemyparty.Parties(room)` → `mobparty.Party.Formation`, a 3×3 grid
  of member keys; `hooks.battleParty(b, parties)` picks the battle's).
  When a front-rank enemy falls, the next re-assembly moves someone up:
  the grid is always the live one, and fallen enemies are no longer in
  it.
- **How hurt an enemy looks** already has a public vocabulary:
  `enemyparty.HealthWord` (unhurt, scratched, wounded, badly wounded,
  near death, down), shown by `scout <group>` and `look <group>` to any
  player. `scout` also marks with `*` the enemies the viewer can reach
  from their cell (`formationcombat.Legal`).
- **Targets:** a fighter's current target is its `Character.Aggro`
  (`UserId` or `MobInstanceId`). Company members are keyed by member key
  (`leader` for the player, `companion:<id>` for a companion); mobs by instance id.
- **Hidden enemies:** `enemyparty.Group.Visible()` leaves out what the
  viewer can't see; `scout` shows only those.
- **The feed:** `modules/gmcp/gmcp.Company.go`'s `companyFeed` keeps
  *extras* (`companyExtra`: `Company.Inventory`, `Company.Camp`) current
  on `companyview.OnRefresh` (every round and after every command, on the
  game loop), each sent only when its JSON changes, forgotten on
  login/copyover/despawn, and re-sent after a new `Company` snapshot
  (the client stores extras under `Company`). A player without a company
  still gets their extras.
- **The event stream** (`internal/combatstream`, 29b) reports fights but
  is never the source of truth for health or aggro. The Phase 31 draft
  wanted the panel fed from it, released with 29f pacing; 29f is not
  built, so there is nothing to pace against yet.
- **The web client:** `window-combat.js` (32g) renders Setup from the
  `Company` snapshot with `textContent` only, reusing `CompanyData` and
  the dock's `uiMenu`. The dock's tab strip already shows an unread
  count on Comm (32g), through the tab group's `setBadge`. `scripts/browser/dock-windows-check.mjs` drives
  the real window scripts in Chromium.
- **Help:** `help webclient` (32g) lists the Combat tab as Setup only;
  `help combat` is the battle hub; `help scout` explains health words and
  reach.

## Scope

1. **Server: a `Company.Battle` message**, a new `companyExtra` in the
   company feed, built on the game loop from `battle.Current` and the live
   room, sent only when it changes, to the player whose battle it is.
2. **Browser: the Battle view** in `window-combat.js`: two facing grids,
   target lines between them, a text list underneath, a live region, and a
   notice on the Combat tab when a battle begins.
3. **Help and tutorial:** `help webclient` and `help combat` describe
   the view; the tutorial's practice-fight lesson points to it.

### The payload

```json
{
  "group": "a band of cutthroats",
  "enemies": [
    {"id": "m:412", "label": "the cutthroat captain", "cell": {"row": 0, "col": 0},
     "health": "wounded", "reach": true, "target": "leader"},
    {"id": "m:413", "label": "the first cutthroat", "cell": {"row": 0, "col": 1},
     "health": "unhurt", "reach": true, "target": "companion:2"}
  ],
  "fallen": [{"id": "m:415", "label": "the slinger"}],
  "company": [
    {"key": "leader", "target": "m:412"},
    {"key": "companion:2", "target": "m:412"}
  ],
  "others": [{"id": "u:7", "name": "Brannoc"}],
  "waiting": ["a pack of grey wolves"]
}
```

- **`enemies`**: the battle group's living, visible members, each with
  its 29d label, its live cell, a health word, whether the player can
  reach it from their cell (as `scout`'s `*`; absent when the player
  isn't placed), and its target (a company member key, an `others` id,
  or absent).
- **`fallen`**: the battle's enemies that have fallen or left, by label,
  in the order they joined the battle (instance order; the order they
  fell would need state the battle doesn't keep). They leave the grid because the live formation
  closes ranks (decision C).
- **`company`**: each company member's target, by member key. Cells,
  health, and fallen state come from the `Company` snapshot and
  `Company.Vitals` the client already holds; they are not repeated.
- **`others`**: a player (or another player's companion) outside the
  company that an enemy is striking, by public name (decision D).
- **`waiting`**: other groups set on the player, waiting their turn, by
  room name, in the order they will come (`battle`'s first-set rounds,
  ties by party id).
- **End of battle:** `{}` clears the view; the tab returns to Setup.
- **As built (review):** in the dark the payload is only
  `{"group":"the enemy","dark":true,"enemies":[]}`, as `scout` refuses; a
  fallen enemy is named unless the view last saw it hidden (a runtime
  memory per player and battle, so a hidden one that dies or slips away
  stays unnamed); a wholly hidden group isn't named in the
  header or the waiting line. The client keeps the last battle across a
  `Company` snapshot (which replaces everything stored under `Company`)
  until the server's re-sent battle arrives.
- **Not in the payload:** exact enemy numbers, levels, or stats.

## Decisions

The owner accepted every recommendation (2026-09-29).

### A. Enemy health: words or numbers

**Decided (owner, 2026-09-29), as recommended: `scout`'s six health words.** They are already public
(anyone can `scout` a group), so the panel reveals nothing new, and the
view and the terminal agree. The grid shows the word and a bar split in
its six steps. Alternatives: exact numbers (leak mob stats that nothing
else shows), or numbers gated on a skill (`peep`-style), which needs its
own design.

### B. Who gets the view

**Decided (owner, 2026-09-29), as recommended: every player in a battle gets their own.** The feed is per
user and each player's battle is their own (29b2), so a GoMud party
member fighting beside the leader sees their battle, with their own
company (usually none). A player not in a battle gets nothing. Phase 31
asked "only the leader"; in Ashveil every player leads their own company,
so that answer is this one.

### C. Fallen enemies: off the grid, listed

**Decided (owner, 2026-09-29), as recommended: the grid is the live formation; the fallen are a line
under it** ("Fallen: the slinger, the first cutthroat"), because the
grid then shows the ranks combat actually uses: when the front falls,
the next enemy steps up. The alternative keeps a fallen enemy greyed in
its last cell, which is more familiar but shows a cell that no longer
exists and needs a remembered layout per battle.

### D. Enemies striking someone outside the company

**Decided (owner, 2026-09-29), as recommended: a small "others" chip beside the company grid**, labelled
with the player's or creature's public name, and in the text list
("the first cutthroat → Brannoc"). Nothing about them beyond the name.
The alternative leaves those lines out, which makes an enemy look idle
when it isn't.

### E. Telling the player a battle began

**Decided (owner, 2026-09-29), as recommended: the Combat tab shows a marker (as Comm's unread count
does) while a battle runs and Combat isn't the active tab; it never
switches tabs on its own.** Switching would take the player out of
whatever they were reading. The alternative, switching to Combat when a
battle begins, could be a Settings switch later.

### F. When the view updates

**Decided (owner, 2026-09-29), as recommended: on the feed's usual beat** (every round and after every
command), sending only when the JSON changes, so a round with no new
target, band, or fall sends nothing. The view may lead the round's
narration by a few lines; 29f pacing, when built, can hold the message
back to its line. The alternative, feeding the view from the event
stream now, duplicates state the stream is not the source of for.

### G. What the Battle view offers

**Decided (owner, 2026-09-29), as recommended: watching only, plus the Setup menu on a company member.**
A battle plays out on its own (32d), and the commands that still work in
one (`strategy`, `formation`, `flee`) are in Setup's member menu and the
terminal. The view keeps the same member menu, so a role or target rule
can be changed mid-battle from the grid, and adds a **Flee** button
(`flee`). No enemy-click commands: there is no command to aim a single
blow. Setup stays one click away under the view.

## Module

- **`modules/gmcp/gmcp.CompanyBattle.go`** (new): `battleExtra()`, the
  payload types, and a builder over a small read interface (the battle,
  the group's members with cells and health, targets, the player's reach)
  so unit tests use fixed inputs. The live reader goes through
  `internal/battle`, `internal/enemyparty`, `internal/mobparty`,
  `internal/formationcombat`, and `internal/company`, as `scout` does;
  all engine packages, so `modules/gmcp` still imports no other module.
  - The battle's group is found as `hooks.battleParty` finds it; the
    matching is moved into `internal/enemyparty` (or `internal/battle`)
    so `hooks` and `gmcp` share it rather than copy it.
- **`modules/gmcp/gmcp.Company.go`**: registers `battleExtra()`.
- **`_datafiles/html/public/static/js/windows/window-combat.js`**: the
  Battle view (grids, SVG target lines, hover/focus highlight, text list,
  live region, Flee), shown while `Company.Battle` has enemies.
- **`_datafiles/html/public/static/js/webclient-core.js`**: no change
  expected; the Combat tab's marker uses the tab group's `setBadge`.
- **`scripts/browser/dock-windows-check.mjs`** and its harness: Battle
  fixtures.
- **Help and tutorial**: below.

## Invariants

- **The clock:** the view reads state; nothing here advances time or
  rounds.
- **Restart and copyover:** no new server-side state. Battles are runtime
  (29b2) and resume as new ones; the feed forgets a user on login and
  copyover, so the view is rebuilt from the next battle's first round.
- **Locks and the game loop:** built on the game loop in the feed's
  refresh. `battle.Current` returns a copy and releases `battle`'s lock
  before any mob or room is read; no two module locks are held at once.
- **Truth:** health and targets are read from the characters, never the
  event stream; the view never shows what `scout` and the terminal
  wouldn't (hidden enemies stay hidden).
- **Security:** labels and names go through `textContent`; the SVG lines
  carry no server strings.

## Acceptance criteria

- **Unit (Go), from fixed inputs:** a group with cells, health words,
  reach, and targets on company members; a fallen enemy moves to
  `fallen` and ranks close; a hidden enemy is left out; an enemy striking
  an outside player fills `others`; a waiting group; an unplaced player
  has no `reach`; no battle builds `{}`; an unchanged battle sends
  nothing, and a health word or target change sends once.
- **Wiring (Go, a real fight through `hooks.DoCombat` on the shipped
  config):**
  - a battle begins: `Company.Battle` carries both sides and targets;
  - a companion's target changes and an enemy falls: it updates;
  - the group is beaten: `{}` is sent;
  - a second player in the room, in their own battle, gets their own
    view, and a player in no battle gets none; a telnet client without
    GMCP gets none;
  - login and copyover mid-battle re-send it.
- **Browser (Playwright, Chromium), real scripts through the harness:**
  - with a fixture battle the grids render facing (both fronts toward the
    middle), each enemy with its label and word, the fallen line, and the
    waiting line;
  - lines follow targets, including to the "others" chip; hovering or
    focusing a fighter highlights its target and its attackers;
  - the text list matches the lines; the live region announces only a
    new target on the player or a fall;
  - `{}` returns to Setup; the Combat tab's marker shows while another
    tab is active and clears on opening it;
  - a member's menu and Flee send the right commands;
  - markup in a label or name renders as text; at a 280 px dock and a
    360 px window there is no horizontal scroll (as built, the grids
    always stack, enemy above company: the dock is one narrow column);
    keyboard focus reaches every fighter.
- **Player help:**
  - `help webclient` gains the Battle view (what the grids, lines, and
    words mean; the marker; Flee);
  - `help combat` points web client players to the Combat tab;
  - `help scout` notes the same words appear in the Battle view;
  - the practice-fight lesson (`modules/tutorial/stages.go`) gains a hint
    for web client players;
  - tests: the pages render through `help`, and
    `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate`, and
  `make js-lint` pass. The independent review is recorded.

## Deferred

- Pacing the view to the narration (29f).
- Casting, wind-ups, statuses, wounds, and guards on the grid (Phase 30
  events), and the battle summary in the view after the end.
- Exact enemy numbers behind a skill (decision A's alternative).
- Switching to the Combat tab on a battle's start (decision E's
  alternative).
