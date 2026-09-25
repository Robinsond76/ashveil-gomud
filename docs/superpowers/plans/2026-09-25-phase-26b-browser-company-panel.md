# Phase 26b: Browser Company Panel — Implementation Plan

Design: [26b spec](../specs/2026-09-25-phase-26b-browser-company-panel-design.md).
Each task is tests first.

- [x] **Task 1: summary additions** (`internal/companyview`). Tests first:
  `TestSummaryLeaderCell`, `TestOnRefreshFiresWithSummary`. Then the
  leader's cell source and the `OnRefresh` hook, fired by `Refresh`.
- [x] **Task 2: payloads** (`modules/gmcp/gmcp.Company.go`). Tests first:
  `TestCompanyPayloadShape` (present, awaiting, dead; nulls for unknowns;
  member keys), `TestCompanyVitalsPayload`.
- [x] **Task 3: change detection and delivery.** Tests first:
  `TestCompanySendsOnlyOnChange`, `TestCompanyVitalsOnlyWhenOnlyVitalsChange`,
  `TestCompanyResentAfterSpawnAndRequest`, `TestCompanyOnlyToLeader`,
  `TestCompanyWebRequest`.
- [x] **Task 4: web client** (`window-party.js`). Company section and safe
  DOM for both sections; state read from `Client.GMCPStructs`; focus kept.
- [x] **Task 5: browser check** (`scripts/browser/company-panel-check.mjs` and
  a harness page). Playwright against the real script with a stub client:
  sections, formation, markup-as-text name, vitals merge, 360px, keyboard,
  accessibility tree.
- [x] **Task 6: wiring** (`modules/death/wiring_gmcp_company_test.go`).
  Through `plugins.Load` with GMCP captured: login, recruit, damage, death,
  resurrection, clock.
- [x] **Task 7: docs and verification.** GMCP help and module guide,
  `go test -race ./...`, `make generate`, `make validate`, `jshint`,
  independent review, `docs/PROJECT_STATUS.md`.
