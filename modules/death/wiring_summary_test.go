package death

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Phase 26a: the company load comes from the encumbrance module.
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
)

// TestCompanySummaryThroughPluginsLoad drives Phase 26a through the real
// entry points: plugins.Load with the company, survival, travel, camping,
// death, and encumbrance modules and their shipped overlays; the shipped
// status, conditions, and inventory layouts; commands through
// usercommands.TryCommand; the prompt through GetCommandPrompt and the
// real NewRound refresh. A leader and two companions are followed through
// hunger, a journey, a camp rest, a death, a resurrection, and a reload.
func TestCompanySummaryThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	writeWiringWorld(t, dataDir)
	shipped := shippedWorld()
	for _, path := range []string{
		"panel-layouts/character/status.yaml",
		"panel-layouts/character/conditions.yaml",
		"panel-layouts/character/inventory.yaml",
		"templates/character/experience.template",
	} {
		data, err := os.ReadFile(filepath.Join(shipped, path))
		require.NoError(t, err, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), data, 0600))
	}
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()

	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)

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
	user := users.NewUserRecord(17, 1)
	user.Username = "tamsin"
	user.Password = "$2a$test"
	user.Character.Name = "Tamsin"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Level = 5
	user.Character.ActionPoints = 100
	user.Character.Validate()
	user.Character.Health = user.Character.HealthMax.Value
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2002))
	t.Cleanup(func() {
		_ = expedition.AbandonForDeath(user.UserId)
		_ = camping.AbandonForDeath(user.UserId)
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
		return tagPattern.ReplaceAllString(strings.Join(messages, "\n"), "")
	}
	newRound := func() {
		events.AddToQueue(events.NewRound{RoundNumber: util.GetRoundCount()})
		events.ProcessEvents()
	}
	prompt := func() string {
		return tagPattern.ReplaceAllString(user.GetCommandPrompt(), "")
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// Into Dunmar; two companions.
	run("south", "")
	require.Equal(t, 2001, user.Character.RoomId)
	for i := 0; i < 2; i++ {
		assert.Contains(t, run("company", "summon training dummy"), "Companion summoned")
	}
	status := run("status", "")
	for _, want := range []string{"Vitals", "Hunger:", "Well fed (100)", "Members:", "3 alive", "Load:", "Unburdened", "Wake at:", "The Chapel of the Wayfarer"} {
		assert.Contains(t, status, want)
	}
	assert.NotContains(t, prompt(), "Hungry", "all is well: the default prompt stays quiet")

	// Hunger: the real NewRound refresh puts it in the prompt.
	_, err := survival.ApplyMemberDrain(user.UserId, survival.LeaderMemberKey, survival.Exertion{Hunger: 60})
	require.NoError(t, err)
	newRound()
	assert.Contains(t, prompt(), "] Hungry")
	assert.Contains(t, run("conditions", ""), "Hungry (40)")

	// A journey: the command's own refresh shows it at once.
	assert.Contains(t, run("north", ""), "step onto the Old King's Road")
	require.True(t, user.InputBlocked())
	user.UnblockInput()
	handled, err := usercommands.TryCommand("north", "", user.UserId, events.CmdSkipScripts|events.CmdIsRequeue)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, prompt(), "Travelling")
	assert.Contains(t, run("status", ""), "Travelling")

	// Off the road (ended directly; its timer is Phase 5's), a camp rest.
	require.NoError(t, expedition.AbandonForDeath(user.UserId))
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2002))
	run("camp", "")
	run("camp", "fire")
	assert.Contains(t, run("camp", "rest"), "rest")
	assert.Contains(t, prompt(), "Resting")
	assert.Contains(t, run("status", ""), "Resting")

	// A companion dies through the real mob suicide.
	instanceID, ok := company.InstanceFor(user.UserId, 1)
	require.True(t, ok)
	mob := mobs.GetInstance(instanceID)
	_, err = mobcommands.Suicide("", mob, rooms.LoadRoom(mob.Character.RoomId))
	require.NoError(t, err)
	events.ProcessEvents()
	newRound()
	assert.Equal(t, "2, 1 dead", user.ProcessPromptString("{company}"))
	assert.Contains(t, run("status", ""), "2 alive, 1 fallen")
	assert.Contains(t, run("inventory", ""), "Company load: Unburdened")

	// The chapel raises it.
	require.NoError(t, camping.AbandonForDeath(user.UserId))
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2004))
	run("east", "")
	require.Equal(t, 2007, user.Character.RoomId)
	spawnKeeper(t, 2007)
	assert.Contains(t, run("resurrect", "#1"), "draws breath again")
	status = run("status", "")
	assert.Contains(t, status, "3 alive")
	assert.NotContains(t, status, "fallen")
	assert.Equal(t, "3", user.ProcessPromptString("{company}"))

	// The summary is built from what the modules keep: after a save and a
	// reload of every module from disk, it reads the same.
	before := companyview.For(user)
	plugins.Save()
	plugins.Load(dataDir)
	after := companyview.For(user)
	assert.Equal(t, before.Alive, after.Alive)
	assert.Equal(t, before.Dead, after.Dead)
	assert.Equal(t, before.Leader.Hunger, after.Leader.Hunger)
	assert.Equal(t, before.LoadLabel, after.LoadLabel)
	assert.Equal(t, before.Checkpoint, after.Checkpoint)
	assert.Equal(t, before.Activity, after.Activity)
	assert.Equal(t, before.RestTier, after.RestTier)
	assert.Equal(t, before.Leader.Warmth, after.Leader.Warmth)
	require.Len(t, after.Companions, len(before.Companions))
	for i := range before.Companions {
		assert.Equal(t, before.Companions[i].Key, after.Companions[i].Key)
		assert.Equal(t, before.Companions[i].Status, after.Companions[i].Status)
	}

	// The leader's own death: experience names the level it cost.
	assert.NotContains(t, run("experience", ""), "last death")
	run("suicide", "")
	assert.Equal(t, 4, user.Character.Level)
	assert.Contains(t, run("experience", ""), "Your last death cost you a level: 5 to 4.")

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}
