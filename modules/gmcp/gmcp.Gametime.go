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
	// LM2: a player's sight changes when they move or a light buff starts
	// or ends, so their Gametime goes out then too, not only each round.
	events.RegisterListener(events.RoomChange{}, g.sightChangeHandler)
	events.RegisterListener(events.BuffsAdded{}, g.sightChangeHandler)
	events.RegisterListener(events.BuffsRemoved{}, g.sightChangeHandler)
	// Holding or putting away a lantern changes the light shown at your hand.
	events.RegisterListener(events.EquipmentChange{}, g.sightChangeHandler)

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

	payload := gametimeBase()
	for _, user := range users.GetAllActiveUsers() {
		sendGametime(user, payload)
	}

	return events.Continue
}

// sightChangeHandler sends one player their Gametime (with their sight) at
// once when they move, gain or lose a buff, or change what they hold.
func (g *GMCPGametimeModule) sightChangeHandler(e events.Event) events.ListenerReturn {
	userId := 0
	switch ev := e.(type) {
	case events.RoomChange:
		userId = ev.UserId
	case events.BuffsAdded:
		userId = ev.UserId
	case events.BuffsRemoved:
		userId = ev.UserId
	case events.EquipmentChange:
		if len(ev.ItemsWorn) == 0 && len(ev.ItemsRemoved) == 0 {
			return events.Continue // gold or bank only
		}
		userId = ev.UserId
	}
	if userId <= 0 {
		return events.Continue
	}
	if user := users.GetByUserId(userId); user != nil {
		sendGametime(user, gametimeBase())
	}
	return events.Continue
}

// gametimeBase is the clock part of Gametime, the same for every player.
func gametimeBase() GMCPGametimeModule_Payload {
	gd := gametime.GetDate()

	return GMCPGametimeModule_Payload{
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
}

// sendGametime queues one player's Gametime: the shared clock plus what they
// can see from where they stand (the zone's weather, and their sight).
func sendGametime(user *users.UserRecord, payload GMCPGametimeModule_Payload) {
	if user == nil || !isGMCPEnabled(user.ConnectionId()) {
		return
	}
	p := payload
	room := rooms.LoadRoom(user.Character.RoomId)
	p.Weather = gametimeWeatherFor(room)
	if room != nil {
		sight := room.SightForUser(user)
		p.Sight = &sight
	}
	events.AddToQueue(GMCPOut{
		UserId:  user.UserId,
		Module:  `Gametime`,
		Payload: p,
	})
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

	// Sight (LM2) is how far the player can see where they stand, so the
	// map can darken and remember what is out of sight.
	Sight *rooms.Sight `json:"sight,omitempty"`
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
