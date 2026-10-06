# Camp consumables — deferred design

Status: recorded at the owner's request on 2026-10-01. **Phase 43a (2026-10-06)
implemented the first slice: fortifying broth, warming draught, cooling salve
and watch incense (see `docs/PROJECT_STATUS.md`); scent paste, weapon oil and
the antidote draught remain open.** Before 43a: no gameplay implemented or phase scheduled. The seven item
ideas are retained; numeric balance, command names and implementation details
below are proposed defaults. Companion design:
[weapon poisons and camp assignments](2026-10-01-weapon-poisons-design.md).

## Purpose and existing foundations

Camp supplies should create choices about the destination, the campsite and
which company member needs protection. Bandages and splints already support
camp wound recovery; whetstones support weapon preparation. These consumables
extend that preparation rather than replacing those systems.

Reuse existing camping/rest, survival, climate/exposure, encounter selection,
item instances, buff persistence and weapon-break mechanics. Camp raids and
Camp Watch depend on the future 33f3 camp-specialist slice; watch incense must
ship with or after those systems. Follow the cargo ownership rules in effect
when this phase is implemented; do not introduce a parallel supply inventory
or restore retired utility skills.

## Catalogue

| Consumable | Proposed benefit | Scope and duration | Decision it creates |
|---|---|---|---|
| Fortifying broth | +5% maximum HP, rounded up, minimum +1 | One member; 15 real minutes after a completed rest | Protect a vulnerable member before a difficult fight |
| Warming draught | Reduce cold-exposure accumulation by 25% | One member; next route segment, at most 15 real minutes | Spend supplies for a cold crossing |
| Cooling salve | Reduce heat-exposure accumulation by 25% | One member; next route segment, at most 15 real minutes | Prepare for a hot region |
| Scent-masking paste | Reduce eligible beast-encounter weight by 25% | Company; next route segment, at most 15 real minutes | Avoid wildlife without protection from bandits |
| Watch incense | +10 percentage points to Camp Watch detection, capped at 90% | Current camp rest only; one attempt | Improve warning rather than prevent a raid |
| Weapon oil | Reduce the weapon's normal break chance by 25% relative | One weapon; next 20 strikes that invoke a break check | Protect valuable equipment alongside sharpening |
| Antidote draught | Halve weapon-poison delivery chance after innate resistance | One member; 15 real minutes | Prepare for venomous opponents |

All seven belong in the future design; a sensible first slice is broth,
exposure protection and watch incense once raids exist. Prices and exact
numbers are finalized against shipped supplies and balance measurements.
Each individual treatment costs one dose per member; weapon oil costs one
per weapon. Scent paste costs one company-sized packet (company cap five);
incense costs one bundle per rest. Give each item a small nonzero weight and
clear shop descriptions stating scope, benefit and duration.

### Effect boundaries

Broth raises the current wound-limited maximum by its bonus without removing
wounds. Grant the benefit after ordinary rest recovery: current HP does not
increase just from consuming broth. Subsequent healing can reach the boosted
limit. On expiry, clamp current HP to the ordinary wound-limited maximum;
expiry cannot kill a living member. It creates no repeatable heal-on-reapply
loop. Broth contributes ordinary nourishment through existing survival rules.

Warming draught and cooling salve reduce harmful exposure accumulation only;
they do not alter world temperature, weather, terrain, travel speed or worn
warmth. Warming gives no heat protection; cooling gives no cold protection.
Lock the applicable route benefit at departure under existing travel modifier
rules. Expiry during a route ends protection prospectively, without changing
exposure already incurred. A route cancellation consumes the route allowance;
a resumable interruption remains part of the same segment.

Scent paste reduces only beast encounter probability. Preserve other event
probabilities by assigning removed beast weight to a no-encounter outcome;
do not redistribute it to bandits, discoveries or other encounters. If a
route lacks typed beast events or a no-encounter outcome, add that modelling
before enabling this item. It does not alter scripted attacks, mandatory
encounters, camp raids or a fight already started. Define beast eligibility
explicitly in encounter data, never by matching mob names.

Watch incense requires a qualifying Camp Watch; it does not invent a watcher.
It modifies the detection roll only, not raid chance, raid timing or enemies.
Where base specialist detection already reaches 100%, preserve that existing
certainty; incense cannot lower it. Otherwise cap the enhanced chance at 90%.
Consume the incense when rest begins, before its raid roll, and carry the
modifier in the saved rest record. The benefit is spent even if no raid comes
or the rest is interrupted; it never retroactively affects a saved raid roll.

Weapon oil affects the normal break probability only, with a minimum positive
chance retained when the weapon normally can break. It does not cancel forced
breaks or other damage to equipment. Round the relative reduction using the
engine's probability precision, rather than turning a small positive chance
into zero. Spend one oil strike per actual break-check invocation, independently
of whether the check breaks the weapon. Oil coexists with sharpening and poison
without resetting them; it cannot repair or restore a broken weapon.

Antidote draught prevents new delivery only; it does not cure an active poison.
For proposed weapon-poison rates, normal targets fall from 40% to 20%, resistant
targets from 20% to 10%, and immune targets stay at 0%. Keep the same single
delivery roll. The separate curative antidote and `curepoison` remove existing
poison. Legacy scripted venom needs an explicit compatible susceptibility
hook before the draught can claim protection from it; help must state coverage.

## Commands and limits

Proposed `camp supplies` lists doses, eligible targets and active benefits.
`camp prepare [item] [self|member|company]` applies or queues the appropriate
supply. `camp prepare status` shows benefits, queued rest supplies and expiry.
Weapon oil uses `camp oil [member] [main|off]`. All preparation requires an
established camp, a stationary/awake actor and no company combat or active
rest. Validate ownership, presence, target scope and sufficient stock before
spending anything. Rejections state the reason and consume nothing.

One **personal preparation benefit** per member covers broth, warming draught,
cooling salve and antidote draught. Existing meals, Rested/Well Rested and
medical treatment retain their established rules. Never silently replace an
active personal benefit: require explicit `camp prepare clear [member]`,
which refunds nothing. Reusing the same active supply neither consumes a dose
nor refreshes its duration. One scent treatment per company and one incense
bundle per rest are independent of personal benefits. Oil is per weapon.

Broth and incense are queued for the next camp rest. Reserve actual supply
instances so they cannot also be sold or used. Cancelling the queue before
rest returns the unconsumed supply. At rest start, consume queued supplies
once; incense takes effect immediately, broth is granted only on successful
rest completion to its still-present living recipient. A broken or abandoned
rest grants no broth benefit and refunds no consumed portion. Queued broth
blocks another personal preparation for that member until cancelled or used.

Other benefits start when applied. Scent/exposure route allowances last until
the next segment completes/cancels or their UTC deadline, whichever comes
first; idle time at camp does not pause the deadline. No automatic spending
on camp creation, login or rest completion.

## Acquisition and later crafting

Begin with finished supplies sold at suitable provisioners/apothecaries:
broth portions, draughts, sealed salve/paste pots, incense bundles and oil flasks.
Reuse existing shop stock, pricing and settlement restrictions where supported.
Set costs against bandages, meals and whetstone uses so routine travel does not
require purchasing every benefit.

Future camp cooking can make broth; future camp brewing can make draughts,
salves, paste and incense using appropriate kits and known recipes. Example
fictional ingredients: hearty roots and stock for broth; emberleaf for warming;
coolmint for salve; bitter moss for masking paste; watchsage for incense;
clearroot for antidote draught. Oil can use purchased oil and binding resin.
Use the eventual camp recipe service, deterministic yields and transactional
ingredient consumption. Do not add a new skill tree merely for these items.

## State and recovery

Personal buffs belong to the recipient, company scent to the company/route,
incense to its rest record, and oil to the mutable weapon instance. Save
identity, potency, absolute UTC expiry where applicable and remaining route
or strike allowance. Do not share mutable pointers across copied items.

Retain effects and reservations through save/load and copyover without
refreshing deadlines or granting supplies twice. Offline time counts toward
absolute expiry. Validate expiry before applying any modifier. Death clears
personal preparation; removal from a company does not transfer another
member's buff. Incense ends when its rest ends. Oil stays with its weapon on
trade, storage or equipment changes and never transfers to replacement gear.
Rest abandonment uses the existing exactly-once cleanup path.

Timers never advance global game time, block the server or manipulate another
company's weather/encounters. Use the authoritative game-loop and durable rest
transitions for consumption and grants. Display scope, remaining time/uses,
queued supplies and blockers in camp/status/conditions and relevant browser
views; use existing combat pacing for effects observed during battle.

## Acceptance criteria and future implementation plan

- Command integration verifies all seven items, recipient scopes, company
  restrictions, reservations, cancellation and insufficient-stock handling.
- Rest integration verifies one consumption/grant across interruption,
  completion, abandonment, death, member departure, restart and copyover.
- Broth respects wound limits, ordinary rest healing and safe expiry; no free
  healing from reapplication. Personal benefits never stack or silently replace.
- Climate protection applies only to its exposure type and remaining covered
  route time; scent changes beast probability without raising other events.
- Incense affects only the saved detection outcome and preserves existing
  certain detection. Test raids both spotted and unspotted through real rest.
- Oil composes with sharpening/poison and preserves forced breaks, positive
  normal break chance and per-item consumption through all attack directions.
- Antidote prevention composes with resistance/immunity; curative antidotes
  remain separate. Verify poison mechanics through actual combat entry points.
- Save/load and companion/item transitions preserve durations and uses; no
  global clock changes or duplicate grants.
- Ship `help campsupplies`, preparation/oil commands, updated `help camp`,
  food/exposure/poison help where affected, keywords and aliases, and Camp
  tutorial guidance. Test help rendering and tutorial pointer resolution.
- Measure combat and travel outcomes at representative levels before final
  prices/potencies. Obtain the normal independent phase review before merging
  implemented gameplay.

The later implementation plan must include state/commands, rest reservations,
climate/route/raid hooks, weapon/poison integration, shops, help/tutorial/browser
presentation, persistence tests, balance measurement and review. Implement
against the then-current phase 33 systems; this record does not reorder the
owner's existing roadmap.
