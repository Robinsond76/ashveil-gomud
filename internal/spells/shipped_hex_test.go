package spells

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v2"
)

// TestShippedWitheringHex (Phase 30d1): the goblin hexer's spell is a
// two-round harmful chant, with its script beside it.
func TestShippedWitheringHex(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "spells")
	data, err := os.ReadFile(filepath.Join(dir, "hex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var hex SpellData
	if err := yaml.Unmarshal(data, &hex); err != nil {
		t.Fatal(err)
	}
	if hex.SpellId != "hex" || hex.Name != "Withering Hex" || hex.Type != HarmSingle || hex.Cost != 8 || hex.WaitRounds != 1 {
		t.Errorf("hex = %+v, want Withering Hex, harmsingle, cost 8, waitrounds 1", hex)
	}
	if _, err := os.Stat(filepath.Join(dir, "hex.js")); err != nil {
		t.Errorf("hex.js: %v", err)
	}
}
