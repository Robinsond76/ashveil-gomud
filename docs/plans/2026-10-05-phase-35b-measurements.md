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

Rows group the cells by the design's section 4 table. The boss escorts are one level below the band top (the zone's own foe levels), and the boss is 2 levels above its escorts with 2.5× their health.

| Band | Row | Fights | Wins | No one fallen | Fallen (mean) | HP lost | Median rounds |
|---|---|---|---|---|---|---|---|
| 1-3 | 2–3 foes, band middle | 200 | 100% | 89% | 0.20 | 16.7% | 10 |
| 1-3 | 2–3 foes, band low | 200 | 99% | 86% | 0.24 | 18.4% | 10 |
| 1-3 | 4 foes, band middle | 50 | 94% | 38% | 1.38 | 45.5% | 17 |
| 1-3 | boss + 4 escorts, band top, with tactics | 50 | 60% | 18% | 2.64 | 64.8% | 30 |
| 8-10 | 2–3 foes, band middle | 300 | 100% | 93% | 0.10 | 13.9% | 9 |
| 8-10 | 2–3 foes, band low | 300 | 100% | 89% | 0.20 | 16.1% | 10 |
| 8-10 | 4 foes, band middle | 50 | 98% | 36% | 1.40 | 42.6% | 17 |
| 8-10 | boss + 4 escorts, band top, with tactics | 50 | 74% | 28% | 1.92 | 55.2% | 33 |
| 8-10 | 2–3 foes, company 3 levels under band low | 300 | 96% | 64% | 0.88 | 30.3% | 13 |
| 18-20 | 2–3 foes, band middle | 300 | 100% | 60% | 0.51 | 19.2% | 9 |
| 18-20 | 2–3 foes, band low | 300 | 100% | 49% | 0.70 | 23.5% | 10 |
| 18-20 | 4 foes, band middle | 50 | 94% | 2% | 2.28 | 52.8% | 19 |
| 18-20 | boss + 4 escorts, band top, with tactics | 50 | 38% | 6% | 3.82 | 81.5% | 32 |
| 18-20 | 2–3 foes, company 3 levels under band low | 300 | 93% | 30% | 1.53 | 38.5% | 13 |
| 28-30 | 2–3 foes, band middle | 300 | 100% | 46% | 0.69 | 21.2% | 9 |
| 28-30 | 2–3 foes, band low | 300 | 100% | 45% | 0.72 | 22.1% | 9 |
| 28-30 | 4 foes, band middle | 50 | 96% | 0% | 2.34 | 52.5% | 15 |
| 28-30 | boss + 4 escorts, band top, with tactics | 50 | 46% | 12% | 3.72 | 78.9% | 27 |
| 28-30 | 2–3 foes, company 3 levels under band low | 300 | 98% | 26% | 1.38 | 34.1% | 11 |

Targets (design section 4) and outcome:

| Row | Target | Result |
|---|---|---|
| Band middle, 2–3 foes | ≥ 97% wins, nobody fallen in ≥ 85%, ≤ 30% HP lost, 4–8 rounds | Wins, HP lost and ≤ 12 rounds pass (asserted). Nobody fallen: 89% / 93% at bands 1–3 and 8–10, but 60% / 46% at 18–20 and 28–30 (**reported**). |
| Band low, 2–3 foes | ≥ 85% wins, ≤ 1 fallen, ≤ 45% HP lost | Pass (asserted) |
| Band middle, 4 foes | ≥ 90% wins, ≤ 1 fallen, ≤ 45% HP lost | Wins pass (asserted); 1.4–2.3 fallen and 43–53% HP lost (**reported**) |
| Boss, band top | 70–85% wins, 10–15 rounds | 60 / 74 / 38 / 46% with tactics, about 30 rounds (**reported**); without tactics 30 / 38 / 5 / 15% in an earlier probe (span 16, 40 fights) |
| 3 levels under band low | 30–60% wins | 93–98% (**reported**) |

**Owner decision (2026-10-05):** "If you can't achieve the goal after these 10 minutes, settle for the best you achieved." The rows marked reported are logged by the harness and left to phase 37, which sets zone encounter sizes, levels and boss rolls.

Why they miss:
- **Fallen against 4 foes, and at bands 18+.** One landed blow is a fifth of a fighter's health, so focused enemies drop a member in about five hits.
- **Bosses.** A boss at +2 levels with 2.5× health plus four escorts is a harder fight than an even 5v5.
- **Under-levelled company.** A 3-level gap is 3 points of skill edge (about a fifth of a full edge at span 14). Against only 2–3 foes, five members win on numbers.

## Deviations from the plan

- **Draught prices** are 50 / 150 / 350 gold. Moilyn sells the minor and lesser draughts; no shop sells the greater one.
- **Spell power** is read through an actor method (`SpellPower`) in the scripts, not a free function.
- **Guard count** is captured as the battle begins, so a level gained mid-battle changes nothing until the next one.
- **Minor Heal** chants for 1 round (above).
- **Combat chances and skill edge span** were retuned (above). The code defaults keep GoMud's engine values; the shipped config sets every one.
- **Changed targets:** coordinated tiers, the 5v5 mirror length, the mana run and the zone round counts (above).

## 5v5 table (TestBalance5v5, 60 fights a cell)

| level | company | enemy | fights | company wins | rounds p10/median/p90 | won-fight rounds mean | stalls | fallen company/enemy | damage company/enemy | net HP lost company/enemy | healing company/enemy | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | lines/round mean/peak | casts begun/cast/fizzled/broken company · enemy |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | spread | spread | 60 | 48% | 11/15/19 | 15.8 | 0 | 3.3/3.1 | 242/253 | 237/227 | 0/0 | 0.74/0.73 | 53/57 | 15/14 | 3.0/1.1/0.8 · 3.2/1.0/0.9 | 0.1/0.2 | 1.7/1.4 | 11.4/29 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 1 | spread | default | 60 | 25% | 13/16/21 | 19.5 | 0 | 4.5/2.3 | 209/282 | 258/199 | 0/0 | 0.73/0.74 | 56/54 | 15/16 | 3.1/1.0/1.0 · 2.4/0.9/0.6 | 0.1/0.1 | 2.3/2.1 | 10.5/28 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 1 | spread | casters | 60 | 17% | 11/15/20 | 17.3 | 0 | 4.5/1.7 | 198/279 | 258/190 | 0/0 | 0.70/0.75 | 56/55 | 14/15 | 2.9/1.3/1.1 · 1.9/1.1/0.8 | 0.1/0.2 | 2.0/1.9 | 11.3/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 1 | kit | spread | 60 | 68% | 15/18/24 | 18.9 | 0 | 2.1/4.2 | 268/236 | 188/248 | 33/0 | 0.58/0.67 | 55/53 | 23/16 | 2.5/1.0/0.9 · 2.9/0.8/0.7 | 0.2/0.1 | 1.6/1.2 | 11.4/30 | 4.1/2.5/0.0/1.0 · 0.0/0.0/0.0/0.0 |
| 1 | default | spread | 60 | 83% | 15/19/26 | 19.6 | 0 | 1.4/4.7 | 280/208 | 148/257 | 50/0 | 0.57/0.69 | 56/54 | 24/16 | 2.8/1.1/0.3 · 2.3/1.2/1.1 | 0.1/0.1 | 1.0/2.4 | 11.1/30 | 5.3/3.9/0.0/0.5 · 0.0/0.0/0.0/0.0 |
| 1 | default | default | 60 | 43% | 16/22/31 | 23.7 | 0 | 3.7/3.8 | 233/273 | 219/215 | 30/0 | 0.53/0.72 | 58/57 | 24/16 | 2.5/0.9/0.9 · 1.6/0.8/0.9 | 0.1/0.1 | 1.4/2.8 | 9.8/28 | 4.7/2.2/0.0/1.6 · 0.0/0.0/0.0/0.0 |
| 1 | default | casters | 60 | 42% | 17/24/33 | 26.3 | 0 | 4.0/4.0 | 245/259 | 235/223 | 4/0 | 0.57/0.72 | 56/54 | 25/16 | 2.5/1.2/0.9 · 1.9/0.9/1.1 | 0.1/0.1 | 1.5/2.8 | 8.9/30 | 1.5/0.3/0.0/0.7 · 0.0/0.0/0.0/0.0 |
| 1 | focus | spread | 60 | 87% | 13/16/24 | 16.1 | 0 | 1.3/4.8 | 287/167 | 162/264 | 0/0 | 0.74/0.69 | 54/52 | 15/15 | 2.4/0.9/0.4 · 3.1/1.5/1.0 | 0.1/0.2 | 0.9/1.9 | 10.5/23 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 1 | focus | default | 60 | 68% | 13/18/25 | 18.3 | 0 | 3.1/4.4 | 265/209 | 194/245 | 0/0 | 0.74/0.72 | 57/56 | 16/15 | 2.0/0.8/0.6 · 2.1/1.2/1.0 | 0.1/0.1 | 1.8/1.9 | 9.1/27 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 1 | focus | casters | 60 | 63% | 15/20/32 | 21.1 | 0 | 3.4/4.4 | 264/224 | 206/244 | 0/0 | 0.71/0.73 | 55/53 | 16/16 | 2.1/0.7/0.8 · 2.4/1.1/1.1 | 0.1/0.1 | 1.8/2.4 | 8.7/24 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | spread | spread | 60 | 52% | 12/14/18 | 14.2 | 0 | 2.8/3.4 | 275/256 | 243/258 | 0/0 | 0.82/0.80 | 54/54 | 17/15 | 2.5/1.4/1.1 · 3.1/1.3/0.8 | 0.1/0.1 | 1.9/1.9 | 12.6/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | spread | default | 60 | 18% | 13/15/21 | 19.5 | 0 | 4.7/1.8 | 219/297 | 276/210 | 0/0 | 0.82/0.83 | 54/54 | 17/15 | 3.3/1.3/1.2 · 2.7/1.2/0.6 | 0.1/0.2 | 2.5/1.5 | 11.6/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | spread | casters | 60 | 15% | 12/15/19 | 17.2 | 0 | 4.7/1.7 | 219/301 | 280/210 | 0/0 | 0.79/0.84 | 57/54 | 15/14 | 3.0/1.4/1.0 · 2.2/1.1/0.8 | 0.1/0.1 | 2.1/2.0 | 12.3/25 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | kit | spread | 60 | 65% | 13/17/24 | 18.6 | 0 | 2.5/3.9 | 277/255 | 209/256 | 30/0 | 0.66/0.74 | 55/55 | 24/17 | 2.2/1.1/1.0 · 2.7/1.1/1.1 | 0.0/0.1 | 2.3/1.3 | 12.4/31 | 3.1/1.3/0.0/1.1 · 0.0/0.0/0.0/0.0 |
| 5 | default | spread | 60 | 88% | 14/18/23 | 18.5 | 0 | 1.1/4.8 | 304/215 | 136/280 | 64/0 | 0.63/0.77 | 54/54 | 24/13 | 2.5/1.6/0.6 · 3.4/1.0/1.2 | 0.1/0.1 | 0.9/3.1 | 12.4/30 | 5.1/3.8/0.0/0.7 · 0.0/0.0/0.0/0.0 |
| 5 | default | default | 60 | 40% | 17/22/31 | 22.8 | 0 | 3.9/3.9 | 258/298 | 249/238 | 26/0 | 0.61/0.82 | 56/56 | 25/18 | 2.8/0.9/1.0 · 2.4/0.7/1.0 | 0.1/0.1 | 2.0/4.2 | 10.5/27 | 4.6/1.9/0.0/2.0 · 0.0/0.0/0.0/0.0 |
| 5 | default | casters | 60 | 45% | 15/21/28 | 21.9 | 0 | 3.9/3.8 | 253/275 | 249/233 | 5/0 | 0.65/0.81 | 55/54 | 28/16 | 2.9/1.1/0.7 · 2.2/0.5/1.1 | 0.1/0.1 | 1.9/3.4 | 10.3/28 | 1.7/0.4/0.0/0.9 · 0.0/0.0/0.0/0.0 |
| 5 | focus | spread | 60 | 100% | 12/16/20 | 15.9 | 0 | 1.3/5.0 | 312/185 | 178/290 | 0/0 | 0.83/0.79 | 54/53 | 16/16 | 2.2/1.0/0.5 · 3.6/1.5/1.2 | 0.1/0.1 | 1.2/2.7 | 12.0/22 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | focus | default | 60 | 53% | 14/19/25 | 18.9 | 0 | 3.8/4.0 | 262/259 | 240/244 | 0/0 | 0.81/0.83 | 54/57 | 16/16 | 2.0/1.0/1.0 · 2.6/0.8/1.0 | 0.1/0.1 | 1.6/3.8 | 9.7/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 5 | focus | casters | 60 | 58% | 15/20/27 | 19.4 | 0 | 3.6/4.3 | 283/252 | 235/262 | 0/0 | 0.80/0.83 | 56/55 | 17/16 | 2.8/0.9/1.1 · 2.9/1.0/1.2 | 0.1/0.1 | 1.6/3.3 | 9.7/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | spread | spread | 60 | 47% | 12/15/19 | 14.9 | 0 | 3.1/3.2 | 290/283 | 269/275 | 0/0 | 0.83/0.81 | 55/54 | 17/16 | 3.2/1.4/1.1 · 3.0/1.2/1.1 | 0.1/0.1 | 2.3/2.4 | 12.7/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | spread | default | 60 | 18% | 13/17/21 | 20.0 | 0 | 4.6/1.8 | 228/327 | 304/220 | 0/0 | 0.80/0.83 | 55/54 | 15/16 | 3.3/1.4/1.2 · 2.4/1.3/0.5 | 0.1/0.1 | 2.1/2.4 | 11.4/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | spread | casters | 60 | 13% | 13/17/21 | 18.9 | 0 | 4.7/1.7 | 236/327 | 306/228 | 0/0 | 0.79/0.84 | 55/53 | 17/13 | 3.6/1.2/1.4 · 2.3/1.4/0.8 | 0.2/0.1 | 2.8/1.9 | 11.8/25 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | kit | spread | 60 | 87% | 16/18/22 | 18.5 | 0 | 0.8/4.6 | 324/239 | 145/300 | 80/0 | 0.64/0.73 | 56/55 | 24/15 | 2.5/0.8/1.3 · 3.1/1.6/1.1 | 0.1/0.1 | 2.3/1.3 | 13.3/31 | 4.5/3.6/0.0/0.6 · 0.0/0.0/0.0/0.0 |
| 10 | default | spread | 60 | 100% | 15/19/24 | 19.7 | 0 | 0.5/5.0 | 344/209 | 124/315 | 77/0 | 0.64/0.75 | 55/53 | 25/14 | 3.6/1.1/0.4 · 3.2/1.1/1.2 | 0.1/0.1 | 1.6/2.9 | 12.4/32 | 5.1/4.6/0.0/0.1 · 0.0/0.0/0.0/0.0 |
| 10 | default | default | 60 | 55% | 17/23/37 | 23.9 | 0 | 3.3/4.2 | 301/290 | 234/278 | 35/0 | 0.62/0.80 | 56/55 | 25/16 | 2.5/1.0/1.2 · 2.2/1.0/1.4 | 0.1/0.1 | 1.9/3.0 | 10.4/27 | 5.1/2.4/0.0/2.0 · 0.0/0.0/0.0/0.0 |
| 10 | default | casters | 60 | 75% | 17/23/35 | 24.0 | 0 | 3.2/4.6 | 323/258 | 240/297 | 5/0 | 0.66/0.79 | 57/55 | 28/16 | 3.0/1.1/1.1 · 2.7/1.1/1.2 | 0.1/0.1 | 2.1/2.8 | 9.7/27 | 1.7/0.3/0.0/0.8 · 0.0/0.0/0.0/0.0 |
| 10 | focus | spread | 60 | 97% | 13/16/21 | 16.3 | 0 | 0.9/4.9 | 334/181 | 177/312 | 0/0 | 0.83/0.78 | 54/54 | 16/14 | 2.6/0.9/0.3 · 3.4/1.3/1.4 | 0.1/0.1 | 1.2/2.3 | 11.8/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | focus | default | 60 | 75% | 16/22/32 | 23.1 | 0 | 3.3/4.6 | 321/258 | 242/299 | 0/0 | 0.82/0.83 | 55/54 | 15/16 | 2.6/0.9/1.0 · 3.0/1.4/1.1 | 0.1/0.1 | 2.0/2.7 | 8.7/27 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 10 | focus | casters | 60 | 75% | 16/22/30 | 21.8 | 0 | 3.2/4.7 | 326/256 | 239/302 | 0/0 | 0.79/0.82 | 57/55 | 18/16 | 2.5/0.9/0.9 · 2.6/1.4/1.3 | 0.2/0.1 | 2.5/2.9 | 9.5/27 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | spread | spread | 60 | 48% | 11/13/16 | 13.3 | 0 | 3.2/3.1 | 346/348 | 329/325 | 0/0 | 0.93/0.91 | 53/53 | 15/16 | 2.5/1.5/1.1 · 2.9/1.3/0.9 | 0.1/0.2 | 1.9/2.6 | 14.1/32 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | spread | default | 60 | 27% | 12/15/20 | 16.9 | 0 | 4.4/2.2 | 295/381 | 352/280 | 0/0 | 0.92/0.95 | 54/48 | 16/16 | 3.2/1.3/1.2 · 3.0/1.2/0.7 | 0.3/0.1 | 2.6/1.1 | 13.0/28 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | spread | casters | 60 | 32% | 12/15/20 | 18.4 | 0 | 4.4/2.3 | 299/381 | 353/284 | 0/0 | 0.89/0.95 | 53/48 | 17/16 | 4.1/1.4/1.3 · 2.8/1.1/0.9 | 0.2/0.2 | 3.3/2.1 | 12.9/29 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | kit | spread | 60 | 80% | 13/16/23 | 16.3 | 0 | 1.7/4.5 | 388/315 | 225/357 | 68/0 | 0.73/0.82 | 54/54 | 22/16 | 2.9/1.2/0.9 · 2.8/1.6/0.8 | 0.1/0.1 | 2.2/1.7 | 13.4/36 | 3.3/2.1/0.0/0.8 · 0.0/0.0/0.0/0.0 |
| 30 | default | spread | 60 | 93% | 13/16/23 | 16.9 | 0 | 1.3/4.9 | 411/282 | 184/374 | 79/0 | 0.73/0.89 | 52/54 | 22/15 | 2.8/1.5/0.6 · 3.9/0.9/1.1 | 0.1/0.2 | 1.2/3.6 | 13.5/30 | 4.6/3.5/0.0/0.3 · 0.0/0.0/0.0/0.0 |
| 30 | default | default | 60 | 57% | 15/20/25 | 20.2 | 0 | 3.9/4.2 | 360/352 | 326/331 | 2/0 | 0.78/0.91 | 57/57 | 21/14 | 2.8/1.0/0.7 · 2.5/0.8/1.2 | 0.1/0.1 | 2.3/3.7 | 11.0/28 | 1.3/0.1/0.0/0.7 · 0.0/0.0/0.0/0.0 |
| 30 | default | casters | 60 | 60% | 15/18/25 | 17.8 | 0 | 3.5/4.3 | 371/329 | 305/336 | 1/0 | 0.76/0.92 | 57/54 | 24/15 | 2.8/1.0/0.8 · 2.4/0.7/1.3 | 0.2/0.1 | 1.7/3.3 | 11.3/27 | 1.2/0.0/0.0/0.6 · 0.0/0.0/0.0/0.0 |
| 30 | focus | spread | 60 | 82% | 11/15/21 | 15.1 | 0 | 2.2/4.8 | 392/280 | 266/365 | 0/0 | 0.95/0.92 | 49/57 | 15/15 | 2.5/1.0/0.5 · 3.3/1.2/1.2 | 0.1/0.1 | 0.9/3.5 | 12.6/25 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | focus | default | 60 | 48% | 13/17/22 | 16.8 | 0 | 4.0/4.2 | 349/347 | 323/326 | 0/0 | 0.92/0.95 | 56/59 | 15/15 | 2.5/0.9/0.9 · 2.6/0.8/1.3 | 0.1/0.1 | 2.0/3.1 | 10.9/26 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 30 | focus | casters | 60 | 55% | 13/16/21 | 15.3 | 0 | 3.5/4.2 | 358/318 | 297/333 | 0/0 | 0.88/0.94 | 57/57 | 16/14 | 2.4/0.8/0.9 · 2.4/0.8/1.0 | 0.1/0.1 | 2.0/3.0 | 11.4/29 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | spread | spread | 60 | 57% | 9/11/15 | 11.6 | 0 | 2.9/3.5 | 412/383 | 363/387 | 0/0 | 1.07/1.05 | 52/52 | 16/14 | 3.0/1.3/0.7 · 2.7/1.4/0.7 | 0.2/0.2 | 2.4/1.7 | 15.6/32 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | spread | default | 60 | 35% | 10/13/17 | 14.2 | 0 | 4.3/2.6 | 349/424 | 393/330 | 0/0 | 1.06/1.10 | 55/45 | 14/16 | 3.3/1.3/1.3 · 2.6/1.2/0.6 | 0.2/0.1 | 2.0/2.0 | 14.6/32 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | spread | casters | 60 | 30% | 10/13/16 | 13.3 | 0 | 4.2/2.4 | 349/427 | 395/332 | 0/0 | 1.01/1.11 | 56/45 | 16/16 | 3.5/1.5/1.1 · 2.6/1.2/0.6 | 0.2/0.1 | 2.2/1.6 | 14.4/34 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | kit | spread | 60 | 98% | 10/12/15 | 12.7 | 0 | 0.6/5.0 | 474/289 | 190/429 | 79/0 | 0.86/0.90 | 54/50 | 22/14 | 2.8/1.1/0.7 · 2.8/1.1/0.6 | 0.2/0.1 | 1.8/1.1 | 16.3/34 | 3.0/2.1/0.0/0.5 · 0.0/0.0/0.0/0.0 |
| 60 | default | spread | 60 | 93% | 11/14/19 | 14.3 | 0 | 1.5/4.9 | 479/330 | 231/424 | 72/0 | 0.86/1.02 | 53/56 | 20/15 | 2.5/1.1/0.6 · 3.0/0.9/1.3 | 0.2/0.2 | 0.8/3.3 | 15.4/31 | 4.0/2.5/0.0/0.6 · 0.0/0.0/0.0/0.0 |
| 60 | default | default | 60 | 42% | 12/16/20 | 15.0 | 0 | 4.2/4.2 | 414/415 | 380/373 | 1/0 | 0.92/1.07 | 59/59 | 20/16 | 2.6/0.8/0.8 · 2.6/0.7/0.9 | 0.2/0.1 | 1.3/3.6 | 12.2/31 | 1.0/0.0/0.0/0.6 · 0.0/0.0/0.0/0.0 |
| 60 | default | casters | 60 | 52% | 13/15/20 | 14.6 | 0 | 3.8/4.3 | 437/391 | 365/389 | 0/0 | 0.88/1.06 | 59/59 | 25/15 | 2.8/1.0/0.5 · 2.9/0.5/0.7 | 0.1/0.1 | 2.9/3.1 | 12.6/30 | 0.8/0.0/0.0/0.5 · 0.0/0.0/0.0/0.0 |
| 60 | focus | spread | 60 | 68% | 11/14/17 | 13.7 | 0 | 2.8/4.5 | 435/354 | 332/401 | 0/0 | 1.09/1.08 | 45/57 | 15/15 | 2.1/1.1/0.7 · 3.4/1.1/1.1 | 0.1/0.1 | 1.0/2.8 | 14.4/31 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | focus | default | 60 | 62% | 11/14/19 | 14.5 | 0 | 3.7/4.4 | 427/378 | 348/394 | 0/0 | 1.07/1.09 | 58/58 | 16/15 | 2.1/0.8/0.9 · 2.9/0.9/0.8 | 0.1/0.2 | 1.7/3.6 | 12.8/32 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 60 | focus | casters | 60 | 60% | 12/15/19 | 14.9 | 0 | 3.6/4.4 | 432/388 | 359/398 | 0/0 | 1.05/1.10 | 56/59 | 16/14 | 2.5/0.9/0.8 · 2.8/0.9/1.0 | 0.1/0.0 | 2.2/3.1 | 12.6/30 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | spread | spread | 60 | 53% | 9/11/15 | 13.0 | 0 | 3.2/3.0 | 452/461 | 436/425 | 0/0 | 1.12/1.11 | 50/50 | 16/17 | 3.3/1.7/0.8 · 3.2/2.1/0.9 | 0.1/0.1 | 2.1/2.5 | 16.3/31 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | spread | default | 60 | 42% | 10/13/18 | 14.8 | 0 | 4.2/2.7 | 414/478 | 445/393 | 0/0 | 1.12/1.15 | 57/46 | 15/16 | 2.7/1.1/0.9 · 2.8/1.0/0.6 | 0.1/0.1 | 2.7/2.1 | 14.9/37 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | spread | casters | 60 | 32% | 10/13/17 | 13.1 | 0 | 4.2/2.4 | 411/481 | 447/392 | 0/0 | 1.07/1.16 | 56/45 | 16/15 | 3.4/1.5/1.1 · 2.9/1.4/0.5 | 0.1/0.2 | 2.7/1.8 | 15.5/37 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | kit | spread | 60 | 97% | 11/12/15 | 12.7 | 0 | 0.5/4.9 | 545/354 | 209/489 | 116/0 | 0.90/0.99 | 54/52 | 20/16 | 2.6/1.1/0.7 · 3.2/1.6/0.6 | 0.2/0.1 | 1.5/2.0 | 17.3/35 | 3.4/2.5/0.0/0.4 · 0.0/0.0/0.0/0.0 |
| 100 | default | spread | 60 | 93% | 10/13/17 | 13.1 | 0 | 0.8/4.9 | 568/334 | 200/493 | 106/0 | 0.89/1.06 | 55/54 | 20/15 | 3.5/1.0/0.3 · 3.2/0.9/1.1 | 0.1/0.2 | 1.6/2.9 | 16.2/33 | 3.8/3.0/0.0/0.3 · 0.0/0.0/0.0/0.0 |
| 100 | default | default | 60 | 55% | 12/17/23 | 17.2 | 0 | 4.0/4.3 | 514/464 | 431/454 | 0/0 | 1.00/1.14 | 58/58 | 18/16 | 3.5/1.0/0.7 · 2.1/1.2/1.0 | 0.1/0.1 | 1.8/2.4 | 12.1/39 | 0.9/0.0/0.0/0.4 · 0.0/0.0/0.0/0.0 |
| 100 | default | casters | 60 | 60% | 11/17/23 | 16.6 | 0 | 3.8/4.5 | 529/447 | 416/463 | 1/0 | 0.94/1.12 | 58/58 | 21/17 | 2.9/1.0/0.6 · 3.0/0.9/0.9 | 0.1/0.1 | 1.7/2.6 | 12.3/38 | 0.8/0.0/0.0/0.4 · 0.0/0.0/0.0/0.0 |
| 100 | focus | spread | 60 | 85% | 10/13/21 | 12.9 | 0 | 1.6/4.8 | 520/323 | 310/483 | 0/0 | 1.15/1.10 | 46/52 | 14/14 | 3.2/1.2/0.4 · 3.5/1.2/1.1 | 0.2/0.2 | 1.2/2.2 | 14.8/33 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | focus | default | 60 | 68% | 11/15/21 | 14.7 | 0 | 3.5/4.5 | 502/412 | 385/468 | 0/0 | 1.14/1.14 | 57/58 | 15/15 | 2.5/1.0/0.8 · 3.5/1.2/0.9 | 0.2/0.2 | 1.9/2.0 | 12.7/36 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |
| 100 | focus | casters | 60 | 52% | 12/15/21 | 15.4 | 0 | 3.8/4.3 | 482/454 | 422/449 | 0/0 | 1.07/1.16 | 57/59 | 14/16 | 2.8/0.8/0.6 · 2.6/1.3/0.9 | 0.1/0.2 | 1.5/3.2 | 12.4/35 | 0.0/0.0/0.0/0.0 · 0.0/0.0/0.0/0.0 |

## Coordinated table (TestBalanceCoordinated, 60 fights a cell)

| level | company | enemy | fights | company wins | rounds p10/median/p90 | won-fight rounds mean | stalls | fallen company/enemy | damage company/enemy | net HP lost company/enemy | healing company/enemy | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | lines/round mean/peak | casts begun/cast/fizzled/broken company · enemy |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | default | default | 60 | 48% | 16/22/32 | 25.7 | 0 | 3.6/3.7 | 227/278 | 214/210 | 40/0 | 0.52/0.72 | 55/56 | 23/17 | 2.1/0.9/1.1 · 1.8/1.0/1.2 | 0.1/0.0 | 1.6/2.8 | 10.1/27 | 5.7/3.1/0.0/1.6 · 0.0/0.0/0.0/0.0 |
| 1 | default | roles1 | 60 | 53% | 17/23/32 | 23.5 | 0 | 3.3/4.0 | 251/266 | 204/228 | 41/1 | 0.54/0.70 | 56/57 | 25/14 | 2.3/1.1/1.2 · 1.8/0.9/1.0 | 0.1/0.0 | 1.7/2.8 | 9.9/28 | 5.5/3.1/0.0/1.6 · 1.1/0.1/0.0/0.5 |
| 1 | default | roles2 | 60 | 33% | 17/21/33 | 23.8 | 0 | 4.2/3.2 | 214/277 | 235/186 | 19/13 | 0.54/0.66 | 55/56 | 23/15 | 2.5/0.8/0.9 · 2.5/0.7/0.8 | 0.1/0.1 | 1.5/3.1 | 10.0/28 | 3.9/1.5/0.0/1.9 · 3.0/1.0/0.0/0.9 |
| 1 | default | roles3 | 60 | 22% | 16/22/33 | 25.4 | 0 | 4.5/2.7 | 205/274 | 248/161 | 4/30 | 0.58/0.62 | 53/56 | 25/15 | 2.8/0.7/1.1 · 3.6/0.5/0.8 | 0.0/0.1 | 1.7/3.0 | 10.0/26 | 1.5/0.3/0.0/0.5 · 4.7/2.3/0.0/0.9 |
| 1 | tactics | roles2 | 60 | 53% | 15/21/31 | 21.9 | 0 | 3.0/3.6 | 238/269 | 188/205 | 57/17 | 0.52/0.67 | 55/54 | 22/15 | 3.1/0.8/1.2 · 2.6/0.6/1.0 | 0.2/0.2 | 1.7/3.2 | 12.2/32 | 7.0/4.3/0.0/1.6 · 3.5/1.2/0.0/0.8 |
| 1 | tactics | roles3 | 60 | 57% | 15/21/31 | 21.1 | 0 | 2.9/3.7 | 250/236 | 181/210 | 39/22 | 0.56/0.62 | 54/53 | 27/14 | 3.2/0.6/1.1 · 3.4/0.7/0.9 | 0.2/0.1 | 1.8/2.9 | 11.9/28 | 5.0/2.9/0.0/1.0 · 4.2/1.6/0.0/1.0 |
| 5 | default | default | 60 | 48% | 16/22/28 | 23.5 | 0 | 3.7/3.9 | 256/294 | 237/236 | 33/0 | 0.61/0.81 | 57/57 | 23/17 | 2.8/1.0/0.8 · 2.5/0.8/1.2 | 0.0/0.1 | 1.6/4.3 | 11.0/29 | 4.9/2.4/0.0/1.9 · 0.0/0.0/0.0/0.0 |
| 5 | default | roles1 | 60 | 47% | 18/22/29 | 22.7 | 0 | 3.8/4.0 | 263/288 | 240/242 | 28/1 | 0.61/0.79 | 55/55 | 26/16 | 2.6/1.2/1.2 · 2.5/0.5/0.9 | 0.1/0.1 | 1.7/4.2 | 10.8/31 | 4.6/2.0/0.0/1.9 · 1.3/0.1/0.0/0.6 |
| 5 | default | roles2 | 60 | 33% | 17/23/32 | 24.9 | 0 | 4.1/3.3 | 241/300 | 254/209 | 22/15 | 0.61/0.72 | 54/56 | 24/15 | 3.0/1.2/1.1 · 3.2/0.7/1.0 | 0.1/0.2 | 2.5/3.2 | 10.8/29 | 4.3/1.7/0.0/2.1 · 3.5/1.2/0.0/1.1 |
| 5 | default | roles3 | 60 | 37% | 18/23/28 | 23.0 | 0 | 4.3/3.2 | 246/281 | 256/201 | 2/31 | 0.67/0.67 | 54/55 | 23/16 | 2.9/1.0/1.1 · 4.2/0.5/0.9 | 0.1/0.2 | 2.0/3.4 | 10.4/27 | 1.5/0.2/0.0/0.7 · 4.9/2.4/0.0/1.1 |
| 5 | tactics | roles2 | 60 | 70% | 17/21/29 | 21.3 | 0 | 2.3/4.4 | 292/258 | 168/266 | 69/5 | 0.60/0.74 | 55/54 | 23/15 | 3.3/0.5/1.3 · 3.7/0.7/1.2 | 0.2/0.1 | 2.3/4.2 | 13.4/31 | 7.3/5.3/0.0/1.3 · 2.8/0.4/0.0/1.3 |
| 5 | tactics | roles3 | 60 | 60% | 15/20/29 | 21.3 | 0 | 2.9/3.8 | 268/247 | 197/229 | 32/23 | 0.64/0.67 | 53/53 | 25/16 | 3.2/0.8/1.0 · 3.7/0.8/1.2 | 0.0/0.1 | 1.9/3.5 | 12.8/31 | 4.3/2.4/0.0/1.1 · 4.3/1.8/0.0/1.1 |
| 10 | default | default | 60 | 65% | 18/23/37 | 24.0 | 0 | 3.0/4.4 | 308/277 | 219/287 | 37/0 | 0.62/0.79 | 57/54 | 26/15 | 2.4/1.1/1.2 · 2.5/0.8/1.2 | 0.1/0.1 | 2.2/3.4 | 10.3/31 | 5.0/2.5/0.0/2.0 · 0.0/0.0/0.0/0.0 |
| 10 | default | roles1 | 60 | 57% | 17/23/35 | 25.4 | 0 | 3.5/4.2 | 296/291 | 241/273 | 30/0 | 0.62/0.79 | 56/56 | 24/15 | 2.9/0.9/1.2 · 2.0/1.1/1.3 | 0.1/0.1 | 2.0/2.8 | 10.3/27 | 4.9/2.1/0.0/2.2 · 0.7/0.0/0.0/0.1 |
| 10 | default | roles2 | 60 | 48% | 20/25/32 | 26.6 | 0 | 4.0/3.9 | 278/314 | 267/258 | 25/0 | 0.62/0.77 | 54/55 | 26/16 | 3.4/1.1/1.2 · 3.1/0.8/1.0 | 0.1/0.1 | 2.7/3.7 | 10.7/29 | 4.8/1.8/0.0/2.4 · 1.1/0.0/0.0/0.7 |
| 10 | default | roles3 | 60 | 32% | 18/23/31 | 26.3 | 0 | 4.4/3.5 | 260/312 | 291/240 | 1/1 | 0.68/0.76 | 56/56 | 23/14 | 3.0/1.1/1.3 · 3.0/0.6/0.8 | 0.2/0.1 | 2.6/3.6 | 9.9/28 | 1.3/0.1/0.0/0.6 · 1.8/0.1/0.0/1.1 |
| 10 | tactics | roles2 | 60 | 83% | 18/21/29 | 22.6 | 0 | 1.8/4.6 | 320/258 | 157/295 | 82/0 | 0.60/0.77 | 54/52 | 25/15 | 3.9/0.8/1.3 · 3.5/1.1/1.2 | 0.1/0.1 | 2.4/3.5 | 13.4/35 | 7.8/5.9/0.0/1.0 · 1.2/0.0/0.0/0.7 |
| 10 | tactics | roles3 | 60 | 47% | 16/21/31 | 22.7 | 0 | 3.6/3.6 | 269/286 | 243/244 | 23/4 | 0.64/0.75 | 52/54 | 26/14 | 4.2/1.0/0.6 · 4.4/0.5/1.0 | 0.2/0.2 | 2.5/2.7 | 12.1/31 | 4.2/1.6/0.0/1.2 · 2.1/0.3/0.0/1.2 |
| 30 | default | default | 60 | 53% | 14/19/23 | 18.5 | 0 | 3.8/4.2 | 361/342 | 314/329 | 3/0 | 0.78/0.91 | 59/56 | 21/15 | 2.4/1.0/0.8 · 2.1/0.8/0.9 | 0.1/0.1 | 1.7/3.2 | 11.1/30 | 1.4/0.2/0.0/0.6 · 0.0/0.0/0.0/0.0 |
| 30 | default | roles1 | 60 | 50% | 15/18/25 | 18.5 | 0 | 3.8/4.2 | 364/346 | 314/329 | 6/0 | 0.79/0.91 | 57/56 | 21/15 | 2.6/1.2/0.9 · 3.0/0.6/1.0 | 0.1/0.1 | 1.9/3.9 | 11.0/31 | 1.4/0.3/0.0/0.6 · 0.2/0.0/0.0/0.1 |
| 30 | default | roles2 | 60 | 32% | 16/20/24 | 21.4 | 0 | 4.4/3.7 | 330/380 | 348/301 | 2/0 | 0.78/0.89 | 57/57 | 19/15 | 2.8/1.0/0.8 · 3.0/0.5/1.0 | 0.1/0.2 | 1.6/3.4 | 10.8/28 | 1.0/0.1/0.0/0.5 · 0.8/0.0/0.0/0.3 |
| 30 | default | roles3 | 60 | 47% | 14/20/26 | 20.8 | 0 | 4.3/4.0 | 348/363 | 334/317 | 1/1 | 0.80/0.85 | 58/56 | 21/15 | 2.6/0.9/0.8 · 2.6/0.7/0.9 | 0.1/0.1 | 2.2/3.2 | 10.6/28 | 0.8/0.1/0.0/0.4 · 1.1/0.1/0.0/0.4 |
| 30 | tactics | roles2 | 60 | 57% | 13/17/25 | 17.1 | 0 | 3.3/4.2 | 362/320 | 272/329 | 25/1 | 0.75/0.88 | 55/55 | 25/14 | 3.8/0.7/0.7 · 2.8/0.7/0.8 | 0.2/0.2 | 2.2/3.1 | 13.3/31 | 3.6/1.4/0.0/1.0 · 0.5/0.1/0.0/0.2 |
| 30 | tactics | roles3 | 60 | 53% | 14/17/23 | 17.2 | 0 | 3.4/4.1 | 361/325 | 280/326 | 21/1 | 0.76/0.84 | 54/55 | 26/15 | 3.9/0.7/0.6 · 2.9/0.6/0.9 | 0.3/0.1 | 2.5/3.0 | 13.3/31 | 2.9/1.1/0.0/0.9 · 1.1/0.0/0.0/0.5 |
