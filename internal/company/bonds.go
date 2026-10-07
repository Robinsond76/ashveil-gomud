package company

import (
	"maps"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/bonds"
)

// Bond is the saved feeling between two companions (Phase 65): one value
// from -100 to 100 per pair, kept on the leader's record. A and B are the
// companions' numbers with A < B, so a pair has one entry whichever way it
// is asked.
type Bond struct {
	A     int `yaml:"a"`
	B     int `yaml:"b"`
	Value int `yaml:"value"`
	// Warned is set once the leader has been told one of a rivalry will
	// leave; it is cleared when the pair mends.
	Warned bool `yaml:"warned,omitempty"`
	// At is when (unix seconds, real time) each source last moved the pair,
	// the cooldown that keeps a repeated act from farming a friendship.
	At map[string]int64 `yaml:"at,omitempty"`
}

func (b Bond) clone() Bond {
	b.At = maps.Clone(b.At)
	return b
}

// BondPair orders two companion numbers.
func BondPair(a, b int) (int, int) {
	if a > b {
		return b, a
	}
	return a, b
}

// BondOf is the pair's bond, and false for a pair with none yet.
func (r Record) BondOf(a, b int) (Bond, bool) {
	a, b = BondPair(a, b)
	for _, bond := range r.Bonds {
		if bond.A == a && bond.B == b {
			return bond, true
		}
	}
	return Bond{A: a, B: b}, false
}

// SetBond stores a bond, replacing the pair's entry. Callers hold a record
// from Registry.Get, whose Bonds are their own copy.
func (r *Record) SetBond(bond Bond) {
	bond.A, bond.B = BondPair(bond.A, bond.B)
	for i, old := range r.Bonds {
		if old.A == bond.A && old.B == bond.B {
			r.Bonds[i] = bond
			return
		}
	}
	r.Bonds = append(r.Bonds, bond)
}

func cloneBonds(in []Bond) []Bond {
	if in == nil {
		return nil
	}
	out := make([]Bond, len(in))
	for i, b := range in {
		out[i] = b.clone()
	}
	return out
}

// pruneBonds keeps the bonds of companions still in the record, clamps
// values, and drops duplicates and self-pairs. Dismissal, desertion and
// death-by-expiry end a companion's bonds in the same save.
func pruneBonds(in []Bond, record Record) []Bond {
	ids := map[int]bool{}
	for _, c := range record.Companions {
		ids[c.ID] = true
	}
	seen := map[[2]int]bool{}
	var out []Bond
	for _, b := range in {
		b.A, b.B = BondPair(b.A, b.B)
		key := [2]int{b.A, b.B}
		if b.A == b.B || !ids[b.A] || !ids[b.B] || seen[key] {
			continue
		}
		seen[key] = true
		b.Value = max(bonds.Min, min(bonds.Max, b.Value))
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
	return out
}

// BondProvider is implemented by the company module: combat asks how two
// members feel (a friend steps in, a rival won't), and reports the moments
// that change a bond.
type BondProvider interface {
	// BondValue is the bond between two members of the leader's company;
	// zero for a pair with none, or when either is the leader.
	BondValue(leaderUserID int, a, b MemberKey) int
	// BondEvent reports that one member stepped in for another (bonds.Rescue)
	// or let a rival take the blow (bonds.Refusal).
	BondEvent(leaderUserID int, a, b MemberKey, source bonds.Source)
}

func bondProvider() BondProvider {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	bp, _ := p.(BondProvider)
	return bp
}

// BondValue is the bond between two members; zero without a provider. Combat
// calls it on the game loop for each blow, so it reads in place.
func BondValue(leaderUserID int, a, b MemberKey) int {
	if bp := bondProvider(); bp != nil {
		return bp.BondValue(leaderUserID, a, b)
	}
	return 0
}

// BondEvent reports a moment in a battle that changes a bond.
func BondEvent(leaderUserID int, a, b MemberKey, source bonds.Source) {
	if bp := bondProvider(); bp != nil {
		bp.BondEvent(leaderUserID, a, b, source)
	}
}

// BondRow is one pair as shown: A and B are the companions' numbers.
type BondRow struct {
	A      int    `json:"a"`
	B      int    `json:"b"`
	AName  string `json:"a_name"`
	BName  string `json:"b_name"`
	Value  int    `json:"value"`
	Tier   int    `json:"tier"`
	Phrase string `json:"phrase"` // "Merek and Ysolde trust each other"
	// Effect is what it does in battle, in words ("" when nothing).
	Effect string `json:"effect"`
	Warned bool   `json:"warned"`
}

// BondFeeling is one member's view of another, for the per-member lines
// ("trusts Ysolde").
type BondFeeling struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Words string `json:"words"` // "trusts Ysolde"
	Tier  int    `json:"tier"`
}

// BondMember is one companion with how it feels about each of the others.
type BondMember struct {
	ID       int           `json:"id"`
	Name     string        `json:"name"`
	Feelings []BondFeeling `json:"feelings"`
}

// BondPanel is the company's bonds, in words.
type BondPanel struct {
	Pairs   []BondRow    `json:"pairs"`
	Members []BondMember `json:"members"`
}

// BondViewer is implemented by the company module.
type BondViewer interface {
	BondPanel(leaderUserID int) (BondPanel, bool)
}

// BondsOf is the leader's companions' bonds; false when no company module
// can say.
func BondsOf(leaderUserID int) (BondPanel, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if v, ok := p.(BondViewer); ok {
		return v.BondPanel(leaderUserID)
	}
	return BondPanel{}, false
}
