package storyevents

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = `
- id: cliff
  title: The Cliff
  triggers:
    - kind: room
      room: 10
  pages:
    start:
      text: A rope hangs over the drop.
      choices:
        - label: Send the climber
          require:
            classes: [rogue]
          hint: a rogue
          risk:
            pct: 40
            less_per_level: 0
          text: "{who} goes down."
          do:
            - kind: item
              item: 36
              count: 2
            - kind: gold
              amount: 20
          fail_text: "{who} falls."
          fail:
            - kind: wound
              pct: 20
        - label: Go down together
          text: You all go down.
          do:
            - kind: move
              room: 11
          next: ledge
        - label: Leave
          text: You leave.
    ledge:
      text: A ledge.
      choices:
        - label: Search the pack
          do:
            - kind: flag
              flag: searched
            - kind: need
              who: all
              stat: thirst
              amount: 5
            - kind: loyalty
              who: all
              amount: -3
            - kind: ailment
              who: random
              ailment: chill
            - kind: lose_item
              item: 36
          next: top
        - label: Climb back up
          do:
            - kind: move
              room: 10
    top:
      text: Back at the top.
      choices:
        - label: Done

- id: ambush
  title: The Ambush
  cooldown_minutes: 10
  min_level: 3
  max_level: 20
  triggers:
    - kind: tag
      tag: lair-door
      chance: 50
  pages:
    start:
      text: Something waits.
      choices:
        - label: Kick the door
          risk:
            pct: 95
          text: It opens.
          fail_text: They were waiting.
          fail:
            - kind: battle
              foes:
                - mob: 5
                - mob: 6
                  level: 9
        - label: Walk away

- id: road-camp
  title: A Visitor
  require:
    flag: searched
  triggers:
    - kind: camp
      zone: Wilds
    - kind: arrival
      zone: Wilds
  pages:
    start:
      text: Someone at the fire.
      choices:
        - label: Wave them in
`

type call struct {
	kind string
	args string
}

type fakeWorld struct {
	user    *users.UserRecord
	members []storyevents.Facts
	company storyevents.Company
	rooms   map[int]fakeRoom
	busy    bool
	sent    []string
	pushed  []payload
	calls   []call
	level   int
}

type fakeRoom struct {
	zone string
	tags []string
}

func newFakeWorld() *fakeWorld {
	u := users.NewUserRecord(7, 1)
	u.Character.RoomId = 10
	u.Character.Level = 8
	return &fakeWorld{
		user: u,
		members: []storyevents.Facts{
			{Key: "leader", Name: "Aldous", Leader: true, Archetype: "warrior", Level: 8},
			{Key: "c1", Name: "Sera", Archetype: "rogue", Level: 6},
			{Key: "c2", Name: "Eder", Archetype: "cleric", Level: 5},
		},
		company: storyevents.Company{Gold: 100, Items: map[int]int{36: 3}},
		rooms: map[int]fakeRoom{
			10: {zone: "Test"}, 11: {zone: "Test"},
			20: {zone: "Test", tags: []string{"lair-door"}},
			30: {zone: "Wilds"},
		},
		level: 4,
	}
}

func (f *fakeWorld) Lookups() storyevents.Lookups { return storyevents.Lookups{} }
func (f *fakeWorld) User(id int) *users.UserRecord {
	if id == f.user.UserId {
		return f.user
	}
	return nil
}
func (f *fakeWorld) Members(int) []storyevents.Facts { return f.members }
func (f *fakeWorld) Company(_ int, flags map[string]bool) storyevents.Company {
	c := f.company
	c.Flags = flags
	return c
}
func (f *fakeWorld) Zone(id int) (string, []string, bool) {
	r, ok := f.rooms[id]
	return r.zone, r.tags, ok
}
func (f *fakeWorld) Busy(int, int, bool) bool { return f.busy }
func (f *fakeWorld) FoeLevel(int, int) int    { return f.level }
func (f *fakeWorld) rec(kind, format string, args ...any) string {
	f.calls = append(f.calls, call{kind, fmt.Sprintf(format, args...)})
	return ""
}
func (f *fakeWorld) Wound(_ int, m storyevents.Facts, pct int) string {
	return f.rec("wound", "%s %d", m.Name, pct)
}
func (f *fakeWorld) Ailment(_ int, m storyevents.Facts, kind string) string {
	return f.rec("ailment", "%s %s", m.Name, kind)
}
func (f *fakeWorld) Need(_ int, m storyevents.Facts, stat string, n int) string {
	return f.rec("need", "%s %s %d", m.Name, stat, n)
}
func (f *fakeWorld) AddItems(_ int, op string, item, n int) string {
	return f.rec("item", "%d x%d %s", item, n, op)
}
func (f *fakeWorld) TakeItems(_ int, item, n int) string {
	return f.rec("lose_item", "%d x%d", item, n)
}
func (f *fakeWorld) Gold(_ int, n int) string { return f.rec("gold", "%d", n) }
func (f *fakeWorld) Loyalty(_ int, op string, who []storyevents.Facts, n int) string {
	names := []string{}
	for _, m := range who {
		names = append(names, m.Name)
	}
	return f.rec("loyalty", "%s %d %s", strings.Join(names, ","), n, op)
}
func (f *fakeWorld) Battle(_ int, room int, foes []storyevents.Foe) string {
	return f.rec("battle", "room %d %v", room, foes)
}
func (f *fakeWorld) Move(_ int, room int) string {
	f.user.Character.RoomId = room
	return f.rec("move", "%d", room)
}
func (f *fakeWorld) Send(_ int, text string) { f.sent = append(f.sent, text) }
func (f *fakeWorld) Push(_ int, module string, p any) {
	if module == "Event" {
		f.pushed = append(f.pushed, p.(payload))
	}
}

func (f *fakeWorld) last() string {
	if len(f.sent) == 0 {
		return ""
	}
	return f.sent[len(f.sent)-1]
}

func (f *fakeWorld) kinds() []string {
	out := []string{}
	for _, c := range f.calls {
		out = append(out, c.kind)
	}
	return out
}

type memStore struct {
	saved Registry
	fail  error
	saves int
}

func (s *memStore) Load(r *Registry) error { *r = s.saved; return nil }
func (s *memStore) Save(r Registry) error {
	if s.fail != nil {
		return s.fail
	}
	s.saves++
	// A real store round-trips through YAML: do the same so aliasing shows.
	s.saved = cloneRegistry(r)
	return nil
}

func cloneRegistry(r Registry) Registry {
	out := Registry{Companies: map[int]State{}}
	for id, st := range r.Companies {
		out.Companies[id] = st.clone()
	}
	return out
}

type rig struct {
	m     *Module
	w     *fakeWorld
	store *memStore
	now   time.Time
	rolls []int
}

// newRig builds a module over the fixture with a fake world, a clock the
// test moves, and a roll queue (empty: every roll is 0, so a risk fails and
// a chance passes).
func newRig(t *testing.T, files ...string) *rig {
	t.Helper()
	r := &rig{w: newFakeWorld(), store: &memStore{}, now: time.Unix(1_800_000_000, 0)}
	m := newModule()
	m.w = r.w
	m.store = r.store
	m.clock = func() time.Time { return r.now }
	m.rng = func(n int) int {
		if len(r.rolls) == 0 {
			return 0
		}
		v := r.rolls[0]
		r.rolls = r.rolls[1:]
		return v % n
	}
	src := fixture
	if len(files) > 0 {
		src = files[0]
	}
	m.readFiles = func() map[string][]byte { return map[string][]byte{"test.yaml": []byte(src)} }
	r.m = m
	return r
}

func (r *rig) enter(room int) bool {
	r.w.user.Character.RoomId = room
	r.m.entered(r.w.user.UserId, room, false)
	_, _, ok := r.m.waiting(r.w.user.UserId)
	return ok
}

func (r *rig) pending() *Pending {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	return r.m.state[r.w.user.UserId].Pending
}

func (r *rig) choose(n int) { r.m.choose(r.w.user.UserId, n) }

func TestARoomTriggerOpensTheFirstPageWithItsChoices(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	assert.Equal(t, "start", r.pending().Page)
	text := r.w.last()
	assert.Contains(t, text, "The Cliff")
	assert.Contains(t, text, "A rope hangs over the drop.")
	assert.Contains(t, text, "1</ansi>. Send the climber")
	assert.Contains(t, text, "(Sera, chancy)", "the best qualified member is named, with the risk in words")
	assert.Contains(t, text, "3</ansi>. Leave")
	require.Len(t, r.w.pushed, 1)
	p := r.w.pushed[0]
	assert.True(t, p.Active)
	assert.Equal(t, "cliff", p.ID)
	require.Len(t, p.Choices, 3)
	assert.Equal(t, choiceView{N: 1, Label: "Send the climber", Open: true, Who: "Sera", Risk: "chancy"}, p.Choices[0])
	assert.Equal(t, 1, r.store.saves, "the page is saved the moment it opens")
}

func TestNoSecondSceneWhileOneWaitsAndNoneInAFight(t *testing.T) {
	r := newRig(t)
	r.w.busy = true
	assert.False(t, r.enter(10), "a fight, a rest or a journey holds the scene back")
	r.w.busy = false
	require.True(t, r.enter(10))
	sent := len(r.w.sent)
	r.m.entered(r.w.user.UserId, 10, false)
	assert.Len(t, r.w.sent, sent, "the waiting page is not shown twice")
}

func TestTheCompanyStaysPutWhileAPageWaits(t *testing.T) {
	r := newRig(t)
	blocked, _ := r.m.MovementBlocked(7)
	assert.False(t, blocked)
	require.True(t, r.enter(10))
	blocked, msg := r.m.MovementBlocked(7)
	assert.True(t, blocked)
	assert.Contains(t, msg, "choose")
	r.choose(3)
	blocked, _ = r.m.MovementBlocked(7)
	assert.False(t, blocked, "answering frees the company")
}

func TestAClosedChoiceNeedsAMemberWhoQualifies(t *testing.T) {
	r := newRig(t)
	r.w.members = r.w.members[:1] // no rogue left
	require.True(t, r.enter(10))
	assert.Contains(t, r.w.last(), "(closed: needs a rogue)")
	r.choose(1)
	assert.Contains(t, r.w.last(), "closed to your company: it needs a rogue")
	assert.Equal(t, "start", r.pending().Page, "the page still waits")
	assert.Empty(t, r.w.calls)
}

func TestARiskyChoiceSucceedsOrFailsByTheRoll(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.rolls = []int{99} // over the 40% risk: success
	r.choose(1)
	assert.Contains(t, r.w.last(), "Sera goes down.")
	assert.Equal(t, []string{"gold", "item"}, r.w.kinds())
	assert.Equal(t, "item", r.w.calls[1].kind)
	assert.Contains(t, r.w.calls[1].args, "36 x2 story:cliff:0:1", "an item gain carries an operation id")
	assert.Nil(t, r.pending(), "the scene ends")
	require.Len(t, r.w.pushed, 2)
	assert.False(t, r.w.pushed[1].Active)

	r = newRig(t)
	require.True(t, r.enter(10))
	r.rolls = []int{10} // under 40%: failure
	r.choose(1)
	assert.Contains(t, r.w.last(), "Sera falls.")
	assert.Equal(t, []call{{"wound", "Sera 20"}}, r.w.calls)
}

func TestFreeChoiceFallsToTheLeaderAndALeaveChoiceEndsTheScene(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.choose(3)
	assert.Contains(t, r.w.last(), "You leave.")
	assert.Nil(t, r.pending())
	assert.Empty(t, r.w.calls)
}

func TestAPageLeadsToTheNextAndThroughEveryOutcomeKind(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.choose(2)
	assert.Equal(t, []call{{"move", "11"}}, r.w.calls)
	assert.Equal(t, "ledge", r.pending().Page)
	assert.Equal(t, 11, r.pending().Room, "the scene follows the company to the ledge")
	assert.Contains(t, r.w.last(), "You all go down.")
	assert.Contains(t, r.w.last(), "A ledge.", "the next page follows the result")

	r.w.calls = nil
	r.choose(1)
	assert.Equal(t, "top", r.pending().Page)
	kinds := r.w.kinds()
	assert.Equal(t, []string{"lose_item", "loyalty", "need", "need", "need", "ailment"}, kinds, "outcomes run in a fixed order, quiet ones first")
	assert.Equal(t, call{"loyalty", "Aldous,Sera,Eder -3 story:cliff:1:2"}, r.w.calls[1], "who: all reaches every member")
	assert.Equal(t, call{"need", "Aldous thirst 5"}, r.w.calls[2])
	assert.Equal(t, call{"ailment", "Aldous chill"}, r.w.calls[5], "who: random takes one member by the roll")

	r.choose(1)
	assert.Nil(t, r.pending())
	r.m.mu.Lock()
	st := r.m.state[7]
	r.m.mu.Unlock()
	assert.Equal(t, []string{"searched"}, st.Flags, "a flag outcome sets a company flag")
	assert.Contains(t, st.Done, "cliff")
}

func TestABattleOutcomeNamesTheRoomAndFoes(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(20))
	r.rolls = []int{0} // the risk (95%) fails on a low roll
	r.choose(1)
	require.Len(t, r.w.calls, 1)
	assert.Equal(t, "battle", r.w.calls[0].kind)
	assert.Contains(t, r.w.calls[0].args, "room 20")
	assert.Contains(t, r.w.calls[0].args, "{5 0} {6 9}")
}

func TestAnEventOpensOnceUnlessItHasACooldown(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.choose(3)
	assert.False(t, r.enter(10), "once per company")

	// The ambush has a ten minute cooldown and a level range.
	r.w.user.Character.Level = 2
	assert.False(t, r.enter(20), "below its level range")
	r.w.user.Character.Level = 8
	require.True(t, r.enter(20))
	r.choose(2)
	assert.False(t, r.enter(20), "inside the cooldown")
	r.now = r.now.Add(11 * time.Minute)
	assert.True(t, r.enter(20), "after the cooldown it opens again")
	r.w.user.Character.Level = 99
	r.choose(2)
	r.now = r.now.Add(time.Hour)
	assert.False(t, r.enter(20), "above its level range")
}

func TestATriggerChanceCanMiss(t *testing.T) {
	r := newRig(t)
	r.rolls = []int{80} // 80 >= 50: the chance misses
	assert.False(t, r.enter(20))
	r.rolls = []int{10}
	assert.True(t, r.enter(20))
}

func TestACampAndAnArrivalTriggerNeedTheirZoneAndAFlag(t *testing.T) {
	r := newRig(t)
	r.w.user.Character.RoomId = 30
	r.m.campRestEnded(7, 30)
	_, _, ok := r.m.waiting(7)
	assert.False(t, ok, "the event wants the searched flag first")

	r.m.mu.Lock()
	st := r.m.state[7]
	st.addFlag("searched")
	r.m.state[7] = st
	r.m.mu.Unlock()
	r.m.campRestEnded(7, 30)
	_, _, ok = r.m.waiting(7)
	require.True(t, ok)
	assert.Contains(t, r.w.last(), "Someone at the fire.")
	r.choose(1)

	r2 := newRig(t)
	r2.m.mu.Lock()
	s2 := r2.m.state[7]
	s2.addFlag("searched")
	r2.m.state[7] = s2
	r2.m.mu.Unlock()
	r2.w.user.Character.RoomId = 30
	r2.m.entered(7, 30, false)
	_, _, ok = r2.m.waiting(7)
	assert.False(t, ok, "an arrival trigger does not fire on an ordinary step")
	r2.m.entered(7, 30, true)
	_, _, ok = r2.m.waiting(7)
	assert.True(t, ok, "it fires when a journey ends")
}

func TestAPageSurvivesARestartAndIsShownAgainAtLogin(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.choose(2) // on the ledge now

	// A new module over the same store: the restart.
	r2 := newRig(t)
	r2.store.saved = cloneRegistry(r.store.saved)
	r2.w.user.Character.RoomId = 11
	r2.m.load()
	_, p, ok := r2.m.waiting(7)
	require.True(t, ok)
	assert.Equal(t, "ledge", p.Page)
	r2.m.onPlayerSpawn(events.PlayerSpawn{UserId: 7})
	assert.Contains(t, r2.w.last(), "A ledge.")
	assert.True(t, r2.w.pushed[len(r2.w.pushed)-1].Active)
	r2.choose(1)
	assert.Equal(t, "top", r2.pending().Page)
}

func TestAPageForARoomTheCompanyLeftHasPassed(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.w.user.Character.RoomId = 11 // a death, a teleport
	_, _, ok := r.m.waiting(7)
	assert.False(t, ok)
	assert.Nil(t, r.pending(), "the page is dropped")
	r.m.mu.Lock()
	_, done := r.m.state[7].Done["cliff"]
	r.m.mu.Unlock()
	assert.False(t, done, "and not counted as done")
	assert.True(t, r.enter(10), "so the scene can open again")
}

func TestAnAnswerIsSavedBeforeItIsApplied(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	r.store.fail = errors.New("disk full")
	r.rolls = []int{99}
	r.choose(1)
	assert.Empty(t, r.w.calls, "nothing is applied when the answer cannot be saved")
	assert.Contains(t, r.w.last(), "the moment holds")
	require.NotNil(t, r.pending())
	assert.Equal(t, "start", r.pending().Page)
	r.store.fail = nil
	r.choose(3)
	assert.Nil(t, r.pending())
}

func TestAFailedSaveStopsAnEventOpening(t *testing.T) {
	r := newRig(t)
	r.store.fail = errors.New("disk full")
	assert.False(t, r.enter(10))
	assert.Nil(t, r.pending())
}

func TestChooseRefusesBadAnswers(t *testing.T) {
	r := newRig(t)
	r.choose(1)
	assert.Contains(t, r.w.last(), "No scene is waiting")
	require.True(t, r.enter(10))
	r.choose(0)
	assert.Contains(t, r.w.last(), "Choose a number from 1 to 3")
	r.choose(9)
	assert.Contains(t, r.w.last(), "Choose a number from 1 to 3")
	r.w.busy = true
	r.choose(3)
	assert.Contains(t, r.w.last(), "middle of a fight")
	assert.NotNil(t, r.pending())
}

func TestEventCommandShowsTheWaitingPageAgain(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	sent := len(r.w.sent)
	assert.True(t, r.m.reshow(7, ""))
	assert.Len(t, r.w.sent, sent+1)
	r.choose(3)
	assert.False(t, r.m.reshow(7, ""))
}

func TestBrokenEventsAreDisabledNotFatal(t *testing.T) {
	r := newRig(t, `
- id: broken
  title: Broken
  triggers:
    - kind: room
      room: 10
  pages:
    start:
      text: x
      choices:
        - label: Only a gated way
          require:
            classes: [rogue]
- id: fine
  title: Fine
  triggers:
    - kind: room
      room: 10
  pages:
    start:
      text: ok
      choices:
        - label: Leave
`)
	cat := r.m.events()
	assert.Equal(t, []string{"fine"}, cat.IDs())
	require.True(t, r.enter(10))
	assert.Equal(t, "fine", r.pending().Event)

	r2 := newRig(t, "not: [valid")
	assert.Equal(t, 0, r2.m.events().Len())
}

func TestPurgeAndTestAreaSnapshotsCoverTheState(t *testing.T) {
	r := newRig(t)
	require.True(t, r.enter(10))
	c := stateContributor{r.m}
	data, err := c.Capture(7)
	require.NoError(t, err)
	require.NotNil(t, data)
	r.choose(3)
	require.Nil(t, r.pending())

	r.w.pushed = nil
	require.NoError(t, c.Restore(7, 10, data))
	require.NotNil(t, r.pending(), "restore puts the waiting page back")
	require.Len(t, r.w.pushed, 1)
	assert.False(t, r.w.pushed[0].Active, "and closes any modal left open")

	require.NoError(t, c.Restore(7, 10, nil))
	assert.Nil(t, r.pending())

	require.True(t, r.enter(10))
	r.m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Nil(t, r.pending())
	r.m.mu.Lock()
	_, held := r.m.state[7]
	r.m.mu.Unlock()
	assert.False(t, held)
}
