package camping

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// buffLedger fakes the buff seams: it records grants and removals, and
// answers holdsBuff from what it has granted.
type buffLedger struct {
	calls   *[]buffCall
	rounds  map[buffCall]int
	removed []buffCall
	held    map[*characters.Character]map[int]bool
}

func installLedger(m *CampingModule, calls *[]buffCall) *buffLedger {
	l := &buffLedger{calls: calls, rounds: map[buffCall]int{}, held: map[*characters.Character]map[int]bool{}}
	m.grantBuff = func(c *characters.Character, id, rounds int) error {
		*l.calls = append(*l.calls, buffCall{c.Name, id})
		l.rounds[buffCall{c.Name, id}] = rounds
		l.hold(c, id)
		return nil
	}
	m.removeBuff = func(c *characters.Character, id int) {
		l.removed = append(l.removed, buffCall{c.Name, id})
		delete(l.held[c], id)
	}
	m.hasBuff = func(c *characters.Character, id int) bool { return l.held[c][id] }
	m.roundSeconds = func() int { return 4 }
	return l
}

func (l *buffLedger) hold(c *characters.Character, id int) {
	if l.held[c] == nil {
		l.held[c] = map[int]bool{}
	}
	l.held[c][id] = true
}

// completeCamp runs a whole camp rest through its timer.
func (e *innEnv) completeCamp(t *testing.T, user *users.UserRecord) {
	t.Helper()
	room := eligibleRoom()
	e.module.establish(user, room)
	e.module.lightFire(user, room)
	require.Contains(t, e.module.startRest(user, room), "settle in")
	*e.now = e.now.Add(camping.RestDuration)
	e.scheduler.fireLatest()
}

// completeInn runs a whole inn stay through its timer.
func (e *innEnv) completeInn(t *testing.T, user *users.UserRecord) {
	t.Helper()
	user.Character.Gold += 100
	require.Contains(t, e.module.innRest(user, innRoom()), "You pay")
	*e.now = e.now.Add(60 * time.Second)
	e.scheduler.fireLatest()
}

func heroEnv(t *testing.T) (*innEnv, *users.UserRecord) {
	e := newInnEnv(t)
	user := campUser(t, 7, 100)
	user.Character.Name = "Hero"
	return e, user
}

func TestCampTimerMarksRestedPendingThenGrantsOnce(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	e.completeCamp(t, user)
	assert.True(t, e.store.saved.RestedPending[7], "owed in the same save as the recovery")
	assert.True(t, e.store.saved.RecoveryApplied[7])
	assert.Empty(t, *e.buffs, "the timer never grants buffs")

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1033}, {"Bran", 1033}}, *e.buffs)
	assert.Equal(t, 225, e.ledger.rounds[buffCall{"Hero", 1033}], "15 minutes at 4-second rounds")
	assert.False(t, e.store.saved.RestedPending[7])
	assert.Contains(t, e.store.saved.Camps, 7, "the camp stays until broken")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "Your company is Rested")

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2, "each rest grants once")
}

func TestCampRestWaitsForOfflineLeader(t *testing.T) {
	e, user := heroEnv(t)
	e.completeCamp(t, user)
	e.module.lookupUser = func(int) *users.UserRecord { return nil }
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, *e.buffs)
	assert.True(t, e.module.restedPending[7])
	e.module.lookupUser = nil
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2)
}

func TestCampRestDoesNotDowngradeWellRested(t *testing.T) {
	e, user := heroEnv(t)
	e.ledger.hold(user.Character, 1030)
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Bran", 1033}}, *e.buffs, "the leader keeps Well Rested and gets nothing")
	assert.True(t, e.ledger.held[user.Character][1030])
	assert.Empty(t, e.ledger.removed)
	assert.Equal(t, []int{camping.FatigueRecovery}, e.surv.amounts, "the rest still restores fatigue")
}

func TestInnWellRestedRemovesRested(t *testing.T) {
	e, user := heroEnv(t)
	e.ledger.hold(e.companion, 1033)
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Bran", 1033}}, e.ledger.removed)
	assert.Equal(t, []buffCall{{"Hero", 1030}, {"Bran", 1030}}, *e.buffs)
	assert.Equal(t, 450, e.ledger.rounds[buffCall{"Bran", 1030}], "30 minutes at 4-second rounds")
	assert.False(t, e.ledger.held[e.companion][1033], "and Rested can't come back when Well Rested ends")
}

func TestBothPendingGrantsWellRested(t *testing.T) {
	e, _ := heroEnv(t)
	e.module.restedPending[7] = true
	e.module.wellRestedPending[7] = true
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1030}, {"Bran", 1030}}, *e.buffs)
	assert.False(t, e.module.restedPending[7])
	assert.False(t, e.module.wellRestedPending[7])
}

// TestWellRestedMarkedMidPassIsNotLost: an inn timer that marks Well
// Rested after the round's snapshot (while it grants Rested) must not have
// its marker or stay cleared by that Rested grant.
func TestWellRestedMarkedMidPassIsNotLost(t *testing.T) {
	e, user := heroEnv(t)
	e.completeCamp(t, user)
	user.Character.Gold = 100
	require.Contains(t, e.module.innRest(user, innRoom()), "You pay")
	*e.now = e.now.Add(60 * time.Second)
	// The inn timer fires while the round pass grants the camp's Rested.
	fired := false
	e.module.grantBuff = func(c *characters.Character, id, rounds int) error {
		if !fired {
			fired = true
			e.scheduler.fireLatest()
		}
		*e.buffs = append(*e.buffs, buffCall{c.Name, id})
		e.ledger.hold(c, id)
		return nil
	}
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	require.True(t, fired)
	assert.Equal(t, []buffCall{{"Hero", 1033}, {"Bran", 1033}}, *e.buffs)
	assert.True(t, e.store.saved.WellRestedPending[7], "still owed")
	assert.Contains(t, e.store.saved.Stays, 7)

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, []buffCall{{"Hero", 1033}, {"Bran", 1033}, {"Hero", 1030}, {"Bran", 1030}}, *e.buffs)
	assert.False(t, e.store.saved.WellRestedPending[7])
	assert.Empty(t, e.store.saved.Stays)
}

func TestTierGrantSaveFailureRetries(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.store.failNextSave = true
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, e.module.restedPending[7], "restored for a retry")
	assert.Empty(t, e.module.owed, "the owed entry isn't kept without its save")
	assert.True(t, e.store.saved.RestedPending[7])

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 4, "the retry only refreshes")
	assert.False(t, e.store.saved.RestedPending[7])
	assert.Contains(t, e.store.saved.Owed[7], 2)
}

func TestLegacyRegistryWithoutTierFieldsLoads(t *testing.T) {
	var registry Registry
	require.NoError(t, decodeRegistry([]byte("well_rested_pending:\n  7: true\n"), &registry))
	assert.Empty(t, registry.RestedPending)
	assert.Empty(t, registry.Owed)
	assert.True(t, registry.WellRestedPending[7])

	require.NoError(t, decodeRegistry([]byte(`rested_pending:
  7: true
  0: true
owed:
  7:
    2: {buff_id: 1033, tier: 1, expires_at_utc: 2026-09-24T12:15:00Z}
    0: {buff_id: 1033, tier: 1, expires_at_utc: 2026-09-24T12:15:00Z}
    3: {buff_id: 1033, tier: 7, expires_at_utc: 2026-09-24T12:15:00Z}
    4: {buff_id: 0, tier: 1, expires_at_utc: 2026-09-24T12:15:00Z}
  -1:
    2: {buff_id: 1033, tier: 1, expires_at_utc: 2026-09-24T12:15:00Z}
`), &registry))
	assert.Equal(t, map[int]bool{7: true}, registry.RestedPending)
	assert.Equal(t, map[int]map[int]camping.OwedGrant{7: {2: {BuffID: 1033, Tier: camping.TierRested, ExpiresAtUTC: time.Date(2026, 9, 24, 12, 15, 0, 0, time.UTC)}}}, registry.Owed, "invalid entries dropped")

	clone := registry.Clone()
	clone.Owed[7][2] = camping.OwedGrant{}
	assert.Equal(t, 1033, registry.Owed[7][2].BuffID, "Clone is deep")
}

// --- owed grants ---

func TestAbsentCompanionGetsOwedTierOnRestoration(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, camping.OwedGrant{BuffID: 1033, Tier: camping.TierRested, ExpiresAtUTC: e.now.Add(15 * time.Minute)}, e.store.saved.Owed[7][2])

	// Still away: nothing yet.
	*e.now = e.now.Add(5 * time.Minute)
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2)

	cara := &characters.Character{Name: "Cara"}
	e.absent = nil
	e.returned = map[int]*characters.Character{2: cara}
	e.module.onNewRound(events.NewRound{RoundNumber: 3})
	assert.Equal(t, buffCall{"Cara", 1033}, (*e.buffs)[2])
	assert.Equal(t, 150, e.ledger.rounds[buffCall{"Cara", 1033}], "the 10 minutes left")
	assert.Empty(t, e.store.saved.Owed, "granted once, then cleared")

	delete(e.ledger.held, cara) // the mob despawns and respawns
	e.module.onNewRound(events.NewRound{RoundNumber: 4})
	assert.Len(t, *e.buffs, 3, "each rest grants once")
}

func TestOwedTierExpires(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	require.Contains(t, e.module.owed[7], 2)

	*e.now = e.now.Add(15 * time.Minute)
	e.returned = map[int]*characters.Character{2: {Name: "Cara"}}
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2, "nothing once the tier would have ended")
	assert.Empty(t, e.store.saved.Owed)
}

func TestExpiredOwedDroppedWhileCompanionStillAway(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	*e.now = e.now.Add(time.Hour)
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Empty(t, e.module.owed)
	assert.Empty(t, e.store.saved.Owed)
}

func TestOwedSurvivesReload(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})

	later := e.now.Add(3 * time.Minute)
	child := newTestModule(e.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return later })
	child.load()
	require.NoError(t, child.loadErr)
	require.Contains(t, child.owed[7], 2)
	calls := []buffCall{}
	ledger := installLedger(child, &calls)
	cara := &characters.Character{Name: "Cara"}
	child.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{2: cara}, []int{2}
	}
	child.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Cara", 1033}}, calls)
	assert.Equal(t, 180, ledger.rounds[buffCall{"Cara", 1033}], "12 minutes left after the restart")
	assert.Empty(t, e.store.saved.Owed)
}

func TestCampOwedNeverDowngradesOwedWellRested(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	require.Equal(t, camping.TierWellRested, e.module.owed[7][2].Tier)

	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, camping.TierWellRested, e.store.saved.Owed[7][2].Tier, "the owed Well Rested is kept")
	assert.Equal(t, 1030, e.store.saved.Owed[7][2].BuffID)

	e.returned = map[int]*characters.Character{2: {Name: "Cara"}}
	e.module.onNewRound(events.NewRound{RoundNumber: 3})
	assert.Equal(t, buffCall{"Cara", 1030}, (*e.buffs)[len(*e.buffs)-1])
}

func TestInnOwedReplacesOwedRested(t *testing.T) {
	e, user := heroEnv(t)
	e.absent = []int{2}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	require.Equal(t, camping.TierRested, e.module.owed[7][2].Tier)
	e.module.mu.Lock()
	e.module.camps = map[int]camping.Camp{}
	e.module.mu.Unlock()
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, camping.TierWellRested, e.store.saved.Owed[7][2].Tier)
}

// --- normalization ---

func TestPlayerSpawnDropsRestedUnderWellRested(t *testing.T) {
	e, user := heroEnv(t)
	e.ledger.hold(user.Character, 1030)
	e.ledger.hold(user.Character, 1033)
	e.module.onPlayerSpawn(events.PlayerSpawn{UserId: 7})
	assert.Equal(t, []buffCall{{"Hero", 1033}}, e.ledger.removed)
	assert.True(t, e.ledger.held[user.Character][1030])

	e.ledger.removed = nil
	e.ledger.hold(user.Character, 1033)
	delete(e.ledger.held[user.Character], 1030)
	e.module.onPlayerSpawn(events.PlayerSpawn{UserId: 7})
	assert.Empty(t, e.ledger.removed, "Rested alone is kept")
}

// TestOnlyOneShippedBuffIsNamedWellRested reads every shipped buff file:
// the default world's and every module's.
func TestOnlyOneShippedBuffIsNamedWellRested(t *testing.T) {
	root := filepath.Join("..", "..")
	paths, err := filepath.Glob(filepath.Join(root, "_datafiles", "world", "default", "buffs", "*.yaml"))
	require.NoError(t, err)
	modulePaths, err := filepath.Glob(filepath.Join(root, "modules", "*", "files", "datafiles", "buffs", "*.yaml"))
	require.NoError(t, err)
	names := map[string][]int{}
	for _, path := range append(paths, modulePaths...) {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var spec struct {
			BuffId int    `yaml:"buffid"`
			Name   string `yaml:"name"`
		}
		require.NoError(t, yaml.Unmarshal(data, &spec), path)
		names[spec.Name] = append(names[spec.Name], spec.BuffId)
	}
	assert.Equal(t, []int{1030}, names["Well Rested"])
	assert.Equal(t, []int{1033}, names["Rested"])
	assert.Equal(t, []int{16}, names["Refreshed"], fmt.Sprint("upstream's nap buff keeps its id"))
}

// TestShippedInnConfigMatchesDefaults parses the real overlay.
func TestShippedInnConfigMatchesDefaults(t *testing.T) {
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	assert.Equal(t, defaultInnSettings(), parseInnSettings(func(name string) any { return cfg[name] }))
}
