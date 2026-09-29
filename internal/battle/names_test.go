package battle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func named(id int, base string) EnemyName { return EnemyName{InstanceId: id, BaseName: base} }

func TestEnemyNamesFreezeAndGrow(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(7, 100, 1, "cutthroats", []int{42, 41})
	AssignEnemyNames(7, []EnemyName{named(42, "bandit cutthroat"), named(41, "bandit cutthroat")})
	assert.Equal(t, "first cutthroat", EnemyDisplayName(41, "fallback"))
	assert.Equal(t, "second cutthroat", EnemyDisplayName(42, "fallback"))

	Grow(7, "cutthroats", []int{42})
	AssignEnemyNames(7, []EnemyName{named(42, "bandit cutthroat")})
	Grow(7, "cutthroats", []int{43})
	AssignEnemyNames(7, []EnemyName{named(43, "bandit cutthroat")})
	assert.Equal(t, "second cutthroat", EnemyDisplayName(42, "fallback"))
	assert.Equal(t, "third cutthroat", EnemyDisplayName(43, "fallback"))
	Grow(7, "cutthroats", []int{40})
	AssignEnemyNames(7, []EnemyName{named(40, "bandit cutthroat")})
	assert.Equal(t, "first cutthroat", EnemyDisplayName(41, "fallback"))
	assert.Equal(t, "second cutthroat", EnemyDisplayName(42, "fallback"))
	assert.Equal(t, "third cutthroat", EnemyDisplayName(43, "fallback"))
	assert.Equal(t, "fourth cutthroat", EnemyDisplayName(40, "fallback"), "growth reserves earlier ordinals even for lower instance IDs")
}

func TestEnemyNamesUniqueAndLateDuplicate(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(7, 100, 1, "bandits", []int{41})
	AssignEnemyNames(7, []EnemyName{named(41, "bandit cutthroat")})
	assert.Equal(t, "bandit cutthroat", EnemyDisplayName(41, "fallback"))
	Grow(7, "bandits", []int{42})
	AssignEnemyNames(7, []EnemyName{named(42, "bandit cutthroat")})
	assert.Equal(t, "bandit cutthroat", EnemyDisplayName(41, "fallback"), "issued labels never change")
	assert.Equal(t, "second cutthroat", EnemyDisplayName(42, "fallback"))

	Grow(7, "bandits", []int{43, 44, 45})
	AssignEnemyNames(7, []EnemyName{
		{InstanceId: 43, BaseName: "the hedge witch"},
		{InstanceId: 44, BaseName: "", Noun: ""},
		{InstanceId: 45, BaseName: "goblin scout", Noun: "raider"},
	})
	assert.Equal(t, "the hedge witch", EnemyDisplayName(43, "fallback"))
	assert.Equal(t, "creature", EnemyDisplayName(44, "fallback"))
	assert.Equal(t, "goblin scout", EnemyDisplayName(45, "fallback"))
}

func TestEnemyNamesCollisions(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(7, 100, 1, "mixed", []int{1, 2, 3, 4, 5, 6, 7, 8, 25})
	AssignEnemyNames(7, []EnemyName{
		named(2, "Bandit Cutthroat"), named(1, "bandit cutthroat"),
		named(3, "goblin cutthroat"), named(4, "goblin cutthroat"),
		named(5, "bandit cutthroat"), named(6, "bandit cutthroat"),
		named(7, "bandit cutthroat"), named(8, "bandit cutthroat"),
		named(25, "first cutthroat"),
	})
	assert.Equal(t, "first bandit cutthroat", EnemyDisplayName(1, "fallback"))
	assert.Equal(t, "second bandit cutthroat", EnemyDisplayName(2, "fallback"))
	assert.Equal(t, "first goblin cutthroat", EnemyDisplayName(3, "fallback"))
	assert.Equal(t, "second goblin cutthroat", EnemyDisplayName(4, "fallback"))
	assert.Equal(t, "first cutthroat", EnemyDisplayName(25, "fallback"), "a literal authored name is reserved before ordinal labels")
	assert.Equal(t, "fifth bandit cutthroat", EnemyDisplayName(7, "fallback"))
	assert.Equal(t, "sixth bandit cutthroat", EnemyDisplayName(8, "fallback"))

	Grow(7, "mixed", []int{9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	members := make([]EnemyName, 0, 15)
	for id := 9; id <= 23; id++ {
		members = append(members, named(id, "bandit cutthroat"))
	}
	AssignEnemyNames(7, members)
	assert.Equal(t, "11th bandit cutthroat", EnemyDisplayName(13, "fallback"))
	assert.Equal(t, "12th bandit cutthroat", EnemyDisplayName(14, "fallback"))
	assert.Equal(t, "13th bandit cutthroat", EnemyDisplayName(15, "fallback"))
	assert.Equal(t, "21st bandit cutthroat", EnemyDisplayName(23, "fallback"))

	Grow(7, "mixed", []int{24})
	AssignEnemyNames(7, []EnemyName{named(24, "second bandit cutthroat")})
	assert.NotEqual(t, EnemyDisplayName(2, "fallback"), EnemyDisplayName(24, "fallback"))
}

func TestEnemyNamesSharedBattles(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(7, 100, 1, "bandits", []int{41, 42})
	AssignEnemyNames(7, []EnemyName{named(41, "bandit cutthroat"), named(42, "bandit cutthroat")})
	Begin(8, 100, 1, "bandits", []int{42, 43})
	AssignEnemyNames(8, []EnemyName{named(42, "bandit cutthroat"), named(43, "bandit cutthroat")})
	Begin(9, 100, 1, "bandits", []int{43, 44})
	AssignEnemyNames(9, []EnemyName{named(43, "bandit cutthroat"), named(44, "bandit cutthroat")})
	assert.Equal(t, "first cutthroat", EnemyDisplayName(41, "fallback"))
	assert.Equal(t, "second cutthroat", EnemyDisplayName(42, "fallback"))
	assert.Equal(t, "third cutthroat", EnemyDisplayName(43, "fallback"))
	assert.Equal(t, "fourth cutthroat", EnemyDisplayName(44, "fallback"))
	for _, userID := range []int{7, 8, 9} {
		current, ok := Current(userID)
		require.True(t, ok)
		assert.Equal(t, "first cutthroat", current.EnemyNames[41].DisplayName)
		assert.Equal(t, "fourth cutthroat", current.EnemyNames[44].DisplayName)
	}

	End(7)
	assert.Equal(t, "third cutthroat", EnemyDisplayName(43, "fallback"), "the other battle retains snapshots")
	Grow(8, "bandits", []int{45})
	AssignEnemyNames(8, []EnemyName{named(45, "bandit cutthroat")})
	assert.Equal(t, "fifth cutthroat", EnemyDisplayName(45, "fallback"))
	current, ok := Current(9)
	require.True(t, ok)
	assert.Equal(t, "fifth cutthroat", current.EnemyNames[45].DisplayName)
	Forget(8)
	assert.Equal(t, "fifth cutthroat", EnemyDisplayName(45, "fallback"))
}

func TestEnemyNamesLifecycleAndCopies(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Begin(7, 100, 1, "bandits", []int{41, 42})
	AssignEnemyNames(7, []EnemyName{named(41, "bandit cutthroat"), named(42, "bandit cutthroat")})
	current, ok := Current(7)
	require.True(t, ok)
	current.EnemyNames[41] = EnemyName{InstanceId: 41, DisplayName: "changed"}
	again, _ := Current(7)
	assert.Equal(t, "first cutthroat", again.EnemyNames[41].DisplayName)

	Retain(map[int]bool{})
	assert.Equal(t, "fallback", EnemyDisplayName(41, "fallback"))
	Begin(7, 100, 2, "bandits", []int{51, 52})
	AssignEnemyNames(7, []EnemyName{named(51, "bandit cutthroat"), named(52, "bandit cutthroat")})
	assert.Equal(t, "first cutthroat", EnemyDisplayName(51, "fallback"))
	End(7)
	assert.Equal(t, "fallback", EnemyDisplayName(51, "fallback"))
	Begin(7, 100, 3, "bandits", []int{61, 62})
	AssignEnemyNames(7, []EnemyName{named(61, "bandit cutthroat"), named(62, "bandit cutthroat")})
	Forget(7)
	assert.Equal(t, "fallback", EnemyDisplayName(61, "fallback"))
	Begin(7, 100, 4, "bandits", []int{71, 72})
	AssignEnemyNames(7, []EnemyName{named(71, "bandit cutthroat"), named(72, "bandit cutthroat")})
	assert.Equal(t, "first cutthroat", EnemyDisplayName(71, "fallback"))
	Reset()
	assert.Equal(t, "fallback", EnemyDisplayName(71, "fallback"))
}
