package gathering

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gathering"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// item ids used by the tests (the shipped ones).
const (
	idMeat     = 29
	idFirewood = 40
	idDamp     = 41
	idLine     = 42
	idFish     = 43
	idWeed     = 44
	idThyme    = 30018
	idHide     = 28
)

type fakeStore struct {
	mu        sync.Mutex
	saved     gathering.Ledger
	saveCalls int
	saveErr   error
	loadErr   error
}

func (f *fakeStore) Load(l *gathering.Ledger) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loadErr != nil {
		return f.loadErr
	}
	*l = f.saved.Clone()
	if l.Rooms == nil {
		l.Rooms = map[int]map[gathering.Kind]gathering.Pool{}
	}
	return nil
}

func (f *fakeStore) Save(l gathering.Ledger) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = l.Clone()
	return nil
}

// world is one leader in one outdoor room, with every seam stubbed.
type world struct {
	t     *testing.T
	m     *GatheringModule
	store *fakeStore
	user  *users.UserRecord
	room  *rooms.Room
	now   time.Time

	rolls []int // rng script; 99 forever after it runs out

	battle, resting, travelling, follower bool
	dark                                  bool
	weather                               string
	forager                               *archetypes.Specialist
	fieldSmith                            bool
	weapons                               []items.Item // what the company wields
	scribe                                bool
	bag                                   map[int]int // the company's carried supplies
	cargoErr                              error       // ErrNoCargo by default: the leader's pack
	cargo                                 map[int]int
	full                                  bool
	efforts                               []int
	encounters                            []int
	encounterRooms                        []int
}

func installSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: idMeat, Name: "raw game meat", Weight: 300},
		{ItemId: idHide, Name: "wolf hide", Weight: 800},
		{ItemId: idFirewood, Name: "firewood bundle", Weight: 2000},
		{ItemId: idDamp, Name: "damp firewood bundle", Weight: 2400},
		{ItemId: idLine, Name: "fishing line", Weight: 50},
		{ItemId: idFish, Name: "raw fish", Weight: 300},
		{ItemId: idWeed, Name: "bitter weed", Weight: 10},
		{ItemId: idThyme, Name: "wild thyme", Weight: 20},
		{ItemId: 30007, Name: "mushroom", Weight: 50},
		{ItemId: 30016, Name: "spotted mushroom", Weight: 50},
		{ItemId: 30008, Name: "goldenbell", Weight: 20},
		{ItemId: 30009, Name: "glacial mint", Weight: 20},
		{ItemId: 30010, Name: "moonshade leaf", Weight: 20},
		{ItemId: 200, Name: "bear hide", Weight: 900},
		{ItemId: 204, Name: "snowcat pelt", Weight: 700},
		{ItemId: 20001, Name: "short bow", Type: items.Weapon, Subtype: "shooting"},
		{ItemId: 20002, Name: "hand axe", Type: items.Weapon, Subtype: "cleaving"},
		{ItemId: 20003, Name: "hunting knife", Type: items.Weapon, Subtype: "stabbing"},
		{ItemId: 20004, Name: "mace", Type: items.Weapon, Subtype: "bludgeoning"},
	} {
		items.SetTestItemSpec(spec)
	}
}

func shippedSettings(t *testing.T) gathering.Settings {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	return parseSettings(func(k string) any { return cfg[k] })
}

func newWorld(t *testing.T, resources ...string) *world {
	t.Helper()
	mudlog.SetupLogger(nil, "", "", false)
	installSpecs(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)

	w := &world{t: t, store: &fakeStore{}, now: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC),
		bag: map[int]int{}, cargo: nil, cargoErr: encumbrance.ErrNoCargo}
	w.user = users.NewUserRecord(7, 1)
	w.user.Character.Name = "Hero"
	w.user.Character.Health = 10
	w.user.Character.RoomId = 5001
	users.SetTestUser(w.user)

	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gatherbiome", Name: "Wood", LitArea: false})
	t.Cleanup(func() { rooms.RemoveTestBiome("gatherbiome") })
	w.room = &rooms.Room{RoomId: 5001, Zone: "Old Kings Road", Biome: "gatherbiome", Resources: resources}
	rooms.SetTestRoom(w.room)
	t.Cleanup(func() { rooms.RemoveTestRoom(5001) })

	m := newModule()
	m.store = w.store
	m.settings = shippedSettings(t)
	m.clock = func() time.Time { return w.now }
	m.rng = func(n int) int {
		v := 99
		if len(w.rolls) > 0 {
			v, w.rolls = w.rolls[0], w.rolls[1:]
		}
		if v >= n {
			v = n - 1
		}
		return v
	}
	m.lookupUser = users.GetByUserId
	m.loadRoom = func(id int) *rooms.Room {
		if id == w.room.RoomId {
			return w.room
		}
		return rooms.LoadRoom(id)
	}
	m.inBattle = func(*users.UserRecord) bool { return w.battle }
	m.resting = func(int) bool { return w.resting }
	m.travelling = func(int) bool { return w.travelling }
	m.follower = func(int) bool { return w.follower }
	m.weatherIn = func(string) (weather.Condition, bool) {
		if w.weather == "" {
			return weather.Condition{}, false
		}
		return weather.Condition{Name: w.weather}, true
	}
	m.dark = func(*users.UserRecord, *rooms.Room) bool { return w.dark }
	m.specialist = func(_ int, utility string, _ ...int) (archetypes.Specialist, bool) {
		switch utility {
		case archetypes.UtilityForage:
			if w.forager != nil {
				return *w.forager, true
			}
		case archetypes.UtilityFieldSmith:
			return archetypes.Specialist{Name: "Brann", Level: 2}, w.fieldSmith
		}
		return archetypes.Specialist{}, false
	}
	m.members = func(u *users.UserRecord) []*characters.Character {
		c := characters.New()
		if len(w.weapons) > 0 {
			c.Equipment.Weapon = w.weapons[0]
		}
		if len(w.weapons) > 1 {
			c.Equipment.Offhand = w.weapons[1]
		}
		if w.scribe {
			c.Name = "Scribe"
		}
		return []*characters.Character{u.Character, c}
	}
	m.scribeRank = func(c *characters.Character) int {
		if c.Name == "Scribe" {
			return 1
		}
		return 0
	}
	m.itemCount = func(_ int, id int) int { return w.bag[id] }
	m.spendItem = func(_ int, id int) bool {
		if w.bag[id] < 1 {
			return false
		}
		w.bag[id]--
		return true
	}
	m.effort = func(_ int, _ int, pct int) { w.efforts = append(w.efforts, pct) }
	m.encounter = func(_ int, roomID int, bonus int) bool {
		w.encounters = append(w.encounters, bonus)
		w.encounterRooms = append(w.encounterRooms, roomID)
		return false
	}
	m.wouldExceed = func(int, int) bool { return w.full }
	m.deposit = func(_ int, _ string, stacks []encumbrance.CargoStack) error {
		if w.cargoErr != nil {
			return w.cargoErr
		}
		for _, s := range stacks {
			if w.cargo == nil {
				w.cargo = map[int]int{}
			}
			w.cargo[s.ItemId] += s.Count
		}
		return nil
	}
	m.giveItem = func(u *users.UserRecord, id int) { u.Character.StoreItem(items.New(id)) }
	w.m = m
	return w
}

func (w *world) packCount(id int) int {
	n := 0
	for _, itm := range w.user.Character.Items {
		if itm.ItemId == id {
			n++
		}
	}
	return n
}

// text runs the events queue and returns what the user was told.
func (w *world) heard(f func()) string {
	var lines []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		lines = append(lines, e.(events.Message).Text)
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, id)
	f()
	events.ProcessEvents()
	return strings.Join(lines, "")
}

// run types a command and ticks the module's clock past the work.
func (w *world) command(verb, rest string) string {
	var out string
	w.t.Helper()
	switch verb {
	case "gather":
		w.heard(func() { _, _ = w.m.gatherCommand(rest, w.user, w.room, 0) })
		return w.heard(func() {})
	}
	return out
}

func (w *world) start(kind gathering.Kind) string {
	return w.m.start(w.user, w.room, kind)
}

// finishAfter advances the clock past the kind's duration and ticks.
func (w *world) finishAfter(kind gathering.Kind) string {
	w.now = w.now.Add(w.m.settings.Rules[kind].Duration)
	return w.heard(w.m.tick)
}

// --- refusals ---

func TestGatheringIsRefusedWhenItCannotStart(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *world)
		kind  gathering.Kind
		want  string
	}{
		{"no herbs here", func(w *world) { w.room.Resources = []string{"firewood"} }, gathering.Herbs, "no herbs"},
		{"no deadfall", func(w *world) { w.room.Resources = []string{"herbs"} }, gathering.Firewood, "no deadfall"},
		{"nowhere to fish", func(w *world) { w.room.Resources = []string{"herbs"} }, gathering.Fishing, "nowhere to fish"},
		{"no game", func(w *world) { w.room.Resources = []string{"herbs"} }, gathering.Game, "no game"},
		{"in battle", func(w *world) { w.battle = true }, gathering.Herbs, "middle of a battle"},
		{"resting", func(w *world) { w.resting = true }, gathering.Herbs, "while you are resting"},
		{"travelling", func(w *world) { w.travelling = true }, gathering.Herbs, "while you are travelling"},
		{"a follower", func(w *world) { w.follower = true }, gathering.Herbs, "leader of the party"},
		{"downed", func(w *world) { w.user.Character.Health = 0 }, gathering.Herbs, "no state"},
		{"no fishing line", func(w *world) { w.room.Resources = []string{"fishing"} }, gathering.Fishing, "need a fishing line"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t, "herbs", "firewood")
			c.setup(w)
			got := w.start(c.kind)
			assert.Contains(t, got, c.want)
			_, busy := w.m.Active(7)
			assert.False(t, busy, "a refusal starts nothing")
		})
	}
}

func TestFishingWorksFromShoreWater(t *testing.T) {
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "shore", Name: "Shore", LitArea: false})
	t.Cleanup(func() { rooms.RemoveTestBiome("shore") })
	w := newWorld(t, "water")
	w.bag[idLine] = 1
	assert.Contains(t, w.start(gathering.Fishing), "nowhere to fish", "water in the woods is not a fishing spot")
	w.room.Biome = "shore"
	assert.Contains(t, w.start(gathering.Fishing), "sets about fishing", "water on a shore is")
}

// --- the timed action ---

func TestGatherHerbsTakesRealTimeThenYields(t *testing.T) {
	w := newWorld(t, "herbs")
	w.forager = &archetypes.Specialist{Name: "Hero", IsLeader: true, Level: 4}
	w.rolls = []int{1, 0, 0, 0, 0} // 2 base + 2 level = 4 thyme

	roundBefore, turnBefore := util.GetRoundCount(), util.GetTurnCount()
	started := w.start(gathering.Herbs)
	assert.Contains(t, started, "sets about gathering herbs")
	assert.Contains(t, started, "20 s")
	kind, busy := w.m.Active(7)
	require.True(t, busy)
	assert.Equal(t, gathering.Herbs, kind)

	w.now = w.now.Add(19 * time.Second)
	w.m.tick()
	assert.Zero(t, w.packCount(idThyme), "nothing before the time is up")
	assert.Equal(t, 3, w.m.ledger.Charges(5001, gathering.Herbs, w.m.settings.Rules[gathering.Herbs], w.now), "and no charge spent")

	got := w.finishAfter(gathering.Herbs)
	assert.Equal(t, 4, w.packCount(idThyme), "four wild thyme, from the table")
	assert.Contains(t, got, "wild thyme")
	_, busy = w.m.Active(7)
	assert.False(t, busy)
	assert.Equal(t, []int{100}, w.efforts, "one step's strain for the company")
	assert.Equal(t, []int{0}, w.encounters, "the danger is rolled once")
	assert.Equal(t, 2, w.m.ledger.Charges(5001, gathering.Herbs, w.m.settings.Rules[gathering.Herbs], w.now), "one charge spent")

	assert.Equal(t, roundBefore, util.GetRoundCount(), "gathering never advances the world clock")
	assert.Equal(t, turnBefore, util.GetTurnCount())
}

func TestHerbBonusesAndPenalties(t *testing.T) {
	w := newWorld(t, "herbs")
	w.forager = &archetypes.Specialist{Name: "Mira", Level: 2}
	w.weapons = []items.Item{{ItemId: 20003}} // a knife in the company
	w.rolls = []int{0, 0, 0, 0}               // base 1 + level 1 + knife 1 = 3; then the scribe roll misses (99)
	w.start(gathering.Herbs)
	got := w.finishAfter(gathering.Herbs)
	assert.Equal(t, 3, w.packCount(idThyme))
	assert.Contains(t, got, "Mira's woodcraft helps")

	// Darkness halves it.
	w = newWorld(t, "herbs")
	w.forager = &archetypes.Specialist{Name: "Mira", Level: 2}
	w.weapons = []items.Item{{ItemId: 20003}}
	w.dark = true
	w.rolls = []int{0, 0, 0, 0}
	w.start(gathering.Herbs)
	got = w.finishAfter(gathering.Herbs)
	assert.Equal(t, 2, w.packCount(idThyme), "3 herbs halved, rounded up")
	assert.Contains(t, got, "too dark")

	// A Scribe may find one rarer herb as well.
	w = newWorld(t, "herbs")
	w.forager = &archetypes.Specialist{Name: "Hero", IsLeader: true, Level: 1}
	w.scribe = true
	w.rolls = []int{0, 0, 0, 0, 0}
	w.start(gathering.Herbs)
	w.finishAfter(gathering.Herbs)
	assert.Equal(t, 1, w.packCount(idThyme), "one common herb")
	assert.Equal(t, 1, w.packCount(30008), "and one goldenbell for the Scribe")
}

func TestBitterWeedWithoutAForager(t *testing.T) {
	w := newWorld(t, "herbs")
	w.rolls = []int{0} // under 15: a weed
	w.start(gathering.Herbs)
	got := w.finishAfter(gathering.Herbs)
	assert.Equal(t, 1, w.packCount(idWeed))
	assert.Zero(t, w.packCount(idThyme))
	assert.Contains(t, got, "bitter weed")

	w = newWorld(t, "herbs")
	w.forager = &archetypes.Specialist{Name: "Mira", Level: 1}
	w.rolls = []int{0, 0, 0, 0}
	w.start(gathering.Herbs)
	w.finishAfter(gathering.Herbs)
	assert.Zero(t, w.packCount(idWeed), "a forager is never fooled")
}

func TestFirewoodYieldAndWeather(t *testing.T) {
	w := newWorld(t, "firewood")
	w.start(gathering.Firewood)
	w.finishAfter(gathering.Firewood)
	assert.Equal(t, 2, w.packCount(idFirewood))

	w = newWorld(t, "firewood")
	w.weapons = []items.Item{{ItemId: 20002}} // an axe
	w.fieldSmith = true
	w.start(gathering.Firewood)
	w.finishAfter(gathering.Firewood)
	assert.Equal(t, 5, w.packCount(idFirewood), "an axe doubles it and a Field Smith adds one")

	w = newWorld(t, "firewood")
	w.weather = "rain"
	w.start(gathering.Firewood)
	got := w.finishAfter(gathering.Firewood)
	assert.Equal(t, 1, w.packCount(idFirewood))
	assert.Equal(t, 1, w.packCount(idDamp), "heavy rain: half come back damp")
	assert.Contains(t, got, "damp")

	w = newWorld(t, "firewood")
	w.weather = "rain"
	w.room.Tags = []string{"indoor"}
	w.start(gathering.Firewood)
	w.finishAfter(gathering.Firewood)
	assert.Equal(t, 2, w.packCount(idFirewood), "rain never reaches an indoor room")
	assert.Zero(t, w.packCount(idDamp))
}

func TestFishingNeedsALineAndCostsHalfTheEffort(t *testing.T) {
	w := newWorld(t, "fishing")
	w.bag[idLine] = 1
	w.rolls = []int{0, 0, 0, 99} // 2 catches at 40, then the line holds
	w.start(gathering.Fishing)
	got := w.finishAfter(gathering.Fishing)
	assert.Equal(t, 2, w.packCount(idFish))
	assert.Contains(t, got, "raw fish")
	assert.Equal(t, []int{50}, w.efforts, "fishing charges half the strain")
	assert.Equal(t, 1, w.bag[idLine], "the line held")
	assert.Equal(t, []int{0}, w.encounters, "no hunting bonus for fishing")
}

func TestAFishingLineCanBreak(t *testing.T) {
	w := newWorld(t, "fishing")
	w.bag[idLine] = 1
	w.rolls = []int{99, 99, 0} // no catch; the line breaks (0 < 5)
	w.start(gathering.Fishing)
	got := w.finishAfter(gathering.Fishing)
	assert.Zero(t, w.packCount(idFish))
	assert.Zero(t, w.bag[idLine], "the line snapped and was spent")
	assert.Contains(t, got, "gives up nothing")
	assert.Contains(t, got, "line snaps")
	assert.Contains(t, w.start(gathering.Fishing), "need a fishing line")
}

func TestHuntingNeedsABowOrSetsSnaresAndAddsDanger(t *testing.T) {
	w := newWorld(t, "game")
	w.rolls = []int{0, 99} // a snare hit at 25, no hide
	started := w.start(gathering.Game)
	assert.Contains(t, started, "snares")
	got := w.finishAfter(gathering.Game)
	assert.Equal(t, 2, w.packCount(idMeat))
	assert.Contains(t, got, "raw game meat")
	assert.Equal(t, []int{10}, w.encounters, "hunting is noisy: +10 to the encounter chance")

	// Snares keep half the chance: a roll of 30 misses where a bow would hit.
	w = newWorld(t, "game")
	w.rolls = []int{30}
	w.start(gathering.Game)
	got = w.finishAfter(gathering.Game)
	assert.Zero(t, w.packCount(idMeat))
	assert.Contains(t, got, "snares come up empty")

	w = newWorld(t, "game")
	w.weapons = []items.Item{{ItemId: 20001}} // a bow
	w.rolls = []int{30, 0, 0}
	started = w.start(gathering.Game)
	assert.NotContains(t, started, "snares")
	got = w.finishAfter(gathering.Game)
	assert.Equal(t, 2, w.packCount(idMeat), "the same roll hits with a bow")
	assert.Equal(t, 1, w.packCount(idHide), "and the goods roll brings a hide")
	assert.Equal(t, []int{100}, w.efforts)
}

func TestHaulGoesToCargoWhenThereIsOneAndStopsAtTheLimit(t *testing.T) {
	w := newWorld(t, "firewood")
	w.cargoErr = nil
	w.start(gathering.Firewood)
	w.finishAfter(gathering.Firewood)
	assert.Equal(t, 2, w.cargo[idFirewood], "into the company cargo")
	assert.Zero(t, w.packCount(idFirewood))

	w = newWorld(t, "firewood")
	w.full = true
	w.start(gathering.Firewood)
	got := w.finishAfter(gathering.Firewood)
	assert.Zero(t, w.packCount(idFirewood))
	assert.Contains(t, got, "can carry no more")
	assert.Equal(t, 3, w.m.ledger.Charges(5001, gathering.Firewood, w.m.settings.Rules[gathering.Firewood], w.now), "the charge is spent either way")

	w = newWorld(t, "firewood")
	w.cargoErr = errors.New("boom")
	w.start(gathering.Firewood)
	got = w.finishAfter(gathering.Firewood)
	assert.Zero(t, w.packCount(idFirewood), "a cargo failure never duplicates into the pack")
	assert.Contains(t, got, "can carry no more")
}

// --- cancelling ---

func TestTypedCommandsCancelTheWork(t *testing.T) {
	cases := []struct {
		text   string
		cancel bool
	}{
		{"look", false}, {"l", false}, {"conditions", false}, {"gather herbs", false}, {"fish", false}, {"hunt", false},
		{"say hello", true}, {"go north", true}, {"attack rat", true}, {"inventory", true}, {"camp", true}, {"n", true},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			w := newWorld(t, "herbs")
			w.start(gathering.Herbs)
			w.m.onInput(events.Input{UserId: 7, InputText: c.text})
			_, busy := w.m.Active(7)
			assert.Equal(t, !c.cancel, busy)
			if c.cancel {
				w.now = w.now.Add(time.Minute)
				w.m.tick()
				assert.Zero(t, w.packCount(idThyme), "nothing is gained")
				assert.Equal(t, 3, w.m.ledger.Charges(5001, gathering.Herbs, w.m.settings.Rules[gathering.Herbs], w.now), "and no charge is spent")
				assert.Empty(t, w.efforts)
			}
		})
	}
}

func TestOtherPeoplesInputAndMobsDoNotCancel(t *testing.T) {
	w := newWorld(t, "herbs")
	w.start(gathering.Herbs)
	w.m.onInput(events.Input{UserId: 8, InputText: "say hi"})
	w.m.onInput(events.Input{UserId: 0, MobInstanceId: 3, InputText: "say hi"})
	w.m.onInput(events.Input{UserId: 7, InputText: "   "})
	_, busy := w.m.Active(7)
	assert.True(t, busy)
}

func TestMovingAFightOrDeathCancelTheWorkAtTheTick(t *testing.T) {
	for name, change := range map[string]func(w *world){
		"moved":      func(w *world) { w.user.Character.RoomId = 5002 },
		"attacked":   func(w *world) { w.battle = true },
		"downed":     func(w *world) { w.user.Character.Health = 0 },
		"resting":    func(w *world) { w.resting = true },
		"travelling": func(w *world) { w.travelling = true },
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, "herbs")
			w.start(gathering.Herbs)
			change(w)
			got := w.finishAfter(gathering.Herbs)
			_, busy := w.m.Active(7)
			assert.False(t, busy)
			assert.Zero(t, w.packCount(idThyme))
			assert.Empty(t, w.efforts)
			assert.Empty(t, w.encounters)
			assert.NotEmpty(t, got, "the leader is told")
			assert.Equal(t, 3, w.m.ledger.Charges(5001, gathering.Herbs, w.m.settings.Rules[gathering.Herbs], w.now))
		})
	}
}

func TestPurgeAndDespawnDropTheWork(t *testing.T) {
	w := newWorld(t, "herbs")
	w.start(gathering.Herbs)
	w.m.onUserPurged(events.UserPurged{UserId: 7})
	_, busy := w.m.Active(7)
	assert.False(t, busy)
	w.start(gathering.Herbs)
	w.m.onDespawn(events.PlayerDespawn{UserId: 7})
	_, busy = w.m.Active(7)
	assert.False(t, busy)
}

func TestOneJobAtATime(t *testing.T) {
	w := newWorld(t, "herbs", "firewood")
	w.start(gathering.Herbs)
	assert.Contains(t, w.start(gathering.Firewood), "already gathering herbs")
	assert.Contains(t, w.start(gathering.Herbs), "already gathering herbs")
}

// --- pools ---

func TestPoolsAreSharedAndRefuseWhenEmpty(t *testing.T) {
	w := newWorld(t, "game")
	rule := w.m.settings.Rules[gathering.Game]
	require.Equal(t, 2, rule.PoolMax)
	w.rolls = []int{99, 99, 99, 99}

	// Two hunts empty the room.
	for i := 0; i < 2; i++ {
		w.start(gathering.Game)
		w.finishAfter(gathering.Game)
	}
	assert.True(t, w.m.isDepleted(5001, "game"), "a failed hunt still spends its charge")
	got := w.start(gathering.Game)
	assert.Contains(t, got, "picked clean for now")
	assert.Contains(t, got, "regrows in")
	// Phase 40a2 review: the bare listing says when, too.
	assert.Contains(t, w.m.offers(w.user, w.room), "picked clean for now, regrows in about 40 minutes")

	// Another company arrives and finds the same empty room.
	other := users.NewUserRecord(8, 1)
	other.Character.Health = 10
	other.Character.RoomId = 5001
	users.SetTestUser(other)
	assert.Contains(t, w.m.start(other, w.room, gathering.Game), "picked clean")

	// 40 minutes later one charge is back, for whoever is first.
	w.now = w.now.Add(40 * time.Minute)
	assert.False(t, w.m.isDepleted(5001, "game"))
	assert.NotContains(t, w.m.start(other, w.room, gathering.Game), "picked clean")
}

func TestTwoCompaniesStartingTogetherSplitTheLastCharge(t *testing.T) {
	w := newWorld(t, "herbs")
	w.m.settings.Rules[gathering.Herbs] = gathering.Rule{Duration: 20 * time.Second, PoolMax: 1, Regrow: 20 * time.Minute, EffortPc: 100}
	other := users.NewUserRecord(8, 1)
	other.Character.Health = 10
	other.Character.RoomId = 5001
	users.SetTestUser(other)

	assert.Contains(t, w.start(gathering.Herbs), "sets about")
	assert.Contains(t, w.m.start(other, w.room, gathering.Herbs), "sets about")
	w.rolls = []int{99, 0, 0, 0}
	got := w.finishAfter(gathering.Herbs)
	assert.Contains(t, got, "another company picked the herbs here clean")
	total := w.packCount(idThyme) + func() int {
		n := 0
		for _, itm := range other.Character.Items {
			if itm.ItemId == idThyme {
				n++
			}
		}
		return n
	}()
	assert.Positive(t, total, "the first company took the bounty")
}

func TestPickedCleanShowsOnLookAndQueuesARedraw(t *testing.T) {
	w := newWorld(t, "herbs", "water")
	rooms.SetDepletedCheck(w.m.isDepleted)
	t.Cleanup(func() { rooms.SetDepletedCheck(nil) })
	assert.Equal(t, "Here: fresh water, herbs.", w.room.ResourceLine())
	assert.Empty(t, w.room.DepletedResources())

	var redraws []int
	id := events.RegisterListener(events.RoomResourcesChanged{}, func(e events.Event) events.ListenerReturn {
		redraws = append(redraws, e.(events.RoomResourcesChanged).RoomId)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.RoomResourcesChanged{}, id) })

	for i := 0; i < 3; i++ {
		w.start(gathering.Herbs)
		w.finishAfter(gathering.Herbs)
	}
	assert.Equal(t, "Here: fresh water, herbs (picked clean).", w.room.ResourceLine())
	assert.Equal(t, []string{"herbs"}, w.room.DepletedResources())
	assert.Equal(t, []int{5001}, redraws, "clients are told once, when it ran dry")

	w.now = w.now.Add(20 * time.Minute)
	assert.Equal(t, "Here: fresh water, herbs.", w.room.ResourceLine(), "and the marker clears when it regrows")
}

func TestPoolsPersistAcrossASaveAndLoad(t *testing.T) {
	w := newWorld(t, "game")
	w.rolls = []int{99, 99, 99, 99}
	for i := 0; i < 2; i++ {
		w.start(gathering.Game)
		w.finishAfter(gathering.Game)
	}
	require.NotZero(t, w.store.saveCalls, "each spend is saved")
	assert.NotEmpty(t, w.store.saved.Rooms[5001], "the room is recorded")

	// A restart or copyover: a new module loads what was saved.
	after := newModule()
	after.store = w.store
	after.clock = func() time.Time { return w.now }
	after.load()
	assert.True(t, after.isDepleted(5001, "game"), "a restart never refills the pool")
	assert.Equal(t, 0, after.ledger.Charges(5001, gathering.Game, after.settings.Rules[gathering.Game], w.now))

	w.now = w.now.Add(time.Hour)
	assert.False(t, after.isDepleted(5001, "game"), "real time regrows it, computed on read")

	// Only rooms in recovery are kept.
	w.now = w.now.Add(3 * time.Hour)
	after.load()
	assert.Empty(t, after.ledger.Rooms, "regrown pools are forgotten on load")
}

func TestALoadFailureStopsGatheringWithoutLosingTheWorld(t *testing.T) {
	w := newWorld(t, "herbs")
	w.store.loadErr = errors.New("disk")
	w.m.load()
	assert.Contains(t, w.start(gathering.Herbs), "unavailable")
	assert.False(t, w.m.isDepleted(5001, "herbs"))
}

func TestEphemeralRoomsNeverRunDry(t *testing.T) {
	w := newWorld(t, "herbs")
	const ephemeral = 1_500_000_000
	require.True(t, rooms.IsEphemeralRoomId(ephemeral))
	w.room.RoomId = ephemeral
	rooms.SetTestRoom(w.room)
	t.Cleanup(func() { rooms.RemoveTestRoom(ephemeral) })
	w.user.Character.RoomId = ephemeral
	for i := 0; i < 5; i++ {
		assert.Contains(t, w.start(gathering.Herbs), "sets about", "attempt %d", i)
		w.finishAfter(gathering.Herbs)
	}
	assert.False(t, w.m.isDepleted(ephemeral, "herbs"))
	assert.Empty(t, w.m.ledger.Rooms, "tutorial copies are never saved")
}

// --- the commands ---

func TestGatherListsWhatTheRoomOffers(t *testing.T) {
	w := newWorld(t, "herbs", "fishing", "game")
	got := w.heard(func() { _, _ = w.m.gatherCommand("", w.user, w.room, 0) })
	assert.Contains(t, got, "Here your company can gather")
	assert.Contains(t, got, "gather herbs")
	assert.Contains(t, got, "3 of 3 ready")
	assert.Contains(t, got, "needs a fishing line")
	assert.Contains(t, got, "snares")
	assert.NotContains(t, got, "gather firewood")

	bare := newWorld(t)
	got = bare.heard(func() { _, _ = bare.m.gatherCommand("", bare.user, bare.room, 0) })
	assert.Contains(t, got, "nothing here to gather")

	w.heard(func() { _, _ = w.m.gatherCommand("", w.user, w.room, 0) })
	got = w.heard(func() { _, _ = w.m.gatherCommand("stones", w.user, w.room, 0) })
	assert.Contains(t, got, "Usage: gather")
}

func TestTheCommandsStartTheWork(t *testing.T) {
	w := newWorld(t, "herbs", "firewood", "fishing", "game")
	w.bag[idLine] = 1
	for verb, kind := range map[string]gathering.Kind{"herbs": gathering.Herbs, "wood": gathering.Firewood, "fish": gathering.Fishing, "game": gathering.Game} {
		w.m.cancel(7, "")
		got := w.heard(func() { _, _ = w.m.gatherCommand(verb, w.user, w.room, 0) })
		assert.Contains(t, got, "Any command other than look or conditions stops the work", verb)
		cur, busy := w.m.Active(7)
		assert.True(t, busy, verb)
		assert.Equal(t, kind, cur, verb)
	}
	w.m.cancel(7, "")
	w.heard(func() { _, _ = w.m.fishCommand("", w.user, w.room, 0) })
	cur, _ := w.m.Active(7)
	assert.Equal(t, gathering.Fishing, cur)
	w.m.cancel(7, "")
	w.heard(func() { _, _ = w.m.huntCommand("", w.user, w.room, 0) })
	cur, _ = w.m.Active(7)
	assert.Equal(t, gathering.Game, cur)
}

// --- config ---

func TestShippedConfigParsesWithRealItems(t *testing.T) {
	s := shippedSettings(t)
	d := gathering.DefaultSettings()
	assert.Equal(t, d.Rules, s.Rules, "the shipped config states the design's numbers")
	assert.Equal(t, 10, s.HuntEncounterBonus)
	assert.Equal(t, gathering.ItemIDs{Firewood: 40, DampFirewood: 41, FishingLine: 42, RawFish: 43, BitterWeed: 44, RawMeat: 29}, s.Items)
	assert.Equal(t, []string{"rain", "storm"}, s.WetWeather)
	for _, tables := range []map[string]gathering.Table{s.Herb, s.RareHerb, s.Fish, s.Goods} {
		assert.NotEmpty(t, tables[gathering.AnyZone], "every table has a fallback")
	}
	assert.NotEmpty(t, s.Herb["Frost Lake"])
}

func TestBadConfigEntriesAreSkipped(t *testing.T) {
	cfg := map[string]any{
		"HerbMin": 3, "HerbMax": 1, // inverted: back to the defaults
		"FishPct":   500,
		"Resources": []any{map[string]any{"Kind": "mining"}, map[string]any{"Kind": "herbs", "Duration": "5s", "Pool": 9}},
		"Herbs":     []any{map[string]any{"Zone": "", "Items": []any{}}},
	}
	s := parseSettings(func(k string) any { return cfg[k] })
	d := gathering.DefaultSettings()
	assert.Equal(t, d.HerbMin, s.HerbMin)
	assert.Equal(t, d.HerbMax, s.HerbMax)
	assert.Equal(t, d.FishPct, s.FishPct, "out of range keeps the default")
	assert.Equal(t, 9, s.Rules[gathering.Herbs].PoolMax)
	assert.Equal(t, 5*time.Second, s.Rules[gathering.Herbs].Duration)
	assert.Equal(t, d.Rules[gathering.Firewood], s.Rules[gathering.Firewood])
	assert.Empty(t, s.Herb)
}
