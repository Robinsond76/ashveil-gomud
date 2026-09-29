# Phase 32g: Web Company Dock — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)) and the owner's
layout request of 2026-09-29. Status: **draft; open decisions A–F await the
owner.**

## The owner's notes

1. (2026-09-28) "A tabbed Company/Comm/Combat panel under the map", with a
   Company tab of Cargo, Status, and Camp, and cargo hover actions "like
   the gear tab".
2. (2026-09-29) "Move character information, vitals and worth from the
   left side of the screen into a tab to the right, and the map from the
   right side to the left, underneath the Time and Date. Pet information
   as well. The gear section we can potentially remove and put it in the
   character tab on the right as well. Organize it as you see fit." Also:
   suggest other tabs worth having.
3. (2026-09-29) "Move room info underneath the Time and Date and the
   Map, to the left." (Settled: the layout below.)

So the dock is no longer "under the map": the map moves left, and the
right column becomes the tabbed dock.

## Prior-art check

Researched on `71498bb4` (2026-09-29).

- **The dock system** (`_datafiles/html/public/static/js/webclient-core.js`):
  two columns, `#dock-left` and `#dock-right`, around the terminal
  (`webclient-pure.html`). A `DockSlot` stacks one panel per
  `VirtualWindow`, each with a title bar, a pop-out button, a close
  button, and a resize handle; panels can be dragged between the columns
  and reordered. `WINDOW_DOCK_DEFAULTS` sets the out-of-the-box side and
  order. `LayoutStore` saves each window's side, height, float geometry,
  and the dock order in `localStorage` (`windowLayout`); Settings has
  **Reset Layout**. There are **no tab groups**: every window is its own
  panel.
- **Today's layout** (`WINDOW_DOCK_DEFAULTS`):
  - left: Time & Date, Vitals, Character, Worth, Gear, Pet;
  - right: Map, Online (off by default), KillStats (off by default),
    Party, Communications, RoomInfo, Tutorial.
- **The windows** (`static/js/windows/`), each registering GMCP handlers
  and keeping its own state in `Client.GMCPStructs`:
  - `window-vitals.js` HP/MP bars (`Char.Vitals`);
  - `window-character.js` already tabbed: Overview, Quests, Skills, Jobs,
    Effects (`Char.Info`, `Char.Stats`, ...);
  - `window-status.js` "Worth": XP bar, gold, bank (`Char.Worth`);
  - `window-gear.js` tabbed Worn/Backpack, with hover tooltips and a
    click menu (`uiMenu`) whose entries send commands by item name
    (`look`, `remove`, `equip`, `eat`, `drink`, `use`, ...). Its
    backpack header still reads "count / —" since 32f dropped GoMud's
    item-count `Max`;
  - `window-pet.js` tabbed Info/Items (`Char.Pets`, GoMud pets; a pet's
    pouch counts in the company load);
  - `window-party.js` the Company section (26b: formation grid, member
    cards with health, needs, chemistry, rescue time) and GoMud's
    Players party (`Company`, `Company.Vitals`, `Party`);
  - `window-comm.js` tabbed chat channels (`Comm`);
  - `window-map.js`, `window-room.js`, `window-gametime.js`,
    `window-tutorial.js`, `window-online.js`, `window-killstats.js`.
- **What the server sends for the company** (`modules/gmcp/gmcp.Company.go`):
  the `Company` snapshot and `Company.Vitals`, built from
  `internal/companyview` on the game loop, sent on change, only to the
  leader. It carries roster, cells, HP, needs, warmth, chemistry, load
  (total, capacity, cargo, companions' gear), activity, rest tier and time
  left, checkpoint. It does **not** carry any item lists, packs, horses,
  cargo contents, camp state (camp here, fire lit), or strategies. 32f
  deferred "capacity in grams, packs, mounts, cargo uses" in GMCP to this
  phase.
- **Commands the dock's buttons would send** (all exist):
  `company inventory`, `company eat|drink|meal`, `cargo put|take <item>`,
  `give <item> <member>`, `mount saddle|unsaddle|release <horse>`,
  `camp`, `camp fire|rest|status|break`, `formation move|swap|clear`,
  `strategy <who> <role>`, `strategy <who> target <rule>`, `scout`.
- **Battles** (`internal/battle`, 29b2/32c/32d) have no GMCP at all. The
  proposed Phase 31 battle panel (`2026-09-26-battle-panel-design.md`)
  specified a `Company.Battle` message and two facing grids; the roadmap
  says 32g's Combat tab replaces Phase 31's separate window.
- **Browser checks:** `scripts/browser/company-panel-check.mjs` and
  `tutorial-panel-check.mjs` drive the real window scripts through a
  harness page in Chromium (Playwright), including a 360 px width, markup
  in a name rendered as text, keyboard focus, and the accessibility tree.
  `_datafiles/html/public/AGENTS.md`: server strings go through
  `textContent`, never `innerHTML`, and every Ashveil window has such a
  check.
- **Help:** there is no player help page for the web client at all.

## The layout

The columns get a job each: **left is the world** (when, where, what's
here), **right is you and yours** (who you are, your company, its fights,
its talk).

```
┌ LEFT: the world ──────┐ ┌──────────────────────┐ ┌ RIGHT: the dock ──────────────┐
│ Time & Date           │ │                      │ │ HP ██████████░░  34/40        │
├───────────────────────┤ │                      │ │ MP ██████░░░░░░  12/20        │
│ Map                   │ │                      │ │ ⚠ Tamsin: Hungry · Load 82%   │
│                       │ │      terminal        │ ├───────────────────────────────┤
│                       │ │                      │ │ Character│Company│Combat│Comm │
├───────────────────────┤ │                      │ ├───────────────────────────────┤
│ Room                  │ │                      │ │ Overview Gear Skills Quests … │
│  exits, who's here    │ │                      │ │                               │
├───────────────────────┤ │                      │ │  (the active tab fills the    │
│ Tutorial (in course)  │ │                      │ │   rest of the column)         │
└───────────────────────┘ └──────────────────────┘ └───────────────────────────────┘
```

**Left column** (unchanged windows, new places):

1. **Time & Date**, as now.
2. **Map**, moved from the right, directly under the time.
3. **Room**, moved from the right (owner, 2026-09-29): it describes the
   room the map marks.
4. **Tutorial**, moved from the right, only while the player is in the
   course (as now: it shows nothing otherwise).

**Right column: the dock**, one panel filling the column:

- **The vitals strip**, always visible above the tabs (decision A): the
  player's HP and MP bars, and one line of warnings drawn from the
  `Company` data already sent (a member's need that warns, a fallen
  member's rescue time, the load at or past its top bands). The line is
  absent when nothing warns.
- **Tabs:** Character, Company, Combat, Comm. Each is described below.
  The active tab and each tab's sub-tab are remembered per browser.
  Every tab can still be popped out into a floating window and docked
  back into its place, as panels can today.

### Character tab

GoMud's Character, Worth, Gear, and Pet windows become one tab with
sub-tabs; the separate Vitals, Worth, Gear, and Pet panels go away.

| Sub-tab | Contents | From |
|---|---|---|
| **Overview** | name, race, archetype, level, alignment, stats; the XP bar, gold and bank | Character Overview + Worth |
| **Gear** | worn slots, then the pack, with today's tooltips and click menus; the header shows the player's carried kg against the company capacity instead of "count / —"; each item's weight in its tooltip | Gear |
| **Skills** | skills, then job proficiencies | Character Skills + Jobs |
| **Quests** | as now | Character Quests |
| **Effects** | buffs and debuffs, as now | Character Effects |
| **Pet** | only while the player has a pet: info, then its items | Pet |

Jobs fold into Skills to keep the strip short; Kill Stats stays an
optional window (off by default) and, when enabled, becomes a **Kills**
sub-tab here.

### Company tab

The Company section leaves the Party window for this tab, with the three
sub-tabs the owner named. A player with no company sees a short note
("You travel alone; `help company` to recruit.") and, in Cargo, their own
share.

- **Status** (26b's section, moved as it is): the formation grid and a
  card per member (health, needs, warmth, chemistry, rescue time), the
  activity and rest line, the load bar, the checkpoint. GoMud's human
  party, when there is one, follows under **Travelling with**, so the
  Party window goes away.
- **Cargo**: the web view of `company inventory` (32f).
  - The load bar and the capacity split (members, horses).
  - A block per member, the player first: weight, the pack that counts,
    worn and carried items, each with a tooltip (weight, uses left,
    description) like the gear window's. A fallen member's gear is shown
    as "with the body".
  - **Horses**: kind, saddle, what it adds, who rides it.
  - **Cargo**: stacks with counts and uses.
  - **Click menus** (the gear window's `uiMenu`), sending real commands:
    - the player's own item: look, equip/eat/drink/use as in Gear,
      **Put in cargo** (`cargo put`), **Give to** each companion present
      (`give <item> <member>`);
    - a cargo stack: **Take** (`cargo take`);
    - a companion's item: look only (moving it is `company give`, still
      deferred by 32f);
    - a horse: **Fit saddle** (each matching saddle in the pack),
      **Unsaddle**, **Release** (asks first; it can't be undone).
  - **Buttons**: **Meal** (`company meal`), **Eat**, **Drink**.
- **Camp**: this camp's state and the rest.
  - Where the camp is (here, elsewhere, none), whether the fire is lit,
    a rest's progress bar and time left, the rest tier and its time left,
    and each member's needs in one compact table.
  - **Buttons**, each shown only when it would work, with the reason in
    its tooltip when greyed: **Make camp** (`camp`), **Light fire**
    (`camp fire`), **Rest** (`camp rest`), **Break camp** (`camp break`),
    **Meal** (`company meal`), **Inn** where there is one (`inn`).

### Combat tab

- **Setup** (this phase): the company's formation grid, larger than
  Status's, and a row per member with their **role** and **target rule**
  (32d). Clicking a member opens a menu to change either (`strategy <who>
  <role>`, `strategy <who> target <rule>`) or move them (`formation move`,
  `formation swap`, `formation clear`). A **Scout** button sends `scout`
  when an enemy group is in the room. Out of battle, this is the whole
  tab.
- **Battle** (decision C: 32g2): during a battle, the enemy group's
  formation faces the company's, with health bands and who targets whom,
  per the Phase 31 design, fed by a new `Company.Battle` message.

### Comm tab

The Communications window as a tab, with its channel tabs as now. A
count of unread messages shows on the Comm tab while another tab is
active.

### Online

Online (off by default, as now) becomes a fifth tab, **Who**, when
enabled in Settings.

## Decisions

Recommendations are marked; each needs the owner's answer before the
plan is written.

### A. Vitals: pinned or in the Character tab

**Recommended: pinned above the tabs.** The owner asked for vitals in the
Character tab; but a battle now plays out on its own while the player
watches, most likely from the Combat or Company tab, and HP and MP hidden
behind the Character tab can't be watched then. The strip costs about
three lines. Worth (XP, gold) goes into Character > Overview as asked.
The alternative is vitals inside Character > Overview only.

### B. How tabs are built: a generic tab group in the dock

**Recommended:** the dock core learns **tab groups**. A `VirtualWindow`
gains `tabGroup: 'dock'`; a `DockSlot` renders every docked member of a
group as one panel with a tab strip, showing one member's content at a
time. Each window keeps its own GMCP handling and DOM exactly as now; the
group only shows and hides it. Popping a tab out floats that window (the
existing `undock`); docking it returns it to its tab. `LayoutStore`
records each window's group and the active tab.

- The Character tab's sub-tabs are the Character window's own tab strip,
  extended: `window-status.js` (Worth), `window-gear.js`, `window-pet.js`,
  and `window-killstats.js` stop registering their own windows and
  instead render into a container the Character window hands them,
  keeping their GMCP handlers. `window-vitals.js` renders into the strip.
- Company's content moves out of `window-party.js` into a new
  `window-company.js` (Status, Cargo, Camp); `window-party.js` is
  retired, its Players section moving into Status.
- A new `window-combat.js` holds the Combat tab.
- The alternative, one hand-built `window-dock.js` that owns every tab,
  is simpler at first but hard-codes the tab list and loses per-tab pop
  out; the tab group costs about 200 lines in `webclient-core.js`.

### C. Split the live battle view into 32g2

**Recommended:** 32g ships the layout, Character, Company (all three
sub-tabs), Combat's **Setup**, and Comm. **32g2** adds Combat's
**Battle** view and its `Company.Battle` message: new server-side data
from `internal/battle` every round, target arrows, health bands, and the
Phase 31 open questions (bands or numbers; whether non-leaders see it),
which deserve their own design. The alternative is one larger phase.

### D. Existing saved layouts: reset once

**Recommended:** bump a layout version in `LayoutStore`; a layout saved
before 32g is discarded once and the new defaults apply, since it names
windows that no longer exist (Vitals, Worth, Gear, Pet, Party) and would
put the map on the right. A terminal line says so the first time: "The
web client's layout has changed; Settings → Reset Layout restores it at
any time." The alternative is to migrate saved sides window by window,
which keeps a customised layout but mostly produces a half-old layout.

### E. New GMCP for the Cargo and Camp sub-tabs and strategies

**Recommended:** two new messages and two new fields, all built on the
game loop from the same sources as today's text commands, sent on change,
only to the leader (a player with no company gets their own share, like
the load), as `Company` is:

- **`Company.Inventory`**: per member (key, name, fallen, weight in
  grams, pack name and bonus, worn items, carried items), horses (id,
  kind, saddle, capacity added, rider key), cargo stacks, and the
  capacity split. Each item: its name as a command would name it, a
  display name, weight in grams, count, uses and max uses, type and
  subtype (for the menus). Built from what `company inventory` reads.
- **`Company.Camp`**: camp here/elsewhere/none, the camp's room name,
  fire lit, resting with percent and seconds left, and whether an inn is
  here. `{}` with no camp and no inn.
- **`Company` members gain `strategy {role, target}`**, from 32d's
  strategy store.
- **`Char.Inventory` gains `capacity_g` and `load_g`** for the Gear
  header, restoring what 32f dropped.

Items are sent as data, never text with markup, and the client sets
every string with `textContent`. The alternative, the client sending
`company inventory` and parsing its text, is fragile and prints to the
terminal.

### F. Tab and sub-tab names

**Recommended:** Character (Overview, Gear, Skills, Quests, Effects,
Pet, Kills); Company (Status, Cargo, Camp); Combat (Setup; Battle in
32g2); Comm; Who. "Cargo" is the owner's name, although it shows every
member's gear too; "Packs" or "Inventory" are the alternatives.

## Module

- **`_datafiles/html/public/static/js/webclient-core.js`**: tab groups
  (`DockSlot`, `VirtualWindow`, `LayoutStore`), the layout version, the
  new `WINDOW_DOCK_DEFAULTS`.
- **`_datafiles/html/public/static/js/windows/`**:
  - `window-character.js` hosts the sub-tabs; `window-status.js`,
    `window-gear.js`, `window-pet.js`, `window-killstats.js` render into
    it; `window-vitals.js` renders into the strip.
  - New `window-company.js` (Status, Cargo, Camp) and
    `window-combat.js` (Setup); `window-party.js` retired.
  - `window-comm.js` joins the tab group, with its unread count.
  - `window-map.js`, `window-room.js`, `window-tutorial.js` change only
    their default side.
- **`_datafiles/html/public/webclient-pure.html`**: script tags, the
  strip's container, any shared tab styles.
- **`modules/gmcp`**: `Company.Inventory`, `Company.Camp`, the
  `strategy` field, `Char.Inventory`'s capacity. Sources are read
  through providers the owning modules register (`modules/company`,
  `modules/encumbrance`, `modules/mount`, `modules/camping`,
  `modules/strategy`), the way `Company` reads `companyview`, so
  `modules/gmcp` doesn't import them.
- **`scripts/browser/`**: a dock check and harness replacing
  `company-panel-check.mjs`.
- **Help and tutorial**: below.

## Invariants

- **The clock:** the dock only reads state and sends ordinary commands;
  nothing here advances time.
- **Restart and copyover:** no new server-side state. Every new payload
  is computed, and the feed forgets a user on login and copyover so the
  full set is re-sent, as `Company` does. The layout lives in the
  browser.
- **Locks and the game loop:** every payload is built on the game loop,
  as `Company` is; providers read their module's state under that
  module's own lock and never hold two at once. Commands from buttons go
  through the ordinary input path, so every rule a typed command obeys
  (resting blocks exits, the capacity refusals, battles ignoring input)
  still applies.
- **Security:** every server string is set with `textContent`; commands
  built from item names use the name the server sent for commands,
  never a display string.

## Acceptance criteria

- **Unit (Go):** each new payload from fixed inputs: a member's worn and
  carried items with weights and uses; a fallen member; horses with
  saddles and riders; cargo stacks with uses; camp here, elsewhere, none,
  resting; an inn; a player with no company; strategies on members.
  Change detection: an unchanged state sends nothing; a used waterskin
  sends `Company.Inventory` only.
- **Wiring (Go, real commands on the shipped config):**
  - `cargo put` then `cargo take` a waterskin: `Company.Inventory` is sent
    after each, with the stack and then the pack showing its uses;
  - `give satchel tamsin`: her pack and the capacity split change;
  - `mount saddle`: the horse's saddle and capacity change;
  - `camp`, `camp fire`, `camp rest`: `Company.Camp` follows each, and
    the rest's end;
  - `strategy tamsin healer`: the `Company` snapshot carries it;
  - login and copyover re-send every message; a non-leader and a telnet
    client without GMCP get none.
- **Browser (Playwright, Chromium), through a harness running the real
  scripts:**
  - the default layout: Time & Date, Map, Room on the left; the strip
    and the four tabs on the right; a pre-32g saved layout is replaced
    once, with the notice;
  - each tab and sub-tab renders from fixture payloads; the active tab
    and sub-tab survive a reload; popping a tab out and docking it back
    restores its place;
  - Cargo's menus send the right commands for the player's item, a
    cargo stack, a companion's item, and a horse; Release asks first;
  - Camp's buttons show only when they'd work;
  - Combat Setup's menus send `strategy` and `formation` commands;
  - Comm's unread count appears and clears;
  - markup in a member's, item's, or pet's name renders as text;
  - at a 280 px dock and a 360 px window the tabs stay usable, with no
    horizontal page scroll; keyboard focus reaches every tab and menu;
    the accessibility tree names the tabs and panels.
- **Player help:**
  - a new **`help webclient`** page (aliases `web client`, `dock`,
    `panels`): the layout, each tab and sub-tab, what its buttons send,
    and Reset Layout;
  - `help company`, `help cargo`, `help company inventory`, `help camp`,
    `help strategy`, and `help formation` each gain a line pointing to
    their tab;
  - listed in `keywords.yaml`; linked from `help company`;
  - the Character lesson's hint mentions the dock (web client players);
  - tests: the page renders through `help`, and
    `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate`, and
  `make js-lint` pass. The independent review is recorded.

## Deferred

- **32g2:** Combat's Battle view and `Company.Battle` (decision C).
- Moving a companion's items from the dock (`company give`, deferred by
  32f).
- Drag and drop between members and cargo; menus cover it.
- A phone layout that collapses both columns into one; this phase keeps
  the two columns usable at 360 px but doesn't redesign for phones.
