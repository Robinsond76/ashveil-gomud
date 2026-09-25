package gmcp

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

func need(v int, label string) companyview.Need {
	return companyview.Need{Known: true, Value: v, Label: label}
}

// sampleCompany: a leader in the front-left cell, a present companion, an
// awaiting one, and a dead one, burdened and resting.
func sampleCompany() companyview.Summary {
	return companyview.Summary{
		Leader: companyview.Member{Key: company.LeaderMemberKey, Leader: true, Name: "Wren", Level: 5, Archetype: "Ranger", ArchetypeKnown: true,
			HasHP: true, HP: 30, HPMax: 40, Hunger: need(40, "Hungry"), Thirst: need(90, "Hydrated"), Fatigue: need(80, "Rested"),
			WarmthKnown: true, Placed: true},
		CompanyKnown: true,
		Companions: []companyview.Member{
			{Key: company.CompanionMemberKey(1), ID: 1, Name: "Bran", Status: company.MemberPresent, Level: 3, Archetype: "Warrior",
				HasHP: true, HP: 12, HPMax: 25, Hunger: need(70, "Sated"), Thirst: need(70, "Comfortable"), Fatigue: need(70, "Ready"),
				Placed: true, Row: 0, Col: 1},
			{Key: company.CompanionMemberKey(2), ID: 2, Name: "Bran", Status: company.MemberAwaiting, Level: 2},
			{Key: company.CompanionMemberKey(3), ID: 3, Name: "<b>Ysolde</b>", Status: company.MemberDead, Level: 5, RescueLeft: 90 * time.Minute},
		},
		Alive: 3, Dead: 1,
		LoadKnown: true, Load: encumbrance.Load{PersonalGrams: 5000, CargoGrams: 3000, CapacityGrams: 10000}, LoadLabel: "Burdened",
		ActivityKnown: true, Activity: companyview.Activity{Kind: companyview.CampRest, Remaining: 12 * time.Minute},
		RestKnown: true, RestTier: camping.TierRested, RestLeft: time.Hour,
		Checkpoint: "The Chapel of the Wayfarer",
	}
}

func noChemistry(int, company.MemberKey) (company.ChemistryStandingView, bool) {
	return company.ChemistryStandingView{}, false
}

func trustedChemistry(int, company.MemberKey) (company.ChemistryStandingView, bool) {
	return company.ChemistryStandingView{Together: 2, Tier: company.TierTrusted, Bonus: 4}, true
}

func TestCompanyPayloadShape(t *testing.T) {
	p, ok := buildCompanyPayload(7, sampleCompany(), trustedChemistry)
	require.True(t, ok)
	data, err := json.Marshal(p)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	leader := got["leader"].(map[string]any)
	assert.Equal(t, "leader", leader["key"])
	assert.Equal(t, "Ranger", leader["archetype"])
	assert.Equal(t, map[string]any{"row": 0.0, "col": 0.0}, leader["cell"])
	assert.Equal(t, "Trusted", leader["chemistry"])

	members := got["members"].([]any)
	require.Len(t, members, 3)
	bran := members[0].(map[string]any)
	assert.Equal(t, "companion:1", bran["key"])
	assert.Equal(t, "present", bran["status"])
	assert.Equal(t, map[string]any{"row": 0.0, "col": 1.0}, bran["cell"])
	assert.NotContains(t, bran, "rescue_seconds", "countdowns live in the live half")
	awaiting := members[1].(map[string]any)
	assert.Equal(t, "awaiting", awaiting["status"])
	assert.Nil(t, awaiting["cell"])
	dead := members[2].(map[string]any)
	assert.Equal(t, "dead", dead["status"])
	assert.Equal(t, map[string]any{"companion:3": 5400.0}, got["rescue"])
	assert.Nil(t, dead["chemistry"], "the dead have no band")
	assert.Equal(t, "<b>Ysolde</b>", dead["name"], "names travel as data; the client renders them as text")

	vitals := got["vitals"].(map[string]any)
	lv := vitals["leader"].(map[string]any)
	assert.Equal(t, 30.0, lv["hp"])
	assert.Equal(t, "", lv["warmth"], "known comfortable")
	assert.Equal(t, map[string]any{"value": 40.0, "label": "Hungry", "warn": true}, lv["needs"].(map[string]any)["hunger"])
	av := vitals["companion:2"].(map[string]any)
	assert.Nil(t, av["hp"], "not present: no health")
	assert.Nil(t, av["needs"], "unknown needs are null")
	assert.Nil(t, av["warmth"])

	assert.Equal(t, 3.0, got["alive"])
	assert.Equal(t, 1.0, got["dead"])
	assert.Equal(t, map[string]any{"label": "Burdened", "total_g": 8000.0, "capacity_g": 10000.0, "cargo_g": 3000.0}, got["load"])
	assert.Equal(t, "Resting 12m", got["activity"])
	assert.Equal(t, map[string]any{"tier": "Rested", "seconds": 3600.0}, got["rest"])
	assert.Equal(t, "The Chapel of the Wayfarer", got["checkpoint"])
}

func TestCompanyPayloadUnknowns(t *testing.T) {
	_, ok := buildCompanyPayload(7, companyview.Summary{Alive: 1}, noChemistry)
	assert.False(t, ok, "no company data")

	s := companyview.Summary{CompanyKnown: true, Alive: 1, Leader: companyview.Member{Key: company.LeaderMemberKey, Leader: true, Name: "Wren"}}
	p, ok := buildCompanyPayload(7, s, noChemistry)
	require.True(t, ok)
	data, _ := json.Marshal(p)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	for _, key := range []string{"load", "activity", "rest", "checkpoint"} {
		assert.Nil(t, got[key], key)
	}
	leader := got["leader"].(map[string]any)
	assert.Nil(t, leader["archetype"], "no archetype provider")
	assert.Nil(t, leader["chemistry"])
	assert.Equal(t, []any{}, got["members"])
}

type sent struct {
	userID int
	module string
	body   map[string]any
}

func testFeed() (*companyFeed, *[]sent) {
	out := &[]sent{}
	f := newCompanyFeed()
	f.chemistry = noChemistry
	f.accepting = func(int) bool { return true }
	f.send = func(userID int, module string, payload []byte) {
		var body map[string]any
		_ = json.Unmarshal(payload, &body)
		*out = append(*out, sent{userID, module, body})
	}
	return f, out
}

func TestCompanySendsOnlyOnChange(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company", (*out)[0].module)
	assert.Contains(t, (*out)[0].body, "vitals", "the snapshot carries the vitals")
	f.update(7, s)
	assert.Len(t, *out, 1, "unchanged: nothing sent")

	s.Companions = s.Companions[:2] // roster change
	f.update(7, s)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company", (*out)[1].module)
}

func TestCompanyVitalsOnlyWhenOnlyVitalsChange(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	s.Companions[0].HP = 5
	f.update(7, s)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company.Vitals", (*out)[1].module)
	vitals := (*out)[1].body["vitals"].(map[string]any)
	assert.Equal(t, 5.0, vitals["companion:1"].(map[string]any)["hp"])
	assert.NotContains(t, (*out)[1].body, "members", "vitals only")
}

func TestCompanyResentAfterForget(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	f.forget(7) // login, copyover, or a request
	f.update(7, s)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company", (*out)[1].module)
}

func TestCompanyUnreadableSendsEmptyOnce(t *testing.T) {
	f, out := testFeed()
	f.update(7, companyview.Summary{Alive: 1})
	f.update(7, companyview.Summary{Alive: 1})
	require.Len(t, *out, 1)
	assert.Equal(t, "Company", (*out)[0].module)
	assert.Empty(t, (*out)[0].body)
}

func TestCompanyOnlyToLeader(t *testing.T) {
	f, out := testFeed()
	f.update(7, sampleCompany())
	f.update(8, companyview.Summary{CompanyKnown: true, Alive: 1, Leader: companyview.Member{Key: company.LeaderMemberKey, Name: "Other"}})
	require.Len(t, *out, 2)
	assert.Equal(t, 7, (*out)[0].userID)
	assert.Equal(t, 8, (*out)[1].userID)
	assert.Equal(t, "Other", (*out)[1].body["leader"].(map[string]any)["name"], "each gets their own")
}

// TestCompanyWebRequest: the web client's !!GMCP(Company) queues a request
// for the connection's own user.
func TestCompanyWebRequest(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(41, 4141)
	users.SetTestUser(u)
	var got []int
	id := events.RegisterListener(GMCPCompanyRequest{}, func(e events.Event) events.ListenerReturn {
		got = append(got, e.(GMCPCompanyRequest).UserId)
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(GMCPCompanyRequest{}, id) })
	assert.True(t, gmcpModule.HandleWebGMCP(u.ConnectionId(), []byte("!!GMCP(Company)")))
	events.ProcessEvents()
	assert.Equal(t, []int{41}, got)
}

// TestCountdownsDontResendSnapshot (review finding 1): a ticking rescue or
// rest timer changes Company.Vitals once a minute, never the snapshot.
func TestCountdownsDontResendSnapshot(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	for i := 1; i <= 15; i++ { // 15 rounds of 4 seconds
		s.Companions[2].RescueLeft -= 4 * time.Second
		s.RestLeft -= 4 * time.Second
		f.update(7, s)
	}
	for _, m := range (*out)[1:] {
		assert.Equal(t, "Company.Vitals", m.module, "no snapshot for a countdown")
	}
	assert.LessOrEqual(t, len(*out)-1, 2, "at most one update per minute boundary")
	last := (*out)[len(*out)-1].body
	assert.Equal(t, map[string]any{"companion:3": 5340.0}, last["rescue"], "whole minutes")
	assert.Equal(t, 3540.0, last["rest"].(map[string]any)["seconds"])
}

func TestWarmthOnlySendsVitals(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	s.Leader.Warmth = "Chilled"
	f.update(7, s)
	require.Len(t, *out, 2)
	assert.Equal(t, "Company.Vitals", (*out)[1].module)
}

func TestCheckpointAndChemistrySendSnapshot(t *testing.T) {
	f, out := testFeed()
	s := sampleCompany()
	f.update(7, s)
	s.Checkpoint = "The Sanctuary"
	f.update(7, s)
	f.chemistry = trustedChemistry
	f.update(7, s)
	require.Len(t, *out, 3)
	assert.Equal(t, "Company", (*out)[1].module)
	assert.Equal(t, "Company", (*out)[2].module)
}

// TestChemistryOnlyInABand (review finding 5): alone, no chemistry.
func TestChemistryOnlyInABand(t *testing.T) {
	alone := func(int, company.MemberKey) (company.ChemistryStandingView, bool) {
		return company.ChemistryStandingView{Together: 1}, true
	}
	p, _ := buildCompanyPayload(7, sampleCompany(), alone)
	assert.Nil(t, p.Leader.Chemistry)
	p, _ = buildCompanyPayload(7, sampleCompany(), trustedChemistry)
	assert.Equal(t, "Trusted", *p.Leader.Chemistry)
}

// TestNothingForNonGMCPConnections (review finding 9): a telnet client that
// hasn't accepted GMCP gets nothing built; once it does, the snapshot.
func TestNothingForNonGMCPConnections(t *testing.T) {
	f, out := testFeed()
	accepting := false
	f.accepting = func(int) bool { return accepting }
	f.update(7, sampleCompany())
	assert.Empty(t, *out)
	accepting = true
	f.update(7, sampleCompany())
	require.Len(t, *out, 1)
	assert.Equal(t, "Company", (*out)[0].module)
}

// TestPruneAndDespawn (review finding 8): users gone offline are dropped,
// and PlayerDespawn forgets.
func TestPruneAndDespawn(t *testing.T) {
	f, _ := testFeed()
	f.update(7, sampleCompany())
	f.update(8, sampleCompany())
	f.prune([]int{8})
	f.mu.Lock()
	_, has7 := f.last[7]
	_, has8 := f.last[8]
	f.mu.Unlock()
	assert.False(t, has7)
	assert.True(t, has8)

	companyFeeds.update(9, sampleCompany())
	events.AddToQueue(events.PlayerDespawn{UserId: 9})
	events.ProcessEvents()
	companyFeeds.mu.Lock()
	_, has9 := companyFeeds.last[9]
	companyFeeds.mu.Unlock()
	assert.False(t, has9)
}
