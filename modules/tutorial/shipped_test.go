package tutorial

import (
	"os"
	"path/filepath"
	"regexp"
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
	assert.Contains(t, string(data), "tutorial replay", "32b")
	for _, stage := range []string{"Survival", "Camp", "Combat", "Alignment"} {
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

// TestShippedOathStone (27d): the Alignment stage's room offers Corvin, an
// outlaw a company made of the course's recruits is refused by the gate.
func TestShippedOathStone(t *testing.T) {
	var comp struct {
		RecruitMaxGap int `yaml:"RecruitMaxGap"`
		Recruiters    []struct {
			RoomId     int `yaml:"RoomId"`
			Candidates []struct {
				Id            string `yaml:"Id"`
				MobTemplateId int    `yaml:"MobTemplateId"`
				Tutorial      bool   `yaml:"Tutorial"`
				Price         int    `yaml:"Price"`
			} `yaml:"Candidates"`
		} `yaml:"Recruiters"`
	}
	readYAML(t, filepath.Join(repoRoot(), "modules", "company", "files", "data-overlays", "config.yaml"), &comp)
	oath := shippedTutorialRooms(t)[stages[stageIndex(StageAlignment)].Room]
	assert.Equal(t, 907, oath)
	var corvin int
	for _, r := range comp.Recruiters {
		if r.RoomId != oath {
			continue
		}
		for _, c := range r.Candidates {
			assert.Equal(t, "corvin", c.Id, "the hints name him")
			assert.False(t, c.Tutorial, "not a free claim")
			assert.Positive(t, c.Price)
			corvin = c.MobTemplateId
		}
	}
	require.NotZero(t, corvin)
	alignment := func(dir string, id int) int {
		found, err := filepath.Glob(filepath.Join(repoRoot(), "_datafiles", "world", "default", "mobs", dir, strconv.Itoa(id)+"-*.yaml"))
		require.NoError(t, err)
		require.Len(t, found, 1, "mob %d", id)
		var mob struct {
			Character struct {
				Alignment int `yaml:"alignment"`
			} `yaml:"character"`
		}
		readYAML(t, found[0], &mob)
		return mob.Character.Alignment
	}
	gap := comp.RecruitMaxGap
	if gap == 0 {
		gap = 60
	}
	average := (alignment("dunmar", 61) + alignment("dunmar", 62)) / 2
	assert.Greater(t, average-alignment("tutorial", corvin), gap, "refused to the course's company")
	assert.Greater(t, 0-alignment("tutorial", corvin), gap, "and to a company of one new, neutral leader")
}

// TestTutorialHelpPointersExist: every "help <topic>" a lesson or the
// tutorial's own help page points to is a shipped help page (in the world,
// or in a module), or an alias of one, and the Combat lesson points to
// the combat pages.
func TestTutorialHelpPointersExist(t *testing.T) {
	pointer := regexp.MustCompile(`help ([a-z-]+)</ansi>`)
	var texts []string
	for _, s := range stages {
		texts = append(texts, s.Hints...)
	}
	page, err := files.ReadFile("files/datafiles/templates/help/tutorial.template")
	require.NoError(t, err)
	texts = append(texts, string(page))

	var kw struct {
		Aliases map[string][]string `yaml:"help-aliases"`
	}
	readYAML(t, filepath.Join(repoRoot(), "_datafiles", "world", "default", "keywords.yaml"), &kw)
	aliasOf := map[string]string{}
	for topic, aliases := range kw.Aliases {
		for _, a := range aliases {
			aliasOf[a] = topic
		}
	}
	exists := func(topic string) bool {
		if real, ok := aliasOf[topic]; ok {
			topic = real
		}
		dirs := []string{filepath.Join(repoRoot(), "_datafiles", "world", "default", "templates", "help")}
		mods, _ := filepath.Glob(filepath.Join(repoRoot(), "modules", "*", "files", "datafiles", "templates", "help"))
		for _, dir := range append(dirs, mods...) {
			for _, ext := range []string{".md", ".template"} {
				if _, err := os.Stat(filepath.Join(dir, topic+ext)); err == nil {
					return true
				}
			}
		}
		return false
	}

	seen := map[string]bool{}
	for _, text := range texts {
		for _, m := range pointer.FindAllStringSubmatch(text, -1) {
			seen[m[1]] = true
			assert.True(t, exists(m[1]), "help %s is a shipped page", m[1])
		}
	}
	for _, topic := range []string{"combat", "formation", "targeting", "battle-summary", "sharpen",
		"adventure", "status", "archetype", "company", "survival", "weather", "temperature", "strain",
		"cargo", "travel", "camp", "inn", "cooking", "alignment", "standing", "market", "rumors"} {
		assert.True(t, seen[topic], "the tutorial points to help %s", topic)
	}
}
