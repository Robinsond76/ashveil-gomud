# Phase 30g6 — combat balance tuning

Work in progress toward the [owner-approved 30g design](../designs/2026-09-30-phase-30g-tempo-defense-design.md), decisions 8, 11–14 and slice 30g6. Based on master `3bdcb5fb`, on `phase-30g6-tuning`.

## Invariants and scope

Tune live configurable HP, damage, tempo and healing through the real company combat round. Preserve five-level stat steps, class HP ordering, no level cap, saved investment and vitals, the two-turn cap, and shared world time. Report level 100 without applying the duration target there. Do not invent a new equipment catalog: measure existing starter kits explicitly and document that limitation. Preserve independent role-group measurements.

Natural health/mana regeneration is already disabled in battles by 32d/33i2. Combat statuses already use the combat pass from 30a. Verify these integration points and explicitly exclude combat-round buffs from the game-round trigger path; ordinary real-time buffs retain their documented timer.

## Tasks

- [x] Extend the harness through levels 30, 60 and 100 and level 15/10 and 30/10 mismatches; inspect spread and focus setup and assert fixture parity.
- [ ] Measure provisional values, calibrate configurable defaults, and assert the approved spread/focus/mismatch targets with adequately sized cells.
- [x] Verify natural regeneration exclusion and game-round/combat-round buff separation through actual hooks.
- [x] Update affected indexed help and tutorial hints, with rendering and pointer checks.
- [x] Record candidate numbers, distributions, before/after HP and limitations in a verification record; retain failed acceptance results.
- [x] Independently review the complete candidate, resolve findings with regressions, run generation, validation and full race checks, update project status and commit on the feature branch.

## Current candidate and acceptance gap

Provisional defaults: damage bonus 2–20, 1.75 damage per effective Strength;
neutral tempo Speed 2; HP base 12, Vitality factor 0.5, class gains through
level 5, then 0.7 per level. Warrior/cleric/ranger/rogue/wizard rates are
3/2.5/2.5/2/1.5, default/unknown 1.5. Healing spells retain their shipped
potency, mana costs and two-round waits. These are **not release-ready**
until the entire opt-in acceptance suite passes.

The final suite asserts all approved outcomes, including focused fight
length and health removed by enemy targeting. Initial corrected samples
meet spread duration and HP scaling, but focus still takes longer at several
levels. Lower default HP, higher minimum hit chance and stronger/faster
healing experiments did not jointly satisfy the criteria. Discarded
experiments are not shipped. Continue tuning against the strict assertions;
do not redefine a faster victory as first kill, ignore losing fights, or
remove unfavorable levels.

Harness repairs load and verify every shipped class rate, pair distinct
opening aims for passive controls, choose the actual weakest focus opening,
and keep passive/personality cells at rabble coordination. Equipment remains
starter gear at every level. Stat training is balanced on both sides;
companions keep real automatic abilities and healer spells, but enemies in
the passive/personality cells have no matching abilities or healer. Separate
coordination cells retain enemy roles. Health removed excludes overkill and
accounts for healing by comparing starting and final live health.

Integration fixes prevent two warriors from duplicating an already successful
queued tackle, and preserve earned physical turns when an earlier fighter
kills their target. Both use the existing battle/targeting rules. The latter
also retains a killer's second earned turn; it changes one target-change
narration event by one round, with all captured attacks/casts/rewards unchanged.

See [verification](2026-10-03-phase-30g6-verification.md) for checks and measurements.
