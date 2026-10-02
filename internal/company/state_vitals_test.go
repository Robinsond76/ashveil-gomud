package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 33h2: saved vitals resolve against today's limits as points, never
// rescaled; an old record (nil) spawns full; a resurrection's share
// resolves at the spawn.
func TestVitalsResolve(t *testing.T) {
	cases := []struct {
		name               string
		v                  *Vitals
		limit, manaMax     int
		wantHealth, wantMP int
	}{
		{"old record spawns full", nil, 40, 12, 40, 12},
		{"saved points kept", &Vitals{Health: 8, Mana: 3}, 40, 12, 8, 3},
		{"a raised maximum grants nothing", &Vitals{Health: 30, Mana: 10}, 60, 20, 30, 10},
		{"a lowered limit clamps", &Vitals{Health: 30, Mana: 10}, 25, 6, 25, 6},
		{"never below 1 health", &Vitals{Health: -4, Mana: -2}, 40, 12, 1, 0},
		{"zero health spawns at 1", &Vitals{Health: 0, Mana: 0}, 40, 12, 1, 0},
		{"resurrection's half", &Vitals{Percent: 50}, 41, 13, 20, 6},
		{"half of a tiny limit is still 1", &Vitals{Percent: 50}, 1, 0, 1, 0},
		{"percent over 100 caps", &Vitals{Percent: 250}, 40, 12, 40, 12},
		{"a zero limit is treated as 1", &Vitals{Health: 5}, 0, 0, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hp, mp := tc.v.Resolve(tc.limit, tc.manaMax)
			assert.Equal(t, tc.wantHealth, hp)
			assert.Equal(t, tc.wantMP, mp)
		})
	}
}

func TestVitalsYAMLAndClone(t *testing.T) {
	var old MemberState
	require.NoError(t, yaml.Unmarshal([]byte("level: 3\nexperience: 1200\n"), &old))
	assert.Nil(t, old.Vitals, "a pre-33h2 record has no vitals")

	s := sampleState()
	s.Vitals = &Vitals{Health: 7, Mana: 2}
	data, err := yaml.Marshal(s)
	require.NoError(t, err)
	assert.Contains(t, string(data), "vitals:")
	assert.NotContains(t, string(data), "percent", "omitted when unset")
	var back MemberState
	require.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, s.Vitals, back.Vitals)

	clone := s.Clone()
	clone.Vitals.Health = 99
	assert.Equal(t, 7, s.Vitals.Health, "Clone copies the vitals")

	data, err = yaml.Marshal(MemberState{Level: 1})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "vitals", "nil vitals are omitted")
}
