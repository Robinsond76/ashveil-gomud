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

// opinionCompany is a company module that can read its companions' opinions.
type opinionCompany struct{ panel company.OpinionPanel }

func (o *opinionCompany) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (o *opinionCompany) InstanceFor(int, int) (int, bool) { return 0, false }
func (o *opinionCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (o *opinionCompany) CampBanter(int, string) []banter.Said { return nil }
func (o *opinionCompany) LastBanter(int) []banter.Said         { return nil }
func (o *opinionCompany) OpinionPanel(int) (company.OpinionPanel, bool) {
	return o.panel, true
}

// Phase 64: Company.Opinions reaches the web client, once, and again only
// when it changes.
func TestOpinionsAreSentWhenTheyChange(t *testing.T) {
	stub := &opinionCompany{panel: company.OpinionPanel{Spared: 2, Members: []company.OpinionRow{{
		Key: "companion:1", ID: 1, Name: "Maren", Personality: "devout", Loyalty: 64, Mood: "loyal",
		Likes: []string{"mercy to the beaten"}, Dislikes: []string{"selling relics"},
	}}}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	f, out := testFeed()
	f.extras = []companyExtra{opinionsExtra()}
	u := users.NewUserRecord(7, 1)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Opinions", (*out)[0].module)
	var got company.OpinionPanel
	raw, err := json.Marshal((*out)[0].body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Len(t, got.Members, 1)
	assert.Equal(t, "Maren", got.Members[0].Name)
	assert.Equal(t, 2, got.Spared)

	f.updateExtras(u)
	assert.Len(t, *out, 1, "unchanged opinions are not resent")
	stub.panel.Members[0].Loyalty = 66
	f.updateExtras(u)
	assert.Len(t, *out, 2, "a loyalty change is")
}

func TestNoOpinionsAreSentWithoutACompanyModule(t *testing.T) {
	company.SetFormationProvider(nil)
	f, out := testFeed()
	f.extras = []companyExtra{opinionsExtra()}
	f.updateExtras(users.NewUserRecord(7, 1))
	assert.Empty(t, *out)
}
