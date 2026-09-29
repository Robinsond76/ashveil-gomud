package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/stretchr/testify/assert"
)

func TestMobRefRetainsBattleLabelAfterMobIsGone(t *testing.T) {
	battle.Reset()
	t.Cleanup(battle.Reset)
	battle.Begin(71, 18, 1, "cutthroats", []int{41, 42})
	battle.AssignEnemyNames(71, []battle.EnemyName{
		{InstanceId: 41, BaseName: "bandit cutthroat", Noun: "cutthroat"},
		{InstanceId: 42, BaseName: "bandit cutthroat", Noun: "cutthroat"},
	})

	assert.Equal(t, "first cutthroat", mobRefById(41).Name)
	assert.Equal(t, "second cutthroat", mobRefById(42).Name)
	battle.End(71)
	assert.Empty(t, mobRefById(41).Name)
}
