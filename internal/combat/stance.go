package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/stance"
)

// stanceName is the character's weapon stance (Phase 69) as the battle
// breakdown names it.
func stanceName(c *characters.Character) string {
	d, _ := stance.Lookup(c.Stance())
	return d.Name + ` stance`
}
