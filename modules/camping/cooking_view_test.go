package camping

import (
	"encoding/json"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestManualCookingCapabilityUsesRecipeRanksAndRealRefresh(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Recipes = []campRecipe{{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1}}
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29] = 1
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	camping.SetMovementProvider(w.m)
	t.Cleanup(func() { camping.SetMovementProvider(nil) })
	v, known := camping.CookingCapability(7)
	require.True(t, known)
	assert.False(t, v.Ready)
	assert.Contains(t, v.Reason, "cooking rank 1")
	w.user.Character.Skills = map[string]int{"cooking": 1}
	v, known = camping.CookingCapability(7)
	require.True(t, known)
	assert.True(t, v.Ready, "a cook without any class choice is eligible")
	assert.Equal(t, 1, cargo.stacks[29], "read does not consume ingredients")
	w.m.campCfg.Recipes[0].MinLevel = 2
	v, _ = camping.CookingCapability(7)
	assert.False(t, v.Ready)
	assert.Contains(t, v.Reason, "cooking rank 2")
	assert.Contains(t, v.Description, "cooking rank 2")
	w.user.Character.Skills["cooking"] = 2
	gmcp.AcceptGMCPForTest(w.user.ConnectionId())
	var received struct {
		Utility []struct {
			ID, Group, Mode, Description, Reason string
			Rank                                 int
			Enabled                              bool
		}
	}
	count := 0
	listener := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module == "Char.Capabilities" {
			assert.Equal(t, 7, out.UserId)
			require.NoError(t, json.Unmarshal(out.Payload.([]byte), &received))
			count++
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, listener) })
	companyview.RefreshUser(7)
	events.ProcessEvents()
	require.Positive(t, count)
	var found bool
	for _, c := range received.Utility {
		if c.ID == "cooking" {
			found = true
			assert.Equal(t, "manual", c.Mode)
			assert.Equal(t, "camp", c.Group)
			assert.True(t, c.Enabled)
			assert.Equal(t, 2, c.Rank)
		}
	}
	require.True(t, found)
	before := count
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, before, count, "unchanged recipe/rank view suppressed")
	_, err := w.m.userCommand("cook", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Equal(t, 0, cargo.stacks[29])
	assert.Equal(t, 1, cargo.stacks[30021])
	v, _ = camping.CookingCapability(7)
	assert.False(t, v.Ready)
	assert.Contains(t, v.Reason, "Missing recipe ingredients")
}
