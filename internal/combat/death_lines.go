package combat

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 29c: death notices in the narration voice. Each takes a tagged
// name (`<ansi fg="mobname">bandit captain</ansi>`) and puts "the" before
// a lowercase one.

var (
	// DeathLines are a slain mob's last line.
	DeathLines = []string{
		`%s crumples and does not rise.`,
		`%s falls and lies still.`,
		`%s goes down hard and stays down.`,
		`%s sags to the ground, and is still.`,
	}
	// BeatenLines are a practice foe's (Phase 27c): beaten, not killed.
	BeatenLines = []string{
		`%s is beaten and yields the field.`,
	}
	// PlayerDeathLines are what the room sees when a player dies.
	PlayerDeathLines = []string{
		`%s falls and does not get up.`,
		`%s goes down, and the light leaves their eyes.`,
	}
)

// DeathLine is a slain mob's death notice.
func DeathLine(name string) string { return pickLine(DeathLines, name) }

// BeatenLine is a practice foe's notice.
func BeatenLine(name string) string { return pickLine(BeatenLines, name) }

// PlayerDeathLine is a player's death notice, for the room.
func PlayerDeathLine(name string) string { return pickLine(PlayerDeathLines, name) }

func pickLine(pool []string, name string) string {
	line := pool[util.Rand(len(pool))]
	return util.CapitalizeFirst(fmt.Sprintf(line, util.Article(name)))
}
