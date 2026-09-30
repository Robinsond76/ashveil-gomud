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

## Owner decisions for 30c1 (2026-09-30)

Put to the owner with AskUserQuestion in this session; the owner chose
the recommended option each time. There is no standing "proceed with your
recommendation" instruction.

5. **A mid-battle focus change turns everyone at once.** At the next
   round's upkeep every member who can reach the new focus's choice turns
   to it, even with its current foe still standing (three companions on
   the archer all turn onto the chief). After that round, aims stick as
   usual.
6. **Enemy personalities act only when an enemy re-aims.** As today, an
   enemy keeps a foe it can reach; its personality decides when it first
   joins, when its target falls, or when it can no longer reach it.
7. **Shipped personalities are by race**, a mob file overriding its race:

   | Race | Rule | Noise |
   |---|---|---|
   | canine (wolves) | `wounded` | 10 |
   | undead (skeleton, lich) | `nearest` | 0 |
   | goblin (forest imp) | `casters` | 15 |
   | insect (spiders, cave stalker), giant spider | `weakest` | 25 |
   | human (ruffians, acolytes) | `weakest` | 10 |

8. **An enemy's `casters` goes for** anyone mid-chant first, then members
   whose role is healer or caster, then the nearest it can reach.

## Prior-art check (against `master` at `6a57216`, 2026-09-30)

- **Targeting (32d):** `strategy.Pick`/`Choose` choose by rule over plain
  `strategy.Foe` values (formation order breaks ties); `enemyparty.Aim`
  and `enemyparty.RuleChoice` adapt a live group; `enemyparty.Attacker`
  carries the rule, built by `PlayerAttacker`/`CompanionAttacker` from
  `MemberStrategy`. Every company aim goes through those two builders:
  the upkeep's `retarget` (`combat_engagement.go`), a lone player's
  `turnAlone` and `keepOnBattle` (`combat_battle.go`), and a caster's
  spell target (`sideActors` in `combat_strategy.go`). So the focus is
  applied in the two builders, not in each caller.
- **Stickiness:** `retarget` keeps a legal target except under a rule
  that `ReaimsEachRound` (assist, defend). A one-round "refocus" flag
  joins that condition.
- **Healing (32d, 30b):** `strategy.Decide` hard-codes half health; since
  30b the hooks pass each ally's wound limit (`HealthLimit()`) as its
  `MaxHP`, so the threshold is a share of the limit.
- **Enemies:** `keepPartyEngaged` gives an enemy with no legal target one
  by `engagement.AssignTarget(..., Weakest, ...)` over the company's
  members (leader first, then companions), legal by 11c's reach. No
  personality. A lone player's foes are turned by `rallyIdleFoes` (one
  target, so no choice to make).
- **Statuses (30a):** `status.LostAction` already stops a staggered or
  knocked-down member from acting; the focus changes only aims, so a
  member who loses its action simply keeps the new aim. Nothing in 30c1
  depends on 30a beyond that.
- **Durable settings (32d):** `modules/strategy` keeps each player's
  strategies in its own plugin file, saved on change with rollback, read
  by the engine only through `internal/strategy`'s `Provider` seam. The
  company record (`modules/company`) exists only for players with
  companions or claims, while a solo cleric has a healing threshold too.
  **So tactics are stored beside the strategies** (the `strategy` file,
  a `Tactics` map), not on the company record; the command is still
  `company tactics` (in `modules/company`), writing through a new
  `strategy.TacticsProvider` seam. This is the one departure from the
  2026-09-29 draft ("on the company record"); it survives restart and
  copyover the same way.
- **Mid-battle lockout (32c/32d):** `usercommands.InBattle` guards
  `strategy`, `formation`, and the rest.
- **Battles (29b2):** `internal/battle` holds each player's battle
  (runtime, mutex-guarded, never held across the engine). Each player
  owns their own company and their own battle, so "leader-only" holds by
  construction: the only player who can change a battle's focus is the
  one whose battle it is.
- **Web (32g/32g2):** `Company.Battle` (`modules/gmcp`) feeds the Combat
  tab's battle view; `window-combat.js` has a Setup view with the member
  menu and sends commands with `Client.SendInput`.
- **Events:** `combatstream` has `TargetChange`; `GuardUsed` and
  `GuardExhausted` are defined for 30c2. A `FocusChange` kind is new.
- **Shipped enemies:** no hostile shipped mob casts spells, so the
  company's `casters` focus falls to the nearest until 30d's enemy
  casters; it is still offered (it costs nothing and 30d fills it in).

## A. `company tactics` (30c1)

A durable setting per player (survives restart and copyover):

| Setting | Values | Default |
|---|---|---|
| **Focus** | `none`, `leader`, `casters`, `nearest`, `weakest`, `strongest`, `wounded` | `none` |
| **Healing** | 10–90 percent (steps of 10) | 50 |

- `company tactics` (also `tactics`) shows both and each member's
  strategy (as `strategy`); `company tactics focus <rule>`, `company
  tactics healing <percent>`, `company tactics default` (both back to
  their defaults).
- **Focus `none`:** every member follows its own strategy rule (32d).
- **Focus set:** every living company member in the battle, the player
  included, aims by the focus instead of its own rule. It overrides *who*
  is aimed at only; roles stay (a healer still heals, a caster still
  casts its attack spell, at the focus's choice). Reach still binds: a
  focus choice out of reach falls to the nearest foe reachable, as in 32d.
- `casters` is a new rule (also usable in `strategy`): foes chanting a
  spell first, then foes that know spells, then the nearest. `leader`,
  `weakest`, and the rest are 32d's rules, unchanged. `assist` and
  `defend` are not offered as focus (they follow a person).
- **Healing threshold:** replaces the fixed half in `strategy.Decide`
  (`Situation.HealBelow`, a percent of the wound limit). A downed player
  still counts as the most hurt.
- In a battle only the focus may change; the healing threshold and
  `default` are refused ("You can't change that in the middle of a
  battle"), as the rest of setup is.

## B. Focus mid-battle (30c1)

- **Scope:** held on the player's battle in `internal/battle` (runtime
  only, like battles and aims; after a restart battles start fresh from
  the saved focus). It never writes the saved setting.
- **Who:** the player whose battle it is (each player leads their own
  company and battle). Others in the room see the line.
- **Cooldown:** one change per combat round. A change is *pending* until
  the next round's upkeep applies it; while one is pending, another is
  refused: "Your company is still turning; try again next round."
- **Command:** `company tactics focus <rule>` in a battle sets this
  battle's focus; `company tactics focus default` returns to the saved
  one. Every other command stays refused.
- **Aims turn** at that upkeep (decision 5): every member able to fight
  turns to the new focus's choice among the foes it can reach, with 29c's
  "turns toward" line, unless it is already on it. A member casting keeps
  its spell; a player who used `break` stays out. After that round, aims
  stick as usual.
- **Narration:** one line when the order is given, 29c voice: "You call
  the company onto the goblin hexer." It names the focus's choice among
  every foe standing (reach aside) when there is one, else the rule ("You
  call the company onto their weakest."). The room sees "Aria calls her
  company onto …" (with the player's pronoun). `none`: "You let each of
  your company choose their own foe."
- **Events:** a `focus-change` event (source the player, `Rule` the new
  focus, `none` for none) in the 29b stream when the upkeep applies it.

## C. Web Combat tab (30c1)

- `Company.Battle` gains `focus` (the effective rule, `none` for none),
  `saved_focus`, and `focus_ready` (no change pending). The battle view
  shows seven focus buttons, the current one marked (`aria-pressed`),
  all disabled while a change is pending.
- A click sends `company tactics focus <rule>` through `Client.SendInput`;
  no new server message.
- The Setup view shows the saved focus and healing threshold with a menu
  that sends the same commands (no new message: `Company` gains
  `tactics: {focus, healing}`).
- Keyboard and screen-reader friendly, like the rest of the dock; the
  confirmation line reaches the existing live region.

## D. Enemy personalities (30c1)

Data, not code, per mob template or race (template wins over race):

```yaml
targeting: wounded        # a strategy rule name
targetingnoise: 20        # percent of (re)aims that pick a random reachable foe
```

- The shipped table is decision 7. No `targeting` anywhere means today's
  `weakest`, no noise, by the same `engagement.AssignTarget` as today.
- When an enemy re-aims (decision 6), it picks by its rule among the
  company members it can reach: `weakest`, `strongest`, `wounded`,
  `nearest`, `furthest`, `leader` (the player), `casters` (decision 8).
  An unknown rule, `assist`, or `defend` reads as `weakest`.
- Noise is one roll per re-aim (not per round), from a seeded source a
  test replaces; on a hit the enemy takes a random member it can reach.
- "Hunts the isolated" (the draft's wolf) needs adjacency data; `wounded`
  for now.
- No focus for enemies; a group's aim is the aim of each member.

## E. Guardian and guards (30c2)

Refined, with owner decisions 9–12, in the
[30c2 design](2026-09-30-phase-30c2-guardian-design.md); that doc wins
where they differ.

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

- **`internal/strategy`:** `Casters` rule and `Foe.Chanting`/`Foe.Caster`;
  `Tactics` (focus, healing), its parsing, and the `TacticsProvider` seam;
  `HealBelow` in `Situation`; `EnemyPick` (rule plus a seeded noise roll)
  for enemies; `Guardian` role (30c2).
- **`modules/strategy`:** stores `Tactics` in its file beside the
  strategies (save on change, rollback, purge).
- **`modules/company`:** the `company tactics` command (`tactics.go`) and
  a `tactics` shorthand.
- **`internal/battle`:** the battle-scoped focus and its pending flag.
- **`internal/enemyparty`, `internal/hooks`:** `Attacker` takes the
  effective rule (focus over strategy); personalities for mob aim;
  guard interception (30c2).
- **`internal/mobs`, `internal/races`:** `targeting`, `targetingnoise`
  fields; shipped race data per decision 7.
- **`internal/combatstream`:** the `FocusChange` kind and `Event.Rule`.
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
- **Wiring** (real commands, `DoCombat` rounds, in `modules/company`'s
  brawl world unless named):
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
- **Web:** the buttons render, mark the current focus, disable while a
  change is pending, and send the command; the Setup view shows the
  saved tactics (checked in Chromium, `scripts/browser`, as 32g2);
  `Company.Battle`'s new fields in `modules/gmcp`'s tests.
- **Player help:** a new `help tactics` (aliases `focus`, `company-tactics`,
  `personalities`); updates to `help strategy` (the company focus overrides
  target rules), `help combat` (hub link; the focus is the one mid-battle
  change), `help targeting` (enemy personalities), `help webclient` (the buttons),
  `help company` (the subcommand), and `help
  guardian` for 30c2; `keywords.yaml`; the Combat lesson points to
  `company tactics`; `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate` pass. The
  independent review is recorded.

## Deferred

- Rotate the wounded; a retreat order; interrupt and mercy toggles (30d,
  30e); hunting the isolated (needs adjacency).
- Per-spell gambit lists and openers (as 32d).
