# Phase 30d1: Broken Chants, Shield Counters, and Enemy Casters — Design

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md);
refines the 2026-09-26 [telegraphs and interrupts proposal](2026-09-26-telegraphs-interrupts-design.md)
(handoff §36). Written 2026-09-30. Open decisions were put to the owner
the same day with AskUserQuestion (answers below). No standing "proceed
with your recommendation" instruction was in force.

## Owner decisions (2026-09-30)

1. **Two slices, chants first.** 30d1 (this doc): chants, interrupts, the
   shield counter, enemy casters. 30d2 (next): physical wind-ups (mob
   data) and an ogre with Crushing Blow.
2. **A chant breaks whenever the chanter takes physical damage.** No
   pressure meter, thresholds, or concentration rolls (the owner's answer,
   replacing the proposal's rules 2–4). Physical damage is **a weapon blow
   that does 1 or more damage**: melee, a sling or bow, a blow a guard or
   11c's interception redirected. A miss, a dodge, a status tick
   (bleeding, burning), and a damaging spell don't break it (the owner
   took the recommended option).
3. **An interrupted spell of a player or companion is cancelled, with half
   its mana back** (rounded down). The caster swings or recasts next round
   as its strategy (or its player) says.
4. **The shield bash is a counter** (the owner's idea, asked "what do you
   think?"): when an attack on a shield-bearer fumbles (misses in any way,
   a dodge included), the bearer may slam the shield back into the
   attacker, and the bash can stun. It is not an action anyone chooses and
   takes no turn. The recommendation adopted here (limits and numbers,
   flagged for the owner in `docs/PROJECT_STATUS.md`):
   - melee only (the attacker is in the room and its weapon is not a bow
     or sling: `shooting` subtype);
   - the bearer can act (standing, not `no-combat`, not knocked down or
     stunned, not chanting);
   - at most once per bearer per combat round;
   - 50% of fumbled attacks; 1d4 damage; a bash stuns 25% of the time
     (30a's Stunned, buff 1107);
   - anyone holding a shield counters: players, companions, and enemies.
5. **The goblin hexer fights in the Dark Forest, with the forest imps.**

The proposal's other outcomes (delay a round, recovery penalty) are not
built. Enemy casters restart from the first word, as the proposal says
(decision 3 covers only the company).

## Why the proposal's pressure model changed

The owner's rule makes chants a formation question, not a meter: keep
healers and casters where blows can't reach them (11c's back rows), and
guard them (30c2). An enemy caster in reach may be broken by any hit (by chance since 30d1b); one in
its back row must be reached by a bow, a sling, or a guard's absence.
30d2's physical wind-ups (the ogre) break only on heavier force (the
owner's decision; see Deferrals). 30d1b later made a chant's break a
chance.

## Prior-art check (against `master` at `d2f7b89`, 2026-09-30)

- **Chants exist.** A spell sets `Aggro{Type: SpellCast, RoundsWaiting}`
  (`Character.SetCast`); the combat loop runs `onWait` until the count
  is 0, then rolls and casts (`handlePlayerCombat`/`handleMobCombat`,
  "START HANDLING MAGIC"). Mana is spent at the start
  (`skill.cast.go`, `mobcommands/cast.go`, `startCast` in
  `combat_strategy.go`). `endCast` returns an automatic caster to its
  aim. `aid` and `tame` use the same chant and so break the same way.
- **Aggro is never saved** (`yaml:"-"`), so a restart or copyover drops
  every chant; nothing about a chant or a restart needs to persist
  (answers the proposal's save/load criterion: "a copyover mid-fight
  resets the action").
- **One seam per blow.** Every weapon blow goes through one of six sites,
  each calling `emitAttack` and then sending the blow's lines: the four in
  `NewRound_DoCombat.go` and 11c's two interception resolvers
  (`combat_formation.go`), which 30c2's guards also use. 30d1 adds one
  call after each site's lines, `afterBlow`, so a break or counter reads
  after the blow that caused it.
- **Events are declared** (29b): `Interrupt`, `OutcomeSucceeded`,
  `CastStart/Progress/Complete`. The summary already tallies
  `InterruptsDealt/Failed/Taken`; with no failed interrupts in this model
  the Interrupts line drops "failed".
- **Shields:** `GetDefense` treats a non-weapon offhand item with damage
  reduction as a shield (`no-block` lifts it while stunned). 30d1 moves
  that test to `Character.HasShield()` and reuses it.
- **Statuses (30a):** `status.Grounded` (knocked down or stunned); a
  stun is `AddBuff(status.Stunned)` and lands only on a holder in a fight
  (`statusBuffLands`).
- **Enemy casters:** no hostile shipped mob casts. A mob casts through
  its `combatcommands` (`cast <spell>` at a turn, by `activitylevel`),
  aiming at its current foe; 30c1's goblin race aims by `casters`
  (company healers and casters first). The company's `casters` focus
  already ranks a chanting foe first, then any with a spellbook.

## Scope

### A. Broken chants (`internal/interrupt`, `internal/hooks/combat_interrupt.go`)

- **The rule** (`interrupt.Breaks`): a blow breaks the target's chant when
  it hit, did at least 1 damage, and the target is chanting (`Aggro.Type ==
  SpellCast`) and not already waiting to restart.
- **A player or companion** (a company member, or any player): the spell
  is cancelled (`endCast`: an automatic caster turns back to its aim),
  half its cost comes back (`interrupt.Refund`), and lines:
  - you: `The blow breaks your chant, and Minor Heal is lost. (Minor Heal
    interrupted, 1 mana back)`;
  - the room: `Brother Oswin's chant breaks off under the blow. (Minor
    Heal interrupted)`.
- **Any other mob** (an enemy): the chant stops, and at its next turn it
  starts again from the first word, costing nothing (the mana was spent
  once), with the spell's full chant time:
  - the room, at the break: `The goblin hexer's chant breaks off under the
    blow. (Withering Hex interrupted)`;
  - the room, at its next turn: `The goblin hexer starts the chant again
    from the first word. (chanting: Withering Hex, 2 rounds)` (generic, so
    it suits any enemy; amended after review).
  A mob waiting to restart isn't chanting (a blow then breaks nothing, and
  it may counter with a shield), though the `casters` rule still ranks it
  first (it is about to chant). If none of its targets still stands (or
  the spell is no longer loaded), it gives the spell up and loses that
  turn; the next round's upkeep aims it again (amended after review: the
  combat loop can't hand a mob with no aim on to a swing that turn).
- **Events:** `Interrupt` (source the attacker, target the chanter,
  `SpellId`, `Status` the spell's name, outcome `succeeded`), and
  `CastComplete` with a new outcome `interrupted`; a restart emits
  `CastStart`.
- **Summary:** `Interrupts: dealt 2 (Withering Hex, Withering Hex) · taken
  1`; "failed" is shown only when non-zero.

### B. Shield counters (`internal/interrupt`, `internal/hooks/combat_interrupt.go`)

- **When:** after a blow that missed (`!Hit`), if the target
  `CanCounter`: holds a shield (`HasShield`, so not while stunned), is
  standing and able (`Health ≥ 1`, not `no-combat`, not
  `status.Grounded`, not chanting), in the attacker's room, the
  attacker's weapon isn't `shooting`, and the bearer hasn't countered this
  combat round (a runtime set cleared each round).
- **Roll** (`interrupt.RollCounter`, with a roll function tests replace):
  50% to bash; damage 1d4; 25% of bashes stun.
- **Effect:** the damage, a stun buff when rolled, and the attacker's
  fall resolved in this round (`roundExtraPlayers`/`roundExtraMobs`). A
  bash never triggers a counter of its own. (It once also broke the
  attacker's chant, which could never happen: the attacker had just swung.
  Removed by [30d2](2026-09-30-phase-30d2-windups-design.md): a bash is a
  counter strike only, owner, 2026-09-30.)
- **Lines:** you: `You turn the blow and drive your shield into the first
  cutthroat. (shield bash, 3 damage, stunned)`; struck (a player): `Tamsin
  Reed turns your blow and drives her shield into you. (shield bash, 3
  damage)`; the room: `Tamsin Reed turns the blow and drives her shield
  into the first cutthroat. (shield bash, 3 damage)`.
- **Events:** an `Attack` event with `WeaponType: "shield-bash"` (so the
  summary counts its damage), and `StatusApplied` for a stun.

### C. Enemy caster content

- **Spell `hex`** (Withering Hex): `harmsingle`, cost 8, `waitrounds: 1`
  (a two-round chant, as Magic Missile), difficulty 40, 2d6+2 damage.
  Script with the chant, keep-chanting, and release lines in the 29c
  voice, mechanics in parentheses.
- **Mob 70, goblin hexer** (Dark Forest, `raceid: 5` goblin, level 18,
  hostile, `groups: [forest-creature]`, spellbook `hex`, `combatcommands:
  ['cast hex']`, `activitylevel: 40`). Its goblin race aims by `casters`.
- **Spawns:** Dark Forest rooms 402 and 531 add a hexer beside their
  imp, so 29b2 groups them.

### D. Player help and tutorial

- **New `help interrupts`** (aliases `interrupt`, `interrupted`, `chant`,
  `chants`, `concentration`, `shield-bash`, `shieldbash`, `counter`,
  `counters`, `hexer`; `chanting` stays `help narration`'s): chants break on a damaging blow; what
  a company caster loses (half mana back) and an enemy's restart; the
  shield counter's rules and numbers; tactics (keep casters in the back
  row, guard them, focus `casters`, bows and slings for a back-row
  hexer).
- **Updated:** `help combat` (hub link and a sentence); `help cast`
  (a chant breaks); `help spell` if it describes casting in combat;
  `help strategy` and `help tactics` (the `casters` rule now has enemy
  casters; healers break under blows); `help statuses` (a shield bash can
  stun); `help guardian` (guarding a caster keeps its chant whole);
  `help battle-summary` (the Interrupts line); `help formation` (casters
  in the back).
- **keywords.yaml:** `interrupts` under the combat help category, with
  its aliases.
- **Tutorial:** the Practice Yard (combat) lesson points to `help
  interrupts` beside its casting hint; `TestTutorialHelpPointersExist`.

## Durable model and invariants

- **Nothing is saved.** Chants were never saved; the restart set and the
  counter set are runtime maps on the game loop in `internal/hooks`,
  pruned each round (a mob gone, no longer in a fight). A restart or
  copyover mid-fight drops them with the chant itself.
- **The clock:** nothing advances or fast-forwards time; everything
  happens inside the combat round's existing blows and turns.
- **Locks:** none added; the maps are touched only inside `DoCombat` on
  the game loop. `internal/battle`'s mutex is not held across any of it.
- **Narration:** 29c voice, mechanics lowercase in parentheses; names by
  29d's labels and pronouns.

## Module

- **`internal/interrupt`** (new, pure): `Breaks`, `Refund`, `Counter`
  eligibility, `RollCounter` (chances as package values), tests.
- **`internal/characters`:** `HasShield()` (and `GetDefense` uses it).
- **`internal/combatstream`:** `OutcomeInterrupted`; the summary's
  Interrupts line.
- **`internal/hooks`:** `combat_interrupt.go` (`afterBlow`, the break,
  the restart, the counter), called at the six blow sites; the restart
  at the head of a mob's magic turn; round reset of the counter set.
- **Data:** `spells/hex.yaml`/`hex.js`, `mobs/dark_forest/70-goblin_hexer.yaml`,
  rooms 402 and 531.
- **Help and tutorial:** section D.

## Constraints and deferrals

- **30d2:** physical wind-ups and the ogre. **Decided by the owner
  (2026-09-30):** a wind-up for a physical attack does not break on an
  ordinary blow, only on heavier force: a critical hit, a stagger, a
  knockdown, or a stun (a big foe is struck by everyone). **Amended by
  [30d2](2026-09-30-phase-30d2-windups-design.md) (owner, 2026-09-30):** a
  shield bash is a counter strike only and breaks neither a wind-up nor a
  chant.
- **Amended by [30d1b](2026-09-30-phase-30d1b-chant-break-chance-design.md)
  (owner, 2026-09-30):** decision 2 above now breaks a chant by chance,
  40 + 2 × (damage × 100 / max health) held to 40–90%, and always on heavy
  force (a crit, a stagger, a knockdown, a stun; the shield bash was
  struck from this list by 30d2); a held
  chant is told and emits a failed `Interrupt`.
- Not built: the proposal's pressure meter, concentration, delayed or
  penalised recoveries; anti-caster abilities; casting on the battle
  view's grid (32g2's deferred item stands); per-spell interrupt outcomes.
- The player's `cast` command and a mob's `cast` still emit no
  `CastStart` (only 32d's automatic casts do); a restart does.
- Charmed pets that aren't companions restart like enemies (they are not
  in a company); accepted.

## Acceptance criteria

- **Unit (`internal/interrupt`):** `Breaks` (hit with damage breaks; a
  miss, 0 damage, or not chanting doesn't); `Refund` (half, rounded
  down); `CanCounter` (each condition); `RollCounter` with a fixed roll.
- **Wiring, through `DoCombat` in the brawl world:**
  - a companion healer's chant breaks on an enemy blow: `Interrupt` and
    `CastComplete(interrupted)`, the line, half the mana back, no heal;
  - a player's own `cast` breaks on a blow the same way;
  - a light blow that does damage breaks it; a miss does not, and a heal
    chanted out of reach completes;
  - an enemy caster's chant breaks on a company blow and restarts at its
    next turn (line, `CastStart`), and lands after its full chant time;
  - a blow redirected by a guard (30c2) breaks the guardian's own chant,
    not the ward's;
  - a shield-bearer countered a missed melee blow: damage, the line, an
    `Attack` event `shield-bash`; forced stun applies Stunned; at most
    once a round; none against a bow; none while stunned; an enemy with a
    shield counters the player;
  - a counter that kills resolves the death that round;
  - the battle summary's Interrupts line.
- **Content:** the hexer and `hex` load; a hexer in a brawl casts `hex`
  at a company healer by `casters`; the Dark Forest rooms spawn it.
- **Player help:** `help interrupts` renders with its aliases; the pages
  above updated; `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate` pass; the
  independent review is recorded in `docs/PROJECT_STATUS.md`.
