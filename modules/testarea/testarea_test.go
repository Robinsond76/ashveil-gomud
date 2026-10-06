package testarea

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestTheAreaIsClosed: every test room's exits lead to another test room,
// so no walk, route or journey can leave (and none leads in from the
// world), and the rooms are named in the zones the command knows.
func TestTheAreaIsClosed(t *testing.T) {
	world := shippedWorld()
	zoneDirs := map[string]string{"Test Area": "test_area", "Test Area Road": "test_area_road"}
	seen := map[int]bool{}
	for zone, dir := range zoneDirs {
		files, err := filepath.Glob(filepath.Join(world, "rooms", dir, "*.yaml"))
		require.NoError(t, err)
		for _, f := range files {
			if filepath.Base(f) == "zone-config.yaml" {
				continue
			}
			data, err := os.ReadFile(f)
			require.NoError(t, err)
			var r struct {
				RoomID int    `yaml:"roomid"`
				Zone   string `yaml:"zone"`
				Exits  map[string]struct {
					RoomID int `yaml:"roomid"`
				} `yaml:"exits"`
			}
			require.NoError(t, yaml.Unmarshal(data, &r), f)
			assert.Equal(t, zone, r.Zone, f)
			assert.True(t, IsAreaRoom(r.RoomID), f)
			seen[r.RoomID] = true
			for dir, e := range r.Exits {
				assert.True(t, IsAreaRoom(e.RoomID), "%s: exit %s leaves the test area", f, dir)
			}
		}
	}
	for id := FirstRoom; id <= LastRoom; id++ {
		assert.True(t, seen[id], "room %d exists", id)
	}
	for word, id := range roomNames {
		assert.True(t, IsAreaRoom(id), word)
	}
	for _, r := range roomList {
		assert.True(t, IsAreaRoom(r.ID), r.Word)
		got, ok := target(r.Word)
		assert.True(t, ok)
		assert.Equal(t, r.ID, got)
	}
}

// TestOnlyAdminsGoInAndToolsNeedATrip.
func TestOnlyAdminsGoInAndToolsNeedATrip(t *testing.T) {
	tr := newTrip(t)
	tr.user.Role = users.RoleUser
	// A player gets no hint the command or its help exists: the word is
	// an unknown command and the help topic is missing.
	tr.messages = nil
	handled, err := usercommands.TryCommand("testarea", "", tr.user.UserId, events.CmdSkipScripts)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.False(t, handled, "testarea is no command to a player")
	for _, msg := range tr.messages {
		assert.NotContains(t, msg, "permission")
		assert.NotContains(t, msg, "testarea")
	}
	assert.Equal(t, 2001, tr.user.Character.RoomId)
	tr.messages = nil
	_, _ = usercommands.TryCommand("help", "testarea", tr.user.UserId, events.CmdSkipScripts)
	events.ProcessEvents()
	help := strings.Join(tr.messages, "\n")
	assert.Contains(t, help, "No help found")
	assert.NotContains(t, help, "closed set of rooms")
	assert.NotContains(t, tr.run("help", ""), "testarea", "the help index lists no admin topic to a player")
	tr.user.Role = users.RoleAdmin

	for _, tool := range []string{"class wizard", "level 9", "gold", "kit weapons", "fight 3 2 58", "companion add warrior"} {
		assert.Contains(t, tr.run("testarea", tool), "Start a trip first", tool)
	}
	assert.Contains(t, tr.run("testarea", "return"), "not on a test area trip")
	assert.Contains(t, tr.run("testarea", "nowhere"), "no test room")
	assert.Equal(t, 2001, tr.user.Character.RoomId)
	assert.Contains(t, tr.run("testarea", "status"), "not on a test area trip")
	assert.Contains(t, tr.run("testarea", "rooms"), "armory")
}

// TestFightsAndCompanionTools drives the combat yard and the class tools
// through the real command.
func TestFightsAndCompanionTools(t *testing.T) {
	tr := newTrip(t)
	u := tr.user
	require.Contains(t, tr.run("testarea", "yard"), "Combat Yard")
	assert.Equal(t, 90002, u.Character.RoomId)
	assert.Contains(t, tr.run("testarea", "status"), "trip begun")

	assert.Contains(t, tr.run("testarea", "fight 4 9 58"), "size must be")
	assert.Contains(t, tr.run("testarea", "fight 4 3 nonsense"), "no foe type")
	assert.Contains(t, tr.run("testarea", "fight 4 3 58"), "3 foes of level 4")
	room := rooms.LoadRoom(90002)
	foes := 0
	for _, id := range room.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.EncounterOwner == u.UserId {
			foes++
			assert.Equal(t, 4, m.Character.Level)
		}
	}
	assert.Equal(t, 3, foes)
	assert.Contains(t, tr.run("testarea", "clear"), "3 foes are gone")
	assert.Contains(t, tr.run("testarea", "clear"), "No foes")

	// Companions of any class and level, and class changes.
	assert.Contains(t, tr.run("testarea", "companion add halberdier 12"), "level 12")
	assert.Contains(t, tr.run("testarea", "companion add shogun 20"), "shogun")
	members, ok := company.CompanyMembers(u.UserId)
	require.True(t, ok)
	require.Len(t, members, 2)
	byClass := map[string]int{}
	for _, m := range members {
		byClass[m.Class] = m.Level
	}
	assert.Equal(t, 20, byClass["shogun"])
	assert.Contains(t, tr.run("testarea", "companion class #1 knight"), "knight")
	assert.Contains(t, tr.run("testarea", "companion level #1 7"), "level 7")
	members, _ = company.CompanyMembers(u.UserId)
	for _, m := range members {
		if m.ID == 1 {
			assert.Equal(t, "knight", m.Class)
			assert.Equal(t, 7, m.Level)
		}
	}
	assert.Contains(t, tr.run("testarea", "companion class #1 nothing"), "no archetype or class")
	assert.Contains(t, tr.run("testarea", "companion level #1 0"), "from 1 to")

	// Own class and level, including an elite and a neutral class.
	for _, c := range []string{"wizard", "paladin", "samurai", "shogun", "dollmaster", "stormcaller"} {
		assert.NotContains(t, tr.run("testarea", "class "+c), "no archetype", c)
		class, _ := u.Character.ClassState()
		if _, isClass := classes.Get(c); isClass {
			assert.Equal(t, c, class, c)
		} else {
			assert.Equal(t, c, u.Character.ArchetypeID(), c)
			assert.Empty(t, class, c)
		}
	}
	assert.Contains(t, tr.run("testarea", "level 40"), "level 40")
	assert.Equal(t, 40, u.Character.Level)
	assert.Contains(t, tr.run("testarea", "level 2"), "level 2")
	assert.Equal(t, 2, u.Character.Level)
	assert.Contains(t, tr.run("testarea", "level 101"), "from 1 to")

	// The armory: any item, kits, search; the camp and stable and the
	// weather deck answer their commands.
	assert.Contains(t, tr.run("testarea", "give nonsense-item"), "No item matches")
	assert.Contains(t, tr.run("testarea", "items sword"), "#")
	assert.Contains(t, tr.run("testarea", "kit armor"), "You take")
	assert.Contains(t, tr.run("testarea", "kit saddles"), "You take")
	assert.Contains(t, tr.run("testarea", "kit supplies"), "You take")
	assert.Contains(t, tr.run("testarea", "kit nothing"), "Usage")
	tr.run("testarea", "weather")
	assert.Contains(t, tr.run("testarea", "weather"), "Test Area weather:")
	assert.Contains(t, tr.run("testarea", "weather rain"), "now rain")
	assert.Contains(t, tr.run("testarea", "weather auto"), "weather is now")
	assert.Contains(t, tr.run("testarea", "weather bogus"), "no \"bogus\"")
	assert.Contains(t, tr.run("testarea", "return"), "back as you were")
	assert.Equal(t, 2001, u.Character.RoomId)
}

// TestRoadCampAlwaysDrawsThievesAndRaiders: the camp room is quiet and the
// road camp, in its own zone, is on the camp theft and raid lists at 100
// percent, so the thieves' road is a toggle by room.
func TestRoadCampAlwaysDrawsThievesAndRaiders(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(shippedWorld(), "..", "..", "..", "modules", "camping", "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	var cfg struct {
		CampTheft []struct {
			Zone      string `yaml:"Zone"`
			ChancePct int    `yaml:"ChancePct"`
		} `yaml:"CampTheft"`
		CampRaids []struct {
			Zone      string `yaml:"Zone"`
			ChancePct int    `yaml:"ChancePct"`
		} `yaml:"CampRaids"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	chance := func(zone string, list []struct {
		Zone      string `yaml:"Zone"`
		ChancePct int    `yaml:"ChancePct"`
	}) int {
		for _, e := range list {
			if e.Zone == zone {
				return e.ChancePct
			}
		}
		return 0
	}
	assert.Equal(t, 100, chance("Test Area Road", cfg.CampTheft))
	assert.Equal(t, 100, chance("Test Area Road", cfg.CampRaids))
	assert.Zero(t, chance("Test Area", cfg.CampTheft), "the quiet camp draws no thieves")
	assert.Zero(t, chance("Test Area", cfg.CampRaids))
}

// TestEveryModuleStateIsSnapshotted: the contributors a trip captures.
func TestEveryModuleStateIsSnapshotted(t *testing.T) {
	assert.Subset(t, userstate.Names(), []string{"archetype", "camping", "company", "encumbrance", "expedition", "exposure", "mount", "strategy", "survival", "walking", "encounters", "death"})
}

// TestHelpPages: the command's own help and `help testarea` both render and
// name every subcommand, and the two agree.
func TestHelpPages(t *testing.T) {
	tr := newTrip(t)
	fromCommand := tr.run("testarea", "help")
	fromHelp := tr.run("help", "testarea")
	for _, want := range []string{"testarea return", "testarea fight", "testarea class", "testarea level", "testarea companion add", "testarea kit", "testarea weather", "closed set of rooms"} {
		assert.Contains(t, fromCommand, want)
		assert.Contains(t, fromHelp, want)
	}
}
