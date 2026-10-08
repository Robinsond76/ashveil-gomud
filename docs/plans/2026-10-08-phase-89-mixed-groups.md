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
    kind from the room's own hostile spawn list; the swapped spawn entry goes
    on cooldown as if the mob had died, and the newcomer joins the group. A
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
flake unrelated to this phase, left for a flake phase.
