package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/stretchr/testify/assert"
)

type titleProvider struct{}

func (titleProvider) CanTrain(int, string) (bool, string)      { return true, "" }
func (titleProvider) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (titleProvider) Exists(id string) bool                    { return id == "warrior" }
func (titleProvider) PlayerArchetype(int) (string, bool)       { return "", false }
func (titleProvider) ArchetypeName(id string) (string, bool) {
	return "Warrior", id == "warrior"
}

// TestClassTitleNamesTheArchetype: found live (Phase 44), status, who and
// GMCP called every Ashveil warrior a "scrub paladin" from legacy
// skill-based professions. The class title is the archetype's name, and
// only falls back to the profession title without one.
func TestClassTitleNamesTheArchetype(t *testing.T) {
	archetypes.SetProvider(titleProvider{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })

	c := New()
	c.HPArchetype = "warrior"
	assert.Equal(t, "Warrior", c.ClassTitle())

	c.HPArchetype = ""
	assert.NotEqual(t, "Warrior", c.ClassTitle(), "no archetype: the profession title")
}
