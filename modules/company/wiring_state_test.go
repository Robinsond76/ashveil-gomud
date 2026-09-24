package company

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mobItemIDs lists every item id a live mob wears or carries, sorted.
func mobItemIDs(mob *mobs.Mob) []int {
	out := []int{}
	for _, slot := range characters.AllSlots() {
		if itm := mob.Character.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			out = append(out, itm.ItemId)
		}
	}
	for _, itm := range mob.Character.Items {
		out = append(out, itm.ItemId)
	}
	sort.Ints(out)
	return out
}

func stateItemIDs(s *domain.MemberState) []int {
	out := []int{}
	for _, slot := range characters.AllSlots() {
		if itm := s.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			out = append(out, itm.ItemId)
		}
	}
	for _, itm := range s.Items {
		out = append(out, itm.ItemId)
	}
	sort.Ints(out)
	return out
}

// TestCompanionGearSurvivesLogoutRestartAndDeath drives Phase 22b through
// the real entry points: plugins.Load, the real plugin store, real mob
// specs with template gear, `company summon` and `give` through
// usercommands.TryCommand, and PlayerDespawn, PlayerSpawn, and MobDeath
// through events.ProcessEvents.
func TestCompanionGearSurvivesLogoutRestartAndDeath(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	_, thisFile, _, _ := runtime.Caller(0)
	shipped := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	human, err := os.ReadFile(filepath.Join(shipped, "races", "1-human.yaml"))
	require.NoError(t, err)
	fixtures := map[string]string{
		"biomes/default.yaml":                                  "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                                        "direction-aliases: {}\n",
		"races/1-human.yaml":                                   string(human),
		"rooms/geartest/zone-config.yaml":                      "name: geartest\nroomid: 921001\n",
		"rooms/geartest/921001.yaml":                           "roomid: 921001\nzone: geartest\ntitle: Camp\ndescription: A quiet camp.\n",
		"items/weapons-10000/10002-guardsmans_broadsword.yaml": "itemid: 10002\nname: guardsman's broadsword\nnamesimple: broadsword\ntype: weapon\nhands: 1\nsubtype: slashing\ndamage:\n  diceroll: 1d6\n",
		"items/weapons-10000/10004-dagger.yaml":                "itemid: 10004\nname: dagger\nnamesimple: dagger\ntype: weapon\nhands: 1\nsubtype: stabbing\ndamage:\n  diceroll: 1d4\n",
		"items/consumables-30000/30004-cheese_sandwich.yaml":   "itemid: 30004\nname: cheese sandwich\nnamesimple: sandwich\ntype: food\nsubtype: edible\nvalue: 20\n",
		"mobs/geartest/58-training_dummy.yaml": "mobid: 58\nzone: geartest\nitemdropchance: 100\ncharacter:\n  name: training dummy\n  raceid: 1\n  level: 2\n  alignment: 40\n" +
			"  equipment:\n    weapon:\n      itemid: 10002\n  items:\n    - itemid: 30004\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "combat-messages"), 0755))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	camp := rooms.LoadRoom(921001)
	require.NotNil(t, camp)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "gearer"
	user.Password = "$2a$test"
	user.Character.RoomId = camp.RoomId
	user.Character.RaceId = 1
	user.Character.Alignment = 40
	user.Character.Validate()
	user.Character.StoreItem(items.New(10004))
	users.SetTestUser(user)
	camp.AddPlayer(user.UserId)
	destroyAll := func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
	}
	t.Cleanup(func() {
		destroyAll()
		camp.RemovePlayer(7)
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})
	messages := captureCompanyMessages(t)
	run := func(cmd, rest string) string {
		t.Helper()
		*messages = nil
		handled, err := usercommands.TryCommand(cmd, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}
	stored := func() *domain.MemberState {
		t.Helper()
		reg := domain.NewRegistry()
		require.NoError(t, pluginStore{plug: module.plug}.Load(reg))
		record, ok := reg.Get(7)
		require.True(t, ok)
		require.Len(t, record.Companions, 1)
		require.NotNil(t, record.Companions[0].State)
		return record.Companions[0].State
	}
	live := func() *mobs.Mob {
		t.Helper()
		instanceID, ok := module.instance(7, 1)
		require.True(t, ok, "the companion is out")
		mob := mobs.GetInstance(instanceID)
		require.NotNil(t, mob)
		return mob
	}

	// Summon: the template gear is minted once and recorded in the real store.
	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy (#1).")
	assert.Equal(t, []int{10002, 30004}, stateItemIDs(stored()))
	assert.Equal(t, 2, stored().Level)
	assert.Contains(t, run("company", "status"), "level 2")

	// Give the companion a dagger: the gear change is saved at once.
	run("give", "dagger dummy")
	assert.Empty(t, user.Character.Items, "the dagger left the leader")
	assert.Equal(t, []int{10002, 10004, 30004}, mobItemIDs(live()))
	assert.Equal(t, []int{10002, 10004, 30004}, stateItemIDs(stored()))
	gear := run("company", "gear dummy")
	assert.Contains(t, gear, "dagger")
	assert.Contains(t, gear, "broadsword")

	// Leader logs out: recorded, and the mob leaves the world with it.
	instanceID := live().InstanceId
	events.AddToQueue(events.PlayerDespawn{UserId: 7})
	events.ProcessEvents()
	assert.False(t, mobs.MobInstanceExists(instanceID), "no lingering mob to loot")
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked)

	// Restart or copyover: no mob instances survive; the registry reloads
	// from the real store. The next login restores the saved gear with no
	// second template kit.
	destroyAll()
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(t, module.loadErr)
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: camp.RoomId})
	events.ProcessEvents()
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
	restored := live()
	assert.Equal(t, []int{10002, 10004, 30004}, mobItemIDs(restored), "exactly the recorded gear")
	assert.Equal(t, 2, restored.Character.Level)
	assert.Equal(t, 1, len(mobs.GetAllMobInstanceIds()))

	// Companion death: its gear dropped or was lost, so it isn't restored.
	deadID := restored.InstanceId
	nativeRuntime{}.Detach(7, deadID)
	events.AddToQueue(events.MobDeath{MobId: 58, InstanceId: deadID, RoomId: camp.RoomId, Level: 2})
	events.ProcessEvents()
	assert.Empty(t, stateItemIDs(stored()))
	assert.Equal(t, 2, stored().Level)
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: camp.RoomId})
	events.ProcessEvents()
	assert.Empty(t, mobItemIDs(live()), "no gear comes back from the dead")
}
