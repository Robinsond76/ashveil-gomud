package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/stretchr/testify/assert"
)

func TestPersonalityTemplateOverRace(t *testing.T) {
	loadShippedPronounData(t)
	wolf := &Mob{Character: *characters.New()}
	wolf.Character.RaceId = 11 // canine: wounded, 10

	rule, noise, ok := wolf.Personality()
	assert.True(t, ok)
	assert.Equal(t, "wounded", rule)
	assert.Equal(t, 10, noise)

	wolf.Targeting, wolf.TargetingNoise = "Nearest", 0
	rule, noise, ok = wolf.Personality()
	assert.True(t, ok)
	assert.Equal(t, "nearest", rule, "the template wins over its race")
	assert.Equal(t, 0, noise)

	plain := &Mob{Character: *characters.New()}
	plain.Character.RaceId = 19 // the training dummy: none
	_, _, ok = plain.Personality()
	assert.False(t, ok, "no personality: today's weakest")

	plain.Character.RaceId = 9999
	_, _, ok = plain.Personality()
	assert.False(t, ok, "an unknown race has none")
}

func TestShippedRacePersonalities(t *testing.T) {
	loadShippedPronounData(t)
	want := map[int]struct {
		rule  string
		noise int
	}{
		11: {"wounded", 10}, // canine
		6:  {"nearest", 0},  // undead
		5:  {"casters", 15}, // goblin
		7:  {"weakest", 25}, // insect
		14: {"weakest", 25}, // giant spider
		1:  {"weakest", 10}, // human
	}
	for id, w := range want {
		r := races.GetRace(id)
		if assert.NotNil(t, r, "race %d", id) {
			assert.Equal(t, w.rule, r.Targeting, "race %d", id)
			assert.Equal(t, w.noise, r.TargetingNoise, "race %d", id)
		}
	}
}
