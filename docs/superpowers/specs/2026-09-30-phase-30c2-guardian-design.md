# Phase 30c2: Guardian Role and Guards — Design

Slice two of [Phase 30c](2026-09-29-phase-30c-company-tactics-design.md)
(its section E), after 30c1 (focus, healing threshold, enemy
personalities). Part of the
[combat roadmap](2026-09-26-combat-presentation-roadmap.md) (handoff §36
item 6, guard reactions, formerly deferred 11d).

## Owner decisions (2026-09-30)

Put to the owner with AskUserQuestion in this session. There is no
standing "proceed with your recommendation" instruction.

9. **Guards refill.** A guardian has 2 guards per battle; a spent guard
   comes back one per 2 combat rounds, up to 2 (recommended, chosen).
10. **`guard` means guardian.** `strategy <who> guard` makes a guardian of
    whoever is most hurt, and `strategy <who> guard <other>` a guardian of
    `<other>`. The word stops being an alias for the `defend` target rule
    (`defend` and `protect` still set it; a saved `defend` is unchanged).
    The owner asked whether strategies are set before or during a battle:
    before. Like every strategy, a guardian and its ward are refused in a
    battle; only the focus changes mid-battle (30c1).
11. **Reach: the same or the next column** (the owner's preference over the
    recommended "anywhere"), as 11c's lateral range. A guardian can step in
    only for a ward within one column of its own. Because this can leave a
    guardian quietly unable to guard, `strategy` and `formation` warn when a
    set ward is out of reach, and the web Setup marks it (chosen). With no
    ward set, the guardian guards the most hurt member within one column.
12. **Knocked down or stunned: can't guard; guards kept** (recommended,
    chosen). While either status lasts the guardian doesn't step in, but it
    keeps its unspent guards and refills as usual. (The question said the
    statuses last 1–2 rounds; the shipped data then had knocked down 4
    combat rounds and stunned 3. Flagged to the owner, who set them to 2
    and 1 afterwards: knocked down loses its next action and stays down one
    more round, stunned loses its next action.)

## Prior-art check (against `master` at `bc4e311`, 2026-09-30)

- **Roles and strategies (32d, 30c1):** `internal/strategy` has
  `Role` (`fighter`, `healer`, `caster`), `Rule`, and `Strategy{Role,
  Rule}` with blank-means-default storage; `modules/strategy` stores it per
  member key (`leader`, `companion:<id>`) with rollback, prune, and purge,
  and the `strategy` command parses `strategy <who> <role|rule>`. `guard`
  is today an alias of `Defend` in `ruleAliases`. `Decide` returns `Swing`
  for any role it doesn't know, and `strategyPass` skips `Fighter` before
  calling `Decide`.
- **Interception (11c):** every enemy blow on the company goes through one
  of two gates in `internal/hooks/combat_formation.go`, which run
  `resolveAttackTarget` (front-row interception, then legality):
  - `gateMobVsPlayerAttack` (blow at the player): a redirect onto a
    companion is resolved in place by `resolveInterceptedMobAttack`
    (`AttackMobVsMob`), reporting `handled`;
  - `gateEnemyAttacksCompanion` (blow at a companion): a redirect onto the
    player is resolved by `resolveInterceptedAttackOnLeader`
    (`AttackMobVsPlayer`); onto another companion it just returns that mob.
  Neither touches the attacker's `Aggro`: interception is recomputed every
  blow. Interceptions are not narrated today (the blow line simply names the
  interceptor). **A guard reuses exactly this:** after `resolveAttackTarget`
  settles who is struck, a ready guardian of that member becomes the target
  instead, through the same three resolutions.
- **Lateral range (11c):** `formationcombat.InLateralRange(a, b)` (within
  one column). An unplaced member fails open everywhere in 11c; a guard
  does the same (an unplaced guardian or ward is "in reach").
- **Statuses (30a):** `internal/status` has `KnockedDown` and `Stunned`
  buff ids; a live (unexpired) buff is found through the character's
  `Buffs`. `status.Has`/`LostAction` show the pattern.
- **Battles (29b2, 30c1):** `internal/battle` holds each player's battle
  (runtime, leaf mutex, copies out) and 30c1's focus state on it. Guard
  counts belong there too: per battle, per member key, gone with the
  battle, never persisted.
- **Combat rounds:** `DoCombat` runs once per combat round
  (`CombatOnCadence`); 30c1's `beginRefocus` shows a once-per-round pass
  after `battlePass`. The refill ticks there.
- **Events (29b):** `combatstream.GuardUsed` and `GuardExhausted` exist; the
  summary already counts `GuardUsed` by source ("Guards  Tamsin Reed 1").
- **Web (32g, 32g2, 30c1):** `Company` carries each member's
  `strategy {role, target}`; `Company.Battle` carries aims and focus;
  `window-combat.js` has `ROLES`, the Setup member menu, and the battle view
  fighter buttons with a sub-line.

## A. The guardian role

- `strategy.Guardian` (`guardian`), listed after `caster`. Never a
  default (`DefaultRole` is unchanged). It fights as a fighter: `Decide`
  swings, and the strategy pass skips it as it skips fighters. Its target
  rule still applies to its own aim.
- **Ward:** `Strategy.Ward`, a member key (`leader` or `companion:<id>`),
  blank for "the most hurt". Stored only for a guardian; any other role
  clears it. A ward that names no current member (dismissed, lost) reads
  as blank; the command's prune drops it. A ward still in the company but
  away or fallen is guarded by no one else (review finding 1).
- **Commands** (all refused in a battle, as every strategy change is):
  - `strategy <who> guardian` / `strategy <who> guard`: guardian, no ward.
  - `strategy <who> guard <other>` (also `guardian <other>`): guardian of
    `<other>` (`me` for the player, or a companion's name or `#id`).
    Refused for itself ("A guardian guards someone else: ...", which
    needs no pronoun).
  - Setting another role (`fighter`, `healer`, `caster`) or `default`
    clears the ward.
  - The confirmation says who is guarded, and warns when the ward stands
    more than one column away (decision 11): "Tamsin will guard Aria,
    stepping in to take a blow meant for her. Tamsin can't reach Aria to
    guard her from where they stand: a guardian must stand in the ward's
    column or the next (see `formation`)."
- **Listing:** `strategy` shows the role column as `guardian` and a
  "guards Aria" (or "guards the most hurt") note, with the same reach
  warning; `strategy <who>` describes it.
- **`formation`:** after any `formation` change (and in the grid view) a
  guardian whose set ward is out of reach gets the warning line.
- Only the player and companions have strategies, so only they can be
  guardians.

## B. A guard

- **When:** an enemy's weapon blow is about to land on a company member
  (after 11c's front-row interception and legality decide who), in a
  battle of that member's player. Spells are not guarded.
- **Who steps in:** the first guardian, in formation order (row by row,
  left to right; unplaced after placed), for whom all hold:
  - it is not the member struck, and the member struck is its ward: its set
    ward, or, with none set, the most hurt member of the company by health
    fraction (health over max) among those in reach, ties by formation
    order, and only someone actually hurt (below full): nobody hurt, nobody
    guarded;
  - the ward is within one column of it (decision 11; unplaced fails open);
  - it is able: alive and in the room (a player above 0 health), no
    `no-combat` flag, not knocked down or stunned (decision 12);
  - it has a guard left.
- **What happens:** the blow is resolved against the guardian instead, by
  the same resolution 11c uses (`AttackMobVsMob` for a companion,
  `AttackMobVsPlayer` for the player, with buffs, messages, onHurt, the
  shield-break check, and the vitals event). The attacker's `Aggro` is
  untouched: the next blow is aimed at the ward again. A guard never
  chains (a guarded blow is not guarded again), and 11c's front-row
  interception is not reapplied to the guardian.
- **Cost:** one guard, spent when the guardian steps in, whatever the blow
  then does (hit, miss, or dodge). Stepping in costs the guardian nothing
  else: it still acts in its own turn.
- **Guards:** 2 at the battle's start (each guardian's count is created
  the first time it is read). At each combat round, a guardian below 2
  gains a charge; at 2 charges it gets a guard back (decision 9): while
  any is spent, one comes back every 2 rounds (a charge carries over a
  second spend). The
  count is runtime only, on the battle; a new battle starts at 2.
- **Narration** (29c voice, mechanics in lowercase parentheses), before
  the blow's own line:
  - room: "Tamsin Reed steps in front of Aria. (guard, 1 left)"
  - the player as guardian: "You step in front of Tamsin Reed. (guard,
    1 left)"; as ward: "Tamsin Reed steps in front of you. (guard, 1 left)"
  - at 0: "(guard, none left)".
  - A guard coming back is silent (the battle view shows the count).
- **Events:** `guard-used` (Source the guardian, Target the ward) each
  time; `guard-exhausted` (Source the guardian) when it
  spends its last. Both in the battle's fight.

## C. Web

- `Company`'s member `strategy` gains `ward` (a member key, omitted when
  none) for a guardian, and `ward_reach: false` when a set ward is out of
  reach.
- `Company.Battle` gains `guards: [{key, left, ward}]`, one per guardian on
  the player's side (`ward` its set ward; blank for the most hurt).
- Setup: `guardian` joins the role menu, with "Guard: <member>" items for
  a guardian (and "Guard: the most hurt"), and the member's row shows
  "guards Aria", marked "(out of reach)" when so.
- Battle view: a guardian's fighter button sub-line adds "guards Aria ·
  2 guards" (or "no guards"). No new server message; everything is sent as
  commands through `Client.SendInput`.

## Module

- **`internal/strategy`:** `Guardian`; `Strategy.Ward`; `ParseRole`
  accepts `guard`/`guardian`/`protector`; `guard` leaves `ruleAliases`;
  `GuardWard` (pure: who a guardian guards among plain values).
- **`internal/formationcombat`:** `GuardReach(f, guardian, ward)` (within
  one column; unplaced fails open).
- **`modules/strategy`:** the command's guardian and ward forms, the
  warnings, clearing and pruning the ward, the listing.
- **`internal/battle`:** per-battle guard counts: `GuardsLeft`,
  `SpendGuard`, `TickGuards`.
- **`internal/hooks`:** `combat_guard.go`: the guard check in both gates,
  the resolution, narration, events, and the round's refill.
- **`modules/company`:** the `formation` warning.
- **`modules/gmcp`, `window-combat.js`:** section C.
- **Help and tutorial:** below.

## Invariants

- **The clock:** the refill counts combat rounds `DoCombat` already runs;
  nothing advances time.
- **Restart and copyover:** the guardian role and ward are strategies,
  durable in the strategy module's file; guard counts are runtime only, on
  the battle, and a battle after a restart starts fresh at 2.
- **Game loop and locks:** guard checks run on the game loop inside the
  combat round; `internal/battle`'s mutex is taken only inside its own
  functions (copy in, copy out), never across the engine; `modules/strategy`'s
  `m.mu` is not held across any engine call.
- **11c unchanged:** with no guardian anywhere, every blow resolves exactly
  as before.

## Acceptance criteria

- **Unit:** `ParseRole` for the guardian words; `guard` no longer a rule
  alias; a ward stored only for a guardian and loaded back (bad ward keys
  dropped); `GuardWard` (set ward, most hurt by fraction, ties, nobody
  hurt, out of reach); `GuardReach`; battle guard counts (start at 2,
  spend, refill one per 2 rounds, cap 2, gone with the battle).
- **Wiring** (real commands and `DoCombat`, `modules/company` brawl, with
  `hooks.UseAimRollForTest`):
  - `TestGuardianGuardsWard`: a companion guardian of the player takes a
    blow aimed at the player (the player's health untouched by it), with
    the line, `guard-used`, and one guard spent;
  - `TestGuardianPlayerGuardsCompanion`: the player as guardian of a
    companion (the leader-as-interceptor resolution);
  - `TestGuardianCompanionGuardsCompanion`;
  - `TestGuardsExhaustAndRefill`: after 2 guards, `guard-exhausted`, the
    next blow lands on the ward, and a guard is back 2 rounds later;
  - `TestGuardianOutOfReach`: a ward two columns away is struck; the
    `strategy` and `formation` warnings;
  - `TestGuardianKnockedDown`: no guard while knocked down, guards kept;
  - `TestGuardianMostHurt`: no ward: guards the most hurt in reach, and
    nobody while nobody is hurt;
  - `TestGuardianRefusedInBattle` and the command forms (set, ward,
    self refused, role change clears the ward, prune);
  - an unguarded ward is struck as before; 11c's, 32d's, and 30c1's fight
    tests pass.
- **Web:** `Company`'s `ward`/`ward_reach` and `Company.Battle`'s `guards`
  in `modules/gmcp`'s tests; Setup's guardian role and ward items, the
  row's note, and the battle view's guard count in
  `scripts/browser/dock-windows-check.mjs` (Chromium).
- **Player help:** a new `help guardian` (aliases `guardians`, `guard`,
  `guards`, `ward`); updates to `help strategy` (the role, the ward, the
  `guard` word), `help combat` (hub link), `help tactics` (guards and the
  focus), `help formation` (a guardian's reach), `help webclient` (Setup and
  the battle view), and `help battle-summary` (the Guards line);
  `keywords.yaml`; the Practice Yard lesson points to `help guardian`;
  `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, `make validate`, `make js-lint`,
  and the Playwright check pass. The independent review is recorded.

## Deferred

- Guarding spells; a guardian's own defence bonus; enemy guardians (no
  shipped enemy needs them); rotating guardians.
