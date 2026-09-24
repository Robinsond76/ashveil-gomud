package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/stretchr/testify/assert"
)

type innStandingStub struct{ company int }

func (s innStandingStub) For(_ int, zone string) (standing.Standing, bool) {
	if zone != "Dunmar" {
		return standing.Standing{}, false
	}
	return standing.Assess(s.company, 40, standing.DefaultRules()), true
}

func useInnStanding(t *testing.T, company int) {
	t.Helper()
	standing.SetProvider(innStandingStub{company: company})
	t.Cleanup(func() { standing.SetProvider(nil) })
}

func TestInnStatusShowsDistrustedPrice(t *testing.T) {
	useInnStanding(t, -60)
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 42
	text := e.module.innStatus(user, innRoom())
	assert.Contains(t, text, "15 gold (5 per member)", "10 gold, 50% more")
	assert.Contains(t, text, "Your company is distrusted here, so the room costs 50% more.")
}

func TestInnRestChargesMarkup(t *testing.T) {
	useInnStanding(t, -60)
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 14
	assert.Contains(t, e.module.innRest(user, innRoom()), "costs 15 gold, and you have only 14")
	user.Character.Gold = 25
	assert.Contains(t, e.module.innRest(user, innRoom()), "You pay 15 gold")
	assert.Equal(t, 10, user.Character.Gold)
	assert.Equal(t, 15, e.store.saved.Stays[7].Paid)
}

func TestInnRestShunnedRefusedWithoutGold(t *testing.T) {
	useInnStanding(t, -100)
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 100
	assert.Contains(t, e.module.innStatus(user, innRoom()), "won't give your company a room")
	assert.Contains(t, e.module.innRest(user, innRoom()), "won't give your company a room")
	assert.Equal(t, 100, user.Character.Gold)
	assert.Empty(t, e.module.stays)
	assert.Zero(t, e.store.saveCalls)
}

func TestInnWelcomeAndUnconfiguredAtNormalPrice(t *testing.T) {
	useInnStanding(t, 40)
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	text := e.module.innStatus(user, innRoom())
	assert.Contains(t, text, "10 gold (5 per member)")
	assert.NotContains(t, text, "more")
}
