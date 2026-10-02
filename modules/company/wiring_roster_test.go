package company

import (
	"math/rand"
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
)

// TestRosterThroughPluginsLoad drives Phase 32a2 through the real entry
// points: plugins.Load merges the shipped roster config and recruiters,
// the shipped recruit base templates spawn real mobs, the world round is
// the real one, and every command goes through usercommands.TryCommand.
func TestRosterThroughPluginsLoad(t *testing.T) {
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
		"rooms/tutorial/zone-config.yaml":                      "name: Tutorial\nroomid: 901\n",
		"rooms/tutorial/901.yaml":                              "roomid: 901\nzone: Tutorial\ntitle: Muster Yard\ndescription: A notched hiring post.\n",
		"rooms/tutorial/907.yaml":                              "roomid: 907\nzone: Tutorial\ntitle: The Oath Stone\ndescription: Notices at the stone's foot.\n",
		"mobs/dunmar/61-tamsin_reed.yaml":                      shipped("mobs/dunmar/61-tamsin_reed.yaml"),
		"mobs/dunmar/62-brother_oswin.yaml":                    shipped("mobs/dunmar/62-brother_oswin.yaml"),
		"mobs/dunmar/63-garrick_vane.yaml":                     shipped("mobs/dunmar/63-garrick_vane.yaml"),
		"mobs/tutorial/69-corvin_blackthorn.yaml":              shipped("mobs/tutorial/69-corvin_blackthorn.yaml"),
		"mobs/dunmar/80-recruit_warrior.yaml":                  shipped("mobs/dunmar/80-recruit_warrior.yaml"),
		"mobs/dunmar/84-recruit_ranger.yaml":                   shipped("mobs/dunmar/84-recruit_ranger.yaml"),
		"items/weapons-10000/10002-guardsmans_broadsword.yaml": "itemid: 10002\nname: guardsman's broadsword\nnamesimple: broadsword\ntype: weapon\nhands: 1\nsubtype: slashing\ndamage:\n  diceroll: 1d6\n",
		"items/weapons-10000/10014-sling.yaml":                 "itemid: 10014\nname: sling\nnamesimple: sling\ntype: weapon\nhands: 1\nsubtype: shooting\ndamage:\n  diceroll: 1d4\n",
		"items/armor-20000/offhand/20004-wooden_shield.yaml":   "itemid: 20004\nname: wooden shield\nnamesimple: shield\ntype: offhand\nsubtype: wearable\n",
		"items/armor-20000/head/20020-leather_cap.yaml":        "itemid: 20020\nname: leather cap\nnamesimple: cap\ntype: head\nsubtype: wearable\n",
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
	module.rng = rand.New(rand.NewSource(32))
	t.Cleanup(func() { module.rng = nil })
	startRound := util.GetRoundCount()
	util.SetRoundCount(500000)
	t.Cleanup(func() { util.SetRoundCount(startRound) })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	newUser := func(id int, name string) *users.UserRecord {
		user := users.NewUserRecord(id, uint64(id))
		user.Username = name
		user.Password = "$2a$test"
		user.Character.Name = name
		user.Character.RoomId = inn.RoomId
		user.Character.RaceId = 1
		user.Character.Level = 2
		user.Character.Alignment = 0
		user.Character.Gold = 500
		user.Character.Validate()
		users.SetTestUser(user)
		inn.AddPlayer(user.UserId)
		return user
	}
	user, other := newUser(7, "Dain"), newUser(8, "Mira")
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
		inn.RemovePlayer(7)
		inn.RemovePlayer(8)
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})
	var byUser map[int][]string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		msg := e.(events.Message)
		byUser[msg.UserId] = append(byUser[msg.UserId], msg.Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	run := func(u *users.UserRecord, cmd, rest string) string {
		t.Helper()
		events.ProcessEvents()
		byUser = map[int][]string{}
		handled, err := usercommands.TryCommand(cmd, rest, u.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return companyTagPattern.ReplaceAllString(strings.Join(byUser[u.UserId], "\n"), "")
	}
	roster := func(leader int) domain.Roster {
		t.Helper()
		record, _ := module.registry.Get(leader)
		ros, ok := record.Roster(2003)
		require.True(t, ok)
		return ros
	}
	stored := func() domain.Record {
		t.Helper()
		reg := domain.NewRegistry()
		require.NoError(t, pluginStore{plug: module.plug}.Load(reg))
		record, ok := reg.Get(7)
		require.True(t, ok)
		return record
	}

	// Two players at one recruiter see their own lists on the notice.
	mine := run(user, "look", "")
	theirs := run(other, "look", "")
	assert.Contains(t, mine, "On the hiring slate by the hearth: ")
	myRoster, theirRoster := roster(7), roster(8)
	require.Len(t, myRoster.Candidates, 3)
	require.Len(t, theirRoster.Candidates, 3)
	assert.NotEqual(t, myRoster.Candidates, theirRoster.Candidates)
	for _, c := range myRoster.Candidates {
		assert.Contains(t, mine, c.Name)
		assert.Contains(t, run(user, "company", "recruit"), c.Name+" (company recruit "+c.Key+"): ")
	}
	for _, c := range theirRoster.Candidates {
		assert.Contains(t, theirs, c.Name)
	}

	// A set list for the rest: two warriors on one template, and one far
	// from the company's ways.
	far := uint64(500000 + 100000)
	myRoster.Candidates = []domain.Candidate{
		{Key: "hild", Name: "Hild Marrow", Archetype: "warrior", MobTemplateID: 80, Level: 3, Alignment: 10, Price: 120, Trait: "A scarred former caravan guard.", Arrived: 499000, Leaves: far},
		{Key: "wren", Name: "Wren", Archetype: "warrior", MobTemplateID: 80, Level: 1, Alignment: -10, Price: 60, Trait: "Quiet, and quick with a sling.", Arrived: 499500, Leaves: far + 1},
		{Key: "morrow", Name: "Morrow Black", Archetype: "ranger", MobTemplateID: 84, Level: 2, Alignment: -90, Price: 90, Arrived: 499900, Leaves: 500000 + 900},
	}
	require.NoError(t, module.registry.PutRoster(7, myRoster))

	assert.Contains(t, run(user, "look", "hild"), "You read about Hild Marrow on the hiring slate by the hearth: warrior, level 3, asking 120 gold.")
	assert.Contains(t, run(user, "company", "inspect morrow"), "They won't join a company so far from their ways.")
	assert.Contains(t, run(user, "company", "recruit morrow"), "Morrow Black (alignment -90, ")
	assert.Equal(t, 500, user.Character.Gold)

	assert.Contains(t, run(user, "company", "recruit hild"), "You pay 120 gold. Hild Marrow joins your company (#1).")
	assert.Contains(t, run(user, "company", "recruit wren"), "You pay 60 gold. Wren joins your company (#2).")
	assert.Equal(t, 320, user.Character.Gold)
	assert.Equal(t, theirRoster, roster(8), "another player's list is untouched by these hires")
	assert.Len(t, run(other, "look", ""), len(theirs))

	// Both are real companions under their own names, on one template.
	record := stored()
	require.Len(t, record.Companions, 2)
	assert.Equal(t, "Hild Marrow", record.Companions[0].Name)
	assert.Equal(t, "Wren", record.Companions[1].Name)
	assert.Equal(t, 80, record.Companions[0].MobTemplateID)
	assert.Equal(t, 80, record.Companions[1].MobTemplateID)
	assert.Equal(t, 3, record.Companions[0].State.Level)
	savedRoster, _ := record.Roster(2003)
	assert.Len(t, savedRoster.Candidates, 1, "the hires rode in the same save")
	liveNames := func() map[string]*mobs.Mob {
		out := map[string]*mobs.Mob{}
		for _, c := range []int{1, 2} {
			if instanceID, ok := module.instance(7, c); ok {
				if mob := mobs.GetInstance(instanceID); mob != nil {
					out[mob.Character.Name] = mob
				}
			}
		}
		return out
	}
	live := liveNames()
	require.Contains(t, live, "Hild Marrow")
	require.Contains(t, live, "Wren")
	assert.Equal(t, 3, live["Hild Marrow"].Character.Level)
	assert.Equal(t, "A scarred former caravan guard.", live["Hild Marrow"].Character.Description)

	status := run(user, "company", "status")
	assert.Contains(t, status, "#1 Hild Marrow, level 3,")
	assert.Contains(t, status, "#2 Wren, level 1,")
	run(user, "formation", "move wren 2 3")
	run(user, "formation", "move hild 1 1")
	run(user, "formation", "move wren 1 2")
	formation := run(user, "formation", "")
	assert.Contains(t, formation, "Hild Marrow(#1)")
	assert.Contains(t, formation, "Wren(#2)")
	assert.Contains(t, run(user, "company", "gear wren"), "#2 Wren, level 1")
	for _, mob := range live {
		assert.Contains(t, inn.GetMobs(), mob.InstanceId, "%s stands in the room", mob.Character.Name)
	}
	members, ok := domain.CompanyMembers(7)
	require.True(t, ok)
	require.Len(t, members, 2)
	assert.Equal(t, "Hild Marrow", members[0].Name, "the browser panel's name")
	assert.Equal(t, "Wren", members[1].Name)

	// Logout, then restart or copyover: the registry reloads from the real
	// store and the next login restores both under their own names.
	events.AddToQueue(events.PlayerDespawn{UserId: 7})
	events.ProcessEvents()
	assert.Empty(t, liveNames())
	plugins.Save()
	for _, instance := range mobs.GetAllMobInstanceIds() {
		nativeRuntime{}.Detach(7, instance)
	}
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(t, module.loadErr)
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: inn.RoomId})
	events.ProcessEvents()
	live = liveNames()
	require.Contains(t, live, "Hild Marrow")
	require.Contains(t, live, "Wren")
	assert.Equal(t, 3, live["Hild Marrow"].Character.Level)
	assert.Equal(t, "Hild Marrow", stored().Companions[0].Name)
	assert.Contains(t, run(user, "company", "status"), "#2 Wren, level 1,")

	// Time passes (the world round, which rosters only read): the last
	// face has gone and new ones have come.
	util.SetRoundCount(far + 2000)
	notice := run(user, "look", "")
	assert.NotContains(t, notice, "Morrow Black")
	fresh := roster(7)
	require.Len(t, fresh.Candidates, 3)
	for _, c := range fresh.Candidates {
		assert.Contains(t, notice, c.Name)
		assert.NotEqual(t, "hild", c.Key, "never a name already in the company")
		assert.NotEqual(t, "wren", c.Key)
	}
	assert.Equal(t, far+2000, util.GetRoundCount(), "never advances the clock")

	// The tutorial's recruiters are authored only.
	tutorialNotice := func(roomID int) string {
		t.Helper()
		room := rooms.LoadRoom(roomID)
		require.NotNil(t, room)
		inn.RemovePlayer(8)
		other.Character.RoomId = roomID
		room.AddPlayer(8)
		t.Cleanup(func() { room.RemovePlayer(8) })
		return run(other, "company", "recruit")
	}
	muster := tutorialNotice(901)
	assert.Contains(t, muster, "Tamsin Reed (company recruit tamsin)")
	assert.Contains(t, muster, "Brother Oswin (company recruit oswin)")
	assert.Equal(t, 2, strings.Count(muster, "(company recruit "), "no one generated: %s", muster)
	oath := tutorialNotice(907)
	assert.Contains(t, oath, "Corvin Blackthorn (company recruit corvin)")
	assert.Equal(t, 1, strings.Count(oath, "(company recruit "))
	otherRecord, _ := module.registry.Get(8)
	for _, ros := range otherRecord.Rosters {
		assert.Equal(t, 2003, ros.RoomID, "no tutorial roster")
	}
}
