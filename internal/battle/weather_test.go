package battle

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/stormcraft"
)

func TestWeatherBelongsToABattleAndRunsOut(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	if _, ok := CallWeather(1, stormcraft.Fog, 4); ok {
		t.Error("no battle, no weather")
	}
	Begin(1, 10, 5, "p", []int{100, 101})
	Begin(2, 10, 5, "q", []int{200})

	replaced, ok := CallWeather(1, stormcraft.Fog, 4)
	if !ok || replaced != stormcraft.None {
		t.Fatalf("call: %q %v", replaced, ok)
	}
	if got := WeatherOf(1); got.Kind != stormcraft.Fog || got.Left != 4 {
		t.Errorf("weather: %+v", got)
	}
	if got := WeatherOf(2); got.Kind != stormcraft.None {
		t.Errorf("another player's battle has none: %+v", got)
	}
	if WeatherAgainst(100) != stormcraft.Fog || WeatherAgainst(101) != stormcraft.Fog {
		t.Error("its foes are under it")
	}
	if WeatherAgainst(200) != stormcraft.None || WeatherAgainst(999) != stormcraft.None {
		t.Error("another battle's foes, and a stranger, are not")
	}
	if cur, _ := Current(1); cur.Weather.Kind != stormcraft.Fog {
		t.Error("the battle's copy carries it")
	}

	// A new call replaces the old one.
	replaced, _ = CallWeather(1, stormcraft.Rain, 4)
	if replaced != stormcraft.Fog || WeatherOf(1).Kind != stormcraft.Rain {
		t.Errorf("replace: %q %+v", replaced, WeatherOf(1))
	}

	for i := 0; i < 3; i++ {
		if ended := TickWeather(); len(ended) != 0 {
			t.Fatalf("round %d: ended too soon: %+v", i+1, ended)
		}
	}
	ended := TickWeather()
	if len(ended) != 1 || ended[0].UserId != 1 || ended[0].Kind != stormcraft.Rain {
		t.Fatalf("it passes after its rounds: %+v", ended)
	}
	if WeatherOf(1).Kind != stormcraft.None || WeatherAgainst(100) != stormcraft.None {
		t.Error("and is gone")
	}
	if ended := TickWeather(); len(ended) != 0 {
		t.Errorf("nothing left to end: %+v", ended)
	}
}

func TestWeatherEndsWithItsBattle(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(1, 10, 5, "p", []int{100})
	CallWeather(1, stormcraft.Chill, 4)
	End(1)
	if WeatherOf(1).Kind != stormcraft.None || WeatherAgainst(100) != stormcraft.None {
		t.Error("a battle's end ends its weather")
	}
	Begin(1, 10, 9, "p", []int{100})
	if WeatherOf(1).Kind != stormcraft.None {
		t.Error("a new battle starts clear")
	}
	if _, ok := CallWeather(1, stormcraft.None, 3); ok {
		t.Error("no weather to call")
	}
	if _, ok := CallWeather(1, stormcraft.Fog, 0); ok {
		t.Error("no rounds, no weather")
	}
}
