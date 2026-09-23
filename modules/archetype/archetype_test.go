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
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		skills.LoadDataFiles()
		spells.LoadSpellFiles()
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
