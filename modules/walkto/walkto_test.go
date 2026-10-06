package walkto

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
	walkplan "github.com/GoMudEngine/GoMud/internal/walkto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

var tags = regexp.MustCompile(`<[^>]*>`)

var aliasesOnce sync.Once

func loadAliases(t *testing.T) {
	t.Helper()
	aliasesOnce.Do(func() {
		_, thisFile, _, _ := runtime.Caller(0)
		dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		keywords.LoadAliases()
	})
}

type stepRecorder struct{ steps [][2]int }

func (s *stepRecorder) Stepped(_, from, to int) { s.steps = append(s.steps, [2]int{from, to}) }

// world is a small valley, every room through the real room manager:
//
//	96001 - 96002 - 96003 - 96004 (Alder Inn)
//	                  |
//	                96005 (cellar, locked door down)      96006 (never visited)
type world struct {
	t      *testing.T
	m      *WalktoModule
	user   *users.UserRecord
	timers []func()
	said   []string
	steps  *stepRecorder

	battle bool
	summ   companyview.Summary
	hostle string
}

func newWorld(t *testing.T) *world {
	t.Helper()
	loadAliases(t)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "forest", Name: "Forest", Symbol: "f"})
	t.Cleanup(func() { rooms.RemoveTestBiome("forest") })

	mk := func(id int, title, legend string, exits map[string]exit.RoomExit) {
		r := &rooms.Room{RoomId: id, Zone: "Valley", Title: title, MapLegend: legend, Biome: "forest", Exits: exits}
		rooms.SetTestRoom(r)
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	mk(96001, "Trail Head", "", map[string]exit.RoomExit{"east": {RoomId: 96002}})
	mk(96002, "Green", "Village", map[string]exit.RoomExit{"west": {RoomId: 96001}, "east": {RoomId: 96003}, "north": {RoomId: 96006}})
	mk(96003, "Fork", "", map[string]exit.RoomExit{"west": {RoomId: 96002}, "east": {RoomId: 96004}, "down": {RoomId: 96005, Lock: gamelock.Lock{Difficulty: 3}}})
	mk(96004, "The Alder Inn", "Inn", map[string]exit.RoomExit{"west": {RoomId: 96003}})
	mk(96005, "Cellar", "", map[string]exit.RoomExit{"up": {RoomId: 96003}})
	mk(96006, "Hollow", "", map[string]exit.RoomExit{"south": {RoomId: 96002}})

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	walkplan.ResetForTest()
	t.Cleanup(walkplan.ResetForTest)

	w := &world{t: t, steps: &stepRecorder{}}
	w.user = users.NewUserRecord(7, 96001)
	w.user.Character.Name = "Hero"
	w.user.Username = "hero"
	w.user.Password = "$2a$test"
	w.user.Character.RoomId = 96001
	w.user.Character.ActionPoints = 1000
	w.user.Character.Validate()
	users.SetTestUser(w.user)
	for _, id := range []int{96001, 96002, 96003, 96004, 96005} {
		w.user.Character.MarkVisitedRoom(id, "Valley", nil)
	}
	walking.SetStepProvider(w.steps)
	t.Cleanup(func() { walking.SetStepProvider(nil) })

	// The registered listeners belong to the package's module, so the test
	// drives that one, with its seams swapped for fakes.
	w.m = module
	lookupUser, loadRoom, after, move := module.lookupUser, module.loadRoom, module.after, module.move
	inBattle, summary, hostileIn := module.inBattle, module.summary, module.hostileIn
	t.Cleanup(func() {
		module.lookupUser, module.loadRoom, module.after, module.move = lookupUser, loadRoom, after, move
		module.inBattle, module.summary, module.hostileIn = inBattle, summary, hostileIn
		module.issued, module.tried = map[int]string{}, map[int]tried{}
	})
	module.issued, module.tried = map[int]string{}, map[int]tried{}
	w.m.after = func(_ time.Duration, f func()) { w.timers = append(w.timers, f) }
	w.m.move = func(userID int, text string) {
		// What the world's input handler does with the queued line.
		events.AddToQueue(events.Input{UserId: userID, InputText: text})
		fields := strings.Fields(text)
		u := users.GetByUserId(userID)
		_, err := usercommands.Go(fields[1], u, rooms.LoadRoom(u.Character.RoomId), 0)
		require.NoError(t, err)
	}
	w.m.inBattle = func(*users.UserRecord) bool { return w.battle }
	w.m.summary = func(*users.UserRecord) companyview.Summary { return w.summ }
	w.m.hostileIn = func(*rooms.Room, int) string { return w.hostle }

	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		w.said = append(w.said, tags.ReplaceAllString(e.(events.Message).Text, ""))
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return w
}

// walk runs the command the way the player's input does.
func (w *world) walk(rest string) string {
	w.t.Helper()
	w.said = nil
	handled, err := w.m.command(rest, w.user, rooms.LoadRoom(w.user.Character.RoomId), 0)
	require.NoError(w.t, err)
	require.True(w.t, handled)
	events.ProcessEvents()
	return strings.Join(w.said, "\n")
}

// tick fires the next step timer, as the real timer's queued event does.
func (w *world) tick() string {
	w.t.Helper()
	require.NotEmpty(w.t, w.timers, "a step timer is armed")
	f := w.timers[0]
	w.timers = w.timers[1:]
	w.said = nil
	f()
	events.ProcessEvents()
	return strings.Join(w.said, "\n")
}

func (w *world) typed(text string) {
	events.AddToQueue(events.Input{UserId: w.user.UserId, InputText: text})
	events.ProcessEvents()
}

func TestWalkToALandmarkWord(t *testing.T) {
	w := newWorld(t)
	out := w.walk("inn")
	assert.Contains(t, out, "You set out for The Alder Inn, 3 steps away")
	assert.Equal(t, 96002, w.user.Character.RoomId, "the first step goes at once, through Go")
	assert.True(t, w.m.Active(7))

	w.tick()
	assert.Equal(t, 96003, w.user.Character.RoomId)
	assert.True(t, w.m.Active(7))
	out = w.tick()
	assert.Equal(t, 96004, w.user.Character.RoomId)
	assert.Contains(t, out, "You arrive at The Alder Inn.", "arrival is told when the last step lands, not a delay later")
	assert.False(t, w.m.Active(7))
	assert.Equal(t, [][2]int{{96001, 96002}, {96002, 96003}, {96003, 96004}}, w.steps.steps,
		"every step went through the Go command's walking-strain hook, like a typed move")
}

func TestWalkToARoomNumberAndStatus(t *testing.T) {
	w := newWorld(t)
	w.walk("96004")
	out := w.walk("")
	assert.Contains(t, out, "You are walking to The Alder Inn, 2 steps to go")
	w.walk("stop")
	assert.False(t, w.m.Active(7))
	assert.Contains(t, w.walk("stop"), "You are not walking anywhere.")
	assert.Contains(t, w.walk(""), "Usage:")
}

func TestWalkNeverAdvancesTheGameClock(t *testing.T) {
	w := newWorld(t)
	before, round := gametime.GetDate(), util.GetRoundCount()
	w.walk("inn")
	for len(w.timers) > 0 {
		w.tick()
	}
	assert.Equal(t, before, gametime.GetDate())
	assert.Equal(t, round, util.GetRoundCount())
}

func TestWalkRefusals(t *testing.T) {
	w := newWorld(t)
	assert.Contains(t, w.walk("hollow"), "don't know a place", "an unvisited room is not a place you know")
	assert.Contains(t, w.walk("96006"), "haven't been there")
	assert.Contains(t, w.walk("96005"), "locked door is in the way")
	assert.Contains(t, w.walk("96001"), "already there")
	assert.False(t, w.m.Active(7))
	assert.Equal(t, 96001, w.user.Character.RoomId)

	w.battle = true
	assert.Contains(t, w.walk("inn"), "middle of a fight")
	w.battle = false

	w.user.Character.Aggro = &characters.Aggro{}
	assert.Contains(t, w.walk("inn"), "middle of a fight")
	w.user.Character.Aggro = nil

	p := parties.New(99)
	p.InvitePlayer(7)
	p.AcceptInvite(7)
	t.Cleanup(func() { p.Disband() })
	assert.Contains(t, w.walk("inn"), "Only your party's leader")
}

func TestWalkThroughALockNeedsTheKey(t *testing.T) {
	w := newWorld(t)
	w.user.Character.SetKey("key-96003-down", "A")
	out := w.walk("96005")
	assert.Contains(t, out, "You set out for Cellar")
}

func TestTypedCommandStopsTheWalkButLookDoesNot(t *testing.T) {
	w := newWorld(t)
	w.walk("inn")
	w.typed("look")
	w.typed("map")
	w.typed("walkto")
	assert.True(t, w.m.Active(7), "look, map and walkto keep it going")

	w.said = nil
	w.typed("say hello")
	assert.False(t, w.m.Active(7))
	assert.Contains(t, strings.Join(w.said, ""), "You stop walking.")

	// the timer armed before is stale and does nothing
	w.tick()
	assert.Equal(t, 96002, w.user.Character.RoomId)
}

func TestTheWalksOwnMovePassesTheInputListener(t *testing.T) {
	w := newWorld(t)
	w.walk("inn")
	require.True(t, w.m.Active(7))
	w.tick()
	assert.True(t, w.m.Active(7), "its own go west/east lines never cancel it")
	assert.Equal(t, 96003, w.user.Character.RoomId)
}

func TestWalkStopsForTrouble(t *testing.T) {
	cases := []struct {
		name string
		do   func(w *world)
		want string
	}{
		{"fight", func(w *world) { w.battle = true }, "A fight breaks out"},
		{"aggro", func(w *world) { w.user.Character.Aggro = &characters.Aggro{} }, "A fight breaks out"},
		{"hostile", func(w *world) { w.hostle = "a wolf" }, "a wolf is here and means trouble"},
		{"moved elsewhere", func(w *world) { require.NoError(w.t, rooms.MoveToRoom(7, 96006)) }, "no longer on your route"},
		{"locked ahead", func(w *world) {
			w.hostle = ""
			room := rooms.LoadRoom(96002)
			e := room.Exits["east"]
			e.Lock = gamelock.Lock{Difficulty: 4}
			e.Lock.SetLocked()
			room.Exits["east"] = e
		}, "door ahead is locked"},
		{"exit changed", func(w *world) { delete(rooms.LoadRoom(96002).Exits, "east") }, "way ahead has changed"},
		{"hungry", func(w *world) { w.summ.Leader.Hunger = companyview.Need{Known: true, Value: 20, Label: "Hungry"} }, "You are hungry"},
		{"exhausted", func(w *world) { w.summ.Leader.Fatigue = companyview.Need{Known: true, Value: 0, Label: "Collapsed"} }, "so you stop walking"},
		{"downed", func(w *world) { w.user.Character.Health = -10 }, "You are down"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			w.walk("inn")
			require.True(t, w.m.Active(7))
			require.Equal(t, 96002, w.user.Character.RoomId)
			tc.do(w)
			out := w.tick()
			assert.Contains(t, out, tc.want)
			assert.False(t, w.m.Active(7), "one line told why, and the walk is over")
		})
	}
}

func TestAWarningAlreadyShowingDoesNotStopTheWalk(t *testing.T) {
	w := newWorld(t)
	w.summ.Leader.Thirst = companyview.Need{Known: true, Value: 20, Label: "Thirsty"}
	w.walk("inn")
	w.tick()
	assert.True(t, w.m.Active(7), "only a new crossing stops it")
	w.summ.Leader.Hunger = companyview.Need{Known: true, Value: 20, Label: "Hungry"}
	assert.Contains(t, w.tick(), "You are hungry")
}

func TestWalkToSomewhereNewReplacesTheWalk(t *testing.T) {
	w := newWorld(t)
	w.walk("96004")
	first, ok := walkplan.Current(7)
	require.True(t, ok)
	require.Len(t, w.timers, 1)

	w.walk("the alder inn")
	second, ok := walkplan.Current(7)
	require.True(t, ok)
	assert.NotEqual(t, first.Gen, second.Gen, "a new start replaces the walk")

	// the first walk's timer is stale; only the second one moves the leader
	before := w.user.Character.RoomId
	w.tick()
	assert.Equal(t, before, w.user.Character.RoomId, "the stale timer does nothing")
	w.tick()
	assert.Equal(t, 96004, w.user.Character.RoomId)
}

func TestSessionEndsEndTheWalk(t *testing.T) {
	w := newWorld(t)
	w.walk("inn")
	w.m.onDespawn(events.PlayerDespawn{UserId: 7})
	assert.False(t, w.m.Active(7))
	w.walk("inn")
	w.m.onUserGone(events.UserPurged{UserId: 7})
	assert.False(t, w.m.Active(7))
}

// TestWalktoThroughPluginsLoad drives the shipped config and the registered
// command through plugins.Load and usercommands.TryCommand, with the real
// Input listener.
func TestWalktoThroughPluginsLoad(t *testing.T) {
	loadAliases(t)
	require.NotNil(t, module, "init registered the module")
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(t.TempDir())
	assert.Equal(t, 1500*time.Millisecond, module.stepEvery, "the shipped config's step time")

	w := newWorld(t)

	w.said = nil
	handled, err := usercommands.TryCommand("walkto", "inn", 7, events.CmdSkipScripts)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(w.said, ""), "You set out for The Alder Inn")
	assert.True(t, module.Active(7))
	w.typed("say hi")
	assert.False(t, module.Active(7), "the registered Input listener ends it")
}

// Review 40d: a step the Go command refuses (no action points here; a room
// script or a no-go buff likewise) used to be re-queued every step forever.
func TestARefusedStepStopsTheWalk(t *testing.T) {
	w := newWorld(t)
	w.walk("inn")
	require.Equal(t, 96002, w.user.Character.RoomId)
	w.user.Character.ActionPoints = 0
	out := w.tick()
	assert.Contains(t, out, "too tired to move", "the refused step is tried once")
	assert.True(t, w.m.Active(7))
	out = w.tick()
	assert.Contains(t, out, "Something keeps you from going on, so you stop walking.")
	assert.False(t, w.m.Active(7))
	assert.Equal(t, 96002, w.user.Character.RoomId)
	assert.Empty(t, w.timers, "no timer re-arms after the stop")
}
