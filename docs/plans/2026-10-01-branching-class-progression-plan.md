# Branching class progression — implementation plan

Design: [branching class progression](../designs/2026-10-01-branching-class-progression-design.md).
Status: planning only. Settle design choices and schedule before gameplay work.
Use an isolated feature worktree and read owning package/module instructions.

## 1. Review and settle progression

- [ ] Confirm tier levels, alignment gates, branch commitment, promotion location
  and no-cost default; finalize names for all five lineage trees.
- [ ] Reinspect current shipped 30g4/33h state; agree ownership of HP, training,
  progression UI and migration changes with those plans.
- [ ] Audit exact archetype-ID consumers, skill retirement, automatic abilities,
  schools/spells, specialist lineage, kits, recruiters, death and resurrection.
- [ ] Audit alignment gains/drift and player choice opportunities; plan missing
  moral quest choices with exactly-once rewards if necessary.

## 2. Class graph and effective capabilities

- [ ] Add versioned class definitions: lineage, parents, tier, level/alignment
  gates, roles, inherited capabilities and signature references.
- [ ] Validate unique IDs, existing parents, acyclic paths, legal lineage/tier
  transitions, alignment bounds and valid ability/spell references.
- [ ] Provide one effective-class resolver for player/companion consumers;
  retain base capabilities and specialist eligibility deliberately.
- [ ] Test graph boundaries and effective unlocks without importing modules
  into engine packages or reviving retired skills.

## 3. Durable records and promotion commands

- [ ] Add leader/companion class state with deterministic old-save migration.
- [ ] Implement paths/status and promote preview/confirm via existing member
  selectors, owner checks and authoritative game-loop state.
- [ ] Commit promotion atomically, roll back on save failure, and deduplicate
  repeat confirmations. Never grant a starter kit, level, refill or points.
- [ ] Drive real command tests for thresholds, drift, absent/dead members,
  combat/travel/rest, high-level unpromoted characters and alternate lineage.
- [ ] Verify save/load/copyover, unknown class recovery and death/regain retention.

## 4. Warrior pilot

- [ ] Specify Knight/Mercenary/Reaver signatures and Paladin/Warlord/Dread Knight
  continuations with exact triggers, action costs, equipment and resources.
- [ ] Implement advanced signatures and automatic strategy integration first;
  keep elite continuations visible as planned until their mechanics ship.
- [ ] Preserve warrior base ability and Field Smith lineage, existing guardians,
  shield/parry rules, wound limits, cooldowns and battle pacing.
- [ ] Test player/companion combat and resource maxima before/after promotion;
  measure pilot roles in matched and mixed-company balance scenarios.

## 5. All advanced lineages

- [ ] Finalize and implement distinct Rogue, Ranger, Cleric and Wizard advanced
  signatures, spells and role priorities from the design table.
- [ ] Integrate optional poison/morale/reaction dependencies explicitly; do not
  advertise unimplemented systems as usable class benefits.
- [ ] Verify inheritance and equipment fallback, healer coordination, spell
  access, training and specialist consumers for every branch.
- [ ] Add branch integration tests and balance measurements; ensure unrestricted
  paths remain competitive and moral paths are not automatic strict upgrades.

## 6. Elite continuations and progression compatibility

- [ ] Implement each level-30 signature upgrade and any approved growth modifiers.
- [ ] Reconcile stat steps/HP/XP knee and future companion training with 30g4/33h;
  preserve spent points and derive modifiers exactly once.
- [ ] Exercise promotion at high level, level-loss/regain, resummon and repeat
  load; prevent branch-ability leakage and duplicated grants.
- [ ] Evaluate levels 10/30/60 and mixed-tier parties against combat-cadence goals.

## 7. Expanded companion catalogue and future creature slices

Expansion: [classes and creature recruits](../designs/2026-10-01-expanded-companion-classes-design.md).

- [ ] Recheck Ogre Battle 64/Unicorn Overlord reference material when accessible;
  finalize Ashveil class names/roles without claiming an exact borrowed roster.
- [ ] Evaluate the [supplied reference extractions](../designs/2026-10-01-class-reference-extractions.md):
  Lancer, Grappler/Monk, Vanguard, Arbalist, Dragonkeeper, Enchanter and Valkyrie;
  track distinct roles and dependencies before adding selectable classes.
- [ ] Define crossbow/shield, unarmed, temporary imbuement and handler contracts;
  defer commander overlays, prestige, forms, flying/aquatic units until designed.
- [ ] Expand creature candidates and authored affinity/form paths; prefer Hellhound
  to Cerberus and bounded dragon branches, preserving equipment/action/roster rules.
- [ ] Deliver additional humanoid paths in role bundles; prioritize Sorcerer,
  define exact signatures and expose only completed classes as selectable.
- [ ] Add nonhuman humanoid recruitment with species separate from lineage,
  bounded racial traits, gear checks and individual alignment.
- [ ] Design creature records, recruitment, innate attacks, allowed equipment,
  upkeep, weight, formation, physiology, bond and family-specific recovery.
- [ ] Pilot Hound and Stone Golem through real company lifecycle tests, then
  Hellhound; flight/undead require separate reach/recovery decisions.
- [ ] Verify cap/ownership, death/resurrection, repairs, save/load/copyover and
  no free summons, cargo, equipment, action or immunity loopholes.
- [ ] Ship expanded class/creature help and browser inspection with each bundle;
  measure mixed-company balance and obtain independent review per delivery.

## 8. Player guidance, content and final gate

- [ ] Ship indexed help for classes/promotion/roles and update affected hubs,
  alignment/experience/spell pages and Departure tutorial guidance.
- [ ] Show lineage/current class, readiness and explicit requirements in text,
  company inspection and browser setup through the same backend.
- [ ] Update creation/recruit descriptions and add needed moral choices without
  auto-promoting old characters or relocating their equipment.
- [ ] Test help rendering, tutorial pointers and all wired persistence/consumer
  paths, including save-failure recovery and multiplayer ownership.
- [ ] Run focused tests during work; run required generation, validation and
  full race suite once on final code, then obtain independent full-phase review.
- [ ] Fix confirmed findings with regression tests, record exact verification
  and review in PROJECT_STATUS.md, and integrate the completed branch.

Later retraining/prestige branches require their own approved design; they are
not a hidden prerequisite for delivering these initial promotions.
