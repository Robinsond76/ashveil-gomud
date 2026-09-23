# Company Life and Onboarding — Spec Roadmap

**Status:** Design direction approved by the project owner on 2026-09-23. These
linked documents are feature specifications for review, not implementation
plans. The existing Phase 18b–21 roadmap remains in effect.

## Intended player experience

A new player chooses an archetype, receives a fitting starter kit, recruits a
small company, and learns to prepare and command it. Rest, weapon preparation,
long-term bonds, and death have visible consequences. Text commands, the browser
client, and the tutorial describe the same live state. Shared world time never
advances for an individual's travel, rest, or death.

## Specs and order

| Order | Spec | Main dependency |
|---|---|---|
| 1 | [Recruitment and creation](2026-09-23-recruitment-character-creation-design.md) | Phase 17 archetypes; durable companion level and gear |
| 2 | [Rest and weapon preparation](2026-09-23-rest-weapon-preparation-design.md) | Phase 7 camping, Phase 16 inns, durable companion gear |
| 3 | [Company chemistry](2026-09-23-company-chemistry-design.md) | Stable company member IDs and Phase 11 combat seam |
| 4 | [Death and resurrection](2026-09-23-death-resurrection-design.md) | Durable companion levels and settlement service content |
| 5 | [Player information surfaces](2026-09-23-player-information-surfaces-design.md) | Read models from the new mechanics |
| 6 | [Ashveil tutorial](2026-09-23-ashveil-tutorial-design.md) | All above; Phase 21 alignment before its lesson ships |

Implementation plans should be scoped and reviewed separately for each spec.
The tutorial may receive lessons in stages, but its alignment lesson must use
the implemented Phase 21 rules. Phase 20 trade rumours are unrelated to this
packet and need not block it.

## Decisions carried forward

- One whetstone sharpens all eligible bladed weapons currently equipped by the
  living company during one camp rest, then is consumed. An automatic setting
  can perform this without a separate command each rest.
- Player death immediately costs one level and returns the player and all
  surviving companions to the church in the last visited eligible city.
- Dead companions can be resurrected at a city church or village shaman within
  three game-day equivalents of the leader's **online** time. The allowance
  pauses on sign-off; the shared game clock does not pause.
- All durable member identity, level, gear, bonds, death records, rest effects,
  and timers survive restart and copyover. Stable companion IDs are never
  reused.

## Integration and review gate

Each implementation slice needs integration coverage at the command and event
seams it changes, restart/copyover recovery coverage, and an independent full
diff review under `CLAUDE.md`. The owner-facing review should compare Telnet
and browser output for the same company. `docs/PROJECT_STATUS.md` records each
completed slice and review outcome.
