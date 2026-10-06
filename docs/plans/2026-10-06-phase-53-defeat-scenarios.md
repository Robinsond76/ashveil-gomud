# Phase 53: defeat scenarios

Spec: phase 53 in [outward survival phases](2026-10-06-outward-survival-phases.md).
Built under the owner's full-autonomy rule (2026-10-06); each decision below
carries its reason.

## What ships

A new death can end in a scenario instead of the church: **rescued**,
**captured**, **left for dead** or **robbed**. Scenarios are YAML rows in
`modules/death/files/data-overlays/config.yaml` (`Scenarios`), keyed by the
killer's race name or mob group (`Foes`) and the zone the company fell in
(`Zones`), so the replacement world writes its own. The church and the
level loss stay as the fallback when no row fits (or the table is empty).

Flow: `suicide` asks the provider's optional `ScenarioProvider.ClaimDefeat`
(new, `internal/death`) after the killer is known and before the death's
penalties. A claim skips the engine's drops and corpse, and `Respawn`
applies the scenario instead of taking a level. The claim (`defeat-scenario`
plus the killing foe's group) lives in the character's `MiscData` beside the
death's pending mark, so a restart mid-defeat resumes the same scenario and
never rolls again (`TestDefeatScenarioResumesAfterAFailedWakeWithoutRerolling`).

| Kind | Where they wake | What it costs |
| --- | --- | --- |
| rescued | a settlement in the fall zone, else the checkpoint church | every member Hungry (hunger at most 45) and Exhausted (fatigue at most 20), through the survival drain |
| captured | the scenario's capture room (test content: Brigands' Tent, 91001) | pack locked in a chest (`Character.Seized`, saved with the items), 25% of gold gone, the rest held; leader Bound (buff 9301, `no-go`, 6 rounds); two weak guards (mob 86, level 2); `reclaim` returns the goods once the guards are down |
| left for dead | where they fell | foes sent away; a lasting wound (20% of max health) on the leader and each companion with them |
| robbed | where they fell | foes sent away; 20% of gold and 25% of loose goods (at most 4 items) gone for good |

Not touched: companions who die still follow the 25b resurrect window;
the test area's wake override still wins (no scenario is claimed there);
`company` relocation brings the companions with the leader as before.

## Decisions

1. **No level loss in a scenario** (the spec's default): the recovery is the
   cost. The church fallback keeps the level loss, so a world with no table
   plays as before.
2. **Unknown killer** (bleed, hunger, poison) fits only rows that name no
   foes: rescued. A scenario never claims a death it can't describe.
3. **Gold in a robbery comes out of the leader's gold, which is the
   treasury.** The spec says both "some gold gone" and "the treasury safe";
   one number can't be both, and a robbery that takes nothing isn't one. The
   loss is a fixed share (20%, 25% when captured) and the camp-theft
   protections (worn gear, quest tokens, keys, camp gear 45-50) still hold
   for goods.
4. **The captured pack is held, never destroyed.** `Seized` rides the user
   file with `Items`, moved in one step, so a restart can't lose or double
   an item. Leaving the capture room without the chest keeps it held until
   the leader returns; there is no timer. Only the quarter of gold is lost.
5. **Guards stand, they do not attack.** The company wakes at half health;
   the player chooses when to fight, and the guards are the "weak group from
   the zone's lowest band" (spawned from the scenario row, tagged with a
   per-company spawn group so two captures don't share guards).
6. **Economy:** nothing here pays out. Robbed goods vanish rather than move
   to a loot table, and captors drop only their normal mob loot.
7. **Foes gone** is real: the killing mob, its spawn group and any ambush or
   encounter set on the leader are removed from the room (room spawns refill
   on their own timer); companions are never touched.
8. **Web client:** the Inventory tab shows "Held by your captors: N items
   and G gold in <room>" with a Reclaim button in the room (`Char.Inventory`
   `seized`). The wake text already prints to the log, and wounds and needs
   already show in the Conditions and status panels.

## Help and tutorial

New `help defeat` (aliases: captured, robbed, left-for-dead, reclaim, ...);
`help death`, `help combat` and `help adventure` rewritten or linked; the
Departure lesson hint names defeat scenarios. Tests: `TestDefeatHelp`,
`TestTutorialHelpPointersExist`.

## Tests

`modules/death/wiring_defeat_test.go` drives each kind through the real
`suicide` command with `plugins.Load`, the shipped overlay and the shipped
brigand camp; `internal/death/scenario_test.go` covers the table, weights
and pack split; `TestShippedDefeatScenarios` pins the content;
`modules/gmcp` covers the held-goods payload.
