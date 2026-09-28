# Phase 32e: Company Experience — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)): "How does XP work
for a company?" and "`exp` should show the company."

## The owner's rule (2026-09-28, roadmap decision 1)

> Every member present gets the **full** award, not a split.

Decision 3 of the play-test round adds: `experience` shows the company.
This phase applies the recommendations below under the owner's standing
"carry on" for the 32 series; nothing here was contested.

## Prior-art check

- **Kill XP** (`internal/mobcommands/suicide.go`, `killMob`): the base is
  `XPTL(level-1)/90`, divided by the number of *players* in
  `PlayerDamage`, then per player a level-delta scaler (0.25–1.5), the
  elite bonus, and `user.GrantXP` (config `XPScale`, `xpscale` mods,
  level-up, events). GoMud *player* parties split a party's share evenly.
- **Companion damage is credited to the leader.** `AttackMobVsMob` calls
  `TrackPlayerDamage(charmedUserId, …)`, so a kill made only by companions
  still pays the leader, and the leader is the only one paid.
- **Companions never gain XP.** A companion's `MemberState.Experience` is
  written by `applyState`/`Snapshot` (22b) but nothing raises it, so
  companions stay at their recruit level for ever.
- **Snapshot seams** (22b) already copy a live companion's level and
  experience into its record on autosave, shutdown, copyover, and the
  leader's logout; `applyState` restores them and `Validate(true)`
  recomputes stats from the level. So earned experience is durable with
  no new seam.
- **`experience`** (`internal/usercommands/experience.go`) shows the
  player only (`character/experience` template).
- **Practice mobs** (27c) already return before any XP, so the tutorial
  and replays award nothing.
- **Companion death** (25b) keeps the companion's level and clears gear;
  a dead companion is off the field until resurrected.

## Scope

1. **Companions earn XP.** When a kill pays a company's leader, every
   companion of that company that is **alive, tracked, and in the room
   where the mob died** earns the **same amount the leader was paid**
   (the leader's figure after the level-delta scaler and elite bonus, and
   before `GrantXP`'s own config scale and mods; each character then
   applies those the way it always does).
   Absent, dead, or awaiting-restoration companions earn nothing.
2. **No split.** A company never divides XP: adding companions doesn't
   shrink the leader's share (the divisor already counts only players).
3. **Level-ups.** A companion whose experience passes its threshold
   levels up (`Character.LevelUp`: stats recalculated, health and mana
   refilled, no training or stat points, since companions don't spend
   them). It may level more than once from one award. The leader is told
   in one line: `Corvin reached level 4!`. The line is sent in the
   round's battle narration slot as ordinary text after the kill.
   **No cap:** a companion can pass its leader.
4. **`experience`** lists the company below the leader's own block: one
   row per companion (name, level, experience into the level, to next
   level), online or not is irrelevant, since the record is the source
   (the live mob's numbers when out, refreshed as `company gear` does).
   A solo player sees the same output as today.
5. **Alignment and kill records stay the leader's** (companions' drift is
   21a's; they have no kill log).

## Durable model

No new fields: `MemberState.Experience` and `Level` (22b). The live mob is
authoritative while out; the existing seams persist it. Crash before the
next autosave loses at most the XP since the last save, exactly as gear
changes do; it can't duplicate or invent XP.

## Module and integration

- `internal/mobcommands/companyxp.ashveil.go`: `awardCompanyXP(leaderId,
  leaderChar, amount, roomId)` walks the leader's charmed instances, keeps those the
  `company.FormationProvider` reports as attached companions
  (`LeaderAndKeyForInstance`) that are alive and in `room`, grants XP,
  and levels them. Called from `killMob` in both the solo and the
  GoMud-party branches, with the figure that leader/member was paid (so a
  party member's company earns that member's share).
- `Character.GrantXP` + `LevelUp` are reused; `company.LevelLine(name,
  level)` formats the line.
- `company.MemberView` gains live level and progress (`Runtime.Progress`),
  carried through `companyview.Member`, for `experience`; the view's level
  is now the live mob's while out, so `status` is fresh too. A companion
  that isn't out shows its level alone.

## Constraints and deferrals

- Never advances the world clock; awards happen inside the existing kill
  path on the game loop, under the world lock already held.
- Restart/copyover: covered by the existing snapshots (tested).
- Deferred: a companion level cap, companion training/stat points (32d
  role work), shares for absent members, XP for the enemy side.

## Acceptance criteria

- A solo player's kill XP is unchanged.
- A company of three in one room: the leader and both companions each
  gain the same XP; a companion in another room, or dead, gains none.
- A kill by companions alone pays the whole company.
- A companion crossing its threshold levels up, is refilled, keeps the
  level after logout/login and after a copyover restore, and the leader
  sees the level-up line.
- Party members' companions earn their member's share.
- `experience` lists each companion; solo output unchanged.
- Practice mobs still award nothing to anyone.
- **Player help:** `help experience` (updated, including the GoMud page
  it makes stale), `help company` (companions earn XP), a hint in the
  tutorial's inspection lesson (`experience` now lists the company).
  Tests: page renders; `TestTutorialHelpPointersExist` passes.
