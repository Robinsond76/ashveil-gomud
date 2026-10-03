# Phase 30g5 verification

Delivered 2026-10-03 on `phase-30g5-action-meter`, based on `b62f1987`
(latest master, including 30f and 30g4).
[Design amendment](../designs/2026-10-02-phase-30g5-action-meter-amendment.md),
[delivery plan](2026-10-02-phase-30g5-action-meter.md).

## Behavior and coverage

Each actor has one ephemeral meter, filled once per combat round from its
own effective Speed and personal burden. One opening turn, a maximum of two
turns per round, and at most 99 fractional points carried. Shared enemies
fill once across several player battles; an overlapping battle preserves
progress, whereas a replacement battle or unmanaged combat restart clears it.
Separated idle companions spend no opening credit in an owner's remote fight.
No meter or combat generation is serialized; global time is unchanged.

Real `DoCombat` regressions cover fast/slow cadence, all four waiting attack
directions on zero-turn rounds, PvP, companions, enemy attacks, fight replacement,
death between passes, first-tick status timing, tackle spending one turn,
opening-strike/aimed-shot replacement, guard budgets and once-per-round shield
counters, chants, and wind-up preparation/release. Help index, aliases, rendering,
hub links, and tutorial pointers are tested. Pure tests cover config validation,
rate clamps, burden/modifiers, fractional boundaries, shared combat lifecycle,
simulator lethal-first/lethal-second ordering, and the ranking turn cap.

Legacy non-tempo brawl fixtures pin one turn per round so their assertions
continue to isolate the mechanic they test. Tempo regressions and the balance
harness explicitly use the production calculation. Narration goldens were
unchanged. A retreat-pressure fixture now hardens enemies so lower 30g4 HP
cannot end its first group before the pressure assertion.

## Independent review

The required full-phase reviewer found five issues, all accepted and fixed with
regressions: zero-turn weapon waits freezing, separated companions inheriting
remote battle meters, simulator bursts before the opponent's primary turn,
ranking omitting a configured one-turn cap, and missing help index inclusion.
The lead verified each finding. Follow-up independent review confirmed the
fixes and additional integration coverage, ran focused tests, and reported no
remaining blockers. No finding was rejected.

## Measurements

`ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=30 go test ./modules/company -run '^TestBalance5v5$' -count=1 -v`
ran 540 fights across 18 cells in 63.144 seconds, with no stalls. This is the
final run after correcting zero-turn weapon waits.

No-focus even fights at levels 1/5/10 have medians **16/60/95** rounds,
compared with the recorded 30g4 **14/44/87** (50 fights per cell) and 30g1
**8/32/68**. Different sample sizes and randomized hit/defense outcomes limit
fine comparisons. Tempo currently slows most low-Speed fighters; 30g6 owns
HP, damage, healing, and tempo calibration toward the 10–15-round target.
This slice does not claim that target is met.

The harness now records nonempty captured leader narration lines per round,
including round aftermath. Cell averages are 7.3–9.6 lines, with a peak of 26.
An 8-second pacing/readability judgment still requires player observation;
line counts alone do not establish it.

| level | company | enemy | fights | company wins | rounds p10/median/p90 | stalls | fallen company/enemy | damage company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | mean/peak narration lines per round |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | spread | spread | 30 | 27% | 12/16/23 | 0 | 4.2/2.0 | 32/62 | 1 | 0.50/0.61 | 35/39 | 19/17 | 0.8/0.4/0.3 · 0.4/0.2/0.3 | 0.1/0.0 | 0.0/0.5 | 9.6/22 |
| 1 | spread | default | 30 | 23% | 12/21/30 | 0 | 4.5/2.5 | 40/63 | 2 | 0.48/0.61 | 37/38 | 21/15 | 0.7/0.6/0.6 · 0.5/0.3/0.2 | 0.1/0.0 | 0.0/0.9 | 8.5/24 |
| 1 | default | spread | 30 | 23% | 12/18/22 | 0 | 4.3/3.1 | 45/65 | 1 | 0.47/0.65 | 39/45 | 26/15 | 0.7/0.6/0.3 · 0.4/0.2/0.4 | 0.0/0.0 | 0.0/0.7 | 8.8/23 |
| 1 | default | default | 30 | 20% | 12/16/27 | 0 | 4.4/3.0 | 43/66 | 2 | 0.44/0.63 | 45/44 | 20/17 | 0.6/0.7/0.3 · 0.4/0.3/0.2 | 0.0/0.1 | 0.2/1.4 | 8.6/24 |
| 1 | focus | spread | 30 | 13% | 12/18/24 | 0 | 4.7/3.1 | 43/69 | 2 | 0.48/0.66 | 40/45 | 23/15 | 0.8/0.6/0.5 · 0.3/0.2/0.5 | 0.1/0.0 | 0.0/0.8 | 8.7/23 |
| 1 | focus | default | 30 | 17% | 11/17/24 | 0 | 4.5/2.8 | 40/63 | 2 | 0.45/0.63 | 43/44 | 21/13 | 0.8/0.5/0.4 · 0.4/0.2/0.3 | 0.1/0.0 | 0.1/0.6 | 8.7/24 |
| 5 | spread | spread | 30 | 40% | 43/60/74 | 0 | 3.9/3.2 | 137/167 | 12 | 0.57/0.67 | 34/34 | 23/16 | 3.4/1.9/1.1 · 1.4/1.6/1.2 | 0.2/0.0 | 4.2/2.9 | 8.4/23 |
| 5 | spread | default | 30 | 43% | 40/51/76 | 0 | 3.6/3.0 | 129/162 | 14 | 0.54/0.66 | 34/36 | 25/16 | 2.5/1.2/1.5 · 1.3/1.2/1.2 | 0.0/0.1 | 4.1/3.7 | 9.0/24 |
| 5 | default | spread | 30 | 33% | 44/61/79 | 0 | 4.0/3.9 | 145/169 | 12 | 0.56/0.72 | 35/38 | 23/14 | 3.1/1.7/1.1 · 1.7/1.6/1.0 | 0.2/0.1 | 3.3/5.5 | 7.6/23 |
| 5 | default | default | 30 | 20% | 40/55/73 | 0 | 4.5/3.4 | 130/180 | 12 | 0.53/0.71 | 35/39 | 25/17 | 2.3/1.3/1.1 · 1.8/0.9/0.7 | 0.1/0.1 | 3.2/8.0 | 8.1/24 |
| 5 | focus | spread | 30 | 10% | 41/54/74 | 0 | 4.7/3.1 | 125/181 | 11 | 0.54/0.72 | 37/39 | 21/15 | 2.5/1.8/1.0 · 1.2/0.9/1.0 | 0.1/0.1 | 3.1/7.1 | 7.6/22 |
| 5 | focus | default | 30 | 33% | 42/54/73 | 0 | 4.0/3.7 | 140/171 | 11 | 0.53/0.71 | 36/38 | 25/15 | 2.4/1.4/0.9 · 1.6/0.7/1.4 | 0.2/0.1 | 3.3/7.9 | 8.0/26 |
| 10 | spread | spread | 30 | 77% | 75/95/125 | 0 | 2.7/4.4 | 288/239 | 15 | 0.59/0.68 | 40/35 | 20/15 | 3.9/2.8/1.6 · 4.3/2.3/2.2 | 0.2/0.2 | 8.4/2.9 | 8.0/23 |
| 10 | spread | default | 30 | 77% | 80/103/147 | 0 | 3.1/4.5 | 295/264 | 20 | 0.57/0.66 | 39/36 | 21/15 | 4.0/2.0/2.5 · 4.0/2.0/2.4 | 0.1/0.1 | 9.9/3.9 | 7.9/25 |
| 10 | default | spread | 30 | 53% | 75/101/130 | 0 | 3.8/4.2 | 273/284 | 18 | 0.58/0.74 | 39/37 | 24/14 | 4.5/2.3/1.3 · 4.0/2.2/1.8 | 0.1/0.2 | 8.6/11.8 | 7.1/24 |
| 10 | default | default | 30 | 60% | 80/109/138 | 0 | 3.7/4.5 | 288/284 | 16 | 0.55/0.74 | 41/37 | 24/15 | 3.3/2.8/1.6 · 3.9/1.8/1.2 | 0.2/0.1 | 11.3/11.8 | 7.2/26 |
| 10 | focus | spread | 30 | 47% | 73/106/128 | 0 | 3.9/4.2 | 270/289 | 16 | 0.59/0.74 | 38/37 | 22/16 | 5.0/3.0/1.7 · 4.4/2.3/2.1 | 0.3/0.1 | 7.2/11.2 | 7.4/24 |
| 10 | focus | default | 30 | 50% | 79/98/129 | 0 | 3.8/4.1 | 274/289 | 16 | 0.56/0.74 | 41/37 | 23/15 | 4.4/2.5/1.9 · 3.8/2.0/1.9 | 0.2/0.1 | 11.0/12.8 | 7.3/26 |

## Repository checks

- `make generate`: passed; no generated wiring changed.
- `make validate`: passed (`gofmt` and `go vet`).
- `go test -race ./...`: passed across the full repository, including tutorial pointers.
- Focused tempo/meter, simulator/ranking and help tests: passed.
- `git diff --check`: passed.
- No JavaScript or Lua was changed; their linters were not applicable.

Checks used Go 1.24.7. Initial full checks could not download two remaining
modules through the sandbox network; dependencies were downloaded with approved
network access and the required checks restarted. This was an environment issue.
