package archetype

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved   *Registry
	saveErr error
	loadErr error
	saves   int
}

func (s *fakeStore) Load(r *Registry) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	if s.saved == nil {
		*r = *NewRegistry()
		return nil
	}
	*r = s.saved.Clone()
	return nil
}

func (s *fakeStore) Save(r Registry) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	cp := r.Clone()
	s.saved = &cp
	s.saves++
	return nil
}

// shippedConfig parses the module's shipped config overlay.
func shippedConfig(t *testing.T) map[string]any {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	raw := map[string]any{}
	require.NoError(t, yaml.Unmarshal(data, &raw))
	return raw
}

var dataOnce sync.Once

// loadRealData loads the shipped world's skills and spells once per binary.
func loadRealData(t *testing.T) {
	t.Helper()
	dataOnce.Do(func() {
		_, thisFile, _, _ := runtime.Caller(0)
		dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir, "Network.LogoutRounds": 3}))
		skills.LoadDataFiles()
		spells.LoadSpellFiles()
		// Buffs before items, as the server loads them: item values count
		// their buffs (the Phase 22a kit balance test relies on it).
		buffs.LoadFlagDataFiles()
		buffs.LoadDataFiles()
		items.LoadDataFiles()
		races.LoadDataFiles() // kit gear is equipped by race hand rules
		keywords.LoadAliases()
	})
}

// testModule builds a module from the shipped config against real skills
// and spells, with a fake store.
func testModule(t *testing.T) (*ArchetypeModule, *fakeStore) {
	t.Helper()
	loadRealData(t)
	raw := shippedConfig(t)
	m := newModule()
	store := &fakeStore{}
	m.store = store
	// Never write user files into the shipped data dir.
	m.saveUser = func(*users.UserRecord) error { return nil }
	m.table = m.buildTable(parseArchetypes(raw["Archetypes"]))
	cfg := parseUtilityConfig(func(k string) any { return raw[k] })
	cfg.UtilitySkills = parseUtilitySkills(raw["Utilities"], cfg.UtilitySkills)
	m.config = cfg
	return m, store
}

func newUser(id int) *users.UserRecord {
	u := users.NewUserRecord(id, uint64(id))
	u.Character.Name = "Tester"
	u.Character.Validate()
	return u
}

func TestShippedArchetypesLoad(t *testing.T) {
	m, _ := testModule(t)
	got := []string{}
	for _, a := range m.table.List() {
		got = append(got, a.ID)
	}
	assert.Equal(t, []string{"cleric", "ranger", "rogue", "warrior", "wizard"}, got, "every shipped archetype resolves against real skills and spells")

	wiz, ok := m.table.Get("wizard")
	require.True(t, ok)
	assert.Equal(t, []string{"floatinglight"}, wiz.GrantSpells)
	assert.True(t, wiz.HasUtility("light"))
	assert.Equal(t, []string{"cleric", "wizard"}, m.table.SkillClaimants("cast"))
	assert.False(t, m.table.SkillClaimed("search"), "search is a trade skill")
	assert.Equal(t, "cast", m.config.UtilitySkills["light"])
	assert.Equal(t, "skulduggery", m.config.UtilitySkills["traps"])
}

func TestEveryShippedSpellSchoolIsClaimed(t *testing.T) {
	m, _ := testModule(t)
	for _, s := range spells.GetAllSpells() {
		if s.School == "" {
			continue
		}
		assert.NotEmpty(t, m.table.SchoolClaimants(string(s.School)), "spell %q school %q", s.SpellId, s.School)
	}
}

func TestBuildTableDropsUnresolvedReferences(t *testing.T) {
	m, _ := testModule(t)
	list := []archetypes.Archetype{
		{ID: "bad-skill", Name: "Bad", Skills: []string{"nosuchskill"}, CompanionLevels: []int{1, 2, 3, 4}},
		{ID: "bad-spell", Name: "Bad", Skills: []string{"cast"}, Schools: []string{"illusion"}, GrantSpells: []string{"heal"}, CompanionLevels: []int{1, 2, 3, 4}},
		{ID: "fine", Name: "Fine", Skills: []string{"cast"}, CompanionLevels: []int{1, 2, 3, 4}},
	}
	table := m.buildTable(list)
	assert.Equal(t, 1, table.Len())
	_, ok := table.Get("fine")
	assert.True(t, ok)
}

func TestChoosePreviewConfirmAndPermanent(t *testing.T) {
	m, store := testModule(t)
	u := newUser(11)

	text := m.choose(u, "wizard", false)
	assert.Contains(t, text, "permanent")
	_, chosen := m.PlayerArchetype(11)
	assert.False(t, chosen, "preview changes nothing")
	assert.Zero(t, u.Character.GetSkillLevel("cast"))

	text = m.choose(u, "wizard", true)
	assert.Contains(t, text, "Wizard")
	id, chosen := m.PlayerArchetype(11)
	require.True(t, chosen)
	assert.Equal(t, "wizard", id)
	assert.Equal(t, "wizard", store.saved.Players[11], "persisted")
	assert.Equal(t, 1, u.Character.GetSkillLevel("cast"), "grant applied")
	assert.True(t, u.Character.HasSpell("floatinglight"))

	text = m.choose(u, "rogue", true)
	assert.Contains(t, text, "permanent")
	id, _ = m.PlayerArchetype(11)
	assert.Equal(t, "wizard", id, "a second choice is refused")

	assert.Contains(t, m.choose(u, "bard", true), "no archetype")
}

func TestChooseSaveFailureLeavesNothing(t *testing.T) {
	m, store := testModule(t)
	store.saveErr = errors.New("disk full")
	u := newUser(12)
	text := m.choose(u, "rogue", true)
	assert.Contains(t, text, "save failed")
	_, chosen := m.PlayerArchetype(12)
	assert.False(t, chosen)
	assert.Zero(t, u.Character.GetSkillLevel("skulduggery"), "no grants without a durable choice")
}

func TestGrantsNeverLowerAndReapplyOnSpawn(t *testing.T) {
	m, _ := testModule(t)
	u := newUser(13)
	u.Character.SetSkill("cast", 3)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(13) })

	m.choose(u, "wizard", true)
	assert.Equal(t, 3, u.Character.GetSkillLevel("cast"), "a higher level is kept")

	// Simulate a crash before the user record saved: the grant is missing
	// on the character but the choice is durable.
	u.Character.UnLearnSpell("floatinglight")
	m.onPlayerSpawn(events.PlayerSpawn{UserId: 13})
	assert.True(t, u.Character.HasSpell("floatinglight"), "re-applied on login")
}

func TestReloadKeepsChoices(t *testing.T) {
	m, store := testModule(t)
	m.choose(newUser(14), "cleric", true)

	reloaded := newModule()
	reloaded.store = store
	reloaded.schoolOf = m.schoolOf
	reloaded.skillExists = m.skillExists
	reloaded.load()
	require.NoError(t, reloaded.persistenceAvailable())
	id, ok := reloaded.PlayerArchetype(14)
	assert.True(t, ok)
	assert.Equal(t, "cleric", id)
}

func TestLoadFailureBlocksWrites(t *testing.T) {
	m := newModule()
	m.store = &fakeStore{loadErr: errors.New("corrupt")}
	m.load()
	assert.Error(t, m.persistenceAvailable())
	assert.Contains(t, m.choose(newUser(15), "wizard", true), "unavailable")
}

func TestAdminReset(t *testing.T) {
	m, store := testModule(t)
	u := newUser(16)
	m.choose(u, "ranger", true)
	text, err := m.reset(16)
	require.NoError(t, err)
	assert.Equal(t, "Archetype cleared.", text)
	_, ok := m.PlayerArchetype(16)
	assert.False(t, ok)
	_, persisted := store.saved.Players[16]
	assert.False(t, persisted)
	assert.Equal(t, 1, u.Character.GetSkillLevel("track"), "granted skills are kept")

	text, err = m.reset(16)
	require.NoError(t, err)
	assert.Contains(t, text, "no archetype")
}

func TestProviderDecisions(t *testing.T) {
	m, _ := testModule(t)
	ok, reason := m.CanTrain(20, "cast")
	assert.False(t, ok)
	assert.Contains(t, reason, "archetype")
	ok, _ = m.CanTrain(20, "search")
	assert.True(t, ok)

	m.choose(newUser(20), "wizard", true)
	ok, _ = m.CanTrain(20, "cast")
	assert.True(t, ok)
	ok, _ = m.CanTrain(20, "tame")
	assert.False(t, ok)

	ok, _ = m.CanLearnSpell(20, "illum")
	assert.True(t, ok, "illusion (after the illlusion typo fix)")
	ok, reason = m.CanLearnSpell(20, "heal")
	assert.False(t, ok)
	assert.Contains(t, reason, "Cleric")
	ok, _ = m.CanLearnSpell(20, "aidskill")
	assert.True(t, ok, "schoolless spells are open")
	ok, _ = m.CanLearnSpell(20, "nosuchspell")
	assert.True(t, ok, "unknown spells are left to the engine")

	assert.True(t, m.Exists("Wizard"))
	name, ok := m.ArchetypeName("cleric")
	assert.True(t, ok)
	assert.Equal(t, "Cleric", name)
}

func TestListShowsTableAndChoice(t *testing.T) {
	m, _ := testModule(t)
	text := m.list(30)
	assert.Contains(t, text, "not chosen")
	for _, name := range []string{"Warrior", "Rogue", "Wizard", "Cleric", "Ranger"} {
		assert.Contains(t, text, name)
	}
	m.choose(newUser(30), "warrior", true)
	assert.Contains(t, m.list(30), "You are a Warrior.")
}

func TestUserCommandParsesConfirm(t *testing.T) {
	m, _ := testModule(t)
	u := newUser(31)
	_, err := m.userCommand("choose wizard confirm", u, nil, 0)
	require.NoError(t, err)
	id, ok := m.PlayerArchetype(31)
	require.True(t, ok)
	assert.Equal(t, "wizard", id)
}

func TestDecodeRegistryDropsBadKeys(t *testing.T) {
	data := []byte("players:\n  0: wizard\n  5: Rogue\n  6: ''\ndisarmed:\n  '': 10\n  1-chest: 99\n")
	r := NewRegistry()
	require.NoError(t, decodeRegistry(data, r))
	keys := []int{}
	for k := range r.Players {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	assert.Equal(t, []int{5}, keys)
	assert.Equal(t, "rogue", r.Players[5])
	assert.Equal(t, map[string]uint64{"1-chest": 99}, r.Disarmed)
}

// Review 17a finding 3: a school typo on either side is reported at load.
func TestSchoolWarnings(t *testing.T) {
	table, errs := archetypes.NewTable([]archetypes.Archetype{
		{ID: "wizard", Name: "Wizard", Skills: []string{"cast"}, Schools: []string{"illusion", "ilusion"}, CompanionLevels: []int{1, 2, 3, 4}},
	})
	require.Empty(t, errs)
	warnings := schoolWarnings(table, map[string]string{
		"floatinglight": "illusion",
		"illum":         "illlusion",
		"aidskill":      "",
	})
	assert.Equal(t, []string{
		`archetype school "ilusion" matches no loaded spell`,
		`spell "illum" has school "illlusion", which no archetype claims`,
	}, warnings)
}

func TestShippedDataHasNoSchoolWarnings(t *testing.T) {
	m, _ := testModule(t)
	assert.Empty(t, schoolWarnings(m.table, nativeSpellSchools()))
}

// Review 17a finding 4: the archetype belongs to the character; a
// permanent death (which swaps in a new character) clears it.
func TestPermadeathClearsChoice(t *testing.T) {
	m, store := testModule(t)
	m.choose(newUser(110), "wizard", true)
	m.choose(newUser(111), "rogue", true)

	m.onPlayerDeath(events.PlayerDeath{UserId: 110, Permanent: false})
	_, kept := m.PlayerArchetype(110)
	assert.True(t, kept, "an ordinary death keeps the archetype")

	m.onPlayerDeath(events.PlayerDeath{UserId: 110, Permanent: true})
	_, kept = m.PlayerArchetype(110)
	assert.False(t, kept)
	_, persisted := store.saved.Players[110]
	assert.False(t, persisted)
	_, other := m.PlayerArchetype(111)
	assert.True(t, other)
}

func TestPermadeathThroughEventQueue(t *testing.T) {
	m, _ := testModule(t)
	m.choose(newUser(112), "wizard", true)
	id := events.RegisterListener(events.PlayerDeath{}, m.onPlayerDeath)
	t.Cleanup(func() { events.UnregisterListener(events.PlayerDeath{}, id) })
	events.AddToQueue(events.PlayerDeath{UserId: 112, Permanent: true})
	events.ProcessEvents()
	_, kept := m.PlayerArchetype(112)
	assert.False(t, kept)
}

// Review 17a finding 5: a stored archetype that is no longer configured
// doesn't strand the player.
func TestUnknownStoredArchetypeCanChooseAgain(t *testing.T) {
	m, store := testModule(t)
	m.registry.Players[113] = "bard"
	u := newUser(113)
	assert.Contains(t, m.choose(u, "wizard", true), "You are now a Wizard")
	assert.Equal(t, "wizard", store.saved.Players[113])
	assert.Contains(t, m.choose(u, "rogue", true), "permanent", "a known choice is still permanent")
}

// Review 17a finding 6: while the registry failed to load, claimed skills
// and schools are refused with an explicit reason; trade skills stay open.
func TestLoadFailureFailsClosedWithReason(t *testing.T) {
	m, _ := testModule(t)
	m.loadErr = errors.New("corrupt")
	ok, reason := m.CanTrain(114, "cast")
	assert.False(t, ok)
	assert.Contains(t, reason, "unavailable")
	ok, _ = m.CanTrain(114, "search")
	assert.True(t, ok)
	ok, reason = m.CanLearnSpell(114, "heal")
	assert.False(t, ok)
	assert.Contains(t, reason, "unavailable")
	ok, _ = m.CanLearnSpell(114, "aidskill")
	assert.True(t, ok)
}

// Review 17a coverage gap: the real config path. Plugins merge their
// data-overlays/config.yaml into the global config as Modules.<name>.*
// (plugins.Load) and read it back flattened (PluginConfig.Get); this test
// replays that path and loads the module from it.
func TestLoadThroughRealPluginConfigPath(t *testing.T) {
	loadRealData(t)
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var dataMap map[string]any
	require.NoError(t, yaml.Unmarshal(data, &dataMap))
	overlay := map[string]any{}
	for k, v := range dataMap {
		overlay["Modules.archetype."+k] = v
	}
	require.NoError(t, configs.AddOverlayOverrides(overlay))

	m := newModule()
	m.plug = plugins.New("archetype", "1.0")
	require.NotNil(t, m.plug, "plugin registration is open in tests")
	m.store = &fakeStore{}
	m.load()

	assert.Equal(t, 5, m.table.Len())
	wiz, ok := m.table.Get("wizard")
	require.True(t, ok)
	assert.Equal(t, map[string]int{"cast": 1}, wiz.GrantSkills)
	assert.Equal(t, []int{1, 10, 20, 30}, wiz.CompanionLevels)
	assert.Equal(t, "skulduggery", m.config.UtilitySkills["traps"])
	assert.Equal(t, 900, m.config.DisarmRounds)
}

// Review 17a coverage gap: the spawn re-grant through the event queue.
func TestSpawnRegrantThroughEventQueue(t *testing.T) {
	m, _ := testModule(t)
	u := newUser(115)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(115) })
	m.choose(u, "wizard", true)
	u.Character.UnLearnSpell("floatinglight")

	id := events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	t.Cleanup(func() { events.UnregisterListener(events.PlayerSpawn{}, id) })
	events.AddToQueue(events.PlayerSpawn{UserId: 115})
	events.ProcessEvents()
	assert.True(t, u.Character.HasSpell("floatinglight"))
}

// The module lock guards the registry against concurrent callers (the
// plugin save callback runs off the command path). Run under -race.
func TestConcurrentRegistryAccess(t *testing.T) {
	m, _, _ := utilModule(t, 50)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			u := newUser(200 + i)
			for j := 0; j < 20; j++ {
				m.choose(u, "rogue", true)
				m.setAutoskill(u.UserId, "light", j%2 == 0)
				m.CanTrain(u.UserId, "cast")
				m.CanLearnSpell(u.UserId, "heal")
				m.TrapArmed("1-chest")
				m.autoskillOn(u.UserId, "traps")
				_, _ = m.PlayerArchetype(u.UserId)
				_ = m.save()
				m.list(u.UserId)
			}
		}()
	}
	wg.Wait()
	for i := 0; i < 8; i++ {
		id, ok := m.PlayerArchetype(200 + i)
		assert.True(t, ok)
		assert.Equal(t, "rogue", id)
	}
}
