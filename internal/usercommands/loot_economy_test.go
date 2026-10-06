package usercommands

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36c: the loot economy through the real commands. Merchants buy
// rolled gear, `sell junk` and `mark` clear a pack, smiths salvage gear
// into materials, and merchants read unidentified gear for a fee.

var plainTags = regexp.MustCompile(`<[^>]*>`)

func plain(s string) string { return plainTags.ReplaceAllString(s, "") }

// shopRoom builds a room with a merchant who stocks the given items, and a
// seller standing in it.
func shopRoom(t *testing.T, stock ...int) (*users.UserRecord, *rooms.Room, *mobs.Mob) {
	t.Helper()
	useWorld(t, "default")
	keywords.LoadAliases()
	useShippedItems(t, ironShortSword, ironShortSpear, 212, 220, 221, 222, 223, 224, 225, 20, 20104, 20134, 20194, 20160, 200, 20300)
	setupCarry(t, map[int]int{7: 50000})
	room := rooms.NewEmptyRoom()
	room.RoomId = 988501
	seller := carrier(t, 7, "Dain", room)
	seller.Character.Gold = 0

	merchant := &mobs.Mob{MobId: 988502, InstanceId: 988503, Character: *characters.New()}
	merchant.Character.Name = "Ivar"
	merchant.Character.RoomId = room.RoomId
	for _, id := range stock {
		merchant.Character.Shop = append(merchant.Character.Shop, characters.ShopItem{ItemId: id, Quantity: 2, QuantityMax: 4})
	}
	mobs.SetTestInstance(merchant)
	t.Cleanup(func() { mobs.RemoveTestInstance(merchant.InstanceId) })
	room.AddMob(merchant.InstanceId)
	return seller, room, merchant
}

func rolledSword(rarity items.Rarity, q items.Quality, identified bool) items.Item {
	itm := items.New(ironShortSword)
	itm.ApplyRoll(items.Rolled{
		Version: items.RollVersion, Tier: 1, ILvl: 5, Quality: q, Rarity: rarity, Identified: identified, Name: "Gloomfang",
		Affixes: []items.RolledAffix{{ID: "keen", Label: "Keen", Mechanic: "statmod:damage", Value: 2, Tier: 1, MinValue: 1, MaxValue: 3}},
	})
	return itm
}

func run(t *testing.T, fn UserCommand, rest string, user *users.UserRecord, room *rooms.Room) string {
	t.Helper()
	return plain(captureUserText(t, func() {
		_, err := fn(rest, user, room, 0)
		require.NoError(t, err)
	}))
}

func goldFor(t *testing.T, text string) int {
	t.Helper()
	m := regexp.MustCompile(`for (\d+) gold`).FindStringSubmatch(text)
	require.NotNil(t, m, text)
	n, _ := strconv.Atoi(m[1])
	return n
}

// A merchant buys rolled gear through the real sell command, priced by its
// quality, and by its rarity at a discount while it is unidentified.
func TestSellRolledGearPricedByQualityAndRarity(t *testing.T) {
	seller, room, merchant := shopRoom(t, ironShortSword)

	plainSword := items.New(ironShortSword)
	fine := rolledSword(items.RarityCommon, items.QualityFine, true)
	fine.Loot.Affixes = nil
	fine.ApplyRoll(fine.Loot)
	crude := rolledSword(items.RarityCommon, items.QualityCrude, true)
	crude.Loot.Affixes = nil
	crude.ApplyRoll(crude.Loot)

	price := func(item items.Item) int { return merchant.GetSellPrice(item) }
	assert.Greater(t, price(fine), price(plainSword), "fine workmanship is worth more")
	assert.Less(t, price(crude), price(plainSword), "crude is worth less")

	read := rolledSword(items.RarityRare, items.QualityStandard, true)
	unread := rolledSword(items.RarityRare, items.QualityStandard, false)
	assert.Greater(t, price(read), price(unread), "unidentified gear sells at a discount to what it fetches once read")
	assert.Greater(t, price(unread), 0, "an unidentified Rare still sells")
	legendary := rolledSword(items.RarityLegendary, items.QualityStandard, false)
	assert.Greater(t, price(legendary), price(unread), "rarity raises an unread item's price")

	// The price never reveals the hidden affix: two unread Rares that differ
	// only in their affixes fetch the same.
	other := rolledSword(items.RarityRare, items.QualityStandard, false)
	other.Loot.Affixes = []items.RolledAffix{{ID: "swift", Label: "Swift", Mechanic: "statmod:speed", Value: 40, Tier: 4, MinValue: 30, MaxValue: 50}}
	other.ApplyRoll(other.Loot)
	assert.Equal(t, price(unread), price(other), "an unread item is priced by rarity, not affixes")

	// Through the real command: the item leaves the pack, the gold arrives,
	// and the merchant's stock of the base item grows (saturation).
	require.True(t, seller.Character.StoreItem(unread))
	want := price(unread)
	text := run(t, Sell, "shortsword", seller, room)
	assert.Contains(t, text, "You sell a ")
	assert.Equal(t, want, goldFor(t, text))
	assert.Equal(t, want, seller.Character.Gold)
	assert.Empty(t, seller.Character.Items, "the sword left the pack")

	// A second identical sale pays less: the merchant already holds more.
	require.True(t, seller.Character.StoreItem(rolledSword(items.RarityRare, items.QualityStandard, false)))
	assert.Less(t, price(seller.Character.Items[0]), want, "a stocked-up merchant pays less (saturation)")
}

// A blob-carrying item and a used-up item are still refused, and a hand-
// edited spec is too; only a roll's own override is sellable.
func TestSellStillRefusesSpecialItems(t *testing.T) {
	seller, room, merchant := shopRoom(t, ironShortSword)

	blob := rolledSword(items.RarityCommon, items.QualityFine, true)
	blob.SetBlob("data")
	assert.Zero(t, merchant.GetSellPrice(blob))

	edited := items.New(ironShortSword)
	spec := edited.GetSpec()
	spec.Value = 9999
	edited.Spec = &spec
	assert.True(t, edited.IsSpecialForSale(), "an item with a hand-edited override is not a roll")
	assert.Zero(t, merchant.GetSellPrice(edited))

	require.True(t, seller.Character.StoreItem(blob))
	run(t, Sell, "shortsword", seller, room)
	assert.Len(t, seller.Character.Items, 1)
	assert.Zero(t, seller.Character.Gold)
}

// offer quotes the same price sell pays.
func TestOfferQuotesRolledGear(t *testing.T) {
	seller, room, merchant := shopRoom(t, ironShortSword)
	item := rolledSword(items.RarityRare, items.QualityFine, true)
	require.True(t, seller.Character.StoreItem(item))
	want := merchant.GetSellPrice(item)
	require.Positive(t, want)
	// The merchant speaks through mob.Command, so check the price directly
	// and that offer does not take the item.
	_, err := Offer("shortsword", seller, room, 0)
	require.NoError(t, err)
	assert.Len(t, seller.Character.Items, 1)
}

// mark and sell junk: marked items and plain junk sell in one transaction;
// unmarked gear, quest items and items no merchant here wants stay.
func TestMarkAndSellJunk(t *testing.T) {
	seller, room, _ := shopRoom(t, ironShortSword, 20)
	sword := items.New(ironShortSword)
	spear := items.New(ironShortSpear)
	keeper := items.New(ironShortSword)
	robe := items.New(20104) // a robe: this smith buys weapons, not cloth
	scrap := items.New(20)   // the shipped junk item
	for _, it := range []items.Item{sword, spear, keeper, robe, scrap} {
		require.True(t, seller.Character.StoreItem(it))
	}

	assert.Contains(t, run(t, Mark, "", seller, room), "nothing marked as junk")
	assert.Contains(t, run(t, Mark, "quilted robe junk", seller, room), "You mark the")
	assert.Contains(t, run(t, Mark, "quilted robe junk", seller, room), "already marked")
	assert.Contains(t, run(t, Mark, "quilted robe keep", seller, room), "take the junk mark off")
	assert.Contains(t, run(t, Mark, "quilted robe keep", seller, room), "isn't marked")
	assert.Contains(t, run(t, Mark, "quilted robe", seller, room), "Mark it how?")
	assert.Contains(t, run(t, Mark, "dagger junk", seller, room), "You don't have that item.")

	run(t, Mark, "quilted robe junk", seller, room)
	// The sword and the spear: mark by uuid-qualified name is awkward, so
	// mark the first match of each name.
	run(t, Mark, "shortsword junk", seller, room)
	run(t, Mark, "spear junk", seller, room)
	listed := run(t, Mark, "", seller, room)
	assert.Contains(t, listed, "iron shortsword")
	assert.Contains(t, listed, "iron short spear")
	assert.Contains(t, listed, "quilted robe")

	marked := 0
	for _, it := range seller.Character.Items {
		if it.Junk {
			marked++
			assert.Contains(t, plain(it.AttrString()), "j", "a marked item shows its flag")
		}
	}
	assert.Equal(t, 3, marked)

	text := run(t, Sell, "junk", seller, room)
	assert.Contains(t, text, "junk item(s)")
	assert.Contains(t, text, "iron shortsword")
	assert.Contains(t, text, "No one here would buy: quilted robe")
	assert.Positive(t, seller.Character.Gold)

	left := map[int]int{}
	for _, it := range seller.Character.Items {
		left[it.ItemId]++
	}
	assert.Equal(t, 1, left[ironShortSword], "the unmarked sword stays")
	assert.Equal(t, 1, left[20104], "the robe no one here wants stays")
	assert.Zero(t, left[20], "plain junk-type items sell without a mark")
	assert.Zero(t, left[ironShortSpear], "the marked spear sold")

	// No merchant, no sale.
	empty := rooms.NewEmptyRoom()
	empty.RoomId = 988599
	assert.Contains(t, run(t, Sell, "junk", seller, empty), "no merchant here")
	assert.Contains(t, run(t, Sell, "junk", seller, room), "wants your junk")
}

// A smith breaks gear into materials by tier, with bonuses; a merchant who
// only buys hides is no smith; an overloaded company is refused.
func TestSalvageThroughTheRealCommand(t *testing.T) {
	seller, room, merchant := shopRoom(t, ironShortSword)
	require.True(t, merchant.IsSmith())

	assert.Contains(t, run(t, Salvage, "shortsword", seller, room), "You don't have that item.")

	rare := rolledSword(items.RarityRare, items.QualityStandard, false)
	require.True(t, seller.Character.StoreItem(rare))
	text := run(t, Salvage, "shortsword", seller, room)
	assert.Contains(t, text, "breaks your")
	assert.Contains(t, text, "scrap iron", "a tier 1 sword is scrap iron")
	count := 0
	for _, it := range seller.Character.Items {
		switch it.ItemId {
		case 212:
			count++
		case ironShortSword:
			t.Fatal("the sword should be gone")
		}
	}
	// Rare adds one piece to the weight-based single scrap iron.
	assert.Equal(t, 2, count)

	// A quest item is never salvaged.
	quest := items.New(ironShortSpear)
	spec := quest.GetSpec()
	spec.QuestToken = "1000-test"
	quest.Spec = &spec
	require.True(t, seller.Character.StoreItem(quest))
	assert.Contains(t, run(t, Salvage, "spear", seller, room), "Quest items cannot be salvaged!")

	// The merchant who stocks no weapons or armor is no smith.
	trader := &mobs.Mob{MobId: 988512, InstanceId: 988513, Character: *characters.New()}
	trader.Character.Name = "Brynja"
	trader.Character.Shop = characters.Shop{{ItemId: 212, Quantity: 2, QuantityMax: 4}}
	mobs.SetTestInstance(trader)
	t.Cleanup(func() { mobs.RemoveTestInstance(trader.InstanceId) })
	assert.False(t, trader.IsSmith())
	shop := rooms.NewEmptyRoom()
	shop.RoomId = 988598
	shop.AddMob(trader.InstanceId)
	seller.Character.RoomId = shop.RoomId
	require.True(t, seller.Character.StoreItem(rolledSword(items.RarityCommon, items.QualityStandard, true)))
	assert.Contains(t, run(t, Salvage, "shortsword", seller, shop), "There is no smith here")

	// A sold-in weapon does not make a trader a smith.
	trader.Character.Shop.StockItem(ironShortSword)
	assert.False(t, trader.IsSmith(), "temporary stock sold by players does not count")
}

// The salvaged materials must fit: too heavy a haul is refused before the
// item is destroyed.
func TestSalvageRefusedWhenMaterialsTooHeavy(t *testing.T) {
	seller, room, _ := shopRoom(t, ironShortSword)
	setupCarry(t, map[int]int{7: 1300}) // exactly the sword
	sword := items.New(ironShortSword)
	require.True(t, seller.Character.StoreItem(sword))
	// Scrap iron is 4 kg; the sword only 1.2 kg.
	text := run(t, Salvage, "shortsword", seller, room)
	assert.Contains(t, text, "too much for your company to carry")
	assert.Len(t, seller.Character.Items, 1)
	assert.Equal(t, ironShortSword, seller.Character.Items[0].ItemId, "the sword was not destroyed")
}

// A merchant reads unidentified gear for a fee that grows with rarity, and
// refuses when the seller cannot pay; plain appraisal stays 20 gold.
func TestAppraiseReadsUnidentifiedGearForAFee(t *testing.T) {
	seller, room, merchant := shopRoom(t, ironShortSword)
	_ = merchant
	rare := rolledSword(items.RarityRare, items.QualityFine, false)
	require.True(t, seller.Character.StoreItem(rare))

	seller.Character.Gold = 59
	text := run(t, Appraise, "shortsword", seller, room)
	assert.Equal(t, 59, seller.Character.Gold, "not enough gold, nothing taken")
	assert.False(t, seller.Character.Items[0].IsIdentified())
	_ = text

	seller.Character.Gold = 100
	text = run(t, Appraise, "shortsword", seller, room)
	assert.Equal(t, 40, seller.Character.Gold, "a Rare costs 60 gold to read")
	assert.True(t, seller.Character.Items[0].IsIdentified(), "the stored item is identified")
	assert.Contains(t, text, "properties are laid bare")
	assert.Contains(t, text, "DAMAGE:       +2", "the affix is shown")

	// Reading it again is a plain 20 gold appraisal.
	text = run(t, Appraise, "shortsword", seller, room)
	assert.Equal(t, 20, seller.Character.Gold)
	assert.NotContains(t, text, "laid bare")

	assert.Equal(t, 60, loot.IdentifyFee(items.RarityRare))
	assert.Equal(t, 150, loot.IdentifyFee(items.RarityEpic))
	assert.Equal(t, 400, loot.IdentifyFee(items.RarityLegendary))
	assert.Equal(t, 400, loot.IdentifyFee(items.RaritySet))
	assert.Less(t, loot.IdentifyFee(items.RarityRare), loot.IdentifyFee(items.RarityEpic))
}

// The new pages render through help and answer to their aliases, and the
// pages 36c made stale no longer say rolled gear can't be sold.
func TestLootEconomyHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "shops" && !topic.AdminOnly {
			indexed[topic.Command] = true
		}
	}
	for _, topic := range []string{"salvage", "mark", "sell", "appraise", "market"} {
		assert.True(t, indexed[topic], "help index lists %s under shops", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for", topic)
	}
	for alias, topic := range map[string]string{"junk": "mark", "marking": "mark", "salvaging": "salvage", "trophies": "goods"} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	for _, topic := range []string{"identify", "goods", "sell", "market", "salvage", "mark", "loot", "appraise"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plainText := tagPattern.ReplaceAllString(text, "")
		assert.NotContains(t, plainText, "can't be sold", topic)
		assert.NotContains(t, plainText, "arrive with the market update", topic)
		assert.NotContains(t, plainText, "not dropped by foes yet", topic)
	}
	identify, _ := GetHelpContents("identify")
	assert.Contains(t, identify, "appraise [item]")
	for _, topic := range []string{"loot", "goods"} {
		text, _ := GetHelpContents(topic)
		for _, want := range []string{"help salvage", "help mark"} {
			assert.True(t, strings.Contains(text, want), "help %s points at %s", topic, want)
		}
	}
}
