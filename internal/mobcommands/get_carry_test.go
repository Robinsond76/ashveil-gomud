package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32f (review test gap): a companion carries for its company, so it
// can't pick up what would put the company over capacity.

type carryFormation struct{}

func (carryFormation) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (carryFormation) InstanceFor(int, int) (int, bool)           { return 0, false }
func (carryFormation) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	if instanceId == 555 {
		return 7, company.MemberKey("companion:1"), true
	}
	return 0, "", false
}

type fixedLoad struct{ load encumbrance.Load }

func (f fixedLoad) CurrentLoad(int) (encumbrance.Load, bool) { return f.load, true }

func TestCompanionGetRefusedWhenCompanyFull(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	const anvil, pebble = 988401, 988402
	items.SetTestItemSpec(&items.ItemSpec{ItemId: anvil, Name: "anvil", NameSimple: "anvil", Type: items.Object, Weight: 4000})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: pebble, Name: "pebble", NameSimple: "pebble", Type: items.Object, Weight: 100})
	company.SetFormationProvider(carryFormation{})
	encumbrance.SetProvider(fixedLoad{load: encumbrance.Load{PersonalGrams: 2000, CapacityGrams: 3000}})
	t.Cleanup(func() {
		items.RemoveTestItemSpec(anvil)
		items.RemoveTestItemSpec(pebble)
		company.SetFormationProvider(nil)
		encumbrance.SetProvider(nil)
	})

	room := rooms.NewEmptyRoom()
	room.AddItem(items.New(anvil), false)
	room.AddItem(items.New(pebble), false)
	companion := &mobs.Mob{InstanceId: 555}
	companion.Character.Name = "Tamsin"

	_, err := Get("anvil", companion, room)
	require.NoError(t, err)
	assert.Empty(t, companion.Character.Items, "too heavy for the company")
	_, onFloor := room.FindOnFloor("anvil", false)
	assert.True(t, onFloor)

	_, err = Get("pebble", companion, room)
	require.NoError(t, err)
	assert.Len(t, companion.Character.Items, 1, "a light thing still fits")

	stranger := &mobs.Mob{InstanceId: 556}
	stranger.Character.Name = "goblin"
	_, err = Get("anvil", stranger, room)
	require.NoError(t, err)
	assert.Len(t, stranger.Character.Items, 1, "a mob of no company carries what it likes")
}
