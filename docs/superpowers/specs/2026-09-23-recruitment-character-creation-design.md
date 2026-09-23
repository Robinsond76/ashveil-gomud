# Recruitment, Character Creation, and Durable Companions

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Goal and existing seams

Players should begin with a meaningful archetype and fitting equipment, then
recruit distinct companions they can keep and equip. Phase 17 already provides
warrior, rogue, wizard, cleric, and ranger archetypes with one-time
`archetype choose <name> confirm`. The company registry currently persists a
companion's ID, mob template ID, archetype, and formation, while
`company summon` only allows the training dummy. Live mob instances and their
gear are reconstructed from the template on login. This spec establishes the
durable member state needed by later sharpening and resurrection work.

## Player flow

1. After account creation and before the Ashveil tutorial, offer the five
   archetypes with their role, starting skills, and kit preview. Confirming
   uses the existing archetype choice and grant rules. Reconnecting during
   creation resumes the choice. Existing characters retain their current
   choice and gear; an unchosen existing character can choose normally.
2. Grant one kit after the archetype choice has committed. Each kit has a
   balanced value and usable equipment, selected from valid shipped item IDs
   in a data table. Proposed roles: warrior blade and shield, rogue light
   blade and lock tools, wizard staff and travel supplies, cleric defensive
   weapon and supplies, ranger ranged weapon and travel supplies. The plan
   verifies ammunition and spell prerequisites before finalizing item IDs.
3. A settlement recruiter offers authored companion candidates with visible
   archetype, level, alignment once Phase 21 exists, equipment, and price.
   `company recruit <candidate>` adds the chosen candidate to the existing
   maximum of four companions; `company status` and `formation` remain the
   management entry points. At least two distinct candidates must ship for
   the tutorial. Each of the two tutorial candidates is free and claimable
   once per character, with no repeatable gear or gold reward; ordinary
   candidates use a configured gold price. Recruitment does not depend on
   Phase 19 market pricing.

## Durable model and boundaries

- Extend the company record for each stable companion ID with level,
  experience/progression required to restore that level, equipped and carried
  item snapshots, and lifecycle state. Template ID remains the source of base
  mob behavior; runtime instance IDs remain transient.
- On first migration from an old record, initialize missing progression and
  equipment once from the mob template, then save the upgraded record before
  restoration. Subsequent logins restore the saved member state rather than
  minting new template equipment. A failed save retains the old record for
  retry and must not issue a second kit or recruit.
- Persist changes to companion level or equipment at their owning event
  seams, including death, resurrection, equip/unequip, logout, restart, and
  copyover. A companion's stable ID is never reused after dismissal or loss.
- Keep recruitment authorization in `modules/company` and pure roster rules
  in `internal/company`. Creation may call the existing archetype provider;
  core login code must not import a gameplay module directly. Item grants
  have a durable per-character claim marker and recover from an interrupted
  grant without duplicate items.

## Acceptance criteria

- A new player can choose each archetype, see its kit, and receive that kit
  exactly once through reconnect, restart, and copyover.
- At least two distinct recruitable companions can join, be placed in the
  3×3 grid, and respect the five-character company cap.
- Companion levels and actual equipped items survive dismissal rules,
  logout/login, restart, and copyover without spawning duplicate equipment.
- Failed payment, full roster, unavailable candidate, and failed persistence
  leave gold and member state consistent and explain the refusal.
- Neither tutorial recruit can be claimed again for equipment or resale.
