package expedition

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	domain "github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/stretchr/testify/assert"
)

type purgeTimer struct{ stopped bool }

func (p *purgeTimer) Stop() bool { p.stopped = true; return true }

// TestUserPurgedDropsTheJourney (Phase 32b): the session goes and its
// timer stops; nobody is moved.
func TestUserPurgedDropsTheJourney(t *testing.T) {
	store := &fakeStore{}
	timer := &purgeTimer{}
	m := &ExpeditionModule{store: store,
		sessions:        map[int]domain.TravelSession{7: {LeaderUserID: 7}, 8: {LeaderUserID: 8}},
		timers:          map[int]Timer{7: timer},
		timerGeneration: map[int]uint64{7: 3},
	}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, map[int]domain.TravelSession{8: {LeaderUserID: 8}}, store.saved.Sessions)
	assert.True(t, timer.stopped)
	assert.Empty(t, m.timers)
}
