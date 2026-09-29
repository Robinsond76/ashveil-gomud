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

// Review findings: a companion charmed away by another player carries
// nothing for this company; one with no record yet weighs its template's
// gear; an unreadable company weighs nothing; an item with a stale spec
// copy weighs its base data.
func TestCompanionGearGramsEdges(t *testing.T) {
	sword, pack := weighted(t, 988011, 1500), weighted(t, 988012, 400)
	live := domain.MemberState{Level: 1}
	live.Equipment.Weapon = sword
	frozen := items.Item{ItemId: 988012, Spec: &items.ItemSpec{ItemId: 988012, Weight: 0}}
	recorded := domain.MemberState{Level: 1, Items: []items.Item{frozen}}
	template := domain.MemberState{Level: 1, Items: []items.Item{pack, pack}}

	runtime := &fakeRuntime{
		live:          map[int]bool{601: true},
		liveState:     map[int]domain.MemberState{601: live},
		stolen:        map[int]bool{},
		templateState: &template,
	}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58},                   // no record yet: its template's gear
			{ID: 2, MobTemplateID: 58, State: &recorded}, // a stale spec copy
			{ID: 3, MobTemplateID: 58},                   // out, live
		}},
	}}, runtime)
	module.setInstance(7, 3, 601)

	assert.Equal(t, 800+400+1500, module.CompanionGearGrams(7))
	runtime.stolen[601] = true
	assert.Equal(t, 800+400, module.CompanionGearGrams(7), "charmed away: not this company's to carry")
	module.loadErr = assert.AnError
	assert.Zero(t, module.CompanionGearGrams(7), "an unreadable company weighs nothing")
}

func pack(t *testing.T, id, bonus int) items.Item {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "pack", Weight: 600, CarryBonus: bonus})
	t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	return items.Item{ItemId: id}
}

// Phase 32f: the companions who carry are the ones whose gear is weighed.
// A live one brings its Strength and its largest pack; one not out brings
// its recorded (or template) pack and no Strength; the fallen and the
// charmed away carry nothing.
func TestCompanionCarry(t *testing.T) {
	satchel, frame := pack(t, 988021, 5000), pack(t, 988022, 15000)
	recorded := domain.MemberState{Level: 1, Items: []items.Item{satchel, frame}}
	live := domain.MemberState{Level: 1, Items: []items.Item{satchel}}
	template := domain.MemberState{Level: 1, Items: []items.Item{satchel}}
	runtime := &fakeRuntime{
		live:          map[int]bool{701: true, 702: true},
		liveState:     map[int]domain.MemberState{701: live, 702: live},
		strength:      map[int]int{701: 12},
		stolen:        map[int]bool{702: true},
		templateState: &template,
	}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, State: &recorded},
			{ID: 2, MobTemplateID: 58},
			{ID: 3, MobTemplateID: 58, State: &recorded, Death: &domain.CompanionDeath{OpID: "x", Remaining: 60}},
			{ID: 4, MobTemplateID: 58},
			{ID: 5, MobTemplateID: 58},
		}},
	}}, runtime)
	module.setInstance(7, 4, 701)
	module.setInstance(7, 5, 702)

	assert.Equal(t, []domain.MemberCarry{
		{PackGrams: 15000},              // #1: the larger of its two packs
		{PackGrams: 5000},               // #2: its template's pack
		{Strength: 12, PackGrams: 5000}, // #4: live
	}, module.CompanionCarry(7), "not the fallen #3 or the charmed-away #5")
	assert.Nil(t, module.CompanionCarry(8), "no company")
	module.loadErr = assert.AnError
	assert.Nil(t, module.CompanionCarry(7), "an unreadable company carries nothing")
}
