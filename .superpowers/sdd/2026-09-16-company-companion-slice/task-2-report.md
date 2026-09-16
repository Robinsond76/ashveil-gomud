# Task 2 report: company persistence and lifecycle restoration

## RED/GREEN evidence

- RED: `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -run 'TestRestoreForLeader' -count=1` failed because `Runtime` and `CompanyModule` were undefined.
- GREEN: focused restoration and death tests pass:
  `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -run 'Test(RestoreForLeader|MobDeath)' -count=1`
- Final verification passes:
  `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -count=1`
  and `GOCACHE=/tmp/ashveil-go-cache go test ./internal/company ./modules/... -count=1`.
- `make generate` completed and generated `modules/all-modules.go` imports `modules/company`.

## Delivered files

- `modules/company/company.go`: module registration, persistence adapter, lifecycle restoration, and event handlers.
- `modules/company/runtime.go`: native GoMud runtime adapter for resolving, spawning, tracking, and detaching mobs.
- `modules/company/company_test.go`: fake runtime/store lifecycle coverage for restore, stale replacement, duplicate prevention, failure retention, and death cleanup.
- `modules/company/files/data-overlays/config.yaml`: default allowed companion mob ID 58.
- `modules/company/AGENTS.md`: module-specific durability and follower guidance.
- `modules/all-modules.go`: generated module import.

## Commit

The initial implementation commit was `2621b743d980c58109ce28ce1628cad71bf2d51a`; the reviewed implementation including the report was `8be7c1f98f5e2bfdcb7d706f17ec2017728e2623`.

## Self-review

- Durable registry records are retained when spawning fails; only stale runtime mappings are cleared.
- Live mappings prevent duplicate restoration; matching mob death clears only the runtime mapping.
- Persistence uses the required `companies` identifier and missing data initializes an empty registry.
- PlayerSpawn is used for both ordinary login and copyover restoration, while normal charm movement remains native.

## Concerns

No known concerns for Task 2. Player commands and allowlist consumption are intentionally deferred to Task 3.

## Fix round

- Root cause: `PlayerSpawn.RoomId` can be stale while the player is queued for entry; the handler was passing that event field directly to restoration.
- Fix: resolve `users.GetByUserId(evt.UserId)`, safely continue when absent, and restore using `user.Character.RoomId`.
- Regression test: `TestPlayerSpawnRestoresUsingCurrentUserRoom` uses event room 12 and live character room 44, asserting spawn room 44.
- RED: `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -run TestPlayerSpawnRestoresUsingCurrentUserRoom -count=1` failed with expected 44 vs actual 12.
- GREEN: `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -run 'Test(PlayerSpawnRestoresUsingCurrentUserRoom|RestoreForLeader|MobDeath)' -count=1` passed.
- Package verification: `GOCACHE=/tmp/ashveil-go-cache go test ./modules/company -count=1` passed.
- Fix commit: `7bfbf64944c075215e0c6966bddc976fbda52059`.
