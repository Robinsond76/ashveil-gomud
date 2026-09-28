package enemyparty

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobparty"

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

// TestAnUnspawnedGroupKeepsItsNameAsMembersFall (32c review): a group that
// formed outside a spawn list (tagged mobs a script or admin placed) is
// named when first seen and keeps the name when its most common kind
// falls, so the word a player typed still names it.
func TestAnUnspawnedGroupKeepsItsNameAsMembersFall(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	captain := testMob(t, 9201, 40, "bandits")
	captain.Character.Name = "bandit captain"
	for _, id := range []int{9202, 9203} {
		m := testMob(t, id, 10, "bandits")
		m.Character.Name = "bandit cutthroat"
	}
	room := testRoom(t, 990201, 9201, 9202, 9203)
	gs := Groups(room)
	require.Len(t, gs, 1)
	assert.Equal(t, "a band of bandit cutthroats", gs[0].Name)

	// A cutthroat falls: a fresh name would now tie and go to the
	// toughest, "a band of bandit captains".
	room.SetTestOccupants(nil, []int{9201, 9202})
	assert.Equal(t, "a band of bandit captains", mobparty.Generate([]mobparty.MobSummary{
		{Name: "bandit captain", EHP: 40}, {Name: "bandit cutthroat", EHP: 10},
	}).Name)
	gs = Groups(room)
	require.Len(t, gs, 1)
	assert.Equal(t, "a band of bandit cutthroats", gs[0].Name)
	_, ok := FindGroup(room, "cutthroats")
	assert.True(t, ok, "the typed name still finds it")
}

// TestTheSuggestedKeywordNamesItsOwnGroup (32c review M3): "a band of big
// rats" answers to "rats" too, so the plain rats' keyword is numbered, and
// each group's suggested word finds that group.
func TestTheSuggestedKeywordNamesItsOwnGroup(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	namedMob(t, 9301, "big rat", "a")
	namedMob(t, 9302, "big rat", "a")
	namedMob(t, 9303, "rat", "b")
	namedMob(t, 9304, "rat", "b")
	room := testRoom(t, 990301, 9301, 9302, 9303, 9304)
	for _, g := range Groups(room) {
		kw := Keyword(room, g)
		found, ok := FindGroup(room, kw)
		require.True(t, ok, kw)
		assert.Equal(t, g.Party.ID, found.Party.ID, "%q names %q", kw, g.Name)
	}
}

// TestAWanderingMemberGoesByItsOwnName (32c review m1): a tag-formed
// group's member that wanders off alone is a lone mob again.
func TestAWanderingMemberGoesByItsOwnName(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	for _, id := range []int{9311, 9312} {
		m := testMob(t, id, 10, "law")
		m.Character.Name = "guard"
	}
	room := testRoom(t, 990311, 9311, 9312)
	require.Equal(t, "a band of guards", Groups(room)[0].Name)
	room.SetTestOccupants(nil, []int{9311})
	away := testRoom(t, 990312, 9312)
	g, ok := FindGroup(away, "guard")
	require.True(t, ok, "attack guard")
	assert.True(t, g.Solo())
	assert.Equal(t, "guard", g.Name)
}
