package mount

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Phase 32f: the company carries, eats, and rides together.
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
)

var logisticsTags = regexp.MustCompile(`<[^>]*>`)

func shippedDefaultWorld() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

func useLogisticsDataDir(t *testing.T, dir string) {
	t.Helper()
	previous := configs.GetFilePathsConfig().DataFiles.String()
	set := func(value string) error {
		flat := map[string]any{}
		for k, v := range configs.Flatten(configs.GetOverrides()) {
			flat[k] = v
		}
		flat["FilePaths.DataFiles"] = value
		return configs.RestoreOverrides(flat)
	}
	require.NoError(t, set(dir))
	t.Cleanup(func() { require.NoError(t, set(previous)) })
}

func writeLogisticsWorld(t *testing.T, dataDir string) {
	t.Helper()
	shipped := shippedDefaultWorld()
	copies := []string{
		"races/1-human.yaml",
		"rooms/dunmar/zone-config.yaml",
		"rooms/dunmar/2001.yaml",
		"rooms/dunmar/2004.yaml",
		"items/other-0/31-satchel.yaml",
		"items/other-0/34-pack_saddle.yaml",
		"items/other-0/35-riding_saddle.yaml",
		"items/consumables-30000/30004-cheese_sandwich.yaml",
		"items/consumables-30000/30015-waterskin.yaml",
		"items/consumables-30000/30021-seared_game_meat.yaml",
		"buffs/17-well_fed.yaml",
		"buffs/34-hydrated.yaml",
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
		"mobs/dunmar/58-training_dummy.yaml": "mobid: 58\nzone: Dunmar\nitemdropchance: 0\ncharacter:\n  name: training dummy\n" +
			"  raceid: 1\n  level: 3\n  alignment: 10\n",
	}
	for path, data := range fixtures {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), []byte(data), 0600))
	}
	for _, dir := range []string{"users", "combat-messages"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, dir), 0755))
	}
}

// TestCompanyLogisticsThroughPluginsLoad drives Phase 32f through the real
// entry points: plugins.Load with the company, encumbrance, survival, and
// mount modules and their shipped overlays; shipped packs, saddles, food,
// and the Dunmar West Gate stable; every command through
// usercommands.TryCommand. A leader recruits, packs, buys and saddles
// horses, stores a half-drunk waterskin, lists the company inventory, and
// feeds everyone; a reload keeps it all and the clock never moves.
func TestCompanyLogisticsThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useLogisticsDataDir(t, dataDir)
	writeLogisticsWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	buffs.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	gate := rooms.LoadRoom(2001)
	require.NotNil(t, gate)
	require.Contains(t, gate.Tags, "stable")

	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	// Other tests here register their own module as the provider; this
	// one needs the registered module, as the server has it.
	mount.SetProvider(registered)
	t.Cleanup(func() { mount.SetProvider(registered) })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
	})
	user := users.NewUserRecord(32, 1)
	user.Username = "dain"
	user.Password = "$2a$test"
	user.Character.Name = "Dain"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Level = 5
	user.Character.Gold = 1000
	user.Character.Validate()
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2001))
	t.Cleanup(func() {
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
	})

	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == user.UserId {
			messages = append(messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	run := func(command, rest string) string {
		t.Helper()
		events.ProcessEvents()
		messages = nil
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return logisticsTags.ReplaceAllString(strings.Join(messages, "\n"), "")
	}
	capacity := func() int {
		t.Helper()
		load, ok := encumbrance.CurrentLoad(user.UserId)
		require.True(t, ok)
		return load.CapacityGrams
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// Alone: a little over 20 kg, not the old flat 200.
	alone := capacity()
	assert.GreaterOrEqual(t, alone, 20000)
	assert.Less(t, alone, 40000)
	assert.Contains(t, run("cargo", ""), "Capacity: members")

	// A recruit carries a share of their own.
	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned")
	two := capacity()
	assert.GreaterOrEqual(t, two-alone, 20000)

	// A satchel adds 5 kg to whoever carries it; a second on the same
	// member adds only its weight.
	user.Character.StoreItem(items.New(31))
	assert.Equal(t, two+5000, capacity(), "the leader's satchel")
	run("give", "satchel dummy")
	assert.Equal(t, two+5000, capacity(), "now the companion's")
	user.Character.StoreItem(items.New(31))
	assert.Equal(t, two+10000, capacity())
	run("give", "satchel dummy")
	assert.Equal(t, two+5000, capacity(), "one pack per member counts")
	packed := capacity()

	// Horses: bought at the stable, saddled from the pack.
	assert.Contains(t, run("mount", "stable pack-horse"), "pack horse")
	assert.Equal(t, 1000-120, user.Character.Gold)
	assert.Equal(t, packed+40000, capacity(), "a bare pack horse carries 40 kg")
	user.Character.StoreItem(items.New(34))
	run("mount", "saddle pack pack saddle")
	assert.Equal(t, packed+100000, capacity(), "saddled, 100 kg")
	run("mount", "stable pack-horse")
	assert.Contains(t, run("mount", "stable pack-horse"), "one pack horse per member: 2 already for 2")
	herd := mountModule().herds[user.UserId]
	assert.Equal(t, 2, len(herd.Horses), "two members keep two pack horses")
	run("mount", "stable riding-horse")
	run("mount", "stable riding-horse")
	assert.Contains(t, run("mount", "stable riding-horse"), "one riding horse per member")
	herd = mountModule().herds[user.UserId]
	assert.Len(t, herd.Horses, 4, "and two riding horses; the third of each is refused")

	// Cargo keeps a half-drunk waterskin half-drunk.
	skin := items.New(30015)
	skin.Uses = 3
	user.Character.StoreItem(skin)
	run("cargo", "put waterskin")
	assert.Contains(t, run("cargo", ""), "waterskin x1 (3 uses left)")
	run("cargo", "take waterskin")
	taken, found := user.Character.FindInBackpack("waterskin")
	require.True(t, found)
	assert.Equal(t, 3, taken.Uses)
	user.Character.RemoveItem(taken)

	// One screen for everything.
	inv := run("company", "inventory")
	for _, want := range []string{"Company load:", "Dain (you)", "training dummy", "pack: satchel (+5.0 kg)", "Horses: #1 pack horse (pack saddle, +100.0 kg)", "Cargo: empty"} {
		assert.Contains(t, inv, want)
	}

	// A meal: the cargo first, then each member's own pack, then the
	// leader's.
	_, err := survival.ApplyMemberDrain(user.UserId, survival.LeaderMemberKey, survival.Exertion{Hunger: 30, Thirst: 50})
	require.NoError(t, err)
	_, err = survival.ApplyMemberDrain(user.UserId, survival.CompanionMemberKey(1), survival.Exertion{Hunger: 50, Thirst: 70})
	require.NoError(t, err)
	meat := items.New(30021)
	user.Character.StoreItem(meat)
	require.Contains(t, run("cargo", "put seared"), "You stow")
	user.Character.StoreItem(items.New(30004))
	companion := liveCompanion(t)
	companion.Character.StoreItem(items.New(30015))

	buffed := []int{}
	buffListener := events.RegisterListener(events.Buff{}, func(e events.Event) events.ListenerReturn {
		if b := e.(events.Buff); b.UserId == user.UserId {
			buffed = append(buffed, b.BuffId)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	meal := run("company", "meal")
	assert.Contains(t, meal, "training dummy eats a seared game meat (cargo)")
	assert.Contains(t, meal, "cheese sandwich (your pack). Hunger: Well fed.")
	assert.Contains(t, meal, "training dummy drinks from a waterskin (own pack)")
	assert.Contains(t, meal, "Dain is still thirsty; there's nothing left to drink.")
	assert.Empty(t, encumbrance.CargoContents(user.UserId), "the cargo meat was eaten")
	sandwich, found := user.Character.FindInBackpack("sandwich")
	require.True(t, found)
	assert.Equal(t, 2, sandwich.Uses)
	assert.Equal(t, []int{17}, buffed, "the leader's sandwich makes them Well Fed; the companion gets no buff")
	own, found := companion.Character.FindInBackpack("waterskin")
	require.True(t, found)
	assert.Equal(t, 4, own.Uses)
	needs := survival.CompanyNeeds(user.UserId)
	require.Len(t, needs, 2)
	assert.Equal(t, 100, needs[0].Needs.Hunger, "70 + 35, capped")
	assert.Equal(t, 50, needs[0].Needs.Thirst, "no water left for the leader")
	assert.Equal(t, 50+40, needs[1].Needs.Hunger)
	assert.Equal(t, 30+40, needs[1].Needs.Thirst)
	assert.Contains(t, run("company", "eat"), "No one in your company is hungry.")

	// Restart: the herd, saddles, and needs come back as they were.
	before := capacity()
	plugins.Save()
	plugins.Load(dataDir)
	assert.Len(t, mountModule().herds[user.UserId].Horses, 4)
	assert.Equal(t, before, capacity())
	assert.Equal(t, 50+40, survival.CompanyNeeds(user.UserId)[1].Needs.Hunger)

	// Water only in the cargo: everyone thirsty drinks from it, and a
	// restart keeps both the drink and what's left in the skin.
	user.Character.StoreItem(items.New(30015))
	require.Contains(t, run("cargo", "put waterskin"), "You stow")
	drink := run("company", "drink")
	assert.Contains(t, drink, "You drink from the waterskin (cargo)")
	assert.Contains(t, drink, "training dummy drinks from a waterskin (cargo)")
	plugins.Save()
	plugins.Load(dataDir)
	assert.Equal(t, []encumbrance.CargoStack{{ItemId: 30015, Count: 1, Uses: 3}}, encumbrance.CargoContents(user.UserId))
	needs = survival.CompanyNeeds(user.UserId)
	assert.Equal(t, 50+40, needs[0].Needs.Thirst)
	assert.Equal(t, 100, needs[1].Needs.Thirst)
	assert.Contains(t, run("company", "drink"), "No one in your company is thirsty.", "the hydrated are skipped")

	// Dismissed: the horses stay, but none can be added.
	run("company", "dismiss all")
	assert.Less(t, capacity(), before)
	assert.Contains(t, run("mount", "stable riding-horse"), "one riding horse per member: 2 already for 1")
	assert.Len(t, mountModule().herds[user.UserId].Horses, 4)

	assert.Equal(t, turn, util.GetTurnCount(), "logistics never advance the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

// mountModule is the registered module instance.
func mountModule() *MountModule {
	return registered
}

func liveCompanion(t *testing.T) *mobs.Mob {
	t.Helper()
	for _, instance := range mobs.GetAllMobInstanceIds() {
		if mob := mobs.GetInstance(instance); mob != nil && mob.MobId == 58 {
			return mob
		}
	}
	t.Fatal("no live companion")
	return nil
}
