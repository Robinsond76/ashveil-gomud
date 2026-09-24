package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChemistryStanding struct {
	view company.ChemistryStandingView
	ok   bool
}

func (fakeChemistryStanding) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (fakeChemistryStanding) InstanceFor(int, int) (int, bool) { return 0, false }
func (fakeChemistryStanding) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}
func (f fakeChemistryStanding) ChemistryHitBonus(int, company.MemberKey) int { return f.view.Bonus }
func (f fakeChemistryStanding) ChemistryStanding(int, company.MemberKey) (company.ChemistryStandingView, bool) {
	return f.view, f.ok
}

func statusBonusesText(t *testing.T, user *users.UserRecord) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Status(`bonuses`, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return strings.Join(*messages, "\n")
}

// TestStatusBonusesShowsChemistry drives the real status command: company
// chemistry has its own box, apart from buffs.
func TestStatusBonusesShowsChemistry(t *testing.T) {
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	user := users.NewUserRecord(7, 1)

	company.SetFormationProvider(nil)
	text := statusBonusesText(t, user)
	assert.Contains(t, text, `Company Chemistry`)
	assert.Contains(t, text, `No bond yet`)

	company.SetFormationProvider(fakeChemistryStanding{ok: true, view: company.ChemistryStandingView{Tier: company.TierTrusted, Partner: `Tamsin (#1)`, Bonus: 4}})
	text = statusBonusesText(t, user)
	assert.Contains(t, text, `Trusted`)
	assert.Contains(t, text, `Tamsin (#1)`)
	assert.Contains(t, text, `+4%`)

	company.SetFormationProvider(fakeChemistryStanding{ok: true, view: company.ChemistryStandingView{Tier: company.TierSworn, Partner: `Tamsin (#1)`}})
	assert.Contains(t, statusBonusesText(t, user), `not at your side`)
}
