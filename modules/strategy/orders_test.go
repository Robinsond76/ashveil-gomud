package strategy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/orders"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runOrders(m *StrategyModule, u *users.UserRecord, line string) string {
	return plain(m.runOrders(u, strings.Fields(line)))
}

func TestOrdersAddListRemoveAndReorder(t *testing.T) {
	m, store, u, _ := testModule(t)
	assert.Contains(t, runOrders(m, u, ""), "None set")
	assert.Contains(t, runOrders(m, u, "dain add ally 50 then guard"), "Order 1 for Dain: When an ally is below 50% health, guard that ally.")
	assert.Contains(t, runOrders(m, u, "dain add chanting then break"), "Order 2 for Dain")
	assert.Contains(t, runOrders(m, u, "dain add chant then interrupt"), "already carries that order (number 2)", "the same order twice is refused")
	assert.Contains(t, runOrders(m, u, "dain add boss then strongest"), "Order 3 for Dain")
	assert.Contains(t, runOrders(m, u, "dain add first then hold"), "most there are", "three at most")
	require.Len(t, store.saved.Orders[u.UserId]["companion:1"], 3, "every change is saved")

	out := runOrders(m, u, "dain")
	assert.Contains(t, out, "1. When an ally is below 50% health, guard that ally.")
	assert.Contains(t, out, "2. When a foe is chanting, turn on the chanter to break its chant.")
	assert.Contains(t, runOrders(m, u, ""), "Dain:")

	assert.Contains(t, runOrders(m, u, "dain up 2"), "order 2 is now read before order 1")
	assert.Equal(t, orders.Break, m.StoredOrders(u.UserId, "companion:1")[0].Do)
	assert.Contains(t, runOrders(m, u, "dain up 1"), "already read first")

	assert.Contains(t, runOrders(m, u, "dain remove 1"), "Took off Dain's order: When a foe is chanting")
	require.Len(t, m.StoredOrders(u.UserId, "companion:1"), 2)
	assert.Contains(t, runOrders(m, u, "dain remove 5"), "Take off which order")
	assert.Contains(t, runOrders(m, u, "dain clear"), "no orders now")
	assert.Empty(t, store.saved.Orders, "an empty list is not stored")
	assert.Contains(t, runOrders(m, u, "dain clear"), "no orders")
}

func TestOrdersRefuseNonsenseAndBattle(t *testing.T) {
	m, _, u, inBattle := testModule(t)
	assert.Contains(t, runOrders(m, u, "dain add ally 52 then heal"), "not an order")
	assert.Contains(t, runOrders(m, u, "dain add chanting then heal"), "heal needs an ally or self condition")
	assert.Contains(t, runOrders(m, u, "dain add dance"), "not an order")
	assert.Contains(t, runOrders(m, u, "nobody add first then hold"), "No one in your company")
	assert.Empty(t, m.StoredOrders(u.UserId, "companion:1"))

	runOrders(m, u, "dain add first then hold")
	*inBattle = true
	assert.Contains(t, runOrders(m, u, "dain clear"), "battle")
	assert.Contains(t, runOrders(m, u, "dain add chanting then break"), "battle")
	assert.Contains(t, runOrders(m, u, "dain"), "first", "reading is allowed in a battle")
	assert.Len(t, m.StoredOrders(u.UserId, "companion:1"), 1)
}

func TestOrdersWarnWhenAMemberHasNothingToCarryThemOutWith(t *testing.T) {
	m, _, u, _ := testModule(t)
	assert.Contains(t, runOrders(m, u, "dain add ally 50 then heal"), "knows no healing spell yet")
	assert.NotContains(t, runOrders(m, u, "oswin add ally 50 then heal"), "no healing spell yet")
	assert.Contains(t, runOrders(m, u, "dain add first then hold"), "knows no attack spell yet")
	assert.NotContains(t, runOrders(m, u, "me add first then hold"), "no attack spell yet")
}

func TestOrdersPresetLoadsTheClassSet(t *testing.T) {
	m, _, u, _ := testModule(t)
	out := runOrders(m, u, "dain preset")
	assert.Contains(t, out, "starting set for warrior")
	assert.Equal(t, orders.Preset("warrior"), m.StoredOrders(u.UserId, "companion:1"))
	runOrders(m, u, "oswin preset")
	assert.Equal(t, orders.Preset("cleric"), m.StoredOrders(u.UserId, "companion:2"))
}

func TestStrategyDefaultLeavesOrdersAlone(t *testing.T) {
	m, _, u, _ := testModule(t)
	runOrders(m, u, "dain add first then hold")
	run(m, u, "dain fighter")
	run(m, u, "dain default")
	assert.Len(t, m.StoredOrders(u.UserId, "companion:1"), 1)
	assert.Contains(t, run(m, u, "dain"), "Orders, read first each round")
}

func TestOrdersAreSavedBackOnAFailedSave(t *testing.T) {
	m, store, u, _ := testModule(t)
	runOrders(m, u, "dain add first then hold")
	store.failNow = true
	assert.Contains(t, runOrders(m, u, "dain add chanting then break"), "couldn't be saved")
	assert.Len(t, m.StoredOrders(u.UserId, "companion:1"), 1, "rolled back")
	assert.Contains(t, runOrders(m, u, "dain clear"), "couldn't be saved")
	assert.Len(t, m.StoredOrders(u.UserId, "companion:1"), 1, "rolled back")
}

func TestOrdersForAMemberLeavingTheCompanyArePruned(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.setOrders(4401, "companion:9", orders.Preset("warrior")))
	require.NoError(t, m.setOrders(4401, "companion:1", orders.Preset("warrior")))
	m.prune(4401, map[string]bool{"leader": true, "companion:1": true})
	assert.Empty(t, m.StoredOrders(4401, "companion:9"))
	assert.NotEmpty(t, m.StoredOrders(4401, "companion:1"))
	assert.Empty(t, store.saved.Orders[4401]["companion:9"])
}

func TestOrdersGoWhenAUserIsPurged(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.setOrders(4402, "leader", orders.Preset("wizard")))
	m.onUserPurged(events.UserPurged{UserId: 4402})
	assert.Empty(t, m.StoredOrders(4402, "leader"))
	assert.Empty(t, store.saved.Orders)
}

func TestStoredOrdersAreACopy(t *testing.T) {
	m, _, _, _ := testModule(t)
	require.NoError(t, m.setOrders(4403, "leader", orders.Preset("warrior")))
	got := m.StoredOrders(4403, "leader")
	got[0].Pct = 90
	assert.Equal(t, 40, m.StoredOrders(4403, "leader")[0].Pct, "the registry is not exposed")
	snapshot := m.registry.Clone()
	snapshot.Orders[4403]["leader"][0].Pct = 10
	assert.Equal(t, 40, m.StoredOrders(4403, "leader")[0].Pct, "a clone is deep")
}

func TestDecodeDropsBadOrdersAndKeepsTheRest(t *testing.T) {
	data := `
orders:
  7:
    "companion:1":
      - {when: ally, pct: 50, do: guard}
      - {when: ally, pct: 51, do: heal}
      - {when: chanting, do: heal}
      - {when: chanting, do: break}
      - {when: first, do: hold}
      - {when: boss, do: strongest}
    "":
      - {when: first, do: hold}
  0:
    "leader":
      - {when: first, do: hold}
`
	reg := NewRegistry()
	require.NoError(t, decodeRegistry([]byte(data), reg))
	list := reg.Orders[7]["companion:1"]
	require.Len(t, list, 3, "invalid orders are dropped, and three is the most")
	assert.Equal(t, []orders.Order{
		{When: orders.AllyHurt, Pct: 50, Do: orders.Guard},
		{When: orders.Chanting, Do: orders.Break},
		{When: orders.FirstRound, Do: orders.Hold},
	}, list)
	assert.Len(t, reg.Orders, 1, "a blank member key and user 0 are dropped")
}

func TestOrdersSurviveARestartThroughThePluginFile(t *testing.T) {
	require.NotNil(t, module, "init registered the module")
	dir := t.TempDir()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dir)
	require.NoError(t, module.loadErr)
	t.Cleanup(func() { module.registry = NewRegistry() })

	require.NoError(t, module.setOrders(4601, "companion:3", orders.Preset("cleric")))
	plugins.Save()
	matches, _ := filepath.Glob(filepath.Join(dir, "plugin-data", "strategy*", "strategy.plugin.dat"))
	require.NotEmpty(t, matches)
	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "heal")

	restarted := newModule()
	restarted.store = pluginStore{plug: module.plug}
	restarted.load()
	assert.Equal(t, orders.Preset("cleric"), restarted.StoredOrders(4601, "companion:3"))
	// The engine reads them through internal/orders.
	assert.Equal(t, orders.Preset("cleric"), orders.For(4601, "companion:3"))
	assert.Empty(t, orders.For(4601, "leader"))
}

func TestOrdersSnapshotAndRestoreForTheTestArea(t *testing.T) {
	m, _, _, _ := testModule(t)
	require.NoError(t, m.setOrders(4701, "leader", orders.Preset("wizard")))
	c := ordersContributor{m}
	snap, err := c.Capture(4701)
	require.NoError(t, err)
	require.NotNil(t, snap)
	require.NoError(t, m.setOrders(4701, "leader", nil))
	require.NoError(t, c.Restore(4701, 0, snap))
	assert.Equal(t, orders.Preset("wizard"), m.StoredOrders(4701, "leader"))
	// A restore of "none" clears.
	require.NoError(t, c.Restore(4701, 0, nil))
	assert.Empty(t, m.StoredOrders(4701, "leader"))
}
