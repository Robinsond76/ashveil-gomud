package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEarnedExperienceSurvivesSnapshotAndRestore (Phase 32e): a companion
// that earned XP and a level is captured by the snapshot the save seams
// use, and a fresh mob built from that state comes back at the same level
// and experience.
func TestEarnedExperienceSurvivesSnapshotAndRestore(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	live := &mobs.Mob{}
	live.InstanceId = 880101
	live.Character = *characters.New()
	live.Character.Level = 1
	live.Character.Validate(true)
	live.Character.GrantXP(live.Character.XPTL(2))
	for {
		if gained, _ := live.Character.LevelUp(); !gained {
			break
		}
	}
	require.GreaterOrEqual(t, live.Character.Level, 2)
	mobs.SetTestInstance(live)
	t.Cleanup(func() { mobs.RemoveTestInstance(880101) })

	state, ok := nativeRuntime{}.Snapshot(live.InstanceId)
	require.True(t, ok)

	reborn := &mobs.Mob{}
	reborn.Character = *characters.New()
	reborn.Character.Level = 1
	applyState(reborn, state)

	assert.Equal(t, live.Character.Level, reborn.Character.Level)
	assert.Equal(t, live.Character.Experience, reborn.Character.Experience)
	into, tnl := reborn.Character.XPTNLActual()
	wantInto, wantTnl := live.Character.XPTNLActual()
	assert.Equal(t, wantInto, into)
	assert.Equal(t, wantTnl, tnl)
}
