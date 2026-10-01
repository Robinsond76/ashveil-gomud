package tutorial

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
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
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanyDockThroughPluginsLoad drives Phase 32g's server half through
// the real entry points: plugins.Load with the shipped modules, the real
// commands, and the gmcp module's company feed as the web client receives
// it. A player with Tamsin out sees both in Company.Inventory; the item
// references it carries work in `cargo put`, `cargo take`, and `give`,
// and each change is sent; a partly used waterskin keeps its uses both
// ways. `camp`, `camp fire`, and `camp rest` each update Company.Camp. A
// login re-sends every message. The clock never moves.
func TestCompanyDockThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	setOverrides(t, map[string]any{
		"FilePaths.DataFiles":        dataDir,
		"SpecialRooms.StartRoom":     1,
		"SpecialRooms.TutorialRooms": []any{"900", "901", "902", "903", "904", "905", "906", "907"},
	})
	writeTutorialWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	buffs.RegisterFS(plugins.GetPluginRegistry())
	buffs.LoadDataFiles()

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

	// What the web client receives, per module, in order.
	sent := map[string][]map[string]any{}
	gid := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out, ok := e.(gmcp.GMCPOut); ok && out.UserId == 7 && strings.HasPrefix(out.Module, "Company") {
			var body map[string]any
			if raw, ok := out.Payload.([]byte); ok {
				_ = json.Unmarshal(raw, &body)
			}
			sent[out.Module] = append(sent[out.Module], body)
		}
		return events.Cancel // no connection to deliver to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, gid) })
	last := func(module string) map[string]any {
		events.ProcessEvents()
		list := sent[module]
		if len(list) == 0 {
			return nil
		}
		return list[len(list)-1]
	}
	count := func(module string) int {
		events.ProcessEvents()
		return len(sent[module])
	}

	run := func(u *users.UserRecord, cmd, rest string) {
		t.Helper()
		events.ProcessEvents()
		handled, err := usercommands.TryCommand(cmd, rest, u.UserId, events.CmdSkipScripts)
		require.NoError(t, err, cmd+" "+rest)
		require.True(t, handled, cmd+" "+rest)
		events.ProcessEvents()
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	start := rooms.LoadRoom(1)
	require.NotNil(t, start)
	start.Tags = append(start.Tags, "camping")

	// A character who skipped the course with Tamsin.
	aria := users.NewUserRecord(7, 700)
	aria.Username = "acctaria"
	aria.Password = "$2a$test"
	aria.Character.Name = "Aria"
	aria.Character.RaceId = 1
	aria.Character.Validate()
	aria.Character.ActionPoints = 1000
	aria.Character.RoomId = -1
	users.SetTestUser(aria)
	void := &rooms.Room{RoomId: -1, Title: "The Void"}
	_, err := usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	aria.GetPrompt().GetNextQuestion().Answer("no")
	_, err = usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	p := progressOf(aria.Character)
	p.Stage = StageCompany
	p.save(aria.Character)
	require.True(t, module.Begin(aria.UserId))
	run(aria, "company", "recruit tamsin")
	run(aria, "tutorial", "skip yes")
	require.Equal(t, 1, aria.Character.RoomId)
	t.Cleanup(func() { start.RemovePlayer(aria.UserId) })

	// The Inventory tab: the player, then Tamsin, from her live mob.
	water := items.New(30015)
	water.Uses = 3
	ration := items.New(30004)
	require.True(t, aria.Character.StoreItem(water))
	require.True(t, aria.Character.StoreItem(ration))
	run(aria, "look", "")
	inv := last("Company.Inventory")
	require.NotNil(t, inv, "Company.Inventory is sent")
	assert.Equal(t, true, inv["companions_known"])
	members := inv["members"].([]any)
	require.Len(t, members, 2)
	assert.Equal(t, "Aria", members[0].(map[string]any)["name"])
	assert.Equal(t, "companion:1", members[1].(map[string]any)["key"])
	refOf := func(member map[string]any, name string) (string, map[string]any) {
		for _, raw := range member["carried"].([]any) {
			itm := raw.(map[string]any)
			if strings.Contains(itm["name"].(string), name) {
				return itm["ref"].(string), itm
			}
		}
		return "", nil
	}
	waterRef, waterItem := refOf(map[string]any{"carried": inv["cargo"]}, "waterskin")
	require.NotEmpty(t, waterRef)
	assert.EqualValues(t, 3, waterItem["uses"])
	assert.NotNil(t, inv["load"], "the load split")

	// Shared cargo has no second backpack: put/take are guidance only.
	before := count("Company.Inventory")
	run(aria, "cargo", "put "+waterRef)
	assert.Equal(t, before, count("Company.Inventory"), "guidance does not move assets")
	inv = last("Company.Inventory")
	_, left := refOf(inv["members"].([]any)[0].(map[string]any), "waterskin")
	assert.Nil(t, left, "out of the pack")
	var stack map[string]any
	for _, raw := range inv["cargo"].([]any) {
		if s := raw.(map[string]any); strings.Contains(s["name"].(string), "waterskin") {
			stack = s
		}
	}
	require.NotNil(t, stack, "in the cargo")
	assert.EqualValues(t, 3, stack["uses"])

	// Take also leaves the exact cargo instance and uses unchanged.
	run(aria, "cargo", "take "+stack["ref"].(string))
	_, back := refOf(map[string]any{"carried": last("Company.Inventory")["cargo"]}, "waterskin")
	require.NotNil(t, back, "still in shared cargo")
	assert.EqualValues(t, 3, back["uses"])

	// An unchanged inventory sends nothing.
	before = count("Company.Inventory")
	run(aria, "look", "")
	assert.Equal(t, before, count("Company.Inventory"), "nothing changed")

	// Giving supplies to your own companion is a no-op: supplies stay shared.
	rationRef, _ := refOf(map[string]any{"carried": last("Company.Inventory")["cargo"]}, ration.GetSpec().Name)
	require.NotEmpty(t, rationRef)
	run(aria, "give", rationRef+" tamsin")
	_, hers := refOf(last("Company.Inventory")["members"].([]any)[1].(map[string]any), ration.GetSpec().Name)
	assert.Nil(t, hers, "Tamsin has no separate personal inventory")
	_, shared := refOf(map[string]any{"carried": last("Company.Inventory")["cargo"]}, ration.GetSpec().Name)
	require.NotNil(t, shared, "the ration remains in cargo")

	// A new Company snapshot (a formation change) replaces what the client
	// stores under Company, so the unchanged Inventory follows it (32g
	// review finding 1).
	snapshots, invs := count("Company"), count("Company.Inventory")
	run(aria, "formation", "move #1 2 2")
	assert.Greater(t, count("Company"), snapshots, "the formation change sends a snapshot")
	assert.Greater(t, count("Company.Inventory"), invs, "and the Inventory follows it")

	// The Camp tab.
	camp := last("Company.Camp")
	require.NotNil(t, camp, "Company.Camp is sent")
	assert.Equal(t, false, camp["has_camp"])
	assert.Equal(t, true, camp["can_camp"], "the start room allows a camp")
	run(aria, "camp", "")
	camp = last("Company.Camp")
	assert.Equal(t, true, camp["has_camp"])
	assert.Equal(t, true, camp["here"])
	assert.Equal(t, false, camp["fire_lit"])
	run(aria, "camp", "fire")
	assert.Equal(t, true, last("Company.Camp")["fire_lit"])
	run(aria, "camp", "rest")
	camp = last("Company.Camp")
	assert.Equal(t, true, camp["resting"])
	assert.Greater(t, camp["rest_seconds"].(float64), 0.0)

	// A login (or copyover) re-sends every message.
	counts := map[string]int{}
	for _, m := range []string{"Company", "Company.Inventory", "Company.Camp"} {
		counts[m] = count(m)
	}
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: 1})
	events.ProcessEvents()
	run(aria, "look", "")
	for m, n := range counts {
		assert.Greater(t, count(m), n, m+" re-sent")
	}

	assert.Equal(t, turn, util.GetTurnCount(), "the clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
}
