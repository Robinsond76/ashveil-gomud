package weather

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/sky"
)

// SkyView is what an observer in one room can know about the sky. It is
// gathered by the caller (rooms.Room.SkyView) and rendered here, so the
// rendering stays pure.
type SkyView struct {
	// WeatherZone is the zone whose weather applies: the room's own zone
	// outdoors, or the glimpsed room's zone indoors.
	WeatherZone string
	// Indoor is true when the observer is indoors.
	Indoor bool
	// GlimpseExit names the exit an indoor observer can see outside
	// through; empty means no view of the outside.
	GlimpseExit string
	Night       bool
	// HasMoon is false for a world configured with no moon.
	HasMoon bool
	Moon    sky.MoonPhase
}

// SkyLines renders the sky for v using the registered weather provider.
// brief is the short form appended to a room's look output; otherwise it
// is the full weather command report.
func SkyLines(v SkyView, brief bool) []string {
	condition, tracked := CurrentCondition(v.WeatherZone)
	return RenderSky(v, condition, tracked, !brief)
}

// RenderSky renders v given the weather condition c for v.WeatherZone
// (tracked false when that zone has no weather). full selects the weather
// command report rather than the short look form.
func RenderSky(v SkyView, c Condition, tracked bool, full bool) []string {
	var lines []string

	if v.Indoor {
		if v.GlimpseExit == "" {
			if full {
				lines = append(lines, "You can't see the sky from in here.")
			}
			return lines
		}
		if tracked {
			lines = append(lines, fmt.Sprintf("Through the %s exit: %s", v.GlimpseExit, c.Description))
		} else if full {
			lines = append(lines, fmt.Sprintf("Through the %s exit you can't make out the weather.", v.GlimpseExit))
		}
		if full {
			lines = append(lines, fogLines(c, tracked)...)
		}
		return lines
	}

	cloudCover := 0
	if tracked {
		if full {
			lines = append(lines, fmt.Sprintf("Sky: %s.", sky.CloudCoverName(c.CloudCover)))
		}
		lines = append(lines, c.Description)
		if full {
			lines = append(lines, fogLines(c, tracked)...)
		}
		cloudCover = c.CloudCover
	} else if full {
		lines = append(lines, "You can't tell what the weather is like here.")
	}

	if v.Night && v.HasMoon {
		if sky.MoonVisible(cloudCover) {
			lines = append(lines, moonLine(v.Moon))
		} else if full {
			lines = append(lines, "Clouds hide the moon.")
		}
	}
	return lines
}

func fogLines(c Condition, tracked bool) []string {
	if !tracked {
		return nil
	}
	switch {
	case c.VisibilityMod <= -2:
		return []string{"Thick fog swallows everything beyond arm's reach."}
	case c.VisibilityMod < 0:
		return []string{"Fog limits how far you can see."}
	}
	return nil
}

func moonLine(p sky.MoonPhase) string {
	switch p {
	case sky.New:
		return "It is a moonless night: the moon is new."
	case sky.Full:
		return "A full moon lights the night sky."
	}
	return fmt.Sprintf("The moon hangs in the night sky, a %s.", p.Name())
}
