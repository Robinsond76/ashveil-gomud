package characters

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestColdSamplesActionStartAndExcludesExpiredBuffs(t *testing.T) {
	c := New()
	c.Buffs.List = []*buffs.Buff{{BuffId: 1010, TriggersLeft: 10}}
	c.SetCast(2, SpellAggroInfo{SpellId: "mm"})
	assert.Equal(t, 2, c.Aggro.RoundsWaiting)
	c.Buffs.List = append(c.Buffs.List, &buffs.Buff{BuffId: 1011, TriggersLeft: 10})
	c.SetCast(2, SpellAggroInfo{SpellId: "mm"})
	assert.Equal(t, 3, c.Aggro.RoundsWaiting)
	assert.True(t, c.Aggro.ColdDelayed)
	c.Buffs.List[1].TriggersLeft = 0
	assert.Zero(t, c.ColdDelay())
	assert.Equal(t, 3, c.Aggro.RoundsWaiting, "warming never resets the active timer")
	c.SetCast(2, SpellAggroInfo{SpellId: "mm"})
	assert.Equal(t, 2, c.Aggro.RoundsWaiting)
}

func TestColdDelaysOnlyExplicitSlingCapability(t *testing.T) {
	c := New()
	c.Buffs.List = []*buffs.Buff{{BuffId: 1012, TriggersLeft: 10}}
	c.Equipment.Weapon = items.Item{ItemId: 123, Spec: &items.ItemSpec{Subtype: items.Shooting, WaitRounds: 1}}
	c.SetAggro(0, 1, DefaultAttack)
	assert.Equal(t, 1, c.Aggro.RoundsWaiting)
	assert.False(t, c.Aggro.ColdDelayed)
	c.Equipment.Weapon.Spec.Sling = true
	c.SetAggro(0, 1, DefaultAttack)
	assert.Equal(t, 2, c.Aggro.RoundsWaiting)
	assert.True(t, c.Aggro.ColdDelayed)
	c.SetAggro(0, 1, DefaultAttack, 0)
	assert.Equal(t, 1, c.Aggro.RoundsWaiting, "explicit wait override cannot evade cold")
	c.Buffs.List = nil
	c.SetAggro(0, 1, DefaultAttack)
	assert.Equal(t, 1, c.Aggro.RoundsWaiting)
}
