package camping

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// opinionStub is a company module with opinions about the camp.
type opinionStub struct {
	banterStub
	choices []opinions.Choice
	said    []string
}

func (s *opinionStub) Opinion(_ int, c opinions.Choice) ([]string, error) {
	s.choices = append(s.choices, c)
	return s.said, nil
}

var _ company.OpinionProvider = (*opinionStub)(nil)

// Phase 64: a rough camp's rest and an inn's stay are choices the company
// has a view on, reported once each from their real starts.
func TestARoughCampRestReportsTheChoice(t *testing.T) {
	stub := &opinionStub{said: []string{`Hild mutters, "Dirt and a wall of dark."`}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return baseTime() })
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	text := module.startRest(user, eligibleRoom())
	require.Len(t, stub.choices, 1)
	assert.Equal(t, opinions.Rough, stub.choices[0].Kind)
	assert.Contains(t, text, `Hild mutters, "Dirt and a wall of dark."`)
	// A rest that does not start is no choice.
	stub.choices = nil
	assert.NotContains(t, module.startRest(user, eligibleRoom()), "Hild mutters")
	assert.Empty(t, stub.choices)
}

func TestAnInnStayReportsTheChoiceAndARefusalDoesNot(t *testing.T) {
	stub := &opinionStub{said: []string{`Brann remarks, "A bed. How decadent of us."`}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 9
	assert.Contains(t, e.module.innRest(user, innRoom()), "only 9")
	assert.Empty(t, stub.choices, "no gold, no stay, no opinion")
	user.Character.Gold = 25
	text := e.module.innRest(user, innRoom())
	assert.Contains(t, text, "You pay 10 gold")
	assert.Contains(t, text, "A bed. How decadent of us.")
	require.Len(t, stub.choices, 1)
	assert.Equal(t, opinions.Inn, stub.choices[0].Kind)
}
