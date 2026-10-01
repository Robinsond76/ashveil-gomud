# Phase 30e: Morale and Mercy — Design for Review

Status: owner approved, 2026-09-30. The owner approved the two-slice delivery,
probabilities, thresholds, prompt lifecycle, alignment/loyalty consequences,
temporary companion flight, and restart policy in this design.

## Goal and scope

A battle can end because enemies lose their nerve. Surrender removes an enemy
from danger immediately, then lets the player decide its fate after the fight.
Companions react according to their values. Their own nerve occasionally
matters in a losing fight without turning routine battles into loss of control.

Preserve automatic strategies, stable enemy names, formation rules, paced
narration, and shared multiplayer time. Do not add manual combat abilities,
a recurring spared-enemy system, or a new combat clock.

## Recommended delivery

Implement two independently verified slices under Phase 30e:

1. **30e1 — enemy morale and mercy:** enemy break checks, yield/flee state,
   post-battle choices, consequences, help, and battle-view reporting.
2. **30e2 — company nerve:** companion hesitation and temporary flight,
   rejoining, durable loyalty consequences, help, and reporting.

This keeps enemy reward/prompt handling separate from companion persistence.
The alternatives are one large release (less intermediate testing isolation)
or only enemy morale (simpler, but leaves the approved company-nerve scope
unfinished). Both recommended slices remain part of 30e.

## Prior art and ownership

Inspected at `792455ea`:

- `internal/battle/battle.go` owns runtime battles, names, and membership.
  Restart/copyover creates fresh battles; it does not restore fight IDs.
- `internal/hooks/NewRound_DoCombat.go` runs battle selection, engagement
  upkeep, statuses, strategies, attacks, affected actors, and settlement.
- `internal/hooks/combat_battle.go` selects the next waiting group immediately
  when a battle ends. Mercy must not pause that queue or the shared world.
- `internal/hooks/combat_stream.go` currently counts every living enemy in
  the room as standing. A yielded enemy needs an explicit exclusion.
- `internal/combatstream` already declares `Yield` and `Mercy`, but its summary
  does not yet fold a yielded ending. It reports state; it must not own it.
- `internal/prompt` and `UserRecord.StartPrompt` support command-backed
  questions. A mercy prompt must coexist with an existing prompt and have
  a validated actor/fight token; typed names alone are not authorization.
- `internal/mobcommands/suicide.go` owns ordinary death rewards and a `vanish`
  route without rewards. Ordinary kills also change alignment: execution
  needs an explicit reward mode to avoid accidentally awarding both the
  ordinary alignment adjustment and the mercy adjustment.
- `modules/company/alignment.go` computes company alignment as the average
  of the leader and living companions. There is no separate alignment score.
  Companion disposition and service/chemistry already have saved state.

Use a small pure `internal/morale` rules package and game-loop adapters in
`internal/hooks`. Keep runtime enemy disposition shared by mob instance across
all battles, rather than allowing two players to surrender the same mob twice.
Company loyalty mutations belong to `modules/company` through a typed provider
seam. No callback may mutate a registry while holding battle/stream locks.

## Enemy morale defaults

Add validated race defaults with optional mob-template overrides. Unknown or
missing temperament preserves today's behavior (`unbreakable`); explicitly
opt in suitable shipped enemies. Practice foes and player-owned/charmed mobs
are excluded from enemy morale. Undead and constructs remain unbreakable;
flagged bosses cannot be overridden into yielding.

| Temperament | Hold | Yield | Flee |
|---|---:|---:|---:|
| Unbreakable | 100% | 0% | 0% |
| Steadfast | 85% | 10% | 5% |
| Wary | 65% | 25% | 10% |
| Craven | 40% | 40% | 20% |
| Skittish beast | 65% | 0% | 35% |

Proposed break triggers: the original enemy leader dies; the first caster or
healer dies; at least half the original group has died; or the last active
enemy falls to 25% health or less. Snapshot the original group and roles when
engaged. A trigger fires once per group, with at most one roll per enemy per
combat round. Simultaneous triggers combine into one check; no repeated rolls
just because a threshold remains true. Yield/flee does not recursively trigger
another check in the same round. A held low-health check cannot be farmed by
healing and damaging the same enemy repeatedly.

Observe resolved deaths/damage and check morale at a consistent end-of-round
point after affected actors resolve and before battle settlement. A surrender
there takes effect immediately; the foe has no further action or damage that
round. Stable iteration and injected random rolls make tests deterministic.

## Yielding and flight

- Yield clears aggro, queued hostile actions, casts and wind-ups, and active
  combat statuses that could continue dealing damage. Do not refund enemy
  mana. The enemy remains visible, marked surrendered, outside the formation.
- A yielded enemy neither attacks nor takes damage from attacks, pets, spells,
  counters, status ticks, or queued scripts. Validate this at shared targeting
  and damage seams as well as strategy selection. Retarget old aims safely.
- Exclude yielded foes from formation protection, guards, active-enemy counts,
  engagement upkeep, group attack selection, and automatic reacquisition.
- A flight outcome removes the enemy from this encounter without rewards and
  reports `Flee`. Recommend narrated departure and removal, avoiding adjacent
  room pursuit/re-engagement in this slice. Ordinary template respawn remains
  allowed; it is a new instance, not a recurring spared character.
- For a mob shared by several battles, the first battle that registered it
  owns any mercy decision (ties resolved by stable player ID). Yield applies
  globally. Do not offer multiple prompts or duplicate rewards. If the owner
  loses or leaves, the surrendered mob escapes for everyone.

## Mercy flow and rewards

At victory, collect the owned yielded enemies in stable enemy order. After
paced closing/summary output drains, ask one question per enemy:
`The first skirmisher kneels, hands raised. Spare him? [yes/no]`.

- **Yes:** narrate departure without the weapon, remove the enemy and all its
  equipment without loot/XP. Do not leave a dropped weapon that contradicts
  the no-loot rule.
- **No:** narrate execution, resolve ordinary kill XP and loot exactly once
  through an explicit death/reward mode, and apply the mercy alignment effect
  instead of the ordinary kill-alignment effect. Keep existing XP eligibility
  and company sharing; do not fabricate damage credit for an uninvolved player.
- **No answer:** 30 seconds per displayed question, then that enemy escapes.
  Use the existing game loop and a real-time deadline, not sleeps/goroutines.
  Invalid input repeats the question without extending the deadline.
- Leaving the room, logout, death, character deletion, tutorial handoff,
  restart/copyover, or a newly starting battle releases all unresolved enemies
  without rewards or alignment changes. Existing waiting battles proceed on
  schedule; they cannot be stalled by ignoring a mercy question.
- Do not replace another active prompt. Release the yielded enemies with a
  short explanation if a mercy interaction cannot begin safely.
- Each decision consumes its pending token before applying effects. Replayed
  input, stale questions, duplicate death callbacks, and a second player's
  answer are no-ops. World mutation remains on the game loop.

The battle summary records `yielded` at fight end. Mercy messages/events follow
it; do not keep the fight artificially open while waiting for input or attach
an outcome to a later fight. The web battle view labels surrendered enemies
and removes their target lines; after battle, the standard text prompt works
in Telnet and the web client. No new graphical decision dialog is needed.

## Alignment and companion reactions

Proposed defaults, clamped to existing ranges:

- Spare: leader alignment +5. Execute: leader alignment −5. The displayed
  company average is recomputed; it is not assigned an independent ±5.
- Each living companion who witnessed the victory reacts once per decision.
  Alignment at least +25 approves mercy; at most −25 approves execution.
  Approval gives +2 loyalty, disapproval −1, neutral companions 0.
- Use pre-decision companion alignment to choose one short reaction line.
  Do not instantly rewrite companions' beliefs; existing alignment drift stays.
- Zero loyalty uses the existing desertion rules. No simultaneous mutation of
  the roster while iterating it; collect effects, persist, then apply departures.

## Company nerve (30e2)

Evaluate once per companion per battle when the company first clearly loses:
at least half its starting members are incapacitated, or its surviving members'
combined health is at most 25% of their combined maximum. Use a snapshot before
rolling; one flight must not recursively trigger additional rolls.

- Eligible: loyalty below 25, or the present band's chemistry below Familiar.
- Loyalty below 25: 20% hesitate, 5% flee, 75% hold.
- Weak chemistry alone: 10% hesitate, 0% flee, 90% hold.
- Hesitation costs at most one action; it cannot stack an extra skipped round
  onto an already action-blocked status. Cancelling a committed cast follows
  the existing interruption/refund rule; no spell can complete after flight.
- Flight removes that companion from combat, not from the durable roster.
  It rejoins the living leader after settlement, including a broken-off fight,
  with −5 loyalty. At zero loyalty, existing desertion applies instead.
- Preserve gear, wounds, experience, and pending death state. A companion that
  dies before the flight transition follows death/resurrection rules, not a
  return that silently revives it. The leader never makes a nerve roll.

## Save and recovery boundaries

Enemy fight state and unanswered mercy tokens are transient, matching existing
battles. A restart/copyover abandons unanswered decisions; it must never restore
an old instance ID, immunity flag, prompt, or claim onto a new spawn. This is an
explicit outcome, not a lost reward awaiting replay.

Accepted alignment, loyalty, XP, gear, and companion return state use existing
owner save paths. Company flight needs a saved pending-return marker keyed by
companion ID, so logout/copyover reattaches the member without losing gear or
charging loyalty twice. Resolve it before snapshotting/removing runtime mobs.
Save failures must be surfaced and leave retryable owner state; do not report
successful consequences before required saves succeed.

Ordinary kill rewards are not an atomic transaction across user, company, and
world files. This phase must not claim crash-atomic rewards or invent a replay
journal that can duplicate loot. Reuse the existing reward/save guarantees,
add graceful restart/copyover tests and injected save-failure tests, and document
any remaining abrupt-crash limitation. A general reward transaction redesign
would require separate scope approval.

## Acceptance and verification

- Pure seeded tests cover every temperament, invalid config, exclusions,
  threshold boundaries, simultaneous triggers, and one roll per trigger.
- Real combat-round tests kill a leader/healer, cross half-loss and low-health
  thresholds, and prove unbreakable enemies continue unchanged.
- Real weapon, spell, pet, status, counter, queued-action, engagement, and
  formation paths cannot damage or reactivate a surrendered foe.
- A battle with only yielded enemies ends; shared enemies have one owner;
  subsequent queued battles proceed without waiting for mercy input.
- Command/prompt tests cover both choices, invalid/repeated input, expiry,
  existing prompts, movement/logout/death/purge/handoff, and paced output order.
- Real execution and escape paths verify XP, loot, alignment, and loyalty
  exactly once within the live process; save/reload verifies durable effects.
- Summary events and GMCP/web rendering distinguish yielded, fled, and dead
  actors without leaking enemy numeric health or names hidden by darkness.
- Nerve tests cover skipped actions, casts, flight/rejoin, zero-loyalty
  desertion, death races, saved equipment, restart/copyover, and failed saves.
- Ship `help morale`, `help mercy`, updated `help combat`, `help alignment`,
  and `help chemistry`, with keyword aliases, tutorial pointers, render tests,
  and `TestTutorialHelpPointersExist`.
- Each slice gets independent review and the final project checks described
  in [the workflow](../AGENT_IMPLEMENTATION_WORKFLOW.md). No world-time advance.

## Approval

The owner approved the detailed gameplay choices on 2026-09-30 ("Looks good approved").
