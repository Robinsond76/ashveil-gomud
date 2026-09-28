package battle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginCurrentGrowEnd(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	_, ok := Current(7)
	assert.False(t, ok)

	b := Begin(7, 100, 5, "spawn:100:1", []int{21, 22})
	assert.Equal(t, 7, b.UserId)
	assert.True(t, b.Has(21))
	assert.False(t, b.Has(30))

	Grow(7, "spawn:100:1", []int{23})
	got, ok := Current(7)
	require.True(t, ok)
	assert.True(t, got.Has(23))
	assert.Equal(t, uint64(5), got.StartRound)

	// Current returns a copy.
	got.Enemies[99] = true
	again, _ := Current(7)
	assert.False(t, again.Has(99))

	assert.Equal(t, []int{7}, Players())
	ended, ok := End(7)
	require.True(t, ok)
	assert.Equal(t, 100, ended.RoomId)
	_, ok = Current(7)
	assert.False(t, ok)
	_, ok = End(7)
	assert.False(t, ok)
}

func TestFirstSetOrder(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.Equal(t, uint64(3), NoteSet(7, "rats", 3))
	assert.Equal(t, uint64(3), NoteSet(7, "rats", 9), "first time only")
	assert.Equal(t, uint64(4), NoteSet(7, "ruffians", 4))

	// The battle's group is forgotten when the battle ends.
	Begin(7, 100, 5, "rats", nil)
	End(7)
	assert.Equal(t, uint64(8), NoteSet(7, "rats", 8), "set again, it waits from now")

	// A group no longer set on the player drops out of line.
	KeepSet(7, []string{"rats"})
	assert.Equal(t, uint64(10), NoteSet(7, "ruffians", 10))

	Forget(7)
	assert.Equal(t, uint64(11), NoteSet(7, "rats", 11))
}

func TestNext(t *testing.T) {
	_, ok := Next(nil)
	assert.False(t, ok)
	i, ok := Next([]Candidate{
		{PartyID: "a", FirstSet: 5, Order: 0},
		{PartyID: "b", FirstSet: 3, Order: 2},
		{PartyID: "c", FirstSet: 3, Order: 1},
	})
	require.True(t, ok)
	assert.Equal(t, 2, i, "set first, ties by room order")
}

func TestEngaged(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.False(t, Engaged(5))
	Begin(1, 10, 1, "p", []int{5, 6})
	assert.True(t, Engaged(5))
	assert.False(t, Engaged(7))
	End(1)
	assert.False(t, Engaged(5))
}

func TestRetain(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(1, 10, 1, "p", []int{5})
	NoteSet(2, "q", 3) // waiting on player 2, who has no battle
	NoteSet(3, "r", 3)
	Retain(map[int]bool{3: true})
	_, ok := Current(1)
	assert.False(t, ok)
	assert.Equal(t, uint64(9), NoteSet(2, "q", 9), "player 2's line was dropped")
	assert.Equal(t, uint64(3), NoteSet(3, "r", 9), "player 3's line is kept")
}

func TestAllows(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.True(t, Allows(1, 5), "no battle: anyone")
	Begin(1, 10, 1, "p", []int{5})
	assert.True(t, Allows(1, 5))
	assert.False(t, Allows(1, 6), "a group waiting its turn")
}
