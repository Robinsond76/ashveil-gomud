package company

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChemistryThroughPluginsLoad drives Phase 24 through the real entry
// points: plugins.Load (shipped overlay, NewRound listener, provider
// registration), company summon and chemistry and status bonuses through
// usercommands.TryCommand, real NewRound events through
// events.ProcessEvents, the real plugin store, plugins.Save, a logout and
// reload, and combat's real AttackPlayerVsMob through the registered
// provider.
func TestChemistryThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	_, thisFile, _, _ := runtime.Caller(0)
	shipped := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	human, err := os.ReadFile(filepath.Join(shipped, "races", "1-human.yaml"))
	require.NoError(t, err)
	fixtures := map[string]string{
		"biomes/default.yaml":                  "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                        "direction-aliases: {}\n",
		"races/1-human.yaml":                   string(human),
		"rooms/chemtest/zone-config.yaml":      "name: chemtest\nroomid: 922001\n",
		"rooms/chemtest/922001.yaml":           "roomid: 922001\nzone: chemtest\ntitle: Camp\ndescription: A quiet camp.\ntags: [lit]\n",
		"mobs/chemtest/58-training_dummy.yaml": "mobid: 58\nzone: chemtest\ncharacter:\n  name: training dummy\n  raceid: 1\n  level: 1\n  alignment: 10\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	// Combat needs its attack messages: the shipped ones.
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "combat-messages"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "items"), 0755))
	generic, err := os.ReadFile(filepath.Join(shipped, "combat-messages", "generic.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "combat-messages", "generic.yaml"), generic, 0600))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	camp := rooms.LoadRoom(922001)
	require.NotNil(t, camp)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)
	assert.Equal(t, domain.DefaultChemistryRules(), module.chemistryRules(), "shipped overlay matches the design defaults")
	module.chemRulesForTest = &domain.ChemistryRules{TierRounds: [3]int{3, 6, 9}, TierBonus: [3]int{2, 4, 6}}

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "bonder"
	user.Password = "$2a$test"
	user.Character.RoomId = camp.RoomId
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Validate()
	user.Character.Health = user.Character.HealthMax.Value
	users.SetTestUser(user)
	camp.AddPlayer(user.UserId)
	destroyAll := func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
	}
	t.Cleanup(func() {
		destroyAll()
		camp.RemovePlayer(7)
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
		module.chemRulesForTest = nil
	})
	messages := captureCompanyMessages(t)
	run := func(cmd, rest string) string {
		t.Helper()
		*messages = nil
		handled, err := usercommands.TryCommand(cmd, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}
	stored := func() domain.Service {
		registry := domain.NewRegistry()
		require.NoError(t, pluginStore{plug: module.plug}.Load(registry))
		s, _ := registry.Companies[7].FindService(domain.CompanionMemberKey(1))
		return s
	}
	next := util.GetRoundCount() + 100
	rounds := func(n int) string {
		t.Helper()
		*messages = nil
		turn, round := util.GetTurnCount(), util.GetRoundCount()
		for i := 0; i < n; i++ {
			next++
			events.AddToQueue(events.NewRound{RoundNumber: next})
			events.ProcessEvents()
		}
		assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
		assert.Equal(t, round, util.GetRoundCount())
		return companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}

	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy (#1).")
	assert.Contains(t, run("company", "chemistry"), "With you: 2 together, Strangers; 0% of the way to Familiar.")
	assert.Contains(t, run("status", "bonuses"), "Strangers band, 2 together: no bonus yet")

	assert.Empty(t, rounds(3))
	assert.Zero(t, domain.ChemistryBonusForUser(7), "not on disk yet")
	*messages = nil
	plugins.Save() // the autosave makes the tier durable
	events.ProcessEvents()
	told := companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, told, "Your band grows closer: Familiar (+2% to hit fighting together).")
	assert.Equal(t, 3, stored().Rounds)

	out := run("company", "chemistry")
	assert.Contains(t, out, "With you: 2 together, Familiar (+2% to hit); 0% of the way to Trusted.")
	assert.Contains(t, out, "training dummy (#1): ")
	bonuses := run("status", "bonuses")
	assert.Contains(t, bonuses, "Company Chemistry")
	assert.Contains(t, bonuses, "Familiar band, 2 together: +2% to hit")

	// The registered provider, as combat calls it, and combat itself.
	instanceID, ok := module.instance(7, 1)
	require.True(t, ok)
	assert.Equal(t, 2, domain.ChemistryBonusForUser(7))
	assert.Equal(t, 2, domain.ChemistryBonusForInstance(instanceID))
	lines := 0
	for i := 0; i < 1500 && lines == 0; i++ {
		def := &mobs.Mob{InstanceId: 999001, Character: *characters.New()}
		def.Character.RoomId = camp.RoomId
		def.Character.RaceId = 1
		def.Character.Health, def.Character.HealthMax.Value = 1000000, 1000000
		def.Character.Stats.Speed.ValueAdj = 100000 // the 25% floor without chemistry
		user.Character.SetAggro(0, def.InstanceId, characters.DefaultAttack)
		for _, msg := range combat.AttackPlayerVsMob(user, def).MessagesToSource {
			if strings.Contains(msg, "Fighting beside a companion you know well") {
				lines++
			}
		}
	}
	user.Character.SetAggro(0, 0, characters.DefaultAttack)
	user.Character.Aggro = nil
	assert.Positive(t, lines, "a real attack sees the leader's chemistry")

	// More rounds, the autosave, then a logout and a reload.
	rounds(1)
	plugins.Save()
	assert.Equal(t, 4, stored().Rounds, "the autosave writes the accrued rounds")
	events.AddToQueue(events.PlayerDespawn{UserId: 7})
	events.ProcessEvents()
	assert.Empty(t, rounds(3), "no accrual while the company is away")
	destroyAll()
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(t, module.loadErr)
	record, _ := module.registry.Get(7)
	service, ok := record.FindService(domain.CompanionMemberKey(1))
	require.True(t, ok)
	assert.Equal(t, 4, service.Rounds, "service survives restart")

	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: camp.RoomId})
	events.ProcessEvents()
	rounds(2)
	*messages = nil
	plugins.Save()
	events.ProcessEvents()
	told = companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, told, "grows closer: Trusted", "the restored companion resumes its service")
	assert.Equal(t, 6, stored().Rounds)
	assert.Equal(t, 4, domain.ChemistryBonusForUser(7))

	// A new recruit dilutes the band: (6 + 6 + 0) / 3 = 4, Familiar.
	assert.Contains(t, run("company", "summon training dummy"), "Companion summoned: training dummy (#2).")
	recruit, ok := module.instance(7, 2)
	require.True(t, ok)
	assert.Equal(t, 2, domain.ChemistryBonusForUser(7))
	assert.Equal(t, 2, domain.ChemistryBonusForInstance(recruit), "the recruit fights with the band")
	assert.Contains(t, run("company", "chemistry"), "With you: 3 together, Familiar (+2% to hit)")
}
