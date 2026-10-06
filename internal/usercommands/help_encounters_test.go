package usercommands

import (
	"strings"
	"testing"

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
	for _, want := range []string{"Help for", "two entries", "30 real seconds", "15 in 100", "one group in\n  six", "twice the health", "half an hour", "Lv 5-7", "dangerous", "two or three escorts", "your company alone", "cache", "its own roll"} {
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
	for level, want := range map[int]string{12: "Your company should manage.", 9: "A fair test", 7: "Risky at your level", 5: "Dangerous at your level: prepare carefully."} {
		user.Character.Level = level
		note := zoneBandNote(user, room)
		assert.Contains(t, note, "Foes in Band Test Wilds are of levels 10 to 12.", "level %d", level)
		assert.Contains(t, note, want, "level %d", level)
	}
	assert.Empty(t, zoneBandNote(user, &rooms.Room{RoomId: 96903, Zone: "No Such Zone"}), "no band, no line")
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
	assert.Contains(t, out, "Foes in Band Look Wilds are of levels 5 to 7. Your company should manage.")
}
