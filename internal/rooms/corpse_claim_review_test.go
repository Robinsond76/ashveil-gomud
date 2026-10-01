package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 33d review: a claimed shared-battle corpse is made even when corpses are
// disabled, but decay skipped every corpse then, so it never went away.
func TestClaimedCorpseDecaysWithCorpsesDisabled(t *testing.T) {
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = false
	gameplay.Death.CorpseDecayTime = "1 hour"
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	room := NewEmptyRoom()
	c := Corpse{ClaimUserId: 7, MobId: 3, Character: *characters.New(), RoundCreated: 1, Gold: 4}
	c.Character.Name = "bandit"
	room.AddCorpse(c)
	room.UpdateCorpses(2)
	require.Len(t, room.Corpses, 1, "fresh claimed loot stays")
	room.UpdateCorpses(1_000_000)
	assert.Empty(t, room.Corpses, "decayed claimed loot goes")
}

// 33d review: look-alike corpses (one group, one round) were removed by
// mob, name and round alone, so burying an empty one could take the twin
// that holds claimed loot.
func TestRemoveCorpseKeepsTheClaimedTwin(t *testing.T) {
	room := NewEmptyRoom()
	claimed := Corpse{ClaimUserId: 7, MobId: 3, Character: *characters.New(), RoundCreated: 5, Gold: 4}
	claimed.Character.Name = "bandit"
	empty := Corpse{MobId: 3, Character: *characters.New(), RoundCreated: 5}
	empty.Character.Name = "bandit"
	room.AddCorpse(claimed)
	room.AddCorpse(empty)
	require.True(t, room.RemoveCorpse(empty))
	require.Len(t, room.Corpses, 1)
	assert.Equal(t, 7, room.Corpses[0].ClaimUserId)
}
