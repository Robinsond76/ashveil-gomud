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
// queues an event during that dispatch would inherit the cause too. The only
// regular off-loop producer is the input worker, whose player Input events
// never inherit one; anything else off the loop queues Broadcasts or web
// client commands, which pacing ignores.

var currentCause atomic.Uint64

// Cause is the combat round that caused the event being dispatched, or 0.
func Cause() uint64 { return currentCause.Load() }

// WithCause runs fn with round as the current cause: events it queues, and
// the events their listeners queue in turn, carry round.
func WithCause(round uint64, fn func()) {
	prev := currentCause.Swap(round)
	defer currentCause.Store(prev)
	fn()
}

// causeFor is the cause a newly queued event carries. A player's typed
// command never inherits one: its output is theirs, not the round's.
func causeFor(e Event) uint64 {
	if in, ok := e.(Input); ok && in.UserId > 0 && in.MobInstanceId == 0 {
		return 0
	}
	return currentCause.Load()
}
