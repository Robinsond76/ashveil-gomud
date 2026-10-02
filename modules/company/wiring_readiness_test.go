package company

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingStore makes the real store's saves fail while err is set.
type failingStore struct {
	Store
	err error
}

func (s *failingStore) Save(r domain.Registry) error {
	if s.err != nil {
		return s.err
	}
	return s.Store.Save(r)
}

// readinessWorld is the 22b gear world (plugins.Load, the real plugin
// store, a real mob spec) with one summoned companion.
type readinessWorld struct {
	t     *testing.T
	user  *users.UserRecord
	camp  *rooms.Room
	store *failingStore
}

func newReadinessWorld(t *testing.T) *readinessWorld {
	t.Helper()
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	_, thisFile, _, _ := runtime.Caller(0)
	shipped := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	human, err := os.ReadFile(filepath.Join(shipped, "races", "1-human.yaml"))
	require.NoError(t, err)
	fixtures := map[string]string{
		"biomes/default.yaml":                   "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                         "direction-aliases: {}\n",
		"races/1-human.yaml":                    string(human),
		"rooms/readytest/zone-config.yaml":      "name: readytest\nroomid: 921101\n",
		"rooms/readytest/921101.yaml":           "roomid: 921101\nzone: readytest\ntitle: Camp\ndescription: A quiet camp.\n",
		"mobs/readytest/58-training_dummy.yaml": "mobid: 58\nzone: readytest\nitemdropchance: 0\ncharacter:\n  name: training dummy\n  raceid: 1\n  level: 4\n  alignment: 40\n  stats:\n    mysticism:\n      training: 6\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "combat-messages"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "items"), 0755))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	camp := rooms.LoadRoom(921101)
	require.NotNil(t, camp)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	encumbrance.SetProvider(nil)
	require.NoError(t, module.loadErr)
	store := &failingStore{Store: module.store}
	module.store = store

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "ready"
	user.Password = "$2a$test"
	user.Character.RoomId = camp.RoomId
	user.Character.RaceId = 1
	user.Character.Alignment = 40
	user.Character.Validate()
	users.SetTestUser(user)
	camp.AddPlayer(user.UserId)
	w := &readinessWorld{t: t, user: user, camp: camp, store: store}
	t.Cleanup(func() {
		w.destroyAll()
		camp.RemovePlayer(7)
		module.store = store.Store
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})
	messages := captureCompanyMessages(t)
	*messages = nil
	handled, err := usercommands.TryCommand("company", "summon training dummy", user.UserId, events.CmdSkipScripts)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	require.Contains(t, strings.Join(*messages, "\n"), "Companion summoned")
	return w
}

func (w *readinessWorld) destroyAll() {
	for _, instance := range mobs.GetAllMobInstanceIds() {
		nativeRuntime{}.Detach(7, instance)
	}
}

// live is the companion's live mob.
func (w *readinessWorld) live() *mobs.Mob {
	w.t.Helper()
	instanceID, ok := module.instance(7, 1)
	require.True(w.t, ok, "the companion is out")
	mob := mobs.GetInstance(instanceID)
	require.NotNil(w.t, mob)
	return mob
}

// hurt sets the live companion's health and mana.
func (w *readinessWorld) hurt(health, mana int) *mobs.Mob {
	mob := w.live()
	require.Greater(w.t, mob.Character.HealthMax.Value, health)
	require.Greater(w.t, mob.Character.ManaMax.Value, mana)
	mob.Character.Health, mob.Character.Mana = health, mana
	return mob
}

func (w *readinessWorld) logout() {
	events.AddToQueue(events.PlayerDespawn{UserId: 7})
	events.ProcessEvents()
	_, tracked := module.instance(7, 1)
	require.False(w.t, tracked)
}

// restart drops every live mob and reloads the registry from the real
// store, as a restart, crash, or copyover does.
func (w *readinessWorld) restart() {
	w.destroyAll()
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(w.t, module.loadErr)
}

func (w *readinessWorld) login() *mobs.Mob {
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: w.camp.RoomId})
	events.ProcessEvents()
	assert.Equal(w.t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(w.t, round, util.GetRoundCount())
	return w.live()
}

// setSaved replaces the companion's recorded vitals in memory.
func (w *readinessWorld) setSaved(v *domain.Vitals) {
	record, ok := module.registry.Get(7)
	require.True(w.t, ok)
	state := record.Companions[0].State.Clone()
	state.Vitals = v
	require.NoError(w.t, module.registry.SetState(7, 1, state))
}

// Phase 33h2: a hurt companion stays hurt through logout and login,
// a restart, the autosave/copyover seam, and a crash, without a refill.
func TestCompanionVitalsSurviveLogoutRestartAndCrash(t *testing.T) {
	w := newReadinessWorld(t)

	// Logout, restart, login: exactly the health and mana it left with.
	w.hurt(5, 1)
	w.logout()
	w.restart()
	mob := w.login()
	assert.Equal(t, 5, mob.Character.Health, "no refill at login")
	assert.Equal(t, 1, mob.Character.Mana)

	// The autosave seam (shutdown and copyover save the same way) records
	// it; a crash afterwards restores the saved values, never full.
	w.hurt(7, 2)
	plugins.Save()
	w.hurt(3, 0) // after the last save
	w.restart()  // a crash: no logout, no save
	mob = w.login()
	assert.Equal(t, 7, mob.Character.Health, "a crash restores the last save")
	assert.Equal(t, 2, mob.Character.Mana)
	assert.Less(t, mob.Character.Health, mob.Character.HealthMax.Value)

	// A failed save at logout keeps the snapshot in memory: logging back
	// in before the next save still finds the companion hurt.
	w.hurt(4, 1)
	w.store.err = errors.New("disk full")
	w.logout()
	w.store.err = nil
	mob = w.login()
	assert.Equal(t, 4, mob.Character.Health, "the in-memory snapshot, not a refill")
	assert.Equal(t, 1, mob.Character.Mana)
}

// An old record with no vitals spawns full once; a changed maximum
// clamps saved points, and a living companion never spawns below 1.
func TestCompanionVitalsMigrateAndClamp(t *testing.T) {
	w := newReadinessWorld(t)
	w.hurt(5, 1)
	w.logout()

	w.setSaved(nil)
	mob := w.login()
	assert.Equal(t, mob.Character.HealthLimit(), mob.Character.Health, "an old record spawns full")
	assert.Equal(t, mob.Character.ManaMax.Value, mob.Character.Mana)
	w.logout()
	record, _ := module.registry.Get(7)
	require.NotNil(t, record.Companions[0].State.Vitals, "tracked from its next snapshot")

	w.setSaved(&domain.Vitals{Health: 100000, Mana: 100000})
	mob = w.login()
	assert.Equal(t, mob.Character.HealthLimit(), mob.Character.Health, "a lowered maximum clamps")
	assert.Equal(t, mob.Character.ManaMax.Value, mob.Character.Mana)
	w.logout()

	w.setSaved(&domain.Vitals{Health: 0, Mana: -3})
	mob = w.login()
	assert.Equal(t, 1, mob.Character.Health, "never spawns dying")
	assert.Equal(t, 0, mob.Character.Mana)
	w.logout()

	// A lasting wound lowers the limit below the saved health.
	record, _ = module.registry.Get(7)
	state := record.Companions[0].State.Clone()
	state.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 4}}
	state.Vitals = &domain.Vitals{Health: 100000, Mana: 0}
	require.NoError(t, module.registry.SetState(7, 1, state))
	mob = w.login()
	assert.Equal(t, mob.Character.HealthMax.Value-4, mob.Character.Health, "held to the wound limit")
}

// A live level-up keeps the companion's health and mana instead of
// GoMud's refill; the next respawn keeps them too.
func TestCompanionLevelUpKeepsVitals(t *testing.T) {
	w := newReadinessWorld(t)
	mob := w.hurt(6, 1)
	level, max := mob.Character.Level, mob.Character.HealthMax.Value
	paid, _ := mobcommands.AwardCompanyXP(7, w.user.Character, 200000, w.camp.RoomId)
	require.Equal(t, 1, paid)
	require.Greater(t, mob.Character.Level, level, "levelled live")
	assert.Greater(t, mob.Character.HealthMax.Value, max, "its maximum rose")
	assert.Equal(t, 6, mob.Character.Health, "no free refill from a level")
	assert.Equal(t, 1, mob.Character.Mana)
	w.logout()
	mob = w.login()
	assert.Equal(t, 6, mob.Character.Health, "a raised maximum grants nothing at the respawn")
}

// Online recovery through the real round listener: out of a battle, with
// the leader online, a companion regains health; logged out, it doesn't.
func TestCompanionRecoversOnlineOnly(t *testing.T) {
	w := newReadinessWorld(t)
	mob := w.hurt(5, 1)
	hooks.AutoHeal(events.NewRound{RoundNumber: 3})
	assert.Equal(t, 5+mob.Character.HealthPerRound(), mob.Character.Health, "regains health on the players' beat")
	assert.Equal(t, 1+mob.Character.ManaPerRound(), mob.Character.Mana)
	hooks.AutoHeal(events.NewRound{RoundNumber: 4})
	assert.Equal(t, 5+mob.Character.HealthPerRound(), mob.Character.Health, "only every third round")

	health, mana := mob.Character.Health, mob.Character.Mana
	w.logout()
	for round := uint64(6); round <= 60; round += 3 {
		hooks.AutoHeal(events.NewRound{RoundNumber: round})
	}
	mob = w.login()
	assert.Equal(t, health, mob.Character.Health, "no offline recovery")
	assert.Equal(t, mana, mob.Character.Mana)
}

// A resurrected companion wakes at half its health limit and mana, and a
// crash before its first snapshot keeps it at half.
func TestResurrectedCompanionWakesAtHalf(t *testing.T) {
	w := newReadinessWorld(t)
	_, err := mobcommands.Suicide("", w.live(), w.camp)
	require.NoError(t, err)
	events.ProcessEvents()
	record, _ := module.registry.Get(7)
	require.True(t, record.Companions[0].Dead())
	assert.Nil(t, record.Companions[0].State.Vitals, "death clears its vitals")

	result, err := module.ResurrectCompanion(7, "dummy", w.camp.RoomId)
	require.NoError(t, err)
	require.True(t, result.Spawned)
	mob := w.live()
	assert.Equal(t, max(1, mob.Character.HealthLimit()/2), mob.Character.Health, "half its health")
	assert.Equal(t, mob.Character.ManaMax.Value/2, mob.Character.Mana, "half its mana")
	assert.Less(t, mob.Character.Health, mob.Character.HealthMax.Value)

	w.restart() // a crash before any snapshot
	mob = w.login()
	assert.Equal(t, max(1, mob.Character.HealthLimit()/2), mob.Character.Health, "still half, never full")
}

// Review finding (33h2): a level never lowers a companion's health, even
// one standing above a fresh wound's limit.
func TestCompanionLevelUpAboveWoundLimitKeepsHealth(t *testing.T) {
	w := newReadinessWorld(t)
	health := w.live().Character.HealthMax.Value - 1
	mob := w.hurt(health, 1)
	// A wound deep enough to hold the limit at its floor (a quarter of the
	// maximum) even after the level raises the maximum.
	mob.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 100000}}
	level := mob.Character.Level
	mobcommands.AwardCompanyXP(7, w.user.Character, mob.Character.XPTL(level+1), w.camp.RoomId)
	require.Greater(t, mob.Character.Level, level)
	require.Less(t, mob.Character.HealthLimit(), health, "standing above its wound limit")
	assert.Equal(t, health, mob.Character.Health, "a level never lowers health")
}

// Review coverage (33h2): a companion that fled at low health is saved by
// BeginFlight and comes back by ReturnFlight as hurt as it ran.
func TestFledCompanionReturnsAsHurt(t *testing.T) {
	w := newReadinessWorld(t)
	w.hurt(6, 2)
	require.NoError(t, module.BeginFlight(7, 1))
	_, tracked := module.instance(7, 1)
	require.False(t, tracked, "fled")
	w.restart() // the flight's save is what a restart finds
	require.NoError(t, module.ReturnFlight(7))
	mob := w.live()
	assert.Equal(t, 6, mob.Character.Health, "no refill on the return")
	assert.Equal(t, 2, mob.Character.Mana)
	record, _ := module.registry.Get(7)
	assert.Zero(t, record.Companions[0].ReturnHP, "the return debt is settled")
}

// Review coverage (33h2): a snapshot taken on the brink (health 0, not
// yet dead) comes back at 1 through the real save seam, never full.
func TestCompanionSnapshotAtZeroReturnsAtOne(t *testing.T) {
	w := newReadinessWorld(t)
	w.live().Character.Health = 0
	plugins.Save()
	w.restart()
	mob := w.login()
	assert.Equal(t, 1, mob.Character.Health)
}
