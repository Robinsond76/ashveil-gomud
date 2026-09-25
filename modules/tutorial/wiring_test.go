package tutorial

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
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
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	_ "github.com/GoMudEngine/GoMud/modules/standing"
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
		"rooms/tutorial/906.yaml",
		"rooms/tutorial/907.yaml",
		"mobs/tutorial/69-corvin_blackthorn.yaml",
		"races/19-dummy.yaml",
		"mobs/tutorial/67-straw_footman.yaml",
		"mobs/tutorial/68-straw_archer.yaml",
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
	// Attack messages, for the practice fight (27c).
	messages, err := filepath.Glob(filepath.Join(shipped, "combat-messages", "*.yaml"))
	require.NoError(t, err)
	for _, path := range messages {
		copies = append(copies, filepath.Join("combat-messages", filepath.Base(path)))
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
		"SpecialRooms.TutorialRooms": []any{"900", "901", "902", "903", "904", "905", "906", "907"},
	})
	writeTutorialWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	for _, id := range []int{900, 901, 902, 903, 904, 905, 906, 907, 1} {
		require.NotNil(t, rooms.LoadRoom(id), "room %d", id)
	}
	require.Equal(t, []int{900, 901, 902, 903, 904, 905, 906, 907}, configuredRooms())

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

	// Phase 27d: the Tutorial GMCP package, as the gmcp module sends it.
	panels := map[int][]map[string]any{}
	gid := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out, ok := e.(gmcp.GMCPOut); ok && out.Module == "Tutorial" {
			var body map[string]any
			if raw, ok := out.Payload.([]byte); ok {
				_ = json.Unmarshal(raw, &body)
			}
			panels[out.UserId] = append(panels[out.UserId], body)
		}
		return events.Cancel // no connection to deliver to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, gid) })
	lastPanel := func(u *users.UserRecord) map[string]any {
		events.ProcessEvents()
		list := panels[u.UserId]
		if len(list) == 0 {
			return nil
		}
		return list[len(list)-1]
	}

	// Creation hands the new character to the course: their own copy of
	// the Waking Hall, with the first lesson shown.
	aria := newPlayer(7, "Aria")
	create(aria)
	got := text(aria)
	assert.Equal(t, 900, template(aria))
	assert.NotEqual(t, 900, aria.Character.RoomId, "a copy, not the template")
	assert.Contains(t, got, "A long, low hall")
	assert.Contains(t, got, "stage 1 of 8")
	assert.Equal(t, StageCharacter, stageOf(aria))
	assert.Contains(t, run(aria, "tutorial", ""), "Goal:")
	panel := lastPanel(aria)
	require.NotNil(t, panel, "the panel is sent")
	assert.Equal(t, "Your character", panel["title"])
	assert.EqualValues(t, 1, panel["stage"])
	assert.EqualValues(t, 8, panel["stages"])

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
	panel = lastPanel(aria)
	assert.Equal(t, "Your company", panel["title"], "the panel follows the course")
	assert.Equal(t, []any{map[string]any{"label": "two companions (0 of 2)", "done": false}}, panel["checklist"])
	sentSoFar := len(panels[aria.UserId])
	run(aria, "say", "still here")
	lastPanel(aria)
	assert.Len(t, panels[aria.UserId], sentSoFar, "nothing resent without a change")

	// Company: recruit both tutorial candidates in the Muster Yard copy.
	got = run(aria, "east", "")
	require.Equal(t, 901, template(aria))
	assert.Contains(t, got, "stage 2 of 8")
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
	assert.Contains(t, got, "stage 3 of 8")
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
	assert.Contains(t, got, "stage 3 of 8")
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
	assert.Contains(t, got, "stage 4 of 8: Survival")
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
	assert.Contains(t, got, "stage 5 of 8: Camp")
	assert.Contains(t, run(aria, "camp", ""), "You make camp here.")
	assert.Contains(t, run(aria, "camp", "fire"), "campfire")
	assert.Contains(t, run(aria, "camp", "rest"), "You settle in by the fire to rest.")
	activity, ok := camping.LeaderRest(aria.UserId)
	require.True(t, ok)
	assert.True(t, activity.Resting, "a real, running rest")
	campRoom := aria.Character.RoomId
	assert.Contains(t, run(aria, "west", ""), "resting at camp")
	assert.Equal(t, campRoom, aria.Character.RoomId, "resting holds the company")
	assert.Equal(t, StageCamp, stageOf(aria))
	// The rest takes a real minute; camping's own tests finish it. Here
	// the Rested it grants at the end is given directly, as its grant does.
	require.NoError(t, aria.Character.AddBuff(1033, false, 100))
	got = run(aria, "look", "")
	assert.Equal(t, StageCombat, stageOf(aria), "Rested passes Camp")
	assert.Contains(t, got, "Head east for the next lesson: Combat")
	_, ok = camping.LeaderRest(aria.UserId)
	assert.False(t, ok, "the course camp is struck")

	// A camp made after the lesson (the Campground still allows one) is
	// struck when the player walks out.
	run(aria, "camp", "")
	_, ok = camping.LeaderRest(aria.UserId)
	require.True(t, ok)

	// Combat (27c): the squad stands in the player's own Practice Yard as
	// one party, footmen in front, the archer behind.
	got = run(aria, "east", "")
	require.Equal(t, 906, template(aria))
	assert.Contains(t, got, "stage 6 of 8: Combat")
	yard := rooms.LoadRoom(aria.Character.RoomId)
	require.NotNil(t, yard)
	squad := map[int]string{}
	var summaries []mobparty.MobSummary
	for _, id := range yard.GetMobs() {
		mob := mobs.GetInstance(id)
		require.NotNil(t, mob)
		if mob.Character.IsCharmed() {
			continue // the company
		}
		require.True(t, mob.Practice, mob.Character.Name)
		squad[id] = mob.Character.Name
		summaries = append(summaries, mobparty.MobSummary{InstanceId: id, Groups: mob.Groups, EHP: float64(mob.Character.HealthMax.Value)})
	}
	require.Len(t, squad, 4)
	parties := mobparty.Assemble(summaries)
	require.Len(t, parties, 1, "one party")
	archer := 0
	for id, name := range squad {
		row, _, found := parties[0].Formation.Find(mobparty.MemberKeyFor(id))
		require.True(t, found)
		if name == "straw archer" {
			archer = id
			assert.Equal(t, 1, row, "the archer stands behind")
		} else {
			assert.Equal(t, 0, row, "a footman in front")
		}
	}
	require.NotZero(t, archer)

	// The real combat round: the test stands in for the world loop, which
	// runs queued mob commands (a beaten foe's "suicide").
	mid := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId > 0 {
			cmd, rest, _ := strings.Cut(in.InputText, " ")
			_, _ = mobcommands.TryCommand(strings.ToLower(cmd), rest, in.MobInstanceId)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, mid) })
	fightRound := func(n uint64) {
		hooks.DoCombat(events.NewRound{RoundNumber: n})
		events.ProcessEvents()
		usercommands.TryCommand("look", "", aria.UserId, events.CmdSkipScripts|events.CmdSecretly) // a refresh, as each round gives
		events.ProcessEvents()
	}
	xpBefore, goldBefore := aria.Character.Experience, aria.Character.Gold

	// Aria takes her own place, behind, in the middle column.
	run(aria, "formation", "move me 3 2")
	f, ok := company.FormationFor(aria.UserId)
	require.True(t, ok)
	row, col, found := f.Find(company.LeaderMemberKey)
	require.True(t, found)
	require.Equal(t, [2]int{2, 1}, [2]int{row, col})

	// An attack at the archer is caught by a footman in front: Aria's own
	// blows land on a footman, never the archer.
	run(aria, "attack", "archer")
	require.NotNil(t, aria.Character.Aggro)
	require.Equal(t, archer, aria.Character.Aggro.MobInstanceId)
	var r uint64
	// Attack messages vary ("You hit", "You punch", ...); a line of Aria's
	// own naming a foe is her blow.
	footmanBlow := regexp.MustCompile(`(?m)^Your? [^\n]*straw footman`)
	archerBlow := regexp.MustCompile(`(?m)^Your? [^\n]*straw archer`)
	seen := ""
	for r = 1; r < 200 && !footmanBlow.MatchString(seen); r++ {
		fightRound(r)
		seen += text(aria)
	}
	require.Regexp(t, footmanBlow, seen, "a blow landed")
	assert.NotRegexp(t, archerBlow, seen, "the archer is shielded")
	assert.Equal(t, "straw footman", squad[aria.Character.Aggro.MobInstanceId], "her aim moves to the footman who caught it")

	// Re-targeting (11b): a foe Aria is aiming at falls to another blow
	// (a companion's, here beaten directly); next round she turns to a
	// standing foe she can reach, without another command.
	run(aria, "attack", "footman")
	aimed := aria.Character.Aggro.MobInstanceId
	require.NotEqual(t, archer, aimed)
	beatenMob := mobs.GetInstance(aimed)
	require.NotNil(t, beatenMob)
	_, err = mobcommands.Suicide("", beatenMob, yard)
	require.NoError(t, err)
	events.ProcessEvents()
	fightRound(r)
	r++
	require.NotNil(t, aria.Character.Aggro, "not dropped")
	assert.NotEqual(t, aimed, aria.Character.Aggro.MobInstanceId, "turned to another foe")
	assert.NotNil(t, mobs.GetInstance(aria.Character.Aggro.MobInstanceId), "a standing one")
	assert.Contains(t, squad, aria.Character.Aggro.MobInstanceId)

	// Fight on until the squad is beaten, attacking again whenever a foe
	// falls to Aria's own blow (a killing blow ends the attacker's aim;
	// others aiming at it are re-targeted, 11b).
	for ; r < 5000 && stageOf(aria) == StageCombat; r++ {
		if aria.Character.Aggro == nil {
			next := "archer"
			for id, name := range squad {
				if mobs.GetInstance(id) != nil && name == "straw footman" {
					next = "footman"
				}
			}
			run(aria, "attack", next)
		}
		fightRound(r)
	}
	require.Equal(t, StageAlignment, stageOf(aria), "the squad is beaten")
	for id := range squad {
		assert.Nil(t, mobs.GetInstance(id), "%s left the field", squad[id])
	}
	assert.Equal(t, xpBefore, aria.Character.Experience, "no XP from practice")
	assert.Equal(t, goldBefore, aria.Character.Gold)
	assert.Empty(t, yard.Items, "nothing dropped")
	assert.Zero(t, yard.Gold)
	assert.Empty(t, yard.Corpses)

	// Alignment (27d): inspections, one a subcommand, and the outlaw
	// weighed against the company and refused by the real gate.
	got = run(aria, "east", "")
	require.Equal(t, 907, template(aria))
	assert.Contains(t, got, "stage 7 of 8: Alignment")
	run(aria, "company", "status")
	assert.False(t, progressOf(aria.Character).Seen["company alignment"], "another subcommand doesn't count")
	assert.Contains(t, run(aria, "company", "alignment"), "Company alignment")
	got = run(aria, "company", "inspect corvin")
	assert.Contains(t, got, "Corvin Blackthorn: alignment")
	assert.Contains(t, got, "They won't join a company so far from their ways.")
	aria.Character.Gold = 500
	assert.Contains(t, run(aria, "company", "recruit corvin"), "won't join a company", "the real gate refuses him")
	aria.Character.Gold = goldBefore
	assert.Equal(t, StageAlignment, stageOf(aria))
	got = run(aria, "standing", "")
	assert.Equal(t, StageDeparture, stageOf(aria))
	assert.Contains(t, got, "Head east for the next lesson: Departure")

	// Departure: out through the gate, with one cap.
	run(aria, "east", "")
	require.Equal(t, 903, template(aria))
	assert.Zero(t, caps(aria))
	got = run(aria, "gate", "")
	assert.Equal(t, 1, aria.Character.RoomId, "the start room")
	assert.Equal(t, stateGraduated, progressOf(aria.Character).State)
	assert.Equal(t, 1, caps(aria))
	assert.Contains(t, got, "You have finished your training")
	_, ok = camping.LeaderRest(aria.UserId)
	assert.False(t, ok, "no camp left in the course")
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
	restAt := func() {
		t.Helper()
		assert.Contains(t, run(bram, "camp", ""), "You make camp here.")
		run(bram, "camp", "fire")
		run(bram, "camp", "rest")
		activity, ok = camping.LeaderRest(bram.UserId)
		require.True(t, ok)
		require.True(t, activity.Resting)
	}
	restAt()

	// Logging out mid-rest strikes the camp; resuming, the rest is made
	// again in the new copies.
	events.AddToQueue(events.PlayerDespawn{UserId: bram.UserId, RoomId: bram.Character.RoomId, Username: bram.Username, CharacterName: bram.Character.Name})
	events.ProcessEvents()
	_, ok = camping.LeaderRest(bram.UserId)
	assert.False(t, ok, "struck on logout")
	if room := rooms.LoadRoom(bram.Character.RoomId); room != nil {
		room.RemovePlayer(bram.UserId)
	}
	bram.Character.RoomId = -1
	events.AddToQueue(events.PlayerSpawn{UserId: bram.UserId, RoomId: -1, Username: bram.Username, CharacterName: bram.Character.Name})
	events.ProcessEvents()
	require.Equal(t, 905, template(bram), "back at the Campground")
	text(bram)
	restAt()
	assert.Contains(t, run(bram, "tutorial", "skip"), "tutorial skip yes")
	assert.Equal(t, 905, template(bram), "asks first")
	require.NotEmpty(t, lastPanel(bram), "Bram's own panel")
	assert.Equal(t, "Camp", lastPanel(bram)["title"])
	run(bram, "tutorial", "skip yes")
	assert.Empty(t, lastPanel(bram), "{} once the course is done")
	assert.Equal(t, 1, bram.Character.RoomId)
	assert.Equal(t, stateSkipped, progressOf(bram.Character).State)
	assert.Zero(t, caps(bram))
	_, ok = camping.LeaderRest(bram.UserId)
	assert.False(t, ok, "the rest is abandoned with the course")
	assert.Contains(t, run(bram, "camp", "status"), "You have no camp.")

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
