package tutorial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
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

	// The company module owns recruiting and formation; survival loads
	// with it. Phase 27b: camping, and the modules behind the Survival
	// stage's inspections (weather, temperature, strain, cargo).
	_ "github.com/GoMudEngine/GoMud/modules/camping"
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
	_ "github.com/GoMudEngine/GoMud/modules/exposure"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
	_ "github.com/GoMudEngine/GoMud/modules/walking"
	_ "github.com/GoMudEngine/GoMud/modules/weather"
)

// setOverrides sets config keys for one test and restores them after.
func setOverrides(t *testing.T, values map[string]any) {
	t.Helper()
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	for k, v := range values {
		flat[k] = v
	}
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
}

// writeTutorialWorld builds a disposable world from the shipped tutorial
// rooms, the start room, the tutorial recruits and their gear, the
// graduation cap, and the shipped keywords (for the command aliases).
func writeTutorialWorld(t *testing.T, dataDir string) {
	t.Helper()
	shipped := filepath.Join(repoRoot(), "_datafiles", "world", "default")
	copies := []string{
		"keywords.yaml",
		"races/1-human.yaml",
		"rooms/tutorial/zone-config.yaml",
		"rooms/tutorial/900.yaml",
		"rooms/tutorial/901.yaml",
		"rooms/tutorial/902.yaml",
		"rooms/tutorial/903.yaml",
		"rooms/tutorial/904.yaml",
		"rooms/tutorial/905.yaml",
		"rooms/frostfang/1.yaml",
		"rooms/nowhere/zone-config.yaml",
		"rooms/nowhere/-1.yaml",
		"mobs/dunmar/61-tamsin_reed.yaml",
		"mobs/dunmar/62-brother_oswin.yaml",
		"items/armor-20000/head/20043-graduation_cap.yaml",
	}
	for _, id := range []string{"10015", "20004", "20008", "30004", "30015"} {
		matches, err := filepath.Glob(filepath.Join(shipped, "items", "*", id+"-*.yaml"))
		require.NoError(t, err)
		more, err := filepath.Glob(filepath.Join(shipped, "items", "*", "*", id+"-*.yaml"))
		require.NoError(t, err)
		matches = append(matches, more...)
		require.Len(t, matches, 1, id)
		rel, err := filepath.Rel(shipped, matches[0])
		require.NoError(t, err)
		copies = append(copies, rel)
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
		"rooms/frostfang/zone-config.yaml": "name: Frostfang\nroomid: 1\n",
	}
	for path, data := range fixtures {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), []byte(data), 0600))
	}
	for _, dir := range []string{"users", "plugin-data", "combat-messages", "buffs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, dir), 0755))
	}
}

// TestTutorialThroughPluginsLoad drives Phase 27a through the real entry
// points: plugins.Load with the tutorial and company modules and their
// shipped overlays, the shipped tutorial rooms, and the real `start`
// command. A new player passes Character with aliases, recruits Tamsin and
// Oswin in their own Muster Yard, sets a formation, logs out and back in
// mid-course and resumes, then walks out of the Gate with one graduation
// cap. A second player skips and gets nothing. The clock never moves.
func TestTutorialThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	setOverrides(t, map[string]any{
		"FilePaths.DataFiles":        dataDir,
		"SpecialRooms.StartRoom":     1,
		"SpecialRooms.TutorialRooms": []any{"900", "901", "902", "903", "904", "905"},
	})
	writeTutorialWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	for _, id := range []int{900, 901, 902, 903, 904, 905, 1} {
		require.NotNil(t, rooms.LoadRoom(id), "room %d", id)
	}
	require.Equal(t, []int{900, 901, 902, 903, 904, 905}, configuredRooms())

	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	assert.Equal(t, 20043, module.graduationItem, "the shipped overlay")
	assert.Equal(t, 30004, module.rationItem)
	assert.Equal(t, 30015, module.waterItem)
	// The rest tier buffs ship with the walking module.
	buffs.RegisterFS(plugins.GetPluginRegistry())
	buffs.LoadDataFiles()
	require.NotNil(t, buffs.GetBuffSpec(1033), "Rested")

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
	newPlayer := func(id int, name string) *users.UserRecord {
		u := users.NewUserRecord(id, uint64(id*100))
		u.Username = "acct" + name
		u.Password = "$2a$test"
		u.Character.Name = name
		u.Character.RaceId = 1
		u.Character.Validate()
		u.Character.ActionPoints = 1000
		u.Character.RoomId = -1
		users.SetTestUser(u)
		t.Cleanup(func() {
			if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
				room.RemovePlayer(u.UserId)
			}
		})
		return u
	}
	var messages = map[int][]string{}
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		m := e.(events.Message)
		messages[m.UserId] = append(messages[m.UserId], m.Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lid) })
	text := func(u *users.UserRecord) string {
		events.ProcessEvents()
		out := tagPattern.ReplaceAllString(strings.Join(messages[u.UserId], "\n"), "")
		messages[u.UserId] = nil
		return out
	}
	run := func(u *users.UserRecord, cmd, rest string) string {
		t.Helper()
		events.ProcessEvents()
		handled, err := usercommands.TryCommand(cmd, rest, u.UserId, events.CmdSkipScripts)
		require.NoError(t, err, cmd+" "+rest)
		require.True(t, handled, cmd+" "+rest)
		return text(u)
	}
	void := &rooms.Room{RoomId: -1, Title: "The Void"}
	create := func(u *users.UserRecord) {
		t.Helper()
		_, err := usercommands.Start("", u, void, 0)
		require.NoError(t, err)
		q := u.GetPrompt().GetNextQuestion()
		require.NotNil(t, q)
		require.Equal(t, `Would you like to skip the tutorial?`, q.Question)
		q.Answer("no")
		_, err = usercommands.Start("", u, void, 0)
		require.NoError(t, err)
	}
	stageOf := func(u *users.UserRecord) StageID { return progressOf(u.Character).Stage }
	template := func(u *users.UserRecord) int { return rooms.GetOriginalRoom(u.Character.RoomId) }
	caps := func(u *users.UserRecord) int {
		n := 0
		for _, itm := range u.Character.GetAllBackpackItems() {
			if itm.ItemId == 20043 {
				n++
			}
		}
		if u.Character.Equipment.Head.ItemId == 20043 {
			n++
		}
		return n
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// Creation hands the new character to the course: their own copy of
	// the Waking Hall, with the first lesson shown.
	aria := newPlayer(7, "Aria")
	create(aria)
	got := text(aria)
	assert.Equal(t, 900, template(aria))
	assert.NotEqual(t, 900, aria.Character.RoomId, "a copy, not the template")
	assert.Contains(t, got, "A long, low hall")
	assert.Contains(t, got, "stage 1 of 6")
	assert.Equal(t, StageCharacter, stageOf(aria))
	assert.Contains(t, run(aria, "tutorial", ""), "Goal:")

	// No command is blocked, and the way east stays shut until the stage
	// is passed.
	run(aria, "say", "hello")
	before := aria.Character.RoomId
	run(aria, "east", "")
	assert.Equal(t, before, aria.Character.RoomId, "no way east yet")

	// Character: the inspections by alias.
	for _, alias := range []string{"score", "i", "xp"} {
		run(aria, alias, "")
	}
	assert.Equal(t, StageCharacter, stageOf(aria), "one inspection still to go")
	got = run(aria, "c", "")
	assert.Equal(t, StageCompany, stageOf(aria))
	assert.Contains(t, got, "Head east for the next lesson")

	// Company: recruit both tutorial candidates in the Muster Yard copy.
	got = run(aria, "east", "")
	require.Equal(t, 901, template(aria))
	assert.Contains(t, got, "stage 2 of 6")
	got = run(aria, "company", "recruit")
	assert.Contains(t, got, "Tamsin Reed")
	assert.Contains(t, got, "Oswin")
	run(aria, "company", "recruit tamsin")
	assert.Equal(t, StageCompany, stageOf(aria), "one companion isn't enough")
	got = run(aria, "company", "recruit oswin")
	assert.Equal(t, StageFormation, stageOf(aria))
	assert.Contains(t, got, "Head east for the next lesson")
	assert.True(t, company.HasClaimed(aria.UserId, 61))
	assert.True(t, company.HasClaimed(aria.UserId, 62))

	// Formation: one in the front row, one behind.
	got = run(aria, "east", "")
	require.Equal(t, 902, template(aria))
	assert.Contains(t, got, "stage 3 of 6")
	run(aria, "formation", "move #1 1 2")
	assert.Equal(t, StageFormation, stageOf(aria), "no one behind yet")

	// Log out mid-lesson, as the engine does (into the Void, saved), and
	// back in: the course resumes in fresh copies at this stage.
	events.AddToQueue(events.PlayerDespawn{UserId: aria.UserId, RoomId: aria.Character.RoomId, Username: aria.Username, CharacterName: aria.Character.Name})
	events.ProcessEvents()
	if room := rooms.LoadRoom(aria.Character.RoomId); room != nil {
		room.RemovePlayer(aria.UserId)
	}
	aria.Character.RoomId = -1
	require.NoError(t, users.SaveUser(*aria))
	data, err := os.ReadFile(filepath.Join(dataDir, "users", "7.yaml"))
	require.NoError(t, err)
	reloaded := users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	reloaded.UserId = aria.UserId
	require.NoError(t, reloaded.Character.Validate(true), "as users.LoadUser does")
	aria = &reloaded
	users.SetTestUser(aria)
	require.Equal(t, StageFormation, stageOf(aria), "progress is in the user file")
	events.AddToQueue(events.PlayerSpawn{UserId: aria.UserId, RoomId: -1, Username: aria.Username, CharacterName: aria.Character.Name})
	got = text(aria)
	assert.Equal(t, 902, template(aria), "back at the Drill Ground")
	assert.Contains(t, got, "back where your training left off")
	assert.Contains(t, got, "stage 3 of 6")
	assert.Equal(t, -1, aria.Character.RoomIdOnReset)
	here := aria.Character.RoomId
	for _, key := range []int{1, 2} {
		instanceID, ok := company.InstanceFor(aria.UserId, key)
		require.True(t, ok, "companion %d restored", key)
		assert.Equal(t, here, mobs.GetInstance(instanceID).Character.RoomId, "companion %d came along", key)
	}
	// The ways back to the passed rooms, and on, are open again.
	run(aria, "west", "")
	assert.Equal(t, 901, template(aria))
	run(aria, "east", "")
	assert.Equal(t, 902, template(aria))

	got = run(aria, "formation", "move #2 2 2")
	assert.Equal(t, StageSurvival, stageOf(aria))

	// Survival (27b): walking into the Weather Yard hands over what the
	// pack lacks, once. The real eat and drink, and the inspections.
	got = run(aria, "east", "")
	require.Equal(t, 904, template(aria))
	assert.Contains(t, got, "stage 4 of 6: Survival")
	assert.Contains(t, got, "cheese sandwich")
	assert.Contains(t, got, "waterskin")
	assert.True(t, progressOf(aria.Character).Supplied)
	run(aria, "eat", "nothing-here")
	assert.False(t, progressOf(aria.Character).Seen[seenFed], "a failed eat doesn't count")
	run(aria, "eat", "sandwich")
	run(aria, "drink", "waterskin")
	p := progressOf(aria.Character)
	assert.True(t, p.Seen[seenFed])
	assert.True(t, p.Seen[seenWatered])
	for _, cmd := range []string{"weather", "temperature", "strain"} {
		require.True(t, usercommands.IsRegistered(cmd), cmd)
		run(aria, cmd, "")
		assert.Equal(t, StageSurvival, stageOf(aria), "after "+cmd)
	}
	got = run(aria, "cargo", "")
	assert.Equal(t, StageCamp, stageOf(aria))
	assert.Contains(t, got, "Head east for the next lesson: Camp")
	run(aria, "west", "")
	run(aria, "east", "")
	assert.Equal(t, 904, template(aria))
	assert.Equal(t, 1, countItem(aria, 30004), "walking back in gives no more")

	// Camp: a real camp and rest in the Campground copy.
	got = run(aria, "east", "")
	require.Equal(t, 905, template(aria))
	assert.Contains(t, got, "stage 5 of 6: Camp")
	assert.Contains(t, run(aria, "camp", ""), "You make camp here.")
	assert.Contains(t, run(aria, "camp", "fire"), "campfire")
	assert.Contains(t, run(aria, "camp", "rest"), "You settle in by the fire to rest.")
	activity, ok := camping.LeaderRest(aria.UserId)
	require.True(t, ok)
	assert.True(t, activity.Resting, "a real, running rest")
	campRoom := aria.Character.RoomId
	run(aria, "east", "")
	assert.Equal(t, campRoom, aria.Character.RoomId, "no way on, and resting holds the company")
	assert.Equal(t, StageCamp, stageOf(aria))
	// The rest takes a real minute; camping's own tests finish it. Here
	// the Rested it grants at the end is given directly, as its grant does.
	require.NoError(t, aria.Character.AddBuff(1033, false, 100))
	got = run(aria, "look", "")
	assert.Equal(t, StageDeparture, stageOf(aria), "Rested passes Camp")
	assert.Contains(t, got, "Head east for the next lesson: Departure")
	_, ok = camping.LeaderRest(aria.UserId)
	assert.False(t, ok, "the course camp is struck")

	// Departure: out through the gate, with one cap.
	run(aria, "east", "")
	require.Equal(t, 903, template(aria))
	assert.Zero(t, caps(aria))
	got = run(aria, "gate", "")
	assert.Equal(t, 1, aria.Character.RoomId, "the start room")
	assert.Equal(t, stateGraduated, progressOf(aria.Character).State)
	assert.Equal(t, 1, caps(aria))
	assert.Contains(t, got, "You have finished your training")
	assert.Equal(t, 0, aria.Character.RoomIdOnReset)
	for _, key := range []int{1, 2} {
		instanceID, ok := company.InstanceFor(aria.UserId, key)
		require.True(t, ok)
		assert.Equal(t, 1, mobs.GetInstance(instanceID).Character.RoomId, "companion %d left the course too", key)
	}

	// A second pass gives none: `start` again goes straight out.
	if room := rooms.LoadRoom(1); room != nil {
		room.RemovePlayer(aria.UserId)
	}
	aria.Character.RoomId = -1
	assert.True(t, module.Begin(aria.UserId))
	events.ProcessEvents()
	assert.Equal(t, 1, aria.Character.RoomId)
	assert.Equal(t, 1, caps(aria), "no second cap")
	assert.Equal(t, stateGraduated, progressOf(aria.Character).State)

	// A second player skips mid-rest: out to the start room with nothing,
	// and no camp left in the course.
	bram := newPlayer(8, "Bram")
	create(bram)
	text(bram)
	require.Equal(t, 900, template(bram))
	assert.NotEqual(t, before, bram.Character.RoomId, "their own copy")
	p = progressOf(bram.Character)
	p.Stage = StageCamp
	p.save(bram.Character)
	require.True(t, module.Begin(bram.UserId), "carries on at the saved stage")
	events.ProcessEvents()
	require.Equal(t, 905, template(bram))
	run(bram, "camp", "")
	run(bram, "camp", "fire")
	run(bram, "camp", "rest")
	activity, ok = camping.LeaderRest(bram.UserId)
	require.True(t, ok)
	require.True(t, activity.Resting)
	assert.Contains(t, run(bram, "tutorial", "skip"), "tutorial skip yes")
	assert.Equal(t, 905, template(bram), "asks first")
	run(bram, "tutorial", "skip yes")
	assert.Equal(t, 1, bram.Character.RoomId)
	assert.Equal(t, stateSkipped, progressOf(bram.Character).State)
	assert.Zero(t, caps(bram))
	_, ok = camping.LeaderRest(bram.UserId)
	assert.False(t, ok, "the rest is abandoned with the course")
	assert.Contains(t, run(bram, "camp", ""), "There is nowhere here to make camp.", "not \"You already have a camp\"")

	assert.Equal(t, turn, util.GetTurnCount(), "the clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
}

func countItem(u *users.UserRecord, itemID int) int {
	n := 0
	for _, itm := range u.Character.GetAllBackpackItems() {
		if itm.ItemId == itemID {
			n++
		}
	}
	return n
}
