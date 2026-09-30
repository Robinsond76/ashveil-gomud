# Combat Presentation and Tactics — Current Roadmap

Updated 2026-09-30. The owner approved the overall direction on 2026-09-26;
current implementation status is in [Project Status](../PROJECT_STATUS.md).

## Remaining phases

| Phase | Scope | Design status |
|---|---|---|
| 30e | Enemy morale, surrender, mercy decisions, alignment/loyalty reactions, company nerve | [Detailed design](2026-09-30-phase-30e-morale-mercy-design.md) under review |
| 30f | Ambush/surprise, cluster attacks, leaping/flanking, narrow ground, fatigue and cold | [Proposal](2026-09-26-battlefield-conditions-design.md); needs detailed design |

Work on 30e next, then 30f. Mounted combat was removed from 30f by the owner
on 2026-09-30. No replacement mounted-combat phase is scheduled.

## Shipped foundations

Phases 29a–29f provide combat fixes, the event stream and summary, narration,
pronouns and stable enemy labels, pain reactions, and paced output.
Phases 30a–30d2 add statuses, wounds, company tactics, guardians, enemy
casters, chant interruption, shield counters, and physical wind-ups.
Phase 32d supplies automatic strategies; 32g2 supplies the live battle view.
Phase 31 was dropped by the owner on 2026-09-29.

## Decisions still in force

- Combat stays round-based. The continuous Readiness/Wind-up timeline and
  sub-round beats from the external reference were declined.
- Combat resolves every second 4-second game round by default. Other
  systems retain their game-round cadence; balance consequences remain in
  Project Status. Combat output is paced, with fast/normal/slow/off choices.
- Narration is dark and physical, without exclamation marks, ALL-CAPS, or
  `***`. Every damaging hit gives its damage in parentheses. Companions
  have names without charmed tags; enemy ordinals remain stable per battle.
- Strategies drive combat automatically. Phase 30c1 permits one company
  focus change per round during battle; broader manual orders remain tabled.
- Clerics heal during combat; `heal wounds` coordinates post-battle care.
  Full inn rest heals wounds; camp treatment consumes available supplies
  under the shipped 30b rules.
- Weapon crits can inflict secondary effects. Spell crits and new core
  stats (DEX/AGI or a physical/magic defense split) were not adopted.
- Shield bashes are counter strikes only; they break neither chants nor
  wind-ups. Chants use the shipped chance rule; physical wind-ups require
  heavy force. Earlier interrupt-pressure proposals are superseded.
- Enemies have temperaments; undead never yield and flagged bosses never
  break. Yielding stops combat participation immediately; mercy is decided
  afterwards. Spared enemies do not return as recurring characters.

## Implementation expectations

Follow [the project workflow](../AGENT_IMPLEMENTATION_WORKFLOW.md): inspect
current code, approve the phase design, plan, implement with real integration
coverage and player help, independently review, and verify. The event stream
reports authoritative state; it must never become the gameplay state owner.
