# Phase 38c: Elite routes (execution plan)

Design: [elite routes](../designs/2026-10-06-elite-routes-design.md), with
the [faith routes](../designs/2026-10-05-faith-routes-design.md) tables for
the cleric and warrior faith elites. Depends on 38b (promotion at 10,
talents, advanced ranks, summon framework).

Eighteen elites with seven ranks each is about 126 rank effects, so 38c
ships in three slices, each with its own build thread, review gate and PR.

| Slice | Scope | Depends on |
|---|---|---|
| **38c1** | Elite framework, UI and help hub; warrior and cleric elites (Warlord, Paladin, Dread Knight, Hierarch, Elder Druid, Demonologist) | 38b |
| **38c2** | Rogue and ranger elites (Pathfinder, Swordmaster, Nightblade, Sentinel, Marksman, Ravager) | 38c1 |
| **38c3** | Wizard and witch elites (Archon, Archmage, Necromancer and its thrall, Wise One, Coven Mother, Crone of Ash) | 38c1 |

38c2 and 38c3 may build at the same time (different lineages, shared files
only cost merge order). 38d and 39i need only 38c1's framework; 36d and 42
wait for all three slices.

Workflow (project memory): a build thread implements and opens the PR
without its own independent review; an Opus review thread reviews, fixes
and merges, including the UI check. Build threads iterate with targeted
package tests and run `make generate`, `make validate`, `go test -race
./...` and `make js-lint` once before opening the PR. Balance runs are
timeboxed: small samples while tuning, one full run at the end.

## Task 0 (every slice): reconcile with 38b

- Merge master and read 38b's class graph, rank resolver, talent store and
  summon code, and its plan.
- For each elite in the slice, compare the advanced signature the design
  assumes with what 38b shipped. Where they differ, rewrite the elite's
  ranks to grow the shipped signature, keeping its role, and list the change
  in the slice's Project Status entry.

## 38c1 tasks

1. **Graph and gates.** Add the eighteen advanced → elite edges with gates
   (+30, −30 or none) to 38b's graph. Promotion at 30 runs through the 38b
   `class promote` path: level, parent class, gate, safe context, preview,
   confirm with revalidation, idempotent repeat. A base character at 30+
   may take advanced then elite in one visit. A gated character waits.
   Tests: real commands for a player and a companion at the gate
   boundaries (+29/+30, −29/−30), drift between preview and confirm,
   rejection in battle, travel and rest, double confirm, save failure
   rollback, copyover and restart keep the elite class.
2. **Elite ranks.** Extend the derived rank resolver to 30–60 for elite
   classes only; catch-up at a late promotion; a level lost to death drops
   ranks above it. Tests: each rank boundary (29/30, 34/35 … 59/60) for a
   player and a companion; an advanced character at 45 has no elite ranks.
3. **Elite talents.** Add the three elite talents per lineage to 38b's
   talent lists, offered from 35 to elites only; apply their effects at
   their real consumers (armor, Attack, Evasion, crit, meter, mana, chant
   break, healing, hex land chance, Second Wind at turn start). Tests:
   refused below 35 and for non-elites; each effect through its consumer.
4. **Warrior and cleric elites.** Warlord ranks (design §2) and the faith
   elites' ranks 30–60 that 38b didn't ship (Paladin, Dread Knight,
   Hierarch with the Angel's ranks, Elder Druid, Demonologist with the
   Demon's ranks and broken binding). Each signature and capstone through
   a real battle, companions using them by strategy role.
5. **UI.** Level-up readiness and gate-wait lines (player and company
   report); milestone schedule (level 30 delivered, elite ranks at 35–60);
   promotion preview with catch-up ranks and elite talents; `class` rank
   lines; `company` / `company inspect` tier and `promote ready` /
   `waiting: alignment` markers; rank-up lines; battle narration for
   signatures and capstones; GMCP member fields `class`, `tier`, `rank`,
   `promotion`; web company window badge and ready marker, character window
   class, tier and rank. Tests for each text and GMCP field; `make js-lint`.
6. **Help and tutorial.** `help elite` (hub with all eighteen elites),
   `help warlord`, updates to `help promotion`, `help talents`, `help
   classes`, `help warrior`, `help cleric`, the warrior and cleric advanced
   and faith elite pages, `help alignment` and `help combat`; keywords and
   aliases; the promotion lesson's tutorial hint. Render tests and
   `TestTutorialHelpPointersExist`.
7. **Balance (timeboxed).** Harness cells at 35 and 50 with warrior and
   cleric elites: band targets hold; siblings within 5 points; elites beat
   the advanced-only twin by 10–20 points in a hard band at 40. Record the
   result in a measurements note.
8. **Status.** Project Status entry and the phase row.

## 38c2 tasks

1. Task 0 for the six rogue and ranger elites.
2. **Pathfinder** (§3): ambush chance halved, Opening Strike on foes that
   haven't acted, Expose Weakness, scouted-ground meter, Vanish, two
   Opening Strikes, cache chance (37), Ambush Master flipping a real
   ambush.
3. **Swordmaster:** riposte damage and uses, parry, Disarming Riposte,
   Perfect Parry, reach ripostes, Unbroken Guard's limit.
4. **Nightblade:** Death Mark passing on, Envenom, finishing threshold,
   Shadowstep (guardians intercept), Killing Spree, Coup de Grâce (damage
   against bosses).
5. **Sentinel:** Overwatch held turn (spends the turn), knockdown of
   leaping foes, Watchful, chant answer, two triggers, Guardian Arrow.
6. **Marksman:** Called Shot, cooldown, Pinning Crit, back-row Attack,
   unblockable crits, Second Nock (once a round), Perfect Shot.
7. **Ravager:** Hunt Down bleeding stacks and flee penalty (flight stays
   possible), Harrow, Rend, Apex morale checks.
8. Each ability through a real battle, for a player and a companion, with
   its strategy role; narration lines; help pages (`help pathfinder`,
   `help swordmaster`, `help nightblade`, `help sentinel`, `help marksman`,
   `help ravager`, updates to `help rogue`, `help ranger` and their advanced
   route pages, the `help elite` table); balance cells at 35 and 50;
   Project Status.

## 38c3 tasks

1. Task 0 for the six wizard and witch elites.
2. **Archon:** Counterspell held turn and edge roll (boss −25), Spellward
   for 2, Mana Shield, Reflection, Archon's Aegis.
3. **Archmage:** Overchannel, chant reduction, mana discount, Arcane
   Barrage, Steady Casting, Archmage's Storm.
4. **Necromancer:** Raise the Fallen through 38b's summon framework (thrall
   template from the fallen mob, half then 75% HP, weapon attacks only,
   bosses refused, never saved, banished by copyover, kills credited), Drain
   Life redirect, Grave Chill, two raises, Death's Harvest, Lich's Bargain.
5. **Wise One, Coven Mother, Crone of Ash** (§6), with the 38a sleep and
   paralysis caps and re-hex immunity held (tests at the cap).
6. Each ability through a real battle, for a player and a companion;
   narration; help pages (`help archon`, `help archmage`, `help
   necromancer`, `help thrall`, `help wise one`, `help coven mother`,
   `help crone of ash`, updates to `help wizard`, `help witch`, `help
   hexes`, `help summoning` and the advanced route pages); balance cells at
   35 and 50; Project Status.

## Invariants (every slice)

- No player input mid-battle; every ability is used by strategy.
- No free actions; held turns spend the turn; per-battle uses are runtime
  only.
- Ranks derived from level and class, never saved; battle state never saved.
- Never advance global game time; copyover and restart keep classes and
  talents and drop battle state.
