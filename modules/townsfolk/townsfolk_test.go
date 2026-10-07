package townsfolk

import (
	"errors"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/townsfolk"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type memStore struct {
	saved   []byte
	saves   int
	loadErr error
}

func (s *memStore) Load(r *Registry) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	*r = Registry{Users: map[int]UserState{}}
	if s.saved == nil {
		return nil
	}
	return yaml.Unmarshal(s.saved, r)
}

func (s *memStore) Save(r Registry) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	s.saved = data
	s.saves++
	return nil
}

type pushed struct {
	user      int
	namespace string
	payload   any
}

type fakeWorld struct {
	names   map[int]string
	flags   map[int]map[string]bool
	tags    map[string]bool // "uid|key|tag"
	weather string
	night   bool
	pushes  []pushed
	setErr  error
}

func (f *fakeWorld) Name(id int) string { return f.names[id] }
func (f *fakeWorld) Flag(id int, flag string) bool {
	return f.flags[id][flag]
}
func (f *fakeWorld) SetFlag(id int, flag string) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.flags[id] == nil {
		f.flags[id] = map[string]bool{}
	}
	f.flags[id][flag] = true
	return nil
}
func (f *fakeWorld) MemberTag(id int, key, tag string) bool {
	return f.tags[strings.Join([]string{itoa(id), key, tag}, "|")]
}
func (f *fakeWorld) Weather(string) string         { return f.weather }
func (f *fakeWorld) Night() bool                   { return f.night }
func (f *fakeWorld) Rand(int) int                  { return 0 }
func (f *fakeWorld) Push(id int, ns string, p any) { f.pushes = append(f.pushes, pushed{id, ns, p}) }

func itoa(n int) string { return strconv.Itoa(n) }

type rig struct {
	m     *Module
	w     *fakeWorld
	store *memStore
	now   time.Time
	log   *chronicle.Memory
}

const testLines = `
- id: boss
  kind: boss
  tags: [gossip]
  text: "{who} put down {subject}."
  sets: boss-slayers
- id: relic-after-boss
  kind: relic
  flag: boss-slayers
  tags: [gossip]
  text: "And now {subject} too, {who}."
- id: rain
  weather: [rain]
  tags: [gossip]
  text: "Wet again."
- id: priest-only
  kind: spared
  tags: [priest]
  text: "Mercy, {who}. The gods note it."
`

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{
		w:     &fakeWorld{names: map[int]string{7: "Mara", 8: "Tobin"}, flags: map[int]map[string]bool{}, tags: map[string]bool{}},
		store: &memStore{},
		now:   time.Unix(1_800_000_000, 0),
		log:   chronicle.NewMemory(),
	}
	r.log.Now = func() time.Time { return r.now }
	chronicle.SetProvider(r.log)
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	r.m = newModule()
	r.m.w = r.w
	r.m.store = r.store
	r.m.clock = func() time.Time { return r.now }
	r.m.readFiles = func() map[string][]byte { return map[string][]byte{"t.yaml": []byte(testLines)} }
	return r
}

func (r *rig) deed(uid int, e chronicle.Entry) {
	r.log.Record(uid, e)
}

// say is a talker's turn whose line was let through: it speaks and the
// telling is confirmed, as the idle hook does.
func (r *rig) say(npc townsfolk.NPC, listeners []int) (townsfolk.Speech, bool) {
	sp, ok := r.m.Speak(npc, listeners)
	if ok {
		sp.Confirm()
	}
	return sp, ok
}

func gossip() townsfolk.NPC {
	return townsfolk.NPC{MobID: 1, Tags: []string{"gossip"}, Zone: "Alderbrook"}
}

func TestADeedIsToldOncePerPlayerAndSurvivesARestart(t *testing.T) {
	r := newRig(t)
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})

	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "@7", sp.To, "a deed is said to the player")
	assert.Equal(t, "Mara put down the Hollow King.", sp.Text)
	assert.Equal(t, 1, r.store.saves, "what was told is saved before it is said")

	_, ok = r.say(gossip(), []int{7})
	assert.False(t, ok, "the same deed is never told twice")

	// A restart remembers.
	again := newRig(t)
	again.store = r.store
	again.m.store = r.store
	again.m.load()
	st := again.m.state[7]
	assert.Equal(t, []int{1}, st.Heard)
	assert.Equal(t, 1, st.Total)
	require.Len(t, st.Told, 1)
	assert.Equal(t, "boss", st.Told[0].Line)
}

func TestEachPlayerHearsTheirOwnCompanysDeeds(t *testing.T) {
	r := newRig(t)
	r.deed(8, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Tobin"}, Subject: "the Pale Wolf"})
	// Mara has no deeds; Tobin does: the talker speaks to Tobin only.
	sp, ok := r.say(gossip(), []int{7, 8})
	require.True(t, ok)
	assert.Equal(t, "@8", sp.To)
	assert.Contains(t, sp.Text, "the Pale Wolf")
	// A second deed for Mara is a separate telling.
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	sp, ok = r.say(gossip(), []int{7, 8})
	require.True(t, ok)
	assert.Equal(t, "@7", sp.To)
}

func TestADeedOutsideTheWindowOrWithoutALineIsNotTold(t *testing.T) {
	r := newRig(t)
	r.deed(7, chronicle.Entry{Kind: chronicle.Fell, Members: []string{"Mara"}})
	r.deed(7, chronicle.Entry{Kind: chronicle.Spared, Members: []string{"Mara"}, Subject: "a bandit"}) // priest-only
	_, ok := r.say(gossip(), []int{7})
	assert.False(t, ok, "no line for a fall; the spare is a priest's")
	priest := townsfolk.NPC{Tags: []string{"priest"}}
	sp, ok := r.say(priest, []int{7})
	require.True(t, ok)
	assert.Contains(t, sp.Text, "Mercy, Mara")

	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	r.now = r.now.Add(15 * 24 * time.Hour)
	_, ok = r.say(gossip(), []int{7})
	assert.False(t, ok, "talk fades after the window")
}

func TestATelledLineLeavesAMarkThatLaterLinesRead(t *testing.T) {
	r := newRig(t)
	r.deed(7, chronicle.Entry{Kind: chronicle.Relic, Members: []string{"Mara"}, Subject: "the Pale Crown"})
	_, ok := r.say(gossip(), []int{7})
	assert.False(t, ok, "the relic line needs the mark first")

	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	_, ok = r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.True(t, r.w.flags[7]["boss-slayers"], "telling the boss deed marks the company")
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "And now the Pale Crown too, Mara.", sp.Text, "the mark opens the relic line")
	assert.Equal(t, []string{"boss-slayers"}, r.m.marks(7, r.m.lines()))
}

func TestAFailedMarkNeverBlocksTheTelling(t *testing.T) {
	r := newRig(t)
	r.w.setErr = errors.New("no flag provider")
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Contains(t, sp.Text, "Hollow King")
}

func TestWithNothingNewTheTalkerSpeaksOfTheWeatherToTheRoomOrNotAtAll(t *testing.T) {
	r := newRig(t)
	_, ok := r.say(gossip(), []int{7})
	assert.False(t, ok, "clear weather and no deeds: silence")
	r.w.weather = "rain"
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "Wet again.", sp.Text)
	assert.Empty(t, sp.To, "a state line is said to the room")
	assert.Zero(t, r.store.saves, "a state line is not a telling and is not kept")

	// A deed for anyone in the room still wins over the weather.
	r.deed(8, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Tobin"}, Subject: "the Pale Wolf"})
	sp, ok = r.say(gossip(), []int{7, 8})
	require.True(t, ok)
	assert.Equal(t, "@8", sp.To)
}

func TestSignedOutListenersAndEmptyCatalogsAreIgnored(t *testing.T) {
	r := newRig(t)
	r.deed(9, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Ghost"}})
	_, ok := r.say(gossip(), []int{9})
	assert.False(t, ok, "user 9 has no name: not signed in")

	empty := newRig(t)
	empty.m.readFiles = func() map[string][]byte { return nil }
	empty.deed(7, chronicle.Entry{Kind: chronicle.Boss})
	_, ok = empty.say(gossip(), []int{7})
	assert.False(t, ok)
}

func TestBadLinesAreSkippedAndGoodOnesKept(t *testing.T) {
	r := newRig(t)
	r.m.readFiles = func() map[string][]byte {
		return map[string][]byte{
			"a.yaml": []byte("- id: ok\n  kind: boss\n  text: fine\n- id: bad\n  kind: nonsense\n  text: x\n"),
			"b.yaml": []byte("not: [a list"),
			"c.yaml": []byte("- id: typo\n  kind: boss\n  text: x\n  tpyo: 1\n"),
		}
	}
	cat := r.m.lines()
	assert.Equal(t, 1, cat.Len())
	assert.Equal(t, "ok", cat.Lines()[0].ID)
}

func TestTheShippedTestLinesAreSound(t *testing.T) {
	files := readLineFiles()
	require.NotEmpty(t, files)
	var all []townsfolk.Line
	for name, data := range files {
		list, err := townsfolk.Parse(data)
		require.NoError(t, err, name)
		all = append(all, list...)
	}
	cat, problems := townsfolk.NewCatalog(all)
	assert.Empty(t, problems)
	assert.Equal(t, len(all), cat.Len())
	for _, l := range cat.Lines() {
		if l.IsDeed() {
			continue
		}
		assert.NotEmpty(t, l.Tags, "%s: a state line for the test talker", l.ID)
	}
}

func TestAPurgedUserForgetsWhatTheyWereTold(t *testing.T) {
	r := newRig(t)
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}})
	r.deed(8, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Tobin"}})
	_, _ = r.say(gossip(), []int{7})
	_, _ = r.say(gossip(), []int{8})
	r.m.onUserPurged(events.UserPurged{UserId: 7})
	assert.NotContains(t, r.m.state, 7)
	assert.Contains(t, r.m.state, 8)
	assert.NotContains(t, string(r.store.saved), "\n  7:")
}

func TestTheTestAreaSnapshotsAndRestoresWhatWasTold(t *testing.T) {
	r := newRig(t)
	c := stateContributor{r.m}
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	snap, err := c.Capture(7)
	require.NoError(t, err)
	assert.Nil(t, snap, "nothing told yet")
	_, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	r.w.pushes = nil
	require.NoError(t, c.Restore(7, 0, snap))
	assert.NotContains(t, r.m.state, 7, "a trip's tellings leave no trace")
	assert.NotEmpty(t, r.w.pushes)
	_, ok = r.say(gossip(), []int{7})
	assert.True(t, ok, "the deed can be told again after the trip")
	assert.Contains(t, userstate.Names(), "townsfolk")
}

func TestRememberedTellingsAreCapped(t *testing.T) {
	r := newRig(t)
	for i := 0; i < maxTold+5; i++ {
		r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "foe"})
		_, ok := r.say(gossip(), []int{7})
		require.True(t, ok)
	}
	st := r.m.state[7]
	assert.Len(t, st.Told, maxTold)
	assert.Equal(t, maxTold+5, st.Total)
	assert.Equal(t, st.Told[maxTold-1].Seq, st.Heard[len(st.Heard)-1])
}

func TestTheCommandAndPanelShowWhatWasToldAndWhatMayBe(t *testing.T) {
	r := newRig(t)
	assert.Contains(t, r.m.render(7), "Nobody has spoken of your company yet")
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	assert.Contains(t, r.m.render(7), "Folk may yet speak of", "a fresh deed is listed before it is told")
	_, _ = r.say(gossip(), []int{7})
	r.now = r.now.Add(2 * time.Hour)
	r.deed(7, chronicle.Entry{Kind: chronicle.Relic, Members: []string{"Mara"}, Subject: "the Pale Crown"})
	r.deed(7, chronicle.Entry{Kind: chronicle.Fell, Members: []string{"Mara"}})

	text := r.m.render(7)
	assert.Contains(t, text, `"Mara put down the Hollow King."`)
	assert.Contains(t, text, "2 hours ago")
	assert.Contains(t, text, "Folk may yet speak of")
	assert.Contains(t, text, "Marks the towns carry of you: boss-slayers.")
	assert.Contains(t, text, "1 telling(s) so far")

	p := r.m.panelFor(7)
	assert.Equal(t, 1, p.Total)
	require.Len(t, p.Told, 1)
	assert.Equal(t, "Mara put down the Hollow King.", p.Told[0].Text)
	require.Len(t, p.Fresh, 1, "the fall has no line, the relic does")
	assert.Equal(t, "relic", p.Fresh[0].Kind)
	assert.Equal(t, []string{"boss-slayers"}, p.Marks)

	r.w.pushes = nil
	r.m.push(7)
	require.Len(t, r.w.pushes, 1)
	assert.Equal(t, "Company.Townsfolk", r.w.pushes[0].namespace)
	r.m.push(99)
	assert.Len(t, r.w.pushes, 1, "a signed-out player is not pushed to")
}

func TestALoadFailureStopsSavingSoNothingIsOverwritten(t *testing.T) {
	r := newRig(t)
	r.store.loadErr = errors.New("unreadable")
	r.m.load()
	r.store.saved = []byte("kept")
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}})
	_, ok := r.say(gossip(), []int{7})
	assert.True(t, ok, "talk goes on")
	assert.Equal(t, "kept", string(r.store.saved))
}

// The real idle turn: a town talker told what a company did says so, once,
// instead of its usual idle command, and the engine's chatter limits still
// apply.
func TestATalkerMentionsADeedThroughTheRealIdleTurn(t *testing.T) {
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	mobs.ForgetChatterForTest()
	t.Cleanup(mobs.ForgetChatterForTest)

	gameplay := configs.GetGamePlayConfig()
	gameplay.MobChatterCooldownRounds = 60
	gameplay.MobChatterMemoryRounds = 900
	gameplay.MobConverseChance = 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	prevRound := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(prevRound) })

	r := newRig(t)
	townsfolk.SetProvider(r.m)
	t.Cleanup(func() { townsfolk.SetProvider(nil) })
	// The real shape of a boss deed (internal/mobcommands/suicide.go): the
	// company's, naming nobody.
	r.deed(41, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:12"})
	r.w.names[41] = "Mara"

	room := &rooms.Room{RoomId: 990401, Title: "Market square"}
	room.SetTestOccupants([]int{41}, []int{990401})
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })

	talker := &mobs.Mob{
		InstanceId:    990401,
		MobId:         990401,
		HomeRoomId:    room.RoomId,
		MaxWander:     -1,
		ActivityLevel: 100,
		Townsfolk:     []string{"gossip"},
		IdleCommands:  []string{"emote scratches his chin"},
	}
	talker.Character.Name = "gossip"
	talker.Character.RoomId = room.RoomId
	mobs.SetTestInstance(talker)
	t.Cleanup(func() { mobs.RemoveTestInstance(talker.InstanceId) })

	var said []string
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId == talker.InstanceId && (strings.HasPrefix(in.InputText, "say") || strings.HasPrefix(in.InputText, "emote")) {
			said = append(said, in.InputText)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })

	for round := uint64(10000); round < 10130; round++ {
		util.SetRoundCount(round)
		hooks.HandleIdleMobs(events.MobIdle{MobInstanceId: talker.InstanceId})
		events.ProcessEvents()
	}

	require.NotEmpty(t, said)
	assert.Equal(t, "sayto @41 Mara put down the Hollow King.", said[0], "the deed is said to the player, in the first idle turn")
	told := 0
	for _, s := range said {
		if strings.Contains(s, "Hollow King") {
			told++
		}
	}
	assert.Equal(t, 1, told, "once per deed per player: %v", said)
	for i := 1; i < len(said); i++ {
		assert.True(t, strings.HasPrefix(said[i], "emote"), "afterwards the usual idle turn: %s", said[i])
	}

	// The same boss again within the hour: the identical line is held back by
	// the chatter memory, so the deed stays untold (68 review) ...
	r.deed(41, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:12"})
	said = nil
	for round := uint64(10200); round < 10400; round++ {
		util.SetRoundCount(round)
		hooks.HandleIdleMobs(events.MobIdle{MobInstanceId: talker.InstanceId})
		events.ProcessEvents()
	}
	for _, s := range said {
		assert.NotContains(t, s, "Hollow King")
	}
	assert.Equal(t, 1, r.m.state[41].Total, "a held-back line is not counted as told")
	// ... and is told once the player could hear the line again.
	said = nil
	for round := uint64(11000); round < 11100; round++ {
		util.SetRoundCount(round)
		hooks.HandleIdleMobs(events.MobIdle{MobInstanceId: talker.InstanceId})
		events.ProcessEvents()
	}
	require.NotEmpty(t, said)
	assert.Equal(t, "sayto @41 Mara put down the Hollow King.", said[0])
	assert.Equal(t, 2, r.m.state[41].Total)
}

// A mob that is no talker never reaches the module.
func TestAMobWithoutTownsfolkTagsIsNotATalker(t *testing.T) {
	r := newRig(t)
	townsfolk.SetProvider(r.m)
	t.Cleanup(func() { townsfolk.SetProvider(nil) })
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}})
	_, ok := townsfolk.Speak(townsfolk.NPC{MobID: 5}, []int{7})
	assert.False(t, ok)
	assert.Zero(t, r.store.saves)
}

// Every talker tag a shipped mob carries has at least one line to say, and
// the test talker loads with its tag.
func TestShippedTalkersHaveLinesToSay(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "_datafiles", "world", "default", "mobs")
	cat := newModule().lines()
	say := map[string]bool{}
	for _, l := range cat.Lines() {
		for _, tag := range l.Tags {
			say[strings.ToLower(tag)] = true
		}
	}
	found := map[int][]string{}
	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var mob mobs.Mob
		if err := yaml.Unmarshal(data, &mob); err != nil {
			return nil // other tests cover mob files
		}
		if len(mob.Townsfolk) > 0 {
			found[int(mob.MobId)] = mob.Townsfolk
		}
		return nil
	}))
	assert.Equal(t, []string{"gossip"}, found[90301], "the test talker")
	for id, tags := range found {
		for _, tag := range tags {
			assert.True(t, say[strings.ToLower(tag)], "mob %d: no line for talker tag %q", id, tag)
		}
	}
}

type fakeFlags struct{ set map[int]map[string]bool }

func (f *fakeFlags) CompanyFlags(id int) []string {
	var out []string
	for k := range f.set[id] {
		out = append(out, k)
	}
	return out
}

func (f *fakeFlags) SetCompanyFlag(id int, flag string) error {
	if f.set[id] == nil {
		f.set[id] = map[string]bool{}
	}
	f.set[id][flag] = true
	return nil
}

// The live world reads and leaves marks through the story-event flag seam
// (Phase 60) and reads member tags through its tag-source seam (the Phase 72
// hook).
func TestTheLiveWorldUsesTheStoryEventSeams(t *testing.T) {
	flags := &fakeFlags{set: map[int]map[string]bool{}}
	storyevents.SetFlagProvider(flags)
	t.Cleanup(func() { storyevents.SetFlagProvider(nil) })
	storyevents.RegisterTagSource(func(leader int, key string) []string {
		if leader == 7 && key == "companion:2" {
			return []string{"devout"}
		}
		if leader == 7 && key == "leader" {
			return []string{"Soldier"} // a source's case, as story events accept it
		}
		return nil
	})
	w := liveWorld{}
	assert.False(t, w.Flag(7, "boss-slayers"))
	require.NoError(t, w.SetFlag(7, "boss-slayers"))
	assert.True(t, w.Flag(7, "boss-slayers"))
	assert.True(t, flags.set[7]["boss-slayers"], "the mark is a company flag a scene can read")
	assert.Error(t, w.SetFlag(7, "Not A Flag"), "the flag seam validates")
	assert.True(t, w.MemberTag(7, "companion:2", "devout"))
	assert.False(t, w.MemberTag(7, "leader", "devout"))
	assert.True(t, w.MemberTag(7, "leader", "soldier"), "tags match regardless of case, as story events' requirements do")
}

type constantTalker struct{}

func (constantTalker) Speak(townsfolk.NPC, []int) (townsfolk.Speech, bool) {
	return townsfolk.Speech{Text: "Wet again."}, true
}

// A line the engine's chatter limits hold back (the room already heard it)
// leaves the talker's usual idle turn to run, so a wandering talker is never frozen.
func TestAHeldBackLineLeavesTheUsualIdleTurn(t *testing.T) {
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	mobs.ForgetChatterForTest()
	t.Cleanup(mobs.ForgetChatterForTest)
	gameplay := configs.GetGamePlayConfig()
	gameplay.MobChatterCooldownRounds = 10
	gameplay.MobChatterMemoryRounds = 900
	gameplay.MobConverseChance = 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	prevRound := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(prevRound) })
	townsfolk.SetProvider(constantTalker{})
	t.Cleanup(func() { townsfolk.SetProvider(nil) })

	room := &rooms.Room{RoomId: 990402, Title: "Market square"}
	room.SetTestOccupants([]int{42}, []int{990402})
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
	talker := &mobs.Mob{InstanceId: 990402, MobId: 990402, HomeRoomId: room.RoomId, MaxWander: -1, ActivityLevel: 100,
		Townsfolk: []string{"gossip"}, IdleCommands: []string{"look"}}
	talker.Character.Name = "gossip"
	talker.Character.RoomId = room.RoomId
	mobs.SetTestInstance(talker)
	t.Cleanup(func() { mobs.RemoveTestInstance(talker.InstanceId) })

	var acts []string
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId == talker.InstanceId {
			acts = append(acts, in.InputText)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })

	for _, round := range []uint64{20000, 20020} { // past the cooldown both times
		util.SetRoundCount(round)
		hooks.HandleIdleMobs(events.MobIdle{MobInstanceId: talker.InstanceId})
		events.ProcessEvents()
	}
	require.Len(t, acts, 2, "%v", acts)
	assert.Equal(t, "say Wet again.", acts[0])
	assert.Equal(t, "look", acts[1], "the player already heard the line, so the usual idle turn ran (a wandering talker keeps wandering)")
}

// A line the chatter limits hold back (Speak without Confirm) uses nothing
// up: the deed is told the next time, and no mark is left (68 review).
func TestAnUnconfirmedTellingLeavesTheDeedUntold(t *testing.T) {
	r := newRig(t)
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Members: []string{"Mara"}, Subject: "the Hollow King"})
	sp, ok := r.m.Speak(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "@7", sp.To)
	assert.Zero(t, r.store.saves)
	assert.Zero(t, r.m.state[7].Total)
	assert.False(t, r.w.flags[7]["boss-slayers"], "no mark for a line never said")

	_, ok = r.say(gossip(), []int{7})
	require.True(t, ok, "the deed is still there to tell")
	assert.Equal(t, 1, r.m.state[7].Total)
	assert.True(t, r.w.flags[7]["boss-slayers"])
}

// The game records a boss slain, a relic found and a mercy answer as the
// company's deed, naming nobody: the line names the leader and member_tag
// reads the leader's tags (68 review; the phase 72 seam).
func TestACompanyDeedIsTheLeaders(t *testing.T) {
	r := newRig(t)
	r.m.readFiles = func() map[string][]byte {
		return map[string][]byte{"t.yaml": []byte(testLines + `
- id: soldier-boss
  kind: boss
  member_tag: soldier
  tags: [veteran]
  text: "A soldier's work, {who}."
`)}
	}
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:12"})
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "Mara put down the Hollow King.", sp.Text, "not \"The company put down\"")

	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Pale Wolf", Ref: "mob:13"})
	veteran := townsfolk.NPC{Tags: []string{"veteran"}}
	_, ok = r.say(veteran, []int{7})
	assert.False(t, ok, "the leader carries no soldier tag yet")
	r.w.tags["7|"+townsfolk.LeaderKey+"|soldier"] = true
	sp, ok = r.say(veteran, []int{7})
	require.True(t, ok)
	assert.Equal(t, "A soldier's work, Mara.", sp.Text)
}

func TestTheLeaderKeyIsTheCompanys(t *testing.T) {
	assert.Equal(t, string(survival.LeaderMemberKey), townsfolk.LeaderKey)
}

// A new deed refreshes the web view, so "not yet spoken of" is current
// (68 review).
func TestANewDeedRefreshesTheView(t *testing.T) {
	r := newRig(t)
	prevW, prevFiles, prevClock := module.w, module.readFiles, module.clock
	module.w, module.readFiles, module.clock = r.w, r.m.readFiles, r.m.clock
	module.mu.Lock()
	prevCat := module.catalog
	module.catalog = nil
	module.mu.Unlock()
	t.Cleanup(func() {
		module.w, module.readFiles, module.clock = prevW, prevFiles, prevClock
		module.mu.Lock()
		module.catalog = prevCat
		module.mu.Unlock()
	})
	chronicle.Record(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King"})
	require.NotEmpty(t, r.w.pushes)
	last := r.w.pushes[len(r.w.pushes)-1]
	assert.Equal(t, "Company.Townsfolk", last.namespace)
	require.Len(t, last.payload.(panel).Fresh, 1)
}

// A shipped town line for a background (Phase 72) is told to a leader who
// has it, through the module's real choosing and the shipped lines.
func TestAShippedLineSpeaksToTheLeadersBackground(t *testing.T) {
	r := newRig(t)
	r.m.readFiles = readLineFiles
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:12"})
	r.w.tags["7|leader|trade-soldier"] = true
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Contains(t, sp.Text, "A soldier, Mara, and it shows")

	r2 := newRig(t)
	r2.m.readFiles = readLineFiles
	r2.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:12"})
	sp, ok = r2.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Contains(t, sp.Text, "put down", "no background, the plain line")
	assert.NotContains(t, sp.Text, "soldier")
}

// Phase 72 review: a background line beats a plain one only while the
// listener has not heard it lately, so a soldier is not told the same drill
// line of every boss.
func TestABackgroundLineIsNotSaidOfEveryDeed(t *testing.T) {
	r := newRig(t)
	r.m.readFiles = func() map[string][]byte {
		return map[string][]byte{"t.yaml": []byte(testLines + `
- id: boss-soldier
  kind: boss
  member_tag: trade-soldier
  tags: [gossip]
  text: "Like drill, {who}."
`)}
	}
	r.w.tags["7|leader|trade-soldier"] = true
	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King"})
	sp, ok := r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "Like drill, Mara.", sp.Text, "first, the line for the background")

	r.deed(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Bone Ogre"})
	sp, ok = r.say(gossip(), []int{7})
	require.True(t, ok)
	assert.Equal(t, "Mara put down the Bone Ogre.", sp.Text, "heard lately, it joins the plain lines")
}
