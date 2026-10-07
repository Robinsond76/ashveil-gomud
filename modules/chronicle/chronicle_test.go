package chronicle

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
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
	*r = Registry{Companies: map[int]chronicle.Log{}}
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
	place   string
	online  map[int]bool
	pushes  []pushed
	rooms   map[int]string
	mobs    map[int]string
	nameFor string
}

func (f *fakeWorld) Place(int) string        { return f.place }
func (f *fakeWorld) RoomTitle(id int) string { return f.rooms[id] }
func (f *fakeWorld) Name(_ int, fallback string) string {
	if f.nameFor != "" {
		return f.nameFor
	}
	return fallback
}
func (f *fakeWorld) MobName(id int) string { return f.mobs[id] }
func (f *fakeWorld) Online(id int) bool    { return f.online[id] }
func (f *fakeWorld) Push(id int, ns string, p any) {
	f.pushes = append(f.pushes, pushed{id, ns, p})
}

type rig struct {
	m     *Module
	w     *fakeWorld
	store *memStore
	now   time.Time
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{
		w:     &fakeWorld{place: "the Waymark Inn", online: map[int]bool{7: true}, rooms: map[int]string{30: "the ford"}, mobs: map[int]string{12: "a wolf"}},
		store: &memStore{},
		now:   time.Unix(1_800_000_000, 0),
	}
	r.m = newModule()
	r.m.w = r.w
	r.m.store = r.store
	r.m.clock = func() time.Time { return r.now }
	return r
}

func TestARecordedDeedIsStampedSavedAndPushed(t *testing.T) {
	r := newRig(t)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Joined, Members: []string{"Mara"}})
	l := r.m.Log(7)
	require.Len(t, l.Entries, 1)
	e := l.Entries[0]
	assert.Equal(t, r.now.Unix(), e.At, "a zero time is stamped")
	assert.Equal(t, "the Waymark Inn", e.Place, "an empty place is where the leader stands")
	assert.Equal(t, 1, e.Seq)
	assert.Equal(t, 1, r.store.saves, "a deed is saved at once")
	require.Len(t, r.w.pushes, 1)
	assert.Equal(t, "Company.Chronicle", r.w.pushes[0].namespace)

	r.m.Record(8, chronicle.Entry{Kind: chronicle.Joined, Members: []string{"Tobin"}, Place: "the road", At: 5})
	assert.Equal(t, "the road", r.m.Log(8).Entries[0].Place, "a given place and time stand")
	assert.Equal(t, int64(5), r.m.Log(8).Entries[0].At)
	assert.Len(t, r.w.pushes, 1, "a signed-out leader is not pushed to")
	assert.Empty(t, r.m.Log(7).Entries[0].Members[1:], "companies are separate")
	assert.Len(t, r.m.Log(8).Entries, 1)
}

func TestTheChronicleSurvivesARestart(t *testing.T) {
	r := newRig(t)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:9"})
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Relic, Subject: "the Pale Crown", Ref: "item:50033"})

	again := newRig(t)
	again.store = r.store
	again.m.store = r.store
	again.m.load()
	l := again.m.Log(7)
	require.Len(t, l.Entries, 2)
	assert.Equal(t, "item:50033", l.Entries[1].Ref)
	assert.Equal(t, 1, l.Tally[chronicle.Boss])
	again.m.Record(7, chronicle.Entry{Kind: chronicle.Boss})
	assert.Equal(t, 3, again.m.Log(7).Entries[2].Seq, "numbering carries on after a restart")
}

func TestAFailedSaveKeepsTheDeedForTheNextSave(t *testing.T) {
	r := newRig(t)
	r.store.failing = true
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Fell, Members: []string{"Mara"}})
	require.Len(t, r.m.Log(7).Entries, 1, "a deed that happened is never refused")
	r.store.failing = false
	require.NoError(t, r.m.save())
	assert.Contains(t, string(r.store.saved), "Mara")
}

func TestALoadFailureStopsSavingSoNothingIsOverwritten(t *testing.T) {
	r := newRig(t)
	r.store.loadErr = errors.New("unreadable")
	r.m.load()
	r.store.saved = []byte("kept")
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Joined})
	assert.Equal(t, "kept", string(r.store.saved), "the unreadable file is left alone")
	assert.Len(t, r.m.Log(7).Entries, 1)
}

func TestAPurgedUserLeavesNoChronicle(t *testing.T) {
	r := newRig(t)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Joined})
	r.m.Record(8, chronicle.Entry{Kind: chronicle.Joined})
	r.m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Empty(t, r.m.Log(7).Entries)
	assert.Len(t, r.m.Log(8).Entries, 1)
	assert.NotContains(t, string(r.store.saved), "companies:\n  7:")
}

func TestTheTestAreaSnapshotsAndRestoresTheChronicle(t *testing.T) {
	r := newRig(t)
	c := stateContributor{r.m}
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Joined, Members: []string{"Mara"}})
	snap, err := c.Capture(7)
	require.NoError(t, err)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King"})
	r.w.pushes = nil
	require.NoError(t, c.Restore(7, 0, snap))
	l := r.m.Log(7)
	require.Len(t, l.Entries, 1, "a trip's deeds leave no trace")
	assert.Equal(t, 0, l.Tally[chronicle.Boss])
	assert.NotEmpty(t, r.w.pushes, "the web tab is refreshed")

	empty, err := c.Capture(99)
	require.NoError(t, err)
	assert.Nil(t, empty)
	require.NoError(t, c.Restore(7, 0, nil))
	assert.Empty(t, r.m.Log(7).Entries, "restoring nothing clears the record")
	assert.Contains(t, userstate.Names(), "chronicle", "the module registered with the test area")
}

func TestTheLeadersOwnDeathIsRecordedWithTheKillerAndPlace(t *testing.T) {
	r := newRig(t)
	r.w.nameFor = "Wren"
	r.m.onPlayerDeath(events.PlayerDeath{UserId: 7, RoomId: 30, CharacterName: "wren", KillerMobId: 12})
	e := r.m.Log(7).Entries[0]
	assert.Equal(t, chronicle.Fell, e.Kind)
	assert.Equal(t, []string{"Wren"}, e.Members)
	assert.Equal(t, "a wolf", e.Subject)
	assert.Equal(t, "the ford", e.Place)
	assert.Equal(t, "Wren fell to a wolf at the ford.", chronicle.Prose(e))

	r.m.onPlayerDeath(events.PlayerDeath{UserId: 7, RoomId: 30, CharacterName: "wren"})
	assert.Equal(t, "Wren fell at the ford.", chronicle.Prose(r.m.Log(7).Entries[1]), "an unknown killer is left out")
}

func TestThePackageSeamReachesTheModule(t *testing.T) {
	r := newRig(t)
	chronicle.SetProvider(r.m)
	t.Cleanup(func() { chronicle.SetProvider(module) })
	chronicle.Record(7, chronicle.Entry{Kind: chronicle.Relic, Subject: "the Pale Crown", Ref: "item:1"})
	assert.True(t, chronicle.Has(7, chronicle.Filter{Ref: "item:1"}))
	assert.Equal(t, 1, chronicle.Total(7, chronicle.Relic))
	assert.NotNil(t, module, "init installed the module as the provider")
}

func (r *rig) fill(n int) {
	for i := 0; i < n; i++ {
		r.now = r.now.Add(time.Hour)
		r.m.Record(7, chronicle.Entry{Kind: chronicle.Joined, Members: []string{"Mara"}})
	}
}

func TestChronicleCommandReadsInPagesAndByKind(t *testing.T) {
	r := newRig(t)
	empty := r.m.render(7, "")
	assert.Contains(t, empty, "chronicle is empty")
	assert.Contains(t, empty, "help chronicle")

	r.fill(13)
	r.now = r.now.Add(2 * time.Hour)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "the Hollow King", Ref: "mob:9"})
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Fell, Members: []string{"Mara"}, Subject: "a wolf"})

	page1 := r.m.render(7, "")
	assert.Contains(t, page1, "Mara fell to a wolf")
	assert.Contains(t, page1, "The company slew the Hollow King at the Waymark Inn.")
	assert.Equal(t, 12, strings.Count(page1, "\n  <ansi"), "twelve deeds a page")
	assert.Contains(t, page1, "Page 1 of 2")
	assert.Contains(t, page1, "All told: 13 joined, 1 fallen, 1 bosses.", "the tally counts every deed")
	assert.Less(t, strings.Index(page1, "wolf"), strings.Index(page1, "Hollow King"), "newest first")

	page2 := r.m.render(7, "2")
	assert.Contains(t, page2, "Page 2 of 2")
	assert.NotContains(t, page2, "wolf")

	assert.Equal(t, r.m.render(7, "99"), page2, "a page past the end shows the last")

	boss := r.m.render(7, "boss")
	assert.Contains(t, boss, "(bosses)")
	assert.Contains(t, boss, "Hollow King")
	assert.NotContains(t, boss, "joined")
	assert.NotContains(t, boss, "All told", "a filtered view has no tally")

	all := r.m.render(7, "all")
	assert.Equal(t, 15, strings.Count(all, "\n  <ansi"), "every deed on one page")
	assert.Contains(t, all, "15 deed(s) kept")
	assert.Contains(t, r.m.render(7, "all joined"), "13 deed(s) kept")

	assert.Contains(t, r.m.render(7, "relics"), "Nothing of that kind", "a kind with no deeds says so")
	bad := r.m.render(7, "gossip")
	assert.Contains(t, bad, `no deeds called "gossip"`)
	assert.Contains(t, bad, "Usage: chronicle")
}

func TestThePanelCarriesTheNewestDeedsTheTallyAndTheKindNames(t *testing.T) {
	r := newRig(t)
	r.fill(70)
	r.m.Record(7, chronicle.Entry{Kind: chronicle.Relic, Subject: "the Pale Crown", Detail: "the Hollow King"})
	p := r.m.panelFor(7)
	assert.Equal(t, 71, p.Total)
	assert.Equal(t, 70, p.Tally["joined"])
	require.Len(t, p.Entries, panelEntries)
	assert.Equal(t, "relic", p.Entries[0].Kind)
	assert.Equal(t, "Relics", p.Entries[0].Label)
	assert.Equal(t, "just now", p.Entries[0].Ago)
	assert.Contains(t, p.Entries[0].Text, "took the Pale Crown from the Hollow King")
	assert.Equal(t, "joined", p.Entries[1].Kind)

	none := r.m.panelFor(99)
	assert.Equal(t, 0, none.Total)
	assert.NotNil(t, none.Entries, "an empty chronicle is an empty list, not null")
}
