package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// campBeforeSleep makes camp at the Fork with a lit fire and one armed
// companion, stopping before the rest begins.
func campBeforeSleep(t *testing.T, stones int) (*CampingModule, *users.UserRecord, *rooms.Room, *characters.Character) {
	t.Helper()
	loadShippedWorld(t)
	races.LoadDataFiles()
	restedSpecs(t)
	sharpenSpecs(t)
	fork := rooms.LoadRoom(2002)
	require.NotNil(t, fork)
	bran := testMob(t, 97231, "Bran", 2002)
	bran.Character.RaceId = 1
	armed(&bran.Character, testSwordID, 0)
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bran"},
	}})
	company.SetFormationProvider(formationMap{1: 97231})
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
	})
	now := baseTime()
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 2002)
	user.Character.Name = "Hero"
	user.Character.RaceId = 1
	armed(user.Character, testSwordID, testDaggerID)
	if stones > 0 {
		user.Character.StoreItem(stone(stones))
	}
	captureMessages(t)
	for _, cmd := range []string{"", "fire"} {
		_, err := module.userCommand(cmd, user, fork, 0)
		require.NoError(t, err)
	}
	return module, user, fork, &bran.Character
}

func activityRowOf(t *testing.T, m *CampingModule, key string) camping.ActivityRow {
	t.Helper()
	s, ok := m.CampStateOf(7, 2002, nil)
	require.True(t, ok)
	for _, a := range s.Activities {
		if a.Key == key {
			return a
		}
	}
	t.Fatalf("no %s activity in %+v", key, s.Activities)
	return camping.ActivityRow{}
}

// Before the company sleeps the Camp tab offers the whetstone, then says
// plainly that nothing is dull once it has been used.
func TestCampActivitiesOfferTheWhetstoneBeforeSleep(t *testing.T) {
	m, user, fork, bran := campBeforeSleep(t, 10)

	row := activityRowOf(t, m, "sharpen")
	assert.True(t, row.Ready)
	assert.Equal(t, "camp sharpen", row.Command)
	assert.Contains(t, row.Note, "Hero")
	assert.Contains(t, row.Note, "Bran")
	assert.Contains(t, row.Note, "whetstone use")

	// The button's command is the player's command.
	text := m.sharpenArgs(user, []string{})
	assert.Contains(t, text, "Sharpened")
	assert.True(t, user.Character.Equipment.Weapon.Sharpened())
	assert.True(t, bran.Equipment.Weapon.Sharpened())

	row = activityRowOf(t, m, "sharpen")
	assert.False(t, row.Ready)
	assert.Equal(t, "No blades are dull enough to need the whetstone.", row.Note)
	_ = fork
}

func TestCampActivitiesSayWhenThereIsNoWhetstone(t *testing.T) {
	m, _, _, _ := campBeforeSleep(t, 0)
	row := activityRowOf(t, m, "sharpen")
	assert.False(t, row.Ready)
	assert.Contains(t, row.Note, "no whetstone")
}

func TestCampActivitiesListPoisonAndCook(t *testing.T) {
	m, _, _, _ := campBeforeSleep(t, 1)
	poison := activityRowOf(t, m, "poison")
	assert.False(t, poison.Ready)
	assert.Contains(t, poison.Note, "No poison is assigned")
	cook := activityRowOf(t, m, "cook")
	assert.Equal(t, "camp cook", cook.Command)
	assert.NotEmpty(t, cook.Note)
}

// Once the company sleeps the activities close: no rows, and the commands
// refuse.
func TestCampActivitiesCloseWhenTheCompanySleeps(t *testing.T) {
	m, user, fork, _ := campBeforeSleep(t, 10)
	_, err := m.userCommand("rest", user, fork, 0)
	require.NoError(t, err)

	s, ok := m.CampStateOf(7, 2002, nil)
	require.True(t, ok)
	assert.Empty(t, s.Activities)

	assert.Contains(t, m.sharpenArgs(user, nil), "while the company rests")
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
	assert.Contains(t, strings.ToLower(m.cook(user, fork, nil)), "while the company rests")
}
