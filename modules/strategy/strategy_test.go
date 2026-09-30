package strategy

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved   *Registry
	failNow bool
	loadErr error
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
	if s.failNow {
		return errors.New("disk full")
	}
	c := r.Clone()
	s.saved = &c
	return nil
}

var tags = regexp.MustCompile(`<[^>]*>`)

func plain(s string) string { return tags.ReplaceAllString(s, "") }

// testEnv: the player (a wizard knowing Magic Missile), Dain (a warrior),
// and Brother Oswin (a cleric knowing Minor Heal).
func testModule(t *testing.T) (*StrategyModule, *fakeStore, *users.UserRecord, *bool) {
	t.Helper()
	m := newModule()
	store := &fakeStore{}
	m.store = store
	battle := false
	knows := func(ids ...string) func(string) bool {
		return func(id string) bool {
			for _, k := range ids {
				if k == id {
					return true
				}
			}
			return false
		}
	}
	m.env = env{
		members: func(u *users.UserRecord) ([]member, bool) {
			return []member{
				{key: "leader", name: "You", archetype: "wizard", isPlayer: true, mana: 12, manaMax: 20, present: true, knows: knows("mm", "floatinglight")},
				{key: "companion:1", name: "Dain", archetype: "warrior", knows: knows()},
				{key: "companion:2", name: "Brother Oswin", archetype: "cleric", present: true, mana: 5, manaMax: 9, knows: knows("heal")},
			}, true
		},
		inBattle: func(*users.UserRecord) bool { return battle },
		spellInfo: func(id string) (string, int, bool) {
			switch id {
			case "mm":
				return "Magic Missile", 6, true
			case "heal":
				return "Minor Heal", 3, true
			}
			return "", 0, false
		},
		archName: func(id string) string { return id },
	}
	u := users.NewUserRecord(4401, 1)
	return m, store, u, &battle
}

func run(m *StrategyModule, u *users.UserRecord, line string) string {
	return plain(m.run(u, strings.Fields(line)))
}

func TestListShowsDefaults(t *testing.T) {
	m, _, u, _ := testModule(t)
	out := run(m, u, "")
	assert.Contains(t, out, "How your company fights")
	assert.Regexp(t, `You\s+wizard\s+caster\s+weakest\s+Magic Missile \(6 mana\)`, out)
	assert.Regexp(t, `Dain\s+warrior\s+fighter\s+weakest`, out)
	assert.Regexp(t, `Brother Oswin\s+cleric\s+healer\s+weakest\s+Minor Heal \(3 mana\)`, out)
}

func TestSetRoleRuleAndDefault(t *testing.T) {
	m, store, u, _ := testModule(t)

	out := run(m, u, "dain target leader")
	assert.Contains(t, out, "Dain will go for their leader, the toughest of them, else the nearest foe in reach.")
	assert.Equal(t, domain.Strategy{Rule: domain.Leader}, m.Stored(4401, "companion:1"))
	require.NotNil(t, store.saved)
	assert.Equal(t, domain.Strategy{Rule: domain.Leader}, store.saved.Players[4401]["companion:1"])

	// The rule alone, and an alias.
	run(m, u, "oswin front")
	assert.Equal(t, domain.Nearest, m.Stored(4401, "companion:2").Rule)

	// A role it can't do yet warns, and is set.
	out = run(m, u, "dain caster")
	assert.Contains(t, out, "Dain knows no attack spell yet")
	assert.Contains(t, out, "Dain is now a caster")
	assert.Equal(t, domain.Strategy{Role: domain.Caster, Rule: domain.Leader}, m.Stored(4401, "companion:1"))

	// Setting the default value stores nothing for it.
	run(m, u, "me caster")
	assert.True(t, m.Stored(4401, "leader").IsZero(), "a wizard's caster role is its default")

	out = run(m, u, "dain default")
	assert.Contains(t, out, "Dain is back to the default: fighter, going for the foe with the least health left.")
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
	assert.Regexp(t, `Dain\s+warrior\s+fighter\s+weakest`, run(m, u, ""))
}

func TestRefusals(t *testing.T) {
	m, _, u, battle := testModule(t)
	assert.Contains(t, run(m, u, "me assist"), "Only a companion can assist you")
	assert.True(t, m.Stored(4401, "leader").IsZero())
	assert.Contains(t, run(m, u, "nobody leader"), `No one in your company answers to "nobody"`)
	assert.Contains(t, run(m, u, "dain dance"), `"dance" is neither a role nor a target`)
	assert.Contains(t, run(m, u, "dain target"), "Target which way?")
	// 32d review: "leader" is a rule, never the player.
	assert.Contains(t, run(m, u, "leader"), `No one in your company answers to "leader"`)

	*battle = true
	assert.Equal(t, usercommands.BattleUnderWay, run(m, u, "dain leader"), "no changes in a battle")
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
	assert.Contains(t, run(m, u, ""), "How your company fights", "reading is fine in a battle")
	assert.Contains(t, run(m, u, "oswin"), "Brother Oswin (cleric): healer")
}

func TestDescribe(t *testing.T) {
	m, _, u, _ := testModule(t)
	out := run(m, u, "oswin")
	assert.Contains(t, out, "Brother Oswin (cleric): healer, heals anyone below the healing threshold, else fights.")
	assert.Contains(t, out, "Goes for the foe with the least health left; out of reach, the nearest foe in reach.")
	assert.Contains(t, out, "Casts Minor Heal (3 mana).")
	assert.Contains(t, out, "Mana 5 of 9.")
	assert.Contains(t, out, "Unchanged from the archetype's default.")
	out = run(m, u, "#1")
	assert.Contains(t, out, "Dain (warrior): fighter")
	assert.NotContains(t, out, "Mana", "Dain isn't here")
}

func TestSaveFailureRollsBack(t *testing.T) {
	m, store, u, _ := testModule(t)
	store.failNow = true
	out := run(m, u, "dain leader")
	assert.Contains(t, out, "couldn't be saved")
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
}

func TestLoadFailureBlocksChanges(t *testing.T) {
	m, store, u, _ := testModule(t)
	store.loadErr = errors.New("corrupt")
	m.load()
	assert.Contains(t, run(m, u, "dain leader"), "can't be changed until they load again")
}

func TestReloadKeepsStrategies(t *testing.T) {
	m, store, u, _ := testModule(t)
	run(m, u, "dain leader")
	run(m, u, "oswin caster")

	fresh := newModule()
	fresh.store = store
	fresh.load()
	assert.Equal(t, domain.Strategy{Rule: domain.Leader}, fresh.Stored(4401, "companion:1"))
	assert.Equal(t, domain.Strategy{Role: domain.Caster}, fresh.Stored(4401, "companion:2"))
}

func TestPruneDropsCompanionsNoLongerThere(t *testing.T) {
	m, store, u, _ := testModule(t)
	run(m, u, "dain leader")
	m.put(4401, "companion:9", domain.Strategy{Rule: domain.Strongest}) // dismissed since
	run(m, u, "")
	assert.True(t, m.Stored(4401, "companion:9").IsZero())
	assert.Equal(t, domain.Leader, store.saved.Players[4401]["companion:1"].Rule)
	_, kept := store.saved.Players[4401]["companion:9"]
	assert.False(t, kept)
}

func TestDecodeRegistryDropsBadEntries(t *testing.T) {
	raw := map[string]any{"players": map[int]map[string]domain.Strategy{
		7:  {"leader": {Role: "healer"}, "companion:1": {Rule: "sideways"}, "": {Rule: "leader"}, "companion:2": {}},
		-1: {"leader": {Role: "caster"}},
	}}
	data, err := yaml.Marshal(raw)
	require.NoError(t, err)
	r := NewRegistry()
	require.NoError(t, decodeRegistry(data, r))
	assert.Equal(t, map[int]map[string]domain.Strategy{7: {"leader": {Role: domain.Healer}}}, r.Players)
}

func TestParseAutoSpells(t *testing.T) {
	raw := []any{
		map[string]any{"Spell": "heal", "Use": "heal"},
		map[string]any{"Spell": "nosuch", "Use": "attack"},
		map[string]any{"Spell": "mm", "Use": "zap"},
		map[string]any{"Spell": "MM", "Use": "attack"},
	}
	got := parseAutoSpells(raw, func(id string) bool { return id != "nosuch" })
	assert.Equal(t, []domain.Spell{{ID: "heal", Use: domain.UseHeal}, {ID: "mm", Use: domain.UseAttack}}, got)
}

func TestShippedConfigAutoSpells(t *testing.T) {
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	got := parseAutoSpells(cfg["AutoSpells"], nil)
	assert.Equal(t, domain.DefaultAutoSpells(), got)
}

func TestUserPurgedForgets(t *testing.T) {
	m, store, u, _ := testModule(t)
	run(m, u, "dain leader")
	id := events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	t.Cleanup(func() { events.UnregisterListener(events.UserPurged{}, id) })
	events.AddToQueue(events.UserPurged{UserId: 4401})
	events.ProcessEvents()
	assert.True(t, m.Stored(4401, "companion:1").IsZero())
	_, kept := store.saved.Players[4401]
	assert.False(t, kept)
}
