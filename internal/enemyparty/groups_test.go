package enemyparty

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthWord(t *testing.T) {
	cases := []struct {
		h, max int
		want   string
	}{
		{10, 10, "unhurt"}, {8, 10, "scratched"}, {5, 10, "wounded"}, {3, 10, "badly wounded"},
		{1, 10, "near death"}, {0, 10, "down"}, {5, 0, "unhurt"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, HealthWord(c.h, c.max), "%d/%d", c.h, c.max)
	}
}

func namedMob(t *testing.T, instanceId int, name, spawnGroup string) {
	t.Helper()
	m := testMob(t, instanceId, 10)
	m.Character.Name = name
	m.SpawnGroup = spawnGroup
}

// TestGroupsAreNamedAndFound: each party is named, a repeat name takes an
// ordinal, and a group is found by its kind, its noun, or #n, never by one
// member's name; a lone mob is found by its own.
func TestGroupsAreNamedAndFound(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false) // buff flags warn
	namedMob(t, 9101, "ruffian", "a")
	namedMob(t, 9102, "ruffian", "a")
	namedMob(t, 9103, "ruffian", "b")
	namedMob(t, 9104, "ruffian", "b")
	namedMob(t, 9105, "cave troll", "")
	room := testRoom(t, 990101, 9101, 9102, 9103, 9104, 9105)

	var names []string
	for _, g := range Groups(room) {
		names = append(names, g.Name)
	}
	assert.ElementsMatch(t, []string{"a band of ruffians", "a second band of ruffians", "cave troll"}, names)

	g, ok := FindGroup(room, "ruffians")
	require.True(t, ok)
	assert.Equal(t, "a band of ruffians", g.Name)
	g, ok = FindGroup(room, "ruffians#2")
	require.True(t, ok)
	assert.Equal(t, "a second band of ruffians", g.Name)
	_, ok = FindGroup(room, "band")
	assert.True(t, ok, "by its noun")
	_, ok = FindGroup(room, "ruffian")
	assert.False(t, ok, "never by one member's name")
	g, ok = FindGroup(room, "troll")
	require.True(t, ok, "a lone mob by its own name")
	assert.True(t, g.Solo())

	g, ok = GroupOf(room, 9104)
	require.True(t, ok)
	assert.Equal(t, "a second band of ruffians", g.Name)
}
