package spells

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// jsString matches a single-quoted, double-quoted, or template string
// literal in a spell script.
var jsString = regexp.MustCompile(`'(?:[^'\\\n]|\\.)*'|"(?:[^"\\\n]|\\.)*"|` + "`[^`]*`")

// TestShippedSpellTextVoice (Phase 29c): no string in a shipped spell
// script breaks the narration voice (no exclamation mark, no ALL-CAPS
// word outside markup).
func TestShippedSpellTextVoice(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "spells")
	files, err := filepath.Glob(filepath.Join(dir, "*.js"))
	if err != nil || len(files) < 10 {
		t.Fatalf("want the shipped spell scripts, got %v (%v)", files, err)
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range jsString.FindAllString(string(data), -1) {
			if problem := util.NarrationVoiceProblem(literal); problem != "" {
				t.Errorf("%s: %s in %s", filepath.Base(path), problem, literal)
			}
		}
	}
}
