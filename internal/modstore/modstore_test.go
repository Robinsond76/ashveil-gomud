package modstore

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type reg struct {
	Items map[int]string `yaml:"items"`
}

func newReg() reg { return reg{Items: map[int]string{}} }

func decodeReg(data []byte, out *reg) error {
	var wire reg
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := newReg()
	for id, v := range wire.Items {
		if id > 0 {
			loaded.Items[id] = v
		}
	}
	*out = loaded
	return nil
}

func TestLoadAbsentFileGivesEmptyRegistry(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("modstore_absent", "1.0")
	out := reg{Items: map[int]string{1: "stale"}}
	require.NoError(t, Load(plug, "reg", newReg, decodeReg, &out))
	assert.Empty(t, out.Items)
}

func TestLoadRejectsMalformedDataAndLeavesFileAlone(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("modstore_malformed", "1.0")
	bad := []byte("items: [invalid")
	require.NoError(t, plug.WriteBytes("reg", bad))
	var out reg
	require.Error(t, Load(plug, "reg", newReg, decodeReg, &out))
	data, err := plug.ReadBytes("reg")
	require.NoError(t, err)
	assert.Equal(t, bad, data)
}

func TestSaveThenLoadRoundTripsThroughDecode(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("modstore_roundtrip", "1.0")
	require.NoError(t, Save(plug, "reg", reg{Items: map[int]string{0: "dropped", 4: "kept"}}))
	var out reg
	require.NoError(t, Load(plug, "reg", newReg, decodeReg, &out))
	assert.Equal(t, map[int]string{4: "kept"}, out.Items)
}

func TestAvailable(t *testing.T) {
	assert.NoError(t, Available("camping", nil, true))
	err := Available("camping", errors.New("boom"), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "camping: persistence unavailable until a successful reload")
	assert.ErrorContains(t, err, "boom")
	assert.EqualError(t, Available("camping", nil, false), "camping: persistence unavailable")
}
