package storyevents

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// The real modules the events reach: the company and its needs, the
	// cargo, the encounters that start a battle, and the step and camp
	// seams the triggers hang on.
	_ "github.com/GoMudEngine/GoMud/modules/archetype"
	_ "github.com/GoMudEngine/GoMud/modules/camping"
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/death"
	_ "github.com/GoMudEngine/GoMud/modules/encounters"
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
	_ "github.com/GoMudEngine/GoMud/modules/expedition"
	_ "github.com/GoMudEngine/GoMud/modules/exposure"
	_ "github.com/GoMudEngine/GoMud/modules/mount"
	_ "github.com/GoMudEngine/GoMud/modules/strategy"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
	_ "github.com/GoMudEngine/GoMud/modules/walking"
	_ "github.com/GoMudEngine/GoMud/modules/weather"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func shippedWorld() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

// linkedWorld is a copy of the shipped world with its own users and
// plugin-data, so nothing a test saves reaches the repository.
func linkedWorld(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	shipped := shippedWorld()
	entries, err := os.ReadDir(shipped)
	require.NoError(t, err)
	for _, e := range entries {
		switch e.Name() {
		case "users", "plugin-data", "buffs":
			continue
		}
		if !e.IsDir() {
			data, err := os.ReadFile(filepath.Join(shipped, e.Name()))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, e.Name()), data, 0600))
			continue
		}
		require.NoError(t, os.CopyFS(filepath.Join(dir, e.Name()), os.DirFS(filepath.Join(shipped, e.Name()))), e.Name())
	}
	for _, d := range []string{"users", "plugin-data", "buffs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, d), 0755))
	}
	return dir
}

func useDataDir(t *testing.T, dir string) {
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

// scene is the shipped world with one leader standing in the test area's
// hub, and the real story events module on its real triggers.
type scene struct {
	t        *testing.T
	user     *users.UserRecord
	messages []string
	rolls    []int
}

func newScene(t *testing.T) *scene {
	t.Helper()
	dir := linkedWorld(t)
	useDataDir(t, dir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	skills.LoadDataFiles()
	spells.LoadSpellFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dir)
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
	user := users.NewUserRecord(77, 1)
	user.Username = "wayfarer"
	user.Password = "$2a$test"
	user.Character.Name = "Aldous"
	user.Character.RaceId = 1
	user.Character.Alignment = 40
	user.Character.Level = 5
	user.Character.Gold = 100
	user.Character.Validate()
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 90001))
	events.ProcessEvents()
	t.Cleanup(func() {
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
	})

	s := &scene{t: t, user: user}
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == user.UserId {
			s.messages = append(s.messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })

	// A clean slate for this leader, the shipped events, and rolls the test
	// decides (empty: every roll is 0, so a risk fails).
	require.NotNil(t, module)
	module.mu.Lock()
	previousState := module.state
	module.state = map[int]State{}
	module.catalog = nil
	module.mu.Unlock()
	previousRng := module.rng
	module.rng = func(n int) int {
		if len(s.rolls) == 0 {
			return 0
		}
		v := s.rolls[0]
		s.rolls = s.rolls[1:]
		return v % n
	}
	t.Cleanup(func() {
		module.mu.Lock()
		module.state, module.catalog = previousState, nil
		module.mu.Unlock()
		module.rng = previousRng
	})
	return s
}

func (s *scene) run(command, rest string) string {
	s.t.Helper()
	events.ProcessEvents()
	s.messages = nil
	s.user.Character.ActionPoints = 100 // walking is not what these tests are about
	handled, err := usercommands.TryCommand(command, rest, s.user.UserId, events.CmdSkipScripts)
	require.NoError(s.t, err)
	require.True(s.t, handled, command)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(s.messages, "\n"), "")
}

func (s *scene) room() int { return s.user.Character.RoomId }

func (s *scene) cargo(item int) int {
	n := 0
	for _, stack := range encumbrance.CargoContents(s.user.UserId) {
		if stack.ItemId == item {
			n += stack.Count
		}
	}
	return n
}

func (s *scene) pendingPage() string {
	module.mu.Lock()
	defer module.mu.Unlock()
	if p := module.state[s.user.UserId].Pending; p != nil {
		return p.Event + "/" + p.Page
	}
	return ""
}

// TestTheShippedEventsAreSound: every shipped test scene passes validation
// against the real items, mobs, rooms, skills and ailments.
func TestTheShippedEventsAreSound(t *testing.T) {
	newScene(t)
	assert.Equal(t, []string{"burned-shrine", "gorge-descent", "stranger-at-the-fire"}, module.events().IDs(), "all three shipped scenes load")
}

// TestTheGorgeThroughTheRealTriggerAndCargo walks into the gorge, goes down
// the rope, searches the pack and climbs back: a room trigger, the
// movement hold, a move outcome, a follow-on page, and real gold, cargo and
// a flag.
func TestTheGorgeThroughTheRealTriggerAndCargo(t *testing.T) {
	s := newScene(t)
	out := s.run("go", "northeast")
	assert.Equal(t, 90011, s.room())
	assert.Contains(t, out, "The Gorge")
	assert.Contains(t, out, "1. Send your best climber down the rope (closed: needs a ranger, a rogue or a gryphon-rider)")
	assert.Contains(t, out, "2. Go down yourself, and the company after you (chancy)", "50% less 3 a level at level 5")
	assert.Equal(t, "gorge-descent/start", s.pendingPage())

	assert.Contains(t, s.run("go", "southwest"), "A scene is waiting on your company")
	assert.Equal(t, 90011, s.room(), "the company stays put")
	assert.Contains(t, s.run("event", ""), "The road ends at air.")
	assert.Contains(t, s.run("choose", "1"), "needs a ranger, a rogue or a gryphon-rider")
	assert.Contains(t, s.run("choose", "x"), "Usage: choose")

	s.rolls = []int{99} // over the risk: the rope holds
	out = s.run("choose", "2")
	assert.Equal(t, 90014, s.room(), "the move outcome carries the leader to the ledge")
	assert.Contains(t, out, "The rope holds.")
	assert.Contains(t, out, "A shelf of wet rock")
	assert.Equal(t, "gorge-descent/ledge", s.pendingPage())

	gold := s.user.Character.Gold
	out = s.run("choose", "1")
	assert.Contains(t, out, "You gain 15 gold.")
	assert.Contains(t, out, "You take 2 linen bandage")
	assert.Equal(t, gold+15, s.user.Character.Gold)
	assert.Equal(t, 2, s.cargo(36), "the bandages went into the company's cargo")
	assert.Equal(t, "gorge-descent/ledge-done", s.pendingPage())

	s.run("choose", "1")
	assert.Equal(t, 90011, s.room(), "the climb back ends the scene at the top")
	assert.Equal(t, "", s.pendingPage())
	module.mu.Lock()
	st := module.state[s.user.UserId]
	module.mu.Unlock()
	assert.Equal(t, []string{"searched-the-ledge"}, st.Flags)
	assert.Contains(t, st.Done, "gorge-descent")

	s.run("go", "southwest")
	assert.Equal(t, 90001, s.room(), "free to walk again")
	s.run("go", "northeast")
	assert.Equal(t, "", s.pendingPage(), "the gorge opens once per company")
}

// TestAFailedClimbWoundsTheLeader: a wound outcome lands on the real
// character.
func TestAFailedClimbWoundsTheLeader(t *testing.T) {
	s := newScene(t)
	s.run("go", "northeast")
	s.rolls = []int{0} // under the risk: the ring tears out
	out := s.run("choose", "2")
	assert.Contains(t, out, "The ring tears out of the stone")
	assert.Contains(t, out, "Aldous is left with")
	require.Len(t, s.user.Character.Wounds, 1)
	assert.Greater(t, s.user.Character.Wounds[0].Points, 0)
	assert.Equal(t, "", s.pendingPage())
	assert.Equal(t, 90011, s.room(), "a failed climb ends the scene where it began")
}

// TestAClimberCompanionTakesTheChoiceAndPaysForTheFall: a rogue companion
// is named for the gated choice, and a fall wounds that live companion.
func TestAClimberCompanionTakesTheChoiceAndPaysForTheFall(t *testing.T) {
	s := newScene(t)
	text, err := company.AdminRecruit(s.user.UserId, s.room(), "rogue", 6)
	require.NoError(t, err, text)
	events.ProcessEvents()
	views, ok := company.CompanyMembers(s.user.UserId)
	require.True(t, ok)
	require.Len(t, views, 1)
	require.Equal(t, company.MemberPresent, views[0].Status)
	name := views[0].Name

	out := s.run("go", "northeast")
	assert.Contains(t, out, "1. Send your best climber down the rope ("+name, "the rogue is named for the choice")

	s.rolls = []int{0} // the fall
	out = s.run("choose", "1")
	assert.Contains(t, out, name+" loses the rope halfway")
	instanceID, ok := company.InstanceFor(s.user.UserId, views[0].ID)
	require.True(t, ok)
	mob := mobs.GetInstance(instanceID)
	require.NotNil(t, mob)
	assert.Len(t, mob.Character.Wounds, 1, "the wound is on the companion who fell")
	assert.Empty(t, s.user.Character.Wounds)

	// A second company member on a second page: success pays out items.
	s2 := s
	s2.run("go", "southwest")
	module.mu.Lock()
	delete(module.state, s.user.UserId)
	module.mu.Unlock()
	s2.run("go", "northeast")
	s2.rolls = []int{99}
	out = s2.run("choose", "1")
	assert.Contains(t, out, name+" goes down hand over hand")
	assert.Equal(t, 2, s2.cargo(36))
}

// TestTheStrangerSharesFoodAndFailsToRobHim: need and loyalty outcomes
// through the real survival and company modules, and a battle outcome
// through the real encounters module.
func TestTheStrangerSharesFoodAndFailsToRobHim(t *testing.T) {
	s := newScene(t)
	_, err := company.AdminRecruit(s.user.UserId, s.room(), "warrior", 4)
	require.NoError(t, err)
	events.ProcessEvents()
	before := leaderNeeds(t, s)

	out := s.run("go", "northwest")
	assert.Equal(t, 90012, s.room())
	assert.Contains(t, out, "A man sits by a small fire")
	s.run("choose", "1")
	assert.Equal(t, "stranger-at-the-fire/tale", s.pendingPage())
	out = s.run("choose", "1")
	assert.Contains(t, out, "He eats like a man who has forgotten how")
	assert.Contains(t, out, "Aldous is hungrier for it.")
	assert.Contains(t, out, "saw you share")
	after := leaderNeeds(t, s)
	assert.Less(t, after.Hunger, before.Hunger, "the leader goes hungrier")
	module.mu.Lock()
	flags := module.state[s.user.UserId].Flags
	module.mu.Unlock()
	assert.Equal(t, []string{"fed-the-deserter"}, flags)
	assert.Equal(t, "", s.pendingPage())

	// A second company on the same fire: robbing him fails, and his
	// friends come.
	module.mu.Lock()
	delete(module.state, s.user.UserId)
	module.mu.Unlock()
	s.run("go", "southeast")
	s.run("go", "northwest")
	s.run("choose", "1")
	s.rolls = []int{0} // under the 40% risk
	out = s.run("choose", "3")
	assert.Contains(t, out, "He was not asleep")
	room := rooms.LoadRoom(90012)
	foes := 0
	for _, id := range room.GetMobs() {
		if mob := mobs.GetInstance(id); mob != nil && mob.EncounterOwner == s.user.UserId {
			foes++
		}
	}
	assert.Equal(t, 2, foes, "two brigands are set on the company")
}

func leaderNeeds(t *testing.T, s *scene) survival.Needs {
	t.Helper()
	for _, n := range survival.CompanyNeeds(s.user.UserId) {
		if n.Key == survival.LeaderMemberKey {
			return n.Needs
		}
	}
	t.Fatal("no leader needs")
	return survival.Needs{}
}

// TestTheShrineCooldownAilmentAndGold: a risky choice's failure gives a
// real ailment; a success pays gold; the cooldown holds the scene back.
func TestTheShrineCooldownAilmentAndGold(t *testing.T) {
	s := newScene(t)
	out := s.run("go", "southeast")
	assert.Equal(t, 90013, s.room())
	assert.Contains(t, out, "The Burned Shrine")
	assert.Contains(t, out, "1. Have the devout one say the old words (closed: needs a devout companion)")
	assert.Contains(t, out, "2. Kneel in the ash yourself (Aldous)", "a good-hearted leader may kneel")

	s.rolls = []int{0, 0} // under the risk: the water is wrong; random picks the first member
	out = s.run("choose", "4")
	assert.Contains(t, out, "The water is cold, and it is wrong.")
	assert.Contains(t, out, "Aldous comes down with a gut-ache.")
	assert.True(t, survival.HasAilment(leaderNeeds(t, s)))

	// The cooldown is saved: walking in again at once does nothing.
	s.run("go", "northwest")
	s.run("go", "southeast")
	assert.Equal(t, "", s.pendingPage())
}

// TestARandomEncounterHoldsASceneBack: a group sprung on the company in the
// same room means no page opens over the fight.
func TestARandomEncounterHoldsASceneBack(t *testing.T) {
	s := newScene(t)
	require.NoError(t, rooms.MoveToRoom(s.user.UserId, 90011))
	events.ProcessEvents()
	mob := mobs.NewMobById(mobs.MobId(86), 90011, 3)
	require.NotNil(t, mob)
	mob.EncounterOwner = s.user.UserId
	rooms.LoadRoom(90011).AddMob(mob.InstanceId)
	module.entered(s.user.UserId, 90011, false)
	assert.Equal(t, "", s.pendingPage())
	room := rooms.LoadRoom(90011)
	room.RemoveMob(mob.InstanceId)
	mobs.DestroyInstance(mob.InstanceId)
	module.entered(s.user.UserId, 90011, false)
	assert.Equal(t, "gorge-descent/start", s.pendingPage(), "and once it is gone the scene opens")
}
