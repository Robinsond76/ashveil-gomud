# Events Package Guide

## Scope

- Use this file for event definitions, listener registration behavior, queue behavior, priorities, and event-flow control in `internal/events`.
- This package is a core integration point across the engine and modules.

## Working Rules

- Preserve compatibility of existing event types unless the task explicitly changes an event contract.
- Be careful with listener ordering, priority behavior, and unique-event semantics. Small changes here can shift game behavior far from this package.
- Prefer adding narrowly scoped events over overloading existing events with unrelated responsibilities.
- If a package relies on event ordering, document that assumption in the consuming package too instead of hiding it only here.
- Avoid introducing event-side policy that belongs in handlers or modules.

## Causal combat tagging (Ashveil Phase 29f)

- `cause.go`: every queued event carries the combat round that caused it
  (`WithCause`, inherited while a caused event is dispatched). `Cause()`
  reads it during dispatch; `hooks.Message_SendMessage` uses it to pace
  combat text.
- The cause is set on the game loop only. A player's `Input` never
  inherits one, since the input worker queues it off the loop. Keep any new
  off-loop producer from queueing `Message` events, or exclude its event in
  `causeFor`.
- Requeued events keep their cause.

## Verification

- Run targeted `internal/events` tests for queue, ordering, or uniqueness changes.
- If the change affects a widely used event, verify at least one real consumer path in addition to package tests.
- Call out any untested downstream listeners when changing event contracts.

## Documentation

- Keep this file about event contracts and ordering risk, not an exhaustive event catalog.
