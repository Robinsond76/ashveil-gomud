package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
)

// 33d review: with no consent between them, a stranger's single blow could
// win a shared kill's whole loot claim through the rotating pick. Outside one
// alliance the claim goes to the most damage, and players see names.
func TestLootClaimOutsideOneAllianceGoesToMostDamage(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	battle.Reset()
	t.Cleanup(battle.Reset)
	t.Cleanup(parties.UseMemoryForTest())
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = true
	gameplay.Death.CorpseItems = false
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	for name, allied := range map[string]bool{"strangers": false, "one alliance": true} {
		t.Run(name, func(t *testing.T) {
			room := rooms.NewEmptyRoom()
			// Instance 94931 rotates the claim to the second (lower-damage) id.
			foe := &mobs.Mob{MobId: 94930, InstanceId: 94931}
			foe.Character = *characters.New()
			foe.Character.Name = "contested foe"
			foe.Character.RoomId = room.RoomId
			foe.Character.Level = 2
			foe.Character.Gold = 5
			foe.Character.PlayerDamage = map[int]int{}
			ids := []int{94940, 94941}
			for i, uid := range ids {
				u := users.NewUserRecord(uid, 0)
				u.Character.Name = []string{"Aria", "Snipe"}[i]
				u.Character.RoomId = room.RoomId
				u.Character.Health = 10
				users.SetTestUser(u)
				battle.Begin(uid, room.RoomId, 1, "contested", []int{foe.InstanceId})
			}
			foe.Character.PlayerDamage[ids[0]] = 40
			foe.Character.PlayerDamage[ids[1]] = 1
			if allied {
				p := parties.New(ids[0])
				p.InvitePlayer(ids[1])
				require.True(t, p.AcceptInvite(ids[1]))
				t.Cleanup(p.Disband)
			}
			var said []string
			id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
				said = append(said, e.(events.Message).Text)
				return events.Cancel
			})
			t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
			_, err := Suicide("", foe, room)
			require.NoError(t, err)
			events.ProcessEvents()
			require.Len(t, room.Corpses, 1)
			want, who := ids[0], "Aria"
			if allied {
				want, who = ids[1], "Snipe" // allies rotate the claim
			}
			assert.Equal(t, want, room.Corpses[0].ClaimUserId)
			out := strings.Join(said, "\n")
			assert.Contains(t, out, "claimed by")
			assert.Contains(t, out, who)
			assert.NotContains(t, out, "#9494")
		})
	}
}
