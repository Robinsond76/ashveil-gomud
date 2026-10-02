# Phase 30g4 verification and calibration

[Plan](2026-10-02-phase-30g4-progression.md), [approved design](../designs/2026-09-30-phase-30g-tempo-defense-design.md).

## Measurements

`ASHVEIL_BALANCE=1 go test ./modules/company -run '^TestBalance5v5$' -count=1 -v`, 50 fights per cell (900 fights).
The even mirror now uses the corresponding archetype's HP on both sides; its
old hidden extra player racial HP is gone. Its original even stat training,
strategy asymmetries and real-round combat remain. Ordinary enemies use the
middle/default rate unless their template/race overrides it.

No-focus medians at levels 1/5/10: **14/44/87**, against 30g1's **8/32/68**
and 30g3's **8/35/77**. All cells ended without a stall. This is measurement,
not a tuning pass: early class HP now rises faster while step growth reduces
stat bonuses. 30g5 adds tempo; 30g6 still owns the 10–15-round target and HP,
damage and healing numbers. No assertion claims that target is already met.

| level | company | enemy | fights | company wins | rounds p10/median/p90 | stalls | fallen company/enemy | damage company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | spread | spread | 50 | 48% | 10/14/20 | 0 | 3.6/3.1 | 44/57 | 1 | 0.67/0.79 | 39/37 | 20/16 | 0.8/0.4/0.4 · 0.5/0.5/0.3 | 0.0/0.0 | 0.4/0.6 |
| 1 | spread | default | 50 | 36% | 9/13/20 | 0 | 3.8/2.8 | 45/54 | 1 | 0.63/0.76 | 39/39 | 22/15 | 0.6/0.7/0.4 · 0.6/0.3/0.2 | 0.0/0.0 | 0.2/0.7 |
| 1 | default | spread | 50 | 20% | 10/14/19 | 0 | 4.5/3.1 | 44/64 | 1 | 0.64/0.86 | 40/42 | 20/15 | 1.0/0.5/0.4 · 0.4/0.2/0.3 | 0.0/0.0 | 0.1/0.9 |
| 1 | default | default | 50 | 16% | 10/13/18 | 0 | 4.5/2.9 | 43/67 | 2 | 0.61/0.83 | 42/44 | 21/17 | 0.7/0.5/0.3 · 0.4/0.2/0.4 | 0.0/0.0 | 0.2/0.9 |
| 1 | focus | spread | 50 | 14% | 9/14/19 | 0 | 4.6/3.1 | 46/67 | 1 | 0.64/0.86 | 42/43 | 21/16 | 0.9/0.4/0.5 · 0.4/0.3/0.4 | 0.1/0.0 | 0.1/0.8 |
| 1 | focus | default | 50 | 24% | 8/13/18 | 0 | 4.4/3.2 | 45/66 | 2 | 0.62/0.83 | 41/44 | 22/15 | 0.5/0.6/0.5 · 0.4/0.2/0.2 | 0.0/0.0 | 0.3/1.2 |
| 5 | spread | spread | 50 | 44% | 32/44/59 | 0 | 3.7/3.1 | 132/162 | 10 | 0.72/0.85 | 33/35 | 22/15 | 2.9/2.0/1.0 · 1.7/1.5/1.2 | 0.1/0.1 | 3.0/2.8 |
| 5 | spread | default | 50 | 42% | 34/44/64 | 0 | 3.6/3.2 | 133/162 | 11 | 0.69/0.83 | 35/35 | 22/14 | 2.5/1.8/1.5 · 1.7/1.0/1.1 | 0.2/0.1 | 3.8/3.4 |
| 5 | default | spread | 50 | 24% | 36/45/63 | 0 | 4.3/3.6 | 135/173 | 11 | 0.70/0.91 | 34/38 | 22/15 | 2.7/1.8/0.8 · 2.0/1.0/1.1 | 0.2/0.1 | 3.0/5.9 |
| 5 | default | default | 50 | 26% | 36/46/62 | 0 | 4.5/3.5 | 137/179 | 11 | 0.68/0.90 | 35/39 | 24/16 | 2.6/1.6/1.1 · 1.6/1.0/1.1 | 0.1/0.1 | 4.5/7.1 |
| 5 | focus | spread | 50 | 20% | 31/41/59 | 0 | 4.5/3.3 | 128/174 | 11 | 0.69/0.91 | 34/39 | 23/15 | 2.8/1.3/0.9 · 1.8/0.7/1.1 | 0.1/0.1 | 3.3/6.3 |
| 5 | focus | default | 50 | 28% | 35/44/61 | 0 | 4.3/3.6 | 139/175 | 11 | 0.68/0.90 | 36/38 | 23/16 | 2.4/1.6/1.3 · 1.5/1.0/0.8 | 0.1/0.0 | 4.0/6.3 |
| 10 | spread | spread | 50 | 68% | 64/87/104 | 0 | 3.5/4.2 | 280/260 | 14 | 0.73/0.85 | 39/34 | 21/16 | 4.6/2.3/2.1 · 3.8/2.4/1.9 | 0.2/0.1 | 7.4/3.4 |
| 10 | spread | default | 50 | 78% | 63/83/116 | 0 | 2.9/4.4 | 288/253 | 16 | 0.69/0.82 | 40/35 | 23/18 | 3.6/2.4/2.0 · 3.2/3.3/1.9 | 0.2/0.1 | 9.5/3.3 |
| 10 | default | spread | 50 | 48% | 63/81/104 | 0 | 3.8/4.2 | 276/278 | 15 | 0.72/0.92 | 40/37 | 24/14 | 4.9/2.4/1.7 · 3.5/2.1/1.7 | 0.1/0.1 | 9.0/11.2 |
| 10 | default | default | 50 | 48% | 60/82/105 | 0 | 3.8/4.2 | 279/281 | 16 | 0.69/0.91 | 40/38 | 24/14 | 3.7/2.5/2.0 · 3.5/2.0/2.2 | 0.2/0.1 | 9.9/12.6 |
| 10 | focus | spread | 50 | 48% | 63/79/106 | 0 | 3.7/4.1 | 272/278 | 15 | 0.72/0.91 | 39/37 | 24/15 | 4.9/2.6/1.7 · 3.9/1.8/2.2 | 0.3/0.1 | 9.1/11.3 |
| 10 | focus | default | 50 | 64% | 59/82/105 | 0 | 3.4/4.4 | 284/272 | 16 | 0.69/0.91 | 40/37 | 26/15 | 3.7/2.1/1.7 · 4.1/2.0/2.0 | 0.2/0.1 | 12.1/11.8 |

## Shipped characters before/after

`go test ./modules/company -run '^TestProgressionShippedCalibration$' -count=1 -v`.
Before is the pre-30g4 racial/HP formula and 33h1 archetype allocation of one
point per level; after uses five-level steps and the current allocation.
Equipment stat modifiers are included; the snapshots have no active effects or
wounds. Starter HP is for an untrained human before starter gear. Player
invested training and unspent points are preserved; companion training is
re-derived by 33h1 from its level and archetype/focus.

| recruit | level | HP before/after | Speed before/after | Vitality before/after |
| --- | --- | --- | --- | --- |
| Tamsin Reed | 1 | 6/11 | 0/0 | 0/0 |
| Brother Oswin | 1 | 10/10 | 0/0 | 1/0 |
| Garrick Vane | 1 | 10/12 | 1/1 | 1/1 |
| Ysolde | 1 | 6/10 | 0/0 | 0/0 |
| Tamsin Reed | 5 | 32/36 | 3/1 | 5/1 |
| Brother Oswin | 5 | 32/32 | 2/1 | 5/2 |
| Garrick Vane | 5 | 36/37 | 4/2 | 6/2 |
| Ysolde | 5 | 24/31 | 4/1 | 3/1 |
| Tamsin Reed | 10 | 64/67 | 7/1 | 11/2 |
| Brother Oswin | 10 | 64/57 | 5/1 | 11/2 |
| Garrick Vane | 10 | 68/68 | 8/2 | 12/3 |
| Ysolde | 10 | 56/56 | 8/2 | 9/1 |
| Tamsin Reed | 20 | 123/129 | 14/3 | 22/4 |
| Brother Oswin | 20 | 123/109 | 10/2 | 22/4 |
| Garrick Vane | 20 | 127/130 | 15/4 | 23/5 |
| Ysolde | 20 | 107/108 | 16/3 | 18/3 |
| Tamsin Reed | 60 | 363/179 | 42/8 | 67/14 |
| Brother Oswin | 60 | 363/159 | 30/6 | 67/14 |
| Garrick Vane | 60 | 367/180 | 43/9 | 68/15 |
| Ysolde | 60 | 315/156 | 48/10 | 55/11 |

| starter archetype | level | HP before/after |
| --- | --- | --- |
| warrior | 1 | 6/11 |
| cleric | 1 | 6/10 |
| ranger | 1 | 6/10 |
| rogue | 1 | 6/9 |
| wizard | 1 | 6/8 |

## Verification

Focused changed-package checks passed before review. New coverage includes
step boundaries, peak/death protection, unchanged pre-knee XP, incremental
post-knee scaling and integer saturation, no hard level cap, legacy YAML,
class choice/reset, template/race enemy HP, enlistment, companion spawn and
saved vitals, player file/copyover restoration, status, help/tutorial pointers,
and HTTP preview parity with live values and downsampled cumulative XP.

The actual progression page's scripts and DOM were exercised in Chromium with
an HTTP preview fixture generated by `TestProgressionPreviewParityAndDownsampledXP`:
class charts/legend, all new preview fields, save payload, display-only MaxLevel,
retired HPPerLevel absence, keyboard focus and 390px layout passed with no
page errors. This was a standalone page with stubbed AdminAPI; the Go HTTP
tests validate the calculations independently. Screenshots were inspected.

Independent full-phase review raised two P2 findings, both accepted and fixed:
(1) a saturated helper result could wrap when HP/stat bonuses were added;
saturating additions and raw/adjusted minimum consistency now have integration
regressions. (2) the Mana preview omitted ManaMax's racial growth; fixed and
covered by HTTP/live parity at default and changed step intervals. No other
blocking findings or rejected findings. Follow-up review accepted both fixes and
independently passed the config, stats, character and HTTP progression regressions.
Generation, validation, JS/Lua lint and the final editor browser checks passed;
the full `go test -race ./...` suite passed.

## Upgrade notes

No new player/company save fields or world-clock changes. Level, XP, peak and
player investment remain. Maxima and racial growth recalculate; saved vitals
are only clamped, never refilled. Copyover defers its vital clamp until module
tables load, including link-dead users. Existing explicit admin overrides
remain overrides: review them if adopting these default balance numbers.
`GamePlay.Progression.HPPerLevel` is retired; use class HPPerLevel, enemy
`hpperlevel`, DefaultHPPerLevel, HPFullLevels and HPAfterFull instead.

The seeded narration outcome fixture was recaptured because stepped stats
change real hit/damage rolls; it remains a lock for later narration-only work.
