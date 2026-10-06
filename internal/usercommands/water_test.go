package usercommands

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a: water sources.

func skinSpec() items.ItemSpec {
	spec := drinkableSpec("waterskin", 40, 5)
	spec.Refillable = "water"
	return spec
}

func waterRoom() *rooms.Room {
	room := testRoom()
	room.Resources = []string{rooms.ResourceWater}
	return room
}

func thirstyProvisioner() *fakeProvisioner {
	return &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.LeaderMemberKey,
		Name:   "Tester",
		Needs:  survival.Needs{Hunger: 100, Thirst: 60, Fatigue: 100},
	}}
}

func TestDrinkWaterAtASourceProvisionsWithoutSpendingItems(t *testing.T) {
	for _, words := range []string{"water", "source", "from water"} {
		fake := thirstyProvisioner()
		useFakeProvisioner(t, fake)
		user := userWithItem(t, 17, skinSpec())
		before := gametime.GetDate()

		out := heard(t, func() {
			handled, err := Drink(words, user, waterRoom(), 0)
			require.NoError(t, err)
			assert.True(t, handled)
		})
		assert.Equal(t, 1, fake.calls, words)
		assert.Equal(t, survival.Benefit{Hydration: WaterSourceHydration}, fake.lastBenefit, words)
		assert.Equal(t, 5, user.Character.Items[0].Uses, "%s: the waterskin is untouched", words)
		assert.Contains(t, out, "drink deeply", words)
		assert.Equal(t, before, gametime.GetDate(), "water takes no game time")
	}
}

func TestDrinkSourceNamesACompanion(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.CompanionMemberKey(2), Name: "Bear",
		Needs: survival.Needs{Hunger: 100, Thirst: 80, Fatigue: 100},
	}}
	fake.selectors = memberSelectors("#2")
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, skinSpec())

	_, err := Drink("water #2", user, waterRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, survival.Benefit{Hydration: WaterSourceHydration}, fake.lastBenefit)
}

func TestDrinkSourceAwayFromWaterIsRefusedAndWaterKeepsItsOldMeaning(t *testing.T) {
	fake := thirstyProvisioner()
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, skinSpec())

	out := heard(t, func() {
		_, err := Drink("source", user, testRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "no fresh water")
	assert.Zero(t, fake.calls)

	// "drink water" with no source still finds the carried waterskin.
	_, err := Drink("water", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, 4, user.Character.Items[0].Uses)
}

func TestDrinkWaterPrefersAnItemNamedWater(t *testing.T) {
	fake := thirstyProvisioner()
	useFakeProvisioner(t, fake)
	spec := drinkableSpec("water", 10, 2)
	user := userWithItem(t, 17, spec)

	_, err := Drink("water", user, waterRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, survival.Benefit{Hydration: 10}, fake.lastBenefit, "the item, not the source")
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestFillRestoresARefillableAndIgnoresTheRest(t *testing.T) {
	useFakeProvisioner(t, thirstyProvisioner())
	user := userWithItem(t, 17, skinSpec())
	user.Character.Items[0].Uses = 1
	potion := drinkableSpec("potion", 10, 2)
	potion.ItemId = 9004
	user.Character.Items = append(user.Character.Items, items.Item{ItemId: 9004, Uses: 1, Spec: &potion})

	out := heard(t, func() {
		_, err := Fill("", user, waterRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "You fill")
	assert.Equal(t, 5, user.Character.Items[0].Uses)
	assert.Equal(t, 1, user.Character.Items[1].Uses, "a potion can't be filled")

	out = heard(t, func() {
		_, err := Fill("potion", user, waterRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "can't fill")
	assert.Equal(t, 1, user.Character.Items[1].Uses)

	out = heard(t, func() {
		_, err := Fill("waterskin", user, waterRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "already full")
}

func TestFillIsRefusedAwayFromWater(t *testing.T) {
	user := userWithItem(t, 17, skinSpec())
	user.Character.Items[0].Uses = 1
	out := heard(t, func() {
		_, err := Fill("", user, testRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "no fresh water")
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestWaterCommandsAreRefusedInABattle(t *testing.T) {
	fake := thirstyProvisioner()
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 18, skinSpec())
	user.Character.Items[0].Uses = 1
	room := waterRoom()
	battle.Reset()
	t.Cleanup(battle.Reset)
	user.Character.RoomId = room.RoomId
	battle.Begin(18, room.RoomId, 1, "party", []int{501})
	user.Character.SetAggro(0, 501, characters.DefaultAttack)
	engagement.Resume(18)

	for name, run := range map[string]func(){
		"drink water": func() { _, _ = Drink("water", user, room, 0) },
		"fill":        func() { _, _ = Fill("", user, room, 0) },
	} {
		out := heard(t, run)
		assert.Equal(t, BattleUnderWay, strings.TrimSpace(out), name)
	}
	assert.Zero(t, fake.calls)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestLookShowsWhatARoomProvides(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())
	keywords.LoadAliases()
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "resbiome", Name: "Meadow", Symbol: "m", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("resbiome") })
	user := userWithItem(t, 19, skinSpec())

	look := func(room *rooms.Room) string {
		room.SetTestOccupants([]int{19}, nil)
		rooms.SetTestRoom(room)
		t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
		return tagPattern.ReplaceAllString(heard(t, func() {
			_, err := Look("", user, room, events.CmdSecretly)
			require.NoError(t, err)
		}), "")
	}
	spring := &rooms.Room{RoomId: 91101, Zone: "Meadow", Biome: "resbiome", Resources: []string{"shelter", "water", "herbs"}}
	assert.Contains(t, look(spring), "Here: fresh water, shelter, herbs.", "table order")

	bare := &rooms.Room{RoomId: 91102, Zone: "Meadow", Biome: "resbiome", Resources: []string{"herbs"}}
	rooms.SetDepletedCheck(func(roomID int, resource string) bool { return roomID == 91102 && resource == "herbs" })
	t.Cleanup(func() { rooms.SetDepletedCheck(nil) })
	assert.Contains(t, look(bare), "Here: herbs (picked clean).", "a picked-clean resource says so (40a2)")
	assert.NotContains(t, look(&rooms.Room{RoomId: 91103, Zone: "Meadow", Biome: "resbiome"}), "Here:")
}

func TestResourcesHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "resources" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists resources")

	want, err := GetHelpContents("resources")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(want, ""), "drink water")
	for _, alias := range []string{"water", "fill", "spring", "shelter", "water-source"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"drink", "survival", "forage", "camp", "company-meal", "webclient"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "resources", "help %s points at help resources", topic)
	}
}

// Phase 43a: the last glug leaves an empty skin, which a fill turns back
// into a full one.
func registerSkins(t *testing.T) (full, empty items.ItemSpec) {
	t.Helper()
	full = skinSpec()
	full.ItemId = 9301
	full.EmptyItemId = 9302
	empty = items.ItemSpec{ItemId: 9302, Name: "empty waterskin", NameSimple: "empty waterskin", Refillable: "water", FilledItemId: 9301}
	items.SetTestItemSpec(&full)
	items.SetTestItemSpec(&empty)
	t.Cleanup(func() { items.RemoveTestItemSpec(9301); items.RemoveTestItemSpec(9302) })
	return full, empty
}

func TestDrinkingTheLastGlugLeavesAnEmptyWaterskin(t *testing.T) {
	useFakeProvisioner(t, thirstyProvisioner())
	full, _ := registerSkins(t)
	user := userWithItem(t, 17, full)
	user.Character.Items[0].Uses = 1

	_, err := Drink("waterskin", user, testRoom(), 0)
	require.NoError(t, err)
	require.Len(t, user.Character.Items, 1, "the skin is not destroyed")
	assert.Equal(t, 9302, user.Character.Items[0].ItemId)
	assert.Zero(t, user.Character.Items[0].Uses)
}

func TestFillTurnsAnEmptyWaterskinIntoAFullOne(t *testing.T) {
	full, empty := registerSkins(t)
	user := userWithItem(t, 17, empty)
	user.Character.Items[0].Uses = 0
	_ = full

	assert.Equal(t, 5, func() int { n, _ := RefillableUses(empty); return n }(), "an empty skin fills to the full one's uses")
	out := heard(t, func() {
		_, err := Fill("", user, waterRoom(), 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "You fill")
	assert.Equal(t, 9301, user.Character.Items[0].ItemId)
	assert.Equal(t, 5, user.Character.Items[0].Uses)

	// Named, and refused away from water.
	user.Character.Items[0] = items.New(9302)
	out = heard(t, func() { _, _ = Fill("empty", user, testRoom(), 0) })
	assert.Contains(t, out, "no fresh water")
	assert.Equal(t, 9302, user.Character.Items[0].ItemId)
	heard(t, func() { _, _ = Fill("empty", user, waterRoom(), 0) })
	assert.Equal(t, 9301, user.Character.Items[0].ItemId)
}
