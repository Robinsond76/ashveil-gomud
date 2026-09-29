package strategy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStrategiesSurviveARestartThroughThePluginFile: the registered module,
// loaded by plugins.Load, saves a change to its own plugin file; a module
// loading that file (as after a restart or copyover) has it, and the
// engine seam reads it.
func TestStrategiesSurviveARestartThroughThePluginFile(t *testing.T) {
	require.NotNil(t, module, "init registered the module")
	dir := t.TempDir()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dir)
	require.NoError(t, module.loadErr)
	t.Cleanup(func() { module.registry = NewRegistry() })

	require.NoError(t, module.set(4501, "companion:3", domain.Strategy{Role: domain.Healer, Rule: domain.Defend}))
	plugins.Save()
	data, err := os.ReadFile(filepath.Join(dir, "plugin-data", "strategy-v1-0", "strategy.plugin.dat"))
	if os.IsNotExist(err) {
		matches, _ := filepath.Glob(filepath.Join(dir, "plugin-data", "strategy*", "strategy.plugin.dat"))
		require.NotEmpty(t, matches, "the plugin file is written")
		data, err = os.ReadFile(matches[0])
	}
	require.NoError(t, err)
	assert.Contains(t, string(data), "defend")

	restarted := newModule()
	restarted.store = pluginStore{plug: module.plug}
	restarted.load()
	assert.Equal(t, domain.Strategy{Role: domain.Healer, Rule: domain.Defend}, restarted.Stored(4501, "companion:3"))

	// The engine reads it through internal/strategy, resolved with defaults.
	assert.Equal(t, domain.Strategy{Role: domain.Healer, Rule: domain.Defend}, domain.For(4501, "companion:3", "warrior"))
	assert.Equal(t, domain.Strategy{Role: domain.Caster, Rule: domain.Weakest}, domain.For(4501, "leader", "wizard"), "unset: the default")
	assert.Equal(t, domain.DefaultAutoSpells(), domain.AutoSpells(), "the shipped automatic spells")
}

// TestStrategyReachesTheCompanyView (Phase 32g wiring): a strategy set by
// the real command is what the company view, and so the web client's
// Company payload, reports: the same resolution a battle aims by.
func TestStrategyReachesTheCompanyView(t *testing.T) {
	m, _, u, _ := testModule(t)
	domain.SetProvider(m)
	t.Cleanup(func() { domain.SetProvider(module) })

	run(m, u, "me healer")
	run(m, u, "me target strongest")
	assert.Equal(t, domain.Strategy{Role: domain.Healer, Rule: domain.Strongest}, companyview.For(u).Leader.Strategy)

	run(m, u, "dain target leader")
	assert.Equal(t, domain.Leader, enemyparty.MemberStrategy(u.UserId, company.CompanionMemberKey(1)).Rule,
		"the battle's resolution, which the view reads")
}
