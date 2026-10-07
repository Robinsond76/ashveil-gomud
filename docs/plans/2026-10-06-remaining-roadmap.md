# Remaining roadmap (2026-10-06)

The owner handed over the rest of the roadmap on 2026-10-06: keep working
until every phase is done, use best judgment wherever owner input would
normally be needed, choose the visual direction, and add phases where a
clear improvement exists. This plan maps every phase not yet merged, with
its dependencies, and records each decision made under that delegation.

Live status stays in [Project Status](../PROJECT_STATUS.md) ("Phase
sequence"). This doc is the reasoning behind it.

## Delegated approval

Handoff rule 20 asks for owner approval of each design. From 2026-10-06
the owner has delegated that: a design or review thread approves a draft,
records each open question's answer with a one-line reason in the design
and in Project Status, and the phase proceeds. The project workflow (build
thread opens the PR, review thread reviews, fixes and merges) is unchanged.

## Phases at a glance

"Now" means the phase can start while 37 and 38b are being built. Shared
files (Project Status, `keywords.yaml`, `modules/tutorial/stages.go`) only
cost merge order, not a dependency.

| Phase | Scope | Depends on | Start |
|---|---|---|---|
| 37 | Random encounters, zone bands, drop tables (loot slice 3) | 35b, 35d, 36b | In flight |
| 38b | Promotion at 10, talents, core routes for six lineages | 35a, 35a2, 35d, 38a | In flight |
| **35e** | **Focus the healer** (new, below) | — | **Now** |
| **44** | **Live smoke playtest** (new, below) | — | **Now** |
| 40e | Structured combat events (GMCP) | — | **Now** |
| 40a | Room resources | — | **Now** |
| **40s1** | **Art sets S0 and S1** (new; see "Visual direction") | — | **Now** |
| **38c-d** | **Elite routes design** for the six lineages (docs only) | — | **Now** |
| 37b | Encounter and pacing tuning (new, below) | 37, 35e | After 37 |
| 36c | Loot economy, plus selling rolled gear and GMCP rolled names | 37 | After 37 |
| 39a–39h | Neutral base classes, at most two building at once | 38b (39e also 39d) | After 38b |
| 38c | Elite promotions at 30, ranks 30–60, six lineages; three slices 38c1–38c3 ([plan](2026-10-06-phase-38c-elite-routes.md)) | 38b, 38c-d | After 38b |
| 38d | Expanded class catalogue bundles, Sorcerer first | 38c | Later |
| 38e | Creature recruits (Hound and Stone Golem pilot) | 38d, 39e | Later |
| 39i | Elite ranks for Halberdier, Samurai, Shaman and Doll Master, plus the balance items (**in review**) | 38c, 39a–39h | PR open |
| 39i2 | Elite ranks for Beast Tamer, Gryphon Rider, Alchemist and Arbalist (new, split from 39i) | 39i | Merged (PR #122 via review PR) |
| 36d | Tier 4–6 gear, Legendary signatures, Set bonuses (loot slice 5) | 36c, 38c | Later |
| 40a2 | Gathering | 40a | After 40a |
| 40a3 | Camp gear (already approved) | 40a2 | After 40a2 |
| **40a4** | **Camp theft** (new, from the deferred item) | 40a3 | After 40a3 |
| 40s2 | Art set S2, terrain and landmarks | 40s1 | After 40s1 |
| 40s3 | Art set S3, battle units and backgrounds | 40s1 | After 40s1 |
| 40s4 | Art set S4, battle animation and effects | 40s3 | After 40s3 |
| 40s5 | Art set S5, advanced and neutral class art | 40s3, 38b | After 38b |
| 40b | Map class sprites, company badge, camps | 40a, 38b, 40s1 | After 38b |
| 40c | Terrain and landmark tiles | 40b, 40s2 | After 40b |
| 40d | Showcase region and click-to-walk | 40c | After 40c |
| 40f | Static battle screen | 40e, 40s3 | After 40e |
| 40g | Battle animation and effects | 40f, 40s4 | After 40f |
| 40h | Promoted and neutral class art in the client | 40b, 40g, 40s5 | Later |
| 40i | Touch layout and installable web app | 40d, 40f | Later |
| **41** | **World building, levels 1–15, tile-ready** (new number) | 40d, 37b | Built as the test-only world ([plan](2026-10-07-phase-41-42-test-world.md)) |
| **42** | **Zones 15–30+, elite content, tier 4–6 placement** (new) | 41, 38c, 36d | Built with 41 (same plan) |
| **43a** | **Camp consumables** (designed 2026-10-01, now scheduled) | 40a2 | After 40a2 |
| **43b** | **Weapon poisons** (designed 2026-10-01, now scheduled) | 43a | After 43a |
| **50** | **Condition carries into battle; meal buffs** ([Outward phases](2026-10-06-outward-survival-phases.md)) | 47 | After 47 |
| **51** | **Rest duties** (sleep, watch, tend, forage, cook, brew) | 47 (coordinate with 49) | After 47 |
| **52** | **Tents with trade-offs** | 51 | After 51 |
| **53** | **Defeat scenarios** in place of the church respawn | Companion equipment, 40a4 | After companion equipment |
| **54** | **Sigils**: prepare the ground before a battle | — | **Now** |
| **55** | **Ailments and remedies** | 50 | After 50 |
| **56** | **Recipe discovery** | 50 | After 50 |

### Order of work

1. **Start now, in parallel with 37 and 38b:** 35e, 40e, 40a, 40s1, 44,
   and the 38c-d design. 35e first: it unblocks enemy healers in 37b.
2. **When 37 merges:** 37b (needs 35e too) and 36c, in parallel.
3. **When 38b merges:** 38c, 39a and 39b (then two neutral classes at a
   time, 39e after 39d), 40b (with 40a and 40s1), and 40s5.
4. **Visual lanes:** battle screen 40e → 40f → 40g, and map 40a → 40b →
   40c → 40d, run side by side. Art sets stay one step ahead of the code
   that needs them. 40a2 → 40a3 → 40a4 and 43a → 43b follow 40a.
5. **Then:** 40h and 40i, 39i, 38d and 38e, 36d, then world building 41
   and 42 last, so new zones use every system above.

## New phases

### 35e Focus the healer

Why: the owner's difficulty rule keeps enemy healers rare until the
company can deal with them. Today the `casters` focus hits any caster, so a
company has no rule that finds the healer first.

Scope: a `healers` focus rule (enemy healers first, a chanting one before
an idle one, then the `casters` order); reach still comes first, so a
member who can't reach the healer picks from reachable foes by formation;
the company's default focus switches to `healers` whenever the enemy group
has a healer, from leader level 5; `company tactics` and the battle view
show it; `help tactics` and a tutorial hint. Balance check: a company with
the default beats a tier 2 group with one healer about as often as one
without.

Decision: the healer default starts at level 5, not 10, because 37 places
enemy healers from the lowest bands and a new company would otherwise face
them with no answer.

### 37b Encounter and pacing tuning

Why: 35b and 35d left measured misses for real encounters (bosses too easy
at low bands, cleric mana dry by fight 3, 35–39% no-damage swings, rounds
of about 8 s). 37 builds the encounters; 37b tunes against them.

Scope: raise enemy healers from rare to uncommon (about one group in six
at band middle) now that 35e exists; harness cells in tier-appropriate
gear (36b's deferral); re-measure the four 35b zone rows and the 35d
misses with real encounter tables; tune boss HP and escorts at low bands.
Timeboxed per the balance rule: record the best result and move on.

### 44 Live smoke playtest

Why: Project Status notes that live server acceptance hasn't run for most
phases. Unit and wiring tests miss start-up, copyover and multi-player
seams.

Scope: a scripted playthrough against a real server process over telnet
(create a character, tutorial, recruit, rest, a fight, save, copyover,
log back in) run by a Go test behind an env flag, plus one two-player
session. Findings become fixes or Known issues. Repeat it at the end of
each lane.

### 39i Neutral elite ranks

The neutral classes design says their elite ranks ship with 38c+. 38c
covers the six original lineages; 39i adds ranks 30–60 for the eight
neutral lineages once all of them and 38c are in.

39i split in two to fit one reviewable PR: **39i** ships the four earliest
lineages (Halberdier, Samurai, Shaman, Doll Master; twelve elites) and the
balance items; **39i2** ships the other four (Beast Tamer, Gryphon Rider,
Alchemist, Arbalist; twelve elites, each still `Planned`), reusing 39i's
pattern: seven-rank tables, elite talents offered through `offerElite`, a
wiring test that fires each signature in a real round, a mirror sim
(`TestPhase39iNeutralElites`, extend its lineage table), one help page per
elite and the lineage routes page updated.

### 40a4 Camp theft

The owner recorded camp theft as a future feature that gives bells and
trip lines (40a3) their point. Scope: a silent camp event while a company
rests without bells, taking a small share of loose loot and supplies,
noticed on waking; never takes equipped gear, gold in the treasury, or
quest items; persisted so a restart can't double-apply.

### 41 and 42 World building

World building waits for the visual milestone (owner, 2026-10-05). 41
builds zones for levels 1–15 to 40d's tile-ready conventions, with 37's
encounter tables. 42 adds zones 15–30 and beyond, so elite promotions
(38c) and tier 4–6 drops (36d) are reachable in play, with bosses and
contracts to hold the Legendary and Set items.

### 43a and 43b Camp consumables and weapon poisons

Both were designed on 2026-10-01 and never scheduled. They come after
gathering (40a2) so ingredients can be gathered as well as bought, and
they give the Alchemist (39g) and rogue routes more to work with. Their
proposed balance numbers are accepted as defaults.

### 50–56 Survival lessons from Outward

Added 2026-10-06 at the owner's request after a research thread on
Outward. They make the existing survival layer matter in the core loop:
condition going into battle (50), choices at the camp fire (51, 52), a
story instead of a church respawn on defeat (53), set-up before a fight
(54), and texture for herbs and cooking (55, 56). Scope, sizes, the
overlap check against 47, 49 and companion equipment, and each phase's
acceptance are in the [phase plan](2026-10-06-outward-survival-phases.md).
Build order: 50 and 54 first, then 51 and 53, then 52, 55 and 56.

## Folded into existing phases

- **36c** also ships the 36a deferrals: merchants buy rolled gear (priced
  by quality and rarity, unidentified at a discount) and GMCP item lists
  show rolled names.
- **36d** ships the Legendary signature effect and Set bonuses (36a
  deferral) with tier 4–6 (loot slice 5).
- **37b** carries the 35d accepted misses and 36b's harness deferral.

## Visual direction (owner asked for a recommendation)

The art direction is settled (grounded Tolkien-style fantasy, Ogre Battle
64 as the sprite reference, never anime). Two choices were left open.

**Which visual comes first: the battle screen.** Battles in Ashveil play
themselves from the company's setup, so watching them is the core loop,
and the server already sends everything a battle screen needs. The battle
lane (40e → 40f → 40g) gets priority; the map lane runs beside it.

**Where the art comes from: code-generated pixel art.** The sprite
specification assumed another agent would draw the sets, with no source
named. Each art set is built by a thread as small pixel maps and part
layers (body, armor, cloak, weapon, palette swaps) compiled to PNG sheets
by a script under `scripts/sprites/` with Pillow, in the specification's
64-color palette and file layout. The art is reproducible, reviewable as a
contact sheet, and replaceable file by file if commissioned art arrives
later, with no code change. The review thread for 40s1 judges the S0
style against the specification's art direction, in place of the owner's
S0 gate. Art threads touch only the sprite folders and the script, so
they never block code.

## Decisions on open questions

Each was answered under the owner's delegation, 2026-10-06.

| Design | Question | Decision | Reason |
|---|---|---|---|
| 40a | Water freezes in blizzards? | Later, with weather work | Keeps 40a small; weather owns freezing |
| 40a | Forage +1 find, shelter halves the weather penalty | Accepted | Small, readable numbers; tune later |
| 40a2 | Times, pools, regrowth, chances | Accepted as defaults | Timeboxed tuning in the phase |
| 40a2 | Pelts and hides wait for goods? | No: `hunt` yields meat plus the shipped hide and pelt goods | 36b shipped the goods (wolf hide, bear hide, snowcat pelt) |
| 40c | Tiles default or opt-in? | Default once every biome has a tile (S2 covers all), classic toggle kept | Generated art covers every biome at once |
| 40d | Showcase area | Approved as recommended: a new compact area off the existing roads | Leaves shipped rooms and tests untouched |
| 40e | Spectators get events? | No | Matches `Company.Battle`; no hidden info leaks |
| 40f | Allied companies on the battle screen | Approved: half-scale reserve formations, read-only swap view | Shows allies without crowding the 3x3 |
| 40g | Sound effects | Out of scope; a later optional phase | Visuals first; GoMud audio stays available |
| 40g | Per-action animation budgets | Accepted | Server timing stays authoritative either way |
| Loot | Bad-luck protection N = 20 per boss per leader | Accepted | Proposed default; 37 measures it |
| Loot | Smart loot 50% toward the company's families | Accepted | Helps without making drops predictable |
| Loot | Market demand saturation | Accepted as proposed, in 36c | Stops goods farming one market |
| 40h, 40i | No design yet | Each build thread drafts a short design in its plan; the review thread approves | Scope is already set in the milestone design |
| Art | S0 approval gate | The 40s1 review thread approves against the art direction | Owner delegated; style rules are explicit |

## Build notes for every brief

- Follow the speed rules and the UI check in the project workflow memory:
  Sonnet builds and opens the PR, an Opus thread reviews, fixes and merges.
- Never advance global game time; persist multiplayer state across
  restart and copyover.
- Ship indexed help and tutorial pointers with every player-facing phase.
- Balance work is timeboxed: record the best result and move on.
