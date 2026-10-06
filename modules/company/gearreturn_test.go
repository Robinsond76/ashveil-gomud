package company

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cargoUUIDs is the set of item instances in a leader's cargo.
func cargoUUIDs(u *users.UserRecord) map[uuid.UUID]int {
	out := map[uuid.UUID]int{}
	for _, itm := range u.Character.Items {
		out[itm.UUID]++
	}
	return out
}

// wornUUIDs is the set of item instances a character wears.
func wornUUIDs(c *characters.Character) map[uuid.UUID]int {
	out := map[uuid.UUID]int{}
	for _, slot := range characters.AllSlots() {
		if itm := c.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			out[itm.UUID]++
		}
	}
	return out
}

func TestDismissReturnsGivenGearAndLeavesDefaultsBehind(t *testing.T) {
	b := equipmentBrawl(t)
	given := items.New(10004)
	given.Sharpen(2, 9)
	b.aria.Character.Items = []items.Item{given}
	defaultWeapon := b.companion(1).Character.Equipment.Weapon
	require.Contains(t, b.cmd("company", "equip #1 "+given.ShorthandId()), "equipment updated")
	require.Equal(t, given.UUID, b.companion(1).Character.Equipment.Weapon.UUID)

	// Everything the leader and the companion hold before the dismissal.
	before := cargoUUIDs(b.aria)
	for id, n := range wornUUIDs(&b.companion(1).Character) {
		before[id] += n
	}

	text := b.cmd("company", "dismiss #1")
	assert.Contains(t, text, "returns to your cargo")
	got := cargoUUIDs(b.aria)
	assert.Equal(t, 1, got[given.UUID], "the given weapon comes back exactly once")
	assert.Equal(t, 1, got[defaultWeapon.UUID], "the weapon the player displaced stays theirs, once")
	for id, n := range got {
		assert.Equal(t, 1, n, "no item is duplicated")
		assert.Contains(t, before, id, "nothing appears from nowhere")
	}
	// The rest of the companion's own kit (default armor and the like) is not minted into cargo.
	assert.LessOrEqual(t, len(got), len(before))
	rec, _ := module.registry.Get(7)
	assert.Len(t, rec.Companions, 3, "the companion left the roster")
}

func TestDismissOfAnUnchangedCompanionReturnsNothing(t *testing.T) {
	b := equipmentBrawl(t)
	b.aria.Character.Items = nil
	text := b.cmd("company", "dismiss #1")
	assert.Contains(t, text, "Companion dismissed")
	assert.NotContains(t, text, "returns to your cargo")
	assert.Empty(t, b.aria.Character.Items, "a recruit's default gear is never minted into the leader's cargo")
}

func TestDismissReturnsGivenGearAfterPendingWriteRecovery(t *testing.T) {
	b := equipmentBrawl(t)
	given := items.New(10004)
	b.aria.Character.Items = []items.Item{given}
	require.Contains(t, b.cmd("company", "equip #1 "+given.ShorthandId()), "equipment updated")

	original := module.saveUser
	t.Cleanup(func() { module.saveUser = original })
	module.saveUser = func(u *users.UserRecord) error {
		if u.UserId == 7 {
			return errors.New("unavailable")
		}
		return original(u)
	}
	b.cmd("company", "dismiss #1")
	rec, _ := module.registry.Get(7)
	require.NotNil(t, rec.AssetOperation, "the company file already holds the return, so a crash cannot lose it")
	assert.Len(t, rec.Companions, 3)
	assert.Equal(t, 0, cargoUUIDs(b.aria)[given.UUID], "the leader's file has not been written yet")

	module.saveUser = original
	require.NoError(t, module.PrepareAssets(7))
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, 1, cargoUUIDs(b.aria)[given.UUID], "recovery returns it once, however often it runs")
	rec, _ = module.registry.Get(7)
	assert.Nil(t, rec.AssetOperation)
}

func TestDismissAllReturnsEveryGivenPiece(t *testing.T) {
	b := equipmentBrawl(t)
	a, z := items.New(10004), items.New(10004)
	b.aria.Character.Items = []items.Item{a, z}
	require.Contains(t, b.cmd("company", "equip #1 "+a.ShorthandId()), "equipment updated")
	require.Contains(t, b.cmd("company", "equip #2 "+z.ShorthandId()), "equipment updated")
	text := b.cmd("company", "dismiss all")
	assert.Contains(t, text, "returns to your cargo")
	got := cargoUUIDs(b.aria)
	assert.Equal(t, 1, got[a.UUID])
	assert.Equal(t, 1, got[z.UUID])
	assert.Empty(t, module.instances[7])
}

func TestLostCompanionGivesItsGearBack(t *testing.T) {
	b := equipmentBrawl(t)
	given := items.New(10004)
	b.aria.Character.Items = []items.Item{given}
	require.Contains(t, b.cmd("company", "equip #1 "+given.ShorthandId()), "equipment updated")
	require.NoError(t, module.registry.MarkDead(7, 1, domain.CompanionDeath{OpID: "op", Allowance: 10, Remaining: 0}))
	require.NoError(t, module.expire(7, 1))
	assert.Equal(t, 1, cargoUUIDs(b.aria)[given.UUID], "gear a lost companion held returns to the leader")
}

func TestReturnableGearSetsDefaultsAsideOnceAndReturnsDollGear(t *testing.T) {
	m := &CompanyModule{starterPackForTest: 9999}
	spec := items.ItemSpec{ItemId: 9999, Name: "starter pack", Weight: 100, CarryBonus: 1000}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	st := domain.MemberState{}
	st.Equipment.Pack = items.New(9999)
	st.Items = []items.Item{items.New(10004)}
	doll := characters.NewDollState(0)
	doll.Equipment.Offhand = items.New(10004)
	st.Dolls = []characters.DollState{doll}
	c := domain.Companion{ID: 1, State: &st}
	got, gold := m.returnableGear(7, c)
	require.Len(t, got, 2, "the starter pack and the doll's cudgel stay; the carried item and the doll's extra return")
	assert.Zero(t, gold)
}

func TestEquipmentViewShowsACompanionAndMatchesTheCommand(t *testing.T) {
	b := equipmentBrawl(t)
	armor := items.ItemSpec{ItemId: 989851, Name: "test plate", Type: items.Body, Subtype: items.Wearable, Weight: 4000, DamageReduction: 6}
	items.SetTestItemSpec(&armor)
	t.Cleanup(func() { items.RemoveTestItemSpec(armor.ItemId) })
	plate := items.New(armor.ItemId)
	b.aria.Character.Items = []items.Item{plate}

	view := domain.EquipmentViewMember(7, "#1", "body")
	require.True(t, view.Available, view.Reason)
	assert.Equal(t, "#1", view.Member)
	require.Len(t, view.Members, 5, "you and four companions")
	assert.Equal(t, "me", view.Members[0].Ref)
	assert.True(t, view.Members[1].Ready)
	body := editorSlot(t, view, "body")
	choice := editorChoice(t, body, plate.ShorthandId())
	require.True(t, choice.Allowed, choice.Reason)
	assert.Contains(t, choice.Command, "company equip #1 ", "the choice names the companion")
	require.NotNil(t, choice.After)
	before := view.Current.Defense
	assert.Greater(t, choice.After.Defense, before, "the preview shows the companion's armor")
	assert.Equal(t, before, b.companion(1).Character.GetDefense(), "a preview changes nothing")

	text := applyEditorChoice(t, b, choice)
	assert.Contains(t, text, "equipment updated")
	assert.Contains(t, text, "Now:", "the reply names what changed")
	assert.Contains(t, text, "protection")
	assert.Equal(t, choice.After.Defense, b.companion(1).Character.GetDefense(), "the applied change matches the preview")

	// Your own view is unchanged by looking at a companion.
	mine := domain.EquipmentViewMember(7, "me", "body")
	assert.Equal(t, "me", mine.Member)
	assert.Len(t, editorSlot(t, mine, "body").Choices, 0, "the plate went to the companion, not your cargo")
}

func TestEquipmentViewForAnAwayCompanionIsNotReadyButStillShown(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.companion(2)
	home := c.Character.RoomId
	c.Character.RoomId = 920102
	defer func() { c.Character.RoomId = home }() // the brawl's cleanup removes it from its room
	view := domain.EquipmentViewMember(7, "#2", "weapon")
	assert.False(t, view.Available)
	assert.Contains(t, view.Reason, "present")
	assert.NotEmpty(t, view.Slots, "its slots still show")
	assert.False(t, view.Members[2].Ready)
	unknown := domain.EquipmentViewMember(7, "#99", "weapon")
	assert.False(t, unknown.Available)
}

func TestStatChangesNamesWhatMoved(t *testing.T) {
	b := equipmentBrawl(t)
	actor := &b.companion(1).Character
	clone, err := cloneCharacter(actor)
	require.NoError(t, err)
	assert.Empty(t, statChanges(actor, clone), "no change, no line")
	clone.Equipment.Weapon = items.New(10004)
	clone.Equipment.Offhand = items.Item{}
	line := statChanges(actor, clone)
	if actor.Equipment.Weapon.ItemId != 10004 {
		assert.Contains(t, line, "damage")
	}
}
