# Phase 27a: Tutorial Framework and the First Lessons

Implements the first part of the
[Ashveil tutorial spec](2026-09-23-ashveil-tutorial-design.md), the last spec
on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

**Split.** The spec has eight stages. Each group needs different content
and systems, so they ship in three slices:

- **27a (this document):**
  - the framework: durable progress, resume, skip, the `tutorial` command,
    and gates that check real results;
  - the Character, Company, and Formation stages;
  - Departure, with a once-only reward.
- **27b:** the Survival and Camp stages (a ration, weather, temperature,
  load, a real camp rest, Rested, and a whetstone).
- **27c:** the practice fight (reach, interception, targeting), the
  Alignment stage, and a browser tutorial panel.

The open decisions below were settled by applying this design's
recommendations, as in Phases 25a–26b, under the owner's "carry on to next"
instruction (2026-09-25).

## Prior-art check

- **Creation** (`internal/usercommands/start.go`): after the name and the
  archetype and starter kit (Phase 22a), `start` asks "skip the tutorial?".
  - **Skip:** moves the player to the start room.
  - **Stay:** copies the tutorial rooms (`TutorialRooms: [900..903]`) into
    per-player ephemeral rooms, sets `RoomIdOnReset = -1`, and moves them
    into the first copy.
  - **Logout mid-tutorial:** `PlayerDespawn_HandleLeave` puts the player in
    the Void (-1). They can't resume.
- **Tutorial rooms** (900–903) are scripted in JS:
  - module-level counters shared by everyone in the room;
  - a strict command allowlist (it blocks `say`, `who`, `company`, and
    more);
  - orb "teachers" (mob 57), a training dummy (58), and a cap and portal to
    the start room at the end.

  The spec forbids exactly these patterns.
- **Recruiters** (Phase 22c) are keyed by room ID. The Waymark Inn offers
  Tamsin (61) and Oswin (62) as free `Tutorial` candidates, claimable once
  per account (the claim is kept on the company record).
- **Seams used:**
  - `companyview.OnRefresh` (after every command and every round, on the
    game loop);
  - `company.FormationFor`;
  - `rooms.GetOriginalRoom` (a copy's template room);
  - `room.SetLocked`.

## Decisions

1. **A tutorial module owns the course.** `modules/tutorial` holds the stage
   list and gates, the `tutorial` command, and placement.
   - `internal/tutorial` is a small seam (`Begin`), so `start` hands a new
     player to it. Without a provider, `start` keeps the engine's path.
   - The JS room scripts and orb teachers go. The module's text is the
     guide, shown in the terminal the same way to Telnet and browser
     players.
2. **Progress is durable, per character.** It lives in the user file's
   `MiscData`:
   - `tutorial-state`: `active`, `skipped`, or `graduated`;
   - `tutorial-stage`: the current stage's ID;
   - `tutorial-seen`: the Character stage's inspections.

   It is saved with the user file, like the 25a checkpoint.
3. **Stages in 27a.**
   - **Character:** the player runs `status`, `inventory`, `experience`,
     and `conditions`, each by any alias. A new `usercommands.OnCommandDone`
     hook reports each handled command, with aliases resolved.
   - **Company:** the company has two living companions. The Muster Yard
     recruiter offers Tamsin and Oswin (the same tutorial claims as the
     Waymark Inn), and `company status` and dismissal are explained.
   - **Formation:** a living companion in the front row and another in a
     rear row. `formation` shows the 3×3 grid, and the stage explains that
     it is tactical, not a map.
   - **Departure:** the gate opens. Walking out grants the graduation cap
     once, sets `graduated`, and arrives at the configured start room.

   Stages are data, so 27b and 27c insert theirs before Departure.
4. **Gates check results, never typed text.** The only exception is the
   Character stage, where running the inspection is the result.
   - Gates are checked on `OnRefresh`. A passed gate is announced, the
     stage advances, the next room's exit unlocks, and the next goal is
     shown.
   - **Stuck on Company:** if both tutorial recruits were already claimed
     (for example after an earlier skip) and the company has fewer than two
     living companions, `tutorial next` lets the player through, without
     another claim.
5. **Rooms.** The four tutorial rooms are rewritten as the Waking Hall
   (900), the Muster Yard (901), the Drill Ground (902), and the Gate
   (903), each with its stage's description.
   - Every player gets their own copies, as before. Exits into the next
     room stay locked until the stage passes.
   - No command is blocked: chat, help, `status`, and recovery all work.
   - Recruiters and room checks resolve a copy to its template room.
6. **Resume.** On `PlayerSpawn` (login, copyover), an `active` player who
   isn't in a tutorial room is given fresh copies. They are placed in their
   current stage's room, with the exits up to it unlocked. The engine's
   logout-to-the-Void still happens first.
7. **The `tutorial` command:**
   - `tutorial` shows the stage (N of M), its goal, a checklist, and hints;
   - `tutorial next` passes an informational or stuck stage when allowed;
   - `tutorial skip` asks for confirmation (`tutorial skip yes`), then
     moves the player to the start room as `skipped`, with no graduation
     reward;
   - once a player has left, the course is done, and there is no restart.
     Claims and kits are never granted twice anyway.
8. **Once-only rewards.**
   - The graduation cap is given only on a transition to `graduated`, so
     skipping never gives it.
   - Starter kits stay with Phase 22a's claim marker, and recruits with
     Phase 22c's claims. The tutorial grants neither.
9. **No new locks.** Everything runs on the game loop. MiscData is
   character state, like the checkpoint.

## Constraints

- Never advances the world clock.
- Progress survives logout, restart, and copyover.
- Nothing the tutorial grants can be claimed twice.
- An unavailable NPC or item never traps the player: `tutorial next` and
  `tutorial skip` always work.

## Acceptance criteria

- **Pure:** the stage list and order; each gate against sample state; the
  progress round-trip through MiscData.
- **Module:**
  - `tutorial` shows the goal and checklist, and `next` and `skip` work;
  - a passed gate advances and unlocks;
  - graduating gives the cap once, and skipping gives nothing;
  - resume places the player in the right stage room with its exits
    unlocked.
- **Wiring** (through `plugins.Load` with the company and tutorial modules,
  the shipped tutorial rooms and config, and the real `start` hand-off or
  `Begin`):
  - inspections by alias pass Character;
  - `company recruit tamsin` and `oswin` in the Muster Yard pass Company;
  - `formation move` passes Formation;
  - walking out of the Gate graduates with one cap, and a second pass gives
    none;
  - a logout and login mid-course resumes;
  - a separate player's skip gives no reward;
  - the clock is unchanged.
- `go test -race ./...`, `make generate`, `make validate`, and
  `make js-lint` pass. The old room scripts are removed.

## Review amendments (2026-09-25)

- **No trap on a failed placement.** `internal/tutorial.Active` reports a
  loaded module. When it is loaded but can't place the player, `start`
  sends them to the start room with a note. The engine's legacy ephemeral
  path runs only without the module, since the shipped rooms no longer
  have its scripts or forward exits.
- **Formation can be waived too.** `tutorial next` waives Company or
  Formation when the company has fewer than two living companions and both
  course recruits are already claimed (for example after a dismissal).
- **Companions travel with the player.** Placement, resume, skip, and
  leaving the course relocate the company (`company.RelocateCompany`),
  since companions only follow on foot. A second `Begin` resumes an active
  course, and a finished one goes to the start room.
- **Deferred to 27c:** any move out of the course (a death respawn, an
  admin teleport) ends it as an unconfirmed skip. No 27a room can kill a
  player; the practice fight must settle this.
- **Known limitation:** the upstream `empty` world still ships the old
  scripted tutorial rooms; with this module loaded, that world's course
  would run under both. Ashveil ships the default world.
