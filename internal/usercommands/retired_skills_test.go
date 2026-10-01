package usercommands

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var retiredCommands = []string{"backstab", "bump", "changeform", "peep", "pickpocket", "portal", "pray", "scribe", "search", "sneak", "tame", "track"}

// TestRetiredSkillCommandsAreGone (33f1): the retired skill commands are no
// longer commands, and their help topics and aliases are gone.
func TestRetiredSkillCommandsAreGone(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	for _, cmd := range retiredCommands {
		_, ok := userCommands[cmd]
		assert.False(t, ok, cmd)
		switch cmd {
		case "backstab", "search", "track":
			continue // their help names open skulduggery, keeneye, and trail
		}
		_, err := GetHelpContents(cmd)
		assert.Error(t, err, "help %s", cmd)
	}
	for _, topic := range []string{"hire", "guide", "explorer", "monster-hunter"} {
		_, err := GetHelpContents(topic)
		assert.Error(t, err, "help %s", topic)
	}
	for _, topic := range []string{"skulduggery", "protection", "jobs", "training-schools"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		for _, cmd := range []string{"sneak", "bump", "pickpocket", "pray", "portal", "scribe", "peep", "tame", "backstab"} {
			assert.NotRegexp(t, `\b`+cmd+`\b`, strings.ToLower(text), "help %s still names %s", topic, cmd)
		}
	}
}

// TestMercenariesAreNotForHire (33f1): a shop stocking a mercenary neither
// lists nor sells it, so no one gains a charmed follower by buying.
func TestMercenariesAreNotForHire(t *testing.T) {
	const sellsword = 989301
	merc := &mobs.Mob{MobId: sellsword, Character: *characters.New()}
	merc.Character.Name = "sellsword"
	merc.Character.Level = 3
	mobs.SetTestSpec(merc)
	t.Cleanup(func() { mobs.RemoveTestSpec(sellsword) })

	setupCarry(t, map[int]int{})
	room := testRoom()
	room.RoomId = 988301
	buyer := carrier(t, 7, "Dain", room)
	buyer.Character.Gold = 100000

	shop := &mobs.Mob{MobId: 989302, InstanceId: 989303, Character: *characters.New()}
	shop.Character.Name = "Bonecrafter"
	shop.Character.RoomId = room.RoomId
	shop.Character.Shop = characters.Shop{{MobId: sellsword}}
	mobs.SetTestInstance(shop)
	t.Cleanup(func() { mobs.RemoveTestInstance(shop.InstanceId) })
	room.AddMob(shop.InstanceId)

	listing := captureUserText(t, func() {
		_, err := List("", buyer, room, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, strings.ToLower(listing), "mercenar")
	assert.False(t, tryPurchase("sellsword", buyer, room, shop, nil))
	assert.Empty(t, buyer.Character.GetCharmIds())
	assert.Equal(t, 100000, buyer.Character.Gold)
}

// TestWorldDataHasNoRetiredSkills (33f1): no trainer, profession, item, or
// world script still teaches or calls a retired skill or the removed charm
// scripting API.
func TestWorldDataHasNoRetiredSkills(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	world := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	retired := regexp.MustCompile(`\b(changeform|peep|portal|scribe|search|tame)\b`)
	api := regexp.MustCompile(`\b(CharmSet|CharmRemove|CharmExpire|GetCharmCount|GetMaxCharmCount|IsTameable|GetTameMastery|SetTameMastery|GetChanceToTame|TrainSkill\("(changeform|peep|portal|scribe|search|tame)")`)

	walk := func(dir string, fn func(path string, data []byte)) {
		require.NoError(t, filepath.Walk(filepath.Join(world, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			fn(path, data)
			return nil
		}))
	}
	walk("rooms", func(path string, data []byte) {
		if strings.HasSuffix(path, ".yaml") {
			var room struct {
				SkillTraining map[string]any `yaml:"skilltraining"`
			}
			require.NoError(t, yaml.Unmarshal(data, &room), path)
			for id := range room.SkillTraining {
				assert.NotRegexp(t, retired, id, path)
			}
		}
	})
	walk("professions", func(path string, data []byte) {
		var p struct {
			Skills []string `yaml:"skills"`
		}
		require.NoError(t, yaml.Unmarshal(data, &p), path)
		for _, id := range p.Skills {
			assert.NotRegexp(t, retired, id, path)
			assert.FileExists(t, filepath.Join(world, "skills", id+".yaml"), path)
		}
	})
	walk("items", func(path string, data []byte) {
		if strings.HasSuffix(path, ".yaml") {
			var it struct {
				StatMods map[string]int `yaml:"statmods"`
			}
			require.NoError(t, yaml.Unmarshal(data, &it), path)
			_, tame := it.StatMods["tame"]
			assert.False(t, tame, path)
		}
	})
	for _, dir := range []string{"rooms", "mobs", "items", "spells", "quests"} {
		walk(dir, func(path string, data []byte) {
			if strings.HasSuffix(path, ".js") {
				assert.NotRegexp(t, api, string(data), path)
			}
			if strings.HasSuffix(path, ".yaml") && strings.Contains(path, "quests") {
				assert.NotRegexp(t, `skillinfo:\s*"?(changeform|peep|portal|scribe|search|tame)`, string(data), path)
			}
		})
	}
}

// TestEveryIndexedHelpTopicResolves (33f1 review): every player topic the
// help index lists, and every help alias, opens a page, so retiring a page
// can't leave a dangling entry.
func TestEveryIndexedHelpTopicResolves(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	// Gaps that predate 33f1 (GoMud topics with no page) and pages a
	// module ships (the tutorial's, not loaded here).
	known := map[string]bool{"bid": true, "bury": true, "help": true, "store": true, "trash": true, "unstore": true, "tutorial": true, "replay": true}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.AdminOnly || known[topic.Command] {
			continue
		}
		_, err := GetHelpContents(topic.Command)
		assert.NoError(t, err, "help %s (%s)", topic.Command, topic.Category)
	}
	for alias := range keywords.GetAllHelpAliases() {
		if known[alias] {
			continue
		}
		_, err := GetHelpContents(alias)
		assert.NoError(t, err, "help alias %s", alias)
	}
}
