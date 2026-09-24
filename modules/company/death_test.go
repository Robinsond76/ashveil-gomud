package company

import (
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deathTemplate is a mob template no fixture world defines, so companion
// names stay the fallback.
const deathTemplate = 990065

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) advance(d time.Duration) { c.now = c.now.Add(d) }

// newDeathModule has leader 7 online with companions #1 (instance 101,
// level 5, wearing a helmet and a sword, carrying a ration, 9 gold) and #2
// (instance 102), #1 in formation cell (0,1). The allowance is 3600
// seconds.
func newDeathModule(t *testing.T) (*CompanyModule, *fakeChemWorld, *fakeRuntime, *testClock) {
	t.Helper()
	useFakeLifecycle(t, &fakeLifecycle{})
	runtime := &fakeRuntime{nextInstanceID: 201, live: map[int]bool{101: true, 102: true}}
	state := domain.MemberState{Level: 5, Experience: 1234, Gold: 9, Items: []items.Item{{ItemId: 3}}}
	state.Equipment.Head = items.Item{ItemId: 1}
	state.Equipment.Weapon = items.Item{ItemId: 2}
	record := domain.Record{LeaderUserID: 7, NextCompanionID: 3, Companions: []domain.Companion{
		{ID: 1, MobTemplateID: deathTemplate, State: &state, Disposition: &domain.Disposition{Alignment: 10, Loyalty: 80}},
		{ID: 2, MobTemplateID: deathTemplate, State: &domain.MemberState{Level: 2}, Disposition: &domain.Disposition{Alignment: 10, Loyalty: 80}},
	}}
	require.NoError(t, record.Formation.Place(domain.CompanionMemberKey(1), 0, 1))
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{7: record}}, runtime)
	module.setInstance(7, 1, 101)
	module.setInstance(7, 2, 102)
	module.world = newFakeWorld()
	world := newFakeChemWorld()
	world.online[7] = true
	module.chem = world
	clock := &testClock{now: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)}
	module.clock = clock.Now
	module.allowanceForTest = 3600
	return module, world, runtime, clock
}

// killOne delivers companion #1's death: its helmet stayed on the body,
// everything else dropped.
func killOne(module *CompanyModule) {
	module.onMobDeath(events.MobDeath{InstanceId: 101, Level: 5, KeptWorn: map[items.ItemType]items.Item{items.Head: {ItemId: 1}}})
}

func companion(t *testing.T, module *CompanyModule, id int) domain.Companion {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	for _, c := range record.Companions {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("companion %d is gone", id)
	return domain.Companion{}
}

func told(world *fakeChemWorld) string { return strings.Join(world.told[7], "\n") }

// TestCompanionDeathMarksDead: the death is durable, keeps only what the
// body kept, leaves the formation, and tells the leader how long they have.
func TestCompanionDeathMarksDead(t *testing.T) {
	module, world, _, _ := newDeathModule(t)
	killOne(module)

	c := companion(t, module, 1)
	require.True(t, c.Dead())
	assert.Equal(t, 3600, c.Death.Allowance)
	assert.Equal(t, 3600, c.Death.Remaining)
	assert.NotEmpty(t, c.Death.OpID)
	assert.True(t, c.Death.Placed)
	assert.Equal(t, 1, c.Death.Col)
	assert.Equal(t, 5, c.State.Level)
	assert.Equal(t, 1, c.State.Equipment.Head.ItemId, "the helmet stayed on the body")
	assert.Zero(t, c.State.Equipment.Weapon.ItemId, "the sword dropped")
	assert.Empty(t, c.State.Items)
	assert.Zero(t, c.State.Gold)
	record, _ := module.registry.Get(7)
	_, _, placed := record.Formation.Find(domain.CompanionMemberKey(1))
	assert.False(t, placed)
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked)
	assert.Equal(t, 1, module.store.(*fakeStore).saveCalls)
	saved := module.store.(*fakeStore).saved.Companies[7].Companions[0]
	assert.True(t, saved.Dead(), "saved at once")
	assert.Contains(t, told(world), "has fallen")
	assert.Contains(t, told(world), "1h 0m")

	// A second death event for the same companion changes nothing.
	module.onMobDeath(events.MobDeath{InstanceId: 101, Level: 5})
	assert.Equal(t, 1, module.store.(*fakeStore).saveCalls)
}

// TestDeadNotRestoredOnSpawn: a dead companion never comes back at login
// or copyover; the living one does.
func TestDeadNotRestoredOnSpawn(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	killOne(module)
	module.clearInstance(7, 2)
	delete(runtime.live, 102)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 1, runtime.spawnCalls, "only #2")
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked)
}

// TestRosterMarksDead: survival sees the dead companion as dead.
func TestRosterMarksDead(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	roster := module.Roster(7)
	require.Len(t, roster, 3)
	assert.True(t, roster[1].Dead)
	assert.False(t, roster[2].Dead)
	assert.Equal(t, survival.CompanionMemberKey(1), roster[1].Key)
}

// TestDeadExcludedFromDriftAndAverage: a dead companion neither drifts,
// loses loyalty, deserts, nor counts in the company alignment.
func TestDeadExcludedFromDriftAndAverage(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	world := newFakeWorld()
	world.leaders[7] = -100
	module.world = world
	record, _ := module.registry.Get(7)
	record.Companions[0].Disposition = &domain.Disposition{Alignment: 100, Loyalty: 1}
	module.registry.Put(record)
	killOne(module)
	average, ok := module.companyAverage(7)
	require.True(t, ok)
	assert.Equal(t, domain.AverageAlignment([]int{-100, 10}), average, "the leader and #2")
	before := companion(t, module, 1).Disposition
	module.driftTick()
	after := companion(t, module, 1)
	assert.Equal(t, before, after.Disposition, "untouched")
}

// TestStatusShowsFallenAndLost: status says who is fallen and how long is
// left, and lists the lost.
func TestStatusShowsFallenAndLost(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Lost = []domain.LostCompanion{{ID: 9, Name: "Old Bran", Level: 4}}
	module.registry.Put(record)
	text := module.status(7)
	assert.Contains(t, text, "#1")
	assert.Contains(t, text, "fallen, 1h 0m left to raise")
	assert.Contains(t, text, "Lost: #9 Old Bran (level 4)")
}

// TestDeadCountsTowardCap: the dead keep their roster slot.
func TestDeadCountsTowardCap(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	module.plug = nil
	record, _ := module.registry.Get(7)
	record.Companions = append(record.Companions, domain.Companion{ID: 3, MobTemplateID: 58}, domain.Companion{ID: 4, MobTemplateID: 58})
	module.registry.Put(record)
	_, err := module.summon(7, 12, "58")
	assert.ErrorIs(t, err, domain.ErrCompanyFull)
}

// TestAllowanceChargedOnlyOnline: time passes only while the leader is
// signed in; offline time and the time before the anchor aren't spent.
func TestAllowanceChargedOnlyOnline(t *testing.T) {
	module, world, _, clock := newDeathModule(t)
	killOne(module)
	for i := 0; i < 10; i++ {
		clock.advance(4 * time.Second)
		module.chargeAllowances()
	}
	assert.Equal(t, 3560, companion(t, module, 1).Death.Remaining)

	world.online[7] = false
	for i := 0; i < 10; i++ {
		clock.advance(time.Hour)
		module.chargeAllowances()
	}
	assert.Equal(t, 3560, companion(t, module, 1).Death.Remaining, "offline")

	world.online[7] = true
	module.chargeAllowances() // a fresh anchor, nothing charged
	clock.advance(4 * time.Second)
	module.chargeAllowances()
	assert.Equal(t, 3556, companion(t, module, 1).Death.Remaining)

	// Fractions of a second carry over rather than being lost.
	clock.advance(1500 * time.Millisecond)
	module.chargeAllowances()
	clock.advance(1500 * time.Millisecond)
	module.chargeAllowances()
	assert.Equal(t, 3553, companion(t, module, 1).Death.Remaining)
}

// TestAllowanceStepCapped: a stalled game loop isn't spent.
func TestAllowanceStepCapped(t *testing.T) {
	module, _, _, clock := newDeathModule(t)
	killOne(module)
	clock.advance(time.Hour)
	module.chargeAllowances()
	assert.Equal(t, 3600-maxChargeStep, companion(t, module, 1).Death.Remaining)
}

// TestAllowanceChargedAtLogoutAndSaved: logout charges up to the moment
// and saves; the time offline and before the next login's anchor is free.
func TestAllowanceChargedAtLogoutAndSaved(t *testing.T) {
	module, world, _, clock := newDeathModule(t)
	killOne(module)
	saves := module.store.(*fakeStore).saveCalls
	clock.advance(30 * time.Second)
	module.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.Greater(t, module.store.(*fakeStore).saveCalls, saves)
	assert.Equal(t, 3570, module.store.(*fakeStore).saved.Companies[7].Companions[0].Death.Remaining)

	world.online[7] = false
	clock.advance(24 * time.Hour)
	world.online[7] = true
	module.startAnchor(7) // login
	clock.advance(4 * time.Second)
	module.chargeAllowances()
	assert.Equal(t, 3566, companion(t, module, 1).Death.Remaining)
}

// TestAllowanceWarnsOnce: the leader is warned once when half an hour is
// left.
func TestAllowanceWarnsOnce(t *testing.T) {
	module, world, _, clock := newDeathModule(t)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Companions[0].Death.Remaining = warnAtSeconds + 10
	module.registry.Put(record)
	world.told[7] = nil
	for i := 0; i < 5; i++ {
		clock.advance(4 * time.Second)
		module.chargeAllowances()
	}
	assert.Equal(t, 1, strings.Count(told(world), "Time is short"))
	assert.True(t, companion(t, module, 1).Death.Warned)
}

// TestExpiryArchivesAndFreesSlot: at zero the companion is lost: archived,
// its slot free, its survival record gone, its ID never reused.
func TestExpiryArchivesAndFreesSlot(t *testing.T) {
	module, world, _, clock := newDeathModule(t)
	lifecycle := &fakeLifecycle{}
	useFakeLifecycle(t, lifecycle)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Companions[0].Death.Remaining = 5
	module.registry.Put(record)
	clock.advance(4 * time.Second)
	module.chargeAllowances()
	assert.True(t, companion(t, module, 1).Dead(), "1 second left")
	clock.advance(4 * time.Second)
	module.chargeAllowances()

	record, _ = module.registry.Get(7)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 2, record.Companions[0].ID)
	require.Len(t, record.Lost, 1)
	assert.Equal(t, 1, record.Lost[0].ID)
	assert.Equal(t, 5, record.Lost[0].Level)
	assert.NotEmpty(t, record.Lost[0].OpID)
	assert.GreaterOrEqual(t, record.NextCompanionID, 3)
	assert.Contains(t, lifecycle.removed, [2]int{7, 1})
	assert.Len(t, module.store.(*fakeStore).saved.Companies[7].Lost, 1, "saved")
	assert.Contains(t, told(world), "is lost to you")
}

// TestExpirySaveFailureRollsBack: a failed save keeps the companion dead
// (at zero) and its survival record, and the next round tries again.
func TestExpirySaveFailureRollsBack(t *testing.T) {
	module, _, _, clock := newDeathModule(t)
	lifecycle := &fakeLifecycle{needs: map[[2]int]survival.Needs{{7, 1}: survival.FullNeeds()}}
	useFakeLifecycle(t, lifecycle)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Companions[0].Death.Remaining = 1
	module.registry.Put(record)
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	clock.advance(4 * time.Second)
	module.chargeAllowances()
	c := companion(t, module, 1)
	assert.True(t, c.Dead())
	assert.Zero(t, c.Death.Remaining)
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Lost)
	assert.Contains(t, lifecycle.needs, [2]int{7, 1}, "survival restored")

	module.store.(*fakeStore).saveErr = nil
	clock.advance(4 * time.Second)
	module.chargeAllowances()
	record, _ = module.registry.Get(7)
	assert.Len(t, record.Lost, 1)
}

// TestLoginReminder: at login the leader is reminded of each dead
// companion and its time.
func TestLoginReminder(t *testing.T) {
	module, world, _, _ := newDeathModule(t)
	killOne(module)
	world.told[7] = nil
	module.remindDead(7)
	assert.Contains(t, told(world), "lies fallen")
	assert.Contains(t, told(world), "1h 0m")
}

// TestLoadedDeadRecordRoundTrips: the death and the lost roll survive the
// company file.
func TestLoadedDeadRecordRoundTrips(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Lost = []domain.LostCompanion{{ID: 9, Name: "Old Bran"}}
	module.registry.Put(record)
	data, err := yamlMarshal(module.registry)
	require.NoError(t, err)
	loaded := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, loaded))
	got, _ := loaded.Get(7)
	require.True(t, got.Companions[0].Dead())
	assert.Equal(t, 3600, got.Companions[0].Death.Remaining)
	assert.Len(t, got.Lost, 1)
}

// TestAlignmentViewShowsFallen: the dead are listed as fallen and sway no
// one's mood.
func TestAlignmentViewShowsFallen(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	text := module.alignmentView(7)
	assert.Contains(t, text, "#1 990065: fallen")
	assert.Contains(t, text, "#2 ")
}
