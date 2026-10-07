package livecompanions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoster []survival.MemberRef

func (f fakeRoster) Roster(int) []survival.MemberRef { return f }

type fakeFormation struct{ instances map[int]int }

func (f fakeFormation) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (f fakeFormation) InstanceFor(_, companionID int) (int, bool) {
	id, ok := f.instances[companionID]
	return id, ok
}
func (f fakeFormation) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}

func spawn(t *testing.T, instanceID, roomID int, health int) {
	t.Helper()
	mob := &mobs.Mob{InstanceId: instanceID, Character: *characters.New()}
	mob.Character.RoomId = roomID
	mob.Character.Health, mob.Character.HealthMax.Value = health, 10
	mobs.SetTestInstance(mob)
	t.Cleanup(func() { mobs.RemoveTestInstance(instanceID) })
}

func setup(t *testing.T) {
	t.Helper()
	survival.SetRosterProvider(fakeRoster{
		{Key: survival.LeaderMemberKey, Name: "Leader"},
		{Key: survival.CompanionMemberKey(1), Name: "Here"},
		{Key: survival.CompanionMemberKey(2), Name: "Dead", Dead: true},
		{Key: survival.CompanionMemberKey(3), Name: "Away", Away: true},
		{Key: survival.CompanionMemberKey(4), Name: "Golem", Needless: true},
		{Key: survival.CompanionMemberKey(5), Name: "Elsewhere"},
		{Key: survival.CompanionMemberKey(6), Name: "Downed"},
		{Key: survival.CompanionMemberKey(7), Name: "Unspawned"},
	})
	company.SetFormationProvider(fakeFormation{instances: map[int]int{1: 9101, 2: 9102, 4: 9104, 5: 9105, 6: 9106}})
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
	})
	spawn(t, 9101, 50, 10)
	spawn(t, 9102, 50, 10)
	spawn(t, 9104, 50, 10)
	spawn(t, 9105, 51, 10)
	spawn(t, 9106, 50, 0) // dead on its feet: IsDisabled
}

func ids(cs []Companion) []int {
	var out []int
	for _, c := range cs {
		out = append(out, c.CompanionID)
	}
	return out
}

func TestOfAppliesTheCallersFilter(t *testing.T) {
	setup(t)

	all, ok := Of(1, nil)
	require.True(t, ok)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7}, ids(all), "the leader's own key is never a companion")
	require.NotNil(t, all[0].Mob)
	assert.Nil(t, all[2].Mob, "a companion without an instance has no mob")
	assert.Nil(t, all[6].Mob)

	living, _ := Of(1, SkipDead)
	assert.Equal(t, []int{1, 3, 4, 5, 6, 7}, ids(living))
	active, _ := Of(1, SkipInactive)
	assert.Equal(t, []int{1, 5, 6, 7}, ids(active), "dead, separated and construct companions are dropped")
}

func TestOfReportsNoRosterProvider(t *testing.T) {
	survival.SetRosterProvider(nil)
	got, ok := Of(1, nil)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestAbleNeedsASpawnedUpCompanionInTheRoom(t *testing.T) {
	setup(t)

	assert.Equal(t, []int{1, 2, 4, 5}, ids(Able(1, nil)), "a downed or unspawned companion is out")
	here := Able(1, func(roomID int) bool { return roomID == 50 })
	assert.Equal(t, []int{1, 2, 4}, ids(here), "a companion in another room is out")
}
