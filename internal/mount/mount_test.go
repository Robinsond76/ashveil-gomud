package mount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSpecs() map[string]MountSpec {
	return map[string]MountSpec{
		"pack-horse":   {Type: "pack-horse", Kind: KindPack, Price: 120, BareCapacityGrams: 40000, SaddledCapacityGrams: 100000},
		"riding-horse": {Type: "riding-horse", Kind: KindRiding, Price: 150, BareCapacityGrams: 10000, SaddledCapacityGrams: 10000, TravelDurationPct: 90, FatiguePct: 75},
		"old-cob":      {Type: "old-cob", Kind: KindRiding, BareCapacityGrams: 5000, SaddledCapacityGrams: 5000, TravelDurationPct: 95},
	}
}

func TestMountSpecValidate(t *testing.T) {
	for _, spec := range testSpecs() {
		require.NoError(t, spec.Validate(), spec.Type)
	}
	for i, bad := range []MountSpec{
		{Type: "", Kind: KindPack},
		{Type: "two words", Kind: KindPack},
		{Type: "mule"},
		{Type: "mule", Kind: "cart"},
		{Type: "mule", Kind: KindPack, Price: -1},
		{Type: "mule", Kind: KindPack, BareCapacityGrams: -1},
		{Type: "mule", Kind: KindPack, SaddledCapacityGrams: -1},
		{Type: "mule", Kind: KindPack, TravelDurationPct: PctMin - 1},
		{Type: "mule", Kind: KindPack, TravelDurationPct: PctMax + 1},
		{Type: "mule", Kind: KindPack, FatiguePct: PctMin - 1},
		{Type: "mule", Kind: KindPack, FatiguePct: PctMax + 1},
	} {
		assert.ErrorIs(t, bad.Validate(), ErrInvalidSpec, "case %d", i)
	}
	unset := MountSpec{Type: "mule", Kind: KindPack}
	assert.NoError(t, unset.Validate(), "zero percentages mean unset")
	assert.Equal(t, 100, unset.EffectiveFatiguePct())
	assert.Equal(t, 100, unset.EffectiveTravelDurationPct())
	assert.Equal(t, "pack horse", testSpecs()["pack-horse"].Name())
}

func TestHorseCapacityBareAndSaddled(t *testing.T) {
	specs := testSpecs()
	herd := Herd{LeaderUserID: 7}
	herd, pack := herd.Add("pack-horse")
	herd, _ = herd.Add("riding-horse")
	assert.Equal(t, 40000+10000, herd.CapacityGrams(specs), "bare")
	herd, old, err := herd.SetSaddle(pack.ID, 34)
	require.NoError(t, err)
	assert.Zero(t, old)
	assert.Equal(t, 100000+10000, herd.CapacityGrams(specs), "a pack saddle takes a pack horse to 100 kg")
	herd, _ = herd.Add("unicorn")
	assert.Equal(t, 110000, herd.CapacityGrams(specs), "an unknown type carries nothing")
}

func TestHerdAddRemoveAndIDs(t *testing.T) {
	herd := Herd{LeaderUserID: 7}
	herd, a := herd.Add("pack-horse")
	herd, b := herd.Add("riding-horse")
	assert.Equal(t, 1, a.ID)
	assert.Equal(t, 2, b.ID)
	herd, removed, err := herd.Remove(1)
	require.NoError(t, err)
	assert.Equal(t, a, removed)
	herd, c := herd.Add("pack-horse")
	assert.Equal(t, 3, c.ID, "ids aren't reused")
	_, _, err = herd.Remove(9)
	assert.ErrorIs(t, err, ErrUnknownHorse)
	_, _, err = herd.SetSaddle(9, 34)
	assert.ErrorIs(t, err, ErrUnknownHorse)
	assert.NoError(t, herd.Validate())
	assert.ErrorIs(t, Herd{}.Validate(), ErrInvalidHerd)
	assert.ErrorIs(t, Herd{LeaderUserID: 7, Horses: []Horse{{ID: 1, Type: "a"}, {ID: 1, Type: "b"}}}.Validate(), ErrInvalidHerd)
	assert.ErrorIs(t, Herd{LeaderUserID: 7, Horses: []Horse{{ID: 0, Type: "a"}}}.Validate(), ErrInvalidHerd)
}

func TestHerdFind(t *testing.T) {
	specs := testSpecs()
	herd := Herd{LeaderUserID: 7}
	herd, _ = herd.Add("riding-horse")
	herd, _ = herd.Add("pack-horse")
	for selector, want := range map[string]int{"#2": 2, "1": 1, "pack": 2, "riding": 1, "horse": 1, "PACK HORSE": 2} {
		horse, ok := herd.Find(selector, specs)
		require.True(t, ok, selector)
		assert.Equal(t, want, horse.ID, selector)
	}
	for _, selector := range []string{"", "#9", "cart"} {
		_, ok := herd.Find(selector, specs)
		assert.False(t, ok, selector)
	}
}

// The owner's rule (2026-09-28): one riding horse and one pack horse per
// member.
func TestCanStablePerKindCap(t *testing.T) {
	specs := testSpecs()
	herd := Herd{LeaderUserID: 7}
	herd, _ = herd.Add("pack-horse")
	herd, _ = herd.Add("riding-horse")
	assert.ErrorIs(t, herd.CanStable(KindPack, specs, 1), ErrHerdFull)
	assert.ErrorIs(t, herd.CanStable(KindRiding, specs, 1), ErrHerdFull)
	assert.NoError(t, herd.CanStable(KindPack, specs, 2))
	herd, _ = herd.Add("pack-horse")
	assert.ErrorIs(t, herd.CanStable(KindPack, specs, 2), ErrHerdFull)
	assert.NoError(t, herd.CanStable(KindRiding, specs, 2))
	assert.Equal(t, 2, herd.CountKind(KindPack, specs))
	assert.ErrorIs(t, Herd{}.CanStable(KindPack, specs, 0), ErrHerdFull, "no members, no horses")
}

func TestReliefAndPace(t *testing.T) {
	specs := testSpecs()
	herd := Herd{LeaderUserID: 7}
	fatigue, riders := herd.Relief(specs)
	assert.Equal(t, 100, fatigue)
	assert.Zero(t, riders)

	herd, pack := herd.Add("pack-horse")
	herd, _, _ = herd.SetSaddle(pack.ID, 34)
	herd, ride := herd.Add("riding-horse")
	_, riders = herd.Relief(specs)
	assert.Zero(t, riders, "a pack horse carries no one; a bare riding horse neither")

	herd, _, _ = herd.SetSaddle(ride.ID, 35)
	fatigue, riders = herd.Relief(specs)
	assert.Equal(t, 75, fatigue)
	assert.Equal(t, 1, riders)
	assert.Equal(t, 90, herd.TravelDurationPct(specs, 1), "everyone rides")
	assert.Equal(t, 100, herd.TravelDurationPct(specs, 2), "someone walks: walking pace")

	herd, cob := herd.Add("old-cob")
	herd, _, _ = herd.SetSaddle(cob.ID, 35)
	assert.Equal(t, 95, herd.TravelDurationPct(specs, 2), "the slowest mount sets the pace")
	_, riders = herd.Relief(specs)
	assert.Equal(t, 2, riders)
	assert.Equal(t, 100, herd.TravelDurationPct(specs, 0))
}

func TestProvidersNeutralWithoutProvider(t *testing.T) {
	SetProvider(nil)
	pct, riders := Relief(7)
	assert.Equal(t, 100, pct)
	assert.Zero(t, riders)
	assert.Equal(t, 100, TravelDurationPct(7))
	assert.Zero(t, CapacityBonus(7))
	assert.Nil(t, HerdOf(7))
}
