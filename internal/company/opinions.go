package company

import (
	"maps"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/opinions"
)

// OpinionNote is one thing a companion said about a choice (Phase 64).
type OpinionNote struct {
	Kind    string `yaml:"kind"`
	Verdict int    `yaml:"verdict"` // opinions.Likes or opinions.Dislikes
	At      int64  `yaml:"at"`      // real time, unix seconds
	Subject string `yaml:"subject,omitempty"`
}

// OpinionMemory is a companion's memory of its own opinions: the last few
// reactions, newest last, and when it last spoke on each kind (the
// cooldown that keeps a repeated choice from farming loyalty).
type OpinionMemory struct {
	Notes []OpinionNote    `yaml:"notes,omitempty"`
	Spoke map[string]int64 `yaml:"spoke,omitempty"`
}

// Clone is a deep copy.
func (m *OpinionMemory) Clone() *OpinionMemory {
	if m == nil {
		return nil
	}
	return &OpinionMemory{Notes: slices.Clone(m.Notes), Spoke: maps.Clone(m.Spoke)}
}

// Remember adds a note, keeping the newest opinions.MaxNotes, and marks the
// kind as spoken on at the note's time.
func (m *OpinionMemory) Remember(n OpinionNote) {
	m.Notes = append(m.Notes, n)
	if len(m.Notes) > opinions.MaxNotes {
		m.Notes = slices.Clone(m.Notes[len(m.Notes)-opinions.MaxNotes:])
	}
	if m.Spoke == nil {
		m.Spoke = map[string]int64{}
	}
	m.Spoke[n.Kind] = n.At
}

// OpinionProvider is implemented by the company module: the leader made a
// choice, and the companions who saw it react.
type OpinionProvider interface {
	Opinion(leaderUserID int, c opinions.Choice) ([]string, error)
}

// Opinion reports a choice the leader made, from its real source. It
// returns what the companions say, one line each, and nil without a
// provider or when nobody has an opinion. A companion reacts at most once
// per choice, and again to the same kind only after its cooldown.
func Opinion(leaderUserID int, c opinions.Choice) ([]string, error) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if op, ok := p.(OpinionProvider); ok {
		return op.Opinion(leaderUserID, c)
	}
	return nil, nil
}

// OpinionNoteView is one remembered reaction, as shown.
type OpinionNoteView struct {
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Verdict int    `json:"verdict"` // 1 approved, -1 disapproved
	Ago     string `json:"ago"`
	Subject string `json:"subject,omitempty"`
}

// OpinionRow is one companion's opinions, in words.
type OpinionRow struct {
	Key         string            `json:"key"`
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Personality string            `json:"personality"`
	Loyalty     int               `json:"loyalty"`
	Mood        string            `json:"mood"`
	Likes       []string          `json:"likes"`
	Dislikes    []string          `json:"dislikes"`
	Notes       []OpinionNoteView `json:"notes"`
	// Deeds are the newest chronicle deeds that name this companion.
	Deeds []string `json:"deeds"`
}

// OpinionPanel is every living companion's opinions plus what the company
// has seen the leader do (lifetime counts from the chronicle).
type OpinionPanel struct {
	Spared   int          `json:"spared"`
	Executed int          `json:"executed"`
	Members  []OpinionRow `json:"members"`
}

// OpinionViewer is implemented by the company module.
type OpinionViewer interface {
	OpinionPanel(leaderUserID int) (OpinionPanel, bool)
}

// OpinionsOf is the leader's companions' opinions; false when no company
// module can say.
func OpinionsOf(leaderUserID int) (OpinionPanel, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if v, ok := p.(OpinionViewer); ok {
		return v.OpinionPanel(leaderUserID)
	}
	return OpinionPanel{}, false
}
