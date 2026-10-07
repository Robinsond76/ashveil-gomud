package camping

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
)

// ritesStub is a company module with rites waiting.
type ritesStub struct {
	banterStub
	offers int
	text   string
}

func (s *ritesStub) OfferRites(int) string {
	s.offers++
	return s.text
}

var _ company.RitesProvider = (*ritesStub)(nil)

// Phase 74: a camp's rest and an inn's stay each announce the rites waiting,
// from their real starts, and a refused start announces nothing.
func TestACampRestAndAnInnStayOfferTheRites(t *testing.T) {
	stub := &ritesStub{text: "The company has not yet mourned:\n  Hild (lost for good)"}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return baseTime() })
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	text := module.startRest(user, eligibleRoom())
	assert.Contains(t, text, "Hild (lost for good)")
	assert.Equal(t, 1, stub.offers)
	// A rest that does not start offers nothing.
	assert.NotContains(t, module.startRest(user, eligibleRoom()), "Hild (lost for good)")
	assert.Equal(t, 1, stub.offers)

	e := newInnEnv(t)
	inn := campUser(t, 7, 2003)
	inn.Character.Gold = 9
	assert.NotContains(t, e.module.innRest(inn, innRoom()), "Hild")
	assert.Equal(t, 1, stub.offers, "no gold, no stay, no offer")
	inn.Character.Gold = 25
	assert.Contains(t, e.module.innRest(inn, innRoom()), "Hild (lost for good)")
	assert.Equal(t, 2, stub.offers)
}
