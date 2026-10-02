package company

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackMigrationJournalPreservesInstancesAndGrantMarkers(t *testing.T) {
	b := equipmentBrawl(t)
	rec, _ := module.registry.Get(7)
	rec.LeaderPackGranted = false
	b.aria.Character.Equipment.Pack = items.Item{}
	for i := range rec.Companions {
		rec.Companions[i].PackGranted = false
		b.companion(rec.Companions[i].ID).Character.Equipment.Pack = items.Item{}
		st, ok := module.runtime.Snapshot(b.companion(rec.Companions[i].ID).InstanceId)
		require.True(t, ok)
		rec.Companions[i].State = &st
	}
	module.registry.Put(rec)
	frame, traveller := items.New(33), items.New(32)
	oldSpec := frame.GetSpec()
	oldSpec.Type, oldSpec.Subtype, oldSpec.Name = items.Object, items.Mundane, "embroidered frame"
	frame.Spec, frame.Enchantments, frame.Uses = &oldSpec, 2, 7
	ballastSpec := items.ItemSpec{ItemId: 989780, Name: "old expedition supplies", Weight: 200000}
	items.SetTestItemSpec(&ballastSpec)
	t.Cleanup(func() { items.RemoveTestItemSpec(ballastSpec.ItemId) })
	ballast := items.New(ballastSpec.ItemId)
	b.aria.Character.Items = []items.Item{frame, traveller, ballast}
	originalSave := module.saveUser
	t.Cleanup(func() { module.saveUser = originalSave })
	module.saveUser = func(*users.UserRecord) error { return errors.New("disk full") }
	require.Error(t, module.PrepareAssets(7))
	pending, _ := module.registry.Get(7)
	require.NotNil(t, pending.AssetOperation)
	assert.True(t, pending.LeaderPackGranted)
	assert.Equal(t, frame.UUID, pending.AssetOperation.Equipment.Pack.UUID)
	assert.Equal(t, traveller.UUID, pending.Companions[0].State.Equipment.Pack.UUID)
	assert.Len(t, b.aria.Character.Items, 3, "user write failed; old state stays until recovery")
	module.saveUser = originalSave
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, frame.UUID, b.aria.Character.Equipment.Pack.UUID)
	assert.Equal(t, uint8(2), b.aria.Character.Equipment.Pack.Enchantments)
	assert.Equal(t, 7, b.aria.Character.Equipment.Pack.Uses)
	assert.Equal(t, items.Pack, b.aria.Character.Equipment.Pack.GetSpec().Type)
	require.Len(t, b.aria.Character.Items, 1)
	assert.Equal(t, ballast.UUID, b.aria.Character.Items[0].UUID)
	load, ok := encumbrance.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 200000, load.TotalGrams(), "no deletion or worn-weight double count")
	assert.Equal(t, 55000, load.CapacityGrams, "15+10+10+10+10 kg")
	packIDs := []string{b.aria.Character.Equipment.Pack.UUID.String()}
	for i := 1; i <= 4; i++ {
		packIDs = append(packIDs, b.companion(i).Character.Equipment.Pack.UUID.String())
	}
	// The real restart/copyover listener and another prepare never mint again.
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: b.aria.Character.RoomId})
	events.ProcessEvents()
	require.NoError(t, module.PrepareAssets(7))
	afterIDs := []string{b.aria.Character.Equipment.Pack.UUID.String()}
	for i := 1; i <= 4; i++ {
		afterIDs = append(afterIDs, b.companion(i).Character.Equipment.Pack.UUID.String())
	}
	assert.Equal(t, packIDs, afterIDs)
	// Removing a spent grant does not recreate it on the next preparation.
	b.aria.Character.Items = nil
	assert.Contains(t, b.cmd("company", "remove me pack"), "equipment updated")
	require.NoError(t, module.PrepareAssets(7))
	assert.Zero(t, b.aria.Character.Equipment.Pack.ItemId)
}

func TestAssignedPacksCapacityLossAndOverloadRecoveryThroughCommands(t *testing.T) {
	b := equipmentBrawl(t)
	load, ok := encumbrance.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 50000, load.CapacityGrams)
	beforeBurden := b.aria.Character.PersonalGrams()
	oldPack := b.aria.Character.Equipment.Pack
	// Assigned pack is excluded from burden.
	b.aria.Character.Equipment.Pack = items.Item{}
	assert.Equal(t, beforeBurden, b.aria.Character.PersonalGrams())
	b.aria.Character.Equipment.Pack = oldPack
	ballastSpec := items.ItemSpec{ItemId: 989781, Name: "weighted cargo", Weight: 49900}
	items.SetTestItemSpec(&ballastSpec)
	t.Cleanup(func() { items.RemoveTestItemSpec(ballastSpec.ItemId) })
	ballast := items.New(ballastSpec.ItemId)
	b.aria.Character.Items = []items.Item{ballast}
	assert.Contains(t, b.cmd("company", "remove me pack"), "equipment updated")
	load, _ = encumbrance.CurrentLoad(7)
	assert.Equal(t, 40000, load.CapacityGrams)
	assert.Equal(t, 50300, load.TotalGrams())
	band, known := encumbrance.CurrentBand(7)
	require.True(t, known)
	assert.Equal(t, 150, band.TravelDurationPct)
	assert.Contains(t, b.cmd("company", "inventory"), "no assigned pack")
	loose := items.New(33)
	b.aria.Character.Items = append(b.aria.Character.Items, loose)
	load, _ = encumbrance.CurrentLoad(7)
	assert.Equal(t, 40000, load.CapacityGrams, "spare frame pack supplies no capacity")
	assert.Contains(t, b.cmd("company", "equip me "+loose.ShorthandId()), "equipment updated")
	load, _ = encumbrance.CurrentLoad(7)
	assert.Equal(t, 55000, load.CapacityGrams)
	assert.Equal(t, 50300, load.TotalGrams(), "old pack is still cargo")
	assert.Equal(t, loose.UUID, b.aria.Character.Equipment.Pack.UUID)
	assert.Contains(t, b.cmd("company", "compare me "+oldPack.ShorthandId()), "55.0 -> 50.0")
	// Returning ordinary worn gear is checked against final cargo weight.
	b.aria.Character.Items = []items.Item{ballast}
	assert.Contains(t, b.cmd("company", "equip me "+oldPack.ShorthandId()), "no longer")
}

func TestPackCapacityTracksPresenceDeathDismissalAndResurrection(t *testing.T) {
	b := equipmentBrawl(t)
	before, _ := encumbrance.CurrentLoad(7)
	live := b.companion(1)
	live.Character.RoomId++
	away, _ := encumbrance.CurrentLoad(7)
	assert.Equal(t, before.CapacityGrams-10000, away.CapacityGrams)
	live.Character.RoomId = b.road.RoomId
	live.Character.Health = 0
	fallen, _ := encumbrance.CurrentLoad(7)
	assert.Equal(t, away.CapacityGrams, fallen.CapacityGrams)
	live.Character.Health = 20
	rec, _ := module.registry.Get(7)
	st := rec.Companions[0].State.Clone()
	packUUID := st.Equipment.Pack.UUID
	require.NoError(t, module.registry.MarkDead(7, 1, domain.CompanionDeath{OpID: "test", Remaining: 100, Allowance: 100}))
	require.NoError(t, module.registry.Revive(7, 1))
	revived, _ := module.registry.Get(7)
	assert.True(t, revived.Companions[0].PackGranted)
	assert.Equal(t, packUUID, revived.Companions[0].State.Equipment.Pack.UUID)
	// Dismissal leaves the recruit's pack with the recruit.
	assert.Contains(t, b.cmd("company", "dismiss tamsin"), "dismissed")
	after, _ := encumbrance.CurrentLoad(7)
	assert.Equal(t, before.CapacityGrams-10000, after.CapacityGrams)
	for _, itm := range b.aria.Character.Items {
		assert.NotEqual(t, packUUID, itm.UUID)
	}
}

func TestPackMigrationCompanySaveFailureDoesNotSpendGrant(t *testing.T) {
	useDataDir(t, t.TempDir())
	spec := items.ItemSpec{ItemId: 989782, Name: "test knapsack", Type: items.Pack, Subtype: items.Wearable, Weight: 400, CarryBonus: 10000}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	u := users.NewUserRecord(7, 1)
	u.Character.CompanyCargo = true
	u.Character.CargoMigrated = true
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(7) })
	m := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	m.starterPackForTest = spec.ItemId
	m.saveUser = func(*users.UserRecord) error { return nil }
	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	require.Error(t, m.PrepareAssets(7))
	_, exists := m.registry.Get(7)
	assert.False(t, exists)
	assert.Zero(t, u.Character.Equipment.Pack.ItemId)
	store.saveErr = nil
	require.NoError(t, m.PrepareAssets(7))
	granted := u.Character.Equipment.Pack
	require.Equal(t, spec.ItemId, granted.ItemId)
	require.NoError(t, m.PrepareAssets(7))
	assert.Equal(t, granted.UUID, u.Character.Equipment.Pack.UUID)
}

func TestPackSlotExcludedFromPersonalCombatModifiers(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	c.Equipment = characters.Worn{}
	assert.Zero(t, c.GetDefense())
	c.Equipment.Weapon = items.New(10013) // shipped tree trunk grants 3 defense
	assert.Equal(t, 3, c.GetDefense())
	beforeDefense, beforeStrength := c.GetDefense(), c.Equipment.StatMod("strength")
	spec := items.ItemSpec{ItemId: 989783, Name: "enchanted old pack", Type: items.Pack, Subtype: items.Wearable, Weight: 3000, CarryBonus: 15000, DamageReduction: 80, StatMods: map[string]int{"strength": 100}, WornBuffIds: []int{989783}}
	items.SetTestItemSpec(&spec)
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 989783, Name: "pack combat bonus", StatMods: map[string]int{"strength": 100}})
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId); buffs.RemoveTestBuffSpec(989783) })
	beforeWeight := c.PersonalGrams()
	pack := items.New(spec.ItemId)
	pack.Enchantments = 3
	// Frozen fields from an older save cannot bypass the equipment-slot rule.
	frozen := spec
	frozen.Type, frozen.Subtype = items.Object, items.Mundane
	pack.Spec = &frozen
	c.Equipment.Pack = pack
	c.SetPermaBuffs(nil)
	assert.Equal(t, beforeWeight, c.PersonalGrams())
	assert.Equal(t, beforeDefense, c.GetDefense())
	assert.Equal(t, beforeStrength, c.Equipment.StatMod("strength"))
	assert.False(t, c.HasBuff(989783))
}

func TestOrdinaryEquipmentRemovalCannotOverflowCargo(t *testing.T) {
	b := equipmentBrawl(t)
	armorSpec := items.ItemSpec{ItemId: 989784, Name: "test cuirass", Type: items.Body, Subtype: items.Wearable, Weight: 1000}
	ballastSpec := items.ItemSpec{ItemId: 989785, Name: "almost full cargo", Weight: 49900}
	for _, spec := range []items.ItemSpec{armorSpec, ballastSpec} {
		copy := spec
		items.SetTestItemSpec(&copy)
		t.Cleanup(func() { items.RemoveTestItemSpec(copy.ItemId) })
	}
	b.aria.Character.Equipment.Body = items.New(armorSpec.ItemId)
	b.aria.Character.Items = []items.Item{items.New(ballastSpec.ItemId)}
	beforeArmor := b.aria.Character.Equipment.Body
	beforeCargo := append([]items.Item(nil), b.aria.Character.Items...)
	beforeRecord, _ := module.registry.Get(7)
	assert.Contains(t, b.cmd("company", "remove me body"), "exceed company cargo capacity")
	assert.Equal(t, beforeArmor, b.aria.Character.Equipment.Body)
	assert.Equal(t, beforeCargo, b.aria.Character.Items)
	afterRecord, _ := module.registry.Get(7)
	assert.Equal(t, beforeRecord, afterRecord)
	// Once there is room, that exact armor returns to cargo.
	b.aria.Character.Items = nil
	assert.Contains(t, b.cmd("company", "remove me body"), "equipment updated")
	assert.Zero(t, b.aria.Character.Equipment.Body.ItemId)
	require.Len(t, b.aria.Character.Items, 1)
	assert.Equal(t, beforeArmor.UUID, b.aria.Character.Items[0].UUID)
}
