package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSharedEnemyRewardsAndLootSettleOnce(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	battle.Reset()
	t.Cleanup(battle.Reset)
	t.Cleanup(parties.UseMemoryForTest())
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = false
	gameplay.Death.CorpseItems = false
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	room := rooms.NewEmptyRoom()
	foe := &mobs.Mob{MobId: 98721, InstanceId: 987211, ItemDropChance: 100}
	foe.Character = *characters.New()
	foe.Character.Name = "shared foe"
	foe.Character.RoomId = room.RoomId
	foe.Character.Level = 5
	foe.Character.TNLScale = 1
	foe.Character.Gold = 13
	foe.Character.PlayerDamage = map[int]int{}
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 98722, Name: "shared blade", Type: items.Weapon})
	t.Cleanup(func() { items.RemoveTestItemSpec(98722) })
	item := items.New(98722)
	foe.Character.Equipment.Weapon = item
	var players []*users.UserRecord
	for i := 0; i < 7; i++ {
		uid := 94710 + i
		u := users.NewUserRecord(uid, 0)
		u.Character.RoomId = room.RoomId
		u.Character.Level = 1
		u.Character.Health = 10
		u.Character.Experience = 0
		users.SetTestUser(u)
		players = append(players, u)
		foe.Character.PlayerDamage[uid] = 10
		battle.Begin(uid, room.RoomId, 1, "shared", []int{foe.InstanceId})
	}
	players[2].Character.RoomId++                     // distant
	players[3].Character.Health = 0                   // fallen
	players[4].Character.CombatWithdrawn = true       // ordered retreat/rout
	foe.Character.PlayerDamage[players[5].UserId] = 0 // no positive contribution
	battle.Begin(players[6].UserId, room.RoomId, 1, "separate", []int{999999})
	p := parties.New(players[0].UserId)
	p.InvitePlayer(players[1].UserId)
	p.AcceptInvite(players[1].UserId)
	p.InvitePlayer(players[5].UserId) // mere invite gives neither rewards nor loot
	expected := eligibleContributors(foe, room.RoomId, false)
	assert.Equal(t, []int{players[0].UserId, players[1].UserId}, expected)

	deaths := 0
	id := events.RegisterListener(events.MobDeath{}, func(events.Event) events.ListenerReturn { deaths++; return events.Continue })
	t.Cleanup(func() { events.UnregisterListener(events.MobDeath{}, id) })
	_, err := Suicide("", foe, room)
	require.NoError(t, err)
	events.ProcessEvents()
	for i, u := range players {
		if i < 2 {
			assert.Positive(t, u.Character.Experience)
		} else {
			assert.Zero(t, u.Character.Experience)
		}
	}
	require.Len(t, room.Corpses, 1, "shared loot is claimed even in a floor-drop world")
	corpse := room.Corpses[0]
	assert.Equal(t, expected[foe.InstanceId%len(expected)], corpse.ClaimUserId)
	assert.Equal(t, 13, corpse.Gold)
	require.Len(t, corpse.Items, 1)
	assert.True(t, item.Equals(corpse.Items[0]))
	assert.Empty(t, corpse.Character.GetAllWornItems(), "rolled worn drop is not duplicated")
	assert.Empty(t, room.Items)
	assert.Zero(t, room.Gold)
	before := players[0].Character.Experience
	_, err = Suicide("", foe, room)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Equal(t, before, players[0].Character.Experience)
	assert.Equal(t, 1, deaths)
	assert.Len(t, room.Corpses, 1)
}
