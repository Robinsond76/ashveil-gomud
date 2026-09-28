package items

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// TestShippedWeaponTextVoice (Phase 29c): every line of every shipped
// weapon pool is in the narration voice, carries no {damage} (the combat
// code appends it), and every group still has every pool.
func TestShippedWeaponTextVoice(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "combat-messages")
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil || len(files) != 8 {
		t.Fatalf("want the eight shipped weapon files, got %v (%v)", files, err)
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var g WeaponAttackMessageGroup
		if err := yaml.Unmarshal(data, &g); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if err := g.Validate(); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		for intensity, opts := range g.Options {
			pools := map[string]MessageOptions{
				"together.toattacker":     opts.Together.ToAttacker,
				"together.todefender":     opts.Together.ToDefender,
				"together.toroom":         opts.Together.ToRoom,
				"separate.toattacker":     opts.Separate.ToAttacker,
				"separate.todefender":     opts.Separate.ToDefender,
				"separate.toattackerroom": opts.Separate.ToAttackerRoom,
				"separate.todefenderroom": opts.Separate.ToDefenderRoom,
			}
			for _, name := range []string{"together.toattacker", "together.todefender", "together.toroom"} {
				if len(pools[name]) == 0 {
					t.Errorf("%s %s %s: empty", filepath.Base(path), intensity, name)
				}
			}
			for name, lines := range pools {
				for _, line := range lines {
					where := filepath.Base(path) + " " + string(intensity) + " " + name
					if p := util.NarrationVoiceProblem(string(line)); p != "" {
						t.Errorf("%s: %s in %q", where, p, line)
					}
					if strings.Contains(string(line), "{damage}") {
						t.Errorf("%s: {damage} in %q", where, line)
					}
				}
			}
		}
	}
}
