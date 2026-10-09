# Public Web Guide

## Scope

- Use this file for public HTML, shared public CSS, and web client JavaScript under `_datafiles/html/public/`.
- Pages in this directory are server-rendered templates except `webclient-pure.html`, which is the standalone web client shell.

## Working Rules

- Keep normal public pages wrapped with `{{template "header" .}}` and `{{template "footer" .}}`.
- Prefix static asset URLs with `{{ .CONFIG.FilePaths.WebCDNLocation }}` where the existing template pattern requires it.
- Preserve the existing public site and web client patterns. Reuse the current layout, CSS variables, and JS structure instead of introducing new frameworks or build tooling.
- Treat `webclient.html` as a thin wrapper and `webclient-pure.html` as the main client surface.
- When adding or changing a window module, follow the existing `static/js/windows/` pattern:
  - register through the current virtual window flow
  - read live state from `Client.GMCPStructs`
  - keep GMCP namespace handling local to the window
- Prefer extending existing shared client code before duplicating terminal, GMCP, docking, or modal logic.
- Ashveil windows (`window-company.js`, `window-combat.js`, `window-vitals.js`'s company rows, `window-tutorial.js`) set every server string with `textContent`, never `innerHTML`, use the theme's secondary text colour rather than `--t-text-dim` for readable muted text, and have a Playwright check in `scripts/browser/` (`dock-check.mjs` for the dock core, `dock-windows-check.mjs` for the dock's windows, `battle-check.mjs` for the battle screen).
- Layout (Phase 32g): the left column is the world (time, map, room, tutorial); the right is the company dock, one tab group (`tabGroup: 'dock'` in `WINDOW_DOCK_DEFAULTS`, whose order is the tab order; `groupHeader` for the vitals strip). A new player-facing window joins one of the two. Character sub-tabs are hosted through `window.CharacterTabs`; shared company helpers live in `company-data.js`. Menu entries that can't be undone use `uiMenu`'s `confirm`. Menus name items by the server's reference (`!<id>:<uuid>`), never by display name alone, when a command acts on a specific item.
- Phone layout (Phase 40i): `mobile.js` and `mobile.css` add `body.mobile` on a viewport of 820 px or less and show one view at a time (`body[data-mview]`: game, map, here, company); a docked panel's `data-win` says which view owns it. New controls on the phone layout must be 44 px targets that send an ordinary command, and every `body.mobile` rule stays under that class. `scripts/browser/mobile-check.mjs` drives the real `webclient-pure.html` at a phone viewport.
- Quick menu: `quickmenu.js` opens on Enter in an empty command box (and the phone touch bar's Menu button) and walks the arrow keys while that box is empty. Its entries come from `QuickMenu.build(state)` (pure, tested in `scripts/js/quickmenu.test.mjs`) and send ordinary commands, naming targets by the server's id. A new room action belongs in `build` so the menu offers only what would work; `scripts/browser/quickmenu-check.mjs` drives the real client.
- Pure client logic that can be tested without a browser (the battle screen's `battle-timeline.js`) lives in `static/js/` as a script that also `module.exports` for Node, with tests in `scripts/js/*.test.mjs` run by `make js-test`.
- Creation panel (Phase 72a): `window-creation.js` shows `Char.Creation` steps as a modal dialog of buttons and answers with the input a telnet player types (the option number, `back`, `skip`, a typed line); it decides nothing. Read the client as bare `Client` (it is a `const`, so `window.Client` is undefined). Player skin and hair colour repaint a figure through `Sprites.tinted(path, SpriteTint.look(skin, hair))`; any new window that draws a player figure passes its look. `scripts/browser/creation-panel-check.mjs` checks the panel (`BROWSER=firefox` for Firefox).
- Keep third-party vendored assets vendored. Do not casually replace or reformat them.

## Verification

- Run `make js-lint` when changing public JavaScript files covered by the repo lint path.
- For template or CSS changes, verify the affected page in a browser when practical.
- For web client changes, check the exact affected behavior: terminal rendering, GMCP-driven panels, modal behavior, iframe shell behavior, or static asset loading.
- If a change depends on CDN-prefixed assets, confirm the rendered URLs still use the current template pattern.

## Documentation

- Keep deep architecture notes out of this file. Put them in narrower docs or code comments if they are still useful.
- If you add a new recurring window or page convention, document the rule here in one or two bullets.
