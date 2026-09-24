package company

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	"gopkg.in/yaml.v2"
)

// TestRecruitThroughPluginsLoad drives Phase 22c through the real entry
// points: plugins.Load merges the shipped Recruiters config and runs
// OnLoad against the real plugin store; the shipped candidate mob files
// spawn real mobs; every command goes through usercommands.TryCommand.
func TestRecruitThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	world := shippedWorld(t)
	shipped := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(world, rel))
		require.NoError(t, err)
		return string(data)
	}
	fixtures := map[string]string{
		"biomes/default.yaml":                                  "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                                        "direction-aliases: {}\n",
		"races/1-human.yaml":                                   shipped("races/1-human.yaml"),
		"rooms/dunmar/zone-config.yaml":                        "name: Dunmar\nroomid: 2003\n",
		"rooms/dunmar/2003.yaml":                               "roomid: 2003\nzone: Dunmar\ntitle: The Waymark Inn\ndescription: A hiring slate hangs by the hearth.\n",
		"rooms/dunmar/2001.yaml":                               "roomid: 2001\nzone: Dunmar\ntitle: Dunmar West Gate\ndescription: A gate.\n",
		"mobs/dunmar/61-tamsin_reed.yaml":                      shipped("mobs/dunmar/61-tamsin_reed.yaml"),
		"mobs/dunmar/62-brother_oswin.yaml":                    shipped("mobs/dunmar/62-brother_oswin.yaml"),
		"mobs/dunmar/63-garrick_vane.yaml":                     shipped("mobs/dunmar/63-garrick_vane.yaml"),
		"items/weapons-10000/10015-crude_cudgel.yaml":          "itemid: 10015\nname: crude cudgel\nnamesimple: cudgel\ntype: weapon\nhands: 1\nsubtype: bludgeoning\ndamage:\n  diceroll: 1d4\n",
		"items/weapons-10000/10002-guardsmans_broadsword.yaml": "itemid: 10002\nname: guardsman's broadsword\nnamesimple: broadsword\ntype: weapon\nhands: 1\nsubtype: slashing\ndamage:\n  diceroll: 1d6\n",
		"items/armor-20000/offhand/20004-wooden_shield.yaml":   "itemid: 20004\nname: wooden shield\nnamesimple: shield\ntype: offhand\nsubtype: wearable\n",
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
	inn := rooms.LoadRoom(2003)
	require.NotNil(t, inn)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "hirer"
	user.Password = "$2a$test"
	user.Character.RoomId = inn.RoomId
	user.Character.RaceId = 1
	user.Character.Alignment = 30
	user.Character.Gold = 150
	user.Character.Validate()
	users.SetTestUser(user)
	inn.AddPlayer(user.UserId)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
		inn.RemovePlayer(7)
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
	stored := func() domain.Record {
		t.Helper()
		reg := domain.NewRegistry()
		require.NoError(t, pluginStore{plug: module.plug}.Load(reg))
		record, ok := reg.Get(7)
		require.True(t, ok)
		return record
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	list := run("company", "recruit")
	assert.Contains(t, list, "The hiring slate by the hearth lists:")
	assert.Contains(t, list, "Tamsin Reed (company recruit tamsin):")
	assert.Contains(t, list, "level 1, alignment 30")
	assert.Contains(t, list, "Gear: crude cudgel, wooden shield")
	assert.Contains(t, list, "Garrick Vane (company recruit garrick):")
	assert.Contains(t, list, "level 3")
	assert.Contains(t, list, "Price: 120 gold")
	assert.Contains(t, list, "Price: free, once only")

	// Names work as well as ids; an ambiguous one is refused.
	assert.Contains(t, run("company", "recruit r"), `No one called "r" is hiring here.`)
	assert.Equal(t, 150, user.Character.Gold)

	// "company summon" can't bypass the price or the claim.
	handled, err := usercommands.TryCommand("company", "summon 61", user.UserId, events.CmdSkipScripts)
	assert.True(t, handled)
	assert.ErrorIs(t, err, domain.ErrTemplateNotAllowed)
	handled, err = usercommands.TryCommand("company", "summon tamsin reed", user.UserId, events.CmdSkipScripts)
	assert.True(t, handled)
	assert.ErrorIs(t, err, domain.ErrTemplateNotAllowed)
	events.ProcessEvents()
	_, hasRecord := module.registry.Get(7)
	assert.False(t, hasRecord, "nothing recorded")

	// The free tutorial candidate joins with its template gear, claimed in
	// the same real save.
	assert.Contains(t, run("company", "recruit reed"), "Tamsin Reed joins your company (#1).")
	assert.Equal(t, 150, user.Character.Gold)
	record := stored()
	assert.True(t, record.HasClaimed(61))
	require.Len(t, record.Companions, 1)
	require.NotNil(t, record.Companions[0].State)
	assert.Equal(t, []int{10015, 20004}, stateItemIDs(record.Companions[0].State))

	// A paid candidate costs exactly its price, and the user is saved.
	assert.Contains(t, run("company", "recruit garrick"), "You pay 120 gold. Garrick Vane joins your company (#2).")
	assert.Equal(t, 30, user.Character.Gold)
	userData, err := os.ReadFile(filepath.Join(dataDir, "users", "7.yaml"))
	require.NoError(t, err)
	var savedUser users.UserRecord
	require.NoError(t, yaml.Unmarshal(userData, &savedUser))
	assert.Equal(t, 30, savedUser.Character.Gold, "the user file moved with the company file")
	assert.Len(t, stored().Companions, 2)

	// Not enough gold for another: nothing changes.
	assert.Contains(t, run("company", "recruit garrick"), "Garrick Vane asks 120 gold, and you have 30.")
	assert.Len(t, stored().Companions, 2)

	// Both are real companions: placed in the grid and listed in status.
	run("formation", "move tamsin 1 2")
	formation, ok := module.FormationFor(7)
	require.True(t, ok)
	assert.Equal(t, domain.CompanionMemberKey(1), formation[0][1])
	status := run("company", "status")
	assert.Contains(t, status, "Company companions (2/4)")
	assert.Contains(t, status, "Tamsin Reed")
	assert.Contains(t, status, "Garrick Vane")

	// Save, restart (reload from the real store), dismiss, and the free
	// offer still doesn't come twice.
	plugins.Save()
	for _, instance := range mobs.GetAllMobInstanceIds() {
		nativeRuntime{}.Detach(7, instance)
	}
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(t, module.loadErr)
	assert.Contains(t, run("company", "dismiss tamsin"), "Companion dismissed: Tamsin Reed (#1).")
	assert.True(t, stored().HasClaimed(61))
	assert.Contains(t, run("company", "recruit tamsin"), "You've already taken Tamsin Reed on once")
	assert.Contains(t, run("company", "recruit"), "Price: already claimed (once only)")
	assert.Contains(t, run("company", "recruit oswin"), "Brother Oswin joins your company (#3).")

	// Elsewhere, no one is hiring.
	gate := rooms.LoadRoom(2001)
	require.NotNil(t, gate)
	user.Character.RoomId = gate.RoomId
	inn.RemovePlayer(7)
	gate.AddPlayer(7)
	t.Cleanup(func() { gate.RemovePlayer(7) })
	assert.Contains(t, run("company", "recruit"), "No one here is hiring.")

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}
