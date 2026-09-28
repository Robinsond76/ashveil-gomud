# Phase 32a: Company Polish — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)). Six small changes to
what a player sees; no new mechanics.

## The owner's notes

1. "I don't like seeing `♥friend` when I type look."
2. "We don't need to see `>>> Tamsin Reed enters from the west.` They're
   part of the company so they'll always be together with the player.
   Another player doesn't need to see that either — one message can
   represent the party entering and leaving the room."
3. "I don't like this message when drinking water: *Ahhhhhh, life giving
   water. Nectar of the gods!* A message giving the status of hydration is
   enough."
4. "I would like a camp and campfire to show in the room when a camp is
   created."
5. "In the tutorial, it's not clear when there is a character in the room
   that can be interacted with, such as Corvin and the other potential
   party members."
6. "The names of the company members in the formation in the UI is hard
   to read."

## Prior-art check

- **`♥friend`:** `characters.Character.SetAdjective("charmed", true)` on
  charm (`internal/characters/character.go`), styled in
  `formattedname.go` (`♥friend` / `♥`, pink). Companions are charmed
  mobs. `IsCharmed()` drives behaviour; the adjective is display only.
  The same tag is hard-coded in `internal/usercommands/party.go` and
  `modules/gmcp/gmcp.Party.go`.
- **Movement lines:** a player's arrival and departure lines are in
  `internal/usercommands/go.go`. Companions follow through
  `modules/follow` (`mob.Command(exit, .25)` on the leader's
  `RoomChange`), so each walks through `internal/mobcommands/go.go` a
  quarter-round later and prints its own "leaves"/"enters from" line to
  the whole room. `company.LeaderAndKeyForInstance(instanceID)` tells
  whether a mob is a companion and whose.
- **Drink:** `buffs/34-hydrated.js` `onStart` prints the flourish.
  `drink` already appends the survival status from `survival.Provision`
  (`provisionSuffix`). The Well Fed buffs (17, 18) print "You feel well
  fed.", which the owner didn't raise; left as is.
- **Camp:** `modules/camping` keeps camps per leader, with a lit-fire
  snapshot (`RoomHasLitFire`) read without the module lock. `look`
  (`internal/usercommands/look.go`) already takes additive lines from
  another package after the description (`weather.SkyLines`).
- **Recruiters:** candidates are config (`Recruiters` in
  `modules/company/files/data-overlays/config.yaml`), a named notice per
  room ("the notched hiring post") with candidates by mob template. No
  mob stands in the room; only `company recruit` lists them.
- **Formation grid:** `.company-formation` in
  `_datafiles/html/public/static/js/windows/window-party.js`,
  `font-size: 0.68em`, ellipsised cells.

## Decisions

### 1. No `♥friend` on company members

A companion's formatted name leaves out the `charmed` adjective wherever
it's rendered (room listing, `party`, GMCP party). Other charmed mobs
(a charm spell's target, a tamed pet) keep it. Behaviour is unchanged:
only the display is filtered. **(recommendation applied)**

### 2. One line for a company on the move

- **The leader** sees no line for their own companions arriving or
  leaving with them.
- **Everyone else** sees one line for the company, for example:
  - leaving: **"Dain leads their company west."**
  - arriving: **"Dain arrives from the east, their company behind."**

  A leader with no companion following keeps today's lines.
- **A companion's own line is dropped** when it moves with its leader:
  it's a companion, and its leader is in the room it's entering (or just
  left the room it's leaving). A companion moving on its own (relocated,
  sent somewhere, the leader gone) keeps its own line.
- The wording is fixed in the plan and set in the voice of 29c;
  neutral pronouns ("their") until 29d gives characters pronouns.

### 3. No drink flourish

Remove the Hydrated buff's message. The status line `drink` already
appends is what's left. The buff still cancels Thirsty.

### 4. The camp in the room

`look` shows one line after the sky lines, to anyone in the room:

- a camp: **"A camp is pitched here: bedrolls around a crackling
  campfire."**
- unlit: **"A camp is pitched here, around a cold fire pit."**

Its text says whose when it isn't the viewer's ("Dain's camp is pitched
here…"). It comes from `modules/camping` through a small read-only
provider in an internal package, the way the lit-fire snapshot is read,
so `look` never waits on the camping lock. It's shown as a room line,
not a fake item: it can't be picked up, and it goes when the camp is
struck. **(recommendation applied)**

### 5. Recruiters shown in the room

In a recruiter room, `look` lists the notice and the candidates the
viewer can still take, after the camp line:

```
On the notched hiring post: Tamsin Reed (free), Brother Oswin (free).
  Type company recruit to see them, or company inspect <name>.
```

- **Per viewer:** candidates already in the viewer's company, and free
  candidates they've already claimed, aren't listed. A candidate their
  company would refuse (Corvin) is listed with "(won't join you)", so
  the Oath Stone lesson reads naturally.
- **`look <candidate>`** in a recruiter room shows the candidate's
  description and points to `company inspect <name>`.
- The tutorial's Company lesson hint says to look at the hiring post.

**Open (for the owner):** should candidates instead be **real NPCs
standing in the room**? That reads better ("Tamsin Reed is here,
sharpening a spear") but means spawning a per-viewer mob that vanishes
once hired, in per-player tutorial copies and shared inns alike. The
recommendation is the listing above now, and NPCs later if the listing
still feels flat in play.

### 6. A readable formation grid

The web client's formation grid uses the panel's normal body size and
text colour. A cell wraps a long name onto two lines rather than cutting
it off, and shows the full name in a tooltip. The leader's cell keeps
its accent. Checked in Chromium at the default and narrow widths, in
both themes.

## Module

- `internal/characters`: skip `charmed` for companions (a predicate set
  by `modules/company` at startup, since `characters` can't import
  `company`).
- `internal/usercommands/go.go`, `internal/mobcommands/go.go`: the
  company line and the suppression, through a companion predicate.
- `internal/usercommands/party.go`, `modules/gmcp/gmcp.Party.go`: the
  tag.
- `_datafiles/world/default/buffs/34-hydrated.js`: the message.
- `modules/camping` plus a provider seam read by `look`.
- `modules/company` (recruiter lines, `look <candidate>`) plus a seam
  read by `look`.
- `window-party.js`: the grid's CSS.

## Invariants

- **The clock:** nothing here advances time.
- **Restart and copyover:** display only. It reads existing durable
  state (camps, the company record, claims); nothing new is stored.
- **Locks:** the camp and recruiter lines are read through lock-free
  snapshots or on the game loop. `look` never takes the camping lock.

## Acceptance criteria

- **Unit:** the companion predicate; the camp line (lit, unlit, whose);
  the recruiter line (claimed, in company, refused, none left).
- **Wiring** (shipped config, real commands):
  - `look` in a room with the viewer's companions shows no `♥friend`; a
    charmed non-companion still shows it;
  - a leader walks with two companions: the leader sees no companion
    line; a second player in the destination sees exactly one arrival
    line, and exactly one departure line in the origin; a companion
    walking alone still has its lines;
  - `drink` water: no flourish, the status line present;
  - `camp` then `look`: the camp line; `camp fire`: lit; `camp break`:
    gone; a second player in the room sees "Dain's camp";
  - the tutorial Muster Yard: the hiring post lists Tamsin and Oswin;
    after `company recruit tamsin`, only Oswin; the Oath Stone lists
    Corvin as won't join; `look tamsin` works;
  - browser (Playwright): the grid's text size and a full-name tooltip.
- **Player help:** `help company` (moving together, recruiters in the
  room), `help camp` (the camp in the room), `help drink` (the status
  line, no flourish) updated; the Company lesson's hint points to the
  hiring post; `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- Recruit candidates as NPCs in the room (open above).
- Pronouns in the company line: 29d.
- The camp in the GMCP room info and the web client's room panel: 32g.
