package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const xpLeader = 5150

// xpProvider attaches the listed mob instances to leader 5150.
type xpProvider struct{ attached map[int]bool }

func (xpProvider) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (xpProvider) InstanceFor(int, int) (int, bool)           { return 0, false }
func (p xpProvider) LeaderAndKeyForInstance(id int) (int, company.MemberKey, bool) {
	if p.attached[id] {
		return xpLeader, company.CompanionMemberKey(id), true
	}
	return 0, "", false
}

func xpMob(t *testing.T, instance, roomID, hp int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{}
	m.InstanceId = instance
	m.Character = *characters.New()
	m.Character.Name = "companion"
	m.Character.Level = 1
	m.Character.RoomId = roomID
	m.Character.Validate(true)
	m.Character.Health = hp
	m.Character.Charm(xpLeader, -1, "")
	mobs.SetTestInstance(m)
	t.Cleanup(func() { mobs.RemoveTestInstance(instance) })
	return m
}

// TestAwardCompanyXPPaysPresentLivingCompanions: attached, living
// companions in the kill room earn the full figure; absent, dead, and
// unattached ones earn nothing.
func TestAwardCompanyXPPaysPresentLivingCompanions(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	const room = 7001
	present := xpMob(t, 880001, room, 5)
	present2 := xpMob(t, 880002, room, 5)
	dead := xpMob(t, 880003, room, 0)
	away := xpMob(t, 880004, room+1, 5)
	stranger := xpMob(t, 880005, room, 5)
	leader := characters.New()
	for _, m := range []*mobs.Mob{present, present2, dead, away, stranger} {
		leader.TrackCharmed(m.InstanceId, true)
	}
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880001: true, 880002: true, 880003: true, 880004: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	before := present.Character.Experience
	AwardCompanyXP(xpLeader, leader, 40, room)

	if got := present.Character.Experience - before; got != 40 {
		t.Errorf("present companion gained %d, want the full 40 (no split)", got)
	}
	if got := present2.Character.Experience - before; got != 40 {
		t.Errorf("second companion gained %d, want 40", got)
	}
	for name, m := range map[string]*mobs.Mob{"dead": dead, "away": away, "unattached": stranger} {
		if m.Character.Experience != before {
			t.Errorf("%s companion gained xp: %d", name, m.Character.Experience-before)
		}
	}
}

// TestAwardCompanyXPLevelsUp: a big award can cross several thresholds; the
// leader gets a line per level, and the companion keeps its health and mana
// (Phase 33h2: a level is no free rest).
func TestAwardCompanyXPLevelsUp(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	const room = 7002
	m := xpMob(t, 880011, room, 1)
	leader := characters.New()
	leader.TrackCharmed(m.InstanceId, true)
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880011: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	health, mana := m.Character.Health, m.Character.Mana
	_, lines := AwardCompanyXP(xpLeader, leader, m.Character.XPTL(3), room)

	if m.Character.Level < 3 {
		t.Fatalf("level = %d, want at least 3", m.Character.Level)
	}
	if len(lines) != m.Character.Level-1 {
		t.Fatalf("lines = %d, want one per level gained (%d)", len(lines), m.Character.Level-1)
	}
	if m.Character.StatPoints != 0 {
		t.Errorf("level points left unspent: %d", m.Character.StatPoints)
	}
	if m.Character.Health != health || m.Character.Mana != mana {
		t.Errorf("vitals changed by the level: %d/%d health, %d/%d mana, want %d and %d",
			m.Character.Health, m.Character.HealthMax.Value, m.Character.Mana, m.Character.ManaMax.Value, health, mana)
	}
	if m.Character.Health >= m.Character.HealthMax.Value {
		t.Fatalf("fixture: the companion must start below its new maximum (%d/%d)", m.Character.Health, m.Character.HealthMax.Value)
	}
	if !strings.Contains(lines[len(lines)-1], "reached level") {
		t.Errorf("line = %q", lines[len(lines)-1])
	}
}

// growthXPProvider also re-deals training (Phase 33h1) and records whom.
type growthXPProvider struct {
	xpProvider
	retrained *[]int
}

func (p growthXPProvider) RetrainCompanion(id int) bool {
	*p.retrained = append(*p.retrained, id)
	return true
}

// TestAwardCompanyXPRetrainsOnlyCompanionsThatLevelled: the company's
// growth deals a levelled companion's points instead of AutoTrain, and a
// companion that did not level is left alone.
func TestAwardCompanyXPRetrainsOnlyCompanionsThatLevelled(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	const room = 7004
	low := xpMob(t, 880031, room, 5)
	high := xpMob(t, 880032, room, 5)
	high.Character.Level = 30
	leader := characters.New()
	leader.TrackCharmed(low.InstanceId, true)
	leader.TrackCharmed(high.InstanceId, true)
	var retrained []int
	company.SetFormationProvider(growthXPProvider{xpProvider{attached: map[int]bool{880031: true, 880032: true}}, &retrained})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	paid, lines := AwardCompanyXP(xpLeader, leader, low.Character.XPTL(2), room)
	assert.Equal(t, 2, paid)
	require.NotEmpty(t, lines)
	assert.Equal(t, []int{880031}, retrained)
}

// TestAwardCompanyXPIgnoresForeignLeader: a mob attached to another leader
// is never paid by this leader's kill.
func TestAwardCompanyXPIgnoresForeignLeader(t *testing.T) {
	const room = 7003
	m := xpMob(t, 880021, room, 5)
	leader := characters.New()
	leader.TrackCharmed(m.InstanceId, true)
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880021: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	AwardCompanyXP(xpLeader+1, leader, 40, room)
	if m.Character.Experience > 1 {
		t.Fatalf("gained xp: %d", m.Character.Experience)
	}
}

// TestKillPaysTheCompanyThroughSuicide is the wiring test: a real kill,
// damaged only through the leader's credit (as a companion's hits are),
// pays the leader and the companion in the room the same figure, and not a
// companion left in another room.
func TestKillPaysTheCompanyThroughSuicide(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(xpLeader, 5151)
	user.Character.Level = 1
	users.SetTestUser(user)

	room := rooms.NewEmptyRoom()
	here := xpMob(t, 880031, room.RoomId, 5)
	away := xpMob(t, 880032, room.RoomId+1, 5)
	user.Character.TrackCharmed(here.InstanceId, true)
	user.Character.TrackCharmed(away.InstanceId, true)
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880031: true, 880032: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	foe := &mobs.Mob{MobId: 999}
	foe.Character.Name = "bandit"
	foe.Character.Level = 5
	foe.Character.TNLScale = 1
	foe.Character.PlayerDamage = map[int]int{xpLeader: 10}
	foe.InstanceId = 424243
	room.AddMob(foe.InstanceId)

	user.Character.RoomId = room.RoomId
	user.Character.Health = 10
	battle.Begin(user.UserId, room.RoomId, 1, "xp", []int{foe.InstanceId})
	t.Cleanup(battle.Reset)
	userBefore, hereBefore, awayBefore := user.Character.Experience, here.Character.Experience, away.Character.Experience
	ok, err := Suicide("", foe, room)
	require.NoError(t, err)
	require.True(t, ok)
	events.ProcessEvents()

	leaderGain := user.Character.Experience - userBefore
	require.Positive(t, leaderGain)
	assert.Equal(t, leaderGain, here.Character.Experience-hereBefore, "the companion here earns the leader's figure in full")
	assert.Equal(t, awayBefore, away.Character.Experience, "an absent companion earns nothing")
}

// TestAwardCompanyXPSkipsBefriendedAway: a companion another player has
// since charmed is no longer this leader's, whatever the leader's list says.
func TestAwardCompanyXPSkipsBefriendedAway(t *testing.T) {
	const room = 7004
	m := xpMob(t, 880041, room, 5)
	m.Character.Charm(xpLeader+9, -1, "")
	leader := characters.New()
	leader.TrackCharmed(m.InstanceId, true)
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880041: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	AwardCompanyXP(xpLeader, leader, 40, room)
	if m.Character.Experience > 1 {
		t.Fatalf("gained xp: %d", m.Character.Experience)
	}
}

// TestPartyKillPaysTheMembersCompany: in a GoMud party the split share goes
// to each member, and each member's companion earns that member's share.
func TestPartyKillPaysTheMembersCompany(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(xpLeader, 5151)
	user.Character.Level = 1
	users.SetTestUser(user)
	p := parties.New(xpLeader)
	require.NotNil(t, p)
	t.Cleanup(p.Disband)

	room := rooms.NewEmptyRoom()
	here := xpMob(t, 880051, room.RoomId, 5)
	user.Character.TrackCharmed(here.InstanceId, true)
	company.SetFormationProvider(xpProvider{attached: map[int]bool{880051: true}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	foe := &mobs.Mob{MobId: 999}
	foe.Character.Name = "bandit"
	foe.Character.Level = 5
	foe.Character.TNLScale = 1
	foe.Character.PlayerDamage = map[int]int{xpLeader: 10}
	foe.InstanceId = 424244
	room.AddMob(foe.InstanceId)

	user.Character.RoomId = room.RoomId
	user.Character.Health = 10
	battle.Begin(user.UserId, room.RoomId, 1, "xp", []int{foe.InstanceId})
	t.Cleanup(battle.Reset)
	userBefore, hereBefore := user.Character.Experience, here.Character.Experience
	ok, err := Suicide("", foe, room)
	require.NoError(t, err)
	require.True(t, ok)
	events.ProcessEvents()

	leaderGain := user.Character.Experience - userBefore
	require.Positive(t, leaderGain)
	assert.Equal(t, leaderGain, here.Character.Experience-hereBefore)
}
