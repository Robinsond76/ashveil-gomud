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
// forward (the module opens those), so a player can only walk back to an
// earlier stage's room.
func TestShippedTutorialRooms(t *testing.T) {
	ids := shippedTutorialRooms(t)
	require.Len(t, ids, len(stages), "one tutorial room per stage")
	dir := filepath.Join(repoRoot(), "_datafiles", "world", "default", "rooms", "tutorial")
	used := map[int]bool{}
	for i, stage := range stages {
		require.Less(t, stage.Room, len(ids))
		assert.False(t, used[stage.Room], "stage %s shares a room", stage.ID)
		used[stage.Room] = true
		id := ids[stage.Room]
		var room struct {
			RoomId      int    `yaml:"roomid"`
			Title       string `yaml:"title"`
			Description string `yaml:"description"`
			Exits       map[string]struct {
				RoomId int `yaml:"roomid"`
			} `yaml:"exits"`
			Mutators  []any    `yaml:"mutators"`
			SpawnInfo []any    `yaml:"spawninfo"`
			Tags      []string `yaml:"tags"`
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
			for j, earlier := range stages {
				if ids[earlier.Room] == e.RoomId {
					back = j
				}
			}
			assert.True(t, back >= 0 && back < i, "room %d exit %s leads back to an earlier stage", id, name)
		}
		// Phase 27b: shelter is shown by comparison, and the Camp stage's
		// room allows a camp.
		switch stage.ID {
		case StageCharacter:
			assert.Contains(t, room.Tags, "indoor", "the Waking Hall is sheltered")
		case StageSurvival:
			assert.Contains(t, room.Tags, "outdoor", "the Weather Yard is exposed")
		case StageCamp:
			assert.Contains(t, room.Tags, "camping")
		case StageCombat:
			assert.NotContains(t, room.Tags, "camping", "the squad is raised by the module, not camped beside")
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
		RationItemId     int   `yaml:"RationItemId"`
		WaterItemId      int   `yaml:"WaterItemId"`
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

	// The graduation cap and the Survival supplies exist; the supplies can
	// be eaten and drunk.
	assert.Equal(t, defaultRationItem, tut.RationItemId)
	assert.Equal(t, defaultWaterItem, tut.WaterItemId)
	for _, id := range []int{tut.GraduationItemId, tut.RationItemId, tut.WaterItemId} {
		found, err := filepath.Glob(filepath.Join(repoRoot(), "_datafiles", "world", "default", "items", "*", strconv.Itoa(id)+"-*.yaml"))
		require.NoError(t, err)
		deeper, err := filepath.Glob(filepath.Join(repoRoot(), "_datafiles", "world", "default", "items", "*", "*", strconv.Itoa(id)+"-*.yaml"))
		require.NoError(t, err)
		found = append(found, deeper...)
		require.Len(t, found, 1, "item %d", id)
		var spec struct {
			Subtype   string `yaml:"subtype"`
			Nutrition int    `yaml:"nutrition"`
			Hydration int    `yaml:"hydration"`
		}
		readYAML(t, found[0], &spec)
		switch id {
		case tut.RationItemId:
			assert.Equal(t, "edible", spec.Subtype)
			assert.Positive(t, spec.Nutrition)
		case tut.WaterItemId:
			assert.Equal(t, "drinkable", spec.Subtype)
			assert.Positive(t, spec.Hydration)
		}
	}
}

// TestShippedHelpTemplate: "help tutorial" ships with the module and is
// listed.
func TestShippedHelpTemplate(t *testing.T) {
	data, err := files.ReadFile("files/datafiles/templates/help/tutorial.template")
	require.NoError(t, err)
	assert.Contains(t, string(data), "tutorial skip")
	for _, stage := range []string{"Survival", "Camp", "Combat"} {
		assert.Contains(t, string(data), stage)
	}
	kw, err := os.ReadFile(filepath.Join(repoRoot(), "_datafiles", "world", "default", "keywords.yaml"))
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(kw), "      - tutorial\n"))
}

// TestShippedPracticeSquad (27c): the squad's foes are practice mobs of the
// harmless dummy race in one party, the footmen hardier than the archer so
// the archer stands behind them.
func TestShippedPracticeSquad(t *testing.T) {
	var tut struct {
		PracticeSquad []int `yaml:"PracticeSquad"`
	}
	readYAML(t, filepath.Join(repoRoot(), "modules", "tutorial", "files", "data-overlays", "config.yaml"), &tut)
	assert.Equal(t, defaultSquad, tut.PracticeSquad)
	var dummy struct {
		Damage struct {
			DiceRoll string `yaml:"diceroll"`
		} `yaml:"damage"`
	}
	readYAML(t, filepath.Join(repoRoot(), "_datafiles", "world", "default", "races", "19-dummy.yaml"), &dummy)
	assert.Equal(t, "0d0", dummy.Damage.DiceRoll, "the dummy race can't hurt anyone")
	levels := map[int]int{}
	for _, id := range tut.PracticeSquad {
		found, err := filepath.Glob(filepath.Join(repoRoot(), "_datafiles", "world", "default", "mobs", "tutorial", strconv.Itoa(id)+"-*.yaml"))
		require.NoError(t, err)
		require.Len(t, found, 1, "mob %d", id)
		var mob struct {
			Practice  bool     `yaml:"practice"`
			Hostile   bool     `yaml:"hostile"`
			Groups    []string `yaml:"groups"`
			Character struct {
				RaceId int `yaml:"raceid"`
				Level  int `yaml:"level"`
				Items  []any
				Gold   int `yaml:"gold"`
			} `yaml:"character"`
		}
		readYAML(t, found[0], &mob)
		assert.True(t, mob.Practice, "mob %d", id)
		assert.False(t, mob.Hostile, "mob %d waits to be attacked", id)
		assert.Equal(t, []string{"practice-squad"}, mob.Groups)
		assert.Equal(t, 19, mob.Character.RaceId)
		assert.Empty(t, mob.Character.Items)
		assert.Zero(t, mob.Character.Gold)
		levels[id] = mob.Character.Level
	}
	assert.Greater(t, levels[67], levels[68], "footmen in front, the archer behind")
}
