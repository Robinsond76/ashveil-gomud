# Phase 30c: Company Tactics — Design

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Rescoped 2026-09-29 (owner) after 32d shipped the casting and role slice
([32d design](2026-09-28-phase-32d-auto-combat-design.md)). This replaces
the 2026-09-26 draft (in git history). Covers handoff §36 items 6 (guard
reactions) and 9 (target-selection personality), formerly deferred 11d.

## Owner decisions

Confirmed in this session (2026-09-29); the owner accepted each
recommendation.

1. **Drop the roles 32d already covers.** Duelist, hunter, cleric, wizard,
   and rogue are not added: 32d's fighter/healer/caster roles and eight
   target rules cover them. Only **guardian** is new (slice two).
2. **The focus may change mid-battle**, the one exception to 32d's "nothing
   mid-battle but `flee`" (rule 5). It is company-wide, temporary (that
   battle only), leader-only, and rate-limited by a one-round cooldown.
   The web Combat tab gets clickable focus buttons calling the same path.
3. **Rotate the wounded is deferred.** Interrupts and mercy toggles arrive
   with 30d and 30e. A retreat order is out of scope.
4. **Two slices:** 30c1 (focus, healing threshold, enemy personalities, UI)
   is independent of 30a. 30c2 (guardian, guards) follows, ideally after
   30a, but can ship first without the knockdown rule.

## Prior-art check

- **Targeting:** `strategy.Pick`/`Choose` (rules), `enemyparty.Aim`,
  `RuleChoice` (32d). `Attacker` carries the rule; the hooks build it in
  `combat_battle.go`, `combat_engagement.go`, and `combat_strategy.go`.
  The focus is one more input to that construction, not a new picker.
- **Healing:** `strategy.Decide` hard-codes half health for a healer.
- **Enemies:** aim at the weakest reachable member
  (`engagement.AssignTarget(..., Weakest, ...)`, 29a); no personality.
- **Mid-battle lockout:** 32d refuses `strategy` changes and everything
  but `flee` in a battle (`help combat`, E in the 32d design).
- **Web client:** 32g2's Combat tab and `Company.Battle` message show the
  live battle; 32g's Setup holds the member menu.
- **Events:** `combatstream` already defines `GuardUsed`/`GuardExhausted`.
  A target change is an existing event.

## A. `company tactics` (30c1)

A durable setting on the company record (survives restart and copyover):

| Setting | Values | Default |
|---|---|---|
| **Focus** | `none`, `leader`, `casters`, `nearest`, `weakest`, `strongest`, `wounded` | `none` |
| **Healing** | 10–90 percent (steps of 10) | 50 |

- `company tactics` shows both and each member's strategy (as `strategy`);
  `company tactics focus <rule>`, `company tactics healing <percent>`,
  `... default`.
- **Focus `none`:** every member follows its own strategy rule (32d).
- **Focus set:** every living company member in the battle, the player
  included, aims by the focus instead of its own rule. It overrides *who*
  is aimed at only; roles stay (a healer still heals, a caster still
  casts its spell, at the focus's choice). Reach still binds: a focus
  choice out of reach falls to the nearest foe reachable, as in 32d.
- `casters` is a new rule (also usable in `strategy`): spell-casting
  foes first (mobs with spells or a cast in progress), then the nearest.
  `leader`, `weakest`, and the rest are 32d's rules, unchanged. `assist`
  and `defend` are not offered as focus (they follow a person).
- **Healing threshold:** replaces the fixed half in `strategy.Decide`
  (a `Situation.HealBelow` percent). A downed player still counts as the
  most hurt.
- Changing the **healing threshold is refused in a battle** (locked, as
  the rest of setup); only the focus is open.

## B. Focus mid-battle (30c1)

- **Scope:** a battle-scoped override held with the battle (runtime only,
  like battles, aims, and the 32d restore map; restart starts fresh
  battles from the saved focus). It never writes the saved setting.
- **Who:** the company's leader (the player who owns the company). Others
  in the room see the line.
- **Cooldown:** one change per combat round. A change takes effect at the
  next round's upkeep, so aims turn once, at the boundary. A change made
  during the cooldown is refused with a line ("Your company is still
  turning; try again next round.").
- **Command:** the same `company tactics focus <rule>` works in a battle
  (the one setting 32c/32d's lockout allows); `... focus default` returns
  to the saved value. Every other command stays refused.
- **Aims turn** through the existing upkeep (`retarget`), sticky as usual:
  a member keeps a legal target only if the new focus's choice equals it
  or the old aim is gone; otherwise it turns, with 29c's "turns toward"
  line.
- **Narration:** one line per change, 29c voice: "You call the company
  onto the goblin hexer." Names the choice when it is visible, else the
  rule ("You call the company onto their weakest").
- **Events:** a `focus-change` event (who, rule) in the 29b stream.

## C. Web Combat tab (30c1)

- `Company.Battle` gains `Focus` (current effective rule), `SavedFocus`,
  and `FocusReady` (cooldown clear). Buttons for the seven values, the
  current one marked, disabled while not ready or for non-leaders.
- A click sends the `company tactics focus <rule>` command through the
  client's existing command path; no new server message.
- Keyboard and screen-reader friendly, like the rest of the dock; a live
  region announces the confirmation line.

## D. Enemy personalities (30c1)

Data, not code, per mob template or race (template wins over race):

```yaml
targeting: wounded        # a strategy rule name
targeting-noise: 20       # percent of aims that pick a random reachable foe
```

| Enemy | Rule | Noise |
|---|---|---|
| wolf | `wounded` | 10 |
| ogre | `nearest` | 5 |
| goblin hexer | `casters` | 10 |
| wild beast | `weakest` | 40 |

- No `targeting` means today's `weakest`, no noise.
- Enemies pick through the same `strategy.Pick` with reach; noise is one
  seeded roll per (re)aim (not per round): a test seeds it.
- "Hunts the isolated" (the draft's wolf) needs adjacency data; use
  `wounded` for now.
- No focus for enemies; a group's aim is the aim of each member.

## E. Guardian and guards (30c2)

- **Role `guardian`** added to `strategy` roles (never a default; set with
  `strategy <who> guardian`). Its **ward** is another company member
  (`strategy <who> guard <other>`); with no ward it guards the most
  hurt member by fraction. Otherwise it fights as a fighter.
- **A guard** redirects one blow aimed at the ward onto the guardian
  (an interception, now narrated, 11c's mechanics). A guardian has **2
  guards** per battle; a spent guard returns one per 2 rounds, up to 2
  (recommended; confirm in the plan). Events: `guard-used`,
  `guard-exhausted` (already defined).
- **Lost while knocked down or stunned** (30a). Until 30a ships, the
  condition never applies; the code is written to the 30a hook.
- Only companions and the player are guardians; a guardian must be able
  to stand (not downed, not `no-combat`).

## Module

- **`internal/strategy`:** `Casters` rule; `Guardian` role; `HealBelow` in
  `Situation`; `Pick` handles noise (a seeded source passed in).
- **`internal/company`:** `Tactics` (focus, healing) on the record and
  the summary; `modules/company`: the `company tactics` command.
- **`internal/battle`:** the battle-scoped focus and its cooldown round.
- **`internal/enemyparty`, `internal/hooks`:** `Attacker` takes the
  effective rule (focus over strategy); personalities for mob aim;
  guard interception (30c2).
- **`internal/mobs`, races:** `targeting`, `targeting-noise` fields.
- **`modules/webclient`** (files under `_datafiles/html`): focus buttons,
  `Company.Battle` fields.
- **Help and tutorial:** below.

## Invariants

- **The clock:** nothing advances or fast-forwards time; the cooldown counts
  combat rounds the loop already runs.
- **Restart and copyover:** saved focus and healing persist on the company
  record; the mid-battle override, cooldown, and guard counts are runtime
  only (battles are).
- **Game loop and locks:** the override and counts are read and written on
  the game loop; the registry keeps its own mutex, never held across the
  engine; `internal/battle`'s mutex is not held across any of it.
- **Narration:** 29c voice, mechanics in lowercase parentheses.

## Acceptance criteria

- **Unit:** the `casters` rule; noise with a seeded source; healing
  threshold in `Decide`; parsing of focus and threshold; the cooldown.
- **Wiring** (real commands, `DoCombat` rounds):
  - saved tactics survive a save/load; healing threshold changes when a
    healer acts;
  - a focus overrides every member's own rule, reach still binds, roles
    still act;
  - `company tactics focus` works in a battle only for the leader, once
    per round, applies at the next upkeep, reverts at the battle's end;
    every other tactics/strategy/formation command is refused in a
    battle;
  - enemy personalities pick by rule, template over race, noise seeded;
  - 32d's, 29a's, 29b2's, and 32c's fight tests pass;
  - 30c2: a guard redirects a blow and is spent, then returns; an
    ungarded ward is struck; guard events fire.
- **Web:** the buttons render, mark the current focus, disable on
  cooldown or for non-leaders, and send the command (checked in Chromium,
  as 32g2).
- **Player help:** a new `help tactics` (aliases `focus`, `company
  tactics`); updates to `help strategy` (the company focus overrides
  target rules), `help combat` (hub link; the focus is the one mid-battle
  change), `help targeting`, `help webclient` (the buttons), and `help
  guardian` for 30c2; `keywords.yaml`; the Combat lesson points to
  `company tactics`; `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate` pass. The
  independent review is recorded.

## Deferred

- Rotate the wounded; a retreat order; interrupt and mercy toggles (30d,
  30e); hunting the isolated (needs adjacency).
- Per-spell gambit lists and openers (as 32d).
