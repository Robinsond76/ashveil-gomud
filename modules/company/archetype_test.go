package company

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// archetypeStub is a minimal archetypes.Provider knowing warrior and rogue.
type archetypeStub struct{}

func (archetypeStub) CanTrain(int, string) (bool, string)      { return true, "" }
func (archetypeStub) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (archetypeStub) Exists(id string) bool                    { return id == "warrior" || id == "rogue" }
func (archetypeStub) ArchetypeName(id string) (string, bool) {
	switch id {
	case "warrior":
		return "Warrior", true
	case "rogue":
		return "Rogue", true
	}
	return "", false
}
func (archetypeStub) PlayerArchetype(int) (string, bool) { return "", false }

func useArchetypes(t *testing.T) {
	t.Helper()
	archetypes.SetProvider(archetypeStub{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
}

func TestSummonRecordsConfiguredCompanionArchetype(t *testing.T) {
	useArchetypes(t)
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{resolved: map[string]int{"training dummy": 58}, nextInstanceID: 101})
	_, err := module.summon(7, 12, "training dummy")
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Equal(t, "warrior", record.Companions[0].Archetype, "template 58 maps to warrior by default")
	store := module.store.(*fakeStore)
	assert.Equal(t, "warrior", store.saved.Companies[7].Companions[0].Archetype, "persisted with the summon")

	got, ok := module.CompanionArchetype(7, 1)
	assert.True(t, ok)
	assert.Equal(t, "warrior", got)
	got, ok = domain.CompanionArchetype(7, 1)
	assert.False(t, ok, "no provider registered in this test, so the seam is empty")
	assert.Empty(t, got)
}

func TestSummonSkipsUnknownConfiguredArchetype(t *testing.T) {
	// No archetype provider: the configured id can't be verified, so it
	// isn't recorded (never guessed).
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{resolved: map[string]int{"training dummy": 58}, nextInstanceID: 101})
	_, err := module.summon(7, 12, "training dummy")
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Companions[0].Archetype)
}

func TestCompanyArchetypeSetsLegacyCompanionOnce(t *testing.T) {
	useArchetypes(t)
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})

	assert.Contains(t, module.setArchetype(7, "1", "bard"), "no archetype")
	assert.Contains(t, module.setArchetype(7, "9", "rogue"), "no companion")

	text := module.setArchetype(7, "1", "rogue")
	assert.Contains(t, text, "Rogue")
	record, _ := module.registry.Get(7)
	assert.Equal(t, "rogue", record.Companions[0].Archetype)
	assert.Equal(t, "rogue", module.store.(*fakeStore).saved.Companies[7].Companions[0].Archetype)

	assert.Contains(t, module.setArchetype(7, "1", "warrior"), "already")
	record, _ = module.registry.Get(7)
	assert.Equal(t, "rogue", record.Companions[0].Archetype)
}

func TestCompanyArchetypeSaveFailureRollsBack(t *testing.T) {
	useArchetypes(t)
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	text := module.setArchetype(7, "1", "rogue")
	assert.Contains(t, text, "disk full")
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Companions[0].Archetype)
}

func TestCompanyStatusShowsArchetype(t *testing.T) {
	useArchetypes(t)
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, Archetype: "rogue"}, {ID: 2, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	status := module.status(7)
	assert.Contains(t, status, "Rogue")
	assert.Contains(t, status, "no archetype")
}

func TestCompanionArchetypesConfigParsing(t *testing.T) {
	raw := []any{
		map[any]any{"MobTemplateId": 58, "Archetype": "Warrior"},
		map[string]any{"mobtemplateid": 60, "archetype": "rogue"},
		map[any]any{"MobTemplateId": 0, "Archetype": "rogue"},
		map[any]any{"MobTemplateId": 61, "Archetype": ""},
	}
	assert.Equal(t, map[int]string{58: "warrior", 60: "rogue"}, parseCompanionArchetypes(raw))
}

// The real config path: the shipped overlay merged as Modules.company.*
// and read back flattened, as PluginConfig.Get does.
func TestCompanionArchetypesThroughRealConfigPath(t *testing.T) {
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var dataMap map[string]any
	require.NoError(t, yaml.Unmarshal(data, &dataMap))
	overlay := map[string]any{}
	for k, v := range dataMap {
		overlay["Modules.company."+k] = v
	}
	require.NoError(t, configs.AddOverlayOverrides(overlay))
	raw := configs.Flatten(configs.GetModulesConfig())["company.CompanionArchetypes"]
	assert.Equal(t, map[int]string{58: "warrior", 61: "warrior", 62: "cleric", 63: "warrior", 64: "ranger"}, parseCompanionArchetypes(raw))
}

func TestCompanyArchetypeUserCommand(t *testing.T) {
	useArchetypes(t)
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	user := users.NewUserRecord(7, 7)
	messages := captureCompanyMessages(t)
	_, err := module.userCommand("archetype 1 Rogue", user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	record, _ := module.registry.Get(7)
	assert.Equal(t, "rogue", record.Companions[0].Archetype)
	assert.NotEmpty(t, *messages)

	_, err = module.userCommand("archetype 1", user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, (*messages)[len(*messages)-1], "company archetype")
}
