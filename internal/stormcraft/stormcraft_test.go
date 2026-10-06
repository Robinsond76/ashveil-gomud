package stormcraft

import "testing"

func TestKindOfNamesTheWeatherEachSpellCalls(t *testing.T) {
	for spell, want := range map[string]Kind{"callfog": Fog, "chillwind": Chill, "rain": Rain} {
		if got, ok := KindOf(spell); !ok || got != want {
			t.Errorf("%s calls %q, got %q %v", spell, want, got, ok)
		}
	}
	for _, spell := range []string{"lightning", "gust", "mm", ""} {
		if _, ok := KindOf(spell); ok {
			t.Errorf("%s calls no weather", spell)
		}
	}
	for _, k := range Kinds {
		if k.Name() == "" || k.EndLine() == "" || k.Effect() == "" {
			t.Errorf("%s has no name, end line or effect", k)
		}
	}
}

func TestRainFeedsLightningAndFogWeakensSpells(t *testing.T) {
	if got := LightningDamage(20, Rain); got != 30 {
		t.Errorf("lightning in rain: %d", got)
	}
	for _, k := range []Kind{None, Fog, Chill} {
		if got := LightningDamage(20, k); got != 20 {
			t.Errorf("lightning in %q: %d", k, got)
		}
	}
	if got := FoggedSpell(10); got != 9 {
		t.Errorf("a fogbound caster's spell: %d", got)
	}
	if Triggers(Rounds) != 4 {
		t.Errorf("a 3-round call counts 4 ticks, got %d", Triggers(Rounds))
	}
}
