package modules

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// worldOnly are the modules that save state but none of it by user, so a
// purged user leaves nothing behind in them. Each needs its reason.
var worldOnly = map[string]string{
	"market":  "stock and news by zone",
	"weather": "weather by zone",
}

// inMemoryPerUser are modules that save nothing but hold state by user
// in memory, which a purge must drop too.
var inMemoryPerUser = []string{"death", "tutorial"}

// TestEveryModuleWithUserStateHandlesThePurge (Phase 32b): a module that
// saves state (SetOnSave) must listen for events.UserPurged, so a purged
// user (a tutorial replay; later, a deleted character) leaves nothing
// behind; a new module can't forget. A module whose state is only about
// the world is listed in worldOnly with its reason.
func TestEveryModuleWithUserStateHandlesThePurge(t *testing.T) {
	dirs, err := os.ReadDir(".")
	require.NoError(t, err)
	var saving []string
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		source := moduleSource(t, d.Name())
		if strings.Contains(source, "SetOnSave(") {
			saving = append(saving, d.Name())
		}
		if !strings.Contains(source, "SetOnSave(") {
			continue
		}
		if reason, ok := worldOnly[d.Name()]; ok {
			assert.False(t, strings.Contains(source, "events.UserPurged{}"), "%s is listed as world-only (%s) but handles the purge", d.Name(), reason)
			continue
		}
		assert.True(t, strings.Contains(source, "events.UserPurged{}"), "module %s saves state but doesn't handle events.UserPurged", d.Name())
	}
	sort.Strings(saving)
	assert.Contains(t, saving, "company", "the scan found the modules")
	for _, name := range inMemoryPerUser {
		assert.True(t, strings.Contains(moduleSource(t, name), "events.UserPurged{}"), "module %s holds state by user but doesn't handle events.UserPurged", name)
	}
}

// moduleSource is a module's non-test Go source, concatenated.
func moduleSource(t *testing.T, dir string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	var b strings.Builder
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		b.Write(data)
	}
	return b.String()
}
