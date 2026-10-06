package usercommands

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shippedItems reads the shipped item files without touching the global
// item table.
func shippedItems(t *testing.T) map[int]*items.ItemSpec {
	t.Helper()
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "items")
	specs, err := fileloader.LoadAllFlatFiles[int, *items.ItemSpec](dir)
	require.NoError(t, err)
	return specs
}

// useShippedItems registers the named shipped items for the test.
func useShippedItems(t *testing.T, ids ...int) {
	t.Helper()
	specs := shippedItems(t)
	for _, id := range ids {
		spec, ok := specs[id]
		require.True(t, ok, "shipped item %d", id)
		items.SetTestItemSpec(spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

const (
	militiaGlaive  = 10151
	ironShortSword = 10101
	ironShortSpear = 10131
	shortbow       = 10171
	kiteShield     = 20304
	bearHide       = 200
	ironOre        = 220
)

func wearer(t *testing.T, id int) *users.UserRecord {
	t.Helper()
	races.LoadDataFiles()
	user := users.NewUserRecord(id, uint64(id))
	user.Character.Name = "Wren"
	user.Character.RaceId = 1
	user.Character.Level = 5
	return user
}

func equipNamed(t *testing.T, user *users.UserRecord, what string) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Equip(what, user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
}

// Phase 36b: the shipped catalog's glaive, spear and bow behave as the
// equipment design says through the real equip command and the reach the
// formation uses: a glaive is two-handed and reaches, a short spear takes a
// shield and does not, a bow shoots from anywhere, and a shield displaces
// the glaive.
func TestCatalogWeaponsEquipWithTheirHandsAndReach(t *testing.T) {
	useWorld(t, "default")
	useShippedItems(t, militiaGlaive, ironShortSpear, shortbow, kiteShield)

	user := wearer(t, 31)
	char := user.Character
	require.True(t, char.StoreItem(items.New(kiteShield)))
	require.True(t, char.StoreItem(items.New(ironShortSpear)))
	require.True(t, char.StoreItem(items.New(militiaGlaive)))
	require.True(t, char.StoreItem(items.New(shortbow)))

	// A short spear leaves the off hand free: shield and spear together.
	assert.Contains(t, equipNamed(t, user, "iron short spear"), "You wield your")
	assert.Contains(t, equipNamed(t, user, "kite shield"), "You wear your")
	assert.Equal(t, ironShortSpear, char.Equipment.Weapon.ItemId)
	assert.Equal(t, kiteShield, char.Equipment.Offhand.ItemId)
	assert.Equal(t, formationcombat.ReachNone, combat.ResolveReach(char, false), "a one-handed spear has no reach")

	// The glaive takes both hands: the shield goes back to the pack, and the
	// wielder reaches a rank deeper.
	equipNamed(t, user, "glaive")
	assert.Equal(t, militiaGlaive, char.Equipment.Weapon.ItemId)
	assert.Zero(t, char.Equipment.Offhand.ItemId, "a two-handed glaive displaces the shield")
	assert.Equal(t, formationcombat.ReachExtended, combat.ResolveReach(char, false))
	assert.NotNil(t, itemIn(char, kiteShield), "the shield returned to the pack")

	// A shield displaces the glaive again.
	equipNamed(t, user, "kite shield")
	assert.Equal(t, kiteShield, char.Equipment.Offhand.ItemId)
	assert.NotEqual(t, militiaGlaive, char.Equipment.Weapon.ItemId, "a shield displaces a two-handed weapon")
	assert.NotNil(t, itemIn(char, militiaGlaive))

	// A bow shoots from any rank.
	equipNamed(t, user, "shortbow")
	assert.Equal(t, shortbow, char.Equipment.Weapon.ItemId)
	assert.Equal(t, formationcombat.ReachAny, combat.ResolveReach(char, false))
}

func itemIn(c *characters.Character, id int) *items.Item {
	for i := range c.Items {
		if c.Items[i].ItemId == id {
			return &c.Items[i]
		}
	}
	return nil
}

// look shows a catalog item's tier and family, and a good's category and
// what a kilogram of it is worth.
func TestLookShowsTierFamilyAndGoodsLines(t *testing.T) {
	useWorld(t, "default")
	useShippedItems(t, militiaGlaive, bearHide)

	glaive := items.New(militiaGlaive)
	assert.Contains(t, glaive.GetLongDescription(), "Tier 1 (Common) glaive. See help equipmenttiers.")
	assert.Contains(t, glaive.GetLongDescription(), "2-Handed weapon")

	hide := items.New(bearHide)
	text := hide.GetLongDescription()
	assert.Contains(t, text, "Trade good (trophy): worth about 7 gold a kg. See help goods.")
	assert.NotContains(t, text, "Tier")
}

// A general trader that already stocks a commodity buys the catalog's goods
// through the real merchant pricing; one that stocks only weapons does not.
func TestMerchantBuysCatalogGoods(t *testing.T) {
	useWorld(t, "default")
	useShippedItems(t, bearHide, ironOre, ironShortSword)

	trader := &mobs.Mob{Character: characters.Character{Shop: characters.Shop{{ItemId: ironOre, Quantity: 2, QuantityMax: 4}}}}
	assert.Positive(t, trader.GetSellPrice(items.New(bearHide)), "a trader that stocks a commodity buys another")

	armorer := &mobs.Mob{Character: characters.Character{Shop: characters.Shop{{ItemId: ironShortSword, Quantity: 2, QuantityMax: 3}}}}
	assert.Zero(t, armorer.GetSellPrice(items.New(bearHide)), "an armorer buys weapons, not hides")
}

// The catalog's help pages render, are indexed, are linked from their hubs,
// and state the numbers the shipped items carry.
func TestGearCatalogHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && !topic.AdminOnly {
			indexed[topic.Command] = true
		}
	}
	pointer := regexp.MustCompile(`help ([a-z-]+)`)
	for _, topic := range []string{"equipmenttiers", "goods"} {
		assert.True(t, indexed[topic], "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for "+topic)
		assert.NotContains(t, plain, "{{", topic)
		for _, m := range pointer.FindAllStringSubmatch(plain, -1) {
			_, err := GetHelpContents(m[1])
			assert.NoError(t, err, "help %s points at help %s", topic, m[1])
		}
	}

	for alias, topic := range map[string]string{
		"tier": "equipmenttiers", "tiers": "equipmenttiers", "glaive": "equipmenttiers", "equipment tiers": "equipmenttiers",
		"trade goods": "goods", "salvage": "goods", "value per kg": "goods",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	for hub, want := range map[string][]string{
		"equipment": {"help equipmenttiers", "help goods"},
		"armor":     {"help equipmenttiers"},
		"shields":   {"help equipmenttiers"},
		"loot":      {"help equipmenttiers", "help goods"},
		"itemlevel": {"help equipmenttiers"},
	} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		for _, w := range want {
			assert.Contains(t, text, w, "help %s mentions %s", hub, w)
		}
	}

	// The numbers the page states are the ones the items carry.
	specs := shippedItems(t)
	tiers, err := GetHelpContents("equipmenttiers")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(tiers, "")
	dice := func(id int) string { return specs[id].Damage.DiceRoll }
	for _, row := range []struct {
		label string
		ids   [3]int
	}{
		{"Sword", [3]int{10101, 10102, 10103}}, {"Axe", [3]int{10111, 10112, 10113}},
		{"Mace", [3]int{10121, 10122, 10123}}, {"Short spear", [3]int{10131, 10132, 10133}},
		{"War spear", [3]int{10141, 10142, 10143}}, {"Glaive", [3]int{10151, 10152, 10153}},
		{"Staff", [3]int{10021, 10162, 10163}}, {"Bow", [3]int{10171, 10172, 10173}},
		{"Crossbow", [3]int{10181, 10182, 10183}},
	} {
		want := fmt.Sprintf("%s %s / %s / %s", row.label, dice(row.ids[0]), dice(row.ids[1]), dice(row.ids[2]))
		compact := regexp.MustCompile(` {2,}`).ReplaceAllString(plain, " ")
		assert.Contains(t, compact, want, "help equipmenttiers states %s damage", row.label)
	}
	for path, base := range map[string]int{"Cloth": 20100, "Leather": 20130, "Medium": 20160, "Heavy": 20190} {
		var dr [3]int
		weight := 0
		for tier := 0; tier < 3; tier++ {
			for slot := 0; slot < 5; slot++ {
				spec := specs[base+slot*3+tier]
				require.NotNil(t, spec, "%s tier %d slot %d", path, tier+1, slot)
				dr[tier] += spec.DamageReduction
				if tier == 0 {
					weight += spec.Weight
				}
			}
		}
		want := fmt.Sprintf("%s %s %d / %d / %d", path, strings.ToLower(specs[base+3].Bulk), dr[0], dr[1], dr[2])
		compact := regexp.MustCompile(` {2,}`).ReplaceAllString(plain, " ")
		assert.Contains(t, compact, want, "help equipmenttiers states the %s set", path)
		assert.Contains(t, compact, fmt.Sprintf("%.1f kg", float64(weight)/1000), "and its weight")
	}
}
