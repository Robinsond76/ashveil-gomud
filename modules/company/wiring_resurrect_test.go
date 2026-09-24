package company

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	death "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// The death module owns the resurrect command and the settlement
	// registry with its keepers.
	_ "github.com/GoMudEngine/GoMud/modules/death"
)

// TestCompanionDeathAndResurrectionThroughPluginsLoad drives Phase 25b
// through the real entry points: plugins.Load registers the company and
// death modules with their shipped overlays; the shipped Dunmar chapel and
// Fernhollow lodge load with their keepers; companions die through the real
// mob suicide; logout and login go through PlayerDespawn and PlayerSpawn;
// time is charged by the real NewRound listener; and resurrect and company
// run through usercommands.TryCommand.
func TestCompanionDeathAndResurrectionThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	_, thisFile, _, _ := runtime.Caller(0)
	shipped := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	copies := []string{
		"races/1-human.yaml",
		"rooms/dunmar/zone-config.yaml",
		"rooms/dunmar/2004.yaml",
		"rooms/dunmar/2007.yaml",
		"rooms/old_kings_road/zone-config.yaml",
		"rooms/old_kings_road/2002.yaml",
		"rooms/fernhollow/zone-config.yaml",
		"rooms/fernhollow/2008.yaml",
		"rooms/fernhollow/2009.yaml",
		"mobs/dunmar/65-sister_maren.yaml",
		"mobs/fernhollow/66-old_wenna.yaml",
	}
	biomes, err := filepath.Glob(filepath.Join(shipped, "biomes", "*.yaml"))
	require.NoError(t, err)
	for _, path := range biomes {
		copies = append(copies, filepath.Join("biomes", filepath.Base(path)))
	}
	for _, path := range copies {
		data, err := os.ReadFile(filepath.Join(shipped, path))
		require.NoError(t, err, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), data, 0600))
	}
	fixtures := map[string]string{
		"keywords.yaml": "direction-aliases: {}\n",
		"items/weapons-10000/10002-guardsmans_broadsword.yaml": "itemid: 10002\nname: guardsman's broadsword\nnamesimple: broadsword\ntype: weapon\nhands: 1\nsubtype: slashing\ndamage:\n  diceroll: 1d6\n",
		"items/consumables-30000/30004-cheese_sandwich.yaml":   "itemid: 30004\nname: cheese sandwich\nnamesimple: sandwich\ntype: food\nsubtype: edible\nvalue: 20\n",
		// Like the shipped recruits: worn gear never drops (itemdropchance
		// 0), so it stays on the body and comes back; the carried sandwich
		// always drops.
		"mobs/dunmar/58-training_dummy.yaml": "mobid: 58\nzone: Dunmar\nitemdropchance: 0\ncharacter:\n  name: training dummy\n  raceid: 1\n  level: 3\n  alignment: 10\n" +
			"  equipment:\n    weapon:\n      itemid: 10002\n  items:\n    - itemid: 30004\n",
	}
	for path, data := range fixtures {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "combat-messages"), 0755))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	for _, id := range []int{2002, 2004, 2007, 2008, 2009} {
		require.NotNil(t, rooms.LoadRoom(id), "room %d", id)
	}

	lifecycle := &fakeLifecycle{}
	useFakeLifecycle(t, lifecycle)
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)
	_, active := death.Active()
	require.True(t, active, "the death module is registered")
	clock := &testClock{now: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)}
	module.clock = clock.Now
	t.Cleanup(func() { module.clock = nil; module.anchors = nil })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "wren"
	user.Password = "$2a$test"
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.ActionPoints = 100
	user.Character.Validate()
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2004))
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})
	messages := captureCompanyMessages(t)
	run := func(cmd, rest string) string {
		t.Helper()
		events.ProcessEvents()
		*messages = nil
		handled, err := usercommands.TryCommand(cmd, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}
	heard := func() string {
		events.ProcessEvents()
		out := companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
		*messages = nil
		return out
	}
	stored := func() domain.Record {
		t.Helper()
		reg := domain.NewRegistry()
		require.NoError(t, pluginStore{plug: module.plug}.Load(reg))
		record, _ := reg.Get(7)
		return record
	}
	kill := func(companionID int) {
		t.Helper()
		instanceID, ok := module.instance(7, companionID)
		require.True(t, ok)
		mob := mobs.GetInstance(instanceID)
		require.NotNil(t, mob)
		_, err := mobcommands.Suicide("", mob, rooms.LoadRoom(mob.Character.RoomId))
		require.NoError(t, err)
		events.ProcessEvents()
	}
	newRound := func() {
		events.AddToQueue(events.NewRound{RoundNumber: util.GetRoundCount()})
		events.ProcessEvents()
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// Three training dummies join; all three share a name.
	for i := 1; i <= 3; i++ {
		assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy")
	}

	// #1 dies through the real mob suicide: dead on the roster, its worn
	// broadsword kept, the carried sandwich dropped, three hours to raise.
	kill(1)
	assert.Contains(t, heard(), "training dummy has fallen. You have 3h 0m of your own time")
	record := stored()
	require.True(t, record.Companions[0].Dead(), "saved dead")
	assert.Equal(t, 10800, record.Companions[0].Death.Remaining)
	assert.Equal(t, []int{10002}, stateItemIDs(record.Companions[0].State))
	assert.Contains(t, run("company", "status"), "fallen, 3h 0m left to raise")

	// Logout, five hours away, login: the dead companion isn't restored and
	// no time was spent offline; the living two are back.
	events.AddToQueue(events.PlayerDespawn{UserId: 7})
	events.ProcessEvents()
	clock.advance(5 * time.Hour)
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: 2004})
	out := heard()
	assert.Contains(t, out, "training dummy lies fallen: 3h 0m of your time remain")
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked, "the dead aren't restored at login")
	for _, id := range []int{2, 3} {
		_, tracked := module.instance(7, id)
		assert.True(t, tracked, "#%d", id)
	}
	clock.advance(4 * time.Second)
	newRound()
	c, _ := findCompanion(func() domain.Record { r, _ := module.registry.Get(7); return r }(), 1)
	assert.Equal(t, 10796, c.Death.Remaining, "only online time is spent")

	// The market square is no place for the rite.
	assert.Contains(t, run("resurrect", "dummy"), "There is no one here who can call back the dead.")
	assert.Contains(t, run("resurrect", ""), "#1 training dummy, level 3: 2h 59m")

	// East into the Chapel of the Wayfarer, before Sister Maren. The name is
	// shared with two living dummies; the dead one answers.
	run("east", "")
	require.Equal(t, 2007, user.Character.RoomId)
	chapel := rooms.LoadRoom(2007)
	chapel.Prepare(false)
	out = run("resurrect", "dummy")
	assert.Contains(t, out, "Sister Maren kneels and calls training dummy back from death.")
	assert.Contains(t, out, "now level 2")
	instanceID, tracked := module.instance(7, 1)
	require.True(t, tracked)
	raised := mobs.GetInstance(instanceID)
	require.NotNil(t, raised)
	assert.Equal(t, 2007, raised.Character.RoomId)
	assert.Equal(t, 2, raised.Character.Level)
	assert.Equal(t, []int{10002}, mobItemIDs(raised), "the kept broadsword returns")
	assert.True(t, raised.Character.IsCharmed(7))
	assert.False(t, stored().Companions[0].Dead(), "saved alive")
	assert.Contains(t, run("resurrect", "#1"), "is not dead", "raised once")

	// #2 dies. The village of Fernhollow, west of the Black Oak: Old Wenna
	// raises it, and the village never becomes the checkpoint.
	kill(2)
	heard()
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2002))
	checkpointBefore := user.Character.GetMiscData(death.CheckpointKey)
	run("west", "")
	run("west", "")
	require.Equal(t, 2009, user.Character.RoomId)
	assert.Equal(t, checkpointBefore, user.Character.GetMiscData(death.CheckpointKey), "a village is never a checkpoint")
	rooms.LoadRoom(2009).Prepare(false)
	out = run("resurrect", "#2")
	assert.Contains(t, out, "Old Wenna kneels and calls training dummy back from death.")
	instanceID, tracked = module.instance(7, 2)
	require.True(t, tracked)
	assert.Equal(t, 2009, mobs.GetInstance(instanceID).Character.RoomId)

	// #3 dies and its time runs out: lost, archived, its slot freed and its
	// ID never reused.
	kill(3)
	heard()
	record, _ = module.registry.Get(7)
	record.Companions[2].Death.Remaining = 3
	module.registry.Put(record)
	clock.advance(4 * time.Second)
	newRound()
	assert.Contains(t, heard(), "training dummy is lost to you.")
	record = stored()
	require.Len(t, record.Companions, 2)
	require.Len(t, record.Lost, 1)
	assert.Equal(t, 3, record.Lost[0].ID)
	assert.Contains(t, lifecycle.removed, [2]int{7, 3})
	assert.Contains(t, run("company", "status"), "Lost: #3 training dummy (level 3)")
	assert.Contains(t, run("resurrect", "#3"), "It is too late")
	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy (#4).")

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}
