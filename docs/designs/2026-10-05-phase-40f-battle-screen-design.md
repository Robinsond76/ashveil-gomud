# Phase 40f: battle screen (static)

Status: **approved 2026-10-06 under the owner's delegation**; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 5b: a still battle screen with formations and health. Art:
set S3 in the [sprite specification](2026-10-05-sprite-specification.md).
Depends on 40e.

## Goal

When a battle starts, a battle screen opens in the web client, styled
after Ogre Battle 64. The company stands on the left and the enemy on the
right, side-on, each in their 3x3 formation, over a background chosen by
the terrain. Units show their health, status, role and morale. The screen
updates as the battle goes, with sprites in their idle poses. 40g adds
the motion.

## Prior art (code as of 2026-10-05)

- **The dock's Battle view** (`window-combat.js`, Phase 32g2) draws both
  formations in the narrow dock. It shows:
  - enemy labels and health in scout's words;
  - reach dots and target lines;
  - fallen and waiting groups;
  - a screen-reader list;
  - Retreat and the company focus buttons (30c).
- **Data feeds:**
  - `Company.Battle`: enemy cells, labels, health words, targets, dark,
    narrow, waiting groups, focus, guards and assessment;
  - `Company`: member cells, archetypes, roles;
  - `Company.Vitals`: company health and mana numbers;
  - `Company.Conditions`: member effects;
  - `Company.Battle.Event` (40e): happenings, including enemy statuses.
- **Formation:** 3x3. Row 1 is the front, row 3 the back, and columns 1–3
  run left to right (`help formation`).
- `window-modal.js` provides a modal window.

## Owner decisions

- An OB64-style separate battle screen, with OB64 as the guide (2026-10-05).
- Original art only.
- **The screen opens automatically** when a battle starts, with a setting
  to switch to manual (2026-10-05).
- Allied companies: undecided. The owner asked for a recommendation
  (below).

## Proposed for approval

### Opening and closing

- The screen opens when `Company.Battle` reports a battle for the player.
  A setting can turn this off (`battleScreen: auto | manual`); in manual
  mode a "Battle" button opens it.
- It can be minimised to a corner badge at any time. The battle continues
  regardless.
- At `fight-end` it shows the outcome for 3 seconds (victory, defeat or
  broken off, with no exclamation marks, per narration style), then closes.
- A new battle in the same room reopens it.

### Layout (320×180 virtual canvas, scaled by whole numbers to fit)

- **Background:** a biome mapping table in the client picks it (sprite
  specification S3). A zone override exists, such as Stormwatchers Keep
  using `ice-keep`. When `dark` is set, the screen is darkened.
- **Positions:** the company stands on the left, facing right; the enemy
  stands on the right, mirrored. Each side's 3x3 grid maps as follows:
  - **row** sets the distance from the centre line: row 1 is nearest, at
    x ±40 from centre, and each row steps 32 px outward;
  - **column** sets the depth lane: column 1 is upper and further, column
    3 lower and nearer, at y 120, 140 and 160, with a small x skew for
    depth.

  Units are drawn back to front by lane, so nearer units overlap further
  ones. Cells use the S3 `cell` marker.
- **Units:** the `idle` loop of the unit's sprite.
  - Company members use their class key (as in 40b). The leader is marked
    with a small crown pip.
  - Enemies use the `sprite` key sent by the server (below), or their
    race's fallback silhouette.
- **In darkness**, an enemy not yet labelled shows as `unknown-*` at 50%
  brightness, matching scout's rule.

### Allied companies (recommendation)

Under the [allied companies design](2026-10-01-phase-33d-allied-companies-design.md),
each company fights its own battle against the **shared** enemy, keeping
its own formation, focus and retreat. There is no shared 3x3 grid. The
screen should show the same thing:

- **Your company is centre stage:** your 3x3 is drawn full-size on the
  left, as above.
- **Allies stand behind as reserve formations.** Each allied company in
  the same encounter is drawn as a compact 3x3 at half scale, stacked
  behind and above yours on the left. It is labelled with its leader's
  name and an ally pennant. At most two are drawn; any more collapse into
  a "+N companies" pennant.
- **Allied units animate their own actions** (40e sends them). Their
  health shows in **bands only**, and they show only statuses that the
  narration reveals, because another company's private conditions are
  never shown.
- **Tap or click an allied formation to watch it full-size.** Your own
  formation shrinks into its place. This view is **read-only**: Retreat,
  focus and member menus act only on your own company, matching 33d's
  authority rules. Tap again, or start a new round, to swap back.
- **The enemy stays one formation on the right.** Target lines show which
  company an enemy is striking.
- **On phones**, allied formations collapse to pennants. Tapping a pennant
  still swaps the view.

Why this approach:
- it keeps OB64's readable 3-against-3 clash at the centre;
- it respects 33d's rule of no shared grid with each company in charge of
  itself;
- allies' contribution is still visible;
- it scales to several companies.

Alternatives considered:
- **A banner only:** simplest, but allies' actions go unseen.
- **Every company full-size side by side:** crowded beyond two companies,
  and it suggests a shared grid that doesn't exist.

**Server addition:** `Company.Battle` gains `allies`, a list of
`{leader, members: [{id, label, sprite, cell, health}]}`, with `health` in
bands. It covers only companies in the player's party that are in the
same encounter.

### Information shown

| Element | Company members | Enemies |
|---|---|---|
| Health bar | Exact, from `Company.Vitals` | **Banded only:** five steps from scout's words, never numbers |
| Status icons | From `Company.Conditions` and 40e | From 40e events (applied/expired) |
| Role icon | From `Company` (fighter, healer, caster, guardian; controller after 38a) | — |
| Morale | Company nerve, from the company feed | Yield and flee markers from 40e |
| Target | Hover or tap lights its target, as in the dock | Same |
| Fallen and surrendered | `fallen` marker in its cell | `fallen` or `surrendered` |

- **Header:** the enemy group's name, the round, battlefield-condition
  banners (dark, narrow, ambush, cold…) and waiting groups.
- **Footer:** Retreat and the focus buttons. They send the same commands
  as the dock and follow the same disabled rules.
- **Narration** stays in the main output. The screen never replaces it.

### Server additions

- `Company.Battle.enemies[]` gains `sprite`. Its value comes from a new
  optional mob-spec field `sprite:` in the mob YAML (for example `sprite:
  wolf-timber`), or else from a race fallback table (`unknown-humanoid`,
  `unknown-beast` or `unknown-large`).
- Shipped mobs get their `sprite` keys, from the sprite specification's
  enemy table.
- Nothing else changes: the existing feeds supply everything.

### Accessibility and devices

- The screen is decorative for assistive technology (`aria-hidden`). The
  dock's Battle view and the narration remain the accessible path, and are
  unchanged.
- At phone widths the canvas scales down to fit, with nothing cropped. A
  full touch layout comes in 40i.

## State and persistence

There is no new server state, and the mob `sprite` key is world data. The
client rebuilds the screen from the next snapshot after a reconnect or
copyover.

## Integration points

| Area | Change |
|---|---|
| `internal/mobs` | optional `Sprite` spec field |
| World data | `sprite` on shipped mobs |
| `modules/gmcp/gmcp.CompanyBattle.go` | enemy `sprite`, with the race fallback; `allies` |
| A new `window-battle.js` | screen, layout, layers, open/close, settings; reuses the 40b sprite loader |
| `window-combat.js` | a "Battle screen" button to open it in manual mode |

## Player help and tutorial

- **New page:** `help battlescreen`, with the aliases `battle screen` and
  `battle map`. It covers:
  - what the screen shows;
  - how the formation maps to the picture;
  - why enemy health is banded;
  - minimising and the setting;
  - that the battle runs on its own (link `help strategy`).

  Link it from `help combat` and `help webclient`.
- **Tutorial:** the Combat lesson (practice fight) adds a hint that the
  battle screen opens in the web client.

## Acceptance tests

1. `Company.Battle` enemies carry `sprite`. A mob without one gets its
   race fallback. A test over the world data checks that every shipped mob
   with `sprite` names an ID in the sprite specification's enemy list.
2. **Browser check** (scripted Chromium, screenshot kept), using the
   tutorial practice fight:
   - the screen opens;
   - both formations stand in the right cells;
   - company health matches the vitals;
   - enemy bars are banded;
   - a status applied in 40e shows its icon;
   - the screen closes after the fight ends.
3. Dark battle: unlabelled enemies show as dimmed silhouettes.
4. Retreat and focus buttons send the same commands as the dock.
5. In manual mode the screen does not auto-open.
6. With an allied company in the fight:
   - its reserve formation shows, with banded health only;
   - tapping swaps the view, read-only;
   - no private ally conditions appear in any payload.
7. Text output is unchanged.
8. `help battlescreen` renders, and `TestTutorialHelpPointersExist`
   passes.
9. `make js-lint` passes.

## Open questions

1. Approve the allied-company recommendation (above).

## Built (2026-10-06)

Built as designed, with the decisions listed in
[PROJECT_STATUS](../PROJECT_STATUS.md): code-drawn placeholder figures that
S3 art replaces with no code change (manifest-keyed), no shipped mob
`sprite:` keys yet, allied reserve formations deferred (no allied relay in
the 40e feed), conditions icons from events only, backdrop overrides for
the tutorial zone and Stormwatchers Keep waiting on S3. Help:
`help battlescreen`. Check: `scripts/browser/battle-check.mjs`.
