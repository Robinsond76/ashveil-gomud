package events

import "sync/atomic"

// Requester identifies the player whose command is running on the game loop.
// It is separate from Typed (which also marks messages) and never authorizes
// an action by itself. Queued follower orders capture and revalidate identity.
var requester atomic.Int64

func Requester() int { return int(requester.Load()) }

// WithRequester scopes script-generated follower orders to their requester.
// Only call on the game loop, like WithCause.
func WithRequester(userID int) func() {
	previous := requester.Swap(int64(userID))
	return func() { requester.Store(previous) }
}

// MemberOrder is runtime provenance, not a persistent action queue.
type MemberOrder struct {
	UserID     int
	RoomID     int
	MemberKey  string // empty for a temporary charmed follower
	CharmToken any    // opaque runtime charm identity; never persisted
}
