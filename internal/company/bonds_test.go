package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func bondRecord() Record {
	return Record{LeaderUserID: 7, Companions: []Companion{
		{ID: 1, MobTemplateID: 61}, {ID: 2, MobTemplateID: 62}, {ID: 3, MobTemplateID: 63},
	}}
}

func TestABondIsOneEntryPerPairWhicheverWayItIsAsked(t *testing.T) {
	r := bondRecord()
	r.SetBond(Bond{A: 3, B: 1, Value: 20})
	r.SetBond(Bond{A: 1, B: 3, Value: 25})
	require.Len(t, r.Bonds, 1)
	bond, ok := r.BondOf(3, 1)
	require.True(t, ok)
	assert.Equal(t, 25, bond.Value)
	assert.Equal(t, 1, bond.A)
	_, ok = r.BondOf(1, 2)
	assert.False(t, ok, "no bond until one is made")
}

func TestPutPrunesBondsOfCompanionsNoLongerThereAndClampsValues(t *testing.T) {
	reg := NewRegistry()
	r := bondRecord()
	r.Bonds = []Bond{
		{A: 1, B: 2, Value: 30},
		{A: 2, B: 1, Value: 99}, // a duplicate of the pair
		{A: 1, B: 9, Value: 40}, // 9 is gone
		{A: 2, B: 2, Value: 10}, // itself
		{A: 2, B: 3, Value: 500},
	}
	reg.Put(r)
	got, _ := reg.Get(7)
	require.Len(t, got.Bonds, 2)
	assert.Equal(t, Bond{A: 1, B: 2, Value: 30}, got.Bonds[0])
	assert.Equal(t, 100, got.Bonds[1].Value)
}

func TestGetGivesACopyOfTheBonds(t *testing.T) {
	reg := NewRegistry()
	r := bondRecord()
	r.Bonds = []Bond{{A: 1, B: 2, Value: 30, At: map[string]int64{"camp": 5}}}
	reg.Put(r)
	got, _ := reg.Get(7)
	got.Bonds[0].Value = 99
	got.Bonds[0].At["camp"] = 99
	again, _ := reg.Get(7)
	assert.Equal(t, 30, again.Bonds[0].Value)
	assert.Equal(t, int64(5), again.Bonds[0].At["camp"])
}

func TestBondsRoundTripThroughYAML(t *testing.T) {
	r := bondRecord()
	r.Bonds = []Bond{{A: 1, B: 2, Value: -62, Warned: true, At: map[string]int64{"refusal": 1_800_000_000}}}
	data, err := yaml.Marshal(r)
	require.NoError(t, err)
	var back Record
	require.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, r.Bonds, back.Bonds)
	var legacy Record
	require.NoError(t, yaml.Unmarshal([]byte("leader_user_id: 7\ncompanions: []\n"), &legacy))
	assert.Empty(t, legacy.Bonds, "a company saved before bonds loads with none")
}

func TestBondsAreAbsentWithoutAProvider(t *testing.T) {
	SetFormationProvider(nil)
	assert.Equal(t, 0, BondValue(7, CompanionMemberKey(1), CompanionMemberKey(2)))
	BondEvent(7, CompanionMemberKey(1), CompanionMemberKey(2), "rescue")
	_, ok := BondsOf(7)
	assert.False(t, ok)
}
