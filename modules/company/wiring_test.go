package company

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var companyTagPattern = regexp.MustCompile(`<[^>]*>`)

// TestCompanyAlignmentThroughPluginsLoad drives Phase 21a through the real
// entry points: plugins.Load registers the company command and NewRound
// listener, merges the shipped config overlay, and runs OnLoad against the
// real plugin store in a disposable world; commands run through
// usercommands.TryCommand with real mob specs and live mobs; a real
// NewRound through events.ProcessEvents drifts a live companion and writes
// the real store.
func TestCompanyAlignmentThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	fixtures := map[string]string{
		"biomes/default.yaml":                   "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                         "direction-aliases: {}\n",
		"rooms/aligntest/zone-config.yaml":      "name: aligntest\nroomid: 920001\n",
		"rooms/aligntest/920001.yaml":           "roomid: 920001\nzone: aligntest\ntitle: Camp\ndescription: A quiet camp.\n",
		"mobs/aligntest/58-training_dummy.yaml": "mobid: 58\nzone: aligntest\ncharacter:\n  name: training dummy\n  level: 1\n  alignment: -20\n",
		"mobs/aligntest/59-paladin.yaml":        "mobid: 59\nzone: aligntest\ncharacter:\n  name: paladin\n  level: 1\n  alignment: 90\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "users"), 0755))
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	camp := rooms.LoadRoom(920001)
	require.NotNil(t, camp)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module, "init registered the module")
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)
	// The shipped overlay is merged (allow list [58], the design defaults).
	rules, every := module.alignmentConfig()
	assert.Equal(t, domain.DefaultAlignmentRules(), rules, "shipped overlay matches the design defaults")
	assert.Equal(t, defaultDriftEveryRounds, every)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "aligner"
	user.Password = "$2a$test" // not a plaintext password, so commands aren't gated
	user.Character.RoomId = camp.RoomId
	user.Character.Alignment = 60
	users.SetTestUser(user)
	camp.AddPlayer(user.UserId)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
		camp.RemovePlayer(7)
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})
	messages := captureCompanyMessages(t)
	run := func(rest string) string {
		t.Helper()
		*messages = nil
		handled, err := usercommands.TryCommand("company", rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return companyTagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}

	user.Character.Alignment = 100
	out := run("inspect training dummy")
	assert.Contains(t, out, "training dummy: alignment 40 (misguided).")
	assert.Contains(t, out, "Your company: 100 (holy).")
	assert.Contains(t, out, "They won't join")
	assert.Contains(t, run("inspect paladin"), "paladin isn't available to recruit.", "not on the shipped allow list")
	out = run("summon training dummy")
	assert.Contains(t, out, "training dummy (alignment 40, misguided) won't join a company of alignment 100, holy.", "gap 120")
	_, exists := module.registry.Get(7)
	assert.False(t, exists, "the refused recruit isn't recruited")

	user.Character.Alignment = 40
	assert.Contains(t, run("inspect training dummy"), "They would join.")
	assert.Contains(t, run("summon training dummy"), "Companion summoned: training dummy (#1).", "gap 60 is exactly the limit")
	record, _ := module.registry.Get(7)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, domain.Disposition{Alignment: -20, Loyalty: 70}, *record.Companions[0].Disposition, "seeded from the real template")

	dummyInstance, ok := module.instance(7, 1)
	require.True(t, ok)
	dummy := mobs.GetInstance(dummyInstance)
	require.NotNil(t, dummy)
	assert.Equal(t, int8(-20), dummy.Character.Alignment)

	out = run("alignment")
	assert.Contains(t, out, "Company alignment: 55 (neutral)", "avg(40, -20) = 10")
	assert.Contains(t, out, "You: 70 (virtuous)")
	assert.Contains(t, out, "#1 training dummy: 40 (misguided), loyalty 70, content", "gap 60 to the leader")
	out = run("status")
	assert.Contains(t, out, "Company alignment: 55 (neutral)")
	assert.Contains(t, out, "training dummy, no archetype, alignment 40 (misguided), loyalty 70 (present)")

	turn, round := util.GetTurnCount(), util.GetRoundCount()
	for i := 0; i < defaultDriftEveryRounds; i++ {
		events.AddToQueue(events.NewRound{})
		events.ProcessEvents()
	}
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
	record, _ = module.registry.Get(7)
	assert.Equal(t, domain.Disposition{Alignment: -18, Loyalty: 72}, *record.Companions[0].Disposition)
	assert.Equal(t, int8(-18), dummy.Character.Alignment, "the live mob drifted")

	stored := domain.NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(stored))
	storedRecord, _ := stored.Get(7)
	require.Len(t, storedRecord.Companions, 1)
	assert.Equal(t, -18, storedRecord.Companions[0].Disposition.Alignment, "written to the real store")
	assert.Equal(t, defaultDriftEveryRounds, stored.DriftIn, "a full interval until the next tick")
}
