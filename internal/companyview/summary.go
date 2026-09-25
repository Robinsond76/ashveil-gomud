package companyview

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Member is one company member: the leader, or a companion keyed by ID.
type Member struct {
	Key    company.MemberKey
	ID     int // 0 for the leader
	Leader bool
	Name   string
	Status company.MemberStatus
	Level  int
	// Archetype is the display name; "" when none. ArchetypeKnown is false
	// when no provider can say (the leader only).
	Archetype      string
	ArchetypeKnown bool
	HasHP          bool
	HP, HPMax      int
	// Hunger, Thirst, and Fatigue are unknown for the dead and whenever
	// survival can't report them.
	Hunger, Thirst, Fatigue Need
	// Warmth is the exposure label ("" when comfortable); WarmthKnown is
	// false when exposure can't report it.
	Warmth      string
	WarmthKnown bool
	Placed      bool
	Row, Col    int
	// RescueLeft is a dead companion's rescue allowance left.
	RescueLeft time.Duration
}

// Summary is a player and their company, as every surface shows them.
// Each Known flag is false when the owning provider can't report that
// value; it is then never shown as a healthy default.
type Summary struct {
	Leader     Member
	Companions []Member
	// CompanyKnown is false when the company can't be read; Companions,
	// Alive, and Dead then cover the leader only.
	CompanyKnown bool
	Alive, Dead  int

	LoadKnown bool
	Load      encumbrance.Load
	LoadLabel string

	ActivityKnown bool
	Activity      Activity

	RestKnown bool
	RestTier  camping.Tier
	RestLeft  time.Duration

	LightKnown bool
	Light      int // 0 dark, 1 dim, 2 lit

	// Alignment is the leader's, on the −100..100 display scale.
	Alignment int

	// Checkpoint is the title of the church where the player wakes; "" when
	// none is recorded.
	Checkpoint string
}

// sources are the providers a summary reads; natives unless a test swaps
// them.
type sources struct {
	members   func(leaderUserID int) ([]company.MemberView, bool)
	needs     func(leaderUserID int) []survival.MemberNeeds
	load      func(leaderUserID int) (encumbrance.Load, bool)
	band      func(leaderUserID int) (encumbrance.LoadBand, bool)
	journey   func(leaderUserID int) (expedition.Progress, bool)
	rest      func(leaderUserID int) (camping.RestActivity, bool)
	tier      func(userID int) (camping.Tier, time.Duration, bool)
	exposure  func(leaderUserID int, memberKey string) (int, bool)
	light     func(user *users.UserRecord) (int, bool)
	archetype func(userID int) (string, bool)
	// The *Reporting seams tell "none" from "no provider to ask".
	archetypeReporting func() bool
	journeyReporting   func() bool
	restReporting      func() bool
	name               func(archetypeID string) (string, bool)
	room               func(roomID int) *rooms.Room
	formation          func(leaderUserID int) (company.Formation, bool)
}

func nativeSources() sources {
	return sources{
		members:            company.CompanyMembers,
		needs:              survival.CompanyNeeds,
		load:               encumbrance.CurrentLoad,
		band:               encumbrance.CurrentBand,
		journey:            expedition.JourneyProgress,
		rest:               camping.LeaderRest,
		tier:               camping.RestTierOf,
		exposure:           climate.ExposureOf,
		light:              nativeLight,
		archetype:          archetypes.PlayerArchetype,
		archetypeReporting: archetypes.Active,
		journeyReporting:   expedition.ProgressReporting,
		restReporting:      camping.RestReporting,
		name:               archetypes.Name,
		room:               rooms.LoadRoom,
		formation:          company.FormationFor,
	}
}

func nativeLight(user *users.UserRecord) (int, bool) {
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return 0, false
	}
	return room.VisibilityForUser(user), true
}

// For builds a user's summary. Call it on the game loop: it reads the
// company module, which has no mutex, and live mobs.
func For(user *users.UserRecord) Summary {
	return nativeSources().summary(user)
}

func (src sources) archetypeName(id string) string {
	if id == "" {
		return ""
	}
	if name, ok := src.name(id); ok {
		return name
	}
	return id
}

func needsOf(n survival.Needs) (hunger, thirst, fatigue Need) {
	return Need{Known: true, Value: n.Hunger, Label: survival.HungerLabel(n.Hunger)},
		Need{Known: true, Value: n.Thirst, Label: survival.ThirstLabel(n.Thirst)},
		Need{Known: true, Value: n.Fatigue, Label: survival.FatigueLabel(n.Fatigue)}
}

func (src sources) summary(user *users.UserRecord) Summary {
	c := user.Character
	uid := user.UserId
	s := Summary{Alive: 1, Alignment: company.DisplayAlignment(int(c.Alignment))}

	s.Leader = Member{Key: company.LeaderMemberKey, Leader: true, Name: c.Name, Status: company.MemberPresent,
		Level: c.Level, HasHP: true, HP: c.Health, HPMax: c.HealthMax.Value}
	if f, ok := src.formation(uid); ok {
		s.Leader.Row, s.Leader.Col, s.Leader.Placed = f.Find(company.LeaderMemberKey)
		if !s.Leader.Placed {
			s.Leader.Row, s.Leader.Col = 0, 0
		}
	}
	if src.archetypeReporting() {
		s.Leader.ArchetypeKnown = true
		if id, ok := src.archetype(uid); ok {
			s.Leader.Archetype = src.archetypeName(id)
		}
	}

	needs := map[company.MemberKey]survival.Needs{}
	for _, n := range src.needs(uid) {
		needs[n.Key] = n.Needs
	}
	if n, ok := needs[company.LeaderMemberKey]; ok {
		s.Leader.Hunger, s.Leader.Thirst, s.Leader.Fatigue = needsOf(n)
	}
	if e, ok := src.exposure(uid, string(company.LeaderMemberKey)); ok {
		s.Leader.Warmth, s.Leader.WarmthKnown = WarmthLabel(e), true
	}

	if views, ok := src.members(uid); ok {
		s.CompanyKnown = true
		for _, v := range views {
			m := Member{Key: company.CompanionMemberKey(v.ID), ID: v.ID, Name: v.Name, Status: v.Status, Level: v.Level,
				Archetype: src.archetypeName(v.Archetype), Placed: v.Placed, Row: v.Row, Col: v.Col}
			switch v.Status {
			case company.MemberDead:
				s.Dead++
				m.RescueLeft = time.Duration(v.RescueSeconds) * time.Second
			default:
				s.Alive++
				if v.Status == company.MemberPresent {
					m.HasHP, m.HP, m.HPMax = true, v.HP, v.HPMax
				}
				if n, ok := needs[m.Key]; ok {
					m.Hunger, m.Thirst, m.Fatigue = needsOf(n)
				}
				if e, ok := src.exposure(uid, string(m.Key)); ok {
					m.Warmth, m.WarmthKnown = WarmthLabel(e), true
				}
			}
			s.Companions = append(s.Companions, m)
		}
	}

	if load, ok := src.load(uid); ok {
		band, _ := src.band(uid)
		s.LoadKnown, s.Load, s.LoadLabel = true, load, LoadLabel(load, band)
	}

	if p, ok := src.journey(uid); ok {
		s.ActivityKnown = true
		s.Activity = Activity{Kind: Travelling, Percent: p.Percent, Remaining: p.Remaining, Route: p.Route}
		if p.Interrupted {
			s.Activity.Kind = Stopped
		}
	} else if r, ok := src.rest(uid); ok {
		s.ActivityKnown = true
		switch {
		case r.Inn:
			s.Activity = Activity{Kind: InnStay, Remaining: r.Remaining}
		case r.Resting:
			s.Activity = Activity{Kind: CampRest, Remaining: r.Remaining}
		default:
			s.Activity = Activity{Kind: Camped}
		}
	} else if src.journeyReporting() && src.restReporting() {
		// Both can report, and neither has anything: idle.
		s.ActivityKnown = true
	}

	if tier, left, ok := src.tier(uid); ok {
		s.RestKnown, s.RestTier, s.RestLeft = true, tier, left
	}

	if level, ok := src.light(user); ok {
		s.LightKnown, s.Light = true, level
	}

	if id := checkpointID(c.GetMiscData(death.CheckpointKey)); id > 0 {
		s.Checkpoint = src.roomTitle(id)
	}
	return s
}

var (
	titleMu sync.Mutex
	titles  = map[int]string{}
)

// roomTitle is a room's title, remembered once read so a refresh never
// reloads an unloaded church from disk.
func (src sources) roomTitle(roomID int) string {
	titleMu.Lock()
	title, ok := titles[roomID]
	titleMu.Unlock()
	if ok {
		return title
	}
	room := src.room(roomID)
	if room == nil {
		return ""
	}
	titleMu.Lock()
	titles[roomID] = room.Title
	titleMu.Unlock()
	return room.Title
}

// checkpointID reads the checkpoint, which YAML may bring back as another
// integer type.
func checkpointID(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	}
	return 0
}

// WarnWords are the prompt's warnings, in their fixed order: the leader's
// hunger, thirst, and fatigue at Low or worse, then darkness, warmth, and
// an overloaded company.
func (s Summary) WarnWords() []string {
	var words []string
	for _, n := range []Need{s.Leader.Hunger, s.Leader.Thirst, s.Leader.Fatigue} {
		if n.Warns() {
			words = append(words, n.Label)
		}
	}
	if s.LightKnown {
		words = append(words, LightLabel(s.Light))
	}
	words = append(words, s.Leader.Warmth)
	if s.LoadLabel == "Overloaded" {
		words = append(words, s.LoadLabel)
	}
	return words
}
