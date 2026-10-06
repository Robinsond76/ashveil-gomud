# Phase 40g: battle animation and effects

Status: **approved 2026-10-06 under the owner's delegation**; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 5c. Art: set S4 in the
[sprite specification](2026-10-05-sprite-specification.md). Depends on
40e and 40f.

## Goal

The battle screen plays out each happening as it is narrated:
- fighters step forward and strike, archers shoot, casters chant and
  release;
- targets flinch, block, parry or dodge;
- statuses flicker over units;
- the fallen collapse, the beaten yield, and the routed run.

Damage numbers pop up in step with the "(6)" in the text. The animations
**never change or delay** the battle itself.

## Prior art

- 40e delivers `Company.Battle.Event` batches, released with the
  narration by the player's combat pace.
- 40f draws the static screen and the units' idle loops.
- **Combat pace** (`help combatpace`) is fast, normal, slow or off. A
  combat round is 8 seconds.

## Owner decisions

- OB64-style battle presentation, with OB64 as the guide (2026-10-05).
- Original art only.
- Narration style (still in force): dark and physical, damage in
  parentheses, no exclamation marks or ALL-CAPS. This applies to any text
  on the screen.

## Proposed for approval

### Event to animation

| Event (40e) | Animation |
|---|---|
| `attack` hit or crit | Melee: the attacker plays `walk` toward the target's side, then `attack`, then walks back. Ranged weapon types play `shoot` plus a projectile. The target shows the weapon's hit effect (slash, stab, blunt, cleave, claw, bite, arrow) and `hurt`. A crit adds the `crit` burst and large digits. Damage digits rise from the target |
| `attack` miss | The attacker swings; the target gets the `miss` feedback |
| `attack` with `defenses` | The target plays `block`, `parry` or `dodge` per strike, with its feedback icon. Units without that animation use `hurt` and the icon only |
| `cast-start`, `cast-progress` | `cast` frames 1–4 loop, plus the school's chant glow |
| `cast-complete` cast | `cast` frames 5–6, then the spell's effect on its target(s). Area effects span the target formation |
| `cast-complete` fizzled, interrupted or wasted | `chant-broken` for an interrupt. Fizzled and wasted show a short fade of the glow |
| `spell-hit` | The spell's impact plus damage digits |
| `heal` | The heal effect plus green digits. Held-back health shows as a dim second number |
| `windup-start` | `windup`, held, with the winding-up status icon |
| `windup-land` | `attack` from the held pose. Wasted shows a swing at empty air |
| `interrupt` | `shield-bash` or `chant-broken` on the victim |
| `status-applied`, `status-expired` | The icon is added or removed, and the overlay loop starts or stops. Armor broken plays `armor-shatter` once |
| `status-tick` | A tick flash plus digits. A lost action shows a grey "skip" pulse on the unit (an icon, no text) |
| `guard-used` | The guardian plays `guard-step` into the ward's lane, with `guard-intercept` |
| `ability` | Tackle: the attacker dashes and the target goes `prone`. Opening Strike and Aimed Shot add a brief highlight before the blow |
| `yield` | `yield` pose held, plus the `surrendered` marker |
| `flee` | Mirrored `walk` off the screen edge |
| `death` | `down`, then the last frame held, then the `fallen` marker |
| `mercy` | The spared enemy fades out, or for an execution the attacker's `attack` then the target's `down` |
| `fight-end` victory | The company plays `victory` |
| `fight-end` defeat or broken off | The company holds its idle pose and the screen fades |

### Timing

- Each happening's animation has a budget, so the screen keeps pace with
  the text:
  - normal pace: about 1.2 s for an action and 0.6 s for a reaction;
  - slow pace: the same lengthened by its pace's gap;
  - fast pace: half;
  - `off`: 0.3 s, with effects only.
- Animations queue **per unit**. A unit's new action waits for its previous
  one. Events for different units may overlap, as in OB64.
- **Never fall behind:** if the queue holds more than one round of work
  when a new round's events arrive, the client skips to the final state of
  the older events (poses, icons, fallen markers) and plays the new ones.
- Animation never sends anything to the server and never affects
  resolution.

### Settings

`battleAnimations` has three values:
- `full`;
- `reduced`: no projectiles' travel, no flashes or shake, short fades;
- `off`: the 40f static screen.

The default is `full`, or `reduced` when the system asks for reduced
motion.

### Code structure

- A pure **event-to-timeline** module (`battle-timeline.js`) turns event
  batches into scheduled steps. It is testable without a browser, under
  Node, which is installed with the repository's lint tooling.
- `window-battle.js` (from 40f) plays the steps on its canvas, reusing the
  sprite loader.
- An S4 **asset manifest** lists each unit's available animation files.
  A missing animation falls back: `attack` to `idle` plus a nudge, `cast`
  to `idle` plus a glow, any spell to its school's glow, `down` to a
  fade.

## State and persistence

Client only. There is no server change beyond 40e and 40f.

## Integration points

| Area | Change |
|---|---|
| A new `battle-timeline.js` | mapping, budgets, catch-up |
| `window-battle.js` | playback, effects layer, digits, settings |
| Static files | S4 animations, effects and the manifest |
| Tests | Node tests for the timeline module, wired into `make js-lint` or a new `make js-test` target that `make test` runs |

## Player help and tutorial

- `help battlescreen` (40f) gains what the animations show, the animation
  setting, and the reminder that the text is the full record.
- **Tutorial:** none new. The 40f Combat lesson hint covers the screen.

## Acceptance tests

1. **Timeline tests (Node):**
   - each 40e kind maps to its steps;
   - defenses map to block, parry or dodge;
   - a missing animation uses its fallback;
   - a backlog over one round collapses to the final state;
   - pace budgets are applied;
   - `off` produces only end states.
2. **Browser check** (scripted Chromium, a short recording or frame
   screenshots kept with the PR) of the tutorial practice fight:
   - a melee hit shows the step-in, strike, hit effect and digits;
   - a heal shows green digits;
   - a fall leaves the fallen marker;
   - victory plays.
3. With `battleAnimations: off`, the screen behaves exactly as in 40f.
4. Reduced motion disables flashes and projectile travel.
5. Battle outcomes, text output and timing are unchanged. The existing
   combat test suite passes untouched.
6. `help battlescreen` renders, and `make js-lint` passes (plus the timeline
   tests).

## Open questions

1. Should sound effects be added? GoMud has an audio module. The proposal
   is out of scope, as a later optional phase.
2. Confirm the per-action budgets.
