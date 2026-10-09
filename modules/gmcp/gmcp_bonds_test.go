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

// bondCompany is a company module that can read its companions' bonds.
type bondCompany struct{ panel company.BondPanel }

func (o *bondCompany) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (o *bondCompany) InstanceFor(int, int) (int, bool)           { return 0, false }
func (o *bondCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (o *bondCompany) CampBanter(int, string) []banter.Said { return nil }
func (o *bondCompany) LastBanter(int) []banter.Said         { return nil }
func (o *bondCompany) BondPanel(int) (company.BondPanel, bool) {
	return o.panel, true
}

// Phase 65: Company.Bonds reaches the web client, once, and again only when
// a bond changes.
func TestBondsAreSentWhenTheyChange(t *testing.T) {
	stub := &bondCompany{panel: company.BondPanel{
		Pairs: []company.BondRow{{A: 1, B: 2, AName: "Maren", BName: "Hob", Value: 34, Tier: 1, Phrase: "Maren and Hob trust each other", Effect: "Each steps in once a battle for the other when hurt (at 50% health or less)."}},
		Members: []company.BondMember{
			{ID: 1, Name: "Maren", Feelings: []company.BondFeeling{{ID: 2, Name: "Hob", Words: "trusts Hob", Tier: 1}}},
			{ID: 2, Name: "Hob", Feelings: []company.BondFeeling{{ID: 1, Name: "Maren", Words: "trusts Maren", Tier: 1}}},
		}}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	f, out := testFeed()
	f.extras = []companyExtra{bondsExtra()}
	u := users.NewUserRecord(7, 1)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Bonds", (*out)[0].module)
	var got company.BondPanel
	raw, err := json.Marshal((*out)[0].body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Len(t, got.Pairs, 1)
	assert.Equal(t, "Maren and Hob trust each other", got.Pairs[0].Phrase)
	assert.Equal(t, "trusts Hob", got.Members[0].Feelings[0].Words)

	f.updateExtras(u)
	assert.Len(t, *out, 1, "unchanged bonds are not resent")
	stub.panel.Pairs[0].Value = 37
	f.updateExtras(u)
	assert.Len(t, *out, 2, "a changed bond is")
}

func TestNoBondsAreSentWithoutACompanyModule(t *testing.T) {
	company.SetFormationProvider(nil)
	f, out := testFeed()
	f.extras = []companyExtra{bondsExtra()}
	f.updateExtras(users.NewUserRecord(7, 1))
	assert.Empty(t, *out)
}
