package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walkto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40d: the Walkto namespace.

func testWalktoFeed(views map[int]walkto.View) (*walktoFeed, *[]sent) {
	out := &[]sent{}
	f := newWalktoFeed()
	f.view = func(userID int) (walkto.View, bool) {
		v, ok := views[userID]
		return v, ok
	}
	f.accepting = func(int) bool { return true }
	f.send = func(userID int, module string, payload []byte) {
		var body map[string]any
		_ = json.Unmarshal(payload, &body)
		*out = append(*out, sent{userID, module, body})
	}
	return f, out
}

func TestWalktoPayloadShape(t *testing.T) {
	body, err := json.Marshal(walktoPayloadOf(walkto.View{Target: 2144, Path: []int{2112, 2113}}, true))
	require.NoError(t, err)
	assert.JSONEq(t, `{"target":2144,"path":[2112,2113]}`, string(body))
	body, _ = json.Marshal(walktoPayloadOf(walkto.View{Target: 5}, true))
	assert.JSONEq(t, `{"target":5,"path":[]}`, string(body), "never null")
	assert.Nil(t, walktoPayloadOf(walkto.View{}, false), "not walking (the feed sends {})")
}

func TestWalktoSentOnChangeOnlyAndClearsWithEmptyObject(t *testing.T) {
	views := map[int]walkto.View{}
	f, out := testWalktoFeed(views)

	f.update(7)
	require.Len(t, *out, 1, "the first update sends {}")
	assert.Equal(t, "Walkto", (*out)[0].module)
	assert.Empty(t, (*out)[0].body)

	f.update(7)
	assert.Len(t, *out, 1, "unchanged: nothing more")

	views[7] = walkto.View{Target: 9, Path: []int{4, 9}}
	f.update(7)
	require.Len(t, *out, 2)
	assert.EqualValues(t, 9, (*out)[1].body["target"])

	views[7] = walkto.View{Target: 9, Path: []int{9}}
	f.update(7)
	assert.Len(t, *out, 3, "each step resends the shorter path")

	delete(views, 7)
	f.update(7)
	require.Len(t, *out, 4)
	assert.Empty(t, (*out)[3].body, "the walk ended: {}")
}

func TestWalktoOnlyToThatPlayerAndForgetResends(t *testing.T) {
	views := map[int]walkto.View{7: {Target: 3, Path: []int{3}}}
	f, out := testWalktoFeed(views)
	f.update(7)
	f.update(8)
	require.Len(t, *out, 2)
	assert.Equal(t, 7, (*out)[0].userID)
	assert.EqualValues(t, 3, (*out)[0].body["target"])
	assert.Equal(t, 8, (*out)[1].userID)
	assert.Empty(t, (*out)[1].body, "another player sees none of it")

	f.forget(7)
	f.update(7)
	assert.Len(t, *out, 3, "a login or request resends")
}

func TestWalktoNotSentToAConnectionThatDidNotAcceptGMCP(t *testing.T) {
	f, out := testWalktoFeed(map[int]walkto.View{7: {Target: 3}})
	f.accepting = func(int) bool { return false }
	f.update(7)
	assert.Empty(t, *out)
}

// The change hook reaches the registered feed: starting, stepping and
// clearing a walk through internal/walkto send at once, with no round between.
func TestWalktoChangeHookDrivesTheRegisteredFeed(t *testing.T) {
	walkto.ResetForTest()
	users.ResetActiveUsers()
	t.Cleanup(func() { walkto.ResetForTest(); users.ResetActiveUsers() })
	u := users.NewUserRecord(7, 1)
	u.Username, u.Password = "walker", "$2a$test"
	users.SetTestUser(u)

	var got []sent
	oldSend, oldAcc := walktoFeeds.send, walktoFeeds.accepting
	walktoFeeds.send = func(userID int, module string, payload []byte) {
		var body map[string]any
		_ = json.Unmarshal(payload, &body)
		got = append(got, sent{userID, module, body})
	}
	walktoFeeds.accepting = func(int) bool { return true }
	walktoFeeds.forget(7)
	t.Cleanup(func() { walktoFeeds.send, walktoFeeds.accepting = oldSend, oldAcc; walktoFeeds.forget(7) })

	a := walkto.Begin(7, walkto.Route{Target: 4, Path: []int{1, 2, 4}, Steps: []walkto.Step{{Exit: "east", To: 2}, {Exit: "east", To: 4}}}, [3]bool{})
	require.Len(t, got, 1)
	assert.Equal(t, "Walkto", got[0].module)
	assert.EqualValues(t, 4, got[0].body["target"])
	assert.Equal(t, []any{float64(2), float64(4)}, got[0].body["path"])

	walkto.Advance(7, a.Gen, 1)
	require.Len(t, got, 2)
	assert.Equal(t, []any{float64(4)}, got[1].body["path"])

	walkto.Clear(7)
	require.Len(t, got, 3)
	assert.Empty(t, got[2].body)
}

// Phase 40d: the World.Map biome table says which biomes the sky does not
// reach, so the map's day/night shading leaves them alone.
func TestBiomeTableCarriesIndoorAndDark(t *testing.T) {
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "t-cellar", Name: "Cellar", Symbol: "c", Indoor: true, DarkArea: true})
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "t-meadow", Name: "Meadow", Symbol: "m"})
	t.Cleanup(func() { rooms.RemoveTestBiome("t-cellar"); rooms.RemoveTestBiome("t-meadow") })
	table := buildBiomeTable()
	body, err := json.Marshal(table["t-cellar"])
	require.NoError(t, err)
	assert.Contains(t, string(body), `"indoor":true`)
	assert.Contains(t, string(body), `"dark":true`)
	body, err = json.Marshal(table["t-meadow"])
	require.NoError(t, err)
	assert.NotContains(t, string(body), "indoor", "an outdoor biome sends no flag")
}
