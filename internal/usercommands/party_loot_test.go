package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClaimedBattleLootUsesSameGateForAllAndGold(t *testing.T) {
	for _, command := range []string{"gold corpse", "all corpse", "blade corpse"} {
		t.Run(command, func(t *testing.T) {
			gameplay := configs.GetGamePlayConfig()
			gameplay.Death.CorpseItems = false
			t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
			room := rooms.NewEmptyRoom()
			owner := users.NewUserRecord(94801, 0)
			other := users.NewUserRecord(94802, 0)
			owner.Character.RoomId = room.RoomId
			other.Character.RoomId = room.RoomId
			owner.Character.Gold = 0
			other.Character.Gold = 0
			items.SetTestItemSpec(&items.ItemSpec{ItemId: 94811, Name: "blade", Weight: 1})
			t.Cleanup(func() { items.RemoveTestItemSpec(94811) })
			item := items.New(94811)
			corpse := rooms.Corpse{ClaimUserId: owner.UserId, MobId: 94812, Character: *characters.New(), Items: []items.Item{item}, Gold: 19}
			corpse.Character.Name = "bandit"
			room.AddCorpse(corpse)
			_, err := Get(command, other, room, 0)
			require.NoError(t, err)
			assert.Zero(t, other.Character.Gold)
			assert.Empty(t, other.Character.Items)
			assert.Equal(t, 19, room.Corpses[0].Gold)
			require.Len(t, room.Corpses[0].Items, 1)
			_, err = Get(command, owner, room, 0)
			require.NoError(t, err)
			if command != "blade corpse" {
				assert.Equal(t, 19, owner.Character.Gold)
				assert.Zero(t, room.Corpses[0].Gold)
			}
			if command != "gold corpse" {
				require.Len(t, owner.Character.Items, 1)
				assert.True(t, item.Equals(owner.Character.Items[0]))
				assert.Empty(t, room.Corpses[0].Items)
			}
			_, err = Get(command, owner, room, 0)
			require.NoError(t, err)
			if command != "blade corpse" {
				assert.Equal(t, 19, owner.Character.Gold)
			}
			if command != "gold corpse" {
				assert.Len(t, owner.Character.Items, 1)
			}
		})
	}
}
