package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 29c: the narration voice's engagement lines.

func mobTag(name string) string  { return fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, name) }
func userTag(name string) string { return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, name) }

// turnsToward is the line for a fighter taking a new target: "The bandit
// cutthroat turns toward Garrick Vane." who and target are tagged names;
// who may be "You".
func turnsToward(who, target string) string {
	if who == `You` {
		return fmt.Sprintf(`You turn toward %s.`, util.Article(target))
	}
	return util.CapitalizeFirst(fmt.Sprintf(`%s turns toward %s.`, util.Article(who), util.Article(target)))
}
