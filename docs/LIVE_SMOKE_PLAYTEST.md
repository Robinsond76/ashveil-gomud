# Live Smoke Playtest

Phase 44. Unit and wiring tests miss start-up, persistence and multi-player
seams, so `live_smoke_test.go` plays the real server over telnet.

## Run it

```bash
make smoke
# or: ASHVEIL_LIVE_SMOKE=1 go test -run TestLiveSmoke -timeout 20m -count=1 -v .
```

It is skipped in a normal `go test` (it needs a few minutes of real time:
a camp rest is a real minute). `ASHVEIL_SMOKE_KEEP=1` keeps the server's
data copy and logs in a temp directory. Re-run it at the end of each lane.

## What it does

The test builds the server, copies `_datafiles` into a temp directory, starts
the binary on a free telnet port (rounds shortened to 2 s), then:

1. Creates a Warrior (login, race, name, archetype) and plays all eight
   tutorial stages: recruit, formation, survival, a real camp rest, the
   straw-squad battle, alignment, the gate. It fails if the practice squad
   hurts anyone.
2. In the world: `status` names the class, a shipped weapon shows its tier,
   `loot`, `company patch`, `formation`.
3. Renders `help` for every topic and alias in `keywords.yaml`.
4. Copyover (`SIGUSR1`): the connection, company and room survive.
5. A second player (a Witch, tutorial skipped) shares the room: `spells`
   lists Slumber and `say` reaches the first player.
6. Logs out, restarts the server (`SIGTERM`), logs back in: the company and
   formation persist.

Throughout, no player ever sees "not recognized", a panic or a runtime error.

## The world smoke (phase 44b)

`make smoke-world` (`TestLiveSmokeWorld`, `live_smoke_world_test.go`) is a
second run that leaves the tutorial behind and plays the world's loops on a
live server. It is its own target because it adds a few more minutes: a
journey, a fight, a timed gather and two stays.

1. A Warrior (tutorial skipped) wakes at Dunmar's West Gate at full health
   and recruits Tamsin and Oswin at the Waymark Inn.
2. The Old Kings Road journey: `route`, the fallen tree's pause, `travel
   resume`, arrival at the Fork at the Black Oak, `weather`, `scout`.
3. A random encounter (two unarmed road brigands) springs on a step into the Fork; the
   battle plays out, the spoils are looted, and both companions' new levels
   show in `company status`.
4. `gather firewood` yields a plural haul; `camp` and `camp fire` (the Fork's
   deadfall feeds it free).
5. `quit`, a restart, and login: the camp, cargo and levels survive.
6. As an admin (no road joins Dunmar to Frostfang yet) it teleports to the
   Frostfang armorer, salvages the sword into scrap iron, sells that at
   Dunmar's market (firewood is refused: it is not resold) and rests a
   night at the Waymark Inn (Well Rested).

Three things are bent in the test's own copy of the world, never in the
shipped data: new characters start at Dunmar's West Gate (`SpecialRooms:
StartRoom`), the Old Kings Road gets an always-springing encounter table
(shipped, the road has none), and the account is promoted to admin during the
restart. A camp *rest* is not played here (the tutorial run covers it);
the Fork's 15% camp-raid roll would make it flaky.

## Extending it

Add a step to `TestLiveSmoke` with the `mudClient` helpers: `do(command,
wantPattern)`, `expect`, `drain`. A line typed within one turn of the last is
dropped by the server, so `send` paces itself; do not shorten that.
Add the new phase's core loop (a rest, a fight, a new command) as a step, and
keep the pattern checks loose enough to survive wording changes.
