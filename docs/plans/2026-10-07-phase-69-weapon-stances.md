# Phase 69: weapon stances (build decisions, 2026-10-07)

Source: [pillars phases](2026-10-07-pillars-phases.md#69-weapon-stances).
Full autonomy applies; each fork below records the pick and why.

## What shipped

One alternate way to use a weapon family, chosen per company member between
battles, that trades one strength for another.

| Stance | Needs | Gain | Cost |
|---|---|---|---|
| `heavy` Heavy blows | a two-handed melee weapon (not a staff, rod or bow) | blows land 30% harder | 15 points less likely to hit |
| `wall` Shield wall | a shield | 12 points more likely to block | 30% fewer turns each round |
| `quick` Quick draw | a bow (not a crossbow) | 25% more turns each round | arrows land 15% lighter |
| `keen` Keen edge | a dagger | 10 points more likely to land a critical hit | blows land 10% shallower |

Commands: `stance`, `stance [who]`, `stance [who] [name]`, `stance [who] off`
(outside a battle only). Web: a Stance button under each member's orders in
the Combat tab; the battle screen names the stance when a figure is tapped.

## Decisions

| Decision | Why |
|---|---|
| **One stance per member, four to choose from (one per family).** A member holding a glaive and a shield picks heavy blows or shield wall. | The plan says one stance per family; a member only ever holds a couple of families, and one at a time keeps the choice a real trade. |
| **The choice is kept even when the weapon is not in hand, and does nothing then.** `stance` says "idle: needs a bow". | Swapping a weapon for a fight should not lose a setup, and a stance can never act without its weapon. Fit is checked at set time (a note), at the round it is applied (no announcement) and at every blow (so putting the weapon away mid-battle silences it). |
| **Families come from item data** (hands, subtype, `weaponclass`, `family`, a shield's damage reduction), not item names. | The replacement world brings its own items; they join a family by data. Crossbows are left out of `quick` (they reload), staffs and rods out of `heavy` (caster implements). |
| **Set outside battle only**, like orders and strategy. | Ogre Battle rule: no input mid-battle but company focus and retreat. |
| **Saved in `modules/strategy`'s registry** (`stances:` per user and member key), with snapshot support for the test area, pruned when a member leaves, cleared on purge, malformed entries dropped on load. | Same durable store and lifecycle as strategies and orders; nothing new to lose on restart or copyover. |
| **Applied each round in the tempo fill** (`internal/hooks/combat_stance.go`), onto `ClassRT.Stance`, read by combat through `Character.StanceEffect()`. | `fillTempo` runs before every combatant's turns for players and companions alike, so the stance is in force before the first blow; `EndFightRT` clears it. Enemies are never given one. |
| **Tempo is scaled after the speed clamp** (floor 0.25). | A fast fighter at the tempo cap would otherwise gain nothing from quick draw. |
| **Where each effect lands:** hit points are a hit-roll modifier (shown as the strike's modifier in `why`); damage scales the landed blow after critical damage; critical points add to the crit roll; block points add to the block chance; `ExpectedDamage` (the company's assessment) reads all of them. | One seam per number, each already named in the explained battle line. |
| **Text:** the first round of a battle tells the leader "Tamsin takes the Heavy blows stance: ...", and every landed blow it changed carries a breakdown note ("Heavy blows stance changed the blow by +6"). | Text claims only real effects (neutral-class lesson); accessible without the picture. |
| **Sidegrades, measured twice.** Expected damage per round at even stats (`TestStancesAreSidegrades`, armored and unarmored foes): heavy 1.08, keen 0.95, quick 1.06 of the plain weapon; the wall deals 0.70 and takes 0.79. The 5v5 mirror cells (100 fights a variant, level 10, `TestBalanceStances`): heavy wins 90 -> 86%, wall 92 -> 96% with 15% less health lost and fights 2.6 rounds longer, quick 100 -> 100%, keen 70 -> 81% (health lost 196 -> 170, noisy; an earlier 15-point keen was a clear win, 10 points and 10% is the settled trade). | Criticals and extra turns are worth more in a real fight than their expected damage (they break chants and bring procs), so the cells set the numbers, not the arithmetic. Timeboxed: three tuning rounds, settled on the last. |
| **No gold, experience or loot effect.** | Pure combat trade. |

## Help and tutorial

`help stances` (indexed under combat with aliases, linked from `help combat`,
`help strategy`, `help orders` and `help webclient`), a hint in the tutorial's
combat lesson beside orders.

## Not done (follow-ups)

- Stances for spears, whips and staffs, or two per family.
- A weapon tooltip line naming the stance a weapon can take.
- Relic awakenings or creeds that care about stances (73).
- A stance that changes with the foe (an order such as "when a boss stands, take heavy blows"): orders cannot set stances yet.
- A companion that is away cannot have its gear checked; `stance` says "not here to check" and sets the choice anyway.

## Review (Opus review thread, independent reviewer subagent plus lead checks)

**Balance re-measured** at 200 fights a variant (level 10 5v5 mirror, one
process a cell): heavy 91 -> 93% wins (health lost 140 -> 130), keen
82 -> 81% (163 -> 173), wall 92 -> 92% (119 -> 115, 25 -> 28 rounds). The
build's keen 70 -> 81% and wall 92 -> 96% were noise (about ±11 points at
100 fights); every gap is under 1.5 standard errors, so no retune. Quick draw
sits at the 100% ceiling in this cell and is guarded by the expected-damage
test (1.06 of the plain bow).

**Fixed (with regression tests):**

| Finding | Fix |
|---|---|
| The pre-battle estimate (`internal/assessment`) never saw a stance: it is only put on the battle state in the round. | `withStance` gives the estimate a copy carrying the stored stance; the live member is untouched (`TestTheEstimateReadsAStanceBeforeABattle`). |
| A dual-wielder's second weapon took the main hand's stance (a sword behind a keen dagger was keen; a dagger behind a sword never was). | Each weapon past the first takes the stance only when it fits (`StanceEffectWith`, `TestASecondWeaponTakesTheStanceOnlyWhenItFits`); the breakdown names the stance either way. |
| The round re-read the store every round, so anything writing it mid-battle (a test-area restore, a prune) would change the stance in a fight. | Read once a battle (`ClassRT.StanceRead`, cleared at the fight's end; `TestAStanceIsReadOnceABattle`). |
| `stance heavy` (no one named) answered "No one in your company answers to heavy". | It sets your own; `stance off` clears yours (`TestStanceWithNoOneNamedIsYours`). |
| `help opinions` called a story choice's tag "the stance of a choice", next to the new `stance` command. | Now "the spirit of a choice". |
| Help said `why` names the stance on every blow it changed (misses carry only the modifier) and that stances go when a member leaves (they are pruned at the next `stance`, `orders` or `strategy`, as orders are). | Help says "every landed blow" and "dropped for members who have left"; it also explains dual-wielding, `stance [name]` for yourself, and that Mountain Speaker's Steady stance is something else. The opening line is no longer coloured as a command. |
| The ordinary-suite stance fight only checked that rounds ran. | It checks the stance reached the companion's battle state. Unused test constant and a stale comment fixed. |

**Rejected or deferred:** the opening stance line goes to the leader's text
(as orders' "as ordered" line does), not the combat stream that feeds
`battlelog`, which is fine for a once-a-battle line; a weapon equipped
mid-battle wakes its stance without the line (rare, and `why` still names
it). The Combat tab menu lists all four stances whatever the gear: picking one
that doesn't fit is allowed by design and the row then says "idle: needs ...";
marking fitting ones would need gear availability in GMCP (follow-up). The
estimate still applies the main hand's stance to a dual-wielder's second
weapon (small, and only in the estimate).
