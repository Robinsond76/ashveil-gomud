package company

import (
	"errors"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// relocationModule is leader 7 with four companions (101-104), all
// standing in room 5 and online.
func relocationModule(t *testing.T) (*CompanyModule, *fakeRuntime) {
	t.Helper()
	runtime := &fakeRuntime{
		live:  map[int]bool{101: true, 102: true, 103: true, 104: true},
		rooms: map[int]int{101: 5, 102: 5, 103: 5, 104: 5},
	}
	record := domain.Record{LeaderUserID: 7, Companions: []domain.Companion{
		{ID: 1, MobTemplateID: chemTemplate}, {ID: 2, MobTemplateID: chemTemplate}, {ID: 3, MobTemplateID: chemTemplate}, {ID: 4, MobTemplateID: chemTemplate},
	}}
	require.NoError(t, record.Formation.Place(domain.CompanionMemberKey(1), 1, 1))
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{7: record}}, runtime)
	for companionID, instanceID := range map[int]int{1: 101, 2: 102, 3: 103, 4: 104} {
		module.setInstance(7, companionID, instanceID)
	}
	world := newFakeChemWorld()
	world.online[7] = true
	module.chem = world
	return module, runtime
}

func separationOf(t *testing.T, m *CompanyModule, companionID int) *domain.Separation {
	t.Helper()
	record, _ := m.registry.Get(7)
	c, ok := findCompanion(record, companionID)
	require.True(t, ok)
	return c.Separation
}

func TestRelocateCompanyMovesLiveAttached(t *testing.T) {
	module, runtime := relocationModule(t)
	before, _ := module.registry.Get(7)

	moved := module.RelocateCompany(7, 5, 18)

	assert.Equal(t, 4, moved)
	assert.Equal(t, map[int]int{101: 18, 102: 18, 103: 18, 104: 18}, runtime.relocated)
	after, _ := module.registry.Get(7)
	assert.Equal(t, before, after, "only live mobs move")
	assert.Zero(t, runtime.detachCalls)
	assert.Zero(t, runtime.spawnCalls)
	instanceID, tracked := module.instance(7, 1)
	assert.True(t, tracked, "still tracked")
	assert.Equal(t, 101, instanceID)
}

func TestRelocateCompanySkipsDeadDetachedAndUnattached(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.dying = map[int]bool{101: true}  // alive no longer
	runtime.stolen = map[int]bool{102: true} // charmed away
	delete(runtime.live, 103)                // gone
	module.clearInstance(7, 4)               // died earlier: untracked

	moved := module.RelocateCompany(7, 5, 18)

	assert.Zero(t, moved)
	assert.Empty(t, runtime.relocated)
	assert.Nil(t, separationOf(t, module, 1), "the dying are left to their death")
	assert.Nil(t, separationOf(t, module, 2), "a companion charmed away is not separated")
	assert.Zero(t, module.RelocateCompany(8, 5, 18), "a leader with no company")
}

// Phase 33h3: a companion already in the destination comes too; one
// standing anywhere else is separated, saved first, then taken off the map.
func TestRelocateCompanySeparatesThoseNotWithTheLeader(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.rooms[102] = 18 // already there
	runtime.rooms[103] = 40 // elsewhere
	runtime.fighting = map[int]bool{101: true}

	moved := module.RelocateCompany(7, 5, 18)

	assert.Equal(t, 3, moved, "the fighter in the old room comes out of its fight")
	assert.Equal(t, map[int]int{101: 18, 102: 18, 104: 18}, runtime.relocated)
	sep := separationOf(t, module, 3)
	require.NotNil(t, sep)
	assert.Equal(t, domain.SeparatedByMove, sep.Reason)
	assert.Equal(t, domain.DefaultSeparationRounds, sep.RoundsLeft)
	_, tracked := module.instance(7, 3)
	assert.False(t, tracked, "its mob is removed")
	assert.Equal(t, 1, runtime.detachCalls)
	saved := module.store.(*fakeStore).saved.Companies[7]
	c, _ := findCompanion(saved, 3)
	require.NotNil(t, c.Separation, "the separation is saved before the mob is removed")
	told := strings.Join(module.chem.(*fakeChemWorld).told[7], "\n")
	assert.Contains(t, told, "was not with you and is separated")
	assert.Contains(t, told, "help separation")
}

func TestSeparateFailedSaveLeavesTheCompanionInPlace(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.rooms[103] = 40
	module.store.(*fakeStore).saveErr = errors.New("disk full")

	module.RelocateCompany(7, 5, 18)

	assert.Nil(t, separationOf(t, module, 3))
	_, tracked := module.instance(7, 3)
	assert.True(t, tracked, "still out where it stood")
	assert.Zero(t, runtime.detachCalls)
}

func TestRelocateCompanyLeavesDeadAndFledAlone(t *testing.T) {
	module, runtime := relocationModule(t)
	record, _ := module.registry.Get(7)
	record.Companions[0].Death = &domain.CompanionDeath{Remaining: 100}
	record.Companions[1].PendingReturn = true
	module.registry.Put(record)
	module.clearInstance(7, 1)
	module.clearInstance(7, 2)

	module.RelocateCompany(7, 5, 18)

	assert.Nil(t, separationOf(t, module, 1))
	assert.Nil(t, separationOf(t, module, 2))
	after, _ := module.registry.Get(7)
	assert.Equal(t, 100, after.Companions[0].Death.Remaining, "a move never touches a dead companion's time")
	assert.Zero(t, runtime.spawnCalls, "nobody is respawned")
}

func TestSweepStraysSeparatesAfterTwoRoundsAway(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.away = map[int]bool{103: true, 104: true}
	runtime.fighting = map[int]bool{104: true}

	module.sweepStrays()
	assert.Nil(t, separationOf(t, module, 3), "one round away is an ordinary follow")
	module.sweepStrays()

	sep := separationOf(t, module, 3)
	require.NotNil(t, sep)
	assert.Equal(t, domain.SeparatedStray, sep.Reason)
	assert.Nil(t, separationOf(t, module, 4), "a companion fighting away is left to its fight")
	assert.Contains(t, strings.Join(module.chem.(*fakeChemWorld).told[7], "\n"), "lost sight of you")
}

func TestSweepStraysResetsWhenTheCompanionCatchesUp(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.away = map[int]bool{103: true}
	module.sweepStrays()
	runtime.away[103] = false
	module.sweepStrays()
	runtime.away[103] = true
	module.sweepStrays()

	assert.Nil(t, separationOf(t, module, 3))
}

func TestSweepStraysWaitsForAnOfflineLeader(t *testing.T) {
	module, runtime := relocationModule(t)
	module.chem.(*fakeChemWorld).online[7] = false
	runtime.away = map[int]bool{103: true}
	module.sweepStrays()
	module.sweepStrays()

	assert.Nil(t, separationOf(t, module, 3))
}

func separatedModule(t *testing.T, rounds int) (*CompanyModule, *fakeRuntime) {
	t.Helper()
	module, runtime := relocationModule(t)
	runtime.rooms[103] = 40
	runtime.nextInstanceID = 200
	module.RelocateCompany(7, 5, 18)
	record, _ := module.registry.Get(7)
	for i := range record.Companions {
		if record.Companions[i].Separation != nil {
			record.Companions[i].Separation.RoundsLeft = rounds
		}
	}
	module.registry.Put(record)
	return module, runtime
}

func TestTickSeparationsRejoinsWhenTimeIsUpAndTheLeaderIsFree(t *testing.T) {
	module, runtime := separatedModule(t, 2)
	free := true
	module.leaderFree = func(int) (int, bool) { return 18, free }

	module.tickSeparations()
	require.NotNil(t, separationOf(t, module, 3))
	assert.Equal(t, 1, separationOf(t, module, 3).RoundsLeft)

	free = false // a battle
	module.tickSeparations()
	require.NotNil(t, separationOf(t, module, 3), "never rejoins into a fight")
	assert.Zero(t, separationOf(t, module, 3).RoundsLeft)

	free = true
	module.tickSeparations()
	assert.Nil(t, separationOf(t, module, 3))
	assert.Equal(t, 18, runtime.spawnedRoomID, "it rejoins beside the leader")
	_, tracked := module.instance(7, 3)
	assert.True(t, tracked)
	saved := module.store.(*fakeStore).saved.Companies[7]
	c, _ := findCompanion(saved, 3)
	assert.Nil(t, c.Separation, "the return is saved")
	assert.Contains(t, strings.Join(module.chem.(*fakeChemWorld).told[7], "\n"), "rejoins you")
}

func TestTickSeparationsCountsOnlyWhileTheLeaderIsOnline(t *testing.T) {
	module, _ := separatedModule(t, 3)
	module.leaderFree = func(int) (int, bool) { return 18, true }
	module.chem.(*fakeChemWorld).online[7] = false

	module.tickSeparations()
	module.tickSeparations()

	assert.Equal(t, 3, separationOf(t, module, 3).RoundsLeft)
}

func TestRejoinFailedSaveKeepsTheSeparation(t *testing.T) {
	module, runtime := separatedModule(t, 0)
	module.leaderFree = func(int) (int, bool) { return 18, true }
	spawns := runtime.spawnCalls
	module.store.(*fakeStore).saveErr = errors.New("disk full")

	module.tickSeparations()

	require.NotNil(t, separationOf(t, module, 3))
	assert.Equal(t, spawns, runtime.spawnCalls)
}

func TestRestoreForLeaderSkipsTheSeparated(t *testing.T) {
	module, runtime := separatedModule(t, 5)
	for _, id := range []int{1, 2, 4} {
		module.clearInstance(7, id) // as after a logout
	}

	require.NoError(t, module.restoreForLeader(7, 18))

	assert.Equal(t, 3, runtime.spawnCalls, "the three with the leader, not the separated one")
	_, tracked := module.instance(7, 3)
	assert.False(t, tracked, "a relog does not bring a separated companion back early")
}

func TestSeparatedViewsAndLoad(t *testing.T) {
	module, _ := separatedModule(t, 5)
	views, ok := module.CompanyMembers(7)
	require.True(t, ok)
	var sep domain.MemberView
	for _, v := range views {
		if v.ID == 3 {
			sep = v
		}
	}
	assert.Equal(t, domain.MemberSeparated, sep.Status)
	assert.Positive(t, sep.RejoinSeconds)
	assert.Contains(t, module.status(7), "separated; back in")
	for _, ref := range module.Roster(7) {
		if ref.Key == domain.CompanionMemberKey(3) {
			assert.True(t, ref.Away, "survival freezes a separated companion")
		}
	}
	for _, carry := range module.CompanionCarry(7) {
		assert.Zero(t, carry.PackGrams)
	}
	assert.Len(t, module.CompanionCarry(7), 3, "its pack is away with it")
}
