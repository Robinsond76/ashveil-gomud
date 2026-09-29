package races

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRacePronounValidation(t *testing.T) {
	base := Race{Name: "test", Description: "test", Size: Medium}
	for _, tc := range []struct {
		value string
		want  string
		valid bool
	}{
		{"", "", true},
		{" SHE ", "she", true},
		{"it", "it", true},
		{"who", "", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			race := base
			race.DefaultPronouns = tc.value
			err := race.Validate()
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, tc.want, race.DefaultPronouns)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRacePronounDataRoundTrip(t *testing.T) {
	original := Race{RaceId: 99, Name: "test", Description: "test", Size: Medium, DefaultPronouns: "it"}
	data, err := yaml.Marshal(&original)
	require.NoError(t, err)
	assert.Contains(t, string(data), "defaultpronouns: it")

	var restored Race
	require.NoError(t, yaml.Unmarshal(data, &restored))
	assert.Equal(t, original.DefaultPronouns, restored.DefaultPronouns)

	var old Race
	require.NoError(t, yaml.Unmarshal([]byte("raceid: 99\nname: test\ndescription: test\nsize: medium\n"), &old))
	assert.Empty(t, old.DefaultPronouns)
}
