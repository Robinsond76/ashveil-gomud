# Phase 32g: Web Company Dock — Plan

Design: [32g design](../specs/2026-09-29-phase-32g-company-dock-design.md).
Decisions A–F were answered by the owner on 2026-09-29 and are recorded in
the design (vitals pinned for every company member; tab groups; the battle
view split into 32g2; saved layouts reset once; the new GMCP; the Company
sub-tab named **Inventory**).

Server tasks (1–6) come first, so the browser tasks render real payload
shapes. Browser tasks run their Playwright check through a harness page
that loads the real scripts, as `scripts/browser/company-panel-check.mjs`
does.

## Task 1: Members' mana (`internal/companyview`, `modules/gmcp`)

- [x] Tests first:
  - `internal/companyview`: the leader's `MP`/`MPMax` from their
    character; an out companion's from its live mob; an awaiting or
    fallen companion has `HasMP` false; a companion with no mana
    (`ManaMax` 0) has `HasMP` false.
  - `modules/gmcp` (`gmcp_company_test.go`): `vitalsOf` carries `mp` and
    `mp_max` when known and omits them otherwise; a mana change alone
    sends `Company.Vitals`, not the full `Company`.
- [x] `Member.HasMP`, `MP`, `MPMax`, read where HP is read
  (`summary.go`); `companyVitals.MP`, `MPMax`.

## Task 2: Strategies in the snapshot (`modules/strategy`, `modules/gmcp`)

- [x] Tests first:
  - `modules/gmcp`: a member's `strategy {role, target}` from a stub
    provider; absent when the provider has none; a strategy change sends
    the full `Company`.
  - Wiring (`modules/strategy`): `strategy tamsin healer` then
    `strategy tamsin target leader` through the real command; the
    `Company` payload built afterwards carries both.
- [x] `companyview.Member.Strategy`, read through
  `enemyparty.MemberStrategy` (the resolution a battle aims by, over
  `internal/strategy`'s existing seam; no new seam needed);
  `companyMember.Strategy`.

## Task 3: `Company.Inventory` (`modules/company`, `modules/gmcp`)

- [x] Tests first:
  - `modules/company`: a pure `inventoryData(user)` returns the members
    (the player first; key, name, fallen, weight, pack and bonus, worn
    and carried items), horses (id, kind, saddle, capacity added, rider
    key), cargo stacks, and the capacity split; each item has its
    command name, display name, weight, count, uses and max uses, type
    and subtype. A fallen member's gear is "with the body"; a player with
    no company gets their own share.
  - As built: the data is `CompanyModule.CompanyInventory` (companions,
    read only: the live mob's gear by `Snapshot`, never
    `refreshSnapshot`, which writes the record; the feed asks every
    round) over `company.InventoryMemberOf`; the text `company inventory`
    keeps its own builder and tests, since it refreshes as it reads.
  - `modules/gmcp`: the payload marshals names as data; an unchanged
    inventory sends nothing; one used waterskin sends `Company.Inventory`
    only.
- [x] `inventoryData`; an `InventoryProvider` seam; the
  `Company.Inventory` payload in the company feed, change-detected on
  its own.
- [x] Confirm how `cargo take`, `give`, and `mount saddle` resolve two
  items of the same name. Found: `cargo put`, `give`, `mount saddle`,
  `eat`, `drink`, `equip` all use `FindInBackpack` → `items.FindMatchIn`,
  which takes `!<id>:<uuid>` for exactly one item (`Item.ShorthandId`,
  as the gear window's GMCP `id`); `cargo take` matches stacks by
  `!<id>`. Each item's `ref` is that form (tested).
- Horses carry `rides` (a saddled riding horse), not a rider key: the
  mount module doesn't expose which member rides.

## Task 4: `Company.Camp` (`modules/camping`, `modules/gmcp`)

- [x] Tests first:
  - `modules/camping`: a pure `campData(leader, room)`: camp here,
    elsewhere (with its room name), or none; fire lit; resting with
    percent and seconds left; `can_camp` in an eligible room with no
    camp; an inn here.
  - `modules/gmcp`: nothing sent without a provider (as built: the
    full object is always sent, since `can_camp` drives the Make camp
    button); change detection.
- [x] As built: `CampingModule.CampStateOf(leader, room, tags)` (read
  under the camping lock, the room title looked up after it), the
  `camping.CampStateProvider` seam; `campExtra` in the company feed.

## Task 5: `Char.Inventory` capacity (`modules/gmcp`)

- [x] Tests first: `Char.Inventory.Backpack` carries `capacity_g` and
  `load_g` from the same load `cargo` reads; a player with no company
  gets their own share.
- [x] As built: `weight_g` (the player's own gear), `load_g`, and
  `capacity_g` in the backpack summary (`backpackSummary`). `Char.Inventory`
  is sent only when the player's items change, so the Gear header takes
  the company figures from `Company.Inventory` when it has them.

## Task 6: Feed wiring (`modules/gmcp`, the modules above)

- [x] Wiring tests first (real commands, shipped config), capturing
  what the feed sends. As built: `modules/tutorial/wiring_dock_test.go`
  (the package that loads every shipped module), with the payload's own
  references typed into the real commands (checked to fail with a broken
  reference):
  - `cargo put` then `cargo take` a 3-use waterskin: `Company.Inventory`
    after each, the stack then the pack showing 3 uses;
  - `give <ref> tamsin` (a ration; no satchel in the test world): hers,
    read from her live mob;
  - horses: unit-tested in `modules/gmcp` (the mount module isn't in
    that harness); `mount saddle` already resolves `!<id>:<uuid>` through
    `FindInBackpack`;
  - `camp`, `camp fire`, `camp rest`, and the rest's end: `Company.Camp`
    follows each;
  - a companion's mana: unit-tested (`modules/company`, the live mob's
    mana; `modules/gmcp`, a mana change sends `Company.Vitals`);
  - login (a `PlayerSpawn`, as login and copyover raise) re-sends
    `Company`, `Company.Inventory`, `Company.Camp`; a connection without
    GMCP gets none (unit-tested; each player only ever gets their own
    company's messages, so there is no "non-leader" case).
- [x] `forget` clears every message's last-sent state; purge drops them
  (`modules/gmcp/purge.go`).

## Task 7: Dock tab groups (`webclient-core.js`)

- [x] Browser check first (`scripts/browser/dock-check.mjs`,
  `dock-harness.html`): four stub windows with `tabGroup: 'dock'` render
  as one panel with a tab strip; the active tab survives a reload; popping
  a tab out floats it and docking it returns it to its tab; closing the
  last tab removes the panel; a window without `tabGroup` stacks as today.
- [x] `VirtualWindow` option `tabGroup`; `DockSlot` renders a group as
  one panel (tablist with `role="tab"`/`tabpanel`, arrow-key moves);
  `LayoutStore` saves group and active tab.
- [x] Layout version: a layout saved before 32g is discarded once, and
  the terminal shows the notice; check it in the harness.
- [x] `WINDOW_DOCK_DEFAULTS`: left Time & Date, Map, RoomInfo, Tutorial;
  right the dock group.

## Task 8: The vitals strip (`window-vitals.js`)

- [x] Browser check first: the player's HP and MP bars with numbers; a
  row per companion (HP; MP only with mana; awaiting dimmed with "not
  with you"; fallen with its rescue time, no bars); the warnings line
  appears and disappears; markup in a name renders as text; each row's
  accessible name carries its numbers.
- [x] `window-vitals.js` renders the strip above the dock group, reading
  `Char.Vitals`, `Company`, and `Company.Vitals`.

## Task 9: Character tab (`window-character.js` and friends)

- [x] Browser check first: sub-tabs Overview (with XP, gold, bank),
  Gear (Worn then pack; header kg of capacity; weight in tooltips),
  Skills (with Jobs), Quests, Effects; Pet only with a pet; the gear
  menus send the same commands as before. As built: Kill Stats, when
  enabled in Settings, is a dock tab of its own (Kills), like Who, not a
  Character sub-tab: an optional window keeps its Settings switch that
  way.
- [x] `window-character.js` joins the dock group and hosts the
  sub-tabs; `window-status.js`, `window-gear.js`, `window-pet.js`,
  `window-killstats.js` render into containers it hands them (keeping
  their GMCP handlers) and stop registering their own windows.

## Task 10: Company tab (`window-company.js`; retire `window-party.js`)

- [x] Browser check first (replaces `company-panel-check.mjs`):
  - Status: 26b's formation grid and cards as before, and **Travelling
    with** for a human party;
  - Inventory: members, horses, cargo; tooltips; menus send `cargo put`,
    `give <ref> "<member>"` (one entry per companion out with you; `give`
    takes the quoted last argument as the receiver), `cargo take <ref>`,
    `mount saddle #<id> <ref>`, `mount unsaddle #<id>`, and `mount release
    #<id>` only after a confirm (`uiMenu` entries gain `confirm`); a
    companion's item has a tooltip and no menu (no command acts on it); Meal/Eat/Drink send
    `company meal|eat|drink`;
  - Camp: each button shown only when it would work (Make camp with
    `can_camp`, Light fire with a cold camp here, Rest with a lit fire,
    Break camp with a camp here, Inn with an inn); the rest bar;
  - no company: the note, and Inventory with the player's share;
  - markup in member and item names renders as text; 280 px and 360 px
    widths; keyboard focus reaches every menu.
- [x] `window-company.js`; remove `window-party.js` and its script tag.

## Task 11: Combat tab, Setup (`window-combat.js`)

- [x] Browser check first: the formation grid and a row per member with
  role and target rule; the member menu sends `strategy <who> <role>`,
  `strategy <who> target <rule>`, `formation move|swap|clear` (`<who>`:
  `me`, or `#<id>` for a companion; no assist for the player); Scout
  sends `scout` (shown when `Room.Info` lists an NPC in a group or an
  aggressive one, else hidden); markup in names renders as text.
- [x] `window-combat.js` in the dock group.

## Task 12: Comm and Who tabs (`window-comm.js`, `window-online.js`)

- [x] Browser check first: a message while another tab is active shows
  an unread count on Comm; opening Comm clears it; Who appears as a tab
  only when Online is enabled in Settings.
- [x] Both join the dock group (Kill Stats too, as Kills). As built:
  docked, each Comm channel keeps its newest 300 messages (the old trim
  only ran for a floating window).

## Task 13: Player help and tutorial

- [x] New `help webclient` (`_datafiles/world/default/templates/help/webclient.template`):
  the two columns, the strip, each tab and sub-tab, what the buttons
  send, and Reset Layout; `help-aliases` `web client`, `web-client`,
  `dock`, `panels`, `tabs`, `layout`; listed in `keywords.yaml` (under
  configuration); linked from `help company`.
- [x] One line pointing to the tab in `help company`, `help cargo`,
  `help company-inventory`, `help camp`, `help strategy`,
  `help formation`.
- [x] The Character lesson's hint mentions the dock
  (`modules/tutorial/stages.go`).
- [x] Tests: the page renders through `help` (pattern:
  `internal/usercommands/help_combat_test.go`; as built:
  `help_webclient_test.go`);
  `TestTutorialHelpPointersExist` passes.

## Task 14: Docs, review, verification

- [ ] `_datafiles/html/public/AGENTS.md`: one bullet on tab groups and
  the dock (new windows join `tabGroup: 'dock'` or the left column).
- [ ] Independent review of `git diff master..HEAD`; verify each finding,
  fix real ones with regression tests.
- [ ] Full verification once: `go test -race ./...`, `make generate`,
  `make validate`, `make js-lint`, every `scripts/browser/` check.
- [ ] `docs/PROJECT_STATUS.md`: the 32g row, a work-log entry with
  **Review:**, 32g2 named as next; the roadmap row updated.
