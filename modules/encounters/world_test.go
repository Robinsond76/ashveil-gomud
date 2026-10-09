package encounters

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phases 41 and 42: the test-only world. A chain of tile-ready zones leads
// east from Alderbrook, one level band after another from 2 to 33 (a level
// 30 company can promote to an elite class), each with a rest point and
// the world's relic bosses in lairs. These tests read the shipped data and
// walk it through the real Go command.

// testWorldZones lists the chain in walking order, with each zone's band.
var testWorldZones = []struct {
	name      string
	low, high int
}{
	{"Alderbrook", 2, 4},
	{"Brindle Downs", 4, 6},
	{"Marrowmere Fen", 6, 8},
	{"Greywatch Pass", 9, 11},
	{"Cinder Hollow", 12, 14},
	{"Thornreach Wood", 15, 17},
	{"Hollowweb Deep", 18, 20},
	{"Ashen Barrows", 21, 24},
	{"Glassvault Depths", 25, 28},
	{"Stormcrown Heights", 29, 33},
}

func loadShippedWorld(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	mobs.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
}

// zoneRooms is every room of a zone, by id.
func zoneRooms(t *testing.T, zone string) map[int]*rooms.Room {
	t.Helper()
	cfg := rooms.GetZoneConfig(zone)
	require.NotNil(t, cfg, zone)
	out := map[int]*rooms.Room{}
	for id := range cfg.RoomIds {
		r := rooms.LoadRoom(id)
		require.NotNil(t, r, "%s room %d", zone, id)
		out[id] = r
	}
	return out
}

func hasTag(r *rooms.Room, tag string) bool {
	for _, t := range r.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// A zone's band matches the chain, its content is whole, and the chain has
// no gap between one band and the next.
func TestTestWorldBandsRunFromTheHamletToTheEliteLevels(t *testing.T) {
	loadShippedWorld(t)
	prevHigh := 0
	for _, z := range testWorldZones {
		cfg := rooms.GetZoneConfig(z.name)
		require.NotNil(t, cfg, z.name)
		valid, diags := cfg.Encounters.Validate(lookup)
		assert.Empty(t, diags, z.name)
		assert.Equal(t, z.low, valid.Band.Low, z.name)
		assert.Equal(t, z.high, valid.Band.High, z.name)
		for name, table := range cfg.Encounters.Tables {
			assert.Len(t, valid.Tables[name], len(table), "%s %s: nothing was disabled", z.name, name)
		}
		if prevHigh > 0 {
			assert.LessOrEqual(t, z.low, prevHigh+1, "%s starts where the last zone ends", z.name)
			assert.Greater(t, z.high, prevHigh, "%s climbs", z.name)
		}
		prevHigh = z.high
	}
	assert.GreaterOrEqual(t, prevHigh, 30, "the chain reaches the elite promotion level")
}

// The design's pacing rules: no more than a third of a zone's rooms
// spring encounters, a rest point (inn or camp) sits within twenty rooms of
// the entry, and a healer group is a fifth of a table at most (validation
// enforces the last, so only the shape is checked here).
func TestTestWorldZonesPaceEncountersAndPlaceRest(t *testing.T) {
	loadShippedWorld(t)
	for _, z := range testWorldZones[1:] {
		rs := zoneRooms(t, z.name)
		enabled, inns, camps := 0, 0, 0
		for _, r := range rs {
			if r.Encounter != nil && r.Encounter.Enabled {
				enabled++
			}
			if hasTag(r, "inn") {
				inns++
			}
			if hasTag(r, "camping") {
				camps++
			}
		}
		assert.LessOrEqual(t, enabled*3, len(rs), "%s: encounters on at most a third of its rooms", z.name)
		assert.Positive(t, enabled, "%s has eligible rooms", z.name)
		assert.Equal(t, 1, inns, "%s has an inn with a hearth", z.name)
		assert.Positive(t, camps, "%s has a campable clearing", z.name)
		cfg := rooms.GetZoneConfig(z.name)
		assert.True(t, cfg.TileReady, "%s is tile-ready", z.name)
		_, ok := cfg.RoomIds[cfg.RoomId]
		assert.True(t, ok, "%s's start room is one of its rooms", z.name)
	}
}

// Every room with an encounter names a table the zone defines, a lair is a
// boss table, and every relic boss the loot module knows lairs somewhere in
// the world so each relic can be earned.
func TestTestWorldPlacesEveryRelicBossInALair(t *testing.T) {
	loadShippedWorld(t)
	placed := map[int]bool{}
	for _, z := range testWorldZones {
		cfg := rooms.GetZoneConfig(z.name)
		for id, r := range zoneRooms(t, z.name) {
			if r.Encounter == nil || !r.Encounter.Enabled {
				continue
			}
			table, ok := cfg.Encounters.Tables[r.Encounter.Table]
			require.True(t, ok, "%s room %d names table %q", z.name, id, r.Encounter.Table)
			if r.Encounter.Table != "lair" {
				continue
			}
			for _, c := range table {
				assert.True(t, c.Boss, "%s lair composition %s", z.name, c.ID)
				placed[c.Members[0].MobID] = true
			}
			assert.GreaterOrEqual(t, r.Encounter.ChanceIn(cfg.Encounters), 20, "%s lair %d", z.name, id)
		}
	}
	// The pilot zones (Dark Forest, Catacombs) lair two bosses of their own.
	for _, zone := range []string{"Dark Forest", "Catacombs"} {
		cfg := rooms.GetZoneConfig(zone)
		require.NotNil(t, cfg)
		for _, c := range cfg.Encounters.Tables["lair"] {
			placed[c.Members[0].MobID] = true
		}
	}
	var missing []int
	for _, boss := range loot.RelicBosses() {
		if !placed[boss] {
			missing = append(missing, boss)
		}
	}
	sort.Ints(missing)
	assert.Empty(t, missing, "relic bosses with no lair")
	// The chain itself places all five bosses (not only the pilots).
	for _, boss := range []int{85, 34, 37, 14, 25} {
		assert.True(t, placed[boss], "boss %d", boss)
	}
}

// A relic needs its wearer to be the boss's intended level (36d: relic ilvl
// is boss level + 5, worn at ilvl - 5). Each chain lair puts its relic boss
// where the boss's own level is within two of that requirement, so a
// company earns a relic it can wear about when it wins it. Only the summit,
// the chain's top, holds bosses meant for higher bands (34-60, deferred to
// the replacement world).
func TestTestWorldLairsMatchTheirRelicLevels(t *testing.T) {
	loadShippedWorld(t)
	top := testWorldZones[len(testWorldZones)-1].name
	for _, z := range testWorldZones {
		cfg := rooms.GetZoneConfig(z.name)
		for _, c := range cfg.Encounters.Tables["lair"] {
			relics := loot.RelicsOf(c.Members[0].MobID)
			if len(relics) == 0 {
				continue
			}
			bossLevel := z.low + encounters.BossLevelBonus
			req := relics[0].Relic.ILvl - 5
			if z.name == top {
				assert.GreaterOrEqual(t, req, bossLevel-2, "%s %s: relic too low for the summit", z.name, c.ID)
				continue
			}
			assert.InDelta(t, req, bossLevel, 2, "%s %s: boss level %d, relic worn at %d", z.name, c.ID, bossLevel, req)
		}
	}
}

// The recipe pages from phase 56 spawn on the floor of landmark rooms and
// come back an hour after someone takes one, so every company can learn
// them (a plain floor item would be gone for good after the first pickup).
func TestTestWorldPlacesTheRecipePages(t *testing.T) {
	loadShippedWorld(t)
	found := map[int]*rooms.Room{}
	for _, z := range testWorldZones[1:] {
		for _, r := range zoneRooms(t, z.name) {
			for _, si := range r.SpawnInfo {
				if si.ItemId > 0 {
					found[si.ItemId] = r
				}
			}
		}
	}
	for _, page := range []int{30063, 30064} {
		r, ok := found[page]
		require.True(t, ok, "recipe page %d is placed", page)
		spec := items.GetItemSpec(page)
		require.NotNil(t, spec)
		assert.Equal(t, page, spec.ItemId)
		for _, si := range r.SpawnInfo {
			if si.ItemId == page {
				assert.Equal(t, "1 real hour", si.RespawnRate, "page %d respawns", page)
			}
		}
		r.Prepare(false)
		_, onFloor := r.FindOnFloor(fmt.Sprintf("!%d", page), false)
		assert.True(t, onFloor, "page %d lies on the floor of room %d", page, r.RoomId)
	}
}

// Every composition of the new zones builds: each template exists and a
// planned foe at the band's levels can be made, so a spawn cannot fail on
// content alone.
func TestTestWorldCompositionsBuild(t *testing.T) {
	loadShippedWorld(t)
	for _, z := range testWorldZones {
		cfg := rooms.GetZoneConfig(z.name)
		start := cfg.RoomId
		for name, table := range cfg.Encounters.Tables {
			for _, c := range table {
				for _, foe := range encounters.Plan(c, cfg.Encounters.Band, func(n int) int { return 0 }) {
					mob := mobs.NewMobById(mobs.MobId(foe.MobID), start, foe.Level)
					require.NotNil(t, mob, "%s %s %s: mob %d builds", z.name, name, c.ID, foe.MobID)
					assert.GreaterOrEqual(t, foe.Level, 1)
					mobs.DestroyInstance(mob.InstanceId)
				}
			}
		}
	}
}

// A company really walks the chain: from the hamlet's road through every
// zone's rooms to the last lair, one Go command per step, and each zone's
// entry is reached from the previous zone's exit.
func TestTestWorldCanBeWalkedFromTheHamletToTheSummit(t *testing.T) {
	loadShippedWorld(t)
	loadAliases(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(parties.UseMemoryForTest())
	walking.SuspendListeners(t)

	const hamletEast = 2125 // Alderbrook's Heather Slope: the chain starts through its east exit
	summit := 0
	for id, r := range zoneRooms(t, "Stormcrown Heights") {
		if r.Encounter != nil && r.Encounter.Table == "lair" {
			summit = id
		}
	}
	require.NotZero(t, summit)

	path := shortestPath(t, hamletEast, summit)
	require.NotEmpty(t, path, "the summit lair is reachable from Alderbrook")

	user := users.NewUserRecord(96801, 96801)
	user.Character.Name = "Walker"
	user.Character.RoomId = hamletEast
	user.Character.ActionPoints = 100000
	user.Character.Health = 50
	user.Character.Validate()
	users.SetTestUser(user)

	zonesSeen := []string{}
	for _, dir := range path {
		from := rooms.LoadRoom(user.Character.RoomId)
		handled, err := usercommands.Go(dir, user, from, 0)
		require.NoError(t, err)
		require.True(t, handled, "go %s from room %d", dir, from.RoomId)
		user.Character.ActionPoints = 100000
		to := rooms.LoadRoom(user.Character.RoomId)
		require.NotNil(t, to)
		if len(zonesSeen) == 0 || zonesSeen[len(zonesSeen)-1] != to.Zone {
			zonesSeen = append(zonesSeen, to.Zone)
		}
	}
	assert.Equal(t, summit, user.Character.RoomId)
	var want []string
	for _, z := range testWorldZones[1:] {
		want = append(want, z.name)
	}
	assert.Equal(t, want, zonesSeen, "every zone is passed through, in band order")
}

// shortestPath is the compass moves from one room to another over the
// shipped exits (a breadth-first search), or nil.
func shortestPath(t *testing.T, from, to int) []string {
	t.Helper()
	type step struct {
		room int
		via  string
		prev int
	}
	seen := map[int]step{from: {room: from, prev: -1}}
	queue := []int{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			var path []string
			for at := cur; at != from; at = seen[at].prev {
				path = append([]string{seen[at].via}, path...)
			}
			return path
		}
		room := rooms.LoadRoom(cur)
		if room == nil {
			continue
		}
		dirs := make([]string, 0, len(room.Exits))
		for dir := range room.Exits {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			ex := room.Exits[dir]
			if ex.Secret || ex.Lock.IsLocked() {
				continue
			}
			if _, ok := seen[ex.RoomId]; ok {
				continue
			}
			seen[ex.RoomId] = step{room: ex.RoomId, via: dir, prev: cur}
			queue = append(queue, ex.RoomId)
		}
	}
	return nil
}
