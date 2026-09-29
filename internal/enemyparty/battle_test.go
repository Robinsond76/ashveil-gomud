package enemyparty

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
)

// TestBattleParty (32g2): the battle's group is the party sharing an enemy
// with it.
func TestBattleParty(t *testing.T) {
	b := battle.Battle{Enemies: map[int]bool{12: true, 13: true}}
	parties := []mobparty.Party{{ID: "x", Members: []int{10, 11}}, {ID: "y", Members: []int{13, 14}}}
	p, ok := BattleParty(b, parties)
	assert.True(t, ok)
	assert.Equal(t, "y", p.ID)

	_, ok = BattleParty(b, parties[:1])
	assert.False(t, ok)
	_, ok = BattleParty(battle.Battle{}, parties)
	assert.False(t, ok)
}
