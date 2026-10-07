package townsfolk

import (
	"encoding/json"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// world is what the module reads outside its own state, so tests can run
// without the game.
type world interface {
	// Name is the signed-in player's character name ("" when not signed in).
	Name(userID int) string
	// Flag, SetFlag read and leave a company flag (the marks story events
	// leave).
	Flag(userID int, flag string) bool
	SetFlag(userID int, flag string) error
	// MemberTag reports whether a company member carries a tag (Phase 72
	// backgrounds register tags).
	MemberTag(userID int, key, tag string) bool
	// Weather is the lowercase weather name in a zone ("" when unknown).
	Weather(zone string) string
	Night() bool
	Rand(n int) int
	// Push sends the web client a GMCP message.
	Push(userID int, namespace string, payload any)
}

type liveWorld struct{}

func (liveWorld) Name(userID int) string {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return ""
	}
	return u.Character.Name
}

func (liveWorld) Flag(userID int, flag string) bool { return storyevents.HasCompanyFlag(userID, flag) }

func (liveWorld) SetFlag(userID int, flag string) error {
	return storyevents.SetCompanyFlag(userID, flag)
}

func (liveWorld) MemberTag(userID int, key, tag string) bool {
	for _, t := range storyevents.TagsFor(userID, key) {
		if t == tag {
			return true
		}
	}
	return false
}

func (liveWorld) Weather(zone string) string {
	c, ok := weather.CurrentCondition(zone)
	if !ok {
		return ""
	}
	return strings.ToLower(c.Name)
}

func (liveWorld) Night() bool { return gametime.IsNight() }

func (liveWorld) Rand(n int) int { return util.Rand(n) }

func (liveWorld) Push(userID int, namespace string, payload any) {
	if users.GetByUserId(userID) == nil {
		return
	}
	f, ok := usercommands.GetExportedFunction("SendGMCPEvent")
	if !ok {
		return
	}
	if send, ok := f.(func(int, string, any)); ok {
		if _, err := json.Marshal(payload); err != nil {
			mudlog.Error("townsfolk: payload", "error", err)
			return
		}
		send(userID, namespace, payload)
	}
}
