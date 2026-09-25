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
	assert.Equal(t, []StageID{StageCharacter, StageCompany, StageFormation, StageDeparture}, ids)
	assert.Equal(t, 1, stageIndex(StageCompany))
	assert.Equal(t, -1, stageIndex("nope"))
	for i, s := range stages {
		assert.Equal(t, i, s.Room, "one room per stage, in order")
		assert.NotEmpty(t, s.Title)
		assert.NotEmpty(t, s.Goal)
	}
}

func TestProgressRoundTrip(t *testing.T) {
	c := &characters.Character{}
	p := progressOf(c)
	assert.Equal(t, stateNone, p.State)
	p = progress{State: stateActive, Stage: StageCompany, Seen: map[string]bool{"status": true, "inventory": true}}
	p.save(c)

	data, err := yaml.Marshal(c)
	require.NoError(t, err)
	reloaded := &characters.Character{}
	require.NoError(t, yaml.Unmarshal(data, reloaded))
	got := progressOf(reloaded)
	assert.Equal(t, p.State, got.State)
	assert.Equal(t, p.Stage, got.Stage)
	assert.Equal(t, p.Seen, got.Seen)
}

func TestCharacterGate(t *testing.T) {
	p := progress{Seen: map[string]bool{}}
	for _, cmd := range inspections[:3] {
		p.Seen[cmd] = true
		assert.False(t, characterDone(p))
	}
	p.Seen[inspections[3]] = true
	assert.True(t, characterDone(p))
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
