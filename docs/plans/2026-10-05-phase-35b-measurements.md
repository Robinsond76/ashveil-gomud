# Phase 35b measurements

These are real-round fights through the balance harness. They use the shipped config and the archetype overlay, after the 35b retune and the review fixes:

```sh
ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=60 go test ./modules/company -run 'TestBalance(5v5|Coordinated|SidesStayEven|StatusesLand)$' -v
ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=100 go test ./modules/company -run 'TestBalance(SkillWins|Mismatches|ManaRun|WarriorsAreTheTanks)$' -v
ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=50 go test ./modules/company -run 'TestBalanceZoneBands$' -v
```

No cell stalled, and no spell fizzled in any battle: the fizzled column is 0 everywhere. Every balance test passes at the numbers below. The equal-level tables (5v5 and coordinated) were run before the skill edge span changed from 16 to 14. The span only acts on a level gap, and those cells have none.

## Tuned values

| Setting | Before | Shipped | Why |
|---|---|---|---|
| `ToHitEven` | 60 | **75** | Shortens fights. At 60, about half of all swings missed, blocked, parried or dodged. Raising it to 80 changed nothing more. |
| `DodgeChanceEven` / `ParryChanceEven` | 12 / 12 | **8 / 8** | Same reason. They now sit below the shield's block, so armor and shields decide more of a fight than footwork. |
| `BlockChanceEven` | 20 | **28** | Makes the warrior the tank. A warrior takes 0.70× a rogue's damage per swing (target ≤ 0.7×). While tuning at 60 fights, 25 gave 0.70× and 30 gave 0.67×. 28 measured 0.68–0.70× across runs, leaving a small margin that 25 lacked. |
| `SkillEdgeSpan` | 16 | **14** | The higher hit chance let a level-10 company beat two level-20 foes 48% of the time over 200 fights (target ≤ 30%, 12% after 35a2). At 14 that is 20% over 100 fights, and a span of 12 gives 4%. |
| Minor Heal `waitrounds` | 2 | **1** | With a two-round chant, company clerics finished only 1.6 of 4.6 heals at level 10 and 0.1 at level 30. Most were interrupted or their caster was killed. One round makes the healer a real part of the fight. Minor Heal All stays at two. |

Tried and reverted, each with no clear gain:
- `DefaultHealing` 70;
- a change to the chant-break chance (the interrupt rule stays as planned);
- patching to full health instead of to the threshold.

## Acceptance rows

| Row | Target | Measured | Result |
|---|---|---|---|
| Owned spells never fizzle in battle | 0 fizzles | 0 in every cell; unit and wiring tests pass | Pass |
| Warriors are the tanks | warrior ≤ 0.7× rogue per swing | 3.31 vs 4.73 (**0.70×**); wizard 7.87 (0.42×); heavy rogue tempo 0.82 vs light 1.40 (0.59×) | Pass |
| Skill wins | L20 vs 3×L10 ≥ 99%; L10 vs 2×L20 ≤ 30% | 100% (5.8% HP lost); **20%** | Pass |
| Mismatches | L15 and L30 beat L10 | 100% / 100% | Pass |
| Equal 5v5 length | spread mirror median 8–12 (35a2: 14–20) | 15 / 14 / 15 / 13 / 11 / 11 at L1 / 5 / 10 / 30 / 60 / 100 | **Changed target**: 8–16 |
| Coordinated tiers | L10 wins ≥ 50% against tiers 2–3 | default company: 48% / 32%; with tactics: 83% / 47% (an earlier 60-fight run: 68% / 63%) | **Changed target**, see below |
| Mana run | casters above reserve after fight 3, below 25% after fight 4 | cleric 42% then 28%; wizard 52% then 34% (wins 100 / 89 / 65 / 45%) | **Changed target**, see below |
| Zone bands | level impact §4 table | see the zone table | Partly changed, see below |

### Why the targets changed

The owner (2026-10-05) left the balance targets to good gameplay, provided each change is explained.

- **Fight length.** 35a2 row 1 sizes a landed blow at about a fifth of a fighter's health, so each fighter takes about five blows. With misses, blocks and turns, five fighters a side cannot finish in 8–12 rounds without undoing that. The new 75% hit chance brought the mirror from 18–20 rounds down to 14–15 at levels 1–10 and 11–13 later. The harness now asserts a median of 8–16.
- **Coordinated enemies.** A company that ignores coordination now loses to it, as it should. It wins 48% against tier 2 and 32% against tier 3 at level 10. The player's answer is the tactics they already have: focus the weakest foe and have the warrior guard the healer. With them, a level-10 company won 83% and 68% against tier 2 in two runs, and 47% and 63% against tier 3. The harness now asserts, for a company with those tactics:
  - at least 50% wins against its own level's tier (tier 2 at level 10);
  - at least 33% one tier above (tier 3 at level 10, a tier normally met from level 25).
  - At level 30 (tier 3 native), tactics win 57% against tier 2 and 53% against tier 3.
- **Mana run.** HP, not mana, ends a run of fights. Patching to the 50% threshold left the company too hurt by fights 3–4, so the test also tends wounds (`heal wounds`) after each fight, as a player would. Casters still end fight 4 well under half their pool. The test asserts a median above 25% after fight 3 and below 50% after fight 4. The design's pools (wizard 40 + 10 a level, cleric 36 + 9) stay as they are.

### Healing, casting and interrupts (level 10, 60 fights)

| Cell | Company casts begun / cast / broken | Enemy casts begun / cast / broken | Healing done / damage taken (company) |
|---|---|---|---|
| default vs default | 5.0 / 2.5 / 2.0 | — | 37 / 277 (13%) |
| default vs roles3 | 1.3 / 0.1 / 0.6 | 1.8 / 0.1 / 1.1 | 1 / 312 |
| tactics vs roles2 | 7.8 / 5.9 / 1.0 | 1.2 / 0.0 / 0.7 | 82 / 258 (32%) |
| tactics vs roles3 | 4.2 / 1.6 / 1.2 | 2.1 / 0.3 / 1.2 | 23 / 286 (8%) |

Enemy healers rarely finish a heal: they are focused or interrupted. Company clerics finish half their chants, or three quarters when guarded.

## Zone bands (TestBalanceZoneBands, 50 fights a cell)

ZONE_TABLE

## Deviations from the plan

- **Draught prices** are 50 / 150 / 350 gold. Moilyn sells the minor and lesser draughts; no shop sells the greater one.
- **Spell power** is read through an actor method (`SpellPower`) in the scripts, not a free function.
- **Guard count** is captured as the battle begins, so a level gained mid-battle changes nothing until the next one.
- **Minor Heal** chants for 1 round (above).
- **Combat chances and skill edge span** were retuned (above). The code defaults keep GoMud's engine values; the shipped config sets every one.
- **Changed targets:** coordinated tiers, the 5v5 mirror length, the mana run and the zone round counts (above).

## 5v5 table (TestBalance5v5, 60 fights a cell)

T5V5

## Coordinated table (TestBalanceCoordinated, 60 fights a cell)

TCOORD
