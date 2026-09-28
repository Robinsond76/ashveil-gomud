package company

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRosterRules() RosterRules {
	return RosterRules{
		Size:    3,
		StayMin: 450, StayMax: 1800,
		RefillMin: 60, RefillMax: 180,
		PriceBase: 30, PricePerLevel: 30,
		AlignmentMin: -80, AlignmentMax: 80,
		BynamePercent: 50,
		Archetypes: []RosterArchetype{
			{Archetype: "warrior", MobTemplateID: 80, Weight: 1},
			{Archetype: "rogue", MobTemplateID: 81, Weight: 1, PricePercent: 90},
			{Archetype: "wizard", MobTemplateID: 82, Weight: 1},
			{Archetype: "cleric", MobTemplateID: 83, Weight: 1, PricePercent: 110},
			{Archetype: "ranger", MobTemplateID: 84, Weight: 1},
		},
		GivenNames: []string{"Hild", "Anselm", "Tobin", "Wren", "Edda", "Bram", "Ilse", "Corrin"},
		Bynames:    []string{"Marrow", "of the Ford"},
		Traits:     []string{"A scarred former caravan guard.", "Quiet, and quick with a sling."},
	}
}

func TestRefreshRosterGeneratesAFullStaggeredRoster(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(1))
	ctx := RosterContext{Now: 10000, LeaderLevel: 3, Taken: map[string]bool{"garrick": true, "hild": true}}
	ros, changed := RefreshRoster(Roster{RoomID: 2003}, rules, ctx, rng)
	require.True(t, changed)
	require.Len(t, ros.Candidates, 3)
	keys := map[string]bool{}
	leaves := map[uint64]bool{}
	for _, c := range ros.Candidates {
		assert.False(t, keys[c.Key], "given names are unique")
		keys[c.Key] = true
		assert.NotEqual(t, "hild", c.Key, "a taken name is never reused")
		assert.Equal(t, strings.ToLower(strings.Fields(c.Name)[0]), c.Key)
		assert.GreaterOrEqual(t, c.Level, 2)
		assert.LessOrEqual(t, c.Level, 4)
		assert.GreaterOrEqual(t, c.Alignment, -80)
		assert.LessOrEqual(t, c.Alignment, 80)
		assert.Greater(t, c.Leaves, ctx.Now, "no one is seen already gone")
		assert.GreaterOrEqual(t, c.Leaves, ctx.Now+rules.StayMin/2, "no one leaves the moment they're seen")
		assert.LessOrEqual(t, c.Leaves-c.Arrived, rules.StayMax)
		assert.NotEmpty(t, c.Trait)
		leaves[c.Leaves] = true
	}
	assert.Len(t, leaves, 3, "stays are staggered")
}

func TestRefreshRosterLevelFloorAndPrice(t *testing.T) {
	rules := testRosterRules()
	for seed := int64(0); seed < 50; seed++ {
		ros, _ := RefreshRoster(Roster{}, rules, RosterContext{Now: 5000, LeaderLevel: 1}, rand.New(rand.NewSource(seed)))
		for _, c := range ros.Candidates {
			assert.GreaterOrEqual(t, c.Level, 1)
			assert.LessOrEqual(t, c.Level, 2)
			var a RosterArchetype
			for _, x := range rules.Archetypes {
				if x.Archetype == c.Archetype {
					a = x
				}
			}
			assert.Equal(t, a.MobTemplateID, c.MobTemplateID)
			assert.Equal(t, CandidatePrice(rules, a, c.Level), c.Price)
		}
	}
	assert.Equal(t, 90, CandidatePrice(rules, rules.Archetypes[0], 2))
	assert.Equal(t, 108, CandidatePrice(rules, rules.Archetypes[1], 3))
	assert.Equal(t, 132, CandidatePrice(rules, rules.Archetypes[3], 3))
}

func TestRefreshRosterArchetypeWeights(t *testing.T) {
	rules := testRosterRules()
	ctx := RosterContext{Now: 5000, LeaderLevel: 2, Weights: map[string]int{"ranger": 1}}
	for seed := int64(0); seed < 20; seed++ {
		ros, _ := RefreshRoster(Roster{}, rules, ctx, rand.New(rand.NewSource(seed)))
		for _, c := range ros.Candidates {
			assert.Equal(t, "ranger", c.Archetype)
			assert.Equal(t, 84, c.MobTemplateID)
		}
	}
	// A heavier weight shows up more often.
	ctx.Weights = map[string]int{"ranger": 9, "warrior": 1}
	counts := map[string]int{}
	for seed := int64(0); seed < 200; seed++ {
		ros, _ := RefreshRoster(Roster{}, rules, ctx, rand.New(rand.NewSource(seed)))
		for _, c := range ros.Candidates {
			counts[c.Archetype]++
		}
	}
	assert.Greater(t, counts["ranger"], 4*counts["warrior"])
	assert.Zero(t, counts["cleric"])
}

func TestRefreshRosterNothingDue(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(3))
	ros, _ := RefreshRoster(Roster{RoomID: 1}, rules, RosterContext{Now: 1000, LeaderLevel: 1}, rng)
	again, changed := RefreshRoster(ros, rules, RosterContext{Now: 1001, LeaderLevel: 1}, rng)
	assert.False(t, changed)
	assert.Equal(t, ros, again)
}

func TestRefreshRosterReplacesALapsedCandidate(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(4))
	ros, _ := RefreshRoster(Roster{RoomID: 1}, rules, RosterContext{Now: 1000, LeaderLevel: 1}, rng)
	first := ros.Candidates[0]
	for _, c := range ros.Candidates[1:] {
		if c.Leaves < first.Leaves {
			first = c
		}
	}
	// Just after the first to leave has gone, and before anyone else.
	now := first.Leaves
	next, changed := RefreshRoster(ros, rules, RosterContext{Now: now, LeaderLevel: 1}, rng)
	require.True(t, changed)
	require.Len(t, next.Candidates, 3)
	gone := true
	newcomers := 0
	for _, c := range next.Candidates {
		if c.Key == first.Key && c.Leaves == first.Leaves {
			gone = false
		}
		if c.Arrived == first.Leaves {
			newcomers++
			assert.Greater(t, c.Leaves, now)
		}
	}
	assert.True(t, gone, "the lapsed candidate left")
	assert.Equal(t, 1, newcomers, "one newcomer took the slot when it emptied")
}

func TestRefreshRosterAfterALongAbsence(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(5))
	ros, _ := RefreshRoster(Roster{RoomID: 1}, rules, RosterContext{Now: 1000, LeaderLevel: 1}, rng)
	now := uint64(1000 + 7*900*6) // six game days a day, for a week
	next, changed := RefreshRoster(ros, rules, RosterContext{Now: now, LeaderLevel: 1}, rng)
	require.True(t, changed)
	require.Len(t, next.Candidates, 3)
	for _, c := range next.Candidates {
		assert.Greater(t, c.Leaves, now, "everyone is here now, not a chain of faces in between")
		assert.GreaterOrEqual(t, c.Leaves, now+rules.StayMin/2)
		for _, old := range ros.Candidates {
			assert.False(t, old.Leaves == c.Leaves && old.Key == c.Key, "every old face is gone")
		}
	}
}

func TestHireFromRosterRefillsAfterADelay(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(6))
	ros, _ := RefreshRoster(Roster{RoomID: 1}, rules, RosterContext{Now: 1000, LeaderLevel: 1}, rng)
	hired := ros.Candidates[1]
	after, ok := HireFromRoster(ros, hired.Key, rules, 1000, rng)
	require.True(t, ok)
	require.Len(t, after.Candidates, 2)
	require.Len(t, after.Openings, 1)
	opening := after.Openings[0]
	assert.GreaterOrEqual(t, opening, uint64(1000)+rules.RefillMin)
	assert.LessOrEqual(t, opening, uint64(1000)+rules.RefillMax)

	_, ok = HireFromRoster(after, hired.Key, rules, 1000, rng)
	assert.False(t, ok, "a hired candidate can't be hired twice")

	// Before the opening: the list stays short.
	same, changed := RefreshRoster(after, rules, RosterContext{Now: opening - 1, LeaderLevel: 1}, rng)
	assert.False(t, changed)
	assert.Len(t, same.Candidates, 2)

	// At the opening: a new face arrives then.
	filled, changed := RefreshRoster(after, rules, RosterContext{Now: opening, LeaderLevel: 1}, rng)
	assert.True(t, changed)
	assert.Len(t, filled.Candidates, 3)
	assert.Empty(t, filled.Openings)
	assert.Equal(t, opening, filled.Candidates[2].Arrived)
}

func TestRefreshRosterRunsOutOfNames(t *testing.T) {
	rules := testRosterRules()
	rules.GivenNames = []string{"Hild", "Wren"}
	ros, _ := RefreshRoster(Roster{}, rules, RosterContext{Now: 1000, LeaderLevel: 1, Taken: map[string]bool{"wren": true}}, rand.New(rand.NewSource(7)))
	require.Len(t, ros.Candidates, 1)
	assert.Equal(t, "hild", ros.Candidates[0].Key)
}

func TestRosterFind(t *testing.T) {
	ros := Roster{Candidates: []Candidate{
		{Key: "hild", Name: "Hild Marrow"},
		{Key: "anselm", Name: "Anselm of the Ford"},
		{Key: "tobin", Name: "Tobin Reyes"},
	}}
	for sel, want := range map[string]string{"hild": "hild", "Anselm of the Ford": "anselm", "reyes": "tobin", "ford": "anselm"} {
		c, ok := ros.Find(sel)
		require.True(t, ok, sel)
		assert.Equal(t, want, c.Key, sel)
	}
	_, ok := ros.Find("o") // several
	assert.False(t, ok)
	_, ok = ros.Find("")
	assert.False(t, ok)
}

func TestRecordRostersArePersistedAndCopied(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.PutRoster(7, Roster{RoomID: 2003, Candidates: []Candidate{{Key: "hild", Name: "Hild"}}}))
	record, ok := reg.Get(7)
	require.True(t, ok, "a roster alone keeps the record")
	ros, ok := record.Roster(2003)
	require.True(t, ok)
	ros.Candidates[0].Name = "changed"
	record.Rosters[0].Candidates[0].Name = "changed too"
	again, _ := reg.Get(7)
	got, _ := again.Roster(2003)
	assert.Equal(t, "Hild", got.Candidates[0].Name, "Get and Roster copy")
	_, ok = again.Roster(2005)
	assert.False(t, ok)
}

func TestSetIdentity(t *testing.T) {
	reg := NewRegistry()
	c, err := reg.Summon(7, 80, map[int]struct{}{80: {}}, 4)
	require.NoError(t, err)
	require.NoError(t, reg.SetIdentity(7, c.ID, Identity{Name: " Hild Marrow ", Description: "Quiet."}))
	record, _ := reg.Get(7)
	assert.Equal(t, "Hild Marrow", record.Companions[0].Name)
	assert.Equal(t, Identity{Name: "Hild Marrow", Description: "Quiet."}, record.Companions[0].Identity())
	assert.ErrorIs(t, reg.SetIdentity(7, 99, Identity{Name: "x"}), ErrUnknownMember)
}

// Phase 32a2 review: a smaller RosterSize takes effect at once.
func TestRefreshRosterShrinks(t *testing.T) {
	rules := testRosterRules()
	rng := rand.New(rand.NewSource(8))
	ros, _ := RefreshRoster(Roster{RoomID: 1}, rules, RosterContext{Now: 1000, LeaderLevel: 1}, rng)
	ros, _ = HireFromRoster(ros, ros.Candidates[2].Key, rules, 1000, rng)
	rules.Size = 1
	small, changed := RefreshRoster(ros, rules, RosterContext{Now: 1001, LeaderLevel: 1}, rng)
	assert.True(t, changed)
	assert.Equal(t, ros.Candidates[:1], small.Candidates)
	assert.Empty(t, small.Openings)
	rules.Size = 0
	none, _ := RefreshRoster(ros, rules, RosterContext{Now: 1001, LeaderLevel: 1}, rng)
	assert.Empty(t, none.Candidates)
}
