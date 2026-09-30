# Phase 30a: Status Effects and Critical-Hit Effects — Design

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md);
refines the 2026-09-26 proposal (handoff §36 item 7; in git history at
`5f46bb4`, removed once this shipped). Written 2026-09-29. Open decisions were put to the owner the same day
(answers below); no standing "proceed with your recommendation" was in force.

## Goal

A critical hit does more than add damage: its secondary effect matches the
weapon. Effects are ordinary timed buffs that later phases (interrupts,
wounds, morale, guards) read through flags. Durations are counted in
**combat rounds** (the 8-second round of 29f), which also retunes the
half-length-buff problem 29f recorded.

## Owner decisions (2026-09-29)

1. **Scope:** all eight statuses of the proposal, plus extras where they are
   good ideas. Extra shipped: **Hobbled** (a whip's crit: cannot flee, slowed).
2. **Stacking:** bleeding stacks to 3 and refreshes its duration; the others
   refresh only.
3. **Fight end:** statuses clear when the fight ends (30b will turn some into
   wounds).

## Prior-art check

- `internal/buffs` already models timed effects, flags, and stat mods, and
  `items.Damage.CritBuffIds` is a per-weapon crit override that
  `combat.calculateCombat` puts in `AttackResult.BuffTarget`, which all six
  attack sites in `internal/hooks` already apply and `attackEvents` already
  reports as `StatusApplied`. **Reuse: the crit table feeds `BuffTarget`.**
- The `no-combat`, `no-flee`, and `cancel-on-water` flags exist; `no-combat`
  skips a holder's whole turn but also removes it from company upkeep
  (`canFight`), so a new flag is used to lose a single action.
- `combatstream` already has `StatusApplied`, `StatusExpired` kinds and
  `Event.BuffId/Status`; nothing emits `StatusExpired` yet. A `StatusTick`
  kind is added.
- Buffs tick per game round (4 s); combat runs every second game round
  (`CombatEveryRounds: 2`). A game-round buff would last half as many combat
  rounds, so **statuses are ticked by the combat loop, not by game rounds.**

## Scope

### Buff data (ids 1100–1109, `_datafiles/world/default/buffs/`)

`triggerrate` is set so large that game-round triggering never fires;
`triggercount` is the number of **combat-round ticks**, which the combat loop
decrements. A status is *alive after a tick* while `TriggersLeft ≥ 1`.

| Id | Status | Count | Effect | Flags |
|---|---|---|---|---|
| 1100 | Bleeding | 3 | 1 damage per stack per tick; stacks to 3 | `combat-status`, `bleeding` |
| 1101 | Staggered | 2 | loses its next action | `combat-status`, `lose-actions` |
| 1102 | Knocked down | 3 | loses its next action; Speed −4 while down (easier to hit) | `combat-status`, `lose-first-action` |
| 1103 | Armor broken | 60 | defense halved until the fight ends | `combat-status`, `armor-broken` |
| 1104 | Exposed | 3 | crit chance against it +25 points for 2 rounds | `combat-status`, `exposed` |
| 1105 | Burning | 4 | 2 fire damage per tick | `combat-status`, `burning`, `cancel-on-water` |
| 1106 | Overloaded | 4 | casting −30, Speed −3 | `combat-status` |
| 1107 | Stunned | 3 | loses its next two actions; can't dodge or shield-block (2026-09-30) | `combat-status`, `lose-actions`, `no-dodge`, `no-block` |
| 1108 | Hobbled | 4 | cannot flee; Speed −3 | `combat-status`, `no-flee` |

(1109 reserved.) **Changed 2026-09-30 (owner):** knocked down lasts 2
rounds (count 3: its next action, then down one more round). Stunned
stays at 2 rounds (count 3: its next two actions) and, while stunned, the
holder can't dodge or block with a shield (flags `no-dodge`, `no-block`;
armor still counts). See `docs/PROJECT_STATUS.md`. "Exposed" is a duration, not "the next hit": consuming it
would need every attack site to remove a buff after a hit, and the combat
resolver works on copies. Recorded deviation from the proposal.

### Crit effects (`internal/status`)

A Go table keyed by `items.ItemSubType`, the weapon's own `CritBuffIds`
overriding it (the proposal said "in data": the override *is* the data
path, and a table test guards the ids). Unarmed and generic subtypes have no
effect.

| Weapon subtype | Effect |
|---|---|
| slashing, claws | Bleeding |
| stabbing | Bleeding, applied twice (deep bleeding: 2 stacks) |
| bludgeoning | Staggered |
| cleaving | knocked down 40 %, armor broken 40 %, stunned 20 % |
| shooting | Exposed |
| whipping | Hobbled |

Deferred: the shield row (needs a shield-attack action: 30c/30d) and a
sling-specific stagger (no sling subtype; a per-weapon override covers it).
Spells never critical hit (owner, 2026-09-27); Burning and Overloaded are
applied by a spell's normal script: **Shower of Sparks** applies Overloaded
to each target; Burning is applied by a new fire spell if the roster
gains one, and by a per-weapon override (a flaming weapon) today.

### Combat loop (`internal/hooks/combat_status.go`)

`statusPass()` runs in `DoCombat` after upkeep and before any blow:
for every online player and mob carrying a `combat-status` buff it
1. clears them silently if no open fight stands in the holder's room (a stray
   left by a restart, or a flight out of the fight);
2. otherwise ticks each: decrement `TriggersLeft`, bleed/burn damage, a
   `StatusTick` event and line, and on reaching 0 a `StatusExpired` event and
   line;
3. resolves any fall it caused through `handleAffected` at once.

`handlePlayerCombat`/`handleMobCombat` skip a holder's turn when it has a
`lose-actions` status alive, or a `lose-first-action` status on its first
alive round, with a line and a `StatusTick` event (outcome `lost-action`).
The company strategy pass skips the same holders. Fights ending
(`fightSides.end`) clear every participant's statuses.

### Engine changes

- `BuffSpec.CombatRounds` (see constraints below).
- `flee` refuses while a `no-flee` buff (Hobbled, and the existing
  Hamstrung, which never enforced it) is on the fighter.
- `buffs.Buff.Stacks`, `BuffSpec.MaxStacks`: `AddBuff` on a live buff refreshes
  its count and adds a stack up to the maximum.
- `Character.GetDefense`: halved while `armor-broken`.
- `combat.Crits`: +25 points against `exposed`.
- `damageSuffix` names the statuses a crit applies; the crit table is applied
  in `calculateCombat` where `critBuffs` already are.

### Text (dark voice, mechanics lowercase in parentheses)

- Hit: `... (critical hit, 6 damage, bleeding)`.
- Bleed tick: `The first skirmisher bleeds. (1 damage, bleeding)`; you: `You bleed. (1 damage, bleeding)`.
- Lost action: `The first skirmisher reels, and loses the action. (staggered)`.
- Expiry: `The first skirmisher's bleeding stops.`

## Durable model and invariants

Nothing here advances the world clock: ticking rides the existing combat
round. Statuses are buffs and so persist with the character, but a fight does
not survive restart/copyover (battle state is runtime-only, 29b), so the pass
above clears any status with no open fight in its room; a copyover cannot
carry a stale bleed. No new files, no locks: the pass runs inside `DoCombat`
like the rest of the loop.

## Constraints and deferrals

- Not built: guards/`can't guard` (30c), leaps (30f), wound conversion (30b),
  interrupt pressure (30d): the statuses set flags; those phases read them.
- `BuffSpec.CombatRounds` marks a buff whose count is combat rounds;
  `buffs.GetDurations` converts it to game rounds (× `CombatEveryRounds`) so
  `conditions` and GMCP show the real time left.
- The live battle view (32g2) does not show statuses yet (its own deferred
  item).

## Acceptance criteria

- A forced crit through the real round, for each weapon subtype, applies that
  subtype's status, emits `StatusApplied`, and names it in the hit line.
- Bleeding ticks each combat round with a line and event, stacks to 3, ends
  at fight end, and can kill (death resolved that round).
- A staggered or knocked-down combatant loses its next action; a stunned one
  loses two; each with a line and event.
- Armor broken halves defense; exposed raises crit chance; both tested.
- Statuses with no open fight are cleared; expiry emits `StatusExpired`.
- Sparks applies Overloaded through a real cast.
- Player help: `help statuses` (aliases `status-effects`, `bleeding`,
  `stagger`, `knockdown`, `stun`, `burning`, `exposed`, `armor-break`,
  `hobbled`) in `keywords.yaml`, linked from `help combat` and `help
  narration`; the tutorial's practice-fight lesson points to it;
  `TestTutorialHelpPointersExist` and a render test pass.
