package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func sampleState() MemberState {
	s := MemberState{Level: 3, Experience: 1200}
	s.Equipment.Weapon = items.Item{ItemId: 10002, Adjectives: []string{"sharp"}}
	s.Equipment.Head = items.Item{ItemId: 20020}
	s.Items = []items.Item{{ItemId: 30004}, {ItemId: 30015, Uses: 2}}
	return s
}

func stateRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	_, err := r.Summon(7, 58, map[int]struct{}{58: {}}, 4)
	require.NoError(t, err)
	require.NoError(t, r.SetState(7, 1, sampleState()))
	return r
}

func TestMemberStateCloneIsDeep(t *testing.T) {
	s := sampleState()
	c := s.Clone()
	c.Items[0].ItemId = 1
	c.Equipment.Weapon.Adjectives[0] = "dull"
	c.Equipment.Head.ItemId = 2
	assert.Equal(t, sampleState(), s)
}

func TestRegistryGetDeepCopiesState(t *testing.T) {
	r := stateRegistry(t)
	record, ok := r.Get(7)
	require.True(t, ok)
	require.NotNil(t, record.Companions[0].State)
	record.Companions[0].State.Items[0].ItemId = 1
	record.Companions[0].State.Level = 99
	again, _ := r.Get(7)
	assert.Equal(t, sampleState(), *again.Companions[0].State, "the registry is unchanged")
}

func TestRegistryCloneDeepCopiesState(t *testing.T) {
	r := stateRegistry(t)
	clone := r.Clone()
	clone.Companies[7].Companions[0].State.Items[0].ItemId = 1
	again, _ := r.Get(7)
	assert.Equal(t, 30004, again.Companions[0].State.Items[0].ItemId)
}

func TestSetState(t *testing.T) {
	r := stateRegistry(t)
	assert.ErrorIs(t, r.SetState(8, 1, MemberState{}), ErrUnknownMember)
	assert.ErrorIs(t, r.SetState(7, 2, MemberState{}), ErrUnknownMember)

	s := sampleState()
	require.NoError(t, r.SetState(7, 1, s))
	s.Items[0].ItemId = 1
	got, _ := r.Get(7)
	assert.Equal(t, 30004, got.Companions[0].State.Items[0].ItemId, "SetState stores a copy")

	s = sampleState()
	s.ClearGear()
	require.NoError(t, r.SetState(7, 1, s))
	got, _ = r.Get(7)
	assert.Equal(t, 3, got.Companions[0].State.Level)
	assert.Zero(t, got.Companions[0].State.Equipment.Weapon.ItemId)
	assert.Empty(t, got.Companions[0].State.Items)
}

func TestStateYAMLRoundTrip(t *testing.T) {
	r := stateRegistry(t)
	data, err := yaml.Marshal(r.Clone())
	require.NoError(t, err)
	var loaded Registry
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	got, ok := loaded.Get(7)
	require.True(t, ok)
	require.NotNil(t, got.Companions[0].State)
	assert.Equal(t, sampleState(), *got.Companions[0].State)

	// A companion without state stays nil (legacy until initialized).
	legacy := NewRegistry()
	_, err = legacy.Summon(9, 58, map[int]struct{}{58: {}}, 4)
	require.NoError(t, err)
	data, err = yaml.Marshal(legacy.Clone())
	require.NoError(t, err)
	assert.NotContains(t, string(data), "state")
}

// Review finding 11: an item's spec override is copied, not shared with
// the live mob.
func TestMemberStateCloneCopiesSpecOverride(t *testing.T) {
	s := MemberState{Items: []items.Item{{ItemId: 10002, Spec: &items.ItemSpec{Name: "named blade"}}}, Gold: 7}
	c := s.Clone()
	c.Items[0].Spec.Name = "renamed"
	assert.Equal(t, "named blade", s.Items[0].Spec.Name)
	assert.Equal(t, 7, c.Gold)
	c.ClearGear()
	assert.Zero(t, c.Gold)
}
