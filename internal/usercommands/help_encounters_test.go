package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/encounters"
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
	for _, want := range []string{"Help for", "two entries", "30 real seconds", "15 in 100", "one group in\n  five", "1.75 times", "two or three escorts", "your company alone", "cache", "its own roll"} {
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
