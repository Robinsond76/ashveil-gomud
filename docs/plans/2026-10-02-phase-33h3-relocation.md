# Phase 33h3: Company Relocation and Separation — Plan

Design: [33h design, "Final implementation decisions: 33h3"](../designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
(owner-approved 2026-10-02). Branch `phase-33h3-relocation`.

## Invariants

- Never advance global game time; separation counts rounds only while the
  leader is online (`onNewRound`), like the 25b allowance.
- Save before removing a live mob (as `BeginFlight`); a failed save leaves
  the companion where it stands. Never respawn the dead or the fled.
- Company saves go through `m.save()`; separation's countdown rides the
  next save and is never written on its own.
- Ordinary exit following is unchanged.

## Tasks

1. **Domain** (`internal/company`): `Companion.Separation`
   (`reason`, `rounds_left`), `Separated()`; `MemberSeparated` status with
   `RejoinSeconds`; `RelocateCompany(leader, origin, room)` replaces the
   25a two-argument seam; provider tests.
2. **Module** (`modules/company`): `RelocateCompany` takes the companions
   in `origin` or `room`, separates the other live ones; `separate`
   (snapshot, mark, save, detach, notice); `sweepStrays` (two rounds away)
   and `tickSeparations` (count down, rejoin when the leader is free) on
   `onNewRound`; `restoreForLeader` skips the separated; member views,
   `company status`, load/carry/presence exclude them; `SeparationRounds`
   config.
3. **Callers:** `ScriptActor.MoveRoom` (users: take the company, move
   only other charmed mobs from the old room; a companion is never moved
   alone), new `ScriptActor.InBattle()` (+ `.d.ts`, object types);
   voluntary world scripts refuse in battle; quest `roomid` reward;
   expedition completion and its recovery (replacing exit-following);
   death (origin passed); tutorial travel/leave; admin teleport.
4. **Surfaces:** GMCP `separated`, companyview, assessment absence,
   browser Company, Vitals and Combat windows.
5. **Help and tutorial:** new `help separation` (aliases separated,
   left behind, rejoin, teleport, portal, relocation), indexed and linked
   from `help company`; update company, travel, death, morale, mount,
   cargo, readiness; Departure hint; render tests.
6. **Integration tests** through real entry points: scripted move with
   company, refusal in battle, quest relocation, journey arrival, death
   with a companion elsewhere, stray sweep, rejoin timing (battle wait,
   logout pause, restart persistence, failed save), dead/fled untouched.
7. **Review and checks:** independent reviewer subagent on the full diff;
   fix and record findings in `docs/PROJECT_STATUS.md`; `make generate`,
   `make validate`, JS/Lua lint, `go test -race ./...`.
