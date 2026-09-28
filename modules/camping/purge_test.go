package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

type stubTimer struct{ stopped bool }

func (s *stubTimer) Stop() bool { s.stopped = true; return true }

// TestUserPurgedDropsCampingState (Phase 32b): a purged leader's camp,
// stay, owed tiers, and timers go without granting anything, and the lit
// fire with them; another leader's camp stays.
func TestUserPurgedDropsCampingState(t *testing.T) {
	store := &fakeStore{}
	m := &CampingModule{store: store, camps: map[int]camping.Camp{}, recoveryApplied: map[int]bool{}, timers: map[int]Timer{}, timerGeneration: map[int]uint64{}}
	m.resetInnState()
	m.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, FireLit: true}
	m.camps[8] = camping.Camp{LeaderUserID: 8, RoomID: 200}
	m.recoveryApplied[7] = true
	m.stays[7] = camping.InnStay{}
	m.restedPending[7] = true
	m.wellRestedPending[7] = true
	m.owed[7] = map[int]camping.OwedGrant{1: {}}
	m.autoSharpen[7] = true
	camp, inn := &stubTimer{}, &stubTimer{}
	m.timers[7], m.innTimers[7] = camp, inn
	m.refreshLitRoomsLocked()
	assert.True(t, m.litRooms[100])

	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, map[int]camping.Camp{8: {LeaderUserID: 8, RoomID: 200}}, store.saved.Camps)
	assert.Empty(t, store.saved.Stays)
	assert.Empty(t, store.saved.RestedPending)
	assert.Empty(t, store.saved.WellRestedPending)
	assert.Empty(t, store.saved.Owed)
	assert.Empty(t, store.saved.AutoSharpen)
	assert.Empty(t, store.saved.RecoveryApplied)
	assert.True(t, camp.stopped)
	assert.True(t, inn.stopped)
	assert.False(t, m.litRooms[100], "the fire went with the camp")

	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls, "nothing left to save")
}
