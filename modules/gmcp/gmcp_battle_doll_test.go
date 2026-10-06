package gmcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39d review: the battle feed lists the company's standing dolls by
// their member keys (the ids their events use and their keys in positions),
// with their names and health, so the Combat tab and the battle screen show
// them; a doll elsewhere or broken is left out.
func TestBattleFeedListsTheCompanysDolls(t *testing.T) {
	company.ResetDollsForTest()
	t.Cleanup(company.ResetDollsForTest)
	add := func(id, room, hp int, name string) {
		m := &mobs.Mob{InstanceId: id}
		m.Character.Name, m.Character.RoomId = name, room
		m.Character.Health, m.Character.HealthMax.Value = hp, 40
		mobs.SetTestInstance(m)
		t.Cleanup(func() { mobs.RemoveTestInstance(id) })
	}
	add(9301, 100, 30, "Pip")
	add(9302, 100, 0, "Pim")   // broken this round
	add(9303, 200, 40, "Pod")  // not in the battle's room
	add(9304, 100, 25, "Bolt") // a companion Master's doll
	company.RegisterDoll(9301, 7, company.LeaderMemberKey, 0)
	company.RegisterDoll(9302, 7, company.LeaderMemberKey, 1)
	company.RegisterDoll(9303, 7, company.LeaderMemberKey, 2)
	company.RegisterDoll(9304, 7, "companion:3", 0)
	company.RegisterDoll(9305, 8, company.LeaderMemberKey, 0) // another leader's

	got := gatherDolls(7, 100)
	require.Len(t, got, 2)
	assert.Equal(t, battleDoll{Key: "doll:companion:3:0", Name: "Bolt", Master: "companion:3", Health: 25, HealthMax: 40}, got[0])
	assert.Equal(t, battleDoll{Key: "doll:leader:0", Name: "Pip", Master: "leader", Health: 30, HealthMax: 40}, got[1])

	raw, _ := json.Marshal(buildBattle(battleFacts{InBattle: true, Dolls: got}))
	assert.True(t, strings.Contains(string(raw), `"dolls":[{"key":"doll:companion:3:0","name":"Bolt","master":"companion:3","hp":25,"hp_max":40}`), string(raw))
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true}))
	assert.NotContains(t, string(raw), "dolls")
}
