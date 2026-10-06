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

// DefaultPatch is the patch threshold, in percent: the share of their wound
// limit the company's healers patch everyone up to after a battle and on
// `company patch` (Phase 35d), until it is set.
const DefaultPatch = 80

// FocusRules are the values a focus takes, in the order they are listed.
// Assist and defend follow a person, so they are no company focus.
var FocusRules = []Rule{NoFocus, Leader, Casters, Healers, Nearest, Weakest, Strongest, Wounded}

// Tactics is a player's company tactics. A blank field is its default.
type Tactics struct {
	Focus   Rule `yaml:"focus,omitempty"`
	Healing int  `yaml:"healing,omitempty"`
	// Patch is the threshold after-battle patching heals to (Phase 35d).
	Patch int `yaml:"patch,omitempty"`
}

// The company's level ladder (Phase 35d): until the player sets a focus, the
// company aims by a default that rises with its leader's level, as the
// enemy's coordination does. An explicit focus, "none" included, always wins.
const (
	// WeakestFocusLevel is the leader level from which the company's
	// default focus is the weakest foe.
	WeakestFocusLevel = 10
	// CastersFocusLevel is the leader level from which the default focus is
	// casters first: casters while any stand, then the weakest.
	CastersFocusLevel = 25
	// HealersFocusLevel is the leader level from which the default focus
	// is the healers rule whenever the enemy group has a healer (Phase
	// 35e). It takes the place of the level's own default for that fight.
	HealersFocusLevel = 5
)

// HealersDefault reports whether the company at a leader level aims at an
// enemy healer first, as its default (Phase 35e): the player has set no
// focus (an explicit one, "none" included, always wins) and the leader is
// level 5 or more. The battle checks that the enemy actually has a healer.
func HealersDefault(userID, level int) bool {
	_, defaulted := FocusFor(userID, level)
	return defaulted && level >= HealersFocusLevel
}

// DefaultFocusAt is the focus a company aims by at a leader level when the
// player has set none: NoFocus (each member by its own rule) below level 10,
// the weakest foe from 10, casters first from 25.
func DefaultFocusAt(level int) Rule {
	switch {
	case level >= CastersFocusLevel:
		return Casters
	case level >= WeakestFocusLevel:
		return Weakest
	}
	return NoFocus
}

// FocusFor is the focus the player's company aims by at the leader's level:
// the focus they set, else the default for the level. defaulted is true when
// the level's default is in force. NoFocus means each member goes by its own
// rule.
func FocusFor(userID, level int) (focus Rule, defaulted bool) {
	var t Tactics
	if p := currentTactics(); p != nil {
		t = p.StoredTactics(userID)
	}
	if t.Focus == "" {
		return DefaultFocusAt(level), true
	}
	return t.Focus, false
}

// IsZero reports whether nothing is set.
func (t Tactics) IsZero() bool {
	return t.Focus == "" && (t.Healing == 0 || t.Healing == DefaultHealing) &&
		(t.Patch == 0 || t.Patch == DefaultPatch)
}

// Resolve fills blank fields with the defaults.
func (t Tactics) Resolve() Tactics {
	if t.Focus == "" {
		t.Focus = NoFocus
	}
	if t.Healing == 0 {
		t.Healing = DefaultHealing
	}
	if t.Patch == 0 {
		t.Patch = DefaultPatch
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

// ParsePatch reads a patch threshold: 50 to 100 percent, with or without a
// percent sign.
func ParsePatch(s string) (int, bool) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	n, err := strconv.Atoi(s)
	if err != nil || n < 50 || n > 100 {
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
	case Strongest, Wounded, Nearest, Furthest, Leader, Casters, Healers:
	default:
		rule = Weakest
	}
	if id, ok := choose(rule, pool, 0); ok {
		return id, true
	}
	return pool[0].ID, true
}
