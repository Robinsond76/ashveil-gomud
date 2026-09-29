package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserPurgedDropsTheCompany (Phase 32b): a purged leader's record,
// claims included, goes and is saved; a companion mob still standing is
// detached; another leader's company stays.
func TestUserPurgedDropsTheCompany(t *testing.T) {
	registry := *domain.NewRegistry()
	registry.Put(domain.Record{LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 61}}, Claimed: []int{61, 62}})
	registry.Put(domain.Record{LeaderUserID: 8, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}})
	m, runtime := newAlignmentModule(registry, newFakeWorld())
	store := &fakeStore{}
	m.store = store
	m.setInstance(7, 1, 501)
	runtime.live = map[int]bool{501: true}

	m.onUserPurged(events.UserPurged{UserId: 7})
	_, ok := m.registry.Get(7)
	assert.False(t, ok)
	_, ok = m.registry.Get(8)
	assert.True(t, ok)
	require.NotNil(t, store.saved)
	_, ok = store.saved.Companies[7]
	assert.False(t, ok, "saved without the leader")
	assert.Equal(t, 1, runtime.detachCalls, "the standing companion is detached")
	_, tracked := m.instance(7, 1)
	assert.False(t, tracked)
}
