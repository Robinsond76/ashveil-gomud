package sigils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestParseAcceptsTheWaysAPlayerTypesAKind(t *testing.T) {
	for in, want := range map[string]Kind{"fire": Fire, "of fire": Fire, "Fire Sigil": Fire, " ward ": Ward, "of stillness": Stillness, "mending sigil": Mending} {
		got, ok := Parse(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "of", "frost", "sigil"} {
		_, ok := Parse(in)
		assert.False(t, ok, in)
	}
}

func TestEveryKindIsDescribed(t *testing.T) {
	assert.Len(t, Kinds, 4)
	for _, k := range Kinds {
		assert.NotEmpty(t, k.Name(), k)
		assert.NotEmpty(t, k.Effect(), k)
		assert.NotEmpty(t, k.Glow(), k)
		assert.Positive(t, k.ManaCost(), k)
	}
}

func TestASigilFadesByTheWallClockAndOnlyHoldsItsRoom(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := Lay(Fire, 7, now)
	assert.True(t, l.In(7, now))
	assert.False(t, l.In(8, now), "another room")
	assert.Equal(t, Minutes, l.MinutesLeft(now))
	assert.Equal(t, 1, l.MinutesLeft(now.Add(Minutes*time.Minute-time.Second)), "rounded up")
	assert.True(t, l.Live(now.Add(Minutes*time.Minute-time.Second)))
	assert.False(t, l.Live(now.Add(Minutes*time.Minute)), "faded")
	assert.Zero(t, l.MinutesLeft(now.Add(time.Hour)))
	assert.False(t, Laid{}.Live(now), "nothing laid")
}

// A sigil is saved with the character: it survives a restart by its Unix
// expiry, and an empty one writes nothing.
func TestASigilSurvivesARestart(t *testing.T) {
	now := time.Now()
	data, err := yaml.Marshal(struct {
		Sigil Laid `yaml:"sigil,omitempty"`
	}{Lay(Ward, 12, now)})
	assert.NoError(t, err)
	var back struct {
		Sigil Laid `yaml:"sigil,omitempty"`
	}
	assert.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, Lay(Ward, 12, now), back.Sigil)
	assert.True(t, back.Sigil.In(12, now.Add(5*time.Minute)), "still lit after a restart")
	assert.False(t, back.Sigil.Live(now.Add(20*time.Minute)), "faded while the server was down")

	empty, _ := yaml.Marshal(struct {
		Sigil Laid `yaml:"sigil,omitempty"`
	}{})
	assert.NotContains(t, string(empty), "sigil")
}

func TestWardCapGrowsWithLevel(t *testing.T) {
	assert.Equal(t, 4, WardCap(1))
	assert.Equal(t, 6, WardCap(6))
	assert.Equal(t, 14, WardCap(30))
	assert.Equal(t, 4, WardCap(0))
}
