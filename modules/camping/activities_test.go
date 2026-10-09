package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
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
	// Review: the leader is "you", and names join with "and".
	assert.Regexp(t, `blades for (yourself|you) and Bran,`, row.Note)
	assert.NotContains(t, row.Note, "Hero")
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
	assert.Equal(t, "You and Bran have dull blades, but you have no whetstone.", row.Note)
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

// Review: the Poison row gives the reason, not the plan's first blade row
// with its markup: "already carries its poison" once every blade is coated,
// and the blocker when vials run short.
func TestCampPoisonActivityGivesTheReason(t *testing.T) {
	vials := stock{bitterleafVial: 1}
	module, user, _, _ := campedFixture(t, vials)
	campPoison(t, module, user, "assign self main bitterleaf")
	campPoison(t, module, user, "assign bran main bitterleaf")

	row := module.poisonActivity(user)
	assert.False(t, row.Ready)
	assert.Equal(t, "Short of Bitterleaf: need 2, have 1.", row.Note)

	campPoison(t, module, user, "unassign bran main")
	row = module.poisonActivity(user)
	assert.True(t, row.Ready)
	assert.Contains(t, row.Note, "You coat 1 blade from your vials")

	assert.Contains(t, campPoison(t, module, user, "apply"), "You coat 1 blade.")
	row = module.poisonActivity(user)
	assert.False(t, row.Ready)
	assert.Equal(t, "Every assigned blade already carries its poison; no application is needed.", row.Note)
	assert.NotContains(t, row.Note, "<ansi")
}

// Review: broth and watch incense, set by before a rest, are chores too;
// each row shows only when the company carries the supply.
func TestCampActivitiesListRestSuppliesCarried(t *testing.T) {
	keys := func(rows []camping.ActivityRow) []string {
		var out []string
		for _, r := range rows {
			out = append(out, r.Key)
		}
		return out
	}
	supplies := stock{}
	w, _, _ := prepWorld(t, supplies)
	assert.Equal(t, []string{"sharpen", "poison", "cook"}, keys(w.m.campActivities(w.user, w.camp())), "no supplies carried, no rows for them")

	supplies[camping.BrothItemID] = 1
	supplies[camping.IncenseItemID] = 1
	rows := w.m.campActivities(w.user, w.camp())
	require.Equal(t, []string{"sharpen", "poison", "cook", "broth", "incense"}, keys(rows))
	assert.True(t, rows[3].Ready)
	assert.Equal(t, "camp prepare broth all", rows[3].Command)
	assert.Contains(t, rows[3].Note, "Sets fortifying broth by for you:")
	assert.False(t, rows[4].Ready)
	assert.Contains(t, rows[4].Note, "Needs a Camp Watch")

	// The buttons run the player's commands; then the rows say it is done.
	assert.Contains(t, prepare(w, "broth", "all"), "Fortifying broth is set by")
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: {Name: "Bran", Level: 1}})
	assert.True(t, w.m.campActivities(w.user, w.camp())[4].Ready)
	assert.Contains(t, prepare(w, "incense"), "watchsage")
	rows = w.m.campActivities(w.user, w.camp())
	assert.False(t, rows[3].Ready)
	assert.Equal(t, "Everyone has broth set by or a draught working.", rows[3].Note)
	assert.False(t, rows[4].Ready)
	assert.Equal(t, "Watch incense is already set for the next rest.", rows[4].Note)
}

// Review: the no-whetstone line names the leader "You", alone or first,
// and one companion takes "has".
func TestCampActivitiesNoWhetstoneGrammar(t *testing.T) {
	m, user, _, bran := campBeforeSleep(t, 0)
	bran.Equipment.Weapon = items.Item{}
	assert.Equal(t, "You have dull blades, but you have no whetstone.", activityRowOf(t, m, "sharpen").Note)

	armed(bran, testSwordID, 0)
	user.Character.Equipment.Weapon = items.Item{}
	user.Character.Equipment.Offhand = items.Item{}
	assert.Equal(t, "Bran has dull blades, but you have no whetstone.", activityRowOf(t, m, "sharpen").Note)
}
