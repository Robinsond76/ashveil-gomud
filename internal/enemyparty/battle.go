package enemyparty

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
)

// BattleParty finds a battle's group among a room's parties: the one
// sharing an enemy with it. The combat round and the web client's battle
// view (32g2) both find it this way.
func BattleParty(b battle.Battle, parties []mobparty.Party) (mobparty.Party, bool) {
	for _, p := range parties {
		for _, instanceId := range p.Members {
			if b.Has(instanceId) {
				return p, true
			}
		}
	}
	return mobparty.Party{}, false
}
