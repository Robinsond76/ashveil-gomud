package company

import (
	"errors"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hiringRoom = 2003

func testRecruiters() map[int]recruiter {
	return map[int]recruiter{hiringRoom: {RoomID: hiringRoom, Name: "the hiring slate", Candidates: []candidate{
		{ID: "tamsin", MobTemplateID: 961, Tutorial: true, Price: 500},
		{ID: "garrick", MobTemplateID: 963, Price: 120},
	}}}
}

type userSaves struct {
	calls int
	err   error
}

func newRecruitModule(t *testing.T, gold int) (*CompanyModule, *fakeRuntime, *users.UserRecord, *userSaves) {
	t.Helper()
	useFakeLifecycle(t, &fakeLifecycle{})
	tpl := geared(1, 919002)
	runtime := &fakeRuntime{nextInstanceID: 101, templateState: &tpl}
	m := newTestModule(*domain.NewRegistry(), runtime)
	m.world = newFakeWorld()
	m.recruitersForTest = testRecruiters()
	saves := &userSaves{}
	m.saveUser = func(*users.UserRecord) error {
		saves.calls++
		return saves.err
	}
	user := users.NewUserRecord(7, 1)
	user.Character.Gold = gold
	return m, runtime, user, saves
}

func TestParseRecruiters(t *testing.T) {
	raw := []any{
		map[any]any{"RoomId": 2003, "Name": "the hiring slate", "Candidates": []any{
			map[any]any{"Id": " Tamsin ", "MobTemplateId": 961, "Tutorial": true},
			map[any]any{"Id": "garrick", "MobTemplateId": 963, "Price": 120},
			map[any]any{"Id": "tamsin", "MobTemplateId": 62}, // repeated id
			map[any]any{"Id": "", "MobTemplateId": 62},       // blank id
			map[any]any{"Id": "nobody", "MobTemplateId": 0},  // no template
			map[any]any{"Id": "greedy", "MobTemplateId": 64, "Price": -5},
			"not a map",
		}},
		map[any]any{"RoomId": 2003, "Name": "a duplicate"}, // room listed twice
		map[any]any{"Name": "no room"},
		map[any]any{"RoomId": 2005, "Candidates": []any{map[any]any{"Id": "ysolde", "MobTemplateId": "64", "Price": "80"}}},
	}
	got := parseRecruiters(raw)
	require.Len(t, got, 2)
	assert.Equal(t, recruiter{RoomID: 2003, Name: "the hiring slate", Candidates: []candidate{
		{ID: "tamsin", MobTemplateID: 961, Tutorial: true},
		{ID: "garrick", MobTemplateID: 963, Price: 120},
	}}, got[2003])
	assert.Equal(t, "the recruiter", got[2005].Name, "a missing name gets a default")
	assert.Equal(t, []candidate{{ID: "ysolde", MobTemplateID: 64, Price: 80}}, got[2005].Candidates)
	assert.Empty(t, parseRecruiters(nil))
}

func TestRecruitListShowsCandidates(t *testing.T) {
	m, _, user, _ := newRecruitModule(t, 0)
	m.world.(*fakeWorld).templates[961] = 30
	text, err := m.recruit(user, hiringRoom, "")
	require.NoError(t, err)
	assert.Contains(t, text, "The hiring slate lists:")
	assert.Contains(t, text, "tamsin (company recruit tamsin): no archetype, level 1, alignment 65 (")
	assert.Contains(t, text, "Gear: item 919002")
	assert.Contains(t, text, "Price: free, once only", "a tutorial candidate is free whatever its price")
	assert.Contains(t, text, "Price: 120 gold")
	assert.Contains(t, text, "Your company: 0/4 companions.")

	require.NoError(t, m.registry.Claim(7, 961))
	text, _ = m.recruit(user, hiringRoom, "list")
	assert.Contains(t, text, "Price: already claimed (once only)")
}

func TestRecruitOutsideRecruiterRoom(t *testing.T) {
	m, runtime, user, _ := newRecruitModule(t, 1000)
	text, err := m.recruit(user, 12, "garrick")
	require.NoError(t, err)
	assert.Contains(t, text, "No one here is hiring.")
	assert.Zero(t, runtime.spawnCalls)
	assert.Equal(t, 1000, user.Character.Gold)
}

func TestRecruitFreeTutorialClaimedOnce(t *testing.T) {
	m, runtime, user, saves := newRecruitModule(t, 50)
	store := m.store.(*fakeStore)
	text, err := m.recruit(user, hiringRoom, "tamsin")
	require.NoError(t, err)
	assert.Contains(t, text, "tamsin joins your company (#1).")
	assert.Equal(t, 50, user.Character.Gold, "free")
	assert.Zero(t, saves.calls, "no gold changed, so no user save")
	assert.Equal(t, 961, runtime.spawnedTemplateID)

	saved, ok := store.saved.Get(7)
	require.True(t, ok)
	assert.True(t, saved.HasClaimed(961), "the claim is in the recruit's own save")
	require.Len(t, saved.Companions, 1)
	assert.NotNil(t, saved.Companions[0].State, "template gear recorded (22b)")

	// Dismissed, the claim stays: the offer doesn't come twice.
	_, err = m.dismiss(7, "#1")
	require.NoError(t, err)
	saved, ok = store.saved.Get(7)
	require.True(t, ok)
	assert.True(t, saved.HasClaimed(961))
	text, err = m.recruit(user, hiringRoom, "tamsin")
	require.NoError(t, err)
	assert.Contains(t, text, "already taken tamsin on once")
	assert.Equal(t, 1, runtime.spawnCalls)
}

func TestRecruitPricedChargesGold(t *testing.T) {
	m, runtime, user, saves := newRecruitModule(t, 150)
	text, err := m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Contains(t, text, "You pay 120 gold. garrick joins your company (#1).")
	assert.Equal(t, 30, user.Character.Gold)
	assert.Equal(t, 1, saves.calls, "the user is saved with the company")
	assert.Equal(t, 963, runtime.spawnedTemplateID)
	record, _ := m.registry.Get(7)
	assert.Empty(t, record.Claimed, "a paid recruit claims nothing")

	// Paid candidates can be hired again (for gold each time).
	user.Character.Gold = 120
	text, err = m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Contains(t, text, "joins your company (#2)")
	assert.Zero(t, user.Character.Gold)

	// A failed user save keeps the deduction in memory.
	user.Character.Gold = 200
	saves.err = errors.New("disk full")
	_, err = m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Equal(t, 80, user.Character.Gold)
}

func TestRecruitRefusalsChangeNothing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *CompanyModule, runtime *fakeRuntime, user *users.UserRecord)
		pick  string
		want  string
	}{
		{"not enough gold", func(_ *CompanyModule, _ *fakeRuntime, user *users.UserRecord) { user.Character.Gold = 119 },
			"garrick", "garrick asks 120 gold, and you have 119."},
		{"full roster", func(m *CompanyModule, _ *fakeRuntime, _ *users.UserRecord) {
			m.registry.Put(domain.Record{LeaderUserID: 7, NextCompanionID: 5, Companions: []domain.Companion{
				{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}, {ID: 3, MobTemplateID: 58}, {ID: 4, MobTemplateID: 58}}})
		}, "tamsin", "Your company is full (4/4 companions)."},
		{"unknown candidate", nil, "nobody", `No one called "nobody" is hiring here.`},
		{"unavailable template", func(_ *CompanyModule, runtime *fakeRuntime, _ *users.UserRecord) { runtime.noTemplateState = true },
			"garrick", "garrick isn't available right now."},
		{"already claimed", func(m *CompanyModule, _ *fakeRuntime, _ *users.UserRecord) { _ = m.registry.Claim(7, 961) },
			"tamsin", "already taken tamsin on once"},
		{"alignment gate", func(m *CompanyModule, _ *fakeRuntime, _ *users.UserRecord) {
			w := m.world.(*fakeWorld)
			w.templates[963] = 90
			w.leaders[7] = -60
		}, "garrick", "won't join a company"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, runtime, user, saves := newRecruitModule(t, 150)
			if tc.setup != nil {
				tc.setup(m, runtime, user)
			}
			gold := user.Character.Gold
			before, existed := m.registry.Get(7)
			text, err := m.recruit(user, hiringRoom, tc.pick)
			require.NoError(t, err, "a refusal is a message")
			assert.Contains(t, text, tc.want)
			assert.Equal(t, gold, user.Character.Gold)
			after, exists := m.registry.Get(7)
			assert.Equal(t, existed, exists)
			assert.Equal(t, before, after)
			assert.Zero(t, runtime.spawnCalls)
			assert.Zero(t, m.store.(*fakeStore).saveCalls)
			assert.Zero(t, saves.calls)
		})
	}
}

func TestRecruitSaveFailureRollsBackClaimAndKeepsGold(t *testing.T) {
	for _, pick := range []string{"tamsin", "garrick"} {
		t.Run(pick, func(t *testing.T) {
			m, runtime, user, saves := newRecruitModule(t, 150)
			m.store.(*fakeStore).failSaveOnCall = 1
			_, err := m.recruit(user, hiringRoom, pick)
			require.Error(t, err)
			assert.Equal(t, 150, user.Character.Gold, "no gold taken")
			assert.Zero(t, saves.calls)
			assert.False(t, runtime.live[101], "the mob is destroyed")
			record, _ := m.registry.Get(7)
			assert.Empty(t, record.Companions)
			assert.False(t, record.HasClaimed(961), "the claim is rolled back with the recruit")
			// The rollback's own save succeeded, so the store agrees.
			saved, _ := m.store.(*fakeStore).saved.Get(7)
			assert.False(t, saved.HasClaimed(961))

			// And it can be retried.
			text, err := m.recruit(user, hiringRoom, pick)
			require.NoError(t, err)
			assert.Contains(t, text, "joins your company")
		})
	}
}

func TestRecruitMatchesByName(t *testing.T) {
	rec := recruiter{Candidates: []candidate{{ID: "a", MobTemplateID: 900001}, {ID: "b", MobTemplateID: 900002}}}
	_, ok := matchCandidate(rec, "zzz")
	assert.False(t, ok)
	c, ok := matchCandidate(rec, " B ")
	require.True(t, ok)
	assert.Equal(t, "b", c.ID)
}

func TestSummonRefusesRecruiterTemplate(t *testing.T) {
	m, runtime, _, _ := newRecruitModule(t, 0)
	_, err := m.summon(7, 12, "961")
	assert.ErrorIs(t, err, domain.ErrTemplateNotAllowed, "a recruiter's candidate can't be summoned for free")
	assert.Zero(t, runtime.spawnCalls)
}

func TestRecruitSpawnOrSurvivalFailureRollsBackClaim(t *testing.T) {
	t.Run("spawn fails", func(t *testing.T) {
		m, runtime, user, saves := newRecruitModule(t, 150)
		runtime.failSpawnOnCall = 1
		_, err := m.recruit(user, hiringRoom, "tamsin")
		require.Error(t, err)
		assert.Equal(t, 150, user.Character.Gold)
		assert.Zero(t, saves.calls)
		record, _ := m.registry.Get(7)
		assert.Empty(t, record.Companions)
		assert.False(t, record.HasClaimed(961))
		saved, _ := m.store.(*fakeStore).saved.Get(7)
		assert.False(t, saved.HasClaimed(961), "the rollback is persisted")
		text, err := m.recruit(user, hiringRoom, "tamsin")
		require.NoError(t, err)
		assert.Contains(t, text, "joins your company")
	})
	t.Run("survival fails", func(t *testing.T) {
		m, runtime, user, _ := newRecruitModule(t, 150)
		useFakeLifecycle(t, &fakeLifecycle{ensureErr: errors.New("survival down")})
		_, err := m.recruit(user, hiringRoom, "tamsin")
		require.Error(t, err)
		assert.Zero(t, runtime.spawnCalls)
		record, _ := m.registry.Get(7)
		assert.Empty(t, record.Companions)
		assert.False(t, record.HasClaimed(961))
		assert.Equal(t, 150, user.Character.Gold)
	})
}
