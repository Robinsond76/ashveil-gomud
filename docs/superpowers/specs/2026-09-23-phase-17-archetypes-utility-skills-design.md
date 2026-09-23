# Phase 17: Archetypes and Utility Skills

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`.
It implements user decision 3: archetypes are exclusive, so a player picks
exactly one and can't learn another archetype's skills, while professions
such as cooking stay open to all. Source request: "Archetype utility skills
(wizard light, rogue locks/traps, cook, ...), auto-triggered when needed
and usable manually."

**Status:** design written 2026-09-23. On 2026-09-23 the user said "Please
implement", which confirms open decisions 1–8 as recommended. Where the
build differs from this text, see "Implementation notes" at the end.

## Prior-art check

- **Skills** (`internal/skills`, `_datafiles/world/default/skills/`) are
  data-driven, keyed by lowercase ids, and stored in `Character.Skills`
  (`map[string]int`, levels 1–4). There are 16 skills: brawling, cast,
  changeform, dual-wield, enchant, inspect, map, peep, portal, protection,
  scribe, search, skulduggery, tame, track, and trading. They are trained
  **only** in `usercommands.Train`, at rooms with `SkillTraining` ranges,
  for training points. Commands gate on `GetSkillLevel(id) > 0`. Nothing
  today stops any character from training any skill.
- **GoMud "professions"** (`internal/skills/profession.go`, 10 datafiles
  such as warrior, assassin, and sorcerer) are **titles derived from skill
  mastery**. They are not choices and they gate nothing. The roadmap's
  "professions (cooking, skinning, mending)" means something else: open
  trade skills. This design says **trade skill** for the roadmap sense, to
  avoid colliding with `skills.Profession`.
- **Spells.**
  - `Character.SpellBook` is learned only through `Character.LearnSpell`,
    which scripts call: trainer mobs (the clergyman teaches heal, Elara
    teaches the party illum), items (the magic missile tome), and the
    shadow master teaching charmrat.
  - Casting needs the `cast` skill plus the spell in your spellbook.
  - Spells carry a `school`: conjuration, illusion, and restoration. One
    file misspells it as `illlusion`.
  - Phase 14's `floatinglight` (school illusion) is a party light. Phase 14
    explicitly deferred "wizard-only gating" to this phase. **No script
    teaches `floatinglight` today.**
- **Traps.** `gamelock.Lock.TrapBuffIds` exist only on locks, on exits or
  containers. Traps fire only when a `picklock` attempt breaks the pick.
  There is no trap detection and no disarm. **No shipped room has a trapped
  lock.** Lock unlock state (`UnlockedRound`) is runtime-only: it isn't
  persisted, so it resets on restart.
- **Company companions** are mobs spawned from a template
  (`company.Companion{ID, MobTemplateID}`). The only allowed template is
  58, the training dummy. The durable per-member state lives in Ashveil
  registries keyed by `survival.MemberKey`: survival, exposure, and
  walking. Companions don't train.
- **Round count** persists across restart (`.roundcount`) and copyover, so
  round-based expiries survive both.
- **Command name collisions:** `disarm` is taken by brawling (disarming an
  opponent's weapon), and `sense` is free. Trap actions therefore go under a
  new `trap` command.
- **Cooking** has nothing to build on yet. Exactly one shipped item has
  `nutrition`, the cheese sandwich. Raw ingredients arrive with Phase 18's
  loot tables.

## Scope

Phase 17 ships in two slices, following the 11a/b/c and 12a/b/c precedent.
The third item is deferred:

- **17a: archetypes.**
  - A data-driven archetype table, and a durable one-time choice for
    players and companions.
  - An `archetype` command.
  - Gating for skill training and spell learning.
  - The archetype shown in `company` and on inspecting a character.
- **17b: utility skills.**
  - Effective utility level for players and companions.
  - Company best-member resolution.
  - Per-character `autoskill` toggles.
  - **Wizard light:** `floatinglight` gated to wizards, taught on
    choosing the archetype, and auto-cast on entering darkness.
  - **Rogue traps:** a `trap` command with `sense` and `disarm`,
    auto-sensing on entering a room and before `picklock`, and one
    trapped-lock proving room.
- **Deferred to after Phase 18: cooking**, as the first open trade skill.
  It needs ingredients from loot (decision 7).

Out of scope:
- floor or room traps
- archetype stat templates, archetype combat abilities, and AI personality
  (Phase 11d)
- respec economics
- alignment gates (Phase 21)
- multiclassing

## Model

### 1. Archetypes (data)

The new datafiles live in `_datafiles/world/default/archetypes/*.yaml` and
are loaded by `internal/archetypes`, with the same loader and validation
pattern as skills:

```yaml
archetypeid: wizard
name: Wizard
description: Scholars of the arcane who light the dark and bend illusion.
skills: [cast, enchant, portal]       # exclusive: only this archetype (or others listing it) may train
schools: [illusion, conjuration]     # exclusive spell schools
grants:                               # applied once, on choosing
  skills: { cast: 1 }
  spells: [floatinglight]
utility: [light]                      # utility actions this archetype performs
companionlevels: [1, 10, 20, 30]      # companion character level for utility skill level 1..4
```

The default table covers five archetypes, echoing the prototype party. All
of it is data.

| Archetype | Exclusive skills | Schools | Grants | Utility |
|---|---|---|---|---|
| warrior | brawling, dual-wield | — | brawling 1 | — |
| rogue | skulduggery, peep | — | skulduggery 1 | traps |
| wizard | cast, enchant, portal | illusion, conjuration | cast 1, `floatinglight` | light |
| cleric | cast, protection | restoration | cast 1, protection 1 | — |
| ranger | track, tame | — | track 1 | — |

- A skill listed by **any** archetype is *claimed*. Only characters whose
  archetype lists it may train it. For example, `cast` belongs to both
  wizard and cleric.
- An unclaimed skill is a **trade skill**, open to all: search, inspect,
  map, trading, scribe, and changeform today, and cooking later.
- A spell whose school is claimed can be *learned* only by that archetype.
  A spell with no school, or an unclaimed school, is open.
- The data fix `illlusion` → `illusion` ships with this phase.
- Validation rejects:
  - an unknown skill id (warning and skip, following the skills
    cross-reference pattern)
  - an empty id or name
  - a grant of a skill the archetype doesn't list
  - a grant of a spell whose school the archetype doesn't claim
  - `companionlevels` that aren't ascending and exactly four long

### 2. Durable choice

`modules/archetype` keeps a plugin-store registry:

```go
type Registry struct {
    Players    map[int]string                    // userID → archetype id
    Companions map[int]map[int]string            // leader userID → companion id → archetype id
    Autoskill  map[int]map[string]bool           // userID → utility → enabled (default on)
    Disarmed   map[string]uint64                 // lock id → round the trap re-arms at
}
```

- **Players** choose with `archetype choose <name>`, which asks for
  confirmation (`archetype choose <name> confirm`). The choice is
  one-time. `archetype` alone lists the table and your current choice.
- Choosing applies the `grants` once through `SetSkill` (never lowering a
  higher existing level) and `LearnSpell`. Grants are idempotent: after a
  crash between saving the registry and saving the user, the next login
  re-applies them harmlessly.
- An **admin** can clear a player's choice with
  `archetype reset <player>`. Granted skills and spells stay; no
  respec economy this phase.
- **Companions** take an archetype from company config.
  `CompanionArchetypes` maps a mob template to an archetype, and template
  58 maps to warrior. The archetype is recorded when the companion is
  recruited. The leader can set it once if it is unset, with
  `company archetype <member> <name>`. Companions recruited before this
  phase have none until the leader sets one.
- **Existing characters are grandfathered.** Skills and spells a character
  already has stay usable whatever archetype they later pick (decision 2).
  Gating only stops new training and new learning.
- Pruning: when a companion leaves the roster, its entry is dropped
  through the existing `survival.Lifecycle`-style roster reconciliation
  used by survival, exposure, and walking.

### 3. Gating (engine seams)

`internal/archetypes` gets a read-only seam, following the
`expedition.MovementBlocked` pattern. With no provider registered, it is a
no-op that allows everything, so upstream behaviour is unchanged when the
module is absent:

- `archetypes.CanTrain(userID, skillID) (bool, reason)` is consulted in
  `usercommands.Train`, both in the panel (claimed skills show "Wizard
  only") and on purchase. An unchosen player training a claimed skill is
  told to choose an archetype first.
- `archetypes.CanLearnSpell(userID, spellID) (bool, reason)` is consulted
  in `ScriptActor.LearnSpell` for users, which covers every scripted
  teacher, and in the party `LearnSpell` for each member. A refusal sends
  the reason and returns false, which scripts already handle as "already
  known". `admin spell` bypasses the gate.
- Casting isn't gated, because known spells are grandfathered. What stops
  a non-wizard from getting `floatinglight` is that the only way to obtain
  it is the gated learn path or the wizard grant.

### 4. Utility level and best-member resolution (17b)

- **Utility skill mapping** (data): `light` → `cast` (wizard);
  `traps` → `skulduggery` (rogue).
- **Effective utility level of a member**, 0–4:
  - **Player:** their level in the mapped skill, but only if their
    archetype lists the utility. Otherwise 0.
  - **Companion:** derived from its archetype's `companionlevels` against
    the live mob's character level. It is 0 if the companion has no
    archetype, isn't spawned, or isn't in the room.
- `BestMember(leaderUserID, roomID, utility)` returns the present member
  with the highest effective level. Ties go to the leader, then the lowest
  companion id, so the result is deterministic. Nothing is stored.

### 5. Wizard light

- **Manual:** `cast floatinglight` is unchanged. It now needs a wizard,
  through the learn gate and the grant.
- **Auto:** on the walker's `RoomChange` after an ordinary step, the
  module checks whether the room is dark for the company (Phase 14's
  per-viewer light) and whether nobody already has a party light. If so,
  it resolves `BestMember(light)` among members with autoskill `light`
  on:
  - **A player wizard** casts `floatinglight` through the normal cast
    command path. That spends mana and respects cooldowns and wait rounds.
  - **A companion wizard** gets the floating-light buff (1000) directly,
    as the Phase 14 party light, at a mana cost from the mob's mana. It
    shares the same `AutoLightCooldown` round cooldown (config: 10
    rounds).
  - There is at most one attempt per step. A lack of mana or an active
    cooldown is silent. A success sends one line ("Orin conjures a
    floating light.").
- Auto-light never fires in combat or while downed.

### 6. Rogue traps

- `trap sense [target]`:
  - The best present member with `traps` rolls:
    `level × SensePerLevel (20) + perception/4` against
    `lock difficulty × SenseDifficultyFactor (5)`.
  - Success names every trapped lock in the room, or the one targeted.
  - Failure says "You find no traps," which is honest ambiguity.
  - Cooldown: `SenseCooldown`, 2 rounds, per character.
- `trap disarm <exit|container>` uses the same resolution with
  `DisarmPerLevel` against `difficulty × DisarmDifficultyFactor`:
  - **Success** records `Disarmed[lockId] = now + DisarmRounds` (config
    default: the lock's relock interval, or 1 hour of rounds).
  - **Failure by more than `DisarmBackfireMargin`** springs the trap on
    whoever attempted it (the same buffs as `picklock`).
  - Requires the utility level to be at least 1.
- Lock ids reuse `picklock`'s `<roomId>-<exit|container>` format.
- **Picklock integration:** `usercommands.Picklock` consults
  `archetypes.TrapArmed(lockId)` before springing `TrapBuffIds`. A
  disarmed trap doesn't fire. When autoskill `traps` is on, it also runs
  one free auto-sense before the first pin, and warns "Your instincts
  prickle: this lock is trapped."
- **Auto-sense on entering a room:** if autoskill `traps` is on and the
  room has any trapped lock, the best member gets one passive roll per room
  entry, at a −20 penalty, with no cooldown cost.
- **Durability:** `Disarmed` expiries are persisted as rounds, and the
  round count survives restart, so a disarmed trap stays disarmed across
  restart until its expiry. Expired entries are pruned on load and on
  every `trap` use.
- **Proving content:** a trapped, locked chest in Dunmar (a new
  container), with a mild existing debuff as the trap.

### 7. `autoskill`

- `autoskill` lists your utilities and their state.
- `autoskill <utility> on|off` sets one.
- Every utility defaults to on.
- Settings are per character and persisted. Companions follow their
  leader's toggles.

## Constraints and invariants

- **Never advance the world clock.** There are no timers. Auto-triggers
  run in the command or event path, and expiries are read against the
  round count, never advanced.
- **Survive restart and copyover.** Archetype choices, companion
  archetypes, toggles, and disarm expiries are all persisted. Grants are
  idempotent, so a crash mid-choice converges.
- **Concurrency.** Everything runs on the game loop: commands,
  `RoomChange` listeners, and script `LearnSpell`. `modules/archetype`
  has one leaf mutex and never calls another module while holding it.
  Provider reads (company roster, Phase 14 light, room locks) happen
  before or after its lock section.
- **Compatibility.**
  - With no provider registered, every seam allows everything.
  - A character with no archetype keeps all existing abilities.
  - A legacy companion has no archetype and no utility.
- Data-driven balance: every number above is module config or archetype
  data.

## Open decisions (recommendations, awaiting confirmation)

1. **The archetype set:** warrior, rogue, wizard, cleric, and ranger,
   with the skill and school split in the table above. `cast` is shared by
   the wizard and cleric, who are separated by spell school.
2. **Grandfather existing abilities.** Gating stops new training and
   learning, and nothing already known is taken away. The alternative is
   stripping off-archetype skills on choice.
3. **The choice is one-time and permanent,** with only an admin reset and
   no player respec this phase.
4. **Unchosen players can still train trade skills,** but must choose an
   archetype before training any claimed skill.
5. **Companion archetypes come from a template map in config,** and
   companion utility levels come from character level through
   `companionlevels`. Companions don't train.
6. **Traps are a `trap sense|disarm` command,** because `disarm` is
   already brawling's. Disarms last one relock interval and are persisted
   by round.
7. **Defer cooking** until Phase 18 supplies ingredients, as the first
   trade skill in a Phase 18b. The alternative is a minimal recipe table
   now with one hand-placed raw item.
8. **Auto-light casts through the real cast path for players,** spending
   mana and respecting wait rounds, and applies the buff directly for
   companions.

## Acceptance criteria

- **Pure tests:**
  - archetype loading and validation, including every rejection case in §1
  - claim resolution: a claimed skill, a shared claim, a trade skill, and
    a school claim
  - `CanTrain` and `CanLearnSpell` decisions for chosen, unchosen, and
    grandfathered characters, and with no provider
  - effective utility level for a player and for a companion (the level
    table)
  - `BestMember` ordering and tie-breaks
  - sense and disarm roll math
  - disarm expiry
- **Module tests (fake store and fake roster):**
  - choose, confirm, and refuse a second choice
  - grants applied once, and re-applied harmlessly after reload
  - admin reset
  - companion archetype from config and set once through `company`
  - roster pruning
  - autoskill toggles persisted
  - `trap sense` and `trap disarm` outcomes, including backfire
  - disarm persisted, expiring, and surviving reload
- **Wiring tests through real entry points:**
  - `usercommands.Train` in a real training room refuses a claimed skill
    for the wrong archetype and allows it for the right one; the panel
    shows the lock
  - a real `ScriptActor.LearnSpell` from a script refuses a restoration
    spell for a wizard
  - `usercommands.Picklock` on the real trapped chest springs the trap
    when armed and doesn't after `trap disarm`
  - an ordinary `Go` into a dark room auto-casts floating light for a
    player wizard (mana spent) and for a companion wizard (buff 1000
    present), and does nothing with autoskill off
  - the `company` display shows archetypes
  - the shipped archetype files load with the real skills and spells
- **Review gate:** an independent reviewer subagent over the phase diff,
  with every finding verified and recorded.
- `go test -race ./...`, `make generate`, and `make validate` pass.

## Implementation notes

Deviations from the text above, recorded as they landed.

- **The archetype table is module config, not engine datafiles.** It
  lives in `modules/archetype/files/data-overlays/config.yaml`, not in
  `_datafiles/world/default/archetypes/`. Plugins load after every engine
  datafile (`main.go`: `loadAllDataFiles`, then `plugins.Load`), so the
  module still cross-checks skills and spells at load. This follows the
  Ashveil rule that balance lives in module config, and needs no new
  engine loader.
- **Companion archetypes live on the company record.** They are stored as
  `company.Companion.Archetype`, not as a `Companions` map in the
  archetype registry, so they are persisted and pruned with the roster
  automatically.
  - The module reads them through a new optional
    `company.ArchetypeProvider` seam.
  - `CompanionArchetypes` in the company config maps template 58 to
    warrior. A configured archetype the archetype module doesn't know is
    skipped and logged, never guessed.
- **Admin reset** is a separate admin-only command,
  `archetypereset <character>`, and only works on online characters.
- **Spell-learning gate:**
  - it applies to players only; mobs are never gated
  - it is skipped for a spell the character already knows, which is
    grandfathered
  - `admin spell` calls `Character.LearnSpell` directly, so it bypasses
    the gate
- **The typo fix changes existing content.** With the `illlusion` →
  `illusion` fix, Elara's party-wide `illum` lesson now teaches only
  wizards (and unchosen characters are refused). This is the intended
  gating.
- **Look display:** `look <player>` and `look <companion>` show an
  "Archetype:" line when one is set.
