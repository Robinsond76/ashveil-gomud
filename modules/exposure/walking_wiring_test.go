package exposure

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// The real walking module registers itself as internal/walking's step
	// provider, with its native seams.
	_ "github.com/GoMudEngine/GoMud/modules/walking"
)

type fatigueRecorder struct{ total map[int]int }

func (f *fatigueRecorder) ApplyMemberDrain(leader int, key survival.MemberKey, cost survival.Exertion) (survival.ExertionResult, error) {
	if key == survival.LeaderMemberKey {
		f.total[leader] += cost.Fatigue
	}
	return survival.ExertionResult{Member: key}, nil
}

// TestExposureRegistryFeedsWalkingColdMultiplier: a real exposure value in
// this module's registry, read through the climate seam by the real
// walking module, raises the strain of a real step.
func TestExposureRegistryFeedsWalkingColdMultiplier(t *testing.T) {
	e := setup(t)
	for _, r := range []*rooms.Room{
		{RoomId: 94001, Zone: "Wood", Biome: "forest"},
		{RoomId: 94002, Zone: "Wood", Biome: "forest"},
	} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	climate.SetProvider(e.m)
	recorder := &fatigueRecorder{total: map[int]int{}}
	survival.SetMemberDrainService(recorder)
	t.Cleanup(func() {
		climate.SetProvider(nil)
		survival.SetMemberDrainService(nil)
	})

	e.addUser(t, 41, 94002)
	e.addUser(t, 42, 94002)
	e.m.registry.Exposure[41] = map[string]int{string(survival.LeaderMemberKey): -55} // frostbitten

	for i := 0; i < 4; i++ {
		walking.Stepped(41, 94001, 94002)
		walking.Stepped(42, 94001, 94002)
	}
	require.NotEmpty(t, recorder.total, "the real walking module is wired")
	assert.Equal(t, 3, recorder.total[41], "frostbitten: 4 × 75 strain")
	assert.Equal(t, 2, recorder.total[42], "comfortable: 4 × 50 strain")
}
