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

// errandCompany is a company module that can read its companions' errands.
type errandCompany struct{ panel company.ErrandPanel }

func (o *errandCompany) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (o *errandCompany) InstanceFor(int, int) (int, bool)           { return 0, false }
func (o *errandCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (o *errandCompany) CampBanter(int, string) []banter.Said { return nil }
func (o *errandCompany) LastBanter(int) []banter.Said         { return nil }
func (o *errandCompany) ErrandPanel(int) (company.ErrandPanel, bool) {
	return o.panel, true
}

// Phase 70: Company.Errands reaches the web client once, and again only when
// an errand starts, ends or comes due, not as its countdown ticks.
func TestErrandsAreSentWhenTheyChange(t *testing.T) {
	stub := &errandCompany{panel: company.ErrandPanel{
		Now: 1000, Here: true, Where: "You can send companions from here.", Zone: "Brindle Downs", Band: "7-9",
		Rows: []company.ErrandRow{
			{ID: 2, Name: "Ysolde", Level: 8, State: "away", Kind: "hunt", KindLabel: "a hunt", ReturnsAt: 8200, Remaining: 7200},
			{ID: 3, Name: "Tamsin", Level: 7, State: "ready"},
		},
		Options: company.ErrandOptions(), Recent: []string{},
	}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	f, out := testFeed()
	f.extras = []companyExtra{errandsExtra()}
	u := users.NewUserRecord(7, 1)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Errands", (*out)[0].module)
	var got company.ErrandPanel
	raw, err := json.Marshal((*out)[0].body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Len(t, got.Rows, 2)
	assert.Equal(t, int64(8200), got.Rows[0].ReturnsAt)
	assert.Equal(t, "ready", got.Rows[1].State)
	require.Len(t, got.Options, 3)
	assert.Equal(t, "short", got.Options[0].Lengths[0].Length)

	stub.panel.Now = 1030
	stub.panel.Rows[0].Remaining = 7170
	f.updateExtras(u)
	assert.Len(t, *out, 1, "a ticking countdown is not a change")
	stub.panel.Rows[0].Due = true
	f.updateExtras(u)
	assert.Len(t, *out, 2, "an errand coming due is")
	stub.panel.Rows[0].State = "ready"
	f.updateExtras(u)
	assert.Len(t, *out, 3, "and so is its return")
}

func TestNoErrandsAreSentWithoutACompanyModule(t *testing.T) {
	company.SetFormationProvider(nil)
	f, out := testFeed()
	f.extras = []companyExtra{errandsExtra()}
	f.updateExtras(users.NewUserRecord(7, 1))
	assert.Empty(t, *out)
}
