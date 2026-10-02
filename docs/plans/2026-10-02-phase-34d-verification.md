# Phase 34d verification and handoff

Implemented against master `ba051303` in `.worktrees/phase-34d`.

Company Conditions reuses engine buff labels/descriptions, StatMods,
GetDurations, CombatRounds, and persisted company wounds. New company and
archetype/camping read-only provider seams feed the existing changed-only
GMCP extra lifecycle. Skills retains its array contract with additive max_level;
Char.Capabilities also appears in full Char responses. Ability specs, automatic
spell configuration, specialist rank resolution and camp recipe selection
remain authoritative. No save format, world clock, readiness or unlock changes.

Independent full-diff review accepted one production scope finding: missing
manual camp cooking. Added its owner-supplied view and shared recipe selection,
with trained/untrained rank, configuration, class independence, nonconsumption,
real GMCP refresh and camp cook command regressions. Follow-up review found no
additional blockers. No review findings were rejected. A new away-member test
initially left stale room membership on teardown; restoring the original room
fixed the fixture. The paired ownership/bystander regression passed three runs,
and the affected package suite passed afterward.

Verified after fixes:

- Focused new effects/ownership/expiry, strategy/spell/rank, cooking, help and
  tutorial pointer tests; complete GMCP, archetype, company, usercommands and
  tutorial packages.
- Chromium dock windows suite (including 34d safe text, keyboard/help, focus,
  actual rank maximum, manual cooking and narrow layouts), dock core suite and
  tutorial panel suite: all passed. The latter two used temporary copies with
  localhost URLs and installed Chromium; their assertions were unchanged.
- `make generate`, `make validate`, `make js-lint
  JSHINT=/workspace/ashveil-env/js/node_modules/.bin/jshint`, `make lua-lint`:
  passed. Lua reported zero warnings/errors.
- `go test -race ./...`: passed; `git diff --check`: passed.

Integration recheck discovered master `f4e28dda` (33h3 relocation/separation).
Integration compatibility and its final verification are recorded below before
pushing. No later phase is implemented by this session.
