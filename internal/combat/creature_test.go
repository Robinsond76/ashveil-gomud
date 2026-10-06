package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
)

// Phase 38e: a creature species' base ranks reach the blow and tempo formulas.

func creature(species string, level int) *characters.Character {
	c := classed("", level)
	c.HPArchetype = species
	return c
}

func TestHoundPouncesOnAFoeThatIsDownExposedOrHobbled(t *testing.T) {
	defenseSpecs(t)
	hound := creature("hound", 1)
	foe := classed("", 1)
	assert.Equal(t, 10, classBlowDamage(hound, foe, 10), "a standing foe is bitten plainly")
	for _, id := range []int{status.Exposed, status.KnockedDown, status.Hobbled} {
		down := classed("", 1)
		down.AddBuff(id, true)
		assert.Equal(t, 13, classBlowDamage(hound, down, 10), "+30%% against status %d", id)
	}
}

func TestHoundSavagePursuitGrowsThePounce(t *testing.T) {
	defenseSpecs(t)
	foe := classed("", 20)
	foe.AddBuff(status.Hobbled, true)
	assert.Equal(t, 15, classBlowDamage(creature("hound", 20), foe, 10), "+50% from rank 20")
}

func TestStoneGolemIsSlowerThanTheSlowestPerson(t *testing.T) {
	defenseSpecs(t)
	person := characters.New()
	person.Stats.Speed.ValueAdj = -100 // the floor
	golem := characters.New()
	golem.Stats.Speed.ValueAdj = -100
	golem.HPArchetype = "stone-golem"
	golem.Level = 1
	assert.InDelta(t, Tempo(person)*0.85, Tempo(golem), 1e-9)
}

func TestStoneGolemTempoIsOrdinaryWithoutTheSpecies(t *testing.T) {
	defenseSpecs(t)
	a, b := characters.New(), characters.New()
	b.HPArchetype = "warrior"
	assert.Equal(t, Tempo(a), Tempo(b))
}
