package archetype

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCapabilitiesShareCurrentClassRanksAndAutoskillEligibility(t *testing.T) {
	m := wired(t, 100)
	u := walker(t, 189, 97590)
	require.Empty(t, archetypes.PlayerCapabilities(u.UserId), "unchosen class has no speculative capabilities")
	m.choose(u, "ranger", true)
	u.Character.SetSkill("track", 3)
	caps := archetypes.PlayerCapabilities(u.UserId)
	require.Len(t, caps, 3)
	for _, c := range caps {
		assert.Equal(t, 3, c.Rank)
		assert.True(t, c.Enabled)
		assert.Equal(t, "track", c.Skill)
	}
	best, ok := m.BestSpecialist(u.UserId, archetypes.UtilityTrail)
	require.True(t, ok)
	assert.Equal(t, caps[0].Rank, best.Level)
	_, err := m.autoskillCommand("trail off", u, &rooms.Room{}, 0)
	require.NoError(t, err)
	caps = archetypes.PlayerCapabilities(u.UserId)
	assert.Equal(t, "Autoskill off", caps[0].Reason)
	_, ok = m.BestSpecialist(u.UserId, archetypes.UtilityTrail)
	assert.False(t, ok)
	u.Character.SetSkill("track", 0)
	caps = archetypes.PlayerCapabilities(u.UserId)
	assert.Equal(t, "Requires trained track", caps[0].Reason)
	// A current class change uses the new table, not any former class's skills.
	m.registry.Players[u.UserId] = "warrior"
	u.Character.SetSkill("brawling", 2)
	caps = archetypes.PlayerCapabilities(u.UserId)
	require.Len(t, caps, 2)
	assert.Equal(t, "watch", caps[0].ID)
	assert.Equal(t, "fieldsmith", caps[1].ID)
	for _, c := range caps {
		assert.Equal(t, "camp", c.Group)
		assert.Equal(t, 2, c.Rank)
	}
	m.registry.Players[u.UserId] = "wizard"
	u.Character.SetSkill("cast", 1)
	u.Character.SpellBook = nil
	caps = archetypes.PlayerCapabilities(u.UserId)
	for _, c := range caps {
		if c.ID == "light" {
			assert.Equal(t, "Requires floatinglight", c.Reason)
		}
	}
}
