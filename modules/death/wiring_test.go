package death

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
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

	// The real modules a death reaches: the company (relocation), travel
	// and camping (abandon), and survival, which they need to load.
	_ "github.com/GoMudEngine/GoMud/modules/camping"
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/expedition"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func useDataDir(t *testing.T, dir string) {
	t.Helper()
	set := func(value string) error {
		flat := configs.Flatten(configs.GetOverrides())
		flat["FilePaths.DataFiles"] = value
		return configs.RestoreOverrides(flat)
	}
	previous := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, set(dir))
	t.Cleanup(func() { require.NoError(t, set(previous)) })
}

// writeWiringWorld builds a disposable world from the shipped Dunmar,
// Old Kings Road, and Frostfang Sanctuary rooms, the shipped chapel keeper,
// and a training dummy companion.
func writeWiringWorld(t *testing.T, dataDir string) {
	t.Helper()
	shipped := shippedWorld()
	copies := []string{
		"races/1-human.yaml",
		"rooms/dunmar/zone-config.yaml",
		"rooms/dunmar/2001.yaml",
		"rooms/dunmar/2004.yaml",
		"rooms/dunmar/2007.yaml",
		"rooms/old_kings_road/zone-config.yaml",
		"rooms/old_kings_road/2002.yaml",
		"rooms/frostfang/18.yaml",
		"mobs/dunmar/65-sister_maren.yaml",
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
		"keywords.yaml":                    "direction-aliases: {}\n",
		"rooms/frostfang/zone-config.yaml": "name: Frostfang\nroomid: 18\n",
		"mobs/dunmar/58-training_dummy.yaml": "mobid: 58\nzone: Dunmar\nitemdropchance: 0\ncharacter:\n  name: training dummy\n" +
			"  raceid: 1\n  level: 3\n  alignment: 10\n",
	}
	for path, data := range fixtures {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "items"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "combat-messages"), 0755))
}

// TestDeathThroughPluginsLoad drives Phase 25a through the real entry points:
// plugins.Load registers the death, company, travel, camping, and survival
// modules with their shipped overlays; the player walks into Dunmar with the
// real go command, recruits a companion, sets out on the Old King's Road,
// and dies through the real suicide command via usercommands.TryCommand.
// They wake in the Chapel of the Wayfarer, a level lower, with the
// companion; the journey is gone; the level and checkpoint survive a user
// save and reload; the clock never moves.
func TestDeathThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	writeWiringWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	for _, id := range []int{2001, 2002, 2004, 2007, 18} {
		require.NotNil(t, rooms.LoadRoom(id), "room %d", id)
	}

	require.NotNil(t, module, "init registered the module")
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	provider, ok := domain.Active()
	require.True(t, ok)
	assert.Same(t, module, provider)
	church, ok := module.config().registry.ChurchFor("Dunmar")
	assert.True(t, ok)
	assert.Equal(t, 2007, church, "the shipped overlay")

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
	user := users.NewUserRecord(7, 1)
	user.Username = "wren"
	user.Password = "$2a$test"
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Level = 5
	user.Character.Experience = user.Character.XPTL(4) + 25
	user.Character.ActionPoints = 100
	user.Character.Validate()
	user.Character.Health = user.Character.HealthMax.Value
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2002))
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
		return tagPattern.ReplaceAllString(strings.Join(messages, "\n"), "")
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// Old Kings Road isn't a city: no checkpoint yet.
	events.ProcessEvents()
	assert.Zero(t, checkpoint(user.Character))

	// Walk into Dunmar.
	out := run("south", "")
	assert.Equal(t, 2001, user.Character.RoomId)
	assert.Contains(t, out, "Should you fall, you will wake in The Chapel of the Wayfarer.")
	assert.Equal(t, 2007, checkpoint(user.Character))

	// Recruit, then set out north on the Old King's Road.
	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy (#1).")
	companion, ok := company.InstanceFor(user.UserId, 1)
	require.True(t, ok)
	assert.Contains(t, run("north", ""), "step onto the Old King's Road")
	// The exit message requeues the command with input blocked; the game
	// loop runs it next and unblocks input.
	require.True(t, user.InputBlocked())
	user.UnblockInput()
	handled, err := usercommands.TryCommand("north", "", user.UserId, events.CmdSkipScripts|events.CmdIsRequeue)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, tagPattern.ReplaceAllString(strings.Join(messages, "\n"), ""), "You lead your company from")
	assert.Equal(t, 2001, user.Character.RoomId, "the journey starts in place")
	blocked, _ := expedition.MovementBlocked(user.UserId)
	require.True(t, blocked, "travelling")

	// Die, with the companion mid-fight.
	fighting := mobs.GetInstance(companion)
	require.NotNil(t, fighting)
	fighting.Character.Aggro = &characters.Aggro{MobInstanceId: 999}
	out = run("suicide", "")
	assert.Contains(t, out, "You lose a level (now level 4).")
	assert.Contains(t, out, "You wake before the altar of The Chapel of the Wayfarer.")
	assert.Contains(t, out, "Your company is with you.")
	c := user.Character
	assert.Equal(t, 2007, c.RoomId)
	assert.Equal(t, 4, c.Level)
	assert.Equal(t, 5, c.PeakLevel)
	assert.Equal(t, c.XPTL(3), c.Experience)
	assert.Equal(t, max(1, c.HealthMax.Value/2), c.Health)
	assert.False(t, module.Pending(user.UserId))
	mob := mobs.GetInstance(companion)
	require.NotNil(t, mob, "the companion lives")
	assert.Equal(t, 2007, mob.Character.RoomId, "and came along")
	assert.Nil(t, mob.Character.Aggro, "out of the fight")
	assert.Contains(t, rooms.LoadRoom(2007).GetMobs(rooms.FindCharmed), companion)
	assert.NotContains(t, rooms.LoadRoom(2001).GetMobs(rooms.FindCharmed), companion)
	blocked, _ = expedition.MovementBlocked(user.UserId)
	assert.False(t, blocked, "the journey is over")
	assert.Contains(t, run("travel", "status"), "You are not travelling.")

	assert.Contains(t, run("company", "status"), "(present)", "still attached")

	// A second suicide in the same round (the combat loop and AutoHeal can
	// both queue one for a death) finds the player already back: ignored.
	run("suicide", "")
	assert.Equal(t, 4, c.Level, "one level for one death")
	assert.Equal(t, 2007, c.RoomId)

	// The level, peak, and checkpoint are in the user file.
	require.NoError(t, users.SaveUser(*user))
	data, err := os.ReadFile(filepath.Join(dataDir, "users", "7.yaml"))
	require.NoError(t, err)
	reloaded := users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	assert.Equal(t, 4, reloaded.Character.Level)
	assert.Equal(t, 5, reloaded.Character.PeakLevel)
	assert.Equal(t, 2007, checkpoint(reloaded.Character))
	assert.Nil(t, reloaded.Character.GetMiscData(domain.PendingKey))

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())

	// A player with no checkpoint wakes at the fallback, Frostfang's
	// Sanctuary, which becomes their checkpoint.
	stranger := users.NewUserRecord(8, 2)
	stranger.Username = "stranger"
	stranger.Password = "$2a$test"
	stranger.Character.Name = "Stranger"
	stranger.Character.RaceId = 1
	stranger.Character.Level = 2
	stranger.Character.Validate()
	users.SetTestUser(stranger)
	require.NoError(t, rooms.MoveToRoom(stranger.UserId, 2002))
	t.Cleanup(func() {
		if room := rooms.LoadRoom(stranger.Character.RoomId); room != nil {
			room.RemovePlayer(stranger.UserId)
		}
	})
	events.ProcessEvents()
	handled, err = usercommands.TryCommand("suicide", "", stranger.UserId, events.CmdSkipScripts)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	assert.Equal(t, 18, stranger.Character.RoomId)
	assert.Equal(t, 1, stranger.Character.Level)
	assert.Equal(t, 18, checkpoint(stranger.Character))
}

// spawnKeeper spawns a room's keeper now, whatever an earlier test left in
// its spawn record (a destroyed keeper would otherwise wait out its
// respawn time).
func spawnKeeper(t *testing.T, roomID int) {
	t.Helper()
	room := rooms.LoadRoom(roomID)
	require.NotNil(t, room)
	for i := range room.SpawnInfo {
		room.SpawnInfo[i].InstanceId = 0
		room.SpawnInfo[i].DespawnedRound = 0
	}
	room.Prepare(false)
	require.NotEmpty(t, room.GetMobs(), "the keeper is in room %d", roomID)
}
