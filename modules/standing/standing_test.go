package standing

import (
	"os"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

func configGetter(values map[string]any) func(string) any {
	return func(key string) any { return values[key] }
}

func knownZones(zones ...string) func(string) bool {
	return func(zone string) bool {
		for _, z := range zones {
			if z == zone {
				return true
			}
		}
		return false
	}
}

func TestParseConfigDefaultsAndBounds(t *testing.T) {
	cfg := parseConfig(configGetter(nil), knownZones())
	assert.Equal(t, domain.DefaultRules(), cfg.rules)
	assert.Equal(t, "blackmarket", cfg.blackMarketTag)
	assert.Empty(t, cfg.settlements)

	cfg = parseConfig(configGetter(map[string]any{
		"WelcomeGap": 10, "ToleratedGap": "20", "DistrustedGap": 30,
		"DistrustedMarkupPct": 90, "DistrustedInnMarkupPct": 500, "BlackMarketRoomTag": " fence ",
	}), knownZones())
	assert.Equal(t, domain.Rules{WelcomeGap: 10, ToleratedGap: 20, DistrustedGap: 30, DistrustedMarkupPct: 90, DistrustedInnMarkupPct: 500}, cfg.rules)
	assert.Equal(t, "fence", cfg.blackMarketTag)

	cfg = parseConfig(configGetter(map[string]any{
		"WelcomeGap": 50, "ToleratedGap": 20, "DistrustedGap": 30,
		"DistrustedMarkupPct": 91, "DistrustedInnMarkupPct": -1,
	}), knownZones())
	assert.Equal(t, domain.DefaultRules(), cfg.rules, "out-of-order gaps and out-of-range markups fall back")
	cfg = parseConfig(configGetter(map[string]any{"DistrustedGap": 201}), knownZones())
	assert.Equal(t, domain.DefaultRules(), cfg.rules)
}

func TestParseSettlements(t *testing.T) {
	cfg := parseConfig(configGetter(map[string]any{"Settlements": []any{
		map[any]any{"Zone": "Dunmar", "Alignment": 40},
		map[string]any{"zone": "Frostfang", "alignment": "30"},
		map[any]any{"Zone": "Nowhere", "Alignment": 0},
		map[any]any{"Zone": "Twice", "Alignment": 0},
		map[any]any{"Zone": "Twice", "Alignment": 10},
		map[any]any{"Zone": "Far", "Alignment": 101},
		map[any]any{"Zone": "NoAlign"},
		"junk",
	}}), knownZones("Dunmar", "Frostfang", "Twice", "Far", "NoAlign"))
	assert.Equal(t, map[string]int{"Dunmar": 40, "Frostfang": 30}, cfg.settlements)
	assert.Equal(t, []string{"Dunmar", "Frostfang"}, cfg.order)
}

func newTestModule(settlements map[string]int, companies map[int]int) *StandingModule {
	order := []string{}
	for _, z := range []string{"Dunmar", "Frostfang", "Old Kings Road"} {
		if _, ok := settlements[z]; ok {
			order = append(order, z)
		}
	}
	m := &StandingModule{
		companyAlignment: func(leader int) (int, bool) {
			a, ok := companies[leader]
			return a, ok
		},
	}
	m.cfg = config{rules: domain.DefaultRules(), blackMarketTag: "blackmarket", settlements: settlements, order: order}
	return m
}

func TestProviderForZone(t *testing.T) {
	m := newTestModule(map[string]int{"Dunmar": 40}, map[int]int{7: -60})
	s, ok := m.For(7, "Dunmar")
	require.True(t, ok)
	assert.Equal(t, domain.Distrusted, s.Tier)
	assert.Equal(t, "Dunmar", s.Zone)
	assert.Equal(t, 20, s.MarkupPct)
	_, ok = m.For(7, "Elsewhere")
	assert.False(t, ok, "not a settlement")
	_, ok = m.For(8, "Dunmar")
	assert.False(t, ok, "company alignment unknown")
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	events.ProcessEvents()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func runStanding(t *testing.T, m *StandingModule, zone string) string {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	users.SetTestUser(user)
	messages := captureMessages(t)
	_, err := m.userCommand("", user, &rooms.Room{Zone: zone}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return strings.Join(*messages, "\n")
}

func TestStandingCommandInSettlement(t *testing.T) {
	m := newTestModule(map[string]int{"Dunmar": 40, "Frostfang": 30, "Old Kings Road": 0}, map[int]int{7: -60})
	out := runStanding(t, m, "Dunmar")
	assert.Contains(t, out, "Dunmar (alignment 70, virtuous) regards your company (alignment 20, evil) as distrusted.")
	assert.Contains(t, out, "20% more")
	assert.Contains(t, out, "50% more")
	assert.Contains(t, out, "black market")
	assert.Contains(t, out, "Frostfang: distrusted")
	assert.Contains(t, out, "Old Kings Road: tolerated")
	assert.NotContains(t, out, "Dunmar: ", "the current settlement isn't repeated")

	m = newTestModule(map[string]int{"Dunmar": 40}, map[int]int{7: -100})
	assert.Contains(t, runStanding(t, m, "Dunmar"), "turn you away")
	m = newTestModule(map[string]int{"Dunmar": 40}, map[int]int{7: 40})
	assert.Contains(t, runStanding(t, m, "Dunmar"), "as welcome")
}

func TestStandingCommandOutsideSettlement(t *testing.T) {
	m := newTestModule(map[string]int{"Dunmar": 40}, map[int]int{7: 0})
	out := runStanding(t, m, "Wilds")
	assert.Contains(t, out, "There's no settlement here.")
	assert.Contains(t, out, "Dunmar: welcome")
	m = newTestModule(map[string]int{"Dunmar": 40}, map[int]int{})
	assert.Contains(t, runStanding(t, m, "Dunmar"), "can't be judged")
	m = newTestModule(map[string]int{}, map[int]int{7: 0})
	assert.Contains(t, runStanding(t, m, "Wilds"), "There are no settlements.")
}
