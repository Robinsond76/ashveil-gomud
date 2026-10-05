# Phase 35a2 measurements

Real-round fights through the balance harness, 100 a cell, with the shipped
config and the archetype overlay:

```sh
ASHVEIL_BALANCE=1 go test ./modules/company -run 'TestBalance5v5$' -v
ASHVEIL_BALANCE=1 go test ./modules/company -run 'TestBalanceZoneBands$' -v
ASHVEIL_BALANCE=1 go test ./modules/company -run 'TestBalance(SkillWins|WarriorsAreTheTanks|Mismatches|SidesStayEven|StatusesLand|Coordinated)$' -v
```

No cell stalled. The unit-level acceptance rows (1 hit size, 2 small HP,
8 wiring, 9 help) run in the ordinary suite and pass.

## Tuned values

The design's numbers were starting values. Two changed:

| Setting | Design | Shipped | Why |
|---|---|---|---|
| `SkillEdgeSpan` | 20 | **16** | At 20, a level-10 company beat two level-20 foes 44% of the time (acceptance 3 wants at most 30%). At 16 it wins 12%. The untrained −10 is then 0.625 of a full edge rather than half: still the significant penalty the owner asked for. |
| `BlockChanceEven` | 15 | **20** | Keeps the shield the steadiest defense: an iron shield blocks 30% at even skill against a sword's 17% parry. It barely moves acceptance 5 (a warrior took 0.83 of a rogue's damage at 15, 0.80–0.84 over two runs at 20); 25 reached 0.71–0.75, still short, and was reverted. |

Three items were retagged **medium** so light-trained classes don't get heavy
protection for free (rangers keep them): wolf pelt (20042, armor 18), snow
wolf mane (20041) and spider exoskeleton (20023).

## Acceptance rows

| Row | Target | Measured | Result |
|---|---|---|---|
| 3. Skill wins | L20 vs 3×L10 ≥ 99%, ≤ 15% HP lost; L10 vs 2×L20 ≤ 30% | 100%, 5.3% lost; 12% | Pass |
| 4. Equal fights stay short | spread mirror median 8–12 rounds | 19 / 18 / 20 / 16 / 14 at L1 / 5 / 10 / 30 / 60 (14 at 100) | **Miss** |
| 5. Warriors are the tanks | warrior ≤ 0.7× rogue and ≤ 0.5× wizard per swing; heavy rogue tempo ≤ 0.8× light | 3.02 vs 3.59 (**0.84×**) and 6.81 (0.44×); tempo 0.82 vs 1.40 (0.59×) | **Miss** on the rogue ratio |
| 6. 30g6 rows | L30 vs L10 ≥ 95%, no member lost in most; L15 vs L10 > 50% and loses at least once | 100%, none lost; L15 **100%** | **Miss** on "L15 can lose" |
| 7. Zone bands | level impact §4 table | HP lost below; win rates not captured this run | Report |
| 30g6 mirror parity, kit, enemy targeting | 35–65%; kit no handicap; targeting costs more | all pass at every level | Pass |
| 33i2 coordinated enemies | with base wins ≥ 60%, each tier keeps company wins ≥ 50% | L10: tier 2 37%, tier 3 26% | **Miss** |
| Sides stay even, statuses land | as before | pass | Pass |

### Why the misses

- **Fight length (4).** Dodge and parry are now two-sided around 12% at even
  skill (before, they started at a 3% floor and only grew), and a landed blow
  is a fifth of a fighter's health by design (row 1). Per fight the mirror
  sees 1.9/1.7/1.2 blocks, parries and dodges a side against 0.8/0.5/0.3 in
  30g6, at a 42% landed-hit rate against 45%. Rows 1 and 4 cannot both be met
  by tuning chances alone: 8–12 rounds would need roughly double the damage
  a round, beyond the 95% hit cap.
- **Tank ratio (5).** A rogue's light armor and higher Evasion (33 to the
  warrior's 30 at level 30) dodge often; the warrior's block and 55 armor
  stop more, but not 30% more.
- **L15 vs L10 (6).** Five levels is a 5-point skill gap (0.31 of an edge at
  span 16), plus the 30g6 stat edge: the intended result of "skill over hit
  points", which conflicts with the old row.
- **Coordinated enemies.** Longer fights give enemy healers and guardians
  more turns; heals are now sized against small HP pools. 35b reworks heals,
  mana and spell power and is the natural place to retune this.

These are open owner questions (see `docs/PROJECT_STATUS.md`); no further
tuning is in 35a2.

## 5v5 table (TestBalance5v5)

| level | company | enemy | fights | company wins | rounds p10/median/p90 | won-fight rounds mean | stalls | fallen company/enemy | damage company/enemy | net HP lost company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | lines/round mean/peak |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | spread | spread | 100 | 60% | 15/19/26 | 20.1 | 0 | 2.8/3.6 | 258/239 | 225/243 | 0 | 0.76/0.75 | 42/42 | 15/13 | 1.9/1.7/1.2 · 1.9/1.9/1.2 | 0.1/0.1 | 1.8/1.6 | 10.7/28 |
| 1 | spread | default | 100 | 24% | 16/20/29 | 26.1 | 0 | 4.5/2.1 | 206/274 | 254/195 | 0 | 0.74/0.76 | 42/41 | 15/16 | 2.0/1.5/1.8 · 1.6/1.3/1.0 | 0.1/0.1 | 1.8/1.6 | 9.7/26 |
| 1 | spread | casters | 100 | 28% | 16/20/27 | 23.9 | 0 | 4.4/2.2 | 215/273 | 252/204 | 0 | 0.73/0.76 | 44/41 | 16/16 | 2.0/2.0/1.6 · 1.5/1.3/1.1 | 0.1/0.1 | 2.2/1.1 | 10.3/27 |
| 1 | kit | spread | 100 | 69% | 17/21/29 | 21.9 | 0 | 2.1/4.0 | 261/217 | 198/241 | 8 | 0.64/0.68 | 43/41 | 23/16 | 1.9/1.1/1.3 · 1.6/1.5/1.7 | 0.1/0.0 | 1.9/1.2 | 11.1/28 |
| 1 | default | spread | 100 | 82% | 17/23/29 | 22.5 | 0 | 1.6/4.7 | 280/193 | 174/257 | 10 | 0.65/0.70 | 42/42 | 23/15 | 1.8/1.5/0.8 · 1.8/1.5/1.8 | 0.1/0.1 | 1.0/2.2 | 10.3/25 |
| 1 | default | default | 100 | 51% | 18/27/41 | 29.5 | 0 | 3.7/3.9 | 242/253 | 225/221 | 7 | 0.60/0.72 | 44/44 | 24/14 | 1.6/1.3/1.6 · 1.1/1.1/1.5 | 0.1/0.0 | 1.9/2.5 | 8.7/28 |
| 1 | default | casters | 100 | 53% | 20/28/39 | 28.3 | 0 | 3.6/4.0 | 248/238 | 217/228 | 2 | 0.60/0.72 | 44/42 | 26/15 | 1.6/1.5/1.2 · 1.3/0.9/1.7 | 0.1/0.0 | 1.7/2.0 | 8.8/26 |
| 1 | focus | spread | 100 | 89% | 16/20/27 | 20.6 | 0 | 1.4/4.8 | 286/180 | 172/262 | 0 | 0.76/0.72 | 42/42 | 15/16 | 1.7/1.4/0.5 · 2.1/1.8/1.8 | 0.1/0.0 | 1.1/2.2 | 10.1/23 |
| 1 | focus | default | 100 | 64% | 18/25/35 | 26.2 | 0 | 3.4/4.3 | 262/227 | 211/241 | 0 | 0.75/0.75 | 43/42 | 15/16 | 1.1/1.2/1.4 · 1.5/1.5/1.4 | 0.1/0.1 | 2.2/2.7 | 8.2/25 |
| 1 | focus | casters | 100 | 62% | 18/25/35 | 23.5 | 0 | 3.2/4.4 | 265/213 | 198/245 | 0 | 0.73/0.75 | 42/41 | 17/16 | 1.6/1.4/1.2 · 1.7/1.6/1.8 | 0.1/0.1 | 2.1/2.6 | 8.7/26 |
| 5 | spread | spread | 100 | 62% | 15/18/24 | 19.0 | 0 | 2.7/3.6 | 275/256 | 243/259 | 0 | 0.83/0.81 | 43/42 | 16/16 | 1.8/1.8/1.4 · 2.1/2.0/1.6 | 0.1/0.1 | 2.3/2.1 | 11.8/26 |
| 5 | spread | default | 100 | 11% | 16/19/24 | 24.4 | 0 | 4.8/1.6 | 201/304 | 282/193 | 0 | 0.82/0.84 | 41/42 | 16/16 | 2.0/1.8/1.5 · 1.5/1.7/0.9 | 0.1/0.0 | 2.5/2.4 | 11.1/27 |
| 5 | spread | casters | 100 | 18% | 16/20/26 | 22.4 | 0 | 4.6/1.8 | 213/299 | 277/204 | 0 | 0.80/0.85 | 42/41 | 14/14 | 2.4/1.9/1.9 · 1.9/1.4/1.2 | 0.1/0.1 | 2.7/1.9 | 11.3/26 |
| 5 | kit | spread | 100 | 61% | 16/21/28 | 22.5 | 0 | 2.6/3.7 | 265/255 | 221/248 | 20 | 0.68/0.76 | 43/43 | 24/17 | 2.2/1.5/1.5 · 1.6/1.4/1.6 | 0.1/0.1 | 1.8/1.4 | 12.1/26 |
| 5 | default | spread | 100 | 78% | 18/23/31 | 23.1 | 0 | 1.9/4.7 | 299/229 | 183/277 | 37 | 0.68/0.80 | 42/43 | 24/14 | 1.9/1.6/0.8 · 2.3/1.4/2.1 | 0.1/0.1 | 1.2/3.5 | 11.3/26 |
| 5 | default | default | 100 | 49% | 19/28/37 | 27.1 | 0 | 3.6/3.9 | 258/273 | 234/238 | 19 | 0.65/0.81 | 43/42 | 25/15 | 1.5/1.6/1.8 · 1.8/1.1/2.0 | 0.1/0.1 | 1.9/3.4 | 10.1/31 |
| 5 | default | casters | 100 | 49% | 20/28/37 | 28.4 | 0 | 3.9/4.0 | 267/268 | 247/247 | 2 | 0.68/0.81 | 44/42 | 26/15 | 1.7/1.7/1.5 · 1.4/1.0/1.9 | 0.0/0.0 | 2.7/4.1 | 9.5/27 |
| 5 | focus | spread | 100 | 92% | 15/19/29 | 20.2 | 0 | 1.5/4.9 | 308/190 | 182/287 | 0 | 0.84/0.82 | 41/42 | 15/13 | 1.4/1.8/0.7 · 1.9/2.0/1.8 | 0.1/0.1 | 1.1/2.5 | 10.9/23 |
| 5 | focus | default | 100 | 56% | 19/26/35 | 26.2 | 0 | 3.8/4.2 | 276/258 | 240/257 | 0 | 0.83/0.85 | 42/43 | 15/14 | 1.8/1.5/1.4 · 2.0/1.2/1.8 | 0.1/0.1 | 2.1/2.7 | 8.5/27 |
| 5 | focus | casters | 100 | 57% | 19/25/33 | 24.2 | 0 | 3.5/4.2 | 273/256 | 237/253 | 0 | 0.80/0.85 | 43/43 | 16/16 | 1.9/1.5/1.4 · 1.7/1.1/1.8 | 0.1/0.1 | 2.1/3.4 | 9.3/27 |
| 10 | spread | spread | 100 | 46% | 17/20/25 | 20.7 | 0 | 3.3/3.0 | 282/287 | 271/267 | 0 | 0.83/0.83 | 41/41 | 14/16 | 2.0/2.1/1.9 · 2.5/1.8/1.5 | 0.1/0.1 | 2.1/2.0 | 11.8/25 |
| 10 | spread | default | 100 | 18% | 16/22/29 | 23.4 | 0 | 4.5/2.0 | 240/319 | 297/229 | 0 | 0.83/0.84 | 42/41 | 16/16 | 2.6/1.9/2.0 · 2.1/1.8/1.2 | 0.1/0.1 | 2.4/2.2 | 10.7/25 |
| 10 | spread | casters | 100 | 9% | 16/20/27 | 22.2 | 0 | 4.8/1.2 | 213/329 | 307/206 | 0 | 0.80/0.85 | 41/42 | 16/16 | 2.7/1.9/1.7 · 1.5/1.6/1.2 | 0.1/0.1 | 3.0/2.2 | 11.4/26 |
| 10 | kit | spread | 100 | 86% | 20/24/32 | 24.8 | 0 | 1.1/4.6 | 323/240 | 169/300 | 63 | 0.67/0.74 | 43/42 | 23/15 | 1.9/1.1/1.5 · 2.1/2.1/1.8 | 0.1/0.1 | 2.6/1.2 | 12.1/28 |
| 10 | default | spread | 100 | 95% | 20/25/34 | 25.6 | 0 | 0.9/4.9 | 337/227 | 154/313 | 68 | 0.66/0.78 | 42/42 | 24/16 | 2.5/1.4/0.8 · 2.4/1.5/2.5 | 0.1/0.1 | 1.8/3.5 | 11.5/26 |
| 10 | default | default | 100 | 55% | 22/29/40 | 30.2 | 0 | 3.4/4.0 | 283/290 | 244/263 | 28 | 0.63/0.80 | 43/43 | 25/15 | 1.9/1.0/1.8 · 1.4/1.1/2.2 | 0.1/0.1 | 2.8/3.5 | 10.1/26 |
| 10 | default | casters | 100 | 65% | 22/30/42 | 31.5 | 0 | 3.5/4.3 | 302/271 | 253/281 | 2 | 0.67/0.79 | 45/42 | 27/16 | 2.1/1.4/1.4 · 1.5/1.2/2.1 | 0.1/0.1 | 2.8/3.1 | 9.3/27 |
| 10 | focus | spread | 100 | 91% | 17/21/29 | 21.6 | 0 | 1.3/4.9 | 332/199 | 191/309 | 0 | 0.84/0.80 | 41/42 | 16/17 | 1.9/1.4/0.8 · 2.4/2.1/2.0 | 0.1/0.1 | 1.6/2.8 | 10.9/24 |
| 10 | focus | default | 100 | 74% | 19/28/37 | 27.0 | 0 | 3.2/4.6 | 316/251 | 235/295 | 0 | 0.84/0.83 | 42/43 | 16/15 | 1.4/1.2/1.6 · 1.8/2.0/2.0 | 0.1/0.1 | 2.6/3.1 | 8.9/27 |
| 10 | focus | casters | 100 | 67% | 19/27/42 | 26.8 | 0 | 3.1/4.5 | 314/251 | 236/292 | 0 | 0.81/0.84 | 43/42 | 16/16 | 2.1/1.4/1.5 · 2.0/1.9/2.0 | 0.1/0.1 | 2.5/3.3 | 8.9/28 |
| 30 | spread | spread | 100 | 54% | 13/16/21 | 16.6 | 0 | 3.0/3.5 | 355/340 | 320/334 | 0 | 0.95/0.93 | 42/41 | 16/15 | 1.7/1.9/1.5 · 2.0/2.0/1.4 | 0.1/0.1 | 2.6/1.9 | 13.5/30 |
| 30 | spread | default | 100 | 21% | 14/17/26 | 22.2 | 0 | 4.6/2.0 | 277/391 | 362/263 | 0 | 0.94/0.97 | 42/38 | 13/15 | 2.4/1.8/1.7 · 1.9/1.4/1.1 | 0.1/0.1 | 2.0/1.8 | 12.3/30 |
| 30 | spread | casters | 100 | 27% | 15/18/23 | 19.6 | 0 | 4.3/2.2 | 293/377 | 350/278 | 0 | 0.91/0.97 | 43/38 | 14/14 | 2.4/1.5/2.0 · 1.8/1.6/1.1 | 0.1/0.1 | 2.4/1.6 | 12.7/26 |
| 30 | kit | spread | 100 | 74% | 16/20/25 | 20.4 | 0 | 1.8/4.2 | 365/322 | 241/340 | 69 | 0.74/0.85 | 43/42 | 20/16 | 1.8/1.4/1.6 · 2.0/2.2/1.1 | 0.1/0.1 | 1.9/0.9 | 13.4/29 |
| 30 | default | spread | 100 | 88% | 17/20/31 | 21.4 | 0 | 1.4/4.8 | 398/283 | 199/368 | 73 | 0.74/0.90 | 42/41 | 22/14 | 2.1/2.0/1.0 · 2.2/1.3/2.3 | 0.1/0.1 | 1.1/3.5 | 12.7/25 |
| 30 | default | default | 100 | 55% | 17/23/30 | 23.7 | 0 | 3.8/4.2 | 356/338 | 312/327 | 2 | 0.81/0.92 | 47/44 | 22/16 | 1.8/1.4/1.5 · 1.6/0.9/1.7 | 0.1/0.1 | 2.2/3.9 | 10.6/28 |
| 30 | default | casters | 100 | 47% | 17/22/32 | 22.7 | 0 | 3.9/4.1 | 349/345 | 321/321 | 0 | 0.77/0.93 | 47/45 | 25/16 | 1.7/1.4/1.3 · 1.6/0.6/1.8 | 0.1/0.1 | 2.0/3.7 | 10.6/28 |
| 30 | focus | spread | 100 | 76% | 15/18/27 | 18.1 | 0 | 2.2/4.7 | 389/273 | 258/360 | 0 | 0.97/0.94 | 38/43 | 15/15 | 1.5/1.5/0.8 · 2.5/1.5/2.2 | 0.1/0.2 | 1.3/3.1 | 12.1/26 |
| 30 | focus | default | 100 | 53% | 17/22/29 | 21.7 | 0 | 3.9/4.2 | 361/345 | 321/335 | 0 | 0.95/0.97 | 44/46 | 16/15 | 1.7/1.3/1.2 · 2.0/1.4/1.6 | 0.1/0.1 | 2.6/3.6 | 10.2/26 |
| 30 | focus | casters | 100 | 48% | 15/21/30 | 19.6 | 0 | 3.8/4.2 | 363/336 | 313/336 | 0 | 0.90/0.98 | 46/46 | 16/14 | 1.7/1.5/1.1 · 2.0/0.9/1.7 | 0.1/0.1 | 1.9/3.2 | 10.3/29 |
| 60 | spread | spread | 100 | 63% | 11/14/19 | 15.0 | 0 | 3.0/3.7 | 414/386 | 364/386 | 0 | 1.10/1.08 | 41/40 | 16/16 | 2.0/1.9/1.2 · 1.9/1.9/1.2 | 0.1/0.1 | 2.5/1.9 | 15.1/31 |
| 60 | spread | default | 100 | 33% | 11/15/21 | 17.5 | 0 | 4.3/2.4 | 340/427 | 395/323 | 0 | 1.08/1.12 | 45/38 | 14/16 | 1.9/1.4/1.8 · 2.0/1.5/0.9 | 0.2/0.1 | 2.4/1.9 | 14.3/31 |
| 60 | spread | casters | 100 | 34% | 13/16/21 | 18.0 | 0 | 4.3/2.6 | 357/429 | 398/337 | 0 | 1.04/1.13 | 45/36 | 16/16 | 2.2/1.8/1.5 · 2.2/1.3/1.1 | 0.2/0.1 | 2.6/1.9 | 13.8/32 |
| 60 | kit | spread | 100 | 84% | 13/16/22 | 17.0 | 0 | 1.5/4.6 | 445/355 | 258/410 | 84 | 0.86/0.97 | 45/43 | 20/15 | 1.8/1.3/1.2 · 2.2/1.7/1.0 | 0.1/0.1 | 1.7/1.3 | 15.0/31 |
| 60 | default | spread | 100 | 85% | 13/17/25 | 18.1 | 0 | 1.8/4.8 | 449/335 | 250/415 | 69 | 0.86/1.06 | 42/43 | 22/15 | 1.9/1.3/0.7 · 2.1/1.3/1.8 | 0.1/0.1 | 1.1/3.5 | 14.4/29 |
| 60 | default | default | 100 | 52% | 14/18/25 | 18.8 | 0 | 3.9/4.2 | 405/395 | 365/373 | 1 | 0.96/1.07 | 49/48 | 20/14 | 1.6/1.1/1.3 · 1.8/0.8/1.4 | 0.1/0.0 | 1.8/3.4 | 12.1/33 |
| 60 | default | casters | 100 | 42% | 14/19/23 | 18.4 | 0 | 4.0/4.0 | 392/402 | 369/362 | 2 | 0.90/1.08 | 48/48 | 22/17 | 1.4/1.2/1.1 · 1.9/0.7/1.3 | 0.1/0.1 | 1.8/3.5 | 12.4/31 |
| 60 | focus | spread | 100 | 69% | 13/16/23 | 16.3 | 0 | 2.6/4.6 | 439/337 | 316/404 | 0 | 1.11/1.10 | 37/45 | 16/17 | 1.8/1.5/0.8 · 2.6/1.4/1.6 | 0.1/0.1 | 1.0/3.0 | 13.6/29 |
| 60 | focus | default | 100 | 56% | 13/17/24 | 17.4 | 0 | 3.9/4.3 | 416/388 | 359/384 | 0 | 1.09/1.13 | 47/48 | 15/15 | 1.6/1.2/1.1 · 1.9/1.2/1.4 | 0.1/0.1 | 2.0/3.3 | 11.9/30 |
| 60 | focus | casters | 100 | 59% | 13/18/24 | 18.1 | 0 | 3.8/4.3 | 418/384 | 357/386 | 0 | 1.06/1.13 | 46/47 | 15/15 | 2.0/1.2/0.9 · 2.0/1.2/1.5 | 0.1/0.1 | 1.9/3.1 | 11.7/30 |
| 100 | spread | spread | 100 | 50% | 11/14/17 | 13.8 | 0 | 3.2/3.2 | 458/447 | 422/432 | 0 | 1.17/1.15 | 41/39 | 15/17 | 2.2/2.3/1.5 · 2.1/2.3/1.1 | 0.1/0.1 | 2.2/2.0 | 16.6/33 |
| 100 | spread | default | 100 | 31% | 12/15/19 | 16.7 | 0 | 4.3/2.4 | 393/488 | 454/374 | 0 | 1.14/1.18 | 47/38 | 15/15 | 2.3/1.6/1.6 · 1.9/1.5/0.9 | 0.1/0.1 | 2.6/1.7 | 15.0/40 |
| 100 | spread | casters | 100 | 24% | 11/14/17 | 16.2 | 0 | 4.4/1.9 | 363/501 | 466/348 | 0 | 1.10/1.18 | 45/40 | 16/17 | 2.3/1.6/1.5 · 1.9/1.5/0.8 | 0.1/0.1 | 2.7/2.8 | 15.5/35 |
| 100 | kit | spread | 100 | 90% | 13/15/20 | 15.8 | 0 | 1.0/4.8 | 518/376 | 253/480 | 112 | 0.92/1.02 | 46/43 | 20/15 | 2.1/1.1/1.3 · 2.2/1.6/1.2 | 0.1/0.1 | 1.9/1.1 | 16.4/35 |
| 100 | default | spread | 100 | 91% | 13/16/21 | 16.3 | 0 | 1.1/4.9 | 528/327 | 235/488 | 82 | 0.93/1.09 | 44/42 | 23/15 | 2.4/1.5/0.8 · 2.3/1.4/1.4 | 0.1/0.1 | 1.6/2.2 | 15.4/33 |
| 100 | default | default | 100 | 53% | 15/19/26 | 19.3 | 0 | 3.9/4.3 | 480/451 | 419/443 | 0 | 1.02/1.14 | 50/48 | 20/16 | 2.2/1.3/1.3 · 1.4/1.5/1.4 | 0.1/0.1 | 2.0/2.6 | 12.1/39 |
| 100 | default | casters | 100 | 48% | 15/21/27 | 20.1 | 0 | 3.9/4.3 | 481/459 | 428/445 | 0 | 0.95/1.15 | 49/48 | 22/15 | 2.3/1.4/1.2 · 1.6/1.2/1.3 | 0.1/0.1 | 2.4/2.9 | 11.8/37 |
| 100 | focus | spread | 100 | 85% | 12/15/22 | 15.3 | 0 | 1.6/4.8 | 521/323 | 309/483 | 0 | 1.17/1.13 | 39/43 | 15/14 | 2.5/1.2/0.7 · 2.4/1.7/1.4 | 0.1/0.2 | 1.3/2.2 | 14.5/32 |
| 100 | focus | default | 100 | 70% | 14/18/24 | 18.1 | 0 | 3.5/4.6 | 502/429 | 399/467 | 0 | 1.18/1.17 | 48/49 | 15/15 | 1.8/1.3/1.3 · 2.2/1.4/1.2 | 0.1/0.1 | 2.2/2.9 | 12.2/36 |
| 100 | focus | casters | 100 | 62% | 13/18/25 | 18.1 | 0 | 3.5/4.5 | 498/426 | 398/463 | 0 | 1.10/1.18 | 48/48 | 16/16 | 1.7/1.4/0.9 · 1.9/1.3/1.4 | 0.1/0.1 | 2.4/2.3 | 11.9/39 |

## Coordinated enemies (TestBalanceCoordinated)

| level | company | enemy | fights | company wins | rounds p10/median/p90 | won-fight rounds mean | stalls | fallen company/enemy | damage company/enemy | net HP lost company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | lines/round mean/peak |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | default | default | 100 | 50% | 19/27/41 | 30.2 | 0 | 3.7/3.9 | 240/250 | 223/221 | 8 | 0.60/0.72 | 43/42 | 23/14 | 1.8/1.5/1.5 · 1.2/1.3/1.7 | 0.1/0.0 | 1.7/2.3 | 8.7/27 |
| 1 | default | roles1 | 100 | 54% | 20/27/39 | 28.6 | 0 | 3.5/4.0 | 245/240 | 213/225 | 8 | 0.61/0.71 | 43/43 | 23/14 | 1.4/1.3/1.5 · 1.2/1.3/1.8 | 0.1/0.1 | 1.3/2.6 | 8.7/27 |
| 1 | default | roles2 | 100 | 37% | 19/27/39 | 31.2 | 0 | 4.2/3.5 | 220/266 | 238/199 | 6 | 0.59/0.69 | 43/43 | 23/15 | 1.7/1.6/1.6 · 1.5/0.7/1.3 | 0.1/0.0 | 1.9/2.9 | 8.9/30 |
| 1 | default | roles3 | 100 | 38% | 18/29/42 | 31.2 | 0 | 4.2/3.4 | 225/261 | 238/203 | 1 | 0.60/0.68 | 43/41 | 23/17 | 1.6/1.4/1.4 · 2.0/0.6/1.2 | 0.1/0.0 | 2.6/3.4 | 8.8/27 |
| 5 | default | default | 100 | 45% | 19/26/39 | 26.1 | 0 | 3.7/3.9 | 254/276 | 235/234 | 20 | 0.64/0.81 | 44/43 | 24/14 | 1.7/1.3/1.7 · 1.6/0.9/1.6 | 0.1/0.1 | 1.6/3.8 | 10.1/29 |
| 5 | default | roles1 | 100 | 53% | 20/25/35 | 25.0 | 0 | 3.4/4.0 | 261/263 | 222/242 | 22 | 0.64/0.79 | 44/43 | 25/14 | 1.6/1.3/1.6 · 1.6/1.1/1.8 | 0.1/0.0 | 1.9/3.4 | 10.6/27 |
| 5 | default | roles2 | 100 | 34% | 21/28/42 | 28.4 | 0 | 4.1/3.5 | 245/286 | 253/218 | 12 | 0.64/0.73 | 43/43 | 23/15 | 1.9/1.5/1.8 · 2.1/0.7/1.7 | 0.1/0.1 | 2.1/3.9 | 10.0/28 |
| 5 | default | roles3 | 100 | 40% | 20/28/38 | 29.4 | 0 | 4.2/3.2 | 243/274 | 251/206 | 3 | 0.68/0.69 | 43/42 | 23/16 | 1.9/1.4/1.7 · 2.3/0.7/1.6 | 0.1/0.1 | 2.4/3.7 | 10.0/26 |
| 10 | default | default | 100 | 60% | 22/29/41 | 28.3 | 0 | 3.0/4.3 | 298/279 | 232/277 | 29 | 0.64/0.80 | 45/43 | 25/16 | 1.6/1.5/1.8 · 1.5/1.2/2.1 | 0.1/0.0 | 2.6/3.6 | 10.2/28 |
| 10 | default | roles1 | 100 | 53% | 21/28/41 | 29.8 | 0 | 3.3/4.0 | 286/289 | 238/265 | 33 | 0.63/0.80 | 43/43 | 25/15 | 1.7/1.3/2.0 · 1.6/1.2/2.0 | 0.1/0.1 | 2.3/3.1 | 10.2/29 |
| 10 | default | roles2 | 100 | 37% | 22/30/41 | 31.8 | 0 | 4.0/3.6 | 266/309 | 271/244 | 17 | 0.64/0.78 | 44/43 | 25/16 | 2.0/1.8/1.9 · 2.0/0.9/1.8 | 0.1/0.0 | 3.1/4.1 | 9.8/28 |
| 10 | default | roles3 | 100 | 26% | 22/30/41 | 32.4 | 0 | 4.5/3.3 | 253/312 | 290/235 | 1 | 0.69/0.76 | 44/44 | 21/15 | 1.8/1.6/2.2 · 2.4/0.7/1.7 | 0.1/0.1 | 2.8/4.0 | 9.3/30 |

## Zone bands (TestBalanceZoneBands)

The run logged only mean net company HP lost per cell. Compared with 35a,
2–3-foe cells lose less health (for example band 1–3, L1 3×L1: 29.8% against
37.2%); boss cells still cost almost the whole company, as in 35a.

| Band / company / enemy group | Mean net company HP lost |
|---|---|
| band1-3/L1/2vL1/bossfalse | 10.8% |
| band1-3/L1/2vL2/bossfalse | 14.9% |
| band1-3/L1/3vL1/bossfalse | 29.8% |
| band1-3/L1/3vL2/bossfalse | 39.7% |
| band1-3/L1/4vL2/bossfalse | 63.1% |
| band1-3/L2/2vL1/bossfalse | 9.0% |
| band1-3/L2/2vL2/bossfalse | 11.5% |
| band1-3/L2/3vL1/bossfalse | 23.5% |
| band1-3/L2/3vL2/bossfalse | 31.3% |
| band1-3/L2/4vL2/bossfalse | 51.2% |
| band1-3/L3/2vL1/bossfalse | 8.9% |
| band1-3/L3/2vL2/bossfalse | 8.4% |
| band1-3/L3/3vL1/bossfalse | 19.9% |
| band1-3/L3/3vL2/bossfalse | 22.7% |
| band1-3/L3/4vL2/bossfalse | 39.6% |
| band1-3/L3/5vL3/bosstrue | 99.6% |
| band8-10/L8/2vL7/bossfalse | 7.8% |
| band8-10/L8/2vL8/bossfalse | 8.9% |
| band8-10/L8/2vL9/bossfalse | 10.7% |
| band8-10/L8/3vL7/bossfalse | 21.7% |
| band8-10/L8/3vL8/bossfalse | 20.9% |
| band8-10/L8/3vL9/bossfalse | 28.3% |
| band8-10/L8/4vL9/bossfalse | 60.2% |
| band8-10/L9/2vL7/bossfalse | 7.1% |
| band8-10/L9/2vL8/bossfalse | 7.5% |
| band8-10/L9/2vL9/bossfalse | 8.8% |
| band8-10/L9/3vL7/bossfalse | 17.7% |
| band8-10/L9/3vL8/bossfalse | 18.8% |
| band8-10/L9/3vL9/bossfalse | 24.0% |
| band8-10/L9/4vL9/bossfalse | 43.7% |
| band8-10/L10/2vL7/bossfalse | 6.0% |
| band8-10/L10/2vL8/bossfalse | 7.0% |
| band8-10/L10/2vL9/bossfalse | 7.2% |
| band8-10/L10/3vL7/bossfalse | 15.6% |
| band8-10/L10/3vL8/bossfalse | 15.8% |
| band8-10/L10/3vL9/bossfalse | 19.0% |
| band8-10/L10/4vL9/bossfalse | 35.0% |
| band8-10/L10/5vL10/bosstrue | 97.0% |
| band18-20/L18/2vL17/bossfalse | 11.2% |
| band18-20/L18/2vL18/bossfalse | 13.1% |
| band18-20/L18/2vL19/bossfalse | 14.3% |
| band18-20/L18/3vL17/bossfalse | 26.4% |
| band18-20/L18/3vL18/bossfalse | 31.7% |
| band18-20/L18/3vL19/bossfalse | 39.9% |
| band18-20/L18/4vL19/bossfalse | 65.2% |
| band18-20/L19/2vL17/bossfalse | 10.2% |
| band18-20/L19/2vL18/bossfalse | 10.5% |
| band18-20/L19/2vL19/bossfalse | 12.7% |
| band18-20/L19/3vL17/bossfalse | 24.5% |
| band18-20/L19/3vL18/bossfalse | 27.6% |
| band18-20/L19/3vL19/bossfalse | 32.5% |
| band18-20/L19/4vL19/bossfalse | 53.6% |
| band18-20/L20/2vL17/bossfalse | 8.4% |
| band18-20/L20/2vL18/bossfalse | 9.6% |
| band18-20/L20/2vL19/bossfalse | 10.8% |
| band18-20/L20/3vL17/bossfalse | 20.5% |
| band18-20/L20/3vL18/bossfalse | 24.4% |
| band18-20/L20/3vL19/bossfalse | 28.5% |
| band18-20/L20/4vL19/bossfalse | 47.1% |
| band18-20/L20/5vL20/bosstrue | 99.1% |
| band28-30/L28/2vL27/bossfalse | 10.7% |
| band28-30/L28/2vL28/bossfalse | 12.2% |
| band28-30/L28/2vL29/bossfalse | 14.3% |
| band28-30/L28/3vL27/bossfalse | 26.7% |
| band28-30/L28/3vL28/bossfalse | 32.9% |
| band28-30/L28/3vL29/bossfalse | 38.8% |
| band28-30/L28/4vL29/bossfalse | 66.1% |
| band28-30/L29/2vL27/bossfalse | 11.1% |
| band28-30/L29/2vL28/bossfalse | 12.5% |
| band28-30/L29/2vL29/bossfalse | 13.8% |
| band28-30/L29/3vL27/bossfalse | 26.4% |
| band28-30/L29/3vL28/bossfalse | 30.8% |
| band28-30/L29/3vL29/bossfalse | 35.0% |
| band28-30/L29/4vL29/bossfalse | 59.0% |
| band28-30/L30/2vL27/bossfalse | 9.5% |
| band28-30/L30/2vL28/bossfalse | 10.2% |
| band28-30/L30/2vL29/bossfalse | 10.8% |
| band28-30/L30/3vL27/bossfalse | 19.3% |
| band28-30/L30/3vL28/bossfalse | 24.9% |
| band28-30/L30/3vL29/bossfalse | 28.3% |
| band28-30/L30/4vL29/bossfalse | 41.7% |
| band28-30/L30/5vL30/bosstrue | 99.7% |
