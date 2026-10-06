package battle

import "github.com/GoMudEngine/GoMud/internal/stormcraft"

// Phase 39c: a Shaman's weather belongs to a battle. It is runtime state
// only, like the battle itself: a battle's end ends it, and a restart or
// copyover resumes fights as new battles. At most one weather at a time; a new
// call replaces it.

// Weather is the battle's weather and the combat rounds it has left.
type Weather struct {
	Kind stormcraft.Kind
	Left int
}

// CallWeather sets the player's battle weather for rounds combat rounds,
// replacing any other, and returns the weather it replaced. ok is false with
// no battle.
func CallWeather(userId int, kind stormcraft.Kind, rounds int) (replaced stormcraft.Kind, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	b, found := battles[userId]
	if !found || kind == stormcraft.None || rounds < 1 {
		return stormcraft.None, false
	}
	replaced = b.Weather.Kind
	b.Weather = Weather{Kind: kind, Left: rounds}
	return replaced, true
}

// WeatherOf is the player's battle weather: None when it has none.
func WeatherOf(userId int) Weather {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		return b.Weather
	}
	return Weather{}
}

// WeatherAgainst is the weather of a battle that has the mob among its
// enemies: what its foes' weather is. None when it fights no battle with one.
func WeatherAgainst(instanceId int) stormcraft.Kind {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range battles {
		if b.Enemies[instanceId] && b.Weather.Kind != stormcraft.None {
			return b.Weather.Kind
		}
	}
	return stormcraft.None
}

// WeatherEnded is a weather that ran out in one battle.
type WeatherEnded struct {
	UserId int
	Kind   stormcraft.Kind
}

// TickWeather counts one combat round off every battle's weather and returns
// the weathers that have passed. Called once per combat round.
func TickWeather() []WeatherEnded {
	mu.Lock()
	defer mu.Unlock()
	var ended []WeatherEnded
	for uid, b := range battles {
		if b.Weather.Kind == stormcraft.None {
			continue
		}
		b.Weather.Left--
		if b.Weather.Left <= 0 {
			ended = append(ended, WeatherEnded{UserId: uid, Kind: b.Weather.Kind})
			b.Weather = Weather{}
		}
	}
	return ended
}
