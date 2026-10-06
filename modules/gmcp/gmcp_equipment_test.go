package gmcp

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipmentExtraLeavesLegacyGearAvailable(t *testing.T) {
	u := inventoryUser(t)
	open := func(int) (string, bool) { return "weapon", true }
	assert.Nil(t, equipmentExtra(open).build(u), "legacy Gear must not be replaced by an empty shared catalogue")
	u.Character.CompanyCargo = true
	assert.NotNil(t, equipmentExtra(open).build(u), "shared Gear reports service availability honestly")
}

// The Gear editor's previews are built only while a client shows it, and a
// reopened editor gets the current view at once (Phase 34 review).
func TestEquipmentExtraBuiltOnlyWhileGearOpen(t *testing.T) {
	f, out := testFeed()
	built := 0
	f.extras = []companyExtra{equipmentExtra(f.watchingGear)}
	f.extras[0].build = func(inner func(*users.UserRecord) []byte) func(*users.UserRecord) []byte {
		return func(u *users.UserRecord) []byte {
			body := inner(u)
			if body != nil {
				built++
			}
			return body
		}
	}(f.extras[0].build)
	u := inventoryUser(t)
	u.Character.CompanyCargo = true

	f.updateExtras(u)
	assert.Zero(t, built, "a closed editor builds nothing")
	assert.Empty(t, *out)

	f.setGearOpen(u.UserId, true, "weapon")
	f.updateExtras(u)
	assert.Equal(t, 1, built)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Equipment", (*out)[0].module)

	f.setGearOpen(u.UserId, false, "")
	f.updateExtras(u)
	assert.Equal(t, 1, built, "closing stops the builds")

	f.setGearOpen(u.UserId, true, "")
	slot, open := f.watchingGear(u.UserId)
	assert.True(t, open)
	assert.Equal(t, "weapon", slot, "an editor that names no slot shows the weapon")
	f.prune([]int{})
	_, open = f.watchingGear(u.UserId)
	assert.False(t, open, "a user gone offline is pruned")
}

// The web client's "!!GMCP(Company.Equipment ...)" reaches the feed for the
// connection's own user: open with a slot, open with none or an unknown
// slot (the weapon), and closed; a login clears it until the client says
// again. A repeated or rapid open asks for no extra refresh (review).
func TestGearWatchWebRequestAndSpawn(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(43, 4343)
	users.SetTestUser(u)
	t.Cleanup(func() { companyFeeds.setGearOpen(43, false, "") })
	say := func(msg string) (string, bool) {
		require.True(t, gmcpModule.HandleWebGMCP(u.ConnectionId(), []byte("!!GMCP(Company.Equipment"+msg+")")))
		events.ProcessEvents()
		return companyFeeds.watchingGear(43)
	}
	slot, open := say(" open body")
	assert.True(t, open)
	assert.Equal(t, "body", slot)
	slot, open = say(" open")
	assert.True(t, open)
	assert.Equal(t, "weapon", slot)
	slot, _ = say(" open " + strings.Repeat("x", 500))
	assert.Equal(t, "weapon", slot, "an unknown slot is the weapon, never stored")
	_, open = say(" closed")
	assert.False(t, open)

	say(" open feet")
	events.AddToQueue(events.PlayerSpawn{UserId: 43})
	events.ProcessEvents()
	_, open = companyFeeds.watchingGear(43)
	assert.False(t, open, "a login clears the editor until the client says again")

	f, _ := testFeed()
	assert.True(t, f.setGearOpen(9, true, "body"), "opening refreshes at once")
	assert.False(t, f.setGearOpen(9, true, "body"), "the same slot again refreshes nothing")
	assert.False(t, f.setGearOpen(9, true, "head"), "another slot within the gap waits for the round")
	slot, _ = f.watchingGear(9)
	assert.Equal(t, "head", slot, "but the slot is still recorded")
	f.mu.Lock()
	w := f.gearOpen[9]
	w.refreshed = w.refreshed.Add(-gearRefreshGap)
	f.gearOpen[9] = w
	f.mu.Unlock()
	assert.True(t, f.setGearOpen(9, true, "feet"), "after the gap a new slot refreshes at once")
}

// Phase 48: the editor names whose gear it shows, and only "me" or "#N"
// is accepted from a client.
func TestGearWatchFollowsTheNamedMember(t *testing.T) {
	f, _ := testFeed()
	assert.True(t, f.setGearOpen(21, true, "weapon", "#2"), "opening refreshes at once")
	assert.Equal(t, "#2", f.watchingGearMember(21))
	slot, open := f.watchingGear(21)
	assert.True(t, open)
	assert.Equal(t, "weapon", slot)
	assert.False(t, f.setGearOpen(21, true, "weapon", "#2"), "the same member and slot refresh nothing")
	f.setGearOpen(21, true, "weapon", "me")
	assert.Equal(t, "me", f.watchingGearMember(21), "switching members is noticed")
	for _, bad := range []string{"#0", "#x", "../etc", "", "leader"} {
		f.setGearOpen(21, true, "body", bad)
		assert.Equal(t, "me", f.watchingGearMember(21), "%q is not a member reference", bad)
	}
	f.setGearOpen(21, false, "")
	assert.Equal(t, "me", f.watchingGearMember(21), "a closed editor shows the leader")
}

func TestEquipmentExtraAsksForTheWatchedMember(t *testing.T) {
	u := inventoryUser(t)
	u.Character.CompanyCargo = true
	open := func(int) (string, bool) { return "weapon", true }
	asked := ""
	extra := equipmentExtra(open, func(int) string { asked = "#3"; return "#3" })
	assert.NotNil(t, extra.build(u))
	assert.Equal(t, "#3", asked)
}
