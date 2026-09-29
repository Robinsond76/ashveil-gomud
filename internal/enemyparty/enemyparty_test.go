package enemyparty

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMob(t *testing.T, instanceId, hp int, groups ...string) *mobs.Mob {
	t.Helper()
	// A tagged test mob is hostile: since Phase 32d only hostile mobs
	// group by a shared tag.
	m := &mobs.Mob{InstanceId: instanceId, Groups: groups, Hostile: len(groups) > 0}
	m.Character.Health = hp
	m.Character.HealthMax.Value = hp
	mobs.SetTestInstance(m)
	t.Cleanup(func() { mobs.RemoveTestInstance(instanceId) })
	return m
}

func testRoom(t *testing.T, roomId int, mobIds ...int) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: roomId}
	r.SetTestOccupants(nil, mobIds)
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(roomId) })
	return r
}

func TestPartiesGroupByTagAndSkipCharmed(t *testing.T) {
	testMob(t, 9001, 10, "bandits")
	testMob(t, 9002, 20, "bandits")
	companion := testMob(t, 9003, 30, "bandits")
	companion.Character.Charmed = characters.NewCharm(7, characters.CharmPermanent, ``)
	testMob(t, 9004, 5) // untagged: its own party
	room := testRoom(t, 990001, 9001, 9002, 9003, 9004)

	parties := Parties(room)
	require.Len(t, parties, 2)
	assert.ElementsMatch(t, []int{9001, 9002}, parties[0].Members, "the charmed companion is not an enemy")
	assert.Equal(t, []int{9004}, parties[1].Members)
	assert.Equal(t, []int{9002, 9001}, parties[0].Members, "front to back by EHP")
}

func TestPartyOfFindsTheMembersParty(t *testing.T) {
	testMob(t, 9011, 10, "wolves")
	testMob(t, 9012, 10, "bandits")
	room := testRoom(t, 990002, 9011, 9012)

	p, ok := PartyOf(room, 9012)
	require.True(t, ok)
	assert.Equal(t, []int{9012}, p.Members)
	_, ok = PartyOf(room, 4242)
	assert.False(t, ok)
	_, ok = PartyOf(nil, 9012)
	assert.False(t, ok)
}

func TestAliveMarksGoneAndDeadMembers(t *testing.T) {
	testMob(t, 9021, 10, "bandits")
	testMob(t, 9022, 0, "bandits")
	p := mobparty.Party{Members: []int{9021, 9022, 9023}}

	alive := Alive(p)
	assert.True(t, alive[mobparty.MemberKeyFor(9021)])
	assert.False(t, alive[mobparty.MemberKeyFor(9022)], "at 0 HP")
	assert.False(t, alive[mobparty.MemberKeyFor(9023)], "no live instance")
}

func TestEffectiveHPMatchesRankMobsFormula(t *testing.T) {
	assert.InDelta(t, 200.0, EffectiveHP(200, 0), 0.001)
	assert.InDelta(t, 200.0, EffectiveHP(100, 100), 0.001)
	assert.InDelta(t, 100.0/0.05, EffectiveHP(100, 10000), 0.001)
}
