package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestAlliancePurgeWaitsForSuccessfulSave(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	t.Cleanup(parties.UseMemoryForTest())
	dir := t.TempDir()
	path := filepath.Join(dir, "state", "alliances.json")
	require.NoError(t, parties.ConfigureStorage(path))
	// A missing directory prevents creation before any membership changes.
	require.NoError(t, os.Mkdir(filepath.Dir(path), 0700))
	p := parties.New(94131)
	require.NotNil(t, p)
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Remove(filepath.Dir(path)))
	require.Equal(t, events.CancelAndRequeue, PurgeAlliance(events.UserPurged{UserId: 94131}))
	require.Same(t, p, parties.Get(94131))
	require.NoError(t, os.Mkdir(filepath.Dir(path), 0700))
	require.Equal(t, events.Continue, PurgeAlliance(events.UserPurged{UserId: 94131}))
	require.Nil(t, parties.Get(94131))
	require.NoError(t, parties.ConfigureStorage(path))
	require.Nil(t, parties.Get(94131))
}
