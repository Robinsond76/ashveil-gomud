package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPartyConsentCommandsAndLeadership(t *testing.T) {
	p := parties.New(93301)
	require.NotNil(t, p)
	t.Cleanup(p.Disband)
	leader := users.NewUserRecord(93301, 0)
	member := users.NewUserRecord(93302, 0)
	leader.Character.Name = "Leader"
	member.Character.Name = "Member"
	for _, u := range []*users.UserRecord{leader, member} {
		users.SetTestUser(u)
		t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	}
	p.InvitePlayer(member.UserId)
	command := func(u *users.UserRecord, text string) {
		t.Helper()
		_, err := Party(text, u, &rooms.Room{}, 0)
		require.NoError(t, err)
	}
	command(member, "follow on")
	command(member, "support on")
	command(member, "autoattack on")
	assert.False(t, p.Follows(member.UserId))
	assert.False(t, p.Supports(member.UserId))
	assert.Empty(t, p.GetAutoAttackUserIds())
	require.True(t, p.AcceptInvite(member.UserId))
	command(member, "follow on")
	command(member, "support on")
	command(member, "autoattack on")
	assert.True(t, p.Follows(member.UserId))
	assert.True(t, p.Supports(member.UserId))
	assert.Equal(t, []int{member.UserId}, p.GetAutoAttackUserIds())
	command(leader, "support on")
	assert.Equal(t, []int{member.UserId}, parties.AlliedLeaders(leader.UserId))
	command(member, "follow nonsense")
	assert.True(t, p.Follows(member.UserId), "invalid syntax cannot change consent")
	command(member, "promote Member")
	assert.Equal(t, leader.UserId, p.LeaderUserId, "nonleader has no promotion authority")
	command(leader, "promote Member")
	assert.Equal(t, member.UserId, p.LeaderUserId)
	assert.Empty(t, p.Followers)
	command(leader, "follow on")
	assert.True(t, p.Follows(leader.UserId))
	command(member, "leave")
	assert.Equal(t, leader.UserId, p.LeaderUserId)
	assert.Empty(t, p.Followers)
	assert.False(t, p.Supports(member.UserId))
	assert.Empty(t, p.GetAutoAttackUserIds())
}

func TestPartyFollowMovementRequiresConsent(t *testing.T) {
	previous := configs.Flatten(configs.GetOverrides())
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	fixtures := map[string]string{
		"biomes/default.yaml":               "biomeid: default\nname: Test\nsymbol: '.'\ndarkarea: true\n",
		"keywords.yaml":                     "direction-aliases: {}\n",
		"rooms/traveltest/zone-config.yaml": "name: traveltest\nroomid: 933001\n",
	}
	for path, data := range bidirectionalExits(933001, 933002, "  north:\n    roomid: 933002\n") {
		fixtures["rooms/traveltest/"+path+".yaml"] = data
	}
	loadTravelTestWorld(t, fixtures)
	for _, scenario := range []string{"default off", "opt in", "separated", "invited"} {
		t.Run(scenario, func(t *testing.T) {
			consent := scenario != "default off"
			leader := travelTestUser(t, 93311, 933001)
			follower := users.NewUserRecord(93312, 0)
			follower.Character.RoomId = 933001
			users.SetTestUser(follower)
			p := parties.New(leader.UserId)
			require.NotNil(t, p)
			t.Cleanup(p.Disband)
			p.InvitePlayer(follower.UserId)
			if scenario != "invited" {
				p.AcceptInvite(follower.UserId)
			}
			if scenario == "separated" {
				follower.Character.RoomId = 933002
			}
			p.SetFollow(follower.UserId, consent)
			var inputs []events.Input
			id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
				inputs = append(inputs, e.(events.Input))
				return events.Cancel
			})
			t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })
			events.ProcessEvents()
			_, err := Go("north", leader, rooms.LoadRoom(933001), 0)
			require.NoError(t, err)
			events.ProcessEvents()
			assert.Equal(t, 933002, leader.Character.RoomId)
			if scenario == "separated" {
				assert.Equal(t, 933002, follower.Character.RoomId)
			} else {
				assert.Equal(t, 933001, follower.Character.RoomId, "only the owner's validated command may move them")
			}
			if scenario == "opt in" {
				require.Len(t, inputs, 1)
				assert.Equal(t, follower.UserId, inputs[0].UserId)
				assert.Equal(t, "go north", inputs[0].InputText)
				assert.NotNil(t, inputs[0].PartyFollow)
			} else {
				assert.Empty(t, inputs)
			}
		})
	}
}

func TestPartyConsentHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	page, err := GetHelpContents("party")
	require.NoError(t, err)
	for _, want := range []string{"party follow on", "party support on", "default off", "Each owner", "copyover", "Minor Heal All", "most damage", "bury", "online member"} {
		assert.Contains(t, page, want)
	}
	for _, alias := range []string{"alliance", "allied-companies", "party-follow", "party-support"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, page, got)
	}
	for _, topic := range []string{"company", "combat", "friendly-effects"} {
		got, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, got, "help party")
	}
}
