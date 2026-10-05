# Ashveil Project Status

**Phase 40a3 design approved (2026-10-05):** the owner approved the camp
gear design as written: the fuel rule, the six items, and their weights,
effects and prices. The execution plan comes when the visual milestone
starts. Documentation only.

**Camp gear decisions (2026-10-05):** bedrolls are a bonus only. Camp
theft is deferred as a future feature: a silent camp event when a company
rests without bells and trip lines, noticed as missing loot on waking.
Recorded in the 40a3 design and the deferred items. Documentation only.
Verification: links and the diff checked.

**Camps need firewood; camp gear designed (2026-10-05):** the owner
approved the 40a2 rule that a camp fire needs firewood. They asked for camp
items that boost camping. Added the [40a3 camp gear](designs/2026-10-05-phase-40a3-camp-gear-design.md)
design: one bundle per rest, with embers between rests, and six durable,
weighted items. These complement the single-use camp consumables design.
Documentation only. Verification: relative Markdown links and the diff
checked; no Go tests required.

**Visual milestone decisions (2026-10-05):** the owner answered the
milestone's questions.
- **Sprite art direction:** mature, grounded high fantasy in the Lord of
  the Rings tradition, for a dangerous, unforgiving world. Not anime in
  any way. OB64 sprites are the reference. Recorded in the
  [sprite specification](designs/2026-10-05-sprite-specification.md).
- **Other players** are not on the map for now (deferred).
- **Every room resource is wanted.** The new [40a2 gathering](designs/2026-10-05-phase-40a2-gathering-design.md)
  design covers herbs, firewood, fishing and game, with room pools and
  firewood for the camp fire.
- **A showcase area** may be built for the new features (40d now
  recommends a new zone). World building moves after the milestone.
- **No race variants** for now.
- **An installable web app** is enough.
- **Travel and battle-screen auto-open** are approved.
- **Allied companies on the battle screen:** 40f recommends half-scale
  reserve formations with a read-only swap view, pending approval.

Documentation only. Verification: relative Markdown links and the diff
checked; no Go tests required.

**Phase 40a–40g designs drafted (2026-10-05):** at the owner's request,
added a design for each item of the visual roadmap. These are drafts
awaiting owner approval; no code has changed.
- 40a: room resources, with water wired into survival (`drink water`,
  `fill`), forage and shelter.
- 40b: class sprites, the company badge and camp markers (party camps
  only).
- 40c: terrain and landmark tiles.
- 40d: a tile-ready pilot region and `travel` (click-to-walk).
- 40e: structured combat-event messages released with paced narration.
- 40f: a static OB64-style battle screen.
- 40g: battle animations and effects.

The drafts are linked from the Phase 40 table below. Documentation only.
Verification: relative Markdown links and the diff checked; no Go tests
required.

**Visual client milestone planned (2026-10-05):** the owner chose to keep
the browser client and add a visual layer:
- a colorful tile map with class sprites, camps and room resource icons,
  inspired by MUME's mapper;
- an Ogre Battle 64–style battle screen.

Added the [milestone design](designs/2026-10-05-visual-client-milestone-design.md)
(phases 40a–40i) and a [sprite specification](designs/2026-10-05-sprite-specification.md)
listing every sprite in art sets S0–S7, for another agent to generate. The
milestone starts after the current phase sequence. Other companies' camps
stay hidden; PvP camp visibility is deferred. Documentation only.
Verification: relative Markdown links and the diff checked; no Go tests
required.

**Roadmap and designs (2026-10-05):** the owner reviewed the game's
progression, balance, loot and activities. Recorded the owner's decisions:
- the encounter contract (2–3 under-level foes, occasionally 4; bosses of up
  to 5 with no strategy);
- every level should matter;
- casters with less fizzling and more mana;
- stronger healing;
- a Witch class with hexes;
- a loot-heavy game;
- the first promotion at level 10;
- a new roadmap order.

Added the [level impact and class power](designs/2026-10-05-level-impact-class-power-design.md)
and [loot system](designs/2026-10-05-loot-system-design.md) designs.
Follow-up owner decisions:
- the Witch is a starting class;
- item level requirements;
- personal loot per allied company;
- unidentified Rare+ items with a caster Scribe;
- a stat point every 2 levels, kept in balance;
- mana refills only by rest or costly draughts, and healers patch the company
  up until they must camp;
- enemy levels set by zone bands, with strategy rising with enemy level;
- hex spell names, including a Miasma poison cloud;
- passive HP recovery trickles only up to 50% of max HP;
- Scribe reuses the retired `scribe` skill. Casters train it, including
  companions after joining, and some recruits already know it. It identifies
  automatically at camp and costs mana on command in the field;
- companions earn training points for optional skills; class specialist
  skills stay automatic by level. Owner
updates went into the branching class, random encounter and expanded class
designs. Documentation only. Verification: relative Markdown links and the
diff checked; no Go tests required.

**Equipment documentation updated (2026-10-02):** removed the proposed new
class and its ability, kit, recruitment, help, and delivery requirements at the
owner's request. Retained glaive weapons, equipment families, tiers, and armor
paths; renamed the equipment design and updated its links and roadmap scope.
Verification: documentation diff, residual-reference search, and relative
Markdown links checked; no gameplay changes or Go tests required.

Living status log for the Ashveil-on-GoMud migration. Update this file whenever a
commit lands or a phase completes, recording **what was done**, **why**, and
**which step/phase completed**. Keep it short and current; link to detailed docs
instead of duplicating them.

- **Last updated:** 2026-10-05 (35a2 skill over hit points built, in review as PR #18, owner chose to merge and retune balance in 35b; 35c companion training merged via PR #16; 35a merged via PR #15; 35a2 skill-over-HP design approved and planned, faith routes design approved (alignment wait at elite, broken binding and hungry dark healing kept, routes final); visual client milestone and sprite specification added; roadmap reprioritized; 35b pending implementation)
- **Latest completed slices:** 35c, companion training; 35a, level impact; 30g5, action meter; 30g4, progression; 30f, battlefield conditions; Phase 34 review follow-up; 33i2, coordinated enemies; 34d, effects and current capabilities; 33h3,
  relocation and separation; 34c, equipment editor; 33h2, readiness and recovery; 34a,
  UI/formation; 34b, packs/capacity; 33h1, companion growth and contracts
  (2026-10-02);
  33g management (equipment catalog/class still pending),
  33i1, company encounter assessment, and
  33f3, camp specialists (built in parallel); 33f2, expedition specialists; 33f1, skill
  and charm retirement; 33e,
  automatic class abilities; 33d, allied
  companies; 33c, company retreat; 33b, friendly effects; 33a, command
  rules; and 30g3, personal load and agility, all 2026-10-01.
- **Upstream baseline:** `39e44013` (GoMud Telnet input-masking fix).

## Roadmap priorities (owner, 2026-10-05)

After a general game review, the owner reordered the next work. World
building waits until these ship. The level-impact design is owner-approved;
35a is complete and merged (PR #15). The
remaining slices are pending implementation or design review as listed below.

1. **Level impact and caster power.** [Design](designs/2026-10-05-level-impact-class-power-design.md)
   (slices 1–2):
   - every level counts: smooth stat growth, a stat point every 2 levels, and
     HP growth through level 20;
   - spells and abilities scale with level;
   - owned spells never fizzle in battle, and the roll-100 bug is fixed;
   - large caster mana pools that refill only through rest or costly mana
     draughts;
   - stronger healing, with healers patching the company up after battle
     from their mana until they must camp;
   - passive HP recovery outside rest only trickles up to 50% of max HP;
   - companions earn derived training points to learn optional skills
     (Scribe, Cooking, later Alchemy) with `company train`; class specialist
     skills stay automatic;
   - harness cells for the owner's encounter contract: groups of 2–3,
     occasionally 4, and bosses of up to 5, with levels set by the zone's
     band (not the player) and strategy growing with enemy level.
2. **Loot system and gear catalog.** [Design](designs/2026-10-05-loot-system-design.md):
   - tiers, quality, rarity, item level and affixes;
   - level requirements on Uncommon and better items, personal loot for each
     allied company, and unidentified Rare+ items read by **Scribe**, a
     trainable caster skill: automatic at camp, and mana-costing on command in
     the field;
   - legendaries and sets;
   - sellable goods for horse hauling, with salvage;
   - zone and boss drop tables.
   Absorbs the approved [equipment tiers](designs/2026-10-01-equipment-tiers-design.md)
   catalog slice. Moved up by the owner.
3. **Random room encounters.** [Design](designs/2026-10-01-random-room-encounters-design.md),
   updated with the encounter contract. Moved up by the owner. Its drop tables
   come from item 2.
4. **Class promotion at level 10 and class routes.** [Branching design](designs/2026-10-01-branching-class-progression-design.md):
   - level 10 is now an owner decision;
   - talents at levels 5, 15 and 25;
   - the **Witch** as a sixth base class (owner decision), with
     level-scaling hexes: Slumber, Earthbind, Leaden Curse, Miasma (a poison
     cloud), Binding Hex, Curse of Frailty, Dread Whisper and Blight;
   - its proposed Hedge Witch, Coven Sage and Hag routes, from the level
     impact design (slice 3).

Items 1 and 2 can be designed in parallel. Item 1 ships first because the
loot and encounter tuning depend on its numbers. Earlier items, including the
30g mirror target, remain as stress checks, not the tuning goal.

### Phase sequence (proposed, 2026-10-05)

Each phase gets its execution plan in `docs/plans/` and the usual
review gate. Each design needs the owner's approval before its first phase is
implemented (handoff rule 20).

| Phase | Scope | Source | Depends on |
|---|---|---|---|
| 35a | Level impact: smooth stats, stat point every 2 levels, HP to level 20, level-up report, zone-band harness cells. [Plan](plans/2026-10-05-phase-35a-level-impact.md), complete (PR #15) | Level impact §1, §4.1 | — |
| 35a2 | Skill over hit points: derived Attack and Evasion ratings by level and class, one skill edge added to every opposed chance (block included), small HP growth with a 15–25% landed hit, a smaller Strength damage bonus, armor bulk with a significant untrained penalty (warriors the tanks), shields for warriors and rangers (bucklers) only, cleric staffs/rods/maces, spell and heal numbers sized to a weapon hit. [Design](designs/2026-10-05-phase-35a2-skill-over-hit-points-design.md), owner-approved 2026-10-05; [plan](plans/2026-10-05-phase-35a2-skill-over-hit-points.md); built, in review as [PR #18](https://github.com/Robinsond76/ashveil-gomud/pull/18) ([measurements](plans/2026-10-05-phase-35a2-measurements.md)); owner chose to merge and retune three balance rows in 35b | Owner direction 2026-10-05 | 35a |
| 35b | Caster power: no fizzle, roll-100 fix, scaling spells and abilities, caster mana pools, no passive mana, mana draughts, healing and after-battle patching, the 50% HP trickle, the easy-fight wound change. [Plan](plans/2026-10-05-phase-35b-caster-power.md), pending implementation; spell, heal and HP numbers wait on 35a2 | Level impact §2, §4 | 35a, 35a2 |
| 35c | Companion training: derived points, `company train`, trained optional skills (Cooking first). [Plan](plans/2026-10-05-phase-35c-companion-training.md), complete, merged via [PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16) | Level impact §5 | 35a |
| 36a | Loot item model and generator: layers, affixes, level requirements, display, persistence; Scribe and identification | Loot design slice 1 | 35b, 35c |
| 36b | Tier 1–3 gear catalog, goods and an audit of existing items | Loot slice 2; equipment tiers | 36a |
| 37 | Random room encounters and zone level bands, with drop tables, caches, boss rolls and personal loot (loot slice 3) | Encounter design; loot slice 3 | 35b, 36b |
| 38a | Witch base class: hexes, three new statuses, controller role | Level impact §3 | 35b |
| 38b | Class promotion at level 10, talents at 5/15/25, core routes for all six lineages; cleric and warrior routes per the approved [faith routes design](designs/2026-10-05-faith-routes-design.md) (summoned Angel and Demon, Paladin and Blackguard fighting healers) | Branching design; level impact §1e; faith routes | 35a, 35a2, 38a |
| 36c | Loot economy: goods in markets, saturation, salvage, `sell junk`, identification fees | Loot slice 4 | 37 |
| 38c+ | Elite promotions (level 30), tier 4–6 gear, legendaries and sets, expanded class catalogue bundles | Later | 38b, 36c |

World building (zones for levels 1–15) now waits until the visual client
milestone below is in place (owner, 2026-10-05). A small showcase area for
the new map features may be built in 40d.

### Next milestone: visual client (Phase 40, owner 2026-10-05)

Starts once the phase sequence above is finished. Art can be produced
earlier, because it touches no code. See the [milestone design](designs/2026-10-05-visual-client-milestone-design.md)
and the [sprite specification](designs/2026-10-05-sprite-specification.md).

| Phase | Scope | Art set |
|---|---|---|
| [40a](designs/2026-10-05-phase-40a-room-resources-design.md) | Room resources: data, `look` line, GMCP, map icons, water in survival, forage, shelter | S1 |
| [40a2](designs/2026-10-05-phase-40a2-gathering-design.md) | Gathering: herbs, firewood, fishing, game; room pools; firewood for the camp fire | S1 |
| [40a3](designs/2026-10-05-phase-40a3-camp-gear-design.md) | Camp gear: a firewood bundle per rest, plus bedroll, tent, fire steel, cookpot, bells, surgeon's kit. **Design approved** | S1 |
| [40b](designs/2026-10-05-phase-40b-map-sprites-design.md) | Class sprite on the map, company badge, own and allied camps | S0, S1 |
| [40c](designs/2026-10-05-phase-40c-terrain-tiles-design.md) | Terrain and landmark tiles, fog, classic toggle | S2 |
| [40d](designs/2026-10-05-phase-40d-tile-region-travel-design.md) | Tile-ready pilot region and click-to-walk | S2 |
| [40e](designs/2026-10-05-phase-40e-combat-event-messages-design.md) | Structured combat events (can run in parallel with 40a–40d) | — |
| [40f](designs/2026-10-05-phase-40f-battle-screen-design.md) | Static OB64-style battle screen | S3 |
| [40g](designs/2026-10-05-phase-40g-battle-animation-design.md) | Battle animation and effects | S4 |
| 40h | Advanced class art (after 38b) | S5 |
| 40i | Touch layout and installable web app | S1 |

Open owner questions are listed in the design: other players on the map,
the resource list, tile-ready world building, race variants, and a store
app.

**Phase 35 delivery (2026-10-05):** 35a is complete and merged
([PR #15](https://github.com/Robinsond76/ashveil-gomud/pull/15)). 35c is
complete and merged ([PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16)). 35b remains pending implementation. Plans and approved scope are in
[the phase 35 handoff](plans/2026-10-05-phase-35-handoff.md).

## Current position

**35a2 skill over hit points (2026-10-05), built, in review as [PR #18](https://github.com/Robinsond76/ashveil-gomud/pull/18):**
Attack and Evasion are derived from level and class (warrior 1/1, rogue
0.9/1.1, ranger 1/0.9, cleric 0.7/0.75, wizard 0.7/0.75, enemies 1/1 plus a
template offset), never saved. Their gap over `SkillEdgeSpan` (tuned 20 → 16)
is added to every opposed chance's stat edge except crits; dodge and parry
are two-sided around 12%, block is 20 (tuned from 15) plus shield armor. HP is
48 + HPStart + the class rate through level 20, a quarter after; the damage
bonus is 6–12 from Strength. Armor bulk (medium −8% tempo, −20% dodge; heavy
−20%, −50%), doubled when untrained with −10 Attack and Evasion and +1 chant
round. Shields: warriors any, rangers bucklers, others none; clerics wield
staffs, rods and maces, with a holy symbol (+5% heals). `SettleClassGear`
puts away disallowed gear on spawn, class choice and copyover. Spells, heals
and Opening Strike scale with the gap and level. New `help evasion` and
`help shields`; armor, defense, stat-edge and the other pages it changes updated; tutorial
hints. Wolf pelt, snow wolf mane and spider exoskeleton retagged medium.

**Balance:** [measurements](plans/2026-10-05-phase-35a2-measurements.md).
Passing: skill wins (L20 beats 3×L10 100%, losing 5.3% HP; L10 beats 2×L20
12%), mirror parity, kit, targeting, sides even, statuses. **Owner
questions:** equal 5v5 fights run a median of 14–20 rounds (target 8–12;
30g6 was 12–14), because two-sided defenses and fifth-of-HP hits can't both
meet it; a warrior takes 0.84× a rogue's damage per swing (target 0.7×);
L15 now always beats L10 (the 30g6 row wants it to lose sometimes); enemy
coordination tiers 2–3 drop a level-10 company to 37% and 26% wins (33i2
wants ≥ 50%). **Owner decision (2026-10-05):** merge 35a2 as is; the
fight-length, tank and coordination targets move to the
[35b plan](plans/2026-10-05-phase-35b-caster-power.md)'s acceptance, and the
"L15 can lose" row is retired (its assertion removed).

**Review:** an independent full-diff reviewer found no blocking bug.
Accepted and fixed with regression tests: `help stat-edge` contradicted the
two-sided defenses (rewritten); a readied Opening Strike that never swung
kept its bonus for a later backstab (cleared); prediction helpers untested
against skill (`TestPredictionsFollowSkill`); measurements and status not
recorded (this entry); `help armor` said bulk adds to burden and gave wrong
examples (fixed); missing coverage of plain `equip`'s warning, `status` in
untrained armor, the companion level line and HPStart through a spawn
(added); cleric class text and listings lacked gear rules (each class's
listing now names its shields and weapons); the tempo hint and bash line
omitted bulk and skill (fixed); an out-of-range bulk share dropped to 0
(now held to 0–1); the equipment preview cache ignored the class (keyed).
Accepted as deviations: `equip` warns after an untrained piece goes on, not
before; the untrained −10 is 0.625 of an edge at span 16, not half.
Rejected: the damage bonus taking the skill edge (decision 4 keeps it
Strength-only so hits stay a steady share); missing `Bulk*`/`Untrained*`
keys defaulting to the shipped values (0 is a valid "off"; shipped config
sets them); caching the profile on the hot path (no measured cost); a
registry-reload test for companion gear (the move is idempotent on every
spawn and the next snapshot saves it; nothing can be lost).

**35c companion training (2026-10-05), complete, merged via [PR #16](https://github.com/Robinsond76/ashveil-gomud/pull/16):** companions
earn training points from level (`ProgressionConfig.TrainingPointsAt`, level 1
counts), never banked: points = earned minus the cost of trained ranks (a rank
costs its own number, so rank 4 costs 1+2+3+4). A lost level can leave a
companion owing points; views show 0 and the debt. `company train [member]
[skill]` previews and `... confirm` (idempotent when it carries the rank)
trains: ranks 1-2 at the leader's own camp or a trainer, 3-4 only at a room
whose `SkillTraining` covers the rank. It refuses in battle, while travelling
or resting, and for dead, separated, absent or fighting members. Optional
skills live in the archetype overlay (`OptionalSkills`; Cooking for everyone
first). Some recruits arrive already trained (20%, one in five of those at
rank 2, price +15% per rank). `camp cook` uses the best cook present among the
leader and living companions in the camp room (ties: leader, then lowest ID)
and names them. GMCP `Company` members carry `skills` and `training_points`;
the web client's Skills tab shows a Company training section. Help:
`company-train` (new) plus `company`, `growth`, `skills`, `cooking`,
`specialists`, `webclient`; Camp tutorial hint. No game time advances.
Deviations from the plan: trained ranks are stored on `company.Companion`
(`Skills`, `GrantedSkills`) rather than `MemberState`, so gear and level
snapshots can never drop them; confirm accepts an explicit rank; `company
inspect` matches your own companions by exact name before recruits and
templates. See the [plan](plans/2026-10-05-phase-35c-companion-training.md).

**Review:** independent full-diff reviewer found no P1. Accepted and fixed:
P2 no command-path best-cook test (added `TestCampCookCommandUsesTheBestCook`
through `camp cook`; presence still uses the test seam); P2 `company train
[member] confirm` without a skill said "no companion like that" (now asks for
a skill); P3 help overstated confirm idempotence (only the rank-bearing line
is); P3 confirm could spend points from a live level not yet saved (confirm
now writes the live level and experience into the record in the same save);
P3 unused `CookingView.Cook`/`CookIsLeader` removed; P3 cooking help example
corrected; P3 eligibility and max-rank refusals now come before battle,
travel, rest and presence. Rejected or noted: a mob template that defines
`cooking` would cook above its trained rank (no shipped recruit template does;
noted); `company inspect [name]` now prefers your own companion (intended; tested by
name and by `#N`); the client capitalises skill ids rather than
display names (fine for Cooking; revisit with multi-word skills). Also fixed
an order-dependent expectation in `TestCompanyMembersStates`.

**35a level impact (2026-10-05), complete, merged via [PR #15](https://github.com/Robinsond76/ashveil-gomud/pull/15):**
fractional racial growth preserves the fifth-level boundary values; keep
`StatStepLevels: 5` per the approved plan instead of the design draft's 1,
which would multiply the curve. Shared point awards now give one every two
levels, with an atomic, once-only peak-level migration on normal load and
copyover. HP grows at the full class rate through 20, then 60%. Player and
companion reports show final before/after values; experience/status announce
planned milestones as `(coming)`. Admin previews, save/reload, help and tutorial
follow the same rules. No milestone is shipped by 35a. 35b/35c remain pending.
See the [plan](plans/2026-10-05-phase-35a-level-impact.md),
[measurements and limitations](plans/2026-10-05-phase-35a-measurements.md), and
[admin screenshot](verification/phase-35a/progression-editor.png).

**Review:** independent full-diff reviewer found no P1 production blocker.
Accepted: P2 shipped human automatic adjusted stats change on only 8/59
level-ups (levels 5, 15, 25, 30, 35, 45, 55, 60), so the "most level-ups"
acceptance criterion is **unmet** despite faithfully implementing the approved
fractional formula. Recorded as a design limitation; no race or curve retuning
in 35a. Accepted and fixed: P2 missing real rendered stat-line, companion
post-retraining report, and experience milestone/no-future coverage. Also
isolated two pre-existing battle fixtures from automatic Tackle and captain
mortality. No findings rejected. The focused report/migration/fixture checks
passed ten repeated runs. Verification: 8,200 zone fights (100 per cell), zero
stalls; unchanged mismatch assertions pass (97% L15 vs L10, 100% L30 vs L10).
Chromium checked smoothing, cadence preview/save/reload, level-200 downsampling,
API reference and console errors. Final verification passed: `make generate`,
`make validate`, `make js-lint` with installed JSHint, local `make lua-lint`,
`go test -race ./...`, inline admin JavaScript syntax and relative Markdown
links. Gameplay implementation complete; merged via PR #15.

**35a PR review follow-up (2026-10-05):** a PR code review found the
once-only migration marked characters migrated even while the server still
ran the 5-level rhythm (for example through a saved
`StatPointsEveryNLevels: 5` override), so a later switch to 2 never paid them,
and assumed every character earned on 5. Accepted and fixed:
`Character.CatchUpStatPoints` tops a character up to the current rhythm's
total at its peak level, counting points it already holds (unspent plus stat
training), so pre-30g4 characters who earned a point every level, and players
given points, are not paid twice; nothing is taken away. It leaves
characters, new ones included, unmarked while the rhythm is still 5, and runs
on load (atomic save) or before a live level-up, so a mid-session change is
neither skipped nor counted in the level-up report. An independent review of
the fix found no P1. Accepted: P2 new characters were pre-marked (fixed, they
now start unmarked); P2 pre-30g4 characters (rhythm 1) would be overpaid
(fixed by the held-points top-up; the cost is that a stat coupon spent before
the catch-up reduces it). Rejected: P3 announce catch-up points (silent like
the original migration); P3 save the live catch-up at once (points and marker
share one record, so a crash reverts both and the next load repays); P3
display lookups of offline users can run the migration save (pre-existing,
once per character, main-loop only). Regression tests cover the load, live
level-up, held-points, lost-level and new-character paths. Full checks
passed (`make generate`, `make validate`, `go test -race ./...`), except one
run where `TestAClericCompanionHealsTheHurt` (company, untouched) failed once;
it passed 100 isolated runs (40 with `-race`) and two full `-race` package
runs, so it is noted as an unreproduced intermittent failure to investigate.

**30g6 stat edges and tuning (2026-10-04):** on top of 30g6a, per the
[amendment](designs/2026-10-04-phase-30g6-amendment.md) A, B, E and F. Every
opposed chance (hit, crit, dodge, parry, block, bash, crit damage, the
Strength edge in damage, tackle) reads a stat difference over
`StatEdgeSpan` (40) instead of a ratio of small stepped stats; damage adds
`DamagePerStrength` with a small `DamageEdgeMax`; HP keeps each archetype's
proportion after `HPFullLevels`; Minor Heal adds the caster's level (in battle
and in `heal wounds`) and Minor Heal All half of it. The whole opt-in suite
passes at 100 fights a cell: spread mirror medians 12–14 rounds at levels
1–60 (17 at 100), even mirrors, kit no handicap, enemy targeting
significantly worse for the company, level 15 beats level 10 90%, level 30
wins 100% and loses no member in most fights; HP L60/L10 2.09–2.20×. New
indexed `help stat-edge`; updated stat, defense, abilities, brawling, attack,
progression, health, heal and friendly-effects pages; Practice Yard hint. See
[verification](plans/2026-10-04-phase-30g6-verification.md) and
[measurements](plans/2026-10-04-phase-30g6-measurements.md). Supersedes the
first candidate (`phase-30g6-tuning`). **Review:** independent full-diff
review, no P1. Accepted and fixed: P2 `help strength` misdescribed the
Strength edge (now `+DamageEdgeMax` at a full span, per point a share); P2
`heal wounds` still healed a flat 2d3 (healers now add their level,
`TestPlanHealAddsTheHealersBonus` and the real-command check in
`TestHealWoundsALeaderClericHealsToTheLimit`); P3 `DamageEdgeMax` missing
from the admin pages and stale "Default" text there; P3 even-value inputs
allowed 0; P3 `hitChance` now rounds like the other chances; P3 help built
ordinals like "3th" from the span (now "1/40"). Rejected: P3 validation
fallbacks differ from shipped values (they guard invalid input, as the
existing tempo fallbacks do; shipped values live in `config.yaml`); P3 tackle
bounds as code constants (fixed odds shared by both tackle paths and their
help; not a tuned number); P3 Strength's ranking weight ignores the damage
cap (documented "below cap" approximation; rankings use no attacker stats);
P3 no test drives the manual `tackle` command (blocked in every battle since
32c; its chance is the tested shared helper); P3 flee and disarm formulas
(outside the stat edge's scope).

**30g6a combat fixes and balance harness (2026-10-04):** split from the first
30g6 candidate (`phase-30g6-tuning`, `5b36efd`) after a review found its
acceptance failures structural, not numeric: stepped stats broke the ratio
formulas, the "even" fight gave only the company abilities and a healer, and
"focus wins sooner" was measured over all fights. The owner approved a
[30g6 amendment](designs/2026-10-04-phase-30g6-amendment.md) and decided that
company focus is reported, not asserted (a strategy, not a guarantee). 30g6a
keeps master's numbers and ships: no repeated tackle within one ability pass,
caster data on replacement targets, an earned turn surviving an earlier kill
for company members *and* enemies, no game-round ticks for combat statuses,
and the harness (levels 1–100, mismatches, caster targeting, net health, a
true mirror plus a kit cell, Welch significance for enemy targeting). Help:
`tempo`, `abilities`, `combat`. On master's numbers the opt-in suite fails
as expected (level-10 fights about 73 rounds); see
[verification](plans/2026-10-04-phase-30g6a-verification.md).
**Review:** independent full-diff review, no P1. Accepted and fixed with
regressions: P2 a queued tackle opened a same-pass Opening Strike (now a
separate `FoeDownQueued`, `TestAQueuedTackleOpensNoStrikeThisPass`); P2 only
the company kept earned turns after an earlier kill (enemies now re-aim too,
`TestBalanceEnemyEarnedTurnSurvivesAnEarlierKill`); P2 the mirror had no
company caster, so the casters cell measured nearest-foe aims (the cleric
keeps the healer role with no mana, `TestBalanceMirrorClericIsACasterWhoCastsNothing`);
P3 routed or fled fighters counted as health removed; P3 tempo help wording.
Rejected: P3 player-side candidates mark casters by spellbook while enemy-side
ones use role (both are pre-existing selectors for different sides; the new
retarget matches its own side's ordinary selector); P3 the "rabble noise"
comment (tier 1's floor is 30%, as written). P3 a round-two cleanup check is
covered by the new queued-tackle test's second round.

**30g5 action meter complete (2026-10-03):** effective Speed and personal
burden now determine fractional turns, with one opening turn, at most two
physical turns per combat round, and no whole-turn banking. Shared enemies
fill once; fight replacement, separation, unmanaged restarts and death clear
stale progress. Chants, waits, wind-ups and upkeep retain their round cadence;
abilities consume one earned turn and reactions retain their existing budgets.
Target-relative unarmed/claw extra attacks are retired. Simulator and weapon
estimates use tempo. Indexed `help tempo`, related help, and the Practice Yard
hint explain the mechanic. No pet proposal remains in this slice.

Independent full-phase review found five issues (zero-turn waits, separated
companions, simulation ordering, ranking cap and help indexing). All were
verified, fixed and regression-tested; follow-up review found no blockers.
`make generate`, `make validate`, `go test -race ./...` and focused regressions
passed. Final measurements: 540 fights, no stalls, no-focus
medians 16/60/95 rounds at levels 1/5/10; narration cell means 7.3–9.6 lines,
peak 26. Pre-merge PR review follow-up: withdrawn fighters no longer bump the
combat generation, a redundant wait flag and single-pass DPS loop were
removed, and a regression confirms a foe felled in the company's pass makes no
blow (unchanged from master; the suspected behavior change was rejected).
Legacy `ExtraAttacks*` keys stay for old overrides. **30g6 tuning is next**, including the 10–15-round duration target.
See [amendment](designs/2026-10-02-phase-30g5-action-meter-amendment.md),
[plan](plans/2026-10-02-phase-30g5-action-meter.md) and
[verification](plans/2026-10-03-phase-30g5-verification.md).

**30g4 progression complete (2026-10-02):** automatic stats and stat-point
awards step at levels 5, 10, 15, …; skill training remains every level.
HP uses configured archetype rates through level 20, then one per level,
with race/template overrides for enemies. XP retains its thresholds through
level 60, then grows incremental costs by 1.10 without a level cap.
Existing player investment and saved vitals survive load/copyover; maxima
clamp vitals without refilling. Companion class HP uses its durable identity.
Admin controls/charts share live formulas; creation, status, level-up,
indexed help and tutorial explain the new progression.

Independent review accepted and fixed two P2 findings with regressions:
saturated HP/stat additions could overflow, and Mana previews omitted an
intrinsic racial term. Follow-up accepted both fixes with no remaining
blockers or rejected findings. Generate, validate, JS/Lua lint and editor
browser checks and the full `go test -race ./...` suite passed.
The 900-fight balance table had no stalls; no-focus medians at levels 1/5/10
are 14/44/87 rounds. These remain provisional; 30g5 adds the action meter,
and 30g6 owns the 10–15-round target. See the
[plan](plans/2026-10-02-phase-30g4-progression.md) and
[verification/upgrade notes](plans/2026-10-02-phase-30g4-verification.md).

Phase 30g4 PR integration with the Phase 34 review follow-up (`0a2facb7`)
preserved both status records. Independent integration review found a Gear
preview clone dropped class HP, and its cache could retain the old rate after
a class change. Both paths now preserve the resolved HP rate and key the cache
on it. Warrior/wizard player and companion clone regressions, plus cached
preview/class-change/applied HP parity, passed; follow-up accepted the fix with
no remaining blockers. Merged generation, validation, JS/Lua lint and the
full race suite passed. An existing retreat assertion failed on the first
post-fix run; three focused reruns passed, new test provider cleanup was
corrected, and the final full suite passed. See PR #13.

PR #13 pre-merge review (2026-10-02) found no blockers and accepted three
minor findings, fixed with regressions: player characters made by
`NewUserRecord`, `CreateUser`, deletion reset and permadeath reset now carry
their user id, so their HP uses the player's archetype path rather than the
enemy race/template override path; `ValidateActiveCharacters` skips a
missing user; and `archetype` and its choose preview show each live HP rate, which the
progression and stat-train help now point to instead of fixed numbers alone.
A follow-up review of those fixes found the replay character and
`ReplaceCharacter` also missed the id (fixed with regressions) and that
stat-train's "stat-step" wording conflated two settings (reworded). Rejected:
permadeath keeps the archetype registry entry, so the fresh character keeps
the old class rate; this predates 30g4 and Ashveil deaths are never
permanent. Lower early HP from Vitality 4 → 1 is noted as
balance for 30g6, not a defect.
Merged after 30f (#12); the only code conflict, `Mob.Validate`, keeps both
the `hpperlevel` and ambush-stealth checks. The merge also fixes a 30f flake in
`TestTutorialThroughPluginsLoad` (about 60% of runs on master): its own-line
matcher read the new dodge line "You sway aside, and the straw archer's blow"
as Aria aiming at the archer; it now skips lines naming the foe as attacker.
The merged suite also exposed two company test flakes: with 30g4's lower
enemy HP, `TestWaitingGroupsDontBlockFlight`'s battle foe could fall before
the retreat (a third of runs), letting a quick waiting group pursue; the
test now toughens it. `TestAnEnemysBleedLeavesALightWound` (flaky on master
too) now disables crits so only the bleed wounds the captain.

**30f battlefield conditions complete (2026-10-02):** ambush opening
rounds, formation clusters and sweeps, leaps/open flanks, narrow-ground
projections, fatigue hit penalties, and cold chant/sling delays, with indexed
player help, representative content, and Battle view updates; merged via
[PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12) after independent review. Accepted and fixed with regression tests:
(1) a leap was narrated and its 3-round cooldown spent on a strike ordinary
reach already allowed (front cell empty or fallen), leaving it unavailable
over a knocked-down protector; it is now spent only when needed; (2) the
fatigue suffix was appended to every strike; it now appears when a penalty
starts or changes; (3) a sweep could strike a front-row guardian twice (its
ward's blow, then its own); each member is now struck at most once, and the
help says so. Rejected: (4) silent melee turns in the narrow reserve, as
both sides are capped at five members and six cells are active, so no
reserve forms in play; (5) an opening recorded for a battle begun at
end-of-round settlement, as encounter groups begin at the top of a round
(travel and camp spawns never wait behind another battle).
After the fixes: make generate, make validate, JS lint, and go test -race
./... (96 packages) passed; Lua lint not run here (no Docker; no Lua changed).
[Approved design](designs/2026-10-02-phase-30f-battlefield-design.md),
[plan](plans/2026-10-02-phase-30f-battlefield-plan.md), and
[verification](verification/phase-30f/verification.md).

**On-demand Gear editor and refresh load benchmark (2026-10-02):** at the
owner's request, to keep the game loop light with many players. The server
builds `Company.Equipment` only while a client shows the Gear editor, and
previews only the selected slot's choices; the client announces open/closed
and the slot. A cache hit no longer marshals the full character, and a
rebuild reads the company load once. New `TestCompanyRefreshLoad`
(`ASHVEIL_LOAD_BENCH=1`) times one player's whole company refresh per round
(summary, snapshot and every extra) with companions, 36 cargo items and
ticking effects. Before → after, per player per round: Gear closed 0.78 →
0.37 ms; open and unchanged 2.77 → 1.09 ms; open while cargo changes every
round 15.1 → 2.3 ms. At 200 players with Gear closed that is about 74 ms of
each 4 s round. Combat, mob AI and movement are not in this measure. The
largest remaining parts are the inventory payload and the summary.
Independent review: no blockers or majors. Accepted and fixed with tests:
an open message refreshes only when the editor opens or changes slot, and
not within 200 ms of the last (a repeating client can't force rebuilds);
an unknown slot name falls back to the weapon; the web request parsing and
login clearing are now tested through `HandleWebGMCP` and `PlayerSpawn`; a
reset selection is announced at once; the benchmark removes the flags it
registers. Not fixed (cosmetic): a reopened editor shows its last view for
the moment before the fresh one arrives; commands revalidate regardless.

**Phase 34 review follow-up complete (2026-10-02):** an owner-requested
review of the finished Phase 34 found the Gear editor's read model rebuilt
every round per player at about 26 ms with 30 armour pieces in cargo (each
preview YAML-cloned the character). `Company.Equipment` now comes from a
cached view keyed on what its previews read: the leader's character less what
ticks each round (vitals, cooldowns, buff counters, play records), cargo, load
and availability; rebuilt at least every 15 rounds, pruned of offline leaders
on rebuild. Previews share one marshal without the cargo: about 0.9 ms per
unchanged round, 6 ms per rebuild. Also: `Registry.Put`'s unreachable
empty-record deletion removed (every leader keeps a record since 34a, which
holds the 34b pack grant); `Company.Conditions` no longer resends every round
for a ticking countdown (the client counts `seconds_left` down; the change key
uses the end round); effects are tagged Harmful or Helpful when known, from
stat modifiers and new `harmful`/`helpful` markers in `buffs-flags` data
(`BuffSpec.Effect()`), with a duration meter; no doubled full stops in effect
and capability text; clearer pack, burden, conditions and away-member wording.

Independent review: seven findings. Accepted and fixed with regression tests:
(1) effects with only a flag (Bleeding, Poisoned, Stunned) or secret were
labelled Helpful; now three-way from data, tested on the default world's
buffs; (2) the cache key included per-round ticks (buff counters, regen,
cooldowns), so ordinary play rebuilt every round; the key now omits them;
(3) refreshes before and after a round's buff tick keyed end rounds one apart,
resending Conditions after each command; ends within one round now match;
(4) no backstop cleared offline leaders' cached views; pruned on rebuild;
(5) the cached view is now documented as read-only to callers; (6) a
capability line with no description ended in a colon; (7) tests now go
through the provider (`EquipmentViewOf`) and simulate real `Buffs.Trigger`
ticks on both sides of the refresh. Accepted limits: a withdrawn companion
skips buff ticks, so its end round moves and Conditions resends every other
round while withdrawn; `seconds_left` sent from the round refresh can read
one round long. Verification: focused packages, all three browser suites
(effect-card contrast >= 4.5 in every theme), JS lint, make generate, make
validate and go test -race ./... (95 packages) passed. Lua lint not run (no Lua
changed; no Docker or luacheck here).

**33i2 coordinated enemies complete (2026-10-02):** enemy groups
fight by a coordination tier from their average level when the battle
begins (rabble 1–9, band 10–24, drilled company 25–44, veteran 45+; a
template's `coordination` overrides). Higher tiers follow their leader's
focus, go for casters, break heals, and say their focus aloud; a rabble
keeps its own aims with at least 30% noise. Templates may give `role:`
healer, caster, or guardian: enemy healers and casters run the companions'
`strategy.Decide`/`startCast` path (mana, chants, interrupts; single-target
heals only, as a group heal's scope reaches only a company), at 30/50/60/70%
thresholds, a rabble one heal a round; enemy guardians step in through the
attack gates, 0/1/2/2 guards a battle per group. Enemies take light wounds
only (`wounds: none` opts out: skeleton, fungus, bone warden) and recover
out of battle over 188 rounds (5 game hours), prorated by round. `scout`,
`consider`, and the Battle view name the tier and visible roles. Six new
templates (87–93) give tiers 1–3 a healer and a guardian (Frost Lake 310,
Slums 489, Catacombs 113). `help coordination`; assessment, scout, consider,
combat, wounds, guardian, tactics, webclient pages and the Combat lesson
hint updated. See the
[33i design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md)
("Final implementation decisions: 33i2") and
[plan](plans/2026-10-02-phase-33i2-coordinated-enemies.md).

Balance (30 fights a cell, default company vs default/roles groups, on the
final code): median rounds L1 11 → 10/11/12 (tiers 1–3), L5 42 → 39/37/46,
L10 95 → 92/98/94, all within the 125% bound; no clear winner flipped
(company wins L1 13% → 43/23/23%, L5 70% → 70/63/73%, L10 53% → 77/53/53%).
`TestBalanceCoordinated` (ASHVEIL_BALANCE=1) asserts the bounds.

Deviations from the plan: recovery applies to any mob outside a company and
not charmed (not only template-hostile ones); a band also avoids doubling
heals (harmless, as companions do); enemies with no personality keep the old
deterministic weakest pick apart from the tier's noise. The phase 29d
outcome golden was recaptured (an enemy crit's wound rolls its place) and
`breakChant` now asks `enemy()` instead of `!woundable()`.

Independent review: no blocking findings; seven raised. Accepted and fixed
with regression tests: (1) recovery rounded up per round healed small
enemies in ~40 rounds, now prorated so any max takes 188 rounds, and the help
no longer claims fled foes recover; (2) guards and a rabble's heal cap were
counted per player's battle, now per group across battles; (3) a hidden or
not-yet-fighting guardian could step in; (4) a busy leader dropped the
focus, now kept; (5) a hidden member's level could lift the assessed tier.
(7) Coverage added: an enemy caster returns to its player aim, enemy healers
against a solo player, shared guards across two battles. Rejected (6): the
focus line's "closes in" stays singular, as group names are "a band of …";
the band's no-double-heal is kept. Verification after the fixes: make
generate, make validate, make js-lint, and go test -race ./... (96 packages)
passed; the balance table passed its bounds. Lua lint can't run here (no Docker or
luacheck) and no Lua changed.

**34d complete:** Company Status now shows owned members' active effects and
wounds with authoritative duration and mechanical meaning, separate from
persistent bonuses. Away/live, recorded-away/separated, fallen and unavailable states
preserve privacy. Character Skills keeps trained ranks and adds automatic
abilities/spells plus actual field/camp eligibility, including strategy, mana,
autoskill and retirement rules. Camp Cooking is explicitly manual and uses
its owner's configured recipe selector. Character Effects uses safe text and
includes wounds. Existing persistence, readiness and global time are untouched.
Help/tutorial and responsive keyboard/focus checks updated. Independent review
accepted and resolved missing camp cooking with real command/GMCP regressions;
follow-up found no further blockers. A test fixture's stale room membership was
fixed and its paired regression passed three runs. All affected packages, all
three browser suites, generate/validate, JS/Lua lint and the full race suite
passed. [Verification](plans/2026-10-02-phase-34d-verification.md).
Integration with 33h3 (`f4e28dda`) preserved relocation/separation and adds
explicit recorded separated wounds. Independent integration review found no
blockers; real passage/reload/rejoin GMCP regression, all browser suites,
generate/validate, JS/Lua lint and the full race suite passed on the integrated
code. The design, delivery plan and handoff now mark all four slices complete.
Phase 34 is complete; stop here and hand over for the owner's next session.

**33h3 relocation and separation complete (2026-10-02):** every move that
isn't an exit (scripted passages, traps, ropes, portals, an inn room, jail,
a quest `roomid` reward, a journey's arrival and its crash recovery, death,
the tutorial, admin teleport) now brings the living companions standing
with the leader through one step, `company.RelocateCompany(leader, origin,
room)`. A living companion elsewhere, or one away from its leader for two
rounds, is saved and taken off the map as separated, and rejoins beside the
leader after `SeparationRounds` (15, about a minute) of online time once the
leader is out of any fight, journey, or camp rest; a relog never brings it
back early, and no world time advances. Player-requested passages refuse in
a battle (`ActorObject.InBattle()`). The dead, the fled, the herd, and cargo
are unaffected. `help separation`, Departure hint; company, travel, death,
morale, mount, cargo, readiness and retreat pages updated. Owner approved
the defaults (with a clarification on dead companions); see the
[33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h3") and
[plan](plans/2026-10-02-phase-33h3-relocation.md). 33h is complete.

Independent review: ten findings, all accepted and fixed with regression
tests. They were: the `portal` help alias broke 33f1's retired-skill check
(dropped); a failed rejoin spawn was announced and never retried (it now
stays separated and due); inn beds and camp rest tiers counted the separated
(skipped); a separated companion could be fed (`ErrAwayMember`); admin
teleport was untested and silent (test, line added); `help retreat` was
stale (updated); the unused `RejoinSeconds` was removed; JS indentation; and
the refusal line now reads "Not while you are fighting." Speculative findings:
S1 (the sweep firing on ordinary follow) was rejected, because a follower
trails by one turn and the sweep needs two round boundaries (about eight
seconds) away and out of a fight; S2 (separation saves gear early) was
accepted as the same window `BeginFlight` already has; S3 (the tutorial
gate) is as intended, and its wiring test now walks the companions instead
of relying on the old recall. Lua lint can't run in this container (no
Docker or luacheck) and no Lua changed. Verification after the fixes:
make generate, make validate, JS lint, and go test -race ./... passed.

**34c complete:** Character Gear now edits main-character equipment by slot,
including Pack, with exact-instance compatible cargo choices, unavailable reasons,
and authoritative current/after comparisons. Previews and commands share the same
hand, curse, capacity and asset-journal rules; stale actions revalidate server-side.
Comparisons include active edge bonuses/strikes for both hands, core modifiers,
worn burden/dodge and company capacity. Legacy Gear and companion commands remain
available. Help, tutorial pointers and keyboard/focus/narrow browser checks updated.
Independent review found missing sharpening information; fixed with regression
coverage, and follow-up review found no further issues. Additional regressions
cover permanent buffs retained by the other hand and legacy-feed availability.
Real command, save/recovery, private GMCP/reconnect, help and browser checks passed;
make generate/validate, JS/Lua lint and go test -race ./... passed.
Integration with 33h2 (63f79b32) passed independent review, browser checks,
generate/validate, JS/Lua lint and the full race suite; companion readiness
snapshots and recovery remain intact.
34d is complete. [Session handoff](plans/2026-10-02-phase-34-session-handoff.md).

**33h2 readiness and recovery complete (2026-10-02):** companions keep
their health and mana across logout, restart, copyover, and crash (as of
the last save) instead of refilling; they recover online out of battle as
players do, never offline; an inn stay restores the company, a live
level-up no longer refills a companion, and resurrection wakes it at half.
`help readiness`. See the [33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h2", owner-approved) and
[plan](plans/2026-10-02-phase-33h2-readiness-recovery.md). 33h3
(relocation and separation) followed (above).

**34b complete:** each member has one assigned Pack slot; new leaders and recruits
receive a 10 kg cloth knapsack. Shared cargo capacity now comes from packs on
living present carriers plus eligible horses, with no base/Strength allowance.
Only unequipped cargo consumes that capacity; Pack contributes no combat burden,
stats, defense or worn buffs. Migration preserves exact instances and journals
once-only grants; capacity loss preserves overloaded cargo and recovery actions.
Inventory shows assigned container availability and one shared cargo list.
Help, tutorial pointers and migration release notes updated. Independent review
resolved four findings (combat modifiers, 10 kg creation kits, armor-removal
coverage, stale help) and a follow-up weapon-defense regression. Final focused
checks, browser checks, make generate/validate, JS/Lua lint and go test -race ./...
passed. Existing creation, death and mount fixtures now exercise the new rules;
formation integration permits valid upkeep retargeting after clearing a rear foe.
Integration with 33h1 passed independent review, focused command/module checks,
browser checks, generate/validate, JS/Lua lint and the full race suite. Owner
requests a new session between slices; 34c is now complete.
[Next-session context](plans/2026-10-02-phase-34-session-handoff.md).

**34a complete:** shared browser Inventory shows cargo without member worn or
personal blocks; panel foreground inheritance and two light-theme secondary
colors corrected. Solo leaders stay at 2,2; recruitment uses deterministic
vacancies; old formations receive a durable one-time backfill. Last-member loss
recenters the leader and failed enlistment/migration restores placement.
[Phase 34 design](designs/2026-10-01-phase-34-company-ui-logistics-design.md)
and [plan](plans/2026-10-01-phase-34-company-ui-logistics-plan.md).
Independent review found one integration-coverage gap; resolved with real
PlayerSpawn, persisted reload, GMCP/text/combat checks, and repeat-login manual
clear preservation. Follow-up review found no further issues. Verification:
focused packages, help/tutorial pointers, all dock browser checks including
4.5:1 load/label contrast in every shipped theme, make generate, make validate,
JS/Lua lint, and go test -race ./... passed. Initial race run hit the existing
random-hit edge test; targeted rerun and final full run passed. Integration against 33h1 (ecd3255e)
passed independent review, focused checks, browser checks, generate/validate,
JS/Lua lint and the full race suite. 34d is complete.


**33h1 growth and contracts complete (2026-10-02):** the owner asked to
continue with 33h ahead of 33g's catalog slices. Companion
training is now derived from level by archetype growth weights plus an
optional focus (`company growth`), re-derived on every spawn and level
change so no level loop mints points; quests with `companyexperience` are
contracts paying every companion with the leader. `help growth`,
`help contracts`. See the [33h design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md)
("Final implementation decisions: 33h1") and
[plan](plans/2026-10-02-phase-33h1-growth-rewards.md). 33h2 (readiness and
recovery) is next, then 33h3 (relocation).

**33g management complete:** pooled treasury (owner choice), shared
instance-preserving cargo, explicit equipment assignment/removal/comparison,
claimed/public loot windows and opt-in autoloot. Equipment presets removed at
the owner's request. Durable equipment journals and cargo migration protect
restart recovery. Browser controls, indexed help, and tutorial hints updated.
See [design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md) and
[plan](plans/2026-10-01-phase-33g-equipment.md). Independent review and
required checks passed;
equipment catalog and tiers are not shipped by this slice.

**Equipment design approved:** [weapon families, six tiers, and armor paths](designs/2026-10-01-equipment-tiers-design.md) records
the owner's approved equipment direction for 33g
and later progression/content work. Design documentation is complete; gameplay
is not implemented, and the phase order below is unchanged. Documentation
verification: whitespace and relative Markdown links checked; no Go changes.

The expedition/company loop, onboarding, combat presentation (29a–29f),
status effects, wounds, tactics, guardians, interrupts, wind-ups, morale
and mercy (30a–30e), active defense (30g2), and personal load (30g3) are shipped. The Phase 32
play-test improvements are implemented; 32b's status still carries an outstanding review note (see its retained plan).

**Current owner priority:** finish Phase 33a–33i in order, choosing the lead's
recommended defaults without further confirmation (2026-10-01). 33a–33d are
complete; 33e (automatic class abilities) and **33f1 (skill and charm
retirement)**, **33f2 (expedition specialists)**, and **33f3 (camp
specialists)** are complete, so 33f is done; **33i1 (group assessment)**
is complete too (33i2 remains); 33g management and its cargo prerequisite are
complete, reviewed and verified. Equipment
catalog delivery and 33h remain. Phase 30g3 is also complete;
30g4 (progression), 30g5 (the action meter), 30g6a (combat fixes and harness) and 30g6 (stat edges and tuning) are complete.

**Combat tempo queue:** Phase 30g, [combat tempo, personal load, and active
defense](designs/2026-09-30-phase-30g-tempo-defense-design.md), whose
decisions the owner settled on 2026-09-30. 30g1 (the balance harness and
baseline), 30g2, 30g3, 30g4, and 30g5 are done; each later slice is measured against 30g1:

1. **30g2, active defense and armor:** complete (work log below): one
   defense per strike: block with a shield (no dodge), else parry a
   melee strike or dodge; the shield's ×1.5 armor removed; the shield
   bash moves to a blocked melee strike (5–20% by Strength); armor's
   suffix reads `absorbed`; `help defense`.
2. **30g3, personal load:** complete (work log below): worn and carried
   weight against a Strength-based capacity (cargo and mounts never
   count); burden lowers dodge only; burden words in `status`, `look`,
   `scout`, and the web Overview; `help burden`.
3. **30g4, progression:** complete (verification above): automatic stats grow in steps every 5 levels;
   HP by archetype, in small numbers; an XP knee at level 60.
4. **30g5, the action meter:** complete: turns from effective Speed and burden,
   one opening turn, at most two a round, no whole-turn banking.
5. **30g6, tuning:** HP, damage, and healing set so a no-focus even 5v5
   lasts 10–15 rounds at levels 1–60; the harness asserts it. It also
   takes 29f's cadence retune.

Phase 30e (morale and mercy) is complete. Phase 30f (battlefield
conditions; mounted combat remains excluded) is implemented on
`phase-30f-battlefield` and merged via [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12).
Other remaining limitations are listed below.

**Future company gameplay:** the owner endorsed the 2026-10-01 gameplay review
and requested [Phase 33a–33i designs](designs/2026-10-01-company-gameplay-roadmap.md):
command consistency, friendly-effect scopes, retreat, allied companies,
automatic class abilities, specialists, equipment/loot, progression/recovery/
relocation, and group assessment/coordinated enemies. These are planning
records now authorized for implementation by the owner, with open defaults
delegated to the lead. 33a–33e are complete.

## Deferred preparation designs

- Owner endorsed company UI/logistics improvements on 2026-10-01, including
  automatic recruit formation placement. [Phase 34 design](designs/2026-10-01-phase-34-company-ui-logistics-design.md)
  and [delivery plan](plans/2026-10-01-phase-34-company-ui-logistics-plan.md)
  split delivery into 34a UI/formation, 34b assigned packs and cargo capacity,
  34c equipment editor, and 34d effects/current capabilities. Design and delivery defaults approved by the owner;
  34a–34d implemented and verified as recorded above.

- Owner requested a [weapon poison design](designs/2026-10-01-weapon-poisons-design.md)
  for future implementation on 2026-10-01: shop-bought temporary blade coatings,
  explicit per-member/per-blade camp assignments, and later camp recipes.
  Recorded only; no phase scheduled or gameplay implemented. Balance numbers
  remain proposed defaults.

- Owner requested the [camp consumables design](designs/2026-10-01-camp-consumables-design.md)
  on 2026-10-01: fortifying broth, warming draught, cooling salve, scent-masking
  paste, watch incense, weapon oil and antidote draught. Includes proposed
  preparation limits, rest/route lifecycle, acquisition and later crafting.
  Documentation only; no phase scheduled or gameplay implemented.

## Planned zone encounters

- Owner requested [zone room encounters and wandering parties](designs/2026-10-01-random-room-encounters-design.md)
  on 2026-10-01, with an [implementation plan](plans/2026-10-01-random-room-encounters-plan.md).
  Encounter-enabled rooms roll on entry using zone enemy tables; random battles
  may become the main ordinary enemy source while public wandering parties remain.
  Includes proposed grace, multiplayer ownership, durable recovery and zone
  migration. Planned only; phase scheduling and proposed defaults remain open.

## Planned branching class progression

- Owner requested a [class progression review and branching design](designs/2026-10-01-branching-class-progression-design.md)
  with an [implementation plan](plans/2026-10-01-branching-class-progression-plan.md)
  on 2026-10-01. All five base lineages gain proposed advanced/elite paths,
  individual alignment eligibility and explicit player promotion choices.
  Includes current-mechanics audit, old-save migration, ability inheritance,
  death retention and 30g4/33h dependencies. Planning only; levels, gates,
  branch signatures and implementation scheduling remain proposed.

- Owner expanded the class direction with an [extended companion catalogue](designs/2026-10-01-expanded-companion-classes-design.md):
  twenty additional advanced/elite paths including Sorcerer, plus eventual
  nonhuman humanoids and creature recruits such as Hellhounds and Golems.
  Species, class and creature growth remain separate; staged delivery includes
  complete upkeep/recovery/gear rules. Planning only; external reference pages
  were blocked by network policy, so the proposed roster is not a verified
  reproduction of Ogre Battle 64 or Unicorn Overlord.

- Added [further class-reference extractions](designs/2026-10-01-class-reference-extractions.md)
  from the owner's supplied Ogre Battle/Unicorn Overlord roster: seven new
  humanoid candidate paths, commander/prestige direction, Cerberus, additional
  creature families and dragon affinities. Includes consolidation of equivalent
  names and explicit equipment/flight/form/aquatic dependencies. Planning only.

## Planned skills and spell progression review

- Owner requested a [skill/utility/spell progression review](designs/2026-10-01-skill-spell-progression-design.md)
  and [implementation plan](plans/2026-10-01-skill-spell-progression-plan.md)
  on 2026-10-01. Inventories twelve shipped skill definitions and eleven spell
  definitions; separates trained ranks, class/level unlocks, field/camp specialists
  and automatic combat. Preserves approved retirements and flags stale prose,
  level/class inheritance and the no-battle-items conflict in earlier antidote
  proposals. Proposed new milestones/capabilities only; no gameplay implemented.

## Phase progress

| Phase | Scope | Status |
|---|---|---|
| 0 | Fork and bootstrap GoMud baseline | Complete |
| 1 | GoMud integration map and handoff docs | Complete |
| 2 | Minimal company slice (one persistent companion) | Complete |
| 3 | Company roster (cap 5) + 3×3 formation state | Complete |
| 4 | Survival state (hunger/thirst/fatigue) | Complete |
| 5 | Terrain and travel profiles | Complete |
| 6 | Travel interruptions | Complete |
| 7 | Camping | Complete |
| 8 | Weather | Complete |
| 9 | Encumbrance and cargo | Complete |
| 10 | Mounts | Complete |
| 11a | Enemy parties | Complete |
| 11b | Unit-vs-unit engagement | Complete: target assignment, reassignment-on-death, and the proactive engagement trigger (pre-existing in `attack.go`, verified) all wired |
| 11c | Formation tactics | Complete: domain layer, schema, read-only query, all three player/mob attack directions wired, and the leader-as-interceptor gap closed |
| 11d | Guard reactions, crit effects, wounds, AI personality | Superseded by shipped Phases 30a–30c2 |
| 12a | Encounter-kind abstraction | Complete: `InterruptionKind` widened to `fallen-tree`/`discovery`/`tracks`, data-driven text lookup |
| 12b | Weighted encounter tables | Complete: `InterruptionProfile.Kinds` weighted-roll form, resolved once at fire time; no shipped route uses it yet |
| 12c | Combat encounters | Complete: `Combat` interruption kind spawns its route's `CombatMobID` into the origin room and commands it to attack; resume/return gated on the mob still being alive and present |
| 13 | Sky and environment | Complete: moon phases, cloud cover, fog, indoor biomes/tags, indoor glimpse, richer `weather`; display-only |
| 14 | Visibility and light | Complete: ambient vs per-viewer light, personal/fixture/party light, darkness hit penalty, `light` command, `floatinglight` spell |
| 15 | Temperature, clothing, exposure | Complete: `internal/climate`, `modules/exposure`, item `warmth`, weather `TemperatureMod`, survival member drain + mutex, `temperature` command |
| 16 | Walking fatigue, inns, travel/rest multipliers | Complete: `internal/walking`, `modules/walking`, inn stays in `modules/camping`, multipliers locked at departure/rest start, Waymark Inn, Old Kings Road zone |
| 17 | Archetypes (17a) and utility skills (17b) | Complete: `internal/archetypes`, `modules/archetype`, training and spell gating, companion archetypes, `autoskill`, `trap`, auto-light. Cooking shipped in Phase 18b |
| 18a | Loot tables | Complete: category tables, boot/reload loading, corpse/floor drops, proving content |
| 18b | Cooking | Complete: `cooking` skill + `cook` profession, per-recipe skill requirements on room containers, deterministic recipe choice, Waymark Inn hearth with three meals |
| 19 | Commodities and markets | Complete: `internal/market`, `modules/market`, `market` command, Dunmar and Old Kings Road markets |
| 19b | Market trading | Complete: `market buy`/`market sell` in tagged market rooms, buy/sell spread, Dunmar Market Square, Trappers' Post; inter-market profit kept (resolved 2026-09-24) |
| 20 | Trade rumours | Complete: `rumors` at inns, fuzzy hints from a persisted market news snapshot refreshed every 150 rounds |
| 21a | Company alignment | Complete: durable companion alignment and loyalty, −100..100 display, drift toward the rest of the company, desertion, recruit gate, `company inspect`/`alignment` |
| 21b | Settlement standing | Complete: `internal/standing`, `modules/standing`, `standing` command, market/inn markups and refusals, black markets, Tanner's Back Alley |
| 22a | Creation step and starter kits | Complete: archetype step in `start`, per-archetype kits, owed-kit record and character claim marker, ash quarterstaff |
| 22b | Durable companion level and gear | Complete: `MemberState` on each companion (level, experience, worn and carried items, gold), restore from the record, snapshot seams, `company gear` |
| 22c | Recruiters and `company recruit` | Complete: recruiter rooms in config, free-once tutorial and paid candidates, claims on the company record, Waymark Inn and Trappers' Post, mobs 61–64 |
| 23a | Rest tiers | Complete: camp Rested (buff 1033, strain 75%), exclusive with inn Well Rested, durable companion grants, buff 16 renamed Refreshed |
| 23b | Whetstones | Complete: 10-use whetstone (item 30) in both markets, `sharpen`/`camp sharpen` any time but not mid-fight, one use per member sharpened, durable per-weapon edge (+1 for 20 strikes) spent in combat, auto at camp rest end |
| 24 | Company chemistry | Complete: company-wide. Each member's durable service with the band; a band's tier from the average saved service of the members together (each capped at Sworn), so recruits dilute it; Familiar/Trusted/Sworn (900/2700/6300 rounds) give everyone in the band +2/+4/+6 hit in all four combat directions; `company chemistry`, `status bonuses` |
| 25a | Player death and church return | Complete: one level lost (no protection levels; peak level stops re-granted points), a durable pending mark so a death is charged once, wake at the last city's church (Dunmar's new Chapel of the Wayfarer, Frostfang's Sanctuary as fallback) with the living company; travel, camp, and inn stay abandoned first |
| 25b | Companion death and resurrection | Complete: a dead companion stays on the roster, keeping the gear its body kept; a 3-game-day rescue allowance spent only in the leader's online time; `resurrect` at a church or village shaman with its keeper costs a level; at zero it is lost and archived; Fernhollow village and Old Wenna |
| 26a | Company summary and text surfaces | Complete: `internal/companyview` read model; prompt tokens from a game-loop cache and a default prompt that warns only when needed; `status` as the Ashveil character sheet, grouped `conditions`, company load in `inventory`, last level lost in `experience` |
| 26b | Browser Company panel (GMCP) | Complete: `Company`/`Company.Vitals` GMCP from the 26a summary, sent on change and only to the leader; a Company section above Players in the web client's Party window, safe DOM, keyboard and screen-reader friendly, checked in Chromium |
| 27a | Tutorial framework and first lessons | Complete: `modules/tutorial` runs the course in per-player copies of rooms 900–903 (Waking Hall, Muster Yard, Drill Ground, Gate); progress in MiscData, resumed on login; gates check results; `tutorial`, `tutorial next`, `tutorial skip`; graduation cap once; old JS rooms removed |
| 27b | Tutorial: Survival and Camp lessons | Complete: Weather Yard (904) and Campground (905) before the Gate; Survival passes on a real meal and drink (`survival.OnProvision`) plus `weather`/`temperature`/`strain`/`cargo`, with food and water given once for what the pack lacks; Camp passes on Rested from a real camp rest; course camps struck (`camping.AbandonCamp`) on pass, skip, leave, logout, and placement |
| 27c | Tutorial: practice fight | Complete: Practice Yard (906) before the Gate; a squad of harmless straw soldiers (three footmen in front, an archer behind) per player; `practice` mobs beaten with no XP, drops, gold, kills, or `MobDeath` (`mobcommands.OnPracticeBeaten`); the gate is the squad beaten; death in the course decided (an ordinary death, ending the course as a skip) |
| 27d | Tutorial: Alignment lesson, browser panel | Complete: the Oath Stone (907) before the Gate; `company alignment`, `company inspect corvin` (an outlaw a new company is refused), and `standing`; `company inspect` weighs any recruiter's candidate; a `Tutorial` GMCP package and web client window from the same checklist as the terminal; a course missing rooms is closed but kept |
| 28 | Item weights and the company's whole load | Complete: every shipped item weighed (grams, per-type ranges), starter kits 4.8–8.5 kg; living companions' worn and carried gear in the company load (`company.CompanionGearGrams`), live mob when out, else record or template; `cargo` shows the split; GMCP `companion_g` |
| 29a | Combat fixes from the 5v5 simulation | Complete: a round-start engagement upkeep keeps an engaged company and enemy party fighting as a whole (the leader turns from an unreachable target, the killer and leader rejoin, the whole party joins, hostility can't lapse mid-fight); unplaced members can be struck; `break` holds; `formation reach` answers against the enemy in a fight; `internal/enemyparty` |
| 29b | Combat event stream and battle summary | Complete: `internal/combatstream` (one event per combat happening, fights of a company against the enemies it fights in a room, a summary folded from the events), producers at every attack, cast, target change, flee, and death; the summary at a fight's end (`set battlesummary`); interceptors fall in the round they're struck; player help for combat (`help combat` and seven pages), pointed to from the tutorial |
| 29b2 | One battle at a time; spawn groups | Complete: `internal/battle` (each player, with their company, fights one enemy group at a time; other groups set on them hold back, then begin the next battle in the order they turned; a waiting group turns on a free player); `attack`/`cast`/`backstab`/`shoot` refuse a waiting group; hostile spawns form groups of two to five from the room's list, a lone survivor or straggler regroups, `solitary` bosses stand alone; travel ambushes are a pair; solo players get battles and summaries; help for every Ashveil system |
| 29c | Narration voice (weapons and spells) | Complete (branch `master-6csfy6`): [design](designs/2026-09-28-phase-29c-narration-voice-design.md). Every weapon line rewritten, `(N damage)` / `(critical hit, N damage)` / `, M blocked` at the end of each hit, only real crits draw the critical pool; no `***`, `!`, caps, or "prepares to fight"; "the" before common names; an opener per fight, "turns toward", death lines in order, a closing line after the last; the fallen notice indented and in words; spells chant with their rounds and land with `(N damage)` / `(N healed)`; `help narration` |
| 29d | Pronouns and ordinals | Complete and integrated: [spec](designs/2026-09-26-combat-pronouns-ordinals-design.md). NPC/race pronouns, neutral generated recruits, stable shared enemy labels, help/tutorial; independent review and full race suite passed. Player selection deferred |
| 29e | Pain reactions | Complete: [design](designs/2026-09-29-phase-29e-pain-reactions-design.md). A victim reacts after each damaging critical strike that leaves them standing; second person for the victim, third person for witnesses, distinct beast-race sets and NPC overrides, without changing combat mechanics |
| 29f | Paced combat output | Complete: [design](designs/2026-09-29-phase-29f-paced-combat-design.md). Combat resolves every `CombatEveryRounds` (2) game rounds, an 8-second combat round; everything a round causes is tagged (`events.WithCause`) and paced out per player (`internal/combatpace`), with `set combatpace fast|normal|slow|off` (off for screen readers); a pause before pain and death lines; typed output, tells, and says never wait; the prompt and the web client's vitals and battle view wait for the lines; flushes on move, quit, pace change, and copyover; `help combatpace` |
| 30a | Status effects and critical-hit effects (weapons only) | Complete (merged as `2ef931c`): [design](designs/2026-09-29-phase-30a-status-crit-effects-design.md). Nine statuses (bleeding, staggered, knocked down, armor broken, exposed, burning, overloaded, stunned, hobbled) as combat-round buffs (`internal/status`); crit effects by weapon subtype; ticked, and actions lost, in the combat round; cleared at fight end; Sparks overloads; `help statuses` |
| 30b | Wounds, treatment, and `heal wounds` | Complete: [design](designs/2026-09-30-phase-30b-wounds-design.md). Lasting wounds from damaging crits, light ones from crushing blows and finished bleeds; healing stops at the wound limit; `heal`/`heal wounds` (clerics' `tend` and heal, splints and bandages, the Waymark Inn physician); inn stay heals, camp rest one wound per splint or bandage (owner amendment); death clears; `status`, GMCP, web strip; `help wounds` |
| 30c | Company tactics | 30c1 complete: [design](designs/2026-09-29-phase-30c-company-tactics-design.md). `company tactics` (`tactics`): a durable company focus (none, leader, casters, nearest, weakest, strongest, wounded) and healing threshold (10–90%); in a battle only the focus changes, for that battle, one order a round, turning everyone at the next upkeep; web focus buttons; the `casters` rule; enemy personalities by race or template (`targeting`, `targetingnoise`); `help tactics`. 30c2 complete: [design](designs/2026-09-30-phase-30c2-guardian-design.md). A `guardian` role (`strategy <who> guard [other]`) steps in for its ward (set, else the most hurt in reach) within one column: 2 guards a battle, one back per 2 combat rounds, none while knocked down or stunned; `help guardian`. Rotate the wounded deferred |
| 30d | Wind-ups, telegraphs, and interrupts | 30d1 complete: [design](designs/2026-09-30-phase-30d1-chant-interrupts-design.md). A weapon blow that draws blood breaks a chant (owner: no pressure meter); 30d1b ([design](designs/2026-09-30-phase-30d1b-chant-break-chance-design.md)) made that a chance (40–90% by damage against max health; a crit, stagger, knockdown, or stun always); a company caster loses the spell with half its mana back, an enemy restarts from the first word; shield counters on a missed melee blow (50%, 1d4, stun 25%, once a round); the goblin hexer and Withering Hex in the Dark Forest; `help interrupts`. 30d2 complete: [design](designs/2026-09-30-phase-30d2-windups-design.md). Physical wind-ups (`internal/windup`, mob `windups`): the forest ogre (Dark Forest room 530) winds up Crushing Blow in plain view, then one swing of double damage that knocks down; only a crit that lands, a stagger, a knockdown, or a stun breaks it (owner); a broken or landed one is followed by 2 turns' cooldown; a shield bash is a counter strike only and breaks nothing (owner) |
| 30e | Morale and mercy | Complete: enemy temperaments and shared break checks; protected surrender; paced mercy decisions, rewards and reactions; companion hesitation, flight and saved return |
| 30f | Battlefield conditions | Complete: [PR #12](https://github.com/Robinsond76/ashveil-gomud/pull/12), reviewed and merged 2026-10-02. [Approved design](designs/2026-10-02-phase-30f-battlefield-design.md), [verification](verification/phase-30f/verification.md). Mounted combat excluded |
| 30g | Combat tempo, personal load, and active defense | In progress: [design](designs/2026-09-30-phase-30g-tempo-defense-design.md). 30g1 complete ([plan](plans/2026-09-30-phase-30g1-balance-harness.md)): the balance harness and baseline (no-focus 5v5: median 8 / 32 / 68 rounds at levels 1 / 5 / 10). 30g2 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g2-active-defense.md)): block, parry, or dodge, one per strike; the shield's ×1.5 removed; the bash on a blocked melee strike (5–20%); `absorbed`; `help defense`. 30g3 complete, merged 2026-10-01 ([plan](plans/2026-10-01-phase-30g3-personal-load.md)): personal load and burden (dodge × (1 − 0.6 b)); `help burden`. 30g4 progression and 30g5 action meter complete; 30g6a fixes and harness ([verification](plans/2026-10-04-phase-30g6a-verification.md)); 30g6 stat edges and tuning, acceptance passing ([amendment](designs/2026-10-04-phase-30g6-amendment.md), [verification](plans/2026-10-04-phase-30g6-verification.md)) |
| 32a | Company polish | Complete (PR from `claude/project-thread-1buera`): [spec](designs/2026-09-28-phase-32a-company-polish-design.md). No `♥friend` on companions; one arrival/departure line per company; no drink flourish; camp and fire in `look`; recruiters listed in the room; a readable formation grid |
| 32a2 | Per-player recruit rosters | Complete (PR #4 from `claude/project-thread-btmgj8`): [spec](designs/2026-09-28-phase-32a2-recruit-rosters-design.md). Generated candidates on each player's own notice, coming and going; companions get their own names |
| 32b | Tutorial replay | Complete, in review: [spec](designs/2026-09-28-phase-32b-tutorial-replay-design.md), [plan](plans/2026-09-28-phase-32b-tutorial-replay.md). `tutorial replay yes` hands the connection to a throwaway level-1 copy (id from 900,000,000, unindexed) that runs the course; any way out hands it back to the real character, exactly as it was; `UserPurged` drops the copy from every module and removes its file; a restart sweeps leftovers |
| 32c | Enemy groups and `scout` | Complete: [design](designs/2026-09-28-phase-32c-enemy-groups-design.md). Groups named as they form ("a band of ruffians") and shown on their own room line; `attack <group>` is the only way to start a fight; a battle plays out on its own (attack, cast, backstab, shoot, tackle, disarm refused in one; a bare `attack` after `break` rejoins); `look <group>` and a free `scout`; summaries name the group |
| 32e | Company experience | Complete: a kill that pays the leader pays every living companion still charmed by them in the kill room the same figure in full (no split), in both the solo and party branches; companions level up live (spending the level's points, so a respawn changes nothing); `experience` lists each companion; `company.MemberView`/`companyview.Member` carry live level and progress. **Verification (2026-09-28):** `go test -race ./...`, `make generate`, `make validate` green. **Review:** Opus reviewer found no blocking bugs. Fixed: a levelled companion kept unspent points until respawn (now `AutoTrain`); a companion charmed away was still paid (now requires `IsCharmed(leader)`); the design contradicted the code on rows for absent companions; dead `Leader` progress fields removed; party-branch and befriended-away regression tests added. Rejected/deferred: full seam test of logout/copyover (the snapshot/restore round-trip is tested, the seams are 22b's), a practice-mob companion test (mobs vanish before any XP), mid-fight level-up refill (accepted). A rare failure of the combat brawl test under `-count=4` was traced to a level-up resetting the test's forced health; the brawl now runs with kill XP off. |
| 32f | Company logistics | Complete: [design](designs/2026-09-28-phase-32f-company-logistics-design.md). Capacity from members (20 kg + Strength), one pack each, and horses; weight the only limit (a full company takes on nothing more; walking never blocked); a herd of riding and pack horses bought at stables, with saddles; cargo keeps uses; `company inventory`; `company eat`/`drink`/`meal` |
| 32h | Character deletion | Complete: [design](designs/2026-09-28-phase-32h-character-deletion-design.md). `delete character`, confirmed by the password (masked) and the name; every module's state purged with the login kept; back in creation on the same connection; a durable `Deleting` flag and a boot sweep; masked in-game password prompts (also `password`) |
| 32d | Automatic combat by strategy | Complete: [design](designs/2026-09-28-phase-32d-auto-combat-design.md). `strategy`: each character's role (fighter, healer, caster) and target rule (weakest, strongest, wounded, nearest, furthest, leader, assist, defend), durable; healers and casters cast real spells with mana; companions know spells by archetype and level and regain mana; wizards/clerics granted Magic Missile/Minor Heal; in a battle only `flee`; only hostile mobs group by tag |
| 32g | Web company dock | Complete: [design](designs/2026-09-29-phase-32g-company-dock-design.md). Left column the world (time, map, room, tutorial); right a tabbed dock: a vitals strip for every member above Character (Overview with worth, Gear with weights, Skills and jobs, Quests, Effects, Pet), Company (Status, Inventory with menus by exact item reference, Camp), Combat (setup: roles, targets, formation, Scout), Comm (unread count), and Who/Kills when enabled; `Company.Inventory`, `Company.Camp`, members' mana and strategies; `help webclient` |
| 32g2 | Live battle view | Complete: [design](designs/2026-09-29-phase-32g2-battle-view-design.md). During a battle the Combat tab shows the enemy group's formation above the company's (fronts to the middle), scout's health words (never numbers), reach, target lines both ways, outsiders struck, the fallen, the waiting groups, a text list, a live region, Flee, and Setup's member menu; a marker on the tab; `Company.Battle` for every player in a battle; dark rooms show nothing, as scout |
| 33a | Company Command Rules and Legacy Action Routes | Complete: [design](designs/2026-10-01-phase-33a-company-command-rules-design.md), [plan](plans/2026-10-01-phase-33a-company-command-rules.md); safe ask aliases, queued ownership checks, battle/script/skill policy |
| 33b | Friendly Effects and Company Membership | Complete: [design](designs/2026-10-01-phase-33b-friendly-effect-scopes-design.md), [plan](plans/2026-10-01-phase-33b-friendly-effects.md); shared target snapshots, eligibility, ownership and callback revalidation |
| 33c | Company Retreat, Rout, and Separation | Complete: [design](designs/2026-10-01-phase-33c-company-retreat-design.md), [plan](plans/2026-10-01-phase-33c-company-retreat.md); ordered withdrawal, paid cover, legal escape and saved separation |
| 33d | Multiplayer Parties and Allied Companies | Complete: [design](designs/2026-10-01-phase-33d-allied-companies-design.md), [plan](plans/2026-10-01-phase-33d-allied-companies.md); durable alliances, explicit consent, contribution-based XP, fixed shared loot claims and independent company authority |
| 33e | Automatic Class Abilities and Combat Roles | Complete: [design](designs/2026-10-01-phase-33e-automatic-class-abilities-design.md), [plan](plans/2026-10-01-phase-33e-class-abilities.md); automatic Tackle, Opening Strike and Aimed Shot, coordinated healing, mana reserve, `strategy [who] abilities on\|off` and `reserve`, `help abilities` |
| 33f | Company Specialists and Expedition Skills | [Design](designs/2026-10-01-phase-33f-company-specialists-design.md) rewritten with the owner (2026-10-01) in three slices. 33f1 complete ([plan](plans/2026-10-01-phase-33f1-skill-retirement.md)): retired peep, portal, tame, change form, scribe, sneak, bump, pickpocket, pray, and backstab; mercenary hiring, mob befriend, and the charm scripting API; one-time training-point refund; Protection capped at 3. 33f2 complete ([plan](plans/2026-10-01-phase-33f2-expedition-specialists.md)): Read the Trail, Keen Eye, Pathfinder, Weather Sense, Haggle, `company specialists`; `search`, stock `track`, and `trading` retired. 33f3 complete ([plan](plans/2026-10-01-phase-33f3-camp-specialists.md)): camp raids and Camp Watch, Field Smith, Vigil, Forage, `camp cook` |
| 33g | Company Equipment and Loot | Management/cargo/treasury complete; reviewed and verified. [Design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md); presets removed by owner. Catalog delivery remains a separate slice. |
| 33h | Company Progression, Rewards, and Expedition Continuity | [Design](designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md) in three slices. 33h1 complete ([plan](plans/2026-10-02-phase-33h1-growth-rewards.md)): derived archetype growth, `company growth` focus, contract quests; `help growth`, `help contracts`. 33h2 complete ([plan](plans/2026-10-02-phase-33h2-readiness-recovery.md)): durable companion vitals, online-only recovery, inn restore, half-vitals resurrection; `help readiness`. 33h3 complete ([plan](plans/2026-10-02-phase-33h3-relocation.md)): relocation and separation; `help separation` |
| 33i | Company Encounter Assessment and Enemy Roles | [Design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md) in two slices. 33i1 complete ([plan](plans/2026-10-01-phase-33i1-company-assessment.md)): the company's assessment of a visible enemy group in `scout [group]`, `consider [enemy]` (one-on-one odds retired), and the Battle view's outlook; `help assessment`. 33i2 complete 2026-10-02: coordination tiers, enemy roles, light enemy wounds and recovery; `help coordination` |
| 12+ | Merchant/injured-NPC/route-choice/camp-opportunity/ruined-site/resource/social encounters | Future ideas, not planned work |

## Recent work log

### 38b faith routes design approved (2026-10-05)

- **Why:** the draft had four points the owner had not decided.
- **Decided:** a character below its elite gate at level 30 waits on its
  advanced route with its ranks and promotes once its alignment recovers;
  the Demon's broken binding stays; Blood Priest healing keeps its 25% mana
  surcharge; routes are final at launch, with retraining a later design.
  Recorded in the [faith routes design](designs/2026-10-05-faith-routes-design.md).
  Docs only; 38b still waits on 38a.

### Phase 33h2: company readiness and recovery (2026-10-02)

- **Why:** logout, restart, copyover, a crash, a level-up, and resurrection
  all refilled a companion for free, and companions never regained health
  on their own. The lead explained the plan and defaults in plain language
  first, as the owner asked; the owner approved them unchanged.
- **Delivered:** `MemberState.Vitals` on the 22b snapshot seams, resolved
  at every spawn by `Vitals.Resolve` (points clamped to the wound limit and
  mana maximum, never below 1; nil, an old record, spawns full once;
  `Percent` for resurrection's half). Online health regeneration for
  companions beside 32d's mana (`internal/hooks`), counted by exposure; a
  live level-up keeps a companion's vitals; an inn's Well Rested grant
  restores the leader and live companions. New `help readiness`; company,
  health, inn, camp, resurrect, wounds, heal, and quit pages updated; rest
  lesson and Departure hints.
- **Independent full-diff review:** no blockers and no remaining free
  refill path. Accepted and fixed: the `recover` alias collided with the
  brawling topic, so `help recover` resolved at random (removed; regression
  test plus a guard that no help alias names two topics); a level-up
  lowered health above a fresh wound's limit (now never lowers; regression
  test); help wording on when a lowered limit applies, the crash window
  (up to about a quarter of an hour), the inn sentence's placement, and an
  over-long line; the resurrection share's comment now says it is fixed
  apart from death's `RespawnVitalsPct`; coverage added for a morale-flight
  return and a snapshot taken at 0 health. Accepted as known risks, not
  changed: an inn restore reaches disk only at the next save seam, so a
  crash in between loses it (the same window as gear, wounds, and the
  leader's own file); and a Well Rested grant handled before a returning
  leader's companions respawn would miss their restore (plausible, not
  reproduced; the same ordering the 30b wound knitting already has).
  Not added: separate tests for the in-battle regen skip (shared with 32d's
  tested mana path) and the stale-mob replacement branch (unchanged 22b
  code; it now carries vitals through the same `Snapshot` call the logout
  and save tests exercise).
- **Verification (after review fixes):** `make generate` (no diff),
  `make validate`, and `go test -race ./...` (95 packages) passed with no
  failures. No JavaScript or Lua changed, so those lints were not
  applicable.
- **Integration:** owner approved merge and push. `origin/master` had moved
  to 34a/34b (`eb21ae3`); merged it into the branch (status log conflict
  only), then `make generate` (no diff), `make validate`, and
  `go test -race ./...` (95 packages) passed again before merging.

### Phase 33h1: companion growth and contracts (2026-10-02)

- **Why:** owner requested 33h next; open defaults chosen by the lead under
  the 2026-10-01 standing authority and recorded in the design.
- **Delivered:** `domain.Deal` (prefix-stable weighted rotation) over
  per-archetype `Growth` weights in the archetype config; training derived
  from `characters.StatPointsAtLevel` on spawn, live level-up
  (`company.RetrainCompanion` replaces `AutoTrain` for tracked companions),
  `company archetype`, and `company growth [member] [stat|balanced]`
  (+2 focus, in memory until the next save seam, refused in battle).
  `rewards.companyexperience` makes a quest a contract; Rodric's Rats and
  The King's Shadow ship as contracts. Also fixed the 32e award retraining
  every later companion once any companion levelled. No save migration:
  training is never saved.
- **Independent full-diff review:** no blocking findings. Accepted and
  fixed: `company growth` saved the company file outside the 22b seams
  (now in memory only, regression test); missing real-entry coverage for
  `company archetype` retraining and resurrection weights (tests added);
  contract message and help ignored the experience scale (now say "before
  scaling"); member names with spaces refused (stat is the last word);
  `company growth` missing from the company page's command list; "pays
  once" overstated (reworded). Rejected: none.
- **Verification (after review fixes):** `make generate`, `make validate`,
  and `go test -race ./...` passed with no failures. No JavaScript or Lua
  changed, so those lints were not applicable.

### Phase 33g: cargo, treasury, equipment, and loot management (2026-10-01)

- **Integration:** owner authorized merging the reviewed management slice
  (`cf8434f1`) to `master` and pushing to `origin`; no gameplay changes after
  the final verification below.
- **Owner choices:** pool carried gold into one treasury; exclude equipment
  presets. Completed the required shared cargo prerequisite and deliberate
  equipment management on `phase-33g-equipment`, based on origin/master
  `022100ca`. [Design](designs/2026-10-01-phase-33g-company-equipment-loot-design.md)
  and [plan](plans/2026-10-01-phase-33g-equipment.md). Equipment tiers/catalog
  remain separate content delivery; 33g is not closed
  as a whole by this management slice.
- **Delivered:** instance-preserving shared cargo and pooled treasury, exact
  member/item equipment assignment, removal and comparison, recoverable
  company/user asset operations, idempotent legacy cargo migration, physical
  pack counting, `loot` and opt-in `autoloot`, two-hour private/two-hour public
  spoils, and battle/ownership checks. Updated real item/meal/cooking/camp
  consumers, text and browser inventories, help and tutorial pointers.
- **Independent full-diff review:** accepted and fixed pending-recovery incoming
  asset loss, missing leader GMCP inventory updates, trade-in pack-capacity
  loss, remaining name-based browser actions, and stale help/corpse claims.
  Added shared cooking/meal coverage and a real saved-user cargo reload check.
  Lead additionally fixed autoloot collecting foreign public spoils, stale
  exact remove references, migration rollback markers, and spawn UUID identity.
  Confirmed findings have regressions; no findings rejected. Follow-up reviews
  found no production blockers. Cooking branch tests use a provider double;
  atomic cargo replacement and the equipment/GMCP path use real implementations.
- **Verification:** `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint JSHINT=/workspace/ashveil-env/js/node_modules/.bin/jshint`, and
  the Chromium dock-window harness passed. Full-suite testing caught old mount
  assertions for private inventory; updated them while preserving burden,
  physical pack, saddle, meal, saved-cargo, reload and no-clock-advance checks.
  Relative Markdown links and whitespace checked. No Lua changed.
- **Retained limit:** uncollected runtime corpses disappear on restart, as the
  engine already did; equipment and collected cargo retain their exact instances.

### Phase 33i1: company encounter assessment (2026-10-01)

**Why:** the owner asked for the early 33i slice, in parallel with 33f3:
players couldn't judge whether their company could take a group; the old
`consider` compared the leader alone with one creature
([design](designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md),
"Final implementation decisions: 33i1"; [plan](plans/2026-10-01-phase-33i1-company-assessment.md)).

**What shipped:** `internal/assessment`, a read model: each counted
member's dice-free expected damage (`combat.ExpectedDamage`) against the
visible foes it can reach (formation reach, unplaced fails open), current
health as staying power; five words (an easy fight … hopeless) and "it
could go either way" near even or a band line; never numbers. Counted:
the leader and companions with them who can fight; fled, fallen,
awaiting, separated, down, and out-of-the-fight members named as not with
you; pets left out; party allies noted, not counted. `scout [group]` ends
with it; `consider [enemy]` gives it for the enemy's whole group (the
owner retired the one-on-one odds) and refuses players, own companions,
and harmless loners; `Company.Battle.outlook` and the web Battle view's
"Outlook" line. Visibility as scout: hidden foes left out, nothing in the
dark. No state, RNG, or world-time change; no durable state, so nothing to
migrate. Help `assessment` (new), `consider` (template replacing GoMud's
page), `scout`, `combat`, `webclient`; the Combat lesson's scout hint.

**Tests:** pure estimate tables (bands, closeness, reach, fail-open, no
one able to strike, no fighters); `ExpectedDamage` draws no dice (seeded
global source unchanged); real `scout`/`consider` in the company brawl
(counted and missing members, wounds, burden, a separated companion,
formation reach agreeing with scout's marks, hidden foes, darkness,
allies, refusals, no round/state change); the live `Company.Battle`
outlook and its absence in the dark; payload test; help render and
aliases; tutorial pointers; dock-windows browser check.

**Independent review** (default model, report-only). Accepted and fixed
with regression tests: (1, major) a company with nobody able to fight read
"a hard fight, could go either way": now hopeless; (2) members standing
here but down or withdrawn were labelled "away": now named with the
reason, the leader too; (3) `consider` by name could land on a hidden
namesake and miss the visible one: it matches visible group members only;
(5) `consider` assessed shopkeepers and other harmless loners: refused, as
scout doesn't list them; (6) an own companion now gets "travels with you";
(7) "Allied companies" reworded "Allies". Accepted as is: (4) reach uses
scout's alive map, so an unseen front-rank foe can put the one behind out
of reach (the design's rule, matching scout's marks); (9) the outlook is
recomputed on each Battle refresh (≤5×9 dice-free estimates on the game
loop; acceptable). Rejected: (8) remove `combat.CombatOdds`: kept as
engine API for upstream parity. Pre-existing, not changed: `scout`'s and
`attack`'s lone-mob lookup (`enemyparty.FindGroup`) can still land on a
hidden namesake.

**Checks:** after the review fixes, `make generate`, `make validate`,
`go test -race ./...` (all passing), and `make js-lint` once; the
dock-windows Chromium check (all passing, outlook included).

### Phase 33f3: camp specialists (2026-10-01)

**Why:** the owner asked for the camp skills from the 33f review and
approved the proposed raids and the simple camp cooking
([design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f3-camp-specialists.md)).

**What shipped:** camp raids (15% per Old Kings Road rest, rolled at rest
start and saved; a pair of new level-3 road brigands, mob 86, through the
shared `enemyparty.SpawnAmbush`); Camp Watch (warrior, 25%/level): spotted,
the rest still counts; unspotted or with the leader logged out, no Rested
and no rewards; raiders left standing when their leader falls or leaves
are sent off. Field Smith (warrior): 20 + 5/level strikes of edge per
whetstone use. Vigil (cleric): +level loyalty to present companions, cap
60. Forage (ranger): 1 + level/2 finds into cargo, within capacity.
Forage and Vigil come at most once per 15 minutes, are collected at the
camp after any fight, and are paid exactly once through operation IDs
saved with the cargo (`Cargo.Applied`) and the company record
(`Record.AppliedOps`). `camp cook`: the hearth's three dishes from pack and
cargo, into cargo. Help `campwatch`, `fieldsmith`, `vigil`, `forage`;
`camp`, `cooking`, `sharpen`, `specialists`, `autoskill`, `combat`
updated; a Camp lesson hint.

**Tests:** raids planned, durable, resolved once through the round pass
(unspotted, spotted, logged out, away, failed spawn), restart; raiders sent
off when the leader falls; rewards through rest completion and the round
pass (once on retry, after a fight, cooldown, forfeited away, capacity);
`camp cook` through the camp command (skill gates, battle refusal,
restore on a failed withdrawal, capacity); Field Smith through `sharpen`;
cargo and loyalty operations in their modules, including the real YAML
decoders; camping registry YAML round trip; shipped config; help render.

**Independent review** (default model, report-only; a first run was cut
off by an interrupted compaction and rerun). Accepted and fixed: (1,
blocking) the company decoder dropped `AppliedOps`, so a vigil could be
paid twice after a restart; (2, blocking) rewards owed while the leader
still fought a spotted raid were dropped: they now wait; (3) free rests
farmed rewards: once per 15 minutes; (4) raiders outlived their target and
could turn on other campers: sent off; (5) logging out dodged raids: it
now spoils the rest; (6) `camp cook` could lose ingredients on a failed
save and skipped capacity: cargo first with restore, capacity checked,
pack-full fallback; (7) rewards used the leader's room at payment: now
the camp's, forfeited elsewhere; (8) the watch was looked up under the
camping lock: moved out; (9) a failed spawn still spoiled the rest; (11)
"food and water" and the `help combat` link; (12) coverage gaps. Rejected:
(10) a retried forage message may name a re-rolled find (the deposit
itself is deduplicated; cosmetic, rare); goimports grouping (gofmt is the
repo's check). Known, unchanged: travel ambush raiders share finding 4's
leftover-mob behaviour (33c/33h scope).

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green). No JavaScript changed.

### Phase 33f2: expedition specialists (2026-10-01)

**Why:** the owner asked for the expedition skills proposed in the 33f
review ([design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f2-expedition-specialists.md)).

**What shipped:** for each capability the best present, living member of
the leader's own company acts (the 17b resolver, through
`archetypes.BestSpecialist`), switched by `autoskill`, resting in the
leader's battle. Read the Trail (ranger): reports hostile groups beyond
the exits on each step and on a bare `track`, more detail per level, two
rooms out at 4 (memory-loaded rooms only); warns of ambush routes at
departure and leads the company around an ambush 20%/level (the pause
becomes tracks). Keen Eye (rogue): spots secret exits on entry, remembered
in `KnownSecretExits` and shown in room text, Room GMCP, the world map, and
`map`. Pathfinder (ranger): 5%/level less strain on rough ground, never
below road. Weather Sense (wizard): zones foretell their next condition
(`ZoneWeather.Next`), so `weather` forecasts truly, more per level.
Haggle (rogue): 2%/level better market prices; every sale stays below the
cheapest possible buy-back. `company specialists`. The `search` command and
skill, the stock `track` command, and the dead `trading` skill (and the
Merchant profession) are retired with the 33f1 refund. Help pages
`specialists`, `trail`, `keeneye`, `pathfinder`, `forecast`, `haggle`;
updated company, weather, market, strain, autoskill, archetype, travel,
and job pages; a Departure hint.

**Tests:** real `go` steps (trail by a companion ranger and a level-1
player, no tracker, autoskill off; Keen Eye spotting, memory, room details;
no spotting without a rogue at a low roll); `track`, `specialists`,
`company specialists`; resolver ownership (separated, downed, other player,
switched off, in battle); walking strain and `strain`; market buys, sales,
listing, and a sweep of every shipped good, stock, standing, and haggle
pairing that no round trip profits; weather command at each level,
neighbour zones, foretold weather on advance and on recovery of old saves;
travel warning, evasion, and resume/return/recovery after it; Room GMCP;
`KnownSecretExits` save/reload; refunds; help render and index.

**Independent review** (default model, report-only). Accepted and fixed:
(1, blocking) a haggled buy-and-sell loop profited on steep goods (and by
toggling the haggler): the sale cap now always applies at the best
possible haggle; (2, blocking) an evaded ambush left a pause the route
validation rejected, stranding the journey: `CanFireAs` accepts it (and,
pre-existing, a weighted table's kinds); (3) specialists acted in the
leader's battle: gated in the resolver, `track`, and Keen Eye; (4) route
config never read `Kinds` or the ambush mob: parsed and tested (no shipped
route has an ambush yet: a content gap, not a code one); (5) coverage for
durability, Room GMCP, and refunds added; (6) the world map and `map`
still used the visited-only rule: fixed; (7) trail now reads only rooms
already in memory; (8) neighbour forecasts use weather zones; (9)
`autoskill` alignment; (10) the stale trading page (the skill is retired),
README, and unused trail colours; (11) trade notes name the gold actually
won. Not changed: Room GMCP needs no forced resend, since Keen Eye runs in
the step before the move's queued room update is built.

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green). No JavaScript changed.

### Phase 33f1: skill and charm retirement (2026-10-01)

**Why:** the owner reviewed GoMud's stock skills against a game about
leading a company on expeditions and asked to remove those built for a
lone player or that clash with Ashveil, and all player-facing charm
([33f design](designs/2026-10-01-phase-33f-company-specialists-design.md),
[plan](plans/2026-10-01-phase-33f1-skill-retirement.md)).

**What shipped:**
- Player commands `peep`, `portal`, `tame`, `changeform`, `scribe`,
  `sneak`, `bump`, `pickpocket`, `pray`, and `backstab` are gone; their
  skills (peep, portal, tame, change form, scribe) are deleted; Protection
  stops at level 3 (level 4 granted only pray). Mob-side sneak, portal, and
  backstab stay.
- Charm: no taming (skill, spell, tame mastery, stat mod), no mercenaries
  for hire (shops skip `mobid` stock; the Bonecrafter's skeleton is gone),
  no mob `befriend`, and the scripting API's charm setters and tame calls
  are removed. Companions keep the internal charm link. Charms are
  runtime-only, so none survive the deploy; pets stay.
- A player still holding a retired skill (or Protection 4) gets the
  training points back once, at their next spawn, saved with the change.
- Archetypes (rogue, wizard, ranger), professions (Explorer and Monster
  Hunter removed), trainers (rooms 830, 160), the Whispering Wastes
  obelisk, the long whip, the shipped admin user, help pages, keywords,
  and the admin scripting reference follow. `search` and `track` stay
  until 33f2 replaces them.

**Tests:** refund through the real spawn listener, once and saved; no
archetype claims a retired skill; retired commands and help topics gone;
a mercenary shop neither lists nor sells (fails on the old code); a
world-data guard (trainers, professions, item stat mods, quests,
scripts); every indexed help topic and alias opens a page (pre-existing
GoMud gaps named). The 29d combat capture was regenerated: removing the
tame-learning roll on each kill shifts the seeded RNG (a dummy roll
reproduces the old capture exactly).

**Independent review** (default model, report-only). Accepted and fixed:
(1) backstab needed `sneak`'s hidden state, so it was retired too
(Opening Strike covers battles); (2) `hire` still indexed under shops;
(3) the skulduggery description named pockets; (4) Protection 4 granted
nothing: capped at 3 with refund (skulduggery's levels still drive traps
and utilities, so no refund); (5) stale admin scripting docs and
AGENTS notes; (6) dead mercenary listing code, `canCarryStolen`, and the
`backstab`/`pickpocket` autocomplete; (8) the admin user's tame mastery
block and a level-0 entry's empty refund notice. Rejected: (7) syncing
`world/empty` (unshipped upstream seed world, never loaded by Ashveil or
its tests; its retired pages are inert); the `MercHirePricePerLevel`
config key and its admin docs stay (engine config, now unused).

**Verification (after review fixes):** `make generate`, `make validate`, `go test -race ./...` (all green), `make js-lint`, and `scripts/browser/dock-windows-check.mjs` in Chromium over localhost HTTP (all dock window checks passed).

### Phase 33e: automatic class abilities (2026-10-01)

- **What:** warriors tackle (knocked down; breaks a foe's chant or
  wind-up; the whole turn), rogues strike an opening on a downed,
  stunned, staggered or exposed foe, and rangers aim a shot (both the
  round's own swing, its first landed blow a critical hit), each on its
  own by archetype (companions) or trained skill (players: brawling,
  skulduggery, track), with cooldowns of 4/2/3 combat rounds. Healers no
  longer double-heal one ally; `strategy [who] reserve [percent]` keeps
  mana back from attack spells; `strategy [who] abilities on|off`. Durable
  fields `no_abilities`/`reserve` (old saves load as on/0); cooldowns are
  runtime only. GMCP `Company` strategies and the web Combat setup show
  abilities. `help abilities` (new) and updated `strategy`, `combat`,
  `company`, `tactics`, `archetype`, `brawling`, `skulduggery`, `track`;
  a practice-fight tutorial hint. The backstab crit no longer reports a
  crit on a round with no landed blow, and its shouted prefix is gone.
  No ability uses an item (owner rule). Decisions: the design's "Final
  implementation decisions"; balance is left to 30g6.
- **Why:** owner priority to finish 33a–33i in order with the lead's
  recommended defaults.
- **Independent review** (report-only subagent) — accepted and fixed, each
  with a regression test: (1) abilities skipped the formation gate, so a
  reaching warrior could tackle a back-row caster the front row shields
  (now the swing's interception rules apply, and a tackle needs
  hand-to-hand reach); (2) companions still used abilities while the
  company prepared to retreat (now none do); (3) a tackle skipped the
  tackler's cancel-on-combat buff removal; (5) help said "rest 4 rounds"
  where the ability is ready every 4th round (now "at most once every 4
  combat rounds"); (6) the tutorial hint promised a "sure" crit; (8) added
  tests for interception, retreat, a casting member, and a readied strike
  set back, and removed an unused test seam. (4) a cooldown spent when the
  foe falls first is accepted and now documented. (7) `help warrior` and
  `help ranger` are GoMud job pages, not archetype pages: rejected as
  stated; `help archetype` carries the change and the design text was
  corrected. Ownership negatives (charms, allies, other companies) are
  structural (the side list is the leader's own companions) and were
  not given separate tests.
- **Verification:** after the review fixes, `make generate`, `make validate`
  and `go test -race ./...` passed (94 packages ok); `make js-lint` passed
  on the final `window-combat.js`; the Playwright dock check
  (`scripts/browser/dock-windows-check.mjs`, system Chromium, localhost)
  passed with the new ability checks. The new integration tests passed 20
  repeated runs. No Lua changed.

### 33d owner review follow-up (2026-10-01)

- **Owner review of 33d** found, and the owner approved fixing, with a
  regression test each (all fail on the old code):
  - A last enemy killed by a round-tick or immediate buff (poison) paid
    no XP: its queued death ran after the same round's battle pass ended
    the battle. Contributors are now frozen at those death sites too
    (not for revive-on-death mobs).
  - 33d dropped `get`'s CorpseItems gate, so in the default world anyone
    looted the gear a body keeps (failed drop rolls, perma-gear). Only a
    claimed corpse is a loot source there, never its worn gear; an
    unclaimed corpse name no longer shadows a floor item.
  - Claimed loot could be buried by anyone or any mob, and never decayed
    with corpses disabled. `bury` now refuses another player's claimed
    loot; claimed corpses decay regardless. `RemoveCorpse` no longer
    confuses look-alike corpses (one group, one round) and deleted a
    claimed twin. `get`, `look` and `bury` prefer the right same-named corpse.
  - Owner decision: outside one alliance the loot claim goes to the most
    damage (companions count; ties to the lowest id); allies keep the
    rotating claim.
  - `party leave` hands leadership to an online member when there is one;
    messages name players instead of `player #id`; `alliances.json` is
    gitignored. `help party` and `help bury` updated and render-tested.
  - The intermittent `TestBystanderCannotHelpAnotherPlayersBattle` was a
    test bug (its enemy patient could die in the opening round); fixed.
- **Independent review** of the fix diff: accepted a get shadowing bug,
  first-corpse-only `bury`/`look`, and an unrendered `bury` help test; all
  fixed with tests (it also exposed the `RemoveCorpse` twin bug). Rejected:
  `look` reading an offline claimant's saved name from disk, accepted as a
  rare, player-driven read during a claim's ~2½ minutes.
- **Owner decisions for a future cargo phase (before 33g):** no personal
  inventory besides worn equipment; everything carried, companions' carried
  items included, lives in company cargo, and spoils go there (`loot`,
  optional `autoloot`, reserved to the winners for 2 game hours, open for
  2 more, then gone). No item may be used during a fight.
- Verification: `make generate`, `make validate` and `go test -race ./...`
  passed after the review fixes; focused package runs passed throughout.
  No JavaScript or Lua changed, so their lint was not run.

### 33c owner review follow-up (2026-10-01)

- **Owner decision (`phase-33c-one-retreat`):** one way out of a fight.
  Review found emergency `flee` beat ordered retreat: it rolled only against
  foes striking the leader (a back-row leader always escaped), and a pinned
  companion was "separated" and rejoined the same round, because the flight
  itself ended the battle. The owner chose: `flee` is another name for
  `retreat [exit]` (wimpy and the web Battle view's button, now "Retreat",
  issue the same order); no separation on withdrawal (a pinned leader or
  member holds the company and is named); every active battle foe pursues;
  the preparation round stays. 30e morale flight remains the only separation.
  The emergency-flee path and `flee.md` are gone (`help flee` is `help
  retreat`); retreat also leaves a fight with another player.
- **Bug found while testing, fixed:** a retreat order carried the leader's old
  target, so that target dying in the preparation round (mob death clears
  every aim at it) silently cancelled the withdrawal; 33c had shipped with it.
  The aim to resume now lives in the runtime `RetreatInfo`.
- **Independent review:** no high findings. Accepted and fixed: wimpy
  re-ordered (and repeated refusals) on every hit, now once a round and not
  while withdrawing; missing wimpy and player-fight round tests (added); "held
  by magic" in `help retreat` (no such status) and missing "players pursue";
  `set wimpy` threshold wording ("below"); a dying leader told their legs were
  pinned; dead `holdsAgainstPlayer` check; a defender kept an empty aim after
  a failed attempt; stale "flee" in `help travel`. Kept: the leftover player
  `Flee` aggro branch, as a harmless guard (nothing sets it).
- **Help:** `retreat`, `combat`, `targeting`, `break`, `statuses`,
  `webclient`, `set-wimpy`, `travel`, `company`, `guardian`, `morale` (retreat
  notes above "See also"), aliases, and the tutorial hint.
- **Checks:** new tests (flee is the order and every battle foe pursues;
  pinned member holds the company with no loyalty loss or separation; wimpy
  once; player fight; target falling mid-order) fail on the old code and pass.
  `make generate`, `make validate`, `go test -race ./...`, `make js-lint` and
  the browser dock check passed. One intermittent failure of
  `TestSecondPlayerTakesTheNextGroup` (no retreat involved) appeared once in
  an early package run and did not recur in 39 later runs, isolated or whole
  package, on this branch or master; noted, not explained. After merging
  33d into the branch, the full race suite was rerun: everything passed except
  upstream `TestAttemptConversation_UsesPluginFile`, which fails about 2% of
  runs (4/200 measured): `getConversation`'s random 2% maintenance purges
  conversations older than 10 rounds, untouched by 33c or 33d; it passed on
  rerun. Left for an upstream-code fix.

### Phase 33d: allied companies completed (2026-10-01)

- Pulled master before implementation (`4b4db05e`); re-fetched and integrated
  latest `854fea80` (33a script authority and 33b battle-boundary fixes)
  before final checks.
- Durable version-1 alliance membership/leadership; atomic replacement and
  full registry rollback on failure; missing-file migration and corrupt startup
  refusal. Logout retains membership, resets consent, promotes online successors;
  character purge retries rather than proceeding after a failed alliance save.
- Independent company grids, commands, focus, assets and retreat. Accepted
  membership grants no autoattack/follow/support; queued orders revalidate tokens.
  Leadership changes revoke follow/autoattack. Recovery restores no consent.
- One enemy XP pool shared only by actual, eligible contributing companies;
  living attached companions receive their owner's share. Death-time battle
  snapshot survives final-enemy battle closure; payout rechecks presence/life.
  Shared ordinary loot/gold has one fixed corpse claimant until corpse decay.
  Legacy ranks are display only. One primary evaluates shared enemy morale and
  owns mercy; existing durable mercy recovery and runtime enemy-life guard remain.
- Full login GMCP refresh includes ownership/consent/offline members; browser
  distinguishes alliance leadership and company command. Indexed party/help
  aliases, protection correction, combat/company/support hubs and tutorial pointer.
- Independent full-phase reviewer found final-enemy payout loss, incomplete
  solo-leave rollback, login structural refresh gap, claimed-corpse inspection
  and missing integration coverage. All fixed with regressions and re-reviewed;
  no remaining blockers. Final-enemy regression narrowed to its own XP/claim.
- Tests cover two actual companies, shared battle/formation/focus/retreat and
  morale-owner departure, separate/idle/distant/fallen/withdrawn reward exclusion,
  duplicate death/loot pickup, delayed World autoattack in both queue branches,
  recovery/save rollback and purge retry, and GMCP recovery payloads.
- Ordinary battles/enemies/corpses remain runtime-only; no replay journal or
  atomic cross-character payout transaction is introduced. Existing character
  and company saves and durable mercy tokens retain their established seams.
- Verification: `make generate`, `make validate`, final integrated
  `go test -race ./...`, JavaScript and Lua lint, Chromium dock-window harness
  (including alliance authority/offline status), and `git diff --check` passed.
  Focused checks passed before the final suite. Earlier fixture checks exposed
  XP-disabled combat setup and an overly broad final-enemy assertion; corrected
  both before the final integrated run. No remaining failed checks.
- The final 33b merge requires mutual consent and overlapping live enemies
  to broaden battle participation for allied support; separate battles and idle
  allies remain blocked. Real cast-start/battle-entry/completion regression covers
  both ally owners and companions, including revoked consent. Manual battle casts
  remain prohibited; the fixture starts its chant before entering battle.
- Full-phase and both master-integration independent re-reviews found no remaining
  blockers. Owner explicitly authorized merge to master and push when confident.

### Phase 33d: initial alliance consent slice (historical) (2026-10-01)

- Pulled origin master `4b4db05e` and confirmed 33d was not implemented;
  isolated branch/worktree `phase-33d-allied-companies` retains this initial unit.
- Reused runtime parties and 33b's helpful-target resolver: accepted members
  opt into following and mutual allied/area support; joining grants neither.
  Company-only spells and companion command/formation/asset authority stay per
  owner. Autoattack consent rejects invitations; leaving/kicking/disbanding
  prune consent. Promotion resets following; logout reports the successor.
- Queued follows capture current leader, origin and a unique consent token,
  checked at actual execution after requeue. Off/on, leaving/rejoining,
  recreated parties, movement, promotion and active prompts invalidate them.
- Shipped indexed `help party`, aliases, company/combat/support links and a
  Departure hint. Tests exercise actual party commands, ordinary movement,
  production hook registration, cast completion and logout/recovery paths.
- Independent review accepted and fixed delayed-follow authority, production
  provider coverage, successor feedback and contradictory design status;
  re-review found solo logout notification, also fixed with regression coverage.
  No rejected findings or unresolved blockers for this initial unit.
- Verification passed: touched-package tests; `go test -race .
  ./internal/hooks -count=1`; `make generate`; `make validate`; final
  `go test -race ./...`; `git diff --check`. Initial race/test failures in the
  new fixture (redundant logger reset and capturing unrelated queued input)
  were corrected before the green final suite. No JS/Lua source changed.
- **At the end of this initial slice, 33d was incomplete and unmerged.** Party state remained
  runtime-only. Encounter participation/XP, deterministic loot, durable alliance
  recovery and rank replacement remain the next tasks in the plan.
### 33b owner review follow-up (2026-10-01)

- **Owner review of 33b:** the owner kept one 33b choice. Minor Heal All
  reaches only the caster's own company, not GoMud party members; 33d brings
  cross-company help back with consent.
- **Battle boundary (`phase-33b-review`):** a bystander could heal or aid
  either side of another player's battle (e.g. keep that player's bandits
  alive). `effecttargets.OtherBattle` now refuses a named patient fighting in a
  battle the caster isn't part of (its foes, its player, and the companions or
  charmed pets fighting for that player, as the battle counts them). `cast`
  and `aid` refuse at start; `Resolve` drops such a patient at completion.
  Group and area help stay narrowed to the caster's company, not refused.
- **Silent chants:** a helpful chant that lost every patient ended in silence.
  It now ends with "Your spell (or aid) finds no one left to help" and a
  `CastComplete` with the `wasted` outcome; the mana stays spent.
- **Help:** 33b paragraphs moved above the see-also lines of `cast`, `spells`
  and `heal`; `friendly-effects`, `protection` and the tutorial hint describe
  the boundary and the fade.
- **Independent review:** no high findings. Accepted and fixed: area help was
  refused beside another battle (pre-check now only for named patients);
  charmed pets ignored (now counted like the battle does); per-round battle
  cloning (cheap `battle.Involving`/`battle.RoomOf`); "spell" wording for
  first aid; refused aid counted as a skill use; weak tests; a ragged help
  line. Recorded, not changed: once 33d installs allied consent,
  `OtherBattle` would drop an ally fighting the same group in their own
  battle; 33d must count a caster sharing that enemy group as part of it.
- **Checks:** the new tests cover each entry point (`cast`, `aid`, chant
  completion, area help, pets); the probes that found the bugs ran against the
  old code. `make generate`, `make validate`, `go test -race ./...` and
  `git diff --check` passed. No JS or Lua changed, so no lint was needed.

### 33a owner review follow-up (2026-10-01)

- **Owner review of 33a:** the owner kept two 33a choices. Players can't get, drop, give or put items during their own battle. Aid is refused during the aider's battle.
- **Help fix:** the 33a–33c sections of `help company` had been added after its "See also" line. That line is last again.
- **Script fix (`fix/33a-company-help`):** 33a treated a follower's own script replies (onAsk/onGive/item/room scripts) as orders from whoever's command triggered them.
  - Another player's follower therefore went silent and showed "You no longer command that member." The owner's follower could not whisper, emote or leave.
  - Script commands on a follower now count as its owner's (`MemberOrder.Scripted`). They are still refused, silently, if they would fight, or change gear during a battle, or if the charm or membership behind them has changed. They skip `ask`'s order list and the room checks.
  - Asking your own follower about a subject reaches its onAsk script again. An unanswered subject points to `help ask`.
  - `help ask` updated.
- **Independent review:** nothing blocking. All six findings accepted:
  - Another player's trigger could make a follower's script attack. Closed by attributing follower scripts to the owner.
  - Delayed replies were dropped after the follower moved. Room check removed for script commands.
  - Typed non-orders lost the pointer to `company`. Restored.
  - Missing tests for battle gear, emote shortcuts, hostile aliases, moving and typed attacks. Added in `modules/company/wiring_order_scripts_test.go`.
  - Inaccurate help wording. Fixed.
  - Status not yet recorded. Done here.
- **Checks:** the new tests fail on the old code and pass now. `make generate`, `make validate` and `go test -race ./...` passed. No JS or Lua files changed, so lint was not needed.

### Phase 33c: company retreat (2026-10-01)

- Ordered `retreat [exit]`: one preparation round, one escape attempt, slowest
  member's current speed/burden/lasting wounds against fastest active pursuer.
  An eligible guardian spends a guard and preparation attack for cover. Stable
  ownership and legal exits revalidate each round; blocked living members hold
  the company, while dead members retain resurrection and morale flights retain
  30e recovery. Only captured live members move; runtime orders never save.
- Emergency flee remains individual escape, with 30e's saved separation for
  blocked living companions. Failed separation save holds the leader; debt is
  paid once on rejoin. End battle only after successful movement; respect
  no-go, effective locks and room permissions. Departed casts/aims clear without
  ending another company's battle. No cloned gear, cargo or herd; no clock jump.
- Text, Company.Battle countdown and browser view; indexed retreat help,
  corrected legacy flee/attack/targeting/break help and tutorial.
- Independent full-diff review found no implementation blockers but six coverage
  gaps. Added real checks for waiting groups, another active company battle,
  late arrivals, departed spell targets, GMCP gather countdown and relocation
  rollback. Reviewer rechecked all six; no remaining blockers, no rejections.
  Stabilized the chant regression by making test blows miss before pruning.
- Verification: focused touched packages and new coverage repeated ten times;
  full browser dock harness with system Chromium/local HTTP passed. Final
  `make generate`, `make validate`, `go test -race ./...`, installed JSHint via
  `make js-lint`, `make lua-lint`, and `git diff --check` all passed.
- Next: 33d. Cloud session creation unavailable; saved handoff, continue here.

### Phase 33b: friendly effects (2026-10-01)

- Shared resolver for player, mob, automatic and script cast paths. Group help
  includes present attached company members; single help retains explicit
  patients. Runtime snapshots only prune, checking rooms, life, stable company
  identity and charm identity/expiry. Temporary followers stay outside group
  effects; allied expansion waits for 33d consent. Healing allows downed living
  players above -10, never revives fallen companions, and retains wound caps.
- Validated scope metadata and shipped spell mappings; costs and durations
  preserved. Indexed friendly-effects help, hub links, tutorial and render tests.
- Independent full-diff review accepted two P2 findings: HelpArea lacked actor
  arrays, and the void callback fix changed onCast's proceed default. Fixed both
  with actual JS/Lua cast regressions; other successful void handlers no longer
  report a missing handler. Reviewer rechecked against latest master (30g3),
  including actual round completion/pruned healing narration and stream; no
  remaining blockers. No findings rejected.
- Verification: focused spell/script/usercommand/company/tutorial tests;
  make generate, make validate, full go test -race ./..., installed JSHint via
  make js-lint, make lua-lint, and git diff --check. Full checks recorded after
  final review fixes. No save migration: cast snapshots and Aggro are runtime.
- Next: 33c. No cloud-session creation tool is available; continuation context
  is in the phase 33 session handoff and work continues in this session.

### Phase 33a: company command rules (2026-10-01)

- Safe `ask` management aliases for living present companions and temporary
  followers; all requested follower hostility is refused. Automatic battles,
  native NPC/lifecycle commands, and administrative input remain distinct.
- Queued orders carry requester, room, stable member key and runtime charm
  identity; execution rechecks battle, presence, ownership, dismissal, death,
  expiry and transfer-back. No persistent fields or migrations were added.
- The player dispatcher checks battle restrictions before scripts, including
  personal/global aliases; aid/tame cannot replace battle actions. Skill
  execution rechecks new battles and protected/owned tame targets.
- Indexed `help ask`, updated company/combat/equipment/consumable/skill pages,
  tutorial pointer and help rendering tests shipped with the implementation.
- Independent review: accepted and fixed both findings (multiword alias
  interception bypass and obsolete aggressive-taming help). Added requested
  aid/battle-owned target/temporary follower coverage. Reviewer rechecked the
  fixes and regressions: no remaining blockers; no findings rejected.
- Verification: focused dispatcher/queue/script/lifecycle tests; `make
  generate`, `make validate`, `go test -race ./...`, `make js-lint` using the
  installed JSHint binary, and `make lua-lint` all passed. `git diff --check`
  passed. Native follow regression caught during implementation was fixed
  by separating `CommandRequested` from autonomous `Command`.
- Next: 33b, company-only friendly-effect correctness first, then allied
  scopes once 33d defines consent. The current tooling has no cloud-session
  creation operation; continue here using the saved phase handoff.


### Company gameplay roadmap and future Phase 33 designs (2026-10-01)

- **What/why:** recorded all nine owner-endorsed review recommendations as
  [33a–33i future designs](designs/2026-10-01-company-gameplay-roadmap.md),
  with code evidence, dependencies, owner decisions, state/recovery rules,
  player help, and real integration acceptance requirements. The guides
  are Ogre Battle for composition/automatic roles and Mount & Blade II for
  readiness/specialists/company control. Updated the design index and combat
  roadmap; corrected its stale next-phase instruction to 30g2.
- **Baseline:** fetched origin master at `626df427`, incorporating shipped
  30e before planning retreat/return extensions. No gameplay changed and
  no future Phase 33 implementation is claimed.
- **Review:** independent default-agent reviewer checked the complete
  documentation diff and relevant code; no actionable findings, accepted
  or rejected findings, or unresolved review blockers.
- **Verification:** 74 local Markdown links and 29 explicit repository
  path references resolve; all nine designs contain scope, evidence,
  ownership/recovery, dependencies, decisions, verification, and help
  requirements. Staged whitespace checks pass. No gameplay tests run for
  this documentation-only change.

### Phase 30e: morale and mercy (2026-10-01)

- Shipped original-group morale checks, race/template temperaments, protected
  surrender and fleeing, and stable shared ownership across player battles.
  Mercy questions wait for the paced summary, expire after 30 seconds, and
  release unresolved prisoners on departure. Spare/execution use exact ±5
  alignment, witnessed loyalty reactions, and execution's ordinary rewards.
- Shipped one-check companion nerve, one-action hesitation and chant refunds,
  durable flight/gear/return with one loyalty penalty, and zero-loyalty departure.
  Mercy recovery saves effect receipts without restoring prisoners or replaying
  rewards; offline retries load the latest saved character.
- Added GMCP/browser presentation, indexed morale/mercy help and tutorial hints.
  [Implementation record](plans/2026-09-30-phase-30e-implementation.md).
- **Review:** independent full-diff review and follow-up found failed-save
  prisoner abandonment, shared-owner transfer, same-round ownership ordering,
  zero-loyalty departure, and stale offline character saves. Fixed all with
  regression coverage; final reviewer found no remaining blocker. The pacing
  concern was withdrawn after checking queued delivery and its regression.
- **Integration:** preserved incoming Phase 30g1 balance-harness work when
  merging `origin/master`; independent integration review found no blocker.
  Generation, validation and the full race suite pass on the combined tree.
- **Verification:** `make generate`, `make validate`, `go test -race ./...`,
  focused mobcommands/company regressions, JS/Lua lint and Chromium dock checks
  all pass. JSHint used the installed executable instead of fetching with npx. Browser checks used the unchanged harness over local HTTP because this
  environment blocks file URLs. The first full race run exposed death-guard
  compatibility failures; the guard now uses a transient per-instance marker.

### Phase 30g1: balance harness and baseline (2026-09-30)

- **What:** the first slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-09-30-phase-30g1-balance-harness.md)),
  test-only (no player-visible change, so no help).
  - `modules/company/balance_test.go`: an even 5v5 through the real
    `DoCombat` with real dice and 30a's statuses: Aria and four
    companions against a mirror group with their kits, both levelled
    alike (`levelTo`). `TestBalance5v5` (gated by `ASHVEIL_BALANCE=1`,
    `ASHVEIL_BALANCE_FIGHTS`, default 50) logs a table over levels 1/5/10
    × company modes spread/default/focus × enemy modes spread/default.
  - Always on: `TestBalanceHarnessRunsAFight`, `TestBalanceSidesStayEven`,
    `TestBalanceStatusesLand`, `TestBalanceTally`,
    `TestBalancePercentile`.
  - The design gained the owner's decisions 10–18 (shield bash on a
    block, no level cap and an XP knee, small archetype HP, stats in
    steps of 5 levels, the spread-out fight as the baseline, parry,
    `absorbed`) and six slices; its 30g1 section is amended to the build.
- **Why:** decision 9: measure before changing combat, so each later 30g
  slice is judged against numbers.
- **Baseline** (the full table is in the plan): the no-focus fight
  (spread × spread) runs a median of **8 rounds at level 1, 32 at level
  5, 68 at level 10** (p90 13 / 46 / 92), against the 10–15 target.
  Fights lengthen about 4× by level 5 and 8.5× by level 10, because HP
  grows with level and weapon damage doesn't: the owner's concern,
  measured. Other readings:
  - The company wins 60% / 66% / 84% of even fights at levels 1 / 5 /
    10: its healer, first strike, and slightly more HP for the player
    (recorded asymmetries).
  - Fighters take a turn in about three rounds of four (0.72–0.85 turns
    per standing fighter per round: casting, lost actions, and falls).
    About 38% of turns land (dodges count as misses), and about 15% of
    those are crits.
  - Shipped targeting (everyone the weakest) is already focus fire; an
    enemy that focuses makes fights longer and closer (level 10: 80
    rounds and 54% against 68 and 84%), not shorter.
- **Verification:** the harness tests looped 3 times while fixing; each
  review fix's test was checked to fail without it
  (`TestBalanceSidesStayEven`, `TestBalanceStatusesLand`). Final
  (2026-09-30, after the review fixes): `make generate` (no diff),
  `make validate`, and `go test -race ./...` all passed, and passed again
  after merging `master` (30d2 and the docs reorganisation) into the
  branch. The gated table runs in about 2–3 minutes at 50 fights a cell.
- **Review:** the independent default-agent reviewer checked fairness,
  the tally, the leader clamp, the real paths, and the invariants (clock,
  leaks, concurrency: all fine); each finding was checked.
  - **Fixed, with regression tests:** template experience levelled
    Garrick and Ysolde on their first kill at level 1 (`levelTo` now
    sets experience and peak level; `TestBalanceSidesStayEven`, which
    failed before the fix, also pins the player HP quirk); shield bashes
    were counted as turns and hits (now counted apart; `TestBalanceTally`);
    status-tick damage wasn't counted (now for the other side). The lead
    found, following the review's tick finding, that no status ever
    landed in the brawl world (no buffs loaded, no `Buff` listener): the
    harness now loads 30a's statuses and the game's listener
    (`TestBalanceStatusesLand`, which fails without it).
  - **Amended in the docs:** the baseline cell is spread × spread (the
    shipped default already focuses); comments on enemy spread and
    regeneration; the plan's modes, names, and cells; the design's 30g1
    section (unseeded, 50 fights, stalls at 200, per-side counts).
  - **Recorded, not fixed (they are the game as it is; 30g6 weighs
    them):** no enemy healer; the players' pass strikes first; enemies
    can't be wounded; a player's `HealthMax` `Base: 1` gives a little
    more HP than a mob; both sides unplaced, so formation is inert here.

### Phase 30g3: personal load and agility (2026-10-01)

- **What:** the third slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-10-01-phase-30g3-personal-load.md)), on
  `claude/phase-30g3-load`. `Character.PersonalGrams` (worn and carried,
  moved from the peep panel) against an agility capacity of
  `AgilityBaseKg` 15 + `AgilityStrengthKg` 0.5 × Strength (new `Combat`
  keys, with `AgilityFreeLoad` 0.35, in `config.yaml` and the admin
  wizard) gives a burden 0–1; dodge becomes dodge × (1 − 0.6 b) in the
  strike loop and the weapon rankings' estimate. Parry and block are
  untouched; the parry-or-dodge choice compares parry with the burdened
  dodge. Cargo, packs' bonus, and mounts never count. Words (unburdened,
  lightly burdened, burdened, heavily burdened) in `status` (Vitals),
  `look` at any character, `scout`/`look` of a group, and the web
  Overview (`Char.Inventory.Backpack.Summary.burden`). Nothing saved.
  Help: new `help burden` (indexed under combat, aliases), `defense`,
  `perception`, `strength`, `cargo`, `encumbrance`, `scout`, `status`,
  `look`, and the combat hub updated; a Combat-lesson hint.
- **Why:** owner decisions 2 and 3.
- **Balance** (30g1 table, 50 fights a cell): no-focus median **8 / 35 /
  77** rounds at levels 1 / 5 / 10 (30g2: 9 / 38 / 71). Every harness
  fighter is unburdened (kits 0.4–4.6 kg, under the 5.25 kg+ free share),
  so 30g3 cannot move these fights: the differences are run-to-run noise.
- **Calibration, for the owner (kept as proposed; 30g6 tunes):** Strength
  stays small (0 at level 1, 2–3 at 5, 5–8 at 10, 10–16 at 20), so
  capacity is 15–23 kg and the free share 5.3–8 kg. A new character's
  starter kit (5.4–9 kg with food, water, and a satchel) starts lightly
  burdened (cleric, rogue, warrior, wizard barely) and the ranger's
  burdened (b ≈ 0.39, dodge −23%); real armor (a breastplate and an iron
  shield, 15 kg) leaves anyone heavily burdened. The design hoped "a light
  kit costs nothing". The company's travel `Load:` also uses the words
  Burdened/Unburdened; the help separates the two.
- **Review:** the independent default-agent reviewer found nothing
  blocking; it checked the math, every dodge caller, the character
  copies, game-loop safety, statistical bounds, hit-forcing helpers, help,
  the web client, and the admin pages. Each finding was verified.
  - **Fixed:** `cargo put`/`take` raised no `ItemOwnership`, so the web
    Overview's burden (and 32g's weight) went stale until another event
    (`TestCargoMovesRaiseItemOwnership`); an unvalidated test config with
    `AgilityFreeLoad` ≥ 1 gave NaN, and the defender silently never dodged
    (now no free share, and NaN burden is 0; regression cases in
    `TestBurdenForUnvalidatedFreeShare` and `TestBurdenedDodge`); help
    hard-coded "0.35 is about a third" beside the configured value, and
    the Strength page now reads the per-Strength key; the plan's alias
    list matched the shipped one.
  - **Rejected/deferred:** scout's Burdened line names duplicates by name
    only (the grid does the same; `look [group]` puts the word inline);
    `look` shows burden for shopkeepers and horses too (harmless, one
    rule for every character); no test looks at another player (same
    helper as a mob) or drives GMCP through a live event (the summary
    path and the event are tested apart); the armor rankings don't weigh
    burden (for 30g6).
  - **Found while reviewing:** `TestAClericCompanionHealsTheHurt`
    (`modules/company`) failed once in a full run and passed 40/40 alone
    on this branch and on master; every brawl member is unburdened, so it
    is an existing flake, not this phase's.
- **Verification** (2026-10-01, after the review fixes and merging
  `origin/master` with Phase 33a): `make generate` (no diff), `make
  validate`, `make js-lint`, and `go test -race ./...` pass (go test's own
  exit status, 92 packages ok, no races). While working, `-count=3` on
  `internal/combat`, `internal/characters`, the new brawl tests, and the
  mount/cargo and encumbrance tests; each new combat test was checked to
  fail with the burden change removed. The web Overview was not exercised
  in a browser (JSHint and the GMCP payload test only).

### Phase 30g2: active defense and armor (2026-10-01)

- **What:** the second slice of Phase 30g
  ([design](designs/2026-09-30-phase-30g-tempo-defense-design.md),
  [plan](plans/2026-10-01-phase-30g2-active-defense.md)), on
  `claude/phase-30g2-defense`. A blow that would hit meets one active
  defense: a shield-bearer blocks any strike (no dodge fallback); with a
  weapon and no shield, a melee strike meets the higher of parry and
  dodge, rolled once; shots, claws, and bare hands meet a dodge; a stunned
  fighter gets none. Block: 15% + the shield's armor + the Strength share,
  held to 15–45% (wooden 20%, iron 25% at even Strength). Parry: Speed,
  5–30%, moved by the weapon (swords, reach weapons, and the staff's new
  item `parry` +5; daggers −5). The shield's ×1.5 is gone from
  `GetDefense` and the armor rankings. The bash follows a blocked melee
  strike, 5–20% by Strength (1d4, stun 25%, once a round, unchanged).
  Attack events carry `Defenses`; the battle summary has a `Defenses`
  line; the armor suffix reads `absorbed`; the 30g1 harness counts
  defenses. Player help: new `help defense`, `help armor` rewritten and
  indexed, nine pages updated, the Combat lesson's hints corrected.
- **Why:** owner decisions 4–6, 10, and 15–17.
- **A first attempt was not fit to merge.** It gave daggers +5 parry (the
  design says −5), left the ×1.5 in `armor_rank.go`, emitted no defense
  events, left help and the tutorial describing the old miss counter, and
  its status entry here called the phase complete while ten
  `modules/company` counter tests failed (a `go test ... | tail` exit code
  was read as the test result). The owner asked for a review of the work;
  it was rebuilt as the plan records.
- **Balance** (30g1 table, 50 fights a cell; the 30g1 baseline re-run on
  the same machine for comparison): the no-focus fight's median is now
  9 / 38 / 71 rounds at levels 1 / 5 / 10, against 9 / 32 / 67 before.
  Shield bashes fall from 4–14 a fight to under 0.3. Per fight at level
  10, each side makes about 3–6 blocks, 2–2.5 parries, and 2 dodges.
  The company's no-focus wins at level 1 fell from 58% to 48% (within the
  noise of 50 fights). Defenses lengthen fights slightly; 30g6 tunes
  against the 10–15 round target.
- **Review:** the independent default-agent reviewer found no blocking
  bug and confirmed the defense choice, numbers, strike loop, bash sites,
  stream, summary, and help. Each finding was verified.
  - **Fixed:** a parry range of 0 still let a sword parry 5% (the weapon's
    modifier came after the clamp), so hit-forcing tests could flake
    (`TestZeroParryRangeTurnsParryOff`); `TestShieldCounterOnlyOnBlock`
    and the stunned-bearer case now require the blow they test; the new
    keys added to `config.yaml` and the config wizard; admin armor pages,
    the items API page, the rankings help, and the empty world's armor
    page no longer describe the ×1.5; the Stunned buff and `help combat`
    name parry; stale comments; `defend` removed from the defense
    aliases (it is a strategy rule); a defended shot from another room
    now gets a line in the shooter's room.
  - **Rejected/deferred:** a reach weapon gets +5 one-handed or not (the
    design said two-handed polearms; none ship yet); a defended shot
    still reads "blow" (generic wording, kept).
- **Found while testing, recorded:** a bandit given a sling in the brawl
  world never attacks (the old bow-counter test passed without a blow);
  the rebuilt test uses the company ranger's shot. Not investigated
  further. `TestJoinsTheFight` (`internal/hooks`) fails under
  `-count=3` on the 30g1 base too: a test-isolation issue, not this
  phase's.
- **Verification** (2026-10-01, after the review fixes): `make generate` (no diff), `make validate`, `make js-lint`, and `go test -race ./...` pass (the go test exit status itself, 91 packages ok, no races). No Lua changed. While working: `go test -count=3` on `internal/combat` and the counter, status, wind-up, and pain tests of `modules/company`; the reviewer ran `internal/combat` 20 times. After merging `origin/master` (Phase 30e) into the branch: `make generate` (no diff), `make validate`, `make js-lint`, and `go test -race ./...` pass again (go test's own exit status).

### Documentation maintenance and Phase 30e preparation (2026-09-30)

- Removed mounted combat from the 30f proposal at the owner's request.
- Removed completed implementation checklists, superseded initial combat
  proposals, the unused legacy workflow redirect, and stale bootstrap/history
  text. Retained current designs, unresolved 32b review material, known issues,
  and operating constraints. Prior files and detailed verification/review
  history are recoverable from commit `792455ea`.
- Removed the project's plugin enablement and mandatory skill workflow;
  retained designs now live in `docs/designs`, plans in `docs/plans`.
- Phase 30e remains in design, not implemented. The [next-task handoff](plans/2026-09-30-phase-30e-handoff.md)
  records context and the owner's requested 6.1 Sol model.
- **Review:** independent reviewer found a stale 11d deferred-work bullet;
  removed it because Phases 30a–30c2 shipped those mechanics. No other blocking
  findings. **Verification:** local documentation links resolve; staged diff
  whitespace checks pass; all eight changed Go files are comment-path edits
  only. No gameplay tests run for this documentation-only change.

### Latest gameplay verification (historical)

Phase 30d2's implementation, review fixes, and verification were recorded
before this cleanup; consult `git show 792455ea:docs/PROJECT_STATUS.md` for
those results. This documentation change does not rerun or supersede them.

## Known issues / deferred items

- **Other players on the map (owner, 2026-10-05):** not shown for now,
  apart from your own party members. Revisit later, alongside camp
  visibility.
- **Race and gender sprite variants (owner, 2026-10-05):** not for now.
  Revisit with the races review.
- **Camp theft (owner, 2026-10-05):** a future camp event. A company
  resting **without camp bells and trip lines** may wake to find some loot
  and supplies missing, with no fight and no warning. Details are in the
  [40a3 camp gear](designs/2026-10-05-phase-40a3-camp-gear-design.md)
  design.
- **PvP camp visibility (owner, 2026-10-05):** the visual map shows only
  your own camp and allied camps. Revisit for PvP: visibility should depend
  on lighting, terrain, and the company's skill at concealing its camp.
  See the [visual client milestone](designs/2026-10-05-visual-client-milestone-design.md).
- **Broken chants and counters (30d1, 30g2), for the owner:** the
  counter's numbers (5–20% of blocked melee blows by Strength since 30g2,
  1d4, a stun one time in four, once a round per bearer) are a
  recommendation to tune. A bash leaves no
  wound; an enemy restarts its chant for ever while hit (no mana spent); a
  mob that runs out of mana wastes the turns it picks `cast` (GoMud's
  combat commands). No casting on the battle view's grid yet (32g2).
- **Chance to break a chant (30d1b), for the owner:** the numbers (40% +
  2 per percent of max health, held to 40–90%) are the lead's choice, to
  tune. A frail caster's floor is rarely 40% (a 30-health healer hit for
  3 is at 60%). A shield bash breaks no chant (owner, 2026-09-30, with
  30d2: a counter strike only). A weapon's own
  crit statuses still land on a crit the armor took (30a), so they make
  that round's other blows heavy.

- **Balance baseline (30g1), recorded for 30g6:** the harness's even
  5v5 keeps the game's asymmetries: no enemy healer, the players' pass
  strikes first, enemies take no wounds, a player's `HealthMax` starts
  at `Base: 1` (a mob's at 0, so a player gains a little more HP per
  level), and both sides fight unplaced (formation inert). The table
  runs only with `ASHVEIL_BALANCE=1`.

- **Wind-ups (30d2), for the owner:** the numbers (35% of the ogre's
  turns, x2 damage, 2 turns' cooldown) are the recommendation to tune.
  Company wind-ups, the proposal's sweep across the front line, and other
  abilities are deferred; a shooter can't wind up. Ordinary blows on a
  wind-up are silent (owner), so the summary counts only broken ones; a
  wind-up lost to a stun no blow dealt is credited to no one. Wind-ups
  aren't shown on the battle view's grid (32g2).

- **Company tactics (30c1), accepted:** an order given in the round its
  battle ends is dropped silently; a solo player has no web tactics row
  (the command works); enemy `wounded` weighs full health, not the wound
  limit; hunting the isolated (adjacency) and rotate-the-wounded are
  deferred. (The `casters` focus now has enemy casters: 30d1's hexer.)

- **Wounds (30b), deferred:**
  - Enemies take no wounds, and there are no monster abilities that
    wound.
  - No wounds on the battle-view grid or in the battle summary, and no
    stream event for them.
  - Pets' strikes don't wound.
  - A companion not out when a rest completes keeps its wounds.
  - The physician's price is flat, with no standing markup.
  - Healing reaches disk at the next save seam.

- **Paced combat (29f), for the owner and Phase 30:**
  - Game-round systems now tick twice per combat round:
    - regeneration heals more between blows;
    - a fight costs about twice the survival;
    - round-timed buffs last half as many combat rounds;
    - bleeding out gives fewer combat rounds to save someone.

    The owner accepted this for now, to retune in Phase 30.
    `CombatEveryRounds: 1` reverts it.
  - A movement command's own first line can precede the flush its move
    triggers.
  - The Party GMCP payload isn't held.
  - Held lines aren't saved, so a crash loses at most one round's
    unshown text.

- **Status effects (30a), accepted:** a status struck in the last round of
  one battle lands if the next battle has already begun against the same
  player (who is then in that fight). No sling-specific stagger, shield stun,
  or fire spell for Burning yet (a weapon override can burn). Mobs are not
  held by `no-flee`, and natural (generic) attacks leave no status. (The
  brawl tests' intermittent failures are fixed; see the work log.)

- **Found fixing the brawl flakes, for the owner (behaviour, not fixed):**
  a level-1 mob, companions included, spawns with level 2's experience, so
  it levels on its first kill whatever the kill pays; and every cast
  fizzles on a roll of 100, even at a 100% chance.

- **Live battle view (32g2), deferred:** casting, statuses, wounds, and guards on the grid (Phase 30); the
  battle summary in the view; exact enemy numbers behind a skill. (29f
  now holds the view until a round's lines are out.) A hidden enemy that
  joins and falls within one round, before the view ever looks, is named
  under Fallen (accepted: the view names every foe it never saw hidden).

- **Automatic combat (32d), for the owner and later phases:** Magic
  Missile's difficulty 75 gives a new wizard about a one-in-three chance
  to cast (GoMud's odds; spell balance not retuned); companion mana and
  health aren't saved (a restart refills both); enemies aim by personality since 30c1; companions following a flee is untested. 32c's two owner items
  (peaceful tag groups; what's allowed mid-battle) are settled by 32d.
  32c's test gaps are in its work-log entry (git history, commit
  `ff1f663e`).
- **Company logistics (32f)** (work log at commit `bea052b`): a crash
  mid-meal can spend one use without its provision; characters made
  before 32f start at ~20–25 kg with no satchel. (The web gear window's
  "count / —" is fixed by 32g: it shows weights.)
- **Character deletion (32h), accepted** (work log at commit `f418e1d`): a replay's hand-back resets the
  wrong-password count; copyover with a flagged user online races the
  resumed connection; a failed reset waits for the boot sweep; the reused
  user id keeps references elsewhere (mob XP credit, party invites,
  charms outside the room); web macros can answer the password question.

Refreshed 2026-09-26 (Phase 28). Earlier entries that later phases
resolved (the weather, load, and mount multipliers, and the 11a–11c
combat wiring) are removed; each phase's work-log entry (in git history)
keeps its own history.

- **Narration (29c):** a player's own death line still comes from the
  queued `suicide` (a user script may cancel the death), so it can follow
  a closing line; untested: no closing on defeat or broken-off, a count of
  "turns toward" lines, `sparks` and `heal` through a real cast, the PvP
  "goes for" line.
- **Tutorial replay (32b), untested paths:** death through the real death
  module, quit through its buff path, link-dead expiry, a real copyover,
  and the purge in every module at once (each module has its own purge
  test). Accepted: a real character's login elsewhere lands before the
  stale replay's purge.
- **Company polish (32a) and rosters (32a2), accepted:** the company move
  line doesn't recheck that every companion made it (a `no-go` buff, a
  locked far door); a pet isn't named in it; an offline leader's camp
  reads "A camp"; recruiter and roster config are parsed on each read, so
  a malformed entry warns on each look; rosters for recruiter rooms later
  removed from config stay on the record; `party`/GMCP "Company" status
  has no dedicated test. 29b2's accepted items (waiting-group order and
  `battle.Retain` unit-tested only) stand.

- **Plugin persistence uses atomic replacement** (`WriteStruct`/`WriteBytes`
  through `util.SafeSave`). Cross-file operations still need recovery markers.
- **Separate plugin files, no cross-file transaction.** Company, survival,
  expedition, camping, and encumbrance each persist separately. Durable
  operation IDs and applied-markers (expedition exertion, camp recovery,
  the survival ledger) make a crash between writes retry, not double-apply.
  Summon and dismiss are still not atomic across the company and survival
  files.
- **Copyover continuity** relies on plugin `SetOnSave`/`SetOnLoad`, because
  modules may not register copyover contributors. `onLoad` runs after
  `copyover.Restore` and reschedules or completes sessions.
- **Game-loop state without mutexes** (company, formation, survival,
  tutorial, the company view) assumes command dispatch stays on the main
  loop.
- **Live server acceptance** hasn't been run for most phases. Unit, race,
  wiring (`plugins.Load`), and browser (Playwright) tests cover them. The
  tutorial has no test of a real copyover or of two players in the course
  at once.
- **Enemy parties in displays (11a):** the room listing groups party
  members but passes zero EHP/DPS into `mobparty.Assemble`
  (`internal/rooms` can't import `internal/combat`), so display order isn't
  combat order. The GMCP room mob list is still per mob, not grouped by
  party. Combat itself assembles parties with real values.
- **Combat upkeep (29a) limits:**
  - A blocked `flee` is followed by the leader turning back onto a foe
    the next round.
  - GoMud's own `lookfortrouble` can still draw a group-hostile
    shopkeeper into a fight; the upkeep never does.
  - The upkeep's cost is per online leader with a companion present, and
    skips rooms with no aggro.
  - `formation reach`'s "can't reach any of the enemy" branch has no
    test.
- **Deferred by design, not scheduled:**
  - the "12+" encounter kinds;
  - mount fatigue, feed, and terrain;
  - companion custom names and AI orders;
  - multi-player camps;
  - forecasts and seasons.
- **Combat event stream (29b) limits:** the stream lives in memory, so a
  restart mid-fight loses that fight's summary. Each company player's own
  summary waits for companies with more than one player.
- **Battles and spawn groups (29b2) limits:**
  - Battles are runtime only: after a restart a fight resumes from
    `Aggro` as a new battle, and rooms respawn and regroup.
  - A lone survivor or straggler with no idle group to join lingers
    alone; roaming groups that wander together are deferred.
  - The shipped `solitary` list (lich, abyssal creeper, ent, spider
    queen) is for the owner to check; the dark acolyte stays grouped.
  - The only shipped route (oak-road) has no ambush, so the ambush pair
    is reached only by tests and future routes.
- **Found writing the help pages (behaviour, not text):** a camp rest
  says it "will be interrupted", but nothing interrupts it; stabling a
  mount is free; Hunger or Thirst at 0 has no further effect; the
  `company` and `archetype` usage strings leave out some subcommands.
- `modules/tutorial`'s wiring test can't run twice in one process
  (`go test -count=2` fails the second run, before 29b too), and running
  it alongside `modules/company` in one `go test` with a high `-count` can
  collide on the shared config-overrides file.
- `make test`'s `js-lint` stage can stall on some hosts (`npx jshint`); `make
  js-lint` on its own and `go test -race ./...` are the documented checks.
- A shipped-world test can leave a gitignored
  `_datafiles/world/default/config-overrides.yaml`; delete a stray copy if
  wiring tests fail on a relative data path.

## Key documents

- [Agent workflow](AGENT_IMPLEMENTATION_WORKFLOW.md) — implementation and review.
- [Handoff](ASHVEIL_GOMUD_AGENT_HANDOFF.md) — design direction and invariants.
- [Combat roadmap](designs/2026-09-26-combat-presentation-roadmap.md) — remaining phases and decisions.
- [Visual client milestone](designs/2026-10-05-visual-client-milestone-design.md) and [sprite specification](designs/2026-10-05-sprite-specification.md) — Phase 40 map and battle screen.
- `docs/designs/` — active proposals and useful shipped design records.
- `docs/plans/` — execution guidance and work still requiring follow-up.
- Nested `AGENTS.md` files — package-specific constraints.
- Git history — retired plans/proposals and detailed completed work logs.
  Use `git show 792455ea:<old-path>` for the snapshot before this cleanup.
