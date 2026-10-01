package parties

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// resetPartyMap clears the global state between tests.
func resetPartyMap() {
	partyMap = map[int]*Party{}
}

func TestNew(t *testing.T) {
	resetPartyMap()

	p := New(1)
	assert.NotNil(t, p)
	assert.Equal(t, 1, p.LeaderUserId)
	assert.Equal(t, []int{1}, p.UserIds)

	// Creating a second party for the same user returns nil
	p2 := New(1)
	assert.Nil(t, p2)
}

func TestGet(t *testing.T) {
	resetPartyMap()

	assert.Nil(t, Get(99))

	New(1)
	assert.NotNil(t, Get(1))
}

func TestParty_IsLeader(t *testing.T) {
	resetPartyMap()
	p := New(1)

	assert.True(t, p.IsLeader(1))
	assert.False(t, p.IsLeader(2))
}

func TestParty_IsMember(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)
	p.AcceptInvite(2)

	assert.True(t, p.IsMember(1))
	assert.True(t, p.IsMember(2))
	assert.False(t, p.IsMember(3))
}

func TestParty_InviteAndAccept(t *testing.T) {
	resetPartyMap()
	p := New(1)

	ok := p.InvitePlayer(2)
	assert.True(t, ok)
	assert.True(t, p.Invited(2))

	ok = p.AcceptInvite(2)
	assert.True(t, ok)
	assert.False(t, p.Invited(2))
	assert.True(t, p.IsMember(2))
}

func TestParty_InviteAlreadyInParty(t *testing.T) {
	resetPartyMap()
	New(1)
	p2 := New(2)

	// user 1 is already in a party; inviting them should fail
	ok := p2.InvitePlayer(1)
	assert.False(t, ok)
}

func TestParty_DeclineInvite(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)

	ok := p.DeclineInvite(2)
	assert.True(t, ok)
	assert.False(t, p.Invited(2))
	assert.False(t, p.IsMember(2))

	// Declining again returns false
	ok = p.DeclineInvite(2)
	assert.False(t, ok)
}

func TestParty_Leave_NonLeader(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)
	p.AcceptInvite(2)

	ok := p.Leave(2)
	assert.True(t, ok)
	assert.False(t, p.IsMember(2))
	assert.True(t, p.IsLeader(1), "leader should not change")
}

func TestParty_Leave_LeaderTransfers(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)
	p.AcceptInvite(2)

	ok := p.Leave(1)
	assert.True(t, ok)
	assert.False(t, p.IsMember(1))
	assert.Equal(t, 2, p.LeaderUserId, "leadership should transfer to remaining member")
}

func TestParty_Leave_LastMemberDisbands(t *testing.T) {
	resetPartyMap()
	p := New(1)

	ok := p.Leave(1)
	assert.True(t, ok)
	assert.Nil(t, Get(1), "party should be disbanded after last member leaves")
}

func TestParty_Disband(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)
	p.AcceptInvite(2)

	p.Disband()
	assert.Nil(t, Get(1))
	assert.Nil(t, Get(2))
}

func TestParty_GetRank_Default(t *testing.T) {
	resetPartyMap()
	p := New(1)

	assert.Equal(t, "middle", p.GetRank(1))
}

func TestParty_SetRank(t *testing.T) {
	resetPartyMap()
	p := New(1)

	p.SetRank(1, "front")
	assert.Equal(t, "front", p.GetRank(1))

	p.SetRank(1, "back")
	assert.Equal(t, "back", p.GetRank(1))

	// Invalid rank resets to middle (deletes from map)
	p.SetRank(1, "middle")
	assert.Equal(t, "middle", p.GetRank(1))
}

func TestParty_ChanceToBeTargetted(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)
	p.AcceptInvite(2)
	p.InvitePlayer(3)
	p.AcceptInvite(3)

	p.SetRank(1, "front")
	p.SetRank(2, "back")
	// user 3 is middle (default)

	assert.Equal(t, 1, p.ChanceToBeTargetted(1))
	assert.Equal(t, 1, p.ChanceToBeTargetted(2))
	assert.Equal(t, 1, p.ChanceToBeTargetted(3))
}

func TestParty_AutoAttack(t *testing.T) {
	resetPartyMap()
	p := New(1)

	// Turning on returns false (was not already on)
	changed := p.SetAutoAttack(1, true)
	assert.False(t, changed)
	assert.Equal(t, []int{1}, p.GetAutoAttackUserIds())

	// Turning on again returns true (already on, no change)
	changed = p.SetAutoAttack(1, true)
	assert.True(t, changed)

	// Turning off returns true (was on)
	changed = p.SetAutoAttack(1, false)
	assert.True(t, changed)
	assert.Empty(t, p.GetAutoAttackUserIds())

	// Turning off when not on returns false
	changed = p.SetAutoAttack(1, false)
	assert.False(t, changed)
}

func TestParty_GetMembers_IsCopy(t *testing.T) {
	resetPartyMap()
	p := New(1)

	members := p.GetMembers()
	members = append(members, 99)
	assert.Equal(t, 1, len(p.UserIds), "modifying returned slice should not affect party")
}

func TestParty_GetInvited_IsCopy(t *testing.T) {
	resetPartyMap()
	p := New(1)
	p.InvitePlayer(2)

	invited := p.GetInvited()
	invited = append(invited, 99)
	assert.Equal(t, 1, len(p.InviteUserIds), "modifying returned slice should not affect party")
}
