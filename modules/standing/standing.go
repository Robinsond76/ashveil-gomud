// Package standing is the Phase 21b settlement standing module: it owns the
// settlement alignments and standing rules, answers standing.For for the
// market and inn modules, and provides the "standing" command. Standing is
// derived from durable alignments every time; the module stores nothing.
package standing

import (
	"embed"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//go:embed files/*
var files embed.FS

const defaultBlackMarketTag = "blackmarket"

type config struct {
	rules          domain.Rules
	blackMarketTag string
	settlements    map[string]int // zone -> alignment
	order          []string       // settlements in config order
}

// StandingModule answers settlement standing.
type StandingModule struct {
	plug *plugins.Plugin
	// companyAlignment is the company average (company.CompanyAlignment).
	companyAlignment func(leaderUserID int) (int, bool)
	zoneExists       func(zone string) bool

	// mu guards cfg, written at load. It is a leaf lock.
	mu  sync.Mutex
	cfg config
}

// module is the registered instance, for wiring tests.
var module *StandingModule

func init() {
	m := &StandingModule{
		plug:             plugins.New("standing", "1.0"),
		companyAlignment: company.CompanyAlignment,
		zoneExists:       zoneExists,
		cfg:              config{rules: domain.DefaultRules(), blackMarketTag: defaultBlackMarketTag, settlements: map[string]int{}},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.AddUserCommand("standing", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	domain.SetProvider(m)
	module = m
}

func zoneExists(zone string) bool {
	for _, name := range rooms.GetAllZoneNames() {
		if name == zone {
			return true
		}
	}
	return false
}

func (m *StandingModule) load() {
	cfg := parseConfig(m.plug.Config.Get, m.zoneExists)
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
}

func (m *StandingModule) config() config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// BlackMarketTag is the room tag marking black markets.
func (m *StandingModule) BlackMarketTag() string { return m.config().blackMarketTag }

// For implements domain.Provider. It reads the company alignment through
// modules/company, which runs on the game loop, so it is called from the
// game loop only.
func (m *StandingModule) For(leaderUserID int, zone string) (domain.Standing, bool) {
	cfg := m.config()
	settlement, ok := cfg.settlements[zone]
	if !ok {
		return domain.Standing{}, false
	}
	companyAlignment, ok := m.companyAlignment(leaderUserID)
	if !ok {
		return domain.Standing{}, false
	}
	s := domain.Assess(companyAlignment, settlement, cfg.rules)
	s.Zone = zone
	return s, true
}

// display renders an engine alignment on the 1-100 scale with its band.
func display(alignment int) string {
	return fmt.Sprintf("alignment %d, %s", company.DisplayAlignment(alignment), company.AlignmentBand(alignment))
}

func effects(s domain.Standing) string {
	switch s.Tier {
	case domain.Distrusted:
		return fmt.Sprintf("Any market here charges your company %d%% more and pays %d%% less, and any inn charges %d%% more for a room. Any black market here will deal with you.",
			s.MarkupPct, s.MarkupPct, s.InnMarkupPct)
	case domain.Shunned:
		return "Any market or inn here turns you away. Only a black market, if there is one, will deal with you."
	}
	return "Any market or inn here serves your company at normal prices."
}

func (m *StandingModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	cfg := m.config()
	if len(cfg.order) == 0 {
		user.SendText("There are no settlements.")
		return true, nil
	}
	zone := ""
	if room != nil {
		zone = room.Zone
	}
	lines := []string{}
	if _, isSettlement := cfg.settlements[zone]; isSettlement {
		s, ok := m.For(user.UserId, zone)
		if !ok {
			user.SendText("Your company's standing can't be judged right now.")
			return true, nil
		}
		lines = append(lines,
			fmt.Sprintf("%s (%s) regards your company (%s) as %s.", zone, display(s.Settlement), display(s.Company), s.Tier),
			effects(s))
	} else {
		lines = append(lines, "There's no settlement here.")
	}
	others := []string{}
	for _, z := range cfg.order {
		if z == zone {
			continue
		}
		if s, ok := m.For(user.UserId, z); ok {
			others = append(others, fmt.Sprintf("  %s: %s", z, s.Tier))
		}
	}
	if len(others) > 0 {
		lines = append(lines, "Elsewhere:")
		lines = append(lines, others...)
	}
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// parseConfig reads the module config; bad values fall back to defaults.
func parseConfig(get func(string) any, zoneExists func(string) bool) config {
	cfg := config{rules: domain.DefaultRules(), blackMarketTag: defaultBlackMarketTag, settlements: map[string]int{}}
	defaults := cfg.rules
	read := func(key string, into *int, lo, hi int) {
		raw := get(key)
		if raw == nil {
			return
		}
		if v, ok := configInt(raw); ok && v >= lo && v <= hi {
			*into = v
			return
		}
		mudlog.Warn("standing: config value out of range; using default", "key", key, "value", raw)
	}
	read("WelcomeGap", &cfg.rules.WelcomeGap, 0, 200)
	read("ToleratedGap", &cfg.rules.ToleratedGap, 0, 200)
	read("DistrustedGap", &cfg.rules.DistrustedGap, 0, 200)
	if !(cfg.rules.WelcomeGap <= cfg.rules.ToleratedGap && cfg.rules.ToleratedGap <= cfg.rules.DistrustedGap) {
		mudlog.Warn("standing: gaps must be ordered WelcomeGap <= ToleratedGap <= DistrustedGap; using defaults")
		cfg.rules.WelcomeGap, cfg.rules.ToleratedGap, cfg.rules.DistrustedGap = defaults.WelcomeGap, defaults.ToleratedGap, defaults.DistrustedGap
	}
	read("DistrustedMarkupPct", &cfg.rules.DistrustedMarkupPct, 0, 90)
	read("DistrustedInnMarkupPct", &cfg.rules.DistrustedInnMarkupPct, 0, 500)
	if tag, _ := get("BlackMarketRoomTag").(string); strings.TrimSpace(tag) != "" {
		cfg.blackMarketTag = strings.TrimSpace(tag)
	}

	list, _ := get("Settlements").([]any)
	seen := map[string]int{}
	type entry struct {
		zone      string
		alignment int
	}
	entries := []entry{}
	for _, raw := range list {
		fields := lowerKeys(raw)
		if fields == nil {
			mudlog.Warn("standing: settlement entry is not a map; skipped", "value", raw)
			continue
		}
		zone, _ := fields["zone"].(string)
		zone = strings.TrimSpace(zone)
		alignment, ok := configInt(fields["alignment"])
		switch {
		case zone == "":
			mudlog.Warn("standing: settlement without a zone; skipped")
			continue
		case !ok || alignment < -100 || alignment > 100:
			mudlog.Warn("standing: settlement alignment must be -100..100; skipped", "zone", zone)
			continue
		case !zoneExists(zone):
			mudlog.Warn("standing: settlement names an unknown zone; skipped", "zone", zone)
			continue
		}
		seen[zone]++
		entries = append(entries, entry{zone, alignment})
	}
	warned := map[string]bool{}
	for _, e := range entries {
		if seen[e.zone] > 1 {
			if !warned[e.zone] {
				warned[e.zone] = true
				mudlog.Warn("standing: settlement listed more than once; skipped", "zone", e.zone)
			}
			continue
		}
		cfg.settlements[e.zone] = e.alignment
		cfg.order = append(cfg.order, e.zone)
	}
	return cfg
}

func lowerKeys(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[strings.ToLower(k)] = v
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = v
			}
		}
		return out
	}
	return nil
}

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if v == math.Trunc(v) && math.Abs(v) <= math.MaxInt32 {
			return int(v), true
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n, true
		}
	}
	return 0, false
}
