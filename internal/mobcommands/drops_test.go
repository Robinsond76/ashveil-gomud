package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dropWorld is a zone with a drop profile, a tier-1 weapon to roll, a
// goods table, and a room, users and foes to kill in it.
type dropWorld struct {
	room  *rooms.Room
	users []*users.UserRecord
	foes  []int
}

const (
	dropZone     = "DropZone"
	dropWeaponID = 98801
	dropGoodsID  = 98802
)

func newDropWorld(t *testing.T, nUsers int, band encounters.Band) *dropWorld {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	battle.Reset()
	t.Cleanup(battle.Reset)
	t.Cleanup(parties.UseMemoryForTest())
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = false
	gameplay.Death.CorpseItems = false
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	items.SetTestItemSpec(&items.ItemSpec{ItemId: dropWeaponID, Name: "drop sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1200, Tier: 1})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: dropGoodsID, Name: "drop pelt"})
	t.Cleanup(func() {
		items.RemoveTestItemSpec(dropWeaponID)
		items.RemoveTestItemSpec(dropGoodsID)
		loot.RemoveTestTable("drop-beast")
	})
	loot.SetTestTable(loot.Table{Category: "drop-beast", Entries: []loot.WeightedLootEntry{{ItemID: dropGoodsID, Weight: 1}}})
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: dropZone, Encounters: encounters.ZoneConfig{Band: band}}))

	w := &dropWorld{room: rooms.NewEmptyRoom()}
	w.room.Zone = dropZone
	for i := 0; i < nUsers; i++ {
		u := users.NewUserRecord(98800+i, 0)
		loot.TakeSpoils(u.UserId) // spoils an earlier test noted for this id
		t.Cleanup(func() { loot.TakeSpoils(u.UserId) })
		u.Character.RoomId = w.room.RoomId
		u.Character.Level = 1
		u.Character.Health = 10
		users.SetTestUser(u)
		w.users = append(w.users, u)
	}
	return w
}

// foe makes a foe of the room's group, damaged by every user and in each's battle.
func (w *dropWorld) foe(id int, mutate func(*mobs.Mob)) *mobs.Mob {
	m := &mobs.Mob{MobId: mobs.MobId(id), InstanceId: id, LootCategory: "drop-beast", EncounterID: "enc-1"}
	m.Character = *characters.New()
	m.Character.Name = "drop foe"
	m.Character.RoomId = w0(w)
	m.Character.Level = 6
	m.Character.TNLScale = 1
	m.Character.Gold = 0
	m.Character.PlayerDamage = map[int]int{}
	if mutate != nil {
		mutate(m)
	}
	mobs.SetTestInstance(m)
	w.room.AddMob(id)
	for _, u := range w.users {
		m.Character.PlayerDamage[u.UserId] = 5
	}
	w.foes = append(w.foes, id)
	for _, u := range w.users {
		battle.Begin(u.UserId, w.room.RoomId, 1, "g", w.foes) // every user fights every foe made so far
	}
	return m
}

func w0(w *dropWorld) int { return w.room.RoomId }

func TestZoneWithoutProfileDropsAsBefore(t *testing.T) {
	w := newDropWorld(t, 1, encounters.Band{})
	m := w.foe(98811, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(m.InstanceId) })
	_, err := Suicide("", m, w.room)
	require.NoError(t, err)
	require.Len(t, w.room.Corpses, 1)
	for _, it := range w.room.Corpses[0].Items {
		assert.NotEqual(t, dropWeaponID, it.ItemId, "no zone drop profile, no generated gear")
	}
	assert.Zero(t, w.room.Corpses[0].Gold, "and no cache gold")
	assert.Empty(t, loot.TakeSpoils(w.users[0].UserId))
}

func TestWonEncounterCacheRollsOnceForTheLastFoe(t *testing.T) {
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	a := w.foe(98821, nil)
	b := w.foe(98822, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(a.InstanceId); mobs.RemoveTestInstance(b.InstanceId) })
	uid := w.users[0].UserId

	_, err := Suicide("", a, w.room)
	require.NoError(t, err)
	require.Len(t, w.room.Corpses, 1)
	gold := w.room.Corpses[0].Gold
	assert.Zero(t, gold, "a foe that is not the last carries no cache gold")

	_, err = Suicide("", b, w.room)
	require.NoError(t, err)
	require.Len(t, w.room.Corpses, 2)
	last := w.room.Corpses[1]
	assert.Equal(t, uid, last.ClaimUserId)
	assert.GreaterOrEqual(t, last.Gold, 6*4, "the cache holds gold by level band")
	assert.NotEmpty(t, loot.TakeSpoils(uid), "the spoils line is noted for the summary")
	assert.Empty(t, loot.TakeSpoils(uid), "read once")
}

func TestOneRoundDoubleKillStillRollsTheCacheOnce(t *testing.T) {
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	a := w.foe(98831, nil)
	b := w.foe(98832, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(a.InstanceId); mobs.RemoveTestInstance(b.InstanceId) })
	a.Character.Health, b.Character.Health = 0, 0 // both fell in the same round
	_, err := Suicide("", a, w.room)
	require.NoError(t, err)
	_, err = Suicide("", b, w.room)
	require.NoError(t, err)
	withGold := 0
	for _, c := range w.room.Corpses {
		if c.Gold > 0 {
			withGold++
		}
	}
	assert.Equal(t, 1, withGold, "exactly one cache for the group")
}

func TestBossRollIsRareAndGuaranteed(t *testing.T) {
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	boss := w.foe(98841, func(m *mobs.Mob) { m.Boss, m.EncounterBoss, m.Character.Level = true, true, 7 })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)
	require.Len(t, w.room.Corpses, 1)
	var gear []items.Item
	for _, it := range w.room.Corpses[0].Items {
		if it.ItemId == dropWeaponID {
			gear = append(gear, it)
		}
	}
	require.GreaterOrEqual(t, len(gear), 2, "a boss drops two items (and the cache's chance of a third)")
	rare := false
	for _, it := range gear {
		if it.RollRarity().Rank() >= items.RarityRare.Rank() {
			rare = true
		}
	}
	assert.True(t, rare, "one of them is Rare or better")
	goods := 0
	for _, it := range w.room.Corpses[0].Items {
		if it.ItemId == dropGoodsID {
			goods++
		}
	}
	assert.GreaterOrEqual(t, goods, 3, "three goods rolls from the boss's table")
}

func TestAlliedCompaniesEachRollTheirOwn(t *testing.T) {
	w := newDropWorld(t, 2, encounters.Band{Low: 5, High: 7})
	p := parties.New(w.users[0].UserId)
	p.InvitePlayer(w.users[1].UserId)
	p.AcceptInvite(w.users[1].UserId)
	boss := w.foe(98851, func(m *mobs.Mob) { m.Boss, m.EncounterBoss, m.Character.Level = true, true, 7 })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)
	require.Len(t, w.room.Corpses, 2, "the claimant's corpse and a corpse for the other company")
	claims := map[int]bool{}
	for _, c := range w.room.Corpses {
		claims[c.ClaimUserId] = true
		var gear int
		for _, it := range c.Items {
			if it.ItemId == dropWeaponID {
				gear++
			}
		}
		assert.GreaterOrEqual(t, gear, 2, "each company has its own boss roll")
		assert.True(t, c.BattleSpoils)
	}
	assert.Len(t, claims, 2, "each company holds its own claim")
	for _, u := range w.users {
		assert.NotEmpty(t, loot.TakeSpoils(u.UserId))
	}
}

// Phase 38c2: a Pathfinder's Trailwise (rank 55) makes a won encounter's
// cache hold 15% more gold.
type trailClass struct {
	classes.PlayerProvider
	class string
}

func (p trailClass) PlayerClass(int) classes.State { return classes.State{Class: p.class} }

func TestTrailwiseFattensTheCacheGold(t *testing.T) {
	cacheGold := func(class string) (lowest int) {
		classes.SetProvider(trailClass{class: class})
		t.Cleanup(func() { classes.SetProvider(nil) })
		lowest = 1 << 30
		for i := 0; i < 40; i++ {
			w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
			w.users[0].Character.Level = 55
			m := w.foe(98830+i, nil)
			_, err := Suicide("", m, w.room)
			require.NoError(t, err)
			require.NotEmpty(t, w.room.Corpses)
			lowest = min(lowest, w.room.Corpses[0].Gold)
			mobs.RemoveTestInstance(m.InstanceId)
		}
		return lowest
	}
	assert.Less(t, cacheGold("scout"), 27, "level 6 caches hold 24 to 36 gold")
	assert.GreaterOrEqual(t, cacheGold("pathfinder"), 27, "and 15% more with Trailwise")
}

// Phase 36d: boss relics. A relic names its boss (the mob template id) and a
// chance per kill by one company.
const (
	relicBossMob   = 98861
	relicSetMob    = 98862
	relicWeaponID  = 98871
	relicLuckID    = 98872
	relicWeaponChn = 100
)

func relicSpecs(t *testing.T) {
	t.Helper()
	spec := &items.ItemSpec{ItemId: relicWeaponID, Name: "Test Reaper", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Tier: 6, Value: 500, Weight: 3000,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 12},
		Relic:  &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: relicBossMob, Chance: relicWeaponChn}}
	items.SetTestItemSpec(spec)
	// A rarer relic of another boss, for the bad-luck count.
	rare := *spec
	rare.ItemId, rare.Name = relicLuckID, "Test Rarity"
	rare.Relic = &items.RelicSpec{Signature: "Rarity", Effects: map[string]int{classes.Wounded: 10}, ILvl: 30, Mob: relicSetMob, Chance: 1}
	items.SetTestItemSpec(&rare)
	t.Cleanup(func() { items.RemoveTestItemSpec(relicWeaponID); items.RemoveTestItemSpec(relicLuckID) })
	prev := loot.Affixes()
	loot.SetAffixes(loot.AffixSet{})
	t.Cleanup(func() { loot.SetAffixes(prev) })
}

func relicInCorpse(w *dropWorld) *items.Item {
	for _, c := range w.room.Corpses {
		for i := range c.Items {
			if c.Items[i].ItemId == relicWeaponID {
				return &c.Items[i]
			}
		}
	}
	return nil
}

// A boss that has relics drops one through the real kill path, alongside
// its ordinary drops, and the spoils line names it.
func TestBossKillDropsItsRelic(t *testing.T) {
	relicSpecs(t)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	boss := w.foe(relicBossMob, func(m *mobs.Mob) { m.Boss, m.EncounterBoss, m.Character.Level = true, true, 7 })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)
	relic := relicInCorpse(w)
	require.NotNil(t, relic, "the boss dropped its relic")
	assert.Equal(t, items.RarityLegendary, relic.RollRarity())
	assert.Equal(t, 30, relic.Loot.ILvl)
	assert.Equal(t, "drop foe", relic.Loot.Source, "it names the boss it came from")
	var gear int
	for _, it := range w.room.Corpses[0].Items {
		if it.ItemId == dropWeaponID {
			gear++
		}
	}
	assert.GreaterOrEqual(t, gear, 2, "ordinary boss drops are unchanged")
	assert.Contains(t, strings.Join(loot.TakeSpoils(w.users[0].UserId), " "), "Test Reaper")
}

// Only a boss drops relics: the same template as an ordinary foe does not.
func TestOrdinaryFoeOfABossTemplateDropsNoRelic(t *testing.T) {
	relicSpecs(t)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	foe := w.foe(relicBossMob, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(foe.InstanceId) })
	_, err := Suicide("", foe, w.room)
	require.NoError(t, err)
	assert.Nil(t, relicInCorpse(w))
}

// A boss in a zone with no drop profile still drops its relic, and nothing
// else a profile would add.
func TestBossOutsideAnyDropProfileStillDropsItsRelic(t *testing.T) {
	relicSpecs(t)
	w := newDropWorld(t, 1, encounters.Band{})
	boss := w.foe(relicBossMob, func(m *mobs.Mob) { m.Boss, m.EncounterID = true, "" })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)
	require.NotNil(t, relicInCorpse(w))
	for _, it := range w.room.Corpses[0].Items {
		assert.NotEqual(t, dropWeaponID, it.ItemId, "no profile, no generated gear")
	}
}

// Each company that fought rolls its own relic (personal loot), and the
// bad-luck count is the leader's, saved on their character.
func TestRelicBadLuckCountIsKeptPerLeader(t *testing.T) {
	relicSpecs(t)
	w := newDropWorld(t, 2, encounters.Band{Low: 5, High: 7})
	boss := &mobs.Mob{MobId: relicSetMob}
	missSource := &scripted{vals: []int{99}}
	for kill := 1; kill < loot.BadLuckKills; kill++ {
		got, err := relicDrop(boss, w.users[0].UserId, "the boss", missSource)
		require.NoError(t, err)
		require.Empty(t, got, "kill %d is a miss", kill)
	}
	assert.Equal(t, loot.BadLuckKills-1, w.users[0].Character.GetMiscData(relicLuckKey(relicSetMob)))
	assert.Nil(t, w.users[1].Character.GetMiscData(relicLuckKey(relicSetMob)), "another company's count is its own")

	got, err := relicDrop(boss, w.users[0].UserId, "the boss", missSource)
	require.NoError(t, err)
	require.Len(t, got, 1, "the 20th kill is guaranteed a relic")
	assert.Equal(t, relicLuckID, got[0].ItemId)
	assert.Equal(t, 0, w.users[0].Character.GetMiscData(relicLuckKey(relicSetMob)), "a drop resets the count")

	// The count survives a save: a restart does not reset a leader's luck.
	w.users[1].Character.SetMiscData(relicLuckKey(relicSetMob), 7)
	data, err := yaml.Marshal(w.users[1].Character.MiscData)
	require.NoError(t, err)
	var back map[string]any
	require.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, 7, back[relicLuckKey(relicSetMob)], "it reads back as a number")
}

type scripted struct {
	vals []int
	i    int
}

func (s *scripted) Intn(n int) int { v := s.vals[s.i%len(s.vals)]; s.i++; return v % n }
