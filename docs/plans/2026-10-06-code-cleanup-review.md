# Code cleanup and refactoring review (2026-10-06)

The owner asked for a pass over the code for cleanup and refactoring
opportunities after the day's phases landed (company, camp, survival
phases 50-56, classes and elites, web client). Two surveys ran, one over
the Ashveil Go modules and one over the web client, plus `staticcheck`
and `deadcode -test`. This document ranks what they found. The first
section is what this pass carried out; the rest are proposals for later
threads, with the reason each was not done here.

Rules this pass followed: avoid the areas two in-flight builds own (39i2
elites, `modules/archetype`; 36d relics and gear sets, loot and items);
never skip or disable a test; add a regression test for each bug fix.

## Carried out in this pass

Ranked by value. Each entry names the decision and its reason.

1. **One config coercion package, `internal/modconfig`.** Thirteen
   modules each carried `configInt`, `configFloat`, `configString`, a
   map-lowercasing helper (`stringMap` or `lowerKeys`) and in three cases
   a duration parser, about 550 lines in all, and the copies had drifted:
   market and standing rejected fractional floats, the rest truncated
   them; only archetype accepted `uint64`; the duration parsers accepted
   different formats (the camp reward cooldown ignored a numeric value).
   Decision: one package with `Int`, `IntOr`, `Float`, `FloatOr`, `String`,
   `Strings`, `Bool`, `Map` and `Duration`, and the strict rule (a fractional value
   is rejected so a mistyped number falls back to the module default rather
   than silently becoming its floor). Reason: the shipped `config.yaml`
   only uses integers, so no behaviour changes, and the strict rule is the
   safer one to converge on. `modules/archetype` keeps its copies until
   39i2 merges.
2. **Camp room tags match the inn's rule.** Making camp compared tags
   exactly (`t == tag`) while the inn used `room.HasTag`, which is
   case-insensitive and includes mutator tags; the Camp tab's `CanCamp`
   flag used a third exact-match copy. A room tagged `Camping` worked for
   one path and not another. Decision: `HasTag` and `EqualFold` everywhere,
   callers pass `GetTags()`. Regression test added.
3. **Member keys parse one way.** Four re-implementations of
   `CompanionIDFromMemberKey`, two using `TrimPrefix` alone, which accepted
   a bare `"5"` as companion 5. All route through the canonical helper now
   (`internal/survival` re-exports it). Regression tests added.
4. **Inn and physician payments refresh Worth.** Shops, stables and
   recruiting queue `EquipmentChange{GoldChange}`, which is what refreshes
   the web client's Worth panel. The inn and the physician only changed
   the number, so the panel went stale. Both queue the event now; the inn
   only after its save succeeds, so a refund sends nothing. Tests added.
5. **Purging a leader saves when only a reward cooldown remains.** The
   camping purge deleted `lastRewards` but did not count it as a change,
   so the persisted cooldown returned on restart. Test added.
6. **Zero vitals and worth reach the client.** `Char.Vitals` and
   `Char.Worth` used `omitempty` on ints; a player at 0 SP or with 0 gold
   lost the field, the client replaces each namespace wholesale, and the
   Vitals bar read "undefined / 50" while Worth showed a dash. Tags fixed,
   test added. The company card also had no branch for the `fled` status
   the server sends (a fled member showed a bare health bar), and the
   gametime countdown could read "1h 60m".
7. **Shared needs line.** The four-way copy of the company needs status
   line is now `survival.NeedsLine`.
8. **Dead code removed.** Six unreferenced Go helpers (`LevelLine`,
   `company.AddedGrams`, `EquipmentCapacityDelta`, `weather.RenderLine`,
   `sharedPackBonus`, `exertionZero`), two unused JS functions
   (`onSettingsChanged`, `rawPosToX`), the WinBox-era `.vw-max/.vw-full/
   .vw-min` rules, `.cw-tt-*` tooltip CSS nothing renders, and 13 theme
   tokens defined in all 20 themes and read nowhere (260 lines).
   Two Ashveil entries remain for their owners: `survival.AilmentKinds`
   (phase 55, nothing calls it yet) and the `world.command` helper in
   `modules/gathering/gathering_test.go`.
   `deadcode` also lists about 90 unreachable upstream GoMud functions
   (`internal/term`, `internal/markdown`, `internal/users/storage.go`,
   and so on); they were left alone so upstream merges stay clean.

## Web client follow-up (2026-10-07)

Web client proposals 1, 4 and 5 below were carried out in a second pass;
each decision has its reason.

- **GMCP dispatch (proposal 1), confirmed and fixed.** The loop had no
  `break`, so a handler on both `Char.Kills` and `Char` ran twice for
  `Char.Kills`, `'*'` handlers ran once per namespace level, and Online's
  two `Game` registrations updated twice. `scripts/browser/
  gmcp-dispatch-check.mjs` fails five checks on the old dispatch and
  passes on the new one. Decision: one rule, "each registered window is
  called once per message, for its own namespace or any parent, most
  specific first, then `'*'`". Reason: Company, Vitals and the map rely on
  a parent name (`Party`, `Company`) receiving its children, so "first
  level only" would break them. `handleGMCP` dedupes by registration
  (`owner`), runs `'*'` once outside the level loop and the header comment
  now says so. The redundant registrations are gone: the map's
  `Party.Vitals` (covered by `Party`) and Online's second registration
  (the window registers with no namespaces; one windowless handler keeps
  the hidden window current). Decision: KillStats keeps `Char.Kills` and
  `Char` but returns early for other `Char.*` feeds, since `Char` is how
  the first full payload arrives and `Char.Vitals` arrives constantly.
- **Tooltips (proposal 4), partly.** `Client.tooltip(id)` owns creation,
  show, hide delay and placement (beside an anchor or the pointer); the
  Character, Gear, Map and Gametime tooltips use it with their own ids and
  CSS, so nothing looks different. Decision: no `innerHTML` escaping and no
  merged `.ui-tooltip` class here. Reason: those strings carry intentional
  markup (item labels, ansi-converted names), so escaping needs a per-caller
  audit and merging the CSS shifts four themed looks; both stay proposals.
- **Tab switchers (proposal 5), partly.** Gear, KillStats and Pet use
  `Client.tabs(root, {button, panel})`. Decision: Character and Company are
  left as they are (they persist the tab and Company hides panels rather
  than toggling `.active`), and arrow-key support is not added. Reason:
  both would change visible behavior; this pass was no-change.

## Proposals: Go

Ranked by value. Risk is the reviewer's estimate of what could change for
a player.

1. **Shared module persistence (medium risk).** Eleven modules define
   the same `Store` interface, `pluginStore`, `persistenceAvailable`,
   `save`/`saveLocked`/`load` and `decodeRegistry` scaffolding (about 250
   lines). A generic `modstore.Store[R]{Name, Decode, New}` plus
   `Available(loadErr, store)` would hold it once. Drift to settle first:
   only market treats an empty file as corrupt (`ErrCorruptStore`); the
   others decode empty bytes as an empty registry, the truncated-write
   hazard market guards against. Archetype saves a `Clone()`; camping
   passes live maps. Not done here because it touches every module's load
   path and archetype is in flight.
2. **One "live companion" roster helper (medium risk).** The pattern
   `survival.CurrentRoster` → `CompanionIDFromMemberKey` → `InstanceFor`
   → `mobs.GetInstance` is written five ways with different liveness
   rules: exposure and walking skip `Dead`; camping tiers skip `Dead ||
   Away`; archetype filters on room and `IsDisabled`; camp specialists
   use `CompanionsWithLeader` plus `Health < 1`. Inside `modules/company`
   the "present and usable" predicate appears in `members.go`,
   `equipment.go`, `train.go`, `inventory_data.go` and `morale.go` with
   different subsets of Dead, PendingReturn, IsLive, IsAttached,
   WithLeader and CharmedByOther. Some differences cite phases 25b and
   33h3, so each caller needs an explicit option rather than a silent
   merge. Suggested shape: `company.LiveCompanions(leader, filter)` in
   `internal/company` and one `m.present(c, inst)` in the module.
3. **One member-selector matcher (medium risk).** `resolveMemberKey`,
   `resolveCompanion`, `ambiguousCompanion`, `nameMatches`, `lostMatch`
   and survival's `resolveMember`/`matchCompanionName` each parse `#N`,
   `me|self|leader` and partial names with different rules: survival
   treats an empty selector as the leader while company errors; survival
   takes the first exact match while company calls duplicates ambiguous;
   `nameMatches` counts exact hits as substring hits. The web client's
   `memberSelector` emits `leader`, which `strategy.resolve` does not
   accept (it takes `me/self/you/myself`). A single
   `MatchByName[T](items, sel, name, id)` would fix the drift but changes
   what players can type, so it needs its own help and test pass.
4. **Scheduler shared between camping and expedition (medium to high
   risk).** The `Timer`, `Scheduler`, `realScheduler`, generation-counter
   `scheduleLocked` and `onTimer` code is line for line the same in both
   modules, but expedition marshals callbacks onto the game loop
   (`travelTimerDue`) while camping runs them on the timer goroutine and
   defers world work through `restedPending`/`campRewards` flags. A shared
   `modtimer` package with one loop-marshalling scheduler is the right
   end state; the threading change is why it is not done here.
5. **Command handler boilerplate (low to medium risk).** Most handlers
   split args with `strings.Fields(strings.ToLower(strings.TrimSpace(rest)))`,
   some drop the trim, and only `company` uses
   `util.SplitButRespectQuotes`, so quoted names work in `company` but not
   in `formation`, `inn` or `trap`. `companyUsage` lists `company gear`
   and omits `equipment`, `treasury` and `compare`, which `equipmentUsage`
   does list. A `cmdargs.Parse(rest)` and a table-driven subcommand
   dispatcher with min-args and usage would remove about 150 lines.
   `equipment.go` is in 36d's area.
6. **Pluralization and time text (low risk, text changes).** Six plural
   helpers (`plural`, `count`, `unit`, `pointsLabel`, and inline
   `companion(s)`) and four time formatters (`"%dh %dm"`, `"2 hours 30
   minutes"`, `"about N minutes"`, raw `Duration.Round(time.Second)` in six
   places). `util.Plural(n, one[, many])` and `util.Countdown(d)` next to
   `article.go`. Tests assert today's text, so each change is visible.
7. **GMCP feed boilerplate (low risk).** `forget`/`prune` (drop users
   who logged off) is repeated in `gmcp.Company.go`,
   `gmcp.CompanyBattle.go` and `gmcp.Tutorial.go`; a generic
   `userCache[T]` with `prune` covers all three. Fields Go sends that no
   JS reads: the whole `Char.Enemies` namespace (built on every full Char
   payload), `Company.checkpoint`, `Company.load.companion_g`/`cargo_g`,
   `Company.tactics.patch`, `Company.Inventory.companions_known`,
   `horses[].kind`, `Company.Battle.outlook.close`. `patch` and
   `companions_known` look meant for the UI; the rest can be trimmed
   once Mudlet scripts are checked.
8. **Gold charging helper (low risk).** With this pass every deduction
   queues the event, but the two messages `"%s asks %d gold, and you have
   %d."` and `"You pay %d gold. %s"` are copied between `roster.go` and
   `recruit.go`, and market wraps prices in `<ansi fg="gold">` while
   camping and company do not. A `users.ChargeGold(user, price, reason)`
   that deducts, queues and saves would hold the rule once.
9. **Small text duplicates (low risk).** `roomTitle` and `sendToLeader`
   are identical in camping and expedition; `Tell` in `chemistry.go` and
   `alignment.go`; `leaderDisplayName` and `leaderName` in company and
   survival; the companion fallback name is `"#3"` in two places and
   `"companion #3"` in sharpen. `kg` formatting is inlined nine times
   across `inventory.go`, `equipment.go`, `status.ashveil.go`,
   encumbrance and mount beside a `kg()` helper.
10. **Large files mixing concerns (low risk, conflicts with in-flight
    work).** `expedition.go` (1405 lines: store, scheduler, adapters,
    config, departure math, state machine, rendering, command),
    `camping.go` (1234), `tutorial.go` (1100), `company.go` (1084) and
    `archetype.go` (1057). Splitting by store / config / state machine /
    render / command is mechanical; do it per module when nothing else
    has the file open.
11. **RNG and markup stripping (low value).** Four random strategies
    (`math/rand` v1 seeded, `rand.Uint64`, `math/rand/v2`, `util.Rand`)
    and three tag-stripping regexes. Converge on `util.Rand` when a file
    is next touched.
12. **staticcheck residue.** 200 findings, mostly upstream GoMud: 69
    capitalised error strings, 30 redundant `strings.HasPrefix` guards,
    18 deprecated `strings.Title` and `rand.Seed` calls, a few Yoda
    conditions and append loops in `modules/gmcp`. Ashveil-owned ones are
    cosmetic (`modules/company/equipment.go` error strings, two unused
    test fields in `modules/tutorial`). Worth a sweep on a quiet day, not
    a phase.

## Proposals: web client

1. **GMCP dispatch fires handlers at every namespace level (medium
   risk).** The comment at `webclient-core.js` says dispatch calls the
   first level that has handlers; the loop has no `break`, so it calls
   every level. KillStats registers `['Char.Kills', 'Char']` and updates
   twice per message (and rebuilds on every `Char.Vitals`); the map
   registers `'Party'` and `'Party.Vitals'`; Online registers the same
   `update` twice. The `'*'` branch sits inside the per-level loop and
   would fire N times. Company relies on the all-levels behaviour to get
   `Party.Vitals` through `'Party'`, so pick one behaviour, fix the
   comment, move `'*'` out of the loop and drop the redundant
   registrations together.
2. **Health thresholds and status words differ per window (low risk,
   visible).** Vitals colours `>60` high and `>25` mid with `Math.floor`;
   the company card uses `>=60` and `>=30` with `Math.round`; combat
   colours by `HealthWord` buckets (75/50/25) and paints "scratched"
   mid. Status text: company "Fallen: 3m to raise", vitals "fallen, 3m
   to raise"; company "Away: rejoins when you return", vitals "not with
   you". `CompanyData.hpLevel(pct)` and `CompanyData.statusText(m, live)`
   would unify them.
3. **Item context menus written three times (low to medium risk).**
   Gear uses `item.id`, company's `yourItemMenu` uses `i.ref` and has no
   use/throw/read, `sharedCargoMenu` capitalises labels and compares
   `subtype` without lowercasing, and the pet window sends `look <name>`
   and `get <name> from pet` although pet items carry an id, so a
   same-named item can be the wrong one. One `itemActions(item, ref,
   opts)` in `company-data.js`.
4. **Four tooltip implementations, with unescaped `innerHTML` (low
   risk).** Character, gear, map and gametime each have create / show /
   position / hide and near-identical CSS; `_positionStatTooltip` and
   gear's `positionTooltip` are byte-identical. Only the pet window
   escapes; gear, map and character quests put server strings into
   `innerHTML`. A `Client.tooltip` and `Client.escapeHtml` in core plus
   one `.ui-tooltip` class.
5. **Five tab switchers (low risk).** Gear, KillStats and Pet share a
   `makeTabSwitcher` with no ARIA; Character and Company set `role=tab`
   and persist to localStorage but one toggles `.active` and the other
   `hidden`; only `DockTabGroup` handles arrow keys. `Client.tabs(root,
   opts)` with keyboard support.
6. **Formatting drift (low risk).** Three modifier formats
   (`_formatMods`, `CompanyData.modsText`, pet abbreviations), the legacy
   Affects path prints raw seconds, and `window-company.js` inlines
   `(g/1000).toFixed(1)` beside `CompanyData.kg`. Use the `CompanyData`
   helpers everywhere.
7. **Map badges vs room badges (low risk).** The map tooltip shows
   `character` and `ephemeral` as raw lowercase text with no colour and
   omits `root`; the room window labels and colours all seven. Share
   `BADGE_LABELS` and the CSS.
8. **Smaller duplicates.** `CompanyData.el` copied into
   `window-tutorial.js`; two screen-reader-only CSS blocks and two
   live-region announcers (`combat`, `tutorial`); `who(m)` and
   `memberSelector(key)` build selectors differently (see Go proposal 3).
9. **Lint gaps.** `.jshintrc` has no `undef` or `unused`. Timers that
   never stop: gametime polls a debug flag every 500 ms in production,
   gear runs `announceGear` every second with a document-wide click
   listener, `company-data.js` runs a page-wide `querySelectorAll` every
   second. The map's outside-click capture listener survives closing the
   settings panel with the gear button. `window-gear.js` keys a `Map` by
   DOM row and deletes by hand where a `WeakMap` would do. Stat cells,
   SP/TP badges and quest/job rows are click-only `div`s. Trailing
   whitespace in `vwin.js` and `window-gametime.js`. Enabling `unused`
   and `undef` in `.jshintrc` is the cheapest first step.
10. **Dead JS left in place.** `window.GameModal` has no external caller;
    `VWin.move/resize/setBackground/addClass/removeClass/removeControl`
    are uncalled; the modal's `html` branch is unreachable because
    `Help.format` is always `"terminal"`; 12 semantic theme lines are
    byte-identical across all 20 themes and could live once in
    `gomud.css`. Left for a pass that decides whether `VWin` is meant to
    keep a public surface.

## Verification

`make generate`, `make validate`, `go test -race -timeout 30m ./...`,
`make js-lint`, `make js-test`, `scripts/browser/gmcp-dispatch-check.mjs` and
the dock, map, room, quickmenu, weather, tutorial, battle and relic browser
checks (second pass), `staticcheck ./...` (no new findings in touched files),
`deadcode -test ./...` (the six Ashveil entries gone).
