package expedition

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// companyRelocations records Phase 33h3 relocations at the company seam.
type companyRelocations struct {
	calls [][3]int
}

func (f *companyRelocations) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (f *companyRelocations) InstanceFor(int, int) (int, bool) { return 0, false }
func (f *companyRelocations) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}
func (f *companyRelocations) RelocateCompany(leaderUserID, originRoomID, roomID int) int {
	f.calls = append(f.calls, [3]int{leaderUserID, originRoomID, roomID})
	return 2
}

func useCompanyRelocations(t *testing.T) *companyRelocations {
	t.Helper()
	fake := &companyRelocations{}
	company.SetFormationProvider(fake)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	return fake
}

func arrivalLines(t *testing.T) *[]string {
	t.Helper()
	lines := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		lines = append(lines, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &lines
}

// Phase 33h3: arrival brings the company from the origin with the leader,
// once, after the leader's move is verified, instead of telling them to
// walk the route's exit.
func TestArrivalBringsTheCompanyWithTheLeader(t *testing.T) {
	relocations := useCompanyRelocations(t)
	lines := arrivalLines(t)
	user := travelUser(t, 7, 100)
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	now := baseTime()
	module := newTestModule(&fakeStore{}, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)
	events.ProcessEvents()

	assert.Equal(t, []int{200}, mover.moves)
	assert.Equal(t, [][3]int{{7, 100, 200}}, relocations.calls)
	text := strings.Join(*lines, "\n")
	assert.Contains(t, text, "You have reached")
	assert.Contains(t, text, company.CompanyFollows)
}

// A failed move relocates nobody; the crash recovery's retry brings the
// company along with the leader.
func TestArrivalRecoveryBringsTheCompany(t *testing.T) {
	relocations := useCompanyRelocations(t)
	user := travelUser(t, 7, 100)
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user, err: assert.AnError}
	now := baseTime()
	module := newTestModule(&fakeStore{}, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)
	assert.Empty(t, relocations.calls, "the leader didn't move")

	mover.err = nil
	module.mu.Lock()
	module.recoverLocked()
	module.mu.Unlock()
	assert.Equal(t, [][3]int{{7, 100, 200}}, relocations.calls)
}
