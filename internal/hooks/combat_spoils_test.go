package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestBattleStartClearsStaleSpoils: spoils noted outside any fight (a
// death processed with no battle open) never reach the next fight's
// summary (Phase 37 review).
func TestBattleStartClearsStaleSpoils(t *testing.T) {
	battle.Reset()
	t.Cleanup(battle.Reset)
	u := users.NewUserRecord(987301, 0)
	u.Character.Name = "Hero"
	loot.NoteSpoils(u.UserId, "a stale sword")

	sd := side{user: u, allies: map[int]bool{}}
	sd.beginBattle(mobparty.Party{ID: "spoils-test"}, &rooms.Room{RoomId: 987301}, 1)

	assert.Empty(t, loot.TakeSpoils(u.UserId), "the new fight starts with no spoils")
}
