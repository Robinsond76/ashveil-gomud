package chronicle

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// world is what the module reads outside its own state, so tests can run
// without the game.
type world interface {
	// Place is the title of the room the leader stands in ("" when unknown).
	Place(userID int) string
	// RoomTitle is a room's title ("" when it does not load).
	RoomTitle(roomID int) string
	// Name is the leader's character name, or fallback.
	Name(userID int, fallback string) string
	// MobName is a mob template's name ("" for 0 or an unknown id).
	MobName(mobID int) string
	// Online reports whether the user is signed in.
	Online(userID int) bool
	// Push sends the web client a GMCP message.
	Push(userID int, namespace string, payload any)
}

type liveWorld struct{}

func (liveWorld) Place(userID int) string {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return ""
	}
	return liveWorld{}.RoomTitle(u.Character.RoomId)
}

func (liveWorld) RoomTitle(roomID int) string {
	if roomID <= 0 {
		return ""
	}
	if r := rooms.LoadRoom(roomID); r != nil {
		return r.Title
	}
	return ""
}

func (liveWorld) Name(userID int, fallback string) string {
	if u := users.GetByUserId(userID); u != nil && u.Character != nil && u.Character.Name != "" {
		return u.Character.Name
	}
	return fallback
}

func (liveWorld) MobName(mobID int) string {
	if mobID <= 0 {
		return ""
	}
	if spec := mobs.GetMobSpec(mobs.MobId(mobID)); spec != nil {
		return spec.Character.Name
	}
	return ""
}

func (liveWorld) Online(userID int) bool { return users.GetByUserId(userID) != nil }

func (liveWorld) Push(userID int, namespace string, payload any) {
	f, ok := usercommands.GetExportedFunction("SendGMCPEvent")
	if !ok {
		return
	}
	if send, ok := f.(func(int, string, any)); ok {
		if _, err := json.Marshal(payload); err != nil {
			mudlog.Error("chronicle: payload", "error", err)
			return
		}
		send(userID, namespace, payload)
	}
}
