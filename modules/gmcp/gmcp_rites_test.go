package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// riteCompany is a company module that can read its rites.
type riteCompany struct{ panel company.RitePanel }

func (o *riteCompany) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (o *riteCompany) InstanceFor(int, int) (int, bool)           { return 0, false }
func (o *riteCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (o *riteCompany) CampBanter(int, string) []banter.Said { return nil }
func (o *riteCompany) LastBanter(int) []banter.Said         { return nil }
func (o *riteCompany) RitePanel(int) (company.RitePanel, bool) {
	return o.panel, true
}

// Phase 74: Company.Rites reaches the web client once, and again only when
// a rite is queued, answered or its place changes.
func TestRitesAreSentWhenTheyChange(t *testing.T) {
	stub := &riteCompany{panel: company.RitePanel{Here: true, Where: "You can hold rites here.", Rows: []company.RiteRow{
		{ID: 1, Name: "Hild", Level: 5, Cause: "lost for good", Close: []string{"Tamsin"}},
	}}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	f, out := testFeed()
	f.extras = []companyExtra{ritesExtra()}
	u := users.NewUserRecord(7, 1)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Rites", (*out)[0].module)
	var got company.RitePanel
	raw, err := json.Marshal((*out)[0].body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Len(t, got.Rows, 1)
	assert.Equal(t, "Hild", got.Rows[0].Name)
	assert.Equal(t, []string{"Tamsin"}, got.Rows[0].Close)

	f.updateExtras(u)
	assert.Len(t, *out, 1, "unchanged, not resent")
	stub.panel.Rows = nil
	f.updateExtras(u)
	require.Len(t, *out, 2)
}
