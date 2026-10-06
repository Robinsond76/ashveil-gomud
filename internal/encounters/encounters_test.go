package encounters

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupOf(m map[int]Template) Lookup {
	return func(id int) (Template, bool) { t, ok := m[id]; return t, ok }
}

var testMobs = lookupOf(map[int]Template{
	1: {}, 2: {}, 3: {}, 4: {Healer: true}, 5: {Solitary: true},
})

func pct(n int) *int { return &n }

func TestValidateDisablesBadContentWithDiagnostics(t *testing.T) {
	z := ZoneConfig{Band: Band{Low: 8, High: 10}, Tables: map[string][]Composition{
		"wood": {
			{ID: "ok", Weight: 60, Members: []Member{{1, 3}}},
			{ID: "zero-weight", Weight: 0, Members: []Member{{1, 2}}},
			{ID: "unknown", Weight: 5, Members: []Member{{99, 2}}},
			{ID: "lonely", Weight: 5, Members: []Member{{1, 1}}},
			{ID: "five", Weight: 5, Members: []Member{{1, 5}}},
			{ID: "solitary", Weight: 5, Members: []Member{{1, 1}, {5, 1}}},
			{ID: "four-healer", Weight: 5, Members: []Member{{1, 3}, {4, 1}}},
			{ID: "ok", Weight: 5, Members: []Member{{2, 2}}},
		},
		"boss": {
			{ID: "chief", Boss: true, Weight: 1, Members: []Member{{1, 1}, {2, 2}}},
			{ID: "lonely-chief", Boss: true, Weight: 1, Members: []Member{{1, 1}, {2, 1}}},
			{ID: "healer-escort", Boss: true, Weight: 1, Members: []Member{{1, 1}, {4, 2}}},
			{ID: "solitary-boss", Boss: true, Weight: 1, Members: []Member{{5, 1}, {2, 2}}},
			{ID: "solitary-escort", Boss: true, Weight: 1, Members: []Member{{1, 1}, {5, 1}, {2, 1}}},
			{ID: "two-bosses", Boss: true, Weight: 1, Members: []Member{{1, 2}, {2, 2}}},
		},
	}}
	got, diags := z.Validate(testMobs)
	require.Len(t, got.Tables["wood"], 1)
	assert.Equal(t, "ok", got.Tables["wood"][0].ID)
	require.Len(t, got.Tables["boss"], 2)
	assert.Equal(t, "chief", got.Tables["boss"][0].ID)
	assert.Equal(t, "solitary-boss", got.Tables["boss"][1].ID, "a solitary template may be the boss")
	reasons := map[string]string{}
	for _, d := range diags {
		reasons[d.Table+"/"+d.Composition] = d.Reason
	}
	assert.Contains(t, reasons["wood/zero-weight"], "weight")
	assert.Contains(t, reasons["wood/unknown"], "unknown mob template 99")
	assert.Contains(t, reasons["wood/lonely"], "2-4")
	assert.Contains(t, reasons["wood/five"], "2-4")
	assert.Contains(t, reasons["wood/solitary"], "solitary")
	assert.Contains(t, reasons["wood/four-healer"], "four-foe")
	assert.Contains(t, reasons["wood/ok"], "duplicate")
	assert.Contains(t, reasons["boss/lonely-chief"], "escorts")
	assert.Contains(t, reasons["boss/healer-escort"], "healer")
	assert.Contains(t, reasons["boss/two-bosses"], "single boss")
	assert.Contains(t, reasons["boss/solitary-escort"], "solitary")
}

func TestValidateRequiresABandForTables(t *testing.T) {
	z := ZoneConfig{Tables: map[string][]Composition{"t": {{ID: "a", Weight: 1, Members: []Member{{1, 2}}}}}}
	got, diags := z.Validate(testMobs)
	assert.Empty(t, got.Tables)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Reason, "band")
}

func TestHealerGroupsStayUncommon(t *testing.T) {
	z := ZoneConfig{Band: Band{8, 10}, Tables: map[string][]Composition{"t": {
		{ID: "wolves", Weight: 80, Members: []Member{{1, 3}}},
		{ID: "priests", Weight: 20, Members: []Member{{1, 2}, {4, 1}}}, // exactly one in five: allowed
	}, "heavy": {
		{ID: "wolves", Weight: 60, Members: []Member{{1, 3}}},
		{ID: "priests", Weight: 40, Members: []Member{{1, 2}, {4, 1}}}, // two in five: dropped
	}}}
	got, diags := z.Validate(testMobs)
	assert.Len(t, got.Tables["t"], 2)
	require.Len(t, got.Tables["heavy"], 1)
	assert.Equal(t, "wolves", got.Tables["heavy"][0].ID)
	require.Len(t, diags, 1)
	assert.Equal(t, "priests", diags[0].Composition)
}

func TestPickIsWeightedAndDeterministic(t *testing.T) {
	table := []Composition{{ID: "a", Weight: 60}, {ID: "b", Weight: 40}}
	for roll, want := range map[int]string{0: "a", 59: "a", 60: "b", 99: "b"} {
		c, ok := Pick(table, func(n int) int { assert.Equal(t, 100, n); return roll })
		require.True(t, ok)
		assert.Equal(t, want, c.ID, "roll %d", roll)
	}
	_, ok := Pick(nil, func(int) int { return 0 })
	assert.False(t, ok)
}

func TestChanceInheritanceAndZeroOverride(t *testing.T) {
	zone := ZoneConfig{}
	assert.Equal(t, 15, zone.Chance(), "the default")
	assert.Equal(t, 25, ZoneConfig{EntryChance: pct(25)}.Chance())
	assert.Equal(t, 0, ZoneConfig{EntryChance: pct(0)}.Chance(), "zero is not absence")
	assert.Equal(t, 25, RoomSetting{Enabled: true}.ChanceIn(ZoneConfig{EntryChance: pct(25)}), "inherits")
	assert.Equal(t, 0, RoomSetting{Enabled: true, Chance: pct(0)}.ChanceIn(ZoneConfig{EntryChance: pct(25)}), "zero overrides inheritance")
	assert.False(t, Roll(0, func(int) int { return 0 }))
	assert.True(t, Roll(15, func(int) int { return 14 }))
	assert.False(t, Roll(15, func(int) int { return 15 }))
}

func TestPlanLevelsFollowTheBand(t *testing.T) {
	band := Band{Low: 8, High: 10}
	high := func(n int) int { return n - 1 }
	three := Composition{Members: []Member{{1, 3}}}
	for _, f := range Plan(three, band, high) {
		assert.Equal(t, 9, f.Level, "2-3 foes top out one below the band's top")
	}
	for _, f := range Plan(three, band, func(int) int { return 0 }) {
		assert.Equal(t, 8, f.Level)
	}
	for _, f := range Plan(Composition{Members: []Member{{1, 4}}}, band, high) {
		assert.Equal(t, 8, f.Level, "four foes come from the band's low")
	}
	for _, f := range Plan(three, Band{Low: 5, High: 5}, high) {
		assert.Equal(t, 5, f.Level, "a single-level band")
	}
	boss := Plan(Composition{Boss: true, Members: []Member{{1, 1}, {2, 2}}}, band, high)
	require.Len(t, boss, 3)
	assert.True(t, boss[0].Boss)
	assert.Equal(t, 8+BossLevelBonus, boss[0].Level, "the boss is over the low by the bonus")
	for _, f := range boss[1:] {
		assert.True(t, f.Escort)
		assert.Equal(t, 8, f.Level, "escorts at the band's low")
	}
}

func TestGraceNeedsTwoEntriesAndThirtySeconds(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	g := NewGrace(now)
	assert.True(t, g.Suppresses(now.Add(time.Second)), "first entry")
	assert.True(t, g.Suppresses(now.Add(time.Minute)), "second entry, though 30s passed")
	assert.True(t, g.Suppresses(now.Add(10*time.Second)), "third entry too soon")
	assert.False(t, g.Suppresses(now.Add(31*time.Second)), "both met")
	assert.Equal(t, 0, g.Entries)
}

// TestPacingGapBetweenBattles pins the pacing the mana run reasons from: at
// the default 15% chance, a battle is followed by two skipped entries and
// then a 15% roll per entry, so about 8.7 eligible entries (a long walk)
// pass between random battles. A cleric dry by the third fight is therefore
// about 25 entries into a road: rest points go no further apart than that.
func TestPacingGapBetweenBattles(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	state := uint64(12345)
	rng := func(n int) int { // a small deterministic generator
		state = state*6364136223846793005 + 1442695040888963407
		return int((state >> 33) % uint64(n))
	}
	g := NewGrace(now)
	entries, battles := 0, 0
	for i := 0; i < 200000; i++ {
		now = now.Add(10 * time.Second) // a step every ten seconds: the 30s never binds
		entries++
		if g.Suppresses(now) {
			continue
		}
		if Roll(DefaultEntryChance, rng) {
			battles++
			g = NewGrace(now)
		}
	}
	gap := float64(entries) / float64(battles)
	assert.InDelta(t, 2+100.0/DefaultEntryChance, gap, 0.3, "entries per battle")
}

func TestRatingFollowsTheGapBetweenLevelAndBand(t *testing.T) {
	band := Band{Low: 10, High: 12}
	for level, want := range map[int]string{14: RatingEasy, 10: RatingEasy, 9: RatingFair, 8: RatingFair, 7: RatingRisky, 6: RatingRisky, 5: RatingDangerous, 1: RatingDangerous} {
		assert.Equal(t, want, Rating(level, band), "level %d", level)
	}
	assert.Empty(t, Rating(5, Band{}), "no band, no rating")
}

func TestAvailableDropsOnlyCoolingBosses(t *testing.T) {
	table := []Composition{{ID: "pack", Weight: 1}, {ID: "chief", Boss: true, Weight: 1}, {ID: "other", Boss: true, Weight: 1}}
	got := Available(table, func(id string) bool { return id == "chief" })
	require.Len(t, got, 2)
	assert.Equal(t, "pack", got[0].ID)
	assert.Equal(t, "other", got[1].ID)
	assert.Len(t, table, 3, "the zone's own table is untouched")
	assert.Len(t, Available(table, func(string) bool { return true }), 1, "ordinary groups never cool")
}
