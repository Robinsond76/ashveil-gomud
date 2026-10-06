package userstate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type thing struct {
	Name  string         `yaml:"name"`
	Items []string       `yaml:"items,omitempty"`
	Tally map[string]int `yaml:"tally,omitempty"`
}

func TestMapsRoundTripOneUserOnly(t *testing.T) {
	things := map[int]thing{1: {Name: "a", Items: []string{"x"}}, 2: {Name: "other"}}
	flags := map[int]bool{1: true}
	nested := map[int]map[string]int{1: {"k": 3}}
	ms := Maps{things, flags, nested}

	data, err := ms.Capture(1)
	require.NoError(t, err)
	require.NotNil(t, data)

	things[1] = thing{Name: "changed"}
	delete(flags, 1)
	nested[1]["k"] = 99
	nested[1]["new"] = 1

	require.NoError(t, ms.Apply(1, data))
	assert.Equal(t, thing{Name: "a", Items: []string{"x"}}, things[1])
	assert.True(t, flags[1])
	assert.Equal(t, map[string]int{"k": 3}, nested[1])
	assert.Equal(t, thing{Name: "other"}, things[2], "another user is untouched")
}

func TestMapsRestoreRemovesWhatWasGained(t *testing.T) {
	things := map[int]thing{}
	flags := map[int]bool{}
	ms := Maps{things, flags}
	data, err := ms.Capture(5)
	require.NoError(t, err)
	assert.Nil(t, data, "no state captures as nil")

	things[5] = thing{Name: "gained"}
	flags[5] = true
	require.NoError(t, ms.Apply(5, nil))
	assert.Empty(t, things)
	assert.Empty(t, flags)
}

func TestMapsRejectNonUserMaps(t *testing.T) {
	_, err := Maps{map[string]int{}}.Capture(1)
	assert.Error(t, err)
}

type fake struct {
	name string
	held map[int]string
}

func (f *fake) Name() string { return f.name }
func (f *fake) Capture(id int) ([]byte, error) {
	if v, ok := f.held[id]; ok {
		return []byte(v), nil
	}
	return nil, nil
}
func (f *fake) Restore(id, _ int, data []byte) error {
	if data == nil {
		delete(f.held, id)
		return nil
	}
	f.held[id] = string(data)
	return nil
}

func TestCaptureAndRestoreAll(t *testing.T) {
	a := &fake{name: "zz-a", held: map[int]string{7: "one"}}
	b := &fake{name: "zz-b", held: map[int]string{}}
	Register(a)
	Register(b)
	defer func() { mu.Lock(); delete(contributors, "zz-a"); delete(contributors, "zz-b"); mu.Unlock() }()

	snap, err := CaptureAll(7)
	require.NoError(t, err)
	assert.Equal(t, []byte("one"), snap["zz-a"])
	assert.NotContains(t, snap, "zz-b")

	a.held[7] = "changed"
	b.held[7] = "gained"
	assert.Empty(t, RestoreAll(7, 100, snap))
	assert.Equal(t, "one", a.held[7])
	assert.NotContains(t, b.held, 7)
}
