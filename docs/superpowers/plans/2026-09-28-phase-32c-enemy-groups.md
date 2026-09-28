# Phase 32c: Enemy Groups — Plan

Design: [32c design](../specs/2026-09-28-phase-32c-enemy-groups-design.md).

## Task 1: Naming, pure (`internal/mobparty`)

- [ ] Tests first (`internal/mobparty/naming_test.go`): `Plural` (regular,
  -y, -s/-x/-ch/-sh, -f/-fe, irregulars such as man/men and mouse/mice,
  last word of a multi-word name); count words; `NameGroup` picks the
  most common kind, ties to the toughest, noun from the member, "band"
  when none; an authored name wins; the words that name a group (noun,
  kind plural, authored name's words); ordinals for repeats ("a second
  band of ruffians").
- [ ] `naming.go`: `Plural`, `CountWord`, `Ordinal`, `NameGroup`,
  `Keywords`, with `MobSummary` gaining `Name`, `Noun`, `GroupName`,
  `GroupDesc`.

## Task 2: The noun and the stable name (`internal/races`, `internal/mobs`, `internal/rooms`)

- [ ] Tests first: a race's `groupnoun` and a mob file's override load;
  a spawn group formed in `Room.Prepare` names every member once
  (`GroupName`); the name holds as members fall; a survivor regrouping
  takes the name of the group it joins; a spawn entry's `groupname` and
  `groupdesc` name its group.
- [ ] `races.Race.GroupNoun`, `mobs.Mob.GroupNoun` (file), `GroupName`
  and `GroupDesc` (runtime); `SpawnInfo.GroupName`, `GroupDesc`.
- [ ] `FormSpawnGroups` names its groups after assigning them.
- [ ] Content: `groupnoun` on the shipped races (band by default; canine
  pack; rodent, insect, giant spider swarm; undead host; the table is
  in `docs/PROJECT_STATUS.md` for the owner).

## Task 3: Groups in the room (`internal/rooms`, `internal/enemyparty`, `modules/gmcp`)

- [ ] Tests first: the room line (count, list, one kind with no list,
  a group of one as today, "(fighting you)", "(fighting Brom)",
  "(waiting)", hidden members left out, all hidden not shown);
  `enemyparty.Groups` names each party and orders repeats;
  `FindGroup` by noun, kind, authored words, `#2`, and not by a
  member's name.
- [ ] `mobparty_display.go` renders the new line from `NameGroup`.
- [ ] `enemyparty`: `Group` (party, name, keyword, description),
  `Groups`, `GroupOf`, `FindGroup`, `FirstAim` (the weakest legal living
  visible member for the attacker's column and reach, else any legal).
- [ ] GMCP `Room.Info.Contents.Npcs[].group`.

## Task 4: Starting and holding a battle (`internal/usercommands`, `internal/hooks`)

- [ ] Tests first (wiring, `modules/company/wiring_groups_test.go`, real
  commands and `DoCombat`):
  - `attack <noun>`/`attack <kind>` start a battle with the group, aimed
    at the weakest reachable member;
  - `attack <member>` refused with the group's command; a group of one
    attacked by its own name; `attack #<id>` starts a battle with the
    mob's group; `attack ruffians#2` takes the second group;
  - in a battle (or once aimed): every `attack` form, `cast`,
    `backstab`, and `shoot` refused, the aim unchanged;
  - out of a battle: a harmful `cast` or a `backstab` or `shoot` at a
    mob refused with the `attack` pointer; a helpful spell casts;
  - a player alone turns on the next member when the first falls.
- [ ] `attack.go`: the resolver (group, member refusal, group of one,
  `#<id>`, bare, random forms), the in-fight refusal, the first aim.
- [ ] `skill.cast.go`, `skill.skulduggery.backstab.go`, `shoot.go`: the
  refusals.
- [ ] `combat_battle.go`: a player with no companion present turns by
  strategy (`closeIdleBattles`, after the rally).
- [ ] Existing wiring tests (29a, 29b, 29b2, 27c) that aim at a member
  to set up a scene set the aim directly or start by the group.

## Task 5: `look <group>` and `scout` (`internal/usercommands`)

- [ ] Tests first: `look <group>` (name, count, doing, members with
  health words, authored description, pointers); `look <member>`
  unchanged; `scout` lists groups; `scout <group>` draws the grid with
  health words and marks what the viewer can reach; a hidden member left
  out; too dark; works in a fight and on a waiting group; spends no
  round.
- [ ] `look.go` (group before member), `scout.go` (new), registered in
  `usercommands.go`.

## Task 6: The fight's name (`internal/combatstream`, `internal/hooks`)

- [ ] Tests first: a summary with a group name heads "The fight with a
  band of ruffians is over" (and defeat, broken off); without one, as
  today.
- [ ] `Stream.Name(fightID, name)`, `FightInfo.GroupName`,
  `Summary.GroupName`; `beginBattle` names the fight.

## Task 7: Tutorial and ambush (`modules/tutorial`, `modules/expedition`)

- [ ] Tests first: the squad is "the straw squad"; `scout squad` and
  `attack squad` pass the Combat lesson; `attack footman` is refused
  with the hint; the ambush pair is named.
- [ ] `raiseSquad` names the squad; the expedition names its pair.

## Task 8: Player help and tutorial hints

- [ ] `help scout` (new, `.template`), `help attack` (rewritten as a
  `.template`), `help targeting`, `help combat` (links `scout`), `help
  cast`, `help shoot`, and the backstab help updated; `scout` in
  `keywords.yaml` with aliases.
- [ ] The Combat lesson's hints (`stages.go`).
- [ ] Render tests for the pages; `TestTutorialHelpPointersExist`.

## Task 9: Verification, review, status

- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review; verify and fix findings with regression tests.
- [ ] `docs/PROJECT_STATUS.md` work-log entry with the **Review:** line.
