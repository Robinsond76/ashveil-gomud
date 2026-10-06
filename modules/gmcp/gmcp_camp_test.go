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
		"rest_percent": 25.0, "rest_seconds": 45.0, "can_camp": false, "inn": false, "room_id": 0.0, "allied_camps": []any{}}, got)

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
