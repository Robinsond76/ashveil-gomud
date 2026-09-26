package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func weighted(t *testing.T, id, grams int) items.Item {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "weighted", Weight: grams})
	t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	return items.Item{ItemId: id}
}

// Phase 28: living companions' worn and carried gear counts toward the
// company load, from the live mob when it's out, else from the record; a
// fallen companion's gear stays with the body.
func TestCompanionGearGrams(t *testing.T) {
	sword, pack, armour, spear := weighted(t, 988001, 1500), weighted(t, 988002, 400), weighted(t, 988003, 9000), weighted(t, 988004, 2000)

	recorded := domain.MemberState{Level: 1, Items: []items.Item{pack}}
	recorded.Equipment.Weapon = sword // 1.9 kg on the record
	stale := domain.MemberState{Level: 1, Items: []items.Item{pack, pack, pack}}
	live := domain.MemberState{Level: 1}
	live.Equipment.Weapon = spear // 2 kg on the live mob
	fallen := domain.MemberState{Level: 1}
	fallen.Equipment.Body = armour

	runtime := &fakeRuntime{live: map[int]bool{501: true}, liveState: map[int]domain.MemberState{501: live}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, State: &recorded},
			{ID: 2, MobTemplateID: 58, State: &stale},
			{ID: 3, MobTemplateID: 58, State: &fallen, Death: &domain.CompanionDeath{OpID: "x", Remaining: 60}},
			{ID: 4, MobTemplateID: 58},
		}},
	}}, runtime)
	module.setInstance(7, 2, 501) // out now: its live gear, not the stale record

	assert.Equal(t, 1900+2000, module.CompanionGearGrams(7), "record for #1, live for #2; not the fallen #3; #4 has none recorded")
	assert.Zero(t, module.CompanionGearGrams(8), "no company")

	delete(runtime.live, 501) // gone: back to its record
	assert.Equal(t, 1900+1200, module.CompanionGearGrams(7))
}
