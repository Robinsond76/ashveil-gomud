package company

import (
	"os"
	"path/filepath"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

func TestPluginStoreRejectsMalformedDataWithoutOverwritingIt(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("company_load_regression", "1.0")
	malformed := []byte("companies: [invalid")
	require.NoError(t, plug.WriteBytes("companies", malformed))
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	module.store = pluginStore{plug: plug}
	module.load()
	_, err := module.summon(7, 12, "58")
	assert.Error(t, err)
	module.save()
	data, err := plug.ReadBytes("companies")
	require.NoError(t, err)
	assert.Equal(t, malformed, data)
}

func TestCompanyNormalLogoutLoginRestoresNativeFollowing(t *testing.T) {
	// Exercise the real logout hook and charm cleanup, with all disk writes
	// confined to a disposable two-room world.
	dataDir := t.TempDir()
	previousDataDir := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	t.Cleanup(func() {
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": previousDataDir}))
	})
	fixtures := map[string]string{
		"biomes/default.yaml":                     "biomeid: default\nname: Test\nsymbol: '.'\ndarkarea: true\n",
		"keywords.yaml":                           "direction-aliases: {}\n",
		"rooms/companytest/zone-config.yaml":      "name: companytest\nroomid: 910001\n",
		"rooms/companytest/910001.yaml":           "roomid: 910001\nzone: companytest\ntitle: Departure\ndescription: A quiet departure room.\nexits:\n  north:\n    roomid: 910002\n",
		"rooms/companytest/910002.yaml":           "roomid: 910002\nzone: companytest\ntitle: Arrival\ndescription: A quiet arrival room.\nexits:\n  south:\n    roomid: 910001\n",
		"mobs/companytest/58-training_dummy.yaml": "mobid: 58\nzone: companytest\ncharacter:\n  name: training dummy\n  level: 1\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	origin, destination := rooms.LoadRoom(910001), rooms.LoadRoom(910002)
	require.NotNil(t, origin)
	require.NotNil(t, destination)
	for _, cleanupCharm := range []bool{false, true} {
		name := "before charm cleanup"
		if cleanupCharm {
			name = "after charm cleanup"
		}
		t.Run(name, func(t *testing.T) {
			users.ResetActiveUsers()
			t.Cleanup(users.ResetActiveUsers)
			events.ProcessEvents()
			user := users.NewUserRecord(7, 1)
			user.Username = "companytester"
			user.Character.RoomId = origin.RoomId
			_, _, err := users.LoginUser(user, 1)
			require.NoError(t, err)
			origin.AddPlayer(user.UserId)
			module := newTestModule(*domain.NewRegistry(), nativeRuntime{})
			_, err = module.summon(7, origin.RoomId, "58")
			require.NoError(t, err)
			oldInstance := module.liveByLeader[7]
			t.Cleanup(func() {
				for _, instance := range mobs.GetAllMobInstanceIds() {
					nativeRuntime{}.Detach(7, instance)
				}
				origin.RemovePlayer(7)
				destination.RemovePlayer(7)
			})
			require.Equal(t, events.Continue, hooks.HandleLeave(events.PlayerDespawn{UserId: 7}))
			require.Nil(t, users.GetByUserId(7))
			require.True(t, mobs.MobInstanceExists(oldInstance))
			require.Zero(t, mobs.GetInstance(oldInstance).Character.Charmed.RoundsRemaining)
			if cleanupCharm {
				hooks.MobRoundTick(events.NewRound{})
				require.False(t, mobs.GetInstance(oldInstance).Character.IsCharmed())
			}
			// A new login record, like user-file loading, has no runtime charm IDs.
			user = users.NewUserRecord(7, 2)
			user.Username = "companytester"
			user.Character.RoomId = origin.RoomId
			user.Character.Health = 100
			user.Character.ActionPoints = 100
			_, _, err = users.LoginUser(user, 2)
			require.NoError(t, err)
			origin.AddPlayer(7)
			require.Equal(t, events.Continue, module.onPlayerSpawn(events.PlayerSpawn{UserId: 7}))
			instance := module.liveByLeader[7]
			mob := mobs.GetInstance(instance)
			require.NotNil(t, mob)
			assert.Contains(t, user.Character.GetCharmIds(), instance)
			assert.True(t, mob.Character.IsCharmed(7))
			if assert.NotNil(t, mob.Character.Charmed) {
				assert.NotZero(t, mob.Character.Charmed.RoundsRemaining)
			}
			assert.Len(t, origin.GetMobs(), 1, "relogin must not leave an orphan companion")
			assert.Contains(t, module.registry.Companies, 7)
			events.ProcessEvents()
			followed := false
			listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
				input := e.(events.Input)
				if input.MobInstanceId == instance && input.InputText == "north" {
					followed = true
					handled, err := mobcommands.Go(input.InputText, mob, rooms.LoadRoom(mob.Character.RoomId))
					assert.True(t, handled)
					assert.NoError(t, err)
				}
				return events.Continue
			})
			t.Cleanup(func() { events.UnregisterListener(events.Input{}, listener) })
			turn, round := util.GetTurnCount(), util.GetRoundCount()
			handled, err := usercommands.Go("north", user, origin, 0)
			require.NoError(t, err)
			require.True(t, handled)
			events.ProcessEvents()
			assert.True(t, followed, "ordinary movement must enqueue native companion following")
			assert.Equal(t, destination.RoomId, user.Character.RoomId)
			assert.Equal(t, destination.RoomId, mob.Character.RoomId)
			assert.Equal(t, turn, util.GetTurnCount())
			assert.Equal(t, round, util.GetRoundCount())
		})
	}
}
