package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/stretchr/testify/assert"
)

// specialistStub is the archetype stub that also renders specialists.
type specialistStub struct{ archetypeStub }

func (specialistStub) BestSpecialist(int, string, ...int) (archetypes.Specialist, bool) {
	return archetypes.Specialist{}, false
}
func (specialistStub) SpecialistsView(leader int) string {
	if leader == 7 {
		return "Company specialists for leader 7"
	}
	return ""
}

// TestCompanySpecialistsShowsTheView (33f2): "company specialists" shows
// the archetype module's view for the leader who asks, through the real
// command.
func TestCompanySpecialistsShowsTheView(t *testing.T) {
	b := newBrawl(t)
	archetypes.SetProvider(specialistStub{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	assert.Contains(t, b.cmd("company", "specialists"), "Company specialists for leader 7")

	archetypes.SetProvider(nil)
	assert.Contains(t, b.cmd("company", "specialists"), "can't be read right now")
}
