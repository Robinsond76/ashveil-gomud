package walkto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// grid builds a graph from "a-b" links; rooms are visited unless listed in
// unvisited.
type spec struct {
	nodes map[int]*Node
}

func newSpec() *spec { return &spec{nodes: map[int]*Node{}} }

func (s *spec) room(id int, zone, title, legend string, visited bool) *spec {
	s.nodes[id] = &Node{ID: id, Zone: zone, Title: title, Legend: legend, Visited: visited}
	return s
}

func (s *spec) link(a, b int, name, back string, edit ...func(*Exit)) *spec {
	ea, eb := Exit{Name: name, To: b}, Exit{Name: back, To: a}
	for _, f := range edit {
		f(&ea)
	}
	s.nodes[a].Exits = append(s.nodes[a].Exits, ea)
	s.nodes[b].Exits = append(s.nodes[b].Exits, eb)
	return s
}

func (s *spec) graph() Graph {
	return func(id int) (Node, bool) {
		n, ok := s.nodes[id]
		if !ok {
			return Node{}, false
		}
		return *n, true
	}
}

// 1 - 2 - 3 - 4(inn)   5 hangs off 3 by a locked door; 6 is unvisited off 2.
func line() *spec {
	s := newSpec().
		room(1, "z", "Start", "", true).
		room(2, "z", "Green", "Village", true).
		room(3, "z", "Fork", "", true).
		room(4, "z", "The Alder Inn", "Inn", true).
		room(5, "z", "Cellar", "", true).
		room(6, "z", "Unknown Hollow", "", false)
	s.link(1, 2, "east", "west").link(2, 3, "east", "west").link(3, 4, "east", "west")
	s.link(3, 5, "down", "up", func(e *Exit) { e.Locked = true })
	s.link(2, 6, "north", "south")
	return s
}

func TestPlanWalksTheShortestKnownRoute(t *testing.T) {
	r, err := Plan(line().graph(), 1, 4)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 4}, r.Path)
	assert.Equal(t, []Step{{"east", 2}, {"east", 3}, {"east", 4}}, r.Steps)
	assert.Equal(t, 4, r.Target)
}

func TestPlanRefusals(t *testing.T) {
	g := line().graph()
	_, err := Plan(g, 1, 1)
	assert.ErrorIs(t, err, ErrHere)
	_, err = Plan(g, 1, 6)
	assert.ErrorIs(t, err, ErrUnvisited, "an unvisited room is refused")
	_, err = Plan(g, 1, 99)
	assert.ErrorIs(t, err, ErrUnvisited, "a room that does not exist reads the same")
	_, err = Plan(g, 1, 5)
	assert.ErrorIs(t, err, ErrLocked, "a locked door without a key")
}

func TestPlanThroughALockWithAKey(t *testing.T) {
	s := line()
	for i, e := range s.nodes[3].Exits {
		if e.To == 5 {
			s.nodes[3].Exits[i].HasKey = true
		}
	}
	r, err := Plan(s.graph(), 1, 5)
	require.NoError(t, err)
	assert.Equal(t, "down", r.Steps[len(r.Steps)-1].Exit)
}

func TestPlanNeverUsesUnknownSecretJourneyOrDelayedExits(t *testing.T) {
	for name, block := range map[string]Block{"secret": BlockSecret, "journey": BlockJourney, "message": BlockMessage} {
		s := newSpec().room(1, "z", "A", "", true).room(2, "z", "B", "", true)
		s.link(1, 2, "east", "west", func(e *Exit) { e.Block = block })
		// the way back is plain, but only 2 -> 1 exists as usable
		_, err := Plan(s.graph(), 1, 2)
		assert.ErrorIs(t, err, ErrNoPath, name)
		r, err := Plan(s.graph(), 2, 1)
		require.NoError(t, err, name)
		assert.Equal(t, []int{2, 1}, r.Path)
	}
}

func TestPlanThroughAnUnvisitedRoomIsRefused(t *testing.T) {
	s := newSpec().room(1, "z", "A", "", true).room(2, "z", "Between", "", false).room(3, "z", "C", "", true)
	s.link(1, 2, "east", "west").link(2, 3, "east", "west")
	_, err := Plan(s.graph(), 1, 3)
	assert.ErrorIs(t, err, ErrNoPath, "only rooms the player has visited are used")
}

func TestPlanCapsAtSixtySteps(t *testing.T) {
	s := newSpec()
	for i := 1; i <= 62; i++ {
		s.room(i, "z", "Road", "", true)
	}
	for i := 1; i < 62; i++ {
		s.link(i, i+1, "east", "west")
	}
	r, err := Plan(s.graph(), 1, 61)
	require.NoError(t, err, "exactly 60 steps is allowed")
	assert.Len(t, r.Steps, MaxSteps)
	_, err = Plan(s.graph(), 1, 62)
	var far TooFarError
	require.ErrorAs(t, err, &far)
	assert.Equal(t, 61, far.Steps)
	assert.Contains(t, far.Error(), "at most 60")
}

func TestPlanPrefersTheShorterOfTwoRoutes(t *testing.T) {
	s := newSpec()
	for i := 1; i <= 5; i++ {
		s.room(i, "z", "R", "", true)
	}
	s.link(1, 2, "north", "south").link(2, 3, "east", "west").link(3, 5, "south", "north") // 3 steps
	s.link(1, 4, "east", "west").link(4, 5, "east", "west")                                // 2 steps
	r, err := Plan(s.graph(), 1, 5)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 4, 5}, r.Path)
}

func TestResolveByLegendTitleAndNearest(t *testing.T) {
	s := line()
	s.room(7, "other", "Far Inn", "Inn", true)
	s.link(4, 7, "east", "west")
	g := s.graph()
	id, err := Resolve(g, 1, "inn")
	require.NoError(t, err)
	assert.Equal(t, 4, id, "the legend matches, in this zone, nearest first")
	id, err = Resolve(g, 1, "VILLAGE")
	require.NoError(t, err)
	assert.Equal(t, 2, id)
	id, err = Resolve(g, 1, "the alder")
	require.NoError(t, err)
	assert.Equal(t, 4, id, "a title prefix")
	id, err = Resolve(g, 1, "fork")
	require.NoError(t, err)
	assert.Equal(t, 3, id, "a title word")
	id, err = Resolve(g, 1, "cellar")
	require.NoError(t, err)
	assert.Equal(t, 5, id, "a place behind a lock still resolves; Plan explains the lock")
	_, err = Resolve(g, 1, "hollow")
	assert.ErrorIs(t, err, ErrUnknownPlace, "unvisited rooms never resolve")
	_, err = Resolve(g, 1, "")
	assert.ErrorIs(t, err, ErrUnknownPlace)
}

func TestStateLifecycleAndView(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)
	var changed []int
	OnChanged.Register(func(id int) int { changed = append(changed, id); return id })

	_, ok := ViewOf(7)
	assert.False(t, ok)
	r, err := Plan(line().graph(), 1, 4)
	require.NoError(t, err)
	a := Begin(7, r, [3]bool{})
	v, ok := ViewOf(7)
	require.True(t, ok)
	assert.Equal(t, View{Target: 4, Path: []int{2, 3, 4}}, v)

	assert.False(t, Advance(7, a.Gen+1, 1), "a stale generation is ignored")
	assert.True(t, Advance(7, a.Gen, 1))
	v, _ = ViewOf(7)
	assert.Equal(t, []int{3, 4}, v.Path)

	a2 := Begin(7, r, [3]bool{})
	assert.NotEqual(t, a.Gen, a2.Gen, "a new start replaces the walk")
	assert.True(t, Clear(7))
	assert.False(t, Clear(7))
	_, ok = Current(7)
	assert.False(t, ok)
	assert.GreaterOrEqual(t, len(changed), 4)
}
