# Phase 30d2: Physical Wind-Ups and an Ogre with Crushing Blow — Design

Slice two of 30d. Follows [Phase 30d1](2026-09-30-phase-30d1-chant-interrupts-design.md)
(broken chants, shield counters, enemy casters) and its amendment
[30d1b](2026-09-30-phase-30d1b-chant-break-chance-design.md) (a blow breaks
a chant only by chance). Refines the 2026-09-26
[telegraphs and interrupts proposal](2026-09-26-telegraphs-interrupts-design.md)
(handoff §36). Written 2026-09-30. Open decisions were put to the owner
the same day with AskUserQuestion (answers below). No standing "proceed
with your recommendation" instruction was in force.

## Owner decisions (2026-09-30)

1. **Heavier force only (decided before this design, recorded in 30d1b):**
   a wind-up for a physical attack does not break on an ordinary blow.
2. **Only a crit and statuses break a wind-up; a shield bash is a counter
   strike only** (asked this session). The shield bash was on 30d1b's list
   of heavy force, but a foe winding up swings no blow, so no counter can
   fire against it. The owner struck the bash from the list outright.
   Heavy force is now: a critical hit that lands (`AttackResult.CritLanded`),
   or a blow that staggers, knocks down, or stuns (`heavyBlow`). The bash's
   dead "always breaks a chant" branch in `counterBlow` is removed too (it
   could not happen: a character that swings isn't chanting). The docs that
   listed the bash as heavy force are corrected: the 30d1 and 30d1b designs
   (amendment notes), the status log's Next line, table row, and Known
   issues, and `internal/interrupt`'s package comment.
3. **Crushing Blow: a 1-round wind-up, then double damage and a
   knockdown** (the recommendation). The ogre spends one turn winding up
   and does nothing else. At its next turn it swings once, with its
   weapon's damage doubled before armor. The swing goes through the normal
   combat roll, so it can miss or be dodged, and formation reach, 11c's
   interception, and 30c2's guards all apply. A swing that gets through
   the armor knocks the target down (30a).
4. **A broken wind-up is lost, then a cooldown** (the recommendation). The
   blow is cancelled and the turn spent winding up is wasted. The ogre
   swings normally at its next turn and can't start another wind-up for 2
   of its turns. The same cooldown follows a blow that lands.
5. **The ogre: Dark Forest, solitary; enemies only** (the recommendation).
   A new ogre race and a solitary level-22 forest ogre with a new two-handed
   great club. It spawns in one quiet Dark Forest road room (530). Wind-ups
   are mob-template data that only enemies use. Company wind-ups are
   deferred.
6. **Ordinary blows on a wind-up are silent** (the recommendation). No
   "holds" line and no failed `Interrupt`: ordinary blows can never break
   a wind-up, and five company members a round would flood the text and
   the summary's "failed" count.

## Prior-art check (against `master` at `66fc6dc`, 2026-09-30)

- **Events exist.** `combatstream.WindUpStart` and `WindUpLand` were
  declared in 29b and are unused. `Interrupt` with `Status` (a name) is
  what the summary lists as "dealt".
- **One seam per blow** (30d1): `afterBlow` runs after every weapon blow at
  all six sites (the four in `NewRound_DoCombat.go` and 11c's two
  interception resolvers, which 30c2's guards also use). `heavyBlow(r)`
  exists (30d1b).
- **A mob's turn** (`handleMobCombat`): it loses its action to a status
  (`statusCostsAction`), handles a chant, then (on `DefaultAttack`) may run
  a combat command (`activitylevel`), then swings at its aim through the
  gates (`gateMobVsPlayerAttack`, `gateMobVsMobAttack`: reach, interception,
  guards) and `combat.AttackMobVsPlayer`/`AttackMobVsMob`. The gates
  resolve an intercepted blow themselves, through the same `combat.Attack*`
  functions with the same attacking mob.
- **Hold checks** (`holdsAgainstPlayer`, `mobHolds`) and the reach check
  (`resolveAttackTarget`) have no side effects.
- **Aggro is never saved** (`yaml:"-"`). `SetAggro` without a wait gives
  the weapon's `WaitRounds`.
- **No ogre** exists (no race, mob, or big club). The troll race (4) is the
  nearest; the ent (34) is the Dark Forest's solitary giant.

## Scope

### A. Rules (`internal/windup`, new, pure; `internal/interrupt`)

- `windup.Ability`: `Id`, `Name`, `Rounds` (turns spent winding up),
  `Multiplier` (weapon damage before armor), `KnockDown`, and its lines.
  One ability is registered: `crushing-blow` (Crushing Blow, 1 round, x2,
  knocks down). `windup.Get(id)`.
- `windup.Cooldown = 2` (its turns after a blow lands, is wasted, or is
  broken).
- `windup.RollStart(chance, roll) bool`: `roll(100) < chance`, and never
  at 0 or less.
- `interrupt.BreaksWindUp(hit, damage, heavy) bool`: a hit for at least 1
  damage with heavy force.

### B. Mob data (`internal/mobs`)

- `Mob.WindUps map[string]int` (`windups:`): ability id to the percent of
  its turns it starts one. Only an enemy uses it (never a company companion
  or a charmed mob).

### C. The blow (`internal/combat`)

- `combat.Power{Name, Multiplier, Status}`. A provider set by the hooks
  (`SetPowerProvider`) is asked, in `AttackMobVsPlayer` and
  `AttackMobVsMob`, whether this mob's blow is a wind-up's. When it is,
  `calculateCombat` resolves **one strike** (one weapon, one attack, no
  pet). The rolled weapon damage (dice + bonus) is multiplied before a
  crit's bonus and the armor. A strike that gets through adds the status
  (Knocked Down) to `BuffTarget`, and its parentheses lead with the name:
  `(Crushing Blow, 18 damage, knocked down)`. Everything else is as any
  strike: hit, dodge, crit, 30b's wounds (a crushing blow wounds), pain
  reactions.

### D. The turn (`internal/hooks/combat_windup.go`)

Runtime maps on the game loop: `windUps` (mob id to ability, turns left,
and the target named at the start), `windUpCooldown`, and `landing` (the
mob whose blow this turn is the wind-up's). `windUpRoll` is injectable
(`UseWindUpRollForTest`); `newBrawl` pins it so no wind-up starts unless a
test asks.

At a mob's physical turn (`Aggro.Type == DefaultAttack`), before its combat
commands, `windUpTurn(mob)`:

1. **Winding up:** one turn fewer. With turns left it holds and the turn is
   spent. With none left it **lands**: it re-aims at the target it named
   (with no wait) if that one still stands in the room, else at its aim if
   that stands. With neither, the blow is **wasted** (line, event,
   cooldown). Otherwise the room hears the release line, `landing` is set,
   and the turn goes on to the normal swing through the gates. The provider
   hands `combat` the power once, so the swing is the Crushing Blow even
   when a guard steps in (the guard takes it).
2. **Cooling down:** one turn fewer; it swings normally.
3. **Starting:** an enemy with `windups`, a standing aim in the room that
   it reaches (the gates' legality check), no wait, and no battle hold,
   rolls each ability's chance. On a success the room hears the telegraph
   and the turn is spent.

A landing whose swing never happened (its target left reach, a hold) is
told as wasted when the next mob's turn begins. `WindUpLand` is emitted
from `afterBlow` for the landing mob's swing (outcome `hit`, or `miss`
when it missed or the armor took it all, as 29c reads such a blow;
damage).

**Breaking:** in `afterBlow`, a blow on a mob winding up that
`BreaksWindUp` (a crit that landed, a stagger, a knockdown, a stun) breaks
it. The room hears the broken line, an `Interrupt` is emitted (source the
striker, `SpellId` the ability id, `Status` its name, outcome `succeeded`),
the wind-up is dropped, and the cooldown starts. A winding-up mob that
loses its turn to a status from any other source (not a blow) loses the
wind-up at that turn the same way, with no one credited. An ordinary blow
does nothing (decision 6).

**Pruning** at each round's start (`interruptRound`): a wind-up, cooldown,
or landing of a mob gone, dead, out of combat, or no longer on a plain
attack (casting, fleeing) is dropped.

### E. Lines (29c voice; names by 29d's labels and pronouns)

With `great club` from the weapon's simple name (the race's unarmed name
when unarmed):

- **Telegraph:** `The forest ogre plants his feet and drags his great club
  up over his shoulder, eyes on Tamsin Reed. (winding up: Crushing Blow, 1
  round)`. The target is told `…, eyes on you.`
- **Release:** `The forest ogre brings his great club down with all his
  weight.`, then the swing's own hit or miss line, e.g. `…
  (Crushing Blow, 18 damage, knocked down)`.
- **Broken:** `The forest ogre staggers, and the blow dies before it can
  fall. (Crushing Blow interrupted)` (no weapon named, so it reads for
  fists too).
- **Wasted:** `The forest ogre lets his great club fall, with nothing
  left to strike. (Crushing Blow wasted)`.
- **Lost to a status** (not a blow; amended after review, since nothing
  staggered it): `The blow the forest ogre was winding up is lost.
  (Crushing Blow interrupted)`.

### F. Content

- **Race 22, ogre:** large, `unarmedname: huge fists`, `1d6+3`, strength
  3, vitality 2, speed -1; not tameable; band.
- **Item 10022, ogre's great club** (`namesimple: great club`):
  two-handed bludgeoning, `2d6+2`, heavy.
- **Mob 85, forest ogre:** Dark Forest, hostile, `solitary: true`, level
  22, `pronouns: he`, the great club, `windups: {crushing-blow: 35}`.
- **Room 530** (Dark Forest road) spawns one.

### G. Player help and tutorial

- **`help interrupts`** gains a Wind-ups section: what a wind-up is, the
  telegraph, Crushing Blow's numbers (1 round, double damage, knockdown),
  what breaks it (a crit that lands, a stagger, a knockdown, a stun; never
  an ordinary blow or a shield bash), what a broken one costs (lost, 2
  turns' cooldown), and the tactics (hit it hard, keep the frail out of
  its reach, a guardian takes it for the ward). Aliases `wind-up`,
  `windup`, `windups`, `wind-ups`, `telegraph`, `telegraphs`,
  `crushing-blow`, `ogre`.
- **Updated:** `help combat` (a sentence and the link), `help statuses`
  (Crushing Blow knocks down; a stagger, knockdown, or stun breaks a
  wind-up), `help guardian` (a guard takes the Crushing Blow),
  `help battle-summary` (a broken wind-up counts as an interrupt),
  `help formation` (keep the frail out of an ogre's reach).
- **keywords.yaml:** the new aliases under `interrupts`.
- **Tutorial:** the Practice Yard hint that points to `help interrupts`
  names wind-ups too; `TestTutorialHelpPointersExist`.

## Durable model and invariants

- **Nothing is saved.** The wind-up, cooldown, and landing maps are
  runtime-only in `internal/hooks` on the game loop, like 30d1's restart
  set. Aggro is never saved, so a restart or copyover mid-fight drops a
  wind-up with the fight (answers the proposal's save/load criterion).
- **The clock:** nothing advances or fast-forwards time. A wind-up counts
  the mob's own turns inside the combat round.
- **Locks:** none added. The maps are touched only inside `DoCombat` (and
  the combat call it makes) on the game loop. The power provider is set
  once at init.
- **Narration:** the 29c voice, mechanics lowercase in parentheses.

## Constraints and deferrals

- Company wind-ups (a warrior's heavy blow, a strategy option), a
  multi-target sweep across the front line (the proposal's picture), and
  other abilities are deferred. `Rounds` above 1 works but is unused.
- A hand-typed attack by a player is never a wind-up.
- The start roll is per ability per turn; one ability is shipped.
- Ordinary blows on a wind-up are not told (decision 6), so the summary's
  Interrupts line counts only broken wind-ups.
- A spell's damage never breaks a wind-up (as with chants); a spell's
  stun costs the turn and so drops it.
- A mob that shoots (a bow makes its `DefaultAttack` `Shooting`) never
  winds up; shipped `windups` on a shooter fail a content test (review).

## Acceptance criteria

- **Unit:** `windup.Get`, `RollStart` (certain, never, boundary);
  `interrupt.BreaksWindUp` (table); `combat`'s power blow (one strike,
  doubled damage, the knockdown on a landed strike and not on a blocked
  one, the name in the parentheses) through `AttackMobVsPlayer` with a
  provider set.
- **Wiring, through `DoCombat` in the brawl world:**
  - an ogre winds up (telegraph, `WindUpStart`, no swing that turn), then
    lands at its next turn on the named target: release line, the hit's
    `Crushing Blow` parentheses, double damage, the target knocked down,
    `WindUpLand`;
  - a crit on the winding ogre breaks it: the broken line, `Interrupt`
    `succeeded` with `Crushing Blow`, no landing, and the cooldown (no
    new wind-up for 2 turns);
  - ordinary blows leave it whole and silent (no line, no `Interrupt`);
  - a guardian steps in and takes the Crushing Blow for the ward;
  - a target gone before the landing: the blow is wasted and re-aims
    nowhere;
  - a shield bash on the ogre breaks nothing (no bash can reach it; the
    counter's break code is gone);
  - the battle summary lists `Crushing Blow` under dealt interrupts;
  - existing brawls are unchanged (the roll pinned in `newBrawl`);
    `modules/company` looped with `-count` for new flakes.
- **Content:** the ogre race, club, and mob load; the mob's `windups`
  name a registered ability; room 530 spawns it.
- **Player help:** `help interrupts` renders the wind-up section and its
  aliases; the pages above updated; `TestTutorialHelpPointersExist`
  passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded in `docs/PROJECT_STATUS.md`.
