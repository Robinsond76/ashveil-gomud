package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heard collects the text sent to users while fn runs.
func heard(t *testing.T, fn func()) string {
	t.Helper()
	events.ProcessEvents()
	var got []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		got = append(got, e.(events.Message).Text)
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, id)
	fn()
	events.ProcessEvents()
	return strings.Join(got, "\n")
}

// Phase 32d: in a battle only flee (since 33c, retreat) takes a player out, and nothing else
// typed changes it: break, eat, drink, use, equip, remove, and walking out
// are refused and change nothing.
func TestBattleRefusesWhatWouldChangeIt(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases() // exit names
	user := userWithItem(t, 18, drinkableSpec("healing potion", 10, 2))
	room := testRoom()
	room.Exits = map[string]exit.RoomExit{"north": {RoomId: 2}}
	battle.Reset()
	t.Cleanup(battle.Reset)
	user.Character.RoomId = room.RoomId
	battle.Begin(18, room.RoomId, 1, "party", []int{501})
	user.Character.SetAggro(0, 501, characters.DefaultAttack)
	engagement.Resume(18)

	cmds := []struct {
		name string
		run  func() (bool, error)
		want string
	}{
		{"break", func() (bool, error) { return Break("", user, room, 0) }, BattleOnlyFlee},
		{"go", func() (bool, error) { return Go("north", user, room, 0) }, BattleOnlyFlee},
		// 32d review: a word that isn't an exit isn't told about flee.
		{"not an exit", func() (bool, error) { return Go("attak", user, room, 0) }, "You can't do that! You are in combat!"},
		{"drink", func() (bool, error) { return Drink("potion", user, room, 0) }, BattleUnderWay},
		{"eat", func() (bool, error) { return Eat("potion", user, room, 0) }, BattleUnderWay},
		{"use", func() (bool, error) { return Use("potion", user, room, 0) }, BattleUnderWay},
		{"equip", func() (bool, error) { return Equip("potion", user, room, 0) }, BattleUnderWay},
		{"remove", func() (bool, error) { return Remove("all", user, room, 0) }, BattleUnderWay},
	}
	for _, c := range cmds {
		var handled bool
		out := heard(t, func() {
			var err error
			handled, err = c.run()
			require.NoError(t, err, c.name)
		})
		assert.True(t, handled, c.name)
		assert.Equal(t, c.want, strings.TrimSpace(out), c.name)
		require.NotNil(t, user.Character.Aggro, "%s: still fighting", c.name)
		assert.Equal(t, 501, user.Character.Aggro.MobInstanceId, c.name)
	}
	assert.False(t, engagement.StoodDown(18), "break stood nobody down")
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 2, user.Character.Items[0].Uses, "the potion is untouched")

	// Flee still works: it orders the retreat (Phase 33c).
	_, err := Flee("", user, room, 0)
	require.NoError(t, err)
	assert.Equal(t, characters.Retreat, user.Character.Aggro.Type)
}

func TestOutOfBattleTheCommandsWork(t *testing.T) {
	user := userWithItem(t, 19, drinkableSpec("waterskin", 10, 2))
	room := testRoom()
	battle.Reset()
	t.Cleanup(battle.Reset)
	assert.False(t, InBattle(user))

	// 32d review: a shot into the next room is not a battle.
	user.Character.Aggro = &characters.Aggro{Type: characters.Shooting, MobInstanceId: 777, ExitName: "north"}
	assert.False(t, InBattle(user))
	user.Character.Aggro = nil

	// A fight with another player is not a battle: break still works.
	other := users.NewUserRecord(20, 1)
	users.SetTestUser(other)
	user.Character.SetAggro(20, 0, characters.DefaultAttack)
	assert.False(t, InBattle(user))
	out := heard(t, func() { _, _ = Break("", user, room, 0) })
	assert.Contains(t, out, "You break off combat.")
	engagement.Resume(19)

	useFakeProvisioner(t, &fakeProvisioner{result: survival.ProvisionResult{Member: survival.LeaderMemberKey, Name: "Tester"}})
	out = heard(t, func() { _, _ = Drink("waterskin", user, room, 0) })
	assert.NotContains(t, out, "battle")
	assert.Equal(t, 1, user.Character.Items[0].Uses, "drank")
}
