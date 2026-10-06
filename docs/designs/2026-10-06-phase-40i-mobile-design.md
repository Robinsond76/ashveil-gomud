# Phase 40i: Touch layout and installable web app

Scope is set by the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
This is the short design the roadmap asked the build thread to draft; the
review thread approves it.

## Goals

- The web client is playable one-handed on a phone: the map, room, company
  dock, battle screen and `walkto` all work at 360 to 430 px wide.
- Every control is a finger-sized target (44 px) that sends an ordinary command.
- The client installs as an app (manifest and icons).
- Nothing the player cannot read in text appears: the views are the same
  windows the desktop dock shows, fed by the same GMCP.

## Design

- **Detection.** `mobile.js` toggles `body.mobile` when the viewport is 820 px
  wide or less (`matchMedia`, so a rotated or resized window switches live).
  The desktop layout is unchanged; every rule in `mobile.css` sits under
  `body.mobile`.
- **Views.** One view at a time, chosen from a bottom bar: **Game** (the
  terminal), **Map** (the map panel), **Here** (Room Info, time, tutorial) and
  **Company** (the dock tab group). Docks become full-screen layers over the
  terminal; each docked panel carries `data-win` (the window id, or
  `group:dock`) so CSS decides which panel a view shows. The command box and
  the touch bar stay on screen in every view.
- **Touch bar.** Compass (N S E W U D), **Walk to...**, a **Stop** button while
  walking, and Look / Inventory / Status / Camp. The bar folds away
  (remembered). Walk to... lists the named places the map knows (rooms whose
  `maplegend` is set), nearest first, and sends `walkto [room]`; the server
  still decides if the walk is allowed (40d rules). With none mapped it offers
  `help walkto`.
- **Map.** Touch pan (one finger), pinch zoom (two), tap a tile for the 40d
  "Walk to [room]" menu; a drag or pinch never opens it. The hover tooltip is
  hidden on phones. `uiMenu` rows are 44 px with a viewport-capped height.
- **Battle screen.** Full screen on a phone; the picture takes the whole width
  at any scale (pixel art stays crisp); buttons are 44 px; allied companies are
  already pennants under 420 px (phase 45), with a larger touch hit area for
  the pennant and figures. The minimised badge sits above the bottom bars.
  Mid-battle inputs are still only retreat and the company focus.
- **Install.** `site.webmanifest` (the old file was empty) names the app
  Ashveil, starts at `/webclient-pure.html`, standalone, with the S1 app icons
  (192, 512, maskable). `webclient-pure.html` gains a viewport meta
  (device-width, `viewport-fit=cover`, `interactive-widget=resizes-content` so
  the keyboard shrinks the layout), theme colour, and the S1 favicon/touch icon.

## Decisions (owner delegated)

1. A width breakpoint, not user-agent or touch detection: a narrow desktop
   window is also a phone-shaped viewport, and the check can drive it.
2. Four views, not a drawer: Room Info and the clock share "Here" so the map
   gets the whole screen; the vitals strip stays at the top of Company.
3. The touch bar sends plain commands (not GMCP actions), so command history,
   the battle refusals and text parity all stay the same.
4. "Walk to..." lists only named places (legend rooms); arbitrary rooms are
   reached by tapping the map. A full visited-room list would be unreadable.
5. No service worker: the game needs a live connection, and current browsers
   install a manifest-only app. Offline play is out of scope.
6. Popped-out (floating) windows are hidden on phones; Settings > Reset Layout
   is unchanged. Autohide docks are ignored.
7. Terminal font is 13 px on phones (about 45 columns at 390 px).

## Acceptance

- `scripts/browser/mobile-check.mjs` passes at a 390 x 780 touch viewport:
  views, touch bar, walk menu, pan and pinch, Room Info gather strip, Camp tab,
  battle screen with an ally pennant tap, manifest, and return to desktop.
- Help: `help mobile` (configuration hub; aliases phone, touch, install...),
  linked from `help webclient`, `help walkto` and `help battlescreen`; tutorial
  hints in the Character and Departure lessons.
