package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainAttackAndRetargetable(t *testing.T) {
	cases := []struct {
		name         string
		aggro        *characters.Aggro
		plain, retar bool
	}{
		{"none", nil, false, true},
		{"melee", &characters.Aggro{Type: characters.DefaultAttack, MobInstanceId: 5}, true, true},
		{"a shot in the room", &characters.Aggro{Type: characters.Shooting, MobInstanceId: 5}, true, true},
		{"a shot through an exit", &characters.Aggro{Type: characters.Shooting, MobInstanceId: 5, ExitName: "north"}, false, false},
		{"a spell", &characters.Aggro{Type: characters.SpellCast}, false, false},
		{"a backstab", &characters.Aggro{Type: characters.BackStab, MobInstanceId: 5}, false, false},
		{"fleeing", &characters.Aggro{Type: characters.Flee}, false, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.plain, plainAttack(c.aggro), c.name)
		assert.Equal(t, c.retar, retargetable(c.aggro), c.name)
	}
}

func TestAttackTypeKeepsAShot(t *testing.T) {
	assert.Equal(t, characters.DefaultAttack, attackType(nil))
	assert.Equal(t, characters.Shooting, attackType(&characters.Aggro{Type: characters.Shooting}))
	assert.Equal(t, characters.DefaultAttack, attackType(&characters.Aggro{Type: characters.DefaultAttack}))
}

func engagementMob(t *testing.T, instanceId, hp, roomId int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: instanceId}
	m.Character.Health = hp
	m.Character.RoomId = roomId
	mobs.SetTestInstance(m)
	t.Cleanup(func() { mobs.RemoveTestInstance(instanceId) })
	return m
}

func TestClassifyPartyTarget(t *testing.T) {
	const room = 990101
	engagementMob(t, 8101, 10, room) // a living party member
	engagementMob(t, 8102, 0, room)  // a dead party member
	engagementMob(t, 8103, 10, room) // a bystander, alive and here
	engagementMob(t, 8104, 10, 4242) // alive, elsewhere
	members := map[int]bool{8101: true, 8102: true}
	alive := map[company.MemberKey]bool{mobparty.MemberKeyFor(8101): true}

	on := func(id int) *characters.Aggro { return &characters.Aggro{MobInstanceId: id} }
	cases := []struct {
		name    string
		aggro   *characters.Aggro
		state   targetState
		current int
	}{
		{"idle", nil, targetNone, 0},
		{"a living party member", on(8101), targetInParty, 8101},
		{"a dead party member", on(8102), targetNone, 8102},
		{"a bystander", on(8103), targetElsewhere, 0},
		{"gone elsewhere", on(8104), targetNone, 0},
		{"no longer exists", on(8199), targetNone, 0},
	}
	for _, c := range cases {
		state, current := classifyPartyTarget(c.aggro, members, alive, room)
		assert.Equal(t, c.state, state, c.name)
		assert.Equal(t, c.current, current, c.name)
	}
}

// twoByTwo is an enemy party with 8201 front-left and 8202 behind it, and
// 8203 front-right.
func twoByTwo(t *testing.T) (mobparty.Party, map[company.MemberKey]bool) {
	t.Helper()
	var f company.Formation
	require.NoError(t, f.Place(mobparty.MemberKeyFor(8201), 0, 0))
	require.NoError(t, f.Place(mobparty.MemberKeyFor(8202), 1, 0))
	require.NoError(t, f.Place(mobparty.MemberKeyFor(8203), 0, 2))
	engagementMob(t, 8201, 9, 1)
	engagementMob(t, 8202, 2, 1)
	engagementMob(t, 8203, 5, 1)
	alive := map[company.MemberKey]bool{
		mobparty.MemberKeyFor(8201): true,
		mobparty.MemberKeyFor(8202): true,
		mobparty.MemberKeyFor(8203): true,
	}
	return mobparty.Party{Members: []int{8201, 8202, 8203}, Formation: f}, alive
}

func TestChooseFromPartyPicksTheWeakestLegal(t *testing.T) {
	party, alive := twoByTwo(t)

	// Column 3, plain melee: only 8203 (front-right) is in reach.
	id, ok := chooseFromParty(0, 2, true, party, alive, formationcombat.ReachNone, strategy.Weakest, 0)
	require.True(t, ok)
	assert.Equal(t, 8203, id)

	// Column 1, plain melee: 8201 (9 HP) blocks 8202 (2 HP); 8203 is out
	// of lateral range. The weakest legal is 8201.
	id, ok = chooseFromParty(0, 0, true, party, alive, formationcombat.ReachNone, strategy.Weakest, 0)
	require.True(t, ok)
	assert.Equal(t, 8201, id)

	// Column 1 with a ranged weapon: 8202 is the weakest and in reach.
	id, ok = chooseFromParty(0, 0, true, party, alive, formationcombat.ReachAny, strategy.Weakest, 0)
	require.True(t, ok)
	assert.Equal(t, 8202, id)

	// Unplaced: fails open, the weakest living member anywhere.
	id, ok = chooseFromParty(0, 0, false, party, alive, formationcombat.ReachNone, strategy.Weakest, 0)
	require.True(t, ok)
	assert.Equal(t, 8202, id)

	// Nothing living: no choice.
	_, ok = chooseFromParty(0, 1, true, party, map[company.MemberKey]bool{}, formationcombat.ReachAny, strategy.Weakest, 0)
	assert.False(t, ok)
}

func TestLegalAgainstParty(t *testing.T) {
	party, alive := twoByTwo(t)
	assert.False(t, legalAgainstParty(2, true, party, 8201, alive, formationcombat.ReachNone), "out of lateral range")
	assert.True(t, legalAgainstParty(2, false, party, 8201, alive, formationcombat.ReachNone), "an unplaced attacker fails open")
	delete(alive, mobparty.MemberKeyFor(8201))
	assert.False(t, legalAgainstParty(2, false, party, 8201, alive, formationcombat.ReachNone), "but never at the dead")
}

func TestLeaderTurnText(t *testing.T) {
	_, alive := twoByTwo(t)
	mobs.GetInstance(8201).Character.Name = "bandit captain"
	mobs.GetInstance(8203).Character.Name = "bandit cutthroat"
	assert.Equal(t, `You can't reach the <ansi fg="mobname">bandit captain</ansi> from here. You turn toward the <ansi fg="mobname">bandit cutthroat</ansi>.`, leaderTurnText(8201, 8203, alive))
	delete(alive, mobparty.MemberKeyFor(8201))
	assert.Equal(t, `You turn toward the <ansi fg="mobname">bandit cutthroat</ansi>.`, leaderTurnText(8201, 8203, alive), "a fallen target is not unreachable")
	assert.Equal(t, `You turn toward the <ansi fg="mobname">bandit cutthroat</ansi>.`, leaderTurnText(0, 8203, alive))
}

// TestReassignWithinLostPartyNeverTurnsOnABystander: once a lost target is
// gone from the room, a company attacker is reassigned only to a party
// hostile to its leader, never to a peaceful bystander.
func TestReassignWithinLostPartyNeverTurnsOnABystander(t *testing.T) {
	const roomId = 990102
	shopkeeper := engagementMob(t, 8301, 5, roomId)
	shopkeeper.Groups = []string{"29a-merchants"}
	room := &rooms.Room{RoomId: roomId}
	room.SetTestOccupants(nil, []int{8301})
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(roomId) })

	_, ok := reassignWithinLostParty(7, 8399, 1, true, formationcombat.ReachAny, strategy.Weakest, 0, room)
	assert.False(t, ok, "the lost target is gone and the only party here is a bystander")

	// The squad the leader was fighting: its group hostile to the leader.
	footman := engagementMob(t, 8302, 8, roomId)
	footman.Groups = []string{"29a-squad"}
	room.SetTestOccupants(nil, []int{8301, 8302})
	mobs.MakeHostile("29a-squad", 7, 100)
	id, ok := reassignWithinLostParty(7, 8399, 1, true, formationcombat.ReachAny, strategy.Weakest, 0, room)
	require.True(t, ok)
	assert.Equal(t, 8302, id)
	_, ok = reassignWithinLostParty(8, 8399, 1, true, formationcombat.ReachAny, strategy.Weakest, 0, room)
	assert.False(t, ok, "hostile to that leader only")

	// A hostile mob is hostile to everyone.
	footman.Groups = nil
	footman.Hostile = true
	id, ok = reassignWithinLostParty(8, 8399, 1, true, formationcombat.ReachAny, strategy.Weakest, 0, room)
	require.True(t, ok)
	assert.Equal(t, 8302, id)

	// While the lost member is still in the room (dead, not yet removed),
	// its own party is used, whatever its hostility.
	footman.Hostile = false
	lost := engagementMob(t, 8303, 0, roomId)
	lost.Groups = []string{"29a-merchants"}
	room.SetTestOccupants(nil, []int{8301, 8302, 8303})
	id, ok = reassignWithinLostParty(8, 8303, 1, true, formationcombat.ReachAny, strategy.Weakest, 0, room)
	require.True(t, ok)
	assert.Equal(t, 8301, id)
}

func TestJoinsTheFight(t *testing.T) {
	m := engagementMob(t, 8401, 5, 1)
	m.Groups = []string{"29a-join"}
	assert.False(t, joinsTheFight(m, 7), "not hostile: stays out")

	m.Hostile = true
	assert.True(t, joinsTheFight(m, 7), "a hostile mob joins")
	m.Hostile = false

	mobs.MakeHostile("29a-join", 7, 100)
	assert.True(t, joinsTheFight(m, 7), "its group is hostile to the leader")
	assert.False(t, joinsTheFight(m, 8), "only to that leader")

	m.SetConversation(1)
	assert.False(t, joinsTheFight(m, 7), "never a mob in conversation")
}

// TestMarkHostilityNeverSpreadsToAnUntouchedGroup: the member a company
// member attacks has all its groups made hostile (as the leader's blow
// does); another member's group is only refreshed if already hostile.
func TestMarkHostilityNeverSpreadsToAnUntouchedGroup(t *testing.T) {
	struck := engagementMob(t, 8501, 5, 1)
	struck.Groups = []string{"29a-mh-a", "29a-mh-b"}
	bystander := engagementMob(t, 8502, 5, 1)
	bystander.Groups = []string{"29a-mh-c"}

	leader := &users.UserRecord{UserId: 7, Character: &characters.Character{}}
	leader.Character.SetAggro(0, 8501, characters.DefaultAttack)
	s := companySide{leader: leader}

	s.markHostility(mobparty.Party{Members: []int{8501, 8502}})
	assert.True(t, mobs.IsHostile("29a-mh-a", 7))
	assert.True(t, mobs.IsHostile("29a-mh-b", 7))
	assert.False(t, mobs.IsHostile("29a-mh-c", 7), "nobody touched the bystander's group")
}

// Phase 32d: the reassignment chooses by the attacker's rule.
func TestChooseFromPartyByRule(t *testing.T) {
	party, alive := twoByTwo(t)
	id, ok := chooseFromParty(0, 0, false, party, alive, formationcombat.ReachNone, strategy.Strongest, 0)
	require.True(t, ok)
	assert.Equal(t, 8201, id, "the strongest (unplaced: everyone is in reach)")
	id, ok = chooseFromParty(0, 0, false, party, alive, formationcombat.ReachNone, strategy.Assist, 8203)
	require.True(t, ok)
	assert.Equal(t, 8203, id, "the player's target")
	id, ok = chooseFromParty(0, 0, true, party, alive, formationcombat.ReachNone, strategy.Assist, 8203)
	require.True(t, ok)
	assert.Equal(t, 8201, id, "the player's target out of reach: the nearest in reach")
}
