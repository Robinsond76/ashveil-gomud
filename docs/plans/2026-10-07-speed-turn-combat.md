# Phase 82: speed turns and a docked battle screen (design and plan, 2026-10-07)

Robinson, 2026-10-07 13:47 (project thread "speed turn combat"):

> I'd like battle messages to be a smaller font so it all tucks in and
> doesn't move the screen too fast. I want the battle screen to open up top
> in the middle section of the webpage, with the bottom of the screen being
> the regular mud screen. In the terminal screen I want the battle messages
> showing. I still think combat is too fast [...] I want each character and
> mob to move based on their speed in battle, one at a time, just like ogre
> battle. Because each one hits one at a time, it'll be easier to follow the
> fight. We'll also be able to determine who moves faster and who can attack
> more often based on their speed.

Decided under full autonomy (Robinson, 2026-10-06). Each decision below
carries its reason. Four build phases, 82a to 82d, follow at the end.

## Why combat reads too fast today

- A combat round is 8 seconds (`CombatEveryRounds: 2` × `RoundSeconds: 4`).
  `DoCombat` resolves the whole round at once: the leader acts, then every
  mob (companions and foes alike, in instance-id order), then a second pass
  for earned extra turns.
- `internal/combatpace` then plays the round's lines back, but every round
  is squeezed into a window (6 s at `normal`, at most 90% of the round). A
  five-member company against three or four foes makes 12 to 20 lines a
  round, so each line gets about 0.3 to 0.5 s, follow-ups less. That is the
  blur Robinson sees.
- Order is by kind, not speed: the leader always goes first and companions
  (spawned earlier) usually go before foes. Speed already decides *how
  often* someone acts (`combat.Tempo`, the action meter, at most 2 turns a
  round, `help tempo`), but never *when*.

So the fix has two halves: order the round by speed, and let the round last
as long as its actions need, one action at a time.

## The model: a round is a turn order played one action at a time

### Speed and tempo (decision 1: keep `combat.Tempo` as the one formula)

Speed in battle is the existing **tempo**: Speed stat against
`TempoSpeedRef`/`TempoSpeedSpan`, the `attacks` bonus, personal burden,
armor bulk, a beast's share, a golem's slowness and the weapon stance, clamped
to 0.6-1.5. It already covers class (archetype Speed and armor training),
mobs (their Speed stat and gear), gear (bulk, `attacks` mods) and stance.

*Why:* every class, mob and relic was tuned against this formula; a second
"initiative" stat would need its own balance pass and would let order and
frequency disagree. Ogre Battle uses one agility value for both.

The number players see is the tempo with one decimal ("Tempo 1.2"), named
Tempo, not Speed. *Why:* `Speed` is already a raw stat shown on the
Character panel; a second "speed" would read as the same number. `help
tempo` already explains the meter.

### How many turns (decision 2: the action meter is unchanged)

`combat.Meter.Fill` still decides each unit's turns per round: one opening
turn, then 100 × tempo points a round, 100 per turn, at most
`MaxTurnsPerRound` (2), fractions carried. So a tempo 1.5 fighter acts three
times in two rounds and a tempo 0.6 one three times in five. *Why:* this is
"who can attack more often", it is balanced, and it keeps chants, wind-ups
and the "an extra turn is only a physical attack" rule (`extraTempoTurn`)
exactly as they are.

### When each turn comes (decision 3: slot = turn number ÷ tempo)

Each round builds a turn order before anyone acts. A unit with *n* turns this
round gets slots at `k / tempo` for `k = 1..n`. All slots in the battle are
sorted ascending and played one at a time.

- A fast fighter (1.5) acts at 0.67, everyone at tempo 1 at 1.0, its second
  turn (when earned) at 1.33, a slow fighter (0.6) at 1.67. So the fast go
  first, a second turn comes after most first turns, the slow go last.
- **Opening bonuses move the first slot earlier** in round 1: `(1 −
  bonus/100) / tempo`, where bonus is the meter bonus the unit already gets
  (Samurai Iaijutsu `OpenMeter`, Scouted ground, a Packlord hound's `Open`,
  the Shogun's ally bonus). These mechanics promised "strikes first"; now
  they really do.
- **Ties** (common: many units sit at tempo 1.0) break by raw Speed stat,
  then Perception, then a per-round shuffle from a seeded source with a test
  seam (`hooks.UseTurnOrderRollForTest`). No side wins ties by rule.

*Why not meter-crossing time (pure CTB):* every unit with no bonus crosses
100 at the very end of round 1, so the opening round would be one big tie,
and a tempo 1.0 unit always ties at the end of every round. Slot = k ÷ tempo
is simple to explain ("the faster you are, the earlier you act") and keeps
count and position from one number.

*Why not a free-running timeline with no rounds:* rounds carry the whole
rule set: chants and wind-ups count rounds, statuses and combat-round buffs
tick per round, focus is once a round, morale and nerve check per round,
weather and sigils last N rounds. Removing rounds would mean re-deriving
every one of those. A round that is a speed-ordered turn list gives the Ogre
Battle feel and keeps them all.

### What happens in a round

1. **Upkeep** (unchanged content, now always first): retreat requests,
   focus orders (`TakeRefocus`), battle orders, stance application, meter
   fill, status and combat-round buff ticks, morale and nerve checks,
   weather and sigil countdowns. Its lines play as one beat before the first
   slot. Retreat is hoisted here from the leader's own loop iteration
   (`handleRetreat` in `handlePlayerCombat`), so a fast or slow leader never
   changes when a retreat lands. *Why:* today the leader's loop runs first,
   so this keeps current behavior exactly.
2. **Slots**, in order. At its slot a unit does what its turn does today:
   attack, shoot, start or release a chant, wind up or release, use a class
   ability (strategy, orders), or lose the turn to a status. A unit felled,
   routed, yielded or withdrawn before its slot loses it. A target felled
   earlier in the round is re-chosen at the slot by the unit's strategy, as
   today's retarget does.
3. **Round end**: deaths, rewards and the battle's end are settled as now
   (`handleAffected`).

Mechanics this touches, and how they fit:

| Mechanic | In the new round |
| --- | --- |
| Battle orders (61), strategy, company tactics | Read at the unit's slot, as today. |
| Weapon stances (69) | Already scale tempo; now also move the slot (a quick-draw bow acts earlier, a shield wall later). |
| Chants | Begin at the caster's slot; advance once a round; release at the caster's first slot of the round they complete. A foe whose slot comes first can break the chant before it releases, so a fast interrupter now really beats a slow caster. Chant speed stays the biggest caster lever. |
| Physical wind-ups | Prepare at the slot one round, release at the slot the next; take the whole round as today. |
| Statuses that cost actions (stun, knockdown, hesitation) | The lost turn shows at the unit's slot as a short beat ("Mira is stunned and loses the turn"). |
| Damage over time, regeneration | Tick at upkeep, as today. |
| Guards, brace, covering shot, step-in (bonds) | Reactions inside another unit's slot, as today; they take no slot. |
| Company focus | Still once per round: set any time, applied at the next upkeep. The battle screen says "next round". |
| Retreat | Set any time, applied at the next upkeep. The battle screen says "retreating at the end of this round". |
| Dolls, summons, beasts | Act on the turn they act today (a doll's strike is its Master's slot; a beast has its own tempo). |
| Allied companies on the same foes | One shared turn order (see the clock below). |

The resolution order change is the only balance-relevant part of this
section (decision 7).

## The battle clock: one action per beat

### Decision 4: resolve the round at once, play it back one action per beat, and start the next round when playback ends

Today every battle in the world resolves on the same 8 s cadence. After this
phase, each **battle cluster** has its own clock:

- A cluster is every player battle that shares a combatant (allied companies
  on the same foes, a shared enemy group), joined transitively, as the
  existing tempo membership (`tempoFights`) already tracks.
- A cluster resolves its round in one call (upkeep and every slot in order,
  as above), on the game loop under `util.LockMud`, exactly as `DoCombat`
  does now. Its next round is due when the playback of this one ends:
  `due = now + upkeep beat + Σ action beats + round tail`, never less than
  `MinRoundMs`.
- A `NewTurn` listener (every 50 ms) resolves each cluster whose `due` has
  passed. Fights that are not player battles (mob against mob, PvP outside a
  battle) keep the old fixed cadence on `NewRound`.

*Why resolve-then-play instead of resolving one action per beat:* the
16k lines of `internal/hooks` combat code assume a round resolves atomically
under the game lock (guards, interrupts, clusters, coordination, morale,
death handling). Because the round resolves strictly in slot order, playing
it back one action at a time *is* sequential turns: nothing a later action
does can change an earlier one. The only thing a mid-round resolution would
add is acting on input mid-round, and the only inputs allowed are retreat
and focus, which are once-a-round by design. So the visible result is
Ogre Battle's, without rebuilding the round.

*Why the round waits for its playback:* that is what makes the pace honest.
The window clamp disappears for battle lines: no round is squeezed, so the
fight is never faster than one action per beat.

### Decision 5: beats

| Pace (`set combatpace`) | Action beat | Follow-up line | Critical / felling extra | Lost turn | Round tail |
| --- | --- | --- | --- | --- | --- |
| fast | 0.6 s | 0.15 s | +0.3 s | 0.3 s | 0.4 s |
| normal (default) | 1.0 s | 0.25 s | +0.5 s | 0.5 s | 0.6 s |
| slow | 1.5 s | 0.3 s | +0.7 s | 0.75 s | 0.8 s |

- An action's beat is the action beat plus one follow-up time for each
  indented line it causes (at most three counted), plus the extra for a
  critical or a fall. All of an action's lines and its `Company.Battle.Event`
  data go out together at the start of its beat, follow-ups spaced by the
  follow-up time.
- `MinRoundMs` 3000, so a one-on-one fight does not flicker.
- A cluster's beat is the **slowest** chosen pace among its players. `off`
  (the screen-reader default) counts as normal for the clock; those players
  still get each round's lines at once, as today.
- The values live in the pacer's `specs`, as now, and `help combatpace`
  quotes them.

*Why these numbers:* the at-level fight tuning (#179) measured band fights
at about 4 rounds. Five members against three foes is about 9 actions a
round with tempo near 1, so a round is about 10-11 s at normal and an
at-level fight 35-45 s, against about 32 s today, with every action
readable. Robinson asked for quick at-level fights and for readability; one
second an action is the shortest beat at which a line and its animation can
be read before the next. `slow` is there for people who want more.

Target, measured in 82c: a band-middle at-level fight takes 25-45 s at
normal, a five-against-five fight at most 90 s.

### Game-round effects on combatants (decision 6)

A battle round now lasts 3-15 s instead of 8, but game rounds stay 4 s
(world time is never touched). Anything that runs per *game* round on a
combatant would then hit harder in long rounds. Rule: **inside a battle,
everything is counted in battle rounds.**

- Combat-round buffs (`combatrounds: true`) already tick in the combat loop;
  they follow the cluster's rounds with no change.
- 82c audits every buff and status a battle can apply (spells, abilities,
  flasks, ailments, sigils, weather) and makes any that ticks or expires by
  game rounds a combat-round buff. Out-of-battle buffs (inn, camp, food)
  stay on game rounds. `buffs.GetDurations` stops multiplying by
  `CombatEveryRounds` for battle display and reports battle rounds.
- Regeneration already stops in a battle (`AutoHeal` skips combatants).
- `CombatEveryRounds` remains for the fixed-cadence fights and as the
  number of game rounds a buff tick equals when converting.

### Copyover and restart

The clock and the turn order are ephemeral game-loop state, like the meter.
After a copyover each cluster's next round is due at once (held lines are
already flushed at copyover). No durable state changes, so nothing is
migrated.

## Balance (decision 7: re-measure, tune only the at-level knob)

- **What changes:** only who acts first within a round. Turn counts,
  damage, healing and every class number stay. Today the leader and the
  companions usually act before foes; afterwards a fast foe can strike
  first. That slightly favors foes, most in short at-level fights.
- **Re-measure in 82b**, after phase 81 (class tuning) has merged:
  `TestBalanceAtLevel` (fights before a rest by company size; Robinson's
  targets from the 10:41 and 10:43 messages still apply: five members 15-20,
  magic 7-12, four 8-14, three 5-7, solo struggles) and the class mirror
  sims (class-balance checks only). Same samples as #179; one full run at
  the end, timeboxed.
- **If at-level fights drift out of band,** tune the at-level random-group
  foe HP share (40% today) or their aim spread, never class numbers and
  never per-company scaling (difficulty comes only from zone level). Class
  mirror drift beyond noise is recorded, not tuned here; it goes to a
  follow-up class pass.
- Sims call `DoCombat` directly and count rounds, so the clock (82c) does
  not change their numbers; 82c reports seconds per fight beside rounds.

## Web client (decision 8: the battle screen docks above the terminal)

### Layout

- The middle column (`#terminal`, between `#dock-left` and `#dock-right`)
  becomes a column: a **battle pane** on top, the terminal below. When a
  battle opens, the battle screen moves from its fixed overlay into the pane;
  the terminal shrinks and `fitAddon.fit()` re-measures it. When the battle
  ends (after the 3 s outcome) or is minimised to the badge, the pane closes
  and the terminal takes the full column again.
- The canvas draws at the largest whole-number scale of 320×180 that fits
  the column's width and at most 50% of its height (1× minimum), so pixel
  art stays crisp. Header, legend, caption and buttons stay under it as now.
- **Phone** (40i layout): the same split inside the Game view, full width,
  1× scale (320×180 fits 360 px). Other phone views are unchanged.
- The floating overlay is removed rather than kept as an option. *Why:*
  Robinson asked for one layout; a second mode doubles the browser checks
  for no asked-for benefit. The "open by itself" setting and the badge stay.

### Font (decision 9: the terminal steps down a size while the battle pane is open)

- xterm.js draws every line at one size, so one line cannot be smaller than
  the next. While the battle pane is open, the terminal font steps down:
  20→15, 18→14, 16→13 px on desktop (by dock layout, as now) and 13→11 on a
  phone; it returns when the pane closes. During a battle the terminal is
  almost all battle lines, so this is what "smaller battle messages" looks
  like in practice, and it fits more lines in the shorter terminal.
- A setting, "Battle text: smaller / same size", defaults to smaller.
- Combined with one action per beat, the terminal scrolls at most a few
  lines a second at normal.

*Why not a separate HTML log under the canvas:* Robinson asked for the
battle messages in the terminal. A second log would duplicate the text and
split the screen-reader path.

### Turn order on the battle screen (82d)

- A strip under the canvas shows the round's turn order as small name tags
  (or class glyphs at phone width), with the unit now acting highlighted and
  those who have acted dimmed; upcoming slots of a fast unit appear twice.
  Hovering or tapping a figure adds its tempo to the tooltip.
- `Company.Battle` gains `order`: the round's slots as unit refs (the same
  refs the screen already uses, `?` for an unseen foe, merged into one
  presence), and each `Company.Battle.Event` batch carries its slot index.
- The Combat tab's Battle view lists the order in text, for the accessible
  path. The Company and Character panels show each member's tempo.
- The title keeps the round number; the terminal gets one dim line at each
  round's start ("Round 3"), so the text alone shows where rounds break.

## Constraints this keeps

- No player input changes a round in progress: only retreat and company
  focus, both applied at the next upkeep (Ogre Battle rule, 2026-09-28).
- Difficulty comes only from zone level; nothing scales to the company.
- Global game time never advances for battles; the clock is real time on
  the game loop and touches no world clock.
- Concurrency: clusters resolve on the game loop under `util.LockMud`; the
  pacer keeps its single mutex and lock order (`LockMud` → pacer).
- Text clients get the same story, ordered and paced the same way.

## Build phases

Order: 82a and 82b may run in parallel (web client vs Go). 82c needs 82b.
82d needs 82a and 82c.

### Phase 82a: docked battle pane and smaller battle text (web only)

- **Rebase point:** start after phase 79 (Fable polish, changing the web
  client) has merged, or merge it in before review; it touches the same
  files (`webclient-pure.html`, `webclient-core.js`, `window-battle.js`).
- Files: `webclient-pure.html` (center column markup and CSS),
  `webclient-core.js` (fit and font size with a battle step),
  `windows/window-battle.js` (open into the pane; badge minimise; scale by
  pane), `mobile.js` (Game view split), settings menu ("Battle text").
- Tests: `make js-lint`, `make js-test`; a browser check in
  `scripts/browser/` (Chromium and Firefox, Robinson's browser) that opens a
  battle at 1280×800 with both docks, with one dock, and at 360×740, and
  asserts the pane sits above the terminal, the terminal is still visible
  with at least 12 rows, the font steps down and back, and minimise/restore
  works. Screenshots to `/mnt/project-files/screens/82a-*.png`.
- Help: update `help battlescreen` (where it opens, minimise, text size)
  and `help webclient`; tutorial hint in the lesson that mentions the battle
  screen (`modules/tutorial/stages.go`, the Combat tab hint) corrected.
- Acceptance: on desktop and phone, a battle opens at the top of the middle
  column with the terminal below it showing the battle lines in the smaller
  size; closing restores the old layout and font.

#### 82a as built (2026-10-07, full autonomy): amendments and checks

- **Built against master, not after 79.** Robinson paused every other task
  for the overhaul (14:03), so 79 is parked; 79 will rebase on this layout.
- **Markup:** `#main-container` holds `#dock-left`, `#center` (a column:
  `#battle-pane` then `#terminal`) and `#dock-right`. The battle screen is
  appended into `#battle-pane` and its CSS is static (no fixed overlay);
  `body.battle-open` and `body[data-battle-text]` carry the state for CSS
  and the terminal's font step.
- **Font step:** `resizeTerminal` steps 20→15, 18→14, 16→13 and 13→11
  while the pane is open and "Smaller text" (default) is on; the choice is
  the `ashveil-battle-text` key (`smaller`/`same`) and a checkbox on the
  screen's head, not a settings menu entry (it belongs beside the thing it
  changes).
- **Scale:** whole-number 1-4× of 320×180 from the pane width and half the
  column height, so the terminal always keeps at least half the column on
  desktop (27 rows at 1280×800 with one dock, both browsers).
- **Phone:** the pane takes at most 55% of the Game view and scrolls, so
  the whole screen (legend, focus buttons, Help) is reachable; a second
  Retreat button sits in the head, in reach without scrolling (phone only).
  A battle opening brings the Game view up (the pane lives there), also
  when it starts while the last fight's outcome still shows; Help keeps the
  pane open above the text it lands in. These were failures of the real
  phone check, not design choices made up front.
- **Checks:** `scripts/browser/battle-pane-check.mjs` (Chromium and
  Firefox; Firefox installed in the container for it), `battle-check.mjs`,
  `dock-windows-check.mjs`, `mobile-check.mjs`, `make js-lint`,
  `make js-test`, help tests. Screenshots:
  `/mnt/project-files/screens/82a-{desktop-one-dock,desktop-no-dock,desktop-restored,phone}-{chromium,firefox}.png`.
- **Left for 82d:** the "Round N" terminal line and the turn-order strip.

### Phase 82b: speed-ordered turns (Go)

- **Rebase and measure point:** merge phase 81 (class tuning) before the
  balance re-measure; if 81 is still open, build first and measure after.
- Refactor `handlePlayerCombat` and `handleMobCombat` loop bodies into
  per-actor functions (`actPlayer(uid, extra)`, `actMob(id, extra)`;
  `continue` becomes `return`), with no behavior change, in its own commit
  so the review can diff it.
- Build the round's turn order (decision 3) after `beginTempoRound`;
  hoist retreat into upkeep; run slots in order; keep `extraTempoTurn`
  rules; ties per decision 3 with `UseTurnOrderRollForTest`.
- `Company.Battle.order` and the slot index on events (data only; the strip
  is 82d).
- Tests: turn-order unit tests (fast first, second turn placement, slow
  last, opening bonus, ties, a felled unit loses its slot); integration
  through the real round in `modules/company` (a fast foe strikes before a
  slow companion; a Samurai with Iaijutsu opens round 1; a chant broken by a
  faster foe before release; retreat still lands at round start). Existing
  wiring tests that assumed leader-first order are updated with a note of
  why.
- Balance: re-run `TestBalanceAtLevel` and the mirror sims (decision 7);
  record numbers and any at-level knob change in this file.
- Help: `help tempo` (order by tempo, the slot rule, opening strikes), `help
  speed` (Speed sets when you act as well as how often), `help combat` hub
  sentence, `help samurai`/`help packlord` where "strikes first" is now
  literal; tutorial hint where the combat lesson explains tempo.
- Acceptance: within a round, actions resolve in slot order across both
  sides; turn counts unchanged; at-level targets met or the drift recorded.

### Phase 82c: the battle clock and per-action beats (Go)

- Battle clusters on their own clock (decision 4) via a `NewTurn` listener;
  other fights keep `CombatOnCadence`.
- Pacer: per-action beats (decision 5) replace the per-line window for
  battle lines; the window clamp stays only for fixed-cadence fights; a dim
  round line at each round start; cluster pace = slowest non-off pace.
- Audit and convert in-battle game-round effects (decision 6); fix
  `GetDurations` display.
- Copyover: next round due at once; held lines flushed (existing).
- Tests: pacer unit tests for beats; integration in
  `modules/company/wiring_pace_test.go` (a round's actions arrive one beat
  apart; the next round waits for playback; two allied companies share one
  clock at the slower pace; `off` gets lines at once but the same clock;
  copyover resumes); a buff applied in battle lasts the same battle rounds
  whatever the round length; non-battle combat still on the old cadence.
- Measure seconds per fight at normal (target 25-45 s band-middle, at most
  90 s for five against five) and record it.
- Help: `help combatpace` rewritten (beats, slowest pace in a shared fight,
  `off`), `help combat` hub, the tutorial combat-pace hint
  (`stages.go`: "A round's lines come to you one by one" becomes one action
  at a time).
- Acceptance: at normal, no two actions of a battle arrive less than 0.6 s
  apart; at-level fight length in range; `make smoke` passes.

### Phase 82d: turn order on screen and tempo shown

- Battle screen strip (actor highlighted, acted dimmed, unseen foes merged),
  tempo in the figure tooltip; Combat tab text list; tempo on the Company and
  Character panels.
- The animation plays one event batch per slot (`battle-timeline.js` already
  plans per batch; check it waits for the slot rather than chaining).
- Tests: `make js-test` for the strip's planning (pure, under Node); a
  browser check at desktop and 360 px; a GMCP payload test for `order`.
- Help: `help battlescreen` (the strip), `help tempo` (where to see it),
  `help company`/character panel help if they list fields.
- Acceptance: a player can see who acts next and why (tempo) without
  reading the text.

## Follow-ups considered and left out

- **Mid-round input** (retreat cutting a round short): not built; the round
  is already resolved, and Robinson's rule allows only once-a-round inputs.
- **Ogre Battle's fixed attacks per battle by row:** not adopted; the
  formation and reach rules already make rows matter, and the meter gives
  speed its frequency.
- **Widening the tempo range** (0.6-1.5) to make speed differences larger:
  not now; measure 82b first. A wider range is a class-balance change.
