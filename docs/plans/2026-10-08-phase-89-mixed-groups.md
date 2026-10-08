# Phase 89: mixed enemy groups (2026-10-08)

Robinson asked for two things: `testarea fight` should build a group of mixed
kinds ("level 2, two imps and a skeleton"), and enemy groups across the game
should not be rows of one creature. The world is temporary, so this changes
the systems that build groups, not hand-made content.

## Decisions

- **Syntax.** `testarea fight <level> <count> <type> [<count> <type> ...]`,
  2-6 foes in all (`testarea fight 2 2 imp 1 skeleton`). The older
  `<level> <size> [type ...]` form still works (types repeat in turn). The
  words after the level are read as counted kinds when they are an even
  number of number-and-type pairs and at least one type is a name; a mob id
  works in either form (in the counted form beside a name). A boss type
  leads the group wherever it is listed. `help testarea` documents both.
- **The rule for natural groups.** An ordinary group of three or more that
  would be all one kind gets a second kind, taking the back half of the group
  (one of three, two of four). Group size and levels are unchanged, so
  difficulty is unchanged. Bosses (a boss and its escorts are authored) and
  deliberate packs stay as written; pairs stay as they are.
  - *Random room encounters* (`encounters.Mix`, called by the encounters
    module before `Plan`): the second kind is drawn by weight from the other
    ordinary compositions of the same zone table, so it fits the zone and the
    band's levels. Healers and solitary templates are never drawn (the healer
    share and four-foe rules stay valid). A table with one kind is left alone.
    A composition marked `pack: true` is never mixed; the wolf, dog, rat and
    bat groups of the shipped zones carry it.
  - *Zone spawn groups* (`FormSpawnGroups`, `planMixes`): an idle spawn group
    of three or more that is all one kind swaps its last member for another
    kind from the room's own hostile spawn list; the swapped spawn entry
    tracks the newcomer (so the room keeps its head count), and the newcomer
    joins the group. A
    group in a fight, already mixed, or in a room with one kind is left.
  - *Test-area random fights and story groups* build their foes by hand
    (`testarea fight`, authored story events), so they are as written.
- **Names.** Each kind numbers itself: "the first imp", "the second imp" and
  "the skeleton" (`TestMixedGroupNamesNumberEachKindOnItsOwn`).

## Tests

- `TestMixGivesOneKindGroupsASecondKind` (rules), `TestAWholeKindGroupComesMixed`
  (the real step into an encounter room), `TestPlanMixesSwapsOneKindGroups`,
  `TestParseFoes`, and the `testarea fight 2 2 imp 1 skeleton` check in
  `TestFightsAndCompanionTools`.

## Help

`help testarea` (admin) and `help combat` (a group of three or more usually
mixes kinds).

## Gates

generate, validate, js-lint, js-test and `go test -race -timeout 30m ./...` ran
once. One failure, `TestWonBattleCompanionsTalk` (`wiring_banter_test.go:73`,
"5 is not <= 4": five spoken lines after a victory where the test allows four):
it fails about 2 in 40 on unmodified master as well, so it is a pre-existing
flake unrelated to this phase, left for a flake phase (the review found
the cause and fixed the test; see below).

## Review (2026-10-08)

Opus review thread. Accepted and fixed:

- **Spawn groups grew.** `mixSpawnGroups` put the swapped member's spawn
  entry on cooldown, so when it came due it respawned its old kind, which
  joined the idle group: a room of three became a group of four. The entry
  now tracks the newcomer; it respawns its own kind only after the newcomer
  dies. `TestARoomsOneKindGroupMixesAndKeepsItsHeadCount` drives `Prepare`
  three times (fails on the old code).
- **The opening line named one kind.** "A patrol of skeletons shambles
  forward" opened a group of two skeletons and a bone warden. A mixed
  encounter's line now names the newcomers ("A bone warden comes with
  them.", "Two forest imps come with them.";
  `TestAMixedGroupsOpeningLineNamesTheNewcomers`). Marrowmere's "three huge
  crocodiles" now reads "huge crocodiles".
- **Glancing prose.** A blow's lines were picked by damage alone, so a
  glancing blow of half the weapon's top took the solid lines ("thrust
  punches through Corrin's guard ... (glancing, 4 damage)"). A glancing
  blow now takes the weak lines and a telling one at least the solid lines
  (`blowProsePct`; `TestABlowsLinesMatchItsQuality` through a real pass,
  fails on the old code). Crits and blows that did nothing are unchanged.
- **`TestWonBattleCompanionsTalk` (2 in 40 on master).** Not a game bug:
  since #203 a spoken line can break around its speaker ("Not one wound,"
  Garrick announces. "A masterpiece."), and the test counted quoted
  stretches, so a four-line exchange counted five. It now counts lines;
  150 of 150 clean (2 failures in 150 before).

Checked and kept: `Mix` draws only from the same zone table's ordinary
compositions, and `Plan` still sets levels from the band and the group's
size, so no foe comes above the band and the at-level numbers (measured
with fixed foes) are unchanged. Across the shipped tables the newcomer's
health at the band's low is within about a fifth of the kind it replaces;
the toughest draw is a pack animal (a wolf joining brigands), which those
zones already field three or four at a time. A pack's members may still be
drawn into another group (a dog running with brigands); kept.
