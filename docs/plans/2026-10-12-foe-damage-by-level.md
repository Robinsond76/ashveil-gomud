# Foe damage tracks foe level (2026-10-12)

Owner report: a level-3 company of five fought two level-1 imps and a
level-1 skeleton (`testarea fight 1 2 imp 1 skeleton`) and lost one member,
with 167 damage taken and a critical hit for 18. Level-1 foes should hit for
about 1d4, crits at most 9.

## Cause

Every blow adds a flat bonus, `damageBonus` (config `DamageBonusMin` 6 to
`DamageBonusMax` 12), on top of the weapon's dice, so a 1d4 dagger hits for
7-10 and a crit for up to 18. The bonus never read the attacker's level, and
health has a 48 point floor (`HPBase`), so a level-1 foe was nearly as hard
and hard-hitting as a level-3 member. Level only moved hit chances.

## Decision (revised the same day, after a second report)

The first fix ramped only enemies. The owner's next log showed the other
half: companions with the same 1d6 weapon as the hero hit level-1 imps for
26, 27 and 36 (telling and bolt multipliers on top of a flat bonus of 12,
because a companion's Strength is trained high), while the hero, with low
Strength, hit for 8 to 14. So the flat bonus is now level-scaled for every
attacker: none at level 1, growing evenly to whole at `DamageFullLevel` (6).
Level-1 blows are the weapon's dice; company and foes shrink together, so
the share of health a fight costs holds while single hits become small.
Level 6 and up, and every cell above the early bands, is unchanged. Code:
`levelDamageBonus` (`internal/combat/calculations.go`), applied in
`calculateCombatPower`.

The hero hits a bit harder than companions by reading `HeroDamageLead` (2)
levels ahead on that ramp (`sourceType == User`): at level 1 the hero gets a
third of the bonus, a companion none; the lead fades by level 6. Chosen over
higher starting stats because stats also move hit, dodge and health, which
the at-level tuning depends on, while the lead touches damage alone and is
one config number.

Rejected: ramping foes only (companions still hit 20+ at level 3);
`DamageFullLevel` 8 with lead 3 (five-member martial company in 3-5 rested
after 48 fights).

## Measurement

`ASHVEIL_BALANCE=1 ASHVEIL_BALANCE_FIGHTS=16 ASHVEIL_BALANCE_BAND=3-5`
(small sample; medians of fights before rest; earlier project median for 5
martial was 18):

| Company | Martial | Magic |
|---|---|---|
| 5 members | 26 | 5 |
| 4 members | 18 | 5 |
| 3 members | 13 | 4 |

Five martial is above its 15-20 target but inside the harness slack. Band
5-7 at 12 fights (depth 8/3 variant): 13 / 19 / 11 martial.

## Wounds only on critical hits (owner, 2026-10-12)

Wounds were happening too often. The two non-crit sources are gone: a
"crushing blow" (a plain hit of a quarter of max health) and the light wound
a bleed left when it ran out. Only a critical hit that gets through wounds
now (`internal/combat/combat.go`, `internal/hooks/combat_status.go`); the
easy-foe crit softening is unchanged. `help wounds` updated; the crushing
test now asserts no wound. Lower foe damage also makes big hits and crits
rarer at low levels.
