// Package expedition defines the durable, GoMud-free travel domain: terrain
// travel profiles, the travel-session state machine, real-time UTC progress and
// checkpoint math, and the provider seams native commands use to start and view
// travel.
//
// The package imports internal/survival only for the Exertion value type. It
// imports no rooms, users, commands, plugins, timers, or clocks, and it never
// advances GoMud's global time or round count.
package expedition

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/survival"
)

// CheckpointCount is the fixed number of durable progress checkpoints on a
// route. Survival exertion is accrued at these boundaries, never up front.
const CheckpointCount = 10

// Errors returned by the expedition domain and provider seams.
var (
	ErrInvalidProfileName     = errors.New("expedition: profile name is required")
	ErrInvalidProfileDuration = errors.New("expedition: profile duration must be positive")
	ErrInvalidProfileExertion = errors.New("expedition: profile exertion must be non-negative")
	ErrInvalidSession         = errors.New("expedition: invalid travel session")
	ErrInvalidTransition      = errors.New("expedition: invalid session transition")
)

// TravelProfile is a data-driven terrain route definition. Duration is real
// UTC time; Exertion is the total cost charged proportionally over the route.
type TravelProfile struct {
	Name     string
	Duration time.Duration
	Exertion survival.Exertion
}

// Validate rejects a malformed profile rather than applying a guess.
func (p TravelProfile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidProfileName
	}
	if p.Duration <= 0 {
		return ErrInvalidProfileDuration
	}
	if p.Exertion.Hunger < 0 || p.Exertion.Thirst < 0 || p.Exertion.Fatigue < 0 {
		return ErrInvalidProfileExertion
	}
	return nil
}

// SessionState is the lifecycle state of a travel session.
type SessionState uint8

const (
	Traveling SessionState = iota
	// Interrupted is reserved for Phase 6; Phase 5 never transitions a live
	// session into it.
	Interrupted
	Completed
	Cancelled
)

func (s SessionState) String() string {
	switch s {
	case Traveling:
		return "traveling"
	case Interrupted:
		return "interrupted"
	case Completed:
		return "completed"
	case Cancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Valid reports whether the state is a known lifecycle value.
func (s SessionState) Valid() bool {
	return s <= Cancelled
}

// CanTransitionTo reports whether a state change is legal. Completed and
// Cancelled are terminal.
func (s SessionState) CanTransitionTo(to SessionState) bool {
	switch s {
	case Traveling:
		return to == Interrupted || to == Completed || to == Cancelled
	case Interrupted:
		return to == Traveling || to == Completed || to == Cancelled
	default:
		return false
	}
}

// TravelSession is the durable, leader-owned record of a real-time journey. It
// never stores sockets or live mob instances; progress is always derived from
// StartedAtUTC against the current clock.
type TravelSession struct {
	LeaderUserID           int          `yaml:"leader_user_id"`
	OriginRoomID           int          `yaml:"origin_room_id"`
	DestinationRoomID      int          `yaml:"destination_room_id"`
	ExitName               string       `yaml:"exit_name"`
	ProfileName            string       `yaml:"profile_name"`
	StartedAtUTC           time.Time    `yaml:"started_at_utc"`
	LastExertionCheckpoint uint8        `yaml:"last_exertion_checkpoint"`
	State                  SessionState `yaml:"state"`
}

// Validate reports whether the session carries stable, complete identity.
func (s TravelSession) Validate() error {
	if s.LeaderUserID <= 0 || s.OriginRoomID <= 0 || s.DestinationRoomID <= 0 {
		return ErrInvalidSession
	}
	if s.OriginRoomID == s.DestinationRoomID {
		return ErrInvalidSession
	}
	if strings.TrimSpace(s.ExitName) == "" || strings.TrimSpace(s.ProfileName) == "" {
		return ErrInvalidSession
	}
	if s.StartedAtUTC.IsZero() {
		return ErrInvalidSession
	}
	if s.LastExertionCheckpoint > CheckpointCount {
		return ErrInvalidSession
	}
	if !s.State.Valid() {
		return ErrInvalidSession
	}
	return nil
}

// ProgressAt returns the clamped fraction of the route completed, in 0..1.
func (s TravelSession) ProgressAt(now time.Time, duration time.Duration) float64 {
	if duration <= 0 {
		return 1
	}
	elapsed := now.Sub(s.StartedAtUTC)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > duration {
		elapsed = duration
	}
	return float64(elapsed) / float64(duration)
}

// CheckpointAt returns the durable checkpoint attained at now, in 0..10.
func (s TravelSession) CheckpointAt(now time.Time, duration time.Duration) uint8 {
	if duration <= 0 {
		return CheckpointCount
	}
	elapsed := now.Sub(s.StartedAtUTC)
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= duration {
		return CheckpointCount
	}
	attained := int(int64(elapsed) * CheckpointCount / int64(duration))
	if attained < 0 {
		attained = 0
	}
	if attained > CheckpointCount {
		attained = CheckpointCount
	}
	return uint8(attained)
}

// ExertionDue returns the incremental proportional cost owed between the last
// persisted checkpoint and the checkpoint attained at now. Cumulative rounding
// telescopes so a completed route charges exactly the profile total once.
func (s TravelSession) ExertionDue(now time.Time, profile TravelProfile) survival.Exertion {
	attained := s.CheckpointAt(now, profile.Duration)
	if attained <= s.LastExertionCheckpoint {
		return survival.Exertion{}
	}
	to := cumulativeExertion(profile.Exertion, attained)
	from := cumulativeExertion(profile.Exertion, s.LastExertionCheckpoint)
	return survival.Exertion{
		Hunger:  to.Hunger - from.Hunger,
		Thirst:  to.Thirst - from.Thirst,
		Fatigue: to.Fatigue - from.Fatigue,
	}
}

// cumulativeExertion rounds the total cost up to a checkpoint deterministically.
// At CheckpointCount it equals total exactly.
func cumulativeExertion(total survival.Exertion, checkpoint uint8) survival.Exertion {
	return survival.Exertion{
		Hunger:  scaledCost(total.Hunger, checkpoint),
		Thirst:  scaledCost(total.Thirst, checkpoint),
		Fatigue: scaledCost(total.Fatigue, checkpoint),
	}
}

func scaledCost(total int, checkpoint uint8) int {
	if total <= 0 || checkpoint == 0 {
		return 0
	}
	return (total*int(checkpoint) + CheckpointCount/2) / CheckpointCount
}

// Transition returns the session in a new state, or ErrInvalidTransition.
func (s TravelSession) Transition(to SessionState) (TravelSession, error) {
	if !s.State.CanTransitionTo(to) {
		return s, ErrInvalidTransition
	}
	s.State = to
	return s, nil
}

// StartRequest describes a profiled exit the native movement layer wants to
// turn into a journey.
type StartRequest struct {
	LeaderUserID      int
	OriginRoomID      int
	DestinationRoomID int
	ExitName          string
	ProfileName       string
}

// StartProvider is implemented by modules/expedition. StartTravel returns
// handled=false when the request is not a travel-enabled exit, and handled=true
// with a nil error once a durable session exists. A handled error leaves the
// caller in place.
type StartProvider interface {
	StartTravel(req StartRequest) (handled bool, err error)
}

// ViewProvider is implemented by modules/expedition. RenderTravelView returns
// handled=true when it rendered an active session view.
type ViewProvider interface {
	RenderTravelView(leaderUserID int) (handled bool, err error)
}

var (
	providerMu    sync.RWMutex
	startProvider StartProvider
	viewProvider  ViewProvider
)

// SetStartProvider registers the active start provider. Passing nil clears it.
func SetStartProvider(p StartProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	startProvider = p
}

// SetViewProvider registers the active view provider. Passing nil clears it.
func SetViewProvider(p ViewProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	viewProvider = p
}

// Start consults the registered start provider. Without one, the exit is simply
// not travel-enabled.
func Start(req StartRequest) (bool, error) {
	providerMu.RLock()
	p := startProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.StartTravel(req)
}

// TravelView consults the registered view provider. Without one, native look
// behavior is unchanged.
func TravelView(leaderUserID int) (bool, error) {
	providerMu.RLock()
	p := viewProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.RenderTravelView(leaderUserID)
}
