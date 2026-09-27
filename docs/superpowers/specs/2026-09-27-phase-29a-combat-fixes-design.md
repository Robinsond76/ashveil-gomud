# Phase 29a: Combat Fixes from the 5v5 Simulation — Design

The first slice of the [combat roadmap](2026-09-26-combat-presentation-roadmap.md)
(build order decided 2026-09-27: 29a first). The findings are recorded in
the [29a findings spec](2026-09-26-combat-fixes-design.md); this document
settles the design for implementing them.

The owner asked to "begin work on next phase" (2026-09-27). The one open
decision (F1: reassign or refuse) and the two smaller ones below were
settled by applying this design's recommendations, as earlier phases did
under "carry on". They are marked **(recommendation applied)** and are
listed for the owner in `docs/PROJECT_STATUS.md`.

## Reproduction with the shipped config (2026-09-27)

The findings spec asked for F2 to be reproduced against the shipped
config before fixing. That was done first, through the real round
(`hooks.DoCombat`, `hooks.IdleMobs`/`HandleIdleMobs`, queued mob commands)
with `_datafiles/config.yaml` loaded (`HPBase: 5`, 4-second rounds):
- **Company:** Aria (level 3), Tamsin, Oswin, Garrick, and Ysolde,
  summoned and placed as in the simulation, with Aria in column 3.
- **Enemies:** five non-hostile bandits sharing `groups: [bandits]`.

What happened:

1. **F1 reproduced.** Aria attacked the captain in the enemy's column 1
   and got `You can't reach that target from here.` for every one of the
   ten rounds the captain lived. She never swung.
2. **After the captain died, Aria stopped for good.** The mob's `suicide`
   clears every player's aggro on it before the round in which
   `reassignPlayerTarget` would run. So the leader is never reassigned
   when *someone else* makes the kill.
3. **The killer stops too.** On a kill, `DoCombat`'s own
   `Health <= 0 → EndAggro` branch clears the killing member's aggro
   outright. Only companions who were still pointed at the dead mob were
   reassigned. Garrick (who killed the captain) and Ysolde (who killed
   the slinger) stood idle until something happened to attack them.
4. **F2 reproduced, and worse than recorded.** The four other bandits
   never joined while their captain was cut down.
   - `mobs.MakeHostile` only runs after the *leader's* blow lands, and
     Aria never landed one. Companion (mob-vs-mob) attacks never make a
     group hostile.
   - A bandit joined only when it was struck itself, through the "mobs
     get aggro when attacked" retaliation.
   - So the party was fought one member at a time, with the rest
     standing by.
5. **F3 stays withdrawn.** With the shipped config, level-1 Tamsin and
   Oswin have 6 HP.
6. **Found while reproducing:** companions summoned but not yet placed in
   the formation fail open at every gate (they can attack anything), but
   `reassignCompanionTarget` needs a formation position. So an unplaced
   companion whose target dies goes idle.

## Prior-art check

- **`internal/hooks/combat_formation.go`:**
  - the three attack gates (11c), each fail-open when there is no
    formation;
  - `reassignPlayerTarget`/`reassignCompanionTarget` (11b), which only
    run when the attacker's own target is found dead or gone *during that
    attacker's turn*;
  - `reassignEnemyTarget`, which picks the weakest legal member of
    `firstHostilePartyInRoom`.
- **`engagement.AssignTarget`** (11b): the weakest/strongest/random
  choice filtered by a legality predicate. `engagement.Engagement` is a
  marker type with no callers yet.
- **`formationcombat.Legal`/`LegalTargets`** (11c): pure, and recomputed
  from who is alive.
- **`mobparty.Assemble`** (11a): never cached. The formation is
  re-derived by EHP from whoever is in the room, the dead included until
  they are removed.
- **Hostility:** `mobs.MakeHostile(group, userId, rounds)` with
  `MinutesToRounds(2) - Perception`, set only in the player-vs-mob branch.
- **`formation reach <member>`** (`modules/company/formation.go`) answers
  against the company's own formation, with plain melee, always.

## Decisions

1. **One upkeep pass at the start of each combat round** keeps every
   engaged fight whole. It runs before any attack is resolved, and it
   covers F1, the kill cases, and F2 together.
   - **Which rooms:** the room of each online leader with a company.
   - **Engaged:** a company and an enemy party (`mobparty.Assemble` over
     the room's non-charmed mobs) are engaged when any living member of
     one side has a plain-attack `Aggro` on a member of the other in that
     room.
   - **For each engaged pair:**
     1. **Hostility is refreshed** for the party's groups against the
        leader (the same `MakeHostile` duration the leader's own blow
        uses), so it can't wear off mid-fight.
     2. **Every living company member** in the room with no target,
        or with a target that is dead, gone, or out of reach, takes the
        weakest legal member of the party. This covers the leader too.
        - **"Out of reach"** means the attack gate would skip it (`You
          can't reach that target from here.`). A target the gate lets
          through, legally or by 11c's front-row interception, is never
          moved. So an attack on a shielded back-row member is still
          caught by the one in front, exactly as before (found by the
          tutorial's practice-fight wiring test during implementation).
     3. **Every living party member** is treated the same way against the
        company: the weakest legal company member.
   - **Nothing legal:** a member keeps its current target, and the gates
     go on skipping it until something changes (11c's self-healing model).
   - **Left alone:** members whose `Aggro` is a spell cast, a backstab,
     or a shot through an exit, and downed players.
   - **Same-room shots count:** a same-room `shoot` (`Aggro.Type`
     `Shooting` with no exit, as Ysolde's sling uses) is an ordinary
     attack. It is counted and retargeted, and keeps its type.
2. **F1: reassign, not refuse (recommendation applied).**
   - **What the leader sees:** `You can't reach the bandit captain from
     here. You turn on the bandit cutthroat.` They swing at the new
     target that same round.
   - **Why not refuse the `attack` up front:** legality changes every
     round, as front-row members fall and the enemy formation re-forms.
     A target refused now may be reachable in two rounds, and a legal one
     may not stay legal. Reassigning is also what companions already do.
3. **The leader rejoins an ongoing fight (recommendation applied).** If
   the leader's target died to someone else's blow, or they had none,
   they take the next legal target while their company is engaged, and
   see `You turn on the bandit slinger.`
   - **Leaving a fight:** `flee` or walking out still works, since only
     members in the room are touched.
4. **Enemies keep fighting as a party (recommendation applied).** A party
   member whose company target is out of reach turns on a legal one,
   instead of waiting forever. This is the enemy-side mirror of F1. Idle
   members join the fight at once.
   - **Before this phase:** 11b declined enemy reassignment as "enemy AI,
     out of scope". Here it is limited to keeping the party *engaged*.
     There is still no preference beyond "weakest legal"; 30c's
     personalities replace that.
5. **Unplaced company members fail open, on both sides.**
   - **As attackers,** they fail open for reassignment as they do at the
     gates: any living party member is eligible.
   - **As targets:** before this phase, an unplaced member could **never
     be struck**. `resolveAttackTarget` treated "not in the formation" as
     illegal, so a company that never set its formation was invulnerable.
     The reproduction showed the captain aiming at unplaced Tamsin for ten
     rounds without landing a blow. Now an unplaced target has no position
     to shield or block it, and the attack proceeds directly. Found during
     reproduction; fixed here as an exploit.
6. **The in-turn reassignment never turns on a bystander.** 11b's
   `reassignPlayerTarget`/`reassignCompanionTarget` used to choose from
   the first party in the room, which could be a shopkeeper.
   - **Now:** they choose from the lost target's own party while it can
     still be assembled. Once that member is gone, they choose from a
     party hostile to the leader (a `hostile` mob, or a group made hostile
     by the fight).
7. **The room text is plain and minimal.** A mob that turns says
   `Garrick Vane turns on the bandit bruiser.` 29c restyles every combat
   line, this one included.
8. **F4: `formation reach` in a fight answers against the enemy.**
   - **Which party:** the one the member is fighting (its target's
     party), or else a party engaged with the company.
   - **Which reach:** the member's own (weapon or innate), not plain
     melee.
   - **Output:** `In this fight, Aria can reach: bandit cutthroat,
     bandit bruiser.`, or `In this fight, Aria can't reach any of the
     enemy right now.`
   - **Outside a fight:** the demonstration is kept, and now says so:
     `Out of a fight, this shows plain-melee reach within your own
     company: ...`.
9. **F3: wiring tests load the shipped config.** The new wiring test runs
   `configs.ReloadConfig()` from the repo root, as the walking and
   exposure modules' tests already do, and asserts level-1 companions
   have more than 1 HP.
10. **F5 and F6 need no code.** F5 (mention the dark) moves to 29c's
   opener, and F6 was a harness artifact.

## Module

- **`internal/hooks`:**
  - a new `combat_engagement.go` holds the upkeep pass (`upkeepEngagements`),
    called at the top of `DoCombat`;
  - pure selection helpers, testable without a live world, where the
    logic allows;
  - `reassignEnemyTarget`/`firstHostilePartyInRoom` are replaced by
    `reassignWithinLostParty` (decision 6);
  - `resolveAttackTarget` fails open for an unplaced target (decision 5).
- **A new `internal/enemyparty` package:** the live-room adapters
  (`Parties`, `PartyOf`, `Alive`) move out of `hooks` so that
  `modules/company` can answer `formation reach` without importing
  `hooks`. `mobparty` stays GoMud-free.
- **`modules/company/formation.go`:** `reach` in and out of a fight.

No persistence changes: `Aggro` and parties are never persisted, and
neither is anything this phase adds.

## Invariants

- **The clock:** the upkeep never advances the world clock or round
  count. It only runs inside `DoCombat` on the game loop.
- **Restart and copyover:** nothing new is stored. After a restart, a
  fight resumes from live `Aggro`, exactly as before.
- **Locks:** the pass reads company state through the existing
  `company.*` provider seams, as the gates already do each attack. It
  takes no new locks and holds none across calls.

## Acceptance criteria

- **F1 regression through the real round:** a leader in column 3 who
  attacks a party member in column 1 swings at a legal party member in
  round 1, and is told about the turn.
- **Kill regression:** when a companion kills the leader's target, the
  leader and the killer both take the next legal target, and the fight
  continues until the party is dead.
- **F2 regression:** attacking one member of a non-hostile party brings
  the whole party into the fight, and no party member is left standing
  idle next to the company when the fight ends.
- **Enemy side:** an enemy whose company target is out of reach turns on
  a legal one.
- **Unplaced companion:** it is reassigned when its target dies.
- **`formation reach`:** in a fight, it names reachable enemies with the
  member's own reach; outside one, it labels the demonstration.
- **F3:** the wiring test loads the shipped config and asserts level-1
  companions have more than 1 HP.
- **Unit tests** for the pure selection and engagement helpers.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- **All combat text:** 29c.
- **The event stream:** 29b.
- **Enemy targeting personalities:** 30c.
