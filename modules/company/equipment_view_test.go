package company

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func editorSlot(t *testing.T, view domain.EquipmentView, slot string) domain.EquipmentSlot {
	t.Helper()
	for _, s := range view.Slots {
		if s.Slot == slot {
			return s
		}
	}
	t.Fatalf("missing slot %s", slot)
	return domain.EquipmentSlot{}
}
func editorChoice(t *testing.T, slot domain.EquipmentSlot, ref string) domain.EquipmentChoice {
	t.Helper()
	for _, choice := range slot.Choices {
		if choice.Ref == ref {
			return choice
		}
	}
	t.Fatalf("missing choice %s", ref)
	return domain.EquipmentChoice{}
}
func applyEditorChoice(t *testing.T, b *brawl, choice domain.EquipmentChoice) string {
	t.Helper()
	command := strings.TrimPrefix(choice.Command, "company ")
	return b.cmd("company", command)
}

func TestEquipmentViewReadOnlyAndPreviewMatchesAppliedStats(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	spec := items.ItemSpec{ItemId: 989801, Name: "layered armor", Type: items.Body, Subtype: items.Wearable, Weight: 12000, DamageReduction: 8, StatMods: map[string]int{"strength": 5, "speed": -3}}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	armor := items.New(spec.ItemId)
	duplicate := items.New(spec.ItemId)
	duplicate.Enchantments = 1
	c.Items = []items.Item{armor, duplicate, items.New(30015)}
	require.NoError(t, c.Validate(true))
	before, err := yaml.Marshal(c)
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	round := util.GetRoundCount()
	view := domain.EquipmentViewOf(7)
	require.True(t, view.Available, view.Reason)
	require.Len(t, view.Slots, len(characters.AllSlots()))
	body := editorSlot(t, view, "body")
	require.Len(t, body.Choices, 2, "supplies never appear as armor choices")
	choice := editorChoice(t, body, duplicate.ShorthandId())
	require.True(t, choice.Allowed, choice.Reason)
	require.NotNil(t, choice.After)
	afterRead, _ := yaml.Marshal(c)
	assert.Equal(t, before, afterRead, "reading previews cannot change live buffs, equipment or items")
	afterRecord, _ := module.registry.Get(7)
	assert.Equal(t, record, afterRecord)
	assert.Equal(t, round, util.GetRoundCount())
	require.Contains(t, applyEditorChoice(t, b, choice), "equipment updated")
	assert.Equal(t, duplicate.UUID, c.Equipment.Body.UUID)
	assert.Equal(t, duplicate.Enchantments, c.Equipment.Body.Enchantments)
	load, _ := encumbrance.CurrentLoad(7)
	assert.Equal(t, *choice.After, equipmentStats(c, load))
	assert.Equal(t, 1, len(editorSlot(t, domain.EquipmentViewOf(7), "body").Choices), "only the unselected exact duplicate remains")
	assert.Contains(t, applyEditorChoice(t, b, choice), "no longer")
}

func TestEquipmentEditorExplicitHandsAndExactRemoval(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	c.SetSkill("dual-wield", 1)
	c.Equipment.Weapon, c.Equipment.Offhand = items.New(10004), items.Item{}
	first, second := items.New(10004), items.New(10004)
	first.Sharpen(2, 9)
	second.Sharpen(1, 7)
	c.Items = []items.Item{first, second}
	view := module.EquipmentView(7)
	main := editorChoice(t, editorSlot(t, view, "weapon"), first.ShorthandId())
	require.True(t, main.Allowed, main.Reason)
	assert.Equal(t, 2, main.After.EdgeBonus)
	assert.Equal(t, 9, main.After.EdgeStrikes)
	require.Contains(t, applyEditorChoice(t, b, main), "equipment updated")
	assert.Equal(t, first.UUID, c.Equipment.Weapon.UUID, "explicit main-hand choice must not route into an empty offhand")
	assert.Zero(t, c.Equipment.Offhand.ItemId)
	assert.Equal(t, 9, c.Equipment.Weapon.SharpStrikes)
	assert.Equal(t, *main.After, equipmentStats(c, mustEditorLoad(t, 7)))
	off := editorChoice(t, editorSlot(t, module.EquipmentView(7), "offhand"), second.ShorthandId())
	require.True(t, off.Allowed, off.Reason)
	assert.Equal(t, 1, off.After.OffhandEdgeBonus)
	assert.Equal(t, 7, off.After.OffhandEdgeStrikes)
	require.Contains(t, applyEditorChoice(t, b, off), "equipment updated")
	assert.Equal(t, second.UUID, c.Equipment.Offhand.UUID)
	assert.Equal(t, *off.After, equipmentStats(c, mustEditorLoad(t, 7)))
	c.Equipment.Weapon.SharpStrikes = 0
	assert.Zero(t, module.EquipmentView(7).Current.EdgeBonus, "spent edges provide no active damage bonus")
	removal := editorSlot(t, module.EquipmentView(7), "offhand").Remove
	require.NotNil(t, removal)
	c.Equipment.Offhand = items.New(10004)
	replacement := c.Equipment.Offhand
	assert.Contains(t, applyEditorChoice(t, b, *removal), "exact worn item is no longer")
	assert.Equal(t, replacement.UUID, c.Equipment.Offhand.UUID)
}

func TestEquipmentEditorTwoHandsUnavailableAndFinalCapacity(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	specs := []items.ItemSpec{
		{ItemId: 989802, Name: "reaching glaive", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Reach: true, Weight: 2000},
		{ItemId: 989803, Name: "full supplies", Weight: 49900},
	}
	for _, spec := range specs {
		copy := spec
		items.SetTestItemSpec(&copy)
		t.Cleanup(func() { items.RemoveTestItemSpec(copy.ItemId) })
	}
	glaive := items.New(989802)
	shield := items.New(20001)
	frozen := shield.GetSpec()
	frozen.Cursed = true
	shield.Spec = &frozen
	c.Equipment.Offhand, c.Items = shield, []items.Item{glaive}
	choice := editorChoice(t, editorSlot(t, module.EquipmentView(7), "weapon"), glaive.ShorthandId())
	assert.False(t, choice.Allowed)
	assert.Contains(t, choice.Reason, "cursed")
	c.Equipment.Offhand.Uncursed = true
	choice = editorChoice(t, editorSlot(t, module.EquipmentView(7), "weapon"), glaive.ShorthandId())
	require.True(t, choice.Allowed, choice.Reason)
	assert.Contains(t, choice.Returned, domain.PlainLabel(c.Equipment.Offhand))
	require.Contains(t, applyEditorChoice(t, b, choice), "equipment updated")
	assert.Zero(t, c.Equipment.Offhand.ItemId)
	assert.Equal(t, *choice.After, equipmentStats(c, mustEditorLoad(t, 7)))
	c.Items = []items.Item{items.New(989803)}
	removal := editorSlot(t, module.EquipmentView(7), "weapon").Remove
	require.NotNil(t, removal)
	assert.False(t, removal.Allowed)
	assert.Contains(t, removal.Reason, "capacity")
	assert.Contains(t, applyEditorChoice(t, b, *removal), "capacity")
	assert.Equal(t, glaive.UUID, c.Equipment.Weapon.UUID)
	packRemoval := editorSlot(t, module.EquipmentView(7), "pack").Remove
	require.NotNil(t, packRemoval)
	require.True(t, packRemoval.Allowed)
	require.Contains(t, applyEditorChoice(t, b, *packRemoval), "equipment updated")
	assert.Equal(t, *packRemoval.After, equipmentStats(c, mustEditorLoad(t, 7)))
	c.SetAggro(0, b.companion(1).InstanceId, characters.DefaultAttack)
	view := module.EquipmentView(7)
	assert.False(t, view.Available)
	assert.Contains(t, view.Reason, "battle")
}
func mustEditorLoad(t *testing.T, id int) encumbrance.Load {
	t.Helper()
	load, ok := encumbrance.CurrentLoad(id)
	require.True(t, ok)
	return load
}

func TestEquipmentEditorJournalRecoveryAndPrivateFeed(t *testing.T) {
	b := equipmentBrawl(t)
	weapon := items.New(10004)
	b.aria.Character.Items = []items.Item{weapon}
	choice := editorChoice(t, editorSlot(t, module.EquipmentView(7), "weapon"), weapon.ShorthandId())
	originalSave := module.saveUser
	t.Cleanup(func() { module.saveUser = originalSave })
	module.saveUser = func(*users.UserRecord) error { return errors.New("disk unavailable") }
	assert.Contains(t, applyEditorChoice(t, b, choice), "awaits recovery")
	assert.False(t, module.EquipmentView(7).Available)
	assert.Contains(t, module.EquipmentView(7).Reason, "awaits recovery")
	module.saveUser = originalSave
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, weapon.UUID, b.aria.Character.Equipment.Weapon.UUID)
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())
	count := 0
	var received domain.EquipmentView
	freshEvents(t)
	listener := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module == "Company.Equipment" {
			assert.Equal(t, 7, out.UserId, "private equipment is sent only to its requester")
			raw, ok := out.Payload.([]byte)
			require.True(t, ok)
			require.NoError(t, json.Unmarshal(raw, &received))
			count++
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, listener) })
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Zero(t, count, "nothing is built for a client not showing the editor")
	events.AddToQueue(gmcp.GMCPGearWatch{UserId: 7, Open: true, Slot: "weapon"})
	events.ProcessEvents()
	require.Positive(t, count, "opening the editor sends it at once")
	assert.False(t, editorSlot(t, received, "weapon").Pending)
	assert.True(t, editorSlot(t, received, "body").Pending, "only the shown slot is previewed")
	assert.Equal(t, weapon.ShorthandId(), editorSlot(t, received, "weapon").Equipped.Ref)
	beforeCount := count
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, beforeCount, count, "unchanged catalogue is not resent")
	events.AddToQueue(gmcp.GMCPCompanyRequest{UserId: 7})
	events.ProcessEvents()
	assert.Greater(t, count, beforeCount, "reconnect/full request resends equipment after the Company snapshot")
	events.AddToQueue(gmcp.GMCPGearWatch{UserId: 7, Open: false})
	events.ProcessEvents()
	closedCount := count
	b.aria.Character.Items = append(b.aria.Character.Items, items.New(10004))
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, closedCount, count, "a closed editor is not rebuilt or sent")
	assert.Empty(t, module.EquipmentView(99999).Slots)
}

func TestExplicitOffhandRetainsOtherHandPermanentBuff(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	buffSpec := buffs.BuffSpec{BuffId: 989809, Name: "weapon vigor", StatMods: map[string]int{"vitality": 20}, TriggerCount: 1}
	buffs.SetTestBuffSpec(&buffSpec)
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(buffSpec.BuffId) })
	c.SetSkill("dual-wield", 1)
	main, off := items.New(10004), items.New(10004)
	main.AddWornBuff(buffSpec.BuffId)
	off.AddWornBuff(buffSpec.BuffId)
	c.Equipment.Weapon, c.Equipment.Offhand = main, off
	require.NoError(t, c.Validate(true))
	require.True(t, c.HasBuff(buffSpec.BuffId))
	c.Health = c.HealthMax.Value
	beforeHealth := c.Health
	_, success, reason := c.Wear(items.New(10004), items.Offhand)
	require.True(t, success, reason)
	assert.True(t, c.HasBuff(buffSpec.BuffId), "main hand still supplies the permanent buff")
	assert.Equal(t, beforeHealth, c.Health, "no temporary buff removal may clamp health")
}

// The company feed asks for the editor every round: an unchanged leader must
// reuse the last view, and any change to equipment, cargo, availability or
// the round budget must rebuild it (review of Phase 34c).
func TestEquipmentViewReusedUntilLeaderChanges(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	spec := items.ItemSpec{ItemId: 989804, Name: "quilted coat", Type: items.Body, Subtype: items.Wearable, Weight: 3000, DamageReduction: 2}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	round := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(round) })
	coat := items.New(spec.ItemId)
	c.Items = []items.Item{coat}
	require.NoError(t, c.Validate(true))
	builds := func() int {
		module.equipmentViews.mu.Lock()
		defer module.equipmentViews.mu.Unlock()
		return module.equipmentViews.builds
	}

	start := builds()
	first := module.EquipmentView(7)
	require.True(t, first.Available, first.Reason)
	assert.Equal(t, first, module.EquipmentView(7))
	assert.Equal(t, start+1, builds(), "an unchanged leader reuses the last view")

	second := items.New(spec.ItemId)
	c.Items = append(c.Items, second)
	assert.Len(t, editorSlot(t, module.EquipmentView(7), "body").Choices, 2, "new cargo rebuilds the view")
	assert.Equal(t, start+2, builds())

	choice := editorChoice(t, editorSlot(t, module.EquipmentView(7), "body"), coat.ShorthandId())
	require.Contains(t, applyEditorChoice(t, b, choice), "equipment updated")
	view := module.EquipmentView(7)
	require.NotNil(t, editorSlot(t, view, "body").Equipped)
	assert.Equal(t, coat.ShorthandId(), editorSlot(t, view, "body").Equipped.Ref, "equipping rebuilds the view")
	assert.Equal(t, choice.After.Defense, view.Current.Defense)

	rebuilt := builds()
	module.EquipmentView(7)
	assert.Equal(t, rebuilt, builds())
	util.SetRoundCount(round + equipmentViewRefreshRounds)
	module.EquipmentView(7)
	assert.Equal(t, rebuilt+1, builds(), "a cached view is rebuilt after its round budget")

	c.SetAggro(0, b.companion(1).InstanceId, characters.DefaultAttack)
	assert.False(t, module.EquipmentView(7).Available, "entering battle rebuilds the view")
	c.Aggro = nil

	module.forgetEquipmentView(7)
	module.EquipmentView(7)
	assert.Equal(t, rebuilt+3, builds(), "a forgotten leader is rebuilt")

	// 35a2 review: choosing a class changes the gear rules, so it rebuilds
	// the view even when the class's health rate matches (ranger and no
	// class both gain 0.8 a level).
	p := balanceHPProvider(t)
	archetypes.SetProvider(p)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	module.EquipmentView(7)
	unchosen := builds()
	p.fakeArchetypes.player = "ranger"
	archetypes.SetProvider(p)
	module.EquipmentView(7)
	assert.Equal(t, unchosen+1, builds(), "a class choice rebuilds the view")
}

// What ticks every round without changing a preview (vitals, cooldowns, a
// buff's counters) must not rebuild the view through the provider the GMCP
// feed calls; a new buff must. A rebuild also drops offline leaders' views
// (review of the Phase 34 follow-up).
func TestEquipmentViewIgnoresRoundTicksAndPrunesOffline(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	spec := buffs.BuffSpec{BuffId: 989905, Name: "Keen", Description: "Sharp eyes.", RoundInterval: 2, TriggerCount: 1000000000}
	buffs.SetTestBuffSpec(&spec)
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(spec.BuffId) })
	round := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(round) })
	c.Items = []items.Item{items.New(10004)}
	c.Buffs.List = []*buffs.Buff{{BuffId: spec.BuffId, PermaBuff: true, TriggersLeft: buffs.TriggersLeftUnlimited}}
	builds := func() int {
		module.equipmentViews.mu.Lock()
		defer module.equipmentViews.mu.Unlock()
		return module.equipmentViews.builds
	}
	module.equipmentViews.mu.Lock()
	module.equipmentViews.byUser[424242] = equipmentViewEntry{}
	module.equipmentViews.mu.Unlock()

	module.forgetEquipmentView(7)
	require.True(t, domain.EquipmentViewOf(7).Available)
	start := builds()
	module.equipmentViews.mu.Lock()
	_, stale := module.equipmentViews.byUser[424242]
	module.equipmentViews.mu.Unlock()
	assert.False(t, stale, "a rebuild prunes offline leaders")

	for i := 0; i < 3; i++ {
		util.SetRoundCount(util.GetRoundCount() + 1)
		c.Buffs.List[0].RoundCounter++
		c.Health, c.Mana = c.Health-1, c.Mana+1
		c.Cooldowns = characters.Cooldowns{"bash": 3 - i}
		domain.EquipmentViewOf(7)
	}
	assert.Equal(t, start, builds(), "round ticks reuse the view")

	c.Buffs.List = append(c.Buffs.List, &buffs.Buff{BuffId: spec.BuffId, TriggersLeft: 3})
	domain.EquipmentViewOf(7)
	assert.Equal(t, start+1, builds(), "a new effect rebuilds the view")
}

// The feed previews only the slot the editor shows: other slots list what
// they hold and are pending, and their previews match the full view once
// selected (Phase 34 review).
func TestEquipmentViewFocusPreviewsOnlyTheShownSlot(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.aria.Character
	c.Items = []items.Item{items.New(10004), items.New(20001)}
	full := module.EquipmentView(7)
	focused := domain.EquipmentViewFocused(7, "weapon")
	for _, slot := range focused.Slots {
		if slot.Slot == "weapon" {
			assert.False(t, slot.Pending)
			assert.Equal(t, editorSlot(t, full, "weapon"), slot, "the shown slot matches the full view")
			continue
		}
		assert.True(t, slot.Pending, slot.Slot)
		assert.Empty(t, slot.Choices, slot.Slot)
		assert.Nil(t, slot.Remove, slot.Slot)
		assert.Equal(t, editorSlot(t, full, slot.Slot).Equipped, slot.Equipped, "a pending slot still shows what it holds")
	}
	offhand := domain.EquipmentViewFocused(7, "offhand")
	assert.Equal(t, editorSlot(t, full, "offhand"), editorSlot(t, offhand, "offhand"))
}
