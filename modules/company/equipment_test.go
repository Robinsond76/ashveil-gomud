package company

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestPlayerShopRecoversSellerBeforeCreditingTreasury(t *testing.T) {
	b := equipmentBrawl(t)
	buyer := users.NewUserRecord(8, 2)
	buyer.Password = "$2a$test"
	buyer.Username, buyer.Character.Name, buyer.Character.RoomId, buyer.Character.Gold = "buyer", "Buyer", b.road.RoomId, 20
	users.SetTestUser(buyer)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	b.aria.Character.Shop = characters.Shop{{ItemId: 10004, Price: 5, Quantity: 10, QuantityMax: 10}}
	weapon := items.New(10004)
	b.aria.Character.Items = []items.Item{weapon}
	gold := b.aria.Character.Gold
	original := module.saveUser
	t.Cleanup(func() { module.saveUser = original })
	module.saveUser = func(u *users.UserRecord) error {
		if u.UserId == 7 {
			return errors.New("unavailable")
		}
		return original(u)
	}
	assert.Contains(t, b.cmd("company", "equip #1 "+weapon.ShorthandId()), "awaits recovery")
	_, err := usercommands.TryCommand("buy", "dagger from aria", 8, events.CmdSkipScripts)
	require.NoError(t, err)
	assert.Equal(t, 20, buyer.Character.Gold)
	assert.Empty(t, buyer.Character.Items)
	assert.Equal(t, 10, b.aria.Character.Shop[0].Quantity)
	module.saveUser = original
	_, err = usercommands.TryCommand("buy", "dagger from aria", 8, events.CmdSkipScripts)
	require.NoError(t, err)
	assert.Equal(t, 15, buyer.Character.Gold)
	assert.Len(t, buyer.Character.Items, 1)
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, gold+5, b.aria.Character.Gold)
}

func TestPurchaseRefusesTradeThatLosesSharedPackCapacity(t *testing.T) {
	b := equipmentBrawl(t)
	packSpec := items.ItemSpec{ItemId: 989741, Name: "trade pack", Weight: 500, CarryBonus: 5000}
	tokenSpec := items.ItemSpec{ItemId: 989742, Name: "trade token", Weight: 500}
	for _, s := range []items.ItemSpec{packSpec, tokenSpec} {
		spec := s
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
	pack := items.New(packSpec.ItemId)
	b.aria.Character.Items = []items.Item{pack, items.New(packSpec.ItemId)}
	load, tracked := encumbrance.CurrentLoad(7)
	require.True(t, tracked)
	ballast := items.ItemSpec{ItemId: 989743, Name: "ballast", Weight: load.CapacityGrams - load.TotalGrams() - 100}
	items.SetTestItemSpec(&ballast)
	t.Cleanup(func() { items.RemoveTestItemSpec(ballast.ItemId) })
	b.aria.Character.Items = append(b.aria.Character.Items, items.New(ballast.ItemId))
	merchant := mobs.NewMobById(9105, b.road.RoomId)
	require.NotNil(t, merchant)
	b.road.AddMob(merchant.InstanceId)
	merchant.Character.Shop = characters.Shop{{ItemId: tokenSpec.ItemId, TradeItemId: packSpec.ItemId, Quantity: 10, QuantityMax: 10, Price: 1}}
	b.aria.Character.Gold = 50
	assert.Contains(t, b.cmd("buy", "trade token"), "too much")
	assert.Equal(t, 50, b.aria.Character.Gold)
	assert.Len(t, b.aria.Character.Items, 3)
	assert.Equal(t, pack.UUID, b.aria.Character.Items[0].UUID)
	assert.Equal(t, 10, merchant.Character.Shop[0].Quantity)
}

func TestPendingAssetsGuardIncomingGiftsPickupAndAutomaticConsumption(t *testing.T) {
	for _, target := range []string{"aria", "tamsin"} {
		t.Run(target, func(t *testing.T) {
			b := equipmentBrawl(t)
			donor := users.NewUserRecord(8, 2)
			donor.Password = "$2a$test"
			donor.Username, donor.Character.Name, donor.Character.RoomId = "donor", "Donor", b.road.RoomId
			users.SetTestUser(donor)
			b.road.AddPlayer(8)
			t.Cleanup(func() { b.road.RemovePlayer(8) })
			gift, weapon, supply := items.New(10004), items.New(10004), items.New(30006)
			gift.Sharpen(1, 6)
			donor.Character.Items = []items.Item{gift}
			donor.Character.Gold = 20
			b.aria.Character.Items = []items.Item{weapon, supply}
			gold := b.aria.Character.Gold
			original := module.saveUser
			t.Cleanup(func() { module.saveUser = original })
			module.saveUser = func(u *users.UserRecord) error {
				if u.UserId == 7 {
					return errors.New("unavailable")
				}
				return original(u)
			}
			assert.Contains(t, b.cmd("company", "equip #1 "+weapon.ShorthandId()), "awaits recovery")
			_, err := usercommands.TryCommand("give", gift.ShorthandId()+" "+target, 8, events.CmdSkipScripts)
			require.NoError(t, err)
			_, err = usercommands.TryCommand("give", "5 gold "+target, 8, events.CmdSkipScripts)
			require.NoError(t, err)
			assert.Len(t, donor.Character.Items, 1)
			assert.Equal(t, 20, donor.Character.Gold)
			b.road.Items, b.road.Gold = []items.Item{gift}, 7
			_, err = mobcommands.Get("all", b.companion(1), b.road)
			require.NoError(t, err)
			assert.Len(t, b.road.Items, 1)
			assert.Equal(t, 7, b.road.Gold)
			require.Error(t, encumbrance.ConsumeCargoItemUse(7, supply))
			require.Error(t, encumbrance.DepositCargo(7, "test", []encumbrance.CargoStack{{ItemId: 30006, Count: 1}}))
			scripting.GetActor(7, 0).AddGold(3)
			assert.Equal(t, events.CancelAndRequeue, hooks.HandleQuestUpdate(events.Quest{UserId: 7, QuestToken: "test-end"}), "quest rewards wait for recovery")
			assert.Equal(t, gold, b.aria.Character.Gold)
			module.saveUser = original
			_, err = usercommands.TryCommand("give", gift.ShorthandId()+" "+target, 8, events.CmdSkipScripts)
			require.NoError(t, err)
			_, err = usercommands.TryCommand("give", "5 gold "+target, 8, events.CmdSkipScripts)
			require.NoError(t, err)
			assert.Empty(t, donor.Character.Items)
			assert.Equal(t, gold+5, b.aria.Character.Gold)
			require.NoError(t, encumbrance.ConsumeCargoItemUse(7, supply))
			require.NoError(t, module.PrepareAssets(7))
			found := false
			for _, itm := range b.aria.Character.Items {
				if itm.Equals(gift) {
					found = true
					assert.Equal(t, 6, itm.SharpStrikes)
				}
				if itm.Equals(supply) {
					assert.Equal(t, supply.Uses-1, itm.Uses)
				}
			}
			assert.True(t, found)
		})
	}
}

func TestCompanionEquipmentRefreshesLeaderGMCPInventory(t *testing.T) {
	b := equipmentBrawl(t)
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())
	weapon := items.New(10004)
	b.aria.Character.Items = []items.Item{weapon}
	var inventory *gmcp.GMCPCharModule_Payload_Inventory
	id := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.UserId == 7 && out.Module == "Char.Inventory" {
			inventory = out.Payload.(*gmcp.GMCPCharModule_Payload_Inventory)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, id) })
	require.Contains(t, b.cmd("company", "equip #1 "+weapon.ShorthandId()), "equipment updated")
	require.NotNil(t, inventory, "real companion transfer must update the leader's Gear dock")
	for _, cargo := range inventory.Backpack.Items {
		assert.NotEqual(t, weapon.ShorthandId(), cargo.Id)
	}
}

func TestAutoLootRoundUsesRealCommandAndOnlyOwnSpoils(t *testing.T) {
	b := equipmentBrawl(t)
	start := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(start) })
	util.SetRoundCount(500)
	foreign := rooms.Corpse{MobId: 1, ClaimUserId: 8, BattleSpoils: true, RoundCreated: 500, Character: *characters.New(), Gold: 13}
	foreign.Character.Name = "bandit"
	own := foreign
	own.ClaimUserId, own.Gold = 7, 9
	itm := items.New(10004)
	itm.Sharpen(2, 7)
	own.Items = []items.Item{itm}
	b.road.Corpses = []rooms.Corpse{foreign, own}
	util.SetRoundCount(gametime.GetDate(500).AddPeriod("2 hours"))
	var inputs []string
	listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		in := e.(events.Input)
		if in.UserId == 7 {
			inputs = append(inputs, in.InputText)
			c, rest, _ := strings.Cut(in.InputText, " ")
			_, err := usercommands.TryCommand(c, rest, 7, events.CmdSkipScripts)
			require.NoError(t, err)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, listener) })
	round := util.GetRoundCount()
	module.onNewRound(events.NewRound{RoundNumber: round})
	events.ProcessEvents()
	assert.Empty(t, inputs, "default is opt-out")
	b.cmd("autoloot", "on")
	b.aria.Character.SetAggro(0, b.bandits["bandit cutthroat"][0], characters.DefaultAttack)
	module.onNewRound(events.NewRound{RoundNumber: round})
	events.ProcessEvents()
	assert.Empty(t, inputs, "battle blocks automatic loot")
	b.aria.Character.Aggro = nil
	gold := b.aria.Character.Gold
	module.onNewRound(events.NewRound{RoundNumber: round})
	events.ProcessEvents()
	assert.Equal(t, []string{"loot own"}, inputs)
	assert.Equal(t, gold+9, b.aria.Character.Gold)
	assert.Equal(t, 13, b.road.Corpses[0].Gold, "another company's public loot stays untouched")
	assert.Zero(t, b.road.Corpses[1].Gold)
	found := false
	for _, cargo := range b.aria.Character.Items {
		if cargo.Equals(itm) {
			found = true
			assert.Equal(t, 7, cargo.SharpStrikes)
		}
	}
	assert.True(t, found)
	assert.Equal(t, round, util.GetRoundCount())
	module.onNewRound(events.NewRound{RoundNumber: round})
	events.ProcessEvents()
	assert.Len(t, inputs, 1, "empty or foreign spoils do not queue another command")
}

func equipmentBrawl(t *testing.T) *brawl {
	b := newBrawl(t)
	b.aria.Character.CompanyCargo = true
	b.aria.Character.CargoMigrated = true
	return b
}

func TestEquipmentCommandsPreserveExactInstancesAndBoundGear(t *testing.T) {
	b := equipmentBrawl(t)
	a := items.New(10004)
	a.Sharpen(2, 9)
	z := items.New(10004)
	z.Enchantments = 2
	b.aria.Character.Items = []items.Item{a, z}
	beforeRound := util.GetRoundCount()
	beforeWorn := b.companion(1).Character.Equipment.Weapon
	assert.Contains(t, b.cmd("company", "equip #1 dagger"), "Several")
	assert.Contains(t, b.cmd("company", "compare #1 "+a.ShorthandId()), "Protection:")
	require.Contains(t, b.cmd("company", "equip #1 "+a.ShorthandId()), "equipment updated")
	worn := b.companion(1).Character.Equipment.Weapon
	assert.Equal(t, a.UUID, worn.UUID)
	assert.Equal(t, 9, worn.SharpStrikes)
	assert.Equal(t, beforeRound, util.GetRoundCount())
	foundOld, foundDuplicate := false, false
	for _, itm := range b.aria.Character.Items {
		if itm.Equals(beforeWorn) {
			foundOld = true
		}
		if itm.Equals(z) {
			foundDuplicate = true
			assert.Equal(t, uint8(2), itm.Enchantments)
		}
	}
	assert.True(t, foundOld)
	assert.True(t, foundDuplicate)
	require.Contains(t, b.cmd("company", "remove #1 weapon"), "equipment updated")
	assert.Zero(t, b.companion(1).Character.Equipment.Weapon.ItemId)
	require.Contains(t, b.cmd("equip", a.ShorthandId()), "equipment updated")
	assert.Equal(t, a.UUID, b.aria.Character.Equipment.Weapon.UUID)
	assert.Contains(t, b.cmd("remove", "!10004:stale-reference"), "exact worn")
	assert.Equal(t, a.UUID, b.aria.Character.Equipment.Weapon.UUID)
	assert.Contains(t, b.cmd("company", "equip #1 !10004:not-a-uuid"), "exact cargo")
	// Binding and curses apply to both explicit and legacy routes.
	b.aria.Character.Equipment.Weapon.CanNeverBeRemoved = true
	assert.Contains(t, b.cmd("remove", a.ShorthandId()), "bound")
	assert.Equal(t, a.UUID, b.aria.Character.Equipment.Weapon.UUID)
	assert.Contains(t, b.cmd("gearup", ""), "deliberately")
}

func TestEquipmentRejectsForeignAbsentDeadAndBattle(t *testing.T) {
	b := equipmentBrawl(t)
	itm := items.New(10004)
	b.aria.Character.Items = []items.Item{itm}
	assert.Contains(t, b.cmd("company", "equip #999 "+itm.ShorthandId()), "no company member")
	c := b.companion(1)
	c.Character.RoomId = 920102
	assert.Contains(t, b.cmd("company", "equip #1 "+itm.ShorthandId()), "alive and present")
	c.Character.RoomId = b.aria.Character.RoomId
	c.Character.Health = 0
	assert.Contains(t, b.cmd("company", "equip #1 "+itm.ShorthandId()), "alive and present")
	c.Character.Health = 100
	other := users.NewUserRecord(8, 1)
	other.Character.RoomId = c.Character.RoomId
	users.SetTestUser(other)
	c.Character.Charmed.UserId = 8
	assert.Contains(t, b.cmd("company", "equip #1 "+itm.ShorthandId()), "alive and present")
	c.Character.Charmed.UserId = 7
	b.aria.Character.SetAggro(0, b.bandits["bandit cutthroat"][0], characters.DefaultAttack)
	assert.Contains(t, b.cmd("company", "equip #1 "+itm.ShorthandId()), "battle is under way")
	assert.Equal(t, itm.UUID, b.aria.Character.Items[0].UUID)
}

func TestEquipmentTwoHandsAndCursedDisplacement(t *testing.T) {
	b := equipmentBrawl(t)
	spec := items.ItemSpec{ItemId: 989510, Name: "test glaive", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Reach: true, Weight: 2200}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	glaive := items.New(spec.ItemId)
	shield := items.New(20001)
	shield.Spec = &items.ItemSpec{ItemId: 20001, Type: items.Offhand, Subtype: items.Wearable, Hands: 1, Cursed: true}
	b.aria.Character.Equipment.Offhand = shield
	b.aria.Character.Items = []items.Item{glaive}
	assert.Contains(t, b.cmd("equip", glaive.ShorthandId()), "cursed")
	assert.Equal(t, shield.UUID, b.aria.Character.Equipment.Offhand.UUID)
	shield.Uncursed = true
	b.aria.Character.Equipment.Offhand = shield
	require.Contains(t, b.cmd("equip", glaive.ShorthandId()), "equipment updated")
	assert.Zero(t, b.aria.Character.Equipment.Offhand.ItemId)
	assert.Equal(t, glaive.UUID, b.aria.Character.Equipment.Weapon.UUID)
	found := false
	for _, itm := range b.aria.Character.Items {
		if itm.Equals(shield) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestAssetOperationRecoversUserSaveFailureAndPoolsExactlyOnce(t *testing.T) {
	b := equipmentBrawl(t)
	c := b.companion(1)
	gift := items.New(10004)
	gift.Sharpen(1, 4)
	c.Character.Items = []items.Item{gift}
	c.Character.Gold = 31
	oldGold := b.aria.Character.Gold
	oldSave := module.saveUser
	t.Cleanup(func() { module.saveUser = oldSave })
	module.saveUser = func(*users.UserRecord) error { return errors.New("disk unavailable") }
	require.Error(t, module.PrepareAssets(7))
	rec, _ := module.registry.Get(7)
	require.NotNil(t, rec.AssetOperation)
	assert.Equal(t, oldGold, b.aria.Character.Gold)
	assert.Empty(t, c.Character.Items)
	assert.Zero(t, c.Character.Gold)
	// Reload the write-ahead record, as a process restart would.
	reg := domain.NewRegistry()
	require.NoError(t, module.store.Load(reg))
	module.registry = *reg
	module.saveUser = oldSave
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, oldGold+31, b.aria.Character.Gold)
	n := 0
	for _, itm := range b.aria.Character.Items {
		if itm.Equals(gift) {
			n++
			assert.Equal(t, 4, itm.SharpStrikes)
		}
	}
	assert.Equal(t, 1, n)
	require.NoError(t, module.PrepareAssets(7))
	assert.Equal(t, oldGold+31, b.aria.Character.Gold)
	// The exact item reference and marker survive a real user YAML round trip.
	saved, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, b.aria.Character.CompanyAssetOp, saved.Character.CompanyAssetOp)
	found := false
	for _, itm := range saved.Character.Items {
		if itm.Equals(gift) {
			found = true
		}
	}
	assert.True(t, found)
}

// failingAssetStore allows failure at the journal or at its acknowledgement.
type failingAssetStore struct {
	Store
	saves, failAt int
}

func (s *failingAssetStore) Save(r domain.Registry) error {
	s.saves++
	if s.saves == s.failAt {
		return errors.New("store failure")
	}
	return s.Store.Save(r)
}

func TestEquipmentCompanySaveFailureAndAcknowledgementRetry(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(strings.Repeat("x", failAt), func(t *testing.T) {
			b := equipmentBrawl(t)
			itm := items.New(10004)
			b.aria.Character.Items = []items.Item{itm}
			old := b.companion(1).Character.Equipment.Weapon
			original := module.store
			store := &failingAssetStore{Store: original, failAt: failAt}
			module.store = store
			t.Cleanup(func() { module.store = original })
			text := b.cmd("company", "equip #1 "+itm.ShorthandId())
			assert.Contains(t, text, "failure")
			if failAt == 1 {
				assert.Equal(t, old.UUID, b.companion(1).Character.Equipment.Weapon.UUID)
				assert.Equal(t, itm.UUID, b.aria.Character.Items[0].UUID)
			} else {
				assert.Equal(t, itm.UUID, b.companion(1).Character.Equipment.Weapon.UUID)
				// Change cargo after the durable user acknowledgement: retry must not restore
				// its old snapshot (an already applied operation is an acknowledgement only).
				b.aria.Character.Items = nil
				require.NoError(t, module.PrepareAssets(7))
				assert.Empty(t, b.aria.Character.Items)
			}
		})
	}
}

func TestAssetOperationAndItemUUIDDecode(t *testing.T) {
	item := items.New(10004)
	rec := domain.Record{LeaderUserID: 7, AssetOperation: &domain.AssetOperation{ID: "op", Items: []items.Item{item}, Gold: 42}}
	reg := domain.NewRegistry()
	reg.Put(rec)
	raw, err := yaml.Marshal(reg)
	require.NoError(t, err)
	var loaded domain.Registry
	require.NoError(t, decodeCompanies(raw, &loaded))
	got, ok := loaded.Get(7)
	require.True(t, ok)
	require.NotNil(t, got.AssetOperation)
	assert.Equal(t, item.UUID, got.AssetOperation.Items[0].UUID)
	clone := got.AssetOperation.Items[0]
	assert.True(t, item.Equals(clone))
}

func TestEquipmentSnapshotKeepsOtherMembersEdgesAndRestoresGear(t *testing.T) {
	b := equipmentBrawl(t)
	other := b.companion(2)
	other.Character.Equipment.Weapon.Sharpen(2, 4)
	target := items.New(10004)
	b.aria.Character.Items = []items.Item{target}
	require.Contains(t, b.cmd("company", "equip #1 "+target.ShorthandId()), "equipment updated")
	assert.Equal(t, 4, other.Character.Equipment.Weapon.SharpStrikes, "assigning one member does not reset another's edge")
	module.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	require.NoError(t, module.restoreForLeader(7, b.aria.Character.RoomId))
	assert.Equal(t, target.UUID, b.companion(1).Character.Equipment.Weapon.UUID, "restored companion keeps its exact item identity")
	assert.Equal(t, 4, b.companion(2).Character.Equipment.Weapon.SharpStrikes)
}
