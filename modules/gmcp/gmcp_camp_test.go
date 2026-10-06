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
		"embers": false, "tent": false, "gear": []any{}, "supplies": []any{}, "prepared": []any{}, "theft_risk": false, "rest_percent": 25.0, "rest_seconds": 45.0, "can_camp": false, "inn": false, "room_id": 0.0, "allied_camps": []any{}, "duties": []any{}, "duties_locked": false}, got)

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
