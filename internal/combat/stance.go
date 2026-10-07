package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/stance"
)

// stanceName is the character's weapon stance (Phase 69) as the battle
// breakdown names it.
func stanceName(c *characters.Character) string {
	st := stance.None
	if c.RT != nil {
		st = c.RT.Stance
	}
	d, _ := stance.Lookup(st)
	return d.Name + ` stance`
}
