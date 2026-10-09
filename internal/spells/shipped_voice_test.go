package spells

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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

// TestShippedChantCountsMatchWaitRounds (Phase 87 review): a script's
// opening "(chanting: X, N turns)" count is its WAIT_ROUNDS, so it must be
// the spell's own waitrounds, or the count skips a number before the spell
// lands (Minor Heal said 3 turns, then 1, then landed).
func TestShippedChantCountsMatchWaitRounds(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "spells")
	files, err := filepath.Glob(filepath.Join(dir, "*.js"))
	if err != nil || len(files) < 10 {
		t.Fatalf("want the shipped spell scripts, got %v (%v)", files, err)
	}
	jsWait := regexp.MustCompile(`(?m)^WAIT_ROUNDS = (\d+);`)
	yamlWait := regexp.MustCompile(`(?m)^waitrounds: (\d+)$`)
	checked := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		js := jsWait.FindSubmatch(data)
		if js == nil {
			continue
		}
		spec, err := os.ReadFile(strings.TrimSuffix(path, ".js") + ".yaml")
		if err != nil {
			t.Fatalf("%s has no spell file: %v", filepath.Base(path), err)
		}
		y := yamlWait.FindSubmatch(spec)
		want := "0"
		if y != nil {
			want = string(y[1])
		}
		if string(js[1]) != want {
			t.Errorf("%s: WAIT_ROUNDS = %s, but its waitrounds is %s", filepath.Base(path), js[1], want)
		}
		checked++
	}
	if checked < 10 {
		t.Fatalf("checked only %d scripts", checked)
	}
}
