package gmcp

// Phase 26b: the Ashveil Company namespace. The payloads are built from
// the internal/companyview summary each time it is refreshed (every round
// and after every command, on the game loop) and sent only when they
// change: "Company" is the full snapshot, "Company.Vitals" only the members'
// health and needs. They go only to the company's own leader. See
// docs/designs/2026-09-25-phase-26b-browser-company-panel-design.md.

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	HP    *int `json:"hp"`
	HPMax *int `json:"hp_max"`
	// HPLimit is the wound limit (Phase 30b), sent only while below max.
	HPLimit *int `json:"hp_limit,omitempty"`
	// MP and MPMax are omitted for a member with no mana to show (32g).
	MP     *int          `json:"mp,omitempty"`
	MPMax  *int          `json:"mp_max,omitempty"`
	Needs  *companyNeeds `json:"needs"`
	Warmth *string       `json:"warmth"`
}

type companyCell struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// companyStrategy is a member's role and target rule (Phase 32g).
type companyStrategy struct {
	Role   string `json:"role"`
	Target string `json:"target"`
	// Ward is a guardian's ward, a member key (Phase 30c2); omitted for the
	// most hurt. WardReach is false (else omitted) when guardian and ward
	// stand more than one column apart.
	Ward      string `json:"ward,omitempty"`
	WardReach *bool  `json:"ward_reach,omitempty"`
	// Abilities are the member's automatic class abilities by name;
	// AbilitiesOff is true when its strategy turns them off; Reserve is
	// its mana reserve in percent (Phase 33e). Each omitted when empty.
	Abilities    []string `json:"abilities,omitempty"`
	AbilitiesOff bool     `json:"abilities_off,omitempty"`
	Reserve      int      `json:"reserve,omitempty"`
}

type companyMember struct {
	Key       string       `json:"key"`
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
	Level     int          `json:"level"`
	Archetype *string      `json:"archetype"`
	Cell      *companyCell `json:"cell"`
	Chemistry *string      `json:"chemistry"`
	// Strategy is nil when unknown.
	Strategy *companyStrategy `json:"strategy"`
	// Skills are a companion's optional-skill ranks by skill id and
	// TrainingPoints its points left to spend (Phase 35c); both omitted
	// for the leader, and Skills when it has none.
	Skills         map[string]int `json:"skills,omitempty"`
	TrainingPoints *int           `json:"training_points,omitempty"`
}

type companyLoad struct {
	Label     string `json:"label"`
	TotalG    int    `json:"total_g"`
	CapacityG int    `json:"capacity_g"`
	CargoG    int    `json:"cargo_g"`
	// CompanionG is the living companions' gear (Phase 28).
	CompanionG int `json:"companion_g"`
}

type companyRest struct {
	Tier    string `json:"tier"`
	Seconds int    `json:"seconds"`
}

// companyTactics is the saved company tactics (Phase 30c).
type companyTactics struct {
	Focus   string `json:"focus"`
	Healing int    `json:"healing"`
	Patch   int    `json:"patch"`
	// Phase 35e: on the level default, the company goes for an enemy healer
	// first.
	HealersFirst bool `json:"healers_first,omitempty"`
}

// companyStructure is what changes only with the roster, formation, load,
// chemistry, checkpoint, or tactics.
type companyStructure struct {
	Leader     companyMember   `json:"leader"`
	Members    []companyMember `json:"members"`
	Alive      int             `json:"alive"`
	Dead       int             `json:"dead"`
	Load       *companyLoad    `json:"load"`
	Checkpoint *string         `json:"checkpoint"`
	Tactics    companyTactics  `json:"tactics"`
}

// companyLive is what changes as time passes: health, needs, and the
// countdowns (activity, rest, a fallen member's rescue time). Countdowns
// are in whole minutes, the client's display granularity, so they don't
// change every round. "Company.Vitals" carries all of it.
type companyLive struct {
	Vitals   map[string]companyVitals `json:"vitals"`
	Activity *string                  `json:"activity"`
	Rest     *companyRest             `json:"rest"`
	// Rescue is each fallen member's rescue time left, by member key.
	Rescue map[string]int `json:"rescue"`
}

// companyPayload is the full "Company" snapshot.
type companyPayload struct {
	companyStructure
	companyLive
}

// wholeMinutes rounds a countdown down to whole minutes, in seconds;
// under a minute it is kept to the second, so "45s" still shows.
func wholeMinutes(seconds int) int {
	if seconds < 60 {
		return max(seconds, 0)
	}
	return seconds - seconds%60
}

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }

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
	case company.MemberFled:
		return "fled"
	case company.MemberSeparated:
		return "separated"
	}
	return "present"
}

func vitalsOf(m companyview.Member) companyVitals {
	v := companyVitals{}
	if m.HasHP {
		v.HP, v.HPMax = intPtr(m.HP), intPtr(m.HPMax)
		if m.HPLimit > 0 && m.HPLimit < m.HPMax {
			v.HPLimit = intPtr(m.HPLimit)
		}
	}
	if m.HasMP {
		v.MP, v.MPMax = intPtr(m.MP), intPtr(m.MPMax)
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
	if !m.Strategy.IsZero() {
		out.Strategy = &companyStrategy{Role: string(m.Strategy.Role), Target: string(m.Strategy.Rule),
			Abilities: strategy.Names(m.Abilities), AbilitiesOff: m.Strategy.NoAbilities, Reserve: m.Strategy.Reserve}
	}
	if !m.Leader {
		points := m.TrainingPoints
		out.TrainingPoints = &points
		if len(m.Skills) > 0 {
			out.Skills = m.Skills
		}
	}
	// Chemistry is shown only for a member standing with a band; alone or
	// dead there is none (not "Strangers" on everyone).
	if m.Status != company.MemberDead {
		if standing, ok := chemistry(leaderUserID, m.Key); ok && standing.Together >= 2 {
			out.Chemistry = strPtr(company.TierName(standing.Tier))
		}
	}
	return out
}

// buildCompanyPayload turns a summary into the snapshot. ok is false when
// the company can't be read; the empty payload is then sent instead.
func buildCompanyPayload(leaderUserID int, s companyview.Summary, chemistry chemistryFunc) (companyPayload, bool) {
	if !s.CompanyKnown {
		return companyPayload{}, false
	}
	p := companyPayload{companyLive: companyLive{Vitals: map[string]companyVitals{}, Rescue: map[string]int{}}}
	p.Leader = memberOf(s.Leader, leaderUserID, chemistry)
	p.Vitals[string(s.Leader.Key)] = vitalsOf(s.Leader)
	p.Members = []companyMember{}
	for _, m := range s.Companions {
		p.Members = append(p.Members, memberOf(m, leaderUserID, chemistry))
		p.Vitals[string(m.Key)] = vitalsOf(m)
		if m.Status == company.MemberDead {
			p.Rescue[string(m.Key)] = wholeMinutes(int(m.RescueLeft.Seconds()))
		}
	}
	markWards(&p, s)
	p.Alive, p.Dead = s.Alive, s.Dead
	if s.LoadKnown {
		p.Load = &companyLoad{Label: s.LoadLabel, TotalG: s.Load.TotalGrams(), CapacityG: s.Load.CapacityGrams, CargoG: s.Load.CargoGrams, CompanionG: s.Load.CompanionGrams}
	}
	if s.ActivityKnown {
		p.Activity = strPtr(s.Activity.Label())
	}
	if s.RestKnown {
		p.Rest = &companyRest{Tier: s.RestTier.String(), Seconds: wholeMinutes(int(s.RestLeft.Seconds()))}
	}
	if s.Checkpoint != "" {
		p.Checkpoint = strPtr(s.Checkpoint)
	}
	t := s.Tactics.Resolve()
	p.Tactics = companyTactics{Focus: string(t.Focus), Healing: t.Healing, Patch: t.Patch, HealersFirst: s.HealersFirst}
	return p, true
}

// companySent is what was last sent to one user.
type companySent struct {
	structure string
	vitals    string
}

// companyExtra is one more message the feed keeps current (Phase 32g):
// built for a user each refresh, sent only when it changed. A nil build
// result sends nothing.
type companyExtra struct {
	module string
	build  func(user *users.UserRecord) []byte
	// buildKeyed, when set, replaces build: key decides whether the body
	// changed, so a value that ticks every round (a countdown the client
	// runs itself) does not resend an otherwise unchanged message. prev is
	// the key last stored for this user (nil after a forget).
	buildKeyed func(user *users.UserRecord, prev []byte) (body, key []byte)
}

// companyFeed decides what to send. It runs on the game loop; mu only
// keeps tests honest.
type companyFeed struct {
	mu        sync.Mutex
	last      map[int]companySent
	extras    []companyExtra
	lastExtra map[int]map[string]string
	chemistry chemistryFunc
	send      func(userID int, module string, payload []byte)
	// accepting reports whether the user's connection may take GMCP; a
	// telnet client that hasn't accepted it gets nothing built.
	accepting func(userID int) bool
	// gearOpen holds the users whose client shows the Gear editor
	// (Company.Equipment open <slot>/closed), with the slot it shows; only
	// they get it built, previewing only that slot.
	gearOpen map[int]gearWatch
}

func newCompanyFeed() *companyFeed {
	f := &companyFeed{
		last:      map[int]companySent{},
		lastExtra: map[int]map[string]string{},
		chemistry: company.ChemistryStanding,
		send: func(userID int, module string, payload []byte) {
			events.AddToQueue(GMCPOut{UserId: userID, Module: module, Payload: payload})
		},
		accepting: nativeAccepting,
		gearOpen:  map[int]gearWatch{},
	}
	f.extras = []companyExtra{inventoryExtra(), equipmentExtra(f.watchingGear), conditionsExtra(), capabilitiesExtra(), campExtra(camping.CampStateOf), battleExtra(gatherBattle)}
	return f
}

type gearWatch struct {
	slot      string
	refreshed time.Time // when an open last asked for a refresh
}

// gearRefreshGap is the least time between the refreshes a user's Gear
// editor may ask for: quick for a person choosing slots, but a client
// repeating itself cannot make the game loop rebuild without pause.
const gearRefreshGap = 200 * time.Millisecond

// setGearOpen records whether a user's client shows the Gear editor and
// which slot it shows; an unknown slot (the client's word) is the weapon.
// It reports whether to refresh the user now: only when the editor opened
// or changed slot, and not within gearRefreshGap of the last (the next
// round's refresh then brings the slot).
func (f *companyFeed) setGearOpen(userID int, open bool, slot string) bool {
	known := false
	for _, s := range items.AllEquipSlots() {
		known = known || string(s) == slot
	}
	if !known {
		slot = string(items.Weapon)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !open {
		delete(f.gearOpen, userID)
		return false
	}
	before, was := f.gearOpen[userID]
	if was && before.slot == slot {
		return false
	}
	now := time.Now()
	refresh := !was || now.Sub(before.refreshed) >= gearRefreshGap
	next := gearWatch{slot: slot, refreshed: before.refreshed}
	if refresh {
		next.refreshed = now
	}
	f.gearOpen[userID] = next
	return refresh
}

// watchingGear reports the slot a user's Gear editor shows, if open.
func (f *companyFeed) watchingGear(userID int) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.gearOpen[userID]
	return w.slot, ok
}

// nativeAccepting is false only for a connection known to the GMCP module
// that hasn't accepted GMCP (a plain telnet client). An unknown one is
// left to dispatchGMCP, which negotiates.
func nativeAccepting(userID int) bool {
	if gmcpModule.cache == nil {
		return true
	}
	settings, known := gmcpModule.cache.Get(users.GetConnectionId(userID))
	return !known || settings.GMCPAccepted
}

// update sends a user's Company or Company.Vitals when it changed since
// the last send.
func (f *companyFeed) update(userID int, s companyview.Summary) {
	if f.accepting != nil && !f.accepting(userID) {
		f.forget(userID) // a full snapshot once GMCP is accepted
		return
	}
	p, ok := buildCompanyPayload(userID, s, f.chemistry)
	structure, err := json.Marshal(p.companyStructure)
	if err != nil {
		mudlog.Error("gmcp: Company payload", "error", err)
		return
	}
	vitals, _ := json.Marshal(p.companyLive)
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
		// The client stores the extras under Company, which this replaces:
		// send them again after it (32g review finding 1).
		f.mu.Lock()
		delete(f.lastExtra, userID)
		f.mu.Unlock()
	case prev.vitals != next.vitals:
		body, _ := json.Marshal(p.companyLive)
		f.send(userID, "Company.Vitals", body)
	}
}

// updateExtras sends each extra message that changed since its last send.
func (f *companyFeed) updateExtras(user *users.UserRecord) {
	if f.accepting != nil && !f.accepting(user.UserId) {
		return // update forgets the user, so all is sent once accepted
	}
	for _, extra := range f.extras {
		var body, key []byte
		if extra.buildKeyed != nil {
			f.mu.Lock()
			var prev []byte
			if last, ok := f.lastExtra[user.UserId][extra.module]; ok {
				prev = []byte(last)
			}
			f.mu.Unlock()
			body, key = extra.buildKeyed(user, prev)
		} else {
			body = extra.build(user)
			key = body
		}
		if body == nil {
			continue
		}
		f.mu.Lock()
		if f.lastExtra[user.UserId] == nil {
			f.lastExtra[user.UserId] = map[string]string{}
		}
		changed := f.lastExtra[user.UserId][extra.module] != string(key)
		f.lastExtra[user.UserId][extra.module] = string(key)
		f.mu.Unlock()
		if changed {
			f.send(user.UserId, extra.module, body)
		}
	}
}

// forget makes the next update send the full snapshot and every extra
// (login, copyover, a request) or drops the user (logout).
func (f *companyFeed) forget(userID int) {
	f.mu.Lock()
	delete(f.last, userID)
	delete(f.lastExtra, userID)
	f.mu.Unlock()
}

// prune drops users no longer online, in case one left without a
// PlayerDespawn.
func (f *companyFeed) prune(online []int) {
	live := make(map[int]bool, len(online))
	for _, id := range online {
		live[id] = true
	}
	f.mu.Lock()
	for id := range f.last {
		if !live[id] {
			delete(f.last, id)
		}
	}
	for id := range f.gearOpen {
		if !live[id] {
			delete(f.gearOpen, id)
		}
	}
	for id := range f.lastExtra {
		if !live[id] {
			delete(f.lastExtra, id)
		}
	}
	f.mu.Unlock()
}

// GMCPGearWatch says whether a user's client shows the Gear editor, whose
// Company.Equipment previews are built only while it does (Phase 34
// review): "!!GMCP(Company.Equipment open weapon)" or "... closed".
type GMCPGearWatch struct {
	UserId int
	Open   bool
	Slot   string // the slot the editor shows
}

func (g GMCPGearWatch) Type() string { return `GMCPGearWatch` }

// GMCPCompanyRequest asks for a user's full Company snapshot now.
type GMCPCompanyRequest struct {
	UserId int
}

func (g GMCPCompanyRequest) Type() string { return `GMCPCompanyRequest` }

var companyFeeds = newCompanyFeed()

func init() {
	companyview.OnRefresh.Register(func(r companyview.Refreshed) companyview.Refreshed {
		// Phase 29f: nothing runs ahead of a player's paced combat lines.
		if holdCompany(r.User.UserId) {
			return r
		}
		companyFeeds.update(r.User.UserId, r.Summary)
		companyFeeds.updateExtras(r.User)
		return r
	})
	events.RegisterListener(events.PlayerSpawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerSpawn); ok {
			companyFeeds.forget(evt.UserId)
			companyFeeds.setGearOpen(evt.UserId, false, "") // the client says again
		}
		return events.Continue
	})
	events.RegisterListener(GMCPGearWatch{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPGearWatch); ok {
			if companyFeeds.setGearOpen(evt.UserId, evt.Open, evt.Slot) {
				companyview.RefreshUser(evt.UserId)
			}
		}
		return events.Continue
	})
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			companyFeeds.forget(evt.UserId)
			companyFeeds.setGearOpen(evt.UserId, false, "")
			battleSeen.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.NewRound{}, func(events.Event) events.ListenerReturn {
		online := users.GetOnlineUserIds()
		companyFeeds.prune(online)
		battleSeen.prune(online)
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

// markWards fills each guardian's ward (Phase 30c2): dropped when it names
// no member, marked out of reach when both are placed more than one column
// apart (formationcombat.InLateralRange; unplaced fails open).
func markWards(p *companyPayload, s companyview.Summary) {
	all := append([]companyview.Member{s.Leader}, s.Companions...)
	byKey := map[string]companyview.Member{}
	for _, m := range all {
		byKey[string(m.Key)] = m
	}
	mark := func(out *companyMember, m companyview.Member) {
		if out.Strategy == nil || m.Strategy.Role != strategy.Guardian || m.Strategy.Ward == "" {
			return
		}
		ward, ok := byKey[m.Strategy.Ward]
		if !ok || ward.Key == m.Key {
			return
		}
		out.Strategy.Ward = m.Strategy.Ward
		if m.Placed && ward.Placed && !formationcombat.InLateralRange(m.Col, ward.Col) {
			out.Strategy.WardReach = boolPtr(false)
		}
	}
	mark(&p.Leader, s.Leader)
	for i, m := range s.Companions {
		mark(&p.Members[i], m)
	}
}
