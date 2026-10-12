package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
)

// The shipped buff files need the full config, which stays loaded for later
// tests in this package, so this file sorts (and runs) last.
func loadShippedBuffs(t *testing.T) {
	t.Helper()
	wd, _ := os.Getwd()
	previous := configs.GetFilePathsConfig().DataFiles.String()
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		configs.SetTestDataFiles(previous)
	})
	_, thisFile, _, _ := runtime.Caller(0)
	if err := os.Chdir(filepath.Join(filepath.Dir(thisFile), "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	mudlog.SetupLogger(nil, "", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
}

// A foe put to sleep (or stunned, or felled) earlier in the round by a unit
// that went first must lose the turn that has not come yet, not act and
// lose only the next round's.
func TestDisablingStatusLandedThisRoundTakesTheHoldersTurn(t *testing.T) {
	loadShippedBuffs(t)
	for _, id := range []int{status.Asleep, status.Stunned, status.Staggered, status.Paralyzed, status.KnockedDown} {
		ResetTempoForTest()
		t.Cleanup(ResetTempoForTest)
		m := engagementMob(t, 8970+id%10, 20, 1)
		m.Character = *characters.New()
		m.Character.Health, m.Character.RoomId = 20, 1
		m.Character.AddBuff(id, false)
		_, lost := status.LostAction(&m.Character)
		assert.False(t, lost, "status %d: the old rule costs nothing before the first tick", id)

		assert.True(t, tempoStatusCostsAction(mobHolder(m)), "status %d takes this round's turn", id)
		assert.True(t, tempoStatusCostsAction(mobHolder(m)), "status %d: any extra turn is lost too", id)

		clear(tempoLostTold)
		status.Tick(&m.Character)
		_, lost = status.LostAction(&m.Character)
		switch id {
		case status.KnockedDown, status.Staggered, status.Paralyzed:
			assert.False(t, lost, "status %d costs one action in all, and this round's was it", id)
		default:
			assert.True(t, lost, "status %d keeps costing turns while it lasts", id)
		}
	}
}

func TestNoStatusNoLostTurn(t *testing.T) {
	loadShippedBuffs(t)
	ResetTempoForTest()
	t.Cleanup(ResetTempoForTest)
	c := characters.New()
	_, lost := status.FreshLostAction(c, false)
	assert.False(t, lost)
}
