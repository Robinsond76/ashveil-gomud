package blessings

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// who is the part of a signed-in character the module reads.
type who struct {
	Name string
	Iron bool
}

// world is what the module reads and does outside its own state, so tests
// can run without the game.
type world interface {
	// Character is the signed-in user's character (false when offline).
	Character(userID int) (who, bool)
	// User is the signed-in user (false when offline).
	User(userID int) (*users.UserRecord, bool)
	// Tell sends the player a line.
	Tell(userID int, text string)
	// Push sends the web client a GMCP message.
	Push(userID int, namespace string, payload any)
}

type liveWorld struct{}

func (liveWorld) Character(userID int) (who, bool) {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil || u.Character.Name == `` || u.IsReplay() {
		return who{}, false
	}
	return who{Name: u.Character.Name, Iron: u.Character.IsIron()}, true
}

func (liveWorld) User(userID int) (*users.UserRecord, bool) {
	u := users.GetByUserId(userID)
	return u, u != nil
}

func (liveWorld) Tell(userID int, text string) {
	if u := users.GetByUserId(userID); u != nil {
		u.SendText(text)
	}
}

func (liveWorld) Push(userID int, namespace string, payload any) {
	f, ok := usercommands.GetExportedFunction("SendGMCPEvent")
	if !ok {
		return
	}
	if send, ok := f.(func(int, string, any)); ok {
		if _, err := json.Marshal(payload); err != nil {
			mudlog.Error("blessings: payload", "error", err)
			return
		}
		send(userID, namespace, payload)
	}
}
