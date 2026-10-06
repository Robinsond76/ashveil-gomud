package survival

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 55: eating raw meat gives a Gut-ache, which is saved with the needs,
// survives a restart, shows in the status, counts down per battle and is
// cured by the remedy.

func TestEatingRawMeatGivesAGutAcheOnce(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	res, err := m.Provision(7, "", domain.Benefit{Nutrition: 10, Ailment: domain.AilmentGutAche})
	require.NoError(t, err)
	assert.Equal(t, domain.AilmentGutAche, res.Caught)
	assert.Equal(t, 3, m.registry.MustNeedsFor(7, domain.LeaderMemberKey).GutAche)
	assert.Equal(t, m.registry, m.store.(*fakeStore).saved, "written before success is reported")

	res, err = m.Provision(7, "", domain.Benefit{Nutrition: 10, Ailment: domain.AilmentGutAche})
	require.NoError(t, err)
	assert.Empty(t, res.Caught, "a second helping starts it over without a new catch")
	assert.Equal(t, 3, m.registry.MustNeedsFor(7, domain.LeaderMemberKey).GutAche)
}

func TestAnAilmentSurvivesARestart(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	caught, err := m.CatchAilment(7, domain.LeaderMemberKey, domain.AilmentFever)
	require.NoError(t, err)
	require.True(t, caught)
	require.NoError(t, m.SpendMealBattle(7, []domain.MemberKey{domain.LeaderMemberKey}))
	require.NoError(t, m.flush())

	data, err := yaml.Marshal(m.store.(*fakeStore).saved)
	require.NoError(t, err)
	var loaded domain.Registry
	require.NoError(t, decodeRegistry(data, &loaded))
	assert.Equal(t, 4, loaded.MustNeedsFor(7, domain.LeaderMemberKey).Fever, "one battle spent before the restart")
}

func TestDecodeCapsAnAilmentAndDropsASpentOne(t *testing.T) {
	data := []byte("leaders:\n" +
		"  7:\n" +
		"    leader: {hunger: 50, thirst: 50, fatigue: 50, chill: 99, gutache: 0}\n")
	var registry domain.Registry
	require.NoError(t, decodeRegistry(data, &registry))
	n := registry.MustNeedsFor(7, domain.LeaderMemberKey)
	assert.Equal(t, 4, n.Chill)
	assert.Zero(t, n.GutAche)
}

func TestCuringAnAilmentEndsItAndSaves(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	_, err := m.CatchAilment(7, domain.LeaderMemberKey, domain.AilmentChill)
	require.NoError(t, err)
	cured, err := m.CureAilment(7, domain.LeaderMemberKey, domain.AilmentChill)
	require.NoError(t, err)
	assert.True(t, cured)
	assert.Zero(t, m.registry.MustNeedsFor(7, domain.LeaderMemberKey).Chill)
	assert.Equal(t, m.registry, m.store.(*fakeStore).saved)
	cured, err = m.CureAilment(7, domain.LeaderMemberKey, domain.AilmentChill)
	require.NoError(t, err)
	assert.False(t, cured)
}

func TestSurvivalStatusShowsAnAilment(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 100, Thirst: 100, Fatigue: 100, Chill: 2}))
	assert.Contains(t, m.status(7), "In battle: Chill: -10% damage (2 battles)")
}
