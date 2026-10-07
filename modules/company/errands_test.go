package company

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/errands"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// fakeErrandWorld stands in for the game: an inn in a 7-9 zone, a leader
// who is free, one lair, and a record of what the leader was paid.
type fakeErrandWorld struct {
	place     errandPlace
	inWorld   bool
	free      bool
	busy      string
	lairs     []lair
	values    map[int]int
	paidGold  int
	paidItems []items.Item
	payCalls  int
}

func newFakeErrandWorld() *fakeErrandWorld {
	return &fakeErrandWorld{
		place:   errandPlace{RoomID: 5, Zone: "Brindle Downs", Town: true, BandLow: 7, BandHigh: 9},
		inWorld: true, free: true,
		lairs:  []lair{{MobID: 301, Name: "Old Grindle"}},
		values: map[int]int{30007: 20, 30018: 3, 99999: 500},
	}
}

func (w *fakeErrandWorld) Place(int) (errandPlace, bool) { return w.place, w.inWorld }
func (w *fakeErrandWorld) Free(int) (int, bool)          { return w.place.RoomID, w.free }
func (w *fakeErrandWorld) Lairs(string) []lair           { return w.lairs }
func (w *fakeErrandWorld) ItemValue(id int) (int, bool) {
	v, ok := w.values[id]
	return v, ok
}
func (w *fakeErrandWorld) Pay(_, gold int, item *items.Item) bool {
	w.payCalls++
	w.paidGold += gold
	if item != nil {
		w.paidItems = append(w.paidItems, *item)
	}
	return true
}
func (w *fakeErrandWorld) Busy(*users.UserRecord) string { return w.busy }

// errandModule is relocationModule with companion 2 and 3 free to send (1
// stands in the formation), a clock the test sets, and the fake world.
func errandModule(t *testing.T) (*CompanyModule, *fakeRuntime, *fakeErrandWorld, *time.Time) {
	t.Helper()
	module, runtime := relocationModule(t)
	runtime.nextInstanceID = 200
	runtime.vitals = map[int][2]int{102: {40, 60}, 103: {40, 60}}
	now := time.Unix(1_000_000, 0)
	module.clock = func() time.Time { return now }
	world := newFakeErrandWorld()
	module.errandSeam = world
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	record, _ := module.registry.Get(7)
	for i := range record.Companions {
		record.Companions[i].State = &domain.MemberState{Level: 8}
	}
	module.registry.Put(record)
	return module, runtime, world, &now
}

func errandOf(t *testing.T, m *CompanyModule, id int) *errands.Errand {
	t.Helper()
	record, _ := m.registry.Get(7)
	c, ok := findCompanion(record, id)
	require.True(t, ok)
	return c.Errand
}

func send(t *testing.T, m *CompanyModule, world *fakeErrandWorld, id int, kind errands.Kind, length errands.Length) string {
	t.Helper()
	record, _ := m.registry.Get(7)
	c, _ := findCompanion(record, id)
	text, err := m.startErrand(7, c, kind, length, world.place)
	require.NoError(t, err)
	return text
}

func TestStartErrandSavesThenRemovesTheMob(t *testing.T) {
	module, runtime, world, now := errandModule(t)

	text := send(t, module, world, 2, errands.Hunt, errands.Medium)

	e := errandOf(t, module, 2)
	require.NotNil(t, e)
	assert.Equal(t, errands.Hunt, e.Kind)
	assert.Equal(t, now.Unix()+2*3600, e.ReturnsAt)
	assert.Equal(t, 8, e.Level)
	assert.Equal(t, 7, e.BandLow)
	assert.Equal(t, 60, e.MaxHealth, "its health maximum is kept for the size of a wound")
	assert.Equal(t, "Brindle Downs", e.Zone)
	_, tracked := module.instance(7, 2)
	assert.False(t, tracked, "its mob is gone")
	assert.Equal(t, 1, runtime.detachCalls)
	saved := module.store.(*fakeStore).saved.Companies[7]
	c, _ := findCompanion(saved, 2)
	require.NotNil(t, c.Errand, "the errand is on disk before the mob leaves")
	assert.Contains(t, text, "two hours")
}

func TestFailedSaveLeavesTheCompanionStanding(t *testing.T) {
	module, runtime, world, _ := errandModule(t)
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	record, _ := module.registry.Get(7)
	c, _ := findCompanion(record, 2)

	_, err := module.startErrand(7, c, errands.Escort, errands.Short, world.place)

	require.Error(t, err)
	assert.Nil(t, errandOf(t, module, 2))
	_, tracked := module.instance(7, 2)
	assert.True(t, tracked)
	assert.Zero(t, runtime.detachCalls)
}

func TestErrandRefusals(t *testing.T) {
	module, runtime, _, _ := errandModule(t)
	cases := map[string]struct {
		id     int
		mutate func()
		want   string
	}{
		"in the formation": {1, func() {}, "stands in your formation"},
		"away already": {2, func() {
			r, _ := module.registry.Get(7)
			r.Companions[1].Errand = &errands.Errand{Kind: errands.Scout, Length: errands.Short, ReturnsAt: 1_000_000 + 600}
			module.registry.Put(r)
		}, "already on an errand"},
		"dead": {3, func() {
			r, _ := module.registry.Get(7)
			r.Companions[2].Death = &domain.CompanionDeath{Remaining: 10}
			module.registry.Put(r)
		}, "fallen"},
		"fighting": {4, func() { runtime.fighting = map[int]bool{104: true} }, "busy fighting"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.mutate()
			r, _ := module.registry.Get(7)
			c, _ := findCompanion(r, tc.id)
			assert.Contains(t, module.errandRefusal(7, r, c), tc.want)
		})
	}
	// a companion who is not with the leader
	runtime.fighting = nil
	runtime.away = map[int]bool{104: true}
	r, _ := module.registry.Get(7)
	c, _ := findCompanion(r, 4)
	assert.Contains(t, module.errandRefusal(7, r, c), "isn't here with you")
}

func TestAwayCompanionIsAbsentEverywhere(t *testing.T) {
	module, runtime, world, _ := errandModule(t)
	send(t, module, world, 2, errands.Escort, errands.Long)

	// roster and survival
	var away bool
	for _, ref := range module.Roster(7) {
		if ref.Name != "" && ref.Away {
			away = true
		}
	}
	assert.True(t, away, "survival spends and recovers nothing for it")
	// the member list
	members, ok := module.CompanyMembers(7)
	require.True(t, ok)
	var status domain.MemberStatus = -1
	for _, mv := range members {
		if mv.ID == 2 {
			status = mv.Status
		}
	}
	assert.Equal(t, domain.MemberErrand, status)
	// formation
	assert.ErrorIs(t, module.registry.PlaceMember(7, domain.CompanionMemberKey(2), 2, 1), domain.ErrMemberAway)
	assert.ErrorIs(t, module.registry.SwapMembers(7, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2)), domain.ErrMemberAway)
	// a relog or restart never brings it back early
	runtime.spawnCalls = 0
	require.NoError(t, module.restoreForLeader(7, 5))
	assert.Zero(t, runtime.spawnCalls)
	_, tracked := module.instance(7, 2)
	assert.False(t, tracked)
	// the separation machinery leaves it alone
	assert.ErrorIs(t, module.separate(7, 2, domain.SeparatedStray), domain.ErrUnknownMember)
	module.tickSeparations()
	assert.Nil(t, separationOf(t, module, 2))
	// the status line
	assert.Contains(t, module.status(7), "away escorting; back in")
	// the company's conditions
	conds, _ := module.CompanyConditions(7)
	for _, c := range conds {
		if c.Key == domain.CompanionMemberKey(2) {
			assert.Equal(t, "errand", c.State)
		}
	}
}

func TestErrandReturnsOnlyWhenDueOnlineAndFree(t *testing.T) {
	module, runtime, world, now := errandModule(t)
	send(t, module, world, 2, errands.Escort, errands.Short)
	chem := module.chem.(*fakeChemWorld)

	module.tickErrands()
	assert.NotNil(t, errandOf(t, module, 2), "not due")

	*now = now.Add(31 * time.Minute)
	world.free = false
	module.tickErrands()
	assert.NotNil(t, errandOf(t, module, 2), "due, but the leader is in a fight")
	world.free = true
	chem.online[7] = false
	module.tickErrands()
	assert.NotNil(t, errandOf(t, module, 2), "due, but the leader is offline: it waits for the login")
	assert.Zero(t, runtime.spawnCalls)

	chem.online[7] = true
	module.tickErrands()

	assert.Nil(t, errandOf(t, module, 2))
	assert.Equal(t, 1, runtime.spawnCalls, "it spawns beside the leader")
	_, tracked := module.instance(7, 2)
	assert.True(t, tracked)
	assert.Contains(t, strings.Join(chem.told[7], "\n"), "is back from")
	saved := module.store.(*fakeStore).saved.Companies[7]
	c, _ := findCompanion(saved, 2)
	assert.Nil(t, c.Errand, "the return is saved")
	// the deed is in the chronicle
	deeds := chronicle.Query(7, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Errand}})
	require.Len(t, deeds, 1)
	assert.Equal(t, "errand:escort", deeds[0].Ref)
	assert.Equal(t, []string{string(domain.CompanionMemberKey(2))}, deeds[0].Keys)
	assert.Contains(t, chronicle.Prose(deeds[0]), "came back from an escort job at Brindle Downs")
	assert.NotContains(t, deeds[0].Detail, "<ansi", "the chronicle is plain text")

	module.tickErrands()
	assert.Equal(t, 1, runtime.spawnCalls, "a second tick changes nothing")
}

// seedFor finds a seed that resolves e the wanted way.
func seedFor(t *testing.T, e errands.Errand, want errands.OutcomeKind, hasLair bool) int64 {
	t.Helper()
	for seed := int64(1); seed < 100000; seed++ {
		e.Seed = seed
		if errands.Resolve(e, hasLair).Kind == want {
			return seed
		}
	}
	t.Fatalf("no seed resolves to %s", want)
	return 0
}

// forceErrand puts companion id away with an errand that resolves to want.
func forceErrand(t *testing.T, m *CompanyModule, id int, want errands.OutcomeKind, level int) {
	t.Helper()
	e := errands.New(errands.Hunt, errands.Short, "Brindle Downs", m.now().Unix(), level, 7, 9, 60, 1)
	e.Seed = seedFor(t, e, want, true)
	record, _ := m.registry.Get(7)
	for i := range record.Companions {
		if record.Companions[i].ID == id {
			record.Companions[i].Errand = &e
		}
	}
	m.registry.Put(record)
	m.clearInstance(7, id)
}

func TestGoldOutcomePaysThePayAndNothingElse(t *testing.T) {
	module, _, world, now := errandModule(t)
	forceErrand(t, module, 2, errands.Gold, 8)
	e := errandOf(t, module, 2)
	*now = now.Add(time.Duration(e.ReturnsAt-now.Unix()+1) * time.Second)

	module.tickErrands()

	assert.Equal(t, errands.Pay(*e), world.paidGold)
	assert.Empty(t, world.paidItems)
}

func TestItemOutcomeNeverWorthMoreThanTheGold(t *testing.T) {
	module, _, world, now := errandModule(t)
	world.values[99999] = 500
	module.plug = nil
	module.errandSeam = world
	for _, id := range []int{2, 3} {
		forceErrand(t, module, id, errands.Item, 8)
	}
	*now = now.Add(3 * time.Hour)
	// pool comes from config, which the unit module has none of: serve one.
	pool := []int{30007, 30018, 99999}
	record, _ := module.registry.Get(7)
	for _, c := range record.Companions {
		if !c.OnErrand() {
			continue
		}
		r := module.resolveErrand(7, world, pool, c)
		budget := errands.Pay(*c.Errand)
		switch r.outcome.Kind {
		case errands.Item:
			require.NotNil(t, r.item)
			assert.LessOrEqual(t, world.values[r.item.ItemId], budget, "an item is never worth more than the gold it replaces")
			assert.NotEqual(t, 99999, r.item.ItemId)
		case errands.Gold:
			assert.Equal(t, budget, r.gold)
		default:
			t.Fatalf("unexpected outcome %s", r.outcome.Kind)
		}
	}
}

func TestItemWithNothingThatFitsPaysGold(t *testing.T) {
	module, _, world, _ := errandModule(t)
	forceErrand(t, module, 2, errands.Item, 8)
	record, _ := module.registry.Get(7)
	c, _ := findCompanion(record, 2)
	r := module.resolveErrand(7, world, []int{99999}, c)
	assert.Equal(t, errands.Gold, r.outcome.Kind)
	assert.Equal(t, errands.Pay(*c.Errand), r.gold)
	assert.Nil(t, r.item)
}

func TestWoundOutcomeIsSavedWithTheReturn(t *testing.T) {
	module, _, world, now := errandModule(t)
	forceErrand(t, module, 2, errands.Wound, 3) // far under the 7-9 band
	e := errandOf(t, module, 2)
	*now = now.Add(time.Duration(e.ReturnsAt-now.Unix()+1) * time.Second)

	module.tickErrands()

	saved := module.store.(*fakeStore).saved.Companies[7]
	c, _ := findCompanion(saved, 2)
	require.NotNil(t, c.State)
	require.Len(t, c.State.Wounds, 1, "the wound is in the same save that brings it home")
	assert.Equal(t, 12, c.State.Wounds[0].Points, "20%% of the 60 health it left with")
	assert.False(t, c.State.Wounds[0].Light, "a lasting wound")
	assert.Zero(t, world.paidGold)
	assert.Zero(t, world.payCalls)
	deeds := chronicle.Query(7, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Errand}})
	require.Len(t, deeds, 1)
	assert.Contains(t, deeds[0].Detail, "wound")
}

func TestRumourNamesAnUnslainLair(t *testing.T) {
	module, _, world, _ := errandModule(t)
	world.lairs = []lair{{MobID: 301, Name: "Old Grindle"}, {MobID: 302, Name: "Mother Thorn"}}
	chronicle.Record(7, chronicle.Entry{Kind: chronicle.Boss, Subject: "Old Grindle", Ref: "mob:301"})
	forceErrand(t, module, 2, errands.Rumour, 8)
	record, _ := module.registry.Get(7)
	c, _ := findCompanion(record, 2)

	for roll := 0; roll < 4; roll++ {
		name, _ := rumourOf(world.lairs, roll, lairSlain(7))
		assert.Equal(t, "Mother Thorn", name, "the lair the company already slew is not news")
	}
	r := module.resolveErrand(7, world, nil, c)
	assert.Equal(t, errands.Rumour, r.outcome.Kind)
	assert.Contains(t, r.what, "Mother Thorn")
	assert.Zero(t, r.gold)
}

func TestTwoErrandsFinishInOneSave(t *testing.T) {
	module, runtime, world, now := errandModule(t)
	forceErrand(t, module, 2, errands.Gold, 8)
	forceErrand(t, module, 3, errands.Gold, 8)
	*now = now.Add(2 * time.Hour)
	store := module.store.(*fakeStore)
	store.saveCalls = 0

	module.tickErrands()

	assert.Equal(t, 2, runtime.spawnCalls)
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, 2, world.payCalls)
}

func TestFailedSaveKeepsTheErrandForNextRound(t *testing.T) {
	module, runtime, world, now := errandModule(t)
	forceErrand(t, module, 2, errands.Gold, 8)
	*now = now.Add(2 * time.Hour)
	module.store.(*fakeStore).saveErr = errors.New("disk full")

	module.tickErrands()

	assert.NotNil(t, errandOf(t, module, 2))
	assert.Zero(t, runtime.spawnCalls)
	assert.Zero(t, world.payCalls, "no reward until the return is on disk")

	module.store.(*fakeStore).saveErr = nil
	module.tickErrands()
	assert.Nil(t, errandOf(t, module, 2))
	assert.Equal(t, 1, world.payCalls, "paid exactly once")
}

func TestOutcomeIsTheSameAfterARestart(t *testing.T) {
	module, _, world, _ := errandModule(t)
	send(t, module, world, 2, errands.Hunt, errands.Long)
	record, _ := module.registry.Get(7)
	c, _ := findCompanion(record, 2)
	before := errands.Resolve(*c.Errand, true)

	data, err := yaml.Marshal(record)
	require.NoError(t, err)
	var loaded domain.Record
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	reloaded, _ := findCompanion(loaded, 2)

	require.NotNil(t, reloaded.Errand)
	assert.Equal(t, *c.Errand, *reloaded.Errand)
	assert.Equal(t, before, errands.Resolve(*reloaded.Errand, true), "a restart cannot reroll the outcome")
}

func TestRecallBringsThemBackWithNothing(t *testing.T) {
	module, runtime, world, _ := errandModule(t)
	send(t, module, world, 2, errands.Hunt, errands.Long)
	record, _ := module.registry.Get(7)
	c, _ := findCompanion(record, 2)

	text, err := module.recallErrand(7, c)

	require.NoError(t, err)
	assert.Contains(t, text, "returns with nothing")
	assert.Nil(t, errandOf(t, module, 2))
	assert.Equal(t, 1, runtime.spawnCalls)
	assert.Zero(t, world.payCalls)
	assert.Empty(t, chronicle.Query(7, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Errand}}), "no deed for a job not done")

	send(t, module, world, 3, errands.Scout, errands.Short)
	world.free = false
	record, _ = module.registry.Get(7)
	c, _ = findCompanion(record, 3)
	text, _ = module.recallErrand(7, c)
	assert.Contains(t, text, "Not while you are fighting")
	assert.NotNil(t, errandOf(t, module, 3))
}

func userFor() *users.UserRecord {
	return &users.UserRecord{UserId: 7, Character: &characters.Character{}}
}

func TestErrandSendCommandParses(t *testing.T) {
	module, _, world, _ := errandModule(t)
	user := userFor()

	text, err := module.errandSend(user, []string{"#2", "hunt", "long"})
	require.NoError(t, err)
	assert.Contains(t, text, "sets out hunting")
	assert.Equal(t, errands.Long, errandOf(t, module, 2).Length)

	text, _ = module.errandSend(user, []string{"#3", "scout"})
	assert.Contains(t, text, "sets out scouting")
	assert.Equal(t, errands.Short, errandOf(t, module, 3).Length, "short is the default")

	text, _ = module.errandSend(user, []string{"#4", "dance"})
	assert.Contains(t, text, "Choose an errand")
	text, _ = module.errandSend(user, []string{"#9", "hunt"})
	assert.Contains(t, text, "no companion like that")
	text, _ = module.errandSend(user, []string{"#1", "hunt"})
	assert.Contains(t, text, "formation")

	world.place.Town = false
	text, _ = module.errandSend(user, []string{"#4", "hunt"})
	assert.Contains(t, text, "from an inn")
	world.place.Town = true
	world.busy = "Not in the middle of a battle."
	text, _ = module.errandSend(user, []string{"#4", "hunt"})
	assert.Contains(t, text, "battle")
	assert.Nil(t, errandOf(t, module, 4))
}

func TestErrandPanelAndView(t *testing.T) {
	module, _, world, _ := errandModule(t)
	send(t, module, world, 2, errands.Scout, errands.Medium)

	panel, ok := module.ErrandPanel(7)
	require.True(t, ok)
	assert.True(t, panel.Here)
	assert.Equal(t, "7-9", panel.Band)
	states := map[int]string{}
	for _, r := range panel.Rows {
		states[r.ID] = r.State
	}
	assert.Equal(t, map[int]string{1: "busy", 2: "away", 3: "ready", 4: "ready"}, states)
	view := module.errandsView(7)
	assert.Contains(t, view, "#2")
	assert.Contains(t, view, "away on a scouting job in Brindle Downs")
	assert.Contains(t, view, "free to send")
	assert.Contains(t, view, "errand send")

	world.place.Town = false
	panel, _ = module.ErrandPanel(7)
	assert.False(t, panel.Here)
	assert.NotContains(t, module.errandsView(7), "errand send [member]")
}
