package templates

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadFileWithNoFileSystems: with no plugin file systems registered, a
// template is not found there (it once read as found and empty, so a help
// page's .md variant shadowed its .template and rendered blank).
func TestReadFileWithNoFileSystems(t *testing.T) {
	saved := fileSystems
	t.Cleanup(func() { fileSystems = saved })

	fileSystems = nil
	_, err := readFile("templates/help/anything.md")
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	fileSystems = []fs.ReadFileFS{fstest.MapFS{"templates/help/x.md": {Data: []byte("hello")}}}
	b, err := readFile("templates/help/x.md")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	_, err = readFile("templates/help/y.md")
	assert.Error(t, err)
}
