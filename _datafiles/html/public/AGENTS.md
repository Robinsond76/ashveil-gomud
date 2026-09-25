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
- Ashveil windows (`window-party.js` Company section, `window-tutorial.js`) set every server string with `textContent`, never `innerHTML`, use the theme's secondary text colour rather than `--t-text-dim` for readable muted text, and have a Playwright check in `scripts/browser/`.
- Keep third-party vendored assets vendored. Do not casually replace or reformat them.

## Verification

- Run `make js-lint` when changing public JavaScript files covered by the repo lint path.
- For template or CSS changes, verify the affected page in a browser when practical.
- For web client changes, check the exact affected behavior: terminal rendering, GMCP-driven panels, modal behavior, iframe shell behavior, or static asset loading.
- If a change depends on CDN-prefixed assets, confirm the rendered URLs still use the current template pattern.

## Documentation

- Keep deep architecture notes out of this file. Put them in narrower docs or code comments if they are still useful.
- If you add a new recurring window or page convention, document the rule here in one or two bullets.
