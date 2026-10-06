package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	g := GMCPGametimeModule{
		plug: plugins.New(`gmcp.Gametime`, `1.0`),
	}

	events.RegisterListener(events.NewRound{}, g.newRoundHandler)

}

type GMCPGametimeModule struct {
	plug *plugins.Plugin
}

func (g *GMCPGametimeModule) newRoundHandler(e events.Event) events.ListenerReturn {

	_, typeOk := e.(events.NewRound)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "NewRound", "Actual Type", e.Type())
		return events.Cancel
	}

	gd := gametime.GetDate()

	payload := GMCPGametimeModule_Payload{
		Calendar:   gd.Calendar + "/" + configs.GetServerConfig().MudName.String(),
		Hour:       gd.Hour,
		Hour24:     gd.Hour24,
		Minute:     gd.Minute,
		AmPm:       gd.AmPm,
		Day:        gd.Day,
		Month:      gd.Month,
		MonthName:  gametime.MonthName(gd.Month),
		Year:       gd.Year,
		Zodiac:     gametime.GetZodiac(gd.Year),
		Night:      gd.Night,
		DayStart:   gd.DayStart,
		NightStart: gd.NightStart,
		SunCount:   gd.SunCount,
		MoonCount:  gd.MoonCount,
	}

	for _, user := range users.GetAllActiveUsers() {
		if !isGMCPEnabled(user.ConnectionId()) {
			continue
		}
		// The weather is per zone, so each player's copy carries what
		// they can see from where they stand.
		p := payload
		p.Weather = gametimeWeatherFor(rooms.LoadRoom(user.Character.RoomId))
		events.AddToQueue(GMCPOut{
			UserId:  user.UserId,
			Module:  `Gametime`,
			Payload: p,
		})
	}

	return events.Continue
}

// /////////////////
// Gametime payload
// /////////////////
type GMCPGametimeModule_Payload struct {
	Calendar   string `json:"calendar"`
	Hour       int    `json:"hour"`
	Hour24     int    `json:"hour24"`
	Minute     int    `json:"minute"`
	AmPm       string `json:"ampm"`
	Day        int    `json:"day"`
	Month      int    `json:"month"`
	MonthName  string `json:"month_name"`
	Year       int    `json:"year"`
	Zodiac     string `json:"zodiac"`
	Night      bool   `json:"night"`
	DayStart   int    `json:"day_start"`
	NightStart int    `json:"night_start"`
	SunCount   int    `json:"sun_count"`  // expected to be 1 or 2
	MoonCount  int    `json:"moon_count"` // expected to be 0 - 3

	// Weather is the weather the player can see from their room (the
	// time panel's sky); absent indoors with no view out, or in a zone
	// with no weather.
	Weather *gametimeWeather `json:"weather,omitempty"`
}

// gametimeWeather is the weather a player can see: the same condition the
// room look line and the weather command report.
type gametimeWeather struct {
	Name        string `json:"name"`        // condition name: clear, overcast, rain, storm, fog, thick-fog, mist
	Description string `json:"description"` // the condition's room line
	CloudCover  int    `json:"cloud_cover"` // 0 (clear) to 3 (overcast)
	Visibility  int    `json:"visibility"`  // 0, or a fog penalty down to -2
	Indoor      bool   `json:"indoor"`      // seen through an exit rather than overhead
}

// gametimeWeatherFor is the weather visible from room, or nil when the room
// is unknown, indoors with no view outside, or in a zone with no weather.
func gametimeWeatherFor(room *rooms.Room) *gametimeWeather {
	if room == nil {
		return nil
	}
	view := room.SkyView()
	if view.Indoor && view.GlimpseExit == `` {
		return nil
	}
	c, ok := weather.CurrentCondition(view.WeatherZone)
	if !ok {
		return nil
	}
	return &gametimeWeather{
		Name:        c.Name,
		Description: c.Description,
		CloudCover:  c.CloudCover,
		Visibility:  c.VisibilityMod,
		Indoor:      view.Indoor,
	}
}
