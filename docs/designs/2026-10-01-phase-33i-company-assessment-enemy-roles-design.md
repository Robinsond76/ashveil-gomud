# Phase 33i: Company Encounter Assessment and Enemy Roles — Future Design

Status: future design, 2026-10-01. The owner endorsed the gameplay-review
recommendations and requested these planning documents. Detailed mechanics,
command names, migrations, and balance defaults below remain proposals;
implementation is not started or authorized by this documentation change.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing, shared constraints, and the decision register. Code context
was checked against origin master `626df427`; recheck before implementation.

## Goal and scope

Encounter information describes company-level risks, and enemy groups offer coordinated roles that make formation, recruitment, and tactics matter. Estimates explain uncertainty instead of promising a result.

## Current mechanics and prior art

`internal/usercommands/consider.go` compares the leader against one character through CombatOdds. `scout.go` already gives visible group formation, health words, and the leader's reach. The 30g1 baseline records no enemy healer, first-strike asymmetry, no enemy wounds, and unplaced formations. Existing enemies have personalities, casters, guardians/interception foundations, wind-ups, and now 30e morale; extend these rather than invent a second combat engine.

Ogre Battle guides composition, formation, and automatic member behavior;
Mount & Blade II guides company readiness, specialists, and broad orders.
These are design inspirations, not promises to reproduce their mechanics.

## Proposed behavior

- Deliver 33i1 group assessment and 33i2 coordinated enemy roles separately.
- Make assessment describe visible counts, composition, formation/reach, exposed support, and the company's actual present readiness. Keep solo consideration explicitly personal or redirect it with an explanation.
- Use qualitative risks/confidence, not exact hidden enemy HP, mana, stats, or guaranteed win percentages. Specialist unlocks may improve intelligence only under an approved information policy.
- Distinguish visible information from inference: hidden enemies, unobserved spells, and reinforcements remain uncertain. Darkness and enemy labels follow scout/battle-view rules.
- Ship opt-in enemy groups with fighters, ranged support, healers, guardians, and approved automatic abilities. Respect resources, turn budgets, reach, defense, interrupts, and surrendered/flight states.
- Review the enemy-wound asymmetry as a separate explicit decision; do not silently apply persistent player wound rules to respawning monsters.
- Rebalance representative fights with healer-versus-healer and placed formations alongside the original no-focus baseline. Include focused play, terrain, burden, guardians, and retreat where available.
- Report casting, statuses, wounds, guards, morale, and relevant threat changes in text/GMCP/battle view as supported by the shipped mechanics.

## State ownership, persistence, and recovery

Assessment is a read model from authoritative company/enemy/battle state, never a second simulation that mutates combat or consumes the gameplay RNG. Enemy roles use template/config metadata plus existing strategy/combat execution; dynamic encounter state belongs to battle/mobs. UI observes the same visibility-filtered data as text.

Preserve shared world time and existing game-loop ownership. Do not advance
the world clock for a local action. Browser and text commands use the same
validated backend; events and GMCP report state rather than own it. Existing
saves remain loadable, and every new durable field needs a migration/default.

## Dependencies and delivery

33i1 can improve visible assessment early; intelligence improvements depend on 33f. 33i2 depends on 33e/33b and coordinates with 30g6 tuning. Multiplayer estimates require 33d's participation contract; retreat/morale use 33c/30e.

Before coding, inspect root/nested guidance, settle the decisions below,
and write a focused execution plan with separate implementation,
help/tutorial, integration tests, migration/recovery, and review tasks.
Use an isolated feature worktree and the existing independent review gate.

## Decisions to settle before implementation

Assessment vocabulary and confidence, specialist knowledge limits, initial enemy role kits, wound applicability, regeneration fairness, difficulty bands, and where enhanced-role balance runs after the original 30g tuning.

The planning request approves documenting the direction, not unresolved
formulas or behavior changes. Record final owner decisions in this design.

## Final implementation decisions: 33i1 (owner, 2026-10-01)

33i1, company encounter assessment, was settled with the owner on
2026-10-01 ([plan](../plans/2026-10-01-phase-33i1-company-assessment.md)).
33i2 (coordinated enemy roles) is separate; its decisions follow below.

1. **Surfaces.** `scout [group]` ends with the assessment, below the grid.
   `consider [enemy]` gives the same assessment for that enemy's whole group
   (a lone creature is a group of one), without the grid. The owner
   retired the old one-on-one odds as misleading in a company game.
   `consider` on a player refuses. The web Battle view shows the same
   outlook (`Company.Battle.outlook`).
2. **Risk words:** an easy fight, a fair fight, a hard fight, a grave
   risk, hopeless. No numbers or percentages anywhere.
3. **Confidence** depends only on how close the estimate is: "it could go
   either way" near even or near the line between two words, else "the
   odds look clear". It never depends on anything hidden.
4. **Method.** Deterministic: each counted member's expected damage per
   round (combat's existing no-dice formula: gear, stats, defense, dodge,
   burden) against the visible foes it can reach from its place now,
   averaged over them; staying power is current health. Ratio =
   (company health / foes' damage) ÷ (foes' health / company damage);
   bands at 2.5, 1.25, 0.8, 0.4. Reach uses `formationcombat.Legal` with
   the alive map `scout`'s marks use; an unplaced member fails open, as in
   combat. Reads only: no state, no RNG.
5. **Counted:** the leader and living companions walking with them who
   haven't surrendered. Listed as not with you: fled, awaiting, fallen,
   and separated companions. Not counted: allied companies (a note when an
   ally is here), pets (left out silently; they may be removed), spells,
   healing and abilities on either side, hidden foes, waiting groups, and
   reinforcements (always said).
6. **Factors shown:** who is counted and missing, own members hurt
   (wounded or worse) or burdened, own members who can reach none of them,
   visible foes none of yours can reach.
7. **No specialist bonus** in 33i1; Read the Trail is unchanged.
8. **Visibility** follows scout: hidden foes are left out, darkness shows
   nothing (no outlook in the Battle view either).
9. **Help:** new `help assessment`; `help scout`, `help consider` (now a
   template), `help combat`, and `help webclient` updated; the Combat
   lesson's scout hint mentions the assessment.

## Final implementation decisions: 33i2 (owner, 2026-10-02)

The owner settled decisions 1–3 and 6 on 2026-10-02 and delegated 4 and 5
to the lead; the lead's choices and the details under each are recorded
here ([plan](../plans/2026-10-02-phase-33i2-coordinated-enemies.md)).
Code context checked against master `f4e28dd`.

What enemies do today: every hostile group fights as fighters. Each member
re-aims by its personality (`targeting`, `targetingnoise`, by template or
race) and may start a wind-up. No enemy heals, guards, or casts through the
strategy system, only company members take wounds
(`combat.go`: `woundable`), and a hostile mob out of a fight never regains
health (`NewRound_AutoHeal` heals players and companions only).

1. **Coordination grows with level (owner).** Every hostile group has a
   coordination tier from its members' average level when the battle
   begins, rounded down. Low-level groups are coordinated, but not
   smartly. A template's `coordination:` sets a tier outright (an
   unusually disciplined low-level patrol, a mindless high-level beast).
   The highest tier among a group's explicit settings wins over the level
   tier.

   | Tier | Levels | Word | Behaviour |
   | --- | --- | --- | --- |
   | 1 | 1–9 | a rabble | Members keep their own aim. Targeting noise at least 30%. A healer heals an ally below 30%; at most one enemy heal a round. No guards. |
   | 2 | 10–24 | a band | Half the fighters (rounded up) take the group's focus: the foe the leader is aiming at. Noise at least 15%. Healers heal below 50%. One guard a battle. |
   | 3 | 25–44 | a drilled company | Every fighter takes the focus. The leader picks it by the `casters` rule when the other side has a caster, else by its own rule. Noise at least 5%. Healers heal below 60% and don't double up (as 33e). Two guards a battle; a caster aims spells at the focus. |
   | 4 | 45+ | a veteran company | As tier 3, with no noise. The focus switches to a foe who is chanting a heal, and healers heal below 70%. |

   Coordination changes only choices: never stats, damage, health, the
   number of actions, or a member's reach.

2. **Roles are opt-in by template.** A template's `role:` (`fighter`,
   `healer`, `caster`, `guardian`) gives an enemy its role, with its
   spells from its spellbook. A mob with no `role:` is a fighter. Enemy
   healers and casters use the same `strategy.Decide`, `autoSpellTargets`
   and `startCast` path as companions. That means mana costs, chants,
   interrupts, reach and turn limits all apply, and surrender or flight
   cancels a chant. An enemy guardian uses the 30c2 guard rules with the
   tier's limit. The tier's healing threshold replaces the company's
   tactics threshold. 33i2 gives roles to a small set of shipped groups,
   at least one healer group and one guardian group in each of tiers 1–3,
   so each role is met early. Wider content follows later.

3. **Enemies take light wounds (owner).** An enemy that is hit hard takes
   wounds as a company member does, except that they are all **light**. A
   crit leaves a light wound of its kind instead of a lasting one, and a
   crushing blow or an expired bleed leaves a light wound as now. Light
   wounds hold back health from healing and close when the fight ends.
   They never persist, so a respawned monster never carries one. A
   template's `wounds: none` exempts it (constructs, oozes, spirits); the
   shipped hostile templates are audited for it. Companions are
   unchanged.

4. **Enemy recovery (owner, value chosen by the lead).** A hostile mob
   that survives a fight recovers from 0 to full health and mana in
   **5 game hours**: 188 rounds, about 12½ real minutes at the shipped
   900-round day. That is 1/188 of its maximum each round, rounded up,
   applied in the existing every-third-round heal pass. It recovers only
   while out of battle and not aggressive, and its light wounds have
   already closed. Recovery runs on rounds, so a wanderer that runs from
   a fight heals over the same real time wherever it goes, and world time
   never advances for it. Mobs aren't saved across restart, so recovery
   has no durable state. Random encounters (the
   [zone encounters design](2026-10-01-random-room-encounters-design.md))
   despawn after their battle, so this rule doesn't apply to them. That
   design should record this when it is built.

5. **Difficulty bands (lead).** The 33i1 risk words and their bands
   (2.5, 1.25, 0.8, 0.4) are unchanged. Coordination, healing, and guards
   are not folded into the ratio, because a number for them would be a
   guess. The assessment says them in words instead (decision 7).
   Balance target: against an even no-focus 5v5 at each harness level, a
   tier's role group may lengthen the median battle by at most 25% and
   must not flip the original baseline's winner. The harness records
   these bounds and asserts them.

6. **When the balance runs (lead).** 33i2 adds role and tier cases to the
   30g1 harness now and asserts decision 5's bounds against today's
   numbers. 30g6 later tunes HP, damage and healing with these cases
   included, so the 10–15-round target covers coordinated groups too.
   33i2 does not retune any number 30g4–30g6 own.

7. **Assessment notes coordination (owner).** `scout [group]`,
   `consider [enemy]` and the Battle view's outlook add a line naming the
   group's coordination word and the roles it shows, for example "They
   fight as a drilled company: a healer and a guardian among them." The
   roles shown are those visible to `scout`: a hidden member's role isn't
   named, and darkness shows nothing, as now. The risk words still
   describe blows only, and the existing "not counted" line adds healing
   and guards, so the player knows the estimate leaves them out.

8. **Narration and feeds.** Enemy heals, guards, casts and focus changes
   use the companion lines and `combatstream` events, so text, GMCP and
   the Battle view report them with no new channel. A tier-2+ group
   announces its focus once per change ("The bandits close on [member].").
   Tier 1 announces nothing.

9. **Help.** New `help coordination` (tiers, the words, roles, enemy
   wounds and recovery). Updates to `help assessment`, `scout`,
   `consider`, `combat`, `wounds`, `guardian`, `tactics` and `webclient`.
   The Combat lesson's scout hint mentions the coordination word.

## Acceptance criteria and verification

Assessment changes with present members, burden, gear, wounds, and formation without exposing hidden values; darkness and hidden foes stay hidden; no state/RNG mutation; real enemy healing/guarding/abilities respect costs and targets; surrender/retreat cancel actions; placed/no-focus and focused multi-level harnesses report outcomes without false guarantees.

Cover each wired real entry point and failure/recovery path, then perform
independent phase review and the required code checks. Validate multiplayer
ownership and no world-time advancement. Record actual checks and findings
in Project Status. This planning change itself requires documentation checks
only; the gameplay checks above are future acceptance requirements.

## Player help and tutorial acceptance

Update `help consider`, `help scout`, `help combat`, strategy/guardian/morale pages, and webclient battle-view guidance. Tutorial scout lesson explains group risk and uncertainty.

Ship player-facing pages as indexed `.template` help with useful aliases and
hub links. Explain commands, costs, eligibility, and numbers that matter;
use `[member]` placeholders. Rendering tests and
`TestTutorialHelpPointersExist` must pass with the implementation. Do not
publish help claiming these future mechanics already exist.
