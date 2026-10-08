package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a3: the field surgeon's kit treats one lasting wound at a camp
// rest's end, through a healer with mana, and wears one use.
const testKitItemID = 50

func TestFieldSurgeryTreatsTheMostWoundedMembersWorstWound(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: testKitItemID, Count: 1}}}
	useCargo(t, cargo)
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 70
	tamsin.Character.Wounds = []wounds.Wound{
		{Kind: wounds.Fracture, Place: "arm", Points: 6},
		{Kind: wounds.Cut, Place: "leg", Points: 2},
	}
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 1}}

	lines, treated := module.FieldSurgery(7, testKitItemID)
	require.True(t, treated)
	require.NotEmpty(t, lines)
	assert.Contains(t, lines[0], "kneels beside")
	assert.Contains(t, lines[0], "the worst hurt")
	assert.Equal(t, []int{testKitItemID}, cargo.consumed, "one use of the kit")
	assert.Less(t, oswin.Character.Mana, 20, "a tend costs mana")
	assert.Equal(t, 70, tamsin.Character.Health, "tends only: the rest's own healing is separate")
	assert.Equal(t, []wounds.Wound{{Kind: wounds.Cut, Place: "leg", Points: 2}}, tamsin.Character.Wounds,
		"the broken arm, the worst, is closed with Oswin's mana; the lesser cut is left alone")
	assert.Len(t, b.aria.Character.Wounds, 1, "one patient only")
}

func TestFieldSurgeryNeedsAKitAHealerWithManaAndAWound(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	tamsin := b.companion(1)
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "leg", Points: 4}}

	// No kit in the company.
	useCargo(t, &fakeCargo{})
	_, treated := module.FieldSurgery(7, testKitItemID)
	assert.False(t, treated)
	assert.Len(t, tamsin.Character.Wounds, 1)

	// A kit but no mana: nothing is treated, and nothing is worn.
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: testKitItemID, Count: 1}}}
	useCargo(t, cargo)
	oswin.Character.Mana = 1
	_, treated = module.FieldSurgery(7, testKitItemID)
	assert.False(t, treated)
	assert.Empty(t, cargo.consumed, "a kit that treats nobody isn't worn")

	// A kit and mana but no lasting wound.
	oswin.Character.Mana = 20
	tamsin.Character.Wounds = nil
	_, treated = module.FieldSurgery(7, testKitItemID)
	assert.False(t, treated)
	assert.Empty(t, cargo.consumed)
}

func TestFieldSurgeryNeedsAHealerWhoKnowsTend(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("", map[int]string{1: "warrior", 2: "warrior"})
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: testKitItemID, Count: 1}}}
	useCargo(t, cargo)
	b.companion(2).Character.Mana = 20
	b.companion(1).Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "leg", Points: 4}}
	_, treated := module.FieldSurgery(7, testKitItemID)
	assert.False(t, treated, "no healer in the company: the kit does nothing alone")
	assert.Empty(t, cargo.consumed)
}

// Gear a member carries counts only while that member is with the company:
// a companion's pack counts while they walk with the leader, and not once
// they are separated.
func TestGearCountSkipsASeparatedCompanion(t *testing.T) {
	b := newBrawl(t)
	cargo := &fakeCargo{}
	useCargo(t, cargo)
	kit := spec(t, items.ItemSpec{ItemId: testKitItemID, Name: "field surgeon's kit", Weight: 1500, Uses: 5})
	assert.Zero(t, module.CompanyItemCount(7, testKitItemID))
	cargo.stacks = []encumbrance.CargoStack{{ItemId: testKitItemID, Count: 1}}
	assert.Equal(t, 1, module.CompanyItemCount(7, testKitItemID), "the cargo")
	cargo.stacks = nil

	tamsin := b.companion(1)
	tamsin.Character.Items = append(tamsin.Character.Items, kit)
	require.Equal(t, 1, module.CompanyItemCount(7, testKitItemID), "a present companion's pack")
	require.NoError(t, module.separate(7, 1, domain.SeparatedByMove))
	assert.Zero(t, module.CompanyItemCount(7, testKitItemID), "a separated companion's gear doesn't count")
}
