package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArchetypes answers only PlayerArchetype; the embedded nil interface
// panics on anything else, so the test shows what the payload calls.
type fakeArchetypes struct {
	archetypes.Provider
	chosen map[int]string
}

func (f fakeArchetypes) PlayerArchetype(userID int) (string, bool) {
	a, ok := f.chosen[userID]
	return a, ok
}

type fakeClasses struct{ class map[int]string }

func (f fakeClasses) PlayerClass(userID int) classes.State { return classes.State{Class: f.class[userID]} }

// TestMapSpriteKeys (Phase 40b): Char.Info and Party vitals carry the
// lineage and class ids the map sprite is chosen by.
func TestMapSpriteKeys(t *testing.T) {
	archetypes.SetProvider(fakeArchetypes{chosen: map[int]string{94951: "warrior", 94952: "cleric"}})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	classes.SetProvider(fakeClasses{class: map[int]string{94952: "priest"}})
	t.Cleanup(func() { classes.SetProvider(nil) })

	l, c := classKeys(94951)
	assert.Equal(t, "warrior", l)
	assert.Equal(t, "warrior", c, "an unpromoted character's class is its archetype")
	l, c = classKeys(94952)
	assert.Equal(t, "cleric", l)
	assert.Equal(t, "priest", c)
	l, c = classKeys(94999)
	assert.Empty(t, l+c, "no archetype chosen yet")

	// Char.Info
	u := users.NewUserRecord(94951, 0)
	u.Character.Name = "Hero"
	g := &GMCPCharModule{}
	info, _ := g.GetCharNode(u, "Char.Info")
	raw, err := json.Marshal(info)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "warrior", got["lineage"])
	assert.Equal(t, "warrior", got["classid"])

	// Party.Vitals
	t.Cleanup(parties.UseMemoryForTest())
	p := parties.New(94952)
	m := users.NewUserRecord(94952, 0)
	m.Character.Name = "Healer"
	m.Character.HealthMax.Value = 10
	m.Character.Health = 10
	users.SetTestUser(m)
	t.Cleanup(func() { users.RemoveTestUser(m.UserId) })
	pg := &GMCPPartyModule{}
	vit, _ := pg.GetPartyNode(p, "Party.Vitals")
	v := vit.(map[string]GMCPPartyModule_Payload_Vitals)["Healer"]
	assert.Equal(t, "cleric", v.Lineage)
	assert.Equal(t, "priest", v.ClassID)
}

// TestCompanyCampAlliedOnly (Phase 40b, the owner's rule): the camp payload
// carries its own room ID and the camps of the viewer's party, and never the
// camp of a company outside the party, even one in the same room.
func TestCompanyCampAlliedOnly(t *testing.T) {
	t.Cleanup(parties.UseMemoryForTest())
	p := parties.New(94961)
	p.InvitePlayer(94962)
	require.True(t, p.AcceptInvite(94962))
	me := users.NewUserRecord(94961, 0)
	ally := users.NewUserRecord(94962, 0)
	ally.Character.Name = "Ally"
	stranger := users.NewUserRecord(94963, 0)
	stranger.Character.Name = "Stranger"
	for _, u := range []*users.UserRecord{me, ally, stranger} {
		users.SetTestUser(u)
		uid := u.UserId
		t.Cleanup(func() { users.RemoveTestUser(uid) })
	}

	camps := map[int]camping.CampState{
		94961: {HasCamp: true, Here: true, RoomID: 11, FireLit: true},
		94962: {HasCamp: true, RoomID: 22, Resting: true},
		94963: {HasCamp: true, RoomID: 11}, // same room as mine, not in my party
	}
	state := func(uid, _ int, _ []string) (camping.CampState, bool) { return camps[uid], true }
	extra := campExtra(state, func(u *users.UserRecord) []alliedCamp { return alliedCampsOf(u, state) })

	var got struct {
		RoomID      int          `json:"room_id"`
		AlliedCamps []alliedCamp `json:"allied_camps"`
	}
	require.NoError(t, json.Unmarshal(extra.build(me), &got))
	assert.Equal(t, 11, got.RoomID)
	require.Len(t, got.AlliedCamps, 1)
	assert.Equal(t, alliedCamp{RoomID: 22, Leader: "Ally", Resting: true}, got.AlliedCamps[0])

	// A player with no party sees no allied camps, as an empty list.
	require.NoError(t, json.Unmarshal(extra.build(stranger), &got))
	assert.Equal(t, 11, got.RoomID)
	assert.Empty(t, got.AlliedCamps)
}
