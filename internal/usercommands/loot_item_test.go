package usercommands

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lootMailID = 99701

func lootMail(t *testing.T) {
	t.Helper()
	spec := &items.ItemSpec{ItemId: lootMailID, Name: "test hauberk", NameSimple: "hauberk", Type: items.Body, Subtype: items.Wearable, Weight: 3000, DamageReduction: 6, Value: 100, Bulk: items.BulkLight}
	require.NoError(t, spec.Validate())
	items.SetTestItemSpec(spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(lootMailID) })
}

func rolledMail(rarity items.Rarity, ilvl int, identified bool) items.Item {
	itm := items.New(lootMailID)
	itm.ApplyRoll(items.Rolled{
		Version: items.RollVersion, Tier: 1, ILvl: ilvl, Quality: items.QualityFine, Rarity: rarity, Identified: identified,
		LevelReq: loot.LevelRequirement(rarity, ilvl), BaseValue: 100, Name: "Gloomward",
		Affixes: []items.RolledAffix{{ID: "strength", Label: "Mighty", Mechanic: "statmod:strength", Value: 3, Tier: 1, MinValue: 1, MaxValue: 3}},
	})
	return itm
}

// A rolled item's level requirement refuses through the real equip command,
// and a wearer who meets it gets the rolled numbers.
func TestEquipRefusesRolledGearBelowItsLevelRequirement(t *testing.T) {
	useWorld(t, "default")
	races.LoadDataFiles()
	lootMail(t)

	user := users.NewUserRecord(8, 2)
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Level = 9
	hauberk := rolledMail(items.RarityRare, 20, true) // requires level 15
	require.Equal(t, 15, hauberk.LevelRequirement())
	require.True(t, user.Character.StoreItem(hauberk))

	messages := captureLookMessages(t)
	_, err := Equip("test hauberk", user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	said := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "The test hauberk requires level 15 to use (the wearer is level 9).")
	assert.Zero(t, user.Character.Equipment.Body.ItemId, "nothing was worn")
	assert.Len(t, user.Character.Items, 1, "the item stays in the pack")

	user.Character.Level = 15
	*messages = nil
	_, err = Equip("test hauberk", user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	said = tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "You wear your")
	worn := user.Character.Equipment.Body
	assert.Equal(t, lootMailID, worn.ItemId)
	assert.Equal(t, 3, worn.GetSpec().StatMods.Get("strength"), "the worn item carries its identified affix")
	assert.Equal(t, 6+1, worn.GetSpec().DamageReduction, "a fine hauberk protects 1 more than a standard one")
}

func TestEquipUnidentifiedGearGivesBaseNumbersOnly(t *testing.T) {
	useWorld(t, "default")
	races.LoadDataFiles()
	lootMail(t)
	user := users.NewUserRecord(9, 3)
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Level = 20
	require.True(t, user.Character.StoreItem(rolledMail(items.RarityRare, 20, false)))

	messages := captureLookMessages(t)
	_, err := Equip("hauberk", user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	said := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "You wear your fine test hauberk")
	assert.Contains(t, said, "unidentified")
	worn := user.Character.Equipment.Body
	assert.False(t, worn.IsIdentified())
	assert.Zero(t, worn.GetSpec().StatMods.Get("strength"))
}

// The admin spawn loot command rolls through the real generator, drops the
// item in the room, and refuses bad words.
func TestSpawnLootRollsAnItemIntoTheRoom(t *testing.T) {
	useWorld(t, "default")
	lootMail(t)
	loot.LoadAffixDataFiles()
	t.Cleanup(func() { loot.SetAffixes(loot.AffixSet{}) })

	user := users.NewUserRecord(10, 4)
	user.Character.Name = "Admin"
	room := &rooms.Room{RoomId: 1}

	messages := captureLookMessages(t)
	handled, err := Spawn("loot hauberk 40 epic fine", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	said := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "fine test hauberk")
	assert.Contains(t, said, "unidentified")
	require.Len(t, room.Items, 1)
	got := room.Items[0]
	assert.Equal(t, items.RarityEpic, got.Loot.Rarity)
	assert.Equal(t, items.QualityFine, got.Loot.Quality)
	assert.Equal(t, 40, got.Loot.ILvl)
	assert.Equal(t, 35, got.LevelRequirement())
	assert.NotEmpty(t, got.Loot.Affixes)
	assert.False(t, got.IsIdentified(), "Rare and better drop unidentified")

	*messages = nil
	_, _ = Spawn("loot hauberk banana", user, room, 0)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "is not an item level, rarity or quality")
	_, _ = Spawn("loot", user, room, 0)
	_, _ = Spawn("loot nosuchthing", user, room, 0)
	assert.Len(t, room.Items, 1, "bad requests spawn nothing")
}

// look shows the layers, and a Scribe of rank 4 sees affix ranges.
func TestLookAtRolledItemShowsLayersAndScribeDetail(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())
	keywords.LoadAliases()
	skills.LoadDataFiles()
	lootMail(t)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(11, 5)
	users.SetTestUser(user)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "lootlit", Name: "Lit Hall", LitArea: true, Indoor: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("lootlit") })
	room := &rooms.Room{RoomId: 91011, Zone: "Test", Biome: "lootlit"}
	room.SetTestOccupants([]int{11}, nil)
	require.True(t, user.Character.StoreItem(rolledMail(items.RarityRare, 20, true)))

	messages := captureLookMessages(t)
	_, err := Look("hauberk", user, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	said := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "Rarity: Rare")
	assert.Contains(t, said, "Quality: fine")
	assert.Contains(t, said, "Requires level 15")
	assert.Contains(t, said, "+3 strength")
	assert.NotContains(t, said, "tier 1, 1 to 3")

	user.Character.Skills = map[string]int{"scribe": 4}
	*messages = nil
	_, err = Look("hauberk", user, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), ""), "(tier 1, 1 to 3)")
}

// gearup never proposes gear the wearer is too low for, and wearing by the
// item's uuid picks the exact instance among copies of one base item.
func TestGearupSkipsRefusedGearAndWearsTheExactInstance(t *testing.T) {
	useWorld(t, "default")
	races.LoadDataFiles()
	lootMail(t)

	user := users.NewUserRecord(12, 6)
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Level = 9
	tooHigh := rolledMail(items.RarityRare, 20, true) // level 15; worth more
	plain := items.New(lootMailID)
	require.True(t, user.Character.StoreItem(tooHigh))
	require.True(t, user.Character.StoreItem(plain))
	require.Greater(t, tooHigh.GetSpec().Value, plain.GetSpec().Value)

	best := user.Character.BestUpgrades()[items.Body]
	assert.Equal(t, plain.UUID, best.UUID, "the refused hauberk is not the upgrade")

	user.Character.Level = 15
	best = user.Character.BestUpgrades()[items.Body]
	assert.Equal(t, tooHigh.UUID, best.UUID, "once the level is met it is")

	messages := captureLookMessages(t)
	_, err := Equip(fmt.Sprintf("!%d:%s", best.ItemId, best.UUID), user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	_ = messages
	assert.Equal(t, tooHigh.UUID, user.Character.Equipment.Body.UUID, "the exact instance was worn")
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, plain.UUID, user.Character.Items[0].UUID)
}
