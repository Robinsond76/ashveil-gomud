package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// autoSharpenCamp drives the camp user commands at the real Fork at the
// Black Oak with a live companion mob found through the native roster and
// formation seams, through the rest timer, to the moment before the
// NewRound grant.
func autoSharpenCamp(t *testing.T, auto bool) (*CampingModule, *users.UserRecord, *characters.Character, *[]string) {
	t.Helper()
	loadShippedWorld(t)
	races.LoadDataFiles() // wielding checks race size
	restedSpecs(t)
	sharpenSpecs(t)
	fork := rooms.LoadRoom(2002)
	require.NotNil(t, fork)

	bran := testMob(t, 97231, "Bran", 2002)
	bran.Character.RaceId = 1 // human: race 0 has no weapon slots
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
	scheduler := &fakeScheduler{}
	module := newTestModule(&fakeStore{}, scheduler, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 2002)
	user.Character.Name = "Hero"
	user.Character.RaceId = 1
	armed(user.Character, testSwordID, testDaggerID)
	user.Character.StoreItem(stone(10))
	messages := captureMessages(t)

	cmds := []string{"", "fire"}
	if auto {
		cmds = append(cmds, "sharpen auto on")
	}
	for _, cmd := range append(cmds, "rest") {
		_, err := module.userCommand(cmd, user, fork, 0)
		require.NoError(t, err)
	}
	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	assert.False(t, user.Character.Equipment.Weapon.Sharpened(), "never from the timer")
	return module, user, &bran.Character, messages
}

// TestAutoSharpenAtCampRestCompletion: with auto on, the NewRound grant
// pass sharpens the company once: two uses for two members, both of the
// leader's blades.
func TestAutoSharpenAtCampRestCompletion(t *testing.T) {
	module, user, bran, messages := autoSharpenCamp(t, true)

	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.HasBuffFlag("rested"))
	assert.True(t, user.Character.Equipment.Weapon.Sharpened())
	assert.True(t, user.Character.Equipment.Offhand.Sharpened())
	assert.True(t, bran.Equipment.Weapon.Sharpened())
	assert.Equal(t, []int{8}, stoneUses(user.Character))

	module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, []int{8}, stoneUses(user.Character), "once per rest")
	events.ProcessEvents()
	joined := strings.Join(*messages, "\n")
	assert.Contains(t, joined, "Rested")
	assert.Contains(t, joined, "Sharpened: Hero, Bran.")
}

func TestAutoOffSpendsNothing(t *testing.T) {
	module, user, bran, _ := autoSharpenCamp(t, false)
	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.HasBuffFlag("rested"))
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
	assert.False(t, bran.Equipment.Weapon.Sharpened())
	assert.Equal(t, []int{10}, stoneUses(user.Character))
}

func TestAutoSkippedInCombatSaysSo(t *testing.T) {
	module, user, bran, messages := autoSharpenCamp(t, true)
	bran.SetAggro(0, 55, characters.DefaultAttack)
	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.HasBuffFlag("rested"), "the rest still counts")
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
	assert.Equal(t, []int{10}, stoneUses(user.Character))
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "no blades were sharpened")
}

// TestManualSharpenBeforeRestEndSpendsNoMoreOnAuto: a manual pass earlier
// leaves nothing for the automatic one to spend.
func TestManualSharpenBeforeRestEndSpendsNoMoreOnAuto(t *testing.T) {
	module, user, _, _ := autoSharpenCamp(t, true)
	_, err := module.sharpenCommand("", user, rooms.LoadRoom(2002), 0)
	require.NoError(t, err)
	assert.Equal(t, []int{8}, stoneUses(user.Character))
	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []int{8}, stoneUses(user.Character))
}
