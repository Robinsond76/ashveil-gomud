# Phase 50: condition carries into battle; meal buffs (execution plan, 2026-10-06)

Scope and acceptance come from the
[survival phase plan](2026-10-06-outward-survival-phases.md#50-condition-carries-into-battle).
This page records how it was built and the decisions the owner delegated.

## Design

- **Rules** live in `internal/survival/meals.go` (pure): the meal kinds,
  `ConditionFor(needs)` and the bands' numbers. Fatigue's existing hit cut
  moved there too (`FatigueHitPenalty`; `formationcombat.FatiguePenalty`
  calls it) so the condition shows the same numbers combat uses.
- **State:** a member's meal buff is two fields on its survival `Needs`
  (`meal`, `meal_battles`), so it is saved, snapshotted (test area),
  reconciled and restored with the needs it sits beside. `Normalize` drops an
  unknown or spent meal and caps a stored count at its meal's length.
- **Battle start:** `beginBattle` calls `startFare` after the sigil. For each
  member standing in the room it reads the needs survival reports, sets
  `ClassRT.FareDamage` and `FareGuard` for the battle (cleared with the rest
  of the class state at fight end), restores a fish meal's mana, records each
  member's summary on the battle (`battle.Fare`), counts one battle off the
  meal buffs of those who fought (`survival.SpendMealBattle`, written with the
  next save like a drain), and sends one "Going in:" line.
- **Combat:** `fareDamage` (after leadroot, before armor) scales a blow by the
  dealer's `FareDamage` and the target's `FareGuard`; `FareSpellFactor` does
  the same in `SpellFactor`. Fatigue's hit cut is unchanged.
- **Meals:** items name their buff with `meal:` (item spec field). Eating
  (`eat`, `company eat`) passes it in the survival `Benefit`; the eat line
  names the buff.

| Need | Hungry / Thirsty / Tired | Starving / Parched / Exhausted (or worse) |
|---|---|---|
| Hunger | -5% damage dealt | -10% damage dealt |
| Thirst | +5% damage taken | +10% damage taken |
| Fatigue | -5 hit chance (existing) | -10 (Collapsed -20, existing) |

| Meal (item) | Buff | Battles | Effect |
|---|---|---|---|
| seared game meat (30021) | Strong | 2 | +5% damage |
| thyme-roasted game (30020) | Steady | 3 | +5% damage, 5% less damage taken |
| hunter's stew (30019) | Hearty | 3 | 10% less damage taken |
| grilled fish (30024) | Clear-headed | 3 | 20% of max mana back as each battle begins |

## Decisions (owner delegated; each with its reason)

1. **Thirst raises damage taken, not tempo.** The plan suggested a tempo cut.
   Built that way, a 5% cut made the whole company lose the same early turn
   (the tempo meter's second round), and the even mirror fell from 50% to
   12% wins for merely Thirsty members. Damage taken scales smoothly.
2. **Fatigue keeps its existing hit cut** rather than gaining a second
   penalty: it already worked in battle (`help battlefield`); this phase only
   names it in the condition.
3. **A meal lasts battles, counted as a battle begins**, for the members who
   fought it. A meal never wears off on the road, and a battle never changes
   halfway (eating is refused mid-battle).
4. **Meal buffs are a field on the needs**, not a new store: they persist,
   snapshot and reconcile for free with the record they describe.
5. **The meal table is in Go; items opt in with `meal:`.** New dishes use an
   existing kind with a YAML line; a new kind is one table row.
   `TestShippedMealBuffs` checks every `meal:` names a known kind, every kind
   is served, and every hearth dish gives one.
6. **No price changes.** Meals keep their values; the market already never
   buys food back above cost.
7. **A well-kept company is untouched**: neutral needs and no meal set every
   factor to zero, so 37b's numbers hold (`fed` is the baseline cell).

## Display

The "Going in:" opening line; `survival` ("In battle:" per member), `status`
(Battle row), `conditions` (Survival group, "In battle" row); GMCP
Company.Vitals `fare` (Company panel "In battle:" line) and Company.Battle
`fare` by member key (battle screen banner "condition: ..." and hover
caption, Combat tab "Condition:" notes).

## Help and tutorial

`help survival` (In battle section; aliases `battle condition`,
`battle-condition`, `going-in`), `help cooking` (Meal buffs; aliases
`meal buff`, `meal-buffs`), plus `help conditions`, `help eat`,
`help company meal` and the `help combat` hub. Tutorial: a hint in the
survival lesson.

## Tests

`internal/survival` (bands, meals, summary), `internal/combat` (blow and spell
factors, a starving dealer's blows through the strike loop), `modules/survival`
(eating gives and saves the buff, survives a restart, decode drops bad
meals, the last battle ends it, status line), `modules/company`
(`wiring_fare_test.go`: the real beginBattle sets the condition, names it,
spends the meal; no service changes nothing), `modules/gmcp` (vitals and
battle feed), `internal/usercommands` (eat passes the meal; help renders),
root `TestShippedMealBuffs`. Balance: `TestPhase50BattleCondition` (opt-in,
`ASHVEIL_BALANCE=1`).
