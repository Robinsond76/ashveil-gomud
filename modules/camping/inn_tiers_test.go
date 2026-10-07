package camping

import (
	"gopkg.in/yaml.v2"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 75: inn room tiers.

func suiteInnRoom() *rooms.Room {
	return &rooms.Room{RoomId: 2003, Title: "The Waymark Inn", Zone: "Dunmar", Tags: []string{"inn", "inn-private", "inn-suite", "indoor"}}
}

func privateInnRoom() *rooms.Room {
	return &rooms.Room{RoomId: 3108, Title: "The Drovers' Rest", Zone: "Brindle Downs", Tags: []string{"inn", "inn-private"}}
}

func TestInnRoomsFollowRoomTags(t *testing.T) {
	s := defaultInnSettings()
	tiers := func(r *rooms.Room) []camping.InnTier {
		var out []camping.InnTier
		for _, o := range s.innRooms(r) {
			out = append(out, o.Tier)
		}
		return out
	}
	assert.Equal(t, []camping.InnTier{camping.InnCommon}, tiers(innRoom()))
	assert.Equal(t, []camping.InnTier{camping.InnCommon, camping.InnPrivate}, tiers(privateInnRoom()))
	assert.Equal(t, []camping.InnTier{camping.InnCommon, camping.InnPrivate, camping.InnSuite}, tiers(suiteInnRoom()))
	assert.Empty(t, tiers(ineligibleRoom()))
	// A suite tag alone never makes a room an inn.
	assert.Empty(t, tiers(&rooms.Room{RoomId: 5, Tags: []string{"inn-suite"}}))
}

func TestParseInnTier(t *testing.T) {
	for word, want := range map[string]camping.InnTier{"": camping.InnCommon, "common": camping.InnCommon, "room": camping.InnCommon, "private": camping.InnPrivate, "SUITE": camping.InnSuite} {
		got, ok := camping.ParseInnTier(word)
		assert.True(t, ok, word)
		assert.Equal(t, want, got, word)
	}
	_, ok := camping.ParseInnTier("penthouse")
	assert.False(t, ok)
}

func TestInnStatusListsEveryRoomWithItsPrice(t *testing.T) {
	e := newInnEnv(t) // company of 2
	user := campUser(t, 7, 2003)
	user.Character.Gold = 500
	text := e.module.innStatus(user, suiteInnRoom())
	assert.Contains(t, text, "Common: 10 gold (5 per member), Well Rested for 30 minutes")
	assert.Contains(t, text, "Private: 30 gold (15 per member), Well Rested for 1 hour")
	assert.Contains(t, text, "Suite: 80 gold (40 per member), Well Rested for 2 hours")
	assert.Contains(t, text, `"inn rest suite"`)
	plain := e.module.innStatus(user, innRoom())
	assert.NotContains(t, plain, "Private")
	assert.NotContains(t, plain, "Suite")
}

func TestInnRestRefusesARoomTheInnLacks(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 500
	assert.Contains(t, e.module.innRestTier(user, innRoom(), camping.InnSuite), "has no suite room")
	assert.Contains(t, e.module.innRestTier(user, privateInnRoom(), camping.InnSuite), "has no suite room")
	assert.Equal(t, 500, user.Character.Gold)
	assert.Empty(t, e.module.stays)
}

func TestInnCommandPicksTheTier(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 500
	messages := captureMessages(t)
	_, err := e.module.innCommand("rest penthouse", user, suiteInnRoom(), 0)
	require.NoError(t, err)
	assert.Empty(t, e.module.stays)
	_, err = e.module.innCommand("rest suite", user, suiteInnRoom(), 0)
	require.NoError(t, err)
	require.Contains(t, e.module.stays, 7)
	assert.Equal(t, camping.InnSuite, e.module.stays[7].Tier)
	assert.Equal(t, 420, user.Character.Gold, "80 gold for a company of 2")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "no \"penthouse\" room")
	assert.Equal(t, camping.InnSuite, e.store.saved.Stays[7].Tier, "the tier is saved")
}

func TestCommonRoomStaysSavedWithoutATier(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 50
	e.module.innRest(user, suiteInnRoom())
	assert.Equal(t, camping.InnTier(""), e.store.saved.Stays[7].Tier, "a common stay reads as one saved before the tiers")
	assert.Equal(t, 40, user.Character.Gold)
}

// TestEachRoomTierLeavesItsOwnWellRestedLength runs a real stay in each
// tier through the timer and the NewRound grant.
func TestEachRoomTierLeavesItsOwnWellRestedLength(t *testing.T) {
	for _, tc := range []struct {
		tier    camping.InnTier
		price   int
		minutes int
	}{
		{camping.InnCommon, 10, 30},
		{camping.InnPrivate, 30, 60},
		{camping.InnSuite, 80, 120},
	} {
		t.Run(string(tc.tier), func(t *testing.T) {
			e := newInnEnv(t)
			user := campUser(t, 7, 2003)
			user.Character.Name = "Hero"
			user.Character.Gold = 100
			messages := captureMessages(t)
			require.Contains(t, e.module.innRestTier(user, suiteInnRoom(), tc.tier), "You pay")
			assert.Equal(t, 100-tc.price, user.Character.Gold)
			*e.now = baseTime().Add(60 * time.Second)
			e.scheduler.fireLatest()
			e.module.onNewRound(events.NewRound{RoundNumber: 1})

			want := tc.minutes * 60 / 4 // four-second rounds
			assert.Equal(t, want, e.ledger.rounds[buffCall{"Hero", 1030}])
			assert.Equal(t, want, e.ledger.rounds[buffCall{"Bran", 1030}], "companions share the room's length")
			assert.Empty(t, e.store.saved.Stays, "the stay is spent once granted")
			// The companion owed the buff keeps the same length for an absence.
			owed := e.store.saved.Owed[7][1]
			assert.Equal(t, camping.TierWellRested, owed.Tier)
			assert.Equal(t, e.now.Add(time.Duration(tc.minutes)*time.Minute), owed.ExpiresAtUTC)
			events.ProcessEvents()
			if tc.tier != camping.InnCommon {
				assert.Contains(t, strings.Join(*messages, "\n"), "Well Rested for")
			}
		})
	}
}

func TestAShorterRoomNeverCutsALongerWellRestedShort(t *testing.T) {
	e := newInnEnv(t)
	hero := &characters.Character{Name: "Hero"}
	s := defaultInnSettings()
	e.module.buffRounds = func(*characters.Character, int) int { return 1800 } // a suite's 2 h, half spent
	e.ledger.hold(hero, 1030)
	assert.False(t, e.module.applyTier(hero, camping.TierWellRested, 450, s), "30 minutes does not replace an hour left")
	assert.Empty(t, *e.buffs)
	assert.True(t, e.module.applyTier(hero, camping.TierWellRested, 1800, s), "an equal or longer grant still refreshes")
	assert.True(t, e.module.applyTier(hero, camping.TierWellRested, 1801, s))
}

func TestInnStandingMarkupAppliesToEveryTier(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 500
	// The settlement's standing prices the whole room, whatever its tier.
	price, members := e.module.innPrice(7, innStanding(7, suiteInnRoom()), 40)
	assert.Equal(t, 2, members)
	assert.Equal(t, 80, price, "no standing, no markup")
}

func TestStayRecordsRejectUnknownTiers(t *testing.T) {
	stay, err := camping.StartInnStay(7, 2003, 80, baseTime(), time.Minute, 60)
	require.NoError(t, err)
	stay.Tier = camping.InnSuite
	assert.NoError(t, stay.Validate())
	stay.Tier = "penthouse"
	assert.Error(t, stay.Validate())
}

func TestCampStateOffersTheInnsRooms(t *testing.T) {
	e := newInnEnv(t)
	rows := e.module.innRoomRows(7, suiteInnRoom())
	require.Len(t, rows, 3)
	assert.Equal(t, camping.InnRoomRow{Tier: camping.InnSuite, Price: 80, Minutes: 120}, rows[2])
	assert.Equal(t, camping.InnRoomRow{Tier: camping.InnCommon, Price: 10, Minutes: 30}, rows[0])
}

// TestShippedInnsOfferTheirRooms reads the default world's rooms: the
// Waymark Inn lets all three rooms, the Drovers' Rest the first two, and a
// room tagged for a dearer room is always an inn.
func TestShippedInnsOfferTheirRooms(t *testing.T) {
	root := filepath.Join("..", "..", "_datafiles", "world", "default", "rooms")
	tags := func(path string) []string {
		raw, err := os.ReadFile(filepath.Join(root, path))
		require.NoError(t, err)
		var room struct {
			Tags []string `yaml:"tags"`
		}
		require.NoError(t, yaml.Unmarshal(raw, &room))
		return room.Tags
	}
	assert.Subset(t, tags("dunmar/2003.yaml"), []string{"inn", "inn-private", "inn-suite"})
	drovers := tags("brindle_downs/3108.yaml")
	assert.Contains(t, drovers, "inn-private")
	assert.NotContains(t, drovers, "inn-suite")

	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		got := tags(rel)
		for _, tag := range []string{"inn-private", "inn-suite"} {
			if slices.Contains(got, tag) {
				assert.Contains(t, got, "inn", "%s is tagged %s but is not an inn", rel, tag)
			}
		}
		return nil
	}))
}
