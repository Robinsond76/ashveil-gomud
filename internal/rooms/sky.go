package rooms

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/sky"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// Room tags that override a biome's indoor flag for a single room, e.g. an
// inn's common room inside a forest zone.
const (
	TagIndoor  = `indoor`
	TagOutdoor = `outdoor`
)

// IsIndoor reports whether the room is under a roof. An `outdoor` or
// `indoor` room tag overrides the biome's own flag.
func (r *Room) IsIndoor() bool {
	if r.HasTag(TagOutdoor) {
		return false
	}
	if r.HasTag(TagIndoor) {
		return true
	}
	biome := r.GetBiome()
	return biome != nil && biome.Indoor
}

// SkyView gathers what an observer in this room can know about the sky: the
// weather zone that applies, whether the room is indoors (and whether an
// exit gives a glimpse outside), and the time of day and moon phase from the
// shared clock. It reads the clock and never advances it.
func (r *Room) SkyView() weather.SkyView {
	return r.skyView(LoadRoom, gametime.GetDate(), sky.CycleDays())
}

func (r *Room) skyView(load func(int) *Room, gd gametime.GameDate, cycleDays int) weather.SkyView {
	v := weather.SkyView{
		WeatherZone: r.Zone,
		Night:       gd.Night,
		HasMoon:     gd.MoonCount > 0,
		Moon:        moonPhase(gd, cycleDays),
	}
	if !r.IsIndoor() {
		return v
	}
	v.Indoor = true
	v.WeatherZone = ``
	if exitName, outside := r.outdoorExit(load); outside != nil {
		v.GlimpseExit = exitName
		v.WeatherZone = outside.Zone
	}
	return v
}

// outdoorExit returns the first non-secret exit, in sorted name order, that
// leads to an outdoor room, so the glimpse is stable from look to look.
func (r *Room) outdoorExit(load func(int) *Room) (string, *Room) {
	names := make([]string, 0, len(r.Exits))
	for name, ex := range r.Exits {
		if !ex.Secret {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		other := load(r.Exits[name].RoomId)
		if other == nil || other.IsIndoor() {
			continue
		}
		return strings.TrimSpace(name), other
	}
	return ``, nil
}

// moonPhase is the moon's phase on the night gd belongs to.
func moonPhase(gd gametime.GameDate, cycleDays int) sky.MoonPhase {
	return sky.PhaseForDay(sky.NightOfDay(gd.DayNumber, gd.Hour24), cycleDays)
}
