package strategy

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// placedModule is testModule with every member placed: the player front
// left, Dain front middle, Brother Oswin back right (two columns from the
// player).
func placedModule(t *testing.T) (*StrategyModule, *fakeStore, *users.UserRecord, *bool) {
	m, store, u, battle := testModule(t)
	base := m.env.members
	m.env.members = func(u *users.UserRecord) ([]member, bool) {
		ms, ok := base(u)
		cols := map[string]int{"leader": 0, "companion:1": 1, "companion:2": 2}
		for i := range ms {
			ms[i].col, ms[i].placed = cols[ms[i].key], true
		}
		return ms, ok
	}
	return m, store, u, battle
}

func TestGuardianForms(t *testing.T) {
	m, store, u, _ := placedModule(t)

	out := run(m, u, "dain guard")
	assert.Contains(t, out, "Dain is now a guardian, guarding whoever is most hurt within reach")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian}, m.Stored(4401, "companion:1"))

	out = run(m, u, "dain guard me")
	assert.Contains(t, out, "Dain will guard you, stepping in to take blows meant for you")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Ward: "leader"}, m.Stored(4401, "companion:1"))
	require.NotNil(t, store.saved)
	assert.Equal(t, "leader", store.saved.Players[4401]["companion:1"].Ward)

	out = run(m, u, "me guardian dain")
	assert.Contains(t, out, "You will guard Dain")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Ward: "companion:1"}, m.Stored(4401, "leader"))

	// A multi-word name, and a rule change keeps the ward.
	run(m, u, "dain guard brother oswin")
	assert.Equal(t, "companion:2", m.Stored(4401, "companion:1").Ward)
	run(m, u, "dain target leader")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Rule: domain.Leader, Ward: "companion:2"}, m.Stored(4401, "companion:1"))

	// Another role, or the default, clears the ward.
	run(m, u, "dain fighter")
	assert.Equal(t, domain.Strategy{Rule: domain.Leader}, m.Stored(4401, "companion:1"))
	run(m, u, "me guard dain")
	run(m, u, "me default")
	assert.True(t, m.Stored(4401, "leader").IsZero())
}

func TestGuardianRefusals(t *testing.T) {
	m, _, u, battle := placedModule(t)
	assert.Contains(t, run(m, u, "dain guard dain"), "A guardian guards someone else")
	assert.Contains(t, run(m, u, "dain guard nobody"), `No one in your company answers to "nobody"`)
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
	// "guard" is no longer the defend rule (the owner's decision 10).
	assert.Contains(t, run(m, u, "dain target guard"), `"guard" is not a target`)
	assert.True(t, m.Stored(4401, "companion:1").IsZero())

	*battle = true
	assert.Equal(t, usercommands.BattleUnderWay, run(m, u, "dain guard me"), "set before a battle")
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
}

func TestGuardianReachWarning(t *testing.T) {
	m, _, u, _ := placedModule(t)
	// Oswin (column 3) guarding the player (column 1): out of reach.
	out := run(m, u, "oswin guard me")
	assert.Contains(t, out, "Brother Oswin will guard you")
	assert.Contains(t, out, "Out of reach: Brother Oswin can't step in for you from there")
	assert.Contains(t, run(m, u, ""), "Out of reach: Brother Oswin can't step in for you from there")
	// Dain (column 2) guarding Oswin: in reach, no warning.
	out = run(m, u, "dain guard oswin")
	assert.NotContains(t, out, "Out of reach")
	// Unplaced members fail open.
	m2, _, u2, _ := testModule(t)
	assert.NotContains(t, run(m2, u2, "oswin guard me"), "Out of reach")
}

func TestGuardianListAndDescribe(t *testing.T) {
	m, _, u, _ := placedModule(t)
	run(m, u, "dain guard me")
	run(m, u, "me guard")
	out := run(m, u, "")
	assert.Regexp(t, `Dain\s+warrior\s+guardian\s+weakest\s+guards you`, out)
	assert.Regexp(t, `You\s+wizard\s+guardian\s+weakest\s+guards the most hurt`, out)
	assert.NotContains(t, out, "knows no spell")
	out = run(m, u, "dain")
	assert.Contains(t, out, "Dain (warrior): guardian, fights, and steps in to take a blow meant for its ward.")
	assert.Contains(t, out, "Guards you (help guardian).")
	assert.NotContains(t, out, "Knows no spell")
}

func TestGuardianWardSurvivesReloadAndPrune(t *testing.T) {
	m, store, u, _ := placedModule(t)
	run(m, u, "me guard dain")
	fresh := newModule()
	fresh.store = store
	fresh.load()
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Ward: "companion:1"}, fresh.Stored(4401, "leader"))

	// A ward naming a companion no longer on the record is pruned.
	m.put(4401, "companion:1", domain.Strategy{Role: domain.Guardian, Ward: "companion:9"})
	run(m, u, "")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian}, m.Stored(4401, "companion:1"))
}

func TestDecodeRegistryCleansWards(t *testing.T) {
	raw := map[string]any{"players": map[int]map[string]domain.Strategy{
		7: {
			"leader":      {Role: "guardian", Ward: "companion:1"},
			"companion:1": {Rule: "leader", Ward: "leader"},        // not a guardian: ward dropped
			"companion:2": {Role: "guardian", Ward: "bogus"},       // not a member key: ward dropped
			"companion:3": {Role: "guardian", Ward: "companion:3"}, // itself: ward dropped
			"companion:4": {Ward: "leader"},                        // nothing left
		},
	}}
	data, err := yaml.Marshal(raw)
	require.NoError(t, err)
	r := NewRegistry()
	require.NoError(t, decodeRegistry(data, r))
	assert.Equal(t, map[int]map[string]domain.Strategy{7: {
		"leader":      {Role: domain.Guardian, Ward: "companion:1"},
		"companion:1": {Rule: domain.Leader},
		"companion:2": {Role: domain.Guardian},
		"companion:3": {Role: domain.Guardian},
	}}, r.Players)
}
