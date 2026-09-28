package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// TestGoForText (Phase 29c): a fight's first attack draws the weapon; an
// attack mid-fight turns toward the new foe (review fix).
func TestGoForText(t *testing.T) {
	c := characters.New()
	assert.Equal(t, `You go for the rat.`, goForText(c, `the rat`, false))
	assert.Equal(t, `You turn toward the rat.`, goForText(c, `the rat`, true))
}
