package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 39e: the beast command, through the real user command path.

func beastCommandBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypesFor("beasttamer", map[int]string{1: "beasttamer", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.companion(1).Character.HPArchetype = "beasttamer"
	return b
}

func TestBeastCommandShowsTheCompanysBeastsAndNamesTheOwn(t *testing.T) {
	b := beastCommandBrawl(t)
	c := b.aria.Character
	out := b.cmd("beast", "")
	assert.Contains(t, out, "You keep Ash, a wolf: whole")
	assert.Contains(t, out, "Tamsin Reed keeps Ash, a wolf: whole", "companion Tamers are listed")

	c.Beast.Damage = 4
	assert.Contains(t, b.cmd("beast", ""), "A rest brings it back to full health.")
	c.Beast.Wounded = true
	assert.Contains(t, b.cmd("beast", ""), "wounded and sits out battles until the company rests")

	assert.Contains(t, b.cmd("beast", "name Briar"), "answers to Briar now")
	assert.Equal(t, "Briar", c.Beast.Name)
	assert.Contains(t, b.cmd("beast", "name 12345"), "letters, spaces")
	assert.Contains(t, b.cmd("beast", "name"), "one to twenty characters")
	assert.Contains(t, b.cmd("beast", "tamsin"), "Tamsin Reed keeps")
	assert.Contains(t, b.cmd("beast", "nobody"), "No Beast Tamer answers to")
}

func TestBeastCommandNeedsATamer(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("warrior", map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	assert.Contains(t, b.cmd("beast", ""), "No one in your company keeps a bonded beast")
}
