package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 30a: a hobbled (or hamstrung) fighter cannot flee; anyone else can.
func TestFleeRefusedWhileHobbled(t *testing.T) {
	buffs.SetTestFlag("no-flee")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 99501, Name: "Test Hobbled", RoundInterval: 100000, TriggerCount: 4, Flags: []string{"no-flee"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(99501) })

	user := users.NewUserRecord(9501, 9501)
	user.Character.SetAggro(0, 1, characters.DefaultAttack)
	assert.NoError(t, user.Character.AddBuff(99501, false))

	handled, err := Flee(``, user, nil, 0)
	assert.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, characters.DefaultAttack, user.Character.Aggro.Type, "the flight was never started")

	user.Character.RemoveBuff(99501)
	Flee(``, user, nil, 0)
	assert.Equal(t, characters.Flee, user.Character.Aggro.Type)
}
