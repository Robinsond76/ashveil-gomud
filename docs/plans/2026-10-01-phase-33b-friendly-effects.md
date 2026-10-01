# Phase 33b execution plan

The owner authorizes implementation and delegates defaults. First deliver
company-only correctness; 33d will wire consenting allied leaders through
the resolver seam before enabling cross-company automatic support.

Decisions: helpful single effects retain explicit member targeting; group
help defaults to the caster's company, area help respects consenting company
membership, and harmful spell types keep their existing boundaries. Pets and
temporary charms are not automatic company targets. A temporary/world mob's
company-scope fallback is itself. `heal`, `healall` and `aidskill` allow downed
players above the existing death boundary (-10); fallen companions remain
ineligible. Other effects require living targets by default. Self is included
unless explicitly disabled. Costs and chant duration do not change.

1. Add validated helpful scope/eligibility metadata with backward defaults
   and explicit shipped content mappings.
2. Implement one resolver for manual/automatic/mob casts. Capture the starting
   target set and ownership identities; prune rather than expand at completion.
   Recheck source/target life, origin room, charm identity, stable member key,
   consent/membership and wound limits (existing AddHealth owns those limits).
3. Wire command starts, automatic starts and script completion. Exercise real
   casts, source/target moves, death, dismissal, transfer, downed eligibility,
   no double mana charge and unchanged harmful targeting.
4. Ship indexed help for friendly scopes, cast/spells/company updates,
   tutorial hint and help rendering tests.
5. Aggro is already runtime-only. New cast snapshots are not durable; old spell
   YAML without metadata defaults safely. No user/company save migration.
6. Independent full-diff review, fix findings, full generate/validate/race and
   applicable lint, record exact verification, commit/merge/push, continue.

Completed: all six tasks; independent review findings fixed and rechecked.
Integrated on latest master with 30g3 personal burden preserved.
