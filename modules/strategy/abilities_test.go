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

// withAbilities gives each test member the abilities its archetype has
// (Dain, a warrior, tackles).
func withAbilities(m *StrategyModule) {
	members := m.env.members
	m.env.members = func(u *users.UserRecord) ([]member, bool) {
		list, ok := members(u)
		for i := range list {
			if !list[i].isPlayer {
				list[i].abilities = domain.CompanionAbilities(list[i].archetype)
			}
		}
		return list, ok
	}
}

// Phase 33e: strategy [who] abilities on|off and reserve [percent].
func TestAbilitiesAndReserveControls(t *testing.T) {
	m, store, u, battle := testModule(t)
	withAbilities(m)

	assert.Regexp(t, `Dain\s+warrior\s+fighter\s+weakest\s*; Tackle`, run(m, u, ""))
	out := run(m, u, "dain")
	assert.Contains(t, out, "Tackle, when its foe is on its feet and within hand-to-hand reach, and it has no bow or sling: knocks the foe down")
	assert.Contains(t, out, "(at most once every 4 combat rounds)")

	out = run(m, u, "dain abilities off")
	assert.Contains(t, out, "Dain will use no class abilities in battle")
	assert.Equal(t, domain.Strategy{NoAbilities: true}, m.Stored(4401, "companion:1"))
	assert.Regexp(t, `Dain\s+warrior\s+fighter\s+weakest\s*; abilities off`, run(m, u, ""))
	assert.Contains(t, run(m, u, "dain"), "Abilities off: uses none in battle.")

	// A later rule or role change keeps it.
	run(m, u, "dain target leader")
	run(m, u, "dain guard oswin")
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Rule: domain.Leader, Ward: "companion:2", NoAbilities: true}, m.Stored(4401, "companion:1"))

	assert.Contains(t, run(m, u, "dain abilities on"), "Dain will use Tackle in battle")
	assert.False(t, m.Stored(4401, "companion:1").NoAbilities)
	assert.Contains(t, run(m, u, "oswin abilities on"), "though has none yet")
	assert.Contains(t, run(m, u, "dain abilities maybe"), "Abilities on or off?")

	out = run(m, u, "me reserve 30%")
	assert.Contains(t, out, "You will cast attack spells only while 30% of your mana would be left")
	assert.Equal(t, 30, m.Stored(4401, "leader").Reserve)
	assert.Regexp(t, `You\s+wizard\s+caster.*; keeps 30% mana`, run(m, u, ""))
	assert.Contains(t, run(m, u, "me"), "Keeps 30% of your mana back from attack spells.")
	run(m, u, "me nearest")
	assert.Equal(t, 30, m.Stored(4401, "leader").Reserve, "a rule change keeps it")
	assert.Contains(t, run(m, u, "me reserve 95"), `"95" is not a reserve`)
	assert.Contains(t, run(m, u, "me reserve"), "Keep how much mana back?")
	assert.Equal(t, 30, m.Stored(4401, "leader").Reserve)

	// Saved, and loaded back.
	require.NotNil(t, store.saved)
	fresh := newModule()
	fresh.store = store
	fresh.load()
	assert.Equal(t, domain.Strategy{Rule: domain.Nearest, Reserve: 30}, fresh.Stored(4401, "leader"))
	assert.Equal(t, domain.Strategy{Role: domain.Guardian, Rule: domain.Leader, Ward: "companion:2"}, fresh.Stored(4401, "companion:1"))

	// Refused in a battle, like every change.
	*battle = true
	assert.Equal(t, usercommands.BattleUnderWay, run(m, u, "dain abilities off"))
	assert.Equal(t, usercommands.BattleUnderWay, run(m, u, "me reserve 0"))
	*battle = false

	// default clears both.
	run(m, u, "dain abilities off")
	run(m, u, "dain default")
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
}

// Old saves (no new fields) load as abilities on and no reserve; a stored
// reserve out of range is dropped, the rest kept.
func TestDecodeAbilityFields(t *testing.T) {
	var r Registry
	require.NoError(t, decodeRegistry([]byte("players:\n  4401:\n    leader:\n      rule: nearest\n"), &r))
	assert.Equal(t, domain.Strategy{Rule: domain.Nearest}, r.Players[4401]["leader"])

	data, err := yaml.Marshal(Registry{Players: map[int]map[string]domain.Strategy{4401: {
		"leader":      {Reserve: 40},
		"companion:1": {NoAbilities: true},
		"companion:2": {Rule: domain.Leader, Reserve: 300},
	}}})
	require.NoError(t, err)
	assert.Contains(t, string(data), "no_abilities: true")
	require.NoError(t, decodeRegistry(data, &r))
	assert.Equal(t, domain.Strategy{Reserve: 40}, r.Players[4401]["leader"])
	assert.Equal(t, domain.Strategy{NoAbilities: true}, r.Players[4401]["companion:1"])
	assert.Equal(t, domain.Strategy{Rule: domain.Leader}, r.Players[4401]["companion:2"])
}
