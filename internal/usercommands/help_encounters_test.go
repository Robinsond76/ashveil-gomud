package usercommands

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncountersHelp (Phase 37): the page renders through help, is indexed
// under combat, answers to its aliases, and the pages the phase touched
// point to it.
func TestEncountersHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "encounters" && topic.Category == "combat" {
			listed = true
		}
	}
	assert.True(t, listed, "help encounters is indexed under combat")

	page, err := GetHelpContents("encounters")
	require.NoError(t, err)
	text := tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{"Help for", "two entries", "30 real seconds", "15 in 100", "one group in\n  six", "twice the health", "half an hour", "Lv 5-7", "dangerous", "two or three escorts", "your company alone", "cache", "its own roll", "two-fifths of the health", "fifteen or more", "five under"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"encounter", "random-encounters", "danger", "boss", "cache", "spoils", "personal loot", "grace"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help encounters", alias)
	}
	for _, topic := range []string{"combat", "loot", "scout", "retreat", "party", "travel", "ambush", "battle-summary"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "encounters", "help %s points to help encounters", topic)
	}
}

// TestScoutWarnsOfADangerousRoom: with nothing in sight, scout says when
// the room can spring an encounter, and stays quiet in an ordinary room.
func TestScoutWarnsOfADangerousRoom(t *testing.T) {
	room := &rooms.Room{RoomId: 96901}
	user := users.NewUserRecord(96901, 0)
	assert.Equal(t, `You see no enemies here.`, scoutList(room, user))
	room.Encounter = &encounters.RoomSetting{Enabled: false}
	assert.Equal(t, `You see no enemies here.`, scoutList(room, user))
	room.Encounter = &encounters.RoomSetting{Enabled: true, Table: "t"}
	assert.Contains(t, scoutList(room, user), "feels dangerous")
	assert.NotContains(t, scoutList(room, user), "levels", "no band, no level line")

	// With a band, scout names the levels the zone's foes come at.
	room.Zone = "Scout Test Wilds"
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: room.Zone, Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 5, High: 7}}}))
	assert.Contains(t, scoutList(room, user), "Foes in Scout Test Wilds are of levels 5 to 7.")
}

// TestZoneBandNoteRatesTheBandAgainstTheViewersLevel (37b): look and scout
// share one line naming the zone's band and how it weighs on the viewer.
func TestZoneBandNoteRatesTheBandAgainstTheViewersLevel(t *testing.T) {
	room := &rooms.Room{RoomId: 96902, Zone: "Band Test Wilds"}
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: room.Zone, Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 10, High: 12}}}))
	user := users.NewUserRecord(96902, 0)
	for level, want := range map[int]string{12: "Your company (level 12) should manage.", 9: "A fair test at your company's level (9).", 7: "Risky at your company's level (7): expect losses.", 5: "Dangerous at your company's level (5): prepare carefully."} {
		user.Character.Level = level
		note := zoneBandNote(user, room)
		assert.Contains(t, note, "Foes in Band Test Wilds are of levels 10 to 12.", "level %d", level)
		assert.Contains(t, note, want, "level %d", level)
	}
	assert.Empty(t, zoneBandNote(user, &rooms.Room{RoomId: 96903, Zone: "No Such Zone"}), "no band, no line")
}

// bandCompany is a fake company provider listing fixed companion levels.
type bandCompany struct{ levels []int }

func (bandCompany) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (bandCompany) InstanceFor(int, int) (int, bool)           { return 0, false }
func (bandCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}
func (b bandCompany) CompanyMembers(int) ([]company.MemberView, bool) {
	out := make([]company.MemberView, len(b.levels))
	for i, l := range b.levels {
		out[i] = company.MemberView{ID: i + 1, Level: l, Status: company.MemberPresent}
	}
	return out, true
}

// TestZoneBandNoteRatesByTheCompanyLevel (37c): a leader under the band
// whose company is of band level reads "should manage"; and the other way.
func TestZoneBandNoteRatesByTheCompanyLevel(t *testing.T) {
	room := &rooms.Room{RoomId: 96905, Zone: "Company Band Wilds"}
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: room.Zone, Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 10, High: 12}}}))
	user := users.NewUserRecord(96905, 0)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	user.Character.Level = 6
	company.SetFormationProvider(bandCompany{levels: []int{12, 12, 12, 12}}) // (6+48)/5 = 10.8 -> 11
	assert.Contains(t, zoneBandNote(user, room), "Your company (level 11) should manage.")

	user.Character.Level = 12
	company.SetFormationProvider(bandCompany{levels: []int{3, 3, 3, 3}}) // (12+12)/5 = 4.8 -> 5
	assert.Contains(t, zoneBandNote(user, room), "Dangerous at your company's level (5)")

	company.SetFormationProvider(nil) // no company readable: the leader alone
	assert.Contains(t, zoneBandNote(user, room), "Your company (level 12) should manage.")
}

// TestLookShowsTheZoneBand (37b): the real look command prints the band.
func TestLookShowsTheZoneBand(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "bandwood", Name: "Band Wood", Symbol: "b", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("bandwood") })
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "Band Look Wilds", Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 5, High: 7}}}))
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(96904, 1)
	user.Character.Level = 5
	users.SetTestUser(user)
	room := &rooms.Room{RoomId: 96904, Zone: "Band Look Wilds", Biome: "bandwood"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(96904) })
	room.SetTestOccupants([]int{96904}, nil)

	messages := captureLookMessages(t)
	_, err := Look("", user, room, events.CmdSecretly)
	require.NoError(t, err)
	events.ProcessEvents()
	out := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, out, "Foes in Band Look Wilds are of levels 5 to 7. Your company (level 5) should manage.")
}

// TestLookAndScoutSayHowLongALairStaysQuiet (37b review): a lair the
// company emptied tells look and scout how long it stays quiet.
func TestLookAndScoutSayHowLongALairStaysQuiet(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "lairwood", Name: "Lair Wood", Symbol: "l", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("lairwood") })
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(96905, 1)
	users.SetTestUser(user)
	room := &rooms.Room{RoomId: 96905, Zone: "Lair Look Wilds", Biome: "lairwood", Encounter: &encounters.RoomSetting{Enabled: true, Table: "lair"}}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(96905) })
	room.SetTestOccupants([]int{96905}, nil)

	prev := encounters.LairQuiet
	t.Cleanup(func() { encounters.LairQuiet = prev })
	left := 90 * time.Second
	encounters.LairQuiet = func(userID, roomID int) time.Duration {
		if userID == 96905 && roomID == 96905 {
			return left
		}
		return 0
	}
	assert.Contains(t, scoutList(room, user), "Your company emptied this lair: its master will not rise here for you for about 2 minutes.")

	messages := captureLookMessages(t)
	_, err := Look("", user, room, events.CmdSecretly)
	require.NoError(t, err)
	events.ProcessEvents()
	out := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, out, "for about 2 minutes.")

	left = 0
	assert.NotContains(t, scoutList(room, user), "emptied this lair", "an awake lair says nothing")
}
