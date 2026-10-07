package stance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReadsNamesAndAliases(t *testing.T) {
	for word, want := range map[string]Stance{
		"heavy": Heavy, "Heavy Blows": Heavy, "heavy-blows": Heavy, "great": Heavy,
		"wall": Wall, "shield wall": Wall, "shield": Wall,
		"quick": Quick, "quick draw": Quick, "bow": Quick,
		"keen": Keen, "keen edge": Keen, "dagger": Keen,
	} {
		got, ok := Parse(word)
		assert.True(t, ok, word)
		assert.Equal(t, want, got, word)
	}
	for _, word := range []string{"", "off", "dance", "sword"} {
		_, ok := Parse(word)
		assert.False(t, ok, word)
	}
}

func TestEveryStanceIsDefinedOnceAndValid(t *testing.T) {
	seen := map[Stance]bool{}
	for _, d := range Defs {
		assert.False(t, seen[d.Key], d.Key)
		seen[d.Key] = true
		assert.True(t, d.Key.Valid())
		assert.False(t, d.Effect.IsZero(), "%s changes something", d.Key)
		assert.NotEmpty(t, d.Gain)
		assert.NotEmpty(t, d.Cost)
		assert.NotEmpty(t, d.Needs)
		_, ok := Parse(string(d.Key))
		assert.True(t, ok, "its key reads")
	}
	assert.True(t, None.Valid())
	assert.False(t, Stance("nonsense").Valid())
}

func TestFitsByFamily(t *testing.T) {
	glaive := Gear{TwoHanded: true, Class: "glaive", Family: "glaive"}
	staff := Gear{TwoHanded: true, Class: "staff"}
	bow := Gear{TwoHanded: true, Shooting: true, Class: "bow"}
	crossbow := Gear{TwoHanded: true, Shooting: true, Class: "crossbow"}
	dagger := Gear{Class: "dagger", Family: "dagger"}
	sword := Gear{Class: "sword"}
	shielded := Gear{Class: "sword", Shield: true}
	for name, tc := range map[string]struct {
		s    Stance
		g    Gear
		want bool
	}{
		"heavy glaive": {Heavy, glaive, true}, "heavy staff": {Heavy, staff, false}, "heavy bow": {Heavy, bow, false}, "heavy sword": {Heavy, sword, false},
		"keen dagger": {Keen, dagger, true}, "keen sword": {Keen, sword, false}, "keen glaive": {Keen, glaive, false},
		"quick bow": {Quick, bow, true}, "quick crossbow": {Quick, crossbow, false}, "quick dagger": {Quick, dagger, false},
		"wall shield": {Wall, shielded, true}, "wall bare": {Wall, sword, false},
		"none": {None, glaive, false}, "unknown": {Stance("x"), glaive, false},
	} {
		assert.Equal(t, tc.want, Fits(tc.s, tc.g), name)
		assert.Equal(t, tc.want, !EffectFor(tc.s, tc.g).IsZero(), name)
	}
	assert.Equal(t, []Stance{Heavy, Wall}, Available(Gear{TwoHanded: true, Class: "glaive", Shield: true}))
	assert.Empty(t, Available(Gear{}))
}

func TestScaleNeverZeroesALandedBlow(t *testing.T) {
	assert.Equal(t, 26, Scale(20, 30))
	assert.Equal(t, 9, Scale(10, -10))
	assert.Equal(t, 1, Scale(1, -20))
	assert.Equal(t, 10, Scale(10, 0))
	assert.Equal(t, 0, Scale(0, 25))
}

func TestDescribeSaysTheTrade(t *testing.T) {
	d, ok := Lookup(Heavy)
	assert.True(t, ok)
	assert.Equal(t, "Heavy blows (great weapon): blows land 30% harder, but 15 points less likely to hit.", d.Describe())
}
