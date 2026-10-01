package parties

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestAlliancePersistenceRecoveryAndRollback(t *testing.T) {
	t.Cleanup(UseMemoryForTest())
	path := filepath.Join(t.TempDir(), "alliances.json")
	require.NoError(t, ConfigureStorage(path))
	p := New(94101)
	require.NotNil(t, p)
	p.InvitePlayer(94102)
	require.True(t, p.AcceptInvite(94102))
	p.InvitePlayer(94103)
	p.SetFollow(94102, true)
	p.SetSupport(94101, true)
	p.SetSupport(94102, true)
	p.SetAutoAttack(94102, true)
	require.NoError(t, ConfigureStorage(path))
	recovered := Get(94102)
	require.NotNil(t, recovered)
	assert.Equal(t, 94101, recovered.LeaderUserId)
	assert.Nil(t, Get(94103), "invitations are not durable membership")
	assert.Empty(t, recovered.Followers)
	assert.Empty(t, recovered.Supporters)
	assert.Empty(t, recovered.AutoAttackers)
	assert.Empty(t, AlliedLeaders(94101))
	require.True(t, recovered.Promote(94102))
	require.NoError(t, ConfigureStorage(path))
	recovered = Get(94101)
	assert.Equal(t, 94102, recovered.LeaderUserId)
	old, err := os.ReadFile(path)
	require.NoError(t, err)
	storagePath = filepath.Join(t.TempDir(), "missing", "alliances.json")
	recovered.InvitePlayer(94105)
	assert.False(t, recovered.AcceptInvite(94105))
	assert.False(t, recovered.IsMember(94105))
	assert.Contains(t, recovered.GetInvited(), 94105)
	assert.False(t, recovered.Leave(94101))
	require.Error(t, LastError())
	assert.Same(t, recovered, Get(94101), "rollback preserves authoritative pointer identity")
	assert.True(t, recovered.IsMember(94101))
	assert.False(t, recovered.Promote(94101))
	assert.Equal(t, 94102, recovered.LeaderUserId)
	assert.False(t, recovered.TryDisband())
	assert.Same(t, recovered, Get(94102))
	assert.Nil(t, New(94104))
	assert.Nil(t, Get(94104))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, old, data)
	storagePath = path
	require.True(t, recovered.Leave(94101))
	require.NoError(t, ConfigureStorage(path))
	assert.Nil(t, Get(94101))
	assert.NotNil(t, Get(94102))
}

func TestAllianceRejectsCorruptRecovery(t *testing.T) {
	for _, data := range []string{`{"version":2}`, `{"version":1,"parties":[{"leader":1,"members":[2]}]}`, `{"version":1,"parties":[{"leader":1,"members":[1,1]}]}`, `{"version":1,"parties":[{"leader":1,"members":[1,2]},{"leader":2,"members":[2]}]}`, `{`} {
		t.Run(data, func(t *testing.T) {
			t.Cleanup(UseMemoryForTest())
			p := New(1)
			path := filepath.Join(t.TempDir(), "alliances.json")
			require.NoError(t, os.WriteFile(path, []byte(data), 0600))
			require.Error(t, ConfigureStorage(path))
			assert.Same(t, p, Get(1), "bad recovery never partially applies")
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, data, string(got))
		})
	}
}

func TestSoloLeaveSaveFailurePreservesConsent(t *testing.T) {
	t.Cleanup(UseMemoryForTest())
	p := New(94120)
	p.SetAutoAttack(94120, true)
	p.SetSupport(94120, true)
	p.SetFollow(94120, true)
	auto, follow := p.AutoAttackToken(94120), p.followTokens[94120]
	storagePath = filepath.Join(t.TempDir(), "missing", "alliances.json")
	require.False(t, p.Leave(94120))
	assert.Same(t, p, Get(94120))
	assert.True(t, p.Supports(94120))
	assert.Equal(t, auto, p.AutoAttackToken(94120))
	assert.Equal(t, follow, p.followTokens[94120])
}
