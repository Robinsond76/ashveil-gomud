package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type banterProviderStub struct{ last []banter.Said }

func (s *banterProviderStub) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (s *banterProviderStub) InstanceFor(int, int) (int, bool) { return 0, false }
func (s *banterProviderStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (s *banterProviderStub) CampBanter(int, string) []banter.Said { return nil }
func (s *banterProviderStub) LastBanter(int) []banter.Said         { return s.last }

// Phase 49: Company.Camp carries the company's latest exchange for the
// Camp tab, nothing before the first, and resends when a new one is had.
func TestCompanyCampPayloadCarriesBanter(t *testing.T) {
	stub := &banterProviderStub{}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	state := camping.CampState{HasCamp: true, Here: true, FireLit: true}
	extra := campExtra(func(int, int, []string) (camping.CampState, bool) { return state, true }, nil)
	u := users.NewUserRecord(7, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	_, has := got["banter"]
	assert.False(t, has, "no banter key before the first exchange")

	stub.last = []banter.Said{
		{Member: 1, Name: "Hild Marrow", Verb: "mutters", Text: "Keep the fire low."},
		{Member: 2, Name: "Brann", Verb: "remarks", Text: "Cheerful as ever."},
	}
	require.NoError(t, json.Unmarshal(extra.build(u), &got))
	assert.Equal(t, []any{
		map[string]any{"name": "Hild Marrow", "verb": "mutters", "text": "Keep the fire low."},
		map[string]any{"name": "Brann", "verb": "remarks", "text": "Cheerful as ever."},
	}, got["banter"])

	f, out := testFeed()
	f.extras = []companyExtra{extra}
	f.updateExtras(u)
	f.updateExtras(u)
	stub.last = stub.last[:1]
	f.updateExtras(u)
	require.Len(t, *out, 2, "sent when it changes: the first time, and with the next exchange")
}
