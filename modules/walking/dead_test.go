package walking

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanionRosterSkipsDead: a dead companion walks nowhere and is never
// drained (Phase 25b).
func TestCompanionRosterSkipsDead(t *testing.T) {
	survival.SetRosterProvider(rosterStub{refs: map[int][]survival.MemberRef{7: {
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bear"},
		{Key: survival.CompanionMemberKey(2), Name: "Wolf", Dead: true},
	}}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })
	members, ok := companionRoster(7)
	require.True(t, ok)
	require.Len(t, members, 1)
	assert.Equal(t, "Bear", members[0].Name)
}
