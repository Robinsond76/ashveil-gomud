package company

import "github.com/GoMudEngine/GoMud/internal/errands"

// Phase 70: a companion on an errand is off the map until it returns. It
// is absent everywhere a separated companion is (no live mob, no load, no
// battles, no camp, no banter), but its own countdown is real time and
// saved, and it comes back with its outcome rather than by finding the way.

// OnErrand reports whether the companion is away on an errand.
func (c Companion) OnErrand() bool { return c.Errand != nil }

// Away reports whether the companion is off the map: separated from its
// leader (Phase 33h3) or on an errand (Phase 70). Use it wherever a member
// must be present; use Separated only for the separation's own countdown.
func (c Companion) Away() bool { return c.Separation != nil || c.Errand != nil }

func (r Record) isAway(key MemberKey) bool {
	id, ok := CompanionIDFromMemberKey(key)
	if !ok {
		return false
	}
	i := r.companionIndex(id)
	return i >= 0 && r.Companions[i].OnErrand()
}

// ErrandRow is one companion's errand as shown.
type ErrandRow struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	// State is "away" (on an errand), "ready" (can be sent), or "busy"
	// (cannot be sent now, with Why).
	State string `json:"state"`
	Why   string `json:"why,omitempty"`
	// The errand, when away.
	Kind      string `json:"kind,omitempty"`
	KindLabel string `json:"kind_label,omitempty"`
	Length    string `json:"length,omitempty"`
	Zone      string `json:"zone,omitempty"`
	ReturnsAt int64  `json:"returns_at,omitempty"`
	Remaining int64  `json:"remaining,omitempty"` // seconds left
	Due       bool   `json:"due,omitempty"`
	// Waiting says why a due errand has not come back yet.
	Waiting string `json:"waiting,omitempty"`
}

// ErrandOption is one job a companion can be sent on.
type ErrandOption struct {
	Kind    string         `json:"kind"`
	Label   string         `json:"label"`
	Blurb   string         `json:"blurb"`
	Lengths []ErrandLength `json:"lengths"`
}

// ErrandLength is one span with the pay it earns here (gold, for a gold
// outcome) and the wound risk for the companion the panel was built for.
type ErrandLength struct {
	Length  string `json:"length"`
	Label   string `json:"label"`
	Seconds int64  `json:"seconds"`
}

// ErrandPanel is the company's errands, in words.
type ErrandPanel struct {
	// Now is the server's real time (Unix seconds) when the panel was
	// built, so the client can run each countdown itself.
	Now int64 `json:"now"`
	// Here is true when the leader stands where companions can be sent
	// (an inn); Where says where they can, or why not here.
	Here    bool           `json:"here"`
	Where   string         `json:"where"`
	Zone    string         `json:"zone,omitempty"`
	Band    string         `json:"band,omitempty"`
	Rows    []ErrandRow    `json:"rows"`
	Options []ErrandOption `json:"options"`
	// Recent is the chronicle's latest errand deeds, newest first.
	Recent []string `json:"recent"`
}

// ErrandViewer is implemented by the company module.
type ErrandViewer interface {
	ErrandPanel(leaderUserID int) (ErrandPanel, bool)
}

// ErrandsOf is the leader's errands; false when no company module can say.
func ErrandsOf(leaderUserID int) (ErrandPanel, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if v, ok := p.(ErrandViewer); ok {
		return v.ErrandPanel(leaderUserID)
	}
	return ErrandPanel{}, false
}

// ErrandOptions are the jobs and spans a companion can be sent on.
func ErrandOptions() []ErrandOption {
	out := make([]ErrandOption, 0, len(errands.Kinds))
	for _, k := range errands.Kinds {
		opt := ErrandOption{Kind: string(k.Kind), Label: k.Label, Blurb: k.Blurb}
		for _, l := range errands.Lengths {
			opt.Lengths = append(opt.Lengths, ErrandLength{Length: string(l.Length), Label: l.Label, Seconds: l.Seconds})
		}
		out = append(out, opt)
	}
	return out
}

// SendAway puts the companion on its errand and takes it out of the
// formation, remembering the cell it held (as a death does).
func (r *Record) SendAway(companionID int, e errands.Errand) bool {
	i := r.companionIndex(companionID)
	if i < 0 {
		return false
	}
	key := CompanionMemberKey(companionID)
	e.Row, e.Col, e.Placed = r.Formation.Find(key)
	if !e.Placed {
		e.Row, e.Col = 0, 0
	}
	r.Formation.Clear(key)
	r.Companions[i].Errand = &e
	return true
}

// BringBack ends the companion's errand. It takes the cell it left when
// that cell is still free.
func (r *Record) BringBack(companionID int) {
	i := r.companionIndex(companionID)
	if i < 0 || r.Companions[i].Errand == nil {
		return
	}
	e := r.Companions[i].Errand
	r.Companions[i].Errand = nil
	if e.Placed && r.Formation.At(e.Row, e.Col) == "" {
		_ = r.Formation.Place(CompanionMemberKey(companionID), e.Row, e.Col)
	}
}
