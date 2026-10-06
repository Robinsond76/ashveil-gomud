package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/flasks"
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

// Phase 39g: a camp rest's end refills a company Alchemist's satchel from the
// leader's reagents, one a flask.
func TestCampRestBrewsAnAlchemistsFlasksWithTheLeadersReagents(t *testing.T) {
	leader := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Aria", Level: 5}}
	mira := &characters.Character{Name: "Mira", Level: 5, HPArchetype: "alchemist"}
	mira.FlasksSpent = 4
	live := map[int]*characters.Character{1: mira}

	assert.Equal(t, []string{"Your flask satchels stay low: you have no reagents."}, brewRestFlasks(leader, live))
	assert.Equal(t, 4, mira.FlasksSpent)

	for i := 0; i < 3; i++ {
		leader.Character.Items = append(leader.Character.Items, items.Item{ItemId: flasks.ReagentItemID})
	}
	lines := brewRestFlasks(leader, live)
	assert.Equal(t, []string{"Mira brews through the night: 3 flask(s) from 3 reagent(s)."}, lines)
	assert.Equal(t, 1, mira.FlasksSpent)
	assert.Zero(t, flasks.Reagents(leader.Character))

	// Nothing missing, or no Alchemist: nothing said.
	mira.FlasksSpent = 0
	assert.Empty(t, brewRestFlasks(leader, live))
	assert.Empty(t, brewRestFlasks(leader, map[int]*characters.Character{2: {Name: "Garrick"}}))
}
