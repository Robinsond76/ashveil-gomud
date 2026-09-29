// Package mount contains the GoMud-free durable horse herd, matching the
// domain/module split of internal/expedition, internal/camping,
// internal/weather, and internal/encumbrance.
//
// A horse carries no decaying state (no fatigue, health, or feed): handoff
// §35 defers those to "Later." Phase 32f turned the one mount per company
// into a herd: each member may keep one riding horse and one pack horse. A
// pack horse carries load; a riding horse carries one member. Each does
// its job fully only with a saddle of its own kind fitted.
package mount

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// PctMin and PctMax bound a MountSpec's TravelDurationPct and FatiguePct,
// matching weather.Condition's and modules/encumbrance's LoadBand
// convention. 100 means unchanged.
const (
	PctMin = 25
	PctMax = 300
)

var (
	ErrInvalidSpec  = errors.New("mount: invalid mount spec")
	ErrInvalidHerd  = errors.New("mount: invalid herd")
	ErrHerdFull     = errors.New("mount: no room for another horse of that kind")
	ErrUnknownHorse = errors.New("mount: no such horse")
)

// Kind is what a horse is for (Phase 32f).
type Kind string

const (
	// KindPack carries load and no one.
	KindPack Kind = "pack"
	// KindRiding carries one member.
	KindRiding Kind = "riding"
)

// MountSpec is one configured horse type.
type MountSpec struct {
	Type        string
	Description string
	Kind        Kind
	// Price is what stabling one costs, in gold.
	Price int
	// BareCapacityGrams is what the horse carries without a saddle;
	// SaddledCapacityGrams with its saddle fitted.
	BareCapacityGrams    int
	SaddledCapacityGrams int
	// TravelDurationPct is a route's duration multiplier when every member
	// rides (0 means 100); FatiguePct is the walking strain multiplier for
	// a rider (0 means 100). Both apply only to a saddled riding horse.
	TravelDurationPct int
	FatiguePct        int
}

// Name is the spec's type for players: "pack-horse" reads "pack horse".
func (s MountSpec) Name() string {
	return strings.ReplaceAll(s.Type, "-", " ")
}

// EffectiveFatiguePct is FatiguePct with 0 read as 100.
func (s MountSpec) EffectiveFatiguePct() int {
	if s.FatiguePct == 0 {
		return 100
	}
	return s.FatiguePct
}

// EffectiveTravelDurationPct is TravelDurationPct with 0 read as 100.
func (s MountSpec) EffectiveTravelDurationPct() int {
	if s.TravelDurationPct == 0 {
		return 100
	}
	return s.TravelDurationPct
}

// CapacityGrams is what a horse of this type carries, saddled or bare.
func (s MountSpec) CapacityGrams(saddled bool) int {
	if saddled {
		return s.SaddledCapacityGrams
	}
	return s.BareCapacityGrams
}

// Validate rejects a malformed spec rather than letting it be guessed at,
// matching weather.Condition.Validate.
func (s MountSpec) Validate() error {
	if s.Type == "" || strings.ContainsAny(s.Type, " \t") {
		return ErrInvalidSpec
	}
	if s.Kind != KindPack && s.Kind != KindRiding {
		return ErrInvalidSpec
	}
	if s.Price < 0 || s.BareCapacityGrams < 0 || s.SaddledCapacityGrams < 0 {
		return ErrInvalidSpec
	}
	if s.TravelDurationPct != 0 && (s.TravelDurationPct < PctMin || s.TravelDurationPct > PctMax) {
		return ErrInvalidSpec
	}
	if s.FatiguePct != 0 && (s.FatiguePct < PctMin || s.FatiguePct > PctMax) {
		return ErrInvalidSpec
	}
	return nil
}

// Horse is one horse in a herd. SaddleItemId is the item id of the saddle
// fitted to it, or 0 when it is bare.
type Horse struct {
	ID           int    `yaml:"id"`
	Type         string `yaml:"type"`
	SaddleItemId int    `yaml:"saddle,omitempty"`
}

// Saddled reports whether a saddle is fitted.
func (h Horse) Saddled() bool { return h.SaddleItemId > 0 }

// Herd is a leader's durable set of horses.
type Herd struct {
	LeaderUserID int     `yaml:"leader_user_id"`
	Horses       []Horse `yaml:"horses,omitempty"`
	NextID       int     `yaml:"next_id,omitempty"`
}

// Validate rejects a herd with no leader, an unnamed or unnumbered horse,
// or a repeated id.
func (h Herd) Validate() error {
	if h.LeaderUserID <= 0 {
		return ErrInvalidHerd
	}
	seen := map[int]bool{}
	for _, horse := range h.Horses {
		if horse.ID <= 0 || horse.Type == "" || horse.SaddleItemId < 0 || seen[horse.ID] {
			return ErrInvalidHerd
		}
		seen[horse.ID] = true
	}
	return nil
}

// Clone returns a deep copy.
func (h Herd) Clone() Herd {
	h.Horses = append([]Horse(nil), h.Horses...)
	return h
}

// nextID is the next free id: NextID, or one past the highest in use.
func (h Herd) nextID() int {
	next := max(h.NextID, 1)
	for _, horse := range h.Horses {
		next = max(next, horse.ID+1)
	}
	return next
}

// Add returns a copy with a new bare horse of the given type, and the horse.
func (h Herd) Add(horseType string) (Herd, Horse) {
	out := h.Clone()
	horse := Horse{ID: out.nextID(), Type: horseType}
	out.Horses = append(out.Horses, horse)
	out.NextID = horse.ID + 1
	return out, horse
}

// Remove returns a copy without the horse, and the removed horse.
func (h Herd) Remove(id int) (Herd, Horse, error) {
	out := h.Clone()
	for i, horse := range out.Horses {
		if horse.ID == id {
			out.Horses = append(out.Horses[:i], out.Horses[i+1:]...)
			return out, horse, nil
		}
	}
	return h, Horse{}, ErrUnknownHorse
}

// SetSaddle returns a copy with the horse's saddle replaced, and the saddle
// item id it had before (0 when bare).
func (h Herd) SetSaddle(id, saddleItemId int) (Herd, int, error) {
	out := h.Clone()
	for i, horse := range out.Horses {
		if horse.ID == id {
			old := horse.SaddleItemId
			out.Horses[i].SaddleItemId = saddleItemId
			return out, old, nil
		}
	}
	return h, 0, ErrUnknownHorse
}

// Sorted returns the horses by ascending id.
func (h Herd) Sorted() []Horse {
	out := append([]Horse(nil), h.Horses...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Find resolves a player's selector: "#2" or "2" by id, else the first
// horse (by id) whose type or kind contains the text ("pack", "riding",
// "horse").
func (h Herd) Find(selector string, specs map[string]MountSpec) (Horse, bool) {
	s := strings.ToLower(strings.TrimSpace(selector))
	if s == "" {
		return Horse{}, false
	}
	if id, err := strconv.Atoi(strings.TrimPrefix(s, "#")); err == nil {
		for _, horse := range h.Horses {
			if horse.ID == id {
				return horse, true
			}
		}
		return Horse{}, false
	}
	for _, horse := range h.Sorted() {
		name := strings.ToLower(horse.Type)
		if spec, ok := specs[horse.Type]; ok {
			name += " " + spec.Name() + " " + string(spec.Kind)
		}
		if strings.Contains(name, s) {
			return horse, true
		}
	}
	return Horse{}, false
}

// CountKind is how many horses of a kind the herd keeps. A horse whose type
// is no longer configured counts toward no kind.
func (h Herd) CountKind(kind Kind, specs map[string]MountSpec) int {
	n := 0
	for _, horse := range h.Horses {
		if spec, ok := specs[horse.Type]; ok && spec.Kind == kind {
			n++
		}
	}
	return n
}

// CanStable reports whether a company of members may keep another horse of
// this kind: one riding and one pack horse per member (owner, 2026-09-28).
func (h Herd) CanStable(kind Kind, specs map[string]MountSpec, members int) error {
	if h.CountKind(kind, specs) >= max(members, 0) {
		return ErrHerdFull
	}
	return nil
}

// CapacityGrams is what the herd carries: each known horse, saddled or
// bare.
func (h Herd) CapacityGrams(specs map[string]MountSpec) int {
	total := 0
	for _, horse := range h.Horses {
		if spec, ok := specs[horse.Type]; ok {
			total += spec.CapacityGrams(horse.Saddled())
		}
	}
	return total
}

// saddledRiders are the saddled riding horses, by ascending id.
func (h Herd) saddledRiders(specs map[string]MountSpec) []MountSpec {
	out := []MountSpec{}
	for _, horse := range h.Sorted() {
		if spec, ok := specs[horse.Type]; ok && spec.Kind == KindRiding && horse.Saddled() {
			out = append(out, spec)
		}
	}
	return out
}

// Relief is the walking strain multiplier for riders and how many members
// ride: one per saddled riding horse. The multiplier is the first such
// horse's (by id); (100, 0) with none.
func (h Herd) Relief(specs map[string]MountSpec) (fatiguePct, riders int) {
	mounted := h.saddledRiders(specs)
	if len(mounted) == 0 {
		return 100, 0
	}
	return mounted[0].EffectiveFatiguePct(), len(mounted)
}

// TravelDurationPct is a route's duration multiplier: the slowest saddled
// riding horse's, when there is one for every member; otherwise 100 (the
// company walks at walking pace). Pack horses never set the pace.
func (h Herd) TravelDurationPct(specs map[string]MountSpec, members int) int {
	mounted := h.saddledRiders(specs)
	if members <= 0 || len(mounted) < members {
		return 100
	}
	pct := 0
	for _, spec := range mounted {
		pct = max(pct, spec.EffectiveTravelDurationPct())
	}
	return pct
}
