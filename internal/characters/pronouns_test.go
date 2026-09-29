package characters

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func loadShippedRaces(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	mudlog.SetupLogger(nil, "", "", false)
	t.Chdir(root)
	races.LoadDataFiles()
}

func TestCombatPronouns(t *testing.T) {
	loadShippedRaces(t)

	forms := map[string]PronounForms{
		"he":   {Subject: "he", Object: "him", Possessive: "his"},
		"she":  {Subject: "she", Object: "her", Possessive: "her"},
		"they": {Subject: "they", Object: "them", Possessive: "their"},
		"it":   {Subject: "it", Object: "it", Possessive: "its"},
	}
	for value, want := range forms {
		t.Run(value, func(t *testing.T) {
			assert.Equal(t, want, PronounFormsFor(value))
		})
	}

	for _, value := range []string{" SHE ", "", "unknown"} {
		t.Run("normalizes "+value, func(t *testing.T) {
			want := forms["they"]
			if value == " SHE " {
				want = forms["she"]
			}
			assert.Equal(t, want, PronounFormsFor(value))
		})
	}

	cases := []struct {
		name      string
		character Character
		want      PronounForms
	}{
		{"race default", Character{RaceId: 21}, forms["it"]},
		{"unknown race", Character{RaceId: 9999}, forms["they"]},
		{"explicit override", Character{RaceId: 21, Pronouns: " she "}, forms["she"]},
		{"invalid override falls through race", Character{RaceId: 21, Pronouns: "who"}, forms["it"]},
		{"effective race change", Character{RaceId: 1, FormRaceId: 21}, forms["it"]},
		{"reptilian fallback", Character{RaceId: 8}, forms["they"]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.character
			assert.Equal(t, tc.want, tc.character.CombatPronouns())
			assert.Equal(t, before, tc.character, "rendering must not materialize defaults")
		})
	}
}

func TestPronounDataRoundTrip(t *testing.T) {
	loadShippedRaces(t)
	original := Character{RaceId: 21, Pronouns: "she", CombatNoun: "warden"}
	data, err := yaml.Marshal(&original)
	require.NoError(t, err)
	assert.Contains(t, string(data), "pronouns: she")
	assert.Contains(t, string(data), "combatnoun: warden")

	var restored Character
	require.NoError(t, yaml.Unmarshal(data, &restored))
	assert.Equal(t, original.Pronouns, restored.Pronouns)
	assert.Equal(t, original.CombatNoun, restored.CombatNoun)

	var old Character
	require.NoError(t, yaml.Unmarshal([]byte("raceid: 21\nname: old save\n"), &old))
	assert.Empty(t, old.Pronouns)
	assert.Empty(t, old.CombatNoun)
	assert.Equal(t, PronounForms{Subject: "it", Object: "it", Possessive: "its"}, old.CombatPronouns())
}
