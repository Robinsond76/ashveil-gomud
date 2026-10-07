# At-level fight tuning (decisions)

Owner ask (Robinson, 2026-10-07): a company in a zone rated for its level
should win quickly and cheaply, in groups of 2-3 (sometimes 4), so it can fight
15-20 battles before resting (7-12 leaning on magic). Refined the same day:
that is for a full five-member company; four members rest after 8-14, three
after 5-7, and a solo player suffers in a zone for their own level. Good gear
and buffs must still give a clear edge. Difficulty stays zone-fixed
(2026-10-06 rule): only a zone above the company's level is hard, and the
numbers by company size come from the company, never from scaling a zone.

## What shipped

- **Ordinary foes spawn softer** (`encounters.OrdinaryHPPercent` = 40): an
  ordinary random group's foe has 40% of its level's HP against a company at or
  above the band's low end. The softness fades evenly to full HP at five levels
  under the band (`encounters.HPPercent`, `UnderBandGap` 5), so the hard
  gradient the 37b work set (about half the fights won five under) is kept.
  Bosses, their escorts and story-event groups keep full HP.
- **Foes spread their blows** (`encounters.Spread`, `SpreadPercent` 100): a
  softened foe's aim noise is its softness (60 at base), so 60% of its re-aims
  take a random foe instead of the weakest. Without it, the default
  "weakest" aim put every blow on the leader or the wizard and the rest of the
  company stood untouched, capping the fights before a rest at about 6-8 for
  any company size. It fades the same way as HP.
- Both are set where a random encounter spawns (`encounters.Soften` in the
  module, applied by `enemyparty.SpawnEncounter`), from the leader's company
  level, so they carry into the replacement world: they key on the zone's band,
  not on any mob.
- Encounter levels and group sizes are unchanged (levels band-low to one under
  the top; four-foe groups at the band's low). Four-foe groups cost about
  twice an ordinary group, which is the "bit tougher" the ask wanted.

## The measure

`TestBalanceAtLevel` (opt-in, `ASHVEIL_BALANCE=1`, `ASHVEIL_BALANCE_FIGHTS`,
`ASHVEIL_BALANCE_ONLY=5,4`) runs a company of 1-5 members at the band's top
against the band's ordinary encounters, one fight at a time from full health,
then draws 400 campaigns per cell (seven fights in eight ordinary, one in eight
four foes). The company rests when a member would have gone down, it has lost
half its health, or a caster is under 20% mana. The leader wears a companion
fighter's kit (the harness leader is bare), companions join in order Tamsin,
Oswin, Garrick, Ysolde (a magic company's wizard first). Magic company: Garrick
is a wizard. It is the class-balance mirror's complement, not a replacement.

Targets by size (fights before a rest, martial / magic): 5: 15-20 / 7-12
(owner), 4: 8-14 / 5-8, 3: 5-7 / 3-5, 2: 3-4 / 2-3 (ours), 1: 1-2 (a solo
player suffers). The magic ranges are ours, scaled the same way.

## Measured (50 fights a cell, median fights before a rest, p25-p75)

| Band | 5 members | 4 | 3 | 2 | 1 |
|---|---|---|---|---|---|
| 3-5 martial | 17 (13-20) | 14 (12-17) | 9 (6-11) | 2 | 1 (wins 64%) |
| 10-12 martial | 11 (8-14) | 12 (9-15) | 8 (6-10) | 2 | 1 (64%) |
| 20-22 martial | 10 (8-12) | 7 (6-9) | 7 (5-9) | 2 | 1 (58%) |
| 3-5 magic | 5 (4-6) | 5 (4-6) | 3 | 2 | |
| 10-12 magic | 5 (3-7) | 5 (2-7) | 2 | 1 | |
| 20-22 magic | 4 (3-6) | 4 (3-6) | 2 | 1 | |

Fights at the band's top last 4 rounds (about 30 seconds) for three or more
members, with 100% wins and 3-6% of company health lost per fight; two members
8 rounds and 30% lost; a solo leader 8-9 rounds, 60-70% lost, 58-64% wins.

Shapes (`TestBalanceEncounterShapes`, 40-160 fights a cell): ordinary groups at
the band: 100% wins, 4 rounds, 4-6% HP lost; four-foe groups 7-10%; healer
groups 6-7%. Two under 95-99%, three under 68-94%, five under 45% wins (was
51%); bosses with two or three escorts unchanged (15-19 rounds, 100% wins).

## Where it falls short (recorded, not chased)

- **Full martial company at high bands**: 10-12 fights, not 15-20 (the 3-5
  band reaches 17). One member, usually the leader, still takes most of the
  damage in the variant that keeps the weakest member above 20%
  (8-14 at five members). More softness would also lift three- and
  four-member companies past their ranges, which are already at the top end.
- **Magic**: 4-5 fights, not 7-12, at every size. Mana is not the limit
  (mana ended 2% of campaigns); the wizard's health is: even with aim spread a
  wizard loses 20-60% of its health per fight. The fix is wizard
  survivability or default guarding, not encounter tuning (follow-up).
- **Three members** rest after 7-9 (target 5-7) and a **solo leader** loses
  about 40% of at-level fights, which costs a level on death. "Suffer" was the
  ask; if that is too harsh, soften the solo case through leader survivability
  (heal items, camp) rather than the zone, per the zone-fixed rule.
- The harness companies are one fixed makeup; real companies differ.
- Trickle healing (to half health) is ignored: about 1 HP a round.

## Gear still gives an edge

One level of company over foes is worth a lot here: the same martial company one
level lower (L11 in the 10-12 band, L4 in 3-5) rested after 6-8 fights instead of
10-14 in the small sims, so a one-level edge (a fully enchanted company is a bit
over one level, phase 71; good armor and weapons are about that) still moves
the rhythm by a third to a half. Gear on foes' side does not exist (foes are
level-set), so there is no ceiling.

## Tests changed

- `TestBalanceEncounterShapes` (opt-in): ordinary, four-foe, healer and
  under-band cells now spawn with the encounter's HP and aim; thresholds
  unchanged and still hold.
- New: `TestBalanceAtLevel` (opt-in); `TestOrdinaryFoes...` in
  `modules/encounters`, `internal/encounters` (policy, bosses and story groups
  unsoftened, spawn wiring from the company's level).
- The harness (`balance_test.go`) gained `OrdinaryHPPercent`; its fallen count
  now starts from the company's size at the fight's start.
