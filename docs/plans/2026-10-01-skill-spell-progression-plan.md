# Skill and spell progression implementation plan

Design: [skill, utility and spell review](../designs/2026-10-01-skill-spell-progression-design.md).
Status: documentation only; settle proposed milestones/choices and schedule before
new gameplay. Preserve prior approved 33f decisions and use an isolated worktree.

## 1. Complete source-to-player audit

- [ ] Trace every shipped skill rank and spell through its real command, trainer,
  class gate, grant, script, target scope, automatic strategy and help page.
- [ ] Publish a matrix of existing, planned, obsolete, internal-only and blocked
  abilities; record stale descriptions and ranks with no distinct benefit.
- [ ] Reconcile Brawling/manual combat, Protection/Aid/permanent death, Track/Search
  replacements, retired commands, dual-wield ranks and trading/auction scope.
- [ ] Audit actual alignment/class/progression state against latest shipped code;
  do not use historical design headers as implementation status.

## 2. Agree unified unlock model

- [ ] Settle level 1/5/10/20/30/40/50 milestones and late modifier choices, preserving
  trained player utility ranks and approved companion 1/10/20/30 thresholds.
- [ ] Define capability records and effective-class resolution, inheritance,
  owned versus usable spells, equipment requirements and current-level gating.
- [ ] Decide whether Alchemy becomes one trade skill and what caster profile
  controls are needed; keep no-extra-currency and automatic-combat defaults.
- [ ] Resolve curative consumable proposals against the owner-approved no-battle-
  items rule, using automatic spells unless an explicit exception is approved.

## 3. Data, state and migration

- [ ] Author validated unlock/ability/spell specifications with explicit contexts,
  scopes, costs, cooldowns, chants, counters and AI priority/fallback.
- [ ] Store only choices/ownership that cannot be derived; reconcile old saves,
  paid points, retired refunds and exactly-once grants without silent respec.
- [ ] Test save failures, duplicate milestone events, death/regain, resummon and
  copyover; ensure no restored mana, costs or cooldown bypass.

## 4. Field and camp progression

- [ ] Deliver approved 33f2/33f3 capabilities through their own plans, reusing
  best-member attribution, toggles and no-stacking rather than competing roles.
- [ ] Add approved recipe progression for Cooking/Alchemy with kits, supplies,
  deterministic yields and camp/field restrictions.
- [ ] Add creature care/repair only after family-specific lifecycle exists.
- [ ] Verify actual commands/steps/rest callbacks, absent/dead members and resource
  transactions; do not restore solo stealth, medic, quartermaster or charm.

## 5. Offensive abilities and spells

- [ ] Keep existing automatic baseline intact; implement a small level/class
  bundle first, prioritizing distinct signatures and Sorcerer repertoire.
- [ ] Wire player and companion execution through normal action/reaction budgets,
  target/reach legality, cooldowns, mana, chants, statuses and healing reservations.
- [ ] Expand class bundles once dependencies work; no selectable placeholder
  abilities, guaranteed control chains or free spell-plus-attack turns.
- [ ] Test real combat paths and class inheritance, including spell ownership/
  school gates, branch leakage, equipment fallback and poison cure behavior.

## 6. Presentation, balance and completion

- [ ] Add unified Field/Camp/Combat unlock views and next milestones; reuse
  specialists/skills/train/spellbook controls and shared browser backend.
- [ ] Correct stale skill descriptions; ship help, keywords, related hubs and
  tutorial pointers together with each changed ability.
- [ ] Test rendered help and tutorial pointers; measure combat cadence, mana,
  healing/control uptime, resource economy and utility usefulness by rank.
- [ ] Run focused checks during work, then required generation/validation/full
  race suite on final code; obtain independent full-phase review.
- [ ] Fix confirmed findings with regressions, record exact checks/review in
  PROJECT_STATUS.md and integrate only completed, reviewed slices.
