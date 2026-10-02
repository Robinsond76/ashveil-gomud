package gmcp

import (
	"testing"

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
