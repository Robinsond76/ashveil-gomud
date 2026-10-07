package company

import (
	"slices"
	"strconv"
)

// MaxRites is how many unanswered rites a record keeps; one more drops the
// oldest (a company would have to lose five companions between two camps).
const MaxRites = 5

// Rite is an occasion for mourning (Phase 74): a companion who was lost for
// good, or who left after long service. It is queued in the same save as the
// departure and waits for the leader to hold it at a camp or an inn, or to
// let it pass. Close and Band are the companion numbers of those who
// trusted the one gone and of everyone else, fixed at the departure because
// the bond ends with the companion.
type Rite struct {
	// Op is the occasion's name, so a retry queues it once.
	Op string `yaml:"op"`
	// Companion is the departed's number, never reused in a company.
	Companion     int    `yaml:"companion"`
	MobTemplateID int    `yaml:"mob_template_id,omitempty"`
	Name          string `yaml:"name"`
	Level         int    `yaml:"level,omitempty"`
	// Cause is "lost" or "left" (internal/rites).
	Cause string `yaml:"cause"`
	Close []int  `yaml:"close,omitempty"`
	Band  []int  `yaml:"band,omitempty"`
	// Offered is set once a camp or an inn stay has announced the rite; a
	// later camp or stay that finds it still unanswered lets it pass.
	Offered bool  `yaml:"offered,omitempty"`
	At      int64 `yaml:"at,omitempty"`
}

func (r Rite) clone() Rite {
	r.Close = slices.Clone(r.Close)
	r.Band = slices.Clone(r.Band)
	return r
}

func cloneRites(in []Rite) []Rite {
	if in == nil {
		return nil
	}
	out := make([]Rite, len(in))
	for i, r := range in {
		out[i] = r.clone()
	}
	return out
}

// RiteOp is the occasion name for a companion and a cause.
func RiteOp(companionID int, cause string) string {
	return "rite:" + cause + ":" + strconv.Itoa(companionID)
}

// RiteOf finds a pending rite by the departed's companion number.
func (r Record) RiteOf(companionID int) (Rite, bool) {
	for _, rite := range r.Rites {
		if rite.Companion == companionID {
			return rite, true
		}
	}
	return Rite{}, false
}

// QueueRite adds a rite and reports whether it was new (the same occasion is
// queued once). Past MaxRites the oldest is dropped.
func (r *Record) QueueRite(rite Rite) bool {
	for _, old := range r.Rites {
		if old.Op == rite.Op {
			return false
		}
	}
	r.Rites = append(r.Rites, rite.clone())
	if len(r.Rites) > MaxRites {
		r.Rites = cloneRites(r.Rites[len(r.Rites)-MaxRites:])
	}
	return true
}

// RemoveRite drops the rite with this occasion name and reports whether it
// was there.
func (r *Record) RemoveRite(op string) bool {
	before := len(r.Rites)
	r.Rites = slices.DeleteFunc(r.Rites, func(rite Rite) bool { return rite.Op == op })
	if len(r.Rites) == 0 {
		r.Rites = nil
	}
	return len(r.Rites) != before
}

// RitesProvider is implemented by the company module: modules/camping calls
// it as a camp or an inn stay begins, so the rites waiting are announced
// (and the ones left unanswered at the last stay pass).
type RitesProvider interface {
	OfferRites(leaderUserID int) string
}

// OfferRites calls through to the provider; "" without one.
func OfferRites(leaderUserID int) string {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if rp, ok := p.(RitesProvider); ok {
		return rp.OfferRites(leaderUserID)
	}
	return ""
}

// RiteRow is one rite waiting, for the Rites tab.
type RiteRow struct {
	// ID is the departed's companion number: `rite hold #ID`.
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	// Cause is in words: "lost for good", "left the company".
	Cause string `json:"cause"`
	// Close are the names of the companions who trusted them.
	Close   []string `json:"close"`
	Offered bool     `json:"offered"`
}

// RitePanel is the Rites tab: what waits, and whether the leader is at a
// camp or an inn where it can be held.
type RitePanel struct {
	Here  bool      `json:"here"`
	Where string    `json:"where"`
	Rows  []RiteRow `json:"rows"`
}

// RiteViewer is implemented by the company module.
type RiteViewer interface {
	RitePanel(leaderUserID int) (RitePanel, bool)
}

// RitesOf is the leader's rites; false when no company module can say.
func RitesOf(leaderUserID int) (RitePanel, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if v, ok := p.(RiteViewer); ok {
		return v.RitePanel(leaderUserID)
	}
	return RitePanel{}, false
}
