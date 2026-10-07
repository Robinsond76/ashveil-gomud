package bounties

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/bounty"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type memStore struct {
	saved   []byte
	saves   int
	failing bool
	loadErr error
}

func (s *memStore) Load(r *Registry) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	*r = Registry{Companies: map[int]bounty.State{}}
	if s.saved == nil {
		return nil
	}
	return yaml.Unmarshal(s.saved, r)
}

func (s *memStore) Save(r Registry) error {
	if s.failing {
		return errors.New("disk full")
	}
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
	board      board
	atBoard    bool
	candidates []bounty.Target
	level      int
	busy       bool
	gold       int
	pushes     []pushed
}

func (f *fakeWorld) Board(int) (board, bool)       { return f.board, f.atBoard }
func (f *fakeWorld) Candidates() []bounty.Target   { return f.candidates }
func (f *fakeWorld) Level(int) int                 { return f.level }
func (f *fakeWorld) Place(int) string              { return "the board" }
func (f *fakeWorld) Busy(int) bool                 { return f.busy }
func (f *fakeWorld) Online(int) bool               { return true }
func (f *fakeWorld) Pay(_ int, gold int) bool      { f.gold += gold; return true }
func (f *fakeWorld) Push(id int, ns string, p any) { f.pushes = append(f.pushes, pushed{id, ns, p}) }

type rig struct {
	m     *Module
	w     *fakeWorld
	store *memStore
	now   time.Time
	mem   *chronicle.Memory
	user  *users.UserRecord
}

const leader = 7

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{
		w: &fakeWorld{board: board{Key: "Alderbrook:2114", Title: "Alderbrook Green", Band: bounty.Band{Low: 2, High: 4}}, atBoard: true, level: 3,
			candidates: []bounty.Target{
				{Kind: bounty.Boss, Zone: "Dark Forest", Ref: "mob:200", Name: "Old Grue", Low: 5, High: 7},
				{Kind: bounty.Group, Zone: "Alderbrook", Ref: "group:field-rats", Name: "Field Rats", Low: 2, High: 4},
				{Kind: bounty.Group, Zone: "Alderbrook", Ref: "group:stray-dogs", Name: "Stray Dogs", Low: 2, High: 4},
			}},
		store: &memStore{},
		now:   time.Unix(1_800_000_000, 0),
		mem:   chronicle.NewMemory(),
		user:  users.NewUserRecord(leader, leader),
	}
	r.mem.Now = func() time.Time { return r.now }
	chronicle.SetProvider(r.mem)
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	r.m = newModule()
	r.m.w = r.w
	r.m.store = r.store
	r.m.clock = func() time.Time { return r.now }
	return r
}

// say runs the command and returns the plain text sent.
func (r *rig) say(t *testing.T, rest string) string {
	t.Helper()
	return stripTags(heard(t, func() {
		handled, err := r.m.command(rest, r.user, nil, events.CmdSecretly)
		require.NoError(t, err)
		require.True(t, handled)
	}))
}

// heard runs fn and returns the text sent to players meanwhile.
func heard(t *testing.T, fn func()) string {
	t.Helper()
	events.ProcessEvents() // flush what earlier tests left queued
	got := ""
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		got += e.(events.Message).Text
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, id)
	fn()
	events.ProcessEvents()
	return got
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, c := range s {
		switch {
		case c == '<':
			in = true
		case c == '>':
			in = false
		case !in:
			b.WriteRune(c)
		}
	}
	return b.String()
}

func (r *rig) posting(t *testing.T, kind string) (int, bounty.Posting) {
	t.Helper()
	for i, p := range r.m.postings(r.w.board) {
		if p.Target.Kind == kind {
			return i + 1, p
		}
	}
	t.Fatalf("no %s posting", kind)
	return 0, bounty.Posting{}
}

// A group bounty through the real take, kill and claim path.
func TestTakeKillAndClaimAGroupBounty(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Group)

	board := r.say(t, "")
	assert.Contains(t, board, "The bounty board at Alderbrook Green")
	assert.Contains(t, board, po.Target.Name)

	assert.Contains(t, r.say(t, "claim"), "holds no bounties")
	assert.Contains(t, r.say(t, "take "+itoa(n)), "You take the bounty")
	assert.Contains(t, r.say(t, "take "+itoa(n)), "already hold a bounty on that mark")

	// A group broken before the bounty was taken does not count.
	assert.Contains(t, r.say(t, "claim"), "no proof")

	deed := chronicle.Entry{Kind: chronicle.Group, Subject: po.Target.Name, Ref: po.Target.Ref, Zone: po.Target.Zone}
	chronicle.Record(leader, deed)
	chronicle.Record(leader, deed)
	assert.Contains(t, r.say(t, "claim"), "no proof", "two of three groups is not enough")
	assert.Contains(t, r.say(t, ""), "2 of 3")
	assert.Zero(t, r.w.gold)

	// A group of the same name in another zone is not this bounty's mark.
	other := deed
	other.Zone = "Elsewhere"
	chronicle.Record(leader, other)
	assert.Contains(t, r.say(t, "claim"), "no proof")

	chronicle.Record(leader, deed)
	out := r.say(t, "claim")
	assert.Contains(t, out, "counts out")
	assert.Equal(t, po.Reward, r.w.gold, "paid once, in gold, at the bounty's rate")
	assert.Equal(t, 1, chronicle.Total(leader, chronicle.Bounty), "the claim is a deed too")
	assert.Contains(t, r.say(t, "claim"), "holds no bounties")
	assert.Equal(t, po.Reward, r.w.gold, "a second claim pays nothing")

	// The same posting cannot be taken again while the board lists it...
	assert.Contains(t, r.say(t, "take "+itoa(n)), "already settled")
	// ...and kills before the next bounty was taken do not pay it.
	r.now = r.now.Add(bounty.WindowSeconds * time.Second)
	n2, _ := r.posting(t, bounty.Group)
	r.say(t, "take "+itoa(n2))
	assert.Contains(t, r.say(t, "claim"), "no proof", "an old deed never pays a new bounty")
}

func TestABossBountyNeedsTheLairMasterInItsZone(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Boss)
	assert.Equal(t, 20*6, po.Reward, "twenty gold a level of the band's middle (5-7)")
	r.say(t, "take "+itoa(n))
	chronicle.Record(leader, chronicle.Entry{Kind: chronicle.Boss, Ref: po.Target.Ref, Zone: "Catacombs", Subject: po.Target.Name})
	assert.Contains(t, r.say(t, "claim"), "no proof", "the same master in another zone is not this lair")
	chronicle.Record(leader, chronicle.Entry{Kind: chronicle.Boss, Ref: po.Target.Ref, Zone: po.Target.Zone, Subject: po.Target.Name})
	assert.Contains(t, r.say(t, "claim "+itoa(1)), "counts out")
	assert.Equal(t, 120, r.w.gold)
	deeds := chronicle.Query(leader, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Bounty}})
	require.Len(t, deeds, 1)
	assert.Equal(t, "120 gold", deeds[0].Detail)
}

func TestTheBoardIsTheSameAcrossARestartAndRotates(t *testing.T) {
	r := newRig(t)
	before := r.m.postings(r.w.board)
	require.NotEmpty(t, before)
	reborn := newModule()
	reborn.w, reborn.clock = r.w, r.m.clock
	assert.Equal(t, before, reborn.postings(r.w.board), "nothing is stored: the list follows from the board and the window")
	r.now = r.now.Add(bounty.WindowSeconds * time.Second)
	after := r.m.postings(r.w.board)
	assert.NotEqual(t, before[0].ID, after[0].ID, "a new window posts new ids")
}

func TestHoldingsSurviveARestartAndLapseOnTheRealClock(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Group)
	r.say(t, "take "+itoa(n))
	require.NotNil(t, r.store.saved)

	reborn := newModule()
	reborn.store, reborn.w, reborn.clock = r.store, r.w, r.m.clock
	reborn.load()
	st, _ := reborn.stateOf(leader)
	require.Len(t, st.Held, 1)
	assert.Equal(t, po.ID, st.Held[0].PostID)

	r.now = r.now.Add((bounty.TermSeconds + 1) * time.Second)
	r.w.atBoard = false
	out := stripTags(reborn.renderHeld(leader))
	assert.Contains(t, out, "lapsed unclaimed")
	assert.Contains(t, out, "holds no bounties")
	assert.NotContains(t, string(r.store.saved), po.ID, "a lapsed bounty is saved away")
}

func TestLimitsAndRefusals(t *testing.T) {
	r := newRig(t)
	assert.Contains(t, r.say(t, "take 9"), "no posting 9")
	assert.Contains(t, r.say(t, "take"), "Take which")
	r.w.busy = true
	assert.Contains(t, r.say(t, "take 1"), "battle")
	assert.Contains(t, r.say(t, "claim"), "battle")
	r.w.busy = false

	r.w.atBoard = false
	assert.Contains(t, r.say(t, "take 1"), "no bounty board here")
	assert.Contains(t, r.say(t, "claim"), "claimed at a board")
	assert.Contains(t, r.say(t, "board"), "no bounty board here")
	assert.Contains(t, r.say(t, ""), "holds no bounties", "the held list reads anywhere")
	assert.Contains(t, r.say(t, "nonsense"), "Usage")
	r.w.atBoard = true

	// MaxHeld bounties at once, then a fourth is refused.
	r.w.candidates = append(r.w.candidates,
		bounty.Target{Kind: bounty.Group, Zone: "Alderbrook", Ref: "group:extra", Name: "Extra", Low: 2, High: 4},
		bounty.Target{Kind: bounty.Group, Zone: "Alderbrook", Ref: "group:more", Name: "More", Low: 2, High: 4})
	require.GreaterOrEqual(t, len(r.m.postings(r.w.board)), bounty.MaxHeld+1)
	for i := 1; i <= bounty.MaxHeld; i++ {
		assert.Contains(t, r.say(t, "take "+itoa(i)), "You take", "posting %d", i)
	}
	st, _ := r.m.stateOf(leader)
	assert.Len(t, st.Held, bounty.MaxHeld)
	assert.Contains(t, r.say(t, "take "+itoa(bounty.MaxHeld+1)), "claim or drop one first")

	assert.Contains(t, r.say(t, "drop 1"), "You drop the bounty")
	st, _ = r.m.stateOf(leader)
	assert.Len(t, st.Held, bounty.MaxHeld-1)
	assert.Contains(t, r.say(t, "drop 9"), "Drop which")
}

func TestAFailedSaveRefusesTheTakeAndTheClaimLeavesNothingHalfDone(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Group)
	r.store.failing = true
	assert.Contains(t, r.say(t, "take "+itoa(n)), "cannot write that down")
	st, _ := r.m.stateOf(leader)
	assert.Empty(t, st.Held)
	r.store.failing = false
	r.say(t, "take "+itoa(n))

	for i := 0; i < bounty.GroupCount; i++ {
		chronicle.Record(leader, chronicle.Entry{Kind: chronicle.Group, Ref: po.Target.Ref, Zone: po.Target.Zone})
	}
	r.store.failing = true
	assert.Contains(t, r.say(t, "claim"), "cannot write that down")
	assert.Zero(t, r.w.gold, "an unsaved claim pays nothing")
	r.store.failing = false
	assert.Contains(t, r.say(t, "claim"), "counts out")
	assert.Equal(t, po.Reward, r.w.gold)
}

func TestALoadFailureStopsSavingSoNothingIsOverwritten(t *testing.T) {
	r := newRig(t)
	r.store.loadErr = errors.New("unreadable")
	r.m.load()
	n, _ := r.posting(t, bounty.Group)
	assert.Contains(t, r.say(t, "take "+itoa(n)), "cannot write that down")
	assert.Nil(t, r.store.saved)
}

func TestAPurgedUserLeavesNoBountiesAndTheTestAreaRoundTrips(t *testing.T) {
	r := newRig(t)
	n, _ := r.posting(t, bounty.Group)
	r.say(t, "take "+itoa(n))

	c := stateContributor{r.m}
	snap, err := c.Capture(leader)
	require.NoError(t, err)
	r.say(t, "drop 1")
	require.NoError(t, c.Restore(leader, 0, snap))
	st, _ := r.m.stateOf(leader)
	assert.Len(t, st.Held, 1, "the test area restores a snapshot")
	var _ userstate.Contributor = c

	r.m.onUserPurged(events.UserPurged{UserId: leader})
	st, _ = r.m.stateOf(leader)
	assert.Empty(t, st.Held)
	assert.NotContains(t, string(r.store.saved), "post_id")
}

func TestThePanelCarriesPostingsHoldingsAndProgress(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Group)
	r.say(t, "take "+itoa(n))
	chronicle.Record(leader, chronicle.Entry{Kind: chronicle.Group, Ref: po.Target.Ref, Zone: po.Target.Zone})

	p := r.m.panelFor(leader)
	assert.True(t, p.AtBoard)
	assert.Equal(t, "Alderbrook Green", p.Board)
	assert.Equal(t, "2-4", p.Band)
	assert.Equal(t, bounty.MaxHeld, p.Max)
	assert.Equal(t, int64(bounty.WindowSeconds)-r.now.Unix()%bounty.WindowSeconds, p.Rotates)
	require.Len(t, p.Held, 1)
	assert.Equal(t, 1, p.Held[0].Have)
	assert.Equal(t, bounty.GroupCount, p.Held[0].Count)
	assert.False(t, p.Held[0].Ready)
	taken := 0
	for _, row := range p.Postings {
		if row.Taken {
			taken++
			assert.Equal(t, po.Target.Name, row.Name)
		}
		assert.NotEmpty(t, row.Rating, "each row tells the company how the zone rates for them")
	}
	assert.Equal(t, 1, taken)

	// Every deed refreshes the web tab.
	r.w.pushes = nil
	r.m.push(leader)
	require.Len(t, r.w.pushes, 1)
	assert.Equal(t, "Company.Bounties", r.w.pushes[0].namespace)

	r.w.atBoard = false
	away := r.m.panelFor(leader)
	assert.False(t, away.AtBoard)
	assert.Empty(t, away.Postings, "away from a board the tab lists only what is held")
	assert.Len(t, away.Held, 1)
}

func itoa(n int) string { return string(rune('0' + n)) }

// --- the shipped world, through the live world seam ---

func loadShippedWorld(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	mobs.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	t.Cleanup(rooms.ResetForTest)
}

func TestTheShippedBoardsPostRealTargetsThatCanBeProved(t *testing.T) {
	loadShippedWorld(t)
	live := liveWorld{}
	cands := live.Candidates()
	require.NotEmpty(t, cands)
	bosses, groups := 0, 0
	for _, c := range cands {
		assert.True(t, c.Band().Valid(), "%s/%s has a zone band", c.Zone, c.Ref)
		assert.NotEmpty(t, c.Name)
		if c.Kind == bounty.Boss {
			bosses++
			assert.True(t, strings.HasPrefix(c.Ref, "mob:"), c.Ref)
		} else {
			groups++
			assert.True(t, strings.HasPrefix(c.Ref, "group:"), c.Ref)
		}
	}
	assert.Positive(t, bosses)
	assert.Positive(t, groups)

	for _, roomID := range []int{2114, 61} {
		room := rooms.LoadRoom(roomID)
		require.NotNil(t, room)
		require.True(t, room.HasTag(BoardTag), "room %d is a board", roomID)
		u := users.NewUserRecord(900+roomID, uint64(900+roomID))
		u.Character.RoomId = roomID
		users.SetTestUser(u)
		t.Cleanup(users.ResetActiveUsers)
		b, ok := live.Board(u.UserId)
		require.True(t, ok)
		sawBoss := false
		for w := int64(0); w < 60; w++ {
			posts := bounty.Post(b.Key, b.Band, w*bounty.WindowSeconds, cands)
			assert.NotEmpty(t, posts, "board %d window %d", roomID, w)
			for _, p := range posts {
				sawBoss = sawBoss || p.Target.Kind == bounty.Boss
			}
		}
		assert.True(t, sawBoss, "board %d now and then posts a lair boss", roomID)
	}
}

// The real command, a real user, the real room tag and gold payment.
func TestTheCommandOnTheShippedBoardPaysGoldToTheCharacter(t *testing.T) {
	loadShippedWorld(t)
	mem := chronicle.NewMemory()
	mem.Now = func() time.Time { return time.Unix(1_800_000_100, 0) }
	chronicle.SetProvider(mem)
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	u := users.NewUserRecord(leader, leader)
	u.Character.RoomId = 2114
	u.Character.Level = 3
	users.SetTestUser(u)
	t.Cleanup(users.ResetActiveUsers)
	m := newModule()
	m.store = &memStore{}
	m.clock = func() time.Time { return time.Unix(1_800_000_000, 0) }

	got := heard(t, func() {
		_, err := m.command("", u, nil, events.CmdSecretly)
		require.NoError(t, err)
	})
	require.Contains(t, got, "The bounty board at Alderbrook Green")

	var target bounty.Target
	var reward int
	for i, p := range m.postings(board{Key: "Alderbrook:2114", Band: bounty.Band{Low: 2, High: 4}}) {
		if p.Target.Kind == bounty.Group {
			heard(t, func() {
				_, err := m.command("take "+itoa(i+1), u, nil, events.CmdSecretly)
				require.NoError(t, err)
			})
			target, reward = p.Target, p.Reward
			break
		}
	}
	require.NotEmpty(t, target.Ref)
	for i := 0; i < bounty.GroupCount; i++ {
		chronicle.Record(leader, chronicle.Entry{Kind: chronicle.Group, Ref: target.Ref, Zone: target.Zone})
	}
	before := u.Character.Gold
	heard(t, func() {
		_, err := m.command("claim", u, nil, events.CmdSecretly)
		require.NoError(t, err)
	})
	assert.Equal(t, before+reward, u.Character.Gold)

	// Away from the board the same command refuses to claim.
	u.Character.RoomId = 2111
	got = heard(t, func() { _, _ = m.command("claim", u, nil, events.CmdSecretly) })
	assert.Contains(t, got, "claimed at a board")
}

// Review: the encounters module writes a group deed only while a live
// bounty names that group in that zone, so ordinary fights do not crowd
// the chronicle.
func TestHuntingIsOnlyAHeldLiveBountyOnThatGroupAndZone(t *testing.T) {
	r := newRig(t)
	n, po := r.posting(t, bounty.Group)
	assert.False(t, r.m.hunting(leader, po.Target.Ref, po.Target.Zone), "nothing held")
	r.say(t, "take "+itoa(n))
	assert.True(t, r.m.hunting(leader, po.Target.Ref, po.Target.Zone))
	assert.False(t, r.m.hunting(leader, po.Target.Ref, "Elsewhere"), "another zone's group of that name")
	assert.False(t, r.m.hunting(leader+1, po.Target.Ref, po.Target.Zone), "another company")
	r.now = r.now.Add(bounty.TermSeconds * time.Second)
	assert.False(t, r.m.hunting(leader, po.Target.Ref, po.Target.Zone), "a lapsed bounty")
}
