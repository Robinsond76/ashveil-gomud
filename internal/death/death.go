// Package death is Ashveil's death domain (Phase 25a): the settlement
// registry that says which rooms are city churches and village shamans,
// where a dead player wakes, and the seam that lets the engine's death path
// hand a player to modules/death. It imports no rooms, users, or plugins and
// never touches the world clock.
package death

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Kind is a settlement's kind. Only a city's church can be a player's
// respawn checkpoint; a village's shaman can't.
type Kind string

const (
	City    Kind = "city"
	Village Kind = "village"
)

// Room tags a settlement's service room must carry.
const (
	ChurchTag = "church"
	ShamanTag = "shaman"
)

// Character MiscData keys. Both are saved in the user file with the level
// they belong to.
const (
	// CheckpointKey holds the room ID of the church where the player wakes.
	CheckpointKey = "death-checkpoint"
	// PendingKey holds the operation ID of a death whose level has been
	// taken but whose return to a church hasn't happened yet.
	PendingKey = "death-pending"
	// LastLossKey holds the most recent death's levels, "from>to" (Phase
	// 26a), for the experience command.
	LastLossKey = "death-last-loss"
)

// LastLoss reads the most recent death's levels from a character's
// MiscData. ok is false when none is recorded.
func LastLoss(c interface{ GetMiscData(string) any }) (from, to int, ok bool) {
	raw, _ := c.GetMiscData(LastLossKey).(string)
	before, after, found := strings.Cut(raw, ">")
	if !found {
		return 0, 0, false
	}
	var err1, err2 error
	from, err1 = strconv.Atoi(before)
	to, err2 = strconv.Atoi(after)
	if err1 != nil || err2 != nil || from < 1 || to < 1 {
		return 0, 0, false
	}
	return from, to, true
}

// LastLossValue is the LastLossKey value for a death from one level to
// another.
func LastLossValue(from, to int) string {
	return strconv.Itoa(from) + ">" + strconv.Itoa(to)
}

var (
	ErrInvalidSettlement = errors.New("death: invalid settlement")
	ErrDuplicateZone     = errors.New("death: settlement zone listed twice")
)

// Settlement is one registered settlement: a zone, its kind, and the room
// holding its church (a city) or shaman (a village).
type Settlement struct {
	Zone          string
	Kind          Kind
	ServiceRoomID int
	// ServiceMobID is the mob template of the priest or shaman who performs
	// resurrections there (Phase 25b); 0 means no one does.
	ServiceMobID int
}

// ServiceTag is the room tag the settlement's service room must carry.
func (s Settlement) ServiceTag() string {
	if s.Kind == City {
		return ChurchTag
	}
	return ShamanTag
}

func (s Settlement) validate() error {
	if strings.TrimSpace(s.Zone) == "" || s.ServiceRoomID <= 0 || s.ServiceMobID < 0 || (s.Kind != City && s.Kind != Village) {
		return fmt.Errorf("%w: %+v", ErrInvalidSettlement, s)
	}
	return nil
}

// Registry is the set of settlements, keyed by zone.
type Registry struct {
	byZone map[string]Settlement
	order  []string
}

// NewRegistry builds a registry from config entries. An entry with no
// zone, an unknown kind, or a room ID below 1 is skipped, as is a zone
// listed a second time; each skipped entry is reported.
func NewRegistry(entries []Settlement) (Registry, []error) {
	r := Registry{byZone: map[string]Settlement{}}
	var errs []error
	for _, s := range entries {
		s.Zone = strings.TrimSpace(s.Zone)
		s.Kind = Kind(strings.ToLower(strings.TrimSpace(string(s.Kind))))
		if err := s.validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if _, dup := r.byZone[s.Zone]; dup {
			errs = append(errs, fmt.Errorf("%w: %s", ErrDuplicateZone, s.Zone))
			continue
		}
		r.byZone[s.Zone] = s
		r.order = append(r.order, s.Zone)
	}
	return r, errs
}

// Settlements returns the registered settlements in config order.
func (r Registry) Settlements() []Settlement {
	out := make([]Settlement, 0, len(r.order))
	for _, zone := range r.order {
		out = append(out, r.byZone[zone])
	}
	return out
}

// ChurchFor returns the church room of a city zone. ok is false for a
// village, or a zone that isn't registered.
func (r Registry) ChurchFor(zone string) (int, bool) {
	s, ok := r.byZone[zone]
	if !ok || s.Kind != City {
		return 0, false
	}
	return s.ServiceRoomID, true
}

// IsChurch reports whether a room is a registered city's church.
func (r Registry) IsChurch(roomID int) bool {
	if roomID <= 0 {
		return false
	}
	for _, s := range r.byZone {
		if s.Kind == City && s.ServiceRoomID == roomID {
			return true
		}
	}
	return false
}

// ServiceAt returns the settlement whose service room roomID is: a city's
// church or a village's shaman (Phase 25b).
func (r Registry) ServiceAt(roomID int) (Settlement, bool) {
	if roomID <= 0 {
		return Settlement{}, false
	}
	for _, zone := range r.order {
		if s := r.byZone[zone]; s.ServiceRoomID == roomID {
			return s, true
		}
	}
	return Settlement{}, false
}

// Destination picks where a dead player wakes: the checkpoint while it is
// still a valid church, otherwise the fallback room when it loads. ok is
// false when neither will do; the death then stays pending.
func Destination(checkpoint, fallback int, validChurch, loads func(roomID int) bool) (int, bool) {
	if checkpoint > 0 && validChurch(checkpoint) {
		return checkpoint, true
	}
	if fallback > 0 && loads(fallback) {
		return fallback, true
	}
	return 0, false
}

// Provider is implemented by modules/death. The engine's death path calls it
// on the game loop.
type Provider interface {
	// Pending reports whether a user's death has taken its level but not yet
	// returned them to a church.
	Pending(userID int) bool
	// JustReturned reports whether a living user was returned to a church
	// this round or the last. A death queued twice in one round (the combat
	// loop and AutoHeal both queue one) is then the same death, not a new
	// one.
	JustReturned(userID int) bool
	// Respawn returns a dead user to a church with their company. For a new
	// death it first takes one level; a retry (newDeath false) never does.
	Respawn(userID int, newDeath bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider installs the death provider. Passing nil clears it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// Active returns the installed provider. ok is false when there is none, and
// the engine's own death path applies.
func Active() (Provider, bool) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider, provider != nil
}
