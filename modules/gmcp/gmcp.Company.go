package gmcp

// Phase 26b: the Ashveil Company namespace. The payloads are built from
// the internal/companyview summary each time it is refreshed (every round
// and after every command, on the game loop) and sent only when they
// change: "Company" is the full snapshot, "Company.Vitals" only the members'
// health and needs. They go only to the company's own leader. See
// docs/superpowers/specs/2026-09-25-phase-26b-browser-company-panel-design.md.

import (
	"encoding/json"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

type companyNeed struct {
	Value int    `json:"value"`
	Label string `json:"label"`
	Warn  bool   `json:"warn"`
}

type companyNeeds struct {
	Hunger  *companyNeed `json:"hunger"`
	Thirst  *companyNeed `json:"thirst"`
	Fatigue *companyNeed `json:"fatigue"`
}

type companyVitals struct {
	HP     *int          `json:"hp"`
	HPMax  *int          `json:"hp_max"`
	Needs  *companyNeeds `json:"needs"`
	Warmth *string       `json:"warmth"`
}

type companyCell struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type companyMember struct {
	Key           string       `json:"key"`
	ID            int          `json:"id"`
	Name          string       `json:"name"`
	Status        string       `json:"status"`
	Level         int          `json:"level"`
	Archetype     *string      `json:"archetype"`
	Cell          *companyCell `json:"cell"`
	RescueSeconds *int         `json:"rescue_seconds"`
	Chemistry     *string      `json:"chemistry"`
}

type companyLoad struct {
	Label     string `json:"label"`
	TotalG    int    `json:"total_g"`
	CapacityG int    `json:"capacity_g"`
	CargoG    int    `json:"cargo_g"`
}

type companyRest struct {
	Tier    string `json:"tier"`
	Seconds int    `json:"seconds"`
}

// companyStructure is everything but the vitals.
type companyStructure struct {
	Leader     companyMember   `json:"leader"`
	Members    []companyMember `json:"members"`
	Alive      int             `json:"alive"`
	Dead       int             `json:"dead"`
	Load       *companyLoad    `json:"load"`
	Activity   *string         `json:"activity"`
	Rest       *companyRest    `json:"rest"`
	Checkpoint *string         `json:"checkpoint"`
}

// companyPayload is the full "Company" snapshot.
type companyPayload struct {
	companyStructure
	Vitals map[string]companyVitals `json:"vitals"`
}

// companyVitalsPayload is "Company.Vitals".
type companyVitalsPayload struct {
	Vitals map[string]companyVitals `json:"vitals"`
}

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }

func needOf(n companyview.Need) *companyNeed {
	if !n.Known {
		return nil
	}
	return &companyNeed{Value: n.Value, Label: n.Label, Warn: n.Warns()}
}

func statusName(s company.MemberStatus) string {
	switch s {
	case company.MemberDead:
		return "dead"
	case company.MemberAwaiting:
		return "awaiting"
	}
	return "present"
}

func vitalsOf(m companyview.Member) companyVitals {
	v := companyVitals{}
	if m.HasHP {
		v.HP, v.HPMax = intPtr(m.HP), intPtr(m.HPMax)
	}
	if m.Hunger.Known || m.Thirst.Known || m.Fatigue.Known {
		v.Needs = &companyNeeds{Hunger: needOf(m.Hunger), Thirst: needOf(m.Thirst), Fatigue: needOf(m.Fatigue)}
	}
	if m.WarmthKnown {
		v.Warmth = strPtr(m.Warmth)
	}
	return v
}

// chemistryOf is a member's chemistry tier name, or nil when unknown.
type chemistryFunc func(leaderUserID int, key company.MemberKey) (company.ChemistryStandingView, bool)

func memberOf(m companyview.Member, leaderUserID int, chemistry chemistryFunc) companyMember {
	out := companyMember{Key: string(m.Key), ID: m.ID, Name: m.Name, Status: statusName(m.Status), Level: m.Level}
	if !m.Leader || m.ArchetypeKnown {
		out.Archetype = strPtr(m.Archetype)
	}
	if m.Placed {
		out.Cell = &companyCell{Row: m.Row, Col: m.Col}
	}
	if m.Status == company.MemberDead {
		out.RescueSeconds = intPtr(int(m.RescueLeft.Seconds()))
	} else if standing, ok := chemistry(leaderUserID, m.Key); ok {
		out.Chemistry = strPtr(company.TierName(standing.Tier))
	}
	return out
}

// buildCompanyPayload turns a summary into the snapshot. ok is false when
// the company can't be read; the empty payload is then sent instead.
func buildCompanyPayload(leaderUserID int, s companyview.Summary, chemistry chemistryFunc) (companyPayload, bool) {
	if !s.CompanyKnown {
		return companyPayload{}, false
	}
	p := companyPayload{Vitals: map[string]companyVitals{}}
	p.Leader = memberOf(s.Leader, leaderUserID, chemistry)
	p.Vitals[string(s.Leader.Key)] = vitalsOf(s.Leader)
	p.Members = []companyMember{}
	for _, m := range s.Companions {
		p.Members = append(p.Members, memberOf(m, leaderUserID, chemistry))
		p.Vitals[string(m.Key)] = vitalsOf(m)
	}
	p.Alive, p.Dead = s.Alive, s.Dead
	if s.LoadKnown {
		p.Load = &companyLoad{Label: s.LoadLabel, TotalG: s.Load.TotalGrams(), CapacityG: s.Load.CapacityGrams, CargoG: s.Load.CargoGrams}
	}
	if s.ActivityKnown {
		p.Activity = strPtr(s.Activity.Label())
	}
	if s.RestKnown {
		p.Rest = &companyRest{Tier: s.RestTier.String(), Seconds: int(s.RestLeft.Seconds())}
	}
	if s.Checkpoint != "" {
		p.Checkpoint = strPtr(s.Checkpoint)
	}
	return p, true
}

// companySent is what was last sent to one user.
type companySent struct {
	structure string
	vitals    string
}

// companyFeed decides what to send. It runs on the game loop; mu only
// keeps tests honest.
type companyFeed struct {
	mu        sync.Mutex
	last      map[int]companySent
	chemistry chemistryFunc
	send      func(userID int, module string, payload []byte)
}

func newCompanyFeed() *companyFeed {
	return &companyFeed{
		last:      map[int]companySent{},
		chemistry: company.ChemistryStanding,
		send: func(userID int, module string, payload []byte) {
			events.AddToQueue(GMCPOut{UserId: userID, Module: module, Payload: payload})
		},
	}
}

// update sends a user's Company or Company.Vitals when it changed since
// the last send.
func (f *companyFeed) update(userID int, s companyview.Summary) {
	p, ok := buildCompanyPayload(userID, s, f.chemistry)
	structure, err := json.Marshal(p.companyStructure)
	if err != nil {
		mudlog.Error("gmcp: Company payload", "error", err)
		return
	}
	vitals, _ := json.Marshal(p.Vitals)
	if !ok {
		structure, vitals = []byte("{}"), nil
	}

	f.mu.Lock()
	prev, seen := f.last[userID]
	next := companySent{structure: string(structure), vitals: string(vitals)}
	f.last[userID] = next
	f.mu.Unlock()

	switch {
	case !seen || prev.structure != next.structure:
		full := []byte("{}")
		if ok {
			full, _ = json.Marshal(p)
		}
		f.send(userID, "Company", full)
	case prev.vitals != next.vitals:
		body, _ := json.Marshal(companyVitalsPayload{Vitals: p.Vitals})
		f.send(userID, "Company.Vitals", body)
	}
}

// forget makes the next update send the full snapshot (login, copyover,
// a request) or drops the user (logout).
func (f *companyFeed) forget(userID int) {
	f.mu.Lock()
	delete(f.last, userID)
	f.mu.Unlock()
}

// GMCPCompanyRequest asks for a user's full Company snapshot now.
type GMCPCompanyRequest struct {
	UserId int
}

func (g GMCPCompanyRequest) Type() string { return `GMCPCompanyRequest` }

var companyFeeds = newCompanyFeed()

func init() {
	companyview.OnRefresh.Register(func(r companyview.Refreshed) companyview.Refreshed {
		companyFeeds.update(r.User.UserId, r.Summary)
		return r
	})
	events.RegisterListener(events.PlayerSpawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerSpawn); ok {
			companyFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			companyFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(GMCPCompanyRequest{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPCompanyRequest); ok {
			companyFeeds.forget(evt.UserId)
			companyview.RefreshUser(evt.UserId)
		}
		return events.Continue
	})
}
