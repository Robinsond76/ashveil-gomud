package tutorial

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStageOrder(t *testing.T) {
	ids := []StageID{}
	for _, s := range stages {
		ids = append(ids, s.ID)
	}
	assert.Equal(t, []StageID{StageCharacter, StageCompany, StageFormation, StageSurvival, StageCamp, StageCombat, StageDeparture}, ids)
	assert.Equal(t, 1, stageIndex(StageCompany))
	assert.Equal(t, -1, stageIndex("nope"))
	// Later rooms (27b's 904 and 905, 27c's 906) come after the Gate (903)
	// in TutorialRooms, so earlier room indexes stay put.
	rooms := []int{}
	for _, s := range stages {
		rooms = append(rooms, s.Room)
		assert.NotEmpty(t, s.Title)
		assert.NotEmpty(t, s.Goal)
	}
	assert.Equal(t, []int{0, 1, 2, 4, 5, 6, 3}, rooms)
}

func everything(string) bool { return true }

func TestRequiredInspectionsSkipMissingCommands(t *testing.T) {
	s := stages[stageIndex(StageSurvival)]
	assert.Equal(t, []string{"weather", "temperature", "strain", "cargo"}, required(s, everything))
	noWeather := func(cmd string) bool { return cmd != "weather" }
	assert.Equal(t, []string{"temperature", "strain", "cargo"}, required(s, noWeather))
	assert.Empty(t, required(stages[stageIndex(StageCompany)], everything))
}

func TestSurvivalGate(t *testing.T) {
	p := progress{Seen: map[string]bool{}}
	for _, cmd := range []string{"weather", "temperature", "strain", "cargo"} {
		p.Seen[cmd] = true
	}
	assert.False(t, survivalDone(p, everything), "not yet fed or watered")
	p.Seen[seenFed] = true
	assert.False(t, survivalDone(p, everything))
	p.Seen[seenWatered] = true
	assert.True(t, survivalDone(p, everything))
	delete(p.Seen, "strain")
	assert.False(t, survivalDone(p, everything))
	assert.True(t, survivalDone(p, func(cmd string) bool { return cmd != "strain" }), "a missing command isn't asked for")
}

func TestProgressRoundTrip(t *testing.T) {
	c := &characters.Character{}
	p := progressOf(c)
	assert.Equal(t, stateNone, p.State)
	p = progress{State: stateActive, Stage: StageCompany, Seen: map[string]bool{"status": true, "inventory": true}, Supplied: true}
	p.save(c)

	data, err := yaml.Marshal(c)
	require.NoError(t, err)
	reloaded := &characters.Character{}
	require.NoError(t, yaml.Unmarshal(data, reloaded))
	got := progressOf(reloaded)
	assert.Equal(t, p.State, got.State)
	assert.Equal(t, p.Stage, got.Stage)
	assert.Equal(t, p.Seen, got.Seen)
	assert.True(t, got.Supplied)
}

func TestCharacterGate(t *testing.T) {
	p := progress{Seen: map[string]bool{}}
	inspections := stages[stageIndex(StageCharacter)].Inspections
	assert.Equal(t, []string{"status", "inventory", "experience", "conditions"}, inspections)
	for _, cmd := range inspections[:3] {
		p.Seen[cmd] = true
		assert.False(t, characterDone(p, everything))
	}
	p.Seen[inspections[3]] = true
	assert.True(t, characterDone(p, everything))
}

func view(id int, status company.MemberStatus) company.MemberView {
	return company.MemberView{ID: id, Status: status}
}

func TestCompanyGate(t *testing.T) {
	assert.False(t, companyDone(nil))
	assert.False(t, companyDone([]company.MemberView{view(1, company.MemberPresent)}))
	assert.False(t, companyDone([]company.MemberView{view(1, company.MemberPresent), view(2, company.MemberDead)}), "the dead don't count")
	assert.True(t, companyDone([]company.MemberView{view(1, company.MemberPresent), view(2, company.MemberAwaiting)}))
}

func TestFormationGate(t *testing.T) {
	members := []company.MemberView{view(1, company.MemberPresent), view(2, company.MemberPresent), view(3, company.MemberDead)}
	var f company.Formation
	assert.False(t, formationDone(f, members))
	require.NoError(t, f.Place(company.CompanionMemberKey(1), 0, 1))
	assert.False(t, formationDone(f, members), "a front-liner alone")
	require.NoError(t, f.Place(company.CompanionMemberKey(2), 0, 2))
	assert.False(t, formationDone(f, members), "both in front")
	require.NoError(t, f.Place(company.CompanionMemberKey(2), 2, 1))
	assert.True(t, formationDone(f, members))

	var g company.Formation
	require.NoError(t, g.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, g.Place(company.CompanionMemberKey(3), 1, 1))
	require.NoError(t, g.Place(company.CompanionMemberKey(1), 1, 0))
	assert.False(t, formationDone(g, members), "the leader and the dead don't count; no companion in front")
}
