package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTempoSharedEnemyAndFightReplacement(t *testing.T) {
	battle.Reset()
	ResetTempoForTest()
	t.Cleanup(battle.Reset)
	t.Cleanup(ResetTempoForTest)
	m := engagementMob(t, 8991, 100, 1)
	m.Character.Stats.Speed.ValueAdj = 30
	m.Character.SetAggro(1, 0, characters.DefaultAttack, 0)
	battle.Begin(1, 1, 1, "shared", []int{8991})
	battle.SetFight(1, 101)
	battle.Begin(2, 1, 1, "shared", []int{8991})
	battle.SetFight(2, 102)
	who := caster{mobId: 8991}
	for _, want := range []int{1, 1, 2} {
		clear(tempoTurns)
		fillTempo(who, &m.Character)
		fillTempo(who, &m.Character)
		assert.Equal(t, want, tempoTurns[who], "one fill per actor, regardless of battles")
	}
	battle.End(1)
	clear(tempoTurns)
	fillTempo(who, &m.Character)
	assert.Equal(t, 1, tempoTurns[who], "remaining engagement keeps fraction")
	assert.InDelta(t, 50, tempoMeters[who].meter.Points, 1e-9)
	battle.End(2)
	battle.Begin(1, 1, 2, "new", []int{8991})
	battle.SetFight(1, 103)
	clear(tempoTurns)
	fillTempo(who, &m.Character)
	assert.Equal(t, 1, tempoTurns[who])
	assert.Zero(t, tempoMeters[who].meter.Points, "new fight opens fresh")
}

func TestTempoUnmanagedRestartAndCharacterReplacement(t *testing.T) {
	users.ResetActiveUsers()
	ResetTempoForTest()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(ResetTempoForTest)
	u := users.NewUserRecord(8992, 1)
	users.SetTestUser(u)
	u.Character.Health = 100
	u.Character.Stats.Speed.ValueAdj = 30
	u.Character.SetAggro(8993, 0, characters.DefaultAttack, 0)
	who := caster{userId: 8992}
	fillTempo(who, u.Character)
	clear(tempoTurns)
	fillTempo(who, u.Character)
	assert.InDelta(t, 50, tempoMeters[who].meter.Points, 1e-9)
	u.Character.EndAggro()
	u.Character.SetAggro(8993, 0, characters.DefaultAttack, 0)
	clear(tempoTurns)
	fillTempo(who, u.Character)
	assert.Zero(t, tempoMeters[who].meter.Points)
	u.Character.EndAggro()
	pruneTempo()
	assert.Empty(t, tempoMeters)
}

type tempoCompanionLookup struct{}

func (tempoCompanionLookup) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (tempoCompanionLookup) InstanceFor(int, int) (int, bool) { return 8994, true }
func (tempoCompanionLookup) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 7, company.CompanionMemberKey(1), true
}

func TestTempoSeparatedCompanionIndependentRestart(t *testing.T) {
	battle.Reset()
	ResetTempoForTest()
	t.Cleanup(battle.Reset)
	t.Cleanup(ResetTempoForTest)
	company.SetFormationProvider(tempoCompanionLookup{})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	m := engagementMob(t, 8994, 100, 2)
	m.Character.Stats.Speed.ValueAdj = 30
	battle.Begin(7, 1, 1, "owner elsewhere", []int{8995})
	who := caster{mobId: 8994}
	for _, active := range []bool{false, true} {
		tempoActive = active
		snapshotTempoFights()
		assert.Empty(t, tempoMembership(who))
	}
	m.Character.SetAggro(0, 8996, characters.DefaultAttack, 0)
	fillTempo(who, &m.Character)
	clear(tempoTurns)
	fillTempo(who, &m.Character)
	assert.InDelta(t, 50, tempoMeters[who].meter.Points, 1e-9)
	m.Character.EndAggro()
	m.Character.SetAggro(0, 8996, characters.DefaultAttack, 0)
	clear(tempoTurns)
	fillTempo(who, &m.Character)
	assert.Zero(t, tempoMeters[who].meter.Points, "owner's battle cannot mask independent restart")
}
