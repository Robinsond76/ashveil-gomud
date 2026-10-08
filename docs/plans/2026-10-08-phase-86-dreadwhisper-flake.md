# Phase 86: two flaky tests

Branch `claude/phase-86-dreadwhisper-flake-zjxit1`. Test-only change, no game code or help changed.

## TestSparksOverloadsItsTargetsThroughARealCast (reproduced, cause found)

Rate before: 5 failures in 240 runs (`-race`, four processes at once). Rate after: 0 in 240.

Cause: not a game bug. The test recasts Sparks once per round (up to 30 times), but five bandits are all on Aria, and any
damaging blow may break her chant ("Shower of Sparks interrupted", seen in nearly every round of a failing run). A cast that
is broken never lands, and a player recasts by hand, so the loop's recasts were all being broken in the failing runs. It was
a test that depended on the chant-break dice while testing something else (overload). Fix: pin the break dice with
`hooks.UseBreakRollForTest` so no blow breaks the chant (interrupts have their own tests in `wiring_interrupts_test.go`).
Decision: pin the dice rather than raise the 30-attempt cap or add retries; the test's subject is the status, not the interrupt.

## TestDreadWhisperMakesAFoeTakeAMoraleCheck (not reproduced)

Tried, all clean: about 1,800 shuffled runs of the morale, pace, battle-event and witch tests (race, four at once, 43 tests),
30 shuffled passes of the dread/morale/witch/hex group, and 400 solo race runs under full CPU load (plus 600 solo in Phase 83).
Read the path end to end: the dice that matter are pinned (hex resist 0, morale roll 99, no hits via `forceBlows`, no counters),
the hex ledger is reset, the cast is "sure" in battle, and `moraleSay` reaches the player in the room.
The one Phase 83 observation (chant restarted, neither line printed) points at a broken chant, and `witchBrawl` did not pin the
chant-break dice, the same gap as the Sparks test. Decision: pin them in `witchBrawl` too (no witch test is about interrupts).
This is a hardening against a mechanism that exists, not a proven cause; if it ever fails again it is a new lead.

## Review (2026-10-08)

No code change. The caster recasts: in three failing unpinned transcripts (3 of 80 race runs) all 31 attempts chanted and each
was broken once, so "never casts again" (Phase 85 note) was a misread, not a caster bug. Breaking the targets on purpose
fails both tests (Sparks without the overloaded buff 3 of 3; Dread Whisper with every hex resisted 2 of 2). Pinned Sparks
passed 160 of 160 race runs. In 20 runs of every witch test `breakRoll` was never called, so the `witchBrawl` pin is
insurance only; a future Dread Whisper failure is still an open lead.
