package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ashveil Phase 62: `why` explains a battle line. The combat engine records
// each strike's roll with its parts as it resolves it; `why` reads the
// latest fight's recent rounds back in plain words. It only reads: it is
// free, instant, and works in or out of a fight.

// whyShown is how many rounds a bare `why` explains.
const whyShown = 3

func Why(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	rolls := combatstream.DefaultRollLog().Recent(user.UserId, 0)
	if len(rolls) == 0 {
		user.SendText(`There is no battle line to explain yet. After a fight, <ansi fg="command">why</ansi> tells how its latest blows landed, missed or were turned aside (<ansi fg="command">help battlelog</ansi>).`)
		return true, nil
	}

	arg := strings.ToLower(strings.TrimSpace(rest))
	switch {
	case arg == ``:
		n := min(whyShown, len(rolls))
		user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">The last %d rounds, newest first</ansi> (<ansi fg="command">why list</ansi> numbers them, <ansi fg="command">why [number]</ansi> explains one):`, n))
		for i := 0; i < n; i++ {
			whySend(user, i+1, rolls[i])
		}
	case arg == `list`:
		user.SendText(`<ansi fg="yellow-bold">Recent rounds, newest first:</ansi>`)
		for i, r := range rolls {
			heading := r.Describe(user.UserId)[0]
			user.SendText(fmt.Sprintf(`  %2d. %s`, i+1, heading))
		}
	default:
		n, err := strconv.Atoi(arg)
		if err != nil || n < 1 {
			user.SendText(`Type <ansi fg="command">why</ansi> for the latest rounds, <ansi fg="command">why list</ansi> to number them, or <ansi fg="command">why [number]</ansi> for one.`)
			return true, nil
		}
		if n > len(rolls) {
			user.SendText(fmt.Sprintf(`Only the last %d rounds are kept. Type <ansi fg="command">why list</ansi> to see them.`, len(rolls)))
			return true, nil
		}
		whySend(user, n, rolls[n-1])
	}
	return true, nil
}

func whySend(user *users.UserRecord, n int, r combatstream.Roll) {
	lines := r.Describe(user.UserId)
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">%d.</ansi> %s`, n, lines[0]))
	for _, l := range lines[1:] {
		user.SendText(l)
	}
}
