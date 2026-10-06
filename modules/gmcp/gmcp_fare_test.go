package gmcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 50: Company vitals carry each member's battle condition (the Company
// panel's "In battle" line), and the battle feed carries what each member
// went in with (the battle screen caption and the Combat tab's notes).
func TestCompanyVitalsCarryTheBattleCondition(t *testing.T) {
	s := sampleCompany()
	s.Leader.Fare = "Hungry: -5% damage"
	p, ok := buildCompanyPayload(7, s, noChemistry)
	require.True(t, ok)
	data, _ := json.Marshal(p)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	vitals := got["vitals"].(map[string]any)
	assert.Equal(t, "Hungry: -5% damage", vitals["leader"].(map[string]any)["fare"])
	assert.Nil(t, vitals["companion:1"].(map[string]any)["fare"], "omitted when nothing")

	f, out := testFeed()
	f.update(7, s)
	s.Leader.Fare = ""
	f.update(7, s)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company.Vitals", (*out)[1].module, "a changed condition sends vitals only")
}

func TestBattleFeedCarriesTheFare(t *testing.T) {
	raw, _ := json.Marshal(buildBattle(battleFacts{InBattle: true, Fare: map[string]string{"leader": "Starving: -10% damage"}}))
	assert.Contains(t, string(raw), `"fare":{"leader":"Starving: -10% damage"}`)
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true, Dark: true, Fare: map[string]string{"leader": "Parched: -10% tempo"}}))
	assert.Contains(t, string(raw), `"fare":{"leader"`, "the dark hides the foes, not your own")
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true}))
	assert.False(t, strings.Contains(string(raw), "fare"))
}
