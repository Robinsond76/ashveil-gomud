# GoMud Baseline Verification

Baseline fork point: `39e44013 fix(telnet): stop Mudlet masking all input for the whole session (#633)`

Verified on 2026-09-16 on macOS arm64 with Go `go1.24.3`; the repository declares Go `1.24.0`.

## Compatibility adjustment

The initial `make help` run failed on macOS BSD awk because the help recipe used an unescaped `/` inside an awk slash-delimited regular expression. The Ashveil fork escapes that slash in `Makefile` and adds `scripts/make_help_test.go`, which runs the real `make help` target. The regression test failed before the recipe change and passes afterward.

This is the only non-documentation modification made before the Ashveil gameplay audit.

## Checks

| Check | Result | Evidence |
|---|---|---|
| `make help` | Pass | Lists documented targets after the BSD awk compatibility fix. |
| `go test ./scripts -run '^TestMakeHelpListsDocumentedTargets$' -count=1` | Pass | Executes the real Make target and verifies usage and documented-target output. |
| `make validate` | Pass | Go formatting check and vet completed on Go 1.24.3. |
| `make test` | Pass | Generation, JavaScript lint, Docker-backed Lua lint, and `go test -race ./...` completed. Docker Desktop was required for the bundled Lua-lint fallback. |
| `make build` | Pass | Produced `go-mud-server`. |
| Local server startup | Pass | Server logged `Server Ready`; default world data loaded. |
| `GET /webclient` | Pass | HTTP 200. |
| `GET /admin/` without credentials | Pass | HTTP 401, confirming the protected admin endpoint is reachable. |

## Scope not exercised

- No admin password was reset and no account credential was changed.
- Startup produced normal generated runtime state in the default world; no authored world data or gameplay configuration was edited.

| Capability | Interactive baseline status | Reason / source-audit evidence |
|---|---|---|
| Player login | Blocked by policy | No deliberately created local credential was used; `users.UserRecord.PasswordMatches` was inspected in `internal/users/userrecord.go`. |
| Ordinary room movement | Not exercised | No login session; normal movement is source-audited at `internal/usercommands/go.go: Go` and `internal/rooms/roommanager.go: MoveToRoom`. |
| Default combat | Not exercised | No login session; round combat is source-audited at `internal/hooks/NewRound_DoCombat.go` and `internal/combat/`. |
| Player party creation | Not exercised | No login session; party command and in-memory party model are source-audited at `internal/usercommands/party.go` and `internal/parties/parties.go`. |
| Hired mercenary lifecycle | Not exercised | No login session; hire/list path is source-audited at `internal/usercommands/buy.go`, `list.go`, and `internal/mobs/mobs.go`. Ownership/following/persistence remains a Phase 2 proof point. |
