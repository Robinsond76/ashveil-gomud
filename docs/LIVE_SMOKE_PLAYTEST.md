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

## Extending it

Add a step to `TestLiveSmoke` with the `mudClient` helpers: `do(command,
wantPattern)`, `expect`, `drain`. A line typed within one turn of the last is
dropped by the server, so `send` paces itself; do not shorten that.
Add the new phase's core loop (a rest, a fight, a new command) as a step, and
keep the pattern checks loose enough to survive wording changes.
