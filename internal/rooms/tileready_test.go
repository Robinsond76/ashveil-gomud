package rooms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func tileRoom(id, x, y, z int, desc string, exits map[string]int) *Room {
	r := &Room{RoomId: id, Title: "T", Description: desc, Biome: "land", HasCoordinates: true, MapX: x, MapY: y, MapZ: z,
		Exits: map[string]exit.RoomExit{}}
	for name, to := range exits {
		r.Exits[name] = exit.RoomExit{RoomId: to}
	}
	return r
}

func allLegends(string) bool { return true }

func TestValidateTileReadyAcceptsAGoodZone(t *testing.T) {
	list := []*Room{
		tileRoom(1, 0, 0, 0, "Grass.", map[string]int{"east": 2, "up": 3, "southeast": 4}),
		tileRoom(2, 1, 0, 0, "More grass. A stream.", map[string]int{"west": 1}),
		tileRoom(3, 0, 0, 1, "A loft.", map[string]int{"down": 1}),
		tileRoom(4, 1, 1, 0, "A rock.", map[string]int{"northwest": 1, "elsewhere": 99}),
	}
	assert.Empty(t, ValidateTileReady(&ZoneConfig{}, list, allLegends))
}

func TestValidateTileReadyCatchesEachConvention(t *testing.T) {
	cases := []struct {
		name string
		edit func(list []*Room)
		want string
	}{
		{"overlap", func(l []*Room) { l[1].MapX, l[1].MapY = 0, 0 }, "shares coordinate 0,0,0 with room 1"},
		{"no coordinates", func(l []*Room) { l[1].HasCoordinates = false }, "no hand-placed coordinates"},
		{"wrong direction", func(l []*Room) { l[0].Exits["east"] = exit.RoomExit{RoomId: 3} }, `exit "east" leads to room 3 at 0,0,1, expected 1,0,0`},
		{"unmapped name", func(l []*Room) { l[0].Exits["door"] = exit.RoomExit{RoomId: 2} }, `exit "door" has no compass direction`},
		{"mapdirection fixes it", func(l []*Room) { l[0].Exits["door"] = exit.RoomExit{RoomId: 2, MapDirection: "east"} }, ""},
		{"no biome", func(l []*Room) { l[0].Biome = "" }, "no biome"},
		{"unknown resource", func(l []*Room) { l[0].Resources = []string{"unobtainium"} }, `unknown resource "unobtainium"`},
		{"long filler", func(l []*Room) { l[0].Description = "One. Two. Three." }, "3 sentences"},
		{"long landmark is fine", func(l []*Room) { l[0].Description = "One. Two. Three."; l[0].MapLegend = "Inn" }, ""},
		{"bad legend", func(l []*Room) { l[0].MapLegend = "Dragonspire" }, `maplegend "Dragonspire" does not map`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := []*Room{
				tileRoom(1, 0, 0, 0, "Grass.", map[string]int{"east": 2, "up": 3}),
				tileRoom(2, 1, 0, 0, "Grass.", map[string]int{"west": 1}),
				tileRoom(3, 0, 0, 1, "Loft.", map[string]int{"down": 1}),
			}
			tc.edit(list)
			got := ValidateTileReady(&ZoneConfig{}, list, func(l string) bool { return l == "inn" })
			if tc.want == "" {
				assert.Empty(t, got)
				return
			}
			require.NotEmpty(t, got)
			assert.Contains(t, strings.Join(got, "\n"), tc.want)
		})
	}
}

func TestValidateTileReadyUsesTheZoneDefaultBiome(t *testing.T) {
	r := tileRoom(1, 0, 0, 0, "Grass.", nil)
	r.Biome = ""
	assert.Empty(t, ValidateTileReady(&ZoneConfig{DefaultBiome: "forest"}, []*Room{r}, allLegends))
}

func TestSentenceCount(t *testing.T) {
	assert.Equal(t, 0, sentenceCount(""))
	assert.Equal(t, 1, sentenceCount("A stone wall."))
	assert.Equal(t, 2, sentenceCount("It is 3.5 paces wide. A door!"))
	assert.Equal(t, 1, sentenceCount("Dr.Who stands here."))
}

// worldRooms reads the shipped world's rooms by zone folder, straight from
// the YAML, with no game state.
func shippedTileZones(t *testing.T) map[string]struct {
	cfg   ZoneConfig
	rooms []*Room
} {
	t.Helper()
	_, src, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(src))), "_datafiles", "world", "default", "rooms")
	dirs, err := os.ReadDir(root)
	require.NoError(t, err)
	out := map[string]struct {
		cfg   ZoneConfig
		rooms []*Room
	}{}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, d.Name(), "zone-config.yaml"))
		if err != nil {
			continue
		}
		var cfg ZoneConfig
		require.NoError(t, yaml.Unmarshal(raw, &cfg), d.Name())
		if !cfg.TileReady {
			continue
		}
		files, err := filepath.Glob(filepath.Join(root, d.Name(), "*.yaml"))
		require.NoError(t, err)
		var list []*Room
		for _, f := range files {
			if filepath.Base(f) == "zone-config.yaml" {
				continue
			}
			b, err := os.ReadFile(f)
			require.NoError(t, err)
			r := &Room{}
			require.NoError(t, yaml.Unmarshal(b, r), f)
			list = append(list, r)
		}
		out[d.Name()] = struct {
			cfg   ZoneConfig
			rooms []*Room
		}{cfg, list}
	}
	return out
}

// Phase 40d acceptance 1: every shipped tile-ready zone passes the validator.
func TestShippedTileReadyZonesFollowTheConventions(t *testing.T) {
	_, src, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(filepath.Dir(src)))
	raw, err := os.ReadFile(filepath.Join(repo, "_datafiles", "html", "public", "static", "sprites", "map", "landmarks.json"))
	require.NoError(t, err)
	var table struct {
		Legends map[string]string `json:"legends"`
		Glyphs  []string          `json:"glyphs"`
	}
	require.NoError(t, json.Unmarshal(raw, &table))
	legendOK := func(l string) bool {
		if _, ok := table.Legends[l]; ok {
			return true
		}
		for _, g := range table.Glyphs {
			if g == l {
				return true
			}
		}
		return false
	}

	zones := shippedTileZones(t)
	require.Contains(t, zones, "alderbrook", "the showcase zone ships tile-ready")
	for name, z := range zones {
		assert.Empty(t, ValidateTileReady(&z.cfg, z.rooms, legendOK), name)
	}
}

// The showcase shows off every feature the milestone built (design:
// several biomes, every resource, landmarks, up/down, a lock, a camp, a loop).
func TestAlderbrookShowsOffTheMapFeatures(t *testing.T) {
	z := shippedTileZones(t)["alderbrook"]
	require.GreaterOrEqual(t, len(z.rooms), 30)
	require.LessOrEqual(t, len(z.rooms), 50)

	biomes, resources, legends := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var up, down, locked, camp bool
	byId := map[int]*Room{}
	for _, r := range z.rooms {
		byId[r.RoomId] = r
		biomes[r.Biome] = true
		for _, res := range r.Resources {
			resources[res] = true
		}
		if r.MapLegend != "" {
			legends[strings.ToLower(r.MapLegend)] = true
		}
		for _, tag := range r.Tags {
			camp = camp || tag == "camping"
		}
		for name, e := range r.Exits {
			up = up || name == "up"
			down = down || name == "down"
			locked = locked || e.Lock.Difficulty > 0
		}
	}
	for _, b := range []string{"road", "land", "forest", "shore", "cliffs", "cave", "city", "house", "farmland"} {
		assert.True(t, biomes[b], "biome %s", b)
	}
	for _, res := range []string{"water", "forage", "shelter", "herbs", "firewood", "fishing", "game"} {
		assert.True(t, resources[res], "resource %s", res)
	}
	assert.GreaterOrEqual(t, len(legends), 6, "landmarks")
	assert.True(t, up && down && locked && camp, "up %v down %v locked %v camp %v", up, down, locked, camp)

	// One dense loop: the lakeshore ring of eight rooms closes on itself.
	ring := 0
	for _, r := range z.rooms {
		if r.MapZ == 0 && r.MapX >= 5 && r.MapX <= 7 && r.MapY >= 2 && r.MapY <= 4 && !(r.MapX == 6 && r.MapY == 3) {
			ring++
		}
	}
	assert.Equal(t, 8, ring, "lake ring")
}

// The way in: Trappers' Post leads to the showcase and back.
func TestAlderbrookIsReachableFromTheOldKingsRoad(t *testing.T) {
	_, src, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(src))), "_datafiles", "world", "default", "rooms")
	load := func(path string) *Room {
		b, err := os.ReadFile(filepath.Join(root, path))
		require.NoError(t, err)
		r := &Room{}
		require.NoError(t, yaml.Unmarshal(b, r))
		return r
	}
	post := load("old_kings_road/2005.yaml")
	assert.Equal(t, 2101, post.Exits["east"].RoomId)
	assert.Equal(t, 2005, load("alderbrook/2101.yaml").Exits["west"].RoomId)
	assert.Contains(t, post.Tags, "market", "the Trappers' Post keeps its tags")
	assert.Contains(t, load("old_kings_road/2002.yaml").Tags, "camping")
}
