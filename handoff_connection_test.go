package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestConnectionLoopFollowsAHandOff: a connection loop's user is re-read
// after every read, so a tutorial replay's hand-off (Ashveil 32b) moves
// input, prompts, and a dropped link to the new user. Before login it stays
// nil, and a connection with no user yet keeps the one it knew.
func TestConnectionLoopFollowsAHandOff(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)

	real := users.NewUserRecord(7, 42)
	replay := users.NewUserRecord(900000001, 42)
	users.SetTestUser(real)
	users.SetTestConnection(42, real.UserId)
	assert.Same(t, real, currentUser(42, real))

	users.RemoveTestUser(real.UserId)
	users.SetTestUser(replay)
	users.SetTestConnection(42, replay.UserId)
	assert.Same(t, replay, currentUser(42, real), "the hand-off is followed")

	assert.Nil(t, currentUser(42, nil), "not logged in yet")
	assert.Same(t, real, currentUser(99, real), "no user on the connection: keep the known one")
}

// TestDisconnectResolvesTheUserOnTheGameLoop: a dropped link is reported
// by connection, so after a hand-off it is the user on the connection now
// who goes link-dead, never the one who left it.
func TestDisconnectResolvesTheUserOnTheGameLoop(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	mudlog.SetupLogger(nil, "", "", false)

	replay := users.NewUserRecord(900000001, 42)
	users.SetTestUser(replay)
	users.SetTestConnection(42, replay.UserId)

	w := &World{}
	w.handleDisconnect(disconnected{connId: 42, linkDead: true})
	assert.True(t, users.IsLinkDeadConnection(42), "the user on the connection now is link-dead")
	assert.Contains(t, users.GetExpiredLinkDeadUsers(^uint64(0)), replay.UserId)
}
