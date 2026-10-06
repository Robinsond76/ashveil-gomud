package camping

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCampStateOf (Phase 32g): the Camp tab's state from the room the
// leader stands in: none (and whether camp can be made here), here with
// the fire and a running rest, elsewhere, and an inn.
func TestCampStateOf(t *testing.T) {
	noWorld(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	m := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return now })
	campUser(t, 7, 100)

	s, ok := m.CampStateOf(7, 100, []string{"camping"})
	require.True(t, ok)
	assert.Equal(t, camping.CampState{CanCamp: true, Recipes: s.Recipes}, s)
	assert.Len(t, s.Recipes, 1, "the common chill remedy is in every book")
	s, _ = m.CampStateOf(7, 100, nil)
	assert.False(t, s.CanCamp, "nowhere to camp here")

	m.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, FireLit: true,
		Rest: &camping.RestSession{StartedAtUTC: now.Add(-15 * time.Second), State: camping.Resting}}
	s, _ = m.CampStateOf(7, 100, []string{"camping"})
	assert.True(t, s.HasCamp)
	assert.True(t, s.Here)
	assert.True(t, s.FireLit)
	assert.True(t, s.Resting)
	assert.Equal(t, 25, s.RestPercent)
	assert.Equal(t, 45, s.RestSeconds)
	assert.False(t, s.CanCamp, "one camp at a time")

	s, _ = m.CampStateOf(7, 200, []string{"camping"})
	assert.True(t, s.HasCamp)
	assert.False(t, s.Here)
	assert.Equal(t, 100, s.RoomID, "the camp's room, for the map marker (Phase 40b)")
	assert.Equal(t, "room #100", s.RoomTitle, "no such room loaded in the test")

	s, _ = m.CampStateOf(8, 300, []string{m.innSettings().RoomTag})
	assert.True(t, s.Inn)
}

// TestCampStateRested (32g review finding 7): a camp whose rest is done
// says so, and isn't resting, so the Camp tab offers no Rest.
func TestCampStateRested(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	m := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return now })
	m.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, FireLit: true,
		Rest: &camping.RestSession{StartedAtUTC: now.Add(-2 * time.Minute), State: camping.Completed}}
	s, _ := m.CampStateOf(7, 100, nil)
	assert.True(t, s.Rested)
	assert.False(t, s.Resting)
}

// TestRoomTagMatchingIsCaseInsensitiveEverywhere: making camp, the Camp tab's
// CanCamp flag and the inn all judge a room by the same rule, so a room
// tagged "Camping" or "INN" works for every path, not just the inn's.
func TestRoomTagMatchingIsCaseInsensitiveEverywhere(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	m := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 100)

	assert.True(t, roomEligible(&rooms.Room{RoomId: 100, Tags: []string{"Camping"}}, m.roomTag()))
	assert.False(t, roomEligible(nil, m.roomTag()))

	s, _ := m.CampStateOf(7, 100, []string{"CAMPING"})
	assert.True(t, s.CanCamp)
	s, _ = m.CampStateOf(7, 100, []string{"Inn"})
	assert.True(t, s.Inn)

	msg := m.establish(user, &rooms.Room{RoomId: 100, Title: "A Clearing", Tags: []string{"Camping"}})
	assert.NotEqual(t, "There is nowhere here to make camp.", msg)
	_, ok := m.camps[7]
	assert.True(t, ok, "camp made in a room whose tag differs only by case")
}
