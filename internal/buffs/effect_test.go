package buffs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// The default world's buffs say whether they harm or help through stat
// modifiers and their flags' data (Phase 34 review: Bleeding, Poisoned and
// Stunned have no stat penalty and were shown as Helpful).
func TestDefaultWorldBuffEffects(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	set := func(value string) error {
		flat := configs.Flatten(configs.GetOverrides())
		flat["FilePaths.DataFiles"] = value
		return configs.RestoreOverrides(flat)
	}
	previous := configs.GetFilePathsConfig().DataFiles.String()
	savedFlags := flagSpecs
	t.Cleanup(func() {
		flagSpecs = savedFlags
		require.NoError(t, set(previous))
	})
	require.NoError(t, set(dir))
	LoadFlagDataFiles()
	// The buff files are read directly: Effect needs only their modifiers,
	// flags and secrecy, not the timing configuration validation uses.
	read := func(file string) *BuffSpec {
		raw, err := os.ReadFile(filepath.Join(dir, "buffs", file))
		require.NoError(t, err)
		var spec BuffSpec
		require.NoError(t, yaml.Unmarshal(raw, &spec))
		return &spec
	}

	for file, want := range map[string]string{
		"1100-bleeding.yaml":   "harmful", // a flag only
		"13-poisoned.yaml":     "harmful", // a flag only
		"1107-stunned.yaml":    "harmful", // lost actions
		"10-weakness.yaml":     "harmful", // negative modifiers
		"17-well_fed.yaml":     "helpful", // positive modifiers
		"29-night_vision.yaml": "helpful", // a flag only
		"9-hidden.yaml":        "",        // a marker, neither
		"1-illumination.yaml":  "helpful", // a light source
	} {
		spec := read(file)
		harmful, helpful := spec.Effect()
		got := ""
		if harmful {
			got = "harmful"
		} else if helpful {
			got = "helpful"
		}
		assert.Equal(t, want, got, spec.Name)
	}
	secret := *read("1100-bleeding.yaml")
	secret.Secret = true
	harmful, helpful := secret.Effect()
	assert.False(t, harmful || helpful, "a secret buff reveals neither")
}
