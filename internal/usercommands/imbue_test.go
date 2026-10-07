package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bruteHeart = 40002
	beastHide  = 40001
)

// enchantRoom is a shop room (a merchant who stocks a sword, so sales can
// be priced) with an enchanter in it, and the seller standing there with
// gold, a sword and a brute's heart.
func enchantRoom(t *testing.T) (*users.UserRecord, *rooms.Room, *mobs.Mob, *mobs.Mob) {
	t.Helper()
	seller, room, merchant := shopRoom(t, ironShortSword)
	useShippedItems(t, bruteHeart, beastHide)
	enchanter := &mobs.Mob{MobId: 988504, InstanceId: 988505, Character: *characters.New()}
	enchanter.Character.Name = "Orsolya"
	enchanter.Character.RoomId = room.RoomId
	enchanter.Character.Health = 20
	enchanter.Character.Gold = 0
	enchanter.Character.Adjectives = []string{"enchanter"}
	mobs.SetTestInstance(enchanter)
	t.Cleanup(func() { mobs.RemoveTestInstance(enchanter.InstanceId) })
	room.AddMob(enchanter.InstanceId)
	seller.Character.Gold = 500
	return seller, room, merchant, enchanter
}

func TestImbueNeedsAnEnchanter(t *testing.T) {
	seller, room, _ := shopRoom(t, ironShortSword)
	text := run(t, Imbue, "sword with heart", seller, room)
	assert.Contains(t, text, "no enchanter here")
	assert.Contains(t, text, "help enchanting")
}

// Through the real command: the fee is taken, the trophy is spent, the
// carried piece is enchanted and says so, and one trophy fills a piece.
func TestImbueWorksATrophyIntoACarriedPiece(t *testing.T) {
	seller, room, _, enchanter := enchantRoom(t)
	require.True(t, seller.Character.StoreItem(items.New(ironShortSword)))
	require.True(t, seller.Character.StoreItem(items.New(bruteHeart)))
	require.True(t, seller.Character.StoreItem(items.New(beastHide)))
	assert.True(t, enchanter.IsEnchanter())

	menu := run(t, Imbue, "", seller, room)
	assert.Contains(t, menu, "Trophies you carry")
	assert.Contains(t, menu, "brute's heart: +1 damage on every landed blow")

	sword := items.New(ironShortSword)
	fee := sword.EnchantFee()
	require.Equal(t, items.EnchantFeePerTier*max(1, sword.GetSpec().Tier), fee)

	text := run(t, Imbue, "shortsword with heart", seller, room)
	assert.Contains(t, text, "works your brute's heart into your")
	assert.Contains(t, text, "+1 damage on every landed blow")
	assert.Equal(t, 500-fee, seller.Character.Gold)
	assert.Equal(t, fee, enchanter.Character.Gold, "the enchanter is paid")
	assert.False(t, holds(seller, bruteHeart), "the trophy is spent")
	require.True(t, holds(seller, beastHide), "the other trophy is kept")
	assert.True(t, holds(seller, ironShortSword))
	got, ok := seller.Character.FindInBackpack("shortsword")
	require.True(t, ok)
	assert.Equal(t, bruteHeart, got.Trophy)

	// A second trophy is refused, and nothing is spent.
	again := run(t, Imbue, "shortsword with hide", seller, room)
	_ = again
	assert.Equal(t, 500-fee, seller.Character.Gold)
	assert.True(t, holds(seller, beastHide))
}

func TestImbueWorksATrophyIntoAWornPiece(t *testing.T) {
	seller, room, _, _ := enchantRoom(t)
	races.LoadDataFiles()
	seller.Character.RaceId = 1
	sword := items.New(ironShortSword)
	_, worn, reason := seller.Character.Wear(sword)
	require.True(t, worn, reason)
	require.True(t, seller.Character.StoreItem(items.New(bruteHeart)))
	assert.Zero(t, seller.Character.ClassEffects().Int("damage"))

	text := run(t, Imbue, "shortsword with heart", seller, room)
	assert.Contains(t, text, "works your brute's heart")
	assert.Equal(t, bruteHeart, seller.Character.Equipment.Weapon.Trophy)
	assert.Equal(t, 1, seller.Character.ClassEffects().Int("damage"), "in force at once")
	assert.False(t, holds(seller, bruteHeart))
}

func TestImbueRefusals(t *testing.T) {
	seller, room, _, _ := enchantRoom(t)
	require.True(t, seller.Character.StoreItem(items.New(ironShortSword)))
	require.True(t, seller.Character.StoreItem(items.New(bruteHeart)))
	require.True(t, seller.Character.StoreItem(items.New(ironOre)))

	for name, args := range map[string]string{
		"no 'with'":        "shortsword heart",
		"no such item":     "greatsword with heart",
		"no such trophy":   "shortsword with moonstone",
		"not a trophy":     "shortsword with ore",
		"trophy as target": "heart with heart",
	} {
		before := seller.Character.Gold
		run(t, Imbue, args, seller, room)
		assert.Equal(t, before, seller.Character.Gold, name)
		assert.True(t, holds(seller, bruteHeart), name+": the trophy stays")
		got, _ := seller.Character.FindInBackpack("shortsword")
		assert.Zero(t, got.Trophy, name)
	}

	seller.Character.Gold = 10
	text := run(t, Imbue, "shortsword with heart", seller, room)
	_ = text
	assert.Equal(t, 10, seller.Character.Gold, "too poor: nothing is taken")
	assert.True(t, holds(seller, bruteHeart))
	got, _ := seller.Character.FindInBackpack("shortsword")
	assert.Zero(t, got.Trophy)
}

// The enchant never raises what a merchant pays: the merchant prices an
// enchanted sword exactly as the plain one, and buys it through the real
// sell command.
func TestAnEnchantedPieceSellsForNoMoreThanAPlainOne(t *testing.T) {
	seller, room, merchant, _ := enchantRoom(t)
	merchant.Character.Gold = 5000

	plain := items.New(ironShortSword)
	enchanted := items.New(ironShortSword)
	require.NoError(t, enchanted.EnchantWithTrophy(bruteHeart))
	price := merchant.GetSellPrice(plain)
	require.Positive(t, price)
	assert.Equal(t, price, merchant.GetSellPrice(enchanted))

	require.True(t, seller.Character.StoreItem(enchanted))
	text := run(t, Sell, "shortsword", seller, room)
	assert.Equal(t, price, goldFor(t, text), text)
}

// The command is a management action: refused during a battle.
func TestImbueIsRefusedInBattle(t *testing.T) {
	assert.True(t, actionpolicy.Management("imbue"))
	assert.False(t, strings.Contains(actionpolicy.BattleUnderWay, "imbue"))
}
