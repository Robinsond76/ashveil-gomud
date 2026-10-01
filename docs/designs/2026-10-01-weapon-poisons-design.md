# Weapon poisons — proposed design

Status: deferred implementation design, recorded at the owner’s request on
2026-10-01. Camp assignment command requested by the owner; balance defaults
remain proposals. No gameplay implemented. Based on local
54c11ac8 (33f1). Fetch/pull from origin failed because the configured proxy
was unreachable; remote freshness is unverified.

Related deferred design: [camp consumables](2026-10-01-camp-consumables-design.md),
including preventative antidote draughts and weapon oil.

## Intent and settled scope

The owner wants temporary poisons on bladed weapons, initially purchased at
shops, with special and stronger poisons eventually crafted at camps using
ingredients. Applying a poison should be useful expedition preparation and
allow company members to fill different combat roles.

Everything below is a proposed default, rather than an approved balance rule.

## Existing foundations

Phase 23b already supports `sharpen` and `camp sharpen` outside combat, with
per-item persistent edges (+1 damage for 20 successful strikes). Its command
spends one whetstone use per member, including both equipped blades.
Poison instead costs one dose per blade, because each coated weapon can
independently deliver an effect. Dual wielding costs two doses.

`internal/items/items.go` owns mutable weapon instances. The combat engine
reads copied weapons and reports edge consumption for application back to
equipment (`internal/combat/combat.go`, `calculations.go`). Poison must use
that same ownership discipline, not mutate a temporary weapon copy.

`internal/status` supplies combat-round statuses, ticking and clearing at
battle end. The legacy Poisoned buff (13) ticks in game rounds and carries a
`poison` flag; `curepoison` cancels buffs with that flag. New weapon poisons
use combat-round effects with that flag and do not reuse buff 13's timing.
Phase 33f3 camp crafting and cargo rules are planning dependencies, not assumed
shipped functionality. Do not revive retired utility skills for this feature.

## Applying and spending a coating

- One purchased vial is one dose for one equipped blade. Reuse the existing
  sharpening blade eligibility; no blunt weapons, bows, claws, or ammunition
  in the first release.
- Apply inside or outside camp, only while stationary, awake, and the whole
  company is out of combat. No application during travel or an active rest.
- Proposed commands: `coat [poison] [self|member] [main|off]`, `coat status`,
  `coat clear [self|member] [main|off]`; `camp coat` is an alias. Default target
  is self, main hand. No automatic application or company-wide spending.
- A coating lasts **10 real minutes from application or 8 damaging contacts**,
  whichever ends first. Time elapses through travel, logout and downtime;
  reconnecting does not refresh it. Display both time and contacts remaining.
- Only a blade hit that causes final positive physical HP damage can deliver
  poison. Misses, parries, dodges and fully absorbed hits consume no contact.
  Immune/resistant targets still use a contact when physically wounded.
- Each eligible contact has a **40% delivery chance**, once per individual
  strike. It consumes a contact regardless of the delivery roll. Critical
  hits do not increase the chance. No spell or reflected-damage delivery.
- One coating per weapon. Existing active coatings cannot be refreshed or
  replaced accidentally: clear the old coating first; clearing returns no
  dose. Sharpening and coatings coexist without resetting either resource.
- A dead target receives no new effect. Interception applies the poison to
  the actual struck guardian, rather than the originally selected victim.

## Camp poison assignments

The owner requested a camp command to choose which poison goes on which blade
user. Provide an explicit preparation list, rather than requiring players to
remember a sequence of individual applications:

```text
camp poison                         # show blades, coatings, doses and assignments
camp poison assign [member] [main|off] [poison]
camp poison unassign [member] [main|off]
camp poison preview                 # show exact dose costs and any blockers
camp poison apply                   # consume doses and coat assigned blades
```

`self` selects the leader; companions use the existing company member selector.
Each equipped eligible blade gets its own row. A dual-wielding member can have
different poisons assigned to main and off hand. `camp poison assign` edits the
preparation list only: it consumes nothing. `camp poison apply` executes the
current validated list, with one dose per blade. Keep `coat` for direct field
application; both entry points share eligibility, costs and combat restrictions.
The preparation list requires an established camp, with no active rest or battle.

Preview example:

```text
You       main  longsword     Bitterleaf   1 dose
Corvin    main  dagger        Leechbane    1 dose
Corvin    off   short sword   Mirethorn    1 dose
Needed: Bitterleaf 1, Leechbane 1, Mirethorn 1.
Ready to apply to 3 blades.
```

Persist assignments on the company as member ID, equipment slot and poison ID;
they are preferences, not reservations or active coatings. Resolve current gear
at preview/application time and show its name. Removed/dead/absent members,
ineligible blades, insufficient doses and differently coated blades block the
whole application: report all blockers and spend nothing. Matching active
coatings are skipped without refreshing them or spending a dose. If every row
is skipped, report that no application is needed. Remove assignments when their
member permanently leaves the roster; do not auto-apply on camp creation, rest,
login, or equipment changes. There is no silent partial company application.

Validate and apply in one authoritative game-loop operation, including dose
consumption and companion gear snapshots. Extend acceptance tests to cover
mixed poisons, both hands, missing stock, changed gear, duplicate application,
member removal and save/load of assignments. Ship these commands in `help
poisons` and `help camp` with tutorial guidance. This command is part of the
shop-only poison launch; crafting remains deferred.

## Poison catalogue

Effect durations below are combat rounds (currently about 8 seconds each).
Effects begin on delivery; damage ticks start next round. Values are starting
points to measure against the combat balance harness.

| Poison | Effect after delivery | Duration | Tactical use | Availability |
|---|---|---|---|---|
| Bitterleaf | 1 HP damage per round | 3 rounds | Cheap attrition against living foes | Common shop |
| Leechbane | Healing received reduced 25%, rounded down; minimum 1 HP for positive heals | 3 rounds | Pressure on healers and regenerating foes | Apothecary |
| Leadroot | Physical damage dealt reduced 15%, rounded down; minimum 1 damage for an otherwise damaging hit | 2 rounds | Soften a dangerous striker | Apothecary |
| Mirethorn | Dodge chance reduced 10 percentage points, clamped at zero | 2 rounds | Help the company catch evasive foes | Apothecary |
| Whisperbane | Chant-break chance from a damaging weapon hit increased 15 percentage points, capped at 90% | 2 rounds | Make casters easier to interrupt; never automatic silence | Specialist shop, later recipe |
| Gravebloom | 2 HP damage per round; healing received reduced 25% | 3 rounds | Rare combined pressure | Future camp-only recipe |

Launch recommendation: Bitterleaf, Leechbane, Leadroot and Mirethorn. Add
Whisperbane and Gravebloom with the later camp-crafting slice. Leechbane never
reduces resurrection, wound treatment, or out-of-combat rest recovery.
Leadroot changes physical strike damage, not spells, status ticks or stat
scores. Mirethorn changes dodge only, not parry, shield block, movement, or
the future Speed action meter.

## Stacking, resistance and counterplay

A victim can carry **one active weapon-poison effect**. It never stacks or
refreshes while active, even from multiple company members or both hands.
A contact against an already affected target still consumes coating, but
performs no delivery roll. This prevents repeated hits from locking enemies
under a debuff or multiplying damage; mixing blades remains useful when
company members fight different targets. Legacy poison remains independent.

Use data-driven poison susceptibility: normal, resistant (delivery chance
halved to 20%), or immune (0%). No second resistance roll. Default living
creatures to normal and explicitly mark undead, constructs and other
appropriate shipped templates immune; do not infer physiology from names.
Show immunity on inspection and once on first contact in a battle.

`curepoison` removes every poison-flagged effect, including legacy poison.
Add a shop antidote as an ordinary combat action for self or a present company
member; it cures poison but grants no immunity. Neither ordinary healing nor
wound treatment removes poison. New weapon-poison effects clear at battle
end, withdrawal, or death, following combat-status lifecycle rules; the blade
coating remains until its own deadline or contacts run out. Poison does not
create lasting wounds or interrupt chants through its damage ticks.

## Shops and future camp recipes

Start with one-dose, clearly labelled vials sold through existing shop stock
and pricing, including stock in the Dunmar/Old Kings Road preparation loop.
Give each vial a small nonzero carried weight. Use existing settlement pricing
rules where that shop path supports them; no parallel poison economy.
Bitterleaf should cost roughly one ordinary bandage, other common poisons
roughly two, and specialist poisons roughly four, translated into actual gold
prices during implementation after inspecting shipped supply prices.

Future `camp brew [recipe] [count]` requires an established camp, appropriate
alchemy kit, known recipe, and the specified fictional ingredients. Reserve
and consume ingredients transactionally under the camp/cargo ownership rules;
failed validation costs nothing. Craft sealed vials, then apply them normally.
Recipes make deterministic quantities; no random failure or free duplication.
Do not advance the world clock or use blocking timers.

Example fictional recipes:

- Bitterleaf: bitterleaf sprig + binding resin + empty vial.
- Concentrated Bitterleaf: bitterleaf sprig + marsh-adder venom + binding resin
  + empty vial; 2 HP per tick for the same 3 rounds.
- Whisperbane: hushcap spores + dusk-moth dust + binding resin + empty vial.
- Gravebloom: gravebloom petals + wraith ichor + binding resin + empty vial.

Potent variants improve an effect within explicit caps, rather than increasing
delivery chance, duration and coating lifetime together. Keep damage at or
below 2 HP per round initially; healing suppression at or below 40%; weakness
at or below 20%; dodge reduction at or below 15 percentage points. Add a camp
specialist only through a separate approved specialist design.

## State, integration and recovery

Store coating kind, potency, absolute UTC expiry, and contacts remaining as
plain instance values on the item. Never use shared mutable pointers. Preserve
those values through equipment swaps, trade, cargo, companion snapshots,
save/load and copyover. Validate expiry at every read/use so an offline weapon
cannot deliver stale poison. Stashing a weapon does not pause its timer.

Store active poison as the existing durable buff state with combat-round
remaining duration and poison identity/potency. Ensure poison applied this
round cannot tick this round, regardless of attacker ordering. Restoring a
battle must retain remaining duration without double ticking; disconnected
characters follow existing battle lifecycle behavior. Ending a battle removes
only its transient effects, never an unspent weapon coating.

Wire delivery into the final physical strike result and the actual resolved
victim for player/player, player/mob, mob/player and mob/mob, including companions,
guardians, off-hand and multi-strike attacks. Apply consumption once through
the equipment result path. Use ordinary battle damage/death/XP attribution
for lethal poison ticks; preserve one death and one reward. Emit delivery,
tick, cure and expiry through combatstream/combatpace, rather than unpaced
messages. Narration says the effect name and actual damage without printing
failed random rolls on every strike.

## Acceptance criteria for implementation

1. Application validates target, ownership, blade, company combat state and
   available dose before mutation; invalid actions spend nothing. Clear and
   reapplication cannot mint doses or reset unrelated sharpening.
2. Save/load, copyover, companion despawn/restore and item transfers retain
   coating state. Expired coatings cannot deliver after restart or logout.
3. All four attack directions, both hands, multi-strike and guardian targets
   deliver and consume correctly. Zero-damage hits and immunity are covered.
4. Tick timing, resistance, one-effect limit, cure, withdrawal, fight end and
   poison-kill rewards pass integration tests through real combat entry points.
5. Healing and damage modifiers compose with existing mechanics and never
   affect wound treatment, resurrection, magic or future action counts.
6. Compare launch poisons in the combat harness at representative levels and
   against healers, evasive foes and resistant targets. Preserve the 30g target
   for 10–15-round even fights; adjust proposed potency/chance if needed.
7. Ship `help poisons`, `help coat`, and antidote guidance, keywords/aliases,
   links from `help combat`, `help camp` and preparation help, plus a tutorial
   preparation hint. Test rendered help and tutorial pointer resolution.
8. Inventory, company gear, conditions and browser views show coating type,
   time and contacts left; active poison shows effect and rounds left.
9. The implementation plan includes command/state, combat delivery/lifecycle,
   shops/antidotes, presentation/help/tutorial, verification and independent
   review tasks. Camp recipes are a later plan, not part of shop-only launch.

## Decisions proposed for owner review

Approve or adjust the 10-minute/8-contact coating, 40% delivery chance,
one-effect-per-victim rule, four-poison shop launch and battle-only victim
effects. The owner requested recording this design for future phase implementation,
including camp poison assignments. Scheduling and balance approval remain
separate; this document does not change game behavior.
