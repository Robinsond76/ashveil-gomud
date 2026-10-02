package combat

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"testing"
)

type fatigueService struct{ rest int }

func (s *fatigueService) ApplyCompanyExertion(int, string, survival.Exertion) ([]survival.ExertionResult, error) {
	return nil, nil
}
func (s *fatigueService) ApplyCompanyRestRecovery(int, string, int) ([]survival.ExertionResult, error) {
	return nil, nil
}
func (s *fatigueService) CompanyNeeds(int) []survival.MemberNeeds {
	return []survival.MemberNeeds{{Key: company.LeaderMemberKey, Needs: survival.Needs{Fatigue: s.rest}}}
}

func TestFatigueReducesRealAttackHitChanceAndReportsPenalty(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	service := &fatigueService{rest: 100}
	survival.SetCompanyService(service)
	t.Cleanup(func() { survival.SetCompanyService(nil) })
	room := &rooms.Room{RoomId: 90019, Zone: "Test", Biome: "city", Tags: []string{rooms.TagLit}}
	rested := countHits(t, room, 3000)
	service.rest = 0
	collapsed := countHits(t, room, 3000)
	assert.Less(t, collapsed, rested*85/100, "rested %d, collapsed %d", rested, collapsed)
	u := users.NewUserRecord(9191, 9191)
	u.Character.RoomId = room.RoomId
	u.Character.RaceId = 1
	u.Character.SetAggro(0, 9292, characters.DefaultAttack, 0)
	m := &mobs.Mob{InstanceId: 9292}
	m.Character = *characters.New()
	m.Character.RoomId = room.RoomId
	m.Character.Health = 1000000
	m.Character.HealthMax.Value = 1000000
	r := AttackPlayerVsMob(u, m)
	assert.Contains(t, r.MessagesToSource[0], "fatigue: hit -20%")
	survival.SetCompanyService(nil)
	assert.Zero(t, fatigueFor(u.UserId, company.LeaderMemberKey), "unavailable survival is neutral")
}
