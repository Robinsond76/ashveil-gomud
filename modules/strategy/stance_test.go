package strategy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runStance(m *StrategyModule, u *users.UserRecord, line string) string {
	return plain(m.runStance(u, strings.Fields(line)))
}

// withGear gives the test company gear: Dain a glaive, you a shield.
func withGear(m *StrategyModule) {
	base := m.env.members
	m.env.members = func(u *users.UserRecord) ([]member, bool) {
		list, ok := base(u)
		list[0].gear, list[0].gearKnown = stance.Gear{Shield: true}, true
		list[1].gear, list[1].gearKnown = stance.Gear{TwoHanded: true, Class: "glaive"}, true
		return list, ok
	}
}

func TestStanceSetListAndClear(t *testing.T) {
	m, store, u, _ := testModule(t)
	withGear(m)
	assert.Contains(t, runStance(m, u, ""), "Dain: no stance (could use: heavy)")
	assert.Contains(t, runStance(m, u, ""), "You: no stance (could use: wall)")

	out := runStance(m, u, "dain heavy")
	assert.Contains(t, out, "Dain is now in the Heavy blows stance: blows land 30% harder, but 15 points less likely to hit.")
	assert.NotContains(t, out, "does nothing", "the glaive fits")
	assert.Equal(t, stance.Heavy, m.StoredStance(u.UserId, "companion:1"))
	assert.Equal(t, stance.Heavy, store.saved.Stances[u.UserId]["companion:1"], "saved")
	assert.Contains(t, runStance(m, u, "dain"), "Heavy blows (ready)")
	assert.Contains(t, runStance(m, u, ""), "Dain: Heavy blows (ready)")
	assert.Contains(t, runStance(m, u, "dain heavy blows"), "already in that stance")

	assert.Contains(t, runStance(m, u, "me wall"), "You are now in the Shield wall stance")
	assert.Contains(t, runStance(m, u, "dain off"), "Dain is out of any stance")
	assert.Equal(t, stance.None, m.StoredStance(u.UserId, "companion:1"))
	assert.Contains(t, runStance(m, u, "dain off"), "in no stance")
	assert.Empty(t, store.saved.Stances[u.UserId]["companion:1"], "none is not stored")
}

func TestStanceWithoutItsWeaponIsKeptButSaysSo(t *testing.T) {
	m, _, u, _ := testModule(t)
	withGear(m)
	out := runStance(m, u, "dain quick")
	assert.Contains(t, out, "Dain is now in the Quick draw stance")
	assert.Contains(t, out, "It does nothing until Dain holds a bow.")
	assert.Contains(t, runStance(m, u, "me keen"), "It does nothing until you hold a dagger.")
	assert.Contains(t, runStance(m, u, "dain"), "Quick draw (idle: needs a bow)")
	assert.Equal(t, stance.Quick, m.StoredStance(u.UserId, "companion:1"), "kept for when the bow is back")
}

func TestStanceRefusesNonsenseAndBattle(t *testing.T) {
	m, _, u, inBattle := testModule(t)
	assert.Contains(t, runStance(m, u, "dain dance"), "not a stance")
	assert.Contains(t, runStance(m, u, "nobody heavy"), "No one in your company")
	assert.Equal(t, stance.None, m.StoredStance(u.UserId, "companion:1"))
	runStance(m, u, "dain heavy")
	*inBattle = true
	assert.Contains(t, runStance(m, u, "dain off"), "battle")
	assert.Contains(t, runStance(m, u, "dain keen"), "battle")
	assert.Contains(t, runStance(m, u, "dain"), "Heavy blows", "reading is allowed in a battle")
	assert.Equal(t, stance.Heavy, m.StoredStance(u.UserId, "companion:1"))
}

func TestStanceIsSavedBackOnAFailedSave(t *testing.T) {
	m, store, u, _ := testModule(t)
	runStance(m, u, "dain heavy")
	store.failNow = true
	assert.Contains(t, runStance(m, u, "dain keen"), "couldn't be saved")
	assert.Equal(t, stance.Heavy, m.StoredStance(u.UserId, "companion:1"), "rolled back")
	assert.Contains(t, runStance(m, u, "dain off"), "couldn't be saved")
	assert.Equal(t, stance.Heavy, m.StoredStance(u.UserId, "companion:1"), "rolled back")
}

func TestStanceForAMemberLeavingTheCompanyIsPruned(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.setStance(4401, "companion:9", stance.Heavy))
	require.NoError(t, m.setStance(4401, "companion:1", stance.Keen))
	m.prune(4401, map[string]bool{"leader": true, "companion:1": true})
	assert.Equal(t, stance.None, m.StoredStance(4401, "companion:9"))
	assert.Equal(t, stance.Keen, m.StoredStance(4401, "companion:1"))
	assert.Empty(t, store.saved.Stances[4401]["companion:9"])
}

func TestStanceGoesWhenAUserIsPurged(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.setStance(4402, "leader", stance.Wall))
	m.onUserPurged(events.UserPurged{UserId: 4402})
	assert.Equal(t, stance.None, m.StoredStance(4402, "leader"))
	assert.Empty(t, store.saved.Stances)
}

func TestDecodeDropsBadStancesAndKeepsTheRest(t *testing.T) {
	data := `
stances:
  7:
    "companion:1": heavy
    "companion:2": nonsense
    "companion:3": ""
    "": keen
  0:
    "leader": wall
`
	reg := NewRegistry()
	require.NoError(t, decodeRegistry([]byte(data), reg))
	assert.Equal(t, map[int]map[string]stance.Stance{7: {"companion:1": stance.Heavy}}, reg.Stances)
}

func TestStanceSurvivesARestartThroughThePluginFile(t *testing.T) {
	require.NotNil(t, module, "init registered the module")
	dir := t.TempDir()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dir)
	require.NoError(t, module.loadErr)
	t.Cleanup(func() { module.registry = NewRegistry() })

	require.NoError(t, module.setStance(4801, "companion:3", stance.Quick))
	plugins.Save()
	matches, _ := filepath.Glob(filepath.Join(dir, "plugin-data", "strategy*", "strategy.plugin.dat"))
	require.NotEmpty(t, matches)
	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "quick")

	restarted := newModule()
	restarted.store = pluginStore{plug: module.plug}
	restarted.load()
	assert.Equal(t, stance.Quick, restarted.StoredStance(4801, "companion:3"))
	// The engine reads it through internal/stance.
	assert.Equal(t, stance.Quick, stance.For(4801, "companion:3"))
	assert.Equal(t, stance.None, stance.For(4801, "leader"))
}

func TestStanceSnapshotAndRestoreForTheTestArea(t *testing.T) {
	m, _, _, _ := testModule(t)
	require.NoError(t, m.setStance(4901, "leader", stance.Keen))
	c := stanceContributor{m}
	snap, err := c.Capture(4901)
	require.NoError(t, err)
	require.NotNil(t, snap)
	require.NoError(t, m.setStance(4901, "leader", stance.None))
	require.NoError(t, c.Restore(4901, 0, snap))
	assert.Equal(t, stance.Keen, m.StoredStance(4901, "leader"))
	require.NoError(t, c.Restore(4901, 0, nil))
	assert.Equal(t, stance.None, m.StoredStance(4901, "leader"))
}

func TestStrategyDefaultLeavesStanceAlone(t *testing.T) {
	m, _, u, _ := testModule(t)
	runStance(m, u, "dain heavy")
	run(m, u, "dain fighter")
	run(m, u, "dain default")
	assert.Equal(t, stance.Heavy, m.StoredStance(u.UserId, "companion:1"))
}
