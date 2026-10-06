package survival

import (
	"fmt"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/climate"
)

// Phase 55: three ailments, each with an obvious cause, one clear battle
// penalty and one remedy made at camp from gathered herbs. They ride on a
// member's Needs like a meal buff (a count of battles left, never game
// time), so they persist, snapshot and reconcile with the needs they sit
// beside, and they use Phase 50's battle-condition plumbing: the penalty is
// fixed when a battle begins and one battle is counted off then.
//
// An ailment fades on its own after a few battles, so none is a trap, but
// the remedy ends it at once. The causes are the company's own choices
// (walking or resting while Frozen, eating raw meat, leaving a puncture
// untreated), never a zone property.

// Ailment kinds, as stored.
const (
	AilmentChill   = "chill"
	AilmentGutAche = "gutache"
	AilmentFever   = "fever"
)

// Ingredient is a quantity of a gathered herb a remedy uses.
type Ingredient struct {
	ItemID int
	Count  int
	Name   string
}

// AilmentSpec is one ailment's rules.
type AilmentSpec struct {
	Kind    string
	Name    string // "Chill", "Gut-ache", "Fever"
	Battles int    // battles it lasts when left alone
	// DamagePct is the percent off the damage the member deals (positive
	// number); GuardPct the percent more damage it takes.
	DamagePct int
	GuardPct  int
	Cause     string // one line, for help and the catch message
	Remedy    []Ingredient
	// RemedyName is the preparation's name: "thyme tea".
	RemedyName string
}

// Effect is the penalty in a few words.
func (a AilmentSpec) Effect() string {
	var parts []string
	if a.DamagePct > 0 {
		parts = append(parts, fmt.Sprintf("-%d%% damage", a.DamagePct))
	}
	if a.GuardPct > 0 {
		parts = append(parts, fmt.Sprintf("+%d%% damage taken", a.GuardPct))
	}
	return strings.Join(parts, ", ")
}

// Wild thyme (30018), mushrooms (30007) and glacial mint (30009) are the
// herbs `gather herbs` yields.
var ailments = []AilmentSpec{
	{
		Kind: AilmentChill, Name: "Chill", Battles: 4, DamagePct: 10,
		Cause:      "walking or resting while Frozen",
		RemedyName: "thyme tea",
		Remedy:     []Ingredient{{ItemID: 30018, Count: 2, Name: "wild thyme"}},
	},
	{
		Kind: AilmentGutAche, Name: "Gut-ache", Battles: 3, GuardPct: 10,
		Cause:      "eating raw game meat",
		RemedyName: "thyme and mushroom tisane",
		Remedy:     []Ingredient{{ItemID: 30018, Count: 1, Name: "wild thyme"}, {ItemID: 30007, Count: 1, Name: "mushroom"}},
	},
	{
		Kind: AilmentFever, Name: "Fever", Battles: 5, DamagePct: 15,
		Cause:      "a puncture wound left untreated through three battles",
		RemedyName: "cooling fever draught",
		Remedy:     []Ingredient{{ItemID: 30009, Count: 1, Name: "glacial mint"}, {ItemID: 30018, Count: 1, Name: "wild thyme"}},
	},
}

// FeverBattles is how many battles a lasting puncture wound can stay
// untreated before it festers into a Fever.
const FeverBattles = 3

// IsFrozen reports whether a signed exposure counts as Frozen: cold at the
// frostbitten band (exposure -50) or worse.
func IsFrozen(exposure int) bool {
	return exposure < 0 && climate.BandFor(exposure) >= climate.BandModerate
}

// Ailments lists every ailment, in display order.
func Ailments() []AilmentSpec { return append([]AilmentSpec(nil), ailments...) }

// AilmentFor is the ailment of a kind.
func AilmentFor(kind string) (AilmentSpec, bool) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, a := range ailments {
		if a.Kind == kind {
			return a, true
		}
	}
	return AilmentSpec{}, false
}

// FindAilment matches a word the player typed: the kind, the name, or its
// first word ("gut", "gut-ache", "chill").
func FindAilment(word string) (AilmentSpec, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return AilmentSpec{}, false
	}
	for _, a := range ailments {
		if word == a.Kind || word == strings.ToLower(a.Name) || word == "gut" && a.Kind == AilmentGutAche {
			return a, true
		}
	}
	return AilmentSpec{}, false
}

// AilmentKinds lists every kind (for data validation).
func AilmentKinds() []string {
	out := make([]string, 0, len(ailments))
	for _, a := range ailments {
		out = append(out, a.Kind)
	}
	return out
}

// battlesOf reads the battles left of a kind.
func battlesOf(n Needs, kind string) int {
	switch kind {
	case AilmentChill:
		return n.Chill
	case AilmentGutAche:
		return n.GutAche
	case AilmentFever:
		return n.Fever
	}
	return 0
}

func setBattles(n *Needs, kind string, v int) {
	switch kind {
	case AilmentChill:
		n.Chill = v
	case AilmentGutAche:
		n.GutAche = v
	case AilmentFever:
		n.Fever = v
	}
}

// AilmentBattles is the battles an ailment has left on a member (0: none).
func AilmentBattles(n Needs, kind string) int { return max(0, battlesOf(n, kind)) }

// ActiveAilments lists the ailments a member has, in display order.
func ActiveAilments(n Needs) []AilmentSpec {
	var out []AilmentSpec
	for _, a := range ailments {
		if battlesOf(n, a.Kind) > 0 {
			out = append(out, a)
		}
	}
	return out
}

// HasAilment reports whether a member has any ailment.
func HasAilment(n Needs) bool { return n.Chill > 0 || n.GutAche > 0 || n.Fever > 0 }

// normalizeAilments caps each stored count at its ailment's length.
func normalizeAilments(in Needs, out *Needs) {
	for _, a := range ailments {
		if v := battlesOf(in, a.Kind); v > 0 {
			setBattles(out, a.Kind, min(v, a.Battles))
		}
	}
}

// CatchAilment gives a member an ailment for its full length. caught is
// true only when the member did not have it: catching it again just starts
// it over, silently.
func (r *Registry) CatchAilment(leaderUserID int, key MemberKey, kind string) (caught bool, err error) {
	spec, ok := AilmentFor(kind)
	if !ok {
		return false, nil
	}
	needs, found := r.NeedsFor(leaderUserID, key)
	if !found {
		return false, ErrUnknownMember
	}
	caught = battlesOf(needs, spec.Kind) <= 0
	setBattles(&needs, spec.Kind, spec.Battles)
	return caught, r.PutNeeds(leaderUserID, key, needs)
}

// CureAilment ends an ailment. cured is true when the member had it.
func (r *Registry) CureAilment(leaderUserID int, key MemberKey, kind string) (cured bool, err error) {
	spec, ok := AilmentFor(kind)
	if !ok {
		return false, nil
	}
	needs, found := r.NeedsFor(leaderUserID, key)
	if !found {
		return false, ErrUnknownMember
	}
	if battlesOf(needs, spec.Kind) <= 0 {
		return false, nil
	}
	setBattles(&needs, spec.Kind, 0)
	return true, r.PutNeeds(leaderUserID, key, needs)
}

// spendAilmentBattle counts one battle off every ailment in n.
func spendAilmentBattle(n *Needs) {
	for _, a := range ailments {
		if v := battlesOf(*n, a.Kind); v > 0 {
			setBattles(n, a.Kind, v-1)
		}
	}
}

// AilmentService is implemented by modules/survival: it gives and ends
// ailments in one durable write.
type AilmentService interface {
	CatchAilment(leaderUserID int, key MemberKey, kind string) (bool, error)
	CureAilment(leaderUserID int, key MemberKey, kind string) (bool, error)
}

var (
	ailmentServiceMu sync.RWMutex
	ailmentService   AilmentService
)

// SetAilmentService registers the active ailment service. nil clears it.
func SetAilmentService(s AilmentService) {
	ailmentServiceMu.Lock()
	defer ailmentServiceMu.Unlock()
	ailmentService = s
}

func currentAilmentService() AilmentService {
	ailmentServiceMu.RLock()
	defer ailmentServiceMu.RUnlock()
	return ailmentService
}

// CatchAilment gives a member an ailment through the registered module; with
// none loaded it does nothing. caught is false when the member already had it.
func CatchAilment(leaderUserID int, key MemberKey, kind string) (bool, error) {
	s := currentAilmentService()
	if s == nil {
		return false, nil
	}
	return s.CatchAilment(leaderUserID, key, kind)
}

// CureAilment ends a member's ailment through the registered module.
func CureAilment(leaderUserID int, key MemberKey, kind string) (bool, error) {
	s := currentAilmentService()
	if s == nil {
		return false, nil
	}
	return s.CureAilment(leaderUserID, key, kind)
}

// Caught is a member that just caught an ailment.
type Caught struct {
	Key  MemberKey
	Name string
}

// CatchChillIfFrozen gives a Chill to each of keys (every living member when
// none are named) whose exposure is Frozen, and returns the ones that caught
// it just now. Callers use it after a step on foot or a rest, so a company
// that walks or sleeps in the cold pays for it. It reads exposure before it
// touches survival, so no lock is held across the two.
func CatchChillIfFrozen(leaderUserID int, keys ...MemberKey) []Caught {
	if currentAilmentService() == nil {
		return nil
	}
	refs := CurrentRoster(leaderUserID)
	if len(refs) == 0 {
		refs = []MemberRef{{Key: LeaderMemberKey}}
	}
	want := map[MemberKey]bool{}
	for _, k := range keys {
		want[k] = true
	}
	var out []Caught
	for _, ref := range refs {
		if ref.Dead || ref.Away || ref.Needless || (len(want) > 0 && !want[ref.Key]) {
			continue
		}
		exposure, ok := climate.ExposureOf(leaderUserID, string(ref.Key))
		if !ok || !IsFrozen(exposure) {
			continue
		}
		if caught, err := CatchAilment(leaderUserID, ref.Key, AilmentChill); err == nil && caught {
			out = append(out, Caught{Key: ref.Key, Name: ref.Name})
		}
	}
	return out
}

// CaughtLine is the line the leader sees when a member catches an ailment.
func CaughtLine(name, kind string) string {
	spec, ok := AilmentFor(kind)
	if !ok {
		return ""
	}
	return fmt.Sprintf(`<ansi fg="yellow">%s has caught a %s</ansi> (%s for %s; "help ailments").`, name, strings.ToLower(spec.Name), spec.Effect(), BattlesLeft(spec.Battles))
}

// AilmentLabels names a member's ailments with the battles each has left:
// "Chill (3 battles)".
func AilmentLabels(n Needs) []string {
	var out []string
	for _, a := range ActiveAilments(n) {
		out = append(out, fmt.Sprintf("%s (%s)", a.Name, BattlesLeft(battlesOf(n, a.Kind))))
	}
	return out
}
