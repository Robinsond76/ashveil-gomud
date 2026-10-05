package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestMobManaTemplate (Phase 35b review): an enemy template's manabase and
// manaperlevel reach every spawn's mana pool.
func TestMobManaTemplate(t *testing.T) {
	var spec Mob
	require.NoError(t, yaml.Unmarshal([]byte("mobid: 990036\nmanabase: 30\nmanaperlevel: 5\n"), &spec))
	require.Equal(t, 30, spec.ManaBase)
	require.Equal(t, 5.0, spec.ManaPerLevel)

	plain := &Mob{MobId: 990037, Character: *characters.New()}
	require.NoError(t, plain.Validate())
	withTemplate(t, plain)
	base := NewMobById(990037, 1, 10)
	require.NotNil(t, base)

	spec.Character = *characters.New()
	require.NoError(t, spec.Validate())
	withTemplate(t, &spec)
	m := NewMobById(990036, 1, 10)
	require.NotNil(t, m)
	assert.Equal(t, 30, m.Character.ManaBaseOverride)
	base10, per := m.Character.ManaRates()
	assert.Equal(t, 30, base10)
	assert.Equal(t, 5.0, per)
	assert.Greater(t, m.Character.ManaMax.Value, base.Character.ManaMax.Value, "a deeper pool than the default enemy")
}
