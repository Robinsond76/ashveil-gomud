package cleanup

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/templates"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 33d review: anyone (or any mob) could bury a corpse holding another
// company's claimed battle loot, destroying it.
func TestBuryLeavesAnotherCompanysClaimedLoot(t *testing.T) {
	c := &CleanupModule{}
	corpse := func(room *rooms.Room) {
		k := rooms.Corpse{ClaimUserId: 94951, MobId: 94950, Character: *characters.New(), Gold: 9}
		k.Character.Name = "bandit"
		room.AddCorpse(k)
	}
	room := rooms.NewEmptyRoom()
	corpse(room)
	stranger := users.NewUserRecord(94952, 0)
	stranger.Character.RoomId = room.RoomId
	_, err := c.userBuryCommand("bandit corpse", stranger, room, 0)
	require.NoError(t, err)
	assert.Len(t, room.Corpses, 1, "a stranger cannot bury it")

	scavenger := &mobs.Mob{InstanceId: 94953}
	scavenger.Character = *characters.New()
	_, err = c.mobBuryCommand("bandit corpse", scavenger, room)
	require.NoError(t, err)
	assert.Len(t, room.Corpses, 1, "nor can a mob")

	claimant := users.NewUserRecord(94951, 0)
	claimant.Character.RoomId = room.RoomId
	_, err = c.userBuryCommand("bandit corpse", claimant, room, 0)
	require.NoError(t, err)
	assert.Empty(t, room.Corpses, "the claimant may")

	corpse(room)
	room.Corpses[0].Gold = 0 // emptied: anyone may bury it
	_, err = c.userBuryCommand("bandit corpse", stranger, room, 0)
	require.NoError(t, err)
	assert.Empty(t, room.Corpses)

	// Among same-named corpses, bury reaches one it may bury (independent
	// review: it only ever tried the first).
	corpse(room)
	empty := rooms.Corpse{MobId: 94950, Character: *characters.New()}
	empty.Character.Name = "bandit"
	room.AddCorpse(empty)
	_, err = c.userBuryCommand("bandit corpse", stranger, room, 0)
	require.NoError(t, err)
	require.Len(t, room.Corpses, 1)
	assert.Equal(t, 94951, room.Corpses[0].ClaimUserId, "the claimed loot stays")
	_, err = c.mobBuryCommand("bandit corpse", scavenger, room)
	require.NoError(t, err)
	assert.Len(t, room.Corpses, 1)
}

func TestBuryHelpExplainsClaimedLoot(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	world := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	flat := configs.Flatten(configs.GetOverrides())
	previous := flat["FilePaths.DataFiles"]
	flat["FilePaths.DataFiles"] = world
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() {
		flat := configs.Flatten(configs.GetOverrides())
		flat["FilePaths.DataFiles"] = previous
		require.NoError(t, configs.RestoreOverrides(flat))
	})
	keywords.LoadAliases()
	templates.RegisterFS(plugins.GetPluginRegistry())
	page, err := usercommands.GetHelpContents("bury")
	require.NoError(t, err)
	assert.Contains(t, page, "battle loot claimed by another player")
	assert.Contains(t, page, "help party")
}
