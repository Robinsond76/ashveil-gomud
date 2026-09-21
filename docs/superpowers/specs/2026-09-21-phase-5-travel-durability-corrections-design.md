# Phase 5 Travel Durability Corrections Design

**Status:** Approved for implementation planning

**Goal:** Close Phase 5's two durability holes: a travel checkpoint must never
apply survival exertion twice across a process crash, and a company that
returns after an offline completion must be reconciled to its destination
before it can receive ordinary commands.

## Scope

This is a corrective extension of the approved Phase 5 travel-profile design.
It changes neither route duration nor the informational-only nature of hunger,
thirst, and fatigue. It does not add interruptions, player cancellation,
camping, global-time advancement, or new travel actions.

## Exactly-once exertion

An expedition checkpoint becomes a durable two-system operation. Before the
expedition module calls survival, it persists a pending charge in the travel
session. The pending charge has a stable operation ID, its target checkpoint,
and the incremental `survival.Exertion` amount calculated from the session's
last finalized checkpoint. The operation ID is deterministic from the stable
leader, session identity, and target checkpoint, so a restart computes the
same key rather than inventing a replacement charge.

The survival registry durably stores applied exertion-operation IDs per leader.
`ApplyCompanyExertion` receives an operation ID. Within its existing single
survival write it either applies the cost and records the ID, or recognizes the
same ID as already applied and returns successfully without changing needs.
It rejects a reused ID whose persisted amount differs from the requested cost.

After survival succeeds, expedition advances `LastExertionCheckpoint`, clears
the pending charge, and persists that finalized session. Recovery handles every
crash boundary:

1. No pending session save means survival was never called.
2. A persisted pending charge with no survival record retries the same ID and
   applies it once.
3. A persisted pending charge whose survival record exists retries the same ID
   as a no-op, then finalizes the expedition checkpoint.
4. A finalized checkpoint never creates another operation for that checkpoint.

This gives exactly-once logical charging across the separate plugin files
without pretending their file writes are one atomic transaction. Expired
operation IDs are not pruned in Phase 5: a route has only ten checkpoints, so
retaining them is the safer bounded proof of idempotency.

## Reconnect and completed-session recovery

The expedition module registers a player-spawn hook using the existing module
event pattern. On a leader spawn/login it locks the session registry and
reconciles that leader before other gameplay command handling:

- An in-progress session derives real UTC progress, synchronizes a pending or
  earned checkpoint, and reschedules its remaining completion timer.
- An overdue in-progress session finalizes its due checkpoint, persists
  `Completed`, and attempts arrival.
- A `Completed` record at the origin retries the destination move once; at the
  destination it only removes the record; elsewhere it remains for operator
  repair.

The hook emits the normal one-time arrival only when its destination move is
verified. A completed record no longer blocks movement merely because it is
present: normal movement is refused only while a session is actively
`Traveling`. Failed completion remains safe and visible through `look`/`travel
status`, but a returning player is reconciled automatically rather than being
stuck until a server reload.

## Failure rules

- Expedition cannot start or advance a new checkpoint if its session store is
  unavailable.
- A survival write failure leaves the same persisted pending operation for a
  safe retry; it does not advance the checkpoint or arrival.
- An expedition finalization-write failure leaves the persisted pending
  operation. A retry observes survival's operation ID and finalizes without a
  second cost.
- A pending operation whose survival record has conflicting cost is retained,
  logs an error, and blocks completion for operator repair.
- Recovery and reconnect preserve the rule that neither travel nor survival
  advances GoMud's global clock or round count.

## Tests and acceptance

- Unit tests cover pending-charge generation, stable IDs, and invalid/conflict
  cases.
- Survival module tests prove retrying an applied ID does not alter any member's
  needs and that the ID persists with the same write as its cost.
- Expedition tests simulate each crash boundary by reconstructing the module
  from its stores, proving the final charge equals the configured profile total
  exactly once.
- Lifecycle tests simulate normal logout followed by a new login after travel
  becomes overdue, proving the leader and current companions land at the
  destination and ordinary movement is available afterward.
- Focused race tests, `make generate`, `make validate`, and `go test -race ./...`
  must pass before the Phase 5 status record again claims completion.
