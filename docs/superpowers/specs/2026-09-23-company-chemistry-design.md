# Company Chemistry

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Goal and existing seams

Long-serving members should fight better together. The company registry has
stable companion IDs and a five-character cap. Phase 11 already resolves
formation legality and attack targets in combat. Chemistry should modify a
legal attack after those decisions, never make an illegal target reachable.

## Bond model

- Store one bond per unordered pair of stable company member keys (leader or
  companion ID) under the leader's company record. A bond records accumulated
  eligible shared rounds and its last charged global round. Ten pairs is the
  maximum for a five-character company.
- A round counts only when the leader is signed in and both members are alive,
  spawned/attached, in the same company, and together in the same room or
  travel/camp session. Offline time, absence, death, and separation pause
  accrual. Merely standing together while signed in counts; combat is not
  required. Count a round once even if several event handlers inspect it.
- The leader starts a bond with each companion on recruitment. Companion
  pairs start when both are in the roster and present. Dismissal or permanent
  loss ends bonds involving that companion; a new recruit gets a new stable
  ID and starts fresh. Resurrection of the same companion ID preserves its
  accumulated bonds and resumes accrual after restoration.
- Thresholds are 1, 3, and 7 game-day equivalents of eligible rounds. With
  the current 900-round day, these are 900, 2700, and 6300 shared rounds.
  Store the accumulated rounds, not a calendar date, so sign-off and time
  display changes do not award free chemistry. Thresholds are config.

## Combat effect and display

- For each living member in a combat, take that member's **highest** bond
  tier with another living, present company member. Tiers grant +2%, +4%,
  or +6 percentage points of hit chance, capped at +6 points and applied
  to the ordinary hit probability before the attack roll and its normal
  bounds. Multiple bonds do not stack. The leader therefore receives a
  bonus only when at least one present companion has served long enough with
  them, as requested. A lone member receives no bonus.
- `company chemistry` shows each member's current tier, strongest partner,
  and progress to the next tier without exposing a false calendar deadline.
  Combat and `status bonuses` identify chemistry separately from buffs.
  The browser Company panel uses the same read model.
- The bond bonus does not affect experience, damage, auto-target selection,
  reach, or interception. It applies in all directions where a company member
  makes a legal attack, including a companion's auto-assigned target.

## Durability and acceptance criteria

Accrual is checkpointed from the global round counter and flushed on logout,
save, and copyover. Restart may replay a checkpoint, but the last-charged
round prevents double credit. Failed persistence retains the last durable
value and reports the failure rather than displaying a tier that cannot
survive restart.

- Two companions and the leader gain independent bonds; only eligible shared
  rounds advance each pair.
- Rotating, dismissing, killing, or logging out pauses or ends exactly the
  relevant bonds. Revival of the same companion resumes its old bonds.
- The leader has no chemistry benefit alone, and several veteran partners
  never exceed the cap.
- Combat integration verifies the hit modifier on real player and companion
  attack paths, while legality and automatic targeting remain unchanged.
