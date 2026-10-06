package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 39d: a camp rest's end mends a company Doll Master's dolls with the
// leader's doll parts.
func TestCampRestMendsADollMastersDollsWithTheLeadersParts(t *testing.T) {
	leader := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Aria", Level: 5}}
	master := &characters.Character{Name: "Tamsin", Level: 5, HPArchetype: "dollmaster"}
	master.EnsureDolls(1)
	master.Dolls[0].Broken, master.Dolls[0].Damage = true, 60
	live := map[int]*characters.Character{1: master}

	// No parts: the dolls stay worn, and the player is told why.
	lines := mendRestDolls(leader, live)
	assert.Equal(t, []string{"Your dolls stay worn: you have no doll parts."}, lines)
	assert.True(t, master.Dolls[0].Broken)

	for i := 0; i < 3; i++ {
		leader.Character.Items = append(leader.Character.Items, items.Item{ItemId: dolls.PartsItemID})
	}
	lines = mendRestDolls(leader, live)
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Tamsin mends the dolls through the night with 3 doll part(s)")
	assert.False(t, master.Dolls[0].Broken)
	assert.Zero(t, dolls.Parts(leader.Character), "the parts are spent")
	assert.Less(t, master.Dolls[0].Damage, 60)

	// Nothing to mend: nothing is said and nothing is spent.
	master.Dolls[0].Damage = 0
	leader.Character.Items = append(leader.Character.Items, items.Item{ItemId: dolls.PartsItemID})
	assert.Empty(t, mendRestDolls(leader, live))
	assert.Equal(t, 1, dolls.Parts(leader.Character))

	// A company with no Doll Master says nothing.
	assert.Empty(t, mendRestDolls(leader, map[int]*characters.Character{2: {Name: "Garrick"}}))
}
