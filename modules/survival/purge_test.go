package survival

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsSurvivalNeeds (Phase 32b).
func TestUserPurgedDropsSurvivalNeeds(t *testing.T) {
	registry := *domain.NewRegistry()
	registry.Leaders[7] = map[domain.MemberKey]domain.Needs{domain.LeaderMemberKey: {}}
	registry.Leaders[8] = map[domain.MemberKey]domain.Needs{domain.LeaderMemberKey: {}}
	registry.ReservedNextCompanionIDs = map[int]int{7: 3}
	registry.AppliedExertion = map[int]map[string]domain.Exertion{7: {"op": {}}}
	registry.AppliedRestOperation = map[int]map[string]int{7: {"rest": 1}}
	store := &fakeStore{}
	m := &SurvivalModule{store: store, registry: registry}

	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls)
	assert.NotContains(t, store.saved.Leaders, 7)
	assert.Contains(t, store.saved.Leaders, 8)
	assert.NotContains(t, store.saved.ReservedNextCompanionIDs, 7)
	assert.NotContains(t, store.saved.AppliedExertion, 7)
	assert.NotContains(t, store.saved.AppliedRestOperation, 7)
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls, "nothing left to save")
}
