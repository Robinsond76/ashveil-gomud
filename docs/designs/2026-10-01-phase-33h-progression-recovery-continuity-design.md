# Phase 33h: Company Progression, Rewards, and Expedition Continuity — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Growth, rewards, recovery, and relocation consistently apply to a persistent company. Preserve archetype identity and prevent reconnects or personal teleports from bypassing expedition costs.

## Current mechanics and prior art

`internal/characters/character.go` AutoTrain spends companion stat points evenly across six stats. Combat companion XP is awarded through `internal/mobcommands/companyxp.ashveil.go`; `internal/hooks/Quest_HandleQuestUpdate.go` grants quest XP to the player alone. `modules/company/runtime.go` restores saved gear/level but companion HP/mana refill is a recorded limitation. `internal/usercommands/skill.portal.go` and quest reward relocation call `rooms.MoveToRoom` directly; ordinary following is exit-based. Death/tutorial already use explicit company relocation; 30e adds saved pending-return state.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Deliver three separately reviewed slices: 33h1 growth/rewards, 33h2 durable readiness/recovery, and 33h3 relocation/separation.
- Growth: archetype-weighted companion training with a small specialization choice. Respect 30g4's stat steps, archetype HP, XP knee, death level loss, and peak-level protections; do not re-award spent points.
- Rewards: distinguish personal quests/skill grants from company contracts. Combat and contract eligibility use approved participation; quest completion stays attached to the player's journal unless a contract explicitly changes it.
- Readiness: persist health, mana, relevant wounds/conditions, and necessary recovery metadata so logout/login, re-summon, and copyover do not refill the band for free. Keep 30b's inn/camp/treatment rules.
- Define offline recovery explicitly. No elapsed real-time healing or global time advancement by default; online round-based recovery remains authoritative until approved otherwise.
- Relocation: every portal, scripted/quest teleport, travel completion, death return, tutorial move, and special exit explicitly takes eligible company members, refuses, or creates explained separation.
- Include herd/cargo presence and dead/30e-fled members in relocation policy. Never recall separated members into combat or respawn a dead member as a shortcut.

## State ownership, persistence, and recovery

Growth/readiness belong to company member state with versioned migration; leader state remains with users, wounds with its existing owner, and survival/travel/cargo with their existing providers. Store stable identities and validated values, not live pointers. Normalize saved HP/mana against changed maxima without granting free recovery. Cross-file operations need applied markers and conservative recovery.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

33h1 aligns with 30g4, 33e, and 33f; reward participation depends on 33d. 33h2/3 coordinate 33c and existing death/morale systems. These continuity fixes may ship before new specialization choices.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Growth weights/specializations; training and quest reward migration; contract XP rules; old saves missing vitals; changed-max handling; offline recovery policy; which transports take companies; supported separation duration and return conditions.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Final implementation decisions: 33h1 (2026-10-02)

The owner asked to continue with 33h after 33g's management slice, under
the standing 2026-10-01 authority to take the lead's recommended defaults.
33h2 and 33h3 settle their own decisions in their plans.

- **Derived training, not spent points.** A companion's trained stats are
  always its template's training plus its level's stat points
  (`StatPointsPerLevel`/`StatPointsEveryNLevels`, the same count a spawn
  uses), dealt by its growth weights. They are recomputed on every spawn,
  live level-up, resurrection, and focus change, never accumulated, so a
  level/death/regain loop cannot mint points and nothing new is saved.
  30g4's stat steps change the racial part only and leave this intact.
- **Growth weights (proposed, tunable in the archetype config):** warrior
  Strength 4, Vitality 3, Speed 2, Perception 1; rogue Speed 4,
  Perception 3, Strength 2, Smarts 1; wizard Mysticism 4, Smarts 3,
  Perception 2, Speed 1; cleric Mysticism 3, Vitality 3, Smarts 2,
  Strength 2; ranger Perception 4, Speed 3, Strength 2, Vitality 1. A
  companion without an archetype keeps the even spread. Points are dealt
  in a fixed weighted rotation, so each new point adds to a stat and none
  is ever taken back by a level-up.
- **Specialization:** one optional focus stat per companion
  (`company growth <member> <stat>`, `balanced` to clear) adds weight 2.
  Free to change out of battle; it re-deals the same points at once.
  Stored on the companion record (`growth_focus`, empty by default).
- **Contracts:** a quest whose rewards set `companyexperience` is a
  contract. When the leader turns it in, every companion alive, attached,
  not withdrawn, and in the leader's room earns that figure in full, as
  combat experience does (32e). Absent, dead, or fled members earn none.
  Quest experience, gold, items, buffs, and skill grants stay personal.
  Shipped contracts: Rodric's Rats (1000) and The King's Shadow (15000).
- **Migration:** none needed. Old companions take their archetype's
  weights at their next spawn; old quests without the field are personal.

## Final implementation decisions: 33h2 (2026-10-02)

The lead explained the defaults below in plain language and the owner
approved them on 2026-10-02 ("All this looks good carry on").

- **Durable vitals.** A companion's health and mana join its saved state
  (`MemberState.Vitals`), captured by the same 22b snapshot seams as gear
  and wounds (autosave, shutdown, copyover, the leader's logout, a stale
  mob's replacement). Every respawn (login, copyover, crash recovery,
  re-attachment) puts them back. A crash restores the last saved values,
  the same accepted window gear already has.
- **Old saves:** a record with no vitals spawns full (to its wound limit)
  once, as it would have before, and is tracked from its next snapshot.
- **Changed maxima:** saved values are absolute points, clamped to the
  new wound limit and mana maximum, and a living companion never spawns
  below 1 health. Never scaled by a percentage. A companion's live level-up
  keeps its health and mana (GoMud's level-up refill no longer applies to
  companions; players are unchanged).
- **Online recovery:** out of a battle and with the leader online, a
  living companion regains health on the players' beat (every third round,
  `HealthPerRound`), stopping at its wound limit, beside 32d's mana.
  Exposure's lethal damage now outpaces a companion's regeneration too.
- **No offline recovery:** elapsed real time restores nothing, and no
  world time advances.
- **Inn stay:** when a finished stay's Well Rested is granted, after its
  wounds knit, the leader and every live companion with them are restored
  to their wound limit and full mana. Camp rests are unchanged.
- **Resurrection:** a resurrected companion returns at half its health
  limit (at least 1) and half its mana, matching a player's church waking.
  Stored as a percentage until the first spawn resolves it, so a crash in
  between cannot refill it.
- **Unchanged:** light wounds and battle-only effects still end on a
  respawn; morale flight's return keeps its own saved health and mana.

## Acceptance criteria and verification

Real combat/quest rewards cover personal versus contract cases and absent members; level/death/regain loops do not mint points; logout/login, re-summon, crash/save failure, and copyover preserve readiness; changed maxima/missing old-save fields migrate safely; portals, quest moves, travel, death, and tutorial relocation preserve exact member/item/cargo ownership and pending returns.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help experience`, leveling/archetype/quest pages, `help company`, `help rest`, `help camp`, `help portal`, travel/death/return guidance; add contract and readiness explanations. Tutorial preparation/Departure points to continuity rules.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
