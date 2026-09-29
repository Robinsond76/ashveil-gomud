package events

import "sync/atomic"

// Ashveil Phase 29f: causal combat tagging.
//
// A combat round's text must be told apart from everything else a player
// receives, so that it alone can be paced. Rather than mark every send, the
// queue carries a "cause": the combat round an event was caused by. Events
// queued while a cause is current inherit it, so a companion's death notice
// (MobDeath -> the company module -> a Message) is as much a part of the
// round as the blow that caused it.
//
// The current cause is set only on the game loop: by WithCause, and while
// ProcessEvents dispatches a caused event. A goroutine off the loop that
// queues an event during that dispatch would inherit the cause too, so the
// input worker queues what players type with AddTyped, which never takes
// a cause; anything else off the loop queues Broadcasts or system events
// that carry no combat text.
//
// "Typed" is the second mark: what a player typed, and everything it
// causes in turn. A typed message (a command's output, a tell) is never held
// back by pacing; an untyped one (a mob's action, a round tick's notice)
// waits behind a player's held combat lines.

var (
	currentCause atomic.Uint64
	currentTyped atomic.Bool
)

// Cause is the combat round that caused the event being dispatched, or 0.
func Cause() uint64 { return currentCause.Load() }

// Typed reports whether the event being dispatched came from what a player
// typed.
func Typed() bool { return currentTyped.Load() }

// WithCause runs fn with round as the current cause: events it queues, and
// the events their listeners queue in turn, carry round. They are not
// typed: a combat round is the game's doing.
func WithCause(round uint64, fn func()) {
	prevCause := currentCause.Swap(round)
	prevTyped := currentTyped.Swap(false)
	defer func() {
		currentCause.Store(prevCause)
		currentTyped.Store(prevTyped)
	}()
	fn()
}

// AddTyped queues what a player typed (the input worker's Input): it never
// inherits a combat round, and what it causes is typed.
func AddTyped(e Event, priority ...int) {
	add(e, 0, true, priority...)
}
