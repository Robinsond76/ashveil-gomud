package archetype

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wiring: these tests go through the real engine entry points (the user
// train command, the scripting LearnSpell exports) with the real module
// registered as the internal/archetypes provider.

// registered builds a module from the shipped config and registers it as
// the archetypes provider for the test.
func registered(t *testing.T) *ArchetypeModule {
	t.Helper()
	m, _ := testModule(t)
	archetypes.SetProvider(m)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	return m
}

// captureText collects text sent to users while fn runs.
func captureText(t *testing.T, fn func()) string {
	t.Helper()
	events.ProcessEvents()
	var messages []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, id)
	fn()
	events.ProcessEvents()
	return strings.Join(messages, "\n")
}

func trainingRoom(t *testing.T, id int) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: id, Zone: "ArchetypeTest", Title: "Training Hall", SkillTraining: map[string]rooms.TrainingRange{
		"cast":   {Min: 1, Max: 4},
		"search": {Min: 1, Max: 4},
	}}
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	return r
}

func trainee(t *testing.T, id, roomID int) *users.UserRecord {
	t.Helper()
	u := newUser(id)
	u.Character.RoomId = roomID
	u.Character.TrainingPoints = 20
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(id) })
	return u
}

func TestWiringTrainRefusesClaimedSkillUntilChosen(t *testing.T) {
	m := registered(t)
	room := trainingRoom(t, 96001)
	u := trainee(t, 41, 96001)

	text := captureText(t, func() {
		_, err := usercommands.Train("cast", u, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Choose an archetype first")
	assert.Zero(t, u.Character.GetSkillLevel("cast"))
	assert.Equal(t, 20, u.Character.TrainingPoints, "no points spent on a refusal")

	_, err := usercommands.Train("search", u, room, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, u.Character.GetSkillLevel("search"), "trade skills stay open")

	m.choose(u, "wizard", true)
	before := u.Character.TrainingPoints
	_, err = usercommands.Train("cast", u, room, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, u.Character.GetSkillLevel("cast"), "a wizard trains cast")
	assert.Less(t, u.Character.TrainingPoints, before)
}

func TestWiringTrainRefusesWrongArchetypeAndMarksPanel(t *testing.T) {
	m := registered(t)
	room := trainingRoom(t, 96002)
	u := trainee(t, 42, 96002)
	m.choose(u, "warrior", true)

	text := captureText(t, func() {
		_, err := usercommands.Train("cast", u, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Cleric or Wizard")
	assert.Zero(t, u.Character.GetSkillLevel("cast"))
	assert.Equal(t, 20, u.Character.TrainingPoints)

	panel := captureText(t, func() {
		_, err := usercommands.Train("", u, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, panel, "Cleric or Wizard only")
}

func TestWiringTrainUnchangedWithoutProvider(t *testing.T) {
	archetypes.SetProvider(nil)
	room := trainingRoom(t, 96003)
	u := trainee(t, 43, 96003)
	_, err := usercommands.Train("cast", u, room, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, u.Character.GetSkillLevel("cast"), "upstream behaviour without the module")
}

func TestWiringScriptLearnSpellIsGated(t *testing.T) {
	m := registered(t)
	u := trainee(t, 44, 96004)
	m.choose(u, "wizard", true)

	actor := scripting.GetActor(44, 0)
	require.NotNil(t, actor)
	assert.False(t, actor.LearnSpell("heal"), "a wizard can't learn restoration")
	assert.False(t, u.Character.HasSpell("heal"))
	assert.True(t, actor.LearnSpell("illum"), "illusion is a wizard school")

	// The party export goes through the same per-member gate.
	actor.GetParty().LearnSpell("healall")
	assert.False(t, u.Character.HasSpell("healall"))
	actor.GetParty().LearnSpell("mm")
	assert.True(t, u.Character.HasSpell("mm"))
}

func TestWiringScriptLearnSpellRefusalExplains(t *testing.T) {
	registered(t)
	trainee(t, 45, 96005)
	text := captureText(t, func() {
		assert.False(t, scripting.GetActor(45, 0).LearnSpell("heal"))
	})
	assert.Contains(t, text, "Cleric")
}

// companyStub is a formation provider that also reports companion
// archetypes (as modules/company does).
type companyStub struct {
	leader    int
	companion int
	instance  int
	archetype string
}

func (c companyStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (c companyStub) InstanceFor(leader, companionID int) (int, bool) {
	return c.instance, leader == c.leader && companionID == c.companion
}
func (c companyStub) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	if instanceId != c.instance {
		return 0, "", false
	}
	return c.leader, company.CompanionMemberKey(c.companion), true
}
func (c companyStub) CompanionArchetype(leader, companionID int) (string, bool) {
	if leader == c.leader && companionID == c.companion && c.archetype != "" {
		return c.archetype, true
	}
	return "", false
}

func TestWiringLookShowsArchetypes(t *testing.T) {
	m := registered(t)
	room := &rooms.Room{RoomId: 96010, Zone: "ArchetypeTest", Title: "Hall"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(96010) })

	viewer := trainee(t, 50, 96010)
	viewer.Character.Name = "Viewer"
	target := trainee(t, 51, 96010)
	target.Character.Name = "Orin"
	m.choose(target, "wizard", true)
	room.AddPlayer(50)
	room.AddPlayer(51)

	bran := &mobs.Mob{InstanceId: 96901, Character: *characters.New()}
	bran.Character.Name = "Bran"
	bran.Character.RoomId = 96010
	mobs.SetTestInstance(bran)
	t.Cleanup(func() { mobs.RemoveTestInstance(96901) })
	room.AddMob(96901)
	company.SetFormationProvider(companyStub{leader: 51, companion: 1, instance: 96901, archetype: "rogue"})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	text := captureText(t, func() {
		_, err := usercommands.Look("orin", viewer, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, `Archetype: <ansi fg="yellow">Wizard</ansi>`)

	text = captureText(t, func() {
		_, err := usercommands.Look("bran", viewer, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, `Archetype: <ansi fg="yellow">Rogue</ansi>`)

	text = captureText(t, func() {
		_, err := usercommands.Look("viewer", target, room, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "Archetype:", "an unchosen player shows nothing")
}

// Review 17a finding 1: scripts (e.g. the Whispering Wastes obelisk) train
// skills through ScriptActor.TrainSkill, which must respect claims too.
func TestWiringScriptTrainSkillIsGated(t *testing.T) {
	m := registered(t)
	u := trainee(t, 46, 96006)
	m.choose(u, "warrior", true)
	actor := scripting.GetActor(46, 0)
	require.NotNil(t, actor)

	var ok bool
	text := captureText(t, func() { ok = actor.TrainSkill("portal", 1) })
	assert.False(t, ok, "portal is a wizard skill")
	assert.Zero(t, u.Character.GetSkillLevel("portal"))
	assert.Contains(t, text, "Wizard")

	assert.True(t, actor.TrainSkill("map", 1), "trade skills stay open")
	assert.Equal(t, 1, u.Character.GetSkillLevel("map"))

	wiz := trainee(t, 47, 96006)
	m.choose(wiz, "wizard", true)
	assert.True(t, scripting.GetActor(47, 0).TrainSkill("portal", 1))
	assert.Equal(t, 1, wiz.Character.GetSkillLevel("portal"))
}

// Review 17a finding 1: quest skill rewards respect claims.
func TestWiringQuestSkillRewardIsGated(t *testing.T) {
	m := registered(t)
	quests.SetTestQuest(&quests.Quest{QuestId: 9601, Name: "Obelisk Lore",
		Steps:   []quests.QuestStep{{Id: "start"}, {Id: "end"}},
		Rewards: quests.QuestReward{SkillInfo: "portal:1"}})
	t.Cleanup(func() { quests.RemoveTestQuest(9601) })

	warrior := trainee(t, 48, 96007)
	m.choose(warrior, "warrior", true)
	hooks.HandleQuestUpdate(events.Quest{UserId: 48, QuestToken: "9601-start"})
	text := captureText(t, func() {
		hooks.HandleQuestUpdate(events.Quest{UserId: 48, QuestToken: "9601-end"})
	})
	assert.Zero(t, warrior.Character.GetSkillLevel("portal"))
	assert.Contains(t, text, "Wizard")

	wizard := trainee(t, 49, 96007)
	m.choose(wizard, "wizard", true)
	hooks.HandleQuestUpdate(events.Quest{UserId: 49, QuestToken: "9601-start"})
	hooks.HandleQuestUpdate(events.Quest{UserId: 49, QuestToken: "9601-end"})
	assert.Equal(t, 1, wizard.Character.GetSkillLevel("portal"))
}
