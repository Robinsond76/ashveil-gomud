package gmcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
)

// Phase 39c review: the battle feed carries a Shaman's weather, so the
// battle screen and the Battle view can show what is up and for how long.
func TestBattleFeedCarriesTheWeather(t *testing.T) {
	if weatherFact(battle.Weather{}) != nil {
		t.Error("a clear sky sends no weather")
	}
	w := weatherFact(battle.Weather{Kind: stormcraft.Fog, Left: stormcraft.Triggers(stormcraft.Rounds)})
	if w == nil || w.Kind != "fog" || w.Name != "fog" || w.Rounds != 3 || !strings.Contains(w.Effect, "ranged") {
		t.Fatalf("fog just called: %+v", w)
	}
	if w := weatherFact(battle.Weather{Kind: stormcraft.Rain, Left: 1}); w == nil || w.Rounds != 1 || !strings.Contains(w.Effect, "Lightning") {
		t.Errorf("rain on its last round: %+v", w)
	}

	raw, _ := json.Marshal(buildBattle(battleFacts{InBattle: true, Weather: weatherFact(battle.Weather{Kind: stormcraft.Chill, Left: 3})}))
	if !strings.Contains(string(raw), `"weather":{"kind":"chill","name":"chill wind","rounds":2,`) {
		t.Errorf("payload: %s", raw)
	}
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true, Dark: true, Weather: weatherFact(battle.Weather{Kind: stormcraft.Fog, Left: 2})}))
	if !strings.Contains(string(raw), `"weather":{"kind":"fog"`) {
		t.Errorf("the dark hides the foes, not the weather: %s", raw)
	}
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true}))
	if strings.Contains(string(raw), "weather") {
		t.Errorf("clear sky: %s", raw)
	}
}
