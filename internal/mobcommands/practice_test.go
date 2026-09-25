package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var practiceBeaten []PracticeBeaten

func init() {
	OnPracticeBeaten.Register(func(b PracticeBeaten) PracticeBeaten {
		practiceBeaten = append(practiceBeaten, b)
		return b
	})
}

// practiceFight kills a mob that user 7 has hit, and reports what came of
// it: the user's XP, the room's items and gold, MobDeath events, and the
// practice hook's calls.
func practiceFight(t *testing.T, practice bool) (xp int, room *rooms.Room, deaths int, beaten []PracticeBeaten) {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 987611, Name: "straw"})
	t.Cleanup(func() { items.RemoveTestItemSpec(987611) })
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 700)
	user.Character.Level = 1
	users.SetTestUser(user)

	room = rooms.NewEmptyRoom()
	mob := &mobs.Mob{MobId: 999, Practice: practice, ItemDropChance: 100}
	mob.Character.Name = "straw footman"
	mob.Character.Level = 5
	mob.Character.TNLScale = 1
	mob.Character.Items = []items.Item{items.New(987611)}
	mob.Character.Gold = 9
	mob.Character.PlayerDamage = map[int]int{7: 10}
	mob.InstanceId = 424242
	room.AddMob(mob.InstanceId)

	lid := events.RegisterListener(events.MobDeath{}, func(events.Event) events.ListenerReturn {
		deaths++
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.MobDeath{}, lid) })
	events.ProcessEvents()
	practiceBeaten = nil

	before := user.Character.Experience
	handled, err := Suicide("", mob, room)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	return user.Character.Experience - before, room, deaths, practiceBeaten
}

// Phase 27c: a practice foe is beaten, not killed: no XP, drops, gold, or
// MobDeath, and the hook reports it.
func TestPracticeMobGivesNothing(t *testing.T) {
	xp, room, deaths, beaten := practiceFight(t, true)
	assert.Zero(t, xp)
	assert.Empty(t, room.Items)
	assert.Zero(t, room.Gold)
	assert.Empty(t, room.Corpses)
	assert.Zero(t, deaths, "no MobDeath")
	require.Len(t, beaten, 1)
	assert.Equal(t, 424242, beaten[0].InstanceId)
	assert.Equal(t, room.RoomId, beaten[0].RoomId)
	assert.NotContains(t, room.GetMobs(), 424242, "it leaves the room")
	user := users.GetByUserId(7)
	assert.Zero(t, user.Character.KD.GetMobKills(999), "no kill counted")
	assert.Zero(t, user.Character.Alignment)
}

func TestOrdinaryMobStillRewards(t *testing.T) {
	xp, room, deaths, beaten := practiceFight(t, false)
	assert.Positive(t, xp)
	assert.Positive(t, room.Gold+len(room.Items)+len(room.Corpses), "it drops its things")
	assert.Equal(t, 1, deaths)
	assert.Empty(t, beaten)
}
