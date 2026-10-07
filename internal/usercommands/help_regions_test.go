package usercommands

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Phases 41 and 42: the regions help page is indexed under the road category,
// answers to its aliases, and names every zone of the chain with the band the
// shipped zone config gives it.
func TestRegionsHelpListsTheRoadAndItsBands(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "regions" && topic.Category == "road" {
			indexed = true
		}
	}
	assert.True(t, indexed, "help index lists regions under road")

	want, err := GetHelpContents("regions")
	require.NoError(t, err)
	for _, alias := range []string{"zones", "level guide", "where to go", "bands"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help regions", alias)
	}
	// Review fix: aliases must not take other topics' words: `zone` is the
	// admin zone command's page and `levels`/`world` are too broad.
	for _, word := range []string{"levels", "zone", "world"} {
		got, _ := GetHelpContents(word)
		assert.NotEqual(t, want, got, "help %s is not help regions", word)
	}
	text := tagPattern.ReplaceAllString(want, "")
	assert.Contains(t, text, "Help for regions")

	zones := []struct {
		name, band string
	}{
		{"Alderbrook", "levels 2-4"}, {"Brindle Downs", "levels 4-6"}, {"Marrowmere Fen", "levels 6-8"},
		{"Greywatch Pass", "levels 9-11"}, {"Cinder Hollow", "levels 12-14"}, {"Thornreach Wood", "levels 15-17"},
		{"Hollowweb Deep", "levels 18-20"}, {"Ashen Barrows", "levels 21-24"}, {"Glassvault Depths", "levels 25-28"},
		{"Stormcrown Heights", "levels 29-33"},
	}
	for _, z := range zones {
		assert.Contains(t, text, z.name)
		assert.Contains(t, text, z.band, z.name)
	}

	// The page matches the data: each zone's configured band reads as the page says.
	_, src, _, _ := runtime.Caller(0)
	roomsDir := filepath.Join(filepath.Dir(src), "..", "..", "_datafiles", "world", "default", "rooms")
	dirs := map[string]string{
		"Alderbrook": "alderbrook", "Brindle Downs": "brindle_downs", "Marrowmere Fen": "marrowmere_fen",
		"Greywatch Pass": "greywatch_pass", "Cinder Hollow": "cinder_hollow", "Thornreach Wood": "thornreach_wood",
		"Hollowweb Deep": "hollowweb_deep", "Ashen Barrows": "ashen_barrows", "Glassvault Depths": "glassvault_depths",
		"Stormcrown Heights": "stormcrown_heights",
	}
	for _, z := range zones {
		raw, err := os.ReadFile(filepath.Join(roomsDir, dirs[z.name], "zone-config.yaml"))
		require.NoError(t, err, z.name)
		var cfg struct {
			Name       string `yaml:"name"`
			Encounters struct {
				Band struct {
					Low  int `yaml:"low"`
					High int `yaml:"high"`
				} `yaml:"band"`
			} `yaml:"encounters"`
		}
		require.NoError(t, yaml.Unmarshal(raw, &cfg), z.name)
		assert.Equal(t, z.name, cfg.Name)
		assert.Contains(t, text, z.name+"  ", z.name)
		assert.Contains(t, text, "levels "+strconv.Itoa(cfg.Encounters.Band.Low)+"-"+strconv.Itoa(cfg.Encounters.Band.High), z.name)
	}

	// The hub pages link it.
	for _, topic := range []string{"encounters", "travel"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, tagPattern.ReplaceAllString(page, ""), "help regions", topic)
	}
}
