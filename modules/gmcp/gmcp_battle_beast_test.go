package gmcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39e: the battle feed lists the company's standing beasts beside its
// dolls, with their kind, so the Combat tab and the battle screen draw them.
func TestBattleFeedListsTheCompanysBeasts(t *testing.T) {
	company.ResetBeastsForTest()
	t.Cleanup(company.ResetBeastsForTest)
	add := func(id, room, hp int, name, kind string) {
		m := &mobs.Mob{InstanceId: id}
		m.Character.Name, m.Character.RoomId = name, room
		m.Character.Health, m.Character.HealthMax.Value = hp, 40
		m.Character.RT = &characters.ClassRT{Beast: &characters.BeastInfo{Kind: kind}}
		mobs.SetTestInstance(m)
		t.Cleanup(func() { mobs.RemoveTestInstance(id) })
	}
	add(9401, 100, 30, "Ash", "wolf")
	add(9402, 100, 0, "Fang", "warhound") // fell this round
	add(9403, 200, 40, "Bruin", "bear")   // not in the battle's room
	company.RegisterBeast(9401, 7, company.LeaderMemberKey)
	company.RegisterBeast(9402, 7, "companion:3")
	company.RegisterBeast(9403, 7, "companion:4")
	company.RegisterBeast(9404, 8, company.LeaderMemberKey) // another leader's

	got := gatherDolls(7, 100)
	require.Len(t, got, 1)
	assert.Equal(t, battleDoll{Key: "beast:leader", Name: "Ash", Master: "leader", Health: 30, HealthMax: 40, Kind: "wolf"}, got[0])
	raw, _ := json.Marshal(buildBattle(battleFacts{InBattle: true, Dolls: got}))
	assert.True(t, strings.Contains(string(raw), `"dolls":[{"key":"beast:leader","name":"Ash","master":"leader","hp":30,"hp_max":40,"kind":"wolf"}`), string(raw))
}
