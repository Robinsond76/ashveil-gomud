package strategy

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// Phase 30c: company tactics. A player's company-wide focus (a target
// rule every member of their side aims by, instead of its own) and the
// healing threshold their healers heal below. Durable, beside the
// strategies (modules/strategy), behind the TacticsProvider seam. A
// battle may override the focus for itself (internal/battle).

// NoFocus is the focus that leaves every member to its own rule.
const NoFocus Rule = "none"

// DefaultHealing is the healing threshold, in percent, until it is set.
const DefaultHealing = 50

// FocusRules are the values a focus takes, in the order they are listed.
// Assist and defend follow a person, so they are no company focus.
var FocusRules = []Rule{NoFocus, Leader, Casters, Nearest, Weakest, Strongest, Wounded}

// Tactics is a player's company tactics. A blank field is its default.
type Tactics struct {
	Focus   Rule `yaml:"focus,omitempty"`
	Healing int  `yaml:"healing,omitempty"`
}

// IsZero reports whether nothing is set.
func (t Tactics) IsZero() bool {
	return (t.Focus == "" || t.Focus == NoFocus) && (t.Healing == 0 || t.Healing == DefaultHealing)
}

// Resolve fills blank fields with the defaults.
func (t Tactics) Resolve() Tactics {
	if t.Focus == "" {
		t.Focus = NoFocus
	}
	if t.Healing == 0 {
		t.Healing = DefaultHealing
	}
	return t
}

// FocusRule is the focus, ok false for none.
func (t Tactics) FocusRule() (Rule, bool) {
	if t.Focus == "" || t.Focus == NoFocus {
		return "", false
	}
	return t.Focus, true
}

// ParseFocus reads a focus value or its alias.
func ParseFocus(s string) (Rule, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "none" || s == "off" {
		return NoFocus, true
	}
	r, ok := ParseRule(s)
	if !ok {
		return "", false
	}
	for _, f := range FocusRules {
		if f == r {
			return r, true
		}
	}
	return "", false
}

// ParseHealing reads a healing threshold: 10 to 90, in tens, with or
// without a percent sign.
func ParseHealing(s string) (int, bool) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	n, err := strconv.Atoi(s)
	if err != nil || n < 10 || n > 90 || n%10 != 0 {
		return 0, false
	}
	return n, true
}

// DescribeFocus says what a focus goes for, for the tactics command.
func DescribeFocus(r Rule) string {
	if r == "" || r == NoFocus {
		return "each member goes for the foe its own strategy picks"
	}
	return "everyone goes for " + r.Describe()
}

// TacticsProvider is the durable store of tactics (modules/strategy).
type TacticsProvider interface {
	// StoredTactics is what the player has set (blank fields mean the
	// defaults).
	StoredTactics(userID int) Tactics
	// SetTactics stores the player's tactics, durably.
	SetTactics(userID int, t Tactics) error
}

// ErrNoTacticsStore is returned when no store is registered.
var ErrNoTacticsStore = errors.New("tactics can't be saved right now")

var (
	tacticsMu       sync.RWMutex
	tacticsProvider TacticsProvider
)

// SetTacticsProvider registers the store. The module calls it once, at
// init.
func SetTacticsProvider(p TacticsProvider) {
	tacticsMu.Lock()
	defer tacticsMu.Unlock()
	tacticsProvider = p
}

func currentTactics() TacticsProvider {
	tacticsMu.RLock()
	defer tacticsMu.RUnlock()
	return tacticsProvider
}

// TacticsFor is the player's tactics, resolved against the defaults.
func TacticsFor(userID int) Tactics {
	var t Tactics
	if p := currentTactics(); p != nil {
		t = p.StoredTactics(userID)
	}
	return t.Resolve()
}

// SaveTactics stores the player's tactics.
func SaveTactics(userID int, t Tactics) error {
	p := currentTactics()
	if p == nil {
		return ErrNoTacticsStore
	}
	return p.SetTactics(userID, t)
}

// Enemy personalities (Phase 30c): an enemy re-aiming at a company picks
// by its rule among the members it can reach, with a chance (noise, in
// percent) of taking a random one instead.

// Roll returns a number in [0, n). EnemyPick draws its noise from one.
type Roll func(n int) int

// RandomRoll is the game's roll.
func RandomRoll(n int) int { return rand.Intn(n) }

// EnemyPick chooses the company member an enemy aims at, by rule, among
// the members it can reach (foes marked Reachable): noise percent of the
// time a random one, else the rule's choice, else the nearest. Assist,
// defend, and unknown rules read as weakest. ok is false when none is in
// reach.
func EnemyPick(rule Rule, foes []Foe, noise int, roll Roll) (int, bool) {
	var pool []Foe
	for _, f := range foes {
		if f.Reachable {
			pool = append(pool, f)
		}
	}
	if len(pool) == 0 {
		return 0, false
	}
	byFormation(pool)
	if roll == nil {
		roll = RandomRoll
	}
	if noise > 0 && roll(100) < noise {
		return pool[roll(len(pool))].ID, true
	}
	switch rule {
	case Strongest, Wounded, Nearest, Furthest, Leader, Casters:
	default:
		rule = Weakest
	}
	if id, ok := choose(rule, pool, 0); ok {
		return id, true
	}
	return pool[0].ID, true
}
