// Package light owns the Phase 14 light and visibility configuration (the
// darkness to-hit penalties), the player-facing light command, and ships the
// partylight buff flag and the Floating Light buff. The light model itself
// lives in internal/rooms (light.go); this module only configures and
// reports it. It stores no state and never touches the world clock.
package light

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//go:embed files/*
var files embed.FS

// LightModule configures and reports per-viewer light.
type LightModule struct {
	plug *plugins.Plugin
}

func init() {
	m := &LightModule{plug: plugins.New("light", "1.0")}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.AddUserCommand("light", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
}

func (m *LightModule) load() {
	rooms.SetDarknessPenalties(
		parsePenalty(m.plug.Config.Get("DarkHitPenalty"), rooms.DefaultDarkHitPenalty),
		parsePenalty(m.plug.Config.Get("DimHitPenalty"), rooms.DefaultDimHitPenalty),
	)
}

// parsePenalty reads a configured penalty, using def when missing or
// negative. An explicit 0 disables that penalty.
func parsePenalty(raw any, def int) int {
	var n int
	switch value := raw.(type) {
	case int:
		n = value
	case int64:
		n = int(value)
	case float64:
		n = int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return def
		}
		n = parsed
	default:
		return def
	}
	if n < 0 {
		return def
	}
	return n
}

// viewerLight is what light the viewer brings: their own, an ally's, or
// night vision.
type viewerLight struct {
	Own         bool
	Party       bool
	NightVision bool
}

var levelNames = [3]string{"dark", "dim", "bright"}

func lightReport(c rooms.LightConditions, ambient, vis int, v viewerLight, hitPenalty int) []string {
	lines := []string{fmt.Sprintf("Ambient light here: %s.", levelNames[clamp(ambient)])}
	lines = append(lines, c.Describe()...)
	if v.Own {
		lines = append(lines, "You carry your own light.")
	} else {
		lines = append(lines, "You carry no light.")
	}
	if v.Party {
		lines = append(lines, "An ally's light shines on you.")
	}
	if v.NightVision {
		lines = append(lines, "Your eyes see clearly in the dark.")
	}
	switch clamp(vis) {
	case 0:
		lines = append(lines, "You can see nothing.")
	case 1:
		lines = append(lines, "You can see this room, but not beyond its exits.")
	default:
		lines = append(lines, "You can see this room and its exits.")
	}
	if hitPenalty > 0 {
		lines = append(lines, fmt.Sprintf("Fighting here, you would strike at a disadvantage (-%d to hit).", hitPenalty))
	}
	return lines
}

func clamp(level int) int {
	if level < 0 {
		return 0
	}
	if level > 2 {
		return 2
	}
	return level
}

func (m *LightModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	c := user.Character
	v := viewerLight{
		Own:         c.HasBuffFlag(rooms.FlagLightSource),
		Party:       room.HasPartyLightFor(user.UserId),
		NightVision: c.HasBuffFlag(rooms.FlagNightVision),
	}
	vis := room.VisibilityForUser(user)
	lines := lightReport(room.LightConditions(), room.GetVisibility(), vis, v, rooms.HitPenaltyForVisibility(vis, false))
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}
