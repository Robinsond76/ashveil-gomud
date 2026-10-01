package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/withdrawal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30a: a hobbled (or hamstrung) fighter cannot flee; anyone else can.
// Since Phase 33c, flee is the retreat order.
func TestFleeRefusedWhileHobbled(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	buffs.SetTestFlag("no-flee")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 99501, Name: "Test Hobbled", RoundInterval: 100000, TriggerCount: 4, Flags: []string{"no-flee"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(99501) })
	battle.Reset()
	t.Cleanup(battle.Reset)

	user := userWithItem(t, 9501, drinkableSpec("waterskin", 10, 1))
	room := testRoom()
	room.Exits = map[string]exit.RoomExit{"north": {RoomId: 2}}
	user.Character.RoomId = room.RoomId
	battle.Begin(user.UserId, room.RoomId, 1, "party", []int{501})
	user.Character.SetAggro(0, 501, characters.DefaultAttack)
	require.NoError(t, user.Character.AddBuff(99501, false))

	out := heard(t, func() { Flee(``, user, room, 0) })
	assert.Contains(t, out, withdrawal.LeaderPinned)
	assert.Equal(t, characters.DefaultAttack, user.Character.Aggro.Type, "the withdrawal was never ordered")

	user.Character.RemoveBuff(99501)
	out = heard(t, func() { Flee(``, user, room, 0) })
	assert.Contains(t, out, "begin an ordered retreat north")
	assert.Equal(t, characters.Retreat, user.Character.Aggro.Type)
	assert.Equal(t, 501, user.Character.Aggro.RetreatInfo.ResumeMobID, "the aim is kept for a failed attempt")
	assert.Zero(t, user.Character.Aggro.MobInstanceId, "the order itself aims at nobody, so a fallen target can't clear it")
}

// Phase 33c owner review: flee (the retreat order) still leaves a fight
// with another player, which is no battle, and out of any fight it is
// refused rather than walking the player off.
func TestFleeLeavesAPlayerFight(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	battle.Reset()
	t.Cleanup(battle.Reset)
	user := userWithItem(t, 9502, drinkableSpec("waterskin", 10, 1))
	room := testRoom()
	room.Exits = map[string]exit.RoomExit{"north": {RoomId: 2}}
	user.Character.RoomId = room.RoomId

	out := heard(t, func() { Flee(``, user, room, 0) })
	assert.Contains(t, out, "You are not in a battle.")
	assert.Nil(t, user.Character.Aggro)

	foe := users.NewUserRecord(9503, 9503)
	foe.Character.RoomId = room.RoomId
	users.SetTestUser(foe)
	t.Cleanup(func() { users.RemoveTestUser(9503) })
	room.AddPlayer(foe.UserId)
	room.AddPlayer(user.UserId)
	foe.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
	require.False(t, InBattle(user))

	out = heard(t, func() { Flee(``, user, room, 0) })
	assert.Contains(t, out, "ordered retreat north")
	require.NotNil(t, user.Character.Aggro)
	assert.Equal(t, characters.Retreat, user.Character.Aggro.Type)
}
