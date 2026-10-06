# Phase 55: ailments and remedies (execution plan, 2026-10-06)

Scope and acceptance come from the
[survival phase plan](2026-10-06-outward-survival-phases.md#55-ailments-and-remedies).
This page records how it was built and the decisions the owner delegated.

## Design

- **Rules** live in `internal/survival/ailments.go` (pure): the three
  ailments, their penalties, causes and remedies, and `CatchChillIfFrozen`.
- **State:** a member's ailments are three battle counts on its survival
  `Needs` (`chill`, `gutache`, `fever`), beside the meal buff, so they persist,
  snapshot, reconcile and restore with the needs. `Normalize` caps a stored
  count at its ailment's length. Writes go through the new
  `survival.AilmentService` (`CatchAilment`, `CureAilment`), implemented by
  `modules/survival` with a save at once, like a meal.
- **Battle:** `ConditionFor` adds each ailment's penalty to the Phase 50
  condition, so `startFare` (and every display that already shows the
  condition) needs no new plumbing. `SpendMealBattle` now also counts a battle
  off the ailments of the members that fought.

| Ailment | Caught by | Battle penalty | Lasts | Remedy (herbs) |
|---|---|---|---|---|
| Chill | walking out of doors, or finishing a camp rest, while Frozen (exposure -50, frostbitten, or worse) | -10% damage dealt | 4 battles | thyme tea: 2 wild thyme |
| Gut-ache | eating raw game meat (item 29, now edible: 10 Hunger, `ailment: gutache`) | +10% damage taken | 3 battles | tisane: 1 wild thyme, 1 mushroom |
| Fever | a lasting puncture wound open through 3 battles (`Wound.Battles`, counted in `startFare`) | -15% damage dealt | 5 battles | draught: 1 glacial mint, 1 wild thyme |

`camp prepare remedy [member|all]` makes the remedy for every ailing member at
the camp from herbs in the company cargo, checking all herbs before spending
any, and cures at once.

## Decisions (owner delegated; each with its reason)

1. **Ailments fade on their own** after 3 to 5 battles, and the remedy ends
   them at once. A permanent penalty with no way to wait it out would be a
   trap, and Outward's own colds pass. The remedy is the fast, certain cure.
2. **Penalties reuse Phase 50's two channels** (damage dealt, damage taken).
   A hit-chance cut would need new plumbing in four attack paths for no extra
   clarity; the numbers are the clear (10%) band and a slightly worse 15% for
   the fever, the one that takes neglect to earn.
3. **Frozen is the frostbitten band** (stored exposure at or below -50). A
   merely chilled member (-25) is fine, so ordinary weather never makes a
   company ill; only going on in the cold with no warmth does.
4. **Chill is caught on foot and at a camp rest, never at an inn.** The step
   check runs after the walking charge for an outdoor destination; the camp
   check runs when the finished rest's grant is paid. Both run outside every
   module lock (exposure holds its lock while calling camping's heat-source
   check, so reading exposure under camping's lock could deadlock).
   Expedition travel does not catch a chill: it runs under its own lock and
   its exposure is the same one the walking steps already check.
5. **Raw meat is edible but always gives a Gut-ache.** The plan said "from
   eating raw game meat"; a certain, visible cause is clearer than a dice
   roll. `company eat` (the planner) never serves it (`larderEntry` refuses
   any item with an `ailment`), so the company never makes itself ill by
   accident. Raw fish stays inedible.
6. **Remedies are made from herbs and used up as they are made**, never an
   item, so there is nothing to buy back (economy rule). The herbs are the
   gatherable thyme, mushroom and glacial mint; none sells for more than a
   quarter of its value, and cooking is unchanged.
7. **A feverish member's wound is not counted again**; the count starts over
   when the fever starts, so one untreated wound makes a fever at most every
   fifth battle.
8. **No hidden difficulty.** Every ailment comes from the company's own
   choice (walking Frozen, eating raw, skipping treatment), shows in the
   "Going in:" line, `status`, `conditions`, `survival`, the Company panel,
   the vitals strip warnings and the battle screen, and a well-kept company
   fights exactly as 37b measured it.

## Display

`Member.Ailments` (companyview) feeds the GMCP vitals `ailments` list (the
vitals strip warning, the Company > Camp "Ailing" line), `status` ("Ailing:"),
`conditions` (a row per ailment with its cure) and `camp prepare status`. The
condition summary (`fare`) already carries them to the Company panel, the
battle screen and the Combat tab.

## Help and tutorial

New `help ailments` (aliases chill, gut-ache, fever, remedy, sickness, raw
meat; listed under the road category); updated `help survival`, `wounds`,
`campsupplies`, `cooking`, `conditions`, `combat` and `eat`. Tutorial: a hint
in the survival lesson.

## Tests

`internal/survival` (conditions, catch and cure, countdown, normalize, frozen
band), `internal/wounds` (festering), `modules/survival` (eating, restart,
decode, cure, status), `modules/company` (`wiring_ailments_test.go`: the real
beginBattle applies, counts, and catches a fever), `modules/camping`
(`remedy_test.go`, `chill_test.go`: the real rest grant), `modules/exposure`
(real walking step catches a chill), `modules/gmcp`, `internal/usercommands`
(eat, help, conditions rows), root `TestShippedAilmentData`.
