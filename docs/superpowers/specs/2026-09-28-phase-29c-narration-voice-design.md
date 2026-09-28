# Phase 29c: Narration Voice (Weapons and Spells) — Design

Builds the [narration feature spec](2026-09-26-combat-narration-design.md)
(owner-approved direction, 2026-09-26). This is the design pass that spec
asked for: prior art checked against today's code, the open decisions
settled, and the acceptance criteria made concrete.

## Decisions (owner, 2026-09-28)

Asked in chat; the owner took the recommendation on all four.

1. **Charmed tag:** dropped for any company companion, whoever is looking.
   Other charmed mobs and pets keep it.
2. **Proper names:** a name whose first letter is a capital is proper
   ("Garrick Vane", "Old Wenna") and gets no article. Anything else gets
   "the" ("the bandit captain"). No data changes.
3. **Critical pool:** only a real critical hit draws the `critical` pool.
   A non-critical roll over 100% (a sharpened top roll) is capped at
   `heavy`.
4. **Blank line between rounds:** deferred to 29f (pacing), which already
   buffers each round per viewer. 29c changes words, not layout.

## Prior-art check (2026-09-28, against `4cb2faf`)

The feature spec's inventory still holds. Line numbers today:

- **Weapon text:** 8 files, 1,225 lines, in
  `_datafiles/world/default/combat-messages/`. `{damage}` appears inside
  sentences in every file (25 times in `bludgeoning.yaml` alone).
- **Pool choice:** `items.GetAttackMessage(subType, pct)`
  (`internal/items/attack_messages.go`), called once in
  `combat.go:391`. `GetPreAttackMessage` serves `prepare`/`wait`
  (`combat.go:125`, `GetWaitMessages` in `NewRound_DoCombat.go`).
- **Crit wrapping:** `combat.go:400–407` wraps all four lines in `***`.
- **Other `!` in the round:** the dodge lines (`combat.go:350–351`), and
  the defender's `[you blocked N]` suffix (`combat.go:423`).
- **Token substitution:** `buildCombatMessages` (`combat.go:174`); a mob's
  name is `GetMobName(0).String()`, already coloured and adjective-tagged.
- **"prepares to fight":** `usercommands/attack.go:216, 273, 277` and
  `mobcommands/attack.go:126, 129, 150`.
- **Death line:** `X has died.` in `usercommands/suicide.go:66` and
  `mobcommands/suicide.go:90` (the path every combat death takes).
- **Fallen notice:** `modules/company/death.go:141`, via
  `formatAllowance`.
- **Charmed tag:** `characters.OnGetFormattedName`
  (`internal/characters/formattedname.go`), fired for every formatted
  name; no module hooks it today.
- **Fight lifecycle:** `internal/hooks/combat_battle.go:152` opens a
  fight (`combatstream.Open`); `combat_stream.go:323` ends it
  (`EndFight`, outcome `victory`/`defeat`/`broken-off`); every
  retarget goes through `emitTargetChange` (`combat_stream.go:123`).
  The stream has no subscriber besides the summary path today.
- **Spells:** `mm.js`, `sparks.js`, `heal.js`, `healall.js` write their
  own text with `SendUserMessage`/`SendRoomMessage` and
  `GetCharacterName(true)`.
- **Pets:** pet attacks use their own messages (`combat.go:447+`). Out of
  scope; noted below.

## Scope

### 1. Weapon text (`combat-messages/*.yaml`)

- Every line rewritten in the dark, physical voice of the reference mock:
  no `!`, no ALL-CAPS word, no `{damage}` token. Pool structure and the
  other tokens stay; pools may be trimmed to three or four good lines a
  variant rather than padded.
- `heavy` is voiced as a strong, clean blow, never as a critical one.
  `critical` is voiced as a decisive, bloody one.
- `prepare` and `wait` lines are kept and re-voiced (aiming, circling,
  drawing breath).

### 2. Suffixes and pool choice (`internal/combat`)

- `GetAttackMessage` gets the crit flag: crit → `critical`; otherwise the
  percent tiers, with anything over 100% capped at `heavy`.
- The `***` wrap is removed. Every hit line in all four variants
  (attacker, defender, attacker's room, defender's room) ends in
  ` (N damage)`, or ` (critical hit, N damage)` on a crit. Misses get
  none.
- The defender's `[you blocked N]` joins the parentheses:
  `(5 damage, 2 blocked)`.
- The dodge lines are re-voiced without `!`.

### 3. Articles

- One helper in `internal/util`: `Article(name)` puts "the " before a name
  whose first visible letter is lowercase (skipping ANSI tags), and
  `CapitalizeFirst(line)` capitalises a line's first visible letter.
- `buildCombatMessages` uses them for `{source}`/`{target}`; every
  weapon line is passed through `CapitalizeFirst`. The death line,
  opener, "turns toward", and the spell scripts use the same pair (the
  scripts through a new actor method, `GetCombatName()`, which returns
  the coloured name with its article).

### 4. Charmed tag

`OnGetFormattedName` carries no mob identity, so it can't tell a
companion from a pet. Instead, `modules/company/runtime.go` clears the
`charmed` adjective right after it charms a companion
(`Character.Charm` sets it). Nothing reads that adjective for logic; the
charm itself (`Character.Charmed`) is untouched. Other charmed mobs keep
the tag.

### 5. Engagement and fight-end lines (`internal/hooks`)

Found in planning: every retarget site already prints its own "turns on"
line beside its `emitTargetChange`, so a separate stream sink would print
twice. The lines are rendered where the events are produced instead:

- **Opener** — `beginBattle` (right after `combatstream.Open`) sends one
  line to the room: a generic pool, or a pool keyed by the first enemy's
  first mob `groups` name when one exists (`bandits`, the practice
  squad's group). The pools are Go maps in
  `internal/hooks/combat_narration.go`; a `fights.yaml` beside the weapon
  files would be loaded as a weapon file, and no one edits these outside
  code yet.
- **Turns toward** — every existing "turns on" line (the battle, engagement,
  and formation retargets, and `groupTurnsOn`) is reworded to "turns
  toward" through one helper, with articles.
- **Closing** — `fightSides.end` sends one closing line to the fight's room
  on `victory` only, before the summary.
- Every "prepares to fight" line is removed.

### 6. Death notices

- Mob death text comes from `suicide`, a queued command, so it would print
  after the closing line. `handleAffected` (`NewRound_DoCombat.go`) now
  prints the combat death line itself, at once, and queues
  `suicide quiet`, which skips the room line (also the practice "is
  beaten" line, printed at once the same way). A mob with
  `revive-on-death` keeps today's path. Other deaths (`suicide` from
  buffs or scripts) print the same re-voiced line from `suicide`.
- The death line is a small pool of physical, final lines
  (`The bandit captain crumples and does not rise.`), exported from
  `mobcommands` as `DeathLine(name string) string`; the player's own death
  line in `usercommands/suicide.go` is re-voiced the same way.
- The company fallen notice is indented four spaces and reads its time
  in words (`3 hours`, `2 hours 30 minutes`, `45 seconds`) through a new
  `allowanceWords`; the other notices keep `formatAllowance`.

### 7. Spells (`mm`, `sparks`, `heal`, `healall`)

- Chant lines each round, voiced per school, ending
  `(chanting: <spell>, N round(s))`.
- Release lines are physical; damage ends ` (N damage)`, heals
  ` (N healed)`.
- `sparks` prints one cast line, then an indented line per target.
- `healall` prints one cast line, then one indented line listing everyone
  healed: `Garrick Vane (3 healed) · Ysolde (2 healed) · you (4 healed)`.
- Other spells (`floatinglight`, `illum`, `poly`, `curepoison`,
  `aidskill`, `tameskill`) lose `!` and ALL-CAPS only.

## Constraints and deferrals

- Never advances the clock; no persisted state; nothing here survives or
  needs to survive a copyover (text only).
- Sinks run on the game loop and must not block (the stream's contract).
- Screen readers: suffixes are words, not symbols.
- **Deferred:** pronouns and ordinals (29d), pain lines (29e), round
  breaks and pacing (29f), pet attack text (upstream pet messages, not in
  the feature spec), crit status names in the suffix (30a).

## Player help

- `help combat` and the pages 29b added (`attack`, `death`, …) are
  updated where they quote combat text or describe `***`.
- A short `help narration` page: what `(N damage)`, `(critical hit, …)`,
  `(N healed)`, and `(chanting: …)` mean; aliases `damage`, `critical`;
  linked from `help combat`.
- The Practice Yard lesson's hint points to it.

## Acceptance criteria

- No line in the eight weapon files or the narration pools contains `!`
  or an ALL-CAPS word, and no weapon line contains `{damage}` (a data
  test).
- A wiring test through `hooks.DoCombat` with the shipped config:
  - every hit line ends in `(N damage)`, N matching the damage dealt;
  - a forced critical hit ends in `(critical hit, N damage)` and its text
    comes from the `critical` pool; a sharpened non-crit top roll does
    not;
  - no `***` and no "prepares to fight" anywhere in the transcript;
  - no company companion's name carries the charmed tag;
  - exactly one opener per fight, and one closing line when the group is
    beaten (none on defeat or broken-off);
  - a retarget mid-fight prints one "turns toward" line;
  - the closing line comes after the last enemy's death line.
- Unit tests: `Article`/`CapitalizeFirst` (tags, empty, proper names);
  pool choice (crit, capped heavy); the fallen-notice format.
- Spell tests: `mm` and `healall` through a real cast print the suffixes.
- The 27c practice-fight tests updated deliberately to the new text.
- `help narration` renders; `TestTutorialHelpPointersExist` passes.
- Before and after transcripts of the same seeded fight are attached to
  the review.
