package encumbrance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 32f: a member's share, and the pickup limit.
func TestMemberCapacity(t *testing.T) {
	assert.Equal(t, 20000, MemberCapacity(20000, 500, 0, 0))
	assert.Equal(t, 20000+3000+5000, MemberCapacity(20000, 500, 6, 5000))
	assert.Equal(t, 20000, MemberCapacity(20000, 500, -4, -1), "nothing negative")
}

func TestLoadWouldExceed(t *testing.T) {
	l := Load{PersonalGrams: 9000, CapacityGrams: 10000}
	assert.False(t, l.WouldExceed(1000), "reaching capacity exactly is allowed")
	assert.True(t, l.WouldExceed(1001))
	assert.False(t, l.WouldExceed(0), "adding nothing")
	over := Load{PersonalGrams: 12000, CapacityGrams: 10000}
	assert.True(t, over.WouldExceed(1), "already over: nothing more")
	assert.False(t, over.WouldExceed(0))
}

type fixedLoad struct {
	load Load
	ok   bool
}

func (f fixedLoad) CurrentLoad(int) (Load, bool) { return f.load, f.ok }

func TestWouldExceedThroughProvider(t *testing.T) {
	SetProvider(nil)
	_, refuse := WouldExceed(7, 5000)
	assert.False(t, refuse, "no provider: never refuses")

	SetProvider(fixedLoad{load: Load{PersonalGrams: 9000, CapacityGrams: 10000}, ok: true})
	t.Cleanup(func() { SetProvider(nil) })
	load, refuse := WouldExceed(7, 2000)
	assert.True(t, refuse)
	assert.Equal(t, 9000, load.TotalGrams())
	_, refuse = WouldExceed(7, 500)
	assert.False(t, refuse)

	SetProvider(fixedLoad{ok: false})
	_, refuse = WouldExceed(7, 5000)
	assert.False(t, refuse, "an untracked load never refuses")
}

func TestTooMuchToCarry(t *testing.T) {
	SetProvider(fixedLoad{load: Load{PersonalGrams: 9500, CapacityGrams: 10000}, ok: true})
	t.Cleanup(func() { SetProvider(nil) })
	text, refuse := TooMuchToCarry(7, 600)
	assert.True(t, refuse)
	assert.Contains(t, text, "9.5 kg of 10.0 kg")
	_, refuse = TooMuchToCarry(7, 500)
	assert.False(t, refuse)
}

// Phase 32f: partly used items stack apart and come out first.
func TestCargoKeepsUses(t *testing.T) {
	c, _ := Established(7)
	c, _ = c.Deposit(300, 2)
	c, _ = c.DepositUses(300, 2, 1)
	c, _ = c.DepositUses(300, 1, 1)
	c, _ = c.DepositUses(300, 2, 1)
	assert.Equal(t, []CargoStack{{ItemId: 300, Count: 2}, {ItemId: 300, Count: 2, Uses: 2}, {ItemId: 300, Count: 1, Uses: 1}}, c.Stacks)
	assert.Equal(t, 5, c.CountOf(300))

	c, uses, err := c.WithdrawOne(300)
	assert.NoError(t, err)
	assert.Equal(t, 1, uses, "the most used first")
	c, uses, _ = c.WithdrawOne(300)
	assert.Equal(t, 2, uses)

	_, _, err = c.WithdrawOne(999)
	assert.ErrorIs(t, err, ErrInsufficientCargo)
	_, err = c.DepositUses(300, -1, 1)
	assert.ErrorIs(t, err, ErrInvalidAmount)
	assert.ErrorIs(t, Cargo{LeaderUserID: 7, Stacks: []CargoStack{{ItemId: 1, Count: 1, Uses: -1}}}.Validate(), ErrInvalidCargo)

	c, err = c.Withdraw(300, 3)
	assert.NoError(t, err)
	assert.Empty(t, c.Stacks)
}

func TestCargoConsumeUse(t *testing.T) {
	c, _ := Established(7)
	c, _ = c.Deposit(300, 1)
	c, err := c.ConsumeUse(300, 3)
	assert.NoError(t, err)
	assert.Equal(t, []CargoStack{{ItemId: 300, Count: 1, Uses: 2}}, c.Stacks, "a full waterskin becomes a partly used one")
	c, _ = c.ConsumeUse(300, 3)
	c, _ = c.ConsumeUse(300, 3)
	assert.Empty(t, c.Stacks, "its last use empties it")

	c, _ = c.Deposit(400, 2)
	c, _ = c.ConsumeUse(400, 0)
	assert.Equal(t, 1, c.CountOf(400), "an item without uses is eaten whole")
	_, err = c.ConsumeUse(999, 1)
	assert.ErrorIs(t, err, ErrInsufficientCargo)
}
