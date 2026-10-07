package bounty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func targets() []Target {
	return []Target{
		{Kind: Boss, Zone: "Dark Forest", Ref: "mob:200", Name: "Old Grue", Low: 5, High: 7},
		{Kind: Boss, Zone: "Catacombs", Ref: "mob:300", Name: "Lich", Low: 10, High: 12},
		{Kind: Boss, Zone: "Alderbrook", Ref: "mob:12", Name: "Big Rat", Low: 2, High: 4},
		{Kind: Boss, Zone: "Marrowmere", Ref: "mob:210", Name: "Fen King", Low: 6, High: 8},
		{Kind: Group, Zone: "Alderbrook", Ref: "group:field-rats", Name: "Field Rats", Low: 2, High: 4},
		{Kind: Group, Zone: "Alderbrook", Ref: "group:stray-dogs", Name: "Stray Dogs", Low: 2, High: 4},
		{Kind: Group, Zone: "Brindle Downs", Ref: "group:wolves", Name: "Wolves", Low: 4, High: 6},
		{Kind: Group, Zone: "Dark Forest", Ref: "group:spiders", Name: "Spiders", Low: 5, High: 7},
		{Kind: Group, Zone: "Catacombs", Ref: "group:ghouls", Name: "Ghouls", Low: 10, High: 12},
	}
}

func TestNearbyBands(t *testing.T) {
	board := Band{2, 4}
	assert.True(t, Nearby(board, Band{2, 4}))
	assert.True(t, Nearby(board, Band{5, 7}), "a zone just above is near")
	assert.False(t, Nearby(board, Band{8, 10}), "four levels past the board's top is far")
	assert.True(t, Nearby(Band{6, 8}, Band{4, 5}), "a little below is near")
	assert.False(t, Nearby(Band{10, 12}, Band{2, 4}), "a gentle zone far below is not posted")
	assert.False(t, Nearby(board, Band{}), "a zone with no band is never posted")
	// A board in a zone with no band posts only the gentle zones.
	assert.True(t, Nearby(Band{}, Band{5, 7}))
	assert.False(t, Nearby(Band{}, Band{7, 9}))
}

func TestRewardWithinTheBand(t *testing.T) {
	boss := Target{Kind: Boss, Low: 7, High: 9}
	group := Target{Kind: Group, Low: 7, High: 9, Count: GroupCount}
	assert.Equal(t, 20*8, Reward(boss))
	assert.Equal(t, 6*8*3, Reward(group))
	assert.Equal(t, 20, Reward(Target{Kind: Boss, Low: 1, High: 1}))
	assert.Less(t, Reward(Target{Kind: Boss, Low: 2, High: 4}), Reward(Target{Kind: Boss, Low: 10, High: 12}), "a higher band pays more")
}

func TestPostIsStableAndNearbyOnly(t *testing.T) {
	board := Band{2, 4}
	a := Post("Alderbrook:2114", board, 1_800_000_000, targets())
	b := Post("Alderbrook:2114", board, 1_800_000_000, targets())
	assert.Equal(t, a, b, "the same board and window give the same list")
	require.NotEmpty(t, a)
	assert.LessOrEqual(t, len(a), PostingsPerBoard)
	bosses := 0
	for _, p := range a {
		assert.True(t, Nearby(board, p.Target.Band()), "%s is far from the board", p.Target.Name)
		assert.Equal(t, p.Reward, Reward(p.Target))
		assert.Equal(t, int64(1_800_000_000+WindowSeconds), p.Until)
		if p.Target.Kind == Boss {
			bosses++
			assert.Equal(t, 1, p.Target.Count)
		} else {
			assert.Equal(t, GroupCount, p.Target.Count)
		}
	}
	assert.LessOrEqual(t, bosses, MaxBosses)
	assert.Equal(t, Boss, a[0].Target.Kind, "lair bosses lead the list")
}

func TestPostRotatesAndBoardsDiffer(t *testing.T) {
	board := Band{5, 7}
	seen := map[string]bool{}
	for w := int64(0); w < 40; w++ {
		for _, p := range Post("B", board, w*WindowSeconds, targets()) {
			seen[p.ID[len(p.ID)-8:]] = true
		}
	}
	assert.Greater(t, len(seen), 5, "over many windows more than one fixed set is posted")
	same := 0
	for w := int64(0); w < 20; w++ {
		a := Post("A", board, w*WindowSeconds, targets())
		b := Post("B", board, w*WindowSeconds, targets())
		if len(a) == len(b) && fmtIDs(a) == fmtIDs(b) {
			same++
		}
	}
	assert.Less(t, same, 20, "two boards do not always post one list")
	assert.Equal(t, int64(100*WindowSeconds), Window(100*WindowSeconds+5))
}

func fmtIDs(ps []Posting) string {
	s := ""
	for _, p := range ps {
		s += p.ID + ","
	}
	return s
}

func TestPostWithNoCandidatesIsEmpty(t *testing.T) {
	assert.Empty(t, Post("B", Band{2, 4}, 0, nil))
	assert.Empty(t, Post("B", Band{2, 4}, 0, []Target{{Kind: Boss, Zone: "Far", Ref: "mob:1", Low: 30, High: 33}}))
}

func TestOnlyBossesFillABoardWithoutGroups(t *testing.T) {
	var only []Target
	for _, c := range targets() {
		if c.Kind == Boss {
			only = append(only, c)
		}
	}
	assert.GreaterOrEqual(t, len(Post("B", Band{5, 8}, 0, only)), 3, "bosses beyond the cap post when no groups exist")
}

func TestTakeHoldsSettleAndLapse(t *testing.T) {
	posts := Post("B", Band{2, 4}, 0, targets())
	var s State
	require.NoError(t, s.Take(posts[0], 100, 7))
	assert.ErrorIs(t, s.Take(posts[0], 101, 7), ErrHeld)
	require.Len(t, s.Held, 1)
	assert.Equal(t, int64(100+TermSeconds), s.Held[0].Due)
	assert.Equal(t, 7, s.Held[0].AfterSeq)

	h := s.Settle(0, 200)
	assert.Equal(t, posts[0].ID, h.PostID)
	assert.Empty(t, s.Held)
	assert.ErrorIs(t, s.Take(posts[0], 300, 9), ErrDone, "a settled posting is not taken twice")

	var full State
	for i := 0; i < MaxHeld; i++ {
		require.NoError(t, full.Take(Posting{ID: string(rune('a' + i)), Target: Target{Zone: "Z", Ref: string(rune('a' + i))}}, 0, 0))
	}
	assert.ErrorIs(t, full.Take(Posting{ID: "z", Target: Target{Zone: "Z", Ref: "z"}}, 0, 0), ErrFull)

	assert.Equal(t, 0, full.Expired(TermSeconds-1))
	assert.Equal(t, MaxHeld, full.Expired(TermSeconds))
	assert.Empty(t, full.Held)

	s.Expired(200 + DoneKeepWindow*TermSeconds)
	assert.Empty(t, s.Done, "settled postings are forgotten after a while")
}

func TestCloneSharesNothing(t *testing.T) {
	s := State{Held: []Held{{PostID: "a"}}, Done: map[string]int64{"x": 1}}
	c := s.Clone()
	c.Held[0].PostID = "b"
	c.Done["x"] = 2
	assert.Equal(t, "a", s.Held[0].PostID)
	assert.Equal(t, int64(1), s.Done["x"])
}
