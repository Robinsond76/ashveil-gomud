package camping

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 23b test items: the whetstone (the default WhetstoneItemId, with
// its shipped 10 uses), two blades, and a club.
const (
	testStoneID  = 30
	testSwordID  = 99101
	testDaggerID = 99102
	testClubID   = 99103
)

func sharpenSpecs(t *testing.T) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testStoneID, Name: "whetstone", Type: items.Object, Subtype: items.Mundane, Uses: 10})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testSwordID, Name: "sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testDaggerID, Name: "dagger", Type: items.Weapon, Subtype: items.Stabbing, Hands: 1})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testClubID, Name: "club", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1})
	t.Cleanup(func() {
		for _, id := range []int{testStoneID, testSwordID, testDaggerID, testClubID} {
			items.RemoveTestItemSpec(id)
		}
	})
}

func stone(uses int) items.Item {
	s := items.New(testStoneID)
	s.Uses = uses
	return s
}

func armed(c *characters.Character, main, off int) {
	if main > 0 {
		c.Equipment.Weapon = items.New(main)
	}
	if off > 0 {
		c.Equipment.Offhand = items.New(off)
	}
}

func member(t *testing.T, name string, roomID, main, off int) *characters.Character {
	t.Helper()
	sharpenSpecs(t)
	c := characters.New()
	c.Name = name
	c.RoomId = roomID
	armed(c, main, off)
	return c
}

// sharpenFixture: a leader in room 100 and the given live companions.
func sharpenFixture(t *testing.T, live map[int]*characters.Character) (*CampingModule, *users.UserRecord, *fakeStore) {
	t.Helper()
	sharpenSpecs(t)
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	module.companionsOf = func(int) (map[int]*characters.Character, []int) {
		roster := []int{}
		for id := range live {
			roster = append(roster, id)
		}
		return live, roster
	}
	user := campUser(t, 7, 100)
	user.Character.Name = "Hero"
	return module, user, store
}

func run(t *testing.T, module *CampingModule, user *users.UserRecord, cmd string) string {
	t.Helper()
	messages := captureMessages(t)
	_, err := module.sharpenCommand(cmd, user, eligibleRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return strings.Join(*messages, "\n")
}

func stoneUses(c *characters.Character) []int {
	var uses []int
	for _, itm := range c.Items {
		if itm.ItemId == testStoneID {
			uses = append(uses, itm.Uses)
		}
	}
	return uses
}

func TestParseSharpenSettings(t *testing.T) {
	s := parseInnSettings(func(name string) any {
		return map[string]any{"WhetstoneItemId": 31, "SharpenedBonus": 2, "SharpenedStrikes": "30"}[name]
	})
	assert.Equal(t, 31, s.WhetstoneItemId)
	assert.Equal(t, 2, s.SharpenedBonus)
	assert.Equal(t, 30, s.SharpenedStrikes)
	bad := parseInnSettings(func(name string) any {
		return map[string]any{"WhetstoneItemId": 0, "SharpenedBonus": -1, "SharpenedStrikes": "many"}[name]
	})
	assert.Equal(t, defaultInnSettings(), bad)
}

func TestSharpenCommandSharpensCompanyOneUseEach(t *testing.T) {
	bran := member(t, "Bran", 100, testSwordID, 0)
	cara := member(t, "Cara", 100, testDaggerID, 0)
	module, user, _ := sharpenFixture(t, map[int]*characters.Character{1: bran, 2: cara})
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))

	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Hero, Bran, Cara.")
	assert.Contains(t, text, "That took 3 uses; 7 uses left.")
	assert.Equal(t, []int{7}, stoneUses(user.Character))
	for _, c := range []*characters.Character{user.Character, bran, cara} {
		assert.Equal(t, 1, c.Equipment.Weapon.SharpBonus, c.Name)
		assert.Equal(t, 20, c.Equipment.Weapon.SharpStrikes, c.Name)
	}
}

func TestSharpenTwoBladesOneUse(t *testing.T) {
	module, user, _ := sharpenFixture(t, nil)
	armed(user.Character, testSwordID, testDaggerID)
	user.Character.StoreItem(stone(10))

	text := run(t, module, user, "")
	assert.Contains(t, text, "That took 1 use; 9 uses left.")
	assert.True(t, user.Character.Equipment.Weapon.Sharpened())
	assert.True(t, user.Character.Equipment.Offhand.Sharpened())
}

func TestSharpenAlreadySharpAndBladelessFree(t *testing.T) {
	bran := member(t, "Bran", 100, testClubID, 0) // blunt: no blade
	cara := member(t, "Cara", 100, testSwordID, testDaggerID)
	cara.Equipment.Weapon.Sharpen(1, 5) // half sharp: only the dagger
	module, user, _ := sharpenFixture(t, map[int]*characters.Character{1: bran, 2: cara})
	armed(user.Character, testSwordID, 0)
	user.Character.Equipment.Weapon.Sharpen(1, 4)
	user.Character.StoreItem(stone(10))

	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Cara.")
	assert.Contains(t, text, "Already sharp: Hero.")
	assert.Contains(t, text, "That took 1 use; 9 uses left.")
	assert.Equal(t, 4, user.Character.Equipment.Weapon.SharpStrikes, "never reset")
	assert.Equal(t, 5, cara.Equipment.Weapon.SharpStrikes, "never reset")
	assert.Equal(t, 20, cara.Equipment.Offhand.SharpStrikes)
	assert.False(t, bran.Equipment.Weapon.Sharpened(), "a club takes no edge")

	again := run(t, module, user, "")
	assert.Contains(t, again, "No blade in your company needs sharpening.")
	assert.Equal(t, []int{9}, stoneUses(user.Character), "a repeat spends nothing")
}

func TestSharpenStoneRunsOutLeavesOut(t *testing.T) {
	live := map[int]*characters.Character{
		3: member(t, "Dag", 100, testSwordID, 0),
		1: member(t, "Bran", 100, testSwordID, 0),
		2: member(t, "Cara", 100, testSwordID, 0),
	}
	module, user, _ := sharpenFixture(t, live)
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(2))

	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Hero, Bran.")
	assert.Contains(t, text, "The whetstone ran out before Cara, Dag.")
	assert.Contains(t, text, "A whetstone is worn away.")
	assert.Empty(t, stoneUses(user.Character), "a spent stone leaves the pack")
	assert.False(t, live[2].Equipment.Weapon.Sharpened())
	assert.False(t, live[3].Equipment.Weapon.Sharpened())
}

func TestSharpenCarriesOntoSecondStone(t *testing.T) {
	live := map[int]*characters.Character{1: member(t, "Bran", 100, testSwordID, 0), 2: member(t, "Cara", 100, testSwordID, 0)}
	module, user, _ := sharpenFixture(t, live)
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))
	user.Character.StoreItem(stone(1)) // the most used goes first

	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Hero, Bran, Cara.")
	assert.Equal(t, []int{8}, stoneUses(user.Character))
	assert.Contains(t, text, "A whetstone is worn away.")
}

func TestSpentStoneQueuesOwnershipLoss(t *testing.T) {
	module, user, _ := sharpenFixture(t, nil)
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(1))
	lost := 0
	id := events.RegisterListener(events.ItemOwnership{}, func(e events.Event) events.ListenerReturn {
		if o := e.(events.ItemOwnership); o.UserId == 7 && !o.Gained && o.Item.ItemId == testStoneID {
			lost++
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.ItemOwnership{}, id) })
	run(t, module, user, "")
	assert.Equal(t, 1, lost)
	assert.Empty(t, user.Character.Items)
}

func TestSharpenWithoutStone(t *testing.T) {
	module, user, _ := sharpenFixture(t, nil)
	armed(user.Character, testSwordID, 0)
	assert.Contains(t, run(t, module, user, ""), "You have no whetstone.")
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
}

func TestSharpenRefusedInCombat(t *testing.T) {
	bran := member(t, "Bran", 100, testSwordID, 0)
	module, user, _ := sharpenFixture(t, map[int]*characters.Character{1: bran})
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))

	user.Character.SetAggro(0, 55, characters.DefaultAttack)
	assert.Contains(t, run(t, module, user, ""), "middle of a fight")
	user.Character.Aggro = nil
	bran.SetAggro(0, 55, characters.DefaultAttack)
	assert.Contains(t, run(t, module, user, ""), "middle of a fight")
	assert.Equal(t, []int{10}, stoneUses(user.Character))
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
	assert.False(t, bran.Equipment.Weapon.Sharpened())
	status := run(t, module, user, "status")
	assert.Contains(t, status, "Not while your company is fighting.")
	assert.Contains(t, status, "(dull)")
	assert.NotContains(t, status, "would be sharpened", "review fix: no contradiction mid-fight")
}

func TestSharpenAbsentCompanionNotHere(t *testing.T) {
	bran := member(t, "Bran", 555, testSwordID, 0)
	module, user, _ := sharpenFixture(t, map[int]*characters.Character{1: bran})
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))

	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Hero.")
	assert.Contains(t, text, "Not here: Bran.")
	assert.Equal(t, []int{9}, stoneUses(user.Character))
	assert.False(t, bran.Equipment.Weapon.Sharpened())
}

func TestSharpenStatusPreviewSpendsNothing(t *testing.T) {
	bran := member(t, "Bran", 100, testSwordID, 0)
	module, user, store := sharpenFixture(t, map[int]*characters.Character{1: bran})
	armed(user.Character, testSwordID, 0)
	user.Character.Equipment.Weapon.Sharpen(1, 6)
	user.Character.StoreItem(stone(3))

	text := run(t, module, user, "status")
	assert.Contains(t, text, "Whetstones: 1 (3 uses left). Auto-sharpen is off.")
	assert.Contains(t, text, "Hero: sword")
	assert.Contains(t, text, "(sharp: 6)")
	assert.Contains(t, text, "(already sharp)")
	assert.Contains(t, text, "Bran: sword (would be sharpened)")
	assert.Contains(t, text, "Sharpening now would use 1 whetstone use.")
	assert.Equal(t, []int{3}, stoneUses(user.Character))
	assert.False(t, bran.Equipment.Weapon.Sharpened())
	assert.Zero(t, store.saveCalls)
}

func TestAutoSharpenSettingDurable(t *testing.T) {
	module, user, store := sharpenFixture(t, nil)
	assert.Contains(t, run(t, module, user, "auto on"), "Auto-sharpen is on")
	assert.True(t, store.saved.AutoSharpen[7])

	reloaded := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	reloaded.load()
	assert.True(t, reloaded.autoSharpenOn(7))

	store.saveErr = errors.New("disk full")
	assert.Contains(t, run(t, module, user, "auto off"), "save failed")
	assert.True(t, module.autoSharpenOn(7), "a failed save keeps the old setting")
	store.saveErr = nil
	assert.Contains(t, run(t, module, user, "auto off"), "Auto-sharpen is off.")
	assert.False(t, store.saved.AutoSharpen[7])
	assert.Contains(t, run(t, module, user, "auto maybe"), sharpenUsage)
}

func TestLegacyRegistryWithoutAutoLoads(t *testing.T) {
	var registry Registry
	require.NoError(t, decodeRegistry([]byte("camps: {}\nrested_pending: {7: true}\n"), &registry))
	assert.NotNil(t, registry.AutoSharpen)
	assert.Empty(t, registry.AutoSharpen)
	require.NoError(t, decodeRegistry([]byte("camps: {}\nauto_sharpen: {7: true, 0: true, 9: false}\n"), &registry))
	assert.Equal(t, map[int]bool{7: true}, registry.AutoSharpen)
}

// TestCampSharpenAndStatusShareTheCommand: "camp sharpen" is "sharpen",
// and "camp status" adds the auto setting and preview.
func TestCampSharpenAndStatusShareTheCommand(t *testing.T) {
	module, user, _ := sharpenFixture(t, nil)
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))
	room := eligibleRoom()
	messages := captureMessages(t)
	for _, cmd := range []string{"", "status", "sharpen auto on", "sharpen"} {
		_, err := module.userCommand(cmd, user, room, 0)
		require.NoError(t, err)
	}
	events.ProcessEvents()
	joined := strings.Join(*messages, "\n")
	assert.Contains(t, joined, "Whetstones: 1 (10 uses left). Auto-sharpen is off.")
	assert.Contains(t, joined, "Auto-sharpen is on")
	assert.Contains(t, joined, "Sharpened: Hero.")
	assert.True(t, user.Character.Equipment.Weapon.Sharpened())
}

// TestShippedWhetstoneMatchesConfig: the configured whetstone is a
// shipped 10-use object, so a bought stone has the owner's 10 uses.
func TestShippedWhetstoneMatchesConfig(t *testing.T) {
	path := filepath.Join("..", "..", "_datafiles", "world", "default", "items", "other-0",
		fmt.Sprintf("%d-whetstone.yaml", defaultInnSettings().WhetstoneItemId))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var spec items.ItemSpec
	require.NoError(t, yaml.Unmarshal(data, &spec))
	assert.Equal(t, defaultInnSettings().WhetstoneItemId, spec.ItemId)
	assert.Equal(t, "whetstone", spec.Name)
	assert.Equal(t, 10, spec.Uses)
	assert.Equal(t, items.Object, spec.Type)
	assert.Equal(t, items.Mundane, spec.Subtype, "not usable: `use` would spend it for nothing")
}

// Review fix: a rostered companion with no live mob is named, not dropped.
func TestSharpenUnspawnedCompanionNamedNotHere(t *testing.T) {
	sharpenSpecs(t)
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	module.companionsOf = func(int) (map[int]*characters.Character, []int) { return nil, []int{2} }
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(2), Name: "Cara"},
	}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })
	user := campUser(t, 7, 100)
	user.Character.Name = "Hero"
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))

	assert.Contains(t, run(t, module, user, "status"), "Cara: not here")
	text := run(t, module, user, "")
	assert.Contains(t, text, "Sharpened: Hero.")
	assert.Contains(t, text, "Not here: Cara.")
	assert.Equal(t, []int{9}, stoneUses(user.Character))
}

// Review fix: a camp rest whose marker is set after the grant pass took
// its snapshot (an inn stay's Well Rested was pending alone) still
// auto-sharpens, because the clear itself reports it.
func TestAutoSharpenWhenCampRestFinishesAfterSnapshot(t *testing.T) {
	sharpenSpecs(t)
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	module.companionsOf = func(int) (map[int]*characters.Character, []int) { return nil, nil }
	module.grantBuff = func(*characters.Character, int, int) error { return nil }
	module.hasBuff = func(*characters.Character, int) bool { return false }
	user := campUser(t, 7, 100)
	user.Character.Name = "Hero"
	armed(user.Character, testSwordID, 0)
	user.Character.StoreItem(stone(10))
	module.wellRestedPending[7] = true
	module.autoSharpen[7] = true
	module.lookupUser = func(id int) *users.UserRecord {
		// The camp rest timer lands between the snapshot and the clear.
		module.mu.Lock()
		module.restedPending[7] = true
		module.mu.Unlock()
		return users.GetByUserId(id)
	}

	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.Equipment.Weapon.Sharpened())
	assert.Equal(t, []int{9}, stoneUses(user.Character))
	assert.False(t, module.restedPending[7])
}
