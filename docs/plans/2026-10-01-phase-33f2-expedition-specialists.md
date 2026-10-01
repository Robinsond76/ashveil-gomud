# Phase 33f2: Expedition Specialists — Plan

Design: [33f design](../designs/2026-10-01-phase-33f-company-specialists-design.md),
"33f2: Expedition specialists". Worktree
`.worktrees/phase-33f2-expedition-specialists`, branch
`phase-33f2-expedition-specialists`.

## Task 1: resolver and view

- `internal/archetypes/specialists.go`: utility ids, `Specialist`,
  optional `SpecialistProvider` (`BestSpecialist`, `SpecialistsView`) and
  `AmbushEvader`, `PctByLevel`.
- `modules/archetype`: utilities table and archetype `Utility` lists
  (ranger trail/pathfinder, rogue keeneye/haggle, wizard weather); config
  for trail and Keen Eye; `BestSpecialist` over the 17b member resolver,
  gated by `autoskill`; `specialists` and `company specialists`.

## Task 2: capabilities

- Read the Trail: step listener and bare `track` (archetype module);
  departure warning and ambush evasion in `modules/expedition`.
- Keen Eye: step listener; `Character.KnownSecretExits`,
  `SeesSecretExit` in room text and GMCP exits.
- Pathfinder: `modules/walking` eases terrain strain (5%/level, floor road
  strain), shown by `strain`.
- Weather Sense: `ZoneWeather.Next` foretold at establish/advance, rolled
  for old saves at recovery; `Forecast`; forecast lines in `weather`.
- Haggle: `market.HaggledBuy`/`HaggledSell`, trades and listing.
- Retire `search` (skill refunded) and the stock `track`; update trainer
  74, professions, the admin user.

## Task 3: help and tutorial

New pages `specialists`, `trail`, `keeneye`, `pathfinder`, `forecast`,
`haggle` (indexed, aliased: `track`, `search`, `secret-exits`,
`weather-sense`...); updated `company`, `weather`, `market`, `strain`,
`autoskill`, `archetype`, `travel`, `ranger`, `treasure-hunter`,
`cooking`, `training-schools`, `stash`; a Departure hint.

## Task 4: integration tests

Real `go` steps (trail by companion ranger, level detail, autoskill off,
no tracker; Keen Eye spotting and memory, room details), `track` and
`specialists` commands, `BestSpecialist` ownership (separated, downed,
other player, switched off), `company specialists`, walking strain and
`strain`, market trades/listing/round trips, weather command at each
level, foretold weather on advance and recovery, travel departure warning
and ambush evasion; help render and index tests; retired commands.

## Task 5: migration and recovery

`search` joins the retired-skill refund. Weather saves without `Next`
gain one on load. `KnownSecretExits` is omitempty on the user record.
No other durable state.

## Task 6: review and integration

Independent reviewer; fixes with regression tests; Project Status;
`make generate`, `make validate`, `go test -race ./...`; merge and push.
