package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// banterStub is a company module that has something to say at camp.
type banterStub struct {
	asked []string
	said  []banter.Said
}

func (s *banterStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (s *banterStub) InstanceFor(int, int) (int, bool)           { return 0, false }
func (s *banterStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (s *banterStub) CampBanter(_ int, context string) []banter.Said {
	s.asked = append(s.asked, context)
	return s.said
}
func (s *banterStub) LastBanter(int) []banter.Said { return nil }

// Phase 49: the company talks as a rest begins and as it ends, in the
// rest's own text.
func TestRestBanterAtStartAndEnd(t *testing.T) {
	stub := &banterStub{said: []banter.Said{
		{Member: 1, Name: "Hild", Verb: "mutters", Text: "Keep the fire low."},
		{Member: 2, Name: "Brann", Verb: "remarks", Text: "Cheerful as ever."},
	}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	now := baseTime()
	scheduler := &fakeScheduler{}
	module := newTestModule(&fakeStore{}, scheduler, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 100)
	messages := captureMessages(t)

	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	text := module.startRest(user, eligibleRoom())
	assert.Contains(t, text, "You settle in by the fire to rest.")
	assert.Contains(t, text, `Hild</ansi> mutters, "Keep the fire low."`)
	assert.Contains(t, text, `Brann</ansi> remarks, "Cheerful as ever."`)

	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	events.ProcessEvents()
	end := strings.Join(*messages, "")
	assert.Contains(t, end, "feels rested")
	assert.Contains(t, end, `Hild</ansi> mutters, "Keep the fire low."`, "and again as the rest ends")
	assert.Equal(t, []string{banter.CtxCamp, banter.CtxRested}, stub.asked)
}

func TestRestWithoutBanterIsUnchanged(t *testing.T) {
	stub := &banterStub{}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	text := module.startRest(user, eligibleRoom())
	assert.NotContains(t, text, "mutters")
	assert.NotContains(t, text, "\n\n\n")
}
