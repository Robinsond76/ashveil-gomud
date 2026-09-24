package company

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// MemberState is a companion's durable progression and gear (Phase 22b).
// The live mob is rebuilt from it on every restore, so gear given to a
// companion survives logout, restart, and copyover, and template gear is
// minted only once.
type MemberState struct {
	Level      int             `yaml:"level"`
	Experience int             `yaml:"experience,omitempty"`
	Equipment  characters.Worn `yaml:"equipment,omitempty"`
	Items      []items.Item    `yaml:"items,omitempty"`
}

func cloneItem(i items.Item) items.Item {
	if i.Adjectives != nil {
		i.Adjectives = append([]string(nil), i.Adjectives...)
	}
	return i
}

// Clone returns a deep copy.
func (s MemberState) Clone() MemberState {
	out := s
	for _, slot := range characters.AllSlots() {
		if itm := s.Equipment.Get(slot); itm != nil {
			out.Equipment.Set(slot, cloneItem(*itm))
		}
	}
	if s.Items != nil {
		out.Items = make([]items.Item, len(s.Items))
		for i, itm := range s.Items {
			out.Items[i] = cloneItem(itm)
		}
	}
	return out
}

// ClearGear drops every worn and carried item, keeping progression.
func (s *MemberState) ClearGear() {
	s.Equipment = characters.Worn{}
	s.Items = nil
}

// SetState records a companion's state. It returns ErrUnknownMember for a
// leader or companion not in the registry.
func (r *Registry) SetState(leaderUserID, companionID int, state MemberState) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID == companionID {
			s := state.Clone()
			record.Companions[i].State = &s
			r.Put(record)
			return nil
		}
	}
	return ErrUnknownMember
}
