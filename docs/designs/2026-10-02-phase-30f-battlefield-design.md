# Phase 30f: Battlefield Conditions

Status: owner approved all proposed defaults, 2026-10-02; implemented on the feature branch; verification complete; PR publication pending. Implementation branch:
`phase-30f-battlefield`. This develops the
[original proposal](2026-09-26-battlefield-conditions-design.md).
The owner requested implementation and an unmerged GitHub PR awaiting
**Opus 5.5** review. The phase remains unmerged and awaits independent review.

## Scope and existing foundations

Deliver all five proposal areas: ambush/surprise, formation clusters and
sweeps, enemy leaps and flanking, narrow ground, and fatigue/cold effects.
Mounted combat remains removed. Combat remains round based; shared world
time never advances for travel, rest, or an opening advantage.

Reuse `enemyparty.SpawnAmbush` (travel interruptions and camp raids),
`battle` ownership and the real `hooks.DoCombat` loop, `formationcombat`
legality, saved `company.Formation`, and the existing survival/exposure
services. `sparks` currently targets multiple foes; its scope becomes a
formation cluster. Camp Watch currently spots raids and preserves rest;
tracker ambush evasion already prevents spawning. Preserve both benefits.

## Approved gameplay decisions

### Ambush and surprise

Resolve once for a newly spawned travel encounter or camp raid. Ordinary
room aggression and subsequent battles with the same surviving foes do
not roll again. The best living, present company observer supplies adjusted
Perception and their own effective visibility. Compare with the strongest
ambusher's adjusted Perception as the initial stealth fallback; an optional
validated template stealth override allows authored encounters.

Detection chance: clamp `50 + observer Perception - enemy stealth
- light penalty - terrain concealment` to 5–95 percent. Light penalty:
0 lit, 10 dim, 25 dark. A reserved `ambush-cover` room tag adds 10 points
of concealment; an ordinary room adds zero. The actual encounter room owns
terrain and visibility, not a guessed destination or route-name substring.
On a failed roll, enemies receive one opening combat round. On success,
neither side is surprised; success by at least 30 points grants the company
the opening round. A successful existing Camp Watch roll guarantees at
least ordinary detection and keeps its existing rest-preservation effect.
Watch failure still permits Perception detection to prevent combat surprise,
but does not retroactively change the existing rest-preservation decision.

An opening round means the surprised side cannot begin/advance casts, use
abilities, wind up, or make attacks that round. Status expiry and world
systems keep ticking. Passive mitigation and guards remain available.
Retreat is permitted; it consumes that company's action as usual. Avoid
executing an extra nested combat loop or granting extra attacks mid-round.
Announce the observer and result once, with no hidden enemy stats exposed.

### Area spells and sweeping attacks

`sparks` chooses a primary legal target in the caster's current battle and
hits that occupied cell plus its orthogonal neighbors (Manhattan distance
at most one). Diagonals and chained adjacency are excluded. Resolve living,
visible targets again on completion, so dead/moved/hidden foes and waiting
groups are excluded. Preserve cast costs, chant interruption, damage, and
secondary statuses. A foe without a formation is a single target, never a
reason to hit the room. Ordinary direct spells keep their current scope.

Add an opt-in enemy sweep capability: one action strikes each eligible
living front-row opponent once, using the existing damage/defense pipeline.
No offhand multiplication, recursive sweeps, allies, spectators, or other
battles. Cooldown: three combat rounds. A sweep uses normal damage
per target initially; measure it in focused scenarios before finalizing.

### Enemy leaping and flanking

Add an opt-in enemy leap capability, not a new player command. A leaper
may bypass its target column's front-row interceptor when that occupant is
absent, dead, or knocked down. It may strike the middle or back row within
normal lateral range. A standing front-liner blocks the leap; guardians
can still intercept the resulting strike. Cooldown: three combat
rounds; a leap replaces an ordinary attack and uses normal strike damage.
Unflagged enemies retain existing reach.

Flanking opens melee access to a rear target when its column has no
standing front-line protector and an adjacent column is also open. Only
living, present, combat-capable protectors count. No permanent placement
or aggro changes; recompute legality as the line breaks and recovers.
Keep ordinary targeting, strategy, and attack gates consistent with the
same rule so automatic actors actually select reachable rear targets.

### Narrow ground

Reserve the `narrow` room tag. Derive a temporary three-row, two-column
battle formation for both sides, preserving every living placed member
and deterministic front-to-back priority. Assign members in original
row-major order into available two-column cells; retain original depth
when space allows and spill collisions rearward. Never mutate saved
formation or its migration version. At most five company members fit;
if a larger enemy formation exceeds six cells, overflow cannot make melee
attacks but remains targetable, and may enter a vacated cell on a later
round. Overflow uses column 3 as an explicitly marked reserve; ranged attacks
and spells remain available. Reserve cells do not neighbor active cells for
clusters, and neither guardians nor their wards use reserve cells. Members
spill rearward first, then forward when all deeper cells are occupied.

Use the effective formation consistently for target selection, reach,
interception, guardians, clusters, sweeps, and the Battle view. Tell players
when narrow ground folds the lines. The stored formation editor continues
to show saved placements; Battle shows current combat placements. Leaving
the room restores ordinary formation immediately.

### Fatigue and cold

Fatigue is remaining rest, not accumulated exhaustion: 100 rested, 0
collapsed. Physical hit modifiers: 51–100 no penalty; Tired
26–50 minus 5 percentage points; Exhausted 1–25 minus 10; Collapsed 0
minus 20. Apply once alongside darkness and other hit modifiers, respecting
existing hit clamps. Read authoritative member needs; missing survival
service/state means fully rested. Apply to leader and eligible companions,
not unrelated enemies without company survival records. No new combat drain.

Existing cold buffs already reduce stats; keep those effects. At Frostbitten
or worse (buffs 1011–1013), add one combat round to starting a spell chant
or loading a sling shot, once per action. Mark slings using an explicit item
capability rather than a display-name match or treating every ranged weapon
as a sling. Cold changes after an action begins affect its next action,
not a timer repeatedly extended forever. All casting entry points use the
same rule, including explicit player casting and automatic strategies.
Narration explains fatigue-affected strikes and cold-delayed actions without
emitting the same condition warning every round.

## Ownership, multiplayer, and recovery

Saved company formation, survival needs, exposure, travel sessions, and
camp rest/raid state keep their existing owners and persistence. New
template capabilities are static authored content with legacy-safe defaults.
Effective formations, cooldowns, and opening-round suppression belong to
runtime battle/encounter state, never presentation or combatstream state.

An encounter advantage is keyed to its spawned group and intended company,
consumed once, and cleared on despawn/end. A group already fighting another
company cannot gain or lose another opening round when a second player
joins. Neither player's area attacks, suppression, nor counters leak into
the other's battle. Restart/copyover follows existing new-battle recovery:
do not replay a spent encounter advantage or duplicate an encounter. Keep
existing durable raid firing and travel checkpoint behavior; test failure
paths and avoid new spawn/save duplication windows.

## Content and player help

Ship a small authored example of narrow ground, a leaping enemy group, and
a sweeping enemy group, chosen from existing suitable locations/templates
after inspecting shipped content. Reserve tags/capability metadata in the
owning package guides. No broad encounter-table rebalance.

New help topics: `battlefield`, `ambush`, and `formations` if no suitable
existing formation page covers these rules. Update existing `combat`,
`formation`/`tactics`, `sparks`/spell help, `survival`, `exposure`, `camp`,
`travel`, and `webclient` topics as applicable. Index topics and aliases,
link from Combat, and add tutorial Combat/Departure hints. Include the
actual thresholds, area shape, opening-round behavior, and counterplay.

## Acceptance and verification

- Seeded travel ambush: failed detection suppresses exactly one company
  round; ordinary success suppresses neither; exceptional success suppresses
  exactly one enemy round. Camp Watch and tracker evasion retain benefits.
- Real combat entry points suppress casts, abilities, wind-ups, and attacks
  consistently; passive defense and permitted retreat still work.
- `sparks` completion hits only current orthogonal neighbors of its primary
  target in the active battle; tests include diagonals, hidden/dead targets,
  other groups, unformed enemies, and enemy casters targeting a company.
- Real sweep applies independent defenses once per target and one cooldown;
  leaping reaches a rear member over a downed protector but not a standing
  one; guards and flanking integrate with automatic target selection.
- Narrow terrain folds both sides consistently, preserves saved placements,
  handles overflow/deaths deterministically, and reports the same positions
  through text and private GMCP without leaking hidden opponents.
- Fatigue changes actual physical hit rolls at every boundary; missing
  service is neutral. Cold delays explicit/automatic chants and sling shots
  exactly once, with accurate narration, and does not delay other weapons.
- Multiplayer, encounter cleanup, restart/copyover, save/spawn failures,
  and unchanged global time have integration coverage.
- Help renders; tutorial pointers exist; focused balance scenarios report
  opening advantage, clusters, and sweep impact without rewriting unrelated
  baseline goldens to conceal a regression.
- Focused tests during work, then `make generate`, `make validate`, applicable
  JS/Lua lint, and `go test -race ./...` after final code changes. Record exact
  results and any unavailable check honestly.

## Delivery and review

Implement only after explicit design approval. Commit and push this feature
branch, create a GitHub PR targeting `master`, and **do not merge**. PR details
describe shipped behavior, defaults, integration coverage, validation, and
limitations. Update Project Status and the combat roadmap on the feature
branch with the actual PR URL and **awaiting review by Opus 5.5**. Do not
claim that independent review occurred until its findings are available.
