package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 39e: a rest's end closes a bonded beast's wounds and brings its
// health back, for the leader's own and each companion's; nothing is spent.
func TestRestMendsABeastTamersBeast(t *testing.T) {
	leader := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Aria", Level: 5, HPArchetype: "beasttamer"}}
	leader.Character.EnsureBeast("Ash").Damage = 12
	tamsin := &characters.Character{Name: "Tamsin", Level: 5, HPArchetype: "beasttamer"}
	tamsin.EnsureBeast("Bruin")
	tamsin.Beast.Wounded, tamsin.Beast.Damage = true, 40
	live := map[int]*characters.Character{1: tamsin, 2: {Name: "Garrick"}}

	lines := restBeasts(leader, live)
	assert.Equal(t, []string{"Ash is rested and whole again.", "Tamsin's Bruin is rested and whole again."}, lines)
	assert.Zero(t, leader.Character.Beast.Damage)
	assert.False(t, tamsin.Beast.Wounded)
	assert.Zero(t, tamsin.Beast.Damage)

	// Nothing hurt: nothing is said.
	assert.Empty(t, restBeasts(leader, live))
	// A company with no beast says nothing.
	assert.Empty(t, restBeasts(&users.UserRecord{Character: &characters.Character{}}, map[int]*characters.Character{3: {Name: "Wick"}}))
}
