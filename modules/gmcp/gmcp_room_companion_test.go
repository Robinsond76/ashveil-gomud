package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// Phase 79: the web room panel tags a company member "companion", not the
// engine's "charmed", which the terminal already hides for them.
func TestRoomPanelTagsCompanionsNotCharmed(t *testing.T) {
	c := characters.New()
	c.Name = "Tamsin"
	c.Charm(7, characters.CharmPermanent, "")
	assert.Equal(t, []string{"charmed"}, c.GetAdjectives(), "the engine tags a charmed mob")
	assert.Equal(t, []string{"charmed"}, roomAdjectives(c), "a charmed mob that is not a company member keeps its tag")

	c.Charmed.Companion = true
	assert.Equal(t, []string{"companion"}, roomAdjectives(c))
}
