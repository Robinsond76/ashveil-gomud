# Phase 11: leader-as-interceptor

## Design decisions

1. **The gap.** `resolveAttackTargetCompanionOnly` (in
   `internal/hooks/combat_formation.go`, used by `gateEnemyAttacksCompanion`
   for the "hostile mob attacks my companion" mob-vs-mob direction)
   deliberately never redirects an intercepted attack to
   `company.LeaderMemberKey`, because doing so crosses `combat.Attack*`
   functions mid-resolution: the attack started as `AttackMobVsMob` (hostile
   → companion) but a leader interceptor needs `AttackMobVsPlayer` (hostile
   → player). The mob-vs-player direction already solved the *opposite*
   crossing (`gateMobVsPlayerAttack`/`resolveInterceptedMobAttack`: an
   attack starts as `AttackMobVsPlayer` against the leader, gets redirected
   to a companion interceptor, and is resolved as `AttackMobVsMob` instead).
   This pass builds the missing symmetric case.

2. **Approach: mirror `gateMobVsPlayerAttack`'s three-state contract.**
   `gateMobVsPlayerAttack` returns `(handled, ok)`: `handled=true` means it
   already resolved the (redirected) attack itself and the caller should
   skip its own attack call and `continue` the round. `gateMobVsMobAttack`
   currently returns only `(*mobs.Mob, bool)` — no `handled` — because
   until now every redirect it could produce stayed within `AttackMobVsMob`
   (companion target, so just swap `defMob`). Change its signature to
   `(*mobs.Mob, bool, bool)` (target, handled, ok), matching the pattern.
   Only `gateEnemyAttacksCompanion` ever needs to set `handled=true`; the
   attacker-is-companion branch (`gateCompanionAttacksEnemy`) is unaffected
   (an enemy party has no "leader" to jump `Attack*` functions for).

3. **`resolveAttackTargetCompanionOnly` is no longer needed.** With the
   crossing now handled by resolving-and-returning-handled instead of by
   refusing the redirect, `gateEnemyAttacksCompanion` can call the plain
   `resolveAttackTarget` (same one used everywhere else) and simply check
   whether the resolved key is `company.LeaderMemberKey` after the fact.
   Delete `resolveAttackTargetCompanionOnly` and its two dedicated tests;
   `resolveAttackTarget`'s existing test suite already covers the
   redirect-to-front-row logic this reuses.

4. **New resolver: `resolveInterceptedAttackOnLeader`.** Mirrors
   `resolveInterceptedMobAttack` (the existing companion-interceptor
   resolver) but calls `combat.AttackMobVsPlayer(mob, leader)` and operates
   on `*users.UserRecord` fields instead of `*mobs.Mob` fields — copied from
   `NewRound_DoCombat.go`'s existing mob-vs-player attack-resolution block
   (buffs, messages, the charmed-mob-assist loop, offhand equipment-break)
   plus a `CharacterVitalsChanged` event so the leader's own client health
   bar updates (the existing companion-defender resolver has no equivalent
   event; mob health isn't pushed to any client the same way, but a
   player's is). Like `resolveInterceptedMobAttack`, this never touches
   `mob.Character.Aggro` — it stays pointed at the companion's instance id,
   so the interception is recomputed fresh every round and combat reverts
   to the companion automatically once the leader no longer blocks.
   `mob.Character.Aggro`'s pre-existing "re-affirm/end aggro" bookkeeping
   (`NewRound_DoCombat.go` lines ~973-981) is deliberately skipped for the
   same reason the companion-interceptor resolver skips it.

5. **Fails open exactly like every prior pass.** No company formation, no
   resolvable attacker column, or a legality check that already fails,
   fall through to the pre-existing behavior (attack the original
   companion target, or skip the round) unchanged. Only the new
   "interceptor resolves to the leader" branch is new behavior.

## Tasks

- [x] Update `gateMobVsMobAttack`/`gateCompanionAttacksEnemy`/
      `gateEnemyAttacksCompanion` to the three-state
      `(target, handled, ok)` contract; delete
      `resolveAttackTargetCompanionOnly` and its two tests.
- [x] Add `resolveInterceptedAttackOnLeader` in `combat_formation.go`.
- [x] Wire the new three-state return through the mob-vs-mob call site in
      `NewRound_DoCombat.go`.
- [x] Add a focused test proving `gateEnemyAttacksCompanion`-style
      resolution now reaches the leader (via `resolveAttackTarget` itself,
      the pure piece — the two adapters stay untested at the engine-glue
      layer, same precedent as every prior combat-wiring pass).
- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Update `docs/PROJECT_STATUS.md`.
