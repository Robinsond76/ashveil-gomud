package spells

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/spellpower"
	"gopkg.in/yaml.v2"
)

// TestShippedSpellPower (Phase 35b): the scaling spells carry 35a2's sizes
// in their power blocks, which validate, and a bad block is refused.
func TestShippedSpellPower(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "spells")
	want := map[string]spellpower.Power{
		"mm":      {Base: 7, Dice: "1d6", LevelDiv: 10, MysticismDiv: 15},
		"sparks":  {Base: 4, Dice: "1d4", LevelDiv: 15, MysticismDiv: 25},
		"hex":     {Base: 8, Dice: "2d4", LevelDiv: 8, MysticismDiv: 12},
		"heal":    {Base: 8, Dice: "2d4", LevelDiv: 6},
		"healall": {Base: 8, Dice: "2d4", LevelDiv: 6, Percent: 55},
	}
	for id, power := range want {
		data, err := os.ReadFile(filepath.Join(dir, id+".yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var sp SpellData
		if err := yaml.Unmarshal(data, &sp); err != nil {
			t.Fatal(err)
		}
		if err := sp.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
		if sp.Power == nil || *sp.Power != power {
			t.Errorf("%s power = %+v, want %+v", id, sp.Power, power)
		}
	}
	bad := SpellData{SpellId: "x", Power: &spellpower.Power{Base: 1, Dice: "2x4"}}
	if bad.Validate() == nil {
		t.Error("a bad power block is refused")
	}
}
