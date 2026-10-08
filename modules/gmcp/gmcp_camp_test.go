package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanyCampPayload (Phase 32g): the camp's state as the Camp tab
// reads it, nothing without a provider, and a rest's countdown resending.
func TestCompanyCampPayload(t *testing.T) {
	state := camping.CampState{HasCamp: true, Here: true, FireLit: true, Resting: true, RestPercent: 25, RestSeconds: 45}
	extra := campExtra(func(int, int, []string) (camping.CampState, bool) { return state, true }, nil)
	u := users.NewUserRecord(7, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, map[string]any{"has_camp": true, "here": true, "room": "", "fire_lit": true, "resting": true, "rested": false,
		"embers": false, "tent": false, "tents": []any{}, "gear": []any{}, "supplies": []any{}, "prepared": []any{}, "theft_risk": false, "rest_percent": 25.0, "rest_seconds": 45.0, "can_camp": false, "inn": false, "room_id": 0.0, "allied_camps": []any{}, "duties": []any{}, "duties_locked": false, "recipes": []any{}, "recipe_book": []any{}}, got)

	// Phase 40a3: a finished rest leaves embers, and a pitched tent shows.
	state = camping.CampState{HasCamp: true, Here: true, Rested: true, Embers: true, Tent: true}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, true, got["embers"])
	assert.Equal(t, true, got["tent"])

	// Phase 40a4: the camp gear the company carries.
	state = camping.CampState{HasCamp: true, Here: true, Gear: []string{"Tent", "Bells and trip lines"}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{"Tent", "Bells and trip lines"}, got["gear"])
	assert.Equal(t, false, got["fire_lit"])

	// Phase 43a: camp supplies carried, and what is queued for the rest.
	state = camping.CampState{HasCamp: true, Here: true, Supplies: []string{"fortifying broth x2"}, Prepared: []string{"fortifying broth for Tamsin", "watch incense"}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{"fortifying broth x2"}, got["supplies"])
	assert.Equal(t, []any{"fortifying broth for Tamsin", "watch incense"}, got["prepared"])

	// Phase 51: each member's rest duty and the duties they can take.
	state = camping.CampState{HasCamp: true, Here: true, DutiesLocked: true, Duties: []camping.DutyRow{
		{Key: "leader", Name: "Aria", Command: "me", Duty: "sleep", Options: []string{"sleep", "watch"}},
		{Key: "companion:1", Name: "Mira", Command: "Mira", Duty: "watch", Options: []string{"sleep", "watch"}},
	}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, true, got["duties_locked"])
	assert.Equal(t, []any{
		map[string]any{"key": "leader", "name": "Aria", "command": "me", "duty": "sleep", "options": []any{"sleep", "watch"}},
		map[string]any{"key": "companion:1", "name": "Mira", "command": "Mira", "duty": "watch", "options": []any{"sleep", "watch"}},
	}, got["duties"])
	// Phase 56: the dishes the leader has learned.
	state = camping.CampState{Recipes: []string{"hunter's stew: 2 raw game meat, 1 wild thyme (cooking 3)"}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{"hunter's stew: 2 raw game meat, 1 wild thyme (cooking 3)"}, got["recipes"])
	state = camping.CampState{RecipeBook: []camping.RecipeRow{{Name: "hunter's stew", Kind: "dish", Skill: "cooking", Level: 3, Ready: true,
		Needs: []camping.RecipeNeed{{Name: "raw game meat", Count: 2, Have: 2}}}}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{map[string]any{"name": "hunter's stew", "kind": "dish", "skill": "cooking", "level": float64(3), "ready": true,
		"needs": []any{map[string]any{"name": "raw game meat", "count": float64(2), "have": float64(2)}}}}, got["recipe_book"])
	// Camp activities: the chores before sleeping, with their notes.
	state = camping.CampState{HasCamp: true, Here: true, Activities: []camping.ActivityRow{
		{Key: "sharpen", Label: "Sharpen", Command: "camp sharpen", Note: "No blades are dull enough to need the whetstone."},
		{Key: "cook", Label: "Cook", Command: "camp cook", Note: "You cook a stew.", Ready: true},
	}}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{
		map[string]any{"key": "sharpen", "label": "Sharpen", "command": "camp sharpen", "note": "No blades are dull enough to need the whetstone.", "ready": false},
		map[string]any{"key": "cook", "label": "Cook", "command": "camp cook", "note": "You cook a stew.", "ready": true},
	}, got["activities"])
	state = camping.CampState{HasCamp: true, Here: true, FireLit: true, Resting: true, RestPercent: 25, RestSeconds: 45}

	none := campExtra(func(int, int, []string) (camping.CampState, bool) { return camping.CampState{}, false }, nil)
	assert.Nil(t, none.build(u), "no provider: nothing sent")

	f, out := testFeed()
	f.extras = []companyExtra{extra}
	f.updateExtras(u)
	state.RestSeconds = 41
	f.updateExtras(u)
	f.updateExtras(u)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company.Camp", (*out)[1].module)
}

// Phase 52: the pitched tent and the tents carried, for the Camp tab's
// picker. A camp saved before tent kinds (Tent, no kind) reads as canvas.
func TestCompanyCampPayloadCarriesTheTents(t *testing.T) {
	state := camping.CampState{HasCamp: true, Here: true, Tent: true, TentKind: camping.TentLarge, TentNote: "wake Well Rested",
		Tents: []camping.TentChoice{
			{Kind: camping.TentCanvas, Name: "oiled canvas tent", Effect: "shelter"},
			{Kind: camping.TentLarge, Name: "large pavilion tent", Effect: "wake Well Rested", Pitched: true},
		}}
	extra := campExtra(func(int, int, []string) (camping.CampState, bool) { return state, true }, nil)
	u := users.NewUserRecord(7, 1)

	var got struct {
		TentKind string `json:"tent_kind"`
		TentName string `json:"tent_name"`
		TentNote string `json:"tent_note"`
		Tents    []struct {
			Kind    string `json:"kind"`
			Name    string `json:"name"`
			Pitched bool   `json:"pitched"`
			Command string `json:"command"`
		} `json:"tents"`
	}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, "large", got.TentKind)
	assert.Equal(t, "large pavilion tent", got.TentName)
	assert.Equal(t, "wake Well Rested", got.TentNote)
	require.Len(t, got.Tents, 2)
	assert.Equal(t, "camp tent canvas", got.Tents[0].Command)
	assert.False(t, got.Tents[0].Pitched)
	assert.True(t, got.Tents[1].Pitched)
	assert.Equal(t, "camp tent large", got.Tents[1].Command)

	state = camping.CampState{HasCamp: true, Here: true, Tent: true}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, "canvas", got.TentKind, "an old camp's tent is canvas")
}

// TestCompanyCampPayloadCarriesTheInnRooms (Phase 75): each room an inn
// lets, with its price and the command that buys it.
func TestCompanyCampPayloadCarriesTheInnRooms(t *testing.T) {
	state := camping.CampState{Inn: true, InnRooms: []camping.InnRoomRow{
		{Tier: camping.InnCommon, Price: 10, Minutes: 30},
		{Tier: camping.InnSuite, Price: 80, Minutes: 120},
	}}
	extra := campExtra(func(int, int, []string) (camping.CampState, bool) { return state, true }, nil)
	var got map[string]any
	require.NoError(t, json.Unmarshal(extra.build(users.NewUserRecord(7, 1)), &got))
	assert.Equal(t, []any{
		map[string]any{"tier": "common", "name": "Common", "price": 10.0, "minutes": 30.0, "command": "inn rest"},
		map[string]any{"tier": "suite", "name": "Suite", "price": 80.0, "minutes": 120.0, "command": "inn rest suite"},
	}, got["inn_rooms"])
	state = camping.CampState{}
	got = nil
	require.NoError(t, json.Unmarshal(extra.build(users.NewUserRecord(7, 1)), &got))
	assert.NotContains(t, got, "inn_rooms", "nothing away from an inn")
}
