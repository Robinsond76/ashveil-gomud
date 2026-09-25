package tutorial

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func repoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func readYAML(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, into))
}

// shippedTutorialRooms reads SpecialRooms.TutorialRooms from the shipped
// config.
func shippedTutorialRooms(t *testing.T) []int {
	t.Helper()
	var cfg struct {
		SpecialRooms struct {
			TutorialRooms []int `yaml:"TutorialRooms"`
		} `yaml:"SpecialRooms"`
	}
	readYAML(t, filepath.Join(repoRoot(), "_datafiles", "config.yaml"), &cfg)
	return cfg.SpecialRooms.TutorialRooms
}

// TestShippedTutorialRooms pins the course's rooms: one per stage, each
// with a description, no script, mutator, or spawns, and no static way
// forward (the module opens those), so a player can only walk back.
func TestShippedTutorialRooms(t *testing.T) {
	ids := shippedTutorialRooms(t)
	require.Len(t, ids, len(stages), "one tutorial room per stage")
	dir := filepath.Join(repoRoot(), "_datafiles", "world", "default", "rooms", "tutorial")
	for i, id := range ids {
		var room struct {
			RoomId      int    `yaml:"roomid"`
			Title       string `yaml:"title"`
			Description string `yaml:"description"`
			Exits       map[string]struct {
				RoomId int `yaml:"roomid"`
			} `yaml:"exits"`
			Mutators  []any `yaml:"mutators"`
			SpawnInfo []any `yaml:"spawninfo"`
		}
		readYAML(t, filepath.Join(dir, strconv.Itoa(id)+".yaml"), &room)
		assert.Equal(t, id, room.RoomId)
		assert.NotEmpty(t, room.Title, "room %d", id)
		assert.NotEmpty(t, room.Description, "room %d", id)
		assert.Empty(t, room.Mutators, "room %d", id)
		assert.Empty(t, room.SpawnInfo, "room %d", id)
		_, err := os.Stat(filepath.Join(dir, strconv.Itoa(id)+".js"))
		assert.True(t, os.IsNotExist(err), "room %d has no script", id)
		for name, e := range room.Exits {
			back := -1
			for j, other := range ids {
				if other == e.RoomId {
					back = j
				}
			}
			assert.True(t, back >= 0 && back < i, "room %d exit %s leads back inside the course", id, name)
		}
	}
	scripts, err := filepath.Glob(filepath.Join(dir, "*.js"))
	require.NoError(t, err)
	assert.Empty(t, scripts, "the old room scripts are removed")
}

// TestShippedTutorialRecruiter pins the Muster Yard: the Company stage's
// room has a recruiter offering exactly the tutorial module's recruits, as
// free tutorial candidates.
func TestShippedTutorialRecruiter(t *testing.T) {
	var tut struct {
		TutorialRecruits []int `yaml:"TutorialRecruits"`
		GraduationItemId int   `yaml:"GraduationItemId"`
	}
	readYAML(t, filepath.Join(repoRoot(), "modules", "tutorial", "files", "data-overlays", "config.yaml"), &tut)
	require.Len(t, tut.TutorialRecruits, 2)

	var comp struct {
		Recruiters []struct {
			RoomId     int `yaml:"RoomId"`
			Candidates []struct {
				MobTemplateId int  `yaml:"MobTemplateId"`
				Tutorial      bool `yaml:"Tutorial"`
				Price         int  `yaml:"Price"`
			} `yaml:"Candidates"`
		} `yaml:"Recruiters"`
	}
	readYAML(t, filepath.Join(repoRoot(), "modules", "company", "files", "data-overlays", "config.yaml"), &comp)

	muster := shippedTutorialRooms(t)[stageIndex(StageCompany)]
	var offered []int
	for _, r := range comp.Recruiters {
		if r.RoomId != muster {
			continue
		}
		for _, c := range r.Candidates {
			assert.True(t, c.Tutorial, "template %d is a tutorial candidate", c.MobTemplateId)
			assert.Zero(t, c.Price)
			offered = append(offered, c.MobTemplateId)
		}
	}
	assert.ElementsMatch(t, tut.TutorialRecruits, offered)

	// The graduation cap exists.
	items, err := filepath.Glob(filepath.Join(repoRoot(), "_datafiles", "world", "default", "items", "*", "*", strconv.Itoa(tut.GraduationItemId)+"-*.yaml"))
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

// TestShippedHelpTemplate: "help tutorial" ships with the module and is
// listed.
func TestShippedHelpTemplate(t *testing.T) {
	data, err := files.ReadFile("files/datafiles/templates/help/tutorial.template")
	require.NoError(t, err)
	assert.Contains(t, string(data), "tutorial skip")
	kw, err := os.ReadFile(filepath.Join(repoRoot(), "_datafiles", "world", "default", "keywords.yaml"))
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(kw), "      - tutorial\n"))
}
