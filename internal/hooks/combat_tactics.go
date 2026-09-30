package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 30c: company tactics. A focus ordered mid-battle (company tactics
// focus, internal/battle) waits for the next round's upkeep, which turns
// every member of that player's side at once onto the new focus's choice
// (the owner's decision 5), instead of only as foes fall. The set of
// players turning this round is game-loop state, taken after the battle
// pass and dropped at the round's end.

// refocusing holds the players whose company turns by a new focus this
// round. Game loop only.
var refocusing map[int]bool

// beginRefocus takes this round's focus orders and reports each on the
// combat event stream.
func beginRefocus() {
	refocusing = nil
	for _, uid := range battle.TakeRefocus() {
		if refocusing == nil {
			refocusing = map[int]bool{}
		}
		refocusing[uid] = true
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil {
			continue
		}
		// The order's own fight, so the event lands on the right one.
		fightID := uint64(0)
		if b, ok := battle.Current(uid); ok {
			fightID = b.FightID
		}
		rule := string(strategy.NoFocus)
		if r, ok := enemyparty.Focus(uid); ok {
			rule = string(r)
		}
		emitCombat(combatstream.Event{Kind: combatstream.FocusChange, FightID: fightID, RoomId: u.Character.RoomId, Source: userRef(u), Rule: rule})
	}
}

// endRefocus drops the round's refocus set.
func endRefocus() { refocusing = nil }
