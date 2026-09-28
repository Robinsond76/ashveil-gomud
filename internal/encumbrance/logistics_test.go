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
