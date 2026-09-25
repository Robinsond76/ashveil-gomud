# Phase 27b: Tutorial Survival and Camp Lessons

The second slice of the [Ashveil tutorial spec](2026-09-23-ashveil-tutorial-design.md),
on the [27a framework](2026-09-25-phase-27a-tutorial-framework-design.md).
It adds stage 4 (Survival) and stage 5 (Camp) before Departure. 27c adds the
practice fight, Alignment, and the browser panel.

The open decisions below were settled by applying this design's
recommendations under the owner's "proceed with next phase" instruction
(2026-09-25).

## Prior-art check

- **Framework (27a):** stages are data with a room index into
  `SpecialRooms.TutorialRooms`; gates run on `companyview.OnRefresh`;
  inspections are counted through `usercommands.OnCommandDone`; progress is
  MiscData (`tutorial-state`, `tutorial-stage`, `tutorial-seen`); the way on
  is a temporary `east` exit between room copies.
- **Survival:** `eat` and `drink` (`internal/usercommands`) call
  `survival.Provision` for an item with nutrition or hydration, for the
  player or a named companion. Every starter kit has a waterskin (30015,
  hydration 40); four of five have a cheese sandwich (30004, nutrition 35),
  the fifth hunter's stew (30019).
- **Inspections:** `weather` (weather module), `temperature` (exposure),
  `strain` (walking), `cargo` (encumbrance). All are module commands.
- **Shelter:** exposure treats a room as indoor by its `indoor`/`outdoor`
  tag or its biome. The tutorial rooms have neither today, so they count as
  outdoors.
- **Camp:** `camp`, `camp fire`, `camp rest` in a room tagged `camping`; the
  rest is a durable, real-time 60 s session (`camping.RestDuration`); on
  completion the company is owed Rested, granted on the next round.
  `camping.RestTierOf` reads the tier a character holds. A camp is keyed by
  leader and remembers its room ID; `camp` refuses a second camp, and
  `camp break` works only in the camp's room. `camping.AbandonForDeath`
  (25a) removes a camp and an inn stay together.
- **Sharpening (23b):** `camp sharpen on` uses a whetstone (item 30, a
  10-use market tool) at rest's end for every eligible blade.

## Decisions

1. **Two stages before Departure.** Survival in a new room 904, the
   Weather Yard; Camp in a new room 905, the Campground.
   `TutorialRooms` becomes `[900, 901, 902, 903, 904, 905]`: stage room
   indexes stay stable and the Gate (903) keeps index 3. The Gate's back
   exit now leads to the Campground. Players already in the course keep
   their stage; one at Formation goes on to Survival.
2. **The Survival gate** is the result, as in 27a:
   - **fed and watered:** a successful provision with nutrition, and one
     with hydration, made by the player (for themself or a companion)
     during the stage. A new `survival.OnProvision` hook reports each
     successful `Provision`; a failed `eat` never counts.
   - **inspections:** `weather`, `temperature`, `strain`, and `cargo`, by
     any alias. A command that isn't registered (its module isn't loaded)
     is dropped from the list, so a missing module never traps anyone
     (`usercommands.IsRegistered`).

   Stages declare their inspections as data; Character's four keep working
   the same way.
3. **Supplies, once.** On first reaching Survival, a player who carries
   nothing edible gets a cheese sandwich, and one with nothing to drink a
   waterskin. It happens once per character (`tutorial-supplied`), and only
   for what's missing, so the kit isn't doubled. Item IDs are config
   (`RationItemId`, `WaterItemId`).
4. **Shelter shown by observation, not mutation.** The Waking Hall is
   tagged `indoor`; the Weather Yard `outdoor`. The lesson asks the player
   to compare `temperature` here with the hall, a short walk back. No zone
   weather or clock is changed. The text explains that walking between
   nearby rooms is local, while `travel` between places is a real-time
   journey that never speeds up the world.
5. **The Camp gate** is a real rest: the player holds Rested or better
   (`camping.RestTierOf`), which only a completed camp rest grants here. The
   Campground is tagged `camping` and `outdoor`. The shipped 60 s rest is
   used unchanged; there is no tutorial override.
6. **Sharpening is explained, not supplied.** The hints describe
   `camp sharpen on` and one stone serving every blade. A whetstone is a
   paid market tool, so the course doesn't hand one out.
7. **Course camps are struck.** A camp in a room copy can't outlive the
   copy: after a restart the copy is gone, and the leader couldn't make
   another camp or break this one. A new `camping.AbandonCamp(leader)` seam
   removes the leader's camp, resting or not; a finished rest keeps its
   recovery and any Rested owed. The tutorial calls it:
   - when the Camp stage passes;
   - on `tutorial skip yes` and on leaving the course;
   - when placing a player (start and resume), since no camp of theirs can
     survive new copies. A rest cut short by a logout is started again (see
     the review amendments).
8. **Waivers (`tutorial next`).**
   - **Survival:** supplies were given and the player still needs to eat or
     drink but carries nothing edible or drinkable.
   - **Camp:** no camping module reports rests (`camping.RestReporting`).
9. **No new locks** in the tutorial. The camping seam takes the camping
   module's own lock, as `AbandonForDeath` does; the tutorial calls it from
   the game loop.

## Constraints

- Never advances the world clock or changes zone weather.
- Progress, supplies, and camps survive logout, restart, and copyover; no
  course camp is left behind.
- Nothing is granted twice.
- `tutorial next` and `tutorial skip` always get a player out.

## Acceptance criteria

- **Pure:** the stage order (Survival and Camp before Departure); stage
  inspections filtered by registration; the Survival gate against sample
  progress.
- **Module:** a provision counts only in the Survival stage; supplies go
  once and only for what's missing; the Camp gate reads the rest tier; the
  camp is struck on passing, skip, leave, and placement; the waivers.
- **Wiring** (through `plugins.Load` with the company, survival, camping,
  weather, exposure, walking, and encumbrance modules and the shipped
  rooms):
  - `eat` and `drink` of the supplied or kit food through the real
    commands, and the four inspections, pass Survival;
  - `camp`, `camp fire`, `camp rest` in the Campground copy start a real
    rest; the Rested tier passes Camp and the camp is struck;
  - `tutorial skip yes` mid-rest leaves no camp;
  - the clock is unchanged.
- **Shipped content:** one room per stage; the Campground is tagged
  `camping`; the Waking Hall is indoor and the Weather Yard outdoor; the
  supply items exist; `help tutorial` covers the new lessons.
- `go test -race ./...`, `make generate`, `make validate` pass.

## Review amendments (2026-09-25)

- **Logout strikes a course camp.** A `PlayerDespawn` in the course strikes
  the camp, so its room copy (whose ID a later player's copy may reuse)
  never keeps a fire lit; the rest is made again on resume. Placement still
  strikes too: after a crash or restart, a rest whose minute ran out
  meanwhile has completed and its Rested is kept.
- **A failed strike is retried.** If camping can't save, `tutorial-strike`
  is set and the strike is retried every refresh, after the course too,
  until it succeeds.
- **Waivers are narrower and cover a missing survival module.** `tutorial
  next` in Survival waives only the meal (when survival isn't available, or
  the supplies are gone and nothing is left); the inspections are still
  asked for. Camp is also waived when survival isn't available, since a
  rest needs it. Anything eaten with hydration counts as something to drink.
