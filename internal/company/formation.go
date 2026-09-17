package company

import (
	"errors"
	"fmt"
)

const (
	FormationRows = 3
	FormationCols = 3
)

// MemberKey identifies a company member within a formation grid.
type MemberKey string

// LeaderMemberKey is the formation key for the company leader.
const LeaderMemberKey MemberKey = "leader"

// CompanionMemberKey returns the formation key for a companion ID.
func CompanionMemberKey(id int) MemberKey {
	return MemberKey(fmt.Sprintf("companion:%d", id))
}

// Formation is a 3x3 grid of member keys. The empty key means unoccupied.
type Formation [FormationRows][FormationCols]MemberKey

var (
	ErrInvalidSlot   = errors.New("formation slot is out of range")
	ErrSlotOccupied  = errors.New("formation slot is occupied")
	ErrUnknownMember = errors.New("member is not part of the formation")
)

// At returns the occupant of a cell, or "" when empty or out of range.
func (f *Formation) At(row, col int) MemberKey {
	if !validSlot(row, col) {
		return ""
	}
	return f[row][col]
}

// Find returns the cell occupied by key.
func (f *Formation) Find(key MemberKey) (int, int, bool) {
	if key == "" {
		return 0, 0, false
	}
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] == key {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// Place moves key to (row, col), clearing any previous cell.
func (f *Formation) Place(key MemberKey, row, col int) error {
	if key == "" {
		return ErrUnknownMember
	}
	if !validSlot(row, col) {
		return ErrInvalidSlot
	}
	if occupant := f[row][col]; occupant != "" && occupant != key {
		return ErrSlotOccupied
	}
	f.Clear(key)
	f[row][col] = key
	return nil
}

// Swap exchanges the cells of two placed members.
func (f *Formation) Swap(a, b MemberKey) error {
	if a == "" || b == "" {
		return ErrUnknownMember
	}
	if a == b {
		return nil
	}
	ar, ac, aok := f.Find(a)
	br, bc, bok := f.Find(b)
	if !aok || !bok {
		return ErrUnknownMember
	}
	f[ar][ac], f[br][bc] = f[br][bc], f[ar][ac]
	return nil
}

// Clear removes a member from every cell.
func (f *Formation) Clear(key MemberKey) {
	if key == "" {
		return
	}
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] == key {
				f[r][c] = ""
			}
		}
	}
}

// Prune removes every member not present in valid.
func (f *Formation) Prune(valid map[MemberKey]bool) {
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] != "" && !valid[f[r][c]] {
				f[r][c] = ""
			}
		}
	}
}

func (f *Formation) empty() bool {
	for r := 0; r < FormationRows; r++ {
		for c := 0; c < FormationCols; c++ {
			if f[r][c] != "" {
				return false
			}
		}
	}
	return true
}

func validSlot(row, col int) bool {
	return row >= 0 && row < FormationRows && col >= 0 && col < FormationCols
}
