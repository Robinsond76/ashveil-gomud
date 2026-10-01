package weather

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// Phase 33f2 Weather Sense: a wizard in the leader's company reads the sky.
// The forecast is the zone's foretold condition (rolled when the current
// one began), so it is never wrong. Level 1 tells which way the weather
// turns, 2 names it, 3 says roughly when, and 4 reads the neighbouring
// zones the room's exits lead into.

// forecastLines are the weather command's forecast lines for the leader,
// none without a forecaster present.
func (m *WeatherModule) forecastLines(user *users.UserRecord, room *rooms.Room, zone string) []string {
	if m.forecaster == nil || user == nil || room == nil {
		return nil
	}
	sp, ok := m.forecaster(user.UserId, room.RoomId)
	if !ok {
		return nil
	}
	reading, ok := m.reading(zone, sp.Level)
	if !ok {
		return nil
	}
	lines := []string{fmt.Sprintf(`<ansi fg="yellow">%s %s the sky: %s.</ansi>`, sp.Subject(), sp.Verb("read", "reads"), reading)}
	if sp.Level >= 4 {
		for _, other := range m.neighbourZones(room) {
			if r, ok := m.reading(other, sp.Level); ok {
				lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">  Over in <ansi fg="zone">%s</ansi>: %s.</ansi>`, other, r))
			}
		}
	}
	return lines
}

// reading is one zone's forecast at a forecaster's level.
func (m *WeatherModule) reading(zone string, level int) (string, bool) {
	current, next, at, ok := m.Forecast(zone)
	if !ok {
		return "", false
	}
	text := tendency(current, next)
	if level >= 2 {
		if next.Name == current.Name {
			text = fmt.Sprintf("more %s to come", strings.ToLower(next.Name))
		} else {
			text = fmt.Sprintf("%s is coming (%s)", strings.ToLower(next.Name), text)
		}
	}
	if level >= 3 {
		text += ", " + m.whenText(at)
	}
	return text, true
}

// tendency says which way the weather turns from current to next.
func tendency(current, next weather.Condition) string {
	var parts []string
	switch {
	case next.TemperatureMod < current.TemperatureMod:
		parts = append(parts, "turning colder")
	case next.TemperatureMod > current.TemperatureMod:
		parts = append(parts, "turning warmer")
	}
	switch {
	case next.CloudCover > current.CloudCover:
		parts = append(parts, "clouding over")
	case next.CloudCover < current.CloudCover:
		parts = append(parts, "clearing")
	}
	switch {
	case next.ExertionPct > current.ExertionPct:
		parts = append(parts, "harder going ahead")
	case next.ExertionPct < current.ExertionPct:
		parts = append(parts, "easier going ahead")
	}
	if len(parts) == 0 {
		return "the weather holds as it is"
	}
	return strings.Join(parts, ", ")
}

// whenText says roughly when a change due at round at comes.
func (m *WeatherModule) whenText(at uint64) string {
	now := m.roundFn()
	per := uint64(1)
	if m.roundsPerHour != nil {
		per = max(m.roundsPerHour(), 1)
	}
	if at <= now {
		return "any moment now"
	}
	hours := (at - now + per/2) / per
	switch {
	case hours < 1:
		return "within the hour"
	case hours == 1:
		return "in about an hour"
	default:
		return fmt.Sprintf("in about %d hours", hours)
	}
}

// neighbourZones lists the other weather zones a room's ordinary exits lead
// into (by each room's sky, as the weather command reads it; 33f2 review).
func (m *WeatherModule) neighbourZones(room *rooms.Room) []string {
	seen := map[string]bool{m.skyView(room).WeatherZone: true}
	var out []string
	for _, ex := range room.Exits {
		if ex.Secret {
			continue
		}
		target := rooms.LoadRoom(ex.RoomId)
		if target == nil {
			continue
		}
		if zone := m.skyView(target).WeatherZone; zone != "" && !seen[zone] {
			seen[zone] = true
			out = append(out, zone)
		}
	}
	sort.Strings(out)
	return out
}
